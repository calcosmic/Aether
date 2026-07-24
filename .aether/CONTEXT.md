# Aether Colony — Current Context

> **This document is the colony's memory. If context collapses, read this file first.**

---

## System Status

| Field | Value |
|-------|-------|
| **Last Updated** | 2026-07-20T20:38:22Z |
| **Current Phase** | 1 |
| **Phase Name** | Lock Claude-First Planning Contract |
| **Phase Status** | ready |
| **Milestone** | First Mound |
| **Colony Status** | READY |
| **Colony Mode** | colony |
| **Safe to Clear?** | YES — Colony paused, safe to clear context |

---

## Current Goal

Dogfood Aether on itself with real Claude-backed iterative planning to identify the necessary fixes to make the Claude-first planning restoration production-ready

---

## What's In Progress

Paused at phase 1

---

## Active Constraints (REDIRECT Signals)

| Constraint | Source | Date Set |
|------------|--------|----------|
| How tightly should this work reuse existing contracts and integrations: Reuse existing contracts where possible. Add only narrow adapters when a current cont... | pheromone | active |
| Which existing surface should own the first implementation slice: Go runtime planning/finalizer contracts and Claude command wrappers own the first implement... | pheromone | active |

---

## Active Pheromones

*None active*

---

## Open Blockers

- Discuss reused stale resolved clarifications from previous colony: new reliability audit shows settled because old Orchestrator Mode decisions remain in pend...
- Plan-only created Orchestrator boundary question pd_1778418330681552000, but aether discuss did not surface it and instead reported no questions with stale r...
- Planning Gatekeeper completed review but could not write /tmp finalizer JSON because role write boundary conflicts with plan-finalize worker artifact contract
- plan-finalize refused new colony plan because COLONY_STATE.json still contains existing plan phases; requires --refresh despite fresh init
- Build plan-only created hard Orchestrator boundary question pd_1778419838148615000, but aether discuss did not surface it and reported no outstanding questions
- Build Tracker Hunt-33 completed root-cause review but could not write /tmp build-finalize JSON because role write boundary conflicts with wrapper artifact co...
- Twist-44 was closed after stalling without writing /tmp/aether-build-1-worker-Twist-44.json after a parent-side malformed legacy timestamp hardening update. ...
- Running AETHER_OUTPUT_MODE=visual aether continue --skip-watchers --verification-depth standard spawned Probe Excavat-92, which heartbeated until worker time...
- Build plan-only created hard Orchestrator boundary question pd_1778424856613270000 for Phase 2, but AETHER_OUTPUT_MODE=visual aether discuss reported 0 quest...
- test
- test
- test
- test
- test
- test

---

## Tasks For Phase 1 — Lock Claude-First Planning Contract

- [ ] Add a failing manifest contract test in cmd/codex_plan_test.go proving aether plan --plan-only emits planning_run_id, iteration, target and max controls, dispatch_mode plan-only, requires_finalizer true, and exactly two dispatches ordered Scout then Route-Setter.
- [ ] Add a failing finalizer proof test in cmd/codex_plan_finalize_test.go proving plan-finalize rejects completions missing Scout evidence, Route-Setter phase_plan, matching planning_run_id, or matching iteration.
- [ ] Add a failing Claude wrapper contract test in cmd/plan_wrapper_ceremony_test.go proving .claude/commands/ant/plan.md uses aether host plan, dispatches Scout before Route-Setter, calls aether plan-finalize, and forbids post-worker synthetic planning.
- [ ] Update cmd/codex_plan.go so the plan manifest includes dispatch_contract.host_plan_behavior set to manifest_only, dispatch_contract.worker_order set to scout and route-setter, and dispatch_contract.finalizer set to aether plan-finalize --completion-file <file>.
- [ ] Update cmd/codex_plan_finalize.go to enforce exactly one Scout result followed by one Route-Setter result, non-empty Scout evidence, non-empty Route-Setter phase plan, and matching planning iteration identity.
- [ ] Update .aether/commands/plan.yaml and .claude/commands/ant/plan.md to state that aether host plan returns a manifest only, Claude performs live visible planning workers, and closeout happens only after plan-finalize.
- [ ] Run focused Go verification for the Phase 1 planning contract.

---

## Recent Decisions

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| — | No recorded decisions | — | — |

---

## Recent Activity (Last 5 Events)

- 2026-07-20T19:35:08Z|planning_scout|plan|Scout summarized surveyed repo context
- 2026-07-20T19:35:08Z|plan_generated|plan|Generated 5 phases with 81% confidence; planning loop stopped: stalled

---

## Next Steps

1. Run `aether resume`
2. Run `aether phase --number 1` to inspect the tracked phase details
3. Run `aether resume-colony` after a context clear if you want the full recovery view

---

## If Context Collapses

1. Run `aether resume` for the quick dashboard restore
2. Run `aether resume-colony` for the full handoff and task view
3. Read `.aether/HANDOFF.md` if a richer session summary was persisted

### Active Todos
- Add a failing manifest contract test in cmd/codex_plan_test.go proving aether plan --plan-only emits planning_run_id, iteration, target and max controls, dispatch_mode plan-only, requires_finalizer true, and exactly two dispatches ordered Scout then Route-Setter.
- Add a failing finalizer proof test in cmd/codex_plan_finalize_test.go proving plan-finalize rejects completions missing Scout evidence, Route-Setter phase_plan, matching planning_run_id, or matching iteration.
- Add a failing Claude wrapper contract test in cmd/plan_wrapper_ceremony_test.go proving .claude/commands/ant/plan.md uses aether host plan, dispatches Scout before Route-Setter, calls aether plan-finalize, and forbids post-worker synthetic planning.
- Update cmd/codex_plan.go so the plan manifest includes dispatch_contract.host_plan_behavior set to manifest_only, dispatch_contract.worker_order set to scout and route-setter, and dispatch_contract.finalizer set to aether plan-finalize --completion-file <file>.
- Update cmd/codex_plan_finalize.go to enforce exactly one Scout result followed by one Route-Setter result, non-empty Scout evidence, non-empty Route-Setter phase plan, and matching planning iteration identity.
- Update .aether/commands/plan.yaml and .claude/commands/ant/plan.md to state that aether host plan returns a manifest only, Claude performs live visible planning workers, and closeout happens only after plan-finalize.
- Run focused Go verification for the Phase 1 planning contract.
