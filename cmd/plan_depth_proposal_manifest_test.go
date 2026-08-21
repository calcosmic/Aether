package cmd

import (
	"strings"
	"testing"
)

// TestPlanManifestCarriesDepthProposal covers Plan 09 Task 1: both plan-only
// result branches (existing-plan early return and fresh-plan) carry the
// three-knob depth proposal and its rendered card, the proposal's
// recommended values match what the result already reports for granularity
// and both depths, the card's accept line carries those same three values,
// an explicitly-supplied depth reports an explicit reason rather than a
// smart-default one, and the plan_manifest carries the proposal too.
func TestPlanManifestCarriesDepthProposal(t *testing.T) {
	t.Run("fresh_plan_branch_carries_proposal_and_card", func(t *testing.T) {
		saveGlobals(t)
		setupPhaseResearchManifestTest(t, researchProposalTestPhases())

		result := runPlanOnly(t, "--refresh", "--depth", "balanced")

		proposal, ok := result["depth_proposal"].(map[string]interface{})
		if !ok {
			t.Fatalf("depth_proposal missing or wrong type: %+v", result["depth_proposal"])
		}
		knobs, ok := proposal["knobs"].([]interface{})
		if !ok || len(knobs) != 3 {
			t.Fatalf("depth_proposal.knobs = %v, want 3 knobs", proposal["knobs"])
		}
		card, _ := result["depth_proposal_card"].(string)
		if card == "" {
			t.Fatal("depth_proposal_card is empty")
		}

		manifest, ok := result["plan_manifest"].(map[string]interface{})
		if !ok {
			t.Fatalf("plan_manifest missing from result: %+v", result)
		}
		if _, ok := manifest["depth_proposal"]; !ok {
			t.Fatalf("manifest missing depth_proposal: %+v", manifest)
		}
		if manifestCard, _ := manifest["depth_proposal_card"].(string); manifestCard == "" {
			t.Fatalf("manifest depth_proposal_card is empty")
		}
	})

	t.Run("existing_plan_early_return_branch_carries_proposal_and_card", func(t *testing.T) {
		saveGlobals(t)
		setupPhaseResearchManifestTest(t, researchProposalTestPhases())

		result := runPlanOnly(t)
		if existing, _ := result["existing_plan"].(bool); !existing {
			t.Fatalf("existing_plan = %v, want true (early-return branch)", result["existing_plan"])
		}
		proposal, ok := result["depth_proposal"].(map[string]interface{})
		if !ok {
			t.Fatalf("depth_proposal missing in early-return branch: %+v", result["depth_proposal"])
		}
		knobs, ok := proposal["knobs"].([]interface{})
		if !ok || len(knobs) != 3 {
			t.Fatalf("depth_proposal.knobs = %v, want 3 knobs in early-return branch", proposal["knobs"])
		}
		card, _ := result["depth_proposal_card"].(string)
		if card == "" {
			t.Fatal("depth_proposal_card is empty in early-return branch")
		}
	})

	t.Run("proposal_recommendations_match_result_values", func(t *testing.T) {
		saveGlobals(t)
		setupPhaseResearchManifestTest(t, researchProposalTestPhases())

		result := runPlanOnly(t, "--refresh", "--depth", "balanced")

		gotGranularity, _ := result["granularity"].(string)
		gotPlanningDepth, _ := result["planning_depth"].(string)
		gotVerificationDepth, _ := result["verification_depth"].(string)

		proposal, _ := result["depth_proposal"].(map[string]interface{})
		knobByKey := map[string]map[string]interface{}{}
		for _, raw := range proposal["knobs"].([]interface{}) {
			knob := raw.(map[string]interface{})
			knobByKey[knob["key"].(string)] = knob
		}

		if got := knobByKey["granularity"]["recommended"].(string); got != gotGranularity {
			t.Errorf("proposal granularity recommendation = %q, want %q (matching result.granularity)", got, gotGranularity)
		}
		if got := knobByKey["planning_depth"]["recommended"].(string); got != gotPlanningDepth {
			t.Errorf("proposal planning_depth recommendation = %q, want %q (matching result.planning_depth)", got, gotPlanningDepth)
		}
		if got := knobByKey["verification_depth"]["recommended"].(string); got != gotVerificationDepth {
			t.Errorf("proposal verification_depth recommendation = %q, want %q (matching result.verification_depth)", got, gotVerificationDepth)
		}

		card, _ := result["depth_proposal_card"].(string)
		var acceptLine string
		for _, line := range strings.Split(card, "\n") {
			if strings.Contains(line, "Accept all:") {
				acceptLine = line
			}
		}
		if acceptLine == "" {
			t.Fatal("could not find Accept all: line in depth_proposal_card")
		}
		for _, want := range []string{gotGranularity, gotPlanningDepth, gotVerificationDepth} {
			if !strings.Contains(acceptLine, want) {
				t.Errorf("accept line %q missing value %q", acceptLine, want)
			}
		}
	})

	t.Run("explicit_planning_depth_reports_explicit_reason", func(t *testing.T) {
		saveGlobals(t)
		setupPhaseResearchManifestTest(t, researchProposalTestPhases())

		result := runPlanOnly(t, "--refresh", "--depth", "balanced", "--planning-depth", "deep")

		proposal, _ := result["depth_proposal"].(map[string]interface{})
		var planningKnob map[string]interface{}
		for _, raw := range proposal["knobs"].([]interface{}) {
			knob := raw.(map[string]interface{})
			if knob["key"].(string) == "planning_depth" {
				planningKnob = knob
			}
		}
		if planningKnob == nil {
			t.Fatal("planning_depth knob not found in depth_proposal")
		}
		reason, _ := planningKnob["reason"].(string)
		if !strings.Contains(reason, "explicitly") {
			t.Errorf("planning_depth reason = %q, want it to mention explicit selection", reason)
		}
	})
}
