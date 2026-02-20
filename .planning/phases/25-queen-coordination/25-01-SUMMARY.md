---
phase: 25-queen-coordination
plan: 01
subsystem: agent-definition
tags: [queen, escalation, workflow-patterns, coordination, build-command, status-command]

# Dependency graph
requires:
  - phase: 23-agent-resilience
    provides: "failure_modes XML structure established in agent files"
  - phase: 24-template-integration
    provides: "build.md fully wired with templates — safe baseline to add pattern selection"
provides:
  - "4-tier escalation chain in Queen failure_modes (Tiers 1-3 silent, Tier 4 user-visible)"
  - "6 named workflow patterns with selection heuristics in Queen agent"
  - "Pattern selection and announcement in build.md (both platforms)"
  - "Escalation banner in build.md wave failure path (both platforms)"
  - "Conditional escalation state in status.md (both platforms)"
affects: [25-02, 25-03, any phase using /ant:build or /ant:status]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "4-tier escalation: worker retry -> parent reassign -> Queen reassign -> user escalation"
    - "Keyword-based pattern selection from phase name at build start"
    - "Escalation state derivable from flag source='escalation' — no new state file needed"
    - "Conditional display: show only when non-zero (no noise when clean)"

key-files:
  created: []
  modified:
    - ".opencode/agents/aether-queen.md"
    - ".claude/commands/ant/build.md"
    - ".opencode/commands/ant/build.md"
    - ".claude/commands/ant/status.md"
    - ".opencode/commands/ant/status.md"

key-decisions:
  - "Critical Failures (STOP immediately) separated from Escalation Chain (tiered retry) — two distinct failure classes"
  - "Tiers 1-3 fully silent — user only hears about failures that survive 3 retry/reassign attempts"
  - "Escalation state derived from flag source='escalation' filter — no new aether-utils.sh commands needed"
  - "Add Tests documented as SPBV variant, not a 7th pattern — selection overhead not worth it"
  - "selected_pattern stored as local variable in build.md — ephemeral per build, captured in BUILD SUMMARY"

requirements-completed: [COORD-01, COORD-02]

# Metrics
duration: 4min
completed: 2026-02-20
---

# Phase 25 Plan 01: Queen Coordination — Escalation Chain + Workflow Patterns Summary

**4-tier escalation chain and 6 named workflow patterns added to Queen agent, wired into build.md (pattern selection + escalation banner) and status.md (conditional escalation count) on both Claude Code and OpenCode platforms**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-20T00:52:25Z
- **Completed:** 2026-02-20T00:56:26Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- Queen agent `<failure_modes>` restructured: Critical Failures (immediate STOP) separated from new 4-tier Escalation Chain (Tiers 1-3 silent, Tier 4 user-visible banner)
- 6 named workflow patterns added to Queen agent with Use when, Phases, Rollback, and Announce fields, plus keyword-based Pattern Selection table
- build.md (both platforms): Step 5.0.5 added for pattern selection/announcement before worker spawning; escalation banner added to partial wave failure path in Step 5.2; Pattern line added to BUILD SUMMARY
- status.md (both platforms): escalation state computation added in Step 2 (jq filter on flag source="escalation"); conditional "Escalated: N task(s)" line added to Step 3 display

## Task Commits

1. **Task 1: Add escalation chain and workflow patterns to Queen agent definition** - `af18cf3` (feat)
2. **Task 2: Wire pattern selection and escalation banner into build.md (both platforms)** - `5a24a05` (feat)
3. **Task 3: Add conditional escalation state to status.md (both platforms)** - `f24f21e` (feat)

**Plan metadata:** [created below]

## Files Created/Modified

- `.opencode/agents/aether-queen.md` — Added `### Escalation Chain` inside failure_modes (4 tiers + banner template + flag-add instruction) and new `## Workflow Patterns` section (6 patterns + Pattern Selection table)
- `.claude/commands/ant/build.md` — Added Step 5.0.5 (pattern selection), escalation path in Step 5.2, Pattern line in Step 7 BUILD SUMMARY
- `.opencode/commands/ant/build.md` — Same additions as Claude Code build.md
- `.claude/commands/ant/status.md` — Added escalation state computation in Step 2, conditional Escalated line in Step 3
- `.opencode/commands/ant/status.md` — Same additions as Claude Code status.md

## Decisions Made

- **Critical Failures vs Escalation Chain distinction:** The original Minor/Major failure split was replaced with two clearly named categories. "Critical Failures" (STOP immediately) covers COLONY_STATE corruption, orphaned workers, and destructive git ops. The new "Escalation Chain" covers all other failures with 4 tiers before user sees anything.
- **Tiers 1-3 fully silent:** Consistent with user constraint that colony should be "very patient" — only Tier 4 surfaces after 3 attempts have been exhausted.
- **Escalation state via flag filter:** Research identified that `flag-list --source` may not be supported; used `jq` post-processing client-side in status command to filter by `.source == "escalation"`. No new aether-utils.sh commands needed.
- **selected_pattern as local variable:** Pattern is ephemeral per build — captured in BUILD SUMMARY output and HANDOFF.md. Not persisted to COLONY_STATE.json. Simpler, no state file growth.
- **Add Tests = SPBV variant:** Not a 7th pattern. Documented clearly in Queen agent as a note.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `npm run lint:sync` exits 1 due to pre-existing content-level drift across 10+ files between Claude Code and OpenCode directories. This is documented known debt in STATE.md. No new drift was introduced by this plan — confirmed from diff output showing only expected intentional platform differences (Claude Code uses "Run using the Bash tool with description" style; OpenCode uses shorter "Run:" style).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Requirements COORD-01 and COORD-02 complete
- Ready for Phase 25-02 (agent merges: Architect into Keeper, Guardian into Auditor)
- Both plans edit Queen agent — COORD-01/02 done together in this plan as recommended by research to avoid merge conflicts

---
*Phase: 25-queen-coordination*
*Completed: 2026-02-20*
