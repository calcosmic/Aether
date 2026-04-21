package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

// worktreeBaseDir is the base directory for managed worktrees.
const worktreeBaseDir = ".aether/worktrees"

// agentBranchRe matches agent-track branch names: phase-N/caste-name
// where N is a positive integer (1-9 followed by any digits) and caste-name
// is lowercase alphanumeric with hyphens.
var agentBranchRe = regexp.MustCompile(`^phase-[1-9]\d*/[a-z0-9-]+$`)

// humanBranchPrefixes lists allowed prefixes for human-track branch names.
var humanBranchPrefixes = []string{"feature/", "fix/", "experiment/", "colony/"}

// ---------------------------------------------------------------------------
// validateBranchName
// ---------------------------------------------------------------------------

// validateBranchName checks that a branch name follows the two-track naming convention.
// Agent track: phase-N/caste-name (e.g., "phase-2/builder-1")
// Human track: prefix/description where prefix is one of the allowed values.
func validateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("invalid branch name \"\": branch name is required")
	}

	// Reject path traversal
	if strings.Contains(name, "..") {
		return fmt.Errorf("invalid branch name %q: path traversal detected", name)
	}

	// Agent track: phase-N/caste-name
	if agentBranchRe.MatchString(name) {
		return nil
	}

	// Human track: prefix/description
	for _, prefix := range humanBranchPrefixes {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			return nil
		}
	}

	return fmt.Errorf("invalid branch name %q: must match phase-N/name or use prefix/name (prefix: feature, fix, experiment, colony)", name)
}

// ---------------------------------------------------------------------------
// sanitizeBranchPath
// ---------------------------------------------------------------------------

// sanitizeBranchPath converts a branch name to a safe filesystem path by
// replacing slashes with hyphens. This prevents directory structure issues
// with branch names like "phase-2/builder-1".
func sanitizeBranchPath(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}

// ---------------------------------------------------------------------------
// generateWorktreeID
// ---------------------------------------------------------------------------

// generateWorktreeID produces a unique worktree identifier.
// Format: wt_<unix_timestamp>_<random_hex>
func generateWorktreeID() string {
	rnd := make([]byte, 4)
	rand.Read(rnd)
	return fmt.Sprintf("wt_%d_%s", time.Now().Unix(), hex.EncodeToString(rnd))
}

// ---------------------------------------------------------------------------
// isWorktreeOrphaned
// ---------------------------------------------------------------------------

// isWorktreeOrphaned checks whether a worktree's last commit time exceeds the
// given threshold duration from now. A zero commitAt is treated as orphaned
// (no commits ever made).
func isWorktreeOrphaned(commitAt time.Time, threshold time.Duration) bool {
	if commitAt.IsZero() {
		return true
	}
	return time.Since(commitAt) > threshold
}

// ---------------------------------------------------------------------------
// getLastCommitTime
// ---------------------------------------------------------------------------

// getLastCommitTime retrieves the timestamp of the most recent commit in a worktree.
func getLastCommitTime(worktreePath string) (time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", worktreePath,
		"log", "-1", "--format=%ct").Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("git log: %w", err)
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse commit timestamp: %w", err)
	}
	return time.Unix(ts, 0), nil
}

// ---------------------------------------------------------------------------
// worktree-allocate command
// ---------------------------------------------------------------------------

var worktreeAllocateCmd = &cobra.Command{
	Use:   "worktree-allocate",
	Short: "Create worktree with enforced naming and state tracking",
	Long: "Creates a git worktree with branch name validation and lifecycle tracking. " +
		"Use --agent/--phase for agent-track names (phase-N/name) or --branch for human-track names.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		agentName, _ := cmd.Flags().GetString("agent")
		phaseNum, _ := cmd.Flags().GetInt("phase")
		branchName, _ := cmd.Flags().GetString("branch")

		// Determine branch name
		var branch string
		if agentName != "" && phaseNum > 0 {
			branch = fmt.Sprintf("phase-%d/%s", phaseNum, agentName)
		} else if branchName != "" {
			branch = branchName
		} else {
			outputError(1, "either --agent/--phase or --branch is required", nil)
			return nil
		}

		// Validate branch name
		if err := validateBranchName(branch); err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		// Check for duplicate in COLONY_STATE.json
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			// If state file doesn't exist, that's ok for allocate
			if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "read") {
				outputError(2, fmt.Sprintf("failed to load colony state: %v", err), nil)
				return nil
			}
		}

		// Check for existing non-merged worktree with same branch
		for _, wt := range state.Worktrees {
			if wt.Branch == branch && wt.Status != colony.WorktreeMerged {
				outputError(1, fmt.Sprintf("worktree %q already tracked with status %q", branch, wt.Status), nil)
				return nil
			}
		}

		// Create worktree directory path
		sanitized := sanitizeBranchPath(branch)
		worktreePath := worktreeBaseDir + "/" + sanitized

		// Create parent directory
		if err := os.MkdirAll(worktreeBaseDir, 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create worktree directory: %v", err), nil)
			return nil
		}

		// Create git worktree: try creating new branch first, then reuse existing
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()

		addCmd := exec.CommandContext(ctx, "git", "worktree", "add", "-b", branch, worktreePath, "HEAD")
		if out, err := addCmd.CombinedOutput(); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputError(2, fmt.Sprintf("git worktree add timed out after %v", GitTimeout), nil)
				return nil
			}
			// Branch may already exist; try reusing it
			reuseCmd := exec.CommandContext(ctx, "git", "worktree", "add", worktreePath, branch)
			if out2, err2 := reuseCmd.CombinedOutput(); err2 != nil {
				outputError(2, fmt.Sprintf("failed to create worktree: %v: %s (also tried reusing branch: %v: %s)",
					err, string(out), err2, string(out2)), nil)
				return nil
			}
		}

		// Register in COLONY_STATE.json
		now := time.Now().UTC().Format(time.RFC3339)
		entry := colony.WorktreeEntry{
			ID:        generateWorktreeID(),
			Branch:    branch,
			Path:      worktreePath,
			Status:    colony.WorktreeAllocated,
			Phase:     phaseNum,
			Agent:     agentName,
			CreatedAt: now,
			UpdatedAt: now,
		}

		state.Worktrees = append(state.Worktrees, entry)

		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			// Rollback: remove the worktree
			rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), GitTimeout)
			defer rollbackCancel()
			exec.CommandContext(rollbackCtx, "git", "worktree", "remove", worktreePath, "--force").Run()
			exec.CommandContext(rollbackCtx, "git", "worktree", "prune").Run()
			outputError(2, fmt.Sprintf("failed to save colony state: %v (worktree rolled back)", err), nil)
			return nil
		}

		// Audit log
		auditEntry := map[string]interface{}{
			"action":    "worktree-allocate",
			"branch":    branch,
			"path":      worktreePath,
			"id":        entry.ID,
			"timestamp": now,
		}
		store.AppendJSONL("state-changelog.jsonl", auditEntry)

		outputOK(map[string]interface{}{
			"ok":      true,
			"branch":  branch,
			"path":    worktreePath,
			"id":      entry.ID,
			"status":  string(entry.Status),
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// worktree-list command
// ---------------------------------------------------------------------------

var worktreeListCmd = &cobra.Command{
	Use:   "worktree-list",
	Short: "List tracked worktrees with lifecycle status",
	Long: "Shows all tracked worktrees from COLONY_STATE.json with on-disk cross-reference. " +
		"Use --status to filter by a specific status.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		statusFilter, _ := cmd.Flags().GetString("status")

		// Load colony state
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			// No state file means no worktrees
			outputOK(map[string]interface{}{
				"worktrees": []interface{}{},
			})
			return nil
		}

		// Get on-disk worktrees via git
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()
		gitOut, err := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain").Output()
		onDiskPaths := map[string]bool{}
		if err == nil {
			for _, p := range parseWorktreePaths(string(gitOut)) {
				onDiskPaths[p] = true
			}
		}

		// Build output with on_disk cross-reference
		type worktreeOutput struct {
			colony.WorktreeEntry
			OnDisk bool `json:"on_disk"`
		}

		var result []worktreeOutput
		for _, wt := range state.Worktrees {
			// Apply status filter
			if statusFilter != "" && string(wt.Status) != statusFilter {
				continue
			}

			// Check if on disk (check both absolute and relative path)
			onDisk := false
			if _, err := os.Stat(wt.Path); err == nil {
				onDisk = true
			} else if onDiskPaths[wt.Path] {
				onDisk = true
			}

			result = append(result, worktreeOutput{
				WorktreeEntry: wt,
				OnDisk:        onDisk,
			})
		}

		if result == nil {
			result = []worktreeOutput{}
		}

		outputOK(map[string]interface{}{
			"worktrees": result,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// worktree-orphan-scan command
// ---------------------------------------------------------------------------

var worktreeOrphanScanCmd = &cobra.Command{
	Use:   "worktree-orphan-scan",
	Short: "Detect orphaned or stale worktrees",
	Long: "Scans tracked worktrees for staleness (no commits in threshold hours) " +
		"and detects untracked worktrees on disk. Updates status to 'orphaned' for stale worktrees.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		thresholdHours, _ := cmd.Flags().GetInt("threshold")
		if thresholdHours <= 0 {
			thresholdHours = 48
		}
		threshold := time.Duration(thresholdHours) * time.Hour

		// Load colony state
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputOK(map[string]interface{}{
				"orphaned":  []interface{}{},
				"stale":     []interface{}{},
				"untracked": []interface{}{},
				"threshold": thresholdHours,
			})
			return nil
		}

		// Get on-disk worktrees
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()
		gitOut, err := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain").Output()
		onDiskPaths := map[string]bool{}
		if err == nil {
			for _, p := range parseWorktreePaths(string(gitOut)) {
				onDiskPaths[p] = true
			}
		}

		now := time.Now().UTC().Format(time.RFC3339)
		var orphaned []colony.WorktreeEntry
		var stale []colony.WorktreeEntry
		stateChanged := false

		// Scan tracked worktrees
		for i, wt := range state.Worktrees {
			// Skip already-merged worktrees
			if wt.Status == colony.WorktreeMerged {
				continue
			}

			// Check if on disk
			onDisk := false
			if _, err := os.Stat(wt.Path); err == nil {
				onDisk = true
			}

			if !onDisk {
				// Worktree in state but not on disk = stale state entry
				stale = append(stale, wt)
				continue
			}

			// Check last commit time
			var lastCommit time.Time
			commitTime, err := getLastCommitTime(wt.Path)
			if err != nil {
				// No commits or git error; use creation time as fallback
				createdAt, parseErr := time.Parse(time.RFC3339, wt.CreatedAt)
				if parseErr == nil {
					lastCommit = createdAt
				} else {
					lastCommit = time.Time{} // treat as orphaned
				}
			} else {
				lastCommit = commitTime
				state.Worktrees[i].LastCommitAt = commitTime.UTC().Format(time.RFC3339)
			}

			if isWorktreeOrphaned(lastCommit, threshold) {
				state.Worktrees[i].Status = colony.WorktreeOrphaned
				state.Worktrees[i].UpdatedAt = now
				orphaned = append(orphaned, wt)
				stateChanged = true

				// Audit log
				store.AppendJSONL("state-changelog.jsonl", map[string]interface{}{
					"action":    "worktree-orphaned",
					"branch":    wt.Branch,
					"path":      wt.Path,
					"id":        wt.ID,
					"timestamp": now,
				})
			}
		}

		// Find untracked worktrees (on disk but not in state)
		trackedPaths := map[string]bool{}
		for _, wt := range state.Worktrees {
			trackedPaths[wt.Path] = true
		}

		var untracked []map[string]interface{}
		for path := range onDiskPaths {
			if !trackedPaths[path] {
				untracked = append(untracked, map[string]interface{}{
					"path": path,
				})
			}
		}

		// Save state if any worktrees were marked as orphaned
		if stateChanged {
			if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
				outputError(2, fmt.Sprintf("failed to save colony state: %v", err), nil)
				return nil
			}
		}

		if orphaned == nil {
			orphaned = []colony.WorktreeEntry{}
		}
		if stale == nil {
			stale = []colony.WorktreeEntry{}
		}
		if untracked == nil {
			untracked = []map[string]interface{}{}
		}

		// Detect orphaned branches (agent-track branches with no worktree)
		orphanBranches, _ := reportOrphanBranches()
		if orphanBranches == nil {
			orphanBranches = []map[string]interface{}{}
		}

		outputOK(map[string]interface{}{
			"orphaned":        orphaned,
			"stale":           stale,
			"untracked":       untracked,
			"orphan_branches": orphanBranches,
			"threshold":       thresholdHours,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// reportOrphanBranches
// ---------------------------------------------------------------------------

// reportOrphanBranches detects git branches matching the agent-track pattern
// (phase-N/name) that have no corresponding worktree on disk. Returns a list
// of orphaned branches with their age for user reporting.
func reportOrphanBranches() ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()

	root := storage.ResolveAetherRoot(context.Background())

	// List all branches
	out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "-a", "--format=%(refname:short) %(committerdate:iso8601)").Output()
	if err != nil {
		return nil, fmt.Errorf("git branch: %w", err)
	}

	// Get on-disk worktrees
	wtOut, err := exec.CommandContext(ctx, "git", "-C", root, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}
	onDiskPaths := map[string]bool{}
	for _, p := range parseWorktreePaths(string(wtOut)) {
		onDiskPaths[p] = true
	}

	// Get tracked worktree branches from state
	var state colony.ColonyState
	trackedBranches := map[string]bool{}
	if err := store.LoadJSON("COLONY_STATE.json", &state); err == nil {
		for _, wt := range state.Worktrees {
			trackedBranches[wt.Branch] = true
		}
	}

	var orphaned []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 1 {
			continue
		}
		branch := strings.TrimSpace(parts[0])
		// Skip remote branches
		if strings.HasPrefix(branch, "remotes/") {
			continue
		}
		// Only check agent-track branches
		if !agentBranchRe.MatchString(branch) {
			continue
		}
		// Skip if tracked in state
		if trackedBranches[branch] {
			continue
		}
		// Check if branch has a worktree
		branchHasWorktree := false
		for path := range onDiskPaths {
			// Worktree path contains sanitized branch name
			sanitized := sanitizeBranchPath(branch)
			if strings.Contains(path, sanitized) {
				branchHasWorktree = true
				break
			}
		}
		if branchHasWorktree {
			continue
		}

		// Calculate age from committer date
		age := "unknown"
		if len(parts) == 2 {
			if t, err := time.Parse("2006-01-02 15:04:05 -0700", strings.TrimSpace(parts[1])); err == nil {
				days := int(time.Since(t).Hours() / 24)
				if days < 1 {
					age = "< 1 day"
				} else {
					age = fmt.Sprintf("%d day(s)", days)
				}
			}
		}

		orphaned = append(orphaned, map[string]interface{}{
			"branch":       branch,
			"age":          age,
			"cleanup_cmd":  fmt.Sprintf("git branch -D %s", branch),
		})
	}

	return orphaned, nil
}

// ---------------------------------------------------------------------------
// createBlocker: reusable helper for creating blocker flags
// ---------------------------------------------------------------------------

// createBlocker creates a blocker flag entry in pending-decisions.json.
// This is used by merge-back to record gate failures.
func createBlocker(store *storage.Store, description string, source string) error {
	var ff colony.FlagsFile
	if err := store.LoadJSON("pending-decisions.json", &ff); err != nil {
		if err2 := store.LoadJSON("flags.json", &ff); err2 != nil {
			ff = colony.FlagsFile{Decisions: []colony.FlagEntry{}}
		}
	}
	if ff.Decisions == nil {
		ff.Decisions = []colony.FlagEntry{}
	}
	ff.Decisions = append(ff.Decisions, colony.FlagEntry{
		ID:          generateFlagID(),
		Type:        "blocker",
		Description: description,
		Source:      source,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		Resolved:    false,
	})
	return store.SaveJSON("pending-decisions.json", ff)
}

// ---------------------------------------------------------------------------
// checkClashesForWorktree: detects file conflicts across worktrees
// ---------------------------------------------------------------------------

// checkClashesForWorktree returns a list of file paths that are modified in the
// given worktree AND in at least one other worktree. It reuses parseWorktreePaths
// from cmd/clash.go (same package).
// entryPath must be an absolute path to the worktree directory.
func checkClashesForWorktree(entryPath string, branch string) ([]string, error) {
	// Derive repo root from the worktree path using git rev-parse
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()

	repoRootOut, err := exec.CommandContext(ctx, "git", "-C", entryPath, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, fmt.Errorf("git rev-parse --show-toplevel: %w", err)
	}
	repoRoot := strings.TrimSpace(string(repoRootOut))

	// Get all worktree paths from the repo root
	out, err := exec.CommandContext(ctx, "git", "-C", repoRoot, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}
	allPaths := parseWorktreePaths(string(out))

	// Determine the base branch (main or master)
	baseBranch := "main"
	if err := exec.CommandContext(ctx, "git", "-C", repoRoot, "rev-parse", "--verify", baseBranch).Run(); err != nil {
		baseBranch = "master"
	}

	// Get files modified in the target branch vs base branch
	diffCtx, diffCancel := context.WithTimeout(context.Background(), GitTimeout)
	defer diffCancel()
	diffOut, err := exec.CommandContext(diffCtx, "git", "-C", repoRoot, "diff", "--name-only", baseBranch+".."+branch).Output()
	if err != nil {
		// No diff or no commits -- nothing to clash with
		return nil, nil
	}
	modifiedFiles := strings.Split(strings.TrimSpace(string(diffOut)), "\n")
	if len(modifiedFiles) == 0 || (len(modifiedFiles) == 1 && modifiedFiles[0] == "") {
		return nil, nil
	}

	// For each modified file, check if any OTHER worktree also modifies it
	var clashes []string
	// Resolve entryPath to its real path for reliable comparison (macOS /var -> /private/var)
	entryReal, err := filepath.EvalSymlinks(entryPath)
	if err != nil {
		entryReal = entryPath
	}

	for _, file := range modifiedFiles {
		if file == "" {
			continue
		}
		for _, wtPath := range allPaths {
			// Resolve and compare paths to handle macOS /var -> /private/var symlinks
			wtReal, err := filepath.EvalSymlinks(wtPath)
			if err != nil {
				wtReal = wtPath
			}
			if wtReal == entryReal {
				continue
			}

			checkCtx, checkCancel := context.WithTimeout(context.Background(), GitTimeout)
			defer checkCancel()
			// Check if the file differs from base in the other worktree
			checkOut, checkErr := exec.CommandContext(checkCtx, "git", "-C", wtPath, "diff", "--name-only", baseBranch, "--", file).Output()
			if checkErr != nil {
				continue // If we can't check, skip it
			}
			if strings.TrimSpace(string(checkOut)) != "" {
				clashes = append(clashes, file)
				break // One clash per file is enough
			}
		}
	}

	return clashes, nil
}

// ---------------------------------------------------------------------------
// worktree-merge-back command
// ---------------------------------------------------------------------------

var worktreeMergeBackCmd = &cobra.Command{
	Use:   "worktree-merge-back",
	Short: "Merge a worktree branch back to main with safety gates",
	Long: "Merges a tracked worktree branch back to the main branch. " +
		"Two gates must pass before merge: (1) go test ./... in the worktree, " +
		"(2) clash detection to prevent file conflicts. On failure, a blocker " +
		"flag is created. On success, the worktree is cleaned up automatically.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		branch := mustGetString(cmd, "branch")
		if branch == "" {
			return nil
		}

		// Step 1: Load state and find the worktree entry
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, fmt.Sprintf("failed to load colony state: %v", err), nil)
			return nil
		}

		var entry *colony.WorktreeEntry
		var entryIndex int
		for i := range state.Worktrees {
			if state.Worktrees[i].Branch == branch {
				entry = &state.Worktrees[i]
				entryIndex = i
				break
			}
		}

		if entry == nil {
			outputError(1, fmt.Sprintf("worktree %q not found in state tracking", branch), nil)
			return nil
		}

		if entry.Status == colony.WorktreeMerged {
			outputError(1, fmt.Sprintf("worktree %q already merged", branch), nil)
			return nil
		}

		// Orphaned worktrees are allowed to re-merge (orphaned -> merged transition)

		// Resolve the worktree path relative to the Aether root
		aetherRoot := os.Getenv("AETHER_ROOT")
		if aetherRoot == "" {
			aetherRoot, _ = os.Getwd()
		}
		wtAbsPath := entry.Path
		if !filepath.IsAbs(wtAbsPath) {
			wtAbsPath = filepath.Join(aetherRoot, wtAbsPath)
		}

		// Step 2: Gate 1 -- Run tests in worktree directory
		testCtx, testCancel := context.WithTimeout(context.Background(), BuildTimeout)
		defer testCancel()
		testCmd := exec.CommandContext(testCtx, "go", "test", "./...")
		testCmd.Dir = wtAbsPath // CRITICAL: run in worktree directory (absolute path)
		testOutput, testErr := testCmd.CombinedOutput()
		if testErr != nil {
			// Tests failed -- create blocker and block merge
			blockerDesc := fmt.Sprintf("Merge blocked: tests failed for %s", branch)
			if createErr := createBlocker(store, blockerDesc, "worktree-merge-back"); createErr != nil {
				outputError(2, fmt.Sprintf("tests failed AND failed to create blocker: %v", createErr), nil)
				return nil
			}
			outputError(2, fmt.Sprintf("merge blocked: tests failed for %s: %s", branch, string(testOutput)), nil)
			return nil
		}

		// Step 3: Gate 2 -- Clash detection
		clashes, clashErr := checkClashesForWorktree(wtAbsPath, branch)
		if clashErr != nil {
			outputError(2, fmt.Sprintf("clash detection failed: %v", clashErr), nil)
			return nil
		}
		if len(clashes) > 0 {
			blockerDesc := fmt.Sprintf("Merge blocked: file clash detected for %s: %s", branch, strings.Join(clashes, ", "))
			if createErr := createBlocker(store, blockerDesc, "worktree-merge-back"); createErr != nil {
				outputError(2, fmt.Sprintf("clash detected AND failed to create blocker: %v", createErr), nil)
				return nil
			}
			outputError(2, fmt.Sprintf("merge blocked: clash detected for %s: %s", branch, strings.Join(clashes, ", ")), nil)
			return nil
		}

		// Step 4: Both gates passed -- execute merge
		gitCtx, gitCancel := context.WithTimeout(context.Background(), GitTimeout)
		defer gitCancel()

		// Checkout main in the main worktree (run from aether root)
		coOut, coErr := exec.CommandContext(gitCtx, "git", "-C", aetherRoot, "checkout", "main").CombinedOutput()
		if coErr != nil {
			// "main" may not exist; try "master" as fallback
			coOut2, coErr2 := exec.CommandContext(gitCtx, "git", "-C", aetherRoot, "checkout", "master").CombinedOutput()
			if coErr2 != nil {
				outputError(2, fmt.Sprintf("failed to checkout main branch: %v: %s (also tried master: %v: %s)",
					coErr, string(coOut), coErr2, string(coOut2)), nil)
				return nil
			}
		}

		// Merge the branch (run from aether root)
		mergeOut, mergeErr := exec.CommandContext(gitCtx, "git", "-C", aetherRoot, "merge", entry.Branch).CombinedOutput()
		if mergeErr != nil {
			blockerDesc := fmt.Sprintf("Merge failed for %s: %s", branch, string(mergeOut))
			if createErr := createBlocker(store, blockerDesc, "worktree-merge-back"); createErr != nil {
				outputError(2, fmt.Sprintf("merge failed AND failed to create blocker: %v", createErr), nil)
				return nil
			}
			outputError(2, fmt.Sprintf("merge failed for %s: %s", branch, string(mergeOut)), nil)
			return nil
		}

		// Step 5: Auto-cleanup
		pruneCtx, pruneCancel := context.WithTimeout(context.Background(), GitTimeout)
		defer pruneCancel()

		// 5a: Remove worktree directory (use absolute path)
		exec.CommandContext(pruneCtx, "git", "-C", aetherRoot, "worktree", "remove", wtAbsPath, "--force").CombinedOutput()

		// 5b: Prune stale references
		exec.CommandContext(pruneCtx, "git", "-C", aetherRoot, "worktree", "prune").Run()

		// 5c: Delete branch (tolerate "not found" -- fast-forward merges may auto-delete)
		branchDelErr := exec.CommandContext(gitCtx, "git", "-C", aetherRoot, "branch", "-d", entry.Branch).Run()
		if branchDelErr != nil {
			// Log warning but don't fail -- branch cleanup is best-effort
			// A "not found" error after fast-forward is expected
		}

		// Step 6: Update state
		now := time.Now().UTC().Format(time.RFC3339)
		state.Worktrees[entryIndex].Status = colony.WorktreeMerged
		state.Worktrees[entryIndex].UpdatedAt = now
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(2, fmt.Sprintf("failed to save colony state: %v", err), nil)
			return nil
		}

		// Step 7: Audit log
		store.AppendJSONL("state-changelog.jsonl", map[string]interface{}{
			"action":    "worktree-merge",
			"branch":    entry.Branch,
			"path":      entry.Path,
			"timestamp": now,
		})

		// Step 8: Output success
		outputOK(map[string]interface{}{
			"merged":   true,
			"branch":   entry.Branch,
			"worktree": entry.Path,
			"status":   "merged",
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// init: register commands
// ---------------------------------------------------------------------------

func init() {
	worktreeAllocateCmd.Flags().String("agent", "", "Agent name (for agent track: phase-N/name)")
	worktreeAllocateCmd.Flags().Int("phase", 0, "Phase number (for agent track)")
	worktreeAllocateCmd.Flags().String("branch", "", "Branch name (for human track)")

	worktreeListCmd.Flags().String("status", "", "Filter by status (allocated, in-progress, merged, orphaned)")

	worktreeOrphanScanCmd.Flags().Int("threshold", 48, "Staleness threshold in hours (default: 48)")

	worktreeMergeBackCmd.Flags().String("branch", "", "Branch name to merge back (required)")

	for _, c := range []*cobra.Command{
		worktreeAllocateCmd, worktreeListCmd, worktreeOrphanScanCmd, worktreeMergeBackCmd,
	} {
		rootCmd.AddCommand(c)
	}
}
