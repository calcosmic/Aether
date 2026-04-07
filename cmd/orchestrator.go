package cmd

import (
	"fmt"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var orchestratorDecomposeCmd = &cobra.Command{
	Use:   "orchestrator-decompose",
	Short: "Show task decomposition for a phase",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		phaseNum := mustGetInt(cmd, "phase")

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		var phase *colony.Phase
		for i := range state.Plan.Phases {
			if state.Plan.Phases[i].ID == phaseNum {
				phase = &state.Plan.Phases[i]
				break
			}
		}
		if phase == nil {
			outputError(1, fmt.Sprintf("phase %d not found", phaseNum), nil)
			return nil
		}

		graph, err := agent.BuildTaskGraph(phase.Tasks)
		if err != nil {
			outputError(1, fmt.Sprintf("build task graph: %v", err), nil)
			return nil
		}

		type taskOutput struct {
			ID        string   `json:"id"`
			Goal      string   `json:"goal"`
			TypeHint  string   `json:"type_hint"`
			Caste     string   `json:"caste"`
			DependsOn []string `json:"depends_on"`
			Status    string   `json:"status"`
		}

		nodes := graph.Nodes()
		tasks := make([]taskOutput, 0, len(nodes))
		for _, n := range nodes {
			tasks = append(tasks, taskOutput{
				ID:        n.ID,
				Goal:      n.Goal,
				TypeHint:  n.TypeHint,
				Caste:     string(n.Caste),
				DependsOn: n.DependsOn,
				Status:    n.Status,
			})
		}

		outputOK(map[string]interface{}{
			"phase":      phaseNum,
			"task_count": len(tasks),
			"tasks":      tasks,
		})
		return nil
	},
}

var orchestratorAssignCmd = &cobra.Command{
	Use:   "orchestrator-assign",
	Short: "Show caste assignments for a phase",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		phaseNum := mustGetInt(cmd, "phase")

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		var phase *colony.Phase
		for i := range state.Plan.Phases {
			if state.Plan.Phases[i].ID == phaseNum {
				phase = &state.Plan.Phases[i]
				break
			}
		}
		if phase == nil {
			outputError(1, fmt.Sprintf("phase %d not found", phaseNum), nil)
			return nil
		}

		graph, err := agent.BuildTaskGraph(phase.Tasks)
		if err != nil {
			outputError(1, fmt.Sprintf("build task graph: %v", err), nil)
			return nil
		}

		type assignOutput struct {
			TaskID string `json:"task_id"`
			Goal   string `json:"goal"`
			Caste  string `json:"caste"`
			Status string `json:"status"`
		}

		nodes := graph.Nodes()
		assignments := make([]assignOutput, 0, len(nodes))
		for _, n := range nodes {
			assignments = append(assignments, assignOutput{
				TaskID: n.ID,
				Goal:   n.Goal,
				Caste:  string(n.Caste),
				Status: n.Status,
			})
		}

		outputOK(map[string]interface{}{
			"phase":       phaseNum,
			"assignments": assignments,
		})
		return nil
	},
}

var orchestratorStatusCmd = &cobra.Command{
	Use:   "orchestrator-status",
	Short: "Show orchestrator state and progress",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		if state.OrchestratorState == nil {
			outputOK(map[string]interface{}{
				"active": false,
				"status": "idle",
				"reason": "no orchestration in progress",
			})
			return nil
		}

		outputOK(state.OrchestratorState)
		return nil
	},
}

func init() {
	orchestratorDecomposeCmd.Flags().Int("phase", 0, "Phase number (required)")
	orchestratorAssignCmd.Flags().Int("phase", 0, "Phase number (required)")

	rootCmd.AddCommand(orchestratorDecomposeCmd)
	rootCmd.AddCommand(orchestratorAssignCmd)
	rootCmd.AddCommand(orchestratorStatusCmd)
}
