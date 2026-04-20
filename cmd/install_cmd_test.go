package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallCommandExists verifies the install command is registered.
func TestInstallCommandExists(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	cmd, _, err := rootCmd.Find([]string{"install"})
	if err != nil {
		t.Fatalf("install command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("install command is nil")
	}
	if cmd.Use != "install" {
		t.Errorf("install command Use = %q, want %q", cmd.Use, "install")
	}
}

func TestInstallCommandFlags(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	cmd, _, err := rootCmd.Find([]string{"install"})
	if err != nil {
		t.Fatalf("install command not found: %v", err)
	}

	expectedFlags := []string{"package-dir", "home-dir", "download-binary", "binary-dest", "binary-version", "skip-build-binary"}
	for _, name := range expectedFlags {
		if f := cmd.Flags().Lookup(name); f == nil {
			t.Errorf("install command missing flag --%s", name)
		}
	}
}

func TestIsAetherSourceCheckout(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/calcosmic/Aether\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	mainDir := filepath.Join(tmpDir, "cmd", "aether")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("failed to create cmd/aether: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	if !isAetherSourceCheckout(filepath.Join(tmpDir, "cmd")) {
		t.Fatal("expected Aether source checkout to be detected from nested path")
	}
}

func TestIsAetherSourceCheckoutFalseForCompanionOnlyPackage(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".aether"), 0755); err != nil {
		t.Fatalf("failed to create .aether: %v", err)
	}
	if isAetherSourceCheckout(tmpDir) {
		t.Fatal("companion-only package should not be treated as an Aether source checkout")
	}
}

func TestResolveInstallPackageDirRejectsInvalidExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()

	resolved, cleanup, err := resolveInstallPackageDir(tmpDir)
	if err == nil {
		if cleanup != nil {
			cleanup()
		}
		t.Fatalf("expected invalid explicit path to fail, got resolved path %q", resolved)
	}
}

func TestInstallUsesEmbeddedAssetsWithoutPackageDir(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	workDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	codexAgent := filepath.Join(homeDir, ".codex", "agents", "aether-builder.toml")
	if _, err := os.Stat(codexAgent); os.IsNotExist(err) {
		t.Fatalf("expected embedded codex agent %s to exist after install", codexAgent)
	}

	hubWorkers := filepath.Join(homeDir, ".aether", "system", "workers.md")
	if _, err := os.Stat(hubWorkers); os.IsNotExist(err) {
		t.Fatalf("expected embedded hub file %s to exist after install", hubWorkers)
	}
}

// TestInstallCopiesClaudeCommands verifies that install copies .claude/commands/ant/
// files to the target directory.
func TestInstallCopiesClaudeCommands(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	// Set up temp directories
	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".claude", "commands", "ant")
	destDir := filepath.Join(homeDir, ".claude", "commands", "ant")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	// Create a test command file
	if err := os.WriteFile(filepath.Join(srcDir, "test.md"), []byte("# Test command"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	// Verify file was copied
	destFile := filepath.Join(destDir, "test.md")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist after install", destFile)
	}

	// Verify output mentions copied files
	output := buf.String()
	if !strings.Contains(output, "\"copied\":1") {
		t.Errorf("expected output to report 1 copied file, got: %s", output)
	}
}

// TestInstallCopiesClaudeAgents verifies that install copies .claude/agents/ant/
// files to the target directory.
func TestInstallCopiesClaudeAgents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".claude", "agents", "ant")
	destDir := filepath.Join(homeDir, ".claude", "agents", "ant")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "builder.md"), []byte("# Builder agent"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	destFile := filepath.Join(destDir, "builder.md")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist after install", destFile)
	}
}

// TestInstallCopiesOpenCodeCommands verifies OpenCode commands are copied.
func TestInstallCopiesOpenCodeCommands(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".opencode", "commands", "ant")
	destDir := filepath.Join(homeDir, ".opencode", "command")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "init.md"), []byte("# Init command"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	destFile := filepath.Join(destDir, "init.md")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist after install", destFile)
	}
}

// TestInstallCopiesOpenCodeAgents verifies OpenCode agents are copied.
func TestInstallCopiesOpenCodeAgents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".opencode", "agents")
	destDir := filepath.Join(homeDir, ".opencode", "agent")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "builder.md"), []byte("# Builder agent"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	destFile := filepath.Join(destDir, "builder.md")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist after install", destFile)
	}
}

// TestInstallSetsUpHub verifies that install creates ~/.aether/ directory.
func TestInstallSetsUpHub(t *testing.T) {
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

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	hubDir := filepath.Join(homeDir, ".aether")
	info, err := os.Stat(hubDir)
	if os.IsNotExist(err) {
		t.Errorf("expected hub directory %s to exist after install", hubDir)
	}
	if err == nil && !info.IsDir() {
		t.Errorf("expected %s to be a directory", hubDir)
	}
}

// TestInstallIdempotent verifies that running install twice does not error
// and is idempotent.
func TestInstallIdempotent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".claude", "commands", "ant")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "test.md"), []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// First install
	var buf1 bytes.Buffer
	stdout = &buf1
	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("first install failed: %v", err)
	}

	// Second install
	var buf2 bytes.Buffer
	stdout = &buf2
	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("second install failed: %v", err)
	}

	// File should still exist
	destFile := filepath.Join(homeDir, ".claude", "commands", "ant", "test.md")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Errorf("expected file to still exist after second install")
	}
}

// TestInstallSkipsUnchanged verifies that unchanged files are skipped.
func TestInstallSkipsUnchanged(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".claude", "commands", "ant")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	content := []byte("# Test command")
	if err := os.WriteFile(filepath.Join(srcDir, "test.md"), content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// First install
	var buf1 bytes.Buffer
	stdout = &buf1
	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("first install failed: %v", err)
	}

	// Second install - should skip unchanged files
	var buf2 bytes.Buffer
	stdout = &buf2
	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("second install failed: %v", err)
	}

	output := buf2.String()
	if !strings.Contains(output, "skipped") && !strings.Contains(output, "unchanged") {
		t.Errorf("expected output to mention skipped/unchanged files, got: %s", output)
	}
}

// TestInstallRemovesStale verifies that files removed from source are also
// removed from destination.
func TestInstallRemovesStale(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".claude", "commands", "ant")
	destDir := filepath.Join(homeDir, ".claude", "commands", "ant")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	// First install with two files
	if err := os.WriteFile(filepath.Join(srcDir, "keep.md"), []byte("# Keep"), 0644); err != nil {
		t.Fatalf("failed to create keep file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "stale.md"), []byte("# Stale"), 0644); err != nil {
		t.Fatalf("failed to create stale file: %v", err)
	}

	var buf1 bytes.Buffer
	stdout = &buf1
	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("first install failed: %v", err)
	}

	// Remove stale.md from source
	if err := os.Remove(filepath.Join(srcDir, "stale.md")); err != nil {
		t.Fatalf("failed to remove stale file: %v", err)
	}

	// Second install
	var buf2 bytes.Buffer
	stdout = &buf2
	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("second install failed: %v", err)
	}

	// keep.md should still exist
	if _, err := os.Stat(filepath.Join(destDir, "keep.md")); os.IsNotExist(err) {
		t.Errorf("expected keep.md to still exist")
	}

	// stale.md should be removed
	if _, err := os.Stat(filepath.Join(destDir, "stale.md")); err == nil {
		t.Errorf("expected stale.md to be removed")
	}
}

// TestInstallOutputJSON verifies the install command produces valid JSON output.
func TestInstallOutputJSON(t *testing.T) {
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

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	output := buf.String()
	// Should be valid JSON with ok:true
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("expected valid JSON output, got parse error: %v, output: %s", err, output)
	}
	if ok, exists := result["ok"]; !exists || ok != true {
		t.Errorf("expected JSON output with ok:true, got: %v", result)
	}
}

// TestInstallSkipsMissingSource verifies that install does not error when
// a source directory doesn't exist.
func TestInstallSkipsMissingSource(t *testing.T) {
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
	// Leave all other source directories absent.

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command should not fail with missing sources: %v", err)
	}
}

// TestInstallWithSubdirs verifies that nested directory structures are preserved.
func TestInstallWithSubdirs(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".claude", "commands", "ant", "subdir")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "nested.md"), []byte("# Nested"), 0644); err != nil {
		t.Fatalf("failed to create nested file: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	destFile := filepath.Join(homeDir, ".claude", "commands", "ant", "subdir", "nested.md")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Errorf("expected nested file %s to exist", destFile)
	}
}

// TestInstallCopiesCodexAgents verifies that install copies .codex/agents/
// files to the target directory.
func TestInstallCopiesCodexAgents(t *testing.T) {
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
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	destFile := filepath.Join(destDir, "aether-builder.toml")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist after install", destFile)
	}
}

// TestInstallCopiesCodexAgentsToHub verifies that .codex/ files are synced
// to the hub at ~/.aether/system/codex/.
func TestInstallCopiesCodexAgentsToHub(t *testing.T) {
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
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	hubCodexFile := filepath.Join(homeDir, ".aether", "system", "codex", "aether-builder.toml")
	if _, err := os.Stat(hubCodexFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist after install", hubCodexFile)
	}
}

// TestInstallShellScriptsGetExecutable verifies that .sh files get chmod 0755.
func TestInstallShellScriptsGetExecutable(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, ".claude", "commands", "ant")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "script.sh"), []byte("#!/bin/bash\necho hi"), 0644); err != nil {
		t.Fatalf("failed to create shell script: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	destFile := filepath.Join(homeDir, ".claude", "commands", "ant", "script.sh")
	info, err := os.Stat(destFile)
	if err != nil {
		t.Fatalf("failed to stat dest file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0111 == 0 {
		t.Errorf("expected .sh file to be executable, got permissions %o", perm)
	}
}
