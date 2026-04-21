package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildWrapperCeremonyContract(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "build.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "build.md"),
	}

	required := []string{
		"AETHER_OUTPUT_MODE=visual aether status",
		"## Active Signals",
		"REDIRECT",
		"FOCUS",
		"FEEDBACK",
		"strength or remaining-life context",
		"why each signal matters right now",
		"## Phase Framing",
		"Phase N of M — Name",
		"## Spawn Ritual",
		"do not guess the worker mix ahead of the runtime",
		"runtime reveal the actual castes, names, waves, and outcomes",
		"Dispatching workers now...",
		"AETHER_OUTPUT_MODE=visual aether build $ARGUMENTS",
		"## After the Build",
		"/ant:continue",
	}

	inOrder := []string{
		"## Colony Context",
		"## Active Signals",
		"## Phase Framing",
		"## Spawn Ritual",
		"Dispatching workers now...",
		"AETHER_OUTPUT_MODE=visual aether build $ARGUMENTS",
		"## After the Build",
	}

	for _, wrapperPath := range wrapperPaths {
		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("read %s: %v", wrapperPath, err)
		}

		text := string(content)
		for _, want := range required {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing %q", wrapperPath, want)
			}
		}

		assertSubstringsInOrder(t, wrapperPath, text, inOrder)
	}
}

func assertSubstringsInOrder(t *testing.T, path, content string, ordered []string) {
	t.Helper()

	cursor := 0
	for _, needle := range ordered {
		idx := strings.Index(content[cursor:], needle)
		if idx < 0 {
			t.Fatalf("%s missing ordered marker %q", path, needle)
		}
		cursor += idx + len(needle)
	}
}
