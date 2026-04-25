# Aether

## What This Is

Aether is a biomimetic AI colony framework: a Go runtime in `cmd/` and `pkg/` that owns state, worker dispatch, verification, memory, and install/update flows, plus companion command surfaces for Claude Code and OpenCode and a runtime-native Codex CLI lane.

`v1.0` restored the lost colony ceremony and runtime visibility surfaces:

- Phase-aware build and continue ceremony on Claude/OpenCode
- `project|meta` colony scope
- stronger `status` and `watch` runtime visibility
- runtime-first pheromone inspection and steering
- hardened OpenCode/Codex packaging parity

`v1.1` then made Aether's context layer trustworthy, inspectable, deterministic, and benchmarkable.

## Core Value

**Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.**

That means:

- worker lifecycle must be inspectable and honest
- dispatch visibility must come from real runtime state
- stale run state must not poison future commands
- verification must lead advancement decisions
- partial success and recovery must be first-class

## Current State

- Go runtime is healthy, v1.0.24 shipped
- All 8 milestones complete (48 phases, 111 plans)
- 2900+ tests passing, full E2E regression coverage
- Stable and dev publish channels with integrity verification
- Plan recovery pipeline hardened (`--force` always recovers)
- All 50 slash commands working across Claude Code, OpenCode, and Codex CLI
- Awaiting next milestone direction (v1.8)

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

</details>

## Architecture / Key Patterns

- **Go runtime is authoritative** for state mutations, verification, and CLI truth
- **Wrappers are presentation-only** on Claude/OpenCode
- **Codex is runtime-native**; no markdown wrapper ceremony
- **YAML remains source-of-truth** for generated wrapper commands
- **Runtime proof beats wrapper theater**
- **Shared lifecycle truth matters**; `build`, `plan`, `colonize`, `watch`, `status`, and `continue` should agree on what a worker is doing

## Capability Contract

See [.planning/REQUIREMENTS.md](/Users/callumcowie/repos/Aether/.planning/REQUIREMENTS.md:1) for the active capability contract for `v1.3`.

## Milestone Sequence

- [x] v1.0 MVP — Phases 1-6
- [x] v1.1 Trusted Context — Phases 7-11
- [x] v1.2 Live Dispatch Truth and Recovery — Phases 12-16
- [x] v1.3 Visual Truth and Core Hardening — Phases 17-24 (shipped 2026-04-21)
- [x] v1.4 Self-Healing Colony — Phases 25-30 (completed 2026-04-21)
- [x] v1.5 Runtime Truth Recovery — Phases 31-38 (completed 2026-04-23, product v1.0.20)
- [x] v1.6 Release Pipeline Integrity — Phases 39-46 (completed 2026-04-24)
- [ ] v1.7 Planning Pipeline Recovery — Phases 47+

## Previous Milestone Result

`v1.6` is complete. All 11 requirements satisfied, stuck-plan investigation confirmed resolved, v1.0.20 ready to ship.

**v1.3 delivered:**
- Visual UX fully restored (caste identity, stage separators, emoji consistency, Codex parity)
- Core paths hardened (install, update, resume, sync, gate-check)
- Recovery made reliable (worktree cleanup, deferred cleanup, orphan detection)
- Full trace instrumentation (replay, summary, inspect, rotate)
- 12/12 requirements satisfied, 8/8 phases verified

The milestone delivered:

- a shared worker lifecycle truth model across `colonize`, `plan`, `build`, `watch`, `status`, and `continue`
- live dispatch visibility with caste identity, deterministic names, wave/stage structure, durations, and real summaries
- run-scoped spawn state so stale `spawned` workers do not poison later commands
- stronger worker execution observability and robust repo-trust / timeout handling
- verification-led `continue` with partial-success awareness
- manual reconciliation and targeted redispatch for failed or missing work
- thin, honest Claude/OpenCode ceremony driven by runtime truth and a solid Codex CLI renderer

`v1.1` is archived at source commit `fc508afb` (`Add runtime proof surfaces and evaluation harness`).

The milestone now delivers:

- runtime-native context proof and skill proof
- deterministic, explainable skill routing with representative stack coverage
- prompt-integrity trust boundaries with visible block behavior
- trust-weighted deterministic context assembly
- application-aware curation that closes the read/write gap
- a benchmarkable proof/evaluation harness for the trusted-context claim

## Next Milestone Goals

Awaiting direction. Potential v1.8 candidates:
- OpenCode subagent dispatch improvements (branch exists: codex/fix-opencode-subagent-dispatch)
- Codex platform experience improvements
- New agent capabilities or workflow features
- Performance optimization or scale testing

## Milestone Sequence

- [x] v1.0 MVP — Phases 1-6
- [x] v1.1 Trusted Context — Phases 7-11
- [x] v1.2 Live Dispatch Truth and Recovery — Phases 12-16
- [x] v1.3 Visual Truth and Core Hardening — Phases 17-24 (shipped 2026-04-21)
- [x] v1.4 Self-Healing Colony — Phases 25-30 (completed 2026-04-21)
- [x] v1.5 Runtime Truth Recovery — Phases 31-38 (completed 2026-04-23, product v1.0.20)
- [x] v1.6 Release Pipeline Integrity — Phases 39-46 (completed 2026-04-24)
- [x] v1.7 Planning Pipeline Recovery — Phases 47-48 (completed 2026-04-24)

## Next Move

Awaiting v1.8 direction.

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

## Explicit Deferrals

These remain promising, but they are not the next best move until dispatch truth and recovery are solid:

- pheromone markets and reputation exchange
- swarm memory beyond the current hive/wisdom path
- federation / inter-colony coordination
- self-mutating agents / evolution engine
- more agents without first fixing dispatch truth, recovery, and runtime honesty
