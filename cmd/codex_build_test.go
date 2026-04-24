package cmd

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestBuildWritesDispatchArtifactsAndUpdatesState(t *testing.T) {
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

	goal := "Bring Codex build parity to the ant process"
	researchID := "1.1"
	implementID := "1.2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Build parity",
					Description: "Replace fake build dispatch with real artifacts and spawn records",
					Status:      colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &researchID, Goal: "Research the missing build orchestration gaps", Status: colony.TaskPending},
						{ID: &implementID, Goal: "Implement the Go-native build packet", Status: colony.TaskPending, DependsOn: []string{researchID}},
					},
					SuccessCriteria: []string{"Build artifacts exist", "Spawn tree reflects the worker packet"},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse build output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got %v", envelope)
	}

	result := envelope["result"].(map[string]interface{})
	if got := int(result["dispatch_count"].(float64)); got != 7 {
		t.Fatalf("dispatch_count = %d, want 7", got)
	}
	if got := int(result["wave_count"].(float64)); got != 2 {
		t.Fatalf("wave_count = %d, want 2", got)
	}
	if got := int(result["parallel_waves"].(float64)); got != 0 {
		t.Fatalf("parallel_waves = %d, want 0", got)
	}
	if next := result["next"].(string); next != "aether continue" {
		t.Fatalf("next = %q, want aether continue", next)
	}
	if waveExecution, ok := result["wave_execution"].([]interface{}); !ok || len(waveExecution) != 2 {
		t.Fatalf("wave_execution = %#v, want 2 wave plans", result["wave_execution"])
	}

	for _, rel := range []string{
		"checkpoints/pre-build-phase-1.json",
		"build/phase-1/manifest.json",
		"last-build-claims.json",
	} {
		if _, err := os.Stat(filepath.Join(dataDir, rel)); err != nil {
			t.Fatalf("expected artifact %s: %v", rel, err)
		}
	}

	var manifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &manifest); err != nil {
		t.Fatalf("failed to load build manifest: %v", err)
	}
	if manifest.Phase != 1 || manifest.PhaseName != "Build parity" {
		t.Fatalf("unexpected manifest header: %+v", manifest)
	}
	if manifest.DispatchMode != "simulated" {
		t.Fatalf("dispatch mode = %q, want simulated", manifest.DispatchMode)
	}
	if len(manifest.Dispatches) != 7 {
		t.Fatalf("expected 7 manifest dispatches, got %d", len(manifest.Dispatches))
	}
	if len(manifest.WorkerBriefs) != 7 {
		t.Fatalf("expected 7 worker briefs in manifest, got %d", len(manifest.WorkerBriefs))
	}
	if len(manifest.Tasks) != 2 {
		t.Fatalf("expected 2 planned tasks, got %d", len(manifest.Tasks))
	}
	if len(manifest.WaveExecution) != 2 {
		t.Fatalf("expected 2 manifest wave execution plans, got %d", len(manifest.WaveExecution))
	}
	for _, plan := range manifest.WaveExecution {
		if plan.Strategy != "serial" {
			t.Fatalf("manifest wave %d strategy = %q, want serial", plan.Wave, plan.Strategy)
		}
	}
	for _, brief := range manifest.WorkerBriefs {
		rel := strings.TrimPrefix(brief, ".aether/data/")
		if _, err := os.Stat(filepath.Join(dataDir, rel)); err != nil {
			t.Fatalf("expected worker brief %s: %v", brief, err)
		}
	}

	var claims codexBuildClaims
	if err := store.LoadJSON("last-build-claims.json", &claims); err != nil {
		t.Fatalf("failed to load last-build-claims.json: %v", err)
	}
	if claims.BuildPhase != 1 {
		t.Fatalf("claims build phase = %d, want 1", claims.BuildPhase)
	}
	if len(claims.FilesCreated) != 0 || len(claims.FilesModified) != 0 {
		t.Fatalf("expected empty claims for pre-execution packet, got %+v", claims)
	}

	spawnTreeData, err := os.ReadFile(filepath.Join(dataDir, "spawn-tree.txt"))
	if err != nil {
		t.Fatalf("expected spawn-tree.txt: %v", err)
	}
	for _, want := range []string{"|Queen|builder|", "|Queen|oracle|", "|Queen|architect|", "|Queen|watcher|", "|Queen|chaos|", "|Queen|archaeologist|"} {
		if !strings.Contains(string(spawnTreeData), want) {
			t.Fatalf("spawn tree missing %q\n%s", want, string(spawnTreeData))
		}
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.CurrentPhase != 1 {
		t.Fatalf("current_phase = %d, want 1", state.CurrentPhase)
	}
	if state.BuildStartedAt == nil {
		t.Fatal("expected BuildStartedAt to be set")
	}
	if state.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("phase status = %s, want in_progress", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[0].Tasks[0].Status != colony.TaskInProgress {
		t.Fatalf("task 1 status = %s, want in_progress", state.Plan.Phases[0].Tasks[0].Status)
	}
	if state.Plan.Phases[0].Tasks[1].Status != colony.TaskPending {
		t.Fatalf("task 2 status = %s, want pending", state.Plan.Phases[0].Tasks[1].Status)
	}
	if len(state.Events) < 2 || !strings.Contains(strings.Join(state.Events[len(state.Events)-2:], "\n"), "build_dispatched|build") {
		t.Fatalf("expected build_dispatched event, got %v", state.Events)
	}

	contextData, err := os.ReadFile(filepath.Join(root, ".aether", "CONTEXT.md"))
	if err != nil {
		t.Fatalf("expected CONTEXT.md: %v", err)
	}
	if !strings.Contains(string(contextData), "aether continue") {
		t.Fatalf("expected CONTEXT.md to point at continue, got:\n%s", string(contextData))
	}

	handoffData, err := os.ReadFile(filepath.Join(root, ".aether", "HANDOFF.md"))
	if err != nil {
		t.Fatalf("expected HANDOFF.md: %v", err)
	}
	if !strings.Contains(string(handoffData), "Phase 1 dispatched") {
		t.Fatalf("expected HANDOFF.md to summarize build progress, got:\n%s", string(handoffData))
	}
}

func TestBuildPlanOnlyPrintsDispatchManifestWithoutMutatingState(t *testing.T) {
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

	goal := "Expose wrapper-spawn build plans"
	taskOneID := "1.1"
	taskTwoID := "1.2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Wrapper bridge",
					Description: "Let Claude and OpenCode spawn workers from a runtime manifest",
					Status:      colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Define the structured build manifest", Status: colony.TaskPending},
						{ID: &taskTwoID, Goal: "Use the manifest in wrappers", Status: colony.TaskPending, DependsOn: []string{taskOneID}},
					},
					SuccessCriteria: []string{"Wrappers do not parse visual output"},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"build", "1", "--plan-only"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build --plan-only returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse plan-only output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got %v", envelope)
	}
	result := envelope["result"].(map[string]interface{})
	if result["plan_only"] != true {
		t.Fatalf("plan_only = %v, want true", result["plan_only"])
	}
	if got := result["dispatch_mode"].(string); got != "plan-only" {
		t.Fatalf("dispatch_mode = %q, want plan-only", got)
	}
	if got := int(result["dispatch_count"].(float64)); got != 7 {
		t.Fatalf("dispatch_count = %d, want 7", got)
	}
	dispatches := result["dispatches"].([]interface{})
	if len(dispatches) != 7 {
		t.Fatalf("dispatches = %d, want 7", len(dispatches))
	}
	for _, raw := range dispatches {
		dispatch := raw.(map[string]interface{})
		if dispatch["status"].(string) != "planned" {
			t.Fatalf("dispatch status = %q, want planned", dispatch["status"])
		}
		if strings.TrimSpace(dispatch["agent_name"].(string)) == "" {
			t.Fatalf("dispatch missing agent_name: %+v", dispatch)
		}
	}

	manifest := result["dispatch_manifest"].(map[string]interface{})
	if manifest["plan_only"] != true {
		t.Fatalf("manifest plan_only = %v, want true", manifest["plan_only"])
	}
	if manifest["dispatch_mode"].(string) != "plan-only" {
		t.Fatalf("manifest dispatch_mode = %q, want plan-only", manifest["dispatch_mode"])
	}
	if manifest["checkpoint"].(string) != "" || manifest["claims_path"].(string) != "" {
		t.Fatalf("plan-only manifest should not claim artifact paths: %+v", manifest)
	}
	if workerBriefs := manifest["worker_briefs"].([]interface{}); len(workerBriefs) != 0 {
		t.Fatalf("plan-only manifest should not write worker briefs, got %v", workerBriefs)
	}

	for _, rel := range []string{
		"checkpoints/pre-build-phase-1.json",
		"build/phase-1/manifest.json",
		"last-build-claims.json",
	} {
		if _, err := os.Stat(filepath.Join(dataDir, rel)); !os.IsNotExist(err) {
			t.Fatalf("plan-only unexpectedly wrote %s (err=%v)", rel, err)
		}
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY", state.State)
	}
	if state.CurrentPhase != 0 {
		t.Fatalf("current_phase = %d, want 0", state.CurrentPhase)
	}
	if state.BuildStartedAt != nil {
		t.Fatal("BuildStartedAt should remain nil")
	}
	if state.Plan.Phases[0].Status != colony.PhaseReady {
		t.Fatalf("phase status = %s, want ready", state.Plan.Phases[0].Status)
	}
}

func TestBuildWaveExecutionPlansRespectParallelMode(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-1", Task: "Task one"},
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-2", Task: "Task two"},
		{Stage: "wave", Wave: 2, Caste: "builder", Name: "Forge-3", Task: "Task three"},
	}

	inRepo := buildWaveExecutionPlans(dispatches, colony.ModeInRepo)
	if len(inRepo) != 2 {
		t.Fatalf("in-repo wave plans = %d, want 2", len(inRepo))
	}
	if inRepo[0].Strategy != "serial" {
		t.Fatalf("wave 1 strategy = %q, want serial", inRepo[0].Strategy)
	}
	if !strings.Contains(inRepo[0].Reason, "main working tree") {
		t.Fatalf("wave 1 reason = %q, want shared working tree guidance", inRepo[0].Reason)
	}
	if inRepo[1].Strategy != "serial" || inRepo[1].WorkerCount != 1 {
		t.Fatalf("wave 2 plan = %+v, want single-task serial", inRepo[1])
	}

	worktree := buildWaveExecutionPlans(dispatches, colony.ModeWorktree)
	if len(worktree) != 2 {
		t.Fatalf("worktree wave plans = %d, want 2", len(worktree))
	}
	if worktree[0].Strategy != "parallel" {
		t.Fatalf("worktree wave 1 strategy = %q, want parallel", worktree[0].Strategy)
	}
	if !strings.Contains(worktree[0].Reason, "isolated worktrees") {
		t.Fatalf("worktree wave 1 reason = %q, want isolated worktree guidance", worktree[0].Reason)
	}
}

func TestBuildSupportsTaskScopedRedispatch(t *testing.T) {
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

	goal := "Redispatch only the missing task"
	taskOneID := "1.1"
	taskTwoID := "1.2"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		ColonyDepth:    "full",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Targeted redispatch",
					Status: colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Keep the completed task closed", Status: colony.TaskCompleted},
						{ID: &taskTwoID, Goal: "Redispatch only the missing task", Status: colony.TaskInProgress, DependsOn: []string{taskOneID}},
					},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"build", "1", "--task", taskTwoID})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse build output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	result := envelope["result"].(map[string]interface{})
	selectedTasks := result["selected_tasks"].([]interface{})
	if len(selectedTasks) != 1 || selectedTasks[0].(string) != taskTwoID {
		t.Fatalf("selected_tasks = %v, want [%s]", selectedTasks, taskTwoID)
	}

	var manifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &manifest); err != nil {
		t.Fatalf("failed to load build manifest: %v", err)
	}
	if len(manifest.SelectedTasks) != 1 || manifest.SelectedTasks[0] != taskTwoID {
		t.Fatalf("manifest selected tasks = %v, want [%s]", manifest.SelectedTasks, taskTwoID)
	}
	if len(manifest.Dispatches) != 4 {
		t.Fatalf("expected 4 manifest dispatches for targeted redispatch, got %d", len(manifest.Dispatches))
	}
	for _, dispatch := range manifest.Dispatches {
		if dispatch.TaskID != "" && dispatch.TaskID != taskTwoID {
			t.Fatalf("unexpected task-scoped dispatch %+v", dispatch)
		}
		if dispatch.Stage == "strategy" {
			t.Fatalf("unexpected strategy dispatch during targeted redispatch: %+v", dispatch)
		}
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if state.Plan.Phases[0].Tasks[0].Status != colony.TaskCompleted {
		t.Fatalf("task 1 status = %s, want completed", state.Plan.Phases[0].Tasks[0].Status)
	}
	if state.Plan.Phases[0].Tasks[1].Status != colony.TaskInProgress {
		t.Fatalf("task 2 status = %s, want in_progress", state.Plan.Phases[0].Tasks[1].Status)
	}
}

func TestBuildRecoversMissingPlanFromPersistedPlanningArtifact(t *testing.T) {
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

	goal := "Recover build after the saved plan vanished"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		ColonyDepth:  "full",
		Plan:         colony.Plan{Phases: []colony.Phase{}},
		Events: []string{
			"2026-04-21T07:50:00Z phase-1-complete: Audit complete.",
			"2026-04-21T08:20:00Z phase-2-complete: Standard designed.",
		},
	})
	if err := store.SaveJSON("planning/phase-plan.json", codexWorkerPlanArtifact{
		Confidence: codexPlanConfidence{Overall: 88},
		Phases: []codexWorkerPlanPhase{
			{Name: "Audit", Tasks: []codexWorkerPlanTask{{Goal: "Audit the existing notes"}}},
			{Name: "Design", Tasks: []codexWorkerPlanTask{{Goal: "Define the frontmatter standard"}}},
			{
				Name:        "Standardize core references",
				Description: "Apply the saved schema to the highest-value notes first.",
				Tasks: []codexWorkerPlanTask{
					{Goal: "Standardize pattern notes"},
					{Goal: "Standardize device specs"},
				},
				SuccessCriteria: []string{"Core notes share the same schema"},
			},
		},
	}); err != nil {
		t.Fatalf("failed to save planning artifact: %v", err)
	}

	result, err := runCodexBuild(root, 3, nil, false)
	if err != nil {
		t.Fatalf("runCodexBuild returned error: %v", err)
	}
	if next := result["next"].(string); next != "aether continue" {
		t.Fatalf("next = %q, want aether continue", next)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if len(state.Plan.Phases) != 3 {
		t.Fatalf("phase count = %d, want 3", len(state.Plan.Phases))
	}
	if state.CurrentPhase != 3 {
		t.Fatalf("current_phase = %d, want 3", state.CurrentPhase)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Fatalf("phase 1 status = %s, want completed", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[1].Status != colony.PhaseCompleted {
		t.Fatalf("phase 2 status = %s, want completed", state.Plan.Phases[1].Status)
	}
	if state.Plan.Phases[2].Status != colony.PhaseInProgress {
		t.Fatalf("phase 3 status = %s, want in_progress", state.Plan.Phases[2].Status)
	}
	if state.Plan.Phases[2].Tasks[0].Status != colony.TaskInProgress {
		t.Fatalf("phase 3 task 1 status = %s, want in_progress", state.Plan.Phases[2].Tasks[0].Status)
	}
	if state.Plan.Phases[2].Tasks[1].Status != colony.TaskInProgress {
		t.Fatalf("phase 3 task 2 status = %s, want in_progress", state.Plan.Phases[2].Tasks[1].Status)
	}
}

func TestBuildRejectsDifferentActivePhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Do not dispatch a different active phase"
	activeTaskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Already active",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &activeTaskID, Goal: "Finish the active work", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Not yet active",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Future work", Status: colony.TaskPending}},
				},
			},
		},
	})

	var errBuf bytes.Buffer
	stderr = &errBuf

	rootCmd.SetArgs([]string{"build", "2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	if !strings.Contains(errBuf.String(), "phase 1 is already active") {
		t.Fatalf("expected active-phase rejection, got: %s", errBuf.String())
	}
}

func TestBuildAllocatesUniqueNamesWhenSpawnHistoryCollides(t *testing.T) {
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

	goal := "Avoid spawn tree collisions"
	taskID := "1.1"
	phase := colony.Phase{
		ID:          1,
		Name:        "Collision handling",
		Description: "Ensure new build workers do not reuse old spawn names",
		Status:      colony.PhaseReady,
		Tasks: []colony.Task{
			{ID: &taskID, Goal: "Implement collision-safe build dispatch names", Status: colony.TaskPending},
		},
	}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{phase}},
	})

	baseDispatches := plannedBuildDispatches(phase, "standard")
	if len(baseDispatches) == 0 {
		t.Fatal("expected planned dispatches")
	}

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", baseDispatches[0].Caste, baseDispatches[0].Name, "Old worker", 1); err != nil {
		t.Fatalf("failed to seed spawn tree: %v", err)
	}
	if err := spawnTree.UpdateStatus(baseDispatches[0].Name, "completed", "old run"); err != nil {
		t.Fatalf("failed to complete seeded spawn: %v", err)
	}

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	var manifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &manifest); err != nil {
		t.Fatalf("failed to load build manifest: %v", err)
	}

	if manifest.Dispatches[0].Name == baseDispatches[0].Name {
		t.Fatalf("expected collided worker name to be renamed, still got %q", manifest.Dispatches[0].Name)
	}
	if !strings.HasPrefix(manifest.Dispatches[0].Name, baseDispatches[0].Name+"-r") {
		t.Fatalf("expected retry-style suffix on renamed worker, got %q", manifest.Dispatches[0].Name)
	}
}

type buildFailInvoker struct{}

func (f *buildFailInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{}, context.DeadlineExceeded
}

func (f *buildFailInvoker) IsAvailable(ctx context.Context) bool { return false }

func (f *buildFailInvoker) ValidateAgent(path string) error { return nil }

func TestBuildRollsBackStateWhenDispatchFails(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to root: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Rollback failed build dispatches"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Rollback phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Try the failing build", Status: colony.TaskPending}},
				},
			},
		},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &buildFailInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	_, err = runCodexBuild(root, 1, nil, false)
	if err == nil {
		t.Fatal("expected build failure")
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY after rollback", state.State)
	}
	if state.CurrentPhase != 0 {
		t.Fatalf("current phase = %d, want 0 after rollback", state.CurrentPhase)
	}
	if state.BuildStartedAt != nil {
		t.Fatal("expected BuildStartedAt to be cleared by rollback")
	}
	if state.Plan.Phases[0].Status != colony.PhaseReady {
		t.Fatalf("phase status = %s, want ready after rollback", state.Plan.Phases[0].Status)
	}

	contextData, readErr := os.ReadFile(filepath.Join(root, ".aether", "CONTEXT.md"))
	if readErr != nil {
		t.Fatalf("expected CONTEXT.md after rollback: %v", readErr)
	}
	if !strings.Contains(string(contextData), "worker dispatcher is unavailable") {
		t.Fatalf("expected rollback context summary, got:\n%s", string(contextData))
	}
}

func TestBuildAllowsRetryWhenBuiltPhaseHasFailedDispatches(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to root: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Retry a poisoned built phase"
	taskID := "1.1"
	startedAt := mustParseRFC3339(t, "2026-04-17T12:00:00Z")
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		BuildStartedAt: func() *time.Time {
			ts := startedAt
			return &ts
		}(),
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Retry phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Recover the failed build", Status: colony.TaskInProgress}},
				},
			},
		},
	})

	if err := store.SaveJSON("build/phase-1/manifest.json", codexBuildManifest{
		Phase:        1,
		PhaseName:    "Retry phase",
		DispatchMode: "real",
		Dispatches: []codexBuildDispatch{
			{Name: "Brick-60", Caste: "builder", Status: "failed", Task: "Recover the failed build"},
			{Name: "Sentinel-29", Caste: "watcher", Status: "failed", Task: "Verify the failed build"},
		},
	}); err != nil {
		t.Fatalf("failed to seed manifest: %v", err)
	}
	if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
		BuildPhase: 1,
		Timestamp:  startedAt.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("failed to seed empty claims: %v", err)
	}

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	if _, err := runCodexBuild(root, 1, nil, false); err != nil {
		t.Fatalf("build retry returned error: %v", err)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT after retry", state.State)
	}
	if state.CurrentPhase != 1 {
		t.Fatalf("current phase = %d, want 1", state.CurrentPhase)
	}

	var manifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &manifest); err != nil {
		t.Fatalf("failed to reload manifest: %v", err)
	}
	if len(manifest.Dispatches) == 0 {
		t.Fatal("expected retried dispatches in manifest")
	}
	if len(manifest.WorkerBriefs) == 0 {
		t.Fatal("expected retried build to regenerate worker briefs")
	}
	for _, dispatch := range manifest.Dispatches {
		if dispatch.Status == "failed" {
			t.Fatalf("expected retried dispatches to avoid seeded failed status, got %+v", dispatch)
		}
	}
}

func floatPtr(v float64) *float64 { return &v }

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("failed to parse timestamp %q: %v", value, err)
	}
	return parsed
}

func TestResolvePheromoneSection_GroupsSignalsByType(t *testing.T) {
	saveGlobals(t)
	dataDir := t.TempDir() + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{Type: "FOCUS", Content: json.RawMessage(`{"text":"security"}`), Active: true, Strength: floatPtr(0.8), CreatedAt: "2026-04-16T00:00:00Z"},
			{Type: "REDIRECT", Content: json.RawMessage(`{"text":"avoid global state"}`), Active: true, Strength: floatPtr(0.9), CreatedAt: "2026-04-16T00:00:00Z"},
			{Type: "FEEDBACK", Content: json.RawMessage(`{"text":"prefer interfaces"}`), Active: true, Strength: floatPtr(0.7), CreatedAt: "2026-04-16T00:00:00Z"},
		},
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("failed to save pheromones: %v", err)
	}

	section := resolvePheromoneSection()
	if section == "" {
		t.Fatal("expected non-empty pheromone section when signals exist")
	}
	if !strings.Contains(section, "### Active Pheromone Signals") {
		t.Fatalf("missing section header in pheromone section:\n%s", section)
	}
	if !strings.Contains(section, "FOCUS") {
		t.Fatalf("missing FOCUS type in pheromone section:\n%s", section)
	}
	if !strings.Contains(section, "REDIRECT") {
		t.Fatalf("missing REDIRECT type in pheromone section:\n%s", section)
	}
	if !strings.Contains(section, "FEEDBACK") {
		t.Fatalf("missing FEEDBACK type in pheromone section:\n%s", section)
	}
	if !strings.Contains(section, "security") {
		t.Fatalf("missing signal content in pheromone section:\n%s", section)
	}
}

func TestResolvePheromoneSection_ReturnsEmptyWhenNoSignals(t *testing.T) {
	saveGlobals(t)
	dataDir := t.TempDir() + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	pf := colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("failed to save pheromones: %v", err)
	}

	section := resolvePheromoneSection()
	if section != "" {
		t.Fatalf("expected empty pheromone section when no signals, got:\n%s", section)
	}
}

func TestResolveSkillSection_FormatsMatchedSkills(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	hubDir := tmpDir + "/hub"
	skillsDir := hubDir + "/skills/colony/test-skill"
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	skillContent := "---\nname: test-skill\ntype: colony\ncategory: testing\nagent_roles:\n  - builder\n---\nThis is the test skill content."
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("failed to write skill: %v", err)
	}

	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	section := resolveSkillSection("builder", "testing task")
	if section == "" {
		t.Fatal("expected non-empty skill section when a matching skill exists")
	}
	if !strings.Contains(section, "### Skill: test-skill") {
		t.Fatalf("missing skill header in skill section:\n%s", section)
	}
	if !strings.Contains(section, "This is the test skill content") {
		t.Fatalf("missing skill content in skill section:\n%s", section)
	}
}

func TestResolveSkillSection_ReturnsEmptyWhenNoMatches(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	hubDir := tmpDir + "/hub"
	if err := os.MkdirAll(hubDir, 0755); err != nil {
		t.Fatalf("failed to create hub dir: %v", err)
	}

	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	section := resolveSkillSection("builder", "some task")
	if section != "" {
		t.Fatalf("expected empty skill section when no skills exist, got:\n%s", section)
	}
}

// TestBuildInRepo_VerifiesGitClaimsForCompletedWorkers proves that in-repo
// builds verify completed worker claims against actual git state.
// After the fix in task 04-01, completed workers have their claims
// checked via applyObservedClaims, not trusted blindly.
func TestBuildInRepo_VerifiesGitClaimsForCompletedWorkers(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")

	goal := "Verify in-repo claims are git-verified for completed workers"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
		ColonyDepth:  "light",
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Claims verification",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Create a file and verify claims", Status: colony.TaskPending}},
			}},
		},
	})

	// Create an invoker that creates a file and reports it as completed
	invoker := &inRepoClaimsInvoker{
		root: root,
	}
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	result, err := runCodexBuild(root, 1, nil, false)
	if err != nil {
		t.Fatalf("runCodexBuild returned error: %v", err)
	}

	dispatches, ok := result["dispatches"].([]map[string]interface{})
	if !ok || len(dispatches) == 0 {
		t.Fatalf("expected dispatches, got %v", result["dispatches"])
	}

	// Verify the worker completed
	dispatch := dispatches[0]
	if status, _ := dispatch["status"].(string); status != "completed" {
		t.Fatalf("expected completed status, got %q", status)
	}

	// Verify the claims file was written and contains the file
	var claims codexBuildClaims
	if err := store.LoadJSON("last-build-claims.json", &claims); err != nil {
		t.Fatalf("failed to load claims: %v", err)
	}

	foundClaimed := false
	for _, f := range claims.FilesCreated {
		if f == "pkg/feature.txt" {
			foundClaimed = true
			break
		}
	}
	if !foundClaimed {
		for _, f := range claims.FilesModified {
			if f == "pkg/feature.txt" {
				foundClaimed = true
				break
			}
		}
	}
	if !foundClaimed {
		t.Fatalf("expected pkg/feature.txt in claims, got FilesCreated=%v FilesModified=%v", claims.FilesCreated, claims.FilesModified)
	}

	// Verify the file exists on disk (proving git verification checked real state)
	if _, err := os.Stat(filepath.Join(root, "pkg", "feature.txt")); err != nil {
		t.Fatalf("expected pkg/feature.txt to exist on disk: %v", err)
	}
}

// inRepoClaimsInvoker is a test invoker that creates a file in-repo
// and reports completion with claimed files.
type inRepoClaimsInvoker struct {
	root string
}

func (i *inRepoClaimsInvoker) Invoke(_ context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	target := filepath.Join(cfg.Root, "pkg", "feature.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return codex.WorkerResult{}, err
	}
	if err := os.WriteFile(target, []byte("in-repo build change\n"), 0644); err != nil {
		return codex.WorkerResult{}, err
	}
	return codex.WorkerResult{
		WorkerName:   cfg.WorkerName,
		Caste:        cfg.Caste,
		TaskID:       cfg.TaskID,
		Status:       "completed",
		Summary:      "in-repo build completed",
		FilesCreated: []string{"pkg/feature.txt"},
	}, nil
}

func (i *inRepoClaimsInvoker) IsAvailable(_ context.Context) bool { return true }
func (i *inRepoClaimsInvoker) ValidateAgent(_ string) error       { return nil }
