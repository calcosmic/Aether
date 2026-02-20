# Roadmap: Aether

## Milestones

- ✅ **v1.0 Repair & Stabilization** — Phases 1-9 (shipped 2026-02-18)
- ✅ **v1.1 Colony Polish & Identity** — Phases 10-13 (shipped 2026-02-18)
- ✅ **v1.2 Hardening & Reliability** — Phases 14-19 (shipped 2026-02-19)
- ✅ **v1.3 The Great Restructuring** — Phases 20-25 (shipped 2026-02-20)
- ✅ **v1.4 Deep Cleanup (partial)** — Phase 26 (shipped 2026-02-20)
- 🚧 **v2.0 Worker Emergence** — Phases 27-31 (in progress)

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

## v2.0 Worker Emergence (Phases 27-31)

**Milestone Goal:** Create real Claude Code subagents from the 22 OpenCode agent definitions. Every ant worker type becomes a first-class Claude Code subagent that resolves from the Task tool instead of the fallback path. Agents distribute through the hub to all target repos. Remaining v1.4 cleanup absorbed.

**49 requirements across 5 phases.**

- [ ] **Phase 27: Distribution Infrastructure + First Core Agents** - Prove the end-to-end chain works: packaging, hub sync, target delivery, with Builder and Watcher as the first two shipped agents
- [ ] **Phase 28: Orchestration Layer + Surveyor Variants** - Queen, Scout, Route-Setter, and all 4 Surveyors — the full orchestration and codebase-context capability
- [ ] **Phase 29: Specialist Agents + Agent Tests** - Keeper, Tracker, Probe, Weaver, Auditor plus the full AVA test suite for agent quality
- [ ] **Phase 30: Niche Agents** - All 8 niche castes completing the full 22-agent roster
- [ ] **Phase 31: Integration Verification + Cleanup** - End-to-end verification, docs cleanup, bash bug fix

## Phase Details

### Phase 27: Distribution Infrastructure + First Core Agents
**Goal**: Users of any repo running `aether update` receive Claude Code agents that resolve correctly when the Task tool spawns them. Builder and Watcher are the first two agents shipped through this proven chain.
**Depends on**: Phase 26
**Requirements**: DIST-01, DIST-02, DIST-03, DIST-04, DIST-05, DIST-06, DIST-07, DIST-08, CORE-02, CORE-03, PWR-01, PWR-02, PWR-03, PWR-04, PWR-05, PWR-06, PWR-07, PWR-08
**Success Criteria** (what must be TRUE):
  1. `npm pack --dry-run` lists agent files from `.claude/agents/ant/` — no GSD agents included, no Aether agents excluded
  2. `npm install -g .` followed by listing `~/.aether/system/agents-claude/` shows the ant agents present
  3. `aether update` in a target repo creates `.claude/agents/ant/` containing the ant agent files
  4. Running `aether update` a second time with unchanged agents reports no files changed (idempotent)
  5. Removing an agent from source, running `npm install -g .` and `aether update`, removes it from the target repo
**Plans**: 4 plans in 2 waves
Plans:
- [ ] 27-01-PLAN.md — Wire distribution pipeline (package.json, cli.js, update-transaction.js, init.js)
- [ ] 27-02-PLAN.md — Create Builder agent (Claude Code subagent with PWR compliance)
- [ ] 27-03-PLAN.md — Create Watcher agent (read-only Claude Code subagent with PWR compliance)
- [ ] 27-04-PLAN.md — End-to-end verification (npm pack, hub sync, agent loading checkpoint)

### Phase 28: Orchestration Layer + Surveyor Variants
**Goal**: The full orchestration and codebase-context capability is available in Claude Code — Queen can coordinate workers, Route-Setter can plan phases, Scout can research, and all 4 Surveyors can characterize a repo.
**Depends on**: Phase 27
**Requirements**: CORE-01, CORE-04, CORE-05, CORE-06, CORE-07, CORE-08, CORE-09
**Success Criteria** (what must be TRUE):
  1. `/agents` in Claude Code shows `aether-queen`, `aether-scout`, `aether-route-setter`, and all 4 surveyor variants loaded without errors
  2. Queen's description routes correctly — it is not invoked for tasks that belong to Builder or Watcher
  3. All 4 surveyor agents have no Write or Edit in their tools field (read-only verified)
  4. Scout agent description explicitly names research and discovery as its trigger cases
**Plans**: TBD

### Phase 29: Specialist Agents + Agent Tests
**Goal**: All P2 specialist agents are shipped and a comprehensive AVA test suite enforces quality standards on every agent file — frontmatter, tool restrictions, naming, and body content.
**Depends on**: Phase 28
**Requirements**: SPEC-01, SPEC-02, SPEC-03, SPEC-04, SPEC-05, TEST-01, TEST-02, TEST-03, TEST-04, TEST-05
**Success Criteria** (what must be TRUE):
  1. `npm test` passes with tests verifying frontmatter completeness on all 22 agent files
  2. `npm test` catches a missing tools field or incorrect agent name format as a test failure
  3. Auditor and Tracker have no Write, Edit, or Bash in their tools field (read-only restriction enforced by tests)
  4. No agent file body contains spawn calls or activity-log requirements (test enforced)
**Plans**: TBD

### Phase 30: Niche Agents
**Goal**: All 8 niche agents exist as Claude Code subagents in `.claude/agents/ant/`, completing the full 22-agent roster. The fallback comment in `build.md` is unreachable for all 22 castes.
**Depends on**: Phase 29
**Requirements**: NICHE-01, NICHE-02, NICHE-03, NICHE-04, NICHE-05, NICHE-06, NICHE-07, NICHE-08
**Success Criteria** (what must be TRUE):
  1. `/agents` in Claude Code shows all 22 aether-* agents loaded (count test passes: TEST-05)
  2. Read-only niche agents (Gatekeeper, Includer, Measurer, Chaos, Archaeologist, Sage) have no Write or Edit in tools field
  3. Each niche agent description names a specific trigger case — not a generic role label
**Plans**: TBD

### Phase 31: Integration Verification + Cleanup
**Goal**: The full colony workflow is verified end-to-end with real agent invocations updating colony state correctly. The repo is clean: docs trimmed, bash bug fixed, repo-structure documented.
**Depends on**: Phase 30
**Requirements**: INT-01, INT-02, INT-03, CLEAN-01, CLEAN-02, CLEAN-03, CLEAN-04
**Success Criteria** (what must be TRUE):
  1. `/ant:build` resolves `subagent_type="aether-builder"` to `.claude/agents/ant/aether-builder.md` (not the fallback path)
  2. After an agent run, COLONY_STATE.json reflects updated state — the agent's output was consumed correctly
  3. `.aether/docs/` contains only 8-10 actively-maintained documents
  4. Bash line wrapping bug is fixed and verified with a test case
**Plans**: TBD

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
| 20. Distribution Simplification | v1.3 | Complete | 2026-02-19 | — |
| 21. Template Foundation | v1.3 | Complete | 2026-02-19 | — |
| 22. Agent Boilerplate Cleanup | v1.3 | Complete | 2026-02-19 | — |
| 23. Agent Resilience | v1.3 | Complete | 2026-02-19 | — |
| 24. Template Integration | v1.3 | Complete | 2026-02-20 | — |
| 25. Queen Coordination | v1.3 | Complete | 2026-02-20 | — |
| 26. Audit & Delete Dead Files | v1.4 | Complete | 2026-02-20 | — |
| 27. Distribution Infrastructure + First Core Agents | v2.0 | 0/4 | Planned | - |
| 28. Orchestration Layer + Surveyor Variants | v2.0 | 0/TBD | Not started | - |
| 29. Specialist Agents + Agent Tests | v2.0 | 0/TBD | Not started | - |
| 30. Niche Agents | v2.0 | 0/TBD | Not started | - |
| 31. Integration Verification + Cleanup | v2.0 | 0/TBD | Not started | - |

---

*Roadmap created: 2026-02-17*
*v1.0 shipped: 2026-02-18*
*v1.1 shipped: 2026-02-18*
*v1.2 shipped: 2026-02-19*
*v1.3 shipped: 2026-02-20*
*v1.4 (partial) shipped: 2026-02-20*
*v2.0 roadmap created: 2026-02-20*
