package cmd

import (
	"strings"
	"testing"
)

func teamCheckinManifestFixture() (map[string]interface{}, []ceremonyDispatch) {
	manifest := map[string]interface{}{
		"queen_execution_policy": map[string]interface{}{
			"spawn_budget": map[string]interface{}{
				"required_castes": []interface{}{"watcher"},
				"selected_reasons": map[string]interface{}{
					"watcher":  "watcher is always required for build flow",
					"measurer": "Score 40 >= threshold 30 for build flow",
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
		{Caste: "watcher", Name: "Sharp-12", Task: "Verify the build"},
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
		"watcher is always required for build flow",
		"Score 40 >= threshold 30 for build flow",
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
	if len(required) != 1 || required[0] != "watcher" {
		t.Fatalf("required bucket = %v, want [watcher]", required)
	}
	if len(optional) != 1 || optional[0] != "measurer" {
		t.Fatalf("optional bucket = %v, want [measurer]", optional)
	}
}

// TestTeamCheckinFallsBackToRosterProduces: when the budget carried no
// per-caste rationale, the roster's produces-prose stands in so no worker
// line is ever reason-less.
func TestTeamCheckinFallsBackToRosterProduces(t *testing.T) {
	manifest, dispatches := teamCheckinManifestFixture()
	budget := mapValue(mapValue(manifest["queen_execution_policy"])["spawn_budget"])
	delete(mapValue(budget["selected_reasons"]), "measurer")

	_, visual := renderCeremonyTeamCheckin("build", manifest, dispatches)
	if !strings.Contains(visual, "performance findings") {
		t.Fatalf("card should fall back to the roster's produces prose.\ncard:\n%s", visual)
	}
}
