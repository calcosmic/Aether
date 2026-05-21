package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Reason         string                 `json:"reason"`
	Headless       bool                   `json:"headless"`
	ReplanInterval int                    `json:"replan_interval"`
	Phases         []autopilotPhaseStatus `json:"phases"`
	LastUpdated    string                 `json:"last_updated"`
}

const autopilotStatePath = "autopilot/state.json"

func normalizeAutopilotPhaseStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "success":
		return "completed"
	default:
		return status
	}
}

// checkAutopilotPauseConditions inspects colony state for conditions that should
// pause autopilot advancement. Returns a non-empty reason string if a pause
// condition is triggered, empty string otherwise.
func checkAutopilotPauseConditions() string {
	if store == nil {
		return ""
	}

	// 1. Active blockers from pending-decisions.json
	var pending PendingDecisionFile
	if err := store.LoadJSON("pending-decisions.json", &pending); err == nil {
		blockerCount := 0
		for _, d := range pending.Decisions {
			if d.Type == "blocker" && !d.Resolved {
				blockerCount++
			}
		}
		if blockerCount > 0 {
			return fmt.Sprintf("active_blockers:%d", blockerCount)
		}
	}

	// 2. Test failures from colony state signals
	var cstate colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &cstate); err == nil {
		for _, signal := range cstate.Signals {
			if strings.Contains(strings.ToLower(signal.Type), "test-failure") ||
				strings.Contains(strings.ToLower(signal.Type), "test_failure") {
				return "test_failures"
			}
		}
		// Also check gate_results for failures
		for _, gr := range cstate.GateResults {
			if !gr.Passed {
				return fmt.Sprintf("gate_failure:%s", gr.Name)
			}
		}
	}

	// 3. Critical chaos findings from midden
	var middenFile struct {
		Entries []colony.MiddenEntry `json:"entries"`
	}
	if err := store.LoadJSON("midden/midden.json", &middenFile); err == nil {
		for _, entry := range middenFile.Entries {
			if strings.Contains(strings.ToLower(entry.Category), "chaos") {
				for _, tag := range entry.Tags {
					if strings.Contains(strings.ToLower(tag), "critical") {
						return "critical_chaos_findings"
					}
				}
			}
		}
	}

	// 4. Uncommitted changes marker
	if _, err := os.Stat(filepath.Join(store.BasePath(), "uncommitted-changes.marker")); err == nil {
		return "uncommitted_changes"
	}

	return ""
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
		status := normalizeAutopilotPhaseStatus(mustGetString(cmd, "status"))
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

		// Check pause conditions when a phase completes
		if status == "completed" && state.Status == "running" {
			if pauseReason := checkAutopilotPauseConditions(); pauseReason != "" {
				state.Status = "paused"
				state.Reason = pauseReason
			}
		}

		if err := store.SaveJSON(autopilotStatePath, state); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		// Trace autopilot phase transition if colony state has a run_id
		if tracer != nil {
			var cstate colony.ColonyState
			if loadErr := store.LoadJSON("COLONY_STATE.json", &cstate); loadErr == nil && cstate.RunID != nil {
				_ = tracer.LogPhaseChange(*cstate.RunID, phase, status, "autopilot-update")
			}
		}

		result := map[string]interface{}{
			"updated": true,
			"phase":   phase,
			"status":  status,
			"current": state.CurrentPhase,
			"total":   state.TotalPhases,
		}
		if state.Status == "paused" {
			result["paused"] = true
			result["reason"] = state.Reason
		}
		outputOK(result)
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

		result := map[string]interface{}{
			"active":          state.Status == "running",
			"status":          state.Status,
			"current_phase":   state.CurrentPhase,
			"total_phases":    state.TotalPhases,
			"headless":        state.Headless,
			"replan_interval": state.ReplanInterval,
			"phases":          state.Phases,
			"initialized_at":  state.InitializedAt,
			"last_updated":    state.LastUpdated,
		}
		if state.Reason != "" {
			result["reason"] = state.Reason
		}
		outputOK(result)
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
			if normalizeAutopilotPhaseStatus(p.Status) == "completed" {
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

// --- autopilot-resume ---

var autopilotResumeCmd = &cobra.Command{
	Use:   "autopilot-resume",
	Short: "Resume autopilot from paused state",
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

		if state.Status != "paused" {
			outputOK(map[string]interface{}{
				"resumed": false,
				"reason":  fmt.Sprintf("autopilot status is %q, not paused", state.Status),
			})
			return nil
		}

		state.Status = "running"
		state.Reason = ""
		state.LastUpdated = time.Now().UTC().Format(time.RFC3339)

		if err := store.SaveJSON(autopilotStatePath, state); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"resumed":       true,
			"current_phase": state.CurrentPhase,
			"total_phases":  state.TotalPhases,
		})
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
		autopilotResumeCmd,
	} {
		rootCmd.AddCommand(c)
	}
}
