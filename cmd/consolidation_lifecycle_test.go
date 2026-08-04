package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// hasConsolidationPhaseEndEvent reports whether the real (non-dry-run)
// consolidation pipeline published its "consolidation.phase_end" event to
// the persisted event bus during this test -- the cleanest available signal
// that runPhaseEndConsolidation actually ran (RESEARCH.md's recommendation).
// Reuses readPersistedCeremonyEvents (cmd/ceremony_emitter_test.go).
func hasConsolidationPhaseEndEvent(t *testing.T) bool {
	t.Helper()
	for _, evt := range readPersistedCeremonyEvents(t) {
		if evt.Topic == "consolidation.phase_end" {
			return true
		}
	}
	return false
}

// seedConsolidationWiringFixture seeds a minimal but real, VALID consolidation
// fixture (an instinct plus an empty-but-parseable observations file) against
// the current store global, so a wiring test's "did consolidation actually
// run" assertion exercises the real mutating pipeline instead of a load
// failure or a no-op on missing files.
func seedConsolidationWiringFixture(t *testing.T) {
	t.Helper()
	if err := store.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: []colony.InstinctEntry{
		{ID: "inst_wiring_fixture", Trigger: "t", Action: "a", TrustScore: 0.05, TrustTier: "untrusted", Confidence: 0.2},
	}}); err != nil {
		t.Fatalf("seed instincts fixture: %v", err)
	}
	if err := store.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed observations fixture: %v", err)
	}
}

// captureStderrForConsolidationTest redirects os.Stderr for the duration of
// fn and returns everything written to it. Modelled on the identical local
// helper in cmd/hive_policy_test.go (TestHiveRuntimePolicyUnrecognizedWarns).
func captureStderrForConsolidationTest(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	fn()

	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copy stderr: %v", err)
	}
	return buf.String()
}

// TestRunPhaseEndConsolidationMutatesOnRealPath proves runPhaseEndConsolidation
// takes the real (mutating) path, exactly as consolidationPhaseEndCmd's own
// non-dry-run branch does -- mirroring TestConsolidationRealRunStillMutates'
// shape (cmd/consolidation_dryrun_test.go).
func TestRunPhaseEndConsolidationMutatesOnRealPath(t *testing.T) {
	saveGlobals(t)

	dataDir := seedConsolidationFixture(t)
	instincts := filepath.Join(dataDir, "instincts.json")
	before := hashFileForTest(t, instincts)

	summary := runPhaseEndConsolidation(1)

	if !summary.Ran {
		t.Fatalf("expected Ran == true on the real path, got summary: %+v", summary)
	}
	if after := hashFileForTest(t, instincts); after == before {
		t.Fatal("runPhaseEndConsolidation left instincts.json untouched; real path did not mutate")
	}
}

// TestRunPhaseEndConsolidationIsNonBlockingOnFailure asserts D-05: a
// consolidation failure never panics or exits, and is reported through the
// summary rather than propagated as an error the caller could act on.
func TestRunPhaseEndConsolidationIsNonBlockingOnFailure(t *testing.T) {
	saveGlobals(t)

	s, _ := newTestStore(t)
	store = s

	// instincts.json containing invalid JSON makes ConsolidationService.Run's
	// LoadJSON step fail; that failure lands in result.Errors rather than a
	// top-level error, which runPhaseEndConsolidation must still treat as a
	// non-blocking failure.
	instinctsPath := filepath.Join(s.BasePath(), "instincts.json")
	if err := os.WriteFile(instinctsPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("seed invalid instincts.json: %v", err)
	}

	var summary phaseEndConsolidationSummary
	stderr := captureStderrForConsolidationTest(t, func() {
		summary = runPhaseEndConsolidation(1)
	})

	if summary.Ran {
		t.Fatalf("expected Ran == false on failure, got summary: %+v", summary)
	}
	if summary.Reason == "" {
		t.Fatal("expected a non-empty Reason on failure")
	}
	if !strings.Contains(stderr, "phase advanced WITHOUT consolidation —") {
		t.Fatalf("expected unmissable D-05 stderr warning, got: %q", stderr)
	}
}

// TestRunPhaseEndConsolidationZeroState asserts D-06/D-07's zero-state
// contract: an empty but VALID store (files exist, parse cleanly, contain
// nothing) still runs cleanly and reports ZeroState() == true.
func TestRunPhaseEndConsolidationZeroState(t *testing.T) {
	saveGlobals(t)

	s, _ := newTestStore(t)
	store = s

	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: []colony.InstinctEntry{}}); err != nil {
		t.Fatalf("seed empty instincts.json: %v", err)
	}
	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed empty learning-observations.json: %v", err)
	}

	summary := runPhaseEndConsolidation(1)

	if !summary.Ran {
		t.Fatalf("expected Ran == true on an empty-but-valid store, got summary: %+v", summary)
	}
	if summary.PromotionCandidates != 0 {
		t.Errorf("expected PromotionCandidates == 0, got %d", summary.PromotionCandidates)
	}
	if summary.QueenEligible != 0 {
		t.Errorf("expected QueenEligible == 0, got %d", summary.QueenEligible)
	}
	if !summary.ZeroState() {
		t.Fatal("expected ZeroState() == true for an empty-but-valid store")
	}
}

// TestContinueAdvanceInvokesPhaseEndConsolidation proves D-04's default-path
// wiring: a durable phase advance through the default continue path invokes
// runPhaseEndConsolidation, observable via the consolidation.phase_end event
// the real (non-dry-run) pipeline publishes. Deleting the call site in
// cmd/codex_continue.go must make this test fail.
func TestContinueAdvanceInvokesPhaseEndConsolidation(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Wire phase-end consolidation on the default continue path"
	now := time.Now().UTC()
	taskOneID := "1.1"
	taskTwoID := "1.2"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Consolidation wiring (default path)",
					Description: "Prove runPhaseEndConsolidation fires beside captureContinueLearning",
					Status:      colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Implement the packet", Status: colony.TaskInProgress},
						{ID: &taskTwoID, Goal: "Verify the packet", Status: colony.TaskInProgress},
					},
				},
				{
					ID:     2,
					Name:   "Next slice",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Keep moving", Status: colony.TaskPending}},
				},
			},
		},
	})

	// A real fixture, not an empty store: proves consolidation actually ran
	// the mutating pipeline, not merely that a summary map was attached.
	seedConsolidationWiringFixture(t)

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-consolidation-1", Task: "Implement the packet", Status: "spawned", TaskID: taskOneID},
		{Stage: "wave", Wave: 1, Caste: "scout", Name: "Ranger-consolidation-1", Task: "Research the packet", Status: "spawned", TaskID: taskTwoID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-consolidation-1", Task: "Independent verification before advancement", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Consolidation wiring (default path)", goal, dispatches)

	result, _, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{})
	if err != nil {
		t.Fatalf("runCodexContinue returned error: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true (precondition for D-04 wiring), got %v", result)
	}

	if !hasConsolidationPhaseEndEvent(t) {
		t.Fatal("expected a consolidation.phase_end event after a default continue advance; runPhaseEndConsolidation call site may be missing")
	}
}

// TestExternalContinueAdvanceInvokesPhaseEndConsolidation proves D-04's
// external/finalize-path wiring: runPhaseEndConsolidation fires after
// advanceExternalContinue returns with err == nil (the stricter-correct
// placement per RESEARCH.md assumption A1, since PhaseCompleted is written
// INSIDE that call). Deleting the call site in
// cmd/codex_continue_finalize.go must make this test fail.
func TestExternalContinueAdvanceInvokesPhaseEndConsolidation(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Wire phase-end consolidation on the external finalize path"
	now := time.Now().UTC()
	taskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Consolidation wiring (external path)",
					Description: "Prove runPhaseEndConsolidation fires after advanceExternalContinue",
					Status:      colony.PhaseInProgress,
					Tasks:       []colony.Task{{ID: &taskID, Goal: "Advance durably", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Continue forward", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedConsolidationWiringFixture(t)

	seedContinueBuildPacket(t, dataDir, 1, "Consolidation wiring (external path)", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-consolidation-2", Task: "Advance durably", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-consolidation-2", Task: "Independent verification before advancement", Status: "completed"},
	})

	planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{HeavyFlag: true})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly: %v", err)
	}
	plan := planResult["continue_manifest"].(codexContinuePlanManifest)
	results := make([]codexContinueExternalDispatch, 0, len(plan.Dispatches))
	for _, dispatch := range plan.Dispatches {
		results = append(results, codexContinueExternalDispatch{
			Stage:   dispatch.Stage,
			Wave:    dispatch.Wave,
			Caste:   dispatch.Caste,
			Name:    dispatch.Name,
			Task:    dispatch.Task,
			TaskID:  dispatch.TaskID,
			Status:  "completed",
			Summary: dispatch.Name + " cleared consolidation wiring review",
		})
	}
	completion := codexExternalContinueCompletion{
		ContinueManifest: &plan,
		Dispatches:       results,
	}

	result, _, _, _, _, _, err := runCodexContinueFinalize(root, completion, false, 0, false)
	if err != nil {
		t.Fatalf("runCodexContinueFinalize: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true (precondition for D-04 wiring), got %v", result)
	}

	if !hasConsolidationPhaseEndEvent(t) {
		t.Fatal("expected a consolidation.phase_end event after an external continue-finalize advance; runPhaseEndConsolidation call site may be missing")
	}
}

// TestContinueWithoutAdvanceDoesNotConsolidate gives D-04's "on real advance
// only" half teeth: a continue run that fails gates (a blocked continue
// watcher) must produce no consolidation side effect at all. Moving the
// default-path call above the atomic COLONY_STATE.json write must make this
// test fail.
func TestContinueWithoutAdvanceDoesNotConsolidate(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	now := time.Now().UTC()
	goal := "Continue that does not advance must not consolidate"
	taskID := "1.1"
	state := colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Consolidation wiring (blocked path)",
			Status: colony.PhaseInProgress,
			Tasks:  []colony.Task{{ID: &taskID, Goal: "Should not advance", Status: colony.TaskPending}},
		}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	seedConsolidationWiringFixture(t)
	instinctsPath := filepath.Join(s.BasePath(), "instincts.json")
	before := hashFileForTest(t, instinctsPath)

	seedContinueBuildPacket(t, s.BasePath(), 1, "Consolidation wiring (blocked path)", goal, []codexBuildDispatch{{
		Stage:  "implementation",
		Caste:  "builder",
		Name:   "Mason-consolidation-3",
		TaskID: taskID,
		Task:   "Should not advance",
		Status: "completed",
	}})
	newCodexWorkerInvoker = func() codex.WorkerInvoker {
		return &continueWatcherTestInvoker{watcherStatus: "blocked", watcherSummary: "Continue watcher rejected the phase"}
	}

	result, _, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{})
	if err != nil {
		t.Fatalf("runCodexContinue returned error: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false for a blocked continue, got %v", result)
	}

	if hasConsolidationPhaseEndEvent(t) {
		t.Fatal("a continue that did not advance must not consolidate (D-04); the call site may have moved above the gate")
	}
	if after := hashFileForTest(t, instinctsPath); after != before {
		t.Fatal("a continue that did not advance mutated instincts.json; consolidation must not run without a durable advance")
	}
}
