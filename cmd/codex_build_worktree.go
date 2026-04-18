package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

type buildWorktreeSession struct {
	Branch  string
	RelPath string
	AbsPath string
}

func effectiveParallelMode(state colony.ColonyState) colony.ParallelMode {
	if state.ParallelMode.Valid() {
		return state.ParallelMode
	}
	return colony.ModeInRepo
}

func dispatchCodexBuildWorkers(ctx context.Context, root string, phase colony.Phase, dispatches []codex.WorkerDispatch, invoker codex.WorkerInvoker, startedAt time.Time, parallelMode colony.ParallelMode) ([]codex.DispatchResult, error) {
	if parallelMode != colony.ModeWorktree {
		return codex.DispatchBatch(ctx, invoker, dispatches)
	}
	if _, ok := invoker.(*codex.FakeInvoker); ok {
		return codex.DispatchBatch(ctx, invoker, dispatches)
	}
	if err := ensureGitRepository(root); err != nil {
		return nil, fmt.Errorf("worktree mode requires a git repository: %w", err)
	}

	waves := codex.GroupByWave(dispatches)
	waveNumbers := make([]int, 0, len(waves))
	for wave := range waves {
		waveNumbers = append(waveNumbers, wave)
	}
	sort.Ints(waveNumbers)

	var results []codex.DispatchResult
	for _, wave := range waveNumbers {
		for _, dispatch := range waves[wave] {
			if ctx.Err() != nil {
				results = append(results, codex.DispatchResult{
					WorkerName: dispatch.WorkerName,
					Status:     "timeout",
					Error:      ctx.Err(),
				})
				continue
			}

			session, err := allocateBuildWorktree(root, phase.ID, dispatch, startedAt)
			if err != nil {
				return nil, fmt.Errorf("allocate worktree for %s: %w", dispatch.WorkerName, err)
			}

			if err := updateBuildWorktreeStatus(session.Branch, colony.WorktreeInProgress); err != nil {
				_ = finalizeBuildWorktree(root, session, colony.WorktreeOrphaned)
				return nil, fmt.Errorf("mark worktree in progress for %s: %w", dispatch.WorkerName, err)
			}

			baseline, err := snapshotWorktreeStatus(session.AbsPath)
			if err != nil {
				_ = finalizeBuildWorktree(root, session, colony.WorktreeOrphaned)
				return nil, fmt.Errorf("snapshot worktree for %s: %w", dispatch.WorkerName, err)
			}

			cfg := codex.WorkerConfig{
				AgentName:        dispatch.AgentName,
				AgentTOMLPath:    dispatch.AgentTOMLPath,
				Caste:            dispatch.Caste,
				WorkerName:       dispatch.WorkerName,
				TaskID:           dispatch.TaskID,
				TaskBrief:        dispatch.TaskBrief,
				ContextCapsule:   dispatch.ContextCapsule,
				Root:             session.AbsPath,
				Timeout:          dispatch.Timeout,
				SkillSection:     dispatch.SkillSection,
				PheromoneSection: dispatch.PheromoneSection,
			}

			result, invokeErr := invoker.Invoke(ctx, cfg)
			dr := codex.DispatchResult{
				WorkerName: dispatch.WorkerName,
			}
			if invokeErr != nil {
				dr.Status = "failed"
				dr.Error = invokeErr
			} else {
				dr.Status = result.Status
				dr.WorkerResult = &result
				if result.Error != nil {
					dr.Error = result.Error
				}
			}

			finalStatus := colony.WorktreeMerged
			if dr.Status != "completed" || dr.WorkerResult == nil {
				finalStatus = colony.WorktreeOrphaned
			} else {
				touched, touchErr := collectWorktreeTouchedPaths(session.AbsPath, baseline, result)
				if touchErr != nil {
					dr.Status = "failed"
					dr.Error = touchErr
					finalStatus = colony.WorktreeOrphaned
				} else if syncErr := syncWorktreeChangesToRoot(root, session.AbsPath, touched); syncErr != nil {
					dr.Status = "failed"
					dr.Error = syncErr
					finalStatus = colony.WorktreeOrphaned
				}
			}

			if cleanupErr := finalizeBuildWorktree(root, session, finalStatus); cleanupErr != nil && dr.Error == nil {
				dr.Status = "failed"
				dr.Error = cleanupErr
			}
			results = append(results, dr)
		}
	}
	return results, nil
}

func ensureGitRepository(root string) error {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--show-toplevel")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func allocateBuildWorktree(root string, phaseID int, dispatch codex.WorkerDispatch, startedAt time.Time) (*buildWorktreeSession, error) {
	branch := fmt.Sprintf("phase-%d/%s-%d", phaseID, sanitizeWorktreeLabel(dispatch.WorkerName), startedAt.UnixNano())
	if err := validateBranchName(branch); err != nil {
		return nil, err
	}
	relPath := filepath.ToSlash(filepath.Join(worktreeBaseDir, sanitizeBranchPath(branch)))
	absPath := filepath.Join(root, relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "worktree", "add", "-b", branch, absPath, "HEAD")
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git worktree add: %v: %s", err, strings.TrimSpace(string(out)))
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := appendBuildWorktreeEntry(colony.WorktreeEntry{
		ID:        generateWorktreeID(),
		Branch:    branch,
		Path:      relPath,
		Status:    colony.WorktreeAllocated,
		Phase:     phaseID,
		Agent:     dispatch.WorkerName,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		_ = removeGitWorktree(root, absPath, branch)
		return nil, err
	}

	if err := syncRootRuntimeIntoWorktree(root, absPath); err != nil {
		_ = updateBuildWorktreeStatus(branch, colony.WorktreeOrphaned)
		_ = removeGitWorktree(root, absPath, branch)
		return nil, err
	}
	return &buildWorktreeSession{Branch: branch, RelPath: relPath, AbsPath: absPath}, nil
}

func sanitizeWorktreeLabel(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastHyphen := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			b.WriteRune('-')
			lastHyphen = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "worker"
	}
	return out
}

func appendBuildWorktreeEntry(entry colony.WorktreeEntry) error {
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return err
	}
	state.Worktrees = append(state.Worktrees, entry)
	return store.SaveJSON("COLONY_STATE.json", state)
}

func updateBuildWorktreeStatus(branch string, status colony.WorktreeStatus) error {
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range state.Worktrees {
		if state.Worktrees[i].Branch != branch {
			continue
		}
		state.Worktrees[i].Status = status
		state.Worktrees[i].UpdatedAt = now
		return store.SaveJSON("COLONY_STATE.json", state)
	}
	return fmt.Errorf("worktree %q not tracked in colony state", branch)
}

func finalizeBuildWorktree(root string, session *buildWorktreeSession, status colony.WorktreeStatus) error {
	if session == nil {
		return nil
	}
	if err := updateBuildWorktreeStatus(session.Branch, status); err != nil {
		return err
	}
	return removeGitWorktree(root, session.AbsPath, session.Branch)
}

func removeGitWorktree(root, absPath, branch string) error {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	exec.CommandContext(ctx, "git", "-C", root, "worktree", "remove", absPath, "--force").Run()
	exec.CommandContext(ctx, "git", "-C", root, "worktree", "prune").Run()
	exec.CommandContext(ctx, "git", "-C", root, "branch", "-D", branch).Run()
	return nil
}

func syncRootRuntimeIntoWorktree(root, worktreePath string) error {
	for _, rel := range []string{
		".aether/CONTEXT.md",
		".aether/HANDOFF.md",
		".aether/data/COLONY_STATE.json",
		".aether/data/pheromones.json",
		".aether/data/session.json",
	} {
		if err := syncRelativePath(root, worktreePath, rel); err != nil {
			return err
		}
	}
	statuses, err := snapshotGitStatus(root)
	if err != nil {
		return err
	}
	for rel, status := range statuses {
		if strings.HasPrefix(rel, ".aether/worktrees/") {
			continue
		}
		if err := applyRelativePathStatus(root, worktreePath, rel, status); err != nil {
			return err
		}
	}
	return nil
}

func snapshotWorktreeStatus(worktreePath string) (map[string]string, error) {
	return snapshotGitStatus(worktreePath)
}

func snapshotGitStatus(root string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain", "--untracked-files=all")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git status: %v: %s", err, strings.TrimSpace(string(out)))
	}

	statuses := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) < 4 {
			continue
		}
		status := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[3:])
		if idx := strings.LastIndex(path, " -> "); idx >= 0 {
			path = strings.TrimSpace(path[idx+4:])
		}
		if path == "" {
			continue
		}
		statuses[filepath.ToSlash(path)] = status
	}
	return statuses, nil
}

func collectWorktreeTouchedPaths(worktreePath string, baseline map[string]string, result codex.WorkerResult) ([]string, error) {
	paths := map[string]struct{}{}
	for _, rel := range append(append([]string{}, result.FilesCreated...), result.FilesModified...) {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel != "" {
			paths[rel] = struct{}{}
		}
	}
	for _, rel := range result.TestsWritten {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel != "" {
			paths[rel] = struct{}{}
		}
	}

	current, err := snapshotWorktreeStatus(worktreePath)
	if err != nil {
		return nil, err
	}
	for rel, status := range current {
		if baseline[rel] != status {
			paths[rel] = struct{}{}
		}
	}
	for rel := range baseline {
		if _, ok := current[rel]; !ok {
			paths[rel] = struct{}{}
		}
	}

	out := make([]string, 0, len(paths))
	for rel := range paths {
		if rel == "" || strings.HasPrefix(rel, ".aether/worktrees/") {
			continue
		}
		out = append(out, rel)
	}
	sort.Strings(out)
	return out, nil
}

func syncWorktreeChangesToRoot(root, worktreePath string, relPaths []string) error {
	for _, rel := range relPaths {
		if err := syncRelativePath(worktreePath, root, rel); err != nil {
			return err
		}
	}
	return nil
}

func syncRelativePath(srcRoot, dstRoot, rel string) error {
	statuses, err := snapshotGitStatus(srcRoot)
	if err == nil {
		if status, ok := statuses[rel]; ok {
			return applyRelativePathStatus(srcRoot, dstRoot, rel, status)
		}
	}
	return applyRelativePathStatus(srcRoot, dstRoot, rel, "")
}

func applyRelativePathStatus(srcRoot, dstRoot, rel, status string) error {
	rel = filepath.Clean(filepath.FromSlash(rel))
	if rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("unsafe relative path %q", rel)
	}
	src := filepath.Join(srcRoot, rel)
	dst := filepath.Join(dstRoot, rel)

	if strings.Contains(status, "D") {
		if err := os.RemoveAll(dst); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.RemoveAll(dst); err != nil && !os.IsNotExist(err) {
				return err
			}
			return nil
		}
		return err
	}
	if info.IsDir() {
		return os.MkdirAll(dst, 0755)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode().Perm())
}
