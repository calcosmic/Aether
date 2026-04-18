package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

var errNoColonyInitialized = errors.New("no colony initialized")

func loadActiveColonyState() (colony.ColonyState, error) {
	if store == nil {
		return colony.ColonyState{}, fmt.Errorf("no store initialized")
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return colony.ColonyState{}, errNoColonyInitialized
		}
		return colony.ColonyState{}, fmt.Errorf("failed to load colony state: %w", err)
	}
	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
		return colony.ColonyState{}, errNoColonyInitialized
	}
	return normalizeLegacyColonyState(state), nil
}

func normalizeLegacyColonyState(state colony.ColonyState) colony.ColonyState {
	rawState := strings.ToUpper(strings.TrimSpace(string(state.State)))
	hasGoal := state.Goal != nil && strings.TrimSpace(*state.Goal) != ""
	hasPlanContext := len(state.Plan.Phases) > 0 || state.CurrentPhase > 0

	switch rawState {
	case "PAUSED":
		state.State = colony.StateREADY
		state.Paused = true
	case "PLANNED", "PLANNING":
		state.State = colony.StateREADY
	case "SEALED":
		state.State = colony.StateCOMPLETED
	case "ENTOMBED":
		state.State = colony.StateIDLE
	case "IDLE":
		if hasGoal && hasPlanContext {
			state.State = colony.StateREADY
		}
	}

	return state
}

func colonyStateLoadMessage(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, errNoColonyInitialized) {
		return `No colony initialized. Run ` + "`aether init \"goal\"`" + ` first.`
	}
	return fmt.Sprintf("Failed to load colony state: %v", err)
}
