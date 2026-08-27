package cmd

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestNoWorkerWithoutStatedReason is D-08's central claim: a proposal of
// several castes with a reason for only some of them sends the ones that have
// one and refuses the rest BY NAME, while the rest of the proposal is
// unaffected. Asserted on the DISPATCH list (plannedBuildDispatchesWithJudgement),
// never the judgement struct alone -- TestQueenChoiceReachesTheDispatchList's
// own lesson (a decision recorded but never reaching a worker reads as
// working when it is not).
func TestNoWorkerWithoutStatedReason(t *testing.T) {
	phase := colony.Phase{
		ID:          1,
		Name:        "Dashboard feels sluggish",
		Description: "Users report the main list takes a long time to appear with many rows",
		Mode:        colony.PhaseModeProduction,
		Status:      colony.PhaseReady,
		Tasks:       []colony.Task{{Goal: "Speed up the list rendering"}},
	}
	state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{phase}}}

	judgement := queenApplyJudgement(
		[]string{"measurer", "chaos"},
		"",
		phase, "build", colony.ColonyState{},
		map[string]string{"measurer": "the complaint is latency even though the phase never says so"},
	)
	if !hasCasteName(judgement.Final, "measurer") {
		t.Errorf("measurer had a stated reason and must be in Final: %v", judgement.Final)
	}
	if hasCasteName(judgement.Final, "chaos") {
		t.Errorf("chaos had no stated reason and must not be in Final: %v", judgement.Final)
	}
	if len(judgement.RefusedNoReason) != 1 || judgement.RefusedNoReason[0] != "chaos" {
		t.Fatalf("RefusedNoReason = %v, want [chaos]", judgement.RefusedNoReason)
	}
	if !strings.Contains(judgement.Summary(), "chaos") {
		t.Errorf("summary must name the worker refused for arriving without a reason: %q", judgement.Summary())
	}

	// The claim that matters is on the dispatch list, not the judgement.
	dispatches := testPlannedBuildDispatchesWithJudgement(
		phase, state, nil, colony.VerificationDepthStandard,
		[]string{"measurer", "chaos"}, "",
		map[string]string{"measurer": "the complaint is latency even though the phase never says so"},
	)
	spawned := map[string]bool{}
	for _, d := range dispatches {
		spawned[d.Caste] = true
	}
	if !spawned["measurer"] {
		t.Errorf("measurer had a stated reason and must spawn; castes = %v", casteKeys(spawned))
	}
	if spawned["chaos"] {
		t.Errorf("chaos had no stated reason and must not spawn; castes = %v", casteKeys(spawned))
	}
}

// TestTeamLevelReasonDoesNotSatisfyPerWorkerReason pins D-08's other half:
// --caste-reason is the team summary, never a substitute for a per-worker
// --caste-why entry. A caste proposed alongside only a team-level string is
// refused exactly as if no reason had been supplied at all, and the team
// string survives only as the Rationale/summary.
func TestTeamLevelReasonDoesNotSatisfyPerWorkerReason(t *testing.T) {
	phase := judgementPhase("Tidy the helpers", "Small cleanup of utility functions", colony.PhaseModeMaintenance)

	// chaos has a non-zero base score on every phase (BaseScore 15, not
	// keyword-gated) so this proves the NEW no-reason refusal fires, not the
	// pre-existing zero-relevance one -- a caste that genuinely has
	// something to do still needs its own stated reason.
	if score := casteRelevanceScore(phase, "chaos"); score == 0 {
		t.Fatalf("precondition: chaos must score above zero on this phase for the test to isolate the right rule, got 0")
	}

	judgement := queenApplyJudgement(
		[]string{"chaos"},
		"one line for the whole team",
		phase, "build", colony.ColonyState{},
	)

	if hasCasteName(judgement.Final, "chaos") {
		t.Errorf("a team-level reason alone must not satisfy the per-worker requirement; final = %v", judgement.Final)
	}
	if len(judgement.RefusedNoReason) != 1 || judgement.RefusedNoReason[0] != "chaos" {
		t.Fatalf("RefusedNoReason = %v, want [chaos]", judgement.RefusedNoReason)
	}
	if judgement.Rationale != "one line for the whole team" {
		t.Errorf("the team-level string must survive as Rationale/summary, got %q", judgement.Rationale)
	}
}

// TestReasonKeyResolvesThroughTheSameAliasPath is the encoding probe
// (194-CONTEXT.md TEAM-03): a --caste-why key written with the other
// separator style, or a synonym, must resolve to the same registry caste
// --castes itself resolves it to -- not vanish into a second, unrelated
// unknown-name channel.
func TestReasonKeyResolvesThroughTheSameAliasPath(t *testing.T) {
	reasons, unknown := parseCasteReasonPairs([]string{
		"route-setter=explains why the plan needed re-breaking-down",
		"security=this changes login and password handling",
	})
	if len(unknown) != 0 {
		t.Fatalf("unknown = %v, want none — both keys resolve via resolveCasteName", unknown)
	}
	if got := reasons["route_setter"]; got == "" {
		t.Errorf("route-setter must resolve to route_setter, reasons = %v", reasons)
	}
	if got := reasons["gatekeeper"]; got == "" {
		t.Errorf("security must resolve to gatekeeper (casteNameAliases), reasons = %v", reasons)
	}

	// A genuinely unresolvable key is reported, not silently dropped -- the
	// SAME unknown-name channel --castes already uses (parseAndMergeCasteWhy
	// folds it into the proposal list so normalizeProposedCastes reports it).
	_, unknownAgain := parseCasteReasonPairs([]string{"wizard=a reason for a caste that does not exist"})
	if len(unknownAgain) != 1 || unknownAgain[0] != "wizard" {
		t.Errorf("an unresolvable caste key must be reported, got %v", unknownAgain)
	}

	mergedProposed, mergedReasons := parseAndMergeCasteWhy([]string{"builder"}, []string{"wizard=a reason for a caste that does not exist"})
	_, unknownFromMerge := normalizeProposedCastes(mergedProposed)
	if len(unknownFromMerge) != 1 || unknownFromMerge[0] != "wizard" {
		t.Errorf("an unresolvable --caste-why key must surface through the same Unknown channel as --castes, got %v", unknownFromMerge)
	}
	if len(mergedReasons) != 0 {
		t.Errorf("an unresolvable key must not appear as a reason, got %v", mergedReasons)
	}
}

// TestReasonOutputIsStablySorted is the ordering probe (194-CONTEXT.md
// TEAM-03): two runs over the same input must produce byte-identical
// RefusedNoReason slices and summary strings.
func TestReasonOutputIsStablySorted(t *testing.T) {
	phase := judgementPhase("Tidy the helpers", "Small cleanup of utility functions", colony.PhaseModeMaintenance)
	proposed := []string{"weaver", "sage", "keeper", "chaos", "includer"}

	first := queenApplyJudgement(proposed, "", phase, "build", colony.ColonyState{})
	second := queenApplyJudgement(proposed, "", phase, "build", colony.ColonyState{})

	if !sort.StringsAreSorted(first.RefusedNoReason) {
		t.Fatalf("RefusedNoReason is not sorted: %v", first.RefusedNoReason)
	}
	if strings.Join(first.RefusedNoReason, ",") != strings.Join(second.RefusedNoReason, ",") {
		t.Fatalf("two runs over the same input produced different RefusedNoReason: %v vs %v", first.RefusedNoReason, second.RefusedNoReason)
	}
	if first.Summary() != second.Summary() {
		t.Fatalf("two runs over the same input produced different summaries:\n%q\nvs\n%q", first.Summary(), second.Summary())
	}
}

// TestRuntimeAddedWorkersCarryRuntimeWrittenReasons (D-09): a caste the
// runtime adds itself -- because the phase requires it, or a named risk
// signal forced it -- carries a reason the RUNTIME wrote, not an empty slot.
// The implementation worker names the task count and a task goal; a forced
// reviewer's reason is the sentence 194-01 already produces
// (forcedReviewerReason), naming the signal and quoting the matched phrase.
func TestRuntimeAddedWorkersCarryRuntimeWrittenReasons(t *testing.T) {
	phase := colony.Phase{
		ID:          1,
		Name:        "Password reset",
		Description: "Let users reset their password via an emailed token",
		Mode:        colony.PhaseModeProduction,
		Status:      colony.PhaseReady,
		Tasks: []colony.Task{
			{Goal: "Add the reset-request endpoint"},
			{Goal: "Send the reset email"},
		},
	}

	judgement := queenApplyJudgement(nil, "", phase, "continue", colony.ColonyState{})

	buildJudgement := queenApplyJudgement(nil, "", phase, "build", colony.ColonyState{})
	builderReason := buildJudgement.Reasons["builder"]
	if !strings.Contains(builderReason, "2 task") {
		t.Errorf("builder's reason must name the task count (\"2 task...\"), got %q", builderReason)
	}

	gatekeeperReason := judgement.Reasons["gatekeeper"]
	if !strings.Contains(gatekeeperReason, "password reset") {
		t.Errorf("a forced security reviewer's reason must quote the matched phrase \"password reset\", got %q", gatekeeperReason)
	}
	if !strings.Contains(gatekeeperReason, "this touches") {
		t.Errorf("a forced security reviewer's reason must be the forcedReviewerReason sentence, got %q", gatekeeperReason)
	}
}

// forbiddenReasonShape is D-09's guard: no reason produced anywhere may
// contain a digit-threshold comparison or the caste's own name standing alone
// as its justification. Expressed as a regexp over the string rather than a
// fixed-literal check, so a future rewording of the score-arithmetic text
// cannot slip past it.
var forbiddenReasonShape = regexp.MustCompile(`[0-9]+\s*>=`)

// TestFallbackPickCarriesASentenceNotAScore is the guard that matters: every
// caste the fallback (no-proposal) engine can select carries a plain sentence
// about what in THIS phase called for it, never "Score 40 >= threshold 30".
func TestFallbackPickCarriesASentenceNotAScore(t *testing.T) {
	fixtures := map[string]colony.Phase{
		"builder": {ID: 1, Name: "Add CSV export", Description: "Implement a CSV export button", Mode: colony.PhaseModePrototype,
			Tasks: []colony.Task{{Goal: "Implement the export endpoint"}}},
		"scout": {ID: 1, Name: "Investigate caching options", Description: "Research and survey available caching approaches", Mode: colony.PhaseModeDiscovery},
		"tracker": {ID: 1, Name: "Fix the regression", Description: "Investigate failure root cause of the checkout bug",
			Tasks: []colony.Task{{Goal: "Find and fix the regression"}}},
		"chronicler": {ID: 1, Name: "Write the guide", Description: "Document the new API in a readme and changelog"},
		"measurer":   {ID: 1, Name: "Speed up the dashboard", Description: "Benchmark and optimize latency and memory use"},
		"gatekeeper": {ID: 1, Name: "Rotate API keys", Description: "Review auth and secrets handling for compliance"},
	}

	for _, profile := range casteRelevanceRegistry {
		phase, ok := fixtures[profile.Caste]
		if !ok {
			continue
		}
		dispatches := queenCandidateDispatches(phase, "build", colony.ColonyState{})
		for _, dispatch := range dispatches {
			if dispatch.Caste != profile.Caste {
				continue
			}
			if forbiddenReasonShape.MatchString(dispatch.Rationale) {
				t.Errorf("%s: rationale contains score arithmetic: %q", profile.Caste, dispatch.Rationale)
			}
			if strings.EqualFold(strings.TrimSpace(dispatch.Rationale), profile.Caste) {
				t.Errorf("%s: rationale is just the caste's own name: %q", profile.Caste, dispatch.Rationale)
			}
			if strings.Contains(dispatch.Rationale, "always required for") {
				t.Errorf("%s: rationale still uses the deleted 'always required for FLOW flow' phrasing: %q", profile.Caste, dispatch.Rationale)
			}
		}
	}
}

// TestCasteDecisionManifestCarriesReasonsAndRefusals is the manifest-facing
// half of the acceptance criteria: a --plan-only build's caste_decision
// (queenCasteDecisionSummary is what populates manifest.CasteDecision, see
// cmd/codex_build.go) carries both a `reasons` object and a
// `refused_no_reason` array when a proposed caste arrives without one.
func TestCasteDecisionManifestCarriesReasonsAndRefusals(t *testing.T) {
	phase := colony.Phase{
		ID:          1,
		Name:        "Add CSV export",
		Description: "Let users export the table as CSV",
		Mode:        colony.PhaseModePrototype,
		Status:      colony.PhaseReady,
		Tasks:       []colony.Task{{Goal: "Implement the export endpoint"}},
	}

	decision := queenCasteDecisionSummary(
		phase, colony.ColonyState{}, colony.VerificationDepthStandard,
		[]string{"measurer", "chaos"}, "",
		map[string]string{"measurer": "the export needs to stay fast on large tables"},
	)

	reasons, ok := decision["reasons"].(map[string]string)
	if !ok || reasons["measurer"] == "" {
		t.Fatalf("caste_decision missing a reasons object with measurer's reason: %+v", decision)
	}
	refused, ok := decision["refused_no_reason"].([]string)
	if !ok || len(refused) != 1 || refused[0] != "chaos" {
		t.Fatalf("caste_decision missing refused_no_reason = [chaos]: %+v", decision)
	}
}

// TestDeletedImplicitFloorsNoLongerScoreToTheMaximum is the acceptance
// criterion behind the applySpecialRules deletion: casteRelevanceScore for
// the quality reviewer on a production-mode phase with no matching keyword
// must fall below the spawn threshold, not sit at the old 100 maximum -- and
// the same must hold for the security reviewer on a high-risk phase with no
// signal wording.
func TestDeletedImplicitFloorsNoLongerScoreToTheMaximum(t *testing.T) {
	productionNoKeyword := colony.Phase{
		ID:          1,
		Name:        "Ship the button copy change",
		Description: "Change the button copy from Submit to Save",
		Mode:        colony.PhaseModeProduction,
		Tasks:       []colony.Task{{Goal: "Update the button label"}},
	}
	if score := casteRelevanceScore(productionNoKeyword, "auditor"); score >= spawnThreshold("build", colony.ColonyState{}) {
		t.Errorf("auditor scored %d on a production phase with no auditor wording -- want below the spawn threshold, not the old maximum", score)
	}

	highRiskNoSignal := colony.Phase{
		ID:          1,
		Name:        "Rework internal role checks",
		Description: "Restructure how role-based checks are organised internally",
		Mode:        colony.PhaseModeProduction,
		Tasks:       []colony.Task{{Goal: "Restructure the internal role checks"}},
	}
	if score := casteRelevanceScore(highRiskNoSignal, "gatekeeper"); score >= spawnThreshold("build", colony.ColonyState{}) {
		t.Errorf("gatekeeper scored %d on a high-risk phase with no security wording -- want below the spawn threshold, not the old maximum", score)
	}
}
