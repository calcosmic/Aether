---
phase: 20-utility-modules
plan: 01
subsystem: infra
tags: [bash, jq, pheromone, decay, exponential-math, shell-utilities]

# Dependency graph
requires:
  - phase: 19-audit-fixes-utility-scaffold
    provides: aether-utils.sh scaffold with case dispatch, json_ok/json_err helpers, atomic-write.sh
provides:
  - 5 pheromone math subcommands (decay, effective, batch, cleanup, combine)
  - Deterministic pheromone calculations replacing LLM-approximated math
affects: [20-02-state-validation, 20-03-memory-ops, 20-04-error-tracking, 21-integration]

# Tech tracking
tech-stack:
  added: []
  patterns: [inline-jq-computation, file-read-transform-atomic-write]

key-files:
  created: []
  modified: [.aether/aether-utils.sh]

key-decisions:
  - "Removed local keyword from case branches (bash local only valid inside functions)"
  - "Used sub() to strip fractional seconds before jq fromdate for ISO-8601 compatibility"

patterns-established:
  - "Inline jq computation in case branches: arg validation, jq expression, json_ok output"
  - "File-read + jq transform + atomic_write pattern for state-modifying commands"

# Metrics
duration: 2min
completed: 2026-02-03
---

# Phase 20 Plan 01: Pheromone Math Summary

**5 pheromone math subcommands (decay, effective, batch, cleanup, combine) using jq for deterministic computation**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-03T17:07:45Z
- **Completed:** 2026-02-03T17:09:45Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- All 5 pheromone math subcommands operational with correct JSON output
- Exponential decay formula verified: 1 half-life yields exactly 0.5
- Batch and cleanup commands handle empty arrays, permanent signals (null half_life), and expired signals
- Help text updated to list all 7 commands (help, version, + 5 pheromone)
- Total script: 87 lines (well under 100 budget)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add 5 pheromone subcommands to aether-utils.sh** - `37e5c40` (feat)

**Plan metadata:** [pending]

## Files Created/Modified
- `.aether/aether-utils.sh` - Added pheromone-decay, pheromone-effective, pheromone-batch, pheromone-cleanup, pheromone-combine subcommands (41 lines added)

## Decisions Made
- Removed `local` keyword from case branches since bash `local` is only valid inside functions -- used plain variable assignment instead
- Kept fractional-second stripping (`sub("\\.[0-9]+Z$";"Z")`) in batch and cleanup for ISO-8601 robustness per research Pitfall 1

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed `local` keyword usage outside functions**
- **Found during:** Task 1 (pheromone-batch and pheromone-cleanup testing)
- **Issue:** Plan code used `local now`, `local before`, `local result`, `local after` in case branches, but bash `local` is only valid inside function definitions
- **Fix:** Removed all `local` keywords from case branch variable declarations
- **Files modified:** .aether/aether-utils.sh
- **Verification:** Both pheromone-batch and pheromone-cleanup execute without errors
- **Committed in:** 37e5c40 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Necessary for correct operation. No scope creep.

## Issues Encountered
None beyond the auto-fixed `local` keyword issue.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- aether-utils.sh has working pheromone module, ready for state validation subcommands (Plan 20-02)
- Pattern established for inline jq computation in case branches
- 87 lines used of 300-line budget, leaving 213 lines for remaining 13 subcommands

---
*Phase: 20-utility-modules*
*Completed: 2026-02-03*
