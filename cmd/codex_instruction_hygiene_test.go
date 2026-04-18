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
			path: ".aether/skills-codex/colony/colony-interaction/SKILL.md",
			required: []string{
				"Do not announce skill usage, intent interpretation, or a preflight summary before running the command.",
				"After the command returns, keep your own wrapper to one short sentence at most.",
				"Do not fast-path colony-shaping commands like `aether init`, `aether plan`,",
				"For `aether init \"<goal>\"`, run a short foundation pass first:",
			},
		},
		{
			path: ".aether/skills-codex/colony/colony-lifecycle/SKILL.md",
			required: []string{
				"Do not prepend exploratory narration like \"I'm checking the repo\" or \"I'm treating this as...\"",
				"If the `aether` CLI already rendered the result, do not restate the same guidance in a second synthetic \"Next Up\" block.",
				"Mediated commands:",
				"run `aether init-research --goal \"<goal>\" --target .`, assemble a",
			},
		},
		{
			path: ".aether/skills-codex/colony/colony-visuals/SKILL.md",
			required: []string{
				"Let the CLI's own visual output stand on its own.",
				"Do not wrap the command with extra decorative commentary before and after execution.",
			},
		},
		{
			path: ".aether/templates/codex-md-template.md",
			required: []string{
				"do not announce that you are \"checking the repo\", \"interpreting the workflow\",",
				"your own wrapper should be at most one short sentence.",
				"mediated workflow: `aether init`, `aether plan`, `aether build`,",
				"if a user asks for `aether dream` or another command the binary does not expose",
			},
		},
		{
			path: ".aether/templates/agents-md-template.md",
			required: []string{
				"do not preface execution with \"I'm checking the repo\" or similar commentary",
				"keep any extra explanation to one short sentence unless the user asks for more",
				"mediated workflow: `aether init`, `aether plan`, `aether build`,",
				"if a user asks for `aether dream` or another command the binary does not expose",
			},
		},
		{
			path: ".codex/CODEX.md",
			required: []string{
				"Command split:",
				"mediated workflow: `aether init`, `aether plan`, `aether build`,",
				"another command the binary does not expose, say",
			},
		},
		{
			path: "AGENTS.md",
			required: []string{
				"Command split:",
				"mediated workflow: `aether init`, `aether plan`, `aether build`,",
				"another command the binary does not expose, say",
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
