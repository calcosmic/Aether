package codex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeProbeScript creates an executable stand-in for a provider CLI whose
// behaviour the test controls.
func writeProbeScript(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-probe")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatalf("write probe script: %v", err)
	}
	return path
}

// TestAvailabilityProbeRetriesOnlyTimeouts is the regression lock for workers
// that could not start on a machine where the provider CLI was installed and
// logged in the whole time.
//
// On 2026-08-21 a Formica build lost two of four workers to
// `claude auth status failed: timed out` — while the other two started fine on
// the same credentials, which is what proves the auth was healthy and the
// probe was not. The same day, this repo's own suite lost a test to
// `codex login status failed: timed out` and passed on a rerun.
//
// The auth probe had a 3-second budget, no retry and no environment override.
// Idle, these commands answer in 0.05–0.3s, so the budget looked generous;
// what it could not absorb was an occasional stall, and a stall was fatal.
//
// The sibling model round-trip probe already learned this and wrote it down
// (hostedPreflightTimeout: 20s → 45s plus one retry). This locks the same
// treatment here, and locks the limit too: only a timeout retries. A genuine
// auth failure must surface on the first attempt rather than costing a second
// wait to reach the same answer.
func TestAvailabilityProbeRetriesOnlyTimeouts(t *testing.T) {
	t.Run("a transient stall is retried and succeeds", func(t *testing.T) {
		// Sleeps past the budget on the first run only, using a marker file
		// to tell the runs apart.
		marker := filepath.Join(t.TempDir(), "seen")
		script := writeProbeScript(t, "if [ ! -f "+marker+" ]; then\n  touch "+marker+"\n  sleep 30\nfi\necho '"+`{"loggedIn":true}`+"'")

		// 2s, not 200ms: the budget has to clear /bin/sh startup on a machine
		// running the whole suite, or the script is killed before it reaches
		// `touch` and BOTH attempts time out -- a fixture that fails for a
		// reason the code under test has nothing to do with. The slow branch
		// sleeps 30s, so the first attempt still times out reliably.
		t.Setenv("AETHER_PROBE_TIMEOUT", "2s")

		output, err := runAvailabilityProbe(context.Background(), script)
		if err != nil {
			t.Fatalf("a probe that stalled once and then answered was reported as a failure: %v\n\nA worker cannot start on this, even though the CLI is installed and logged in.", err)
		}
		if !strings.Contains(output, "loggedIn") {
			t.Fatalf("retry returned no usable output: %q", output)
		}
		if _, statErr := os.Stat(marker); statErr != nil {
			t.Fatal("fixture never took the slow path, so nothing was retried and this test proves nothing")
		}
	})

	t.Run("a real auth failure is not retried", func(t *testing.T) {
		// Counts its runs. A non-timeout failure must be reported after one.
		counter := filepath.Join(t.TempDir(), "runs")
		script := writeProbeScript(t, `echo x >> `+counter+`
echo 'not logged in' >&2
exit 1`)

		t.Setenv("AETHER_PROBE_TIMEOUT", "5s")

		if _, err := runAvailabilityProbe(context.Background(), script); err == nil {
			t.Fatal("expected a failing probe to report failure")
		}
		data, readErr := os.ReadFile(counter)
		if readErr != nil {
			t.Fatalf("read run counter: %v", readErr)
		}
		if runs := strings.Count(string(data), "x"); runs != 1 {
			t.Fatalf("a non-timeout failure ran %d times, want 1 — retrying it doubles the user's wait to reach the same answer", runs)
		}
	})

	t.Run("a persistent stall still fails, as a timeout", func(t *testing.T) {
		script := writeProbeScript(t, "sleep 30")
		t.Setenv("AETHER_PROBE_TIMEOUT", "1s")

		_, err := runAvailabilityProbe(context.Background(), script)
		if err == nil {
			t.Fatal("a probe that never answers must still fail")
		}
		if !errors.Is(err, errAvailabilityProbeTimeout) {
			t.Fatalf("a persistent stall reported %v, want a timeout — the retry must not disguise what went wrong", err)
		}
	})
}

// TestAvailabilityProbeTimeoutIsOverridable locks the escape hatch a slow host
// needs. The model round-trip probe has had AETHER_PREFLIGHT_TIMEOUT for
// months; the auth probe had no override at all, so a host where it ran slow
// had no fix short of a rebuild.
func TestAvailabilityProbeTimeoutIsOverridable(t *testing.T) {
	t.Run("honours a valid duration", func(t *testing.T) {
		t.Setenv("AETHER_PROBE_TIMEOUT", "90s")
		if got := resolvedAvailabilityProbeTimeout(); got != 90*time.Second {
			t.Fatalf("resolvedAvailabilityProbeTimeout() = %v, want 90s", got)
		}
	})

	for _, bad := range []string{"", "not-a-duration", "0s", "-5s"} {
		t.Run("falls back on "+strings.ReplaceAll(bad, "-", "minus"), func(t *testing.T) {
			t.Setenv("AETHER_PROBE_TIMEOUT", bad)
			if got := resolvedAvailabilityProbeTimeout(); got != defaultProbeTimout {
				t.Fatalf("a mistyped override (%q) produced %v instead of the compiled default %v — a typo must never brick dispatch", bad, got, defaultProbeTimout)
			}
		})
	}
}

// TestAvailabilityProbeBudgetCoversObservedLatency guards the number itself.
//
// Measured on 2026-08-21: `claude auth status --json` answers in ~0.3s idle
// and ~0.7s with eight running concurrently under build load; `codex login
// status` in ~0.06s. The old 3s budget was roughly 4x the loaded figure and
// still timed out in the field, so the budget must keep a margin far wider
// than "looks like enough".
func TestAvailabilityProbeBudgetCoversObservedLatency(t *testing.T) {
	const observedLoadedLatency = 700 * time.Millisecond
	if defaultProbeTimout < 10*observedLoadedLatency {
		t.Fatalf("auth probe budget is %v, under 10x the measured loaded latency (%v). 3s was ~4x and still lost workers in the field; a stall this cannot absorb stops a worker starting on a machine that is correctly logged in.", defaultProbeTimout, observedLoadedLatency)
	}
}

// TestAvailabilityProbeBudgetBoundsWallClock locks what makes the budget real
// rather than advisory.
//
// The provider CLIs are wrappers that spawn children. Cancelling the context
// kills only the direct child; cmd.Run() then blocks until every inherited
// pipe closes, which means until the grandchild finishes on its own. A probe
// could therefore burn far longer than its budget and still report "timed
// out" — the wait it was supposed to bound happened anyway.
//
// configureWorkerCommand already solves this for workers and for the model
// round-trip preflight (process group, group-wide cancel, WaitDelay backstop).
// The auth probe was the one caller building a bare exec.CommandContext and
// inheriting none of it.
//
// The fixture deliberately does NOT use `exec`, so the shell stays alive as a
// parent and leaves `sleep` holding the pipes — the shape a Node-wrapper CLI
// produces. Without the fix this takes the child's full sleep; with it, the
// budget holds.
func TestAvailabilityProbeBudgetBoundsWallClock(t *testing.T) {
	script := writeProbeScript(t, "sleep 60 &\nsleep 60")
	t.Setenv("AETHER_PROBE_TIMEOUT", "1s")

	start := time.Now()
	_, err := runAvailabilityProbe(context.Background(), script)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("a probe that never answers must fail")
	}
	// Two attempts at 1s, plus process teardown. Generous enough not to be
	// flaky, far below the 60s the orphaned child would otherwise impose.
	if limit := 20 * time.Second; elapsed > limit {
		t.Fatalf("probe took %v for a 1s budget over %d attempts (limit %v).\n\nThe deadline killed the direct child but the wait continued until an orphaned grandchild released the output pipes, so the budget bounded nothing. This is what made a slow provider CLI stop a worker starting.", elapsed.Round(time.Millisecond), availabilityProbeAttempts, limit)
	}
}
