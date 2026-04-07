package cmd

import (
	"fmt"
	"time"

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

const autopilotStatePath = "autopilot/state.json"

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
		state := autopilotState{
			InitializedAt:  now,
			TotalPhases:    phases,
			CurrentPhase:   0,
			Status:         "initialized",
			Headless:       false,
			ReplanInterval: 3,
			Phases:         make([]autopilotPhaseStatus, 0, phases),
			LastUpdated:    now,
		}

		if err := store.SaveJSON(autopilotStatePath, state); err != nil {
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

		var state autopilotState
		if err := store.LoadJSON(autopilotStatePath, &state); err != nil {
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

		if err := store.SaveJSON(autopilotStatePath, state); err != nil {
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

		var state autopilotState
		if err := store.LoadJSON(autopilotStatePath, &state); err != nil {
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

		var state autopilotState
		if err := store.LoadJSON(autopilotStatePath, &state); err != nil {
			outputError(1, fmt.Sprintf("autopilot not initialized: %v", err), nil)
			return nil
		}

		state.Status = "stopped"
		state.LastUpdated = time.Now().UTC().Format(time.RFC3339)

		if err := store.SaveJSON(autopilotStatePath, state); err != nil {
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

		var state autopilotState
		if err := store.LoadJSON(autopilotStatePath, &state); err != nil {
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

		var state autopilotState
		if err := store.LoadJSON(autopilotStatePath, &state); err != nil {
			outputError(1, fmt.Sprintf("autopilot not initialized: %v", err), nil)
			return nil
		}

		state.Headless = value
		state.LastUpdated = time.Now().UTC().Format(time.RFC3339)

		if err := store.SaveJSON(autopilotStatePath, state); err != nil {
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

		var state autopilotState
		if err := store.LoadJSON(autopilotStatePath, &state); err != nil {
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
