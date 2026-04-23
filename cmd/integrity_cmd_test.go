package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Unit tests: scanIntegrity function
// ---------------------------------------------------------------------------

func TestScanIntegrityReturnsEmptyWhenHealthy(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.20"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	issues := scanIntegrity()
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for healthy state, got %d: %+v", len(issues), issues)
	}
}

func TestScanIntegrityCriticalWhenHubNotInstalled(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	issues := scanIntegrity()
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for missing hub, got %d", len(issues))
	}
	if issues[0].Severity != "critical" {
		t.Errorf("expected severity=critical, got %q", issues[0].Severity)
	}
	if issues[0].Category != "integrity" {
		t.Errorf("expected category=integrity, got %q", issues[0].Category)
	}
	if !issues[0].Fixable {
		t.Error("expected fixable=true for hub-not-installed issue")
	}
	if !strings.Contains(issues[0].Message, "install") {
		t.Errorf("expected message to contain 'install', got: %s", issues[0].Message)
	}
}

func TestScanIntegrityCriticalWhenVersionsDisagree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	issues := scanIntegrity()
	// Should have at least a critical version mismatch issue
	foundVersionMismatch := false
	for _, issue := range issues {
		if issue.Category == "integrity" && strings.Contains(issue.Message, "does not match") {
			foundVersionMismatch = true
			if issue.Severity != "critical" {
				t.Errorf("expected critical severity for version mismatch, got %q", issue.Severity)
			}
			if !issue.Fixable {
				t.Error("expected fixable=true for version mismatch")
			}
			if !strings.Contains(issue.Message, "publish") {
				t.Errorf("expected recovery guidance in message, got: %s", issue.Message)
			}
		}
	}
	if !foundVersionMismatch {
		t.Errorf("expected version mismatch issue, got %d issues: %+v", len(issues), issues)
	}
}

func TestScanIntegrityWarningForStaleInfo(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	// Create a partial hub (missing companion files) to trigger staleInfo
	system := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(filepath.Join(system, "commands", "claude"), 0755); err != nil {
		t.Fatalf("failed to create hub system: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(system, "commands", "opencode"), 0755); err != nil {
		t.Fatalf("failed to create opencode dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(system, "agents"), 0755); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(system, "codex"), 0755); err != nil {
		t.Fatalf("failed to create codex dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(system, "skills-codex"), 0755); err != nil {
		t.Fatalf("failed to create skills-codex dir: %v", err)
	}
	// Write matching version but minimal companion files
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.20"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	issues := scanIntegrity()
	// Should have at least one warning for incomplete companion files
	foundStaleWarning := false
	for _, issue := range issues {
		if issue.Category == "integrity" && issue.Severity == "warning" {
			foundStaleWarning = true
			if !issue.Fixable {
				t.Error("expected fixable=true for stale info issue")
			}
			if !strings.Contains(issue.Message, "Recovery:") {
				t.Errorf("expected recovery text in message, got: %s", issue.Message)
			}
		}
	}
	if !foundStaleWarning {
		t.Errorf("expected stale info warning, got %d issues: %+v", len(issues), issues)
	}
}

// ---------------------------------------------------------------------------
// Medic deep scan integration: scanIntegrity wired into performHealthScan
// ---------------------------------------------------------------------------

func TestMedicDeepIncludesIntegrity(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Create a temp repo with colony data
	repoDir := t.TempDir()
	dataDir := filepath.Join(repoDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	t.Setenv("AETHER_ROOT", repoDir)

	goal := "Test integrity in medic deep"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateIDLE,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	// Hub version disagrees with binary version
	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	result, err := performHealthScan(MedicOptions{Deep: true})
	if err != nil {
		t.Fatalf("performHealthScan returned error: %v", err)
	}

	foundIntegrity := false
	for _, issue := range result.Issues {
		if issue.Category == "integrity" {
			foundIntegrity = true
			break
		}
	}
	if !foundIntegrity {
		t.Errorf("expected at least one integrity-category issue in deep scan, got %d issues: %+v",
			len(result.Issues), result.Issues)
	}
}

func TestMedicDeepIntegrityRecovery(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Create a temp repo with colony data
	repoDir := t.TempDir()
	dataDir := filepath.Join(repoDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	t.Setenv("AETHER_ROOT", repoDir)

	goal := "Test recovery commands in medic deep"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateIDLE,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	// Hub version disagrees with binary version
	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	result, err := performHealthScan(MedicOptions{Deep: true})
	if err != nil {
		t.Fatalf("performHealthScan returned error: %v", err)
	}

	for _, issue := range result.Issues {
		if issue.Category == "integrity" {
			if !issue.Fixable {
				t.Errorf("expected integrity issue to be fixable, got: %+v", issue)
			}
			if !strings.Contains(strings.ToLower(issue.Message), "publish") &&
				!strings.Contains(strings.ToLower(issue.Message), "version") {
				t.Errorf("expected actionable recovery text in integrity issue, got: %s", issue.Message)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Integrity command tests
// ---------------------------------------------------------------------------

func TestIntegrityCommandExists(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	if integrityCmd.Use != "integrity" {
		t.Errorf("integrity command Use = %q, want %q", integrityCmd.Use, "integrity")
	}
	if !strings.Contains(integrityCmd.Short, "release pipeline") {
		t.Errorf("integrity command Short missing 'release pipeline', got: %s", integrityCmd.Short)
	}
	for _, name := range []string{"json", "channel", "source"} {
		f := integrityCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("integrity command missing flag --%s", name)
		}
	}
}

func TestIntegrityJSONOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.20"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	// Chdir to temp dir (consumer context)
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	repoDir := t.TempDir()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"integrity", "--json"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("integrity --json returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, buf.String())
	}
	for _, key := range []string{"context", "channel", "checks", "overall"} {
		if _, ok := result[key]; !ok {
			t.Errorf("JSON output missing key %q, got: %s", key, buf.String())
		}
	}
	checks, ok := result["checks"].([]interface{})
	if !ok || len(checks) == 0 {
		t.Errorf("expected non-empty checks array, got: %v", checks)
	}
}

func TestIntegrityDetectSourceContext(t *testing.T) {
	saveGlobals(t)

	ctx := detectIntegrityContext()
	if ctx != "source" {
		t.Errorf("expected source context when running from Aether repo, got: %q", ctx)
	}
}

func TestIntegrityDetectConsumerContext(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	ctx := detectIntegrityContext()
	if ctx != "consumer" {
		t.Errorf("expected consumer context when outside Aether repo, got: %q", ctx)
	}
}

func TestIntegrityExitCodeFail(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	// No hub installed

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	repoDir := t.TempDir()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"integrity", "--json"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when hub not installed, got nil")
	}
}

func TestIntegrityChannelFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	devHubDir := filepath.Join(homeDir, ".aether-dev")
	createHubWithExpectedCounts(t, devHubDir)
	if err := os.WriteFile(filepath.Join(devHubDir, "version.json"), []byte(`{"version":"1.0.20"}`), 0644); err != nil {
		t.Fatalf("failed to write dev hub version: %v", err)
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	repoDir := t.TempDir()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"integrity", "--json", "--channel", "dev"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("integrity --json --channel dev returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, buf.String())
	}
	if result["channel"] != "dev" {
		t.Errorf("expected channel=dev, got: %v", result["channel"])
	}
}

func TestIntegritySourceFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.20"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	repoDir := t.TempDir()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"integrity", "--json", "--source"})
	defer rootCmd.SetArgs([]string{})

	// --source forces source context which adds checkSourceVersion.
	// From a consumer temp dir, checkSourceVersion will fail (no .aether/version.json),
	// so the command returns an error. That is expected behavior.
	err = rootCmd.Execute()

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, buf.String())
	}
	if result["context"] != "source" {
		t.Errorf("expected context=source, got: %v", result["context"])
	}
	checks, _ := result["checks"].([]interface{})
	if len(checks) != 5 {
		t.Errorf("expected 5 checks for source context, got %d", len(checks))
	}
	// The command should fail because source version check can't resolve from temp dir
	if err == nil {
		t.Error("expected error when --source is used outside Aether repo (source version check fails)")
	}
}

func TestIntegrityVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	hubDir := filepath.Join(homeDir, ".aether")
	createHubWithExpectedCounts(t, hubDir)
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.20"}`), 0644); err != nil {
		t.Fatalf("failed to write hub version: %v", err)
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldDir)
	repoDir := t.TempDir()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	oldVersion := Version
	Version = "1.0.20"
	defer func() { Version = oldVersion }()

	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"integrity"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("integrity returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "R E L E A S E   I N T E G R I T Y") {
		t.Errorf("visual output missing 'R E L E A S E   I N T E G R I T Y', got:\n%s", output)
	}
	// Should contain check markers (pass or fail)
	hasCheckMarker := strings.Contains(output, "\u2713") || strings.Contains(output, "\u2717")
	if !hasCheckMarker {
		t.Errorf("visual output missing check markers, got:\n%s", output)
	}
}
