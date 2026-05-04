package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceCheckValidatesCurrentSourceSurfaces(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	result := runSourceCheck(repoRoot)
	if !result.OK {
		t.Fatalf("source check should pass for current checkout, issues: %+v", result.Issues)
	}

	for _, want := range []string{
		"canonical source surfaces",
		"retired source mirrors",
		"generated command wrappers",
	} {
		if !sourceCheckHasComponent(result, want) {
			t.Fatalf("source check missing component %q", want)
		}
	}
}

func TestSourceCheckRejectsRetiredMirrorDirectories(t *testing.T) {
	root := minimalSourceCheckRoot(t)
	writeFile(t, root, filepath.Join(".aether", "agents-claude", "aether-builder.md"), []byte("# stale mirror\n"))

	result := runSourceCheck(root)
	if result.OK {
		t.Fatal("source check should fail when retired source mirrors are recreated")
	}

	if !sourceCheckHasIssue(result, ".aether/agents-claude", "retired packaging mirror exists") {
		t.Fatalf("missing retired mirror issue, got: %+v", result.Issues)
	}
}

func minimalSourceCheckRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{
		".aether/commands",
		".aether/skills",
		".aether/templates",
		".aether/docs",
		".aether/utils",
		".aether/exchange",
		".claude/agents/ant",
		".claude/commands/ant",
		".opencode/agents",
		".opencode/commands/ant",
		".codex/agents",
	} {
		writeFile(t, root, filepath.Join(dir, ".keep"), []byte(""))
	}

	writeFile(t, root, filepath.Join(".aether", "workers.md"), []byte("# Workers\n"))
	writeFile(t, root, filepath.Join(".aether", "commands", "status.yaml"), []byte("name: ant-status\n"))
	header := []byte("<!-- Generated from .aether/commands/status.yaml - DO NOT EDIT DIRECTLY -->\n")
	writeFile(t, root, filepath.Join(".claude", "commands", "ant", "status.md"), header)
	writeFile(t, root, filepath.Join(".opencode", "commands", "ant", "status.md"), header)

	return root
}

func sourceCheckHasComponent(result sourceCheckResult, name string) bool {
	for _, component := range result.Components {
		if component.Name == name {
			return true
		}
	}
	return false
}

func sourceCheckHasIssue(result sourceCheckResult, path, message string) bool {
	for _, issue := range result.Issues {
		if issue.Path == path && strings.Contains(issue.Message, message) {
			return true
		}
	}
	return false
}
