---
phase: 32-intelligence-enhancements
plan: 03
subsystem: init
tags: [bash, testing, integration-tests, scan, pheromone, governance, colony-context]

# Dependency graph
requires:
  - phase: 32-01
    provides: "_scan_colony_context, _scan_governance, _scan_pheromone_suggestions in scan.sh"
provides:
  - "17 integration tests covering all 3 intelligence sub-scan functions"
  - "npm test:intelligence script for running intelligence tests independently"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Scan function shim: source scan.sh directly with minimal dependency stubs to test internal functions without triggering main dispatch"
    - "charter-write subcommand used in tests to populate QUEEN.md reliably instead of brittle sed inserts"

key-files:
  created:
    - "tests/bash/test-intelligence.sh"
  modified:
    - "package.json"

key-decisions:
  - "Scan functions tested via minimal shim sourcing scan.sh directly (not through aether-utils.sh dispatch) for isolation"
  - "Shim uses set -uo pipefail without set -e because scan functions contain piped commands that return non-zero on empty results"
  - "Charter entries populated via charter-write subcommand rather than sed to match real usage patterns"

requirements-completed: [INTEL-01, INTEL-02, INTEL-03]

# Metrics
duration: 7min
completed: 2026-03-27
---

# Phase 32 Plan 03: Intelligence Sub-Scan Integration Tests Summary

**17 integration tests covering colony context extraction, pheromone suggestions, and governance inference -- all pass with 0 failures**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-27T18:09:15Z
- **Completed:** 2026-03-27T18:16:26Z
- **Tasks:** 2
- **Files created:** 1
- **Files modified:** 1

## Accomplishments
- 5 tests for _scan_colony_context: chambers extraction, no-chambers empty result, manifest-only (no CROWNED-ANTHILL.md), max-3 cap, existing charter from QUEEN.md
- 6 tests for _scan_pheromone_suggestions: .env REDIRECT, test config+tests FOCUS, config-no-tests REDIRECT (cross-reference), empty repo, 5-cap truncation, priority sort order
- 5 tests for _scan_governance: CONTRIBUTING.md detection, TDD with tests, TDD without tests (cross-reference skip), CI/CD detection, empty repo
- 1 integration test verifying init-research returns colony_context, governance, and pheromone_suggestions fields
- npm test:intelligence script wired and added to test:all for full suite inclusion
- All 616 existing tests continue to pass with no regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Create test-intelligence.sh with 17 integration tests** - `16edca5` (test)
2. **Task 2: Wire npm test:intelligence script** - `3a32d5c` (chore)

## Files Created/Modified
- `tests/bash/test-intelligence.sh` - 824 lines, 17 integration tests covering all 3 intelligence sub-scan functions
- `package.json` - Added test:intelligence script, updated test:all to include intelligence tests

## Decisions Made
- Scan functions tested through a lightweight shim that sources scan.sh directly and stubs json_ok/json_err/error constants, avoiding the aether-utils.sh main dispatch
- Shim avoids `set -e` because scan functions contain pipes like `ls | sort` that return non-zero on empty directories under pipefail
- Charter content for test 5 populated via charter-write subcommand (same as production code) rather than manual sed inserts into QUEEN.md

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed scan function invocation approach**
- **Found during:** Task 1 (initial test run)
- **Issue:** `bash -c "source aether-utils.sh; _scan_colony_context ..."` triggers the main dispatch (line 1005 `cmd="${1:-help}"`) and outputs the help JSON instead of calling the function
- **Fix:** Created a minimal shim that defines only the dependencies scan.sh needs (json_ok, json_err, error constants) and sources scan.sh directly, bypassing the dispatch
- **Files modified:** tests/bash/test-intelligence.sh
- **Committed in:** 16edca5

**2. [Rule 1 - Bug] Fixed set -e crash with empty chamber directories**
- **Found during:** Task 1 (test 5 failure)
- **Issue:** `set -euo pipefail` in the shim caused early exit when `ls -1d "$chambers_dir"/*/` matched nothing (returns exit code 1), killing the entire function before it reached charter extraction
- **Fix:** Changed shim to `set -uo pipefail` (without `-e`) since scan functions rely on aether-utils.sh error trapping rather than bare set -e
- **Files modified:** tests/bash/test-intelligence.sh
- **Committed in:** 16edca5

---

**Total deviations:** 2 auto-fixed (1 blocking issue, 1 bug)
**Impact on plan:** Necessary for correct test execution. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 3 intelligence sub-scan functions now have comprehensive test coverage
- Phase 32 is complete: Plan 01 (implementation), Plan 02 (prompt enrichment), Plan 03 (testing)

## Self-Check: PASSED

All files and commits verified:
- tests/bash/test-intelligence.sh: FOUND (824 lines, >= 200 min)
- package.json test:intelligence: FOUND
- Commit 16edca5 (Task 1): FOUND
- Commit 3a32d5c (Task 2): FOUND
- 32-03-SUMMARY.md: FOUND

---
*Phase: 32-intelligence-enhancements*
*Completed: 2026-03-27*
