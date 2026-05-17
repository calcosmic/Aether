package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestAgentDefinitionPathUsesSourceCheckoutLocalAgents(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AETHER_HUB_DIR", filepath.Join(home, ".aether"))

	writeTestFile(t, filepath.Join(root, "go.mod"), "module github.com/calcosmic/Aether\n")
	writeTestFile(t, filepath.Join(root, "cmd", "aether", "main.go"), "package main\n")
	writeTestFile(t, filepath.Join(home, ".codex", "agents", "aether-builder.toml"), "name = \"global\"\n")

	tests := []struct {
		platform Platform
		want     string
	}{
		{PlatformClaude, filepath.Join(root, ".claude", "agents", "ant", "aether-builder.md")},
		{PlatformOpenCode, filepath.Join(root, ".opencode", "agents", "aether-builder.md")},
		{PlatformCodex, filepath.Join(root, ".codex", "agents", "aether-builder.toml")},
	}

	for _, tt := range tests {
		got := AgentDefinitionPath(root, tt.platform, "aether-builder")
		if got != tt.want {
			t.Fatalf("AgentDefinitionPath source %s = %q, want %q", tt.platform, got, tt.want)
		}
	}
}

func TestAgentDefinitionPathUsesGlobalHomesForConsumerRepo(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	hub := filepath.Join(home, ".aether")
	t.Setenv("HOME", home)
	t.Setenv("AETHER_HUB_DIR", hub)

	// Stale local copies should not win in consumer repos.
	writeTestFile(t, filepath.Join(root, ".claude", "agents", "ant", "aether-builder.md"), "local")
	writeTestFile(t, filepath.Join(root, ".opencode", "agents", "aether-builder.md"), "local")
	writeTestFile(t, filepath.Join(root, ".codex", "agents", "aether-builder.toml"), "local")

	claudeGlobal := filepath.Join(home, ".claude", "agents", "ant", "aether-builder.md")
	opencodeGlobal := filepath.Join(home, ".config", "opencode", "agents", "aether-builder.md")
	codexGlobal := filepath.Join(home, ".codex", "agents", "aether-builder.toml")
	writeTestFile(t, claudeGlobal, "global")
	writeTestFile(t, opencodeGlobal, "global")
	writeTestFile(t, codexGlobal, "global")

	tests := []struct {
		platform Platform
		want     string
	}{
		{PlatformClaude, claudeGlobal},
		{PlatformOpenCode, opencodeGlobal},
		{PlatformCodex, codexGlobal},
	}

	for _, tt := range tests {
		got := AgentDefinitionPath(root, tt.platform, "aether-builder")
		if got != tt.want {
			t.Fatalf("AgentDefinitionPath consumer %s = %q, want %q", tt.platform, got, tt.want)
		}
	}
}

func TestAgentDefinitionPathFallsBackToHub(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	hub := filepath.Join(home, ".aether")
	t.Setenv("HOME", home)
	t.Setenv("AETHER_HUB_DIR", hub)

	claudeHub := filepath.Join(hub, "system", "agents-claude", "aether-builder.md")
	opencodeHub := filepath.Join(hub, "system", "agents", "aether-builder.md")
	codexHub := filepath.Join(hub, "system", "codex", "aether-builder.toml")
	writeTestFile(t, claudeHub, "hub")
	writeTestFile(t, opencodeHub, "hub")
	writeTestFile(t, codexHub, "hub")

	tests := []struct {
		platform Platform
		want     string
	}{
		{PlatformClaude, claudeHub},
		{PlatformOpenCode, opencodeHub},
		{PlatformCodex, codexHub},
	}

	for _, tt := range tests {
		got := AgentDefinitionPath(root, tt.platform, "aether-builder")
		if got != tt.want {
			t.Fatalf("AgentDefinitionPath hub fallback %s = %q, want %q", tt.platform, got, tt.want)
		}
	}
}

func TestProviderAvailabilityCategories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	const secretOutput = "sk-proj-provider-secret-123"

	tests := []struct {
		name             string
		provider         Platform
		makeDispatcher   func(t *testing.T, dir string) PlatformDispatcher
		wantAvailable    bool
		wantCategory     string
		wantReasonPart   string
		forbiddenReasons []string
	}{
		{
			name:     "codex missing binary",
			provider: PlatformCodex,
			makeDispatcher: func(t *testing.T, dir string) PlatformDispatcher {
				return &RealInvoker{binaryName: filepath.Join(dir, "missing-codex-binary")}
			},
			wantCategory:   "binary_missing",
			wantReasonPart: "not found",
		},
		{
			name:     "codex probe failure redacts output",
			provider: PlatformCodex,
			makeDispatcher: func(t *testing.T, dir string) PlatformDispatcher {
				binary := writeFakeProviderCLI(t, dir, "codex", "login status", "stdout "+secretOutput, "stderr ghp_codex_secret", 42)
				return &RealInvoker{binaryName: binary}
			},
			wantCategory:     "auth_probe_failed",
			wantReasonPart:   "login status failed",
			forbiddenReasons: []string{secretOutput, "ghp_codex_secret", "stdout", "stderr"},
		},
		{
			name:     "codex inactive session",
			provider: PlatformCodex,
			makeDispatcher: func(t *testing.T, dir string) PlatformDispatcher {
				binary := writeFakeProviderCLI(t, dir, "codex", "login status", "Not logged in", "", 0)
				return &RealInvoker{binaryName: binary}
			},
			wantCategory:   "auth_inactive",
			wantReasonPart: "authenticated session",
		},
		{
			name:     "codex inactive session with intervening wording",
			provider: PlatformCodex,
			makeDispatcher: func(t *testing.T, dir string) PlatformDispatcher {
				binary := writeFakeProviderCLI(t, dir, "codex", "login status", "Not currently logged in", "", 0)
				return &RealInvoker{binaryName: binary}
			},
			wantCategory:   "auth_inactive",
			wantReasonPart: "authenticated session",
		},
		{
			name:     "codex authenticated session",
			provider: PlatformCodex,
			makeDispatcher: func(t *testing.T, dir string) PlatformDispatcher {
				binary := writeFakeProviderCLI(t, dir, "codex", "login status", "Logged in as test@example.com", "", 0)
				return &RealInvoker{binaryName: binary}
			},
			wantAvailable:  true,
			wantCategory:   "available",
			wantReasonPart: "",
		},
		{
			name:     "claude inactive session",
			provider: PlatformClaude,
			makeDispatcher: func(t *testing.T, dir string) PlatformDispatcher {
				binary := writeFakeProviderCLI(t, dir, "claude", "auth status --json", `{"loggedIn":false}`, "", 0)
				return &ClaudeDispatcher{binaryName: binary}
			},
			wantCategory:   "auth_inactive",
			wantReasonPart: "no active login",
		},
		{
			name:     "claude invalid provider output",
			provider: PlatformClaude,
			makeDispatcher: func(t *testing.T, dir string) PlatformDispatcher {
				binary := writeFakeProviderCLI(t, dir, "claude", "auth status --json", "not-json", "", 0)
				return &ClaudeDispatcher{binaryName: binary}
			},
			wantCategory:   "invalid_auth_output",
			wantReasonPart: "invalid JSON",
		},
		{
			name:     "claude authenticated session",
			provider: PlatformClaude,
			makeDispatcher: func(t *testing.T, dir string) PlatformDispatcher {
				binary := writeFakeProviderCLI(t, dir, "claude", "auth status --json", `{"loggedIn":true}`, "", 0)
				return &ClaudeDispatcher{binaryName: binary}
			},
			wantAvailable: true,
			wantCategory:  "available",
		},
		{
			name:     "opencode missing credentials",
			provider: PlatformOpenCode,
			makeDispatcher: func(t *testing.T, dir string) PlatformDispatcher {
				binary := writeFakeProviderCLI(t, dir, "opencode", "auth list", "No credentials configured", "", 0)
				return &OpenCodeDispatcher{binaryName: binary}
			},
			wantCategory:   "credentials_missing",
			wantReasonPart: "no configured credentials",
		},
		{
			name:     "opencode authenticated credentials",
			provider: PlatformOpenCode,
			makeDispatcher: func(t *testing.T, dir string) PlatformDispatcher {
				binary := writeFakeProviderCLI(t, dir, "opencode", "auth list", "\u25cf default\n", "", 0)
				return &OpenCodeDispatcher{binaryName: binary}
			},
			wantAvailable: true,
			wantCategory:  "available",
		},
		{
			name:     "opencode probe failure redacts output",
			provider: PlatformOpenCode,
			makeDispatcher: func(t *testing.T, dir string) PlatformDispatcher {
				binary := writeFakeProviderCLI(t, dir, "opencode", "auth list", "token "+secretOutput, "stderr opencode-secret", 42)
				return &OpenCodeDispatcher{binaryName: binary}
			},
			wantCategory:     "auth_probe_failed",
			wantReasonPart:   "auth list failed",
			forbiddenReasons: []string{secretOutput, "opencode-secret", "token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			dispatcher := tt.makeDispatcher(t, dir)

			status := dispatcher.Availability(context.Background())
			if status.Platform != tt.provider {
				t.Fatalf("Platform = %s, want %s", status.Platform, tt.provider)
			}
			if status.Available != tt.wantAvailable {
				t.Fatalf("Available = %v, want %v; reason=%q", status.Available, tt.wantAvailable, status.Reason)
			}
			if got := string(status.Category); got != tt.wantCategory {
				t.Fatalf("Category = %q, want %q; reason=%q", got, tt.wantCategory, status.Reason)
			}
			if tt.wantReasonPart != "" && !strings.Contains(status.Reason, tt.wantReasonPart) {
				t.Fatalf("Reason = %q, want to contain %q", status.Reason, tt.wantReasonPart)
			}
			assertAvailabilityStatusRedacts(t, status, tt.forbiddenReasons...)
		})
	}
}

func TestProviderAvailabilityOverridePathReportsProbeSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	tests := []struct {
		name           string
		makeDispatcher func(path string) PlatformDispatcher
	}{
		{
			name: "codex override path",
			makeDispatcher: func(path string) PlatformDispatcher {
				return &RealInvoker{binaryName: path}
			},
		},
		{
			name: "claude override path",
			makeDispatcher: func(path string) PlatformDispatcher {
				return &ClaudeDispatcher{binaryName: path}
			},
		},
		{
			name: "opencode override path",
			makeDispatcher: func(path string) PlatformDispatcher {
				return &OpenCodeDispatcher{binaryName: path}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			wrapperPath := writeFakeProviderCLI(t, dir, "provider-wrapper", "should not be probed", "", "", 99)

			status := tt.makeDispatcher(wrapperPath).Availability(context.Background())
			if !status.Available {
				t.Fatalf("Available = false, want true for override path; reason=%q", status.Reason)
			}
			if got := string(status.Category); got != "probe_skipped" {
				t.Fatalf("Category = %q, want probe_skipped; reason=%q", got, status.Reason)
			}
			if !strings.Contains(status.Reason, "probe skipped") {
				t.Fatalf("Reason = %q, want probe skipped explanation", status.Reason)
			}
		})
	}
}

func TestSelectPlatformInvokerReportsOrderedEvaluatedFallbackCandidates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	dir := t.TempDir()
	claudeWrapper := writeFakeProviderCLI(t, dir, "provider-wrapper", "should not be probed", "", "", 99)
	opencodeCalled := filepath.Join(dir, "opencode-called")
	opencode := writeInvokedMarkerCLI(t, dir, "opencode", opencodeCalled)

	t.Setenv(envActivePlatform, string(PlatformCodex))
	t.Setenv("AETHER_CODEX_PATH", filepath.Join(dir, "missing-codex-binary"))
	t.Setenv(envClaudePath, claudeWrapper)
	t.Setenv(envOpenCodePath, opencode)

	invoker := SelectPlatformInvoker(context.Background())
	if got := PlatformFromInvoker(invoker); got != PlatformClaude {
		t.Fatalf("selected platform = %s, want %s", got, PlatformClaude)
	}

	meta, ok := invoker.(selectionMetadata)
	if !ok {
		t.Fatalf("selected invoker does not expose candidate metadata: %T", invoker)
	}
	statuses := meta.CandidateStatuses()
	if len(statuses) != 2 {
		t.Fatalf("candidate count = %d, want evaluated candidates only [codex, claude]: %+v", len(statuses), statuses)
	}
	assertCandidateStatus(t, statuses[0], PlatformCodex, false, "binary_missing")
	assertCandidateStatus(t, statuses[1], PlatformClaude, true, "probe_skipped")
	if _, err := os.Stat(opencodeCalled); !os.IsNotExist(err) {
		t.Fatalf("opencode candidate was probed despite evaluated-candidates semantics")
	}
	description := DescribeInvokerAvailability(invoker, context.Background())
	for _, want := range []string{"detected host codex", "falling back to claude worker dispatcher"} {
		if !strings.Contains(description, want) {
			t.Fatalf("DescribeInvokerAvailability() = %q, want to contain %q", description, want)
		}
	}
	if strings.Contains(description, "opencode") {
		t.Fatalf("DescribeInvokerAvailability() = %q, want evaluated fallback only", description)
	}
}

func TestSelectPlatformInvokerRejectsUnsupportedWorkerPlatformOverride(t *testing.T) {
	t.Setenv(envActivePlatform, string(PlatformCodex))
	t.Setenv(envWorkerPlatform, "banana")
	t.Setenv("AETHER_CODEX_PATH", "go")

	invoker := SelectPlatformInvoker(context.Background())
	if got := PlatformFromInvoker(invoker); got != PlatformUnknown {
		t.Fatalf("selected platform = %s, want unknown for unsupported override", got)
	}

	meta, ok := invoker.(selectionMetadata)
	if !ok {
		t.Fatalf("unsupported override invoker does not expose candidate metadata: %T", invoker)
	}
	statuses := meta.CandidateStatuses()
	if len(statuses) != 1 {
		t.Fatalf("candidate count = %d, want one unsupported-provider status: %+v", len(statuses), statuses)
	}
	assertCandidateStatus(t, statuses[0], PlatformUnknown, false, "unsupported_provider")

	status := invoker.(interface {
		Availability(context.Context) AvailabilityStatus
	}).Availability(context.Background())
	if got := string(status.Category); got != "unsupported_provider" {
		t.Fatalf("unavailable category = %q, want unsupported_provider; reason=%q", got, status.Reason)
	}

	description := DescribeInvokerAvailability(invoker, context.Background())
	for _, want := range []string{"unsupported AETHER_WORKER_PLATFORM", "banana", "codex, claude, or opencode"} {
		if !strings.Contains(description, want) {
			t.Fatalf("DescribeInvokerAvailability() = %q, want to contain %q", description, want)
		}
	}
	if strings.Contains(description, "falling back") {
		t.Fatalf("DescribeInvokerAvailability() = %q, unsupported override must not fall back", description)
	}
}

func TestSelectPlatformInvokerUnavailableStatusRedactsAndCategorizes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	dir := t.TempDir()
	token := "sk-proj-unavailable-secret-123"
	codexPath := writeFakeProviderCLI(t, dir, "codex", "login status", "stdout "+token, "stderr ghp_unavailable_secret", 42)

	t.Setenv(envActivePlatform, string(PlatformCodex))
	t.Setenv("AETHER_CODEX_PATH", codexPath)
	t.Setenv(envClaudePath, filepath.Join(dir, "missing-claude"))
	t.Setenv(envOpenCodePath, filepath.Join(dir, "missing-opencode"))

	invoker := SelectPlatformInvoker(context.Background())
	if got := PlatformFromInvoker(invoker); got != PlatformUnknown {
		t.Fatalf("selected platform = %s, want unknown for all unavailable", got)
	}

	meta, ok := invoker.(selectionMetadata)
	if !ok {
		t.Fatalf("unavailable invoker does not expose candidate metadata: %T", invoker)
	}
	statuses := meta.CandidateStatuses()
	if len(statuses) != 3 {
		t.Fatalf("candidate count = %d, want 3 unavailable candidates: %+v", len(statuses), statuses)
	}
	assertCandidateStatus(t, statuses[0], PlatformCodex, false, "auth_probe_failed")
	assertCandidateStatus(t, statuses[1], PlatformClaude, false, "binary_missing")
	assertCandidateStatus(t, statuses[2], PlatformOpenCode, false, "binary_missing")

	status := invoker.(interface {
		Availability(context.Context) AvailabilityStatus
	}).Availability(context.Background())
	if got := string(status.Category); got != "auth_probe_failed" {
		t.Fatalf("unavailable category = %q, want auth_probe_failed; reason=%q", got, status.Reason)
	}
	if !strings.Contains(status.Reason, "detected host platform codex") {
		t.Fatalf("unavailable reason = %q, want active platform context", status.Reason)
	}
	assertAvailabilityStatusRedacts(t, status, token, "ghp_unavailable_secret", "stdout", "stderr")
}

func TestSelectPlatformInvokerNoCredentialsReportsUnavailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	dir := t.TempDir()
	codexPath := writeFakeProviderCLI(t, dir, "codex", "login status", "Not logged in", "", 0)
	claudePath := writeFakeProviderCLI(t, dir, "claude", "auth status --json", `{"loggedIn":false}`, "", 0)
	opencodePath := writeFakeProviderCLI(t, dir, "opencode", "auth list", "No credentials configured", "", 0)

	t.Setenv(envActivePlatform, string(PlatformCodex))
	t.Setenv("AETHER_CODEX_PATH", codexPath)
	t.Setenv(envClaudePath, claudePath)
	t.Setenv(envOpenCodePath, opencodePath)

	invoker := SelectPlatformInvoker(context.Background())
	if got := PlatformFromInvoker(invoker); got != PlatformUnknown {
		t.Fatalf("selected platform = %s, want unknown for all no-credential providers", got)
	}

	meta, ok := invoker.(selectionMetadata)
	if !ok {
		t.Fatalf("unavailable invoker does not expose candidate metadata: %T", invoker)
	}
	statuses := meta.CandidateStatuses()
	if len(statuses) != 3 {
		t.Fatalf("candidate count = %d, want 3 no-credential candidates: %+v", len(statuses), statuses)
	}
	assertCandidateStatus(t, statuses[0], PlatformCodex, false, "auth_inactive")
	assertCandidateStatus(t, statuses[1], PlatformClaude, false, "auth_inactive")
	assertCandidateStatus(t, statuses[2], PlatformOpenCode, false, "credentials_missing")

	status := invoker.(interface {
		Availability(context.Context) AvailabilityStatus
	}).Availability(context.Background())
	if status.Available {
		t.Fatalf("unavailable invoker reported available under no-credential conditions")
	}
	if got := string(status.Category); got != "auth_inactive" {
		t.Fatalf("unavailable category = %q, want auth_inactive; reason=%q", got, status.Reason)
	}
	for _, want := range []string{
		"detected host platform codex",
		"codex login status did not confirm an authenticated session",
		"claude auth status reported no active login",
		"opencode auth list reported no configured credentials",
	} {
		if !strings.Contains(status.Reason, want) {
			t.Fatalf("unavailable reason missing %q:\n%s", want, status.Reason)
		}
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeFakeProviderCLI(t *testing.T, dir, name, expectedArgs, stdout, stderr string, exitCode int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	script := `#!/bin/sh
if [ "$*" != "` + expectedArgs + `" ]; then
  echo "unexpected args: $*" >&2
  exit 99
fi
cat <<'AETHER_STDOUT'
` + stdout + `
AETHER_STDOUT
cat >&2 <<'AETHER_STDERR'
` + stderr + `
AETHER_STDERR
exit ` + strconv.Itoa(exitCode) + `
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write fake provider CLI %s: %v", path, err)
	}
	return path
}

func writeInvokedMarkerCLI(t *testing.T, dir, name, markerPath string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	script := `#!/bin/sh
printf invoked > "` + markerPath + `"
exit 77
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write marker CLI %s: %v", path, err)
	}
	return path
}

func assertCandidateStatus(t *testing.T, status AvailabilityStatus, wantPlatform Platform, wantAvailable bool, wantCategory string) {
	t.Helper()
	if status.Platform != wantPlatform {
		t.Fatalf("candidate platform = %s, want %s", status.Platform, wantPlatform)
	}
	if status.Available != wantAvailable {
		t.Fatalf("%s available = %v, want %v", status.Platform, status.Available, wantAvailable)
	}
	if got := string(status.Category); got != wantCategory {
		t.Fatalf("%s category = %q, want %q; reason=%q", status.Platform, got, wantCategory, status.Reason)
	}
}

func assertAvailabilityStatusRedacts(t *testing.T, status AvailabilityStatus, forbidden ...string) {
	t.Helper()
	if len(forbidden) == 0 {
		return
	}
	encoded, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	haystacks := map[string]string{
		"reason": status.Reason,
		"json":   string(encoded),
	}
	for label, haystack := range haystacks {
		for _, needle := range forbidden {
			if needle != "" && strings.Contains(haystack, needle) {
				t.Fatalf("%s leaked forbidden provider output %q in %q", label, needle, haystack)
			}
		}
	}
}

// --- AETHER_OPENCODE_AGENT_URL env var injection tests ---

func TestInvokeHostedWorkerEnvVarOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	t.Setenv(envOpenCodeAgentURL, "http://localhost:9876/agent")

	dir := t.TempDir()
	agentPath := createTestMarkdownAgent(t, dir, "aether-builder", "Builder")

	envCapturePath := filepath.Join(dir, "captured-env.txt")
	scriptPath := filepath.Join(dir, "fake-opencode.sh")
	script := `#!/bin/sh
	env | grep -i AETHER > "$ENV_CAPTURE_PATH"
	cat <<'EOF'
{"type":"message.part.updated","part":{"type":"text","text":"{\"ant_name\":\"Forge-1\",\"caste\":\"builder\",\"task_id\":\"1.1\",\"status\":\"completed\",\"summary\":\"done\",\"files_created\":[],\"files_modified\":[],\"tests_written\":[],\"tool_count\":0,\"blockers\":[],\"spawns\":[]}"}}
EOF
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake opencode script: %v", err)
	}

	invoker := &OpenCodeDispatcher{binaryName: scriptPath}
	t.Setenv("ENV_CAPTURE_PATH", envCapturePath)
	t.Setenv(envOpenCodePrimary, "")

	_, err := invoker.Invoke(t.Context(), WorkerConfig{
		AgentName:      "aether-builder",
		AgentTOMLPath:  agentPath,
		Caste:          "builder",
		WorkerName:     "Forge-1",
		TaskID:         "1.1",
		TaskBrief:      "Build the feature.",
		ContextCapsule: "Goal: test",
		Root:           dir,
	})
	if err != nil {
		t.Fatalf("OpenCode Invoke returned error: %v", err)
	}

	envData, err := os.ReadFile(envCapturePath)
	if err != nil {
		t.Fatalf("failed to read captured env: %v", err)
	}
	envText := string(envData)

	// Verify the subprocess received AETHER_OPENCODE_AGENT_URL
	if !strings.Contains(envText, "AETHER_OPENCODE_AGENT_URL=http://localhost:9876/agent") {
		t.Fatalf("expected subprocess to receive AETHER_OPENCODE_AGENT_URL in env, got:\n%s", envText)
	}
}

func TestInvokeHostedWorkerNoEnvVarOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	// Ensure the env var is NOT set
	t.Setenv(envOpenCodeAgentURL, "")

	dir := t.TempDir()
	agentPath := createTestMarkdownAgent(t, dir, "aether-builder", "Builder")

	envCapturePath := filepath.Join(dir, "captured-env.txt")
	scriptPath := filepath.Join(dir, "fake-opencode.sh")
	script := `#!/bin/sh
	env | grep -i AETHER > "$ENV_CAPTURE_PATH"
	cat <<'EOF'
{"type":"message.part.updated","part":{"type":"text","text":"{\"ant_name\":\"Forge-2\",\"caste\":\"builder\",\"task_id\":\"2.1\",\"status\":\"completed\",\"summary\":\"done\",\"files_created\":[],\"files_modified\":[],\"tests_written\":[],\"tool_count\":0,\"blockers\":[],\"spawns\":[]}"}}
EOF
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake opencode script: %v", err)
	}

	invoker := &OpenCodeDispatcher{binaryName: scriptPath}
	t.Setenv("ENV_CAPTURE_PATH", envCapturePath)
	t.Setenv(envOpenCodePrimary, "")

	_, err := invoker.Invoke(t.Context(), WorkerConfig{
		AgentName:      "aether-builder",
		AgentTOMLPath:  agentPath,
		Caste:          "builder",
		WorkerName:     "Forge-2",
		TaskID:         "2.1",
		TaskBrief:      "Build the feature.",
		ContextCapsule: "Goal: test",
		Root:           dir,
	})
	if err != nil {
		t.Fatalf("OpenCode Invoke returned error: %v", err)
	}

	envData, err := os.ReadFile(envCapturePath)
	if err != nil {
		t.Fatalf("failed to read captured env: %v", err)
	}
	envText := string(envData)

	// Verify AETHER_OPENCODE_AGENT_URL is empty (not overridden) in the subprocess env.
	// When t.Setenv sets it to "", the var name still appears in env but with no value.
	// The important check: it should NOT have a non-empty URL value.
	for _, line := range strings.Split(envText, "\n") {
		if strings.HasPrefix(line, "AETHER_OPENCODE_AGENT_URL=") {
			val := strings.TrimPrefix(line, "AETHER_OPENCODE_AGENT_URL=")
			if val != "" {
				t.Fatalf("expected AETHER_OPENCODE_AGENT_URL to be empty in subprocess, got value: %q", val)
			}
		}
	}
}

func TestIsAgentDelegateSession(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    bool
	}{
		{
			name:    "no env vars set",
			envVars: map[string]string{},
			want:    false,
		},
		{
			name: "CLAUDE_CODE_SIMPLE=1",
			envVars: map[string]string{
				"CLAUDE_CODE_SIMPLE": "1",
			},
			want: true,
		},
		{
			name: "OPENCODE_AGENT=1",
			envVars: map[string]string{
				"OPENCODE_AGENT": "1",
			},
			want: true,
		},
		{
			name: "AETHER_AGENT_DELEGATE=1",
			envVars: map[string]string{
				"AETHER_AGENT_DELEGATE": "1",
			},
			want: true,
		},
		{
			name: "all three set",
			envVars: map[string]string{
				"CLAUDE_CODE_SIMPLE":    "1",
				"OPENCODE_AGENT":        "1",
				"AETHER_AGENT_DELEGATE": "1",
			},
			want: true,
		},
		{
			name: "CLAUDE_CODE_SIMPLE=0 (disabled)",
			envVars: map[string]string{
				"CLAUDE_CODE_SIMPLE": "0",
			},
			want: false,
		},
		{
			name: "OPENCODE_AGENT=0 (disabled)",
			envVars: map[string]string{
				"OPENCODE_AGENT": "0",
			},
			want: false,
		},
		{
			name: "AETHER_AGENT_DELEGATE=0 (disabled)",
			envVars: map[string]string{
				"AETHER_AGENT_DELEGATE": "0",
			},
			want: false,
		},
		{
			name: "CLAUDE_CODE_SIMPLE= (empty)",
			envVars: map[string]string{
				"CLAUDE_CODE_SIMPLE": "",
			},
			want: false,
		},
		{
			name: "OPENCODE_AGENT and AETHER_AGENT_DELEGATE set",
			envVars: map[string]string{
				"OPENCODE_AGENT":        "1",
				"AETHER_AGENT_DELEGATE": "1",
			},
			want: true,
		},
		{
			name: "unrelated env vars set",
			envVars: map[string]string{
				"PATH": "/usr/bin",
				"HOME": "/tmp",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars first
			for _, key := range []string{"CLAUDE_CODE_SIMPLE", "OPENCODE_AGENT", "AETHER_AGENT_DELEGATE"} {
				t.Setenv(key, "")
			}
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			got := IsAgentDelegateSession()
			if got != tt.want {
				t.Fatalf("IsAgentDelegateSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldUseAgentDelegatePath(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		platform string
		want     bool
	}{
		{
			name:     "claude agent session",
			envVars:  map[string]string{"CLAUDE_CODE_SIMPLE": "1"},
			platform: "claude",
			want:     true,
		},
		{
			name:     "opencode agent session",
			envVars:  map[string]string{"OPENCODE_AGENT": "1"},
			platform: "opencode",
			want:     true,
		},
		{
			name:     "codex session stays local",
			envVars:  map[string]string{"AETHER_AGENT_DELEGATE": "1"},
			platform: "codex",
			want:     false,
		},
		{
			name:     "claude platform without delegate marker",
			envVars:  map[string]string{},
			platform: "claude",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{"CLAUDE_CODE_SIMPLE", "OPENCODE_AGENT", "AETHER_AGENT_DELEGATE"} {
				t.Setenv(key, "")
			}
			t.Setenv(envActivePlatform, tt.platform)
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}
			if got := ShouldUseAgentDelegatePath(); got != tt.want {
				t.Fatalf("ShouldUseAgentDelegatePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClassifyHostedExecutionErrorExplainsOpenCodeLocalServerFailure(t *testing.T) {
	err := classifyHostedExecutionError("opencode", os.ErrNotExist, "POST http://localhost:4000/messages returned 404", false)
	text := err.Error()
	for _, want := range []string{
		"opencode worker dispatcher unavailable",
		"local OpenCode server",
		"AETHER_WORKER_PLATFORM=claude/codex",
		"localhost:4000/messages",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("classified error missing %q:\n%s", want, text)
		}
	}
}

func TestClassifyHostedExecutionErrorRedactsProviderOutput(t *testing.T) {
	err := classifyHostedExecutionError("claude", os.ErrPermission, "auth failed token sk-proj-secret-123 stderr ghp_worker_secret", true)
	text := err.Error()
	for _, forbidden := range []string{"sk-proj-secret-123", "ghp_worker_secret"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("hosted execution error leaked %q:\n%s", forbidden, text)
		}
	}
	if !strings.Contains(text, "[redacted]") {
		t.Fatalf("hosted execution error did not show redaction marker:\n%s", text)
	}
}

func TestWriteHostedWorkerOutputDebugRedactsProviderOutput(t *testing.T) {
	root := t.TempDir()
	rel := writeHostedWorkerOutputDebug(
		root,
		"opencode",
		WorkerConfig{WorkerName: "Forge-2", Caste: "builder", TaskID: "2.2", AgentName: "aether-builder"},
		[]string{"run", "--agent", "build", "prompt with sk-proj-prompt-secret"},
		"stdout sk-proj-stdout-secret token=stdout-secret",
		"stderr ghp_stderr_secret secret=stderr-secret",
		fmt.Errorf("parse failed sk-proj-error-secret"),
	)
	if rel == "" {
		t.Fatal("expected debug artifact path")
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read debug artifact: %v", err)
	}
	text := string(data)
	for _, forbidden := range []string{
		"sk-proj-prompt-secret",
		"sk-proj-stdout-secret",
		"stdout-secret",
		"ghp_stderr_secret",
		"stderr-secret",
		"sk-proj-error-secret",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("debug artifact leaked %q:\n%s", forbidden, text)
		}
	}
	for _, want := range []string{`"stdout_bytes"`, `"stderr_bytes"`, "[redacted]"} {
		if !strings.Contains(text, want) {
			t.Fatalf("debug artifact missing %q:\n%s", want, text)
		}
	}
}

// createTestMarkdownAgent creates a minimal markdown agent file for testing.
func createTestMarkdownAgent(t *testing.T, dir, name, description string) string {
	t.Helper()
	agentPath := filepath.Join(dir, name+".md")
	content := "---\nname: " + name + "\ndescription: " + description + "\nmode: subagent\n---\nYou are the " + description + ".\n"
	if err := os.WriteFile(agentPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write agent markdown: %v", err)
	}
	return agentPath
}
