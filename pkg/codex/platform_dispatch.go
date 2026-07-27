package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	envActivePlatform   = "AETHER_ACTIVE_PLATFORM"
	envWorkerPlatform   = "AETHER_WORKER_PLATFORM"
	envClaudePath       = "AETHER_CLAUDE_PATH"
	envOpenCodePath     = "AETHER_OPENCODE_PATH"
	envOpenCodePrimary  = "AETHER_OPENCODE_PRIMARY_AGENT"
	envOpenCodeAgentURL = "AETHER_OPENCODE_AGENT_URL"
	defaultProbeTimout  = 3 * time.Second
)

const defaultOpenCodePrimaryAgent = "aether-worker-router"

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
	// Category is the stable machine-readable diagnostic contract. Reason is prose.
	Category AvailabilityCategory `json:"category"`
	Reason   string               `json:"reason,omitempty"`
}

type AvailabilityCategory string

const (
	AvailabilityCategoryAvailable           AvailabilityCategory = "available"
	AvailabilityCategoryBinaryMissing       AvailabilityCategory = "binary_missing"
	AvailabilityCategoryAuthProbeFailed     AvailabilityCategory = "auth_probe_failed"
	AvailabilityCategoryAuthInactive        AvailabilityCategory = "auth_inactive"
	AvailabilityCategoryInvalidAuthOutput   AvailabilityCategory = "invalid_auth_output"
	AvailabilityCategoryCredentialsMissing  AvailabilityCategory = "credentials_missing"
	AvailabilityCategoryProbeSkipped        AvailabilityCategory = "probe_skipped"
	AvailabilityCategoryUnsupportedProvider AvailabilityCategory = "unsupported_provider"
	AvailabilityCategoryProviderConfig      AvailabilityCategory = "provider_config_invalid"
)

type PlatformDispatcher interface {
	WorkerInvoker
	Platform() Platform
	Availability(ctx context.Context) AvailabilityStatus
}

type WorkerProviderPreflighter interface {
	Preflight(ctx context.Context, root string) AvailabilityStatus
}

type selectionMetadata interface {
	ActivePlatform() Platform
	// CandidateStatuses returns ordered evaluated candidates, stopping after selection.
	CandidateStatuses() []AvailabilityStatus
}

type markdownAgentDefinition struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type openCodePermissionDefinition struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Mode        string          `yaml:"mode"`
	Tools       map[string]bool `yaml:"tools"`
	Permission  struct {
		ExternalDirectory string `yaml:"external_directory"`
	} `yaml:"permission"`
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
		return AvailabilityStatus{
			Platform:  PlatformUnknown,
			Available: false,
			Category:  AvailabilityCategoryBinaryMissing,
			Reason:    "no platform dispatcher selected",
		}
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
	if contract, ok := PlatformContractFor(s.selected.Platform()); !ok || !contract.canDispatchWorkers() {
		return WorkerResult{}, fmt.Errorf("worker dispatcher unavailable: platform %s does not support worker dispatch", s.selected.Platform())
	}
	return s.selected.Invoke(ctx, config)
}

func (s *SelectedInvoker) InvokeWithProgress(ctx context.Context, config WorkerConfig, observer WorkerProgressObserver) (WorkerResult, error) {
	if s == nil || s.selected == nil {
		return WorkerResult{}, fmt.Errorf("worker dispatcher unavailable: no platform selected")
	}
	if contract, ok := PlatformContractFor(s.selected.Platform()); !ok || !contract.canDispatchWorkers() {
		return WorkerResult{}, fmt.Errorf("worker dispatcher unavailable: platform %s does not support worker dispatch", s.selected.Platform())
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

func (s *SelectedInvoker) Preflight(ctx context.Context, root string) AvailabilityStatus {
	if s == nil || s.selected == nil {
		return AvailabilityStatus{
			Platform:  PlatformUnknown,
			Available: false,
			Category:  AvailabilityCategoryBinaryMissing,
			Reason:    "no platform dispatcher selected",
		}
	}
	if preflighter, ok := s.selected.(WorkerProviderPreflighter); ok {
		return preflighter.Preflight(ctx, root)
	}
	return s.selected.Availability(ctx)
}

func (u *UnavailableInvoker) Platform() Platform { return PlatformUnknown }

func (u *UnavailableInvoker) Availability(ctx context.Context) AvailabilityStatus {
	return AvailabilityStatus{
		Platform:  PlatformUnknown,
		Available: false,
		Category:  availabilityCategoryForStatuses(u.available),
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

func (u *UnavailableInvoker) Preflight(ctx context.Context, root string) AvailabilityStatus {
	return u.Availability(ctx)
}

func IsAgentDelegateSession() bool {
	for _, key := range []string{"CLAUDE_CODE_SIMPLE", "OPENCODE_AGENT", "AETHER_AGENT_DELEGATE"} {
		if strings.TrimSpace(os.Getenv(key)) == "1" {
			return true
		}
	}
	return false
}

func ShouldUseAgentDelegatePath() bool {
	if !IsAgentDelegateSession() {
		return false
	}
	switch DetectActivePlatform() {
	case PlatformClaude, PlatformOpenCode:
		return true
	default:
		return false
	}
}

func AgentDelegateFallbackReason() string {
	platform := DetectActivePlatform()
	if platform == PlatformUnknown {
		return "agent-delegate session detected; active platform is unknown"
	}
	if platform != PlatformClaude && platform != PlatformOpenCode {
		return fmt.Sprintf("agent-delegate session detected on %s; nested worker dispatch remains local", platform)
	}
	return fmt.Sprintf("agent-delegate session detected on %s; host platform must dispatch workers directly", platform)
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

	local := localAgentDefinitionPath(root, platform, base)
	// Hosted CLIs resolve project-local agents before global agents. Validate
	// and report the same file the provider will actually select.
	if (platform == PlatformClaude || platform == PlatformOpenCode) && fileExists(local) {
		return local
	}
	if isAetherSourceRoot(root) {
		return local
	}

	if global := globalAgentDefinitionPath(platform, base); fileExists(global) {
		return global
	}
	if hub := hubAgentDefinitionPath(platform, base); fileExists(hub) {
		return hub
	}
	if fileExists(local) {
		return local
	}

	if global := globalAgentDefinitionPath(platform, base); global != "" {
		return global
	}
	if hub := hubAgentDefinitionPath(platform, base); hub != "" {
		return hub
	}
	return local
}

func localAgentDefinitionPath(root string, platform Platform, base string) string {
	switch platform {
	case PlatformClaude:
		return filepath.Join(root, ".claude", "agents", "ant", base+".md")
	case PlatformOpenCode:
		return filepath.Join(root, ".opencode", "agents", base+".md")
	default:
		return filepath.Join(root, ".codex", "agents", base+".toml")
	}
}

func globalAgentDefinitionPath(platform Platform, base string) string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	switch platform {
	case PlatformClaude:
		return filepath.Join(home, ".claude", "agents", "ant", base+".md")
	case PlatformOpenCode:
		return filepath.Join(home, ".config", "opencode", "agents", base+".md")
	default:
		return filepath.Join(home, ".codex", "agents", base+".toml")
	}
}

func hubAgentDefinitionPath(platform Platform, base string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	hub := strings.TrimSpace(os.Getenv("AETHER_HUB_DIR"))
	if hub == "" && strings.TrimSpace(home) != "" {
		hub = filepath.Join(home, defaultAetherHubDirName())
	}
	if hub == "" {
		return ""
	}
	switch platform {
	case PlatformClaude:
		return filepath.Join(hub, "system", "agents-claude", base+".md")
	case PlatformOpenCode:
		return filepath.Join(hub, "system", "agents", base+".md")
	default:
		return filepath.Join(hub, "system", "codex", base+".toml")
	}
}

func defaultAetherHubDirName() string {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("AETHER_CHANNEL")), "dev") {
		return ".aether-dev"
	}
	if len(os.Args) > 0 {
		binary := strings.ToLower(strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0])))
		if binary == "aether-dev" {
			return ".aether-dev"
		}
	}
	return ".aether"
}

func isAetherSourceRoot(root string) bool {
	if strings.TrimSpace(root) == "" {
		return false
	}
	goMod := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(goMod)
	if err != nil || !strings.Contains(string(data), "github.com/calcosmic/Aether") {
		return false
	}
	if _, err := os.Stat(filepath.Join(root, "cmd", "aether", "main.go")); err != nil {
		return false
	}
	return true
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func SelectPlatformInvoker(ctx context.Context) WorkerInvoker {
	active := DetectActivePlatform()
	preferences := defaultWorkerPlatformPreferences(active)
	explicitOverride := PlatformUnknown
	if rawOverride := strings.TrimSpace(os.Getenv(envWorkerPlatform)); rawOverride != "" {
		override := normalizePlatform(rawOverride)
		if override == PlatformUnknown {
			return &UnavailableInvoker{
				active:    active,
				available: []AvailabilityStatus{unsupportedWorkerPlatformStatus(rawOverride)},
			}
		}
		if override != PlatformFake {
			explicitOverride = override
			preferences = []Platform{override}
		}
	}

	dispatchers := reorderDispatchers([]PlatformDispatcher{
		NewCodexDispatcher(),
		NewClaudeDispatcher(),
		NewOpenCodeDispatcher(),
	}, preferences...)
	if explicitOverride != PlatformUnknown {
		for _, dispatcher := range dispatchers {
			if dispatcher.Platform() == explicitOverride {
				dispatchers = []PlatformDispatcher{dispatcher}
				break
			}
		}
	}

	statuses := make([]AvailabilityStatus, 0, len(dispatchers))
	for _, dispatcher := range dispatchers {
		contract, supported := PlatformContractFor(dispatcher.Platform())
		if !supported || !contract.canDispatchWorkers() {
			statuses = append(statuses, AvailabilityStatus{
				Platform:  dispatcher.Platform(),
				Available: false,
				Category:  AvailabilityCategoryUnsupportedProvider,
				Reason:    fmt.Sprintf("platform %s does not support Aether worker dispatch", dispatcher.Platform()),
			})
			continue
		}
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

func unsupportedWorkerPlatformStatus(value string) AvailabilityStatus {
	value = safeWorkerPlatformOverride(value)
	return AvailabilityStatus{
		Platform:  PlatformUnknown,
		Binary:    value,
		Available: false,
		Category:  AvailabilityCategoryUnsupportedProvider,
		Reason:    fmt.Sprintf("unsupported %s %q; expected codex, claude, or opencode", envWorkerPlatform, value),
	}
}

func safeWorkerPlatformOverride(value string) string {
	value = strings.TrimSpace(sanitizeWorkerDiagnosticOutput(value))
	if value == "" {
		return "(empty)"
	}
	runes := []rune(value)
	if len(runes) <= 80 {
		return value
	}
	return string(runes[:80]) + "..."
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
		status.Category = AvailabilityCategoryBinaryMissing
		status.Reason = fmt.Sprintf("%s binary %q not found in PATH", status.Platform, status.Binary)
		return status
	}
	if !shouldProbeCLIAuth(status.Binary, "codex") {
		status.Available = true
		status.Category = AvailabilityCategoryProbeSkipped
		status.Reason = formatAvailabilityProbeSkipped(status.Platform, status.Binary)
		return status
	}
	output, err := runAvailabilityProbe(ctx, status.Binary, "login", "status")
	if err != nil {
		status.Category = AvailabilityCategoryAuthProbeFailed
		status.Reason = formatAvailabilityProbeError(status.Platform, "login status", err, output)
		return status
	}
	if codexLoginStatusIsActive(output) {
		status.Available = true
		status.Category = AvailabilityCategoryAvailable
		return status
	}
	status.Category = AvailabilityCategoryAuthInactive
	status.Reason = fmt.Sprintf("%s login status did not confirm an authenticated session", status.Platform)
	return status
}

func (c *ClaudeDispatcher) Availability(ctx context.Context) AvailabilityStatus {
	status := AvailabilityStatus{Platform: PlatformClaude, Binary: strings.TrimSpace(c.binaryName)}
	if status.Binary == "" {
		status.Binary = "claude"
	}
	if _, err := exec.LookPath(status.Binary); err != nil {
		status.Category = AvailabilityCategoryBinaryMissing
		status.Reason = fmt.Sprintf("%s binary %q not found in PATH", status.Platform, status.Binary)
		return status
	}
	if !shouldProbeCLIAuth(status.Binary, "claude") {
		status.Available = true
		status.Category = AvailabilityCategoryProbeSkipped
		status.Reason = formatAvailabilityProbeSkipped(status.Platform, status.Binary)
		return status
	}
	output, err := runAvailabilityProbe(ctx, status.Binary, "auth", "status", "--json")
	if err != nil {
		status.Category = AvailabilityCategoryAuthProbeFailed
		status.Reason = formatAvailabilityProbeError(status.Platform, "auth status", err, output)
		return status
	}
	var payload struct {
		LoggedIn bool `json:"loggedIn"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		status.Category = AvailabilityCategoryInvalidAuthOutput
		status.Reason = fmt.Sprintf("%s auth status returned invalid JSON: %v", status.Platform, err)
		return status
	}
	if !payload.LoggedIn {
		status.Category = AvailabilityCategoryAuthInactive
		status.Reason = fmt.Sprintf("%s auth status reported no active login", status.Platform)
		return status
	}
	status.Available = true
	status.Category = AvailabilityCategoryAvailable
	return status
}

func (o *OpenCodeDispatcher) Availability(ctx context.Context) AvailabilityStatus {
	status := AvailabilityStatus{Platform: PlatformOpenCode, Binary: strings.TrimSpace(o.binaryName)}
	if status.Binary == "" {
		status.Binary = "opencode"
	}
	if _, err := exec.LookPath(status.Binary); err != nil {
		status.Category = AvailabilityCategoryBinaryMissing
		status.Reason = fmt.Sprintf("%s binary %q not found in PATH", status.Platform, status.Binary)
		return status
	}
	if !shouldProbeCLIAuth(status.Binary, "opencode") {
		status.Available = true
		status.Category = AvailabilityCategoryProbeSkipped
		status.Reason = formatAvailabilityProbeSkipped(status.Platform, status.Binary)
		return status
	}
	output, err := runAvailabilityProbe(ctx, status.Binary, "auth", "list")
	if err != nil {
		status.Category = AvailabilityCategoryAuthProbeFailed
		status.Reason = formatAvailabilityProbeError(status.Platform, "auth list", err, output)
		return status
	}
	if countOpenCodeCredentials(output) == 0 {
		status.Category = AvailabilityCategoryCredentialsMissing
		status.Reason = fmt.Sprintf("%s auth list reported no configured credentials or environment keys", status.Platform)
		return status
	}
	status.Available = true
	status.Category = AvailabilityCategoryAvailable
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

func (c *ClaudeDispatcher) Preflight(ctx context.Context, root string) AvailabilityStatus {
	status := c.Availability(ctx)
	if !status.Available {
		return status
	}

	// --strict-mcp-config keeps the probe from booting the target repo's MCP
	// servers, plugins, and hooks just to answer a liveness question. That
	// startup cost scales with project config and is what pushed the probe
	// past its timeout (and cost ~$0.59/probe in a heavily configured repo).
	args := []string{
		"-p",
		"Return exactly OK.",
		"--output-format", "json",
		"--permission-mode", "plan",
		"--strict-mcp-config",
	}
	return runHostedProviderPreflight(ctx, status, root, args)
}

func (o *OpenCodeDispatcher) Preflight(ctx context.Context, root string) AvailabilityStatus {
	status := o.Availability(ctx)
	if !status.Available {
		return status
	}

	args := []string{
		"run",
		"--agent", openCodePrimaryAgent(),
		"--format", "json",
		"Return exactly OK.",
	}
	return runHostedProviderPreflight(ctx, status, root, args)
}

// The preflight is a real model round-trip, so it inherits cold-start and
// network variance. 20s was tight enough that ordinary contention aborted whole
// commands: the 27 July M4L /ant-plan run died 22 seconds in, while the same
// probe run by hand answered in 5. Declared as vars so tests can shrink the
// wait without waiting out the production budget.
var (
	hostedPreflightTimeout = 45 * time.Second
	// One retry, and only for timeouts. A timeout is the transient case;
	// missing credentials or a bad model name will fail identically twice and
	// should surface immediately.
	hostedPreflightAttempts = 2
)

func runHostedProviderPreflight(ctx context.Context, status AvailabilityStatus, root string, args []string) AvailabilityStatus {
	binary := strings.TrimSpace(status.Binary)
	if binary == "" {
		binary = string(status.Platform)
	}

	var failure AvailabilityStatus
	for attempt := 1; attempt <= hostedPreflightAttempts; attempt++ {
		result, timedOut := attemptHostedProviderPreflight(ctx, status, root, args, binary)
		if result.Available {
			return result
		}
		failure = result
		if !timedOut || ctx.Err() != nil || attempt == hostedPreflightAttempts {
			break
		}
	}
	return failure
}

func attemptHostedProviderPreflight(ctx context.Context, status AvailabilityStatus, root string, args []string, binary string) (AvailabilityStatus, bool) {
	probeCtx, cancel := context.WithTimeout(ctx, hostedPreflightTimeout)
	defer cancel()

	cmd := exec.CommandContext(probeCtx, binary, args...)
	if root := strings.TrimSpace(root); root != "" {
		cmd.Dir = root
	}
	configureWorkerCommand(cmd)
	if status.Platform == PlatformOpenCode {
		if agentURL := os.Getenv(envOpenCodeAgentURL); agentURL != "" {
			cmd.Env = append(os.Environ(), envOpenCodeAgentURL+"="+agentURL)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		raw := strings.TrimSpace(combinedWorkerOutput(stdout.String(), stderr.String()))
		reason := strings.TrimSpace(sanitizeWorkerDiagnosticOutput(raw))
		if reason == "" {
			reason = sanitizeWorkerDiagnosticOutput(err.Error())
		}
		category := AvailabilityCategoryProviderConfig
		timedOut := probeCtx.Err() == context.DeadlineExceeded
		if timedOut {
			reason = fmt.Sprintf("%s provider/model preflight timed out before worker dispatch after %d attempts", status.Platform, hostedPreflightAttempts)
			category = AvailabilityCategoryAuthProbeFailed
		}
		return AvailabilityStatus{
			Platform:  status.Platform,
			Binary:    binary,
			Available: false,
			Category:  category,
			Reason:    fmt.Sprintf("%s provider/model preflight failed before worker dispatch: %s", status.Platform, reason),
		}, timedOut
	}
	return AvailabilityStatus{
		Platform:  status.Platform,
		Binary:    binary,
		Available: true,
		Category:  AvailabilityCategoryAvailable,
	}, false
}

func (c *ClaudeDispatcher) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	return c.InvokeWithProgress(ctx, config, nil)
}

func (o *OpenCodeDispatcher) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	return o.InvokeWithProgress(ctx, config, nil)
}

func (c *ClaudeDispatcher) InvokeWithProgress(ctx context.Context, config WorkerConfig, observer WorkerProgressObserver) (WorkerResult, error) {
	start := time.Now()
	permission, permissionErr := ResolvePermissionDecision(PlatformClaude, config.Caste, config.PermissionProfile)
	if permissionErr != nil {
		return permissionDeniedWorkerResult(config, start, permissionErr)
	}
	config.PermissionProfile = permission.Profile
	// Scout-aware schema: without this, the Claude path handed a scout a
	// schema whose additionalProperties:false forbade the scout_report the
	// task brief demanded — a contradiction in the worker's context on every
	// planning run. Codex already used the config-aware variant.
	schemaJSON, err := marshalJSON(workerClaimsSchemaForConfig(config))
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
	prompt := strings.TrimSpace(AssembleHostedPrompt(config.ContextCapsule, config.HandoffSection, config.SkillSection, config.PheromoneSection, config.TaskBrief) + "\n\n" + RenderPermissionProfileSection(permission) + "\n\n" + renderResponseContract(config))
	args := []string{"-p", prompt, "--output-format", "json", "--json-schema", string(schemaJSON), "--agent", strings.TrimSpace(config.AgentName)}
	if permission.Profile.Name == PermissionRepositoryReadOnly {
		args = append(args, "--permission-mode", "plan")
	} else {
		settingsPath, settingsErr := writeClaudeWorkspaceSettings()
		if settingsErr != nil {
			return permissionDeniedWorkerResult(config, start, fmt.Errorf("prepare Claude workspace sandbox: %w", settingsErr))
		}
		defer os.Remove(settingsPath)
		args = append(args, "--permission-mode", "acceptEdits", "--settings", settingsPath)
	}
	return invokeHostedWorker(ctx, c, config, observer, args, "claude")
}

func (o *OpenCodeDispatcher) InvokeWithProgress(ctx context.Context, config WorkerConfig, observer WorkerProgressObserver) (WorkerResult, error) {
	start := time.Now()
	permission, permissionErr := ResolvePermissionDecision(PlatformOpenCode, config.Caste, config.PermissionProfile)
	if permissionErr != nil {
		return permissionDeniedWorkerResult(config, start, permissionErr)
	}
	if primary := openCodePrimaryAgent(); primary != defaultOpenCodePrimaryAgent {
		return permissionDeniedWorkerResult(config, start, fmt.Errorf(
			"permission profile %q requires OpenCode primary router %q; configured router is %q",
			permission.Profile.Name, defaultOpenCodePrimaryAgent, primary,
		))
	}
	if err := validateOpenCodePermissionBoundary(config.AgentTOMLPath, permission.Profile); err != nil {
		return permissionDeniedWorkerResult(config, start, err)
	}
	if err := validateOpenCodeRouter(config.Root); err != nil {
		return permissionDeniedWorkerResult(config, start, err)
	}
	config.PermissionProfile = permission.Profile
	args := []string{"run", "--agent", openCodePrimaryAgent(), "--format", "json"}
	workerPrompt := strings.TrimSpace(AssembleHostedPrompt(config.ContextCapsule, config.HandoffSection, config.SkillSection, config.PheromoneSection, config.TaskBrief) + "\n\n" + RenderPermissionProfileSection(permission) + "\n\n" + renderResponseContract(config))
	prompt := renderOpenCodeSubagentDispatchPrompt(config, workerPrompt)
	args = append(args, prompt)
	return invokeHostedWorker(ctx, o, config, observer, args, "opencode")
}

func writeClaudeWorkspaceSettings() (string, error) {
	settings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"defaultMode":                  "acceptEdits",
			"disableBypassPermissionsMode": "disable",
		},
		"sandbox": map[string]interface{}{
			"enabled":                  true,
			"failIfUnavailable":        true,
			"autoAllowBashIfSandboxed": true,
			"allowUnsandboxedCommands": false,
		},
	}
	payload, err := json.Marshal(settings)
	if err != nil {
		return "", err
	}
	return writeTempFile("", "aether-claude-settings-*.json", payload)
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
	observer = synchronizedWorkerProgressObserver(observer)
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
	configureWorkerCommand(cmd)

	if agentURL := os.Getenv(envOpenCodeAgentURL); agentURL != "" {
		cmd.Env = append(os.Environ(), envOpenCodeAgentURL+"="+agentURL)
	}

	var stdout, stderr bytes.Buffer
	running := newWorkerRunningSignal(observer)
	cmd.Stdout = &workerProgressWriter{buffer: &stdout, running: running, message: "worker output observed"}
	cmd.Stderr = &workerProgressWriter{buffer: &stderr, running: running, message: "worker stderr observed"}
	if err := cmd.Start(); err != nil {
		startupErr := fmt.Errorf("worker startup failed: %s start failed: %w", label, err)
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "failed", Duration: time.Since(start), Error: startupErr}, startupErr
	}
	GlobalProcessTracker().TrackProcess(cmd.Process.Pid, TrackedProcess{
		WorkerName:    config.WorkerName,
		TaskID:        config.TaskID,
		Caste:         config.Caste,
		Platform:      label,
		Root:          workerTrackingRoot(config),
		ProviderRunID: config.ProviderRunID,
		Binding:       config.ExecutionBinding,
	})
	defer GlobalProcessTracker().UntrackProcess(cmd.Process.Pid)
	emitWorkerProgress(observer, WorkerProgressEvent{
		Status:     "running",
		Message:    "provider process started",
		OccurredAt: time.Now().UTC(),
		ProcessID:  cmd.Process.Pid,
	})

	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	heartbeat := time.NewTicker(workerHeartbeatInterval(timeout))
	defer heartbeat.Stop()

	var waitErr error
	ctxDone := ctx.Done()
waitLoop:
	for {
		select {
		case waitErr = <-waitCh:
			break waitLoop
		case <-ctxDone:
			ctxDone = nil
			running.Pulse("worker cancellation requested")
		case <-heartbeat.C:
			running.Pulse("worker heartbeat observed")
		}
	}

	duration := time.Since(start)
	rawOutput := combinedWorkerOutput(stdout.String(), stderr.String())
	safeRawOutput := sanitizeWorkerDiagnosticOutput(rawOutput)
	if ctx.Err() == context.DeadlineExceeded {
		reportedTimeout := duration.Round(time.Millisecond)
		if duration >= time.Second {
			reportedTimeout = duration.Round(time.Second)
		}
		timeoutErr := fmt.Errorf("worker timeout after %v", reportedTimeout)
		if debugPath := writeHostedWorkerOutputDebug(workerTrackingRoot(config), label, config, args, stdout.String(), stderr.String(), timeoutErr, hostedWorkerDebugDetails{Duration: duration, ExitCode: -1, FailureMode: "timeout"}); debugPath != "" {
			timeoutErr = fmt.Errorf("%w (debug: %s)", timeoutErr, debugPath)
		}
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "timeout", Duration: duration, RawOutput: safeRawOutput, Error: timeoutErr}, nil
	}
	if waitErr != nil {
		execErr := classifyHostedExecutionError(label, waitErr, stderr.String(), running.Observed())
		exitCode := -1
		var exitError *exec.ExitError
		if errors.As(waitErr, &exitError) {
			exitCode = exitError.ExitCode()
		}
		if debugPath := writeHostedWorkerOutputDebug(workerTrackingRoot(config), label, config, args, stdout.String(), stderr.String(), execErr, hostedWorkerDebugDetails{Duration: duration, ExitCode: exitCode, FailureMode: "non_zero_exit"}); debugPath != "" {
			execErr = fmt.Errorf("%w (debug: %s)", execErr, debugPath)
		}
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "failed", Duration: duration, RawOutput: safeRawOutput, Error: execErr}, nil
	}
	// Provider failures arrive as well-formed JSON on stdout with exit code 0,
	// so they reach this point looking like output to parse. Detect them
	// before claims parsing: an error envelope means no claims exist, and
	// running the parser first can only produce a mislabelled parse error.
	if providerErr, ok := detectProviderErrorEnvelope(rawOutput); ok {
		err := error(providerErr)
		if strings.EqualFold(label, "opencode") && looksLikeOpenCodeLocalServerFailure(providerErr.URL+" "+providerErr.Message) {
			err = fmt.Errorf("%w; the local OpenCode server rejected the run request — ensure OpenCode is running for `opencode run`, or set AETHER_WORKER_PLATFORM=claude/codex", providerErr)
		}
		if debugPath := writeHostedWorkerOutputDebug(workerTrackingRoot(config), label, config, args, stdout.String(), stderr.String(), err, hostedWorkerDebugDetails{Duration: duration, ExitCode: -1, FailureMode: "provider_error_envelope"}); debugPath != "" {
			err = fmt.Errorf("%w (debug: %s)", err, debugPath)
		}
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "failed", Duration: duration, RawOutput: safeRawOutput, Error: err}, nil
	}
	claims, parseErr := parseHostedWorkerOutput(label, rawOutput)
	if parseErr != nil {
		err := classifyWorkerFinalMessageError("parse worker output", parseErr, running.Observed())
		if debugPath := writeHostedWorkerOutputDebug(workerTrackingRoot(config), label, config, args, stdout.String(), stderr.String(), parseErr, hostedWorkerDebugDetails{Duration: duration, ExitCode: -1, FailureMode: "parse_failure"}); debugPath != "" {
			err = fmt.Errorf("%w (debug: %s)", err, debugPath)
		}
		return WorkerResult{WorkerName: config.WorkerName, Caste: config.Caste, TaskID: config.TaskID, Status: "failed", Duration: duration, RawOutput: safeRawOutput, Error: err}, nil
	}
	claims = normalizeWorkerClaims(claims, config)
	return hostedWorkerResultFromClaims(config, claims, duration, safeRawOutput), nil
}

// hostedWorkerResultFromClaims maps parsed claims onto the WorkerResult the
// colony consumes. Every content field must cross this seam: Artifacts and
// ScoutReport were once omitted here, so a scout's research parsed
// successfully and was then discarded — scoutReportFromWorkerResult found
// both fields empty and planning fell back to synthesized boilerplate. The
// Codex path (worker.go) and FakeInvoker always carried them; hosted must
// match, and TestHostedWorkerResultCarriesAllClaimsContent pins it.
func hostedWorkerResultFromClaims(config WorkerConfig, claims workerClaims, duration time.Duration, safeRawOutput string) WorkerResult {
	return WorkerResult{
		WorkerName:    config.WorkerName,
		Caste:         config.Caste,
		TaskID:        config.TaskID,
		Status:        claims.Status,
		Summary:       claims.Summary,
		FilesCreated:  claims.FilesCreated,
		FilesModified: claims.FilesModified,
		TestsWritten:  claims.TestsWritten,
		Artifacts:     claims.Artifacts,
		ScoutReport:   claims.ScoutReport,
		ToolCount:     claims.ToolCount,
		Blockers:      claims.Blockers,
		Spawns:        claims.Spawns,
		Handoff:       claims.Handoff,
		Duration:      duration,
		RawOutput:     safeRawOutput,
	}
}

func parseHostedWorkerOutput(label, output string) (workerClaims, error) {
	platform := strings.ToLower(strings.TrimSpace(label))
	var hostedErr error
	switch platform {
	case "claude", "opencode":
		claims, err := parseHostedJSONWorkerOutput(platform, output)
		if err == nil {
			return claims, nil
		}
		hostedErr = err
	}
	claims, err := ParseWorkerOutput(output)
	if err == nil {
		return claims, nil
	}
	if hostedErr != nil {
		// Report why the worker's actual answer was rejected, not why the CLI
		// transport envelope wrapped around it is not worker claims. The
		// generic fallback can only ever say "no JSON found in output", which
		// sent the 27 July M4L investigation after a nonexistent code-fence
		// bug while the real cause was a field type mismatch.
		return workerClaims{}, fmt.Errorf("%w (envelope fallback: %v)", hostedErr, err)
	}
	return workerClaims{}, err
}

func parseHostedJSONWorkerOutput(label, output string) (workerClaims, error) {
	candidates := hostedJSONTextCandidates(output)
	var firstErr error
	for i := len(candidates) - 1; i >= 0; i-- {
		claims, err := ParseWorkerOutput(candidates[i])
		if err == nil {
			return claims, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if len(candidates) > 0 {
		if claims, err := ParseWorkerOutput(strings.Join(candidates, "\n")); err == nil {
			return claims, nil
		}
	}
	if firstErr != nil {
		// firstErr comes from ParseWorkerOutput and is already self-labelled;
		// stacking another "parse ... output" prefix produced the doubled
		// messages users saw.
		return workerClaims{}, firstErr
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

// WorkerDebugRetentionMaxAge and WorkerDebugRetentionMaxFiles are the
// worker-debug artifact retention policy: files older than the max age are
// pruned first, then, if more than the file cap remain, the oldest are
// removed until the cap is met. Declared once here (the package that owns
// the .aether/data/worker-debug directory layout) and referenced by both
// the write-time prune below and `aether data-clean`'s worker-debug step
// (cmd/maintenance.go) so the two prune paths cannot silently drift apart.
// 50 files / 14 days was chosen as generous enough to cover a week of heavy
// debugging without letting the directory grow unbounded on disk.
const (
	WorkerDebugRetentionMaxAge   = 14 * 24 * time.Hour
	WorkerDebugRetentionMaxFiles = 50
)

// hostedWorkerDebugDetails carries per-failure-mode facts that the debug
// writer cannot infer from config, args, or cause alone. Kept as a struct
// rather than positional parameters so a fifth failure mode doesn't require
// a fifth argument at every call site.
type hostedWorkerDebugDetails struct {
	// Duration is the worker's elapsed wall-clock time.
	Duration time.Duration
	// ExitCode is the subprocess exit status. Use -1 when no exit code is
	// available (timeout, provider error envelope, parse failure) rather
	// than a fabricated 0, so a reader never has to guess whether the
	// field was absent or genuinely zero.
	ExitCode int
	// FailureMode is one of "timeout", "non_zero_exit",
	// "provider_error_envelope", "parse_failure" — lets a reader tell the
	// four failure paths apart without inferring from the error text.
	FailureMode string
}

func writeHostedWorkerOutputDebug(root, label string, config WorkerConfig, args []string, stdoutText, stderrText string, cause error, details hostedWorkerDebugDetails) string {
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
		"created_at":      now.Format(time.RFC3339Nano),
		"platform":        strings.TrimSpace(label),
		"worker_name":     strings.TrimSpace(config.WorkerName),
		"caste":           strings.TrimSpace(config.Caste),
		"task_id":         strings.TrimSpace(config.TaskID),
		"agent_name":      strings.TrimSpace(config.AgentName),
		"args":            safeHostedWorkerArgs(args),
		"stdout_bytes":    len(stdoutText),
		"stderr_bytes":    len(stderrText),
		"stdout_excerpt":  workerOutputExcerpt(stdoutText),
		"stderr_excerpt":  workerOutputExcerpt(stderrText),
		"error":           sanitizeWorkerDiagnosticOutput(cause.Error()),
		"duration_ms":     details.Duration.Milliseconds(),
		"exit_code":       details.ExitCode,
		"provider_run_id": strings.TrimSpace(config.ProviderRunID),
		"failure_mode":    strings.TrimSpace(details.FailureMode),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ""
	}
	if err := os.WriteFile(filepath.Join(debugDir, filename), data, 0644); err != nil {
		return ""
	}
	// Pruning is best-effort: the artifact just written is the operator's
	// evidence for the failure they're about to see, and losing it to a
	// housekeeping error would recreate the exact problem this function
	// exists to fix. Never let pruning fail the write, and never prune the
	// file just written even if the clock is skewed.
	pruneWorkerDebugArtifacts(debugDir, filename)
	return relPath
}

// pruneWorkerDebugArtifacts enforces WorkerDebugRetentionMaxAge and
// WorkerDebugRetentionMaxFiles against dir, in that order: age-based
// removal first, then cap-based oldest-first removal of whatever remains.
// keepFilename is never removed, even if its mod time would otherwise
// qualify it for pruning. Errors reading or removing individual files are
// ignored — see the comment at the call site for why.
func pruneWorkerDebugArtifacts(dir, keepFilename string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type debugArtifact struct {
		name    string
		modTime time.Time
	}
	var artifacts []debugArtifact
	cutoff := time.Now().Add(-WorkerDebugRetentionMaxAge)
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == keepFilename {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
			continue
		}
		artifacts = append(artifacts, debugArtifact{name: entry.Name(), modTime: info.ModTime()})
	}
	if len(artifacts) <= WorkerDebugRetentionMaxFiles-1 {
		return
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].modTime.Before(artifacts[j].modTime) })
	excess := len(artifacts) - (WorkerDebugRetentionMaxFiles - 1)
	for i := 0; i < excess; i++ {
		_ = os.Remove(filepath.Join(dir, artifacts[i].name))
	}
}

// safeHostedWorkerArgs prepares an arg vector for the debug artifact. It
// redacts by argument identity, not position: the old positional rule
// ("redact the last arg") was built for the OpenCode vector where the prompt
// is last, but on the Claude vector the last arg is the --permission-mode
// value — so debug files recorded `"--permission-mode", "<prompt: 4 bytes>"`
// while the actual 19 KB prompt at index 1 was written to disk verbatim,
// bypassing the diagnostic sanitizer entirely.
func safeHostedWorkerArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	// Values following these flags are large or sensitive payloads, not
	// diagnostics. Everything else (flag names, modes, agent names, paths)
	// is diagnostic gold and stays readable.
	redactedFlags := map[string]string{
		"-p":                     "prompt",
		"--print":                "prompt",
		"--json-schema":          "schema",
		"--system-prompt":        "system prompt",
		"--append-system-prompt": "system prompt",
	}
	out := make([]string, len(args))
	redactNext := ""
	for i, arg := range args {
		if redactNext != "" {
			out[i] = fmt.Sprintf("<%s: %d bytes>", redactNext, len(arg))
			redactNext = ""
			continue
		}
		if label, ok := redactedFlags[arg]; ok {
			out[i] = arg
			redactNext = label
			continue
		}
		out[i] = sanitizeWorkerDiagnosticOutput(arg)
	}
	// Fallback for vectors that pass the prompt as a bare trailing positional
	// (the OpenCode `run ... <prompt>` shape): a long final arg that is not a
	// flag value is the prompt.
	if redactNext == "" && len(args) >= 2 {
		last := args[len(args)-1]
		prev := args[len(args)-2]
		if !strings.HasPrefix(last, "-") && !strings.HasPrefix(prev, "--") && len(last) > 200 {
			out[len(out)-1] = fmt.Sprintf("<prompt: %d bytes>", len(last))
		}
	}
	return out
}

// workerOutputExcerpt keeps both ends of an oversized worker output. A
// head-only excerpt spends its whole budget on the CLI's transport envelope
// (usage counters, cache statistics, session ids) and cuts away the tail of the
// worker's answer — which is exactly where a malformed field sits. The 27 July
// M4L debug artifacts were unusable for that reason: diagnosing them needed the
// raw CLI session transcript instead.
func workerOutputExcerpt(value string) string {
	value = sanitizeWorkerDiagnosticOutput(value)
	const (
		headLimit = 3000
		tailLimit = 3000
	)
	runes := []rune(value)
	if len(runes) <= headLimit+tailLimit {
		return value
	}
	omitted := len(runes) - headLimit - tailLimit
	return string(runes[:headLimit]) +
		fmt.Sprintf("\n[... %d characters omitted ...]\n", omitted) +
		string(runes[len(runes)-tailLimit:])
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

func defaultWorkerPlatformPreferences(active Platform) []Platform {
	preferences := []Platform{PlatformClaude}
	if active != PlatformUnknown && active != PlatformClaude {
		preferences = append(preferences, active)
	}
	preferences = append(preferences, PlatformCodex, PlatformOpenCode)
	return preferences
}

func reorderDispatchers(dispatchers []PlatformDispatcher, preferred ...Platform) []PlatformDispatcher {
	seen := make(map[Platform]struct{}, len(dispatchers))
	out := make([]PlatformDispatcher, 0, len(dispatchers))
	for _, platform := range preferred {
		if platform == PlatformUnknown {
			continue
		}
		for _, dispatcher := range dispatchers {
			if dispatcher.Platform() != platform {
				continue
			}
			if _, ok := seen[dispatcher.Platform()]; ok {
				continue
			}
			out = append(out, dispatcher)
			seen[dispatcher.Platform()] = struct{}{}
		}
	}
	for _, dispatcher := range dispatchers {
		if _, ok := seen[dispatcher.Platform()]; ok {
			continue
		}
		out = append(out, dispatcher)
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

func formatAvailabilityProbeError(platform Platform, action string, err error, _ string) string {
	detail := sanitizedProbeFailureDetail(err)
	if detail != "" {
		return fmt.Sprintf("%s %s failed: %s; sensitive details omitted", platform, action, detail)
	}
	return fmt.Sprintf("%s %s failed; sensitive details omitted", platform, action)
}

func formatAvailabilityProbeSkipped(platform Platform, binary string) string {
	name := strings.TrimSpace(filepath.Base(binary))
	if name == "" {
		name = "override binary"
	}
	return fmt.Sprintf("%s auth probe skipped for override binary %q; provider authentication was not verified", platform, name)
}

func sanitizedProbeFailureDetail(err error) string {
	if err == nil {
		return ""
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if code := exitErr.ExitCode(); code >= 0 {
			return fmt.Sprintf("exit status %d", code)
		}
		return "process exited unsuccessfully"
	}
	if strings.EqualFold(strings.TrimSpace(err.Error()), "timed out") {
		return "timed out"
	}
	return "probe command failed"
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

func codexLoginStatusIsActive(output string) bool {
	cleaned := strings.ToLower(stripANSIEscapeCodes(output))
	if !strings.Contains(cleaned, "logged in") {
		return false
	}
	if negativeCodexLoginStatusPattern.MatchString(cleaned) {
		return false
	}
	return true
}

var negativeCodexLoginStatusPattern = regexp.MustCompile(`\b(?:not|no|never|without)\b.{0,40}\blogged in\b|\bnot authenticated\b|\bunauthenticated\b|\bno authenticated session\b`)

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
	frontmatter, err := markdownFrontmatter(data)
	if err != nil {
		return markdownAgentDefinition{}, err
	}
	var def markdownAgentDefinition
	if err := yaml.Unmarshal(frontmatter, &def); err != nil {
		return markdownAgentDefinition{}, err
	}
	return def, nil
}

func markdownFrontmatter(data []byte) ([]byte, error) {
	text := strings.TrimSpace(string(data))
	if !strings.HasPrefix(text, "---") {
		return nil, fmt.Errorf("missing YAML frontmatter")
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
		return nil, fmt.Errorf("unterminated YAML frontmatter")
	}
	return []byte(strings.Join(lines[1:end], "\n")), nil
}

func parseOpenCodePermissionDefinition(path string) (openCodePermissionDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return openCodePermissionDefinition{}, fmt.Errorf("read %s: %w", path, err)
	}
	frontmatter, err := markdownFrontmatter(data)
	if err != nil {
		return openCodePermissionDefinition{}, fmt.Errorf("%s: %w", path, err)
	}
	var def openCodePermissionDefinition
	if err := yaml.Unmarshal(frontmatter, &def); err != nil {
		return openCodePermissionDefinition{}, fmt.Errorf("%s: %w", path, err)
	}
	return def, nil
}

func validateOpenCodePermissionBoundary(path string, profile PermissionProfile) error {
	def, err := parseOpenCodePermissionDefinition(path)
	if err != nil {
		return fmt.Errorf("OpenCode permission attestation failed: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(def.Permission.ExternalDirectory), "deny") {
		return fmt.Errorf("OpenCode permission attestation failed for %s: external_directory must be denied", def.Name)
	}
	if profile.Name == PermissionRepositoryReadOnly {
		for _, tool := range []string{"write", "edit", "bash"} {
			allowed, declared := def.Tools[tool]
			if !declared || allowed {
				return fmt.Errorf("OpenCode permission attestation failed for %s: read-only profile requires tools.%s=false", def.Name, tool)
			}
		}
	}
	return nil
}

func validateOpenCodeRouter(root string) error {
	path := localAgentDefinitionPath(root, PlatformOpenCode, defaultOpenCodePrimaryAgent)
	if !fileExists(path) {
		path = globalAgentDefinitionPath(PlatformOpenCode, defaultOpenCodePrimaryAgent)
	}
	if !fileExists(path) {
		return fmt.Errorf("OpenCode permission attestation failed: primary router %q is not installed", defaultOpenCodePrimaryAgent)
	}
	def, err := parseOpenCodePermissionDefinition(path)
	if err != nil {
		return fmt.Errorf("OpenCode permission attestation failed: %w", err)
	}
	if def.Name != defaultOpenCodePrimaryAgent || def.Mode != "primary" ||
		def.Tools["write"] || def.Tools["edit"] || def.Tools["bash"] || !def.Tools["task"] ||
		!strings.EqualFold(strings.TrimSpace(def.Permission.ExternalDirectory), "deny") {
		return fmt.Errorf("OpenCode permission attestation failed: router %s does not enforce task-only delegation", path)
	}
	return nil
}

func classifyHostedExecutionError(label string, err error, stderr string, runningObserved bool) error {
	rawDetail := strings.TrimSpace(stderr)
	detail := sanitizeWorkerDiagnosticOutput(stderr)
	prefix := strings.TrimSpace(label)
	if prefix == "" {
		prefix = "worker process"
	}
	if strings.EqualFold(prefix, "opencode") && looksLikeOpenCodeLocalServerFailure(rawDetail) {
		return fmt.Errorf("opencode worker dispatcher unavailable: local OpenCode server rejected the run request; ensure OpenCode is running for `opencode run`, or set AETHER_WORKER_PLATFORM=claude/codex to use another dispatcher: %w (stderr: %s)", err, detail)
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

func looksLikeOpenCodeLocalServerFailure(stderr string) bool {
	text := strings.ToLower(strings.TrimSpace(stderr))
	if text == "" {
		return false
	}
	return strings.Contains(text, "localhost:4000/messages") ||
		(strings.Contains(text, "/messages") && strings.Contains(text, "404"))
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

func availabilityCategoryForStatuses(statuses []AvailabilityStatus) AvailabilityCategory {
	priority := []AvailabilityCategory{
		AvailabilityCategoryUnsupportedProvider,
		AvailabilityCategoryAuthProbeFailed,
		AvailabilityCategoryAuthInactive,
		AvailabilityCategoryInvalidAuthOutput,
		AvailabilityCategoryCredentialsMissing,
		AvailabilityCategoryProviderConfig,
		AvailabilityCategoryBinaryMissing,
		AvailabilityCategoryProbeSkipped,
	}
	for _, category := range priority {
		for _, status := range statuses {
			if status.Category == category && !status.Available {
				return category
			}
		}
	}
	for _, status := range statuses {
		if status.Category != "" {
			return status.Category
		}
	}
	return AvailabilityCategoryBinaryMissing
}
