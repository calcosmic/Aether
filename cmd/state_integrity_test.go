package cmd

import (
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestInsertPhaseMaintainsSequentialIDs is the WP4/H-02 gate: inserting a
// phase mid-plan must leave phase IDs strictly sequential (phase.ID ==
// index+1) and keep CurrentPhase pointing at the same phase, because
// production call sites index phases by ordinal (phaseNum-1).
func TestInsertPhaseMaintainsSequentialIDs(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Sequential phases"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 2,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Foundation", Status: colony.PhaseCompleted},
			{ID: 2, Name: "Feature", Status: colony.PhaseInProgress},
			{ID: 3, Name: "Polish", Status: colony.PhasePending},
		}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"phase-insert", "--after", "1", "--name", "Hotfix", "--description", "Corrective work"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("phase-insert execute: %v", err)
	}

	var updated colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &updated); err != nil {
		t.Fatalf("reload state: %v", err)
	}

	if len(updated.Plan.Phases) != 4 {
		t.Fatalf("expected 4 phases, got %d", len(updated.Plan.Phases))
	}
	for i, phase := range updated.Plan.Phases {
		if phase.ID != i+1 {
			t.Fatalf("phase at index %d has ID %d; IDs must be sequential (got order %v)", i, phase.ID, testPhaseIDs(updated.Plan.Phases))
		}
	}
	if updated.Plan.Phases[1].Name != "Hotfix" {
		t.Fatalf("inserted phase not at index 1: %v", phaseNames(updated.Plan.Phases))
	}
	// CurrentPhase was phase "Feature" (old ID 2); after the insert it sits at
	// index 2, so its ID — and CurrentPhase — must be 3.
	if updated.CurrentPhase != 3 {
		t.Fatalf("CurrentPhase not remapped: got %d, want 3 (phase %q)", updated.CurrentPhase, updated.Plan.Phases[2].Name)
	}
	if updated.Plan.Phases[2].Name != "Feature" {
		t.Fatalf("expected Feature at index 2, got %q", updated.Plan.Phases[2].Name)
	}
}

// TestColonizeBeforeInitDoesNotCreateState is the WP4/H-03 gate: recording a
// survey with no colony must not fabricate a goalless READY state, because
// loadActiveColonyState rejects it and the colony is poisoned.
func TestColonizeBeforeInitDoesNotCreateState(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	recorded, err := updateSurveyState("2026-08-04T00:00:00Z", 5)
	if err != nil {
		t.Fatalf("updateSurveyState: %v", err)
	}
	if recorded {
		t.Fatal("survey reported as recorded with no colony state present")
	}

	var state colony.ColonyState
	if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr == nil {
		t.Fatalf("COLONY_STATE.json was fabricated by colonize-before-init: %+v", state)
	}
}

func testPhaseIDs(phases []colony.Phase) []int {
	ids := make([]int, len(phases))
	for i, p := range phases {
		ids[i] = p.ID
	}
	return ids
}

func phaseNames(phases []colony.Phase) []string {
	names := make([]string, len(phases))
	for i, p := range phases {
		names[i] = p.Name
	}
	return names
}
