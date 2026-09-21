package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	resume, err := resolvePlanningStageResumeForTest(t, root)
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

	resume, err := resolvePlanningStageResumeForTest(t, root)
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

// TestParkedRunIsFoundWithNoIterationPointer covers the real downstream
// colony that reported this defect: a run stopped at route_running with no
// iteration-state.json anywhere on disk. Keying the resume solely off that
// pointer left exactly the stuck colonies this feature exists to rescue still
// stuck, and the original fixture hid it by always writing the pointer.
func TestParkedRunIsFoundWithNoIterationPointer(t *testing.T) {
	root, routeManifest := planningStageResumeTestParkedRun(t)

	// Delete the convenience pointer, leaving only the durable stage files --
	// exactly the shape the reporting colony was in.
	if err := os.Remove(filepath.Join(store.BasePath(), planningIterationStateRel)); err != nil {
		t.Fatalf("remove iteration state: %v", err)
	}

	resume, err := resolvePlanningStageResumeForTest(t, root)
	if err != nil {
		t.Fatalf("resolve planning stage resume: %v", err)
	}
	if resume == nil {
		t.Fatal("a parked run with no iteration pointer was not found: the colony stays stuck exactly as reported")
	}
	if resume.RunID != routeManifest.RunID {
		t.Fatalf("discovered run = %q, want the parked run %q", resume.RunID, routeManifest.RunID)
	}
	if resume.Caste != planningStageCasteRouteSetter {
		t.Fatalf("discovered caste = %q, want Route-Setter", resume.Caste)
	}
}

// TestAmbiguousParkedRunsAreNotGuessed locks the refusal half: with no pointer
// and two runs parked at route_running, the planner must NOT pick one. Starting
// the wrong run is the stray-run damage this path exists to prevent.
func TestAmbiguousParkedRunsAreNotGuessed(t *testing.T) {
	root, routeManifest := planningStageResumeTestParkedRun(t)
	if err := os.Remove(filepath.Join(store.BasePath(), planningIterationStateRel)); err != nil {
		t.Fatalf("remove iteration state: %v", err)
	}

	// Clone the parked run's stage file under a second run id, so two runs
	// both claim to be waiting on Route-Setter.
	src := filepath.Join(store.BasePath(), "planning", routeManifest.RunID, "stage-state.json")
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read parked stage state: %v", err)
	}
	twinID := routeManifest.RunID + "-twin"
	twin := filepath.Join(store.BasePath(), "planning", twinID)
	if err := os.MkdirAll(twin, 0o755); err != nil {
		t.Fatalf("make twin run dir: %v", err)
	}
	// The twin must be internally consistent -- its own run_id matching its
	// own directory -- or the loader rejects it and the ambiguity never
	// actually occurs. A fixture in a shape the runtime cannot produce would
	// make this test pass without testing anything.
	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode parked stage state: %v", err)
	}
	decoded["run_id"] = twinID
	reencoded, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("re-encode twin stage state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(twin, "stage-state.json"), reencoded, 0o644); err != nil {
		t.Fatalf("write twin stage state: %v", err)
	}
	// Prove the fixture is real: the twin must genuinely load as parked.
	if twinState, err := loadPlanningStageState(root, twinID); err != nil || twinState.Stage != planningStageRouteRunning {
		t.Fatalf("twin fixture is not a loadable parked run (stage=%v err=%v); the ambiguity being tested would not occur", twinState.Stage, err)
	}

	resume, err := resolvePlanningStageResumeForTest(t, root)
	if err != nil {
		t.Fatalf("resolve planning stage resume: %v", err)
	}
	if resume != nil {
		t.Fatalf("two parked runs were ambiguous but one was chosen anyway (%q); starting the wrong run is the damage this prevents", resume.RunID)
	}
}

// resolvePlanningStageResumeForTest runs the resolver the way production does:
// inside an already-open planning session. Calling it with its own fresh
// session instead would never hold the lock first, which is exactly how the
// self-deadlock below escaped every test until a real colony hung on it.
func resolvePlanningStageResumeForTest(t *testing.T, root string) (*planningStageResume, error) {
	t.Helper()
	var resume *planningStageResume
	err := withPlanningMutationSession(root, "test-resolve-resume", func(session *planningMutationSession) error {
		var inner error
		// Resolve against the specification the fixture's run was made for
		// (planningStageResumeTestParkedRun seeds it from the same helper), so
		// these tests keep asking what they always asked: is the parked run
		// found? TestParkedRunAgainstASupersededSpecificationIsNotResumed
		// covers the other half.
		resume, inner = resolvePlanningStageResume(session, planningStageTestState(planningStageScoutReady).Specification)
		return inner
	})
	return resume, err
}

// TestResumeResolvesInsideAHeldLock is the regression test for a hang, not a
// wrong answer. The plan-only path already holds the planning lock when it asks
// what to resume; the first version of this resolver called loaders that each
// open a session of their own, so the process waited on a lock it was already
// holding. Against a real colony `aether plan` sat forever burning no CPU and
// printing nothing -- the worst possible failure shape.
//
// The bounded wait is the assertion: a deadlock fails this in seconds with a
// clear message instead of hanging the suite until the global test timeout.
func TestResumeResolvesInsideAHeldLock(t *testing.T) {
	root, _ := planningStageResumeTestParkedRun(t)

	done := make(chan error, 1)
	go func() {
		_, err := resolvePlanningStageResumeForTest(t, root)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("resolve inside a held planning lock: %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("resolving a resume deadlocked while the caller already held the planning lock: the process waits on a lock it is holding, which is how `aether plan` hung against a real colony")
	}
}

// planningStageResumeTestSecondScoutPass drives a parked run one step further
// than the route fixture: through the Route-Setter pass, so the run lands at
// scout_running with a Scout that a Route-Setter pass authorized. This is the
// state a real colony reached after its first pass scored below target and the
// loop asked for a second research round.
func planningStageResumeTestSecondScoutPass(t *testing.T) (string, string, int) {
	t.Helper()
	// Use the established route-stage fixture: it builds an approved
	// specification, drives a real Scout pass, and hands back a route manifest
	// with a result the runtime genuinely accepts. Driving the real coordinator
	// is what makes the resulting scout_running state a shape the runtime can
	// actually produce, rather than one hand-written to satisfy the assertion.
	root, routeManifest, routeResult := planningRouteStageTestFixture(t)

	coordinated, err := coordinatePlanningRouteStage(root, routeManifest, planningRouteStageTestBytes(t, routeResult))
	if err != nil {
		t.Fatalf("coordinate Route-Setter pass: %v", err)
	}
	if coordinated.ScoutDispatch == nil {
		t.Skip("this route pass did not ask for a second research round; nothing to exercise")
	}
	state, err := loadPlanningStageState(root, routeManifest.RunID)
	if err != nil {
		t.Fatalf("load stage state after route pass: %v", err)
	}
	if state.Stage != planningStageScoutRunning {
		t.Skipf("run landed at %q rather than scout_running", state.Stage)
	}
	// Prove the fixture really is the reported shape before asserting on it.
	if _, err := loadPlanningRouteScoutDispatch(root, state.RunID, state.Pass); err != nil {
		t.Fatalf("second-round Scout carries no route-authored authorization (%v); the fixture is not the reported state", err)
	}
	return root, routeManifest.RunID, state.Pass
}

// TestSecondRoundScoutIsResumable is the regression test for the stall one step
// past the first fix: a run that scored below target, moved to scout_running for
// a second research round, and could not be picked up because discovery counted
// only route_running runs. Rerunning would have started a brand-new run and
// abandoned the committed pass-1 work.
func TestSecondRoundScoutIsResumable(t *testing.T) {
	root, runID, pass := planningStageResumeTestSecondScoutPass(t)

	// Remove the pointer so discovery is the only path, matching the colony.
	_ = os.Remove(filepath.Join(root, ".aether", "data", planningIterationStateRel))

	resume, err := resolvePlanningStageResumeForTest(t, root)
	if err != nil {
		t.Fatalf("resolve planning stage resume: %v", err)
	}
	if resume == nil {
		t.Fatalf("run %s parked at scout_running pass %d was not resumable: rerunning would abandon the committed first pass", runID, pass)
	}
	if resume.RunID != runID {
		t.Fatalf("resumed run = %q, want the parked run %q", resume.RunID, runID)
	}
	if resume.Caste != planningStageCasteScout {
		t.Fatalf("resumed caste = %q, want Scout for a second research round", resume.Caste)
	}
}

// TestAbandonedFirstPassScoutIsNotResumed is the other half: a run whose Scout
// stage was merely opened, with no Route-Setter pass having authorized it, must
// never be mistaken for work in progress. Two such strays sat beside the real
// run on the reporting colony.
func TestAbandonedFirstPassScoutIsNotResumed(t *testing.T) {
	binding := bindCommandTestRepository(t)
	root := binding.Root
	seedState := planningStageTestState(planningStageScoutReady)
	_, scoutManifest, _ := planningScoutStageTestFixtureInRoot(t, root, 1, seedState.Specification)

	state, err := loadPlanningStageState(root, scoutManifest.RunID)
	if err != nil {
		t.Fatalf("load stage state: %v", err)
	}
	if state.Stage != planningStageScoutRunning {
		t.Fatalf("fixture stage = %q, want scout_running to exercise the stray case", state.Stage)
	}
	if _, err := loadPlanningRouteScoutDispatch(root, state.RunID, state.Pass); err == nil {
		t.Fatal("fixture unexpectedly carries a route-authored Scout authorization; it is not a stray")
	}

	resume, err := resolvePlanningStageResumeForTest(t, root)
	if err != nil {
		t.Fatalf("resolve planning stage resume: %v", err)
	}
	if resume != nil {
		t.Fatalf("an abandoned first-pass Scout run was treated as resumable (%q); it holds no authorized work", resume.RunID)
	}
}
