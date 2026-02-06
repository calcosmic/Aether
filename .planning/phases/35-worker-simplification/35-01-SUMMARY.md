---
phase: 35-worker-simplification
plan: 01
subsystem: workers
tags: [worker-roles, documentation, simplification, pheromones]

# Dependency graph
requires:
  - phase: 34-core-command-rewrite
    provides: build.md and continue.md commands for worker integration
provides:
  - Consolidated workers.md with all 6 role definitions
  - Shared section (activity log, spawn requests, visual identity, output format)
  - Per-role sections with purpose, when-to-use, signals, workflow
affects: [35-02, commands, build.md]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single file consolidation pattern for role definitions"
    - "Keyword-based signal lists instead of sensitivity matrices"

key-files:
  created:
    - ".aether/workers.md"
  modified: []

key-decisions:
  - "Signal keywords instead of sensitivity tables (FOCUS, REDIRECT, etc.)"
  - "Watcher includes extra quality gate content for phase approval"
  - "171 lines final (within 150-250 target)"

patterns-established:
  - "Role section structure: emoji, purpose, when-to-use, signals, workflow"
  - "Shared 'All Workers' section for common functionality"

# Metrics
duration: 2min
completed: 2026-02-06
---

# Phase 35 Plan 01: Worker Consolidation Summary

**Consolidated 6 worker specs (1,866 lines) into single workers.md (171 lines) with role definitions, signal keywords, and shared sections**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-06T13:42:16Z
- **Completed:** 2026-02-06T13:44:01Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Extracted essential content from 6 worker files (architect, builder, colonizer, route-setter, scout, watcher)
- Created consolidated workers.md with 171 lines (91% reduction from 1,866)
- Established shared section for activity log, spawn requests, visual identity, output format
- Each role has purpose, when-to-use, signals, and workflow hints
- Watcher includes quality gate role for phase approval

## Task Commits

Each task was committed atomically:

1. **Task 1-2: Extract and consolidate worker specs** - `80b2ace` (feat)

**Plan metadata:** (pending)

## Files Created/Modified
- `.aether/workers.md` - Consolidated worker role definitions with shared and per-role sections

## Decisions Made
- Signal keywords instead of sensitivity matrices (e.g., "FOCUS, REDIRECT" vs 8-line tables)
- Watcher gets 3 extra lines for quality gate role (mandatory review, approve/block)
- 171 lines final count (within 150-250 target, on lower end for maximum simplicity)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- workers.md ready for command integration in Plan 02
- Old worker files in .aether/workers/ still exist (deletion in Plan 02)
- build.md and continue.md need updates to reference new consolidated file

---
*Phase: 35-worker-simplification*
*Completed: 2026-02-06*
