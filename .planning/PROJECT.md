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

- **v1.25 Switch It On — IN PROGRESS** (started 2026-07-25)
- **Product version: v1.0.42**
- Phase 165 complete (2026-08-03) — core lifecycle commands: `init`/`plan`/`build`/`continue` wrappers rewritten to carry engineering method (stage purpose, files-to-read, spawn choreography, stop conditions) instead of manifest-parsing protocol, on both platforms plus flat mirrors, fenced by wrapper ceremony contract tests; gap closure moved shelf promotion inside the `aether init` transaction (atomic, consent-gated via `--promote-shelf`/`--dismiss-shelf`); 10/10 plans, re-verification passed 10/10; one pre-existing `init-ceremony` sealed-colony risk deferred to Phase 169 (SEE-11)
- Phase 164 complete (2026-08-02) — research feeds planning: Queen decides research need per phase, research Scouts dispatch automatically through the kept `confidence-loop.ts`, dispatch field fidelity fixed (permission_profile + brief now survive the Go→TS boundary), `plan-research-escalate` works on fresh colonies and degrades to a warning on failure; 11/11 plans, verification passed 5/5
- Phase 163.2 complete (2026-08-01) — ts-host preflight configurability: per-platform TTL preflight success cache (1h default, `AETHER_PREFLIGHT_CACHE_TTL`), one shared `AETHER_PREFLIGHT_TIMEOUT` knob across Go and TS hosts (hardcoded 20s gone), neutral temp-dir probes, loud `AETHER_SKIP_PREFLIGHT` notice, auth-failure cache invalidation; 7/7 decisions verified, code review Critical+5 Warnings fixed
- Phase 163.1 complete (2026-08-01) — wrapper-runtime completion contract: schema-validated completion packets (submitted bytes, batch rejection), stage-time validation with digest rebinding, verification honesty, read-only artifact evidence on both continue paths, composed brief file channel, publish-order fix; 7/7 requirements verified
- v1.24 shipped: Architecture boundary defined, 27 agent YAMLs + 28 prompt MDs + 9 phase YAMLs + 7 playbooks extracted, TypeScript control plane with 68 tests, NDJSON event stream, 80% Classic parity verified, end-to-end demo flow running
- Focus: Deleting layers, restoring Queen judgement and colony richness
- 27 caste YAMLs and 10 policy files live in `colony/`; the Queen's execution policy does not — it is compiled into `cmd/codex_dispatch_contract.go`
- Four parallel execution paths currently coexist: markdown wrappers + playbooks (~13k lines), `.aether/ts-host` (40 files), `control-ts` (33 files, self-described as retired), and the Go runtime
- `v5.4.0` tag remains available as the Classic behaviour baseline for Queen orchestration archaeology

## Current Milestone: v1.25 Switch It On

**Goal:** Make Aether usable daily on an inexpensive model by switching on machinery that already exists and has never run.

**How this milestone was arrived at:** an initial plan was written, then subjected to six specialist review agents — claim verification, adversarial refutation, goal-backward plan checking, architectural blast-radius, completeness archaeology, and a devil's advocate. The review refuted the original diagnosis and several of its load-bearing claims. This milestone is the rebuild.

**What the review established:**

1. **Almost nothing was deleted; things were never switched on.** `pkg/memory/pipeline.go` wires the entire learning loop and is invoked by nothing — the colony has never learned. `colony/policies/model-routing.yaml` maps every caste to a model and has zero readers — the cheap-model mechanism exists and is inert. `colony-vital-signs` computes colony health 0-100 and is unplugged. The approved charter never reaches a worker.
2. **The TypeScript host is an asset, not a liability.** `.aether/ts-host/src/` holds the only playbook loader, the only confidence loop, and the only worker dashboard and swarm display. The original plan deleted it in Phase 160 and rebuilt its functions in Phases 161-166. It also breaks `go build` outright via `//go:embed`.
3. **Failures are silent.** Seven playbook CLI calls were executed against a built binary and failed — including the Gatekeeper security scan — while stderr is redirected to `/dev/null`. The project's own drift test passes because its regex sees only `--flags`, never positional arguments.
4. **The framework's real disease is churn, not a severed wire.** Build orchestration was rearchitected at least five times in eight weeks. Several previous milestones fixed defects that later silently reverted.

**Target features:**
- Fail loudly — fix seven broken calls, close the drift-test blind spot, stop suppressing stderr, correct three documents that describe behaviour which has never happened
- Cheap models by design — wire `model-routing.yaml`, give the six unread policy files readers, and distribute `colony/` so policies work outside this repo
- Switch on learning — run the consolidation pipeline at phase end and at seal; reconcile the two competing learning systems
- Context reaches workers — the capsule, survey, phase research, `suggest-analyze`, and the approved charter; measured, bounded, and inspectable
- Typed control — migrate `mode` before requiring it; close ten-plus prose-to-control-flow sites, starting with the grounding-gate exemption
- Your eyes back — wire the health meter and dashboard that already exist; decide what a live panel means on Claude Code before building one
- Reclaim the 37 subcommands that no playbook reconnection can recover
- Retire `control-ts` only; keep the TS host and the narrator
- Prove it on three real tasks with an inexpensive model

**Explicitly not in scope:** deleting the TS host, a four-mode Queen policy (it never existed), collapsing castes, Codex parity, hive trust redesign, a Dream caste.

**Previous Milestone: v1.24 Hybrid Architecture Salvage** — Shipped 2026-05-24

Key accomplishments: Architecture boundary documented with hard rule, Classic parity checklist with 16 items, 162 Go symbols classified for extraction, 27 agent YAMLs + 28 prompt MDs + 9 phase YAMLs + 7 playbooks + 8 policy YAMLs created under `colony/`, Go runtime refactored to load from files, TypeScript control plane with loaders, orchestrator, and 68 tests, NDJSON event stream shared between Go and TS, 80% Classic parity verified, end-to-end demo CLI emits 48 events per run.

**Stats:** 86 commits, 375 files changed, +45,732 / −507 lines

Full details: `.planning/milestones/v1.24-ROADMAP.md`

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
- **Codex is runtime-native**; no markdown wrapper ceremony
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
- [ ] v1.25 Switch It On -- Phases 160-171 (started 2026-07-25)

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

### Active

- [ ] CATALOG-01: Build command reliability matrix from audit-catalog, current docs, and historical tags (v1.10-v1.21, v5.4.0)
- [ ] CATALOG-02: Classify every command as public lifecycle, public utility, internal runtime, alias, or deprecated
- [ ] TEST-01: Every public command has at least one smoke/fixture/e2e test with executable evidence
- [ ] TEST-02: Test matrix documents which commands are covered and how
- [ ] WORKFLOW-01: Colonize workflow restored and verified end-to-end
- [ ] WORKFLOW-02: Iterative plan workflow restored and verified end-to-end
- [ ] WORKFLOW-03: Iterative oracle/RALF workflow restored and verified end-to-end
- [ ] WORKFLOW-04: Build workflow restored and verified end-to-end
- [ ] WORKFLOW-05: Continue workflow restored and verified end-to-end
- [ ] WORKFLOW-06: Run (autopilot) workflow restored and verified end-to-end
- [ ] WORKFLOW-07: Swarm workflow restored and verified end-to-end
- [ ] WORKFLOW-08: Seal workflow restored and verified end-to-end
- [ ] WORKFLOW-09: Entomb workflow restored and verified end-to-end
- [ ] QUEEN-01: Queen execution policy restored with 4 modes: direct, single-worker, focused-review, full-colony
- [ ] QUEEN-02: Deterministic commands and small lifecycle steps do not spawn unnecessary agents
- [ ] RUNTIME-01: Stale decision/session leakage between colonies eliminated
- [ ] RUNTIME-02: Worker artifact/result collection preserves completed work (no silent loss)
- [ ] PROOF-01: End-to-end smoke test in a separate downstream repo proves Aether works without modifying itself during the run

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

## Context

Shipped v1.10 with 452 files changed, +53,409 / -562 lines across 204 commits.
Tech stack: Go 1.24, Cobra CLI, pkg/storage file locking.
34 plans across 14 phases (57-69). All verified.

## Explicit Deferrals

These remain promising but are not the next best move:

- pheromone markets and reputation exchange
- swarm memory beyond the current hive/wisdom path
- federation / inter-colony coordination
- self-mutating agents / evolution engine

## Next Move

Execute v1.15 with `/gsd-discuss-phase 100`.

## Evolution

This document evolves at phase transitions and milestone boundaries.

## Historical Milestone Briefs (superseded)

*The three sections below are archived milestone briefs from v1.11, v1.12, and v1.16. They are retained for context only and are NOT the current milestone. The current milestone is v1.25 Switch It On, defined near the top of this file.*

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

*Last updated: 2026-08-01 — Phase 163.2 complete (ts-host preflight configurability)*

### v1.12 Safe Colony (shipped)

**Goal:** Make Aether loop-proof and give users independent control over planning depth and verification depth, with smart defaults that adapt to phase position and code change risk.

**Target features:**
- Full loop audit — scan every Aether command for potential infinite loops, add circuit breakers
- Independent depth controls — separate planning depth from verification depth, both user-settable
- Smart depth defaults — auto-select depth from phase position + code change risk signals
- User depth override — tick-a-box UI at `/ant-plan` start to override either depth before plan creation

*Last updated: 2026-05-01 — v1.12 Safe Colony milestone shipped*

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

*Last updated: 2026-08-03 — Phase 165 (core lifecycle commands) complete*
