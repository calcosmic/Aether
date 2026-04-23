package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishCommandExists(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	cmd, _, err := rootCmd.Find([]string{"publish"})
	if err != nil {
		t.Fatalf("publish command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("publish command is nil")
	}
	if cmd.Use != "publish" {
		t.Errorf("publish command Use = %q, want %q", cmd.Use, "publish")
	}
}

func TestPublishCommandFlags(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	cmd, _, err := rootCmd.Find([]string{"publish"})
	if err != nil {
		t.Fatalf("publish command not found: %v", err)
	}

	expectedFlags := []string{"package-dir", "home-dir", "channel", "binary-dest", "skip-build-binary"}
	for _, name := range expectedFlags {
		if f := cmd.Flags().Lookup(name); f == nil {
			t.Errorf("publish command missing flag --%s", name)
		}
	}
}

func TestPublishRejectsNonSourceCheckout(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"publish", "--package-dir", tmpDir, "--home-dir", t.TempDir(), "--skip-build-binary"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected publish to fail for non-source checkout")
	}
	if !strings.Contains(err.Error(), "source checkout") {
		t.Errorf("expected error to mention 'source checkout', got: %v", err)
	}
}

func TestPublishVerificationFailure(t *testing.T) {
	// This test validates the verification logic by ensuring publish
	// detects and corrects a version mismatch between source and hub.
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()

	// Create mock source checkout with version 1.0.20
	packageDir := t.TempDir()
	rootDir := packageDir
	if err := os.WriteFile(filepath.Join(rootDir, "go.mod"), []byte("module github.com/calcosmic/Aether\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	mainDir := filepath.Join(rootDir, "cmd", "aether")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("failed to create cmd/aether: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	aetherDir := filepath.Join(rootDir, ".aether")
	if err := os.MkdirAll(aetherDir, 0755); err != nil {
		t.Fatalf("failed to create .aether: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aetherDir, "version.json"), []byte(`{"version":"1.0.20","updated_at":"now"}`), 0644); err != nil {
		t.Fatalf("failed to write version.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aetherDir, "workers.md"), []byte("# Workers\n"), 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}

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
}
