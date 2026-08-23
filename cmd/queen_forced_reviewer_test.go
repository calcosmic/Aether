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

// forcedReasonForCaste returns the Rationale of the dispatch with the given
// caste, and whether that dispatch is present at all. A forced reviewer's
// Rationale always starts with "this touches" (forcedReviewerReason) — that
// prefix is what distinguishes "the named-risk table forced this" from "the
// pre-existing keyword/mode scoring engine happened to also select this
// caste", which the SAME caste name cannot distinguish on its own (e.g.
// production mode alone already draws an auditor via the untouched legacy
// floor this plan deliberately leaves in place — see 194-01-PLAN.md "Known
// interim state").
func forcedReasonForCaste(dispatches []CasteDispatch, caste string) (string, bool) {
	for _, dispatch := range dispatches {
		if dispatch.Caste == caste {
			return dispatch.Rationale, true
		}
	}
	return "", false
}

func anyDispatchIsForced(dispatches []CasteDispatch) bool {
	for _, dispatch := range dispatches {
		if strings.HasPrefix(dispatch.Rationale, "this touches") {
			return true
		}
	}
	return false
}

// TestReviewerForcedOnlyByNamedRisk is the table the roadmap names (D-01,
// D-04), asserted on the CONTINUE dispatch list queenContinueDispatchesWithJudgement
// actually returns — never on an intermediate decision record, which is this
// repo's signature failure (TestQueenChoiceReachesTheDispatchList exists
// because of it).
func TestReviewerForcedOnlyByNamedRisk(t *testing.T) {
	cases := []struct {
		name        string
		phaseName   string
		description string
		mode        colony.PhaseMode
		wantCaste   string // "" means no signal-forced reviewer
		wantPhrase  string
	}{
		{
			name:        "CSV export forces nothing",
			phaseName:   "Add CSV export",
			description: "Let users download their table as a CSV file",
			mode:        colony.PhaseModeProduction,
		},
		{
			name:        "password reset forces gatekeeper",
			phaseName:   "Password reset",
			description: "Let users reset their password by email",
			mode:        colony.PhaseModePrototype,
			wantCaste:   "gatekeeper",
			wantPhrase:  "password reset",
		},
		{
			name:        "refund button forces gatekeeper",
			phaseName:   "Refund button",
			description: "Let support staff refund a charge",
			mode:        colony.PhaseModePrototype,
			wantCaste:   "gatekeeper",
			wantPhrase:  "refund",
		},
		{
			name:        "delete stale accounts forces auditor",
			phaseName:   "Delete stale accounts",
			description: "Remove accounts that have been dormant for a year",
			mode:        colony.PhaseModePrototype,
			wantCaste:   "auditor",
			wantPhrase:  "delete stale",
		},
		{
			name:        "adding a migration column forces auditor",
			phaseName:   "Add a column to the orders table",
			description: "Write the migration",
			mode:        colony.PhaseModePrototype,
			wantCaste:   "auditor",
			wantPhrase:  "add a column",
		},
		{
			name:        "production phase with no signal wording forces nothing",
			phaseName:   "Improve dashboard performance",
			description: "Speed up the analytics dashboard for large accounts",
			mode:        colony.PhaseModeProduction,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			phase := judgementPhase(tc.phaseName, tc.description, tc.mode)
			dispatches := queenContinueDispatchesWithJudgement(phase, colony.VerificationDepthLight, nil, "", nil)

			if tc.wantCaste == "" {
				if anyDispatchIsForced(dispatches) {
					t.Fatalf("expected no signal-forced reviewer, dispatches = %+v", dispatches)
				}
				return
			}
			reason, ok := forcedReasonForCaste(dispatches, tc.wantCaste)
			if !ok || !strings.HasPrefix(reason, "this touches") {
				t.Fatalf("expected %s forced by a named signal, dispatches = %+v", tc.wantCaste, dispatches)
			}
			if !strings.Contains(reason, tc.wantPhrase) {
				t.Fatalf("forced reason %q does not quote the matched phrase %q", reason, tc.wantPhrase)
			}
		})
	}
}

// TestTwoSignalsOneCasteCollapseToOneDispatch is D-04's collapse rule: a
// phase mentioning both a login and a refund (auth + payments — both
// gatekeeper signals) must produce exactly ONE gatekeeper dispatch, whose
// reason names both signals, not two dispatches of the same caste.
func TestTwoSignalsOneCasteCollapseToOneDispatch(t *testing.T) {
	phase := judgementPhase(
		"Refund after login",
		"Let users log in and then request a refund for a recent charge",
		colony.PhaseModePrototype,
	)
	dispatches := queenContinueDispatchesWithJudgement(phase, colony.VerificationDepthLight, nil, "", nil)

	gatekeeperCount := 0
	var reason string
	for _, dispatch := range dispatches {
		if dispatch.Caste == "gatekeeper" {
			gatekeeperCount++
			reason = dispatch.Rationale
		}
	}
	if gatekeeperCount != 1 {
		t.Fatalf("gatekeeper dispatched %d times, want exactly 1: %+v", gatekeeperCount, dispatches)
	}
	if !strings.HasPrefix(reason, "this touches") {
		t.Fatalf("gatekeeper dispatch was not signal-forced: rationale = %q", reason)
	}
	for _, want := range []string{"logins and passwords", "money", "log in", "refund"} {
		if !strings.Contains(reason, want) {
			t.Errorf("collapsed reason %q missing %q — both signals must be named", reason, want)
		}
	}
}

// TestEmptyPhaseForcesNoReviewer is the empty-input probe (TEAM-01/TEAM-03): a
// phase with no name, no description and no tasks forces no reviewer AND
// requires exactly the one build caste. Plan 194-01 could only assert the
// first half ("the 'requires exactly the one build caste' half of that probe
// belongs to plan 194-02, where the floor actually shrinks"); this plan
// completes it now that queenBuildSafetyRequiredCastes has shrunk.
func TestEmptyPhaseForcesNoReviewer(t *testing.T) {
	phase := colony.Phase{}

	if reviewers := queenForcedReviewersForPhase(phase); len(reviewers) != 0 {
		t.Fatalf("empty phase forced reviewers = %+v, want none", reviewers)
	}

	dispatches := queenContinueDispatchesWithJudgement(phase, colony.VerificationDepthLight, nil, "", nil)
	if anyDispatchIsForced(dispatches) {
		t.Fatalf("empty phase produced a signal-forced dispatch: %+v", dispatches)
	}

	required := queenBuildSafetyRequiredCastes(phase)
	if len(required) != 1 || required[0] != "builder" {
		t.Fatalf("empty phase required build castes = %+v, want exactly [builder]", required)
	}
}

// TestSignalMatchingIsWordBounded pins D-02's encoding probe: matching is
// case-insensitive and anchored at word boundaries, so "token" inside
// "tokenizer" or "token bucket" never fires (the known false alarm this
// signal table deliberately excludes the lone word "token" for), while a
// genuine phrase like "api key" still fires.
func TestSignalMatchingIsWordBounded(t *testing.T) {
	falseAlarm := judgementPhase(
		"Add rate limiting",
		"Add a token bucket rate limiter and run the tokenizer over the input",
		colony.PhaseModePrototype,
	)
	if reviewers := queenForcedReviewersForPhase(falseAlarm); len(reviewers) != 0 {
		t.Fatalf("token bucket / tokenizer wording forced a reviewer: %+v", reviewers)
	}

	genuineHit := judgementPhase(
		"Rotate credentials",
		"Rotate the api key used by the external integration",
		colony.PhaseModePrototype,
	)
	reviewers := queenForcedReviewersForPhase(genuineHit)
	if len(reviewers) != 1 || reviewers[0].Caste != "gatekeeper" {
		t.Fatalf("'rotate the api key' should force gatekeeper, got %+v", reviewers)
	}
}
