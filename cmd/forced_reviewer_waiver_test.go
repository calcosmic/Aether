package cmd

import (
	"encoding/json"
	"os"
	"strconv"
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

// TestDecisionAnswerCannotForgeAForcedReviewerWaiver is CR-01's forgery
// proof (194-REVIEW.md): the whole design of D-03 rests on `aether
// decision-answer` being the owner's ONLY way to decline a forced reviewer,
// but the deterministic waiver question is computable from public
// information alone (phaseID + one of five fixed PlainEnglish strings).
// Before this fix, calling the CLI COMMAND directly with a correctly-shaped
// --question and no prior card render silently recorded a waiver anyway.
// This test goes through the actual decisionAnswerCmd command path (not
// just recordDecisionAnswer, which legitimate test setup elsewhere in this
// file calls directly to simulate an already-answered decision) -- the real
// attack surface is the CLI, not the Go function.
func TestDecisionAnswerCannotForgeAForcedReviewerWaiver(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupBuildFlowTest(t)

	question := forcedReviewerWaiverQuestionText(7, "credentials/auth", "logins and passwords")
	rootCmd.SetArgs([]string{"decision-answer", "--question", question, "--answer", "auto", "--phase", "7"})
	defer rootCmd.SetArgs([]string{})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("decision-answer returned error: %v", err)
	}

	phase := waiverFixturePhase(7, "Password reset", "Let users reset their password via an emailed token")
	reviewers := queenForcedContinueReviewers(phase, nil, nil)
	if len(reviewers) != 1 {
		t.Fatalf("a forged decision-answer with no runtime-created pending row must not waive the forced reviewer; reviewers = %+v", reviewers)
	}

	if waived, _ := forcedReviewerWaiver(7, "credentials/auth"); waived {
		t.Fatalf("forcedReviewerWaiver reports the signal waived after a forged decision-answer call")
	}
}

// TestDecisionAnswerResolvesARuntimeCreatedWaiverRow is CR-01's positive
// proof: the real owner path -- the check-in card renders a live forced
// reviewer (which writes the pending row via
// ensureForcedReviewerWaiverPendingDecision), then `aether decision-answer`
// resolves that SAME row -- still works end to end through the actual CLI
// command.
func TestDecisionAnswerResolvesARuntimeCreatedWaiverRow(t *testing.T) {
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
	// Rendering the card is what writes the pending, unresolved row.
	if _, visual := renderCeremonyTeamCheckin("build", manifestMap, dispatches); !strings.Contains(visual, "To decline, run:") {
		t.Fatalf("expected the card to show a live decline command before answering it; visual:\n%s", visual)
	}

	question := forcedReviewerWaiverQuestionText(1, "credentials/auth", "logins and passwords")
	rootCmd.SetArgs([]string{"decision-answer", "--question", question, "--answer", "already checked by hand", "--phase", "1"})
	defer rootCmd.SetArgs([]string{})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("decision-answer returned error: %v", err)
	}

	waived, reason := forcedReviewerWaiver(1, "credentials/auth")
	if !waived {
		t.Fatalf("expected the signal to be waived after the owner path recorded an answer")
	}
	if reason != "already checked by hand" {
		t.Fatalf("reason = %q, want the owner's recorded answer", reason)
	}
}

// spawnLogArgsForPhase is the real `aether spawn-log` invocation the wrapper
// triplet (.claude/commands/ant/build.md and its two mirrors) now sends
// before every worker, with --phase added by this fix. It is what actually
// closes the forced-reviewer decline window (closeForcedReviewerWaiverWindowForPhase,
// cmd/forced_reviewer_waiver.go), called from spawn-log's own RunE
// (cmd/spawn.go).
func spawnLogArgsForPhase(phase int) []string {
	return []string{
		"spawn-log",
		"--parent", "Queen",
		"--caste", "builder",
		"--name", "Mason-1",
		"--task", "build the password reset flow",
		"--depth", "1",
		"--phase", strconv.Itoa(phase),
	}
}

// TestForcedReviewerDeclineWindowClosesWhenDispatchBegins is the CR-01
// residual's negative proof (194-REVIEW.md iteration 2): the check-in card
// renders a live forced reviewer (writing the pending row), the owner does
// NOT decline it (the default, recommended "proceed" path -- no
// decision-answer call at all), and dispatch genuinely begins (the real
// spawn-log CLI command, exactly as the wrapper now calls it, with
// --phase). A decision-answer call made AFTER that point -- simulating a
// worker's own Bash tool, a stray script, or a prompt-injected instruction
// running the exact command the card legitimately displayed -- must be
// refused: the reviewer the owner never declined must still be forced.
func TestForcedReviewerDeclineWindowClosesWhenDispatchBegins(t *testing.T) {
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
	if _, visual := renderCeremonyTeamCheckin("build", manifestMap, dispatches); !strings.Contains(visual, "To decline, run:") {
		t.Fatalf("expected the card to show a live decline command; visual:\n%s", visual)
	}

	// Dispatch genuinely begins -- the owner never ran decision-answer.
	rootCmd.SetArgs(spawnLogArgsForPhase(1))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-log returned error: %v", err)
	}
	rootCmd.SetArgs([]string{})

	// A later decision-answer call with the card's exact question -- the
	// forgery/misuse this test exists for -- must now be refused.
	question := forcedReviewerWaiverQuestionText(1, "credentials/auth", "logins and passwords")
	rootCmd.SetArgs([]string{"decision-answer", "--question", question, "--answer", "auto", "--phase", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("decision-answer returned error: %v", err)
	}
	rootCmd.SetArgs([]string{})

	if waived, reason := forcedReviewerWaiver(1, "credentials/auth"); waived {
		t.Fatalf("a decision-answer call issued AFTER dispatch began must never waive the reviewer; reason = %q", reason)
	}
	reviewers := queenForcedContinueReviewers(phase, nil, nil)
	if len(reviewers) != 1 {
		t.Fatalf("reviewer the owner never declined must still be forced after dispatch began; reviewers = %+v", reviewers)
	}
}

// TestOwnerDeclineBeforeDispatchStaysHonoredOnceDispatchBegins is the
// positive companion: the owner's REAL decline path (running
// decision-answer at the check-in pause, before any worker is dispatched)
// must keep working exactly as before, even after spawn-log later closes
// the window for that phase.
func TestOwnerDeclineBeforeDispatchStaysHonoredOnceDispatchBegins(t *testing.T) {
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
	renderCeremonyTeamCheckin("build", manifestMap, dispatches)

	// The owner declines BEFORE dispatch begins -- the real path.
	question := forcedReviewerWaiverQuestionText(1, "credentials/auth", "logins and passwords")
	rootCmd.SetArgs([]string{"decision-answer", "--question", question, "--answer", "already checked by hand", "--phase", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("decision-answer returned error: %v", err)
	}
	rootCmd.SetArgs([]string{})

	// Dispatch now begins.
	rootCmd.SetArgs(spawnLogArgsForPhase(1))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-log returned error: %v", err)
	}
	rootCmd.SetArgs([]string{})

	waived, reason := forcedReviewerWaiver(1, "credentials/auth")
	if !waived {
		t.Fatalf("the owner's genuine decline, made before dispatch began, must still be honored after dispatch begins")
	}
	if reason != "already checked by hand" {
		t.Fatalf("reason = %q, want the owner's recorded answer", reason)
	}
	if reviewers := queenForcedContinueReviewers(phase, nil, nil); len(reviewers) != 0 {
		t.Fatalf("owner's declined reviewer must not be forced; reviewers = %+v", reviewers)
	}
}

// TestSpawnLogWithNoCardRenderedIsANoOpAndReviewerStaysForced covers the
// --no-checkin / autopilot lane: no check-in card is ever rendered, so no
// pending forced-reviewer row is ever created. spawn-log's window-closing
// call (closeForcedReviewerWaiverWindowForPhase) must be a silent no-op in
// that case -- no error, no stray row created -- and the forced reviewer
// must still be live at continue time, exactly as
// TestAutopilotNeverWaives already requires for the no-decision-answer
// path.
func TestSpawnLogWithNoCardRenderedIsANoOpAndReviewerStaysForced(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupBuildFlowTest(t)

	phase := waiverFixturePhase(12, "Password reset", "Let users reset their password via an emailed token")

	rootCmd.SetArgs(spawnLogArgsForPhase(12))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-log returned error with no card ever rendered: %v", err)
	}
	rootCmd.SetArgs([]string{})

	pending := loadPendingDecisionFile()
	for _, d := range pending.Decisions {
		if d.Source == "forced-reviewer-waiver" {
			t.Fatalf("spawn-log with no card render must not create a forced-reviewer-waiver row; got %+v", d)
		}
	}

	reviewers := queenForcedContinueReviewers(phase, nil, nil)
	if len(reviewers) != 1 {
		t.Fatalf("forced reviewer must still be live when no card was ever rendered; reviewers = %+v", reviewers)
	}
}
