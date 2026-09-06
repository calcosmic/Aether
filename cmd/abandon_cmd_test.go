package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func abandonFixture(t *testing.T, dataDir string) {
	t.Helper()
	goal := "Build a thing I will regret"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "First", Status: colony.PhaseCompleted},
				{ID: 2, Name: "Second", Status: colony.PhaseInProgress},
			},
		},
	})
}

// TestAbandonPreviewDoesNotMutate is the corollary this project already learned
// the hard way: consolidation-phase-end and consolidation-seal both wrote to
// instincts.json for months behind a --dry-run flag reading "Report without
// modifying". A preview that mutates is worse than no preview, because the
// operator uses it precisely to decide whether to mutate.
func TestAbandonPreviewDoesNotMutate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	abandonFixture(t, dataDir)

	before, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}

	rootCmd.SetArgs([]string{"abandon"})
	defer rootCmd.SetArgs([]string{})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("abandon preview returned error: %v", err)
	}

	after, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read state after preview: %v", err)
	}
	if string(before) != string(after) {
		t.Error("abandon preview mutated COLONY_STATE.json; it must only report")
	}
	if backups, _ := filepath.Glob(filepath.Join(dataDir, "backups", "*.bak")); len(backups) > 0 {
		t.Errorf("abandon preview wrote a backup (%v); it should not touch anything", backups)
	}
}

// TestAbandonPreviewNamesWhatWouldBeLost keeps the hidden compatibility path
// specific: the retired command must say that it has no authority to mutate.
func TestAbandonPreviewNamesWhatWouldBeLost(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	abandonFixture(t, dataDir)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_PLATFORM", "codex")

	var buf strings.Builder
	oldStdout := stdout
	stdout = &buf
	defer func() { stdout = oldStdout }()

	rootCmd.SetArgs([]string{"abandon"})
	defer rootCmd.SetArgs([]string{})
	_ = rootCmd.Execute()

	got := buf.String()
	for _, want := range []string{
		"standalone abandon command is retired",
		"did not change state",
		"aether seal --force --reason",
		"aether status",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("preview must mention %q, got:\n%s", want, got)
		}
	}
}

// TestAbandonConfirmedClearsColonyAndKeepsBackup keeps the historical input
// fail-closed: even a former confirmation flag cannot regain clear authority.
func TestAbandonConfirmedClearsColonyAndKeepsBackup(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	abandonFixture(t, dataDir)
	before, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read state before retired command: %v", err)
	}
	var buf strings.Builder
	stdout = &buf

	rootCmd.SetArgs([]string{"abandon", "--confirm"})
	defer rootCmd.SetArgs([]string{})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("abandon --confirm returned error: %v", err)
	}

	after, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read state after retired command: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("retired abandon --confirm mutated colony state")
	}
	if !strings.Contains(buf.String(), "standalone abandon command is retired") || !strings.Contains(buf.String(), `"state_effect":"none"`) {
		t.Fatalf("retired abandon did not explain its zero-write behavior: %s", buf.String())
	}
	if backups, _ := filepath.Glob(filepath.Join(dataDir, "backups", "*.bak")); len(backups) != 0 {
		t.Fatalf("retired abandon wrote unexpected backups: %v", backups)
	}
}

// TestAbandonWithNoColonyIsNotAnError pins the empty case. Telling an operator
// something failed when nothing needed doing sends them looking for a problem
// that does not exist.
func TestAbandonWithNoColonyIsNotAnError(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	createTestColonyState(t, dataDir, colony.ColonyState{Version: "3.0", State: colony.StateIDLE})

	rootCmd.SetArgs([]string{"abandon"})
	defer rootCmd.SetArgs([]string{})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("abandon with no colony returned error: %v", err)
	}
	if backups, _ := filepath.Glob(filepath.Join(dataDir, "backups", "*.bak")); len(backups) > 0 {
		t.Errorf("abandon with no colony wrote a backup: %v", backups)
	}
}
