# Roadmap: Aether

## Milestones

- **v1.0 MVP** - Phases 1-6 (shipped)
- **v1.1 Trusted Context** - Phases 7-11 (shipped)
- **v1.2 Live Dispatch Truth and Recovery** - Phases 12-16 (shipped)
- **v1.3 Visual Truth and Core Hardening** - Phases 17-24 (shipped 2026-04-21)
- **v1.4 Self-Healing Colony** - Phases 25-30 (completed 2026-04-21)
- **v1.5 Runtime Truth Recovery** - Phases 31-38 (completed 2026-04-23, product v1.0.20)
- **v1.6 Release Pipeline Integrity** - Phases 39-46 (completed 2026-04-24)
- **v1.7 Planning Pipeline Recovery** - Phases 47-48 (completed 2026-04-24)
- **v1.8 Colony Recovery** - Phases 49-51 (shipped 2026-04-25)
- **v1.9 Review Persistence** - Phases 52-56 (shipped 2026-04-26)

## Phases

<details>
<summary>v1.0 MVP (Phases 1-6) -- SHIPPED</summary>

- Phase 1: Housekeeping and Foundation
- Phase 2: Colony Scope System
- Phase 3: Restore Build Ceremony
- Phase 4: Restore Continue Ceremony
- Phase 5: Living Watch and Status Surfaces
- Phase 6: Pheromone Visibility and Steering

</details>

<details>
<summary>v1.1 Trusted Context (Phases 7-11) -- SHIPPED</summary>

- Phase 7: Context Ledger and Skill Routing Foundation
- Phase 8: Prompt Integrity and Trust Boundaries
- Phase 9: Trust-Weighted Context Assembly
- Phase 10: Curation Spine and Structural Learning
- Phase 11: Competitive Proof Surfaces and Evaluation

</details>

<details>
<summary>v1.2 Live Dispatch Truth and Recovery (Phases 12-16) -- SHIPPED</summary>

- Phase 12: Dispatch Truth Model and Run Scoping
- Phase 13: Live Workflow Visibility Across Colonize, Plan, and Build
- Phase 14: Worker Execution Robustness and Honest Activity Tracking
- Phase 15: Verification-Led Continue and Partial Success
- Phase 16: Recovery, Reconciliation, and Runtime UX Finalization

</details>

<details>
<summary>v1.3 Visual Truth and Core Hardening (Phases 17-24) -- SHIPPED 2026-04-21</summary>

- Phase 17: Slash Command Format Audit
- Phase 18: Visual UX Restoration -- Caste Identity and Spawn Lists
- Phase 19: Visual UX Restoration -- Stage Separators and Ceremony
- Phase 20: Visual UX Restoration -- Emoji Consistency
- Phase 21: Codex CLI Visual Parity
- Phase 22: Core Path Hardening
- Phase 23: Recovery and Continuity
- Phase 24: Full Instrumentation -- Trace Logging

</details>

<details>
<summary>v1.4 Self-Healing Colony (Phases 25-30) -- COMPLETED 2026-04-21</summary>

- Phase 25: Medic Ant Core -- Health diagnosis command, colony data scanner
- Phase 26: Auto-Repair -- Fix common colony data issues with `--fix` flag
- Phase 27: Medic Skill -- Healthy state specification skill file
- Phase 28: Ceremony Integrity -- Verify wrapper/runtime parity
- Phase 29: Trace Diagnostics -- Remote debugging via trace export analysis
- Phase 30: Medic Worker Integration -- Caste integration, auto-spawn

</details>

<details>
<summary>v1.5 Runtime Truth Recovery (Phases 31-38) -- COMPLETED 2026-04-23</summary>

8 phases, 17 plans, 176 commits. P0 runtime truth fixes, continue unblock, dispatch robustness, cleanup, platform parity, v1.0.20 release, codebase hygiene, Nyquist validation. [Full archive -> milestones/v1.5-ROADMAP.md]

</details>

<details>
<summary>v1.6 Release Pipeline Integrity (Phases 39-46) -- COMPLETED 2026-04-24</summary>

8 phases (including inserted 44.1, 44.2), 10 plans. OpenCode agent frontmatter fix, stable publish hardening, dev channel isolation, stale publish detection, release integrity checks, doc alignment, E2E regression coverage, stuck-plan investigation. [Full archive -> milestones/v1.6-ROADMAP.md]

</details>

<details>
<summary>v1.7 Planning Pipeline Recovery (Phases 47-48) -- COMPLETED 2026-04-24</summary>

2 phases. Plan --force recovery, fallback artifact cleanup, scout timeout, E2E recovery test. [Full archive -> milestones/v1.7-ROADMAP.md]

</details>

<details>
<summary>v1.8 Colony Recovery (Phases 49-51) -- SHIPPED 2026-04-25</summary>

3 phases, 6 plans, 17 requirements. Stuck-state scanner (7 detectors), auto-repair pipeline (backup-first, atomic rollback), 10 E2E tests. [Full archive -> milestones/v1.8-ROADMAP.md]

</details>

<details>
<summary>v1.9 Review Persistence (Phases 52-56) -- SHIPPED 2026-04-26</summary>

5 phases, 9 plans, 31 requirements. Continue-review worker reports, 7-domain review ledger CRUD, colony-prime prior-reviews injection, 7 agent definition updates (28 files, 4 surfaces), seal/entomb/status/init lifecycle integration. [Full archive -> milestones/v1.9-ROADMAP.md]

</details>

### v1.10 Colony Polish (Phases 57-65) -- IN PROGRESS

- [x] **Phase 57: QUEEN.md Pipeline Fix** - Data cleanup, dedup, colony-prime wiring, auto-promotion (completed 2026-04-26)
- [ ] **Phase 58: Smart Review Depth** - Auto/light/heavy review modes, --light flag, final-phase override
- [x] **Phase 59: Gate Failure Recovery** - Actionable recovery instructions, incremental gate checking, veto confirmation (completed 2026-04-27)
- [x] **Phase 60: Oracle Loop Fix** - Callback fix, research formulation, depth selection, state management (completed 2026-04-27)
- [x] **Phase 61: Porter Ant** - 26th caste registration, agent definitions across 4 surfaces, seal lifecycle wiring (completed 2026-04-27)
- [x] **Phase 62: Lifecycle Ceremony -- Seal and Init** - Flag blocking, wisdom promotion, pheromone cleanup, rich init research (completed 2026-04-27)
- [x] **Phase 63: Lifecycle Ceremony -- Status, Entomb, Resume** - Version display, wisdom extraction, staleness checks (completed 2026-04-27)
- [x] **Phase 64: Lifecycle Ceremony -- Discuss, Chaos, Oracle, Patrol** - Codebase-aware questioning, auto-flagging, persistence suggestions, health checks (completed 2026-04-27)
- [ ] **Phase 65: Idea Shelving** - Persistent shelf, auto-shelve at seal, surface at init, entomb preservation

### Gap Closure Phases (from v1.10 audit)

- [x] **Phase 66: Continue Flow Fix** - Watcher timeout loops, token burn, stuck-in-continue bug (HIGHEST PRIORITY) (completed 2026-04-27)
- [x] **Phase 67: Seal Hive Brain Wiring** - Wire hive-promote into seal (CERE-02, CERE-04), OpenCode parity, close Phase 62 verification gaps (completed 2026-04-27)
- [x] **Phase 68: Gate Recovery Verification** - Fix CR-01/WR-01/WR-02 bugs, verify subcommands, create Phase 59 VERIFICATION.md (planned 2026-04-28)
- [x] **Phase 69: Idea Shelving Verification** - Verify SHELF-01 through SHELF-05, create Phase 65 VERIFICATION.md, execute validation wave 0 (completed 2026-04-28)

## Phase Details

### Phase 57: QUEEN.md Pipeline Fix
**Goal**: The QUEEN.md wisdom pipeline is fully wired -- no duplicate entries, global wisdom reaches all workers, and high-confidence instincts promote automatically
**Depends on**: Nothing (first phase in milestone -- foundational)
**Requirements**: QUEE-01, QUEE-02, QUEE-03, QUEE-04, QUEE-05, QUEE-06, QUEE-07
**Success Criteria** (what must be TRUE):
  1. Running `queen-seed-from-hive` twice reports 0 new entries on the second run (no duplicates)
  2. Colony-prime worker prompt includes global QUEEN.md wisdom, Philosophies, and Anti-Patterns sections alongside local wisdom
  3. `queen-promote-instinct` writes to global `~/.aether/QUEEN.md` so promoted instincts reach all colonies
  4. Running `/ant-seal` automatically promotes instincts with confidence >= 0.8 to QUEEN.md without manual commands
  5. Hive wisdom test entry and all ~270 duplicate `<repo> wisdom` lines are removed from QUEEN.md
**Plans**: 3 plans

Plans:
- [x] 57-01-PLAN.md -- Dedup foundation and readQUEENMd section extension (QUEE-02, QUEE-05)
- [x] 57-02-PLAN.md -- Seed-from-hive filtering and promote-instinct global write (QUEE-03, QUEE-06)
- [x] 57-03-PLAN.md -- Colony-prime global QUEEN.md injection and seal auto-promotion (QUEE-01, QUEE-04, QUEE-07)

### Phase 58: Smart Review Depth
**Goal**: Intermediate phases get fast, light review while final phases and security-sensitive phases always get full review -- saving time without sacrificing safety
**Depends on**: Phase 57 (colony-prime changes in QUEEN.md fix affect same code area)
**Requirements**: DEPTH-01, DEPTH-02, DEPTH-03, DEPTH-04, DEPTH-05, DEPTH-06
**Success Criteria** (what must be TRUE):
  1. Running `/ant-build` on an intermediate phase skips Auditor, Gatekeeper, Probe, Weaver, Medic, Measurer, and Chaos by default
  2. Running `/ant-build` on the final phase always runs the full review gauntlet regardless of any flags
  3. Phases with security or release keywords in their name automatically get heavy review
  4. User sees a review depth message like "Review depth: light (Phase 3 of 7)" in wrapper output
  5. `--light` flag is accepted by build and continue commands but cannot override final-phase heavy review
**Plans**: 2 plans

Plans:
- [x] 58-01-PLAN.md -- Depth resolution logic, tests, and CLI flags (DEPTH-01, DEPTH-02, DEPTH-03, DEPTH-04)
- [x] 58-02-PLAN.md -- Build/continue dispatch filtering, visual display, colony-prime injection (DEPTH-02, DEPTH-05, DEPTH-06)

### Phase 59: Gate Failure Recovery
**Goal**: When verification gates fail, the user gets clear recovery instructions and can fix and re-check only what failed -- no more starting from scratch
**Depends on**: Phase 58 (review depth changes affect same continue verification flow)
**Requirements**: GATE-01, GATE-02, GATE-03
**Success Criteria** (what must be TRUE):
  1. When a verification gate fails, the output shows specific recovery instructions (not just "FAILED")
  2. Watcher Veto asks for explicit user confirmation before stashing work -- no silent auto-stash
  3. Re-running `/ant-continue` after a gate failure only re-checks the previously failed gates, not all gates
**Plans**: 2 plans

Plans:
- [x] 59-01-PLAN.md -- Gate result types, recovery templates, skip logic in Go runtime (GATE-01, GATE-03)
- [x] 59-02-PLAN.md -- Playbook recovery templates, incremental gate checking, veto confirmation (GATE-01, GATE-02, GATE-03)

### Phase 60: Oracle Loop Fix
**Goal**: The Oracle has a working research formulation step, depth selection, and proper state management -- it produces deep research, not shallow one-shots
**Depends on**: Nothing (standalone, no dependency on prior phases)
**Requirements**: ORCL-01, ORCL-02, ORCL-03, ORCL-04
**Success Criteria** (what must be TRUE):
  1. Oracle research begins with a formulated research brief that provides context before iterative research starts
  2. User can choose research depth (quick, balanced, deep, exhaustive) before the Oracle starts working
  3. Research state (configuration, research gaps, synthesis, progress tracking) persists across Oracle iterations
  4. OpenCode worker callback uses the correct messaging endpoint (not LiteLLM proxy)
**Plans**: 3 plans

Plans:
- [x] 60-01-PLAN.md -- OpenCode callback URL env var override (ORCL-01)
- [x] 60-02-PLAN.md -- Research brief formulation and depth selection (ORCL-02, ORCL-03)
- [x] 60-03-PLAN.md -- Smart question formulation replacing lowest-confidence heuristic (ORCL-04)

### Phase 61: Porter Ant
**Goal**: Aether has a Porter ant (26th caste) that surfaces interactive publish/push/deploy options after seal -- delivery is part of the lifecycle, not a separate manual step
**Depends on**: Nothing (caste registration is independent; seal integration can coexist with Phase 62 changes)
**Requirements**: PORT-01, PORT-02, PORT-03, PORT-04, PORT-05
**Success Criteria** (what must be TRUE):
  1. Porter appears as the 26th caste with correct emoji, color, label, and name prefixes in all visual maps
  2. Porter agent definition files exist across all 4 surfaces (Claude, agents-claude mirror, OpenCode, Codex TOML)
  3. After seal completes, Porter prompts the user interactively with publish/push/deploy options
  4. `/ant-porter` command exists in YAML source and all platform wrappers
  5. `porter check` subcommand reports pipeline alignment and readiness
**Plans**: 3 plans

Plans:
- [x] 61-01-PLAN.md -- Caste registration, Gatekeeper emoji swap, count bumps (PORT-01)
- [x] 61-02-PLAN.md -- Agent definitions 4 surfaces, /ant-porter command (PORT-02, PORT-04)
- [x] 61-03-PLAN.md -- Porter check subcommand, seal lifecycle wiring (PORT-03, PORT-05)

### Phase 62: Lifecycle Ceremony -- Seal and Init
**Goal**: Seal and init have real ceremony -- seal blocks on active blockers, promotes wisdom, cleans pheromones, and enriches the archive; init researches the codebase deeply before planning started
**Depends on**: Phase 57 (QUEE-06/07 auto-promotion wiring needed for CERE-02 seal hive promotion)
**Requirements**: CERE-01, CERE-02, CERE-03, CERE-04, CERE-05
**Success Criteria** (what must be TRUE):
  1. Running `/ant-seal` with active blocker-severity flags blocks completion (with `--force` override available)
  2. Seal automatically promotes instincts with confidence >= 0.8 to Hive Brain (non-blocking -- failures logged but don't stop)
  3. Seal expires all FOCUS pheromones while preserving REDIRECT pheromones (hard constraints survive)
  4. CROWNED-ANTHILL.md includes learnings count, promoted instincts count, expired signals, and flags resolved
  5. Running `/ant-init` provides deeper codebase analysis -- reads README, scans directory structure, detects test frameworks, checks CI configs, reads key source files
**Plans**: 3 plans

### Phase 63: Lifecycle Ceremony -- Status, Entomb, Resume
**Goal**: Status shows runtime context, entomb extracts near-miss wisdom and cleans up properly, and resume detects stale state that could mislead
**Depends on**: Phase 62 (seal ceremony should be established before entomb and resume changes)
**Requirements**: CERE-06, CERE-07, CERE-08
**Success Criteria** (what must be TRUE):
  1. `/ant-status` dashboard shows runtime version line and a one-line signal summary
  2. `/ant-entomb` extracts near-miss wisdom (confidence 0.5-0.8), cleans temp files (spawn trees, manifests, review artifacts), and updates registry to inactive with final stats
  3. `/ant-resume` detects stale FOCUS pheromones referencing completed phases and suggests review
**Plans**: 3 plans
  - **Wave 1** (Plans 01, 02): Status version/signal summary + SourcePhase field (01) | Entomb near-miss extraction, temp sweep, registry stats (02)
  - **Wave 2** *(blocked on Wave 1 completion)* (Plan 03): Resume stale FOCUS detection with wrapper-runtime contract

Plans:
- [x] 63-01-PLAN.md -- Status version line, signal summary, SourcePhase field (CERE-06)
- [x] 63-02-PLAN.md -- Entomb near-miss extraction, temp sweep, registry final stats (CERE-07)
- [x] 63-03-PLAN.md -- Resume stale FOCUS detection with wrapper-runtime contract (CERE-08)

### Phase 64: Lifecycle Ceremony -- Discuss, Chaos, Oracle, Patrol
**Goal**: Discuss/council asks comprehensive codebase-aware questions, chaos auto-flags findings, oracle suggests persisting research, and patrol does active health checks
**Depends on**: Phase 63 (wrapper-level changes, no hard dependency but follows lifecycle ceremony progression)
**Requirements**: CERE-09, CERE-10, CERE-11, CERE-12
**Success Criteria** (what must be TRUE):
  1. Running `/ant-discuss` or `/ant-council` analyzes the codebase first, then asks comprehensive multiple-choice questions covering features, priorities, scope, trade-offs, and architecture
  2. Running `/ant-chaos` auto-flags HIGH severity findings and suggests REDIRECT for recurring midden patterns
  3. Running `/ant-oracle` suggests persisting high-value research findings as pheromone signals or hive wisdom entries
  4. Running `/ant-patrol` detects stale pheromones, verifies data file integrity (valid JSON), and checks for interrupted builds
**Plans**: 3 plans
  - **Wave 1** (Plans 01, 02, 03): All parallel -- discuss-analyze Go subcommand + wrappers (01) | chaos + oracle wrapper enhancements (02) | patrol-check Go subcommand + wrapper (03)

Plans:
- [x] 64-01-PLAN.md -- discuss-analyze subcommand and discuss/council wrapper wiring (CERE-09)
- [x] 64-02-PLAN.md -- Chaos midden recurrence + oracle persistence wrapper enhancements (CERE-10, CERE-11)
- [x] 64-03-PLAN.md -- patrol-check subcommand with JSON validity, stale detection, interrupted builds (CERE-12)


### Phase 64.1: Fix continue watcher timeout blocking advancement (INSERTED)

**Goal:** External continue watcher timeout is treated as advisory (warning) rather than blocking when runtime verification steps passed; actual watcher failures still block
**Requirements**: FIX-01, FIX-02, FIX-03
**Depends on:** Phase 64
**Plans:** 0/1 plans complete

Plans:
- [ ] 64.1-01-PLAN.md -- Fix attachExternalContinueWatcher timeout advisory behavior (FIX-01, FIX-02, FIX-03)


### Phase 65: Idea Shelving
**Goal**: Colonies have continuity -- promising ideas get shelved at seal, surface at init, recurring REDIRECTs become permanent guidance, and shelved ideas survive entomb
**Depends on**: Phase 62 (seal ceremony integration), Phase 63 (entomb preservation)
**Requirements**: SHELF-01, SHELF-02, SHELF-03, SHELF-04, SHELF-05
**Success Criteria** (what must be TRUE):
  1. A persistent shelf file (`.aether/data/shelf.json`) stores deferred ideas with trigger conditions and metadata
  2. Running `/ant-seal` automatically shelves promising but unimplemented ideas (low-confidence instincts, unaddressed pheromones, user-mentioned ideas)
  3. Running `/ant-init` surfaces relevant shelved ideas and lets the user promote them to the new colony or defer again
  4. REDIRECT pheromones recurring across 2+ phases (same content hash) get auto-shelved as permanent guidance
  5. Shelved ideas survive `/ant-entomb` -- archived to chambers, not lost
**Plans**: TBD


### Phase 66: Continue Flow Fix
**Goal**: The continue command stops getting stuck in watcher timeout loops and burning tokens — it advances cleanly when runtime verification passes, only blocking on genuine failures
**Depends on**: Nothing (standalone fix, highest priority)
**Requirements**: FIX-01, FIX-02, FIX-03
**Success Criteria** (what must be TRUE):
  1. External watcher timeout is treated as advisory (warning, not blocker) when runtime verification steps all passed
  2. Continue does not re-run already-passed gates from scratch on retry
  3. No infinite retry loops — the command terminates with a clear result (pass, fail, or partial) within bounded time
  4. Resume stale FOCUS detection does not incorrectly flag signals from partial continues
**Plans**: 2 plans
  - **Wave 1** (Plans 01, 02): Parallel -- internal watcher advisory fix + context deadline detection (FIX-01) | stale FOCUS guard + incremental gate verification (FIX-02, FIX-03)

Plans:
- [x] 66-01-PLAN.md -- Fix internal watcher timeout advisory and context deadline detection (FIX-01)
- [x] 66-02-PLAN.md -- Verify incremental gate checking and harden stale FOCUS detection (FIX-02, FIX-03)


### Phase 67: Seal Hive Brain Wiring
**Goal**: Seal ceremony fully wires into Hive Brain — high-confidence instincts promote automatically, not just to local QUEEN.md
**Depends on**: Phase 57 (QUEE-06/07 auto-promotion wiring), Phase 65 (shelf detection integration)
**Requirements**: CERE-02, CERE-04 (CERE-01, CERE-03, CERE-05 already delivered in Phase 62)
**Success Criteria** (what must be TRUE):
  1. Running `/ant-seal` calls hive-promote for instincts with confidence >= 0.8 (non-blocking -- failures logged)
  2. OpenCode seal.md matches Claude seal.md — no parity drift
  3. CROWNED-ANTHILL.md Colony Statistics table includes Hive-promoted count (building on Phase 62 metrics)
  4. Phase 62 VERIFICATION.md updated to reflect all gaps closed
**Plans**: 2 plans

Plans:
- [x] 67-01-PLAN.md -- Extract promoteToHive, wire into sealCmd, update enrichment (CERE-02, CERE-04)
- [x] 67-02-PLAN.md -- Fix wrapper parity, update Phase 62 VERIFICATION.md (CERE-04 verification closure)


### Phase 68: Gate Recovery Verification
**Goal**: Phase 59 gate recovery subcommands are verified as implemented and playbooks integrate correctly with the Go runtime
**Depends on**: Nothing (verification-only phase)
**Requirements**: GATE-01, GATE-02, GATE-03
**Success Criteria** (what must be TRUE):
  1. gate-results-read, gate-results-write, should-skip-gate, gate-recovery-template subcommands exist in the binary
  2. Continue playbooks (continue-verify.md, continue-gates.md) correctly call runtime subcommands
  3. Phase 59 VERIFICATION.md exists and all GATE requirements verified
**Plans**: 2 plans

Plans:
- [x] 68-01-PLAN.md -- Fix CR-01/WR-01/WR-02 bugs and update ROADMAP
- [x] 68-02-PLAN.md -- Create Phase 59 VERIFICATION.md with evidence


### Phase 69: Idea Shelving Verification
**Goal**: Phase 65 shelf system verified end-to-end — shelf CRUD, seal auto-detection, init surfacing, entomb preservation all work
**Depends on**: Phase 66 (seal blocked-state fix may affect shelf detection during seal)
**Requirements**: SHELF-01, SHELF-02, SHELF-03, SHELF-04, SHELF-05
**Success Criteria** (what must be TRUE):
  1. Shelf CRUD operations (create, read, promote, archive) work correctly
  2. Seal auto-detects shelf candidates from instincts, pheromones, and user ideas
  3. Init surfaces relevant shelved ideas and offers promotion
  4. Recurring REDIRECT pheromones auto-shelved as permanent guidance
  5. Shelved ideas survive entomb — archived to chambers
  6. Phase 65 VERIFICATION.md exists and VALIDATION.md wave 0 executed
**Plans**: 1 plan

Plans:
- [x] 69-01-PLAN.md -- Collect test/grep evidence and write Phase 65 VERIFICATION.md (SHELF-01 to SHELF-05)

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 57. QUEEN.md Pipeline Fix | v1.10 | 3/3 | Complete    | 2026-04-26 |
| 58. Smart Review Depth | v1.10 | 0/2 | Planned | - |
| 59. Gate Failure Recovery | v1.10 | 2/2 | Complete    | 2026-04-27 |
| 60. Oracle Loop Fix | v1.10 | 3/3 | Complete    | 2026-04-27 |
| 61. Porter Ant | v1.10 | 3/3 | Complete    | 2026-04-27 |
| 62. Lifecycle Ceremony -- Seal and Init | v1.10 | 3/3 | Complete    | 2026-04-27 |
| 63. Lifecycle Ceremony -- Status, Entomb, Resume | v1.10 | 3/3 | Complete    | 2026-04-27 |
| 64. Lifecycle Ceremony -- Discuss, Chaos, Oracle, Patrol | v1.10 | 3/3 | Complete    | 2026-04-27 |
| 65. Idea Shelving | v1.10 | 0/? | Not started | - |
| 66. Continue Flow Fix | v1.10 | 2/2 | Complete    | 2026-04-27 |
| 67. Seal Hive Brain Wiring | v1.10 | 2/2 | Complete    | 2026-04-27 |
| 68. Gate Recovery Verification | v1.10 | 2/2 | Complete    | 2026-04-28 |
| 69. Idea Shelving Verification | v1.10 | 1/1 | Complete    | 2026-04-28 |
