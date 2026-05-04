package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// setupCmd implements "aether setup" which prepares repo-local Aether state
// while shared assets stay in the global hub/platform homes.
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up Aether in the current directory from hub",
	Long: `Set up Aether in the current directory from the selected distribution
hub (~/.aether/system/ for stable, ~/.aether-dev/system/ for dev).

Creates repo-local state directories (data/, dreams/, oracle/, checkpoints/, locks/),
a repo-local .aether/QUEEN.md, and a .gitignore.
Does NOT create COLONY_STATE.json (use "aether init" for that).
Existing local files are preserved (user data takes precedence). Shared agents,
commands, shipped skills, templates, docs, utils, workers, exchange files, and
references stay global instead of being copied into this repo.`,
	Args: cobra.NoArgs,
	RunE: runSetup,
}

var (
	setupRepoDir string
	setupHomeDir string
)

func init() {
	setupCmd.Flags().String("repo-dir", "", "Path to the repository (default: $CWD)")
	setupCmd.Flags().String("home-dir", "", "User home directory (default: $HOME)")
	setupCmd.Flags().String("channel", "", "Runtime channel to set up from (stable or dev; default: infer from binary/env)")

	rootCmd.AddCommand(setupCmd)
}

// runSetup executes the setup logic.
func runSetup(cmd *cobra.Command, args []string) error {
	channel := runtimeChannelFromFlag(cmd.Flags())

	repoDir, err := cmd.Flags().GetString("repo-dir")
	if err != nil {
		return fmt.Errorf("failed to read --repo-dir: %w", err)
	}
	homeDir, err := cmd.Flags().GetString("home-dir")
	if err != nil {
		return fmt.Errorf("failed to read --home-dir: %w", err)
	}

	// Resolve home directory
	if homeDir == "" {
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			homeDir = os.Getenv("USERPROFILE")
		}
		if homeDir == "" {
			return fmt.Errorf("cannot determine home directory: set HOME or use --home-dir")
		}
	}

	// Resolve repo directory
	if repoDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot determine working directory: %w", err)
		}
		repoDir = wd
	}

	// Check hub exists
	hubDir := resolveHubPathForHome(homeDir, channel)
	hubVersionFile := filepath.Join(hubDir, "version.json")
	if _, err := os.Stat(hubVersionFile); os.IsNotExist(err) {
		outputErrorMessage("Aether hub not installed. Run \"aether install\" first.")
		return nil
	}

	hubSystem := filepath.Join(hubDir, "system")
	localAether := filepath.Join(repoDir, ".aether")

	results := []map[string]interface{}{}
	totalCopied := 0
	totalSkipped := 0
	var syncErrors []string

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

		// Normalize destDir to be under repoDir (handle ../ correctly)
		absDestDir, err := filepath.Abs(destDir)
		if err != nil {
			continue
		}
		absRepoDir, err := filepath.Abs(repoDir)
		if err != nil {
			continue
		}

		// Skip if dest would escape the repo directory
		if !strings.HasPrefix(absDestDir, absRepoDir+string(filepath.Separator)) && absDestDir != absRepoDir {
			continue
		}

		result := syncDir(srcDir, destDir, syncOptions{
			cleanup:              pair.cleanup,
			preserveLocalChanges: true,
			protectedDirs:        protectedDirs,
			protectedFiles:       protectedFiles,
			validate:             pair.validate,
			include:              pair.include,
			mapRelPath:           pair.mapRelPath,
			cleanupInclude:       pair.cleanupInclude,
		})
		entry := map[string]interface{}{
			"label":   pair.label,
			"copied":  result.copied,
			"skipped": result.skipped,
		}
		if len(result.errors) > 0 {
			entry["errors"] = result.errors
			syncErrors = append(syncErrors, result.errors...)
		}
		results = append(results, entry)
		totalCopied += result.copied
		totalSkipped += result.skipped
	}

	for _, entry := range []struct {
		label string
		res   syncResult
	}{
		{"Local state scaffold", ensureRepoLocalScaffold(localAether)},
		{"Prune legacy repo platform assets", pruneLegacyRepoPlatformAssets(repoDir)},
		{"Prune shipped repo skills", pruneShippedRepoSkills(hubSystem, localAether, false)},
	} {
		result := map[string]interface{}{
			"label":   entry.label,
			"copied":  entry.res.copied,
			"skipped": entry.res.skipped,
			"removed": len(entry.res.removed),
		}
		if len(entry.res.errors) > 0 {
			result["errors"] = entry.res.errors
			syncErrors = append(syncErrors, entry.res.errors...)
		}
		results = append(results, result)
		totalCopied += entry.res.copied
		totalSkipped += entry.res.skipped
	}

	docResults, docCopied, docSkipped, docErrors := syncProjectDocs(hubSystem, repoDir)
	results = append(results, docResults...)
	totalCopied += docCopied
	totalSkipped += docSkipped
	if len(docErrors) > 0 {
		syncErrors = append(syncErrors, docErrors...)
	}

	if len(syncErrors) > 0 {
		outputError(2, fmt.Sprintf("setup failed with %d sync error(s)", len(syncErrors)), map[string]interface{}{"details": results})
		return nil
	}

	restartTargets := platformRestartTargets(results)
	message := fmt.Sprintf("Setup complete: %d files copied, %d unchanged", totalCopied, totalSkipped)
	if restartNote := platformRestartMessage(restartTargets); restartNote != "" {
		message += ". " + restartNote
	}
	result := map[string]interface{}{
		"message":                message,
		"details":                results,
		"restart_required":       len(restartTargets) > 0,
		"restart_targets":        restartTargets,
		"codex_restart_required": len(restartTargets) > 0,
		"codex_restart_targets":  restartTargets,
	}
	outputWorkflow(result, renderSetupVisual(repoDir, results, totalCopied, totalSkipped, restartTargets))

	return nil
}

// setupSyncDirProtected is the exported-internal variant of setupSyncDir
// used by tests. It delegates to setupSyncDir with protection parameters.
func setupSyncDirProtected(src, dest string, protectedDirs, protectedFiles map[string]bool) syncResult {
	return syncDir(src, dest, syncOptions{
		cleanup:              false,
		preserveLocalChanges: true,
		protectedDirs:        protectedDirs,
		protectedFiles:       protectedFiles,
	})
}
