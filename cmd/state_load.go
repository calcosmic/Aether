package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

var errNoColonyInitialized = errors.New("no colony initialized")

func loadActiveColonyState() (colony.ColonyState, error) {
	if store == nil {
		return colony.ColonyState{}, fmt.Errorf("no store initialized")
	}

	state, err := loadColonyStateWithCompatibilityRepair()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return colony.ColonyState{}, errNoColonyInitialized
		}
		return colony.ColonyState{}, fmt.Errorf("failed to load colony state: %w", err)
	}
	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
		return colony.ColonyState{}, errNoColonyInitialized
	}
	state = normalizeLegacyColonyState(state)
	repaired, _, err := repairMissingPlanFromArtifacts(state)
	if err != nil {
		return colony.ColonyState{}, err
	}
	return repaired, nil
}

func loadColonyStateWithCompatibilityRepair() (colony.ColonyState, error) {
	var state colony.ColonyState
	loadErr := store.LoadJSON("COLONY_STATE.json", &state)
	if loadErr == nil {
		return state, nil
	}

	raw, rawErr := store.LoadRawJSON("COLONY_STATE.json")
	if rawErr != nil {
		return colony.ColonyState{}, loadErr
	}

	repairedRaw, repaired, repairErr := repairLegacyNumericStringFields(raw)
	if repairErr != nil || !repaired {
		return colony.ColonyState{}, loadErr
	}
	if err := json.Unmarshal(repairedRaw, &state); err != nil {
		return colony.ColonyState{}, loadErr
	}

	state.Events = append(trimmedEvents(state.Events),
		fmt.Sprintf("%s|state_repaired|load|Normalized legacy numeric string fields in COLONY_STATE.json", time.Now().UTC().Format(time.RFC3339)),
	)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		return colony.ColonyState{}, fmt.Errorf("failed to persist repaired colony state: %w", err)
	}
	return state, nil
}

func repairLegacyNumericStringFields(raw []byte) ([]byte, bool, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, false, err
	}

	rawPhase, ok := fields["current_phase"]
	if !ok {
		return raw, false, nil
	}

	var phaseText string
	if err := json.Unmarshal(rawPhase, &phaseText); err != nil {
		return raw, false, nil
	}

	phase, err := strconv.Atoi(strings.TrimSpace(phaseText))
	if err != nil {
		return raw, false, nil
	}
	fields["current_phase"] = json.RawMessage([]byte(strconv.Itoa(phase)))

	repaired, err := json.Marshal(fields)
	if err != nil {
		return nil, false, err
	}
	return repaired, true, nil
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
