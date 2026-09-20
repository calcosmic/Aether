package cmd

import (
	"sort"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// oneTaskBugFixPhase builds the measured field case this test proves
// against: an ordinary one-task defect fix, carrying none of the five
// named-risk signals (194-CONTEXT.md D-01) and no performance, security or
// research vocabulary -- the exact shape of phase that was measured at
// eight workers on 2026-08-22 (five at build: builder, watcher, auditor,
// probe, tracker; three at continue: watcher, auditor, probe), against
// three to four in the pre-v1.26 version of Aether
// (.planning/research/v1.27-milestone-brief.md).
func oneTaskBugFixPhase() colony.Phase {
	taskID := "1.1"
	return colony.Phase{
		ID:          42,
		Name:        "Fix the retry loop",
		Description: "The retry loop never backs off between attempts and hammers the endpoint",
		Mode:        colony.PhaseModePrototype,
		Status:      colony.PhaseReady,
		Tasks: []colony.Task{
			{ID: &taskID, Goal: "Add exponential backoff to the retry loop", Status: colony.TaskPending},
		},
	}
}

// countWorkersAcrossBothBoundaries runs the SAME phase and state through
// both dispatch-list constructors -- plannedBuildDispatchesWithJudgement
// (build) and plannedContinueReviewDispatches (continue) -- and returns the
// TOTAL dispatch count summed across both boundaries plus the sorted union
// of castes dispatched. The owner never experiences "build" and "continue"
// as two separate numbers; they feel one team, so this is the one function
// this plan's own tests call for that number, rather than each writing its
// own counter that could silently disagree with another's (194-08-PLAN.md's
// own warning: "two counters that could disagree is how a proof stops
// proving anything").
//
// root is fixed at "/tmp" and the invoker is codex.FakeInvoker{} -- the
// same no-op harness cmd/phase_verified_once_test.go
// (TestPhaseVerifiedOnce) and cmd/continue_fastpath_castes_test.go already
// use for this exact call shape, since neither dispatch constructor
// performs I/O beyond composing the dispatch list itself.
func countWorkersAcrossBothBoundaries(phase colony.Phase, state colony.ColonyState, reviewDepth colony.VerificationDepth, proposedCastes []string, casteReason string, reasons map[string]string) (total int, castes []string) {
	buildDispatches := testPlannedBuildDispatchesWithJudgement(phase, state, nil, reviewDepth, proposedCastes, casteReason, reasons)
	continueDispatches := plannedContinueReviewDispatches(
		"/tmp", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{},
		&codex.FakeInvoker{}, time.Minute, reviewDepth, proposedCastes, casteReason, reasons,
	)

	seen := map[string]bool{}
	for _, d := range buildDispatches {
		seen[d.Caste] = true
	}
	for _, d := range continueDispatches {
		seen[d.Caste] = true
	}
	castes = make([]string, 0, len(seen))
	for c := range seen {
		castes = append(castes, c)
	}
	sort.Strings(castes)

	return len(buildDispatches) + len(continueDispatches), castes
}

// TestOneTaskBugFixIsOneWorkerPlusChecks is the number the owner actually
// feels. A one-task bug fix was measured at EIGHT AI worker dispatches on
// 2026-08-22 -- five at build (builder, watcher, auditor, probe, tracker),
// three at continue (watcher, auditor, probe) -- against three to four in
// the pre-v1.26 version of Aether (194-CONTEXT.md, .planning/research/
// v1.27-milestone-brief.md). Plans 194-01 through 194-07, combined, are
// meant to bring that down to one worker on both the proposal path and the
// no-proposal fallback path, counted across build and continue together.
// This test measures AI worker dispatches only -- it does not re-measure
// Phase 193's deterministic floor (the program's own build/test/lint
// checks), which runs regardless and is the "plus checks" half of the
// phase's own name for this outcome.
func TestOneTaskBugFixIsOneWorkerPlusChecks(t *testing.T) {
	saveGlobalsCmd(t)
	phase := oneTaskBugFixPhase()
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}

	const baseline = 8

	// The fallback path: nil proposal, nil reasons -- autopilot, or any
	// wrapper that never re-fetches with --castes. D-11 is the retune that
	// makes this path cost what the judged path costs.
	fallbackTotal, fallbackCastes := countWorkersAcrossBothBoundaries(phase, state, colony.VerificationDepthStandard, nil, "", nil)
	if fallbackTotal != 1 {
		t.Fatalf("fallback path: one-task bug fix dispatched %d worker(s) (%v), want exactly 1 -- measured %d on 2026-08-22; a climb back toward that number is the regression this test exists to catch", fallbackTotal, fallbackCastes, baseline)
	}
	if len(fallbackCastes) != 1 || fallbackCastes[0] != "builder" {
		t.Fatalf("fallback path: one-task bug fix dispatched %v, want exactly [builder]", fallbackCastes)
	}

	// The proposal path: the chat names the implementation worker with a
	// per-worker reason (D-08). D-11's own point is that this path and the
	// fallback path must now cost the same -- before this milestone the
	// automatic (no-proposal) path was the MORE expensive one.
	proposalTotal, proposalCastes := countWorkersAcrossBothBoundaries(
		phase, state, colony.VerificationDepthStandard,
		[]string{"builder"}, "the phase describes one ordinary bug fix",
		map[string]string{"builder": "writes the backoff fix for the retry loop"},
	)
	if proposalTotal != 1 {
		t.Fatalf("proposal path: one-task bug fix dispatched %d worker(s) (%v), want exactly 1 -- measured %d on 2026-08-22; a climb back toward that number is the regression this test exists to catch", proposalTotal, proposalCastes, baseline)
	}

	// The guard row: an explicit heavy request must still buy the full
	// review panel at the checking step, on the SAME phase. A proof that
	// can only pass by the pipeline having gone quiet proves nothing --
	// this is the v1.26 lesson (194-CONTEXT.md, "Established Patterns": "a
	// test that can only pass proves nothing").
	heavyTotal, heavyCastes := countWorkersAcrossBothBoundaries(phase, state, colony.VerificationDepthHeavy, nil, "", nil)
	if heavyTotal <= 1 {
		t.Fatalf("heavy request on the same phase dispatched only %d worker(s) (%v) -- an explicit --heavy request must still produce the full review panel, or this proof would keep passing even if the pipeline had gone inert", heavyTotal, heavyCastes)
	}
}
