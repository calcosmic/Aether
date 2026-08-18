package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// worktreeSafety records the answer to one question: may this worktree be
// destroyed without losing work?
type worktreeSafety struct {
	Safe                bool
	Reason              string
	DirtyFileCount      int
	UnmergedCommitCount int
	Path                string
	Branch              string
}

// worktreeDestructionSafety answers exactly one question: may this worktree
// be destroyed without losing work? It is the shared pre-destruction gate
// every destructive worktree path in Phase 187 must call before removing a
// worktree or deleting its branch.
//
// Two rules this function must obey, both stated here so a future reader
// cannot undo them by accident:
//
//  1. It MUST NOT skip colony.WorktreeOrphaned entries. scanDirtyWorktrees
//     (cmd/recover_scanner.go) skips them, which is correct for a read-only
//     health report but catastrophic for a pre-destruction gate:
//     preserveWorktree in reconcileWorktreeWave (cmd/codex_build_worktree.go)
//     marks an entry Orphaned specifically to protect it, and
//     gcOrphanedWorktrees then treats Orphaned as delete-me. Inheriting the
//     skip would blind the guard on the exact case it exists for. This
//     function therefore evaluates every entry it is given regardless of
//     Status — the caller decides which entries to check, this function
//     never filters by WorktreeOrphaned itself.
//  2. Every failure to determine an answer resolves to Safe=false.
//     Uncertainty is not permission.
func worktreeDestructionSafety(root string, entry colony.WorktreeEntry) worktreeSafety {
	result := worktreeSafety{
		Path:   entry.Path,
		Branch: entry.Branch,
	}

	// Step 1: resolve the absolute path.
	absPath := entry.Path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, entry.Path)
	}
	result.Path = absPath

	// Step 2: a missing directory holds no work. Mirrors the existing
	// short-circuit in gcOrphanedWorktrees (cmd/codex_build_worktree.go).
	if _, statErr := os.Stat(absPath); os.IsNotExist(statErr) {
		result.Safe = true
		result.Reason = "worktree path no longer exists on disk"
		return result
	}

	// Step 3: dirty check. A guard that cannot see must refuse, never assume
	// safe.
	statusCtx, statusCancel := context.WithTimeout(context.Background(), GitTimeout)
	statusOut, statusErr := exec.CommandContext(statusCtx, "git", "-C", absPath, "status", "--porcelain").CombinedOutput()
	statusCancel()
	if statusErr != nil {
		result.Safe = false
		result.Reason = "cannot determine whether this worktree has unsaved changes"
		return result
	}
	trimmedStatus := strings.TrimSpace(string(statusOut))
	if trimmedStatus != "" {
		lines := strings.Split(trimmedStatus, "\n")
		result.DirtyFileCount = len(lines)
		result.Safe = false
		result.Reason = fmt.Sprintf("worktree has %d uncommitted change(s)", result.DirtyFileCount)
		return result
	}

	// Step 4: unmerged-commit check. Determine the integration branch by
	// trying main and falling back to master, matching the checkout
	// fallback already used in mergePhaseWorktrees
	// (cmd/codex_build_worktree.go).
	integrationBranch := "main"
	verifyCtx, verifyCancel := context.WithTimeout(context.Background(), GitTimeout)
	if _, verifyErr := exec.CommandContext(verifyCtx, "git", "-C", root, "rev-parse", "--verify", "main").CombinedOutput(); verifyErr != nil {
		integrationBranch = "master"
	}
	verifyCancel()

	revListCtx, revListCancel := context.WithTimeout(context.Background(), GitTimeout)
	revListOut, revListErr := exec.CommandContext(revListCtx, "git", "-C", root, "rev-list", "--count",
		integrationBranch+".."+entry.Branch).CombinedOutput()
	revListCancel()
	if revListErr != nil {
		result.Safe = false
		result.Reason = "cannot determine whether this branch holds unmerged commits"
		return result
	}

	// Note on the audit's own caveat: .planning/WORKTREE-BRANCH-AUDIT-2026-07-27.md
	// records that `git rev-list --count` overstates loss for diverged
	// lineage (i.e. it counts commits that diverged rather than commits that
	// would truly be lost). That is acceptable here — this guard is
	// deliberately conservative, and over-preserving is the safe error
	// direction under D-01. Do not "fix" this into a lossy check.
	var unmergedCount int
	if _, scanErr := fmt.Sscanf(strings.TrimSpace(string(revListOut)), "%d", &unmergedCount); scanErr != nil {
		result.Safe = false
		result.Reason = "cannot determine whether this branch holds unmerged commits"
		return result
	}
	if unmergedCount > 0 {
		result.UnmergedCommitCount = unmergedCount
		result.Safe = false
		result.Reason = fmt.Sprintf("branch holds %d commit(s) not yet on %s", unmergedCount, integrationBranch)
		return result
	}

	// Step 5: clean and fully merged.
	result.Safe = true
	result.Reason = "clean and fully merged"
	return result
}
