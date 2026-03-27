---
phase: 13-monolith-modularization
plan: 04
subsystem: infra
tags: [bash, modularization, shell-modules, suggest-system, pheromone-suggestions]

requires:
  - phase: 13-monolith-modularization
    provides: Session domain extraction pattern and dual-range one-liner dispatch contract (Plan 03)
provides:
  - Suggest domain extracted to .aether/utils/suggest.sh (6 subcommands + get_type_emoji helper)
  - One-liner dispatch pattern validated for contiguous block extraction
  - Smoke test pattern replicated for suggest module
affects: [13-05, 13-06, 13-07, 13-08, 13-09]

tech-stack:
  added: []
  patterns: [contiguous-block-extraction, helper-function-migration, subprocess-dispatch-preservation]

key-files:
  created:
    - .aether/utils/suggest.sh
    - tests/bash/test-suggest-module.sh
  modified:
    - .aether/aether-utils.sh

key-decisions:
  - "Verbatim extraction of contiguous block -- same no-refactoring policy as Plans 01-03"
  - "get_type_emoji moved into suggest.sh -- only caller is _suggest_approve, keeps helper co-located"
  - "Cross-domain pheromone-write calls preserved as subprocess dispatch (bash $0) -- no conversion to direct function calls"

patterns-established:
  - "Helper function migration: get_type_emoji moved alongside its only caller (_suggest_approve) to keep module self-contained"
  - "Subprocess dispatch preservation: cross-domain calls via bash $0 remain unchanged during extraction"

requirements-completed: [QUAL-07]

duration: 5min
completed: 2026-03-24
---

# Phase 13 Plan 04: Suggest Domain Extraction Summary

**6 suggest subcommands (~580 lines) plus get_type_emoji helper extracted from aether-utils.sh into utils/suggest.sh with one-liner dispatch entries and smoke tests**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-24T08:39:13Z
- **Completed:** 2026-03-24T08:44:27Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Extracted 6 suggest subcommands (suggest-analyze, suggest-record, suggest-check, suggest-clear, suggest-approve, suggest-quick-dismiss) into self-contained module
- Moved get_type_emoji helper function into suggest.sh (only used by suggest-approve)
- Reduced aether-utils.sh by 579 lines (10713 -> 10134)
- Created suggest.sh module (611 lines) handling contiguous block extraction
- All 584 existing tests pass with zero regressions
- 3 new smoke tests validating module extraction

## Task Commits

Each task was committed atomically:

1. **Task 1: Extract suggest domain into suggest.sh module** - `961a149` (feat)
2. **Task 2: Create suggest module smoke tests** - `5249253` (test)

## Files Created/Modified
- `.aether/utils/suggest.sh` - New module containing 6 suggest domain functions plus get_type_emoji helper
- `.aether/aether-utils.sh` - Replaced multi-line case blocks with one-liner dispatches, added source line
- `tests/bash/test-suggest-module.sh` - Smoke tests for extracted suggest module

## Decisions Made
- Verbatim extraction with no refactoring -- structural move only, preserving all SUPPRESS:OK comments and error handling exactly as they were
- get_type_emoji moved into suggest.sh alongside suggest-approve (its only caller) to keep the module self-contained and avoid orphaning the helper in the main file
- Cross-domain pheromone-write calls in suggest-approve preserved as subprocess dispatch (bash "$0" pheromone-write) -- not converted to direct function calls per plan instructions

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Suggest extraction validates contiguous block extraction pattern
- One-liner dispatch contract continues to work across all 584 tests
- Smoke test pattern ready to replicate for subsequent modules
- aether-utils.sh at 10134 lines, ready for next extraction (Plan 05)

## Self-Check: PASSED

All artifacts verified:
- .aether/utils/suggest.sh: FOUND
- tests/bash/test-suggest-module.sh: FOUND
- 13-04-SUMMARY.md: FOUND
- Commit 961a149: FOUND
- Commit 5249253: FOUND

---
*Phase: 13-monolith-modularization*
*Completed: 2026-03-24*
