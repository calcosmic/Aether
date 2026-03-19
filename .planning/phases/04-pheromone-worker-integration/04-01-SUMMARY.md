---
phase: 04-pheromone-worker-integration
plan: 01
subsystem: agents
tags: [pheromone, agent-definitions, builder, watcher, scout, signal-handling]

# Dependency graph
requires:
  - phase: 03-pheromone-signal-plumbing
    provides: "prompt_section injection pipeline (pheromone-prime, colony-prime, build-wave injection)"
provides:
  - "Builder agent with pheromone_protocol section for REDIRECT/FOCUS/FEEDBACK handling"
  - "Watcher agent with verification-oriented pheromone protocol"
  - "Scout agent with research-scoped pheromone protocol"
  - "Byte-identical mirror copies in .aether/agents-claude/"
affects: [04-02-cross-phase-signal-verification, build-wave, colony-prime]

# Tech tracking
tech-stack:
  added: []
  patterns: [pheromone_protocol section in agent definitions]

key-files:
  created: []
  modified:
    - ".claude/agents/ant/aether-builder.md"
    - ".claude/agents/ant/aether-watcher.md"
    - ".claude/agents/ant/aether-scout.md"
    - ".aether/agents-claude/aether-builder.md"
    - ".aether/agents-claude/aether-watcher.md"
    - ".aether/agents-claude/aether-scout.md"

key-decisions:
  - "Placed pheromone_protocol after critical_rules, before return_format -- signals are critical but secondary to core rules like TDD"
  - "Kept each protocol under 35 lines -- principle-based, not rule-based, since workers are LLMs"
  - "Pre-existing lint:sync command count mismatch (38 vs 37) logged to deferred-items, not fixed (out of scope)"

patterns-established:
  - "pheromone_protocol section: standardized XML section for agent signal handling, placed after critical_rules"
  - "Agent-specific adaptations: same base protocol with role-specific behavioral rules"

requirements-completed: [PHER-03]

# Metrics
duration: 3min
completed: 2026-03-19
---

# Phase 4 Plan 01: Core Agent Pheromone Protocol Summary

**Added pheromone_protocol sections to builder, watcher, and scout agents with role-specific signal handling for REDIRECT, FOCUS, and FEEDBACK**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-19T18:26:08Z
- **Completed:** 2026-03-19T18:29:11Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Builder agent now has explicit instructions to constrain implementation choices on REDIRECT, increase test coverage on FOCUS areas, and adjust coding patterns on FEEDBACK
- Watcher agent treats REDIRECT signals as verification checkpoints (verify avoidance), FOCUS as deeper scrutiny areas, FEEDBACK as quality scoring weights
- Scout agent constrains research scope on REDIRECT (never recommend redirected patterns), prioritizes FOCUS areas, and uses FEEDBACK for source credibility weighting
- All three mirror copies in .aether/agents-claude/ are byte-identical to canonical

## Task Commits

Each task was committed atomically:

1. **Task 1: Add pheromone_protocol to builder, watcher, scout** - `8634cce` (feat)
2. **Task 2: Mirror agent definitions and verify sync** - `cf13e0d` (chore)

## Files Created/Modified
- `.claude/agents/ant/aether-builder.md` - Added 35-line pheromone_protocol section with builder-specific implementation constraints
- `.claude/agents/ant/aether-watcher.md` - Added 34-line pheromone_protocol section with verification checkpoint behavior
- `.claude/agents/ant/aether-scout.md` - Added 35-line pheromone_protocol section with research scope constraints
- `.aether/agents-claude/aether-builder.md` - Byte-identical mirror of canonical builder
- `.aether/agents-claude/aether-watcher.md` - Byte-identical mirror of canonical watcher
- `.aether/agents-claude/aether-scout.md` - Byte-identical mirror of canonical scout

## Decisions Made
- Placed pheromone_protocol after critical_rules, before return_format -- signals are important but secondary to core rules (TDD Iron Law, Evidence Iron Law)
- Kept each protocol under 35 lines using principle-based instructions rather than prescriptive conditional logic
- Pre-existing lint:sync command count mismatch (Claude Code 38 vs OpenCode 37) logged to deferred-items.md, not fixed (out of scope)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `npm run lint:sync` fails with command count mismatch (38 vs 37) -- this is pre-existing and unrelated to agent definition sync. Verified by stashing changes and confirming same failure on clean state. Agent mirror sync verified independently via diff (zero differences).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Agent definitions now contain explicit pheromone protocol instructions
- Ready for 04-02 (cross-phase signal influence verification and midden threshold auto-REDIRECT tests)
- PHER-03 requirement satisfied, enabling PHER-04 and PHER-05 verification

## Self-Check: PASSED

All 7 files verified present. Both commit hashes (8634cce, cf13e0d) confirmed in git log.

---
*Phase: 04-pheromone-worker-integration*
*Completed: 2026-03-19*
