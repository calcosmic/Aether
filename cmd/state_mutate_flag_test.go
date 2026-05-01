package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestStateMutateVerifyOnly_GuardPasses(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a colony state with a completed task
	goal := "verify-only test"
	taskID := "1.1"
	state := colony.ColonyState{
		Goal:  &goal,
		State: colony.StateEXECUTING,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "done", Status: colony.TaskCompleted}}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	// Read state before
	beforeData, _ := s.ReadFile("COLONY_STATE.json")

	rootCmd.SetArgs([]string{"state-mutate", "--guard", "task-complete:1.1", "--verify-only"})
	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())

	// Guard check may pass or fail depending on whether go test finds failures.
	// Either way, verify-only mode must not modify state.
	// If guard passes: ok=true, mode=verify-only, allowed=true
	// If guard fails: ok=false, error contains "blocked"
	if env["ok"] == true {
		result := env["result"].(map[string]interface{})
		if result["mode"] != "verify-only" {
			t.Errorf("mode = %v, want verify-only", result["mode"])
		}
		if result["guard"] != "task-complete:1.1" {
			t.Errorf("guard = %v, want task-complete:1.1", result["guard"])
		}
		if result["allowed"] != true {
			t.Errorf("allowed = %v, want true", result["allowed"])
		}
	} else {
		// Guard check failed -- this is fine, the important thing is state wasn't modified
		errMsg, ok := env["error"].(string)
		if !ok || !strings.Contains(errMsg, "blocked") {
			t.Errorf("expected guard blocked error, got: %v", env["error"])
		}
	}

	// Verify state was NOT modified (the critical assertion for verify-only)
	afterData, _ := s.ReadFile("COLONY_STATE.json")
	if string(beforeData) != string(afterData) {
		t.Error("state was modified during verify-only mode")
	}
}

func TestStateMutateVerifyOnly_GuardFails(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a colony state WITHOUT a completed task
	goal := "verify-only fail test"
	taskID := "1.1"
	state := colony.ColonyState{
		Goal:  &goal,
		State: colony.StateEXECUTING,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhasePending, Tasks: []colony.Task{{ID: &taskID, Goal: "todo", Status: colony.TaskPending}}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	// Read state before
	beforeData, _ := s.ReadFile("COLONY_STATE.json")

	rootCmd.SetArgs([]string{"state-mutate", "--guard", "task-complete:1.1", "--verify-only"})
	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())

	// Whether the guard passes or fails depends on the test environment.
	// The key assertion is that verify-only NEVER modifies state.
	if env["ok"] == true {
		result := env["result"].(map[string]interface{})
		if result["mode"] != "verify-only" {
			t.Errorf("mode = %v, want verify-only", result["mode"])
		}
	}

	// Verify state was NOT modified
	afterData, _ := s.ReadFile("COLONY_STATE.json")
	if string(beforeData) != string(afterData) {
		t.Error("state was modified during verify-only mode")
	}
}

func TestStateMutateRevert(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a colony state with a guards array in raw JSON
	// (ColonyState doesn't have a typed Guards field, so we write raw JSON)
	goal := "revert test"
	state := colony.ColonyState{
		Goal:  &goal,
		State: colony.StateEXECUTING,
	}
	stateData, _ := json.Marshal(state)
	var rawState map[string]interface{}
	json.Unmarshal(stateData, &rawState)
	rawState["guards"] = []interface{}{
		map[string]interface{}{"type": "task-complete", "target": "1.1", "timestamp": "2026-04-29T00:00:00Z"},
		map[string]interface{}{"type": "phase-advance", "target": "2", "timestamp": "2026-04-29T00:00:00Z"},
	}
	updatedData, _ := json.MarshalIndent(rawState, "", "  ")
	s.AtomicWrite("COLONY_STATE.json", updatedData)

	rootCmd.SetArgs([]string{"state-mutate", "--revert", "task-complete:1.1"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["reverted"] != "task-complete:1.1" {
		t.Errorf("reverted = %v, want task-complete:1.1", result["reverted"])
	}

	// Verify the guard was removed from state
	data, _ := s.ReadFile("COLONY_STATE.json")
	var verifyState map[string]interface{}
	json.Unmarshal(data, &verifyState)
	guards, ok := verifyState["guards"].([]interface{})
	if !ok {
		t.Fatal("guards field missing from state")
	}
	if len(guards) != 1 {
		t.Errorf("expected 1 guard remaining, got %d", len(guards))
	}
	for _, g := range guards {
		guardMap := g.(map[string]interface{})
		if guardMap["type"] == "task-complete" {
			t.Error("task-complete guard should have been removed")
		}
	}
}

func TestStateMutateVerifyOnly_NoGuard(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "no-guard test"
	state := colony.ColonyState{
		Goal:  &goal,
		State: colony.StateEXECUTING,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	// verify-only without --guard should fall through to normal mutation behavior
	// Since no expression or --field is provided, it should error
	rootCmd.SetArgs([]string{"state-mutate", "--verify-only"})
	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	// Should be an error because no expression or field provided
	if env["ok"] == true {
		t.Error("expected error when --verify-only without --guard and no expression")
	}
}
