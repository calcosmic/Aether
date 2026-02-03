---
phase: 21-command-integration
plan: 02
subsystem: colony-workers
tags: [pheromone, aether-utils, worker-specs, signal-computation, bash]

# Dependency graph
requires:
  - phase: 20-utility-modules
    provides: "aether-utils.sh with pheromone-effective subcommand"
  - phase: 16-worker-knowledge
    provides: "Worker spec files with pheromone sensitivity tables and worked examples"
provides:
  - "All 6 worker specs instruct ants to use bash pheromone-effective for signal computation"
  - "Deterministic pheromone math via shell utility instead of LLM-approximated multiplication"
affects: [21-command-integration remaining plans, any future worker spec updates]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Worker specs reference aether-utils.sh subcommands via Bash tool invocation"
    - "Fallback instruction included for resilience when shell utility unavailable"

key-files:
  created: []
  modified:
    - ".aether/workers/architect-ant.md"
    - ".aether/workers/builder-ant.md"
    - ".aether/workers/colonizer-ant.md"
    - ".aether/workers/route-setter-ant.md"
    - ".aether/workers/scout-ant.md"
    - ".aether/workers/watcher-ant.md"

key-decisions:
  - "Spawning scenario inline math also updated to Bash tool calls for consistency"

patterns-established:
  - "Worker spec Bash tool invocation: Run/Result format for worked examples"

# Metrics
duration: 2min
completed: 2026-02-03
---

# Phase 21 Plan 02: Pheromone Math Integration Summary

**All 6 worker specs now use `bash aether-utils.sh pheromone-effective` for deterministic signal computation instead of inline LLM multiplication**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-03T17:38:29Z
- **Completed:** 2026-02-03T17:40:10Z
- **Tasks:** 1
- **Files modified:** 6

## Accomplishments
- Replaced inline `effective_signal = sensitivity * signal_strength` formula with Bash tool calls in all 6 worker specs
- Updated worked examples to show Run/Result invocation pattern with same numeric values
- Updated spawning scenario decision process examples to use Bash tool calls
- Added fallback instruction for resilience when shell utility is unavailable
- Preserved all sensitivity tables, threshold interpretation, combination effects, and other spec content unchanged

## Task Commits

Each task was committed atomically:

1. **Task 1: Replace Pheromone Math in all 6 worker specs** - `3808042` (feat)

**Plan metadata:** [pending below]

## Files Created/Modified
- `.aether/workers/architect-ant.md` - Pheromone Math section + spawning scenario updated
- `.aether/workers/builder-ant.md` - Pheromone Math section + spawning scenario updated
- `.aether/workers/colonizer-ant.md` - Pheromone Math section + spawning scenario updated
- `.aether/workers/route-setter-ant.md` - Pheromone Math section + spawning scenario updated
- `.aether/workers/scout-ant.md` - Pheromone Math section + spawning scenario updated
- `.aether/workers/watcher-ant.md` - Pheromone Math section + spawning scenario updated

## Decisions Made
- Spawning scenario inline math (e.g., `FEEDBACK(0.6) * strength(0.8) = 0.48`) also updated to Bash tool invocations for consistency across the entire spec

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- INT-04 (pheromone-effective integration) complete
- All 6 worker specs now reference aether-utils.sh for signal computation
- Ready for remaining 21-command-integration plans (state validation, memory, error tracking integrations)

---
*Phase: 21-command-integration*
*Completed: 2026-02-03*
