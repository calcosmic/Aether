---
phase: 22-cleanup
plan: 01
subsystem: commands
tags: [pheromone-batch, aether-utils, cleanup, dead-code-removal]

# Dependency graph
requires:
  - phase: 20-utility-modules
    provides: pheromone-batch subcommand in aether-utils.sh
  - phase: 21-command-integration
    provides: build.md and status.md reference implementations using pheromone-batch
provides:
  - 4 command files wired to pheromone-batch (plan, pause-colony, resume-colony, colonize)
  - aether-utils.sh trimmed from 15 to 11 subcommands (no orphans)
affects: [22-02-PLAN, 23-enforcement]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "All pheromone decay goes through aether-utils.sh pheromone-batch -- no inline formulas"

key-files:
  created: []
  modified:
    - ".claude/commands/ant/plan.md"
    - ".claude/commands/ant/pause-colony.md"
    - ".claude/commands/ant/resume-colony.md"
    - ".claude/commands/ant/colonize.md"
    - ".aether/aether-utils.sh"

key-decisions:
  - "Followed plan exactly -- pure text replacement with no architectural changes"

patterns-established:
  - "pheromone-batch call pattern: Bash tool runs aether-utils.sh pheromone-batch, parse result array, filter by current_strength threshold"

# Metrics
duration: 2min
completed: 2026-02-03
---

# Phase 22 Plan 01: Wire pheromone-batch + Remove Dead Subcommands Summary

**4 command files wired to pheromone-batch for decay calculation; 4 orphaned subcommands (pheromone-combine, memory-token-count, memory-search, error-dedup) removed from aether-utils.sh**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-03T18:34:12Z
- **Completed:** 2026-02-03T18:35:46Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Eliminated all inline pheromone decay formulas from command files (CLEAN-01 satisfied)
- Removed 4 orphaned subcommands with zero consumers from aether-utils.sh (CLEAN-05 satisfied)
- aether-utils.sh now has exactly 11 subcommands, all with active consumers

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire pheromone-batch into plan.md, pause-colony.md, resume-colony.md, colonize.md** - `cba3bae` (feat)
2. **Task 2: Remove 4 orphaned subcommands from aether-utils.sh** - `11547c6` (fix)

## Files Created/Modified
- `.claude/commands/ant/plan.md` - Step 3 now calls pheromone-batch instead of inline formula
- `.claude/commands/ant/pause-colony.md` - Step 2 now calls pheromone-batch instead of inline formula
- `.claude/commands/ant/resume-colony.md` - Step 2 now calls pheromone-batch instead of inline formula
- `.claude/commands/ant/colonize.md` - Step 2 now calls pheromone-batch instead of inline formula
- `.aether/aether-utils.sh` - Removed pheromone-combine, memory-token-count, memory-search, error-dedup; updated help text

## Decisions Made
None - followed plan as specified.

## Deviations from Plan
None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- CLEAN-01 and CLEAN-05 are complete
- Ready for 22-02-PLAN.md which addresses CLEAN-02, CLEAN-03, CLEAN-04 (memory-compress, error-pattern-check, error-summary wiring)
- No blockers or concerns

---
*Phase: 22-cleanup*
*Completed: 2026-02-03*
