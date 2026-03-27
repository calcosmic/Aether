---
phase: 11-dead-code-deprecation
plan: 02
subsystem: testing
tags: [deprecation, test-updates, stderr, assertions, backward-compat]

# Dependency graph
requires:
  - phase: 11-dead-code-deprecation
    provides: "11-01 deprecation warnings on 18 subcommands emitting [deprecated] to stderr"
provides:
  - "All test files updated to handle stderr deprecation warnings"
  - "Deprecation warning assertion tests for 7 deprecated subcommands across 6 files"
affects: [dead-code-removal]

# Tech tracking
tech-stack:
  added: []
  patterns: ["2>/dev/null for stderr suppression when capturing JSON stdout from deprecated subcommands", "spawnSync for separate stderr capture in Node.js tests"]

key-files:
  created: []
  modified:
    - "tests/bash/test-aether-utils.sh"
    - "tests/bash/test-skills.sh"
    - "tests/bash/test-session-freshness.sh"
    - "tests/e2e/test-adv.sh"
    - "tests/e2e/test-vis.sh"
    - "tests/integration/suggest-pheromones.test.js"

key-decisions:
  - "Use 2>/dev/null (not 2>&1) for tests that parse JSON stdout from deprecated subcommands"
  - "Use 2>&1 >/dev/null pattern to capture only stderr for deprecation warning assertions"
  - "Use spawnSync in Node.js test for separate stderr capture (execSync merges stdio)"

patterns-established:
  - "Deprecation warning test pattern: capture stderr only via `2>&1 >/dev/null`, assert contains [deprecated]"

requirements-completed: [QUAL-02]

# Metrics
duration: 6min
completed: 2026-03-24
---

# Phase 11 Plan 02: Test Updates for Deprecation Warnings Summary

**Updated 6 test files to handle stderr deprecation warnings from 18 deprecated subcommands, adding explicit deprecation assertions for 7 subcommands**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-24T05:12:45Z
- **Completed:** 2026-03-24T05:19:18Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Fixed 3 bash test files where `2>&1` was mixing deprecation stderr into JSON output (test-skills.sh, test-aether-utils.sh, test-session-freshness.sh)
- Added 10 deprecation warning assertion tests across all 6 files verifying `[deprecated]` appears on stderr
- All 6 updated test files pass with zero regressions from our changes

## Task Commits

Each task was committed atomically:

1. **Task 1: Update bash/e2e test files for deprecated subcommand assertions** - `85c8411` (fix)
2. **Task 2: Add suggest-clear deprecation warning assertion** - `d76f738` (test)

## Files Created/Modified
- `tests/bash/test-aether-utils.sh` - Redirected error-summary stderr, added deprecation warning test
- `tests/bash/test-skills.sh` - Redirected skill-index-read/skill-manifest-read/skill-is-user-created stderr, added 3 deprecation warning tests
- `tests/bash/test-session-freshness.sh` - Redirected survey-verify-fresh/survey-clear stderr, added 2 deprecation warning tests
- `tests/e2e/test-adv.sh` - Added ADV-DEPR test invoking swarm-display-inline to verify deprecation warning
- `tests/e2e/test-vis.sh` - Added VIS-DEPR test invoking swarm-display-inline to verify deprecation warning
- `tests/integration/suggest-pheromones.test.js` - Added suggest-clear deprecation warning test using spawnSync

## Decisions Made
- Used `2>/dev/null` instead of `2>&1` for tests that parse JSON stdout from deprecated subcommands (prevents deprecation warning from corrupting JSON parsing)
- Used `2>&1 >/dev/null` idiom for deprecation assertion tests to capture stderr-only content
- Used `spawnSync` in the Node.js test (instead of `execSync`) to get separate access to stderr

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- QUAL-02 complete: all deprecation warnings are fully integrated without breaking the test suite
- Phase 11 (Dead Code Deprecation) is now complete - both plans finished
- One-cycle deprecation confirmation period is active: any callers of the 18 deprecated subcommands will see warnings

## Self-Check: PASSED

- [x] 11-02-SUMMARY.md exists
- [x] Commit 85c8411 (Task 1) found
- [x] Commit d76f738 (Task 2) found
- [x] All 6 modified test files exist on disk

---
*Phase: 11-dead-code-deprecation*
*Completed: 2026-03-24*
