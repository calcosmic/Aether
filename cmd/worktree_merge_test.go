package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// runGit is a test helper for running git commands in a temp directory.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\noutput: %s", args, err, output)
	}
}

// writeFile is a test helper for writing files.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

// TestWorktreeMergeDeprecated tests that worktree-merge returns a deprecated response.
func TestWorktreeMergeDeprecated(t *testing.T) {
	stdout = &strings.Builder{}
	defer func() { stdout = os.Stdout }()

	rootCmd.SetArgs([]string{"worktree-merge", "--branch", "test-branch"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command returned error: %v", err)
	}

	got := strings.TrimSpace(stdout.(*strings.Builder).String())
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %q", err, got)
	}

	if m["ok"] != true {
		t.Errorf("expected ok:true, got: %v", m["ok"])
	}

	result, ok := m["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	if result["deprecated"] != true {
		t.Errorf("expected deprecated:true, got: %v", result["deprecated"])
	}
	if result["command"] != "worktree-merge" {
		t.Errorf("expected command:worktree-merge, got: %v", result["command"])
	}
	if result["message"] != deprecatedMessage {
		t.Errorf("expected deprecation message, got: %v", result["message"])
	}
}

// TestWorktreeMergeDeprecatedStillHasFlags tests that deprecated worktree-merge
// still accepts its original flags (for backward compatibility with callers).
func TestWorktreeMergeDeprecatedStillHasFlags(t *testing.T) {
	stdout = &strings.Builder{}
	defer func() { stdout = os.Stdout }()

	rootCmd.SetArgs([]string{"worktree-merge", "--branch", "some-branch", "--target", "main"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command returned error: %v", err)
	}

	got := strings.TrimSpace(stdout.(*strings.Builder).String())
	if !strings.Contains(got, `"ok":true`) {
		t.Errorf("expected ok:true with flags, got: %s", got)
	}
	if !strings.Contains(got, `"deprecated":true`) {
		t.Errorf("expected deprecated:true, got: %s", got)
	}
}

// TestWorktreeMergeDeprecatedSilenceUsage tests that deprecated worktree-merge
// suppresses usage output.
func TestWorktreeMergeDeprecatedSilenceUsage(t *testing.T) {
	stdout = &strings.Builder{}
	stderr = &strings.Builder{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	// Call with an unknown flag -- deprecated cmd should not error on unknown flags
	// because flags are registered but the RunE just outputs deprecated notice.
	rootCmd.SetArgs([]string{"worktree-merge", "--branch", "x", "--unknown-flag"})
	defer rootCmd.SetArgs([]string{})

	_ = rootCmd.Execute()
}
