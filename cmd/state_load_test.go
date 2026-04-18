package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestLoadActiveColonyStateNormalizesLegacyPausedState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Normalize old paused colonies"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.State("PAUSED"),
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Legacy paused phase", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: &taskID, Goal: "Resume safely", Status: colony.TaskInProgress}}},
			},
		},
	})

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState returned error: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY", state.State)
	}
	if !state.Paused {
		t.Fatal("expected legacy PAUSED state to normalize with paused flag set")
	}
}

func TestLoadActiveColonyStateNormalizesBrokenIdleStateWithGoal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Recover broken idle colony"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateIDLE,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Restorable phase", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: &taskID, Goal: "Finish restoring", Status: colony.TaskInProgress}}},
			},
		},
	})

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState returned error: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY", state.State)
	}
}
