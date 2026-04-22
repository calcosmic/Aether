---
phase: 31-p0-runtime-truth-fixes
plan: 03
subsystem: state-management
tags: [atomic-write, colony-state, continue-flow, storage]

# Dependency graph
requires:
  - phase: 31-p0-runtime-truth-fixes
    provides: UpdateJSONAtomically helper and continue advancement reordering
provides:
  - Atomic colony state advancement via UpdateJSONAtomically
  - State saved before side effects and report writes
  - UpdateJSONAtomically transaction-scoped helper in pkg/storage
  - Ordering tests proving state-before-report guarantee
affects: [continue-flow, colony-state, storage-layer]

# Tech tracking
tech-stack:
  added: []
  patterns: [atomic-read-modify-write, state-before-side-effects]

key-files:
  created: []
  modified:
    - cmd/codex_continue.go
    - cmd/codex_continue_test.go
    - pkg/storage/storage.go
    - pkg/storage/storage_malformed_test.go

key-decisions:
  - "State committed before side effects: housekeeping/context/closures run after durable state save"
  - "Side-effect events appended in best-effort second save after side effects complete"
  - "Report save failures do not roll back state (by design)"
  - "UpdateJSONAtomically wraps read-mutate-write in single locked cycle"

patterns-established:
  - "Atomic state commit: UpdateJSONAtomically for colony state mutations"
  - "State before reports: COLONY_STATE.json saved before continue.json"

requirements-completed: [R051]

# Metrics
duration: 24min
completed: 2026-04-22
---

# Phase 31 Plan 03: Atomic Phase Advancement Summary

**Colony state advancement uses atomic read-modify-write with side effects and reports deferred until state is durable**

## Performance

- **Duration:** 24 min
- **Started:** 2026-04-22T20:24:22Z
- **Completed:** 2026-04-22T20:48:XXZ
- **Tasks:** 4
- **Files modified:** 4

## Accomplishments

- Colony state is saved BEFORE side effects (housekeeping, context updates, worker closures) and report writes
- `UpdateJSONAtomically` provides a transaction-scoped helper that rolls back on mutation errors
- Phase advancement uses a single locked read-modify-write cycle via `store.UpdateJSONAtomically`
- Two ordering tests prove the state-before-report guarantee using timestamps and chmod interception

## Task Commits

Each task was committed atomically:

1. **Task 03-01: Reorder runCodexContinue to save state before side effects** - `431251fa` (feat)
2. **Task 03-02: Add transaction-scoped state update helper in pkg/storage** - `dbd5e4e1` (feat)
3. **Task 03-03: Refactor runCodexContinue to use the atomic update helper** - `e1161a2f` (refactor)
4. **Task 03-04: Add tests for atomic phase advancement** - `832592ff` (test)

## Files Created/Modified

- `cmd/codex_continue.go` - Reordered advancement to save state before side effects; refactored to use UpdateJSONAtomically
- `cmd/codex_continue_test.go` - Updated rollback tests to reflect state-committed-first semantics; added ordering tests
- `pkg/storage/storage.go` - Added UpdateJSONAtomically method for transaction-scoped JSON updates
- `pkg/storage/storage_malformed_test.go` - Added rollback and commit tests for UpdateJSONAtomically

## Decisions Made

- **State committed before side effects:** The colony state is saved immediately after computing the advancement mutation, before housekeeping, context updates, worker closures, or report writes. This means side-effect failures do NOT roll back the state -- the state remains valid and consistent.
- **Best-effort event append:** Worker flow events (from housekeeping, review workers) are appended to the state in a best-effort second save after side effects complete. If this second save fails, the core advancement is still durable.
- **Report save failures are non-fatal to state:** When the continue report save fails, the state is already committed. The function returns an error but the state on disk is valid.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated tests to match new state-commit-first semantics**
- **Found during:** Task 03-01
- **Issue:** Existing tests `TestContinueRollsBackStateWhenContextUpdateFails` and `TestContinueDoesNotAdvanceStateWhenHousekeepingFails` expected state to NOT be saved when side effects fail. With the new ordering, state IS saved before side effects.
- **Fix:** Renamed tests to `TestContinueStateCommittedBeforeContextUpdate` and `TestContinueStateCommittedBeforeHousekeeping`, updated assertions to verify state IS persisted when side effects fail.
- **Files modified:** cmd/codex_continue_test.go
- **Committed in:** 431251fa

**2. [Rule 2 - Missing Critical] Added best-effort second state save for worker flow events**
- **Found during:** Task 03-01
- **Issue:** Moving the state save before side effects meant worker flow events (from housekeeping, review) were not persisted to disk, since they are computed during side effects.
- **Fix:** Added a best-effort `store.SaveJSON("COLONY_STATE.json", updated)` after side effects complete, to persist the worker flow events.
- **Files modified:** cmd/codex_continue.go
- **Committed in:** 431251fa

**3. [Rule 3 - Blocking] Adapted report save failure test to use chmod during housekeeping hook**
- **Found during:** Task 03-04
- **Issue:** Cannot directly mock `store.SaveJSON` since `Store` is a concrete struct. Needed a way to make the continue report save fail while verification/gate report saves succeed.
- **Fix:** Used the `continueSignalHousekeeper` hook to chmod the report directory read-only after state is saved but before the report write. Also adjusted test to check stderr instead of return error (Cobra swallows errors).
- **Files modified:** cmd/codex_continue_test.go
- **Committed in:** 832592ff

---

**Total deviations:** 3 auto-fixed (1 bug, 1 missing critical, 1 blocking)
**Impact on plan:** All auto-fixes necessary for correctness. No scope creep.

## Issues Encountered

- The `continueCmd.RunE` function swallows errors via `outputError` and returns nil to Cobra. Tests checking for report save failures must inspect stderr rather than the Execute return value.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Colony state advancement is now atomic and ordered correctly
- `UpdateJSONAtomically` is available for future state mutations
- All continue tests pass with the new semantics

---
*Phase: 31-p0-runtime-truth-fixes*
*Completed: 2026-04-22*

## Self-Check: PASSED

All 4 modified files verified present. All 4 task commits verified in git history.
