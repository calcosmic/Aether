package cmd

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// TestRecruitmentProbeReadsTimeoutEnv proves resolvedRecruitmentProbeTimeout
// follows the resolve-warn-fallback shape resolvePreflightTimeoutMs uses on
// the TypeScript side: a valid Go duration wins, an invalid one falls back
// to the compiled default.
func TestRecruitmentProbeReadsTimeoutEnv(t *testing.T) {
	t.Setenv(recruitmentProbeTimeoutEnv, "2500ms")
	if got := resolvedRecruitmentProbeTimeout(); got != 2500*time.Millisecond {
		t.Fatalf("resolvedRecruitmentProbeTimeout() = %v, want 2.5s", got)
	}

	t.Setenv(recruitmentProbeTimeoutEnv, "not-a-duration")
	if got := resolvedRecruitmentProbeTimeout(); got != recruitmentProbeDefaultTimeout {
		t.Fatalf("resolvedRecruitmentProbeTimeout() with invalid value = %v, want default %v", got, recruitmentProbeDefaultTimeout)
	}

	t.Setenv(recruitmentProbeTimeoutEnv, "")
	if got := resolvedRecruitmentProbeTimeout(); got != recruitmentProbeDefaultTimeout {
		t.Fatalf("resolvedRecruitmentProbeTimeout() with empty value = %v, want default %v", got, recruitmentProbeDefaultTimeout)
	}
}

// TestRecruitmentProbeBoundsWallClock proves a probe whose fake launcher
// never returns (the counterpart to the pre-configureWorkerCommand 60s
// overrun documented on runAvailabilityProbeOnce, pkg/codex/platform_dispatch.go)
// still resolves within a small multiple of a 200ms budget -- the same
// measurement TestRecruitmentDispatchTerminatesWholeProcessGroup already
// uses for dispatchRecruitment's own process-group teardown.
func TestRecruitmentProbeBoundsWallClock(t *testing.T) {
	resetRecruitmentProbeCacheForTest()
	defer resetRecruitmentProbeCacheForTest()

	t.Setenv(recruitmentProbeTimeoutEnv, "200ms")

	previous := recruitmentProbeRunner
	defer func() { recruitmentProbeRunner = previous }()
	recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}

	start := time.Now()
	result := probeNativeNestingOnce()
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Fatalf("probeNativeNestingOnce took %v against a 200ms budget, want well under 2s", elapsed)
	}
	if result.Supported {
		t.Fatalf("expected an unsupported verdict on timeout, got Supported=true (%+v)", result)
	}
	if result.Verdict == "" {
		t.Fatal("expected a non-empty verdict on timeout")
	}
}

// TestRecruitmentProbeRunsOncePerProcess proves two calls to
// probeNativeNestingOnce in one process produce exactly one launch --
// the second reads the cached result, per this plan's own must_have.
func TestRecruitmentProbeRunsOncePerProcess(t *testing.T) {
	resetRecruitmentProbeCacheForTest()
	defer resetRecruitmentProbeCacheForTest()

	var launches int64
	previous := recruitmentProbeRunner
	defer func() { recruitmentProbeRunner = previous }()
	recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
		atomic.AddInt64(&launches, 1)
		return "NATIVE_NESTING_AVAILABLE=false", nil
	}

	first := probeNativeNestingOnce()
	second := probeNativeNestingOnce()

	if got := atomic.LoadInt64(&launches); got != 1 {
		t.Fatalf("expected exactly 1 launch across two calls, got %d", got)
	}
	if first.ProbedAt != second.ProbedAt {
		t.Fatalf("expected the cached result to be returned byte-identical, got first=%+v second=%+v", first, second)
	}
}

// TestRecruitmentProbeDirectoryIsCleanedUp proves the probe's working
// directory is a fresh temp directory outside the repository, and that it is
// removed once the probe returns, on both the success and the timeout path.
func TestRecruitmentProbeDirectoryIsCleanedUp(t *testing.T) {
	t.Run("success path", func(t *testing.T) {
		resetRecruitmentProbeCacheForTest()
		defer resetRecruitmentProbeCacheForTest()

		var recordedDir string
		previous := recruitmentProbeRunner
		defer func() { recruitmentProbeRunner = previous }()
		recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
			recordedDir = dir
			return "NATIVE_NESTING_AVAILABLE=true", nil
		}

		result := probeNativeNestingOnce()
		if recordedDir == "" {
			t.Fatal("fixture is broken: recruitmentProbeRunner was never called")
		}
		if !result.Supported {
			t.Fatalf("expected a supported verdict, got %+v", result)
		}
		if _, err := os.Stat(recordedDir); !os.IsNotExist(err) {
			t.Fatalf("expected probe directory %q to be removed after a successful probe, stat err = %v", recordedDir, err)
		}
	})

	t.Run("timeout path", func(t *testing.T) {
		resetRecruitmentProbeCacheForTest()
		defer resetRecruitmentProbeCacheForTest()

		t.Setenv(recruitmentProbeTimeoutEnv, "50ms")

		var recordedDir string
		previous := recruitmentProbeRunner
		defer func() { recruitmentProbeRunner = previous }()
		recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
			recordedDir = dir
			<-ctx.Done()
			return "", ctx.Err()
		}

		probeNativeNestingOnce()
		if recordedDir == "" {
			t.Fatal("fixture is broken: recruitmentProbeRunner was never called")
		}
		if _, err := os.Stat(recordedDir); !os.IsNotExist(err) {
			t.Fatalf("expected probe directory %q to be removed after a timed-out probe, stat err = %v", recordedDir, err)
		}
	})
}

// TestRecruitmentProbeNeverInfersFromPlatformName proves a probe verdict
// comes only from the launcher's reported output, never from a hardcoded
// platform/version table -- the must_have this plan names explicitly.
// Forcing two different reported outputs on the SAME resolved binary/platform
// must produce two different verdicts; a hardcoded table keyed on platform
// name alone could not do this.
func TestRecruitmentProbeNeverInfersFromPlatformName(t *testing.T) {
	previous := recruitmentProbeRunner
	defer func() { recruitmentProbeRunner = previous }()

	resetRecruitmentProbeCacheForTest()
	recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
		return "NATIVE_NESTING_AVAILABLE=true", nil
	}
	supported := probeNativeNestingOnce()
	if !supported.Supported {
		t.Fatalf("expected Supported=true when the launcher reports availability, got %+v", supported)
	}

	resetRecruitmentProbeCacheForTest()
	recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
		return "NATIVE_NESTING_AVAILABLE=false", nil
	}
	unsupported := probeNativeNestingOnce()
	if unsupported.Supported {
		t.Fatalf("expected Supported=false when the launcher reports unavailability, got %+v", unsupported)
	}
}
