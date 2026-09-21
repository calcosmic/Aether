package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexLiteralCommandGuidanceStaysMinimal(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	testCases := []struct {
		path     string
		required []string
	}{
		{
			path: ".aether/skills/colony/colony-interaction/SKILL.md",
			required: []string{
				"Do not announce skill usage, intent interpretation, or a preflight summary before running the command.",
				"After the command returns, show the CLI's screen unchanged, then at most two short sentences of your own.",
			},
		},
		{
			path: ".aether/skills/colony/colony-lifecycle/SKILL.md",
			required: []string{
				"Do not prepend exploratory narration like \"I'm checking the repo\" or \"I'm treating this as...\"",
				"If the `aether` CLI already rendered the result, do not restate the same guidance in a second synthetic \"Next Up\" block.",
			},
		},
		{
			path: ".aether/skills/colony/colony-visuals/SKILL.md",
			required: []string{
				"Let the CLI's own visual output stand on its own.",
				"Do not wrap the command with extra decorative commentary before and after execution.",
			},
		},
		{
			path: ".aether/templates/codex-md-template.md",
			required: []string{
				"Do not preface\nliteral passthrough execution with repo archaeology or skill narration",
				"output is primary — show the CLI's screen unchanged, then at most two short sentences of extra explanation.",
			},
		},
		{
			path: ".aether/templates/agents-md-template.md",
			required: []string{
				"Do not preface\nliteral passthrough execution with repo archaeology or skill narration",
				"output is primary — show the CLI's screen unchanged, then at most two short sentences of extra explanation.",
			},
		},
	}

	for _, tc := range testCases {
		content, err := os.ReadFile(filepath.Join(repoRoot, tc.path))
		if err != nil {
			t.Fatalf("failed to read %s: %v", tc.path, err)
		}
		text := string(content)
		for _, want := range tc.required {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing %q", tc.path, want)
			}
		}
	}
}
