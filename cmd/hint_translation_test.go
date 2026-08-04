package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNextUpTranslatesWrapperCommandsPerPlatform is the WP-2 gate: in Claude
// Code and OpenCode the user types /ant-continue, so a hint that says
// "Run `aether continue`" names a command they cannot run. Commands with no
// slash wrapper must survive verbatim on every platform.
func TestNextUpTranslatesWrapperCommandsPerPlatform(t *testing.T) {
	cases := []struct {
		name     string
		platform string
		in       string
		want     string
		notWant  string
	}{
		{"claude rewrites continue", "claude", "Run `aether continue` again.", "/ant-continue", "aether continue"},
		{"claude keeps trailing args", "claude", "Run `aether build 3` to start the next phase", "/ant-build 3", "aether build"},
		{"opencode rewrites too", "opencode", "Colony complete. Run `aether seal` to finalize.", "/ant-seal", "aether seal"},
		{"codex left alone", "codex", "Run `aether continue` again.", "aether continue", "/ant-continue"},
		{"publish has no wrapper", "claude", "Recover with: aether publish", "aether publish", "/ant-publish"},
		{"host has no wrapper", "claude", "Run `aether host plan --dry-run`", "aether host plan", "/ant-host"},
		{"flag-resolve has no wrapper", "claude", "  aether flag-resolve --id flag_123", "aether flag-resolve --id flag_123", "/ant-flag"},
		{"env-prefixed literal untouched", "claude", "AETHER_OUTPUT_MODE=json aether build 3 --plan-only", "AETHER_OUTPUT_MODE=json aether build 3", "/ant-build"},
		{"prose word untouched", "claude", "the aether runtime owns state", "aether runtime", "/ant-runtime"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := translateHintCommandsForPlatform(tc.in, tc.platform)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("translate(%q, %s) = %q, want it to contain %q", tc.in, tc.platform, got, tc.want)
			}
			if tc.notWant != "" && strings.Contains(got, tc.notWant) {
				t.Fatalf("translate(%q, %s) = %q, must not contain %q", tc.in, tc.platform, got, tc.notWant)
			}
		})
	}
}

// The funnel itself must apply the translation — otherwise all 78 call sites
// would each need their own fix.
func TestRenderNextUpAppliesTranslationAtTheFunnel(t *testing.T) {
	t.Setenv("AETHER_PLATFORM", "claude")
	out := renderNextUp("Run `aether continue` to verify.", "Run `aether status` to inspect.")
	if !strings.Contains(out, "/ant-continue") || !strings.Contains(out, "/ant-status") {
		t.Fatalf("renderNextUp did not translate primary and alternative:\n%s", out)
	}
	if strings.Contains(out, "`aether continue`") {
		t.Fatalf("raw CLI form survived the funnel:\n%s", out)
	}

	t.Setenv("AETHER_PLATFORM", "codex")
	out = renderNextUp("Run `aether continue` to verify.")
	if !strings.Contains(out, "aether continue") || strings.Contains(out, "/ant-continue") {
		t.Fatalf("codex hints must stay raw CLI:\n%s", out)
	}
}

// TestWrapperCommandNamesMatchCanonicalCorpus is the anti-drift guard: adding
// or removing a slash wrapper without updating the allowlist fails here rather
// than silently showing users the wrong command form.
func TestWrapperCommandNamesMatchCanonicalCorpus(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	entries, err := filepath.Glob(filepath.Join(repoRoot, ".claude", "commands", "ant", "*.md"))
	if err != nil {
		t.Fatalf("glob canonical commands: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no canonical wrapper commands found")
	}

	canonical := map[string]bool{}
	for _, entry := range entries {
		canonical[strings.TrimSuffix(filepath.Base(entry), ".md")] = true
	}
	for name := range canonical {
		if !wrapperCommandNames[name] {
			t.Errorf("wrapper /ant-%s exists on disk but is missing from wrapperCommandNames", name)
		}
	}
	for name := range wrapperCommandNames {
		if !canonical[name] {
			t.Errorf("wrapperCommandNames lists %q but .claude/commands/ant/%s.md does not exist", name, name)
		}
	}
}

// The translation only works if every "Next Up" block goes through the funnel.
func TestNextUpIsTheOnlyNextUpFunnel(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(repoRoot, "cmd", "*.go"))
	if err != nil {
		t.Fatalf("glob cmd sources: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") || filepath.Base(file) == "codex_visuals.go" {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(data), `renderBanner(commandEmoji("next-up")`) {
			t.Errorf("%s builds a Next Up banner outside renderNextUp; hints there escape platform translation", filepath.Base(file))
		}
	}
}
