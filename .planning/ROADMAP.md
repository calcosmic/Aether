# Roadmap: Aether

## Milestones

- ✅ **v1.0 Repair & Stabilization** — Phases 1-9 (shipped 2026-02-18)
- ✅ **v1.1 Colony Polish & Identity** — Phases 10-13 (shipped 2026-02-18)
- ✅ **v1.2 Hardening & Reliability** — Phases 14-19 (shipped 2026-02-19)
- ✅ **v1.3 The Great Restructuring** — Phases 20-25 (shipped 2026-02-20)
- ✅ **v1.4 Deep Cleanup (partial)** — Phase 26 (shipped 2026-02-20)
- ✅ **v2.0 Worker Emergence** — Phases 27-31 (shipped 2026-02-20)
- ✅ **v3.0 Wisdom & Pheromone Evolution** — Phases 32-35 (shipped 2026-02-21)
- 🔄 **v4.0 Colony Context Enhancement** — Phases 36-39 (in progress)

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

<details>
<summary>✅ v1.3 The Great Restructuring (Phases 20-25) — SHIPPED 2026-02-20</summary>

- [x] Phase 20: Distribution Simplification — eliminate runtime/ staging, simplify build pipeline
- [x] Phase 21: Template Foundation — extract 5 critical templates, add to distribution
- [x] Phase 22: Agent Boilerplate Cleanup — strip redundant sections from all 25 agents
- [x] Phase 23: Agent Resilience — add failure modes, success criteria, read-only declarations
- [x] Phase 24: Template Integration — wire commands to read templates instead of inline structures
- [x] Phase 25: Queen Coordination — escalation chain, workflow patterns, agent merges

**24/24 requirements satisfied.**

</details>

<details>
<summary>✅ v1.4 Deep Cleanup (Phase 26) — SHIPPED 2026-02-20</summary>

- [x] Phase 26: Audit & Delete Dead Files — full repo audit, dead duplicates removed, README updated

**10/10 requirements satisfied (phases 27-30 absorbed into v2.0).**

</details>

<details>
<summary>✅ v2.0 Worker Emergence (Phases 27-31) — SHIPPED 2026-02-20</summary>

- [x] Phase 27: Distribution Infrastructure + First Core Agents (4 plans) — Builder, Watcher, hub sync proven
- [x] Phase 28: Orchestration Layer + Surveyor Variants (3 plans) — Queen, Scout, Route-Setter, 4 Surveyors
- [x] Phase 29: Specialist Agents + Agent Tests (3 plans) — Keeper, Tracker, Probe, Weaver, Auditor + 6 AVA tests
- [x] Phase 30: Niche Agents (3 plans) — Chaos, Archaeologist, Gatekeeper, Includer, Measurer, Sage, Ambassador, Chronicler
- [x] Phase 31: Integration Verification + Cleanup (3 plans) — 22 agents verified, bash bug fixed, docs curated

**49 requirements verified. Full details: `.planning/milestones/v2.0-ROADMAP.md`**

</details>

<details>
<summary>✅ v3.0 Wisdom & Pheromone Evolution (Phases 32-35) — SHIPPED 2026-02-21</summary>

- [x] Phase 32: Wire QUEEN.md into Commands (3 plans) — colony-prime(), two-level loading, init.md integration
- [x] Phase 33: Add Promotion Proposals (3 plans) — learning-observe, learning-check-promotion, continue.md display
- [x] Phase 34: Add User Approval UX (3 plans) — tick-to-approve, threshold enforcement, undo support
- [x] Phase 35: Lifecycle Integration (2 plans) — seal.md wisdom review, entomb.md final wisdom check

**25 requirements verified.**

</details>

<details>
<summary>🔄 v4.0 Memory Pipeline (Phases 36-37) — IN PROGRESS</summary>

- [x] Phase 36: Memory Capture — Learnings on continue, failures on build, lower promotion threshold (completed 2026-02-21)
- [x] Phase 37: Changelog + Visibility — Continuous updates, rich resume/status (completed 2026-02-21)

**6 requirements mapped. Goal: Make the memory pipeline actually work.**

</details>

## Phase Details

### Phase 36: Memory Capture

**Goal:** Wire the existing memory systems so they actually capture and store learnings

**Depends on:** Phase 35 (v3.0 complete)

**Requirements:** MEM-01, MEM-02, MEM-03

**Success Criteria** (what must be TRUE):
1. `/ant:continue` asks "What did you learn this phase?" — approved answers write to QUEEN.md
2. `/ant:build` logs failed approaches to midden/ AND calls learning-observe with type=failure
3. Promotion threshold lowered to 1 observation + user approval (not 5)
4. QUEEN.md accumulates real wisdom after each phase

**Plans:** 3/3 plans complete
- [ ] 36-01-PLAN.md — Lower promotion threshold to 1 observation (MEM-03)
- [ ] 36-02-PLAN.md — Integrate learning approval into continue.md (MEM-01)
- [ ] 36-03-PLAN.md — Add failure logging to build.md with midden storage (MEM-02)

**Why this matters:** Right now QUEEN.md stays empty. After this phase, it grows with every completed phase.

---

### Phase 37: Changelog + Visibility

**Goal:** Continuous changelog updates and visible memory health

**Depends on:** Phase 36 (memory capture working)

**Requirements:** LOG-01, VIS-01, VIS-02

**Success Criteria** (what must be TRUE):
1. Workers update CHANGELOG.md during work — decisions, files changed, what worked/didn't
2. `/ant:resume` shows recent learnings, failed approaches, and accumulated wisdom
3. `/ant:status` shows memory health — wisdom count, recent learnings, what's being remembered
4. Human can see colony memory at a glance

**Plans:** 3/3 plans complete
- [ ] 37-01-PLAN.md — Memory metrics utilities (memory-metrics, midden-recent-failures, resume-dashboard) (VIS-02)
- [ ] 37-02-PLAN.md — Changelog system (changelog-append, changelog-collect-plan-data) (LOG-01)
- [ ] 37-03-PLAN.md — Resume/status integration with drill-down command (VIS-01, VIS-02)

**Why this matters:** Two weeks later, you can see what was tried and what didn't work.

---

## Requirement Coverage

| Requirement | Phase | Description |
|-------------|-------|-------------|
| MEM-01 | 36 | /ant:continue asks for learnings → writes to QUEEN.md | Complete    | 2026-02-21 | 36 | /ant:build logs failures to midden + learning-observe |
| MEM-03 | 36 | Lower promotion threshold to 1 + user approval |
| LOG-01 | 37 | Workers update CHANGELOG.md during work | Complete    | 2026-02-21 | 37 | /ant:resume shows learnings, failures, wisdom |
| VIS-02 | 37 | /ant:status shows memory health |

**Coverage:** 6/6 requirements mapped ✓
- v1 requirements (P1 Core): 4/4
- v2 requirements (P2 Enhancements): 2/2

---

## Architecture Alignment

**Principle:** Wire the existing memory systems together. No new infrastructure.

| Component | v4.0 Enhancement |
|-----------|------------------|
| `/ant:continue` | Asks "What did you learn?" → writes to QUEEN.md |
| `/ant/build` | Logs failures to midden/ + learning-observe |
| `learning-observe` | Called automatically on failure (not just manually) |
| `queen-promote` | Threshold 1 + approval (not 5) |
| `QUEEN.md` | Actually gets populated now |
| `colony-prime` | Reads populated QUEEN.md (already works) |
| `CHANGELOG.md` | New — continuously updated by workers |

---

## Out of Scope

| Feature | Reason |
|---------|--------|
| New pheromone types | Existing signals work |
| Decay mechanisms | Explicit logging is simpler |
| NEST.md, TRAILS/, BROOD/ | Over-engineering |
| Complex documentation | QUEEN.md + CHANGELOG.md is enough |

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
| 15. Distribution Chain | v1.2 | 3/3 | Complete | 2026-02-19 |
| 16. Lock Lifecycle Hardening | v1.2 | 3/3 | Complete | 2026-02-19 |
| 17. Error Code Standardization | v1.2 | 3/3 | Complete | 2026-02-19 |
| 18. Reliability & Architecture Gaps | v1.2 | 4/4 | Complete | 2026-02-19 |
| 19. Milestone Polish | v1.2 | 4/4 | Complete | 2026-02-19 |
| 20. Distribution Simplification | v1.3 | Complete | 2026-02-19 | — |
| 21. Template Foundation | v1.3 | Complete | 2026-02-19 | — |
| 22. Agent Boilerplate Cleanup | v1.3 | Complete | 2026-02-19 | — |
| 23. Agent Resilience | v1.3 | Complete | 2026-02-20 | — |
| 24. Template Integration | v1.3 | Complete | 2026-02-20 | — |
| 25. Queen Coordination | v1.3 | Complete | 2026-02-20 | — |
| 26. Audit & Delete Dead Files | v1.4 | Complete | 2026-02-20 | — |
| 27. Distribution Infrastructure + First Core Agents | v2.0 | 4/4 | Complete | 2026-02-20 |
| 28. Orchestration Layer + Surveyor Variants | v2.0 | Complete | 2026-02-20 | - |
| 29. Specialist Agents + Agent Tests | v2.0 | Complete | 2026-02-20 | - |
| 30. Niche Agents | v2.0 | Complete | 2026-02-20 | - |
| 31. Integration Verification + Cleanup | v2.0 | 3/3 | Complete | 2026-02-20 |
| 32. Wire QUEEN.md into Commands | v3.0 | Complete    | 2026-02-20 | — |
| 33. Add Promotion Proposals | v3.0 | Complete    | 2026-02-20 | — |
| 34. Add User Approval UX | v3.0 | Complete    | 2026-02-20 | — |
| 35. Lifecycle Integration | v3.0 | Complete    | 2026-02-21 | — |
| 36. Memory Capture | v4.0 | 3/3 | Complete | 2026-02-21 |
| 37. Changelog + Visibility | v4.0 | 0/3 | Planned | — |

---

*Roadmap created: 2026-02-17*
*v1.0 shipped: 2026-02-18*
*v1.1 shipped: 2026-02-18*
*v1.2 shipped: 2026-02-19*
*v1.3 shipped: 2026-02-20*
*v1.4 (partial) shipped: 2026-02-20*
*v2.0 shipped: 2026-02-20*
*v3.0 shipped: 2026-02-21*
*v4.0 defined: 2026-02-21 (memory pipeline)*
