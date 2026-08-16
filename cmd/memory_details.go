package cmd

import (
	"math"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
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

		summary := loadMemoryHealthSummary(store)

		result := map[string]interface{}{
			"wisdom": map[string]interface{}{
				"total": summary.WisdomTotal,
			},
			"pending": map[string]interface{}{
				"total": summary.PendingPromotions,
			},
			"instincts": map[string]interface{}{
				"active":    summary.ActiveInstincts,
				"archived":  summary.ArchivedInstincts,
				"applied":   summary.AppliedInstincts,
				"last_used": summary.LastInstinctTouched,
			},
			"curation": map[string]interface{}{
				"review_candidates": summary.ReviewCandidates,
				"reread_candidates": summary.RereadCandidates,
			},
			"recent_failures": map[string]interface{}{
				"count": summary.RecentFailures,
			},
			"last_activity": map[string]interface{}{
				"queen_md_updated":  "",
				"learning_captured": summary.LastLearning,
				"last_failure":      summary.LastFailure,
				"instinct_used":     summary.LastInstinctTouched,
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

		result := computeColonyVitalSigns(store, state)
		outputWorkflow(result, renderPatrolVisual(result))
		return nil
	},
}

// computeColonyVitalSigns derives the colony health snapshot. Extracted from
// the colony-vital-signs command so `aether status` can show the same numbers
// — v5.4.0's status called this data and, until Phase-168-spirit wiring, the
// modern status never did.
func computeColonyVitalSigns(s *storage.Store, state colony.ColonyState) map[string]interface{} {
	// Count active pheromones for signal health
	signalCount := 0
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err == nil {
		for _, sig := range pf.Signals {
			if sig.Active {
				signalCount++
			}
		}
	}

	// Compute basic metrics
	instinctCount := activeInstinctCount(s, &state)
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

	// Age, velocity and error rate come from state the function already
	// holds — they were placeholder zeros until the health breakdown gained
	// a render (SEE-01), which made the gap visible.
	ageHours := 0.0
	if state.InitializedAt != nil {
		ageHours = time.Since(*state.InitializedAt).Hours()
		if ageHours < 0 {
			ageHours = 0
		}
	}
	ageDays := ageHours / 24
	phasesPerDay := 0.0
	errorsPerDay := 0.0
	velocityTrend := "starting"
	if ageDays >= 1 {
		phasesPerDay = float64(completedPhases) / ageDays
		errorsPerDay = float64(errorCount) / ageDays
		velocityTrend = "steady"
	} else if completedPhases > 0 {
		phasesPerDay = float64(completedPhases)
		errorsPerDay = float64(errorCount)
		velocityTrend = "fresh"
	}
	errorStatus := "clean"
	if errorCount > 0 {
		errorStatus = "recorded"
	}

	return map[string]interface{}{
		"build_velocity": map[string]interface{}{
			"phases_per_day": math.Round(phasesPerDay*10) / 10,
			"trend":          velocityTrend,
		},
		"error_rate": map[string]interface{}{
			"errors_per_day": math.Round(errorsPerDay*10) / 10,
			"status":         errorStatus,
		},
		"signal_health": map[string]interface{}{
			"active_count": signalCount,
			"status":       signalStatus,
		},
		"memory_pressure": map[string]interface{}{
			"instinct_count": instinctCount,
			"status":         memoryStatus,
		},
		"colony_age_hours": math.Round(ageHours*10) / 10,
		"overall_health":   healthScore,
		"health_label":     healthLabel,
	}
}

func init() {
	rootCmd.AddCommand(memoryMetricsCmd)
	rootCmd.AddCommand(colonyVitalSignsCmd)
}
