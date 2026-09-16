# Aether

## What This Is

Aether is a biomimetic AI colony framework: a Go runtime in `cmd/` and `pkg/` that owns state, worker dispatch, verification, memory, and install/update flows, plus companion command surfaces for Claude Code and OpenCode and a runtime-native Codex CLI lane.

`v1.0` restored the lost colony ceremony and runtime visibility surfaces.

`v1.1` made Aether's context layer trustworthy, inspectable, deterministic, and benchmarkable.

`v1.8` added the colony recovery system: `aether recover` detects 7 stuck-state classes, auto-fixes safe issues, prompts for destructive ones, and proves correctness through 10 E2E tests.

`v1.9` added the review persistence system: 7-domain review ledgers accumulate findings across phases, agents persist findings via CLI, colony-prime injects prior reviews into worker context, and full lifecycle integration (seal/entomb/status/init).

`v1.10` completed the colony polish: smart review depth (light/heavy), gate failure recovery with skip logic, Porter ant (26th caste), full lifecycle ceremony (seal, init, status, entomb, resume, discuss, chaos, oracle, patrol), Oracle loop fix with research formulation, idea shelving system, QUEEN.md pipeline fix, and Hive Brain wiring into seal.

`v1.11` unified Aether: removed self-hosting artifacts (stale agents, duplicate commands, orphaned companion files), restored lost Smart Init intelligence (charter ceremony, rich init-research, suggest-analyze), hardened the 3-platform experience, and improved user-facing flows.

`v1.12` made Aether loop-proof and depth-aware: 6 loop safety requirements (watcher auto-skip, recovery redirect, circuit breaker, cycle detection, lifecycle exclusion, telemetry), independent 3-level planning and verification depth with smart defaults, depth persistence across plan→build→continue, and a unified depth selection UI.

## Core Value

**Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.**

That means:

- worker lifecycle must be inspectable and honest
- dispatch visibility must come from real runtime state
- stale run state must not poison future commands
- verification must lead advancement decisions
- partial success and recovery must be first-class
- stuck colonies must be recoverable with a single command
- review findings must survive `/clear` and accumulate across phases

## Current State

- **Phase 204.1 — Codex Ant Skill Surface implemented (2026-09-16), verification pending**: all seven plans and the fresh-install review repair are merged. Nine canonical dollar skills have clean-install and normal legacy-update proof on fresh Codex CLI 0.154.0 sessions; immutable executable version metadata fixes unstamped Go-build installation. Eighteen positive live cases and four rejecting controls pass. Three recorded human-verification items remain, and the full suite retains the same 17 historical failures. This is bounded entrypoint/distribution proof, not full workflow/native-helper parity, release qualification or owner acceptance.
- **v1.27 The Queen Decides, the Program Checks — SHIPPED 2026-09-02**: 9 phases, 84 plans, 195 recorded tasks, and 48/48 requirements. The audit found no blocking requirement, integration, or end-to-end gaps; one Phase 198.2 real-project memory observation and documented warnings carry as explicit debt.
- **Product version: v1.0.79** (last verified local release, recorded in Phase 205-13; Codex parity changes require a new qualified candidate)
- **v1.28 Classic Colony Restoration is active**: approved Phases 199-205, including inserted 204.1-204.5, map 81 testable requirements and all 72 Classic capability rows. The authoritative brief is `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md`; the ledger is traceability, not another owner interview.
- **Phase 199 — Front Door and Classic Contract is complete (2026-09-07)**: 34/34 plans and 12/12 owner UAT checks passed. The ordinary journey, shared lifecycle projection, safe pause/resume, honest seal/entomb split, expert maintenance boundary, and executable cross-platform Classic corpus are now delivered.
- **Phase 204 -- Learning Governor is complete (2026-09-15)**: 16/16 plans including five gap-closure plans; re-verified 9/11 with two owner-accepted deferrals. The program's memory now carries versions and lineage, every run leaves a durable episode record, a real shadow grader and automatic canary pass run at the end of every check, and a source-improvement proposal can be drafted automatically but never merged, published or deployed by the program. Known open limits: the automatic hypothesis promoter cannot fire until the learning store and the guidance-application store share an identifier (WINDOWS.md entry 44); check-lane usage/cost recorded as absent (entry 48); 30 of 51 bank fixtures unguarded (entry 42).
- The priority implementation spec v3 (Downloads, 2026-08-21) is the ratified governing backlog (ruling D1); its order was amended by D12 on 2026-08-22 so v1.27 leads with team judgement, single verification, the cost line, the next-action card and Classic display restoration
- Phase 194's end-to-end proof reduced the measured one-task bug fix from 8 workers to exactly 1 Builder plus the program's free checks on both the proposed and automatic paths; an explicit heavy request still produces 4 workers, proving the review pipeline remains live. Broader owner-watched proof follows restoration rather than preceding it.
- Four parallel execution paths have been reduced to two since v1.24: the Go runtime (authoritative) and the thin markdown wrappers; `control-ts/` retired in v1.25, the playbooks no longer loaded, 39 zero-reader config files and 8 dead commands deleted in v1.26
- `v5.4.0` tag remains the Classic behaviour baseline; the 2026-08-22 display audit lists 19 Classic elements absent today and 13 thinner (recorded in `research/v1.27-milestone-brief.md`)

## Current Milestone: v1.28 Classic Colony Restoration — Active

**Goal:** Reunite the modern Go safety kernel with the understandable, visible, Queen-led February-April colony experience.

**Approved delivery:** Phases 199-205, including Classic Visual Voice (202.1) and owner-requested Codex parity insertions (204.1-204.5). The remaining Phase 205 walkthrough and restoration seal follow the qualified Codex candidate. See the [2026-09-16 scope amendment](research/ant-codex-parity-2026-09-16/GSD-INTEGRATION.md).

**Approved planning contract:** 81 requirements (65 existing plus 16 Codex ANT requirements) in `.planning/REQUIREMENTS.md`; 72/72 Classic capability rows routed in `.planning/research/v1.28-classic-capability-ledger.md`; no unsupported retirement and no repeat owner discovery round. Each Phase 199-204 must reconstruct the relevant Classic mechanism from historical source, audit the current Go path, explain what created the owner value, compare alternatives, and record an evidence-backed modern synthesis before implementation plans are approved. The three pending todos route to Phase 200 spec output, Phase 201 turnaround, and Phase 203 platform preflight.

**Sequence:** Phases 199-204 restored the product; Phase 205 already completed plans 01-13 and plan 14 task 1. Owner direction on 2026-09-16 inserts Phases 204.1-204.5 before the remaining owner walkthrough, superseding the later-Codex deferral. Preserve completed evidence and the original before-inventory, then qualify a new candidate and separately record any new preparation before resuming acceptance.

## Previous Milestone: v1.27 The Queen Decides, the Program Checks — Shipped 2026-09-02

**Goal:** Aether stops burning tokens on workers nobody needed — the model's judgement picks the team and says why, the program's free checks are the floor that never moves, each phase is verified once, the cost is printed, and the owner can see every decision — so a one-task bug fix costs one worker plus checks.

**Delivered:** the program's checks became the unskippable safety floor; ordinary work shrank to one explained Builder; related tasks became coherent jobs; cost and next-step truth reached every lifecycle screen; runtime evidence and learning became visible and fed back into helpers; and a fail-closed six-phase unattended run completed.

**Closeout:** 48/48 requirements satisfied; 8/8 integration seams and 4/4 end-to-end flows verified. The audit status is `tech_debt`, not `gaps_found`.

**Governing rulings:** D11 (deterministic checks are the floor, reviewers are judgement) and D12 (order amendment) — `.planning/decisions/2026-08-21-owner-rulings-priority-spec-v3.md`; full brief `.planning/research/v1.27-milestone-brief.md`.

## Previous Milestone: v1.26 Intelligent Orchestration — Shipped 2026-08-22

**Goal (as roadmapped 2026-08-08):** the colony reads the work, sends only the workers that work needs, hands each one what it needs to know, and can prove what it cost. Rescoped to hardening on 2026-08-14 ("remove things until a small job costs a small amount") and extended by the 2026-08-17 truth audit.

**Key accomplishments:** a wiring ratchet (no feature without a caller, release gate proven by execution); fail-closed delegation guards; token measurement with stated arithmetic (186× undercount fixed); Aether's internals no longer leak into other projects' workers; crash-safe worktrees; one truth for failures and phase advances; complete, non-duplicated worker briefs; 39 zero-reader configs and 8 dead commands deleted; field hardening from three downstream repos in one day; the owner in the loop (team check-in, decision routing, honest no-change results).

**What it did not fix (and why v1.27 exists):** the default team is still decided by a keyword engine filling a budget of 6-8, a checker worker is mandatory on every build, and each phase is verified two or three times. 21/35 requirements satisfied; 7 carried into v1.27, 7 dropped or deferred by recorded rulings.

**Stats:** 571 commits, 1,223 files changed, +171,000 / −39,221 lines, 2026-08-08 → 2026-08-22. Product v1.0.47 → v1.0.63.

Full details: `.planning/milestones/v1.26-ROADMAP.md`, `.planning/MILESTONES.md`.

## Previous Milestone: v1.23 Daily Driver Reliability — Shipped 2026-05-21

Key accomplishments: Removed 29+ error suppressions, built audit-catalog classification matrix, added tests for 12 critical source files, restored learning lifecycle and 9 flagship workflows, aligned Queen policy with Go runtime, added provider error classification + build-reconcile + worktree safety, full lifecycle smoke test in downstream repo. 30/30 requirements, 27 plans, 7 phases.

**Stats:** 41 commits, 213 files changed, +20,787 / −9,965 lines

Full details: `.planning/milestones/v1.23-ROADMAP.md`

## Previous Milestone: v1.23 Daily Driver Reliability

**Shipped:** 2026-05-21
**Phases:** 7 (145-151) | **Plans:** 27

**Key accomplishments:**
- Removed error suppression from 29+ data-persistence CLI calls — errors now surface honestly
- Built command reliability matrix and `audit-catalog` classification report
- Added tests for 12 HIGH-severity untested source files (eventbus, queen, instinct, midden, hive, spawn, autopilot, flags, council, shelf)
- Restored learning lifecycle (hypothesis/validate/disprove) and all 9 flagship workflows
- Aligned Queen execution policy docs with Go runtime; verified deterministic commands spawn zero agents
- Added provider error classification, build-reconcile recovery, worktree merge-back, orphan detection
- Full lifecycle smoke test in downstream repo + TS host e2e + Go manifest integration tests

**Stats:** 41 commits, 213 files changed, +20,787 / −9,965 lines

Full details: `.planning/milestones/v1.23-ROADMAP.md`

## Previous Milestone: v1.22 Grounded Planning + Ceremony Restore

**Shipped:** 2026-05-19
**Phases:** 4 (141-144) | **Plans:** 8

**Key accomplishments:**
- Unified 5 divergent skip lists into canonical ScanFilter with 36-entry noise map
- Source anchors extracted from clean survey, wired through to planner context
- Plan-grounding validation gate + decision conflict detection with 9 contradiction pairs
- PlaybookLoader TS module restored ceremony (loads 5 build + 2 plan playbooks)
- 84 new tests proving end-to-end pipeline: survey → anchors → grounding → ceremony → regression

**Stats:** 44 commits, 53 files changed, +7,396 / -151 lines

Full details: `.planning/milestones/v1.22-ROADMAP.md`

<details>
<summary>v1.21 Live Colony Summary</summary>

**Shipped:** 2026-05-18 | **Phases:** 5 (136-140) | **Plans:** 18

Key accomplishments: Production TS host dispatch with ceremony, worker-to-worker spawning with budget enforcement, confidence-driven build iteration, cross-colony hive wisdom injection, 465 TS tests.

Full details: `.planning/milestones/v1.21-ROADMAP.md`
</details>

<details>
<summary>v1.18 Hybrid Runtime Parity & Release Gate Summary</summary>

**Shipped:** 2026-05-14
**Phases:** 5 (119-123)
**Requirements:** 25/25 Complete
**Product Version:** v1.0.38

Key accomplishments:
- TS host stabilized — typecheck clean, 168 tests passing, event bridge teardown fixed
- Platform dispatch corrected — Codex prompt passing fixed, Claude/OpenCode/Codex arg tests added
- Go test suite restored — 7 failures resolved (time-bomb + archived docs), all 17 packages green
- Classic parity verified — golden tests match v5.4 baseline for build, continue, Oracle, dashboard, install/update, state
- Release gate passed — dev channel publish v1.0.38 succeeded, downstream smoke test verified

Full details: `.planning/milestones/v1.18-ROADMAP.md`
</details>

<details>
<summary>v1.17 Classic Restoration Summary</summary>

**Shipped:** 2026-05-14
**Phases:** 7 (112-118)
**Requirements:** 32/32 Complete

Key accomplishments:
- TS host consumes Go ceremony events via JSONL stream and renders banners, spawn frames, stage separators
- Real platform worker dispatch (Claude Code, OpenCode, Codex) with parallel waves and retry logic
- Live terminal swarm dashboard with animated spinners, progress bars, chamber activity map
- Queen Orchestrator with workflow pattern selection, Builder-Probe Lock, midden checks
- Oracle RALF loop with phase-aware prompts, diminishing returns detection, template-specific synthesis
- Golden workflow snapshot tests proving restored system matches Classic v5.4 behavior
- Cross-platform parity tests + state safety integration tests

Full details: `.planning/milestones/v1.17-ROADMAP.md`
</details>

<details>
<summary>Prior State History</summary>

- v1.17 (Classic Restoration, Phases 112-118): TS host control plane, ceremony narrator, swarm dashboard, Queen orchestration, Oracle enhancement, parity verification — shipped 2026-05-14
- v1.16 (Hybrid Runtime Boundary, Phases 106-111): Boundary contract, Classic baseline, TS host prototype, Go safety invariants — shipped 2026-05-13
- v1.0 (MVP, Phases 1-6): Colony ceremony and runtime visibility
- v1.1 (Trusted Context, Phases 7-11): Context proof and skill routing
- v1.2 (Live Dispatch Truth, Phases 12-16): Worker dispatch honesty
- v1.3 (Visual Truth, Phases 17-24): Caste identity, stage separators, trace logging
- v1.4 (Self-Healing, Phases 25-30): Medic ant, ceremony integrity
- v1.5 (Runtime Truth Recovery, Phases 31-38): Continue unblock, release v1.0.20
- v1.6 (Release Pipeline, Phases 39-46): Publish hardening, E2E regression
- v1.7 (Planning Pipeline, Phases 47-48): Plan --force recovery, E2E recovery test
- v1.8 (Colony Recovery, Phases 49-51): Stuck-state detection, auto-repair, E2E verification

</details>

## Architecture / Key Patterns

- **Go runtime is authoritative** for state mutations, verification, and CLI truth
- **Wrappers are presentation-only** on Claude/OpenCode
- **Codex uses the Go runtime as authority**; inserted Phases 204.1-204.5 add canonical `$ant-*` orchestration skills with governed native workers
- **YAML remains source-of-truth** for generated wrapper commands
- **Runtime proof beats wrapper theater**
- **Shared lifecycle truth matters**; `build`, `plan`, `colonize`, `watch`, `status`, and `continue` should agree on what a worker is doing
- **Recovery is first-class**; stuck colonies get a rescue button, not manual file surgery
- **Review persistence is first-class**; findings accumulate across phases and survive session resets

## Milestone Sequence

- [x] v1.0 MVP -- Phases 1-6
- [x] v1.1 Trusted Context -- Phases 7-11
- [x] v1.2 Live Dispatch Truth and Recovery -- Phases 12-16
- [x] v1.3 Visual Truth and Core Hardening -- Phases 17-24 (shipped 2026-04-21)
- [x] v1.4 Self-Healing Colony -- Phases 25-30 (completed 2026-04-21)
- [x] v1.5 Runtime Truth Recovery -- Phases 31-38 (completed 2026-04-23, product v1.0.20)
- [x] v1.6 Release Pipeline Integrity -- Phases 39-46 (completed 2026-04-24)
- [x] v1.7 Planning Pipeline Recovery -- Phases 47-48 (completed 2026-04-24)
- [x] v1.8 Colony Recovery -- Phases 49-51 (completed 2026-04-25)
- [x] v1.9 Review Persistence -- Phases 52-56 (completed 2026-04-26)
- [x] v1.10 Colony Polish -- Phases 57-69 (shipped 2026-04-28)
- [x] v1.11 Aether Unification -- Phases 70-79 (shipped 2026-04-30)
- [x] v1.12 Safe Colony -- Phases 80-87 (shipped 2026-05-01)
- [x] v1.13 Recovery Hardening & Hive Learning -- Phases 88-92 (shipped 2026-05-03)
- [x] v1.14 Queen Authority -- Phases 93-99 (shipped 2026-05-04)
- [x] v1.15 Framework Coherence, Efficiency, and Ship Readiness -- Phases 100-105 (shipped 2026-05-08)
- [x] v1.16 Hybrid Runtime Boundary and Orchestration Recovery -- Phases 106-111 (shipped 2026-05-13)
- [x] v1.17 Classic Restoration -- Phases 112-118 (shipped 2026-05-14)
- [x] v1.18 Hybrid Runtime Parity & Release Gate -- Phases 119-123 (shipped 2026-05-14)
- [x] v1.19 TypeScript Host Cutover + Oracle Confidence Recovery -- Phases 124-129 (shipped 2026-05-15)
- [x] v1.20 Host Contract Hardening and Wrapper Reality Check -- Phases 130-135 (shipped 2026-05-15)
- [x] v1.21 Live Colony -- Phases 136-140 (shipped 2026-05-18)
- [x] v1.22 Grounded Planning + Ceremony Restore -- Phases 141-144 (shipped 2026-05-19)
- [x] v1.23 Daily Driver Reliability -- Phases 145-151 (shipped 2026-05-21)
- [x] v1.24 Hybrid Architecture Salvage -- Phases 152-159 (shipped 2026-05-24)
- [~] v1.25 Switch It On -- Phases 160-171 (superseded by v1.26)
- [x] v1.26 Intelligent Orchestration -- Phases 172-191.1 (shipped 2026-08-22)
- [x] v1.27 The Queen Decides, the Program Checks -- Phases 193-198.3 (shipped 2026-09-02)
- [ ] v1.28 Classic Colony Restoration -- active; approved Phases 199-205

## Requirements

### Validated

- Colony ceremony and runtime visibility -- v1.0
- Context proof and skill routing -- v1.1
- Worker dispatch honesty -- v1.2
- Caste identity, stage separators, trace logging -- v1.3
- Medic ant, ceremony integrity -- v1.4
- Continue unblock, release pipeline -- v1.5
- Publish hardening, E2E regression -- v1.6
- Plan recovery, E2E recovery test -- v1.7
- Stuck-state detection, auto-repair, E2E verification -- v1.8
- 7-domain review ledger CRUD with colony-prime injection -- v1.9
- Review agent Write tools with scoped guardrails (28 files, 4 surfaces) -- v1.9
- Full review lifecycle (seal/entomb/status/init) -- v1.9
- Smart review depth (auto/light/heavy, `--light` flag, final phase always heavy) -- v1.10
- Gate failure recovery (recovery templates, per-gate skip, Watcher Veto) -- v1.10
- Porter ant (26th caste, interactive delivery, wired into seal) -- v1.10
- Lifecycle ceremony (seal, init, status, entomb, resume, discuss, chaos, oracle, patrol) -- v1.10
- Oracle loop fix (research formulation, depth selection, state persistence) -- v1.10
- Idea shelving (persistent backlog, auto-shelve, init surfacing, entomb survival) -- v1.10
- QUEEN.md pipeline fix (dedup, global wisdom injection, auto-promotion) -- v1.10
- Hive Brain wiring (seal auto-promotes high-confidence instincts) -- v1.10
- Independent planning depth and verification depth controls -- v1.12 (Phase 83 & 84)
- Smart depth defaults based on phase position and code change risk -- v1.12 (Phase 83 & 84)
- User depth override UI at plan start with persistence -- v1.12 (Phase 86)
- Loop safety (watcher auto-skip, recovery redirect, circuit breaker, cycle detection, lifecycle exclusion, telemetry) -- v1.12
- Recovery hardening (build-complete validation, provenance checks, confidence-targeted Oracle) -- v1.13
- Gate self-healing (Fixer caste, smart gate retry, recoverable banners, /ant-unblock) -- v1.13
- Hive learning (colony memory store, SQLite FTS recall, auto-created skills, privacy gate) -- v1.13
- Worker lifecycle hardening (heartbeats, process groups, PID tracking, stale cleanup) -- v1.13
- Gate classification (hard_block/soft_block/advisory) with audit trail -- v1.14
- Auto-recovery orchestrator (bounded retry, peer redistribution, Fixer dispatch) -- v1.14
- Queen decision layer (pure-function coordinator, plan-only + finalize) -- v1.14
- Smart gates (depth-aware auto-resolve, hard blocks never auto-resolved) -- v1.14
- Queen wave lifecycle (always-advance, dependency injection, recovery) -- v1.14
- Output filtering (verbose-aware, phase-end summaries, queen audit) -- v1.14
- Host Surface Completeness (all 7 subcommands, flag parsing, reference docs) -- v1.20
- Test Coverage for Host Commands (23 TS tests, realistic flag combinations) -- v1.20
- Wrapper Ownership Decision (host-assisted orchestrators, boundary ADR) -- v1.20
- Lifecycle Honesty (simulation gated behind --simulate, error on missing platforms) -- v1.20
- Oracle Storage Pattern (atomic Go-owned storage, interrupt recovery) -- v1.20
- Doc-CLI Alignment Smoke Test (YAML auto-discovery, release gate blocking) -- v1.20
- Wiring ratchet: no registered subcommand without a caller; release gate proven by execution (WIRE-01..03) -- v1.26
- Delegation guards fail closed: parent-derived depth, whole-run ceiling, cycle detection, tamper-refusing ledgers (SPAWN-01..08) -- v1.26
- Token measurement with stated arithmetic incl. cache tokens; estimates separated from measurements (SPEND-01/03/04) -- v1.26
- Orchestration visibility: who was sent, why, who was not, where the runtime overrode (SEEN-01..03) -- v1.26
- Zero-reader config files deleted with reappearance ratchets; dead skill-lifecycle commands removed (ROSTER-01/02, SKILL-01/02) -- v1.26
- Hardening H1-H5: no internals leak into other projects, task-first skill scoring, castes with nothing to do refused, the worker cap caps -- v1.26
- Crash-safe worktrees, one canonical failure log, atomic phase advance, complete non-duplicated briefs, field hardening -- v1.26 (Phases 187-191.1)
- Team check-in before spawning, owner decision routing, honest `completed_no_change` / `verified_existing` results -- v1.26 (2026-08-21)
- Free checks are the floor: one `runDeterministicFloor` body for both continue lanes decides advancement; no flag, depth or proposal can skip a check; build-side reviewer only on an explicit Queen ask (Watcher verified once); reconciliation and program-re-run builder evidence count as proof; unprovable criteria wait for the owner; verification scoped per phase and full at the end; exactly one bounded automatic fix attempt (FLOOR-01..04) -- v1.27 (Phase 193)
- The Queen decides the team: ordinary work gets one Builder, every worker carries a readable per-worker reason, five named risks can force one explained reviewer at continue, owner depth/team controls remain effective, and no caste is dispatched at both build and continue unless explicitly requested (TEAM-01..05) -- v1.27 (Phase 194)
- Coherent jobs: dependency-safe grouping, per-task receipts, exact partial credit, append-only recovery, and one worktree per grouped job work on native and external lanes (JOBS-01..04) -- v1.27 (Phase 195)
- Honest cost: provider-reported token rows survive both wrapper platforms, one cost block ends build/check, `aether spend` is read-only, and model choices carry reasons (COST-01..05) -- v1.27 (Phase 196)
- One canonical next-action resolver supplies every lifecycle command and machine-readable lane, including session start and pause aliases (NEXT-01..06) -- v1.27 (Phase 197)
- Runtime evidence is visible again: direct and wrapper ceremonies agree; checks stream progress; evidence, durations, findings, resume progress and plan confidence render; seal requires owner review (SHOW-01..05) -- v1.27 (Phase 198)
- The learning loop is fed at the source: build and check worker outcomes become failure records and observations through one guarded boundary on both lanes (quick and swarm too); instinct use is a recorded fact so QUEEN.md promotion can actually happen; finished checks, answered questions and failure streaks leave notes; expiring notes reach eternal memory; strong instincts reach the hive at every check under the existing switch; the memory drill-down shows real content; every CLAUDE.md learning claim names a test (FEED-01..06) -- v1.27 (Phase 198.1)
- Memory reaches every relevant helper: plan and colonize delegate manifests carry the runtime-built memory capsule; Oracle research files and registers itself; the greeting shows preferences, strongest habits and the last helper note; dead capsule sections are removed under a writer invariant; survey content and the previous phase's outcome reach later briefs within named budgets (WIRE-01..07) -- v1.27 (Phase 198.2; one real-world UAT carried as acknowledged debt)
- Overnight stamina: owner-eye work queues, genuine danger stops, replans survive restarts, elapsed/cost reporting is durable, blocker truth fails closed, external swarm evidence is issuance-bound, and the six-phase fixture completes unattended (STAM-01..06) -- v1.27 (Phase 198.3)
- Front door and Classic lifecycle contract: one grouped normal journey, automatic territory freshness, one authoritative orientation model, exactly-once pause/resume, retained honest seal plus verified optional entomb, transaction-backed expert maintenance, and a versioned cross-platform behavior corpus (SYNTH-01, CEC-01/02/04/08, LIFE-01..06, PROOF-01) -- v1.28 (Phase 199)
- Queen-led work cycle: the Queen picks and explains the smallest capable team, ordinary work stays proportionate, related tasks form coherent jobs and waves, build/continue/autopilot tell one story with one verification boundary, complete result truth (verdict, checks, cost, elapsed time, changed files, knowledge deltas) binds to the exact attempt, recovery checkpoints and restores, goal-level autopilot advances with declared stops, and worker turnaround is measured (SYNTH-03, CEC-06, WORK-01..08) -- v1.28 (Phase 201; verified 101/101, UAT 74/74)
- Learning governor: versioned, lineage-carrying memory records where a hypothesis is never shown as verified; every lifecycle lane (build, continue, plan, oracle, swarm, recovery) leaves a durable episode record with evidence, gate results, changed decisions and interventions; a real per-fixture shadow grader plus an automatic compare/admit/canary/rollback pass at the end of every check on both check lanes; a regression-fixture bank seeded from the project's own confirmed incidents with a two-sided unguarded-count ratchet; seven budgeted test gates that prove they ran everything; automatic source-improvement proposals that structurally cannot merge, publish or deploy (SYNTH-06, LEARN-01..04, LEARN-06..08) -- v1.28 (Phase 204; verified 9/11 with two owner-accepted deferrals: check-lane token usage/cost stay recorded as absent, 30 of 51 fixtures still unguarded so LEARN-05 stays open)

### Active

The active scope is v1.28 Classic Colony Restoration: restore the February–April front door, iterative planning, Queen-led work, substantive Swarm/Oracle/live visibility, biological recruitment and information exchange, and governed learning around the modern safety kernel. The amended contract contains 81 requirements across Phases 199-205, including 204.1-204.5, and maps all 72 Classic capability rows. Every restoration phase treats its feature names as research headings, reconstructs the exact old causal mechanisms, audits the present Go architecture, and implements only the evidence-backed synthesis.

Long-lived items not tied to a milestone:

- [ ] QUEEN-02: Deterministic commands and small lifecycle steps do not spawn unnecessary agents — *re-expressed as v1.27 feature 2*
- [ ] CATALOG-01/02, TEST-01/02: command/test truth is absorbed into LIFE-06, LEARN-05, and restoration proof rather than a separate pre-milestone gate
- [ ] SKILL-03/04: skill validation on the Claude path remains outside the v1.28 Golden Path. Codex lifecycle skills now have their own ANT-01..16 requirements in Phases 204.1-204.5.

### Out of Scope

| Feature | Reason |
|---------|--------|
| Cross-colony ledger sharing | Findings contain code-specific file paths and line numbers that go stale across repos |
| Auto-block on critical findings | Would create conflicting signals with existing continue-review blocking |
| Auto finding-to-pheromone promotion | Mapping between "finding" and "action" requires judgment, not automation |
| Real-time ledger sync across agents | YAGNI -- agents write during build/continue, not concurrently |
| Ledger web UI | CLI-only for now; web dashboard is a future consideration |

## Key Decisions

| Decision | Outcome | Status |
|----------|---------|--------|
| Review findings are colony-scoped (not cross-colony) | Code-specific paths go stale across repos | Good |
| Domain ledger uses append pattern with computed summaries | No separate phase snapshots needed (YAGNI) | Good |
| All new struct fields use `omitempty` | Backward compatibility with old JSON | Good |
| Zero new dependencies | Uses existing pkg/storage/, cobra, Go stdlib | Good |
| Tracker gets bugs domain carve-out | Write for findings only, never for applying fixes | Good |
| Colony-prime reads from cached summary | Performance over 7 direct ledger reads | Good |
| Rescope v1.26 to hardening: cut 174 (plans 3-9), 176, 177, 178 (2026-08-14) | Removing things until a small job costs a small amount; shipped as 180-184 | Good |
| Phase 192 "stop building when the benchmark passes" rule (2026-08-17) | Superseded 2026-08-21 when the owner ratified the priority spec as the backlog (D1) | — Superseded |
| Automatic model routing approved, after the cost line, reasons always shown (D2, 2026-08-21) | Reverses the 2026-07-28 rejection | — Pending (v1.27 feature 4) |
| Reviewer workers are the Queen's call; the floor is deterministic checks (D11, 2026-08-22) | Replaces "Watcher always required on build"; a 1-task bug fix now costs 1 Builder + checks, with reviewers forced only by five named risks | Good — floor and team judgement shipped (Phases 193-194) |
| A forced-reviewer waiver is owner-only, one signal on one phase, and must be recorded before dispatch begins | The protected `decision-answer` path is capability-checked; generic resolution and late declines fail closed, and repeated card renders preserve commands already shown | Good — Phase 194 plus review fixes |
| The floor is computed from the build-time reviewer value; a reviewer verdict can only ever add a block, never supply a pass (193-01, 2026-08-22) | `TestDeterministicFloorIsTheOnlySourceOfAPass`; both continue lanes share one body, parity table-tested | Good |
| Build-side reviewer dispatch gated on the Queen's explicit proposal, not the required-caste floor (193-02, 2026-08-22) | Watcher no longer reviewed twice; Probe/Auditor/Gatekeeper double-dispatch recorded in `.planning/WINDOWS.md` #1 for Phase 194 | Good — gap deferred |
| A criterion no machine can prove becomes `needs_owner_confirmation`: the phase advances, seal waits for the answer via `aether decision-answer` (193-04, D-05) | No reviewer is ever spawned to guess at it; `--force --reason` still overrides | Good |
| Exactly one automatic builder fix attempt when a free check fails with no reviewer sent; append-only journal entry, then stop with one command (193-05, D-02/D-03) | Distinct from the declined 2026-08-21 recovery retry — one bounded repair, never a loop, never a reviewer first | Good |
| v1.27 order amendment: team judgement, single verification, cost line, next-action card, Classic display restoration first (D12, 2026-08-22) | All nine scoped phases shipped and the milestone audit found no blocking gaps | Good |
| One memory-feed boundary for both build lanes, guarded by AST; feeding memory can never fail a build or check (198.1-01, 2026-08-30) | `TestEveryBuildLaneFeedsMemoryThroughOneBoundary`, `TestFeedingMemoryNeverFailsABuild` | Good |
| Worker-authored text is sanitised before every memory store because midden entries are replayed into later briefs; two writers missed it and were fixed at review (198.1, CR-02, 2026-08-30) | `TestCheckAndQuickFailureTextIsSanitisedBeforeMemory` | Good |
| Instinct use is counted only when the text was genuinely in a worker's capsule on a passing phase; QUEEN.md promotion gated on recorded use (198.1-03, 2026-08-30) | `TestWorkerLessonBecomesQueenFileWisdom`, `TestQueenPromotionNeverHappensWithoutRecordedUse` | Good |
| Hive promotion at every check, same policy switch and confidence gate as seal; `--no-learn` scope documented as legacy capture only (198.1-05, 2026-08-30) | `TestStrongInstinctReachesTheSharedStoreAtCheck`, `TestHivePromotionAtCheckHonoursThePolicySwitch` | Good |
| The runtime assembles one memory capsule; delegate wrappers only pass it through, and plan/colonize wrapper triplets remain byte-identical (198.2-01, 2026-08-30) | Prevents the real Claude/OpenCode lane from silently receiving less context than the in-process lane | Good |
| Oracle research uses one shared finish path to file, register and promote findings with origin labels, including useful partial runs (198.2-02, 2026-08-30) | Paid-for research reaches later helpers without follow-up commands while empty runs write nothing | Good |
| Memory-pack parts without live writers are removed; an AST-driven writer invariant and zero-floor exception ratchet prevent new empty sections (198.2-04, 2026-08-30) | Prevents decorative headings that can never carry content | Good |
| Survey-map content and previous-phase outcomes use two separate named brief budgets; neither adds a worker or owner check-in (198.2-06/07, 2026-08-30) | More grounding reaches helpers without crowding out the task or making small jobs heavier | Good |
| Owner chose to advance from Phase 198.2 while its real-world WIRE-06 confirmation remains untested (2026-08-31) | Carry as verification debt; do not treat it as passed or as a known defect; revisit with `$gsd-verify-work 198.2` | Pending evidence |
| v1.27 ends at Phase 198.3; Classic restoration is v1.28, not a Phase 198.4 questionnaire (2026-09-02) | The comprehensive report already records owner intent; the 72-row ledger is traceability and owner-watched proof follows restoration | Good |
| Codex `$ant-*` parity precedes remaining Phase 205 owner acceptance (2026-09-16) | Owner requested GSD integration before the pending phase; insert 204.1-204.5, retain completed evidence, and qualify the changed candidate | Planned — ANT-01..16 |
| Historical behavior is evidence, while the modern Go runtime remains state and safety authority | Restore outcomes and experience without reviving shell/wrapper state mutation | — Locked for v1.28 |
| Classic feature names are investigation headings, not predetermined solutions (2026-09-02) | Every Phase 199-204 must reconstruct the exact historical mechanism and owner value, trace the current Go implementation, compare keep/restore/replace options, and plan from a written synthesis | — Locked for v1.28 |
| One read-only lifecycle projection owns orientation and Next Up (Phase 199) | Help, status, phase, history, recovery, and closeouts agree; missing evidence remains explicitly unknown rather than being repaired or invented | Good — shipped and UAT-verified |
| Pause/resume are the sole public interruption and recovery path (Phase 199) | Handoffs and recovery are transactional and replay-safe; retired names survive only as bounded hidden input compatibility | Good — shipped and UAT-verified |
| Seal retains reviewable state; entomb separately verifies archive bytes before clearing (Phase 199) | Forced-incomplete closure remains owner-only and visibly unverified through status, archive, and replay | Good — shipped and UAT-verified |
| Expert maintenance uses exact previewed targets and one rollback-capable transaction (Phase 199) | Read-only inspection stays mutation-free and Aether-managed cleanup preserves unmanaged/custom files | Good — shipped and UAT-verified |
| Claude/OpenCode `/ant-*` surfaces and runtime-native Codex share Go-owned lifecycle truth (Phase 199) | A 73-row coverage ledger and executable public journeys prove behavior and state effects without claiming native Codex `$ant-*` support | Good — shipped and UAT-verified |
| Truthful registers over ticked boxes (Phase 204, 2026-09-15) | A requirement is ticked only when its own wording is true of the running program and a named test fails when the wiring is removed; the two truths that remained partial were accepted as deferred by owner override with the exact remaining limit named (WINDOWS.md entries 42 and 48) rather than rounded up | Good -- the post-closure review still caught a green-tested feature that was dead in production (the automatic hypothesis promoter, WINDOWS.md entry 44), which is exactly the failure this rule exists to surface |
| Fixture dates read by the compiled binary must move with the clock (2026-09-15) | Two families of planning fixtures with literal September dates crossed production's seven-day candidate window mid-phase and turned fifteen tests red with no code change; in-process fixtures now pin the candidate clock, binary-facing fixtures derive their instants from the wall clock | Good -- both detonations fixed; rule recorded in the fixture comments |

## Context

Shipped v1.27 on 2026-09-02: 9 phases, 84 plans, 195 recorded tasks, 48/48 requirements; the final normal and race suites each reported 8,566 passing Go tests across 20 packages, and the TypeScript host reported 540 passing tests. Phase 199 then completed the first v1.28 restoration slice with 34 plans, an exact normal/race completion receipt, and 12/12 owner UAT checks.
Tech stack: Go 1.24 runtime (`cmd/`, `pkg/`), thin markdown wrappers for Claude Code and OpenCode, runtime-native Codex lane, a kept TypeScript host for autopilot dispatch. Full normal and race suites remain the release gate.
Owner feedback themes (2026-08-21/22): builds feel heavy for small jobs; the next step after a command is often unclear; projects get stuck with an expensive way out; the display feels thin compared with v5.4.0.
Known debt: Phase 172.1 (CI gate environment), Phase 173's one pending human check, the TS-host probe-timeout twin, Phase 198.2's carried real-world memory-pack UAT, and the acknowledged deferred items in STATE.md. Closed by Phase 194: `.planning/WINDOWS.md` #1 — Probe/Auditor/Gatekeeper no longer double-dispatch at build and continue — and the small-job pipeline now proves one Builder plus checks. Closed by Phase 195: a decision-free one-worker build now takes the visible fast path while genuine owner decisions and explicit `--checkin` still pause.

## Explicit Deferrals

These remain promising but are not the next best move:

- pheromone markets and reputation exchange
- swarm memory beyond the current hive/wisdom path
- federation / inter-colony coordination
- self-mutating agents / evolution engine

## Next Move

Execute Phase 204.1 — Codex Ant Skill Surface from its seven independently checked plans across five waves. Continue the inserted sequence through 204.5, then resume the remaining Phase 205 owner-only walkthrough and acceptance with the qualified candidate handoff.

## Evolution

This document evolves at phase transitions and milestone boundaries.

*Last updated: 2026-09-16 after owner-requested Codex parity insertion before remaining Phase 205 acceptance*

## Historical Milestone Briefs (superseded)

*The three sections below are archived milestone briefs from v1.11, v1.12, and v1.16. They are retained for context only and are NOT the current milestone. The current milestone is v1.28 Classic Colony Restoration, defined near the top of this file.*

### v1.11 Aether Unification (shipped)

**Goal:** Make Aether clean, canonical, and intelligent again — remove self-hosting artifacts, restore lost Smart Init intelligence, harden the 3-platform experience, and improve user-facing flows.

**Target features:**
- Self-hosting cleanup — audit and remove all artifacts that exist because Aether was used to develop itself
- Smart Init ceremony — re-port charter approval flow, repo scanning, governance detection to Go
- Rich init-research — port deep codebase analysis (colony context, governance, pheromone suggestions, complexity)
- Suggest-analyze — restore automatic pheromone suggestions during builds
- Platform hardening — fix OpenCode parity gaps, harden error handling, cross-platform consistency
- User experience — better onboarding, clearer feedback, smoother flows

**Known losses from shell-to-Go migration (April 2026):**
- Colony charter ceremony (scan.sh → charter-write → approval flow)
- Rich init-research (tech stack, directory analysis, colony context, governance detection, 10 pheromone patterns)
- Suggest-analyze (618 lines of automatic pattern detection)
- Bayesian confidence scoring (40/35/25 weighted, 60-day half-life decay, 7 trust tiers)
- Circuit breaker (cascade failure protection)
- State machine transitions (explicit validation, pheromone-triggered, checkpoints)
- Council system (deliberation framework)
- Curation ant pipeline (8-ant orchestrated pipeline)
- Consolidation pipeline (phase-end knowledge compression)

*Last updated: 2026-08-26 after Phase 194*

### v1.12 Safe Colony (shipped)

**Goal:** Make Aether loop-proof and give users independent control over planning depth and verification depth, with smart defaults that adapt to phase position and code change risk.

**Target features:**
- Full loop audit — scan every Aether command for potential infinite loops, add circuit breakers
- Independent depth controls — separate planning depth from verification depth, both user-settable
- Smart depth defaults — auto-select depth from phase position + code change risk signals
- User depth override — tick-a-box UI at `/ant-plan` start to override either depth before plan creation

*Last updated: 2026-08-22 — Phase 193 (Free Checks Are the Floor) complete: verification 4/4, FLOOR-01..04 done*

### v1.16 Hybrid Runtime Boundary and Orchestration Recovery (shipped)

**Goal:** Prove one lifecycle workflow can be restored through a hybrid architecture — Go as safety kernel, TypeScript as orchestration control plane, Markdown/YAML/TOML as editable colony brain, Bash only as small glue.

**Why this matters:**
- The Go runtime has valuable safety machinery, but the Bash/Node-to-Go migration caused regressions in the living parts of Aether
- Queen orchestration, visible worker waves, ceremony, Oracle/RALF confidence iteration, swarm visibility, and platform-specific agent dispatch behavior all degraded
- Research converges on a boundary fix, not a language rewrite: keep Go for safety, restore orchestration in TypeScript, keep editable assets as the colony brain
- The best Classic version (likely v5.4.0) should be used as a behavior baseline, not a permanent second product

**Target features:**
1. Runtime boundary contract — what Go owns, what TypeScript owns, what editable assets own, what Bash may still do
2. Classic baseline identification and smoke-test — verify v5.4.0 as the behavior comparison anchor
3. Golden workflow tests — snapshot/golden tests for `plan -> build 1 -> continue` covering ceremony, worker activity, and state side effects
4. Minimal TypeScript orchestration host — calls Go manifests/finalizers, dispatches visible workers, records spawn-log/spawn-complete, never writes `.aether/data` directly
5. Go safety invariants preserved — Go remains sole authority for state mutation, finalizers, locking, install/update/publish, verification contracts
6. Follow-up migration map — concrete next steps for Oracle confidence iteration, swarm visibility, and broader build/continue parity

**Core principle:** Go should own safety, not soul. The TypeScript control plane restores the living orchestration behavior; the Go kernel remains the only authority for state mutation.

**Non-goals:**
- Do not rewrite the whole runtime in TypeScript
- Do not restore raw Bash state mutation
- Do not maintain Classic and Go as two long-term products
- Do not move install/update/publish safety out of Go
- Do not make visual output parsing authoritative

*Last updated: 2026-09-11 after Phase 201*
