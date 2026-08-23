package cmd

import (
	"sort"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// fallbackTeamPhase builds a minimal phase fixture for these tests, mirroring
// the style of probeGatingPhase / judgementPhase elsewhere in this package.
func fallbackTeamPhase(name, description string, mode colony.PhaseMode) colony.Phase {
	taskID := "1.1"
	return colony.Phase{
		ID:          1,
		Name:        name,
		Description: description,
		Mode:        mode,
		Status:      colony.PhaseReady,
		Tasks:       []colony.Task{{ID: &taskID, Goal: description, Status: colony.TaskPending}},
	}
}

// TestNoProposalSendsBuilderPlusForcedOnly is the tracer for D-11: with no
// proposal, the automatic path now costs what the judgement path costs. A
// one-task bug fix used to be measured at eight workers across build and
// continue combined (194-CONTEXT.md, the v1.27 milestone brief); this test
// pins the floor that measurement is meant to fix.
func TestNoProposalSendsBuilderPlusForcedOnly(t *testing.T) {
	bugFix := fallbackTeamPhase("Fix the retry loop", "The retry loop never backs off between attempts", colony.PhaseModePrototype)
	standardState := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}

	buildCastes := casteDispatchSummary(queenOrchestrate(bugFix, "build", standardState))
	if len(buildCastes) != 1 || buildCastes[0] != "builder" {
		t.Fatalf("no-proposal build on a one-task bug fix = %v, want exactly [builder]", buildCastes)
	}

	continueCastes := casteDispatchSummary(queenOrchestrate(bugFix, "continue", standardState))
	if len(continueCastes) != 0 {
		t.Fatalf("no-proposal continue on a plain bug fix = %v, want no reviewer dispatch at all", continueCastes)
	}

	// A phase that DOES name one of the five signals (D-01) still gets its
	// forced reviewer at continue -- the build dispatch, per D-05, still
	// announces rather than dispatches it.
	passwordReset := fallbackTeamPhase("Password reset", "Let users reset their password by email", colony.PhaseModePrototype)

	buildCastes = casteDispatchSummary(queenOrchestrate(passwordReset, "build", standardState))
	if len(buildCastes) != 1 || buildCastes[0] != "builder" {
		t.Fatalf("no-proposal build on a password-reset phase = %v, want exactly [builder]", buildCastes)
	}

	continueCastes = casteDispatchSummary(queenOrchestrate(passwordReset, "continue", standardState))
	if len(continueCastes) != 1 || continueCastes[0] != "gatekeeper" {
		t.Fatalf("no-proposal continue on a password-reset phase = %v, want exactly [gatekeeper]", continueCastes)
	}
}

// TestDiscoveryFallbackSendsOneResearcher pins D-12: a discovery-mode build
// with no proposal suppresses the implementation caste (isCasteSuppressed,
// unchanged by this plan), so the required-caste floor alone would dispatch
// nobody. Research is the deliverable there, so the fallback is exactly one
// scout rather than an empty team.
func TestDiscoveryFallbackSendsOneResearcher(t *testing.T) {
	discovery := fallbackTeamPhase("Evaluate queue options", "Research candidate message brokers", colony.PhaseModeDiscovery)
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}

	dispatches := queenOrchestrate(discovery, "build", state)
	if len(dispatches) != 1 {
		t.Fatalf("discovery build with no proposal = %+v, want exactly one dispatch", dispatches)
	}
	if dispatches[0].Caste != "scout" {
		t.Fatalf("discovery build's one dispatch has caste %q, want scout", dispatches[0].Caste)
	}
}

// TestKeywordEngineNoLongerSelectsOnBuildOrContinue is the falsification test
// for D-11: a phase whose wording WOULD have cleared an optional specialist's
// spawn threshold under the old keyword engine gets nobody extra now, on
// either build or continue, with no proposal.
func TestKeywordEngineNoLongerSelectsOnBuildOrContinue(t *testing.T) {
	// "latency" and "memory" both match measurer's keyword list; measurer's
	// base score (20) plus two keyword hits (20) clears the old spawn
	// threshold (30) with room to spare.
	phase := fallbackTeamPhase(
		"Dashboard feels sluggish",
		"Users report the main list has high latency; investigate memory usage before shipping",
		colony.PhaseModeProduction,
	)
	standardState := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}

	// Confirm the premise: the old (still-live) selector WOULD pick measurer
	// for this fixture -- otherwise this test proves nothing.
	if !HasCaste(queenCandidateDispatches(phase, "build", standardState), "measurer") {
		t.Fatalf("fixture is broken: queenCandidateDispatches should still select measurer for this phase's wording")
	}

	buildCastes := casteDispatchSummary(queenOrchestrate(phase, "build", standardState))
	if len(buildCastes) != 1 || buildCastes[0] != "builder" {
		t.Fatalf("build with no proposal = %v, want exactly [builder] -- the keyword engine must not add measurer", buildCastes)
	}

	continueCastes := casteDispatchSummary(queenOrchestrate(phase, "continue", standardState))
	if len(continueCastes) != 0 {
		t.Fatalf("continue with no proposal = %v, want nothing -- the keyword engine must not add measurer", continueCastes)
	}
}

// TestOtherFlowsKeepTheirRequiredSets is the guard against over-reach: D-11
// gates exactly build and continue (queenSelectorIsGatedForFlow). Plan,
// colonize, swarm and seal must still select through the relevance engine
// exactly as they did before this plan -- proven by comparing queenOrchestrate
// (the gated entry point) against the pre-plan computation
// (applyQueenSpawnBudget over queenCandidateDispatches, unchanged by this
// plan) for a representative phase on each of the four flows.
func TestOtherFlowsKeepTheirRequiredSets(t *testing.T) {
	phase := fallbackTeamPhase(
		"Migrate user data to new schema",
		"Design the new table shape and write the migration script",
		colony.PhaseModeProduction,
	)
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}

	for _, flowType := range []string{"plan", "colonize", "swarm", "seal"} {
		if queenSelectorIsGatedForFlow(flowType) {
			t.Fatalf("flow %q must not be gated by D-11 -- only build and continue change in this phase", flowType)
		}

		before := casteDispatchSummary(applyQueenSpawnBudget(queenCandidateDispatches(phase, flowType, state), phase, flowType, state))
		after := casteDispatchSummary(queenOrchestrate(phase, flowType, state))

		sort.Strings(before)
		sort.Strings(after)
		if !equalStringSlices(before, after) {
			t.Errorf("flow %q caste set changed: before %v, after %v", flowType, before, after)
		}
		if len(after) == 0 {
			t.Errorf("flow %q produced an empty caste set for this fixture -- the comparison above would pass vacuously", flowType)
		}
	}
}
