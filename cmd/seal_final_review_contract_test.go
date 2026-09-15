package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestSealFinalReviewBriefCarriesHandoffSchema locks the 2026-09-14 field
// report's first defect: a seal reviewer's brief must state the exact
// handoff result-shape the finalizer enforces (mergeExternalSealReviewResults
// -> mergeExternalContinueResults -> ValidateWorkerHandoff), using the same
// single-source constants (codex.HandoffFieldsSummary,
// codex.HandoffOpenDecisionsGuidance) the continue and build lanes already
// append at their own external dispatch call sites
// (continueExternalBriefWithHandoffSchema, composeBuildManifestBrief). A
// reviewer that follows its brief exactly but never sees this sentence has
// no way to know its handoff will be rejected on the first attempt.
//
// This is a result-shape instruction, not a shell-command instruction, so it
// applies to every seal review caste including the security reviewer
// (gatekeeper) and quality reviewer (auditor) that CLAUDE.md's "Biological
// Runtime" section documents as holding no shell — that exception
// (TestReviewSpecsDoNotInstructBashlessCastes) covers being told to run a
// command, not being told the shape their own JSON result must take.
func TestSealFinalReviewBriefCarriesHandoffSchema(t *testing.T) {
	root, phase, state := seedHandoffColonyForBriefTests(t, "seal-handoff-schema-contract")

	invoker := &codex.FakeInvoker{}
	dispatches := plannedSealFinalReviewDispatches(root, state, phase, invoker, 0, colony.VerificationDepthHeavy)
	if len(dispatches) == 0 {
		t.Fatal("fixture broken: expected non-empty seal review dispatches at heavy depth")
	}

	for _, dispatch := range dispatches {
		t.Run(dispatch.Caste, func(t *testing.T) {
			brief := dispatch.TaskBrief

			fieldsCount := strings.Count(brief, codex.HandoffFieldsSummary)
			if fieldsCount != 1 {
				t.Fatalf("caste %q brief contains codex.HandoffFieldsSummary %d times, want exactly 1:\n%s", dispatch.Caste, fieldsCount, brief)
			}

			guidanceCount := strings.Count(brief, codex.HandoffOpenDecisionsGuidance)
			if guidanceCount != 1 {
				t.Fatalf("caste %q brief contains codex.HandoffOpenDecisionsGuidance %d times, want exactly 1:\n%s", dispatch.Caste, guidanceCount, brief)
			}
		})
	}
}

// TestSealFinalReviewBriefHandoffSchemaCoversEveryQueenSelectedCaste is a
// second angle on the same guarantee, driven directly off
// queenSealReviewSpecs (the same selector plannedSealFinalReviewDispatches
// calls) rather than the dispatch list, so a future refactor of
// plannedSealFinalReviewDispatches that stops calling queenSealReviewSpecs
// cannot silently narrow which castes this test actually covers.
func TestSealFinalReviewBriefHandoffSchemaCoversEveryQueenSelectedCaste(t *testing.T) {
	root, phase, state := seedHandoffColonyForBriefTests(t, "seal-handoff-schema-per-caste")

	specs := queenSealReviewSpecs(state, phase, colony.VerificationDepthHeavy)
	if len(specs) == 0 {
		t.Fatal("fixture broken: expected non-empty seal review specs at heavy depth")
	}

	for _, spec := range specs {
		t.Run(spec.Caste, func(t *testing.T) {
			rendered := renderSealFinalReviewBrief(root, state, phase, spec)
			brief := sealExternalBriefWithHandoffSchema(rendered)

			if strings.Count(brief, codex.HandoffFieldsSummary) != 1 {
				t.Fatalf("caste %q brief missing exactly-once codex.HandoffFieldsSummary:\n%s", spec.Caste, brief)
			}
			if strings.Count(brief, codex.HandoffOpenDecisionsGuidance) != 1 {
				t.Fatalf("caste %q brief missing exactly-once codex.HandoffOpenDecisionsGuidance:\n%s", spec.Caste, brief)
			}
			// Wrapping must never happen twice for the same brief -- appending
			// at renderSealFinalReviewBrief's own call site AND again at the
			// dispatch site would double the sentence for a native-lane worker
			// that already receives the contract through the response-contract
			// channel (mirrors D-06's reasoning for continue).
			if strings.Count(sealExternalBriefWithHandoffSchema(brief), codex.HandoffFieldsSummary) != 2 {
				t.Fatalf("sealExternalBriefWithHandoffSchema is not idempotent-safe to detect double-wrapping for caste %q", spec.Caste)
			}
		})
	}
}
