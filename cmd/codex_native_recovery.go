package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Recovery is a read-only projection of the accepted attempt. It grants neither
// launch permission nor task credit; reserve and finalize retain those duties.
type codexNativeRecovery struct {
	AttemptID  string                      `json:"attempt_id"`
	Phase      int                         `json:"phase"`
	Valid      bool                        `json:"valid"`
	Finished   []codexNativeRecoveryWorker `json:"finished"`
	Unfinished []codexNativeRecoveryWorker `json:"unfinished"`
	Unresolved []codexNativeRecoveryWorker `json:"unresolved"`
	Active     []codexNativeRecoveryWorker `json:"active"`
	Next       string                      `json:"next"`
	Summary    string                      `json:"summary"`
	Why        string                      `json:"why"`
	Error      string                      `json:"error,omitempty"`
}

type codexNativeRecoveryWorker struct {
	WorkerName      string `json:"worker_name"`
	Caste           string `json:"caste"`
	TaskID          string `json:"task_id"`
	LaunchID        string `json:"launch_id,omitempty"`
	HostSessionID   string `json:"host_session_id,omitempty"`
	ChildID         string `json:"child_id,omitempty"`
	Status          string `json:"status"`
	HostStatus      string `json:"host_status,omitempty"`
	LastHostStatus  string `json:"last_host_status,omitempty"`
	HostAction      string `json:"host_action,omitempty"`
	ObservedAt      string `json:"observed_at,omitempty"`
	ResultSHA256    string `json:"result_sha256,omitempty"`
	CancelRequested bool   `json:"cancel_requested,omitempty"`
}

func buildCodexNativeRecovery(state colony.ColonyState) *codexNativeRecovery {
	_, attempt, ok := loadRelevantBuildAttempt(state)
	if !ok || !buildAttemptStatusActive(attempt.Status) || attempt.PlanManifest == nil || !attempt.PlanManifest.PlanOnly || attempt.ExecutionOwner != "host-queen" {
		return nil
	}
	hasNative := false
	for _, worker := range attempt.WorkerRuns {
		hasNative = hasNative || worker.Native != nil
	}
	// Host platform does not select a worker transport. Existing runs with no
	// native binding belong to the ordinary adapter recovery path. An empty
	// Codex attempt still needs native guidance before its first reservation;
	// any native binding keeps mixed or damaged evidence under validation here.
	if !hasNative && (len(attempt.WorkerRuns) > 0 || attempt.PlanManifest.HostPlatform != "codex") {
		return nil
	}
	recovery := &codexNativeRecovery{AttemptID: attempt.ID, Phase: attempt.Phase,
		Finished: []codexNativeRecoveryWorker{}, Unfinished: []codexNativeRecoveryWorker{},
		Unresolved: []codexNativeRecoveryWorker{}, Active: []codexNativeRecoveryWorker{},
		Next: fmt.Sprintf("aether codex-native-worker inspect --phase %d", attempt.Phase)}
	invalid := func(err error) *codexNativeRecovery {
		recovery.Error = err.Error()
		recovery.Next = "aether status"
		recovery.Summary = "Saved native recovery evidence does not match the current assignment or workspace. Preserve the saved work and inspect the conflict."
		recovery.Why = "Uncertain saved evidence cannot authorize a replacement helper or finalization."
		return recovery
	}
	manifest := attempt.PlanManifest
	if manifest.ExecutionBinding == nil || manifest.Phase != attempt.Phase || len(manifest.Dispatches) == 0 {
		return invalid(fmt.Errorf("missing accepted native manifest binding"))
	}
	if err := validateBuildExecutionBinding(attempt, *manifest.ExecutionBinding, attempt.ManifestSHA256, false); err != nil {
		return invalid(err)
	}
	digest, err := buildManifestSHA256(*manifest)
	if err != nil || digest != attempt.ManifestSHA256 {
		return invalid(fmt.Errorf("saved native manifest digest changed"))
	}
	root, err := filepath.EvalSymlinks(manifest.Root)
	if err != nil {
		return invalid(err)
	}
	activeRoot, err := filepath.EvalSymlinks(buildAttemptWorkspaceRoot())
	if err != nil || root != activeRoot {
		return invalid(fmt.Errorf("native recovery workspace changed"))
	}
	// Pausing does not invalidate retained terminal evidence. All other phase
	// currency checks remain identical to finalization's existing authority.
	currency := state
	currency.Paused = false
	if err := validateBuildFinalizeStateStillCurrent(currency, attempt.Phase); err != nil {
		return invalid(err)
	}
	if err := validateCodexNativeAttemptState(attempt, state); err != nil {
		return invalid(err)
	}
	matched := 0
	for _, dispatch := range manifest.Dispatches {
		item := codexNativeRecoveryWorker{WorkerName: dispatch.Name, Caste: dispatch.Caste, TaskID: normalizedDispatchTaskID(dispatch), Status: "never_started"}
		var saved *buildAttemptWorkerRun
		for i := range attempt.WorkerRuns {
			worker := &attempt.WorkerRuns[i]
			if worker.WorkerName != item.WorkerName || worker.TaskID != item.TaskID {
				continue
			}
			if saved != nil {
				return invalid(fmt.Errorf("multiple saved launches for native assignment %s", item.WorkerName))
			}
			saved = worker
			matched++
		}
		if saved == nil {
			recovery.Unfinished = append(recovery.Unfinished, item)
			continue
		}
		if err := validateCodexNativeSavedWorker(attempt, dispatch, *saved); err != nil {
			return invalid(err)
		}
		projected := projectCodexNativeWorkerState(*saved)
		item.LaunchID, item.HostSessionID, item.ChildID = saved.ProviderRunID, saved.Native.HostSessionID, saved.Native.ChildID
		item.HostStatus, item.CancelRequested = projected.HostStatus, projected.CancelRequested
		for i := len(saved.Native.Observations) - 1; i >= 0; i-- {
			observation := saved.Native.Observations[i]
			if observation.Status == "context_delivered" || observation.Status == "context_fetched" {
				continue
			}
			item.ObservedAt, item.LastHostStatus = observation.ObservedAt, observation.Status
			break
		}
		if !projected.Terminal {
			if item.ChildID == "" {
				item.HostAction = fmt.Sprintf("Reconcile launch %s with host session %s. Bind only its actual child; if the host cannot locate the launch, preserve it as unresolved. A missing process ID does not permit another launch.", item.LaunchID, item.HostSessionID)
			} else {
				item.HostAction = fmt.Sprintf("Ask host session %s to observe the same child %s. To stop it, use the host's actual interruption/cancellation tool against that child and record its matching terminal acknowledgement. Without that capability or acknowledgement, cancellation stays pending.", item.HostSessionID, item.ChildID)
			}
		}
		if projected.Terminal {
			digest, err := jsonSHA256(saved.Result)
			if err != nil || digest != saved.ResultSHA256 {
				return invalid(fmt.Errorf("saved native terminal result digest changed"))
			}
			item.Status, item.ResultSHA256 = saved.Status, saved.ResultSHA256
			recovery.Finished = append(recovery.Finished, item)
		} else if projected.HostStatus == "running" || projected.CancelRequested {
			item.Status = "last_observed_running"
			if projected.CancelRequested {
				item.Status = "cancellation_pending"
			}
			recovery.Active = append(recovery.Active, item)
		} else {
			item.Status = "host_unavailable"
			if item.ChildID == "" {
				item.Status = "awaiting_binding"
			}
			recovery.Unresolved = append(recovery.Unresolved, item)
		}
	}
	if matched != len(attempt.WorkerRuns) {
		return invalid(fmt.Errorf("saved native worker is outside the accepted manifest"))
	}
	recovery.Valid = true
	recovery.Summary = fmt.Sprintf("%d helper result(s) saved; %d assignment(s) never started; %d helper(s) awaiting host reconciliation; %d helper(s) with a saved activity or cancellation observation.", len(recovery.Finished), len(recovery.Unfinished), len(recovery.Unresolved), len(recovery.Active))
	recovery.Why = "Keep saved results. The host may reserve only never-started assignments from this attempt; an existing launch must be reconciled using the same host session and child identity. Saved activity is an observation, not a fresh liveness check."
	if len(recovery.Finished) == len(manifest.Dispatches) {
		recovery.Next = fmt.Sprintf("aether codex-native-worker stage --phase %d", attempt.Phase)
		recovery.Why = "Every helper has a saved terminal result. Explicitly stage those exact results before finalization; no helper needs to run again."
	}
	if strings.TrimSpace(attempt.CompletionPath) != "" && strings.TrimSpace(attempt.CompletionSHA256) != "" {
		recovery.Next = buildFinalizeRecoveryCommand(attempt.Phase, attempt.CompletionPath)
		recovery.Why = "The exact accepted completion packet is durable. Finalize it to apply verified results without dispatching helpers again."
	}
	if state.Paused && !recovery.pendingBoundary() {
		recovery.Next = "aether resume"
		recovery.Why = "Restore the validated pause handoff before continuing with the saved native attempt."
	}
	return recovery
}

func (recovery *codexNativeRecovery) pendingBoundary() bool {
	return !recovery.Valid || len(recovery.Unresolved) > 0 || len(recovery.Active) > 0
}

func applyCodexNativeRecovery(result map[string]interface{}, recovery *codexNativeRecovery) {
	result["native_recovery"] = recovery
	result["resume_override_command"], result["resume_override_why"] = recovery.Next, recovery.Why
	if block, ok := result["recovery"].(map[string]interface{}); ok {
		block["summary"], block["next"] = recovery.Summary, recovery.Next
	}
	if block, ok := result["session"].(map[string]interface{}); ok {
		block["summary"], block["suggested_next"] = recovery.Summary, recovery.Next
	}
}

func renderCodexNativeRecovery(recovery *codexNativeRecovery) string {
	var b strings.Builder
	b.WriteString(recovery.Summary + "\n")
	for _, group := range [][]codexNativeRecoveryWorker{recovery.Finished, recovery.Unfinished, recovery.Unresolved, recovery.Active} {
		for _, worker := range group {
			b.WriteString("  " + renderCodexNativeWorkerActivity(worker) + "\n")
			if worker.HostAction != "" {
				b.WriteString("    " + worker.HostAction + "\n")
			}
		}
	}
	b.WriteString(recovery.Why + "\nNext: " + recovery.Next)
	return b.String()
}
