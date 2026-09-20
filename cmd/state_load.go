package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

var errNoColonyInitialized = errors.New("no colony initialized")

// loadActiveColonyStateReadOnly applies compatibility and missing-plan repairs
// in memory so callers can validate the state without authorizing a write.
func loadActiveColonyStateReadOnly() (colony.ColonyState, error) {
	if store == nil {
		return colony.ColonyState{}, fmt.Errorf("no store initialized")
	}

	state, _, err := loadColonyStateWithCompatibilityRepairReadOnly()
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
	staged, _, err := stageMissingPlanFromArtifacts(state)
	if err != nil {
		return colony.ColonyState{}, err
	}
	migrated, err := migrateLoadedPlanningState(staged)
	if err != nil {
		return colony.ColonyState{}, err
	}
	return migrated, nil
}

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
	migrated, err := migrateLoadedPlanningState(repaired)
	if err != nil {
		return colony.ColonyState{}, err
	}
	return migrated, nil
}

// migrateLoadedPlanningState is deliberately an in-memory boundary. Commands
// that only inspect or preview state must remain byte-for-byte read-only; a
// later state-changing transaction persists the returned classification through
// its existing safe writer along with the command's actual lifecycle change.
func migrateLoadedPlanningState(state colony.ColonyState) (colony.ColonyState, error) {
	if store == nil {
		return colony.ColonyState{}, fmt.Errorf("no store initialized")
	}
	root, err := planningRepositoryRoot(store.BasePath())
	if err != nil {
		return colony.ColonyState{}, err
	}
	migration, err := migratePlanningState(root, state)
	if err != nil {
		return colony.ColonyState{}, fmt.Errorf("failed to migrate planning state: %w", err)
	}
	return migration.State, nil
}

func planningRepositoryRoot(dataRoot string) (string, error) {
	dataRoot = filepath.Clean(strings.TrimSpace(dataRoot))
	if dataRoot == "." || filepath.Base(dataRoot) != "data" || filepath.Base(filepath.Dir(dataRoot)) != ".aether" {
		return "", fmt.Errorf("failed to migrate planning state: colony data root is not repository-relative .aether/data: %s", dataRoot)
	}
	return filepath.Dir(filepath.Dir(dataRoot)), nil
}

func loadColonyStateWithCompatibilityRepair() (colony.ColonyState, error) {
	state, repaired, err := loadColonyStateWithCompatibilityRepairReadOnly()
	if err != nil || !repaired {
		return state, err
	}

	state.Events = append(trimmedEvents(state.Events),
		fmt.Sprintf("%s|state_repaired|load|Normalized legacy numeric string fields in COLONY_STATE.json", time.Now().UTC().Format(time.RFC3339)),
	)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		return colony.ColonyState{}, fmt.Errorf("failed to persist repaired colony state: %w", err)
	}
	return state, nil
}

// loadColonyStateWithCompatibilityRepairReadOnly decodes the one supported
// legacy numeric-string shape without appending an event or saving the result.
func loadColonyStateWithCompatibilityRepairReadOnly() (colony.ColonyState, bool, error) {
	if store == nil {
		return colony.ColonyState{}, false, fmt.Errorf("no store initialized")
	}
	return loadColonyStateWithCompatibilityRepairReadOnlyFromPath(filepath.Join(store.BasePath(), "COLONY_STATE.json"))
}

// loadColonyStateWithCompatibilityRepairReadOnlyFromPath is the byte-level
// compatibility boundary used by orientation reads. It deliberately bypasses
// storage.Store's locking API: acquiring one of those locks can create lock
// files, which makes an otherwise logical read mutate repository metadata.
// Legacy numeric strings are normalized only in the returned value.
func loadColonyStateWithCompatibilityRepairReadOnlyFromPath(path string) (colony.ColonyState, bool, error) {
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		return colony.ColonyState{}, false, readErr
	}

	var state colony.ColonyState
	loadErr := json.Unmarshal(raw, &state)
	if loadErr == nil {
		return state, false, nil
	}

	repairedRaw, repaired, repairErr := repairLegacyNumericStringFields(raw)
	if repairErr != nil || !repaired {
		return colony.ColonyState{}, false, loadErr
	}
	if err := json.Unmarshal(repairedRaw, &state); err != nil {
		return colony.ColonyState{}, false, loadErr
	}
	return state, true, nil
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
	state.Plan.EvidencePolicy = inferredPlanEvidencePolicy(state.Plan)

	if rawState == "" {
		if hasGoal || hasPlanContext {
			state.State = colony.StateREADY
		} else {
			state.State = colony.StateIDLE
		}
		return state
	}

	switch rawState {
	case "IDLE":
		state.State = colony.StateIDLE
		if hasGoal && hasPlanContext {
			state.State = colony.StateREADY
		}
	case "READY":
		state.State = colony.StateREADY
	case "EXECUTING":
		state.State = colony.StateEXECUTING
	case "BUILT":
		state.State = colony.StateBUILT
	case "COMPLETED":
		state.State = colony.StateCOMPLETED
	case "PAUSED":
		state.State = colony.StateREADY
		state.Paused = true
	case "PLANNED", "PLANNING":
		state.State = colony.StateREADY
	case "SEALED":
		state.State = colony.StateCOMPLETED
	case "ENTOMBED":
		state.State = colony.StateIDLE
	case "DONE", "COMPLETE", "FINISHED":
		state.State = colony.StateCOMPLETED
	case "ACTIVE", "RUNNING":
		state.State = colony.StateEXECUTING
	case "BUILD":
		state.State = colony.StateBUILT
	}

	if !isValidColonyLifecycleState(state.State) {
		if hasGoal || hasPlanContext {
			state.State = colony.StateREADY
		} else {
			state.State = colony.StateIDLE
		}
	}

	return state
}

func isValidColonyLifecycleState(state colony.State) bool {
	switch state {
	case colony.StateIDLE, colony.StateREADY, colony.StateEXECUTING, colony.StateBUILT, colony.StateCOMPLETED:
		return true
	default:
		return false
	}
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
