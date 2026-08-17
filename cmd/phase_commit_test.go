package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// WS2 — the save-point after every verified phase. Real git repos, real
// commits; the seam (phaseCommitGitRunner) is stubbed only where the test is
// about argv or failure behaviour, never to fake success.

func phaseCommitTestState(goal string, phaseCount int, mode colony.PhaseCommitMode) colony.ColonyState {
	state := colony.ColonyState{PhaseCommits: mode}
	state.Goal = &goal
	for i := 1; i <= phaseCount; i++ {
		state.Plan.Phases = append(state.Plan.Phases, colony.Phase{ID: i, Name: fmt.Sprintf("Phase %d", i)})
	}
	return state
}

// initPhaseCommitRepo sets up a store inside a real git repo and returns the
// repo root. The store's data dir lives under the repo, exactly as in a real
// colony.
func initPhaseCommitRepo(t *testing.T) string {
	t.Helper()
	s, tmpDir := newTestStore(t)
	store = s
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@aether.local"},
		{"config", "user.name", "Aether Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	for _, args := range [][]string{{"add", "README.md"}, {"commit", "-q", "-m", "seed"}} {
		cmd := exec.Command("git", append([]string{"-C", tmpDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	return tmpDir
}

func writePhaseHandoffs(t *testing.T, phase int, files ...string) {
	t.Helper()
	if err := store.SaveJSON(workerHandoffsPath, workerHandoffFile{Entries: []workerHandoffRecord{
		{ID: "h1", Phase: phase, WorkerName: "Mason-1", ChangedFiles: files},
	}}); err != nil {
		t.Fatalf("write handoffs: %v", err)
	}
}

func gitOut(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v (%s)", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestPhaseAdvanceCreatesCommit(t *testing.T) {
	saveGlobals(t)
	root := initPhaseCommitRepo(t)

	if err := os.WriteFile(filepath.Join(root, "feature.go"), []byte("package x\n"), 0644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	writePhaseHandoffs(t, 3, "feature.go")
	state := phaseCommitTestState("Build the widget factory", 7, "")

	result := commitPhaseAdvance(root, state, colony.Phase{ID: 3, Name: "Wire the widgets"})
	if !result.Committed {
		t.Fatalf("phase advance did not commit: %+v", result)
	}

	subject := gitOut(t, root, "log", "-1", "--format=%s")
	if !regexp.MustCompile(`^aether\(phase-\d+\): `).MatchString(subject) {
		t.Fatalf("commit subject not greppable for the Archaeologist: %q", subject)
	}
	trailer := gitOut(t, root, "log", "-1", "--format=%(trailers:key=Aether-Phase,valueonly)")
	if strings.TrimSpace(trailer) != "3" {
		t.Fatalf("Aether-Phase trailer = %q, want 3", trailer)
	}
	committed := gitOut(t, root, "show", "--name-only", "--format=", "HEAD")
	if committed != "feature.go" {
		t.Fatalf("committed tree = %q, want exactly feature.go", committed)
	}
	body := gitOut(t, root, "log", "-1", "--format=%b")
	if !strings.Contains(body, "Build the widget factory") || !strings.Contains(body, "3 of 7") {
		t.Fatalf("commit body missing colony context: %q", body)
	}
}

func TestPhaseCommitOffSwitchSuppresses(t *testing.T) {
	saveGlobals(t)
	root := initPhaseCommitRepo(t)
	head := gitOut(t, root, "rev-parse", "HEAD")

	if err := os.WriteFile(filepath.Join(root, "feature.go"), []byte("package x\n"), 0644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	writePhaseHandoffs(t, 1, "feature.go")
	state := phaseCommitTestState("goal", 2, colony.PhaseCommitsOff)

	result := commitPhaseAdvance(root, state, colony.Phase{ID: 1, Name: "First"})
	if result.Committed || result.SkipReason == "" {
		t.Fatalf("off switch did not suppress the commit: %+v", result)
	}
	if got := gitOut(t, root, "rev-parse", "HEAD"); got != head {
		t.Fatalf("HEAD moved with phase commits off: %s -> %s", head, got)
	}
}

// TestPhaseCommitDoesNotSweepUnrelatedDirtyFile is the test the
// pathspec-commit choice exists to pass: the owner's stray edits — dirty,
// untracked, and even already-STAGED files — must never ride into a colony
// commit.
func TestPhaseCommitDoesNotSweepUnrelatedDirtyFile(t *testing.T) {
	saveGlobals(t)
	root := initPhaseCommitRepo(t)

	// The worker's file.
	if err := os.WriteFile(filepath.Join(root, "feature.go"), []byte("package x\n"), 0644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	// The owner's dirty tracked file.
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("owner draft edit\n"), 0644); err != nil {
		t.Fatalf("dirty README: %v", err)
	}
	// The owner's untracked scratch file.
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("private notes\n"), 0644); err != nil {
		t.Fatalf("write notes: %v", err)
	}
	// The owner's already-staged file — the hardest case: plain `git commit`
	// would include it.
	if err := os.WriteFile(filepath.Join(root, "staged.txt"), []byte("staged by owner\n"), 0644); err != nil {
		t.Fatalf("write staged: %v", err)
	}
	gitOut(t, root, "add", "staged.txt")

	writePhaseHandoffs(t, 1, "feature.go")
	state := phaseCommitTestState("goal", 1, "")
	result := commitPhaseAdvance(root, state, colony.Phase{ID: 1, Name: "First"})
	if !result.Committed {
		t.Fatalf("expected a commit: %+v", result)
	}

	committed := gitOut(t, root, "show", "--name-only", "--format=", "HEAD")
	if committed != "feature.go" {
		t.Fatalf("colony commit swept owner files: %q", committed)
	}
	status := gitOut(t, root, "status", "--porcelain")
	for _, want := range []string{"README.md", "notes.txt", "staged.txt"} {
		if !strings.Contains(status, want) {
			t.Fatalf("owner file %s vanished from the working state after the phase commit:\n%s", want, status)
		}
	}
}

func TestPhaseCommitFailureNeverBlocksAdvance(t *testing.T) {
	saveGlobals(t)
	root := initPhaseCommitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "feature.go"), []byte("package x\n"), 0644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	writePhaseHandoffs(t, 1, "feature.go")
	state := phaseCommitTestState("goal", 1, "")

	origRunner := phaseCommitGitRunner
	defer func() { phaseCommitGitRunner = origRunner }()
	phaseCommitGitRunner = func(gitRoot string, args ...string) (string, error) {
		if args[0] == "commit" {
			return "simulated hook rejection", fmt.Errorf("exit status 1")
		}
		return origRunner(gitRoot, args...)
	}

	result := commitPhaseAdvance(root, state, colony.Phase{ID: 1, Name: "First"})
	if result.Committed {
		t.Fatalf("stubbed failure still reported a commit")
	}
	if result.Err == "" {
		t.Fatalf("commit failure was silent: %+v", result)
	}
	// The failure writes the autopilot pause marker so unattended runs stop
	// stacking phases on a broken tree.
	if _, err := os.Stat(filepath.Join(store.BasePath(), uncommittedChangesMarkerFile)); err != nil {
		t.Fatalf("commit failure did not write the autopilot pause marker: %v", err)
	}
}

func TestPhaseCommitNeverPushes(t *testing.T) {
	saveGlobals(t)
	root := initPhaseCommitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "feature.go"), []byte("package x\n"), 0644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	writePhaseHandoffs(t, 1, "feature.go")
	state := phaseCommitTestState("goal", 1, "")

	allowed := map[string]bool{"rev-parse": true, "ls-files": true, "status": true, "add": true, "commit": true}
	var recorded [][]string
	origRunner := phaseCommitGitRunner
	defer func() { phaseCommitGitRunner = origRunner }()
	phaseCommitGitRunner = func(gitRoot string, args ...string) (string, error) {
		recorded = append(recorded, args)
		return origRunner(gitRoot, args...)
	}

	result := commitPhaseAdvance(root, state, colony.Phase{ID: 1, Name: "First"})
	if !result.Committed {
		t.Fatalf("expected a commit: %+v", result)
	}
	if len(recorded) == 0 {
		t.Fatalf("seam recorded no git invocations")
	}
	for _, args := range recorded {
		if !allowed[args[0]] {
			t.Fatalf("phase commit ran a git subcommand outside the allowed set: %v", args)
		}
		for _, token := range args {
			if strings.Contains(token, "push") {
				t.Fatalf("phase commit invoked push: %v", args)
			}
		}
		// The add must name explicit paths — a blanket add is exactly the
		// sweep the pathspec design forbids.
		if args[0] == "add" {
			for _, token := range args[1:] {
				if token == "-A" || token == "--all" || token == "." {
					t.Fatalf("phase commit staged with a blanket add: %v", args)
				}
			}
		}
	}
}

func TestPhaseCommitSkipsCleanWorktreeMode(t *testing.T) {
	saveGlobals(t)
	root := initPhaseCommitRepo(t)

	// Worktree-mode shape: the merge-back at build-finalize already landed
	// and committed the worker's file; continue-time finds a clean tree.
	if err := os.WriteFile(filepath.Join(root, "feature.go"), []byte("package x\n"), 0644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	gitOut(t, root, "add", "feature.go")
	gitOut(t, root, "commit", "-q", "-m", "merged by worktree sync-back")
	head := gitOut(t, root, "rev-parse", "HEAD")

	writePhaseHandoffs(t, 1, "feature.go")
	state := phaseCommitTestState("goal", 1, "")
	result := commitPhaseAdvance(root, state, colony.Phase{ID: 1, Name: "First"})
	if result.Committed || result.Err != "" {
		t.Fatalf("clean tree should skip cleanly: %+v", result)
	}
	if !strings.Contains(result.SkipReason, "no uncommitted changes") {
		t.Fatalf("skip reason = %q", result.SkipReason)
	}
	if got := gitOut(t, root, "rev-parse", "HEAD"); got != head {
		t.Fatalf("HEAD moved on a clean skip")
	}
}

func TestPhaseCommitSkipsOutsideGitRepo(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	store = s
	writePhaseHandoffs(t, 1, "feature.go")
	state := phaseCommitTestState("goal", 1, "")
	result := commitPhaseAdvance(tmpDir, state, colony.Phase{ID: 1, Name: "First"})
	if result.Committed || result.Err != "" {
		t.Fatalf("non-git dir should skip cleanly: %+v", result)
	}
	if !strings.Contains(result.SkipReason, "not a git repository") {
		t.Fatalf("skip reason = %q", result.SkipReason)
	}
}
