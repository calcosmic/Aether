package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRunUpdateSyncMaintainsLocalOnlyAetherState(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()
	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(filepath.Join(hubSystem, "docs"), 0755); err != nil {
		t.Fatalf("create hub docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("write hub workers: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "docs", "guide.md"), []byte("# Guide"), 0644); err != nil {
		t.Fatalf("write hub docs: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, false)
	if len(result.errors) > 0 {
		t.Fatalf("runUpdateSync errors: %v", result.errors)
	}

	for _, rel := range []string{"data", "checkpoints", "locks"} {
		if info, err := os.Stat(filepath.Join(repoDir, ".aether", rel)); err != nil || !info.IsDir() {
			t.Fatalf("expected .aether/%s directory, err=%v", rel, err)
		}
	}
	for _, rel := range []string{"workers.md", filepath.Join("docs", "guide.md")} {
		path := filepath.Join(repoDir, ".aether", rel)
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("expected %s to stay global, but update copied it", rel)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}

func TestRunUpdateSyncPrunesLegacyRepoAssetsButPreservesCustomFiles(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(hubDir, "system", "skills", "colony", "build-discipline"), 0755); err != nil {
		t.Fatalf("create hub skills: %v", err)
	}
	shippedSkill := []byte("# Shipped skill\n")
	if err := os.WriteFile(filepath.Join(hubDir, "system", "skills", "colony", "build-discipline", "SKILL.md"), shippedSkill, 0644); err != nil {
		t.Fatalf("write hub skill: %v", err)
	}

	generated := []byte("<!-- Generated from .aether/commands/build.yaml - DO NOT EDIT DIRECTLY -->\n# Build\n")
	custom := []byte("# Custom command\n")
	legacyCommandDir := filepath.Join(repoDir, ".claude", "commands", "ant")
	if err := os.MkdirAll(legacyCommandDir, 0755); err != nil {
		t.Fatalf("create legacy commands: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyCommandDir, "build.md"), generated, 0644); err != nil {
		t.Fatalf("write generated command: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyCommandDir, "custom.md"), custom, 0644); err != nil {
		t.Fatalf("write custom command: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoDir, ".codex", "agents"), 0755); err != nil {
		t.Fatalf("create codex agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ".codex", "agents", "aether-builder.toml"), validCodexAgentTOML("aether-builder", "builder"), 0644); err != nil {
		t.Fatalf("write codex agent: %v", err)
	}
	localSkillDir := filepath.Join(repoDir, ".aether", "skills", "colony", "build-discipline")
	if err := os.MkdirAll(localSkillDir, 0755); err != nil {
		t.Fatalf("create local skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localSkillDir, "SKILL.md"), shippedSkill, 0644); err != nil {
		t.Fatalf("write local shipped skill: %v", err)
	}
	customSkillDir := filepath.Join(repoDir, ".aether", "skills", "domain", "repo-only")
	if err := os.MkdirAll(customSkillDir, 0755); err != nil {
		t.Fatalf("create custom skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(customSkillDir, "SKILL.md"), []byte("# Repo skill\n"), 0644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	writeRepoFile := func(rel string, data []byte) {
		t.Helper()
		path := filepath.Join(repoDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("create parent for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	staleSystemFiles := []string{
		".aether/codex/aether-builder.toml",
		".aether/settings/claude/settings.json",
		".aether/.npmignore",
		".aether/aether-utils.sh",
		".aether/ledger.jsonl",
		".aether/manifest.json",
		".aether/model-profiles.yaml",
		".aether/registry.json",
		".aether/version.json",
		".aether/workers.md",
	}
	for _, rel := range staleSystemFiles {
		writeRepoFile(rel, []byte("stale managed system file\n"))
	}
	preservedLocalFiles := map[string][]byte{
		".aether/CONTEXT.md":                  []byte("# Local context\n"),
		".aether/HANDOFF.md":                  []byte("# Local handoff\n"),
		".aether/CROWNED-ANTHILL.md":          []byte("# Local seal\n"),
		".aether/chambers/demo/manifest.json": []byte(`{"name":"demo"}`),
		".aether/custom/manifest.json":        []byte(`{"name":"local"}`),
		".aether/temp/scratch.txt":            []byte("scratch\n"),
	}
	for rel, data := range preservedLocalFiles {
		writeRepoFile(rel, data)
	}

	result := runUpdateSync(hubDir, repoDir, false)
	if len(result.errors) > 0 {
		t.Fatalf("runUpdateSync errors: %v", result.errors)
	}

	if _, err := os.Stat(filepath.Join(legacyCommandDir, "build.md")); err == nil {
		t.Fatal("expected generated repo-local Claude command to be pruned")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat generated command: %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacyCommandDir, "custom.md")); err != nil {
		t.Fatalf("custom command should be preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoDir, ".codex", "agents", "aether-builder.toml")); err == nil {
		t.Fatal("expected repo-local Codex agent to be pruned")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat codex agent: %v", err)
	}
	if _, err := os.Stat(filepath.Join(localSkillDir, "SKILL.md")); err == nil {
		t.Fatal("expected unchanged shipped repo skill to be pruned")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat shipped skill: %v", err)
	}
	if _, err := os.Stat(filepath.Join(customSkillDir, "SKILL.md")); err != nil {
		t.Fatalf("custom repo skill should be preserved: %v", err)
	}
	for _, rel := range staleSystemFiles {
		if _, err := os.Stat(filepath.Join(repoDir, filepath.FromSlash(rel))); err == nil {
			t.Fatalf("expected stale system file %s to be pruned", rel)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat stale system file %s: %v", rel, err)
		}
	}
	for rel, want := range preservedLocalFiles {
		got, err := os.ReadFile(filepath.Join(repoDir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("local file %s should be preserved: %v", rel, err)
		}
		if string(got) != string(want) {
			t.Fatalf("local file %s changed: got %q want %q", rel, string(got), string(want))
		}
	}
}

func TestRunUpdateSyncDoesNotPruneAetherSourceCheckout(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module github.com/calcosmic/Aether\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	mainPath := filepath.Join(repoDir, "cmd", "aether", "main.go")
	if err := os.MkdirAll(filepath.Dir(mainPath), 0755); err != nil {
		t.Fatalf("create cmd/aether: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	preserved := map[string][]byte{
		".aether/version.json":   []byte(`{"version":"9.9.9"}`),
		".aether/manifest.json":  []byte(`{"files":{}}`),
		".aether/codex/dev.toml": []byte("name = \"dev\"\n"),
		".aether/docs/dev.md":    []byte("# Dev doc\n"),
		".aether/workers.md":     []byte("# Workers\n"),
	}
	for rel, data := range preserved {
		path := filepath.Join(repoDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("create parent for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	result := runUpdateSync(hubDir, repoDir, true)
	if len(result.errors) > 0 {
		t.Fatalf("runUpdateSync errors: %v", result.errors)
	}
	for rel, want := range preserved {
		got, err := os.ReadFile(filepath.Join(repoDir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("source checkout file %s should be preserved: %v", rel, err)
		}
		if string(got) != string(want) {
			t.Fatalf("source checkout file %s changed: got %q want %q", rel, string(got), string(want))
		}
	}
}

func TestRunUpdateSyncPreservesProtectedState(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()
	localAether := filepath.Join(repoDir, ".aether")
	statePath := filepath.Join(localAether, "data", "COLONY_STATE.json")
	queenPath := filepath.Join(localAether, "QUEEN.md")
	if err := os.MkdirAll(filepath.Dir(statePath), 0755); err != nil {
		t.Fatalf("create state dir: %v", err)
	}
	state := []byte(`{"state":"ACTIVE"}`)
	queen := []byte("# Local Queen\n")
	if err := os.WriteFile(statePath, state, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}
	if err := os.WriteFile(queenPath, queen, 0644); err != nil {
		t.Fatalf("write queen: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, true)
	if len(result.errors) > 0 {
		t.Fatalf("runUpdateSync errors: %v", result.errors)
	}
	if got, err := os.ReadFile(statePath); err != nil || string(got) != string(state) {
		t.Fatalf("state changed: got %q err=%v", string(got), err)
	}
	if got, err := os.ReadFile(queenPath); err != nil || string(got) != string(queen) {
		t.Fatalf("queen changed: got %q err=%v", string(got), err)
	}
}

func TestRunUpdateSyncCopiesAllowedClaudeRepoFiles(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()
	settingsDir := filepath.Join(hubDir, "system", "settings", "claude")
	rulesDir := filepath.Join(hubDir, "system", "rules")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatalf("create settings dir: %v", err)
	}
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("create rules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte("{\"hooks\":{}}\n"), 0644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "testing.md"), []byte("# Testing\n"), 0644); err != nil {
		t.Fatalf("write rule: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, true)
	if len(result.errors) > 0 {
		t.Fatalf("runUpdateSync errors: %v", result.errors)
	}
	for _, rel := range []string{filepath.Join(".claude", "settings.json"), filepath.Join(".claude", "rules", "testing.md")} {
		if _, err := os.Stat(filepath.Join(repoDir, rel)); err != nil {
			t.Fatalf("expected %s to sync: %v", rel, err)
		}
	}
}

func TestSyncPlatformHomeAssetsFromHubRefreshesGlobalPlatformHomes(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	homeDir := t.TempDir()
	systemDir := filepath.Join(hubDir, "system")

	writeHubFile := func(rel string, data []byte) {
		t.Helper()
		path := filepath.Join(systemDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("create parent for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("write hub %s: %v", rel, err)
		}
	}

	claudeCommand := []byte("<!-- Generated from .aether/commands/build.yaml - DO NOT EDIT DIRECTLY -->\n# Build from hub\n")
	claudeAgent := []byte("---\nname: aether-builder\ndescription: Builder agent\n---\n# Claude Builder\n")
	openCodeCommand := []byte("# OpenCode Build from hub\n")
	openCodeAgent := []byte("---\n" + validAgentFrontmatter + "\n---\n\n# OpenCode Builder\n")
	codexAgent := validCodexAgentTOML("aether-builder", "builder")

	writeHubFile(filepath.Join("commands", "claude", "build.md"), claudeCommand)
	writeHubFile(filepath.Join("agents-claude", "aether-builder.md"), claudeAgent)
	writeHubFile(filepath.Join("commands", "opencode", "build.md"), openCodeCommand)
	writeHubFile(filepath.Join("agents", "aether-builder.md"), openCodeAgent)
	writeHubFile(filepath.Join("codex", "aether-builder.toml"), codexAgent)

	staleClaudeCommand := filepath.Join(homeDir, ".claude", "commands", "ant-build.md")
	if err := os.MkdirAll(filepath.Dir(staleClaudeCommand), 0755); err != nil {
		t.Fatalf("create stale Claude command parent: %v", err)
	}
	if err := os.WriteFile(staleClaudeCommand, []byte("stale\n"), 0644); err != nil {
		t.Fatalf("write stale Claude command: %v", err)
	}
	customOpenCodeAgent := filepath.Join(homeDir, ".config", "opencode", "agents", "mds-implementer.md")
	if err := os.MkdirAll(filepath.Dir(customOpenCodeAgent), 0755); err != nil {
		t.Fatalf("create custom OpenCode agent parent: %v", err)
	}
	if err := os.WriteFile(customOpenCodeAgent, []byte("# Custom MDS agent\n"), 0644); err != nil {
		t.Fatalf("write custom OpenCode agent: %v", err)
	}

	results, errors := syncPlatformHomeAssetsFromHub(hubDir, homeDir, channelStable)
	if len(errors) > 0 {
		t.Fatalf("syncPlatformHomeAssetsFromHub errors: %v", errors)
	}
	if len(results) == 0 {
		t.Fatal("expected platform sync results")
	}

	assertFileContent := func(rel string, want []byte) {
		t.Helper()
		got, err := os.ReadFile(filepath.Join(homeDir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if string(got) != string(want) {
			t.Fatalf("%s content mismatch\ngot:\n%s\nwant:\n%s", rel, string(got), string(want))
		}
	}

	assertFileContent(filepath.Join(".claude", "commands", "ant-build.md"), claudeCommand)
	assertFileContent(filepath.Join(".claude", "agents", "ant", "aether-builder.md"), claudeAgent)
	assertFileContent(filepath.Join(".config", "opencode", "commands", "ant", "build.md"), openCodeCommand)
	assertFileContent(filepath.Join(".config", "opencode", "agents", "aether-builder.md"), openCodeAgent)
	assertFileContent(filepath.Join(".codex", "agents", "aether-builder.toml"), codexAgent)

	if _, err := os.Stat(customOpenCodeAgent); err != nil {
		t.Fatalf("custom OpenCode agent should be preserved: %v", err)
	}
}

func TestCheckStalePublishExpectedCounts(t *testing.T) {
	hubDir := t.TempDir()
	createHubWithExpectedCounts(t, hubDir)

	result := checkStalePublish(hubDir, "1.0.27", "1.0.27", channelStable, nil)
	if result.Classification != staleOK {
		t.Fatalf("expected staleOK, got %s: %+v", result.Classification, result.Components)
	}
}

func createHubWithExpectedCounts(t *testing.T, hubDir string) {
	t.Helper()
	system := filepath.Join(hubDir, "system")

	for rel, count := range map[string]int{
		filepath.Join("commands", "claude"):   expectedClaudeCommandCount,
		filepath.Join("commands", "opencode"): expectedOpenCodeCommandCount,
		"agents-claude":                       expectedClaudeAgents,
		"agents":                              expectedOpenCodeAgentCount,
		"codex":                               expectedCodexAgentCount,
	} {
		dir := filepath.Join(system, rel)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("create %s: %v", rel, err)
		}
		ext := ".md"
		if rel == "codex" {
			ext = ".toml"
		}
		for i := 0; i < count; i++ {
			content := []byte("# test")
			if filepath.ToSlash(rel) == "agents" {
				content = []byte(fmt.Sprintf(`---
name: aether-fixture-%02d
description: "OpenCode fixture agent used for update and stale publish tests"
mode: subagent
color: "#3b82f6"
tools:
  write: true
  edit: true
  bash: true
  grep: true
  glob: true
  task: true
---

# OpenCode fixture agent
`, i))
			}
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file_%02d%s", i, ext)), content, 0644); err != nil {
				t.Fatalf("write %s fixture: %v", rel, err)
			}
		}
	}

	for rel, count := range map[string]int{
		filepath.Join("skills", "colony"): expectedColonySkills,
		filepath.Join("skills", "domain"): expectedDomainSkills,
	} {
		for i := 0; i < count; i++ {
			dir := filepath.Join(system, rel, fmt.Sprintf("skill_%02d", i))
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatalf("create skill fixture: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# test"), 0644); err != nil {
				t.Fatalf("write skill fixture: %v", err)
			}
		}
	}
}
