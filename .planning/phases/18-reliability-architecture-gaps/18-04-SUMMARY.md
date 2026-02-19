---
phase: 18-reliability-architecture-gaps
plan: 04
subsystem: error-handling
tags: [bash, json-validation, schema-migration, queen-read, validate-state]

requires:
  - phase: 18-01
    provides: startup ordering fix (ARCH-09) and composed exit trap (ARCH-10) that are prerequisites for reliability work

provides:
  - queen-read JSON validation Gate 1 (metadata before --argjson)
  - queen-read JSON validation Gate 2 (assembled result before json_ok)
  - validate-state colony schema migration (_migrate_colony_state)
  - W_MIGRATED notification on pre-3.0 state file migration
  - Corrupt COLONY_STATE.json backup before E_JSON_INVALID error
  - known-issues.md fully updated with Phase 18 fix attributions
  - 2 new structural regression tests (ARCH-06, ARCH-02)

affects: [any callers of queen-read, any colony initialization that may use pre-3.0 state files]

tech-stack:
  added: []
  patterns:
    - "Double-gate validation: validate input before use (Gate 1) and validate output before return (Gate 2)"
    - "Additive schema migration: never remove/rename fields, only add missing ones with empty defaults"
    - "Auto-migrate + notify: silent migration with W_MIGRATED stderr warning for observability"
    - "Corrupt file backup pattern: back up before error to enable recovery"

key-files:
  created: []
  modified:
    - .aether/aether-utils.sh
    - .aether/docs/known-issues.md
    - tests/bash/test-aether-utils.sh

key-decisions:
  - "Do NOT auto-reset QUEEN.md on malformed metadata — QUEEN.md contains user-accumulated wisdom; emit actionable error with Try: suggestion instead"
  - "Migration is additive only (never removes fields) — idempotent and safe for concurrent access (Pitfall 5)"
  - "W_MIGRATED goes to stderr not stdout — callers only see final validation result on stdout"
  - "Test fixture updated to v3.0 format to avoid migration side effects in existing test"

requirements-completed:
  - ARCH-06
  - ARCH-02

duration: 7min
completed: 2026-02-19
---

# Phase 18 Plan 04: Queen-Read JSON Validation and State Schema Migration Summary

**queen-read gets two validation gates (malformed METADATA and invalid assembled output both caught with actionable E_JSON_INVALID), validate-state colony auto-migrates pre-3.0 state files to v3.0 with W_MIGRATED notification and backup-before-error for corrupt files**

## Performance

- **Duration:** ~7 min
- **Started:** 2026-02-19T15:07:45Z
- **Completed:** 2026-02-19T15:14:21Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- queen-read can no longer trigger an ERR trap on malformed METADATA — Gate 1 catches bad JSON before `--argjson` use, Gate 2 catches invalid assembled result before `json_ok`
- validate-state colony auto-migrates pre-3.0 state files to v3.0 by adding missing fields (`signals`, `graveyards`, `events`) — additive only, idempotent, safe
- Corrupt COLONY_STATE.json (unparseable JSON) triggers backup to `.aether/data/backups/` then a clean E_JSON_INVALID error with recovery suggestion
- known-issues.md fully updated: GAP-001/002/003/004/005/006 and ISSUE-002/003/007 all marked FIXED with Phase 18 attribution
- 31 bash tests, 0 failures (28 pre-existing + 3 from 18-02/03 + 2 new from 18-04)

## Task Commits

1. **Task 1: Add JSON validation gates to queen-read** - `b722424` (feat)
2. **Task 2: Schema migration, known-issues, and tests** - `e1c9f6f` (feat)

## Files Created/Modified

- `.aether/aether-utils.sh` - Two validation gates in queen-read; `_migrate_colony_state` function and call in validate-state colony
- `.aether/docs/known-issues.md` - GAP-001/002/003/004/005/006 and ISSUE-002/003/007 marked FIXED with Phase 18 attribution
- `tests/bash/test-aether-utils.sh` - 2 new structural tests (ARCH-06, ARCH-02); test fixture updated to v3.0 format

## Decisions Made

- Do not auto-reset QUEEN.md on malformed metadata — QUEEN.md contains user-accumulated wisdom (philosophies, patterns, decrees). Report error with actionable "Try:" suggestion; user decides whether to fix JSON manually or run queen-init to reset.
- Migration is additive only — never removes or renames fields, only adds missing ones with empty defaults (`[]`). This is idempotent and safe for concurrent access.
- W_MIGRATED notification goes to stderr, not stdout. Callers receive only the final validation result on stdout; the migration warning is visible in debug/logging contexts but doesn't break JSON pipelines.
- Existing test fixture updated to include `"version": "3.0"` so it doesn't trigger migration side effects that would corrupt the test's output assertions.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test fixture caused migration side effect breaking test assertion**
- **Found during:** Task 2 (after running test suite)
- **Issue:** Existing `test_validate_state_colony` fixture used a COLONY_STATE.json without a `version` field. Migration ran and emitted W_MIGRATED to stderr. Test captured `2>&1`, so two JSON lines appeared where one was expected. `assert_ok_true` failed because `jq` was given multi-line input.
- **Fix:** Updated fixture to include `"version": "3.0"` with all v3.0 fields (`signals`, `graveyards`) so migration is not triggered during the test.
- **Files modified:** `tests/bash/test-aether-utils.sh`
- **Verification:** Test 3 passes after fixture update; 0 test failures
- **Committed in:** `e1c9f6f` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - Bug)
**Impact on plan:** Fix was necessary to avoid breaking an existing test. No scope creep.

## Issues Encountered

None beyond the auto-fixed deviation above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 18 core reliability work is complete: ARCH-01 through ARCH-10 all addressed across 18-01 through 18-04
- All documented architecture gaps (GAP-001 through GAP-006) and issues (ISSUE-002, ISSUE-003, ISSUE-007) are marked FIXED
- 31 bash tests passing, 0 failures
- Pre-existing unit test failures (2) in validate-state.test.js are unrelated to Phase 18 work (TypeError: error.error.includes is not a function — error.error is object not string)
- Ready to finalize Phase 18 and publish

## Self-Check: PASSED

- FOUND: .aether/aether-utils.sh
- FOUND: .aether/docs/known-issues.md
- FOUND: tests/bash/test-aether-utils.sh
- FOUND: 18-04-SUMMARY.md
- FOUND: commit b722424 (feat: queen-read JSON validation gates)
- FOUND: commit e1c9f6f (feat: schema migration, known-issues, tests)
- FOUND: Gate 1 (malformed METADATA) in aether-utils.sh
- FOUND: Gate 2 (assemble queen-read) in aether-utils.sh
- FOUND: _migrate_colony_state function in aether-utils.sh
- FOUND: W_MIGRATED in aether-utils.sh

---
*Phase: 18-reliability-architecture-gaps*
*Completed: 2026-02-19*
