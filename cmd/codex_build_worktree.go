package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func invokeCodexWorkerWithRuntimeProgress(
	ctx context.Context,
	invoker codex.WorkerInvoker,
	cfg codex.WorkerConfig,
	dispatch codex.WorkerDispatch,
	wave int,
) (codex.WorkerResult, error) {
	if progressInvoker, ok := invoker.(codex.ProgressAwareWorkerInvoker); ok {
		return progressInvoker.InvokeWithProgress(ctx, cfg, func(event codex.WorkerProgressEvent) {
			status := strings.TrimSpace(event.Status)
			switch status {
			case "running", "active":
				_ = updateCodexBuildDispatchRuntimeStatus(dispatch.WorkerName, "running", buildDispatchActiveSummary(dispatch, wave))
				emitBuildCeremonyWorkerRunning(dispatch, wave, event.Message)
				emitCodexDispatchWorkerRunning(dispatch, wave, event.Message)
			}
		})
	}
	return invoker.Invoke(ctx, cfg)
}

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

func updateWorktreeState(mutator func(*colony.ColonyState) error) error {
	if store == nil {
		return fmt.Errorf("no store initialized")
	}
	return store.UpdateFile("COLONY_STATE.json", func(existing []byte) ([]byte, error) {
		var state colony.ColonyState
		if len(existing) > 0 {
			if err := json.Unmarshal(existing, &state); err != nil {
				return nil, fmt.Errorf("unmarshal COLONY_STATE.json: %w", err)
			}
		}
		if err := mutator(&state); err != nil {
			return nil, err
		}
		encoded, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal COLONY_STATE.json: %w", err)
		}
		return append(encoded, '\n'), nil
	})
}

func applyObservedClaims(root string, baseline map[string]string, touched []string, result *codex.WorkerResult) {
	if result == nil || len(touched) == 0 {
		return
	}
	created := make([]string, 0, len(touched))
	modified := make([]string, 0, len(touched))
	tests := make([]string, 0, len(touched))
	for _, rel := range touched {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" {
			continue
		}
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil || info.IsDir() {
			continue
		}
		base := filepath.Base(rel)
		if isTestFile(base) {
			tests = append(tests, rel)
			continue
		}
		if _, existed := baseline[rel]; existed {
			modified = append(modified, rel)
		} else {
			created = append(created, rel)
		}
	}
	result.FilesCreated = uniqueSortedStrings(created)
	result.FilesModified = uniqueSortedStrings(modified)
	result.TestsWritten = uniqueSortedStrings(tests)
}

func collectRepoTouchedPaths(root string, baseline map[string]string, result codex.WorkerResult) ([]string, error) {
	current, err := snapshotGitStatus(root)
	if err != nil {
		return nil, err
	}
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

func dispatchCodexBuildWorkers(ctx context.Context, root string, phase colony.Phase, dispatches []codex.WorkerDispatch, invoker codex.WorkerInvoker, startedAt time.Time, parallelMode colony.ParallelMode) ([]codex.DispatchResult, error) {
	if parallelMode != colony.ModeWorktree {
		return dispatchCodexBuildWorkersInRepo(ctx, phase, dispatches, invoker, parallelMode)
	}
	if _, ok := invoker.(*codex.FakeInvoker); ok {
		return dispatchCodexBuildWorkersInRepo(ctx, phase, dispatches, invoker, parallelMode)
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
	var rootOpsMu sync.Mutex
	for _, wave := range waveNumbers {
		waveDispatches := waves[wave]
		emitBuildCeremonyWaveStart(phase, wave, waveDispatches, parallelMode)
		emitCodexBuildWaveProgress(phase, wave, waveDispatches, parallelMode)
		waveResults := make([]codex.DispatchResult, len(waveDispatches))
		var wg sync.WaitGroup
		for idx, dispatch := range waveDispatches {
			wg.Add(1)
			go func(i int, dispatch codex.WorkerDispatch) {
				defer wg.Done()

				if ctx.Err() != nil {
					waveResults[i] = codex.DispatchResult{
						WorkerName: dispatch.WorkerName,
						Status:     "timeout",
						Error:      ctx.Err(),
					}
					emitBuildCeremonyWorkerTimeout(dispatch, wave, ctx.Err())
					return
				}

				var session *buildWorktreeSession
				var baseline map[string]string
				var allocErr error

				rootOpsMu.Lock()
				session, allocErr = allocateBuildWorktree(root, phase.ID, dispatch, startedAt)
				if allocErr == nil {
					allocErr = updateBuildWorktreeStatus(session.Branch, colony.WorktreeInProgress)
				}
				if allocErr == nil {
					baseline, allocErr = snapshotWorktreeStatus(session.AbsPath)
				}
				if allocErr == nil {
					allocErr = updateCodexBuildDispatchRuntimeStatus(dispatch.WorkerName, "starting", workerDispatchSummary(dispatch))
				}
				rootOpsMu.Unlock()

				if allocErr != nil {
					if session != nil {
						rootOpsMu.Lock()
						_ = finalizeBuildWorktree(root, session, colony.WorktreeOrphaned)
						rootOpsMu.Unlock()
					}
					waveResults[i] = codex.DispatchResult{
						WorkerName: dispatch.WorkerName,
						Status:     "failed",
						Error:      allocErr,
					}
					emitBuildCeremonyWorkerFailed(dispatch, wave, allocErr)
					return
				}

				emitBuildCeremonyWorkerStarting(dispatch, wave)
				emitCodexBuildWorkerStarted(dispatch, wave)

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

				result, invokeErr := invokeCodexWorkerWithRuntimeProgress(ctx, invoker, cfg, dispatch, wave)
				dr := codex.DispatchResult{WorkerName: dispatch.WorkerName}
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
					} else {
						applyObservedClaims(session.AbsPath, baseline, touched, dr.WorkerResult)
						rootOpsMu.Lock()
						if syncErr := syncWorktreeChangesToRoot(root, session.AbsPath, touched); syncErr != nil {
							dr.Status = "failed"
							dr.Error = syncErr
							finalStatus = colony.WorktreeOrphaned
						} else if pheromoneResult, pheromoneErr := syncPheromoneStores(session.AbsPath, root, pheromoneSyncOptions{}); pheromoneErr != nil {
							dr.Status = "failed"
							dr.Error = pheromoneErr
							finalStatus = colony.WorktreeOrphaned
						} else if dr.WorkerResult != nil {
							syncSummary := formatPheromoneSyncSummary(pheromoneResult)
							if syncSummary != "" {
								if strings.TrimSpace(dr.WorkerResult.Summary) == "" {
									dr.WorkerResult.Summary = syncSummary
								} else {
									dr.WorkerResult.Summary = strings.TrimSpace(dr.WorkerResult.Summary) + " " + syncSummary
								}
							}
							if tracer != nil {
								var state colony.ColonyState
								if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr == nil && state.RunID != nil {
									_ = tracer.LogArtifact(*state.RunID, "worktree.merge", map[string]interface{}{
										"worker":       dispatch.WorkerName,
										"files_synced": len(touched),
										"pheromones":   syncSummary,
									})
								}
							}
						}
						rootOpsMu.Unlock()
					}
				}

				rootOpsMu.Lock()
				if cleanupErr := finalizeBuildWorktree(root, session, finalStatus); cleanupErr != nil && dr.Error == nil {
					dr.Status = "failed"
					dr.Error = cleanupErr
				}
				if dr.Status == "" {
					dr.Status = "failed"
				}
				statusErr := updateCodexBuildDispatchRuntimeStatus(dispatch.WorkerName, dr.Status, buildDispatchResultSummary(dispatch, dr))
				rootOpsMu.Unlock()
				if statusErr != nil {
					dr.Status = "failed"
					dr.Error = fmt.Errorf("complete worker %s: %w", dispatch.WorkerName, statusErr)
				}

				emitBuildCeremonyWorkerFinished(dispatch, dr)
				emitCodexBuildWorkerFinished(dispatch, dr)
				waveResults[i] = dr
			}(idx, dispatch)
		}
		wg.Wait()
		emitBuildCeremonyWaveEnd(phase, wave, waveResults)
		results = append(results, waveResults...)
	}
	return results, nil
}

func dispatchCodexBuildWorkersInRepo(ctx context.Context, phase colony.Phase, dispatches []codex.WorkerDispatch, invoker codex.WorkerInvoker, parallelMode colony.ParallelMode) ([]codex.DispatchResult, error) {
	waves := codex.GroupByWave(dispatches)
	waveNumbers := make([]int, 0, len(waves))
	for wave := range waves {
		waveNumbers = append(waveNumbers, wave)
	}
	sort.Ints(waveNumbers)

	var results []codex.DispatchResult
	for _, wave := range waveNumbers {
		waveDispatches := waves[wave]
		emitBuildCeremonyWaveStart(phase, wave, waveDispatches, parallelMode)
		emitCodexBuildWaveProgress(phase, wave, waveDispatches, parallelMode)
		waveResults := make([]codex.DispatchResult, 0, len(waveDispatches))
		for _, dispatch := range waveDispatches {
			if ctx.Err() != nil {
				dr := codex.DispatchResult{
					WorkerName: dispatch.WorkerName,
					Status:     "timeout",
					Error:      ctx.Err(),
				}
				emitBuildCeremonyWorkerTimeout(dispatch, wave, ctx.Err())
				waveResults = append(waveResults, dr)
				results = append(results, dr)
				continue
			}

			baseline, baselineErr := snapshotGitStatus(dispatch.Root)
			if err := updateCodexBuildDispatchRuntimeStatus(dispatch.WorkerName, "starting", workerDispatchSummary(dispatch)); err != nil {
				return nil, fmt.Errorf("mark worker starting for %s: %w", dispatch.WorkerName, err)
			}
			emitBuildCeremonyWorkerStarting(dispatch, wave)
			emitCodexBuildWorkerStarted(dispatch, wave)

			cfg := codex.WorkerConfig{
				AgentName:        dispatch.AgentName,
				AgentTOMLPath:    dispatch.AgentTOMLPath,
				Caste:            dispatch.Caste,
				WorkerName:       dispatch.WorkerName,
				TaskID:           dispatch.TaskID,
				TaskBrief:        dispatch.TaskBrief,
				ContextCapsule:   dispatch.ContextCapsule,
				Root:             dispatch.Root,
				Timeout:          dispatch.Timeout,
				SkillSection:     dispatch.SkillSection,
				PheromoneSection: dispatch.PheromoneSection,
			}

			result, err := invokeCodexWorkerWithRuntimeProgress(ctx, invoker, cfg, dispatch, wave)
			dr := codex.DispatchResult{WorkerName: dispatch.WorkerName}
			if err != nil {
				dr.Status = "failed"
				dr.Error = err
			} else {
				dr.Status = result.Status
				dr.WorkerResult = &result
				if result.Error != nil {
					dr.Error = result.Error
				}
				if baselineErr == nil && dr.Status == "completed" {
					if touched, touchErr := collectRepoTouchedPaths(dispatch.Root, baseline, result); touchErr == nil {
						applyObservedClaims(dispatch.Root, baseline, touched, dr.WorkerResult)
					}
				}
			}
			if dr.Status == "" {
				dr.Status = "failed"
			}
			if err := updateCodexBuildDispatchRuntimeStatus(dispatch.WorkerName, dr.Status, buildDispatchResultSummary(dispatch, dr)); err != nil {
				return nil, fmt.Errorf("complete worker %s: %w", dispatch.WorkerName, err)
			}
			emitBuildCeremonyWorkerFinished(dispatch, dr)
			emitCodexBuildWorkerFinished(dispatch, dr)
			waveResults = append(waveResults, dr)
			results = append(results, dr)
		}
		emitBuildCeremonyWaveEnd(phase, wave, waveResults)
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

	// Clean up any leftover path from a previous failed allocation
	if _, err := os.Stat(absPath); err == nil {
		if rmErr := os.RemoveAll(absPath); rmErr != nil {
			return nil, fmt.Errorf("worktree path %s already exists and cannot be removed: %v", absPath, rmErr)
		}
	}

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
	return updateWorktreeState(func(state *colony.ColonyState) error {
		state.Worktrees = append(state.Worktrees, entry)
		return nil
	})
}

func updateBuildWorktreeStatus(branch string, status colony.WorktreeStatus) error {
	return updateWorktreeState(func(state *colony.ColonyState) error {
		now := time.Now().UTC().Format(time.RFC3339)
		for i := range state.Worktrees {
			if state.Worktrees[i].Branch != branch {
				continue
			}
			state.Worktrees[i].Status = status
			state.Worktrees[i].UpdatedAt = now
			return nil
		}
		return fmt.Errorf("worktree %q not tracked in colony state", branch)
	})
}

func finalizeBuildWorktree(root string, session *buildWorktreeSession, status colony.WorktreeStatus) error {
	if session == nil {
		return nil
	}
	if err := updateBuildWorktreeStatus(session.Branch, status); err != nil {
		return err
	}
	if err := removeGitWorktree(root, session.AbsPath, session.Branch); err != nil {
		// Removal failed — mark as orphaned and propagate the error
		_ = updateBuildWorktreeStatus(session.Branch, colony.WorktreeOrphaned)
		return err
	}
	return nil
}

func removeGitWorktree(root, absPath, branch string) error {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()

	var errs []string
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "worktree", "remove", absPath, "--force").CombinedOutput(); err != nil {
		errs = append(errs, fmt.Sprintf("worktree remove: %v (output: %s)", err, string(out)))
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "worktree", "prune").CombinedOutput(); err != nil {
		errs = append(errs, fmt.Sprintf("worktree prune: %v (output: %s)", err, string(out)))
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "-D", branch).CombinedOutput(); err != nil {
		errs = append(errs, fmt.Sprintf("branch delete: %v (output: %s)", err, string(out)))
	}
	if len(errs) > 0 {
		return fmt.Errorf("worktree cleanup failed: %s", strings.Join(errs, "; "))
	}
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

// cleanupBuildWorktrees removes any unfinalized worktrees for a phase.
// It scans the colony state for worktree entries with Allocated or InProgress status,
// attempts to remove them, and updates their status to Orphaned on failure.
func cleanupBuildWorktrees(phaseID int) (cleaned int, orphaned int, err error) {
	var state colony.ColonyState
	if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr != nil {
		return 0, 0, loadErr
	}

	root := storage.ResolveAetherRoot(context.Background())
	var remaining []colony.WorktreeEntry
	for _, entry := range state.Worktrees {
		if entry.Phase != phaseID {
			remaining = append(remaining, entry)
			continue
		}
		if entry.Status != colony.WorktreeAllocated && entry.Status != colony.WorktreeInProgress {
			remaining = append(remaining, entry)
			continue
		}

		absPath := filepath.Join(root, entry.Path)
		// If the path doesn't exist on disk, just remove the stale entry
		if _, statErr := os.Stat(absPath); statErr != nil && os.IsNotExist(statErr) {
			cleaned++
			continue
		}
		if removeErr := removeGitWorktree(root, absPath, entry.Branch); removeErr != nil {
			entry.Status = colony.WorktreeOrphaned
			remaining = append(remaining, entry)
			orphaned++
		} else {
			cleaned++
			// Don't append — entry is removed
		}
	}

	state.Worktrees = remaining
	if saveErr := store.SaveJSON("COLONY_STATE.json", state); saveErr != nil {
		return cleaned, orphaned, saveErr
	}
	return cleaned, orphaned, nil
}

// gcOrphanedWorktrees scans all tracked worktrees and cleans up any that are
// stale (Allocated, InProgress, or Orphaned status). It returns counts of
// cleaned and orphaned worktrees. Unlike cleanupBuildWorktrees, it operates
// across all phases and does not filter by phase ID.
func gcOrphanedWorktrees() (cleaned int, orphaned int, err error) {
	var state colony.ColonyState
	if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr != nil {
		return 0, 0, loadErr
	}

	root := storage.ResolveAetherRoot(context.Background())
	var remaining []colony.WorktreeEntry
	for _, entry := range state.Worktrees {
		if entry.Status != colony.WorktreeAllocated && entry.Status != colony.WorktreeInProgress && entry.Status != colony.WorktreeOrphaned {
			remaining = append(remaining, entry)
			continue
		}

		absPath := filepath.Join(root, entry.Path)
		// If the path doesn't exist on disk, just remove the stale entry
		if _, statErr := os.Stat(absPath); statErr != nil && os.IsNotExist(statErr) {
			cleaned++
			continue
		}
		if removeErr := removeGitWorktree(root, absPath, entry.Branch); removeErr != nil {
			entry.Status = colony.WorktreeOrphaned
			remaining = append(remaining, entry)
			orphaned++
		} else {
			cleaned++
			// Don't append — entry is removed
		}
	}

	state.Worktrees = remaining
	if saveErr := store.SaveJSON("COLONY_STATE.json", state); saveErr != nil {
		return cleaned, orphaned, saveErr
	}
	return cleaned, orphaned, nil
}
