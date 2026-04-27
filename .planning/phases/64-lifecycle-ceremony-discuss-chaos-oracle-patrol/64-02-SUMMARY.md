---
phase: 64-lifecycle-ceremony-discuss-chaos-oracle-patrol
plan: 02
subsystem: commands
tags: [chaos, oracle, midden, pheromone, hive, wrappers]

# Dependency graph
requires: []
provides:
  - "Chaos wrapper Step 6.6 midden recurrence detection with REDIRECT suggestion"
  - "Oracle wrapper post-completion persistence suggestions for pheromone-write and hive-store"
affects: [chaos-command, oracle-command, lifecycle-ceremony]

# Tech tracking
tech-stack:
  added: []
  patterns: [midden-recurrence-check, oracle-persistence-suggestion]

key-files:
  created: []
  modified:
    - .claude/commands/ant/chaos.md
    - .opencode/commands/ant/chaos.md
    - .aether/commands/chaos.yaml
    - .claude/commands/ant/oracle.md
    - .opencode/commands/ant/oracle.md
    - .aether/commands/oracle.yaml

key-decisions:
  - "Midden recurrence threshold set at 3+ occurrences per category before suggesting REDIRECT"
  - "Oracle persistence suggestions are user-approved, not auto-created"
  - "Persistence only suggested when oracle completes successfully (not blocked/stopped)"

patterns-established:
  - "Post-action intelligence: wrappers suggest colony learning actions after primary work"

requirements-completed: [CERE-10, CERE-11]

# Metrics
duration: 3min
completed: 2026-04-27
---

# Phase 64 Plan 02: Chaos Midden Recurrence + Oracle Persistence Summary

**Chaos wrapper gains midden recurrence check with REDIRECT suggestion; oracle wrapper gains post-completion persistence suggestions for pheromone-write and hive-store**

## Performance

- **Duration:** 3 min
- **Started:** 2026-04-27T18:44:43Z
- **Completed:** 2026-04-27T18:47:53Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Chaos wrappers (Claude + OpenCode) now check midden for recurring failure patterns (3+ per category) and suggest REDIRECT pheromone to user
- Oracle wrappers (Claude + OpenCode) now suggest persisting high-value research findings as FOCUS pheromones or hive wisdom after successful completion
- YAML source files updated with follow_up sections documenting the new behavior

## Task Commits

Each task was committed atomically:

1. **Task 1: Add midden recurrence check to chaos wrapper (Claude + OpenCode)** - `362e222c` (feat)
2. **Task 2: Add post-completion persistence suggestions to oracle wrapper (Claude + OpenCode)** - `3e715f66` (feat)

## Files Created/Modified
- `.claude/commands/ant/chaos.md` - Added Step 6.6 midden recurrence check between Step 6.5 and Step 7
- `.opencode/commands/ant/chaos.md` - Added Step 6.6 midden recurrence check (mirrors Claude version)
- `.aether/commands/chaos.yaml` - Added follow_up.midden_recurrence entry
- `.claude/commands/ant/oracle.md` - Added Post-Completion Persistence Suggestions section at end
- `.opencode/commands/ant/oracle.md` - Added Post-Completion Persistence Suggestions section (mirrors Claude version)
- `.aether/commands/oracle.yaml` - Added follow_up.persistence entry

## Decisions Made
- Midden recurrence threshold of 3+ occurrences per category chosen as meaningful signal (prevents noise from one-off failures)
- User-approval pattern for both REDIRECT and persistence suggestions aligns with existing wrapper-runtime contract (wrappers suggest, users approve)
- Oracle persistence gated on completion status to avoid surfacing incomplete or blocked research

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- CERE-10 and CERE-11 requirements satisfied
- All changes are wrapper-only (no Go code changes), no test impact
- Ready for plan 03 (patrol ceremony enhancements)

---
*Phase: 64-lifecycle-ceremony-discuss-chaos-oracle-patrol*
*Completed: 2026-04-27*
