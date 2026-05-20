# Roadmap: Aether

## Milestones

- **v1.0 MVP** - Phases 1-6 (shipped)
- **v1.1 Trusted Context** - Phases 7-11 (shipped)
- **v1.2 Live Dispatch Truth and Recovery** - Phases 12-16 (shipped 2026-04-21)
- **v1.3 Visual Truth and Core Hardening** - Phases 17-24 (shipped 2026-04-21)
- **v1.4 Self-Healing Colony** - Phases 25-30 (completed 2026-04-21)
- **v1.5 Runtime Truth Recovery** - Phases 31-38 (completed 2026-04-23, product v1.0.20)
- **v1.6 Release Pipeline Integrity** - Phases 39-46 (completed 2026-04-24)
- **v1.7 Planning Pipeline Recovery** - Phases 47-48 (completed 2026-04-24)
- **v1.8 Colony Recovery** - Phases 49-51 (shipped 2026-04-25)
- **v1.9 Review Persistence** - Phases 52-56 (shipped 2026-04-26)
- **v1.10 Colony Polish** - Phases 57-69 (shipped 2026-04-28)
- **v1.11 Aether Unification** - Phases 70-79 (shipped 2026-04-30)
- **v1.12 Safe Colony** - Phases 80-87 (shipped 2026-05-01)
- **v1.13 Recovery Hardening & Hive Learning** - Phases 88-92 (shipped 2026-05-03)
- **v1.14 Queen Authority** - Phases 93-99 (shipped 2026-05-04)
- **v1.15 Framework Coherence, Efficiency, and Ship Readiness** - Phases 100-105 (shipped 2026-05-08)
- **v1.16 Hybrid Runtime Boundary and Orchestration Recovery** - Phases 106-111 (shipped 2026-05-13)
- **v1.17 Classic Restoration** - Phases 112-118 (shipped 2026-05-14) — [Archive](milestones/v1.17-ROADMAP.md)
- **v1.18 Hybrid Runtime Parity & Release Gate** - Phases 119-123 (shipped 2026-05-14) — [Archive](milestones/v1.18-ROADMAP.md)
- **v1.19 TypeScript Host Cutover + Oracle Confidence Recovery** - Phases 124-129 (shipped 2026-05-15) — [Archive](milestones/v1.19-ROADMAP.md)
- **v1.20 Host Contract Hardening and Wrapper Reality Check** - Phases 130-135 (shipped 2026-05-15) — [Archive](milestones/v1.20-ROADMAP.md)
- **v1.21 Live Colony** - Phases 136-140 (shipped 2026-05-18) — [Archive](milestones/v1.21-ROADMAP.md)
- **v1.22 Grounded Planning + Ceremony Restore** - Phases 141-144 (shipped 2026-05-19) — [Archive](milestones/v1.22-ROADMAP.md)

## Phases

### v1.23 Daily Driver Reliability (In Progress)

**Milestone Goal:** Restore Aether as a reliable daily-driver system -- prove every command works, fix silent data loss, streamline Queen orchestration, and demonstrate end-to-end in a downstream repo.

- [x] **Phase 145: Silent Pipeline Fix** - Remove error suppression so data-persistence calls fail honestly
- [ ] **Phase 146: Command Classification** - Build reliability matrix and classify every command
- [ ] **Phase 147: Critical Test Coverage** - Add tests for 12 HIGH-severity untested source files
- [ ] **Phase 148: Learning and Workflow Restoration** - Verify and restore the learning lifecycle and flagship workflows
- [ ] **Phase 149: Queen Execution Policy** - Align documentation and spawning with Go runtime behavior
- [ ] **Phase 150: Runtime Safety** - Prevent silent data loss on interrupt and fix worktree safety
- [ ] **Phase 151: End-to-End Proof** - Full lifecycle smoke test in a downstream repo

<details>
<summary>v1.22 Grounded Planning + Ceremony Restore (Phases 141-144) — SHIPPED 2026-05-19</summary>

- [x] Phase 141: Survey Noise Filter + Source Anchors (2/2 plans)
- [x] Phase 142: Grounding Gate + Decision Binding (2/2 plans)
- [x] Phase 143: Build + Plan Ceremony Restore (2/2 plans)
- [x] Phase 144: Regression + Execution Path Cleanup (2/2 plans)

</details>

<details>
<summary>v1.21 Live Colony (Phases 136-140) — SHIPPED 2026-05-18</summary>

- [x] Phase 136: Production dispatch with ceremony (4/4 plans)
- [x] Phase 137: Worker-to-worker spawning (4/4 plans)
- [x] Phase 138: Confidence-driven build iteration (4/4 plans)
- [x] Phase 139: Hive wisdom injection (3/3 plans)
- [x] Phase 140: Hardening and validation (3/3 plans)

</details>

## Phase Details

### Phase 145: Silent Pipeline Fix
**Goal**: All data-persistence CLI calls produce honest errors instead of silently dropping output, and all state writes go through state-mutate to prevent corruption
**Depends on**: Nothing (first phase)
**Requirements**: PIPE-01, PIPE-02, PIPE-03, PIPE-04, PIPE-05
**Success Criteria** (what must be TRUE):
  1. Running any playbook build or continue that triggers learning, midden, pheromone, or memory capture produces visible errors if the underlying CLI call fails -- no silent drops
  2. No playbook or wrapper instruction writes COLONY_STATE.json directly -- all state mutations go through `state-mutate` with targeted jq
  3. Starting a new colony with `/ant-init` clears stale session.json from any prior colony, preventing old decisions from leaking in
  4. Reading pending decisions from any command (flag, plan, council) filters by current session scope -- stale decisions from prior colonies never appear
  5. The event array in colony state is automatically capped at 100 entries at the storage layer -- no caller can overflow it
**Plans**: 4 plans

Plans:
**Wave 1**
- [x] 145-01-PLAN.md — Remove error suppression from data-persistence CLI calls in playbooks

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 145-02-PLAN.md — Add session.json cleanup to init and event cap to storage layer
- [x] 145-03-PLAN.md — Enforce pending-decisions session scoping across all readers
- [x] 145-04-PLAN.md — Replace direct COLONY_STATE.json writes with state-mutate calls

### Phase 146: Command Classification
**Goal**: Every Aether command is classified and documented in a reliability matrix, enabling systematic verification of the full command surface
**Depends on**: Phase 145
**Requirements**: CATALOG-01, CATALOG-02, CATALOG-03
**Success Criteria** (what must be TRUE):
  1. Running `aether audit-catalog` outputs a classification for every registered command (public lifecycle, public utility, internal runtime, alias, or deprecated)
  2. The reliability matrix references historical tags from v1.10 through v1.21 and Classic v5.4.0, showing which commands were reliable in which era
  3. The classification is queryable and machine-readable, enabling downstream automation to verify which commands need tests
**Plans**: TBD

### Phase 147: Critical Test Coverage
**Goal**: The 12 most critical untested source files (handling event bus, queen, instinct, midden, hive, spawn, and autopilot) have test files proving their data-persistence paths work
**Depends on**: Phase 145
**Requirements**: TEST-01, TEST-02, TEST-03
**Success Criteria** (what must be TRUE):
  1. Test files exist for all 12 HIGH-severity source files (eventbus, queen, instinct, midden, hive, spawn, autopilot, flags, council, shelf)
  2. Every public lifecycle command (build, continue, plan, seal, entomb, colonize, status, oracle, swarm) has at least one smoke or fixture test that executes and produces evidence
  3. A test matrix document lists every command, its test file, and test type (smoke, fixture, integration, e2e) -- making coverage gaps immediately visible
**Plans**: TBD

### Phase 148: Learning and Workflow Restoration
**Goal**: The learning extraction lifecycle works end-to-end (hypothesis, validated, disproven), workers receive full context, and all 9 flagship workflows produce correct state and ceremony
**Depends on**: Phase 145, Phase 147
**Requirements**: WORKFLOW-01, WORKFLOW-02, WORKFLOW-03, WORKFLOW-04, WORKFLOW-05, WORKFLOW-06, WORKFLOW-07, WORKFLOW-08, WORKFLOW-09, WORKFLOW-10, WORKFLOW-11
**Success Criteria** (what must be TRUE):
  1. After running `/ant-continue`, learnings are extracted as hypotheses, tracked with evidence, and promoted to instincts with validated/disproven status -- the learning pipeline accumulates real wisdom across phases
  2. Workers spawned during build receive pheromone signals, matched skills, survey data, colony goal, and phase description in their context -- nothing is silently dropped
  3. Oracle findings flow end-to-end: research results become instincts, instincts become learnings, learnings promote to QUEEN.md, and high-confidence instincts reach the Hive Brain
  4. All 9 flagship workflows (build, continue, plan, colonize, autopilot, seal, entomb, swarm, oracle) produce correct state mutations and ceremony output when run sequentially
  5. Autopilot (`aether run`) respects all 10 Classic pause conditions -- it stops appropriately on test failures, critical chaos findings, quality gate failures, and other trigger events
**Plans**: TBD

### Phase 149: Queen Execution Policy
**Goal**: Documentation and spawning instructions accurately reflect what the Go runtime actually does -- no contradictions between what users are told and what the system does
**Depends on**: Phase 145
**Requirements**: QUEEN-01, QUEEN-02, QUEEN-03, QUEEN-04
**Success Criteria** (what must be TRUE):
  1. CLAUDE.md execution mode documentation matches Go runtime VerificationDepth behavior exactly -- a user reading the docs gets accurate expectations about watcher behavior at each depth
  2. Playbook spawning instructions match Go caste relevance system -- no playbook instructs spawning an agent that the runtime would reject, and no playbook skips an agent the runtime requires
  3. Deterministic commands (display, signal, session, admin) never spawn agents regardless of context -- verified by running them and confirming zero agent dispatch
  4. Continue at standard depth spawns only the minimum required agents -- no over-spawning of Auditor, Chaos, or other specialists that belong at heavy depth only
**Plans**: TBD

### Phase 150: Runtime Safety
**Goal**: Completed worker work is never silently lost, worktree branches don't orphan, and provider errors are reported accurately
**Depends on**: Phase 145, Phase 147
**Requirements**: RUNTIME-01, RUNTIME-02, RUNTIME-03, RUNTIME-04, RUNTIME-05
**Success Criteria** (what must be TRUE):
  1. If a build is interrupted after workers complete their tasks but before finalization, the user can run a recovery command that discovers the completed work on disk and records it into colony state -- no code is silently lost
  2. Worktree branches from build waves are merged back at build-complete time, not only at continue-advance -- interrupting after build does not leave invisible branches with valuable code
  3. When a worker process fails due to provider API errors (auth, rate limit), the error message says "provider auth failure" or "rate limited" -- not "parse worker output: no JSON found"
  4. Running `/ant-build` after a prior interrupted build warns about orphaned worktree branches from that prior run, giving the user a chance to recover them
  5. Running `/ant-status` shows whether there are unreconciled worker changes from incomplete builds -- the user always knows if there is unrecorded work
**Plans**: TBD

### Phase 151: End-to-End Proof
**Goal**: Aether works end-to-end in a downstream repo without modifying itself during the run, proving it is a reliable daily-driver tool
**Depends on**: Phase 148, Phase 150
**Requirements**: PROOF-01, PROOF-02, PROOF-03
**Success Criteria** (what must be TRUE):
  1. Running a full colony lifecycle (init, colonize, plan, build, continue, seal, entomb) in a separate downstream repo completes without errors and produces correct state at each step -- Aether drives the entire lifecycle without self-modification
  2. End-to-end tests through the TypeScript host layer prove that build, continue, seal, plan, and colonize work correctly through the host dispatch path -- not just the Go CLI path
  3. All 9 remaining public utility commands (preferences, pause-colony, resume-colony, data-clean, insert-phase, quick, verify-castes, bump-version, maturity) have test files with executable evidence
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 145 → 146 → 147 → 148 → 149 → 150 → 151

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 145. Silent Pipeline Fix | v1.23 | 4/4 | Complete    | 2026-05-20 |
| 146. Command Classification | v1.23 | 0/? | Not started | - |
| 147. Critical Test Coverage | v1.23 | 0/? | Not started | - |
| 148. Learning and Workflow Restoration | v1.23 | 0/? | Not started | - |
| 149. Queen Execution Policy | v1.23 | 0/? | Not started | - |
| 150. Runtime Safety | v1.23 | 0/? | Not started | - |
| 151. End-to-End Proof | v1.23 | 0/? | Not started | - |
