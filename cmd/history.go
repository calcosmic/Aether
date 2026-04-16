package cmd

import (
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
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
			outputWorkflow(map[string]interface{}{"events": []interface{}{}}, renderHistoryVisual(map[string]interface{}{"events": []interface{}{}, "empty_message": "No colony history found."}))
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
			outputWorkflow(map[string]interface{}{"events": []interface{}{}}, renderHistoryVisual(map[string]interface{}{"events": []interface{}{}, "empty_message": "No events recorded."}))
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
			outputOK(buildHistoryResult(events))
			return nil
		}

		result := buildHistoryResult(events)
		result["filter"] = historyFilter
		result["limit"] = historyLimit
		outputWorkflow(result, renderHistoryVisual(result))
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

func buildHistoryResult(events []string) map[string]interface{} {
	entries := make([]map[string]interface{}, 0, len(events))
	for i := len(events) - 1; i >= 0; i-- {
		ts, et, src, msg := parseEvent(events[i])
		entries = append(entries, map[string]interface{}{
			"timestamp": ts,
			"type":      et,
			"source":    src,
			"message":   msg,
		})
	}

	return map[string]interface{}{
		"events": entries,
		"count":  len(entries),
	}
}
