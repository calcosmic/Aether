# Aether Colony System

## What This Is

A colony-based development framework that orchestrates AI agents using an ant colony metaphor. The system provides slash commands (like `/ant:build`, `/ant:continue`, `/ant:swarm`) that spawn specialized worker agents to execute tasks in parallel. v1.0 delivers hardened infrastructure with comprehensive testing, error handling, and state restoration.

## Core Value

Autonomous multi-agent orchestration that scales from single-user development to team collaboration, with pheromone-based constraints guiding agent behavior.

## Requirements

### Validated

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

### Active

- [ ] Worker caste specializations (v1.1)
- [ ] Enhanced swarm command visualization (v1.1)
- [ ] Real-time colony monitoring improvements (v1.1)
- [ ] Cross-repo collaboration features (v1.1)

### Out of Scope

- Web UI — CLI-first approach, target v2+
- Cloud deployment — local-first, repo-local state, target v2+
- OAuth/multi-user auth — single developer focus, target v2+
- Mobile support — Desktop CLI tool only

## Context

**Current State (post-v1.0):**
- **Shipped:** v1.0 Infrastructure with 5 phases, 14 plans, 16 requirements
- **LOC:** ~230k across JavaScript, Bash, JSON
- **Test Coverage:** 52+ tests (AVA unit + bash integration)
- **Tech Stack:** Node.js CLI, commander.js, picocolors, bash utilities

**Technical Environment:**
- Node.js CLI with commander.js (bin/cli.js)
- Bash utilities (aether-utils.sh with 59+ subcommands)
- Markdown-based command definitions
- JSON state files with locking and atomic writes

**Recent Work (v1.0):**
- Fixed Oracle-discovered bugs: missing signatures.json, hash comparison, CLI clarity
- Established comprehensive testing foundation
- Built centralized error handling with graceful degradation
- Migrated to commander.js with semantic color palette
- Implemented state loading with file locking and handoff detection

**Known Issues Resolved in v1.0:**
- ✓ Update command reliability (hash comparison prevents unnecessary writes)
- ✓ System getting stuck in loops (file locking prevents race conditions)
- ✓ Error handling gaps (AetherError classes with recovery suggestions)

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
| AVA over Jest | Lightweight, fast ES module support | ✓ Good (52+ tests) |
| picocolors over chalk | 14x smaller, 2x faster | ✓ Good |
| Semantic color naming | queen, colony, worker hierarchy | ✓ Good |
| File locking with flock | Prevent race conditions | ✓ Good |
| Handoff pattern | Pheromone trail metaphor | ✓ Good |

## Current Milestone: v1.1 Bug Fixes & Update System Repair

**Goal:** Fix critical bugs causing phase loops and repair the update system for reliable multi-repo synchronization

**Target fixes:**
- Fix phase advancement logic to prevent AI model from repeating the same phases
- Repair `aether update` command for reliable cross-repo synchronization
- Fix build checkpoint stashing user data (critical data loss risk)
- Fix misleading output timing from `run_in_background` in build commands
- Add missing package-lock.json for deterministic builds
- Add unit tests for core sync functions in cli.js

---
*Last updated: 2026-02-14 — starting v1.1 milestone*
