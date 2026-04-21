package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/trace"
	"github.com/spf13/cobra"
)

var traceReplayCmd = &cobra.Command{
	Use:   "trace-replay",
	Short: "Replay trace entries for a run",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		runID, _ := cmd.Flags().GetString("run-id")
		if runID == "" {
			outputError(1, "--run-id is required", nil)
			return nil
		}

		levelFilter, _ := cmd.Flags().GetString("level")
		sinceStr, _ := cmd.Flags().GetString("since")
		limit, _ := cmd.Flags().GetInt("limit")
		if limit <= 0 {
			limit = 100
		}

		var since time.Time
		if sinceStr != "" {
			var err error
			since, err = time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				outputError(1, fmt.Sprintf("invalid --since format: %v", err), nil)
				return nil
			}
		}

		levels := map[string]bool{}
		if levelFilter != "" {
			for _, l := range strings.Split(levelFilter, ",") {
				levels[strings.TrimSpace(l)] = true
			}
		}

		lines, err := store.ReadJSONL("trace.jsonl")
		if err != nil {
			outputOK(map[string]interface{}{
				"entries": []trace.TraceEntry{},
				"count":   0,
				"run_id":  runID,
			})
			return nil
		}

		var results []trace.TraceEntry
		for _, line := range lines {
			var entry trace.TraceEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				continue
			}
			if entry.RunID != runID {
				continue
			}
			if len(levels) > 0 && !levels[string(entry.Level)] {
				continue
			}
			if !since.IsZero() {
				entryTime, err := time.Parse(time.RFC3339, entry.Timestamp)
				if err == nil && entryTime.Before(since) {
					continue
				}
			}
			results = append(results, entry)
			if len(results) >= limit {
				break
			}
		}

		outputOK(map[string]interface{}{
			"entries": results,
			"count":   len(results),
			"run_id":  runID,
		})
		return nil
	},
}

var traceExportCmd = &cobra.Command{
	Use:   "trace-export",
	Short: "Export trace entries for a run to JSON",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		runID, _ := cmd.Flags().GetString("run-id")
		if runID == "" {
			outputError(1, "--run-id is required", nil)
			return nil
		}

		outputPath, _ := cmd.Flags().GetString("output")
		levelFilter, _ := cmd.Flags().GetString("level")
		sinceStr, _ := cmd.Flags().GetString("since")

		var since time.Time
		if sinceStr != "" {
			var err error
			since, err = time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				outputError(1, fmt.Sprintf("invalid --since format: %v", err), nil)
				return nil
			}
		}

		levels := map[string]bool{}
		if levelFilter != "" {
			for _, l := range strings.Split(levelFilter, ",") {
				levels[strings.TrimSpace(l)] = true
			}
		}

		lines, err := store.ReadJSONL("trace.jsonl")
		if err != nil {
			outputOK(map[string]interface{}{
				"entries": []trace.TraceEntry{},
				"count":   0,
				"run_id":  runID,
			})
			return nil
		}

		var results []trace.TraceEntry
		for _, line := range lines {
			var entry trace.TraceEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				continue
			}
			if entry.RunID != runID {
				continue
			}
			if len(levels) > 0 && !levels[string(entry.Level)] {
				continue
			}
			if !since.IsZero() {
				entryTime, err := time.Parse(time.RFC3339, entry.Timestamp)
				if err == nil && entryTime.Before(since) {
					continue
				}
			}
			results = append(results, entry)
		}

		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			outputError(2, fmt.Sprintf("failed to marshal export: %v", err), nil)
			return nil
		}

		if outputPath != "" {
			if err := os.WriteFile(outputPath, append(data, '\n'), 0644); err != nil {
				outputError(2, fmt.Sprintf("failed to write export: %v", err), nil)
				return nil
			}
			outputOK(map[string]interface{}{
				"exported": true,
				"count":    len(results),
				"run_id":   runID,
				"output":   outputPath,
			})
		} else {
			fmt.Fprintln(stdout, string(data))
		}
		return nil
	},
}

var traceSummaryCmd = &cobra.Command{
	Use:   "trace-summary",
	Short: "Summarize trace entries for a run",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		runID, _ := cmd.Flags().GetString("run-id")
		if runID == "" {
			outputError(1, "--run-id is required", nil)
			return nil
		}

		lines, err := store.ReadJSONL("trace.jsonl")
		if err != nil {
			outputOK(map[string]interface{}{
				"run_id": runID,
				"summary": map[string]interface{}{
					"total_entries": 0,
				},
			})
			return nil
		}

		var entries []trace.TraceEntry
		for _, line := range lines {
			var entry trace.TraceEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				continue
			}
			if entry.RunID == runID {
				entries = append(entries, entry)
			}
		}

		summary := summarizeTraceEntries(entries)
		summary["run_id"] = runID
		summary["total_entries"] = len(entries)
		outputOK(summary)
		return nil
	},
}

var traceInspectCmd = &cobra.Command{
	Use:   "trace-inspect",
	Short: "Inspect a focused timeline from a trace run",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		runID, _ := cmd.Flags().GetString("run-id")
		if runID == "" {
			outputError(1, "--run-id is required", nil)
			return nil
		}

		focus, _ := cmd.Flags().GetString("focus")
		validFocus := map[string]bool{
			"state":        true,
			"phase":        true,
			"error":        true,
			"token":        true,
			"intervention": true,
			"artifact":     true,
		}
		if focus != "" && !validFocus[focus] {
			outputError(1, fmt.Sprintf("invalid --focus %q (must be one of: state, phase, error, token, intervention, artifact)", focus), nil)
			return nil
		}

		lines, err := store.ReadJSONL("trace.jsonl")
		if err != nil {
			outputOK(map[string]interface{}{
				"run_id":   runID,
				"focus":    focus,
				"timeline": []trace.TraceEntry{},
				"suggestions": []string{},
			})
			return nil
		}

		var timeline []trace.TraceEntry
		for _, line := range lines {
			var entry trace.TraceEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				continue
			}
			if entry.RunID != runID {
				continue
			}
			if focus != "" && string(entry.Level) != focus {
				continue
			}
			timeline = append(timeline, entry)
		}

		suggestions := generateInspectSuggestions(timeline, focus)
		outputOK(map[string]interface{}{
			"run_id":      runID,
			"focus":       focus,
			"count":       len(timeline),
			"timeline":    timeline,
			"suggestions": suggestions,
		})
		return nil
	},
}

// summarizeTraceEntries computes aggregate stats from trace entries.
func summarizeTraceEntries(entries []trace.TraceEntry) map[string]interface{} {
	var firstTime, lastTime time.Time
	stateTransitions := []map[string]interface{}{}
	phases := map[int]bool{}
	var errorCount int
	var errorSeverities []string
	var totalInputTokens, totalOutputTokens int64
	var totalCost float64
	var interventionCount int
	var interventionTypes []string

	for _, entry := range entries {
		t, _ := time.Parse(time.RFC3339, entry.Timestamp)
		if firstTime.IsZero() || t.Before(firstTime) {
			firstTime = t
		}
		if lastTime.IsZero() || t.After(lastTime) {
			lastTime = t
		}

		switch entry.Level {
		case trace.TraceLevelState:
			if entry.Topic == "state.transition" {
				stateTransitions = append(stateTransitions, map[string]interface{}{
					"timestamp": entry.Timestamp,
					"from":      entry.Payload["from"],
					"to":        entry.Payload["to"],
				})
			}
		case trace.TraceLevelPhase:
			if phaseNum, ok := entry.Payload["phase"].(float64); ok {
				phases[int(phaseNum)] = true
			}
		case trace.TraceLevelError:
			errorCount++
			if sev, ok := entry.Payload["severity"].(string); ok && sev != "" {
				errorSeverities = append(errorSeverities, sev)
			}
		case trace.TraceLevelToken:
			if it, ok := entry.Payload["input_tokens"].(float64); ok {
				totalInputTokens += int64(it)
			}
			if ot, ok := entry.Payload["output_tokens"].(float64); ok {
				totalOutputTokens += int64(ot)
			}
			if c, ok := entry.Payload["usd_cost"].(float64); ok {
				totalCost += c
			}
		case trace.TraceLevelIntervention:
			interventionCount++
			interventionTypes = append(interventionTypes, entry.Topic)
		}
	}

	duration := ""
	if !firstTime.IsZero() && !lastTime.IsZero() {
		duration = lastTime.Sub(firstTime).String()
	}

	phaseList := make([]int, 0, len(phases))
	for p := range phases {
		phaseList = append(phaseList, p)
	}
	sort.Ints(phaseList)

	return map[string]interface{}{
		"duration":           duration,
		"state_transitions":  stateTransitions,
		"state_transition_count": len(stateTransitions),
		"phases":             phaseList,
		"phase_count":        len(phaseList),
		"errors": map[string]interface{}{
			"count":      errorCount,
			"severities": errorSeverities,
		},
		"token_usage": map[string]interface{}{
			"total_input_tokens":  totalInputTokens,
			"total_output_tokens": totalOutputTokens,
			"total_usd_cost":      totalCost,
		},
		"interventions": map[string]interface{}{
			"count": len(interventionTypes),
			"types": interventionTypes,
		},
	}
}

// generateInspectSuggestions produces human-readable hints from a timeline.
func generateInspectSuggestions(timeline []trace.TraceEntry, focus string) []string {
	var suggestions []string
	if focus == "error" {
		phaseErrors := map[int]int{}
		for _, entry := range timeline {
			if entry.Level != trace.TraceLevelError {
				continue
			}
			if phaseNum, ok := entry.Payload["phase"].(float64); ok {
				phaseErrors[int(phaseNum)]++
			}
		}
		for phaseNum, count := range phaseErrors {
			if count >= 3 {
				suggestions = append(suggestions, fmt.Sprintf("%d errors during phase %d", count, phaseNum))
			}
		}
		if len(suggestions) == 0 && len(timeline) > 0 {
			suggestions = append(suggestions, fmt.Sprintf("%d error(s) recorded", len(timeline)))
		}
	}
	if focus == "phase" {
		phaseStatuses := map[string]int{}
		for _, entry := range timeline {
			if status, ok := entry.Payload["status"].(string); ok {
				phaseStatuses[status]++
			}
		}
		for status, count := range phaseStatuses {
			suggestions = append(suggestions, fmt.Sprintf("%d phase entries with status %q", count, status))
		}
	}
	if focus == "token" && len(timeline) > 0 {
		var totalCost float64
		for _, entry := range timeline {
			if c, ok := entry.Payload["usd_cost"].(float64); ok {
				totalCost += c
			}
		}
		suggestions = append(suggestions, fmt.Sprintf("Total estimated cost: $%.4f", totalCost))
	}
	return suggestions
}

func init() {
	traceReplayCmd.Flags().String("run-id", "", "Filter by run ID (required)")
	traceReplayCmd.Flags().String("level", "", "Filter by level (comma-separated)")
	traceReplayCmd.Flags().String("since", "", "Filter by timestamp (RFC3339)")
	traceReplayCmd.Flags().Int("limit", 100, "Max entries (default 100)")

	traceExportCmd.Flags().String("run-id", "", "Filter by run ID (required)")
	traceExportCmd.Flags().String("level", "", "Filter by level (comma-separated)")
	traceExportCmd.Flags().String("since", "", "Filter by timestamp (RFC3339)")
	traceExportCmd.Flags().String("output", "", "Output file path (default stdout)")

	traceSummaryCmd.Flags().String("run-id", "", "Filter by run ID (required)")

	traceInspectCmd.Flags().String("run-id", "", "Filter by run ID (required)")
	traceInspectCmd.Flags().String("focus", "", "Focus on one level: state, phase, error, token, intervention, artifact")

	rootCmd.AddCommand(traceReplayCmd)
	rootCmd.AddCommand(traceExportCmd)
	rootCmd.AddCommand(traceSummaryCmd)
	rootCmd.AddCommand(traceInspectCmd)
}
