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
	// Manages its own hub via --home-dir; opt out of suite-wide hub isolation.
	t.Setenv("AETHER_HUB_DIR", "")
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

// TestPublishSucceedsOnFirstRunAfterVersionBump proves the first `aether
// publish` after a version bump succeeds without a rerun, even when the
// in-process Version ldflags variable — standing in for "the binary
// currently executing publish was built before the bump" — is stale.
//
// The mechanism under test: setupInstallHub (cmd/install_cmd.go) must not
// re-derive the hub version from resolveVersion(packageDir), because
// resolveVersion prefers the running binary's baked-in ldflags Version (see
// cmd/root.go's priority order) over the source checkout's .aether/version.json.
// runPublish already resolves the correct version from the source checkout
// (cmd/publish_cmd.go:69-76) and must pass it straight through to
// setupInstallHub rather than letting it re-resolve independently.
func TestPublishSucceedsOnFirstRunAfterVersionBump(t *testing.T) {
	// Manages its own hub via --home-dir; opt out of suite-wide hub isolation.
	t.Setenv("AETHER_HUB_DIR", "")
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Create mock source checkout with the BUMPED version 1.0.21.
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
	if err := os.WriteFile(filepath.Join(aetherDir, "version.json"), []byte(`{"version":"1.0.21","updated_at":"now"}`), 0644); err != nil {
		t.Fatalf("failed to write version.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aetherDir, "workers.md"), []byte("# Workers\n"), 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}
	writeBuiltTsHostFixture(t, packageDir)

	// Pre-seed the hub at the OLD version 1.0.20 in BOTH the legacy top-level
	// version.json and system/version.json — mirroring a hub that was last
	// published before this bump.
	hubDir := filepath.Join(homeDir, ".aether")
	hubSystemDir := filepath.Join(hubDir, "system")
	if err := os.MkdirAll(hubSystemDir, 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.20","updated_at":"old"}`), 0644); err != nil {
		t.Fatalf("failed to write stale hub version.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystemDir, "version.json"), []byte(`{"version":"1.0.20","updated_at":"old"}`), 0644); err != nil {
		t.Fatalf("failed to write stale hub system/version.json: %v", err)
	}

	// This is the part that reproduces the bug: override the package-level
	// Version to the STALE 1.0.20, standing in for "the binary executing
	// publish was built before the bump." Without the fix, setupInstallHub
	// calls resolveVersion(packageDir), which returns this stale override
	// instead of the source checkout's bumped version.json.
	oldVersion := Version
	Version = "1.0.20"
	t.Cleanup(func() {
		Version = oldVersion
	})

	var buf bytes.Buffer
	stdout = &buf

	// A single publish invocation — no rerun. A second run is exactly the
	// workaround this requirement removes.
	rootCmd.SetArgs([]string{"publish", "--package-dir", packageDir, "--home-dir", homeDir, "--skip-build-binary"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("publish failed on first run after version bump: %v", err)
	}

	output := buf.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("publish output is not valid JSON: %v\noutput: %s", err, output)
	}
	if ok, _ := result["ok"].(bool); !ok {
		t.Fatalf("publish returned ok:false, output: %s", output)
	}
	inner, _ := result["result"].(map[string]interface{})
	if v, _ := inner["version"].(string); v != "1.0.21" {
		t.Errorf("publish envelope version = %q, want %q", v, "1.0.21")
	}

	// Assert on the filesystem, not just the envelope — asserting only the
	// envelope would pass even if the legacy top-level file stayed stale,
	// which is the exact defect this test guards against.
	legacyVersion := readVersionJSONFile(filepath.Join(hubDir, "version.json"))
	if legacyVersion != "1.0.21" {
		t.Errorf("hub legacy version.json = %q, want %q", legacyVersion, "1.0.21")
	}
	systemVersion := readVersionJSONFile(filepath.Join(hubSystemDir, "version.json"))
	if systemVersion != "1.0.21" {
		t.Errorf("hub system/version.json = %q, want %q", systemVersion, "1.0.21")
	}
}

// TestE2EVersionCheckFlag verifies that `aether version --check` passes when
// binary and hub agree, and fails when they disagree.
func TestE2EVersionCheckFlag(t *testing.T) {
	// Manages its own hub via --home-dir; opt out of suite-wide hub isolation.
	t.Setenv("AETHER_HUB_DIR", "")
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
