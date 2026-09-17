package cmd

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const codexNativeWorkerSchemaVersion = 1

var errCodexNativeReplay = errors.New("native worker replay: no write")

type codexNativeDisposition string

const (
	codexNativeAccepted codexNativeDisposition = "accepted"
	codexNativeReplayed codexNativeDisposition = "replay"
)

// Hooks are instance-local: tests can stop after prevalidation without a global
// callback racing other requests. Production always uses the empty value.
type codexNativeWorkerHooks struct{ BeforeWrite func() }

// Native provenance extends the existing worker journal, never its credit authority.
// Prompt is retained verbatim so replay never silently recomposes an assignment.
type codexNativeWorkerBinding struct {
	SchemaVersion      int                     `json:"schema_version"`
	LaunchState        string                  `json:"launch_state"`
	HostSessionID      string                  `json:"host_session_id"`
	ChildID            string                  `json:"child_id,omitempty"`
	DispatchSHA256     string                  `json:"dispatch_sha256"`
	PromptSHA256       string                  `json:"prompt_sha256"`
	PermissionProfile  codex.PermissionProfile `json:"permission_profile"`
	Workspace          string                  `json:"workspace"`
	WorkspaceRoot      string                  `json:"workspace_root,omitempty"`
	ContextDeliveryIDs []string                `json:"context_delivery_ids,omitempty"`
	Prompt             string                  `json:"prompt"`
	Release            string                  `json:"release,omitempty"`
	SourceEventID      string                  `json:"source_event_id,omitempty"`
	SourceEventSHA256  string                  `json:"source_event_sha256,omitempty"`
	RawResult          json.RawMessage         `json:"raw_result,omitempty"`
}

type codexNativeWorkerRequest struct {
	SchemaVersion     int                    `json:"schema_version"`
	Phase             int                    `json:"phase"`
	ExecutionBinding  codex.ExecutionBinding `json:"execution_binding"`
	WorkerName        string                 `json:"worker_name,omitempty"`
	TaskID            string                 `json:"task_id,omitempty"`
	LaunchID          string                 `json:"launch_id,omitempty"`
	HostSessionID     string                 `json:"host_session_id,omitempty"`
	ChildID           string                 `json:"child_id,omitempty"`
	DispatchSHA256    string                 `json:"dispatch_sha256,omitempty"`
	PromptSHA256      string                 `json:"prompt_sha256,omitempty"`
	Workspace         string                 `json:"workspace,omitempty"`
	HostPermission    string                 `json:"host_permission,omitempty"`
	SourceEventID     string                 `json:"source_event_id,omitempty"`
	SourceEventSHA256 string                 `json:"source_event_sha256,omitempty"`
	Result            *internalWorkerResult  `json:"result,omitempty"`
	RawResult         json.RawMessage        `json:"-"`
}

type codexNativeWorkerResponse struct {
	Disposition      codexNativeDisposition  `json:"disposition,omitempty"`
	SchemaVersion    int                     `json:"schema_version"`
	ExecutionBinding codex.ExecutionBinding  `json:"execution_binding"`
	LaunchAllowed    bool                    `json:"launch_allowed"`
	Replay           bool                    `json:"replay,omitempty"`
	Dispatch         *codexBuildDispatch     `json:"dispatch,omitempty"`
	Worker           *buildAttemptWorkerRun  `json:"worker,omitempty"`
	Workers          []buildAttemptWorkerRun `json:"workers,omitempty"`
	CompletionPath   string                  `json:"completion_path,omitempty"`
	Complete         bool                    `json:"complete,omitempty"`
}

func init() {
	command := &cobra.Command{Use: "codex-native-worker", Short: "Internal non-launching native worker journal bridge", Hidden: true}
	for _, operation := range []string{"reserve", "bind", "record", "stage", "inspect"} {
		operation := operation
		child := &cobra.Command{Use: operation, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			path, _ := cmd.Flags().GetString("request")
			response, err := runCodexNativeWorker(operation, path)
			if err != nil {
				outputError(1, sanitizeInternalWorkerAdapterError(err.Error()), nil)
				return renderedErrorExit(1)
			}
			outputOK(response)
			return nil
		}}
		child.Flags().String("request", "", "Absolute regular JSON file under an aether-worker-request-* temporary directory")
		command.AddCommand(child)
	}
	rootCmd.AddCommand(command)
}

func loadCodexNativeWorkerRequest(path string) (codexNativeWorkerRequest, error) {
	var request codexNativeWorkerRequest
	if !filepath.IsAbs(path) {
		return request, fmt.Errorf("native request requires an absolute --request file")
	}
	if err := validateFinalizerCompletionFilePath(path); err != nil {
		return request, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return request, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return request, fmt.Errorf("native request must be a regular non-symlink file")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return request, err
	}
	temp, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return request, err
	}
	rel, err := filepath.Rel(temp, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || !strings.HasPrefix(filepath.Base(filepath.Dir(resolved)), "aether-worker-request-") {
		return request, fmt.Errorf("native request must use an aether-worker-request-* directory under the system temporary directory")
	}
	file, err := os.Open(path)
	if err != nil {
		return request, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, internalWorkerRequestMaxBytes+1))
	if err != nil {
		return request, err
	}
	if len(data) > internalWorkerRequestMaxBytes {
		return request, fmt.Errorf("native request exceeds %d bytes", internalWorkerRequestMaxBytes)
	}
	type requestFields codexNativeWorkerRequest
	wire := struct {
		*requestFields
		Result json.RawMessage `json:"result"`
	}{requestFields: (*requestFields)(&request)}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return request, fmt.Errorf("parse native request: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return request, fmt.Errorf("native request must contain exactly one JSON object")
	}
	if request.SchemaVersion != codexNativeWorkerSchemaVersion || request.Phase <= 0 {
		return request, fmt.Errorf("native request requires schema_version 1 and a positive phase")
	}
	if len(wire.Result) != 0 && string(wire.Result) != "null" {
		result, err := normalizeCodexNativeResult(wire.Result)
		if err != nil {
			return request, err
		}
		request.Result, request.RawResult = &result, wire.Result
	}
	return request, nil
}

// The installed Builder's public wire contract has ant_name, tdd and
// code_written. Normalize only these supported aliases, retaining the original
// JSON alongside its source event. TDD prose never becomes provider telemetry.
func normalizeCodexNativeResult(raw []byte) (internalWorkerResult, error) {
	var wire struct {
		internalWorkerResult
		AntName string `json:"ant_name"`
		TDD     *struct {
			CyclesCompleted int      `json:"cycles_completed"`
			TestsAdded      int      `json:"tests_added"`
			CoveragePercent *float64 `json:"coverage_percent"`
			AllPassing      bool     `json:"all_passing"`
		} `json:"tdd,omitempty"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return internalWorkerResult{}, fmt.Errorf("native terminal wire: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return internalWorkerResult{}, fmt.Errorf("native terminal requires exactly one JSON object")
	}
	if wire.Name != "" && wire.AntName != "" && wire.Name != wire.AntName {
		return internalWorkerResult{}, fmt.Errorf("native name and ant_name conflict")
	}
	if wire.Name == "" {
		wire.Name = wire.AntName
	}
	wire.Status = strings.ToLower(strings.TrimSpace(wire.Status))
	if wire.Status == "code_written" {
		wire.Status = buildWorkerCompleted
	}
	for i := range wire.TaskReceipts {
		if wire.TaskReceipts[i].Status == "code_written" {
			wire.TaskReceipts[i].Status = "completed"
		}
	}
	return wire.internalWorkerResult, nil
}

// This bridge cannot launch processes or providers. Only the host's spawn_agent
// consumes a new reservation, then bind releases that exact child to work.
func runCodexNativeWorker(operation, path string) (codexNativeWorkerResponse, error) {
	return runCodexNativeWorkerWithHooks(operation, path, codexNativeWorkerHooks{})
}

func runCodexNativeWorkerWithHooks(operation, path string, hooks codexNativeWorkerHooks) (codexNativeWorkerResponse, error) {
	request, err := loadCodexNativeWorkerRequest(path)
	if err != nil {
		return codexNativeWorkerResponse{}, err
	}
	response := codexNativeWorkerResponse{SchemaVersion: 1, ExecutionBinding: request.ExecutionBinding}
	attemptPath, current, ok := loadLatestBuildAttempt(request.Phase)
	if !ok {
		return response, fmt.Errorf("native worker requires a durable accepted build attempt")
	}
	validate := func(record buildAttemptRecord) error {
		if err := validateBuildExecutionBinding(record, request.ExecutionBinding, record.ManifestSHA256, false); err != nil {
			return err
		}
		if record.PlanManifest == nil || !record.PlanManifest.PlanOnly || record.ExecutionOwner != "host-queen" {
			return fmt.Errorf("native worker requires a host-queen plan-only attempt")
		}
		manifest := record.PlanManifest
		if len(manifest.Dispatches) == 0 || manifest.ExecutionBinding == nil || *manifest.ExecutionBinding != request.ExecutionBinding || manifest.Phase != request.Phase {
			return fmt.Errorf("native worker requires a nonempty exactly bound manifest")
		}
		digest, err := buildManifestSHA256(*manifest)
		if err != nil || digest != record.ManifestSHA256 {
			return fmt.Errorf("saved native manifest content no longer matches its accepted digest")
		}
		root, err := filepath.EvalSymlinks(manifest.Root)
		if err != nil {
			return err
		}
		activeRoot, err := filepath.EvalSymlinks(buildAttemptWorkspaceRoot())
		if err != nil || root != activeRoot {
			return fmt.Errorf("native manifest workspace is not the accepted workspace")
		}
		var pointer latestBuildAttemptPointer
		if err := store.LoadJSON(latestBuildAttemptPointerPath(request.Phase), &pointer); err != nil {
			return err
		}
		if pointer.AttemptID != record.ID {
			return fmt.Errorf("native attempt has been superseded")
		}
		return nil
	}
	if err := validate(current); err != nil {
		return response, err
	}
	if operation == "inspect" {
		response.Workers = current.WorkerRuns
		response.CompletionPath = current.CompletionPath
		response.Complete = current.CompletionPath != ""
		return response, nil
	}
	if operation == "stage" {
		if len(current.WorkerRuns) == 0 {
			return response, fmt.Errorf("native stage requires saved terminal workers")
		}
		for _, worker := range current.WorkerRuns {
			if worker.Native == nil || worker.Native.LaunchState != "terminal" || worker.Result == nil {
				return response, fmt.Errorf("native stage requires bound terminal results for every worker")
			}
		}
		response.CompletionPath, response.Complete, err = stageBuildAttemptCompletionFromWorkerRuns(request.Phase, request.ExecutionBinding)
		if err == nil && !response.Complete {
			err = fmt.Errorf("native stage is incomplete: required terminal results are missing")
		}
		return response, err
	}
	if request.WorkerName == "" || request.TaskID == "" || request.HostSessionID == "" {
		return response, fmt.Errorf("native worker requires worker_name, task_id and host_session_id")
	}
	var updated buildAttemptRecord
	if hooks.BeforeWrite != nil {
		hooks.BeforeWrite()
	}
	buildWorkerRunMutationMu.Lock()
	defer buildWorkerRunMutationMu.Unlock()
	err = store.UpdateJSONAtomically(attemptPath, &updated, func() error {
		if err := validate(updated); err != nil {
			return err
		}
		var dispatch *codexBuildDispatch
		for i := range updated.PlanManifest.Dispatches {
			d := &updated.PlanManifest.Dispatches[i]
			if d.Name == request.WorkerName && normalizedDispatchTaskID(*d) == request.TaskID {
				dispatch = d
				break
			}
		}
		if dispatch == nil {
			return fmt.Errorf("native assignment is not in the accepted manifest")
		}
		if strings.TrimSpace(dispatch.Name) == "" || strings.TrimSpace(dispatch.Caste) == "" || strings.TrimSpace(dispatch.TaskID) == "" || dispatch.ExecutionWave < 1 {
			return fmt.Errorf("native assignment is incomplete")
		}
		response.Dispatch = dispatch
		var worker *buildAttemptWorkerRun
		for i := range updated.WorkerRuns {
			run := &updated.WorkerRuns[i]
			if run.WorkerName == request.WorkerName && run.TaskID == request.TaskID {
				worker = run
				break
			}
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		if operation == "reserve" {
			if request.LaunchID != "" || request.ChildID != "" || request.Result != nil || request.DispatchSHA256 != "" || request.PromptSHA256 != "" || request.SourceEventID != "" || request.SourceEventSHA256 != "" {
				return fmt.Errorf("reserve cannot supply a launch, child or result")
			}
			workspace, err := filepath.EvalSymlinks(updated.PlanManifest.Root)
			if err != nil {
				return err
			}
			if request.Workspace != workspace || request.HostPermission != string(codex.PermissionWorkspaceWrite) || updated.PlanManifest.ParallelMode == "worktree" {
				return fmt.Errorf("native host must use the accepted shared workspace and inherited workspace_write permission; requested workspace or permission is unsupported")
			}
			permission, err := codex.ResolvePermissionProfile(dispatch.Caste, dispatch.PermissionProfile)
			if err != nil {
				return err
			}
			if permission.Name != codex.PermissionWorkspaceWrite || permission.Filesystem != codex.FilesystemWorkspaceWrite || len(permission.WriteScopes) != 0 {
				return fmt.Errorf("native host cannot enforce this requested permission profile")
			}
			if worker != nil {
				if err := validateCodexNativeSavedWorker(updated, *dispatch, *worker); err != nil {
					return err
				}
				if worker.Native == nil || worker.Native.HostSessionID != request.HostSessionID || worker.Native.Workspace != workspace {
					return fmt.Errorf("assignment already reserved by a different execution; inspect retained evidence, do not respawn")
				}
				response.Worker, response.Replay = worker, true
				return errCodexNativeReplay
			}
			if updated.Status != buildAttemptAwaiting && updated.Status != buildAttemptDispatching {
				return fmt.Errorf("attempt %s cannot reserve workers while %s", updated.ID, updated.Status)
			}
			if err := validateCodexNativeLaunchCurrency(updated); err != nil {
				return err
			}
			for _, predecessor := range updated.PlanManifest.Dispatches {
				if predecessor.ExecutionWave < dispatch.ExecutionWave {
					if _, ok := latestTerminalBuildWorkerRun(updated.WorkerRuns, predecessor.Name, normalizedDispatchTaskID(predecessor)); !ok {
						return fmt.Errorf("earlier execution wave is unfinished")
					}
				}
			}
			launch, err := codex.NewExecutionRunID()
			if err != nil {
				return err
			}
			prompt, err := codexNativeLaunchPrompt(*updated.PlanManifest, *dispatch, launch)
			if err != nil {
				return err
			}
			digest, err := jsonSHA256(dispatch)
			if err != nil {
				return err
			}
			native := &codexNativeWorkerBinding{SchemaVersion: 1, LaunchState: "reserved", HostSessionID: request.HostSessionID, DispatchSHA256: digest, PromptSHA256: lifecycleDigest([]byte(prompt)), PermissionProfile: permission, Workspace: workspace, WorkspaceRoot: workspace, Prompt: prompt}
			updated.WorkerRuns = append(updated.WorkerRuns, buildAttemptWorkerRun{ProviderRunID: launch, WorkerName: dispatch.Name, TaskID: request.TaskID, Caste: dispatch.Caste, Platform: codex.PlatformCodex, Status: buildWorkerDispatching, StartedAt: now, UpdatedAt: now, Native: native})
			response.Worker = &updated.WorkerRuns[len(updated.WorkerRuns)-1]
			response.LaunchAllowed = true
			updated.Status, updated.UpdatedAt = buildAttemptDispatching, now
			return nil
		}
		if worker == nil || worker.Native == nil {
			return fmt.Errorf("native worker has no reservation")
		}
		native := worker.Native
		if err := validateCodexNativeSavedWorker(updated, *dispatch, *worker); err != nil {
			return err
		}
		if (request.Workspace != "" && request.Workspace != native.Workspace) || (request.HostPermission != "" && request.HostPermission != string(native.PermissionProfile.Name)) {
			return fmt.Errorf("native workspace or permission does not match reservation")
		}
		if request.LaunchID != worker.ProviderRunID || request.HostSessionID != native.HostSessionID || request.DispatchSHA256 != native.DispatchSHA256 || request.PromptSHA256 != native.PromptSHA256 || request.ChildID == "" {
			return fmt.Errorf("native launch/session/dispatch/prompt identity does not match reservation")
		}
		response.Worker = worker
		switch operation {
		case "bind":
			if request.Result != nil || request.SourceEventID != "" || request.SourceEventSHA256 != "" {
				return fmt.Errorf("bind cannot submit a result")
			}
			if native.ChildID != "" {
				if native.ChildID != request.ChildID {
					return fmt.Errorf("native reservation is bound to a different child")
				}
				response.Replay = true
				return errCodexNativeReplay
			}
			if native.LaunchState != "reserved" {
				return fmt.Errorf("native launch is not reserved")
			}
			if err := validateCodexNativeLaunchCurrency(updated); err != nil {
				return err
			}
			release, err := codex.NewExecutionRunID()
			if err != nil {
				return err
			}
			native.ChildID, native.LaunchState = request.ChildID, "bound"
			native.Release = fmt.Sprintf("AETHER_NATIVE_RELEASE %s %s %s", worker.ProviderRunID, request.ChildID, release)
		case "record":
			if native.ChildID != request.ChildID || (native.LaunchState != "bound" && native.LaunchState != "terminal") {
				return fmt.Errorf("terminal result requires the exact bound child")
			}
			if request.Result == nil || strings.TrimSpace(request.Result.Summary) == "" {
				return fmt.Errorf("nonempty terminal result and summary are required")
			}
			eventHash, hashErr := hex.DecodeString(strings.TrimPrefix(request.SourceEventSHA256, "sha256:"))
			if request.SourceEventID == "" || hashErr != nil || len(eventHash) != 32 {
				return fmt.Errorf("terminal result requires source event identity and SHA-256")
			}
			result := *request.Result
			result.Status = strings.ToLower(strings.TrimSpace(result.Status))
			if result.Name != worker.WorkerName || result.Caste != worker.Caste || result.TaskID != worker.TaskID {
				return fmt.Errorf("native result worker/caste/task identity does not match reservation")
			}
			if result.Status == buildWorkerCompleted && result.Error != "" {
				return fmt.Errorf("completed native result cannot carry an error")
			}
			// Reject malformed relay data before making the terminal immutable.
			// The existing finalizer still decides whether its evidence earns credit.
			if err := codex.ValidateWorkerHandoff(result.Handoff); err != nil {
				return fmt.Errorf("native handoff: %w", err)
			}
			if result.Status == buildWorkerCompleted && codex.IsEmptyWorkerHandoff(result.Handoff) {
				return fmt.Errorf("completed native result requires a nonempty handoff")
			}
			for _, receipt := range result.TaskReceipts {
				if err := codex.ValidateWorkerHandoff(receipt.Handoff); err != nil {
					return fmt.Errorf("native task %s handoff: %w", receipt.TaskID, err)
				}
			}
			// Worker prose cannot supply provider measurements.
			if result.Usage != (codex.WorkerUsage{}) {
				return fmt.Errorf("native provider usage is unavailable through a submitted result")
			}
			if native.SourceEventID != "" && (native.SourceEventID != request.SourceEventID || native.SourceEventSHA256 != request.SourceEventSHA256) {
				return fmt.Errorf("native terminal source event conflicts with saved evidence")
			}
			// Exercise the same applicable completion semantics before freezing the
			// first result. Limit the projection to this dispatch; other workers
			// need not have finished yet. No journal bytes are mutated on refusal.
			preview := updated
			previewWorker := *worker
			previewWorker.ResultSHA256 = ""
			preview.WorkerRuns = []buildAttemptWorkerRun{previewWorker}
			previewManifest := *updated.PlanManifest
			previewManifest.Dispatches = []codexBuildDispatch{*dispatch}
			preview.PlanManifest = &previewManifest
			if err := applyBuildWorkerTerminal(&preview, &preview.WorkerRuns[0], &result, now); err != nil {
				return err
			}
			completion, ok := buildCompletionFromWorkerRuns(preview)
			if !ok {
				return fmt.Errorf("native result cannot form a terminal completion")
			}
			if violations := validateCompletionPacketSemantics(updated.PlanManifest.Root, completion); len(violations) != 0 {
				return fmt.Errorf("native terminal completion semantics: %v", violations)
			}
			if err := applyBuildWorkerTerminal(&updated, worker, &result, now); err != nil {
				if errors.Is(err, errCodexNativeReplay) {
					response.Replay = true
				}
				return err
			}
			native.SourceEventID, native.SourceEventSHA256, native.LaunchState = request.SourceEventID, request.SourceEventSHA256, "terminal"
			native.RawResult = append(json.RawMessage(nil), request.RawResult...)
			return nil
		default:
			return fmt.Errorf("unsupported native worker operation %q", operation)
		}
		worker.UpdatedAt, updated.UpdatedAt = now, now
		return nil
	})
	if errors.Is(err, errCodexNativeReplay) {
		response.Disposition = codexNativeReplayed
		err = nil
	} else if err == nil {
		response.Disposition = codexNativeAccepted
	}
	return response, err
}

func validateCodexNativeLaunchCurrency(record buildAttemptRecord) error {
	if record.Status != buildAttemptAwaiting && record.Status != buildAttemptDispatching {
		return fmt.Errorf("native attempt %s cannot launch while %s", record.ID, record.Status)
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return err
	}
	return validateBuildFinalizeStateStillCurrent(state, record.Phase)
}

func validateCodexNativeSavedWorker(record buildAttemptRecord, dispatch codexBuildDispatch, worker buildAttemptWorkerRun) error {
	native := worker.Native
	if native == nil {
		return fmt.Errorf("assignment belongs to an ordinary provider execution")
	}
	digest, err := jsonSHA256(dispatch)
	if err != nil {
		return err
	}
	permission, err := codex.ResolvePermissionProfile(dispatch.Caste, dispatch.PermissionProfile)
	if err != nil {
		return err
	}
	root, err := filepath.EvalSymlinks(record.PlanManifest.Root)
	if err != nil {
		return err
	}
	permissionDigest, _ := jsonSHA256(permission)
	savedPermissionDigest, _ := jsonSHA256(native.PermissionProfile)
	if native.SchemaVersion != 1 || native.HostSessionID == "" || worker.ProviderRunID == "" || worker.Caste != dispatch.Caste || worker.WorkerName != dispatch.Name || worker.TaskID != normalizedDispatchTaskID(dispatch) || native.DispatchSHA256 != digest || native.PromptSHA256 != lifecycleDigest([]byte(native.Prompt)) || strings.TrimSpace(native.Prompt) == "" || native.Workspace != root || (native.WorkspaceRoot != "" && native.WorkspaceRoot != root) || savedPermissionDigest != permissionDigest {
		return fmt.Errorf("native saved assignment/prompt/permission/workspace identity changed")
	}
	return nil
}

func codexNativeLaunchPrompt(manifest codexBuildManifest, dispatch codexBuildDispatch, launch string) (string, error) {
	brief := dispatch.Brief
	if dispatch.BriefPath != "" {
		path := filepath.Join(manifest.Root, filepath.FromSlash(dispatch.BriefPath))
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", fmt.Errorf("read native brief: %w", err)
		}
		root, err := filepath.EvalSymlinks(manifest.Root)
		if err != nil {
			return "", err
		}
		rel, err := filepath.Rel(root, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("native brief is outside accepted workspace")
		}
		raw, err := os.ReadFile(resolved)
		if err != nil {
			return "", err
		}
		brief = string(raw)
	}
	if strings.TrimSpace(brief) == "" {
		return "", fmt.Errorf("native assignment has no required brief")
	}
	prompt := fmt.Sprintf("You are %s (%s), assigned workspace %s. You are not alone; preserve others' edits.\nWAIT: do not read files, run checks or edit until the parent sends AETHER_NATIVE_RELEASE %s with your actual child ID and the runtime release token after binding. If no release arrives, remain waiting; do not do the job.\n", dispatch.Name, dispatch.Caste, manifest.Root, launch)
	prompt += manifest.ContextCapsule + "\n\n" + brief + "\n\n" + dispatch.SkillSection
	for _, line := range clarifiedIntentPromptRenderResult().Lines {
		if !strings.Contains(manifest.ContextCapsule, line) {
			prompt += "\n" + line
		}
	}
	prompt += fmt.Sprintf("\nReturn one JSON terminal result: name or ant_name=%q (if both are supplied they must agree), caste=%q, task_id=%q, status (code_written/completed/failed/blocked/timeout), summary, files_created, files_modified, tests_written, task_receipts, blockers, spawns, handoff; optional installed Builder tdd fields cycles_completed/tests_added/coverage_percent/all_passing are preserved without becoming provider telemetry. Each task receipt has task_id, status, summary, files_created, files_modified, tests_written, handoff. Every handoff uses %s. verification_status MUST be one enum value: pass, fail, partial, not_run, or unknown; put explanations in summary/known_failures, never in verification_status. Report only actual checks; omit usage. Do not stage, finalize, commit, recruit or launch helpers. Parent saves your terminal response.\n", dispatch.Name, dispatch.Caste, normalizedDispatchTaskID(dispatch), codex.HandoffFieldsSummary)
	return prompt, nil
}
