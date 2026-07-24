package codex

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWorkspaceFingerprintTracksCheckoutAndBranchNotFileContents(t *testing.T) {
	root := t.TempDir()
	runGitForBindingTest(t, root, "init")
	runGitForBindingTest(t, root, "config", "user.email", "aether@example.test")
	runGitForBindingTest(t, root, "config", "user.name", "Aether Test")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("one\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitForBindingTest(t, root, "add", "tracked.txt")
	runGitForBindingTest(t, root, "commit", "-m", "initial")

	first, err := WorkspaceFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("worker edit\n"), 0644); err != nil {
		t.Fatal(err)
	}
	second, err := WorkspaceFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("ordinary worker edits must not invalidate the workspace identity")
	}
	runGitForBindingTest(t, root, "switch", "-c", "other-branch")
	third, err := WorkspaceFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if third == first {
		t.Fatal("branch change must invalidate the workspace identity")
	}
}

func TestExecutionBindingValidationRejectsIncompleteIdentity(t *testing.T) {
	binding := ExecutionBinding{SchemaVersion: ExecutionBindingSchemaVersion}
	if err := binding.Validate(); err == nil {
		t.Fatal("incomplete execution binding should be rejected")
	}
}

func runGitForBindingTest(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
