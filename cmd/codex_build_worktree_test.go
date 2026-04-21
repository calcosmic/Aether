package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

type worktreeBuildInvoker struct {
	t         *testing.T
	mainRoot  string
	rootsSeen []string
}

func (i *worktreeBuildInvoker) Invoke(_ context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	i.rootsSeen = append(i.rootsSeen, cfg.Root)
	if cfg.Root == i.mainRoot {
		i.t.Fatalf("expected worktree root, got main root %s", cfg.Root)
	}
	if !strings.Contains(filepath.ToSlash(cfg.Root), ".aether/worktrees/") {
		i.t.Fatalf("expected worktree path, got %s", cfg.Root)
	}

	target := filepath.Join(cfg.Root, "pkg", "feature.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return codex.WorkerResult{}, err
	}
	if err := os.WriteFile(target, []byte("worktree build change\n"), 0644); err != nil {
		return codex.WorkerResult{}, err
	}

	return codex.WorkerResult{
		WorkerName:    cfg.WorkerName,
		Caste:         cfg.Caste,
		TaskID:        cfg.TaskID,
		Status:        "completed",
		Summary:       "worktree build completed",
		FilesModified: []string{"pkg/feature.txt"},
	}, nil
}

func (i *worktreeBuildInvoker) IsAvailable(_ context.Context) bool { return true }
func (i *worktreeBuildInvoker) ValidateAgent(_ string) error       { return nil }

type worktreePheromoneInvoker struct{}

func (i *worktreePheromoneInvoker) Invoke(_ context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	s, err := storage.NewStore(filepath.Join(cfg.Root, ".aether", "data"))
	if err != nil {
		return codex.WorkerResult{}, err
	}
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		return codex.WorkerResult{}, err
	}
	pf.Signals = append(pf.Signals, colony.PheromoneSignal{
		ID:        "sig-worktree-new",
		Type:      "FEEDBACK",
		Priority:  "low",
		Source:    "worker",
		CreatedAt: "2026-04-19T12:00:00Z",
		Active:    true,
		Content:   []byte(`{"text":"prefer narrower scopes"}`),
	})
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		return codex.WorkerResult{}, err
	}
	return codex.WorkerResult{
		WorkerName: cfg.WorkerName,
		Caste:      cfg.Caste,
		TaskID:     cfg.TaskID,
		Status:     "completed",
		Summary:    "worktree pheromone emitted",
	}, nil
}

func (i *worktreePheromoneInvoker) IsAvailable(_ context.Context) bool { return true }
func (i *worktreePheromoneInvoker) ValidateAgent(_ string) error       { return nil }

func TestBuildWorktreeModeDispatchesIntoIsolatedRoots(t *testing.T) {
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

	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "feature.txt"), []byte("base\n"), 0644); err != nil {
		t.Fatalf("failed to write feature file: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")

	goal := "Exercise worktree build execution"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
		ColonyDepth:  "light",
		ParallelMode: colony.ModeWorktree,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Worktree build",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Modify the feature file", Status: colony.TaskPending}},
			}},
		},
	})

	originalInvoker := newCodexWorkerInvoker
	invoker := &worktreeBuildInvoker{t: t, mainRoot: root}
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	result, err := runCodexBuild(root, 1, nil, false)
	if err != nil {
		t.Fatalf("runCodexBuild returned error: %v", err)
	}
	if got := result["parallel_mode"]; got != "worktree" {
		t.Fatalf("parallel_mode = %v, want worktree", got)
	}
	if len(invoker.rootsSeen) == 0 {
		t.Fatal("expected at least one worktree-root invocation")
	}

	data, err := os.ReadFile(filepath.Join(root, "pkg", "feature.txt"))
	if err != nil {
		t.Fatalf("failed to read synced file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "worktree build change" {
		t.Fatalf("expected worktree change synced back to root, got %q", string(data))
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if len(state.Worktrees) == 0 {
		t.Fatal("expected tracked worktrees in state")
	}
	if state.Worktrees[0].Status != colony.WorktreeMerged {
		t.Fatalf("worktree status = %s, want merged", state.Worktrees[0].Status)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(state.Worktrees[0].Path))); !os.IsNotExist(err) {
		t.Fatalf("expected cleaned up worktree path, got err=%v", err)
	}
}

func TestBuildWorktreeModeMergesPheromoneChangesBackToRoot(t *testing.T) {
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

	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")

	goal := "Exercise worktree pheromone merge-back"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
		ColonyDepth:  "light",
		ParallelMode: colony.ModeWorktree,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Worktree pheromone build",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Emit a pheromone in worktree mode", Status: colony.TaskPending}},
			}},
		},
	})
	if err := store.SaveJSON("pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{{
			ID:        "sig-root-1",
			Type:      "FOCUS",
			Priority:  "normal",
			Source:    "root",
			CreatedAt: "2026-04-19T10:00:00Z",
			Active:    true,
			Content:   []byte(`{"text":"security"}`),
		}},
	}); err != nil {
		t.Fatalf("save root pheromones: %v", err)
	}

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &worktreePheromoneInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	result, err := runCodexBuild(root, 1, nil, false)
	if err != nil {
		t.Fatalf("runCodexBuild returned error: %v", err)
	}

	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("reload pheromones: %v", err)
	}
	if len(pf.Signals) != 2 {
		t.Fatalf("expected root pheromones to include merged worktree signal, got %d signals", len(pf.Signals))
	}

	found := false
	for _, sig := range pf.Signals {
		if extractText(sig.Content) == "prefer narrower scopes" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected worktree-emitted pheromone to be merged back to root")
	}

	dispatches, ok := result["dispatches"].([]map[string]interface{})
	if !ok || len(dispatches) == 0 {
		t.Fatalf("dispatches shape = %#v", result["dispatches"])
	}
	summary, _ := dispatches[0]["summary"].(string)
	if !strings.Contains(summary, "Pheromone sync: 1 new") {
		t.Fatalf("dispatch summary should mention pheromone sync, got %q", summary)
	}
}

// --- Worktree Lifecycle Tests (Phase 23) ---

func TestRemoveGitWorktreeErrorPropagation(t *testing.T) {
	// Test with a non-existent path — should return error
	err := removeGitWorktree("/nonexistent", "/nonexistent/path", "nonexistent-branch")
	if err == nil {
		t.Error("removeGitWorktree should return error for non-existent path")
	}
	if !strings.Contains(err.Error(), "worktree cleanup failed") {
		t.Errorf("error should contain 'worktree cleanup failed': %v", err)
	}
}

func TestCleanupBuildWorktrees(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	_ = setupBuildFlowTest(t)

	// Create a state with a stale worktree entry
	now := time.Now().UTC().Format(time.RFC3339)
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    strPtr("test"),
		State:   colony.StateREADY,
		Worktrees: []colony.WorktreeEntry{
			{ID: "wt-1", Branch: "test-branch", Path: ".aether/worktrees/test", Status: colony.WorktreeAllocated, Phase: 1, CreatedAt: now, UpdatedAt: now},
		},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	// Run cleanup for phase 1
	cleaned, _, err := cleanupBuildWorktrees(1)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	// The worktree path doesn't exist on disk, so removal should "succeed" (no-op)
	// and the entry should be removed from state
	if cleaned != 1 {
		t.Errorf("expected 1 cleaned, got %d", cleaned)
	}

	// Verify entry removed from state
	var updated colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &updated); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(updated.Worktrees) > 0 {
		t.Errorf("expected worktree entry removed, got %d entries", len(updated.Worktrees))
	}
}

func TestAllocateBuildWorktreeCleansExistingPath(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupBuildFlowTest(t)

	root := storage.ResolveAetherRoot(context.Background())
	startedAt := time.Now().UTC()

	// Init a git repo so worktree add works
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")

	// Create a minimal COLONY_STATE.json so appendBuildWorktreeEntry can load it
	dataDir := filepath.Join(root, ".aether", "data")
	goal := "test"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
	})

	// Create a leftover directory at the expected worktree path
	dispatch := codex.WorkerDispatch{WorkerName: "test-worker", Caste: "builder"}
	branch := fmt.Sprintf("phase-1/%s-%d", sanitizeWorktreeLabel(dispatch.WorkerName), startedAt.UnixNano())
	relPath := filepath.ToSlash(filepath.Join(worktreeBaseDir, sanitizeBranchPath(branch)))
	absPath := filepath.Join(root, relPath)

	// Create a leftover directory
	if err := os.MkdirAll(absPath, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(absPath, "leftover.txt"), []byte("old"), 0644); err != nil {
		t.Fatalf("write leftover: %v", err)
	}

	// Now allocate should clean it up first
	_, err := allocateBuildWorktree(root, 1, dispatch, startedAt)
	if err != nil {
		t.Fatalf("allocate should succeed after cleaning leftover: %v", err)
	}

	// Verify the leftover file is gone (replaced by git worktree)
	if _, err := os.Stat(filepath.Join(absPath, "leftover.txt")); err == nil {
		t.Error("leftover file should have been cleaned up")
	}
}
