package cmd

import (
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func containsDispatchCaste(dispatches []codex.WorkerDispatch, caste string) bool {
	for _, dispatch := range dispatches {
		if dispatch.Caste == caste {
			return true
		}
	}
	return false
}

// TestContinueFastPathHonoursCasteProposal pins the wiring the wrapper
// documents but the runtime dropped: `aether continue --castes ...` flowed
// into codexContinueOptions.QueenCastes, but the default (fast) continue path
// called the nil-proposal spec variant, so the Queen's judgement only ever
// took effect on the heavy plan-only path. On the path users actually run,
// the deterministic keyword engine was unchallenged.
func TestContinueFastPathHonoursCasteProposal(t *testing.T) {
	saveGlobalsCmd(t)
	phase := probeGatingPhase("Write the README", "Documentation for installation and usage", colony.PhaseModeMaintenance)
	root := t.TempDir()

	base := plannedContinueReviewDispatches(root, phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, &codex.FakeInvoker{}, time.Minute, colony.VerificationDepthStandard, nil, "")
	if containsDispatchCaste(base, "measurer") {
		t.Fatalf("fixture broken: measurer must not be in the unproposed baseline, got %+v", base)
	}

	// D-08: a proposal needs a reason PER WORKER, not just a team-level
	// string, or the worker is refused by name rather than sent unexplained.
	proposed := plannedContinueReviewDispatches(root, phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, &codex.FakeInvoker{}, time.Minute, colony.VerificationDepthStandard, []string{"measurer"}, "perf phrasing without perf keywords", map[string]string{"measurer": "perf phrasing without perf keywords"})
	if !containsDispatchCaste(proposed, "measurer") {
		t.Fatalf("the fast continue path ignored the Queen's --castes proposal; only the heavy plan-only path honoured it. got %+v", proposed)
	}

	// D-08 (this plan, 194-06): the SAME proposal on the SAME fast path, but
	// with no per-worker reason supplied, must be refused BY NAME -- not
	// silently sent, and not silently ignored. The original bug this test
	// pins was a proposal that only took effect on the heavy plan-only path;
	// the same asymmetry would now be possible for the reason/refusal
	// machinery too if the fast path dropped reasons on the floor while the
	// plan-only path honoured them.
	refused := plannedContinueReviewDispatches(root, phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, &codex.FakeInvoker{}, time.Minute, colony.VerificationDepthStandard, []string{"measurer"}, "perf phrasing without perf keywords")
	if containsDispatchCaste(refused, "measurer") {
		t.Fatalf("the fast continue path sent measurer with no per-worker reason; want it refused by name (D-08). got %+v", refused)
	}
}

// TestContinueDoesNotSummonKeeperOnIncidentalWords: "standard", "document"
// and "pattern" are everyday phase vocabulary — two incidental hits cleared
// the continue threshold and bought a knowledge-preservation reviewer for
// phases that had nothing to preserve.
// TestContinueDoesNotSummonKeeperOnIncidentalWords exercises the scoring
// registry directly (queenCandidateDispatches). Plan 194-05 (D-11) removed
// the no-proposal keyword-scoring fallback from queenOrchestrate's continue
// path entirely, so neither phase below would select ANY optional
// specialist through that entry point any more -- the claim this test
// protects (the registry can tell incidental wording apart from genuine
// preservation intent) still lives in the scoring function itself.
func TestContinueDoesNotSummonKeeperOnIncidentalWords(t *testing.T) {
	incidental := probeGatingPhase("Tidy the export module", "Document the standard pattern used by the export code", colony.PhaseModeMaintenance)
	if HasCaste(queenCandidateDispatches(incidental, "continue", colony.ColonyState{}), "keeper") {
		t.Fatalf("incidental standard/document/pattern wording must not buy a Keeper run")
	}
	preservation := probeGatingPhase("Capture conventions", "Preserve knowledge and conventions for future workers", colony.PhaseModeMaintenance)
	if !HasCaste(queenCandidateDispatches(preservation, "continue", colony.ColonyState{}), "keeper") {
		t.Fatalf("a genuine knowledge-preservation phase should still select the Keeper")
	}
}
