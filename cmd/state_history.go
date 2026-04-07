package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var (
	stateHistoryDiff bool
	stateHistoryTail int
	stateHistoryJSON bool
)

var stateHistoryCmd = &cobra.Command{
	Use:   "state-history",
	Short: "View colony state mutation history",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		logger := storage.NewAuditLogger(store)
		entries, err := logger.ReadHistory(stateHistoryTail)
		if err != nil || len(entries) == 0 {
			if stateHistoryJSON {
				outputOK(map[string]interface{}{"entries": []interface{}{}})
			} else {
				fmt.Fprintln(stdout, "No mutation history found.")
			}
			return nil
		}

		// JSON output mode
		if stateHistoryJSON {
			outputOK(map[string]interface{}{"entries": entries})
			return nil
		}

		// Diff mode -- show full before/after for each entry
		if stateHistoryDiff {
			renderDiffOutput(entries)
			return nil
		}

		// Default: compact table (like git log --oneline)
		renderStateHistoryTable(entries)
		return nil
	},
}

// renderStateHistoryTable displays a compact table of mutations.
// Format: Timestamp | Command | Summary | Destructive
func renderStateHistoryTable(entries []storage.AuditEntry) {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Timestamp", "Command", "Summary", "Destructive"})

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		ts := formatAuditTimestamp(e.Timestamp)
		summary := e.Summary
		if summary == "" {
			summary = e.Command
		}
		if len(summary) > 60 {
			summary = summary[:57] + "..."
		}
		destructive := ""
		if e.Destructive {
			destructive = "YES"
		}
		t.AppendRow(table.Row{ts, e.Command, summary, destructive})
	}
	t.SetStyle(table.StyleRounded)
	fmt.Fprintln(stdout, t.Render())
}

// renderDiffOutput shows full before/after for each audit entry.
func renderDiffOutput(entries []storage.AuditEntry) {
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		ts := formatAuditTimestamp(e.Timestamp)
		fmt.Fprintf(stdout, "--- Entry %d ---\n", len(entries)-i)
		fmt.Fprintf(stdout, "Timestamp:   %s\n", ts)
		fmt.Fprintf(stdout, "Command:     %s\n", e.Command)
		fmt.Fprintf(stdout, "Summary:     %s\n", e.Summary)
		fmt.Fprintf(stdout, "Checksum:    %s\n", e.Checksum)
		fmt.Fprintf(stdout, "Destructive: %v\n", e.Destructive)
		if e.Path != "" {
			fmt.Fprintf(stdout, "Path:        %s\n", e.Path)
		}
		if len(e.Before) > 0 {
			fmt.Fprintln(stdout, "Before:")
			var pretty bytes.Buffer
			if json.Indent(&pretty, e.Before, "  ", "  ") == nil {
				fmt.Fprintln(stdout, "  "+pretty.String())
			} else {
				fmt.Fprintln(stdout, "  "+string(e.Before))
			}
		}
		if len(e.After) > 0 {
			fmt.Fprintln(stdout, "After:")
			var pretty bytes.Buffer
			if json.Indent(&pretty, e.After, "  ", "  ") == nil {
				fmt.Fprintln(stdout, "  "+pretty.String())
			} else {
				fmt.Fprintln(stdout, "  "+string(e.After))
			}
		}
		fmt.Fprintln(stdout)
	}
}

// formatAuditTimestamp converts RFC3339Nano to human-readable format.
func formatAuditTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return ts
		}
	}
	return t.Format("2006-01-02 15:04:05")
}

func init() {
	stateHistoryCmd.Flags().BoolVar(&stateHistoryDiff, "diff", false, "Show full before/after diffs")
	stateHistoryCmd.Flags().IntVar(&stateHistoryTail, "tail", 20, "Number of recent entries to show (0 = all)")
	stateHistoryCmd.Flags().BoolVar(&stateHistoryJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(stateHistoryCmd)
}
