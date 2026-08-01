package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/spf13/cobra"
)

// internalWorkerAdapterSchemaVersion stays 1: the Preflight field on
// internalWorkerAdapterResponse below is additive and optional, and the TS
// host asserts schema_version === 1.
const (
	internalWorkerAdapterSchemaVersion = 1
	internalWorkerRequestMaxBytes      = 2 << 20
	internalWorkerTimeoutMax           = 60 * time.Minute
)

var internalWorkerSecretPattern = regexp.MustCompile(`(?i)(sk-|gh[pousr]_|github_pat_|npm_)[A-Za-z0-9_-]+`)

type internalWorkerDispatchRequest struct {
	SchemaVersion     int                     `json:"schema_version"`
	Workflow          string                  `json:"workflow,omitempty"`
	Phase             int                     `json:"phase,omitempty"`
	AgentName         string                  `json:"agent_name,omitempty"`
	Caste             string                  `json:"caste"`
	WorkerName        string                  `json:"worker_name"`
	TaskID            string                  `json:"task_id,omitempty"`
	Task              string                  `json:"task,omitempty"`
	TaskBrief         string                  `json:"task_brief,omitempty"`
	ContextCapsule    string                  `json:"context_capsule,omitempty"`
	SkillSection      string                  `json:"skill_section,omitempty"`
	HiveSection       string                  `json:"hive_section,omitempty"`
	PheromoneSection  string                  `json:"pheromone_section,omitempty"`
	HandoffSection    string                  `json:"handoff_section,omitempty"`
	TimeoutMS         int64                   `json:"timeout_ms,omitempty"`
	PermissionProfile codex.PermissionProfile `json:"permission_profile"`
	ExecutionBinding  *codex.ExecutionBinding `json:"execution_binding,omitempty"`
}

type internalWorkerResult struct {
	Name          string                     `json:"name"`
	Caste         string                     `json:"caste"`
	TaskID        string                     `json:"task_id,omitempty"`
	Status        string                     `json:"status"`
	Summary       string                     `json:"summary,omitempty"`
	FilesCreated  []string                   `json:"files_created,omitempty"`
	FilesModified []string                   `json:"files_modified,omitempty"`
	TestsWritten  []string                   `json:"tests_written,omitempty"`
	Artifacts     map[string]json.RawMessage `json:"artifacts,omitempty"`
	ScoutReport   json.RawMessage            `json:"scout_report,omitempty"`
	ToolCount     int                        `json:"tool_count,omitempty"`
	Blockers      []string                   `json:"blockers,omitempty"`
	Spawns        []string                   `json:"spawns,omitempty"`
	Duration      float64                    `json:"duration,omitempty"`
	Error         string                     `json:"error,omitempty"`
	Handoff       codex.WorkerHandoff        `json:"handoff,omitempty"`
}

type internalWorkerAdapterResponse struct {
	SchemaVersion       int                       `json:"schema_version"`
	ExecutionOwner      string                    `json:"execution_owner"`
	Platform            codex.Platform            `json:"platform"`
	PlatformContract    codex.PlatformContract    `json:"platform_contract"`
	Availability        codex.AvailabilityStatus  `json:"availability"`
	ProviderDiagnostics string                    `json:"provider_diagnostics,omitempty"`
	PermissionDecision  *codex.PermissionDecision `json:"permission_decision,omitempty"`
	ExecutionBinding    *codex.ExecutionBinding   `json:"execution_binding,omitempty"`
	ProviderRunID       string                    `json:"provider_run_id,omitempty"`
	Worker              *internalWorkerResult     `json:"worker,omitempty"`
	Preflight           *preflightOutcome         `json:"preflight,omitempty"`
}

var internalWorkerAdapterCmd = &cobra.Command{
	Use:    "internal-worker-adapter",
	Short:  "Internal Go-owned worker provider adapter boundary",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		preflight, _ := cmd.Flags().GetBool("preflight")
		simulate, _ := cmd.Flags().GetBool("simulate")
		requestPath, _ := cmd.Flags().GetString("request-file")

		response, err := runInternalWorkerAdapter(cmd.Context(), requestPath, preflight, simulate)
		if err != nil {
			outputError(1, sanitizeInternalWorkerAdapterError(err.Error()), nil)
			return renderedErrorExit(1)
		}
		outputOK(response)
		return nil
	},
}

func init() {
	internalWorkerAdapterCmd.Flags().String("request-file", "", "Approved temporary JSON worker request")
	internalWorkerAdapterCmd.Flags().Bool("preflight", false, "Validate the selected provider without dispatching a worker")
	internalWorkerAdapterCmd.Flags().Bool("simulate", false, "Use the deterministic test-only adapter")
	rootCmd.AddCommand(internalWorkerAdapterCmd)
}

func runInternalWorkerAdapter(ctx context.Context, requestPath string, preflight, simulate bool) (internalWorkerAdapterResponse, error) {
	root, err := os.Getwd()
	if err != nil {
		return internalWorkerAdapterResponse{}, fmt.Errorf("resolve worker root: %w", err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return internalWorkerAdapterResponse{}, fmt.Errorf("resolve absolute worker root: %w", err)
	}

	invoker, err := internalWorkerInvoker(ctx, simulate)
	if err != nil {
		return internalWorkerAdapterResponse{}, err
	}
	platform := codex.PlatformFromInvoker(invoker)
	contract, ok := codex.PlatformContractFor(platform)
	if !ok {
		return internalWorkerAdapterResponse{}, fmt.Errorf("selected provider %q has no Aether platform contract", platform)
	}

	availability := internalWorkerAvailability(ctx, invoker)
	response := internalWorkerAdapterResponse{
		SchemaVersion:       internalWorkerAdapterSchemaVersion,
		ExecutionOwner:      "go-adapter",
		Platform:            platform,
		PlatformContract:    contract,
		Availability:        availability,
		ProviderDiagnostics: dispatchProviderDiagnostics(invoker),
	}
	if !availability.Available {
		return response, dispatchUnavailableError(invoker)
	}

	if preflight {
		status := availability
		if provider, ok := invoker.(codex.WorkerProviderPreflighter); ok {
			var outcome preflightOutcome
			status, outcome = gatedProviderPreflight(ctx, provider, platform, root, time.Now())
			if outcome.Source != "" && outcome.Notice != "" {
				response.Preflight = &outcome
			}
		}
		response.Availability = status
		if !status.Available {
			return response, &workerProviderPreflightError{status: status}
		}
		return response, nil
	}

	request, err := loadInternalWorkerRequest(requestPath)
	if err != nil {
		return response, err
	}
	if err := validateInternalWorkerExecutionBinding(root, request); err != nil {
		return response, err
	}
	config, err := internalWorkerConfig(root, invoker, request)
	if err != nil {
		return response, err
	}
	permission, err := codex.ResolvePermissionDecision(platform, config.Caste, config.PermissionProfile)
	if err != nil {
		response.PermissionDecision = &permission
		return response, err
	}
	response.PermissionDecision = &permission
	providerRunID, err := codex.NewExecutionRunID()
	if err != nil {
		return response, err
	}
	config.ProviderRunID = providerRunID
	response.ProviderRunID = providerRunID
	if request.ExecutionBinding != nil {
		binding := *request.ExecutionBinding
		config.ExecutionBinding = &binding
		response.ExecutionBinding = &binding
	}
	if strings.EqualFold(strings.TrimSpace(request.Workflow), "build") && request.ExecutionBinding != nil {
		cached, err := beginBuildAttemptWorkerRun(request.Phase, *request.ExecutionBinding, request, providerRunID, platform)
		if err != nil {
			return response, err
		}
		if cachedResult := cachedInternalWorkerResult(cached); cachedResult != nil {
			response.ProviderRunID = cached.ProviderRunID
			response.Worker = cachedResult
			return response, nil
		}
	}
	observer := codex.WorkerProgressObserver(nil)
	if strings.EqualFold(strings.TrimSpace(request.Workflow), "build") && request.ExecutionBinding != nil {
		observer = func(event codex.WorkerProgressEvent) {
			if event.ProcessID > 0 {
				_ = recordBuildAttemptWorkerProcess(request.Phase, *request.ExecutionBinding, providerRunID, event.ProcessID)
			}
		}
	}
	result, invokeErr := invokeInternalWorker(ctx, invoker, config, observer)
	if err := validateInternalWorkerResult(request, &result, invokeErr); err != nil {
		if request.ExecutionBinding != nil && strings.EqualFold(strings.TrimSpace(request.Workflow), "build") {
			failed := &internalWorkerResult{
				Name:   strings.TrimSpace(request.WorkerName),
				Caste:  strings.TrimSpace(request.Caste),
				TaskID: effectiveInternalWorkerTaskID(request),
				Status: buildWorkerFailed,
				Error:  sanitizeInternalWorkerAdapterError(err.Error()),
			}
			_ = recordBuildAttemptWorkerTerminal(request.Phase, *request.ExecutionBinding, providerRunID, failed)
		}
		return response, err
	}
	response.Worker = mapInternalWorkerResult(result, invokeErr)
	if request.ExecutionBinding != nil && strings.EqualFold(strings.TrimSpace(request.Workflow), "build") {
		if err := recordBuildAttemptWorkerTerminal(request.Phase, *request.ExecutionBinding, providerRunID, response.Worker); err != nil {
			return response, err
		}
		if _, _, err := stageBuildAttemptCompletionFromWorkerRuns(request.Phase, *request.ExecutionBinding); err != nil {
			return response, fmt.Errorf("stage terminal build completion: %w", err)
		}
	}
	return response, nil
}

func invokeInternalWorker(ctx context.Context, invoker codex.WorkerInvoker, config codex.WorkerConfig, observer codex.WorkerProgressObserver) (codex.WorkerResult, error) {
	if progressInvoker, ok := invoker.(codex.ProgressAwareWorkerInvoker); ok {
		return progressInvoker.InvokeWithProgress(ctx, config, observer)
	}
	return invoker.Invoke(ctx, config)
}

func validateInternalWorkerExecutionBinding(root string, request internalWorkerDispatchRequest) error {
	workflow := strings.ToLower(strings.TrimSpace(request.Workflow))
	if workflow != "build" {
		if request.ExecutionBinding != nil {
			return codex.ValidateExecutionWorkspace(root, *request.ExecutionBinding)
		}
		return nil
	}
	if request.Phase < 1 {
		return fmt.Errorf("build worker request requires phase")
	}
	if request.ExecutionBinding == nil {
		return fmt.Errorf("build worker request requires execution_binding")
	}
	_, record, ok := loadLatestBuildAttempt(request.Phase)
	if !ok {
		return fmt.Errorf("build worker execution binding has no durable attempt")
	}
	if err := validateBuildExecutionBinding(record, *request.ExecutionBinding, record.ManifestSHA256, false); err != nil {
		return err
	}
	if record.Status != buildAttemptAwaiting && record.Status != buildAttemptDispatching {
		return fmt.Errorf("build attempt %s is %s and cannot dispatch workers", record.ID, record.Status)
	}
	return nil
}

func internalWorkerInvoker(ctx context.Context, simulate bool) (codex.WorkerInvoker, error) {
	if simulate {
		return &codex.FakeInvoker{}, nil
	}
	if value := strings.ToLower(strings.TrimSpace(os.Getenv("AETHER_CODEX_REAL_DISPATCH"))); value == "0" || value == "false" || value == "fake" {
		return nil, fmt.Errorf("synthetic worker dispatch is disabled at the production adapter boundary; use explicit --simulate only in tests")
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("AETHER_WORKER_PLATFORM")), "fake") {
		return nil, fmt.Errorf("AETHER_WORKER_PLATFORM=fake is test-only and cannot select a production worker provider")
	}
	invoker := codex.SelectPlatformInvoker(ctx)
	if invoker == nil || !invoker.IsAvailable(ctx) {
		return nil, dispatchUnavailableError(invoker)
	}
	return invoker, nil
}

func internalWorkerAvailability(ctx context.Context, invoker codex.WorkerInvoker) codex.AvailabilityStatus {
	if reporter, ok := invoker.(interface {
		Availability(context.Context) codex.AvailabilityStatus
	}); ok {
		return reporter.Availability(ctx)
	}
	platform := codex.PlatformFromInvoker(invoker)
	return codex.AvailabilityStatus{
		Platform:  platform,
		Binary:    string(platform),
		Available: invoker != nil && invoker.IsAvailable(ctx),
		Category:  codex.AvailabilityCategoryAvailable,
	}
}

func loadInternalWorkerRequest(path string) (internalWorkerDispatchRequest, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return internalWorkerDispatchRequest{}, fmt.Errorf("flag --request-file is required for worker dispatch")
	}
	if err := validateFinalizerCompletionFilePath(path); err != nil {
		return internalWorkerDispatchRequest{}, fmt.Errorf("%s", strings.ReplaceAll(err.Error(), "completion file", "worker request file"))
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return internalWorkerDispatchRequest{}, fmt.Errorf("resolve worker request file: %w", err)
	}
	tmp, err := filepath.Abs(os.TempDir())
	if err != nil {
		return internalWorkerDispatchRequest{}, fmt.Errorf("resolve temporary directory: %w", err)
	}
	resolvedTmp, err := filepath.EvalSymlinks(tmp)
	if err != nil {
		return internalWorkerDispatchRequest{}, fmt.Errorf("resolve temporary directory symlinks: %w", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return internalWorkerDispatchRequest{}, fmt.Errorf("resolve worker request file symlinks: %w", err)
	}
	rel, err := filepath.Rel(resolvedTmp, resolvedPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return internalWorkerDispatchRequest{}, fmt.Errorf("worker request file must be under the system temporary directory")
	}
	if !strings.HasPrefix(filepath.Base(filepath.Dir(resolvedPath)), "aether-worker-request-") {
		return internalWorkerDispatchRequest{}, fmt.Errorf("worker request file must use an aether-worker-request-* temporary directory")
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return internalWorkerDispatchRequest{}, fmt.Errorf("read worker request file metadata: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return internalWorkerDispatchRequest{}, fmt.Errorf("worker request file must be a regular non-symlink file")
	}
	if info.Size() > internalWorkerRequestMaxBytes {
		return internalWorkerDispatchRequest{}, fmt.Errorf("worker request file exceeds %d bytes", internalWorkerRequestMaxBytes)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return internalWorkerDispatchRequest{}, fmt.Errorf("read worker request file: %w", err)
	}
	var request internalWorkerDispatchRequest
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return internalWorkerDispatchRequest{}, fmt.Errorf("parse worker request file: %w", err)
	}
	if request.SchemaVersion != internalWorkerAdapterSchemaVersion {
		return internalWorkerDispatchRequest{}, fmt.Errorf("worker request schema_version must be %d", internalWorkerAdapterSchemaVersion)
	}
	return request, nil
}

func internalWorkerConfig(root string, invoker codex.WorkerInvoker, request internalWorkerDispatchRequest) (codex.WorkerConfig, error) {
	request.Caste = strings.TrimSpace(request.Caste)
	request.WorkerName = strings.TrimSpace(request.WorkerName)
	request.TaskID = strings.TrimSpace(request.TaskID)
	request.Task = strings.TrimSpace(request.Task)
	request.TaskBrief = strings.TrimSpace(request.TaskBrief)
	if request.Caste == "" || request.WorkerName == "" {
		return codex.WorkerConfig{}, fmt.Errorf("worker request requires caste and worker_name")
	}
	if request.TaskBrief == "" {
		request.TaskBrief = request.Task
	}
	if request.TaskBrief == "" {
		return codex.WorkerConfig{}, fmt.Errorf("worker request requires task or task_brief")
	}
	if request.PermissionProfile.Name == "" {
		return codex.WorkerConfig{}, fmt.Errorf("worker request requires permission_profile")
	}
	permissionProfile, err := codex.ResolvePermissionProfile(request.Caste, request.PermissionProfile)
	if err != nil {
		return codex.WorkerConfig{}, err
	}
	if request.TaskID == "" {
		request.TaskID = request.WorkerName
	}
	agentName := strings.TrimSpace(request.AgentName)
	if agentName == "" {
		agentName = codexAgentNameForCaste(request.Caste)
	}
	timeout := codex.DefaultWorkerTimeout
	if request.TimeoutMS < 0 {
		return codex.WorkerConfig{}, fmt.Errorf("worker timeout_ms cannot be negative")
	}
	if request.TimeoutMS > 0 {
		timeout = time.Duration(request.TimeoutMS) * time.Millisecond
	}
	if timeout > internalWorkerTimeoutMax {
		return codex.WorkerConfig{}, fmt.Errorf("worker timeout cannot exceed %v", internalWorkerTimeoutMax)
	}
	skillSection := joinInternalWorkerSections(request.SkillSection, request.HiveSection)
	config := codex.WorkerConfig{
		AgentName:         agentName,
		AgentTOMLPath:     dispatchAgentPath(root, invoker, agentName),
		Caste:             request.Caste,
		WorkerName:        request.WorkerName,
		TaskID:            request.TaskID,
		TaskBrief:         request.TaskBrief,
		ContextCapsule:    strings.TrimSpace(request.ContextCapsule),
		Root:              root,
		Timeout:           timeout,
		SkillSection:      skillSection,
		PheromoneSection:  strings.TrimSpace(request.PheromoneSection),
		HandoffSection:    strings.TrimSpace(request.HandoffSection),
		PermissionProfile: permissionProfile,
	}
	if config.ContextCapsule == "" {
		config.ContextCapsule = resolveCodexWorkerContext()
	}
	if _, ok := invoker.(*codex.FakeInvoker); !ok {
		if err := invoker.ValidateAgent(config.AgentTOMLPath); err != nil {
			return codex.WorkerConfig{}, fmt.Errorf("validate %s worker agent: %w", codex.PlatformFromInvoker(invoker), err)
		}
	}
	return config, nil
}

func joinInternalWorkerSections(sections ...string) string {
	joined := make([]string, 0, len(sections))
	for _, section := range sections {
		if section = strings.TrimSpace(section); section != "" {
			joined = append(joined, section)
		}
	}
	return strings.Join(joined, "\n\n")
}

func validateInternalWorkerResult(request internalWorkerDispatchRequest, result *codex.WorkerResult, invokeErr error) error {
	if result == nil {
		return fmt.Errorf("worker provider returned no result")
	}
	status := strings.ToLower(strings.TrimSpace(result.Status))
	switch status {
	case "completed", "failed", "blocked", "timeout":
		result.Status = status
	default:
		return fmt.Errorf("worker provider returned invalid terminal status %q", result.Status)
	}
	if strings.TrimSpace(result.WorkerName) == "" {
		result.WorkerName = strings.TrimSpace(request.WorkerName)
	}
	if strings.TrimSpace(result.Caste) == "" {
		result.Caste = strings.TrimSpace(request.Caste)
	}
	if strings.TrimSpace(result.TaskID) == "" {
		result.TaskID = strings.TrimSpace(request.TaskID)
		if result.TaskID == "" {
			result.TaskID = strings.TrimSpace(request.WorkerName)
		}
	}
	if result.WorkerName != strings.TrimSpace(request.WorkerName) || !strings.EqualFold(result.Caste, strings.TrimSpace(request.Caste)) {
		return fmt.Errorf("worker provider result identity does not match the issued request")
	}
	expectedTaskID := strings.TrimSpace(request.TaskID)
	if expectedTaskID == "" {
		expectedTaskID = strings.TrimSpace(request.WorkerName)
	}
	if result.TaskID != expectedTaskID {
		return fmt.Errorf("worker provider result task identity does not match the issued request")
	}
	if invokeErr != nil && result.Status == "completed" {
		return fmt.Errorf("worker provider returned completed with an invocation error")
	}
	return nil
}

func mapInternalWorkerResult(result codex.WorkerResult, invokeErr error) *internalWorkerResult {
	errorText := ""
	if result.Error != nil {
		errorText = sanitizeInternalWorkerAdapterError(result.Error.Error())
	} else if invokeErr != nil {
		errorText = sanitizeInternalWorkerAdapterError(invokeErr.Error())
	}
	return &internalWorkerResult{
		Name:          result.WorkerName,
		Caste:         result.Caste,
		TaskID:        result.TaskID,
		Status:        result.Status,
		Summary:       strings.TrimSpace(result.Summary),
		FilesCreated:  append([]string(nil), result.FilesCreated...),
		FilesModified: append([]string(nil), result.FilesModified...),
		TestsWritten:  append([]string(nil), result.TestsWritten...),
		Artifacts:     result.Artifacts,
		ScoutReport:   result.ScoutReport,
		ToolCount:     result.ToolCount,
		Blockers:      append([]string(nil), result.Blockers...),
		Spawns:        append([]string(nil), result.Spawns...),
		Duration:      result.Duration.Seconds(),
		Error:         errorText,
		Handoff:       result.Handoff,
	}
}

func sanitizeInternalWorkerAdapterError(message string) string {
	message = internalWorkerSecretPattern.ReplaceAllString(message, "[redacted]")
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > 1000 {
		message = message[:1000] + "..."
	}
	return message
}
