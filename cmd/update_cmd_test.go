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
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file_%02d%s", i, ext)), []byte("# test"), 0644); err != nil {
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
