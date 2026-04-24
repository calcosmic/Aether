package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	expectedFlags := []string{"channel", "dry-run", "force", "download-binary", "binary-version"}
	for _, name := range expectedFlags {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("update command missing flag --%s", name)
		}
	}
}

func TestUpdateUsesDevHubWhenChannelIsDev(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether-dev")
	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystem, 0755); err != nil {
		t.Fatalf("failed to create dev hub system: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.19"}`), 0644); err != nil {
		t.Fatalf("failed to write dev hub version: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), []byte("# Dev Workers"), 0644); err != nil {
		t.Fatalf("failed to write dev hub workers.md: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"update", "--channel", "dev"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update command failed: %v", err)
	}

	repoWorkers := filepath.Join(repoDir, ".aether", "workers.md")
	content, err := os.ReadFile(repoWorkers)
	if err != nil {
		t.Fatalf("expected repo workers.md after dev update: %v", err)
	}
	if string(content) != "# Dev Workers" {
		t.Fatalf("repo workers.md = %q, want dev hub content", string(content))
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
	if mode, _ := inner["binary_refresh_mode"].(string); mode != "unchanged" {
		t.Errorf("expected binary_refresh_mode=unchanged, got: %v", inner["binary_refresh_mode"])
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

func TestUpdateRefreshesManagedCodexProjectDocs(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	hubDir := filepath.Join(homeDir, ".aether")
	hubSystem := filepath.Join(hubDir, "system")
	hubTemplates := filepath.Join(hubSystem, "templates")
	if err := os.MkdirAll(hubTemplates, 0755); err != nil {
		t.Fatalf("failed to create hub templates dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.5"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubTemplates, "agents-md-template.md"), []byte("# {COLONY_NAME}\n\n> Aether Colony -- Codex CLI Instructions\n\nUse `AETHER_OUTPUT_MODE=visual aether continue`.\n\n*Generated by Aether setup. Customize this file for your project.*\n"), 0644); err != nil {
		t.Fatalf("failed to create AGENTS template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubTemplates, "codex-md-template.md"), []byte("# CODEX.md -- Aether Codex Workflow Guide\n\nRun `AETHER_OUTPUT_MODE=visual aether build 1`.\n\n*Generated by Aether setup. Customize this file for your project.*\n"), 0644); err != nil {
		t.Fatalf("failed to create CODEX template: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(repoDir, ".aether"), 0755); err != nil {
		t.Fatalf("failed to create local .aether: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoDir, ".codex"), 0755); err != nil {
		t.Fatalf("failed to create local .codex: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "AGENTS.md"), []byte("# Old\n\n> Aether Colony -- Codex CLI Instructions\n"), 0644); err != nil {
		t.Fatalf("failed to seed AGENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ".codex", "CODEX.md"), []byte("# CODEX.md -- Aether Codex Workflow Guide\n\nOld content\n"), 0644); err != nil {
		t.Fatalf("failed to seed .codex/CODEX.md: %v", err)
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

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected JSON output, got parse error: %v, output: %s", err, buf.String())
	}
	inner, _ := result["result"].(map[string]interface{})
	if got, _ := inner["codex_restart_required"].(bool); !got {
		t.Fatalf("expected codex_restart_required=true, got: %#v", inner["codex_restart_required"])
	}
	if mode, _ := inner["binary_refresh_mode"].(string); mode != "unchanged" {
		t.Fatalf("expected binary_refresh_mode=unchanged, got: %#v", inner["binary_refresh_mode"])
	}
	note, _ := inner["binary_refresh_note"].(string)
	if !strings.Contains(note, "only syncs repo companion files, not the shared binary") {
		t.Fatalf("expected runtime note in update result, got: %q", note)
	}
	message, _ := inner["message"].(string)
	if !strings.Contains(message, "Close this Codex chat and start a new one in this repo") {
		t.Fatalf("expected restart guidance in update message, got: %q", message)
	}

	agentsContent, err := os.ReadFile(filepath.Join(repoDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agentsContent), "AETHER_OUTPUT_MODE=visual aether continue") {
		t.Fatalf("expected refreshed AGENTS.md, got:\n%s", string(agentsContent))
	}

	codexContent, err := os.ReadFile(filepath.Join(repoDir, ".codex", "CODEX.md"))
	if err != nil {
		t.Fatalf("failed to read .codex/CODEX.md: %v", err)
	}
	if !strings.Contains(string(codexContent), "AETHER_OUTPUT_MODE=visual aether build 1") {
		t.Fatalf("expected refreshed .codex/CODEX.md, got:\n%s", string(codexContent))
	}
}

func TestUpdatePreservesCustomAgentsMD(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	hubDir := filepath.Join(homeDir, ".aether")
	hubSystem := filepath.Join(hubDir, "system")
	hubTemplates := filepath.Join(hubSystem, "templates")
	if err := os.MkdirAll(hubTemplates, 0755); err != nil {
		t.Fatalf("failed to create hub templates dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.5"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubTemplates, "agents-md-template.md"), []byte("# Managed template\n\n> Aether Colony -- Codex CLI Instructions\n"), 0644); err != nil {
		t.Fatalf("failed to create AGENTS template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubTemplates, "codex-md-template.md"), []byte("# CODEX.md -- Aether Codex Workflow Guide\n"), 0644); err != nil {
		t.Fatalf("failed to create CODEX template: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(repoDir, ".aether"), 0755); err != nil {
		t.Fatalf("failed to create local .aether: %v", err)
	}
	customAgents := "# My Project\n\nCustom AGENTS instructions\n"
	if err := os.WriteFile(filepath.Join(repoDir, "AGENTS.md"), []byte(customAgents), 0644); err != nil {
		t.Fatalf("failed to seed custom AGENTS.md: %v", err)
	}

	t.Setenv("HOME", homeDir)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer os.Chdir(oldDir)

	rootCmd.SetArgs([]string{"update"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	agentsContent, err := os.ReadFile(filepath.Join(repoDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}
	if string(agentsContent) != customAgents {
		t.Fatalf("custom AGENTS.md should be preserved\ngot:\n%s\nwant:\n%s", string(agentsContent), customAgents)
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
	codexContent := validCodexAgentTOML("aether-builder", "builder")
	if err := os.WriteFile(filepath.Join(codexDir, "aether-builder.toml"), codexContent, 0644); err != nil {
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
	destFile := filepath.Join(repoDir, ".codex", "agents", "aether-builder.toml")
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

// --- Stale-publish integration tests ---

// createMinimalHub creates a hub with only files that don't trigger sync validation.
// Used for integration tests that need the sync to succeed so stale detection can run.
func createMinimalHub(t *testing.T, hubDir string) {
	t.Helper()
	system := filepath.Join(hubDir, "system")

	// Only create directories that have no sync validation
	dirs := map[string]int{
		"commands/claude":   expectedClaudeCommandCount,
		"commands/opencode": expectedOpenCodeCommandCount,
		"skills-codex":      expectedCodexSkillCount,
	}
	for rel, count := range dirs {
		dir := filepath.Join(system, filepath.FromSlash(rel))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", rel, err)
		}
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("file_%02d.md", i)
			if err := os.WriteFile(filepath.Join(dir, name), []byte("# test"), 0644); err != nil {
				t.Fatalf("failed to write %s: %v", name, err)
			}
		}
	}
}

func TestUpdateDetectsCriticalStale(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createMinimalHub(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.19"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"update", "--force"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for critical stale, got nil")
	}
	if !strings.Contains(err.Error(), "stale publish detected") {
		t.Errorf("expected error to contain 'stale publish detected', got: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON output: %v, output: %s", err, buf.String())
	}
	inner, _ := result["result"].(map[string]interface{})
	stale, _ := inner["stale_publish"].(map[string]interface{})
	if stale["classification"] != "critical" {
		t.Errorf("expected classification=critical, got: %v", stale["classification"])
	}
}

func TestUpdateDetectsWarningStale(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createMinimalHub(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.21"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"update"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for warning stale, got: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON output: %v, output: %s", err, buf.String())
	}
	inner, _ := result["result"].(map[string]interface{})
	stale, _ := inner["stale_publish"].(map[string]interface{})
	if stale["classification"] != "warning" {
		t.Errorf("expected classification=warning, got: %v", stale["classification"])
	}
}

func TestUpdateDetectsInfoStaleMissingFiles(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createMinimalHub(t, hubDir)
	// Empty claude commands
	os.RemoveAll(filepath.Join(hubDir, "system", "commands", "claude"))
	os.MkdirAll(filepath.Join(hubDir, "system", "commands", "claude"), 0755)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.20"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"update"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for info stale, got: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON output: %v, output: %s", err, buf.String())
	}
	inner, _ := result["result"].(map[string]interface{})
	stale, _ := inner["stale_publish"].(map[string]interface{})
	if stale["classification"] != "info" {
		t.Errorf("expected classification=info, got: %v", stale["classification"])
	}
	components, _ := stale["components"].([]interface{})
	foundClaude := false
	for _, c := range components {
		comp, _ := c.(map[string]interface{})
		if strings.Contains(comp["name"].(string), "claude") {
			foundClaude = true
		}
	}
	if !foundClaude {
		t.Errorf("expected components to contain claude entry, got: %v", components)
	}
}

func TestUpdateDryRunDetectsStale(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createMinimalHub(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.19"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"update", "--dry-run"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for dry-run critical stale, got nil")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON output: %v, output: %s", err, buf.String())
	}
	inner, _ := result["result"].(map[string]interface{})
	stale, _ := inner["stale_publish"].(map[string]interface{})
	if stale["classification"] != "critical" {
		t.Errorf("expected classification=critical, got: %v", stale["classification"])
	}
}

func TestUpdateForceDoesNotSuppressStale(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createMinimalHub(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.19"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"update", "--force"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for critical stale despite --force, got nil")
	}
	if !strings.Contains(err.Error(), "stale publish detected") {
		t.Errorf("expected stale publish error, got: %v", err)
	}
}

func TestUpdateDevChannelDetectsStale(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether-dev")
	createMinimalHub(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.19"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"update", "--force", "--channel", "dev"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for dev channel critical stale, got nil")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON output: %v, output: %s", err, buf.String())
	}
	inner, _ := result["result"].(map[string]interface{})
	stale, _ := inner["stale_publish"].(map[string]interface{})
	if stale["classification"] != "critical" {
		t.Errorf("expected classification=critical, got: %v", stale["classification"])
	}
	if stale["channel"] != "dev" {
		t.Errorf("expected channel=dev, got: %v", stale["channel"])
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
		"commands/claude":   "../.claude/commands/ant",
		"commands/opencode": "../.opencode/commands/ant",
		"agents":            "../.opencode/agents",
		"agents-claude":     "../.claude/agents/ant",
		"rules":             "../.claude/rules",
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

func TestRunUpdateSyncForceSyncsClaudeSettings(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSettings := filepath.Join(hubDir, "system", "settings", "claude")
	if err := os.MkdirAll(hubSettings, 0755); err != nil {
		t.Fatalf("failed to create hub settings dir: %v", err)
	}
	hubContent := []byte("{\n  \"env\": {\n    \"AETHER_ACTIVE_PLATFORM\": \"claude\"\n  },\n  \"hooks\": {}\n}\n")
	if err := os.WriteFile(filepath.Join(hubSettings, "settings.json"), hubContent, 0644); err != nil {
		t.Fatalf("failed to write hub Claude settings: %v", err)
	}

	localClaude := filepath.Join(repoDir, ".claude")
	if err := os.MkdirAll(localClaude, 0755); err != nil {
		t.Fatalf("failed to create repo .claude dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localClaude, "settings.json"), []byte("{\"hooks\":{}}\n"), 0644); err != nil {
		t.Fatalf("failed to seed repo Claude settings: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, true)
	if len(result.errors) > 0 {
		t.Fatalf("runUpdateSync returned errors: %v", result.errors)
	}

	data, err := os.ReadFile(filepath.Join(localClaude, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read synced Claude settings: %v", err)
	}
	if !strings.Contains(string(data), "\"AETHER_ACTIVE_PLATFORM\": \"claude\"") {
		t.Fatalf("expected synced Claude settings to include platform env, got:\n%s", string(data))
	}
}

// --- Stale-publish detection unit tests ---

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.19", "1.0.20", -1},
		{"1.0.20", "1.0.19", 1},
		{"1.0.20", "1.0.20", 0},
		{"1.0.3", "1.0.20", -1},
		{"1.1.0", "1.0.20", 1},
		{"v1.0.20", "1.0.20", 0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.a, tt.b), func(t *testing.T) {
			got := compareVersions(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCheckStalePublishCritical(t *testing.T) {
	hubDir := t.TempDir()
	createHubWithExpectedCounts(t, hubDir)

	result := checkStalePublish(hubDir, "1.0.19", "1.0.20", channelStable, nil)
	if result.Classification != staleCritical {
		t.Errorf("expected critical, got %s", result.Classification)
	}
	if !strings.Contains(result.Message, "hub version 1.0.19 is behind binary version 1.0.20") {
		t.Errorf("unexpected message: %s", result.Message)
	}
	if !strings.Contains(result.RecoveryCommand, "aether publish") {
		t.Errorf("expected recovery command to contain 'aether publish', got: %s", result.RecoveryCommand)
	}
}

func TestCheckStalePublishWarning(t *testing.T) {
	hubDir := t.TempDir()
	createHubWithExpectedCounts(t, hubDir)

	result := checkStalePublish(hubDir, "1.0.21", "1.0.20", channelStable, nil)
	if result.Classification != staleWarning {
		t.Errorf("expected warning, got %s", result.Classification)
	}
}

func TestCheckStalePublishInfoMissingCommands(t *testing.T) {
	hubDir := t.TempDir()
	createHubWithExpectedCounts(t, hubDir)
	// Remove claude commands
	os.RemoveAll(filepath.Join(hubDir, "system", "commands", "claude"))
	os.MkdirAll(filepath.Join(hubDir, "system", "commands", "claude"), 0755)

	result := checkStalePublish(hubDir, "1.0.20", "1.0.20", channelStable, nil)
	if result.Classification != staleInfo {
		t.Errorf("expected info, got %s", result.Classification)
	}
	if len(result.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(result.Components))
	}
	if result.Components[0].Name != "Commands (claude)" {
		t.Errorf("expected component name 'Commands (claude)', got %s", result.Components[0].Name)
	}
	if result.Components[0].Expected != 50 {
		t.Errorf("expected expected=50, got %d", result.Components[0].Expected)
	}
	if result.Components[0].Actual != 0 {
		t.Errorf("expected actual=0, got %d", result.Components[0].Actual)
	}
}

func TestCheckStalePublishOK(t *testing.T) {
	hubDir := t.TempDir()
	createHubWithExpectedCounts(t, hubDir)

	result := checkStalePublish(hubDir, "1.0.20", "1.0.20", channelStable, nil)
	if result.Classification != staleOK {
		t.Errorf("expected ok, got %s", result.Classification)
	}
	if len(result.Components) != 0 {
		t.Errorf("expected 0 components, got %d", len(result.Components))
	}
}

func TestCheckStalePublishDevChannelRecoveryCommand(t *testing.T) {
	hubDir := t.TempDir()
	createHubWithExpectedCounts(t, hubDir)

	result := checkStalePublish(hubDir, "1.0.19", "1.0.20", channelDev, nil)
	if !strings.Contains(result.RecoveryCommand, "--channel dev") {
		t.Errorf("expected recovery command to contain '--channel dev', got: %s", result.RecoveryCommand)
	}
}

func TestCheckStalePublishUnknownHubVersion(t *testing.T) {
	hubDir := t.TempDir()
	createHubWithExpectedCounts(t, hubDir)

	result := checkStalePublish(hubDir, "unknown", "1.0.20", channelStable, nil)
	if result.Classification != staleInfo {
		t.Errorf("expected info, got %s", result.Classification)
	}
	if !strings.Contains(result.Message, "unknown") {
		t.Errorf("expected message to contain 'unknown', got: %s", result.Message)
	}
}

func createHubWithExpectedCounts(t *testing.T, hubDir string) {
	t.Helper()
	system := filepath.Join(hubDir, "system")

	dirs := map[string]int{
		"commands/claude":   expectedClaudeCommandCount,
		"commands/opencode": expectedOpenCodeCommandCount,
		"skills-codex":      expectedCodexSkillCount,
	}
	for rel, count := range dirs {
		dir := filepath.Join(system, filepath.FromSlash(rel))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", rel, err)
		}
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("file_%02d.md", i)
			if err := os.WriteFile(filepath.Join(dir, name), []byte("# test"), 0644); err != nil {
				t.Fatalf("failed to write %s: %v", name, err)
			}
		}
	}

	// OpenCode agents use .md extension with YAML frontmatter
	agentsDir := filepath.Join(system, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}
	for i := 0; i < expectedOpenCodeAgentCount; i++ {
		name := fmt.Sprintf("agent_%02d.md", i)
		content := fmt.Sprintf(`---
description: "This is a valid test agent description for agent %02d"
mode: subagent
model: anthropic/claude-sonnet-4-20250514
tools:
  write: true
  edit: true
  bash: true
color: "#f1c40f"
---

# Test Agent %02d

Test agent content.
`, i, i)
		if err := os.WriteFile(filepath.Join(agentsDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	// Codex agents use .toml extension
	codexDir := filepath.Join(system, "codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		t.Fatalf("failed to create codex dir: %v", err)
	}
	for i := 0; i < expectedCodexAgentCount; i++ {
		name := fmt.Sprintf("agent_%02d.toml", i)
		if err := os.WriteFile(filepath.Join(codexDir, name), validCodexAgentTOML(fmt.Sprintf("agent-%02d", i), "test"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}
}

func TestRunUpdateSyncPreservesClaudeSettingsWithoutForce(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()

	hubSettings := filepath.Join(hubDir, "system", "settings", "claude")
	if err := os.MkdirAll(hubSettings, 0755); err != nil {
		t.Fatalf("failed to create hub settings dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSettings, "settings.json"), []byte("{\"env\":{\"AETHER_ACTIVE_PLATFORM\":\"claude\"},\"hooks\":{}}\n"), 0644); err != nil {
		t.Fatalf("failed to write hub Claude settings: %v", err)
	}

	localClaude := filepath.Join(repoDir, ".claude")
	if err := os.MkdirAll(localClaude, 0755); err != nil {
		t.Fatalf("failed to create repo .claude dir: %v", err)
	}
	custom := "{\n  \"hooks\": {\n    \"Stop\": []\n  }\n}\n"
	if err := os.WriteFile(filepath.Join(localClaude, "settings.json"), []byte(custom), 0644); err != nil {
		t.Fatalf("failed to seed repo Claude settings: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, false)
	if len(result.errors) > 0 {
		t.Fatalf("runUpdateSync returned errors: %v", result.errors)
	}

	data, err := os.ReadFile(filepath.Join(localClaude, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read preserved Claude settings: %v", err)
	}
	if string(data) != custom {
		t.Fatalf("expected local Claude settings to be preserved without force, got:\n%s", string(data))
	}
}
