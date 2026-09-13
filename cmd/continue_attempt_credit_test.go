package cmd

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// twoAttemptPhaseFixture seeds one phase with two tasks and drives the REAL
// build-start transaction twice against the same repository: attempt A proves
// task 1.1, attempt B proves task 1.2. Both attempts are genuine journal
// records written by commitBuildStart, never hand-authored JSON -- the
// alternation this reproduces is only meaningful if the attempt records are
// the exact shape the runtime writes.
//
// It returns the phase, the CURRENT continue manifest (attempt B's, because
// build/phase-N/manifest.json is overwritten by every build) and the root.
func twoAttemptPhaseFixture(t *testing.T) (colony.Phase, codexContinueManifest, string) {
	t.Helper()
	saveGlobals(t)

	taskOne, taskTwo := "1.1", "1.2"
	seedPhase := colony.Phase{ID: 1, Name: "Two partial attempts"}
	seedPhase.Tasks = []colony.Task{
		{ID: &taskOne, Goal: "Proven in the first attempt", Status: colony.TaskPending},
		{ID: &taskTwo, Goal: "Proven in the second attempt", Status: colony.TaskPending},
	}
	goal := "Prove continue credits every finalized attempt of the phase"
	seedState := colony.ColonyState{Goal: &goal, State: colony.StateREADY, Plan: colony.Plan{
		AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
		EvidencePolicy:   colony.PlanEvidenceNotRequired,
		Phases:           []colony.Phase{seedPhase},
	}}

	attemptA := time.Now().UTC().Add(-time.Hour)
	fixture := commitTestBuildStart(t, testBuildStartOptions{
		Variant: buildStartDirect, GeneratedAt: attemptA,
		SelectedTasks: []string{"1.1"},
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-1", Caste: "builder", TaskID: "1.1", CoveredTaskIDs: []string{"1.1"}, Status: "completed"},
		},
		ExecutionOwner: "go-runtime", DispatchMode: "direct", MakeLatest: testBuildStartBool(true),
		PrepareRoot: func(root string) {
			createTestColonyState(t, filepath.Join(root, ".aether", "data"), seedState)
		},
	})
	root := fixture.Root

	// Attempt B on the SAME repository: the second partial redispatch. This
	// overwrites build/phase-1/manifest.json with only its own dispatches.
	second := commitTestBuildStartAt(t, root, fixture.Phase.ID, time.Now().UTC(), testBuildStartOptions{
		Variant: buildStartDirect, GeneratedAt: time.Now().UTC(),
		SelectedTasks: []string{"1.2"},
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-2", Caste: "builder", TaskID: "1.2", CoveredTaskIDs: []string{"1.2"}, Status: "completed"},
		},
		ExecutionOwner: "go-runtime", DispatchMode: "direct", MakeLatest: testBuildStartBool(true),
	})

	records := listBuildAttemptsForPhase(second.Phase.ID)
	if len(records) < 2 {
		t.Fatalf("fixture must produce two real attempt records, got %d", len(records))
	}
	manifest := loadCodexContinueManifest(second.Phase.ID)
	if !manifest.Present {
		t.Fatalf("expected a current continue manifest for phase %d", second.Phase.ID)
	}
	// Guard the premise: the current manifest must carry ONLY attempt B's
	// dispatch. If a future change makes manifest.json cumulative, this test
	// is no longer exercising the reported condition and must be revisited
	// rather than silently passing for the wrong reason.
	covered := map[string]bool{}
	for _, d := range manifest.Data.Dispatches {
		for _, id := range dispatchCoveredTaskIDs(d) {
			covered[id] = true
		}
	}
	if covered["1.1"] {
		t.Fatalf("premise broken: the current manifest already carries attempt A's task 1.1; this test no longer reproduces the reported condition")
	}
	if !covered["1.2"] {
		t.Fatalf("premise broken: the current manifest does not carry attempt B's task 1.2, got %+v", manifest.Data.Dispatches)
	}
	return second.Phase, manifest, root
}

func passingContinueVerification(phase colony.Phase) codexContinueVerificationReport {
	return codexContinueVerificationReport{
		Phase:        phase.ID,
		ChecksPassed: true,
		Passed:       true,
		Claims:       codexClaimVerification{Present: true, Passed: true, Checked: len(phase.Tasks)},
	}
}

// TestContinueCreditsTasksProvenInAnEarlierAttempt proves the check step
// credits work proven in an EARLIER finalized attempt of the same phase, not
// only the latest one.
//
// Reported downstream on v1.0.74: a phase built across two partial attempts
// could never advance. build/phase-N/manifest.json is overwritten by every
// build, so the check step saw only the latest attempt's dispatches and
// reported the other attempt's tasks as having no evidence at all -- then
// handed back a recovery command naming exactly those tasks. Running it
// flipped the pair, and the loop never terminated.
func TestContinueCreditsTasksProvenInAnEarlierAttempt(t *testing.T) {
	phase, manifest, _ := twoAttemptPhaseFixture(t)
	verification := passingContinueVerification(phase)

	assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{}, time.Now().UTC())

	for _, task := range assessment.Tasks {
		if task.TaskID == "1.1" && (task.Outcome == "missing" || task.RecoveryAction == "redispatch") {
			t.Fatalf("task 1.1 was proven in an earlier finalized attempt but the check reports outcome=%q recovery=%q summary=%q",
				task.Outcome, task.RecoveryAction, task.Summary)
		}
	}
	if !assessment.PositiveEvidence {
		t.Fatalf("expected positive evidence across both finalized attempts, got blocking issues %v", assessment.BlockingIssues)
	}
	if !assessment.Passed {
		t.Fatalf("expected the check to pass with every task proven across two attempts, got %v", assessment.BlockingIssues)
	}
}

// TestPartialRedispatchRecoveryNeverNamesAlreadyProvenWork is the loop itself,
// asserted directly: the command the owner is handed must never ask them to
// redo a task an earlier finalized attempt already proved. This is the
// assertion that fails on the reported bug even if the block message changes.
func TestPartialRedispatchRecoveryNeverNamesAlreadyProvenWork(t *testing.T) {
	phase, manifest, _ := twoAttemptPhaseFixture(t)
	verification := passingContinueVerification(phase)

	assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{}, time.Now().UTC())

	for _, taskID := range assessment.RedispatchTasks {
		if taskID == "1.1" {
			t.Fatalf("recovery asks the owner to redo task 1.1, already proven in an earlier finalized attempt; redispatch set = %v", assessment.RedispatchTasks)
		}
	}
	for _, taskID := range assessment.Recovery.ReconcileTasks {
		if taskID == "1.1" {
			t.Fatalf("recovery asks the owner to reconcile task 1.1, already proven in an earlier finalized attempt; reconcile set = %v", assessment.Recovery.ReconcileTasks)
		}
	}
	if cmd := strings.TrimSpace(assessment.Recovery.ReconcileCommand); strings.Contains(cmd, "1.1") {
		t.Fatalf("recovery command names already-proven task 1.1: %q", cmd)
	}
}
