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

// TestSetupSyncDirSkipsProtectedDirs verifies that setupSyncDir skips
// files under protected directories (data/, dreams/).
func TestSetupSyncDirSkipsProtectedDirs(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")

	// Create source with a protected directory containing files
	if err := os.MkdirAll(filepath.Join(srcDir, "data"), 0755); err != nil {
		t.Fatalf("failed to create src data dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "dreams"), 0755); err != nil {
		t.Fatalf("failed to create src dreams dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "data", "COLONY_STATE.json"), []byte(`{"from":"hub"}`), 0644); err != nil {
		t.Fatalf("failed to create src state file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "dreams", "dream1.md"), []byte("# dream"), 0644); err != nil {
		t.Fatalf("failed to create src dream file: %v", err)
	}

	// Also create a normal file that should be copied
	if err := os.WriteFile(filepath.Join(srcDir, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to create src workers file: %v", err)
	}

	// Pre-create local data/ with user content that must NOT be overwritten
	localDataDir := filepath.Join(destDir, "data")
	if err := os.MkdirAll(localDataDir, 0755); err != nil {
		t.Fatalf("failed to create local data dir: %v", err)
	}
	userState := `{"goal":"user goal","state":"ACTIVE"}`
	if err := os.WriteFile(filepath.Join(localDataDir, "COLONY_STATE.json"), []byte(userState), 0644); err != nil {
		t.Fatalf("failed to create local state: %v", err)
	}

	protectedDirs := map[string]bool{"data": true, "dreams": true}
	protectedFiles := map[string]bool{}

	result := setupSyncDirProtected(srcDir, destDir, protectedDirs, protectedFiles)

	// workers.md should have been copied
	if _, err := os.Stat(filepath.Join(destDir, "workers.md")); os.IsNotExist(err) {
		t.Error("expected workers.md to be copied")
	}

	// Protected directory files must NOT have been copied
	content, err := os.ReadFile(filepath.Join(localDataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("failed to read local state: %v", err)
	}
	if string(content) != userState {
		t.Errorf("protected data/ file was overwritten\ngot:  %s\nwant: %s", string(content), userState)
	}

	// dreams/ should not exist at all in dest (never created)
	if _, err := os.Stat(filepath.Join(destDir, "dreams")); err == nil {
		t.Error("expected dreams/ to NOT be created in dest (protected)")
	}

	if result.copied != 1 {
		t.Errorf("expected 1 file copied, got %d", result.copied)
	}
	if result.skipped != 2 {
		t.Errorf("expected 2 files skipped (protected), got %d", result.skipped)
	}
}

// TestSetupSyncDirSkipsProtectedFiles verifies that setupSyncDir skips
// protected files like QUEEN.md even at the root level.
func TestSetupSyncDirSkipsProtectedFiles(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")

	// Create source with a protected file
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "QUEEN.md"), []byte("# Queen from hub"), 0644); err != nil {
		t.Fatalf("failed to create src QUEEN.md: %v", err)
	}
	// Normal file
	if err := os.WriteFile(filepath.Join(srcDir, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to create src workers.md: %v", err)
	}

	// Pre-create local QUEEN.md with user content
	localQueen := "# My QUEEN.md content"
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "QUEEN.md"), []byte(localQueen), 0644); err != nil {
		t.Fatalf("failed to create local QUEEN.md: %v", err)
	}

	protectedDirs := map[string]bool{}
	protectedFiles := map[string]bool{"QUEEN.md": true}

	result := setupSyncDirProtected(srcDir, destDir, protectedDirs, protectedFiles)

	// workers.md should be copied
	if _, err := os.Stat(filepath.Join(destDir, "workers.md")); os.IsNotExist(err) {
		t.Error("expected workers.md to be copied")
	}

	// QUEEN.md must be preserved
	content, err := os.ReadFile(filepath.Join(destDir, "QUEEN.md"))
	if err != nil {
		t.Fatalf("failed to read local QUEEN.md: %v", err)
	}
	if string(content) != localQueen {
		t.Errorf("protected QUEEN.md was overwritten\ngot:  %s\nwant: %s", string(content), localQueen)
	}

	if result.copied != 1 {
		t.Errorf("expected 1 file copied, got %d", result.copied)
	}
	if result.skipped != 1 {
		t.Errorf("expected 1 file skipped (protected), got %d", result.skipped)
	}
}

// TestSetupCommandRespectsProtectedDirs verifies the full setup command
// does not overwrite files in data/ or dreams/ directories.
func TestSetupCommandRespectsProtectedDirs(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	// Create hub with a data/ directory containing a file (simulating
	// a corrupted or malicious hub)
	hubSystem := filepath.Join(homeDir, ".aether", "system")
	hubDataDir := filepath.Join(hubSystem, "data")
	if err := os.MkdirAll(hubDataDir, 0755); err != nil {
		t.Fatalf("failed to create hub data dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDataDir, "COLONY_STATE.json"), []byte(`{"from":"hub","corrupted":true}`), 0644); err != nil {
		t.Fatalf("failed to create hub state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to create hub file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".aether", "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}

	// Pre-create local data/ with user content
	localDataDir := filepath.Join(repoDir, ".aether", "data")
	if err := os.MkdirAll(localDataDir, 0755); err != nil {
		t.Fatalf("failed to create local data dir: %v", err)
	}
	userState := `{"goal":"user goal","state":"ACTIVE"}`
	if err := os.WriteFile(filepath.Join(localDataDir, "COLONY_STATE.json"), []byte(userState), 0644); err != nil {
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

	// User data must be preserved
	content, err := os.ReadFile(filepath.Join(localDataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("failed to read local state: %v", err)
	}
	if string(content) != userState {
		t.Errorf("user COLONY_STATE.json was overwritten by hub version\ngot:  %s\nwant: %s", string(content), userState)
	}
}

// TestSetupSyncsCodexAgents verifies that setup syncs the Codex agents
// directory from hub system/codex/ to the local .codex/agents/ directory.
func TestSetupSyncsCodexAgents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	// Create hub structure with a codex/ directory containing agent files
	hubSystem := filepath.Join(homeDir, ".aether", "system")
	codexDir := filepath.Join(hubSystem, "codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		t.Fatalf("failed to create hub codex dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".aether", "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexDir, "aether-builder.toml"), validCodexAgentTOML("aether-builder", "builder"), 0644); err != nil {
		t.Fatalf("failed to create codex agent file: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("setup command failed: %v", err)
	}

	// Verify codex agent file was copied to .codex/agents/
	destFile := filepath.Join(repoDir, ".codex", "agents", "aether-builder.toml")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Errorf("expected %s to exist after setup (Codex sync pair missing)", destFile)
	}

	// Verify output mentions the Codex sync
	output := buf.String()
	if !strings.Contains(output, "codex") && !strings.Contains(output, "Codex") {
		t.Errorf("expected output to mention Codex agents, got: %s", output)
	}
}

// TestSetupGeneratesAgentsMD verifies that setup creates AGENTS.md in the
// repo root when it does not already exist, using the hub template.
func TestSetupGeneratesAgentsMD(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	// Create hub structure
	hubSystem := filepath.Join(homeDir, ".aether", "system")
	hubTemplates := filepath.Join(hubSystem, "templates")
	if err := os.MkdirAll(hubTemplates, 0755); err != nil {
		t.Fatalf("failed to create hub templates dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".aether", "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}

	// Create AGENTS.md template with placeholders
	templateContent := "# {COLONY_NAME}\n\nColony goal: {COLONY_GOAL}\n\n## Aether CLI\n\nUse `aether` commands.\n"
	if err := os.WriteFile(filepath.Join(hubTemplates, "agents-md-template.md"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("setup command failed: %v", err)
	}

	// Verify AGENTS.md was created
	agentsMD := filepath.Join(repoDir, "AGENTS.md")
	content, err := os.ReadFile(agentsMD)
	if err != nil {
		t.Fatalf("AGENTS.md was not created: %v", err)
	}

	// Placeholders should be replaced with empty strings
	result := string(content)
	if strings.Contains(result, "{COLONY_NAME}") {
		t.Error("expected {COLONY_NAME} placeholder to be replaced, but it remains")
	}
	if strings.Contains(result, "{COLONY_GOAL}") {
		t.Error("expected {COLONY_GOAL} placeholder to be replaced, but it remains")
	}
}

// TestSetupGeneratesCodexMD verifies that setup creates .codex/CODEX.md in the
// repo when the template exists in the hub.
func TestSetupGeneratesCodexMD(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	hubSystem := filepath.Join(homeDir, ".aether", "system")
	hubTemplates := filepath.Join(hubSystem, "templates")
	if err := os.MkdirAll(hubTemplates, 0755); err != nil {
		t.Fatalf("failed to create hub templates dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".aether", "version.json"), []byte(`{"version":"1.0.5"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubTemplates, "agents-md-template.md"), []byte("# {COLONY_NAME}\n"), 0644); err != nil {
		t.Fatalf("failed to create AGENTS template: %v", err)
	}
	codexTemplate := "# CODEX.md -- Aether Codex Workflow Guide\n\nUse `AETHER_OUTPUT_MODE=visual aether build 1`.\n\n*Generated by Aether setup. Customize this file for your project.*\n"
	if err := os.WriteFile(filepath.Join(hubTemplates, "codex-md-template.md"), []byte(codexTemplate), 0644); err != nil {
		t.Fatalf("failed to create CODEX template: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup command failed: %v", err)
	}

	codexPath := filepath.Join(repoDir, ".codex", "CODEX.md")
	content, err := os.ReadFile(codexPath)
	if err != nil {
		t.Fatalf(".codex/CODEX.md was not created: %v", err)
	}
	if !strings.Contains(string(content), "AETHER_OUTPUT_MODE=visual") {
		t.Fatalf("expected visual execution guidance in .codex/CODEX.md, got:\n%s", string(content))
	}
}

// TestSetupSkipsExistingAgentsMD verifies that setup does not overwrite an
// existing AGENTS.md file.
func TestSetupSkipsExistingAgentsMD(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	repoDir := t.TempDir()

	// Create hub structure
	hubSystem := filepath.Join(homeDir, ".aether", "system")
	hubTemplates := filepath.Join(hubSystem, "templates")
	if err := os.MkdirAll(hubTemplates, 0755); err != nil {
		t.Fatalf("failed to create hub templates dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".aether", "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to create hub version: %v", err)
	}

	templateContent := "# {COLONY_NAME}\n\nTemplate content\n"
	if err := os.WriteFile(filepath.Join(hubTemplates, "agents-md-template.md"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Pre-create AGENTS.md with user content
	userContent := "# My Project\n\nCustom AGENTS.md content\n"
	if err := os.WriteFile(filepath.Join(repoDir, "AGENTS.md"), []byte(userContent), 0644); err != nil {
		t.Fatalf("failed to create user AGENTS.md: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("setup command failed: %v", err)
	}

	// AGENTS.md should be preserved
	content, err := os.ReadFile(filepath.Join(repoDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}
	if string(content) != userContent {
		t.Errorf("existing AGENTS.md was overwritten\ngot:  %s\nwant: %s", string(content), userContent)
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
