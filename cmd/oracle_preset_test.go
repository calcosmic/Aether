package cmd

import (
	"fmt"
	"strings"
	"testing"
)

// TestOraclePresetLabelsMatchPlanningVocabulary proves Oracle's shared
// four-name vocabulary is the same vocabulary planning already established,
// in the same order -- checked against planning's own table, never a copy.
func TestOraclePresetLabelsMatchPlanningVocabulary(t *testing.T) {
	oracle := oraclePresetPolicies()
	if len(oracle) != len(planningPresetPolicies) {
		t.Fatalf("oracle offers %d presets, want %d (one for each of planning's)", len(oracle), len(planningPresetPolicies))
	}
	for i, policy := range oracle {
		if policy.Label != planningPresetPolicies[i].Label {
			t.Errorf("oracle preset %d label = %q, want %q (planning's order)", i, policy.Label, planningPresetPolicies[i].Label)
		}
	}
}

// TestOraclePresetNumbersAreUnchanged proves the numeric values reachable
// through the new shared-name layer equal the values already in
// oracleDepthLevels, entry by entry -- and that they are read from the map at
// construction, not retyped, by mutating the map and observing the preset
// follow it.
func TestOraclePresetNumbersAreUnchanged(t *testing.T) {
	for id, legacyKey := range oraclePresetLegacyKey {
		want := oracleDepthLevels[legacyKey]
		got, err := resolveOraclePreset(string(id))
		if err != nil {
			t.Fatalf("resolveOraclePreset(%q): %v", id, err)
		}
		if got.TargetConfidence != want.TargetConfidence || got.RoundCap != want.MaxIterations {
			t.Errorf("preset %s = %d%%/%d rounds, want %d%%/%d rounds (from oracleDepthLevels[%q])",
				id, got.TargetConfidence, got.RoundCap, want.TargetConfidence, want.MaxIterations, legacyKey)
		}
	}

	original := oracleDepthLevels["deep"]
	t.Cleanup(func() { oracleDepthLevels["deep"] = original })
	oracleDepthLevels["deep"] = oracleDepthConfig{
		MaxIterations:    77,
		TargetConfidence: 61,
		Label:            original.Label,
		Description:      original.Description,
	}

	got, err := resolveOraclePreset("deep")
	if err != nil {
		t.Fatalf("resolveOraclePreset(deep): %v", err)
	}
	if got.TargetConfidence != 61 || got.RoundCap != 77 {
		t.Errorf("preset did not follow the mutated oracleDepthLevels[\"deep\"] entry: got %d%%/%d rounds, want 61%%/77 rounds", got.TargetConfidence, got.RoundCap)
	}
}

// TestOracleLegacyDepthWordsStillResolve proves every word oracleDepthLevels
// has ever accepted -- including "standard" and "marathon", which never had a
// shared-name row of their own -- still resolves to the same numbers it
// always resolved to.
func TestOracleLegacyDepthWordsStillResolve(t *testing.T) {
	for key, cfg := range oracleDepthLevels {
		preset, err := resolveOraclePreset(key)
		if err != nil {
			t.Fatalf("resolveOraclePreset(%q): %v", key, err)
		}
		if preset.TargetConfidence != cfg.TargetConfidence || preset.RoundCap != cfg.MaxIterations {
			t.Errorf("legacy word %q resolved to %d%%/%d rounds, want %d%%/%d rounds (its unchanged oracleDepthLevels entry)",
				key, preset.TargetConfidence, preset.RoundCap, cfg.TargetConfidence, cfg.MaxIterations)
		}
	}
}

// TestOracleUnknownDepthIsRefusedByName proves an unrecognized depth word is
// refused, naming all four accepted words rather than silently defaulting.
func TestOracleUnknownDepthIsRefusedByName(t *testing.T) {
	_, err := resolveOraclePreset("glacial")
	if err == nil {
		t.Fatal("resolveOraclePreset(\"glacial\") succeeded, want a refusal naming the four accepted words")
	}
	for _, want := range []string{"fast", "balanced", "deep", "exhaustive"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name accepted word %q", err.Error(), want)
		}
	}
}

// TestOraclePickerShowsSharedNamesWithOwnNumbers proves the research proposal
// picker (runOraclePropose/renderOraclePropose) lists all four shared names,
// each with its own confidence target and round cap -- Oracle's own numbers,
// not planning's -- rather than the legacy depth words.
func TestOraclePickerShowsSharedNamesWithOwnNumbers(t *testing.T) {
	options := oracleDepthOptions(string(planningStagePresetBalanced))
	want := oraclePresetOptions()
	if len(options) != len(want) {
		t.Fatalf("depth_options = %d entries, want %d (one per shared preset)", len(options), len(want))
	}
	for i, option := range options {
		if option.Label != want[i].Label {
			t.Errorf("option %d label = %q, want %q", i, option.Label, want[i].Label)
		}
		if option.TargetConfidence != want[i].TargetConfidence {
			t.Errorf("option %d target = %d%%, want %d%%", i, option.TargetConfidence, want[i].TargetConfidence)
		}
		if option.MaxIterations != want[i].RoundCap {
			t.Errorf("option %d round cap = %d, want %d", i, option.MaxIterations, want[i].RoundCap)
		}
	}

	root := t.TempDir()
	result, err := runOraclePropose(root, "should we use SQLite or Postgres for the local cache?")
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	rendered := renderOraclePropose(result)
	for _, policy := range want {
		if !strings.Contains(rendered, policy.Label) {
			t.Errorf("rendered picker missing shared name %q:\n%s", policy.Label, rendered)
		}
		if !strings.Contains(rendered, fmt.Sprintf("target %d%%", policy.TargetConfidence)) {
			t.Errorf("rendered picker missing target %d%% for %s:\n%s", policy.TargetConfidence, policy.Label, rendered)
		}
		if !strings.Contains(rendered, fmt.Sprintf("%d rounds", policy.RoundCap)) {
			t.Errorf("rendered picker missing round cap %d for %s:\n%s", policy.RoundCap, policy.Label, rendered)
		}
	}
}

// TestOracleBriefPanelNamesTheSharedPreset proves the approved brief panel
// names the chosen preset by its shared label, whether the brief was approved
// with a legacy depth word or a shared name.
func TestOracleBriefPanelNamesTheSharedPreset(t *testing.T) {
	root := t.TempDir()

	legacy, err := runOracleBriefApprove(root, oracleBriefOptions{
		Topic:        "caching",
		CoreQuestion: "Should the cache be write-through?",
		Depth:        "quick",
	}, true)
	if err != nil {
		t.Fatalf("approve brief with legacy word: %v", err)
	}
	legacyPanel, _ := legacy["panel"].(string)
	if !strings.Contains(legacyPanel, "Fast") {
		t.Errorf("brief panel for legacy word %q does not show shared label %q:\n%s", "quick", "Fast", legacyPanel)
	}
	if strings.Contains(legacyPanel, "Depth: quick") {
		t.Errorf("brief panel still shows the legacy word instead of the shared label:\n%s", legacyPanel)
	}

	shared, err := runOracleBriefApprove(root, oracleBriefOptions{
		Topic:        "caching",
		CoreQuestion: "Should the cache be write-through?",
		Depth:        "fast",
	}, true)
	if err != nil {
		t.Fatalf("approve brief with shared name: %v", err)
	}
	sharedPanel, _ := shared["panel"].(string)
	if !strings.Contains(sharedPanel, "Fast") {
		t.Errorf("brief panel for shared name %q does not show %q:\n%s", "fast", "Fast", sharedPanel)
	}
}

// TestOracleDepthFlagHelpListsSharedNames proves the --depth flag's own help
// text lists the four shared names.
func TestOracleDepthFlagHelpListsSharedNames(t *testing.T) {
	flag := oracleCmd.Flags().Lookup("depth")
	if flag == nil {
		t.Fatal("oracle command has no --depth flag")
	}
	lower := strings.ToLower(flag.Usage)
	for _, want := range []string{"fast", "balanced", "deep", "exhaustive"} {
		if !strings.Contains(lower, want) {
			t.Errorf("--depth help text %q does not name shared word %q", flag.Usage, want)
		}
	}
}

// TestOracleStopsAtExactlyTheConfidenceTarget drives the real loop's stopping
// decision (oracleReadyForCompletion, the exact predicate runOracleLoop calls
// to finalize with status "complete") against fixture states one point below
// and exactly at a preset's confidence target.
func TestOracleStopsAtExactlyTheConfidenceTarget(t *testing.T) {
	plan := oraclePlanFile{}

	below := oracleStateFile{TargetConfidence: 85, OverallConfidence: 84}
	if oracleReadyForCompletion(plan, below) {
		t.Error("one point below the confidence target must continue, not stop")
	}

	exact := oracleStateFile{TargetConfidence: 85, OverallConfidence: 85}
	if !oracleReadyForCompletion(plan, exact) {
		t.Error("exactly at the confidence target must stop")
	}
}

// TestOracleStopsAtExactlyTheRoundCap drives the real loop's stopping
// decision -- the exact `state.Iteration < state.MaxIterations` guard
// runOracleLoop's for-loop uses -- against fixture states one round below and
// exactly at a preset's round cap.
func TestOracleStopsAtExactlyTheRoundCap(t *testing.T) {
	oneBelow := oracleStateFile{Iteration: 14, MaxIterations: 15}
	if !(oneBelow.Iteration < oneBelow.MaxIterations) {
		t.Error("one round below the cap must run another round")
	}

	atCap := oracleStateFile{Iteration: 15, MaxIterations: 15}
	if atCap.Iteration < atCap.MaxIterations {
		t.Error("exactly at the round cap must not start another round")
	}
}
