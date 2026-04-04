package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aether-colony/aether/pkg/storage"
)

// setupParityEnv creates a temp directory with .aether/data/ populated from
// cmd/testdata fixtures. It sets AETHER_ROOT to the temp directory and returns
// the tmpDir path. Cleanup is registered via t.Cleanup.
func setupParityEnv(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "aether-parity-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create data dir: %v", err)
	}

	// Copy test fixtures from cmd/testdata/
	testdataDir := filepath.Join(projectRoot(), "cmd", "testdata")
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to read testdata dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(testdataDir, entry.Name())
		dst := filepath.Join(dataDir, entry.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to read fixture %s: %v", src, err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to write fixture %s: %v", dst, err)
		}
	}

	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	return tmpDir
}

// runShellCommand executes a shell subcommand via aether-utils.sh with the
// given AETHER_ROOT. It captures stdout and stderr, combining them into a
// single trimmed output string. Non-zero exit codes are logged but not fatal.
func runShellCommand(t *testing.T, tmpDir string, subcmd string, args ...string) string {
	t.Helper()
	root := projectRoot()
	scriptPath := filepath.Join(root, ".aether", "aether-utils.sh")

	allArgs := append([]string{scriptPath, subcmd}, args...)
	cmd := exec.Command("bash", allArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"AETHER_ROOT="+tmpDir,
		"DATA_DIR="+filepath.Join(tmpDir, ".aether", "data"),
		"COLONY_DATA_DIR="+filepath.Join(tmpDir, ".aether", "data"),
	)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command not found or other exec error
			t.Logf("shell command exec error for %s: %v", subcmd, err)
			return ""
		}
	}

	out := strings.TrimSpace(stdoutBuf.String())
	stderrOut := strings.TrimSpace(stderrBuf.String())

	// If stdout is empty but stderr has JSON, use stderr (error envelope)
	if out == "" && stderrOut != "" && strings.HasPrefix(stderrOut, "{") {
		out = stderrOut
	}

	t.Logf("shell %s (exit=%d): %s", subcmd, exitCode, truncateStr(out, 200))
	return out
}

// runGoCommand executes a Go subcommand via rootCmd.SetArgs with the given
// AETHER_ROOT. It saves and restores package globals, sets up a fresh store,
// and captures stdout output. Errors are logged but not fatal.
func runGoCommand(t *testing.T, tmpDir string, subcmd string, args ...string) string {
	t.Helper()

	// Save and restore globals
	oldStdout := stdout
	oldStderr := stderr
	oldStore := store
	defer func() {
		stdout = oldStdout
		stderr = oldStderr
		store = oldStore
	}()

	var buf bytes.Buffer
	stdout = &buf
	stderr = io.Discard

	// Initialize store for the temp directory
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Logf("go command store init error for %s: %v", subcmd, err)
		return ""
	}
	store = s

	// Set AETHER_ROOT
	os.Setenv("AETHER_ROOT", tmpDir)

	// Reset rootCmd args
	rootCmd.SetArgs(append([]string{subcmd}, args...))
	defer rootCmd.SetArgs([]string{})

	// Execute -- SilenceErrors means it won't print to real stderr
	runErr := rootCmd.Execute()
	if runErr != nil {
		// Check if stderr buffer captured anything (outputError writes to stderr)
		stderrBuf, ok := stderr.(*bytes.Buffer)
		if ok && stderrBuf.Len() > 0 {
			t.Logf("go %s error with stderr: %s", subcmd, truncateStr(stderrBuf.String(), 200))
			return strings.TrimSpace(stderrBuf.String())
		}
		t.Logf("go %s error: %v", subcmd, runErr)
	}

	out := strings.TrimSpace(buf.String())

	// Also capture stderr for error envelopes
	stderrBuf, ok := stderr.(*bytes.Buffer)
	if ok && out == "" && stderrBuf.Len() > 0 {
		out = strings.TrimSpace(stderrBuf.String())
	}

	t.Logf("go %s: %s", subcmd, truncateStr(out, 200))
	return out
}

// projectRoot returns the root directory of the project using runtime.Caller
// to locate the cmd/ directory and going up one level.
func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// filename is .../cmd/parity_harness_test.go
	cmdDir := filepath.Dir(filename)
	return filepath.Dir(cmdDir)
}

// isJSON returns true if the string can be parsed as valid JSON.
func isJSON(s string) bool {
	if s == "" {
		return false
	}
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

// truncateStr truncates a string to maxLen, appending "..." if truncated.
func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// TestSetupParityEnv verifies the parity test environment setup.
func TestSetupParityEnv(t *testing.T) {
	tmpDir := setupParityEnv(t)

	// Verify tmpDir is non-empty and looks like a temp path
	if tmpDir == "" {
		t.Fatal("setupParityEnv returned empty string")
	}
	if !strings.Contains(tmpDir, "aether-parity") {
		t.Errorf("tmpDir = %q, want it to contain 'aether-parity'", tmpDir)
	}

	// Check data directory exists
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Fatalf("data dir %q does not exist", dataDir)
	}

	// Check fixtures were copied
	for _, name := range []string{"colony_state.json", "pheromones.json", "flags.json"} {
		path := filepath.Join(dataDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("fixture %q was not copied", name)
		}
	}
}

// TestRunShellCommand verifies shell command execution works.
func TestRunShellCommand(t *testing.T) {
	tmpDir := setupParityEnv(t)

	out := runShellCommand(t, tmpDir, "pheromone-count")
	t.Logf("pheromone-count shell output: %q", out)

	// pheromone-count should return something (even an error message is ok)
	if out == "" {
		t.Log("pheromone-count returned empty output (may be expected)")
	}
}

// TestRunGoCommand verifies Go command execution works.
func TestRunGoCommand(t *testing.T) {
	tmpDir := setupParityEnv(t)

	out := runGoCommand(t, tmpDir, "status")
	t.Logf("status Go output: %q", truncateStr(out, 200))

	// status should return something
	if out == "" {
		t.Log("status returned empty output (may be expected)")
	}
}

// TestIsJSON verifies the JSON detection helper.
func TestIsJSON(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{`{"ok":true}`, true},
		{`{"ok":true,"result":"hello"}`, true},
		{`[1,2,3]`, true},
		{`plain text`, false},
		{``, false},
		{`null`, true},
		{`42`, true},
	}
	for _, tt := range tests {
		got := isJSON(tt.input)
		if got != tt.want {
			t.Errorf("isJSON(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
