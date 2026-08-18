package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	if err := beginDirectBuildWorkerRun(dispatch, invoker); err != nil {
		return codex.WorkerResult{}, err
	}
	if progressInvoker, ok := invoker.(codex.ProgressAwareWorkerInvoker); ok {
		return progressInvoker.InvokeWithProgress(ctx, cfg, func(event codex.WorkerProgressEvent) {
			if event.ProcessID > 0 && dispatch.ExecutionBinding != nil {
				_ = recordBuildAttemptWorkerProcess(dispatch.Phase, *dispatch.ExecutionBinding, dispatch.ProviderRunID, event.ProcessID)
			}
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

// declaredPathsForTask computes the repo-relative paths a task claims
// ownership of before workers run. Declarations come from task-level
// criterion evidence artifacts and from hints that are exactly one
// repo-relative file path. Runtime and companion files under .aether/ never
// participate in ownership.
func declaredPathsForTask(task colony.Task) []string {
	paths := make([]string, 0, len(task.Hints)+1)
	add := func(raw string) {
		normalized, err := normalizeCriterionArtifactPath(raw)
		if err != nil {
			return
		}
		if strings.HasPrefix(normalized, ".aether/") {
			return
		}
		paths = append(paths, normalized)
	}
	for _, requirement := range task.EvidenceRequirements {
		for _, artifact := range requirement.Artifacts {
			add(artifact)
		}
	}
	for _, hint := range task.Hints {
		hint = strings.TrimSpace(hint)
		if hint == "" || strings.ContainsAny(hint, " \t") {
			continue
		}
		if looksLikeFile(hint) || filePathPattern.MatchString(hint) || bareFileNameWithExtension(hint) {
			add(hint)
		}
	}
	return uniqueSortedStrings(paths)
}

var bareFileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+\.[a-zA-Z0-9]+$`)

func bareFileNameWithExtension(value string) bool {
	return bareFileNamePattern.MatchString(value) && !strings.Contains(value, "/")
}

// validateDeclaredWorktreeOwnership rejects a worktree-mode build before any
// worker runs when two tasks in the same wave declare the same path. Parallel
// claims on one path cannot reconcile, so the conflict is surfaced while it is
// still cheap. Declared overlaps across waves are allowed: waves execute
// sequentially and later worktrees inherit the earlier waves' synced output.
func validateDeclaredWorktreeOwnership(dispatches []codex.WorkerDispatch) error {
	declaredByWave := map[int]map[string]string{}
	for _, dispatch := range dispatches {
		for _, path := range dispatch.DeclaredPaths {
			owners := declaredByWave[dispatch.Wave]
			if owners == nil {
				owners = map[string]string{}
				declaredByWave[dispatch.Wave] = owners
			}
			if previous, ok := owners[path]; ok && previous != dispatch.TaskID {
				return fmt.Errorf("worktree declared ownership conflict: wave %d tasks %s and %s both declare %s; declare disjoint paths, move the tasks into different waves, or run with --parallel-mode in-repo", dispatch.Wave, previous, dispatch.TaskID, path)
			}
			owners[path] = dispatch.TaskID
		}
	}
	return nil
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

func dispatchCodexBuildWorkers(ctx context.Context, root string, phase colony.Phase, dispatches []codex.WorkerDispatch, invoker codex.WorkerInvoker, startedAt time.Time, parallelMode colony.ParallelMode, cb *CircuitBreaker) ([]codex.DispatchResult, error) {
	return dispatchCodexBuildWorkersWithReconciliation(ctx, root, phase, dispatches, invoker, startedAt, parallelMode, cb)
}

// worktreeWaveOutcome captures everything a worker goroutine produced. No
// result is synced, journaled, or finalized until the whole wave's ownership
// decision has been made.
type worktreeWaveOutcome struct {
	dispatch codex.WorkerDispatch
	result   codex.DispatchResult
	session  *buildWorktreeSession
	touched  []string
}

func dispatchCodexBuildWorkersWithReconciliation(ctx context.Context, root string, phase colony.Phase, dispatches []codex.WorkerDispatch, invoker codex.WorkerInvoker, startedAt time.Time, parallelMode colony.ParallelMode, cb *CircuitBreaker) ([]codex.DispatchResult, error) {
	if parallelMode != colony.ModeWorktree {
		return dispatchCodexBuildWorkersInRepo(ctx, phase, dispatches, invoker, parallelMode, cb)
	}
	if _, ok := invoker.(*codex.FakeInvoker); ok {
		return dispatchCodexBuildWorkersInRepo(ctx, phase, dispatches, invoker, parallelMode, cb)
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
		outcomes := make([]*worktreeWaveOutcome, len(waveDispatches))
		cb.Reset() // Per D-06: per-wave reset
		var wg sync.WaitGroup
		for idx, dispatch := range waveDispatches {
			wg.Add(1)
			go func(i int, dispatch codex.WorkerDispatch) {
				defer wg.Done()
				outcome := &worktreeWaveOutcome{dispatch: dispatch}
				outcomes[i] = outcome

				if ctx.Err() != nil {
					outcome.result = codex.DispatchResult{
						WorkerName: dispatch.WorkerName,
						Status:     "timeout",
						Error:      ctx.Err(),
					}
					emitBuildCeremonyWorkerTimeout(dispatch, wave, ctx.Err())
					return
				}

				if !cb.Allow(dispatch.WorkerName) {
					peer := findSameCastePeer(waveDispatches, dispatch, cb)
					if peer != nil {
						emitCircuitBreakerRedistributed(phase, wave, dispatch.WorkerName, peer.WorkerName)
						dispatch = *peer
						outcome.dispatch = dispatch
					} else {
						emitCircuitBreakerNoPeer(phase, wave, dispatch.WorkerName)
						outcome.result = codex.DispatchResult{
							WorkerName: dispatch.WorkerName,
							Status:     "failed",
							Error:      fmt.Errorf("circuit breaker tripped, no same-caste peer for redistribution"),
						}
						emitBuildCeremonyWorkerFailed(dispatch, wave, outcome.result.Error)
						return
					}
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
					outcome.result = codex.DispatchResult{
						WorkerName: dispatch.WorkerName,
						Status:     "failed",
						Error:      allocErr,
					}
					emitBuildCeremonyWorkerFailed(dispatch, wave, allocErr)
					return
				}
				outcome.session = session

				emitBuildCeremonyWorkerStarting(dispatch, wave)
				emitCodexBuildWorkerStarted(dispatch, wave)

				cfg := codex.WorkerConfig{
					AgentName:         dispatch.AgentName,
					AgentTOMLPath:     dispatch.AgentTOMLPath,
					Caste:             dispatch.Caste,
					WorkerName:        dispatch.WorkerName,
					TaskID:            dispatch.TaskID,
					TaskBrief:         dispatch.TaskBrief,
					ContextCapsule:    dispatch.ContextCapsule,
					Root:              session.AbsPath,
					TrackingRoot:      root,
					Timeout:           dispatch.Timeout,
					SkillSection:      dispatch.SkillSection,
					PheromoneSection:  dispatch.PheromoneSection,
					HandoffSection:    dispatch.HandoffSection,
					PermissionProfile: dispatch.PermissionProfile,
					ExecutionBinding:  dispatch.ExecutionBinding,
					ProviderRunID:     dispatch.ProviderRunID,
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

				if dr.Status == "completed" && dr.WorkerResult != nil {
					touched, touchErr := collectWorktreeTouchedPaths(session.AbsPath, baseline, *dr.WorkerResult)
					if touchErr != nil {
						dr.Status = "failed"
						dr.Error = touchErr
					} else {
						applyObservedClaims(session.AbsPath, baseline, touched, dr.WorkerResult)
						outcome.touched = touched
					}
				}
				outcome.result = dr
			}(idx, dispatch)
		}
		wg.Wait()
		waveResults := reconcileWorktreeWave(root, phase, wave, outcomes, cb)
		emitBuildCeremonyWaveEnd(phase, wave, waveResults)
		results = append(results, waveResults...)
	}
	return results, nil
}

// reconcileWorktreeWave makes one atomic ownership decision for a completed
// wave. A conflict-free wave syncs every accepted worker in deterministic
// dispatch order. Any conflict rejects the entire wave: nothing syncs into the
// root checkout, conflicting workers fail with the exact paths and owners, and
// conflict-free workers are blocked rather than silently accepted, so the root
// never carries a partial wave. Terminal results are journaled only after the
// decision, so the attempt journal always matches the reconciled outcome.
func reconcileWorktreeWave(root string, phase colony.Phase, wave int, outcomes []*worktreeWaveOutcome, cb *CircuitBreaker) []codex.DispatchResult {
	conflicts, conflictWorkers := detectWorktreeWaveConflicts(outcomes)

	accepted := map[int]bool{}
	if len(conflicts) == 0 {
		for i, outcome := range outcomes {
			if outcome != nil && outcome.result.Status == "completed" && outcome.result.WorkerResult != nil && outcome.session != nil {
				accepted[i] = true
			}
		}
	}

	rejectionReason := ""
	if len(conflicts) > 0 {
		rejectionReason = fmt.Sprintf("worktree wave reconciliation conflict: %s", strings.Join(conflicts, ", "))
	}

	for i, outcome := range outcomes {
		if outcome == nil {
			continue
		}
		dr := outcome.result
		session := outcome.session
		preserveWorktree := false

		switch {
		case accepted[i]:
			if syncErr := syncWorktreeChangesToRoot(root, session.AbsPath, outcome.touched); syncErr != nil {
				dr.Status = "failed"
				dr.Error = syncErr
				preserveWorktree = true
			} else if pheromoneResult, pheromoneErr := syncPheromoneStores(session.AbsPath, root, pheromoneSyncOptions{}); pheromoneErr != nil {
				dr.Status = "failed"
				dr.Error = pheromoneErr
				preserveWorktree = true
			} else {
				appendPheromoneSyncSummary(dr.WorkerResult, pheromoneResult)
				logWorktreeMergeTrace(outcome.dispatch, outcome.touched, pheromoneResult)
			}
		case len(conflicts) > 0 && dr.Status == "completed" && outcome.session != nil:
			preserveWorktree = true
			if conflictWorkers[i] {
				dr.Status = "failed"
				dr.Error = fmt.Errorf("%s", rejectionReason)
			} else {
				dr.Status = "blocked"
				dr.Error = fmt.Errorf("wave reconciliation rejected the wave before this worker's output could sync: %s", rejectionReason)
			}
		case dr.Status != "completed" && session != nil:
			preserveWorktree = true
		}

		if session != nil {
			if preserveWorktree {
				if statusErr := updateBuildWorktreeStatus(session.Branch, colony.WorktreeOrphaned); statusErr != nil && dr.Error == nil {
					dr.Status = "failed"
					dr.Error = statusErr
				}
			} else if accepted[i] || dr.Status == "completed" {
				finalStatus := colony.WorktreeMerged
				if dr.Status != "completed" {
					finalStatus = colony.WorktreeOrphaned
				}
				if cleanupErr := finalizeBuildWorktree(root, session, finalStatus); cleanupErr != nil && dr.Error == nil {
					dr.Status = "failed"
					dr.Error = cleanupErr
				}
			}
		}
		if dr.Status == "" {
			dr.Status = "failed"
		}

		if dr.Status == "completed" {
			cb.RecordSuccess(outcome.dispatch.WorkerName)
		} else if cb.RecordFailure(outcome.dispatch.WorkerName) {
			cb.emitCircuitBreakerTripped(phase, wave, outcome.dispatch.WorkerName)
		}
		if statusErr := updateCodexBuildDispatchRuntimeStatus(outcome.dispatch.WorkerName, dr.Status, buildDispatchResultSummary(outcome.dispatch, dr)); statusErr != nil {
			dr.Status = "failed"
			dr.Error = fmt.Errorf("complete worker %s: %w", outcome.dispatch.WorkerName, statusErr)
		}
		if journalErr := recordDirectBuildWorkerTerminal(outcome.dispatch, dr); journalErr != nil {
			dr.Status = "failed"
			dr.Error = fmt.Errorf("journal terminal worker %s: %w", outcome.dispatch.WorkerName, journalErr)
			_ = updateCodexBuildDispatchRuntimeStatus(outcome.dispatch.WorkerName, dr.Status, buildDispatchResultSummary(outcome.dispatch, dr))
		}

		emitBuildCeremonyWorkerFinished(outcome.dispatch, dr)
		emitCodexBuildWorkerFinished(outcome.dispatch, dr)
		outcome.result = dr
	}

	results := make([]codex.DispatchResult, 0, len(outcomes))
	for _, outcome := range outcomes {
		if outcome == nil {
			continue
		}
		results = append(results, outcome.result)
	}
	return results
}

// detectWorktreeWaveConflicts finds every same-wave ownership violation among
// completed workers: a path touched by a worker whose task did not declare it
// when another same-wave task did, or a path produced by more than one worker
// with no declared owner to arbitrate. Returns human-readable conflict
// descriptions and the set of outcome indexes involved.
func detectWorktreeWaveConflicts(outcomes []*worktreeWaveOutcome) ([]string, map[int]bool) {
	declared := map[string]string{}
	for _, outcome := range outcomes {
		if outcome == nil {
			continue
		}
		for _, path := range outcome.dispatch.DeclaredPaths {
			if _, ok := declared[path]; !ok {
				declared[path] = outcome.dispatch.TaskID
			}
		}
	}

	touchedBy := map[string][]int{}
	for i, outcome := range outcomes {
		if outcome == nil || outcome.result.Status != "completed" || outcome.result.WorkerResult == nil || outcome.session == nil {
			continue
		}
		for _, path := range outcome.touched {
			touchedBy[path] = append(touchedBy[path], i)
		}
	}

	conflictWorkers := map[int]bool{}
	conflicts := make([]string, 0)
	for _, path := range uniqueSortedStrings(mapKeys(touchedBy)) {
		idxs := touchedBy[path]
		if len(idxs) > 1 {
			names := make([]string, 0, len(idxs))
			for _, idx := range idxs {
				names = append(names, outcomes[idx].dispatch.WorkerName)
				conflictWorkers[idx] = true
			}
			if owner, ok := declared[path]; ok {
				conflicts = append(conflicts, fmt.Sprintf("%s produced by multiple workers (%s) but declared by task %s", path, strings.Join(names, ", "), owner))
			} else {
				conflicts = append(conflicts, fmt.Sprintf("%s produced by multiple workers (%s) with no declared owner", path, strings.Join(names, ", ")))
			}
			continue
		}
		idx := idxs[0]
		if owner, ok := declared[path]; ok && owner != outcomes[idx].dispatch.TaskID {
			conflicts = append(conflicts, fmt.Sprintf("%s touched by worker %s (task %s) but declared by task %s", path, outcomes[idx].dispatch.WorkerName, outcomes[idx].dispatch.TaskID, owner))
			conflictWorkers[idx] = true
		}
	}
	return conflicts, conflictWorkers
}

func mapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func appendPheromoneSyncSummary(result *codex.WorkerResult, pheromoneResult pheromoneSyncResult) {
	if result == nil {
		return
	}
	syncSummary := formatPheromoneSyncSummary(pheromoneResult)
	if syncSummary == "" {
		return
	}
	if strings.TrimSpace(result.Summary) == "" {
		result.Summary = syncSummary
	} else {
		result.Summary = strings.TrimSpace(result.Summary) + " " + syncSummary
	}
}

func logWorktreeMergeTrace(dispatch codex.WorkerDispatch, touched []string, pheromoneResult pheromoneSyncResult) {
	if tracer == nil {
		return
	}
	var state colony.ColonyState
	if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr == nil && state.RunID != nil {
		_ = tracer.LogArtifact(*state.RunID, "worktree.merge", map[string]interface{}{
			"worker":       dispatch.WorkerName,
			"files_synced": len(touched),
			"pheromones":   formatPheromoneSyncSummary(pheromoneResult),
		})
	}
}

func dispatchCodexBuildWorkersInRepo(ctx context.Context, phase colony.Phase, dispatches []codex.WorkerDispatch, invoker codex.WorkerInvoker, parallelMode colony.ParallelMode, cb *CircuitBreaker) ([]codex.DispatchResult, error) {
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
		cb.Reset() // Per D-06: per-wave reset
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

			if !cb.Allow(dispatch.WorkerName) {
				peer := findSameCastePeer(waveDispatches, dispatch, cb)
				if peer != nil {
					emitCircuitBreakerRedistributed(phase, wave, dispatch.WorkerName, peer.WorkerName)
					dispatch = *peer
				} else {
					emitCircuitBreakerNoPeer(phase, wave, dispatch.WorkerName)
					dr := codex.DispatchResult{
						WorkerName: dispatch.WorkerName,
						Status:     "failed",
						Error:      fmt.Errorf("circuit breaker tripped, no same-caste peer for redistribution"),
					}
					emitBuildCeremonyWorkerFailed(dispatch, wave, dr.Error)
					waveResults = append(waveResults, dr)
					results = append(results, dr)
					continue
				}
			}

			baseline, baselineErr := snapshotGitStatus(dispatch.Root)
			if err := updateCodexBuildDispatchRuntimeStatus(dispatch.WorkerName, "starting", workerDispatchSummary(dispatch)); err != nil {
				return nil, fmt.Errorf("mark worker starting for %s: %w", dispatch.WorkerName, err)
			}
			emitBuildCeremonyWorkerStarting(dispatch, wave)
			emitCodexBuildWorkerStarted(dispatch, wave)

			cfg := codex.WorkerConfig{
				AgentName:         dispatch.AgentName,
				AgentTOMLPath:     dispatch.AgentTOMLPath,
				Caste:             dispatch.Caste,
				WorkerName:        dispatch.WorkerName,
				TaskID:            dispatch.TaskID,
				TaskBrief:         dispatch.TaskBrief,
				ContextCapsule:    dispatch.ContextCapsule,
				Root:              dispatch.Root,
				TrackingRoot:      dispatch.TrackingRoot,
				Timeout:           dispatch.Timeout,
				SkillSection:      dispatch.SkillSection,
				PheromoneSection:  dispatch.PheromoneSection,
				HandoffSection:    dispatch.HandoffSection,
				PermissionProfile: dispatch.PermissionProfile,
				ExecutionBinding:  dispatch.ExecutionBinding,
				ProviderRunID:     dispatch.ProviderRunID,
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
			// Record result with circuit breaker
			if dr.Status == "completed" {
				cb.RecordSuccess(dispatch.WorkerName)
			} else if cb.RecordFailure(dispatch.WorkerName) {
				cb.emitCircuitBreakerTripped(phase, wave, dispatch.WorkerName)
			}
			if statusErr := updateCodexBuildDispatchRuntimeStatus(dispatch.WorkerName, dr.Status, buildDispatchResultSummary(dispatch, dr)); statusErr != nil {
				dr.Status = "failed"
				dr.Error = fmt.Errorf("complete worker %s: %w", dispatch.WorkerName, statusErr)
			}
			if journalErr := recordDirectBuildWorkerTerminal(dispatch, dr); journalErr != nil {
				dr.Status = "failed"
				dr.Error = fmt.Errorf("journal terminal worker %s: %w", dispatch.WorkerName, journalErr)
				_ = updateCodexBuildDispatchRuntimeStatus(dispatch.WorkerName, dr.Status, buildDispatchResultSummary(dispatch, dr))
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
	if store == nil {
		return 0, 0, fmt.Errorf("no store initialized")
	}
	var state colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &state, func() error {
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
		return nil
	}); err != nil {
		return 0, 0, err
	}
	return cleaned, orphaned, nil
}

// gcOrphanedWorktrees scans all tracked worktrees (Allocated, InProgress, or
// Orphaned status) and decides, per entry, whether it is safe to forget about
// without losing work. It returns counts of cleaned (stale state entries
// whose path no longer exists on disk) and preserved (everything else — kept
// on purpose, never deleted). Unlike cleanupBuildWorktrees, it operates
// across all phases and does not filter by phase ID.
//
// This function runs from three recovery paths — resume
// (cmd/session_flow_cmds.go), continue (cmd/codex_continue.go) and init
// (cmd/init_cmd.go) — all three of which a user reaches while trying to
// recover from something going wrong. A recovery path must never be the
// thing that destroys what is being recovered (D-01,
// .planning/phases/187-crash-safe-worktrees-ecosystem-neutrality/187-CONTEXT.md).
// It therefore NEVER calls removeGitWorktree. Destruction now lives only in
// the explicitly named, operator-invoked `worktree-reap` command
// (cmd/worktree_reap.go).
//
// No phase filter is applied here deliberately. CONTEXT.md notes this
// function has none today, which lets resuming phase 5 act on a stalled
// phase 2 worktree — but since this function no longer destroys anything,
// the blast radius that made the missing filter dangerous is gone. Adding a
// filter now would instead hide older worktrees from the report, which is
// how ten branches stranded unnoticed between May and July 2026
// (.planning/WORKTREE-BRANCH-AUDIT-2026-07-27.md). Each entry's Phase is
// included in the preservation report so a user resuming phase 5 who sees a
// phase 2 worktree mentioned understands what is meant.
// detectOrphanedWorktrees's current-phase blind spot (cmd/codex_build.go) is
// a separate, explicitly deferred concern (CONTEXT.md <deferred>) and is not
// touched here.
func gcOrphanedWorktrees() (cleaned int, preserved int, err error) {
	if store == nil {
		return 0, 0, fmt.Errorf("no store initialized")
	}
	if _, statErr := os.Stat(filepath.Join(store.BasePath(), "COLONY_STATE.json")); statErr != nil {
		if os.IsNotExist(statErr) {
			return 0, 0, nil
		}
		return 0, 0, statErr
	}
	var state colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &state, func() error {
		root := storage.ResolveAetherRoot(context.Background())
		var remaining []colony.WorktreeEntry
		for _, entry := range state.Worktrees {
			if entry.Status != colony.WorktreeAllocated && entry.Status != colony.WorktreeInProgress && entry.Status != colony.WorktreeOrphaned {
				remaining = append(remaining, entry)
				continue
			}

			safety := worktreeDestructionSafety(root, entry)

			if !safety.Safe {
				// Dirty, unmerged, or undeterminable — preserve and keep the
				// entry, regardless of whether preservation itself errors. A
				// failed stash is even more reason not to delete.
				_, detail, preserveErr := preserveWorktreeWork(root, entry, safety)
				if preserveErr != nil {
					detail = fmt.Sprintf("could not stash automatically (%v); branch %s was left alone", preserveErr, entry.Branch)
				}
				reportWorktreePreservation(safety, fmt.Sprintf("phase %d: %s", entry.Phase, detail))
				entry.Status = colony.WorktreeOrphaned
				remaining = append(remaining, entry)
				preserved++
				continue
			}

			// safety.Safe is true. Either the path is gone (nothing to keep,
			// safety.Reason is "worktree path no longer exists on disk") or
			// the worktree is clean and fully merged (something to keep, but
			// only an operator invoking worktree-reap may remove it).
			// worktreeDestructionSafety already resolved and stat'd the
			// absolute path (safety.Path); re-derive from that rather than
			// re-stat'ing entry.Path (which may be relative) a second time.
			if _, statErr := os.Stat(safety.Path); statErr == nil {
				// Path exists — clean and merged. D-01's implication is
				// explicit: destruction is deferred to an explicit, named
				// operator-invoked command, never to an automatic cleanup
				// running inside resume, continue or init.
				detail := fmt.Sprintf("phase %d: %s — safe to remove, run: aether worktree-reap --force --branch %s", entry.Phase, safety.Reason, entry.Branch)
				reportWorktreePreservation(safety, detail)
				entry.Status = colony.WorktreeOrphaned
				remaining = append(remaining, entry)
				preserved++
				continue
			}

			// Path no longer exists on disk — a missing directory holds
			// nothing to preserve. Drop the stale entry.
			cleaned++
		}

		state.Worktrees = remaining
		return nil
	}); err != nil {
		return 0, 0, err
	}
	return cleaned, preserved, nil
}

// ---------------------------------------------------------------------------
// Worktree Merge-Back (extracted for reuse by build-finalize and continue)
// ---------------------------------------------------------------------------

// mergePhaseWorktrees merges all unmerged worktree branches for the given phase.
// It returns a summary of merged and failed branches.
func mergePhaseWorktrees(phaseNum int) (merged []string, failed []string, err error) {
	if store == nil {
		return nil, nil, fmt.Errorf("no store initialized")
	}

	var state colony.ColonyState
	if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr != nil {
		return nil, nil, fmt.Errorf("load colony state: %w", loadErr)
	}

	root := resolveAetherRoot()
	for _, entry := range state.Worktrees {
		if entry.Phase != phaseNum {
			continue
		}
		if entry.Status == colony.WorktreeMerged {
			continue
		}

		wtAbsPath := entry.Path
		if !filepath.IsAbs(wtAbsPath) {
			wtAbsPath = filepath.Join(root, wtAbsPath)
		}

		// Gate 1: run tests in worktree
		testCtx, testCancel := context.WithTimeout(context.Background(), BuildTimeout)
		testCmd := exec.CommandContext(testCtx, "go", "test", "./...")
		testCmd.Dir = wtAbsPath
		_, testErr := testCmd.CombinedOutput()
		testCancel()
		if testErr != nil {
			failed = append(failed, fmt.Sprintf("%s (tests failed)", entry.Branch))
			continue
		}

		// Gate 2: clash detection
		clashes, clashErr := checkClashesForWorktree(wtAbsPath, entry.Branch)
		if clashErr != nil {
			failed = append(failed, fmt.Sprintf("%s (clash detection error)", entry.Branch))
			continue
		}
		if len(clashes) > 0 {
			failed = append(failed, fmt.Sprintf("%s (clash: %s)", entry.Branch, strings.Join(clashes, ", ")))
			continue
		}

		// Merge
		gitCtx, gitCancel := context.WithTimeout(context.Background(), GitTimeout)
		coOut, coErr := exec.CommandContext(gitCtx, "git", "-C", root, "checkout", "main").CombinedOutput()
		if coErr != nil {
			coOut2, coErr2 := exec.CommandContext(gitCtx, "git", "-C", root, "checkout", "master").CombinedOutput()
			if coErr2 != nil {
				gitCancel()
				failed = append(failed, fmt.Sprintf("%s (checkout failed: %s / %s)", entry.Branch, string(coOut), string(coOut2)))
				continue
			}
		}

		mergeOut, mergeErr := exec.CommandContext(gitCtx, "git", "-C", root, "merge", entry.Branch).CombinedOutput()
		gitCancel()
		if mergeErr != nil {
			failed = append(failed, fmt.Sprintf("%s (merge failed: %s)", entry.Branch, string(mergeOut)))
			continue
		}

		merged = append(merged, entry.Branch)
	}

	return merged, failed, nil
}

// detectOrphanedWorktrees finds worktree branches that belong to phases other
// than the current one and are not yet merged.
func detectOrphanedWorktrees(currentPhase int) []colony.WorktreeEntry {
	if store == nil {
		return nil
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return nil
	}
	var orphans []colony.WorktreeEntry
	for _, entry := range state.Worktrees {
		if entry.Phase == currentPhase {
			continue
		}
		if entry.Status == colony.WorktreeMerged {
			continue
		}
		orphans = append(orphans, entry)
	}
	return orphans
}
