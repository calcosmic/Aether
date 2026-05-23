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
- **v1.23 Daily Driver Reliability** - Phases 145-151 (shipped 2026-05-21) — [Archive](milestones/v1.23-ROADMAP.md)

## Phases

### v1.24 [Next Milestone] (Planned)

*Next milestone not yet defined. Run `/gsd-new-milestone` to start planning.*

<details>
<summary>v1.23 Daily Driver Reliability (Phases 145-151) — SHIPPED 2026-05-21</summary>

- [x] Phase 145: Silent Pipeline Fix (4/4 plans) — completed 2026-05-20
- [x] Phase 146: Command Classification (4/4 plans) — completed 2026-05-21
- [x] Phase 147: Critical Test Coverage (4/4 plans) — completed 2026-05-21
- [x] Phase 148: Learning and Workflow Restoration (4/4 plans) — completed 2026-05-21
- [x] Phase 149: Queen Execution Policy (4/4 plans) — completed 2026-05-21
- [x] Phase 150: Runtime Safety (4/4 plans) — completed 2026-05-21
- [x] Phase 151: End-to-End Proof (3/3 plans) — completed 2026-05-21

</details>

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

### Phase 152: Boundary & Parity
**Goal:** The architecture boundary is documented, Classic behaviour is catalogued, and extraction targets are identified.
**Depends on:** Phase 151 (v1.23 completion)
**Requirements:** BOUNDARY-01, BOUNDARY-02, BOUNDARY-03, BOUNDARY-04
**Success Criteria** (what must be TRUE):
  1. User can read `docs/ARCHITECTURE_BOUNDARY.md` and understand what stays in Go, what moves to TS, and what moves to Markdown/YAML/JSON
  2. User can read `docs/PARITY_CLASSIC_VS_GO.md` and see 15+ verifiable items covering init ceremony, queen loads, worker spawn, planner, oracle loop, scout, build wave, watcher, gatekeeper, probe, memory, lessons, skills, events, recovery, and cleanup
  3. User can read `docs/BEHAVIOUR_EXTRACTION_AUDIT.md` and see every Go symbol containing agent/prompt/phase/skill/memory/ritual/ceremony logic classified as KEEP_IN_GO, MOVE_TO_TS, MOVE_TO_YAML, MOVE_TO_MARKDOWN, MOVE_TO_JSON, DELETE, or UNKNOWN
  4. The hard rule "Compiled code may execute behaviour, but editable assets must define behaviour" is documented and referenced in all three docs
**Plans**: 2 plans (all complete)

### Phase 153: TypeScript Scaffold & Schemas
**Goal:** The TypeScript control plane project exists with validated schemas for all colony asset types.
**Depends on:** Phase 152
**Requirements:** CONTROL-01, SCHEMA-01, SCHEMA-02, SCHEMA-03, SCHEMA-04
**Success Criteria** (what must be TRUE):
  1. User can run `npm install` in `control-ts/` and the project builds without errors
  2. User can run `npm run test:schemas` and all Zod schemas validate against sample YAML files
  3. Agent schema validates id, role, prompt_file, allowed_tools, and confirms referenced prompt files exist
  4. Phase schema validates id, entry_agent, required_agents, inputs, outputs, success_criteria, failure_policy, and ceremony
  5. Event schema validates NDJSON shape (type, timestamp, payload)
  6. Policy schema validates model-routing, memory-rules, skill-creation, and safety-gates
**Plans**: 2 plans (complete)

**Wave 1**
- [x] 153-01-PLAN.md — Bootstrap `control-ts/` project scaffold with package.json, tsconfig, vitest config, and sample fixtures

**Wave 2**
- [x] 153-02-PLAN.md — Implement Zod schemas for agents, phases, events, policies and validate against fixtures

### Phase 154: Colony Assets
**Goal:** All living behaviour is defined in editable YAML and Markdown files under `colony/`.
**Depends on:** Phase 152
**Requirements:** EXTRACT-01, EXTRACT-02, EXTRACT-03, EXTRACT-04, EXTRACT-05, EXTRACT-06
**Success Criteria** (what must be TRUE):
  1. User can run `cat colony/agents/queen.yaml` (and 7 other agent files) and see complete agent definitions
  2. User can run `cat colony/prompts/builder.md` (and other prompt files) and see complete prompt text
  3. User can run `cat colony/phases/init.yaml` (and other phase files) and see complete phase definitions
  4. User can run `cat colony/playbooks/build.md` (and other playbook files) and see complete playbook content
  5. User can run `cat colony/policies/model-routing.yaml` (and other policy files) and see routing and memory rules
  6. No agent personality, prompt text, phase ritual, playbook, model-routing policy, or memory rule exists only in compiled Go code
**Plans**: TBD

### Phase 155: Go Boundary Refactor
**Goal:** Go runtime loads behaviour from files instead of hardcoded strings, while preserving its role as the runtime spine.
**Depends on:** Phase 154
**Requirements:** EXTRACT-07
**Success Criteria** (what must be TRUE):
  1. User can run `go test ./...` and all tests pass after extraction (no regressions)
  2. User can inspect Go source and see that previously hardcoded agent/prompt/phase strings now load from `colony/` or `.aether/` files at runtime
  3. Go commands that previously embedded behaviour inline now reference file paths with fallback/error text only
  4. The Go binary still builds and `aether version` reports correctly
**Plans**: TBD

### Phase 156: TS Control Plane Core
**Goal:** TypeScript control plane can load colony assets and execute a single phase and a full plan sequence.
**Depends on:** Phase 153, Phase 155
**Requirements:** CONTROL-02, CONTROL-03, CONTROL-04, CONTROL-05, CONTROL-06, CONTROL-09, CONTROL-10
**Success Criteria** (what must be TRUE):
  1. User can run a script that calls `loadAgents.ts` and sees all 8 core agents loaded from YAML with validated schemas
  2. User can run a script that calls `loadPhases.ts` and sees all phase definitions loaded from YAML with validated schemas
  3. User can run a script that calls `assemblePrompt.ts` and sees a complete prompt assembled from Markdown fragments
  4. User can run `runPhase.ts` with a phase ID and see it load the phase definition, select the entry agent, and emit events
  5. User can run `executePlan.ts` and see a full sequence execute: init → plan → build → verify → seal
  6. User can run a script that loads skills from `control-ts/src/skills/` and reads/writes memory via `control-ts/src/memory/`
**Plans**: TBD

### Phase 157: TS Adapters & Oracle
**Goal:** Platform adapter stubs and Oracle loop stub exist in TypeScript.
**Depends on:** Phase 156
**Requirements:** CONTROL-07, CONTROL-08
**Success Criteria** (what must be TRUE):
  1. User can inspect `control-ts/src/adapters/` and see stub implementations for Claude Code, Codex, OpenCode, and MCP
  2. User can inspect `control-ts/src/oracle/` and see confidence evaluation and research planning stubs
  3. Adapter stubs have defined interfaces that match the platform dispatch patterns used in the Go runtime
  4. Oracle stub can accept a query, evaluate confidence, and return a research plan object
**Plans**: TBD

### Phase 158: Event Stream
**Goal:** NDJSON event stream is the shared observable truth between Go and TypeScript.
**Depends on:** Phase 155, Phase 156
**Requirements:** EVENT-01, EVENT-02, EVENT-03, EVENT-04
**Success Criteria** (what must be TRUE):
  1. User can run `tail -f .aether/events/current.ndjson` and see JSON lines appear during colony operations
  2. Every phase start, agent selection, task planned, worker spawned, tool call, verification result, memory update, and run seal produces a machine-readable JSON line with type, timestamp, and structured payload
  3. Go runtime writes NDJSON lines instead of prose-only logs where behaviour is observable
  4. TypeScript control plane can append to and read from the event stream concurrently with Go
**Plans**: TBD

### Phase 159: End-to-End Acceptance
**Goal:** The hybrid system is validated: no hardcoded behaviour remains, Classic parity is partially verified, and a demo flow runs end-to-end.
**Depends on:** Phase 157, Phase 158
**Requirements:** TEST-01, TEST-02, TEST-03, TEST-04, TEST-05, TEST-06, TEST-07
**Success Criteria** (what must be TRUE):
  1. User can run `go test ./...` and all tests pass (no regressions in runtime spine)
  2. User can run `npm run test:schemas` and all Zod schemas validate against sample YAML files
  3. User can run `npm run test:control` and the phase runner and orchestrator tests pass
  4. User can run `npm run aether:control -- --task "create a small test file and verify it"` and see the task complete with events for each lifecycle step
  5. A grep/audit confirms no prompt text is hardcoded in Go except fallback/error text
  6. The extraction audit confirms no agent behaviour exists only in Go
  7. The Classic parity checklist exists and at least 50% of items are verifiable against the new hybrid system
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 145 → 146 → 147 → 148 → 149 → 150 → 151 → 152 → 153 → 154 → 155 → 156 → 157 → 158 → 159

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 145. Silent Pipeline Fix | v1.23 | 4/4 | Complete    | 2026-05-20 |
| 146. Command Classification | v1.23 | 4/4 | Complete    | 2026-05-21 |
| 147. Critical Test Coverage | v1.23 | 4/4 | Complete    | 2026-05-21 |
| 148. Learning and Workflow Restoration | v1.23 | 4/4 | Complete    | 2026-05-21 |
| 149. Queen Execution Policy | v1.23 | 4/4 | Complete    | 2026-05-21 |
| 150. Runtime Safety | v1.23 | 4/4 | Complete    | 2026-05-21 |
| 151. End-to-End Proof | v1.23 | 3/3 | Complete    | 2026-05-21 |
| 152. Boundary & Parity | v1.24 | 2/2 | Complete | 2026-05-22 |
| 153. TS Scaffold & Schemas | v1.24 | 1/2 | In progress | 2026-05-22 |
| 154. Colony Assets | v1.24 | 0/3 | Not started | - |
| 155. Go Boundary Refactor | v1.24 | 0/2 | Not started | - |
| 156. TS Control Plane Core | v1.24 | 0/4 | Not started | - |
| 157. TS Adapters & Oracle | v1.24 | 0/2 | Not started | - |
| 158. Event Stream | v1.24 | 0/2 | Not started | - |
| 159. End-to-End Acceptance | v1.24 | 0/3 | Not started | - |
