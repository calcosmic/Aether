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

// TestE2EStableUpdateDetectsCriticalStale proves stable channel update detects
// critical stale publish when hub version is behind binary version.
func TestE2EStableUpdateDetectsCriticalStale(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.19","updated_at":"old"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
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
	if stale["binary_version"] != "1.0.20" {
		t.Errorf("expected binary_version=1.0.20, got: %v", stale["binary_version"])
	}
	if stale["hub_version"] != "1.0.19" {
		t.Errorf("expected hub_version=1.0.19, got: %v", stale["hub_version"])
	}
	if stale["channel"] != "stable" {
		t.Errorf("expected channel=stable, got: %v", stale["channel"])
	}
	recovery, _ := stale["recovery_command"].(string)
	if !strings.Contains(recovery, "aether publish") {
		t.Errorf("expected recovery_command to contain 'aether publish', got: %v", recovery)
	}
}

// TestE2EDevUpdateDetectsCriticalStale proves dev channel update detects
// critical stale publish.
func TestE2EDevUpdateDetectsCriticalStale(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether-dev")
	createHubWithExpectedCounts(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.19","updated_at":"old"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
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
	recovery, _ := stale["recovery_command"].(string)
	if !strings.Contains(recovery, "--channel dev") {
		t.Errorf("expected recovery_command to contain '--channel dev', got: %v", recovery)
	}
}

// TestE2EUpdateDetectsInfoStale proves info-level detection when versions agree
// but companion files are incomplete.
func TestE2EUpdateDetectsInfoStale(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	// Remove most claude commands to trigger info
	claudeDir := filepath.Join(hubDir, "system", "commands", "claude")
	entries, _ := os.ReadDir(claudeDir)
	for _, entry := range entries {
		if !entry.IsDir() {
			os.Remove(filepath.Join(claudeDir, entry.Name()))
		}
	}
	// Create only 5 files
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("file_%02d.md", i)
		if err := os.WriteFile(filepath.Join(claudeDir, name), []byte("# test"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.20"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
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
			if comp["actual"] != float64(5) {
				t.Errorf("expected actual=5 for claude commands, got: %v", comp["actual"])
			}
			if comp["expected"] != float64(50) {
				t.Errorf("expected expected=50 for claude commands, got: %v", comp["expected"])
			}
		}
	}
	if !foundClaude {
		t.Errorf("expected components to contain claude entry, got: %v", components)
	}
}

// TestE2EUpdateDetectsOK proves ok path when versions agree and companion files
// are complete.
func TestE2EUpdateDetectsOK(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.20"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
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
		t.Fatalf("expected no error for ok path, got: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON output: %v, output: %s", err, buf.String())
	}
	inner, _ := result["result"].(map[string]interface{})
	stale, _ := inner["stale_publish"].(map[string]interface{})
	if stale["classification"] != "ok" {
		t.Errorf("expected classification=ok, got: %v", stale["classification"])
	}
	components, _ := stale["components"].([]interface{})
	if len(components) > 0 {
		t.Errorf("expected empty components for ok path, got: %v", components)
	}
}

// TestE2EUpdateDryRunDetectsCriticalStale proves dry-run still reports critical
// stale honestly.
func TestE2EUpdateDryRunDetectsCriticalStale(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.19"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
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

// TestE2EUpdateVisualBannerForCriticalStale proves visual output mode shows the
// stale publish banner for critical classification.
func TestE2EUpdateVisualBannerForCriticalStale(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.19"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"update", "--force"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for critical stale, got nil")
	}

	output := buf.String()
	if !strings.Contains(output, "S T A L E") || !strings.Contains(output, "D E T E C T E D") {
		t.Errorf("expected banner title in output, got: %s", output)
	}
	if !strings.Contains(output, "Binary version: 1.0.20") {
		t.Errorf("expected binary version in output, got: %s", output)
	}
	if !strings.Contains(output, "Hub version: 1.0.19") {
		t.Errorf("expected hub version in output, got: %s", output)
	}
	if !strings.Contains(output, "Classification: critical") {
		t.Errorf("expected classification in output, got: %s", output)
	}
	if !strings.Contains(output, "Recovery") {
		t.Errorf("expected Recovery section in output, got: %s", output)
	}
	if !strings.Contains(output, "aether publish") {
		t.Errorf("expected recovery command in output, got: %s", output)
	}
}

// TestE2EUpdateVisualBannerForInfoStale proves visual output mode shows info
// status for incomplete companion files.
func TestE2EUpdateVisualBannerForInfoStale(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	claudeDir := filepath.Join(hubDir, "system", "commands", "claude")
	entries, _ := os.ReadDir(claudeDir)
	for _, entry := range entries {
		if !entry.IsDir() {
			os.Remove(filepath.Join(claudeDir, entry.Name()))
		}
	}
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("file_%02d.md", i)
		if err := os.WriteFile(filepath.Join(claudeDir, name), []byte("# test"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.20"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	repoDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"update"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for info stale, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "P U B L I S H") || !strings.Contains(output, "S T A T U S") {
		t.Errorf("expected banner title in output, got: %s", output)
	}
	if !strings.Contains(output, "Commands (claude): 5 found, expected 50") {
		t.Errorf("expected component count in output, got: %s", output)
	}
}
