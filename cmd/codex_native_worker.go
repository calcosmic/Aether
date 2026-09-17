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
type codexNativeWorkerHooks struct {
	BeforeWrite        func()
	AfterCurrencyCheck func()
	AfterContextRender func()
	AfterTransition    func(codexNativeWorkerReceipt)
}

// Receipt identifies one accepted transition independently of later worker
// observations. Replays return these original values, never a new timestamp.
type codexNativeWorkerReceipt struct {
	ContextDeliveryID    string `json:"context_delivery_id,omitempty"`
	ContextPayloadSHA256 string `json:"context_payload_sha256,omitempty"`
	Operation            string `json:"operation"`
	LaunchID             string `json:"launch_id"`
	ChildID              string `json:"child_id,omitempty"`
	At                   string `json:"at"`
	DispatchSHA256       string `json:"dispatch_sha256"`
	PromptSHA256         string `json:"prompt_sha256"`
	SourceEventID        string `json:"source_event_id,omitempty"`
	SourceEventSHA256    string `json:"source_event_sha256,omitempty"`
	ResultSHA256         string `json:"result_sha256,omitempty"`
}

type codexNativeHostObservation struct {
	ContextDeliveryID string `json:"context_delivery_id,omitempty"`
	SchemaVersion     int    `json:"schema_version"`
	Status            string `json:"status"`
	ChildID           string `json:"child_id,omitempty"`
	ObservedAt        string `json:"observed_at"`
	Detail            string `json:"detail,omitempty"`
	SourceEventID     string `json:"source_event_id"`
	SourceEventSHA256 string `json:"source_event_sha256"`
}

type codexNativeWorkerState struct {
	ContextDeliveryIDs []string `json:"context_delivery_ids,omitempty"`
	WorkerName         string   `json:"worker_name"`
	TaskID             string   `json:"task_id"`
	LaunchID           string   `json:"launch_id"`
	ChildID            string   `json:"child_id,omitempty"`
	LaunchState        string   `json:"launch_state"`
	HostStatus         string   `json:"host_status"`
	Terminal           bool     `json:"terminal"`
	CancelRequested    bool     `json:"cancel_requested,omitempty"`
	ResultSHA256       string   `json:"result_sha256,omitempty"`
}

// Native provenance extends the existing worker journal, never its credit authority.
// Prompt is retained verbatim so replay never silently recomposes an assignment.
type codexNativeWorkerBinding struct {
	ContextDeliveries  []codexNativeContextReceipt  `json:"context_deliveries,omitempty"`
	SchemaVersion      int                          `json:"schema_version"`
	LaunchState        string                       `json:"launch_state"`
	BoundAt            string                       `json:"bound_at,omitempty"`
	Observations       []codexNativeHostObservation `json:"observations,omitempty"`
	HostSessionID      string                       `json:"host_session_id"`
	ChildID            string                       `json:"child_id,omitempty"`
	DispatchSHA256     string                       `json:"dispatch_sha256"`
	PromptSHA256       string                       `json:"prompt_sha256"`
	PermissionProfile  codex.PermissionProfile      `json:"permission_profile"`
	Workspace          string                       `json:"workspace"`
	WorkspaceRoot      string                       `json:"workspace_root,omitempty"`
	ContextDecisionIDs []string                     `json:"context_decision_ids,omitempty"`
	ContextDeliveryIDs []string                     `json:"context_delivery_ids,omitempty"`
	Prompt             string                       `json:"prompt"`
	Release            string                       `json:"release,omitempty"`
	SourceEventID      string                       `json:"source_event_id,omitempty"`
	SourceEventSHA256  string                       `json:"source_event_sha256,omitempty"`
	RawResult          json.RawMessage              `json:"raw_result,omitempty"`
}

type codexNativeWorkerRequest struct {
	RequireGovernedNesting bool                        `json:"require_governed_nesting,omitempty"`
	Question               *codexNativeQuestion        `json:"question,omitempty"`
	ContextDeliveryID      string                      `json:"context_delivery_id,omitempty"`
	ContextDelivery        *codexNativeContextDelivery `json:"context_delivery,omitempty"`
	ContextSend            *codexNativeContextSend     `json:"context_send,omitempty"`
	ObservationStatus      string                      `json:"observation_status,omitempty"`
	ObservedAt             string                      `json:"observed_at,omitempty"`
	ObservationDetail      string                      `json:"observation_detail,omitempty"`
	SchemaVersion          int                         `json:"schema_version"`
	Phase                  int                         `json:"phase"`
	ExecutionBinding       codex.ExecutionBinding      `json:"execution_binding"`
	WorkerName             string                      `json:"worker_name,omitempty"`
	TaskID                 string                      `json:"task_id,omitempty"`
	LaunchID               string                      `json:"launch_id,omitempty"`
	HostSessionID          string                      `json:"host_session_id,omitempty"`
	ChildID                string                      `json:"child_id,omitempty"`
	DispatchSHA256         string                      `json:"dispatch_sha256,omitempty"`
	PromptSHA256           string                      `json:"prompt_sha256,omitempty"`
	Workspace              string                      `json:"workspace,omitempty"`
	HostPermission         string                      `json:"host_permission,omitempty"`
	SourceEventID          string                      `json:"source_event_id,omitempty"`
	SourceEventSHA256      string                      `json:"source_event_sha256,omitempty"`
	Result                 *internalWorkerResult       `json:"result,omitempty"`
	RawResult              json.RawMessage             `json:"-"`
}

type codexNativeWorkerResponse struct {
	Decisions        []codexNativeDecisionView   `json:"decisions,omitempty"`
	ContextStatus    string                      `json:"context_status,omitempty"`
	ContextDelivery  *codexNativeContextDelivery `json:"context_delivery,omitempty"`
	Receipt          *codexNativeWorkerReceipt   `json:"receipt,omitempty"`
	WorkerStates     []codexNativeWorkerState    `json:"worker_states,omitempty"`
	Disposition      codexNativeDisposition      `json:"disposition,omitempty"`
	SchemaVersion    int                         `json:"schema_version"`
	ExecutionBinding codex.ExecutionBinding      `json:"execution_binding"`
	LaunchAllowed    bool                        `json:"launch_allowed"`
	Replay           bool                        `json:"replay,omitempty"`
	Dispatch         *codexBuildDispatch         `json:"dispatch,omitempty"`
	Worker           *buildAttemptWorkerRun      `json:"worker,omitempty"`
	Workers          []buildAttemptWorkerRun     `json:"workers,omitempty"`
	CompletionPath   string                      `json:"completion_path,omitempty"`
	Complete         bool                        `json:"complete,omitempty"`
}

func init() {
	command := &cobra.Command{Use: "codex-native-worker", Short: "Internal non-launching native worker journal bridge", Hidden: true}
	for _, operation := range []string{"reserve", "bind", "record", "stage", "inspect", "observe", "context", "question"} {
		operation := operation
		child := &cobra.Command{Use: operation, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			path, _ := cmd.Flags().GetString("request")
			var response codexNativeWorkerResponse
			var err error
			if cmd.Flags().Changed("phase") {
				if cmd.Flags().Changed("request") {
					err = fmt.Errorf("native --phase and --request are mutually exclusive")
				} else {
					phase, _ := cmd.Flags().GetInt("phase")
					response, err = runCodexNativeWorkerForPhase(operation, phase)
				}
			} else {
				response, err = runCodexNativeWorker(operation, path)
			}
			if err != nil {
				outputError(1, sanitizeInternalWorkerAdapterError(err.Error()), nil)
				return renderedErrorExit(1)
			}
			outputOK(response)
			return nil
		}}
		child.Flags().String("request", "", "Absolute regular JSON file under an aether-worker-request-* temporary directory")
		if operation == "inspect" || operation == "stage" {
			child.Flags().Int("phase", 0, "Use the exact current saved execution binding for this phase")
		}
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
	return executeCodexNativeWorkerRequest(operation, request, hooks)
}

// Phase-only recovery derives identity from the current journal without
// creating a temporary request file. Stage delegates to its existing authority.
func runCodexNativeWorkerForPhase(operation string, phase int) (codexNativeWorkerResponse, error) {
	if (operation != "inspect" && operation != "stage") || phase <= 0 {
		return codexNativeWorkerResponse{}, fmt.Errorf("native --phase requires inspect/stage and a positive phase")
	}
	_, record, ok := loadLatestBuildAttempt(phase)
	if !ok || record.PlanManifest == nil || record.PlanManifest.ExecutionBinding == nil {
		return codexNativeWorkerResponse{}, fmt.Errorf("native phase has no current saved binding")
	}
	request := codexNativeWorkerRequest{SchemaVersion: 1, Phase: phase, ExecutionBinding: *record.PlanManifest.ExecutionBinding}
	return executeCodexNativeWorkerRequest(operation, request, codexNativeWorkerHooks{})
}

func executeCodexNativeWorkerRequest(operation string, request codexNativeWorkerRequest, hooks codexNativeWorkerHooks) (codexNativeWorkerResponse, error) {
	if operation == "question" {
		return runCodexNativeQuestions(request, hooks)
	}
	if request.Question != nil {
		return codexNativeWorkerResponse{}, fmt.Errorf("native questions require the question operation")
	}
	response := codexNativeWorkerResponse{SchemaVersion: 1, ExecutionBinding: request.ExecutionBinding}
	var err error
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
		for _, worker := range current.WorkerRuns {
			if worker.Native != nil {
				response.WorkerStates = append(response.WorkerStates, projectCodexNativeWorkerState(worker))
			}
		}
		response.CompletionPath = current.CompletionPath
		response.Complete = current.CompletionPath != ""
		return response, nil
	}
	if operation == "stage" {
		if len(current.WorkerRuns) == 0 {
			return response, fmt.Errorf("native stage requires saved terminal workers")
		}
		for _, worker := range current.WorkerRuns {
			if !codexNativeWorkerIsTerminal(worker) {
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
	if operation != "observe" && (request.ObservationStatus != "" || request.ObservedAt != "" || request.ObservationDetail != "") {
		return response, fmt.Errorf("host observations require the observe operation")
	}
	if operation == "context" {
		response, err = readCodexNativeContext(current, request)
		if err != nil {
			return response, err
		}
		if hooks.AfterContextRender != nil {
			hooks.AfterContextRender()
		}
		if response.ContextStatus == "awaiting_delivery" {
			dispatch, worker, err := validateCodexNativeContextTarget(current, request)
			if err != nil {
				return codexNativeWorkerResponse{}, err
			}
			if err := validateCodexNativeContextLive(current, *worker); err != nil {
				return codexNativeWorkerResponse{}, err
			}
			latest, err := composeCodexNativeContextDelivery(current, *dispatch, *worker, response.ContextDelivery.DecisionIDs)
			if err != nil {
				return codexNativeWorkerResponse{}, err
			}
			if latest == nil || latest.DeliveryID != response.ContextDelivery.DeliveryID {
				return codexNativeWorkerResponse{}, fmt.Errorf("native answer changed while rendering; reread current context")
			}
		}
		if err := validateCodexNativeContextScope(*current.PlanManifest); err != nil {
			return codexNativeWorkerResponse{}, err
		}
		// Recheck currency and journal identity after rendering; reads never
		// take a mutation session or create delivery acknowledgements.
		if err := validate(current); err != nil {
			return codexNativeWorkerResponse{}, err
		}
		_, latest, ok := loadLatestBuildAttempt(request.Phase)
		firstDigest, _ := jsonSHA256(current)
		latestDigest, _ := jsonSHA256(latest)
		if !ok || firstDigest != latestDigest {
			return codexNativeWorkerResponse{}, fmt.Errorf("native context changed during inspection; reread current assignment")
		}
		return response, nil
	}
	if operation != "observe" && (request.ContextDeliveryID != "" || request.ContextDelivery != nil || request.ContextSend != nil) {
		return response, fmt.Errorf("context evidence requires context/observe operation")
	}
	var updated buildAttemptRecord
	if hooks.BeforeWrite != nil {
		hooks.BeforeWrite()
	}
	err = withPlanningMutationSession(buildAttemptWorkspaceRoot(), "codex-native-worker", func(_ *planningMutationSession) error {
		buildWorkerRunMutationMu.Lock()
		defer buildWorkerRunMutationMu.Unlock()
		return store.UpdateJSONAtomically(attemptPath, &updated, func() error {
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
			// Auxiliary reviewers use the same derived task identity as manifest
			// lookup and completion accounting; their raw TaskID is intentionally empty.
			if strings.TrimSpace(dispatch.Name) == "" || strings.TrimSpace(dispatch.Caste) == "" || normalizedDispatchTaskID(*dispatch) == "" || dispatch.ExecutionWave < 1 {
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
				permission, err := codex.ResolvePermissionProfile(dispatch.Caste, dispatch.PermissionProfile)
				if err != nil {
					return err
				}
				if err := codex.ValidateNativeWorkerRequirements(codex.NativeWorkerRequirements{Profile: permission, HostPermission: request.HostPermission, Workspace: request.Workspace, AcceptedWorkspace: workspace, ParallelMode: updated.PlanManifest.ParallelMode, RequireGovernedNesting: request.RequireGovernedNesting}); err != nil {
					return err
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
				if hooks.AfterCurrencyCheck != nil {
					hooks.AfterCurrencyCheck()
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
				if err := validateCodexNativeContextScope(*updated.PlanManifest); err != nil {
					return err
				}
				prompt, err := composeCodexNativePrompt(*updated.PlanManifest, *dispatch, launch)
				if err != nil {
					return err
				}
				digest, err := jsonSHA256(dispatch)
				if err != nil {
					return err
				}
				native := &codexNativeWorkerBinding{SchemaVersion: 1, LaunchState: "reserved", HostSessionID: request.HostSessionID, DispatchSHA256: digest, PromptSHA256: prompt.SHA256, PermissionProfile: permission, Workspace: workspace, WorkspaceRoot: workspace, Prompt: prompt.Prompt, ContextDecisionIDs: prompt.DecisionIDs}
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
			if request.RequireGovernedNesting {
				return fmt.Errorf("native host cannot provide requested Aether-governed nesting; no release authorized")
			}
			if (request.Workspace != "" && request.Workspace != native.Workspace) || (request.HostPermission != "" && request.HostPermission != string(native.PermissionProfile.Name)) {
				return fmt.Errorf("native workspace or permission does not match reservation")
			}
			if request.LaunchID != worker.ProviderRunID || request.HostSessionID != native.HostSessionID || request.DispatchSHA256 != native.DispatchSHA256 || request.PromptSHA256 != native.PromptSHA256 || (request.ChildID == "" && operation != "observe") {
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
				if native.LaunchState != "reserved" && native.LaunchState != "launch_unresolved" {
					return fmt.Errorf("native launch is not reserved")
				}
				if err := validateCodexNativeLaunchCurrency(updated); err != nil {
					return err
				}
				if hooks.AfterCurrencyCheck != nil {
					hooks.AfterCurrencyCheck()
				}
				release, err := codex.NewExecutionRunID()
				if err != nil {
					return err
				}
				native.ChildID, native.LaunchState = request.ChildID, "bound"
				native.BoundAt = now
				native.Release = fmt.Sprintf("AETHER_NATIVE_RELEASE %s %s %s", worker.ProviderRunID, request.ChildID, release)
			case "record":
				if native.ChildID != request.ChildID || (native.LaunchState != "bound" && native.LaunchState != "terminal") {
					return fmt.Errorf("terminal result requires the exact bound child")
				}
				return recordCodexNativeTerminal(&updated, worker, *dispatch, request, now)
			case "observe":
				receipt, err := observeCodexNativeWorker(&updated, worker, *dispatch, request, now)
				response.Receipt = receipt
				if receipt != nil && receipt.ContextDeliveryID != "" {
					response.ContextStatus = "delivered"
					response.ContextDelivery = request.ContextDelivery
				}
				return err
			default:
				return fmt.Errorf("unsupported native worker operation %q", operation)
			}
			worker.UpdatedAt, updated.UpdatedAt = now, now
			return nil
		})
	})
	if errors.Is(err, errCodexNativeReplay) {
		response.Replay = true
		response.Disposition = codexNativeReplayed
		err = nil
	} else if err == nil {
		response.Disposition = codexNativeAccepted
	}
	if err == nil && response.Worker != nil {
		if response.Receipt == nil {
			receipt := codexNativeTransitionReceipt(operation, *response.Worker)
			response.Receipt = &receipt
		}
		if !response.Replay {
			emitCodexNativeTransition(operation, *response.Dispatch, *response.Worker)
			if hooks.AfterTransition != nil {
				hooks.AfterTransition(*response.Receipt)
			}
		}
	}
	return response, err
}

func canonicalCodexNativeEventHash(eventID, digest string) (string, error) {
	decoded, err := hex.DecodeString(strings.TrimPrefix(digest, "sha256:"))
	if strings.TrimSpace(eventID) == "" || err != nil || len(decoded) != 32 {
		return "", fmt.Errorf("native host evidence requires source event identity and SHA-256")
	}
	return hex.EncodeToString(decoded), nil
}

func codexNativeWorkerIsTerminal(worker buildAttemptWorkerRun) bool {
	return worker.Native != nil && (worker.Native.LaunchState == "terminal" || worker.Native.LaunchState == "no_launch") && worker.Result != nil && worker.ResultSHA256 != ""
}

func projectCodexNativeWorkerState(worker buildAttemptWorkerRun) codexNativeWorkerState {
	native := worker.Native
	state := codexNativeWorkerState{WorkerName: worker.WorkerName, TaskID: worker.TaskID, LaunchID: worker.ProviderRunID, ChildID: native.ChildID, LaunchState: native.LaunchState, Terminal: codexNativeWorkerIsTerminal(worker), ResultSHA256: worker.ResultSHA256, HostStatus: "launch_unresolved"}
	if native.ChildID != "" {
		state.HostStatus = "unavailable"
	}
	state.ContextDeliveryIDs = append([]string(nil), native.ContextDeliveryIDs...)
	for _, observation := range native.Observations {
		if observation.Status != "context_delivered" {
			state.HostStatus = observation.Status
		}
	}
	for _, observation := range native.Observations {
		if observation.Status == "cancel_requested" {
			state.CancelRequested = true
		}
	}
	if state.CancelRequested {
		state.HostStatus = "cancel_requested"
	}
	if state.Terminal {
		state.CancelRequested = false
		state.HostStatus = worker.Status
	}
	return state
}

func codexNativeTransitionReceipt(operation string, worker buildAttemptWorkerRun) codexNativeWorkerReceipt {
	native := worker.Native
	receipt := codexNativeWorkerReceipt{Operation: operation, LaunchID: worker.ProviderRunID, DispatchSHA256: native.DispatchSHA256, PromptSHA256: native.PromptSHA256, At: worker.StartedAt}
	if operation != "reserve" {
		receipt.ChildID = native.ChildID
		receipt.At = native.BoundAt
	}
	if operation == "record" {
		receipt.At, receipt.SourceEventID, receipt.SourceEventSHA256, receipt.ResultSHA256 = worker.CompletedAt, native.SourceEventID, native.SourceEventSHA256, worker.ResultSHA256
	}
	return receipt
}

func emitCodexNativeTransition(operation string, assignment codexBuildDispatch, worker buildAttemptWorkerRun) {
	dispatch := codex.WorkerDispatch{WorkerName: worker.WorkerName, TaskID: worker.TaskID, Caste: worker.Caste, TaskBrief: assignment.Task, ProviderRunID: worker.ProviderRunID}
	switch {
	case operation == "bind":
		emitCodexDispatchWorkerStarted(dispatch, assignment.ExecutionWave)
	case codexNativeWorkerIsTerminal(worker):
		emitCodexDispatchWorkerFinished(dispatch, codex.DispatchResult{Status: worker.Status})
	case operation == "observe" && len(worker.Native.Observations) > 0 && worker.Native.Observations[len(worker.Native.Observations)-1].Status == "running":
		emitCodexDispatchWorkerRunning(dispatch, assignment.ExecutionWave, "host confirmed activity")
	}
}

func observeCodexNativeWorker(record *buildAttemptRecord, worker *buildAttemptWorkerRun, dispatch codexBuildDispatch, request codexNativeWorkerRequest, now string) (*codexNativeWorkerReceipt, error) {
	if request.ObservationStatus == "context_delivered" {
		return observeCodexNativeContext(record, worker, dispatch, request, now)
	}
	if request.ContextDeliveryID != "" || request.ContextDelivery != nil || request.ContextSend != nil {
		return nil, fmt.Errorf("native context evidence requires context_delivered observation")
	}
	native := worker.Native
	digest, err := canonicalCodexNativeEventHash(request.SourceEventID, request.SourceEventSHA256)
	if err != nil {
		return nil, err
	}
	at, err := time.Parse(time.RFC3339Nano, request.ObservedAt)
	if err != nil {
		return nil, fmt.Errorf("native observation requires an actual observed_at timestamp")
	}
	observation := codexNativeHostObservation{SchemaVersion: 1, Status: request.ObservationStatus, ChildID: request.ChildID, ObservedAt: at.UTC().Format(time.RFC3339Nano), Detail: request.ObservationDetail, SourceEventID: request.SourceEventID, SourceEventSHA256: digest}
	receipt := codexNativeTransitionReceipt("observe", *worker)
	receipt.ChildID = observation.ChildID
	receipt.At, receipt.SourceEventID, receipt.SourceEventSHA256 = observation.ObservedAt, observation.SourceEventID, digest
	for _, saved := range native.Observations {
		if saved.Status == "cancel_requested" && observation.Status == "running" {
			return nil, fmt.Errorf("native cancellation remains pending")
		}
		if saved.SourceEventID != observation.SourceEventID {
			continue
		}
		if saved != observation {
			return nil, fmt.Errorf("native host event conflicts with saved observation")
		}
		if saved.Status == "cancelled" || saved.Status == "no_launch" {
			receipt.ResultSHA256 = worker.ResultSHA256
		}
		if request.Result != nil {
			normalized, err := normalizedBuildWorkerResult(request.Result)
			if err != nil {
				return nil, err
			}
			resultDigest, err := jsonSHA256(normalized)
			if err != nil || resultDigest != worker.ResultSHA256 {
				return nil, fmt.Errorf("native observation result conflicts with terminal")
			}
		}
		return &receipt, errCodexNativeReplay
	}
	if codexNativeWorkerIsTerminal(*worker) {
		return nil, fmt.Errorf("native terminal cannot accept a later observation")
	}
	if native.ChildID != request.ChildID {
		return nil, fmt.Errorf("native observation must name the exact bound child")
	}
	if len(native.Observations) > 0 {
		last := native.Observations[len(native.Observations)-1]
		previous, _ := time.Parse(time.RFC3339Nano, last.ObservedAt)
		if !at.After(previous) {
			return nil, fmt.Errorf("native observation arrived out of order")
		}
		if last.Status == "cancel_requested" && observation.Status == "running" {
			return nil, fmt.Errorf("native cancellation remains pending")
		}
	}
	terminal := false
	switch observation.Status {
	case "running", "cancel_requested", "cancelled":
		if native.ChildID == "" {
			return nil, fmt.Errorf("native observation requires a bound child")
		}
		terminal = observation.Status == "cancelled"
	case "launch_unresolved":
		if native.ChildID != "" {
			return nil, fmt.Errorf("a known child cannot become an unresolved launch")
		}
	case "unavailable":
	case "no_launch":
		if native.ChildID != "" || request.Result != nil {
			return nil, fmt.Errorf("no-launch evidence requires an unbound reservation without work")
		}
		terminal = true
	default:
		return nil, fmt.Errorf("unsupported native observation %q", observation.Status)
	}
	if !terminal && request.Result != nil {
		return nil, fmt.Errorf("nonterminal host observation cannot submit a result")
	}
	if terminal {
		if request.Result == nil {
			request.Result = &internalWorkerResult{Name: worker.WorkerName, Caste: worker.Caste, TaskID: worker.TaskID, Status: buildWorkerCancelled, Summary: "Host confirmed " + observation.Status, Error: observation.Detail}
		}
		if strings.ToLower(strings.TrimSpace(request.Result.Status)) != buildWorkerCancelled {
			return nil, fmt.Errorf("cancellation observation requires a cancelled result")
		}
		if err := recordCodexNativeTerminal(record, worker, dispatch, request, now); err != nil {
			return nil, err
		}
		if observation.Status == "no_launch" {
			native.LaunchState = "no_launch"
		}
		receipt.ResultSHA256 = worker.ResultSHA256
	} else if native.ChildID == "" {
		native.LaunchState = "launch_unresolved"
	}
	native.Observations = append(native.Observations, observation)
	worker.UpdatedAt, record.UpdatedAt = now, now
	return &receipt, nil
}

// recordCodexNativeTerminal receives an exact assignment and runs only inside
// the attempt mutation. It preserves all failure evidence before aggregation.
func recordCodexNativeTerminal(record *buildAttemptRecord, worker *buildAttemptWorkerRun, dispatch codexBuildDispatch, request codexNativeWorkerRequest, now string) error {
	native := worker.Native
	if request.Result == nil || strings.TrimSpace(request.Result.Summary) == "" {
		return fmt.Errorf("nonempty terminal result and summary are required")
	}
	eventHash, err := canonicalCodexNativeEventHash(request.SourceEventID, request.SourceEventSHA256)
	if err != nil {
		return err
	}
	request.SourceEventSHA256 = eventHash
	result, err := normalizedBuildWorkerResult(request.Result)
	if err != nil {
		return err
	}
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
	covered := make(map[string]bool)
	for _, taskID := range dispatchCoveredTaskIDs(dispatch) {
		covered[taskID] = true
	}
	seenReceipts := make(map[string]bool)
	for _, receipt := range result.TaskReceipts {
		if !covered[receipt.TaskID] || seenReceipts[receipt.TaskID] {
			return fmt.Errorf("native task receipt is unassigned or duplicated: %s", receipt.TaskID)
		}
		seenReceipts[receipt.TaskID] = true
		if err := codex.ValidateWorkerHandoff(receipt.Handoff); err != nil {
			return fmt.Errorf("native task %s handoff: %w", receipt.TaskID, err)
		}
	}
	// Worker prose cannot supply provider measurements.
	if result.Usage != (codex.WorkerUsage{}) {
		return fmt.Errorf("native provider usage is uncollected; submitted result usage is not a trusted provider measurement")
	}
	if native.SourceEventID != "" && (native.SourceEventID != request.SourceEventID || native.SourceEventSHA256 != request.SourceEventSHA256) {
		return fmt.Errorf("native terminal source event conflicts with saved evidence")
	}
	// An immutable terminal replay must not depend on today's filesystem.
	if worker.ResultSHA256 != "" {
		return applyBuildWorkerTerminal(record, worker, result, now)
	}
	// Exercise the same applicable completion semantics before freezing the
	// first result. Limit the projection to this dispatch; other workers
	// need not have finished yet. No journal bytes are mutated on refusal.
	preview := *record
	previewWorker := *worker
	previewWorker.ResultSHA256 = ""
	preview.WorkerRuns = []buildAttemptWorkerRun{previewWorker}
	previewManifest := *record.PlanManifest
	previewManifest.Dispatches = []codexBuildDispatch{dispatch}
	preview.PlanManifest = &previewManifest
	if err := applyBuildWorkerTerminal(&preview, &preview.WorkerRuns[0], result, now); err != nil {
		return err
	}
	completion, ok := buildCompletionFromWorkerRuns(preview)
	if !ok {
		return fmt.Errorf("native result cannot form a terminal completion")
	}
	// Cancellation stays cancelled in the journal. Use the established
	// interrupted vocabulary only for this validation projection; Plan 05
	// owns its final aggregate mapping.
	if result.Status == buildWorkerCancelled {
		completion.Dispatches[0].Status = "interrupted"
	}
	if violations := validateCompletionPacketSemantics(record.PlanManifest.Root, completion); len(violations) != 0 {
		return fmt.Errorf("native terminal completion semantics: %v", violations)
	}
	if err := applyBuildWorkerTerminal(record, worker, result, now); err != nil {
		return err
	}
	native.SourceEventID, native.SourceEventSHA256, native.LaunchState = request.SourceEventID, request.SourceEventSHA256, "terminal"
	native.RawResult = append(json.RawMessage(nil), request.RawResult...)
	return nil
}

func validateCodexNativeLaunchCurrency(record buildAttemptRecord) error {
	if record.Status != buildAttemptAwaiting && record.Status != buildAttemptDispatching {
		return fmt.Errorf("native attempt %s cannot launch while %s", record.ID, record.Status)
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return err
	}
	if err := validateBuildFinalizeStateStillCurrent(state, record.Phase); err != nil {
		return err
	}
	return validateCodexNativeAttemptState(record, state)
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
