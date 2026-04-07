package cmd

import (
	"fmt"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// Autopilot manages automated build-verify-advance cycles.

type autopilotPhaseStatus struct {
	Phase  int    `json:"phase"`
	Status string `json:"status"`
	At     string `json:"at,omitempty"`
}

type autopilotState struct {
	InitializedAt  string                 `json:"initialized_at"`
	TotalPhases    int                    `json:"total_phases"`
	CurrentPhase   int                    `json:"current_phase"`
	Status         string                 `json:"status"` // running, paused, stopped, completed
	Headless       bool                   `json:"headless"`
	ReplanInterval int                    `json:"replan_interval"`
	Phases         []autopilotPhaseStatus `json:"phases"`
	LastUpdated    string                 `json:"last_updated"`
}

const colonyStatePath = "COLONY_STATE.json"

func loadAutopilotFromColony() (*autopilotState, error) {
	var state colony.ColonyState
	if err := store.LoadJSON(colonyStatePath, &state); err != nil {
		return nil, err
	}
	if state.OrchestratorState == nil {
		return nil, fmt.Errorf("orchestrator not initialized")
	}
	phases := make([]autopilotPhaseStatus, 0)
	ap := &autopilotState{
		CurrentPhase:   state.OrchestratorState.Phase,
		Status:         state.OrchestratorState.Status,
		TotalPhases:    state.OrchestratorState.TaskCount,
		Headless:       state.OrchestratorState.Headless,
		ReplanInterval: state.OrchestratorState.ReplanInterval,
		InitializedAt:  state.OrchestratorState.StartedAt,
		LastUpdated:    state.OrchestratorState.UpdatedAt,
		Phases:         phases,
	}
	return ap, nil
}

func saveAutopilotToColony(ap *autopilotState) error {
	var state colony.ColonyState
	if err := store.LoadJSON(colonyStatePath, &state); err != nil {
		return err
	}
	if state.OrchestratorState == nil {
		state.OrchestratorState = &colony.OrchestratorState{}
	}
	state.OrchestratorState.Phase = ap.CurrentPhase
	state.OrchestratorState.Status = ap.Status
	state.OrchestratorState.TaskCount = ap.TotalPhases
	state.OrchestratorState.UpdatedAt = ap.LastUpdated
	state.OrchestratorState.Headless = ap.Headless
	state.OrchestratorState.ReplanInterval = ap.ReplanInterval
	state.OrchestratorState.StartedAt = ap.InitializedAt
	return store.SaveJSON(colonyStatePath, state)
}

// --- autopilot-init ---

var autopilotInitCmd = &cobra.Command{
	Use:   "autopilot-init",
	Short: "Initialize autopilot state for N phases",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		phases := mustGetInt(cmd, "phases")

		now := time.Now().UTC().Format(time.RFC3339)

		// Ensure COLONY_STATE.json exists with minimal state
		var colonyState colony.ColonyState
		if err := store.LoadJSON(colonyStatePath, &colonyState); err != nil {
			colonyState = colony.ColonyState{
				Version: "1.0",
				State:   colony.StateREADY,
			}
		}
		colonyState.OrchestratorState = &colony.OrchestratorState{
			Phase:          0,
			Status:         "initialized",
			TaskCount:      phases,
			StartedAt:      now,
			UpdatedAt:      now,
			Headless:       false,
			ReplanInterval: 3,
		}
		if err := store.SaveJSON(colonyStatePath, colonyState); err != nil {
			outputError(2, fmt.Sprintf("failed to save autopilot state: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"initialized":  true,
			"total_phases": phases,
			"status":       "initialized",
		})
		return nil
	},
}

// --- autopilot-update ---

var autopilotUpdateCmd = &cobra.Command{
	Use:   "autopilot-update",
	Short: "Update autopilot phase status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		phase := mustGetInt(cmd, "phase")
		status := mustGetString(cmd, "status")
		if status == "" {
			return nil
		}

		state, err := loadAutopilotFromColony()
		if err != nil {
			outputError(1, fmt.Sprintf("autopilot not initialized: %v", err), nil)
			return nil
		}

		now := time.Now().UTC().Format(time.RFC3339)
		ps := autopilotPhaseStatus{Phase: phase, Status: status, At: now}

		found := false
		for i, p := range state.Phases {
			if p.Phase == phase {
				state.Phases[i] = ps
				found = true
				break
			}
		}
		if !found {
			state.Phases = append(state.Phases, ps)
		}

		state.CurrentPhase = phase
		state.LastUpdated = now

		if status == "completed" && phase >= state.TotalPhases {
			state.Status = "completed"
		} else if state.Status == "initialized" {
			state.Status = "running"
		}

		if err := saveAutopilotToColony(state); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"updated": true,
			"phase":   phase,
			"status":  status,
			"current": state.CurrentPhase,
			"total":   state.TotalPhases,
		})
		return nil
	},
}

// --- autopilot-status ---

var autopilotStatusCmd = &cobra.Command{
	Use:   "autopilot-status",
	Short: "Return current autopilot state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		state, err := loadAutopilotFromColony()
		if err != nil {
			outputOK(map[string]interface{}{"active": false, "reason": "not initialized"})
			return nil
		}

		outputOK(map[string]interface{}{
			"active":          state.Status == "running",
			"status":          state.Status,
			"current_phase":   state.CurrentPhase,
			"total_phases":    state.TotalPhases,
			"headless":        state.Headless,
			"replan_interval": state.ReplanInterval,
			"phases":          state.Phases,
			"initialized_at":  state.InitializedAt,
			"last_updated":    state.LastUpdated,
		})
		return nil
	},
}

// --- autopilot-stop ---

var autopilotStopCmd = &cobra.Command{
	Use:   "autopilot-stop",
	Short: "Stop autopilot and save state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		state, err := loadAutopilotFromColony()
		if err != nil {
			outputError(1, fmt.Sprintf("autopilot not initialized: %v", err), nil)
			return nil
		}

		state.Status = "stopped"
		state.LastUpdated = time.Now().UTC().Format(time.RFC3339)

		if err := saveAutopilotToColony(state); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"stopped":       true,
			"current_phase": state.CurrentPhase,
			"total_phases":  state.TotalPhases,
		})
		return nil
	},
}

// --- autopilot-check-replan ---

var autopilotCheckReplanCmd = &cobra.Command{
	Use:   "autopilot-check-replan",
	Short: "Check if replan is recommended",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		interval := mustGetInt(cmd, "interval")

		state, err := loadAutopilotFromColony()
		if err != nil {
			outputOK(map[string]interface{}{"replan": false, "reason": "not initialized"})
			return nil
		}

		if state.Status != "running" {
			outputOK(map[string]interface{}{"replan": false, "reason": "autopilot not running"})
			return nil
		}

		completedPhases := 0
		for _, p := range state.Phases {
			if p.Status == "completed" {
				completedPhases++
			}
		}

		if interval > 0 && completedPhases > 0 && completedPhases%interval == 0 {
			outputOK(map[string]interface{}{
				"replan":         true,
				"reason":         "interval_reached",
				"completed":      completedPhases,
				"interval":       interval,
				"next_replan_at": completedPhases + interval,
			})
		} else {
			nextReplan := completedPhases + interval - (completedPhases % interval)
			if interval == 0 || completedPhases == 0 {
				nextReplan = interval
			}
			outputOK(map[string]interface{}{
				"replan":         false,
				"completed":      completedPhases,
				"interval":       interval,
				"next_replan_at": nextReplan,
			})
		}
		return nil
	},
}

// --- autopilot-set-headless ---

var autopilotSetHeadlessCmd = &cobra.Command{
	Use:   "autopilot-set-headless",
	Short: "Set headless mode",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		value := mustGetBool(cmd, "value")

		state, err := loadAutopilotFromColony()
		if err != nil {
			outputError(1, fmt.Sprintf("autopilot not initialized: %v", err), nil)
			return nil
		}

		state.Headless = value
		state.LastUpdated = time.Now().UTC().Format(time.RFC3339)

		if err := saveAutopilotToColony(state); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"headless": value})
		return nil
	},
}

// --- autopilot-headless-check ---

var autopilotHeadlessCheckCmd = &cobra.Command{
	Use:   "autopilot-headless-check",
	Short: "Check if running in headless mode",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		state, err := loadAutopilotFromColony()
		if err != nil {
			outputOK(map[string]interface{}{"headless": false, "reason": "not initialized"})
			return nil
		}

		outputOK(map[string]interface{}{"headless": state.Headless})
		return nil
	},
}

func init() {
	autopilotInitCmd.Flags().Int("phases", 0, "Number of phases (required)")
	autopilotUpdateCmd.Flags().Int("phase", 0, "Phase number (required)")
	autopilotUpdateCmd.Flags().String("status", "", "Phase status (required)")
	autopilotCheckReplanCmd.Flags().Int("interval", 3, "Replan interval in phases")
	autopilotSetHeadlessCmd.Flags().Bool("value", false, "Headless mode value")

	for _, c := range []*cobra.Command{
		autopilotInitCmd, autopilotUpdateCmd, autopilotStatusCmd,
		autopilotStopCmd, autopilotCheckReplanCmd,
		autopilotSetHeadlessCmd, autopilotHeadlessCheckCmd,
	} {
		rootCmd.AddCommand(c)
	}
}
