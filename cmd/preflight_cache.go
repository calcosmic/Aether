package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// preflightCachePathRel is store-relative, so the cache lands at
// .aether/data/preflight-cache.json (honoring COLONY_DATA_DIR automatically
// via the store's path resolution).
const preflightCachePathRel = "preflight-cache.json"

const preflightCacheSchemaVersion = 1

// preflightCacheDefaultTTL is a var (not a const) so tests can shrink it with
// the shrinkPreflightTimeout cleanup idiom from
// pkg/codex/preflight_retry_test.go:64.
var preflightCacheDefaultTTL = 1 * time.Hour

// Env var names for the preflight cost-policy layer (D-02, D-06).
const (
	envPreflightCacheTTL = "AETHER_PREFLIGHT_CACHE_TTL"
	envSkipPreflight     = "AETHER_SKIP_PREFLIGHT"
)

// preflightCacheEntry records a single platform's most recent preflight
// result, so a fresh success can be trusted without re-running the paid
// probe.
type preflightCacheEntry struct {
	Platform  string `json:"platform"`
	Binary    string `json:"binary,omitempty"`
	Available bool   `json:"available"`
	Category  string `json:"category,omitempty"`
	CheckedAt string `json:"checked_at"`
}

// preflightCacheFile is the on-disk shape of the cache. Entries are keyed by
// platform so switching AETHER_WORKER_PLATFORM back and forth never destroys
// another platform's trust window, and a lookup for platform X never sees
// platform Y's entry (D-03 "keyed per platform").
type preflightCacheFile struct {
	SchemaVersion int                            `json:"schema_version"`
	Entries       map[string]preflightCacheEntry `json:"entries"`
}

// resolvedPreflightCacheTTL returns the trust window duration, honoring the
// AETHER_PREFLIGHT_CACHE_TTL env var (Go duration syntax, e.g. "90s") so the
// window can be widened or narrowed without a rebuild. Invalid or
// non-positive values fall back to the compiled default — a broken env var
// must not brick dispatch. Never errors.
func resolvedPreflightCacheTTL() time.Duration {
	envValue := strings.TrimSpace(os.Getenv(envPreflightCacheTTL))
	if envValue == "" {
		return preflightCacheDefaultTTL
	}
	ttl, err := time.ParseDuration(envValue)
	if err != nil || ttl <= 0 {
		return preflightCacheDefaultTTL
	}
	return ttl
}

// preflightSkipRequested reports whether AETHER_SKIP_PREFLIGHT explicitly
// opts out of the probe. Only a small set of truthy tokens count; anything
// else (including "0", "false", "banana", or empty) is false.
func preflightSkipRequested() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(envSkipPreflight)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// preflightCacheEntryIsFresh reports whether entry is still inside its trust
// window as of now. Only recorded successes are ever trusted; an empty or
// unparseable checked_at, a timestamp more than a minute in the future
// (clock-skew guard, copied from sealFinalReviewIsFresh), or a stale
// timestamp are all rejected.
func preflightCacheEntryIsFresh(entry preflightCacheEntry, now time.Time, ttl time.Duration) bool {
	if !entry.Available {
		return false
	}
	checkedAt := strings.TrimSpace(entry.CheckedAt)
	if checkedAt == "" {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, checkedAt)
	if err != nil {
		return false
	}
	if parsed.After(now.Add(1 * time.Minute)) {
		return false
	}
	return now.Sub(parsed) <= ttl
}

// loadFreshPreflightCache looks up a fresh, available cache entry for
// platform. Any read/unmarshal error, a missing store, a schema mismatch, a
// missing entry, or a stale entry are all treated as a miss (never a
// failure). On a hit it also returns the remaining trust window (ttl minus
// elapsed, floored at zero).
func loadFreshPreflightCache(platform codex.Platform, now time.Time) (preflightCacheEntry, time.Duration, bool) {
	if store == nil {
		return preflightCacheEntry{}, 0, false
	}

	var file preflightCacheFile
	if err := store.LoadJSON(preflightCachePathRel, &file); err != nil {
		return preflightCacheEntry{}, 0, false
	}
	if file.SchemaVersion != preflightCacheSchemaVersion {
		return preflightCacheEntry{}, 0, false
	}

	platformKey := string(platform)
	entry, ok := file.Entries[platformKey]
	if !ok {
		return preflightCacheEntry{}, 0, false
	}
	if entry.Platform != platformKey {
		return preflightCacheEntry{}, 0, false
	}

	ttl := resolvedPreflightCacheTTL()
	if !preflightCacheEntryIsFresh(entry, now, ttl) {
		return preflightCacheEntry{}, 0, false
	}

	checkedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(entry.CheckedAt))
	if err != nil {
		return preflightCacheEntry{}, 0, false
	}
	remaining := ttl - now.Sub(checkedAt)
	if remaining < 0 {
		remaining = 0
	}
	return entry, remaining, true
}

// recordPreflightSuccess persists a fresh success for status.Platform so a
// later dispatch can skip the paid probe. Failures, empty platforms, and
// PlatformUnknown are all no-ops — only successes are ever cached. Other
// platforms' entries survive via load-merge-save.
func recordPreflightSuccess(status codex.AvailabilityStatus, now time.Time) error {
	if store == nil {
		return nil
	}
	if !status.Available {
		return nil
	}
	if status.Platform == "" || status.Platform == codex.PlatformUnknown {
		return nil
	}

	platformKey := string(status.Platform)
	var file preflightCacheFile
	return store.UpdateJSONAtomically(preflightCachePathRel, &file, func() error {
		if file.Entries == nil {
			file.Entries = map[string]preflightCacheEntry{}
		}
		file.SchemaVersion = preflightCacheSchemaVersion
		file.Entries[platformKey] = preflightCacheEntry{
			Platform:  platformKey,
			Binary:    status.Binary,
			Available: true,
			Category:  string(status.Category),
			CheckedAt: now.UTC().Format(time.RFC3339),
		}
		return nil
	})
}

// clearPreflightCache removes platform's cached entry so the next command
// re-probes (D-03 invalidation on provider failure). A missing file or a
// missing entry is success, not an error.
func clearPreflightCache(platform codex.Platform) error {
	if store == nil {
		return nil
	}

	platformKey := string(platform)
	var file preflightCacheFile
	return store.UpdateJSONAtomically(preflightCachePathRel, &file, func() error {
		if file.Entries == nil {
			return nil
		}
		delete(file.Entries, platformKey)
		return nil
	})
}

// emitPreflightSkipNotice writes a single loud line stating that the
// preflight probe was skipped and auth/model config were NOT verified. This
// is D-06's never-silent requirement.
func emitPreflightSkipNotice(w io.Writer) {
	fmt.Fprintf(w, "Warning: preflight skipped via %s — provider auth and model config were NOT verified before dispatch\n", envSkipPreflight)
}

// emitPreflightCacheHitNotice writes a single line reporting that a cached
// preflight success was reused, and how much of the trust window remains.
func emitPreflightCacheHitNotice(w io.Writer, platform codex.Platform, remaining time.Duration) {
	minutes := int(remaining / time.Minute)
	if minutes < 0 {
		minutes = 0
	}
	fmt.Fprintf(w, "preflight: cached OK for %s, %dm left in trust window\n", platform, minutes)
}
