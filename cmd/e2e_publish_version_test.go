package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2EPublishVersionAgreement verifies that publish updates a stale hub
// version to match the source version.
func TestE2EPublishVersionAgreement(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Create mock source checkout with version 1.0.20
	packageDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(packageDir, "go.mod"), []byte("module github.com/calcosmic/Aether\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	mainDir := filepath.Join(packageDir, "cmd", "aether")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("failed to create cmd/aether: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	aetherDir := filepath.Join(packageDir, ".aether")
	if err := os.MkdirAll(aetherDir, 0755); err != nil {
		t.Fatalf("failed to create .aether: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aetherDir, "version.json"), []byte(`{"version":"1.0.20","updated_at":"now"}`), 0644); err != nil {
		t.Fatalf("failed to write version.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aetherDir, "workers.md"), []byte("# Workers\n"), 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}
	writeBuiltTsHostFixture(t, packageDir)

	// Pre-seed hub with stale version 1.0.19
	hubDir := filepath.Join(homeDir, ".aether")
	if err := os.MkdirAll(hubDir, 0755); err != nil {
		t.Fatalf("failed to create hub dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.19","updated_at":"old"}`), 0644); err != nil {
		t.Fatalf("failed to write stale version.json: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"publish", "--package-dir", packageDir, "--home-dir", homeDir, "--skip-build-binary"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	// Verify hub version was updated to 1.0.20
	hubVersion := readHubVersionAtPath(hubDir)
	if hubVersion != "1.0.20" {
		t.Errorf("hub version = %q, want %q", hubVersion, "1.0.20")
	}

	// Verify output is valid JSON with ok:true
	output := buf.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("publish output is not valid JSON: %v\noutput: %s", err, output)
	}
	if ok, _ := result["ok"].(bool); !ok {
		t.Fatalf("publish returned ok:false, output: %s", output)
	}
}

// TestE2EVersionCheckFlag verifies that `aether version --check` passes when
// binary and hub agree, and fails when they disagree.
func TestE2EVersionCheckFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Set binary version to 1.0.20
	oldVersion := Version
	Version = "1.0.20"
	t.Cleanup(func() {
		Version = oldVersion
	})

	// Create hub with matching version 1.0.20
	hubDir := filepath.Join(homeDir, ".aether")
	if err := os.MkdirAll(hubDir, 0755); err != nil {
		t.Fatalf("failed to create hub dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.20","updated_at":"now"}`), 0644); err != nil {
		t.Fatalf("failed to write version.json: %v", err)
	}

	// --- Check passes when versions agree ---
	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"version", "--check"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version --check failed when versions agree: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Version check passed") {
		t.Errorf("expected 'Version check passed' in output, got: %s", output)
	}

	// --- Check fails when versions disagree ---
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.19","updated_at":"old"}`), 0644); err != nil {
		t.Fatalf("failed to write mismatched version.json: %v", err)
	}

	buf.Reset()
	stdout = &buf
	rootCmd.SetArgs([]string{"version", "--check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected version --check to fail when versions disagree")
	}
	if !strings.Contains(err.Error(), "version mismatch") {
		t.Errorf("expected error to contain 'version mismatch', got: %v", err)
	}
}
