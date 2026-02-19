---
phase: 17-error-code-standardization
plan: 02
subsystem: error-handling
tags: [bash, error-codes, json-errors, shell-scripting, chamber-utils]

requires:
  - phase: 17-01
    provides: E_DEPENDENCY_MISSING and E_RESOURCE_NOT_FOUND constants, all aether-utils.sh json_err calls using E_* constants

provides:
  - chamber-utils.sh guarded json_err definition (yields to error-handler.sh when loaded)
  - chamber-compare.sh sources error-handler.sh directly and has guarded json_err fallback
  - All 20 bare-string json_err calls in chamber scripts converted to E_* constants with friendly messages
  - Override bug fixed: error-handler.sh's enhanced json_err no longer overwritten by chamber-utils.sh

affects:
  - 17-03 (error-codes.md documentation — chamber constants now fully documented scope)
  - Phase 14 test workaround (if (.error | type) == "object") now unnecessary — can be cleaned up

tech-stack:
  added: []
  patterns:
    - "Guard pattern for utility scripts that define json_err as fallback: if ! type json_err &>/dev/null"
    - "Fallback E_* constants using no-op assignment: : \"${E_NAME:=E_NAME}\""
    - "Standalone scripts source error-handler.sh from SCRIPT_DIR with conditional guard"

key-files:
  created: []
  modified:
    - .aether/utils/chamber-utils.sh
    - .aether/utils/chamber-compare.sh

key-decisions:
  - "Guard pattern chosen over removing local json_err — preserves standalone fallback while yielding to error-handler.sh when loaded"
  - "chamber-compare.sh sources error-handler.sh directly since it always runs standalone (bash chamber-compare.sh)"
  - "Phase 14 test workaround (if (.error | type) == \"object\") left in place — not in this plan's scope, deferred to cleanup pass"

patterns-established:
  - "Guard pattern: if ! type json_err &>/dev/null — any utility script that is both sourced and run standalone should use this"
  - "Source + guard combination for standalone scripts: source first, guard fallback second"

requirements-completed:
  - ERR-02

duration: 2min
completed: 2026-02-19
---

# Phase 17 Plan 02: Chamber Script json_err Override Fix Summary

**Chamber-utils.sh and chamber-compare.sh json_err override bug fixed with guard pattern; all 20 bare-string calls converted to E_* constants**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-19T13:53:01Z
- **Completed:** 2026-02-19T13:55:48Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Fixed the chamber override bug: chamber-utils.sh now uses `if ! type json_err` guard so error-handler.sh's enhanced json_err is preserved when loaded via aether-utils.sh
- chamber-compare.sh now sources error-handler.sh directly at startup, ensuring enhanced json_err and E_* constants are available in standalone execution
- Converted all 15 bare-string json_err calls in chamber-utils.sh to E_* constants with friendly "Try:" messages
- Converted all 5 bare-string json_err calls in chamber-compare.sh to E_* constants with friendly "Try:" messages
- All 18 bash tests pass with no regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Guard chamber-utils.sh json_err and convert 15 bare-string calls** - `1c846c9` (feat)
2. **Task 2: Source error-handler.sh in chamber-compare.sh, guard json_err, convert 5 calls** - `48eba7d` (feat)

**Plan metadata:** (see below)

## Files Created/Modified

- `.aether/utils/chamber-utils.sh` - Added guard pattern around json_err definition; added 5 E_* fallback constants; converted all 15 bare-string calls to use E_* constants
- `.aether/utils/chamber-compare.sh` - Added source of error-handler.sh; added 3 E_* fallback constants; replaced unconditional json_err with guarded fallback; converted all 5 bare-string calls

## Decisions Made

- **Guard pattern over removal:** Retained the local json_err definition but wrapped it with `if ! type json_err &>/dev/null`. This preserves the standalone fallback behavior while yielding to error-handler.sh's enhanced version when sourced through aether-utils.sh.
- **chamber-compare.sh sources error-handler.sh:** Since chamber-compare.sh always runs as `bash chamber-compare.sh` (never sourced), it needs to actively load error-handler.sh. The conditional source `[[ -f "$SCRIPT_DIR/error-handler.sh" ]] && source` ensures it fails gracefully if the file is missing.
- **Phase 14 workaround left intact:** The test at test-aether-utils.sh line 844 (`if (.error | type) == "object"`) is now technically unnecessary since the override bug is fixed, but removing it is a cleanup task outside this plan's scope.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. The pre-existing `user-modification-detection.test.js` unit test failure (unexpected `process.exit()`) was confirmed pre-existing via stash check and is unrelated to this work.

## Next Phase Readiness

- `grep 'json_err "[^\$]' .aether/utils/chamber-utils.sh` returns 0 — chamber-utils.sh is clean
- `grep 'json_err "[^\$]' .aether/utils/chamber-compare.sh` returns 0 — chamber-compare.sh is clean
- ERR-02 fully complete: zero bare-string json_err calls in all three affected files (aether-utils.sh, chamber-utils.sh, chamber-compare.sh)
- Phase 17-03 can proceed with error-codes.md documentation

---
*Phase: 17-error-code-standardization*
*Completed: 2026-02-19*

## Self-Check: PASSED

- FOUND: .aether/utils/chamber-utils.sh
- FOUND: .aether/utils/chamber-compare.sh
- FOUND: .planning/phases/17-error-code-standardization/17-02-SUMMARY.md
- FOUND: 1c846c9 (Task 1 commit)
- FOUND: 48eba7d (Task 2 commit)
