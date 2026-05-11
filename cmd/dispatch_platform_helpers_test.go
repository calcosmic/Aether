package cmd

import (
	"path/filepath"
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
