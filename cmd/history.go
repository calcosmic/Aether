package cmd

import (
	"fmt"
	"strings"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var (
	historyLimit  int
	historyFilter string
	historyJSON   bool
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show colony event history",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			if historyJSON {
				outputOK(map[string]interface{}{
					"events": []interface{}{},
				})
				return nil
			}
			fmt.Fprintln(stdout, "No colony history found.")
			return nil
		}

		events := state.Events
		if len(events) == 0 {
			if historyJSON {
				outputOK(map[string]interface{}{
					"events": []interface{}{},
				})
				return nil
			}
			fmt.Fprintln(stdout, "No events recorded.")
			return nil
		}

		// Apply filter
		if historyFilter != "" {
			var filtered []string
			for _, evt := range events {
				if strings.Contains(evt, historyFilter) {
					filtered = append(filtered, evt)
				}
			}
			events = filtered
		}

		// Apply limit (show newest first)
		if historyLimit > 0 && len(events) > historyLimit {
			events = events[len(events)-historyLimit:]
		}

		if historyJSON {
			type historyEntry struct {
				Timestamp string `json:"timestamp"`
				Type      string `json:"type"`
				Source    string `json:"source"`
				Message   string `json:"message"`
			}
			var entries []historyEntry
			for i := len(events) - 1; i >= 0; i-- {
				ts, et, src, msg := parseEvent(events[i])
				entries = append(entries, historyEntry{Timestamp: ts, Type: et, Source: src, Message: msg})
			}
			if entries == nil {
				entries = []historyEntry{}
			}
			outputOK(map[string]interface{}{
				"events": entries,
			})
			return nil
		}

		// Display events in reverse chronological order
		renderHistoryTable(events)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.Flags().IntVar(&historyLimit, "limit", 20, "Maximum number of events to show")
	historyCmd.Flags().StringVar(&historyFilter, "filter", "", "Filter events by type or text")
	historyCmd.Flags().BoolVar(&historyJSON, "json", false, "Output as JSON")
}

// parseEvent splits a pipe-delimited event string into parts.
// Format: "timestamp|type|source|message"
func parseEvent(event string) (timestamp, eventType, source, message string) {
	parts := strings.SplitN(event, "|", 4)
	switch len(parts) {
	case 4:
		return parts[0], parts[1], parts[2], parts[3]
	case 3:
		return parts[0], parts[1], parts[2], ""
	case 2:
		return parts[0], parts[1], "", ""
	default:
		return event, "", "", ""
	}
}

// renderHistoryTable displays events in a formatted table.
func renderHistoryTable(events []string) {
	if len(events) == 0 {
		fmt.Fprintln(stdout, "No events to display.")
		return
	}

	t := table.NewWriter()
	t.AppendHeader(table.Row{"Timestamp", "Type", "Source", "Message"})

	// Display newest first
	for i := len(events) - 1; i >= 0; i-- {
		timestamp, eventType, source, message := parseEvent(events[i])
		// Format timestamp for display
		ts := formatTimestamp(timestamp)
		// Truncate message for display
		if len(message) > 50 {
			message = message[:47] + "..."
		}
		t.AppendRow(table.Row{ts, eventType, source, message})
	}
	t.SetStyle(table.StyleRounded)

	fmt.Fprintln(stdout, t.Render())
}
