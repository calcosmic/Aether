---
phase: 57-queen-md-pipeline-fix
plan: 03
subsystem: context-assembly
tags: [colony-prime, queen-md, wisdom-pipeline, seal]

# Dependency graph
requires:
  - phase: 54-colony-prime-prior-reviews-section
    provides: colony-prime context assembly pattern with section scoring
provides:
  - Global QUEEN.md injection into colony-prime worker context
  - Protected status for global_queen_md section (survives budget trimming)
  - Auto-promotion instructions in seal wrappers for high-confidence instincts
affects: [seal-lifecycle, colony-prime-context, wisdom-pipeline]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "global_queen_md as protected colony-prime section"
    - "seal wrapper auto-promotion of confidence >= 0.8 instincts"

key-files:
  created:
    - cmd/colony_prime_queen_test.go
  modified:
    - cmd/colony_prime_context.go
    - cmd/context_weighting.go
    - cmd/queen_hygiene_test.go
    - .claude/commands/ant/seal.md
    - .opencode/commands/ant/seal.md

key-decisions:
  - "Global QUEEN.md gets priority 5, relevance 0.75 -- below user_preferences (0.80) but above hive_wisdom (0.25)"
  - "global_queen_md is protected to ensure cross-colony wisdom always survives budget trimming"
  - "Seal wrapper auto-promotion is non-blocking -- failures logged but never stop the seal"

patterns-established:
  - "Protected section pattern: add to protectedSectionPolicy switch + relevanceScore switch in context_weighting.go"
  - "Wrapper parity: identical auto-promotion instructions in Claude and OpenCode seal wrappers"

requirements-completed: []

# Metrics
duration: 6min
completed: 2026-04-26
---

# Phase 57 Plan 03: Global QUEEN.md Injection and Seal Auto-Promotion Summary

**Global QUEEN.md wisdom injected into colony-prime context as protected section; seal wrappers gain auto-promotion of high-confidence instincts**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-26T20:49:36Z
- **Completed:** 2026-04-26T20:55:35Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Global QUEEN.md wisdom now flows into colony-prime worker context (cross-colony patterns visible to all workers)
- Protected section status ensures global queen wisdom survives budget trimming
- Seal wrappers now auto-promote instincts with confidence >= 0.8 to local QUEEN.md Wisdom section
- 3 new tests cover injection, graceful degradation, and hygiene

## Task Commits

Each task was committed atomically:

1. **Task 1: Inject global QUEEN.md into colony-prime and add protected status** - `8d216fad` (feat)
2. **Task 2: Add auto-promotion instructions to seal wrapper markdown** - `7d128a8c` (feat)

## Files Created/Modified
- `cmd/colony_prime_context.go` - Added global_queen_md section after hive wisdom, before user preferences
- `cmd/context_weighting.go` - Added global_queen_md to protectedSectionPolicy and sectionRelevanceScore
- `cmd/colony_prime_queen_test.go` - New test file with TestColonyPrimeIncludesGlobalQueen and TestColonyPrimeGlobalQueenSurvivesWithoutFile
- `cmd/queen_hygiene_test.go` - Added TestGlobalQueenWisdomHygiene for global QUEEN.md cleanliness
- `.claude/commands/ant/seal.md` - Added auto-promotion instructions for high-confidence instincts
- `.opencode/commands/ant/seal.md` - Identical auto-promotion instructions (wrapper parity)

## Decisions Made
- Global QUEEN.md section gets priority 5 (same as local queen wisdom) and relevance 0.75, placing it above hive wisdom (0.25) but below user preferences (0.80)
- Marked as protected so it always survives budget trimming -- cross-colony wisdom is too valuable to lose
- Seal auto-promotion targets confidence >= 0.8 (matching existing hive promotion threshold) and is non-blocking

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Pre-existing test failure in `TestColonyPrimeLargePheromonesTrimLowerPriority` (verified present on base commit, unrelated to changes)

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Global QUEEN.md now flows through colony-prime to all workers
- Seal lifecycle includes automatic instinct promotion
- Wisdom pipeline is fully connected: observe -> promote -> inject -> seal

---
*Phase: 57-queen-md-pipeline-fix*
*Completed: 2026-04-26*

## Self-Check: PASSED

All 7 files verified present. Both commits (8d216fad, 7d128a8c) verified in git log.
