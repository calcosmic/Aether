package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
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

// errPreflightCacheFutureSchema aborts a cache write when the on-disk file
// was written by a newer binary (schema_version above ours). Mixed binary
// versions against one repo are a real situation (aether vs aether-dev
// channels): an older binary must treat the newer file as a cache miss and
// leave it untouched, never stamp schema_version 1 over half-decoded future
// entries (WR-05). Returned from the UpdateJSONAtomically mutation, which
// guarantees no write occurs.
var errPreflightCacheFutureSchema = errors.New("preflight cache: file has an unknown future schema_version; refusing to rewrite it")

// preflightCacheUnmarshalError reports whether err came from failing to
// decode an existing (corrupt) cache file, as opposed to a lock, read, or
// write failure. UpdateJSONAtomically wraps the json error with %w, so
// errors.As sees through it.
func preflightCacheUnmarshalError(err error) bool {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	return errors.As(err, &syntaxErr) || errors.As(err, &typeErr)
}

// recordPreflightSuccess persists a fresh success for status.Platform so a
// later dispatch can skip the paid probe. Failures, empty platforms, and
// PlatformUnknown are all no-ops — only successes are ever cached. Other
// platforms' entries survive via load-merge-save.
//
// A corrupt existing file (partial write from a crash, manual edit) must not
// disable caching forever: UpdateJSONAtomically refuses to mutate content it
// cannot decode, so on an unmarshal-classified error the corrupt file is
// replaced wholesale with a fresh single-entry cache — a cache is
// disposable, and overwriting a corrupt one is always safe. A loud one-line
// notice is printed so the self-heal is never silent.
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
	entry := preflightCacheEntry{
		Platform:  platformKey,
		Binary:    status.Binary,
		Available: true,
		Category:  string(status.Category),
		CheckedAt: now.UTC().Format(time.RFC3339),
	}

	var file preflightCacheFile
	err := store.UpdateJSONAtomically(preflightCachePathRel, &file, func() error {
		if file.SchemaVersion != 0 && file.SchemaVersion != preflightCacheSchemaVersion {
			return errPreflightCacheFutureSchema
		}
		if file.Entries == nil {
			file.Entries = map[string]preflightCacheEntry{}
		}
		file.SchemaVersion = preflightCacheSchemaVersion
		file.Entries[platformKey] = entry
		return nil
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, errPreflightCacheFutureSchema) {
		// A future-schema file is simply not our cache to write: skip the
		// record (the read path already treats it as a miss) and leave the
		// newer binary's file intact.
		return nil
	}
	if !preflightCacheUnmarshalError(err) {
		return err
	}

	fmt.Fprintf(stderr, "preflight: cache file %s was corrupt — rewriting it fresh\n", preflightCachePathRel)
	fresh := preflightCacheFile{
		SchemaVersion: preflightCacheSchemaVersion,
		Entries:       map[string]preflightCacheEntry{platformKey: entry},
	}
	return store.SaveJSON(preflightCachePathRel, &fresh)
}

// clearPreflightCache removes platform's cached entry so the next command
// re-probes (D-03 invalidation on provider failure). A missing file or a
// missing entry is success, not an error. A future-schema file is left
// untouched (WR-05): its entries were never trusted by this binary's read
// path, so there is nothing of ours to clear — and rewriting it would splice
// half-decoded future entries into a v1 file.
func clearPreflightCache(platform codex.Platform) error {
	if store == nil {
		return nil
	}

	platformKey := string(platform)
	var file preflightCacheFile
	err := store.UpdateJSONAtomically(preflightCachePathRel, &file, func() error {
		if file.SchemaVersion != 0 && file.SchemaVersion != preflightCacheSchemaVersion {
			return errPreflightCacheFutureSchema
		}
		if file.Entries == nil {
			return nil
		}
		delete(file.Entries, platformKey)
		return nil
	})
	if errors.Is(err, errPreflightCacheFutureSchema) {
		return nil
	}
	return err
}

// preflightSkipNoticeLine returns the exact one-line text (no trailing
// newline) stating that the preflight probe was skipped and auth/model
// config were NOT verified. This is D-06's never-silent requirement, and the
// same bytes are used for both the CLI notice (emitPreflightSkipNotice) and
// the JSON preflightOutcome.Notice field so the two surfaces cannot drift.
func preflightSkipNoticeLine() string {
	return fmt.Sprintf("Warning: preflight skipped via %s — provider auth and model config were NOT verified before dispatch", envSkipPreflight)
}

// emitPreflightSkipNotice writes a single loud line stating that the
// preflight probe was skipped and auth/model config were NOT verified. This
// is D-06's never-silent requirement.
func emitPreflightSkipNotice(w io.Writer) {
	fmt.Fprintf(w, "%s\n", preflightSkipNoticeLine())
}

// preflightCacheHitNoticeLine returns the exact one-line text (no trailing
// newline) reporting that a cached preflight success was reused, and how
// much of the trust window remains. The same bytes are used for both the
// CLI notice (emitPreflightCacheHitNotice) and the JSON
// preflightOutcome.Notice field so the two surfaces cannot drift.
func preflightCacheHitNoticeLine(platform codex.Platform, remaining time.Duration) string {
	minutes := int(remaining / time.Minute)
	if minutes < 0 {
		minutes = 0
	}
	return fmt.Sprintf("preflight: cached OK for %s, %dm left in trust window", platform, minutes)
}

// emitPreflightCacheHitNotice writes a single line reporting that a cached
// preflight success was reused, and how much of the trust window remains.
func emitPreflightCacheHitNotice(w io.Writer, platform codex.Platform, remaining time.Duration) {
	fmt.Fprintf(w, "%s\n", preflightCacheHitNoticeLine(platform, remaining))
}

// preflightOutcome is the source-and-notice pair returned by
// gatedProviderPreflight, printed on the direct-Go CLI path and carried
// across the adapter boundary in the TS host's preflight JSON field.
type preflightOutcome struct {
	Source string `json:"source"`
	Notice string `json:"notice,omitempty"`
}

// Source constants for preflightOutcome. These are the exact strings the TS
// host contract in plan 03 reads from response.preflight.source.
const (
	preflightSourceProbe   = "probe"
	preflightSourceCache   = "cache"
	preflightSourceSkipped = "skipped"
)

// gatedProviderPreflight is the single skip/cache/probe/record decision both
// dispatch chokepoints (cmd/dispatch_runtime.go preflightWorkerProvider and
// cmd/internal_worker_adapter.go's --preflight branch) call, so a direct-Go
// dispatch and a TS-hosted dispatch share one trust window instead of two.
//
// Decision order (a skip must short-circuit everything, including the cache
// read):
//  1. AETHER_SKIP_PREFLIGHT -> Available/probe_skipped, source "skipped".
//     The cache is never touched: skipping must not write a trust window.
//  2. An empty or PlatformUnknown platform bypasses the cache entirely and
//     goes straight to the probe (source "probe", no notice) -- an
//     unkeyable platform must never share another platform's window.
//  3. A fresh cache hit -> Available, source "cache". No probe is invoked.
//  4. Otherwise the real probe runs. A success is recorded (best-effort; a
//     cache write failure must not fail dispatch). A failure is never
//     cached and never clears an existing entry here.
func gatedProviderPreflight(ctx context.Context, preflighter codex.WorkerProviderPreflighter, platform codex.Platform, root string, now time.Time) (codex.AvailabilityStatus, preflightOutcome) {
	if preflightSkipRequested() {
		status := codex.AvailabilityStatus{
			Platform:  platform,
			Available: true,
			Category:  codex.AvailabilityCategoryProbeSkipped,
			Reason:    fmt.Sprintf("preflight skipped via %s", envSkipPreflight),
		}
		return status, preflightOutcome{Source: preflightSourceSkipped, Notice: preflightSkipNoticeLine()}
	}

	keyable := platform != "" && platform != codex.PlatformUnknown
	if keyable {
		if entry, remaining, hit := loadFreshPreflightCache(platform, now); hit {
			status := codex.AvailabilityStatus{
				Platform:  platform,
				Binary:    entry.Binary,
				Available: true,
				Category:  codex.AvailabilityCategoryAvailable,
			}
			return status, preflightOutcome{Source: preflightSourceCache, Notice: preflightCacheHitNoticeLine(platform, remaining)}
		}
	}

	if preflighter == nil {
		return codex.AvailabilityStatus{}, preflightOutcome{Source: preflightSourceProbe}
	}

	status := preflighter.Preflight(ctx, root)
	if status.Available {
		_ = recordPreflightSuccess(status, now)
	}
	return status, preflightOutcome{Source: preflightSourceProbe}
}

// providerAuthFailureCategories are the AvailabilityCategory values that mean
// the provider itself, not the task, is broken -- a lapsed login, missing
// credentials, or invalid provider config.
var providerAuthFailureCategories = map[codex.AvailabilityCategory]bool{
	codex.AvailabilityCategoryAuthProbeFailed:    true,
	codex.AvailabilityCategoryAuthInactive:       true,
	codex.AvailabilityCategoryInvalidAuthOutput:  true,
	codex.AvailabilityCategoryCredentialsMissing: true,
	codex.AvailabilityCategoryProviderConfig:     true,
}

// providerAuthFailureRegexp is the Go behavioural twin of
// .aether/ts-host/src/worker-dispatch.ts isAuthError. Keep the two patterns
// byte-identical -- this is the vocabulary both hosts agree marks a
// provider/auth failure worth clearing the trust window for.
var providerAuthFailureRegexp = regexp.MustCompile(`(?i)\b(auth(?:entication|orization)?|credentials?|login|permission denied|api[_ -]?key)\b`)

// providerAuthFailure reports whether err represents a provider/auth failure
// (D-03): either a workerProviderPreflightError carrying one of the five
// provider/auth AvailabilityCategory values, or an error whose message
// matches the auth vocabulary mirrored from the TS isAuthError classifier.
//
// This inspects ONLY the error value -- never a worker's self-reported report
// text, its full captured process output, or its task description. A worker
// whose task is literally about authentication must not invalidate the cache
// just for describing its work.
func providerAuthFailure(err error) bool {
	if err == nil {
		return false
	}
	var preflightErr *workerProviderPreflightError
	if errors.As(err, &preflightErr) && providerAuthFailureCategories[preflightErr.status.Category] {
		return true
	}
	return providerAuthFailureRegexp.MatchString(err.Error())
}

// invalidatePreflightCacheOnProviderFailure clears platform's cached trust
// window when any result's Error, or its WorkerResult's Error, is a
// provider/auth failure (D-03). Returns true when a clear was triggered. A
// clear failure is best-effort -- log nothing and never fail the batch.
func invalidatePreflightCacheOnProviderFailure(platform codex.Platform, results []codex.DispatchResult) bool {
	for _, result := range results {
		if providerAuthFailure(result.Error) {
			_ = clearPreflightCache(platform)
			return true
		}
		if result.WorkerResult != nil && providerAuthFailure(result.WorkerResult.Error) {
			_ = clearPreflightCache(platform)
			return true
		}
	}
	return false
}
