package cmd

import (
	"encoding/json"
	"fmt"
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
		"generated Codex skill surface",
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
	seedCodexSkillSupportFixture(t, root)
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

func TestCodexAntSkillSourceCheck(t *testing.T) {
	for _, kind := range []string{"clean", "missing-name", "duplicate-name", "duplicate-directory", "unexpected-name", "bad-frontmatter", "wrong-guide", "route-drift", "missing-support-reference", "missing-support-file", "empty-support-file"} {
		t.Run(kind, func(t *testing.T) {
			antPayloadEnvironment(t)
			root := minimalSourceCheckRoot(t)
			original := sourceCheckCodexShims
			t.Cleanup(func() { sourceCheckCodexShims = original })
			shims := original()
			index := -1
			for i, shim := range shims {
				if shim.Name == "ant-plan" {
					index = i
				}
			}
			if index < 0 {
				t.Fatal("actual generator omitted ant-plan")
			}
			want := ""
			switch kind {
			case "missing-name":
				shims = append(shims[:index], shims[index+1:]...)
				want = "missing public skill"
			case "duplicate-name":
				shims[index].Name = "ant-init"
				want = "duplicate public name"
			case "duplicate-directory":
				shims = append(shims, shims[index])
				want = "duplicate public directory"
			case "unexpected-name":
				shims[index].Name, shims[index].Dir = "ant-pause", "ant-pause"
				want = "unexpected public skill"
			case "bad-frontmatter":
				shims[index].Description = "[invalid YAML"
				want = "frontmatter is invalid"
			case "wrong-guide":
				shims[index].Body = strings.Replace(shims[index].Body, "aether command-guide plan --platform codex", "aether command-guide build --platform codex", 1)
				want = "guide route"
			case "route-drift":
				shims[index].Body = strings.Replace(shims[index].Body, "aether plan-finalize", "aether plan", 1)
				want = "runtime route"
			case "missing-support-reference":
				shims[index].Body = strings.ReplaceAll(shims[index].Body, "../support/aether-colony-build-cycle.md", "../support/missing.md")
				want = "private support reference"
			case "missing-support-file", "empty-support-file":
				path := filepath.Join(root, ".aether/skills/colony/aether-colony-build-cycle/SKILL.md")
				if kind == "missing-support-file" {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.WriteFile(path, nil, 0644); err != nil {
						t.Fatal(err)
					}
				}
				want = "private support source"
			}
			// Mutate the generator's subject, never the checker or its result.
			sourceCheckCodexShims = func() []codexSkillShim { return shims }
			before := antSnapshot(t, root)
			resetFlags(rootCmd)
			output, err := antPayloadCommand(t, "source-check", "--root", root, "--json")
			var result sourceCheckResult
			if decodeErr := json.Unmarshal([]byte(output), &result); decodeErr != nil {
				t.Fatalf("registered checker output: %v", decodeErr)
			}
			antAssertSnapshot(t, root, before)
			for _, component := range []string{"canonical source surfaces", "retired source mirrors", "generated command wrappers", "generated Codex skill surface"} {
				if !sourceCheckHasComponent(result, component) {
					t.Fatalf("registered checker omitted %s", component)
				}
			}
			if kind == "clean" {
				if err != nil || !result.OK {
					t.Fatalf("valid surface refused: %v %+v", err, result.Issues)
				}
				for _, component := range result.Components {
					if component.Name == "generated Codex skill surface" && component.Checked != 12 {
						t.Fatalf("expected nine public + three support checks: %+v", component)
					}
					if component.Name == "generated command wrappers" && component.Checked != 2 {
						t.Fatal("Claude/OpenCode wrappers no longer checked")
					}
				}
				return
			}
			if err == nil || result.OK || result.Verification.Status != "fail" || len(result.Blockers) == 0 {
				t.Fatalf("invalid %s accepted: %v %+v", kind, err, result)
			}
			found := false
			for _, issue := range result.Issues {
				if issue.Area == "codex_skills" && strings.Contains(issue.Message, want) && issue.Path != "" && issue.Expected != "" && issue.Actual != "" {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing structured %s finding: %s", want, fmt.Sprint(result.Issues))
			}
		})
	}
}
