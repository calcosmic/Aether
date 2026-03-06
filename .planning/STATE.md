# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-06)

**Core value:** Workers automatically receive all relevant context -- the colony improves itself.
**Current focus:** Phase 4: Pheromone Auto-Emission (COMPLETE)

## Current Position

Phase: 4 of 5 (Pheromone Auto-Emission)
Plan: 2 of 2 in current phase (COMPLETE)
Status: Phase complete
Last activity: 2026-03-07 -- Completed 04-02 (auto-emission integration tests)

Progress: [█████████░] 90%

## Performance Metrics

**Velocity:**
- Total plans completed: 9
- Average duration: 2.5min
- Total execution time: 0.40 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-instinct-pipeline | 3 | 6min | 2min |
| 02-learnings-injection | 2 | 5min | 2.5min |
| 03-context-expansion | 2 | 7min | 3.5min |
| 04-pheromone-auto-emission | 2 | 6min | 3min |

**Recent Trend:**
- Last 5 plans: 03-01 (3min), 03-02 (4min), 04-01 (3min), 04-02 (3min)
- Trend: stable

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: 5 vertical pipeline phases, each delivering complete data flow from capture to injection
- [Roadmap]: Phase 1 starts with instinct pipeline (write instincts in continue, read in colony-prime)
- [01-01]: Confidence floor raised from 0.4 to 0.7 -- only validated patterns become instincts
- [01-01]: Error patterns get 0.8 confidence (higher than success 0.7) as stronger signals
- [01-01]: Success instincts capped at 2 per phase to prevent noise
- [01-02]: Same domain-grouped format for compact and non-compact modes
- [01-02]: No changes needed to build-context.md or build-wave.md -- existing pipeline chain works
- [01-03]: IEEE 754 floating point requires approximate comparison for confidence boost assertions
- [02-01]: Learnings placed between context-capsule and pheromone signals in prompt assembly order
- [02-01]: Inherited learnings sorted first (before numeric phases) for foundational visibility
- [02-01]: Compact mode: 5 claims max; non-compact: 15 claims max
- [02-02]: Extended setupTestColony helper with phaseLearnings and currentPhase rather than shared module
- [03-01]: Decisions placed after PHASE LEARNINGS and before BLOCKER WARNINGS in prompt assembly order
- [03-01]: BLOCKER WARNINGS uses [source: ...] prefix format distinct from REDIRECT [strength] prefix
- [03-01]: Decision cap: 5 non-compact, 3 compact; Blocker cap: 3 non-compact, 2 compact
- [03-02]: Blocker exclusion assertions target BLOCKER WARNINGS section boundary, not full prompt_section, to avoid context capsule false positives
- [04-01]: auto: source prefix namespace for auto-emitted pheromones (auto:decision, auto:error, auto:success)
- [04-01]: Decisions extracted from CONTEXT.md table (memory.decisions is always empty)
- [04-01]: midden-recent-failures used instead of errors.flagged_patterns for error detection
- [04-01]: Error threshold raised from 2+ to 3+ occurrences for higher confidence
- [04-01]: memory-capture resolution call retained from old 2.1b for error patterns
- [04-02]: Deduplication is caller responsibility -- pheromone-write always appends
- [04-02]: Midden test data uses entries[] key format matching midden-recent-failures subcommand
- [04-02]: Success criteria recurrence verified via JS grouping mirroring jq approach

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

## Session Continuity

Last session: 2026-03-07
Stopped at: Completed 04-02-PLAN.md (Phase 4 complete)
Resume file: .planning/phases/04-pheromone-auto-emission/04-02-SUMMARY.md
