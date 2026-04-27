package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// setupSealTestStore creates a fresh temp store with a minimal colony state
// where all phases are completed, ready for seal.
func setupSealTestStore(t *testing.T) (*storage.Store, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	goal := "Test colony goal"
	state := colony.ColonyState{
		Goal:         &goal,
		CurrentPhase: 1,
		State:        colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Discovery", Status: colony.PhaseCompleted},
			},
		},
		Memory: colony.Memory{
			PhaseLearnings: []colony.PhaseLearning{},
		},
		Events: []string{},
	}

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Create local QUEEN.md for promotion tests
	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	if err := os.WriteFile(queenPath, []byte(queenDefaultContent), 0644); err != nil {
		t.Fatal(err)
	}

	return s, tmpDir
}

// runSealCmd runs the seal command with the given args and returns stdout/stderr output.
func runSealCmd(t *testing.T, s *storage.Store, tmpDir string, args []string) (string, string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	t.Setenv("COLONY_DATA_DIR", dataDir)

	store = s
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	stdout = outBuf
	stderr = errBuf

	allArgs := append([]string{"seal"}, args...)
	rootCmd.SetArgs(allArgs)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.Execute()

	return outBuf.String(), errBuf.String()
}

// TestSealBlockerCheck verifies that seal blocks when blocker flags exist.
func TestSealBlockerCheck(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add a blocker flag
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{
				ID:          "blk-001",
				Type:        "blocker",
				Description: "Critical issue blocking seal",
				Resolved:    false,
				CreatedAt:   "2026-04-27",
				Source:      "test",
			},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatal(err)
	}

	_, errOut := runSealCmd(t, s, tmpDir, nil)

	// Should output error containing BLOCKED
	if !strings.Contains(errOut, "BLOCKED") {
		t.Errorf("expected error output to contain 'BLOCKED', got: %s", errOut)
	}
	if !strings.Contains(errOut, "blk-001") {
		t.Errorf("expected error output to contain blocker ID 'blk-001', got: %s", errOut)
	}

	// Verify colony state was NOT mutated (still READY, not COMPLETED)
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State == colony.StateCOMPLETED {
		t.Error("seal should not have mutated colony state when blockers exist")
	}
}

// TestSealForceBlockers verifies that seal --force proceeds despite blockers.
func TestSealForceBlockers(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add a blocker flag
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{
				ID:          "blk-002",
				Type:        "blocker",
				Description: "Critical issue",
				Resolved:    false,
				CreatedAt:   "2026-04-27",
				Source:      "test",
			},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatal(err)
	}

	out, _ := runSealCmd(t, s, tmpDir, []string{"--force"})

	// Should contain the warning about overriding
	if !strings.Contains(out, "WARNING: Overriding") {
		t.Errorf("expected stdout to contain override warning, got: %s", out)
	}

	// Verify colony state WAS mutated (COMPLETED)
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Errorf("expected state COMPLETED, got: %s", state.State)
	}
}

// TestSealIssueWarning verifies that seal proceeds with a warning when issues exist but no blockers.
func TestSealIssueWarning(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add an issue (not blocker) flag
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{
				ID:          "issue-001",
				Type:        "issue",
				Description: "Non-critical issue",
				Resolved:    false,
				CreatedAt:   "2026-04-27",
				Source:      "test",
			},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatal(err)
	}

	out, _ := runSealCmd(t, s, tmpDir, nil)

	// Should contain the NOTE about unresolved issues
	if !strings.Contains(out, "NOTE:") {
		t.Errorf("expected stdout to contain NOTE about issues, got: %s", out)
	}

	// Verify colony state WAS mutated (seal proceeded)
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Errorf("expected state COMPLETED, got: %s", state.State)
	}
}

// TestCheckSealBlockers unit tests the checkSealBlockers helper.
func TestCheckSealBlockers(t *testing.T) {
	s, _ := setupSealTestStore(t)

	// No flags file: should return empty
	blockers, issues := checkSealBlockers(s)
	if len(blockers) != 0 || len(issues) != 0 {
		t.Errorf("expected empty with no flags file, got %d blockers, %d issues", len(blockers), len(issues))
	}

	// Mixed flags
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{ID: "b1", Type: "blocker", Resolved: false},
			{ID: "b2", Type: "blocker", Resolved: true},
			{ID: "i1", Type: "issue", Resolved: false},
			{ID: "n1", Type: "note", Resolved: false},
		},
	}
	_ = s.SaveJSON("pending-decisions.json", flags)

	blockers, issues = checkSealBlockers(s)
	if len(blockers) != 1 || blockers[0].ID != "b1" {
		t.Errorf("expected 1 unresolved blocker 'b1', got %d: %v", len(blockers), blockers)
	}
	if len(issues) != 1 || issues[0].ID != "i1" {
		t.Errorf("expected 1 unresolved issue 'i1', got %d: %v", len(issues), issues)
	}
}

// TestRenderBlockerSummary verifies the blocker summary table output.
func TestRenderBlockerSummary(t *testing.T) {
	blockers := []colony.FlagEntry{
		{ID: "blk-001", Description: "Critical blocker", Type: "blocker", CreatedAt: "2026-04-27"},
	}
	issues := []colony.FlagEntry{
		{ID: "issue-001", Description: "Non-critical", Type: "issue", CreatedAt: "2026-04-27"},
	}

	out := renderBlockerSummary(blockers, issues)

	if !strings.Contains(out, "blk-001") {
		t.Error("summary should contain blocker ID")
	}
	if !strings.Contains(out, "BLOCKED") {
		t.Error("summary should contain BLOCKED message")
	}
	if !strings.Contains(out, "--resolve") {
		t.Error("summary should contain resolution hint")
	}
	if !strings.Contains(out, "issue-severity") {
		t.Error("summary should mention issue-severity flags")
	}
}

// TestCountResolvedFlags unit tests the countResolvedFlags helper.
func TestCountResolvedFlags(t *testing.T) {
	s, _ := setupSealTestStore(t)

	// No flags file
	count := countResolvedFlags(s)
	if count != 0 {
		t.Errorf("expected 0 with no flags file, got %d", count)
	}

	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{ID: "b1", Resolved: true},
			{ID: "b2", Resolved: true},
			{ID: "i1", Resolved: false},
		},
	}
	_ = s.SaveJSON("pending-decisions.json", flags)

	count = countResolvedFlags(s)
	if count != 2 {
		t.Errorf("expected 2 resolved, got %d", count)
	}
}

// TestSealBlockerSummaryJSON verifies the error output is valid JSON.
func TestSealBlockerSummaryJSON(t *testing.T) {
	s, _ := setupSealTestStore(t)

	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{ID: "blk-json", Type: "blocker", Description: "JSON test blocker", Resolved: false, CreatedAt: "2026-04-27", Source: "test"},
		},
	}
	_ = s.SaveJSON("pending-decisions.json", flags)

	// Use JSON output mode (no visual rendering)
	saveGlobals(t)
	resetRootCmd(t)
	store = s
	dataDir := s.BasePath()
	t.Setenv("COLONY_DATA_DIR", dataDir)
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	stdout = outBuf
	stderr = errBuf
	os.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Cleanup(func() { os.Unsetenv("AETHER_OUTPUT_MODE") })

	rootCmd.SetArgs([]string{"seal"})
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.Execute()

	// stderr should be valid JSON envelope
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(errBuf.String())), &envelope); err != nil {
		t.Fatalf("expected valid JSON error, got: %s", errBuf.String())
	}
	if ok, _ := envelope["ok"].(bool); ok {
		t.Error("expected ok:false in error envelope")
	}

	// State should not be mutated
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State == colony.StateCOMPLETED {
		t.Error("seal should not have completed when blockers exist")
	}
}
