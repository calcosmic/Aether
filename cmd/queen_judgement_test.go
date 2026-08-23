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

// TestQueenCannotSkipSecurityReviewOnSecurityWork pins the other floor. A
// proposal is judgement about which optional specialists help, not permission
// to skip a security review on work that touches credentials.
//
// Plan 194-02 (D-05, D-07) moved this floor: it is no longer an unconditional
// build-side restoration (mode/production inferred auditor+gatekeeper on
// every build) but a named-risk signal forced at the continue step only.
// This test now proves both halves of that move: the caste is still
// unskippable (queenApplyJudgement restores it into Final on the continue
// flow), AND the restored dispatch states WHY (D-09) -- something the old
// build-side restoration never had to say.
func TestQueenCannotSkipSecurityReviewOnSecurityWork(t *testing.T) {
	phase := judgementPhase("Password reset", "Let users reset their password via an emailed token", colony.PhaseModeProduction)

	judgement := queenApplyJudgement([]string{"builder"}, "straightforward form work", phase, "continue", colony.ColonyState{})
	if !hasCasteName(judgement.Final, "gatekeeper") {
		t.Fatalf("gatekeeper must be restored on credential work; final = %v", judgement.Final)
	}

	dispatches := queenContinueDispatchesWithJudgement(phase, colony.VerificationDepthLight, []string{"builder"}, "straightforward form work", nil)
	found := false
	for _, dispatch := range dispatches {
		if dispatch.Caste != "gatekeeper" {
			continue
		}
		found = true
		if !strings.Contains(dispatch.Rationale, "this touches") {
			t.Errorf("gatekeeper dispatch should state why it was forced, got rationale %q", dispatch.Rationale)
		}
	}
	if !found {
		t.Fatalf("gatekeeper missing from continue dispatch list: %+v", dispatches)
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
		[]string{"builder", "watcher", "wizard", "test-writer"},
		"", phase, "build", colony.ColonyState{})

	// "security" is deliberately NOT used here: it now resolves to gatekeeper
	// via casteNameAliases, because a near-miss synonym should land rather than
	// vanish. Only genuinely unrecognisable names reach Unknown.
	if len(judgement.Unknown) != 2 {
		t.Fatalf("unknown = %v, want both unrecognisable names", judgement.Unknown)
	}
	if !strings.Contains(judgement.Summary(), "Ignored unknown caste(s)") {
		t.Errorf("summary must disclose unknown castes, got: %s", judgement.Summary())
	}
	for _, bogus := range []string{"wizard", "test-writer"} {
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
		if strings.TrimSpace(entry["produces"]) == "" {
			t.Errorf("roster entry %q has no description of what it produces", entry["caste"])
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
	// Phase 193 (D-08) stopped the build's own verification-stage dispatch
	// from firing without an explicit Queen proposal naming the watcher --
	// it did not here, so no build-side watcher spawns. Plan 194-02 (D-07)
	// went further and removed watcher from the required-castes floor
	// entirely, so there is no longer a restoration path to prove here
	// either. Agent review for an unrequested watcher lives in `continue`,
	// not the build boundary (ruling D11 rule 4: a phase is verified once).
	if spawned["watcher"] {
		t.Errorf("watcher must not spawn at the build boundary without an explicit Queen proposal; castes = %v", casteKeys(spawned))
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

// TestQueenTrimsContinueReviewersAndKeepsTheWatcher covers the expensive flow.
// Continue fans out to a reviewer per caste, each a full agent run — a real
// session spent roughly 350k tokens on three of them, one of which was a
// Measurer selected because an audio phase's vocabulary contains "latency" and
// "memory". Nobody judged that worth doing; a word matched.
func TestQueenTrimsContinueReviewersAndKeepsTheWatcher(t *testing.T) {
	phase := judgementPhase(
		"Fix envelope retrigger",
		"The envelope re-arms once and never again when the step lane sends a steady value; latency and memory behaviour is unchanged",
		colony.PhaseModeProduction,
	)
	depth := colony.VerificationDepthStandard

	// Keyword scoring on this phase pulls in specialists the change does not
	// need. The Queen reading it knows the question is correctness.
	before := queenContinueDispatches(phase, depth)
	after := queenContinueDispatchesWithJudgement(phase, depth, []string{"watcher"}, "this is a correctness fix, not a performance question", nil)

	if len(after) > len(before) {
		t.Errorf("judgement should not grow the review team here: before %d, after %d", len(before), len(after))
	}
	if !queenContinueHasCaste(after, "watcher") {
		t.Errorf("Watcher must survive: %v", casteNames(after))
	}
}

// TestContinueJudgementCannotDropASecurityReview keeps the continue floor equal
// to the build floor. Trimming reviewers is a cost decision; skipping a
// security review on credential work is not available at any cost.
func TestContinueJudgementCannotDropASecurityReview(t *testing.T) {
	phase := judgementPhase("Password reset", "Rotate the emailed reset token", colony.PhaseModeProduction)

	after := queenContinueDispatchesWithJudgement(
		phase, colony.VerificationDepthStandard,
		[]string{"watcher"}, "looks simple", nil)

	if !queenContinueHasCaste(after, "watcher") {
		t.Errorf("Watcher must survive: %v", casteNames(after))
	}
	required := stringSet(queenSpawnBudgetForPhase(phase, "continue", colony.ColonyState{}).RequiredCastes)
	for caste := range required {
		if !queenContinueHasCaste(after, caste) {
			t.Errorf("required continue caste %s was dropped: %v", caste, casteNames(after))
		}
	}
}

// TestContinueWithNoProposalIsUnchanged keeps the change additive.
func TestContinueWithNoProposalIsUnchanged(t *testing.T) {
	phase := judgementPhase("Add CSV export", "Download the table", colony.PhaseModeProduction)
	depth := colony.VerificationDepthStandard

	base := casteNames(queenContinueDispatches(phase, depth))
	same := casteNames(queenContinueDispatchesWithJudgement(phase, depth, nil, "", nil))
	if len(base) != len(same) {
		t.Fatalf("no-proposal continue changed: %v vs %v", base, same)
	}
	for _, caste := range base {
		if !hasCasteName(same, caste) {
			t.Errorf("no-proposal continue dropped %s", caste)
		}
	}
}

// TestEveryCasteHasAWrittenCapability closes the gap that made the roster
// useless. queenCasteRoster built its descriptions from
// strings.Join(profile.Keywords, ", "), so the judgement layer built to
// out-reason keyword scoring was handed the keyword table as its worldview.
// A caste with no written capability silently falls back to that, so this
// fails rather than shipping the fallback.
func TestEveryCasteHasAWrittenCapability(t *testing.T) {
	for _, profile := range casteRelevanceRegistry {
		capability, ok := casteCapabilities[profile.Caste]
		if !ok {
			t.Errorf("caste %q has no written capability; the roster would fall back to its keyword list", profile.Caste)
			continue
		}
		if strings.TrimSpace(capability.Produces) == "" {
			t.Errorf("caste %q has an empty Produces description", profile.Caste)
		}
		// The keyword list must not be the description. That is the bug.
		for _, keyword := range profile.Keywords {
			if strings.EqualFold(strings.TrimSpace(capability.Produces), strings.Join(profile.Keywords, ", ")) {
				t.Errorf("caste %q describes itself with its keyword list (%q)", profile.Caste, keyword)
				break
			}
		}
	}
}

// TestRosterNamesWhenNotToSpawn pins the half that makes disagreement possible.
// A description saying only what a caste is good at cannot help a reader decide
// against it; the anti-goal is what lets the Queen conclude a phase saying
// "latency is unchanged" does not need a Measurer.
func TestRosterNamesWhenNotToSpawn(t *testing.T) {
	roster := queenCasteRoster()
	withAntiGoal := 0
	for _, entry := range roster {
		if strings.TrimSpace(entry["avoid_for"]) != "" {
			withAntiGoal++
		}
		if strings.TrimSpace(entry["produces"]) == "" {
			t.Errorf("roster entry %q has no produces description", entry["caste"])
		}
	}
	if withAntiGoal < len(roster)-1 {
		t.Errorf("only %d of %d roster entries say when not to spawn; the anti-goal is what makes disagreeing with the keywords possible",
			withAntiGoal, len(roster))
	}

	// The measurer entry is the one the real failure turned on.
	for _, entry := range roster {
		if entry["caste"] != "measurer" {
			continue
		}
		if !strings.Contains(strings.ToLower(entry["avoid_for"]), "unchanged") {
			t.Errorf("measurer's anti-goal must name the stated-unchanged case that cost 111.8k tokens, got: %q", entry["avoid_for"])
		}
	}
}

// TestCasteNameResolutionSurvivesSeparatorAndSynonym covers the silent-drop
// bug. The registry mixes separators (route_setter, surveyor-nest), so a model
// writing "route-setter" was reported unknown and the phase ran without it.
func TestCasteNameResolutionSurvivesSeparatorAndSynonym(t *testing.T) {
	for _, tc := range []struct{ proposed, want string }{
		{"route-setter", "route_setter"},
		{"route_setter", "route_setter"},
		{"surveyor_nest", "surveyor-nest"},
		{"surveyor-nest", "surveyor-nest"},
		{"security", "gatekeeper"},
		{"performance", "measurer"},
		{"verifier", "watcher"},
		{"MEASURER", "measurer"},
	} {
		known, unknown := normalizeProposedCastes([]string{tc.proposed})
		if len(unknown) > 0 {
			t.Errorf("%q was reported unknown, want it to resolve to %q", tc.proposed, tc.want)
			continue
		}
		if len(known) != 1 || known[0] != tc.want {
			t.Errorf("%q resolved to %v, want [%s]", tc.proposed, known, tc.want)
		}
	}

	// A genuinely unrecognisable name must still surface rather than resolve to
	// something plausible-looking.
	_, unknown := normalizeProposedCastes([]string{"wizard"})
	if len(unknown) != 1 {
		t.Errorf("an unrecognisable caste must still be reported, got %v", unknown)
	}
}
