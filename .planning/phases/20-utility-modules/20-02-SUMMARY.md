---
phase: 20-utility-modules
plan: 02
subsystem: infra
tags: [bash, jq, validation, json-schema, state-files]

# Dependency graph
requires:
  - phase: 19-audit-fixes-utility-scaffold
    provides: aether-utils.sh scaffold with file-lock.sh and atomic-write.sh
  - phase: 20-utility-modules plan 01
    provides: pheromone math subcommands and JSON output pattern
provides:
  - validate-state subcommand with 6 validators (colony, pheromones, errors, memory, events, all)
  - Empty state files (errors.json, memory.json, events.json) for downstream modules
affects: [20-03-memory-ops, 20-04-error-tracking, 21-integration]

# Tech tracking
tech-stack:
  added: []
  patterns: [jq-schema-validation, nested-case-dispatch]

key-files:
  created: [.aether/data/errors.json, .aether/data/events.json, .aether/data/memory.json]
  modified: [.aether/aether-utils.sh]

key-decisions:
  - "Inline jq type-checking per validator (no shared helper function) -- more compact than check_fields approach"
  - "Created empty state files (errors.json, memory.json, events.json) so validators can pass immediately"
  - "validate-state all uses recursive self-invocation for each target file"

patterns-established:
  - "Nested case dispatch: top-level validate-state -> nested colony|pheromones|errors|memory|events|all"
  - "jq def chk(f;t) pattern for field-presence + type checking with descriptive error messages"

# Metrics
duration: 2min
completed: 2026-02-03
---

# Phase 20 Plan 02: State Validation Summary

**6 validate-state subcommands using jq schema validation for all 5 colony state files with field-level error reporting**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-03T17:11:29Z
- **Completed:** 2026-02-03T17:13:25Z
- **Tasks:** 1
- **Files modified:** 4

## Accomplishments
- Added validate-state subcommand with nested case dispatch for 6 sub-subcommands
- Colony validator checks 5 required fields (goal, state, current_phase, workers, spawn_outcomes) with type checking
- Pheromones validator checks signals array structure and per-signal required fields
- Errors validator checks errors/flagged_patterns arrays and per-error required fields
- Memory validator checks phase_learnings, decisions, patterns arrays
- Events validator checks events array and per-event required fields
- validate-state all runs all 5 validators and reports aggregate pass/fail
- Created missing state files (errors.json, memory.json, events.json) for complete validation coverage

## Task Commits

Each task was committed atomically:

1. **Task 1: Add validate-state subcommand with 6 validators** - `e0d4635` (feat)

**Plan metadata:** [pending] (docs: complete plan)

## Files Created/Modified
- `.aether/aether-utils.sh` - Added validate-state case branch with 6 validators (~75 lines added, total now 163 lines)
- `.aether/data/errors.json` - Empty canonical state file (errors array + flagged_patterns array)
- `.aether/data/memory.json` - Empty canonical state file (phase_learnings + decisions + patterns arrays)
- `.aether/data/events.json` - Empty canonical state file (events array)

## Decisions Made
- Used inline jq approach rather than shared check_fields helper -- plan explicitly noted "do NOT add the check_fields helper"
- Created errors.json, memory.json, events.json as empty canonical state files since they didn't exist yet but are required for validation and downstream modules (error-add, memory-compress, etc.)
- validate-state all uses recursive bash self-invocation rather than function calls to keep the implementation flat

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Created missing state files (errors.json, memory.json, events.json)**
- **Found during:** Task 1 (verification step)
- **Issue:** errors.json, memory.json, events.json did not exist in .aether/data/ -- validators returned "file not found" which is correct behavior, but downstream modules (20-03, 20-04) need these files
- **Fix:** Created empty canonical state files matching the schemas from research
- **Files modified:** .aether/data/errors.json, .aether/data/memory.json, .aether/data/events.json
- **Verification:** All 5 validators pass, validate-state all reports aggregate pass:true
- **Committed in:** e0d4635 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** State files are required for validation testing and downstream modules. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 5 state files now exist and validate successfully
- validate-state all provides a single command to verify colony state integrity
- Ready for 20-03 (memory operations) and 20-04 (error tracking) which will read/write these files
- aether-utils.sh at 163 lines, ~137 lines of budget remaining for plans 03 and 04

---
*Phase: 20-utility-modules*
*Completed: 2026-02-03*
