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

- **v1.23 Daily Driver Reliability — SHIPPED 2026-05-21**
- **Product version: v1.0.41**
- v1.23 shipped: Honest error pipeline, command classification, critical test coverage, learning lifecycle, Queen policy alignment, runtime safety, end-to-end proof
- Focus: Planning next milestone
- Go = truth (state, gates, finalizers), playbooks = soul, YAML = packaging
- 512 TS tests, 2900+ Go tests, 86 skills, 60 commands per platform, 27 Codex agents

## Current Milestone: v1.24 Hybrid Architecture Salvage

**Goal:** Demote Go from "whole organism" to "runtime spine," extract living behaviour into editable Markdown/YAML, and bootstrap a TypeScript control plane — without rewriting from scratch.

**Target features:**
- Define the architecture boundary (Go vs TS vs config)
- Create Classic parity checklist as the golden reference
- Audit Go codebase for behaviour that must move out
- Build minimal TS control plane that loads agents/phases/prompts from files
- Extract hardcoded Go prompt/agent/phase logic into editable assets
- Establish NDJSON event stream as observable truth
- Validate: one end-to-end flow (init → plan → build → verify → seal) via TS orchestrator

**Key constraint:** Salvage, not rewrite. Preserve Go's runtime reliability. Make behaviour human-editable.

**Previous Milestone: v1.23 Daily Driver Reliability** — Shipped 2026-05-21

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
- [ ] v1.21 Live Colony -- Phases 136+
- [x] v1.22 Grounded Planning + Ceremony Restore -- Phases 141-144 (shipped 2026-05-19)
- [ ] v1.23 Daily Driver Reliability -- Phases 145+

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

## Current Milestone: v1.11 Aether Unification

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

*Last updated: 2026-05-20 after v1.23 milestone started*

## Current Milestone: v1.12 Safe Colony

**Goal:** Make Aether loop-proof and give users independent control over planning depth and verification depth, with smart defaults that adapt to phase position and code change risk.

**Target features:**
- Full loop audit — scan every Aether command for potential infinite loops, add circuit breakers
- Independent depth controls — separate planning depth from verification depth, both user-settable
- Smart depth defaults — auto-select depth from phase position + code change risk signals
- User depth override — tick-a-box UI at `/ant-plan` start to override either depth before plan creation

*Last updated: 2026-05-01 — v1.12 Safe Colony milestone shipped*

## Current Milestone: v1.16 Hybrid Runtime Boundary and Orchestration Recovery

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

*Last updated: 2026-05-12 — milestone pivoted from adaptive caste orchestration to hybrid runtime recovery*
