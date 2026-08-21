package cmd

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func spawnTreeDispatchObserver(spawnTree *agent.SpawnTree, activePrefix string) codex.DispatchObserver {
	if spawnTree == nil {
		return nil
	}
	return func(event codex.DispatchLifecycleEvent) {
		status := strings.TrimSpace(event.Status)
		if status == "" {
			return
		}

		summary := dispatchLifecycleSummary(event, activePrefix)
		_ = spawnTree.UpdateStatus(event.Dispatch.WorkerName, status, summary)
	}
}

func runtimeVisualDispatchObserver(spawnTree *agent.SpawnTree, activePrefix string, wave int) codex.DispatchObserver {
	treeObserver := spawnTreeDispatchObserver(spawnTree, activePrefix)
	return func(event codex.DispatchLifecycleEvent) {
		if treeObserver != nil {
			treeObserver(event)
		}

		switch strings.TrimSpace(event.Status) {
		case "starting":
			emitCodexDispatchWorkerStarted(event.Dispatch, wave)
		case "running", "active":
			emitCodexDispatchWorkerRunning(event.Dispatch, wave, event.Message)
		case "completed", "failed", "blocked", "timeout", "superseded", "manually-reconciled":
			emitCodexDispatchWorkerFinished(event.Dispatch, dispatchLifecycleResult(event))
		}
	}
}

func dispatchLifecycleResult(event codex.DispatchLifecycleEvent) codex.DispatchResult {
	return codex.DispatchResult{
		WorkerName:   event.Dispatch.WorkerName,
		Status:       normalizeRuntimeDispatchStatus(event.Status),
		WorkerResult: event.WorkerResult,
		Error:        event.Error,
	}
}

func dispatchBatchByWaveWithVisuals(
	ctx context.Context,
	invoker codex.WorkerInvoker,
	dispatches []codex.WorkerDispatch,
	parallelMode colony.ParallelMode,
	waveLabel string,
	parallelWithinWave bool,
	observerFactory func(wave int) codex.DispatchObserver,
) ([]codex.DispatchResult, error) {
	if err := preflightWorkerProvider(ctx, invoker, dispatches); err != nil {
		return nil, err
	}

	waves := codex.GroupByWave(dispatches)
	waveNumbers := make([]int, 0, len(waves))
	for wave := range waves {
		waveNumbers = append(waveNumbers, wave)
	}
	sort.Ints(waveNumbers)

	platform := codex.PlatformFromInvoker(invoker)
	results := make([]codex.DispatchResult, 0, len(dispatches))
	for _, wave := range waveNumbers {
		waveDispatches := waves[wave]
		emitCodexDispatchWaveProgress(waveLabel, wave, waveDispatches, parallelMode)

		var observer codex.DispatchObserver
		if observerFactory != nil {
			observer = observerFactory(wave)
		}

		waveResults, err := codex.DispatchWaveWithObserver(ctx, invoker, waveDispatches, observer, parallelWithinWave)
		if err != nil {
			// D-03: invalidate the trust window using whatever results were
			// collected from prior successful waves before this error.
			invalidatePreflightCacheOnProviderFailure(platform, results)
			return nil, err
		}
		results = append(results, waveResults...)
	}

	// D-03: a provider/auth-classified dispatch failure clears the trust
	// window so the next command re-probes. Placement is after dispatch, not
	// before -- this is the reaction to a completed dispatch.
	invalidatePreflightCacheOnProviderFailure(platform, results)
	return results, nil
}

type workerProviderPreflightError struct {
	status codex.AvailabilityStatus
}

func (e *workerProviderPreflightError) Error() string {
	status := e.status
	reason := strings.TrimSpace(status.Reason)
	if reason == "" {
		reason = "no diagnostic was returned"
	}
	if strings.Contains(strings.ToLower(reason), "preflight failed before worker dispatch") {
		return reason
	}
	if status.Platform == codex.PlatformCodex || strings.Contains(strings.ToLower(reason), "codex") {
		return fmt.Sprintf("Codex provider/model preflight failed before worker dispatch: %s", reason)
	}
	if status.Platform != codex.PlatformUnknown {
		return fmt.Sprintf("%s provider preflight failed before worker dispatch: %s", status.Platform, reason)
	}
	return fmt.Sprintf("worker provider preflight failed before dispatch: %s", reason)
}

func preflightWorkerProvider(ctx context.Context, invoker codex.WorkerInvoker, dispatches []codex.WorkerDispatch) error {
	if invoker == nil || len(dispatches) == 0 {
		return nil
	}
	preflighter, ok := invoker.(codex.WorkerProviderPreflighter)
	if !ok {
		return nil
	}
	root := ""
	for _, dispatch := range dispatches {
		if strings.TrimSpace(dispatch.Root) != "" {
			root = dispatch.Root
			break
		}
	}
	platform := codex.PlatformFromInvoker(invoker)
	status, outcome := gatedProviderPreflight(ctx, preflighter, platform, root, time.Now())
	if outcome.Notice != "" {
		visualFprintf(stderr, "%s\n", outcome.Notice)
	}
	if status.Available {
		return nil
	}
	return &workerProviderPreflightError{status: status}
}

func isWorkerProviderPreflightError(err error) bool {
	var preflightErr *workerProviderPreflightError
	return errors.As(err, &preflightErr)
}

func dispatchLifecycleSummary(event codex.DispatchLifecycleEvent, activePrefix string) string {
	status := strings.TrimSpace(event.Status)
	switch status {
	case "starting":
		if task := strings.TrimSpace(event.Dispatch.TaskBrief); task != "" {
			return firstContentLine(task)
		}
		if task := strings.TrimSpace(event.Dispatch.TaskID); task != "" {
			return task
		}
		return "Starting worker"
	case "active", "running":
		task := strings.TrimSpace(firstContentLine(event.Dispatch.TaskBrief))
		if task == "" {
			task = strings.TrimSpace(event.Dispatch.TaskID)
		}
		if task == "" {
			task = strings.TrimSpace(event.Dispatch.WorkerName)
		}
		if strings.TrimSpace(activePrefix) == "" {
			return task
		}
		return strings.TrimSpace(activePrefix + ": " + task)
	default:
		if event.WorkerResult != nil {
			if summary := strings.TrimSpace(event.WorkerResult.Summary); summary != "" {
				return summary
			}
			if len(event.WorkerResult.Blockers) > 0 {
				return strings.Join(event.WorkerResult.Blockers, "; ")
			}
		}
		if event.Error != nil {
			return strings.TrimSpace(event.Error.Error())
		}
		return strings.TrimSpace(event.Dispatch.WorkerName)
	}
}

func normalizeRuntimeDispatchStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "", "error":
		return "failed"
	case "timed_out":
		return "timeout"
	case "manual", "manually_reconciled":
		return "manually-reconciled"
	default:
		return status
	}
}

func firstContentLine(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}
