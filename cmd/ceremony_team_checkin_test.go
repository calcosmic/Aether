package cmd

import (
	"strings"
	"testing"
)

func teamCheckinManifestFixture() (map[string]interface{}, []ceremonyDispatch) {
	manifest := map[string]interface{}{
		"queen_execution_policy": map[string]interface{}{
			"spawn_budget": map[string]interface{}{
				"required_castes": []interface{}{"builder"},
				"selected_reasons": map[string]interface{}{
					"builder":  "writes the code for 3 task(s): add the password reset flow",
					"measurer": "Measurer — the phase mentions a slow page load to investigate",
				},
				"pruned_reasons": map[string]interface{}{
					"chaos": "pruned by worker budget (max 5)",
				},
			},
		},
		"caste_roster": []interface{}{
			map[string]interface{}{"caste": "measurer", "produces": "performance findings"},
		},
	}
	dispatches := []ceremonyDispatch{
		{Caste: "builder", Name: "Mason-12", Task: "Build the login form"},
		{Caste: "measurer", Name: "Gauge-7", Task: "Check latency"},
	}
	return manifest, dispatches
}

// TestTeamCheckinCardShowsReasonAndRequiredMarking pins the card the owner
// approves before any worker spawns: every worker carries its one-line
// reason, safety workers are marked REQUIRED (the runtime re-adds them
// whatever the owner picks, so offering them for trimming would be a lie),
// and already-pruned castes are shown so the owner sees what was saved.
func TestTeamCheckinCardShowsReasonAndRequiredMarking(t *testing.T) {
	manifest, dispatches := teamCheckinManifestFixture()
	result, visual := renderCeremonyTeamCheckin("build", manifest, dispatches)

	for _, want := range []string{
		"T E A M   C H E C K - I N",
		"REQUIRED",
		"OPTIONAL",
		"writes the code for 3 task(s): add the password reset flow",
		"Measurer — the phase mentions a slow page load to investigate",
		"Not sent:",
		"pruned by worker budget (max 5)",
		"Required workers stay",
	} {
		if !strings.Contains(visual, want) {
			t.Errorf("check-in card missing %q.\ncard:\n%s", want, visual)
		}
	}

	required, _ := result["required"].([]string)
	optional, _ := result["optional"].([]string)
	if len(required) != 1 || required[0] != "builder" {
		t.Fatalf("required bucket = %v, want [builder]", required)
	}
	if len(optional) != 1 || optional[0] != "measurer" {
		t.Fatalf("optional bucket = %v, want [measurer]", optional)
	}
}

// TestTeamCheckinNeverShowsAGenericBlurbAsAReason replaces the retired test
// that asserted the exact fallback D-09 forbids (see
// .aether/docs/retired-tests-ledger.md). When the budget carried no per-caste
// rationale for a worker, the roster's generic "what it does" description
// must never stand in for the reason it was sent — that reads as a
// justification and is not one. The description may still appear, but only
// in the separately labelled what_it_does slot.
func TestTeamCheckinNeverShowsAGenericBlurbAsAReason(t *testing.T) {
	manifest, dispatches := teamCheckinManifestFixture()
	budget := mapValue(mapValue(manifest["queen_execution_policy"])["spawn_budget"])
	delete(mapValue(budget["selected_reasons"]), "measurer")

	result, visual := renderCeremonyTeamCheckin("build", manifest, dispatches)

	reasons, _ := result["reasons"].(map[string]string)
	if reasons["measurer"] == "performance findings" {
		t.Fatalf("roster description leaked into the reason slot: %q", reasons["measurer"])
	}
	if strings.TrimSpace(reasons["measurer"]) == "" {
		t.Fatalf("a caste with no per-phase reason must render an explicit marker, got empty string")
	}

	whatItDoes, _ := result["what_it_does"].(map[string]string)
	if whatItDoes["measurer"] != "performance findings" {
		t.Fatalf("what_it_does should carry the roster description separately, got %q", whatItDoes["measurer"])
	}
	if !strings.Contains(visual, "what it does: performance findings") {
		t.Fatalf("card should label the roster description as what it does, never as the reason.\ncard:\n%s", visual)
	}
}
