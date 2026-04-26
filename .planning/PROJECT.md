# Aether

## What This Is

Aether is a biomimetic AI colony framework: a Go runtime in `cmd/` and `pkg/` that owns state, worker dispatch, verification, memory, and install/update flows, plus companion command surfaces for Claude Code and OpenCode and a runtime-native Codex CLI lane.

`v1.0` restored the lost colony ceremony and runtime visibility surfaces.

`v1.1` made Aether's context layer trustworthy, inspectable, deterministic, and benchmarkable.

`v1.8` added the colony recovery system: `aether recover` detects 7 stuck-state classes, auto-fixes safe issues, prompts for destructive ones, and proves correctness through 10 E2E tests.

`v1.9` added the review persistence system: 7-domain review ledgers accumulate findings across phases, agents persist findings via CLI, colony-prime injects prior reviews into worker context, and full lifecycle integration (seal/entomb/status/init).

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

- Go runtime is healthy, v1.0.24 shipped
- All 10 milestones complete (56 phases, 128 plans)
- 2910+ tests passing, full E2E regression coverage
- Stable and dev publish channels with integrity verification
- Plan recovery pipeline hardened (`--force` always recovers)
- Colony recovery system shipped: `aether recover` + `--apply` for stuck-state rescue
- Review persistence system shipped: 7-domain ledger CRUD, colony-prime injection, agent Write tools, lifecycle integration
- All 50 slash commands working across Claude Code, OpenCode, and Codex CLI

<details>
<summary>Prior State History</summary>

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
- [ ] v1.10 Colony Polish -- Phases 57+

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

### Active

- Smart review depth (auto/light/heavy) -- v1.10
- Gate failure recovery -- v1.10
- Porter ant (26th caste, interactive delivery) -- v1.10
- Lifecycle ceremony (seal, init, status, entomb, resume, discuss, chaos, oracle, patrol) -- v1.10
- Oracle loop fix (research formulation + depth selection) -- v1.10
- Idea shelving system (persistent colony backlog) -- v1.10
- QUEEN.md pipeline fix (dedup, wiring, auto-promotion) -- v1.10

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

Shipped v1.9 with 104 files changed, +9,713 / -300 lines of Go.
Tech stack: Go 1.24, Cobra CLI, pkg/storage file locking.
58+ tests added across milestone. All passing.

## Explicit Deferrals

These remain promising but are not the next best move:

- pheromone markets and reputation exchange
- swarm memory beyond the current hive/wisdom path
- federation / inter-colony coordination
- self-mutating agents / evolution engine

## Current Milestone: v1.10 Colony Polish

**Goal:** Make every colony interaction feel complete and self-sustaining — review depth is smart, gate failures are recoverable, lifecycle commands have real ceremony, delivery has an interactive ant, the Oracle has proper research formulation, ideas get shelved for future colonies, and QUEEN.md is fully wired.

**Target features:**
- Smart review depth — auto/light/heavy modes, `--light` flag, final phase always heavy
- Gate failure recovery — clear, actionable recovery paths when verification gates fail
- Porter ant — 26th caste, wired into lifecycle (especially seal), interactive publishing prompts
- Lifecycle ceremony — seal (flags, midden, wisdom, pheromone cleanup), init (deeper analysis), status (version info), entomb (wisdom extraction), resume (staleness), discuss/council (codebase-aware comprehensive questioning), chaos/oracle/patrol (signal integration)
- Oracle loop fix — fix callback URL AND restore research formulation with depth selection
- Idea shelving — persistent backlog, auto-shelve at seal, surface at init
- QUEEN.md full fix — dedup explosion, pipeline wiring (global wisdom not injected, sections ignored, auto-promotion missing)

**Design context:** Full plan detail preserved in `.planning/research/v1.10-PLANS-CONTEXT.md`

## Next Move

Plan next milestone with `/gsd-new-milestone`.

## Evolution

This document evolves at phase transitions and milestone boundaries.

*Last updated: 2026-04-26 after starting v1.10 milestone*
