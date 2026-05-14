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

## Current Milestone: v1.18 (TBD)

**Goal:** *(To be defined via `/gsd-new-milestone`)*

## Current State

- **v1.17 Classic Restoration shipped (2026-05-14)** — Full milestone archive in `.planning/milestones/`
- Go runtime is healthy, v1.0.34 shipped
- TS host is the active control plane: event bridge, ceremony narrator, worker dispatch, swarm dashboard, Queen orchestration
- 44+ TS host tests passing + 2900+ Go tests passing
- 10 ceremony snapshot files prove deterministic renderer output
- Cross-platform parity verified: 27 castes on Claude, OpenCode, and Codex
- State safety verified: 3-layer defense (runtime boundary, static analysis, integration tests)
- All 32 v1.17 requirements met

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
- [ ] v1.17 Classic Restoration -- Phases 112+

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

### Active

- [ ] TSHOST-01: TS host dispatches real platform workers (not simulated) with parallel wave execution
- [ ] TSHOST-02: TS host handles error recovery, retry logic, and timeout management
- [ ] TSHOST-03: Event bridge streams Go ceremony events to TS/wrapper consumers
- [ ] CEREMONY-01: Ceremony restored to command wrappers as editable markdown (banners, art, spawn notifications, seal rituals)
- [ ] CEREMONY-02: Shared ceremony config in YAML (caste emoji/color/label maps, naming conventions)
- [ ] CEREMONY-03: Go ceremony rendering code replaced by event emission (Go emits, wrappers render)
- [ ] ORCHESTRA-01: Queen selects workflow patterns (SPBV, Investigate-Fix, Refactor, Compliance, Documentation Sprint) based on phase type
- [ ] ORCHESTRA-02: Builder-Probe Lock restored — builders return code_written, only Probe upgrades to completed
- [ ] ORCHESTRA-03: Tiered escalation chain (worker retry → parent reassignment → Queen reassignment → user escalation)
- [ ] ORCHESTRA-04: Intra-build midden threshold checks with auto-REDIRECT pheromone emission
- [ ] ORACLE-01: Phase-aware prompt directives (survey/investigate/synthesize/verify) injected into Oracle worker briefs
- [ ] ORACLE-02: Diminishing returns detection with novelty delta tracking and forced phase advancement
- [ ] ORACLE-03: Template-specific synthesis sections (tech-eval, architecture-review, bug-investigation, best-practices)
- [ ] SWARM-01: Live terminal dashboard with animated spinners, per-ant progress bars, and tool usage counters
- [ ] SWARM-02: Chamber activity map showing which project areas have active workers
- [ ] PARITY-01: Golden workflow tests comparing output and behavior against v5.4 Classic baseline

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

*Last updated: 2026-04-30 after v1.11 Aether Unification milestone shipped*

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
