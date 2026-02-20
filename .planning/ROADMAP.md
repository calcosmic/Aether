# Roadmap: Aether

## Milestones

- ✅ **v1.0 Repair & Stabilization** — Phases 1-9 (shipped 2026-02-18)
- ✅ **v1.1 Colony Polish & Identity** — Phases 10-13 (shipped 2026-02-18)
- ✅ **v1.2 Hardening & Reliability** — Phases 14-19 (shipped 2026-02-19)

## Phases

<details>
<summary>✅ v1.0 Repair & Stabilization (Phases 1-9) — SHIPPED 2026-02-18</summary>

- [x] Phase 1: Diagnostic (3 plans) — 120 tests, 66% pass, 9 critical failures identified
- [x] Phase 2: Core Infrastructure (5 plans) — fixed command foundations
- [x] Phase 3: Visual Experience (2 plans) — swarm display, emoji castes, colors
- [x] Phase 4: Context Persistence (2 plans) — drift detection, rich resume dashboard
- [x] Phase 5: Pheromone System (3 plans) — FOCUS/REDIRECT/FEEDBACK, auto-injection, eternal memory
- [x] Phase 6: Colony Lifecycle (3 plans) — seal ceremony, entomb archival, tunnels browser
- [x] Phase 7: Advanced Workers (3 plans) — oracle, chaos, archaeology, dream, interpret synced
- [x] Phase 8: XML Integration (4 plans) — pheromone/wisdom/registry XML, seal export, entomb hard-stop
- [x] Phase 9: Polish & Verify (4 plans) — 46/46 requirements PASS, full e2e test suite

**46 requirements verified. Full details: `.planning/milestones/v1.0-ROADMAP.md`**

</details>

<details>
<summary>✅ v1.1 Colony Polish & Identity (Phases 10-13) — SHIPPED 2026-02-18</summary>

- [x] Phase 10: Noise Reduction (4 plans) — bash descriptions on 34 commands, ~40% header reduction, version cache
- [x] Phase 11: Visual Identity (6 plans) — ━━━━ banners, progress bars, Next Up blocks, canonical caste-system.md
- [x] Phase 12: Build Progress (2 plans) — spawn announcements, completion lines, BUILD SUMMARY, tmux gating
- [x] Phase 13: Distribution Reliability (1 plan) — .update-pending sentinel, atomic recovery, version detection fix

**14/15 requirements satisfied. Full details: `.planning/milestones/v1.1-ROADMAP.md`**

</details>

<details>
<summary>✅ v1.2 Hardening & Reliability (Phases 14-19) — SHIPPED 2026-02-19</summary>

- [x] Phase 14: Foundation Safety (1 plan) — json_err fallback fix, template path resolution
- [x] Phase 15: Distribution Chain (3 plans) — hub source dir fix, dead duplicates removed, npm deprecation
- [x] Phase 16: Lock Lifecycle Hardening (3 plans) — uniform trap pattern, stale lock prompts, atomic-write fix
- [x] Phase 17: Error Code Standardization (3 plans) — 49 bare-string calls converted to E_* constants, error-codes.md
- [x] Phase 18: Reliability & Architecture Gaps (4 plans) — startup ordering, EXIT trap, spawn-tree rotation, queen-read validation
- [x] Phase 19: Milestone Polish (4 plans) — audit gap closure, 21 new AVA tests, 446 tests passing

**24/24 requirements satisfied. Full details: `.planning/milestones/v1.2-ROADMAP.md`**

</details>

## v1.3 The Great Restructuring (Phases 20-25)

**Goal:** Make Aether more reliable — extract templates, clean agents, simplify pipeline, define failure handling.

**24 requirements across 6 phases:**

- [x] Phase 20: Distribution Simplification (PIPE-01 through PIPE-03) — eliminate runtime/ staging, simplify build pipeline (completed 2026-02-19)
  Plans:
  - [ ] 20-01-PLAN.md — Core pipeline restructure (packaging, cli.js, update-transaction.js, delete runtime/)
  - [ ] 20-02-PLAN.md — Pre-commit hook, aether-utils.sh cleanup, bash test updates
  - [ ] 20-03-PLAN.md — Documentation cleanup (CLAUDE.md, OPENCODE.md, rules, CHANGELOG)
- [x] Phase 21: Template Foundation (TMPL-01 through TMPL-06) — extract 5 critical templates, add to distribution (completed 2026-02-19)
  Plans:
  - [ ] 21-01-PLAN.md — Create JSON templates (colony-state, constraints) and jq reset filter
  - [ ] 21-02-PLAN.md — Create markdown templates (crowned-anthill, handoff)
  - [ ] 21-03-PLAN.md — Register templates in validate-package.sh, verify distribution
- [x] Phase 22: Agent Boilerplate Cleanup (AGENT-01 through AGENT-04) — strip redundant sections from all 25 agents (completed 2026-02-19)
  **Plans:** 3 plans
  Plans:
  - [ ] 22-01-PLAN.md — Strip boilerplate from Core 5 and Development 4 agents (9 agents)
  - [ ] 22-02-PLAN.md — Strip boilerplate from Knowledge 4 and Quality 4 agents (8 agents)
  - [ ] 22-03-PLAN.md — Strip boilerplate from Special 3 agents and update Surveyor 4 descriptions (7 agents)
- [x] Phase 23: Agent Resilience (RESIL-01 through RESIL-03) — add failure modes, success criteria, read-only declarations (completed 2026-02-19)
  **Plans:** 3 plans
  Plans:
  - [ ] 23-01-PLAN.md — Add resilience sections to 7 HIGH-risk agents (queen, builder, watcher, weaver, route-setter, ambassador, tracker)
  - [ ] 23-02-PLAN.md — Add resilience sections to 17 MEDIUM + LOW-risk agents and surveyors
  - [ ] 23-03-PLAN.md — Add resilience blocks to 6 high-risk slash commands (init, build, lay-eggs, seal, entomb, colonize)
- [x] Phase 24: Template Integration (WIRE-01 through WIRE-05) — wire commands to read templates instead of inline structures (completed 2026-02-20)
  **Plans:** 3 plans
  Plans:
  - [ ] 24-01-PLAN.md — Wire init.md (both platforms) to colony-state and constraints templates
  - [ ] 24-02-PLAN.md — Wire seal.md and entomb.md (both platforms) to ceremony and handoff templates
  - [ ] 24-03-PLAN.md — Create build HANDOFF templates and wire build.md (both platforms)
- [ ] Phase 25: Queen Coordination (COORD-01 through COORD-04) — escalation chain, workflow patterns, agent merges
  **Plans:** 3 plans
  Plans:
  - [ ] 25-01-PLAN.md — Queen rewrite: 4-tier escalation chain + 6 named workflow patterns + build/status wiring
  - [ ] 25-02-PLAN.md — Agent merges: Architect→Keeper, Guardian→Auditor, delete old files, update spawn refs
  - [ ] 25-03-PLAN.md — Count cleanup: update all "25 agents" refs to "23", annotate caste-system.md and workers.md

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Diagnostic | v1.0 | 3/3 | Complete | 2026-02-18 |
| 2. Core Infrastructure | v1.0 | 5/5 | Complete | 2026-02-18 |
| 3. Visual Experience | v1.0 | 2/2 | Complete | 2026-02-18 |
| 4. Context Persistence | v1.0 | 2/2 | Complete | 2026-02-18 |
| 5. Pheromone System | v1.0 | 3/3 | Complete | 2026-02-18 |
| 6. Colony Lifecycle | v1.0 | 3/3 | Complete | 2026-02-18 |
| 7. Advanced Workers | v1.0 | 3/3 | Complete | 2026-02-18 |
| 8. XML Integration | v1.0 | 4/4 | Complete | 2026-02-18 |
| 9. Polish & Verify | v1.0 | 4/4 | Complete | 2026-02-18 |
| 10. Noise Reduction | v1.1 | 4/4 | Complete | 2026-02-18 |
| 11. Visual Identity | v1.1 | 6/6 | Complete | 2026-02-18 |
| 12. Build Progress | v1.1 | 2/2 | Complete | 2026-02-18 |
| 13. Distribution Reliability | v1.1 | 1/1 | Complete | 2026-02-18 |
| 14. Foundation Safety | v1.2 | 1/1 | Complete | 2026-02-18 |
| 15. Distribution Chain | v1.2 | 3/3 | Complete | 2026-02-18 |
| 16. Lock Lifecycle Hardening | v1.2 | 3/3 | Complete | 2026-02-19 |
| 17. Error Code Standardization | v1.2 | 3/3 | Complete | 2026-02-19 |
| 18. Reliability & Architecture Gaps | v1.2 | 4/4 | Complete | 2026-02-19 |
| 19. Milestone Polish | v1.2 | 4/4 | Complete | 2026-02-19 |
| 20. Distribution Simplification | v1.3 | Complete    | 2026-02-19 | — |
| 21. Template Foundation | v1.3 | Complete    | 2026-02-19 | — |
| 22. Agent Boilerplate Cleanup | v1.3 | Complete    | 2026-02-19 | — |
| 23. Agent Resilience | v1.3 | Complete    | 2026-02-19 | — |
| 24. Template Integration | v1.3 | Complete    | 2026-02-20 | — |
| 25. Queen Coordination | v1.3 | 0/3 | Pending | — |

---

*Roadmap created: 2026-02-17*
*v1.0 shipped: 2026-02-18*
*v1.1 shipped: 2026-02-18*
*v1.2 shipped: 2026-02-19*
*v1.3 roadmap created: 2026-02-19*
