---
gsd_state_version: 1.0
milestone: v1.17
milestone_name: Classic Restoration
status: planning
last_updated: "2026-05-13T15:30:00.000Z"
last_activity: 2026-05-13
progress:
  total_phases: 7
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-13)

**Core value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.
**Current focus:** Phase 112 — Foundation (event bridge, ceremony config, Node bump, boundary enforcement)

## Current Position

Phase: 112 of 118 (Foundation)
Plan: —
Status: Ready to plan
Last activity: 2026-05-13 — Roadmap created for v1.17 Classic Restoration (7 phases, 32 requirements mapped)

Progress: [░░░░░░░░░░░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**

- Total plans completed: 0 (v1.17)
- Average duration: —
- Total execution time: —

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| —     | —     | —     | —        |

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- v1.16 shipped 2026-05-13: Hybrid Runtime Boundary complete, TS host prototype proven, migration map defined
- v1.17 roadmap created 2026-05-13: 7 phases derived from 32 requirements, starting at Phase 112
- Build order: Event bridge first, then ceremony narrator, then real dispatch, then swarm, then orchestration, then Oracle, then parity tests
- Node engine bump to >=20 required for chokidar@5 and log-update@8 compatibility
- Three output modes required: json (machine), visual (TTY ANSI), markdown (plain text)

### Pending Todos

1. Plan Phase 112 (Foundation)
2. Capture Classic v5.4 baseline output for golden parity tests (before restoration work begins)
3. Add `requires_merge_back` flag to Go manifest schema (needed for Phase 114)
4. Add `escalation_level` field to completion file schema (needed for Phase 116)

### Blockers/Concerns

- Frankenstein state risk: TS host must never write to `.aether/data/` — boundary enforcement is critical
- Ceremony drift risk: shared YAML config must be consumed by Go, TS host, and wrappers
- Golden test brittleness: need careful design to compare hybrid output against Classic Bash without fragile string snapshots
- Simulation code shipping risk: ensure no simulated worker code remains in production paths after Phase 114

## Deferred Items

Items acknowledged and carried forward from previous milestones:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Tech debt | Phase 64.1 missing VERIFICATION.md | Non-critical | v1.10 |
| Tech debt | Phase 71 dispatch test covers 1/25 agent types | Non-critical | v1.11 |
| Tech debt | Phase 71 state-mutate flags registered but never read | Non-critical | v1.11 |
| v2 scope | State machine transitions (INTEL-06) | Deferred | v1.11 |
| v2 scope | Council system (INTEL-07) | Deferred | v1.11 |
| v2 scope | Curation ant pipeline (INTEL-08) | Deferred | v1.11 |
| v2 scope | Consolidation pipeline (INTEL-09) | Deferred | v1.11 |
| v2 scope | Queen autonomy levels (QUEEN-01) | Needs user testing | v1.14 |
| v2 scope | Cross-phase queen continuity (QUEEN-02) | Needs proven recovery loop first | v1.14 |
| v2 scope | Queen context budget config (QUEEN-03) | Needs empirical validation | v1.14 |
| v2 scope | Cross-phase coordination (QUEEN-04) | Architectural risk | v1.14 |

## Session Continuity

Last session: 2026-05-13T15:30:00.000Z
Stopped at: Roadmap created, Phase 112 ready to plan
Resume file: None
