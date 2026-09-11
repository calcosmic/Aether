package cmd

import (
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
