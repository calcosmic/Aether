package cmd

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	aetherassets "github.com/calcosmic/Aether"
	"github.com/calcosmic/Aether/pkg/downloader"
	"github.com/spf13/cobra"
)

// installCmd implements "aether install" which copies commands, agents,
// and sets up the hub directory for global Aether access.
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install platform assets and refresh the shared Aether hub",
	Long: "Install Aether globally by copying platform assets to their respective\n" +
		"directories and setting up the distribution hub. By default, Aether installs\n" +
		"the companion files embedded in the Go binary. Use --package-dir only when\n" +
		"developing from a local source checkout.\n\n" +
		"When install runs from an Aether source checkout, it also rebuilds the shared\n" +
		"`aether` binary unless `--skip-build-binary` is used. Other repos on the same\n" +
		"machine already share that binary; `aether update` there only syncs companion\n" +
		"files unless `--download-binary` is explicitly requested.\n\n" +
		"Copies:\n" +
		"  .claude/commands/ant/  -> ~/.claude/commands/ant/\n" +
		"  .claude/agents/ant/    -> ~/.claude/agents/ant/\n" +
		"  .opencode/commands/ant/ -> ~/.opencode/command/\n" +
		"  .opencode/agents/      -> ~/.opencode/agent/\n" +
		"  .codex/agents/         -> ~/.codex/agents/\n" +
		"  .aether/skills-codex/  -> ~/.codex/skills/aether/\n\n" +
		"Also creates the selected hub directory (~/.aether/ for stable, ~/.aether-dev/ for dev) for cross-repo coordination.",
	Args: cobra.NoArgs,
	RunE: runInstall,
}

// installFlags holds the parsed flags for the install command.
var (
	installPackageDir      string
	installHomeDir         string
	installDownloadBinary  bool
	installBinaryDest      string
	installBinaryVersion   string
	installSkipBuildBinary bool
	installChannel         string
)

func init() {
	installCmd.Flags().String("package-dir", "", "Override the embedded install assets with a local Aether checkout or package directory")
	installCmd.Flags().String("home-dir", "", "User home directory (default: $HOME)")
	installCmd.Flags().String("channel", "", "Runtime channel to install (stable or dev; default: infer from binary/env)")
	installCmd.Flags().Bool("download-binary", false, "Also download the Go binary from GitHub Releases")
	installCmd.Flags().String("binary-dest", "", "Destination directory for binary (default: channel-specific hub bin, or current/local bin when rebuilding from source)")
	installCmd.Flags().String("binary-version", "", "Binary version to download (default: current version)")
	installCmd.Flags().Bool("skip-build-binary", false, "Skip auto-building the Go binary when installing from an Aether source checkout")

	rootCmd.AddCommand(installCmd)
}

// runInstall executes the install logic.
func runInstall(cmd *cobra.Command, args []string) error {
	channel := runtimeChannelFromFlag(cmd.Flags())

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

	resolvedPackageDir, cleanupPackageDir, err := resolveInstallPackageDir(packageDir)
	if err != nil {
		return err
	}
	if cleanupPackageDir != nil {
		defer cleanupPackageDir()
	}
	packageDir = resolvedPackageDir

	results := []map[string]interface{}{}
	var syncErrors []string

	if shouldSyncPlatformHomes(channel) {
		for _, pair := range installSyncPairs() {
			srcDir := filepath.Join(packageDir, filepath.FromSlash(pair.srcRel))
			destDir := filepath.Join(homeDir, filepath.FromSlash(pair.destRel))

			result := syncDir(srcDir, destDir, syncOptions{
				cleanup:              pair.cleanup,
				preserveLocalChanges: pair.preserveLocalChanges,
				validate:             pair.validate,
				include:              pair.include,
			})
			entry := map[string]interface{}{
				"label":   pair.label,
				"src":     pair.srcRel,
				"dest":    pair.destRel,
				"copied":  result.copied,
				"skipped": result.skipped,
				"removed": len(result.removed),
			}
			if len(result.errors) > 0 {
				entry["errors"] = result.errors
				syncErrors = append(syncErrors, result.errors...)
			}
			results = append(results, entry)
		}
	} else {
		results = append(results, map[string]interface{}{
			"label":   "Platform homes",
			"src":     ".claude/.opencode/.codex",
			"dest":    "skipped",
			"copied":  0,
			"skipped": 0,
			"note":    "Dev channel leaves global Claude/OpenCode/Codex home assets untouched by default.",
		})
	}

	// Set up hub directory
	hubDir := resolveHubPathForHome(homeDir, channel)
	hubResult := setupInstallHub(hubDir, packageDir)
	results = append(results, hubResult)
	if errVal, ok := hubResult["error"].(string); ok && errVal != "" {
		syncErrors = append(syncErrors, errVal)
	}

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

	if len(syncErrors) > 0 {
		outputError(2, fmt.Sprintf("install failed with %d sync error(s)", len(syncErrors)), map[string]interface{}{"details": results})
		return nil
	}

	result := map[string]interface{}{
		"message":             fmt.Sprintf("Install complete: %d files copied, %d unchanged", totalCopied, totalSkipped),
		"details":             results,
		"channel":             string(channel),
		"binary_refresh_mode": installBinaryRefreshMode(cmd, packageDir),
		"binary_refresh_note": installBinaryRefreshNote(installBinaryRefreshMode(cmd, packageDir), channel),
	}
	outputWorkflow(result, renderInstallVisual(homeDir, results, totalCopied, totalSkipped, installBinaryRefreshMode(cmd, packageDir)))

	// In a source checkout, install should keep the local binary in sync with
	// the companion files it just published to the hub. Otherwise fast-moving
	// command files can call subcommands that the installed binary does not have.
	downloadBinary, _ := cmd.Flags().GetBool("download-binary")
	skipBuildBinary, _ := cmd.Flags().GetBool("skip-build-binary")
	if !downloadBinary && !skipBuildBinary && isAetherSourceCheckout(packageDir) {
		if err := runLocalBinaryBuildFromInstall(cmd, homeDir, packageDir, channel); err != nil {
			return fmt.Errorf("install succeeded but local binary build failed: %w", err)
		}
	}

	// Download Go binary if requested. This remains opt-in because release
	// downloads require network access and should not replace local source builds.
	if downloadBinary {
		if err := runBinaryDownloadFromInstall(cmd, homeDir, channel); err != nil {
			return fmt.Errorf("install succeeded but binary download failed: %w", err)
		}
	}

	return nil
}

func installBinaryRefreshMode(cmd *cobra.Command, packageDir string) string {
	downloadBinary, _ := cmd.Flags().GetBool("download-binary")
	if downloadBinary {
		return "release-download"
	}
	skipBuildBinary, _ := cmd.Flags().GetBool("skip-build-binary")
	if !skipBuildBinary && isAetherSourceCheckout(packageDir) {
		return "local-build"
	}
	return "unchanged"
}

func installBinaryRefreshNote(mode string, channel runtimeChannel) string {
	binaryLabel := defaultBinaryName(channel)
	switch strings.TrimSpace(mode) {
	case "release-download":
		return fmt.Sprintf("The hub was refreshed; a published %s release binary will be downloaded next.", binaryLabel)
	case "local-build":
		return fmt.Sprintf("The hub was refreshed from a source checkout; the shared local %s binary will be rebuilt next.", binaryLabel)
	default:
		return fmt.Sprintf("The file-sync step refreshed the hub only. Shared %s runtime changes require either a local rebuild from source or an explicit release download.", binaryLabel)
	}
}

func resolveInstallPackageDir(explicit string) (string, func(), error) {
	if explicit != "" {
		resolved := normalizeInstallPackageDir(explicit)
		if resolved == "" {
			return "", nil, fmt.Errorf("--package-dir %q does not contain Aether companion files", explicit)
		}
		return resolved, nil, nil
	}

	if wd, err := os.Getwd(); err == nil {
		if resolved := normalizeInstallPackageDir(wd); resolved != "" {
			return resolved, nil, nil
		}
	}

	tempDir, err := os.MkdirTemp("", "aether-install-assets-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp install package: %w", err)
	}
	if err := aetherassets.MaterializeInstallPackage(tempDir); err != nil {
		os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("materialize embedded install assets: %w", err)
	}
	return tempDir, func() { _ = os.RemoveAll(tempDir) }, nil
}

func normalizeInstallPackageDir(candidate string) string {
	if candidate == "" {
		return ""
	}
	abs, err := filepath.Abs(candidate)
	if err == nil {
		candidate = abs
	}
	if isInstallPackageDir(candidate) {
		return candidate
	}
	if root := findAetherModuleRoot(candidate); root != "" && isInstallPackageDir(root) {
		return root
	}
	return ""
}

func isInstallPackageDir(dir string) bool {
	if dir == "" {
		return false
	}
	knownPaths := []string{
		filepath.Join(dir, ".aether", "workers.md"),
		filepath.Join(dir, ".codex", "agents"),
		filepath.Join(dir, ".claude", "commands", "ant"),
		filepath.Join(dir, ".claude", "agents", "ant"),
		filepath.Join(dir, ".opencode", "commands", "ant"),
		filepath.Join(dir, ".opencode", "agents"),
		filepath.Join(dir, ".aether", "skills"),
		filepath.Join(dir, ".aether", "docs"),
	}
	for _, path := range knownPaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// syncResult holds the outcome of a directory sync operation.
type syncResult struct {
	copied  int
	skipped int
	removed []string
	errors  []string
}

type syncOptions struct {
	cleanup              bool
	preserveLocalChanges bool
	protectedDirs        map[string]bool
	protectedFiles       map[string]bool
	validate             syncValidator
	include              syncFilter
}

// syncDir copies files from src to dest, optionally preserving changed local
// files, validating source files, and removing stale files.
func syncDir(src, dest string, opts syncOptions) syncResult {
	result := syncResult{}

	// Check source exists
	srcInfo, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", src, err))
		return result
	}
	if !srcInfo.IsDir() {
		result.errors = append(result.errors, fmt.Sprintf("%s is not a directory", src))
		return result
	}

	// Create destination
	if err := os.MkdirAll(dest, 0755); err != nil {
		result.errors = append(result.errors, fmt.Sprintf("mkdir %s: %v", dest, err))
		return result
	}

	// Walk source and copy files
	srcFiles := listFilesRecursive(src)
	if opts.include != nil {
		srcFiles = filterSyncFiles(srcFiles, opts.include)
	}
	for _, relPath := range srcFiles {
		if syncPathHasComponent(relPath, "node_modules") {
			result.skipped++
			continue
		}
		if syncPathProtected(relPath, opts.protectedDirs, opts.protectedFiles) {
			result.skipped++
			continue
		}
		srcPath := filepath.Join(src, relPath)
		destPath := filepath.Join(dest, relPath)

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("mkdir %s: %v", filepath.Dir(destPath), err))
			continue
		}

		srcData, err := os.ReadFile(srcPath)
		if err != nil {
			result.errors = append(result.errors, fmt.Sprintf("read %s: %v", srcPath, err))
			continue
		}
		if opts.validate != nil {
			if err := opts.validate(srcPath, relPath, srcData); err != nil {
				result.errors = append(result.errors, err.Error())
				continue
			}
		}

		// Check if file is unchanged or locally modified
		if _, err := os.Stat(destPath); err == nil {
			srcHash, srcErr := fileSHA256(srcPath)
			destHash, destErr := fileSHA256(destPath)
			if srcErr == nil && destErr == nil && srcHash == destHash {
				result.skipped++
				continue
			}
			if opts.preserveLocalChanges {
				result.skipped++
				continue
			}
		}

		// Copy file
		if err := copyFile(srcPath, destPath); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("copy %s -> %s: %v", srcPath, destPath, err))
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

	if opts.cleanup {
		// Remove stale files (in dest but not in src)
		destFiles := listFilesRecursive(dest)
		srcSet := make(map[string]struct{}, len(srcFiles))
		for _, f := range srcFiles {
			srcSet[f] = struct{}{}
		}

		for _, relPath := range destFiles {
			if syncPathHasComponent(relPath, "node_modules") {
				continue
			}
			if syncPathProtected(relPath, opts.protectedDirs, opts.protectedFiles) {
				continue
			}
			if opts.include != nil && !opts.include(relPath) {
				continue
			}
			if _, exists := srcSet[relPath]; !exists {
				destPath := filepath.Join(dest, relPath)
				if err := os.Remove(destPath); err == nil {
					result.removed = append(result.removed, relPath)
				} else {
					result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", destPath, err))
				}
			}
		}

		// Clean empty directories
		if len(result.removed) > 0 {
			cleanEmptyDirs(dest)
		}
	}

	return result
}

func filterSyncFiles(relPaths []string, include syncFilter) []string {
	if include == nil {
		return relPaths
	}

	filtered := make([]string, 0, len(relPaths))
	for _, relPath := range relPaths {
		if include(relPath) {
			filtered = append(filtered, relPath)
		}
	}

	return filtered
}

func syncPathProtected(relPath string, protectedDirs, protectedFiles map[string]bool) bool {
	if len(protectedDirs) == 0 && len(protectedFiles) == 0 {
		return false
	}
	firstComponent := relPath
	if idx := strings.Index(relPath, string(filepath.Separator)); idx >= 0 {
		firstComponent = relPath[:idx]
	}
	if protectedDirs[firstComponent] {
		return true
	}
	return protectedFiles[filepath.Base(relPath)]
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
	var dirs []string
	_ = filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == baseDir {
			return nil
		}
		dirs = append(dirs, path)
		return nil
	})

	sort.Slice(dirs, func(i, j int) bool {
		return strings.Count(dirs[i], string(filepath.Separator)) > strings.Count(dirs[j], string(filepath.Separator))
	})

	for _, dir := range dirs {
		if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
			if errors.Is(err, os.ErrExist) || strings.Contains(strings.ToLower(err.Error()), "directory not empty") {
				continue
			}
			if entries, readErr := os.ReadDir(dir); readErr == nil && len(entries) > 0 {
				continue
			}
			log.Printf("install: cleanEmptyDirs failed to remove %s: %v", dir, err)
		}
	}
}

// hubExcludeDirs are .aether/ subdirectories that should never be synced to the hub.
// These are private/local paths that belong to individual colonies and should
// never be published into the shared hub.
var hubExcludeDirs = map[string]bool{
	"data":         true,
	"dreams":       true,
	"oracle":       true,
	"checkpoints":  true,
	"locks":        true,
	"temp":         true,
	"archive":      true,
	"chambers":     true,
	"agents":       true, // agents/ is opencode-only, agents-claude/ is the packaging mirror
	"examples":     true,
	"node_modules": true,
	"__pycache__":  true,
}

// setupInstallHub creates the hub directory at ~/.aether/ and syncs companion files
// from .aether/ to ~/.aether/system/.
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

	// Sync companion files from .aether/ to ~/.aether/system/
	systemDir := filepath.Join(hubDir, "system")
	srcAether := filepath.Join(packageDir, ".aether")
	hubSyncResult := syncDirToHub(srcAether, systemDir)
	result["copied"] = hubSyncResult.copied
	result["skipped"] = hubSyncResult.skipped
	result["removed"] = len(hubSyncResult.removed)
	if len(hubSyncResult.errors) > 0 {
		result["errors"] = hubSyncResult.errors
	}

	// Sync Codex agents to hub system directory
	// Sync only .codex/agents/ to avoid nesting: syncing .codex/ preserves agents/ subdir
	// in hub as system/codex/agents/*.toml, then setup maps system/codex/ -> .codex/agents/,
	// landing at .codex/agents/agents/*.toml. Syncing just agents/ fixes this.
	codexSrc := filepath.Join(packageDir, ".codex", "agents")
	codexDest := filepath.Join(systemDir, "codex")
	codexSyncResult := syncDirToHubWithExclusion(codexSrc, codexDest, nil, validateCodexAgentFile, isShippedAetherCodexAgent)
	result["codex_copied"] = codexSyncResult.copied
	result["codex_skipped"] = codexSyncResult.skipped
	if len(codexSyncResult.errors) > 0 {
		existing, _ := result["errors"].([]string)
		result["errors"] = append(existing, codexSyncResult.errors...)
	}

	// Sync wrapper commands and OpenCode agents into the hub layout that
	// `aether update` reads from. The `.aether/commands/*.yaml` specs remain in
	// system/commands/, while the generated wrapper surfaces live in subdirs.
	for _, pair := range []struct {
		srcDir  string
		destDir string
		include syncFilter
	}{
		{
			srcDir:  filepath.Join(packageDir, ".claude", "commands", "ant"),
			destDir: filepath.Join(systemDir, "commands", "claude"),
		},
		{
			srcDir:  filepath.Join(packageDir, ".claude"),
			destDir: filepath.Join(systemDir, "settings", "claude"),
			include: isClaudeSettingsFile,
		},
		{
			srcDir:  filepath.Join(packageDir, ".opencode", "commands", "ant"),
			destDir: filepath.Join(systemDir, "commands", "opencode"),
		},
		{
			srcDir:  filepath.Join(packageDir, ".opencode", "agents"),
			destDir: filepath.Join(systemDir, "agents"),
		},
	} {
		syncRes := syncDirToHubWithExclusion(pair.srcDir, pair.destDir, nil, nil, pair.include)
		hubSyncResult.copied += syncRes.copied
		hubSyncResult.skipped += syncRes.skipped
		hubSyncResult.removed = append(hubSyncResult.removed, syncRes.removed...)
		if len(syncRes.errors) > 0 {
			existing, _ := result["errors"].([]string)
			result["errors"] = append(existing, syncRes.errors...)
		}
	}

	result["copied"] = hubSyncResult.copied
	result["skipped"] = hubSyncResult.skipped
	result["removed"] = len(hubSyncResult.removed)

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

	// Write version.json using git tags or ldflags (not the hardcoded default)
	versionPath := filepath.Join(hubDir, "version.json")
	resolved := resolveVersion(packageDir)
	versionContent := fmt.Sprintf(`{"version":"%s","updated_at":"now"}`, resolved)
	if err := os.WriteFile(versionPath, []byte(versionContent), 0644); err != nil {
		result["version_error"] = fmt.Sprintf("failed to write version: %v", err)
	} else {
		result["version"] = resolved
	}

	return result
}

// syncDirToHub copies files from src (.aether/) to dest (~/.aether/system/),
// skipping excluded directories and unchanged files (by SHA-256 hash).
// Also removes stale files in dest that no longer exist in src.
func syncDirToHub(src, dest string) syncResult {
	return syncDirToHubWithExclusion(src, dest, hubExcludeDirs, nil, nil)
}

// syncDirToHubWithExclusion is like syncDirToHub but accepts a custom exclusion map.
// Pass nil to exclude nothing.
func syncDirToHubWithExclusion(src, dest string, exclude map[string]bool, validate syncValidator, include syncFilter) syncResult {
	// Default to no exclusions if nil
	if exclude == nil {
		exclude = map[string]bool{}
	}
	result := syncResult{}

	srcInfo, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", src, err))
		return result
	}
	if !srcInfo.IsDir() {
		result.errors = append(result.errors, fmt.Sprintf("%s is not a directory", src))
		return result
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		result.errors = append(result.errors, fmt.Sprintf("mkdir %s: %v", dest, err))
		return result
	}

	// Walk source and copy files, skipping excluded directories
	srcFiles := listFilesRecursiveWithExclusion(src, exclude)
	if include != nil {
		srcFiles = filterSyncFiles(srcFiles, include)
	}
	for _, relPath := range srcFiles {
		if syncPathHasComponent(relPath, "node_modules") {
			result.skipped++
			continue
		}
		srcPath := filepath.Join(src, relPath)
		destPath := filepath.Join(dest, relPath)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("mkdir %s: %v", filepath.Dir(destPath), err))
			continue
		}

		srcData, err := os.ReadFile(srcPath)
		if err != nil {
			result.errors = append(result.errors, fmt.Sprintf("read %s: %v", srcPath, err))
			continue
		}
		if validate != nil {
			if err := validate(srcPath, relPath, srcData); err != nil {
				result.errors = append(result.errors, err.Error())
				continue
			}
		}

		// Check if file is unchanged
		if _, err := os.Stat(destPath); err == nil {
			srcHash, srcErr := fileSHA256(srcPath)
			destHash, destErr := fileSHA256(destPath)
			if srcErr == nil && destErr == nil && srcHash == destHash {
				result.skipped++
				continue
			}
		}

		if err := copyFile(srcPath, destPath); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("copy %s -> %s: %v", srcPath, destPath, err))
			continue
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
		// Don't remove files in excluded dirs (they may have been added manually)
		if pathHasExcludedComponent(relPath, exclude) || syncPathHasComponent(relPath, "node_modules") {
			continue
		}
		if include != nil && !include(relPath) {
			continue
		}
		if _, exists := srcSet[relPath]; !exists {
			destPath := filepath.Join(dest, relPath)
			if err := os.Remove(destPath); err == nil {
				result.removed = append(result.removed, relPath)
			} else {
				result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", destPath, err))
			}
		}
	}

	if len(result.removed) > 0 {
		cleanEmptyDirs(dest)
	}

	return result
}

// listFilesRecursiveWithExclusion returns all file paths relative to baseDir,
// skipping directories listed in the exclude map.
func listFilesRecursiveWithExclusion(baseDir string, exclude map[string]bool) []string {
	var files []string
	_ = filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// Skip excluded directories
			rel, relErr := filepath.Rel(baseDir, path)
			if relErr == nil && pathHasExcludedComponent(rel, exclude) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(baseDir, path)
		if relErr != nil {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	return files
}

func pathHasExcludedComponent(relPath string, exclude map[string]bool) bool {
	if len(exclude) == 0 {
		return false
	}
	relPath = filepath.Clean(relPath)
	if relPath == "." || relPath == "" {
		return false
	}
	for _, part := range strings.Split(relPath, string(filepath.Separator)) {
		if exclude[part] {
			return true
		}
	}
	return false
}

func syncPathHasComponent(relPath, component string) bool {
	component = strings.TrimSpace(component)
	if component == "" {
		return false
	}
	relPath = filepath.Clean(relPath)
	if relPath == "." || relPath == "" {
		return false
	}
	for _, part := range strings.Split(relPath, string(filepath.Separator)) {
		if part == component {
			return true
		}
	}
	return false
}

// runBinaryDownloadFromInstall handles the --download-binary flag on the install command.
func runBinaryDownloadFromInstall(cmd *cobra.Command, homeDir string, channel runtimeChannel) error {
	versionFlag, _ := cmd.Flags().GetString("binary-version")
	version, err := resolveReleaseVersion(versionFlag)
	if err != nil {
		return err
	}

	destDir, _ := cmd.Flags().GetString("binary-dest")
	if destDir == "" {
		destDir = filepath.Join(homeDir, defaultBinaryDestSubdirForChannel(channel))
	}

	outputWorkflow(map[string]interface{}{
		"message": fmt.Sprintf("Downloading %s v%s binary...", defaultBinaryName(channel), version),
		"version": version,
		"dest":    destDir,
	}, renderBinaryActionVisual("Binary Download", fmt.Sprintf("Downloading %s v%s binary...", defaultBinaryName(channel), version), version, destDir))

	result, err := downloader.DownloadBinary(version, destDir)
	if err != nil {
		if downloader.IsVersionNotFoundErr(err) {
			return fmt.Errorf("version v%s not found: %w", version, err)
		}
		return err
	}
	result, err = alignDownloadedBinaryToChannel(result, destDir, channel)
	if err != nil {
		return fmt.Errorf("rename downloaded binary for %s channel: %w", channel, err)
	}

	outputWorkflow(map[string]interface{}{
		"message": fmt.Sprintf("Binary installed to %s", result.Path),
		"path":    result.Path,
		"version": result.Version,
	}, renderBinaryActionVisual("Binary Ready", fmt.Sprintf("Binary installed to %s", result.Path), result.Version, result.Path))

	return nil
}

func isAetherSourceCheckout(packageDir string) bool {
	root := findAetherModuleRoot(packageDir)
	if root == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(root, "cmd", "aether", "main.go")); err != nil {
		return false
	}
	return true
}

func runLocalBinaryBuildFromInstall(cmd *cobra.Command, homeDir, packageDir string, channel runtimeChannel) error {
	sourceRoot := findAetherModuleRoot(packageDir)
	if sourceRoot == "" {
		return fmt.Errorf("cannot locate Aether go.mod from %s", packageDir)
	}

	destDir, _ := cmd.Flags().GetString("binary-dest")
	if destDir == "" {
		destDir = defaultLocalBinaryDest(homeDir, channel)
	}

	version := resolveVersion(sourceRoot)
	if version == "" || version == "0.0.0-dev" {
		version = "0.0.0-dev"
	}

	result, err := buildLocalBinary(sourceRoot, destDir, version, channel)
	if err != nil {
		return err
	}

	outputWorkflow(map[string]interface{}{
		"message": fmt.Sprintf("Built local aether binary to %s", result.Path),
		"path":    result.Path,
		"version": result.Version,
	}, renderBinaryActionVisual("Binary Build", fmt.Sprintf("Built local %s binary to %s", defaultBinaryName(channel), result.Path), result.Version, result.Path))
	return nil
}

func defaultLocalBinaryDest(homeDir string, channel runtimeChannel) string {
	if exe, err := os.Executable(); err == nil {
		base := filepath.Base(exe)
		if base == defaultBinaryName(channel) || base == defaultBinaryName(channel)+".exe" {
			return filepath.Dir(exe)
		}
	}
	localBin := filepath.Join(homeDir, ".local", "bin")
	if info, err := os.Stat(localBin); err == nil && info.IsDir() {
		return localBin
	}
	return filepath.Join(homeDir, defaultBinaryDestSubdirForChannel(channel))
}

func buildLocalBinary(sourceRoot, destDir, version string, channel runtimeChannel) (*downloader.DownloadResult, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create binary destination %q: %w", destDir, err)
	}

	binaryName := defaultBinaryName(channel)
	if strings.EqualFold(filepath.Ext(os.Args[0]), ".exe") {
		binaryName += ".exe"
	}
	destPath := filepath.Join(destDir, binaryName)
	tmpPath := filepath.Join(destDir, fmt.Sprintf(".aether-build-%d", os.Getpid()))
	defer os.Remove(tmpPath)

	ldflags := fmt.Sprintf("-X github.com/calcosmic/Aether/cmd.Version=%s", version)
	buildCmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", tmpPath, "./cmd/aether")
	buildCmd.Dir = sourceRoot
	out, err := buildCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go build failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return nil, fmt.Errorf("chmod built binary: %w", err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return nil, fmt.Errorf("install built binary %q: %w", destPath, err)
	}

	return &downloader.DownloadResult{
		Success: true,
		Path:    destPath,
		Version: version,
	}, nil
}
