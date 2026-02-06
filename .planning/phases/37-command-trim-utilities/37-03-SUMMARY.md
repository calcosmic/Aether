---
phase: 37-command-trim-utilities
plan: 03
subsystem: commands
tags: [colonize, codebase-analysis, surface-scan]

# Dependency graph
requires:
  - phase: 33-state-foundation
    provides: COLONY_STATE.json structure
provides:
  - Single-pass colonize command
  - CODEBASE.md output format
affects: [future-planning, colony-initialization]

# Tech tracking
tech-stack:
  added: []
  patterns: [surface-scan-pattern]

key-files:
  created: []
  modified:
    - .claude/commands/ant/colonize.md

key-decisions:
  - "Single-pass surface scan replaces multi-colonizer spawning"
  - "Output to .planning/CODEBASE.md instead of terminal"
  - "20 file read cap prevents context bloat"

patterns-established:
  - "Surface scan: manifest + docs + entry + config files"
  - "Minimal state updates (IDLE only, no complex structures)"

# Metrics
duration: 2min
completed: 2026-02-06
---

# Phase 37 Plan 03: Colonize Reduction Summary

**Surface scan colonize pattern replacing multi-colonizer spawning, 530 to 94 lines (82% reduction)**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-06T18:27:46Z
- **Completed:** 2026-02-06T18:30:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Reduced colonize.md from 530 lines to 94 lines (82% reduction)
- Replaced multi-colonizer spawning with single-pass surface scan
- Defined Glob patterns for 7 project types (JS/TS, Rust, Python, Go, Ruby, Java)
- Specified CODEBASE.md output format (~50 lines max)

## Task Commits

Each task was committed atomically:

1. **Task 1: Rewrite colonize.md with surface scan pattern** - `46180fc` (feat)
2. **Task 2: Verify colonize produces valid CODEBASE.md format** - verification only, no file changes

**Plan metadata:** [pending]

## Files Created/Modified
- `.claude/commands/ant/colonize.md` - Single-pass codebase analysis command (94 lines)

## Decisions Made
- Surface scan pattern covers common project types via Glob patterns
- 20 file read cap prevents context bloat while capturing key information
- Output to .planning/CODEBASE.md keeps analysis persistent
- Minimal state update (IDLE only) simplifies state machine

## Deviations from Plan
None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Colonize command ready for use in colony initialization flow
- Works with existing COLONY_STATE.json structure from Phase 33
- Outputs to .planning/ directory for planning integration

---
*Phase: 37-command-trim-utilities*
*Completed: 2026-02-06*
