package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// waiverFixturePhase mirrors judgementPhase (cmd/queen_judgement_test.go) but
// gives the caller control over the phase ID, since the whole point of the
// waiver's scoping rule (D-03) is that phase ID is part of what the question
// text is matched against.
func waiverFixturePhase(id int, name, description string) colony.Phase {
	return colony.Phase{ID: id, Name: name, Description: description, Mode: colony.PhaseModePrototype}
}

// TestOnlyTheOwnerCanWaiveAForcedReviewer proves the negative: with NO
// recorded decision-answer, nothing else can make a forced reviewer go
// away. Four different attempts to drop it -- a lighter review depth, a
// standard review depth, an explicit caste proposal that simply omits it,
// and an explicit proposal that names other castes instead -- all still
// carry the forced reviewer (T-194-11).
func TestOnlyTheOwnerCanWaiveAForcedReviewer(t *testing.T) {
	phase := waiverFixturePhase(7, "Password reset", "Let users reset their password via an emailed token")

	cases := []struct {
		name     string
		depth    colony.VerificationDepth
		proposed []string
		reason   string
	}{
		{name: "light depth, no proposal", depth: colony.VerificationDepthLight},
		{name: "standard depth, no proposal", depth: colony.VerificationDepthStandard},
		{name: "explicit proposal omitting the reviewer", depth: colony.VerificationDepthStandard, proposed: []string{"builder"}, reason: "just the form work"},
		{name: "explicit proposal naming other castes", depth: colony.VerificationDepthStandard, proposed: []string{"builder", "measurer"}, reason: "form work plus a latency check"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dispatches := queenContinueDispatchesWithJudgement(phase, tc.depth, tc.proposed, tc.reason, nil, nil)
			if !queenContinueHasCaste(dispatches, "gatekeeper") {
				t.Fatalf("%s: forced reviewer dropped with no recorded waiver; dispatches = %+v", tc.name, dispatches)
			}
		})
	}
}

// TestWaiverCoversOneSignalOnOnePhase proves the three non-carry cases D-03
// requires: a waiver recorded for one signal on one phase does not waive a
// different signal on the same phase, and does not waive the same signal on
// a different phase.
func TestWaiverCoversOneSignalOnOnePhase(t *testing.T) {
	setupBuildFlowTest(t)

	if _, err := recordDecisionAnswer(
		forcedReviewerWaiverQuestionText(7, "credentials/auth", "logins and passwords"),
		"the login flow here is trivial, I checked it myself", 7, "owner",
	); err != nil {
		t.Fatalf("recordDecisionAnswer: %v", err)
	}

	t.Run("waived signal is gone on the phase it was waived for", func(t *testing.T) {
		phase := waiverFixturePhase(7, "Password reset", "Let users reset their password via an emailed token")
		reviewers := queenForcedContinueReviewers(phase, nil, nil)
		if len(reviewers) != 0 {
			t.Fatalf("credentials signal on phase 7 should be waived; got %+v", reviewers)
		}
	})

	t.Run("a different signal on the same phase is not waived", func(t *testing.T) {
		phase := waiverFixturePhase(7, "Refund after login", "Let users log in and request a refund for a recent charge")
		reviewers := queenForcedContinueReviewers(phase, nil, nil)
		if len(reviewers) != 1 {
			t.Fatalf("payments signal on phase 7 must still force a reviewer; got %+v", reviewers)
		}
		if !strings.Contains(reviewers[0].Reason, "money") || strings.Contains(reviewers[0].Reason, "logins") {
			t.Fatalf("phase 7's live reviewer must be forced by payments alone, not credentials; reason = %q", reviewers[0].Reason)
		}
	})

	t.Run("the same signal on a different phase is not waived", func(t *testing.T) {
		phase := waiverFixturePhase(8, "Password reset", "Let users reset their password via an emailed token")
		reviewers := queenForcedContinueReviewers(phase, nil, nil)
		if len(reviewers) != 1 {
			t.Fatalf("credentials signal on phase 8 must still force a reviewer; got %+v", reviewers)
		}
	})
}

// TestWaivedSignalStaysWaivedWhenTheFilesRedetectIt: a signal waived from the
// plan's own wording must stay waived even when the builder's changed files
// independently re-detect the same signal (D-03) -- the file detector can
// only ADD a reviewer the plan-wording detector missed, it can never revive
// one the owner has already declined.
func TestWaivedSignalStaysWaivedWhenTheFilesRedetectIt(t *testing.T) {
	setupBuildFlowTest(t)

	phase := waiverFixturePhase(9, "Password reset", "Let users reset their password via an emailed token")

	if _, err := recordDecisionAnswer(
		forcedReviewerWaiverQuestionText(9, "credentials/auth", "logins and passwords"),
		"already reviewed this by hand", 9, "owner",
	); err != nil {
		t.Fatalf("recordDecisionAnswer: %v", err)
	}

	reviewers := queenForcedContinueReviewers(phase, nil, []string{"internal/auth/session.go"})
	if len(reviewers) != 0 {
		t.Fatalf("a signal waived from plan wording must stay waived when changed files re-detect it; got %+v", reviewers)
	}
}

// TestAutopilotNeverWaives asserts the unattended path autopilot reaches
// (queenContinueDispatchesWithJudgement with no proposal, the same function
// TestOnlyTheOwnerCanWaiveAForcedReviewer's first two cases exercise) still
// carries the forced reviewer with no recorded answer, and that the
// autopilot source itself never calls the one function that could record
// one (recordDecisionAnswer) -- autopilot has no code path that could ever
// reach a waiver, not merely a code path that happens not to today.
func TestAutopilotNeverWaives(t *testing.T) {
	phase := waiverFixturePhase(11, "Password reset", "Let users reset their password via an emailed token")
	dispatches := queenContinueDispatchesWithJudgement(phase, colony.VerificationDepthStandard, nil, "", nil, nil)
	if !queenContinueHasCaste(dispatches, "gatekeeper") {
		t.Fatalf("autopilot's no-proposal path dropped the forced reviewer with no recorded waiver; dispatches = %+v", dispatches)
	}

	src, err := os.ReadFile("autopilot.go")
	if err != nil {
		t.Fatalf("reading autopilot.go: %v", err)
	}
	if strings.Contains(string(src), "recordDecisionAnswer") {
		t.Fatalf("autopilot.go must never call recordDecisionAnswer -- autopilot can never record a waiver")
	}
}

// TestTeamCheckinDoesNotMutateWithAWaiverPresent extends the existing
// non-mutation contract for the check-in command (TestTeamCheckinDoesNotMutate,
// cmd/ceremony_team_checkin_test.go) to the waiver-present case: an
// inspection command that mutates is a named corollary of this repo's
// Definition of Done, and a waiver recorded before the card renders must not
// change that.
func TestTeamCheckinDoesNotMutateWithAWaiverPresent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]

	phase := checkinFixturePhase(
		"Password reset",
		"Let users reset their password via an emailed token",
		colony.PhaseModePrototype,
	)
	setUpCheckinFixtureColony(t, dataDir, phase)

	manifestMap, dispatches := manifestMapFromBuild(t, root, 1)
	forcedRecords := forcedReviewerRecordsFromManifest(manifestMap)
	if len(forcedRecords) == 0 {
		t.Fatalf("expected the password-reset fixture to force a reviewer before the waiver test can proceed")
	}

	if _, err := recordDecisionAnswer(
		forcedReviewerWaiverQuestionText(1, "credentials/auth", "logins and passwords"),
		"already reviewed by hand", 1, "owner",
	); err != nil {
		t.Fatalf("recordDecisionAnswer: %v", err)
	}

	before, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState before: %v", err)
	}

	result, visual := renderCeremonyTeamCheckin("build", manifestMap, dispatches)
	waived, _ := result["waived"].(map[string]interface{})
	if len(waived) == 0 {
		t.Fatalf("expected a waived signal on the card once the waiver is recorded; visual:\n%s", visual)
	}
	if !strings.Contains(visual, "Declined by owner") {
		t.Fatalf("card should show the waived reviewer as declined by owner; visual:\n%s", visual)
	}

	after, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState after: %v", err)
	}
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	if string(beforeJSON) != string(afterJSON) {
		t.Fatalf("rendering the check-in card with a waiver present mutated colony state:\nbefore: %s\nafter:  %s", beforeJSON, afterJSON)
	}
}
