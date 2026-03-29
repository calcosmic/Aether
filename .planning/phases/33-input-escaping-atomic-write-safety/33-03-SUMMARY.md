---
phase: 33-input-escaping-atomic-write-safety
plan: 03
subsystem: infra
tags: [bash, locking, atomic-write, safety-stats, trap]

requires:
  - phase: 33-02
    provides: "json_ok escaping patterns across utils"
provides:
  - "Trap-based lock cleanup on all acquire_lock callers in pheromone.sh"
  - "Safety stats tracking for stale lock cleanups and JSON validation rejects"
  - "Documented lock/atomic_write responsibility separation"
affects: [status-display, data-safety, lock-management]

tech-stack:
  added: []
  patterns:
    - "trap 'release_lock 2>/dev/null || true' EXIT after every acquire_lock"
    - "Best-effort safety stats via _safety_stats_increment (never fails calling operation)"

key-files:
  created: []
  modified:
    - ".aether/utils/pheromone.sh"
    - ".aether/utils/atomic-write.sh"
    - ".aether/utils/file-lock.sh"

key-decisions:
  - "Trap-based cleanup is the standard pattern; existing explicit release_lock calls kept as defense-in-depth"
  - "Safety stats are best-effort and never fail the calling operation"
  - "Safety stats stored in .aether/data/safety-stats.json (local-only DATA_DIR)"

patterns-established:
  - "Lock safety pattern: acquire_lock -> trap EXIT -> do work -> release_lock + trap - EXIT"
  - "Safety stats: _safety_stats_increment for best-effort event tracking"

requirements-completed: [SAFE-04]

duration: 27min
completed: 2026-03-29
---

# Phase 33 Plan 03: Lock Safety & Atomic Write Hardening Summary

**Trap-based lock cleanup on all pheromone.sh lock-acquiring functions, safety stats tracking for stale lock cleanups and JSON validation rejects**

## Performance

- **Duration:** 27 min
- **Started:** 2026-03-29T05:12:30Z
- **Completed:** 2026-03-29T05:39:00Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Audited all acquire_lock callers across 6 files; found 3 functions in pheromone.sh missing trap-based cleanup
- Added EXIT traps to pheromone-write, pheromone-expire (2 lock sections), and eternal-store
- Added safety stats tracking (stale_locks_cleaned, json_validation_rejects) with best-effort _safety_stats_increment helper
- Documented that atomic_write does NOT interact with locks (caller responsibility)

## Task Commits

Each task was committed atomically:

1. **Task 1: Audit all acquire_lock callers and ensure lock release on every exit path** - `4623cb4` (fix)
2. **Task 2: Harden atomic_write JSON validation and stale lock auto-expiry** - `cfa4c55` (feat)

## Files Created/Modified
- `.aether/utils/pheromone.sh` - Added trap-based lock cleanup to pheromone-write, pheromone-expire, eternal-store
- `.aether/utils/atomic-write.sh` - Added _safety_stats_increment helper, JSON validation reject tracking, lock responsibility documentation
- `.aether/utils/file-lock.sh` - Added stale lock cleanup tracking in auto and prompt modes

## Decisions Made
- Trap-based cleanup is the standard pattern; explicit release_lock calls before json_err are kept as defense-in-depth (double-release is safe since release_lock checks LOCK_ACQUIRED)
- Safety stats are best-effort: _safety_stats_increment uses return 0 on all error paths so stats tracking never fails the calling operation
- file-lock.sh calls _safety_stats_increment via `type ... &>/dev/null && ...` guard since file-lock.sh is sourced before atomic-write.sh where the function is defined

## Deviations from Plan

### Audit Results (No Changes Needed)

The audit of learning.sh, midden.sh, hive.sh, and flag.sh found all four files already had correct trap-based lock release patterns from prior work. Only pheromone.sh had missing traps.

### Stash Contamination Recovery (Rule 3 - Blocking)

During execution, git stash operations contaminated the working tree with uncommitted changes from a prior plan execution. Required careful git checkout to isolate only the target file changes. No code impact -- only workflow interruption.

---

**Total deviations:** 0 auto-fixed code issues. Audit found 4 of 6 audited files already correct.
**Impact on plan:** Plan executed as written. Fewer changes needed than anticipated because prior work had already established the pattern in 4 of 6 files.

## Issues Encountered
None affecting correctness.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All acquire_lock callers now have trap-based cleanup
- Safety stats file tracks data safety events
- Ready for Plan 04 (if applicable)

## Self-Check: PASSED
- All 3 modified files exist
- Both commits (4623cb4, cfa4c55) found in git history
- trap statement present in pheromone.sh
- _safety_stats_increment present in atomic-write.sh and file-lock.sh
- 616 tests pass

---
*Phase: 33-input-escaping-atomic-write-safety*
*Completed: 2026-03-29*
