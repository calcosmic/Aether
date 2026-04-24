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
		"## Dispatch Manifest",
		"Asking the runtime for the dispatch manifest...",
		"AETHER_OUTPUT_MODE=json aether build $ARGUMENTS --plan-only",
		"result.dispatch_manifest",
		"## Playbook Procedure",
		".aether/docs/command-playbooks/build-wave.md",
		"## Wave Execution",
		"dispatch_manifest.execution_plan",
		"execution_wave",
		"aether spawn-log",
		"subagent_type",
		"aether spawn-complete",
		"## Completion Packet",
		"AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file",
		"dispatch_mode: external-task",
		"## After the Build",
		"/ant-continue",
		"Do NOT run `aether build` without `--plan-only`",
		"Do NOT run `aether build --synthetic` after real",
	}

	inOrder := []string{
		"## Colony Context",
		"## Active Signals",
		"## Phase Framing",
		"## Dispatch Manifest",
		"AETHER_OUTPUT_MODE=json aether build $ARGUMENTS --plan-only",
		"## Playbook Procedure",
		"## Wave Execution",
		"dispatch_manifest.execution_plan",
		"## Completion Packet",
		"AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file",
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
			"AETHER_OUTPUT_MODE=visual aether build $ARGUMENTS",
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
