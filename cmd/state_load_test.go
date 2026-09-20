package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestLoadActiveColonyStateReadOnlyLeavesNormalStateUnchanged(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Inspect a normal colony without writing"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Ready phase",
			Status: colony.PhaseReady,
			Tasks:  []colony.Task{{ID: &taskID, Goal: "Stay ready", Status: colony.TaskPending}},
		}}},
		Events: []string{"2026-09-01T10:00:00Z|initialized|init|Ready for work"},
	})

	before := snapshotProjectDataTree(t, store.BasePath())
	state, err := loadActiveColonyStateReadOnly()
	if err != nil {
		t.Fatalf("loadActiveColonyStateReadOnly returned error: %v", err)
	}
	after := snapshotProjectDataTree(t, store.BasePath())

	if state.CurrentPhase != 1 || len(state.Plan.Phases) != 1 {
		t.Fatalf("read-only state = phase %d with %d plan phases, want phase 1 with one plan phase", state.CurrentPhase, len(state.Plan.Phases))
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("read-only normal load changed saved data:\nbefore: %v\nafter:  %v", before, after)
	}
}

func TestLoadActiveColonyStateReadOnlyRepairsStringCurrentPhaseInMemory(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s
	raw := []byte(`{
  "version": "3.0",
  "goal": "Stage a legacy numeric repair",
  "state": "READY",
  "current_phase": "1",
  "plan": {"phases": [{"id": 1, "name": "Legacy phase", "status": "ready", "tasks": []}]},
  "events": ["2026-09-01T10:00:00Z|initialized|init|Legacy fixture"]
}`)
	if err := store.SaveRawJSON("COLONY_STATE.json", raw); err != nil {
		t.Fatalf("failed to save raw state: %v", err)
	}

	before := snapshotProjectDataTree(t, store.BasePath())
	state, err := loadActiveColonyStateReadOnly()
	if err != nil {
		t.Fatalf("loadActiveColonyStateReadOnly returned error: %v", err)
	}
	after := snapshotProjectDataTree(t, store.BasePath())

	if state.CurrentPhase != 1 {
		t.Fatalf("in-memory current_phase = %d, want 1", state.CurrentPhase)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("read-only compatibility repair changed saved data:\nbefore: %v\nafter:  %v", before, after)
	}
	persistedRaw, err := store.LoadRawJSON("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("failed to reload raw state: %v", err)
	}
	if !strings.Contains(string(persistedRaw), `"current_phase": "1"`) {
		t.Fatalf("read-only compatibility repair rewrote current_phase: %s", persistedRaw)
	}
	if strings.Contains(string(persistedRaw), "state_repaired|load") {
		t.Fatalf("read-only compatibility repair persisted an event: %s", persistedRaw)
	}
}

func TestLoadActiveColonyStateReadOnlyStagesMissingPlanWithoutPersisting(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Stage a missing plan without writing"
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

	before := snapshotProjectDataTree(t, store.BasePath())
	state, err := loadActiveColonyStateReadOnly()
	if err != nil {
		t.Fatalf("loadActiveColonyStateReadOnly returned error: %v", err)
	}
	after := snapshotProjectDataTree(t, store.BasePath())

	if len(state.Plan.Phases) != 3 || state.CurrentPhase != 3 {
		t.Fatalf("in-memory recovered state = phase %d with %d plan phases, want phase 3 with three plan phases", state.CurrentPhase, len(state.Plan.Phases))
	}
	if len(state.Events) == 0 || !strings.Contains(state.Events[len(state.Events)-1], "plan_recovered|state|Recovered 3 phases") {
		t.Fatalf("expected in-memory plan recovery event, got %v", state.Events)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("read-only missing-plan staging changed saved data:\nbefore: %v\nafter:  %v", before, after)
	}

	var persisted colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &persisted); err != nil {
		t.Fatalf("failed to reload persisted state: %v", err)
	}
	if len(persisted.Plan.Phases) != 0 {
		t.Fatalf("read-only missing-plan staging persisted %d phases", len(persisted.Plan.Phases))
	}
	if strings.Contains(strings.Join(persisted.Events, "\n"), "plan_recovered|state") {
		t.Fatalf("read-only missing-plan staging persisted a repair event: %v", persisted.Events)
	}
}

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

func TestNormalizeLegacyColonyStateHandlesCommonRuntimeValues(t *testing.T) {
	goal := "Normalize production state drift"
	phase := colony.Phase{ID: 1, Name: "Recover", Status: colony.PhaseReady}
	cases := []struct {
		name         string
		rawState     colony.State
		withGoal     bool
		withPlan     bool
		currentPhase int
		want         colony.State
	}{
		{name: "lowercase ready", rawState: colony.State("ready"), withGoal: true, withPlan: true, currentPhase: 1, want: colony.StateREADY},
		{name: "done", rawState: colony.State("done"), withGoal: true, withPlan: true, currentPhase: 1, want: colony.StateCOMPLETED},
		{name: "complete", rawState: colony.State("COMPLETE"), withGoal: true, withPlan: true, currentPhase: 1, want: colony.StateCOMPLETED},
		{name: "finished", rawState: colony.State("finished"), withGoal: true, withPlan: true, currentPhase: 1, want: colony.StateCOMPLETED},
		{name: "active", rawState: colony.State("active"), withGoal: true, withPlan: true, currentPhase: 1, want: colony.StateEXECUTING},
		{name: "running", rawState: colony.State("running"), withGoal: true, withPlan: true, currentPhase: 1, want: colony.StateEXECUTING},
		{name: "build", rawState: colony.State("build"), withGoal: true, withPlan: true, currentPhase: 1, want: colony.StateBUILT},
		{name: "empty with goal", rawState: colony.State(""), withGoal: true, want: colony.StateREADY},
		{name: "empty without context", rawState: colony.State(""), want: colony.StateIDLE},
		{name: "unknown with plan", rawState: colony.State("garbage"), withPlan: true, currentPhase: 1, want: colony.StateREADY},
		{name: "unknown without context", rawState: colony.State("garbage"), want: colony.StateIDLE},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := colony.ColonyState{
				Version:      "3.0",
				State:        tc.rawState,
				CurrentPhase: tc.currentPhase,
			}
			if tc.withGoal {
				state.Goal = &goal
			}
			if tc.withPlan {
				state.Plan = colony.Plan{Phases: []colony.Phase{phase}}
			}
			got := normalizeLegacyColonyState(state)
			if got.State != tc.want {
				t.Fatalf("normalized state = %q, want %q", got.State, tc.want)
			}
		})
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
	stderr = &out

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

	// D-04's confirmation gate (198-03): pre-record the answer, the same
	// way an owner running seal twice (ask, then confirm) would.
	autoRecordSealConfirmationForTest(t, s)

	rootCmd.SetArgs([]string{"seal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal returned error: %v", err)
	}

	summaryPath := filepath.Join(root, ".aether", "CROWNED-ANTHILL.md")
	if _, err := os.Stat(summaryPath); !os.IsNotExist(err) {
		t.Fatalf("seal must not reconstruct a missing plan or write a closure summary, stat err=%v", err)
	}

	var persisted colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &persisted); err != nil {
		t.Fatalf("failed to reload sealed state: %v", err)
	}
	if len(persisted.Plan.Phases) != 0 {
		t.Fatalf("seal must retain the missing-plan state for owner repair, got %d phases", len(persisted.Plan.Phases))
	}
	if persisted.CurrentPhase != 2 {
		t.Fatalf("current_phase = %d, want 2", persisted.CurrentPhase)
	}
	if !strings.Contains(out.String(), "no colony plan exists to verify") {
		t.Fatalf("expected typed missing-plan refusal, got %s", out.String())
	}
}

func TestStateLoadMigratesPlanningLegacyPlanAndPersistsWithNextSafeWrite(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s
	goal := "Keep a pre-Phase-200 plan buildable"
	taskID := "1.1"
	legacy := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Legacy active phase",
			Status: colony.PhaseReady,
			Tasks:  []colony.Task{{ID: &taskID, Goal: "Still build this task", Status: colony.TaskPending}},
		}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", legacy); err != nil {
		t.Fatalf("save legacy state: %v", err)
	}

	loaded, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState returned error: %v", err)
	}
	if loaded.Plan.AcceptancePolicy != colony.PlanAcceptanceLegacyUnbound {
		t.Fatalf("loaded acceptance policy = %q, want %q", loaded.Plan.AcceptancePolicy, colony.PlanAcceptanceLegacyUnbound)
	}
	if loaded.CurrentPhase != legacy.CurrentPhase || !reflect.DeepEqual(loaded.Plan.Phases, legacy.Plan.Phases) {
		t.Fatalf("loaded migration changed buildable plan: before=%+v after=%+v", legacy.Plan.Phases, loaded.Plan.Phases)
	}
	if firstBuildablePhase(loaded.Plan.Phases) != 1 {
		t.Fatalf("first buildable phase = %d, want 1", firstBuildablePhase(loaded.Plan.Phases))
	}
	if loaded.Specification != nil || len(loaded.Plan.Candidates) != 0 {
		t.Fatalf("loaded migration fabricated authority: specification=%+v candidates=%+v", loaded.Specification, loaded.Plan.Candidates)
	}

	beforeWrite, err := store.LoadRawJSON("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read state after in-memory migration: %v", err)
	}
	var notYetPersisted colony.ColonyState
	if err := json.Unmarshal(beforeWrite, &notYetPersisted); err != nil {
		t.Fatalf("decode state after in-memory migration: %v", err)
	}
	if notYetPersisted.Plan.AcceptancePolicy != "" {
		t.Fatalf("state load implicitly persisted acceptance policy %q", notYetPersisted.Plan.AcceptancePolicy)
	}

	// A real state-changing command saves the already-migrated value through
	// Store's existing safe writer; no migration-specific sidecar or direct
	// filesystem write is needed.
	if err := store.SaveJSON("COLONY_STATE.json", loaded); err != nil {
		t.Fatalf("persist migrated state through safe writer: %v", err)
	}
	firstRaw, err := store.LoadRawJSON("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read safely persisted migration: %v", err)
	}
	var persisted colony.ColonyState
	if err := json.Unmarshal(firstRaw, &persisted); err != nil {
		t.Fatalf("decode safely persisted migration: %v", err)
	}
	if persisted.Plan.AcceptancePolicy != colony.PlanAcceptanceLegacyUnbound {
		t.Fatalf("safely persisted acceptance policy = %q, want %q", persisted.Plan.AcceptancePolicy, colony.PlanAcceptanceLegacyUnbound)
	}

	if _, err := loadActiveColonyState(); err != nil {
		t.Fatalf("second loadActiveColonyState returned error: %v", err)
	}
	secondRaw, err := store.LoadRawJSON("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read second migrated state: %v", err)
	}
	if string(firstRaw) != string(secondRaw) {
		t.Fatalf("second state-load migration rewrote state:\nfirst:  %s\nsecond: %s", firstRaw, secondRaw)
	}
}

func TestStateLoadReadOnlyMigratesPlanningInMemoryWithoutPersistence(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s
	goal := "Inspect a legacy plan without writing"
	taskID := "1.1"
	legacy := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID: 1, Name: "Legacy phase", Status: colony.PhaseReady,
			Tasks: []colony.Task{{ID: &taskID, Goal: "Remain pending", Status: colony.TaskPending}},
		}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", legacy); err != nil {
		t.Fatalf("save legacy state: %v", err)
	}
	before, err := store.LoadRawJSON("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read legacy state before load: %v", err)
	}

	loaded, err := loadActiveColonyStateReadOnly()
	if err != nil {
		t.Fatalf("loadActiveColonyStateReadOnly returned error: %v", err)
	}
	if loaded.Plan.AcceptancePolicy != colony.PlanAcceptanceLegacyUnbound {
		t.Fatalf("in-memory acceptance policy = %q, want %q", loaded.Plan.AcceptancePolicy, colony.PlanAcceptanceLegacyUnbound)
	}
	after, err := store.LoadRawJSON("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read legacy state after load: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("read-only planning migration persisted state:\nbefore: %s\nafter:  %s", before, after)
	}
}

func TestStateLoadRejectsPlanningArtifactThatEscapesRepository(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, root := newTestStore(t)
	store = s
	goal := "Reject escaped legacy planning evidence"
	legacy := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Legacy phase", Status: colony.PhaseReady}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", legacy); err != nil {
		t.Fatalf("save legacy state: %v", err)
	}
	external := filepath.Join(filepath.Dir(root), "outside-planning.json")
	if err := os.WriteFile(external, []byte(`{"authority":"must not be imported"}`), 0o644); err != nil {
		t.Fatalf("write external fixture: %v", err)
	}
	planningDir := filepath.Join(store.BasePath(), "planning")
	if err := os.MkdirAll(planningDir, 0o755); err != nil {
		t.Fatalf("create planning directory: %v", err)
	}
	if err := os.Symlink(external, filepath.Join(planningDir, "iteration-state.json")); err != nil {
		t.Fatalf("create escaped artifact symlink: %v", err)
	}

	before, err := store.LoadRawJSON("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read state before load: %v", err)
	}
	_, err = loadActiveColonyState()
	if err == nil || !strings.Contains(err.Error(), "resolves outside .aether/data/planning") {
		t.Fatalf("load error = %v, want planning containment refusal", err)
	}
	after, readErr := store.LoadRawJSON("COLONY_STATE.json")
	if readErr != nil {
		t.Fatalf("read state after refused load: %v", readErr)
	}
	if string(before) != string(after) {
		t.Fatalf("refused planning migration changed state:\nbefore: %s\nafter:  %s", before, after)
	}
}
