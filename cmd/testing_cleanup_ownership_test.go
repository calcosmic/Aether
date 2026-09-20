package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestCleanupTestWorktreesPreservesUnregisteredDecoys(t *testing.T) {
	repoRoot := t.TempDir()
	runGit(t, repoRoot, "init", "-b", "main")
	runGit(t, repoRoot, "config", "user.email", "test@example.com")
	runGit(t, repoRoot, "config", "user.name", "Test")
	runGit(t, repoRoot, "commit", "--allow-empty", "-m", "initial")

	ownedBranch := "feature/test-owned-cleanup"
	ownedPath := filepath.Join(repoRoot, ".aether", "worktrees", "registered-cleanup")
	registerTestOwnedWorktree(t, repoRoot, ownedPath, ownedBranch)
	runGit(t, repoRoot, "worktree", "add", "-b", ownedBranch, ownedPath, "HEAD")

	phaseDecoyBranch := "phase-199/owner-work"
	featureDecoyBranch := "feature/owner-work"
	worktreeDecoyBranch := "owner/aether-shaped-worktree"
	runGit(t, repoRoot, "branch", phaseDecoyBranch)
	runGit(t, repoRoot, "branch", featureDecoyBranch)
	decoyPath := filepath.Join(repoRoot, ".aether", "worktrees", "owner-decoy")
	runGit(t, repoRoot, "worktree", "add", "-b", worktreeDecoyBranch, decoyPath, "HEAD")
	markerPath := filepath.Join(decoyPath, "owner-work.txt")
	marker := []byte("owner work must survive test cleanup\n")
	if err := os.WriteFile(markerPath, marker, 0644); err != nil {
		t.Fatalf("write decoy marker: %v", err)
	}

	cleanupTestWorktrees()
	cleanupTestWorktrees()

	if _, err := os.Lstat(ownedPath); !os.IsNotExist(err) {
		t.Fatalf("registered worktree still exists: %v", err)
	}
	if got, err := os.ReadFile(markerPath); err != nil || string(got) != string(marker) {
		t.Fatalf("unregistered decoy bytes changed: got %q, err %v", got, err)
	}

	wantBranches := []string{"feature/owner-work", "main", "owner/aether-shaped-worktree", "phase-199/owner-work"}
	if got := cleanupTestBranches(t, repoRoot); strings.Join(got, "\n") != strings.Join(wantBranches, "\n") {
		t.Fatalf("branches after cleanup = %v, want exactly %v", got, wantBranches)
	}
	wantWorktrees := canonicalCleanupTestPaths(t, repoRoot, decoyPath)
	if got := cleanupTestWorktreePaths(t, repoRoot); strings.Join(got, "\n") != strings.Join(wantWorktrees, "\n") {
		t.Fatalf("worktrees after cleanup = %v, want exactly %v", got, wantWorktrees)
	}
}

func cleanupTestBranches(t *testing.T, repoRoot string) []string {
	t.Helper()
	output, err := exec.Command("git", "-C", repoRoot, "for-each-ref", "--format=%(refname:short)", "refs/heads").Output()
	if err != nil {
		t.Fatalf("list cleanup-test branches: %v", err)
	}
	branches := strings.Fields(string(output))
	sort.Strings(branches)
	return branches
}

func cleanupTestWorktreePaths(t *testing.T, repoRoot string) []string {
	t.Helper()
	output, err := exec.Command("git", "-C", repoRoot, "worktree", "list", "--porcelain").Output()
	if err != nil {
		t.Fatalf("list cleanup-test worktrees: %v", err)
	}
	var paths []string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			paths = append(paths, canonicalCleanupTestPaths(t, strings.TrimPrefix(line, "worktree "))...)
		}
	}
	sort.Strings(paths)
	return paths
}

func canonicalCleanupTestPaths(t *testing.T, paths ...string) []string {
	t.Helper()
	canonical := make([]string, 0, len(paths))
	for _, path := range paths {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			t.Fatalf("resolve cleanup-test path %s: %v", path, err)
		}
		canonical = append(canonical, filepath.Clean(resolved))
	}
	sort.Strings(canonical)
	return canonical
}
