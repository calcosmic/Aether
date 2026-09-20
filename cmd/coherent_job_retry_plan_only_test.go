package cmd

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// setUpPartialCreditFixture stages the exact on-disk situation a grouped job
// leaves behind when it credits some of its tasks and then dies: the phase is
// mid-build (EXECUTING), four of six tasks are proven complete, at least one
// worker was genuinely spawned (so the phase carries a dispatch-start marker),
// and the parent attempt has reached its partial terminal state.
//
// It returns the repository root and the parent attempt's ID.
func setUpPartialCreditFixture(t *testing.T) (string, colony.Phase, colony.ColonyState, []string, string, codexBuildDispatch) {
	t.Helper()
	tasks, ids := sixChainedTasks()
	credited := ids[:4]
	for i := range tasks {
		if i < len(credited) {
			tasks[i].Status = colony.TaskCompleted
		}
	}
	goal := "Six-task grouped job"
	phase := colony.Phase{
		ID:     1,
		Name:   goal,
		Status: colony.PhaseInProgress,
		Tasks:  tasks,
	}
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		ColonyDepth:  "full",
		CurrentPhase: 1,
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			EvidencePolicy:   colony.PlanEvidenceNotRequired,
			Phases:           []colony.Phase{phase},
		},
	}

	startedAt := time.Now().UTC()
	dispatch := codexBuildDispatch{
		Name:             "Mason-1",
		Caste:            "builder",
		Stage:            "wave",
		TaskID:           ids[0],
		CoveredTaskIDs:   append([]string{}, ids...),
		JobName:          "automatic-six-step",
		Status:           "failed",
		CompletedTaskIDs: append([]string{}, credited...),
	}
	fixture := commitTestBuildStart(t, testBuildStartOptions{
		Variant: buildStartDirect, GeneratedAt: startedAt,
		SelectedTasks: ids, Dispatches: []codexBuildDispatch{dispatch},
		ExecutionOwner: "go-runtime", DispatchMode: "direct", MakeLatest: testBuildStartBool(true),
		PrepareRoot: func(root string) {
			createTestColonyState(t, filepath.Join(root, ".aether", "data"), state)
		},
	})
	root, phase, state, parentRel := fixture.Root, fixture.Phase, fixture.State, fixture.AttemptPath
	if err := transitionBuildAttempt(parentRel, buildAttemptPartial, "partial credit committed", []codexBuildDispatch{dispatch}, nil, "", nil); err != nil {
		t.Fatalf("mark parent attempt partial: %v", err)
	}
	var parent buildAttemptRecord
	if err := store.LoadJSON(parentRel, &parent); err != nil {
		t.Fatalf("load parent attempt: %v", err)
	}
	return root, phase, state, credited, parent.ID, dispatch
}

// TestPartialRetryCommandIsAcceptedOnThePlanOnlyPath is the permanent
// regression lock for the 195 review's third warning (WR-03).
//
// The owner is handed a recovery command after a partially credited build.
// The only build path the interactive wrapper ever takes is
// `aether build <N> ... --plan-only`. Creating the recovery attempt used to
// repoint the phase's "latest attempt" marker at a brand-new ACTIVE record
// while the phase's dispatch-start marker was still set from the parent
// build, so the very next plan-only call refused the owner's own recovery
// command with "already has workers in flight" -- and the recovery attempt
// it named had no completion packet to finalize, so there was no way out.
//
// This test does not inspect the marker or the attempt record. It runs the
// literal command the owner is handed, through the literal function the
// wrapper calls, and fails if the runtime refuses it.
func TestPartialRetryCommandIsAcceptedOnThePlanOnlyPath(t *testing.T) {
	saveGlobals(t)
	root, phase, state, credited, parentID, dispatch := setUpPartialCreditFixture(t)

	outcome, err := reconcilePartialBuildRetry(state, 1, phase, parentID, time.Now().UTC(), []codexBuildDispatch{dispatch})
	if err != nil {
		t.Fatalf("reconcilePartialBuildRetry: %v", err)
	}
	if outcome == nil {
		t.Fatal("expected a partial-retry outcome for a four-of-six grouped job")
	}

	phaseArg, taskArgs, force := parseRedispatchTaskArgs(t, outcome.RedispatchCommand)
	if phaseArg != "1" {
		t.Fatalf("recovery command targets phase %q, want 1: %q", phaseArg, outcome.RedispatchCommand)
	}

	// The wrapper's ONLY build path: `aether build <args> --plan-only`.
	result, _, _, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, 1, taskArgs, codexBuildOptions{Force: force})
	if err != nil {
		t.Fatalf("the recovery command the owner was handed (%q) is refused on the wrapper's plan-only path: %v", outcome.RedispatchCommand, err)
	}
	if result == nil {
		t.Fatal("plan-only returned no result for the recovery command")
	}

	plannedTasks := map[string]struct{}{}
	for _, d := range dispatches {
		for _, id := range dispatchCoveredTaskIDs(d) {
			plannedTasks[id] = struct{}{}
		}
	}
	for _, id := range credited {
		if _, redone := plannedTasks[id]; redone {
			t.Fatalf("the recovery command's plan-only manifest dispatches credited task %s again", id)
		}
	}
	for _, id := range outcome.UnfinishedTaskIDs {
		if _, ok := plannedTasks[id]; !ok {
			t.Fatalf("the recovery command's plan-only manifest never dispatches unfinished task %s", id)
		}
	}
}

// TestPartialRetryAttemptDoesNotBecomeTheLatestAttempt is the mechanism lock
// underneath the end-to-end test above: the D-10 recovery record is an
// append-only journal entry describing work still to do, not an attempt with
// workers in flight. Making it the phase's "latest attempt" is what made the
// runtime believe a partially-built phase still had live workers.
func TestPartialRetryAttemptDoesNotBecomeTheLatestAttempt(t *testing.T) {
	saveGlobals(t)
	_, phase, state, _, parentID, dispatch := setUpPartialCreditFixture(t)

	outcome, err := reconcilePartialBuildRetry(state, 1, phase, parentID, time.Now().UTC(), []codexBuildDispatch{dispatch})
	if err != nil {
		t.Fatalf("reconcilePartialBuildRetry: %v", err)
	}
	if outcome == nil {
		t.Fatal("expected a partial-retry outcome")
	}

	_, latest, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("phase 1 has no latest attempt after a partial credit")
	}
	if latest.ID == outcome.RetryAttemptID {
		t.Fatalf("the D-10 recovery attempt %s became phase 1's latest attempt with status %q; an active latest attempt plus the parent build's dispatch-start marker makes every later plan-only call refuse the owner's own recovery command", latest.ID, latest.Status)
	}
	if latest.ID != parentID {
		t.Fatalf("latest attempt = %s, want the parent %s", latest.ID, parentID)
	}
	if buildAttemptStatusActive(latest.Status) {
		t.Fatalf("phase 1's latest attempt %s is still active (%q) after a partial credit", latest.ID, latest.Status)
	}

	// The recovery record itself must still be discoverable in the journal --
	// not repointing the marker must not lose it.
	if _, _, found := findExistingBuildAttemptRetry(1, parentID); !found {
		t.Fatal("the D-10 recovery attempt is no longer discoverable in the phase's attempt journal")
	}
}
