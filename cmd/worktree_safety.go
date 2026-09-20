package cmd

import (
	"context"
	"fmt"
	"os"
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

	// Step 2.5: confirm absPath is the TOP LEVEL of its own git worktree
	// before trusting any status answer computed from it. `git -C <path>
	// status` does not fail when <path> is a plain directory that is not
	// itself a worktree — git walks UP the directory tree and answers about
	// the first enclosing repository it finds instead. Without this check, a
	// stale directory whose worktree registration is gone (crash during
	// `git worktree add`, a manually deleted `.git/worktrees/...` entry, a
	// restored backup) would silently report the ENCLOSING repo's status —
	// "clean" if the root happens to be clean — rather than refusing (CR-02).
	topCtx, topCancel := context.WithTimeout(context.Background(), GitTimeout)
	topOut, topErr := readOnlyGitCommand(topCtx, absPath, "rev-parse", "--show-toplevel").Output()
	topCancel()
	if topErr != nil {
		result.Safe = false
		result.Reason = "cannot determine whether this worktree has unsaved changes"
		return result
	}
	resolvedTop, _ := filepath.EvalSymlinks(strings.TrimSpace(string(topOut)))
	resolvedAbs, _ := filepath.EvalSymlinks(absPath)
	if resolvedTop != resolvedAbs {
		result.Safe = false
		result.Reason = "this folder is no longer a separate worker workspace, so its contents cannot be checked"
		return result
	}

	// Step 3: dirty check. A guard that cannot see must refuse, never assume
	// safe.
	statusCtx, statusCancel := context.WithTimeout(context.Background(), GitTimeout)
	statusOut, statusErr := readOnlyGitCommand(statusCtx, absPath, "status", "--porcelain").CombinedOutput()
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

	// Step 3.5 / 4 / 5: nameless-branch refusal + the unmerged-commit check,
	// shared with the branch-only path (branchMergeSafety) used where no
	// worktree directory exists at all — see that function's doc comment.
	merge := branchMergeSafety(root, entry.Branch)
	result.Safe = merge.Safe
	result.Reason = merge.Reason
	result.UnmergedCommitCount = merge.UnmergedCommitCount
	return result
}

// branchMergeSafety answers a narrower question than worktreeDestructionSafety:
// ignoring any worktree directory entirely, does this branch name hold commits
// that are not yet on the integration branch? This is the check
// worktreeDestructionSafety's Step 3.5/4 already perform once a worktree path
// has passed its dirty-file check — factored out here so a caller with NO
// worktree directory to inspect (a bare branch, e.g. an "orphan branch" with
// no worktree and no state entry — recover_repair.go's repairDirtyWorktree)
// can still get the same unmerged-commit protection without a synthetic,
// nonexistent path defeating worktreeDestructionSafety's Step 2 short-circuit
// ("a missing directory holds no work" — true for a worktree, not true for a
// branch that was never checked out into one).
//
// Per worktreeDestructionSafety's own rule 2: every failure to determine an
// answer resolves to Safe=false. Uncertainty is not permission.
func branchMergeSafety(root, branch string) worktreeSafety {
	result := worktreeSafety{Branch: branch}

	// A nameless branch cannot be checked, and uncertainty is not permission.
	// Without this guard, git rev-list --count "main.." (empty right-hand
	// side) is valid git, returns 0 with exit status 0, and the unmerged-
	// commit check below falls through to Safe=true — CR-01.
	if strings.TrimSpace(branch) == "" {
		result.Safe = false
		result.Reason = "no branch name was given, so its work cannot be checked"
		return result
	}

	// Determine the integration branch by trying main and falling back to
	// master, matching the checkout fallback already used in
	// mergePhaseWorktrees (cmd/codex_build_worktree.go).
	integrationBranch := "main"
	verifyCtx, verifyCancel := context.WithTimeout(context.Background(), GitTimeout)
	if _, verifyErr := readOnlyGitCommand(verifyCtx, root, "rev-parse", "--verify", "main").CombinedOutput(); verifyErr != nil {
		integrationBranch = "master"
	}
	verifyCancel()

	// "--" separates the revision range from any option flags, so a branch
	// name beginning with "-" cannot be misread by git as a flag (CR-01).
	revListCtx, revListCancel := context.WithTimeout(context.Background(), GitTimeout)
	revListOut, revListErr := readOnlyGitCommand(revListCtx, root, "rev-list", "--count",
		integrationBranch+".."+branch, "--").CombinedOutput()
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

	result.Safe = true
	result.Reason = "clean and fully merged"
	return result
}

// worktreeDestructionCategory classifies the named, operator-invoked
// destruction command the same way recover_repair.go classifies
// "dirty_worktree" — destructive, requires an explicit --force (or
// equivalent) flag, never runs implicitly.
const worktreeDestructionCategory = "worktree_destruction"

// isDestructiveWorktreeAction reports whether the given action name is the
// worktree-destruction category. Sibling to isDestructiveCategory
// (cmd/recover_repair.go) so plan 03's named destruction command can
// classify itself the same way recover classifies dirty_worktree.
func isDestructiveWorktreeAction(action string) bool {
	return action == worktreeDestructionCategory
}

// preserveWorktreeWork makes unsafe work recoverable without discarding it.
// Per D-01 ("preserves the work and continues — it does not delete, and it
// does not stop and ask"), this function never blocks and never calls a
// destructive git command.
//
// It must never call removeGitWorktree, `git branch -D`, or
// `git worktree remove`. It must never read from stdin — confirmRepair's
// interactive prompt shape (cmd/recover_repair.go) is explicitly rejected by
// D-01 because it blocks unattended runs.
func preserveWorktreeWork(root string, entry colony.WorktreeEntry, safety worktreeSafety) (preserved bool, detail string, err error) {
	// A clean, safe-to-destroy worktree has nothing to preserve. This
	// function must never act on it.
	if safety.Safe {
		return false, "", nil
	}

	absPath := entry.Path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, entry.Path)
	}

	switch {
	case safety.DirtyFileCount > 0:
		// Stash rather than discard — the exact command already used and
		// tested at cmd/recover_repair.go's repairDirtyWorktree.
		stashCtx, stashCancel := context.WithTimeout(context.Background(), GitTimeout)
		stashOut, stashErr := worktreeGitCommand(stashCtx, absPath, false, "stash", "--include-untracked").CombinedOutput()
		stashCancel()
		if stashErr != nil {
			return false, "", fmt.Errorf("stash worktree changes: %w: %s", stashErr, strings.TrimSpace(string(stashOut)))
		}
		detail = fmt.Sprintf("%d uncommitted change(s) stashed on branch %s", safety.DirtyFileCount, entry.Branch)
		return true, detail, nil

	case safety.UnmergedCommitCount > 0:
		// Nothing to stash — the commits are already durable on the branch.
		// The preservation action here is the decision NOT to run
		// `branch -D`.
		detail = fmt.Sprintf("%d commit(s) kept on branch %s (not yet merged)", safety.UnmergedCommitCount, entry.Branch)
		return true, detail, nil

	default:
		// The safety reason was an inability to determine state — nothing
		// was actually stashed or otherwise saved here, only decided against
		// deleting. Reporting preserved=true here (CR-04) would tell a
		// caller like worktree-reap's --force --include-unmerged path that
		// it is safe to proceed with destruction because "the work was
		// saved first" — but nothing was examined, let alone saved. Return
		// preserved=false so any caller about to destroy on top of this
		// must refuse instead of reading "no error" as "saved".
		detail = fmt.Sprintf("could not check the work on branch %s, so nothing could be saved", entry.Branch)
		return false, detail, nil
	}
}

// describeWorktreePreservation returns a single plain-English line for a
// non-technical reader, naming the branch, saying the work was kept rather
// than deleted, and saying how to get it back. Per D-02 and this repo's
// non-technical-owner rule in CLAUDE.md, this string must never use the
// words "orphaned", "GC", "ratchet", or "residue" — those are repo-invented
// terms the owner does not know.
func describeWorktreePreservation(safety worktreeSafety, detail string) string {
	branch := safety.Branch
	if branch == "" {
		branch = "(unknown branch)"
	}
	if detail == "" {
		detail = safety.Reason
	}
	return fmt.Sprintf("Kept the work on branch %s instead of deleting it (%s). "+
		"Inspect it with `aether maintenance recovery-inspect` (State effect: none). "+
		"To restore runnable lifecycle state, run `aether resume`.", branch, detail)
}

// reportWorktreePreservation writes the plain-English preservation line to
// stderr, matching the existing precedent at cmd/init_cmd.go. It must write
// unconditionally on every call — D-02 states "reported in plain language on
// every occurrence" and "silent handling is prohibited on this path".
//
// A caller that swallows this report reintroduces the defect that let ten
// branches strand unnoticed between May and July 2026
// (.planning/WORKTREE-BRANCH-AUDIT-2026-07-27.md). Do not wrap this call in
// a conditional that can suppress it.
func reportWorktreePreservation(safety worktreeSafety, detail string) {
	fmt.Fprintf(os.Stderr, "%s\n", describeWorktreePreservation(safety, detail))
}

// scanUnrecordedWorktrees finds git worktrees living on disk under
// worktreesDir that are NOT among knownPaths (paths already tracked in
// COLONY_STATE.json), and evaluates each with worktreeDestructionSafety.
//
// This exists for exactly one crash window: a worktree created by `git
// worktree add` but killed before its state entry was appended
// (cmd/codex_build_worktree.go's appendBuildWorktreeEntry). Such a worktree
// is invisible to gcOrphanedWorktrees, which only ever iterates
// state.Worktrees — it counts as "nothing preserved" and a caller that
// wipes worktreesDir on that basis (cmd/init_cmd.go) destroys it with zero
// git-level check, dirty or not. Reading the directory listing directly,
// independent of what state records, is the only way to see it.
//
// root is passed through unchanged to worktreeDestructionSafety, which
// itself resolves entry.Path relative to root when not absolute — since the
// paths returned here come from os.ReadDir(worktreesDir) they are already
// absolute (worktreesDir is expected to be absolute), so entry.Path is set
// absolute and worktreeDestructionSafety's relative-join branch is not
// taken.
func scanUnrecordedWorktrees(root, worktreesDir string, knownPaths map[string]bool) []worktreeSafety {
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		// No directory, or unreadable -- nothing to find. A caller about to
		// RemoveAll a directory that cannot even be listed has nothing this
		// function can add.
		return nil
	}

	var results []worktreeSafety
	for _, dirEntry := range entries {
		if !dirEntry.IsDir() {
			continue
		}
		candidatePath := filepath.Join(worktreesDir, dirEntry.Name())

		resolvedCandidate, _ := filepath.EvalSymlinks(candidatePath)
		if resolvedCandidate == "" {
			resolvedCandidate = candidatePath
		}
		found := knownPaths[candidatePath]
		if !found {
			for kp := range knownPaths {
				resolvedKnown, _ := filepath.EvalSymlinks(kp)
				if resolvedKnown == resolvedCandidate {
					found = true
					break
				}
			}
		}
		if found {
			continue
		}

		// Confirm this directory is actually the TOP LEVEL of its own git
		// worktree, not merely a plain subdirectory that git resolves
		// upward through to some enclosing repository (e.g. the Aether root
		// itself, if worktreesDir's parent happens to sit inside it). `git
		// -C <path> rev-parse` does not fail for a non-worktree directory --
		// it walks UP until it finds an enclosing .git, exactly the
		// footgun worktreeDestructionSafety's own Step 2.5 guards against.
		// Applying the identical toplevel check here, before ever
		// constructing a colony.WorktreeEntry, prevents a stray empty
		// directory from being misreported as "this folder is no longer a
		// separate worker workspace" (which reads as an unsafe verdict on
		// something that was never a worktree to begin with).
		topCtx, topCancel := context.WithTimeout(context.Background(), GitTimeout)
		topOut, topErr := readOnlyGitCommand(topCtx, candidatePath, "rev-parse", "--show-toplevel").Output()
		topCancel()
		if topErr != nil {
			// Not inside any git repository at all -- definitely not a
			// worktree.
			continue
		}
		resolvedTop, _ := filepath.EvalSymlinks(strings.TrimSpace(string(topOut)))
		resolvedCand, _ := filepath.EvalSymlinks(candidatePath)
		if resolvedTop != resolvedCand {
			// This directory resolves to some ENCLOSING repository rather
			// than being a worktree root itself -- not this function's
			// concern; a plain non-worktree directory under worktreesDir
			// holds no worktree-shaped work to lose.
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		branchOut, branchErr := readOnlyGitCommand(ctx, candidatePath, "rev-parse", "--abbrev-ref", "HEAD").Output()
		cancel()
		if branchErr != nil {
			// Confirmed a worktree root but branch name unreadable (e.g.
			// detached HEAD edge case) -- still not this function's concern
			// to guess; skip rather than fabricate a branch name.
			continue
		}
		branch := strings.TrimSpace(string(branchOut))

		entry := colony.WorktreeEntry{
			Path:   candidatePath,
			Branch: branch,
		}
		safety := worktreeDestructionSafety(root, entry)
		results = append(results, safety)
	}
	return results
}
