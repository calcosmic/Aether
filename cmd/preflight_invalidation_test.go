package cmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// TestProviderAuthFailureClassification asserts providerAuthFailure's
// decision for every category and message shape the D-03 invalidation path
// must recognize. Do not weaken the false-cases: they are what stops a flaky
// test failure from silently reintroducing a paid probe on every dispatch.
func TestProviderAuthFailureClassification(t *testing.T) {
	preflightErrorWithCategory := func(category codex.AvailabilityCategory) error {
		return &workerProviderPreflightError{status: codex.AvailabilityStatus{
			Platform: codex.PlatformClaude,
			Category: category,
			Reason:   "fake preflight failure",
		}}
	}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"category: auth_probe_failed", preflightErrorWithCategory(codex.AvailabilityCategoryAuthProbeFailed), true},
		{"category: auth_inactive", preflightErrorWithCategory(codex.AvailabilityCategoryAuthInactive), true},
		{"category: invalid_auth_output", preflightErrorWithCategory(codex.AvailabilityCategoryInvalidAuthOutput), true},
		{"category: credentials_missing", preflightErrorWithCategory(codex.AvailabilityCategoryCredentialsMissing), true},
		{"category: provider_config_invalid", preflightErrorWithCategory(codex.AvailabilityCategoryProviderConfig), true},
		{"message: authentication failed", errors.New("authentication failed"), true},
		{"message: invalid credentials", errors.New("invalid credentials"), true},
		{"message: please run login again", errors.New("please run login again"), true},
		{"message: permission denied", errors.New("permission denied"), true},
		{"message: missing API key", errors.New("missing API key"), true},
		{"message: authorization error", errors.New("authorization error"), true},
		{"ordinary: worker timed out", errors.New("worker timed out after 600s"), false},
		{"ordinary: task blocked", errors.New("task blocked: missing input file"), false},
		{"ordinary: exit status", errors.New("exit status 1"), false},
		{"ordinary: test suite failed", errors.New("test suite failed: 3 assertions"), false},
		{"nil error", nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := providerAuthFailure(tc.err); got != tc.want {
				t.Fatalf("providerAuthFailure(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestAuthFailedDispatchClearsTrustWindow warms the cache through the plan 04
// gate, runs invalidatePreflightCacheOnProviderFailure with a results slice
// containing one auth-classified error, then calls the gate again and
// asserts the probe ran a second time with source "probe".
func TestAuthFailedDispatchClearsTrustWindow(t *testing.T) {
	withTestPreflightStore(t)
	now := time.Now()

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	if _, outcome := gatedProviderPreflight(context.Background(), fake, codex.PlatformClaude, t.TempDir(), now); outcome.Source != preflightSourceProbe {
		t.Fatalf("warm-up outcome.Source = %q, want probe", outcome.Source)
	}
	if fake.calls != 1 {
		t.Fatalf("warm-up probe count = %d, want 1", fake.calls)
	}

	results := []codex.DispatchResult{
		{WorkerName: "Scout-1", Error: errors.New("authentication failed")},
	}
	if cleared := invalidatePreflightCacheOnProviderFailure(codex.PlatformClaude, results); !cleared {
		t.Fatal("invalidatePreflightCacheOnProviderFailure returned false, want true for an auth-classified failure")
	}

	_, outcome := gatedProviderPreflight(context.Background(), fake, codex.PlatformClaude, t.TempDir(), now)
	if fake.calls != 2 {
		t.Fatalf("probe count after invalidation = %d, want 2 -- the trust window must have been cleared", fake.calls)
	}
	if outcome.Source != preflightSourceProbe {
		t.Fatalf("outcome.Source after invalidation = %q, want probe", outcome.Source)
	}
}

// TestOrdinaryFailureKeepsTrustWindow proves a flaky task cannot restore
// per-dispatch probe cost: ordinary worker failures must never clear the
// cache.
func TestOrdinaryFailureKeepsTrustWindow(t *testing.T) {
	withTestPreflightStore(t)
	now := time.Now()

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	if _, outcome := gatedProviderPreflight(context.Background(), fake, codex.PlatformClaude, t.TempDir(), now); outcome.Source != preflightSourceProbe {
		t.Fatalf("warm-up outcome.Source = %q, want probe", outcome.Source)
	}
	if fake.calls != 1 {
		t.Fatalf("warm-up probe count = %d, want 1", fake.calls)
	}

	results := []codex.DispatchResult{
		{WorkerName: "Scout-1", Error: errors.New("worker timed out after 600s")},
		{WorkerName: "Scout-2", Error: errors.New("task blocked: missing input file")},
	}
	if cleared := invalidatePreflightCacheOnProviderFailure(codex.PlatformClaude, results); cleared {
		t.Fatal("invalidatePreflightCacheOnProviderFailure returned true, want false for ordinary task failures")
	}

	_, outcome := gatedProviderPreflight(context.Background(), fake, codex.PlatformClaude, t.TempDir(), now)
	if fake.calls != 1 {
		t.Fatalf("probe count after ordinary failures = %d, want 1 -- the trust window must survive", fake.calls)
	}
	if outcome.Source != preflightSourceCache {
		t.Fatalf("outcome.Source after ordinary failures = %q, want cache", outcome.Source)
	}
}

// TestInvalidationIsPerPlatform warms both a Claude and a Codex entry,
// invalidates on a Claude auth failure, and asserts the Codex entry still
// hits.
func TestInvalidationIsPerPlatform(t *testing.T) {
	withTestPreflightStore(t)
	now := time.Now()

	claudeFake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	if _, outcome := gatedProviderPreflight(context.Background(), claudeFake, codex.PlatformClaude, t.TempDir(), now); outcome.Source != preflightSourceProbe {
		t.Fatalf("claude warm-up outcome.Source = %q, want probe", outcome.Source)
	}

	codexFake := &gatePreflightFake{platform: codex.PlatformCodex, available: true}
	if _, outcome := gatedProviderPreflight(context.Background(), codexFake, codex.PlatformCodex, t.TempDir(), now); outcome.Source != preflightSourceProbe {
		t.Fatalf("codex warm-up outcome.Source = %q, want probe", outcome.Source)
	}

	results := []codex.DispatchResult{
		{WorkerName: "Scout-1", Error: errors.New("authentication failed")},
	}
	if cleared := invalidatePreflightCacheOnProviderFailure(codex.PlatformClaude, results); !cleared {
		t.Fatal("invalidatePreflightCacheOnProviderFailure returned false, want true")
	}

	_, claudeOutcome := gatedProviderPreflight(context.Background(), claudeFake, codex.PlatformClaude, t.TempDir(), now)
	if claudeFake.calls != 2 {
		t.Fatalf("claude probe count = %d, want 2 -- claude's window was invalidated", claudeFake.calls)
	}
	if claudeOutcome.Source != preflightSourceProbe {
		t.Fatalf("claude outcome.Source = %q, want probe", claudeOutcome.Source)
	}

	_, codexOutcome := gatedProviderPreflight(context.Background(), codexFake, codex.PlatformCodex, t.TempDir(), now)
	if codexFake.calls != 1 {
		t.Fatalf("codex probe count = %d, want 1 -- a claude invalidation must never touch codex's window", codexFake.calls)
	}
	if codexOutcome.Source != preflightSourceCache {
		t.Fatalf("codex outcome.Source = %q, want cache", codexOutcome.Source)
	}
}

// TestInvalidationSafeWithoutStore proves invalidatePreflightCacheOnProviderFailure
// never panics when the package-level store is nil.
func TestInvalidationSafeWithoutStore(t *testing.T) {
	original := store
	store = nil
	t.Cleanup(func() { store = original })

	results := []codex.DispatchResult{
		{WorkerName: "Scout-1", Error: errors.New("authentication failed")},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("invalidatePreflightCacheOnProviderFailure panicked with nil store: %v", r)
		}
	}()
	invalidatePreflightCacheOnProviderFailure(codex.PlatformClaude, results)
}
