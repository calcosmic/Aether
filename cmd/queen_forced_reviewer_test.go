package cmd

import (
	"reflect"
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
			dispatches := queenContinueDispatchesWithJudgement(phase, colony.VerificationDepthLight, nil, "", nil, nil)

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
	dispatches := queenContinueDispatchesWithJudgement(phase, colony.VerificationDepthLight, nil, "", nil, nil)

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

	dispatches := queenContinueDispatchesWithJudgement(phase, colony.VerificationDepthLight, nil, "", nil, nil)
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

// TestChangedFilesCanOnlyAddAForcedReviewer is D-02's add-only proof for the
// file-detected half of the forced-reviewer union
// (queenRiskSignalHitsFromPaths, wired through queenForcedContinueReviewers):
// changed files that match no signal leave the recorded set byte-identical
// to running with no changed files at all, and a genuine hit (a migrations/
// file) can only ADD a caste — the recorded caste and its reason survive
// unchanged. See queen_risk_signals.go's own doc comment naming D-02; the
// function must never remove a caste or a signal the build's own record
// carried.
func TestChangedFilesCanOnlyAddAForcedReviewer(t *testing.T) {
	recorded := []codexForcedReviewerRecord{{
		Caste:   "gatekeeper",
		Signals: []string{"credentials/auth"},
		Matches: []string{"password reset"},
		Sources: []string{"plan wording"},
		Reason:  `this touches logins and passwords (the plan mentions "password reset")`,
	}}
	phase := judgementPhase("Password reset", "Let users reset their password via an emailed token", colony.PhaseModePrototype)

	empty := queenForcedContinueReviewers(phase, recorded, nil)
	noMatch := queenForcedContinueReviewers(phase, recorded, []string{"src/dashboard/widget.go"})
	if !reflect.DeepEqual(empty, noMatch) {
		t.Fatalf("changed files matching no pattern altered the forced set:\nno changed files = %+v\nno-match changed files = %+v", empty, noMatch)
	}
	if len(empty) != 1 || empty[0].Caste != "gatekeeper" {
		t.Fatalf("recorded set = %+v, want exactly the recorded gatekeeper", empty)
	}

	withMigration := queenForcedContinueReviewers(phase, recorded, []string{"migrations/0007_add_column.sql"})
	if len(withMigration) != 2 {
		t.Fatalf("a migrations/ changed file did not ADD a caste, want exactly two: %+v", withMigration)
	}
	var sawGatekeeper, sawAuditor bool
	for _, reviewer := range withMigration {
		switch reviewer.Caste {
		case "gatekeeper":
			sawGatekeeper = true
			if !strings.Contains(reviewer.Reason, "password reset") {
				t.Errorf("the recorded gatekeeper's reason did not survive the union: %q", reviewer.Reason)
			}
		case "auditor":
			sawAuditor = true
			if !strings.HasPrefix(reviewer.Reason, "this touches") {
				t.Errorf("file-detected auditor reason missing the forced-reviewer sentence shape: %q", reviewer.Reason)
			}
			// IN-03 (194-REVIEW.md): the rendered reason strips the raw
			// pattern's trailing directory separator before it reaches the
			// owner — "migrations" reads as a sentence, "migrations/" reads
			// as an internal identifier fragment.
			if !strings.Contains(reviewer.Reason, "migrations") || strings.Contains(reviewer.Reason, "migrations/") {
				t.Errorf("file-detected auditor reason does not name the matched pattern without a trailing slash: %q", reviewer.Reason)
			}
		default:
			t.Errorf("unexpected caste in the union: %s", reviewer.Caste)
		}
	}
	if !sawGatekeeper {
		t.Fatalf("recorded gatekeeper did not survive a changed-file union: %+v", withMigration)
	}
	if !sawAuditor {
		t.Fatalf("a migrations/ changed file did not add auditor: %+v", withMigration)
	}
}

// TestBothContinueLanesForceTheSameReviewers is the two-lane parity proof
// D-02/D-05 require: the in-process continue lane
// (plannedContinueReviewDispatches) and the wrapper/external lane
// (plannedExternalContinueDispatches) must force the SAME caste set for the
// same phase, the same recorded forced-reviewer set, and the same changed
// files — both read phaseChangedFilesFromHandoffs from the same store, so a
// divergence here would mean the two boundaries stopped calling the shared
// union the same way (the exact class of bug the 2026-08-21 review gate
// found: the in-process lane half-wired while the wrapper lane worked).
func TestBothContinueLanesForceTheSameReviewers(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]

	phase := colony.Phase{
		ID:          1,
		Name:        "Password reset",
		Description: "Let users reset their password via an emailed token",
		Mode:        colony.PhaseModePrototype,
	}
	recorded := forcedReviewerRecords(queenForcedReviewersForPhase(phase))
	if len(recorded) != 1 || recorded[0].Caste != "gatekeeper" {
		t.Fatalf("fixture phase did not force the expected recorded gatekeeper: %+v", recorded)
	}
	// A migration file the build never mentioned in wording — both lanes
	// must add auditor from this alone (D-02).
	writePhaseHandoffs(t, phase.ID, "migrations/0007_add_column.sql")

	manifest := codexContinueManifest{Present: true, Data: codexBuildManifest{ForcedReviewers: recorded}}

	inProcess := plannedContinueReviewDispatches(
		root, phase, manifest, codexContinueVerificationReport{}, codexContinueAssessment{},
		&codex.FakeInvoker{}, time.Minute, colony.VerificationDepthLight, nil, "",
	)
	external := plannedExternalContinueDispatches(
		root, phase, manifest, codexContinueVerificationReport{}, codexContinueAssessment{},
		time.Minute, colony.VerificationDepthLight, true, nil, "",
	)

	inProcessCastes := map[string]bool{}
	for _, dispatch := range inProcess {
		inProcessCastes[dispatch.Caste] = true
	}
	externalCastes := map[string]bool{}
	for _, dispatch := range external {
		if dispatch.Stage != "review" {
			continue
		}
		externalCastes[dispatch.Caste] = true
	}

	if len(inProcessCastes) != 2 || !inProcessCastes["gatekeeper"] || !inProcessCastes["auditor"] {
		t.Fatalf("in-process lane castes = %+v, want exactly gatekeeper+auditor", inProcessCastes)
	}
	if len(externalCastes) != len(inProcessCastes) {
		t.Fatalf("lanes disagree on caste count: in-process = %+v, external = %+v", inProcessCastes, externalCastes)
	}
	for caste := range inProcessCastes {
		if !externalCastes[caste] {
			t.Errorf("wrapper/external lane missing a caste the in-process lane forced: %s (in-process=%+v external=%+v)", caste, inProcessCastes, externalCastes)
		}
	}
	for caste := range externalCastes {
		if !inProcessCastes[caste] {
			t.Errorf("in-process lane missing a caste the wrapper/external lane forced: %s (in-process=%+v external=%+v)", caste, inProcessCastes, externalCastes)
		}
	}
}

// TestPathPatternMatchingRequiresAWordBoundary (WR-02, 194-REVIEW.md): a
// bare PathPattern like "session" must never fire on a plain substring
// buried inside an unrelated word -- matchesPathPatternAtBoundary replaces
// the old plain strings.Contains, which had no boundary check at all. The
// concrete false positive the review named: "session" matching
// "repossession_handler.go", a file with nothing to do with sessions.
func TestPathPatternMatchingRequiresAWordBoundary(t *testing.T) {
	falseAlarm := waiverFixturePhase(21, "Leasing feature", "Track leases and repossessions")
	reviewers := queenForcedContinueReviewers(falseAlarm, nil, []string{"pkg/leasing/repossession_handler.go"})
	if len(reviewers) != 0 {
		t.Fatalf("repossession_handler.go should not false-positive on the bare 'session' pattern: %+v", reviewers)
	}

	genuineHit := waiverFixturePhase(22, "Session feature", "Nothing risky in the wording itself")
	reviewers = queenForcedContinueReviewers(genuineHit, nil, []string{"internal/auth/session.go"})
	if len(reviewers) != 1 || reviewers[0].Caste != "gatekeeper" {
		t.Fatalf("a genuine session.go file should still force gatekeeper: %+v", reviewers)
	}
}

// TestBareAuthPathPatternCatchesAuthNamedFiles (WR-03, 194-REVIEW.md): the
// credentials/auth signal's PathPatterns carries a bare "auth" (not only
// the directory-anchored "auth/" the review flagged as the odd one out), so
// auth-named files at any depth are caught the same way session/
// credential/secrets files already are. matchesPathPatternAtBoundary's
// boundary discipline (WR-02) means a name where "auth" is fused directly
// into a longer identifier with no separator (authorization.go) is still
// outside this pattern's reach -- the same "token" vs "tokenizer"
// trade-off the phrase table already accepts for prose, applied here to
// paths.
func TestBareAuthPathPatternCatchesAuthNamedFiles(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		{"auth.go at the root of a package", "cmd/auth.go", true},
		{"auth_service.go with an underscore", "internal/auth_service.go", true},
		{"auth-config.yaml with a hyphen", "config/auth-config.yaml", true},
		{"a directory literally named auth/", "auth/handler.go", true},
		{"authorization.go is a different word, no boundary", "pkg/authorization.go", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			phase := waiverFixturePhase(23, "Unrelated wording", "Nothing risky mentioned here")
			reviewers := queenForcedContinueReviewers(phase, nil, []string{tc.path})
			got := len(reviewers) == 1 && reviewers[0].Caste == "gatekeeper"
			if got != tc.want {
				t.Errorf("path %q: forced gatekeeper = %v, want %v (reviewers = %+v)", tc.path, got, tc.want, reviewers)
			}
		})
	}
}

// TestForcedReviewerReasonNeverShowsATrailingSlash (IN-03, 194-REVIEW.md):
// forcedReviewerReasonClause must never quote a raw PathPattern with its
// trailing directory separator still attached -- "migrations/" reads as an
// internal identifier fragment to a non-technical owner, not a sentence.
func TestForcedReviewerReasonNeverShowsATrailingSlash(t *testing.T) {
	hit := riskSignalHit{
		Signal: queenRiskSignalTable[4], // "database migration"
		Match:  "migrations/",
		Source: "changed files",
	}
	clause := forcedReviewerReasonClause(hit)
	if strings.Contains(clause, "migrations/") {
		t.Fatalf("clause still carries a trailing slash: %q", clause)
	}
	if !strings.Contains(clause, "migrations") {
		t.Fatalf("clause dropped the matched pattern entirely: %q", clause)
	}
}
