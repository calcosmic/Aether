package cmd

import (
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
}
