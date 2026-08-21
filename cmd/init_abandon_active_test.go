package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// activeColonyFixture writes a mid-flight colony: real goal, work in progress,
// nowhere near sealed.
func activeColonyFixture(t *testing.T, dataDir string) {
	t.Helper()
	goal := "Build a feature that turned out not to be worth building"
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

// TestActiveColonyRefusalNamesTheEscapeHatch pins the message, not just the
// refusal. Abandoning a half-built colony is an ordinary thing to want — the
// goal turns out not to be worth finishing — and the old refusal said only
// "colony already initialized", naming no way forward at all. Entomb requires
// Crowned Anthill, so the only documented route was to seal work you had just
// decided to bin, which runs the whole ceremony over it and promotes its
// instincts to the cross-colony hive.
func TestActiveColonyRefusalNamesTheEscapeHatch(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	activeColonyFixture(t, dataDir)

	var buf strings.Builder
	oldStderr := stderr
	stderr = &buf
	defer func() { stderr = oldStderr }()

	rootCmd.SetArgs([]string{"init", "A completely different goal"})
	defer rootCmd.SetArgs([]string{})
	_ = rootCmd.Execute()

	got := buf.String()
	for _, want := range []string{
		"active colony",
		"--confirm-reinit",
		"backed up",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("refusal must mention %q so the operator has a way forward, got:\n%s", want, got)
		}
	}
}

// TestActiveColonyIsNotReplacedWithoutConfirmation keeps the guard. The escape
// hatch must be deliberate — an accidental second init should not destroy work
// in progress.
func TestActiveColonyIsNotReplacedWithoutConfirmation(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	activeColonyFixture(t, dataDir)

	rootCmd.SetArgs([]string{"init", "A completely different goal"})
	defer rootCmd.SetArgs([]string{})
	_ = rootCmd.Execute()

	var after colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &after); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if after.Goal == nil || !strings.Contains(*after.Goal, "not to be worth building") {
		t.Errorf("unconfirmed init replaced an active colony; goal is now %q", ptrStr(after.Goal))
	}
}

// TestConfirmedReinitAbandonsActiveColonyAndBacksItUp is the feature. The
// backup assertion is the load-bearing half: a confirmation that silently
// destroyed the only copy of a colony's history would be worse than the refusal
// it replaced.
func TestConfirmedReinitAbandonsActiveColonyAndBacksItUp(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	activeColonyFixture(t, dataDir)

	rootCmd.SetArgs([]string{"init", "A completely different goal", "--confirm-reinit"})
	defer rootCmd.SetArgs([]string{})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("confirmed re-init returned error: %v", err)
	}

	var after colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &after); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if after.Goal == nil || *after.Goal != "A completely different goal" {
		t.Fatalf("confirmed re-init did not replace the colony; goal = %q", ptrStr(after.Goal))
	}
	if after.CurrentPhase != 0 || len(after.Plan.Phases) != 0 {
		t.Errorf("abandoned colony left residue: phase=%d, phases=%d", after.CurrentPhase, len(after.Plan.Phases))
	}

	// The old colony must still be recoverable.
	backups, err := filepath.Glob(filepath.Join(dataDir, "backups", "COLONY_STATE.pre-init.*.bak"))
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	if len(backups) == 0 {
		t.Fatal("no backup written; the abandoned colony's history is unrecoverable")
	}
	raw, err := os.ReadFile(backups[0])
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if !strings.Contains(string(raw), "not to be worth building") {
		t.Errorf("backup does not contain the abandoned colony's goal:\n%s", string(raw))
	}
}

// TestSealedColonyPathStillRequiresConfirmation guards against the active-path
// branch loosening the sealed one. Both need saying out loud; neither should
// inherit the other's rules by accident.
func TestSealedColonyPathStillRequiresConfirmation(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	goal := "A finished colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:   "3.0",
		Goal:      &goal,
		State:     colony.StateCOMPLETED,
		Milestone: "Crowned Anthill",
		Plan:      colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Done", Status: colony.PhaseCompleted}}},
	})

	var buf strings.Builder
	oldStderr := stderr
	stderr = &buf
	defer func() { stderr = oldStderr }()

	rootCmd.SetArgs([]string{"init", "Something new"})
	defer rootCmd.SetArgs([]string{})
	_ = rootCmd.Execute()

	got := buf.String()
	if !strings.Contains(got, "sealed colony") {
		t.Errorf("sealed refusal changed shape, got:\n%s", got)
	}
	if !strings.Contains(got, "entomb") {
		t.Errorf("sealed refusal must still prefer entomb, got:\n%s", got)
	}
}
