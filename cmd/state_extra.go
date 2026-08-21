package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var stateCheckpointCmd = &cobra.Command{
	Use:   "state-checkpoint",
	Short: "Save current COLONY_STATE.json as a named checkpoint",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		name := mustGetString(cmd, "name")
		if name == "" {
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		checkpointPath := filepath.Join("checkpoints", name+".json")
		if err := store.SaveJSON(checkpointPath, state); err != nil {
			outputError(2, fmt.Sprintf("failed to save checkpoint: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"checkpoint": name,
			"path":       checkpointPath,
		})
		return nil
	},
}

var stateWriteCmd = &cobra.Command{
	Use:   "state-write [json-blob]",
	Short: "Direct write to COLONY_STATE.json (bypasses transition validation)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		// Positional JSON mode: replace entire state file
		if len(args) > 0 {
			if !json.Valid([]byte(args[0])) {
				outputError(1, "positional argument must be valid JSON", nil)
				return nil
			}
			// Reject if --field/--value flags also provided
			field := mustGetString(cmd, "field")
			if field != "" {
				outputError(1, "cannot use both positional JSON and --field/--value flags", nil)
				return nil
			}
			if err := store.AtomicWrite("COLONY_STATE.json", []byte(args[0])); err != nil {
				outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
				return nil
			}
			outputOK(map[string]interface{}{
				"updated":  true,
				"replaced": true,
			})
			return nil
		}

		field := mustGetString(cmd, "field")
		if field == "" {
			return nil
		}
		value := mustGetString(cmd, "value")
		if value == "" {
			return nil
		}

		// Load raw COLONY_STATE.json as map for arbitrary field setting
		data, err := store.ReadFile("COLONY_STATE.json")
		if err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			outputError(1, fmt.Sprintf("failed to parse COLONY_STATE.json: %v", err), nil)
			return nil
		}

		m[field] = value

		if err := store.SaveJSON("COLONY_STATE.json", m); err != nil {
			outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"updated": true,
			"field":   field,
			"value":   value,
		})
		return nil
	},
}

var phaseInsertCmd = &cobra.Command{
	Use:     "phase-insert",
	Short:   "Insert a corrective phase into the active plan",
	Args:    cobra.NoArgs,
	Aliases: []string{"insert-phase"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		after := mustGetInt(cmd, "after")
		// mustGetString already emits an ok:false envelope and sets a non-zero
		// exit code for an empty required flag; these guards only stop the
		// command from continuing, they are not a silent return.
		name := mustGetString(cmd, "name")
		if name == "" {
			return nil
		}
		description := mustGetString(cmd, "description")
		if description == "" {
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		// Validate after index
		if after < 0 || after > len(state.Plan.Phases) {
			outputError(1, fmt.Sprintf("invalid after index %d (plan has %d phases)", after, len(state.Plan.Phases)), nil)
			return nil
		}

		newPhase := colony.Phase{
			Name:        name,
			Description: description,
			Status:      colony.PhasePending,
			Tasks:       []colony.Task{},
		}

		previousPhaseCount := len(state.Plan.Phases)

		// Insert after the specified index (0-based)
		insertAt := after
		state.Plan.Phases = append(state.Plan.Phases[:insertAt], append([]colony.Phase{newPhase}, state.Plan.Phases[insertAt:]...)...)

		// Renumber so phase.ID == index+1 holds after every insert. Production
		// call sites index phases by ordinal (phaseNum-1); a mid-slice insert
		// carrying max+1 would leave orders like [1,3,2] and misroute build
		// and continue for every phase after the insertion point.
		oldToNew := make(map[int]int, previousPhaseCount)
		for i := range state.Plan.Phases {
			if state.Plan.Phases[i].ID > 0 {
				oldToNew[state.Plan.Phases[i].ID] = i + 1
			}
			state.Plan.Phases[i].ID = i + 1
		}
		insertedID := insertAt + 1
		if mapped, ok := oldToNew[state.CurrentPhase]; ok && state.CurrentPhase > 0 {
			state.CurrentPhase = mapped
		}

		if shouldReopenInsertedPhase(state, insertAt, previousPhaseCount) {
			state.State = colony.StateREADY
			state.CurrentPhase = insertedID
			state.Plan.Phases[insertAt].Status = colony.PhaseReady
		}

		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"inserted": true,
			"phase_id": insertedID,
			"after":    after,
		})
		return nil
	},
}

func shouldReopenInsertedPhase(state colony.ColonyState, insertAt, previousPhaseCount int) bool {
	if state.State != colony.StateCOMPLETED {
		return false
	}
	if strings.TrimSpace(state.Milestone) == "Crowned Anthill" {
		return false
	}
	return insertAt == previousPhaseCount
}

var validateOracleStateCmd = &cobra.Command{
	Use:   "validate-oracle-state",
	Short: "Validate oracle-specific state structure",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		issues := []string{}
		files := map[string]bool{}

		// Path validation: oracle state must be under .aether/data/oracle/
		statePath := oracleStatePath()
		if !strings.HasPrefix(statePath, filepath.Join(".aether", "data", "oracle")) {
			issues = append(issues, fmt.Sprintf("oracle state path %q is outside .aether/data/oracle/", statePath))
		}

		// Check oracle/state.json
		stateData, err := store.ReadFile("oracle/state.json")
		if err != nil {
			files["state.json"] = false
			issues = append(issues, "oracle/state.json not found")
		} else if !json.Valid(stateData) {
			files["state.json"] = false
			issues = append(issues, "oracle/state.json is not valid JSON")
		} else {
			files["state.json"] = true
		}

		// Check oracle/plan.json
		planData, err := store.ReadFile("oracle/plan.json")
		if err != nil {
			files["plan.json"] = false
			issues = append(issues, "oracle/plan.json not found")
		} else if !json.Valid(planData) {
			files["plan.json"] = false
			issues = append(issues, "oracle/plan.json is not valid JSON")
		} else {
			files["plan.json"] = true
		}

		valid := len(issues) == 0

		outputOK(map[string]interface{}{
			"valid":  valid,
			"files":  files,
			"issues": issues,
		})
		return nil
	},
}

func init() {
	stateCheckpointCmd.Flags().String("name", "", "Checkpoint name (required)")

	stateWriteCmd.Flags().String("field", "", "Field to set (required)")
	stateWriteCmd.Flags().String("value", "", "Value to set (required)")

	phaseInsertCmd.Flags().Int("after", 0, "Insert after this phase index (0-based, required)")
	phaseInsertCmd.Flags().String("name", "", "Phase name (required)")
	phaseInsertCmd.Flags().String("description", "", "Phase description (required)")

	rootCmd.AddCommand(stateCheckpointCmd)
	rootCmd.AddCommand(stateWriteCmd)
	rootCmd.AddCommand(phaseInsertCmd)
	rootCmd.AddCommand(validateOracleStateCmd)
}
