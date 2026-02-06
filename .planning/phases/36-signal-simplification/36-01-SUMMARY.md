---
phase: 36-signal-simplification
plan: 01
subsystem: signals
tags: [pheromones, ttl, priority, expiration]

# Dependency graph
requires:
  - phase: 35-worker-simplification
    provides: Consolidated worker files with keyword-based pheromone guidance
provides:
  - TTL-based signal emission with expires_at + priority schema
  - Duration parsing for --ttl flag (30m, 2h, 1d)
  - Default phase_end expiration for all signals
  - Priority mapping: REDIRECT=high, FOCUS=normal, FEEDBACK=low
affects: [36-02, status.md, build.md, continue.md]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TTL-based signal expiration (expires_at timestamp or 'phase_end')"
    - "Priority levels replace sensitivity matrices (high/normal/low)"

key-files:
  created: []
  modified:
    - commands/ant/focus.md
    - commands/ant/redirect.md
    - commands/ant/feedback.md

key-decisions:
  - "phase_end as default expiration (not wall-clock based)"
  - "Simple priority levels (high/normal/low) replace numeric strength"
  - "Duration parsing: m=minutes, h=hours, d=days"

patterns-established:
  - "Signal schema: {id, type, content, priority, created_at, expires_at, source}"
  - "TTL flag pattern: --ttl <duration> with duration parsing"

# Metrics
duration: 2min
completed: 2026-02-06
---

# Phase 36 Plan 01: Signal Emission TTL Update Summary

**TTL-based signal emission with expires_at + priority schema replacing exponential decay**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-06T16:59:00Z
- **Completed:** 2026-02-06T17:01:16Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Updated focus.md, redirect.md, feedback.md to use TTL-based schema
- Replaced strength/half_life with priority/expires_at
- Added --ttl flag parsing with duration support (30m, 2h, 1d)
- Default expiration set to "phase_end" (phase-scoped, not wall-clock)
- Removed sensitivity matrix displays from all signal commands

## Task Commits

Each task was committed atomically:

1. **Task 1: Update focus.md with TTL-based signal schema** - `a493684` (feat)
2. **Task 2: Update redirect.md and feedback.md with TTL-based signal schema** - `5831f23` (feat)

## Files Created/Modified
- `commands/ant/focus.md` - FOCUS signal emission with priority normal, TTL support
- `commands/ant/redirect.md` - REDIRECT signal emission with priority high, TTL support
- `commands/ant/feedback.md` - FEEDBACK signal emission with priority low, TTL support

## Decisions Made
- Used "phase_end" as default expiration (matches user context requirements)
- Priority mapping per SIMP-03: REDIRECT=high, FOCUS=normal, FEEDBACK=low
- Duration format: simple suffix (m/h/d) for intuitive usage
- Removed all sensitivity matrix displays (no longer meaningful with priority system)

## Deviations from Plan
None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Signal emission commands complete with new schema
- Next plan (36-02) can update signal consumers (status.md, build.md, continue.md)
- Pheromone filtering logic needed in reader commands
- aether-utils.sh pheromone commands can be removed

---
*Phase: 36-signal-simplification*
*Completed: 2026-02-06*
