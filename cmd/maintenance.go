package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
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

		// Worker-debug artifacts (D-04): reported in both dry-run and
		// confirmed mode — unlike the pheromones `removed` field below,
		// `worker_debug_prunable` is deliberately non-zero without
		// --confirm, so data-clean without --confirm is still useful as an
		// inspection command rather than reporting zeros for everything.
		workerDebugDir := filepath.Join(store.BasePath(), "worker-debug")
		wdTotal, wdPrunable, wdRemoved, wdErr := pruneWorkerDebugDirectory(workerDebugDir, confirm)
		if wdErr != nil {
			outputError(3, fmt.Sprintf("failed to prune worker-debug artifacts: %v", wdErr), nil)
			return nil
		}

		// Load pheromones.json
		data, err := store.ReadFile("pheromones.json")
		if err != nil {
			outputOK(map[string]interface{}{
				"scanned":               true,
				"removed":               0,
				"dry_run":               !confirm,
				"worker_debug_total":    wdTotal,
				"worker_debug_prunable": wdPrunable,
				"worker_debug_removed":  wdRemoved,
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
				"scanned":               true,
				"removed":               0,
				"dry_run":               !confirm,
				"worker_debug_total":    wdTotal,
				"worker_debug_prunable": wdPrunable,
				"worker_debug_removed":  wdRemoved,
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
			"scanned":               true,
			"removed":               reportedRemoved,
			"dry_run":               !confirm,
			"worker_debug_total":    wdTotal,
			"worker_debug_prunable": wdPrunable,
			"worker_debug_removed":  wdRemoved,
		})
		return nil
	},
}

// pruneWorkerDebugDirectory applies the shared codex.WorkerDebugRetentionMaxAge
// / codex.WorkerDebugRetentionMaxFiles policy against dir: files older than
// the max age are counted as prunable first, then, if more than the file cap
// survive, the oldest of those are counted as prunable too. total is the
// artifact count before pruning; prunable is what WOULD be removed, reported
// regardless of confirm; removed is what was actually deleted, which is
// always 0 when confirm is false. A missing directory is not an error —
// dataCleanCmd zeros all three and continues.
func pruneWorkerDebugDirectory(dir string, confirm bool) (total, prunable, removed int, err error) {
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return 0, 0, 0, nil
		}
		return 0, 0, 0, readErr
	}

	type debugArtifact struct {
		name    string
		modTime time.Time
	}
	var artifacts []debugArtifact
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		artifacts = append(artifacts, debugArtifact{name: entry.Name(), modTime: info.ModTime()})
	}
	total = len(artifacts)

	cutoff := time.Now().Add(-codex.WorkerDebugRetentionMaxAge)
	var toPrune []string
	var survivors []debugArtifact
	for _, a := range artifacts {
		if a.modTime.Before(cutoff) {
			toPrune = append(toPrune, a.name)
			continue
		}
		survivors = append(survivors, a)
	}
	if len(survivors) > codex.WorkerDebugRetentionMaxFiles {
		sort.Slice(survivors, func(i, j int) bool { return survivors[i].modTime.Before(survivors[j].modTime) })
		excess := len(survivors) - codex.WorkerDebugRetentionMaxFiles
		for i := 0; i < excess; i++ {
			toPrune = append(toPrune, survivors[i].name)
		}
	}
	prunable = len(toPrune)
	if !confirm {
		return total, prunable, 0, nil
	}
	for _, name := range toPrune {
		if removeErr := os.Remove(filepath.Join(dir, name)); removeErr != nil {
			log.Printf("data-clean: failed to remove worker-debug artifact %s: %v", name, removeErr)
			continue
		}
		removed++
	}
	return total, prunable, removed, nil
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

// isTestArtifact checks if a pheromone signal matches test artifact patterns.
// Used by data-clean to identify and remove test/demo signals.
func isTestArtifact(signal map[string]interface{}) bool {
	id, _ := signal["id"].(string)
	contentRaw := signal["content"]
	content := ""
	if contentMap, ok := contentRaw.(map[string]interface{}); ok {
		content, _ = contentMap["text"].(string)
	} else if contentStr, ok := contentRaw.(string); ok {
		content = contentStr
	}

	if strings.HasPrefix(id, "test_") || strings.HasPrefix(id, "demo_") {
		return true
	}

	lower := strings.ToLower(content)
	if strings.Contains(lower, "test signal") || strings.Contains(lower, "demo pattern") {
		return true
	}

	return false
}

func init() {
	dataCleanCmd.Flags().Bool("confirm", false, "Confirm removal (default: dry-run)")
	backupPruneGlobalCmd.Flags().Int("cap", 50, "Maximum backups to keep")

	rootCmd.AddCommand(dataCleanCmd)
	rootCmd.AddCommand(backupPruneGlobalCmd)
	rootCmd.AddCommand(tempCleanCmd)
}
