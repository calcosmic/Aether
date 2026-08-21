package cmd

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ---------------------------------------------------------------------------
// mergePhaseWorktrees real-git fixture tests
//
// mergePhaseWorktrees is the build-path merge gate (called live from
// cmd/codex_build_finalize.go:569 on every worktree-mode build finalization).
// Its only prior test, TestMergePhaseWorktreesEmpty, passes an empty
// worktree list, so the loop body -- the test gate, the clash gate, the
// checkout fallback and the merge -- was entirely uncovered. These tests
// build a real git repository via runGit, a real `git worktree add`, and
// real commits, then call mergePhaseWorktrees directly (it is unexported --
// there is no cobra command for it).
//
// Rooting note: mergePhaseWorktrees expands each entry's relative Path with
// resolveAetherRoot() (AETHER_ROOT env, then process cwd), while
// resolveTestCommand() roots at the parent-of-parent of store.BasePath().
// With AETHER_ROOT = tmpDir and dataDir = tmpDir/.aether/data, both resolve
// to tmpDir, so one fixture satisfies both resolvers.
// ---------------------------------------------------------------------------

// mergeWorktreeFixture builds a real git repo at t.TempDir(), with a single
// worktree entry tracked in COLONY_STATE.json for phase 1, and points the
// package-level store and AETHER_ROOT at it. Callers write whichever project
// manifest they need (CLAUDE.md, go.mod, package.json, ...) into tmpDir
// before or after calling this helper -- this helper only creates the repo,
// the worktree, and the state; it does not decide the project's ecosystem.
func mergeWorktreeFixture(t *testing.T, branch string) (tmpDir, wtAbsPath string, s *storage.Store) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir = t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "checkout", "-b", "main")

	return tmpDir, "", nil
}

// finishMergeWorktreeFixture commits whatever manifest/source files the
// caller has already written to tmpDir, creates the worktree, commits a
// real change inside it, writes COLONY_STATE.json with one tracked entry,
// and wires AETHER_ROOT + the package store. Returns the worktree's
// absolute path and the initialized store.
func finishMergeWorktreeFixture(t *testing.T, tmpDir, branch, agentSlug string) (wtAbsPath string, s *storage.Store) {
	t.Helper()

	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	relPath := ".aether/worktrees/phase-1-" + agentSlug
	wtAbsPath = tmpDir + "/" + relPath
	runGit(t, tmpDir, "worktree", "add", "-b", branch, wtAbsPath, "HEAD")

	if err := os.WriteFile(wtAbsPath+"/notes.txt", []byte("work done in worktree\n"), 0644); err != nil {
		t.Fatalf("failed to write worktree file: %v", err)
	}
	runGit(t, wtAbsPath, "add", ".")
	runGit(t, wtAbsPath, "commit", "-m", "worktree work")

	now := time.Now().UTC().Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_" + agentSlug + "_001",
			Branch:    branch,
			Path:      relPath,
			Status:    colony.WorktreeInProgress,
			Phase:     1,
			Agent:     agentSlug,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	dataDir := tmpDir + "/.aether/data"
	stateJSON := makeTestStateWithWorktrees(worktrees)
	if err := os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(stateJSON), 0644); err != nil {
		t.Fatalf("failed to write COLONY_STATE.json: %v", err)
	}

	t.Setenv("AETHER_ROOT", tmpDir)

	newStore, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = newStore

	return wtAbsPath, newStore
}

func TestMergePhaseWorktreesUsesProjectTestCommandInNodeRepo(t *testing.T) {
	tmpDir, _, _ := mergeWorktreeFixture(t, "phase-1/builder-a")

	// NO go.mod. extractTestCommand() only recognizes the literal substrings
	// "go test", "npm test", "cargo test" -- put "npm test" in CLAUDE.md so
	// the resolver's first tier (CLAUDE.md) answers deterministically without
	// depending on npm being installed in this environment. Also write a
	// package.json so the fixture is recognisably a Node project.
	if err := os.WriteFile(tmpDir+"/CLAUDE.md", []byte("# Project\n\nRun tests: npm test\n"), 0644); err != nil {
		t.Fatalf("failed to write CLAUDE.md: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/package.json", []byte(`{"name":"test","scripts":{"test":"exit 0"}}`), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	finishMergeWorktreeFixture(t, tmpDir, "phase-1/builder-a", "builder-a")

	merged, failed, err := mergePhaseWorktrees(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(failed) != 0 {
		t.Errorf("expected 0 failed, got %v", failed)
	}
	found := false
	for _, b := range merged {
		if b == "phase-1/builder-a" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected phase-1/builder-a in merged, got %v", merged)
	}
}

func TestMergePhaseWorktreesRefusesWhenTestCommandUnknown(t *testing.T) {
	tmpDir, _, _ := mergeWorktreeFixture(t, "phase-1/builder-b")

	// No go.mod, no package.json, no Cargo.toml, no pom.xml, no CLAUDE.md,
	// no .aether/data/codebase.md -- resolveTestCommand() must return "".
	if err := os.WriteFile(tmpDir+"/README.md", []byte("# unresolvable project\n"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	finishMergeWorktreeFixture(t, tmpDir, "phase-1/builder-b", "builder-b")

	merged, failed, err := mergePhaseWorktrees(1)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot determine how to test this project") {
		t.Errorf("expected error to contain 'cannot determine how to test this project', got: %v", err)
	}
	if len(merged) != 0 {
		t.Errorf("expected 0 merged, got %v", merged)
	}
	// The refusal is an error, not a per-branch failure -- a failed entry
	// would tell build-finalize a specific branch was blocked, which is a
	// different and misleading claim than "cannot determine how to test".
	if len(failed) != 0 {
		t.Errorf("expected 0 failed (refusal surfaces via err, not failed), got %v", failed)
	}
}

func TestMergePhaseWorktreesRefusalPreservesTheWorktree(t *testing.T) {
	tmpDir, _, _ := mergeWorktreeFixture(t, "phase-1/builder-c")

	if err := os.WriteFile(tmpDir+"/README.md", []byte("# unresolvable project\n"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	branch := "phase-1/builder-c"
	wtAbsPath, s := finishMergeWorktreeFixture(t, tmpDir, branch, "builder-c")

	_, _, err := mergePhaseWorktrees(1)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	// The worktree directory must still exist on disk.
	if _, statErr := os.Stat(wtAbsPath); statErr != nil {
		t.Errorf("expected worktree directory to still exist at %s: %v", wtAbsPath, statErr)
	}

	// git worktree list must still contain the worktree path.
	listOut, listErr := exec.Command("git", "-C", tmpDir, "worktree", "list").CombinedOutput()
	if listErr != nil {
		t.Fatalf("git worktree list failed: %v: %s", listErr, string(listOut))
	}
	if !strings.Contains(string(listOut), wtAbsPath) {
		t.Errorf("expected git worktree list to contain %s, got: %s", wtAbsPath, string(listOut))
	}

	// git branch --list must still show the branch.
	branchOut, branchErr := exec.Command("git", "-C", tmpDir, "branch", "--list", branch).CombinedOutput()
	if branchErr != nil {
		t.Fatalf("git branch --list failed: %v: %s", branchErr, string(branchOut))
	}
	if strings.TrimSpace(string(branchOut)) == "" {
		t.Errorf("expected git branch --list %s to be non-empty", branch)
	}

	// The commit made inside the worktree must still be reachable from the
	// branch.
	logOut, logErr := exec.Command("git", "-C", tmpDir, "log", "--oneline", branch).CombinedOutput()
	if logErr != nil {
		t.Fatalf("git log --oneline %s failed: %v: %s", branch, logErr, string(logOut))
	}
	if !strings.Contains(string(logOut), "worktree work") {
		t.Errorf("expected git log %s to contain the worktree commit, got: %s", branch, string(logOut))
	}

	// The reloaded COLONY_STATE.json entry's Status must NOT be Merged.
	var reloadState colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &reloadState); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	for _, wt := range reloadState.Worktrees {
		if wt.Branch == branch && wt.Status == colony.WorktreeMerged {
			t.Errorf("expected status NOT merged, got %s", wt.Status)
		}
	}
}

func TestMergePhaseWorktreesStillMergesGoRepo(t *testing.T) {
	tmpDir, _, _ := mergeWorktreeFixture(t, "phase-1/builder-d")

	if err := os.WriteFile(tmpDir+"/go.mod", []byte("module merge-fixture\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/root_test.go", []byte("package merge_fixture\n\nimport \"testing\"\n\nfunc TestRoot(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("failed to write root_test.go: %v", err)
	}

	branch := "phase-1/builder-d"
	wtAbsPath, _ := finishMergeWorktreeFixture(t, tmpDir, branch, "builder-d")

	// Add a second passing test file inside the worktree itself.
	if err := os.WriteFile(wtAbsPath+"/worktree_test.go", []byte("package merge_fixture\n\nimport \"testing\"\n\nfunc TestWorktree(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("failed to write worktree_test.go: %v", err)
	}
	runGit(t, wtAbsPath, "add", ".")
	runGit(t, wtAbsPath, "commit", "-m", "add worktree test")

	merged, failed, err := mergePhaseWorktrees(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(failed) != 0 {
		t.Errorf("expected 0 failed, got %v", failed)
	}
	found := false
	for _, b := range merged {
		if b == branch {
			found = true
		}
	}
	if !found {
		t.Errorf("expected %s in merged, got %v", branch, merged)
	}

	// Note: mergePhaseWorktrees (unlike worktreeMergeBackCmd in worktree.go)
	// does not itself write state.Worktrees[i].Status = WorktreeMerged after
	// a successful merge -- that is a pre-existing gap unrelated to this
	// plan's ecosystem-neutrality fix and out of scope here. The git-level
	// merge is what this test proves: confirm the branch's commit is now
	// reachable from main.
	logOut, logErr := exec.Command("git", "-C", tmpDir, "log", "--oneline", "main").CombinedOutput()
	if logErr != nil {
		t.Fatalf("git log --oneline main failed: %v: %s", logErr, string(logOut))
	}
	if !strings.Contains(string(logOut), "add worktree test") {
		t.Errorf("expected main's log to contain the merged worktree commit, got: %s", string(logOut))
	}
}
