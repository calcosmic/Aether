# Roadmap: Aether

## Milestones

- ✅ **v1.3 Maintenance & Pheromone Integration** — Phases 1-8 (shipped 2026-03-19)
- ✅ **v2.1 Production Hardening** — Phases 9-16 (shipped 2026-03-24)
- ✅ **v2.2 Living Wisdom** — Phases 17-20 (shipped 2026-03-25)
- ✅ **v2.3 Per-Caste Model Routing** — Phases 21-24 (shipped 2026-03-27)
- ✅ **v2.4 Living Wisdom** — Phases 25-28 (shipped 2026-03-27)
- ✅ **v2.5 Smart Init** — Phases 29-32 (shipped 2026-03-27)
- **v2.6 Bugfix & Hardening** — Phases 33-38 (in progress)

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

<details>
<summary>v2.3 Per-Caste Model Routing (Phases 21-24) — SHIPPED 2026-03-27</summary>

- [x] Phase 21: Test Infrastructure Refactor (3 plans) — completed 2026-03-27
- [x] Phase 22: Config Foundation & Core Routing (3 plans) — completed 2026-03-27
- [x] Phase 23: Tooling & Overrides (2 plans) — completed 2026-03-27
- [x] Phase 24: Safety & Verification (2 plans) — completed 2026-03-27

</details>

<details>
<summary>v2.4 Living Wisdom (Phases 25-28) — SHIPPED 2026-03-27</summary>

- [x] Phase 25: Agent Definitions (Oracle + Architect) — completed 2026-03-27
- [x] Phase 26: Wisdom Pipeline Wiring — completed 2026-03-27
- [x] Phase 27: Deterministic Fallback + Dedup — completed 2026-03-27
- [x] Phase 28: Integration Validation — completed 2026-03-27

</details>

<details>
<summary>v2.5 Smart Init (Phases 29-32) — SHIPPED 2026-03-27</summary>

- [x] Phase 29: Repo Scanning Module (3/3 plans) — completed 2026-03-27
- [x] Phase 30: Charter Management (2/2 plans) — completed 2026-03-27
- [x] Phase 31: Init.md Smart Init Rewrite (2/2 plans) — completed 2026-03-27
- [x] Phase 32: Intelligence Enhancements (3/3 plans) — completed 2026-03-27

</details>

### v2.6 Bugfix & Hardening (In Progress)

**Milestone Goal:** Fix critical data-corruption bugs, harden infrastructure, clear high-priority TODOs.

- [x] **Phase 33: Input Escaping & Atomic Write Safety** - Fix ant_name injection in grep/JSON, jq-escape all dynamic values, release locks on validation failure (completed 2026-03-29)
- [ ] **Phase 34: Cross-Colony Isolation** - Eliminate information bleed between colonies via proper name extraction, lock scoping, and file namespacing
- [ ] **Phase 35: Colony Depth & Model Routing** - Depth selector gates Oracle/Scout spawns; model routing either wired end-to-end or dead code removed
- [ ] **Phase 36: YAML Command Generator** - Single YAML source produces both Claude and OpenCode command markdown
- [ ] **Phase 37: XML Core Integration** - XML export/import wired into seal, entomb, and init lifecycle commands
- [ ] **Phase 38: Cleanup & Maintenance** - Deprecate old npm versions, generate error code docs, remove dead awk code

## Phase Details

### Phase 33: Input Escaping & Atomic Write Safety
**Goal**: Dynamic values flowing through grep patterns, JSON construction, and atomic writes cannot corrupt data or break commands
**Depends on**: Nothing (first phase — fixes data-corrupting bugs)
**Requirements**: SAFE-01, SAFE-03, SAFE-04
**Success Criteria** (what must be TRUE):
  1. Running `grep -F` on ant_name values containing regex metacharacters (e.g., `worker.builder+1`) returns correct matches without errors across spawn.sh, swarm.sh, spawn-tree.sh, aether-utils.sh
  2. JSON output from all 14 identified locations in utils/ produces valid JSON even when dynamic values contain quotes, newlines, or backslashes
  3. When atomic_write encounters a JSON validation failure, the lock file is released and no stale locks remain
  4. All 616+ existing tests still pass after escaping changes
**Plans**: 4 plans
- [ ] 33-01-PLAN.md — Fix grep pattern injection: add -F flag to all ant_name greps + escape ant_name in JSON output
- [ ] 33-02-PLAN.md — Fix json_ok string interpolation across all utils/ modules (session, queen, learning, pheromone, etc.)
- [ ] 33-03-PLAN.md — Audit lock release on all acquire_lock callers + harden atomic_write + stale lock auto-expiry
- [ ] 33-04-PLAN.md — Dedicated data-safety.test.js + Data Safety section in /ant:status

### Phase 34: Cross-Colony Isolation
**Goal**: Two colonies running on the same machine cannot read or corrupt each other's state
**Depends on**: Phase 33 (escaping fixes prevent masking isolation bugs)
**Requirements**: SAFE-02
**Success Criteria** (what must be TRUE):
  1. Colony name extraction uses `_colony_name()` from queen.sh instead of fragile session_id splitting — verified at learning.sh:378, learning.sh:930, aether-utils.sh:3655
  2. `LOCK_DIR` in hive.sh is passed as a function parameter, never mutated as a global variable
  3. Shared data files (pheromones.json, learning-observations.json, session.json, run-state.json) include colony namespace so two colonies writing concurrently do not overwrite each other
  4. Existing single-colony workflows still work identically (no regression)
**Plans:** 5 plans

Plans:
- [ ] 34-01-PLAN.md — Replace all 13 session_id splitting locations with colony-name subcommand (3 shell + 9 playbook + 1 OpenCode)
- [ ] 34-02-PLAN.md — Add acquire_lock_at/release_lock_at to file-lock.sh and refactor hive.sh to eliminate LOCK_DIR mutation
- [ ] 34-03-PLAN.md — Add COLONY_DATA_DIR resolution + auto-migration infrastructure, update aether-utils.sh file references
- [ ] 34-04-PLAN.md — Update all 15 utils/ modules to use COLONY_DATA_DIR for per-colony file references
- [ ] 34-05-PLAN.md — Integration tests for colony isolation (COLONY_DATA_DIR, migration, lock tagging, backwards compat)

### Phase 35: Colony Depth & Model Routing
**Goal**: Colony operators can control how deeply the system investigates (gating expensive agent spawns) and model routing is either functional end-to-end or honestly removed
**Depends on**: Phase 34 (colony state changes in 34 affect COLONY_STATE.json which depth selector also modifies)
**Requirements**: INFRA-01, INFRA-02
**Success Criteria** (what must be TRUE):
  1. `colony_depth` field exists in COLONY_STATE.json with values light/standard/deep/full, defaulting to standard
  2. Oracle spawns in build-wave.md are gated by colony depth (only spawn at deep/full), Scout spawns respect depth setting
  3. Model routing either passes the resolved model slot to the actual agent spawn call, or all model routing code (model-profiles.yaml, caste table, model-slot subcommand) is removed with a documented decision — no dead code left in between
  4. `/ant:status` or colony dashboard displays the active depth setting
**Plans**: TBD

### Phase 36: YAML Command Generator
**Goal**: A single set of YAML source files produces both Claude Code and OpenCode command markdown, eliminating manual duplication of 44 commands
**Depends on**: Phase 33 (no hard dependency, but safety fixes first)
**Requirements**: INFRA-03
**Success Criteria** (what must be TRUE):
  1. YAML source files exist for each command, containing the canonical command spec
  2. Running the generator script produces .claude/commands/ant/*.md and .opencode/commands/ant/*.md from YAML sources
  3. Generated output matches (or improves upon) the current hand-written command files — no loss of functionality
  4. `npm run lint:sync` validates that generated files are up-to-date with YAML sources
**Plans**: TBD

### Phase 37: XML Core Integration
**Goal**: XML export/import is wired into colony lifecycle commands so cross-colony data transfer happens automatically at key moments
**Depends on**: Phase 34 (colony isolation must be solid before auto-exporting colony data)
**Requirements**: INFRA-04
**Success Criteria** (what must be TRUE):
  1. `/ant:seal` automatically exports pheromone signals and wisdom to XML as part of the seal process
  2. `/ant:entomb` archives XML exchange files alongside the colony chamber
  3. `/ant:init` can import XML files from a previous colony to seed a new one (opt-in, not automatic)
  4. XML files in .aether/exchange/ are included in `validate-package.sh` distribution checks
**Plans**: TBD

### Phase 38: Cleanup & Maintenance
**Goal**: Registry housekeeping, developer documentation, and dead code removal — small items that don't warrant their own phase
**Depends on**: Phase 33 (MAINT-02 depends on error-handler.sh being stable after escaping fixes; MAINT-03 depends on no new awk usage introduced by earlier phases)
**Requirements**: MAINT-01, MAINT-02, MAINT-03
**Success Criteria** (what must be TRUE):
  1. Old 2.x npm versions are marked deprecated on the registry with a message pointing to current version
  2. An error code reference document exists in .aether/docs/ listing all error codes from error-handler.sh with descriptions, and is included in npm distribution
  3. The unused `models[]` awk array is removed from aether-utils.sh with no test regressions
**Plans**: TBD

## Progress

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
| 21. Test Infrastructure Refactor | v2.3 | Complete | Complete | 2026-03-27 |
| 22. Config Foundation & Core Routing | v2.3 | Complete | Complete | 2026-03-27 |
| 23. Tooling & Overrides | v2.3 | Complete | Complete | 2026-03-27 |
| 24. Safety & Verification | v2.3 | 2/2 | Complete | 2026-03-27 |
| 25. Agent Definitions (Oracle + Architect) | v2.4 | Complete | Complete | 2026-03-27 |
| 26. Wisdom Pipeline Wiring | v2.4 | Complete | Complete | 2026-03-27 |
| 27. Deterministic Fallback + Dedup | v2.4 | Complete | Complete | 2026-03-27 |
| 28. Integration Validation | v2.4 | Complete | Complete | 2026-03-27 |
| 29. Repo Scanning Module | v2.5 | 3/3 | Complete | 2026-03-27 |
| 30. Charter Management | v2.5 | 2/2 | Complete | 2026-03-27 |
| 31. Init.md Smart Init Rewrite | v2.5 | 2/2 | Complete | 2026-03-27 |
| 32. Intelligence Enhancements | v2.5 | 3/3 | Complete | 2026-03-27 |
| 33. Input Escaping & Atomic Write Safety | v2.6 | Complete    | 2026-03-29 | - |
| 34. Cross-Colony Isolation | v2.6 | 0/TBD | Not started | - |
| 35. Colony Depth & Model Routing | v2.6 | 0/TBD | Not started | - |
| 36. YAML Command Generator | v2.6 | 0/TBD | Not started | - |
| 37. XML Core Integration | v2.6 | 0/TBD | Not started | - |
| 38. Cleanup & Maintenance | v2.6 | 0/TBD | Not started | - |
