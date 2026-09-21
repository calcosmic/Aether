package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// pauseDigestTestRepo makes a real git repository with one commit, because
// repositoryDirtyDigest works from real `git status` output.
func pauseDigestTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-q", "-m", "init")
	return root
}

// TestPauseFingerprintSurvivesAFolderShortcut is the regression for a real
// downstream failure (2026-09-21): a project with unsaved shortcuts (symlinks)
// to folders could not be paused at all, because the fingerprint followed each
// shortcut and tried to read the folder it pointed at as a file.
func TestPauseFingerprintSurvivesAFolderShortcut(t *testing.T) {
	root := pauseDigestTestRepo(t)
	if err := os.MkdirAll(filepath.Join(root, "real", "aws-thing"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "devices"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "devices", "aws-thing")
	if err := os.Symlink(filepath.Join("..", "real", "aws-thing"), link); err != nil {
		t.Skipf("symlinks unavailable here: %v", err)
	}

	first, err := repositoryDirtyDigest(root)
	if err != nil {
		t.Fatalf("fingerprint with an unsaved folder shortcut = %v, want success", err)
	}

	// The shortcut is recorded as the shortcut itself, never followed: adding a
	// file inside the folder it points at changes that file's own entry, but
	// re-pointing the shortcut must change the fingerprint too.
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "real", "other"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "real", "other"), link); err != nil {
		t.Fatal(err)
	}
	second, err := repositoryDirtyDigest(root)
	if err != nil {
		t.Fatalf("fingerprint after re-pointing the shortcut = %v", err)
	}
	if first == second {
		t.Fatal("re-pointing a shortcut did not change the fingerprint; a resume could miss that change")
	}
}

// TestPauseFingerprintNeverReadsThroughAShortcut: a shortcut to a file outside
// the project must not pull that file's bytes into the fingerprint.
func TestPauseFingerprintNeverReadsThroughAShortcut(t *testing.T) {
	root := pauseDigestTestRepo(t)
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlinks unavailable here: %v", err)
	}
	before, err := repositoryDirtyDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("two, and different"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := repositoryDirtyDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("the fingerprint changed when a file OUTSIDE the project changed; the shortcut was followed")
	}
}

// TestPauseFingerprintSurvivesANestedProjectFolder: git reports a nested
// repository as a single path ending in "/", which is a folder, not a file.
func TestPauseFingerprintSurvivesANestedProjectFolder(t *testing.T) {
	root := pauseDigestTestRepo(t)
	nested := filepath.Join(root, "vendor-tool")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = nested
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("nested git init: %v: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(nested, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := repositoryDirtyDigest(root); err != nil {
		t.Fatalf("fingerprint with a nested project folder = %v, want success", err)
	}
}
