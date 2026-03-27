---
phase: 30-charter-management
plan: 02
subsystem: testing
tags: [bash, integration-tests, queen-md, charter, colony-name]

# Dependency graph
requires:
  - phase: 30-01
    provides: "_colony_name() and _queen_write_charter() functions in queen.sh"
provides:
  - Integration test suite proving CHARTER-01, CHARTER-02, CHARTER-03 requirements
  - 12 tests covering colony-name fallback chain, charter-write init/re-init safety, no new headers, METADATA accuracy
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Isolated temp dir test environment with full aether-utils.sh + utils + exchange copies"
    - "Section content extraction via sed range patterns (macOS-compatible, no head -n -1)"
    - "Error JSON on stderr requires capturing 2>&1 for assertion tests"

key-files:
  created:
    - tests/bash/test-queen-charter.test.sh
  modified: []

key-decisions:
  - "xml-utils.sh requires exchange directory to be present -- test setup must copy both utils/ and exchange/"
  - "macOS head -n -1 not portable -- use sed '$d' for stripping last line from range extraction"
  - "json_err writes to stderr -- error path tests must capture stderr (2>&1) not just stdout"

patterns-established:
  - "Charter test setup copies utils/ + exchange/ to avoid xml-utils.sh cd failure"

requirements-completed: [CHARTER-01, CHARTER-02, CHARTER-03]

# Metrics
duration: 9min
completed: 2026-03-27
---

# Phase 30 Plan 02: Charter Tests Summary

**12 integration tests proving charter-write writes to correct QUEEN.md sections, re-init preserves non-charter content, no new headers created, and colony-name fallback chain works**

## Performance

- **Duration:** 9 min
- **Started:** 2026-03-27T16:41:18Z
- **Completed:** 2026-03-27T16:50:48Z
- **Tasks:** 2
- **Files created:** 1

## Accomplishments
- 12 integration tests for charter-write and colony-name subcommands
- CHARTER-01 verified: first init writes Intent/Vision to User Preferences, Governance/Goals to Codebase Patterns
- CHARTER-02 verified: charter entries appear in correct QUEEN.md sections
- CHARTER-03 verified: re-init safety -- non-charter content (wisdom, learnings, instincts) preserved on re-write
- No new ## headers test proves charter-write never creates section headers
- Colony-name fallback chain verified: COLONY_STATE.json -> package.json -> directory basename
- METADATA stats accuracy verified after first write and re-init (no drift)
- Content truncation at 200 chars with "..." suffix verified
- Error handling for missing QUEEN.md verified (E_FILE_NOT_FOUND on stderr)

## Task Commits

Each task was committed atomically:

1. **Task 1: Write integration tests for charter management** - `e044dfb` (test)

Task 2 (full suite verification) required no code changes -- only confirmed existing tests pass.

## Files Created/Modified
- `tests/bash/test-queen-charter.test.sh` - 12 integration tests for charter-write and colony-name

## Decisions Made
- xml-utils.sh sources at startup and tries to cd into exchange/ directory -- test setup must copy both utils/ and exchange/ to temp dirs to avoid E_BASH_ERROR
- macOS `head -n -1` is not portable -- used `sed '$d'` for stripping last line from sed range extraction
- `json_err` in queen.sh writes error JSON to stderr -- error path tests must capture stderr with `2>&1`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] macOS head -n -1 not portable**
- **Found during:** Task 1 (test_charter_write_first_init)
- **Issue:** Plan used `head -n -1` to strip the trailing `## ` header from sed range output, but macOS head does not support negative line counts
- **Fix:** Replaced with `sed '$d'` which works on both macOS and Linux
- **Files modified:** tests/bash/test-queen-charter.test.sh
- **Verification:** Test 5 passes with correct section content extraction

**2. [Rule 3 - Blocking] Missing exchange/ directory in test 1 setup**
- **Found during:** Task 1 (test_colony_name_from_directory)
- **Issue:** xml-utils.sh (sourced by aether-utils.sh) attempts `cd "$SCRIPT_DIR/../exchange"` at startup. Test 1 only copied utils/ but not exchange/, causing E_BASH_ERROR and silent failure with set -euo pipefail
- **Fix:** Added exchange/ directory copy to test 1 setup, matching the pattern from setupCharterTest() helper and test-scan-module.sh
- **Files modified:** tests/bash/test-queen-charter.test.sh
- **Verification:** Test 1 now passes with correct directory basename derivation

**3. [Rule 3 - Blocking] json_err output goes to stderr, not stdout**
- **Found during:** Task 1 (test_charter_write_queen_missing)
- **Issue:** run_charter_write helper suppresses stderr with `2>/dev/null`, but json_err writes error JSON to stderr. The error test was capturing empty output
- **Fix:** Changed test 12 to capture both stdout and stderr with `2>&1` instead of using the helper
- **Files modified:** tests/bash/test-queen-charter.test.sh
- **Verification:** Test 12 now correctly asserts `ok: false` and `error.code: E_FILE_NOT_FOUND`

---

**Total deviations:** 3 auto-fixed (1 bug, 2 blocking)
**Impact on plan:** All fixes were test infrastructure issues, not logic changes. No scope creep.

## Issues Encountered
- Pre-existing spawn-tree.test.js timeout failure (ETIMEDOUT on spawn-tree-load) -- out of scope, deferred. Not related to charter changes.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 12 charter management tests pass
- CHARTER-01, CHARTER-02, CHARTER-03 requirements verified through integration tests
- Phase 31 (smart-init rewrite) can confidently use charter-write and colony-name subcommands
- 615 existing tests pass with zero regressions (1 pre-existing timeout in spawn-tree.test.js)

---
*Phase: 30-charter-management*
*Completed: 2026-03-27*

## Self-Check: PASSED

All deliverables verified:
- Commit: e044dfb exists
- File: tests/bash/test-queen-charter.test.sh created (731 lines)
- Tests: 12/12 charter tests pass
- Regression: 615 existing tests pass (1 pre-existing failure in spawn-tree.test.js unrelated to changes)
- Requirements: CHARTER-01, CHARTER-02, CHARTER-03 all verified through passing tests
