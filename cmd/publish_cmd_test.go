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
	// Manages its own hub via --home-dir; opt out of suite-wide hub isolation.
	t.Setenv("AETHER_HUB_DIR", "")
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
	// This test manages its own hub via --home-dir; opt out of the
	// suite-wide AETHER_HUB_DIR isolation so home-based resolution applies.
	t.Setenv("AETHER_HUB_DIR", "")
	// This test validates the verification logic by ensuring publish
	// detects and corrects a version mismatch between source and hub.
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()

	// Create mock source checkout with version 1.0.20
	packageDir := t.TempDir()
	seedCodexSkillSupportFixture(t, packageDir)
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
	writeBuiltTsHostFixture(t, rootDir)

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
	assertCodexSkillFixtureInstalled(t, packageDir, homeDir)

	// Verify hub version was updated to 1.0.20
	hubVersion := readHubVersionAtPath(hubDir)
	if hubVersion != "1.0.20" {
		t.Errorf("hub version = %q, want %q", hubVersion, "1.0.20")
	}
}

func TestPublishHubVersionWarningIncludesRecoveryAndVerificationCommands(t *testing.T) {
	// This test manages its own hub via --home-dir; opt out of the
	// suite-wide AETHER_HUB_DIR isolation so home-based resolution applies.
	t.Setenv("AETHER_HUB_DIR", "")
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	packageDir := createMockSourceCheckout(t, "1.0.20")

	hubDir := filepath.Join(homeDir, ".aether")
	if err := os.MkdirAll(hubDir, 0755); err != nil {
		t.Fatalf("failed to create hub dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version":"1.0.19","updated_at":"old"}`), 0644); err != nil {
		t.Fatalf("failed to write stale version.json: %v", err)
	}

	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf

	rootCmd.SetArgs([]string{"publish", "--package-dir", packageDir, "--home-dir", homeDir, "--skip-build-binary", "--channel", "stable"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	assertCodexSkillFixtureInstalled(t, packageDir, homeDir)

	warning := errBuf.String()
	for _, want := range []string{
		"Warning: hub version updated from 1.0.19 to 1.0.20",
		"aether publish",
		"aether version --check",
		"aether integrity --source --channel stable",
		"aether update --force",
	} {
		if !strings.Contains(warning, want) {
			t.Fatalf("hub-version warning missing %q:\n%s", want, warning)
		}
	}
}

func TestPublishSyncsStablePlatformHomeCommands(t *testing.T) {
	// Manages its own hub via --home-dir; opt out of suite-wide hub isolation.
	t.Setenv("AETHER_HUB_DIR", "")
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	packageDir := createMockSourceCheckout(t, "1.0.20")
	commandDir := filepath.Join(packageDir, ".claude", "commands", "ant")
	if err := os.MkdirAll(commandDir, 0755); err != nil {
		t.Fatalf("failed to create command dir: %v", err)
	}
	generated := []byte("<!-- Aether-managed: runtime spec at .aether/commands/build.yaml. Synced by aether update. -->\n---\nname: ant-build\n---\n")
	if err := os.WriteFile(filepath.Join(commandDir, "build.md"), generated, 0644); err != nil {
		t.Fatalf("failed to write command: %v", err)
	}

	legacyDir := filepath.Join(homeDir, ".claude", "commands", "ant")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatalf("failed to create legacy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "build.md"), generated, 0644); err != nil {
		t.Fatalf("failed to write legacy command: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"publish", "--package-dir", packageDir, "--home-dir", homeDir, "--skip-build-binary", "--channel", "stable"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	assertCodexSkillFixtureInstalled(t, packageDir, homeDir)

	if _, err := os.Stat(filepath.Join(homeDir, ".claude", "commands", "ant-build.md")); err != nil {
		t.Fatalf("expected publish to sync flat Claude command: %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacyDir, "build.md")); err == nil {
		t.Fatal("expected publish to remove generated legacy Claude command")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat legacy command: %v", err)
	}
}

// createMockSourceCheckout creates a minimal Aether source directory for testing.
func createMockSourceCheckout(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	seedCodexSkillSupportFixture(t, dir)
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
	writeBuiltTsHostFixture(t, dir)
	return dir
}

func writeBuiltTsHostFixture(t *testing.T, root string) {
	t.Helper()
	tsHostDir := filepath.Join(root, ".aether", "ts-host")
	if err := os.MkdirAll(filepath.Join(tsHostDir, "dist"), 0755); err != nil {
		t.Fatalf("failed to create TS host fixture dist: %v", err)
	}
	pkg := []byte(`{"name":"@aether/test-ts-host","version":"0.0.0","type":"module","dependencies":{}}` + "\n")
	if err := os.WriteFile(filepath.Join(tsHostDir, "package.json"), pkg, 0644); err != nil {
		t.Fatalf("failed to write TS host package.json: %v", err)
	}
	lock := []byte(`{"name":"@aether/test-ts-host","lockfileVersion":3,"packages":{"":{"name":"@aether/test-ts-host","version":"0.0.0","dependencies":{}}}}` + "\n")
	if err := os.WriteFile(filepath.Join(tsHostDir, "package-lock.json"), lock, 0644); err != nil {
		t.Fatalf("failed to write TS host package-lock.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tsHostDir, "dist", "host.js"), []byte("export {};\n"), 0644); err != nil {
		t.Fatalf("failed to write TS host dist/host.js: %v", err)
	}
}

func TestPublishSyncsBuiltTsHostToHub(t *testing.T) {
	// This test manages its own hub via --home-dir; opt out of the
	// suite-wide AETHER_HUB_DIR isolation so home-based resolution applies.
	t.Setenv("AETHER_HUB_DIR", "")
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	packageDir := createMockSourceCheckout(t, "1.0.20")

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"publish", "--package-dir", packageDir, "--home-dir", homeDir, "--skip-build-binary", "--channel", "stable"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	assertCodexSkillFixtureInstalled(t, packageDir, homeDir)

	for _, rel := range []string{"package.json", "package-lock.json", filepath.Join("dist", "host.js")} {
		path := filepath.Join(homeDir, ".aether", "system", "ts-host", rel)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected TS host artifact %s after publish: %v", path, err)
		}
	}
}

func TestPublishChannelIsolation(t *testing.T) {
	// This test manages its own hub via --home-dir; opt out of the
	// suite-wide AETHER_HUB_DIR isolation so home-based resolution applies.
	t.Setenv("AETHER_HUB_DIR", "")
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	stableSource := createMockSourceCheckout(t, "1.0.20-stable")
	devSource := createMockSourceCheckout(t, "1.0.20-dev")

	var buf bytes.Buffer
	stdout = &buf

	// Publish stable first
	rootCmd.SetArgs([]string{"publish", "--package-dir", stableSource, "--home-dir", homeDir, "--skip-build-binary", "--channel", "stable"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("stable publish failed: %v", err)
	}
	assertCodexSkillFixtureInstalled(t, stableSource, homeDir)

	stableHubVersion := readHubVersionAtPath(filepath.Join(homeDir, ".aether"))
	if stableHubVersion != "1.0.20-stable" {
		t.Errorf("stable hub version = %q, want %q", stableHubVersion, "1.0.20-stable")
	}

	// Publish dev second
	rootCmd.SetArgs([]string{"publish", "--package-dir", devSource, "--home-dir", homeDir, "--skip-build-binary", "--channel", "dev"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("dev publish failed: %v", err)
	}

	devHubVersion := readHubVersionAtPath(filepath.Join(homeDir, ".aether-dev"))
	if devHubVersion != "1.0.20-dev" {
		t.Errorf("dev hub version = %q, want %q", devHubVersion, "1.0.20-dev")
	}

	// Verify stable hub is untouched by dev publish
	stableHubVersionAfter := readHubVersionAtPath(filepath.Join(homeDir, ".aether"))
	if stableHubVersionAfter != "1.0.20-stable" {
		t.Errorf("stable hub version after dev publish = %q, want %q", stableHubVersionAfter, "1.0.20-stable")
	}

	// Verify dev hub contains no stable files
	stableSystemPath := filepath.Join(homeDir, ".aether-dev", "system", "version.json")
	if _, err := os.Stat(stableSystemPath); err != nil {
		t.Fatalf("dev hub system/version.json missing: %v", err)
	}
	data, err := os.ReadFile(stableSystemPath)
	if err != nil {
		t.Fatalf("failed to read dev hub system/version.json: %v", err)
	}
	if !strings.Contains(string(data), "1.0.20-dev") {
		t.Errorf("dev hub system/version.json does not contain dev version: %s", string(data))
	}

	// Reverse order: publish dev first, then stable, on a fresh home
	homeDir2 := t.TempDir()
	rootCmd.SetArgs([]string{"publish", "--package-dir", devSource, "--home-dir", homeDir2, "--skip-build-binary", "--channel", "dev"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("reverse dev publish failed: %v", err)
	}

	rootCmd.SetArgs([]string{"publish", "--package-dir", stableSource, "--home-dir", homeDir2, "--skip-build-binary", "--channel", "stable"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("reverse stable publish failed: %v", err)
	}

	devHubVersion2 := readHubVersionAtPath(filepath.Join(homeDir2, ".aether-dev"))
	if devHubVersion2 != "1.0.20-dev" {
		t.Errorf("reverse dev hub version = %q, want %q", devHubVersion2, "1.0.20-dev")
	}
}

func TestPublishDevBlocksStableHub(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	sourceDir := createMockSourceCheckout(t, "1.0.20")
	homeDir := t.TempDir()
	stableHubDir := filepath.Join(homeDir, ".aether")
	if err := os.MkdirAll(stableHubDir, 0755); err != nil {
		t.Fatalf("failed to create stable hub dir: %v", err)
	}

	t.Setenv("AETHER_HUB_DIR", stableHubDir)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"publish", "--package-dir", sourceDir, "--home-dir", homeDir, "--skip-build-binary", "--channel", "dev"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected dev publish to fail when targeting stable hub")
	}
	if !strings.Contains(err.Error(), "dev publish cannot target stable hub") {
		t.Errorf("expected error to mention 'dev publish cannot target stable hub', got: %v", err)
	}
}

func TestPublishStableBlocksDevHub(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	sourceDir := createMockSourceCheckout(t, "1.0.20")
	homeDir := t.TempDir()
	devHubDir := filepath.Join(homeDir, ".aether-dev")
	if err := os.MkdirAll(devHubDir, 0755); err != nil {
		t.Fatalf("failed to create dev hub dir: %v", err)
	}

	t.Setenv("AETHER_HUB_DIR", devHubDir)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"publish", "--package-dir", sourceDir, "--home-dir", homeDir, "--skip-build-binary", "--channel", "stable"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected stable publish to fail when targeting dev hub")
	}
	if !strings.Contains(err.Error(), "stable publish cannot target dev hub") {
		t.Errorf("expected error to mention 'stable publish cannot target dev hub', got: %v", err)
	}
}

func TestPublishDevAllowsDevHub(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	sourceDir := createMockSourceCheckout(t, "1.0.20-dev")
	homeDir := t.TempDir()
	devHubDir := filepath.Join(homeDir, ".aether-dev")
	if err := os.MkdirAll(devHubDir, 0755); err != nil {
		t.Fatalf("failed to create dev hub dir: %v", err)
	}

	t.Setenv("AETHER_HUB_DIR", devHubDir)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"publish", "--package-dir", sourceDir, "--home-dir", homeDir, "--skip-build-binary", "--channel", "dev"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("dev publish to dev hub failed: %v", err)
	}

	devHubVersion := readHubVersionAtPath(devHubDir)
	if devHubVersion != "1.0.20-dev" {
		t.Errorf("dev hub version = %q, want %q", devHubVersion, "1.0.20-dev")
	}
}

// Under `go run ./cmd/aether publish` the executable is a temp go-build
// binary named "aether"; trusting its directory sent the freshly built binary
// into the go-run temp dir while publish reported success. Two publishes
// shipped stale binaries this way before it was caught.
func TestDefaultLocalBinaryDestIgnoresGoRunTempExecutable(t *testing.T) {
	if isEphemeralExecutablePath("/Users/dev/.local/bin/aether") {
		t.Error("real install path misclassified as ephemeral")
	}
	for _, exe := range []string{
		"/private/var/folders/pj/x/T/go-build2384/b001/exe/aether",
		"/tmp/go-build812345/b001/exe/aether",
	} {
		if !isEphemeralExecutablePath(exe) {
			t.Errorf("go-run temp executable %q not classified as ephemeral", exe)
		}
	}
}
