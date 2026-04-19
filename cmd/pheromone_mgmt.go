package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var pheromonePrimeCmd = &cobra.Command{
	Use:   "pheromone-prime",
	Short: "Format active pheromone signals for prompt injection",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		pf := loadPheromones()
		if pf == nil {
			outputOK(map[string]interface{}{
				"section":      "",
				"signal_count": 0,
			})
			return nil
		}

		now := time.Now().UTC()
		var focus, redirect, feedback []colony.PheromoneSignal
		for _, sig := range filterSignalsForPrompt(pf.Signals, now) {
			switch sig.Type {
			case "FOCUS":
				focus = append(focus, sig)
			case "REDIRECT":
				redirect = append(redirect, sig)
			case "FEEDBACK":
				feedback = append(feedback, sig)
			}
		}

		var sb strings.Builder
		total := 0

		if len(redirect) > 0 {
			sb.WriteString("## ACTIVE REDIRECT SIGNALS (Hard Constraints)\n\n")
			for _, sig := range redirect {
				text := extractText(sig.Content)
				strength := "1.0"
				if sig.Strength != nil {
					strength = fmt.Sprintf("%.1f", *sig.Strength)
				}
				sb.WriteString(fmt.Sprintf("- [REDIRECT] %s (priority: %s, strength: %s)\n", text, sig.Priority, strength))
			}
			sb.WriteString("\n")
			total += len(redirect)
		}

		if len(focus) > 0 {
			sb.WriteString("## ACTIVE FOCUS SIGNALS\n\n")
			for _, sig := range focus {
				text := extractText(sig.Content)
				strength := "1.0"
				if sig.Strength != nil {
					strength = fmt.Sprintf("%.1f", *sig.Strength)
				}
				sb.WriteString(fmt.Sprintf("- [FOCUS] %s (priority: %s, strength: %s)\n", text, sig.Priority, strength))
			}
			sb.WriteString("\n")
			total += len(focus)
		}

		if len(feedback) > 0 {
			sb.WriteString("## ACTIVE FEEDBACK SIGNALS\n\n")
			for _, sig := range feedback {
				text := extractText(sig.Content)
				strength := "1.0"
				if sig.Strength != nil {
					strength = fmt.Sprintf("%.1f", *sig.Strength)
				}
				sb.WriteString(fmt.Sprintf("- [FEEDBACK] %s (priority: %s, strength: %s)\n", text, sig.Priority, strength))
			}
			sb.WriteString("\n")
			total += len(feedback)
		}

		section := strings.TrimSpace(sb.String())
		if section == "" {
			section = "No active pheromone signals."
		}

		outputOK(map[string]interface{}{
			"section":        section,
			"signal_count":   total,
			"focus_count":    len(focus),
			"redirect_count": len(redirect),
			"feedback_count": len(feedback),
		})
		return nil
	},
}

var colonyPrimeCmd = &cobra.Command{
	Use:   "colony-prime",
	Short: "Assemble full colony context for worker prompt injection",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		compact, _ := cmd.Flags().GetBool("compact")
		outputOK(buildColonyPrimeOutput(compact))
		return nil
	},
}

// pheromoneDisplayCmd renders active pheromone signals in a formatted table.
var pheromoneDisplayCmd = &cobra.Command{
	Use:          "pheromone-display",
	Short:        "Display active pheromone signals in formatted table",
	Aliases:      []string{"pheromones"},
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		pf := loadPheromones()
		if pf == nil {
			outputOK(map[string]interface{}{
				"signals": []interface{}{},
				"count":   0,
				"display": "No pheromone signals found.",
			})
			return nil
		}

		filterType, _ := cmd.Flags().GetString("type")
		activeOnly, _ := cmd.Flags().GetBool("active-only")

		var filtered []colony.PheromoneSignal
		for _, sig := range pf.Signals {
			if activeOnly && !sig.Active {
				continue
			}
			if filterType != "" && sig.Type != filterType {
				continue
			}
			filtered = append(filtered, sig)
		}

		if len(filtered) == 0 {
			outputOK(map[string]interface{}{
				"signals": []interface{}{},
				"count":   0,
				"display": "No pheromone signals found.",
			})
			return nil
		}

		// Format as text table
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%-10s %-10s %-10s %s\n", "TYPE", "PRIORITY", "STRENGTH", "CONTENT"))
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		for _, sig := range filtered {
			strength := "1.0"
			if sig.Strength != nil {
				strength = fmt.Sprintf("%.1f", *sig.Strength)
			}
			text := extractText(sig.Content)
			if len(text) > 60 {
				text = text[:57] + "..."
			}
			sb.WriteString(fmt.Sprintf("%-10s %-10s %-10s %s\n", sig.Type, sig.Priority, strength, text))
		}

		display := sb.String()
		fmt.Fprintf(stdout, "%s", display)

		// Build serializable signals list
		signals := make([]map[string]interface{}, len(filtered))
		for i, sig := range filtered {
			entry := map[string]interface{}{
				"id":       sig.ID,
				"type":     sig.Type,
				"priority": sig.Priority,
				"active":   sig.Active,
			}
			if sig.Strength != nil {
				entry["strength"] = *sig.Strength
			}
			signals[i] = entry
		}

		outputOK(map[string]interface{}{
			"signals": signals,
			"count":   len(filtered),
			"display": display,
		})
		return nil
	},
}

var pheromoneSnapshotInjectCmd = &cobra.Command{
	Use:   "pheromone-snapshot-inject",
	Short: "Copy active pheromone signals from one repo/worktree root into another",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		sourceRoot, _ := cmd.Flags().GetString("source-root")
		targetRoot, _ := cmd.Flags().GetString("target-root")
		if strings.TrimSpace(targetRoot) == "" {
			outputError(1, "--target-root is required", nil)
			return nil
		}

		result, err := syncPheromoneStores(sourceRoot, targetRoot, pheromoneSyncOptions{ActiveOnly: true})
		if err != nil {
			outputError(2, fmt.Sprintf("failed to inject pheromone snapshot: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"injected": true,
			"result":   result,
		})
		return nil
	},
}

var pheromoneMergeBackCmd = &cobra.Command{
	Use:   "pheromone-merge-back",
	Short: "Merge pheromone changes from one repo/worktree root back into another",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceRoot, _ := cmd.Flags().GetString("source-root")
		targetRoot, _ := cmd.Flags().GetString("target-root")
		if strings.TrimSpace(sourceRoot) == "" {
			outputError(1, "--source-root is required", nil)
			return nil
		}

		result, err := syncPheromoneStores(sourceRoot, targetRoot, pheromoneSyncOptions{})
		if err != nil {
			outputError(2, fmt.Sprintf("failed to merge pheromones: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"merged": true,
			"result": result,
		})
		return nil
	},
}

func init() {
	colonyPrimeCmd.Flags().Bool("compact", false, "Use 4000 char budget instead of 8000")

	pheromoneDisplayCmd.Flags().String("type", "", "Filter by signal type (FOCUS/REDIRECT/FEEDBACK)")
	pheromoneDisplayCmd.Flags().Bool("active-only", true, "Only show active signals")
	pheromoneSnapshotInjectCmd.Flags().String("source-root", "", "Repo or worktree root to copy active pheromones from (default current AETHER_ROOT)")
	pheromoneSnapshotInjectCmd.Flags().String("target-root", "", "Repo or worktree root to inject active pheromones into")
	pheromoneMergeBackCmd.Flags().String("source-root", "", "Repo or worktree root to merge pheromones from")
	pheromoneMergeBackCmd.Flags().String("target-root", "", "Repo or worktree root to merge pheromones into (default current AETHER_ROOT)")

	rootCmd.AddCommand(pheromonePrimeCmd)
	rootCmd.AddCommand(colonyPrimeCmd)
	rootCmd.AddCommand(pheromoneSnapshotInjectCmd)
	rootCmd.AddCommand(pheromoneMergeBackCmd)
	rootCmd.AddCommand(pheromoneDisplayCmd)
}

func colonyLifecycleSignalContext(state colony.ColonyState) string {
	lifecycleLine := fmt.Sprintf("Colony is %s. ", state.State)
	switch state.State {
	case colony.StateREADY:
		if len(state.Plan.Phases) == 0 {
			lifecycleLine += "Signals should guide planning scope and approach."
		} else {
			lifecycleLine += "Signals are pre-build guidance for upcoming execution."
		}
	case colony.StateEXECUTING:
		lifecycleLine += "Signals are active implementation constraints."
	case colony.StateBUILT:
		lifecycleLine += "Signals guide verification and learning extraction."
	default:
		lifecycleLine += "Signals provide ongoing context."
	}
	return lifecycleLine
}

// extractText extracts the text field from JSON content like {"text":"..."}.
func extractText(raw json.RawMessage) string {
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err == nil {
		if text, ok := m["text"]; ok {
			return text
		}
	}
	return strings.TrimSpace(string(raw))
}
