package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestRiskSignalTableHasExactlyFiveEntries pins D-01: the named-risk
// vocabulary is exactly the five signals the owner ratified, and every one
// of them maps to one of the two reviewer castes this program can force
// (gatekeeper, auditor). Adding, removing or misrouting an entry must fail
// this test — see queenRiskSignalTable's own doc comment.
func TestRiskSignalTableHasExactlyFiveEntries(t *testing.T) {
	if len(queenRiskSignalTable) != 5 {
		t.Fatalf("queenRiskSignalTable has %d entries, want exactly 5: %+v", len(queenRiskSignalTable), queenRiskSignalTable)
	}
	for _, signal := range queenRiskSignalTable {
		if signal.Caste != "gatekeeper" && signal.Caste != "auditor" {
			t.Errorf("signal %q forces caste %q, want gatekeeper or auditor", signal.Name, signal.Caste)
		}
		if strings.TrimSpace(signal.PlainEnglish) == "" {
			t.Errorf("signal %q has no plain-English description", signal.Name)
		}
		if len(signal.Phrases) == 0 {
			t.Errorf("signal %q has no phrases", signal.Name)
		}
	}
}

// TestForcedReviewerCrossesTheBuildContinueBoundary is the tracer's proof:
// a password-reset phase records ONE forced security reviewer on the build
// manifest with a plain-English reason naming the matched phrase, and that
// SAME record — not a fresh re-derivation — is what continue dispatches
// (D-05: one derivation, one boundary). The build never dispatches it
// itself.
func TestForcedReviewerCrossesTheBuildContinueBoundary(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]

	goal := "Password reset"
	taskID := "1.1"
	phase := colony.Phase{
		ID:          1,
		Name:        "Password reset",
		Description: "Let users reset their password via an emailed token",
		Mode:        colony.PhaseModePrototype,
		Status:      colony.PhaseReady,
		Tasks:       []colony.Task{{ID: &taskID, Goal: "Add the reset-password form and endpoint", Status: colony.TaskPending}},
	}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan:         colony.Plan{Phases: []colony.Phase{phase}},
	})

	// --- Build: the record, never the dispatch ---
	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	if len(manifest.ForcedReviewers) != 1 {
		t.Fatalf("build manifest forced_reviewers = %+v, want exactly one entry", manifest.ForcedReviewers)
	}
	record := manifest.ForcedReviewers[0]
	if record.Caste != "gatekeeper" {
		t.Fatalf("forced reviewer caste = %q, want gatekeeper", record.Caste)
	}
	if !strings.Contains(record.Reason, "password reset") {
		t.Fatalf("forced reviewer reason = %q, want it to quote %q", record.Reason, "password reset")
	}

	// --- Continue: dispatches exactly what build recorded ---
	continueManifest := codexContinueManifest{
		Present: true,
		Data:    codexBuildManifest{ForcedReviewers: manifest.ForcedReviewers},
	}
	dispatches := plannedContinueReviewDispatches(
		root, phase, continueManifest, codexContinueVerificationReport{}, codexContinueAssessment{},
		&codex.FakeInvoker{}, time.Minute, colony.VerificationDepthLight, nil, "",
	)
	if len(dispatches) != 1 {
		t.Fatalf("continue dispatches = %+v, want exactly one", dispatches)
	}
	if dispatches[0].Caste != "gatekeeper" {
		t.Fatalf("continue dispatch caste = %q, want gatekeeper", dispatches[0].Caste)
	}
	if !strings.Contains(dispatches[0].TaskBrief, "password reset") {
		t.Fatalf("continue dispatch brief does not carry the forced-reviewer reason (missing %q): %s", "password reset", dispatches[0].TaskBrief)
	}
}
