package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Phase/Task Count Reconciliation Tests
//
// These tests demonstrate bugs where completed phases can carry incomplete
// task records, or where manifest task counts diverge from colony state.
// Task 2.2 will implement the fix; the BUG CONFIRMED tests should FAIL.
// ---------------------------------------------------------------------------

// TestApplyCodexBuildState_DoesNotCompletePriorPhaseWithIncompleteTasks is a
// focused unit test that applyCodexBuildState should not mark a prior phase
// as completed when its tasks are not all completed.
//
// BUG: cmd/codex_build.go:633-634 unconditionally marks prior phases as
// completed without calling phaseTasksAllCompleted() which exists at line 1549.
func TestApplyCodexBuildState_DoesNotCompletePriorPhaseWithIncompleteTasks(t *testing.T) {
	taskOneID := "1.1"
	taskTwoID := "1.2"
	phaseTwoTaskID := "2.1"

	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         func() *string { g := "test"; return &g }(),
		State:        colony.StateREADY,
		CurrentPhase: 0,
		ColonyDepth:  "light",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Prior phase",
					Status: colony.PhaseInProgress, // Not completed yet
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Pending task", Status: colony.TaskPending},
						{ID: &taskTwoID, Goal: "In progress task", Status: colony.TaskInProgress, DependsOn: []string{taskOneID}},
					},
				},
				{
					ID:     2,
					Name:   "Active phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &phaseTwoTaskID, Goal: "Active work", Status: colony.TaskPending}},
				},
			},
		},
	}

	startedAt := time.Now().UTC()
	applyCodexBuildState(&state, 2, startedAt, nil, colony.VerificationDepthLight)

	// The prior phase (phase 1) should NOT be marked completed because its tasks
	// are still pending/in_progress.
	if state.Plan.Phases[0].Status == colony.PhaseCompleted {
		t.Errorf("BUG CONFIRMED: applyCodexBuildState marked phase 1 as completed despite having incomplete tasks (task %s=%s, task %s=%s)",
			taskOneID, state.Plan.Phases[0].Tasks[0].Status,
			taskTwoID, state.Plan.Phases[0].Tasks[1].Status,
		)
	}

	// Phase 2 should be in progress
	if state.Plan.Phases[1].Status != colony.PhaseInProgress {
		t.Errorf("phase 2 status = %s, want in_progress", state.Plan.Phases[1].Status)
	}
}

// TestValidateCodexBuildState_DetectsCompletedPhaseWithPendingTasks verifies
// that validateCodexBuildState should catch the case where a prior phase is
// marked completed but has pending tasks.
//
// BUG: validateCodexBuildState (line 539-543) only checks phase.Status,
// not individual task completion.
func TestValidateCodexBuildState_DetectsCompletedPhaseWithPendingTasks(t *testing.T) {
	taskOneID := "1.1"
	taskTwoID := "1.2"
	phaseTwoTaskID := "2.1"

	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         func() *string { g := "test"; return &g }(),
		State:        colony.StateREADY,
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Completed phase with pending tasks",
					Status: colony.PhaseCompleted, // Phase says completed...
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Pending task", Status: colony.TaskPending},
						{ID: &taskTwoID, Goal: "In progress task", Status: colony.TaskInProgress, DependsOn: []string{taskOneID}},
					},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &phaseTwoTaskID, Goal: "Next work", Status: colony.TaskPending}},
				},
			},
		},
	}

	// validateCodexBuildState should detect that phase 1 is completed but has
	// incomplete tasks and return an error.
	err := validateCodexBuildState(state, 2, nil, false)
	if err == nil {
		t.Error("BUG CONFIRMED: validateCodexBuildState accepted a completed phase with pending/in_progress tasks")
	} else {
		// Check the error mentions the divergence
		if !strings.Contains(err.Error(), "task") && !strings.Contains(err.Error(), "incomplete") {
			t.Errorf("validateCodexBuildState rejected but error does not mention task divergence: %v", err)
		}
		t.Logf("validateCodexBuildState correctly detected divergence: %v", err)
	}
}

// TestBuildState_PriorPhaseIncompleteTasks demonstrates that validateCodexBuildState
// correctly rejects building phase N when phase N-1 is not yet completed.
// This is the EXPECTED behavior (not a bug) — it validates the existing guard works.
func TestBuildState_PriorPhaseIncompleteTasks(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Prior phase has incomplete tasks"
	phaseOneTaskID := "1.1"
	phaseOneSecondTaskID := "1.2"
	phaseTwoTaskID := "2.1"

	// Phase 1 is NOT completed yet (status is still in_progress) and its tasks
	// are pending/in_progress. Phase 2 should NOT be allowed to proceed.
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
		ColonyDepth:  "light",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Incomplete prior phase",
					Status: colony.PhaseInProgress, // NOT completed
					Tasks: []colony.Task{
						{ID: &phaseOneTaskID, Goal: "Pending task one", Status: colony.TaskPending},
						{ID: &phaseOneSecondTaskID, Goal: "In progress task two", Status: colony.TaskInProgress, DependsOn: []string{phaseOneTaskID}},
					},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &phaseTwoTaskID, Goal: "Next work", Status: colony.TaskPending}},
				},
			},
		},
	})

	// Attempting to build phase 2 when phase 1 is not completed should fail
	_, err := runCodexBuild(root, 2, nil, true)
	if err == nil {
		t.Fatal("expected build phase 2 to fail when phase 1 is not completed")
	}
	if !strings.Contains(err.Error(), "not complete") {
		t.Fatalf("error = %q, want phase-not-complete rejection", err.Error())
	}

	// Verify the state was NOT corrupted
	var state colony.ColonyState
	if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr != nil {
		t.Fatalf("failed to reload state: %v", loadErr)
	}
	if state.Plan.Phases[0].Status == colony.PhaseCompleted {
		t.Error("BUG: validateCodexBuildState should have caught incomplete prior phase but it was marked completed")
	}
}

// TestBuildFinalize_DivergentManifestTaskCount demonstrates that build-finalize
// does not detect when a manifest claims fewer tasks than the colony state phase.
//
// BUG: validateBuildManifestTaskSetForPhase checks task ID sets match, but when
// the manifest has fewer tasks than the phase, the allowMissingTasks=true path
// in finalize can skip the validation entirely.
func TestBuildFinalize_DivergentManifestTaskCount(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Manifest task count divergence"
	taskOneID := "1.1"
	taskTwoID := "1.2"

	// Phase has 2 tasks in colony state
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "standard",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Manifest divergence",
				Status: colony.PhaseReady,
				Tasks: []colony.Task{
					{ID: &taskOneID, Goal: "First task", Status: colony.TaskPending},
					{ID: &taskTwoID, Goal: "Second task", Status: colony.TaskPending, DependsOn: []string{taskOneID}},
				},
			}},
		},
	})

	// Create a plan-only manifest with only 1 task (divergent from colony state's 2)
	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnly returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)

	// Tamper with the manifest: remove one task to create divergence
	if len(manifest.Tasks) >= 2 {
		manifest.Tasks = manifest.Tasks[:1]
	}

	// Create completion with a worker result
	dispatchResults := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		dispatchResults = append(dispatchResults, codexExternalBuildWorkerResult{
			Stage:         dispatch.Stage,
			Wave:          dispatch.Wave,
			ExecutionWave: normalizedDispatchWave(dispatch),
			Caste:         dispatch.Caste,
			Name:          dispatch.Name,
			TaskID:        dispatch.TaskID,
			Status:        "completed",
			Summary:       dispatch.Name + " completed",
			FilesModified: []string{"main.go"},
		})
	}

	completion := codexExternalBuildCompletion{
		DispatchManifest: &manifest,
		Dispatches:       dispatchResults,
	}
	completionData, err := json.MarshalIndent(completion, "", "  ")
	if err != nil {
		t.Fatalf("marshal completion: %v", err)
	}
	completionPath := filepath.Join(root, "completion.json")
	if err := os.WriteFile(completionPath, completionData, 0644); err != nil {
		t.Fatalf("write completion: %v", err)
	}

	// Attempt to finalize — the manifest has 1 task but colony state has 2.
	// This should be detected as a divergence.
	rootCmd.SetArgs([]string{"build-finalize", "1", "--completion-file", completionPath})
	executeErr := rootCmd.Execute()

	if executeErr == nil {
		// BUG: finalize accepted divergent task counts
		var state colony.ColonyState
		if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr != nil {
			t.Fatalf("failed to reload state: %v", loadErr)
		}

		manifestTaskCount := len(manifest.Tasks)
		stateTaskCount := len(state.Plan.Phases[0].Tasks)
		if manifestTaskCount != stateTaskCount {
			t.Errorf("BUG CONFIRMED: finalize accepted manifest with %d tasks but colony state phase has %d tasks (divergence not detected)",
				manifestTaskCount, stateTaskCount)
		}
	} else {
		// Finalize rejected the divergence — check it's for the right reason
		errOutput := stderr.(*bytes.Buffer).String()
		if !strings.Contains(errOutput, "task") && !strings.Contains(executeErr.Error(), "task") {
			t.Fatalf("finalize rejected but for wrong reason: %v", executeErr)
		}
		t.Logf("finalize correctly detected task count mismatch: %v", executeErr)
	}
}

// TestBuildFinalize_ManifestTaskCountMismatchWithState demonstrates the specific
// case where a build manifest has MORE tasks than the colony state phase, and
// finalize should detect this before proceeding.
func TestBuildFinalize_ManifestTaskCountMismatchWithState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Manifest has extra tasks not in state"
	taskOneID := "1.1"
	taskTwoID := "1.2"
	taskThreeID := "1.3"

	// Colony state has 2 tasks
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "standard",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Extra manifest tasks",
				Status: colony.PhaseReady,
				Tasks: []colony.Task{
					{ID: &taskOneID, Goal: "First task", Status: colony.TaskPending},
					{ID: &taskTwoID, Goal: "Second task", Status: colony.TaskPending, DependsOn: []string{taskOneID}},
				},
			}},
		},
	})

	// Generate plan-only manifest
	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnly returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)

	// Tamper: add an extra task to the manifest that does not exist in colony state
	manifest.Tasks = append(manifest.Tasks, codexBuildTaskPlan{
		ID:     taskThreeID,
		Goal:   "Extra task not in colony state",
		Status: colony.TaskPending,
	})

	// Create completion
	dispatchResults := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		dispatchResults = append(dispatchResults, codexExternalBuildWorkerResult{
			Stage:         dispatch.Stage,
			Wave:          dispatch.Wave,
			ExecutionWave: normalizedDispatchWave(dispatch),
			Caste:         dispatch.Caste,
			Name:          dispatch.Name,
			TaskID:        dispatch.TaskID,
			Status:        "completed",
			Summary:       dispatch.Name + " completed",
			FilesModified: []string{"main.go"},
		})
	}

	completion := codexExternalBuildCompletion{
		DispatchManifest: &manifest,
		Dispatches:       dispatchResults,
	}
	completionData, err := json.MarshalIndent(completion, "", "  ")
	if err != nil {
		t.Fatalf("marshal completion: %v", err)
	}
	completionPath := filepath.Join(root, "completion.json")
	if err := os.WriteFile(completionPath, completionData, 0644); err != nil {
		t.Fatalf("write completion: %v", err)
	}

	rootCmd.SetArgs([]string{"build-finalize", "1", "--completion-file", completionPath})
	executeErr := rootCmd.Execute()

	// The manifest now has 3 tasks, colony state has 2. This should be caught.
	// validateBuildManifestTaskSetForPhase should detect the mismatch.
	if executeErr == nil {
		manifestTaskCount := len(manifest.Tasks)
		stateTaskCount := 2 // from colony state
		t.Errorf("BUG CONFIRMED: finalize accepted manifest with %d tasks when colony state has %d tasks (divergence not detected)",
			manifestTaskCount, stateTaskCount)
	} else {
		errOutput := stderr.(*bytes.Buffer).String()
		combinedErr := executeErr.Error() + " " + errOutput
		if !strings.Contains(combinedErr, "task") {
			t.Fatalf("finalize rejected but for wrong reason: %v", executeErr)
		}
		t.Logf("finalize correctly detected task count mismatch: %v", executeErr)
	}
}

// TestContinueAdvance_DivergentTaskCounts demonstrates that the continue
// assessment correctly identifies incomplete tasks. The current gate system
// already blocks advance when tasks are missing/unverified, but the advance
// code at lines 1011-1013 is still defensively dangerous because it
// unconditionally overwrites all task statuses.
//
// CONTEXT: cmd/codex_continue_finalize.go:1011-1013 — advanceExternalContinue
// sets all tasks to TaskCompleted. This is currently guarded by gate checks
// that prevent the advance from being reached when tasks are incomplete, but
// the code itself has no defensive check.
func TestContinueAdvance_DivergentTaskCounts(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Divergent task counts on continue advance"
	now := time.Now().UTC()
	taskOneID := "1.1"
	taskTwoID := "1.2"
	taskThreeID := "1.3"
	nextTaskID := "2.1"

	// Phase 1 has 3 tasks, but only task 1.1 has completed dispatch evidence.
	// Tasks 1.2 and 1.3 are still pending/in_progress in colony state.
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
					Name:   "Divergent task counts",
					Status: colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Completed task", Status: colony.TaskCompleted},
						{ID: &taskTwoID, Goal: "Still pending task", Status: colony.TaskPending},
						{ID: &taskThreeID, Goal: "In progress task", Status: colony.TaskInProgress},
					},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Next work", Status: colony.TaskPending}},
				},
			},
		},
	})

	// Only seed dispatch evidence for task 1.1 — tasks 1.2 and 1.3 have no dispatch.
	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-diverge-1", Task: "Completed task", Status: "completed", TaskID: taskOneID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-diverge-2", Task: "Independent verification", Status: "completed"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Divergent task counts", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})

	// task_evidence is at the top level of the result
	taskEvidenceRaw, ok := result["task_evidence"]
	if !ok {
		t.Fatal("continue result missing task_evidence")
	}
	taskEvidence, ok := taskEvidenceRaw.([]interface{})
	if !ok {
		t.Fatal("task_evidence is not an array")
	}

	// Count how many tasks are NOT verified
	incompleteTasks := 0
	for _, te := range taskEvidence {
		task := te.(map[string]interface{})
		outcome, _ := task["outcome"].(string)
		taskID, _ := task["task_id"].(string)
		if outcome != "verified" && outcome != "manually_reconciled" {
			incompleteTasks++
			t.Logf("task %s assessed as %q (expected incomplete)", taskID, outcome)
		}
	}

	// The assessment SHOULD identify at least tasks 1.2 and 1.3 as incomplete
	if incompleteTasks < 2 {
		t.Errorf("expected at least 2 incomplete tasks in assessment (1.2, 1.3), got %d", incompleteTasks)
	}

	// The current gate system blocks advance when tasks are missing, which is correct.
	// But the advance code at 1011-1013 has no defensive task-status check.
	advanced, _ := result["advanced"].(bool)
	blocked, _ := result["blocked"].(bool)

	if advanced {
		t.Error("BUG: continue advanced phase despite incomplete task assessment")
	}

	if blocked {
		t.Logf("continue correctly blocked advance for %d incomplete tasks", incompleteTasks)
	}

	// Verify the colony state was NOT corrupted
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}

	// Tasks 1.2 and 1.3 should retain their original statuses
	if state.Plan.Phases[0].Tasks[1].Status == colony.TaskCompleted {
		t.Errorf("BUG: task %s was pending but was overwritten to completed during blocked continue", taskTwoID)
	}
	if state.Plan.Phases[0].Tasks[2].Status == colony.TaskCompleted {
		t.Errorf("BUG: task %s was in_progress but was overwritten to completed during blocked continue", taskThreeID)
	}

	// Phase 1 should NOT be completed
	if state.Plan.Phases[0].Status == colony.PhaseCompleted {
		t.Error("BUG: Phase 1 marked completed despite incomplete task records")
	}
}

// TestContinueAdvance_SingleMissingTaskBlocksAdvance verifies that a single
// missing task (no dispatch evidence at all) blocks the continue advance and
// does not corrupt task statuses.
func TestContinueAdvance_SingleMissingTaskBlocksAdvance(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Single missing task blocks advance"
	now := time.Now().UTC()
	taskOneID := "1.1"
	taskTwoID := "1.2"
	nextTaskID := "2.1"

	// Both tasks are completed in colony state
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
					Name:   "Missing task phase",
					Status: colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Has dispatch", Status: colony.TaskCompleted},
						{ID: &taskTwoID, Goal: "No dispatch", Status: colony.TaskCompleted},
					},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Next work", Status: colony.TaskPending}},
				},
			},
		},
	})

	// Only seed dispatch for task 1.1 — task 1.2 has no dispatch at all
	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-missing-1", Task: "Has dispatch", Status: "completed", TaskID: taskOneID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-missing-2", Task: "Independent verification", Status: "completed"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Missing task phase", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	advanced, _ := result["advanced"].(bool)
	blocked, _ := result["blocked"].(bool)

	if advanced {
		t.Error("BUG: continue advanced despite task 1.2 having no dispatch evidence")
	}

	if blocked {
		// Verify the assessment correctly classified task 1.2 as missing
		taskEvidenceRaw, ok := result["task_evidence"]
		if ok {
			tasks := taskEvidenceRaw.([]interface{})
			for _, te := range tasks {
				task := te.(map[string]interface{})
				taskID, _ := task["task_id"].(string)
				if taskID == taskTwoID {
					outcome, _ := task["outcome"].(string)
					if outcome == "missing" {
						t.Logf("task %s correctly assessed as 'missing', advance blocked", taskTwoID)
					} else {
						t.Errorf("task %s assessed as %q but should be 'missing'", taskTwoID, outcome)
					}
				}
			}
		}
	}

	// Verify task statuses were not corrupted
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.Plan.Phases[0].Status == colony.PhaseCompleted {
		t.Error("BUG: Phase 1 marked completed despite task 1.2 having no dispatch evidence")
	}
}

// TestPhaseTasksAllCompleted_Unit is a focused unit test for the
// phaseTasksAllCompleted helper to confirm it works correctly.
func TestPhaseTasksAllCompleted_Unit(t *testing.T) {
	tests := []struct {
		name  string
		tasks []colony.Task
		want  bool
	}{
		{
			name:  "all completed",
			tasks: []colony.Task{{Status: colony.TaskCompleted}, {Status: colony.TaskCompleted}},
			want:  true,
		},
		{
			name:  "one pending",
			tasks: []colony.Task{{Status: colony.TaskCompleted}, {Status: colony.TaskPending}},
			want:  false,
		},
		{
			name:  "one in_progress",
			tasks: []colony.Task{{Status: colony.TaskCompleted}, {Status: colony.TaskInProgress}},
			want:  false,
		},
		{
			name:  "empty tasks",
			tasks: []colony.Task{},
			want:  true,
		},
		{
			name:  "all pending",
			tasks: []colony.Task{{Status: colony.TaskPending}, {Status: colony.TaskPending}},
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			phase := colony.Phase{Tasks: tc.tasks}
			got := phaseTasksAllCompleted(phase)
			if got != tc.want {
				t.Errorf("phaseTasksAllCompleted() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIncompletePhaseTaskSummary_Unit is a focused unit test for the
// incompletePhaseTaskSummary helper.
func TestIncompletePhaseTaskSummary_Unit(t *testing.T) {
	taskOneID := "1.1"
	taskTwoID := "1.2"

	phase := colony.Phase{
		ID:   1,
		Name: "Test phase",
		Tasks: []colony.Task{
			{ID: &taskOneID, Goal: "Completed task", Status: colony.TaskCompleted},
			{ID: &taskTwoID, Goal: "Pending task", Status: colony.TaskPending},
		},
	}

	summary := incompletePhaseTaskSummary(phase)
	if summary == "" || summary == "none" {
		t.Errorf("incompletePhaseTaskSummary() = %q, want to mention pending task %s", summary, taskTwoID)
	}
	if !strings.Contains(summary, taskTwoID) {
		t.Errorf("incompletePhaseTaskSummary() = %q, want to contain task ID %s", summary, taskTwoID)
	}
}
