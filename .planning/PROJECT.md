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

- Go runtime is healthy and release-ready
- `v1.3` is complete and shipped 2026-04-21
- Visual UX fully restored: caste identity, stage separators, emoji consistency, Codex parity
- Core paths hardened: resume freshness checks, gate integration, transactional updates, model slot system
- Recovery made reliable: worktree lifecycle fixes, deferred cleanup, orphan branch detection
- Full trace instrumentation: every run leaves a structured trace with replay, summary, and inspect capabilities
- All 49 slash commands verified working across Claude Code, OpenCode, and Codex CLI
- Previous milestones: v1.0 (MVP), v1.1 (Trusted Context), v1.2 (Live Dispatch Truth), v1.3 (Visual Truth) all shipped
- Defining v1.4: Self-Healing Colony with Medic ant

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
- [ ] v1.4 Self-Healing Colony — Phases 25+ (defining)

## Previous Milestone Result

`v1.3` is complete at source commit `1b6829b5` (`Backfill gap closure artifacts for phases 19-23`).

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

## Current Milestone: v1.4 Self-Healing Colony

**Goal:** Give Aether the ability to diagnose and repair its own colony data, ceremony integrity, and runtime state — reducing manual intervention and preventing the documentation gaps we saw in v1.3.

**Target features:**

- **Medic Ant** (`/ant:medic`, `aether medic`) — Diagnose colony health across all data files
- **Colony Health Scan** — Detect corrupted JSON, stale state, missing artifacts, wrapper/runtime drift
- **Auto-Repair** — Fix common issues (clear stale spawns, rebuild indexes, reconcile orphaned worktrees)
- **Medic Skill** — `.aether/skills/colony/medic.md` documenting healthy state for every colony file
- **Ceremony Integrity Check** — Verify stage markers, emoji consistency, context-clear guidance
- **Trace Diagnostic** — Use trace logs to debug issues in other repos without access

## Next Move

Define requirements and roadmap for v1.4, then execute phases.

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
