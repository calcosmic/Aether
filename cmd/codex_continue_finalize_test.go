package cmd

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestBuildLearningContent(t *testing.T) {
	phase := colony.Phase{ID: 42, Name: "Test Phase"}

	// Test with no workers
	content := buildLearningContent(phase, nil)
	if !strings.Contains(content, "Phase 42: Test Phase") {
		t.Errorf("expected phase header, got: %s", content)
	}
	if !strings.Contains(content, "No workers completed successfully") {
		t.Errorf("expected no-workers message, got: %s", content)
	}

	// Test with completed worker containing findings and lessons
	workerFlow := []codexContinueWorkerFlowStep{
		{
			Name:            "Builder-1",
			Caste:           "builder",
			Status:          "completed",
			Task:            "Fix auth bug",
			Findings:        []codexReviewFinding{{Description: "Worker A found issue X"}},
			ReusableLessons: []string{"Always validate tokens before use"},
			Recommendations: []string{"Add rate limiting"},
			WeakSpots:       []string{"Token refresh logic"},
			EdgeCases:       []string{"Empty token string"},
		},
		{
			Name:   "Watcher-1",
			Caste:  "watcher",
			Status: "failed",
			Task:   "Run tests",
		},
	}

	content = buildLearningContent(phase, workerFlow)

	if !strings.Contains(content, "Workers completed: 1") {
		t.Errorf("expected 1 completed worker, got: %s", content)
	}
	if !strings.Contains(content, "Builder-1 (builder)") {
		t.Errorf("expected worker name, got: %s", content)
	}
	if !strings.Contains(content, "Worker A found issue X") {
		t.Errorf("expected finding text, got: %s", content)
	}
	if !strings.Contains(content, "Always validate tokens before use") {
		t.Errorf("expected reusable lesson, got: %s", content)
	}
	if !strings.Contains(content, "Add rate limiting") {
		t.Errorf("expected recommendation, got: %s", content)
	}
	if !strings.Contains(content, "Token refresh logic") {
		t.Errorf("expected weak spot, got: %s", content)
	}
	if !strings.Contains(content, "Empty token string") {
		t.Errorf("expected edge case, got: %s", content)
	}

	// Failed worker should NOT appear
	if strings.Contains(content, "Watcher-1") {
		t.Errorf("failed worker should not appear, got: %s", content)
	}
}

func TestBuildLearningContent_MultipleWorkers(t *testing.T) {
	phase := colony.Phase{ID: 10, Name: "Multi Worker Phase"}
	workerFlow := []codexContinueWorkerFlowStep{
		{
			Name:     "Builder-1",
			Caste:    "builder",
			Status:   "completed",
			Findings: []codexReviewFinding{{Description: "Finding 1"}},
		},
		{
			Name:            "Builder-2",
			Caste:           "builder",
			Status:          "completed",
			ReusableLessons: []string{"Lesson 2"},
		},
	}

	content := buildLearningContent(phase, workerFlow)
	if !strings.Contains(content, "Finding 1") {
		t.Errorf("expected finding from worker 1, got: %s", content)
	}
	if !strings.Contains(content, "Lesson 2") {
		t.Errorf("expected lesson from worker 2, got: %s", content)
	}
}

// TestContinueFinalizeRefusesToAdvanceOnSupersededState is the finalize-path
// twin of TestContinueStaleStateDoesNotOverwritePausedState
// (cmd/codex_continue_test.go): a concurrent process (an operator running
// `aether pause-colony`, or any other writer) mutates COLONY_STATE.json's
// Paused flag during the window between continue-finalize's own state read
// (validateExternalContinueState, at the very top of runCodexContinueFinalize)
// and its atomic commit (advanceExternalContinue, at the very end) -- exactly
// the class of race 188-CONTEXT.md's D-04/D-05 exist to close.
// validateExternalContinueState never checks Paused (only
// validateRuntimeStateStillCurrent does), so nothing earlier in the flow
// would catch this on its own -- before this fix, advanceExternalContinue had
// no supersession check at all and would silently clobber the concurrent
// write. This test seeds the mutation directly via the store, immediately
// before calling runCodexContinueFinalize, mirroring the "writing a competing
// state change directly via the store, simulating a concurrent process"
// technique from 188-02-PLAN.md Task 3 -- there is no live worker dispatch
// inside the finalize path to hook mid-call the way the default path's sibling
// test does.
func TestContinueFinalizeRefusesToAdvanceOnSupersededState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Continue finalize does not advance over pause"
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
					ID:     1,
					Name:   "Pause during continue-finalize",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Pause while finalize runs", Status: colony.TaskCompleted}},
				},
				{
					ID:     2,
					Name:   "Must not be readied",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Wait for explicit resume", Status: colony.TaskPending}},
				},
			},
		},
	})
	seedContinueBuildPacket(t, dataDir, 1, "Pause during continue-finalize", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-supersede", Task: "Pause while finalize runs", Status: "completed", TaskID: taskID},
	})

	planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{LightFlag: true, SkipWatchers: true})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly returned error: %v", err)
	}
	plan, ok := planResult["continue_manifest"].(codexContinuePlanManifest)
	if !ok {
		t.Fatalf("expected continue_manifest in result, got %#v", planResult["continue_manifest"])
	}

	// Simulate a concurrent process pausing the colony after continue-finalize
	// would have read state, but before it commits -- the external-dispatch
	// completion window this criterion targets is exactly this kind of gap.
	var pausedState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &pausedState); err != nil {
		t.Fatalf("load state for mutation: %v", err)
	}
	pausedAt := time.Now().UTC().Format(time.RFC3339)
	pausedState.Paused = true
	pausedState.PausedAt = &pausedAt
	if err := store.SaveJSON("COLONY_STATE.json", pausedState); err != nil {
		t.Fatalf("write competing state change: %v", err)
	}

	result, _, _, _, _, _, err := runCodexContinueFinalize(root, codexExternalContinueCompletion{
		ContinueManifest: &plan,
		Dispatches:       []codexContinueExternalDispatch{},
	}, false, 0, false)
	if err != nil {
		t.Fatalf("runCodexContinueFinalize returned an error instead of a blocked result: %v", err)
	}
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Errorf("result[blocked] = %v, want true (result: %#v)", result["blocked"], result)
	}
	if superseded, _ := result["superseded"].(bool); !superseded {
		t.Errorf("result[superseded] = %v, want true (result: %#v)", result["superseded"], result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Errorf("result[advanced] = %v, want false", result["advanced"])
	}

	var after colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &after); err != nil {
		t.Fatalf("reload state after finalize: %v", err)
	}
	if !after.Paused {
		t.Fatalf("expected the competing Paused=true write to survive; got Paused=%v", after.Paused)
	}
	if after.Plan.Phases[0].Status == colony.PhaseCompleted {
		t.Fatalf("phase 1 was advanced to completed despite the superseded (paused) state")
	}
	if after.CurrentPhase != 1 {
		t.Fatalf("current phase = %d, want 1 (unchanged)", after.CurrentPhase)
	}
	if after.Plan.Phases[1].Status == colony.PhaseReady {
		t.Fatalf("superseded finalize readied phase 2")
	}
}
