package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/spf13/cobra"
)

// --- midden-recent-failures ---

var middenRecentFailuresCmd = &cobra.Command{
	Use:   "midden-recent-failures",
	Short: "Return recent failure entries, newest first",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		limit, _ := cmd.Flags().GetInt("limit")

		var mf colony.MiddenFile
		if err := store.LoadJSON("midden.json", &mf); err != nil {
			outputOK(map[string]interface{}{"entries": []colony.MiddenEntry{}, "total": 0})
			return nil
		}

		// Sort by timestamp descending (newest first)
		sorted := make([]colony.MiddenEntry, len(mf.Entries))
		copy(sorted, mf.Entries)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Timestamp > sorted[j].Timestamp
		})

		if limit > 0 && len(sorted) > limit {
			sorted = sorted[:limit]
		}

		outputOK(map[string]interface{}{"entries": sorted, "total": len(sorted)})
		return nil
	},
}

// --- midden-review ---

var middenReviewCmd = &cobra.Command{
	Use:   "midden-review",
	Short: "Return unacknowledged entries grouped by category",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var mf colony.MiddenFile
		if err := store.LoadJSON("midden.json", &mf); err != nil {
			outputOK(map[string]interface{}{"groups": map[string][]colony.MiddenEntry{}, "total": 0})
			return nil
		}

		groups := map[string][]colony.MiddenEntry{}
		total := 0
		for _, e := range mf.Entries {
			if e.Acknowledged != nil && *e.Acknowledged {
				continue
			}
			groups[e.Category] = append(groups[e.Category], e)
			total++
		}

		outputOK(map[string]interface{}{"groups": groups, "total": total})
		return nil
	},
}

// --- midden-acknowledge ---

var middenAcknowledgeCmd = &cobra.Command{
	Use:   "midden-acknowledge",
	Short: "Mark entries as acknowledged by ID or category",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		id, _ := cmd.Flags().GetString("id")
		category, _ := cmd.Flags().GetString("category")

		if id == "" && category == "" {
			outputError(1, "--id or --category is required", nil)
			return nil
		}

		var mf colony.MiddenFile
		if err := store.LoadJSON("midden.json", &mf); err != nil {
			outputError(1, "midden.json not found", nil)
			return nil
		}

		now := time.Now().UTC().Format(time.RFC3339)
		count := 0
		for i := range mf.Entries {
			if id != "" && mf.Entries[i].ID == id {
				t := true
				mf.Entries[i].Acknowledged = &t
				mf.Entries[i].AcknowledgedAt = &now
				count++
			} else if category != "" && mf.Entries[i].Category == category {
				t := true
				mf.Entries[i].Acknowledged = &t
				mf.Entries[i].AcknowledgedAt = &now
				count++
			}
		}

		if err := store.SaveJSON("midden.json", mf); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"acknowledged": count})
		return nil
	},
}

// --- midden-search ---

var middenSearchCmd = &cobra.Command{
	Use:   "midden-search",
	Short: "Search failure entries by text",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		query := mustGetString(cmd, "query")
		if query == "" {
			return nil
		}
		query = strings.ToLower(query)

		var mf colony.MiddenFile
		if err := store.LoadJSON("midden.json", &mf); err != nil {
			outputOK(map[string]interface{}{"entries": []colony.MiddenEntry{}, "total": 0})
			return nil
		}

		var results []colony.MiddenEntry
		for _, e := range mf.Entries {
			if strings.Contains(strings.ToLower(e.Message), query) ||
				strings.Contains(strings.ToLower(e.Category), query) ||
				strings.Contains(strings.ToLower(e.ID), query) {
				results = append(results, e)
			}
		}

		outputOK(map[string]interface{}{"entries": results, "total": len(results), "query": query})
		return nil
	},
}

// --- midden-tag ---

var middenTagCmd = &cobra.Command{
	Use:   "midden-tag",
	Short: "Add a tag to a failure entry",
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
		tag := mustGetString(cmd, "tag")
		if tag == "" {
			return nil
		}

		var mf colony.MiddenFile
		if err := store.LoadJSON("midden.json", &mf); err != nil {
			outputError(1, "midden.json not found", nil)
			return nil
		}

		found := false
		for i := range mf.Entries {
			if mf.Entries[i].ID == id {
				mf.Entries[i].Tags = append(mf.Entries[i].Tags, tag)
				found = true
				break
			}
		}

		if !found {
			outputError(1, fmt.Sprintf("entry %q not found", id), nil)
			return nil
		}

		if err := store.SaveJSON("midden.json", mf); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"tagged": true, "id": id, "tag": tag})
		return nil
	},
}

// --- midden-collect ---

var middenCollectCmd = &cobra.Command{
	Use:   "midden-collect",
	Short: "Collect branch failures into main midden",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		branch, _ := cmd.Flags().GetString("branch")
		mergeSha, _ := cmd.Flags().GetString("merge-sha")

		var mf colony.MiddenFile
		if err := store.LoadJSON("midden.json", &mf); err != nil {
			outputError(1, "midden.json not found", nil)
			return nil
		}

		// Collect is a placeholder: tag entries from branch context
		count := 0
		for i := range mf.Entries {
			if mf.Entries[i].Source == branch {
				mf.Entries[i].Tags = append(mf.Entries[i].Tags, "merged:"+mergeSha)
				count++
			}
		}

		if err := store.SaveJSON("midden.json", mf); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"collected": count, "branch": branch, "merge_sha": mergeSha})
		return nil
	},
}

// --- midden-handle-revert ---

var middenHandleRevertCmd = &cobra.Command{
	Use:   "midden-handle-revert",
	Short: "Tag entries affected by a revert",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		sha := mustGetString(cmd, "sha")
		if sha == "" {
			return nil
		}

		var mf colony.MiddenFile
		if err := store.LoadJSON("midden.json", &mf); err != nil {
			outputError(1, "midden.json not found", nil)
			return nil
		}

		count := 0
		for i := range mf.Entries {
			if strings.Contains(mf.Entries[i].Source, sha) || strings.Contains(mf.Entries[i].ID, sha) {
				mf.Entries[i].Tags = append(mf.Entries[i].Tags, "reverted")
				count++
			}
		}

		if err := store.SaveJSON("midden.json", mf); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"tagged": count, "sha": sha})
		return nil
	},
}

// --- midden-cross-pr-analysis ---

var middenCrossPRAnalysisCmd = &cobra.Command{
	Use:   "midden-cross-pr-analysis",
	Short: "Detect failure patterns across multiple PRs",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var mf colony.MiddenFile
		if err := store.LoadJSON("midden.json", &mf); err != nil {
			outputError(1, "midden.json not found", nil)
			return nil
		}

		// Group by category and count
		catCounts := map[string]int{}
		for _, e := range mf.Entries {
			catCounts[e.Category]++
		}

		// Find patterns (categories with 2+ entries)
		type pattern struct {
			Category string `json:"category"`
			Count    int    `json:"count"`
		}
		var patterns []pattern
		for cat, count := range catCounts {
			if count >= 2 {
				patterns = append(patterns, pattern{Category: cat, Count: count})
			}
		}
		sort.Slice(patterns, func(i, j int) bool {
			return patterns[i].Count > patterns[j].Count
		})

		outputOK(map[string]interface{}{
			"patterns":      patterns,
			"pattern_count": len(patterns),
			"total_entries": len(mf.Entries),
		})
		return nil
	},
}

// --- midden-prune ---

var middenPruneCmd = &cobra.Command{
	Use:   "midden-prune",
	Short: "Remove entries older than N days",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		days := mustGetInt(cmd, "days")
		if days <= 0 {
			days = 30
		}

		var mf colony.MiddenFile
		if err := store.LoadJSON("midden.json", &mf); err != nil {
			outputError(1, "midden.json not found", nil)
			return nil
		}

		cutoff := time.Now().AddDate(0, 0, -days)
		before := len(mf.Entries)
		var kept []colony.MiddenEntry
		for _, e := range mf.Entries {
			t, err := time.Parse(time.RFC3339, e.Timestamp)
			if err != nil {
				kept = append(kept, e) // keep if unparseable
				continue
			}
			if t.After(cutoff) {
				kept = append(kept, e)
			}
		}
		mf.Entries = kept

		if err := store.SaveJSON("midden.json", mf); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"pruned":    before - len(mf.Entries),
			"remaining": len(mf.Entries),
			"before":    before,
			"days":      days,
		})
		return nil
	},
}

// --- midden-write ---

var middenWriteCmd = &cobra.Command{
	Use:   "midden-write",
	Short: "Write a failure record to the midden",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		category, _ := cmd.Flags().GetString("category")
		if category == "" {
			category = "general"
		}
		message, _ := cmd.Flags().GetString("message")
		source, _ := cmd.Flags().GetString("source")
		if source == "" {
			source = "unknown"
		}

		// Graceful degradation: if no message, return success but note it
		if message == "" {
			outputOK(map[string]interface{}{
				"success":  true,
				"warning":  "no_message_provided",
				"entry_id": nil,
			})
			return nil
		}

		// Generate entry ID: midden_{timestamp}_{pid}
		ts := time.Now().UTC()
		entryID := fmt.Sprintf("midden_%d_%d", ts.Unix(), os.Getpid())

		// Load or initialize midden.json
		var mf colony.MiddenFile
		if err := store.LoadJSON("midden.json", &mf); err != nil {
			mf = colony.MiddenFile{
				Version: "1.0.0",
				Entries: []colony.MiddenEntry{},
			}
		}
		if mf.Entries == nil {
			mf.Entries = []colony.MiddenEntry{}
		}

		// Build entry
		entry := colony.MiddenEntry{
			ID:        entryID,
			Timestamp: ts.Format(time.RFC3339),
			Category:  category,
			Source:    source,
			Message:   message,
			Reviewed:  false,
			Tags:      []string{},
		}

		// Append
		mf.Entries = append(mf.Entries, entry)

		// Save
		if err := store.SaveJSON("midden.json", mf); err != nil {
			outputError(2, fmt.Sprintf("failed to save midden: %v", err), nil)
			return nil
		}

		emitLifecycleCeremony(events.CeremonyTopicMiddenRecord, events.CeremonyPayload{
			TaskID:  entryID,
			Task:    category,
			Status:  "recorded",
			Message: message,
		}, "aether-midden")

		outputOK(map[string]interface{}{
			"success":      true,
			"entry_id":     entryID,
			"category":     category,
			"midden_total": len(mf.Entries),
		})
		return nil
	},
}

func init() {
	middenRecentFailuresCmd.Flags().Int("limit", 10, "Max entries to return")
	middenAcknowledgeCmd.Flags().String("id", "", "Entry ID to acknowledge")
	middenAcknowledgeCmd.Flags().String("category", "", "Category to acknowledge")
	middenSearchCmd.Flags().String("query", "", "Search query (required)")
	middenTagCmd.Flags().String("id", "", "Entry ID (required)")
	middenTagCmd.Flags().String("tag", "", "Tag to add (required)")
	middenCollectCmd.Flags().String("branch", "", "Branch name")
	middenCollectCmd.Flags().String("merge-sha", "", "Merge commit SHA")
	middenHandleRevertCmd.Flags().String("sha", "", "Revert commit SHA (required)")
	middenPruneCmd.Flags().Int("days", 30, "Remove entries older than N days")
	middenWriteCmd.Flags().String("category", "general", "Failure category")
	middenWriteCmd.Flags().String("message", "", "Failure message (required)")
	middenWriteCmd.Flags().String("source", "unknown", "Failure source")

	rootCmd.AddCommand(middenRecentFailuresCmd)
	rootCmd.AddCommand(middenReviewCmd)
	rootCmd.AddCommand(middenAcknowledgeCmd)
	rootCmd.AddCommand(middenSearchCmd)
	rootCmd.AddCommand(middenTagCmd)
	rootCmd.AddCommand(middenCollectCmd)
	rootCmd.AddCommand(middenHandleRevertCmd)
	rootCmd.AddCommand(middenCrossPRAnalysisCmd)
	rootCmd.AddCommand(middenPruneCmd)
	rootCmd.AddCommand(middenWriteCmd)
}
