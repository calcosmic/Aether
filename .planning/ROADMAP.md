# Roadmap: Aether

## Milestones

- ✅ **v1.3 Maintenance & Pheromone Integration** — Phases 1-8 (shipped 2026-03-19)
- ✅ **v2.1 Production Hardening** — Phases 9-16 (shipped 2026-03-24)
- [ ] **v2.2 Living Wisdom** — Phases 17-20 (in progress)

## Phases

<details>
<summary>v1.3 Maintenance & Pheromone Integration (Phases 1-8) — SHIPPED 2026-03-19</summary>

- [x] Phase 1: Data Purge (2/2 plans) — completed 2026-03-19
- [x] Phase 2: Command Audit & Data Tooling (2/2 plans) — completed 2026-03-19
- [x] Phase 3: Pheromone Signal Plumbing (3/3 plans) — completed 2026-03-19
- [x] Phase 4: Pheromone Worker Integration (2/2 plans) — completed 2026-03-19
- [x] Phase 5: Learning Pipeline Validation (2/2 plans) — completed 2026-03-19
- [x] Phase 6: XML Exchange Activation (2/2 plans) — completed 2026-03-19
- [x] Phase 7: Fresh Install Hardening (2/2 plans) — completed 2026-03-19
- [x] Phase 8: Documentation Update (2/2 plans) — completed 2026-03-19

See: `.planning/milestones/v1.3-ROADMAP.md` for full details.

</details>

<details>
<summary>v2.1 Production Hardening (Phases 9-16) — SHIPPED 2026-03-24</summary>

- [x] Phase 9: Quick Wins (2/2 plans) — completed 2026-03-24
- [x] Phase 10: Error Triage (3/3 plans) — completed 2026-03-24
- [x] Phase 11: Dead Code Deprecation (2/2 plans) — completed 2026-03-24
- [x] Phase 12: State API & Verification (3/3 plans) — completed 2026-03-24
- [x] Phase 13: Monolith Modularization (9/9 plans) — completed 2026-03-24
- [x] Phase 14: Planning Depth (2/2 plans) — completed 2026-03-24
- [x] Phase 15: Documentation Accuracy (3/3 plans) — completed 2026-03-24
- [x] Phase 16: Ship (2/2 plans) — completed 2026-03-24

</details>

### v2.2 Living Wisdom (In Progress)

**Milestone Goal:** Make QUEEN.md and hive brain actually work — automatic wisdom accumulation during colony work, cross-colony knowledge sharing, so the system genuinely learns and gets smarter over time.

- [x] **Phase 17: Local Wisdom Accumulation** - Colony automatically writes learnings to QUEEN.md after builds and promotes high-confidence instincts (completed 2026-03-24)
- [x] **Phase 18: Local Wisdom Injection** - Colony-prime reads local QUEEN.md and injects wisdom into worker prompts so builds get smarter (completed 2026-03-25)
- [x] **Phase 19: Cross-Colony Hive** - Seal promotes wisdom to global hive, init seeds new colonies from hive, domain scoping filters relevance (completed 2026-03-25)
- [ ] **Phase 20: Hub Wisdom Layer** - Global QUEEN.md accumulates cross-cutting wisdom and colony-prime reads it alongside local wisdom

## Phase Details

### Phase 17: Local Wisdom Accumulation
**Goal**: Colony automatically captures what it learns during builds and promotes high-confidence patterns to QUEEN.md
**Depends on**: Nothing (first phase of v2.2)
**Requirements**: QUEEN-01, QUEEN-02
**Success Criteria** (what must be TRUE):
  1. After `/ant:build` completes, the local QUEEN.md contains a new entry describing what worked or failed during that build (without user intervention)
  2. Colony instincts that reach confidence >= 0.8 appear in the QUEEN.md wisdom section automatically (user never manually edits QUEEN.md)
  3. Repeated builds accumulate multiple wisdom entries in QUEEN.md — the file grows with real colony experience, not template placeholders
  4. A fresh colony's QUEEN.md starts minimal and fills up as builds execute — the before/after difference is visible
**Plans:** 2/2 plans complete
Plans:
- [ ] 17-01-PLAN.md — Restructure QUEEN.md template to 4-section format, update all parsers, add write subcommands
- [ ] 17-02-PLAN.md — Wire write subcommands into continue playbooks, migrate current repo QUEEN.md

### Phase 18: Local Wisdom Injection
**Goal**: Workers actually read accumulated QUEEN.md wisdom so each build benefits from previous colony experience
**Depends on**: Phase 17 (QUEEN.md must have real content to inject)
**Requirements**: QUEEN-03
**Success Criteria** (what must be TRUE):
  1. Colony-prime includes local QUEEN.md wisdom entries in the prompt section sent to builder and watcher agents
  2. A colony that has accumulated wisdom entries produces different (more informed) worker context than a fresh colony with empty QUEEN.md
  3. Wisdom injection respects the existing token budget — QUEEN.md content is trimmed if the budget is exceeded, not silently dropped or allowed to overflow
**Plans:** 1/1 plans complete
Plans:
- [ ] 18-01-PLAN.md — Post-extraction wisdom filtering, content-detection gate, fresh-vs-accumulated tests

### Phase 19: Cross-Colony Hive
**Goal**: Wisdom flows between colonies — seal exports to the global hive, init imports from it, and domain tags ensure relevance
**Depends on**: Phase 17 (local wisdom must exist to promote), Phase 18 (injection pattern established)
**Requirements**: HIVE-01, HIVE-02, HIVE-03
**Success Criteria** (what must be TRUE):
  1. Running `/ant:seal` on a colony with high-confidence instincts results in new entries appearing in `~/.aether/hive/wisdom.json`
  2. Running `/ant:init` in a new repo seeds its QUEEN.md with relevant wisdom from the hive — the new colony does not start from zero
  3. A web project colony receives web-relevant hive wisdom but not CLI-specific patterns (domain scoping filters by registry tags)
  4. The hive-to-colony flow is end-to-end verifiable: colony A seals wisdom, colony B inits and receives it
**Plans:** 1/1 plans complete
Plans:
- [ ] 19-01-PLAN.md — Fix seal domain tags, add domain auto-detection, create queen-seed-from-hive, wire init hive seeding, end-to-end tests

### Phase 20: Hub Wisdom Layer
**Goal**: A global QUEEN.md at hub level accumulates cross-cutting preferences and wisdom that apply to every colony
**Depends on**: Phase 18 (local injection pattern), Phase 19 (hive promotion established)
**Requirements**: HUB-01, HUB-02
**Success Criteria** (what must be TRUE):
  1. The global QUEEN.md (`~/.aether/QUEEN.md`) accumulates user preferences and cross-cutting wisdom that is not specific to any single colony
  2. Colony-prime reads both local QUEEN.md and global QUEEN.md, injecting relevant wisdom from both sources into worker prompts
  3. Global wisdom and local wisdom are distinguishable in the injected prompt — workers can see which patterns are colony-specific vs universal
**Plans:** 1 plan
Plans:
- [ ] 20-01-PLAN.md — Split colony-prime into global/local QUEEN WISDOM sections, v1 migration, budget enforcement, tests

## Progress

**Execution Order:**
Phases execute in numeric order: 17 -> 18 -> 19 -> 20

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Data Purge | v1.3 | 2/2 | Complete | 2026-03-19 |
| 2. Command Audit & Data Tooling | v1.3 | 2/2 | Complete | 2026-03-19 |
| 3. Pheromone Signal Plumbing | v1.3 | 3/3 | Complete | 2026-03-19 |
| 4. Pheromone Worker Integration | v1.3 | 2/2 | Complete | 2026-03-19 |
| 5. Learning Pipeline Validation | v1.3 | 2/2 | Complete | 2026-03-19 |
| 6. XML Exchange Activation | v1.3 | 2/2 | Complete | 2026-03-19 |
| 7. Fresh Install Hardening | v1.3 | 2/2 | Complete | 2026-03-19 |
| 8. Documentation Update | v1.3 | 2/2 | Complete | 2026-03-19 |
| 9. Quick Wins | v2.1 | 2/2 | Complete | 2026-03-24 |
| 10. Error Triage | v2.1 | 3/3 | Complete | 2026-03-24 |
| 11. Dead Code Deprecation | v2.1 | 2/2 | Complete | 2026-03-24 |
| 12. State API & Verification | v2.1 | 3/3 | Complete | 2026-03-24 |
| 13. Monolith Modularization | v2.1 | 9/9 | Complete | 2026-03-24 |
| 14. Planning Depth | v2.1 | 2/2 | Complete | 2026-03-24 |
| 15. Documentation Accuracy | v2.1 | 3/3 | Complete | 2026-03-24 |
| 16. Ship | v2.1 | 2/2 | Complete | 2026-03-24 |
| 17. Local Wisdom Accumulation | v2.2 | Complete    | 2026-03-24 | - |
| 18. Local Wisdom Injection | v2.2 | Complete    | 2026-03-25 | - |
| 19. Cross-Colony Hive | v2.2 | Complete    | 2026-03-25 | - |
| 20. Hub Wisdom Layer | v2.2 | 0/1 | Not started | - |
