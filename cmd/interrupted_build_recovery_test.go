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
	if next != "aether build 1 --force" {
		t.Fatalf("next command = %q, want force redispatch", next)
	}
	if err := validateCodexBuildState(state, 1, nil, true); err != nil {
		t.Fatalf("recommended recovery command is rejected by build validation: %v", err)
	}
	primary, _ := workflowSuggestionsForState(state)
	if !strings.Contains(primary, "aether build 1 --force") {
		t.Fatalf("visual recovery suggestion = %q, want force redispatch", primary)
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
	if got := nextCommandFromState(state); got != "aether entomb" {
		t.Fatalf("sealed next command = %q, want aether entomb", got)
	}
	if !colonyNeedsEntomb(state) {
		t.Fatal("sealed colony was not treated as entombable")
	}
}
