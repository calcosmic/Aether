package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/downloader"
	"github.com/spf13/cobra"
)

// updateCmd implements "aether update" which syncs companion files from the
// hub and optionally downloads a new binary.
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Aether companion files and optionally the binary",
	Long: "Update Aether by syncing companion files from the distribution hub\n" +
		"(~/.aether/system/ for stable, ~/.aether-dev/system/ for dev) and\n" +
		"refreshing repo-local state scaffolding.\n\n" +
		"Shared agents, commands, shipped skills, templates, docs, utils, workers,\n" +
		"exchange files, and references stay global instead of being copied into\n" +
		"target repos. Local user data (COLONY_STATE.json, pheromones, etc.) is\n" +
		"never overwritten.\n\n" +
		"By default this does not replace the installed `aether` binary.\n" +
		"Use `--download-binary` to fetch a published release binary.\n" +
		"If you need an unreleased local runtime fix from an Aether source checkout,\n" +
		"run `aether publish --package-dir <Aether checkout>` in the Aether repo first.",
	Args: cobra.NoArgs,
	RunE: runMaintenanceUpdate,
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
	updateCmd.Flags().Bool("sync-platform-homes", false, "For dev channel, also sync global Claude/OpenCode/Codex home assets")

	rootCmd.AddCommand(updateCmd)
}

// runMaintenanceUpdate is the public update path. It first builds one exact
// target manifest, emits that same manifest for --dry-run, and otherwise hands
// every changed repository/platform/binary target to LifecycleTransaction.
// The older sync kernels remain below for install/backward-compatible unit
// coverage, but update itself no longer commits through them directly.
func runMaintenanceUpdate(cmd *cobra.Command, _ []string) error {
	channel := runtimeChannelFromFlag(cmd.Flags())
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	repositoryRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}
	repositoryRoot, err = filepath.Abs(repositoryRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	if store == nil {
		return fmt.Errorf("update requires an initialized lifecycle store")
	}
	repoVersionBefore := ""
	if marker, ok := readInstalledVersionMarker(repositoryRoot); ok {
		repoVersionBefore = marker.Version
	}
	aliasSurfacesMissingBefore := missingDeclaredAliasSurfaces(homeDir)
	dataRoot := filepath.Clean(store.BasePath())
	hubRoot := filepath.Clean(resolveHubPathForHome(homeDir, channel))
	hubVersion := normalizeVersion(readHubVersionAtPath(hubRoot))
	if hubVersion == "" {
		return fmt.Errorf("Aether hub not installed; run aether install first")
	}
	if err := validateMaintenanceVersionAgreement(repositoryRoot, hubRoot, hubVersion); err != nil {
		return err
	}
	if isAetherSourceCheckout(repositoryRoot) {
		check := runSourceCheck(repositoryRoot)
		if !check.OK {
			return fmt.Errorf("maintenance update: source/generated parity check failed")
		}
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	syncPlatformHomes, _ := cmd.Flags().GetBool("sync-platform-homes")
	downloadBinary, _ := cmd.Flags().GetBool("download-binary")
	now := time.Now().UTC()
	plan := maintenanceMutationPlan{
		SchemaVersion:   maintenanceMutationSchemaVersion,
		Operation:       "update",
		TransactionID:   "update-" + now.Format("20060102T150405.000000000Z"),
		SourceRoot:      hubRoot,
		DestinationRoot: repositoryRoot,
		Channel:         channel,
		CurrentVersion:  resolveVersion(),
		DesiredVersion:  hubVersion,
		Checkpoint:      "maintenance:update:validated",
		Recovery:        "aether resume",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot:    repositoryRoot,
			LifecycleDataRoot: dataRoot,
			Hub: lifecycleTransactionHubRoot{
				Channel: maintenanceHubChannel(channel),
				Path:    hubRoot,
			},
		},
	}
	if err := appendMaintenanceUpdateRepositoryTargets(&plan, hubRoot, repositoryRoot, force, now); err != nil {
		return err
	}
	if shouldSyncPlatformHomes(channel, syncPlatformHomes) {
		if channel == channelDev {
			return fmt.Errorf("maintenance update: dev channel is isolated from stable platform homes")
		}
		if err := appendMaintenanceUpdatePlatformTargets(&plan, hubRoot, homeDir); err != nil {
			return err
		}
	}
	if downloadBinary {
		versionFlag, _ := cmd.Flags().GetString("binary-version")
		versionFlag = normalizeVersion(versionFlag)
		if versionFlag == "" || versionFlag == "latest" {
			versionFlag = hubVersion
		}
		if versionFlag != hubVersion {
			return fmt.Errorf("maintenance update: binary version %s must match companion version %s", versionFlag, hubVersion)
		}
		staged, err := stageMaintenanceBinaryDownload(versionFlag, channel, downloader.DownloadBinary)
		if err != nil {
			return err
		}
		destinationDir := filepath.Join(homeDir, defaultBinaryDestSubdirForChannel(channel))
		destination, err := filepath.Abs(filepath.Join(destinationDir, staged.Name))
		if err != nil {
			return fmt.Errorf("resolve binary destination: %w", err)
		}
		plan.Allowlist.BinaryDestination = destination
		plan.Targets = append(plan.Targets, maintenanceMutationTarget{
			Root: lifecycleTransactionRootBinaryDestination, RelativeTarget: filepath.Base(destination),
			Source: staged.Source, Action: lifecycleTransactionWrite, Content: staged.Content, Managed: true,
		})
	}

	preview, err := prepareMaintenanceMutation(plan)
	if err != nil {
		return err
	}
	if dryRun {
		result := map[string]interface{}{
			"operation": "update", "preview": preview, "state_effect": "none",
			"binary_refresh_mode": updateBinaryRefreshMode(downloadBinary, true), "recovery": plan.Recovery,
		}
		details, copied, skipped := maintenancePreviewSyncDetails(preview)
		stale := checkStalePublish(hubRoot, hubVersion, resolveVersion(), channel, details)
		result["stale_publish"] = staleResultToMap(stale)
		outputWorkflow(result, renderUpdateVisual(repositoryRoot, hubVersion, resolveVersion(), renderRepoVersionTransition(repoVersionBefore, hubVersion, true), force, true, details, copied, skipped, nil, updateBinaryRefreshMode(downloadBinary, true), hubVersion == resolveVersion(), result))
		return nil
	}

	mutation, err := commitMaintenanceMutation(plan)
	result := map[string]interface{}{
		"operation": "update", "preview": mutation.Preview, "targets": mutation.Targets,
		"transaction": mutation.Preview.TransactionID, "receipt": mutation.Receipt,
		"state_effect": mutation.StateEffect, "verification": mutation.Verification,
		"recovery": mutation.Recovery, "hub_version": hubVersion,
		"local_version": resolveVersion(), "binary_refresh_mode": updateBinaryRefreshMode(downloadBinary, false),
	}
	details, copied, skipped := maintenancePreviewSyncDetails(mutation.Preview)
	aliasRepairReport := diffAliasRepairs(aliasSurfacesMissingBefore)
	message := fmt.Sprintf("Updated: %d files copied, %d unchanged", copied, skipped)
	if repair := aliasRepairReport.Message(); repair != "" {
		message += ". " + repair
	}
	result["message"] = message
	result["alias_wrapper_repairs"] = aliasRepairReport.Repairs
	result["stale_publish"] = staleResultToMap(checkStalePublish(hubRoot, hubVersion, resolveVersion(), channel, details))
	if err != nil {
		outputWorkflow(result, renderMaintenanceMutationPreview(mutation.Preview, false))
		return err
	}
	closeLifecycleCommand(result, updateLastCommandFact(aliasRepairReport.Message()), "", "")
	restartTargets := platformRestartTargets(details)
	outputWorkflow(result, renderUpdateVisual(repositoryRoot, hubVersion, resolveVersion(), renderRepoVersionTransition(repoVersionBefore, hubVersion, false), force, false, details, copied, skipped, restartTargets, updateBinaryRefreshMode(downloadBinary, false), hubVersion == resolveVersion(), result))
	return nil
}

func maintenancePreviewSyncDetails(preview maintenanceMutationPreview) ([]map[string]interface{}, int, int) {
	type counts struct{ copied, skipped, removed int }
	order := []lifecycleTransactionRootKind{}
	byRoot := map[lifecycleTransactionRootKind]*counts{}
	for _, target := range preview.Targets {
		count := byRoot[target.Root]
		if count == nil {
			count = &counts{}
			byRoot[target.Root] = count
			order = append(order, target.Root)
		}
		switch target.Change {
		case maintenanceMutationChangeWrite:
			count.copied++
		case maintenanceMutationChangeRemove:
			count.removed++
		default:
			count.skipped++
		}
	}
	var details []map[string]interface{}
	totalCopied, totalSkipped := 0, 0
	for _, root := range order {
		count := byRoot[root]
		details = append(details, map[string]interface{}{"label": "Transaction root " + string(root), "copied": count.copied, "skipped": count.skipped, "removed": count.removed})
		totalCopied += count.copied
		totalSkipped += count.skipped
	}
	return details, totalCopied, totalSkipped
}

func maintenanceHubChannel(channel runtimeChannel) lifecycleTransactionHubChannel {
	if channel == channelDev {
		return lifecycleTransactionHubDev
	}
	return lifecycleTransactionHubStable
}

func appendMaintenanceUpdateRepositoryTargets(plan *maintenanceMutationPlan, hubRoot, repositoryRoot string, force bool, now time.Time) error {
	hubSystem := filepath.Join(hubRoot, "system")
	localAether := filepath.Join(repositoryRoot, ".aether")
	if !isAetherSourceCheckout(repositoryRoot) {
		for _, pair := range repoSyncPairs() {
			if pair.consumerOnly || pair.hubRel != "." {
				base := filepath.Clean(filepath.Join(".aether", filepath.FromSlash(pair.destRel)))
				spec := maintenanceSyncSpec{
					Root: lifecycleTransactionRootRepository, SourceDir: filepath.Join(hubSystem, filepath.FromSlash(pair.hubRel)), DestinationBase: base,
					Options:             syncOptions{cleanup: pair.cleanup, preserveLocalChanges: !force && pair.preserveLocalChanges, protectedDirs: map[string]bool{"data": true, "dreams": true, "oracle": true, "locks": true, "checkpoints": true, "archive": true, "backups": true, "chambers": true, "temp": true}, protectedFiles: map[string]bool{"QUEEN.md": true, "CROWNED-ANTHILL.md": true}, validate: pair.validate, include: pair.include, mapRelPath: pair.mapRelPath, cleanupInclude: pair.cleanupInclude, merge: pair.merge},
					PruneRetiredAliases: pair.cleanupLegacyClaude,
				}
				if pair.consumerOnly {
					spec.CleanupOwned = func(relativePath string, _ []byte) bool { return isManagedAetherSystemPath(relativePath) }
				}
				if err := appendMaintenanceSyncTargets(plan, spec); err != nil {
					return fmt.Errorf("plan %s: %w", pair.label, err)
				}
			}
		}
		if err := appendMaintenanceLegacyRepositoryTargets(plan, hubSystem, repositoryRoot); err != nil {
			return err
		}
	}

	if err := appendMaintenanceProjectDocTargets(plan, hubSystem, repositoryRoot); err != nil {
		return err
	}
	if err := appendMaintenanceTsHostTargets(plan, hubRoot); err != nil {
		return err
	}
	if err := appendMaintenanceScaffoldTargets(plan, localAether); err != nil {
		return err
	}
	marker, err := json.MarshalIndent(installedVersionMarker{Version: normalizeVersion(plan.DesiredVersion), UpdatedAt: now.Format(time.RFC3339)}, "", "  ")
	if err != nil {
		return err
	}
	plan.Targets = append(plan.Targets, maintenanceMutationTarget{
		Root: lifecycleTransactionRootRepository, RelativeTarget: filepath.Join(".aether", filepath.FromSlash(installedVersionMarkerRel)),
		Source: "selected hub version manifest", Action: lifecycleTransactionWrite, Content: append(marker, '\n'), Managed: true,
	})
	return nil
}

func appendMaintenanceProjectDocTargets(plan *maintenanceMutationPlan, hubSystem, repositoryRoot string) error {
	for _, spec := range []projectDocSpec{
		{templateRel: filepath.Join("templates", "agents-md-template.md"), destRel: "AGENTS.md", managedFn: isAetherManagedAgentsDoc},
		{templateRel: filepath.Join("templates", "codex-md-template.md"), destRel: filepath.Join(".codex", "CODEX.md"), managedFn: isAetherManagedCodexDoc},
		{templateRel: filepath.Join("templates", "opencode-md-template.md"), destRel: filepath.Join(".opencode", "OPENCODE.md"), managedFn: isAetherManagedOpenCodeDoc},
	} {
		source := filepath.Join(hubSystem, spec.templateRel)
		data, err := os.ReadFile(source)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		destination := filepath.Join(repositoryRoot, spec.destRel)
		if existing, readErr := os.ReadFile(destination); readErr == nil && !spec.managedFn(string(existing)) {
			continue
		} else if readErr != nil && !os.IsNotExist(readErr) {
			return readErr
		}
		plan.Targets = append(plan.Targets, maintenanceMutationTarget{Root: lifecycleTransactionRootRepository, RelativeTarget: spec.destRel, Source: source, Action: lifecycleTransactionWrite, Content: []byte(renderProjectDocTemplate(string(data))), Managed: true})
	}
	return nil
}

func appendMaintenanceScaffoldTargets(plan *maintenanceMutationPlan, localAether string) error {
	files := []struct {
		rel     string
		content string
	}{
		{".aether/WHAT-IS-THIS.md", "# What is this directory?\n\nThis is Aether's colony state for this repository — durable working memory, not a build cache.\n"},
		{".aether/QUEEN.md", queenDefaultContent},
	}
	for _, file := range files {
		if _, err := os.Lstat(filepath.Join(filepath.Dir(localAether), filepath.FromSlash(file.rel))); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		plan.Targets = append(plan.Targets, maintenanceMutationTarget{Root: lifecycleTransactionRootRepository, RelativeTarget: filepath.FromSlash(file.rel), Source: "Aether local-state schema", Action: lifecycleTransactionWrite, Content: []byte(file.content), Managed: true})
	}
	gitignore := filepath.Join(localAether, ".gitignore")
	content := "# Aether local state - not versioned\ndata/\ncheckpoints/\nlocks/\ndreams/\noracle/\nts-host/node_modules/\n"
	if current, err := os.ReadFile(gitignore); err == nil {
		if strings.Contains(string(current), "ts-host/node_modules") {
			content = string(current)
		} else {
			content = strings.TrimRight(string(current), "\n") + "\nts-host/node_modules/\n"
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	plan.Targets = append(plan.Targets, maintenanceMutationTarget{Root: lifecycleTransactionRootRepository, RelativeTarget: ".aether/.gitignore", Source: "Aether local-state schema", Action: lifecycleTransactionWrite, Content: []byte(content), Managed: true})
	return nil
}

func appendMaintenanceLegacyRepositoryTargets(plan *maintenanceMutationPlan, hubSystem, repositoryRoot string) error {
	for _, base := range []string{filepath.Join(".claude", "commands"), filepath.Join(".opencode", "commands", "ant")} {
		root := filepath.Join(repositoryRoot, base)
		files, err := listMaintenanceRegularFilesIfPresent(root)
		if err != nil {
			return err
		}
		for _, rel := range files {
			data, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				return err
			}
			if !isGeneratedAetherCommandWrapper(data) {
				continue
			}
			plan.Targets = append(plan.Targets, maintenanceMutationTarget{Root: lifecycleTransactionRootRepository, RelativeTarget: filepath.Join(base, rel), Source: "legacy generated wrapper ownership header", Action: lifecycleTransactionRemove, Managed: true})
		}
	}
	localSkills := filepath.Join(repositoryRoot, ".aether", "skills")
	for _, rel := range listFilesRecursive(localSkills) {
		localPath := filepath.Join(localSkills, rel)
		hubPath := filepath.Join(hubSystem, "skills", rel)
		local, localErr := os.ReadFile(localPath)
		hub, hubErr := os.ReadFile(hubPath)
		if localErr == nil && hubErr == nil && bytes.Equal(local, hub) {
			plan.Targets = append(plan.Targets, maintenanceMutationTarget{Root: lifecycleTransactionRootRepository, RelativeTarget: filepath.Join(".aether", "skills", rel), Source: hubPath + " exact shipped match", Action: lifecycleTransactionRemove, Managed: true})
		}
	}
	return nil
}

func appendMaintenanceTsHostTargets(plan *maintenanceMutationPlan, hubRoot string) error {
	source := tsHostHubDir(hubRoot)
	if _, err := os.Lstat(source); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if err := validateTsHostHubArtifacts(hubRoot); err != nil {
		return err
	}
	if err := appendMaintenanceSyncTargets(plan, maintenanceSyncSpec{Root: lifecycleTransactionRootRepository, SourceDir: filepath.Join(source, "dist"), DestinationBase: filepath.Join(".aether", "ts-host", "dist"), Options: syncOptions{cleanup: true}, CleanupOwned: func(string, []byte) bool { return true }}); err != nil {
		return err
	}
	for _, name := range []string{"package.json", "package-lock.json"} {
		data, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			return err
		}
		plan.Targets = append(plan.Targets, maintenanceMutationTarget{Root: lifecycleTransactionRootRepository, RelativeTarget: filepath.Join(".aether", "ts-host", name), Source: filepath.Join(source, name), Action: lifecycleTransactionWrite, Content: data, Managed: true})
	}
	return nil
}

func appendMaintenanceUpdatePlatformTargets(plan *maintenanceMutationPlan, hubRoot, homeDir string) error {
	claudeRoot := filepath.Join(homeDir, ".claude")
	codexRoot := filepath.Join(homeDir, ".codex")
	needClaude, needOpenCode, needCodex := false, false, false
	for _, pair := range platformHomeHubSyncPairs() {
		if _, err := os.Lstat(filepath.Join(hubRoot, "system", filepath.FromSlash(pair.srcRel))); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		switch {
		case strings.HasPrefix(filepath.ToSlash(pair.destRel), ".claude/"):
			needClaude = true
		case strings.HasPrefix(filepath.ToSlash(pair.destRel), ".codex/"):
			needCodex = true
		default:
			needOpenCode = true
		}
	}
	for label, root := range map[string]string{"Claude": claudeRoot, "Codex": codexRoot} {
		if (label == "Claude" && !needClaude) || (label == "Codex" && !needCodex) {
			continue
		}
		info, err := os.Lstat(root)
		if err != nil {
			return fmt.Errorf("maintenance update: %s home is unavailable; run aether install first: %w", label, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("maintenance update: %s home is not a real directory", label)
		}
	}
	if needClaude {
		plan.Allowlist.ClaudeHome = claudeRoot
	}
	if needOpenCode {
		plan.Allowlist.OpenCodeHome = homeDir
	}
	if needCodex {
		plan.Allowlist.CodexHome = codexRoot
	}
	for _, pair := range platformHomeHubSyncPairs() {
		sourceDir := filepath.Join(hubRoot, "system", filepath.FromSlash(pair.srcRel))
		if _, err := os.Lstat(sourceDir); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		rootKind := lifecycleTransactionRootOpenCodeHome
		base := filepath.FromSlash(pair.destRel)
		if strings.HasPrefix(filepath.ToSlash(pair.destRel), ".claude/") {
			rootKind = lifecycleTransactionRootClaudeHome
			base = strings.TrimPrefix(filepath.ToSlash(pair.destRel), ".claude/")
		} else if strings.HasPrefix(filepath.ToSlash(pair.destRel), ".codex/") {
			rootKind = lifecycleTransactionRootCodexHome
			base = strings.TrimPrefix(filepath.ToSlash(pair.destRel), ".codex/")
		}
		err := appendMaintenanceSyncTargets(plan, maintenanceSyncSpec{
			Root: rootKind, SourceDir: sourceDir, DestinationBase: filepath.FromSlash(base),
			Options:             syncOptions{cleanup: pair.cleanup, preserveLocalChanges: pair.preserveLocalChanges, validate: pair.validate, include: pair.include, mapRelPath: pair.mapRelPath, cleanupInclude: pair.cleanupInclude},
			PruneRetiredAliases: pair.cleanupLegacyClaude,
		})
		if err != nil {
			return fmt.Errorf("plan %s: %w", pair.label, err)
		}
	}
	return nil
}

func renderMaintenanceMutationPreview(preview maintenanceMutationPreview, dryRun bool) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("update"), "Maintenance Update"))
	if dryRun {
		b.WriteString("Preview only — no targets were changed.\n")
	} else {
		b.WriteString("Maintenance transaction completed.\n")
	}
	fmt.Fprintf(&b, "Version: %s -> %s\n", preview.CurrentVersion, preview.DesiredVersion)
	for _, target := range preview.Targets {
		fmt.Fprintf(&b, "- %s %s:%s (%s -> %s)\n", target.Change, target.Root, filepath.ToSlash(target.RelativeTarget), target.CurrentDigest, target.DesiredDigest)
	}
	b.WriteString(renderNextUp("Use `aether resume` only if the transaction reports recovery required."))
	return b.String()
}

// updateLastCommandFact is what update reports it did, fed to the one
// resolver as the "what changed" fact. Someone whose project was just
// repaired should be told something different from someone whose project
// was already fine -- repairMsg is empty in the second case.
func updateLastCommandFact(repairMsg string) string {
	if strings.TrimSpace(repairMsg) == "" {
		return "update"
	}
	return "update (" + repairMsg + ")"
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

	appendSyncResult(&result.details, &result, "Local state scaffold", ensureRepoLocalScaffold(localAether))

	sourceCheckout := isAetherSourceCheckout(repoDir)
	for _, pair := range repoSyncPairs() {
		if pair.consumerOnly && sourceCheckout {
			appendSyncResult(&result.details, &result, pair.label, syncResult{})
			continue
		}
		srcDir := filepath.Join(hubSystem, filepath.FromSlash(pair.hubRel))
		destDir := filepath.Join(localAether, filepath.FromSlash(pair.destRel))

		syncRes := syncDir(srcDir, destDir, syncOptions{
			cleanup:              pair.cleanup,
			preserveLocalChanges: !force && pair.preserveLocalChanges,
			protectedDirs:        protectedDirs,
			protectedFiles:       protectedFiles,
			validate:             pair.validate,
			include:              pair.include,
			mapRelPath:           pair.mapRelPath,
			cleanupInclude:       pair.cleanupInclude,
			merge:                pair.merge,
		})
		if pair.cleanupLegacyClaude && force {
			removed, errors := removeLegacyClaudeCommandNamespace(destDir)
			syncRes.removed = append(syncRes.removed, removed...)
			syncRes.errors = append(syncRes.errors, errors...)
		}
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

	appendSyncResult(&result.details, &result, "Prune legacy repo platform assets", pruneLegacyRepoPlatformAssets(repoDir))
	appendSyncResult(&result.details, &result, "Prune repo Codex skill mirror", pruneRepoCodexSkillMirror(repoDir, force))
	appendSyncResult(&result.details, &result, "Prune shipped repo skills", pruneShippedRepoSkills(hubSystem, localAether, force))

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

// validateMaintenanceVersionAgreement binds a maintenance transaction to one
// release identity. Source checkouts must agree across the canonical Go asset
// version and npm package; the selected hub must carry the same version before
// any destination is staged.
func validateMaintenanceVersionAgreement(sourceRoot, hubRoot, desiredVersion string) error {
	desiredVersion = normalizeVersion(strings.TrimSpace(desiredVersion))
	if desiredVersion == "" {
		return fmt.Errorf("maintenance update: desired version is required")
	}
	hubVersion := normalizeVersion(readHubVersionAtPath(hubRoot))
	if hubVersion == "" {
		return fmt.Errorf("maintenance update: selected hub has no version manifest")
	}
	if hubVersion != desiredVersion {
		return fmt.Errorf("maintenance update: hub version %s does not match desired version %s", hubVersion, desiredVersion)
	}

	sourceVersionPath := filepath.Join(sourceRoot, ".aether", "version.json")
	npmVersionPath := filepath.Join(sourceRoot, "npm", "package.json")
	_, sourceStatErr := os.Stat(sourceVersionPath)
	_, npmStatErr := os.Stat(npmVersionPath)
	if !isAetherSourceCheckout(sourceRoot) && (sourceStatErr != nil || npmStatErr != nil) {
		return nil
	}
	if sourceStatErr != nil {
		return fmt.Errorf("maintenance update: source version manifest unavailable: %w", sourceStatErr)
	}
	if npmStatErr != nil {
		return fmt.Errorf("maintenance update: npm version manifest unavailable: %w", npmStatErr)
	}
	sourceVersion, err := readJSONVersion(sourceVersionPath)
	if err != nil {
		return fmt.Errorf("maintenance update: read source version: %w", err)
	}
	npmVersion, err := readJSONVersion(npmVersionPath)
	if err != nil {
		return fmt.Errorf("maintenance update: read npm version: %w", err)
	}
	sourceVersion = normalizeVersion(sourceVersion)
	npmVersion = normalizeVersion(npmVersion)
	if sourceVersion != npmVersion || sourceVersion != desiredVersion {
		return fmt.Errorf("maintenance update: version disagreement source=%s npm=%s hub=%s desired=%s", sourceVersion, npmVersion, hubVersion, desiredVersion)
	}
	return nil
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
	expectedClaudeCommandCount   = 60
	expectedOpenCodeCommandCount = 60
	expectedOpenCodeAgentCount   = 27
	expectedCodexAgentCount      = 27
	expectedCodexSkillCount      = 86
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
		{"Agents (claude)", filepath.Join(hubSystem, "agents-claude"), expectedClaudeAgents, nil, false},
		{"Commands (opencode)", filepath.Join(hubSystem, "commands", "opencode"), expectedOpenCodeCommandCount, nil, false},
		{"Agents (opencode)", filepath.Join(hubSystem, "agents"), expectedOpenCodeAgentCount, nil, false},
		{"Agents (codex)", filepath.Join(hubSystem, "codex"), expectedCodexAgentCount, func(name string) bool { return strings.HasSuffix(name, ".toml") }, false},
		{"Skills (hub)", filepath.Join(hubSystem, "skills"), expectedCodexSkillCount, nil, true},
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

// syncTsHostFromHub copies TS host assets from the hub to the local repo. Older
// hubs without TS host assets are skipped; partial TS host bundles fail.
func syncTsHostFromHub(hubDir, repoDir string) error {
	srcDir := tsHostHubDir(hubDir)
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return nil // No TS host in hub, skip silently
	}
	if err := validateTsHostHubArtifacts(hubDir); err != nil {
		return err
	}

	dstDir := tsHostRepoDir(repoDir)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dstDir, err)
	}

	// Sync dist/
	srcDist := filepath.Join(srcDir, "dist")
	dstDist := filepath.Join(dstDir, "dist")
	res := syncDir(srcDist, dstDist, syncOptions{cleanup: true})
	if len(res.errors) > 0 {
		return fmt.Errorf("sync dist/: %s", strings.Join(res.errors, "; "))
	}

	// Copy package.json
	srcPkg := filepath.Join(srcDir, "package.json")
	dstPkg := filepath.Join(dstDir, "package.json")
	if err := copyFile(srcPkg, dstPkg); err != nil {
		return fmt.Errorf("copy package.json: %w", err)
	}

	// Copy package-lock.json.
	srcLock := filepath.Join(srcDir, "package-lock.json")
	dstLock := filepath.Join(dstDir, "package-lock.json")
	if err := copyFile(srcLock, dstLock); err != nil {
		return fmt.Errorf("copy package-lock.json: %w", err)
	}

	return validateTsHostRepoArtifacts(repoDir)
}

// ensureTsHostBuilt checks that TS host dependencies are installed and dist/
// is built. Runs npm ci and npm run build when needed.
func ensureTsHostBuilt(repoDir string) error {
	return ensureTsHostDepsAt(tsHostRepoDir(repoDir))
}

// tsHostLockStampRel records which package-lock.json the installed
// node_modules was built from, so dependency drift triggers a reinstall
// instead of a cryptic ERR_MODULE_NOT_FOUND at runtime.
const tsHostLockStampRel = "node_modules/.aether-lock-hash"

// tsHostNpmCommand runs npm in the TS host directory. Overridable in tests.
// npm's chatter must NEVER reach stdout: with AETHER_OUTPUT_MODE=json the
// process's stdout is a machine-readable envelope, and the first update in
// a fresh repo used to emit npm's install summary ("added 63 packages...
// 2 vulnerabilities") ahead of the JSON, breaking every wrapper that
// parses it. Progress goes to stderr, where humans still see it.
var tsHostNpmCommand = func(dir string, args ...string) error {
	cmd := exec.Command("npm", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ensureTsHostDepsAt provisions a TS host directory in place: installs
// node_modules when missing or stale (lock hash mismatch) and builds dist/
// when absent. It is the single choke point used by update and by every
// `aether host` invocation, so fresh installs and hub-fallback paths
// self-heal instead of failing inside node.
func ensureTsHostDepsAt(tsHostDir string) error {
	if err := requireTsHostFile(tsHostDir, "package.json"); err != nil {
		return err
	}

	hasRuntimeDeps, err := tsHostHasRuntimeDependencies(tsHostDir)
	if err != nil {
		return err
	}
	distHostPath := filepath.Join(tsHostDir, filepath.FromSlash(tsHostEntryRelPath))
	if !hasRuntimeDeps && fileExists(distHostPath) {
		return validateTsHostArtifactSet(tsHostDir)
	}

	// Check npm availability
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm not found in PATH: %w", err)
	}

	lockPath := filepath.Join(tsHostDir, "package-lock.json")
	lockHash, err := fileSHA256(lockPath)
	if err != nil {
		return fmt.Errorf("hash package-lock.json: %w", err)
	}
	stampPath := filepath.Join(tsHostDir, filepath.FromSlash(tsHostLockStampRel))
	needInstall := true
	if data, readErr := os.ReadFile(stampPath); readErr == nil && strings.TrimSpace(string(data)) == lockHash {
		needInstall = false
	}
	if needInstall {
		if err := tsHostNpmCommand(tsHostDir, "ci", "--no-audit", "--no-fund"); err != nil {
			return fmt.Errorf("npm ci failed: %w", err)
		}
		if err := os.WriteFile(stampPath, []byte(lockHash+"\n"), 0644); err != nil {
			return fmt.Errorf("write dependency stamp: %w", err)
		}
	}

	if _, err := os.Stat(distHostPath); os.IsNotExist(err) {
		if err := tsHostNpmCommand(tsHostDir, "run", "build"); err != nil {
			return fmt.Errorf("npm run build failed: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("stat TS host dist entry: %w", err)
	}

	return validateTsHostArtifactSet(tsHostDir)
}
