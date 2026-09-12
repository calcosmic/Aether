package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planningStageResumeTestParkedRun drives a real Scout pass to the
// route_running boundary inside a bound test repository, and records the
// iteration state that names it as the in-flight run -- exactly the state a
// colony is left in after `aether plan-finalize` accepts a Scout result.
func planningStageResumeTestParkedRun(t *testing.T) (string, planningStageManifest) {
	t.Helper()
	binding := bindCommandTestRepository(t)
	root := binding.Root

	seedState := planningStageTestState(planningStageScoutReady)
	_, scoutManifest, scoutResult := planningScoutStageTestFixtureInRoot(t, root, 1, seedState.Specification)

	coordinated, err := coordinatePlanningScoutStage(root, scoutManifest, planningScoutStageTestBytes(t, scoutResult))
	if err != nil {
		t.Fatalf("coordinate Scout stage: %v", err)
	}
	if coordinated.RouteDispatch == nil {
		t.Fatal("Scout pass did not authorize Route-Setter; fixture cannot exercise the parked-run path")
	}

	// The iteration state is how a later `aether plan` finds the run at all.
	if err := store.SaveJSON(planningIterationStateRel, codexPlanIterationState{
		PlanningRunID: scoutManifest.RunID,
		Goal:          "resume the parked planning run",
		Root:          root,
		LastIteration: 1,
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("save planning iteration state: %v", err)
	}
	return root, coordinated.RouteDispatch.Manifest
}

// TestParkedRouteRunResumesInsteadOfStartingASecondRun is the regression test
// for the defect that stopped a staged plan from ever finishing: a run parked
// at route_running was unreachable, because the planner always prepared a
// fresh Scout stage. Re-running produced a stray SECOND run at the Scout stage
// while the Route-Setter stage manifest the first run had already issued had
// nowhere to be submitted.
//
// This asserts the planner now reports the parked run's own Route-Setter
// dispatch. It fails the moment resolvePlanningStageResume stops recognising a
// route_running run, or starts pointing at a different run.
func TestParkedRouteRunResumesInsteadOfStartingASecondRun(t *testing.T) {
	root, routeManifest := planningStageResumeTestParkedRun(t)

	resume, err := resolvePlanningStageResume(root)
	if err != nil {
		t.Fatalf("resolve planning stage resume: %v", err)
	}
	if resume == nil {
		t.Fatal("a run parked at route_running reported nothing to resume: the planner would start a second run beside it")
	}
	if resume.Caste != planningStageCasteRouteSetter {
		t.Fatalf("resumed caste = %q, want Route-Setter: the parked run is waiting on Route-Setter, not anyone else", resume.Caste)
	}
	if resume.RunID != routeManifest.RunID {
		t.Fatalf("resumed run = %q, want the parked run %q: a different run id is the stray-run bug itself", resume.RunID, routeManifest.RunID)
	}
	if resume.Manifest.ID != routeManifest.ID || resume.Manifest.ContentHash != routeManifest.ContentHash {
		t.Fatalf("resumed stage manifest = %s/%s, want the exact issued Route-Setter manifest %s/%s",
			resume.Manifest.ID, resume.Manifest.ContentHash, routeManifest.ID, routeManifest.ContentHash)
	}
}

// TestResumedRouteManifestSatisfiesTheFinalizerContract is the other half of
// the same defect. Producing a manifest is worthless if the finalizer refuses
// it, so this asserts the rebuilt envelope satisfies every binding rule
// runCodexRouteStageFinalizeInSession enforces before it will accept a
// Route-Setter result. Each assertion names the finalizer rule it stands for;
// breaking any single copied field turns exactly one of them red.
func TestResumedRouteManifestSatisfiesTheFinalizerContract(t *testing.T) {
	root, routeManifest := planningStageResumeTestParkedRun(t)

	resume, err := resolvePlanningStageResume(root)
	if err != nil || resume == nil {
		t.Fatalf("resolve planning stage resume: resume=%v err=%v", resume, err)
	}

	goal := "resume the parked planning run"
	state := colony.ColonyState{Goal: &goal}
	manifest, err := buildPlanningStageResumeManifest(
		root, state, *resume, colony.GranularitySprint,
		"balanced", "standard", "standard",
		codexSurveyContext{}, "", codexPlanOptions{}, time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("build resumed manifest: %v", err)
	}

	// "plan_manifest must come from `aether plan --plan-only` or an
	// agent-delegate planning response"
	if manifest.DispatchMode != "plan-only" || !manifest.RequiresFinalizer {
		t.Fatalf("dispatch_mode=%q requires_finalizer=%v, want plan-only and true", manifest.DispatchMode, manifest.RequiresFinalizer)
	}
	// "Route-Setter completion requires the exact stage_manifest"
	if manifest.StageManifest == nil {
		t.Fatal("resumed manifest carries no stage_manifest; the finalizer refuses it outright")
	}
	if manifest.StageManifest.ExpectedCaste != planningStageCasteRouteSetter {
		t.Fatalf("stage_manifest caste = %q, want Route-Setter", manifest.StageManifest.ExpectedCaste)
	}
	// "plan_manifest does not match the exact Route-Setter stage run, pass,
	// preset, or base plan revision"
	if manifest.PlanningRunID != routeManifest.RunID {
		t.Fatalf("planning_run_id = %q, want %q", manifest.PlanningRunID, routeManifest.RunID)
	}
	if manifest.Iteration != routeManifest.Pass {
		t.Fatalf("iteration = %d, want %d", manifest.Iteration, routeManifest.Pass)
	}
	if manifest.SelectedPreset != routeManifest.Preset {
		t.Fatalf("selected_preset = %q, want %q", manifest.SelectedPreset, routeManifest.Preset)
	}
	if manifest.BaseRevisionID != routeManifest.BasePlanRevisionID {
		t.Fatalf("base_revision_id = %q, want %q", manifest.BaseRevisionID, routeManifest.BasePlanRevisionID)
	}
	if manifest.BasePlanStateHash != routeManifest.BasePlanRevisionHash {
		t.Fatalf("base_plan_state_hash = %q, want %q", manifest.BasePlanStateHash, routeManifest.BasePlanRevisionHash)
	}
	// "Route-Setter plan_manifest must contain exactly one Route-Setter
	// dispatch" and "Route-Setter dispatch does not bind the exact
	// stage_manifest"
	if len(manifest.Dispatches) != 1 || len(manifest.ExpectedWorkers) != 1 {
		t.Fatalf("dispatches=%d expected_workers=%d, want exactly one of each", len(manifest.Dispatches), len(manifest.ExpectedWorkers))
	}
	for label, dispatch := range map[string]codexPlanningDispatch{
		"dispatch":        manifest.Dispatches[0],
		"expected_worker": manifest.ExpectedWorkers[0],
	} {
		if !strings.EqualFold(dispatch.Caste, string(planningStageCasteRouteSetter)) {
			t.Fatalf("%s caste = %q, want route_setter", label, dispatch.Caste)
		}
		if dispatch.StageManifest == nil {
			t.Fatalf("%s carries no stage_manifest binding", label)
		}
		if dispatch.StageManifest.ID != routeManifest.ID || dispatch.StageManifest.ContentHash != routeManifest.ContentHash {
			t.Fatalf("%s binds %s/%s, want the exact issued manifest %s/%s",
				label, dispatch.StageManifest.ID, dispatch.StageManifest.ContentHash, routeManifest.ID, routeManifest.ContentHash)
		}
	}
	// "validateFinalizerManifestRoot"
	if err := validateFinalizerManifestRoot("plan_manifest", manifest.Root, root); err != nil {
		t.Fatalf("resumed manifest root is not the repository root: %v", err)
	}
	// "validateCodexPlanManifestFreshness"
	if err := validateCodexPlanManifestFreshness(manifest, time.Now().UTC()); err != nil {
		t.Fatalf("resumed manifest is not fresh: %v", err)
	}
}

// TestStagedFinalizeCloseoutSucceedsAfterCommit is the regression test for a
// finalize that committed its work and then reported failure.
//
// A staged Scout finalize returns a committed result carrying no lifecycle
// projection. Before the projection guard, that result reached
// applyLifecycleCloseout, failed its "requires one lifecycle projection"
// check, and exited non-zero -- after the receipt and the next-stage
// authorization had already been written. This drives the real committed
// result through the real closeout and fails if it is refused.
func TestStagedFinalizeCloseoutSucceedsAfterCommit(t *testing.T) {
	binding := bindCommandTestRepository(t)
	root := binding.Root

	seedState := planningStageTestState(planningStageScoutReady)
	_, scoutManifest, scoutResult := planningScoutStageTestFixtureInRoot(t, root, 1, seedState.Specification)

	result, err := runCodexScoutStageFinalize(root, codexPlanManifest{
		Root:              root,
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		PlanningRunID:     scoutManifest.RunID,
		Iteration:         scoutManifest.Pass,
		SelectedPreset:    scoutManifest.Preset,
		BaseRevisionID:    scoutManifest.BasePlanRevisionID,
		BasePlanStateHash: scoutManifest.BasePlanRevisionHash,
		StageManifest:     &scoutManifest,
		Dispatches:        planningStageResumeTestScoutDispatches(scoutManifest),
		ExpectedWorkers:   planningStageResumeTestScoutDispatches(scoutManifest),
	}, codexExternalPlanCompletion{ScoutResult: planningScoutStageTestBytes(t, scoutResult)})
	if err != nil {
		t.Fatalf("staged Scout finalize: %v", err)
	}
	if _, ok := lifecycleCloseoutProjectionFromValue(result[lifecycleProjectionKey]); ok {
		t.Fatal("fixture is not exercising the defect: the staged result already carries a projection")
	}

	if err := applyPlanFinalizeCloseout(result); err != nil {
		t.Fatalf("closeout refused a committed staged finalize: %v\nthis is the commit-then-report-failure defect", err)
	}
	if strings.TrimSpace(stringValue(result[nextActionCommandKey])) == "" {
		t.Fatal("closeout succeeded but named no next command")
	}
}

func planningStageResumeTestScoutDispatches(manifest planningStageManifest) []codexPlanningDispatch {
	stage := manifest
	return []codexPlanningDispatch{{
		Caste:         string(planningStageCasteScout),
		Stage:         string(planningStageScoutRunning),
		TaskID:        stage.AuthorizationID,
		StageManifest: &stage,
	}}
}
