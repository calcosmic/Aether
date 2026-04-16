package cmd

import (
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var memoryMetricsCmd = &cobra.Command{
	Use:     "memory-metrics",
	Short:   "Show memory health metrics",
	Aliases: []string{"memory-details"},
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		// Load learning observations
		wisdomTotal := 0
		var lastLearning, lastCapture string
		var learnings colony.LearningFile
		if err := store.LoadJSON("learning-observations.json", &learnings); err == nil {
			wisdomTotal = len(learnings.Observations)
			if wisdomTotal > 0 {
				lastCapture = learnings.Observations[wisdomTotal-1].LastSeen
			}
		}

		// Load midden for failure count
		failureCount := 0
		var lastFailure string
		var midden colony.MiddenFile
		if err := store.LoadJSON("midden/midden.json", &midden); err == nil {
			failureCount = len(midden.Entries)
			if failureCount > 0 {
				lastFailure = midden.Entries[failureCount-1].Timestamp
			}
		}

		// Compute pending promotions (observations that haven't been promoted)
		pendingTotal := 0
		for _, obs := range learnings.Observations {
			if obs.TrustScore == nil {
				pendingTotal++
			}
		}

		result := map[string]interface{}{
			"wisdom": map[string]interface{}{
				"total": wisdomTotal,
			},
			"pending": map[string]interface{}{
				"total": pendingTotal,
			},
			"recent_failures": map[string]interface{}{
				"count": failureCount,
			},
			"last_activity": map[string]interface{}{
				"queen_md_updated":  lastLearning,
				"learning_captured": lastCapture,
				"last_failure":      lastFailure,
			},
		}
		outputOK(result)
		return nil
	},
}

var colonyVitalSignsCmd = &cobra.Command{
	Use:     "colony-vital-signs",
	Short:   "Show colony vital signs",
	Aliases: []string{"patrol"},
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		// Compute vital signs from available data
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputErrorMessage("failed to load colony state")
			return nil
		}

		// Count active pheromones for signal health
		signalCount := 0
		var pf colony.PheromoneFile
		if err := store.LoadJSON("pheromones.json", &pf); err == nil {
			for _, sig := range pf.Signals {
				if sig.Active {
					signalCount++
				}
			}
		}

		// Compute basic metrics
		instinctCount := len(state.Memory.Instincts)
		errorCount := len(state.Errors.Records)
		completedPhases := 0
		for _, phase := range state.Plan.Phases {
			if phase.Status == "completed" {
				completedPhases++
			}
		}

		// Derive health score (0-100)
		healthScore := 50 // baseline
		if instinctCount > 0 {
			healthScore += 10
		}
		if signalCount > 0 {
			healthScore += 10
		}
		if errorCount == 0 {
			healthScore += 15
		}
		if completedPhases > 0 {
			healthScore += 15
		}
		if healthScore > 100 {
			healthScore = 100
		}

		// Health label
		healthLabel := "Critical"
		switch {
		case healthScore >= 80:
			healthLabel = "Thriving"
		case healthScore >= 60:
			healthLabel = "Healthy"
		case healthScore >= 40:
			healthLabel = "Stable"
		case healthScore >= 20:
			healthLabel = "Struggling"
		}

		signalStatus := "none"
		if signalCount > 0 {
			signalStatus = "active"
		}

		memoryStatus := "low"
		if instinctCount > 5 {
			memoryStatus = "normal"
		} else if instinctCount > 0 {
			memoryStatus = "building"
		}

		result := map[string]interface{}{
			"build_velocity": map[string]interface{}{
				"phases_per_day": 0,
				"trend":          "starting",
			},
			"error_rate": map[string]interface{}{
				"errors_per_day": 0,
				"status":         "clean",
			},
			"signal_health": map[string]interface{}{
				"active_count": signalCount,
				"status":       signalStatus,
			},
			"memory_pressure": map[string]interface{}{
				"instinct_count": instinctCount,
				"status":         memoryStatus,
			},
			"colony_age_hours": 0,
			"overall_health":   healthScore,
			"health_label":     healthLabel,
		}
		outputWorkflow(result, renderPatrolVisual(result))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(memoryMetricsCmd)
	rootCmd.AddCommand(colonyVitalSignsCmd)
}
