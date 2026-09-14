package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/spf13/cobra"
)

var (
	instinctTrigger    string
	instinctAction     string
	instinctConfidence float64
	instinctDomain     string
	instinctSource     string
	instinctEvidence   string
	instinctMinScore   float64
	instinctLimit      int
	instinctDecayDays  int
	instinctDryRun     bool
	instinctID         string
)

var instinctCreateCmd = &cobra.Command{
	Use:   "instinct-create [trigger] [action]",
	Short: "Create or reinforce an instinct entry",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		trigger := mustGetStringCompat(cmd, args, "trigger", 0)
		action := mustGetStringCompat(cmd, args, "action", 1)
		if trigger == "" || action == "" {
			return nil
		}

		confidence, _ := cmd.Flags().GetFloat64("confidence")
		domain, _ := cmd.Flags().GetString("domain")
		source, _ := cmd.Flags().GetString("source")
		evidence, _ := cmd.Flags().GetString("evidence")

		var file colony.InstinctsFile
		if err := store.LoadJSON("instincts.json", &file); err != nil {
			file = colony.InstinctsFile{Version: "1.0"}
		}

		// Check for duplicate by trigger+action
		duplicate := false
		now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		for i := range file.Instincts {
			if file.Instincts[i].Trigger == trigger && file.Instincts[i].Action == action && !file.Instincts[i].Archived {
				duplicate = true
				newConf := file.Instincts[i].Confidence + 0.05
				if newConf > 1.0 {
					newConf = 1.0
				}
				file.Instincts[i].Confidence = newConf
				file.Instincts[i].Provenance.ApplicationCount++
				file.Instincts[i].Provenance.LastApplied = &now
				file.Instincts[i].TrustScore = file.Instincts[i].Confidence
				tierName, _ := memory.Tier(file.Instincts[i].TrustScore)
				file.Instincts[i].TrustTier = tierName

				if err := store.SaveJSON("instincts.json", file); err != nil {
					outputError(1, fmt.Sprintf("failed to save: %v", err), nil)
					return nil
				}

				outputOK(map[string]interface{}{
					"id":              file.Instincts[i].ID,
					"trigger":         trigger,
					"action":          action,
					"confidence":      file.Instincts[i].Confidence,
					"trust_tier":      file.Instincts[i].TrustTier,
					"status":          "reinforced",
					"duplicate":       duplicate,
					"total_instincts": len(file.Instincts),
				})
				return nil
			}
		}

		// New instinct
		id := fmt.Sprintf("inst_%d", time.Now().Unix())
		tierName, tierIndex := memory.Tier(confidence)
		entry := colony.InstinctEntry{
			ID:         id,
			Trigger:    trigger,
			Action:     action,
			Domain:     domain,
			TrustScore: confidence,
			TrustTier:  tierName,
			Confidence: confidence,
			Provenance: colony.InstinctProvenance{
				Source:           source,
				SourceType:       "observation",
				Evidence:         evidence,
				CreatedAt:        now,
				ApplicationCount: 1,
			},
			ApplicationHistory: []colony.InstinctApplicationEntry{},
			RelatedInstincts:   []interface{}{},
			Archived:           false,
		}

		file.Instincts = append(file.Instincts, entry)

		// Cap at 30 instincts: archive lowest-confidence
		activeCount := 0
		for _, inst := range file.Instincts {
			if !inst.Archived {
				activeCount++
			}
		}
		if activeCount > 30 {
			lowestIdx := -1
			lowestConf := 2.0
			for i, inst := range file.Instincts {
				if !inst.Archived && inst.Confidence < lowestConf {
					lowestConf = inst.Confidence
					lowestIdx = i
				}
			}
			if lowestIdx >= 0 {
				file.Instincts[lowestIdx].Archived = true
			}
		}

		if err := store.SaveJSON("instincts.json", file); err != nil {
			outputError(1, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"id":               id,
			"trigger":          trigger,
			"action":           action,
			"confidence":       confidence,
			"trust_tier":       tierName,
			"trust_tier_index": tierIndex,
			"status":           "active",
			"duplicate":        duplicate,
			"total_instincts":  len(file.Instincts),
		})
		return nil
	},
}

var instinctReadTrustedCmd = &cobra.Command{
	Use:   "instinct-read-trusted",
	Short: "Read instincts filtered by minimum trust score",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		minScore, _ := cmd.Flags().GetFloat64("min-score")
		domain, _ := cmd.Flags().GetString("domain")
		limit, _ := cmd.Flags().GetInt("limit")

		var file colony.InstinctsFile
		if err := store.LoadJSON("instincts.json", &file); err != nil {
			outputOK(map[string]interface{}{"instincts": []interface{}{}, "count": 0})
			return nil
		}

		var filtered []colony.InstinctEntry
		for _, inst := range file.Instincts {
			if inst.Archived {
				continue
			}
			if inst.TrustScore < minScore {
				continue
			}
			if domain != "" && inst.Domain != domain {
				continue
			}
			filtered = append(filtered, inst)
		}

		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].TrustScore > filtered[j].TrustScore
		})

		if limit > 0 && len(filtered) > limit {
			filtered = filtered[:limit]
		}

		outputOK(map[string]interface{}{
			"instincts": filtered,
			"count":     len(filtered),
		})
		return nil
	},
}

var instinctDecayAllCmd = &cobra.Command{
	Use:   "instinct-decay-all",
	Short: "Apply trust decay to all non-archived instincts",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		days, _ := cmd.Flags().GetInt("days")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		var file colony.InstinctsFile
		if err := store.LoadJSON("instincts.json", &file); err != nil {
			outputOK(map[string]interface{}{"processed": 0, "archived": 0, "days": days, "dry_run": dryRun})
			return nil
		}

		processed := 0
		archived := 0
		for i := range file.Instincts {
			if file.Instincts[i].Archived {
				continue
			}
			decayed := memory.Decay(file.Instincts[i].TrustScore, days)
			file.Instincts[i].TrustScore = decayed
			tierName, _ := memory.Tier(decayed)
			file.Instincts[i].TrustTier = tierName
			processed++

			if decayed < 0.25 {
				file.Instincts[i].Archived = true
				archived++
			}
		}

		if !dryRun {
			if err := store.SaveJSON("instincts.json", file); err != nil {
				outputError(1, fmt.Sprintf("failed to save: %v", err), nil)
				return nil
			}
		}

		outputOK(map[string]interface{}{
			"processed": processed,
			"archived":  archived,
			"days":      days,
			"dry_run":   dryRun,
		})
		return nil
	},
}

var instinctArchiveCmd = &cobra.Command{
	Use:   "instinct-archive",
	Short: "Archive an instinct by ID",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		id := mustGetString(cmd, "id")
		if id == "" {
			return nil
		}

		var file colony.InstinctsFile
		if err := store.LoadJSON("instincts.json", &file); err != nil {
			outputError(1, "instincts.json not found", nil)
			return nil
		}

		found := false
		for i := range file.Instincts {
			if file.Instincts[i].ID == id {
				file.Instincts[i].Archived = true
				found = true
				break
			}
		}

		if !found {
			outputError(1, fmt.Sprintf("instinct %q not found", id), nil)
			return nil
		}

		if err := store.SaveJSON("instincts.json", file); err != nil {
			outputError(1, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"archived": id})
		return nil
	},
}

func init() {
	// instinct-create
	rootCmd.AddCommand(instinctCreateCmd)
	instinctCreateCmd.Flags().StringVar(&instinctTrigger, "trigger", "", "Trigger pattern (required)")
	instinctCreateCmd.Flags().StringVar(&instinctAction, "action", "", "Action pattern (required)")
	instinctCreateCmd.Flags().Float64Var(&instinctConfidence, "confidence", 0.75, "Initial confidence")
	instinctCreateCmd.Flags().StringVar(&instinctDomain, "domain", "", "Domain tag")
	instinctCreateCmd.Flags().StringVar(&instinctSource, "source", "observation", "Source type")
	instinctCreateCmd.Flags().StringVar(&instinctEvidence, "evidence", "", "Supporting evidence")

	// instinct-read-trusted
	rootCmd.AddCommand(instinctReadTrustedCmd)
	instinctReadTrustedCmd.Flags().Float64Var(&instinctMinScore, "min-score", 0.5, "Minimum trust score")
	instinctReadTrustedCmd.Flags().StringVar(&instinctDomain, "domain", "", "Filter by domain")
	instinctReadTrustedCmd.Flags().IntVar(&instinctLimit, "limit", 20, "Maximum results")

	// instinct-decay-all
	rootCmd.AddCommand(instinctDecayAllCmd)
	instinctDecayAllCmd.Flags().IntVar(&instinctDecayDays, "days", 30, "Days of decay to apply")
	instinctDecayAllCmd.Flags().BoolVar(&instinctDryRun, "dry-run", false, "Report without modifying")

	// instinct-archive
	rootCmd.AddCommand(instinctArchiveCmd)
	instinctArchiveCmd.Flags().StringVar(&instinctID, "id", "", "Instinct ID to archive (required)")
}
