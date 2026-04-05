package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSetupCommandExists verifies the setup command is registered.
func TestSetupCommandExists(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	cmd, _, err := rootCmd.Find([]string{"setup"})
	if err != nil {
		t.Fatalf("setup command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("setup command is nil")
	}
	if cmd.Use != "setup" {
		t.Errorf("setup command Use = %q, want %q", cmd.Use, "setup")
	}
}

// TestSetupFailsWithoutHub verifies setup reports error when no hub directory exists.
func TestSetupFailsWithoutHub(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	var buf bytes.Buffer
	stderr = &buf

	rootCmd.SetArgs([]string{"setup", "--repo-dir", tmpDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	// Command returns nil (error printed to stderr), consistent with install behavior
	_ = err

	output := buf.String()
	if !strings.Contains(output, "hub not installed") && !strings.Contains(output, "hub not found") {
		t.Errorf("expected error about missing hub, got: %s", output)
	}
}

// TestSetupCopiesSystemFiles verifies that setup copies hub system files
// to the local .aether/ directory.
func TestSetupCopiesSystemFiles(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	// Create hub structure: ~/.aether/system/ with a file
	hubSystem := filepath.Join(homeDir, ".aether", "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	// Create hub version marker
	if err := os.WriteFile(filepath.Join(homeDir, ".aether", "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to create hub file: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("setup command failed: %v", err)
	}

	// Verify system file was copied to .aether/
	destFile := filepath.Join(repoDir, ".aether", "workers.md")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Errorf("expected %s to exist after setup", destFile)
	}
}

// TestSetupCreatesRequiredDirs verifies that setup creates data/checkpoints/locks.
func TestSetupCreatesRequiredDirs(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	// Create minimal hub
	hubDir := filepath.Join(homeDir, ".aether")
	if err := os.MkdirAll(hubDir, 0755); err != nil {
		t.Fatalf("failed to create hub dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("setup command failed: %v", err)
	}

	for _, dir := range []string{"data", "checkpoints", "locks"} {
		p := filepath.Join(repoDir, ".aether", dir)
		if info, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected %s to exist after setup", p)
		} else if err == nil && !info.IsDir() {
			t.Errorf("expected %s to be a directory", p)
		}
	}
}

// TestSetupIdempotent verifies running setup twice does not error.
func TestSetupIdempotent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(homeDir, ".aether", "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".aether", "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to create hub file: %v", err)
	}

	// First setup
	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("first setup failed: %v", err)
	}

	// Second setup
	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("second setup failed: %v", err)
	}

	// File should still exist
	destFile := filepath.Join(repoDir, ".aether", "workers.md")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Errorf("expected workers.md to still exist after second setup")
	}
}

// TestSetupPreservesLocalData verifies that existing local files like
// COLONY_STATE.json are not overwritten.
func TestSetupPreservesLocalData(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(homeDir, ".aether", "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".aether", "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}

	// Pre-create local COLONY_STATE.json with user data
	localData := filepath.Join(repoDir, ".aether", "data")
	if err := os.MkdirAll(localData, 0755); err != nil {
		t.Fatalf("failed to create local data dir: %v", err)
	}
	localState := `{"goal":"user goal","state":"ACTIVE"}`
	if err := os.WriteFile(filepath.Join(localData, "COLONY_STATE.json"), []byte(localState), 0644); err != nil {
		t.Fatalf("failed to create local state: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("setup command failed: %v", err)
	}

	// COLONY_STATE.json should be preserved unchanged
	content, err := os.ReadFile(filepath.Join(localData, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("failed to read local state: %v", err)
	}
	if string(content) != localState {
		t.Errorf("local COLONY_STATE.json was overwritten\ngot:  %s\nwant: %s", string(content), localState)
	}
}

// TestSetupOutputJSON verifies the setup command produces valid JSON output.
func TestSetupOutputJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	hubDir := filepath.Join(homeDir, ".aether")
	if err := os.MkdirAll(hubDir, 0755); err != nil {
		t.Fatalf("failed to create hub dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("setup command failed: %v", err)
	}

	output := buf.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("expected valid JSON output, got parse error: %v, output: %s", err, output)
	}
	if ok, exists := result["ok"]; !exists || ok != true {
		t.Errorf("expected JSON output with ok:true, got: %v", result)
	}
}

// TestSetupSkipsUnchangedFiles verifies that identical files are skipped.
func TestSetupSkipsUnchangedFiles(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(homeDir, ".aether", "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".aether", "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}
	content := []byte("# Workers file")
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), content, 0644); err != nil {
		t.Fatalf("failed to create hub file: %v", err)
	}

	// First setup
	var buf1 bytes.Buffer
	stdout = &buf1
	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("first setup failed: %v", err)
	}

	// Second setup - should skip unchanged
	var buf2 bytes.Buffer
	stdout = &buf2
	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("second setup failed: %v", err)
	}

	output := buf2.String()
	if !strings.Contains(output, "skipped") && !strings.Contains(output, "unchanged") {
		t.Errorf("expected output to mention skipped/unchanged files, got: %s", output)
	}
}
