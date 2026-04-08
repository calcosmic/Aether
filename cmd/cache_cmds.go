package cmd

import (
	"fmt"
	"time"

	"github.com/calcosmic/Aether/pkg/cache"
	"github.com/spf13/cobra"
)

var cacheCleanCmd = &cobra.Command{
	Use:   "cache-clean",
	Short: "Remove all session cache files from the data directory",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		sc := cache.NewSessionCache(store.BasePath())
		removed, err := sc.Clear()
		if err != nil {
			outputError(1, fmt.Sprintf("cache clean failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"files_removed": removed,
		})
		return nil
	},
}

var cacheCleanStaleCmd = &cobra.Command{
	Use:   "cache-clean-stale",
	Short: "Remove session cache files older than 24 hours",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		maxAge, _ := cmd.Flags().GetDuration("max-age")
		if maxAge == 0 {
			maxAge = 24 * time.Hour
		}

		sc := cache.NewSessionCache(store.BasePath())
		removed, err := sc.ClearStale(maxAge)
		if err != nil {
			outputError(1, fmt.Sprintf("cache clean-stale failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"files_removed": removed,
		})
		return nil
	},
}

func init() {
	cacheCleanStaleCmd.Flags().Duration("max-age", 24*time.Hour, "Maximum age for cache files (default: 24h)")

	rootCmd.AddCommand(cacheCleanCmd)
	rootCmd.AddCommand(cacheCleanStaleCmd)
}
