---
gsd_state_version: 1.0
milestone: v1.25
milestone_name: Switch It On
status: executing
stopped_at: Phase 165 context gathered
last_updated: "2026-08-02T21:52:05.602Z"
last_activity: 2026-08-02 -- Phase 165 planning complete
progress:
  total_phases: 29
  completed_phases: 5
  total_plans: 45
  completed_plans: 39
  percent: 87
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-25)

**Core value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.
**Current focus:** Phase 164 — research-feeds-planning
**Milestone:** v1.25 Switch It On — ROADMAPPED (rebuilt twice, awaiting user approval)
**Previous milestone:** v1.24 Hybrid Architecture Salvage (shipped 2026-05-24, phases 152-159)
**Product version:** v1.0.45

## Current Position

Phase: 165
Plan: Not started
Status: Ready to execute
Last activity: 2026-08-02 -- Phase 165 planning complete

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**

- Total plans completed: 25 (v1.25)
- Average duration: — (v1.24 average: 8 min)
- Total execution time: 0 hours

**By Phase:**

*v1.25 not yet started*

## Accumulated Context

### Roadmap Evolution

- Phase 163.1 inserted after Phase 163: Wrapper-Runtime Completion Contract: completion-packet schema, batch validation, attempt recovery, verification honesty, reconcile escape hatch, brief file channel — from 2026-07-31 M4L usage diagnosis (URGENT)
- Phase 163.2 inserted after Phase 163.1: ts-host preflight configurability (M4L residual diagnosis item 2) (URGENT)

### Decisions

- **v1.25 was renamed "Working Again" → "Switch It On" and rebuilt twice.** First rebuild (58 requirements, 8 phases): six specialist review agents refuted the original draft's central diagnosis — see REQUIREMENTS.md's "Corrections carried forward" for what was actually false (playbooks were never fully orphaned, no four-mode Queen policy ever existed, caste colours already render, circuit breaker/Bayesian scoring/oracle RALF were never lost). Second correction (88 requirements, 12 phases, this state): that first rebuild over-corrected and dropped three categories no review agent had refuted — research feeding planning, lifecycle commands carrying method vs. protocol, and provable phase-fit caste selection — and compressed the rich-terminal-experience category too far. All restored
- **The TypeScript host must be KEPT, not deleted.** `.aether/ts-host/src/` holds the only playbook loader, the only confidence loop (`confidence-loop.ts`), and the only dashboard/swarm-display/narrator. Deleting it breaks `go build` via `//go:embed` and hard-fails `aether publish`/`aether integrity`. Only `control-ts/` is safe to delete. Keeping the TS host is also what makes Research Feeds Planning (Phase 164) *easier* — `confidence-loop.ts` (250 lines) is already the target-confidence loop that phase needs
- **The real disease is "never switched on," not "deleted."** Learning (`pkg/memory/pipeline.go`), cheap-model routing (`colony/policies/model-routing.yaml`), and colony health (`colony-vital-signs`) are all fully built with zero callers/readers. This reframes most of the milestone as wiring existing code, stated explicitly in each phase goal
- **12 phases (160-171), up from 8**: three new phases inserted after Context Reaches Workers — Research Feeds Planning (164), Core Lifecycle Commands (165), Full Colony On Demand (166) — plus "Your Eyes Back" split into two (168 Live Visibility, 169 Charter & Standards) because its 14 requirements were too broad for one independently-verifiable phase. Reclaim (now 170) absorbed 3 more requirements (swarm state quartet, remaining Class A orphans, `queen-seed-from-hive`) without needing a new phase
- **`build.md` ownership resolved (CMD-05)**: Phase 165 (Core Lifecycle Commands) is the sole structural owner. Phase 160 may only fix specific broken call arguments inside it (merges first — 165 depends on 160). Phase 168 (Live Visibility) may only append a next-step-guidance layer on top of Phase 165's output (165 must land first in practice, though its formal dependency is the shared Phase 160 foundation). No other phase restructures `build.md`
- **SEE-11 (re-init preserves colony state) is data safety, not ceremony** — it lives in Phase 169 (Charter & Standards) and its success criterion is required to be a dedicated automated test, not an observation
- RETIRE-02/03/04 remain folded into Phase 160 as constraints/guardrails, not their own phase
- Every phase 161-170 depends on Phase 160 (shared foundation), plus real content dependencies where they exist: 164→163 (research rides the context plumbing), 165→163+164 (documents both), 170→162 (RECLAIM-09 ties to the hive-default decision). Phase 171 depends on all of 160-170
- Several requirements remain explicitly decision-shaped, not build-shaped: SEE-03, SEE-07, LEARN-03, LEARN-04, COLONY-05 (new — Dream as 28th caste or not), PROOF-04
- `colony/agents/` (27 caste YAMLs) and `colony/policies/` (10 policy files, including `model-routing.yaml`) already exist; Phase 161 wires readers, Phase 166 gives `colony/agents/*.yaml` its first Go reader
- TYPED-01/02 (migration backfill) MUST precede TYPED-03 (making `mode` required) within Phase 167 — `mode` is `omitempty` today and requiring it before backfilling would brick every existing colony

### Pending Todos

- [ ] Get user approval on the corrected v1.25 roadmap (12 phases, 88 requirements)
- [ ] Plan Phase 160: Fail Loudly
- [ ] Plan Phase 161: Cheap Models By Design
- [ ] Plan Phase 162: Switch On Learning
- [ ] Plan Phase 163: Context Reaches Workers
- [ ] Plan Phase 164: Research Feeds Planning
- [ ] Plan Phase 165: Core Lifecycle Commands (owns build.md structurally)
- [ ] Plan Phase 166: Full Colony On Demand
- [ ] Plan Phase 167: Typed Control (migrate mode before requiring it)
- [ ] Plan Phase 168: Your Eyes Back — Live Visibility
- [ ] Plan Phase 169: Your Eyes Back — Charter & Standards
- [ ] Plan Phase 170: Reclaim The Unreachable
- [ ] Plan Phase 171: Prove It

### Blockers/Concerns

- Roadmap has been rebuilt/corrected twice and is unapproved — do not start Phase 160 planning/build until the user signs off on the 12-phase, 88-requirement structure
- Any prior mental model referencing the 8-phase "Switch It On" draft (SEE as one 7-requirement phase, no Research/CMD/Colony phases) should be treated as superseded by this correction
- Any prior mental model referencing the original discarded "Working Again" draft (deleting `.aether/ts-host/`, a four-mode Queen policy, "orphaned playbooks" as sole root cause) remains superseded

## Deferred Items

Carried from v1.23, deferred again in v1.25 (see REQUIREMENTS.md "Deferred"):

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Catalog | CATALOG-01, CATALOG-02 | Deferred | v1.25 roadmap |
| Test coverage | TEST-01, TEST-02 | Deferred | v1.25 roadmap |
| Workflow | WORKFLOW-01 … WORKFLOW-09 | Deferred | v1.25 roadmap |
| Runtime | RUNTIME-01, RUNTIME-02 | Deferred | v1.25 roadmap |

## Session Continuity

Last session: 2026-08-02T21:10:21.138Z
Stopped at: Phase 165 context gathered
Resume file: .planning/phases/165-core-lifecycle-commands/165-CONTEXT.md
