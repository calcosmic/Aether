package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
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

	contextData, err := os.ReadFile(filepath.Join(root, ".aether", "CONTEXT.md"))
	if err != nil {
		t.Fatalf("expected CONTEXT.md: %v", err)
	}
	if !strings.Contains(string(contextData), "aether build 1") {
		t.Fatalf("expected CONTEXT.md to point at the first build, got:\n%s", string(contextData))
	}

	handoffData, err := os.ReadFile(filepath.Join(root, ".aether", "HANDOFF.md"))
	if err != nil {
		t.Fatalf("expected HANDOFF.md: %v", err)
	}
	if !strings.Contains(string(handoffData), goal) {
		t.Fatalf("expected HANDOFF.md to include the goal, got:\n%s", string(handoffData))
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

func TestPlanIncludesClarificationWarningWhenPendingClarificationsExist(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Reuse the current plan carefully"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Existing phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Use the existing plan", Status: colony.TaskPending}},
				},
			},
		},
	})
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{
		Decisions: []PendingDecision{{
			ID:          "pd_clarify",
			Type:        clarificationDecisionType,
			Description: "Which verification bar do you want?",
			Source:      discussSource("verification", false),
			Resolved:    false,
			CreatedAt:   "2026-04-19T10:00:00Z",
		}},
	}); err != nil {
		t.Fatalf("seed pending decisions: %v", err)
	}

	rootCmd.SetArgs([]string{"plan"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse plan output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	result := envelope["result"].(map[string]interface{})
	if got := int(result["unresolved_clarifications"].(float64)); got != 1 {
		t.Fatalf("unresolved_clarifications = %d, want 1", got)
	}
	if warning := stringValue(result["clarification_warning"]); !strings.Contains(warning, "Unresolved clarifications exist") {
		t.Fatalf("expected clarification warning in plan result, got %q", warning)
	}
}

func TestPlanUsesWorkerWrittenArtifactsWhenProvided(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	goal := "Ground the plan in worker artifacts"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &planningArtifactInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"plan"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["dispatch_mode"]; got != "real" {
		t.Fatalf("dispatch_mode = %v, want real", got)
	}
	if got := result["artifact_source"]; got != "worker-written" {
		t.Fatalf("artifact_source = %v, want worker-written", got)
	}
	if got := result["plan_source"]; got != "worker-artifact" {
		t.Fatalf("plan_source = %v, want worker-artifact", got)
	}

	phases := result["phases"].([]interface{})
	firstPhase := phases[0].(map[string]interface{})
	if firstPhase["name"] != "Worker planned phase" {
		t.Fatalf("first phase name = %v, want Worker planned phase", firstPhase["name"])
	}

	for _, check := range []struct {
		path string
		want string
	}{
		{filepath.Join(dataDir, "planning", "SCOUT.md"), "worker-authored scout"},
		{filepath.Join(dataDir, "planning", "ROUTE-SETTER.md"), "worker-authored route-setter"},
		{filepath.Join(dataDir, "phase-research", "phase-1-research.md"), "worker-authored phase research"},
	} {
		data, err := os.ReadFile(check.path)
		if err != nil {
			t.Fatalf("read %s: %v", filepath.Base(check.path), err)
		}
		if !strings.Contains(string(data), check.want) {
			t.Fatalf("expected %s to be preserved, got:\n%s", filepath.Base(check.path), string(data))
		}
	}
}

func TestPlanFallsBackWhenRealPlanningDispatchFails(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	goal := "Fall back when planner workers stall"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &failingPlanningInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"plan"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["dispatch_mode"]; got != "fallback" {
		t.Fatalf("dispatch_mode = %v, want fallback", got)
	}
	if got := strings.TrimSpace(result["planning_warning"].(string)); got == "" {
		t.Fatal("expected planning_warning to be populated")
	}
	dispatches := result["dispatches"].([]interface{})
	if len(dispatches) != 2 {
		t.Fatalf("expected 2 planning dispatches, got %d", len(dispatches))
	}
	first := dispatches[0].(map[string]interface{})
	if first["status"] != "timeout" {
		t.Fatalf("first dispatch status = %v, want timeout", first["status"])
	}
	if count := int(result["count"].(float64)); count == 0 {
		t.Fatal("expected fallback plan to still contain phases")
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

type planningArtifactInvoker struct{}

func (p *planningArtifactInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	result := codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    "worker-authored planning artifact",
	}
	switch config.Caste {
	case "scout":
		target := filepath.Join(config.Root, ".aether", "data", "planning", "SCOUT.md")
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return codex.WorkerResult{}, err
		}
		if err := os.WriteFile(target, []byte("# worker-authored scout\n"), 0644); err != nil {
			return codex.WorkerResult{}, err
		}
		result.FilesCreated = []string{filepath.ToSlash(filepath.Join(".aether", "data", "planning", "SCOUT.md"))}
	case "route_setter":
		planningDir := filepath.Join(config.Root, ".aether", "data", "planning")
		if err := os.MkdirAll(planningDir, 0755); err != nil {
			return codex.WorkerResult{}, err
		}
		if err := os.WriteFile(filepath.Join(planningDir, "ROUTE-SETTER.md"), []byte("# worker-authored route-setter\n"), 0644); err != nil {
			return codex.WorkerResult{}, err
		}
		planArtifact := `{
  "phases": [
    {
      "name": "Worker planned phase",
      "description": "Phase loaded from route-setter artifact.",
      "tasks": [
        {
          "goal": "Land the worker-authored planning flow",
          "constraints": ["Keep plan artifact authoritative"],
          "hints": ["cmd/codex_plan.go"],
          "success_criteria": ["The worker plan is applied"],
          "depends_on": []
        }
      ],
      "success_criteria": ["Worker route-setter plan used"]
    },
    {
      "name": "Verification",
      "description": "Verify the worker-authored plan.",
      "tasks": [
        {
          "goal": "Confirm the worker plan survives serialization",
          "constraints": [],
          "hints": ["cmd/codex_plan_test.go"],
          "success_criteria": ["Regression coverage exists"],
          "depends_on": ["1.1"]
        }
      ],
      "success_criteria": ["Plan verification ready"]
    }
  ],
  "confidence": {
    "knowledge": 91,
    "requirements": 88,
    "risks": 84,
    "dependencies": 79,
    "effort": 86,
    "overall": 86
  },
  "gaps": ["Worker identified one remaining follow-up."]
}`
		if err := os.WriteFile(filepath.Join(planningDir, "phase-plan.json"), []byte(planArtifact), 0644); err != nil {
			return codex.WorkerResult{}, err
		}

		researchDir := filepath.Join(config.Root, ".aether", "data", "phase-research")
		if err := os.MkdirAll(researchDir, 0755); err != nil {
			return codex.WorkerResult{}, err
		}
		if err := os.WriteFile(filepath.Join(researchDir, "phase-1-research.md"), []byte("# worker-authored phase research\n"), 0644); err != nil {
			return codex.WorkerResult{}, err
		}

		result.FilesCreated = []string{
			filepath.ToSlash(filepath.Join(".aether", "data", "planning", "ROUTE-SETTER.md")),
			filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json")),
			filepath.ToSlash(filepath.Join(".aether", "data", "phase-research", "phase-1-research.md")),
		}
	}
	return result, nil
}

func (p *planningArtifactInvoker) IsAvailable(_ context.Context) bool {
	return true
}

func (p *planningArtifactInvoker) ValidateAgent(_ string) error {
	return nil
}

type failingPlanningInvoker struct{}

func (f *failingPlanningInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "timeout",
		Error:      context.DeadlineExceeded,
	}, nil
}

func (f *failingPlanningInvoker) IsAvailable(_ context.Context) bool {
	return true
}

func (f *failingPlanningInvoker) ValidateAgent(_ string) error {
	return nil
}

func TestDispatchRealPlanningWorkers_CancelledContext_ReturnsTimeoutError(t *testing.T) {
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
	if err == nil {
		t.Fatal("expected timeout error for cancelled context")
	}
	if result == nil {
		t.Fatal("expected non-nil result for cancelled context")
	}
}
