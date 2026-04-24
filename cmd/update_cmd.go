package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/calcosmic/Aether/pkg/downloader"
	"github.com/spf13/cobra"
)

// updateCmd implements "aether update" which syncs companion files from the
// hub and optionally downloads a new binary.
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Aether companion files and optionally the binary",
	Long: "Update Aether by syncing companion files from the distribution hub\n" +
		"(~/.aether/system/ for stable, ~/.aether-dev/system/ for dev) to the local .aether/ directory.\n\n" +
		"This updates slash commands, agent definitions, skills, templates, and docs.\n" +
		"Local user data (COLONY_STATE.json, pheromones, etc.) is never overwritten.\n\n" +
		"By default this does not replace the installed `aether` binary.\n" +
		"Use `--download-binary` to fetch a published release binary.\n" +
		"If you need an unreleased local runtime fix from an Aether source checkout,\n" +
		"run `aether install --package-dir <Aether checkout>` in the Aether repo first.",
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
	updateCmd.Flags().String("channel", "", "Runtime channel to update from (stable or dev; default: infer from binary/env)")
	updateCmd.Flags().Bool("download-binary", false, "Also download a binary from GitHub Releases")
	updateCmd.Flags().String("binary-version", "", "Binary version to download (default: resolved installed version)")
	updateCmd.Flags().Bool("dry-run", false, "Show what would be updated without making changes")
	updateCmd.Flags().Bool("force", false, "Overwrite modified companion files and remove stale ones")

	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	channel := runtimeChannelFromFlag(cmd.Flags())

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	// Check hub exists
	hubDir := resolveHubPathForHome(homeDir, channel)
	hubVersionFile := filepath.Join(hubDir, "version.json")
	if _, err := os.Stat(hubVersionFile); os.IsNotExist(err) {
		outputErrorMessage("Aether hub not installed. Run \"aether install\" first.")
		return nil
	}

	// Read hub version for comparison
	hubVersion := readHubVersion(hubVersionFile)
	binaryVersion := resolveVersion()
	downloadBinary, _ := cmd.Flags().GetBool("download-binary")
	binaryMode := updateBinaryRefreshMode(downloadBinary, dryRun)

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
			"message":             fmt.Sprintf("Dry run — would sync companion files from hub [%s]", mode),
			"hub_version":         hubVersion,
			"local_version":       resolveVersion(),
			"force":               force,
			"binary_refresh_mode": binaryMode,
			"binary_refresh_note": updateBinaryRefreshNote(binaryMode, channel),
			"actions": []string{
				"Sync .aether/ system files (commands, agents, skills, templates, docs)",
				"Refresh repo-level Codex guidance (AGENTS.md, .codex/CODEX.md) when managed by Aether",
				"Sync .claude/commands/ant/",
				"Sync .claude/settings.json",
				"Sync .claude/agents/ant/",
				"Sync .codex/agents/",
				"Sync .codex/skills/",
				"Sync .opencode/commands/ant/",
				"Sync .opencode/agents/",
				fmt.Sprintf("Do not change the installed %s binary unless --download-binary is also used", defaultBinaryName(channel)),
			},
		}
		staleResult := checkStalePublish(hubDir, hubVersion, binaryVersion, channel, []map[string]interface{}{})
		result["stale_publish"] = staleResultToMap(staleResult)
		visual := renderUpdateVisual(repoDir, hubVersion, binaryVersion, force, true, []map[string]interface{}{
			{"label": "System files", "copied": 0, "skipped": 0},
			{"label": "Commands (claude)", "copied": 0, "skipped": 0},
			{"label": "Settings (claude)", "copied": 0, "skipped": 0},
			{"label": "Agents (claude)", "copied": 0, "skipped": 0},
			{"label": "Agents (codex)", "copied": 0, "skipped": 0},
			{"label": "Skills (codex)", "copied": 0, "skipped": 0},
			{"label": "Commands (opencode)", "copied": 0, "skipped": 0},
			{"label": "Agents (opencode)", "copied": 0, "skipped": 0},
		}, 0, 0, nil, binaryMode, hubVersion == binaryVersion)
		if staleResult.Classification != staleOK {
			visual += renderStalePublishBanner(staleResult)
		}
		outputWorkflow(result, visual)
		if staleResult.Classification == staleCritical {
			return fmt.Errorf("stale publish detected: %s", staleResult.Message)
		}
	} else {
		syncResult := runUpdateSync(hubDir, repoDir, force)
		if len(syncResult.errors) > 0 {
			outputError(2, fmt.Sprintf("update failed with %d sync error(s)", len(syncResult.errors)), map[string]interface{}{
				"hub_version":         hubVersion,
				"local_version":       resolveVersion(),
				"force":               force,
				"details":             syncResult.details,
				"binary_refresh_mode": binaryMode,
				"binary_refresh_note": updateBinaryRefreshNote(binaryMode, channel),
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
				"hub_version":         hubVersion,
				"local_version":       resolveVersion(),
				"force":               force,
				"details":             syncResult.details,
				"binary_refresh_mode": binaryMode,
				"binary_refresh_note": updateBinaryRefreshNote(binaryMode, channel),
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
		staleResult := checkStalePublish(hubDir, hubVersion, binaryVersion, channel, syncResult.details)
		result := map[string]interface{}{
			"message":                 message,
			"hub_version":             hubVersion,
			"local_version":           binaryVersion,
			"force":                   force,
			"details":                 syncResult.details,
			"binary_refresh_mode":     binaryMode,
			"binary_refresh_note":     updateBinaryRefreshNote(binaryMode, channel),
			"legacy_session_restored": mirrorRestored,
			"codex_restart_required":  len(restartTargets) > 0,
			"codex_restart_targets":   restartTargets,
			"stale_publish":           staleResultToMap(staleResult),
		}
		visual := renderUpdateVisual(repoDir, hubVersion, binaryVersion, force, false, syncResult.details, syncResult.copied, syncResult.skipped, restartTargets, binaryMode, hubVersion == binaryVersion)
		if staleResult.Classification != staleOK {
			visual += renderStalePublishBanner(staleResult)
		}
		outputWorkflow(result, visual)
		if staleResult.Classification == staleCritical {
			return fmt.Errorf("stale publish detected: %s", staleResult.Message)
		}
	}

	// Download binary if requested
	if downloadBinary && !dryRun {
		versionFlag, _ := cmd.Flags().GetString("binary-version")
		if normalizeVersion(versionFlag) == "latest" {
			versionFlag = ""
		}
		version, err := resolveReleaseVersion(versionFlag)
		if err != nil {
			return err
		}

		destDir := filepath.Join(homeDir, defaultBinaryDestSubdirForChannel(channel))
		outputWorkflow(map[string]interface{}{
			"message": fmt.Sprintf("Downloading aether %s binary...", version),
		}, renderBinaryActionVisual("Binary Download", fmt.Sprintf("Downloading aether %s binary...", version), version, destDir))

		result, err := downloader.DownloadBinary(version, destDir)
		if err != nil {
			return fmt.Errorf("file sync succeeded but binary download failed: %w", err)
		}
		result, err = alignDownloadedBinaryToChannel(result, destDir, channel)
		if err != nil {
			return fmt.Errorf("file sync succeeded but channel binary rename failed: %w", err)
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

func updateBinaryRefreshMode(downloadBinary, dryRun bool) string {
	if !downloadBinary {
		return "unchanged"
	}
	if dryRun {
		return "release-download-preview"
	}
	return "release-download"
}

func updateBinaryRefreshNote(mode string, channel runtimeChannel) string {
	binaryLabel := defaultBinaryName(channel)
	switch mode {
	case "release-download-preview":
		return fmt.Sprintf("Companion files would be synced first, then a published %s release binary would be downloaded.", binaryLabel)
	case "release-download":
		return fmt.Sprintf("Companion files were synced first; a published %s release binary will be downloaded next.", binaryLabel)
	default:
		return fmt.Sprintf("The installed %s binary is unchanged — `aether update` only syncs repo companion files, not the shared binary. Run `aether publish` in the Aether repo to update the binary.", binaryLabel)
	}
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
		".aether":     true,
		"archive":     true,
		"backups":     true,
		"chambers":    true,
		"checkpoints": true,
		"data":        true,
		"dreams":      true,
		"locks":       true,
		"oracle":      true,
		"temp":        true,
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
			include:              pair.include,
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

// --- stale-publish detection ---

type stalePublishClassification string

const (
	staleOK       stalePublishClassification = "ok"
	staleCritical stalePublishClassification = "critical"
	staleWarning  stalePublishClassification = "warning"
	staleInfo     stalePublishClassification = "info"
)

const (
	expectedClaudeCommandCount   = 50
	expectedOpenCodeCommandCount = 50
	expectedOpenCodeAgentCount   = 25
	expectedCodexAgentCount      = 25
	expectedCodexSkillCount      = 29
)

type staleComponent struct {
	Name     string `json:"name"`
	Expected int    `json:"expected"`
	Actual   int    `json:"actual"`
}

type stalePublishResult struct {
	Classification  stalePublishClassification `json:"classification"`
	BinaryVersion   string                     `json:"binary_version"`
	HubVersion      string                     `json:"hub_version"`
	Channel         string                     `json:"channel"`
	Message         string                     `json:"message"`
	Components      []staleComponent           `json:"components,omitempty"`
	RecoveryCommand string                     `json:"recovery_command"`
}

// compareVersions compares two semver strings segment by segment.
// Returns -1 if a < b, 0 if equal, 1 if a > b.
func compareVersions(a, b string) int {
	a = normalizeVersion(a)
	b = normalizeVersion(b)
	if a == b {
		return 0
	}
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}
	for i := 0; i < maxLen; i++ {
		var aInt, bInt int
		if i < len(aParts) {
			aInt, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bInt, _ = strconv.Atoi(bParts[i])
		}
		if aInt < bInt {
			return -1
		}
		if aInt > bInt {
			return 1
		}
	}
	return 0
}

func checkStalePublish(hubDir, hubVersion, binaryVersion string, channel runtimeChannel, syncDetails []map[string]interface{}) stalePublishResult {
	result := stalePublishResult{
		BinaryVersion: binaryVersion,
		HubVersion:    hubVersion,
		Channel:       string(channel),
	}

	if hubVersion == "" || hubVersion == "unknown" {
		result.Classification = staleInfo
		result.Message = "Hub version is unknown — cannot verify publish freshness."
		result.RecoveryCommand = recoveryCommandForChannel(channel)
		return result
	}

	cmp := compareVersions(hubVersion, binaryVersion)
	switch {
	case cmp < 0:
		result.Classification = staleCritical
		result.Message = fmt.Sprintf("Critical: hub version %s is behind binary version %s", hubVersion, binaryVersion)
	case cmp > 0:
		result.Classification = staleWarning
		result.Message = fmt.Sprintf("Warning: hub version %s is ahead of binary version %s", hubVersion, binaryVersion)
	default:
		result.Classification = staleOK
	}

	// Check companion-file completeness in hubDir
	hubSystem := filepath.Join(hubDir, "system")
	checks := []struct {
		name      string
		path      string
		expected  int
		filter    func(string) bool
		recursive bool
	}{
		{"Commands (claude)", filepath.Join(hubSystem, "commands", "claude"), expectedClaudeCommandCount, nil, false},
		{"Commands (opencode)", filepath.Join(hubSystem, "commands", "opencode"), expectedOpenCodeCommandCount, nil, false},
		{"Agents (opencode)", filepath.Join(hubSystem, "agents"), expectedOpenCodeAgentCount, nil, false},
		{"Agents (codex)", filepath.Join(hubSystem, "codex"), expectedCodexAgentCount, func(name string) bool { return strings.HasSuffix(name, ".toml") }, false},
		{"Skills (codex)", filepath.Join(hubSystem, "skills-codex"), expectedCodexSkillCount, nil, true},
	}

	for _, check := range checks {
		var actual int
		if check.recursive {
			actual = countEntriesRecursive(check.path, check.filter)
		} else {
			actual = countEntriesInDir(check.path, check.filter)
		}
		if actual < check.expected {
			result.Components = append(result.Components, staleComponent{
				Name:     check.name,
				Expected: check.expected,
				Actual:   actual,
			})
		}
	}

	if len(result.Components) > 0 && result.Classification == staleOK {
		result.Classification = staleInfo
		result.Message = "Info: companion files are incomplete."
	}

	if result.Classification == staleOK {
		result.Message = "Publish is fresh: binary and hub versions agree, companion files look complete."
	}

	result.RecoveryCommand = recoveryCommandForChannel(channel)
	return result
}

func countEntriesInDir(dir string, filter func(string) bool) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filter != nil && !filter(entry.Name()) {
			continue
		}
		count++
	}
	return count
}

func countEntriesRecursive(dir string, filter func(string) bool) int {
	count := 0
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filter != nil && !filter(d.Name()) {
			return nil
		}
		count++
		return nil
	})
	return count
}

func recoveryCommandForChannel(channel runtimeChannel) string {
	if channel == channelDev {
		return "In the Aether repo, run: aether publish --channel dev"
	}
	return "In the Aether repo, run: aether publish"
}

func staleResultToMap(r stalePublishResult) map[string]interface{} {
	components := make([]map[string]interface{}, len(r.Components))
	for i, c := range r.Components {
		components[i] = map[string]interface{}{
			"name":     c.Name,
			"expected": c.Expected,
			"actual":   c.Actual,
		}
	}
	return map[string]interface{}{
		"classification":   string(r.Classification),
		"binary_version":   r.BinaryVersion,
		"hub_version":      r.HubVersion,
		"channel":          r.Channel,
		"message":          r.Message,
		"components":       components,
		"recovery_command": r.RecoveryCommand,
	}
}
