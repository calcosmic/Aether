package cmd

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

func TestDispatchAgentPathDoesNotDefaultUnknownPlatformToCodex(t *testing.T) {
	root := t.TempDir()
	if got := dispatchAgentPathForPlatform(root, codex.PlatformUnknown, "aether-builder"); got != "" {
		t.Fatalf("unknown platform path = %q, want empty", got)
	}
}

func TestDispatchAgentPathUsesExplicitCodexPlatform(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	want := filepath.Join(home, ".codex", "agents", "aether-builder.toml")
	if got := dispatchAgentPathForPlatform(root, codex.PlatformCodex, "aether-builder"); got != want {
		t.Fatalf("codex platform path = %q, want %q", got, want)
	}
}

func TestDispatchAvailabilityMessageDefaultsUnknown(t *testing.T) {
	got := dispatchAvailabilityMessage(dispatchAvailabilityUnknownStub{})
	want := "no authenticated worker platform is available"
	if got != want {
		t.Fatalf("dispatchAvailabilityMessage() = %q, want %q", got, want)
	}
}

func TestDispatchAvailabilityMessageRedactsProbeOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub uses POSIX sh")
	}

	dir := t.TempDir()
	token := "sk-proj-cmd-secret-123"
	fakeCodex := writeDispatchFakeProviderCLI(t, dir, "codex", "login status", "stdout "+token, "stderr ghp_cmd_secret", 42)

	t.Setenv("AETHER_ACTIVE_PLATFORM", string(codex.PlatformCodex))
	t.Setenv("AETHER_CODEX_PATH", fakeCodex)
	t.Setenv("AETHER_CLAUDE_PATH", filepath.Join(dir, "missing-claude"))
	t.Setenv("AETHER_OPENCODE_PATH", filepath.Join(dir, "missing-opencode"))

	invoker := codex.SelectPlatformInvoker(context.Background())
	message := dispatchAvailabilityMessage(invoker)
	for _, forbidden := range []string{token, "ghp_cmd_secret", "stdout", "stderr"} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("dispatchAvailabilityMessage leaked provider output %q in %q", forbidden, message)
		}
	}
	for _, want := range []string{"codex", "login status failed"} {
		if !strings.Contains(message, want) {
			t.Fatalf("dispatchAvailabilityMessage() = %q, want to contain %q", message, want)
		}
	}
	if !strings.Contains(message, "Next:") || !strings.Contains(message, "fix the provider login") {
		t.Fatalf("dispatchAvailabilityMessage() = %q, want provider next action", message)
	}

	err := dispatchUnavailableError(invoker)
	if err == nil {
		t.Fatal("dispatchUnavailableError() = nil, want error")
	}
	for _, forbidden := range []string{token, "ghp_cmd_secret"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("dispatchUnavailableError leaked provider output %q in %q", forbidden, err.Error())
		}
	}
}

func TestDispatchAvailabilityMessageExplainsProviderCauseAndNextAction(t *testing.T) {
	invoker := dispatchAvailabilityCandidateStub{
		active: codex.PlatformCodex,
		statuses: []codex.AvailabilityStatus{
			{
				Platform:  codex.PlatformCodex,
				Binary:    "codex",
				Available: false,
				Category:  codex.AvailabilityCategoryAuthInactive,
				Reason:    "codex login status did not confirm an authenticated session",
			},
			{
				Platform:  codex.PlatformClaude,
				Binary:    "claude",
				Available: false,
				Category:  codex.AvailabilityCategoryBinaryMissing,
				Reason:    "claude binary \"claude\" not found in PATH",
			},
		},
	}

	message := dispatchAvailabilityMessage(invoker)
	for _, want := range []string{
		"detected host platform codex",
		"codex provider:",
		"authenticated session",
		"Next: sign in to codex",
		"claude provider:",
		"Next: install claude",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("dispatchAvailabilityMessage() = %q, want to contain %q", message, want)
		}
	}
}

type dispatchAvailabilityUnknownStub struct{}

func (dispatchAvailabilityUnknownStub) Invoke(context.Context, codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{}, nil
}

func (dispatchAvailabilityUnknownStub) IsAvailable(context.Context) bool {
	return false
}

func (dispatchAvailabilityUnknownStub) ValidateAgent(string) error {
	return nil
}

func (dispatchAvailabilityUnknownStub) Availability(context.Context) codex.AvailabilityStatus {
	return codex.AvailabilityStatus{}
}

type dispatchAvailabilityCandidateStub struct {
	active   codex.Platform
	statuses []codex.AvailabilityStatus
}

func (s dispatchAvailabilityCandidateStub) Invoke(context.Context, codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{}, nil
}

func (s dispatchAvailabilityCandidateStub) IsAvailable(context.Context) bool {
	return false
}

func (s dispatchAvailabilityCandidateStub) ValidateAgent(string) error {
	return nil
}

func (s dispatchAvailabilityCandidateStub) Availability(context.Context) codex.AvailabilityStatus {
	return codex.AvailabilityStatus{
		Platform:  codex.PlatformUnknown,
		Available: false,
		Category:  codex.AvailabilityCategoryAuthInactive,
		Reason:    "worker dispatchers unavailable",
	}
}

func (s dispatchAvailabilityCandidateStub) ActivePlatform() codex.Platform {
	return s.active
}

func (s dispatchAvailabilityCandidateStub) CandidateStatuses() []codex.AvailabilityStatus {
	return s.statuses
}

func writeDispatchFakeProviderCLI(t *testing.T, dir, name, expectedArgs, stdout, stderr string, exitCode int) string {
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
