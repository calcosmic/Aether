package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

Use --download-binary to also fetch the latest Go binary from GitHub Releases.`,
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
	updateCmd.Flags().Bool("download-binary", false, "Also download the latest binary from GitHub Releases")
	updateCmd.Flags().String("binary-version", "", "Binary version to download (default: latest)")
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
		outputOK(map[string]interface{}{
			"message":       fmt.Sprintf("Dry run — would sync companion files from hub [%s]", mode),
			"hub_version":   hubVersion,
			"local_version": resolveVersion(),
			"force":         force,
			"actions": []string{
				"Sync .aether/ system files (commands, agents, skills, templates, docs)",
				"Sync .claude/commands/ant/",
				"Sync .claude/agents/ant/",
				"Sync .codex/agents/",
				"Sync .opencode/commands/ant/",
				"Sync .opencode/agents/",
			},
		})
	} else {
		syncResult := runUpdateSync(hubDir, repoDir, force)
		outputOK(map[string]interface{}{
			"message":       fmt.Sprintf("Updated: %d files copied, %d unchanged", syncResult.copied, syncResult.skipped),
			"hub_version":   hubVersion,
			"local_version": resolveVersion(),
			"force":         force,
			"details":       syncResult.details,
		})
	}

	// Download binary if requested
	downloadBinary, _ := cmd.Flags().GetBool("download-binary")
	if downloadBinary && !dryRun {
		version, _ := cmd.Flags().GetString("binary-version")
		if version == "" {
			version = "latest"
		}

		destDir := filepath.Join(homeDir, downloader.DefaultDestSubdir())
		outputOK(map[string]interface{}{
			"message": fmt.Sprintf("Downloading aether %s binary...", version),
		})

		result, err := downloader.DownloadBinary(version, destDir)
		if err != nil {
			return fmt.Errorf("file sync succeeded but binary download failed: %w", err)
		}

		outputOK(map[string]interface{}{
			"message": fmt.Sprintf("Binary installed to %s", result.Path),
			"path":    result.Path,
			"version": result.Version,
		})
	} else if downloadBinary && dryRun {
		outputOK(map[string]interface{}{
			"message": "Would download binary from GitHub Releases",
		})
	}

	return nil
}

// updateSyncResult holds the result of an update sync.
type updateSyncResult struct {
	copied  int
	skipped int
	details []map[string]interface{}
}

// runUpdateSync syncs companion files from hub to local repo.
func runUpdateSync(hubDir, repoDir string, force bool) updateSyncResult {
	result := updateSyncResult{}

	hubSystem := filepath.Join(hubDir, "system")
	localAether := filepath.Join(repoDir, ".aether")

	// Directories to never overwrite or remove (user data)
	protectedDirs := map[string]bool{
		"data":    true,
		"dreams":  true,
	}
	protectedFiles := map[string]bool{
		"QUEEN.md":         true,
		"CROWNED-ANTHILL.md": true,
	}

	syncPairs := []struct {
		srcRel  string
		destRel string
		label   string
	}{
		{".", ".", "System files"},
		{"commands/claude", "../.claude/commands/ant", "Commands (claude)"},
		{"commands/opencode", "../.opencode/commands/ant", "Commands (opencode)"},
		{"agents", "../.opencode/agents", "Agents (opencode)"},
		{"agents-claude", "../.claude/agents/ant", "Agents (claude)"},
		{"codex", "../.codex/agents", "Agents (codex)"},
		{"rules", "../.claude/rules", "Rules (claude)"},
	}

	for _, pair := range syncPairs {
		srcDir := filepath.Join(hubSystem, filepath.FromSlash(pair.srcRel))
		destDir := filepath.Join(localAether, filepath.FromSlash(pair.destRel))

		var syncRes syncResult
		if force {
			// Force: overwrite changed files AND remove stale ones
			syncRes = syncDirProtected(srcDir, destDir, protectedDirs, protectedFiles)
		} else {
			// Normal: overwrite changed files but don't remove stale ones
			syncRes = syncDirUpdate(srcDir, destDir, protectedDirs, protectedFiles)
		}
		result.details = append(result.details, map[string]interface{}{
			"label":   pair.label,
			"copied":  syncRes.copied,
			"skipped": syncRes.skipped,
		})
		result.copied += syncRes.copied
		result.skipped += syncRes.skipped
	}

	return result
}

// syncDirProtected copies files from src to dest like syncDirWithCleanup,
// but skips protected directories and files.
// syncDirUpdate copies files from src to dest, overwriting changed files (by SHA-256)
// and respecting protected directories/files. Unlike syncDirProtected, it does NOT
// remove stale files. Used by normal (non-force) updates so companion files (rules,
// commands, agents) always get updated from the hub while user data stays safe.
func syncDirUpdate(src, dest string, protectedDirs, protectedFiles map[string]bool) syncResult {
	result := syncResult{}

	srcInfo, err := os.Stat(src)
	if err != nil || !srcInfo.IsDir() {
		return result
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return result
	}

	srcFiles := listFilesRecursive(src)
	for _, relPath := range srcFiles {
		// Skip protected paths
		firstComponent := relPath
		if idx := strings.Index(relPath, string(filepath.Separator)); idx >= 0 {
			firstComponent = relPath[:idx]
		}
		if protectedDirs[firstComponent] {
			result.skipped++
			continue
		}
		if protectedFiles[filepath.Base(relPath)] {
			result.skipped++
			continue
		}

		srcPath := filepath.Join(src, relPath)
		destPath := filepath.Join(dest, relPath)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			continue
		}

		// Skip if identical (SHA-256 match)
		if _, err := os.Stat(destPath); err == nil {
			srcHash, srcErr := fileSHA256(srcPath)
			destHash, destErr := fileSHA256(destPath)
			if srcErr == nil && destErr == nil && srcHash == destHash {
				result.skipped++
				continue
			}
		}

		if err := copyFile(srcPath, destPath); err != nil {
			continue
		}

		if strings.HasSuffix(relPath, ".sh") {
			os.Chmod(destPath, 0755)
		}

		result.copied++
	}

	return result
}

func syncDirProtected(src, dest string, protectedDirs, protectedFiles map[string]bool) syncResult {
	result := syncResult{}

	srcInfo, err := os.Stat(src)
	if err != nil || !srcInfo.IsDir() {
		return result
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return result
	}

	srcFiles := listFilesRecursive(src)
	for _, relPath := range srcFiles {
		// Skip protected paths
		firstComponent := relPath
		if idx := strings.Index(relPath, string(filepath.Separator)); idx >= 0 {
			firstComponent = relPath[:idx]
		}
		if protectedDirs[firstComponent] {
			result.skipped++
			continue
		}
		if protectedFiles[filepath.Base(relPath)] {
			result.skipped++
			continue
		}

		srcPath := filepath.Join(src, relPath)
		destPath := filepath.Join(dest, relPath)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			continue
		}

		// Copy if new or changed (SHA-256 diff)
		if info, err := os.Stat(destPath); err == nil {
			srcHash, srcErr := fileSHA256(srcPath)
			destHash, destErr := fileSHA256(destPath)
			if srcErr == nil && destErr == nil && srcHash == destHash {
				result.skipped++
				continue
			}
			_ = info
		}

		if err := copyFile(srcPath, destPath); err != nil {
			continue
		}

		if strings.HasSuffix(relPath, ".sh") {
			os.Chmod(destPath, 0755)
		}

		result.copied++
	}

	// Remove stale files (in dest but not in src), respecting protections
	destFiles := listFilesRecursive(dest)
	srcSet := make(map[string]struct{}, len(srcFiles))
	for _, f := range srcFiles {
		srcSet[f] = struct{}{}
	}

	for _, relPath := range destFiles {
		firstComponent := relPath
		if idx := strings.Index(relPath, string(filepath.Separator)); idx >= 0 {
			firstComponent = relPath[:idx]
		}
		if protectedDirs[firstComponent] {
			continue
		}
		if protectedFiles[filepath.Base(relPath)] {
			continue
		}

		if _, exists := srcSet[relPath]; !exists {
			destPath := filepath.Join(dest, relPath)
			if err := os.Remove(destPath); err == nil {
				result.removed = append(result.removed, relPath)
			}
		}
	}

	if len(result.removed) > 0 {
		cleanEmptyDirs(dest)
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
