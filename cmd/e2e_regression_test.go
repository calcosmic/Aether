package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2ERegressionStablePublishUpdate proves the full stable pipeline:
// publish -> downstream update -> version agreement.
func TestE2ERegressionStablePublishUpdate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Step 1: Create mock source checkout
	sourceDir := createMockSourceCheckout(t, "1.0.99-test")

	// Step 2: Publish to stable hub
	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"publish", "--package-dir", sourceDir, "--home-dir", homeDir, "--skip-build-binary"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("stable publish failed: %v", err)
	}

	// Step 3: Verify hub has correct version
	hubDir := filepath.Join(homeDir, ".aether")
	hubVersion := readHubVersionAtPath(hubDir)
	if hubVersion != "1.0.99-test" {
		t.Errorf("hub version = %q, want %q", hubVersion, "1.0.99-test")
	}

	// Step 4: Create downstream repo and update
	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}

	buf.Reset()
	stdout = &buf

	rootCmd.SetArgs([]string{"update", "--force"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("stable update failed: %v", err)
	}

	// Step 5: Verify downstream has workers.md
	repoWorkers := filepath.Join(repoDir, ".aether", "workers.md")
	if _, err := os.Stat(repoWorkers); os.IsNotExist(err) {
		t.Fatal("downstream workers.md not created by update")
	}

	// Step 6: Verify version agreement — hub version matches source version
	hubVersionAfter := readHubVersionAtPath(hubDir)
	if hubVersionAfter != "1.0.99-test" {
		t.Errorf("hub version after update = %q, want %q", hubVersionAfter, "1.0.99-test")
	}
}

// TestE2ERegressionDevPublishUpdate proves the full dev pipeline:
// dev publish -> dev update -> version agreement.
func TestE2ERegressionDevPublishUpdate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Step 1: Create mock source checkout with dev version
	sourceDir := createMockSourceCheckout(t, "2.0.0-dev")

	// Step 2: Publish to dev channel
	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"publish", "--package-dir", sourceDir, "--home-dir", homeDir, "--skip-build-binary", "--channel", "dev"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("dev publish failed: %v", err)
	}

	// Step 3: Verify dev hub (NOT stable hub) has correct version
	devHubDir := filepath.Join(homeDir, ".aether-dev")
	devHubVersion := readHubVersionAtPath(devHubDir)
	if devHubVersion != "2.0.0-dev" {
		t.Errorf("dev hub version = %q, want %q", devHubVersion, "2.0.0-dev")
	}

	// Step 4: Create downstream repo and run dev update
	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}

	buf.Reset()
	stdout = &buf

	rootCmd.SetArgs([]string{"update", "--force", "--channel", "dev"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("dev update failed: %v", err)
	}

	// Step 5: Verify downstream has workers.md
	repoWorkers := filepath.Join(repoDir, ".aether", "workers.md")
	if _, err := os.Stat(repoWorkers); os.IsNotExist(err) {
		t.Fatal("downstream workers.md not created by dev update")
	}

	// Step 6: Verify dev hub version matches source
	devHubVersionAfter := readHubVersionAtPath(devHubDir)
	if devHubVersionAfter != "2.0.0-dev" {
		t.Errorf("dev hub version after update = %q, want %q", devHubVersionAfter, "2.0.0-dev")
	}
}

// TestE2ERegressionStalePublishDetection proves stale publish is caught at
// the downstream update boundary with critical classification and recovery command.
func TestE2ERegressionStalePublishDetection(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Step 1: Create hub with expected companion file counts
	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)

	// Step 2: Write stale hub version
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.18-stale","updated_at":"old"}`), 0644); err != nil {
		t.Fatalf("failed to write stale version.json: %v", err)
	}

	// Step 3: Set binary Version ahead of hub
	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	// Step 4: Create downstream repo and update
	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"update", "--force"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for critical stale publish, got nil")
	}
	if !strings.Contains(err.Error(), "stale publish detected") {
		t.Errorf("expected error to contain 'stale publish detected', got: %v", err)
	}

	// Step 5: Parse JSON output and verify stale_publish details
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON output: %v, output: %s", err, buf.String())
	}

	inner, _ := result["result"].(map[string]interface{})
	stale, _ := inner["stale_publish"].(map[string]interface{})

	if stale["classification"] != "critical" {
		t.Errorf("expected classification=critical, got: %v", stale["classification"])
	}
	if stale["binary_version"] != "1.0.20" {
		t.Errorf("expected binary_version=1.0.20, got: %v", stale["binary_version"])
	}
	if stale["hub_version"] != "1.0.18-stale" {
		t.Errorf("expected hub_version=1.0.18-stale, got: %v", stale["hub_version"])
	}

	recovery, _ := stale["recovery_command"].(string)
	if !strings.Contains(recovery, "aether publish") {
		t.Errorf("expected recovery_command to contain 'aether publish', got: %v", recovery)
	}
}

// TestE2ERegressionChannelIsolation proves dev publish does not contaminate
// the stable hub at all — version.json and workers.md remain unchanged.
func TestE2ERegressionChannelIsolation(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()

	// Step 1: Publish stable version
	stableSource := createMockSourceCheckout(t, "1.0.20-stable")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"publish", "--package-dir", stableSource, "--home-dir", homeDir, "--skip-build-binary"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("stable publish failed: %v", err)
	}

	// Step 2: Record stable hub state
	stableHubDir := filepath.Join(homeDir, ".aether")
	stableVersionBefore := readHubVersionAtPath(stableHubDir)
	stableWorkersBefore, err := os.ReadFile(filepath.Join(stableHubDir, "system", "workers.md"))
	if err != nil {
		t.Fatalf("failed to read stable workers.md before dev publish: %v", err)
	}

	// Step 3: Publish dev version
	devSource := createMockSourceCheckout(t, "2.0.0-dev")

	buf.Reset()
	stdout = &buf

	rootCmd.SetArgs([]string{"publish", "--package-dir", devSource, "--home-dir", homeDir, "--skip-build-binary", "--channel", "dev"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("dev publish failed: %v", err)
	}

	// Step 4: Verify dev hub has correct version
	devHubDir := filepath.Join(homeDir, ".aether-dev")
	devHubVersion := readHubVersionAtPath(devHubDir)
	if devHubVersion != "2.0.0-dev" {
		t.Errorf("dev hub version = %q, want %q", devHubVersion, "2.0.0-dev")
	}

	// Step 5: Verify stable hub version.json unchanged
	stableVersionAfter := readHubVersionAtPath(stableHubDir)
	if stableVersionAfter != stableVersionBefore {
		t.Errorf("stable hub version changed after dev publish: before=%q, after=%q", stableVersionBefore, stableVersionAfter)
	}

	// Step 6: Verify stable hub workers.md content unchanged
	stableWorkersAfter, err := os.ReadFile(filepath.Join(stableHubDir, "system", "workers.md"))
	if err != nil {
		t.Fatalf("failed to read stable workers.md after dev publish: %v", err)
	}
	if string(stableWorkersAfter) != string(stableWorkersBefore) {
		t.Errorf("stable hub workers.md changed after dev publish")
	}

	// Step 7: Verify no dev version string leaked into stable workers.md
	if strings.Contains(string(stableWorkersAfter), "2.0.0-dev") {
		t.Error("dev version string leaked into stable hub workers.md")
	}
}
