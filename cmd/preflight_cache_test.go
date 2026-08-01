package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/storage"
)

// withTestPreflightStore points the package-level store at a fresh temp-dir
// store for the duration of the test, restoring the previous value on
// cleanup, so no test touches the real .aether/data/.
func withTestPreflightStore(t *testing.T) *storage.Store {
	t.Helper()
	s, _ := newTestStore(t)
	original := store
	store = s
	t.Cleanup(func() { store = original })
	return s
}

func TestResolvedPreflightCacheTTLEnvOverride(t *testing.T) {
	if preflightCacheDefaultTTL != 1*time.Hour {
		t.Fatalf("preflightCacheDefaultTTL = %v, want 1h (a silent default change must fail this test)", preflightCacheDefaultTTL)
	}

	t.Setenv(envPreflightCacheTTL, "90s")
	if got := resolvedPreflightCacheTTL(); got != 90*time.Second {
		t.Fatalf("resolvedPreflightCacheTTL = %v, want 90s", got)
	}

	t.Setenv(envPreflightCacheTTL, "banana")
	if got := resolvedPreflightCacheTTL(); got != preflightCacheDefaultTTL {
		t.Fatalf("invalid env value must fall back to default, got %v", got)
	}

	t.Setenv(envPreflightCacheTTL, "-5s")
	if got := resolvedPreflightCacheTTL(); got != preflightCacheDefaultTTL {
		t.Fatalf("non-positive env value must fall back to default, got %v", got)
	}

	t.Setenv(envPreflightCacheTTL, "")
	if got := resolvedPreflightCacheTTL(); got != preflightCacheDefaultTTL {
		t.Fatalf("empty env value must use default, got %v", got)
	}
}

func TestPreflightSkipRequestedParsing(t *testing.T) {
	truthy := []string{"1", "true", "TRUE", "yes", "on"}
	for _, v := range truthy {
		t.Run("truthy_"+v, func(t *testing.T) {
			t.Setenv(envSkipPreflight, v)
			if !preflightSkipRequested() {
				t.Fatalf("preflightSkipRequested(%q) = false, want true", v)
			}
		})
	}

	falsy := []string{"", "0", "false", "no", "banana"}
	for _, v := range falsy {
		t.Run("falsy_"+v, func(t *testing.T) {
			t.Setenv(envSkipPreflight, v)
			if preflightSkipRequested() {
				t.Fatalf("preflightSkipRequested(%q) = true, want false", v)
			}
		})
	}
}

func TestPreflightCacheEntryFreshness(t *testing.T) {
	now := time.Now().UTC()
	ttl := 1 * time.Hour

	fresh := preflightCacheEntry{Available: true, CheckedAt: now.Format(time.RFC3339)}
	if !preflightCacheEntryIsFresh(fresh, now, ttl) {
		t.Fatal("entry stamped now must be fresh")
	}

	stale := preflightCacheEntry{Available: true, CheckedAt: now.Add(-2 * time.Hour).Format(time.RFC3339)}
	if preflightCacheEntryIsFresh(stale, now, ttl) {
		t.Fatal("entry older than the TTL must be stale")
	}

	future := preflightCacheEntry{Available: true, CheckedAt: now.Add(2 * time.Minute).Format(time.RFC3339)}
	if preflightCacheEntryIsFresh(future, now, ttl) {
		t.Fatal("entry stamped 2 minutes in the future must be rejected (clock skew)")
	}

	empty := preflightCacheEntry{Available: true, CheckedAt: ""}
	if preflightCacheEntryIsFresh(empty, now, ttl) {
		t.Fatal("empty checked_at must be rejected")
	}

	unparseable := preflightCacheEntry{Available: true, CheckedAt: "not-a-timestamp"}
	if preflightCacheEntryIsFresh(unparseable, now, ttl) {
		t.Fatal("unparseable checked_at must be rejected")
	}

	unavailable := preflightCacheEntry{Available: false, CheckedAt: now.Format(time.RFC3339)}
	if preflightCacheEntryIsFresh(unavailable, now, ttl) {
		t.Fatal("available:false entry must be rejected even when the timestamp is fresh")
	}
}

func TestPreflightCacheIsKeyedPerPlatform(t *testing.T) {
	withTestPreflightStore(t)
	now := time.Now().UTC()

	claudeStatus := codex.AvailabilityStatus{Platform: codex.PlatformClaude, Available: true}
	if err := recordPreflightSuccess(claudeStatus, now); err != nil {
		t.Fatalf("recordPreflightSuccess(claude) = %v, want nil", err)
	}

	if _, _, hit := loadFreshPreflightCache(codex.PlatformCodex, now); hit {
		t.Fatal("codex platform must be a miss after only claude was recorded")
	}
	if _, _, hit := loadFreshPreflightCache(codex.PlatformClaude, now); !hit {
		t.Fatal("claude platform must still hit")
	}

	codexStatus := codex.AvailabilityStatus{Platform: codex.PlatformCodex, Available: true}
	if err := recordPreflightSuccess(codexStatus, now); err != nil {
		t.Fatalf("recordPreflightSuccess(codex) = %v, want nil", err)
	}

	if _, _, hit := loadFreshPreflightCache(codex.PlatformClaude, now); !hit {
		t.Fatal("claude entry must survive recording a codex success (merge, not overwrite)")
	}
	if _, _, hit := loadFreshPreflightCache(codex.PlatformCodex, now); !hit {
		t.Fatal("codex platform must hit after being recorded")
	}
}

func TestPreflightCacheRoundTripAndClear(t *testing.T) {
	withTestPreflightStore(t)
	now := time.Now().UTC()

	status := codex.AvailabilityStatus{Platform: codex.PlatformClaude, Available: true}
	if err := recordPreflightSuccess(status, now); err != nil {
		t.Fatalf("recordPreflightSuccess = %v, want nil", err)
	}

	ttl := resolvedPreflightCacheTTL()
	entry, remaining, hit := loadFreshPreflightCache(codex.PlatformClaude, now)
	if !hit {
		t.Fatal("expected a hit after recording a success")
	}
	if !entry.Available {
		t.Fatal("hit entry must be available")
	}
	if remaining <= 0 || remaining > ttl {
		t.Fatalf("remaining = %v, want in (0, %v]", remaining, ttl)
	}

	if err := clearPreflightCache(codex.PlatformClaude); err != nil {
		t.Fatalf("clearPreflightCache = %v, want nil", err)
	}
	if _, _, hit := loadFreshPreflightCache(codex.PlatformClaude, now); hit {
		t.Fatal("expected a miss after clearing the cache")
	}

	if err := clearPreflightCache(codex.PlatformClaude); err != nil {
		t.Fatalf("clearing an already-cleared platform must return nil, got %v", err)
	}
}

func TestPreflightCacheRejectsUnknownSchemaVersion(t *testing.T) {
	s := withTestPreflightStore(t)
	now := time.Now().UTC()

	file := preflightCacheFile{
		SchemaVersion: 99,
		Entries: map[string]preflightCacheEntry{
			string(codex.PlatformClaude): {
				Platform:  string(codex.PlatformClaude),
				Available: true,
				CheckedAt: now.Format(time.RFC3339),
			},
		},
	}
	if err := s.SaveJSON(preflightCachePathRel, &file); err != nil {
		t.Fatalf("SaveJSON = %v, want nil", err)
	}

	if _, _, hit := loadFreshPreflightCache(codex.PlatformClaude, now); hit {
		t.Fatal("an unknown schema_version must be treated as a miss")
	}
}

func TestPreflightCacheHelpersSafeWithoutStore(t *testing.T) {
	original := store
	store = nil
	t.Cleanup(func() { store = original })

	now := time.Now().UTC()

	if _, _, hit := loadFreshPreflightCache(codex.PlatformClaude, now); hit {
		t.Fatal("loadFreshPreflightCache with nil store must be a miss")
	}

	status := codex.AvailabilityStatus{Platform: codex.PlatformClaude, Available: true}
	if err := recordPreflightSuccess(status, now); err != nil {
		t.Fatalf("recordPreflightSuccess with nil store = %v, want nil", err)
	}

	if err := clearPreflightCache(codex.PlatformClaude); err != nil {
		t.Fatalf("clearPreflightCache with nil store = %v, want nil", err)
	}
}

func TestPreflightNoticesAreLoudAndSingleLine(t *testing.T) {
	var skipBuf bytes.Buffer
	emitPreflightSkipNotice(&skipBuf)
	skipOut := skipBuf.String()
	if !strings.Contains(skipOut, "AETHER_SKIP_PREFLIGHT") {
		t.Fatalf("skip notice missing AETHER_SKIP_PREFLIGHT: %q", skipOut)
	}
	if !strings.Contains(skipOut, "NOT verified") {
		t.Fatalf("skip notice missing 'NOT verified': %q", skipOut)
	}
	if strings.Count(skipOut, "\n") != 1 {
		t.Fatalf("skip notice must contain exactly one newline, got %d: %q", strings.Count(skipOut, "\n"), skipOut)
	}

	var hitBuf bytes.Buffer
	emitPreflightCacheHitNotice(&hitBuf, codex.PlatformClaude, 43*time.Minute)
	hitOut := hitBuf.String()
	if !strings.Contains(hitOut, "cached OK") {
		t.Fatalf("cache-hit notice missing 'cached OK': %q", hitOut)
	}
	if !strings.Contains(hitOut, string(codex.PlatformClaude)) {
		t.Fatalf("cache-hit notice missing platform name: %q", hitOut)
	}
	if strings.Count(hitOut, "\n") != 1 {
		t.Fatalf("cache-hit notice must contain exactly one newline, got %d: %q", strings.Count(hitOut, "\n"), hitOut)
	}
}
