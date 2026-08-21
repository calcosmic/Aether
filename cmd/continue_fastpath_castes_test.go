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

	proposed := plannedContinueReviewDispatches(root, phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, &codex.FakeInvoker{}, time.Minute, colony.VerificationDepthStandard, []string{"measurer"}, "perf phrasing without perf keywords")
	if !containsDispatchCaste(proposed, "measurer") {
		t.Fatalf("the fast continue path ignored the Queen's --castes proposal; only the heavy plan-only path honoured it. got %+v", proposed)
	}
}

// TestContinueDoesNotSummonKeeperOnIncidentalWords: "standard", "document"
// and "pattern" are everyday phase vocabulary — two incidental hits cleared
// the continue threshold and bought a knowledge-preservation reviewer for
// phases that had nothing to preserve.
func TestContinueDoesNotSummonKeeperOnIncidentalWords(t *testing.T) {
	incidental := probeGatingPhase("Tidy the export module", "Document the standard pattern used by the export code", colony.PhaseModeMaintenance)
	if HasCaste(queenOrchestrate(incidental, "continue", colony.ColonyState{}), "keeper") {
		t.Fatalf("incidental standard/document/pattern wording must not buy a Keeper run")
	}
	preservation := probeGatingPhase("Capture conventions", "Preserve knowledge and conventions for future workers", colony.PhaseModeMaintenance)
	if !HasCaste(queenOrchestrate(preservation, "continue", colony.ColonyState{}), "keeper") {
		t.Fatalf("a genuine knowledge-preservation phase should still select the Keeper")
	}
}
