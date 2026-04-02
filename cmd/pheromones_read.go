package cmd

import (
	"github.com/aether-colony/aether/pkg/colony"
	"github.com/spf13/cobra"
)

var pheromoneReadCmd = &cobra.Command{
	Use:   "pheromone-read",
	Short: "Read pheromone signals",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var pf colony.PheromoneFile
		if err := store.LoadJSON("pheromones.json", &pf); err != nil {
			outputOK(map[string]interface{}{
				"signals": []interface{}{},
			})
			return nil
		}

		// Filter to active signals only
		var active []colony.PheromoneSignal
		for _, sig := range pf.Signals {
			if sig.Active {
				active = append(active, sig)
			}
		}
		if active == nil {
			active = []colony.PheromoneSignal{}
		}

		outputOK(map[string]interface{}{
			"signals": active,
		})
		return nil
	},
}

var pheromoneCountCmd = &cobra.Command{
	Use:   "pheromone-count",
	Short: "Count active pheromone signals",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var pf colony.PheromoneFile
		if err := store.LoadJSON("pheromones.json", &pf); err != nil {
			outputOK(map[string]interface{}{
				"FOCUS":    0,
				"REDIRECT": 0,
				"FEEDBACK": 0,
			})
			return nil
		}

		counts := map[string]int{
			"FOCUS":    0,
			"REDIRECT": 0,
			"FEEDBACK": 0,
		}
		for _, sig := range pf.Signals {
			if sig.Active {
				counts[sig.Type]++
			}
		}

		outputOK(counts)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pheromoneReadCmd)
	rootCmd.AddCommand(pheromoneCountCmd)
}
