# Aether Colony System

## What This Is

A colony-based development framework that orchestrates AI agents using an ant colony metaphor. The system provides slash commands (like `/ant:build`, `/ant:continue`, `/ant:swarm`) that spawn specialized worker agents to execute tasks in parallel. v1.0 delivers hardened infrastructure with comprehensive testing, error handling, and state restoration.

## Core Value

Autonomous multi-agent orchestration that scales from single-user development to team collaboration, with pheromone-based constraints guiding agent behavior.

## Requirements

### Validated (v1.0)

- ✓ CLI installation and update system — v1.0 (npm published)
- ✓ Core colony commands (init, build, continue, plan, phase) — v1.0
- ✓ Worker caste system (Builder, Watcher, Scout, Chaos, Oracle) — v1.0
- ✓ State management (COLONY_STATE.json, flags, constraints) — v1.0
- ✓ File locking infrastructure — v1.0 (flock-based with cleanup)
- ✓ Atomic write operations — v1.0 (temp file + mv pattern)
- ✓ Targeted git stashing for checkpoints — v1.0 (Aether-managed only)
- ✓ Signatures.json template — v1.0 (5 pattern definitions)
- ✓ Hash comparison for sync operations — v1.0 (SHA-256)
- ✓ Comprehensive test coverage — v1.0 (52+ AVA and bash tests)
- ✓ Centralized error handling — v1.0 (AetherError classes, sysexits.h)
- ✓ Structured logging — v1.0 (activity.log integration)
- ✓ Graceful degradation — v1.0 (feature flags pattern)
- ✓ commander.js CLI — v1.0 (migrated from manual parsing)
- ✓ Colored output — v1.0 (picocolors with semantic palette)
- ✓ State loading with locks — v1.0 (state-loader.sh)
- ✓ Spawn tree persistence — v1.0 (reconstruction from spawn-tree.txt)
- ✓ Context restoration — v1.0 (HANDOFF.md lifecycle)

### Validated (v1.1)

- ✓ Safe checkpoint system with explicit allowlist — v1.1 (never captures user data)
- ✓ State Guard with Iron Law enforcement — v1.1 (prevents phase advancement loops)
- ✓ FileLock with PID-based stale detection — v1.1 (prevents concurrent modification)
- ✓ Audit trail system with event sourcing — v1.1 (phase transition history)
- ✓ UpdateTransaction with two-phase commit — v1.1 (automatic rollback on failure)
- ✓ Build output timing fixed — v1.1 (foreground execution)
- ✓ E2E integration test suite — v1.1 (checkpoint → update → build workflow)
- ✓ Init copies system files from hub — v1.1 (auto-registers for update --all)

### Active (v1.2 Candidates)

- [ ] Worker caste specializations
- [ ] Enhanced swarm command visualization
- [ ] Real-time colony monitoring improvements
- [ ] Cross-repo collaboration features
- [ ] Version-aware update notifications (NOTIFY-01)
- [ ] Checkpoint recovery tracking (RECOVER-01)

### Out of Scope

- Web UI — CLI-first approach, target v2+
- Cloud deployment — local-first, repo-local state, target v2+
- OAuth/multi-user auth — single developer focus, target v2+
- Mobile support — Desktop CLI tool only

## Context

**Current State (post-v1.1):**
- **Shipped:** v1.1 Bug Fixes & Update System Repair with 3 phases, 14 plans, 25 requirements
- **LOC:** ~36k JavaScript, Bash utilities, JSON configuration
- **Test Coverage:** 209 tests (AVA unit + integration + E2E)
- **Tech Stack:** Node.js CLI, commander.js, picocolors, sinon, proxyquire

**Technical Environment:**
- Node.js CLI with commander.js (bin/cli.js)
- Bash utilities (aether-utils.sh with 59+ subcommands)
- State Guard with Iron Law enforcement (bin/lib/state-guard.js)
- Update Transaction with two-phase commit (bin/lib/update-transaction.js)
- FileLock with PID-based stale detection (bin/lib/file-lock.js)
- Markdown-based command definitions
- JSON state files with locking and atomic writes

**Recent Work (v1.1):**
- Fixed critical data loss risk: Safe checkpoint system with explicit allowlist
- Fixed phase advancement loops: Iron Law enforcement requires verification evidence
- Fixed update system reliability: Two-phase commit with automatic rollback
- Fixed build output timing: Foreground execution ensures accurate summaries
- Fixed init gap: Now copies system files and auto-registers repos
- Established comprehensive testing with mocked filesystem (209 tests)

**Known Issues Resolved in v1.1:**
- ✓ User data protected in checkpoints (explicit allowlist, never user data)
- ✓ Phase advancement loops prevented (Iron Law enforcement)
- ✓ Update failures recoverable (automatic rollback + recovery commands)
- ✓ Build output timing accurate (foreground execution)
- ✓ New repos properly initialized (system files + registry)

## Constraints

- **Tech Stack**: Node.js >= 16, Bash, jq — Minimal external dependencies
- **Distribution**: npm package (aether-colony)
- **Platform**: macOS/Linux, Claude Code and OpenCode support
- **State**: Repo-local only (no cloud dependencies)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Repo-local state | Each project has own .aether/data/ | ✓ Good |
| Markdown commands | Easy to edit, version control | ✓ Good |
| Bash utilities | Works across shells, no Python dep | ✓ Good |
| Colony metaphor | Clear role separation | ✓ Good |
| AVA over Jest | Lightweight, fast ES module support | ✓ Good (209+ tests) |
| picocolors over chalk | 14x smaller, 2x faster | ✓ Good |
| Semantic color naming | queen, colony, worker hierarchy | ✓ Good |
| File locking with flock | Prevent race conditions | ✓ Good |
| Handoff pattern | Pheromone trail metaphor | ✓ Good |
| Checkpoint allowlist (v1.1) | Never risk user data loss | ✓ Good |
| Iron Law enforcement (v1.1) | Prevent phase advancement loops | ✓ Good |
| Two-phase commit updates (v1.1) | Reliable cross-repo sync | ✓ Good |
| Foreground build execution (v1.1) | Accurate output timing | ✓ Good |
| Init auto-registration (v1.1) | Seamless update --all | ✓ Good |

## Current Milestone: Planning v1.2

**Goal:** Fix critical bugs causing phase loops and repair the update system for reliable multi-repo synchronization

**Next Milestone Goals (v1.2):**
- Worker caste specializations (Builder, Watcher, Scout refinements)
- Enhanced visualization for swarm command
- Real-time monitoring improvements
- Version-aware update notifications
- Checkpoint recovery tracking

---
*Last updated: 2026-02-14 — v1.1 shipped, planning v1.2*
