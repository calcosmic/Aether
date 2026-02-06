---
phase: 36-signal-simplification
plan: 04
subsystem: utilities
tags: [pheromones, ttl, gap-closure]

# Dependency graph
requires:
  - phase: 36-03
    provides: decay math removal from .aether/aether-utils.sh
provides:
  - runtime/aether-utils.sh aligned with .aether/aether-utils.sh
  - TTL-based schema validation in runtime utility layer
affects: [37-command-trim]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - TTL expiration instead of exponential decay
    - Priority levels (high/normal/low) instead of numeric strength

key-files:
  created: []
  modified:
    - runtime/aether-utils.sh

key-decisions:
  - "Keep pheromone-validate (content length validation still useful)"

patterns-established:
  - "Signal schema: id, type, content, priority, created_at, expires_at"

# Metrics
duration: 2min
completed: 2026-02-06
---

# Phase 36 Plan 04: Runtime Utility Decay Removal Summary

**Removed exponential decay commands from runtime/aether-utils.sh to match .aether/aether-utils.sh (gap closure)**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-06T17:48:44Z
- **Completed:** 2026-02-06T17:50:06Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments

- Removed pheromone-decay, pheromone-effective, pheromone-batch, pheromone-cleanup commands
- Updated help command to only list pheromone-validate
- Updated validate-state pheromones to use new schema (priority, expires_at)
- Reduced runtime/aether-utils.sh from 373 to 317 lines (-56 lines)

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove decay commands and update help** - `214564d` (feat)

## Files Created/Modified

- `runtime/aether-utils.sh` - Removed decay math, updated help and schema validation

## Decisions Made

- Keep pheromone-validate command - content length validation (min 20 chars) is still useful for ensuring quality signals

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Gap closure plan 36-05 ready (update plan/organize/colonize commands, delete runtime/workers/*.md)
- After 36-05: Phase 37 command trimming

---
*Phase: 36-signal-simplification*
*Completed: 2026-02-06*
