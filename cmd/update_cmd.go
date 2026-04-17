package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/calcosmic/Aether/pkg/downloader"
	"github.com/spf13/cobra"
)

// updateCmd implements "aether update" which syncs companion files from the
// hub and optionally downloads a new binary.
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Aether companion files and optionally the binary",
	Long: `Update Aether by syncing companion files from the distribution hub
(~/.aether/system/) to the local .aether/ directory.

This updates slash commands, agent definitions, skills, templates, and docs.
Local user data (COLONY_STATE.json, pheromones, etc.) is never overwritten.

Use --download-binary to also fetch a Go binary from GitHub Releases.`,
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

var (
	updateDownloadBinary bool
	updateBinaryVersion  string
	updateDryRun         bool
	updateForce          bool
)

func init() {
	updateCmd.Flags().Bool("download-binary", false, "Also download a binary from GitHub Releases")
	updateCmd.Flags().String("binary-version", "", "Binary version to download (default: resolved installed version)")
	updateCmd.Flags().Bool("dry-run", false, "Show what would be updated without making changes")
	updateCmd.Flags().Bool("force", false, "Overwrite modified companion files and remove stale ones")

	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	// Check hub exists
	hubDir := filepath.Join(homeDir, ".aether")
	hubVersionFile := filepath.Join(hubDir, "version.json")
	if _, err := os.Stat(hubVersionFile); os.IsNotExist(err) {
		outputErrorMessage("Aether hub not installed. Run \"aether install\" first.")
		return nil
	}

	// Read hub version for comparison
	hubVersion := readHubVersion(hubVersionFile)

	// Get repo directory
	repoDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	// Sync companion files from hub
	if dryRun {
		mode := "safe (new files only)"
		if force {
			mode = "force (overwrite changed + remove stale)"
		}
		result := map[string]interface{}{
			"message":       fmt.Sprintf("Dry run — would sync companion files from hub [%s]", mode),
			"hub_version":   hubVersion,
			"local_version": resolveVersion(),
			"force":         force,
			"actions": []string{
				"Sync .aether/ system files (commands, agents, skills, templates, docs)",
				"Refresh repo-level Codex guidance (AGENTS.md, .codex/CODEX.md) when managed by Aether",
				"Sync .claude/commands/ant/",
				"Sync .claude/agents/ant/",
				"Sync .codex/agents/",
				"Sync .codex/skills/",
				"Sync .opencode/commands/ant/",
				"Sync .opencode/agents/",
			},
		}
		outputWorkflow(result, renderUpdateVisual(repoDir, hubVersion, resolveVersion(), force, true, []map[string]interface{}{
			{"label": "System files", "copied": 0, "skipped": 0},
			{"label": "Commands (claude)", "copied": 0, "skipped": 0},
			{"label": "Agents (claude)", "copied": 0, "skipped": 0},
			{"label": "Agents (codex)", "copied": 0, "skipped": 0},
			{"label": "Skills (codex)", "copied": 0, "skipped": 0},
			{"label": "Commands (opencode)", "copied": 0, "skipped": 0},
			{"label": "Agents (opencode)", "copied": 0, "skipped": 0},
		}, 0, 0, nil))
	} else {
		syncResult := runUpdateSync(hubDir, repoDir, force)
		if len(syncResult.errors) > 0 {
			outputError(2, fmt.Sprintf("update failed with %d sync error(s)", len(syncResult.errors)), map[string]interface{}{
				"hub_version":   hubVersion,
				"local_version": resolveVersion(),
				"force":         force,
				"details":       syncResult.details,
			})
			return nil
		}
		docResults, docCopied, docSkipped, docErrors := syncCodexProjectDocs(filepath.Join(hubDir, "system"), repoDir)
		syncResult.details = append(syncResult.details, docResults...)
		syncResult.copied += docCopied
		syncResult.skipped += docSkipped
		if len(docErrors) > 0 {
			syncResult.errors = append(syncResult.errors, docErrors...)
			outputError(2, fmt.Sprintf("update failed with %d sync error(s)", len(syncResult.errors)), map[string]interface{}{
				"hub_version":   hubVersion,
				"local_version": resolveVersion(),
				"force":         force,
				"details":       syncResult.details,
			})
			return nil
		}
		mirrorRestored := false
		if restored, err := ensureLegacySessionMirror(store); err == nil {
			mirrorRestored = restored
		}
		restartTargets := codexRestartTargets(syncResult.details)
		message := fmt.Sprintf("Updated: %d files copied, %d unchanged", syncResult.copied, syncResult.skipped)
		if restartNote := codexRestartMessage(restartTargets); restartNote != "" {
			message += ". " + restartNote
		}
		result := map[string]interface{}{
			"message":                 message,
			"hub_version":             hubVersion,
			"local_version":           resolveVersion(),
			"force":                   force,
			"details":                 syncResult.details,
			"legacy_session_restored": mirrorRestored,
			"codex_restart_required":  len(restartTargets) > 0,
			"codex_restart_targets":   restartTargets,
		}
		outputWorkflow(result, renderUpdateVisual(repoDir, hubVersion, resolveVersion(), force, false, syncResult.details, syncResult.copied, syncResult.skipped, restartTargets))
	}

	// Download binary if requested
	downloadBinary, _ := cmd.Flags().GetBool("download-binary")
	if downloadBinary && !dryRun {
		versionFlag, _ := cmd.Flags().GetString("binary-version")
		if normalizeVersion(versionFlag) == "latest" {
			versionFlag = ""
		}
		version, err := resolveReleaseVersion(versionFlag)
		if err != nil {
			return err
		}

		destDir := filepath.Join(homeDir, downloader.DefaultDestSubdir())
		outputWorkflow(map[string]interface{}{
			"message": fmt.Sprintf("Downloading aether %s binary...", version),
		}, renderBinaryActionVisual("Binary Download", fmt.Sprintf("Downloading aether %s binary...", version), version, destDir))

		result, err := downloader.DownloadBinary(version, destDir)
		if err != nil {
			return fmt.Errorf("file sync succeeded but binary download failed: %w", err)
		}

		outputWorkflow(map[string]interface{}{
			"message": fmt.Sprintf("Binary installed to %s", result.Path),
			"path":    result.Path,
			"version": result.Version,
		}, renderBinaryActionVisual("Binary Ready", fmt.Sprintf("Binary installed to %s", result.Path), result.Version, result.Path))
	} else if downloadBinary && dryRun {
		outputWorkflow(map[string]interface{}{
			"message": "Would download binary from GitHub Releases",
		}, renderBinaryActionVisual("Binary Download", "Would download binary from GitHub Releases", "", ""))
	}

	return nil
}

// updateSyncResult holds the result of an update sync.
type updateSyncResult struct {
	copied  int
	skipped int
	details []map[string]interface{}
	errors  []string
}

// runUpdateSync syncs companion files from hub to local repo.
func runUpdateSync(hubDir, repoDir string, force bool) updateSyncResult {
	result := updateSyncResult{}

	hubSystem := filepath.Join(hubDir, "system")
	localAether := filepath.Join(repoDir, ".aether")

	// Directories to never overwrite or remove (user data)
	protectedDirs := map[string]bool{
		"data":   true,
		"dreams": true,
	}
	protectedFiles := map[string]bool{
		"QUEEN.md":           true,
		"CROWNED-ANTHILL.md": true,
	}

	for _, pair := range repoSyncPairs() {
		srcDir := filepath.Join(hubSystem, filepath.FromSlash(pair.hubRel))
		destDir := filepath.Join(localAether, filepath.FromSlash(pair.destRel))

		syncRes := syncDir(srcDir, destDir, syncOptions{
			cleanup:              force,
			preserveLocalChanges: !force && pair.preserveLocalChanges,
			protectedDirs:        protectedDirs,
			protectedFiles:       protectedFiles,
			validate:             pair.validate,
		})
		entry := map[string]interface{}{
			"label":   pair.label,
			"copied":  syncRes.copied,
			"skipped": syncRes.skipped,
			"removed": len(syncRes.removed),
		}
		if len(syncRes.errors) > 0 {
			entry["errors"] = syncRes.errors
			result.errors = append(result.errors, syncRes.errors...)
		}
		result.details = append(result.details, entry)
		result.copied += syncRes.copied
		result.skipped += syncRes.skipped
	}

	return result
}

// readHubVersion reads the version from the hub's version.json.
func readHubVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "unknown"
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return "unknown"
	}
	return v.Version
}
