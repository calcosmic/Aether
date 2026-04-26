# Aether

## What This Is

Aether is a biomimetic AI colony framework: a Go runtime in `cmd/` and `pkg/` that owns state, worker dispatch, verification, memory, and install/update flows, plus companion command surfaces for Claude Code and OpenCode and a runtime-native Codex CLI lane.

`v1.0` restored the lost colony ceremony and runtime visibility surfaces.

`v1.1` made Aether's context layer trustworthy, inspectable, deterministic, and benchmarkable.

`v1.8` added the colony recovery system: `aether recover` detects 7 stuck-state classes, auto-fixes safe issues, prompts for destructive ones, and proves correctness through 10 E2E tests.

## Core Value

**Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.**

That means:

- worker lifecycle must be inspectable and honest
- dispatch visibility must come from real runtime state
- stale run state must not poison future commands
- verification must lead advancement decisions
- partial success and recovery must be first-class
- stuck colonies must be recoverable with a single command

## Current State

- Go runtime is healthy, v1.0.24 shipped
- All 9 milestones complete (51 phases, 119 plans)
- 2910+ tests passing, full E2E regression coverage
- Stable and dev publish channels with integrity verification
- Plan recovery pipeline hardened (`--force` always recovers)
- Colony recovery system shipped: `aether recover` + `--apply` for stuck-state rescue
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

## Milestone Sequence

- [x] v1.0 MVP — Phases 1-6
- [x] v1.1 Trusted Context — Phases 7-11
- [x] v1.2 Live Dispatch Truth and Recovery — Phases 12-16
- [x] v1.3 Visual Truth and Core Hardening — Phases 17-24 (shipped 2026-04-21)
- [x] v1.4 Self-Healing Colony — Phases 25-30 (completed 2026-04-21)
- [x] v1.5 Runtime Truth Recovery — Phases 31-38 (completed 2026-04-23, product v1.0.20)
- [x] v1.6 Release Pipeline Integrity — Phases 39-46 (completed 2026-04-24)
- [x] v1.7 Planning Pipeline Recovery — Phases 47-48 (completed 2026-04-24)
- [x] v1.8 Colony Recovery — Phases 49-51 (completed 2026-04-25)

## Next Move

Plan next milestone with `/gsd-new-milestone`.

## Evolution

This document evolves at phase transitions and milestone boundaries.

*Last updated: 2026-04-26 after v1.8 milestone*

## Explicit Deferrals

These remain promising but are not the next best move:

- pheromone markets and reputation exchange
- swarm memory beyond the current hive/wisdom path
- federation / inter-colony coordination
- self-mutating agents / evolution engine
