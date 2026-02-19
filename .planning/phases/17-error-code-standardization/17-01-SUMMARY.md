---
phase: 17-error-code-standardization
plan: 01
subsystem: error-handling
tags: [bash, error-codes, json-errors, shell-scripting]

requires:
  - phase: 14-foundation-safety
    provides: fallback json_err function that emits structured JSON with code parameter

provides:
  - E_DEPENDENCY_MISSING and E_RESOURCE_NOT_FOUND constants fully wired in error-handler.sh
  - All 27 bare-string json_err calls in aether-utils.sh converted to E_* constants
  - Friendly error messages with "Try:" suggestions across all json_err calls

affects:
  - 17-02 (chamber-utils.sh bare-string json_err fix — same pattern)
  - Any downstream tooling that parses json_err output by code field

tech-stack:
  added: []
  patterns:
    - "json_err always takes E_* constant as first arg, friendly English message as second"
    - "Error messages use 'Couldn't find...' tone, always include 'Try:' suggestion"
    - "E_DEPENDENCY_MISSING for missing utility files/binaries, E_RESOURCE_NOT_FOUND for missing runtime state"

key-files:
  created: []
  modified:
    - .aether/utils/error-handler.sh
    - .aether/aether-utils.sh

key-decisions:
  - "Error message format locked: friendly tone ('Couldn't find...') + always include 'Try:' suggestion"
  - "E_DEPENDENCY_MISSING for missing utility scripts/binaries; E_RESOURCE_NOT_FOUND for missing runtime state (sessions, etc.)"
  - "All 10 CONTEXT.md not-found calls harmonized to identical message — same error condition"
  - "xmllint errors use E_FEATURE_UNAVAILABLE not E_DEPENDENCY_MISSING — xmllint is optional XML feature, not a hard dep"

patterns-established:
  - "json_err '$E_CODE' 'Friendly message. Try: actionable fix.' — mandatory pattern for all new error calls"
  - "Recovery functions (_recovery_*) provide generic fallback; specific Try: in message provides context-specific guidance"

requirements-completed:
  - ERR-02

duration: 4min
completed: 2026-02-19
---

# Phase 17 Plan 01: Error Code Standardization — Core Fix Summary

**E_DEPENDENCY_MISSING and E_RESOURCE_NOT_FOUND added to error-handler.sh; all 27 bare-string json_err calls in aether-utils.sh converted to E_* constants with friendly "Try:" messages**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-19T13:46:09Z
- **Completed:** 2026-02-19T13:50:10Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added 2 missing E_* constants (E_DEPENDENCY_MISSING, E_RESOURCE_NOT_FOUND) to error-handler.sh with recovery functions, case entries, and exports
- Converted all 27 bare-string json_err calls in aether-utils.sh to use E_* constants with friendly messages
- Updated 2 existing E_* calls (lines ~3875 and ~5104) to the new friendly message style with "Try:" suggestions
- Fallback definitions for both new constants added to aether-utils.sh; 394 tests pass with no regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Add E_DEPENDENCY_MISSING and E_RESOURCE_NOT_FOUND constants** - `e2b9609` (feat)
2. **Task 2: Convert all 29 bare-string json_err calls to E_* constants** - `73be694` (feat)

**Plan metadata:** (see below)

## Files Created/Modified

- `.aether/utils/error-handler.sh` - Added 2 new constants, 2 recovery functions, 2 case entries, exports for both constants and functions
- `.aether/aether-utils.sh` - Fallback definitions for 2 new constants; 27 bare-string json_err calls converted + 2 existing calls improved

## Decisions Made

- Error message format locked: friendly tone ("Couldn't find...") plus "Try:" suggestion required on all errors
- xmllint errors use E_FEATURE_UNAVAILABLE, not E_DEPENDENCY_MISSING — xmllint is an optional XML capability, not a hard dependency
- All 10 CONTEXT.md not-found calls harmonized to the same message since they represent the same error condition
- E_DEPENDENCY_MISSING reserved for missing utility scripts/binaries; E_RESOURCE_NOT_FOUND for missing runtime state (sessions, active files)

## Deviations from Plan

None - plan executed exactly as written.

The plan expected `grep -c 'E_DEPENDENCY_MISSING' error-handler.sh` to return >= 4, but returns 3. This is because the recovery function is named `_recovery_dependency_missing` and does not contain the literal string `E_DEPENDENCY_MISSING`. All four logical requirements (definition, recovery function, case entry, export) are present and correct.

## Issues Encountered

None.

## Next Phase Readiness

- `grep 'json_err "[^\$]' .aether/aether-utils.sh` returns 0 — aether-utils.sh is clean
- Phase 17-02 can now fix chamber-utils.sh and chamber-compare.sh bare-string json_err overrides (documented in STATE.md as blocker, now unblocked for fixing)

---
*Phase: 17-error-code-standardization*
*Completed: 2026-02-19*

## Self-Check: PASSED

- FOUND: .aether/utils/error-handler.sh
- FOUND: .aether/aether-utils.sh
- FOUND: 17-01-SUMMARY.md
- FOUND: e2b9609 (Task 1 commit)
- FOUND: 73be694 (Task 2 commit)
