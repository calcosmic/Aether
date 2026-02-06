---
phase: 36-signal-simplification
plan: 05
subsystem: commands
tags: [ttl, signals, pheromones, commands, cleanup]

# Dependency graph
requires:
  - phase: 36-01
    provides: TTL-based signal schema with expires_at and priority
  - phase: 36-02
    provides: TTL filtering patterns for signal consumers
provides:
  - TTL-based signal filtering in plan.md, organize.md, colonize.md
  - Removal of obsolete runtime/workers/ directory (1,866 lines deleted)
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Direct pheromones.json read with TTL filtering"
    - "Priority-based signal display (high/normal/low)"

key-files:
  created: []
  modified:
    - commands/ant/plan.md
    - commands/ant/organize.md
    - commands/ant/colonize.md

key-decisions:
  - "Use inline TTL filtering instead of pheromone-batch utility call"
  - "Display signals with priority and expiration time instead of strength bar"

patterns-established:
  - "Signal filtering pattern: check expires_at for phase_end or timestamp comparison"

# Metrics
duration: 2min
completed: 2026-02-06
---

# Phase 36 Plan 05: Gap Closure (Commands & Workers) Summary

**Updated 3 commands to use TTL-based signal filtering and deleted 1,866 lines of obsolete worker specs**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-06T17:48:48Z
- **Completed:** 2026-02-06T17:50:28Z
- **Tasks:** 2
- **Files modified:** 3 commands, 6 files deleted

## Accomplishments

- Updated plan.md, organize.md, colonize.md to use TTL filtering
- Replaced pheromone-batch calls with direct pheromones.json reads
- Deleted 6 obsolete runtime/workers/*.md files (1,866 lines)
- Removed empty runtime/workers/ directory

## Task Commits

Each task was committed atomically:

1. **Task 1: Update commands to TTL filtering** - `ba6250e` (feat)
2. **Task 2: Delete obsolete worker files** - `ccdc0a2` (chore)

## Files Created/Modified

- `commands/ant/plan.md` - TTL filtering in Step 3
- `commands/ant/organize.md` - TTL filtering in Step 2
- `commands/ant/colonize.md` - TTL filtering in Step 2
- `runtime/workers/*.md` - 6 files DELETED (builder, watcher, scout, colonizer, architect, route-setter)
- `runtime/workers/` - Directory DELETED

## Decisions Made

- Used inline TTL filtering instead of calling pheromone-batch utility
- Retained terminology consistency: "signals" replaces "pheromones" in display text
- Deleted entire runtime/workers/ directory since .aether/workers.md is the canonical source

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All Phase 36 verification gaps now closed
- Commands use TTL filtering (no pheromone-batch)
- Obsolete worker files deleted (1,866 lines removed)
- Ready for Phase 37 (final simplification) or verification

---
*Phase: 36-signal-simplification*
*Completed: 2026-02-06*
