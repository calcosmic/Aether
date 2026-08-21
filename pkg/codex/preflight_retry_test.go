package codex

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// writeCountingProbe creates a stand-in provider binary that records how many
// times it ran, hangs on the runs listed in hangOn, and exits 0 otherwise.
func writeCountingProbe(t *testing.T, hangOn map[int]bool, exitCode int) (binary, counter string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell probe stand-in is POSIX-only")
	}
	dir := t.TempDir()
	binary = filepath.Join(dir, "probe.sh")
	counter = filepath.Join(dir, "runs")

	var hangs []string
	for run := range hangOn {
		hangs = append(hangs, strconv.Itoa(run))
	}

	script := fmt.Sprintf(`#!/bin/sh
COUNTER=%q
COUNT=$(cat "$COUNTER" 2>/dev/null || echo 0)
COUNT=$((COUNT + 1))
printf '%%s' "$COUNT" > "$COUNTER"
for hang in %s; do
  if [ "$COUNT" = "$hang" ]; then
    sleep 30
    exit 1
  fi
done
exit %d
`, counter, strings.Join(hangs, " "), exitCode)

	if err := os.WriteFile(binary, []byte(script), 0755); err != nil {
		t.Fatalf("write probe: %v", err)
	}
	return binary, counter
}

func probeRuns(t *testing.T, counter string) int {
	t.Helper()
	data, err := os.ReadFile(counter)
	if err != nil {
		return 0
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parse counter %q: %v", string(data), err)
	}
	return count
}

func shrinkPreflightTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	original := hostedPreflightTimeout
	hostedPreflightTimeout = d
	t.Cleanup(func() { hostedPreflightTimeout = original })
}

// A transient timeout must not abort the whole command. This is the first of
// the three consecutive /ant-plan failures in M4L on 27 July 2026: the probe
// timed out once, and nothing retried it.
func TestHostedPreflightRetriesAfterTimeout(t *testing.T) {
	shrinkPreflightTimeout(t, 1500*time.Millisecond)
	binary, counter := writeCountingProbe(t, map[int]bool{1: true}, 0)

	status := AvailabilityStatus{Platform: PlatformClaude, Binary: binary, Available: true}
	result := runHostedProviderPreflight(context.Background(), status, []string{"-p", "Return exactly OK."}, "")

	if !result.Available {
		t.Fatalf("preflight failed despite a healthy second attempt: %s", result.Reason)
	}
	if runs := probeRuns(t, counter); runs != 2 {
		t.Fatalf("probe ran %d times, want 2 (one timeout plus one retry)", runs)
	}
}

// A non-timeout failure is not transient. Retrying it doubles the wait and
// changes nothing, so it must surface on the first attempt.
func TestHostedPreflightDoesNotRetryHardFailure(t *testing.T) {
	shrinkPreflightTimeout(t, 5*time.Second)
	binary, counter := writeCountingProbe(t, nil, 1)

	status := AvailabilityStatus{Platform: PlatformClaude, Binary: binary, Available: true}
	result := runHostedProviderPreflight(context.Background(), status, []string{"-p", "Return exactly OK."}, "")

	if result.Available {
		t.Fatal("preflight reported available for a probe that exited non-zero")
	}
	if runs := probeRuns(t, counter); runs != 1 {
		t.Fatalf("probe ran %d times, want 1 (hard failures must not retry)", runs)
	}
}

// When every attempt times out the command still fails — tolerance must not
// become an unbounded retry loop.
func TestHostedPreflightGivesUpAfterAllAttempts(t *testing.T) {
	shrinkPreflightTimeout(t, 1500*time.Millisecond)
	binary, counter := writeCountingProbe(t, map[int]bool{1: true, 2: true}, 0)

	status := AvailabilityStatus{Platform: PlatformClaude, Binary: binary, Available: true}
	result := runHostedProviderPreflight(context.Background(), status, []string{"-p", "Return exactly OK."}, "")

	if result.Available {
		t.Fatal("preflight reported available despite every attempt timing out")
	}
	if !strings.Contains(result.Reason, "timed out") {
		t.Fatalf("reason does not mention the timeout: %s", result.Reason)
	}
	if runs := probeRuns(t, counter); runs != hostedPreflightAttempts {
		t.Fatalf("probe ran %d times, want %d", runs, hostedPreflightAttempts)
	}
}

// AETHER_PREFLIGHT_TIMEOUT widens the probe budget without a rebuild; invalid
// or non-positive values fall back to the compiled default instead of
// bricking dispatch.
func TestResolvedPreflightTimeoutEnvOverride(t *testing.T) {
	t.Setenv("AETHER_PREFLIGHT_TIMEOUT", "90s")
	if got := resolvedPreflightTimeout(); got != 90*time.Second {
		t.Fatalf("resolvedPreflightTimeout = %v, want 90s", got)
	}

	t.Setenv("AETHER_PREFLIGHT_TIMEOUT", "banana")
	if got := resolvedPreflightTimeout(); got != hostedPreflightTimeout {
		t.Fatalf("invalid env value must fall back to default, got %v", got)
	}

	t.Setenv("AETHER_PREFLIGHT_TIMEOUT", "-5s")
	if got := resolvedPreflightTimeout(); got != hostedPreflightTimeout {
		t.Fatalf("non-positive env value must fall back to default, got %v", got)
	}

	t.Setenv("AETHER_PREFLIGHT_TIMEOUT", "")
	if got := resolvedPreflightTimeout(); got != hostedPreflightTimeout {
		t.Fatalf("empty env value must use default, got %v", got)
	}
}

// writeRecordingProbe creates a stand-in provider binary that records its
// resolved working directory, argv, and stdin into files under recordDir,
// then exits with exitCode. hangOn lists 1-indexed run numbers that hang
// instead of exiting, mirroring writeCountingProbe's retry-test shape, so
// the same probe can double as a run counter when a test needs both.
func writeRecordingProbe(t *testing.T, hangOn map[int]bool, exitCode int) (binary, recordDir, counter string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell probe stand-in is POSIX-only")
	}
	dir := t.TempDir()
	binary = filepath.Join(dir, "probe.sh")
	recordDir = filepath.Join(dir, "record")
	if err := os.MkdirAll(recordDir, 0o755); err != nil {
		t.Fatalf("mkdir record dir: %v", err)
	}
	counter = filepath.Join(dir, "runs")

	var hangs []string
	for run := range hangOn {
		hangs = append(hangs, strconv.Itoa(run))
	}

	script := fmt.Sprintf(`#!/bin/sh
COUNTER=%q
COUNT=$(cat "$COUNTER" 2>/dev/null || echo 0)
COUNT=$((COUNT + 1))
printf '%%s' "$COUNT" > "$COUNTER"
pwd -P > %q/pwd.txt
printf '%%s\n' "$@" > %q/argv.txt
cat > %q/stdin.txt
for hang in %s; do
  if [ "$COUNT" = "$hang" ]; then
    sleep 30
    exit 1
  fi
done
exit %d
`, counter, recordDir, recordDir, recordDir, strings.Join(hangs, " "), exitCode)

	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatalf("write probe: %v", err)
	}
	return binary, recordDir, counter
}

func readRecordedFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read recorded file %s: %v", path, err)
	}
	return strings.TrimSpace(string(data))
}

// forcePreflightTempDirFailure swaps the makePreflightTempDir seam to force
// the degradation path, restoring the original on cleanup. Mirrors
// shrinkPreflightTimeout's "swap a package-level var, restore in
// t.Cleanup" idiom.
func forcePreflightTempDirFailure(t *testing.T) {
	t.Helper()
	original := makePreflightTempDir
	makePreflightTempDir = func() (string, error) {
		return "", errors.New("forced temp dir failure")
	}
	t.Cleanup(func() { makePreflightTempDir = original })
}

// The probe must never run in the repo working tree: a hostile or heavily
// configured repo could otherwise use the probe's cwd to load its own MCP
// servers, hooks, or plugins during a liveness check (D-07, T-163.2-07).
func TestHostedPreflightRunsOutsideRepoWorkingTree(t *testing.T) {
	binary, recordDir, _ := writeRecordingProbe(t, nil, 0)

	repoDir := t.TempDir()
	t.Chdir(repoDir)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	status := AvailabilityStatus{Platform: PlatformClaude, Binary: binary, Available: true}
	result := runHostedProviderPreflight(context.Background(), status, []string{"-p", "Return exactly OK."}, "")
	if !result.Available {
		t.Fatalf("preflight failed: %s", result.Reason)
	}

	recordedPwd := readRecordedFile(t, filepath.Join(recordDir, "pwd.txt"))
	if recordedPwd == wd {
		t.Fatalf("probe ran inside the process working directory %q", wd)
	}
	base := filepath.Base(recordedPwd)
	if !strings.HasPrefix(base, "aether-preflight-") {
		t.Fatalf("probe cwd %q does not carry the aether-preflight- prefix", recordedPwd)
	}
	if _, statErr := os.Stat(recordedPwd); !os.IsNotExist(statErr) {
		t.Fatalf("probe temp dir %q still exists after the call (stat err=%v)", recordedPwd, statErr)
	}
}

// The Codex probe must inherit the shared AETHER_PREFLIGHT_TIMEOUT and
// hostedPreflightAttempts retry policy like Claude and OpenCode. This fails
// if RealInvoker.Preflight ever regains its own hardcoded timeout or stops
// retrying.
func TestCodexPreflightHonorsSharedTimeoutAndRetry(t *testing.T) {
	binary, counter := writeCountingProbe(t, map[int]bool{1: true}, 0)
	t.Setenv("AETHER_CODEX_PATH", binary)
	t.Setenv("AETHER_PREFLIGHT_TIMEOUT", "1500ms")

	result := NewRealInvoker().Preflight(context.Background(), t.TempDir())

	if !result.Available {
		t.Fatalf("codex preflight failed despite a healthy second attempt: %s", result.Reason)
	}
	if runs := probeRuns(t, counter); runs != hostedPreflightAttempts {
		t.Fatalf("probe ran %d times, want %d (one timeout plus one retry)", runs, hostedPreflightAttempts)
	}
}

// The Codex probe must run outside the repo root passed to Preflight, same
// as Claude and OpenCode (D-07).
func TestCodexPreflightRunsOutsideRepoWorkingTree(t *testing.T) {
	binary, recordDir, _ := writeRecordingProbe(t, nil, 0)
	t.Setenv("AETHER_CODEX_PATH", binary)

	root := t.TempDir()
	evaledRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve root symlinks: %v", err)
	}

	result := NewRealInvoker().Preflight(context.Background(), root)
	if !result.Available {
		t.Fatalf("codex preflight failed: %s", result.Reason)
	}

	recordedPwd := readRecordedFile(t, filepath.Join(recordDir, "pwd.txt"))
	if recordedPwd == evaledRoot {
		t.Fatalf("codex probe ran inside the root argument %q passed to Preflight", root)
	}
}

// D-04 payload guard: the probe must stay a real model round-trip. This
// fails if the probe prompt is ever swapped for a cheaper non-model check
// or gains a model-selection flag, on any of the three providers.
func TestPreflightProbeRemainsARealModelRoundTrip(t *testing.T) {
	claudeBinary, claudeRecordDir, _ := writeRecordingProbe(t, nil, 0)
	claudeStatus := (&ClaudeDispatcher{binaryName: claudeBinary}).Preflight(context.Background(), t.TempDir())
	if !claudeStatus.Available {
		t.Fatalf("claude preflight failed: %s", claudeStatus.Reason)
	}
	claudeArgv := readRecordedFile(t, filepath.Join(claudeRecordDir, "argv.txt"))
	if !strings.Contains(claudeArgv, "Return exactly OK.") {
		t.Fatalf("claude preflight argv missing the real model round-trip prompt: %q", claudeArgv)
	}

	openCodeBinary, openCodeRecordDir, _ := writeRecordingProbe(t, nil, 0)
	openCodeStatus := (&OpenCodeDispatcher{binaryName: openCodeBinary}).Preflight(context.Background(), t.TempDir())
	if !openCodeStatus.Available {
		t.Fatalf("opencode preflight failed: %s", openCodeStatus.Reason)
	}
	openCodeArgv := readRecordedFile(t, filepath.Join(openCodeRecordDir, "argv.txt"))
	if !strings.Contains(openCodeArgv, "Return exactly OK.") {
		t.Fatalf("opencode preflight argv missing the real model round-trip prompt: %q", openCodeArgv)
	}

	codexBinary, codexRecordDir, _ := writeRecordingProbe(t, nil, 0)
	t.Setenv("AETHER_CODEX_PATH", codexBinary)
	codexStatus := NewRealInvoker().Preflight(context.Background(), t.TempDir())
	if !codexStatus.Available {
		t.Fatalf("codex preflight failed: %s", codexStatus.Reason)
	}
	codexStdin := readRecordedFile(t, filepath.Join(codexRecordDir, "stdin.txt"))
	if !strings.Contains(codexStdin, "Return exactly OK.") {
		t.Fatalf("codex preflight stdin missing the real model round-trip prompt: %q", codexStdin)
	}
}

// The degradation guard for Task 2 (T-163.2-29): a makePreflightTempDir
// failure must never brick dispatch. This fails if anyone converts the
// temp-dir error into an early return, an unavailable status, or a
// dispatch failure.
func TestHostedPreflightSurvivesTempDirFailure(t *testing.T) {
	forcePreflightTempDirFailure(t)
	binary, recordDir, counter := writeRecordingProbe(t, nil, 0)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	status := AvailabilityStatus{Platform: PlatformClaude, Binary: binary, Available: true}
	result := runHostedProviderPreflight(context.Background(), status, []string{"-p", "Return exactly OK."}, "")

	if !result.Available {
		t.Fatalf("preflight failed after a forced temp-dir failure: %s", result.Reason)
	}
	if runs := probeRuns(t, counter); runs != 1 {
		t.Fatalf("probe ran %d times, want 1 (temp-dir failure must not short-circuit execution)", runs)
	}

	recordedPwd := readRecordedFile(t, filepath.Join(recordDir, "pwd.txt"))
	if recordedPwd != wd {
		evaledWd, evalErr := filepath.EvalSymlinks(wd)
		if evalErr != nil || recordedPwd != evaledWd {
			t.Fatalf("probe cwd = %q, want process working directory %q (cmd.Dir must stay unset on temp-dir failure)", recordedPwd, wd)
		}
	}
}
