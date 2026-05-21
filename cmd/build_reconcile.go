package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var buildReconcileCmd = &cobra.Command{
	Use:   "build-reconcile",
	Short: "Discover unrecorded worker changes and create a synthetic build packet",
	Long: `Scans the git working tree for changes not recorded in the current build
packet and creates a synthetic packet so interrupted builds do not lose work.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		result, err := runBuildReconcile(dryRun)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, "")
		return nil
	},
}

func init() {
	buildReconcileCmd.Flags().Bool("dry-run", false, "Preview changes without writing")
	rootCmd.AddCommand(buildReconcileCmd)
}

// runBuildReconcile scans for unrecorded changes and optionally writes a synthetic packet.
func runBuildReconcile(dryRun bool) (map[string]interface{}, error) {
	state, err := loadActiveColonyState()
	if err != nil {
		return nil, fmt.Errorf("load colony state: %w", err)
	}

	currentPhase := state.CurrentPhase
	if currentPhase == 0 {
		return nil, fmt.Errorf("no active phase")
	}

	// Run git status --short
	root := resolveAetherRoot()
	gitCmd := exec.Command("git", "-C", root, "status", "--short")
	out, err := gitCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}

	// Parse changed files
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var added, modified, deleted []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < 3 {
			continue
		}
		statusCode := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[2:])
		if path == "" {
			continue
		}
		// Handle renames
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			if len(parts) == 2 {
				added = append(added, strings.TrimSpace(parts[1]))
				deleted = append(deleted, strings.TrimSpace(parts[0]))
				continue
			}
		}
		switch {
		case strings.Contains(statusCode, "D"):
			deleted = append(deleted, path)
		case strings.Contains(statusCode, "A"):
			added = append(added, path)
		default:
			modified = append(modified, path)
		}
	}

	// Load existing claims to compare
	existingClaims, _ := loadCodexBuildClaims()
	recorded := make(map[string]bool)
	for _, f := range existingClaims.FilesCreated {
		recorded[f] = true
	}
	for _, f := range existingClaims.FilesModified {
		recorded[f] = true
	}
	for _, f := range existingClaims.TestsWritten {
		recorded[f] = true
	}

	// Filter out already-recorded files
	var newAdded, newModified, newDeleted []string
	for _, f := range added {
		if !recorded[f] {
			newAdded = append(newAdded, f)
		}
	}
	for _, f := range modified {
		if !recorded[f] {
			newModified = append(newModified, f)
		}
	}
	for _, f := range deleted {
		if !recorded[f] {
			newDeleted = append(newDeleted, f)
		}
	}

	totalNew := len(newAdded) + len(newModified) + len(newDeleted)

	result := map[string]interface{}{
		"phase":          currentPhase,
		"dry_run":        dryRun,
		"total_changes":  totalNew,
		"added":          newAdded,
		"modified":       newModified,
		"deleted":        newDeleted,
		"synthetic":      true,
		"recommendation": "",
	}

	if totalNew == 0 {
		result["recommendation"] = "No unrecorded changes found."
		return result, nil
	}

	result["recommendation"] = fmt.Sprintf("Recorded %d unrecorded change(s) into synthetic build packet.", totalNew)

	if dryRun {
		result["recommendation"] = fmt.Sprintf("Would record %d unrecorded change(s). Run without --dry-run to apply.", totalNew)
		return result, nil
	}

	// Create or update synthetic build claims
	syntheticClaims := codexBuildClaims{
		BuildPhase:    currentPhase,
		FilesCreated:  append(existingClaims.FilesCreated, newAdded...),
		FilesModified: append(existingClaims.FilesModified, newModified...),
		TestsWritten:  append(existingClaims.TestsWritten, newDeleted...), // deleted tracked as tests_written for now
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}

	// Save claims
	if err := store.SaveJSON("last-build-claims.json", syntheticClaims); err != nil {
		return nil, fmt.Errorf("save build claims: %w", err)
	}

	// Save synthetic packet marker
	packet := map[string]interface{}{
		"phase":       currentPhase,
		"synthetic":   true,
		"reconciled":  true,
		"timestamp":   syntheticClaims.Timestamp,
		"files_added": newAdded,
		"files_modified": newModified,
		"files_deleted": newDeleted,
	}
	packetPath := filepath.Join(store.BasePath(), fmt.Sprintf("build-reconcile-phase-%d.json", currentPhase))
	packetData, _ := json.MarshalIndent(packet, "", "  ")
	if err := os.WriteFile(packetPath, append(packetData, '\n'), 0644); err != nil {
		return nil, fmt.Errorf("write reconcile packet: %w", err)
	}

	return result, nil
}
