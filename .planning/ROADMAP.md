# Roadmap: Aether

## Milestones

- ✅ **v1.3 Maintenance & Pheromone Integration** — Phases 1-8 (shipped 2026-03-19)
- ✅ **v2.1 Production Hardening** — Phases 9-16 (shipped 2026-03-24)
- ✅ **v2.2 Living Wisdom** — Phases 17-20 (shipped 2026-03-25)
- [ ] **v2.3 Per-Caste Model Routing** — Phases 21-24 (in progress)

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

<details>
<summary>v2.2 Living Wisdom (Phases 17-20) — SHIPPED 2026-03-25</summary>

- [x] Phase 17: Local Wisdom Accumulation — completed 2026-03-24
- [x] Phase 18: Local Wisdom Injection — completed 2026-03-25
- [x] Phase 19: Cross-Colony Hive — completed 2026-03-25
- [x] Phase 20: Hub Wisdom Layer — completed 2026-03-25

</details>

### v2.3 Per-Caste Model Routing (In Progress)

**Milestone Goal:** Reasoning-heavy castes use GLM-5 via the opus slot while execution castes stay on GLM-5-turbo via sonnet, using model-profiles.yaml as the single source of truth.

- [ ] **Phase 21: Test Infrastructure Refactor** - Centralize test mocks to read from model-profiles.yaml so tests survive YAML changes
- [ ] **Phase 22: Config Foundation & Core Routing** - Map castes to model slots via agent frontmatter, update settings and YAML config
- [ ] **Phase 23: Tooling & Overrides** - Slot resolution functions, CLI subcommand, and build-time model override flag
- [ ] **Phase 24: Safety & Verification** - Spawn-tree tracking, verify-castes display, GLM-5 loop warnings, config swap docs

## Phase Details

### Phase 21: Test Infrastructure Refactor
**Goal**: Tests read caste-to-model mappings from model-profiles.yaml instead of hardcoding model names, so any future YAML change does not break the test suite
**Depends on**: Nothing (first phase of v2.3)
**Requirements**: TEST-01, TEST-02, TEST-03
**Success Criteria** (what must be TRUE):
  1. Changing a caste's model assignment in model-profiles.yaml causes all related test assertions to update automatically (no manual test edits needed)
  2. A new test can import the mock profile helper and get the full caste-to-model mapping without duplicating any YAML content
  3. Running the full test suite after swapping two castes' model assignments in YAML produces zero failures
**Plans**: 3 plans (2 completed + 1 gap closure)
  - [x] 21-01-PLAN.md — Create centralized mock profile helper and refactor 3 core test files
  - [x] 21-02-PLAN.md — Refactor CLI/telemetry test files and add regression test
  - [ ] 21-03-PLAN.md — Gap closure: replace hardcoded model names in integration test

### Phase 22: Config Foundation & Core Routing
**Goal**: Reasoning castes route to the opus model slot and execution castes route to the sonnet slot, with a single settings.json change activating the GLM-5 mapping chain
**Depends on**: Phase 21 (test infrastructure must be centralized before YAML changes)
**Requirements**: ROUTE-01, ROUTE-02, ROUTE-03, ROUTE-04, ROUTE-05, ROUTE-06, ROUTE-07, ROUTE-08
**Success Criteria** (what must be TRUE):
  1. The queen, archaeologist, and route-setter agent files declare `model: opus` in their frontmatter; all other non-specialist agents declare `model: sonnet`
  2. `ANTHROPIC_DEFAULT_OPUS_MODEL` in settings.json.3model points to `glm-5`, making opus-slot agents use GLM-5 through the proxy
  3. A user who swaps to Claude API mode (no proxy) sees the same slot-to-caste assignments resolve to native Claude models without any Aether code changes
  4. model-profiles.yaml reflects the two-tier split and workers.md no longer claims per-caste routing is impossible
**Plans**: TBD

### Phase 23: Tooling & Overrides
**Goal**: Aether provides functions and CLI commands to resolve any caste to its model slot, and builders can override the default slot for an entire build
**Depends on**: Phase 22 (routing must be established before tooling queries it)
**Requirements**: TOOL-01, TOOL-02, TOOL-03, TOOL-04
**Success Criteria** (what must be TRUE):
  1. Running `model-slot get builder` outputs `sonnet`; running `model-slot get queen` outputs `opus`
  2. The `getModelSlotForCaste()` function in bin/lib/model-profiles.js returns the correct slot for all 22 castes when reading from model-profiles.yaml
  3. Passing `--model opus` to `/ant:build <phase>` forces all workers in that build to use the opus slot regardless of their default assignment
  4. Invalid slot names (e.g., `--model gpt-4`) produce a clear error listing the valid options (opus, sonnet, haiku, inherit)
**Plans**: TBD

### Phase 24: Safety & Verification
**Goal**: Users can verify which model slot each caste uses at runtime, spawned workers show their slot in the spawn tree, and GLM-5 loop risk is documented in reasoning caste agent definitions
**Depends on**: Phase 22 (castes must have slot assignments to display), Phase 23 (slot resolution must work for verify-castes)
**Requirements**: SAFE-01, SAFE-02, SAFE-03, SAFE-04
**Success Criteria** (what must be TRUE):
  1. spawn-tree.txt includes a `model` column showing the slot (opus/sonnet/inherit) used for each spawned worker
  2. `/ant:verify-castes` prints a table mapping every caste to its assigned model slot, with reasoning castes visually distinguished from execution castes
  3. The queen, archaeologist, and route-setter agent definitions contain a safety note warning about GLM-5 loop risk when generation constraints are not enforced
  4. A user reading workers.md or verify-castes.md can follow step-by-step instructions to swap between Claude API and GLM proxy modes
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 21 -> 22 -> 23 -> 24

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
| 17. Local Wisdom Accumulation | v2.2 | Complete | Complete | 2026-03-24 |
| 18. Local Wisdom Injection | v2.2 | Complete | Complete | 2026-03-25 |
| 19. Cross-Colony Hive | v2.2 | Complete | Complete | 2026-03-25 |
| 20. Hub Wisdom Layer | v2.2 | Complete | Complete | 2026-03-25 |
| 21. Test Infrastructure Refactor | v2.3 | 0/2 | Not started | - |
| 22. Config Foundation & Core Routing | v2.3 | 0/? | Not started | - |
| 23. Tooling & Overrides | v2.3 | 0/? | Not started | - |
| 24. Safety & Verification | v2.3 | 0/? | Not started | - |
