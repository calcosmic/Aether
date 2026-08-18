package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
// TestCleanupBuildWorktreesSurvivesCrashOnBuildPath
// ---------------------------------------------------------------------------

// TestCleanupBuildWorktreesSurvivesCrashOnBuildPath is CR-05's fail-then-pass
// proof. cleanupBuildWorktrees runs on EVERY build (cmd/codex_build.go,
// unconditionally after every dispatch) and selects exactly the
// Allocated/InProgress entries a same-phase crash leaves behind — the
// scenario this whole phase exists to make safe. Before the fix, it called
// removeGitWorktree directly with no safety guard at all. This test puts a
// dirty, uncommitted worktree belonging to the phase being cleaned up in
// front of cleanupBuildWorktrees and proves it survives, exactly as
// gcOrphanedWorktrees already proves for the resume/continue/init paths.
func TestCleanupBuildWorktreesSurvivesCrashOnBuildPath(t *testing.T) {
	root, dataDir := crashSafetyFixture(t)

	branch := "phase-4/builder-crashed-midbuild"
	wtPath, relPath := addCrashSafetyWorktree(t, root, branch, "phase-4-builder-crashed-midbuild")

	const uncommittedContent = "distinctive uncommitted content from a same-phase crash\n"
	if err := os.WriteFile(filepath.Join(wtPath, "uncommitted-work.txt"), []byte(uncommittedContent), 0644); err != nil {
		t.Fatalf("write uncommitted file: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	entry := colony.WorktreeEntry{
		ID:        "wt-crash-midbuild",
		Branch:    branch,
		Path:      relPath,
		Status:    colony.WorktreeInProgress, // exactly what a crash between dispatch and finalize leaves behind
		Phase:     4,
		CreatedAt: now,
		UpdatedAt: now,
	}
	writeColonyStateWithWorktrees(t, dataDir, []colony.WorktreeEntry{entry})

	cleaned, orphaned, err := cleanupBuildWorktrees(4)
	if err != nil {
		t.Fatalf("cleanupBuildWorktrees returned error: %v", err)
	}
	if orphaned < 1 {
		t.Errorf("expected orphaned >= 1 (the dirty worktree must be preserved, not destroyed), got orphaned=%d cleaned=%d", orphaned, cleaned)
	}

	// The worktree directory must still exist on disk.
	if _, statErr := os.Stat(wtPath); statErr != nil {
		t.Fatalf("expected worktree directory to still exist after cleanupBuildWorktrees, stat error: %v", statErr)
	}

	// The branch must still be a real git branch.
	branchOut, err := exec.Command("git", "-C", root, "branch", "--list", branch).CombinedOutput()
	if err != nil {
		t.Fatalf("git branch --list: %v: %s", err, branchOut)
	}
	if strings.TrimSpace(string(branchOut)) == "" {
		t.Fatalf("expected branch %q to still exist, got empty branch --list output", branch)
	}

	// The uncommitted content must be recoverable — either still present in
	// the working tree, or stashed and recoverable via `git stash pop`.
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
			t.Fatal("uncommitted content is neither present in the working tree nor in a stash — it was lost by cleanupBuildWorktrees")
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

// TestCleanupBuildWorktreesNeverCallsRemoveGitWorktree is CR-05's own
// source-level ratchet, the same shape as TestGCNeverCallsRemoveGitWorktree.
// cleanupBuildWorktrees runs on every build and previously called
// removeGitWorktree directly with no safety guard; this guards against a
// future refactor quietly reinstating that call.
func TestCleanupBuildWorktreesNeverCallsRemoveGitWorktree(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	src, err := os.ReadFile(filepath.Join(repoRoot, "cmd", "codex_build_worktree.go"))
	if err != nil {
		t.Fatalf("read cmd/codex_build_worktree.go: %v", err)
	}

	body, ok := extractFuncBody(string(src), "func cleanupBuildWorktrees(")
	if !ok {
		t.Fatalf("could not locate the body of cleanupBuildWorktrees in cmd/codex_build_worktree.go — a test that cannot locate what it is checking must not silently pass")
	}

	stripped := stripGoLineComments(body)
	if strings.Contains(stripped, "removeGitWorktree") {
		t.Errorf("cleanupBuildWorktrees's body still contains a call to removeGitWorktree outside of comments — this reintroduces the exact unguarded build-path destruction CR-05 removed")
	}
}

// TestRemoveGitWorktreeHasOnlySanctionedCallers is the review's requested
// wider ratchet: rather than trusting a comment to say who calls
// removeGitWorktree, it scans every non-test .go source file under cmd/ and
// asserts every call site is one of the three functions this phase leaves as
// sanctioned callers. A caller outside this set means either an unguarded
// lifecycle path was reintroduced, or this list (and the reasoning in the
// worktreeReapCmd doc comment explaining why each is safe) needs updating —
// either way, a human must look, not silently pass.
func TestRemoveGitWorktreeHasOnlySanctionedCallers(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	sanctionedFuncs := map[string]bool{
		"func removeGitWorktree(":     true, // the function's own definition
		"func allocateBuildWorktree(": true, // rollback of a worktree it just created, never registered as holding output
		"func finalizeBuildWorktree(": true, // runs only after a successful worker's changes are already synced to root
		"func runWorktreeReap(":       true, // the named, operator-invoked destruction command
	}

	cmdDir := filepath.Join(repoRoot, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		t.Fatalf("read cmd dir: %v", err)
	}

	scanned := 0
	var violations []string
	for _, dirEntry := range entries {
		name := dirEntry.Name()
		if dirEntry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(cmdDir, name))
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		content := string(data)
		if !strings.Contains(content, "removeGitWorktree") {
			continue
		}
		scanned++

		lines := strings.Split(content, "\n")
		currentFunc := ""
		for _, line := range lines {
			for sig := range sanctionedFuncs {
				if strings.HasPrefix(line, sig) {
					currentFunc = sig
					break
				}
			}
			// A new top-level function that is not in the sanctioned map
			// resets currentFunc so calls inside it are correctly attributed
			// as unsanctioned, not misread as still belonging to the prior
			// sanctioned function.
			if strings.HasPrefix(line, "func ") {
				if _, ok := sanctionedFuncs[funcSigPrefix(line)]; !ok {
					currentFunc = ""
				}
			}
			trimmed := strings.TrimSpace(line)
			if idx := strings.Index(trimmed, "//"); idx >= 0 {
				trimmed = trimmed[:idx]
			}
			if strings.Contains(trimmed, "removeGitWorktree(") && !strings.HasPrefix(trimmed, "func removeGitWorktree(") {
				if currentFunc == "" || !sanctionedFuncs[currentFunc] {
					violations = append(violations, fmt.Sprintf("%s: %s", name, strings.TrimSpace(line)))
				}
			}
		}
	}

	if scanned == 0 {
		t.Fatalf("scanned zero source files mentioning removeGitWorktree — a test that finds nothing to check would pass vacuously forever")
	}
	if len(violations) > 0 {
		t.Errorf("found call(s) to removeGitWorktree outside the sanctioned caller set %v: %v", sanctionedFuncsList(sanctionedFuncs), violations)
	}
}

// funcSigPrefix extracts the "func Name(" prefix from a function-declaration
// line, trimming trailing parameters and everything after the opening paren.
func funcSigPrefix(line string) string {
	idx := strings.Index(line, "(")
	if idx < 0 {
		return strings.TrimSpace(line)
	}
	return line[:idx+1]
}

func sanctionedFuncsList(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
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
// TestRemoveGitWorktreeDoesNotDeleteBranchWhenRemovalFails
// ---------------------------------------------------------------------------

// TestRemoveGitWorktreeDoesNotDeleteBranchWhenRemovalFails is CR-03's
// fail-then-pass proof. Before the fix, the three git commands inside
// removeGitWorktree ran unconditionally and only accumulated errors: when
// `git worktree remove` failed, `git branch -D` still ran and still
// succeeded (-D force-deletes even unmerged branches), so a caller reading
// the returned error as "nothing was destroyed" was wrong — the branch,
// carrying a real unmerged commit, was already gone.
//
// The failure mode is produced by making the worktree directory
// unwritable (chmod 0500) before removal. Real git (2.52) unregisters the
// worktree from `git worktree list` BEFORE it finishes deleting the
// directory's contents, so when the delete step then hits "Permission
// denied" on a file it cannot remove, `worktree remove` exits non-zero
// while the worktree is already unregistered — meaning `git branch -D`, if
// it still runs afterward, succeeds and destroys the branch. This was
// confirmed manually against the real git binary before writing this test:
// `git worktree lock` was tried first and does NOT reproduce the defect,
// because a genuinely locked/registered worktree also protects the branch
// from `branch -D` ("used by worktree at ...") — only the unregistered
// case actually loses data, which is what this test reproduces.
func TestRemoveGitWorktreeDoesNotDeleteBranchWhenRemovalFails(t *testing.T) {
	root, _ := crashSafetyFixture(t)

	branch := "phase-1/builder-unwritable"
	wtPath, _ := addCrashSafetyWorktree(t, root, branch, "phase-1-builder-unwritable")

	// A unique unmerged commit on the branch — if `branch -D` runs anyway,
	// this commit becomes unreachable and is exactly what CR-03 protects.
	if err := os.WriteFile(filepath.Join(wtPath, "unmerged.txt"), []byte("unique unmerged content\n"), 0644); err != nil {
		t.Fatalf("write unmerged file: %v", err)
	}
	runGit(t, wtPath, "add", ".")
	runGit(t, wtPath, "commit", "-m", "unmerged commit")

	shaOut, err := exec.Command("git", "-C", root, "rev-parse", branch).CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse %s: %v: %s", branch, err, shaOut)
	}
	sha := strings.TrimSpace(string(shaOut))

	// Make the worktree directory itself unwritable so git can unregister
	// the worktree but then fails partway through deleting its contents.
	if chmodErr := os.Chmod(wtPath, 0500); chmodErr != nil {
		t.Fatalf("chmod worktree dir: %v", chmodErr)
	}
	t.Cleanup(func() {
		_ = os.Chmod(wtPath, 0755) // restore so t.TempDir() cleanup can remove it
	})

	removeErr := removeGitWorktree(root, wtPath, branch)
	if removeErr == nil {
		t.Fatal("expected removeGitWorktree to return an error when the worktree directory cannot be fully deleted")
	}

	// The branch MUST still exist — a non-nil error must mean nothing was
	// destroyed.
	branchOut, branchErr := exec.Command("git", "-C", root, "branch", "--list", branch).CombinedOutput()
	if branchErr != nil {
		t.Fatalf("git branch --list: %v: %s", branchErr, branchOut)
	}
	if strings.TrimSpace(string(branchOut)) == "" {
		t.Fatalf("expected branch %q to still exist after a failed removal, but it is gone — removeGitWorktree deleted the branch even though it reported an error", branch)
	}

	// The unmerged commit must still be reachable via its own SHA.
	showOut, showErr := exec.Command("git", "-C", root, "cat-file", "-e", sha).CombinedOutput()
	if showErr != nil {
		t.Fatalf("expected commit %s to still be reachable after a failed removal, got: %v: %s", sha, showErr, showOut)
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

// TestWorktreeReapIncludeUnmergedRefusesUndeterminableState is CR-04's
// end-to-end fail-then-pass proof. It reproduces the exact defect: an entry
// with no branch recorded (CR-01's shape) still has a genuine `git worktree
// add`-created directory on disk, so `git worktree remove` on it actually
// SUCCEEDS in destroying the directory — only the trailing `branch -D ""`
// step fails, because there is no branch to delete. worktreeDestructionSafety
// cannot determine unmerged-commit state for an empty branch (Safe=false,
// DirtyFileCount=0, UnmergedCommitCount=0), landing preserveWorktreeWork in
// its `default` branch. Before the fix, that branch reported preserved=true
// having stashed nothing, worktree-reap discarded the boolean with `_,`, and
// proceeded to call removeGitWorktree — which genuinely deletes the worktree
// directory on disk even though it also returns a non-nil error (from the
// unrelated branch-delete failure). This test proves the directory itself
// must survive.
func TestWorktreeReapIncludeUnmergedRefusesUndeterminableState(t *testing.T) {
	root, dataDir := reapFixture(t)

	staleBranch := "" // CR-01's shape: a worktree with no branch recorded.
	wtPath, wtRel := addCrashSafetyWorktree(t, root, "phase-1/builder-undeterminable-src", "phase-1-builder-undeterminable")
	if err := os.WriteFile(filepath.Join(wtPath, "precious.txt"), []byte("must survive\n"), 0644); err != nil {
		t.Fatalf("write file in worktree: %v", err)
	}
	runGit(t, wtPath, "add", ".")
	runGit(t, wtPath, "commit", "-m", "commit before entry loses its branch field")

	now := time.Now().UTC().Format(time.RFC3339)
	entry := colony.WorktreeEntry{
		ID:        "wt-undeterminable",
		Branch:    staleBranch, // the entry itself claims no branch
		Path:      wtRel,
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

	// The directory and its content must still exist — an undeterminable
	// state must never be read as "saved, so it is safe to destroy".
	if _, statErr := os.Stat(wtPath); statErr != nil {
		t.Fatalf("expected the undeterminable-state worktree to survive --force --include-unmerged, but it is gone: %v", statErr)
	}
	content, readErr := os.ReadFile(filepath.Join(wtPath, "precious.txt"))
	if readErr != nil {
		t.Fatalf("read file in worktree: %v", readErr)
	}
	if string(content) != "must survive\n" {
		t.Errorf("expected file content unchanged, got: %q", string(content))
	}

	reloaded := loadColonyStateFixture(t, dataDir)
	found := false
	for _, wt := range reloaded.Worktrees {
		if wt.Path == wtRel {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected entry for path %q to still be present in COLONY_STATE.json after refusing to destroy it, got worktrees: %+v", wtRel, reloaded.Worktrees)
	}
}
