package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// This file proves ROADMAP criterion 1 end to end: a crash between dispatch
// and finalize, followed by a resume, must never destroy the work being
// resumed. Every fixture here uses a real `git init` repository and a real
// `git worktree add` — a test pointing at a path that was never created on
// disk cannot reach the destructive branch, and therefore cannot prove
// anything about it (see TestCleanupBuildWorktrees's own admission at
// cmd/codex_build_worktree_test.go:717-756).
//
// WorktreeEntry.Path is stored RELATIVE to root in production (see
// cmd/worktree_test.go's fixtures, e.g. ".aether/worktrees/phase-2-builder-1")
// and joined with root by callers (gcOrphanedWorktrees, worktreeDestructionSafety,
// worktree-reap). Every fixture here follows that convention deliberately —
// an absolute Path in a fixture would cause filepath.Join(root, entry.Path)
// to double-prefix root, silently pointing every git operation at a
// nonexistent directory and making a test pass regardless of what the code
// under test actually does.

// ---------------------------------------------------------------------------
// Shared fixture helper
// ---------------------------------------------------------------------------

// crashSafetyFixture builds a real git repo on branch main with one commit,
// sets AETHER_ROOT and the package-level store to point at it, and returns
// the root so callers can add worktrees with runGit(t, root, "worktree",
// "add", ...).
func crashSafetyFixture(t *testing.T) (root, dataDir string) {
	t.Helper()

	root = t.TempDir()
	dataDir = filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir dataDir: %v", err)
	}

	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("initial\n"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")

	oldRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", root)
	t.Cleanup(func() { os.Setenv("AETHER_ROOT", oldRoot) })

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	store = s

	return root, dataDir
}

// addCrashSafetyWorktree runs a real `git worktree add` under root and
// returns both the absolute path (for filesystem/git operations in the
// test) and the root-relative path (for WorktreeEntry.Path, matching
// production convention).
func addCrashSafetyWorktree(t *testing.T, root, branch, dirName string) (absPath, relPath string) {
	t.Helper()
	relPath = filepath.Join(".aether", "worktrees", dirName)
	absPath = filepath.Join(root, relPath)
	runGit(t, root, "worktree", "add", "-b", branch, absPath, "HEAD")
	return absPath, relPath
}

// writeColonyStateWithWorktrees writes a minimal but valid COLONY_STATE.json
// containing exactly the given worktree entries.
func writeColonyStateWithWorktrees(t *testing.T, dataDir string, worktrees []colony.WorktreeEntry) {
	t.Helper()
	state := makeTestStateWithWorktrees(worktrees)
	if err := os.WriteFile(filepath.Join(dataDir, "COLONY_STATE.json"), []byte(state), 0644); err != nil {
		t.Fatalf("write COLONY_STATE.json: %v", err)
	}
}

// captureRealStderr redirects the process's real os.Stderr (not the
// package-level stderr variable) to a pipe for the duration of fn, and
// returns everything written to it. reportWorktreePreservation
// (cmd/worktree_safety.go) writes via fmt.Fprintf(os.Stderr, ...) directly,
// so tests asserting on its output must capture the real file descriptor.
func captureRealStderr(t *testing.T, fn func()) string {
	t.Helper()
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	fn()

	w.Close()
	var buf bytes.Buffer
	if _, copyErr := buf.ReadFrom(r); copyErr != nil {
		t.Fatalf("read captured stderr: %v", copyErr)
	}
	return buf.String()
}

func loadColonyStateFixture(t *testing.T, dataDir string) colony.ColonyState {
	t.Helper()
	var state colony.ColonyState
	raw, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read COLONY_STATE.json: %v", err)
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("parse COLONY_STATE.json: %v", err)
	}
	return state
}

// ---------------------------------------------------------------------------
// TestCrashBetweenDispatchAndFinalizeSurvivesResume
// ---------------------------------------------------------------------------

func TestCrashBetweenDispatchAndFinalizeSurvivesResume(t *testing.T) {
	root, dataDir := crashSafetyFixture(t)

	branch := "phase-3/builder-crashed"
	wtPath, relPath := addCrashSafetyWorktree(t, root, branch, "phase-3-builder-crashed")

	// A distinctive committed file inside the worktree.
	if err := os.WriteFile(filepath.Join(wtPath, "committed-work.txt"), []byte("distinctive committed content\n"), 0644); err != nil {
		t.Fatalf("write committed file: %v", err)
	}
	runGit(t, wtPath, "add", ".")
	runGit(t, wtPath, "commit", "-m", "builder work before crash")

	// An additional uncommitted file, left behind by the crash.
	const uncommittedContent = "distinctive uncommitted content\n"
	if err := os.WriteFile(filepath.Join(wtPath, "uncommitted-work.txt"), []byte(uncommittedContent), 0644); err != nil {
		t.Fatalf("write uncommitted file: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	entry := colony.WorktreeEntry{
		ID:        "wt-crash-1",
		Branch:    branch,
		Path:      relPath,
		Status:    colony.WorktreeInProgress, // the state a kill between dispatch and finalize leaves behind
		Phase:     3,
		CreatedAt: now,
		UpdatedAt: now,
	}
	writeColonyStateWithWorktrees(t, dataDir, []colony.WorktreeEntry{entry})

	cleaned, preserved, err := gcOrphanedWorktrees()
	if err != nil {
		t.Fatalf("gcOrphanedWorktrees returned error: %v", err)
	}

	if preserved < 1 {
		t.Errorf("expected preserved >= 1, got %d", preserved)
	}
	if cleaned != 0 {
		t.Errorf("expected this entry not counted as cleaned, got cleaned=%d", cleaned)
	}

	// The worktree directory must still exist on disk.
	if _, statErr := os.Stat(wtPath); statErr != nil {
		t.Fatalf("expected worktree directory to still exist, stat error: %v", statErr)
	}

	// The branch must still be a real git branch.
	branchOut, err := exec.Command("git", "-C", root, "branch", "--list", branch).CombinedOutput()
	if err != nil {
		t.Fatalf("git branch --list: %v: %s", err, branchOut)
	}
	if strings.TrimSpace(string(branchOut)) == "" {
		t.Fatalf("expected branch %q to still exist, got empty branch --list output", branch)
	}

	// The committed file must still be reachable on that branch.
	showOut, err := exec.Command("git", "-C", root, "show", branch+":committed-work.txt").CombinedOutput()
	if err != nil {
		t.Fatalf("git show %s:committed-work.txt: %v: %s", branch, err, showOut)
	}
	if strings.TrimSpace(string(showOut)) != "distinctive committed content" {
		t.Fatalf("expected committed file content on branch, got: %q", string(showOut))
	}

	// The uncommitted content must be recoverable — either still in the
	// working tree, or stashed and recoverable via `git stash pop`.
	restoredDirectly := false
	if content, readErr := os.ReadFile(filepath.Join(wtPath, "uncommitted-work.txt")); readErr == nil {
		if string(content) == uncommittedContent {
			restoredDirectly = true
		}
	}
	if !restoredDirectly {
		stashListOut, stashErr := exec.Command("git", "-C", wtPath, "stash", "list").CombinedOutput()
		if stashErr != nil {
			t.Fatalf("git stash list: %v: %s", stashErr, stashListOut)
		}
		if strings.TrimSpace(string(stashListOut)) == "" {
			t.Fatal("uncommitted content is neither present in the working tree nor in a stash — it was lost")
		}
		popOut, popErr := exec.Command("git", "-C", wtPath, "stash", "pop").CombinedOutput()
		if popErr != nil {
			t.Fatalf("git stash pop: %v: %s", popErr, popOut)
		}
		restored, readErr := os.ReadFile(filepath.Join(wtPath, "uncommitted-work.txt"))
		if readErr != nil {
			t.Fatalf("read restored file after stash pop: %v", readErr)
		}
		if string(restored) != uncommittedContent {
			t.Fatalf("expected byte-identical content after stash pop, got: %q", string(restored))
		}
	}

	// The entry must still be present in the reloaded state.
	reloaded := loadColonyStateFixture(t, dataDir)
	found := false
	for _, wt := range reloaded.Worktrees {
		if wt.Branch == branch {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected entry for branch %q to still be present in COLONY_STATE.json, got worktrees: %+v", branch, reloaded.Worktrees)
	}
}

// ---------------------------------------------------------------------------
// TestResumeReportsPreservedWorkInPlainEnglish
// ---------------------------------------------------------------------------

func TestResumeReportsPreservedWorkInPlainEnglish(t *testing.T) {
	root, dataDir := crashSafetyFixture(t)

	branch := "phase-2/builder-loud"
	wtPath, relPath := addCrashSafetyWorktree(t, root, branch, "phase-2-builder-loud")

	if err := os.WriteFile(filepath.Join(wtPath, "dirty.txt"), []byte("dirty\n"), 0644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	entry := colony.WorktreeEntry{
		ID:        "wt-loud-1",
		Branch:    branch,
		Path:      relPath,
		Status:    colony.WorktreeInProgress,
		Phase:     2,
		CreatedAt: now,
		UpdatedAt: now,
	}
	writeColonyStateWithWorktrees(t, dataDir, []colony.WorktreeEntry{entry})

	// reportWorktreePreservation writes to the real os.Stderr, not the
	// package-level stderr variable — capture it via a pipe.
	output := captureRealStderr(t, func() {
		if _, _, err := gcOrphanedWorktrees(); err != nil {
			t.Fatalf("gcOrphanedWorktrees returned error: %v", err)
		}
	})

	if !strings.Contains(output, branch) {
		t.Errorf("expected report to name the branch %q, got: %s", branch, output)
	}
	if !strings.Contains(output, "aether recover") {
		t.Errorf("expected report to contain a recovery command the user can type, got: %s", output)
	}

	lower := strings.ToLower(output)
	for _, banned := range []string{"orphan", "gc", "prune"} {
		if strings.Contains(lower, banned) {
			t.Errorf("expected report to omit repo-invented term %q, got: %s", banned, output)
		}
	}
}

// ---------------------------------------------------------------------------
// TestResumingOnePhaseDoesNotDestroyAnotherPhasesWorktree
// ---------------------------------------------------------------------------

func TestResumingOnePhaseDoesNotDestroyAnotherPhasesWorktree(t *testing.T) {
	root, dataDir := crashSafetyFixture(t)

	stalledBranch := "phase-2/builder-stalled"
	stalledPath, stalledRel := addCrashSafetyWorktree(t, root, stalledBranch, "phase-2-builder-stalled")
	if err := os.WriteFile(filepath.Join(stalledPath, "stalled.txt"), []byte("stalled work\n"), 0644); err != nil {
		t.Fatalf("write stalled file: %v", err)
	}
	runGit(t, stalledPath, "add", ".")
	runGit(t, stalledPath, "commit", "-m", "stalled unmerged commit")

	liveBranch := "phase-5/builder-live"
	_, liveRel := addCrashSafetyWorktree(t, root, liveBranch, "phase-5-builder-live")

	now := time.Now().UTC().Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{ID: "wt-stalled", Branch: stalledBranch, Path: stalledRel, Status: colony.WorktreeInProgress, Phase: 2, CreatedAt: now, UpdatedAt: now},
		{ID: "wt-live", Branch: liveBranch, Path: liveRel, Status: colony.WorktreeInProgress, Phase: 5, CreatedAt: now, UpdatedAt: now},
	}
	writeColonyStateWithWorktrees(t, dataDir, worktrees)

	// Simulate resuming phase 5 — the caller has no phase filter today, and
	// intentionally still has none after this phase (CONTEXT.md).
	if _, _, err := gcOrphanedWorktrees(); err != nil {
		t.Fatalf("gcOrphanedWorktrees returned error: %v", err)
	}

	if _, statErr := os.Stat(stalledPath); statErr != nil {
		t.Fatalf("expected stalled phase-2 worktree directory to still exist, stat error: %v", statErr)
	}
	branchOut, err := exec.Command("git", "-C", root, "branch", "--list", stalledBranch).CombinedOutput()
	if err != nil {
		t.Fatalf("git branch --list: %v: %s", err, branchOut)
	}
	if strings.TrimSpace(string(branchOut)) == "" {
		t.Fatalf("expected stalled phase-2 branch %q to still exist", stalledBranch)
	}
}

// ---------------------------------------------------------------------------
// TestGCNeverCallsRemoveGitWorktree
// ---------------------------------------------------------------------------

// TestGCNeverCallsRemoveGitWorktree is a source-level ratchet: it isolates
// the body of gcOrphanedWorktrees, strips // comments, and fails if the
// remainder still contains removeGitWorktree. This guards against a future
// refactor quietly reinstating the deletion this phase removed.
func TestGCNeverCallsRemoveGitWorktree(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	src, err := os.ReadFile(filepath.Join(repoRoot, "cmd", "codex_build_worktree.go"))
	if err != nil {
		t.Fatalf("read cmd/codex_build_worktree.go: %v", err)
	}

	body, ok := extractFuncBody(string(src), "func gcOrphanedWorktrees(")
	if !ok {
		t.Fatalf("could not locate the body of gcOrphanedWorktrees in cmd/codex_build_worktree.go — a test that cannot locate what it is checking must not silently pass")
	}

	stripped := stripGoLineComments(body)
	if strings.Contains(stripped, "removeGitWorktree") {
		t.Errorf("gcOrphanedWorktrees's body still contains a call to removeGitWorktree outside of comments — this reintroduces the exact destructive path this phase removed")
	}
}

// extractFuncBody finds the top-level function whose signature starts with
// signaturePrefix (e.g. "func gcOrphanedWorktrees(") and returns everything
// from that line through the matching closing brace at column 0.
func extractFuncBody(src, signaturePrefix string) (string, bool) {
	lines := strings.Split(src, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, signaturePrefix) {
			start = i
			break
		}
	}
	if start == -1 {
		return "", false
	}
	for i := start + 1; i < len(lines); i++ {
		if lines[i] == "}" {
			return strings.Join(lines[start:i+1], "\n"), true
		}
	}
	return "", false
}

// stripGoLineComments removes the // portion of every line, without trying
// to handle string literals containing "//" — sufficient for a ratchet
// scanning this codebase's own source, which does not use "//" inside
// string literals in this function.
func stripGoLineComments(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// TestWorktreeReapHasNoLifecycleCaller
// ---------------------------------------------------------------------------

// TestWorktreeReapHasNoLifecycleCaller is T-187-13's ratchet: worktree-reap
// must have no caller other than a human typing it. It scans every named
// lifecycle source file for a call to runWorktreeReap or the literal
// command name and fails if either appears.
func TestWorktreeReapHasNoLifecycleCaller(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	lifecycleFiles := []string{
		"session_flow_cmds.go",
		"codex_continue.go",
		"init_cmd.go",
		"codex_build.go",
		"codex_build_finalize.go",
		"run_cmd.go",
	}

	scanned := 0
	var violations []string
	for _, name := range lifecycleFiles {
		path := filepath.Join(repoRoot, "cmd", name)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			t.Fatalf("read %s: %v", path, readErr)
		}
		scanned++
		content := string(data)
		if strings.Contains(content, "runWorktreeReap") || strings.Contains(content, "worktree-reap") {
			violations = append(violations, name)
		}
	}

	if scanned == 0 {
		t.Fatalf("scanned zero lifecycle files across %v — a test that finds nothing to check would pass vacuously forever", lifecycleFiles)
	}
	if len(violations) > 0 {
		t.Errorf("found a reference to runWorktreeReap or the literal command name \"worktree-reap\" in lifecycle file(s) %v — destruction must stay operator-invoked only; wiring the reaper into an automatic path must fail this test rather than silently ship", violations)
	}
}

// ---------------------------------------------------------------------------
// worktree-reap CLI tests
// ---------------------------------------------------------------------------

func reapFixture(t *testing.T) (root, dataDir string) {
	t.Helper()
	root, dataDir = crashSafetyFixture(t)
	saveGlobals(t)
	resetRootCmd(t)
	return root, dataDir
}

func TestWorktreeReapWithoutForceDeletesNothing(t *testing.T) {
	root, dataDir := reapFixture(t)

	branch := "phase-1/builder-clean-reap"
	wtPath, relPath := addCrashSafetyWorktree(t, root, branch, "phase-1-builder-clean-reap")

	now := time.Now().UTC().Format(time.RFC3339)
	entry := colony.WorktreeEntry{
		ID:        "wt-reap-clean",
		Branch:    branch,
		Path:      relPath,
		Status:    colony.WorktreeInProgress,
		Phase:     1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	writeColonyStateWithWorktrees(t, dataDir, []colony.WorktreeEntry{entry})

	statePath := filepath.Join(dataDir, "COLONY_STATE.json")
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state before: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	rootCmd.SetArgs([]string{"worktree-reap"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("worktree-reap returned error: %v", err)
	}

	if _, statErr := os.Stat(wtPath); statErr != nil {
		t.Fatalf("expected worktree directory to still exist without --force, stat error: %v", statErr)
	}

	listOut, err := exec.Command("git", "-C", root, "worktree", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("git worktree list: %v: %s", err, listOut)
	}
	if !strings.Contains(string(listOut), wtPath) {
		t.Errorf("expected %q to still be registered in git worktree list, got: %s", wtPath, listOut)
	}

	branchOut, err := exec.Command("git", "-C", root, "branch", "--list", branch).CombinedOutput()
	if err != nil {
		t.Fatalf("git branch --list: %v: %s", err, branchOut)
	}
	if strings.TrimSpace(string(branchOut)) == "" {
		t.Errorf("expected branch %q to still exist without --force", branch)
	}

	after, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state after: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("expected COLONY_STATE.json bytes unchanged by a read-only worktree-reap call, but they differ")
	}
}

func TestWorktreeReapForceKeepsUnmergedWork(t *testing.T) {
	root, dataDir := reapFixture(t)

	cleanBranch := "phase-1/builder-clean-force"
	cleanPath, cleanRel := addCrashSafetyWorktree(t, root, cleanBranch, "phase-1-builder-clean-force")

	dirtyBranch := "phase-1/builder-dirty-force"
	dirtyPath, dirtyRel := addCrashSafetyWorktree(t, root, dirtyBranch, "phase-1-builder-dirty-force")
	const dirtyContent = "unmerged precious content\n"
	if err := os.WriteFile(filepath.Join(dirtyPath, "precious.txt"), []byte(dirtyContent), 0644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{ID: "wt-clean", Branch: cleanBranch, Path: cleanRel, Status: colony.WorktreeInProgress, Phase: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "wt-dirty", Branch: dirtyBranch, Path: dirtyRel, Status: colony.WorktreeInProgress, Phase: 1, CreatedAt: now, UpdatedAt: now},
	}
	writeColonyStateWithWorktrees(t, dataDir, worktrees)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	rootCmd.SetArgs([]string{"worktree-reap", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("worktree-reap --force returned error: %v", err)
	}

	if _, statErr := os.Stat(cleanPath); statErr == nil {
		t.Errorf("expected clean worktree directory to be removed, but it still exists")
	}
	cleanBranchOut, _ := exec.Command("git", "-C", root, "branch", "--list", cleanBranch).CombinedOutput()
	if strings.TrimSpace(string(cleanBranchOut)) != "" {
		t.Errorf("expected clean branch %q to be removed, got: %s", cleanBranch, cleanBranchOut)
	}

	if _, statErr := os.Stat(dirtyPath); statErr != nil {
		t.Fatalf("expected dirty worktree directory to survive --force alone, stat error: %v", statErr)
	}
	content, readErr := os.ReadFile(filepath.Join(dirtyPath, "precious.txt"))
	if readErr != nil {
		t.Fatalf("read dirty worktree's file: %v", readErr)
	}
	if string(content) != dirtyContent {
		t.Errorf("expected dirty content to still be recoverable in the working tree, got: %q", string(content))
	}
}

func TestWorktreeReapIncludeUnmergedStashesBeforeDestroying(t *testing.T) {
	root, dataDir := reapFixture(t)

	dirtyBranch := "phase-1/builder-dirty-consent"
	dirtyPath, dirtyRel := addCrashSafetyWorktree(t, root, dirtyBranch, "phase-1-builder-dirty-consent")
	const dirtyContent = "consented-to-destroy content\n"
	if err := os.WriteFile(filepath.Join(dirtyPath, "precious.txt"), []byte(dirtyContent), 0644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	entry := colony.WorktreeEntry{
		ID:        "wt-dirty-consent",
		Branch:    dirtyBranch,
		Path:      dirtyRel,
		Status:    colony.WorktreeInProgress,
		Phase:     1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	writeColonyStateWithWorktrees(t, dataDir, []colony.WorktreeEntry{entry})

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	rootCmd.SetArgs([]string{"worktree-reap", "--force", "--include-unmerged"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("worktree-reap --force --include-unmerged returned error: %v", err)
	}

	if _, statErr := os.Stat(dirtyPath); statErr == nil {
		t.Errorf("expected dirty worktree to be removed under --force --include-unmerged, but it still exists")
	}

	// The content must still be recoverable from a stash on the main repo
	// (git worktree remove deletes the worktree's own directory, but a
	// stash created before removal lives in the shared repository, not the
	// worktree's private working copy, so it survives).
	stashListOut, err := exec.Command("git", "-C", root, "stash", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("git stash list: %v: %s", err, stashListOut)
	}
	if strings.TrimSpace(string(stashListOut)) == "" {
		t.Fatal("expected a non-empty stash after destroying dirty work with consent — destruction with consent must not lose bytes")
	}
}

func TestWorktreeReapIncludeUnmergedAloneDoesNothing(t *testing.T) {
	root, dataDir := reapFixture(t)

	dirtyBranch := "phase-1/builder-dirty-alone"
	dirtyPath, dirtyRel := addCrashSafetyWorktree(t, root, dirtyBranch, "phase-1-builder-dirty-alone")
	if err := os.WriteFile(filepath.Join(dirtyPath, "precious.txt"), []byte("should stay\n"), 0644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	entry := colony.WorktreeEntry{
		ID:        "wt-dirty-alone",
		Branch:    dirtyBranch,
		Path:      dirtyRel,
		Status:    colony.WorktreeInProgress,
		Phase:     1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	writeColonyStateWithWorktrees(t, dataDir, []colony.WorktreeEntry{entry})

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	rootCmd.SetArgs([]string{"worktree-reap", "--include-unmerged"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("worktree-reap --include-unmerged returned error: %v", err)
	}

	if _, statErr := os.Stat(dirtyPath); statErr != nil {
		t.Fatalf("expected dirty worktree to survive --include-unmerged without --force, stat error: %v", statErr)
	}
	content, readErr := os.ReadFile(filepath.Join(dirtyPath, "precious.txt"))
	if readErr != nil {
		t.Fatalf("read dirty worktree's file: %v", readErr)
	}
	if string(content) != "should stay\n" {
		t.Errorf("expected file content unchanged, got: %q", string(content))
	}
}
