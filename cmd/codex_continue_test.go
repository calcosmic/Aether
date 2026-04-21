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
	"github.com/calcosmic/Aether/pkg/storage"
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

func TestContinueAdvancesWithoutWatcherDispatchWhenVerificationPasses(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Advance without watcher dispatch when verification passes"
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
					Name:   "Watcher optional when verification is real",
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
	seedContinueBuildPacket(t, dataDir, 1, "Watcher optional when verification is real", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); blocked {
		t.Fatalf("expected blocked:false, got %v", result)
	}
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true, got %v", result)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Fatalf("state = %s, want COMPLETED", state.State)
	}
}

func TestContinueAdvancesOnVerifiedPartialSuccess(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Advance despite partial worker failure"
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
					Name:   "Partial success phase",
					Status: colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Land the first change", Status: colony.TaskInProgress},
						{ID: &taskTwoID, Goal: "Land the second change", Status: colony.TaskInProgress},
					},
				},
				{
					ID:     2,
					Name:   "Next verified phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Keep going", Status: colony.TaskPending}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-61", Task: "Land the first change", Status: "completed", TaskID: taskOneID},
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-62", Task: "Land the second change", Status: "timeout", TaskID: taskTwoID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-63", Task: "Independent verification before advancement", Status: "blocked"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Partial success phase", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true, got %v", result)
	}
	if partial, _ := result["partial_success"].(bool); !partial {
		t.Fatalf("expected partial_success:true, got %v", result)
	}
	issues := stringSliceValue(result["operational_issues"])
	if len(issues) == 0 {
		t.Fatalf("expected operational issues in partial success result, got %v", result)
	}

	spawnTreeData, err := os.ReadFile(filepath.Join(dataDir, "spawn-tree.txt"))
	if err != nil {
		t.Fatalf("failed to read spawn tree: %v", err)
	}
	for _, want := range []string{
		"|Forge-61|completed|Completed before continue verification",
		"|Forge-62|timeout|",
		"|Keen-63|blocked|",
	} {
		if !strings.Contains(string(spawnTreeData), want) {
			t.Fatalf("spawn tree missing partial-success line %q\n%s", want, string(spawnTreeData))
		}
	}
}

func TestContinueAllowsManualReconciliationForVerifiedManualFix(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Advance after a manual fix"
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
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true, got %v", result)
	}
	reconciled := stringSliceValue(result["reconciled_tasks"])
	if len(reconciled) != 1 || reconciled[0] != taskID {
		t.Fatalf("expected reconciled task %s, got %v", taskID, reconciled)
	}

	spawnTreeData, err := os.ReadFile(filepath.Join(dataDir, "spawn-tree.txt"))
	if err != nil {
		t.Fatalf("failed to read spawn tree: %v", err)
	}
	if !strings.Contains(string(spawnTreeData), "|Forge-71|manually-reconciled|Task was manually reconciled before continue advancement") {
		t.Fatalf("expected manually-reconciled worker in spawn tree, got:\n%s", string(spawnTreeData))
	}
}

func TestContinueBlockedResultSuggestsTargetedRedispatch(t *testing.T) {
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
}

func TestContinueUsesManifestWhenBuildStartedAtIsMissing(t *testing.T) {
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
		DispatchMode: "simulated",
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

func TestVerifyCodexBuildClaims_SimulatedMode_AllCompleted_Passes(t *testing.T) {
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
	if !result.Passed {
		t.Fatalf("expected Passed=true for simulated mode (all dispatches completed), got Passed=false: %s", result.Summary)
	}
	if !strings.Contains(result.Summary, "simulated mode") {
		t.Fatalf("expected summary to mention simulated mode, got: %s", result.Summary)
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
