---
phase: 05-pheromone-system
plan: 02
subsystem: pheromone-signals
tags: [pheromones, signals, instincts, colony-communication, bash, jq, worker-priming]

requires:
  - phase: 05-01
    provides: pheromone-write, pheromone-count, pheromone-read with decay — the canonical signal store this plan reads from
provides:
  - instinct-read subcommand filtering COLONY_STATE.json memory.instincts by confidence and status
  - pheromone-prime subcommand combining signals + instincts into a single prompt-ready markdown block
  - build.md Step 4 pheromone-prime call with "Primed: N signals, M instincts" display
  - build.md Step 5.1 pheromone_section injection into builder prompts with REDIRECT hard constraints reminder
  - build.md Step 5.4 pheromone_section injection into watcher prompts
  - checkpoint polling instruction in builder prompts for mid-work signal detection
affects: [build.md, watcher prompts, builder prompts, pheromone consumption]

tech-stack:
  added: []
  patterns: [pheromone-prime-combinator, prompt-section-injection, graceful-degradation-on-missing-data]

key-files:
  created: []
  modified:
    - .aether/aether-utils.sh
    - .claude/commands/ant/build.md

key-decisions:
  - "Watcher prompts receive pheromone_section between file list and verification sections (same injection pattern as builders)"
  - "Checkpoint polling is lightweight polling (check at natural breakpoints) not a formal queue — practical and zero-infrastructure"
  - "Graceful degradation: pheromone-prime failure never blocks a build — pheromone_section defaults to empty string"

patterns-established:
  - "pheromone-prime combinator: read signals + instincts, format both into a single markdown block, return JSON with counts and log_line"
  - "Prompt injection: insert pheromone_section after queen_wisdom, before Work section in builder prompts"
  - "REDIRECT labeling: grouped under 'HARD CONSTRAINTS - MUST follow' to distinguish from flexible guidance"

requirements-completed: [PHER-04, PHER-05]

duration: 8min
completed: 2026-02-17
---

# Phase 05 Plan 02: Pheromone Signal Consumption Summary

**`instinct-read` and `pheromone-prime` subcommands added to aether-utils.sh; build.md now injects active signals and learned instincts into every builder and watcher prompt via pheromone-prime, with REDIRECT constraints labeled as hard requirements and checkpoint polling for mid-work signal detection.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-17T~T09:00Z
- **Completed:** 2026-02-17
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- `instinct-read` subcommand reads `COLONY_STATE.json` `memory.instincts`, filters by confidence >= 0.5 and status != "disproven", sorts by confidence descending, caps at 5 (all overridable via flags)
- `pheromone-prime` subcommand combines active signals (with decay) and filtered instincts into a formatted markdown section ready for prompt injection, returns JSON with `signal_count`, `instinct_count`, `prompt_section`, and `log_line`
- `build.md` Step 4 now calls `pheromone-prime` and displays "Primed: N signals, M instincts" — builders and watchers always see the colony's accumulated guidance
- Builder prompts receive `pheromone_section` after queen_wisdom with a prominent REDIRECT hard constraints reminder
- Watcher prompts receive `pheromone_section` between "Files to verify" and "Verification" sections — watchers validate against REDIRECT constraints too
- Checkpoint polling instruction added to builder Work section — builders check for new signals at natural breakpoints

## Task Commits

Each task was committed atomically:

1. **Task 1: Add instinct-read and pheromone-prime subcommands** - `fdb2e9f` (feat)
2. **Task 2: Wire signal and instinct injection into build.md** - `faedcd3` (feat)

## Files Created/Modified

- `.aether/aether-utils.sh` - Added `instinct-read` case (lines ~4048-4117) and `pheromone-prime` case (lines ~4119-4243)
- `.claude/commands/ant/build.md` - Added pheromone-prime call to Step 4, pheromone_section injection to Step 5.1 (builder) and Step 5.4 (watcher), checkpoint polling instruction

## Decisions Made

- **Watcher prompt injection point:** Between "Files to verify" and "Verification" sections — watchers need to know REDIRECT constraints before they verify builder work
- **Checkpoint polling over formal queue:** Workers check for new signals at natural breakpoints using `pheromone-read all`. Zero infrastructure cost, practical implementation of the mid-work signal detection requirement
- **Graceful degradation as first-class:** `pheromone-prime` wrapped in `2>/dev/null` with empty-string fallback so pheromone unavailability never blocks a build

## Deviations from Plan

None - plan executed exactly as written. The Task 1 commit (`fdb2e9f`) predated this execution session — both subcommands were already implemented. Task 2 required adding the missing Watcher prompt injection (Step 5.4), which was specified in the plan but not yet applied.

## Issues Encountered

- Task 2 was partially complete from a prior session: Steps 4 and 5.1 changes existed in working tree but uncommitted, and Step 5.4 (Watcher prompt) was missing. Added the Watcher pheromone_section injection and committed all Task 2 changes atomically.

## Next Phase Readiness

- Signal consumption is complete: signals are written (05-01), read with decay (05-01), and injected into all worker prompts (05-02)
- Phase 05-03 can build on this foundation for any remaining pheromone system requirements
- Real-world test: next `/ant:build` call will display "Primed: N signals, M instincts" with active signals shown (currently 6 signals visible in pheromones.json)

## Self-Check

- FOUND: `.aether/aether-utils.sh` (verified instinct-read and pheromone-prime work via bash tests)
- FOUND: `.claude/commands/ant/build.md` (verified all three injection points present)
- FOUND: commit `fdb2e9f` (feat(05-02): add instinct-read and pheromone-prime subcommands)
- FOUND: commit `faedcd3` (feat(05-02): wire pheromone signals and instincts into builder and watcher prompts)

## Self-Check: PASSED

---
*Phase: 05-pheromone-system*
*Completed: 2026-02-17*
