package cmd

import (
	"fmt"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/spf13/cobra"
)

// --- learning-approve-proposals ---

var learningApproveProposalsCmd = &cobra.Command{
	Use:   "learning-approve-proposals",
	Short: "Approve pending learning proposals",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		approveAll, _ := cmd.Flags().GetBool("all")
		idsStr, _ := cmd.Flags().GetString("ids")

		if !approveAll && idsStr == "" {
			outputError(1, "requires --all or --ids flag", nil)
			return nil
		}

		var file colony.LearningFile
		if err := store.LoadJSON("learning-observations.json", &file); err != nil {
			outputError(1, "learning-observations.json not found", nil)
			return nil
		}

		approved := 0
		for i := range file.Observations {
			if file.Observations[i].SourceType != "proposed" {
				continue
			}
			if approveAll {
				file.Observations[i].SourceType = "approved"
				approved++
			} else {
				for _, id := range splitCSV(idsStr) {
					if file.Observations[i].ContentHash == id {
						file.Observations[i].SourceType = "approved"
						approved++
						break
					}
				}
			}
		}

		if err := store.SaveJSON("learning-observations.json", file); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"approved": approved,
		})
		return nil
	},
}

// --- learning-defer-proposals ---

var learningDeferProposalsCmd = &cobra.Command{
	Use:   "learning-defer-proposals",
	Short: "Defer pending learning proposals",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		deferAll, _ := cmd.Flags().GetBool("all")
		idsStr, _ := cmd.Flags().GetString("ids")

		if !deferAll && idsStr == "" {
			outputError(1, "requires --all or --ids flag", nil)
			return nil
		}

		var file colony.LearningFile
		if err := store.LoadJSON("learning-observations.json", &file); err != nil {
			outputError(1, "learning-observations.json not found", nil)
			return nil
		}

		deferred := 0
		for i := range file.Observations {
			if file.Observations[i].SourceType != "proposed" {
				continue
			}
			if deferAll {
				file.Observations[i].SourceType = "deferred"
				deferred++
			} else {
				for _, id := range splitCSV(idsStr) {
					if file.Observations[i].ContentHash == id {
						file.Observations[i].SourceType = "deferred"
						deferred++
						break
					}
				}
			}
		}

		if err := store.SaveJSON("learning-observations.json", file); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"deferred": deferred,
		})
		return nil
	},
}

// --- learning-display-proposals ---

var learningDisplayProposalsCmd = &cobra.Command{
	Use:   "learning-display-proposals",
	Short: "Display pending learning proposals",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var file colony.LearningFile
		if err := store.LoadJSON("learning-observations.json", &file); err != nil {
			outputOK(map[string]interface{}{"proposals": []interface{}{}, "count": 0})
			return nil
		}

		var proposals []colony.Observation
		for _, obs := range file.Observations {
			if obs.SourceType == "proposed" {
				proposals = append(proposals, obs)
			}
		}

		if proposals == nil {
			proposals = []colony.Observation{}
		}

		outputOK(map[string]interface{}{
			"proposals": proposals,
			"count":     len(proposals),
		})
		return nil
	},
}

// --- learning-extract-fallback ---

var learningExtractFallbackCmd = &cobra.Command{
	Use:   "learning-extract-fallback",
	Short: "Extract learning with fallback to simpler approach",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		category := mustGetString(cmd, "category")
		if category == "" {
			return nil
		}
		content := mustGetString(cmd, "content")
		if content == "" {
			return nil
		}
		fallback, _ := cmd.Flags().GetString("fallback")

		// Try to use content as the learning; fall back if empty
		extracted := content
		if extracted == "" && fallback != "" {
			extracted = fallback
		}

		outputOK(map[string]interface{}{
			"category":      category,
			"extracted":     extracted,
			"used_fallback": extracted == fallback,
		})
		return nil
	},
}

// --- learning-inject ---

var learningInjectCmd = &cobra.Command{
	Use:   "learning-inject",
	Short: "Directly inject a learning observation",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		category := mustGetString(cmd, "category")
		if category == "" {
			return nil
		}
		content := mustGetString(cmd, "content")
		if content == "" {
			return nil
		}
		trustScore, _ := cmd.Flags().GetFloat64("trust-score")
		source, _ := cmd.Flags().GetString("source")
		if source == "" {
			source = "manual"
		}

		var file colony.LearningFile
		if err := store.LoadJSON("learning-observations.json", &file); err != nil {
			file = colony.LearningFile{}
		}

		now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		id := fmt.Sprintf("obs_%d", time.Now().Unix())

		obs := colony.Observation{
			ContentHash:      id,
			Content:          content,
			WisdomType:       category,
			ObservationCount: 1,
			FirstSeen:        now,
			LastSeen:         now,
			Colonies:         []string{},
			TrustScore:       &trustScore,
			SourceType:       source,
		}

		file.Observations = append(file.Observations, obs)

		if err := store.SaveJSON("learning-observations.json", file); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"injected": true,
			"id":       id,
		})
		return nil
	},
}

// --- learning-promote ---

var learningPromoteCmd = &cobra.Command{
	Use:   "learning-promote",
	Short: "Promote an observation to an instinct",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		obsID := args[0]

		var file colony.LearningFile
		if err := store.LoadJSON("learning-observations.json", &file); err != nil {
			outputError(1, "learning-observations.json not found", nil)
			return nil
		}

		var found *colony.Observation
		foundIdx := -1
		for i := range file.Observations {
			if file.Observations[i].ContentHash == obsID {
				found = &file.Observations[i]
				foundIdx = i
				break
			}
		}

		if found == nil {
			outputError(1, fmt.Sprintf("observation %q not found", obsID), nil)
			return nil
		}

		bus := events.NewBus(store, events.DefaultConfig())
		promoteService := learn.NewPromoteService(store, bus)
		ctx, cancel := timeoutCtx(cmd)
		defer cancel()

		result, err := promoteService.Promote(ctx, *found, "manual-promote")
		if err != nil {
			outputError(2, fmt.Sprintf("failed to promote observation: %v", err), nil)
			return nil
		}

		// Mark observation as promoted
		file.Observations[foundIdx].SourceType = "promoted"
		if err := store.SaveJSON("learning-observations.json", file); err != nil {
			outputError(2, fmt.Sprintf("failed to save observations: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"promoted":    true,
			"observation": obsID,
			"instinct_id": result.Instinct.ID,
			"deduped":     result.WasDeduped,
			"is_new":      result.IsNew,
		})
		return nil
	},
}

// --- learning-select-proposals ---

var learningSelectProposalsCmd = &cobra.Command{
	Use:   "learning-select-proposals",
	Short: "Select specific proposals for interactive review",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		category, _ := cmd.Flags().GetString("category")

		var file colony.LearningFile
		if err := store.LoadJSON("learning-observations.json", &file); err != nil {
			outputOK(map[string]interface{}{"proposals": []interface{}{}, "count": 0})
			return nil
		}

		var proposals []colony.Observation
		for _, obs := range file.Observations {
			if obs.SourceType != "proposed" {
				continue
			}
			if category != "" && obs.WisdomType != category {
				continue
			}
			proposals = append(proposals, obs)
		}

		if proposals == nil {
			proposals = []colony.Observation{}
		}

		outputOK(map[string]interface{}{
			"proposals": proposals,
			"count":     len(proposals),
		})
		return nil
	},
}

// --- learning-undo-promotions ---

var learningUndoPromotionsCmd = &cobra.Command{
	Use:   "learning-undo-promotions",
	Short: "Revert recent instinct promotions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		count, _ := cmd.Flags().GetInt("count")
		if count <= 0 {
			count = 1
		}

		file := loadInstinctFileOrEmpty(store)
		active := sortedActiveInstinctEntries(file)
		totalInstincts := len(active)
		if count > totalInstincts {
			count = totalInstincts
		}

		// Collect source observation hashes from instincts being archived.
		removedSources := map[string]bool{}
		for i := totalInstincts - count; i < totalInstincts; i++ {
			targetID := active[i].ID
			for idx := range file.Instincts {
				if file.Instincts[idx].ID != targetID {
					continue
				}
				file.Instincts[idx].Archived = true
				if src := file.Instincts[idx].Provenance.Source; src != "" {
					removedSources[src] = true
				}
				break
			}
		}

		if err := store.SaveJSON("instincts.json", file); err != nil {
			outputError(2, fmt.Sprintf("failed to save instincts: %v", err), nil)
			return nil
		}

		// Revert corresponding observations back to "proposed"
		var obsFile colony.LearningFile
		if err := store.LoadJSON("learning-observations.json", &obsFile); err == nil {
			reverted := 0
			for i := range obsFile.Observations {
				if !removedSources[obsFile.Observations[i].ContentHash] {
					continue
				}
				if obsFile.Observations[i].SourceType == "promoted" {
					obsFile.Observations[i].SourceType = "proposed"
					reverted++
				}
			}
			if reverted > 0 {
				store.SaveJSON("learning-observations.json", obsFile)
			}
		}

		outputOK(map[string]interface{}{
			"undone":              count,
			"remaining_instincts": len(sortedActiveInstinctEntries(file)),
		})
		return nil
	},
}

// splitCSV splits a comma-separated string into trimmed strings.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	for _, p := range splitByComma(s) {
		trimmed := trimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// splitByComma splits a string by comma.
func splitByComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// trimSpace trims whitespace from a string.
func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func init() {
	learningApproveProposalsCmd.Flags().Bool("all", false, "Approve all pending proposals")
	learningApproveProposalsCmd.Flags().String("ids", "", "Comma-separated list of proposal IDs")
	learningApproveProposalsCmd.Flags().Bool("deferred", false, "Also approve deferred proposals")
	learningApproveProposalsCmd.Flags().Bool("verbose", false, "Show detailed approval results")

	learningDeferProposalsCmd.Flags().Bool("all", false, "Defer all pending proposals")
	learningDeferProposalsCmd.Flags().String("ids", "", "Comma-separated list of proposal IDs")

	learningExtractFallbackCmd.Flags().String("category", "", "Learning category (required)")
	learningExtractFallbackCmd.Flags().String("content", "", "Primary content (required)")
	learningExtractFallbackCmd.Flags().String("fallback", "", "Fallback content if extraction fails")

	learningInjectCmd.Flags().String("category", "", "Observation category (required)")
	learningInjectCmd.Flags().String("content", "", "Observation content (required)")
	learningInjectCmd.Flags().Float64("trust-score", 0.5, "Trust score for the observation")
	learningInjectCmd.Flags().String("source", "manual", "Source of the observation")

	learningSelectProposalsCmd.Flags().String("category", "", "Filter by category")

	learningUndoPromotionsCmd.Flags().Int("count", 1, "Number of recent promotions to undo")

	rootCmd.AddCommand(learningApproveProposalsCmd)
	rootCmd.AddCommand(learningDeferProposalsCmd)
	rootCmd.AddCommand(learningDisplayProposalsCmd)
	rootCmd.AddCommand(learningExtractFallbackCmd)
	rootCmd.AddCommand(learningInjectCmd)
	rootCmd.AddCommand(learningPromoteCmd)
	rootCmd.AddCommand(learningSelectProposalsCmd)
	rootCmd.AddCommand(learningUndoPromotionsCmd)
}
