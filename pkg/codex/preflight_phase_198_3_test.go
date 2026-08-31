package codex

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAvailabilityPreflightUsesSharedTimeoutSetting(t *testing.T) {
	if hostedPreflightTimeout != 45*time.Second {
		t.Fatalf("hostedPreflightTimeout = %v, want the 45s readiness default", hostedPreflightTimeout)
	}

	t.Setenv("AETHER_PREFLIGHT_TIMEOUT", "2300ms")
	t.Setenv("AETHER_PROBE_TIMEOUT", "17ms")
	if got := resolvedPreflightTimeout(); got != 2300*time.Millisecond {
		t.Fatalf("model preflight timeout = %v, want 2.3s", got)
	}
	if got := resolvedAvailabilityProbeTimeout(); got != 2300*time.Millisecond {
		t.Fatalf("availability timeout = %v, want the same 2.3s setting as model preflight", got)
	}

	t.Setenv("AETHER_PREFLIGHT_TIMEOUT", "not-a-duration")
	if got := resolvedAvailabilityProbeTimeout(); got != 45*time.Second {
		t.Fatalf("invalid shared timeout resolved to %v, want the 45s default", got)
	}
}

func TestAvailabilityPreflightAttemptsAreBoundedByFailureClass(t *testing.T) {
	if availabilityProbeAttempts != 2 {
		t.Fatalf("availabilityProbeAttempts = %d, want exactly 2", availabilityProbeAttempts)
	}

	for _, tc := range []struct {
		name     string
		exitCode int
	}{
		{name: "auth rejected", exitCode: 1},
		{name: "login required", exitCode: 2},
		{name: "binary failure", exitCode: 126},
		{name: "other non-timeout failure", exitCode: 17},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binary, counter := writeCountingProbe(t, nil, tc.exitCode)
			if _, err := runAvailabilityProbe(context.Background(), binary); err == nil {
				t.Fatal("non-timeout probe unexpectedly succeeded")
			}
			if got := probeRuns(t, counter); got != 1 {
				t.Fatalf("attempts = %d, want 1 for a non-timeout readiness failure", got)
			}
		})
	}

	t.Run("timeout gets one retry", func(t *testing.T) {
		binary, counter := writeCountingProbe(t, map[int]bool{1: true}, 0)
		t.Setenv("AETHER_PREFLIGHT_TIMEOUT", "500ms")
		t.Setenv("AETHER_PROBE_TIMEOUT", "500ms") // keeps the RED run bounded before the shared setting lands

		if _, err := runAvailabilityProbe(context.Background(), binary); err != nil {
			t.Fatalf("second readiness attempt should succeed: %v", err)
		}
		if got := probeRuns(t, counter); got != 2 {
			t.Fatalf("attempts = %d, want exactly 2 after one timeout", got)
		}
	})
}

func TestLiveReadinessSourceHasOneTimeoutEnvironmentVariable(t *testing.T) {
	source, err := os.ReadFile("platform_dispatch.go")
	if err != nil {
		t.Fatalf("read platform_dispatch.go: %v", err)
	}
	text := string(source)
	if strings.Contains(text, "AETHER_PROBE_TIMEOUT") {
		t.Fatal("platform_dispatch.go still contains AETHER_PROBE_TIMEOUT; every live readiness path must use AETHER_PREFLIGHT_TIMEOUT")
	}
	if !strings.Contains(text, "AETHER_PREFLIGHT_TIMEOUT") {
		t.Fatal("platform_dispatch.go does not expose AETHER_PREFLIGHT_TIMEOUT")
	}
}
