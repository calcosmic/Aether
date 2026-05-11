package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// setupPheromoneTest creates a temp dir with a fresh store and resets globals.
func setupPheromoneTest(t *testing.T) (tmpDir string, buf *bytes.Buffer) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	tmpDir = t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	buf = &bytes.Buffer{}
	stdout = buf

	os.Setenv("AETHER_ROOT", tmpDir)
	t.Cleanup(func() { os.Unsetenv("AETHER_ROOT") })

	return tmpDir, buf
}

// loadPheromoneFile reads pheromones.json from the store for assertions.
func loadPheromoneFile(t *testing.T) colony.PheromoneFile {
	t.Helper()
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("failed to load pheromones.json: %v", err)
	}
	return pf
}

// parseOutput parses the JSON envelope from stdout.
func parseOutput(t *testing.T, output string) map[string]interface{} {
	t.Helper()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", output)
	}
	return envelope
}

// derefInt safely dereferences an *int, returning -1 if nil.
func derefInt(p *int) int {
	if p == nil {
		return -1
	}
	return *p
}

// derefFloat safely dereferences a *float64, returning -1 if nil.
func derefFloat(p *float64) float64 {
	if p == nil {
		return -1
	}
	return *p
}

func TestPheromoneDedup_ReinforcesInsteadOfDuplicating(t *testing.T) {
	_, buf := setupPheromoneTest(t)

	content := "focus on error handling"

	// First write: should create a new signal
	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", content})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	firstOutput := parseOutput(t, buf.String())
	firstResult := firstOutput["result"].(map[string]interface{})
	if firstResult["replaced"] != false {
		t.Errorf("first write: expected replaced=false, got %v", firstResult["replaced"])
	}
	if firstResult["total"] != float64(1) {
		t.Errorf("first write: expected total=1, got %v", firstResult["total"])
	}

	buf.Reset()

	// Second write with same type + content: should reinforce
	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", content})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	secondOutput := parseOutput(t, buf.String())
	secondResult := secondOutput["result"].(map[string]interface{})
	if secondResult["replaced"] != true {
		t.Errorf("second write: expected replaced=true, got %v", secondResult["replaced"])
	}
	if secondResult["total"] != float64(1) {
		t.Errorf("second write: expected total=1, got %v", secondResult["total"])
	}

	// Verify pheromones.json has exactly 1 signal
	pf := loadPheromoneFile(t)
	if len(pf.Signals) != 1 {
		t.Fatalf("expected 1 signal in pheromones.json, got %d", len(pf.Signals))
	}

	sig := pf.Signals[0]
	if sig.ReinforcementCount == nil || *sig.ReinforcementCount != 1 {
		t.Errorf("reinforcement_count = %d, want 1", derefInt(sig.ReinforcementCount))
	}
	if sig.Strength == nil || *sig.Strength != 1.0 {
		t.Errorf("strength = %v, want 1.0", sig.Strength)
	}
}

func TestPheromoneDedup_MultipleReinforcements(t *testing.T) {
	_, buf := setupPheromoneTest(t)

	content := "focus on error handling"

	// Write 3 times with same content
	for i := 0; i < 3; i++ {
		buf.Reset()
		rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", content})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("write %d failed: %v", i+1, err)
		}
	}

	// Verify only 1 signal exists
	pf := loadPheromoneFile(t)
	if len(pf.Signals) != 1 {
		t.Fatalf("expected 1 signal after 3 writes, got %d", len(pf.Signals))
	}

	sig := pf.Signals[0]
	if sig.ReinforcementCount == nil || *sig.ReinforcementCount != 2 {
		t.Errorf("reinforcement_count = %d, want 2 (first write creates, next 2 reinforce)", derefInt(sig.ReinforcementCount))
	}
}

func TestPheromoneDedup_DifferentContentCreatesSeparateSignals(t *testing.T) {
	_, buf := setupPheromoneTest(t)

	// Write two different FOCUS signals
	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", "focus on error handling"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	buf.Reset()
	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", "focus on performance"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	output := parseOutput(t, buf.String())
	result := output["result"].(map[string]interface{})
	if result["replaced"] != false {
		t.Errorf("different content should NOT be deduped, got replaced=%v", result["replaced"])
	}
	if result["total"] != float64(2) {
		t.Errorf("expected total=2 for different content, got %v", result["total"])
	}

	pf := loadPheromoneFile(t)
	if len(pf.Signals) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(pf.Signals))
	}
}

func TestPheromoneDedup_DifferentTypeCreatesSeparateSignals(t *testing.T) {
	_, buf := setupPheromoneTest(t)

	content := "avoid global variables"

	// Write as FOCUS
	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", content})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("FOCUS write failed: %v", err)
	}

	buf.Reset()

	// Write same content as REDIRECT
	rootCmd.SetArgs([]string{"pheromone-write", "--type", "REDIRECT", "--content", content})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("REDIRECT write failed: %v", err)
	}

	output := parseOutput(t, buf.String())
	result := output["result"].(map[string]interface{})
	if result["replaced"] != false {
		t.Errorf("same content different type should NOT dedup, got replaced=%v", result["replaced"])
	}
	if result["total"] != float64(2) {
		t.Errorf("expected total=2 for different types, got %v", result["total"])
	}

	pf := loadPheromoneFile(t)
	if len(pf.Signals) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(pf.Signals))
	}

	// Verify one is FOCUS and one is REDIRECT
	types := map[string]bool{}
	for _, sig := range pf.Signals {
		types[sig.Type] = true
	}
	if !types["FOCUS"] || !types["REDIRECT"] {
		t.Errorf("expected both FOCUS and REDIRECT, got types: %v", types)
	}
}

func TestPheromoneDedup_ContentHashIsSHA256(t *testing.T) {
	_, _ = setupPheromoneTest(t)

	content := "test content for hashing"

	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", content})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Compute expected SHA-256 hash
	h := sha256.Sum256([]byte(content))
	expectedHash := "sha256:" + hex.EncodeToString(h[:])

	pf := loadPheromoneFile(t)
	if len(pf.Signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(pf.Signals))
	}

	sig := pf.Signals[0]
	if sig.ContentHash == nil {
		t.Fatal("content_hash is nil")
	}
	if *sig.ContentHash != expectedHash {
		t.Errorf("content_hash = %q, want %q", *sig.ContentHash, expectedHash)
	}
}

func TestPheromoneDedup_DoesNotMatchInactiveSignals(t *testing.T) {
	_, buf := setupPheromoneTest(t)
	_ = buf // used indirectly via stdout global

	content := "focus on error handling"

	// Write first signal
	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", content})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	pf := loadPheromoneFile(t)
	sigID := pf.Signals[0].ID

	// Expire the signal (set Active = false)
	buf.Reset()
	rootCmd.SetArgs([]string{"pheromone-expire", "--id", sigID})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expire failed: %v", err)
	}

	// Write the same content again: should create a NEW signal since old one is inactive
	buf.Reset()
	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", content})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	output := parseOutput(t, buf.String())
	result := output["result"].(map[string]interface{})
	if result["replaced"] != false {
		t.Errorf("should create new signal when existing is inactive, got replaced=%v", result["replaced"])
	}
	if result["total"] != float64(2) {
		t.Errorf("expected total=2 (1 inactive + 1 new), got %v", result["total"])
	}

	pf = loadPheromoneFile(t)
	if len(pf.Signals) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(pf.Signals))
	}
}

func TestPheromoneDedup_StrengthReinforcedToMax(t *testing.T) {
	_, buf := setupPheromoneTest(t)

	content := "focus on error handling"

	// Write with a low strength
	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", content, "--strength", "0.3"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	// Reinforce
	buf.Reset()
	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", content, "--strength", "0.5"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	pf := loadPheromoneFile(t)
	if len(pf.Signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(pf.Signals))
	}

	sig := pf.Signals[0]
	if sig.Strength == nil || *sig.Strength != 1.0 {
		t.Errorf("strength after reinforcement = %v, want 1.0 (maxed)", derefFloat(sig.Strength))
	}
}

func TestPheromoneDedup_SameContentAllThreeTypes(t *testing.T) {
	_, buf := setupPheromoneTest(t)

	content := "test dedup across types"

	for _, sigType := range []string{"FOCUS", "REDIRECT", "FEEDBACK"} {
		buf.Reset()
		rootCmd.SetArgs([]string{"pheromone-write", "--type", sigType, "--content", content})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("%s write failed: %v", sigType, err)
		}
		output := parseOutput(t, buf.String())
		result := output["result"].(map[string]interface{})
		if result["replaced"] != false {
			t.Errorf("%s: first write should not be replaced, got replaced=%v", sigType, result["replaced"])
		}
	}

	// Reinforce each type
	for _, sigType := range []string{"FOCUS", "REDIRECT", "FEEDBACK"} {
		buf.Reset()
		rootCmd.SetArgs([]string{"pheromone-write", "--type", sigType, "--content", content})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("%s reinforce failed: %v", sigType, err)
		}
		output := parseOutput(t, buf.String())
		result := output["result"].(map[string]interface{})
		if result["replaced"] != true {
			t.Errorf("%s: second write should be replaced/reinforced, got replaced=%v", sigType, result["replaced"])
		}
	}

	// Should have exactly 3 signals
	pf := loadPheromoneFile(t)
	if len(pf.Signals) != 3 {
		t.Fatalf("expected 3 signals (one per type), got %d", len(pf.Signals))
	}

	// Each should have reinforcement_count = 1
	for _, sig := range pf.Signals {
		if sig.ReinforcementCount == nil || *sig.ReinforcementCount != 1 {
			t.Errorf("type %s: reinforcement_count = %d, want 1", sig.Type, derefInt(sig.ReinforcementCount))
		}
	}
}
