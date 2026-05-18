---
gsd_state_version: 1.0
milestone: v1.21
milestone_name: Live Colony
status: planning
last_updated: "2026-05-18T12:00:00.000Z"
progress:
  total_phases: 0
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-18)

**Core value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.
**Current focus:** Milestone v1.21 — Live Colony

## Current Position

Phase: Not started (defining requirements)
Plan: —
Status: Defining requirements
Last activity: 2026-05-18 — Milestone v1.21 Live Colony started

## Known Blockers

None

## Next Actions

1. Define requirements for v1.21
2. Create roadmap with phases starting at 136
3. Plan Phase 136

## Key Decisions (Carried Forward)

- Wrappers are "host-assisted orchestrators"; they call `aether host` for manifests
- Go remains sole authority for state mutation and finalizers
- Oracle state uses atomic Go-owned storage with interrupt recovery
- Simulation is gated behind `--simulate` flag
