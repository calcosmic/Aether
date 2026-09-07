package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPlanManifestCarriesDepthProposal retains the historical test name while
// pinning Phase 200's replacement contract: one unbiased four-preset choice,
// or one exact selected policy and one Scout-stage dispatch.
func TestPlanManifestCarriesDepthProposal(t *testing.T) {
	t.Run("unflagged_plan_requires_unbiased_preset_before_dispatch", func(t *testing.T) {
		saveGlobals(t)
		_, _ = setupPhaseResearchManifestTest(t, researchProposalTestPhases())
		before, err := os.ReadFile(filepath.Join(store.BasePath(), "COLONY_STATE.json"))
		if err != nil {
			t.Fatal(err)
		}
		result := runPlanOnly(t)
		if required, _ := result["preset_required"].(bool); !required {
			t.Fatalf("preset_required = %v, want true", result["preset_required"])
		}
		options, ok := result["preset_options"].([]interface{})
		if !ok || len(options) != 4 {
			t.Fatalf("preset_options = %#v, want four choices", result["preset_options"])
		}
		for _, raw := range options {
			option := raw.(map[string]interface{})
			if _, recommended := option["recommended"]; recommended {
				t.Fatalf("preset option contains forbidden recommendation: %+v", option)
			}
		}
		if selected, _ := result["selected_preset"].(string); selected != "" {
			t.Fatalf("selected_preset = %q, want no implicit default", selected)
		}
		if count, _ := result["dispatch_count"].(float64); count != 0 {
			t.Fatalf("dispatch_count = %v, want zero", result["dispatch_count"])
		}
		if _, exists := result["plan_manifest"]; exists {
			t.Fatal("unflagged invocation emitted a worker manifest")
		}
		after, err := os.ReadFile(filepath.Join(store.BasePath(), "COLONY_STATE.json"))
		if err != nil {
			t.Fatal(err)
		}
		if string(before) != string(after) {
			t.Fatal("preset selection prompt mutated planning state")
		}
	})

	t.Run("existing_plan_early_return_carries_selected_policy", func(t *testing.T) {
		saveGlobals(t)
		setupPhaseResearchManifestTest(t, researchProposalTestPhases())
		result := runPlanOnly(t, "--preset", "fast")
		if existing, _ := result["existing_plan"].(bool); !existing {
			t.Fatalf("existing_plan = %v, want true", result["existing_plan"])
		}
		if result["selected_preset"] != "fast" || int(result["target_confidence"].(float64)) != 80 || int(result["max_iterations"].(float64)) != 4 {
			t.Fatalf("early-return preset policy = %+v", result)
		}
	})

	t.Run("fresh_plan_manifest_matches_selected_policy_and_dispatches_only_scout", func(t *testing.T) {
		saveGlobals(t)
		setupPhaseResearchManifestTest(t, researchProposalTestPhases())
		result := runPlanOnly(t, "--refresh", "--preset", "balanced")
		manifest, ok := result["plan_manifest"].(map[string]interface{})
		if !ok {
			t.Fatalf("plan_manifest missing: %+v", result)
		}
		if result["selected_preset"] != "balanced" || manifest["selected_preset"] != result["selected_preset"] {
			t.Fatalf("result/manifest preset mismatch: result=%v manifest=%v", result["selected_preset"], manifest["selected_preset"])
		}
		if manifest["target_confidence"] != result["target_confidence"] || manifest["max_iterations"] != result["max_iterations"] {
			t.Fatalf("result/manifest policy mismatch: result=%+v manifest=%+v", result, manifest)
		}
		dispatches, ok := manifest["dispatches"].([]interface{})
		if !ok || len(dispatches) != 1 {
			t.Fatalf("manifest dispatches = %#v, want one Scout stage", manifest["dispatches"])
		}
		worker := dispatches[0].(map[string]interface{})
		if worker["caste"] != "scout" {
			t.Fatalf("first authorized caste = %v, want scout", worker["caste"])
		}
		if _, forbidden := manifest["depth_proposal"]; forbidden {
			t.Fatal("manifest retained obsolete three-knob depth proposal")
		}
	})
}
