package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestAutopilotPauseOnBlockers(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Initialize autopilot
	rootCmd.SetArgs([]string{"autopilot-init", "--phases", "3"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-init failed: %v", err)
	}

	// Create a pending blocker decision
	pending := PendingDecisionFile{
		Decisions: []PendingDecision{
			{ID: "b1", Type: "blocker", Description: "Critical issue", Resolved: false, CreatedAt: "2026-01-01T00:00:00Z"},
		},
	}
	if err := store.SaveJSON("pending-decisions.json", pending); err != nil {
		t.Fatalf("save pending decisions: %v", err)
	}

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"autopilot-update", "--phase", "1", "--status", "completed"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-update failed: %v", err)
	}

	var state autopilotState
	if err := store.LoadJSON(autopilotStatePath, &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.Status != "paused" {
		t.Errorf("expected paused, got %q", state.Status)
	}
	if !strings.Contains(state.Reason, "active_blockers") {
		t.Errorf("expected reason to contain active_blockers, got %q", state.Reason)
	}
}

func TestAutopilotPauseOnGateFailure(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"autopilot-init", "--phases", "3"})
	_ = rootCmd.Execute()

	// Create colony state with failed gate
	cstate := colony.ColonyState{
		GateResults: []colony.GateResultEntry{
			{Name: "quality", Passed: false, Timestamp: "2026-01-01T00:00:00Z"},
		},
	}
	if err := store.SaveJSON("COLONY_STATE.json", cstate); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"autopilot-update", "--phase", "1", "--status", "completed"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-update failed: %v", err)
	}

	var state autopilotState
	_ = store.LoadJSON(autopilotStatePath, &state)
	if state.Status != "paused" {
		t.Errorf("expected paused, got %q", state.Status)
	}
	if !strings.Contains(state.Reason, "gate_failure") {
		t.Errorf("expected reason to contain gate_failure, got %q", state.Reason)
	}
}

func TestAutopilotResume(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"autopilot-init", "--phases", "3"})
	_ = rootCmd.Execute()

	// Pause autopilot with a blocker
	pending := PendingDecisionFile{
		Decisions: []PendingDecision{
			{ID: "b1", Type: "blocker", Description: "Issue", Resolved: false, CreatedAt: "2026-01-01T00:00:00Z"},
		},
	}
	_ = store.SaveJSON("pending-decisions.json", pending)

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"autopilot-update", "--phase", "1", "--status", "completed"})
	_ = rootCmd.Execute()

	var state autopilotState
	_ = store.LoadJSON(autopilotStatePath, &state)
	if state.Status != "paused" {
		t.Fatalf("expected paused before resume, got %q", state.Status)
	}

	// Resolve the blocker and resume
	pending.Decisions[0].Resolved = true
	_ = store.SaveJSON("pending-decisions.json", pending)

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"autopilot-resume"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-resume failed: %v", err)
	}

	_ = store.LoadJSON(autopilotStatePath, &state)
	if state.Status != "running" {
		t.Errorf("expected running after resume, got %q", state.Status)
	}
	if state.Reason != "" {
		t.Errorf("expected empty reason after resume, got %q", state.Reason)
	}
}

func TestAutopilotResumeWhenNotPaused(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"autopilot-init", "--phases", "3"})
	_ = rootCmd.Execute()

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"autopilot-resume"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-resume failed: %v", err)
	}

	// Should return ok=false or indicate not paused
	// The command outputs JSON, we just verify it doesn't crash
}
