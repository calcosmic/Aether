package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Ship visual output parity"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"plan"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"📋", "P L A N", "P L A N   D I S P A T C H", "Planning Wave 1 starting", "✓", "aether build 1"} {
		if !strings.Contains(output, want) {
			t.Errorf("plan visual output missing %q\n%s", want, output)
		}
	}
}

func TestBuildVisualOutputShowsSpawnPlan(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Improve command visuals"
	taskOneID := "task-1"
	taskTwoID := "task-2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Visual pass",
					Description: "Bring ceremony to Codex lifecycle output",
					Status:      colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Implement lifecycle renderer", Status: colony.TaskPending},
						{ID: &taskTwoID, Goal: "Document the new output style", Status: colony.TaskPending, DependsOn: []string{taskOneID}},
					},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"🔨", "B U I L D   D I S P A T C H   1", "S P A W N   P L A N", "Builder", "Watcher", "Execution: serial", "single task in this wave", "aether continue", "── Context ──", "── Tasks ──", "── Dispatch ──", "── Verification ──", "── Housekeeping ──", "── Colony Complete ──", "It's safe to clear your context now."} {
		if !strings.Contains(output, want) {
			t.Errorf("build visual output missing %q\n%s", want, output)
		}
	}
}

func TestBuildVisualOutputShowsArtifactContract(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Lock the build packet contract"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Contract mapping",
					Status: colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &taskID, Goal: "Render the build contract", Status: colony.TaskPending},
					},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"A R T I F A C T S",
		".aether/data/build/phase-1/manifest.json",
		".aether/data/last-build-claims.json",
		".aether/data/spawn-tree.txt",
		"── Context ──",
		"── Tasks ──",
		"── Dispatch ──",
		"It's safe to clear your context now.",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("build visual output missing %q\n%s", want, output)
		}
	}
}

func TestRenderSpawnPlanForDispatchesShowsWaveExecutionStrategy(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-11", Task: "Task one", Status: "spawned"},
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-12", Task: "Task two", Status: "spawned"},
		{Stage: "verification", Caste: "watcher", Name: "Keen-13", Task: "Verify it", Status: "spawned"},
	}

	output := renderSpawnPlanForDispatches(dispatches, colony.ModeInRepo)
	for _, want := range []string{
		"Wave 1",
		"Execution: serial",
		"dependency-independent tasks share the main working tree in in-repo mode",
		"set `aether parallel-mode set worktree` for isolated parallel builders",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("spawn plan output missing %q\n%s", want, output)
		}
	}
}

func TestColonizeVisualOutputShowsDispatchPreview(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	defer os.Chdir(oldDir)

	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Test workspace\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}

	goal := "Survey the repo"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"colonize"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colonize returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"🗺️", "C O L O N I Z E   D I S P A T C H", "Survey Wave 1 starting", "Surveyors", "C O L O N I Z E", "aether plan"} {
		if !strings.Contains(output, want) {
			t.Errorf("colonize visual output missing %q\n%s", want, output)
		}
	}
}

func TestColonizeVisualOutputShowsSpawnTreeContract(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	defer os.Chdir(oldDir)

	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	goal := "Surface colonize contracts"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"colonize"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colonize returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, ".aether/data/spawn-tree.txt") {
		t.Errorf("colonize visual output missing spawn tree contract\n%s", output)
	}
}

func TestColonizeVisualOutputShowsDispatchContractDetails(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/colonize-visual-contract-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Render colonize contracts honestly"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"colonize"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colonize returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"Contract",
		"1 wave, parallel read-only worker execution",
		effectiveSurveyorDispatchTimeout(0).String() + " worker max",
		"One surveyor timing out does not reduce sibling surveyor budgets",
		"authenticated platform dispatcher",
		"dispatch_mode, survey_warning, artifact_source",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("colonize visual output missing %q\n%s", want, output)
		}
	}
}

func TestPlanVisualOutputShowsSpawnTreeContract(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Surface planning contracts"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"plan"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, ".aether/data/spawn-tree.txt") {
		t.Errorf("plan visual output missing spawn tree contract\n%s", output)
	}
}

func TestPlanVisualOutputShowsDispatchContractDetails(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/plan-visual-contract-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	goal := "Render plan contracts honestly"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"plan"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"Contract",
		"2 staged workers, scout then route-setter",
		effectivePlanningDispatchTimeout(0).String() + " worker max",
		"route-setter only runs after a completed scout stage",
		"authenticated platform dispatcher",
		"dispatch_mode, planning_warning, artifact_source, plan_source",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("plan visual output missing %q\n%s", want, output)
		}
	}
}

func TestContinueVisualOutputShowsVerificationArtifactsAndSpawnTree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Surface continue contracts"
	now := mustParseRFC3339(t, "2026-04-20T11:00:00Z")
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
					Name:   "Verify contracts",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Verify the build packet", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Keep going", Status: colony.TaskPending}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-41", Task: "Verify the build packet", Status: "spawned", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-42", Task: "Independent verification before advancement", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Verify contracts", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	expectedWatcher := deterministicAntName("watcher", "phase:1:continue:watcher")
	for _, want := range []string{
		"── Verification ──",
		"Phase 1 verified and completed: Verify contracts",
		"Continue Worker Flow",
		expectedWatcher + " [watcher] completed",
		"Watcher " + expectedWatcher + " completed independent verification before advancement",
		"Signal housekeeping [system] completed",
		"── Housekeeping ──",
		"A R T I F A C T S",
		"Workers",
		"Forge-41",
		"Keen-42",
		"Verification passed during continue",
		"── Next Phase ──",
		".aether/data/build/phase-1/verification.json",
		".aether/data/build/phase-1/gates.json",
		".aether/data/build/phase-1/review.json",
		".aether/data/build/phase-1/continue.json",
		".aether/data/spawn-tree.txt",
		"It's safe to clear your context now.",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("continue visual output missing %q\n%s", want, output)
		}
	}
	if strings.Contains(output, "Forge-41 [builder] completed") {
		t.Fatalf("expected continue worker flow to avoid builder closure entries, got:\n%s", output)
	}
}

func TestContinueBlockedVisualOutputShowsWorkerFlow(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Render blocked continue flow honestly"
	now := mustParseRFC3339(t, "2026-04-22T12:36:18Z")
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
					Name:   "Blocked flow",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Stay blocked until watcher approval", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Still pending",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Wait for a clean verification", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Blocked flow", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-141", Task: "Stay blocked until watcher approval", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-build-142", Task: "Build-time verification", Status: "completed"},
	})

	invoker := &continueWatcherTestInvoker{
		watcherStatus:  "blocked",
		watcherSummary: "Continue watcher rejected the phase",
	}
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"C O N T I N U E   B L O C K E D",
		"Continue Worker Flow",
		"Continue watcher rejected the phase",
		"blocked",
		"aether continue --reconcile-task 1.1",
		"A R T I F A C T S",
		".aether/data/build/phase-1/verification.json",
		".aether/data/build/phase-1/gates.json",
		".aether/data/build/phase-1/continue.json",
		".aether/data/spawn-tree.txt",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("blocked continue visual output missing %q\n%s", want, output)
		}
	}
	if strings.Contains(output, "Run `aether continue` to recover the blocked work") {
		t.Fatalf("blocked continue suggested the identical no-op retry:\n%s", output)
	}
	if strings.Contains(output, "Forge-141 [builder] completed") {
		t.Fatalf("expected blocked continue worker flow to avoid builder closure entries, got:\n%s", output)
	}
}

func TestContinueVisualOutputShowsColonyCompleteStageMarker(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Complete the colony honestly"
	now := mustParseRFC3339(t, "2026-04-20T11:15:00Z")
	taskID := "1.1"
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
					Name:   "Finish the final slice",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Finish cleanly", Status: colony.TaskInProgress}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-51", Task: "Finish cleanly", Status: "spawned", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-52", Task: "Independent verification before advancement", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Finish the final slice", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"Phase 1 verified and completed: Finish the final slice",
		"── Colony Complete ──",
		"All planned phases are complete. The colony is ready for Crowned Anthill.",
		"aether seal",
		"It's safe to clear your context now.",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("continue visual output missing %q\n%s", want, output)
		}
	}
}

func TestWatchVisualOutputShowsSnapshotArtifacts(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Watch snapshot contracts"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		Scope:        colony.ScopeMeta,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Execution", Status: colony.PhaseInProgress}},
		},
	})

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "Hammer-9", "Inspect the watch contract", 1); err != nil {
		t.Fatalf("record spawn: %v", err)
	}
	if err := spawnTree.UpdateStatus("Hammer-9", "active", "Running"); err != nil {
		t.Fatalf("mark active: %v", err)
	}

	rootCmd.SetArgs([]string{"watch"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("watch returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"Scope: meta",
		"Active Workers",
		".aether/data/spawn-tree.txt",
		".aether/data/watch-status.txt",
		".aether/data/watch-progress.txt",
		"Run in a TTY for live refresh.",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("watch visual output missing %q\n%s", want, output)
		}
	}
}

func TestWatchLiveRefreshRequiresTTY(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	if shouldUseLiveWatchRefresh(&bytes.Buffer{}, false) {
		t.Fatal("expected non-TTY visual output to stay snapshot-friendly")
	}
	if shouldUseLiveWatchRefresh(&bytes.Buffer{}, true) {
		t.Fatal("expected --once style behavior to disable live refresh")
	}
}

func TestPrintNextUpVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Test next-up visuals"
	name := "test-colony"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		ColonyName:     &name,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Phase 1", Status: colony.PhaseInProgress}},
		},
	})

	rootCmd.SetArgs([]string{"print-next-up"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("print-next-up returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"N E X T   U P", "aether continue"} {
		if !strings.Contains(output, want) {
			t.Errorf("next-up visual output missing %q\n%s", want, output)
		}
	}
}

func TestOutputErrorVisual(t *testing.T) {
	saveGlobals(t)

	var buf bytes.Buffer
	stderr = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	outputError(1, "something failed", nil)

	output := buf.String()
	if strings.Contains(output, `{"ok":false`) {
		t.Fatalf("expected visual error output, got JSON: %s", output)
	}
	for _, want := range []string{"❌", "E R R O R", "something failed"} {
		if !strings.Contains(output, want) {
			t.Errorf("visual error output missing %q\n%s", want, output)
		}
	}
}

func TestShouldRenderVisualOutputTTYOverride(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	if !shouldRenderVisualOutput(&bytes.Buffer{}) {
		t.Fatal("expected visual mode override to force visual output")
	}

	t.Setenv("AETHER_OUTPUT_MODE", "json")
	if shouldRenderVisualOutput(os.Stdout) {
		t.Fatal("expected json mode override to disable visual output")
	}
}

func TestColorizeCasteUsesANSIForVisualOutput(t *testing.T) {
	saveGlobals(t)

	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("NO_COLOR", "")

	got := colorizeCaste("builder", "builder")
	if !strings.Contains(got, "\x1b[33m") {
		t.Fatalf("expected ANSI-highlighted builder caste text, got %q", got)
	}
	if !strings.Contains(got, "builder") || !strings.Contains(got, "\x1b[0m") {
		t.Fatalf("expected ANSI-highlighted builder label, got %q", got)
	}
}

func TestShouldUseANSIColorsUsesVisualModeOrForce(t *testing.T) {
	saveGlobals(t)

	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("NO_COLOR", "")

	if !shouldUseANSIColors() {
		t.Fatal("expected visual mode to allow ANSI colors for caste highlighting")
	}

	t.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Setenv("AETHER_FORCE_COLOR", "1")
	if !shouldUseANSIColors() {
		t.Fatal("expected AETHER_FORCE_COLOR to override ANSI detection")
	}
}

func TestInstallVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	workDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--home-dir", homeDir, "--skip-build-binary"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"📦", "I N S T A L L", "█████╗", "aether lay-eggs", "shared `aether` binary", "Published release repos:", "aether update --force --download-binary", "aether update --force"} {
		if !strings.Contains(output, want) {
			t.Errorf("install visual output missing %q\n%s", want, output)
		}
	}
}

func TestRenderUpdateVisualNoChangesSaysNoFollowUpRequired(t *testing.T) {
	output := renderUpdateVisual(
		"/tmp/example",
		"1.0.7",
		"1.0.7",
		false,
		false,
		[]map[string]interface{}{
			{"label": "System files", "copied": 0, "skipped": 10},
			{"label": "AGENTS.md", "skipped": 1, "reason": "unchanged"},
			{"label": ".codex/CODEX.md", "skipped": 1, "reason": "unchanged"},
		},
		0,
		12,
		nil,
		"unchanged",
		true,
	)

	if !strings.Contains(output, "No follow-up is required.") {
		t.Fatalf("expected no-follow-up guidance, got:\n%s", output)
	}
	if !strings.Contains(output, "Binary: already at the current hub version") {
		t.Fatalf("expected binary-already-current message in update visual, got:\n%s", output)
	}
	if strings.Contains(output, "Run `aether status` to inspect the colony after the refresh.") {
		t.Fatalf("expected generic next-step guidance to be suppressed, got:\n%s", output)
	}
}

func TestWorkflowSuggestionsForPausedCurrentPhaseUsesCurrentPhaseNumber(t *testing.T) {
	goal := "Keep the phase suggestion sane"
	primary, _ := workflowSuggestionsForState(colony.ColonyState{
		Goal:         &goal,
		State:        colony.State("PAUSED"),
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Reproduce and isolate", Status: colony.PhaseInProgress},
				{ID: 2, Name: "Targeted fix", Status: colony.PhasePending},
			},
		},
	})

	if !strings.Contains(primary, "aether resume") {
		t.Fatalf("expected paused colony to suggest resume, got: %s", primary)
	}
	if strings.Contains(primary, "aether build 2") {
		t.Fatalf("expected paused colony not to skip ahead, got: %s", primary)
	}
}

func TestWorkflowSuggestionsForPausedFlagSuggestsResume(t *testing.T) {
	goal := "Pause should route to resume"
	primary, _ := workflowSuggestionsForState(colony.ColonyState{
		Goal:   &goal,
		State:  colony.StateREADY,
		Paused: true,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Reproduce and isolate", Status: colony.PhaseInProgress},
			},
		},
	})

	if !strings.Contains(primary, "aether resume") {
		t.Fatalf("expected paused colony to suggest resume, got: %s", primary)
	}
}

func TestWorkflowSuggestionsForInterruptedExecutingSuggestsRestartBuild(t *testing.T) {
	goal := "Interrupted build should restart clearly"
	primary, _ := workflowSuggestionsForState(colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Done", Status: colony.PhaseCompleted},
				{ID: 2, Name: "Interrupted", Status: colony.PhaseInProgress},
			},
		},
	})

	if !strings.Contains(primary, "aether build 2") {
		t.Fatalf("expected interrupted executing colony to suggest restarting build 2, got: %s", primary)
	}
}

func TestSetupVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()
	hubSystem := filepath.Join(homeDir, ".aether", "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".aether", "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}

	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup returned error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"🥚", "L A Y   E G G S", "aether init"} {
		if !strings.Contains(output, want) {
			t.Errorf("setup visual output missing %q\n%s", want, output)
		}
	}
}

func TestUpdateDryRunVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()
	hubDir := filepath.Join(homeDir, ".aether")
	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer os.Chdir(oldDir)
	t.Setenv("HOME", homeDir)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"update", "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update --dry-run returned error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, `{"ok":true`) {
		t.Fatalf("expected visual output, got JSON: %s", output)
	}
	for _, want := range []string{"🔄", "U P D A T E", "█████╗", "Dry run complete", "aether update", "Published release:", "Local source checkout:"} {
		if !strings.Contains(output, want) {
			t.Errorf("update visual output missing %q\n%s", want, output)
		}
	}
}

func TestPauseResumePatrolPhaseAndHistoryVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Close the remaining Codex UX gaps"
	taskOneID := "task-1"
	taskTwoID := "task-2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Session UX",
					Description: "Add missing resume and handoff flow",
					Status:      colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Write pause-colony", Status: colony.TaskCompleted},
						{ID: &taskTwoID, Goal: "Write resume-colony", Status: colony.TaskInProgress},
					},
				},
			},
		},
		Events: []string{
			"2026-04-15T10:00:00Z|init|queen|Colony initialized",
			"2026-04-15T10:15:00Z|build|builder|Worker wave launched",
		},
	})

	if err := store.SaveJSON("pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{Type: "FOCUS", Content: []byte(`{"text":"keep the Codex UX coherent"}`), Active: true},
		},
	}); err != nil {
		t.Fatalf("failed to seed pheromones: %v", err)
	}

	if err := store.SaveJSON("flags.json", colony.FlagsFile{
		Version: "1.0",
		Decisions: []colony.FlagEntry{
			{ID: "flag-1", Type: "blocker", Description: "resume parity still missing survey context", CreatedAt: "2026-04-15T10:20:00Z"},
		},
	}); err != nil {
		t.Fatalf("failed to seed flags: %v", err)
	}

	surveyedAt := "2026-04-15T10:05:00Z"
	var seededState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &seededState); err != nil {
		t.Fatalf("failed to load seeded state: %v", err)
	}
	seededState.TerritorySurveyed = &surveyedAt
	if err := store.SaveJSON("COLONY_STATE.json", seededState); err != nil {
		t.Fatalf("failed to resave seeded state: %v", err)
	}
	surveyDir := filepath.Join(dataDir, "survey")
	if err := os.MkdirAll(surveyDir, 0755); err != nil {
		t.Fatalf("failed to create survey dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(surveyDir, "blueprint.json"), []byte(`{"name":"ux"}`), 0644); err != nil {
		t.Fatalf("failed to seed survey file: %v", err)
	}

	checkVisual := func(args []string, wants ...string) {
		stdout = &bytes.Buffer{}
		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("%s returned error: %v", strings.Join(args, " "), err)
		}
		output := stdout.(*bytes.Buffer).String()
		if strings.Contains(output, `{"ok":true`) {
			t.Fatalf("%s unexpectedly returned JSON: %s", strings.Join(args, " "), output)
		}
		for _, want := range wants {
			if !strings.Contains(output, want) {
				t.Errorf("%s visual output missing %q\n%s", strings.Join(args, " "), want, output)
			}
		}
	}

	checkVisual([]string{"pause-colony"}, "💾", "P A U S E   C O L O N Y", "HANDOFF.md", "aether resume")
	checkVisual([]string{"resume-colony"}, "💾", "R E S U M E   C O L O N Y", "Session UX", "Active Signals", "Blockers", "Survey Context", "Source:", "aether build 1")
	checkVisual([]string{"patrol"}, "📊", "P A T R O L", "Signals: 1 active")
	checkVisual([]string{"phase"}, "🧱", "Session UX", "Write resume-colony")
	checkVisual([]string{"history"}, "📜", "Colony initialized", "Worker wave launched")
}

func TestRenderColonizeVisual_FakeDispatch(t *testing.T) {
	// When all surveyors have "spawned" status, the legacy display should be used.
	result := map[string]interface{}{
		"root":          "/tmp/test",
		"detected_type": "go",
		"languages":     []interface{}{"go"},
		"frameworks":    []interface{}{"cobra"},
		"domains":       []interface{}{"cli"},
		"stats": map[string]interface{}{
			"files":       42,
			"directories": 7,
		},
		"survey_dir": "/tmp/test/.aether/data/survey",
		"surveyors": []interface{}{
			map[string]interface{}{"name": "Nest-42", "caste": "surveyor-nest", "task": "Map architecture", "status": "spawned"},
			map[string]interface{}{"name": "Disc-7", "caste": "surveyor-disciplines", "task": "Map disciplines", "status": "spawned"},
			map[string]interface{}{"name": "Path-3", "caste": "surveyor-pathogens", "task": "Identify pathogens", "status": "spawned"},
			map[string]interface{}{"name": "Prov-1", "caste": "surveyor-provisions", "task": "Map provisions", "status": "spawned"},
		},
		"survey_files": []interface{}{"BLUEPRINT.md", "CHAMBERS.md"},
	}

	output := renderColonizeVisual(result)

	// Legacy display should show surveyor names and tasks (no status icons or durations)
	if !strings.Contains(output, "Nest-42") {
		t.Errorf("legacy output missing surveyor name %q\n%s", "Nest-42", output)
	}
	if !strings.Contains(output, "Map architecture") {
		t.Errorf("legacy output missing surveyor task %q\n%s", "Map architecture", output)
	}
	// Should show a non-real dispatch indicator
	if !strings.Contains(output, "Dispatch: Simulated") && !strings.Contains(output, "Dispatch: Synthetic") {
		t.Errorf("legacy output should contain a non-real dispatch indicator\n%s", output)
	}
	// Should NOT show real dispatch indicator
	if strings.Contains(output, "Dispatch: Real") {
		t.Errorf("legacy output should not contain 'Dispatch: Real'\n%s", output)
	}
	// Should still show standard sections
	if !strings.Contains(output, "C O L O N I Z E") {
		t.Errorf("missing banner in output\n%s", output)
	}
}

func TestRenderColonizeVisual_RealDispatch(t *testing.T) {
	// When surveyors have "completed" or "failed" status, show execution data.
	result := map[string]interface{}{
		"root":          "/tmp/test",
		"detected_type": "go",
		"languages":     []interface{}{"go"},
		"frameworks":    []interface{}{"cobra"},
		"domains":       []interface{}{"cli"},
		"stats": map[string]interface{}{
			"files":       42,
			"directories": 7,
		},
		"survey_dir": "/tmp/test/.aether/data/survey",
		"surveyors": []interface{}{
			map[string]interface{}{"name": "Nest-42", "caste": "surveyor-nest", "task": "Map architecture", "status": "completed", "summary": "Mapped the chamber layout", "duration": 12.3},
			map[string]interface{}{"name": "Disc-7", "caste": "surveyor-disciplines", "task": "Map disciplines", "status": "completed", "duration": 8.1},
			map[string]interface{}{"name": "Path-3", "caste": "surveyor-pathogens", "task": "Identify pathogens", "status": "failed", "summary": "Blocked on missing evidence", "duration": 5.2},
			map[string]interface{}{"name": "Prov-1", "caste": "surveyor-provisions", "task": "Map provisions", "status": "completed", "duration": 15.7},
		},
		"survey_files": []interface{}{"BLUEPRINT.md", "CHAMBERS.md"},
	}

	output := renderColonizeVisual(result)

	// Real dispatch should show the "Dispatch: Real" indicator
	if !strings.Contains(output, "Dispatch: Real") {
		t.Errorf("real dispatch output missing 'Dispatch: Real'\n%s", output)
	}
	// Should show status icons
	if !strings.Contains(output, "\u2713") {
		t.Errorf("real dispatch output missing checkmark for completed surveyors\n%s", output)
	}
	if !strings.Contains(output, "\u2717") {
		t.Errorf("real dispatch output missing X for failed surveyors\n%s", output)
	}
	// Should show durations
	if !strings.Contains(output, "12.3s") {
		t.Errorf("real dispatch output missing duration 12.3s\n%s", output)
	}
	if !strings.Contains(output, "Mapped the chamber layout") {
		t.Errorf("real dispatch output missing worker summary\n%s", output)
	}
	if !strings.Contains(output, "8.1s") {
		t.Errorf("real dispatch output missing duration 8.1s\n%s", output)
	}
	// Should show summary line
	if !strings.Contains(output, "3/4 surveyors completed") {
		t.Errorf("real dispatch output missing summary '3/4 surveyors completed'\n%s", output)
	}
	// Should contain the "Surveyors" section header with real results
	if !strings.Contains(output, "\nSurveyors\n") {
		t.Errorf("real dispatch output should have 'Surveyors' header\n%s", output)
	}
}

func TestRenderColonizeVisual_MixedResults(t *testing.T) {
	// Mix of "spawned" and real statuses: spawned is treated as fake.
	result := map[string]interface{}{
		"root":          "/tmp/test",
		"detected_type": "go",
		"languages":     []interface{}{"go"},
		"frameworks":    []interface{}{},
		"domains":       []interface{}{},
		"stats": map[string]interface{}{
			"files":       10,
			"directories": 2,
		},
		"survey_dir": "/tmp/test/.aether/data/survey",
		"surveyors": []interface{}{
			map[string]interface{}{"name": "Nest-42", "caste": "surveyor-nest", "task": "Map architecture", "status": "completed", "duration": 5.0},
			map[string]interface{}{"name": "Disc-7", "caste": "surveyor-disciplines", "task": "Map disciplines", "status": "spawned"},
			map[string]interface{}{"name": "Path-3", "caste": "surveyor-pathogens", "task": "Identify pathogens", "status": "failed", "duration": 3.0},
		},
		"survey_files": []interface{}{"BLUEPRINT.md"},
	}

	output := renderColonizeVisual(result)

	// Should show "Dispatch: Real" because at least one has real execution data
	if !strings.Contains(output, "Dispatch: Real") {
		t.Errorf("mixed output should show 'Dispatch: Real'\n%s", output)
	}
	// Should show 1/3 completed (only Nest-42 completed; Disc-7 spawned; Path-3 failed)
	if !strings.Contains(output, "1/3 surveyors completed") {
		t.Errorf("mixed output missing correct summary '1/3 surveyors completed'\n%s", output)
	}
}

func TestRenderColonizeVisual_ShowsSurveyFallbackWarning(t *testing.T) {
	result := map[string]interface{}{
		"root":           "/tmp/test",
		"detected_type":  "go",
		"languages":      []interface{}{"go"},
		"frameworks":     []interface{}{"cobra"},
		"domains":        []interface{}{"cli"},
		"dispatch_mode":  "fallback",
		"survey_warning": "Real surveyors did not finish cleanly, so Aether fell back to local survey synthesis.",
		"stats": map[string]interface{}{
			"files":       42,
			"directories": 7,
		},
		"survey_dir": "/tmp/test/.aether/data/survey",
		"surveyors": []interface{}{
			map[string]interface{}{"name": "Nest-42", "caste": "surveyor-nest", "task": "Map architecture", "status": "timeout", "summary": "timed out", "duration": 60.0},
			map[string]interface{}{"name": "Disc-7", "caste": "surveyor-disciplines", "task": "Map disciplines", "status": "timeout", "summary": "timed out", "duration": 0.0},
		},
		"survey_files": []interface{}{"BLUEPRINT.md", "CHAMBERS.md"},
	}

	output := renderColonizeVisual(result)

	if !strings.Contains(output, "Survey Warning") {
		t.Errorf("output missing survey warning header\n%s", output)
	}
	if !strings.Contains(output, "fell back to local survey synthesis") {
		t.Errorf("output missing survey fallback explanation\n%s", output)
	}
	if !strings.Contains(output, "Dispatch: Fallback") {
		t.Errorf("output missing fallback dispatch label\n%s", output)
	}
}

func TestRenderSurveyorResults_Formatting(t *testing.T) {
	surveyors := []codexSurveyorDispatch{
		{Caste: "surveyor-nest", Name: "Nest-42", Task: "Map architecture", Status: "completed", Summary: "Mapped chamber layout", Duration: 12.3},
		{Caste: "surveyor-disciplines", Name: "Disc-7", Task: "Map disciplines", Status: "completed", Duration: 8.1},
		{Caste: "surveyor-pathogens", Name: "Path-3", Task: "Identify pathogens", Status: "failed", Summary: "Blocked by missing config", Duration: 5.2},
		{Caste: "surveyor-provisions", Name: "Prov-1", Task: "Map provisions", Status: "completed", Duration: 15.7},
	}

	output := renderSurveyorResults(surveyors)

	// Each surveyor should be on its own line with emoji, name, caste, status, and duration
	if !strings.Contains(output, "Nest-42") {
		t.Errorf("missing Nest-42\n%s", output)
	}
	if !strings.Contains(output, "Disc-7") {
		t.Errorf("missing Disc-7\n%s", output)
	}
	if !strings.Contains(output, "Path-3") {
		t.Errorf("missing Path-3\n%s", output)
	}
	if !strings.Contains(output, "Prov-1") {
		t.Errorf("missing Prov-1\n%s", output)
	}

	// Should use caste label "Surveyor" from casteLabelMap
	if !strings.Contains(output, "Surveyor") {
		t.Errorf("missing Surveyor caste label\n%s", output)
	}
	// Should use single emoji from casteEmojiMap (surveyor = bar chart)
	if !strings.Contains(output, "\U0001f4ca") {
		t.Errorf("missing surveyor emoji\n%s", output)
	}

	// Completed should have checkmark, failed should have X
	if !strings.Contains(output, "\u2713") || !strings.Contains(output, "completed") {
		t.Errorf("missing completed worker status details\n%s", output)
	}
	if !strings.Contains(output, "\u2717") || !strings.Contains(output, "failed") {
		t.Errorf("missing failed worker status details\n%s", output)
	}

	// Should have durations
	if !strings.Contains(output, "12.3s") {
		t.Errorf("missing duration 12.3s\n%s", output)
	}
	if !strings.Contains(output, "Task: Map architecture") {
		t.Errorf("missing task detail line\n%s", output)
	}
	if !strings.Contains(output, "Mapped chamber layout") {
		t.Errorf("missing worker summary line\n%s", output)
	}

	// Summary line at the end
	if !strings.Contains(output, "3/4 surveyors completed") {
		t.Errorf("missing summary line\n%s", output)
	}
}

func TestRenderSurveyorResults_Empty(t *testing.T) {
	output := renderSurveyorResults(nil)
	if output != "" {
		t.Errorf("expected empty output for nil surveyors, got %q", output)
	}
	output = renderSurveyorResults([]codexSurveyorDispatch{})
	if output != "" {
		t.Errorf("expected empty output for empty surveyors, got %q", output)
	}
}

func TestCasteIdentitySurveyorSubtypes(t *testing.T) {
	// Surveyor subtypes should resolve to the Surveyor emoji/label/color, not generic Ant.
	os.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	subtypes := []string{"surveyor-nest", "surveyor-pathogens", "surveyor-provisions", "surveyor-disciplines"}
	for _, subtype := range subtypes {
		identity := casteIdentity(subtype)
		if !strings.Contains(identity, "📊") {
			t.Errorf("casteIdentity(%q): expected 📊 emoji, got %q", subtype, identity)
		}
		if !strings.Contains(identity, "Surveyor") {
			t.Errorf("casteIdentity(%q): expected 'Surveyor' label, got %q", subtype, identity)
		}
		emoji := casteEmoji(subtype)
		if emoji != "📊" {
			t.Errorf("casteEmoji(%q): expected 📊, got %q", subtype, emoji)
		}
		label := casteLabel(subtype)
		if !strings.Contains(label, "Surveyor") {
			t.Errorf("casteLabel(%q): expected 'Surveyor', got %q", subtype, label)
		}
		color := casteANSIColor(subtype)
		if color == "" {
			t.Errorf("casteANSIColor(%q): expected a color code, got empty", subtype)
		}
	}
}

func TestCasteIdentityUnknownCaste(t *testing.T) {
	os.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	// Unknown castes should fall back to generic Ant.
	identity := casteIdentity("unknown-worker")
	if !strings.Contains(identity, "🐜") {
		t.Errorf("expected 🐜 fallback for unknown caste, got %q", identity)
	}
	if !strings.Contains(identity, "Ant") {
		t.Errorf("expected 'Ant' fallback for unknown caste, got %q", identity)
	}
}

func TestCasteIdentityExactMatchPreferred(t *testing.T) {
	os.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	// Exact matches should be used, not prefix fallback.
	identity := casteIdentity("builder")
	if !strings.Contains(identity, "🔨") {
		t.Errorf("expected 🔨 for builder, got %q", identity)
	}
	if !strings.Contains(identity, "Builder") {
		t.Errorf("expected 'Builder' label, got %q", identity)
	}
}

func TestDispatchStatusIcon_DistinguishesRunningAndTimeout(t *testing.T) {
	if got := dispatchStatusIcon("running"); got != "…" {
		t.Fatalf("dispatchStatusIcon(running) = %q, want ellipsis", got)
	}
	if got := dispatchStatusIcon("timeout"); got != "✗" {
		t.Fatalf("dispatchStatusIcon(timeout) = %q, want failure mark", got)
	}
}

func TestRenderPlanVisual_SimulatedDispatch(t *testing.T) {
	// When all planning workers have "spawned" status, the legacy display should be used.
	phases := []colony.Phase{
		{ID: 1, Name: "Discovery", Description: "Map the codebase", Status: colony.PhaseReady,
			Tasks: []colony.Task{{Goal: "Read code paths"}}},
	}
	phaseMaps := make([]interface{}, len(phases))
	for i, p := range phases {
		phaseMaps[i] = phaseToMap(p)
	}

	result := map[string]interface{}{
		"existing_plan": false,
		"goal":          "Build feature X",
		"granularity":   "milestone",
		"confidence":    map[string]interface{}{"overall": 72},
		"phases":        phaseMaps,
		"dispatches": []interface{}{
			map[string]interface{}{"name": "Scout-7", "caste": "scout", "task": "Survey the repo", "status": "spawned"},
			map[string]interface{}{"name": "Route-12", "caste": "route_setter", "task": "Convert findings into phases", "status": "spawned"},
		},
	}

	output := renderPlanVisual(result)

	// Should show worker names and tasks (legacy style, no status icons)
	if !strings.Contains(output, "Scout-7") {
		t.Errorf("simulated output missing scout name\n%s", output)
	}
	if !strings.Contains(output, "Route-12") {
		t.Errorf("simulated output missing route-setter name\n%s", output)
	}
	if !strings.Contains(output, "Survey the repo") {
		t.Errorf("simulated output missing scout task\n%s", output)
	}
	// Should NOT show dispatch mode indicator or status icons
	if strings.Contains(output, "Dispatch: Real") {
		t.Errorf("simulated output should not contain 'Dispatch: Real'\n%s", output)
	}
	if strings.Contains(output, "Dispatch: Simulated") {
		t.Errorf("simulated output should not contain 'Dispatch: Simulated'\n%s", output)
	}
	// Should still show standard sections
	if !strings.Contains(output, "P L A N") {
		t.Errorf("missing banner in output\n%s", output)
	}
}

func TestRenderPlanVisual_RealDispatch(t *testing.T) {
	// When planning workers have "completed" or "failed" status, show execution data.
	phases := []colony.Phase{
		{ID: 1, Name: "Discovery", Description: "Map the codebase", Status: colony.PhaseReady,
			Tasks: []colony.Task{{Goal: "Read code paths"}}},
	}
	phaseMaps := make([]interface{}, len(phases))
	for i, p := range phases {
		phaseMaps[i] = phaseToMap(p)
	}

	result := map[string]interface{}{
		"existing_plan": false,
		"goal":          "Build feature X",
		"granularity":   "milestone",
		"confidence":    map[string]interface{}{"overall": 72},
		"phases":        phaseMaps,
		"dispatches": []interface{}{
			map[string]interface{}{"name": "Scout-7", "caste": "scout", "task": "Survey the repo", "status": "completed", "summary": "Mapped the runtime terrain", "duration": 3.5},
			map[string]interface{}{"name": "Route-12", "caste": "route_setter", "task": "Convert findings into phases", "status": "completed", "summary": "Shaped the next phases", "duration": 2.1},
		},
	}

	output := renderPlanVisual(result)

	// Should show "Dispatch: Real" indicator
	if !strings.Contains(output, "Dispatch: Real") {
		t.Errorf("real dispatch output missing 'Dispatch: Real'\n%s", output)
	}
	// Should show status icons (checkmark for completed)
	if !strings.Contains(output, "\u2713") {
		t.Errorf("real dispatch output missing checkmark for completed workers\n%s", output)
	}
	// Should show durations
	if !strings.Contains(output, "3.5s") {
		t.Errorf("real dispatch output missing duration 3.5s\n%s", output)
	}
	if !strings.Contains(output, "2.1s") {
		t.Errorf("real dispatch output missing duration 2.1s\n%s", output)
	}
	if !strings.Contains(output, "Mapped the runtime terrain") {
		t.Errorf("real dispatch output missing scout summary\n%s", output)
	}
	// Should show summary line
	if !strings.Contains(output, "2/2 workers completed") {
		t.Errorf("real dispatch output missing summary '2/2 workers completed'\n%s", output)
	}
	// Should show "Workers" section header
	if !strings.Contains(output, "\nWorkers\n") {
		t.Errorf("real dispatch output should have 'Workers' header\n%s", output)
	}
}

func TestRenderPlanVisual_RealDispatchWithFailure(t *testing.T) {
	// Scout completed but route-setter failed.
	phases := []colony.Phase{
		{ID: 1, Name: "Discovery", Description: "Map the codebase", Status: colony.PhaseReady,
			Tasks: []colony.Task{{Goal: "Read code paths"}}},
	}
	phaseMaps := make([]interface{}, len(phases))
	for i, p := range phases {
		phaseMaps[i] = phaseToMap(p)
	}

	result := map[string]interface{}{
		"existing_plan": false,
		"goal":          "Build feature X",
		"granularity":   "milestone",
		"confidence":    map[string]interface{}{"overall": 72},
		"phases":        phaseMaps,
		"dispatches": []interface{}{
			map[string]interface{}{"name": "Scout-7", "caste": "scout", "task": "Survey the repo", "status": "completed", "summary": "Captured the repo shape", "duration": 4.0},
			map[string]interface{}{"name": "Route-12", "caste": "route_setter", "task": "Convert findings into phases", "status": "failed", "summary": "Blocked by missing planning file", "duration": 1.2},
		},
	}

	output := renderPlanVisual(result)

	// Should show "Dispatch: Real"
	if !strings.Contains(output, "Dispatch: Real") {
		t.Errorf("output missing 'Dispatch: Real'\n%s", output)
	}
	// Should show checkmark for scout and X for route-setter
	if !strings.Contains(output, "\u2713") {
		t.Errorf("output missing checkmark\n%s", output)
	}
	if !strings.Contains(output, "\u2717") {
		t.Errorf("output missing X for failed worker\n%s", output)
	}
	// Should show 1/2 completed
	if !strings.Contains(output, "1/2 workers completed") {
		t.Errorf("output missing summary '1/2 workers completed'\n%s", output)
	}
	// Should show total duration
	if !strings.Contains(output, "5.2s") {
		t.Errorf("output missing total duration 5.2s\n%s", output)
	}
}

func TestRenderPlanVisual_ShowsPlanningFallbackWarning(t *testing.T) {
	phases := []colony.Phase{
		{ID: 1, Name: "Discovery", Description: "Map the codebase", Status: colony.PhaseReady,
			Tasks: []colony.Task{{Goal: "Read code paths"}}},
	}
	phaseMaps := make([]interface{}, len(phases))
	for i, p := range phases {
		phaseMaps[i] = phaseToMap(p)
	}

	result := map[string]interface{}{
		"existing_plan":    false,
		"goal":             "Build feature X",
		"granularity":      "milestone",
		"confidence":       map[string]interface{}{"overall": 72},
		"phases":           phaseMaps,
		"dispatch_mode":    "fallback",
		"planning_warning": "Real planning workers did not finish cleanly, so Aether fell back to local synthesis.",
		"dispatches": []interface{}{
			map[string]interface{}{"name": "Scout-7", "caste": "scout", "task": "Survey the repo", "status": "timeout", "summary": "timed out", "duration": 60.0},
			map[string]interface{}{"name": "Route-12", "caste": "route_setter", "task": "Convert findings into phases", "status": "timeout", "summary": "timed out", "duration": 0.0},
		},
	}

	output := renderPlanVisual(result)

	if !strings.Contains(output, "Planning Warning") {
		t.Errorf("output missing planning warning header\n%s", output)
	}
	if !strings.Contains(output, "fell back to local synthesis") {
		t.Errorf("output missing fallback explanation\n%s", output)
	}
	if !strings.Contains(output, "Dispatch: Fallback") {
		t.Errorf("output missing fallback dispatch label\n%s", output)
	}
}

func TestRenderPlanVisual_NoDispatches(t *testing.T) {
	// No dispatches at all -- should not crash, should show legacy output.
	phases := []colony.Phase{
		{ID: 1, Name: "Discovery", Description: "Map the codebase", Status: colony.PhaseReady,
			Tasks: []colony.Task{{Goal: "Read code paths"}}},
	}
	phaseMaps := make([]interface{}, len(phases))
	for i, p := range phases {
		phaseMaps[i] = phaseToMap(p)
	}

	result := map[string]interface{}{
		"existing_plan": false,
		"goal":          "Build feature X",
		"granularity":   "milestone",
		"confidence":    map[string]interface{}{"overall": 72},
		"phases":        phaseMaps,
	}

	output := renderPlanVisual(result)

	// Should not crash, should still show phases
	if !strings.Contains(output, "Discovery") {
		t.Errorf("output missing phase name\n%s", output)
	}
	if !strings.Contains(output, "P L A N") {
		t.Errorf("missing banner\n%s", output)
	}
}

func TestRenderPlanVisual_ExistingPlan(t *testing.T) {
	// Existing plan with no dispatches -- should show the existing plan message.
	phases := []colony.Phase{
		{ID: 1, Name: "Phase 1", Status: colony.PhaseReady,
			Tasks: []colony.Task{{Goal: "Task A"}}},
	}
	phaseMaps := make([]interface{}, len(phases))
	for i, p := range phases {
		phaseMaps[i] = phaseToMap(p)
	}

	result := map[string]interface{}{
		"existing_plan": true,
		"goal":          "Previous goal",
		"phases":        phaseMaps,
	}

	output := renderPlanVisual(result)

	if !strings.Contains(output, "Existing colony plan loaded") {
		t.Errorf("existing plan output missing 'Existing colony plan loaded'\n%s", output)
	}
}

func TestRenderPlanVisual_ShowsClarificationWarning(t *testing.T) {
	phaseMaps := []interface{}{
		map[string]interface{}{
			"id":     1,
			"name":   "Phase 1",
			"status": colony.PhaseReady,
			"tasks": []interface{}{
				map[string]interface{}{"goal": "Task A", "status": colony.TaskPending},
			},
		},
	}

	result := map[string]interface{}{
		"existing_plan":             true,
		"goal":                      "Previous goal",
		"phases":                    phaseMaps,
		"unresolved_clarifications": 2,
		"clarification_warning":     "Unresolved clarifications exist. Run `aether discuss` to resolve them before planning, or proceed with implicit assumptions.",
	}

	output := renderPlanVisual(result)
	if !strings.Contains(output, "Clarifications") {
		t.Fatalf("expected clarifications section in plan output\n%s", output)
	}
	if !strings.Contains(output, "2 unresolved clarification(s)") {
		t.Fatalf("expected unresolved clarification count in plan output\n%s", output)
	}
	if !strings.Contains(output, "Run `aether discuss`") {
		t.Fatalf("expected discuss guidance in plan output\n%s", output)
	}
}

func TestRenderPlanVisual_IncludesTaskMetadata(t *testing.T) {
	phaseMaps := []interface{}{
		map[string]interface{}{
			"id":               1,
			"name":             "Planning orchestration",
			"description":      "Add a scout plus route-setter planning pass.",
			"status":           colony.PhaseReady,
			"success_criteria": []interface{}{"Plan generation is ant-driven", "The colony has a grounded next phase"},
			"tasks": []interface{}{
				map[string]interface{}{
					"id":               "1.1",
					"goal":             "Generate a route-setter plan with task constraints, hints, and success criteria",
					"status":           colony.TaskPending,
					"constraints":      []interface{}{"The first phase must become ready", "The saved plan must match the displayed plan"},
					"hints":            []interface{}{"COLONY_STATE.json", "renderPlanVisual"},
					"success_criteria": []interface{}{"Plan generation is grounded in repo context", "Spawn records show scout and route-setter activity"},
					"depends_on":       []interface{}{"0.2"},
				},
			},
		},
	}

	result := map[string]interface{}{
		"existing_plan": false,
		"goal":          "Ground the plan in worker artifacts",
		"phases":        phaseMaps,
	}

	output := renderPlanVisual(result)

	for _, want := range []string{
		"Task 1.1",
		"Constraints:",
		"The first phase must become ready",
		"Hints:",
		"COLONY_STATE.json",
		"Success Criteria:",
		"Plan generation is grounded in repo context",
		"Depends on: 0.2",
		"Phase Success Criteria:",
		"The colony has a grounded next phase",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected plan visual to contain %q\n%s", want, output)
		}
	}
}

// phaseToMap converts a colony.Phase to a map[string]interface{} for test construction.
func phaseToMap(p colony.Phase) map[string]interface{} {
	tasks := make([]interface{}, len(p.Tasks))
	for i, task := range p.Tasks {
		m := map[string]interface{}{
			"goal":   task.Goal,
			"status": task.Status,
		}
		if task.ID != nil {
			m["id"] = *task.ID
		}
		tasks[i] = m
	}
	return map[string]interface{}{
		"id":          p.ID,
		"name":        p.Name,
		"description": p.Description,
		"status":      p.Status,
		"tasks":       tasks,
	}
}

func TestCasteIdentityAllCastes(t *testing.T) {
	t.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	for caste := range casteEmojiMap {
		identity := casteIdentity(caste)
		emoji := casteEmoji(caste)
		label := casteLabel(caste)

		if !strings.Contains(identity, emoji) {
			t.Errorf("casteIdentity(%q): expected emoji %q, got %q", caste, emoji, identity)
		}
		if !strings.Contains(identity, label) {
			t.Errorf("casteIdentity(%q): expected label %q, got %q", caste, label, identity)
		}
	}
}

func TestBuildWaveProgressShowsCasteIdentity(t *testing.T) {
	t.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	dispatches := []codex.WorkerDispatch{
		{WorkerName: "Forge-41", Caste: "builder", TaskBrief: "Implement feature X"},
		{WorkerName: "Keen-42", Caste: "watcher", TaskBrief: "Verify tests pass"},
	}

	emitCodexBuildWaveProgress(colony.Phase{ID: 1, Name: "Test Phase"}, 1, dispatches, colony.ModeInRepo)

	output := buf.String()
	for _, want := range []string{"🔨", "Builder", "Forge-41", "👁️", "Watcher", "Keen-42"} {
		if !strings.Contains(output, want) {
			t.Errorf("wave progress missing %q\n%s", want, output)
		}
	}
}

func TestBuildWorkerStartedShowsCasteIdentity(t *testing.T) {
	t.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	dispatch := codex.WorkerDispatch{WorkerName: "Forge-41", Caste: "builder", TaskBrief: "Implement feature X"}
	emitCodexBuildWorkerStarted(dispatch, 1)

	output := buf.String()
	for _, want := range []string{"🔨", "Builder", "Forge-41"} {
		if !strings.Contains(output, want) {
			t.Errorf("worker started missing %q\n%s", want, output)
		}
	}
}

func TestBuildWorkerFinishedShowsCasteIdentity(t *testing.T) {
	t.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	dispatch := codex.WorkerDispatch{WorkerName: "Forge-41", Caste: "builder", TaskBrief: "Implement feature X"}
	result := codex.DispatchResult{WorkerName: "Forge-41", Status: "completed", WorkerResult: &codex.WorkerResult{Duration: 5 * time.Second, Summary: "Done"}}
	emitCodexBuildWorkerFinished(dispatch, result)

	output := buf.String()
	for _, want := range []string{"🔨", "Builder", "Forge-41", "completed"} {
		if !strings.Contains(output, want) {
			t.Errorf("worker finished missing %q\n%s", want, output)
		}
	}
}

func TestInstallJsonModeStillProducesJson(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	workDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	t.Setenv("AETHER_OUTPUT_MODE", "json")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--home-dir", homeDir, "--skip-build-binary"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("expected JSON output, got parse error: %v\n%s", err, buf.String())
	}
	if parsed["ok"] != true {
		t.Fatalf("expected ok:true envelope, got %v", parsed)
	}
}

// --- Medic Caste Tests (Phase 25) ---

func TestCasteIdentityMedic(t *testing.T) {
	os.Setenv("AETHER_FORCE_COLOR", "1")
	defer os.Unsetenv("AETHER_FORCE_COLOR")

	const medicEmoji = "🩹"

	identity := casteIdentity("medic")
	if !strings.Contains(identity, medicEmoji) {
		t.Errorf("casteIdentity(medic): expected 🩹 emoji, got %q", identity)
	}
	if !strings.Contains(identity, "Medic") {
		t.Errorf("casteIdentity(medic): expected 'Medic' label, got %q", identity)
	}
	emoji := casteEmoji("medic")
	if emoji != medicEmoji {
		t.Errorf("casteEmoji(medic): expected 🩹, got %q", emoji)
	}
	label := casteLabel("medic")
	if label != "Medic" {
		t.Errorf("casteLabel(medic): expected 'Medic', got %q", label)
	}
	color := casteANSIColor("medic")
	if color == "" {
		t.Errorf("casteANSIColor(medic): expected a color code, got empty")
	}
}

// --- Emoji Consistency Tests (Phase 20) ---

func TestCasteEmojiMapCompleteness(t *testing.T) {
	for key := range casteLabelMap {
		if _, ok := casteEmojiMap[key]; !ok {
			t.Errorf("casteEmojiMap missing key %q (present in casteLabelMap)", key)
		}
		if _, ok := casteColorMap[key]; !ok {
			t.Errorf("casteColorMap missing key %q (present in casteLabelMap)", key)
		}
	}
	for key := range casteEmojiMap {
		if _, ok := casteLabelMap[key]; !ok {
			t.Errorf("casteEmojiMap has extra key %q (not in casteLabelMap)", key)
		}
	}
	for key := range casteColorMap {
		if _, ok := casteLabelMap[key]; !ok {
			t.Errorf("casteColorMap has extra key %q (not in casteLabelMap)", key)
		}
	}
	for key, emoji := range casteEmojiMap {
		if emoji == "" {
			t.Errorf("casteEmojiMap[%q] is empty", key)
		}
	}
	for key, label := range casteLabelMap {
		if label == "" {
			t.Errorf("casteLabelMap[%q] is empty", key)
		}
	}
}

func TestCommandEmojiMapNoEmptyValues(t *testing.T) {
	for key, emoji := range commandEmojiMap {
		if emoji == "" {
			t.Errorf("commandEmojiMap[%q] is empty", key)
		}
	}
}

func TestCommandEmojiFallback(t *testing.T) {
	got := commandEmoji("nonexistent-command-xyz")
	if got != "🐜" {
		t.Errorf("commandEmoji fallback: got %q, want 🐜", got)
	}
}

func TestWrapperDescriptionEmojiConsistency(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	dirs := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant"),
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read dir %s: %v", dir, err)
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			fp := filepath.Join(dir, entry.Name())
			content, err := os.ReadFile(fp)
			if err != nil {
				t.Fatalf("read %s: %v", fp, err)
			}
			text := string(content)

			for _, line := range strings.Split(text, "\n") {
				if !strings.HasPrefix(line, "description:") {
					continue
				}
				// Skip help.md — its command emoji IS 🐜
				if strings.Contains(entry.Name(), "help") {
					continue
				}
				// Check that 🐜 does not appear in a sandwich pattern
				// Sandwich = content between two emoji around 🐜
				if strings.Contains(line, "🐜") {
					t.Errorf("%s: description still contains 🐜 sandwich: %s", fp, line)
				}
				break
			}
		}
	}
}

// --- Codex Visual Parity Tests (Phase 21) ---

func TestCodexVisualParity(t *testing.T) {
	t.Run("CasteIdentity", func(t *testing.T) {
		// Codex uses the same casteIdentity() function — verify it renders correctly
		for _, caste := range []string{"builder", "watcher", "scout", "queen"} {
			identity := casteIdentity(caste)
			emoji := casteEmoji(caste)
			label := casteLabel(caste)
			if !strings.Contains(identity, emoji) {
				t.Errorf("casteIdentity(%q) missing emoji %q: %s", caste, emoji, identity)
			}
			if !strings.Contains(identity, label) {
				t.Errorf("casteIdentity(%q) missing label %q: %s", caste, label, identity)
			}
		}
	})

	t.Run("StageSeparators", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		dataDir := setupBuildFlowTest(t)
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		goal := "Codex parity check"
		taskID := "task-parity"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
			Plan: colony.Plan{
				Phases: []colony.Phase{
					{ID: 1, Name: "Parity", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskID, Goal: "Check parity", Status: colony.TaskPending}}},
				},
			},
		})

		rootCmd.SetArgs([]string{"build", "1"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build error: %v", err)
		}

		output := stdout.(*bytes.Buffer).String()
		for _, marker := range []string{"── Context ──", "── Tasks ──", "── Dispatch ──", "── Verification ──", "── Housekeeping ──", "── Colony Complete ──"} {
			if !strings.Contains(output, marker) {
				t.Errorf("build visual missing stage marker %q (Codex parity)", marker)
			}
		}
	})

	t.Run("CommandEmojiParity", func(t *testing.T) {
		// Verify commandEmoji returns expected values for key commands
		parity := map[string]string{
			"build":    "🔨",
			"continue": "👁️",
			"init":     "🥚",
			"plan":     "📋",
			"seal":     "🏺",
			"status":   "📊",
			"patrol":   "📊",
			"history":  "📜",
			"pause":    "💾",
			"resume":   "💾",
		}
		for cmd, wantEmoji := range parity {
			got := commandEmoji(cmd)
			if got != wantEmoji {
				t.Errorf("commandEmoji(%q) = %q, want %q", cmd, got, wantEmoji)
			}
		}
	})

	t.Run("SpawnListParity", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		dataDir := setupBuildFlowTest(t)
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		goal := "Spawn parity"
		taskID := "task-spawn"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
			Plan: colony.Plan{
				Phases: []colony.Phase{
					{ID: 1, Name: "Spawn", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskID, Goal: "Spawn check", Status: colony.TaskPending}}},
				},
			},
		})

		rootCmd.SetArgs([]string{"build", "1"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build error: %v", err)
		}

		output := stdout.(*bytes.Buffer).String()
		if !strings.Contains(output, "S P A W N   P L A N") {
			t.Error("build visual missing SPAWN PLAN header (Codex parity)")
		}
		if !strings.Contains(output, "Builder") {
			t.Error("build visual missing Builder in spawn list (Codex parity)")
		}
	})

	t.Run("VisualModeRouting", func(t *testing.T) {
		// Simulate Codex invocation: AETHER_OUTPUT_MODE=visual
		t.Setenv("AETHER_OUTPUT_MODE", "visual")
		if !shouldRenderVisualOutput(stdout) {
			t.Error("shouldRenderVisualOutput() false with AETHER_OUTPUT_MODE=visual (Codex parity)")
		}

		// JSON mode should not render visual
		t.Setenv("AETHER_OUTPUT_MODE", "json")
		if shouldRenderVisualOutput(stdout) {
			t.Error("shouldRenderVisualOutput() true with AETHER_OUTPUT_MODE=json")
		}
	})

	t.Run("ContextClearParity", func(t *testing.T) {
		guidance := renderContextClearGuidance()
		if !strings.Contains(guidance, "safe to clear") {
			t.Errorf("renderContextClearGuidance() missing clear guidance: %s", guidance)
		}
	})
}
