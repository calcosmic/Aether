package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
