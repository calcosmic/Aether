# STATE — v6.0 System Integration

> Wire existing systems together — no new features, just integration

---

## Project Reference

**Core Value**: Aether prevents context rot across Claude Code sessions with self-managing colony that learns and guides users

**Current Focus**: Fix integration gaps in update system, learning pipeline, and pheromone suggestions.

**Milestone Goal**: Existing functions work together — no new features, just wiring.

---

## Current Position

**Phase**: 44-suggest-pheromones
**Plan**: 03 (complete)
**Status**: Plan 44-03 complete — pheromone suggestions integrated into build command flow

**Progress**: 2/3 phases complete, Phase 44 in progress

```
[████████████░░░░░░] 70% — v6.0 System Integration
```

---

## Phase Status

| Phase | Status | Plans | Completed |
|-------|--------|-------|-----------|
| 42. Fix Update Bugs | ✓ Complete | 2/2 | 2026-02-22 |
| 43. Make Learning Flow | ✓ Complete | 3/3 | 2026-02-22 |
| 44. Suggest Pheromones | In Progress | 3/TBD | 2026-02-22 |

---

## Accumulated Context

### Decisions Made

1. **Wire, don't build** — Use existing functions rather than creating new ones
2. **ERR trap handling** — Disable ERR trap during grep operations to handle "no matches" exit code 1
3. **jq for deduplication** — Use jq for JSON manipulation since bash 3.2 lacks associative arrays
4. **Phase 42 starts at 42** — Continuing from v5.0 which ended at Phase 41
5. **Three phases only** — Deep integration over broad coverage
6. **Atomic write pattern** — Use temp file + rename for all file copies to prevent corruption
7. **Counter accuracy** — Only increment counters when files are actually copied (not in dry-run)
8. **Trash safety** — Move removed files to `.aether/.trash/` instead of deleting
9. **Protected paths** — Never touch data/, dreams/, oracle/, midden/, or QUEEN.md
10. **One-at-a-time UI** — Present proposals individually with Approve/Reject/Skip actions
11. **Aligned thresholds** — All learning functions use consistent threshold values
12. **Failure type support** — Failure observations map to Patterns section in QUEEN.md
13. **Environment variable override** — AETHER_ROOT respects existing env var for testability
14. **Single-line METADATA** — QUEEN.md supports both single-line and multi-line METADATA formats
15. **Pheromone suggestion UI** — Tick-to-approve pattern with Approve/Reject/Skip/Dismiss All actions
16. **Non-interactive safety** — Auto-skip suggestions in CI/CD to prevent blocking
17. **Build flow timing** — Suggestions run after colony-prime (user sees current signals) but before swarm init (no worker delay)

### Open Questions

1. ~~Should pheromone suggestions run before or after colony-prime?~~ **RESOLVED**: After colony-prime (Step 4.2), so users see current signals before suggestions
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

None. Plan 44-02 complete. suggest-approve command implemented:
- Tick-to-approve UI with one-at-a-time display
- Approve/Reject/Skip/Dismiss All actions
- Flags: --yes, --dry-run, --no-suggest, --verbose
- Non-interactive mode detection (prevents blocking CI/CD)
- suggest-quick-dismiss helper for bulk dismissal

Ready for Plan 44-03: Build flow integration.

---

## Session Continuity

**Last Action**: Completed Plan 44-03 — integrated pheromone suggestions into build command as Step 4.2
**Next Action**: Phase 44 complete — all three plans finished (analyzer, UI, integration)
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
