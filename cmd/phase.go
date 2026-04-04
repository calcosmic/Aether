package cmd

import (
	"fmt"
	"strings"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/jedib0t/go-pretty/v6/table"
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
			type taskEntry struct {
				ID     string `json:"id"`
				Goal   string `json:"goal"`
				Status string `json:"status"`
			}
			var tasks []taskEntry
			for _, t := range phase.Tasks {
				id := ""
				if t.ID != nil {
					id = *t.ID
				}
				tasks = append(tasks, taskEntry{ID: id, Goal: t.Goal, Status: t.Status})
			}
			if tasks == nil {
				tasks = []taskEntry{}
			}
			outputOK(map[string]interface{}{
				"name":        phase.Name,
				"status":      phase.Status,
				"description": phase.Description,
				"tasks":       tasks,
			})
			return nil
		}

		renderPhaseDetails(phase, phaseNum)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(phaseCmd)
	phaseCmd.Flags().IntVar(&phaseNumber, "number", 0, "Phase number to display (default: current phase)")
	phaseCmd.Flags().BoolVar(&phaseJSON, "json", false, "Output as JSON")
}

// renderPhaseDetails displays phase details with a task list table.
func renderPhaseDetails(phase colony.Phase, num int) {
	var b strings.Builder

	fmt.Fprintf(&b, "Phase %d: %s\n", num, phase.Name)
	fmt.Fprintf(&b, "Status: %s\n", phase.Status)
	if phase.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", phase.Description)
	}
	b.WriteString("\n")

	// Task table
	if len(phase.Tasks) > 0 {
		completed := 0
		for _, task := range phase.Tasks {
			if task.Status == "completed" {
				completed++
			}
		}
		pct := 0
		if len(phase.Tasks) > 0 {
			pct = completed * 100 / len(phase.Tasks)
		}
		fmt.Fprintf(&b, "Tasks (%d/%d completed, %d%%)\n", completed, len(phase.Tasks), pct)

		t := table.NewWriter()
		t.AppendHeader(table.Row{"ID", "Goal", "Status"})

		for _, task := range phase.Tasks {
			id := ""
			if task.ID != nil {
				id = *task.ID
			}
			t.AppendRow(table.Row{id, task.Goal, task.Status})
		}
		t.SetStyle(table.StyleRounded)
		b.WriteString(t.Render() + "\n")
	} else {
		b.WriteString("No tasks defined for this phase.\n")
	}

	fmt.Fprint(stdout, b.String())
}
