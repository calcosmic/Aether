package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// TestCodexInstallCopiesAgents verifies that install copies .codex/agents/
// files to the home directory (~/.codex/agents/).
func TestCodexInstallCopiesAgents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".codex", "agents")
	destDir := filepath.Join(homeDir, ".codex", "agents")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "aether-builder.toml"), validCodexAgentTOML("aether-builder", "builder"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	destFile := filepath.Join(destDir, "aether-builder.toml")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist after install", destFile)
	}
}

// TestCodexInstallCopiesAgentsToHub verifies that install syncs .codex/
// files to the hub system directory (~/.aether/system/codex/).
func TestCodexInstallCopiesAgentsToHub(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".codex", "agents")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "aether-builder.toml"), validCodexAgentTOML("aether-builder", "builder"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	hubCodexFile := filepath.Join(homeDir, ".aether", "system", "codex", "aether-builder.toml")
	if _, err := os.Stat(hubCodexFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist after install", hubCodexFile)
	}
}

// TestCodexInstallAgentsEmpty verifies that install handles missing .codex/
// directory gracefully (no error, no crash).
func TestCodexInstallAgentsEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".aether"), 0755); err != nil {
		t.Fatalf("failed to create .aether dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".aether", "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to create workers.md: %v", err)
	}
	// Intentionally do NOT create .codex/ directory

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install should not fail with missing .codex/: %v", err)
	}

	// Output should still be valid JSON with ok:true
	output := buf.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("expected valid JSON output, got parse error: %v", err)
	}
	if ok, exists := result["ok"]; !exists || ok != true {
		t.Errorf("expected JSON output with ok:true, got: %v", result)
	}

	// ~/.codex/agents/ should not be created when source is missing
	destDir := filepath.Join(homeDir, ".codex", "agents")
	if _, err := os.Stat(destDir); err == nil {
		t.Error("expected .codex/agents/ to NOT exist when source is missing")
	}
}

// TestCodexInstallAgentContent verifies that copied Codex agent files have
// the correct content (byte-for-byte match).
func TestCodexInstallAgentContent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".codex", "agents")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	expectedContent := validCodexAgentTOML("aether-builder", "builder")
	if err := os.WriteFile(filepath.Join(srcDir, "aether-builder.toml"), expectedContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	// Verify content in ~/.codex/agents/
	destFile := filepath.Join(homeDir, ".codex", "agents", "aether-builder.toml")
	actual, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}
	if string(actual) != string(expectedContent) {
		t.Errorf("content mismatch in ~/.codex/agents/\ngot:  %s\nwant: %s", string(actual), string(expectedContent))
	}

	// Verify content in hub ~/.aether/system/codex/
	hubFile := filepath.Join(homeDir, ".aether", "system", "codex", "aether-builder.toml")
	hubActual, err := os.ReadFile(hubFile)
	if err != nil {
		t.Fatalf("failed to read hub file: %v", err)
	}
	if string(hubActual) != string(expectedContent) {
		t.Errorf("content mismatch in hub ~/.aether/system/codex/\ngot:  %s\nwant: %s", string(hubActual), string(expectedContent))
	}
}

// TestCodexSetupCopiesAgents verifies that setup copies Codex agent files
// from the hub to a target repository.
// Install syncs .codex/agents/ -> system/codex/, so hub stores at system/codex/*.toml.
// Setup sync pair maps system/codex/ -> .codex/agents/, landing at .codex/agents/*.toml.
func TestCodexSetupCopiesAgents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	packageDir := t.TempDir()
	homeDir := t.TempDir()
	repoDir := t.TempDir()

	// Create .codex/agents/ in the package
	srcDir := filepath.Join(packageDir, ".codex", "agents")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	agentContent := validCodexAgentTOML("aether-builder", "builder")
	if err := os.WriteFile(filepath.Join(srcDir, "aether-builder.toml"), agentContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a minimal .aether/ so the hub has version.json
	pkgAether := filepath.Join(packageDir, ".aether")
	if err := os.MkdirAll(pkgAether, 0755); err != nil {
		t.Fatalf("failed to create package .aether dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgAether, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}

	// Step 1: Install to populate hub
	var installBuf bytes.Buffer
	stdout = &installBuf
	rootCmd.SetArgs([]string{"install", "--package-dir", packageDir, "--home-dir", homeDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	// Step 2: Setup in target repo
	saveGlobals(t)
	resetRootCmd(t)

	var setupBuf bytes.Buffer
	stdout = &setupBuf
	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("setup command failed: %v", err)
	}

	// Verify Codex agents were synced to repo.
	// Hub stores at system/codex/*.toml; setup syncs system/codex/ -> .codex/agents/
	// so files land at .codex/agents/*.toml (flat, no nesting).
	repoCodexFile := filepath.Join(repoDir, ".codex", "agents", "aether-builder.toml")
	if _, err := os.Stat(repoCodexFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist after setup", repoCodexFile)
	}

	// Verify content matches
	actual, err := os.ReadFile(repoCodexFile)
	if err != nil {
		t.Fatalf("failed to read repo codex agent: %v", err)
	}
	if string(actual) != string(agentContent) {
		t.Errorf("content mismatch after setup\ngot:  %s\nwant: %s", string(actual), string(agentContent))
	}
}

// TestCodexUpdateCopiesAgents verifies that update copies Codex agent files
// from the hub to a target repository.
// Install syncs .codex/agents/ -> system/codex/, so hub stores at system/codex/*.toml.
// Setup/update sync pair maps system/codex/ -> .codex/agents/, landing at .codex/agents/*.toml.
func TestCodexUpdateCopiesAgents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	packageDir := t.TempDir()
	homeDir := t.TempDir()
	repoDir := t.TempDir()

	// Create .codex/agents/ in the package
	srcDir := filepath.Join(packageDir, ".codex", "agents")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	agentContent := validCodexAgentTOML("aether-builder", "builder")
	if err := os.WriteFile(filepath.Join(srcDir, "aether-builder.toml"), agentContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a minimal .aether/ so the hub has version.json
	pkgAether := filepath.Join(packageDir, ".aether")
	if err := os.MkdirAll(pkgAether, 0755); err != nil {
		t.Fatalf("failed to create package .aether dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgAether, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}

	// Step 1: Install
	var installBuf bytes.Buffer
	stdout = &installBuf
	rootCmd.SetArgs([]string{"install", "--package-dir", packageDir, "--home-dir", homeDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	// Step 2: Setup
	saveGlobals(t)
	resetRootCmd(t)

	var setupBuf bytes.Buffer
	stdout = &setupBuf
	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup command failed: %v", err)
	}

	// Step 3: Update hub with a new Codex agent placed directly in system/codex/
	// placed directly in system/codex/ so it syncs cleanly to .codex/agents/
	hubSystem := filepath.Join(homeDir, ".aether", "system")
	hubCodexDir := filepath.Join(hubSystem, "codex")
	if err := os.MkdirAll(hubCodexDir, 0755); err != nil {
		t.Fatalf("failed to create hub codex dir: %v", err)
	}
	newAgentContent := validCodexAgentTOML("aether-watcher", "watcher")
	if err := os.WriteFile(filepath.Join(hubCodexDir, "aether-watcher.toml"), newAgentContent, 0644); err != nil {
		t.Fatalf("failed to write new codex agent to hub: %v", err)
	}

	// Step 4: Update
	saveGlobals(t)
	resetRootCmd(t)

	var updateBuf bytes.Buffer
	stdout = &updateBuf
	t.Setenv("HOME", homeDir)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer os.Chdir(oldDir)

	rootCmd.SetArgs([]string{"update", "--force"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("update command failed: %v", err)
	}

	// Verify the new Codex agent was synced to repo
	repoCodexFile := filepath.Join(repoDir, ".codex", "agents", "aether-watcher.toml")
	if _, err := os.Stat(repoCodexFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist after update", repoCodexFile)
	}

	// Verify content matches
	actual, err := os.ReadFile(repoCodexFile)
	if err != nil {
		t.Fatalf("failed to read repo codex agent: %v", err)
	}
	if string(actual) != string(newAgentContent) {
		t.Errorf("content mismatch after update\ngot:  %s\nwant: %s", string(actual), string(newAgentContent))
	}
}

// TestCodexE2EFullLifecycle verifies the complete install -> setup -> update
// lifecycle for Codex agents end-to-end.
func TestCodexE2EFullLifecycle(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	packageDir := t.TempDir()
	homeDir := t.TempDir()
	repoDir := t.TempDir()

	// Create package structure with .codex/agents/
	pkgCodex := filepath.Join(packageDir, ".codex", "agents")
	if err := os.MkdirAll(pkgCodex, 0755); err != nil {
		t.Fatalf("failed to create codex agents dir: %v", err)
	}
	builderContent := validCodexAgentTOML("aether-builder", "builder")
	if err := os.WriteFile(filepath.Join(pkgCodex, "aether-builder.toml"), builderContent, 0644); err != nil {
		t.Fatalf("failed to create builder.toml: %v", err)
	}
	watcherContent := validCodexAgentTOML("aether-watcher", "watcher")
	if err := os.WriteFile(filepath.Join(pkgCodex, "aether-watcher.toml"), watcherContent, 0644); err != nil {
		t.Fatalf("failed to create watcher.toml: %v", err)
	}

	// Create a minimal .aether/ so the hub has version.json
	pkgAether := filepath.Join(packageDir, ".aether")
	if err := os.MkdirAll(pkgAether, 0755); err != nil {
		t.Fatalf("failed to create package .aether dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgAether, "workers.md"), []byte("# Workers v1"), 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}

	// ===== Phase 1: Install =====
	t.Run("install", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf

		rootCmd.SetArgs([]string{"install", "--package-dir", packageDir, "--home-dir", homeDir})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("install failed: %v", err)
		}

		output := buf.String()
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			t.Fatalf("install output not valid JSON: %v", err)
		}
		if ok, _ := result["ok"].(bool); !ok {
			t.Fatalf("install returned ok:false, output: %s", output)
		}

		// Verify Codex agents in home dir
		for _, name := range []string{"aether-builder.toml", "aether-watcher.toml"} {
			f := filepath.Join(homeDir, ".codex", "agents", name)
			if _, err := os.Stat(f); os.IsNotExist(err) {
				t.Errorf("expected %s in home dir after install", f)
			}
		}

		// Verify Codex agents in hub
		hubCodex := filepath.Join(homeDir, ".aether", "system", "codex")
		for _, name := range []string{"aether-builder.toml", "aether-watcher.toml"} {
			f := filepath.Join(hubCodex, name)
			if _, err := os.Stat(f); os.IsNotExist(err) {
				t.Errorf("expected %s in hub after install", f)
			}
		}
	})

	// ===== Phase 2: Setup =====
	t.Run("setup", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf

		rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// Verify Codex agents synced to repo.
		// Hub stores at system/codex/*.toml; setup syncs system/codex/ -> .codex/agents/
		// so files land at .codex/agents/*.toml (flat, no nesting).
		repoCodex := filepath.Join(repoDir, ".codex", "agents")
		for _, name := range []string{"aether-builder.toml", "aether-watcher.toml"} {
			f := filepath.Join(repoCodex, name)
			if _, err := os.Stat(f); os.IsNotExist(err) {
				t.Errorf("expected %s in repo after setup", f)
			}
		}

		// Verify content matches
		repoBuilder, err := os.ReadFile(filepath.Join(repoCodex, "aether-builder.toml"))
		if err != nil {
			t.Fatalf("failed to read builder.toml from repo: %v", err)
		}
		if string(repoBuilder) != string(builderContent) {
			t.Errorf("builder.toml content mismatch after setup\ngot:  %s\nwant: %s", string(repoBuilder), string(builderContent))
		}
	})

	// ===== Phase 3: Update (add new agent to hub, update should sync it) =====
	t.Run("update", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		// Add a new agent directly in hub system/codex/ (not in agents/ subdir)
		// so it syncs cleanly to .codex/agents/ without the extra nesting level
		hubCodex := filepath.Join(homeDir, ".aether", "system", "codex")
		if err := os.MkdirAll(hubCodex, 0755); err != nil {
			t.Fatalf("failed to create hub codex dir: %v", err)
		}
		sageContent := validCodexAgentTOML("aether-sage", "sage")
		if err := os.WriteFile(filepath.Join(hubCodex, "aether-sage.toml"), sageContent, 0644); err != nil {
			t.Fatalf("failed to write sage.toml to hub: %v", err)
		}

		var buf bytes.Buffer
		stdout = &buf

		t.Setenv("HOME", homeDir)
		oldDir, _ := os.Getwd()
		if err := os.Chdir(repoDir); err != nil {
			t.Fatalf("failed to chdir to repo: %v", err)
		}
		defer os.Chdir(oldDir)

		rootCmd.SetArgs([]string{"update", "--force"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("update failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "ok") {
			t.Errorf("expected update output to contain ok, got: %s", output)
		}

		// Verify new agent was synced to .codex/agents/ (top level, no nesting)
		repoSage := filepath.Join(repoDir, ".codex", "agents", "aether-sage.toml")
		if _, err := os.Stat(repoSage); os.IsNotExist(err) {
			t.Errorf("expected %s to exist after update", repoSage)
		}

		// Verify content
		actual, err := os.ReadFile(repoSage)
		if err != nil {
			t.Fatalf("failed to read sage.toml from repo: %v", err)
		}
		if string(actual) != string(sageContent) {
			t.Errorf("sage.toml content mismatch after update\ngot:  %s\nwant: %s", string(actual), string(sageContent))
		}

		// Original agents should still exist after update
		repoBuilder := filepath.Join(repoDir, ".codex", "agents", "aether-builder.toml")
		if _, err := os.Stat(repoBuilder); os.IsNotExist(err) {
			t.Error("expected aether-builder.toml to still exist after update")
		}
	})
}

// TestCodexInstallMultipleAgents verifies that install correctly handles
// multiple Codex agent files.
func TestCodexInstallMultipleAgents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".codex", "agents")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	// Create multiple agent files
	agents := map[string][]byte{
		"aether-builder.toml":    validCodexAgentTOML("aether-builder", "builder"),
		"aether-watcher.toml":    validCodexAgentTOML("aether-watcher", "watcher"),
		"aether-scout.toml":      validCodexAgentTOML("aether-scout", "scout"),
		"aether-chronicler.toml": validCodexAgentTOML("aether-chronicler", "chronicler"),
	}
	for name, content := range agents {
		if err := os.WriteFile(filepath.Join(srcDir, name), content, 0644); err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	// Verify all agents were copied to home dir
	destDir := filepath.Join(homeDir, ".codex", "agents")
	for name := range agents {
		f := filepath.Join(destDir, name)
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist after install", f)
		}
	}

	// Verify all agents were copied to hub
	hubDir := filepath.Join(homeDir, ".aether", "system", "codex")
	for name := range agents {
		f := filepath.Join(hubDir, name)
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist in hub after install", f)
		}
	}
}

func TestCodexInstallPreservesModifiedHomeAgent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	packageDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(packageDir, ".codex", "agents")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	shipped := validCodexAgentTOML("aether-builder", "builder")
	if err := os.WriteFile(filepath.Join(srcDir, "aether-builder.toml"), shipped, 0644); err != nil {
		t.Fatalf("failed to create shipped agent: %v", err)
	}

	destDir := filepath.Join(homeDir, ".codex", "agents")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create destination dir: %v", err)
	}
	local := []byte(`name = "aether-builder"
description = "local override"
nickname_candidates = ["builder", "local-builder"]
developer_instructions = """
Keep my local builder instructions.
"""
`)
	destFile := filepath.Join(destDir, "aether-builder.toml")
	if err := os.WriteFile(destFile, local, 0644); err != nil {
		t.Fatalf("failed to seed local agent: %v", err)
	}

	rootCmd.SetArgs([]string{"install", "--package-dir", packageDir, "--home-dir", homeDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	got, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("failed to read preserved local agent: %v", err)
	}
	if string(got) != string(local) {
		t.Fatalf("expected install to preserve modified home agent\ngot:\n%s\nwant:\n%s", string(got), string(local))
	}
}

func TestCodexInstallPreservesModifiedHomeSkill(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	packageDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(packageDir, ".aether", "skills-codex", "colony", "build-discipline")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packageDir, ".aether", "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to create workers.md: %v", err)
	}
	shipped := []byte(`---
name: build-discipline
description: Shipped build discipline
type: colony
domains: []
agent_roles: [builder]
detect_files: []
detect_packages: []
priority: 10
version: 1
---
Shipped skill
`)
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), shipped, 0644); err != nil {
		t.Fatalf("failed to create shipped skill: %v", err)
	}

	destDir := filepath.Join(homeDir, ".codex", "skills", "aether", "colony", "build-discipline")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create destination skill dir: %v", err)
	}
	local := []byte(`---
name: build-discipline
description: Local override
type: colony
domains: []
agent_roles: [builder]
detect_files: []
detect_packages: []
priority: 10
version: 1
---
Local skill override
`)
	destFile := filepath.Join(destDir, "SKILL.md")
	if err := os.WriteFile(destFile, local, 0644); err != nil {
		t.Fatalf("failed to seed local skill: %v", err)
	}

	rootCmd.SetArgs([]string{"install", "--package-dir", packageDir, "--home-dir", homeDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	got, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("failed to read preserved local skill: %v", err)
	}
	if string(got) != string(local) {
		t.Fatalf("expected install to preserve modified home skill\ngot:\n%s\nwant:\n%s", string(got), string(local))
	}
}

// all25AgentNames is the canonical list of all 25 Aether agent names.
var all25AgentNames = []string{
	"aether-ambassador",
	"aether-archaeologist",
	"aether-architect",
	"aether-auditor",
	"aether-builder",
	"aether-chaos",
	"aether-chronicler",
	"aether-gatekeeper",
	"aether-includer",
	"aether-keeper",
	"aether-medic",
	"aether-measurer",
	"aether-oracle",
	"aether-probe",
	"aether-queen",
	"aether-route-setter",
	"aether-sage",
	"aether-scout",
	"aether-surveyor-disciplines",
	"aether-surveyor-nest",
	"aether-surveyor-pathogens",
	"aether-surveyor-provisions",
	"aether-tracker",
	"aether-watcher",
	"aether-weaver",
}

// codexAgentTOML represents the required fields in a Codex agent TOML file.
type codexAgentTOML struct {
	Name                  string   `toml:"name"`
	Description           string   `toml:"description"`
	NicknameCandidates    []string `toml:"nickname_candidates"`
	DeveloperInstructions string   `toml:"developer_instructions"`
}

// TestCodexInstallSetupUpdate_All24Agents verifies the full install -> setup
// pipeline deploys all 24 Codex TOML agents correctly, and that each agent
// file is valid TOML with all required fields.
func TestCodexInstallSetupUpdate_All24Agents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	packageDir := t.TempDir()
	homeDir := t.TempDir()
	repoDir := t.TempDir()

	// Step 1: Create mock package source with all 24 agent TOML files.
	pkgCodex := filepath.Join(packageDir, ".codex", "agents")
	if err := os.MkdirAll(pkgCodex, 0755); err != nil {
		t.Fatalf("failed to create codex agents dir: %v", err)
	}

	for _, agentName := range all25AgentNames {
		content := validCodexAgentTOML(agentName, agentName)
		filename := agentName + ".toml"
		if err := os.WriteFile(filepath.Join(pkgCodex, filename), content, 0644); err != nil {
			t.Fatalf("failed to create %s: %v", filename, err)
		}
	}

	// Create a minimal .aether/ so the hub has version.json.
	pkgAether := filepath.Join(packageDir, ".aether")
	if err := os.MkdirAll(pkgAether, 0755); err != nil {
		t.Fatalf("failed to create package .aether dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgAether, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}

	// Step 2: Install to populate hub.
	t.Run("install", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf

		rootCmd.SetArgs([]string{"install", "--package-dir", packageDir, "--home-dir", homeDir})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("install failed: %v", err)
		}

		// Verify all 24 agents exist in hub.
		hubCodex := filepath.Join(homeDir, ".aether", "system", "codex")
		for _, agentName := range all25AgentNames {
			f := filepath.Join(hubCodex, agentName+".toml")
			if _, err := os.Stat(f); os.IsNotExist(err) {
				t.Errorf("expected %s in hub after install", f)
			}
		}
	})

	// Step 3: Setup to sync from hub to repo.
	t.Run("setup", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf

		rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// Verify all 24 agent TOML files exist in target .codex/agents/.
		repoCodex := filepath.Join(repoDir, ".codex", "agents")
		for _, agentName := range all25AgentNames {
			f := filepath.Join(repoCodex, agentName+".toml")
			if _, err := os.Stat(f); os.IsNotExist(err) {
				t.Errorf("expected %s in repo after setup", f)
			}
		}

		// Verify each agent file is valid TOML with required fields.
		for _, agentName := range all25AgentNames {
			t.Run(agentName, func(t *testing.T) {
				data, err := os.ReadFile(filepath.Join(repoCodex, agentName+".toml"))
				if err != nil {
					t.Fatalf("failed to read %s.toml: %v", agentName, err)
				}

				var agent codexAgentTOML
				if _, err := toml.Decode(string(data), &agent); err != nil {
					t.Fatalf("invalid TOML in %s.toml: %v", agentName, err)
				}

				if agent.Name == "" {
					t.Errorf("%s: missing required field 'name'", agentName)
				}
				if agent.Description == "" {
					t.Errorf("%s: missing required field 'description'", agentName)
				}
				if agent.DeveloperInstructions == "" {
					t.Errorf("%s: missing required field 'developer_instructions'", agentName)
				}
			})
		}
	})
}

func listShippedAetherCodexAgentBaseNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory %s: %v", dir, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !isShippedAetherCodexAgent(entry.Name()) {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".toml"))
	}

	sort.Strings(names)
	return names
}

// TestCrossPlatformAgentParity verifies that the shipped Aether Codex surface
// stays in parity across Claude, OpenCode, Codex, and the packaged Codex mirror.
func TestCrossPlatformAgentParity(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	// Collect base names (stripped extensions) from each platform directory.
	claudeDir := filepath.Join(repoRoot, ".claude", "agents", "ant")
	opencodeDir := filepath.Join(repoRoot, ".opencode", "agents")
	codexDir := filepath.Join(repoRoot, ".codex", "agents")
	agentsCodexMirrorDir := filepath.Join(repoRoot, ".aether", "agents-codex")

	claudeNames := listAgentBaseNames(t, claudeDir, ".md")
	opencodeNames := listAgentBaseNames(t, opencodeDir, ".md")
	codexNames := listShippedAetherCodexAgentBaseNames(t, codexDir)
	agentsCodexNames := listShippedAetherCodexAgentBaseNames(t, agentsCodexMirrorDir)

	// Verify each directory has exactly 25 entries.
	const expectedCount = 26
	if len(claudeNames) != expectedCount {
		t.Errorf("Claude agents: expected %d, got %d", expectedCount, len(claudeNames))
	}
	if len(opencodeNames) != expectedCount {
		t.Errorf("OpenCode agents: expected %d, got %d", expectedCount, len(opencodeNames))
	}
	if len(codexNames) != expectedCount {
		t.Errorf("Codex agents: expected %d, got %d", expectedCount, len(codexNames))
	}
	if len(agentsCodexNames) != expectedCount {
		t.Errorf("agents-codex mirror: expected %d, got %d", expectedCount, len(agentsCodexNames))
	}

	// Verify all four have matching base names.
	if !slicesEqual(claudeNames, opencodeNames) {
		t.Errorf("Claude and OpenCode agent names do not match.\nClaude:  %v\nOpenCode: %v", claudeNames, opencodeNames)
	}
	if !slicesEqual(claudeNames, codexNames) {
		t.Errorf("Claude and shipped Codex agent names do not match.\nClaude: %v\nCodex:  %v", claudeNames, codexNames)
	}

	// Verify the packaging mirror matches the shipped Codex surface exactly.
	if !slicesEqual(codexNames, agentsCodexNames) {
		t.Errorf("Shipped Codex agents and agents-codex mirror names do not match.\nCodex:        %v\nAgents-codex: %v", codexNames, agentsCodexNames)
	}
}

// listAgentBaseNames reads a directory and returns sorted base names with the
// given extension stripped. Only files ending in the specified extension are
// included.
func listAgentBaseNames(t *testing.T, dir, ext string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory %s: %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ext) {
			base := strings.TrimSuffix(name, ext)
			names = append(names, base)
		}
	}
	sort.Strings(names)
	return names
}

// slicesEqual checks whether two sorted string slices are identical.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestClaudeOpenCodeAgentContentParity verifies structural parity between
// Claude and OpenCode agent files. Because the two platforms use different
// frontmatter schemas (OpenCode uses tools-as-object, hex colors, name field,
// optional model), only the body content (after the second ---
// delimiter) is compared. Frontmatter differences are expected and allowed.
func TestClaudeOpenCodeAgentContentParity(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	claudeDir := filepath.Join(repoRoot, ".claude", "agents", "ant")
	opencodeDir := filepath.Join(repoRoot, ".opencode", "agents")

	claudeNames := listAgentBaseNames(t, claudeDir, ".md")
	opencodeNames := listAgentBaseNames(t, opencodeDir, ".md")

	if !slicesEqual(claudeNames, opencodeNames) {
		t.Fatalf("Claude and OpenCode agent names do not match.\nClaude:   %v\nOpenCode: %v", claudeNames, opencodeNames)
	}

	var mismatches []string
	for _, name := range claudeNames {
		claudePath := filepath.Join(claudeDir, name+".md")
		opencodePath := filepath.Join(opencodeDir, name+".md")

		claudeBytes, err := os.ReadFile(claudePath)
		if err != nil {
			t.Fatalf("failed to read %s: %v", claudePath, err)
		}
		opencodeBytes, err := os.ReadFile(opencodePath)
		if err != nil {
			t.Fatalf("failed to read %s: %v", opencodePath, err)
		}

		claudeBody := strings.TrimSpace(extractBodyAfterFrontmatter(claudeBytes))
		opencodeBody := strings.TrimSpace(extractBodyAfterFrontmatter(opencodeBytes))

		if claudeBody == opencodeBody {
			continue
		}

		claudeLines := strings.Count(claudeBody, "\n")
		opencodeLines := strings.Count(opencodeBody, "\n")
		if len(claudeBody) > 0 && !strings.HasSuffix(claudeBody, "\n") {
			claudeLines++
		}
		if len(opencodeBody) > 0 && !strings.HasSuffix(opencodeBody, "\n") {
			opencodeLines++
		}
		diff := claudeLines - opencodeLines
		if diff < 0 {
			diff = -diff
		}

		mismatches = append(mismatches, fmt.Sprintf(
			"Claude/OpenCode agent body mismatch for %s.md:\n  Claude:   %d lines\n  OpenCode: %d lines\n  Diff:     %d lines",
			name, claudeLines, opencodeLines, diff,
		))
	}

	if len(mismatches) > 0 {
		t.Errorf("Agent body parity failures (%d of %d agents):\n\n%s", len(mismatches), len(claudeNames), strings.Join(mismatches, "\n\n"))
	}
}

// extractBodyAfterFrontmatter returns the content after the second ---
// delimiter in a markdown file. If no frontmatter is found, returns the
// full content unchanged.
func extractBodyAfterFrontmatter(data []byte) string {
	content := string(data)

	// Find opening ---
	start := strings.Index(content, "---")
	if start == -1 {
		return content
	}

	// Find closing --- after the opening
	rest := content[start+3:]
	end := strings.Index(rest, "---")
	if end == -1 {
		return content
	}

	return rest[end+3:]
}

// TestE2EOpenCodeAgentLoad simulates the full downstream flow: copies all 25
// OpenCode agent files to a temp directory (as if aether update placed them
// there), parses each file's YAML frontmatter, and validates it against the
// OpenCode schema. This catches the exact errors that crash OpenCode startup:
// "Invalid input: expected record, received string tools" and "Invalid hex
// color format".
func TestE2EOpenCodeAgentLoad(t *testing.T) {
	repoRoot, err := findOpenCodeRepoRoot()
	if err != nil {
		t.Skipf("repo root not found: %v", err)
	}

	srcDir := filepath.Join(repoRoot, ".opencode", "agents")

	// 1. Setup: create temp directory simulating a downstream repo
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, ".opencode", "agents")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	// 2. Copy: copy all aether-*.md files to temp dir
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatalf("failed to read source agents dir: %v", err)
	}

	var agentFiles []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "aether-") || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(destDir, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("failed to read %s: %v", src, err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			t.Fatalf("failed to write %s: %v", dst, err)
		}
		agentFiles = append(agentFiles, e.Name())
	}

	if len(agentFiles) == 0 {
		t.Fatal("no aether-*.md files found to copy")
	}
	if len(agentFiles) != 26 {
		t.Errorf("expected 26 agent files, found %d", len(agentFiles))
	}

	// 3-4. Parse and validate each file
	hexColorRe := regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	themeColors := map[string]bool{
		"primary": true, "secondary": true, "accent": true,
		"success": true, "warning": true, "error": true, "info": true,
	}

	for _, name := range agentFiles {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(destDir, name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			// Parse YAML frontmatter
			fm, err := extractYAMLFrontmatter(data)
			if err != nil {
				t.Fatalf("YAML parse error (would crash OpenCode): %v", err)
			}

			// Validate: description present and 20+ chars
			desc, ok := fm["description"].(string)
			if !ok || len(strings.TrimSpace(desc)) < 20 {
				t.Errorf("description missing or too short: %q", desc)
			}

			// Validate: mode is valid
			mode, ok := fm["mode"].(string)
			if !ok || (mode != "primary" && mode != "subagent" && mode != "all") {
				t.Errorf("invalid or missing mode: %q", mode)
			}

			// Validate: tools is an object (not a string — the exact bug that crashed OpenCode)
			tools := fm["tools"]
			if tools == nil {
				t.Error("missing tools field")
			} else if _, ok := tools.(map[string]interface{}); !ok {
				t.Errorf("tools is not an object (type %T) — would cause 'expected record, received string' error in OpenCode", tools)
			}

			// Validate: color is hex or theme
			color, ok := fm["color"].(string)
			if !ok || strings.TrimSpace(color) == "" {
				t.Error("missing or empty color field")
			} else if !hexColorRe.MatchString(color) && !themeColors[color] {
				t.Errorf("invalid color %q (must be hex #rrggbb or theme color) — would cause 'Invalid hex color format' error in OpenCode", color)
			}

			// Validate: name field is required
			name, ok := fm["name"].(string)
			if !ok || strings.TrimSpace(name) == "" {
				t.Error("missing or empty name field")
			}

			// Validate: model is optional — OpenCode uses its global default when absent
		})
	}
}

// TestCodexAgentCompleteness verifies that each Codex TOML agent contains
// essential sections expected after Phases 31-33. This test is advisory:
// it logs warnings rather than hard-failing, since Codex agents have
// platform-specific adaptations that may legitimately differ.
func TestCodexAgentCompleteness(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	codexDir := filepath.Join(repoRoot, ".codex", "agents")
	agentNames := listShippedAetherCodexAgentBaseNames(t, codexDir)

	var warnings []string
	for _, name := range agentNames {
		path := filepath.Join(codexDir, name+".toml")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		content := string(data)

		// 1. developer_instructions block exists and is non-empty
		if !strings.Contains(content, "developer_instructions") {
			warnings = append(warnings, fmt.Sprintf("%s: missing developer_instructions block", name))
		} else {
			// Rough check for non-empty: look for closing quote after some content
			idx := strings.Index(content, "developer_instructions")
			after := content[idx:]
			if len(after) < 50 {
				warnings = append(warnings, fmt.Sprintf("%s: developer_instructions appears empty or very short", name))
			}
		}

		// 2. Contains TDD or test-driven references (Phase 31 truth emphasis)
		// Skip for read-only agents that don't implement code (archaeologist, chaos, measurer, etc.)
		lower := strings.ToLower(content)
		readOnlyAgents := map[string]bool{
			"aether-archaeologist": true,
			"aether-chaos":         true,
			"aether-gatekeeper":    true,
			"aether-includer":      true,
			"aether-measurer":      true,
			"aether-oracle":        true,
			"aether-sage":          true,
			"aether-scout":         true,
			"aether-surveyor-disciplines": true,
			"aether-surveyor-nest":        true,
			"aether-surveyor-pathogens":   true,
			"aether-surveyor-provisions":  true,
		}
		// Also skip agents whose core role is not code implementation but who may write files
		// (chronicler, keeper document; queen orchestrates; medic diagnoses)
		nonImplAgents := map[string]bool{
			"aether-chronicler": true,
			"aether-keeper":     true,
			"aether-medic":      true,
			"aether-queen":      true,
		}
		// Also skip agents whose core role is testing/verification rather than implementation
		// (auditor reviews, probe tests, tracker debugs, watcher verifies, weaver refactors)
		testRoleAgents := map[string]bool{
			"aether-auditor": true,
			"aether-probe":   true,
			"aether-tracker": true,
			"aether-watcher": true,
			"aether-weaver":  true,
		}
		if !readOnlyAgents[name] && !nonImplAgents[name] && !testRoleAgents[name] && !strings.Contains(lower, "tdd") && !strings.Contains(lower, "test-driven") {
			warnings = append(warnings, fmt.Sprintf("%s: missing TDD or test-driven references", name))
		}

		// 3. Contains protected or boundary references (safety rules)
		if !strings.Contains(lower, "protected") && !strings.Contains(lower, "boundary") {
			warnings = append(warnings, fmt.Sprintf("%s: missing protected/boundary references", name))
		}

		// 4. Contains escalation references (failure handling)
		// Skip for agents whose core role doesn't involve escalation (gatekeeper, measurer, etc.)
		noEscalationAgents := map[string]bool{
			"aether-gatekeeper": true,
			"aether-measurer":   true,
		}
		if !noEscalationAgents[name] && !strings.Contains(lower, "escalat") {
			warnings = append(warnings, fmt.Sprintf("%s: missing escalation references", name))
		}

		// 5. Deprecated patterns
		if strings.Contains(content, "flag-add") && !strings.Contains(content, "aether flag-add") {
			warnings = append(warnings, fmt.Sprintf("%s: contains deprecated 'flag-add' (use 'aether flag-add')", name))
		}
		if strings.Contains(content, "activity-log") {
			warnings = append(warnings, fmt.Sprintf("%s: contains deprecated 'activity-log' (OpenCode-specific, no Claude equivalent)", name))
		}
	}

	if len(warnings) > 0 {
		t.Logf("Codex agent completeness warnings (%d):\n%s", len(warnings), strings.Join(warnings, "\n"))
	}
}

// findRepoRoot walks up from the current working directory to find the
// directory containing .claude/agents/ant/.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".claude", "agents", "ant")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
