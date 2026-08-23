package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
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
// rationale for a
// worker, the roster's generic "what it does" description must never stand
// in for the reason it was sent — that reads as a justification and is not
// one. The description may still appear, but only in the separately labelled
// what_it_does slot.
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

// manifestMapFromBuild runs a real plan-only build for the given phase and
// returns the manifest in the SAME shape the check-in card actually reads it
// in production: the JSON-round-tripped codexBuildManifest, not a hand-typed
// fixture. This is what lets TestRequiredMeansBuilderOrNamedSignal and
// TestCardNamesTheSignalForEveryForcedReviewer assert on the real dispatch
// list and the real forced-reviewer derivation instead of an intermediate
// struct (this repo's own established pattern — assert the spawn list, not
// the decision record).
func manifestMapFromBuild(t *testing.T, root string, phaseNum int) (map[string]interface{}, []ceremonyDispatch) {
	t.Helper()
	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, phaseNum, nil, codexBuildOptions{})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions: %v", err)
	}
	manifest, ok := result["dispatch_manifest"].(codexBuildManifest)
	if !ok {
		t.Fatalf("dispatch_manifest is not a codexBuildManifest: %T", result["dispatch_manifest"])
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	var manifestMap map[string]interface{}
	if err := json.Unmarshal(raw, &manifestMap); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	return manifestMap, ceremonyDispatchesFromManifest(manifestMap)
}

func checkinFixturePhase(name, description string, mode colony.PhaseMode) colony.Phase {
	taskID := "1.1"
	return colony.Phase{
		ID:          1,
		Name:        name,
		Description: description,
		Mode:        mode,
		Status:      colony.PhaseReady,
		Tasks:       []colony.Task{{ID: &taskID, Goal: "Do the work", Status: colony.TaskPending}},
	}
}

func setUpCheckinFixtureColony(t *testing.T, dataDir string, phase colony.Phase) {
	t.Helper()
	goal := phase.Name
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan:         colony.Plan{Phases: []colony.Phase{phase}},
	})
}

// TestCardNamesTheSignalForEveryForcedReviewer pins D-05's owner-facing
// wording: a phase whose text names one of the five risk signals must render
// the reviewer's plain-English signal words AND the matched phrase on the
// card itself — asserted on the rendered card, not the result map alone,
// because the card is what the owner reads. A plain phase with no signal
// wording must render no announcement block and no empty heading.
func TestCardNamesTheSignalForEveryForcedReviewer(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]

	t.Run("password reset phase names the signal", func(t *testing.T) {
		phase := checkinFixturePhase(
			"Password reset",
			"Let users reset their password via an emailed token",
			colony.PhaseModePrototype,
		)
		setUpCheckinFixtureColony(t, dataDir, phase)
		manifestMap, dispatches := manifestMapFromBuild(t, root, 1)
		_, visual := renderCeremonyTeamCheckin("build", manifestMap, dispatches)

		for _, want := range []string{"logins", "password reset", "Not sent with this team"} {
			if !strings.Contains(visual, want) {
				t.Errorf("card missing %q for a forced reviewer.\ncard:\n%s", want, visual)
			}
		}
	})

	t.Run("plain phase has no announcement block", func(t *testing.T) {
		phase := checkinFixturePhase(
			"Change button copy",
			"Change the Submit button label to Save",
			colony.PhaseModePrototype,
		)
		setUpCheckinFixtureColony(t, dataDir, phase)
		manifestMap, dispatches := manifestMapFromBuild(t, root, 1)
		_, visual := renderCeremonyTeamCheckin("build", manifestMap, dispatches)

		if strings.Contains(visual, "Not sent with this team") {
			t.Fatalf("plain phase should render no forced-reviewer announcement block.\ncard:\n%s", visual)
		}
	})
}

// TestRequiredMeansBuilderOrNamedSignal is the explicit assertion D-10
// demands rather than trusting it holds by construction: across a plain
// phase, a password-reset phase, a discovery phase, and a two-signal phase,
// every caste the card marks REQUIRED is either the implementation worker
// (builder) or a caste the manifest's own forced_reviewers names with a
// non-empty signal list. Nothing else may ever be REQUIRED — this is the
// test that fails if a future change quietly re-adds an implicit
// requirement, whatever it is called.
func TestRequiredMeansBuilderOrNamedSignal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]

	cases := []struct {
		name  string
		phase colony.Phase
	}{
		{
			name: "plain phase",
			phase: checkinFixturePhase(
				"Change button copy",
				"Change the Submit button label to Save",
				colony.PhaseModePrototype,
			),
		},
		{
			name: "password reset phase",
			phase: checkinFixturePhase(
				"Password reset",
				"Let users reset their password by email",
				colony.PhaseModePrototype,
			),
		},
		{
			name: "discovery phase",
			phase: checkinFixturePhase(
				"Research payment providers",
				"Survey the available payment providers, no code yet",
				colony.PhaseModeDiscovery,
			),
		},
		{
			name: "two-signal phase",
			phase: checkinFixturePhase(
				"Refund after login",
				"Let users log in and then request a refund for a recent charge",
				colony.PhaseModePrototype,
			),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setUpCheckinFixtureColony(t, dataDir, tc.phase)
			manifestMap, dispatches := manifestMapFromBuild(t, root, 1)
			result, visual := renderCeremonyTeamCheckin("build", manifestMap, dispatches)

			forced, _ := result["forced"].(map[string]interface{})
			required, _ := result["required"].([]string)
			for _, caste := range required {
				if caste == "builder" {
					continue
				}
				entry, ok := forced[caste].(map[string]interface{})
				if !ok {
					t.Fatalf("REQUIRED caste %q is neither builder nor forced by a named signal (forced=%v).\ncard:\n%s", caste, forced, visual)
				}
				signals, _ := entry["signals"].([]string)
				if len(signals) == 0 {
					t.Fatalf("REQUIRED caste %q has an empty signal list", caste)
				}
			}
		})
	}
}

// TestTeamCheckinCardIsVisuallySeparated pins the 2026-08-23 owner feedback on
// plan 194-07: the card was "just like text," a wall with no lines breaking
// it up. Two invariants, not literal strings, so the assertion survives a
// future rename of any one section: every two adjacent worker rows are
// separated by a blank line, and the card's sections are broken up by rule
// lines (the same `── Title ──` stage-marker style already used elsewhere in
// this repo) rather than running straight into each other.
func TestTeamCheckinCardIsVisuallySeparated(t *testing.T) {
	manifest, dispatches := teamCheckinManifestFixture()
	_, visual := renderCeremonyTeamCheckin("build", manifest, dispatches)

	lines := strings.Split(visual, "\n")
	workerRowIdx := []int{}
	for i, line := range lines {
		if strings.Contains(line, "REQUIRED") || strings.Contains(line, "OPTIONAL") {
			workerRowIdx = append(workerRowIdx, i)
		}
	}
	if len(workerRowIdx) < 2 {
		t.Fatalf("fixture must produce at least two worker rows to assert separation; card:\n%s", visual)
	}
	for i := 1; i < len(workerRowIdx); i++ {
		prev, cur := workerRowIdx[i-1], workerRowIdx[i]
		if cur-prev < 2 || strings.TrimSpace(lines[prev+1]) != "" {
			t.Fatalf("worker rows %d and %d are not separated by a blank line; card:\n%s", prev, cur, visual)
		}
	}

	ruleCount := strings.Count(visual, "── ")
	if ruleCount < 2 {
		t.Fatalf("card must use rule lines (── Title ──) to separate sections, found %d; card:\n%s", ruleCount, visual)
	}
	if !strings.Contains(visual, "── Team ──") {
		t.Fatalf("card must open its worker list with a Team rule line; card:\n%s", visual)
	}
	if !strings.Contains(visual, "── Summary ──") {
		t.Fatalf("card must precede its closing line with a Summary rule line; card:\n%s", visual)
	}
}

// TestOneWorkerTeamStillPauses pins D-14: the owner's pre-build pause never
// disappears just because the team is small. A manifest whose only worker is
// the implementation caste itself must still render a full check-in card,
// with that worker still marked REQUIRED -- the wrapper's decision to pause
// and ask is driven by the existence of a card to show, not by team size, and
// this is the regression guard against a future "skip the checkin for a
// single-worker build" shortcut.
func TestOneWorkerTeamStillPauses(t *testing.T) {
	manifest := map[string]interface{}{
		"queen_execution_policy": map[string]interface{}{
			"spawn_budget": map[string]interface{}{
				"required_castes": []interface{}{"builder"},
				"selected_reasons": map[string]interface{}{
					"builder": "writes the code for 1 task(s): fix the off-by-one error in the pager",
				},
			},
		},
	}
	dispatches := []ceremonyDispatch{
		{Caste: "builder", Name: "Mason-1", Task: "Fix the off-by-one error"},
	}

	result, visual := renderCeremonyTeamCheckin("build", manifest, dispatches)

	if !strings.Contains(visual, "T E A M   C H E C K - I N") {
		t.Fatalf("a one-worker team must still render the check-in card.\ncard:\n%s", visual)
	}
	if !strings.Contains(visual, "REQUIRED") {
		t.Fatalf("a one-worker team's own worker must still be marked REQUIRED.\ncard:\n%s", visual)
	}
	required, _ := result["required"].([]string)
	if len(required) != 1 || required[0] != "builder" {
		t.Fatalf("one-worker team required = %v, want [builder]", required)
	}
}

// TestTeamCheckinDoesNotMutate is re-run in this plan's own acceptance
// criteria to prove the reason/forced-reviewer rendering work stayed
// read-only — the check-in command is an inspection, and an inspection
// command that mutates is a named corollary of this repo's Definition of
// Done.
func TestTeamCheckinDoesNotMutate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]

	phase := checkinFixturePhase(
		"Password reset",
		"Let users reset their password via an emailed token",
		colony.PhaseModePrototype,
	)
	setUpCheckinFixtureColony(t, dataDir, phase)

	before, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState before: %v", err)
	}

	manifestMap, dispatches := manifestMapFromBuild(t, root, 1)
	_, _ = renderCeremonyTeamCheckin("build", manifestMap, dispatches)

	after, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState after: %v", err)
	}
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	if string(beforeJSON) != string(afterJSON) {
		t.Fatalf("rendering the check-in card mutated colony state:\nbefore: %s\nafter:  %s", beforeJSON, afterJSON)
	}
}
