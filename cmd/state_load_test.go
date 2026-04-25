package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestLoadActiveColonyStateNormalizesLegacyPausedState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Normalize old paused colonies"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.State("PAUSED"),
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Legacy paused phase", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: &taskID, Goal: "Resume safely", Status: colony.TaskInProgress}}},
			},
		},
	})

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState returned error: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY", state.State)
	}
	if !state.Paused {
		t.Fatal("expected legacy PAUSED state to normalize with paused flag set")
	}
}

func TestLoadActiveColonyStateNormalizesBrokenIdleStateWithGoal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Recover broken idle colony"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateIDLE,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Restorable phase", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: &taskID, Goal: "Finish restoring", Status: colony.TaskInProgress}}},
			},
		},
	})

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState returned error: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY", state.State)
	}
}

func TestLoadActiveColonyStateRepairsMissingPlanFromPlanningArtifact(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Repair a colony that lost plan.phases"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		Plan:         colony.Plan{Phases: []colony.Phase{}},
		Events: []string{
			"2026-04-21T07:50:00Z phase-1-complete: Audit complete.",
			"2026-04-21T08:20:00Z phase-2-complete: Standard designed.",
		},
	})
	if err := store.SaveJSON("planning/phase-plan.json", codexWorkerPlanArtifact{
		Confidence: codexPlanConfidence{Overall: 82},
		Phases: []codexWorkerPlanPhase{
			{Name: "Audit", Tasks: []codexWorkerPlanTask{{Goal: "Complete the audit"}}},
			{Name: "Design", Tasks: []codexWorkerPlanTask{{Goal: "Design the standard"}}},
			{Name: "Rollout", Tasks: []codexWorkerPlanTask{{Goal: "Apply the schema"}}},
		},
	}); err != nil {
		t.Fatalf("failed to save planning artifact: %v", err)
	}

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState returned error: %v", err)
	}
	if len(state.Plan.Phases) != 3 {
		t.Fatalf("repaired plan phase count = %d, want 3", len(state.Plan.Phases))
	}
	if state.CurrentPhase != 3 {
		t.Fatalf("current_phase = %d, want 3", state.CurrentPhase)
	}
	if state.Plan.GeneratedAt == nil {
		t.Fatal("expected repaired plan generated_at to be set")
	}
	if state.Plan.Confidence == nil || *state.Plan.Confidence != 0.82 {
		t.Fatalf("plan confidence = %v, want 0.82", state.Plan.Confidence)
	}
	if state.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Fatalf("phase 1 status = %s, want completed", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[1].Status != colony.PhaseCompleted {
		t.Fatalf("phase 2 status = %s, want completed", state.Plan.Phases[1].Status)
	}
	if state.Plan.Phases[2].Status != colony.PhaseReady {
		t.Fatalf("phase 3 status = %s, want ready", state.Plan.Phases[2].Status)
	}
	if len(state.Events) == 0 || !strings.Contains(state.Events[len(state.Events)-1], "plan_recovered|state|Recovered 3 phases") {
		t.Fatalf("expected repair event, got %v", state.Events)
	}

	var persisted colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &persisted); err != nil {
		t.Fatalf("failed to reload repaired state: %v", err)
	}
	if len(persisted.Plan.Phases) != 3 || persisted.CurrentPhase != 3 {
		t.Fatalf("persisted state was not repaired: %+v", persisted)
	}
}

func TestLoadActiveColonyStateRepairsStringCurrentPhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s
	raw := []byte(`{
  "version": "3.0",
  "goal": "Repair legacy string current phase",
  "state": "READY",
  "current_phase": "1",
  "plan": {"phases": []},
  "events": []
}`)
	if err := store.SaveRawJSON("COLONY_STATE.json", raw); err != nil {
		t.Fatalf("failed to save raw state: %v", err)
	}

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState returned error: %v", err)
	}
	if state.CurrentPhase != 1 {
		t.Fatalf("current_phase = %d, want 1", state.CurrentPhase)
	}
	if len(state.Events) == 0 || !strings.Contains(state.Events[len(state.Events)-1], "state_repaired|load|Normalized legacy numeric string fields") {
		t.Fatalf("expected state repair event, got %v", state.Events)
	}

	persistedRaw, err := store.LoadRawJSON("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("failed to reload raw state: %v", err)
	}
	var persisted map[string]json.RawMessage
	if err := json.Unmarshal(persistedRaw, &persisted); err != nil {
		t.Fatalf("persisted state is invalid JSON: %v", err)
	}
	if got := string(persisted["current_phase"]); got != "1" {
		t.Fatalf("persisted current_phase = %s, want numeric 1", got)
	}
}

func TestLoadActiveColonyStateRejectsNonNumericStringCurrentPhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s
	raw := []byte(`{
  "version": "3.0",
  "goal": "Reject bad legacy current phase",
  "state": "READY",
  "current_phase": "phase-one",
  "plan": {"phases": []},
  "events": []
}`)
	if err := store.SaveRawJSON("COLONY_STATE.json", raw); err != nil {
		t.Fatalf("failed to save raw state: %v", err)
	}

	if _, err := loadActiveColonyState(); err == nil {
		t.Fatal("loadActiveColonyState returned nil error for non-numeric current_phase")
	}

	persistedRaw, err := store.LoadRawJSON("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("failed to reload raw state: %v", err)
	}
	if !strings.Contains(string(persistedRaw), `"current_phase": "phase-one"`) {
		t.Fatalf("state was unexpectedly rewritten: %s", persistedRaw)
	}
	if strings.Contains(string(persistedRaw), "state_repaired") {
		t.Fatalf("unexpected repair event in unrepaired state: %s", persistedRaw)
	}
}

func TestSealRecoversMissingPlanFromPlanningArtifactWithoutPlanRef(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, root := newTestStore(t)
	store = s
	var out strings.Builder
	stdout = &out

	goal := "Seal with recovered plan"
	if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateCOMPLETED,
		CurrentPhase: 2,
		Plan:         colony.Plan{Phases: []colony.Phase{}},
		Events: []string{
			"2026-04-21T07:50:00Z|phase_advanced|continue|Completed phase 1, ready for phase 2",
			"2026-04-21T08:20:00Z|phase_completed|continue|Completed final phase 2",
		},
	}); err != nil {
		t.Fatalf("failed to save colony state: %v", err)
	}
	if err := store.SaveJSON("planning/phase-plan.json", codexWorkerPlanArtifact{
		Confidence: codexPlanConfidence{Overall: 91},
		Phases: []codexWorkerPlanPhase{
			{Name: "Recover first phase", Tasks: []codexWorkerPlanTask{{Goal: "Recover first task"}}},
			{Name: "Recover final phase", Tasks: []codexWorkerPlanTask{{Goal: "Recover final task"}}},
		},
	}); err != nil {
		t.Fatalf("failed to save planning artifact: %v", err)
	}

	rootCmd.SetArgs([]string{"seal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal returned error: %v", err)
	}

	summaryPath := filepath.Join(root, ".aether", "CROWNED-ANTHILL.md")
	if _, err := os.Stat(summaryPath); err != nil {
		t.Fatalf("expected seal summary at %s: %v", summaryPath, err)
	}

	var persisted colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &persisted); err != nil {
		t.Fatalf("failed to reload sealed state: %v", err)
	}
	if len(persisted.Plan.Phases) != 2 {
		t.Fatalf("persisted phase count = %d, want 2", len(persisted.Plan.Phases))
	}
	for _, phase := range persisted.Plan.Phases {
		if phase.Status != colony.PhaseCompleted {
			t.Fatalf("phase %d status = %s, want completed", phase.ID, phase.Status)
		}
	}
	if persisted.CurrentPhase != 2 {
		t.Fatalf("current_phase = %d, want 2", persisted.CurrentPhase)
	}
	events := strings.Join(persisted.Events, "\n")
	if !strings.Contains(events, "plan_recovered|state|Recovered 2 phases") {
		t.Fatalf("expected plan recovery event, got %v", persisted.Events)
	}
	if !strings.Contains(events, "sealed|seal|Colony sealed at Crowned Anthill") {
		t.Fatalf("expected seal event, got %v", persisted.Events)
	}
}
