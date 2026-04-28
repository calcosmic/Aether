# Aether

## What This Is

Aether is a biomimetic AI colony framework: a Go runtime in `cmd/` and `pkg/` that owns state, worker dispatch, verification, memory, and install/update flows, plus companion command surfaces for Claude Code and OpenCode and a runtime-native Codex CLI lane.

`v1.0` restored the lost colony ceremony and runtime visibility surfaces.

`v1.1` made Aether's context layer trustworthy, inspectable, deterministic, and benchmarkable.

`v1.8` added the colony recovery system: `aether recover` detects 7 stuck-state classes, auto-fixes safe issues, prompts for destructive ones, and proves correctness through 10 E2E tests.

`v1.9` added the review persistence system: 7-domain review ledgers accumulate findings across phases, agents persist findings via CLI, colony-prime injects prior reviews into worker context, and full lifecycle integration (seal/entomb/status/init).

`v1.10` completed the colony polish: smart review depth (light/heavy), gate failure recovery with skip logic, Porter ant (26th caste), full lifecycle ceremony (seal, init, status, entomb, resume, discuss, chaos, oracle, patrol), Oracle loop fix with research formulation, idea shelving system, QUEEN.md pipeline fix, and Hive Brain wiring into seal.

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
- Milestone v1.10 Colony Polish complete (shipped 2026-04-28)
- 2910+ tests passing, full E2E regression coverage
- Stable and dev publish channels with integrity verification
- Colony recovery system: `aether recover` + `--apply` for stuck-state rescue
- Review persistence system: 7-domain ledger CRUD, colony-prime injection, lifecycle integration
- Idea shelving: persistent colony backlog, auto-shelve at seal, surface at init, survive entomb
- Hive Brain: seal auto-promotes high-confidence instincts to cross-colony wisdom
- Smart review depth: light/heavy modes, final phase always heavy, `--light` flag
- Porter ant: 26th caste, wired into seal lifecycle with interactive publishing
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
- [x] v1.10 Colony Polish -- Phases 57-69 (shipped 2026-04-28)

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

### Active

(None — define next milestone goals with `/gsd-new-milestone`)

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

Plan next milestone with `/gsd-new-milestone`.

## Evolution

This document evolves at phase transitions and milestone boundaries.

*Last updated: 2026-04-28 after v1.10 milestone completion*
