package codex

import (
	"context"
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
