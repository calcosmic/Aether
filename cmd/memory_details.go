package cmd

import (
	"math"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

var memoryMetricsCmd = &cobra.Command{
	Use:   "memory-metrics",
	Short: "Show memory health metrics",
	Args:  cobra.NoArgs,
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

// memoryDetailsCmd is the rendered drill-down: the actual wisdom sentences,
// pending and deferred lessons, and recent failures -- not the bare counts
// memory-metrics returns. It used to be a plain alias of memory-metrics with
// no renderer of its own; it is now its own command with its own content.
var memoryDetailsCmd = &cobra.Command{
	Use:   "memory-details",
	Short: "Show what the colony has learned: wisdom, pending lessons, and recent failures",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		details := buildMemoryDetails(store)
		result := memoryDetailsResultMap(details)

		if jsonOutput {
			outputOK(result)
			return nil
		}
		outputWorkflow(result, renderMemoryDetailsVisual(details))
		return nil
	},
}

// memoryDetailsResultMap builds the machine envelope from memoryDetails.
// Keeps the same top-level keys memory-metrics/the old alias emitted
// (wisdom, pending, instincts, curation, recent_failures, last_activity) so
// an existing JSON consumer keeps working, adds a "deferred" key, and
// populates last_activity.queen_md_updated from the real value instead of a
// hardcoded empty string.
func memoryDetailsResultMap(d memoryDetails) map[string]interface{} {
	summary := d.Instincts

	wisdomEntries := make([]map[string]interface{}, 0, len(d.Wisdom))
	for _, cat := range d.Wisdom {
		wisdomEntries = append(wisdomEntries, map[string]interface{}{
			"category": cat.Category,
			"entries":  cat.Entries,
		})
	}

	pendingEntries := make([]map[string]interface{}, 0, len(d.Pending))
	for _, p := range d.Pending {
		pendingEntries = append(pendingEntries, map[string]interface{}{
			"content":           p.Content,
			"observation_count": p.ObservationCount,
			"last_seen":         p.LastSeen,
			"trust_score":       p.TrustScore,
		})
	}

	deferredEntries := make([]map[string]interface{}, 0, len(d.Deferred))
	for _, p := range d.Deferred {
		deferredEntries = append(deferredEntries, map[string]interface{}{
			"content":           p.Content,
			"observation_count": p.ObservationCount,
			"last_seen":         p.LastSeen,
			"trust_score":       p.TrustScore,
		})
	}

	failureEntries := make([]map[string]interface{}, 0, len(d.Failures))
	for _, f := range d.Failures {
		failureEntries = append(failureEntries, map[string]interface{}{
			"timestamp": f.Timestamp,
			"source":    f.Source,
			"category":  f.Category,
			"message":   f.Message,
		})
	}

	return map[string]interface{}{
		"wisdom": map[string]interface{}{
			"total":      summary.WisdomTotal,
			"categories": wisdomEntries,
		},
		"pending": map[string]interface{}{
			"total":   summary.PendingPromotions,
			"entries": pendingEntries,
		},
		"deferred": map[string]interface{}{
			"total":   len(d.Deferred),
			"entries": deferredEntries,
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
			"count":   summary.RecentFailures,
			"entries": failureEntries,
		},
		"last_activity": map[string]interface{}{
			"queen_md_updated":  d.QueenMDUpdated,
			"learning_captured": summary.LastLearning,
			"last_failure":      summary.LastFailure,
			"instinct_used":     summary.LastInstinctTouched,
		},
	}
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
	memoryDetailsCmd.Flags().Bool("json", false, "Output the machine envelope instead of the rendered drill-down")
	rootCmd.AddCommand(memoryMetricsCmd)
	rootCmd.AddCommand(memoryDetailsCmd)
	rootCmd.AddCommand(colonyVitalSignsCmd)
}
