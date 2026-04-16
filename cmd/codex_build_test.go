package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
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
	if got := int(result["dispatch_count"].(float64)); got != 6 {
		t.Fatalf("dispatch_count = %d, want 6", got)
	}
	if got := int(result["parallel_waves"].(float64)); got != 2 {
		t.Fatalf("parallel_waves = %d, want 2", got)
	}
	if next := result["next"].(string); next != "aether continue" {
		t.Fatalf("next = %q, want aether continue", next)
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
	if len(manifest.Dispatches) != 6 {
		t.Fatalf("expected 6 manifest dispatches, got %d", len(manifest.Dispatches))
	}
	if len(manifest.WorkerBriefs) != 6 {
		t.Fatalf("expected 6 worker briefs in manifest, got %d", len(manifest.WorkerBriefs))
	}
	if len(manifest.Tasks) != 2 {
		t.Fatalf("expected 2 planned tasks, got %d", len(manifest.Tasks))
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
	for _, want := range []string{"|Queen|scout|", "|Queen|builder|", "|Queen|oracle|", "|Queen|architect|", "|Queen|watcher|", "|Queen|chaos|"} {
		if !strings.Contains(string(spawnTreeData), want) {
			t.Fatalf("spawn tree missing %q\n%s", want, string(spawnTreeData))
		}
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if state.State != colony.StateEXECUTING {
		t.Fatalf("state = %s, want EXECUTING", state.State)
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
