# Roadmap: Aether

## Milestones

- ✅ **v1.3 Maintenance & Pheromone Integration** — Phases 1-8 (shipped 2026-03-19)
- [ ] **v2.1 Production Hardening** — Phases 9-16 (in progress)

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

### v2.1 Production Hardening (In Progress)

**Milestone Goal:** Address Oracle audit findings, make Aether genuinely production-ready — deeper planning, verified features, accurate docs, great first-user experience.

- [ ] **Phase 9: Quick Wins** - Six independent reliability fixes with outsized impact
- [ ] **Phase 10: Error Triage** - Classify and fix dangerous error suppressions
- [ ] **Phase 11: Dead Code Deprecation** - Audit and deprecate unused subcommands across all surfaces
- [ ] **Phase 12: State API & Verification** - Centralize state access and harden verification evidence chain
- [ ] **Phase 13: Monolith Modularization** - Extract domain modules from aether-utils.sh
- [ ] **Phase 14: Planning Depth** - Add per-phase research to planning and richer build context
- [ ] **Phase 15: Documentation Accuracy** - Update all docs to match verified post-hardening behavior
- [ ] **Phase 16: Ship** - Verify clean install end-to-end, version bump, publish to npm

## Phase Details

### Phase 9: Quick Wins
**Goal**: Critical reliability bugs are fixed and the test suite is fully green before structural work begins
**Depends on**: Nothing (first phase of v2.1)
**Requirements**: REL-01, REL-02, REL-03, REL-04, REL-05, REL-06
**Success Criteria** (what must be TRUE):
  1. Hive wisdom retrieval returns entries regardless of whether confidence is stored as string or number
  2. Concurrent midden writes do not overwrite each other (PID-scoped temp files)
  3. A corrupted learning-observations.json recovers automatically from template instead of silently killing the memory pipeline
  4. COLONY_STATE.json is backed up before every build-wave, with at most 3 rolling checkpoints retained
  5. Colony-prime emits a visible notice when context is trimmed, and the continue-advance state write goes through a locked subcommand
**Plans**: 2 plans

Plans:
- [ ] 09-01-PLAN.md — Fix data integrity bugs (hive type coercion, midden race condition, learning recovery, state checkpoints)
- [ ] 09-02-PLAN.md — Close state write lock gap and add context trimming notifications

### Phase 10: Error Triage
**Goal**: All 338 error-swallowing patterns are classified, and the ~40 dangerous ones on state-mutation paths are replaced with proper error handling
**Depends on**: Phase 9
**Requirements**: REL-07, REL-08, REL-09
**Success Criteria** (what must be TRUE):
  1. Every `2>/dev/null`, `|| true`, and `|| :` in aether-utils.sh is categorized as intentional, lazy, or dangerous
  2. All dangerous-category suppressions on state-writing paths have been replaced with explicit error handling that surfaces failures
  3. All intentional suppressions carry a `# SUPPRESS:OK` comment explaining why they are safe
  4. The test suite remains green after all error-handling changes (no regressions from removing suppressions)
**Plans**: TBD

Plans:
- [ ] 10-01: TBD
- [ ] 10-02: TBD

### Phase 11: Dead Code Deprecation
**Goal**: Unused subcommands are identified across all three command surfaces and marked with deprecation warnings before any code is moved or removed
**Depends on**: Phase 10
**Requirements**: QUAL-01, QUAL-02, QUAL-03
**Success Criteria** (what must be TRUE):
  1. All 76 suspected-dead subcommands have been audited against `.claude/commands/`, `.opencode/commands/`, and `~/.aether/skills/` (three-surface grep)
  2. Confirmed-dead subcommands emit a deprecation warning to stderr when invoked
  3. Subcommands found to be alive on any surface are documented and removed from the dead list
  4. Code identified as dead-but-useful is extracted into optional utility modules rather than deleted
**Plans**: TBD

Plans:
- [ ] 11-01: TBD
- [ ] 11-02: TBD

### Phase 12: State API & Verification
**Goal**: COLONY_STATE.json access is centralized through a single facade, and the verification chain catches fabricated worker claims
**Depends on**: Phase 11
**Requirements**: QUAL-04, QUAL-08, QUAL-09
**Success Criteria** (what must be TRUE):
  1. A `state-api.sh` module handles all COLONY_STATE.json reads and writes with lock/validate/migrate encapsulated
  2. Test runner exit codes are captured during build-verify and cross-referenced against Watcher verification claims
  3. Builder-claimed file paths are verified against actual filesystem before phase advancement
  4. The pre-existing context-continuity test failure is fixed and the full test suite is green
**Plans**: TBD

Plans:
- [ ] 12-01: TBD
- [ ] 12-02: TBD

### Phase 13: Monolith Modularization
**Goal**: aether-utils.sh is reduced from ~11,000 lines to a slim dispatcher by extracting domain modules into independently testable files
**Depends on**: Phase 12 (state-api facade must exist before domain extraction)
**Requirements**: QUAL-05, QUAL-06, QUAL-07
**Success Criteria** (what must be TRUE):
  1. Pheromone domain (~1,800 lines including colony-prime) lives in `pheromone.sh` and all pheromone subcommands work through the dispatch contract
  2. Learning/instinct domain (~1,200 lines) lives in `learning.sh` and the full memory pipeline functions correctly
  3. Remaining domains (queen, colony, swarm, session, spawn, flag, suggest) are extracted following the same proven pattern
  4. aether-utils.sh is under 2,000 lines (setup, sourcing, case dispatch, shared helpers only)
  5. All 530+ tests pass with the modularized structure
**Plans**: TBD

Plans:
- [ ] 13-01: TBD
- [ ] 13-02: TBD
- [ ] 13-03: TBD

### Phase 14: Planning Depth
**Goal**: Aether produces deeper, more substantive plans and builds by investigating domain knowledge before decomposing tasks
**Depends on**: Phase 13 (needs stable infrastructure)
**Requirements**: UX-01, UX-02
**Success Criteria** (what must be TRUE):
  1. `/ant:plan` includes a per-phase research step where scouts investigate domain knowledge, library docs, and patterns before task decomposition
  2. `/ant:build` incorporates research context into builder prompts so workers have domain understanding, not just task lists
  3. Research depth is bounded (one cycle, not a full Oracle RALF loop) to prevent planning from taking longer than execution
**Plans**: TBD

Plans:
- [ ] 14-01: TBD
- [ ] 14-02: TBD

### Phase 15: Documentation Accuracy
**Goal**: All documentation matches the post-hardening system state with no aspirational claims
**Depends on**: Phase 14 (all code changes complete)
**Requirements**: UX-03, UX-04, UX-05
**Success Criteria** (what must be TRUE):
  1. README.md has accurate feature descriptions, a clear getting-started guide, and correct counts/capabilities
  2. CLAUDE.md reflects the current system state (correct subcommand counts, module structure after extraction, accurate architecture diagram)
  3. All files in `docs/` describe verified behavior only — no aspirational claims remain
  4. Known inaccuracies identified by the Oracle audit (trim order, security gate label, etc.) are all corrected
**Plans**: TBD

Plans:
- [ ] 15-01: TBD
- [ ] 15-02: TBD

### Phase 16: Ship
**Goal**: Aether v2.1.0 is published to npm with a verified clean install experience
**Depends on**: Phase 15
**Requirements**: UX-06, UX-07
**Success Criteria** (what must be TRUE):
  1. A clean `npm install -g aether-colony` followed by `/ant:lay-eggs`, `/ant:init`, `/ant:plan`, `/ant:build` completes without errors on a fresh environment
  2. `bin/validate-package.sh` passes with all content-aware checks for the new module structure
  3. Version is bumped to v2.1.0 and published to npm
**Plans**: TBD

Plans:
- [ ] 16-01: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 9 -> 10 -> 11 -> 12 -> 13 -> 14 -> 15 -> 16

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
| 9. Quick Wins | v2.1 | 0/2 | Not started | - |
| 10. Error Triage | v2.1 | 0/TBD | Not started | - |
| 11. Dead Code Deprecation | v2.1 | 0/TBD | Not started | - |
| 12. State API & Verification | v2.1 | 0/TBD | Not started | - |
| 13. Monolith Modularization | v2.1 | 0/TBD | Not started | - |
| 14. Planning Depth | v2.1 | 0/TBD | Not started | - |
| 15. Documentation Accuracy | v2.1 | 0/TBD | Not started | - |
| 16. Ship | v2.1 | 0/TBD | Not started | - |
