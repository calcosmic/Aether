package cmd

import (
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

func allPhasesCompleted(state colony.ColonyState) bool {
	if len(state.Plan.Phases) == 0 {
		return false
	}
	for _, phase := range state.Plan.Phases {
		if phase.Status != colony.PhaseCompleted {
			return false
		}
	}
	return true
}

func colonyNeedsEntomb(state colony.ColonyState) bool {
	return strings.TrimSpace(state.Milestone) == "Crowned Anthill"
}

func completedPhaseCount(state colony.ColonyState) int {
	count := 0
	for _, phase := range state.Plan.Phases {
		if phase.Status == colony.PhaseCompleted {
			count++
		}
	}
	return count
}
