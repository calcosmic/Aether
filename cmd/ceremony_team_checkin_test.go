package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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

// TestRenderCeremonyTeamCheckinStillRendersFullCardForOneWorkerWhenCalled
// replaces the old TestOneWorkerTeamStillPauses (D-14/195-CONTEXT.md
// reversed the "every one-worker build pauses" default). What survives from
// that test is narrower and still true: the FULL card renderer itself is
// completely unchanged -- calling it directly for a one-worker manifest
// still produces a real check-in card with that worker marked REQUIRED.
// What changed is who decides to call it: decideBuildCheckin
// (TestBuildCheckinDecisionMatrix, TestOneWorkerBuildSkipsCheckin below),
// not team size alone.
func TestRenderCeremonyTeamCheckinStillRendersFullCardForOneWorkerWhenCalled(t *testing.T) {
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
		t.Fatalf("the full renderer must still produce a check-in card for a one-worker manifest.\ncard:\n%s", visual)
	}
	if !strings.Contains(visual, "REQUIRED") {
		t.Fatalf("a one-worker team's own worker must still be marked REQUIRED.\ncard:\n%s", visual)
	}
	required, _ := result["required"].([]string)
	if len(required) != 1 || required[0] != "builder" {
		t.Fatalf("one-worker team required = %v, want [builder]", required)
	}
}

// TestBuildCheckinDecisionMatrix is the pure, table-driven regression guard
// for decideBuildCheckin's D-11..D-14 precedence: autopilot and --no-checkin
// always stay non-interactive; --checkin always forces the pause; any
// pending owner decision forces the pause even for one worker; exactly one
// implementation dispatch with nothing else pending takes the fast path;
// everything else pauses as before. The "grouped" and "one ungrouped
// worker" rows are deliberately identical inputs at this pure layer --
// decideBuildCheckin only ever sees a dispatch COUNT, never whether that one
// dispatch happens to cover more than one task. The distinction (which
// grouping produced the count) belongs to the summary renderer's own tests
// (TestOneWorkerFastPathSummaryCarriesEveryFact), not here.
func TestBuildCheckinDecisionMatrix(t *testing.T) {
	cases := []struct {
		name       string
		input      buildCheckinDecisionInput
		wantReq    bool
		wantReason buildCheckinReasonCode
	}{
		{
			name:       "autopilot never pauses",
			input:      buildCheckinDecisionInput{Autopilot: true, ImplementationDispatches: 3},
			wantReq:    false,
			wantReason: buildCheckinReasonNonInteractive,
		},
		{
			name:       "--no-checkin never pauses",
			input:      buildCheckinDecisionInput{NoCheckin: true, ImplementationDispatches: 3},
			wantReq:    false,
			wantReason: buildCheckinReasonNonInteractive,
		},
		{
			name:       "--checkin forces a pause on an otherwise-eligible one-worker build",
			input:      buildCheckinDecisionInput{Checkin: true, ImplementationDispatches: 1},
			wantReq:    true,
			wantReason: buildCheckinReasonExplicitCheckin,
		},
		{
			// The CLI refuses --checkin combined with --no-checkin before
			// decideBuildCheckin is ever called (TestCheckinFlagConflictHasNoSideEffects);
			// this row documents the pure function's own deterministic
			// precedence as defense in depth, never as sanctioned input.
			name:       "both flags together: non-interactive wins in the pure policy",
			input:      buildCheckinDecisionInput{Checkin: true, NoCheckin: true, ImplementationDispatches: 1},
			wantReq:    false,
			wantReason: buildCheckinReasonNonInteractive,
		},
		{
			name:       "one grouped worker takes the fast path",
			input:      buildCheckinDecisionInput{ImplementationDispatches: 1},
			wantReq:    false,
			wantReason: buildCheckinReasonOneWorkerFastPath,
		},
		{
			name:       "one ungrouped worker takes the fast path",
			input:      buildCheckinDecisionInput{ImplementationDispatches: 1},
			wantReq:    false,
			wantReason: buildCheckinReasonOneWorkerFastPath,
		},
		{
			name:       "two workers still pause",
			input:      buildCheckinDecisionInput{ImplementationDispatches: 2},
			wantReq:    true,
			wantReason: buildCheckinReasonDefaultPause,
		},
		{
			name: "a live forced-reviewer waiver keeps the pause even for one worker",
			input: buildCheckinDecisionInput{
				ImplementationDispatches: 1,
				PendingOwnerDecision:     true,
				PendingOwnerDecisionWhy:  "a forced reviewer is still waiting for the owner's check-in decision",
			},
			wantReq:    true,
			wantReason: buildCheckinReasonPendingOwnerDecision,
		},
		{
			name: "an unanswered boundary question keeps the pause even for one worker",
			input: buildCheckinDecisionInput{
				ImplementationDispatches: 1,
				PendingOwnerDecision:     true,
				PendingOwnerDecisionWhy:  "an unanswered planning question is still waiting for the owner",
			},
			wantReq:    true,
			wantReason: buildCheckinReasonPendingOwnerDecision,
		},
		{
			name: "another persisted unanswered owner decision keeps the pause even for one worker",
			input: buildCheckinDecisionInput{
				ImplementationDispatches: 1,
				PendingOwnerDecision:     true,
				PendingOwnerDecisionWhy:  "a worker left a question only the owner can answer",
			},
			wantReq:    true,
			wantReason: buildCheckinReasonPendingOwnerDecision,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := decideBuildCheckin(tc.input)
			if got.Requested != tc.wantReq {
				t.Fatalf("Requested = %v, want %v (reason %s, why %q)", got.Requested, tc.wantReq, got.Reason, got.Why)
			}
			if got.Reason != tc.wantReason {
				t.Fatalf("Reason = %s, want %s", got.Reason, tc.wantReason)
			}
			if strings.TrimSpace(got.Why) == "" {
				t.Fatalf("Why must never be empty")
			}
		})
	}
}

// TestOneWorkerBuildSkipsCheckin runs real plan-only builds through the
// whole coherent-job planner and proves decideBuildCheckin's fast path on
// the actual dispatch shapes it must handle: one ungrouped task, one
// grouped job covering a dependent pair, and two genuinely independent
// tasks that must still pause.
func TestOneWorkerBuildSkipsCheckin(t *testing.T) {
	t.Run("one ungrouped task", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		dataDir := setupBuildFlowTest(t)
		root := dataDir[:len(dataDir)-len("/.aether/data")]

		phase := checkinFixturePhase("Fix the pager", "Fix the off-by-one error in the pager", colony.PhaseModePrototype)
		setUpCheckinFixtureColony(t, dataDir, phase)

		result, _, _, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{})
		if err != nil {
			t.Fatalf("runCodexBuildPlanOnlyWithOptions: %v", err)
		}
		if len(dispatches) != 1 {
			t.Fatalf("expected exactly one dispatch, got %d", len(dispatches))
		}
		manifest, ok := result["dispatch_manifest"].(codexBuildManifest)
		if !ok {
			t.Fatalf("dispatch_manifest is not a codexBuildManifest: %T", result["dispatch_manifest"])
		}
		pending, why := buildHasPendingOwnerDecision(manifest)
		if pending {
			t.Fatalf("a plain one-task phase must not report a pending owner decision (why=%q)", why)
		}
		decision := decideBuildCheckin(buildCheckinDecisionInput{ImplementationDispatches: len(dispatches), PendingOwnerDecision: pending})
		if decision.Requested {
			t.Fatalf("one ungrouped worker must skip the check-in, got Requested=true (%s)", decision.Reason)
		}
		if decision.Reason != buildCheckinReasonOneWorkerFastPath {
			t.Fatalf("reason = %s, want %s", decision.Reason, buildCheckinReasonOneWorkerFastPath)
		}
	})

	t.Run("one grouped job covering two dependent tasks", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		dataDir := setupBuildFlowTest(t)
		root := dataDir[:len(dataDir)-len("/.aether/data")]

		id1, id2 := "1", "2"
		goal := "Coherent job fast path"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			ColonyDepth:  "full",
			CurrentPhase: 0,
			Plan: colony.Plan{Phases: []colony.Phase{{
				ID:     1,
				Name:   "Coherent job fast path",
				Mode:   colony.PhaseModePrototype,
				Status: colony.PhaseReady,
				Tasks: []colony.Task{
					{ID: &id1, Goal: "Add the shared template", Status: colony.TaskPending},
					{ID: &id2, Goal: "Wire the shared template into the form", Status: colony.TaskPending, DependsOn: []string{id1}},
				},
			}}},
		})

		result, _, _, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{})
		if err != nil {
			t.Fatalf("runCodexBuildPlanOnlyWithOptions: %v", err)
		}
		if len(dispatches) != 1 {
			t.Fatalf("expected the dependent pair to merge into one dispatch, got %d", len(dispatches))
		}
		if covered := dispatchCoveredTaskIDs(dispatches[0]); len(covered) != 2 {
			t.Fatalf("expected the one dispatch to cover both tasks, covered = %v", covered)
		}
		manifest, _ := result["dispatch_manifest"].(codexBuildManifest)
		pending, why := buildHasPendingOwnerDecision(manifest)
		decision := decideBuildCheckin(buildCheckinDecisionInput{ImplementationDispatches: len(dispatches), PendingOwnerDecision: pending, PendingOwnerDecisionWhy: why})
		if decision.Requested {
			t.Fatalf("one grouped worker covering multiple tasks must still skip the check-in, got Requested=true (%s)", decision.Reason)
		}
	})

	t.Run("two independent workers still pause", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		dataDir := setupBuildFlowTest(t)
		root := dataDir[:len(dataDir)-len("/.aether/data")]

		id1, id2 := "1", "2"
		goal := "Two independent tasks"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			ColonyDepth:  "full",
			CurrentPhase: 0,
			Plan: colony.Plan{Phases: []colony.Phase{{
				ID:     1,
				Name:   "Two independent tasks",
				Mode:   colony.PhaseModePrototype,
				Status: colony.PhaseReady,
				Tasks: []colony.Task{
					{ID: &id1, Goal: "Fix the pager off-by-one error", Status: colony.TaskPending},
					{ID: &id2, Goal: "Fix an unrelated typo in the footer", Status: colony.TaskPending},
				},
			}}},
		})

		_, _, _, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{})
		if err != nil {
			t.Fatalf("runCodexBuildPlanOnlyWithOptions: %v", err)
		}
		if len(dispatches) != 2 {
			t.Fatalf("expected two independent tasks to dispatch as two workers, got %d", len(dispatches))
		}
		decision := decideBuildCheckin(buildCheckinDecisionInput{ImplementationDispatches: len(dispatches)})
		if !decision.Requested {
			t.Fatalf("two workers must still pause for the check-in")
		}
		if decision.Reason != buildCheckinReasonDefaultPause {
			t.Fatalf("reason = %s, want %s", decision.Reason, buildCheckinReasonDefaultPause)
		}
	})
}

// TestOneWorkerWithPersistedOwnerDecisionStillPauses is the third
// D-13 pending-owner-decision source: a worker's own unanswered
// open_decisions handoff question (distinct from the forced-reviewer
// waiver and the orchestrator boundary question, each covered by their own
// dedicated test in forced_reviewer_waiver_test.go and
// orchestrator_boundary_questions_test.go respectively).
func TestOneWorkerWithPersistedOwnerDecisionStillPauses(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]

	phase := checkinFixturePhase("Fix the pager", "Fix the off-by-one error in the pager", colony.PhaseModePrototype)
	setUpCheckinFixtureColony(t, dataDir, phase)
	seedHandoffOpenDecision(t, "Should the fix also cover the mobile pager?")

	result, _, _, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions: %v", err)
	}
	if len(dispatches) != 1 {
		t.Fatalf("expected exactly one dispatch, got %d", len(dispatches))
	}
	manifest, _ := result["dispatch_manifest"].(codexBuildManifest)
	pending, why := buildHasPendingOwnerDecision(manifest)
	if !pending {
		t.Fatalf("a worker's unanswered open decision must count as a pending owner decision")
	}
	if strings.TrimSpace(why) == "" {
		t.Fatalf("expected a plain-English reason")
	}
	decision := decideBuildCheckin(buildCheckinDecisionInput{
		ImplementationDispatches: len(dispatches),
		PendingOwnerDecision:     pending,
		PendingOwnerDecisionWhy:  why,
	})
	if !decision.Requested {
		t.Fatalf("a pending worker-raised decision must keep the check-in even for one worker")
	}
	if decision.Reason != buildCheckinReasonPendingOwnerDecision {
		t.Fatalf("reason = %s, want %s", decision.Reason, buildCheckinReasonPendingOwnerDecision)
	}
}

// TestPendingDecisionStillRendersFullCheckinCard proves the full check-in
// card (and its Phase 194 waiver option) is completely unchanged by the
// Phase 195 fast path: a one-worker build with a live forced-reviewer
// signal still pauses (decideBuildCheckin), and the SAME full card, with
// the SAME decline-a-required-reviewer flow, is what renders for it -- the
// fast-path summary is never substituted in when a decision is pending.
func TestPendingDecisionStillRendersFullCheckinCard(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]

	// Phase ID 5 (not checkinFixturePhase's shared ID 1) sidesteps
	// chaosShouldRunInLightMode's deterministic phaseID%10<3 sampling
	// (cmd/review_depth.go) so LightFlag keeps this build to exactly the
	// one builder -- the forced-reviewer signal below is a build-time
	// ANNOUNCEMENT independent of review depth (D-05), so it stays live
	// regardless.
	taskID := "1.1"
	phase := colony.Phase{
		ID:          5,
		Name:        "Password reset",
		Description: "Let users reset their password via an emailed token",
		Mode:        colony.PhaseModePrototype,
		Status:      colony.PhaseReady,
		Tasks:       []colony.Task{{ID: &taskID, Goal: "Do the work", Status: colony.TaskPending}},
	}
	setUpCheckinFixtureColony(t, dataDir, phase)

	result, _, _, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{LightFlag: true})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions: %v", err)
	}
	if len(dispatches) != 1 {
		t.Fatalf("expected exactly one dispatch, got %d", len(dispatches))
	}
	manifest, ok := result["dispatch_manifest"].(codexBuildManifest)
	if !ok {
		t.Fatalf("dispatch_manifest is not a codexBuildManifest: %T", result["dispatch_manifest"])
	}
	pending, why := buildHasPendingOwnerDecision(manifest)
	if !pending {
		t.Fatalf("a phase naming a forced-reviewer signal must report a pending owner decision")
	}
	decision := decideBuildCheckin(buildCheckinDecisionInput{
		ImplementationDispatches: len(dispatches),
		PendingOwnerDecision:     pending,
		PendingOwnerDecisionWhy:  why,
	})
	if !decision.Requested {
		t.Fatalf("expected the check-in to still be requested, got %+v", decision)
	}

	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	var manifestMap map[string]interface{}
	if err := json.Unmarshal(raw, &manifestMap); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	ceremonyDispatches := ceremonyDispatchesFromManifest(manifestMap)
	_, visual := renderCeremonyTeamCheckin("build", manifestMap, ceremonyDispatches)
	for _, want := range []string{"T E A M   C H E C K - I N", "REQUIRED", "To decline, run:"} {
		if !strings.Contains(visual, want) {
			t.Fatalf("full check-in card missing %q for a pending one-worker build.\ncard:\n%s", want, visual)
		}
	}
}

// TestOneWorkerFastPathSummaryCarriesEveryFact pins D-12: the compact
// summary must name the worker, every covered task, the accepted
// relationship and benefit (or the honest single-task reason), and why no
// approval is required.
func TestOneWorkerFastPathSummaryCarriesEveryFact(t *testing.T) {
	fastPathDecision := buildCheckinDecision{
		Requested: false,
		Reason:    buildCheckinReasonOneWorkerFastPath,
		Why:       "no owner decision is pending, so dispatch continues",
	}

	t.Run("grouped job", func(t *testing.T) {
		phase := colony.Phase{ID: 1, Tasks: []colony.Task{
			{ID: strPtr("1"), Goal: "Add the shared template"},
			{ID: strPtr("2"), Goal: "Wire the shared template into the form"},
		}}
		dispatch := codexBuildDispatch{
			Caste:          "builder",
			Name:           "Mason-12",
			TaskID:         "1",
			CoveredTaskIDs: []string{"1", "2"},
			JobName:        "automatic-1",
			JobSource:      coherentJobSourceAutomatic,
			JobReason:      "tasks 1 and 2 modify the same templates, so one builder avoids repeated setup and write conflicts",
		}

		result, visual := renderBuildFastPathSummary(phase, dispatch, fastPathDecision)

		if worker, _ := result["worker"].(string); !strings.Contains(worker, "Mason-12") {
			t.Fatalf("result worker %q must name the deterministic worker", worker)
		}
		coveredIDs, _ := result["covered_task_ids"].([]string)
		if len(coveredIDs) != 2 || coveredIDs[0] != "1" || coveredIDs[1] != "2" {
			t.Fatalf("covered_task_ids = %v, want [1 2]", coveredIDs)
		}
		if got, _ := result["relationship"].(string); got != "tasks 1 and 2 modify the same templates" {
			t.Fatalf("relationship = %q", got)
		}
		if got, _ := result["benefit"].(string); got != "one builder avoids repeated setup and write conflicts" {
			t.Fatalf("benefit = %q", got)
		}
		if why, _ := result["why_no_approval"].(string); !strings.Contains(why, "no owner decision is pending") {
			t.Fatalf("why_no_approval = %q", why)
		}
		if !strings.Contains(visual, "Mason-12") {
			t.Fatalf("visual missing worker name:\n%s", visual)
		}
		for _, want := range []string{"1 (Add the shared template)", "2 (Wire the shared template into the form)"} {
			if !strings.Contains(visual, want) {
				t.Fatalf("visual missing covered task %q:\n%s", want, visual)
			}
		}
	})

	t.Run("single ungrouped task", func(t *testing.T) {
		phase := colony.Phase{ID: 1, Tasks: []colony.Task{
			{ID: strPtr("1"), Goal: "Fix the off-by-one error in the pager"},
		}}
		dispatch := codexBuildDispatch{
			Caste:     "builder",
			Name:      "Mason-1",
			TaskID:    "1",
			JobSource: coherentJobSourceSingle,
			JobReason: "task 1 is not connected to another selected task by a safe automatic grouping edge, so one builder owns its implementation without crossing a caste boundary",
		}

		result, visual := renderBuildFastPathSummary(phase, dispatch, fastPathDecision)

		if single, _ := result["single_task"].(bool); !single {
			t.Fatalf("expected single_task=true")
		}
		if got, _ := result["relationship"].(string); got != "one worker owns this one task" {
			t.Fatalf("relationship = %q, want the honest single-task reason, not the raw grouping-edge sentence", got)
		}
		if got, _ := result["benefit"].(string); got != "no grouping was needed" {
			t.Fatalf("benefit = %q", got)
		}
		if !strings.Contains(visual, "no grouping was needed") {
			t.Fatalf("visual must state no grouping was needed:\n%s", visual)
		}
		if strings.Contains(visual, "not connected to another selected task") {
			t.Fatalf("visual leaked the internal grouping-edge sentence instead of the honest single-task reason:\n%s", visual)
		}
	})
}

// TestFastPathSummaryIsNonBlocking pins the other half of D-12: the compact
// renderer never asks anything -- no question, no approval choice, no
// resume signal, and no waiver option (that stays exclusive to the full
// card, Requested=true).
func TestFastPathSummaryIsNonBlocking(t *testing.T) {
	phase := colony.Phase{ID: 1, Tasks: []colony.Task{{ID: strPtr("1"), Goal: "Fix the pager"}}}
	dispatch := codexBuildDispatch{Caste: "builder", Name: "Mason-1", TaskID: "1", JobSource: coherentJobSourceSingle}
	decision := buildCheckinDecision{
		Requested: false,
		Reason:    buildCheckinReasonOneWorkerFastPath,
		Why:       "no owner decision is pending, so dispatch continues",
	}

	result, visual := renderBuildFastPathSummary(phase, dispatch, decision)

	for _, forbidden := range []string{"Approve", "Adjust", "Waive", "Proceed with this team", "Trim optional workers", "Decline a required reviewer"} {
		if strings.Contains(visual, forbidden) {
			t.Fatalf("fast-path summary must never contain an approval prompt or choice (%q found):\n%s", forbidden, visual)
		}
	}
	if _, ok := result["waive_commands"]; ok {
		t.Fatalf("fast-path result must never carry a waive_commands field, it never offers one")
	}
	if reason, _ := result["checkin_reason"].(string); reason != string(buildCheckinReasonOneWorkerFastPath) {
		t.Fatalf("checkin_reason = %q", reason)
	}
}

// TestCheckinFlagConflictHasNoSideEffects pins D-14's fail-closed flag
// handling: --checkin combined with --no-checkin is refused before
// plan-only ever opens an attempt, writes a manifest, writes a checkpoint,
// or touches colony state.
func TestCheckinFlagConflictHasNoSideEffects(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	phase := checkinFixturePhase("Fix the pager", "Fix the off-by-one error in the pager", colony.PhaseModePrototype)
	setUpCheckinFixtureColony(t, dataDir, phase)

	before, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState before: %v", err)
	}
	_, _, hadAttemptBefore := loadLatestBuildAttempt(1)
	manifestPath := filepath.Join(dataDir, "build", "phase-1", "manifest.json")
	checkpointPath := filepath.Join(dataDir, "checkpoints", "pre-build-phase-1.json")
	beforeSnapshot := snapshotDataDirForTest(t, dataDir)

	rootCmd.SetArgs([]string{"build", "1", "--plan-only", "--checkin", "--no-checkin"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned an unexpected Go error: %v", err)
	}
	rootCmd.SetArgs([]string{})

	env := parseEnvelope(t, stderr.(*bytes.Buffer).String())
	if env["ok"] != false {
		t.Fatalf("expected an error envelope, got %#v", env)
	}
	errText := stringValue(env["error"])
	if !strings.Contains(errText, "--checkin") || !strings.Contains(errText, "--no-checkin") {
		t.Fatalf("conflict error must name both flags, got %q", errText)
	}

	after, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState after: %v", err)
	}
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	if string(beforeJSON) != string(afterJSON) {
		t.Fatalf("flag conflict mutated colony state:\nbefore: %s\nafter:  %s", beforeJSON, afterJSON)
	}

	_, _, hasAttemptAfter := loadLatestBuildAttempt(1)
	if hadAttemptBefore || hasAttemptAfter {
		t.Fatalf("flag conflict must never create a build attempt; before=%v after=%v", hadAttemptBefore, hasAttemptAfter)
	}
	if _, statErr := os.Stat(manifestPath); !os.IsNotExist(statErr) {
		t.Fatalf("flag conflict must never write a manifest file, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(checkpointPath); !os.IsNotExist(statErr) {
		t.Fatalf("flag conflict must never write a checkpoint file, stat err = %v", statErr)
	}

	afterSnapshot := snapshotDataDirForTest(t, dataDir)
	if len(beforeSnapshot) != len(afterSnapshot) {
		t.Fatalf("flag conflict changed the number of files under the data directory: before=%d after=%d", len(beforeSnapshot), len(afterSnapshot))
	}
	for path, beforeContent := range beforeSnapshot {
		afterContent, ok := afterSnapshot[path]
		if !ok {
			t.Fatalf("flag conflict deleted %s from the data directory", path)
		}
		if beforeContent != afterContent {
			t.Fatalf("flag conflict changed the contents of %s", path)
		}
	}
	for path := range afterSnapshot {
		if _, ok := beforeSnapshot[path]; !ok {
			t.Fatalf("flag conflict created a new file %s under the data directory", path)
		}
	}
}

// snapshotDataDirForTest walks dataDir and returns every regular file's
// repository-relative path mapped to its full byte content, so a caller can
// assert the ENTIRE data directory is byte-equivalent before and after an
// operation that must have zero side effects -- not just the handful of
// paths the test happens to already know about.
func snapshotDataDirForTest(t *testing.T, dataDir string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dataDir, path)
		if relErr != nil {
			return relErr
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		snapshot[rel] = string(content)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot data directory: %v", err)
	}
	return snapshot
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

// TestTeamCardNamesModelAndReasonForEveryWorker is D-02's visible half
// (196-CONTEXT.md): the reason an expensive role is expensive is shown on
// the pre-build team card, before anything is spawned. Every worker line
// names the model it runs on, and a worker on the expensive model shows the
// recorded reason beside it.
//
// The two facts are also fields on the returned result, not only text inside
// the rendered card: a wrapper that narrates this must read fields, never
// scrape prose.
func TestTeamCardNamesModelAndReasonForEveryWorker(t *testing.T) {
	t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "")

	manifest, dispatches := teamCheckinManifestFixture()
	result, visual := renderCeremonyTeamCheckin("build", manifest, dispatches)

	models, _ := result["models"].(map[string]string)
	if models == nil {
		t.Fatalf("result carries no models field — a wrapper would have to scrape the card text")
	}
	for _, caste := range []string{"builder", "measurer"} {
		if strings.TrimSpace(models[caste]) == "" {
			t.Errorf("result models[%q] is empty — every worker on the card names its model", caste)
		}
		if want := resolveCasteModel(caste); models[caste] != want {
			t.Errorf("result models[%q] = %q, want the resolved model %q", caste, models[caste], want)
		}
		if !strings.Contains(visual, resolveCasteModel(caste)) {
			t.Errorf("card never names %s's model %q.\ncard:\n%s", caste, resolveCasteModel(caste), visual)
		}
	}

	modelReasons, _ := result["model_reasons"].(map[string]string)
	if modelReasons == nil {
		t.Fatalf("result carries no model_reasons field")
	}
	want := casteModelReason("measurer")
	if want == "" {
		t.Fatal("the fixture's expensive worker records no reason — this test would then assert nothing")
	}
	if modelReasons["measurer"] != want {
		t.Errorf("result model_reasons[measurer] = %q, want the recorded reason %q", modelReasons["measurer"], want)
	}
	if !strings.Contains(visual, want) {
		t.Errorf("card never shows why the expensive worker is expensive.\nwant: %s\ncard:\n%s", want, visual)
	}
}

// TestCheapModelWorkerNeedsNoReasonOnTheCard is the other direction: a
// worker on the cheaper model shows its model and nothing else, because
// there is no expense to justify. A justification rendered for every worker
// would be noise, and would make the one that matters harder to see.
func TestCheapModelWorkerNeedsNoReasonOnTheCard(t *testing.T) {
	t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "")

	manifest, dispatches := teamCheckinManifestFixture()
	result, visual := renderCeremonyTeamCheckin("build", manifest, dispatches)

	modelReasons, _ := result["model_reasons"].(map[string]string)
	if got := modelReasons["builder"]; got != "" {
		t.Errorf("the cheap-model worker carries a model reason %q — there is nothing to justify", got)
	}

	// The builder's line still names its model, and carries no "why the
	// more expensive model" clause.
	for _, line := range strings.Split(visual, "\n") {
		if !strings.Contains(line, "Builder") {
			continue
		}
		if !strings.Contains(line, "sonnet") {
			t.Errorf("the cheap-model worker's line does not name its model: %q", line)
		}
		if strings.Contains(line, "more expensive model") {
			t.Errorf("the cheap-model worker's line justifies an expense it does not incur: %q", line)
		}
	}
}

// TestFastPathSummaryCarriesModelAndReason pins the surface that never
// pauses: a one-worker build shows the compact summary instead of the full
// card, so if the model and its reason only lived on the card the owner
// would never see them on exactly the builds that skip it.
func TestFastPathSummaryCarriesModelAndReason(t *testing.T) {
	t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "")

	decision := buildCheckinDecision{
		Requested: false,
		Reason:    buildCheckinReasonOneWorkerFastPath,
		Why:       "no owner decision is pending, so dispatch continues",
	}
	phase := colony.Phase{ID: 1, Tasks: []colony.Task{
		{ID: strPtr("1"), Goal: "Find why the export drops rows"},
	}}

	t.Run("expensive worker shows its reason", func(t *testing.T) {
		dispatch := codexBuildDispatch{
			Caste:     "tracker",
			Name:      "Trail-3",
			TaskID:    "1",
			JobSource: coherentJobSourceSingle,
		}
		result, visual := renderBuildFastPathSummary(phase, dispatch, decision)

		if got, _ := result["model"].(string); got != "opus" {
			t.Errorf("result model = %q, want opus", got)
		}
		want := casteModelReason("tracker")
		if want == "" {
			t.Fatal("tracker records no model reason — this test would then assert nothing")
		}
		if got, _ := result["model_reason"].(string); got != want {
			t.Errorf("result model_reason = %q, want %q", got, want)
		}
		if !strings.Contains(visual, "opus") {
			t.Errorf("fast-path summary never names the model:\n%s", visual)
		}
		if !strings.Contains(visual, want) {
			t.Errorf("fast-path summary never says why the expensive model was kept:\n%s", visual)
		}
	})

	t.Run("cheap worker names the model and justifies nothing", func(t *testing.T) {
		dispatch := codexBuildDispatch{
			Caste:     "builder",
			Name:      "Mason-1",
			TaskID:    "1",
			JobSource: coherentJobSourceSingle,
		}
		result, visual := renderBuildFastPathSummary(phase, dispatch, decision)

		if got, _ := result["model"].(string); got != "sonnet" {
			t.Errorf("result model = %q, want sonnet", got)
		}
		if got, _ := result["model_reason"].(string); got != "" {
			t.Errorf("result model_reason = %q, want empty for a cheap-model worker", got)
		}
		if !strings.Contains(visual, "sonnet") {
			t.Errorf("fast-path summary never names the model:\n%s", visual)
		}
		if strings.Contains(visual, "more expensive model") {
			t.Errorf("fast-path summary justifies an expense the worker does not incur:\n%s", visual)
		}
	})
}
