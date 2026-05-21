package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestBuildReconcileDryRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	// Init git
	exec.Command("git", "init").Run()
	exec.Command("git", "config", "user.email", "test@aether").Run()
	exec.Command("git", "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".aether/\n"), 0644)
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "initial").Run()

	// Create uncommitted file
	os.WriteFile(filepath.Join(root, "new.go"), []byte("package main\n"), 0644)

	goal := "Test"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "P1", Status: colony.PhaseInProgress}},
		},
	})

	var buf strings.Builder
	stdout = &buf

	rootCmd.SetArgs([]string{"build-reconcile", "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build-reconcile returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"dry_run":true`) {
		t.Errorf("expected dry_run=true in output, got: %s", output)
	}
	if !strings.Contains(output, `"total_changes":1`) {
		t.Errorf("expected total_changes=1, got: %s", output)
	}
	if !strings.Contains(output, `"added":`) {
		t.Errorf("expected added array, got: %s", output)
	}

	// Verify claims were NOT written
	_, err := os.Stat(filepath.Join(dataDir, "build-reconcile-phase-1.json"))
	if err == nil {
		t.Error("reconcile packet should not exist in dry-run mode")
	}
}

func TestBuildReconcileCreatesSyntheticPacket(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	exec.Command("git", "init").Run()
	exec.Command("git", "config", "user.email", "test@aether").Run()
	exec.Command("git", "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".aether/\n"), 0644)
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "initial").Run()

	os.WriteFile(filepath.Join(root, "reconciled.go"), []byte("package main\n"), 0644)

	goal := "Test"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "P1", Status: colony.PhaseInProgress}},
		},
	})

	var buf strings.Builder
	stdout = &buf

	rootCmd.SetArgs([]string{"build-reconcile"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build-reconcile returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"synthetic":true`) {
		t.Errorf("expected synthetic=true, got: %s", output)
	}
	if !strings.Contains(output, `"total_changes":1`) {
		t.Errorf("expected total_changes=1, got: %s", output)
	}

	// Verify claims were written
	_, err := os.Stat(filepath.Join(dataDir, "build-reconcile-phase-1.json"))
	if err != nil {
		t.Errorf("reconcile packet should exist: %v", err)
	}
}

func TestBuildReconcileNoChanges(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	exec.Command("git", "init").Run()
	exec.Command("git", "config", "user.email", "test@aether").Run()
	exec.Command("git", "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".aether/\n"), 0644)
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "initial").Run()

	goal := "Test"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "P1", Status: colony.PhaseInProgress}},
		},
	})

	var buf strings.Builder
	stdout = &buf

	rootCmd.SetArgs([]string{"build-reconcile"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build-reconcile returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"total_changes":0`) {
		t.Errorf("expected total_changes=0 for clean repo, got: %s", output)
	}
}

func TestBuildReconcileAppendsToExistingPacket(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	exec.Command("git", "init").Run()
	exec.Command("git", "config", "user.email", "test@aether").Run()
	exec.Command("git", "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".aether/\n"), 0644)
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "initial").Run()

	os.WriteFile(filepath.Join(root, "existing.go"), []byte("package main\n"), 0644)
	exec.Command("git", "add", "existing.go").Run()
	exec.Command("git", "commit", "-m", "add existing").Run()

	// Pre-existing claims
	claims := codexBuildClaims{
		BuildPhase:    1,
		FilesCreated:  []string{"existing.go"},
		FilesModified: []string{},
	}
	if err := store.SaveJSON("last-build-claims.json", claims); err != nil {
		t.Fatalf("save claims: %v", err)
	}

	// New uncommitted file
	os.WriteFile(filepath.Join(root, "new.go"), []byte("package main\n"), 0644)

	goal := "Test"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "P1", Status: colony.PhaseInProgress}},
		},
	})

	var buf strings.Builder
	stdout = &buf

	rootCmd.SetArgs([]string{"build-reconcile"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build-reconcile returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"total_changes":1`) {
		t.Errorf("expected only 1 new change, got: %s", output)
	}

	// Verify claims include both files
	var savedClaims codexBuildClaims
	if err := store.LoadJSON("last-build-claims.json", &savedClaims); err != nil {
		t.Fatalf("load saved claims: %v", err)
	}
	foundExisting := false
	for _, f := range savedClaims.FilesCreated {
		if f == "existing.go" {
			foundExisting = true
			break
		}
	}
	if !foundExisting {
		t.Error("saved claims should still include existing.go")
	}
}

func TestBuildReconcileSyntheticFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	exec.Command("git", "init").Run()
	exec.Command("git", "config", "user.email", "test@aether").Run()
	exec.Command("git", "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".aether/\n"), 0644)
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "initial").Run()

	os.WriteFile(filepath.Join(root, "synthetic.go"), []byte("package main\n"), 0644)

	goal := "Test"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "P1", Status: colony.PhaseInProgress}},
		},
	})

	var buf strings.Builder
	stdout = &buf

	rootCmd.SetArgs([]string{"build-reconcile"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build-reconcile returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"synthetic":true`) {
		t.Errorf("expected synthetic flag in output, got: %s", output)
	}

	// Verify packet has synthetic marker
	packetPath := filepath.Join(dataDir, "build-reconcile-phase-1.json")
	data, err := os.ReadFile(packetPath)
	if err != nil {
		t.Fatalf("read packet: %v", err)
	}
	var packet map[string]interface{}
	if err := json.Unmarshal(data, &packet); err != nil {
		t.Fatalf("parse packet: %v", err)
	}
	if packet["synthetic"] != true {
		t.Errorf("packet should have synthetic=true, got: %v", packet["synthetic"])
	}
}
