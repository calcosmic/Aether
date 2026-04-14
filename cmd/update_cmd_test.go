package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUpdateCommandExists verifies the update command is registered.
func TestUpdateCommandExists(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	cmd, _, err := rootCmd.Find([]string{"update"})
	if err != nil {
		t.Fatalf("update command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("update command is nil")
	}
	if cmd.Use != "update" {
		t.Errorf("update command Use = %q, want %q", cmd.Use, "update")
	}
}

// TestUpdateCommandFlags verifies the update command has the expected flags.
func TestUpdateCommandFlags(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	cmd, _, err := rootCmd.Find([]string{"update"})
	if err != nil {
		t.Fatalf("update command not found: %v", err)
	}

	expectedFlags := []string{"dry-run", "force", "download-binary", "binary-version"}
	for _, name := range expectedFlags {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("update command missing flag --%s", name)
		}
	}
}

// TestUpdateDryRunOutput verifies dry-run mode produces valid JSON without
// modifying any files on disk.
func TestUpdateDryRunOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()

	// Create minimal hub structure
	hubDir := filepath.Join(homeDir, ".aether")
	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}

	// Create a repo dir with existing companion file (should NOT be modified)
	repoDir := t.TempDir()
	localAether := filepath.Join(repoDir, ".aether")
	if err := os.MkdirAll(localAether, 0755); err != nil {
		t.Fatalf("failed to create local .aether dir: %v", err)
	}
	originalContent := []byte("# Original workers file")
	if err := os.WriteFile(filepath.Join(localAether, "workers.md"), originalContent, 0644); err != nil {
		t.Fatalf("failed to create local workers.md: %v", err)
	}

	// Create a newer version in hub
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), []byte("# New workers file"), 0644); err != nil {
		t.Fatalf("failed to create hub workers.md: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	// Set HOME and chdir to our temp dirs
	t.Setenv("HOME", homeDir)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer os.Chdir(oldDir)

	rootCmd.SetArgs([]string{"update", "--dry-run"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("update dry-run failed: %v", err)
	}

	output := buf.String()

	// Verify valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("expected valid JSON output, got parse error: %v, output: %s", err, output)
	}

	// Verify dry-run message
	message, _ := result["result"].(map[string]interface{})["message"].(string)
	if !strings.Contains(message, "Dry run") {
		t.Errorf("expected dry-run message, got: %s", message)
	}

	// Verify no files were actually modified
	content, err := os.ReadFile(filepath.Join(localAether, "workers.md"))
	if err != nil {
		t.Fatalf("failed to read local workers.md: %v", err)
	}
	if string(content) != string(originalContent) {
		t.Errorf("dry-run modified local file\ngot:  %s\nwant: %s", string(content), string(originalContent))
	}
}

// TestUpdateDryRunWithForce verifies dry-run + force shows force mode description.
func TestUpdateDryRunWithForce(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()

	hubDir := filepath.Join(homeDir, ".aether")
	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}

	repoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoDir, ".aether"), 0755); err != nil {
		t.Fatalf("failed to create local .aether dir: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	t.Setenv("HOME", homeDir)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer os.Chdir(oldDir)

	rootCmd.SetArgs([]string{"update", "--dry-run", "--force"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("update dry-run --force failed: %v", err)
	}

	output := buf.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("expected valid JSON output, got parse error: %v, output: %s", err, output)
	}

	// The force field should be true in the result
	inner, _ := result["result"].(map[string]interface{})
	if forceVal, ok := inner["force"]; !ok || forceVal != true {
		t.Errorf("expected force=true in dry-run output, got: %v", inner["force"])
	}
}

// TestUpdateFailsWithoutHub verifies update reports error when no hub exists.
func TestUpdateFailsWithoutHub(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	var buf bytes.Buffer
	stderr = &buf

	t.Setenv("HOME", homeDir)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer os.Chdir(oldDir)

	rootCmd.SetArgs([]string{"update"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	// Command returns nil (error printed to stderr)
	_ = err

	output := buf.String()
	if !strings.Contains(output, "hub not installed") {
		t.Errorf("expected error about missing hub, got: %s", output)
	}
}

// --- Unit tests for runUpdateSync and sync functions ---

// TestRunUpdateSyncOverwritesChanged verifies that normal update overwrites
// changed companion files from hub to local.
func TestUpdateSyncOverwritesChanged(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	// Create hub system with a file
	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system: %v", err)
	}
	hubContent := []byte("# Updated workers v2")
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), hubContent, 0644); err != nil {
		t.Fatalf("failed to write hub file: %v", err)
	}

	// Create local .aether/ with an older version
	localAether := filepath.Join(repoDir, ".aether")
	if err := os.MkdirAll(localAether, 0755); err != nil {
		t.Fatalf("failed to create local .aether: %v", err)
	}
	oldContent := []byte("# Old workers v1")
	if err := os.WriteFile(filepath.Join(localAether, "workers.md"), oldContent, 0644); err != nil {
		t.Fatalf("failed to write local file: %v", err)
	}

	// Run sync (normal mode, force=false)
	result := runUpdateSync(hubDir, repoDir, false)

	if result.copied < 1 {
		t.Errorf("expected at least 1 file copied, got %d", result.copied)
	}

	// Verify local file was updated
	content, err := os.ReadFile(filepath.Join(localAether, "workers.md"))
	if err != nil {
		t.Fatalf("failed to read local file: %v", err)
	}
	if string(content) != string(hubContent) {
		t.Errorf("local file not updated\ngot:  %s\nwant: %s", string(content), string(hubContent))
	}
}

// TestRunUpdateSyncSkipsUnchanged verifies that identical files are skipped.
func TestUpdateSyncSkipsUnchanged(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system: %v", err)
	}
	content := []byte("# Same workers file")
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), content, 0644); err != nil {
		t.Fatalf("failed to write hub file: %v", err)
	}

	localAether := filepath.Join(repoDir, ".aether")
	if err := os.MkdirAll(localAether, 0755); err != nil {
		t.Fatalf("failed to create local .aether: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localAether, "workers.md"), content, 0644); err != nil {
		t.Fatalf("failed to write local file: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, false)

	if result.copied != 0 {
		t.Errorf("expected 0 files copied (all unchanged), got %d", result.copied)
	}
	if result.skipped < 1 {
		t.Errorf("expected at least 1 file skipped, got %d", result.skipped)
	}
}

// TestRunUpdateSyncNewFilesAreCopied verifies that new files in hub are copied
// to local (not just overwrites of existing files).
func TestUpdateSyncNewFilesAreCopied(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system: %v", err)
	}
	newContent := []byte("# Brand new file")
	if err := os.WriteFile(filepath.Join(hubSystem, "newfile.md"), newContent, 0644); err != nil {
		t.Fatalf("failed to write hub file: %v", err)
	}

	localAether := filepath.Join(repoDir, ".aether")
	if err := os.MkdirAll(localAether, 0755); err != nil {
		t.Fatalf("failed to create local .aether: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, false)

	if result.copied < 1 {
		t.Errorf("expected at least 1 file copied, got %d", result.copied)
	}

	// Verify new file exists in local
	destFile := filepath.Join(localAether, "newfile.md")
	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("expected new file to exist: %v", err)
	}
	if string(content) != string(newContent) {
		t.Errorf("new file content mismatch\ngot:  %s\nwant: %s", string(content), string(newContent))
	}
}

// TestRunUpdateSyncPreservesProtectedDirs verifies that data/ and dreams/
// directories are never overwritten.
func TestUpdateSyncPreservesProtectedDirs(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(hubDir, "system")
	hubData := filepath.Join(hubSystem, "data")
	if err := os.MkdirAll(hubData, 0755); err != nil {
		t.Fatalf("failed to create hub data dir: %v", err)
	}

	// Hub has a data file that differs from local
	hubDataContent := []byte(`{"goal":"hub_goal","state":"HUB"}`)
	if err := os.WriteFile(filepath.Join(hubData, "COLONY_STATE.json"), hubDataContent, 0644); err != nil {
		t.Fatalf("failed to write hub data file: %v", err)
	}

	localAether := filepath.Join(repoDir, ".aether")
	localData := filepath.Join(localAether, "data")
	if err := os.MkdirAll(localData, 0755); err != nil {
		t.Fatalf("failed to create local data dir: %v", err)
	}

	localDataContent := []byte(`{"goal":"user_goal","state":"ACTIVE"}`)
	if err := os.WriteFile(filepath.Join(localData, "COLONY_STATE.json"), localDataContent, 0644); err != nil {
		t.Fatalf("failed to write local data file: %v", err)
	}

	// Sync with force=true -- protected dirs should still be safe
	result := runUpdateSync(hubDir, repoDir, true)
	_ = result

	// The data dir should have been skipped, not copied
	content, err := os.ReadFile(filepath.Join(localData, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("failed to read local data file: %v", err)
	}
	if string(content) != string(localDataContent) {
		t.Errorf("protected data/ was overwritten\ngot:  %s\nwant: %s", string(content), string(localDataContent))
	}
}

// TestRunUpdateSyncPreservesProtectedFiles verifies that QUEEN.md and
// CROWNED-ANTHILL.md are never overwritten.
func TestUpdateSyncPreservesProtectedFiles(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system: %v", err)
	}

	// Hub has a QUEEN.md
	hubQueen := []byte("# Hub QUEEN")
	if err := os.WriteFile(filepath.Join(hubSystem, "QUEEN.md"), hubQueen, 0644); err != nil {
		t.Fatalf("failed to write hub QUEEN.md: %v", err)
	}

	localAether := filepath.Join(repoDir, ".aether")
	if err := os.MkdirAll(localAether, 0755); err != nil {
		t.Fatalf("failed to create local .aether: %v", err)
	}

	localQueen := []byte("# Local QUEEN - do not overwrite")
	if err := os.WriteFile(filepath.Join(localAether, "QUEEN.md"), localQueen, 0644); err != nil {
		t.Fatalf("failed to write local QUEEN.md: %v", err)
	}

	// Sync with force=true
	result := runUpdateSync(hubDir, repoDir, true)
	_ = result

	content, err := os.ReadFile(filepath.Join(localAether, "QUEEN.md"))
	if err != nil {
		t.Fatalf("failed to read local QUEEN.md: %v", err)
	}
	if string(content) != string(localQueen) {
		t.Errorf("protected QUEEN.md was overwritten\ngot:  %s\nwant: %s", string(content), string(localQueen))
	}
}

// TestRunUpdateSyncPreservesDreamsDir verifies that dreams/ directory
// content is never overwritten even in force mode.
func TestUpdateSyncPreservesDreamsDir(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(hubDir, "system")
	hubDreams := filepath.Join(hubSystem, "dreams")
	if err := os.MkdirAll(hubDreams, 0755); err != nil {
		t.Fatalf("failed to create hub dreams dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDreams, "dream1.md"), []byte("# Hub dream"), 0644); err != nil {
		t.Fatalf("failed to write hub dream file: %v", err)
	}

	localAether := filepath.Join(repoDir, ".aether")
	localDreams := filepath.Join(localAether, "dreams")
	if err := os.MkdirAll(localDreams, 0755); err != nil {
		t.Fatalf("failed to create local dreams dir: %v", err)
	}
	localDream := []byte("# My personal dream")
	if err := os.WriteFile(filepath.Join(localDreams, "dream1.md"), localDream, 0644); err != nil {
		t.Fatalf("failed to write local dream file: %v", err)
	}

	// Sync with force
	result := runUpdateSync(hubDir, repoDir, true)
	_ = result

	content, err := os.ReadFile(filepath.Join(localDreams, "dream1.md"))
	if err != nil {
		t.Fatalf("failed to read local dream file: %v", err)
	}
	if string(content) != string(localDream) {
		t.Errorf("protected dreams/ was overwritten\ngot:  %s\nwant: %s", string(content), string(localDream))
	}
}

// TestRunUpdateSyncForceRemovesStale verifies that force mode removes files
// from local that no longer exist in hub.
func TestUpdateSyncForceRemovesStale(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system: %v", err)
	}
	// Hub only has keep.md
	if err := os.WriteFile(filepath.Join(hubSystem, "keep.md"), []byte("# Keep"), 0644); err != nil {
		t.Fatalf("failed to write keep.md: %v", err)
	}

	localAether := filepath.Join(repoDir, ".aether")
	if err := os.MkdirAll(localAether, 0755); err != nil {
		t.Fatalf("failed to create local .aether: %v", err)
	}
	// Local has both keep.md and stale.md
	if err := os.WriteFile(filepath.Join(localAether, "keep.md"), []byte("# Keep"), 0644); err != nil {
		t.Fatalf("failed to write keep.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localAether, "stale.md"), []byte("# Stale"), 0644); err != nil {
		t.Fatalf("failed to write stale.md: %v", err)
	}

	// Force sync should remove stale.md
	result := runUpdateSync(hubDir, repoDir, true)
	_ = result

	// keep.md should still exist
	if _, err := os.Stat(filepath.Join(localAether, "keep.md")); os.IsNotExist(err) {
		t.Error("keep.md should still exist after force sync")
	}

	// stale.md should be removed
	if _, err := os.Stat(filepath.Join(localAether, "stale.md")); err == nil {
		t.Error("stale.md should be removed after force sync")
	}
}

// TestRunUpdateSyncNormalKeepsStale verifies that normal (non-force) mode does
// NOT remove stale files from local.
func TestUpdateSyncNormalKeepsStale(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system: %v", err)
	}
	// Hub only has keep.md
	if err := os.WriteFile(filepath.Join(hubSystem, "keep.md"), []byte("# Keep"), 0644); err != nil {
		t.Fatalf("failed to write keep.md: %v", err)
	}

	localAether := filepath.Join(repoDir, ".aether")
	if err := os.MkdirAll(localAether, 0755); err != nil {
		t.Fatalf("failed to create local .aether: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localAether, "keep.md"), []byte("# Keep"), 0644); err != nil {
		t.Fatalf("failed to write keep.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localAether, "stale.md"), []byte("# Stale"), 0644); err != nil {
		t.Fatalf("failed to write stale.md: %v", err)
	}

	// Normal sync should NOT remove stale.md
	result := runUpdateSync(hubDir, repoDir, false)
	_ = result

	// stale.md should still exist in normal mode
	if _, err := os.Stat(filepath.Join(localAether, "stale.md")); os.IsNotExist(err) {
		t.Error("stale.md should still exist in normal (non-force) mode")
	}
}

// TestRunUpdateSyncHandlesMissingSource verifies that missing hub system
// directories are handled gracefully.
func TestUpdateSyncHandlesMissingSource(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	// No system/ directory in hub -- should not error
	localAether := filepath.Join(repoDir, ".aether")
	if err := os.MkdirAll(localAether, 0755); err != nil {
		t.Fatalf("failed to create local .aether: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, false)

	if result.copied != 0 {
		t.Errorf("expected 0 files copied with missing source, got %d", result.copied)
	}
}

// TestRunUpdateSyncOutputJSON verifies the update command produces valid JSON
// with the expected structure when run for real (not dry-run).
func TestUpdateSyncOutputJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	// Create minimal hub
	hubDir := filepath.Join(homeDir, ".aether")
	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to write hub workers.md: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	t.Setenv("HOME", homeDir)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer os.Chdir(oldDir)

	rootCmd.SetArgs([]string{"update"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("update command failed: %v", err)
	}

	output := buf.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("expected valid JSON output, got parse error: %v, output: %s", err, output)
	}
	if ok, exists := result["ok"]; !exists || ok != true {
		t.Errorf("expected JSON output with ok:true, got: %v", result)
	}

	// Verify the inner result has expected fields
	inner, ok := result["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result to contain a 'result' map")
	}
	if _, has := inner["message"]; !has {
		t.Error("expected result to contain 'message' field")
	}
	if _, has := inner["details"]; !has {
		t.Error("expected result to contain 'details' field")
	}
}

// TestRunUpdateSyncShellScriptsGetExecutable verifies that .sh files get
// executable permissions during update.
func TestUpdateSyncShellScriptsGetExecutable(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "script.sh"), []byte("#!/bin/bash\necho hi"), 0644); err != nil {
		t.Fatalf("failed to write hub script: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, false)
	if result.copied < 1 {
		t.Errorf("expected at least 1 file copied, got %d", result.copied)
	}

	destFile := filepath.Join(repoDir, ".aether", "script.sh")
	info, err := os.Stat(destFile)
	if err != nil {
		t.Fatalf("failed to stat dest file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0111 == 0 {
		t.Errorf("expected .sh file to be executable, got permissions %o", perm)
	}
}

// TestRunUpdateSyncNestedDirs verifies that nested directory structures
// are preserved during sync.
func TestUpdateSyncNestedDirs(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(hubDir, "system")
	nestedDir := filepath.Join(hubSystem, "commands", "claude", "subdir")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	content := []byte("# Nested command")
	if err := os.WriteFile(filepath.Join(nestedDir, "nested.md"), content, 0644); err != nil {
		t.Fatalf("failed to write nested file: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, false)
	if result.copied < 1 {
		t.Errorf("expected at least 1 file copied, got %d", result.copied)
	}

	destFile := filepath.Join(repoDir, ".claude", "commands", "ant", "subdir", "nested.md")
	destContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("expected nested file to exist: %v", err)
	}
	if string(destContent) != string(content) {
		t.Errorf("nested file content mismatch\ngot:  %s\nwant: %s", string(destContent), string(content))
	}
}

// TestRunUpdateSyncForceDoesNotRemoveProtectedStale verifies that force mode
// does NOT remove stale files inside protected directories.
func TestUpdateSyncForceDoesNotRemoveProtectedStale(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	// Hub has no data/ directory at all
	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to write hub file: %v", err)
	}

	// Local has a data/ directory with user files
	localAether := filepath.Join(repoDir, ".aether")
	localData := filepath.Join(localAether, "data")
	if err := os.MkdirAll(localData, 0755); err != nil {
		t.Fatalf("failed to create local data dir: %v", err)
	}
	userFile := []byte(`{"user":"data"}`)
	if err := os.WriteFile(filepath.Join(localData, "user.json"), userFile, 0644); err != nil {
		t.Fatalf("failed to write local user file: %v", err)
	}

	// Force sync should NOT remove files in protected dirs
	result := runUpdateSync(hubDir, repoDir, true)
	_ = result

	content, err := os.ReadFile(filepath.Join(localData, "user.json"))
	if err != nil {
		t.Fatalf("protected user file was removed: %v", err)
	}
	if string(content) != string(userFile) {
		t.Errorf("protected user file was modified\ngot:  %s\nwant: %s", string(content), string(userFile))
	}
}

// TestUpdateSyncCodexPair verifies the Codex sync pair is present and functional.
func TestUpdateSyncCodexPair(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(hubDir, "system")
	codexDir := filepath.Join(hubSystem, "codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		t.Fatalf("failed to create codex dir: %v", err)
	}
	codexContent := []byte("# Codex agent file")
	if err := os.WriteFile(filepath.Join(codexDir, "agent.md"), codexContent, 0644); err != nil {
		t.Fatalf("failed to write codex agent: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, false)

	// Verify Codex sync pair produced a detail entry
	foundCodex := false
	for _, detail := range result.details {
		if label, ok := detail["label"].(string); ok && label == "Agents (codex)" {
			foundCodex = true
			break
		}
	}
	if !foundCodex {
		t.Error("expected sync details to include 'Agents (codex)' label")
	}

	// Verify the file was copied to the correct destination
	destFile := filepath.Join(repoDir, ".codex", "agents", "agent.md")
	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("expected codex agent file at %s: %v", destFile, err)
	}
	if string(content) != string(codexContent) {
		t.Errorf("codex agent content mismatch\ngot:  %s\nwant: %s", string(content), string(codexContent))
	}
}

// TestUpdateDryRunIncludesCodexAction verifies dry-run output mentions .codex/agents/.
func TestUpdateDryRunIncludesCodexAction(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()

	hubDir := filepath.Join(homeDir, ".aether")
	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}

	repoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoDir, ".aether"), 0755); err != nil {
		t.Fatalf("failed to create local .aether dir: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	t.Setenv("HOME", homeDir)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer os.Chdir(oldDir)

	rootCmd.SetArgs([]string{"update", "--dry-run"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("update dry-run failed: %v", err)
	}

	output := buf.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("expected valid JSON output: %v, output: %s", err, output)
	}

	inner, _ := result["result"].(map[string]interface{})
	actions, ok := inner["actions"].([]interface{})
	if !ok {
		t.Fatal("expected actions to be an array")
	}

	foundCodexAction := false
	for _, action := range actions {
		if actionStr, ok := action.(string); ok && strings.Contains(actionStr, ".codex/agents") {
			foundCodexAction = true
			break
		}
	}
	if !foundCodexAction {
		t.Errorf("expected dry-run actions to include '.codex/agents/', got: %v", actions)
	}
}

// TestRunUpdateSyncMultipleSyncPairs verifies that all sync pairs (commands,
// agents, rules) are processed.
func TestUpdateSyncMultipleSyncPairs(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(hubDir, "system")

	// Create files for multiple sync pairs
	syncDirs := map[string]string{
		"commands/claude": "../.claude/commands/ant",
		"commands/opencode": "../.opencode/commands/ant",
		"agents": "../.opencode/agents",
		"agents-claude": "../.claude/agents/ant",
		"rules": "../.claude/rules",
	}

	for hubRel, _ := range syncDirs {
		dir := filepath.Join(hubSystem, filepath.FromSlash(hubRel))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", hubRel, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "test.md"), []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to write test file in %s: %v", hubRel, err)
		}
	}

	// Also create a system file (the "." sync pair)
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, false)

	// Verify details were collected for each sync pair
	if len(result.details) == 0 {
		t.Error("expected sync details for each pair, got none")
	}

	// Total copied should be at least the number of sync pairs (1 file each)
	if result.copied < len(syncDirs)+1 {
		t.Errorf("expected at least %d files copied, got %d", len(syncDirs)+1, result.copied)
	}

	// Verify specific destinations exist
	expectedDests := []string{
		filepath.Join(repoDir, ".aether", "workers.md"),
		filepath.Join(repoDir, ".claude", "commands", "ant", "test.md"),
		filepath.Join(repoDir, ".claude", "agents", "ant", "test.md"),
		filepath.Join(repoDir, ".claude", "rules", "test.md"),
	}
	for _, dest := range expectedDests {
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			t.Errorf("expected %s to exist after sync", dest)
		}
	}
}
