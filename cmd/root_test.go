package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommandExists(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}
	if rootCmd.Use != "aether" {
		t.Errorf("rootCmd.Use = %q, want \"aether\"", rootCmd.Use)
	}
}

func TestVersionFlag(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--version"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("--version returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "aether") {
		t.Errorf("--version output %q does not contain \"aether\"", output)
	}
	resolved := resolveVersion()
	if !strings.Contains(output, resolved) {
		t.Errorf("--version output %q does not contain version %q", output, resolved)
	}
}

func TestHelpFlag(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--help"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("--help returned error: %v", err)
	}

	output := buf.String()
	// Cobra uses "Available Commands" in help output
	if !strings.Contains(output, "Usage") && !strings.Contains(output, "Available Commands") {
		t.Errorf("--help output does not contain usage information: %q", output)
	}
}

func TestSpecCommandIsRegistered(t *testing.T) {
	configureFrontDoorHelp()

	command, _, err := rootCmd.Find([]string{"spec"})
	if err != nil {
		t.Fatalf("find spec command: %v", err)
	}
	if command != specCmd {
		t.Fatalf("registered spec command = %p, want public spec command %p", command, specCmd)
	}
	if command.GroupID != frontDoorNormalGroupID {
		t.Fatalf("spec command group = %q, want %q", command.GroupID, frontDoorNormalGroupID)
	}

	for _, name := range []string{"discuss", "plan", "build", "run", "status", "update"} {
		registered, _, findErr := rootCmd.Find([]string{name})
		if findErr != nil || registered == rootCmd || registered.Name() != name {
			t.Errorf("existing public command %q is no longer registered", name)
		}
	}
}

func TestRootCodexHelpShowsRestoredLifecycleOrder(t *testing.T) {
	output := rootHelpOutputForPlatform(t, "codex")
	compact := strings.Join(strings.Fields(output), " ")

	last := -1
	for _, command := range []string{
		`aether init "goal"`,
		"aether discuss",
		"aether spec",
		"aether plan",
		"aether build",
		"aether run",
	} {
		index := strings.Index(compact, command)
		if index < 0 {
			t.Fatalf("Codex root help is missing %q:\n%s", command, output)
		}
		if index <= last {
			t.Fatalf("Codex root help lists %q out of lifecycle order:\n%s", command, output)
		}
		last = index
	}

	for _, want := range []string{
		"Clarify material intent before drafting the specification.",
		"Draft, review, revise, and explicitly approve the owner-readable specification.",
		"Generate an evidence-driven candidate plan from the approved specification.",
	} {
		if !strings.Contains(compact, want) {
			t.Errorf("Codex root help is missing authority copy %q", want)
		}
	}
	for _, unsupported := range []string{"/ant-", "$ant-"} {
		if strings.Contains(output, unsupported) {
			t.Errorf("Codex root help exposes unsupported syntax %q:\n%s", unsupported, output)
		}
	}
}

func TestRootHelpSeparatesSpecificationAndPlanAuthority(t *testing.T) {
	groups := frontDoorRenderedHelpGroups("codex")
	var specificationDescription, planDescription string
	for _, group := range groups {
		for _, entry := range group.entries {
			switch entry.command {
			case "aether spec":
				specificationDescription = entry.description
			case "aether plan":
				planDescription = entry.description
			}
		}
	}
	if specificationDescription == "" || planDescription == "" {
		t.Fatalf("root journey is missing separated spec/plan entries: spec=%q plan=%q", specificationDescription, planDescription)
	}
	if !strings.Contains(strings.ToLower(specificationDescription), "approve") {
		t.Errorf("spec description does not explain explicit specification approval: %q", specificationDescription)
	}
	if !strings.Contains(strings.ToLower(planDescription), "candidate plan") {
		t.Errorf("plan description does not explain candidate generation: %q", planDescription)
	}
	for _, forbidden := range []string{"accepts a plan", "activates a plan", "accept the plan", "activate the plan"} {
		if strings.Contains(strings.ToLower(specificationDescription), forbidden) {
			t.Errorf("spec help crosses the plan authority boundary with %q", forbidden)
		}
	}
}

func rootHelpOutputForPlatform(t *testing.T, platform string) string {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	root := t.TempDir()
	t.Setenv("AETHER_ROOT", root)
	t.Setenv("COLONY_DATA_DIR", filepath.Join(root, ".aether", "data"))
	t.Setenv("AETHER_PLATFORM", platform)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("COLUMNS", "120")
	store = nil
	var output bytes.Buffer
	stdout = &output
	stderr = &output
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	t.Cleanup(func() { rootCmd.SetErr(os.Stderr) })
	rootCmd.SetArgs([]string{"--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root help for %s: %v", platform, err)
	}
	return output.String()
}

func TestPersistentPreRunStoreInit(t *testing.T) {
	// Create a temp directory with .aether/data/
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create temp data dir: %v", err)
	}

	// Set environment to point to temp directory
	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	// Reset store for test isolation
	store = nil

	// Create a test command that requires store
	testCmd := &cobra.Command{
		Use: "test-store-init",
		RunE: func(cmd *cobra.Command, args []string) error {
			if store == nil {
				return errStoreNil
			}
			return nil
		},
	}
	_ = cobra.Command{}
	rootCmd.AddCommand(testCmd)
	defer rootCmd.RemoveCommand(testCmd)

	rootCmd.SetArgs([]string{"test-store-init"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	// The test command should succeed if store was initialized
	if err != nil {
		t.Errorf("store init test command failed: %v", err)
	}

	if store == nil {
		t.Error("store was not initialized by PersistentPreRunE")
	}

	if store != nil && store.BasePath() != dataDir {
		t.Errorf("store.BasePath() = %q, want %q", store.BasePath(), dataDir)
	}
}

// errStoreNil is a sentinel error for testing.
var errStoreNil = func() error {
	return os.ErrNotExist
}()

func TestResolveVersionPrefersRepoVersionFile(t *testing.T) {
	originalVersion := Version
	Version = "0.0.0-dev"
	defer func() { Version = originalVersion }()

	tmpDir := t.TempDir()
	goMod := []byte("module github.com/calcosmic/Aether\n\ngo 1.26\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), goMod, 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".aether"), 0755); err != nil {
		t.Fatalf("failed to create .aether dir: %v", err)
	}
	versionPayload, err := json.Marshal(map[string]string{
		"version":    "1.0.17",
		"updated_at": "2026-04-22",
	})
	if err != nil {
		t.Fatalf("failed to marshal version payload: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".aether", "version.json"), versionPayload, 0644); err != nil {
		t.Fatalf("failed to write version.json: %v", err)
	}

	if got := resolveVersion(tmpDir); got != "1.0.17" {
		t.Fatalf("resolveVersion() = %q, want %q", got, "1.0.17")
	}
}

func TestReadHubVersionAtPathFallsBackToSystemVersion(t *testing.T) {
	hubDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(hubDir, "system"), 0755); err != nil {
		t.Fatalf("failed to create hub system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "system", "version.json"), []byte(`{"version":"1.0.27"}`), 0644); err != nil {
		t.Fatalf("failed to write system version: %v", err)
	}

	if got := readHubVersionAtPath(hubDir); got != "1.0.27" {
		t.Fatalf("readHubVersionAtPath() = %q, want %q", got, "1.0.27")
	}
}
