package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestInterruptedBuildRecoveryCommandMatchesBuildValidator(t *testing.T) {
	taskID := "1.1"
	state := colony.ColonyState{
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{
			{
				ID:     1,
				Name:   "Interrupted phase",
				Status: colony.PhaseInProgress,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Resume safely", Status: colony.TaskInProgress}},
			},
		}},
	}

	next := nextCommandFromState(state)
	if next != "aether resume" {
		t.Fatalf("next command = %q, want canonical resume", next)
	}
	primary, _ := workflowSuggestionsForState(state)
	if !strings.Contains(primary, "aether resume") {
		t.Fatalf("visual recovery suggestion = %q, want canonical resume", primary)
	}
}

func TestCompletedColonyMustSealBeforeEntomb(t *testing.T) {
	state := colony.ColonyState{
		State: colony.StateCOMPLETED,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Status: colony.PhaseCompleted,
		}}},
	}
	if got := nextCommandFromState(state); got != "aether seal" {
		t.Fatalf("completed unsealed next command = %q, want aether seal", got)
	}
	if colonyNeedsEntomb(state) {
		t.Fatal("completed unsealed colony was treated as entombable")
	}

	state.Milestone = "Crowned Anthill"
	if got := nextCommandFromState(state); got != "aether status" {
		t.Fatalf("sealed next command = %q, want status-first review", got)
	}
	if !colonyNeedsEntomb(state) {
		t.Fatal("sealed colony was not treated as entombable")
	}
}
