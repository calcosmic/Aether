package cmd

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// boundaryDoubleDispatchPhase builds a production-mode phase fixture for
// these tests, mirroring probeGatingPhase / fallbackTeamPhase's style
// elsewhere in this package.
func boundaryDoubleDispatchPhase(id int, name, description string) colony.Phase {
	return colony.Phase{
		ID:          id,
		Name:        name,
		Description: description,
		Mode:        colony.PhaseModeProduction,
		Status:      colony.PhaseReady,
		Tasks:       []colony.Task{{Goal: description}},
	}
}

// attemptWithVerificationBoundaryRecorded derives, saves, and marks latest a
// minimal real build attempt for phaseID (via newTestVerificationBoundaryAttempt,
// cmd/verification_boundary_test.go's own production-constructor fixture),
// then reconciles and attaches the requested boundary choice to it, and
// finally writes the latest-attempt pointer so loadLatestBuildAttempt(phaseID)
// -- the exact read path queenBuildPostWaveDispatches and
// plannedContinueReviewDispatches now use -- resolves it. Requires a store to
// already be installed (saveGlobals(t) + newTestStore(t) + store assignment).
func attemptWithVerificationBoundaryRecorded(t *testing.T, phaseID int, attemptID string, choice string, reason string) string {
	t.Helper()
	attemptRel := newTestVerificationBoundaryAttempt(t, phaseID, attemptID)
	decision := queenApplyVerificationBoundary(choice, reason, colony.Phase{}, colony.ColonyState{})
	if err := attachVerificationBoundary(attemptRel, decision); err != nil {
		t.Fatalf("attach verification boundary: %v", err)
	}
	if err := store.SaveJSON(latestBuildAttemptPointerPath(phaseID), latestBuildAttemptPointer{
		SchemaVersion: buildAttemptSchemaVersion,
		AttemptID:     attemptID,
		Path:          attemptRel,
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("write latest-attempt pointer: %v", err)
	}
	return attemptRel
}

// TestNoCasteIsDispatchedAtBothBoundaries closes .planning/WINDOWS.md #1.
//
// Before Phase 194, a production-mode phase whose wording named a security
// or blast-radius risk drew the same required specialist independently at
// BOTH boundaries: the build side via queenBuildSafetyRequiredCastes' own
// mode/security branches (deleted by 194-02), the continue side via
// isAlwaysRequired's continue case -- one caste, computed twice, dispatched
// twice, with nobody having asked for it twice. That is the exact defect
// 193-02-SUMMARY.md flagged and .planning/WINDOWS.md recorded as entry #1,
// scoped to this phase to fix.
//
// This test walks the REAL manifest-facing dispatch constructors for both
// boundaries -- plannedBuildDispatchesWithJudgement,
// plannedContinueReviewDispatches -- the same discipline
// cmd/phase_verified_once_test.go's TestPhaseVerifiedOnce already
// established for the narrower "watcher" claim. That test's own scope note
// names this wider probe/auditor/gatekeeper gap as explicitly NOT its own
// territory and hands it to Phase 194; this test is where it lands.
func TestNoCasteIsDispatchedAtBothBoundaries(t *testing.T) {
	saveGlobalsCmd(t)
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}

	for _, tc := range []struct {
		name  string
		phase colony.Phase
	}{
		{
			name:  "production, security signal (credentials/auth), no proposal",
			phase: boundaryDoubleDispatchPhase(1, "Password reset", "Let users reset their password via an emailed token"),
		},
		{
			name:  "production, migration signal, no proposal",
			phase: boundaryDoubleDispatchPhase(2, "Migrate user data to new schema", "Design the new table shape and write the migration script"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buildDispatches := testPlannedBuildDispatchesWithJudgement(tc.phase, state, nil, colony.VerificationDepthStandard, nil, "")
			buildCastes := map[string]bool{}
			for _, d := range buildDispatches {
				buildCastes[d.Caste] = true
			}

			continueDispatches := plannedContinueReviewDispatches(
				"/tmp", tc.phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{},
				&codex.FakeInvoker{}, time.Minute, colony.VerificationDepthStandard, nil, "",
			)
			continueCastes := map[string]bool{}
			for _, d := range continueDispatches {
				continueCastes[d.Caste] = true
			}

			// The fixture must actually exercise a forced reviewer, or the
			// intersection check below would pass vacuously on an inert
			// pipeline (the v1.26 lesson: a test that can only pass proves
			// nothing).
			if len(continueCastes) == 0 {
				t.Fatalf("fixture is broken: no reviewer was forced at continue for phase %q -- this fixture must contain a named-risk signal (194-CONTEXT.md D-01) or the intersection check below proves nothing", tc.phase.Name)
			}

			intersection := map[string]bool{}
			for caste := range buildCastes {
				if continueCastes[caste] {
					intersection[caste] = true
				}
			}
			if len(intersection) != 0 {
				t.Fatalf("caste(s) dispatched at BOTH the build and continue boundaries with no Queen proposal: %v (build=%v continue=%v) -- this is the exact defect recorded as .planning/WINDOWS.md #1", intersection, buildCastes, continueCastes)
			}

			// Reuse task 1's own counter (countWorkersAcrossBothBoundaries)
			// for the caste union rather than a second, independently
			// written walker: if a caste were double-dispatched across the
			// two boundaries, the summed dispatch total would exceed the
			// distinct-caste union (the overlapping caste counted twice in
			// the sum, once in the union). This must hold exactly when the
			// intersection check above holds -- two proofs of the same
			// fact, computed by the same arithmetic, so they cannot
			// silently disagree (194-08-PLAN.md's own warning).
			total, union := countWorkersAcrossBothBoundaries(tc.phase, state, colony.VerificationDepthStandard, nil, "", nil)
			if total != len(union) {
				t.Fatalf("countWorkersAcrossBothBoundaries: total dispatch count %d != distinct caste count %d (%v) -- a caste is being dispatched more than once across the build and continue boundaries", total, len(union), union)
			}
		})
	}

	// Phase 201-05 (D-05): the loop above proves no double-dispatch under the
	// UNRECORDED (check-step-default) boundary. Strengthen this test to walk
	// the same phase fixtures against the REAL, explicitly recorded boundary
	// -- both check_step and build_end -- so a future regression that makes
	// either dispatcher stop reading the recorded decision (and fall back to
	// re-deriving from phase content) is caught by this same guard, at every
	// boundary value the runtime can actually produce.
	s, _ := newTestStore(t)
	store = s
	for _, tc := range []struct {
		name    string
		phaseID int
		choice  string
	}{
		{"recorded check-step boundary: no double dispatch, continue still reviews", 11, verificationBoundaryChoiceCheckStep},
		{"recorded build-end boundary: no double dispatch, continue dispatches nothing", 12, verificationBoundaryChoiceBuildEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			phase := boundaryDoubleDispatchPhase(tc.phaseID, "Password reset", "Let users reset their password via an emailed token")
			attemptWithVerificationBoundaryRecorded(t, phase.ID, fmt.Sprintf("attempt-recorded-boundary-%d", phase.ID), tc.choice, "double-dispatch regression fixture")

			buildDispatches := testPlannedBuildDispatchesWithJudgement(phase, state, nil, colony.VerificationDepthStandard, nil, "")
			buildCastes := map[string]bool{}
			for _, d := range buildDispatches {
				buildCastes[d.Caste] = true
			}

			continueDispatches := plannedContinueReviewDispatches(
				"/tmp", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{},
				&codex.FakeInvoker{}, time.Minute, colony.VerificationDepthStandard, nil, "",
			)
			continueCastes := map[string]bool{}
			for _, d := range continueDispatches {
				continueCastes[d.Caste] = true
			}

			if tc.choice == verificationBoundaryChoiceBuildEnd {
				// D-05: with judgement recorded to land at build-end, the
				// check step dispatches NO reviewer of its own -- it reads
				// the build-end findings back instead
				// (continueReviewReportFromBuildEndFindings), proven
				// separately by TestCheckStepReviewersGateOnTheRecordedBoundary.
				if len(continueCastes) != 0 {
					t.Fatalf("recorded build-end boundary: check step still dispatched %v, want none", continueCastes)
				}
			} else {
				// D-01: the check-step default still forces its own
				// reviewer for this phase's credentials signal -- a recorded
				// check_step choice must not silently suppress it.
				if len(continueCastes) == 0 {
					t.Fatalf("recorded check-step boundary: no reviewer was forced at continue for phase %q", phase.Name)
				}
			}

			intersection := map[string]bool{}
			for caste := range buildCastes {
				if continueCastes[caste] {
					intersection[caste] = true
				}
			}
			if len(intersection) != 0 {
				t.Fatalf("caste(s) dispatched at BOTH the build and continue boundaries under a %q recorded decision: %v (build=%v continue=%v)", tc.choice, intersection, buildCastes, continueCastes)
			}
		})
	}
}

// TestBuildEndReviewersGateOnTheRecordedBoundary is Task 1's own proof
// (201-05-PLAN.md): queenBuildPostWaveDispatches reads (never re-derives)
// the verification-boundary decision recorded on the phase's current build
// attempt, and dispatches a post-wave reviewer only when that record names
// build-end.
func TestBuildEndReviewersGateOnTheRecordedBoundary(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	queenCastes := map[string]bool{"auditor": true, "measurer": true, "chaos": true}

	t.Run("no recorded boundary dispatches nothing", func(t *testing.T) {
		phase := boundaryDoubleDispatchPhase(201, "Release hardening", "Harden the release signoff path")
		post := queenBuildPostWaveDispatches(phase, queenCastes, 9)
		if len(post) != 0 {
			t.Fatalf("expected no post-wave dispatches with no recorded boundary, got %+v", post)
		}
	})

	t.Run("recorded check-step boundary dispatches nothing", func(t *testing.T) {
		phase := boundaryDoubleDispatchPhase(202, "Release hardening", "Harden the release signoff path")
		attemptWithVerificationBoundaryRecorded(t, phase.ID, "attempt-gate-check-step-202", verificationBoundaryChoiceCheckStep, "low risk, review at continue")
		post := queenBuildPostWaveDispatches(phase, queenCastes, 9)
		if len(post) != 0 {
			t.Fatalf("expected no post-wave dispatches with a recorded check-step boundary, got %+v", post)
		}
	})

	t.Run("recorded build-end boundary dispatches exactly the justified reviewers, each with a reason", func(t *testing.T) {
		phase := boundaryDoubleDispatchPhase(203, "Release hardening", "Harden the release signoff path")
		attemptWithVerificationBoundaryRecorded(t, phase.ID, "attempt-gate-build-end-203", verificationBoundaryChoiceBuildEnd, "release sign-off")
		post := queenBuildPostWaveDispatches(phase, queenCastes, 9)
		if len(post) != 3 {
			t.Fatalf("expected exactly 3 post-wave reviewers (auditor, measurer, chaos), got %d: %+v", len(post), post)
		}
		gotCastes := map[string]bool{}
		for _, d := range post {
			gotCastes[d.Caste] = true
			if strings.TrimSpace(d.Task) == "" {
				t.Errorf("dispatch %s carries no task/reason: %+v", d.Caste, d)
			}
			if d.ExecutionWave != 9 {
				t.Errorf("dispatch %s is on wave %d, want the requested review wave 9", d.Caste, d.ExecutionWave)
			}
		}
		for _, want := range []string{"auditor", "measurer", "chaos"} {
			if !gotCastes[want] {
				t.Errorf("expected %s among the build-end reviewers, got %v", want, gotCastes)
			}
		}
	})

	t.Run("a caste the Queen never selected is never dispatched, even under a build-end boundary", func(t *testing.T) {
		phase := boundaryDoubleDispatchPhase(204, "Release hardening", "Harden the release signoff path")
		attemptWithVerificationBoundaryRecorded(t, phase.ID, "attempt-gate-build-end-204", verificationBoundaryChoiceBuildEnd, "release sign-off")
		post := queenBuildPostWaveDispatches(phase, map[string]bool{"auditor": true}, 9)
		if len(post) != 1 || post[0].Caste != "auditor" {
			t.Fatalf("expected exactly [auditor], got %+v", post)
		}
	})
}

// TestReviewerCountAtZeroOneAndCeiling proves the three threshold cases
// Task 1 calls out explicitly: zero forced reviewers dispatches none, one
// forced reviewer dispatches exactly that one, and the configured depth
// ceiling (queenBuildPostWavePlans' own full set -- auditor, measurer,
// chaos) dispatches the ceiling and no more. Every count below is read as an
// exact integer off the real dispatch list, never rounded or estimated.
func TestReviewerCountAtZeroOneAndCeiling(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	for _, tc := range []struct {
		name        string
		phaseID     int
		queenCastes map[string]bool
		want        int
	}{
		{"zero forced reviewers dispatches none", 211, map[string]bool{}, 0},
		{"exactly one forced reviewer dispatches exactly that one", 212, map[string]bool{"auditor": true}, 1},
		{"at the configured ceiling dispatches the ceiling and no more", 213, map[string]bool{"auditor": true, "measurer": true, "chaos": true}, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			phase := boundaryDoubleDispatchPhase(tc.phaseID, "Reviewer count fixture", "Prove the reviewer count matches the selected team exactly")
			attemptWithVerificationBoundaryRecorded(t, phase.ID, fmt.Sprintf("attempt-reviewer-count-%d", phase.ID), verificationBoundaryChoiceBuildEnd, "reviewer count fixture")
			post := queenBuildPostWaveDispatches(phase, tc.queenCastes, 9)
			if got := len(post); got != tc.want {
				t.Fatalf("post-wave reviewer count = %d, want exactly %d (dispatches=%+v)", got, tc.want, post)
			}
		})
	}
}

// TestDeterministicChecksAreUnchangedByTheBoundary proves the boundary
// mechanism is scoped strictly to reviewer judgement: every non-reviewer
// build-time dispatch (the coherent-job task workers, the pre-wave
// specialists) is byte-identical for the same phase fixture regardless of
// which verification-boundary decision is recorded for the attempt --
// nothing about the program's own free checks, or the workers that write
// and structure the code those checks run against, ever depends on where
// reviewer judgement lands.
func TestDeterministicChecksAreUnchangedByTheBoundary(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}
	reviewCastes := map[string]bool{"auditor": true, "measurer": true, "chaos": true}
	nonReviewerDispatches := func(dispatches []codexBuildDispatch) []codexBuildDispatch {
		kept := make([]codexBuildDispatch, 0, len(dispatches))
		for _, d := range dispatches {
			if reviewCastes[d.Caste] {
				continue
			}
			kept = append(kept, d)
		}
		return kept
	}

	phaseNoBoundary := boundaryDoubleDispatchPhase(221, "Ordinary feature", "Add a small UI affordance")
	noBoundary := nonReviewerDispatches(testPlannedBuildDispatchesWithJudgement(phaseNoBoundary, state, nil, colony.VerificationDepthStandard, nil, ""))

	phaseCheckStep := boundaryDoubleDispatchPhase(222, "Ordinary feature", "Add a small UI affordance")
	attemptWithVerificationBoundaryRecorded(t, phaseCheckStep.ID, "attempt-deterministic-checkstep-222", verificationBoundaryChoiceCheckStep, "default landing")
	withCheckStep := nonReviewerDispatches(testPlannedBuildDispatchesWithJudgement(phaseCheckStep, state, nil, colony.VerificationDepthStandard, nil, ""))

	phaseBuildEnd := boundaryDoubleDispatchPhase(223, "Ordinary feature", "Add a small UI affordance")
	attemptWithVerificationBoundaryRecorded(t, phaseBuildEnd.ID, "attempt-deterministic-buildend-223", verificationBoundaryChoiceBuildEnd, "release sign-off")
	withBuildEnd := nonReviewerDispatches(testPlannedBuildDispatchesWithJudgement(phaseBuildEnd, state, nil, colony.VerificationDepthStandard, nil, ""))

	sameShape := func(t *testing.T, label string, got []codexBuildDispatch) {
		t.Helper()
		if len(got) != len(noBoundary) {
			t.Fatalf("%s: non-reviewer dispatch count = %d, want %d (matching the no-boundary baseline)", label, len(got), len(noBoundary))
		}
		for i := range noBoundary {
			a, b := noBoundary[i], got[i]
			if a.Caste != b.Caste || a.Stage != b.Stage || a.Task != b.Task || a.ExecutionWave != b.ExecutionWave {
				t.Fatalf("%s: non-reviewer dispatch %d differs from the no-boundary baseline:\n  baseline: %+v\n  got:      %+v", label, i, a, b)
			}
		}
	}
	sameShape(t, "recorded check-step boundary", withCheckStep)
	sameShape(t, "recorded build-end boundary", withBuildEnd)

	// The literal deterministic check commands (build/types/lint/test) live
	// entirely in the continue-side floor and never even see a boundary
	// value or a phase's dispatch list -- resolveCodexVerificationCommands
	// takes only a root path. Lock that independence down explicitly so a
	// future change cannot thread the boundary into check-command
	// resolution without this test naming it.
	if got := resolveCodexVerificationCommands("/tmp"); got != resolveCodexVerificationCommands("/tmp") {
		t.Fatalf("resolveCodexVerificationCommands is not stable across calls for the same root: %+v vs %+v", got, resolveCodexVerificationCommands("/tmp"))
	}
}

// TestCheckStepReviewersGateOnTheRecordedBoundary is Task 2's own proof
// (201-05-PLAN.md): the check-time reviewer dispatch path
// (plannedContinueReviewDispatches, runCodexContinueReview) reads the same
// stored boundary record and dispatches no reviewer of its own when the
// recorded choice is build-end -- consuming the build-end reviewer findings
// already bound to the attempt instead of re-running them.
func TestCheckStepReviewersGateOnTheRecordedBoundary(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	t.Run("recorded build-end boundary: no check-step dispatch, build-end findings surface in the check result", func(t *testing.T) {
		phase := boundaryDoubleDispatchPhase(231, "Password reset", "Let users reset their password via an emailed token")
		attemptRel := attemptWithVerificationBoundaryRecorded(t, phase.ID, "attempt-checkstep-231", verificationBoundaryChoiceBuildEnd, "credentials handling")

		// Bind a real reviewer worker run to the attempt -- the "already
		// happened" finding the check step must read back instead of
		// re-dispatching a second reviewer for the same phase.
		var record buildAttemptRecord
		if err := store.UpdateJSONAtomically(attemptRel, &record, func() error {
			record.WorkerRuns = append(record.WorkerRuns, buildAttemptWorkerRun{
				WorkerName: "Sentinel-9",
				Caste:      "auditor",
				Status:     buildWorkerCompleted,
				Result:     &internalWorkerResult{Summary: "No quality issues found."},
			})
			return nil
		}); err != nil {
			t.Fatalf("bind build-end worker run: %v", err)
		}

		dispatches := plannedContinueReviewDispatches(
			"/tmp", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{},
			&codex.FakeInvoker{}, time.Minute, colony.VerificationDepthStandard, nil, "",
		)
		if len(dispatches) != 0 {
			t.Fatalf("expected no check-step reviewer dispatches with a recorded build-end boundary, got %+v", dispatches)
		}

		report := runCodexContinueReview("/tmp", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, time.Minute, colony.VerificationDepthStandard, false, nil, "")
		found := false
		for _, w := range report.Workers {
			if w.Caste == "auditor" && w.Summary == "No quality issues found." {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected the build-end auditor finding to appear in the check result, got %+v", report.Workers)
		}
		if !report.Passed {
			t.Fatalf("expected the check result to pass on a clean build-end finding, got blockers=%v", report.BlockingIssues)
		}
	})

	t.Run("recorded check-step boundary: check-step dispatches its own reviewer as before", func(t *testing.T) {
		phase := boundaryDoubleDispatchPhase(232, "Password reset", "Let users reset their password via an emailed token")
		attemptWithVerificationBoundaryRecorded(t, phase.ID, "attempt-checkstep-232", verificationBoundaryChoiceCheckStep, "default landing")

		dispatches := plannedContinueReviewDispatches(
			"/tmp", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{},
			&codex.FakeInvoker{}, time.Minute, colony.VerificationDepthStandard, nil, "",
		)
		if len(dispatches) == 0 {
			t.Fatalf("expected the check step to dispatch its own reviewer(s) with a recorded check-step boundary")
		}
	})
}
