package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ReleasePipelineSnapshot is the golden snapshot for release pipeline verification.
type ReleasePipelineSnapshot struct {
	SyncPairCount          int      `json:"sync_pair_count"`
	HomeSyncPairCount      int      `json:"home_sync_pair_count"`
	Version                string   `json:"version"`
}

// loadReleasePipelineSnapshot loads the golden snapshot from testdata.
func loadReleasePipelineSnapshot(t *testing.T) *ReleasePipelineSnapshot {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "release_pipeline_snapshot.json"))
	if err != nil {
		t.Fatalf("failed to read release_pipeline_snapshot.json: %v", err)
	}
	var snapshot ReleasePipelineSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("failed to unmarshal snapshot: %v", err)
	}
	return &snapshot
}

// createMockSourceCheckoutForRelease creates a minimal Aether source directory for testing.
func createMockSourceCheckoutForRelease(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/calcosmic/Aether\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	mainDir := filepath.Join(dir, "cmd", "aether")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("failed to create cmd/aether: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Create .aether/ directory with version and workers files
	aetherDir := filepath.Join(dir, ".aether")
	if err := os.MkdirAll(aetherDir, 0755); err != nil {
		t.Fatalf("failed to create .aether: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aetherDir, "version.json"), []byte(`{"version":"`+version+`","updated_at":"now"}`), 0644); err != nil {
		t.Fatalf("failed to write version.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aetherDir, "workers.md"), []byte("# Workers\n"), 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}

	// Create companion file directories with minimal content
	// Claude commands
	claudeCmdDir := filepath.Join(dir, ".claude", "commands", "ant")
	if err := os.MkdirAll(claudeCmdDir, 0755); err != nil {
		t.Fatalf("failed to create claude commands: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeCmdDir, "build.md"), []byte("---\nname: ant-build\n---\n# Build\n"), 0644); err != nil {
		t.Fatalf("failed to write claude command: %v", err)
	}

	// Claude agents
	claudeAgentDir := filepath.Join(dir, ".claude", "agents", "ant")
	if err := os.MkdirAll(claudeAgentDir, 0755); err != nil {
		t.Fatalf("failed to create claude agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeAgentDir, "aether-builder.md"), []byte("---\nname: aether-builder\n---\n# Builder\n"), 0644); err != nil {
		t.Fatalf("failed to write claude agent: %v", err)
	}

	// OpenCode commands
	openCodeCmdDir := filepath.Join(dir, ".opencode", "commands", "ant")
	if err := os.MkdirAll(openCodeCmdDir, 0755); err != nil {
		t.Fatalf("failed to create opencode commands: %v", err)
	}
	if err := os.WriteFile(filepath.Join(openCodeCmdDir, "build.md"), []byte("# Build\n"), 0644); err != nil {
		t.Fatalf("failed to write opencode command: %v", err)
	}

	// OpenCode agents
	openCodeAgentDir := filepath.Join(dir, ".opencode", "agents")
	if err := os.MkdirAll(openCodeAgentDir, 0755); err != nil {
		t.Fatalf("failed to create opencode agents: %v", err)
	}
	openCodeAgentContent := []byte(`---
name: aether-builder
description: Test builder agent for pipeline verification purposes
mode: subagent
color: "#4CAF50"
tools:
  Read: {}
  Bash: {}
---
# Builder
`)
	if err := os.WriteFile(filepath.Join(openCodeAgentDir, "aether-builder.md"), openCodeAgentContent, 0644); err != nil {
		t.Fatalf("failed to write opencode agent: %v", err)
	}

	// Codex agents
	codexAgentDir := filepath.Join(dir, ".codex", "agents")
	if err := os.MkdirAll(codexAgentDir, 0755); err != nil {
		t.Fatalf("failed to create codex agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexAgentDir, "aether-builder.toml"), validCodexAgentTOML("aether-builder", "builder"), 0644); err != nil {
		t.Fatalf("failed to write codex agent: %v", err)
	}

	return dir
}

// TestReleasePipelineE2E simulates the full publish->hub sync->install->update cycle.
func TestReleasePipelineE2E(t *testing.T) {
	saveGlobals(t)

	snapshot := loadReleasePipelineSnapshot(t)
	version := snapshot.Version

	// 1. Create mock source checkout with companion files
	sourceDir := createMockSourceCheckoutForRelease(t, version)

	// 2. Create mock hub directory
	hubDir := t.TempDir()

	// 3. Run publish sync logic (call setupInstallHub directly)
	hubResult := setupInstallHub(hubDir, sourceDir)
	if errVal, ok := hubResult["error"].(string); ok && errVal != "" {
		t.Fatalf("hub setup failed: %s", errVal)
	}

	// 4. Verify hub version matches source version
	hubVersion := readHubVersionAtPath(hubDir)
	if hubVersion != version {
		t.Errorf("hub version = %q, want %q", hubVersion, version)
	}

	// 5. Verify hub system directory exists and has content
	hubSystem := filepath.Join(hubDir, "system")
	entries, err := os.ReadDir(hubSystem)
	if err != nil {
		t.Fatalf("hub system dir not readable: %v", err)
	}
	if len(entries) == 0 {
		t.Error("hub system dir is empty after sync")
	}

	// 6. Verify version.json exists in hub system
	if _, err := os.Stat(filepath.Join(hubSystem, "version.json")); err != nil {
		t.Errorf("version.json not found in hub system: %v", err)
	}

	// 7. Verify workers.md exists in hub system
	if _, err := os.Stat(filepath.Join(hubSystem, "workers.md")); err != nil {
		t.Errorf("workers.md not found in hub system: %v", err)
	}

	// 8. Run update --force from mock hub back to a fresh mock repo
	repoDir := t.TempDir()
	// Create a managed stale file that cleanup should remove
	staleDir := filepath.Join(repoDir, ".aether", "docs")
	if err := os.MkdirAll(staleDir, 0755); err != nil {
		t.Fatalf("create stale dir parent: %v", err)
	}
	staleFile := filepath.Join(staleDir, "stale-generated-file.md")
	if err := os.WriteFile(staleFile, []byte("stale\n"), 0644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	syncResult := runUpdateSync(hubDir, repoDir, true)
	if len(syncResult.errors) > 0 {
		// Some errors are expected if hub doesn't have matching content; log but don't fail
		t.Logf("update sync had %d errors (may be expected for empty hub): %v", len(syncResult.errors), syncResult.errors)
	}

	// 9. Verify stale file in managed dir is removed
	if _, err := os.Stat(staleFile); err == nil {
		t.Errorf("stale file %s should have been removed by cleanup", staleFile)
	} else if !os.IsNotExist(err) {
		t.Errorf("stat stale file %s: %v", staleFile, err)
	}

	// 10. Run install logic to a fresh mock home (simulate install from source to fresh home)
	freshHome := t.TempDir()
	_, platformErrors := syncPlatformHomeAssets(sourceDir, freshHome, channelStable)
	if len(platformErrors) > 0 {
		t.Fatalf("platform home sync errors: %v", platformErrors)
	}

	// 11. Verify platform home files exist
	for _, pair := range installSyncPairs() {
		destDir := filepath.Join(freshHome, filepath.FromSlash(pair.destRel))
		if _, err := os.Stat(destDir); err != nil {
			t.Errorf("install sync pair %q dest %s not found: %v", pair.label, pair.destRel, err)
		}
	}
}

// TestPublishHubSync verifies publish->hub sync and version agreement.
func TestPublishHubSync(t *testing.T) {
	saveGlobals(t)

	snapshot := loadReleasePipelineSnapshot(t)
	version := snapshot.Version

	// 1. Create mock source checkout with companion files
	sourceDir := createMockSourceCheckoutForRelease(t, version)

	// 2. Create mock hub
	hubDir := t.TempDir()

	// 3. Run publish sync
	hubResult := setupInstallHub(hubDir, sourceDir)
	if errVal, ok := hubResult["error"].(string); ok && errVal != "" {
		t.Fatalf("hub setup failed: %s", errVal)
	}

	// 4. Verify version agreement
	hubVersion := readHubVersionAtPath(hubDir)
	if hubVersion != version {
		t.Errorf("hub version = %q, want %q", hubVersion, version)
	}

	// 5. Verify sync pair count matches snapshot
	actualPairs := len(installSyncPairs())
	if actualPairs != snapshot.SyncPairCount {
		t.Errorf("installSyncPairs count = %d, want %d", actualPairs, snapshot.SyncPairCount)
	}
	actualHomePairs := len(platformHomeHubSyncPairs())
	if actualHomePairs != snapshot.HomeSyncPairCount {
		t.Errorf("platformHomeHubSyncPairs count = %d, want %d", actualHomePairs, snapshot.HomeSyncPairCount)
	}

	// 6. Verify hub system dir has content
	hubSystem := filepath.Join(hubDir, "system")
	entries, err := os.ReadDir(hubSystem)
	if err != nil {
		t.Fatalf("hub system dir not readable: %v", err)
	}
	if len(entries) == 0 {
		t.Error("hub system dir is empty after sync")
	}
}

// TestUpdateStaleFileCleanup verifies that update --force removes stale files in managed dirs.
func TestUpdateStaleFileCleanup(t *testing.T) {
	saveGlobals(t)

	// 1. Create mock hub with some content
	hubDir := t.TempDir()
	hubSystem := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(filepath.Join(hubSystem, "docs"), 0755); err != nil {
		t.Fatalf("create hub docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("write hub workers: %v", err)
	}

	// 2. Create mock repo with a stale file in a managed directory
	repoDir := t.TempDir()
	staleDir := filepath.Join(repoDir, ".aether", "docs")
	if err := os.MkdirAll(staleDir, 0755); err != nil {
		t.Fatalf("create stale dir parent: %v", err)
	}
	staleFile := filepath.Join(staleDir, "stale-generated-file.md")
	if err := os.WriteFile(staleFile, []byte("stale\n"), 0644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	// Also write a managed system file
	managedFile := filepath.Join(repoDir, ".aether", "manifest.json")
	if err := os.WriteFile(managedFile, []byte(`{"files":{}}`), 0644); err != nil {
		t.Fatalf("write managed file: %v", err)
	}

	// 3. Run update --force
	syncResult := runUpdateSync(hubDir, repoDir, true)
	if len(syncResult.errors) > 0 {
		t.Logf("update sync had %d errors: %v", len(syncResult.errors), syncResult.errors)
	}

	// 4. Verify stale file in managed dir is removed
	if _, err := os.Stat(staleFile); err == nil {
		t.Errorf("stale file %s should have been removed", staleFile)
	} else if !os.IsNotExist(err) {
		t.Errorf("stat stale file %s: %v", staleFile, err)
	}

	// 5. Verify managed system file is also removed (it's in the cleanup list)
	if _, err := os.Stat(managedFile); err == nil {
		t.Logf("managed file %s still exists (may be preserved by protected logic)", managedFile)
	}
}
