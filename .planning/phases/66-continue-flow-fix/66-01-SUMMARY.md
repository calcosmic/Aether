---
phase: 66-continue-flow-fix
plan: 01
subsystem: runtime
tags: [go, timeout, advisory, watcher, verification]

# Dependency graph
requires:
  - phase: 64.1
    provides: "external path advisory timeout pattern in codex_continue_finalize.go"
provides:
  - "internal watcher advisory timeout pattern mirroring external path"
  - "context.DeadlineExceeded detection in dispatch error and result paths"
affects: [66-02, continue-flow]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "advisory timeout: watcher timeout + verification passed = warning not block"

key-files:
  created: []
  modified:
    - cmd/codex_continue.go
    - cmd/codex_continue_test.go

key-decisions:
  - "Dispatch result path also checked for context.DeadlineExceeded (not just error path)"
  - "Two existing tests updated to reflect new advisory behavior (they asserted the bug)"

patterns-established:
  - "Advisory timeout pattern: watcher.Status == 'timeout' && checksPassed -> advisory warning"

requirements-completed: [FIX-01]

# Metrics
duration: 8min
completed: 2026-04-27
---

# Phase 66 Plan 01: Internal Watcher Advisory Timeout Summary

**Internal watcher timeout treated as advisory when build/types/lint/tests pass, mirroring Phase 64.1 external path fix**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-27T22:18:00Z
- **Completed:** 2026-04-27T22:25:51Z
- **Tasks:** 1 (TDD: RED + GREEN, no REFACTOR needed)
- **Files modified:** 2

## Accomplishments
- Fixed "stuck in continue loops" bug: internal watcher timeout no longer hard-blocks when runtime verification passes
- Added context.DeadlineExceeded detection in both dispatch error path and dispatch result path
- Advisory pattern: timeout + verification passed = warning with "runtime verification passed independently" message
- Four new tests covering advisory-on-timeout-passed, blocks-on-timeout-failed, blocks-on-genuine-failure, and timeout-status-from-context-deadline

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix internal watcher timeout detection and advisory behavior** - `fae4a93a` (test: RED phase)
   - `60bb7883` (feat: GREEN phase)

## Files Created/Modified
- `cmd/codex_continue.go` - Advisory timeout pattern in runCodexContinueVerification, context.DeadlineExceeded detection in runCodexContinueWatcherVerification
- `cmd/codex_continue_test.go` - 4 new tests, 2 existing tests updated to reflect correct behavior

## Decisions Made
- Dispatch result path also checked for context.DeadlineExceeded: `invokeDispatch` in `pkg/codex/dispatch.go` sets `Status: "failed"` when the invoker returns `context.DeadlineExceeded` as an error. Added a check in `runCodexContinueWatcherVerification` to override "failed" to "timeout" when `result.Error` is `context.DeadlineExceeded`. This keeps the fix scoped to the continue path without modifying the shared dispatch package.
- Two existing tests (`TestContinueBlocksWhenWatcherTimesOut`, `TestContinue_BlocksOnWatcherTimeout`) asserted the old buggy behavior (block on timeout even when verification passed). Updated both to assert the new correct advisory behavior and renamed them to reflect the change.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated two existing tests that asserted the bug being fixed**
- **Found during:** Task 1 (GREEN phase verification)
- **Issue:** `TestContinueBlocksWhenWatcherTimesOut` and `TestContinue_BlocksOnWatcherTimeout` both expected `blocked:true` when watcher times out but verification passes. This was the exact bug being fixed.
- **Fix:** Updated assertions to expect `blocked:false` (advisory), `advanced:true`, and `checks_passed:true`. Renamed tests to `TestContinueAdvisoryWhenWatcherTimesOutButVerificationPasses` and `TestContinue_AdvisoryOnWatcherTimeout`.
- **Files modified:** cmd/codex_continue_test.go
- **Verification:** All TestContinue tests pass
- **Committed in:** `60bb7883` (part of GREEN commit)

**2. [Rule 1 - Bug] Context.DeadlineExceeded also detected in dispatch result path**
- **Found during:** Task 1 (test 4 RED phase)
- **Issue:** Plan specified fixing the `err != nil` error path in `runCodexContinueWatcherVerification`, but `dispatchBatchByWaveWithVisuals` never returns errors for context deadlines. The `invokeDispatch` function catches context cancellation and returns a `DispatchResult` with `Status: "failed"` (not "timeout") when the invoker returns `context.DeadlineExceeded`.
- **Fix:** Added a check after `results[0]` processing: if `status == "failed"` and `result.Error` is `context.DeadlineExceeded`, override status to `"timeout"`. This keeps the fix in the continue layer without modifying `pkg/codex/dispatch.go`.
- **Files modified:** cmd/codex_continue.go
- **Verification:** Test 4 (`TestRunCodexContinueWatcherVerification_ContextDeadlineProducesTimeoutStatus`) passes
- **Committed in:** `60bb7883` (part of GREEN commit)

---

**Total deviations:** 2 auto-fixed (2 bug fixes)
**Impact on plan:** Both fixes were necessary for correctness. The plan's error-path fix was insufficient because context deadlines flow through the result path, not the error path. Existing tests needed updating because they encoded the bug.

## Issues Encountered
- Test 4 initially tried to test the error path (`dispatchBatchByWaveWithVisuals` returning `context.DeadlineExceeded`), but discovered that `DispatchWaveWithObserver` never returns errors -- it captures context cancellation in the dispatch result. Had to redesign the test to use a blocking invoker that triggers context cancellation, and add a result-path check for `context.DeadlineExceeded`.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Internal watcher advisory timeout pattern is complete and tested
- Plan 02 can proceed independently (different files)
- Root cause in `pkg/codex/dispatch.go` `invokeDispatch` not checked: could be addressed in a future phase if other callers need context deadline detection

---
*Phase: 66-continue-flow-fix*
*Completed: 2026-04-27*
