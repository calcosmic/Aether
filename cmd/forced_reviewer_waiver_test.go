package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// waiverFixturePhase mirrors judgementPhase (cmd/queen_judgement_test.go) but
// gives the caller control over the phase ID, since the whole point of the
// waiver's scoping rule (D-03) is that phase ID is part of what the question
// text is matched against.
func waiverFixturePhase(id int, name, description string) colony.Phase {
	return colony.Phase{ID: id, Name: name, Description: description, Mode: colony.PhaseModePrototype}
}

func waiverCapabilityFromCheckin(t *testing.T, result map[string]interface{}, signal string) string {
	t.Helper()
	commands, ok := result["waive_commands"].(map[string]string)
	if !ok {
		t.Fatalf("waive_commands has unexpected type: %#v", result["waive_commands"])
	}
	command := commands[signal]
	const marker = "--waiver-capability '"
	start := strings.LastIndex(command, marker)
	if start < 0 {
		t.Fatalf("decline command has no waiver capability: %q", command)
	}
	capability := command[start+len(marker):]
	capability = strings.TrimSuffix(capability, "'")
	if capability == "" {
		t.Fatalf("decline command has an empty waiver capability: %q", command)
	}
	return capability
}

func recordResolvedForcedReviewerWaiverForTest(t *testing.T, phaseID int, signal, plainEnglish, reason string) {
	t.Helper()
	file := loadPendingDecisionFile()
	question := forcedReviewerWaiverQuestionText(phaseID, signal, plainEnglish)
	file.Decisions = append(file.Decisions, PendingDecision{
		ID:                     fmt.Sprintf("waiver-%d-%s", phaseID, signal),
		Type:                   clarificationDecisionType,
		Description:            formatClarificationDescription(question, nil),
		Phase:                  &phaseID,
		Source:                 "forced-reviewer-waiver",
		Resolution:             reason,
		Resolved:               true,
		CreatedAt:              time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
		ResolvedAt:             time.Now().UTC().Format(time.RFC3339),
		AttemptID:              "attempt-test-owner-card",
		WaiverCapabilitySHA256: "test-capability-digest",
	})
	if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
		t.Fatalf("write resolved forced-reviewer waiver: %v", err)
	}
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

	recordResolvedForcedReviewerWaiverForTest(t, 7, "credentials/auth", "logins and passwords", "the login flow here is trivial, I checked it myself")

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

	recordResolvedForcedReviewerWaiverForTest(t, 9, "credentials/auth", "logins and passwords", "already reviewed this by hand")

	reviewers := queenForcedContinueReviewers(phase, nil, []string{"internal/auth/session.go"})
	if len(reviewers) != 0 {
		t.Fatalf("a signal waived from plan wording must stay waived when changed files re-detect it; got %+v", reviewers)
	}
}

// TestWaiverControlsBothFinalContinueDispatchLists is the dispatch-boundary
// proof for D-03. It is not enough for queenForcedContinueReviewers to filter
// a waived signal: both continue lanes must omit the reviewer from the team
// they actually dispatch. A second, still-live signal for the same caste must
// continue to force that reviewer.
func TestWaiverControlsBothFinalContinueDispatchLists(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]
	phase := checkinFixturePhase("Password reset", "Let users reset their password", colony.PhaseModePrototype)
	setUpCheckinFixtureColony(t, dataDir, phase)
	manifestMap, dispatches := manifestMapFromBuild(t, root, 1)
	checkin, _ := renderCeremonyTeamCheckin("build", manifestMap, dispatches)
	capability := waiverCapabilityFromCheckin(t, checkin, "credentials/auth")
	question := forcedReviewerWaiverQuestionText(1, "credentials/auth", "logins and passwords")
	if _, found, err := resolveForcedReviewerWaiverPendingDecision(question, "owner checked this login change", 1, capability); err != nil || !found {
		t.Fatalf("resolve authentic waiver row: found=%v err=%v", found, err)
	}

	assertBothLanes := func(t *testing.T, phase colony.Phase, wantGatekeeper bool) {
		t.Helper()
		inProcess := plannedContinueReviewDispatches(
			root, phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{},
			&codex.FakeInvoker{}, time.Minute, colony.VerificationDepthStandard, nil, "",
		)
		external := plannedExternalContinueDispatches(
			root, phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{},
			time.Minute, colony.VerificationDepthStandard, true, nil, "",
		)
		if got := containsDispatchCaste(inProcess, "gatekeeper"); got != wantGatekeeper {
			t.Fatalf("in-process gatekeeper=%v, want %v; dispatches=%+v", got, wantGatekeeper, inProcess)
		}
		if got := ownerDialsHasReviewCaste(external, "gatekeeper"); got != wantGatekeeper {
			t.Fatalf("external gatekeeper=%v, want %v; dispatches=%+v", got, wantGatekeeper, external)
		}
	}

	t.Run("waived signal is absent from both final teams", func(t *testing.T) {
		phase := waiverFixturePhase(1, "Password reset", "Let users reset their password")
		assertBothLanes(t, phase, false)
	})

	t.Run("second live signal for same caste still forces reviewer", func(t *testing.T) {
		phase := waiverFixturePhase(1, "Refund after login", "Let users log in and request a refund")
		assertBothLanes(t, phase, true)
	})
}

// TestWaiverReconciliationPreservesCanonicalExplicitProposal is CR-04's
// final-boundary regression. Judgement accepts aliases and comma-packed caste
// flags; the later waiver reconciliation must preserve the same canonical
// reviewer rather than comparing its raw spelling.
func TestWaiverReconciliationPreservesCanonicalExplicitProposal(t *testing.T) {
	setupBuildFlowTest(t)
	recordResolvedForcedReviewerWaiverForTest(
		t, 21, "credentials/auth", "logins and passwords", "owner waived the automatic credentials signal",
	)
	phase := waiverFixturePhase(21, "Password reset", "Let users reset their password")

	cases := []struct {
		name     string
		proposed []string
		reasons  map[string]string
	}{
		{
			name:     "security alias",
			proposed: []string{"security"},
			reasons:  map[string]string{"gatekeeper": "independently review the login boundary"},
		},
		{
			name:     "comma-packed proposal",
			proposed: []string{"gatekeeper,auditor"},
			reasons: map[string]string{
				"gatekeeper": "independently review the login boundary",
				"auditor":    "audit the reset-flow behavior",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dispatches := queenContinueDispatchesWithJudgement(
				phase,
				colony.VerificationDepthStandard,
				tc.proposed,
				"explicit owner team",
				nil,
				nil,
				tc.reasons,
			)
			if !queenContinueHasCaste(dispatches, "gatekeeper") {
				t.Fatalf("canonical explicit Gatekeeper was removed after waiver reconciliation: %+v", dispatches)
			}
		})
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

	recordResolvedForcedReviewerWaiverForTest(t, 1, "credentials/auth", "logins and passwords", "already reviewed by hand")

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
	checkin, visual := renderCeremonyTeamCheckin("build", manifestMap, dispatches)
	if !strings.Contains(visual, "To decline, run:") {
		t.Fatalf("expected the card to show a live decline command before answering it; visual:\n%s", visual)
	}
	capability := waiverCapabilityFromCheckin(t, checkin, "credentials/auth")

	question := forcedReviewerWaiverQuestionText(1, "credentials/auth", "logins and passwords")
	rootCmd.SetArgs([]string{"decision-answer", "--question", question, "--answer", "already checked by hand", "--phase", "1", "--waiver-capability", capability})
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

// TestGenericPendingDecisionResolverCannotWaiveForcedReviewer is CR-01's
// public-command regression. The generic pending-decision commands may list a
// protected waiver row for observability, but its ID is not authorization:
// only decision-answer with the capability printed on the check-in card may
// resolve it.
func TestGenericPendingDecisionResolverCannotWaiveForcedReviewer(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
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
		t.Fatalf("expected the card render to create a protected waiver row; visual:\n%s", visual)
	}

	var listOut bytes.Buffer
	stdout = &listOut
	rootCmd.SetArgs([]string{"pending-decision-list", "--unresolved"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pending-decision-list returned error: %v", err)
	}
	listEnvelope := parseEnvelope(t, listOut.String())
	listResult := listEnvelope["result"].(map[string]interface{})
	listed := listResult["decisions"].([]interface{})
	if len(listed) != 1 {
		t.Fatalf("listed decisions = %d, want the one protected waiver row", len(listed))
	}
	row := listed[0].(map[string]interface{})
	id, _ := row["id"].(string)
	if id == "" {
		t.Fatal("protected row was not observable through pending-decision-list")
	}

	var resolveOut, resolveErr bytes.Buffer
	stdout = &resolveOut
	stderr = &resolveErr
	rootCmd.SetArgs([]string{"pending-decision-resolve", "--id", id, "--resolution", "worker bypass"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pending-decision-resolve returned command error: %v", err)
	}

	if waived, reason := forcedReviewerWaiver(1, "credentials/auth"); waived {
		t.Fatalf("generic pending-decision-resolve bypassed the owner capability: %q", reason)
	}
	reviewers := queenForcedContinueReviewers(phase, nil, nil)
	if len(reviewers) != 1 || reviewers[0].Caste != "gatekeeper" {
		t.Fatalf("protected reviewer did not remain forced after generic resolve: %+v", reviewers)
	}

	pending := loadPendingDecisionFile()
	if len(pending.Decisions) != 1 || pending.Decisions[0].Resolved {
		t.Fatalf("protected waiver row was mutated by the generic resolver: %+v", pending.Decisions)
	}
	if _, exposed := row["waiver_capability_sha256"]; exposed {
		t.Fatalf("generic list exposed the protected capability hash: %#v", row)
	}
	if _, exposed := row["attempt_id"]; exposed {
		t.Fatalf("generic list exposed the protected build-attempt binding: %#v", row)
	}
}

// TestRepeatedCheckinRenderKeepsFirstWaiverCapabilityLive is CR-03's
// idempotence regression. The wrapper renders the card once for the owner and
// immediately renders it again as JSON. That second representation must not
// invalidate the decline command the owner already saw.
func TestRepeatedCheckinRenderKeepsFirstWaiverCapabilityLive(t *testing.T) {
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

	first, _ := renderCeremonyTeamCheckin("build", manifestMap, dispatches)
	firstCapability := waiverCapabilityFromCheckin(t, first, "credentials/auth")
	second, _ := renderCeremonyTeamCheckin("build", manifestMap, dispatches)
	_ = waiverCapabilityFromCheckin(t, second, "credentials/auth")

	question := forcedReviewerWaiverQuestionText(1, "credentials/auth", "logins and passwords")
	if _, found, err := resolveForcedReviewerWaiverPendingDecision(
		question, "owner used the first displayed command", 1, firstCapability,
	); err != nil || !found {
		t.Fatalf("first displayed capability was invalidated by the second render: found=%v err=%v", found, err)
	}
	if waived, reason := forcedReviewerWaiver(1, "credentials/auth"); !waived || reason != "owner used the first displayed command" {
		t.Fatalf("first displayed command did not produce the owner waiver: waived=%v reason=%q", waived, reason)
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
	checkin, visual := renderCeremonyTeamCheckin("build", manifestMap, dispatches)
	if !strings.Contains(visual, "To decline, run:") {
		t.Fatalf("expected the card to show a live decline command; visual:\n%s", visual)
	}
	capability := waiverCapabilityFromCheckin(t, checkin, "credentials/auth")

	// Dispatch genuinely begins -- the owner never ran decision-answer.
	rootCmd.SetArgs(spawnLogArgsForPhase(1))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-log returned error: %v", err)
	}
	rootCmd.SetArgs([]string{})

	// A later decision-answer call with the card's exact question -- the
	// forgery/misuse this test exists for -- must now be refused.
	question := forcedReviewerWaiverQuestionText(1, "credentials/auth", "logins and passwords")
	rootCmd.SetArgs([]string{"decision-answer", "--question", question, "--answer", "auto", "--phase", "1", "--waiver-capability", capability})
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

// TestSpawnLogFailsClosedWhenWaiverWindowCannotPersist is CR-02's durable
// boundary proof. If the dispatch marker cannot be written, spawn-log must
// refuse before recording the worker; otherwise the wrapper would launch a
// worker while the displayed owner capability remained usable.
func TestSpawnLogFailsClosedWhenWaiverWindowCannotPersist(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
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
		t.Fatalf("expected a live owner capability before testing dispatch persistence; visual:\n%s", visual)
	}

	// A directory at the marker's file path forces the storage layer's atomic
	// rename to fail without changing permissions on the whole test store.
	markerPath := filepath.Join(dataDir, phaseDispatchWindowFileName)
	if err := os.Mkdir(markerPath, 0o755); err != nil {
		t.Fatalf("obstruct dispatch marker path: %v", err)
	}

	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf
	rootCmd.SetArgs(spawnLogArgsForPhase(1))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-log returned a Cobra error instead of a structured refusal: %v", err)
	}
	if errBuf.Len() == 0 {
		t.Fatalf("spawn-log succeeded even though the waiver-window marker was not durable: %s", outBuf.String())
	}
	envelope := parseEnvelope(t, errBuf.String())
	if envelope["ok"] != false {
		t.Fatalf("spawn-log did not fail closed: %v", envelope)
	}

	spawnData, err := os.ReadFile(filepath.Join(dataDir, "spawn-tree.txt"))
	if err == nil && strings.Contains(string(spawnData), "Mason-1") {
		t.Fatalf("spawn-log recorded a worker after marker persistence failed:\n%s", spawnData)
	}
	if _, started := phaseDispatchStartedAt(1); started {
		t.Fatal("failed marker write unexpectedly reported a durable dispatch start")
	}
}

func TestWorkerCannotForceReplanToReopenReviewerDecline(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]
	forceBuildJSONOutput(t)
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	phase := checkinFixturePhase(
		"Password reset",
		"Let users reset their password via an emailed token",
		colony.PhaseModePrototype,
	)
	setUpCheckinFixtureColony(t, dataDir, phase)

	manifestMap, dispatches := manifestMapFromBuild(t, root, 1)
	checkin, _ := renderCeremonyTeamCheckin("build", manifestMap, dispatches)
	capability := waiverCapabilityFromCheckin(t, checkin, "credentials/auth")
	rootCmd.SetArgs(spawnLogArgsForPhase(1))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-log returned error: %v", err)
	}

	// This is the worker-accessible sequence from CR-04: force a fresh public
	// plan-only attempt, render a card, then submit the deterministic answer.
	rootCmd.SetArgs([]string{"build", "1", "--plan-only", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("forced plan-only command returned error: %v", err)
	}
	renderCeremonyTeamCheckin("build", manifestMap, dispatches)
	question := forcedReviewerWaiverQuestionText(1, "credentials/auth", "logins and passwords")
	rootCmd.SetArgs([]string{"decision-answer", "--question", question, "--answer", "worker forged this", "--phase", "1", "--waiver-capability", capability})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("decision-answer returned error: %v", err)
	}

	if waived, reason := forcedReviewerWaiver(1, "credentials/auth"); waived {
		t.Fatalf("worker force/replan/card sequence forged an owner waiver: %q", reason)
	}
	if reviewers := queenForcedContinueReviewers(phase, nil, nil); len(reviewers) != 1 {
		t.Fatalf("reviewer must remain forced after worker attack; reviewers=%+v", reviewers)
	}
}

func TestForcedReviewerWaiverRejectsLookalikeDecisionSource(t *testing.T) {
	setupBuildFlowTest(t)
	phaseID := 1
	question := forcedReviewerWaiverQuestionText(phaseID, "credentials/auth", "logins and passwords")
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{{
		ID:          "lookalike",
		Type:        clarificationDecisionType,
		Description: formatClarificationDescription(question, nil),
		Phase:       &phaseID,
		Source:      "worker-handoff",
		Resolution:  "worker supplied",
		Resolved:    true,
		CreatedAt:   time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
		ResolvedAt:  time.Now().UTC().Format(time.RFC3339),
	}}}); err != nil {
		t.Fatalf("write lookalike decision: %v", err)
	}
	if waived, reason := forcedReviewerWaiver(phaseID, "credentials/auth"); waived {
		t.Fatalf("lookalike source forged a waiver: %q", reason)
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
	checkin, _ := renderCeremonyTeamCheckin("build", manifestMap, dispatches)
	capability := waiverCapabilityFromCheckin(t, checkin, "credentials/auth")

	// The owner declines BEFORE dispatch begins -- the real path.
	question := forcedReviewerWaiverQuestionText(1, "credentials/auth", "logins and passwords")
	rootCmd.SetArgs([]string{"decision-answer", "--question", question, "--answer", "already checked by hand", "--phase", "1", "--waiver-capability", capability})
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

// TestForcedReviewerDeclineWindowReopensOnRetriedBuild is CR-01's residual
// regression proof (194-REVIEW.md iteration 3/4): the decline window closing
// correctly WITHIN one attempt (TestForcedReviewerDeclineWindowClosesWhenDispatchBegins)
// had the unintended side effect of never reopening for a LATER attempt of
// the same phase number -- and build, fail or get reviewed, rebuild the same
// phase is this repo's own normal loop, not an edge case. This reproduces
// exactly that: attempt 1 renders the card and genuinely dispatches (closing
// the window), attempt 2 (the retry) renders the card again and the owner
// declines BEFORE any attempt-2 worker is dispatched -- the real path, made
// in the real order, through the real CLI command. Before this fix,
// forcedReviewerWaiver(1, ...) returned false here because attempt 1's stale
// dispatch-start record never expired.
func TestForcedReviewerDeclineWindowReopensOnRetriedBuild(t *testing.T) {
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

	// Attempt 1: card renders, dispatch genuinely begins (the owner never
	// declines -- the default "proceed" path), attempt 1 fails/ends.
	manifestMap1, dispatches1 := manifestMapFromBuild(t, root, 1)
	if _, visual := renderCeremonyTeamCheckin("build", manifestMap1, dispatches1); !strings.Contains(visual, "To decline, run:") {
		t.Fatalf("expected attempt 1's card to show a live decline command")
	}
	rootCmd.SetArgs(spawnLogArgsForPhase(1))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("attempt 1 spawn-log returned error: %v", err)
	}
	rootCmd.SetArgs([]string{})
	if err := interruptLatestBuildAttempt(1, "attempt 1 failed before retry"); err != nil {
		t.Fatalf("mark attempt 1 terminal before retry: %v", err)
	}

	// Attempt 2 (the retry): a brand-new `aether build --plan-only` call for
	// the SAME phase number. The card renders again, showing a live decline
	// command -- and the owner runs it BEFORE any attempt-2 worker has been
	// dispatched.
	manifestMap2, dispatches2 := manifestMapFromBuild(t, root, 1)
	checkin2, visual := renderCeremonyTeamCheckin("build", manifestMap2, dispatches2)
	if !strings.Contains(visual, "To decline, run:") {
		t.Fatalf("expected attempt 2's (retried) card to show a live decline command")
	}
	capability := waiverCapabilityFromCheckin(t, checkin2, "credentials/auth")

	question := forcedReviewerWaiverQuestionText(1, "credentials/auth", "logins and passwords")
	rootCmd.SetArgs([]string{"decision-answer", "--question", question, "--answer", "already checked by hand", "--phase", "1", "--waiver-capability", capability})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("decision-answer returned error: %v", err)
	}
	rootCmd.SetArgs([]string{})

	waived, reason := forcedReviewerWaiver(1, "credentials/auth")
	if !waived {
		t.Fatalf("the owner's genuine decline on the RETRIED attempt, made before that attempt's dispatch began, must be honored -- attempt 1's stale dispatch-start record must not block it")
	}
	if reason != "already checked by hand" {
		t.Fatalf("reason = %q, want the owner's recorded answer", reason)
	}
	if reviewers := queenForcedContinueReviewers(phase, nil, nil); len(reviewers) != 0 {
		t.Fatalf("owner's declined reviewer must not be forced after the retry's genuine decline; reviewers = %+v", reviewers)
	}

	// The within-attempt guarantee must still hold for attempt 2 itself: had
	// the owner NOT declined and dispatch had genuinely begun instead, a
	// LATER decision-answer call would still be refused. Verified by the
	// pre-existing TestForcedReviewerDeclineWindowClosesWhenDispatchBegins,
	// which this test does not duplicate.
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
