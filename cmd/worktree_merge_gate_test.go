package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// worktree-merge-back ecosystem-neutrality tests
//
// These tests exercise the resolveTestCommand()-driven gate added to
// worktreeMergeBackCmd in place of the hard-coded `go test ./...` literal.
// They follow the exact fixture shape used by TestWorktreeMergeBackSuccess
// and TestWorktreeMergeBackTestsFail (cmd/worktree_test.go): real git repos
// via runGit, a real `git worktree add`, real AETHER_ROOT, and the actual
// cobra command executed through rootCmd.Execute().
//
// Fixture rooting note: resolveTestCommand() resolves the repo root as the
// parent of the parent of store.BasePath(). With dataDir := tmpDir +
// "/.aether/data", the resolved root is tmpDir, so CLAUDE.md / package.json
// must be written at tmpDir, not inside the worktree.
// ---------------------------------------------------------------------------

func TestWorktreeMergeBackUsesProjectTestCommandInNodeRepo(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	binding := bindCommandTestRepository(t)
	tmpDir := binding.Root
	dataDir := binding.DataDir

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "checkout", "-b", "main")

	// NO go.mod anywhere. extractTestCommand() only recognizes the literal
	// substrings "go test", "npm test", and "cargo test" -- so put "npm
	// test" in CLAUDE.md at tmpDir. This exercises the resolver's first
	// tier deterministically without requiring npm to be installed: the
	// test only asserts that the resolved command ran (not "go test"),
	// not that it succeeded via a real npm invocation. A package.json
	// fixture would risk depending on npm being present in the
	// environment; CLAUDE.md keeps the command fully under this test's
	// control per the resolver's own documented priority (CLAUDE.md
	// before language sniff).
	os.WriteFile(tmpDir+"/CLAUDE.md", []byte("# Project\n\nRun tests: npm test\n"), 0644)
	os.WriteFile(tmpDir+"/package.json", []byte(`{"name":"test","scripts":{"test":"exit 0"}}`), 0644)
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	runGit(t, tmpDir, "worktree", "add", "-b", "phase-1/builder-node", tmpDir+"/.aether/worktrees/phase-1-builder-node", "HEAD")
	wtPath := tmpDir + "/.aether/worktrees/phase-1-builder-node"
	os.WriteFile(wtPath+"/notes.txt", []byte("node work\n"), 0644)
	runGit(t, wtPath, "add", ".")
	runGit(t, wtPath, "commit", "-m", "add node work")

	now := time.Now().UTC().Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_node_001",
			Branch:    "phase-1/builder-node",
			Path:      ".aether/worktrees/phase-1-builder-node",
			Status:    colony.WorktreeInProgress,
			Phase:     1,
			Agent:     "builder-node",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	state := makeTestStateWithWorktrees(worktrees)
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	s := binding.Store

	rootCmd.SetArgs([]string{"worktree-merge-back", "--branch", "phase-1/builder-node"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stdoutOutput := stdoutBuf.String()
	if strings.Contains(strings.ToLower(stdoutOutput), "go test") || strings.Contains(strings.ToLower(stderrBuf.String()), "go test") {
		t.Errorf("expected no mention of go test, got stdout=%q stderr=%q", stdoutOutput, stderrBuf.String())
	}

	var reloadState colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &reloadState); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	found := false
	for _, wt := range reloadState.Worktrees {
		if wt.Branch == "phase-1/builder-node" {
			found = true
			if wt.Status != colony.WorktreeMerged {
				t.Errorf("expected status merged, got %s", wt.Status)
			}
		}
	}
	if !found {
		t.Fatal("expected worktree entry for phase-1/builder-node in reloaded state")
	}
}

func TestWorktreeMergeBackRefusesWhenTestCommandUnknown(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stderrBuf bytes.Buffer
	stderr = &stderrBuf

	binding := bindCommandTestRepository(t)
	tmpDir := binding.Root
	dataDir := binding.DataDir

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "checkout", "-b", "main")

	// No go.mod, no package.json, no Cargo.toml, no pom.xml, no CLAUDE.md,
	// no .aether/data/codebase.md -- resolveTestCommand() must return "".
	os.WriteFile(tmpDir+"/README.md", []byte("# unresolvable project\n"), 0644)
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	runGit(t, tmpDir, "worktree", "add", "-b", "phase-1/builder-unknown", tmpDir+"/.aether/worktrees/phase-1-builder-unknown", "HEAD")
	wtPath := tmpDir + "/.aether/worktrees/phase-1-builder-unknown"
	os.WriteFile(wtPath+"/notes.txt", []byte("unresolvable work\n"), 0644)
	runGit(t, wtPath, "add", ".")
	runGit(t, wtPath, "commit", "-m", "add work")

	now := time.Now().UTC().Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_unknown_001",
			Branch:    "phase-1/builder-unknown",
			Path:      ".aether/worktrees/phase-1-builder-unknown",
			Status:    colony.WorktreeInProgress,
			Phase:     1,
			Agent:     "builder-unknown",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	state := makeTestStateWithWorktrees(worktrees)
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	s := binding.Store

	rootCmd.SetArgs([]string{"worktree-merge-back", "--branch", "phase-1/builder-unknown"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stderrBuf.String()
	if !strings.Contains(output, "cannot determine how to test this project") {
		t.Errorf("expected 'cannot determine how to test this project' in stderr, got: %s", output)
	}

	var reloadState colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &reloadState); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	for _, wt := range reloadState.Worktrees {
		if wt.Branch == "phase-1/builder-unknown" && wt.Status == colony.WorktreeMerged {
			t.Errorf("expected status NOT merged, got %s", wt.Status)
		}
	}

	var ff colony.FlagsFile
	if err := s.LoadJSON("pending-decisions.json", &ff); err != nil {
		t.Fatalf("expected pending-decisions.json to exist: %v", err)
	}
	found := false
	for _, d := range ff.Decisions {
		if d.Type == "blocker" && strings.Contains(d.Description, "cannot determine how to test this project") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected blocker flag for undeterminable test command")
	}
}

func TestWorktreeMergeBackRefusalPreservesTheWorktree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stderrBuf bytes.Buffer
	stderr = &stderrBuf

	binding := bindCommandTestRepository(t)
	tmpDir := binding.Root
	dataDir := binding.DataDir

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "checkout", "-b", "main")

	os.WriteFile(tmpDir+"/README.md", []byte("# unresolvable project\n"), 0644)
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	branch := "phase-1/builder-preserve"
	wtPath := tmpDir + "/.aether/worktrees/phase-1-builder-preserve"
	runGit(t, tmpDir, "worktree", "add", "-b", branch, wtPath, "HEAD")

	// A committed change on the worktree branch -- the work that must not
	// be destroyed by the refusal.
	os.WriteFile(wtPath+"/precious.txt", []byte("do not lose this\n"), 0644)
	runGit(t, wtPath, "add", ".")
	runGit(t, wtPath, "commit", "-m", "precious work")

	now := time.Now().UTC().Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_preserve_001",
			Branch:    branch,
			Path:      ".aether/worktrees/phase-1-builder-preserve",
			Status:    colony.WorktreeInProgress,
			Phase:     1,
			Agent:     "builder-preserve",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	state := makeTestStateWithWorktrees(worktrees)
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	rootCmd.SetArgs([]string{"worktree-merge-back", "--branch", branch})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stderrBuf.String(), "cannot determine how to test this project") {
		t.Fatalf("expected refusal message, got: %s", stderrBuf.String())
	}

	// The worktree directory must still exist on disk.
	if _, statErr := os.Stat(wtPath); statErr != nil {
		t.Errorf("expected worktree directory to still exist at %s: %v", wtPath, statErr)
	}

	// git worktree list must still contain the worktree path.
	listOut, listErr := exec.Command("git", "-C", tmpDir, "worktree", "list").CombinedOutput()
	if listErr != nil {
		t.Fatalf("git worktree list failed: %v: %s", listErr, string(listOut))
	}
	if !strings.Contains(string(listOut), wtPath) {
		t.Errorf("expected git worktree list to contain %s, got: %s", wtPath, string(listOut))
	}

	// git branch --list must still show the branch.
	branchOut, branchErr := exec.Command("git", "-C", tmpDir, "branch", "--list", branch).CombinedOutput()
	if branchErr != nil {
		t.Fatalf("git branch --list failed: %v: %s", branchErr, string(branchOut))
	}
	if strings.TrimSpace(string(branchOut)) == "" {
		t.Errorf("expected git branch --list %s to be non-empty", branch)
	}
}

func TestWorktreeMergeBackStillWorksInGoRepo(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	binding := bindCommandTestRepository(t)
	tmpDir := binding.Root
	dataDir := binding.DataDir

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "checkout", "-b", "main")

	os.WriteFile(tmpDir+"/go.mod", []byte("module test\n\ngo 1.22\n"), 0644)
	os.MkdirAll(tmpDir+"/cmd", 0755)
	os.WriteFile(tmpDir+"/cmd/testhelper_test.go", []byte(`package cmd
import "testing"
func TestHelper(t *testing.T) {}
`), 0644)
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	runGit(t, tmpDir, "worktree", "add", "-b", "phase-1/builder-goregress", tmpDir+"/.aether/worktrees/phase-1-builder-goregress", "HEAD")
	wtPath := tmpDir + "/.aether/worktrees/phase-1-builder-goregress"
	os.WriteFile(wtPath+"/cmd/newfile_test.go", []byte(`package cmd
import "testing"
func TestNewFile(t *testing.T) {}
`), 0644)
	runGit(t, wtPath, "add", ".")
	runGit(t, wtPath, "commit", "-m", "add new test")

	now := time.Now().UTC().Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_goregress_001",
			Branch:    "phase-1/builder-goregress",
			Path:      ".aether/worktrees/phase-1-builder-goregress",
			Status:    colony.WorktreeInProgress,
			Phase:     1,
			Agent:     "builder-goregress",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	state := makeTestStateWithWorktrees(worktrees)
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	s := binding.Store

	rootCmd.SetArgs([]string{"worktree-merge-back", "--branch", "phase-1/builder-goregress"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var reloadState colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &reloadState); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	found := false
	for _, wt := range reloadState.Worktrees {
		if wt.Branch == "phase-1/builder-goregress" {
			found = true
			if wt.Status != colony.WorktreeMerged {
				t.Errorf("expected status merged, got %s (stderr=%s)", wt.Status, stderrBuf.String())
			}
		}
	}
	if !found {
		t.Fatal("expected worktree entry for phase-1/builder-goregress in reloaded state")
	}
}

func TestWorktreeMergeBackHelpTextDoesNotPromiseGo(t *testing.T) {
	if strings.Contains(worktreeMergeBackCmd.Long, "go test") {
		t.Errorf("expected worktreeMergeBackCmd.Long to not contain 'go test', got: %s", worktreeMergeBackCmd.Long)
	}
}
