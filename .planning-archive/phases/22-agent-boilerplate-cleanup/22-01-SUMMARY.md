---
phase: 22-agent-boilerplate-cleanup
plan: 01
subsystem: agents
tags: [opencode, agent-definitions, boilerplate-removal, documentation]

# Dependency graph
requires: []
provides:
  - 9 OpenCode agent files cleaned of Aether Integration, Depth-Based Behavior, and Reference boilerplate sections
  - 5 Core agent descriptions updated to "Use this agent for..." format
  - All unique agent content (Spawn Protocol, flag-add, Spawning Sub-Workers, Refactoring Techniques, etc.) preserved
affects: [22-02, 22-03, any phase referencing .opencode/agents/]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Agent files as focused job descriptions: no universal integration boilerplate, only role-specific content"
    - "Description format: 'Use this agent for [task]. The [role] [capability].'"

key-files:
  created: []
  modified:
    - .opencode/agents/aether-watcher.md
    - .opencode/agents/aether-scout.md
    - .opencode/agents/aether-route-setter.md
    - .opencode/agents/aether-weaver.md
    - .opencode/agents/aether-probe.md
    - .opencode/agents/aether-ambassador.md
    - .opencode/agents/aether-tracker.md

key-decisions:
  - "Queen and builder were already clean — no changes needed, confirmed by reading both files before editing"
  - "Pre-existing lint:sync failure (34 Claude vs 33 OpenCode commands) and 2 test failures logged as deferred — not caused by this plan"

patterns-established:
  - "Boilerplate pattern: three removable sections are Aether Integration, Depth-Based Behavior, and Reference"
  - "Preserved pattern: Activity Logging is retained in all agents as an active operational section"

requirements-completed: [AGENT-01, AGENT-02, AGENT-03, AGENT-04]

# Metrics
duration: 2min
completed: 2026-02-19
---

# Phase 22 Plan 01: Agent Boilerplate Cleanup — Batches 1 & 2 Summary

**9 OpenCode agents stripped of 3 universal boilerplate sections each (Aether Integration, Depth-Based Behavior, Reference), cutting ~120 lines of redundant text while preserving all role-specific content**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-02-19T21:47:57Z
- **Completed:** 2026-02-19T21:50:49Z
- **Tasks:** 2 of 2
- **Files modified:** 7

## Accomplishments

- Removed Aether Integration, Depth-Based Behavior, and Reference sections from 7 agents (watcher, scout, route-setter, weaver, probe, ambassador, tracker)
- Updated 3 Core agent descriptions to "Use this agent for..." format (watcher, scout, route-setter)
- Confirmed queen and builder were already clean — no changes required
- All unique agent content preserved: Spawn Protocol (queen), Spawning Sub-Workers (builder), Creating Flags/flag-add (watcher), Spawning section (scout), Planning Discipline (route-setter), Refactoring Techniques (weaver), Testing Strategies (probe), Integration Patterns (ambassador), Debugging Techniques/3-Fix Rule (tracker)

## Task Commits

Each task was committed atomically:

1. **Task 1: Strip boilerplate from Core 5 agents (Batch 1)** - `4541534` (feat)
2. **Task 2: Strip boilerplate from Development 4 agents (Batch 2)** - `f937cac` (feat)

**Plan metadata:** TBD (docs: complete plan)

## Files Created/Modified

- `.opencode/agents/aether-watcher.md` - Removed Aether Integration, Depth-Based Behavior, Reference; updated description
- `.opencode/agents/aether-scout.md` - Removed Aether Integration, Depth-Based Behavior, Reference; updated description
- `.opencode/agents/aether-route-setter.md` - Removed Aether Integration, Reference (had no Depth-Based Behavior); updated description
- `.opencode/agents/aether-weaver.md` - Removed Aether Integration, Depth-Based Behavior, Reference
- `.opencode/agents/aether-probe.md` - Removed Aether Integration, Depth-Based Behavior, Reference
- `.opencode/agents/aether-ambassador.md` - Removed Aether Integration, Depth-Based Behavior, Reference
- `.opencode/agents/aether-tracker.md` - Removed Aether Integration, Depth-Based Behavior, Reference

## Decisions Made

- Queen and builder were already clean before this plan ran — their earlier cleanup was done as part of a prior session. Only 7 of the 9 agents needed edits.
- Route-setter had no Depth-Based Behavior section (only Aether Integration and Reference) — removed only what was present.

## Deviations from Plan

None — plan executed exactly as written. The only discovery was that queen and builder were already clean, which is a positive finding, not a deviation.

## Issues Encountered

Two pre-existing issues discovered during verification (both confirmed pre-existing via git stash):
1. `npm run lint:sync` failing: 34 Claude Code commands vs 33 OpenCode commands (count mismatch pre-dates this plan)
2. `npm test` has 2 pre-existing failures in `validate-state.test.js`

Both logged to deferred-items. Neither was caused by this plan's changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Batch 1 (Core 5) and Batch 2 (Development 4) complete — 9 agents cleaned
- Plan 22-02 covers Batch 3 (Knowledge Cluster: chronicler, keeper, auditor, sage)
- Plan 22-03 covers Batch 4 (Quality Cluster: guardian, measurer, includer, gatekeeper) and remaining agents

---
*Phase: 22-agent-boilerplate-cleanup*
*Completed: 2026-02-19*
