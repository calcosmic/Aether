# STATE — v6.0 System Integration

> Wire existing systems together — no new features, just integration

---

## Project Reference

**Core Value**: Aether prevents context rot across Claude Code sessions with self-managing colony that learns and guides users

**Current Focus**: Fix integration gaps in update system, learning pipeline, and pheromone suggestions.

**Milestone Goal**: Existing functions work together — no new features, just wiring.

---

## Current Position

**Phase**: 43-make-learning-flow
**Plan**: 02 (complete)
**Status**: Plan 43-02 complete — FLOW-02 verified

**Progress**: 2/3 phases complete, 2/3 plans in Phase 43

```
[████████████░░░░░░] 67% — v6.0 System Integration
```

---

## Phase Status

| Phase | Status | Plans | Completed |
|-------|--------|-------|-----------|
| 42. Fix Update Bugs | ✓ Complete | 2/2 | 2026-02-22 |
| 43. Make Learning Flow | In Progress | 2/3 | 2026-02-22 |
| 44. Suggest Pheromones | Not planned | TBD | - |

---

## Accumulated Context

### Decisions Made

1. **Wire, don't build** — Use existing functions rather than creating new ones
2. **Phase 42 starts at 42** — Continuing from v5.0 which ended at Phase 41
3. **Three phases only** — Deep integration over broad coverage
4. **Atomic write pattern** — Use temp file + rename for all file copies to prevent corruption
5. **Counter accuracy** — Only increment counters when files are actually copied (not in dry-run)
6. **Trash safety** — Move removed files to `.aether/.trash/` instead of deleting
7. **Protected paths** — Never touch data/, dreams/, oracle/, midden/, or QUEEN.md
8. **One-at-a-time UI** — Present proposals individually with Approve/Reject/Skip actions
9. **Aligned thresholds** — All learning functions use consistent threshold values
10. **Failure type support** — Failure observations map to Patterns section in QUEEN.md

### Open Questions

1. Should pheromone suggestions run before or after colony-prime?
2. How many suggestions to show at once?
3. Should failed observations auto-promote or require approval?

### Decisions Made During Execution

- **Proposal UI**: One-at-a-time presentation with [A]pprove/[R]eject/[S]kip actions
- **Threshold values**: Uniform threshold=1 for all types except decree (threshold=0)
- **Failure handling**: Retry prompt on QUEEN.md write failure, keep pending if declined
- **Post-promotion**: Skipped proposals go to deferred, rejected are logged but not deferred

### Known Risks

1. **Update system touches many files** — Risk of breaking existing installs
2. **Learning pipeline has timing issues** — When exactly to check thresholds?
3. **Pheromone suggestions could be noisy** — Need to tune signal-to-noise

---

## Blockers

None. Plan 43-02 complete. FLOW-03 plan needed for Phase 43.

---

## Session Continuity

**Last Action**: Completed Plan 43-02 — wired learning pipeline with end-of-build promotion checking
**Next Action**: Run `/gsd:plan-phase 43` to create FLOW-03 plan for end-to-end testing
**Context Freshness**: Current

---

## Files

- PROJECT.md — Core value and milestone context
- ROADMAP.md — Phase structure and success criteria
- REQUIREMENTS.md — Detailed requirements with traceability
- STATE.md — This file
- research/CODEBASE-FLOW.md — Research on update/learning/QUEEN flow

---

*Created: 2026-02-22*
*Milestone: v6.0 System Integration*
