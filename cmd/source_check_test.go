package cmd

import (
	"os"
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

func TestSourceCheckRejectsMissingExchangeXMLAssets(t *testing.T) {
	root := minimalSourceCheckRoot(t)
	missing := filepath.Join(".aether", "exchange", "queen-wisdom.xml")
	if err := os.Remove(filepath.Join(root, missing)); err != nil {
		t.Fatalf("remove exchange fixture: %v", err)
	}

	result := runSourceCheck(root)
	if result.OK {
		t.Fatal("source check should fail when required exchange XML assets are missing")
	}

	if !sourceCheckHasIssue(result, missing, "required exchange XML asset is missing") {
		t.Fatalf("missing exchange XML issue, got: %+v", result.Issues)
	}
}

func TestSourceCheckAcceptsMatchingWrapperContractFields(t *testing.T) {
	root := minimalSourceCheckRoot(t)

	result := runSourceCheck(root)
	if !result.OK {
		t.Fatalf("source check should accept matching wrapper contract fields, issues: %+v", result.Issues)
	}
}

func TestSourceCheckRejectsWrapperFrontmatterDrift(t *testing.T) {
	root := minimalSourceCheckRoot(t)
	writeFile(t, root, filepath.Join(".claude", "commands", "ant", "status.md"), sourceCheckWrapperFixture(
		".aether/commands/status.yaml",
		"ant-wrong",
		"📊 Show colony status at a glance through the Aether CLI runtime",
		"AETHER_OUTPUT_MODE=visual aether status",
	))

	result := runSourceCheck(root)
	if result.OK {
		t.Fatal("source check should fail when generated wrapper frontmatter drifts from YAML")
	}

	if !sourceCheckHasIssue(result, ".claude/commands/ant/status.md", "generated wrapper frontmatter name does not match YAML") {
		t.Fatalf("missing wrapper name drift issue, got: %+v", result.Issues)
	}
}

func TestSourceCheckRejectsYAMLCommandNameDrift(t *testing.T) {
	root := minimalSourceCheckRoot(t)
	writeFile(t, root, filepath.Join(".aether", "commands", "status.yaml"), []byte(`name: ant-wrong
description: "📊 Show colony status at a glance through the Aether CLI runtime"
source_of_truth: "Use the Go `+"`"+`aether`+"`"+` CLI as the source of truth."
runtime:
  command: "AETHER_OUTPUT_MODE=visual aether status $ARGUMENTS"
`))

	result := runSourceCheck(root)
	if result.OK {
		t.Fatal("source check should fail when YAML command name drifts from the filename")
	}

	if !sourceCheckHasIssue(result, ".aether/commands/status.yaml", "YAML command name does not match command file") {
		t.Fatalf("missing YAML name drift issue, got: %+v", result.Issues)
	}
}

func TestSourceCheckRejectsMissingWrapperDescription(t *testing.T) {
	root := minimalSourceCheckRoot(t)
	writeFile(t, root, filepath.Join(".claude", "commands", "ant", "status.md"), sourceCheckWrapperFixture(
		".aether/commands/status.yaml",
		"ant-status",
		"",
		"AETHER_OUTPUT_MODE=visual aether status",
	))

	result := runSourceCheck(root)
	if result.OK {
		t.Fatal("source check should fail when generated wrapper frontmatter is missing description")
	}

	if !sourceCheckHasIssue(result, ".claude/commands/ant/status.md", "generated wrapper frontmatter is missing description") {
		t.Fatalf("missing wrapper description issue, got: %+v", result.Issues)
	}
}

func TestSourceCheckRejectsRuntimeCommandDrift(t *testing.T) {
	root := minimalSourceCheckRoot(t)
	writeFile(t, root, filepath.Join(".opencode", "commands", "ant", "status.md"), sourceCheckWrapperFixture(
		".aether/commands/status.yaml",
		"ant-status",
		"📊 Show colony status at a glance through the Aether CLI runtime",
		"AETHER_OUTPUT_MODE=visual aether wrong-status",
	))

	result := runSourceCheck(root)
	if result.OK {
		t.Fatal("source check should fail when generated wrapper runtime command drifts from YAML")
	}

	if !sourceCheckHasIssue(result, ".opencode/commands/ant/status.md", "generated wrapper is missing YAML runtime command") {
		t.Fatalf("missing runtime command drift issue, got: %+v", result.Issues)
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
	for _, name := range sourceCheckRequiredExchangeXMLAssets {
		writeFile(t, root, filepath.Join(".aether", "exchange", name), []byte("<fixture />\n"))
	}
	writeFile(t, root, filepath.Join(".aether", "commands", "status.yaml"), []byte(`name: ant-status
description: "📊 Show colony status at a glance through the Aether CLI runtime"
source_of_truth: "Use the Go `+"`"+`aether`+"`"+` CLI as the source of truth."
runtime:
  command: "AETHER_OUTPUT_MODE=visual aether status $ARGUMENTS"
`))
	wrapper := sourceCheckWrapperFixture(
		".aether/commands/status.yaml",
		"ant-status",
		"📊 Show colony status at a glance through the Aether CLI runtime",
		"AETHER_OUTPUT_MODE=visual aether status",
	)
	writeFile(t, root, filepath.Join(".claude", "commands", "ant", "status.md"), wrapper)
	writeFile(t, root, filepath.Join(".opencode", "commands", "ant", "status.md"), wrapper)

	return root
}

func sourceCheckWrapperFixture(source, name, description, runtimeCommand string) []byte {
	return []byte(`<!-- Aether-managed: runtime spec at ` + source + `. Synced by aether update. -->
---
name: ` + name + `
description: "` + description + `"
---

Use the Go ` + "`" + `aether` + "`" + ` CLI as the source of truth.

` + runtimeCommand + `

- If docs and runtime disagree, runtime wins.
`)
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
