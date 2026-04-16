package cmd

import (
	"fmt"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var (
	phaseNumber int
	phaseJSON   bool
)

var phaseCmd = &cobra.Command{
	Use:   "phase",
	Short: "Display current phase details",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputErrorMessage("failed to load colony state")
			return nil
		}

		// Determine which phase to display
		phaseNum := phaseNumber
		if phaseNum == 0 {
			phaseNum = state.CurrentPhase
		}

		// Find the phase
		if phaseNum <= 0 || phaseNum > len(state.Plan.Phases) {
			outputErrorMessage(fmt.Sprintf("phase %d not found (plan has %d phases)", phaseNum, len(state.Plan.Phases)))
			return nil
		}

		phase := state.Plan.Phases[phaseNum-1]

		if phaseJSON {
			outputOK(buildPhaseResult(phase, phaseNum, len(state.Plan.Phases)))
			return nil
		}

		result := buildPhaseResult(phase, phaseNum, len(state.Plan.Phases))
		outputWorkflow(result, renderPhaseVisual(result))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(phaseCmd)
	phaseCmd.Flags().IntVar(&phaseNumber, "number", 0, "Phase number to display (default: current phase)")
	phaseCmd.Flags().BoolVar(&phaseJSON, "json", false, "Output as JSON")
}

// renderPhaseDetails displays phase details with a task list table.
func buildPhaseResult(phase colony.Phase, num, total int) map[string]interface{} {
	tasks := make([]map[string]interface{}, 0, len(phase.Tasks))
	completed := 0
	for _, t := range phase.Tasks {
		id := ""
		if t.ID != nil {
			id = *t.ID
		}
		if t.Status == colony.TaskCompleted {
			completed++
		}
		tasks = append(tasks, map[string]interface{}{
			"id":     id,
			"goal":   t.Goal,
			"status": t.Status,
		})
	}

	pct := 0
	if len(tasks) > 0 {
		pct = completed * 100 / len(tasks)
	}

	return map[string]interface{}{
		"number":       num,
		"total_phases": total,
		"name":         phase.Name,
		"status":       phase.Status,
		"description":  phase.Description,
		"tasks":        tasks,
		"completed":    completed,
		"task_count":   len(tasks),
		"progress_pct": pct,
	}
}
