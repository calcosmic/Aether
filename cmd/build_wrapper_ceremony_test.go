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
		"Use the Go `aether` CLI as the source of truth.",
		"AETHER_OUTPUT_MODE=visual aether status",
		"## Active Signals",
		"REDIRECT",
		"FOCUS",
		"FEEDBACK",
		"strength or remaining-life context",
		"## Phase Framing",
		"Phase N of M -- Name",
		"## Dispatch Manifest",
		"aether host build --dry-run",
		"temporary manifest file outside `.aether/data/`",
		"result.manifest.dispatch_manifest",
		"## Runtime Spawn Ceremony",
		"AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow build --manifest-file <manifest_file>",
		"Do not set `run_in_background`",
		"Do NOT run `aether host build` without `--dry-run` from this wrapper",
		"Do NOT run `aether build --synthetic` after real",
		"AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file",
		"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file",
		"/ant-continue",
		"dispatch_manifest.context_capsule",
	}

	inOrder := []string{
		"## Ownership Split",
		"## Colony Context",
		"## Active Signals",
		"## Phase Framing",
		"## Dispatch Manifest",
		"aether host build --dry-run",
		"## Runtime Spawn Ceremony",
		"AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file",
		"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file",
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
		for _, forbidden := range []string{
			"Do NOT load playbooks",
			"\nAETHER_OUTPUT_MODE=visual aether build $ARGUMENTS\n",
			"\nAETHER_OUTPUT_MODE=json aether build $ARGUMENTS --plan-only\n",
			"Do NOT run `aether build` without `--plan-only` from this wrapper.",
			"Do NOT run direct `aether build` from this wrapper for manifest generation",
		} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s still contains old pass-through contract %q", wrapperPath, forbidden)
			}
		}
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
