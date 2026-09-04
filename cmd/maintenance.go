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
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

const (
	maintenanceCleanupSchemaVersion = "maintenance-cleanup/v1"
	maintenanceCleanupOwner         = "aether-runtime"
	maintenanceCleanupCheckpoint    = "maintenance:data-clean:validated"
	maintenanceCleanupManifestRel   = "maintenance/data-clean.json"
)

// maintenanceCleanupManifest is deletion authority. A filename prefix, age,
// or glob match can suggest a candidate, but only an exact path with the
// expected owner and byte digest may enter a cleanup transaction.
type maintenanceCleanupManifest struct {
	SchemaVersion string                     `json:"schema_version"`
	Owner         string                     `json:"owner"`
	Checkpoint    string                     `json:"checkpoint"`
	Targets       []maintenanceCleanupTarget `json:"targets"`
}

type maintenanceCleanupTarget struct {
	RelativePath string `json:"relative_path"`
	Owner        string `json:"owner"`
	Digest       string `json:"digest"`
}

type maintenanceCleanupRequest struct {
	RepositoryRoot string
	DataRoot       string
	TransactionID  string
	Manifest       maintenanceCleanupManifest
	Rename         func(oldPath, newPath string) error
	Fault          lifecycleTransactionFaultHook
}

type maintenanceCleanupPlan struct {
	Request  maintenanceCleanupRequest
	Preview  maintenanceMutationPreview
	mutation maintenanceMutationPlan
}

// prepareMaintenanceCleanup is deliberately read-only. It reduces an
// explicit cleanup manifest to the shared maintenance preview only after each
// owned file still matches the manifest's exact digest.
func prepareMaintenanceCleanup(request maintenanceCleanupRequest) (maintenanceCleanupPlan, error) {
	plan := maintenanceCleanupPlan{}
	request.RepositoryRoot = filepath.Clean(request.RepositoryRoot)
	request.DataRoot = filepath.Clean(request.DataRoot)
	request.TransactionID = strings.TrimSpace(request.TransactionID)
	request.Manifest.Targets = append([]maintenanceCleanupTarget(nil), request.Manifest.Targets...)
	plan.Request = request

	if request.Manifest.SchemaVersion != maintenanceCleanupSchemaVersion {
		return plan, fmt.Errorf("maintenance cleanup: schema_version must be %s", maintenanceCleanupSchemaVersion)
	}
	if request.Manifest.Owner != maintenanceCleanupOwner {
		return plan, fmt.Errorf("maintenance cleanup: unknown manifest owner %q", request.Manifest.Owner)
	}
	if request.Manifest.Checkpoint != maintenanceCleanupCheckpoint {
		return plan, fmt.Errorf("maintenance cleanup: checkpoint must be %q", maintenanceCleanupCheckpoint)
	}
	if request.TransactionID == "" {
		return plan, fmt.Errorf("maintenance cleanup: transaction id is required")
	}

	targets := append([]maintenanceCleanupTarget(nil), request.Manifest.Targets...)
	sort.Slice(targets, func(i, j int) bool { return targets[i].RelativePath < targets[j].RelativePath })
	mutation := maintenanceMutationPlan{
		SchemaVersion:   maintenanceMutationSchemaVersion,
		Operation:       "data-clean",
		TransactionID:   request.TransactionID,
		SourceRoot:      request.RepositoryRoot,
		DestinationRoot: request.DataRoot,
		Checkpoint:      maintenanceCleanupCheckpoint,
		Recovery:        "Preserve the named transaction journal and run `aether resume`, or rerun data-clean with a fresh exact manifest after a confirmed rollback.",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot:    request.RepositoryRoot,
			LifecycleDataRoot: request.DataRoot,
		},
		Rename: request.Rename,
		Fault:  request.Fault,
	}
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if target.Owner != maintenanceCleanupOwner {
			return plan, fmt.Errorf("maintenance cleanup: target %q has unknown owner %q", target.RelativePath, target.Owner)
		}
		if strings.TrimSpace(target.Digest) == "" || target.Digest == lifecycleTransactionMissingDigest {
			return plan, fmt.Errorf("maintenance cleanup: target %q requires an existing baseline digest", target.RelativePath)
		}
		clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(target.RelativePath)))
		if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean != filepath.FromSlash(target.RelativePath) {
			return plan, fmt.Errorf("maintenance cleanup: target %q must be a canonical contained relative path", target.RelativePath)
		}
		if _, duplicate := seen[clean]; duplicate {
			return plan, fmt.Errorf("maintenance cleanup: duplicate target %q", clean)
		}
		seen[clean] = struct{}{}
		path := filepath.Join(request.DataRoot, clean)
		info, err := os.Lstat(path)
		if err != nil {
			return plan, fmt.Errorf("maintenance cleanup: inspect target %q: %w", clean, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return plan, fmt.Errorf("maintenance cleanup: target %q must be a regular non-symlink file", clean)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return plan, fmt.Errorf("maintenance cleanup: read target %q: %w", clean, err)
		}
		if got := lifecycleDigest(content); got != target.Digest {
			return plan, fmt.Errorf("maintenance cleanup: baseline changed for %q", clean)
		}
		mutation.Targets = append(mutation.Targets, maintenanceMutationTarget{
			Root:           lifecycleTransactionRootData,
			RelativeTarget: clean,
			Source:         "exact cleanup manifest owned by " + maintenanceCleanupOwner,
			Action:         lifecycleTransactionRemove,
			ExpectedDigest: target.Digest,
			Managed:        true,
		})
	}
	preview, err := prepareMaintenanceMutation(mutation)
	if err != nil {
		return plan, err
	}
	plan.Preview = preview
	plan.mutation = mutation
	return plan, nil
}

func commitMaintenanceCleanup(plan maintenanceCleanupPlan) (maintenanceMutationResult, error) {
	prepared, err := prepareMaintenanceCleanup(plan.Request)
	if err != nil {
		return maintenanceMutationResult{
			SchemaVersion: maintenanceMutationSchemaVersion,
			Operation:     "data-clean",
			Preview:       plan.Preview,
			StateEffect:   colony.LifecycleStateEffectNone,
			Recovery:      plan.mutation.Recovery,
		}, err
	}
	return commitMaintenanceMutation(prepared.mutation)
}

var dataCleanCmd = &cobra.Command{
	Use:   "data-clean",
	Short: "Remove explicitly owned artifacts from an exact cleanup manifest",
	Args:  cobra.NoArgs,
	RunE:  runMaintenanceDataClean,
}

func runMaintenanceDataClean(cmd *cobra.Command, _ []string) error {
	if store == nil {
		return fmt.Errorf("data-clean: no store initialized")
	}
	confirm, _ := cmd.Flags().GetBool("confirm")
	manifestRel, _ := cmd.Flags().GetString("manifest")
	if strings.TrimSpace(manifestRel) == "" {
		manifestRel = maintenanceCleanupManifestRel
	}
	manifest, manifestPath, manifestBytes, exists, err := loadMaintenanceCleanupManifest(store.BasePath(), manifestRel)
	if err != nil {
		outputError(2, err.Error(), map[string]interface{}{
			"operation": "data-clean", "manifest": manifestRel,
			"state_effect": colony.LifecycleStateEffectNone,
			"recovery":     "Correct the exact owned manifest, then rerun data-clean preview.",
		})
		return nil
	}
	if exists {
		relativeManifest, _ := filepath.Rel(filepath.Clean(store.BasePath()), manifestPath)
		manifest.Targets = append(manifest.Targets, maintenanceCleanupTarget{
			RelativePath: relativeManifest,
			Owner:        maintenanceCleanupOwner,
			Digest:       lifecycleDigest(manifestBytes),
		})
	}
	repositoryRoot := filepath.Clean(repoRootFromStore(store))
	idDigest := strings.NewReplacer(":", "-", "/", "-").Replace(lifecycleDigest(manifestBytes))
	if len(idDigest) > 20 {
		idDigest = idDigest[:20]
	}
	request := maintenanceCleanupRequest{
		RepositoryRoot: repositoryRoot,
		DataRoot:       filepath.Clean(store.BasePath()),
		TransactionID:  "data-clean-" + idDigest,
		Manifest:       manifest,
	}
	plan, err := prepareMaintenanceCleanup(request)
	if err != nil {
		outputError(2, err.Error(), map[string]interface{}{
			"operation": "data-clean", "manifest": manifestRel,
			"state_effect": colony.LifecycleStateEffectNone,
			"recovery":     "Correct the exact owned manifest, then rerun data-clean preview.",
		})
		return nil
	}
	result := maintenanceMutationResult{
		SchemaVersion: maintenanceMutationSchemaVersion,
		Operation:     "data-clean",
		Preview:       plan.Preview,
		Targets:       append([]maintenanceMutationTargetPreview(nil), plan.Preview.Targets...),
		StateEffect:   colony.LifecycleStateEffectNone,
		Recovery:      plan.mutation.Recovery,
	}
	if confirm {
		result, err = commitMaintenanceCleanup(plan)
	}
	removed := 0
	if result.StateEffect == colony.LifecycleStateEffectCommitted {
		for _, target := range result.Targets {
			if target.Change == maintenanceMutationChangeRemove {
				removed++
			}
		}
	}
	payload := map[string]interface{}{
		"operation":             "data-clean",
		"scanned":               true,
		"removed":               removed,
		"dry_run":               !confirm,
		"manifest":              manifestRel,
		"preview":               result.Preview,
		"transaction":           result.Preview.TransactionID,
		"receipt":               result.Receipt,
		"state_effect":          result.StateEffect,
		"verification":          result.Verification,
		"recovery":              result.Recovery,
		"worker_debug_total":    0,
		"worker_debug_prunable": 0,
		"worker_debug_removed":  0,
	}
	if err != nil {
		outputError(3, err.Error(), payload)
		return nil
	}
	outputOK(payload)
	return nil
}

func loadMaintenanceCleanupManifest(dataRoot, relativePath string) (maintenanceCleanupManifest, string, []byte, bool, error) {
	empty := maintenanceCleanupManifest{
		SchemaVersion: maintenanceCleanupSchemaVersion,
		Owner:         maintenanceCleanupOwner,
		Checkpoint:    maintenanceCleanupCheckpoint,
		Targets:       []maintenanceCleanupTarget{},
	}
	root := filepath.Clean(dataRoot)
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(relativePath)))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean != filepath.FromSlash(relativePath) {
		return empty, "", nil, false, fmt.Errorf("data-clean: manifest must be a canonical path below the lifecycle data root")
	}
	path := filepath.Join(root, clean)
	if !pathIsWithin(root, path) {
		return empty, "", nil, false, fmt.Errorf("data-clean: manifest escapes the lifecycle data root")
	}
	if err := rejectLifecycleSymlinkTarget(root, path); err != nil {
		return empty, "", nil, false, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		encoded, _ := json.Marshal(empty)
		return empty, path, encoded, false, nil
	}
	if err != nil {
		return empty, "", nil, false, fmt.Errorf("data-clean: read manifest: %w", err)
	}
	var manifest maintenanceCleanupManifest
	if err := decodeLifecycleJSON(raw, &manifest); err != nil {
		return empty, "", nil, false, fmt.Errorf("data-clean: decode manifest: %w", err)
	}
	return manifest, path, raw, true, nil
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
	dataCleanCmd.Flags().String("manifest", maintenanceCleanupManifestRel, "Exact owned cleanup manifest below the lifecycle data root")
	backupPruneGlobalCmd.Flags().Int("cap", 50, "Maximum backups to keep")

	rootCmd.AddCommand(dataCleanCmd)
	rootCmd.AddCommand(backupPruneGlobalCmd)
	rootCmd.AddCommand(tempCleanCmd)
}
