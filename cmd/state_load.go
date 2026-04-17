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
	return state, nil
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
