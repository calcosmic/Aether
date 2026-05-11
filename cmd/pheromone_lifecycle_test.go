package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ---------------------------------------------------------------------------
// Integration tests: full pheromone lifecycle
// write -> read -> prime -> display -> expire -> verify gone
// ---------------------------------------------------------------------------

// setupLifecycleStore creates a fresh temp store with empty pheromones.json
// for lifecycle tests. Each call gets its own temp dir to prevent state leaks.
func setupLifecycleStore(t *testing.T) (*storage.Store, string) {
	t.Helper()
	forceJSONOutputModeForTest(t)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Start with an empty pheromones file
	pf := colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}
	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}
	return s, tmpDir
}

// parseLifecycleEnvelope parses the JSON envelope output from any command.
// Some commands (e.g., pheromone-display) write a text table to stdout before
// the JSON envelope, so we extract the JSON object starting with "{" from the
// output.
func parseLifecycleEnvelope(t *testing.T, output string) map[string]interface{} {
	t.Helper()
	trimmed := strings.TrimSpace(output)
	// Find the start of the JSON envelope (first '{')
	idx := strings.Index(trimmed, "{")
	if idx < 0 {
		t.Fatalf("no JSON object found in output: %s", trimmed)
	}
	jsonStr := trimmed[idx:]
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &envelope); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, jsonStr)
	}
	return envelope
}

// runCommand executes a rootCmd subcommand with the given args, returns output.
func runCommand(t *testing.T, args []string, outBuf, errBuf *bytes.Buffer) string {
	t.Helper()
	resetRootCmd(t)
	rootCmd.SetArgs(args)
	rootCmd.SetOut(outBuf)
	if err := rootCmd.Execute(); err != nil {
		// Some commands return nil error and write ok:false to stderr
	}
	return outBuf.String()
}

// --- Test 1: Full lifecycle FOCUS signal ---

func TestPheromoneLifecycle_FocusSignal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := setupLifecycleStore(t)
	store = s

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	var buf bytes.Buffer
	stdout = &buf

	// Step 1: Write a FOCUS signal
	rootCmd.SetArgs([]string{
		"pheromone-write",
		"--type", "FOCUS",
		"--content", "focus on error handling",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-write failed: %v", err)
	}
	writeOutput := parseLifecycleEnvelope(t, buf.String())
	if writeOutput["ok"] != true {
		t.Fatalf("write failed: %s", buf.String())
	}
	writeResult := writeOutput["result"].(map[string]interface{})
	signal := writeResult["signal"].(map[string]interface{})
	sigID := signal["id"].(string)
	if signal["type"] != "FOCUS" {
		t.Errorf("signal type = %v, want FOCUS", signal["type"])
	}
	if signal["priority"] != "normal" {
		t.Errorf("FOCUS default priority = %v, want normal", signal["priority"])
	}

	// Step 2: Read it back
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-read failed: %v", err)
	}
	readOutput := parseLifecycleEnvelope(t, buf.String())
	readResult := readOutput["result"].(map[string]interface{})
	signals := readResult["signals"].([]interface{})
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal after write, got %d", len(signals))
	}
	readSig := signals[0].(map[string]interface{})
	if readSig["id"] != sigID {
		t.Errorf("read signal id = %v, want %v", readSig["id"], sigID)
	}

	// Step 3: Prime it (format for prompt injection)
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-prime"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-prime failed: %v", err)
	}
	primeOutput := parseLifecycleEnvelope(t, buf.String())
	primeResult := primeOutput["result"].(map[string]interface{})
	if primeResult["signal_count"] != float64(1) {
		t.Errorf("prime signal_count = %v, want 1", primeResult["signal_count"])
	}
	if primeResult["focus_count"] != float64(1) {
		t.Errorf("prime focus_count = %v, want 1", primeResult["focus_count"])
	}
	section := primeResult["section"].(string)
	if !strings.Contains(section, "FOCUS") || !strings.Contains(section, "error handling") {
		t.Errorf("prime section should contain FOCUS and 'error handling', got: %s", section)
	}

	// Step 4: Display it
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-display"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-display failed: %v", err)
	}
	displayOutput := parseLifecycleEnvelope(t, buf.String())
	displayResult := displayOutput["result"].(map[string]interface{})
	if displayResult["count"] != float64(1) {
		t.Errorf("display count = %v, want 1", displayResult["count"])
	}
	display := displayResult["display"].(string)
	for _, want := range []string{"LIFE", "phase-scoped", "30d decay"} {
		if !strings.Contains(display, want) {
			t.Errorf("display missing %q, got: %s", want, display)
		}
	}
	displaySignals := displayResult["signals"].([]interface{})
	if len(displaySignals) != 1 {
		t.Fatalf("display signals count = %d, want 1", len(displaySignals))
	}

	// Step 5: Expire it
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-expire", "--id", sigID})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-expire failed: %v", err)
	}
	expireOutput := parseLifecycleEnvelope(t, buf.String())
	if expireOutput["ok"] != true {
		t.Fatalf("expire failed: %s", buf.String())
	}
	expireResult := expireOutput["result"].(map[string]interface{})
	if expireResult["expired"] != true {
		t.Errorf("expired = %v, want true", expireResult["expired"])
	}

	// Step 6: Verify it's gone from reads
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-read after expire failed: %v", err)
	}
	verifyOutput := parseLifecycleEnvelope(t, buf.String())
	verifyResult := verifyOutput["result"].(map[string]interface{})
	verifySignals := verifyResult["signals"].([]interface{})
	if len(verifySignals) != 0 {
		t.Errorf("expected 0 signals after expire, got %d", len(verifySignals))
	}
}

// --- Test 2: Full lifecycle REDIRECT signal ---

func TestPheromoneLifecycle_RedirectSignal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := setupLifecycleStore(t)
	store = s

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	var buf bytes.Buffer
	stdout = &buf

	// Write REDIRECT signal
	rootCmd.SetArgs([]string{
		"pheromone-write",
		"--type", "REDIRECT",
		"--content", "avoid global variables",
		"--strength", "0.9",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-write failed: %v", err)
	}
	writeOutput := parseLifecycleEnvelope(t, buf.String())
	writeResult := writeOutput["result"].(map[string]interface{})
	signal := writeResult["signal"].(map[string]interface{})
	sigID := signal["id"].(string)

	if signal["type"] != "REDIRECT" {
		t.Errorf("signal type = %v, want REDIRECT", signal["type"])
	}
	if signal["priority"] != "high" {
		t.Errorf("REDIRECT default priority = %v, want high", signal["priority"])
	}
	if signal["strength"] != float64(0.9) {
		t.Errorf("strength = %v, want 0.9", signal["strength"])
	}
	// REDIRECT signals should have an ExpiresAt (30 days)
	if signal["expires_at"] == nil {
		t.Error("REDIRECT signal should have expires_at set")
	}

	// Read and verify
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	rootCmd.Execute()
	readOutput := parseLifecycleEnvelope(t, buf.String())
	readSignals := readOutput["result"].(map[string]interface{})["signals"].([]interface{})
	if len(readSignals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(readSignals))
	}

	// Prime and verify redirect section
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-prime"})
	rootCmd.Execute()
	primeOutput := parseLifecycleEnvelope(t, buf.String())
	primeResult := primeOutput["result"].(map[string]interface{})
	if primeResult["redirect_count"] != float64(1) {
		t.Errorf("redirect_count = %v, want 1", primeResult["redirect_count"])
	}
	section := primeResult["section"].(string)
	if !strings.Contains(section, "REDIRECT") {
		t.Errorf("prime section should contain REDIRECT, got: %s", section)
	}
	if !strings.Contains(section, "Hard Constraints") {
		t.Errorf("prime section should contain 'Hard Constraints', got: %s", section)
	}

	// Display and verify
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-display"})
	rootCmd.Execute()
	displayOutput := parseLifecycleEnvelope(t, buf.String())
	displayResult := displayOutput["result"].(map[string]interface{})
	displaySignals := displayResult["signals"].([]interface{})
	if len(displaySignals) != 1 {
		t.Fatalf("display should show 1 signal, got %d", len(displaySignals))
	}
	display := displayResult["display"].(string)
	for _, want := range []string{"LIFE", "ttl", "60d decay"} {
		if !strings.Contains(display, want) {
			t.Errorf("display missing %q, got: %s", want, display)
		}
	}
	ds := displaySignals[0].(map[string]interface{})
	if ds["type"] != "REDIRECT" {
		t.Errorf("display type = %v, want REDIRECT", ds["type"])
	}

	// Expire and verify gone
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-expire", "--id", sigID})
	rootCmd.Execute()
	parseLifecycleEnvelope(t, buf.String())

	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	rootCmd.Execute()
	goneOutput := parseLifecycleEnvelope(t, buf.String())
	goneSignals := goneOutput["result"].(map[string]interface{})["signals"].([]interface{})
	if len(goneSignals) != 0 {
		t.Errorf("expected 0 signals after expire, got %d", len(goneSignals))
	}
}

// --- Test 3: Full lifecycle FEEDBACK signal ---

func TestPheromoneLifecycle_FeedbackSignal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := setupLifecycleStore(t)
	store = s

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	var buf bytes.Buffer
	stdout = &buf

	// Write FEEDBACK signal
	rootCmd.SetArgs([]string{
		"pheromone-write",
		"--type", "FEEDBACK",
		"--content", "prefer composition over inheritance",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-write failed: %v", err)
	}
	writeOutput := parseLifecycleEnvelope(t, buf.String())
	writeResult := writeOutput["result"].(map[string]interface{})
	signal := writeResult["signal"].(map[string]interface{})
	sigID := signal["id"].(string)

	if signal["type"] != "FEEDBACK" {
		t.Errorf("signal type = %v, want FEEDBACK", signal["type"])
	}
	if signal["priority"] != "low" {
		t.Errorf("FEEDBACK default priority = %v, want low", signal["priority"])
	}
	// FEEDBACK signals should have an ExpiresAt (7 days)
	if signal["expires_at"] == nil {
		t.Error("FEEDBACK signal should have expires_at set")
	}

	// Read, prime, display
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	rootCmd.Execute()
	readOutput := parseLifecycleEnvelope(t, buf.String())
	readSignals := readOutput["result"].(map[string]interface{})["signals"].([]interface{})
	if len(readSignals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(readSignals))
	}

	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-prime"})
	rootCmd.Execute()
	primeOutput := parseLifecycleEnvelope(t, buf.String())
	primeResult := primeOutput["result"].(map[string]interface{})
	if primeResult["feedback_count"] != float64(1) {
		t.Errorf("feedback_count = %v, want 1", primeResult["feedback_count"])
	}

	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-display"})
	rootCmd.Execute()
	displayOutput := parseLifecycleEnvelope(t, buf.String())
	displayResult := displayOutput["result"].(map[string]interface{})
	if displayResult["count"] != float64(1) {
		t.Errorf("display count = %v, want 1", displayResult["count"])
	}

	// Expire and verify gone
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-expire", "--id", sigID})
	rootCmd.Execute()
	parseLifecycleEnvelope(t, buf.String())

	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	rootCmd.Execute()
	goneOutput := parseLifecycleEnvelope(t, buf.String())
	goneSignals := goneOutput["result"].(map[string]interface{})["signals"].([]interface{})
	if len(goneSignals) != 0 {
		t.Errorf("expected 0 signals after expire, got %d", len(goneSignals))
	}
}

// --- Test 4: Multiple signals, mixed types, lifecycle ---

func TestPheromoneLifecycle_MultipleMixedSignals(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := setupLifecycleStore(t)
	store = s

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	var buf bytes.Buffer
	stdout = &buf

	// Write 3 signals of different types
	types := []struct {
		sigType  string
		content  string
		strength float64
		priority string
	}{
		{"FOCUS", "focus on tests", 0.8, "normal"},
		{"REDIRECT", "avoid magic numbers", 1.0, "high"},
		{"FEEDBACK", "good progress", 0.5, "low"},
	}

	var sigIDs []string
	for _, tc := range types {
		buf.Reset()
		resetRootCmd(t)
		args := []string{
			"pheromone-write",
			"--type", tc.sigType,
			"--content", tc.content,
			"--strength", fmt.Sprintf("%.1f", tc.strength),
		}
		if tc.priority != "" {
			args = append(args, "--priority", tc.priority)
		}
		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("pheromone-write %s failed: %v", tc.sigType, err)
		}
		output := parseLifecycleEnvelope(t, buf.String())
		sig := output["result"].(map[string]interface{})["signal"].(map[string]interface{})
		sigIDs = append(sigIDs, sig["id"].(string))
	}

	// Read should show all 3
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	rootCmd.Execute()
	readOutput := parseLifecycleEnvelope(t, buf.String())
	readSignals := readOutput["result"].(map[string]interface{})["signals"].([]interface{})
	if len(readSignals) != 3 {
		t.Fatalf("expected 3 signals, got %d", len(readSignals))
	}

	// Count should show 1 of each type
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-count"})
	rootCmd.Execute()
	countOutput := parseLifecycleEnvelope(t, buf.String())
	countResult := countOutput["result"].(map[string]interface{})
	if countResult["focus"] != float64(1) {
		t.Errorf("focus count = %v, want 1", countResult["focus"])
	}
	if countResult["redirect"] != float64(1) {
		t.Errorf("redirect count = %v, want 1", countResult["redirect"])
	}
	if countResult["feedback"] != float64(1) {
		t.Errorf("feedback count = %v, want 1", countResult["feedback"])
	}
	if countResult["total"] != float64(3) {
		t.Errorf("total count = %v, want 3", countResult["total"])
	}

	// Prime should have all 3
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-prime"})
	rootCmd.Execute()
	primeOutput := parseLifecycleEnvelope(t, buf.String())
	primeResult := primeOutput["result"].(map[string]interface{})
	if primeResult["signal_count"] != float64(3) {
		t.Errorf("prime signal_count = %v, want 3", primeResult["signal_count"])
	}

	// Display should show all 3
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-display"})
	rootCmd.Execute()
	displayOutput := parseLifecycleEnvelope(t, buf.String())
	displayResult := displayOutput["result"].(map[string]interface{})
	if displayResult["count"] != float64(3) {
		t.Errorf("display count = %v, want 3", displayResult["count"])
	}

	// Expire REDIRECT signal only
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-expire", "--id", sigIDs[1]})
	rootCmd.Execute()
	parseLifecycleEnvelope(t, buf.String())

	// Read should now show 2
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	rootCmd.Execute()
	readOutput2 := parseLifecycleEnvelope(t, buf.String())
	readSignals2 := readOutput2["result"].(map[string]interface{})["signals"].([]interface{})
	if len(readSignals2) != 2 {
		t.Fatalf("expected 2 signals after expiring one, got %d", len(readSignals2))
	}

	// Count should show 0 redirect
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-count"})
	rootCmd.Execute()
	countOutput2 := parseLifecycleEnvelope(t, buf.String())
	countResult2 := countOutput2["result"].(map[string]interface{})
	if countResult2["redirect"] != float64(0) {
		t.Errorf("redirect count after expire = %v, want 0", countResult2["redirect"])
	}
	if countResult2["total"] != float64(2) {
		t.Errorf("total count after expire = %v, want 2", countResult2["total"])
	}

	// Expire remaining 2
	for _, id := range []string{sigIDs[0], sigIDs[2]} {
		buf.Reset()
		resetRootCmd(t)
		rootCmd.SetArgs([]string{"pheromone-expire", "--id", id})
		rootCmd.Execute()
		parseLifecycleEnvelope(t, buf.String())
	}

	// Read should show 0
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	rootCmd.Execute()
	finalOutput := parseLifecycleEnvelope(t, buf.String())
	finalSignals := finalOutput["result"].(map[string]interface{})["signals"].([]interface{})
	if len(finalSignals) != 0 {
		t.Errorf("expected 0 signals after expiring all, got %d", len(finalSignals))
	}
}

// --- Test 5: Deduplication during lifecycle ---

func TestPheromoneLifecycle_Deduplication(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := setupLifecycleStore(t)
	store = s

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	var buf bytes.Buffer
	stdout = &buf

	// Write same signal twice
	for i := 0; i < 2; i++ {
		buf.Reset()
		resetRootCmd(t)
		rootCmd.SetArgs([]string{
			"pheromone-write",
			"--type", "FOCUS",
			"--content", "focus on tests",
		})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("write %d failed: %v", i+1, err)
		}
	}

	// Second write should report replaced=true
	secondOutput := parseLifecycleEnvelope(t, buf.String())
	secondResult := secondOutput["result"].(map[string]interface{})
	if secondResult["replaced"] != true {
		t.Errorf("second write replaced = %v, want true", secondResult["replaced"])
	}
	if secondResult["total"] != float64(1) {
		t.Errorf("total after dedup = %v, want 1", secondResult["total"])
	}

	// Read should still show 1
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	rootCmd.Execute()
	readOutput := parseLifecycleEnvelope(t, buf.String())
	readSignals := readOutput["result"].(map[string]interface{})["signals"].([]interface{})
	if len(readSignals) != 1 {
		t.Fatalf("expected 1 signal after dedup writes, got %d", len(readSignals))
	}

	// Verify reinforcement count was incremented
	sig := readSignals[0].(map[string]interface{})
	rc, ok := sig["reinforcement_count"]
	if !ok {
		t.Error("signal should have reinforcement_count set")
	} else if rc != float64(1) {
		t.Errorf("reinforcement_count = %v, want 1", rc)
	}

	// Prime should show 1 signal
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-prime"})
	rootCmd.Execute()
	primeOutput := parseLifecycleEnvelope(t, buf.String())
	primeResult := primeOutput["result"].(map[string]interface{})
	if primeResult["signal_count"] != float64(1) {
		t.Errorf("prime signal_count = %v, want 1", primeResult["signal_count"])
	}

	// Display should show 1
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-display"})
	rootCmd.Execute()
	displayOutput := parseLifecycleEnvelope(t, buf.String())
	displayResult := displayOutput["result"].(map[string]interface{})
	if displayResult["count"] != float64(1) {
		t.Errorf("display count = %v, want 1", displayResult["count"])
	}
}

// --- Test 6: TTL-based expiration via ExpiresAt field ---

func TestPheromoneLifecycle_TTLExpiration(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := setupLifecycleStore(t)
	store = s

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	var buf bytes.Buffer
	stdout = &buf

	// Write a REDIRECT signal (gets 30-day TTL)
	rootCmd.SetArgs([]string{
		"pheromone-write",
		"--type", "REDIRECT",
		"--content", "avoid global state",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	writeOutput := parseLifecycleEnvelope(t, buf.String())
	writeResult := writeOutput["result"].(map[string]interface{})
	signal := writeResult["signal"].(map[string]interface{})
	sigID := signal["id"].(string)

	// Verify ExpiresAt was set
	expiresAt, ok := signal["expires_at"].(string)
	if !ok || expiresAt == "" {
		t.Fatal("REDIRECT signal should have expires_at set")
	}

	// Parse the expires_at and verify it's ~30 days in the future
	expiryTime, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		t.Fatalf("failed to parse expires_at: %v", err)
	}
	expectedExpiry := time.Now().UTC().Add(30 * 24 * time.Hour)
	diff := expectedExpiry.Sub(expiryTime)
	if diff < -time.Hour || diff > time.Hour {
		t.Errorf("expires_at = %v, expected ~30 days from now (off by %v)", expiresAt, diff)
	}

	// Write a FEEDBACK signal (gets 7-day TTL)
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{
		"pheromone-write",
		"--type", "FEEDBACK",
		"--content", "keep it simple",
	})
	rootCmd.Execute()
	feedbackOutput := parseLifecycleEnvelope(t, buf.String())
	feedbackSignal := feedbackOutput["result"].(map[string]interface{})["signal"].(map[string]interface{})
	feedbackExpires, ok := feedbackSignal["expires_at"].(string)
	if !ok || feedbackExpires == "" {
		t.Fatal("FEEDBACK signal should have expires_at set")
	}
	feedbackExpiry, err := time.Parse(time.RFC3339, feedbackExpires)
	if err != nil {
		t.Fatalf("failed to parse feedback expires_at: %v", err)
	}
	expectedFeedbackExpiry := time.Now().UTC().Add(7 * 24 * time.Hour)
	fbDiff := expectedFeedbackExpiry.Sub(feedbackExpiry)
	if fbDiff < -time.Hour || fbDiff > time.Hour {
		t.Errorf("feedback expires_at off by %v", fbDiff)
	}

	// Write a FOCUS signal (no ExpiresAt -- expires at phase end)
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{
		"pheromone-write",
		"--type", "FOCUS",
		"--content", "test coverage",
	})
	rootCmd.Execute()
	focusOutput := parseLifecycleEnvelope(t, buf.String())
	focusSignal := focusOutput["result"].(map[string]interface{})["signal"].(map[string]interface{})
	if focusSignal["expires_at"] != nil {
		t.Errorf("FOCUS signal should NOT have expires_at, got: %v", focusSignal["expires_at"])
	}

	// Expire the REDIRECT signal manually
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-expire", "--id", sigID})
	rootCmd.Execute()
	parseLifecycleEnvelope(t, buf.String())

	// Verify read shows 2 remaining (FEEDBACK + FOCUS)
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	rootCmd.Execute()
	readOutput := parseLifecycleEnvelope(t, buf.String())
	readSignals := readOutput["result"].(map[string]interface{})["signals"].([]interface{})
	if len(readSignals) != 2 {
		t.Errorf("expected 2 signals after expiring REDIRECT, got %d", len(readSignals))
	}
}

// --- Test 7: Custom priority and strength ---

func TestPheromoneLifecycle_CustomPriorityAndStrength(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := setupLifecycleStore(t)
	store = s

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	var buf bytes.Buffer
	stdout = &buf

	// Write FOCUS with custom high priority and low strength
	rootCmd.SetArgs([]string{
		"pheromone-write",
		"--type", "FOCUS",
		"--content", "security review",
		"--priority", "high",
		"--strength", "0.3",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	writeOutput := parseLifecycleEnvelope(t, buf.String())
	writeResult := writeOutput["result"].(map[string]interface{})
	signal := writeResult["signal"].(map[string]interface{})
	sigID := signal["id"].(string)

	if signal["priority"] != "high" {
		t.Errorf("priority = %v, want high", signal["priority"])
	}
	if signal["strength"] != float64(0.3) {
		t.Errorf("strength = %v, want 0.3", signal["strength"])
	}

	// Read and verify
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	rootCmd.Execute()
	readOutput := parseLifecycleEnvelope(t, buf.String())
	readSignals := readOutput["result"].(map[string]interface{})["signals"].([]interface{})
	if len(readSignals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(readSignals))
	}
	readSig := readSignals[0].(map[string]interface{})
	if readSig["priority"] != "high" {
		t.Errorf("read priority = %v, want high", readSig["priority"])
	}

	// Prime and verify the strength appears in section
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-prime"})
	rootCmd.Execute()
	primeOutput := parseLifecycleEnvelope(t, buf.String())
	primeResult := primeOutput["result"].(map[string]interface{})
	section := primeResult["section"].(string)
	if !strings.Contains(section, "0.3") {
		t.Errorf("prime section should contain strength 0.3, got: %s", section)
	}

	// Display and verify table format
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-display"})
	rootCmd.Execute()
	displayOutput := parseLifecycleEnvelope(t, buf.String())
	displayResult := displayOutput["result"].(map[string]interface{})
	displaySignals := displayResult["signals"].([]interface{})
	ds := displaySignals[0].(map[string]interface{})
	if ds["priority"] != "high" {
		t.Errorf("display priority = %v, want high", ds["priority"])
	}
	if ds["strength"] != float64(0.3) {
		t.Errorf("display strength = %v, want 0.3", ds["strength"])
	}

	// Expire and verify gone
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-expire", "--id", sigID})
	rootCmd.Execute()
	parseLifecycleEnvelope(t, buf.String())

	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-read"})
	rootCmd.Execute()
	goneOutput := parseLifecycleEnvelope(t, buf.String())
	goneSignals := goneOutput["result"].(map[string]interface{})["signals"].([]interface{})
	if len(goneSignals) != 0 {
		t.Errorf("expected 0 signals after expire, got %d", len(goneSignals))
	}
}

// --- Test 8: Expire non-existent signal ---

func TestPheromoneLifecycle_ExpireNonExistent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := setupLifecycleStore(t)
	store = s

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	var buf bytes.Buffer
	stderr = &buf

	rootCmd.SetArgs([]string{"pheromone-expire", "--id", "sig_nonexistent_123"})
	rootCmd.Execute()

	output := strings.TrimSpace(buf.String())
	if !strings.Contains(output, `"ok":false`) {
		t.Errorf("expected ok:false for non-existent signal, got: %s", output)
	}
	if !strings.Contains(output, "not found") {
		t.Errorf("error should mention 'not found', got: %s", output)
	}
}

// --- Test 9: Empty store lifecycle (no signals) ---

func TestPheromoneLifecycle_EmptyStore(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := setupLifecycleStore(t)
	store = s

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	var buf bytes.Buffer
	stdout = &buf

	// Read empty store
	rootCmd.SetArgs([]string{"pheromone-read"})
	rootCmd.Execute()
	readOutput := parseLifecycleEnvelope(t, buf.String())
	readSignals := readOutput["result"].(map[string]interface{})["signals"].([]interface{})
	if len(readSignals) != 0 {
		t.Errorf("expected 0 signals from empty store, got %d", len(readSignals))
	}

	// Prime empty store
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-prime"})
	rootCmd.Execute()
	primeOutput := parseLifecycleEnvelope(t, buf.String())
	primeResult := primeOutput["result"].(map[string]interface{})
	if primeResult["signal_count"] != float64(0) {
		t.Errorf("prime signal_count = %v, want 0", primeResult["signal_count"])
	}
	if primeResult["section"].(string) != "No active pheromone signals." {
		t.Errorf("prime section for empty = %q, want 'No active pheromone signals.'", primeResult["section"])
	}

	// Display empty store
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-display"})
	rootCmd.Execute()
	displayOutput := parseLifecycleEnvelope(t, buf.String())
	displayResult := displayOutput["result"].(map[string]interface{})
	if displayResult["count"] != float64(0) {
		t.Errorf("display count = %v, want 0", displayResult["count"])
	}

	// Count empty store
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-count"})
	rootCmd.Execute()
	countOutput := parseLifecycleEnvelope(t, buf.String())
	countResult := countOutput["result"].(map[string]interface{})
	if countResult["total"] != float64(0) {
		t.Errorf("count total = %v, want 0", countResult["total"])
	}
}

// --- Test 10: Display filtering by type ---

func TestPheromoneLifecycle_DisplayFilterByType(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := setupLifecycleStore(t)
	store = s

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	var buf bytes.Buffer
	stdout = &buf

	// Write 3 signals
	for _, args := range [][]string{
		{"pheromone-write", "--type", "FOCUS", "--content", "focus area"},
		{"pheromone-write", "--type", "REDIRECT", "--content", "avoid thing"},
		{"pheromone-write", "--type", "FEEDBACK", "--content", "good stuff"},
	} {
		buf.Reset()
		resetRootCmd(t)
		rootCmd.SetArgs(args)
		rootCmd.Execute()
		parseLifecycleEnvelope(t, buf.String())
	}

	// Display only REDIRECT
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-display", "--type", "REDIRECT"})
	rootCmd.Execute()
	displayOutput := parseLifecycleEnvelope(t, buf.String())
	displayResult := displayOutput["result"].(map[string]interface{})
	if displayResult["count"] != float64(1) {
		t.Errorf("display count for REDIRECT filter = %v, want 1", displayResult["count"])
	}
	displaySignals := displayResult["signals"].([]interface{})
	ds := displaySignals[0].(map[string]interface{})
	if ds["type"] != "REDIRECT" {
		t.Errorf("display type = %v, want REDIRECT", ds["type"])
	}

	// Display only FOCUS
	buf.Reset()
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"pheromone-display", "--type", "FOCUS"})
	rootCmd.Execute()
	focusOutput := parseLifecycleEnvelope(t, buf.String())
	focusResult := focusOutput["result"].(map[string]interface{})
	if focusResult["count"] != float64(1) {
		t.Errorf("display count for FOCUS filter = %v, want 1", focusResult["count"])
	}
}
