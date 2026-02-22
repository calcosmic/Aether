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
**Plan**: 04 (complete)
**Status**: Phase 44 complete — all 4 plans finished (analyzer, UI, integration, tests)

**Progress**: 3/3 phases complete, v6.0 System Integration milestone complete

```
[██████████████████] 100% — v6.0 System Integration Complete
```

---

## Phase Status

| Phase | Status | Plans | Completed |
|-------|--------|-------|-----------|
| 42. Fix Update Bugs | ✓ Complete | 2/2 | 2026-02-22 |
| 43. Make Learning Flow | ✓ Complete | 3/3 | 2026-02-22 |
| 44. Suggest Pheromones | ✓ Complete | 4/4 | 2026-02-22 |

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
18. **Exclusion pattern boundaries** — Use path boundaries like `/.aether/` instead of `.aether` to avoid matching partial paths
19. **Bash 3.2 compatibility** — Use functions with case statements instead of associative arrays
20. **JSON output hygiene** — Redirect all UI text to stderr so stdout contains only valid JSON

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

None. Phase 44 complete. All 4 plans finished:
- Plan 01: suggest-analyze command with 6 pattern heuristics
- Plan 02: suggest-approve command with tick-to-approve UI
- Plan 03: Build flow integration at Step 4.2
- Plan 04: Integration tests (26 tests, all passing)

v6.0 System Integration milestone is complete.

---

## Session Continuity

**Last Action**: Completed Plan 44-04 — created comprehensive integration tests for pheromone suggestion system (26 tests, all passing)
**Next Action**: v6.0 System Integration milestone complete — all 3 phases finished
**Context Freshness**: Current
**Last Session**: 2026-02-22
**Stopped at**: Session resumed, project status reviewed
**Resume file**: N/A — milestone complete

## Completed Work Summary

### Phase 44: Suggest Pheromones (Complete)
- **Plan 01**: suggest-analyze command — analyzes codebase and suggests pheromones based on 6 heuristics
- **Plan 02**: suggest-approve command — tick-to-approve UI for reviewing suggestions
- **Plan 03**: Build flow integration — suggestions run at Step 4.2 of build command
- **Plan 04**: Integration tests — 26 comprehensive tests covering all functionality

### Bug Fixes During Plan 04
- Fixed exclusion pattern to use path boundaries (was matching temp dirs)
- Fixed bash 3.2 compatibility (replaced associative array with function)
- Fixed JSON output pollution (redirected UI to stderr)

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
