package cmd

import (
	"fmt"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var (
	flagTypeFilter   string
	flagStatusFilter string
	flagListJSON     bool
	flagPhaseFilter  int
)

var flagsCmd = &cobra.Command{
	Use:     "flag-list",
	Short:   "List all flags",
	Args:    cobra.NoArgs,
	Aliases: []string{"flags"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var flags colony.FlagsFile
		// Try both file names for compatibility
		if err := store.LoadJSON("pending-decisions.json", &flags); err != nil {
			if err2 := store.LoadJSON("flags.json", &flags); err2 != nil {
				result := map[string]interface{}{
					"flags": []colony.FlagEntry{},
					"total": 0,
				}
				if flagListJSON {
					outputOK(result)
					return nil
				}
				outputWorkflow(result, renderFlagsVisual(result))
				return nil
			}
		}

		// Apply filters
		filtered := filterFlags(flags.Decisions)

		if flagListJSON {
			if filtered == nil {
				filtered = []colony.FlagEntry{}
			}
			outputOK(map[string]interface{}{"flags": filtered, "total": len(filtered)})
			return nil
		}

		result := map[string]interface{}{"flags": filtered, "total": len(filtered)}
		outputWorkflow(result, renderFlagsVisual(result))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(flagsCmd)
	flagsCmd.Flags().StringVar(&flagTypeFilter, "type", "", "Filter by type (blocker, issue, note)")
	flagsCmd.Flags().StringVar(&flagStatusFilter, "status", "", "Filter by status (active, resolved)")
	flagsCmd.Flags().BoolVar(&flagListJSON, "json", false, "Output as JSON")
	flagsCmd.Flags().IntVar(&flagPhaseFilter, "phase", 0, "Filter by phase number (0 means no filter)")
}

// filterFlags applies type and status filters to flag entries.
func filterFlags(entries []colony.FlagEntry) []colony.FlagEntry {
	var result []colony.FlagEntry
	for _, entry := range entries {
		if flagTypeFilter != "" && entry.Type != flagTypeFilter {
			continue
		}
		if flagStatusFilter == "active" && entry.Resolved {
			continue
		}
		if flagStatusFilter == "resolved" && !entry.Resolved {
			continue
		}
		if flagPhaseFilter > 0 && (entry.Phase == nil || *entry.Phase != flagPhaseFilter) {
			continue
		}
		result = append(result, entry)
	}
	return result
}

// renderFlagsTable displays flags in a formatted table.
func renderFlagsTable(entries []colony.FlagEntry) {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"ID", "Description", "Type", "Resolved", "Source"})

	for _, entry := range entries {
		resolved := "no"
		if entry.Resolved {
			resolved = "yes"
		}
		desc := entry.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		t.AppendRow(table.Row{entry.ID, desc, entry.Type, resolved, entry.Source})
	}
	t.SetStyle(table.StyleRounded)

	fmt.Fprintln(stdout, t.Render())
}
