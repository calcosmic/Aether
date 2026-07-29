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
- **v1.24 Hybrid Architecture Salvage** - Phases 152-159 (shipped 2026-05-24) — [Archive](milestones/v1.24-ROADMAP.md)
- **v1.25 Switch It On** - Phases 160-171 (in progress, started 2026-07-25; DRAFT ROADMAP, not yet user-approved)

## Phases

### v1.25 Switch It On (DRAFT — awaiting user approval)

**Milestone Goal:** Make Aether usable daily on an inexpensive model by switching on machinery that already exists and has never run.

**Rebuild history:** the milestone was fully rebuilt once already (58 requirements, 8 phases) after six specialist review agents refuted the original "Working Again" diagnosis (see REQUIREMENTS.md's "Corrections carried forward"). That rebuild then over-corrected and dropped three categories of requirement that no review agent had actually refuted — research feeding planning, lifecycle commands carrying method instead of protocol, and full phase-fit caste selection — plus compressed "Your Eyes Back" (the rich terminal experience) too far. This revision restores all of it. Requirement count moves from 58 to **88**, and the phase count grows from 8 to **12** (160-171) to hold the three restored categories plus a split of the now much larger "Your Eyes Back" work.

**What was restored, and why it doesn't reopen any refuted claim:**
1. **Research Feeds Planning (RESEARCH-01..10, new)** — the user's own specific request, and nothing in the review touched it. Keeping the TS host (a review finding, not a new decision) makes this *easier*: `.aether/ts-host/src/confidence-loop.ts` (250 lines) already implements the target-confidence loop this needs, and `RESEARCH-07` requires using it as-is rather than reimplementing it.
2. **Core Lifecycle Commands (CMD-01..05, new)** — lifecycle wrappers carrying engineering method instead of manifest-parsing protocol. Not refuted; scoped by content, not line count, since the review separately established that a short `oracle.md` is not evidence of hollowing (its loop moved into Go). `CMD-05` is new: it addresses a defect the plan-checker review found in the *original* discarded draft — multiple phases rewriting `build.md` with no stated merge order — by naming a single structural owner and an explicit merge sequence (see the note under Phase 165 below).
3. **Full Colony On Demand (COLONY-01..05, new)** — all 27 castes kept, phase-fit selection. Corrected on restoration: `COLONY-02` no longer references Dream as a caste (it's a command — a review finding). `COLONY-03` is rewritten so the criterion can actually fail (the old "fewer than 27 loaded" framing already passes today and proves nothing). `COLONY-05` is new and decision-shaped: whether Dream becomes a 28th caste is a decision to record, not assumed either way.
4. **Your Eyes Back expanded (SEE-01..14, was 7)** — this is the rich terminal experience the user cares about most, and it was compressed too far in the rebuild. Restored: the rolling worker panel with caste emoji and deterministic name (SEE-08), incremental live progress (SEE-09), the full charter ceremony with genuine approve/edit/cancel (SEE-10), re-init preserving all colony state (SEE-11 — a data-safety requirement with its own test, not a ceremony detail), progressive-disclosure output (SEE-12), zero-context worker task packets (SEE-13), and "what happens next" on every command (SEE-14).
5. **Reclaim expanded (RECLAIM-01..09, was 6)** — added the swarm state quartet (`/ant-swarm` is a flagship workflow whose state machine currently has no caller), the remaining Class A orphans, and `queen-seed-from-hive`.

**Phase count and the SEE split:** SEE alone reached 14 requirements — too broad for one phase to stay independently verifiable. It splits along a real seam: **Phase 168 (Live Visibility)** covers everything that renders *during* a running command — the health meter, the TS host dashboard, the worker panel, incremental progress, next-step guidance, decision blocks, and the two decisions (what a "live panel" honestly means on Claude Code, and why caste identity feels absent) that gate what gets built. **Phase 169 (Charter & Standards)** covers everything that is a *document or standard*, not a live render — the charter ceremony structure, the progressive-disclosure output format, the task-packet format, and the data-safety guarantee that re-init doesn't silently wipe colony state. These test differently (live-behavior observation vs. document structure plus an automated data-safety test) and ship independently, which is why they're two phases rather than one 14-requirement phase or an arbitrary half-and-half split.

**Where `build.md` is owned (CMD-05):** four phases touch `build.md` this milestone (updated 2026-07-29: Phase 163 added as a content-only toucher — a two-bullet dispatch-instruction amendment for manifest-level context, no structural change; verified by the extended wrapper ceremony contract test). **Phase 165 (Core Lifecycle Commands) is the sole structural owner** — it performs the full method-over-protocol rewrite. Phase 160 (Fail Loudly) may only make narrow, isolated edits to fix specific broken CLI call arguments inside it, and must merge *before* Phase 165, so the rewrite starts from corrected calls rather than stale ones. Phase 168 (Live Visibility) may only append a "what happens next" / visual-guidance layer on top of the file Phase 165 already restructured, and must merge *after* it. No other phase edits `build.md`'s structure. This ordering is enforced by the dependency chain below (165 depends on 160; 168 depends on 160 and, for this reason, effectively follows 165 in execution even though its formal dependency is the shared Phase 160 foundation — see the phase detail note).

- [ ] **Phase 160: Fail Loudly** - Seven broken CLI calls fixed, drift detection catches positional-argument drift, stderr suppression removed where load-bearing; `control-ts/` deleted as a standalone opener; `.aether/ts-host/` and `.aether/ts/` explicitly kept and verified untouched. Only narrow, isolated `build.md` call-argument fixes here — see the `build.md` ownership note above
- [ ] **Phase 161: Cheap Models By Design** - DESCOPED 2026-07-28: no automatic model selection (user decision — see phase detail). Residual scope: `colony/` distribution + zero-reader policy file reckoning. Phase 163 executes next; "cheap models" is served by context/clarity work, not allocation
- [ ] **Phase 162: Switch On Learning** - The complete but never-invoked consolidation pipeline runs at phase end and seal; `pkg/learn` vs `pkg/memory` is reconciled; the Hive Brain default is a written decision, not an accident
- [ ] **Phase 163: Context Reaches Workers** - The context capsule, survey, phase research, `suggest-analyze`, and the approved charter reach a wrapper-spawned worker's actual prompt — at manifest level, measured, and inspectable by a person
- [ ] **Phase 164: Research Feeds Planning** - The Queen decides whether a phase needs research and, when it does, a worker runs automatically using the existing `confidence-loop.ts` (kept, not rebuilt) rather than a stub the user pastes findings into
- [ ] **Phase 165: Core Lifecycle Commands** - `init`/`plan`/`build`/`continue` carry engineering method instead of manifest-parsing protocol; sole structural owner of `build.md` this milestone
- [ ] **Phase 166: Full Colony On Demand** - All 27 castes stay available; phase-fit selection is provable, not assumed; whether Dream becomes a 28th caste is decided
- [ ] **Phase 167: Typed Control** - `mode` migrated onto every existing phase before it's required anywhere; the grounding-gate and review-depth keyword-inference sites closed; remaining sites catalogued with a decision each
- [ ] **Phase 168: Your Eyes Back — Live Visibility** - The existing colony health meter and TS host dashboard/swarm-display/next-step guidance get wired into `/ant-status` and builds; what "live panel" means on Claude Code, and why caste identity feels absent, are decided before anything is built against them
- [ ] **Phase 169: Your Eyes Back — Charter & Standards** - The full charter ceremony with genuine approve/edit/cancel; re-init proven by test to preserve all colony state; output and worker task packets follow written standards
- [ ] **Phase 170: Reclaim The Unreachable** - Of 37 subcommands no playbook reconnection recovers (now including the `/ant-swarm` state quartet, the remaining Class A orphans, and `queen-seed-from-hive`), the ones worth having back become reachable; 3,475 duplicate playbook lines deleted
- [ ] **Phase 171: Prove It** - Three real tasks on a cheap model, an interrupted-session resume, a downstream-repo lifecycle run, and a benchmark harness that is fixed to actually invoke Aether or explicitly retired

<details>
<summary>v1.24 Hybrid Architecture Salvage (Phases 152-159) — SHIPPED 2026-05-24</summary>

- [x] Phase 152: Boundary & Parity (2/2 plans) — completed 2026-05-22
- [x] Phase 153: TS Scaffold & Schemas (2/2 plans) — completed 2026-05-22
- [x] Phase 154: Colony Assets (3/3 plans) — completed 2026-05-23
- [x] Phase 155: Go Boundary Refactor (3/3 plans) — completed 2026-05-23
- [x] Phase 156: TS Control Plane Core (3/3 plans) — completed 2026-05-23
- [x] Phase 157: TS Adapters & Oracle (2/2 plans) — completed 2026-05-24
- [x] Phase 158: Event Stream (3/3 plans) — completed 2026-05-24
- [x] Phase 159: End-to-End Acceptance (3/3 plans) — completed 2026-05-24

</details>

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
**Plans**: 3 plans

### Phase 147: Critical Test Coverage
**Goal**: The 12 most critical untested source files (handling event bus, queen, instinct, midden, hive, spawn, and autopilot) have test files proving their data-persistence paths work
**Depends on**: Phase 145
**Requirements**: TEST-01, TEST-02, TEST-03
**Success Criteria** (what must be TRUE):
  1. Test files exist for all 12 HIGH-severity source files (eventbus, queen, instinct, midden, hive, spawn, autopilot, flags, council, shelf)
  2. Every public lifecycle command (build, continue, plan, seal, entomb, colonize, status, oracle, swarm) has at least one smoke or fixture test that executes and produces evidence
  3. A test matrix document lists every command, its test file, and test type (smoke, fixture, integration, e2e) -- making coverage gaps immediately visible
**Plans**: 3 plans

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
**Plans**: 3 plans

### Phase 149: Queen Execution Policy
**Goal**: Documentation and spawning instructions accurately reflect what the Go runtime actually does -- no contradictions between what users are told and what the system does
**Depends on**: Phase 145
**Requirements**: QUEEN-01, QUEEN-02, QUEEN-03, QUEEN-04
**Success Criteria** (what must be TRUE):
  1. CLAUDE.md execution mode documentation matches Go runtime VerificationDepth behavior exactly -- a user reading the docs gets accurate expectations about watcher behavior at each depth
  2. Playbook spawning instructions match Go caste relevance system -- no playbook instructs spawning an agent that the runtime would reject, and no playbook skips an agent the runtime requires
  3. Deterministic commands (display, signal, session, admin) never spawn agents regardless of context -- verified by running them and confirming zero agent dispatch
  4. Continue at standard depth spawns only the minimum required agents -- no over-spawning of Auditor, Chaos, or other specialists that belong at heavy depth only
**Plans**: 3 plans

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
**Plans**: 3 plans

### Phase 151: End-to-End Proof
**Goal**: Aether works end-to-end in a downstream repo without modifying itself during the run, proving it is a reliable daily-driver tool
**Depends on**: Phase 148, Phase 150
**Requirements**: PROOF-01, PROOF-02, PROOF-03
**Success Criteria** (what must be TRUE):
  1. Running a full colony lifecycle (init, colonize, plan, build, continue, seal, entomb) in a separate downstream repo completes without errors and produces correct state at each step -- Aether drives the entire lifecycle without self-modification
  2. End-to-end tests through the TypeScript host layer prove that build, continue, seal, plan, and colonize work correctly through the host dispatch path -- not just the Go CLI path
  3. All 9 remaining public utility commands (preferences, pause-colony, resume-colony, data-clean, insert-phase, quick, verify-castes, bump-version, maturity) have test files with executable evidence
**Plans**: 3 plans

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
**Plans**: 3 plans

Plans:
**Wave 1**
- [x] 154-01-PLAN.md — Create all 27 agent YAMLs and prompt Markdowns, expand AgentSchema to 27 roles
- [x] 154-02-PLAN.md — Create all phase YAMLs and canonical playbook Markdowns

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 154-03-PLAN.md — Create all policy YAMLs, extend PolicySchema, expand policy tests

### Phase 155: Go Boundary Refactor
**Goal:** Go runtime loads behaviour from files instead of hardcoded strings, while preserving its role as the runtime spine.
**Depends on:** Phase 154
**Requirements:** EXTRACT-07
**Success Criteria** (what must be TRUE):
  1. User can run `go test ./...` and all tests pass after extraction (no regressions)
  2. User can inspect Go source and see that previously hardcoded agent/prompt/phase strings now load from `colony/` or `.aether/` files at runtime
  3. Go commands that previously embedded behaviour inline now reference file paths with fallback/error text only
  4. The Go binary still builds and `aether version` reports correctly
**Plans**: 3 plans

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
**Plans**: 3 plans

### Phase 157: TS Adapters & Oracle
**Goal:** Platform adapter stubs and Oracle loop stub exist in TypeScript.
**Depends on:** Phase 156
**Requirements:** CONTROL-07, CONTROL-08
**Success Criteria** (what must be TRUE):
  1. User can inspect `control-ts/src/adapters/` and see stub implementations for Claude Code, Codex, OpenCode, and MCP
  2. User can inspect `control-ts/src/oracle/` and see confidence evaluation and research planning stubs
  3. Adapter stubs have defined interfaces that match the platform dispatch patterns used in the Go runtime
  4. Oracle stub can accept a query, evaluate confidence, and return a research plan object
**Plans**: 3 plans

### Phase 158: Event Stream
**Goal:** NDJSON event stream is the shared observable truth between Go and TypeScript.
**Depends on:** Phase 155, Phase 156
**Requirements:** EVENT-01, EVENT-02, EVENT-03, EVENT-04
**Success Criteria** (what must be TRUE):
  1. User can run `tail -f .aether/events/current.ndjson` and see JSON lines appear during colony operations
  2. Every phase start, agent selection, task planned, worker spawned, tool call, verification result, memory update, and run seal produces a machine-readable JSON line with type, timestamp, and structured payload
  3. Go runtime writes NDJSON lines instead of prose-only logs where behaviour is observable
  4. TypeScript control plane can append to and read from the event stream concurrently with Go
**Plans**: 3 plans

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
**Plans**: 3 plans

### Phase 160: Fail Loudly
**Goal**: Every playbook and wrapper CLI call either succeeds or visibly fails -- no call is silently swallowed by a `/dev/null` redirect -- and the drift-detection test that should have caught this catches positional-argument drift, not only `--flag` drift. This is fixing seven confirmed-broken calls (`survey-load`, `check-antipattern`, `print-next-up`, `verify-claims`, `state-checkpoint`, `generate-progress-bar`, `skill-detect`) and closing a test blind spot, not writing new subsystems. Alongside it, `control-ts/` -- the one component confirmed genuinely dead, self-described in its own `package.json` as a retired prototype -- is deleted as a standalone first commit; `.aether/ts-host/` and `.aether/ts/` are explicitly kept, since deleting either breaks `go build` and hard-fails `aether publish`/`aether integrity`. Any edit this phase makes to `build.md` is limited to fixing a specific broken call's arguments -- the full method rewrite belongs to Phase 165, which depends on this phase completing first.
**Depends on**: Phase 159 (v1.24 completion)
**Requirements**: LOUD-01, LOUD-02, LOUD-03, LOUD-04, LOUD-05, LOUD-06, LOUD-07, LOUD-08, LOUD-09, RETIRE-01, RETIRE-02, RETIRE-03, RETIRE-04
**Success Criteria** (what must be TRUE):
  1. A build or continue run where a previously-broken call fails (any of the seven: survey-load, check-antipattern, print-next-up, verify-claims, state-checkpoint, generate-progress-bar, skill-detect) now shows a visible error in the terminal -- not silence
  2. The Gatekeeper security gate (`check-antipattern`) actually executes during continue, and its pass/fail result visibly affects the continue outcome
  3. *(Code-verifiable only, cannot be confirmed by a non-technical user watching the tool run)*: the drift-detection test fails when a playbook calls a positional-argument command incorrectly, and a full-execution audit -- not regex -- covers every documented CLI call
  4. `.aether/docs/structural-learning-stack.md`, `CLAUDE.md`, and `AGENTS.md` no longer claim `/ant-continue` runs phase-end consolidation today (that claim becomes true only once Phase 162 ships)
  5. Any guidance from `cmd/unblock_cmd.go` points at a command that actually exists and works -- not a slash command available on no platform
  6. `control-ts/` no longer exists in the repository; `.aether/ts-host/` and `.aether/ts/` are unchanged; `go build`, `aether publish`, and `aether integrity` all still succeed after the deletion
  7. *(Code-verifiable only)*: any test removed with `control-ts/` is recorded in a ledger as dead-with-no-replacement, or its coverage is named as continuing in a specific surviving test
**Plans**: 8 plans in 2 waves

Plans:
- [x] 160-01-PLAN.md — RETIRE-04 safety net: Go policy schema test + retired-tests ledger (wave 1)
- [x] 160-02-PLAN.md — LOUD-01/03: correct the six broken documented call sites + survey-load absence test (wave 1)
- [x] 160-03-PLAN.md — LOUD-06/07: correct the consolidation-runs doc claims + stderr-suppression invariant (wave 1)
- [x] 160-04-PLAN.md — LOUD-02: wire check-antipattern into the live continue gate pipeline (wave 1)
- [x] 160-05-PLAN.md — D-03/04/05: worker debug artifacts on every failure mode, capped, worktree-safe (wave 1)
- [x] 160-06-PLAN.md — RETIRE-01/02/03/04: delete control-ts as a standalone commit, prove ts-host/ts survive (wave 2, needs 01)
- [x] 160-07-PLAN.md — LOUD-04/05 + D-01: positional-drift execution audit + gate/enrichment classification (wave 2, needs 02, 04)
- [x] 160-08-PLAN.md — LOUD-08 (D-02): build the /ant-unblock wrapper on Claude + OpenCode (wave 2, needs 03)

### Phase 161: Cheap Models By Design
**Goal**: > **DESCOPED 2026-07-28 (user decision, pre-planning):** automatic model selection is
> rejected. The user's intent behind "usable daily on an inexpensive model" is
> capability — the colony works well when the USER runs it on a cheap model — not
> allocation. Phase 163 (the clarity work that actually serves that intent) executes
> next instead. What survives of this phase: the `colony/` distribution gap and the
> wire-or-delete reckoning for the seven zero-reader policy files (with
> `model-routing.yaml`'s likely fate being deletion or a strictly manual opt-in —
> never default-on routing). Requirements MODEL-01..06 need revision before any
> planning here.
>
> Original goal (superseded): `colony/policies/model-routing.yaml` already maps every caste to a model tier -- builders, watchers, scouts, and surveyors to the cheap tier; oracle, architect, route-setter, and archaeologist to the expensive one. It has zero readers. This phase wires an existing, fully-written YAML to dispatch; it is not writing new routing logic. `colony/` also gets distributed, since `cmd/policy_loader.go:19` currently resolves a relative path and nothing embeds, publishes, or installs `colony/` to `~/.aether/system/colony` -- so every policy silently falls back to compiled defaults the moment Aether runs outside this repo.
**Depends on**: Phase 160
**Requirements**: MODEL-01, MODEL-02, MODEL-03, MODEL-04, MODEL-05, MODEL-06
**Success Criteria** (what must be TRUE):
  1. Starting a build, the user can see which model each spawned worker is using, and that builders/watchers/scouts/surveyors run on the cheap tier while oracle/architect/route-setter/archaeologist run on the expensive tier
  2. Editing `colony/policies/model-routing.yaml` to move a caste to a different model tier changes which model that caste's next worker actually runs on -- no Go rebuild required
  3. Of the six other zero-reader policy files (`autopilot.yaml`, `memory-rules.yaml`, `pheromone-lifecycle.yaml`, `safety-gates.yaml`, `signal-rules.yaml`, `skill-creation.yaml`), each either has a visible runtime effect or is deleted -- `safety-gates.yaml` in particular actually gates something observable
  4. Running Aether from a separate downstream repo still applies the routing and policy decisions from `colony/` -- checked by observing a worker's model/behaviour in that other repo, not only in this one
  5. If `colony/` is ever absent (e.g. a fresh install before distribution completes), dispatch still works using Go's compiled fallback defaults instead of crashing
**Plans**: TBD

### Phase 162: Switch On Learning
**Goal**: `pkg/memory/pipeline.go` already wires Observe -> Promote -> Queen -> Consolidate. It is constructed in exactly two places (`consolidation-phase-end`, `consolidation-seal`) and neither is invoked by any wrapper, playbook, or Go call site -- the colony described in CLAUDE.md has never learned anything. This phase invokes the two subcommands that already exist; it does not build a learning system. It also reconciles the fact that a second, different learning system (`pkg/learn`) already runs live on every continue, and makes the Hive Brain's off-by-default behaviour a deliberate, written decision instead of an accident.
**Depends on**: Phase 160
**Requirements**: LEARN-01, LEARN-02, LEARN-03, LEARN-04, LEARN-05
**Success Criteria** (what must be TRUE):
  1. Running `/ant-continue` at the end of a phase visibly reports learning activity (e.g. observations promoted to instincts) that did not appear before this phase shipped
  2. Running `/ant-seal` visibly reports the full eight-ant consolidation pass, instinct decay, and an archive/report artifact being written
  3. *(Decision-shaped, not build-shaped)*: a written decision states which of `pkg/learn` or `pkg/memory` is authoritative and what happens to the other -- retired, or explicitly subordinate -- documented in the repo, not only implemented
  4. *(Decision-shaped)*: a written decision states whether the Hive Brain default changes from off to on, with the reasoning; if it stays off, the documentation stops implying it is on by default
  5. The same worker task, run once with colony memory populated and once with it wiped, produces demonstrably different output (referencing a prior instinct, wisdom entry, or pattern) -- recorded as a before/after comparison, not assumed
**Plans**: TBD

### Phase 163: Context Reaches Workers
**Goal**: Everything the colony knows demonstrably arrives in a worker's actual prompt, measured and inspectable by a person. *(Goal corrected 2026-07-29 during planning, per Phase 163 D-07 and the phase research: two of the four "confirmed disconnected" items were already reconnected by commits `281dd34a` and `a2c8288e` on 2026-07-26 -- survey findings and phase research reach build workers today. `build-context.md` and `codexBuildPlaybooks()` were deleted in Phase 160 and stay dead.)* What is genuinely missing: the colony-prime context capsule never reaches a wrapper-spawned worker, the approved charter has never reached any worker, and `suggest-analyze` has never executed. This phase carries the capsule at manifest level -- once per build, not an 8K copy per dispatch -- adds charter to the same integrity-assessed pipeline plus a compliance gate, gives `suggest-analyze` its first caller, refines the existing `--print-brief` inspector into a ten-second checklist, and fixes the three worker write-contract contradictions folded in from Phase 160 (the `.aether/data/` guardrail workers were forced to evade, scout's read-only profile versus its own write instruction, and the `{}`-only artifacts schema).
**Depends on**: Phase 160
**Requirements**: CONTEXT-01, CONTEXT-02, CONTEXT-03, CONTEXT-04, CONTEXT-05, CONTEXT-06, CONTEXT-07, CONTEXT-08, CONTEXT-09
**Success Criteria** (what must be TRUE):
  1. A wrapper-spawned build worker receives the colony-prime context capsule, carried once at manifest level, and a named test fails when it stops arriving or starts duplicating per dispatch
  2. Running `aether build <n> --print-brief` shows a sectioned checklist -- each context section present or absent, its size, and the total against a real budget -- with `--full` for the raw assembled prompt
  3. That checklist shows the context capsule, pheromone signals, survey findings (with a staleness warning), phase research, and the approved charter's governance rules; an approved charter is no longer invisible to every worker, and an ignored mechanically-checkable charter rule surfaces as a gate finding at continue
  4. `suggest-analyze` runs on every completed build, exactly once, and its suggestions appear once at closeout with copyable approve and dismiss commands
  5. Total assembled context is measured against the budgets the codebase actually declares (colony-prime 8000/4000, skills 8000, phase research 3500, codegraph 2200) and bounded by an invariant test -- a guard on future additions, not a trimming programme *(the "playbook injection 7K" figure in the original criterion no longer exists; playbook injection was removed entirely)*
  6. A worker told to write planning or research artifacts can do so without evading the guardrail, while colony state stays blocked; scout's permissions match its own brief; and the artifacts schema accepts named typed fields
  7. *(CONTEXT-09, descoped per D-08)*: no staged benchmark. The inspector plus the phase's automated presence and budget tests are the evidence; real-world cheap-model validation happens through the user's own repos after the phase ships
**Plans**: 6 plans

Plans:
- [ ] 163-01-PLAN.md -- CONTEXT-02/03: context capsule reaches wrapper workers, once, at manifest level (wave 1)
- [ ] 163-02-PLAN.md -- CONTEXT-06/D-09: charter as a protected colony-prime section + charter compliance gate wired into continue (wave 1)
- [ ] 163-03-PLAN.md -- D-04/D-05: sanctioned scratch-dir allowlist in the hook, scout permission fix, artifacts schema fix, aligned rules docs (wave 1)
- [ ] 163-04-PLAN.md -- CONTEXT-01/04, D-07/D-10: survey staleness warning, survey+research presence tests, REQUIREMENTS.md rewording (wave 1)
- [ ] 163-05-PLAN.md -- CONTEXT-05/D-11: suggest-analyze gets a live caller at build-finalize, tick-to-approve at closeout (wave 1)
- [ ] 163-06-PLAN.md -- CONTEXT-07/08/09, D-06/D-03/D-08: --print-brief checklist default, --full, budget invariant (wave 2, needs 01, 02, 04)

### Phase 164: Research Feeds Planning
**Goal**: Before a phase is planned, the Queen decides whether it needs research and states why; when it does, a research worker runs automatically and its findings persist and feed the plan directly -- restoring `v5.4.0` `plan.md` Step 3.6 "Phase Domain Research" choreography. Current `plan.md` contains zero references to Oracle; research is a standalone command the user must remember to run and paste in. Nothing in the review refuted this, and keeping the TS host (Phase 160's decision) makes it *easier*, not harder: `.aether/ts-host/src/confidence-loop.ts` (250 lines, with `maxIterations` already wired) is a working target-confidence loop. This phase uses it as-is; it does not reimplement a confidence loop.
**Depends on**: Phase 160, Phase 163 (research findings ride the same manifest-level context path Phase 163 restores, rather than a second parallel assembly)
**Requirements**: RESEARCH-01, RESEARCH-02, RESEARCH-03, RESEARCH-04, RESEARCH-05, RESEARCH-06, RESEARCH-07, RESEARCH-08, RESEARCH-09, RESEARCH-10
**Success Criteria** (what must be TRUE):
  1. Before planning a phase, the Queen states whether that phase needs research and why; the user can override the decision either way, and when research is warranted a worker runs automatically without the user having to invoke it
  2. Research findings persist to a durable per-phase artifact and appear in the planner's context automatically -- not pasted in by hand
  3. Re-running plan on a phase re-researches from scratch rather than reusing stale findings, and territory survey context (where it exists) reaches both the research worker and the planner
  4. During a live plan run, a confidence readout is visible while research runs, using the existing `confidence-loop.ts` rather than a reimplementation, and the loop stops at the depth-bound target/iteration budget (fast 80%/4, balanced 90%/6, deep 95%/8, exhaustive 99%/12) with an accept override to exit early
  5. At plan time, the Queen proposes plan granularity, task decomposition depth, and verification depth with a plain-English reason each, and the user can accept or change any of them before planning proceeds
**Plans**: TBD

### Phase 165: Core Lifecycle Commands
**Goal**: `init`, `plan`, `build`, and `continue` wrappers are rewritten to carry engineering method -- stage purpose, files to read, spawn choreography, stop conditions -- instead of protocol instructions whose primary job is parsing `result.manifest.dispatch_manifest`. Scoped by content, not line count: a short `oracle.md` is not evidence of hollowing, since its RALF loop moved into Go. **This phase is the sole structural owner of `build.md` for this milestone.** Phase 160 may only have made narrow, isolated fixes to specific broken call arguments inside it (merged first); Phase 168 may only append a "what happens next" / visual-guidance layer on top of the file this phase produces (merges after). No other phase restructures it.
**Depends on**: Phase 160, Phase 163, Phase 164 (documents the research step Phase 164 adds and the context injection Phase 163 restores)
**Requirements**: CMD-01, CMD-02, CMD-03, CMD-04, CMD-05
**Success Criteria** (what must be TRUE):
  1. Reading `build.md`, `plan.md`, `continue.md`, and `init.md` shows stage purpose, files-to-read guidance, spawn choreography, and stop conditions -- not instructions whose primary job is parsing an internal JSON envelope
  2. Grepping the four core lifecycle wrappers for instructions to parse an internal JSON envelope or write to a temporary manifest file as their primary job returns zero matches
  3. A user reading `build.md` before running a build can describe what each stage does and what context/research a worker receives, without opening Go source
  4. `/ant-chaos`, `/ant-archaeology`, `/ant-dream`, `/ant-oracle`, `/ant-swarm`, `/ant-sage`, `/ant-colonize`, and `/ant-council` continue to work unchanged
  5. `build.md`'s structural rewrite is committed by this phase alone; the commit or a header comment states what Phase 160 fixed beforehand and reserves the trailer section Phase 168 appends afterward, so the merge order is traceable, not assumed
**Plans**: TBD

### Phase 166: Full Colony On Demand
**Goal**: All 27 castes are markdown and YAML and cost nothing at rest; none are deleted or merged. Dream is a command, not a caste -- corrected on restoration, since the earlier draft's "Sage and Dream at milestone close" language implied a Dream worker that cannot exist. This phase makes phase-fit caste selection actually provable: the old success criterion ("fewer than 27 castes loaded") already passes today and proves nothing, so it is replaced with a criterion that can fail. It also gives `colony/agents/*.yaml` its first Go reader -- today only the deleted `control-ts` ever read it.
**Depends on**: Phase 160
**Requirements**: COLONY-01, COLONY-02, COLONY-03, COLONY-04, COLONY-05
**Success Criteria** (what must be TRUE):
  1. All 27 caste YAMLs in `colony/agents/` still exist -- none deleted or merged
  2. A phase touching legacy code spawns Archaeologist, a hardening phase spawns Chaos, an auth-touching phase spawns Gatekeeper, and milestone close spawns Sage -- observable in the run's dispatch log
  3. A specific, failable test proves phase-fit selection: a legacy-touching phase's dispatch log shows Archaeologist loaded AND Includer not loaded -- replacing the old "fewer than 27 loaded" framing, which would pass trivially today
  4. Adding or editing a caste requires only editing its `colony/agents/*.yaml` file, and that file now has a working Go reader -- verified by editing a caste's YAML and observing the change take effect without a rebuild
  5. *(Decision-shaped)*: a written decision states whether Dream becomes a 28th caste or remains a command, with the reasoning recorded
**Plans**: TBD

### Phase 167: Typed Control
**Goal**: `mode` becomes a required, explicit field on every phase -- but only after a migration backfills it onto every existing phase, since `mode` is currently `omitempty` and untouched by `runMigrateState`; requiring it first would hard-block every colony planned to date, including Aether's own. Beyond `InferPhaseMode`, the two live behavioural sites also stop inferring from phase-name prose: `plan_grounding.go:40` currently exempts a phase from the grounding gate purely because its name contains "research", and `review_depth.go:153` selects review depth the same way. Ten-plus inference sites exist in total; the ones not fixed this phase get a written decision instead of silent continuation.
**Depends on**: Phase 160
**Requirements**: TYPED-01, TYPED-02, TYPED-03, TYPED-04, TYPED-05, TYPED-06, TYPED-07, TYPED-08
**Success Criteria** (what must be TRUE):
  1. Running the migration on this repo's own colony state backfills `mode` on every phase that lacks it, using `InferPhaseMode` once at migration time and writing the result explicitly to disk -- inspectable in the state file afterward, before any validation requiring `mode` is turned on
  2. After migration, planning a new phase without an explicit `mode` fails with a clear, readable error instead of silently inferring one
  3. A phase named or described with "research"/"explore"/"spike" language no longer skips the grounding gate purely because of that name -- the grounding gate applies based on the typed mode
  4. Review depth for a phase is chosen from its typed mode, not inferred from words in its description; where keyword hints are still shown to the author, they are visibly labeled as suggestions, never silently applied
  5. *(Code-verifiable only)*: the remaining prose-to-control-flow sites (`oracle_loop.go:88`/`:154`, `recovery_engine.go:46`, `codex_continue.go:3499`, `hive.go:209`, `codex_visuals.go:222`) are catalogued with a written decision recorded for each -- fix, accept, or defer
  6. A regression test proves a phase description containing "research" still dispatches a Builder when its typed mode says production
**Plans**: TBD

### Phase 168: Your Eyes Back — Live Visibility
**Goal**: Most of what this phase needs already exists. `colony-vital-signs` computes build velocity, error rate, signal health, memory pressure, colony age, and overall health 0-100 -- v5.4.0's `status.md` called it and nothing does now. The TS host already holds `dashboard.ts`, `swarm-display.ts`, and `narrator.ts`. The Go runtime already computes next-step guidance (`nextCommandFromState`, `closeoutNextCommand`, `continueNextCommandForAssessment`) and zero of 60 wrappers surface it. This phase wires all of that into the commands a user actually runs; it does not reimplement any of it. Two things are decided and written down before anything is built against them: what "live worker panel" honestly means inside Claude Code's Task-tool spawn model (no repaintable surface), and why caste identity feels absent when colours already render via `AETHER_FORCE_COLOR=1`. This phase's only edit to `build.md` is appending the next-step-guidance/visual layer on top of Phase 165's already-restructured file -- it does not restructure `build.md` itself.
**Depends on**: Phase 160
**Requirements**: SEE-01, SEE-02, SEE-03, SEE-04, SEE-05, SEE-06, SEE-07, SEE-08, SEE-09, SEE-14
**Success Criteria** (what must be TRUE):
  1. Running `/ant-status` shows a colony health score (0-100) with its component signals (build velocity, error rate, signal health, memory pressure, colony age) -- using the existing `colony-vital-signs` computation, not a new one
  2. Running a build surfaces the existing TS host dashboard/swarm display rather than a bare log
  3. *(Decision-shaped, not build-shaped)*: a written decision states exactly what "live worker panel" means on Claude Code specifically, given the Task-tool spawn model has no repaintable surface -- and that decision, not an assumed animated panel, is what gets built
  4. During a build, a rolling panel shows each worker as it spawns, runs, and completes, with caste emoji, ANSI-coloured label, and deterministic name, in whatever form criterion 3 establishes as achievable; progress is visible incrementally, not only in a final summary
  5. Lifecycle commands show banner-framed stages, structured summaries, and the next-step guidance the Go runtime already computes, at the end of every command
  6. When a command needs a decision from the user, it appears in a visually distinct block, not buried in prose
  7. When context health reaches "replace," a handoff is written and confirmed saved before the tool suggests `/clear`, and `/ant-resume` afterward restores the work without the user re-explaining it
  8. *(Decision-shaped)*: a written diagnosis states the real, specific reason caste identity feels absent to the user -- given colours already render -- before any further fix is scoped or built against it
**Plans**: TBD

### Phase 169: Your Eyes Back — Charter & Standards
**Goal**: Everything in this phase is a document or a standard, not a live render -- which is why it is split from Phase 168. The colony charter ceremony gets its full structure with genuine approve/edit/cancel, re-init is proven by an automated test (not manual observation) to preserve all colony state rather than silently discarding it, and command output plus worker task packets follow explicit, written standards instead of an ad hoc format.
**Depends on**: Phase 160
**Requirements**: SEE-10, SEE-11, SEE-12, SEE-13
**Success Criteria** (what must be TRUE):
  1. The colony charter ceremony presents its full structure (Prior Context, Charter with Intent/Vision/Governance/Goals, Context, Pheromone suggestions) with genuine approve/edit/cancel -- a user can reject or modify what's proposed, not just click through
  2. A dedicated automated test -- not manual observation -- proves re-running init on an existing colony preserves all colony state, wisdom, instincts, learnings, pheromones, and phase progress; this test fails if re-init silently wipes any of them
  3. Command output follows a progressive-disclosure standard: compact single-line summaries by default, with bracketed counts marking expandable detail, and verbose output only on request
  4. Worker task packets are written for a worker with zero prior context -- one action per task, explicit acceptance criteria, and the exact command to verify -- inspectable by reading an actual generated task packet
**Plans**: TBD

### Phase 170: Reclaim The Unreachable
**Goal**: Of 83 subcommands that lost every caller since v5.4.0, 46 return automatically once Phase 163 reconnects the playbooks. These 37 do not -- no caller anywhere, playbook reconnection or otherwise. This phase makes the ones worth having back reachable: the original 6 (checkpoint/rollback, registry population, XML exchange, `recover`, the duplicate full-playbook deletion, and the unread `colony/` file audit), plus the swarm state quartet (`/ant-swarm` is a flagship workflow whose state machine currently has no caller), the remaining Class A orphans, and `queen-seed-from-hive`. Separately, it deletes 3,475 duplicate playbook lines (`build-full.md` + `continue-full.md`, 37% of the total playbook surface).
**Depends on**: Phase 160, Phase 162 (`RECLAIM-09` ties directly to the hive-default decision Phase 162 makes)
**Requirements**: RECLAIM-01, RECLAIM-02, RECLAIM-03, RECLAIM-04, RECLAIM-05, RECLAIM-06, RECLAIM-07, RECLAIM-08, RECLAIM-09
**Success Criteria** (what must be TRUE):
  1. Before a risky auto-repair runs, a checkpoint is visibly created, and a user can roll back to it afterward if the repair made things worse (`autofix-checkpoint`/`autofix-rollback` reachable)
  2. Running `/ant-init` and `/ant-seal` populates the colony registry, checkable via a registry command -- this is what supplies the domain tags used to scope hive wisdom
  3. The XML exchange path either produces a real colony archive at seal (matching v5.4.0 behaviour) or is explicitly retired, with its stale files in `.aether/exchange/` and references in `.aether/docs/xml-utilities.md` removed -- not left half-alive
  4. `aether recover` (`cmd/recover_scanner.go`) is reachable from a documented command path, not only by direct binary invocation
  5. `build-full.md` and `continue-full.md` no longer exist in the repository, and the split playbooks they duplicated still work
  6. Every file in `colony/playbooks/`, `colony/phases/`, and `colony/prompts/` either has a verifiable reader or is deleted -- an audit lists each file's disposition
  7. Running `/ant-swarm` populates and reads real swarm state (`swarm-findings-init`, `swarm-findings-add`, `swarm-solution-set`, `swarm-cleanup`) instead of a state machine with no caller
  8. The remaining Class A orphans (`pheromone-count`, `data-safety-stats`, `clash-setup`, `domain-detect`, `skill-list`) are each reached or explicitly retired with a reason recorded
  9. `queen-seed-from-hive` is reconnected to `/ant-init` (matching v5.4.0) or explicitly retired alongside the Phase 162 hive-default decision -- not left silently unreferenced
**Plans**: TBD

### Phase 171: Prove It
**Goal**: Aether's daily-driver claim is tested against real evidence, on real repositories, using an inexpensive model. The existing benchmark cannot do this job today: `aether_bench/aether_colony.py:143-166` runs the identical `claude --dangerously-skip-permissions --print` invocation as the solo arm and never calls a single Aether command -- it compares Claude Code to Claude Code. This phase either fixes that or explicitly retires the harness; it is not left described as merely "needing repair."
**Depends on**: Phase 160, Phase 161, Phase 162, Phase 163, Phase 164, Phase 165, Phase 166, Phase 167, Phase 168, Phase 169, Phase 170
**Requirements**: PROOF-01, PROOF-02, PROOF-03, PROOF-04, PROOF-05
**Success Criteria** (what must be TRUE):
  1. Three real development tasks are completed in real repositories on this machine using an inexpensive model, with the number of operator interventions counted and recorded per task -- the milestone's primary verdict
  2. A task interrupted mid-session resumes in a fresh session without the user re-explaining what was in progress
  3. A full colony lifecycle runs successfully in a separate downstream repo without Aether modifying its own repository -- proving the Phase 161 `colony/` distribution work actually works outside this repo
  4. *(Decision-shaped, not build-shaped)*: a written decision states whether `aether-bench`'s colony arm has been rewritten to actually invoke Aether commands, or whether the harness is retired -- it is not left described as merely "needing repair"
  5. If the benchmark runs, its colony-vs-solo result is recorded in the repository as the milestone's honest verdict, including if it is unfavourable
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 145 → 146 → 147 → 148 → 149 → 150 → 151 → 152 → 153 → 154 → 155 → 156 → 157 → 158 → 159 → 160 → 161 → 162 → 163 → 164 → 165 → 166 → 167 → 168 → 169 → 170 → 171

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
| 153. TS Scaffold & Schemas | v1.24 | 2/2 | Complete | 2026-05-22 |
| 154. Colony Assets | v1.24 | 3/3 | Complete | 2026-05-23 |
| 155. Go Boundary Refactor | v1.24 | 3/3 | Complete | 2026-05-23 |
| 156. TS Control Plane Core | v1.24 | 4/4 | Complete | 2026-05-23 |
| 157. TS Adapters & Oracle | v1.24 | 2/2 | Complete   | 2026-05-24 |
| 158. Event Stream | v1.24 | 3/3 | Complete | 2026-05-24 |
| 159. End-to-End Acceptance | v1.24 | 3/3 | Complete | 2026-05-24 |
| 160. Fail Loudly | v1.25 | 8/8 | Complete   | 2026-07-27 |
| 161. Cheap Models By Design | v1.25 | 0/TBD | Not started | - |
| 162. Switch On Learning | v1.25 | 0/TBD | Not started | - |
| 163. Context Reaches Workers | v1.25 | 0/TBD | Not started | - |
| 164. Research Feeds Planning | v1.25 | 0/TBD | Not started | - |
| 165. Core Lifecycle Commands | v1.25 | 0/TBD | Not started | - |
| 166. Full Colony On Demand | v1.25 | 0/TBD | Not started | - |
| 167. Typed Control | v1.25 | 0/TBD | Not started | - |
| 168. Your Eyes Back — Live Visibility | v1.25 | 0/TBD | Not started | - |
| 169. Your Eyes Back — Charter & Standards | v1.25 | 0/TBD | Not started | - |
| 170. Reclaim The Unreachable | v1.25 | 0/TBD | Not started | - |
| 171. Prove It | v1.25 | 0/TBD | Not started | - |
