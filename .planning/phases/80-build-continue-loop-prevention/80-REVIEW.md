---
phase: 80-build-continue-loop-prevention
reviewed: 2026-04-30T00:00:00Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - cmd/circuit_breaker_test.go
  - cmd/codex_continue.go
  - cmd/codex_continue_test.go
  - pkg/colony/colony.go
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 80: Code Review Report

**Reviewed:** 2026-04-30T00:00:00Z
**Depth:** standard
**Files Reviewed:** 4
**Status:** issues_found

## Summary

Reviewed the build-continue loop prevention implementation across four files: the new `CircuitBreaker` type and its tests, the continue flow integration in `codex_continue.go` (and its extensive test suite), and the `colony.go` types that underpin the state model.

The core loop prevention mechanisms are sound: the circuit breaker correctly trips per-worker after configurable consecutive failures, the watcher failure counter auto-skips after the threshold, and parameter loop detection correctly compares options snapshots across invocations. The atomic state commit pattern in continue is well-implemented with proper side-effect ordering.

Two warnings and three info items were identified. No critical/blocker issues found.

## Warnings

### WR-01: Test isolation leak -- `continueSignalHousekeeper` not restored after mutation

**File:** `cmd/codex_continue_test.go:817,903,992`
**Issue:** Three test functions mutate the package-level `continueSignalHousekeeper` variable to inject a failing housekeeper, but none of them restore the original value via `t.Cleanup()`. The affected tests are:

- `TestContinueStateCommittedBeforeHousekeeping` (line 817)
- `TestContinueDoesNotCloseBuildWorkersWhenHousekeepingFails` (line 903)
- `TestContinueDoesNotRewriteContextWhenHousekeepingFails` (line 992)

By contrast, the test at line 3613 (`TestContinueStateAdvancesEvenWhenReportSaveFails`) correctly saves the original and restores it in `t.Cleanup()`.

When these tests run in isolation they pass, but if a subsequent test in the same package expects `continueSignalHousekeeper` to behave normally (the default behavior that calls `runSignalHousekeepingWithState`), it will get the failing stub instead. With `go test -count=1` and parallel test execution, this can cause cascading failures in unrelated tests.

**Fix:**
```go
// In each of the three affected test functions, add before the mutation:
origHousekeeper := continueSignalHousekeeper
t.Cleanup(func() {
    continueSignalHousekeeper = origHousekeeper
})
```

### WR-02: Watcher failure counter errors silently discarded

**File:** `cmd/codex_continue.go:1220,1223`
**Issue:** The LOOP-01 watcher failure counter updates use `_ =` to discard errors from `incrementWatcherFailureCount` and `resetWatcherFailureCount`. Both functions perform an atomic JSON update on `COLONY_STATE.json` and can fail (e.g., file lock contention, state corruption, phase not found). If the increment fails, the counter never reaches the auto-skip threshold, and the watcher will be dispatched indefinitely on every `continue` call -- the exact loop the counter is meant to prevent.

The error is logged nowhere, so the operator has no visibility into why auto-skip never activates.

**Fix:** At minimum, log the error to stderr so operators can diagnose why auto-skip is not triggering:
```go
if continueWatcher.Present && !continueWatcher.Passed && continueWatcher.Status == "failed" {
    if err := incrementWatcherFailureCount(phase.ID); err != nil {
        fmt.Fprintf(os.Stderr, "warning: failed to increment watcher failure count for phase %d: %v\n", phase.ID, err)
    }
} else if continueWatcher.Passed && continueWatcher.Status != "skipped" {
    if err := resetWatcherFailureCount(phase.ID); err != nil {
        fmt.Fprintf(os.Stderr, "warning: failed to reset watcher failure count for phase %d: %v\n", phase.ID, err)
    }
}
```

## Info

### IN-01: Misleading condition in `evaluateContinueWatcherVerification`

**File:** `cmd/codex_continue.go:1447`
**Issue:** The skip condition uses `&&` where the semantic intent is to find a dispatch that is a watcher in the verification stage. The current condition:

```go
if stage != "verification" && caste != "watcher" {
    continue
}
```

This processes a dispatch if it is EITHER in the verification stage OR has watcher caste. A builder dispatch in the verification stage would be mistakenly treated as a watcher. In practice this never happens because the manifest always pairs verification stage with watcher caste, so this is not a runtime bug, but the condition is semantically misleading.

**Fix:** Use `||` in the skip condition to match "must be both verification stage AND watcher caste":
```go
if stage != "verification" || caste != "watcher" {
    continue
}
```

### IN-02: Circuit breaker event emission has a theoretical TOCTOU gap

**File:** `cmd/circuit_breaker.go:115-126`
**Issue:** `emitCircuitBreakerTripped` acquires the lock separately from `RecordFailure`. Between `RecordFailure` returning `true` (tripped) and `emitCircuitBreakerTripped` reading the failure count under its own lock, another goroutine could theoretically call `RecordSuccess` on the same worker, resetting the count to 0. The emitted event would then say "tripped after 0 consecutive failures."

In practice this cannot happen because each goroutine handles a unique `WorkerName`, so no two goroutines operate on the same worker. But the lock-splitting pattern is fragile.

**Fix:** Consider having `RecordFailure` return the count and threshold directly, avoiding the second lock acquisition:
```go
func (cb *CircuitBreaker) RecordFailure(workerName string) (tripped bool, count int, threshold int) {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    cb.failures[workerName]++
    if cb.failures[workerName] >= cb.threshold {
        cb.tripped[workerName] = true
        return true, cb.failures[workerName], cb.threshold
    }
    return false, cb.failures[workerName], cb.threshold
}
```

### IN-03: `colony.go` is a types-only file with no behavioral changes in this phase

**File:** `pkg/colony/colony.go`
**Issue:** The `colony.go` file was included in the review scope but contains only data type definitions and JSON unmarshaling logic. No changes in this phase modify this file. The `WatcherFailureCount` field on `Phase` (line 329) and `GateResultEntry` type (line 147) were likely added in a prior phase. Reviewing this file found no issues -- the types are well-structured with proper backward compatibility handling in the custom `UnmarshalJSON` methods.

---

_Reviewed: 2026-04-30T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
