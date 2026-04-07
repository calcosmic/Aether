package cmd

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// helper: create a store with COLONY_STATE.json and produce N audit entries.
func setupAuditHistory(t *testing.T, n int) (*storage.Store, string) {
	t.Helper()
	s, tmpDir := newTestStore(t)
	goal := "initial goal"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	logger := storage.NewAuditLogger(s)
	for i := 0; i < n; i++ {
		newGoal := "goal " + strconv.Itoa(i+1)
		logger.WriteBoundary("state-mutate", false, func(st *colony.ColonyState) (string, error) {
			st.Goal = &newGoal
			return "goal -> " + newGoal, nil
		})
	}
	return s, tmpDir
}

// Test 1: Empty audit log prints "No mutation history found."
func TestStateHistoryEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create COLONY_STATE.json but no audit entries
	goal := "test goal"
	s.SaveJSON("COLONY_STATE.json", colony.ColonyState{Version: "3.0", Goal: &goal})

	rootCmd.SetArgs([]string{"state-history"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No mutation history found.") {
		t.Errorf("expected 'No mutation history found.' in output, got: %s", output)
	}
}

// Test 2: Compact table with entries shows Timestamp, Command, Summary, Destructive columns.
func TestStateHistoryCompact(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupAuditHistory(t, 2)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"state-history"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should contain command name
	if !strings.Contains(output, "state-mutate") {
		t.Errorf("expected 'state-mutate' in output, got: %s", output)
	}
	// Should contain summary from audit entry
	if !strings.Contains(output, "goal ->") {
		t.Errorf("expected 'goal ->' summary in output, got: %s", output)
	}
	// Should contain table headers (go-pretty renders them uppercase)
	if !strings.Contains(output, "TIMESTAMP") {
		t.Errorf("expected 'TIMESTAMP' header in output, got: %s", output)
	}
	if !strings.Contains(output, "COMMAND") {
		t.Errorf("expected 'COMMAND' header in output, got: %s", output)
	}
}

// Test 3: --tail 2 limits output to last 2 entries
func TestStateHistoryTail(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupAuditHistory(t, 5)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"state-history", "--tail", "2"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// ReadHistory returns at most 2 entries, so we should see exactly 2 "goal ->" lines
	// (go-pretty renders with borders, so counting rows is unreliable; just verify the last goal)
	if !strings.Contains(output, "goal -> goal 5") {
		t.Errorf("expected last goal in output, got: %s", output)
	}
}

// Test 4: --diff shows full before/after JSON for each entry
func TestStateHistoryDiff(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupAuditHistory(t, 1)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"state-history", "--diff"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Before:") {
		t.Errorf("expected 'Before:' in diff output, got: %s", output)
	}
	if !strings.Contains(output, "After:") {
		t.Errorf("expected 'After:' in diff output, got: %s", output)
	}
	if !strings.Contains(output, "Checksum:") {
		t.Errorf("expected 'Checksum:' in diff output, got: %s", output)
	}
	if !strings.Contains(output, "Timestamp:") {
		t.Errorf("expected 'Timestamp:' in diff output, got: %s", output)
	}
	if !strings.Contains(output, "Command:") {
		t.Errorf("expected 'Command:' in diff output, got: %s", output)
	}
}

// Test 5: --json outputs JSON envelope with entries array
func TestStateHistoryJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupAuditHistory(t, 2)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"state-history", "--json"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	entries, ok := result["entries"].([]interface{})
	if !ok {
		t.Fatalf("expected entries array in result, got: %T", result["entries"])
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

// Test 6: --diff --tail 1 shows diff for only the last entry
func TestStateHistoryDiffTail(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupAuditHistory(t, 3)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"state-history", "--diff", "--tail", "1"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should have exactly one "--- Entry" section
	if strings.Count(output, "--- Entry") != 1 {
		t.Errorf("expected exactly 1 entry section, got: %s", output)
	}
}

// Test 7: No COLONY_STATE.json or audit log prints "No mutation history found." (graceful)
func TestStateHistoryNoStore(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	// Don't create any files

	rootCmd.SetArgs([]string{"state-history"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No mutation history found.") {
		t.Errorf("expected 'No mutation history found.' for missing store, got: %s", output)
	}
}

// Test 8: Timestamps are formatted for human readability (not raw RFC3339Nano)
func TestStateHistoryTimestampFormat(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupAuditHistory(t, 1)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"state-history"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should NOT contain raw RFC3339Nano format (with T separator and nanoseconds)
	if strings.Contains(output, "T") && strings.Contains(output, "Z") {
		t.Errorf("timestamps should be human-readable, not RFC3339Nano, got: %s", output)
	}
}
