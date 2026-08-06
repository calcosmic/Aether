package cmd

import (
	"sort"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func judgementPhase(name, description string, mode colony.PhaseMode) colony.Phase {
	return colony.Phase{ID: 1, Name: name, Description: description, Mode: mode}
}

func hasCasteName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

// TestQueenCannotDropTheWatcher is the load-bearing test of the whole
// judgement path. Letting a model choose the team is only safe if its
// judgement is bounded: a Queen that proposes a build with nothing verifying
// it must get a Watcher anyway. Without this, "the Queen is intelligent"
// becomes "the Queen can decide not to be checked".
func TestQueenCannotDropTheWatcher(t *testing.T) {
	phase := judgementPhase("Add a hello endpoint", "Implement the /hello route", colony.PhaseModePrototype)

	judgement := queenApplyJudgement([]string{"builder"}, "simple change, builder is enough", phase, "build", colony.ColonyState{})

	if !hasCasteName(judgement.Final, "watcher") {
		t.Fatalf("Watcher must be restored when omitted; final = %v", judgement.Final)
	}
	if !hasCasteName(judgement.Added, "watcher") {
		t.Errorf("restoring the Watcher must be reported in Added, got %v", judgement.Added)
	}
	if !strings.Contains(judgement.Summary(), "required for this phase regardless") {
		t.Errorf("summary must disclose the override, got: %s", judgement.Summary())
	}
}

// TestQueenCannotSkipSecurityReviewOnSecurityWork pins the other floor. A
// proposal is judgement about which optional specialists help, not permission
// to skip a security review on work that touches credentials.
func TestQueenCannotSkipSecurityReviewOnSecurityWork(t *testing.T) {
	phase := judgementPhase("Password reset", "Let users reset their password via an emailed token", colony.PhaseModeProduction)

	judgement := queenApplyJudgement([]string{"builder", "watcher"}, "straightforward form work", phase, "build", colony.ColonyState{})

	for _, caste := range []string{"gatekeeper", "auditor"} {
		if !hasCasteName(judgement.Final, caste) {
			t.Errorf("%s must be restored on credential work; final = %v", caste, judgement.Final)
		}
	}
}

// TestQueenCanAddASpecialistKeywordsWouldMiss is the point of the feature. The
// deterministic engine scores literal keywords, so a phase that needs a
// specialist without naming its vocabulary gets nobody. A reader can see what
// the words only imply.
func TestQueenCanAddASpecialistKeywordsWouldMiss(t *testing.T) {
	// No "performance", "latency", "benchmark" or "optimize" anywhere — the
	// keyword engine has nothing to match on, but a reader knows this is a
	// performance problem.
	phase := judgementPhase("Dashboard feels sluggish", "Users report the main list takes a long time to appear with many rows", colony.PhaseModeProduction)

	deterministic := casteNames(queenOrchestrate(phase, "build", colony.ColonyState{}))
	if hasCasteName(deterministic, "measurer") {
		t.Skip("keyword engine already selects measurer; fixture no longer demonstrates the gap")
	}

	judgement := queenApplyJudgement(
		[]string{"builder", "watcher", "measurer"},
		"the complaint is latency even though the phase never says so",
		phase, "build", colony.ColonyState{})

	if !hasCasteName(judgement.Final, "measurer") {
		t.Errorf("Queen's added specialist must survive; final = %v", judgement.Final)
	}
	if judgement.Source != "queen" {
		t.Errorf("source = %q, want queen", judgement.Source)
	}
	if judgement.Rationale == "" {
		t.Error("rationale must be carried through for display")
	}
}

// TestNoProposalFallsBackToDeterministic keeps the change additive. Every
// existing caller passes no proposal and must behave exactly as before.
func TestNoProposalFallsBackToDeterministic(t *testing.T) {
	phase := judgementPhase("Add user authentication", "Implement login and sessions", colony.PhaseModeProduction)

	judgement := queenApplyJudgement(nil, "", phase, "build", colony.ColonyState{})
	deterministic := casteNames(queenOrchestrate(phase, "build", colony.ColonyState{}))

	if judgement.Source != "deterministic" {
		t.Errorf("source = %q, want deterministic", judgement.Source)
	}
	if len(judgement.Final) != len(deterministic) {
		t.Fatalf("fallback team = %v, want %v", judgement.Final, deterministic)
	}
	for _, caste := range deterministic {
		if !hasCasteName(judgement.Final, caste) {
			t.Errorf("fallback dropped %s: %v", caste, judgement.Final)
		}
	}
}

// TestBudgetTrimsTheQueensOptionalPicksNotItsRequiredOnes pins the ceiling. A
// Queen that asks for everything gets trimmed, and the trimming must take its
// optional picks rather than the floor.
func TestBudgetTrimsTheQueensOptionalPicksNotItsRequiredOnes(t *testing.T) {
	phase := judgementPhase("Tidy the helpers", "Small cleanup of utility functions", colony.PhaseModeMaintenance)
	state := colony.ColonyState{}
	state.VerificationDepth = string(colony.VerificationDepthLight)

	greedy := []string{
		"builder", "watcher", "architect", "measurer", "chaos",
		"weaver", "archaeologist", "includer", "sage", "keeper",
	}
	judgement := queenApplyJudgement(greedy, "everything, just in case", phase, "build", state)

	budget := queenSpawnBudgetForPhase(phase, "build", state)
	if len(judgement.Final) > budget.MaxWorkers && len(judgement.Final) > len(budget.RequiredCastes) {
		t.Errorf("final team %v exceeds budget %d", judgement.Final, budget.MaxWorkers)
	}
	for _, caste := range budget.RequiredCastes {
		if !hasCasteName(judgement.Final, caste) {
			t.Errorf("budget trimming removed required caste %s: %v", caste, judgement.Final)
		}
	}
	if len(judgement.Dropped) == 0 {
		t.Error("expected the over-sized proposal to report what was dropped")
	}
	if !strings.Contains(judgement.Summary(), "over the worker budget") {
		t.Errorf("summary must disclose the trim, got: %s", judgement.Summary())
	}
}

// TestUnknownCasteIsReportedNotSwallowed matters because the Queen is a model
// writing names from memory. Asking for "security" when the caste is
// "gatekeeper" must surface, or the phase quietly runs without the specialist
// the Queen believed it had requested.
func TestUnknownCasteIsReportedNotSwallowed(t *testing.T) {
	phase := judgementPhase("Add CSV export", "Download the table", colony.PhaseModeProduction)

	judgement := queenApplyJudgement(
		[]string{"builder", "watcher", "security", "test-writer"},
		"", phase, "build", colony.ColonyState{})

	if len(judgement.Unknown) != 2 {
		t.Fatalf("unknown = %v, want both unrecognised names", judgement.Unknown)
	}
	if !strings.Contains(judgement.Summary(), "Ignored unknown caste(s)") {
		t.Errorf("summary must disclose unknown castes, got: %s", judgement.Summary())
	}
	for _, bogus := range []string{"security", "test-writer"} {
		if hasCasteName(judgement.Final, bogus) {
			t.Errorf("unknown caste %q must not reach the final team: %v", bogus, judgement.Final)
		}
	}
}

// TestProposalAcceptsCommaSeparatedValues keeps the flag forgiving: a model
// writing `--castes builder,watcher` and one writing repeated flags should
// both work, because getting this wrong looks like the Queen being ignored.
func TestProposalAcceptsCommaSeparatedValues(t *testing.T) {
	known, unknown := normalizeProposedCastes([]string{"builder, watcher", "probe"})
	if len(unknown) != 0 {
		t.Fatalf("unknown = %v, want none", unknown)
	}
	for _, want := range []string{"builder", "watcher", "probe"} {
		if !hasCasteName(known, want) {
			t.Errorf("missing %s from %v", want, known)
		}
	}
}

// TestRosterNamesEveryDispatchableCaste guards the input side. The Queen can
// only choose well from a roster that is complete; a caste missing here is one
// it will never propose.
func TestRosterNamesEveryDispatchableCaste(t *testing.T) {
	roster := queenCasteRoster()
	if len(roster) != len(casteRelevanceRegistry) {
		t.Fatalf("roster has %d entries, registry has %d", len(roster), len(casteRelevanceRegistry))
	}
	names := map[string]bool{}
	for _, entry := range roster {
		names[entry["caste"]] = true
		if strings.TrimSpace(entry["good_at"]) == "" {
			t.Errorf("roster entry %q has no description of what it is good at", entry["caste"])
		}
	}
	for _, profile := range casteRelevanceRegistry {
		if !names[profile.Caste] {
			t.Errorf("roster omits dispatchable caste %q", profile.Caste)
		}
	}
}

// TestQueenChoiceReachesTheDispatchList is the test that would have caught the
// gap between deciding and spawning.
//
// The judgement was recorded correctly in the manifest — proposed measurer,
// final included measurer — while applyBuildDispatchPolicyCastes deleted it one
// line later, so nothing spawned. Every visible surface said the Queen had been
// heard. A decision that does not reach a worker is worse than no decision,
// because it reads as working.
func TestQueenChoiceReachesTheDispatchList(t *testing.T) {
	phase := colony.Phase{
		ID:          1,
		Name:        "Dashboard feels sluggish",
		Description: "Users report the main list takes a long time to appear with many rows",
		Mode:        colony.PhaseModeProduction,
		Status:      colony.PhaseReady,
		Tasks:       []colony.Task{{Goal: "Speed up the list rendering"}},
	}
	state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{phase}}}

	dispatches := plannedBuildDispatchesWithJudgement(
		phase, state, nil, colony.VerificationDepthStandard,
		[]string{"builder", "measurer"},
		"the complaint is latency even though the phase never says so",
	)

	spawned := map[string]bool{}
	for _, dispatch := range dispatches {
		spawned[dispatch.Caste] = true
	}

	if !spawned["measurer"] {
		t.Errorf("Queen asked for a Measurer and none spawned; castes = %v", casteKeys(spawned))
	}
	// The floor still holds in the same list.
	if !spawned["watcher"] {
		t.Errorf("Watcher must spawn regardless of the proposal; castes = %v", casteKeys(spawned))
	}
}

// TestDepthPolicyStillAppliesWithoutAQueenChoice keeps the exemption narrow.
// Measurer and Chaos remain off by default at ordinary depth — the Queen's
// explicit request is the only thing that overrides the policy, not the mere
// existence of the judgement path.
func TestDepthPolicyStillAppliesWithoutAQueenChoice(t *testing.T) {
	phase := colony.Phase{
		ID:          1,
		Name:        "Dashboard feels sluggish",
		Description: "Users report the main list takes a long time to appear",
		Mode:        colony.PhaseModeProduction,
		Status:      colony.PhaseReady,
		Tasks:       []colony.Task{{Goal: "Speed up the list"}},
	}
	state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{phase}}}

	dispatches := plannedBuildDispatchesWithJudgement(phase, state, nil, colony.VerificationDepthStandard, nil, "")
	for _, dispatch := range dispatches {
		if dispatch.Caste == "measurer" {
			t.Error("Measurer must stay off at standard depth when the Queen did not ask for it")
		}
	}
}

func casteKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
