package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestRecoveryTaskDoesNotFreezeTheAcceptedPlanRecord reproduces the
// 2026-09-25 French Fluency dead end: a blocked check appended recovery
// tasks to an explicit_owner phase, the phase later advanced, and the
// accepted revision's copy of phase/task status stopped being updated
// because syncActivePlanRevisionExecutionFacts gave up on any phase whose
// live task count differed from the revision's. Every later load then
// refused with "active plan phases do not match active revision" and the
// owner lost resume, pause, run and continue at once.
func TestRecoveryTaskDoesNotFreezeTheAcceptedPlanRecord(t *testing.T) {
	goal := "Prove a recovery task never freezes the accepted plan record"
	first, second, third := "1.1", "1.2", "2.1"
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
		Version: "3.0", Goal: &goal, ColonyDepth: "standard",
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "First", Status: colony.PhaseInProgress, Tasks: []colony.Task{
				{ID: &first, Goal: "Do the first thing", Status: colony.TaskCompleted},
				{ID: &second, Goal: "Do the second thing", Status: colony.TaskCompleted},
			}},
			{ID: 2, Name: "Second", Status: colony.PhasePending, Tasks: []colony.Task{
				{ID: &third, Goal: "Do the third thing", Status: colony.TaskPending},
			}},
		}},
	})
	state := accepted.State
	if err := validateCurrentPlanningState(state); err != nil {
		t.Fatalf("fixture must start valid: %v", err)
	}

	// A blocked check writes recovery tasks back onto phase 1, exactly as
	// the continue path does.
	recovery := recoveryTasksForBlockedContinue(state.Plan.Phases[0], codexContinueAssessment{
		Recovery: codexContinueRecoveryPlan{ReconcileTasks: []string{first, second}},
	})
	if added := appendRecoveryTasks(&state.Plan.Phases[0], recovery); added != 2 {
		t.Fatalf("appendRecoveryTasks added %d, want 2", added)
	}
	syncActivePlanRevisionExecutionFacts(&state.Plan)
	if err := validateCurrentPlanningState(state); err != nil {
		t.Fatalf("state with recovery tasks refused before any advance: %v", err)
	}

	// The recovery tasks are reconciled and the phase advances.
	for i := range state.Plan.Phases[0].Tasks {
		state.Plan.Phases[0].Tasks[i].Status = colony.TaskCompleted
	}
	state.Plan.Phases[0].Status = colony.PhaseCompleted
	state.Plan.Phases[1].Status = colony.PhaseReady
	syncActivePlanRevisionExecutionFacts(&state.Plan)

	if err := validateCurrentPlanningState(state); err != nil {
		t.Fatalf("advancing a phase that carries recovery tasks froze the accepted plan record: %v", err)
	}
	// Autopilot and build ask the same question a second way, by hashing
	// the live phases against the accepted proposal. That view must not
	// count recovery tasks either, or the owner sees "Missing: accepted
	// plan authority" with nothing to run.
	facts := lifecycleFactsFromStateSnapshot(state, false, time.Now().UTC())
	bindings := loadPlanAuthorityVerifiedBindings(accepted.Root, facts)
	if decision := validateAcceptedPlanAuthority(facts, bindings); !decision.Eligible {
		t.Fatalf("recovery tasks cost the plan its accepted authority: %s (%s)", decision.RefusalCode, decision.Diagnostic)
	}

	for _, revision := range state.Plan.Revisions {
		if revision.ID != state.Plan.ActiveRevisionID {
			continue
		}
		if got := len(revision.Phases[0].Tasks); got != 2 {
			t.Fatalf("accepted revision phase 1 has %d tasks, want the 2 it was accepted with (recovery tasks must never enter it)", got)
		}
		if revision.Phases[1].Status != colony.PhaseReady {
			t.Fatalf("accepted revision phase 2 status = %q, want ready", revision.Phases[1].Status)
		}
	}
}

// TestStatusDriftLeftByTheOldSyncHealsOnLoad covers a project already written
// by the old sync (the French Fluency state on disk): the accepted revision's
// status is stale while the live plan has advanced. Loading must re-copy the
// status and nothing else -- a live plan whose accepted content genuinely
// changed is still refused.
func TestStatusDriftLeftByTheOldSyncHealsOnLoad(t *testing.T) {
	goal := "Prove stale revision status heals and real edits do not"
	first, second := "1.1", "2.1"
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
		Version: "3.0", Goal: &goal, ColonyDepth: "standard",
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "First", Status: colony.PhaseInProgress, Tasks: []colony.Task{
				{ID: &first, Goal: "Do the first thing", Status: colony.TaskCompleted},
			}},
			{ID: 2, Name: "Second", Status: colony.PhasePending, Tasks: []colony.Task{
				{ID: &second, Goal: "Do the second thing", Status: colony.TaskPending},
			}},
		}},
	})
	stale := accepted.State
	appendRecoveryTasks(&stale.Plan.Phases[0], recoveryTasksForBlockedContinue(stale.Plan.Phases[0], codexContinueAssessment{
		Recovery: codexContinueRecoveryPlan{ReconcileTasks: []string{first}},
	}))
	// Advance the live plan WITHOUT syncing, the shape the old sync left.
	stale.Plan.Phases[0].Status = colony.PhaseCompleted
	stale.Plan.Phases[0].Tasks[1].Status = colony.TaskCompleted
	stale.Plan.Phases[1].Status = colony.PhaseReady
	if reflect.DeepEqual(planPhasesExcludingRecoveryTasks(stale.Plan.Phases), activeRevisionPhasesForTest(stale)) {
		t.Fatal("fixture must reproduce the stale on-disk shape (revision status behind the live plan)")
	}

	healed, changed := healActiveRevisionStatusDrift(stale)
	if !changed {
		t.Fatal("stale revision status was not healed")
	}
	if err := validateCurrentPlanningState(healed); err != nil {
		t.Fatalf("healed state still refused: %v", err)
	}
	if again, changedAgain := healActiveRevisionStatusDrift(healed); changedAgain || validateCurrentPlanningState(again) != nil {
		t.Fatal("healing is not idempotent")
	}

	edited := stale
	edited.Plan.Phases = clonePhases(stale.Plan.Phases)
	edited.Plan.Phases[1].Tasks[0].Goal = "A goal the owner never accepted"
	edited, _ = healActiveRevisionStatusDrift(edited)
	if err := validateCurrentPlanningState(edited); err == nil {
		t.Fatal("healing status let an unaccepted change to a task through")
	}
}

// TestStaleRevisionStatusOnDiskLoadsOnEveryPath is blocker 2 of the Phase
// 210 fortnight: the first fix healed stale revision status in one loader,
// but '/ant-build 2' reads the project through the in-session loader, which
// validated the raw state and refused again with "active plan phases do not
// match active revision". The stale file here is written to disk exactly as
// the old sync left it, and every loader and the plan-authority check must
// accept it.
func TestStaleRevisionStatusOnDiskLoadsOnEveryPath(t *testing.T) {
	goal := "Prove every loader heals stale revision status"
	first, second := "1.1", "2.1"
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
		Version: "3.0", Goal: &goal, ColonyDepth: "standard",
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "First", Status: colony.PhaseInProgress, Tasks: []colony.Task{
				{ID: &first, Goal: "Do the first thing", Status: colony.TaskCompleted},
			}},
			{ID: 2, Name: "Second", Status: colony.PhasePending, Tasks: []colony.Task{
				{ID: &second, Goal: "Do the second thing", Status: colony.TaskPending},
			}},
		}},
	})
	stale := accepted.State
	appendRecoveryTasks(&stale.Plan.Phases[0], recoveryTasksForBlockedContinue(stale.Plan.Phases[0], codexContinueAssessment{
		Recovery: codexContinueRecoveryPlan{ReconcileTasks: []string{first}},
	}))
	stale.Plan.Phases[0].Status = colony.PhaseCompleted
	stale.Plan.Phases[0].Tasks[1].Status = colony.TaskCompleted
	stale.Plan.Phases[1].Status = colony.PhaseReady
	stale.CurrentPhase = 2
	raw, err := json.MarshalIndent(stale, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(accepted.Root, ".aether", "data", "COLONY_STATE.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := loadSpecificationColonyState(accepted.Root); err != nil {
		t.Fatalf("specification loader refused stale revision status: %v", err)
	}
	if err := withPlanningMutationSession(accepted.Root, "test-stale-revision-status", func(session *planningMutationSession) error {
		loaded, err := loadSpecificationColonyStateInSession(session)
		if err != nil {
			return err
		}
		return validatePlanningState(loaded)
	}); err != nil {
		t.Fatalf("in-session loader (the /ant-build path) refused stale revision status: %v", err)
	}
	if !lifecycleAcceptedPlanValid(stale) {
		t.Fatal("lifecycle facts called the stale-status plan unaccepted")
	}
}

func activeRevisionPhasesForTest(state colony.ColonyState) []colony.Phase {
	for _, revision := range state.Plan.Revisions {
		if revision.ID == state.Plan.ActiveRevisionID {
			return revision.Phases
		}
	}
	return nil
}
