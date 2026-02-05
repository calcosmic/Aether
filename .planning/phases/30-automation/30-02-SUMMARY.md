---
phase: 30-automation
plan: 02
subsystem: build-orchestration
tags: [pheromone-recommendations, tech-debt-report, build-output, project-completion, continue.md, build.md]

# Dependency graph
requires:
  - phase: 30-automation-01
    provides: Advisory reviewer spawn (Step 5c.i) and debugger spawn (Step 5c.f2) in build.md
provides:
  - Pheromone recommendations in build.md Step 7e (max 3 natural language suggestions)
  - Between-wave urgent recommendations in Step 5c.i on CRITICAL findings
  - Tech debt report generation at project completion in continue.md Step 2.5
affects: [30-03, 31, 32]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pheromone recommendations: Queen synthesizes max 3 natural language observations from build data (worker results, watcher report, errors.json, reviewer findings)"
    - "Tech debt aggregation: error-summary + error-pattern-check + memory.json phase_learnings + activity.log -> comprehensive report"
    - "Between-wave urgent recommendations: immediate inline guidance on CRITICAL findings, separate from end-of-build max-3"

key-files:
  created: []
  modified:
    - ".claude/commands/ant/build.md"
    - ".claude/commands/ant/continue.md"

key-decisions:
  - "Between-wave urgent recommendations are separate from end-of-build max-3 -- they appear immediately on CRITICAL and do not count toward the limit"
  - "Recommendations must sound like senior engineer observations, not automated alerts -- explicitly prohibited from starting with Run: or /ant:"
  - "Tech debt report persisted to .aether/data/tech-debt-report.md AND displayed in terminal"
  - "Step 2.5 is strictly conditional on no-next-phase -- mid-project continue flow completely unaffected"

patterns-established:
  - "Pheromone recommendation pattern: analyze build outcomes against trigger patterns, produce natural language guidance with Signal attribution"
  - "Tech debt report pattern: aggregate errors.json + memory.json + activity.log at project completion into persistent report"

# Metrics
duration: 2min
completed: 2026-02-05
---

# Phase 30 Plan 02: Pheromone Recommendations & Tech Debt Report Summary

**Max-3 natural language pheromone recommendations in build output with signal attribution, between-wave urgent recs on CRITICAL findings, and comprehensive tech debt report at project completion persisted to .aether/data/tech-debt-report.md**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-05T13:23:13Z
- **Completed:** 2026-02-05T13:24:49Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Build output now includes max 3 natural language pheromone recommendations in Step 7e, synthesized from worker results, watcher report, errors.json flagged patterns, and reviewer findings
- Between-wave urgent recommendations appear immediately in Step 5c.i when reviewer detects CRITICAL issues (both rebuild and persist-after-2-rebuilds cases)
- Tech debt report generates at project completion (continue.md Step 2.5) with Persistent Issues, Error Summary, Unresolved Items, Phase Quality Trend, and Recommendations
- Tech debt report is both displayed and persisted to .aether/data/tech-debt-report.md

## Task Commits

Each task was committed atomically:

1. **Task 1: Add pheromone recommendations to build output (AUTO-03)** - `3a975e2` (feat)
2. **Task 2: Add tech debt report at project completion (INT-06)** - `490f2b3` (feat)

## Files Created/Modified
- `.claude/commands/ant/build.md` - Added between-wave urgent recommendations in Step 5c.i, pheromone recommendations section in Step 7e with trigger patterns, format constraints, and display format
- `.claude/commands/ant/continue.md` - Added Step 2.5 (tech debt report generation) conditional on project completion, updated completion message to reference tech-debt-report.md

## Decisions Made
- Between-wave urgent recommendations are separate from end-of-build max-3 and do not count toward the limit -- urgent signals need immediate visibility
- Both the rebuild case (critical_count > 0, rebuild < 2) and the persist case (critical_count > 0, rebuild >= 2) get urgent recommendations, with different messaging intensity
- Tech debt report uses all available data sources: error-summary, error-pattern-check, memory.json phase_learnings, and activity.log
- Step 2 flow restructured to proceed to Step 2.5 instead of stopping immediately, with explicit "Stop here" after Step 2.5 completes

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- build.md now has complete pheromone recommendations (Step 7e) and urgent between-wave recommendations (Step 5c.i)
- continue.md generates tech debt report at project completion (Step 2.5)
- Ready for Plan 03 (ANSI visual output -- recommendations display will get colors)
- No blockers

---
*Phase: 30-automation*
*Completed: 2026-02-05*
