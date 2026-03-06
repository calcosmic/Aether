---
phase: 02-learnings-injection
plan: 01
subsystem: workflow
tags: [phase-learnings, colony-prime, jq, aether-utils, builder-prompts]

# Dependency graph
requires:
  - phase: 01-instinct-pipeline
    provides: "colony-prime prompt_section assembly pattern, instinct formatting via pheromone-prime"
provides:
  - "Phase learnings extraction and formatting in colony-prime prompt_section"
  - "Validated learnings from previous phases visible to builders"
  - "Learning count in colony-prime log_line"
  - "Compact mode cap (5 claims) and non-compact cap (15 claims)"
affects: [02-02, colony-prime, build]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "jq group_by for phase-based learning clustering in colony-prime"
    - "unique_by(.claim) for deduplication of inherited vs phase learnings"
    - "Type-based select for mixed string/numeric phase values"

key-files:
  created: []
  modified:
    - ".aether/aether-utils.sh"

key-decisions:
  - "Learnings placed between context-capsule and pheromone signals in prompt assembly order"
  - "Inherited learnings sorted first (before numeric phases) for foundational visibility"
  - "Compact mode: 5 claims max; non-compact: 15 claims max"
  - "No changes to build-context.md or build-wave.md (confirmed again, same as Phase 1)"

patterns-established:
  - "Phase learnings display: phase header followed by indented bullet claims"
  - "Conditional section pattern: only append when count > 0 (no empty headers)"

requirements-completed: [LEARN-01, LEARN-04]

# Metrics
duration: 3min
completed: 2026-03-06
---

# Phase 2 Plan 1: Learnings Injection Summary

**Validated phase learnings extracted from COLONY_STATE.json and formatted as actionable guidance in colony-prime prompt_section, grouped by phase with inherited-first ordering**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-06T21:34:48Z
- **Completed:** 2026-03-06T21:38:13Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Wired phase learnings read-side into colony-prime (closes the data flow gap where continue writes learnings but builders never see them)
- Validated claims from previous phases now appear as formatted text between context-capsule and pheromone signals
- Inherited learnings (phase="inherited") handled correctly via jq type checking
- Compact mode caps at 5 claims, non-compact at 15
- Empty/missing phase_learnings produce no section (no empty headers)
- Log line updated to include learning count (e.g., "Primed: 3 signals, 2 instincts, 4 learnings")

## Task Commits

Each task was committed atomically:

1. **Task 1: Add phase learnings extraction and formatting to colony-prime** - `daa38cd` (feat)

## Files Created/Modified
- `.aether/aether-utils.sh` - Added 61-line phase learnings extraction and formatting block to colony-prime subcommand (lines 7633-7692)

## Decisions Made
- Learnings positioned between context-capsule and pheromone signals in prompt assembly (matches information hierarchy: historical context before current guidance)
- Used simpler jq approach (flatten all validated claims, cap, then group for display) rather than complex per-group cap enforcement
- Added unique_by(.claim) deduplication to handle potential overlap between inherited and per-phase learnings
- No new subcommands created; inline extraction follows the same pattern as instinct-read in pheromone-prime

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase learnings now flow from continue (write-side) through colony-prime (read-side) to builder prompts
- Ready for Plan 02 (integration tests for the learnings injection pipeline)
- Full learning pipeline working: continue extracts learnings -> COLONY_STATE.json stores them -> colony-prime reads and formats -> builders see validated insights

## Self-Check: PASSED

All files verified:
- .aether/aether-utils.sh: FOUND (modified with phase learnings block)
- Commit daa38cd: FOUND

---
*Phase: 02-learnings-injection*
*Completed: 2026-03-06*
