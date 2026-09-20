package cmd

import (
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// ownerDialsPhase mirrors probeGatingPhase's style for these fixtures.
func ownerDialsPhase(name, description string, mode colony.PhaseMode) colony.Phase {
	return colony.Phase{
		ID:          3,
		Name:        name,
		Description: description,
		Mode:        mode,
	}
}

// TestContinueRequiredSetByDepth pins D-13's table exactly: light and
// standard require nothing unconditionally at continue; heavy requires the
// security reviewer and the quality reviewer always, and the coverage
// reviewer only where the phase produces testable code.
func TestContinueRequiredSetByDepth(t *testing.T) {
	docsPhase := ownerDialsPhase("Write the README", "Documentation for installation and usage", colony.PhaseModeMaintenance)
	codePhase := ownerDialsPhase("Add user authentication", "Implement login endpoints and session handling", colony.PhaseModeProduction)

	for _, tc := range []struct {
		name           string
		depth          colony.VerificationDepth
		phase          colony.Phase
		wantGatekeeper bool
		wantAuditor    bool
		wantProbe      bool
	}{
		{"light, documentation phase", colony.VerificationDepthLight, docsPhase, false, false, false},
		{"light, testable-code phase", colony.VerificationDepthLight, codePhase, false, false, false},
		{"standard, documentation phase", colony.VerificationDepthStandard, docsPhase, false, false, false},
		{"standard, testable-code phase", colony.VerificationDepthStandard, codePhase, false, false, false},
		{"heavy, documentation phase", colony.VerificationDepthHeavy, docsPhase, true, true, false},
		{"heavy, testable-code phase", colony.VerificationDepthHeavy, codePhase, true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := colony.ColonyState{VerificationDepth: string(tc.depth)}
			if got := isAlwaysRequired("gatekeeper", "continue", tc.phase, state); got != tc.wantGatekeeper {
				t.Errorf("isAlwaysRequired(gatekeeper) = %v, want %v", got, tc.wantGatekeeper)
			}
			if got := isAlwaysRequired("auditor", "continue", tc.phase, state); got != tc.wantAuditor {
				t.Errorf("isAlwaysRequired(auditor) = %v, want %v", got, tc.wantAuditor)
			}
			if got := isAlwaysRequired("probe", "continue", tc.phase, state); got != tc.wantProbe {
				t.Errorf("isAlwaysRequired(probe) = %v, want %v", got, tc.wantProbe)
			}
		})
	}

	// Explicit blanket check the acceptance criteria names: at light and at
	// standard, EVERY registered caste returns false for continue -- not
	// just the three reviewers spot-checked above.
	for _, depth := range []colony.VerificationDepth{colony.VerificationDepthLight, colony.VerificationDepthStandard} {
		state := colony.ColonyState{VerificationDepth: string(depth)}
		for _, profile := range casteRelevanceRegistry {
			if isAlwaysRequired(profile.Caste, "continue", docsPhase, state) {
				t.Errorf("isAlwaysRequired(%s, continue, %s) = true, want false -- %s requires nothing unconditionally (D-13)", profile.Caste, depth, depth)
			}
		}
	}
}

// TestLightCannotDropAForcedReviewer proves T-194-08: a light request can
// remove optional reviewers, but it can never reach a reviewer a named risk
// signal forced. The forced reviewer is unioned into
// queenRequiredCastesForBudget's own continue branch regardless of depth, so
// it bypasses the budget entirely (applyQueenSpawnBudget's isRequired
// branch) rather than surviving because it happened to score high enough.
func TestLightCannotDropAForcedReviewer(t *testing.T) {
	passwordReset := ownerDialsPhase("Password reset", "Let users reset their password by email", colony.PhaseModePrototype)
	lightState := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)}

	dispatches := queenOrchestrate(passwordReset, "continue", lightState)
	if !HasCaste(dispatches, "gatekeeper") {
		t.Fatalf("a light continue request dropped the forced security reviewer: %v", casteDispatchSummary(dispatches))
	}
}

// TestExplicitCastesOutrankLight proves the more specific instruction wins:
// when the owner asks for light AND names a caste with a reason, the named
// caste survives -- light only trims what was not named.
func TestExplicitCastesOutrankLight(t *testing.T) {
	phase := ownerDialsPhase("Clean up the settings page", "Reorganise the settings form for readability", colony.PhaseModePrototype)
	lightState := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)}

	judgement := queenApplyJudgement(
		[]string{"auditor"}, "",
		phase, "continue", lightState,
		map[string]string{"auditor": "the owner asked for an extra quality pass before this ships"},
	)
	if !hasCasteName(judgement.Final, "auditor") {
		t.Fatalf("an explicitly named, reasoned caste should survive a light request; Final = %v, Refused = %v, RefusedNoReason = %v",
			judgement.Final, judgement.Refused, judgement.RefusedNoReason)
	}
}

// TestPositionNoLongerRaisesVerificationDepth pins D-06: the last phase of a
// plan gets the same automatic verification depth as a middle phase with
// identical name, mode and risk. An explicit --heavy request on that same
// middle phase still returns heavy, so this test cannot pass merely because
// depth became a constant.
func TestPositionNoLongerRaisesVerificationDepth(t *testing.T) {
	totalPhases := 5
	newPhase := func(id int) colony.Phase {
		return colony.Phase{
			ID:   id,
			Name: "Clean up the settings page",
			Mode: colony.PhaseModePrototype,
		}
	}

	lastPhase := newPhase(5)
	middlePhase := newPhase(3)

	lastDepth := resolveSmartVerificationDepth(lastPhase, totalPhases)
	middleDepth := resolveSmartVerificationDepth(middlePhase, totalPhases)
	if lastDepth != middleDepth {
		t.Fatalf("phase position changed automatic verification depth: last phase (id %d) = %s, middle phase (id %d) = %s -- position must no longer raise depth (D-06)",
			lastPhase.ID, lastDepth, middlePhase.ID, middleDepth)
	}

	// The half of the rule that survives: an explicit --heavy request still
	// raises depth, on any phase, position included.
	if got := resolveVerificationDepth(middlePhase, totalPhases, false, true, ""); got != colony.VerificationDepthHeavy {
		t.Fatalf("an explicit --heavy request on the middle phase = %s, want heavy", got)
	}
}

// TestHeavyGivesTheFullReviewPanel proves D-13 (TEAM-05) on the path people
// actually run, not just on the shared judgement function underneath it: an
// explicit heavy request with no proposal produces the full review panel
// (security + quality + coverage-where-testable) on BOTH continue lanes --
// the in-process lane (plannedContinueReviewDispatches) and the wrapper lane
// (plannedExternalContinueDispatches), matching the two-lane parity
// discipline TestBothContinueLanesForceTheSameReviewers pins for D-02.
func TestHeavyGivesTheFullReviewPanel(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]

	phase := colony.Phase{
		ID:          3,
		Name:        "Add user authentication",
		Description: "Implement login endpoints and session handling",
		Mode:        colony.PhaseModeProduction,
	}
	manifest := codexContinueManifest{}

	inProcess := plannedContinueReviewDispatches(
		root, phase, manifest, codexContinueVerificationReport{}, codexContinueAssessment{},
		&codex.FakeInvoker{}, time.Minute, colony.VerificationDepthHeavy, nil, "",
	)
	external := plannedExternalContinueDispatches(
		root, phase, manifest, codexContinueVerificationReport{}, codexContinueAssessment{},
		time.Minute, colony.VerificationDepthHeavy, true, nil, "",
	)

	for _, want := range []string{"gatekeeper", "auditor", "probe"} {
		if !containsDispatchCaste(inProcess, want) {
			t.Errorf("in-process heavy continue missing %s: %+v", want, inProcess)
		}
		found := false
		for _, dispatch := range external {
			if dispatch.Stage == "review" && dispatch.Caste == want {
				found = true
			}
		}
		if !found {
			t.Errorf("wrapper/external heavy continue missing %s: %+v", want, external)
		}
	}
}

// TestAutopilotPathSendsTheFallbackTeam exercises the exact function
// autopilot's own CLI path reaches for its no-proposal team: autopilot.go
// carries no dispatch call of its own -- it shells out to the same
// aether build / aether continue commands a human runs, with no --castes
// proposal -- so this asserts the function those unproposed commands call
// (queenOrchestrate -> queenFallbackTeam, D-11), per this task's own
// fall-back instruction for when the CLI path itself is not callable from a
// unit test without heavy fixtures.
//
// The fixture's wording ("preserve knowledge and conventions") is the exact
// phrasing TestContinueDoesNotSummonKeeperOnIncidentalWords already proves
// clears the OLD scored engine's threshold for Keeper (a genuine, non-zero
// relevance hit, not incidental wording) -- proving the no-proposal fallback
// team does NOT carry Keeper for this same phase proves it reached
// queenFallbackTeam, D-11's required-caste floor, rather than
// queenCandidateDispatches, the old scored selector.
func TestAutopilotPathSendsTheFallbackTeam(t *testing.T) {
	phase := ownerDialsPhase("Capture conventions", "Preserve knowledge and conventions for future workers", colony.PhaseModeMaintenance)
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}

	// Prove the fixture is real: the scored engine WOULD select keeper for
	// this wording (queenCandidateDispatches is untouched by D-11) --
	// otherwise the assertions below would prove nothing.
	if !HasCaste(queenCandidateDispatches(phase, "continue", state), "keeper") {
		t.Fatalf("fixture broken: the scored engine no longer selects keeper for this wording")
	}

	buildDispatches := queenOrchestrate(phase, "build", state)
	if HasCaste(buildDispatches, "keeper") {
		t.Fatalf("autopilot's build path reached the old scored team, not the D-11 fallback: %v", casteDispatchSummary(buildDispatches))
	}
	if !HasCaste(buildDispatches, "builder") {
		t.Fatalf("autopilot's build fallback is missing the implementation worker: %v", casteDispatchSummary(buildDispatches))
	}

	continueDispatches := queenOrchestrate(phase, "continue", state)
	if HasCaste(continueDispatches, "keeper") {
		t.Fatalf("autopilot's continue path reached the old scored team, not the D-11 fallback: %v", casteDispatchSummary(continueDispatches))
	}
}

// TestForcedReviewerAddedByChangedFilesSatisfiesThePlannedSubsetCheck proves
// the wrapper lane's planned-subset guarantee (continueReviewCastesArePlannedSubset,
// cmd/codex_continue_finalize.go) still holds once the changed-files detector
// (task 1) can add a reviewer between the planned manifest and the finalize
// call: the file detector fires INSIDE plannedExternalContinueDispatches,
// which is the exact call that produces the "planned" manifest the wrapper
// persists (codexContinuePlanManifest.Dispatches, cmd/codex_continue_plan.go)
// — so a file-forced reviewer is already part of "planned" the moment
// continue plans, before any worker is dispatched. This task's own action
// text requires that if the check fails, the planned set is corrected, never
// the subset check weakened; this test proves the check ALREADY holds by
// construction, so no correction was needed.
func TestForcedReviewerAddedByChangedFilesSatisfiesThePlannedSubsetCheck(t *testing.T) {
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
	writePhaseHandoffs(t, phase.ID, "migrations/0007_add_column.sql")
	manifest := codexContinueManifest{Present: true, Data: codexBuildManifest{ForcedReviewers: recorded}}

	// This is the exact call that produces the persisted "planned" manifest
	// (codexContinuePlanManifest.Dispatches) at continue-plan time.
	planned := plannedExternalContinueDispatches(
		root, phase, manifest, codexContinueVerificationReport{}, codexContinueAssessment{},
		time.Minute, colony.VerificationDepthLight, true, nil, "",
	)
	if !ownerDialsHasReviewCaste(planned, "auditor") {
		t.Fatalf("fixture broken: the changed file did not add auditor to the planned set: %+v", planned)
	}

	expectedCastes := expectedContinueReviewCastes(colony.VerificationDepthLight, planned)
	// The workers that actually ran are exactly the ones the plan
	// dispatched — the finalize step's "actual" castes.
	actualCastes := ownerDialsActualReviewCastes(planned)

	if !continueReviewCastesArePlannedSubset(actualCastes, expectedCastes) {
		t.Fatalf("planned-subset check failed with a changed-files-forced reviewer: actual=%v expected=%v", actualCastes, expectedCastes)
	}
}

// ownerDialsHasReviewCaste checks a codexContinueExternalDispatch
// review-stage list for a caste — a distinct helper from HasCaste, which is
// typed for []CasteDispatch, because expectedContinueReviewCastes' input
// type is codexContinueExternalDispatch, not CasteDispatch.
func ownerDialsHasReviewCaste(dispatches []codexContinueExternalDispatch, caste string) bool {
	for _, dispatch := range dispatches {
		if dispatch.Stage == "review" && dispatch.Caste == caste {
			return true
		}
	}
	return false
}

// ownerDialsActualReviewCastes mirrors actualContinueReviewCastes' shape but
// reads straight off the planned dispatch list rather than a worker-flow
// report — standing in for "the workers that actually ran", which for this
// test are exactly the ones the plan dispatched.
func ownerDialsActualReviewCastes(dispatches []codexContinueExternalDispatch) []string {
	castes := make([]string, 0, len(dispatches))
	for _, dispatch := range dispatches {
		if dispatch.Stage == "review" {
			castes = append(castes, dispatch.Caste)
		}
	}
	return castes
}
