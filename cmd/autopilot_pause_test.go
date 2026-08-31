package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestLegacyAutopilotCommandsAbsent(t *testing.T) {
	legacyCommands := []string{
		"autopilot-init",
		"autopilot-update",
		"autopilot-status",
		"autopilot-stop",
		"autopilot-set-headless",
		"autopilot-headless-check",
		"autopilot-resume",
	}
	wantedSupported := map[string]bool{"run": false, "status": false}
	for _, entry := range buildAuditCatalog(rootCmd) {
		for _, name := range legacyCommands {
			if entry.Name == name {
				t.Errorf("legacy autopilot command %q is still registered", name)
			}
		}
		if _, ok := wantedSupported[entry.Name]; ok {
			wantedSupported[entry.Name] = true
		}
	}
	for name, found := range wantedSupported {
		if !found {
			t.Errorf("supported autopilot lifecycle command %q is not registered", name)
		}
	}

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate autopilot removal test")
	}
	productionFiles, err := filepath.Glob(filepath.Join(filepath.Dir(testFile), "*.go"))
	if err != nil {
		t.Fatalf("list command sources: %v", err)
	}
	legacySymbols := []string{
		"autopilotInitCmd",
		"autopilotUpdateCmd",
		"autopilotStatusCmd",
		"autopilotStopCmd",
		"autopilotSetHeadlessCmd",
		"autopilotHeadlessCheckCmd",
		"autopilotResumeCmd",
		"checkAutopilotPauseConditions",
	}
	for _, path := range productionFiles {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", filepath.Base(path), err)
		}
		for _, symbol := range legacySymbols {
			if strings.Contains(string(source), symbol) {
				t.Errorf("legacy autopilot symbol %q remains in %s", symbol, filepath.Base(path))
			}
		}
	}
}

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
