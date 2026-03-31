# Roadmap: Aether

## Milestones

- **v1.3 Maintenance & Pheromone Integration** -- Phases 1-8 (shipped 2026-03-19)
- **v2.1 Production Hardening** -- Phases 9-16 (shipped 2026-03-24)
- **v2.2 Living Wisdom** -- Phases 17-20 (shipped 2026-03-25)
- **v2.3 Per-Caste Model Routing** -- Phases 21-24 (shipped 2026-03-27)
- **v2.4 Living Wisdom** -- Phases 25-28 (shipped 2026-03-27)
- **v2.5 Smart Init** -- Phases 29-32 (shipped 2026-03-27)
- **v2.6 Bugfix & Hardening** -- Phases 33-38 (shipped 2026-03-30)
- **v2.7 PR Workflow + Stability** -- Phases 39-44 (planned)

## Phases

<details>
<summary>v1.3 Maintenance & Pheromone Integration (Phases 1-8) -- SHIPPED 2026-03-19</summary>

- [x] Phase 1: Data Purge (2/2 plans) -- completed 2026-03-19
- [x] Phase 2: Command Audit & Data Tooling (2/2 plans) -- completed 2026-03-19
- [x] Phase 3: Pheromone Signal Plumbing (3/3 plans) -- completed 2026-03-19
- [x] Phase 4: Pheromone Worker Integration (2/2 plans) -- completed 2026-03-19
- [x] Phase 5: Learning Pipeline Validation (2/2 plans) -- completed 2026-03-19
- [x] Phase 6: XML Exchange Activation (2/2 plans) -- completed 2026-03-19
- [x] Phase 7: Fresh Install Hardening (2/2 plans) -- completed 2026-03-19
- [x] Phase 8: Documentation Update (2/2 plans) -- completed 2026-03-19

</details>

<details>
<summary>v2.1 Production Hardening (Phases 9-16) -- SHIPPED 2026-03-24</summary>

- [x] Phase 9: Quick Wins (2/2 plans) -- completed 2026-03-24
- [x] Phase 10: Error Triage (3/3 plans) -- completed 2026-03-24
- [x] Phase 11: Dead Code Deprecation (2/2 plans) -- completed 2026-03-24
- [x] Phase 12: State API & Verification (3/3 plans) -- completed 2026-03-24
- [x] Phase 13: Monolith Modularization (9/9 plans) -- completed 2026-03-24
- [x] Phase 14: Planning Depth (2/2 plans) -- completed 2026-03-24
- [x] Phase 15: Documentation Accuracy (3/3 plans) -- completed 2026-03-24
- [x] Phase 16: Ship (2/2 plans) -- completed 2026-03-24

</details>

<details>
<summary>v2.2 Living Wisdom (Phases 17-20) -- SHIPPED 2026-03-25</summary>

- [x] Phase 17: Local Wisdom Accumulation -- completed 2026-03-24
- [x] Phase 18: Local Wisdom Injection -- completed 2026-03-25
- [x] Phase 19: Cross-Colony Hive -- completed 2026-03-25
- [x] Phase 20: Hub Wisdom Layer -- completed 2026-03-25

</details>

<details>
<summary>v2.3 Per-Caste Model Routing (Phases 21-24) -- SHIPPED 2026-03-27</summary>

- [x] Phase 21: Test Infrastructure Refactor (3 plans) -- completed 2026-03-27
- [x] Phase 22: Config Foundation & Core Routing (3 plans) -- completed 2026-03-27
- [x] Phase 23: Tooling & Overrides (2 plans) -- completed 2026-03-27
- [x] Phase 24: Safety & Verification (2 plans) -- completed 2026-03-27

</details>

<details>
<summary>v2.4 Living Wisdom (Phases 25-28) -- SHIPPED 2026-03-27</summary>

- [x] Phase 25: Agent Definitions (Oracle + Architect) -- completed 2026-03-27
- [x] Phase 26: Wisdom Pipeline Wiring -- completed 2026-03-27
- [x] Phase 27: Deterministic Fallback + Dedup -- completed 2026-03-27
- [x] Phase 28: Integration Validation -- completed 2026-03-27

</details>

<details>
<summary>v2.5 Smart Init (Phases 29-32) -- SHIPPED 2026-03-27</summary>

- [x] Phase 29: Repo Scanning Module (3/3 plans) -- completed 2026-03-27
- [x] Phase 30: Charter Management (2/2 plans) -- completed 2026-03-27
- [x] Phase 31: Init.md Smart Init Rewrite (2/2 plans) -- completed 2026-03-27
- [x] Phase 32: Intelligence Enhancements (3/3 plans) -- completed 2026-03-27

</details>

### v2.6 Bugfix & Hardening (Shipped)

**Milestone Goal:** Fix critical data-corruption bugs, harden infrastructure, clear high-priority TODOs.

- [x] **Phase 33: Input Escaping & Atomic Write Safety** - Fix ant_name injection in grep/JSON, jq-escape all dynamic values, release locks on validation failure (completed 2026-03-29)
- [x] **Phase 34: Cross-Colony Isolation** - Eliminate information bleed between colonies via proper name extraction, lock scoping, and file namespacing (completed 2026-03-29)
- [x] **Phase 35: Colony Depth & Model Routing** - Depth selector gates Oracle/Scout spawns; model routing either wired end-to-end or dead code removed (completed 2026-03-29)
- [x] **Phase 36: YAML Command Generator** - Single YAML source produces both Claude and OpenCode command markdown (completed 2026-03-29)
- [x] **Phase 37: XML Core Integration** - XML export/import wired into seal, entomb, and init lifecycle commands (completed 2026-03-29)
- [x] **Phase 38: Cleanup & Maintenance** - Deprecate old npm versions, generate error code docs, remove dead awk code (completed 2026-03-29)

### v2.7 PR Workflow + Stability (Planned)

**Milestone Goal:** Make multi-branch colony work safe and productive -- clash detection prevents conflicts, pheromones propagate across branches, failures collect on merge, and CI agents get structured context.

- [ ] **Phase 39: State Safety** -- STATE-01, STATE-02
- [x] **Phase 40: Pheromone Propagation** -- PHERO-01, PHERO-02, PHERO-03 (completed 2026-03-30)
- [x] **Phase 41: Midden Collection** -- MIDD-01, MIDD-02, MIDD-03 (completed 2026-03-30)
- [ ] **Phase 42: CI Context Assembly** -- CI-01, CI-02, CI-03
- [ ] **Phase 43: Clash Detection Integration** -- CLASH-01, CLASH-02, CLASH-03
- [ ] **Phase 44: Release Hygiene & Ship** -- REL-01, REL-02, REL-03, TEST-01, TEST-02

## Phase Details

### Phase 33: Input Escaping & Atomic Write Safety
**Goal**: Dynamic values flowing through grep patterns, JSON construction, and atomic writes cannot corrupt data or break commands
**Depends on**: Nothing (first phase -- fixes data-corrupting bugs)
**Requirements**: SAFE-01, SAFE-03, SAFE-04
**Success Criteria** (what must be TRUE):
  1. Running `grep -F` on ant_name values containing regex metacharacters (e.g., `worker.builder+1`) returns correct matches without errors across spawn.sh, swarm.sh, spawn-tree.sh, aether-utils.sh
  2. JSON output from all 14 identified locations in utils/ produces valid JSON even when dynamic values contain quotes, newlines, or backslashes
  3. When atomic_write encounters a JSON validation failure, the lock file is released and no stale locks remain
  4. All 616+ existing tests still pass after escaping changes
**Plans**: 4 plans
- [ ] 33-01-PLAN.md -- Fix grep pattern injection: add -F flag to all ant_name greps + escape ant_name in JSON output
- [ ] 33-02-PLAN.md -- Fix json_ok string interpolation across all utils/ modules (session, queen, learning, pheromone, etc.)
- [ ] 33-03-PLAN.md -- Audit lock release on all acquire_lock callers + harden atomic_write + stale lock auto-expiry
- [ ] 33-04-PLAN.md -- Dedicated data-safety.test.js + Data Safety section in /ant:status

### Phase 34: Cross-Colony Isolation
**Goal**: Two colonies running on the same machine cannot read or corrupt each other's state
**Depends on**: Phase 33 (escaping fixes prevent masking isolation bugs)
**Requirements**: SAFE-02
**Success Criteria** (what must be TRUE):
  1. Colony name extraction uses `_colony_name()` from queen.sh instead of fragile session_id splitting -- verified at learning.sh:378, learning.sh:930, aether-utils.sh:3655
  2. `LOCK_DIR` in hive.sh is passed as a function parameter, never mutated as a global variable
  3. Shared data files (pheromones.json, learning-observations.json, session.json, run-state.json) include colony namespace so two colonies writing concurrently do not overwrite each other
  4. Existing single-colony workflows still work identically (no regression)
**Plans:** 5/5 plans complete

Plans:
- [x] 34-01-PLAN.md -- Replace all 13 session_id splitting locations with colony-name subcommand (3 shell + 9 playbook + 1 OpenCode)
- [x] 34-02-PLAN.md -- Add acquire_lock_at/release_lock_at to file-lock.sh and refactor hive.sh to eliminate LOCK_DIR mutation
- [x] 34-03-PLAN.md -- Add COLONY_DATA_DIR resolution + auto-migration infrastructure, update aether-utils.sh file references
- [x] 34-04-PLAN.md -- Update all 15 utils/ modules to use COLONY_DATA_DIR for per-colony file references
- [x] 34-05-PLAN.md -- Integration tests for colony isolation (COLONY_DATA_DIR, migration, lock tagging, backwards compat)

### Phase 35: Colony Depth & Model Routing
**Goal**: Colony operators can control how deeply the system investigates (gating expensive agent spawns) and model routing is either functional end-to-end or honestly removed
**Depends on**: Phase 34 (colony state changes in 34 affect COLONY_STATE.json which depth selector also modifies)
**Requirements**: INFRA-01, INFRA-02
**Success Criteria** (what must be TRUE):
  1. `colony_depth` field exists in COLONY_STATE.json with values light/standard/deep/full, defaulting to standard
  2. Oracle spawns in build-wave.md are gated by colony depth (only spawn at deep/full), Scout spawns respect depth setting
  3. Model routing either passes the resolved model slot to the actual agent spawn call, or all model routing code (model-profiles.yaml, caste table, model-slot subcommand) is removed with a documented decision -- no dead code left in between
  4. `/ant:status` or colony dashboard displays the active depth setting
**Plans**: TBD

### Phase 36: YAML Command Generator
**Goal**: A single set of YAML source files produces both Claude Code and OpenCode command markdown, eliminating manual duplication of 44 commands
**Depends on**: Phase 33 (no hard dependency, but safety fixes first)
**Requirements**: INFRA-03
**Success Criteria** (what must be TRUE):
  1. YAML source files exist for each command, containing the canonical command spec
  2. Running the generator script produces .claude/commands/ant/*.md and .opencode/commands/ant/*.md from YAML sources
  3. Generated output matches (or improves upon) the current hand-written command files -- no loss of functionality
  4. `npm run lint:sync` validates that generated files are up-to-date with YAML sources
**Plans**: 4 plans

Plans:
- [x] 36-01-PLAN.md -- Generator engine (bin/generate-commands.js) + unit tests
- [ ] 36-02-PLAN.md -- Convert 22 simpler commands to YAML source format
- [x] 36-03-PLAN.md -- Convert 22 complex commands to YAML (including build.md, continue.md)
- [ ] 36-04-PLAN.md -- Update sync tooling (generate-commands.sh) + npm scripts + full validation

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
**Goal**: Registry housekeeping, developer documentation, and dead code removal -- small items that don't warrant their own phase
**Depends on**: Phase 33 (MAINT-02 depends on error-handler.sh being stable after escaping fixes; MAINT-03 depends on no new awk usage introduced by earlier phases)
**Requirements**: MAINT-01, MAINT-02, MAINT-03
**Success Criteria** (what must be TRUE):
  1. Old 2.x npm versions are marked deprecated on the registry with a message pointing to current version
  2. An error code reference document exists in .aether/docs/ listing all error codes from error-handler.sh with descriptions, and is included in npm distribution
  3. The unused `models[]` awk array is removed from spawn-tree.sh with no test regressions
**Plans**: 2 plans

Plans:
- [x] 38-01-PLAN.md -- Remove dead models[] awk array from spawn-tree.sh + audit error-codes.md completeness
- [ ] 38-02-PLAN.md -- Deprecate old npm versions + fix dist-tag + align package.json version

### Phase 39: State Safety
**Goal**: All COLONY_STATE.json writes use atomic mutations and test suite passes on clean/empty state
**Depends on**: Nothing (foundation for all subsequent phases)
**Requirements**: STATE-01, STATE-02
**Success Criteria** (what must be TRUE):
  1. `grep -rn 'jq "\(.*\)" ' .aether/ --include='*.sh' | grep COLONY_STATE` returns zero results -- no raw jq writes to state file
  2. Every COLONY_STATE.json write path goes through `state-mutate` with atomic file locking
  3. `npm test` passes with zero failures when COLONY_STATE.json contains minimal valid state
  4. State validation tests handle missing optional fields gracefully (no hard failure on empty colony)
**Existing work**: state-mutate subcommand exists in state-api.sh; 4 pheromone subcommands already migrated; design doc at `.aether/docs/state-contract-design.md`
**Plans:** 2 plans

Plans:
- [ ] 39-01-PLAN.md -- Stash protection: add pathspec exclusion to 3 stash entry points (swarm.sh, build-prep.md, build-full.md)
- [ ] 39-02-PLAN.md -- State migration + test fixes: migrate queen.sh to _state_mutate, reset COLONY_STATE.json, fix 11 failing tests

---

### Phase 40: Pheromone Propagation
**Goal**: Pheromone signals flow across git branches -- signals from main reach worktrees, and branch-specific signals merge back after PR
**Depends on**: Phase 39 (state safety must be solid before adding cross-branch writes)
**Requirements**: PHERO-01, PHERO-02, PHERO-03
**Success Criteria** (what must be TRUE):
  1. Creating a worktree branch via `_worktree_create` automatically copies active main-branch pheromones into the branch
  2. `pheromone-snapshot-inject` produces a valid snapshot JSON that can be read by the branch's pheromone system
  3. `pheromone-merge-back` merges user-created branch signals into main without duplicating existing signals
  4. Merge conflict resolution follows priority: REDIRECT > FOCUS > FEEDBACK, with strength-based dedup
**Existing work**: 4 subcommand stubs in pheromone.sh; design doc at `.aether/docs/pheromone-propagation-design.md`; test file at `test/pheromone-snapshot-merge.sh`

---

### Phase 41: Midden Collection
**Goal**: Failure records from merged branches are collected into main's midden with idempotency and cross-PR pattern detection
**Depends on**: Phase 40 (pheromone propagation pattern informs midden flow)
**Requirements**: MIDD-01, MIDD-02, MIDD-03
**Success Criteria** (what must be TRUE):
  1. `midden-collect --branch <branch> --merge-sha <sha>` ingests failure records from the branch into main's midden
  2. Running midden-collect twice with the same merge SHA produces no duplicates (idempotent)
  3. `midden-handle-revert --sha <sha>` tags affected entries rather than deleting them
  4. `midden-cross-pr-analysis` returns failure patterns detected across 2+ PRs
**Existing work**: Design doc at `.aether/docs/midden-collection-design.md`; existing midden.sh has core write/read/acknowledge functions

---

### Phase 42: CI Context Assembly
**Goal**: CI agents get machine-readable colony context via `pr-context` subcommand, replacing interactive colony-prime for automated workflows
**Depends on**: Phase 41 (needs midden data for complete context)
**Requirements**: CI-01, CI-02, CI-03
**Success Criteria** (what must be TRUE):
  1. `aether pr-context` outputs valid JSON with sections: colony_state, pheromones, phase_context, blockers, hive_wisdom
  2. When a source file is missing or corrupt, pr-context returns partial data with the missing section marked as `null` -- never hard-fails
  3. Normal mode output stays under 6,000 characters; compact mode under 3,000 characters
  4. Token budget trimming follows the same priority order as colony-prime (rolling summary first, blockers never)
**Existing work**: Design doc at `.aether/docs/ci-context-assembly-design.md`; colony-prime prompt assembly exists as reference implementation
**Plans:** 2 plans

Plans:
- [ ] 42-01-PLAN.md -- Extract _budget_enforce(), implement pr-context with all sections, cache, midden, tests
- [ ] 42-02-PLAN.md -- Wire pr-context into /ant:continue and /ant:run playbooks

---

### Phase 42.1: Release hygiene -- version drift, npm packaging, command sync, doc accuracy (INSERTED)

**Goal:** Fix version drift in CLAUDE.md, expand validate-package.sh coverage to all packaged utils, regenerate stale YAML-generated commands, and correct inaccurate documentation counts.
**Requirements**: REL-01, REL-02
**Depends on:** Phase 42
**Plans:** 2/2 plans complete

Plans:
- [x] 42.1-01-PLAN.md -- Expand validate-package.sh REQUIRED_FILES to all 35 utils + regenerate stale commands
- [ ] 42.1-02-PLAN.md -- Fix CLAUDE.md version drift and stale documentation counts

### Phase 43: Clash Detection Integration
**Goal**: Task-as-PR workflow prevents file conflicts between parallel worktrees via hooks and automatic context setup
**Depends on**: Phase 40 (worktree creation needs pheromone injection)
**Requirements**: CLASH-01, CLASH-02, CLASH-03
**Success Criteria** (what must be TRUE):
  1. Editing a file that is modified in another active worktree triggers a PreToolUse hook that blocks the edit with a clear message
  2. `_worktree_create` automatically copies colony context (COLONY_STATE.json, pheromones.json) and runs pheromone-snapshot-inject
  3. `.gitattributes` merge driver resolves package-lock.json conflicts by keeping "ours"
  4. `.aether/data/` files are on the allowlist -- never trigger clash detection (branch-local state)
**Existing work**: clash-detect.sh, clash-pre-tool-use.js hook, worktree.sh, merge-driver-lockfile.sh; 4 test files (~1,036 lines total)
**Plans:** 1/2 plans executed

Plans:
- [x] 43-01-PLAN.md -- Wire clash-detect.sh and worktree.sh into aether-utils.sh dispatcher (source lines, dispatch cases, help JSON)
- [ ] 43-02-PLAN.md -- Wire clash detection and merge driver setup into /ant:init (Step 7.6, read-only list fix, hook verification)

---

### Phase 44: Release Hygiene & Ship
**Goal**: Published package is clean of dev artifacts, all tests pass, and v2.7.0 ships to npm
**Depends on**: Phases 39-43 (all features must be complete)
**Requirements**: REL-01, REL-02, REL-03, TEST-01, TEST-02
**Success Criteria** (what must be TRUE):
  1. `npm pack --dry-run` output contains no test data, worktree references, colony state, or dev artifacts
  2. `bin/validate-package.sh` passes with zero warnings
  3. `npm test` shows 620+ passing tests with zero failures
  4. `npm install -g . && aether --version` succeeds on a clean machine
  5. CLAUDE.md updated with v2.7 changes, version bumped to v2.7.0
**Existing work**: validate-package.sh exists; .npmignore covers most paths; new v2.7 modules may have added untracked file types

---

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
| 33. Input Escaping & Atomic Write Safety | v2.6 | Complete    | Complete | 2026-03-29 | - |
| 34. Cross-Colony Isolation | v2.6 | 2/5 | Complete    | 2026-03-29 |
| 35. Colony Depth & Model Routing | v2.6 | 0/TBD | Complete    | 2026-03-29 |
| 36. YAML Command Generator | v2.6 | 2/4 | Complete    | 2026-03-29 |
| 37. XML Core Integration | v2.6 | 3/3 | Complete | 2026-03-29 |
| 38. Cleanup & Maintenance | v2.6 | 1/2 | Complete    | 2026-03-29 |
| 39. State Safety | v2.7 | 0/2 | Pending | -- |
| 40. Pheromone Propagation | v2.7 | 1/1 | Complete   | 2026-03-30 |
| 41. Midden Collection | v2.7 | 0/0 | Complete    | 2026-03-30 |
| 42. CI Context Assembly | v2.7 | 0/2 | Pending | -- |
| 42.1 Release Hygiene | v2.7 | 1/2 | Complete    | 2026-03-31 |
| 43. Clash Detection Integration | v2.7 | 1/2 | In Progress|  |
| 44. Release Hygiene & Ship | v2.7 | 0/0 | Pending | -- |
