---
phase: 30-automation
plan: 01
subsystem: build-orchestration
tags: [reviewer, debugger, advisory-mode, wave-loop, auto-spawn, build.md]

# Dependency graph
requires:
  - phase: 29-colony-intelligence
    provides: Calibrated watcher scoring rubric, wave parallelism, auto-approval
provides:
  - Advisory reviewer spawn after each wave in build.md Step 5c.i
  - Debugger spawn on worker retry failure in build.md Step 5c.f2
  - Post-debugger criticality-aware task handling in Step 5c.g
affects: [30-02, 30-03, 31, 32]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Advisory mode spawn: reuse existing worker spec (watcher-ant.md) with constrained prompt instead of new caste"
    - "Debugger spawn: reuse builder-ant.md with PATCH constraints instead of new caste"
    - "Severity boundary definitions: CRITICAL/HIGH/MEDIUM/LOW with rebuild-only-on-CRITICAL gate"

key-files:
  created: []
  modified:
    - ".claude/commands/ant/build.md"

key-decisions:
  - "Reviewer reuses watcher-ant.md in advisory mode -- no new reviewer-ant.md file created"
  - "Debugger reuses builder-ant.md with patch constraints -- no new debugger-ant.md file created"
  - "Retry threshold changed from < 2 to < 1: worker gets exactly one retry before debugger spawns"
  - "LIGHTWEIGHT mode and single-worker waves skip reviewer entirely"
  - "Only CRITICAL severity triggers wave rebuild (max 2 iterations per wave)"
  - "Post-debugger uses criticality inference: success-criterion tasks get warnings, supporting tasks get skipped"

patterns-established:
  - "Advisory spawn pattern: spawn existing caste with mode override in prompt, findings displayed but non-blocking"
  - "Debugger spawn pattern: spawn builder with diagnostic constraints, PATCH-only, preserve original approach"
  - "Graduated failure handling: retry -> debugger -> criticality-aware skip/flag"

# Metrics
duration: 2min
completed: 2026-02-05
---

# Phase 30 Plan 01: Reviewer & Debugger Summary

**Advisory reviewer auto-spawn after each wave (Step 5c.i) and debugger auto-spawn on retry failure (Step 5c.f2) integrated into build.md execution loop**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-05T13:18:48Z
- **Completed:** 2026-02-05T13:21:10Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Advisory reviewer spawns after each wave (skipped in LIGHTWEIGHT mode and single-worker waves), displays findings inline, only CRITICAL triggers rebuild with max 2 iterations
- Debugger spawns after worker retry failure using builder-ant.md with PATCH-only constraints, handles both successful fix and undiagnosable cases
- Graduated failure flow: worker attempt -> one retry -> debugger diagnosis -> criticality-aware resolution
- Step 7e progress checklist updated with "Post-Wave Review" entry

## Task Commits

Each task was committed atomically:

1. **Task 1: Add advisory reviewer spawn after each wave (Step 5c.i)** - `8a0d860` (feat)
2. **Task 2: Add debugger spawn on worker retry failure (Step 5c.f2)** - `7e421b1` (feat)

## Files Created/Modified
- `.claude/commands/ant/build.md` - Added Step 5c.i (advisory reviewer), Step 5c.f2 (debugger spawn), modified Step 5c.f (retry threshold < 1), replaced Step 5c.g (post-debugger logic), updated Step 7e checklist

## Decisions Made
- Retry threshold changed from `< 2` to `< 1` so workers get exactly one retry before debugger spawns (per CONTEXT.md: "worker gets one retry attempt first, debugger triggers on second failure")
- Reviewer skipped for single-worker waves (no cross-worker interaction to catch) in addition to LIGHTWEIGHT mode
- Severity boundary definitions included directly in reviewer prompt: CRITICAL = code doesn't run / security / data corruption; HIGH/MEDIUM/LOW = non-blocking
- Post-debugger criticality inference: tasks mapping to success criteria get flagged for review; supporting tasks get skipped
- Six-caste architecture preserved: no new worker spec files created

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- build.md Step 5c now has complete reviewer + debugger integration
- Ready for Plan 02 (pheromone recommendations + tech debt report)
- Ready for Plan 03 (ANSI visual output -- reviewer summary display will get colors)
- No blockers

---
*Phase: 30-automation*
*Completed: 2026-02-05*
