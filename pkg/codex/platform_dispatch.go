package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	envActivePlatform  = "AETHER_ACTIVE_PLATFORM"
	envWorkerPlatform  = "AETHER_WORKER_PLATFORM"
	envClaudePath      = "AETHER_CLAUDE_PATH"
	envOpenCodePath    = "AETHER_OPENCODE_PATH"
	envOpenCodePrimary = "AETHER_OPENCODE_PRIMARY_AGENT"
	defaultProbeTimout = 3 * time.Second
)

const defaultOpenCodePrimaryAgent = "build"

type Platform string

const (
	PlatformUnknown  Platform = "unknown"
	PlatformCodex    Platform = "codex"
	PlatformClaude   Platform = "claude"
	PlatformOpenCode Platform = "opencode"
	PlatformFake     Platform = "fake"
)

type AvailabilityStatus struct {
	Platform  Platform `json:"platform"`
	Binary    string   `json:"binary"`
	Available bool     `json:"available"`
	Reason    string   `json:"reason,omitempty"`
}

type PlatformDispatcher interface {
	WorkerInvoker
	Platform() Platform
	Availability(ctx context.Context) AvailabilityStatus
}

type selectionMetadata interface {
	ActivePlatform() Platform
	CandidateStatuses() []AvailabilityStatus
}

type markdownAgentDefinition struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type CodexDispatcher = RealInvoker

type ClaudeDispatcher struct {
	binaryName string
}

type OpenCodeDispatcher struct {
	binaryName string
}

type SelectedInvoker struct {
	active    Platform
	selected  PlatformDispatcher
	available []AvailabilityStatus
}

type UnavailableInvoker struct {
	active    Platform
	available []AvailabilityStatus
}

func NewCodexDispatcher() *CodexDispatcher {
	return NewRealInvoker()
}

func NewClaudeDispatcher() *ClaudeDispatcher {
	name := strings.TrimSpace(os.Getenv(envClaudePath))
	if name == "" {
		name = "claude"
	}
	return &ClaudeDispatcher{binaryName: name}
}

func NewOpenCodeDispatcher() *OpenCodeDispatcher {
	name := strings.TrimSpace(os.Getenv(envOpenCodePath))
	if name == "" {
		name = "opencode"
	}
	return &OpenCodeDispatcher{binaryName: name}
}

func (f *FakeInvoker) Platform() Platform        { return PlatformFake }
func (r *RealInvoker) Platform() Platform        { return PlatformCodex }
func (c *ClaudeDispatcher) Platform() Platform   { return PlatformClaude }
func (o *OpenCodeDispatcher) Platform() Platform { return PlatformOpenCode }

func (s *SelectedInvoker) Platform() Platform {
	if s == nil || s.selected == nil {
		return PlatformUnknown
	}
	return s.selected.Platform()
}

func (s *SelectedInvoker) Availability(ctx context.Context) AvailabilityStatus {
	if s == nil || s.selected == nil {
		return AvailabilityStatus{Platform: PlatformUnknown, Available: false, Reason: "no platform dispatcher selected"}
	}
	return s.selected.Availability(ctx)
}

func (s *SelectedInvoker) ActivePlatform() Platform {
	if s == nil {
		return PlatformUnknown
	}
	return s.active
}

func (s *SelectedInvoker) CandidateStatuses() []AvailabilityStatus {
	if s == nil {
		return nil
	}
	out := make([]AvailabilityStatus, len(s.available))
	copy(out, s.available)
	return out
}

func (s *SelectedInvoker) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	if s == nil || s.selected == nil {
		return WorkerResult{}, fmt.Errorf("worker dispatcher unavailable: no platform selected")
	}
	return s.selected.Invoke(ctx, config)
}

func (s *SelectedInvoker) InvokeWithProgress(ctx context.Context, config WorkerConfig, observer WorkerProgressObserver) (WorkerResult, error) {
	if s == nil || s.selected == nil {
		return WorkerResult{}, fmt.Errorf("worker dispatcher unavailable: no platform selected")
	}
	if invoker, ok := s.selected.(ProgressAwareWorkerInvoker); ok {
		return invoker.InvokeWithProgress(ctx, config, observer)
	}
	return s.selected.Invoke(ctx, config)
}

func (s *SelectedInvoker) IsAvailable(ctx context.Context) bool {
	return s.Availability(ctx).Available
}

func (s *SelectedInvoker) ValidateAgent(path string) error {
	if s == nil || s.selected == nil {
		return fmt.Errorf("worker dispatcher unavailable: no platform selected")
	}
	return s.selected.ValidateAgent(path)
}

func (u *UnavailableInvoker) Platform() Platform { return PlatformUnknown }

func (u *UnavailableInvoker) Availability(ctx context.Context) AvailabilityStatus {
	return AvailabilityStatus{
		Platform:  PlatformUnknown,
		Available: false,
		Reason:    describeAvailabilitySet(u.active, u.available),
	}
}

func (u *UnavailableInvoker) ActivePlatform() Platform {
	if u == nil {
		return PlatformUnknown
	}
	return u.active
}

func (u *UnavailableInvoker) CandidateStatuses() []AvailabilityStatus {
	if u == nil {
		return nil
	}
	out := make([]AvailabilityStatus, len(u.available))
	copy(out, u.available)
	return out
}

func (u *UnavailableInvoker) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	start := time.Now()
	err := fmt.Errorf("worker dispatcher unavailable: %s", describeAvailabilitySet(u.active, u.available))
	return WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "failed",
		Duration:   time.Since(start),
		Error:      err,
	}, err
}

func (u *UnavailableInvoker) IsAvailable(ctx context.Context) bool { return false }

func (u *UnavailableInvoker) ValidateAgent(path string) error {
	return fmt.Errorf("worker dispatcher unavailable: %s", describeAvailabilitySet(u.active, u.available))
}

func DetectActivePlatform() Platform {
	if platform := normalizePlatform(os.Getenv(envActivePlatform)); platform != PlatformUnknown {
		return platform
	}
	if platform := detectPlatformFromEnv(); platform != PlatformUnknown {
		return platform
	}
	return detectPlatformFromProcessTree(context.Background())
}

func PlatformFromInvoker(invoker WorkerInvoker) Platform {
	if dispatcher, ok := invoker.(interface{ Platform() Platform }); ok {
		return dispatcher.Platform()
	}
	return PlatformUnknown
}

func AgentDefinitionPath(root string, platform Platform, agentName string) string {
	root = strings.TrimSpace(root)
	base := strings.TrimSpace(agentName)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if base == "" {
		return ""
	}
	switch platform {
	case PlatformClaude:
		return filepath.Join(root, ".claude", "agents", "ant", base+".md")
	case PlatformOpenCode:
		return filepath.Join(root, ".opencode", "agents", base+".md")
	default:
		return filepath.Join(root, ".codex", "agents", base+".toml")
	}
}

func SelectPlatformInvoker(ctx context.Context) WorkerInvoker {
	active := DetectActivePlatform()
	preferred := active
	if override := normalizePlatform(os.Getenv(envWorkerPlatform)); override != PlatformUnknown && override != PlatformFake {
		preferred = override
	}

	dispatchers := reorderDispatchers([]PlatformDispatcher{
		NewCodexDispatcher(),
		NewClaudeDispatcher(),
		NewOpenCodeDispatcher(),
	}, preferred)

	statuses := make([]AvailabilityStatus, 0, len(dispatchers))
	for _, dispatcher := range dispatchers {
		status := dispatcher.Availability(ctx)
		statuses = append(statuses, status)
		if status.Available {
			return &SelectedInvoker{
				active:    active,
				selected:  dispatcher,
				available: statuses,
			}
		}
	}

	return &UnavailableInvoker{
		active:    active,
		available: statuses,
	}
}

func DescribeInvokerAvailability(invoker WorkerInvoker, ctx context.Context) string {
	active := DetectActivePlatform()
	if meta, ok := invoker.(selectionMetadata); ok {
		active = meta.ActivePlatform()
		if availability, ok := invoker.(interface {
			Availability(context.Context) AvailabilityStatus
		}); ok {
			status := availability.Availability(ctx)
			if status.Available {
				if active != PlatformUnknown {
					if active == status.Platform {
						return fmt.Sprintf("using %s worker dispatcher (detected host: %s)", status.Platform, active)
					}
					return fmt.Sprintf("detected host %s, falling back to %s worker dispatcher", active, status.Platform)
				}
				return fmt.Sprintf("using %s worker dispatcher", status.Platform)
			}
		}
		return describeAvailabilitySet(active, meta.CandidateStatuses())
	}
	if availability, ok := invoker.(interface {
		Availability(context.Context) AvailabilityStatus
	}); ok {
		status := availability.Availability(ctx)
		if status.Available {
			if active != PlatformUnknown {
				return fmt.Sprintf("using %s worker dispatcher (detected host: %s)", status.Platform, active)
			}
			return fmt.Sprintf("using %s worker dispatcher", status.Platform)
		}
		if strings.TrimSpace(status.Reason) != "" {
			return status.Reason
		}
	}
	if platform := PlatformFromInvoker(invoker); platform != PlatformUnknown {
		return fmt.Sprintf("using %s worker dispatcher", platform)
	}
	return "worker dispatcher availability unknown"
}

func (r *RealInvoker) Availability(ctx context.Context) AvailabilityStatus {
	status := AvailabilityStatus{Platform: PlatformCodex, Binary: strings.TrimSpace(r.binaryName)}
	if status.Binary == "" {
		status.Binary = "codex"
	}
	if _, err := exec.LookPath(status.Binary); err != nil {
		status.Reason = fmt.Sprintf("%s binary %q not found in PATH", status.Platform, status.Binary)
		return status
	}
	if !shouldProbeCLIAuth(status.Binary, "codex") {
		status.Available = true
		return status
	}
	output, err := runAvailabilityProbe(ctx, status.Binary, "login", "status")
	if err != nil {
		status.Reason = formatAvailabilityProbeError(status.Platform, "login status", err, output)
		return status
	}
	if strings.Contains(strings.ToLower(stripANSIEscapeCodes(output)), "logged in") {
		status.Available = true
		return status
	}
	status.Reason = fmt.Sprintf("%s login status did not confirm an authenticated session", status.Platform)
	return status
}

func (c *ClaudeDispatcher) Availability(ctx context.Context) AvailabilityStatus {
	status := AvailabilityStatus{Platform: PlatformClaude, Binary: strings.TrimSpace(c.binaryName)}
	if status.Binary == "" {
		status.Binary = "claude"
	}
	if _, err := exec.LookPath(status.Binary); err != nil {
		status.Reason = fmt.Sprintf("%s binary %q not found in PATH", status.Platform, status.Binary)
		return status
	}
	if !shouldProbeCLIAuth(status.Binary, "claude") {
		status.Available = true
		return status
	}
	output, err := runAvailabilityProbe(ctx, status.Binary, "auth", "status", "--json")
	if err != nil {
		status.Reason = formatAvailabilityProbeError(status.Platform, "auth status", err, output)
		return status
	}
	var payload struct {
		LoggedIn bool `json:"loggedIn"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		status.Reason = fmt.Sprintf("%s auth status returned invalid JSON: %v", status.Platform, err)
		return status
	}
	if !payload.LoggedIn {
		status.Reason = fmt.Sprintf("%s auth status reported no active login", status.Platform)
		return status
	}
	status.Available = true
	return status
}

func (o *OpenCodeDispatcher) Availability(ctx context.Context) AvailabilityStatus {
	status := AvailabilityStatus{Platform: PlatformOpenCode, Binary: strings.TrimSpace(o.binaryName)}
	if status.Binary == "" {
		status.Binary = "opencode"
	}
	if _, err := exec.LookPath(status.Binary); err != nil {
		status.Reason = fmt.Sprintf("%s binary %q not found in PATH", status.Platform, status.Binary)
		return status
	}
	if !shouldProbeCLIAuth(status.Binary, "opencode") {
		status.Available = true
		return status
	}
	output, err := runAvailabilityProbe(ctx, status.Binary, "auth", "list")
	if err != nil {
		status.Reason = formatAvailabilityProbeError(status.Platform, "auth list", err, output)
		return status
	}
	if countOpenCodeCredentials(output) == 0 {
		status.Reason = fmt.Sprintf("%s auth list reported no configured credentials or environment keys", status.Platform)
		return status
	}
	status.Available = true
	return status
}

func (c *ClaudeDispatcher) IsAvailable(ctx context.Context) bool {
	return c.Availability(ctx).Available
}
func (o *OpenCodeDispatcher) IsAvailable(ctx context.Context) bool {
	return o.Availability(ctx).Available
}
func (c *ClaudeDispatcher) ValidateAgent(path string) error   { return validateMarkdownAgent(path) }
func (o *OpenCodeDispatcher) ValidateAgent(path string) error { return validateMarkdownAgent(path) }

func (c *ClaudeDispatcher) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	return c.InvokeWithProgress(ctx, config, nil)
}

func (o *OpenCodeDispatcher) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	return o.InvokeWithProgress(ctx, config, nil)
}

func (c *ClaudeDispatcher) InvokeWithProgress(ctx context.Context, config WorkerConfig, observer WorkerProgressObserver) (WorkerResult, error) {
	start := time.Now()
	schemaJSON, err := marshalJSON(workerClaimsSchema())
	if err != nil {
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Duration:   time.Since(start),
			Error:      fmt.Errorf("marshal worker claims schema: %w", err),
		}, err
	}
	args := []string{"-p", "--output-format", "json", "--json-schema", string(schemaJSON), "--agent", strings.TrimSpace(config.AgentName), "--permission-mode", "bypassPermissions"}
	if root := strings.TrimSpace(config.Root); root != "" {
		args = append(args, "--add-dir", root)
	}
	prompt := strings.TrimSpace(AssembleHostedPrompt(config.ContextCapsule, config.SkillSection, config.PheromoneSection, config.TaskBrief) + "\n\n" + renderResponseContract(config))
	args = append(args, prompt)
	return invokeHostedWorker(ctx, c, config, observer, args, "claude")
}

func (o *OpenCodeDispatcher) InvokeWithProgress(ctx context.Context, config WorkerConfig, observer WorkerProgressObserver) (WorkerResult, error) {
	args := []string{"run", "--agent", openCodePrimaryAgent(), "--format", "json"}
	workerPrompt := strings.TrimSpace(AssembleHostedPrompt(config.ContextCapsule, config.SkillSection, config.PheromoneSection, config.TaskBrief) + "\n\n" + renderResponseContract(config))
	prompt := renderOpenCodeSubagentDispatchPrompt(config, workerPrompt)
	args = append(args, prompt)
	return invokeHostedWorker(ctx, o, config, observer, args, "opencode")
}

func openCodePrimaryAgent() string {
	agent := strings.TrimSpace(os.Getenv(envOpenCodePrimary))
	if agent == "" {
		return defaultOpenCodePrimaryAgent
	}
	return agent
}

func renderOpenCodeSubagentDispatchPrompt(config WorkerConfig, workerPrompt string) string {
	agentName := strings.TrimSpace(config.AgentName)
	description := fmt.Sprintf("%s %s: task %s", strings.TrimSpace(config.Caste), strings.TrimSpace(config.WorkerName), strings.TrimSpace(config.TaskID))
	description = strings.TrimSpace(description)
	if description == "" {
		description = "Aether worker dispatch"
	}
	return strings.TrimSpace(fmt.Sprintf(`Aether worker dispatch request.

Use the Task tool exactly once with:
- subagent_type: %q
- description: %q
- prompt: the complete worker prompt below

The final worker claims JSON must preserve:
- ant_name: %q
- caste: %q
- task_id: %q

Wait for the Task tool to finish. Then return ONLY the worker claims JSON produced by that subagent.
Do not summarize, wrap, or reformat the JSON. Do not run the worker task yourself unless the Task tool is unavailable; if unavailable, return a failed worker claims JSON object that names the Task tool as the blocker.

## Worker Prompt

%s`, agentName, description, strings.TrimSpace(config.WorkerName), strings.TrimSpace(config.Caste), strings.TrimSpace(config.TaskID), workerPrompt))
}

func invokeHostedWorker(ctx context.Context, dispatcher PlatformDispatcher, config WorkerConfig, observer WorkerProgressObserver, args []string, label string) (WorkerResult, error) {
	start := time.Now()
	status := dispatcher.Availability(ctx)
	if !status.Available {
		err := fmt.Errorf("worker startup failed: %s", status.Reason)
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "failed", Duration: time.Since(start), Error: err}, err
	}
	if strings.TrimSpace(config.AgentName) == "" {
		err := fmt.Errorf("worker startup failed: missing agent name")
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "failed", Duration: time.Since(start), Error: err}, err
	}
	if strings.TrimSpace(config.AgentTOMLPath) == "" {
		err := fmt.Errorf("worker startup failed: missing platform agent definition path")
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "failed", Duration: time.Since(start), Error: err}, err
	}
	if err := validateWorkerLaunchConfig(config); err != nil {
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "failed", Duration: time.Since(start), Error: err}, err
	}
	if err := dispatcher.ValidateAgent(config.AgentTOMLPath); err != nil {
		err = fmt.Errorf("worker startup failed: %w", err)
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "failed", Duration: time.Since(start), Error: err}, err
	}

	timeout := config.effectiveTimeout()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	binary := status.Binary
	if binary == "" {
		binary = string(dispatcher.Platform())
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	if strings.TrimSpace(config.Root) != "" {
		cmd.Dir = config.Root
	}

	var stdout, stderr bytes.Buffer
	running := newWorkerRunningSignal(observer)
	cmd.Stdout = &workerProgressWriter{buffer: &stdout, running: running, message: "worker output observed"}
	cmd.Stderr = &workerProgressWriter{buffer: &stderr, running: running, message: "worker stderr observed"}
	if err := cmd.Start(); err != nil {
		startupErr := fmt.Errorf("worker startup failed: %s start failed: %w", label, err)
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "failed", Duration: time.Since(start), Error: startupErr}, startupErr
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	heartbeat := time.NewTicker(workerHeartbeatInterval(timeout))
	defer heartbeat.Stop()

	var waitErr error
waitLoop:
	for {
		select {
		case waitErr = <-waitCh:
			break waitLoop
		case <-heartbeat.C:
			running.Report("worker heartbeat observed")
		}
	}

	duration := time.Since(start)
	rawOutput := combinedWorkerOutput(stdout.String(), stderr.String())
	if ctx.Err() == context.DeadlineExceeded {
		reportedTimeout := duration.Round(time.Millisecond)
		if duration >= time.Second {
			reportedTimeout = duration.Round(time.Second)
		}
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "timeout", Duration: duration, RawOutput: rawOutput, Error: fmt.Errorf("worker timeout after %v", reportedTimeout)}, nil
	}
	if waitErr != nil {
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "failed", Duration: duration, RawOutput: rawOutput, Error: classifyHostedExecutionError(label, waitErr, stderr.String(), running.Observed())}, nil
	}
	claims, parseErr := parseHostedWorkerOutput(label, rawOutput)
	if parseErr != nil {
		err := classifyWorkerFinalMessageError("parse worker output", parseErr, running.Observed())
		if debugPath := writeHostedWorkerOutputDebug(config.Root, label, config, args, stdout.String(), stderr.String(), parseErr); debugPath != "" {
			err = fmt.Errorf("%w (debug: %s)", err, debugPath)
		}
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "failed", Duration: duration, RawOutput: rawOutput, Error: err}, nil
	}
	claims = normalizeWorkerClaims(claims, config)
	return WorkerResult{
		WorkerName:    config.WorkerName,
		Caste:         config.Caste,
		TaskID:        config.TaskID,
		Status:        claims.Status,
		Summary:       claims.Summary,
		FilesCreated:  claims.FilesCreated,
		FilesModified: claims.FilesModified,
		TestsWritten:  claims.TestsWritten,
		ToolCount:     claims.ToolCount,
		Blockers:      claims.Blockers,
		Spawns:        claims.Spawns,
		Duration:      duration,
		RawOutput:     rawOutput,
	}, nil
}

func parseHostedWorkerOutput(label, output string) (workerClaims, error) {
	platform := strings.ToLower(strings.TrimSpace(label))
	switch platform {
	case "claude", "opencode":
		if claims, err := parseHostedJSONWorkerOutput(platform, output); err == nil {
			return claims, nil
		}
	}
	return ParseWorkerOutput(output)
}

func parseHostedJSONWorkerOutput(label, output string) (workerClaims, error) {
	candidates := hostedJSONTextCandidates(output)
	for i := len(candidates) - 1; i >= 0; i-- {
		claims, err := ParseWorkerOutput(candidates[i])
		if err == nil {
			return claims, nil
		}
	}
	if len(candidates) > 0 {
		if claims, err := ParseWorkerOutput(strings.Join(candidates, "\n")); err == nil {
			return claims, nil
		}
	}
	return workerClaims{}, fmt.Errorf("parse %s json output: no worker claims found", strings.TrimSpace(label))
}

func hostedJSONTextCandidates(output string) []string {
	var candidates []string
	trimmed := strings.TrimSpace(stripANSIEscapeCodes(output))
	if trimmed == "" {
		return nil
	}
	var fullEvent interface{}
	if err := json.Unmarshal([]byte(trimmed), &fullEvent); err == nil {
		collectHostedTextCandidates(fullEvent, &candidates)
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(stripANSIEscapeCodes(line))
		if line == "" {
			continue
		}
		var event interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		collectHostedTextCandidates(event, &candidates)
	}
	return compactStrings(candidates)
}

func collectHostedTextCandidates(value interface{}, candidates *[]string) {
	switch typed := value.(type) {
	case string:
		appendHostedTextCandidate(candidates, typed)
	case []interface{}:
		for _, item := range typed {
			collectHostedTextCandidates(item, candidates)
		}
	case map[string]interface{}:
		if isWorkerClaimsMap(typed) {
			if data, err := json.Marshal(typed); err == nil {
				appendHostedTextCandidate(candidates, string(data))
			}
		}
		for _, key := range []string{
			"text",
			"content",
			"message",
			"output",
			"result",
			"final",
			"delta",
			"part",
			"parts",
			"properties",
			"data",
		} {
			if nested, ok := typed[key]; ok {
				collectHostedTextCandidates(nested, candidates)
			}
		}
	}
}

func appendHostedTextCandidate(candidates *[]string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if !strings.Contains(value, "{") && !strings.Contains(value, "ant_name") {
		return
	}
	*candidates = append(*candidates, value)
}

func isWorkerClaimsMap(value map[string]interface{}) bool {
	matches := 0
	for _, key := range []string{
		"ant_name",
		"caste",
		"task_id",
		"status",
		"summary",
		"files_created",
		"files_modified",
		"tests_written",
		"blockers",
		"spawns",
	} {
		if _, ok := value[key]; ok {
			matches++
		}
	}
	return matches >= 2
}

func writeHostedWorkerOutputDebug(root, label string, config WorkerConfig, args []string, stdoutText, stderrText string, cause error) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	debugDir := filepath.Join(root, ".aether", "data", "worker-debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return ""
	}
	now := time.Now().UTC()
	filename := fmt.Sprintf("%s-%s-%d.json", safeDebugToken(label), safeDebugToken(config.WorkerName), now.UnixNano())
	relPath := filepath.ToSlash(filepath.Join(".aether", "data", "worker-debug", filename))
	payload := map[string]interface{}{
		"created_at":     now.Format(time.RFC3339Nano),
		"platform":       strings.TrimSpace(label),
		"worker_name":    strings.TrimSpace(config.WorkerName),
		"caste":          strings.TrimSpace(config.Caste),
		"task_id":        strings.TrimSpace(config.TaskID),
		"agent_name":     strings.TrimSpace(config.AgentName),
		"args":           safeHostedWorkerArgs(args),
		"stdout_bytes":   len(stdoutText),
		"stderr_bytes":   len(stderrText),
		"stdout_excerpt": workerOutputExcerpt(stdoutText),
		"stderr_excerpt": workerOutputExcerpt(stderrText),
		"error":          strings.TrimSpace(cause.Error()),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ""
	}
	if err := os.WriteFile(filepath.Join(debugDir, filename), data, 0644); err != nil {
		return ""
	}
	return relPath
}

func safeHostedWorkerArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, len(args))
	copy(out, args)
	last := strings.TrimSpace(out[len(out)-1])
	if last != "" {
		out[len(out)-1] = fmt.Sprintf("<prompt: %d bytes>", len(last))
	}
	return out
}

func workerOutputExcerpt(value string) string {
	value = strings.TrimSpace(stripANSIEscapeCodes(value))
	const limit = 4000
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "\n[truncated]"
}

func safeDebugToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "worker"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	token := strings.Trim(b.String(), "-_")
	if token == "" {
		return "worker"
	}
	return token
}

func normalizePlatform(raw string) Platform {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "codex", "codex-cli":
		return PlatformCodex
	case "claude", "claude-code":
		return PlatformClaude
	case "opencode", "open-code":
		return PlatformOpenCode
	case "fake", "test":
		return PlatformFake
	default:
		return PlatformUnknown
	}
}

func shouldProbeCLIAuth(binaryPath, expected string) bool {
	base := strings.ToLower(strings.TrimSpace(filepath.Base(binaryPath)))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return base == strings.ToLower(strings.TrimSpace(expected))
}

func detectPlatformFromEnv() Platform {
	switch {
	case hasAnyEnv("CODEX_THREAD_ID", "CODEX_SESSION_ID", "CODEX_CI"):
		return PlatformCodex
	case hasEnvPrefix("CLAUDE_CODE_") || hasAnyEnv("CLAUDECODE", "CLAUDECODE_PROJECT_DIR", "CLAUDE_PROJECT_DIR", "CLAUDE_CODE_SIMPLE"):
		return PlatformClaude
	case hasEnvPrefix("OPENCODE_"):
		return PlatformOpenCode
	default:
		return PlatformUnknown
	}
}

func detectPlatformFromProcessTree(ctx context.Context) Platform {
	pid := os.Getppid()
	for depth := 0; depth < 16 && pid > 1; depth++ {
		command, nextPID, err := lookupParentProcess(ctx, pid)
		if err != nil {
			return PlatformUnknown
		}
		switch identifyPlatformFromCommand(command) {
		case PlatformCodex, PlatformClaude, PlatformOpenCode:
			return identifyPlatformFromCommand(command)
		}
		pid = nextPID
	}
	return PlatformUnknown
}

func lookupParentProcess(ctx context.Context, pid int) (string, int, error) {
	if pid <= 1 {
		return "", 0, fmt.Errorf("no parent process")
	}
	commandOut, err := runAvailabilityProbe(ctx, "ps", "-o", "args=", "-p", strconv.Itoa(pid))
	if err != nil {
		return "", 0, err
	}
	ppidOut, err := runAvailabilityProbe(ctx, "ps", "-o", "ppid=", "-p", strconv.Itoa(pid))
	if err != nil {
		return "", 0, err
	}
	ppid, convErr := strconv.Atoi(strings.TrimSpace(ppidOut))
	if convErr != nil {
		return strings.TrimSpace(commandOut), 0, convErr
	}
	return strings.TrimSpace(commandOut), ppid, nil
}

func identifyPlatformFromCommand(command string) Platform {
	value := strings.ToLower(strings.TrimSpace(command))
	switch {
	case strings.Contains(value, "codex"):
		return PlatformCodex
	case strings.Contains(value, "claude"):
		return PlatformClaude
	case strings.Contains(value, "opencode"):
		return PlatformOpenCode
	default:
		return PlatformUnknown
	}
}

func hasAnyEnv(keys ...string) bool {
	for _, key := range keys {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}

func hasEnvPrefix(prefix string) bool {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return false
	}
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, prefix) {
			return true
		}
	}
	return false
}

func reorderDispatchers(dispatchers []PlatformDispatcher, preferred Platform) []PlatformDispatcher {
	if preferred == PlatformUnknown {
		return dispatchers
	}
	out := make([]PlatformDispatcher, 0, len(dispatchers))
	for _, dispatcher := range dispatchers {
		if dispatcher.Platform() == preferred {
			out = append(out, dispatcher)
		}
	}
	for _, dispatcher := range dispatchers {
		if dispatcher.Platform() != preferred {
			out = append(out, dispatcher)
		}
	}
	return out
}

func runAvailabilityProbe(ctx context.Context, binary string, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultProbeTimout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := combinedWorkerOutput(stdout.String(), stderr.String())
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return output, fmt.Errorf("timed out")
		}
		return output, err
	}
	return output, nil
}

func formatAvailabilityProbeError(platform Platform, action string, err error, output string) string {
	output = strings.TrimSpace(stripANSIEscapeCodes(output))
	if output != "" {
		return fmt.Sprintf("%s %s failed: %v (%s)", platform, action, err, output)
	}
	return fmt.Sprintf("%s %s failed: %v", platform, action, err)
}

func stripANSIEscapeCodes(value string) string {
	var b strings.Builder
	inEscape := false
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if inEscape {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				inEscape = false
			}
			continue
		}
		if ch == 0x1b {
			inEscape = true
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

func countOpenCodeCredentials(output string) int {
	cleaned := stripANSIEscapeCodes(output)
	count := 0
	for _, line := range strings.Split(cleaned, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "●") {
			count++
		}
	}
	return count
}

func validateMarkdownAgent(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("validate agent: read %s: %w", path, err)
	}
	def, err := parseMarkdownAgentDefinition(data)
	if err != nil {
		return fmt.Errorf("validate agent: %s: %w", path, err)
	}
	if strings.TrimSpace(def.Name) == "" {
		return fmt.Errorf("validate agent: %s: missing name", path)
	}
	if strings.TrimSpace(def.Description) == "" {
		return fmt.Errorf("validate agent: %s: missing description", path)
	}
	return nil
}

func parseMarkdownAgentDefinition(data []byte) (markdownAgentDefinition, error) {
	text := strings.TrimSpace(string(data))
	if !strings.HasPrefix(text, "---") {
		return markdownAgentDefinition{}, fmt.Errorf("missing YAML frontmatter")
	}
	lines := strings.Split(text, "\n")
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return markdownAgentDefinition{}, fmt.Errorf("unterminated YAML frontmatter")
	}
	frontmatter := strings.Join(lines[1:end], "\n")
	var def markdownAgentDefinition
	if err := yaml.Unmarshal([]byte(frontmatter), &def); err != nil {
		return markdownAgentDefinition{}, err
	}
	return def, nil
}

func classifyHostedExecutionError(label string, err error, stderr string, runningObserved bool) error {
	detail := strings.TrimSpace(stderr)
	prefix := strings.TrimSpace(label)
	if prefix == "" {
		prefix = "worker process"
	}
	if !runningObserved {
		prefix = "worker startup failed"
	} else {
		prefix = prefix + " failed"
	}
	if detail != "" {
		return fmt.Errorf("%s: %w (stderr: %s)", prefix, err, detail)
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

func describeAvailabilitySet(active Platform, statuses []AvailabilityStatus) string {
	parts := make([]string, 0, len(statuses)+1)
	if active != PlatformUnknown {
		parts = append(parts, fmt.Sprintf("detected host platform %s", active))
	}
	for _, status := range statuses {
		description := "available"
		if !status.Available {
			description = strings.TrimSpace(status.Reason)
			if description == "" {
				description = "unavailable"
			}
		}
		parts = append(parts, fmt.Sprintf("%s: %s", status.Platform, description))
	}
	if len(parts) == 0 {
		return "no worker dispatchers available"
	}
	return strings.Join(parts, "; ")
}
