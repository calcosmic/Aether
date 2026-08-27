package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
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
	if cfg.Caste != "builder" {
		return codex.WorkerResult{
			WorkerName: cfg.WorkerName,
			Caste:      cfg.Caste,
			TaskID:     cfg.TaskID,
			Status:     "completed",
			Summary:    "read-only verification completed",
		}, nil
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

type overlappingWorktreeInvoker struct {
	mu      sync.Mutex
	arrived int
	release chan struct{}
	once    sync.Once
}

func newOverlappingWorktreeInvoker() *overlappingWorktreeInvoker {
	return &overlappingWorktreeInvoker{release: make(chan struct{})}
}

func (i *overlappingWorktreeInvoker) Invoke(ctx context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	if cfg.Caste != "builder" {
		return codex.WorkerResult{WorkerName: cfg.WorkerName, Caste: cfg.Caste, TaskID: cfg.TaskID, Status: "completed", Summary: "read-only worker completed"}, nil
	}

	content := []byte(cfg.WorkerName + "\n")
	if err := os.WriteFile(filepath.Join(cfg.Root, "shared.txt"), content, 0644); err != nil {
		return codex.WorkerResult{}, err
	}
	i.mu.Lock()
	i.arrived++
	if i.arrived == 2 {
		i.once.Do(func() { close(i.release) })
	}
	i.mu.Unlock()
	select {
	case <-i.release:
	case <-ctx.Done():
		return codex.WorkerResult{}, ctx.Err()
	case <-time.After(5 * time.Second):
		return codex.WorkerResult{}, fmt.Errorf("timed out waiting for overlapping builder peer")
	}

	return codex.WorkerResult{
		WorkerName:   cfg.WorkerName,
		Caste:        cfg.Caste,
		TaskID:       cfg.TaskID,
		Status:       "completed",
		Summary:      "created the shared path",
		FilesCreated: []string{"shared.txt"},
	}, nil
}

func (i *overlappingWorktreeInvoker) IsAvailable(context.Context) bool { return true }
func (i *overlappingWorktreeInvoker) ValidateAgent(string) error       { return nil }

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

func TestBuildWorktreeModeRejectsOverlappingUntrackedPaths(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-overlap\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")

	goal := "Detect overlapping worktree writes"
	taskOne := "1.1"
	taskTwo := "1.2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "light",
		ParallelMode: colony.ModeWorktree,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Overlap conflict",
			Status: colony.PhaseReady,
			Tasks: []colony.Task{
				{ID: &taskOne, Goal: "Create the shared output for path A", Status: colony.TaskPending},
				{ID: &taskTwo, Goal: "Create the shared output for path B", Status: colony.TaskPending},
			},
		}}},
	})

	originalInvoker := newCodexWorkerInvoker
	overlapInvoker := newOverlappingWorktreeInvoker()
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return overlapInvoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	_, err := runCodexBuild(root, 1, nil, false)
	if err == nil {
		t.Fatal("overlapping worktree writes succeeded; want an atomic wave rejection")
	}

	// The wave reconciles as one atomic decision: neither conflicting worker's
	// output reaches the root checkout.
	if _, statErr := os.Stat(filepath.Join(root, "shared.txt")); !os.IsNotExist(statErr) {
		data, _ := os.ReadFile(filepath.Join(root, "shared.txt"))
		t.Fatalf("rejected wave left worker output in the root checkout: %q", string(data))
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload colony state: %v", err)
	}
	if state.State == colony.StateBUILT {
		t.Fatal("overlapping worktree writes advanced colony to BUILT")
	}
	orphaned := 0
	for _, entry := range state.Worktrees {
		if entry.Status != colony.WorktreeOrphaned {
			continue
		}
		orphaned++
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(entry.Path))); err != nil {
			t.Fatalf("orphaned conflict worktree was not preserved at %s: %v", entry.Path, err)
		}
	}
	if orphaned < 2 {
		t.Fatalf("rejected wave preserved %d orphaned worktrees, want both conflicting workers: %+v", orphaned, state.Worktrees)
	}

	// The attempt journal records the reconciled terminal outcome per worker.
	_, attempt, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("no build attempt journal recorded for the rejected wave")
	}
	reconciliationErrors := 0
	for _, run := range attempt.WorkerRuns {
		if strings.Contains(run.Error, "wave reconciliation") {
			reconciliationErrors++
		}
	}
	if reconciliationErrors == 0 {
		t.Fatalf("attempt journal lacks wave reconciliation errors: %+v", attempt.WorkerRuns)
	}
	if _, _, cleanupErr := gcOrphanedWorktrees(); cleanupErr != nil {
		t.Fatalf("clean conflict worktree fixture: %v", cleanupErr)
	}
}

type countingWorktreeInvoker struct {
	mu    sync.Mutex
	calls int
}

func (i *countingWorktreeInvoker) Invoke(_ context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	i.mu.Lock()
	i.calls++
	i.mu.Unlock()
	return codex.WorkerResult{WorkerName: cfg.WorkerName, Caste: cfg.Caste, TaskID: cfg.TaskID, Status: "completed", Summary: "counted"}, nil
}

func (i *countingWorktreeInvoker) IsAvailable(context.Context) bool { return true }
func (i *countingWorktreeInvoker) ValidateAgent(string) error       { return nil }
func (i *countingWorktreeInvoker) callCount() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.calls
}

func TestBuildWorktreeModeGroupsDeclaredOverlapBeforeDispatch(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-declared-overlap\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")

	goal := "Group same-wave declared overlap before workers run"
	taskOne := "1.1"
	taskTwo := "1.2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "light",
		ParallelMode: colony.ModeWorktree,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Declared overlap",
			Status: colony.PhaseReady,
			Tasks: []colony.Task{
				{ID: &taskOne, Goal: "Produce the shared artifact for path A", Hints: []string{"shared.txt"}, Status: colony.TaskPending},
				{ID: &taskTwo, Goal: "Produce the shared artifact for path B", Hints: []string{"shared.txt"}, Status: colony.TaskPending},
			},
		}}},
	})

	originalInvoker := newCodexWorkerInvoker
	invoker := &countingWorktreeInvoker{}
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	if _, err := runCodexBuild(root, 1, nil, false); err != nil {
		t.Fatalf("grouped worktree overlap failed: %v", err)
	}
	if calls := invoker.callCount(); calls != 1 {
		t.Fatalf("declared overlap dispatched %d workers, want one coherent job", calls)
	}
	var manifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &manifest); err != nil {
		t.Fatalf("load grouped worktree manifest: %v", err)
	}
	waves := buildWaveDispatches(manifest.Dispatches)
	if len(waves) != 1 || !reflect.DeepEqual(waves[0].CoveredTaskIDs, []string{taskOne, taskTwo}) {
		t.Fatalf("worktree ownership validation ran before grouping: %+v", waves)
	}
}

type sequentialSharedFileInvoker struct {
	t *testing.T
}

func (i *sequentialSharedFileInvoker) Invoke(_ context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	if cfg.Caste != "builder" {
		return codex.WorkerResult{WorkerName: cfg.WorkerName, Caste: cfg.Caste, TaskID: cfg.TaskID, Status: "completed", Summary: "read-only worker completed"}, nil
	}
	sharedPath := filepath.Join(cfg.Root, "shared.txt")
	if strings.Contains(cfg.TaskBrief, "Extend the shared artifact") {
		if err := os.WriteFile(sharedPath, []byte("task-1.1+task-1.2\n"), 0644); err != nil {
			return codex.WorkerResult{}, err
		}
		return codex.WorkerResult{
			WorkerName: cfg.WorkerName, Caste: cfg.Caste, TaskID: cfg.TaskID, Status: "completed",
			Summary: "completed both covered tasks in one coherent job", FilesCreated: []string{"shared.txt"},
		}, nil
	}
	switch cfg.TaskID {
	case "1.1":
		if err := os.WriteFile(sharedPath, []byte("task-1.1\n"), 0644); err != nil {
			return codex.WorkerResult{}, err
		}
	case "1.2":
		data, err := os.ReadFile(sharedPath)
		if err != nil {
			return codex.WorkerResult{}, fmt.Errorf("dependent wave worktree lacks the earlier wave's synced file: %w", err)
		}
		if !strings.Contains(string(data), "task-1.1") {
			return codex.WorkerResult{}, fmt.Errorf("dependent wave worktree inherited %q, want the earlier wave's content", string(data))
		}
		if err := os.WriteFile(sharedPath, []byte("task-1.1+task-1.2\n"), 0644); err != nil {
			return codex.WorkerResult{}, err
		}
	}
	return codex.WorkerResult{
		WorkerName:   cfg.WorkerName,
		Caste:        cfg.Caste,
		TaskID:       cfg.TaskID,
		Status:       "completed",
		Summary:      "advanced the shared file",
		FilesCreated: []string{"shared.txt"},
	}, nil
}

func (i *sequentialSharedFileInvoker) IsAvailable(context.Context) bool { return true }
func (i *sequentialSharedFileInvoker) ValidateAgent(string) error       { return nil }

func TestBuildWorktreeModeGroupsDeclaredOverlapAcrossFormerWaves(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-cross-wave\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")

	goal := "Sequential waves build on the same declared file"
	taskOne := "1.1"
	taskTwo := "1.2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "light",
		ParallelMode: colony.ModeWorktree,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Cross-wave declared overlap",
			Status: colony.PhaseReady,
			Tasks: []colony.Task{
				{ID: &taskOne, Goal: "Create the shared artifact", Hints: []string{"shared.txt"}, Status: colony.TaskPending},
				{ID: &taskTwo, Goal: "Extend the shared artifact", Hints: []string{"shared.txt"}, DependsOn: []string{"1.1"}, Status: colony.TaskPending},
			},
		}}},
	})

	originalInvoker := newCodexWorkerInvoker
	invoker := &sequentialSharedFileInvoker{t: t}
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	if _, err := runCodexBuild(root, 1, nil, false); err != nil {
		t.Fatalf("cross-wave declared overlap failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "shared.txt"))
	if err != nil {
		t.Fatalf("read cross-wave shared output: %v", err)
	}
	if strings.TrimSpace(string(data)) != "task-1.1+task-1.2" {
		t.Fatalf("cross-wave shared output = %q, want the sequentially advanced content", string(data))
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload colony state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("cross-wave declared overlap ended in %s, want BUILT", state.State)
	}
}

type declaredViolationInvoker struct{}

func (i *declaredViolationInvoker) Invoke(_ context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	if cfg.Caste != "builder" {
		return codex.WorkerResult{WorkerName: cfg.WorkerName, Caste: cfg.Caste, TaskID: cfg.TaskID, Status: "completed", Summary: "read-only worker completed"}, nil
	}
	target := "first.txt"
	if cfg.TaskID == "1.2" {
		target = "owned.txt"
	}
	if err := os.WriteFile(filepath.Join(cfg.Root, target), []byte(cfg.WorkerName+"\n"), 0644); err != nil {
		return codex.WorkerResult{}, err
	}
	return codex.WorkerResult{
		WorkerName:   cfg.WorkerName,
		Caste:        cfg.Caste,
		TaskID:       cfg.TaskID,
		Status:       "completed",
		Summary:      "wrote " + target,
		FilesCreated: []string{target},
	}, nil
}

func (i *declaredViolationInvoker) IsAvailable(context.Context) bool { return true }
func (i *declaredViolationInvoker) ValidateAgent(string) error       { return nil }

func TestBuildWorktreeModeRejectsDeclaredPathViolation(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-drift\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")

	goal := "Reject a worker that touches another task's declared path"
	taskOne := "1.1"
	taskTwo := "1.2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "light",
		ParallelMode: colony.ModeWorktree,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Declared path violation",
			Status: colony.PhaseReady,
			Tasks: []colony.Task{
				{ID: &taskOne, Goal: "Write the first artifact", Hints: []string{"owned.txt"}, Status: colony.TaskPending},
				{ID: &taskTwo, Goal: "Write the second artifact", Status: colony.TaskPending},
			},
		}}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &declaredViolationInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	_, err := runCodexBuild(root, 1, nil, false)
	if err == nil {
		t.Fatal("declared path violation succeeded; want an atomic wave rejection")
	}

	// All-or-nothing: the violating worker's drift and the innocent worker's
	// valid output both stay out of the root checkout.
	for _, rel := range []string{"owned.txt", "first.txt"} {
		if _, statErr := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(statErr) {
			t.Fatalf("rejected wave left %s in the root checkout", rel)
		}
	}

	_, attempt, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("no build attempt journal recorded for the rejected wave")
	}
	violatorFound := false
	bystanderFound := false
	for _, run := range attempt.WorkerRuns {
		switch run.TaskID {
		case "1.2":
			if run.Status == "failed" && strings.Contains(run.Error, "declared by task 1.1") {
				violatorFound = true
			}
		case "1.1":
			if run.Status == "blocked" {
				bystanderFound = true
			}
		}
	}
	if !violatorFound {
		t.Fatalf("journal lacks the violating worker's declared-ownership failure: %+v", attempt.WorkerRuns)
	}
	if !bystanderFound {
		t.Fatalf("journal lacks the innocent worker's blocked outcome: %+v", attempt.WorkerRuns)
	}
	if _, _, cleanupErr := gcOrphanedWorktrees(); cleanupErr != nil {
		t.Fatalf("clean conflict worktree fixture: %v", cleanupErr)
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
	// Test with a non-existent path — should return error. Per CR-03,
	// removeGitWorktree now fails fast on the first failing step rather than
	// accumulating errors from all three commands, so the message names the
	// specific step that failed ("worktree remove: ...") instead of the old
	// generic "worktree cleanup failed" wrapper.
	err := removeGitWorktree("/nonexistent", "/nonexistent/path", "nonexistent-branch")
	if err == nil {
		t.Error("removeGitWorktree should return error for non-existent path")
	}
	if !strings.Contains(err.Error(), "worktree remove") {
		t.Errorf("error should contain 'worktree remove': %v", err)
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

func TestDetectOrphanedWorktrees(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Test"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Worktrees: []colony.WorktreeEntry{
			{Phase: 1, Branch: "phase-1-wt", Status: colony.WorktreeInProgress, Path: ".aether/worktrees/wt1"},
			{Phase: 2, Branch: "phase-2-wt", Status: colony.WorktreeInProgress, Path: ".aether/worktrees/wt2"},
			{Phase: 2, Branch: "phase-2-merged", Status: colony.WorktreeMerged, Path: ".aether/worktrees/wt3"},
		},
	})

	// Current phase 1 should detect phase-2 unmerged worktree as orphan
	orphans := detectOrphanedWorktrees(1)
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan for current phase 1, got %d", len(orphans))
	}
	if orphans[0].Branch != "phase-2-wt" {
		t.Errorf("expected orphan branch phase-2-wt, got %s", orphans[0].Branch)
	}

	// Current phase 2 should detect phase-1 as orphan
	orphans = detectOrphanedWorktrees(2)
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan for current phase 2, got %d", len(orphans))
	}
	if orphans[0].Branch != "phase-1-wt" {
		t.Errorf("expected orphan branch phase-1-wt, got %s", orphans[0].Branch)
	}

	// Current phase 3 should detect both unmerged as orphans
	orphans = detectOrphanedWorktrees(3)
	if len(orphans) != 2 {
		t.Fatalf("expected 2 orphans for current phase 3, got %d", len(orphans))
	}
}

func TestDetectOrphanedWorktreesNone(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Test"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:   "3.0",
		Goal:      &goal,
		State:     colony.StateREADY,
		Worktrees: []colony.WorktreeEntry{},
	})

	orphans := detectOrphanedWorktrees(1)
	if len(orphans) != 0 {
		t.Errorf("expected 0 orphans, got %d", len(orphans))
	}
}

func TestMergePhaseWorktreesEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Test"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:   "3.0",
		Goal:      &goal,
		State:     colony.StateREADY,
		Worktrees: []colony.WorktreeEntry{},
	})

	merged, failed, err := mergePhaseWorktrees(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(merged) != 0 {
		t.Errorf("expected 0 merged, got %d", len(merged))
	}
	if len(failed) != 0 {
		t.Errorf("expected 0 failed, got %d", len(failed))
	}
}

// --- Phase 195 / JOBS-04: coherent jobs in isolated worktree mode ---

// calVaultWorktreeInvoker is the literal six-batch CalVault shape from the
// 2026-08-14 field run, executed in worktree mode: one source list, six
// dependent copy steps. Every invocation records the checkout it ran in so
// the test can prove one grouped job took one worktree, not six.
type calVaultWorktreeInvoker struct {
	mu        sync.Mutex
	rootsSeen []string
	briefs    []string
}

func (i *calVaultWorktreeInvoker) Invoke(_ context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	i.mu.Lock()
	i.rootsSeen = append(i.rootsSeen, cfg.Root)
	i.briefs = append(i.briefs, cfg.TaskBrief)
	i.mu.Unlock()

	if cfg.Caste != "builder" {
		return codex.WorkerResult{
			WorkerName: cfg.WorkerName,
			Caste:      cfg.Caste,
			TaskID:     cfg.TaskID,
			Status:     "completed",
			Summary:    "read-only worker completed",
		}, nil
	}

	created := make([]string, 0, 6)
	for batch := 1; batch <= 6; batch++ {
		rel := calVaultBatchPath(batch)
		target := filepath.Join(cfg.Root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return codex.WorkerResult{}, err
		}
		if err := os.WriteFile(target, []byte(fmt.Sprintf("batch %d copied\n", batch)), 0644); err != nil {
			return codex.WorkerResult{}, err
		}
		created = append(created, rel)
	}
	return codex.WorkerResult{
		WorkerName:   cfg.WorkerName,
		Caste:        cfg.Caste,
		TaskID:       cfg.TaskID,
		Status:       "completed",
		Summary:      "copied all six recovery-matrix batches in one pass",
		FilesCreated: created,
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "pass",
			CommandsRun:        []string{"ls templates"},
		},
	}, nil
}

func (i *calVaultWorktreeInvoker) IsAvailable(context.Context) bool { return true }
func (i *calVaultWorktreeInvoker) ValidateAgent(string) error       { return nil }

func (i *calVaultWorktreeInvoker) builderRoots() []string {
	i.mu.Lock()
	defer i.mu.Unlock()
	out := make([]string, 0, len(i.rootsSeen))
	for _, r := range i.rootsSeen {
		if strings.Contains(filepath.ToSlash(r), ".aether/worktrees/") {
			out = append(out, r)
		}
	}
	return out
}

func calVaultBatchPath(batch int) string {
	return fmt.Sprintf("templates/batch-%d.md", batch)
}

// calVaultSixBatchTasks reproduces the field fixture exactly: six dependent
// copy steps over the same recovery-matrix source list, each declaring its
// own output file.
func calVaultSixBatchTasks() ([]colony.Task, []string) {
	ids := make([]string, 0, 6)
	tasks := make([]colony.Task, 0, 6)
	prev := ""
	for batch := 1; batch <= 6; batch++ {
		id := fmt.Sprintf("3.%d", batch)
		ids = append(ids, id)
		taskID := id
		task := colony.Task{
			ID:     &taskID,
			Goal:   fmt.Sprintf("Copy CalVault file batch %d and record its recovery-matrix entries", batch),
			Status: colony.TaskPending,
			Hints:  []string{calVaultBatchPath(batch)},
		}
		if prev != "" {
			task.DependsOn = []string{prev}
		}
		prev = id
		tasks = append(tasks, task)
	}
	return tasks, ids
}

func newCalVaultWorktreeRepo(t *testing.T) (string, string) {
	t.Helper()
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-calvault\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	return dataDir, root
}

// TestCalVaultSixBatchesBecomeOneWorktreeJob is the worktree-mode counterpart
// of TestCalVaultSixBatchesBecomeOneInRepoJob. Phase 184/H5 deliberately
// excluded worktree mode from coalescing, which meant the exact field failure
// this project measured -- six fresh ~100k-token agents re-reading the same
// source list -- was only fixed for in-repo users. JOBS-04 requires the same
// one job to take exactly one worktree, one branch, one worker session and one
// merge-back.
func TestCalVaultSixBatchesBecomeOneWorktreeJob(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir, root := newCalVaultWorktreeRepo(t)

	goal := "Copy the identified templates into a new folder tree, keeping a log"
	tasks, ids := calVaultSixBatchTasks()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "light",
		CurrentPhase: 0,
		ParallelMode: colony.ModeWorktree,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:          1,
			Name:        "Template recovery",
			Description: "Copy the identified templates into a new folder tree, keeping a log.",
			Status:      colony.PhaseReady,
			Tasks:       tasks,
		}}},
	})

	originalInvoker := newCodexWorkerInvoker
	invoker := &calVaultWorktreeInvoker{}
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	if _, err := runCodexBuild(root, 1, nil, false); err != nil {
		t.Fatalf("CalVault worktree build returned error: %v", err)
	}

	builderRoots := invoker.builderRoots()
	if len(builderRoots) != 1 {
		t.Fatalf("six dependent CalVault batches ran in %d worktree checkouts, want exactly 1 coherent job: %v",
			len(builderRoots), builderRoots)
	}

	// One grouped dispatch, carrying every covered task in order.
	var manifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &manifest); err != nil {
		t.Fatalf("load CalVault worktree manifest: %v", err)
	}
	waves := buildWaveDispatches(manifest.Dispatches)
	if len(waves) != 1 {
		t.Fatalf("CalVault worktree build planned %d task workers, want 1: %+v", len(waves), waves)
	}
	if !reflect.DeepEqual(waves[0].CoveredTaskIDs, ids) {
		t.Fatalf("grouped worktree job covers %v, want the six batches in order %v", waves[0].CoveredTaskIDs, ids)
	}
	if strings.TrimSpace(waves[0].JobReason) == "" {
		t.Fatalf("grouped worktree job carries no reason: %+v", waves[0])
	}

	// Exactly one worktree, allocated once and merged once.
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload colony state: %v", err)
	}
	if len(state.Worktrees) != 1 {
		t.Fatalf("CalVault worktree build allocated %d worktrees, want 1: %+v", len(state.Worktrees), state.Worktrees)
	}
	if state.Worktrees[0].Status != colony.WorktreeMerged {
		t.Fatalf("CalVault worktree status = %s, want merged", state.Worktrees[0].Status)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(state.Worktrees[0].Path))); !os.IsNotExist(err) {
		t.Fatalf("merged CalVault worktree was not cleaned up: err=%v", err)
	}

	// The merge-back brought every batch into the root checkout, and every
	// covered task is credited.
	for batch := 1; batch <= 6; batch++ {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(calVaultBatchPath(batch)))); err != nil {
			t.Fatalf("batch %d never reached the root checkout: %v", batch, err)
		}
	}
	for _, task := range state.Plan.Phases[0].Tasks {
		if task.Status != colony.TaskCompleted {
			t.Fatalf("task %s is %q after the one grouped worker finished every batch, want %q",
				*task.ID, task.Status, colony.TaskCompleted)
		}
	}
}

// TestDistinctWorktreeJobsStillRejectOverlap keeps the guard that grouping is
// not allowed to weaken. Two genuinely unrelated tasks that only share an
// incidental bookkeeping file are deliberately NOT combined into one job
// (D-01), so they remain two owners of one path in the same wave -- which
// worktree mode still cannot reconcile and must refuse before any worker runs.
func TestDistinctWorktreeJobsStillRejectOverlap(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir, root := newCalVaultWorktreeRepo(t)

	goal := "Two unrelated jobs must not silently share one declared path"
	taskOne := "1.1"
	taskTwo := "1.2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "light",
		ParallelMode: colony.ModeWorktree,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Unrelated overlap",
			Status: colony.PhaseReady,
			Tasks: []colony.Task{
				{ID: &taskOne, Goal: "Document the import pipeline", Hints: []string{"README.md"}, Status: colony.TaskPending},
				{ID: &taskTwo, Goal: "Document the export pipeline", Hints: []string{"README.md"}, Status: colony.TaskPending},
			},
		}}},
	})

	originalInvoker := newCodexWorkerInvoker
	invoker := &countingWorktreeInvoker{}
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	_, err := runCodexBuild(root, 1, nil, false)
	if err == nil {
		t.Fatal("two unrelated jobs declaring one path were accepted; want an atomic pre-dispatch refusal")
	}
	if !strings.Contains(err.Error(), "README.md") {
		t.Fatalf("ownership refusal does not name the contested path: %v", err)
	}
	if calls := invoker.callCount(); calls != 0 {
		t.Fatalf("ownership refusal ran %d workers; the conflict must be caught before any worker starts", calls)
	}
	if _, statErr := os.Stat(filepath.Join(root, "README.md")); !os.IsNotExist(statErr) {
		t.Fatal("refused build left worker output in the root checkout")
	}
}

// TestGroupedWorktreeOwnsUnionedPaths asserts the real dispatch list the
// worktree adapter consumes, not an intermediate planning record: one grouped
// job must reach codex.WorkerDispatch carrying every covered task ID in order
// and the unique sorted union of all its tasks' declared paths, so the
// intentional within-job overlap has exactly one owner.
func TestGroupedWorktreeOwnsUnionedPaths(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir, root := newCalVaultWorktreeRepo(t)
	_ = dataDir

	tasks, ids := calVaultSixBatchTasks()
	phase := colony.Phase{
		ID:          1,
		Name:        "Template recovery",
		Description: "Copy the identified templates into a new folder tree, keeping a log.",
		Status:      colony.PhaseReady,
		Tasks:       tasks,
	}

	planned := waveDispatchesOnly(plannedBuildDispatchesForSelectionWithState(
		phase, colony.ColonyState{ParallelMode: colony.ModeWorktree}, nil, colony.VerificationDepthStandard))
	if len(planned) != 1 {
		t.Fatalf("planner produced %d task jobs, want 1 before worktree ownership runs", len(planned))
	}

	workerDispatches, err := buildCodexWorkerDispatches(
		root, phase, planned, time.Now().UTC(), &codex.FakeInvoker{}, "", time.Minute, nil)
	if err != nil {
		t.Fatalf("convert grouped dispatch for worktree execution: %v", err)
	}
	if len(workerDispatches) != 1 {
		t.Fatalf("grouped job became %d worker dispatches, want 1", len(workerDispatches))
	}
	got := workerDispatches[0]
	if !reflect.DeepEqual(got.CoveredTaskIDs, ids) {
		t.Fatalf("WorkerDispatch.CoveredTaskIDs = %v, want the six batches in order %v", got.CoveredTaskIDs, ids)
	}
	if strings.TrimSpace(got.JobName) == "" || strings.TrimSpace(got.JobReason) == "" {
		t.Fatalf("WorkerDispatch lost its grouped job identity: name=%q reason=%q", got.JobName, got.JobReason)
	}
	wantPaths := make([]string, 0, 6)
	for batch := 1; batch <= 6; batch++ {
		wantPaths = append(wantPaths, calVaultBatchPath(batch))
	}
	if !reflect.DeepEqual(got.DeclaredPaths, wantPaths) {
		t.Fatalf("WorkerDispatch.DeclaredPaths = %v, want the unique sorted union %v", got.DeclaredPaths, wantPaths)
	}

	// One owner for the whole union: the same-wave ownership guard must not
	// see a grouped job's intentional internal overlap as a conflict.
	if err := validateDeclaredWorktreeOwnership(workerDispatches); err != nil {
		t.Fatalf("grouped job's own unioned paths were refused as a conflict: %v", err)
	}

	// Two distinct jobs sharing one path still fail, and the refusal names
	// both jobs, not just their primary task IDs.
	rival := got
	rival.WorkerName = "Rival-1"
	rival.TaskID = "9.9"
	rival.CoveredTaskIDs = []string{"9.9"}
	rival.JobName = "rival-job"
	conflictErr := validateDeclaredWorktreeOwnership([]codex.WorkerDispatch{got, rival})
	if conflictErr == nil {
		t.Fatal("two distinct jobs declaring the same path were accepted")
	}
	if !strings.Contains(conflictErr.Error(), "rival-job") || !strings.Contains(conflictErr.Error(), got.JobName) {
		t.Fatalf("ownership conflict does not name both jobs: %v", conflictErr)
	}
}
