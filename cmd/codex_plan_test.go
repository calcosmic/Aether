package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/codex"
)

func TestPlanUsesSurveyAndRecordsPlanningDispatches(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n\nrequire github.com/spf13/cobra v1.9.0\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "main_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main_test.go: %v", err)
	}

	goal := "Bring Codex core colony commands to true ant-process parity"
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

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"plan"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse plan output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got %v", envelope)
	}
	result := envelope["result"].(map[string]interface{})
	if existing, _ := result["existing_plan"].(bool); existing {
		t.Fatal("expected a fresh generated plan, not existing_plan:true")
	}
	if count := int(result["count"].(float64)); count < 4 {
		t.Fatalf("expected a grounded multi-phase plan, got %d phases", count)
	}
	dispatches := result["dispatches"].([]interface{})
	if len(dispatches) != 2 {
		t.Fatalf("expected 2 planning dispatches, got %d", len(dispatches))
	}
	planningFiles := result["planning_files"].([]interface{})
	if len(planningFiles) != 2 {
		t.Fatalf("expected 2 planning files, got %d", len(planningFiles))
	}
	phaseResearchFiles := result["phase_research_files"].([]interface{})
	if len(phaseResearchFiles) != int(result["count"].(float64)) {
		t.Fatalf("expected phase research files to match phase count, got %d", len(phaseResearchFiles))
	}

	for _, name := range []string{"SCOUT.md", "ROUTE-SETTER.md"} {
		if _, err := os.Stat(filepath.Join(dataDir, "planning", name)); err != nil {
			t.Fatalf("expected planning artifact %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dataDir, "phase-research", "phase-1-research.md")); err != nil {
		t.Fatalf("expected phase research file: %v", err)
	}

	spawnTreeData, err := os.ReadFile(filepath.Join(dataDir, "spawn-tree.txt"))
	if err != nil {
		t.Fatalf("expected spawn-tree.txt: %v", err)
	}
	if count := strings.Count(string(spawnTreeData), "|Queen|scout|"); count != 1 {
		t.Fatalf("expected 1 scout spawn entry, got %d\n%s", count, string(spawnTreeData))
	}
	if count := strings.Count(string(spawnTreeData), "|Queen|route_setter|"); count != 1 {
		t.Fatalf("expected 1 route_setter spawn entry, got %d\n%s", count, string(spawnTreeData))
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if state.Plan.GeneratedAt == nil {
		t.Fatal("expected GeneratedAt to be set")
	}
	if state.Plan.Confidence == nil || *state.Plan.Confidence <= 0 {
		t.Fatal("expected plan confidence to be set")
	}
	if len(state.Plan.Phases) == 0 || state.Plan.Phases[0].Status != colony.PhaseReady {
		t.Fatalf("expected first phase to be ready, got %+v", state.Plan.Phases)
	}
	if len(state.Events) == 0 || !strings.Contains(state.Events[len(state.Events)-1], "plan_generated|plan") {
		t.Fatalf("expected plan_generated event, got %v", state.Events)
	}
}

func TestPlanReturnsExistingPlanWithoutRefresh(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Reuse the current plan"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Existing phase",
					Description: "Already planned",
					Status:      colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &taskID, Goal: "Use the existing plan", Status: colony.TaskPending},
					},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"plan"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse plan output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	result := envelope["result"].(map[string]interface{})
	if existing, _ := result["existing_plan"].(bool); !existing {
		t.Fatalf("expected existing_plan:true, got %v", result)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "spawn-tree.txt")); err == nil {
		t.Fatal("expected no new planning spawns when reusing existing plan")
	}
}

func TestPlanRefreshRejectsActivePhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Refresh while active"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Active phase",
					Description: "Already executing",
					Status:      colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskID, Goal: "Do the work", Status: colony.TaskInProgress},
					},
				},
			},
		},
	})

	var errBuf bytes.Buffer
	stderr = &errBuf

	rootCmd.SetArgs([]string{"plan", "--refresh"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	if !strings.Contains(errBuf.String(), "cannot refresh the plan while phase 1 is already active") {
		t.Fatalf("expected refresh rejection, got: %s", errBuf.String())
	}
}

// --- dispatchRealPlanningWorkers tests ---

func TestDispatchRealPlanningWorkers_NilInvoker_ReturnsNil(t *testing.T) {
	result, err := dispatchRealPlanningWorkers(context.Background(), "/tmp/test-repo", nil)
	if err != nil {
		t.Fatalf("expected nil error for nil invoker, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for nil invoker, got: %+v", result)
	}
}

func TestDispatchRealPlanningWorkers_UnavailableInvoker_ReturnsNil(t *testing.T) {
	// Use a custom invoker that reports unavailable (separate type to avoid redeclaration).
	unavailable := &planTestUnavailableInvoker{}
	result, err := dispatchRealPlanningWorkers(context.Background(), "/tmp/test-repo", unavailable)
	if err != nil {
		t.Fatalf("expected nil error for unavailable invoker, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for unavailable invoker, got: %+v", result)
	}
}

func TestDispatchRealPlanningWorkers_AvailableInvoker_ReturnsDispatches(t *testing.T) {
	tmpDir := t.TempDir()
	codexAgentsDir := filepath.Join(tmpDir, ".codex", "agents")
	if err := os.MkdirAll(codexAgentsDir, 0755); err != nil {
		t.Fatalf("failed to create .codex/agents: %v", err)
	}
	for _, name := range []string{"aether-scout.toml", "aether-route-setter.toml"} {
		if err := os.WriteFile(filepath.Join(codexAgentsDir, name), []byte(`name = "test"
description = "test agent"
developer_instructions = "test instructions"`), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	invoker := &codex.FakeInvoker{}
	result, err := dispatchRealPlanningWorkers(context.Background(), tmpDir, invoker)
	if err != nil {
		t.Fatalf("expected nil error for available invoker, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for available invoker")
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 dispatches, got %d", len(result))
	}
	if result[0].Caste != "scout" {
		t.Fatalf("expected first dispatch caste 'scout', got %q", result[0].Caste)
	}
	if result[1].Caste != "route_setter" {
		t.Fatalf("expected second dispatch caste 'route_setter', got %q", result[1].Caste)
	}
	if result[0].Status != "completed" {
		t.Fatalf("expected first dispatch status 'completed', got %q", result[0].Status)
	}
	if result[1].Status != "completed" {
		t.Fatalf("expected second dispatch status 'completed', got %q", result[1].Status)
	}
}

// planTestUnavailableInvoker is a WorkerInvoker that always reports unavailable.
type planTestUnavailableInvoker struct{}

func (u *planTestUnavailableInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{}, nil
}

func (u *planTestUnavailableInvoker) IsAvailable(ctx context.Context) bool {
	return false
}

func (u *planTestUnavailableInvoker) ValidateAgent(path string) error {
	return nil
}

func TestDispatchRealPlanningWorkers_CancelledContext_ReturnsResults(t *testing.T) {
	tmpDir := t.TempDir()
	codexAgentsDir := filepath.Join(tmpDir, ".codex", "agents")
	if err := os.MkdirAll(codexAgentsDir, 0755); err != nil {
		t.Fatalf("failed to create .codex/agents: %v", err)
	}
	for _, name := range []string{"aether-scout.toml", "aether-route-setter.toml"} {
		if err := os.WriteFile(filepath.Join(codexAgentsDir, name), []byte(`name = "test"
description = "test agent"
developer_instructions = "test instructions"`), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	invoker := &codex.FakeInvoker{}
	result, err := dispatchRealPlanningWorkers(ctx, tmpDir, invoker)
	if err != nil {
		t.Fatalf("expected nil error for cancelled context, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for cancelled context (workers should still return results)")
	}
}
