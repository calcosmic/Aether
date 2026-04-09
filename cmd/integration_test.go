package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Integration tests: verify end-to-end command paths
// ---------------------------------------------------------------------------

// setupIntegrationStore creates a realistic temp store with colony state,
// pheromones, and hub files so that pr-context and colony-prime can run
// end-to-end without hitting the real filesystem.
func setupIntegrationStore(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	goal := "Integration test: verify core commands work end-to-end"
	now := time.Now().Format(time.RFC3339)
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		ColonyDepth:  "standard",
		Milestone:    "Open Chambers",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID: 1, Name: "Integration Verification", Status: "in_progress",
					Tasks: []colony.Task{
						{ID: strPtr("it1"), Goal: "Verify binary builds and runs", Status: "completed"},
						{ID: strPtr("it2"), Goal: "Verify pr-context end-to-end", Status: "in_progress"},
					},
				},
			},
		},
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{ID: "id1", Phase: 1, Claim: "Use JSON envelope output", Rationale: "Machine-parseable", Timestamp: now},
			},
		},
		Events: []string{
			now + "|init|system|Colony initialized",
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	s0_8 := 0.8
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID: "sig1", Type: "FOCUS", Priority: "normal", Source: "user",
				CreatedAt: now, Active: true, Strength: &s0_8,
				Content: json.RawMessage(`{"text": "Integration testing"}`),
			},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	// Set up hub directory with QUEEN.md
	hubDir := filepath.Join(tmpDir, "hub")
	if err := os.MkdirAll(filepath.Join(hubDir, "hive"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte("# QUEEN.md\n\n## Wisdom\nPrefer simplicity.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

// strPtr is a helper to create a string pointer (avoids duplicating from context_bench_test.go).
func strPtr(s string) *string { return &s }

// TestIntegrationVersion verifies the version command runs and produces output.
func TestIntegrationVersion(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := buf.String()

	// Version output is wrapped in a JSON envelope
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("version output should contain ok:true, got: %s", output)
	}
	if !strings.Contains(output, "0.0.0-dev") {
		t.Errorf("version output should contain version string, got: %s", output)
	}
}

// TestIntegrationPRContext verifies pr-context runs end-to-end with a temp store
// and produces a valid JSON envelope with the expected schema.
func TestIntegrationPRContext(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := setupIntegrationStore(t)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"pr-context"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("pr-context command failed: %v", err)
	}

	output := buf.String()

	// Must be valid JSON
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("pr-context output is not valid JSON: %v\noutput: %s", err, output)
	}

	// Must have ok:true
	if ok, exists := envelope["ok"]; !exists || ok != true {
		t.Errorf("expected ok:true, got: %v", envelope["ok"])
	}

	// Must have a result with schema
	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result object in envelope")
	}
	if schema, exists := result["schema"]; !exists {
		t.Error("expected schema field in result")
	} else if schema != "pr-context-v1" {
		t.Errorf("schema = %v, want pr-context-v1", schema)
	}

	// Must have prompt_section (the core output)
	if _, exists := result["prompt_section"]; !exists {
		t.Error("expected prompt_section in result")
	}
}

// TestIntegrationColonyPrime verifies colony-prime runs end-to-end with a temp store
// and produces a valid JSON envelope.
func TestIntegrationColonyPrime(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := setupIntegrationStore(t)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"colony-prime"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("colony-prime command failed: %v", err)
	}

	output := buf.String()

	// Must be valid JSON
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("colony-prime output is not valid JSON: %v\noutput: %s", err, output)
	}

	// Must have ok:true
	if ok, exists := envelope["ok"]; !exists || ok != true {
		t.Errorf("expected ok:true, got: %v", envelope["ok"])
	}

	// Must have a result object
	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result object in envelope")
	}

	// Must have context (the assembled prompt output)
	if _, exists := result["context"]; !exists {
		t.Error("expected 'context' field in result")
	}

	// Must have budget and sections metadata
	if _, exists := result["budget"]; !exists {
		t.Error("expected 'budget' field in result")
	}
	if _, exists := result["sections"]; !exists {
		t.Error("expected 'sections' field in result")
	}
}

// TestIntegrationCacheClean verifies cache-clean runs end-to-end with a temp store.
func TestIntegrationCacheClean(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := setupIntegrationStore(t)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	// Create some cache files to clean up
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	os.WriteFile(filepath.Join(dataDir, ".cache_COLONY_STATE.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dataDir, ".cache_pheromones.json"), []byte("{}"), 0644)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"cache-clean"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("cache-clean command failed: %v", err)
	}

	output := buf.String()

	// Must be valid JSON with ok:true
	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			FilesRemoved int `json:"files_removed"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("cache-clean output is not valid JSON: %v\noutput: %s", err, output)
	}
	if !envelope.OK {
		t.Error("expected ok:true in cache-clean output")
	}
	if envelope.Result.FilesRemoved != 2 {
		t.Errorf("files_removed = %d, want 2", envelope.Result.FilesRemoved)
	}

	// Verify cache files were actually removed
	if _, err := os.Stat(filepath.Join(dataDir, ".cache_COLONY_STATE.json")); err == nil {
		t.Error(".cache_COLONY_STATE.json should be removed after cache-clean")
	}
	if _, err := os.Stat(filepath.Join(dataDir, ".cache_pheromones.json")); err == nil {
		t.Error(".cache_pheromones.json should be removed after cache-clean")
	}
}
