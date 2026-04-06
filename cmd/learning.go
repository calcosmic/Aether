package cmd

import (
	"fmt"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/spf13/cobra"
)

var learningObserveCmd = &cobra.Command{
	Use:   "learning-observe",
	Short: "Capture a learning observation",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		content := mustGetString(cmd, "content")
		if content == "" {
			return nil
		}
		wisdomType := mustGetString(cmd, "type")
		if wisdomType == "" {
			return nil
		}

		colonyName, _ := cmd.Flags().GetString("colony-name")
		if colonyName == "" {
			colonyName = "unknown"
		}
		sourceType, _ := cmd.Flags().GetString("source-type")
		if sourceType == "" {
			sourceType = "observation"
		}
		evidenceType, _ := cmd.Flags().GetString("evidence-type")
		if evidenceType == "" {
			evidenceType = "anecdotal"
		}

		bus := events.NewBus(store, events.DefaultConfig())
		obsService := memory.NewObservationService(store, bus)

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()
		result, err := obsService.CaptureWithTrust(ctx, content, wisdomType, colonyName, sourceType, evidenceType)
		if err != nil {
			outputError(2, fmt.Sprintf("failed to capture observation: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"captured":           true,
			"is_new":             result.IsNew,
			"observation_id":     result.Observation.ContentHash,
			"observation_count":  result.Observation.ObservationCount,
			"trust_score":        result.Observation.TrustScore,
			"promotion_eligible": result.PromotionEligible,
			"promotion_reason":   result.PromotionReason,
		})
		return nil
	},
}

var learningCheckPromotionCmd = &cobra.Command{
	Use:   "learning-check-promotion",
	Short: "Check if an observation is eligible for promotion",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		checkAll, _ := cmd.Flags().GetBool("all")

		var file colony.LearningFile
		if err := store.LoadJSON("learning-observations.json", &file); err != nil {
			outputError(1, "learning-observations.json not found", nil)
			return nil
		}

		if checkAll {
			var eligible []map[string]interface{}
			for _, obs := range file.Observations {
				isEligible, reason := memory.CheckPromotion(obs)
				if isEligible {
					entry := map[string]interface{}{
						"content_hash":      obs.ContentHash,
						"observation_count": obs.ObservationCount,
						"trust_score":       obs.TrustScore,
						"reason":            reason,
					}
					eligible = append(eligible, entry)
				}
			}
			outputOK(map[string]interface{}{
				"eligible": eligible,
				"total":    len(file.Observations),
			})
			return nil
		}

		obsID := mustGetString(cmd, "observation-id")
		if obsID == "" {
			return nil
		}

		var found *colony.Observation
		for i := range file.Observations {
			if file.Observations[i].ContentHash == obsID {
				found = &file.Observations[i]
				break
			}
		}

		if found == nil {
			outputError(1, fmt.Sprintf("observation %q not found", obsID), nil)
			return nil
		}

		eligible, reason := memory.CheckPromotion(*found)

		outputOK(map[string]interface{}{
			"promotable":        eligible,
			"reason":            reason,
			"observation_count": found.ObservationCount,
			"trust_score":       found.TrustScore,
		})
		return nil
	},
}

var learningPromoteAutoCmd = &cobra.Command{
	Use:   "learning-promote-auto",
	Short: "Auto-promote eligible observations to instincts",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var file colony.LearningFile
		if err := store.LoadJSON("learning-observations.json", &file); err != nil {
			outputOK(map[string]interface{}{"promoted": 0, "reason": "no observations file"})
			return nil
		}

		bus := events.NewBus(store, events.DefaultConfig())
		promoteService := memory.NewPromoteService(store, bus)

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()

		promoted := 0
		for _, obs := range file.Observations {
			eligible, _ := memory.CheckPromotion(obs)
			if eligible {
				result, err := promoteService.Promote(ctx, obs, "auto-promote")
				if err != nil {
					continue
				}
				if result.IsNew {
					promoted++
				}
			}
		}

		outputOK(map[string]interface{}{
			"promoted":       promoted,
			"total_observed": len(file.Observations),
		})
		return nil
	},
}

var memoryCaptureCmd = &cobra.Command{
	Use:   "memory-capture",
	Short: "Capture a memory observation (simplified learning-observe)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		content := mustGetString(cmd, "content")
		if content == "" {
			return nil
		}
		obsType, _ := cmd.Flags().GetString("type")
		if obsType == "" {
			obsType = "observation"
		}

		bus := events.NewBus(store, events.DefaultConfig())
		obsService := memory.NewObservationService(store, bus)

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()
		result, err := obsService.Capture(ctx, content, obsType, "unknown")
		if err != nil {
			outputError(2, fmt.Sprintf("failed to capture: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"captured":    true,
			"is_new":      result.IsNew,
			"trust_score": result.Observation.TrustScore,
		})
		return nil
	},
}

func init() {
	learningObserveCmd.Flags().String("content", "", "Observation content (required)")
	learningObserveCmd.Flags().String("type", "", "Wisdom type (required)")
	learningObserveCmd.Flags().String("colony-name", "", "Colony name")
	learningObserveCmd.Flags().String("source-type", "", "Source type")
	learningObserveCmd.Flags().String("evidence-type", "", "Evidence type")

	learningCheckPromotionCmd.Flags().String("observation-id", "", "Observation content hash (required)")
	learningCheckPromotionCmd.Flags().Bool("all", false, "Check all observations for promotion eligibility")

	memoryCaptureCmd.Flags().String("content", "", "Observation content (required)")
	memoryCaptureCmd.Flags().String("type", "", "Wisdom type (default: observation)")

	rootCmd.AddCommand(learningObserveCmd)
	rootCmd.AddCommand(learningCheckPromotionCmd)
	rootCmd.AddCommand(learningPromoteAutoCmd)
	rootCmd.AddCommand(memoryCaptureCmd)
}

// ensure timeoutCtx is used (compile-time check)
var _ = timeoutCtx
