package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/BurntSushi/toml"
)

// DefaultWorkerTimeout is the default timeout for worker invocations.
const DefaultWorkerTimeout = 10 * time.Minute

const defaultWorkerTimeout = DefaultWorkerTimeout

const (
	defaultWorkerHeartbeatInterval = 2 * time.Second
	minWorkerHeartbeatInterval     = 250 * time.Millisecond
)

// envRealDispatch is the environment variable that enables real codex CLI invocation.
const envRealDispatch = "AETHER_CODEX_REAL_DISPATCH"

// WorkerConfig specifies all parameters needed to invoke a single worker.
// Field names match the documented codexWorkerConfig in doc.go.
type WorkerConfig struct {
	AgentName        string        // TOML agent name (e.g., "aether-builder")
	AgentTOMLPath    string        // Absolute path to the agent's TOML file
	Caste            string        // Worker caste (builder, watcher, scout, etc.)
	WorkerName       string        // Deterministic ant name (e.g., "Hammer-23")
	TaskID           string        // Task identifier from the build dispatch
	TaskBrief        string        // The markdown task brief content
	ContextCapsule   string        // The assembled compact colony-prime context
	Root             string        // Repository root directory (working dir for subprocess)
	Timeout          time.Duration // Per-worker timeout (default: 10 minutes)
	SkillSection     string        // Skill guidance content injected into worker prompts
	PheromoneSection string        // Pheromone signal content injected into worker prompts
	ConfigOverrides  []string      // Optional codex config overrides passed as -c key=value
	ResponsePath     string        // Optional controller-managed response file path
}

// effectiveTimeout returns the configured timeout or the default.
func (c WorkerConfig) effectiveTimeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return defaultWorkerTimeout
}

// WorkerResult captures the outcome of a worker invocation.
// Field names match the documented codexWorkerResult in doc.go.
type WorkerResult struct {
	WorkerName    string        // The worker's assigned name
	Caste         string        // Worker caste
	TaskID        string        // Task identifier
	Status        string        // "completed", "failed", "blocked", or "timeout"
	Summary       string        // Worker's self-reported summary
	FilesCreated  []string      // Files the worker claims to have created
	FilesModified []string      // Files the worker claims to have modified
	TestsWritten  []string      // Test files the worker created
	ToolCount     int           // Number of tool calls reported
	Blockers      []string      // Blocking issues reported
	Spawns        []string      // Sub-workers spawned
	Duration      time.Duration // Wall-clock time of the invocation
	RawOutput     string        // Full stdout from the subprocess
	Error         error         // Invocation error (if any)
}

type jsonSchema struct {
	Type                 string                 `json:"type"`
	AdditionalProperties bool                   `json:"additionalProperties"`
	Properties           map[string]interface{} `json:"properties"`
	Required             []string               `json:"required,omitempty"`
}

// workerClaims represents the trailing JSON block returned by a Codex worker.
type workerClaims struct {
	AntName       string   `json:"ant_name"`
	Caste         string   `json:"caste"`
	TaskID        string   `json:"task_id"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary"`
	FilesCreated  []string `json:"files_created"`
	FilesModified []string `json:"files_modified"`
	TestsWritten  []string `json:"tests_written"`
	ToolCount     int      `json:"tool_count"`
	Blockers      []string `json:"blockers"`
	Spawns        []string `json:"spawns"`
}

// agentTOML represents the required fields from a Codex agent TOML file.
type agentTOML struct {
	Name                  string   `toml:"name"`
	Description           string   `toml:"description"`
	NicknameCandidates    []string `toml:"nickname_candidates"`
	DeveloperInstructions string   `toml:"developer_instructions"`
}

// WorkerProgressEvent reports non-terminal worker execution progress.
type WorkerProgressEvent struct {
	Status     string
	Message    string
	OccurredAt time.Time
}

// WorkerProgressObserver receives worker execution progress events.
type WorkerProgressObserver func(WorkerProgressEvent)

// ProgressAwareWorkerInvoker extends WorkerInvoker with runtime progress events.
type ProgressAwareWorkerInvoker interface {
	InvokeWithProgress(ctx context.Context, config WorkerConfig, observer WorkerProgressObserver) (WorkerResult, error)
}

// WorkerInvoker defines the contract for invoking a Codex worker.
type WorkerInvoker interface {
	// Invoke spawns a codex CLI subprocess for the given worker configuration.
	Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error)

	// IsAvailable checks whether the platform dispatcher is installed and authenticated.
	IsAvailable(ctx context.Context) bool

	// ValidateAgent checks that a TOML agent file is parseable and contains
	// all required fields (name, description, developer_instructions).
	ValidateAgent(path string) error
}

// --- FakeInvoker ---

// FakeInvoker returns deterministic results for testing.
type FakeInvoker struct{}

// Invoke returns a deterministic WorkerResult for the given config.
func (f *FakeInvoker) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	return f.InvokeWithProgress(ctx, config, nil)
}

// InvokeWithProgress returns a deterministic WorkerResult for the given config while
// emitting a synthetic running transition for runtime tests and simulated flows.
func (f *FakeInvoker) InvokeWithProgress(ctx context.Context, config WorkerConfig, observer WorkerProgressObserver) (WorkerResult, error) {
	start := time.Now()

	// Simulate brief processing delay
	select {
	case <-time.After(10 * time.Millisecond):
	case <-ctx.Done():
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Duration:   time.Since(start),
			Error:      ctx.Err(),
		}, nil
	}

	emitWorkerProgress(observer, WorkerProgressEvent{
		Status:     "running",
		Message:    "simulated worker heartbeat observed",
		OccurredAt: time.Now().UTC(),
	})

	claims := workerClaims{
		AntName:       config.WorkerName,
		Caste:         config.Caste,
		TaskID:        config.TaskID,
		Status:        "completed",
		Summary:       fmt.Sprintf("FakeInvoker completed task %s for worker %s (caste: %s)", config.TaskID, config.WorkerName, config.Caste),
		FilesCreated:  []string{},
		FilesModified: []string{},
		TestsWritten:  []string{},
		ToolCount:     0,
		Blockers:      nil,
		Spawns:        nil,
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return WorkerResult{}, fmt.Errorf("fake invoker: marshal claims: %w", err)
	}

	rawOutput := fmt.Sprintf("Fake invocation for %s\n%s", config.WorkerName, string(claimsJSON))

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
		Duration:      time.Since(start),
		RawOutput:     rawOutput,
	}, nil
}

// IsAvailable always returns true for FakeInvoker.
func (f *FakeInvoker) IsAvailable(ctx context.Context) bool {
	return true
}

// ValidateAgent always returns nil for FakeInvoker.
func (f *FakeInvoker) ValidateAgent(path string) error {
	return nil
}

// --- RealInvoker ---

// RealInvoker invokes the actual codex CLI binary as a subprocess.
type RealInvoker struct {
	binaryName string // Path to codex binary; defaults to "codex"
}

// NewRealInvoker creates a RealInvoker with the given binary name.
func NewRealInvoker() *RealInvoker {
	name := os.Getenv("AETHER_CODEX_PATH")
	if name == "" {
		name = "codex"
	}
	return &RealInvoker{binaryName: name}
}

func codexWritableDirs() []string {
	var dirs []string
	if dir := strings.TrimSpace(os.Getenv("CODEX_HOME")); dir != "" {
		dirs = append(dirs, filepath.Clean(dir))
	} else if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		dirs = append(dirs, filepath.Join(home, ".codex"))
	}

	seen := make(map[string]struct{}, len(dirs))
	out := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		out = append(out, dir)
	}
	return out
}

// IsAvailable checks whether the codex dispatcher is runnable and authenticated.
func (r *RealInvoker) IsAvailable(ctx context.Context) bool {
	return r.Availability(ctx).Available
}

// ValidateAgent parses and validates a TOML agent file.
func (r *RealInvoker) ValidateAgent(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("validate agent: read %s: %w", path, err)
	}

	var agent agentTOML
	if _, err := toml.Decode(string(data), &agent); err != nil {
		return fmt.Errorf("validate agent: parse %s: %w", path, err)
	}

	var missing []string
	if agent.Name == "" {
		missing = append(missing, "name")
	}
	if agent.Description == "" {
		missing = append(missing, "description")
	}
	if agent.DeveloperInstructions == "" {
		missing = append(missing, "developer_instructions")
	}
	if len(missing) > 0 {
		return fmt.Errorf("validate agent: %s: missing required fields: %s", path, strings.Join(missing, ", "))
	}

	return nil
}

// Invoke runs the codex CLI as a subprocess with timeout.
func (r *RealInvoker) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	return r.InvokeWithProgress(ctx, config, nil)
}

// InvokeWithProgress runs the codex CLI as a subprocess with timeout while
// emitting proof-backed runtime progress.
func (r *RealInvoker) InvokeWithProgress(ctx context.Context, config WorkerConfig, observer WorkerProgressObserver) (WorkerResult, error) {
	start := time.Now()

	if status := r.Availability(ctx); !status.Available {
		err := fmt.Errorf("worker startup failed: %s", strings.TrimSpace(status.Reason))
		if strings.TrimSpace(status.Reason) == "" {
			err = fmt.Errorf("worker startup failed: %s worker dispatcher is unavailable", PlatformCodex)
		}
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Duration:   time.Since(start),
			Error:      err,
		}, err
	}

	if strings.TrimSpace(config.AgentTOMLPath) == "" {
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Duration:   time.Since(start),
			Error:      fmt.Errorf("worker startup failed: missing agent TOML path"),
		}, fmt.Errorf("worker startup failed: missing agent TOML path for %s", config.WorkerName)
	}
	if err := validateWorkerLaunchConfig(config); err != nil {
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Duration:   time.Since(start),
			Error:      err,
		}, err
	}
	if err := r.ValidateAgent(config.AgentTOMLPath); err != nil {
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Duration:   time.Since(start),
			Error:      err,
		}, fmt.Errorf("worker startup failed: %w", err)
	}

	prompt, err := AssemblePrompt(config.AgentTOMLPath, config.ContextCapsule, config.SkillSection, config.PheromoneSection, config.TaskBrief)
	if err != nil {
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Duration:   time.Since(start),
			Error:      err,
		}, fmt.Errorf("worker startup failed: assemble worker prompt: %w", err)
	}
	prompt = strings.TrimSpace(prompt + "\n\n" + renderResponseContract(config))

	// Create a timeout context
	timeout := config.effectiveTimeout()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	lastMessagePath, err := writeTempFile("", "aether-codex-last-*.json", nil)
	if err != nil {
		return WorkerResult{}, fmt.Errorf("create last-message file: %w", err)
	}
	defer os.Remove(lastMessagePath)

	schemaJSON, err := marshalJSON(workerClaimsSchema())
	if err != nil {
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Duration:   time.Since(start),
			Error:      err,
		}, fmt.Errorf("marshal worker claims schema: %w", err)
	}

	schemaPath, err := writeTempFile("", "aether-codex-schema-*.json", schemaJSON)
	if err != nil {
		return WorkerResult{}, fmt.Errorf("create output schema file: %w", err)
	}
	defer os.Remove(schemaPath)

	// Build the command: codex exec --full-auto --ephemeral --output-last-message FILE --output-schema FILE
	args := []string{
		"exec",
		"--full-auto",
		"--json",
		"--ephemeral",
		"--skip-git-repo-check",
		"--output-last-message", lastMessagePath,
		"--output-schema", schemaPath,
	}
	for _, dir := range codexWritableDirs() {
		args = append(args, "--add-dir", dir)
	}
	for _, override := range compactStrings(config.ConfigOverrides) {
		args = append(args, "-c", override)
	}
	cmd := exec.CommandContext(ctx, r.binaryName, args...)
	if strings.TrimSpace(config.Root) != "" {
		cmd.Dir = config.Root
	}
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	running := newWorkerRunningSignal(observer)
	cmd.Stdout = &workerProgressWriter{
		buffer:  &stdout,
		running: running,
		message: "worker output observed",
	}
	cmd.Stderr = &workerProgressWriter{
		buffer:  &stderr,
		running: running,
		message: "worker stderr observed",
	}

	if err := cmd.Start(); err != nil {
		startupErr := fmt.Errorf("worker startup failed: codex exec start failed: %w", err)
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Duration:   time.Since(start),
			Error:      startupErr,
		}, startupErr
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

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
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "timeout",
			Duration:   duration,
			RawOutput:  rawOutput,
			Error:      fmt.Errorf("worker timeout after %v", reportedTimeout),
		}, nil
	}

	if waitErr != nil {
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Duration:   duration,
			RawOutput:  rawOutput,
			Error:      classifyWorkerExecutionError(waitErr, stderr.String(), running.Observed()),
		}, nil
	}

	lastMessage, readErr := os.ReadFile(lastMessagePath)
	if readErr != nil {
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Duration:   duration,
			RawOutput:  rawOutput,
			Error:      classifyWorkerFinalMessageError("read final worker message", readErr, running.Observed()),
		}, nil
	}

	claims, parseErr := ParseWorkerOutput(string(lastMessage))
	if parseErr != nil {
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Duration:   duration,
			RawOutput:  strings.TrimSpace(rawOutput + "\n" + string(lastMessage)),
			Error:      classifyWorkerFinalMessageError("parse worker output", parseErr, running.Observed()),
		}, nil
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

// --- ParseWorkerOutput ---

// ParseWorkerOutput extracts the last JSON object from worker stdout
// and parses it as workerClaims.
func ParseWorkerOutput(output string) (workerClaims, error) {
	var claims workerClaims
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return claims, fmt.Errorf("parse worker output: empty output")
	}

	if direct, ok := unmarshalWorkerClaims(trimmed); ok {
		return normalizeWorkerClaims(direct, WorkerConfig{}), nil
	}
	if fenced, ok := unmarshalWorkerClaims(stripCodeFence(trimmed)); ok {
		return normalizeWorkerClaims(fenced, WorkerConfig{}), nil
	}

	// Find all JSON-like substrings (objects starting with { and ending with })
	// Walk backward from the end to find the last complete JSON object
	lastBrace := strings.LastIndex(trimmed, "}")
	if lastBrace == -1 {
		return claims, fmt.Errorf("parse worker output: no JSON found in output")
	}

	// Find the matching opening brace by scanning backward
	depth := 0
	startIdx := -1
	for i := lastBrace; i >= 0; i-- {
		switch trimmed[i] {
		case '}':
			depth++
		case '{':
			depth--
			if depth == 0 {
				startIdx = i
				break
			}
		}
		if startIdx != -1 {
			break
		}
	}

	if startIdx == -1 {
		return claims, fmt.Errorf("parse worker output: no JSON found in output")
	}

	jsonStr := trimmed[startIdx : lastBrace+1]
	if err := json.Unmarshal([]byte(jsonStr), &claims); err != nil {
		return claims, fmt.Errorf("parse worker output: invalid JSON: %w", err)
	}
	if !isWorkerClaimsLike(claims) {
		return claims, fmt.Errorf("parse worker output: no JSON found in output")
	}

	return normalizeWorkerClaims(claims, WorkerConfig{}), nil
}

// --- Factory ---

// NewWorkerInvokerOrError returns a fake or platform-selected worker invoker
// based on the AETHER_CODEX_REAL_DISPATCH environment variable. It returns an
// error when no authenticated platform dispatcher is available and synthetic
// mode is not explicitly requested.
func NewWorkerInvokerOrError() (WorkerInvoker, error) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envRealDispatch))) {
	case "0", "false", "fake":
		return &FakeInvoker{}, nil
	case "1", "true", "real":
		invoker := SelectPlatformInvoker(context.Background())
		if invoker.IsAvailable(context.Background()) {
			return invoker, nil
		}
		return nil, fmt.Errorf("worker dispatcher is unavailable: %s", DescribeInvokerAvailability(invoker, context.Background()))
	}
	if normalizePlatform(os.Getenv(envWorkerPlatform)) == PlatformFake {
		return &FakeInvoker{}, nil
	}
	if runningInGoTest() {
		return &FakeInvoker{}, nil
	}
	invoker := SelectPlatformInvoker(context.Background())
	if invoker.IsAvailable(context.Background()) {
		return invoker, nil
	}
	return nil, fmt.Errorf("worker dispatcher is unavailable: %s; set %s=fake to run in synthetic mode", DescribeInvokerAvailability(invoker, context.Background()), envRealDispatch)
}

// NewWorkerInvoker returns a fake or platform-selected worker invoker based on
// the AETHER_CODEX_REAL_DISPATCH environment variable. When no authenticated
// dispatcher is available it returns an unavailable invoker instead of panicking
// so callers can surface a concrete fallback reason.
func NewWorkerInvoker() WorkerInvoker {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envRealDispatch))) {
	case "0", "false", "fake":
		return &FakeInvoker{}
	case "1", "true", "real":
		return SelectPlatformInvoker(context.Background())
	}
	if normalizePlatform(os.Getenv(envWorkerPlatform)) == PlatformFake {
		return &FakeInvoker{}
	}
	if runningInGoTest() {
		return &FakeInvoker{}
	}
	return SelectPlatformInvoker(context.Background())
}

func stripCodeFence(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "```") {
		return text
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 3 {
		return text
	}
	lines = lines[1:]
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func unmarshalWorkerClaims(text string) (workerClaims, bool) {
	var claims workerClaims
	if err := json.Unmarshal([]byte(text), &claims); err != nil {
		return workerClaims{}, false
	}
	if !isWorkerClaimsLike(claims) {
		return workerClaims{}, false
	}
	return claims, true
}

func isWorkerClaimsLike(claims workerClaims) bool {
	score := 0
	for _, value := range []string{
		claims.AntName,
		claims.Caste,
		claims.TaskID,
		claims.Status,
		claims.Summary,
	} {
		if strings.TrimSpace(value) != "" {
			score++
		}
	}
	if len(claims.FilesCreated) > 0 {
		score++
	}
	if len(claims.FilesModified) > 0 {
		score++
	}
	if len(claims.TestsWritten) > 0 {
		score++
	}
	if len(claims.Blockers) > 0 {
		score++
	}
	if len(claims.Spawns) > 0 {
		score++
	}
	return score >= 2
}

func renderResponseContract(config WorkerConfig) string {
	root := strings.TrimSpace(config.Root)
	if root == "" {
		root = "."
	}
	statusLine := "completed, failed, blocked"
	if strings.EqualFold(strings.TrimSpace(config.Caste), "builder") {
		statusLine = "code_written, completed, failed, blocked"
	}
	return strings.TrimSpace(fmt.Sprintf(`
## Final Response Contract

Return ONLY a single JSON object as your final response.
- Do not wrap the JSON in markdown code fences.
- Use repo-relative paths rooted at %q in files_created, files_modified, and tests_written.
- Set status to one of: %s.
- Report blockers truthfully. If blocked, explain why in blockers.
- Keep summary concise and concrete.
`, filepath.Clean(root), statusLine))
}

func workerClaimsSchema() jsonSchema {
	stringArray := map[string]interface{}{
		"type": "array",
		"items": map[string]interface{}{
			"type": "string",
		},
	}
	return jsonSchema{
		Type:                 "object",
		AdditionalProperties: false,
		Required: []string{
			"ant_name",
			"caste",
			"task_id",
			"status",
			"summary",
			"files_created",
			"files_modified",
			"tests_written",
			"tool_count",
			"blockers",
			"spawns",
		},
		Properties: map[string]interface{}{
			"ant_name": map[string]interface{}{"type": "string"},
			"caste":    map[string]interface{}{"type": "string"},
			"task_id":  map[string]interface{}{"type": "string"},
			"status": map[string]interface{}{
				"type": "string",
				"enum": []string{"completed", "code_written", "failed", "blocked"},
			},
			"summary":        map[string]interface{}{"type": "string"},
			"files_created":  stringArray,
			"files_modified": stringArray,
			"tests_written":  stringArray,
			"tool_count": map[string]interface{}{
				"type":    "integer",
				"minimum": 0,
			},
			"blockers": stringArray,
			"spawns":   stringArray,
		},
	}
}

func marshalJSON(v interface{}) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return data, nil
}

func writeTempFile(dir, pattern string, data []byte) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	if len(data) > 0 {
		if _, err := f.Write(data); err != nil {
			_ = f.Close()
			_ = os.Remove(name)
			return "", err
		}
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}

func normalizeWorkerClaims(claims workerClaims, config WorkerConfig) workerClaims {
	if strings.TrimSpace(claims.AntName) == "" {
		claims.AntName = config.WorkerName
	}
	if strings.TrimSpace(claims.Caste) == "" {
		claims.Caste = config.Caste
	}
	if strings.TrimSpace(claims.TaskID) == "" {
		claims.TaskID = config.TaskID
	}
	status := strings.ToLower(strings.TrimSpace(claims.Status))
	switch status {
	case "completed", "failed", "blocked":
		claims.Status = status
	case "code_written":
		claims.Status = "completed"
	default:
		if len(claims.Blockers) > 0 {
			claims.Status = "blocked"
		} else {
			claims.Status = "failed"
		}
	}
	claims.FilesCreated = normalizeClaimPaths(config.Root, claims.FilesCreated)
	claims.FilesModified = normalizeClaimPaths(config.Root, claims.FilesModified)
	claims.TestsWritten = normalizeClaimPaths(config.Root, claims.TestsWritten)
	claims.Blockers = compactStrings(claims.Blockers)
	claims.Spawns = compactStrings(claims.Spawns)
	return claims
}

func normalizeClaimPaths(root string, paths []string) []string {
	root = filepath.Clean(strings.TrimSpace(root))
	normalized := make([]string, 0, len(paths))
	seen := map[string]bool{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		cleaned := filepath.Clean(path)
		if root != "" && filepath.IsAbs(cleaned) {
			if rel, err := filepath.Rel(root, cleaned); err == nil && !strings.HasPrefix(rel, "..") {
				cleaned = rel
			}
		}
		cleaned = filepath.ToSlash(cleaned)
		if cleaned == "." || cleaned == "" || seen[cleaned] {
			continue
		}
		seen[cleaned] = true
		normalized = append(normalized, cleaned)
	}
	return normalized
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

type workerRunningSignal struct {
	observer WorkerProgressObserver
	seen     atomic.Bool
}

func newWorkerRunningSignal(observer WorkerProgressObserver) *workerRunningSignal {
	return &workerRunningSignal{observer: observer}
}

func (s *workerRunningSignal) Report(message string) {
	if s == nil {
		return
	}
	if !s.seen.CompareAndSwap(false, true) {
		return
	}
	emitWorkerProgress(s.observer, WorkerProgressEvent{
		Status:     "running",
		Message:    strings.TrimSpace(message),
		OccurredAt: time.Now().UTC(),
	})
}

func (s *workerRunningSignal) Observed() bool {
	if s == nil {
		return false
	}
	return s.seen.Load()
}

type workerProgressWriter struct {
	buffer  *bytes.Buffer
	running *workerRunningSignal
	message string
}

func (w *workerProgressWriter) Write(p []byte) (int, error) {
	if len(bytes.TrimSpace(p)) > 0 {
		w.running.Report(w.message)
	}
	return w.buffer.Write(p)
}

func emitWorkerProgress(observer WorkerProgressObserver, event WorkerProgressEvent) {
	if observer == nil {
		return
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	observer(event)
}

func validateWorkerLaunchConfig(config WorkerConfig) error {
	root := strings.TrimSpace(config.Root)
	if root == "" {
		return nil
	}
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("worker startup failed: invalid working directory %q: %w", root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("worker startup failed: working directory %q is not a directory", root)
	}
	return nil
}

func workerHeartbeatInterval(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		timeout = defaultWorkerTimeout
	}
	interval := timeout / 20
	if interval < minWorkerHeartbeatInterval {
		interval = minWorkerHeartbeatInterval
	}
	if interval > defaultWorkerHeartbeatInterval {
		interval = defaultWorkerHeartbeatInterval
	}
	return interval
}

func combinedWorkerOutput(stdout, stderr string) string {
	return strings.TrimSpace(strings.TrimSpace(stdout) + "\n" + strings.TrimSpace(stderr))
}

func classifyWorkerExecutionError(err error, stderr string, runningObserved bool) error {
	detail := strings.TrimSpace(stderr)
	prefix := "codex exec failed"
	if !runningObserved {
		prefix = "worker startup failed"
	}
	if detail != "" {
		return fmt.Errorf("%s: %w (stderr: %s)", prefix, err, detail)
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

func classifyWorkerFinalMessageError(action string, err error, runningObserved bool) error {
	prefix := strings.TrimSpace(action)
	if prefix == "" {
		prefix = "worker final message error"
	}
	if !runningObserved {
		return fmt.Errorf("worker startup failed before proof of life: %s: %w", prefix, err)
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

func runningInGoTest() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}
