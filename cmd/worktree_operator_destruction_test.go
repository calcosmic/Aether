package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// This file proves 187-VERIFICATION.md's GAP-1, GAP-2 and GAP-3: three real,
// shipped code paths that destroyed a worktree's uncommitted or unmerged work
// unconditionally, with zero call to worktreeDestructionSafety. Every fixture
// here uses a real `git init` + `git worktree add` repository -- a test that
// never creates the worktree on disk cannot reach the destructive branch and
// proves nothing about it (the same discipline cmd/worktree_crash_safety_test.go
// documents for criterion 1).
//
// Per this repo's Definition of Done and the executor's proof requirement,
// each test below was run against the pre-fix code first and observed to
// FAIL. The failure message actually observed is quoted in each test's
// doc comment, immediately above the test function, exactly as seen before
// the corresponding fix in cmd/worktree.go / cmd/clash.go / cmd/init_cmd.go
// was applied.

// ---------------------------------------------------------------------------
// GAP-1: worktree-merge-back (cmd/worktree.go, worktreeMergeBackCmd Step 5)
// ---------------------------------------------------------------------------

// TestWorktreeMergeBackPreservesUncommittedWorkAfterMerge proves that after a
// successful merge, Step 5's cleanup does not silently discard an
// uncommitted, untracked file left in the worktree alongside the merged
// commit.
//
// Observed pre-fix failure, actually run against the unfixed code (before
// cmd/worktree.go's Step 5 was gated on worktreeDestructionSafety) by
// stashing the fix and re-running this test:
//
//	worktree_operator_destruction_test.go:153: expected the uncommitted file
//	to survive merge-back (directly or via stash), but it is gone: stat
//	/var/folders/.../T/TestWorktreeMergeBackPreservesUncommittedWorkAfterMerge.../
//	001/.aether/worktrees/phase-1-builder-preserve/leftover-notes.txt: no such
//	file or directory (stash list also empty)
//
// (`git worktree remove --force` deleted the untracked file the instant Step
// 5 ran, before this test's assertions even executed -- the merge itself had
// already succeeded, which is exactly the deceptive shape 187-VERIFICATION.md
// describes: the merge gates prove the committed HEAD is safe, but say
// nothing about uncommitted content sitting beside it.)
func TestWorktreeMergeBackPreservesUncommittedWorkAfterMerge(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "checkout", "-b", "main")

	os.WriteFile(tmpDir+"/go.mod", []byte("module test\n\ngo 1.22\n"), 0644)
	os.MkdirAll(tmpDir+"/cmd", 0755)
	os.WriteFile(tmpDir+"/cmd/testhelper_test.go", []byte(`package cmd
import "testing"
func TestHelperPreserve(t *testing.T) {}
`), 0644)
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	branch := "phase-1/builder-preserve"
	wtRelPath := ".aether/worktrees/phase-1-builder-preserve"
	wtPath := tmpDir + "/" + wtRelPath
	runGit(t, tmpDir, "worktree", "add", "-b", branch, wtPath, "HEAD")

	// A real, committed change so the merge itself has something to bring
	// across and succeeds.
	os.WriteFile(wtPath+"/cmd/newfile_test.go", []byte(`package cmd
import "testing"
func TestNewFilePreserve(t *testing.T) {}
`), 0644)
	runGit(t, wtPath, "add", ".")
	runGit(t, wtPath, "commit", "-m", "add new test")

	// The uncommitted, untracked file this test is actually about. Left
	// behind the same way a crash mid-task would leave it.
	const leftoverContent = "distinctive leftover content that must survive\n"
	if err := os.WriteFile(wtPath+"/leftover-notes.txt", []byte(leftoverContent), 0644); err != nil {
		t.Fatalf("write leftover file: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_preserve_001",
			Branch:    branch,
			Path:      wtRelPath,
			Status:    colony.WorktreeInProgress,
			Phase:     1,
			Agent:     "builder-preserve",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	state := makeTestStateWithWorktrees(worktrees)
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-merge-back", "--branch", branch})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The merge must still have succeeded -- this test is about the cleanup
	// step, not the merge gates.
	stdoutOutput := stdoutBuf.String()
	if stdoutOutput == "" {
		t.Fatalf("expected JSON output on stdout, got empty. stderr: %s", stderrBuf.String())
	}
	envelope := assertOKEnvelope(t, stdoutOutput)
	result := envelope["result"].(map[string]interface{})
	if result["merged"] != true {
		t.Fatalf("expected merged=true, got %v (stderr: %s)", result["merged"], stderrBuf.String())
	}

	// The uncommitted content must be recoverable -- either still present in
	// the working tree (destruction was skipped entirely), or stashed (a
	// preservation path chose to stash rather than leave the dirty worktree
	// in place). Either is an acceptable D-01 outcome; silent loss is not.
	survivedDirectly := false
	if content, readErr := os.ReadFile(wtPath + "/leftover-notes.txt"); readErr == nil {
		if string(content) == leftoverContent {
			survivedDirectly = true
		}
	}
	if !survivedDirectly {
		stashOut, stashErr := exec.Command("git", "-C", wtPath, "stash", "list").CombinedOutput()
		if stashErr != nil || strings.TrimSpace(string(stashOut)) == "" {
			t.Fatalf("expected the uncommitted file to survive merge-back (directly or via stash), but it is gone: stat %s: no such file or directory (stash list also empty)", wtPath+"/leftover-notes.txt")
		}
	}

	// D-02: the preservation (or the decision not to destroy) must be
	// reported in plain language, not silent.
	if survivedDirectly {
		combined := stdoutOutput + stderrBuf.String()
		if !strings.Contains(combined, "Kept the work") && !strings.Contains(combined, "cleaned_up") {
			t.Errorf("expected a plain-language report that work was kept, got stdout=%q stderr=%q", stdoutOutput, stderrBuf.String())
		}
	}
}

// ---------------------------------------------------------------------------
// GAP-2: worktree-cleanup (cmd/clash.go, worktreeCleanupCmd)
// ---------------------------------------------------------------------------

// TestWorktreeCleanupRefusesToDestroyDirtyWorktree proves worktree-cleanup no
// longer force-removes a worktree that holds uncommitted changes.
//
// Observed pre-fix failure, actually run against the unfixed code (before
// cmd/clash.go's worktreeCleanupCmd called worktreeDestructionSafety) by
// stashing the fix and re-running this test:
//
//	worktree_operator_destruction_test.go:250: expected a plain-language
//	report that the work was kept, got: {"ok":false,"error":"failed to
//	remove worktree: exit status 128: fatal: 'phase-1/builder-dirty' is not
//	a working tree\n","code":2}
//
// (the pre-fix command ran `git worktree remove <branch> --force` passing
// the BRANCH NAME directly as the path argument -- for this fixture's
// directory naming convention that name is not a valid path, so git refuses
// with "is not a working tree" rather than silently deleting anything. That
// is still the bug 187-VERIFICATION.md names: no safety check of any kind
// runs, the worktree is never located correctly, and for any branch name
// that DOES happen to equal its own worktree directory name -- exactly the
// convention validateBranchName's agent track produces when a caller
// constructs the path that way -- the same call would have deleted real,
// uncommitted content unconditionally, exit 0, no warning.)
func TestWorktreeCleanupRefusesToDestroyDirtyWorktree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "checkout", "-b", "main")
	os.WriteFile(tmpDir+"/README.md", []byte("initial\n"), 0644)
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	branch := "phase-1/builder-dirty"
	wtRelPath := ".aether/worktrees/phase-1-builder-dirty"
	wtPath := tmpDir + "/" + wtRelPath
	runGit(t, tmpDir, "worktree", "add", "-b", branch, wtPath, "HEAD")

	const dirtyContent = "distinctive dirty content that must survive\n"
	if err := os.WriteFile(wtPath+"/dirty-work.txt", []byte(dirtyContent), 0644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-cleanup", "--branch", branch})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The worktree directory itself must still exist.
	if _, statErr := os.Stat(wtPath); statErr != nil {
		t.Fatalf("expected worktree directory to still exist, stat error: %v", statErr)
	}

	// The dirty file must be recoverable -- directly present or stashed.
	survivedDirectly := false
	if content, readErr := os.ReadFile(wtPath + "/dirty-work.txt"); readErr == nil {
		if string(content) == dirtyContent {
			survivedDirectly = true
		}
	}
	if !survivedDirectly {
		stashOut, stashErr := exec.Command("git", "-C", wtPath, "stash", "list").CombinedOutput()
		if stashErr != nil || strings.TrimSpace(string(stashOut)) == "" {
			t.Fatalf("expected the dirty file to survive worktree-cleanup, but it is gone: stat %s: no such file or directory", wtPath+"/dirty-work.txt")
		}
	}

	// D-02: report, don't stay silent.
	combined := stdoutBuf.String() + stderrBuf.String()
	if !strings.Contains(combined, "left alone") && !strings.Contains(combined, "Kept the work") && !strings.Contains(combined, "preserved") {
		t.Errorf("expected a plain-language report that the work was kept, got: %s", combined)
	}
}

// TestWorktreeCleanupRemovesCleanMergedWorktree is the companion positive
// case: a worktree with no uncommitted changes and no unmerged commits must
// still be removable by worktree-cleanup (the fix must not turn this into a
// command that can never clean anything up).
func TestWorktreeCleanupRemovesCleanMergedWorktree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "checkout", "-b", "main")
	os.WriteFile(tmpDir+"/README.md", []byte("initial\n"), 0644)
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	branch := "phase-1/builder-clean"
	wtRelPath := ".aether/worktrees/phase-1-builder-clean"
	wtPath := tmpDir + "/" + wtRelPath
	// No commits ahead of main, no dirty files: fully clean and merged.
	runGit(t, tmpDir, "worktree", "add", "-b", branch, wtPath, "HEAD")

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-cleanup", "--branch", branch})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(wtPath); statErr == nil {
		t.Errorf("expected clean, merged worktree to be removed, but it still exists")
	}

	envelope := assertOKEnvelope(t, stdoutBuf.String())
	result := envelope["result"].(map[string]interface{})
	if result["cleaned"] != true {
		t.Errorf("expected cleaned=true, got %v (stderr: %s)", result["cleaned"], stderrBuf.String())
	}
}

// ---------------------------------------------------------------------------
// GAP-3: init's worktrees-directory wipe (cmd/init_cmd.go)
// ---------------------------------------------------------------------------

// TestInitPreservesUnrecordedWorktreeWithUncommittedWork proves that `aether
// init` no longer wipes a worktree that was created on disk but never made it
// into COLONY_STATE.json -- the exact crash window between `git worktree add`
// and appendBuildWorktreeEntry that this phase exists to protect.
//
// Observed pre-fix failure, actually run against the unfixed code (before
// cmd/init_cmd.go's wtPreserved==0 check consulted the disk, only the
// tracked-entry count) by stashing the fix and re-running this test:
//
//	worktree_operator_destruction_test.go:420: expected the unrecorded
//	worktree's uncommitted file to survive init, but it is gone: stat
//	/var/folders/.../T/TestInitPreservesUnrecordedWorktreeWithUncommittedWork.../
//	001/.aether/worktrees/phase-9-builder-unrecorded/crash-notes.txt: no such
//	file or directory (stash list also empty, directory itself gone)
//
// (COLONY_STATE.json in this fixture has ZERO worktree entries -- the crash
// happened before appendBuildWorktreeEntry ever ran -- so gcOrphanedWorktrees
// sees nothing to preserve, wtPreserved comes back 0, and the pre-fix
// `os.RemoveAll(filepath.Join(aetherDir, "worktrees"))` wiped the directory
// and its uncommitted content unconditionally.)
func TestInitPreservesUnrecordedWorktreeWithUncommittedWork(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "checkout", "-b", "main")
	os.WriteFile(tmpDir+"/README.md", []byte("initial\n"), 0644)
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// A worktree created directly via git -- simulating the crash window
	// where `git worktree add` succeeded but appendBuildWorktreeEntry never
	// ran, so COLONY_STATE.json below has no worktree entries at all.
	branch := "phase-9/builder-unrecorded"
	wtRelPath := ".aether/worktrees/phase-9-builder-unrecorded"
	wtPath := tmpDir + "/" + wtRelPath
	runGit(t, tmpDir, "worktree", "add", "-b", branch, wtPath, "HEAD")

	const crashContent = "distinctive crash-window content that must survive\n"
	if err := os.WriteFile(wtPath+"/crash-notes.txt", []byte(crashContent), 0644); err != nil {
		t.Fatalf("write crash file: %v", err)
	}

	// A pre-existing colony state with a real goal so init takes the
	// "already initialized" branch that reaches the worktree-wipe logic,
	// and crucially: zero worktree entries, matching the crash window.
	priorGoal := "prior colony goal"
	priorState := colony.ColonyState{
		Version:      "3.0",
		Goal:         &priorGoal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Worktrees:    []colony.WorktreeEntry{},
	}

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := s.SaveJSON("COLONY_STATE.json", priorState); err != nil {
		t.Fatalf("save prior state: %v", err)
	}
	store = s

	rootCmd.SetArgs([]string{"init", "New goal after crash", "--confirm-reinit"})

	// reportWorktreePreservation (cmd/worktree_safety.go) writes via
	// fmt.Fprintf(os.Stderr, ...) directly to the real file descriptor, not
	// the package-level `stderr` var this test also redirects -- capture
	// both so D-02's report is actually observed.
	realStderr := captureRealStderr(t, func() {
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if stdoutBuf.String() == "" {
		t.Fatalf("expected JSON output on stdout, got empty. stderr: %s / %s", stderrBuf.String(), realStderr)
	}

	// The worktree directory (or at minimum its content) must survive.
	survivedDirectly := false
	if content, readErr := os.ReadFile(wtPath + "/crash-notes.txt"); readErr == nil {
		if string(content) == crashContent {
			survivedDirectly = true
		}
	}
	if !survivedDirectly {
		// Directory may have been legitimately relocated/stashed by a
		// preservation path; check for a stash as the fallback proof of
		// survival, matching the crash-safety test's own acceptance shape.
		if _, statErr := os.Stat(wtPath); statErr == nil {
			stashOut, stashErr := exec.Command("git", "-C", wtPath, "stash", "list").CombinedOutput()
			if stashErr == nil && strings.TrimSpace(string(stashOut)) != "" {
				survivedDirectly = true // preserved via stash, directory intact
			}
		}
	}
	if !survivedDirectly {
		t.Fatalf("expected the unrecorded worktree's uncommitted file to survive init, but it is gone: stat %s: no such file or directory (stash list also empty, directory itself gone)", wtPath+"/crash-notes.txt")
	}

	// D-02: the preservation must be reported in plain language.
	combined := stdoutBuf.String() + stderrBuf.String() + realStderr
	if !strings.Contains(combined, "aether recover") && !strings.Contains(combined, "Kept the work") && !strings.Contains(combined, "left in place") {
		t.Errorf("expected a plain-language report that leftover work was kept, got: %s", combined)
	}
}

// TestInitStillWipesWorktreesDirectoryWhenTrulyEmpty is the companion
// positive case: when the worktrees directory holds nothing (no tracked
// entries, no unrecorded git worktrees), init must still perform its normal
// clean-slate wipe. The fix must not turn every init into a permanent
// preservation no-op.
func TestInitStillWipesWorktreesDirectoryWhenTrulyEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "checkout", "-b", "main")
	os.WriteFile(tmpDir+"/README.md", []byte("initial\n"), 0644)
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// A stray, empty, non-worktree directory under .aether/worktrees --
	// something that is NOT a git worktree at all (e.g. leftover empty dir).
	strayDir := filepath.Join(tmpDir, ".aether", "worktrees", "not-a-worktree")
	os.MkdirAll(strayDir, 0755)

	priorGoal := "prior colony goal"
	priorState := colony.ColonyState{
		Version:      "3.0",
		Goal:         &priorGoal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Worktrees:    []colony.WorktreeEntry{},
	}

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := s.SaveJSON("COLONY_STATE.json", priorState); err != nil {
		t.Fatalf("save prior state: %v", err)
	}
	store = s

	rootCmd.SetArgs([]string{"init", "New goal, truly empty worktrees", "--confirm-reinit"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdoutBuf.String() == "" {
		t.Fatalf("expected JSON output on stdout, got empty")
	}

	worktreesDir := filepath.Join(tmpDir, ".aether", "worktrees")
	if _, statErr := os.Stat(worktreesDir); statErr == nil {
		t.Errorf("expected worktrees directory to be wiped when it holds no real worktree work, but it still exists")
	}
}
