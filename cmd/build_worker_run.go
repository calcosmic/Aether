package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// FileLocker is reentrant within one Store. Serialize in-process worker
// mutations too; the existing file lock still arbitrates separate hosts.
var buildWorkerRunMutationMu sync.Mutex

const (
	buildWorkerDispatching = "dispatching"
	buildWorkerCompleted   = "completed"
	buildWorkerFailed      = "failed"
	buildWorkerBlocked     = "blocked"
	buildWorkerTimeout     = "timeout"
	buildWorkerCancelled   = "cancelled"
)

type buildAttemptWorkerRun struct {
	Native        *codexNativeWorkerBinding `json:"native,omitempty"`
	ProviderRunID string                    `json:"provider_run_id"`
	WorkerName    string                    `json:"worker_name"`
	TaskID        string                    `json:"task_id"`
	Caste         string                    `json:"caste"`
	Platform      codex.Platform            `json:"platform"`
	Status        string                    `json:"status"`
	ProcessID     int                       `json:"process_id,omitempty"`
	StartedAt     string                    `json:"started_at"`
	UpdatedAt     string                    `json:"updated_at"`
	CompletedAt   string                    `json:"completed_at,omitempty"`
	ResultSHA256  string                    `json:"result_sha256,omitempty"`
	Result        *internalWorkerResult     `json:"result,omitempty"`
	Error         string                    `json:"error,omitempty"`
}

// The accepted manifest chooses the worker lane before any provider mutation.
// Historical empty protocols retain the generic provider compatibility route.
func validateBuildWorkerProviderLane(record buildAttemptRecord) error {
	if record.PlanManifest != nil && record.PlanManifest.ContextProtocol != "" {
		return fmt.Errorf("saved context protocol %q requires native workers; generic provider dispatch is unavailable", record.PlanManifest.ContextProtocol)
	}
	return nil
}

func beginBuildAttemptWorkerRun(phase int, binding codex.ExecutionBinding, request internalWorkerDispatchRequest, providerRunID string, platform codex.Platform) (*buildAttemptWorkerRun, error) {
	buildWorkerRunMutationMu.Lock()
	defer buildWorkerRunMutationMu.Unlock()
	attemptRel, record, ok := loadLatestBuildAttempt(phase)
	if !ok {
		return nil, fmt.Errorf("build worker execution binding has no durable attempt")
	}
	if err := validateBuildExecutionBinding(record, binding, record.ManifestSHA256, false); err != nil {
		return nil, err
	}
	if err := validateBuildWorkerProviderLane(record); err != nil {
		return nil, err
	}
	if !buildAttemptHasDispatch(record, request) {
		return nil, fmt.Errorf("worker %s task %s is not authorized by build attempt %s", request.WorkerName, request.TaskID, record.ID)
	}

	now := time.Now().UTC()
	var cached *buildAttemptWorkerRun
	var updated buildAttemptRecord
	if err := store.UpdateJSONAtomically(attemptRel, &updated, func() error {
		if err := validateBuildExecutionBinding(updated, binding, updated.ManifestSHA256, false); err != nil {
			return err
		}
		if updated.Status != buildAttemptAwaiting && updated.Status != buildAttemptDispatching {
			return fmt.Errorf("build attempt %s is %s and cannot dispatch workers", updated.ID, updated.Status)
		}
		if err := validateBuildWorkerProviderLane(updated); err != nil {
			return err
		}
		if !buildAttemptHasDispatch(updated, request) {
			return fmt.Errorf("worker %s task %s is not authorized by build attempt %s", request.WorkerName, request.TaskID, updated.ID)
		}
		for i := len(updated.WorkerRuns) - 1; i >= 0; i-- {
			existing := &updated.WorkerRuns[i]
			if existing.WorkerName != strings.TrimSpace(request.WorkerName) || existing.TaskID != effectiveInternalWorkerTaskID(request) {
				continue
			}
			if existing.Native != nil {
				return fmt.Errorf("native assignment already reserved; inspect it without provider redispatch")
			}
			if existing.Status == buildWorkerCompleted && existing.Result != nil && existing.ResultSHA256 != "" {
				copyRun := *existing
				copyResult := *existing.Result
				copyRun.Result = &copyResult
				cached = &copyRun
				return nil
			}
			if existing.Status == buildWorkerDispatching && buildWorkerRunStillActive(*existing, now) {
				return fmt.Errorf("worker %s task %s is already active in provider run %s", existing.WorkerName, existing.TaskID, existing.ProviderRunID)
			}
			if existing.Status == buildWorkerDispatching {
				existing.Status = buildWorkerCancelled
				existing.Error = "provider process ended without a terminal result"
				existing.CompletedAt = now.Format(time.RFC3339Nano)
				existing.UpdatedAt = existing.CompletedAt
			}
			break
		}
		if cached != nil {
			return nil
		}
		started := now.Format(time.RFC3339Nano)
		updated.WorkerRuns = append(updated.WorkerRuns, buildAttemptWorkerRun{
			ProviderRunID: strings.TrimSpace(providerRunID),
			WorkerName:    strings.TrimSpace(request.WorkerName),
			TaskID:        effectiveInternalWorkerTaskID(request),
			Caste:         strings.TrimSpace(request.Caste),
			Platform:      platform,
			Status:        buildWorkerDispatching,
			StartedAt:     started,
			UpdatedAt:     started,
		})
		updated.Status = buildAttemptDispatching
		updated.UpdatedAt = started
		return nil
	}); err != nil {
		return nil, fmt.Errorf("record build worker dispatch: %w", err)
	}
	return cached, nil
}

func recordBuildAttemptWorkerProcess(phase int, binding codex.ExecutionBinding, providerRunID string, pid int) error {
	buildWorkerRunMutationMu.Lock()
	defer buildWorkerRunMutationMu.Unlock()
	if pid <= 0 {
		return nil
	}
	attemptRel, current, ok := loadLatestBuildAttempt(phase)
	if !ok {
		return fmt.Errorf("build worker execution binding has no durable attempt")
	}
	if err := validateBuildExecutionBinding(current, binding, current.ManifestSHA256, false); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var record buildAttemptRecord
	if err := store.UpdateJSONAtomically(attemptRel, &record, func() error {
		if err := validateBuildExecutionBinding(record, binding, record.ManifestSHA256, false); err != nil {
			return err
		}
		for i := range record.WorkerRuns {
			if record.WorkerRuns[i].ProviderRunID == strings.TrimSpace(providerRunID) {
				if record.WorkerRuns[i].Native != nil {
					return fmt.Errorf("native workers have no provider process")
				}
				record.WorkerRuns[i].ProcessID = pid
				record.WorkerRuns[i].UpdatedAt = now
				return nil
			}
		}
		return fmt.Errorf("provider run %s is not registered", providerRunID)
	}); err != nil {
		return fmt.Errorf("record build worker process: %w", err)
	}
	return nil
}

func recordBuildAttemptWorkerTerminal(phase int, binding codex.ExecutionBinding, providerRunID string, result *internalWorkerResult) error {
	buildWorkerRunMutationMu.Lock()
	defer buildWorkerRunMutationMu.Unlock()
	if result == nil {
		return fmt.Errorf("terminal worker result is required")
	}
	status := strings.ToLower(strings.TrimSpace(result.Status))
	switch status {
	case buildWorkerCompleted, buildWorkerFailed, buildWorkerBlocked, buildWorkerTimeout:
	default:
		return fmt.Errorf("terminal worker status %q is invalid", result.Status)
	}
	attemptRel, current, ok := loadLatestBuildAttempt(phase)
	if !ok {
		return fmt.Errorf("build worker execution binding has no durable attempt")
	}
	if err := validateBuildExecutionBinding(current, binding, current.ManifestSHA256, false); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var record buildAttemptRecord
	if err := store.UpdateJSONAtomically(attemptRel, &record, func() error {
		if err := validateBuildExecutionBinding(record, binding, record.ManifestSHA256, false); err != nil {
			return err
		}
		for i := range record.WorkerRuns {
			workerRun := &record.WorkerRuns[i]
			if workerRun.ProviderRunID != strings.TrimSpace(providerRunID) {
				continue
			}
			if workerRun.Native != nil {
				return fmt.Errorf("native terminal results require codex-native-worker record")
			}
			return applyBuildWorkerTerminal(&record, workerRun, result, now)
		}
		return fmt.Errorf("provider run %s is not registered", providerRunID)
	}); err != nil {
		return fmt.Errorf("record terminal build worker result: %w", err)
	}
	return nil
}

// applyBuildWorkerTerminal runs under the existing attempt lock in both lanes.
func applyBuildWorkerTerminal(record *buildAttemptRecord, workerRun *buildAttemptWorkerRun, result *internalWorkerResult, now string) error {
	result, err := normalizedBuildWorkerResult(result)
	if err != nil {
		return err
	}
	switch result.Status {
	case buildWorkerCompleted, buildWorkerFailed, buildWorkerBlocked, buildWorkerTimeout:
	case buildWorkerCancelled:
		if workerRun.Native == nil {
			return fmt.Errorf("only confirmed native cancellation is terminal here")
		}
	default:
		return fmt.Errorf("terminal worker status %q is invalid", result.Status)
	}
	digest, err := jsonSHA256(result)
	if err != nil {
		return err
	}
	if workerRun.ResultSHA256 != "" {
		if workerRun.ResultSHA256 != digest {
			return fmt.Errorf("provider run %s already has a different terminal result", workerRun.ProviderRunID)
		}
		if workerRun.Native != nil {
			return errCodexNativeReplay
		}
	}
	copyResult := *result
	workerRun.Result, workerRun.ResultSHA256, workerRun.Status = &copyResult, digest, result.Status
	workerRun.UpdatedAt, workerRun.CompletedAt, workerRun.ProcessID = now, now, 0
	workerRun.Error, record.UpdatedAt = strings.TrimSpace(result.Error), now
	return nil
}

func normalizedBuildWorkerResult(result *internalWorkerResult) (*internalWorkerResult, error) {
	if result == nil {
		return nil, fmt.Errorf("terminal worker result is required")
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var normalized internalWorkerResult
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, err
	}
	normalized.Status = strings.ToLower(strings.TrimSpace(normalized.Status))
	return &normalized, nil
}

func buildAttemptHasDispatch(record buildAttemptRecord, request internalWorkerDispatchRequest) bool {
	requestTaskID := effectiveInternalWorkerTaskID(request)
	for _, dispatch := range record.Dispatches {
		if dispatch.Name == strings.TrimSpace(request.WorkerName) && normalizedDispatchTaskID(dispatch) == requestTaskID && strings.EqualFold(dispatch.Caste, strings.TrimSpace(request.Caste)) {
			return true
		}
	}
	return false
}

func effectiveInternalWorkerTaskID(request internalWorkerDispatchRequest) string {
	if taskID := strings.TrimSpace(request.TaskID); taskID != "" {
		return taskID
	}
	return strings.TrimSpace(request.WorkerName)
}

func buildWorkerRunStillActive(run buildAttemptWorkerRun, now time.Time) bool {
	if run.Native != nil {
		return !codexNativeWorkerIsTerminal(run)
	}
	if run.ProcessID > 0 {
		return processAlive(run.ProcessID)
	}
	startedAt, err := time.Parse(time.RFC3339Nano, run.StartedAt)
	return err == nil && now.Sub(startedAt) < 30*time.Second
}

func cachedInternalWorkerResult(run *buildAttemptWorkerRun) *internalWorkerResult {
	if run == nil || run.Result == nil {
		return nil
	}
	copyResult := *run.Result
	return &copyResult
}

func cancelBuildAttemptWorkerRuns(attemptRel, reason string) error {
	buildWorkerRunMutationMu.Lock()
	defer buildWorkerRunMutationMu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var record buildAttemptRecord
	if err := store.UpdateJSONAtomically(attemptRel, &record, func() error {
		changed := false
		for i := range record.WorkerRuns {
			workerRun := &record.WorkerRuns[i]
			if workerRun.Native != nil || workerRun.Status != buildWorkerDispatching {
				continue
			}
			changed = true
			workerRun.Status = buildWorkerCancelled
			workerRun.ProcessID = 0
			workerRun.Error = strings.TrimSpace(reason)
			workerRun.UpdatedAt = now
			workerRun.CompletedAt = now
		}
		if !changed {
			return errCodexNativeReplay
		}
		record.UpdatedAt = now
		return nil
	}); err != nil {
		if errors.Is(err, errCodexNativeReplay) {
			return nil
		}
		return fmt.Errorf("record cancelled build workers: %w", err)
	}
	return nil
}

func stageBuildAttemptCompletionFromWorkerRuns(phase int, binding codex.ExecutionBinding) (string, bool, error) {
	attemptRel, record, ok := loadLatestBuildAttempt(phase)
	if !ok {
		return "", false, fmt.Errorf("build worker execution binding has no durable attempt")
	}
	if err := validateBuildExecutionBinding(record, binding, record.ManifestSHA256, false); err != nil {
		return "", false, err
	}
	if record.PlanManifest == nil || !record.PlanManifest.PlanOnly {
		return "", false, nil
	}
	completion, complete := buildCompletionFromWorkerRuns(record)
	if !complete {
		return "", false, nil
	}
	path, _, err := stageBuildAttemptCompletion(attemptRel, completion)
	if err != nil {
		return "", false, err
	}
	return path, true, nil
}

func beginDirectBuildWorkerRun(dispatch codex.WorkerDispatch, invoker codex.WorkerInvoker) error {
	if dispatch.ExecutionBinding == nil {
		return nil
	}
	request := internalWorkerDispatchRequest{
		SchemaVersion:     internalWorkerAdapterSchemaVersion,
		Workflow:          "build",
		Phase:             dispatch.Phase,
		AgentName:         dispatch.AgentName,
		Caste:             dispatch.Caste,
		WorkerName:        dispatch.WorkerName,
		TaskID:            dispatch.TaskID,
		Task:              dispatch.TaskBrief,
		TaskBrief:         dispatch.TaskBrief,
		ContextCapsule:    dispatch.ContextCapsule,
		SkillSection:      dispatch.SkillSection,
		PheromoneSection:  dispatch.PheromoneSection,
		HandoffSection:    dispatch.HandoffSection,
		TimeoutMS:         dispatch.Timeout.Milliseconds(),
		PermissionProfile: dispatch.PermissionProfile,
		ExecutionBinding:  dispatch.ExecutionBinding,
	}
	cached, err := beginBuildAttemptWorkerRun(
		dispatch.Phase,
		*dispatch.ExecutionBinding,
		request,
		dispatch.ProviderRunID,
		codex.PlatformFromInvoker(invoker),
	)
	if err != nil {
		return err
	}
	if cached != nil {
		return fmt.Errorf("worker %s task %s already has a terminal result in build run %s", dispatch.WorkerName, dispatch.TaskID, dispatch.ExecutionBinding.RunID)
	}
	return nil
}

func recordDirectBuildWorkerTerminal(dispatch codex.WorkerDispatch, result codex.DispatchResult) error {
	if dispatch.ExecutionBinding == nil {
		return nil
	}
	terminal := &internalWorkerResult{
		Name:   dispatch.WorkerName,
		Caste:  dispatch.Caste,
		TaskID: dispatch.TaskID,
		Status: strings.ToLower(strings.TrimSpace(result.Status)),
	}
	if result.WorkerResult != nil {
		terminal = mapInternalWorkerResult(*result.WorkerResult, result.Error)
		terminal.Name = dispatch.WorkerName
		terminal.Caste = dispatch.Caste
		terminal.TaskID = dispatch.TaskID
		terminal.Status = strings.ToLower(strings.TrimSpace(result.Status))
	}
	if result.Error != nil {
		terminal.Error = sanitizeInternalWorkerAdapterError(result.Error.Error())
		if strings.TrimSpace(terminal.Summary) == "" {
			terminal.Summary = terminal.Error
		}
	}
	switch terminal.Status {
	case buildWorkerCompleted, buildWorkerFailed, buildWorkerBlocked, buildWorkerTimeout:
	default:
		return fmt.Errorf("worker %s returned invalid terminal status %q", dispatch.WorkerName, terminal.Status)
	}
	return recordBuildAttemptWorkerTerminal(
		dispatch.Phase,
		*dispatch.ExecutionBinding,
		dispatch.ProviderRunID,
		terminal,
	)
}

func buildCompletionFromWorkerRuns(record buildAttemptRecord) (codexExternalBuildCompletion, bool) {
	if record.PlanManifest == nil || len(record.PlanManifest.Dispatches) == 0 {
		return codexExternalBuildCompletion{}, false
	}
	results := make([]codexExternalBuildWorkerResult, 0, len(record.PlanManifest.Dispatches))
	for _, dispatch := range record.PlanManifest.Dispatches {
		workerRun, ok := latestTerminalBuildWorkerRun(record.WorkerRuns, dispatch.Name, normalizedDispatchTaskID(dispatch))
		if !ok || workerRun.Result == nil {
			return codexExternalBuildCompletion{}, false
		}
		result := workerRun.Result
		status := result.Status
		if workerRun.Native != nil && status == buildWorkerCancelled {
			// Host-confirmed cancellation/no-launch remains immutable in the
			// journal. The existing completion contract calls it interrupted.
			status = "interrupted"
		}
		summary := strings.TrimSpace(result.Summary)
		if summary == "" {
			summary = strings.TrimSpace(result.Error)
		}
		if summary == "" {
			summary = fmt.Sprintf("Worker %s returned %s", dispatch.Name, result.Status)
		}
		results = append(results, codexExternalBuildWorkerResult{
			Stage:         dispatch.Stage,
			Wave:          dispatch.Wave,
			ExecutionWave: dispatch.ExecutionWave,
			Caste:         dispatch.Caste,
			Name:          dispatch.Name,
			Task:          dispatch.Task,
			Status:        status,
			Summary:       summary,
			TaskID:        dispatch.TaskID,
			Duration:      result.Duration,
			ToolCount:     result.ToolCount,
			FilesCreated:  append([]string(nil), result.FilesCreated...),
			FilesModified: append([]string(nil), result.FilesModified...),
			TestsWritten:  append([]string(nil), result.TestsWritten...),
			// Threaded through unchanged, whatever the terminal status (D-08,
			// D-09): a native worker run that failed or was interrupted can
			// still carry honest task-specific receipts, and dropping them
			// here would silently make the failed-terminal-status branch of
			// completedBuildTaskIDs unreachable for this lane.
			TaskReceipts: append([]codex.TaskReceipt(nil), result.TaskReceipts...),
			Blockers:     append([]string(nil), result.Blockers...),
			Handoff:      result.Handoff,
			Artifacts:    result.Artifacts,
			ScoutReport:  result.ScoutReport,
		})
	}
	manifest := *record.PlanManifest
	return codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}, true
}

func latestTerminalBuildWorkerRun(runs []buildAttemptWorkerRun, workerName, taskID string) (buildAttemptWorkerRun, bool) {
	for i := len(runs) - 1; i >= 0; i-- {
		run := runs[i]
		if run.WorkerName != workerName || run.TaskID != taskID || run.Result == nil || run.ResultSHA256 == "" {
			continue
		}
		switch run.Status {
		case buildWorkerCompleted, buildWorkerFailed, buildWorkerBlocked, buildWorkerTimeout:
			return run, true
		case buildWorkerCancelled:
			if run.Native != nil {
				return run, true
			}
		}
	}
	return buildAttemptWorkerRun{}, false
}
