---
phase: 13-monolith-modularization
plan: 07
subsystem: infra
tags: [bash, modularization, shell-modules, learning-pipeline, instinct-system]

requires:
  - phase: 13-monolith-modularization
    provides: Swarm domain extraction pattern and non-contiguous block one-liner dispatch contract (Plan 06)
provides:
  - Learning/instinct domain extracted to .aether/utils/learning.sh (13 subcommands from 3 non-contiguous ranges)
  - Full memory pipeline (observe -> check-promotion -> promote-auto -> instinct) functional via module dispatch
  - Smoke test pattern replicated for learning module
affects: [13-08, 13-09]

tech-stack:
  added: []
  patterns: [three-range-non-contiguous-extraction, cross-domain-subprocess-preservation]

key-files:
  created:
    - .aether/utils/learning.sh
    - tests/bash/test-learning-module.sh
  modified:
    - .aether/aether-utils.sh

key-decisions:
  - "Verbatim extraction of 3 non-contiguous blocks -- same no-refactoring policy as Plans 01-06"
  - "get_wisdom_threshold and get_wisdom_thresholds_json stay in main file -- shared by queen and learning domains"
  - "memory-capture stays in main file -- uses learning-observe and learning-promote-auto as subprocess calls, not a learning domain function"

patterns-established:
  - "Three-range extraction: blocks at lines ~1586, ~3528, and ~5560 all extracted to single module with consistent one-liner dispatches"
  - "Cross-domain subprocess preservation: queen-promote, pheromone-write, activity-log, rolling-summary, generate-threshold-bar, parse-selection all kept as bash $0 subprocess calls"

requirements-completed: [QUAL-06]

duration: 10min
completed: 2026-03-24
---

# Phase 13 Plan 07: Learning/Instinct Domain Extraction Summary

**13 learning/instinct subcommands (~1481 lines) extracted from 3 non-contiguous blocks in aether-utils.sh into utils/learning.sh with full memory pipeline intact**

## Performance

- **Duration:** 10 min
- **Started:** 2026-03-24T09:10:20Z
- **Completed:** 2026-03-24T09:20:41Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Extracted 13 subcommands from 3 non-contiguous blocks (learning-promote/inject at ~line 1586, learning-observe through learning-undo-promotions at ~line 3528, instinct-read/create/apply at ~line 5560) into self-contained module
- Reduced aether-utils.sh by 1481 lines (8706 -> 7225)
- Created learning.sh module (1552 lines) -- second-largest domain extraction in phase 13
- All 584 existing tests pass with zero regressions
- 4 new smoke tests validating module extraction
- Full memory pipeline (observe -> check-promotion -> promote-auto -> instinct) verified functional

## Task Commits

Each task was committed atomically:

1. **Task 1: Extract learning/instinct domain into learning.sh module** - `f00d79b` (feat)
2. **Task 2: Create learning module smoke tests** - `f8d031d` (test)

## Files Created/Modified
- `.aether/utils/learning.sh` - New module containing 13 learning/instinct domain functions
- `.aether/aether-utils.sh` - Replaced multi-line case blocks with one-liner dispatches, added source line
- `tests/bash/test-learning-module.sh` - Smoke tests for extracted learning module

## Decisions Made
- Verbatim extraction with no refactoring -- structural move only, preserving all SUPPRESS:OK comments and error handling exactly as they were
- get_wisdom_threshold() and get_wisdom_thresholds_json() remain in the main file because they are shared by both the queen and learning domains (established in Plan 05)
- memory-capture stays in the main file -- it orchestrates learning-observe and learning-promote-auto via subprocess dispatch, but is itself a separate subcommand, not a learning domain function
- All cross-domain subprocess calls (queen-promote, pheromone-write, activity-log, rolling-summary, generate-threshold-bar, parse-selection) preserved as bash "$0" dispatch

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Learning/instinct extraction validates the three-range non-contiguous extraction pattern (most complex multi-block extraction in phase 13)
- One-liner dispatch contract continues to work across all 584 tests
- Smoke test pattern ready to replicate for subsequent modules
- aether-utils.sh at 7225 lines, ready for next extraction (Plan 08)

## Self-Check: PASSED

All artifacts verified:
- .aether/utils/learning.sh: FOUND
- tests/bash/test-learning-module.sh: FOUND
- 13-07-SUMMARY.md: FOUND
- Commit f00d79b: FOUND
- Commit f8d031d: FOUND

---
*Phase: 13-monolith-modularization*
*Completed: 2026-03-24*
