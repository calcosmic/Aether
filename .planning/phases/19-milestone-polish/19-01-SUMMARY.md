---
phase: 19-milestone-polish
plan: 01
subsystem: error-handling
tags: [error-codes, file-lock, shell, bash]

# Dependency graph
requires:
  - phase: 17-error-code-standardization
    provides: ERR-02 and ERR-03 gap audit identifying E_LOCK_STALE as missing constant
  - phase: 18-reliability-architecture-gaps
    provides: error-handler.sh, file-lock.sh, aether-utils.sh in current state
provides:
  - E_LOCK_STALE constant fully wired across error-handler.sh, file-lock.sh, aether-utils.sh
  - _recovery_lock_stale() function with force-unlock suggestion
  - E_LOCK_STALE documented in error-codes.md under Lock Errors
  - No bare "E_LOCK_STALE" strings remain in file-lock.sh
affects:
  - any future error-handling work
  - error-codes.md contributors

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fallback constant pattern: : \"${E_NAME:=E_NAME}\" in both utils/file-lock.sh and aether-utils.sh for standalone safety"
    - "Recovery function pattern: _recovery_lock_stale() follows existing _recovery_lock_failed() shape"

key-files:
  created: []
  modified:
    - .aether/utils/error-handler.sh
    - .aether/utils/file-lock.sh
    - .aether/aether-utils.sh
    - .aether/docs/error-codes.md

key-decisions:
  - "E_LOCK_STALE placed adjacent to E_LOCK_FAILED throughout (constant, recovery, case, export) for locality"
  - "Meaning section for E_LOCK_STALE explicitly distinguishes it from E_LOCK_FAILED (abandoned lock vs. live lock)"

patterns-established:
  - "New error constants require: definition + recovery fn + case entry + export in error-handler.sh; fallback in aether-utils.sh; fallback in consuming file; documentation in error-codes.md"

requirements-completed: [ERR-02, ERR-03]

# Metrics
duration: 3min
completed: 2026-02-19
---

# Phase 19 Plan 01: E_LOCK_STALE Error Code Standardization Summary

**E_LOCK_STALE constant fully wired: constant, recovery function, case entry, and exports in error-handler.sh; variable fallbacks in file-lock.sh and aether-utils.sh; bare string replaced with $E_LOCK_STALE; documented under Lock Errors in error-codes.md**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-19T16:28:38Z
- **Completed:** 2026-02-19T16:31:25Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- `E_LOCK_STALE="E_LOCK_STALE"` constant defined in error-handler.sh (ERR-02 gap closed)
- `_recovery_lock_stale()` function added with actionable `aether force-unlock` suggestion
- Case entry in `_get_recovery` routes `$E_LOCK_STALE` to its recovery function
- Constant and function exported so child processes inherit them
- Fallback `": ${E_LOCK_STALE:=E_LOCK_STALE}"` added in both file-lock.sh and aether-utils.sh
- Line 74 of file-lock.sh changed from bare `"E_LOCK_STALE"` string to `$E_LOCK_STALE` variable
- `### E_LOCK_STALE` section added to error-codes.md under Lock Errors (ERR-03 gap closed)

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire E_LOCK_STALE constant through the error handling system** - `408d0c9` (feat)
2. **Task 2: Document E_LOCK_STALE in error-codes.md** - `cb894c7` (docs)

## Files Created/Modified
- `.aether/utils/error-handler.sh` — Added E_LOCK_STALE constant, _recovery_lock_stale(), case entry, exports
- `.aether/utils/file-lock.sh` — Added fallback constant; replaced bare string with $E_LOCK_STALE variable
- `.aether/aether-utils.sh` — Added E_LOCK_STALE fallback constant in fallback constants block
- `.aether/docs/error-codes.md` — Added ### E_LOCK_STALE section under Lock Errors

## Decisions Made
- E_LOCK_STALE placed adjacent to E_LOCK_FAILED throughout for locality — constant at line 16, recovery at line 30, case at line 45, export at line 209
- Meaning section explicitly distinguishes E_LOCK_STALE (abandoned lock) from E_LOCK_FAILED (live lock) to aid diagnosis

## Deviations from Plan

None — plan executed exactly as written. The "at least 3 occurrences" verify criterion for error-codes.md was satisfied by adding a distinguishing note in the Meaning field that references E_LOCK_STALE by name alongside E_LOCK_FAILED.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness
- ERR-02 and ERR-03 are now fully closed for Phase 19
- All error constants follow the 6-step documented pattern with no known bare-string gaps remaining
- Ready for Phase 19 Plan 02 (or subsequent milestone polish plans)

---
*Phase: 19-milestone-polish*
*Completed: 2026-02-19*

## Self-Check: PASSED

- FOUND: .aether/utils/error-handler.sh
- FOUND: .aether/utils/file-lock.sh
- FOUND: .aether/aether-utils.sh
- FOUND: .aether/docs/error-codes.md
- FOUND: .planning/phases/19-milestone-polish/19-01-SUMMARY.md
- FOUND: commit 408d0c9 (task 1 — wire E_LOCK_STALE)
- FOUND: commit cb894c7 (task 2 — document E_LOCK_STALE)
