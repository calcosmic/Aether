---
phase: 11-dead-code-deprecation
plan: 01
subsystem: utilities
tags: [deprecation, aether-utils, dead-code, maintenance]

# Dependency graph
requires:
  - phase: 11-dead-code-deprecation
    provides: "11-RESEARCH.md audit identifying 18 confirmed-dead subcommands"
provides:
  - "_deprecation_warning function in aether-utils.sh"
  - "18 deprecated subcommand calls emitting stderr warnings"
  - "Deprecated section in help JSON with all 18 entries"
  - "[DEPRECATED] markers on Skills Engine section entries"
affects: [11-02-PLAN, dead-code-removal]

# Tech tracking
tech-stack:
  added: []
  patterns: ["_deprecation_warning stderr helper for one-cycle deprecation confirmation"]

key-files:
  created: []
  modified:
    - ".aether/aether-utils.sh"

key-decisions:
  - "Deprecation warning goes to stderr only (printf >&2) to avoid breaking JSON stdout contracts"
  - "Warning format '[deprecated] name -- will be removed in v3.0' chosen for grep-ability and clarity"
  - "Function placed after error constants block, before fallback atomic_write"

patterns-established:
  - "_deprecation_warning pattern: insert call as first line of case handler, before any logic"

requirements-completed: [QUAL-01, QUAL-02, QUAL-03]

# Metrics
duration: 3min
completed: 2026-03-24
---

# Phase 11 Plan 01: Dead Code Deprecation Summary

**Deprecation warnings added to 18 confirmed-dead subcommands in aether-utils.sh with help JSON markers and a Deprecated section**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-24T05:07:29Z
- **Completed:** 2026-03-24T05:10:42Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Added `_deprecation_warning` helper function that emits `[deprecated] <name> -- will be removed in v3.0` to stderr
- Inserted deprecation calls as the first line of all 18 confirmed-dead subcommand case handlers
- Updated help JSON: 3 Skills Engine entries tagged `[DEPRECATED]`, new Deprecated section with all 18 entries
- All deprecated subcommands continue to execute normally (no behavior change)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add _deprecation_warning function and deprecation calls** - `d1f106a` (feat)
2. **Task 2: Update help JSON with deprecation markers** - `7bf5579` (feat)

## Files Created/Modified
- `.aether/aether-utils.sh` - Added _deprecation_warning function (6 lines), 18 deprecation calls, 3 [DEPRECATED] description updates, new 18-entry Deprecated section in help JSON

## Decisions Made
- Deprecation warning uses `printf >&2` (not echo) for portability and to avoid interfering with JSON stdout
- Warning placed as absolute first line of each handler to ensure it fires regardless of early-exit paths
- Help JSON: only 3 commands (skill-index-read, skill-manifest-read, skill-is-user-created) appear in named sections; the other 15 appear only in the flat commands array and the new Deprecated section

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Audit Record (QUAL-01)
The audit satisfying QUAL-01 criterion 3 (alive subcommands documented) is recorded in `11-RESEARCH.md` in this phase directory. That research identified the 18 confirmed-dead subcommands targeted by this plan and documented the remaining ~107 alive subcommands.

## Next Phase Readiness
- All 18 deprecated subcommands are now emitting warnings to stderr
- Plan 02 can proceed with whatever the next step in the deprecation lifecycle is
- One-cycle confirmation period has begun: any caller of these subcommands will see the warning

## Self-Check: PASSED

- [x] 11-01-SUMMARY.md exists
- [x] Commit d1f106a (Task 1) found
- [x] Commit 7bf5579 (Task 2) found

---
*Phase: 11-dead-code-deprecation*
*Completed: 2026-03-24*
