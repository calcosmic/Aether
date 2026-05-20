---
gsd_state_version: 1.0
milestone: v1.23
milestone_name: Daily Driver Reliability
status: planning
last_updated: "2026-05-20T00:00:00Z"
last_activity: 2026-05-20 -- Milestone v1.23 started
progress:
  total_phases: 0
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-20)

**Core value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.
**Current focus:** v1.23 Daily Driver Reliability

## Current Position

Phase: Not started (defining requirements)
Plan: -
Status: Defining requirements
Last activity: 2026-05-20 -- Milestone v1.23 started

## Known Blockers

(None)

## Next Actions

1. Define requirements for v1.23
2. Create roadmap with phase breakdown

## Key Decisions (Carried Forward)

- Wrappers are "host-assisted orchestrators"; they call `aether host` for manifests
- Go remains sole authority for state mutation and finalizers
- Oracle state uses atomic Go-owned storage with interrupt recovery
- Simulation is gated behind `--simulate` flag
- worker-dispatch.ts simulateWorkers check already defaults to real dispatch; no source change needed
- Lifecycle smoke harness (lifecycle.ts) stays simulate-only; production uses dedicated host commands
- Build/plan/continue commands use "dispatched" runner type with ceremony rendering and Go finalizers
- PlaybookLoader resolves candidates: absolute -> root+path -> root+playbooks-dir -> hub-system -> hub-root -> bare (mirrors Go)
- YAML orchestration blocks removed from build.yaml and plan.yaml; codex_orchestration preserved (CEREMONY-04)
- Playbook context is document-injection, not parsed steps -- host.ts loads playbooks and appends to task_brief (CEREMONY-06)
- M4L regression test proves end-to-end survey-to-grounding pipeline composition (Phases 141+142)

## Operator Next Steps

- Define requirements and create roadmap
