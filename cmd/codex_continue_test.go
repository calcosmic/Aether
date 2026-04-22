package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestContinueConsumesBuildPacketAndAdvancesPhase(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Advance the verified phase"
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
					Name:        "Verify the build packet",
					Description: "Close the live build workers after verification",
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

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-11", Task: "Implement the packet", Status: "spawned", TaskID: taskOneID},
		{Stage: "wave", Wave: 1, Caste: "scout", Name: "Ranger-12", Task: "Research the packet", Status: "spawned", TaskID: taskTwoID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-13", Task: "Independent verification before advancement", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Verify the build packet", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true, got %v", result)
	}
	if blocked, _ := result["blocked"].(bool); blocked {
		t.Fatalf("expected unblocked continue result, got %v", result)
	}
	if nextPhase := int(result["next_phase"].(float64)); nextPhase != 2 {
		t.Fatalf("next_phase = %d, want 2", nextPhase)
	}

	for _, rel := range []string{
		"build/phase-1/verification.json",
		"build/phase-1/gates.json",
		"build/phase-1/continue.json",
	} {
		if _, err := os.Stat(filepath.Join(dataDir, rel)); err != nil {
			t.Fatalf("expected report %s: %v", rel, err)
		}
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY", state.State)
	}
	if state.BuildStartedAt != nil {
		t.Fatal("expected BuildStartedAt to be cleared")
	}
	if state.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Fatalf("phase 1 status = %s, want completed", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[1].Status != colony.PhaseReady {
		t.Fatalf("phase 2 status = %s, want ready", state.Plan.Phases[1].Status)
	}

	spawnTreeData, err := os.ReadFile(filepath.Join(dataDir, "spawn-tree.txt"))
	if err != nil {
		t.Fatalf("failed to read spawn tree: %v", err)
	}
	for _, want := range []string{
		"|Forge-11|completed|Completed before continue verification",
		"|Ranger-12|completed|Completed before continue verification",
		"|Keen-13|completed|Verification passed during continue",
	} {
		if !strings.Contains(string(spawnTreeData), want) {
			t.Fatalf("spawn tree missing completion line %q\n%s", want, string(spawnTreeData))
		}
	}

	contextData, err := os.ReadFile(filepath.Join(root, ".aether", "CONTEXT.md"))
	if err != nil {
		t.Fatalf("expected CONTEXT.md: %v", err)
	}
	if !strings.Contains(string(contextData), "aether build 2") {
		t.Fatalf("expected CONTEXT.md to point at the next build, got:\n%s", string(contextData))
	}

	handoffData, err := os.ReadFile(filepath.Join(root, ".aether", "HANDOFF.md"))
	if err != nil {
		t.Fatalf("expected HANDOFF.md: %v", err)
	}
	if !strings.Contains(string(handoffData), "Keep moving") {
		t.Fatalf("expected HANDOFF.md to include next-phase task, got:\n%s", string(handoffData))
	}
}

func TestContinueRecordsWorkerFlowInStateReportAndSpawnSummary(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Record continue worker flow"
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
					Name:   "Continue bookkeeping",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Record worker flow", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next verified phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Carry on", Status: colony.TaskPending}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-14", Task: "Record worker flow", Status: "spawned", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-15", Task: "Independent verification before advancement", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Continue bookkeeping", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	expectedWatcher := deterministicAntName("watcher", "phase:1:continue:watcher")
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	eventText := strings.Join(state.Events, "\n")
	for _, want := range []string{
		"continue_review|continue|",
		"watcher_verification|continue|Watcher " + expectedWatcher + " completed",
		"signal_housekeeping|continue|Signal housekeeping",
	} {
		if !strings.Contains(eventText, want) {
			t.Fatalf("expected state events to contain %q, got:\n%s", want, eventText)
		}
	}

	var report map[string]interface{}
	if err := store.LoadJSON("build/phase-1/continue.json", &report); err != nil {
		t.Fatalf("load continue report: %v", err)
	}
	flow, _ := report["worker_flow"].([]interface{})
	if len(flow) < 5 {
		t.Fatalf("expected worker_flow to record review, watcher verification, and housekeeping, got %#v", report["worker_flow"])
	}

	flowText, err := json.Marshal(flow)
	if err != nil {
		t.Fatalf("marshal worker_flow: %v", err)
	}
	for _, want := range []string{"review", expectedWatcher, "Signal housekeeping"} {
		if !strings.Contains(string(flowText), want) {
			t.Fatalf("expected worker_flow to include %q, got %s", want, string(flowText))
		}
	}
	if strings.Contains(string(flowText), "Forge-14") {
		t.Fatalf("expected continue worker_flow to avoid synthetic builder closure entries, got %s", string(flowText))
	}

	summary := loadSpawnActivitySummaryForState(store, &state)
	if summary.CurrentCommand != "continue" {
		t.Fatalf("expected current spawn run to be continue, got %q", summary.CurrentCommand)
	}
	if len(summary.RecentOutcomeEntries) == 0 {
		t.Fatalf("expected continue spawn summary to expose recent outcomes, got %+v", summary)
	}

	recentText, err := json.Marshal(summary.RecentOutcomeEntries)
	if err != nil {
		t.Fatalf("marshal recent outcomes: %v", err)
	}
	for _, want := range []string{"Signal housekeeping", "Gatekeeper continue review"} {
		if !strings.Contains(string(recentText), want) {
			t.Fatalf("expected recent continue outcomes to include %q, got %s", want, string(recentText))
		}
	}
	for _, unwanted := range []string{"Forge-14", "Keen-15"} {
		if strings.Contains(string(recentText), unwanted) {
			t.Fatalf("expected recent continue outcomes to avoid build-worker closure entry %q, got %s", unwanted, string(recentText))
		}
	}
}

func TestContinueRollsBackStateWhenContextUpdateFails(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Rollback continue state when context update fails"
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
					Name:   "Rollback phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Keep state pending until finalize succeeds", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Advance only after finalization", Status: colony.TaskPending}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-16", Task: "Keep state pending until finalize succeeds", Status: "spawned", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-17", Task: "Independent verification before advancement", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Rollback phase", goal, dispatches)
	origUpdater := continueContextUpdater
	continueContextUpdater = func(colony.Phase, codexContinueManifest, []codexContinueClosedWorker, time.Time) error {
		return fmt.Errorf("forced context update failure")
	}
	t.Cleanup(func() {
		continueContextUpdater = origUpdater
	})

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned unexpected error: %v", err)
	}
	output := stderr.(*bytes.Buffer).String()
	if !strings.Contains(output, "forced context update failure") {
		t.Fatalf("expected continue stderr to report the forced context failure, got:\n%s", output)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT after rollback", state.State)
	}
	if state.CurrentPhase != 1 {
		t.Fatalf("current phase = %d, want 1 after rollback", state.CurrentPhase)
	}
	if state.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("phase 1 status = %s, want in_progress after rollback", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[1].Status != colony.PhasePending {
		t.Fatalf("phase 2 status = %s, want pending after rollback", state.Plan.Phases[1].Status)
	}
}

func TestContinueDoesNotCloseBuildWorkersWhenContextUpdateFails(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Keep build workers open until context updates succeed"
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
					Name:   "Context rollback phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Do not close workers yet", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Remain pending", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Context rollback phase", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-18", Task: "Do not close workers yet", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-19", Task: "Verify the phase", Status: "completed"},
	})

	spawnTreePath := filepath.Join(dataDir, "spawn-tree.txt")
	before, err := os.ReadFile(spawnTreePath)
	if err != nil {
		t.Fatalf("read spawn tree before continue: %v", err)
	}

	origUpdater := continueContextUpdater
	continueContextUpdater = func(colony.Phase, codexContinueManifest, []codexContinueClosedWorker, time.Time) error {
		return fmt.Errorf("forced context update failure")
	}
	t.Cleanup(func() {
		continueContextUpdater = origUpdater
	})

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned unexpected error: %v", err)
	}

	after, err := os.ReadFile(spawnTreePath)
	if err != nil {
		t.Fatalf("read spawn tree after continue: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("expected spawn tree to remain unchanged when context update fails\nbefore:\n%s\nafter:\n%s", string(before), string(after))
	}
}

func TestContinueDoesNotAdvanceStateWhenHousekeepingFails(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Do not advance on housekeeping failure"
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
					Name:   "Housekeeping failure phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Stay put on failure", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase should not unlock",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Remain pending", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Housekeeping failure phase", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-24", Task: "Stay put on failure", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-25", Task: "Verify the phase", Status: "completed"},
	})

	continueSignalHousekeeper = func(now time.Time, state colony.ColonyState) (signalHousekeepingResult, error) {
		return signalHousekeepingResult{}, fmt.Errorf("housekeeping exploded")
	}

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	if !strings.Contains(stderr.(*bytes.Buffer).String(), "housekeeping exploded") {
		t.Fatalf("expected housekeeping failure in stderr, got: %s", stderr.(*bytes.Buffer).String())
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.CurrentPhase != 1 {
		t.Fatalf("current phase = %d, want 1", state.CurrentPhase)
	}
	if state.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("phase 1 status = %s, want in_progress", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[1].Status != colony.PhasePending {
		t.Fatalf("phase 2 status = %s, want pending", state.Plan.Phases[1].Status)
	}

	if _, err := os.Stat(filepath.Join(dataDir, "build", "phase-1", "continue.json")); !os.IsNotExist(err) {
		t.Fatalf("expected continue report to be absent on housekeeping failure, err=%v", err)
	}
}

func TestContinueDoesNotCloseBuildWorkersWhenHousekeepingFails(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Keep build workers open until housekeeping succeeds"
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
					Name:   "Housekeeping failure phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Do not close workers yet", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase should not unlock",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Remain pending", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Housekeeping failure phase", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-28", Task: "Do not close workers yet", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-29", Task: "Verify the phase", Status: "completed"},
	})

	spawnTreePath := filepath.Join(dataDir, "spawn-tree.txt")
	before, err := os.ReadFile(spawnTreePath)
	if err != nil {
		t.Fatalf("read spawn tree before continue: %v", err)
	}

	continueSignalHousekeeper = func(now time.Time, state colony.ColonyState) (signalHousekeepingResult, error) {
		return signalHousekeepingResult{}, fmt.Errorf("housekeeping exploded")
	}

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	after, err := os.ReadFile(spawnTreePath)
	if err != nil {
		t.Fatalf("read spawn tree after continue: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("expected spawn tree to remain unchanged when housekeeping fails\nbefore:\n%s\nafter:\n%s", string(before), string(after))
	}
}

func TestContinueDoesNotRewriteContextWhenHousekeepingFails(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Preserve context until housekeeping succeeds"
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
					Name:   "Housekeeping failure phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Stay put on failure", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase should not unlock",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Remain pending", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Housekeeping failure phase", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-26", Task: "Stay put on failure", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-27", Task: "Verify the phase", Status: "completed"},
	})

	contextPath := filepath.Join(root, ".aether", "CONTEXT.md")
	contextBefore := `# Aether Colony — Current Context

## System Status

| Field | Value |
|-------|-------|
| **Last Updated** | 2026-04-22T00:00:00Z |
| **Current Phase** | 1 |
| **Phase Name** | Housekeeping failure phase |
| **Milestone** | First Mound |
| **Colony Status** | executing |
| **Safe to Clear?** | NO — Build in progress |

## What's In Progress

Phase 1 Build IN PROGRESS
Workers: 2
  - 2026-04-22T00:00:00Z: Spawned Forge-26 (builder) for: Stay put on failure
  - 2026-04-22T00:00:00Z: Spawned Keen-27 (watcher) for: Verify the phase
`
	if err := os.MkdirAll(filepath.Dir(contextPath), 0755); err != nil {
		t.Fatalf("mkdir context dir: %v", err)
	}
	if err := os.WriteFile(contextPath, []byte(contextBefore), 0644); err != nil {
		t.Fatalf("write context: %v", err)
	}

	continueSignalHousekeeper = func(now time.Time, state colony.ColonyState) (signalHousekeepingResult, error) {
		return signalHousekeepingResult{}, fmt.Errorf("housekeeping exploded")
	}

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	contextAfter, err := os.ReadFile(contextPath)
	if err != nil {
		t.Fatalf("read CONTEXT.md: %v", err)
	}
	if string(contextAfter) != contextBefore {
		t.Fatalf("expected CONTEXT.md to remain unchanged on housekeeping failure, got:\n%s", string(contextAfter))
	}
}

func TestContinueExpiresWorkerContinueSignalsUsingAdvancedPhaseState(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Expire stale continue guidance after three completed phases"
	now := time.Now().UTC()
	taskID := "4.1"
	nextTaskID := "5.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   4,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "phase 1", Status: colony.PhaseCompleted},
				{ID: 2, Name: "phase 2", Status: colony.PhaseCompleted},
				{ID: 3, Name: "phase 3", Status: colony.PhaseCompleted},
				{
					ID:     4,
					Name:   "phase 4",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "finish phase 4", Status: colony.TaskInProgress}},
				},
				{
					ID:     5,
					Name:   "phase 5",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "pick up the next phase", Status: colony.TaskPending}},
				},
			},
		},
		Events: []string{
			"2026-04-12T09:00:00Z|phase_advanced|continue|Completed phase 1, ready for phase 2",
			"2026-04-13T09:00:00Z|phase_advanced|continue|Completed phase 2, ready for phase 3",
			"2026-04-14T09:00:00Z|phase_advanced|continue|Completed phase 3, ready for phase 4",
		},
	})

	seedContinueBuildPacket(t, dataDir, 4, "phase 4", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-401", Task: "finish phase 4", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-402", Task: "Verify phase 4", Status: "completed"},
	})

	s1_0 := 1.0
	writeTestPheromones(t, dataDir, colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:        "sig_stale_continue",
				Type:      "FEEDBACK",
				Priority:  "low",
				Source:    "worker:continue",
				CreatedAt: "2026-04-12T12:00:00Z",
				Active:    true,
				Strength:  &s1_0,
				Content:   json.RawMessage(`{"text":"stale continue guidance"}`),
			},
		},
	})

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	housekeeping := result["signal_housekeeping"].(map[string]interface{})
	if housekeeping["expired_worker_continue"] != float64(1) {
		t.Fatalf("expired_worker_continue = %v, want 1", housekeeping["expired_worker_continue"])
	}

	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("reload pheromones: %v", err)
	}
	if pf.Signals[0].Active {
		t.Fatal("expected stale worker:continue signal to be deactivated after continue advancement")
	}
}

func TestContinueCompletesFinalPhase(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Finish the colony"
	now := time.Now().UTC()
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
					Name:   "Final phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Finish it", Status: colony.TaskInProgress}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-21", Task: "Finish it", Status: "spawned", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-22", Task: "Independent verification before advancement", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Final phase", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if completed, _ := result["completed"].(bool); !completed {
		t.Fatalf("expected completed:true, got %v", result)
	}
	if next := result["next"].(string); next != "aether seal" {
		t.Fatalf("next = %q, want aether seal", next)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Fatalf("state = %s, want COMPLETED", state.State)
	}
}

func TestContinueBlocksWhenWatcherUsesFakeInvoker(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Block advancement when watcher is FakeInvoker"
	now := time.Now().UTC()
	taskOneID := "1.1"
	taskTwoID := "1.2"
	taskThreeID := "1.3"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Watcher blocks on FakeInvoker",
					Status: colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Do work one", Status: colony.TaskInProgress},
						{ID: &taskTwoID, Goal: "Do work two", Status: colony.TaskInProgress},
						{ID: &taskThreeID, Goal: "Do work three", Status: colony.TaskInProgress},
					},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-31", Task: "Do work one", Status: "completed", TaskID: taskOneID},
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-32", Task: "Do work two", Status: "completed", TaskID: taskTwoID},
		{Stage: "wave", Wave: 1, Caste: "scout", Name: "Ranger-33", Task: "Do work three", Status: "completed", TaskID: taskThreeID},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Watcher blocks on FakeInvoker", goal, dispatches)

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &continueUnavailableInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true when FakeInvoker is used, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false when FakeInvoker is used, got %v", result)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.State != colony.StateEXECUTING {
		t.Fatalf("state = %s, want EXECUTING", state.State)
	}
}

func TestContinueBlocksWhenWorkerTimesOut(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Block advancement when a worker timed out"
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
					ID:     1,
					Name:   "Timeout worker phase",
					Status: colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Land the first change", Status: colony.TaskInProgress},
						{ID: &taskTwoID, Goal: "Land the second change", Status: colony.TaskInProgress},
					},
				},
				{
					ID:     2,
					Name:   "Next phase should not unlock",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Wait for redispatch", Status: colony.TaskPending}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-61", Task: "Land the first change", Status: "completed", TaskID: taskOneID},
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-62", Task: "Land the second change", Status: "timeout", TaskID: taskTwoID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-63", Task: "Independent verification before advancement", Status: "completed"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Timeout worker phase", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true when a worker timed out, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false when a worker timed out, got %v", result)
	}

	recovery := result["recovery"].(map[string]interface{})
	if got := recovery["redispatch_command"].(string); got != "aether build 1 --task 1.2" {
		t.Fatalf("redispatch command = %q, want %q", got, "aether build 1 --task 1.2")
	}

	issues := stringSliceValue(result["operational_issues"])
	if len(issues) == 0 {
		t.Fatalf("expected operational issues in blocked result, got %v", result)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("phase 1 status = %s, want in_progress", state.Plan.Phases[0].Status)
	}
}

func TestContinueAdvancesWhenFreshWatcherPassesDespiteStaleBuildWatcher(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Trust fresh continue watcher over stale build watcher"
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
					Name:   "Watcher gate phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Ship the verified change", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Advance after fresh watcher verification", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Watcher gate phase", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-64", Task: "Ship the verified change", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-65", Task: "Independent verification before advancement", Status: "blocked", Summary: "Watcher could not confirm the change"},
	})

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); blocked {
		t.Fatalf("expected blocked:false when fresh continue watcher passes, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true when fresh continue watcher passes, got %v", result)
	}
	if next := result["next"].(string); next != "aether build 2" {
		t.Fatalf("next = %q, want aether build 2", next)
	}

	verification := result["verification"].(map[string]interface{})
	watcher := verification["watcher"].(map[string]interface{})
	if passed, _ := watcher["passed"].(bool); !passed {
		t.Fatalf("expected fresh continue watcher verification to pass, got %v", watcher)
	}
	if status := watcher["status"].(string); status != "completed" {
		t.Fatalf("watcher status = %q, want completed", status)
	}
	blocking := stringSliceValue(result["blocking_issues"])
	if containsString(blocking, "Watcher could not confirm the change") {
		t.Fatalf("blocking_issues = %v, stale build watcher should not block", blocking)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY", state.State)
	}
	if state.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Fatalf("phase 1 status = %s, want completed", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[1].Status != colony.PhaseReady {
		t.Fatalf("phase 2 status = %s, want ready", state.Plan.Phases[1].Status)
	}
}

func TestContinueBlocksWhenContinueWatcherRejectsPhase(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Require a continue-time watcher before advancement"
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
					Name:   "Continue watcher gate",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Ship only after fresh watcher verification", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Still blocked",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Wait for continue-time verification", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Continue watcher gate", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-101", Task: "Ship only after fresh watcher verification", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-build-102", Task: "Build-time verification", Status: "completed"},
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

	if invoker.watcherCalls != 1 {
		t.Fatalf("expected exactly one continue watcher dispatch, got %d", invoker.watcherCalls)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false, got %v", result)
	}
	if next := result["next"].(string); next != "aether continue" {
		t.Fatalf("next = %q, want aether continue when the continue watcher blocks advancement", next)
	}

	verification := result["verification"].(map[string]interface{})
	watcher := verification["watcher"].(map[string]interface{})
	if worker := watcher["worker"].(string); worker != invoker.watcherName {
		t.Fatalf("watcher worker = %q, want %q", worker, invoker.watcherName)
	}
	if summary := watcher["summary"].(string); summary != "Continue watcher rejected the phase" {
		t.Fatalf("watcher summary = %q, want continue watcher rejection", summary)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.Plan.Phases[1].Status != colony.PhasePending {
		t.Fatalf("phase 2 status = %s, want pending", state.Plan.Phases[1].Status)
	}
}

func TestContinueBlocksWhenWatcherTimesOut(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Block advancement when continue watcher times out"
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
					Name:   "Watcher timeout phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Ship only after watcher completes", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Still blocked",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Wait for non-timed-out watcher", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Watcher timeout phase", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-201", Task: "Ship only after watcher completes", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-build-202", Task: "Build-time verification", Status: "completed"},
	})

	invoker := &continueWatcherTestInvoker{
		watcherStatus:  "timeout",
		watcherSummary: "Continue watcher timed out during independent verification",
	}
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true when watcher times out, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false when watcher times out, got %v", result)
	}

	verification := result["verification"].(map[string]interface{})
	if passed, _ := verification["checks_passed"].(bool); passed {
		t.Fatalf("expected checks_passed:false when watcher times out, got %v", verification)
	}
	watcher := verification["watcher"].(map[string]interface{})
	if passed, _ := watcher["passed"].(bool); passed {
		t.Fatalf("expected watcher.passed:false on timeout, got %v", watcher)
	}
	if status := watcher["status"].(string); status != "timeout" {
		t.Fatalf("watcher status = %q, want timeout", status)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("phase 1 status = %s, want in_progress", state.Plan.Phases[0].Status)
	}
}

func TestContinueBlockedFlowIsRecordedInStateAndSpawnSummary(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Record blocked continue worker flow"
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
					Name:   "Blocked continue flow",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Stay blocked until watcher signs off", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Still pending",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Wait for approval", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Blocked continue flow", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-121", Task: "Stay blocked until watcher signs off", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-build-122", Task: "Build-time verification", Status: "completed"},
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

	if invoker.watcherName == "" {
		t.Fatal("expected continue watcher name to be captured")
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	eventText := strings.Join(state.Events, "\n")
	wantEvent := "watcher_verification|continue|Watcher " + invoker.watcherName + " closed independent verification with status blocked: Continue watcher rejected the phase"
	if !strings.Contains(eventText, wantEvent) {
		t.Fatalf("expected blocked continue watcher event %q, got:\n%s", wantEvent, eventText)
	}

	summary := loadSpawnActivitySummaryForState(store, &state)
	if summary.CurrentCommand != "continue" {
		t.Fatalf("expected current spawn run to be continue, got %q", summary.CurrentCommand)
	}
	if len(summary.RecentOutcomeEntries) == 0 {
		t.Fatalf("expected blocked continue to expose recent outcomes, got %+v", summary)
	}

	recentText, err := json.Marshal(summary.RecentOutcomeEntries)
	if err != nil {
		t.Fatalf("marshal recent outcomes: %v", err)
	}
	if !strings.Contains(string(recentText), invoker.watcherName) {
		t.Fatalf("expected blocked continue outcomes to include watcher %q, got %s", invoker.watcherName, string(recentText))
	}
	if !strings.Contains(string(recentText), "Continue watcher rejected the phase") {
		t.Fatalf("expected blocked continue outcomes to preserve watcher rejection summary, got %s", string(recentText))
	}
	if strings.Contains(string(recentText), "Forge-121") {
		t.Fatalf("expected blocked continue outcomes to avoid builder closure entries, got %s", string(recentText))
	}
}

func TestContinueWorkerFlowUsesContinueWatcherInsteadOfBuildManifestWatcher(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Record the actual continue watcher"
	now := time.Now().UTC()
	taskID := "1.1"
	nextTaskID := "2.1"
	buildWatcherName := "Keen-build-111"
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
					Name:   "Record continue watcher flow",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Verify with a fresh watcher", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Advance once the continue watcher clears", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Record continue watcher flow", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-110", Task: "Verify with a fresh watcher", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: buildWatcherName, Task: "Build-time verification", Status: "completed"},
	})

	invoker := &continueWatcherTestInvoker{
		watcherStatus:  "completed",
		watcherSummary: "Continue watcher approved the phase",
	}
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	if invoker.watcherCalls != 1 {
		t.Fatalf("expected exactly one continue watcher dispatch, got %d", invoker.watcherCalls)
	}
	if invoker.watcherName == "" {
		t.Fatal("expected continue watcher name to be captured")
	}

	var report map[string]interface{}
	if err := store.LoadJSON("build/phase-1/continue.json", &report); err != nil {
		t.Fatalf("load continue report: %v", err)
	}
	flow, _ := report["worker_flow"].([]interface{})
	flowText, err := json.Marshal(flow)
	if err != nil {
		t.Fatalf("marshal worker_flow: %v", err)
	}
	if !strings.Contains(string(flowText), invoker.watcherName) {
		t.Fatalf("expected continue worker flow to include fresh watcher %q, got %s", invoker.watcherName, string(flowText))
	}
	if strings.Contains(string(flowText), buildWatcherName) {
		t.Fatalf("expected continue worker flow to avoid build manifest watcher %q, got %s", buildWatcherName, string(flowText))
	}
}

func TestContinueBlocksWhenReconciledTaskLacksClaimEvidence(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Block when reconciled task lacks builder claim evidence"
	now := time.Now().UTC()
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
					Name:   "Manual recovery phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Land the manual fix", Status: colony.TaskInProgress}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-71", Task: "Land the manual fix", Status: "failed", TaskID: taskID},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Manual recovery phase", goal, dispatches)
	if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
		BuildPhase: 1,
		Timestamp:  now.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("failed to overwrite build claims: %v", err)
	}

	rootCmd.SetArgs([]string{"continue", "--reconcile-task", taskID})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true when reconciled task lacks claim evidence, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false when reconciled task lacks claim evidence, got %v", result)
	}

	reconciled := stringSliceValue(result["reconciled_tasks"])
	if len(reconciled) != 1 || reconciled[0] != taskID {
		t.Fatalf("expected reconciled task %s, got %v", taskID, reconciled)
	}

	blockingIssues := stringSliceValue(result["blocking_issues"])
	hasWarning := false
	for _, issue := range blockingIssues {
		if strings.Contains(issue, "manually reconciled") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Fatalf("expected blocking issues to contain reconcile warning, got %v", blockingIssues)
	}
}

func TestContinueBlocksWhenUnreconciledTaskStillHasNoEvidence(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Reconcile only the task that was manually completed"
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
					ID:     1,
					Name:   "Partial manual reconciliation",
					Status: colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Apply the manual fix", Status: colony.TaskInProgress},
						{ID: &taskTwoID, Goal: "Keep the second task accounted for", Status: colony.TaskInProgress},
					},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Advance only after all tasks are covered", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Partial manual reconciliation", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-72", Task: "Apply the manual fix", Status: "completed", TaskID: taskOneID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-73", Task: "Independent verification before advancement", Status: "completed"},
	})

	rootCmd.SetArgs([]string{"continue", "--reconcile-task", taskOneID})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true when task %s still lacks evidence, got %v", taskTwoID, result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false when task %s still lacks evidence, got %v", taskTwoID, result)
	}

	taskEvidence, _ := result["task_evidence"].([]interface{})
	if len(taskEvidence) != 2 {
		t.Fatalf("expected task evidence for both tasks, got %#v", result["task_evidence"])
	}
	secondTask := taskEvidence[1].(map[string]interface{})
	if got := secondTask["outcome"].(string); got != "missing" {
		t.Fatalf("task %s outcome = %q, want missing", taskTwoID, got)
	}

	recovery := result["recovery"].(map[string]interface{})
	if got := recovery["redispatch_command"].(string); got != "aether build 1 --task 1.2" {
		t.Fatalf("redispatch command = %q, want %q", got, "aether build 1 --task 1.2")
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("phase 1 status = %s, want in_progress", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[1].Status != colony.PhasePending {
		t.Fatalf("phase 2 status = %s, want pending", state.Plan.Phases[1].Status)
	}
}

func TestContinueBlocksWhenBuilderClaimsMismatch(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Block on artifact verification mismatch"
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
					Name:   "Artifact verification phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Write the real artifact", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Follow-up phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Advance only when verified", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Artifact verification phase", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-91", Task: "Write the real artifact", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-92", Task: "Verify the artifact", Status: "completed"},
	})
	if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
		FilesModified: []string{"missing-artifact.txt"},
		BuildPhase:    1,
		Timestamp:     now.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("failed to overwrite build claims: %v", err)
	}

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false, got %v", result)
	}

	verification := result["verification"].(map[string]interface{})
	if passed, _ := verification["checks_passed"].(bool); passed {
		t.Fatalf("expected verification checks to fail when artifact claims mismatch, got %v", verification)
	}
	claims := verification["claims"].(map[string]interface{})
	if passed, _ := claims["passed"].(bool); passed {
		t.Fatalf("expected claims verification to fail, got %v", claims)
	}
	if !strings.Contains(claims["summary"].(string), "worker claims mismatch") {
		t.Fatalf("expected claims mismatch summary, got %q", claims["summary"])
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("phase 1 status = %s, want in_progress", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[1].Status != colony.PhasePending {
		t.Fatalf("phase 2 status = %s, want pending", state.Plan.Phases[1].Status)
	}
}

func TestContinueVerificationUsesDocumentationCommands(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("## Verification Commands\n\n```bash\n# Verify Go binary builds\nprintf claude-build\n\n# Run Go tests\nprintf claude-test\n```\n"), 0644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".aether", "data"), 0755); err != nil {
		t.Fatalf("mkdir codebase dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".aether", "data", "codebase.md"), []byte("## Commands\n- Types: `printf codebase-types`\n- Lint: `printf codebase-lint`\n"), 0644); err != nil {
		t.Fatalf("write codebase.md: %v", err)
	}

	goal := "Honor documented verification commands"
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
					Name:   "Doc-driven verification",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Verify using documented commands", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Advance after documented verification", Status: colony.TaskPending}},
				},
			},
		},
	})
	seedContinueBuildPacket(t, dataDir, 1, "Doc-driven verification", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-doc-1", Task: "Verify using documented commands", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-doc-1", Task: "Independent verification before advancement", Status: "completed"},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true, got %v", result)
	}

	steps := verificationStepsByName(t, result["verification"].(map[string]interface{}))
	if got := steps["build"]["command"].(string); got != "printf claude-build" {
		t.Fatalf("build command = %q, want %q", got, "printf claude-build")
	}
	if got := steps["tests"]["command"].(string); got != "printf claude-test" {
		t.Fatalf("tests command = %q, want %q", got, "printf claude-test")
	}
	if got := steps["types"]["command"].(string); got != "printf codebase-types" {
		t.Fatalf("types command = %q, want %q", got, "printf codebase-types")
	}
	if got := steps["lint"]["command"].(string); got != "printf codebase-lint" {
		t.Fatalf("lint command = %q, want %q", got, "printf codebase-lint")
	}
}

func TestContinueBlocksWhenDocumentedVerificationCommandFails(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("## Verification Commands\n\n```bash\n# Verify Go binary builds\nprintf broken-build && exit 1\n\n# Run Go tests\nprintf claude-test\n```\n"), 0644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".aether", "data"), 0755); err != nil {
		t.Fatalf("mkdir codebase dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".aether", "data", "codebase.md"), []byte("## Commands\n- Types: `printf codebase-types`\n- Lint: `printf codebase-lint`\n"), 0644); err != nil {
		t.Fatalf("write codebase.md: %v", err)
	}

	goal := "Block on documented verification failure"
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
					Name:   "Doc-driven failure",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Fail on the documented build command", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Still blocked",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Advance only after verification passes", Status: colony.TaskPending}},
				},
			},
		},
	})
	seedContinueBuildPacket(t, dataDir, 1, "Doc-driven failure", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-doc-2", Task: "Fail on the documented build command", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-doc-2", Task: "Independent verification before advancement", Status: "completed"},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false, got %v", result)
	}

	steps := verificationStepsByName(t, result["verification"].(map[string]interface{}))
	build := steps["build"]
	if got := build["command"].(string); got != "printf broken-build && exit 1" {
		t.Fatalf("build command = %q, want %q", got, "printf broken-build && exit 1")
	}
	if passed, _ := build["passed"].(bool); passed {
		t.Fatalf("expected documented build command to fail, got %v", build)
	}
}

func TestExtractMarkdownSectionPreservesFencedCommandComments(t *testing.T) {
	content := "## Verification Commands\n\n```bash\n# Verify Go binary builds\nprintf claude-build\n\n# Run Go tests\nprintf claude-test\n```\n\n## Another Section\nignored\n"

	section := extractMarkdownSection(content, "## Verification Commands")

	for _, want := range []string{
		"# Verify Go binary builds",
		"printf claude-build",
		"# Run Go tests",
		"printf claude-test",
	} {
		if !strings.Contains(section, want) {
			t.Fatalf("expected section to include %q, got:\n%s", want, section)
		}
	}
	if strings.Contains(section, "Another Section") {
		t.Fatalf("expected section extraction to stop at the next real heading, got:\n%s", section)
	}
}

func TestContinueVerificationUsesAGENTSCommands(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("## Verification Commands\n\n```bash\n# Verify Go binary builds\nprintf agents-build\n\n# Run Go tests\nprintf agents-test\n```\n"), 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".aether", "data"), 0755); err != nil {
		t.Fatalf("mkdir codebase dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".aether", "data", "codebase.md"), []byte("## Commands\n- Types: `printf codebase-types`\n- Lint: `printf codebase-lint`\n"), 0644); err != nil {
		t.Fatalf("write codebase.md: %v", err)
	}

	goal := "Honor AGENTS verification commands"
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
					Name:   "AGENTS-driven verification",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Verify using AGENTS commands", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Advance after AGENTS verification", Status: colony.TaskPending}},
				},
			},
		},
	})
	seedContinueBuildPacket(t, dataDir, 1, "AGENTS-driven verification", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-agents-1", Task: "Verify using AGENTS commands", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-agents-1", Task: "Independent verification before advancement", Status: "completed"},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true, got %v", result)
	}

	steps := verificationStepsByName(t, result["verification"].(map[string]interface{}))
	if got := steps["build"]["command"].(string); got != "printf agents-build" {
		t.Fatalf("build command = %q, want %q", got, "printf agents-build")
	}
	if got := steps["tests"]["command"].(string); got != "printf agents-test" {
		t.Fatalf("tests command = %q, want %q", got, "printf agents-test")
	}
}

func TestResolveCodexVerificationCommandsReadsCODEXAndOPENCODEDocs(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, ".codex"), 0755); err != nil {
		t.Fatalf("mkdir .codex: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".opencode"), 0755); err != nil {
		t.Fatalf("mkdir .opencode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".codex", "CODEX.md"), []byte("### Verification Commands\n\n```bash\n# Verify Go binary builds\nprintf codex-build\n```\n"), 0644); err != nil {
		t.Fatalf("write CODEX.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".opencode", "OPENCODE.md"), []byte("## Verification Commands\n\n```bash\n# Run Go tests\nprintf opencode-test\n```\n"), 0644); err != nil {
		t.Fatalf("write OPENCODE.md: %v", err)
	}

	commands := resolveCodexVerificationCommands(root)
	if commands.Build != "printf codex-build" {
		t.Fatalf("build command = %q, want %q", commands.Build, "printf codex-build")
	}
	if commands.Test != "printf opencode-test" {
		t.Fatalf("test command = %q, want %q", commands.Test, "printf opencode-test")
	}
}

func TestContinueReconcileTaskDoesNotTrustOtherTasks(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Reconcile only the task you actually fixed"
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
					ID:     1,
					Name:   "Scoped reconciliation phase",
					Status: colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Land the manual fix", Status: colony.TaskInProgress},
						{ID: &taskTwoID, Goal: "Do not trust this unrelated task", Status: colony.TaskInProgress},
					},
				},
				{
					ID:     2,
					Name:   "Still blocked next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Advance later", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Scoped reconciliation phase", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-71", Task: "Land the manual fix", Status: "failed", TaskID: taskOneID},
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-72", Task: "Do not trust this unrelated task", Status: "completed", TaskID: taskTwoID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-73", Task: "Verify before advancement", Status: "completed"},
	})
	if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
		FilesModified: []string{"missing-artifact.txt"},
		BuildPhase:    1,
		Timestamp:     now.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("failed to overwrite build claims: %v", err)
	}

	rootCmd.SetArgs([]string{"continue", "--reconcile-task", taskOneID})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false, got %v", result)
	}

	recovery := result["recovery"].(map[string]interface{})
	if got := recovery["redispatch_command"].(string); got != "aether build 1 --task 1.2" {
		t.Fatalf("redispatch command = %q, want %q", got, "aether build 1 --task 1.2")
	}

	taskOutcomes := map[string]string{}
	for _, raw := range result["task_evidence"].([]interface{}) {
		entry := raw.(map[string]interface{})
		taskOutcomes[entry["task_id"].(string)] = entry["outcome"].(string)
	}
	if taskOutcomes[taskOneID] != "manually_reconciled" {
		t.Fatalf("task %s outcome = %q, want manually_reconciled", taskOneID, taskOutcomes[taskOneID])
	}
	if taskOutcomes[taskTwoID] != "implemented_unverified" {
		t.Fatalf("task %s outcome = %q, want implemented_unverified", taskTwoID, taskOutcomes[taskTwoID])
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.Plan.Phases[1].Status != colony.PhasePending {
		t.Fatalf("phase 2 status = %s, want pending", state.Plan.Phases[1].Status)
	}
}

func TestContinueBlockedResultSuggestsTargetedRedispatch(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() { this_will_not_compile }\n"), 0644); err != nil {
		t.Fatalf("failed to break workspace build: %v", err)
	}

	goal := "Suggest task-scoped redispatch"
	now := time.Now().UTC()
	taskOneID := "1.1"
	taskTwoID := "1.2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Redispatch recovery phase",
					Status: colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Recover the missing implementation", Status: colony.TaskInProgress},
						{ID: &taskTwoID, Goal: "Preserve the completed work", Status: colony.TaskInProgress},
					},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-81", Task: "Recover the missing implementation", Status: "failed", TaskID: taskOneID},
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-82", Task: "Preserve the completed work", Status: "completed", TaskID: taskTwoID},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Redispatch recovery phase", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true, got %v", result)
	}
	recovery := result["recovery"].(map[string]interface{})
	if got := recovery["redispatch_command"].(string); got != "aether build 1 --task 1.1" {
		t.Fatalf("redispatch command = %q, want %q", got, "aether build 1 --task 1.1")
	}

	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		t.Fatalf("load session.json: %v", err)
	}
	if session.SuggestedNext != "aether build 1 --task 1.1" {
		t.Fatalf("session suggested_next = %q, want %q", session.SuggestedNext, "aether build 1 --task 1.1")
	}

	contextData, err := os.ReadFile(filepath.Join(root, ".aether", "CONTEXT.md"))
	if err != nil {
		t.Fatalf("read CONTEXT.md: %v", err)
	}
	if !strings.Contains(string(contextData), "aether build 1 --task 1.1") {
		t.Fatalf("expected CONTEXT.md to carry targeted recovery command, got:\n%s", string(contextData))
	}

	handoffData, err := os.ReadFile(filepath.Join(root, ".aether", "HANDOFF.md"))
	if err != nil {
		t.Fatalf("read HANDOFF.md: %v", err)
	}
	if !strings.Contains(string(handoffData), "aether build 1 --task 1.1") {
		t.Fatalf("expected HANDOFF.md to carry targeted recovery command, got:\n%s", string(handoffData))
	}
}

func TestContinueUsesManifestWhenBuildStartedAtIsMissing(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Continue from built manifest without timestamp"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Manifest-backed phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Finish from manifest", Status: colony.TaskInProgress}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-51", Task: "Finish from manifest", Status: "spawned", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-52", Task: "Independent verification before advancement", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Manifest-backed phase", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if completed, _ := result["completed"].(bool); !completed {
		t.Fatalf("expected completed:true, got %v", result)
	}
}

func seedContinueBuildPacket(t *testing.T, dataDir string, phase int, phaseName, goal string, dispatches []codexBuildDispatch) {
	t.Helper()

	buildDir := filepath.Join(dataDir, "build", fmt.Sprintf("phase-%d", phase))
	if err := os.MkdirAll(filepath.Join(buildDir, "worker-briefs"), 0755); err != nil {
		t.Fatalf("failed to create worker brief dir: %v", err)
	}

	normalizedDispatches := make([]codexBuildDispatch, len(dispatches))
	copy(normalizedDispatches, dispatches)
	for i := range normalizedDispatches {
		if normalizedDispatches[i].Status == "" || normalizedDispatches[i].Status == "spawned" {
			normalizedDispatches[i].Status = "completed"
		}
	}

	briefs := make([]string, 0, len(normalizedDispatches))
	for _, dispatch := range normalizedDispatches {
		rel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase), "worker-briefs", dispatch.Name+".md"))
		if err := store.AtomicWrite(rel, []byte("# brief\n")); err != nil {
			t.Fatalf("failed to write worker brief: %v", err)
		}
		briefs = append(briefs, displayDataPath(rel))
	}

	manifest := codexBuildManifest{
		Phase:        phase,
		PhaseName:    phaseName,
		Goal:         goal,
		Root:         filepath.Dir(filepath.Dir(dataDir)),
		ColonyDepth:  "standard",
		DispatchMode: "real",
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		State:        string(colony.StateBUILT),
		ClaimsPath:   displayDataPath("last-build-claims.json"),
		WorkerBriefs: briefs,
		Dispatches:   normalizedDispatches,
	}
	if err := store.SaveJSON(filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase), "manifest.json")), manifest); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	claims := codexBuildClaims{BuildPhase: phase, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	for _, dispatch := range normalizedDispatches {
		if dispatch.Caste == "builder" {
			claims.FilesModified = append(claims.FilesModified, "main.go")
			break
		}
	}
	if err := store.SaveJSON("last-build-claims.json", claims); err != nil {
		t.Fatalf("failed to write claims: %v", err)
	}

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, dispatch := range normalizedDispatches {
		if err := spawnTree.RecordSpawn("Queen", dispatch.Caste, dispatch.Name, dispatch.Task, 1); err != nil {
			t.Fatalf("failed to seed spawn tree: %v", err)
		}
	}
}

func verificationStepsByName(t *testing.T, verification map[string]interface{}) map[string]map[string]interface{} {
	t.Helper()

	rawSteps, ok := verification["steps"].([]interface{})
	if !ok {
		t.Fatalf("verification steps missing: %#v", verification)
	}

	steps := make(map[string]map[string]interface{}, len(rawSteps))
	for _, raw := range rawSteps {
		step, ok := raw.(map[string]interface{})
		if !ok {
			t.Fatalf("unexpected verification step: %#v", raw)
		}
		name, _ := step["name"].(string)
		steps[name] = step
	}
	return steps
}

func seedBlockedContinueReport(t *testing.T, dataDir string, phase int, generatedAt time.Time, summary, next string, recovery codexContinueRecoveryPlan) {
	t.Helper()

	reportPath := filepath.Join(dataDir, "build", fmt.Sprintf("phase-%d", phase), "continue.json")
	if err := os.MkdirAll(filepath.Dir(reportPath), 0755); err != nil {
		t.Fatalf("mkdir continue report dir: %v", err)
	}
	report := codexContinueReport{
		Phase:       phase,
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339),
		Summary:     summary,
		Recovery:    recovery,
		Advanced:    false,
		Completed:   false,
		Next:        next,
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal continue report: %v", err)
	}
	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		t.Fatalf("write continue report: %v", err)
	}
}

func TestVerifyCodexBuildClaims_SimulatedMode_AllCompleted_Fails(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Write empty claims (simulated mode -- FakeInvoker produces empty arrays)
	claims := codexBuildClaims{BuildPhase: 1, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	if err := store.SaveJSON("last-build-claims.json", claims); err != nil {
		t.Fatalf("failed to write claims: %v", err)
	}

	manifest := codexContinueManifest{
		Present: true,
		Path:    "build/phase-1/manifest.json",
		Data: codexBuildManifest{
			Phase:        1,
			DispatchMode: "simulated",
			ClaimsPath:   displayDataPath("last-build-claims.json"),
			Dispatches: []codexBuildDispatch{
				{Stage: "wave", Caste: "builder", Name: "Forge-1", Task: "Build it", Status: "completed"},
				{Stage: "verification", Caste: "watcher", Name: "Keen-1", Task: "Verify it", Status: "completed"},
			},
		},
	}

	result := verifyCodexBuildClaims(tmpDir, manifest)
	if result.Passed {
		t.Fatalf("expected Passed=false for simulated mode, got Passed=true: %s", result.Summary)
	}
	if !strings.Contains(result.Summary, "rerun `aether build <phase>` without `--synthetic`") {
		t.Fatalf("expected summary to recommend rerunning without --synthetic, got: %s", result.Summary)
	}
}

func TestVerifyCodexBuildClaims_IncompleteDispatches_StillFails(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Write empty claims
	claims := codexBuildClaims{BuildPhase: 1, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	if err := store.SaveJSON("last-build-claims.json", claims); err != nil {
		t.Fatalf("failed to write claims: %v", err)
	}

	manifest := codexContinueManifest{
		Present: true,
		Path:    "build/phase-1/manifest.json",
		Data: codexBuildManifest{
			Phase:        1,
			DispatchMode: "real",
			ClaimsPath:   displayDataPath("last-build-claims.json"),
			Dispatches: []codexBuildDispatch{
				{Stage: "wave", Caste: "builder", Name: "Forge-1", Task: "Build it", Status: "failed"},
				{Stage: "verification", Caste: "watcher", Name: "Keen-1", Task: "Verify it", Status: "completed"},
			},
		},
	}

	result := verifyCodexBuildClaims(tmpDir, manifest)
	if result.Passed {
		t.Fatalf("expected Passed=false when dispatches are incomplete, got Passed=true: %s", result.Summary)
	}
}

func TestVerifyCodexBuildClaims_RealMode_EmptyClaimsFail(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	claims := codexBuildClaims{BuildPhase: 1, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	if err := store.SaveJSON("last-build-claims.json", claims); err != nil {
		t.Fatalf("failed to write claims: %v", err)
	}

	manifest := codexContinueManifest{
		Present: true,
		Path:    "build/phase-1/manifest.json",
		Data: codexBuildManifest{
			Phase:        1,
			DispatchMode: "real",
			ClaimsPath:   displayDataPath("last-build-claims.json"),
			Dispatches: []codexBuildDispatch{
				{Stage: "wave", Caste: "builder", Name: "Forge-1", Task: "Build it", Status: "completed"},
				{Stage: "verification", Caste: "watcher", Name: "Keen-1", Task: "Verify it", Status: "completed"},
			},
		},
	}

	result := verifyCodexBuildClaims(tmpDir, manifest)
	if result.Passed {
		t.Fatalf("expected Passed=false for real mode empty claims, got Passed=true: %s", result.Summary)
	}
	if !strings.Contains(result.Summary, "real mode") {
		t.Fatalf("expected summary to mention real mode, got: %s", result.Summary)
	}
}

func TestAssessCodexContinue_SimulatedDispatchDoesNotCountAsEvidence(t *testing.T) {
	phase := colony.Phase{
		ID:     1,
		Name:   "Synthetic phase",
		Status: colony.PhaseInProgress,
		Tasks: []colony.Task{
			{Goal: "Implement the feature"},
		},
	}

	manifest := codexContinueManifest{
		Present: true,
		Path:    "build/phase-1/manifest.json",
		Data: codexBuildManifest{
			Phase:        1,
			DispatchMode: "simulated",
			Dispatches: []codexBuildDispatch{
				{Stage: "wave", Caste: "builder", Name: "Forge-1", Task: "Implement the feature", Status: "completed", TaskID: "task-1"},
			},
		},
	}

	verification := codexContinueVerificationReport{
		ChecksPassed: true,
		Passed:       true,
		Claims: codexClaimVerification{
			Present: true,
			Passed:  false,
			Summary: "builder claims file is empty because the build ran in simulated mode",
		},
	}

	assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{}, time.Now().UTC())
	if assessment.Passed {
		t.Fatalf("expected simulated dispatch assessment to fail, got passed=true: %+v", assessment)
	}
	if assessment.PositiveEvidence {
		t.Fatalf("expected simulated dispatch not to count as positive evidence: %+v", assessment)
	}
	if len(assessment.RedispatchTasks) != 1 || assessment.RedispatchTasks[0] != "task-1" {
		t.Fatalf("expected simulated task to require redispatch, got %+v", assessment.RedispatchTasks)
	}
	if len(assessment.Tasks) != 1 || assessment.Tasks[0].Outcome != "simulated" {
		t.Fatalf("expected simulated task outcome, got %+v", assessment.Tasks)
	}
}

type continueWatcherTestInvoker struct {
	fake           codex.FakeInvoker
	watcherStatus  string
	watcherSummary string
	watcherCalls   int
	watcherName    string
}

func (f *continueWatcherTestInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	if config.Caste == "watcher" {
		f.watcherCalls++
		f.watcherName = config.WorkerName
		status := strings.TrimSpace(f.watcherStatus)
		if status == "" {
			status = "completed"
		}
		summary := strings.TrimSpace(f.watcherSummary)
		if summary == "" {
			summary = "Continue watcher completed"
		}
		result := codex.WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     status,
			Summary:    summary,
		}
		if status != "completed" {
			result.Blockers = []string{summary}
		}
		return result, nil
	}
	return f.fake.Invoke(ctx, config)
}

func (f *continueWatcherTestInvoker) IsAvailable(ctx context.Context) bool { return true }

func (f *continueWatcherTestInvoker) ValidateAgent(path string) error { return nil }

type continueUnavailableInvoker struct{}

func (i *continueUnavailableInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	err := fmt.Errorf("no authenticated worker platform is available")
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "failed",
		Error:      err,
	}, err
}

func (i *continueUnavailableInvoker) IsAvailable(ctx context.Context) bool { return false }

func (i *continueUnavailableInvoker) ValidateAgent(path string) error { return nil }

// TestContinue_BlocksOnVerifiedPartial verifies that a phase with a
// non-completed worker does NOT advance even when verification steps pass.
// This proves the partial-success bypass is closed.
func TestContinue_BlocksOnVerifiedPartial(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Block advancement when any worker failed despite passing checks"
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
					ID:     1,
					Name:   "Partial completion phase",
					Status: colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Ship the first part", Status: colony.TaskInProgress},
						{ID: &taskTwoID, Goal: "Ship the second part", Status: colony.TaskInProgress},
					},
				},
				{
					ID:     2,
					Name:   "Next should stay blocked",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Waits for full completion", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Partial completion phase", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-301", Task: "Ship the first part", Status: "completed", TaskID: taskOneID},
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-302", Task: "Ship the second part", Status: "failed", TaskID: taskTwoID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-303", Task: "Independent verification before advancement", Status: "completed"},
	})

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true when a worker failed, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false when a worker failed, got %v", result)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("phase 1 status = %s, want in_progress", state.Plan.Phases[0].Status)
	}
}

// TestContinue_BlocksOnWatcherTimeout proves that a watcher timeout
// prevents phase advancement and sets checksPassed to false.
func TestContinue_BlocksOnWatcherTimeout(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Block advancement when continue watcher verification times out"
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
					Name:   "Watcher timeout integration test",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Ship after watcher completes", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Remains blocked",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Await watcher success", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Watcher timeout integration test", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-401", Task: "Ship after watcher completes", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-build-402", Task: "Build-time verification", Status: "completed"},
	})

	invoker := &continueWatcherTestInvoker{
		watcherStatus:  "timeout",
		watcherSummary: "Continue watcher timed out during independent verification",
	}
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true on watcher timeout, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false on watcher timeout, got %v", result)
	}

	verification := result["verification"].(map[string]interface{})
	if passed, _ := verification["checks_passed"].(bool); passed {
		t.Fatalf("expected checks_passed:false on watcher timeout, got %v", verification)
	}
	watcher := verification["watcher"].(map[string]interface{})
	if passed, _ := watcher["passed"].(bool); passed {
		t.Fatalf("expected watcher.passed:false on timeout, got %v", watcher)
	}
}

// TestContinue_ReconcileDoesNotBypassClaims proves that --reconcile-task
// does not advance the phase when the reconciled task has no git evidence
// to support the claim.
func TestContinue_ReconcileDoesNotBypassClaims(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Reconcile does not bypass claims verification"
	now := time.Now().UTC()
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
					Name:   "Reconcile bypass test",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Needs git evidence", Status: colony.TaskInProgress}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-501", Task: "Needs git evidence", Status: "failed", TaskID: taskID},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Reconcile bypass test", goal, dispatches)

	// Overwrite claims to be empty so there is no git evidence for the task
	if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
		BuildPhase: 1,
		Timestamp:  now.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("failed to write empty claims: %v", err)
	}

	rootCmd.SetArgs([]string{"continue", "--reconcile-task", taskID})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true when reconciled task lacks evidence, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false when reconciled task lacks evidence, got %v", result)
	}

	reconciled := stringSliceValue(result["reconciled_tasks"])
	if len(reconciled) != 1 || reconciled[0] != taskID {
		t.Fatalf("expected reconciled task %s, got %v", taskID, reconciled)
	}

	blockingIssues := stringSliceValue(result["blocking_issues"])
	hasReconcileWarning := false
	for _, issue := range blockingIssues {
		if strings.Contains(issue, "manually reconciled") {
			hasReconcileWarning = true
			break
		}
	}
	if !hasReconcileWarning {
		t.Fatalf("expected blocking issues to mention reconcile, got %v", blockingIssues)
	}
}

func withTestWorkspace(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main_test.go: %v", err)
	}
}

func withWorkingDir(t *testing.T, root string) {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
}
