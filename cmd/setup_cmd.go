package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// setupCmd implements "aether setup" which copies hub system files from
// ~/.aether/system/ to the local .aether/ directory.
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up Aether in the current directory from hub",
	Long: `Set up Aether in the current directory by copying system files
from the distribution hub (~/.aether/system/) to the local .aether/ directory.

Creates required directories (data/, checkpoints/, locks/) and a .gitignore.
Does NOT create COLONY_STATE.json (use "aether init" for that).
Existing local files are preserved (user data takes precedence).`,
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

	rootCmd.AddCommand(setupCmd)
}

// runSetup executes the setup logic.
func runSetup(cmd *cobra.Command, args []string) error {
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
	hubDir := filepath.Join(homeDir, ".aether")
	hubVersionFile := filepath.Join(hubDir, "version.json")
	if _, err := os.Stat(hubVersionFile); os.IsNotExist(err) {
		outputErrorMessage("Aether hub not installed. Run \"aether install\" first.")
		return nil
	}

	// Define sync pairs: hub source -> local destination
	hubSystem := filepath.Join(hubDir, "system")
	localAether := filepath.Join(repoDir, ".aether")

	syncPairs := []struct {
		srcRel  string // relative to hubSystem
		destRel string // relative to localAether
		label   string
	}{
		{".", ".", "System files"},
		{"commands/claude", "../.claude/commands/ant", "Commands (claude)"},
		{"commands/opencode", "../.opencode/commands/ant", "Commands (opencode)"},
		{"agents", "../.opencode/agents", "Agents (opencode)"},
		{"agents-claude", "../.claude/agents/ant", "Agents (claude)"},
	}

	results := []map[string]interface{}{}
	totalCopied := 0
	totalSkipped := 0

	for _, pair := range syncPairs {
		srcDir := filepath.Join(hubSystem, filepath.FromSlash(pair.srcRel))
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

		result := setupSyncDir(srcDir, destDir)
		results = append(results, map[string]interface{}{
			"label":   pair.label,
			"copied":  result.copied,
			"skipped": result.skipped,
		})
		totalCopied += result.copied
		totalSkipped += result.skipped
	}

	// Create required directories
	for _, dir := range []string{"data", "checkpoints", "locks"} {
		if err := os.MkdirAll(filepath.Join(localAether, dir), 0755); err != nil {
			// Non-fatal
			results = append(results, map[string]interface{}{
				"label": fmt.Sprintf("Directory %s", dir),
				"error": err.Error(),
			})
		}
	}

	// Create .gitignore if it doesn't exist
	gitignorePath := filepath.Join(localAether, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		gitignoreContent := "# Aether local state - not versioned\ndata/\ncheckpoints/\nlocks/\n"
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err == nil {
			results = append(results, map[string]interface{}{
				"label":  ".gitignore",
				"copied": 1,
			})
			totalCopied++
		}
	}

	outputOK(map[string]interface{}{
		"message": fmt.Sprintf("Setup complete: %d files copied, %d unchanged", totalCopied, totalSkipped),
		"details": results,
	})

	return nil
}

// setupSyncDir copies files from src to dest, skipping identical files
// (by SHA-256 hash). Unlike syncDirWithCleanup (used by install), this
// does NOT remove stale files -- local files take precedence.
func setupSyncDir(src, dest string) syncResult {
	result := syncResult{}

	// Check source exists
	srcInfo, err := os.Stat(src)
	if err != nil || !srcInfo.IsDir() {
		return result
	}

	// Create destination
	if err := os.MkdirAll(dest, 0755); err != nil {
		return result
	}

	// Walk source and copy files
	srcFiles := listFilesRecursive(src)
	for _, relPath := range srcFiles {
		srcPath := filepath.Join(src, relPath)
		destPath := filepath.Join(dest, relPath)

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			continue
		}

		// Check if file is unchanged
		if _, err := os.Stat(destPath); err == nil {
			srcHash, srcErr := fileSHA256(srcPath)
			destHash, destErr := fileSHA256(destPath)
			if srcErr == nil && destErr == nil && srcHash == destHash {
				result.skipped++
				continue
			}
			// File exists but is different -- skip, local takes precedence
			result.skipped++
			continue
		}

		// Copy file
		if err := copyFile(srcPath, destPath); err != nil {
			continue
		}

		// Make .sh files executable
		if strings.HasSuffix(relPath, ".sh") {
			if err := os.Chmod(destPath, 0755); err != nil {
				log.Printf("setup: failed to chmod %s: %v", destPath, err)
			}
		}

		result.copied++
	}

	return result
}
