package cmd

import (
	"fmt"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var flagAddCmd = &cobra.Command{
	Use:   "flag-add [title]",
	Short: "Create a new flag",
	Args:  cobra.MaximumNArgs(1),
	Aliases: []string{
		"flag",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		title, _ := cmd.Flags().GetString("title")
		if title == "" && len(args) > 0 {
			title = args[0]
		}
		if title == "" {
			outputError(1, "flag title is required", nil)
			return nil
		}
		severity, _ := cmd.Flags().GetString("severity")
		if severity == "" {
			severity = "high"
		}
		source, _ := cmd.Flags().GetString("source")
		flagType, _ := cmd.Flags().GetString("type")
		description, _ := cmd.Flags().GetString("description")
		phaseNum, _ := cmd.Flags().GetInt("phase")

		if flagType == "" {
			flagType = "issue"
		}
		if description == "" {
			description = title
		}

		// Validate severity
		switch severity {
		case "critical", "high", "low":
		default:
			outputError(1, fmt.Sprintf("invalid severity %q: must be critical, high, or low", severity), nil)
			return nil
		}

		var phasePtr *int
		if phaseNum > 0 {
			phasePtr = &phaseNum
		}

		flag := colony.FlagEntry{
			ID:          generateFlagID(),
			Type:        flagType,
			Description: description,
			Source:      source,
			Phase:       phasePtr,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			Resolved:    false,
		}

		// Load existing flags
		var ff colony.FlagsFile
		loaded := false
		if err := store.LoadJSON("pending-decisions.json", &ff); err == nil {
			loaded = true
		} else if err2 := store.LoadJSON("flags.json", &ff); err2 == nil {
			loaded = true
		}
		if !loaded {
			ff = colony.FlagsFile{Decisions: []colony.FlagEntry{}}
		}
		if ff.Decisions == nil {
			ff.Decisions = []colony.FlagEntry{}
		}

		ff.Decisions = append(ff.Decisions, flag)

		if err := store.SaveJSON("pending-decisions.json", ff); err != nil {
			outputError(2, fmt.Sprintf("failed to save flags: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"created": true,
			"flag":    flag,
			"total":   len(ff.Decisions),
		})
		return nil
	},
}

var flagResolveCmd = &cobra.Command{
	Use:   "flag-resolve",
	Short: "Resolve a flag by ID",
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
		message, _ := cmd.Flags().GetString("message")

		var ff colony.FlagsFile
		if err := store.LoadJSON("pending-decisions.json", &ff); err != nil {
			if err2 := store.LoadJSON("flags.json", &ff); err2 != nil {
				outputError(1, "flags file not found", nil)
				return nil
			}
		}

		found := false
		for i := range ff.Decisions {
			if ff.Decisions[i].ID == id {
				ff.Decisions[i].Resolved = true
				ff.Decisions[i].ResolvedAt = time.Now().UTC().Format(time.RFC3339)
				ff.Decisions[i].Resolution = message
				found = true
				break
			}
		}

		if !found {
			outputError(1, fmt.Sprintf("flag %q not found", id), nil)
			return nil
		}

		if err := store.SaveJSON("pending-decisions.json", ff); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"resolved":  true,
			"id":        id,
			"message":   message,
			"timestamp": ff.Decisions[0].ResolvedAt,
		})
		return nil
	},
}

var flagCheckBlockersCmd = &cobra.Command{
	Use:   "flag-check-blockers",
	Short: "Check for active critical flags (blockers)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var ff colony.FlagsFile
		if err := store.LoadJSON("pending-decisions.json", &ff); err != nil {
			if err2 := store.LoadJSON("flags.json", &ff); err2 != nil {
				outputOK(map[string]interface{}{
					"blockers":     0,
					"issues":       0,
					"notes":        0,
					"has_blockers": false,
				})
				return nil
			}
		}

		blockers := 0
		issues := 0
		notes := 0
		for _, f := range ff.Decisions {
			if f.Resolved {
				continue
			}
			switch f.Type {
			case "blocker":
				blockers++
			case "issue":
				issues++
			case "note":
				notes++
			default:
				issues++
			}
		}

		outputOK(map[string]interface{}{
			"blockers":     blockers,
			"issues":       issues,
			"notes":        notes,
			"has_blockers": blockers > 0,
		})
		return nil
	},
}

var flagAcknowledgeCmd = &cobra.Command{
	Use:   "flag-acknowledge",
	Short: "Acknowledge a flag by ID",
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

		var ff colony.FlagsFile
		if err := store.LoadJSON("pending-decisions.json", &ff); err != nil {
			if err2 := store.LoadJSON("flags.json", &ff); err2 != nil {
				outputError(1, "flags file not found", nil)
				return nil
			}
		}

		found := false
		for i := range ff.Decisions {
			if ff.Decisions[i].ID == id {
				ff.Decisions[i].Acknowledged = true
				found = true
				break
			}
		}

		if !found {
			outputError(1, fmt.Sprintf("flag %q not found", id), nil)
			return nil
		}

		if err := store.SaveJSON("pending-decisions.json", ff); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"acknowledged": true,
			"id":           id,
			"at":           time.Now().UTC().Format(time.RFC3339),
		})
		return nil
	},
}

var flagAutoResolveCmd = &cobra.Command{
	Use:   "flag-auto-resolve",
	Short: "Auto-resolve flags matching patterns (e.g., old flags)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		maxDays, _ := cmd.Flags().GetInt("max-days")
		if maxDays == 0 {
			maxDays = 7
		}

		var ff colony.FlagsFile
		if err := store.LoadJSON("pending-decisions.json", &ff); err != nil {
			if err2 := store.LoadJSON("flags.json", &ff); err2 != nil {
				outputOK(map[string]interface{}{"resolved": 0})
				return nil
			}
		}

		cutoff := time.Now().UTC().AddDate(0, 0, -maxDays)
		resolved := 0

		for i := range ff.Decisions {
			if ff.Decisions[i].Resolved {
				continue
			}
			createdAt, err := time.Parse(time.RFC3339, ff.Decisions[i].CreatedAt)
			if err != nil {
				continue
			}
			if createdAt.Before(cutoff) {
				ff.Decisions[i].Resolved = true
				resolved++
			}
		}

		if resolved > 0 {
			if err := store.SaveJSON("pending-decisions.json", ff); err != nil {
				outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
				return nil
			}
		}

		outputOK(map[string]interface{}{
			"resolved": resolved,
			"max_days": maxDays,
		})
		return nil
	},
}

func init() {
	flagAddCmd.Flags().String("title", "", "Flag title/description (required)")
	flagAddCmd.Flags().String("severity", "", "Severity: critical, high, low (required)")
	flagAddCmd.Flags().String("source", "", "Source of the flag")
	flagAddCmd.Flags().String("type", "", "Flag type: blocker, issue, note (default: issue)")
	flagAddCmd.Flags().String("description", "", "Detailed description (defaults to title)")
	flagAddCmd.Flags().Int("phase", 0, "Phase number (0 means no phase)")

	flagResolveCmd.Flags().String("id", "", "Flag ID to resolve (required)")
	flagResolveCmd.Flags().String("message", "", "Resolution message")
	flagAcknowledgeCmd.Flags().String("id", "", "Flag ID to acknowledge (required)")

	flagAutoResolveCmd.Flags().Int("max-days", 7, "Maximum age in days for auto-resolution")

	rootCmd.AddCommand(flagAddCmd)
	rootCmd.AddCommand(flagResolveCmd)
	rootCmd.AddCommand(flagCheckBlockersCmd)
	rootCmd.AddCommand(flagAcknowledgeCmd)
	rootCmd.AddCommand(flagAutoResolveCmd)
}
