package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/spf13/cobra"
)

var hiveSearchCmd = &cobra.Command{
	Use:   "hive-search [query]",
	Short: "Search colony learning with full-text search",
	Long:  "Search across worker summaries, gate failures, decisions, and memory text using natural language queries.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		classification, _ := cmd.Flags().GetString("classification")
		minConfidence, _ := cmd.Flags().GetFloat64("min-confidence")

		dbPath := filepath.Join(store.BasePath(), "colony.db")
		sqliteStore, err := learn.NewSQLiteColonyStore(dbPath)
		if err != nil {
			outputError(2, fmt.Sprintf("failed to open colony database: %v", err), nil)
			return nil
		}
		defer sqliteStore.Close()

		filter := learn.EntryFilter{
			Limit:          limit,
			Classification: learn.Classification(classification),
			MinConfidence:  minConfidence,
		}

		results, err := sqliteStore.Search(query, filter)
		if err != nil {
			outputError(2, fmt.Sprintf("search failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"query":   query,
			"results": results,
			"total":   len(results),
		})
		return nil
	},
}

func init() {
	hiveSearchCmd.Flags().Int("limit", 20, "Maximum results to return")
	hiveSearchCmd.Flags().String("classification", "", "Filter by classification (repo-local, hive-shareable)")
	hiveSearchCmd.Flags().Float64("min-confidence", 0, "Minimum confidence threshold")
	rootCmd.AddCommand(hiveSearchCmd)
}
