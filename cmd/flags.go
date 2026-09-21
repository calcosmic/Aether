package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
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

// renderFlagsTable renders flags in the classic headed house style: one line
// per flag with a status icon, nested detail underneath — not a machine
// table. This was written during the v5.4.0-richness restoration but never
// wired in (flagsCmd kept the flat list); it is now THE /ant-flags renderer,
// locked by TestFlagsCmdUsesClassicRenderer.
func renderFlagsTable(entries []colony.FlagEntry) string {
	if len(entries) == 0 {
		return "🚩 Flags: none\n"
	}
	c := classifyOpenFlags(entries)
	blockers, issues, notes := len(c.Blockers), len(c.Issues), len(c.Notes)
	var b strings.Builder
	for _, entry := range entries {
		icon := "🚩"
		state := "open"
		if entry.Resolved {
			icon = "✅"
			state = "resolved"
		} else if entry.Acknowledged {
			state = "parked"
		}
		b.WriteString(fmt.Sprintf("%s %s (%s)\n", icon, strings.TrimSpace(entry.Description), state))
		detail := entry.ID
		if entry.Type != "" {
			detail += ", " + entry.Type
		}
		if entry.Source != "" {
			detail += ", from " + entry.Source
		}
		if entry.Phase != nil {
			detail += fmt.Sprintf(", phase %d", *entry.Phase)
		}
		b.WriteString("   └── " + detail + "\n")
	}
	fmt.Fprintf(&b, "\nOpen: %d blocker(s), %d issue(s), %d note(s)\n", blockers, issues, notes)
	if blockers > 0 {
		b.WriteString("Blockers stop advancement — resolve with `aether flag-resolve --id <id> --message \"what fixed it\"`, or /ant-unblock dispatches the Fixer.\n")
	}
	return b.String()
}
