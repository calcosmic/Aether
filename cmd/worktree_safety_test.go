package cmd

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Shared fixture helper
// ---------------------------------------------------------------------------

// newWorktreeSafetyFixture builds a genuine git repository on branch main
// with one commit, then adds a genuine worktree on the given branch via
// `git worktree add`. This mirrors TestWorktreeMergeBackSuccess
// (cmd/worktree_test.go) — a worktree path that was never created on disk
// cannot exercise worktreeDestructionSafety, so every test here uses a real
// worktree.
func newWorktreeSafetyFixture(t *testing.T, branch string) (root, wtPath string, entry colony.WorktreeEntry) {
	t.Helper()

	root = t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")

	if err := os.WriteFile(root+"/README.md", []byte("initial\n"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")

	wtPath = root + "/.aether/worktrees/" + strings.ReplaceAll(branch, "/", "-")
	runGit(t, root, "worktree", "add", "-b", branch, wtPath, "HEAD")

	now := time.Now().UTC().Format(time.RFC3339)
	entry = colony.WorktreeEntry{
		ID:        "wt-safety-test",
		Branch:    branch,
		Path:      wtPath,
		Status:    colony.WorktreeInProgress,
		Phase:     1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return root, wtPath, entry
}

// ---------------------------------------------------------------------------
// worktreeDestructionSafety tests
// ---------------------------------------------------------------------------

func TestWorktreeSafetyRefusesDirtyWorktree(t *testing.T) {
	root, wtPath, entry := newWorktreeSafetyFixture(t, "phase-1/builder-dirty")

	// Write an uncommitted file into the worktree without committing.
	if err := os.WriteFile(wtPath+"/uncommitted.txt", []byte("dirty\n"), 0644); err != nil {
		t.Fatalf("write uncommitted file: %v", err)
	}

	safety := worktreeDestructionSafety(root, entry)

	if safety.Safe {
		t.Fatalf("expected Safe=false for a dirty worktree, got Safe=true (reason: %q)", safety.Reason)
	}
	if safety.DirtyFileCount < 1 {
		t.Errorf("expected DirtyFileCount >= 1, got %d", safety.DirtyFileCount)
	}
	if safety.Reason == "" {
		t.Error("expected a non-empty Reason")
	}
}

func TestWorktreeSafetyRefusesUnmergedCommits(t *testing.T) {
	root, wtPath, entry := newWorktreeSafetyFixture(t, "phase-1/builder-unmerged")

	// Commit the new file inside the worktree so the working tree is clean
	// but the branch is ahead of main. A `git status --porcelain` check
	// alone would wrongly pass this case — that is exactly why the guard
	// needs both checks.
	if err := os.WriteFile(wtPath+"/newfile.txt", []byte("committed\n"), 0644); err != nil {
		t.Fatalf("write new file: %v", err)
	}
	runGit(t, wtPath, "add", ".")
	runGit(t, wtPath, "commit", "-m", "add new file")

	safety := worktreeDestructionSafety(root, entry)

	if safety.Safe {
		t.Fatalf("expected Safe=false for a branch with unmerged commits, got Safe=true (reason: %q)", safety.Reason)
	}
	if safety.UnmergedCommitCount < 1 {
		t.Errorf("expected UnmergedCommitCount >= 1, got %d", safety.UnmergedCommitCount)
	}
}

func TestWorktreeSafetyAllowsCleanMergedWorktree(t *testing.T) {
	root, _, entry := newWorktreeSafetyFixture(t, "phase-1/builder-clean")

	// Nothing changed in the worktree — it sits at HEAD, same as main.
	safety := worktreeDestructionSafety(root, entry)

	if !safety.Safe {
		t.Fatalf("expected Safe=true for a clean, fully-merged worktree, got Safe=false (reason: %q)", safety.Reason)
	}
}

func TestWorktreeSafetyDoesNotSkipOrphanedEntries(t *testing.T) {
	// scanDirtyWorktrees (cmd/recover_scanner.go) skips colony.WorktreeOrphaned
	// entries, and gcOrphanedWorktrees (cmd/codex_build_worktree.go) selects
	// exactly that status for deletion. A guard that inherits the skip is
	// blind precisely where it matters — this test is what stops that
	// regression from returning.
	root, wtPath, entry := newWorktreeSafetyFixture(t, "phase-1/builder-orphaned")
	entry.Status = colony.WorktreeOrphaned

	if err := os.WriteFile(wtPath+"/uncommitted.txt", []byte("dirty\n"), 0644); err != nil {
		t.Fatalf("write uncommitted file: %v", err)
	}

	safety := worktreeDestructionSafety(root, entry)

	if safety.Safe {
		t.Fatal("expected Safe=false for a dirty WorktreeOrphaned entry — the guard must evaluate Orphaned entries, not skip them")
	}
}

// TestWorktreeSafetyRefusesEmptyBranch is CR-01's fail-then-pass proof. An
// entry with no branch recorded (a hand-edited or partially-written
// COLONY_STATE.json) must never read as safe to destroy. Before the CR-01
// fix, `git rev-list --count "main.."` is valid git — it returns 0 with exit
// status 0 for an empty right-hand side — so the unmerged-commit check fell
// through to Safe=true even though the branch could not actually be
// verified at all.
func TestWorktreeSafetyRefusesEmptyBranch(t *testing.T) {
	root, _, entry := newWorktreeSafetyFixture(t, "phase-1/builder-emptybranch")

	// Simulate a WorktreeEntry that was appended before its branch field was
	// assigned — the worktree directory and git branch created by the
	// fixture still exist, but the *entry* claims no branch.
	entry.Branch = ""

	safety := worktreeDestructionSafety(root, entry)

	if safety.Safe {
		t.Fatalf("expected Safe=false for an entry with an empty branch name, got Safe=true (reason: %q) — an empty branch must never resolve to 'safe to destroy'", safety.Reason)
	}
}

func TestWorktreeSafetyRefusesWhenGitCannotAnswer(t *testing.T) {
	root, _, entry := newWorktreeSafetyFixture(t, "phase-1/builder-unreadable")

	// Point the entry at a directory that exists but is not a git worktree.
	notAWorktree := root + "/.aether/not-a-worktree"
	if err := os.MkdirAll(notAWorktree, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	entry.Path = notAWorktree

	safety := worktreeDestructionSafety(root, entry)

	if safety.Safe {
		t.Fatal("expected Safe=false when git cannot determine worktree state — uncertainty must not read as permission")
	}
}

// ---------------------------------------------------------------------------
// preserveWorktreeWork tests
// ---------------------------------------------------------------------------

func TestPreserveWorktreeWorkStashesRatherThanDiscards(t *testing.T) {
	root, wtPath, entry := newWorktreeSafetyFixture(t, "phase-1/builder-stash")

	const content = "precious uncommitted work\n"
	if err := os.WriteFile(wtPath+"/precious.txt", []byte(content), 0644); err != nil {
		t.Fatalf("write uncommitted file: %v", err)
	}

	safety := worktreeDestructionSafety(root, entry)
	if safety.Safe {
		t.Fatalf("fixture setup error: expected the guard to see the dirty file")
	}

	preserved, detail, err := preserveWorktreeWork(root, entry, safety)
	if err != nil {
		t.Fatalf("preserveWorktreeWork returned error: %v", err)
	}
	if !preserved {
		t.Fatal("expected preserved=true for a dirty worktree")
	}
	if detail == "" {
		t.Error("expected a non-empty detail string")
	}

	// The content must be recoverable, never discarded.
	stashListOut, err := exec.Command("git", "-C", wtPath, "stash", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("git stash list: %v: %s", err, stashListOut)
	}
	if strings.TrimSpace(string(stashListOut)) == "" {
		t.Fatal("expected a non-empty stash list after preservation")
	}

	popOut, err := exec.Command("git", "-C", wtPath, "stash", "pop").CombinedOutput()
	if err != nil {
		t.Fatalf("git stash pop: %v: %s", err, popOut)
	}
	restored, err := os.ReadFile(wtPath + "/precious.txt")
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(restored) != content {
		t.Fatalf("expected restored content %q, got %q", content, string(restored))
	}
}

func TestPreserveWorktreeWorkIgnoresCleanWorktree(t *testing.T) {
	root, wtPath, entry := newWorktreeSafetyFixture(t, "phase-1/builder-clean-preserve")

	safety := worktreeDestructionSafety(root, entry)
	if !safety.Safe {
		t.Fatalf("fixture setup error: expected the guard to see a clean worktree, got reason: %q", safety.Reason)
	}

	preserved, detail, err := preserveWorktreeWork(root, entry, safety)
	if err != nil {
		t.Fatalf("preserveWorktreeWork returned error: %v", err)
	}
	if preserved {
		t.Fatal("expected preserved=false for a clean, safe-to-destroy worktree")
	}
	if detail != "" {
		t.Errorf("expected an empty detail for a clean worktree, got %q", detail)
	}

	stashListOut, err := exec.Command("git", "-C", wtPath, "stash", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("git stash list: %v: %s", err, stashListOut)
	}
	if strings.TrimSpace(string(stashListOut)) != "" {
		t.Errorf("expected an empty stash list for a clean worktree, got: %s", stashListOut)
	}
}

// ---------------------------------------------------------------------------
// describeWorktreePreservation tests
// ---------------------------------------------------------------------------

func TestPreservationReportIsPlainEnglish(t *testing.T) {
	safety := worktreeSafety{
		Safe:           false,
		Reason:         "worktree has 1 uncommitted change(s)",
		DirtyFileCount: 1,
		Branch:         "phase-1/builder-dirty",
	}
	detail := "1 uncommitted change(s) stashed on branch phase-1/builder-dirty"

	msg := describeWorktreePreservation(safety, detail)

	if !strings.Contains(msg, safety.Branch) {
		t.Errorf("expected message to name the branch %q, got: %s", safety.Branch, msg)
	}
	if !strings.Contains(msg, "aether recover") {
		t.Errorf("expected message to contain the recovery command %q, got: %s", "aether recover", msg)
	}

	lower := strings.ToLower(msg)
	for _, banned := range []string{"orphan", "gc", "prune"} {
		if strings.Contains(lower, banned) {
			t.Errorf("expected message to omit repo-invented term %q, got: %s", banned, msg)
		}
	}
}
