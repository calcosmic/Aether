package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

var dataCleanCmd = &cobra.Command{
	Use:   "data-clean",
	Short: "Remove test artifacts from colony data files",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		confirm, _ := cmd.Flags().GetBool("confirm")

		// Load pheromones.json
		data, err := store.ReadFile("pheromones.json")
		if err != nil {
			outputOK(map[string]interface{}{
				"scanned": true,
				"removed": 0,
				"dry_run": !confirm,
			})
			return nil
		}

		var pheromonesFile map[string]interface{}
		if err := json.Unmarshal(data, &pheromonesFile); err != nil {
			outputError(1, fmt.Sprintf("failed to parse pheromones.json: %v", err), nil)
			return nil
		}

		rawSignals, _ := pheromonesFile["signals"].([]interface{})
		if rawSignals == nil {
			outputOK(map[string]interface{}{
				"scanned": true,
				"removed": 0,
				"dry_run": !confirm,
			})
			return nil
		}

		var kept []interface{}
		removed := 0
		for _, raw := range rawSignals {
			signal, ok := raw.(map[string]interface{})
			if !ok {
				kept = append(kept, raw)
				continue
			}
			if isTestArtifact(signal) {
				removed++
				continue
			}
			kept = append(kept, raw)
		}

		if confirm && removed > 0 {
			pheromonesFile["signals"] = kept
			if err := store.SaveJSON("pheromones.json", pheromonesFile); err != nil {
				outputError(2, fmt.Sprintf("failed to save pheromones.json: %v", err), nil)
				return nil
			}
		}

		// In dry-run mode, report 0 removed (nothing was actually removed)
		reportedRemoved := removed
		if !confirm {
			reportedRemoved = 0
		}

		outputOK(map[string]interface{}{
			"scanned": true,
			"removed": reportedRemoved,
			"dry_run": !confirm,
		})
		return nil
	},
}

var backupPruneGlobalCmd = &cobra.Command{
	Use:   "backup-prune-global",
	Short: "Prune old backups to a cap",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		cap, _ := cmd.Flags().GetInt("cap")
		if cap <= 0 {
			cap = 50
		}

		backupDir := filepath.Join(store.BasePath(), "backups")

		entries, err := os.ReadDir(backupDir)
		if err != nil {
			if os.IsNotExist(err) {
				outputOK(map[string]interface{}{
					"pruned":     0,
					"kept":       0,
					"dir_exists": false,
				})
				return nil
			}
			outputError(1, fmt.Sprintf("failed to read backup directory: %v", err), nil)
			return nil
		}

		// Collect files with mod times
		type fileInfo struct {
			name    string
			modTime time.Time
		}
		var files []fileInfo
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			files = append(files, fileInfo{name: entry.Name(), modTime: info.ModTime()})
		}

		// Sort by mod time ascending (oldest first)
		sort.Slice(files, func(i, j int) bool {
			return files[i].modTime.Before(files[j].modTime)
		})

		if len(files) <= cap {
			outputOK(map[string]interface{}{
				"pruned": 0,
				"kept":   len(files),
			})
			return nil
		}

		pruneCount := len(files) - cap
		for i := 0; i < pruneCount; i++ {
			if err := os.Remove(filepath.Join(backupDir, files[i].name)); err != nil {
				log.Printf("data-clean: failed to remove backup %s: %v", files[i].name, err)
			}
		}

		outputOK(map[string]interface{}{
			"pruned": pruneCount,
			"kept":   cap,
		})
		return nil
	},
}

var tempCleanCmd = &cobra.Command{
	Use:   "temp-clean",
	Short: "Remove temp files older than 7 days",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		aetherRoot := storage.ResolveAetherRoot(context.Background())
		tempDir := filepath.Join(aetherRoot, ".aether", "temp")

		entries, err := os.ReadDir(tempDir)
		if err != nil {
			if os.IsNotExist(err) {
				outputOK(map[string]interface{}{
					"cleaned":    0,
					"dir_exists": false,
				})
				return nil
			}
			outputError(1, fmt.Sprintf("failed to read temp directory: %v", err), nil)
			return nil
		}

		cutoff := time.Now().Add(-7 * 24 * time.Hour)
		cleaned := 0

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(tempDir, entry.Name()))
				cleaned++
			}
		}

		outputOK(map[string]interface{}{
			"cleaned": cleaned,
		})
		return nil
	},
}

func init() {
	dataCleanCmd.Flags().Bool("confirm", false, "Confirm removal (default: dry-run)")
	backupPruneGlobalCmd.Flags().Int("cap", 50, "Maximum backups to keep")

	rootCmd.AddCommand(dataCleanCmd)
	rootCmd.AddCommand(backupPruneGlobalCmd)
	rootCmd.AddCommand(tempCleanCmd)
}
