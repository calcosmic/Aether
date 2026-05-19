---
gsd_state_version: 1.0
milestone: v1.22
milestone_name: Grounded Planning + Ceremony Restore
status: milestone_archived
last_updated: "2026-05-19T12:00:00Z"
last_activity: 2026-05-19 -- v1.22 milestone archived, product v1.0.40 published
progress:
  total_phases: 4
  completed_phases: 4
  total_plans: 8
  completed_plans: 8
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-19)

**Core value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.
**Current focus:** Planning next milestone — run `/gsd-new-milestone` to start

## Current Position

Phase: N/A
Plan: N/A
Status: Milestone archived
Last activity: 2026-05-19

## Known Blockers

- Go `spawn-log`/`spawn-complete` may need minor adjustments to accept children not in the original manifest
- Confidence metric definition for builds is unclear -- multiple gate results need aggregation into a single score

## Next Actions

1. Start next milestone with `/gsd-new-milestone`

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

- Start the next milestone with /gsd-new-milestone
