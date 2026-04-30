---
plan: 80-01
phase: 80-build-continue-loop-prevention
status: complete
requirements: [LOOP-01, LOOP-02]
---

# Plan 80-01: Watcher Failure Tracking & Recovery Loop Prevention

## Objective
Add watcher failure tracking with auto-skip (LOOP-01) and recovery command loop-back prevention (LOOP-02) to the continue flow.

## What Was Built

### LOOP-01: Watcher Auto-Skip After Consecutive Failures
- Added `WatcherFailureCount int` field to `Phase` struct in `pkg/colony/colony.go` (omitempty for backward compat)
- Added `defaultWatcherFailureThreshold = 3` constant
- Added `getWatcherFailureCount`, `incrementWatcherFailureCount`, `resetWatcherFailureCount` helpers using `store.UpdateJSONAtomically`
- Added auto-skip branch in watcher decision chain: when failure count >= threshold, watcher is auto-skipped with diagnostic message
- Counter increments on `status="failed"` only (not timeout -- advisory per Phase 64.1)
- Counter resets to 0 on genuine watcher pass (not on skip)

### LOOP-02: Recovery Command Loop-Back Prevention
- Added `codexContinueOptionsJSON` type for serializing options to continue.json
- Added `LastContinueOptions` field to `codexContinueReport` struct
- Added `continueOptionsToJSON` converter and `continueOptionsMatchCurrent` comparison function
- Added `loadLastContinueOptions` reader with nil store guard
- Modified `buildContinueVerificationTimeoutRecoveryCommand` and `buildContinueTimeoutRecoveryCommand` to accept last options and ensure suggested timeout doubles if it matches the last invocation
- Updated `continueNextCommandForBlocked` to load last options and fall back to `aether build N --force` when no different-parameter recovery exists (D-10)
- Updated all `continueNextCommandForBlocked` call sites with `phaseID` argument
- Added `LastContinueOptions` to all `codexContinueReport` save sites

### Tests (8 new, all passing)
- `TestContinueAutoSkipsWatcherAfterConsecutiveFailures` -- watcher skipped after 3 failures
- `TestContinueWatcherFailureCounterResetsOnPass` -- counter resets on pass
- `TestContinueWatcherTimeoutDoesNotIncrementCounter` -- timeouts don't increment
- `TestContinueWatcherFailureCountPerPhase` -- independent per-phase tracking
- `TestRecoveryCommandDiffersFromLastInvocation` -- timeout doubles from last
- `TestRecoveryFallbackToBuildForce` -- falls back to build --force on match
- `TestContinueOptionsMatchCurrent` -- parameter comparison function
- `TestContinueAutoSkipsWhenWorkerPlatformUnavailable` -- existing test still passes

## Deviations
- Agent died mid-execution; uncommitted test work was rescued and committed manually
- Added nil guard on `store` in `loadLastContinueOptions` to prevent panic in test environments

## Files Modified
- `pkg/colony/colony.go` -- Phase struct: +1 field
- `cmd/codex_continue.go` -- +172 lines: types, helpers, auto-skip logic, recovery comparison
- `cmd/codex_continue_test.go` -- +489 lines: 8 new test functions

## Self-Check: PASSED
- All acceptance criteria verified via grep
- All 8 new tests pass
- Full test suite passes: `go test ./cmd/ -count=1`
