---
phase: 40-state-utility-alignment
plan: 01
subsystem: commands
tags: [state, signals, TTL, jq, colony-state]

# Dependency graph
requires:
  - phase: 36-signal-simplification
    provides: TTL-based signal expiration model
  - phase: 38-signal-schema-unification
    provides: Single COLONY_STATE.json with nested signals
provides:
  - Commands read signals inline from COLONY_STATE.json
  - No legacy utility function calls
  - Documentation reflects unified state structure
affects: [40-02-PLAN]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Inline TTL filtering (expires_at null or > now)
    - Single state file with nested sections

key-files:
  created: []
  modified:
    - .claude/commands/ant/ant.md
    - .claude/commands/ant/plan.md
    - .claude/commands/ant/build.md
    - .claude/commands/ant/organize.md
    - .claude/commands/ant/pause-colony.md
    - .claude/commands/ant/resume-colony.md
    - .claude/commands/ant/continue.md
    - .claude/commands/ant/init.md

key-decisions:
  - "TTL filtering replaces decay math in all commands"
  - "Signals read inline from COLONY_STATE.json, no utility calls"
  - "Memory compression via cap enforcement, not utility function"

patterns-established:
  - "Inline signal filtering: check expires_at is null OR > current timestamp"

# Metrics
duration: 3min
completed: 2026-02-06
---

# Phase 40 Plan 01: State Utility Alignment Summary

**Removed legacy utility calls (pheromone-batch, pheromone-cleanup, memory-compress) and updated commands to read signals inline from COLONY_STATE.json**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-06T15:30:00Z
- **Completed:** 2026-02-06T15:33:00Z
- **Tasks:** 3
- **Files modified:** 8

## Accomplishments
- ant.md documents single COLONY_STATE.json with nested sections
- All signal-reading commands use inline TTL filtering
- Removed Bayesian terminology per SIMP-05
- validate-state now uses "colony" instead of "all"

## Task Commits

Each task was committed atomically:

1. **Task 1: Update ant.md documentation** - `dc32189` (docs)
2. **Task 2: Update signal-reading commands** - `487a576` (feat)
3. **Task 3: Update continue.md and init.md** - `e835903` (feat)

## Files Created/Modified
- `.claude/commands/ant/ant.md` - Updated state documentation, TTL expiration, removed Bayesian
- `.claude/commands/ant/plan.md` - Inline signal reading from COLONY_STATE.json
- `.claude/commands/ant/build.md` - Inline signal reading with TTL filtering
- `.claude/commands/ant/organize.md` - Inline signal reading with TTL filtering
- `.claude/commands/ant/pause-colony.md` - Inline signal reading with expires_at
- `.claude/commands/ant/resume-colony.md` - Inline signal reading with expires_at
- `.claude/commands/ant/continue.md` - Removed pheromone-cleanup and memory-compress calls
- `.claude/commands/ant/init.md` - Changed validate-state all to validate-state colony

## Decisions Made
- TTL filtering (expires_at) replaces decay math in all commands
- Signals read inline from COLONY_STATE.json signals array, no utility calls
- Memory compression handled by cap enforcement when writing (30 decisions, 50 events)
- Expired signals filtered on read, no explicit cleanup needed

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All commands now use inline signal reading from COLONY_STATE.json
- Ready for plan 40-02 (aether-utils.sh cleanup) which can now remove unused utility functions
- No blockers

---
*Phase: 40-state-utility-alignment*
*Completed: 2026-02-06*
