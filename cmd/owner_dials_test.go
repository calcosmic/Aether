package cmd

import (
	"testing"

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
