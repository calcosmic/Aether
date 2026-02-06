---
phase: 36-signal-simplification
plan: 03
subsystem: infra
tags: [pheromones, ttl, utility-layer, documentation]

# Dependency graph
requires:
  - phase: 36-01
    provides: TTL-based signal schema with expires_at and priority
provides:
  - Simplified aether-utils.sh without decay math
  - Updated pheromones.md documentation for TTL system
affects: [signal-commands, worker-specs]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TTL-based signal expiration (filter on read, no cleanup)"
    - "Priority levels instead of sensitivity math"

key-files:
  created: []
  modified:
    - ".aether/aether-utils.sh"
    - ".aether/docs/pheromones.md"

key-decisions:
  - "Keep pheromone-validate for content length validation"
  - "Filter expired signals on read (no cleanup command needed)"
  - "Priority processing order: high, normal, low"

patterns-established:
  - "Signal validation checks expires_at and priority fields"
  - "Documentation uses TTL/priority terminology exclusively"

# Metrics
duration: 2min
completed: 2026-02-06
---

# Phase 36 Plan 03: Decay Code Removal Summary

**Removed 56 lines of exponential decay math from aether-utils.sh, updated pheromones.md to document TTL-based system**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-06T17:02:44Z
- **Completed:** 2026-02-06T17:05:30Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Removed 4 decay commands (pheromone-decay, pheromone-effective, pheromone-batch, pheromone-cleanup)
- Updated validate-state pheromones to check expires_at and priority fields
- Rewrote pheromones.md documentation for TTL-based system
- Added Pause-Aware TTL section to docs

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove pheromone decay commands from aether-utils.sh** - `1407d55` (refactor)
2. **Task 2: Update pheromones.md documentation** - `e11d237` (docs)

## Files Created/Modified
- `.aether/aether-utils.sh` - Removed decay/effective/batch/cleanup commands, updated validate-state schema
- `.aether/docs/pheromones.md` - Complete rewrite for TTL-based system

## Decisions Made
- **Keep pheromone-validate:** Content length validation (min 20 chars) still useful
- **No cleanup command needed:** Expired signals filtered on read in commands
- **Priority processing:** Workers check high priority first, then normal, then low

## Deviations from Plan
None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Signal simplification complete (SIMP-03 done)
- aether-utils.sh reduced from 373 to 317 lines (-56 lines, 15% reduction)
- Ready for final consolidation and verification in next phase

---
*Phase: 36-signal-simplification*
*Completed: 2026-02-06*
