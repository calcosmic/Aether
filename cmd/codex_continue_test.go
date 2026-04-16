package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestContinueConsumesBuildPacketAndAdvancesPhase(t *testing.T) {
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
		State:          colony.StateEXECUTING,
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
		"|Forge-11|completed|Closed by continue after verification",
		"|Ranger-12|completed|Closed by continue after verification",
		"|Keen-13|completed|Verification passed during continue",
	} {
		if !strings.Contains(string(spawnTreeData), want) {
			t.Fatalf("spawn tree missing completion line %q\n%s", want, string(spawnTreeData))
		}
	}
}

func TestContinueCompletesFinalPhase(t *testing.T) {
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
		State:          colony.StateEXECUTING,
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

func TestContinueBlocksWhenWatcherGateFails(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Block invalid advancement"
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
					Name:   "Watcher missing",
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
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-31", Task: "Do work one", Status: "spawned", TaskID: taskOneID},
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-32", Task: "Do work two", Status: "spawned", TaskID: taskTwoID},
		{Stage: "wave", Wave: 1, Caste: "scout", Name: "Ranger-33", Task: "Do work three", Status: "spawned", TaskID: taskThreeID},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Watcher missing", goal, dispatches)

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
	blockers := stringSliceValue(result["blocking_issues"])
	found := false
	for _, blocker := range blockers {
		if strings.Contains(blocker, "no watcher dispatch recorded") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected watcher gate blocker, got %v", blockers)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.State != colony.StateEXECUTING {
		t.Fatalf("state = %s, want EXECUTING", state.State)
	}
}

func seedContinueBuildPacket(t *testing.T, dataDir string, phase int, phaseName, goal string, dispatches []codexBuildDispatch) {
	t.Helper()

	buildDir := filepath.Join(dataDir, "build", fmt.Sprintf("phase-%d", phase))
	if err := os.MkdirAll(filepath.Join(buildDir, "worker-briefs"), 0755); err != nil {
		t.Fatalf("failed to create worker brief dir: %v", err)
	}

	briefs := make([]string, 0, len(dispatches))
	for _, dispatch := range dispatches {
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
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		State:        string(colony.StateEXECUTING),
		ClaimsPath:   displayDataPath("last-build-claims.json"),
		WorkerBriefs: briefs,
		Dispatches:   dispatches,
	}
	if err := store.SaveJSON(filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase), "manifest.json")), manifest); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{BuildPhase: phase, Timestamp: time.Now().UTC().Format(time.RFC3339)}); err != nil {
		t.Fatalf("failed to write claims: %v", err)
	}

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, dispatch := range dispatches {
		if err := spawnTree.RecordSpawn("Queen", dispatch.Caste, dispatch.Name, dispatch.Task, 1); err != nil {
			t.Fatalf("failed to seed spawn tree: %v", err)
		}
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
