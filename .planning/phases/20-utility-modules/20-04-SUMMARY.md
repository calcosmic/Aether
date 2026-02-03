---
phase: 20-utility-modules
plan: 04
subsystem: infra
tags: [bash, jq, error-tracking, pattern-detection, deduplication]

# Dependency graph
requires:
  - phase: 20-utility-modules plan 03
    provides: memory subcommands, aether-utils.sh scaffold at 196 lines
provides:
  - error-add subcommand for deterministic error recording with auto-ID and retention
  - error-pattern-check subcommand for category-based pattern flagging
  - error-summary subcommand for category/severity aggregation
  - error-dedup subcommand for duplicate removal within time window
  - Complete aether-utils.sh with all 18 subcommands under 300 lines
affects: [21-integration]

# Tech tracking
tech-stack:
  added: []
  patterns: [jq-group-by-aggregation, timestamp-based-dedup, from-entries-pattern]

key-files:
  created: []
  modified: [.aether/aether-utils.sh]

key-decisions:
  - "error-add accepts ANY string as category (no validation against 12 known categories)"
  - "error-dedup groups by category+description, keeps earliest, drops others within 60s"
  - "error-summary uses jq group_by + from_entries for compact category/severity aggregation"
  - "jq from_entries requires {key, value} pairs not {key, count} (fixed during implementation)"

patterns-established:
  - "jq group_by + map + from_entries for pivot-table-style aggregation"
  - "Timestamp-based dedup: group, sort by time, keep first, filter rest by time delta"

# Metrics
duration: 2min
completed: 2026-02-03
---

# Phase 20 Plan 04: Error Tracking Summary

**4 error tracking subcommands completing all 18 utility commands: auto-ID error recording with 50-entry retention, 3+ occurrence pattern flagging, category/severity aggregation, and 60-second window deduplication -- all in 241 lines total**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-03T17:18:26Z
- **Completed:** 2026-02-03T17:20:36Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Added error-add: appends error with auto-generated ID (err_epoch_hex), enforces 50-error retention limit
- Added error-pattern-check: groups errors by category, flags categories with 3+ occurrences with first/last seen timestamps
- Added error-summary: outputs total count and breakdowns by category and severity using jq from_entries
- Added error-dedup: removes duplicate errors (same category+description within 60 seconds), keeps earliest
- Updated help text to list all 15 commands (help, version, 13 operational subcommands)
- Verified all 18 subcommands end-to-end: 20 success-path tests (exit 0, valid JSON) and 3 error-path tests (exit non-zero)
- Final line count: 241 lines (59 lines under 300 budget)
- Phase 20 is now complete: all 4 plans delivered all 18 subcommands

## Task Commits

Each task was committed atomically:

1. **Task 1: Add 4 error tracking subcommands** - `afec07c` (feat)
2. **Task 2: Full verification of all 18 subcommands** - `22fd9c9` (test)

## Files Created/Modified
- `.aether/aether-utils.sh` - Added 4 error case branches + updated help text (~45 lines added, total now 241 lines)

## Decisions Made
- error-add accepts any string as category (does not validate against 12 known categories) -- matches success criteria using "build"
- error-summary uses `{key, value}` format for jq `from_entries` (fixed from initial `{key, count}` which produced nulls)
- error-dedup compares timestamps relative to group's first entry, not pairwise between consecutive entries
- Test errors cleaned from errors.json after verification to leave clean state

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed from_entries format in error-summary**
- **Found during:** Task 1 verification
- **Issue:** jq `from_entries` requires objects with `key` and `value` fields, but initial implementation used `{key, count}` producing null values
- **Fix:** Changed `count: length` to `value: length` in the map expression
- **Files modified:** .aether/aether-utils.sh
- **Commit:** afec07c (included in Task 1 commit)

## Issues Encountered
None beyond the from_entries format fix noted above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 20 complete: all 4 plans delivered, all 18 subcommands verified
- aether-utils.sh at 241 lines (under 300 budget)
- Ready for Phase 21 (Integration) which connects utility subcommands to command prompts

---
*Phase: 20-utility-modules*
*Completed: 2026-02-03*
