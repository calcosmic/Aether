package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// gatePreflightFake is a WorkerInvoker + WorkerProviderPreflighter fake that
// counts how many times Preflight was called and reports a configurable,
// real platform so cache entries are keyable. Extends the same fake shape as
// preflightBlockingInvoker in cmd/dispatch_platform_helpers_test.go.
type gatePreflightFake struct {
	calls     int
	available bool
	platform  codex.Platform
}

func (f *gatePreflightFake) Invoke(context.Context, codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{}, nil
}

func (f *gatePreflightFake) IsAvailable(context.Context) bool { return true }

func (f *gatePreflightFake) ValidateAgent(string) error { return nil }

func (f *gatePreflightFake) Platform() codex.Platform { return f.platform }

func (f *gatePreflightFake) Preflight(ctx context.Context, root string) codex.AvailabilityStatus {
	f.calls++
	if f.available {
		return codex.AvailabilityStatus{
			Platform:  f.platform,
			Available: true,
			Category:  codex.AvailabilityCategoryAvailable,
		}
	}
	return codex.AvailabilityStatus{
		Platform:  f.platform,
		Available: false,
		Category:  codex.AvailabilityCategoryAuthProbeFailed,
		Reason:    "fake probe failure",
	}
}

// withCapturedGateStderr points the package-level stderr writer at a fresh
// bytes.Buffer for the duration of the test, restoring the previous writer
// on cleanup.
func withCapturedGateStderr(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	original := stderr
	stderr = &buf
	t.Cleanup(func() { stderr = original })
	return &buf
}

func gatePreflightDispatches(root string) []codex.WorkerDispatch {
	return []codex.WorkerDispatch{{
		ID:         "d1",
		WorkerName: "Scout-1",
		Caste:      "scout",
		TaskID:     "gate-test",
		TaskBrief:  "Exercise the shared preflight gate",
		Root:       root,
		Wave:       1,
	}}
}

func TestGatedPreflightProbesOnColdCache(t *testing.T) {
	withTestPreflightStore(t)

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	status, outcome := gatedProviderPreflight(context.Background(), fake, codex.PlatformClaude, t.TempDir(), time.Now())

	if fake.calls != 1 {
		t.Fatalf("Preflight called %d times on a cold cache, want 1", fake.calls)
	}
	if outcome.Source != preflightSourceProbe {
		t.Fatalf("outcome.Source = %q, want %q", outcome.Source, preflightSourceProbe)
	}
	if !status.Available {
		t.Fatal("status.Available = false, want true")
	}
}

func TestGatedPreflightSkipsProbeOnWarmCache(t *testing.T) {
	withTestPreflightStore(t)
	buf := withCapturedGateStderr(t)

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	dispatches := gatePreflightDispatches(t.TempDir())

	if err := preflightWorkerProvider(context.Background(), fake, dispatches); err != nil {
		t.Fatalf("first preflightWorkerProvider = %v, want nil", err)
	}
	buf.Reset()
	if err := preflightWorkerProvider(context.Background(), fake, dispatches); err != nil {
		t.Fatalf("second preflightWorkerProvider = %v, want nil", err)
	}

	if fake.calls != 1 {
		t.Fatalf("Preflight called %d times across two dispatches, want 1 -- this is the test that fails if the per-dispatch cost ever comes back", fake.calls)
	}
	if !strings.Contains(buf.String(), "cached OK") {
		t.Fatalf("second-call stderr = %q, want it to contain %q", buf.String(), "cached OK")
	}
}

func TestGatedPreflightDoesNotReuseAnotherPlatformsWindow(t *testing.T) {
	withTestPreflightStore(t)
	now := time.Now()

	claudeFake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	if _, outcome := gatedProviderPreflight(context.Background(), claudeFake, codex.PlatformClaude, t.TempDir(), now); outcome.Source != preflightSourceProbe {
		t.Fatalf("claude warm-up outcome.Source = %q, want probe", outcome.Source)
	}
	if claudeFake.calls != 1 {
		t.Fatalf("claude probe count = %d, want 1", claudeFake.calls)
	}

	codexFake := &gatePreflightFake{platform: codex.PlatformCodex, available: true}
	_, outcome := gatedProviderPreflight(context.Background(), codexFake, codex.PlatformCodex, t.TempDir(), now)
	if codexFake.calls != 1 {
		t.Fatalf("codex probe count = %d, want 1 -- a warm claude window must never cover codex", codexFake.calls)
	}
	if outcome.Source != preflightSourceProbe {
		t.Fatalf("codex outcome.Source = %q, want probe", outcome.Source)
	}

	totalProbes := claudeFake.calls + codexFake.calls
	if totalProbes != 2 {
		t.Fatalf("total probe count = %d, want 2", totalProbes)
	}
}

func TestGatedPreflightNeverCachesFailure(t *testing.T) {
	withTestPreflightStore(t)
	now := time.Now()

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: false}
	_, outcome1 := gatedProviderPreflight(context.Background(), fake, codex.PlatformClaude, t.TempDir(), now)
	_, outcome2 := gatedProviderPreflight(context.Background(), fake, codex.PlatformClaude, t.TempDir(), now)

	if fake.calls != 2 {
		t.Fatalf("Preflight called %d times across two failing calls, want 2 -- a lapsed login must keep failing", fake.calls)
	}
	if outcome1.Source == preflightSourceCache || outcome2.Source == preflightSourceCache {
		t.Fatalf("a failed probe must never produce a cache outcome: outcome1=%q outcome2=%q", outcome1.Source, outcome2.Source)
	}
}

func TestGatedPreflightSkipSwitchBypassesProbeAndCache(t *testing.T) {
	withTestPreflightStore(t)
	buf := withCapturedGateStderr(t)

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	dispatches := gatePreflightDispatches(t.TempDir())

	// Warm the cache first so the skip check is genuinely exercised against a
	// live trust window: skip must win even when a cache hit is available,
	// proving the skip branch runs before the cache read (decision order 1
	// before 3), not merely when the cache happens to be cold.
	if err := preflightWorkerProvider(context.Background(), fake, dispatches); err != nil {
		t.Fatalf("cache warm-up preflightWorkerProvider = %v, want nil", err)
	}
	if fake.calls != 1 {
		t.Fatalf("Preflight called %d times during warm-up, want 1", fake.calls)
	}
	entryBefore, _, hitBefore := loadFreshPreflightCache(codex.PlatformClaude, time.Now())
	if !hitBefore {
		t.Fatal("expected a warm cache entry after the warm-up call")
	}

	buf.Reset()
	t.Setenv(envSkipPreflight, "1")

	if err := preflightWorkerProvider(context.Background(), fake, dispatches); err != nil {
		t.Fatalf("preflightWorkerProvider with skip switch set = %v, want nil", err)
	}
	if fake.calls != 1 {
		t.Fatalf("Preflight called %d times total with a warm cache and skip switch set, want 1 (skip must win over an available cache hit)", fake.calls)
	}
	out := buf.String()
	if !strings.Contains(out, "AETHER_SKIP_PREFLIGHT") {
		t.Fatalf("skip stderr = %q, want it to contain %q", out, "AETHER_SKIP_PREFLIGHT")
	}
	if !strings.Contains(out, "NOT verified") {
		t.Fatalf("skip stderr = %q, want it to contain %q", out, "NOT verified")
	}
	entryAfter, _, hitAfter := loadFreshPreflightCache(codex.PlatformClaude, time.Now())
	if !hitAfter || entryAfter.CheckedAt != entryBefore.CheckedAt {
		t.Fatalf("the skip switch must not write or alter the cache entry: before=%+v after=%+v (hit=%v)", entryBefore, entryAfter, hitAfter)
	}

	// A subsequent call with the switch removed and a cold cache must probe.
	if err := clearPreflightCache(codex.PlatformClaude); err != nil {
		t.Fatalf("clearPreflightCache = %v, want nil", err)
	}
	buf.Reset()
	t.Setenv(envSkipPreflight, "0")
	if err := preflightWorkerProvider(context.Background(), fake, dispatches); err != nil {
		t.Fatalf("preflightWorkerProvider after removing skip switch = %v, want nil", err)
	}
	if fake.calls != 2 {
		t.Fatalf("Preflight called %d times after removing skip switch and clearing cache, want 2", fake.calls)
	}
}

func TestGatedPreflightExpiredWindowReprobes(t *testing.T) {
	withTestPreflightStore(t)
	now := time.Now()

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	if _, outcome := gatedProviderPreflight(context.Background(), fake, codex.PlatformClaude, t.TempDir(), now); outcome.Source != preflightSourceProbe {
		t.Fatalf("warm-up outcome.Source = %q, want probe", outcome.Source)
	}

	later := now.Add(2 * time.Hour) // past the 1h default TTL
	_, outcome := gatedProviderPreflight(context.Background(), fake, codex.PlatformClaude, t.TempDir(), later)
	if fake.calls != 2 {
		t.Fatalf("Preflight called %d times, want 2 -- an expired trust window must re-probe", fake.calls)
	}
	if outcome.Source != preflightSourceProbe {
		t.Fatalf("outcome.Source after expiry = %q, want probe", outcome.Source)
	}
}

func TestAdapterPreflightResponseCarriesOutcome(t *testing.T) {
	withTestPreflightStore(t)
	t.Chdir(t.TempDir())

	t.Setenv(envSkipPreflight, "1")
	skipResp, err := runInternalWorkerAdapter(context.Background(), "", true, true)
	if err != nil {
		t.Fatalf("runInternalWorkerAdapter with skip switch = %v, want nil", err)
	}
	skipJSON, err := json.Marshal(skipResp)
	if err != nil {
		t.Fatalf("marshal skip response: %v", err)
	}
	var skipDecoded map[string]json.RawMessage
	if err := json.Unmarshal(skipJSON, &skipDecoded); err != nil {
		t.Fatalf("unmarshal skip response: %v", err)
	}
	preflightRaw, ok := skipDecoded["preflight"]
	if !ok {
		t.Fatalf("skip-path JSON missing the preflight key: %s", skipJSON)
	}
	var preflightDecoded map[string]interface{}
	if err := json.Unmarshal(preflightRaw, &preflightDecoded); err != nil {
		t.Fatalf("unmarshal preflight field: %v", err)
	}
	if preflightDecoded["source"] != "skipped" {
		t.Fatalf("preflight.source = %v, want %q", preflightDecoded["source"], "skipped")
	}
	if notice, _ := preflightDecoded["notice"].(string); notice == "" {
		t.Fatalf("preflight.notice missing or empty in skip response: %s", skipJSON)
	}

	// The ordinary probe path must leave the JSON byte-identical to today:
	// no preflight key at all.
	t.Setenv(envSkipPreflight, "0")
	probeResp, err := runInternalWorkerAdapter(context.Background(), "", true, true)
	if err != nil {
		t.Fatalf("runInternalWorkerAdapter ordinary probe path = %v, want nil", err)
	}
	probeJSON, err := json.Marshal(probeResp)
	if err != nil {
		t.Fatalf("marshal probe response: %v", err)
	}
	var probeDecoded map[string]json.RawMessage
	if err := json.Unmarshal(probeJSON, &probeDecoded); err != nil {
		t.Fatalf("unmarshal probe response: %v", err)
	}
	if _, ok := probeDecoded["preflight"]; ok {
		t.Fatalf("ordinary probe-path JSON must omit the preflight key entirely: %s", probeJSON)
	}
}

func TestDispatchChokepointsShareOneCache(t *testing.T) {
	withTestPreflightStore(t)

	// Warm the cache through the direct-Go dispatch chokepoint.
	fakeA := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	dispatches := gatePreflightDispatches(t.TempDir())
	if err := preflightWorkerProvider(context.Background(), fakeA, dispatches); err != nil {
		t.Fatalf("preflightWorkerProvider warm-up = %v, want nil", err)
	}
	if fakeA.calls != 1 {
		t.Fatalf("direct-Go chokepoint probed %d times, want 1", fakeA.calls)
	}

	// The adapter-side gate (the exact function the --preflight branch of
	// cmd/internal_worker_adapter.go calls) must see the warm window for the
	// same platform without a second probe.
	fakeB := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	_, outcome := gatedProviderPreflight(context.Background(), fakeB, codex.PlatformClaude, t.TempDir(), time.Now())
	if outcome.Source != preflightSourceCache {
		t.Fatalf("adapter-side gate outcome.Source = %q, want cache -- both chokepoints must share one trust window", outcome.Source)
	}
	if fakeB.calls != 0 {
		t.Fatalf("adapter-side gate probed %d times on a warm cache, want 0", fakeB.calls)
	}

	// And vice versa: warm through the adapter-side gate, then confirm the
	// direct-Go chokepoint reuses it too.
	if err := clearPreflightCache(codex.PlatformClaude); err != nil {
		t.Fatalf("clearPreflightCache = %v, want nil", err)
	}
	fakeC := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	if _, outcome := gatedProviderPreflight(context.Background(), fakeC, codex.PlatformClaude, t.TempDir(), time.Now()); outcome.Source != preflightSourceProbe {
		t.Fatalf("cold-cache adapter-side warm-up outcome.Source = %q, want probe", outcome.Source)
	}
	if fakeC.calls != 1 {
		t.Fatalf("adapter-side gate warm-up probed %d times, want 1", fakeC.calls)
	}

	fakeD := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	if err := preflightWorkerProvider(context.Background(), fakeD, dispatches); err != nil {
		t.Fatalf("preflightWorkerProvider after adapter warm-up = %v, want nil", err)
	}
	if fakeD.calls != 0 {
		t.Fatalf("direct-Go chokepoint probed %d times on a warm cache, want 0 -- both chokepoints must share one trust window", fakeD.calls)
	}
}
