---
phase: 18-reliability-architecture-gaps
plan: 02
subsystem: testing
tags: [shell, error-handling, json, spawn, model-routing]

# Dependency graph
requires:
  - phase: 18-01
    provides: startup ordering and composed EXIT trap already in place
provides:
  - model-get/model-list subprocess error handling with E_BASH_ERROR and Try: suggestions
  - spawn-complete failure event logging to COLONY_STATE.json events array
  - Two regression tests: no-exec pattern and Try: suggestion in model errors

affects: [model-routing, spawn-complete, aether-utils]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Subprocess-not-exec pattern: set +e; result=$(bash); exit_code=$?; set -e for capturable error handling"
    - "Phase 17 friendly error style: Couldn't [do thing]. Try: [action]."
    - "spawn_failed event type in COLONY_STATE.json events array for audit trail"

key-files:
  created: []
  modified:
    - .aether/aether-utils.sh
    - tests/bash/test-aether-utils.sh

key-decisions:
  - "ARCH-07: model-get/model-list use subprocess (not exec) so exit code and stderr can be captured before reporting friendly error"
  - "ARCH-04: spawn-complete logs spawn_failed events to COLONY_STATE.json events array — only when status is 'failed' or 'error'; independent tasks are not blocked"
  - "Shell scoping: local keyword not valid in case statements — use prefixed variable names (spawn_complete_state_file, spawn_complete_updated) to avoid collisions"

patterns-established:
  - "Subprocess error capture pattern: set +e; result=$(cmd 2>&1); exit_code=$?; set -e — standard wrapper for delegating commands that may fail"
  - "Event logging in case blocks: use prefixed variable names instead of local to avoid SC2168 shellcheck errors"

requirements-completed:
  - ARCH-07
  - ARCH-04

# Metrics
duration: 15min
completed: 2026-02-19
---

# Phase 18 Plan 02: Model Error Handling and Spawn Failure Logging Summary

**model-get/model-list replaced exec with subprocess calls for capturable error reporting, and spawn-complete now logs failed spawns to COLONY_STATE.json events for audit trail**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-02-19T15:05:00Z
- **Completed:** 2026-02-19T16:13:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- model-get and model-list now use subprocess calls (not exec) so exit codes are captured and errors produce friendly JSON with E_BASH_ERROR code and "Try:" suggestions
- spawn-complete logs spawn_failed events to COLONY_STATE.json events array whenever status is "failed" or "error" — provides audit trail for debugging failed worker completions
- Two new regression tests: `test_model_get_no_exec_pattern` (confirms exec pattern removed) and `test_model_get_error_has_try_suggestion` (confirms friendly error format)
- 31 bash tests total, 0 failures

## Task Commits

Each task was committed atomically:

1. **Task 1: Replace exec with subprocess error handling in model-get and model-list** - `291ccab` (feat)
2. **Task 2: Add spawn failure event logging and tests** - `ef3f1f6` (feat, accumulated with 18-03 stash work)

**Plan metadata:** See final commit in this session

## Files Created/Modified

- `.aether/aether-utils.sh` - model-get/model-list subprocess pattern; spawn-complete failure event logging
- `tests/bash/test-aether-utils.sh` - Two new tests: test_model_get_no_exec_pattern, test_model_get_error_has_try_suggestion

## Decisions Made

- ARCH-07: exec replaced with subprocess pattern: `set +e; result=$(bash "$0" model-profile ...); exit_code=$?; set -e` — allows stderr capture and friendly error emission
- ARCH-04: spawn_failed events logged to COLONY_STATE.json events array on "failed" or "error" status. Only dependent tasks are blocked — independent parallel tasks continue (fail-fast per user decision)
- Shell scoping: `local` keyword not valid in case statements (SC2168). Used prefixed variable names (`spawn_complete_state_file`, `spawn_complete_updated`) instead

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] shellcheck SC2168 — local keyword not valid outside function**
- **Found during:** Task 2 (spawn failure event logging)
- **Issue:** Plan code sample used `local state_file` and `local updated` inside a case statement, which is not valid in bash (SC2168 error)
- **Fix:** Replaced `local` variables with prefixed names `spawn_complete_state_file` and `spawn_complete_updated` to avoid name collisions without using local scope
- **Files modified:** .aether/aether-utils.sh
- **Verification:** shellcheck --severity=error passes with no errors
- **Committed in:** ef3f1f6 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - bug in plan code sample)
**Impact on plan:** Minor fix required — plan code sample used `local` in a case block which shellcheck rejects. Functionally equivalent fix using prefixed variable names.

## Issues Encountered

- During Task 1 commit, the pre-commit sync hook ran twice (second run saw empty staging area) — commit succeeded on first hook execution as confirmed by git log. Benign hook quirk.
- Task 2 changes (spawn_failed logic + tests) were committed as part of `ef3f1f6` due to stash pop accumulation from a prior session's 18-03 work — confirmed all required changes are present in HEAD

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- ARCH-07 and ARCH-04 both complete
- model-get/model-list errors are now user-friendly and auditable
- spawn failures are recorded for debugging — ready for any remaining Phase 18 plans

---
*Phase: 18-reliability-architecture-gaps*
*Completed: 2026-02-19*
