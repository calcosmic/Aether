package cmd

import (
	"fmt"
	"strings"
)

// Oracle speaks planning's shared four research-depth names -- Fast,
// Balanced, Deep and Exhaustive -- laid over its own unchanged depth
// economics. Planning's numbers (80/4, 90/6, 95/8, 99/12) and Oracle's
// numbers (60/5, 85/15, 95/30, 99/50) are allowed to differ; only the label
// layer is shared, per SYN-202-10 in 202-CLASSIC-SYNTHESIS.md. Rewriting
// Oracle's targets to match planning's would be a real regression to fix
// what is only a display inconsistency (RESEARCH.md Pitfall 4).

// oraclePresetPolicy is one shared-vocabulary entry over Oracle's own depth
// machinery: a shared identifier and label, paired with Oracle's own
// confidence target and round cap -- read from oracleDepthLevels, never
// retyped, so the two cannot drift.
type oraclePresetPolicy struct {
	ID               planningStagePreset `json:"id"`
	Label            string              `json:"label"`
	TargetConfidence int                 `json:"target"`
	RoundCap         int                 `json:"round_cap"`
}

// oraclePresetOrder keeps the shared vocabulary in the same cheapest-first
// order planning renders it in.
var oraclePresetOrder = []planningStagePreset{
	planningStagePresetFast,
	planningStagePresetBalanced,
	planningStagePresetDeep,
	planningStagePresetExhaustive,
}

// oraclePresetLabels are the four shared names, identical to planning's.
var oraclePresetLabels = map[planningStagePreset]string{
	planningStagePresetFast:       "Fast",
	planningStagePresetBalanced:   "Balanced",
	planningStagePresetDeep:       "Deep",
	planningStagePresetExhaustive: "Exhaustive",
}

// oraclePresetLegacyKey maps each shared preset onto the oracleDepthLevels key
// that has always carried its numbers, so a later change to Oracle's own
// economics flows through automatically without touching this file.
var oraclePresetLegacyKey = map[planningStagePreset]string{
	planningStagePresetFast:       "quick",
	planningStagePresetBalanced:   "balanced",
	planningStagePresetDeep:       "deep",
	planningStagePresetExhaustive: "exhaustive",
}

// oraclePresetAliasKey maps every previously accepted oracleDepthLevels key
// onto the shared preset it has always resolved to -- including "standard"
// and "marathon", which never had a row of their own -- so old input keeps
// working unchanged.
var oraclePresetAliasKey = map[string]planningStagePreset{
	"quick":      planningStagePresetFast,
	"balanced":   planningStagePresetBalanced,
	"standard":   planningStagePresetBalanced,
	"deep":       planningStagePresetDeep,
	"exhaustive": planningStagePresetExhaustive,
	"marathon":   planningStagePresetExhaustive,
}

// oraclePresetPolicies builds the four shared-vocabulary entries fresh from
// oracleDepthLevels on every call -- never a value copied in once at init --
// so a later change to Oracle's own numbers is observed immediately,
// including by a test that mutates the map directly.
func oraclePresetPolicies() []oraclePresetPolicy {
	policies := make([]oraclePresetPolicy, 0, len(oraclePresetOrder))
	for _, id := range oraclePresetOrder {
		cfg := oracleDepthLevels[oraclePresetLegacyKey[id]]
		policies = append(policies, oraclePresetPolicy{
			ID:               id,
			Label:            oraclePresetLabels[id],
			TargetConfidence: cfg.TargetConfidence,
			RoundCap:         cfg.MaxIterations,
		})
	}
	return policies
}

// oraclePresetOptions returns the four shared presets in picker order, each
// carrying its own confidence target and round cap for display.
func oraclePresetOptions() []oraclePresetPolicy {
	return oraclePresetPolicies()
}

// resolveOraclePreset accepts both the four shared names and every word
// oracleDepthLevels has always accepted, mapping the cheapest legacy word
// ("quick") and the standard word ("standard"/"marathon") onto their shared
// names. Anything else is refused by name, listing the four accepted words.
func resolveOraclePreset(value string) (oraclePresetPolicy, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	policies := oraclePresetPolicies()
	for _, policy := range policies {
		if string(policy.ID) == normalized {
			return policy, nil
		}
	}
	if shared, ok := oraclePresetAliasKey[normalized]; ok {
		for _, policy := range policies {
			if policy.ID == shared {
				return policy, nil
			}
		}
	}
	return oraclePresetPolicy{}, fmt.Errorf("invalid research depth %q: choose fast, balanced, deep, or exhaustive", value)
}
