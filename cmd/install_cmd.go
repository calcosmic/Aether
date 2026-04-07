package cmd

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/downloader"
	"github.com/spf13/cobra"
)

// installCmd implements "aether install" which copies commands, agents,
// and sets up the hub directory for global Aether access.
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install commands and agents to ~/.claude/ and set up distribution hub",
	Long: `Install Aether globally by copying slash commands and agent definitions
to their respective directories and setting up the distribution hub.

Copies:
  .claude/commands/ant/  -> ~/.claude/commands/ant/
  .claude/agents/ant/    -> ~/.claude/agents/ant/
  .opencode/commands/ant/ -> ~/.opencode/command/
  .opencode/agents/      -> ~/.opencode/agent/

Also creates the hub directory at ~/.aether/ for cross-repo coordination.`,
	Args: cobra.NoArgs,
	RunE: runInstall,
}

// installFlags holds the parsed flags for the install command.
var (
	installPackageDir     string
	installHomeDir        string
	installDownloadBinary bool
	installBinaryDest     string
	installBinaryVersion  string
)

func init() {
	installCmd.Flags().String("package-dir", "", "Path to the Aether package directory (contains .claude/, .opencode/, .aether/)")
	installCmd.Flags().String("home-dir", "", "User home directory (default: $HOME)")
	installCmd.Flags().Bool("download-binary", false, "Also download the Go binary from GitHub Releases")
	installCmd.Flags().String("binary-dest", "", "Destination directory for binary (default: ~/.aether/bin)")
	installCmd.Flags().String("binary-version", "", "Binary version to download (default: current version)")

	rootCmd.AddCommand(installCmd)
}

// runInstall executes the install logic.
func runInstall(cmd *cobra.Command, args []string) error {
	packageDir, err := cmd.Flags().GetString("package-dir")
	if err != nil {
		return fmt.Errorf("failed to read --package-dir: %w", err)
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

	// Resolve package directory (directory containing the aether binary or CWD)
	if packageDir == "" {
		// Try to find the package root by looking for .claude/commands/ant/
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot determine working directory: %w", err)
		}
		packageDir = wd
	}

	// Define sync pairs: source subpath -> destination subpath (relative to packageDir/homeDir)
	syncPairs := []struct {
		srcRel  string // relative to packageDir
		destRel string // relative to homeDir
		label   string // human-readable label
	}{
		{".claude/commands/ant", ".claude/commands/ant", "Commands (claude)"},
		{".claude/agents/ant", ".claude/agents/ant", "Agents (claude)"},
		{".opencode/commands/ant", ".opencode/command", "Commands (opencode)"},
		{".opencode/agents", ".opencode/agent", "Agents (opencode)"},
	}

	results := []map[string]interface{}{}

	for _, pair := range syncPairs {
		srcDir := filepath.Join(packageDir, filepath.FromSlash(pair.srcRel))
		destDir := filepath.Join(homeDir, filepath.FromSlash(pair.destRel))

		result := syncDirWithCleanup(srcDir, destDir)
		results = append(results, map[string]interface{}{
			"label":   pair.label,
			"src":     pair.srcRel,
			"dest":    pair.destRel,
			"copied":  result.copied,
			"skipped": result.skipped,
			"removed": len(result.removed),
		})
	}

	// Set up hub directory
	hubDir := filepath.Join(homeDir, ".aether")
	hubResult := setupInstallHub(hubDir, packageDir)
	results = append(results, hubResult)

	totalCopied := 0
	totalSkipped := 0
	for _, r := range results {
		if c, ok := r["copied"].(int); ok {
			totalCopied += c
		}
		if s, ok := r["skipped"].(int); ok {
			totalSkipped += s
		}
	}

	outputOK(map[string]interface{}{
		"message": fmt.Sprintf("Install complete: %d files copied, %d unchanged", totalCopied, totalSkipped),
		"details": results,
	})

	// Download Go binary if requested
	downloadBinary, _ := cmd.Flags().GetBool("download-binary")
	if downloadBinary {
		if err := runBinaryDownloadFromInstall(cmd, homeDir); err != nil {
			return fmt.Errorf("install succeeded but binary download failed: %w", err)
		}
	}

	return nil
}

// syncResult holds the outcome of a directory sync operation.
type syncResult struct {
	copied  int
	skipped int
	removed []string
}

// syncDirWithCleanup copies files from src to dest, skipping unchanged files
// (by SHA-256 hash), and removing stale files that no longer exist in src.
func syncDirWithCleanup(src, dest string) syncResult {
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
		if info, err := os.Stat(destPath); err == nil {
			srcHash, srcErr := fileSHA256(srcPath)
			destHash, destErr := fileSHA256(destPath)
			if srcErr == nil && destErr == nil && srcHash == destHash {
				result.skipped++
				continue
			}
			_ = info // used for size comparison in the future if needed
		}

		// Copy file
		if err := copyFile(srcPath, destPath); err != nil {
			continue
		}

		// Make .sh files executable
		if strings.HasSuffix(relPath, ".sh") {
			if err := os.Chmod(destPath, 0755); err != nil {
				log.Printf("install: failed to chmod %s: %v", destPath, err)
			}
		}

		result.copied++
	}

	// Remove stale files (in dest but not in src)
	destFiles := listFilesRecursive(dest)
	srcSet := make(map[string]struct{}, len(srcFiles))
	for _, f := range srcFiles {
		srcSet[f] = struct{}{}
	}

	for _, relPath := range destFiles {
		if _, exists := srcSet[relPath]; !exists {
			destPath := filepath.Join(dest, relPath)
			if err := os.Remove(destPath); err == nil {
				result.removed = append(result.removed, relPath)
			}
		}
	}

	// Clean empty directories
	if len(result.removed) > 0 {
		cleanEmptyDirs(dest)
	}

	return result
}

// listFilesRecursive returns all file paths relative to baseDir.
func listFilesRecursive(baseDir string) []string {
	var files []string
	_ = filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	return files
}

// fileSHA256 computes the SHA-256 hash of a file.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// copyFile copies a file from src to dest.
func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// cleanEmptyDirs removes empty directories under baseDir, bottom-up.
func cleanEmptyDirs(baseDir string) {
	_ = filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if path == baseDir {
			return nil
		}
		// Try to remove; will fail if not empty
		if err := os.Remove(path); err != nil {
			log.Printf("install: cleanEmptyDirs failed to remove %s: %v", path, err)
			}
		return nil
	})
}

// setupInstallHub creates the hub directory at ~/.aether/ if it doesn't exist.
func setupInstallHub(hubDir, packageDir string) map[string]interface{} {
	result := map[string]interface{}{
		"label": "Hub",
		"src":   ".aether/",
		"dest":  hubDir,
	}

	if err := os.MkdirAll(hubDir, 0755); err != nil {
		result["error"] = fmt.Sprintf("failed to create hub: %v", err)
		return result
	}

	// Create registry.json if it doesn't exist
	registryPath := filepath.Join(hubDir, "registry.json")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		content := `{"schema_version":1,"repos":[]}`
		if err := os.WriteFile(registryPath, []byte(content), 0644); err == nil {
			result["registry"] = "initialized"
		}
	} else {
		result["registry"] = "preserved"
	}

	// Write version.json
	versionPath := filepath.Join(hubDir, "version.json")
	versionContent := fmt.Sprintf(`{"version":"%s","updated_at":"now"}`, Version)
	if err := os.WriteFile(versionPath, []byte(versionContent), 0644); err != nil {
		result["version_error"] = fmt.Sprintf("failed to write version: %v", err)
	} else {
		result["version"] = Version
	}

	result["copied"] = 0
	result["skipped"] = 0
	result["removed"] = 0

	return result
}

// runBinaryDownloadFromInstall handles the --download-binary flag on the install command.
func runBinaryDownloadFromInstall(cmd *cobra.Command, homeDir string) error {
	version, _ := cmd.Flags().GetString("binary-version")
	if version == "" {
		version = Version
	}

	destDir, _ := cmd.Flags().GetString("binary-dest")
	if destDir == "" {
		destDir = filepath.Join(homeDir, downloader.DefaultDestSubdir())
	}

	outputOK(map[string]interface{}{
		"message": fmt.Sprintf("Downloading aether v%s binary...", version),
		"version": version,
		"dest":    destDir,
	})

	result, err := downloader.DownloadBinary(version, destDir)
	if err != nil {
		if downloader.IsVersionNotFoundErr(err) {
			return fmt.Errorf("version v%s not found: %w", version, err)
		}
		return err
	}

	outputOK(map[string]interface{}{
		"message": fmt.Sprintf("Binary installed to %s", result.Path),
		"path":    result.Path,
		"version": result.Version,
	})

	return nil
}
