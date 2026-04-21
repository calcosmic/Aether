package codex

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// WorkerDispatch represents a single worker to be executed as part of a batch.
// It maps to the codexBuildDispatch from cmd/codex_build.go but lives in the
// pkg layer to avoid cmd dependencies.
type WorkerDispatch struct {
	ID               string        // Unique dispatch identifier
	WorkerName       string        // Deterministic ant name (e.g., "Hammer-23")
	AgentName        string        // TOML agent name (e.g., "aether-builder")
	AgentTOMLPath    string        // Absolute path to the agent's TOML file
	Caste            string        // Worker caste (builder, watcher, scout, etc.)
	TaskID           string        // Task identifier from the build dispatch
	TaskBrief        string        // The markdown task brief content
	ContextCapsule   string        // The assembled compact colony-prime context for the worker
	Wave             int           // Wave number for dependency ordering (1-based)
	Root             string        // Working tree root for the worker process
	Timeout          time.Duration // Per-worker timeout override
	SkillSection     string        // Skill guidance content injected into worker prompts
	PheromoneSection string        // Pheromone signal content injected into worker prompts
}

// DispatchResult captures the outcome of a single worker dispatch within a batch.
type DispatchResult struct {
	WorkerName   string        // The worker's assigned name
	Status       string        // "completed", "failed", or "timeout"
	WorkerResult *WorkerResult // The full worker result (nil if invocation failed entirely)
	Error        error         // Error from invocation (if any)
}

// DispatchLifecycleEvent reports a runtime transition for a worker dispatch.
type DispatchLifecycleEvent struct {
	Dispatch     WorkerDispatch
	Status       string
	Message      string
	WorkerResult *WorkerResult
	Error        error
	OccurredAt   time.Time
}

// DispatchObserver receives runtime lifecycle events while dispatches execute.
type DispatchObserver func(DispatchLifecycleEvent)

// ClaimsSummary aggregates file claims across all successful workers in a batch.
// It matches the last-build-claims.json schema used by cmd/codex_build.go.
type ClaimsSummary struct {
	FilesCreated  []string            `json:"files_created"`
	FilesModified []string            `json:"files_modified"`
	TestsWritten  []string            `json:"tests_written"`
	TaskClaims    []TaskClaimsSummary `json:"task_claims,omitempty"`
}

type TaskClaimsSummary struct {
	TaskID        string   `json:"task_id"`
	FilesCreated  []string `json:"files_created,omitempty"`
	FilesModified []string `json:"files_modified,omitempty"`
	TestsWritten  []string `json:"tests_written,omitempty"`
}

// GroupByWave groups dispatches by their Wave field and returns a map sorted
// by wave number. Wave numbers are 1-based.
func GroupByWave(dispatches []WorkerDispatch) map[int][]WorkerDispatch {
	groups := make(map[int][]WorkerDispatch)
	for _, d := range dispatches {
		groups[d.Wave] = append(groups[d.Wave], d)
	}
	return groups
}

// DispatchBatch executes multiple workers with wave-based dependency ordering.
// Workers are grouped by their Wave field and executed wave-by-wave (wave 1 first,
// then wave 2, etc.). Within each wave, workers execute sequentially.
//
// Failed workers are recorded but do NOT stop subsequent waves from executing.
// Returns all results when all waves complete. The returned error is always nil
// (failures are captured per-worker in DispatchResult.Error).
func DispatchBatch(ctx context.Context, invoker WorkerInvoker, dispatches []WorkerDispatch) ([]DispatchResult, error) {
	return DispatchBatchWithObserver(ctx, invoker, dispatches, nil)
}

// DispatchBatchWithObserver executes multiple workers while emitting lifecycle transitions.
func DispatchBatchWithObserver(ctx context.Context, invoker WorkerInvoker, dispatches []WorkerDispatch, observer DispatchObserver) ([]DispatchResult, error) {
	if len(dispatches) == 0 {
		return nil, nil
	}

	waves := GroupByWave(dispatches)
	sortedWaves := sortedWaveKeys(waves)

	var allResults []DispatchResult

	for _, waveNum := range sortedWaves {
		waveDispatches := waves[waveNum]

		for _, d := range waveDispatches {
			// Check if context is already cancelled before invoking
			if ctx.Err() != nil {
				emitDispatchLifecycle(observer, d, "timeout", "", nil, ctx.Err())
				allResults = append(allResults, DispatchResult{
					WorkerName: d.WorkerName,
					Status:     "timeout",
					Error:      ctx.Err(),
				})
				continue
			}

			config := WorkerConfig{
				AgentName:        d.AgentName,
				AgentTOMLPath:    d.AgentTOMLPath,
				Caste:            d.Caste,
				WorkerName:       d.WorkerName,
				TaskID:           d.TaskID,
				TaskBrief:        d.TaskBrief,
				ContextCapsule:   d.ContextCapsule,
				Root:             d.Root,
				Timeout:          d.Timeout,
				SkillSection:     d.SkillSection,
				PheromoneSection: d.PheromoneSection,
			}

			emitDispatchLifecycle(observer, d, "starting", "", nil, nil)
			if progressInvoker, ok := invoker.(ProgressAwareWorkerInvoker); ok {
				result, err := progressInvoker.InvokeWithProgress(ctx, config, func(progress WorkerProgressEvent) {
					status := normalizeDispatchProgressStatus(progress.Status)
					if status == "" {
						return
					}
					emitDispatchLifecycle(observer, d, status, progress.Message, nil, nil)
				})
				dr := DispatchResult{
					WorkerName: d.WorkerName,
				}
				if err != nil {
					dr.Status = "failed"
					dr.Error = err
				} else if result.Status == "completed" {
					dr.Status = "completed"
					dr.WorkerResult = &result
				} else {
					dr.Status = result.Status
					dr.WorkerResult = &result
					if result.Error != nil {
						dr.Error = result.Error
					}
				}

				emitDispatchLifecycle(observer, d, dr.Status, "", dr.WorkerResult, dr.Error)
				allResults = append(allResults, dr)
				continue
			}
			result, err := invoker.Invoke(ctx, config)

			dr := DispatchResult{
				WorkerName: d.WorkerName,
			}

			if err != nil {
				dr.Status = "failed"
				dr.Error = err
			} else if result.Status == "completed" {
				dr.Status = "completed"
				dr.WorkerResult = &result
			} else {
				// Map worker result status to dispatch result status
				dr.Status = result.Status
				dr.WorkerResult = &result
				if result.Error != nil {
					dr.Error = result.Error
				}
			}

			emitDispatchLifecycle(observer, d, dr.Status, "", dr.WorkerResult, dr.Error)
			allResults = append(allResults, dr)
		}
	}

	return allResults, nil
}

// ExtractClaims aggregates files_created, files_modified, and tests_written
// from all successful (status == "completed") dispatch results.
func ExtractClaims(results []DispatchResult) *ClaimsSummary {
	summary := &ClaimsSummary{
		FilesCreated:  []string{},
		FilesModified: []string{},
		TestsWritten:  []string{},
	}
	taskClaims := map[string]*TaskClaimsSummary{}

	for _, r := range results {
		if r.Status != "completed" || r.WorkerResult == nil {
			continue
		}
		summary.FilesCreated = append(summary.FilesCreated, r.WorkerResult.FilesCreated...)
		summary.FilesModified = append(summary.FilesModified, r.WorkerResult.FilesModified...)
		summary.TestsWritten = append(summary.TestsWritten, r.WorkerResult.TestsWritten...)
		taskID := strings.TrimSpace(r.WorkerResult.TaskID)
		if taskID == "" {
			continue
		}
		entry, ok := taskClaims[taskID]
		if !ok {
			entry = &TaskClaimsSummary{TaskID: taskID}
			taskClaims[taskID] = entry
		}
		entry.FilesCreated = append(entry.FilesCreated, r.WorkerResult.FilesCreated...)
		entry.FilesModified = append(entry.FilesModified, r.WorkerResult.FilesModified...)
		entry.TestsWritten = append(entry.TestsWritten, r.WorkerResult.TestsWritten...)
	}

	if len(taskClaims) > 0 {
		taskIDs := make([]string, 0, len(taskClaims))
		for taskID := range taskClaims {
			taskIDs = append(taskIDs, taskID)
		}
		sort.Strings(taskIDs)
		summary.TaskClaims = make([]TaskClaimsSummary, 0, len(taskIDs))
		for _, taskID := range taskIDs {
			entry := taskClaims[taskID]
			summary.TaskClaims = append(summary.TaskClaims, TaskClaimsSummary{
				TaskID:        entry.TaskID,
				FilesCreated:  append([]string{}, entry.FilesCreated...),
				FilesModified: append([]string{}, entry.FilesModified...),
				TestsWritten:  append([]string{}, entry.TestsWritten...),
			})
		}
	}

	return summary
}

// sortedWaveKeys returns the wave numbers from the groups map in ascending order.
func sortedWaveKeys(groups map[int][]WorkerDispatch) []int {
	keys := make([]int, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// String returns a human-readable summary of the dispatch result.
func (r DispatchResult) String() string {
	status := r.Status
	if r.Error != nil {
		status = fmt.Sprintf("%s (%v)", status, r.Error)
	}
	return fmt.Sprintf("[%s] %s", status, r.WorkerName)
}

func emitDispatchLifecycle(observer DispatchObserver, dispatch WorkerDispatch, status string, message string, workerResult *WorkerResult, err error) {
	if observer == nil {
		return
	}
	observer(DispatchLifecycleEvent{
		Dispatch:     dispatch,
		Status:       status,
		Message:      strings.TrimSpace(message),
		WorkerResult: workerResult,
		Error:        err,
		OccurredAt:   time.Now().UTC(),
	})
}

func normalizeDispatchProgressStatus(status string) string {
	switch normalizeDispatchStatus(status) {
	case "active", "running":
		return "running"
	default:
		return ""
	}
}

func normalizeDispatchStatus(status string) string {
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
