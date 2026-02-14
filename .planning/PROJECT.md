# Aether Colony System

## What This Is

A colony-based development framework that orchestrates AI agents using an ant colony metaphor. The system provides slash commands (like `/ant:build`, `/ant:continue`, `/ant:swarm`) that spawn specialized worker agents to execute tasks in parallel. v1.0 delivers hardened infrastructure with comprehensive testing, error handling, and state restoration.

## Core Value

Autonomous multi-agent orchestration that scales from single-user development to team collaboration, with pheromone-based constraints guiding agent behavior.

## Requirements

### Validated (v3.0.0)

- ✓ CLI installation and update system — v3.0.0 (npm published)
- ✓ Core colony commands (init, build, continue, plan, phase) — v3.0.0
- ✓ Worker caste system (Builder, Watcher, Scout, Chaos, Oracle) — v3.0.0
- ✓ State management (COLONY_STATE.json, flags, constraints) — v3.0.0
- ✓ File locking infrastructure — v3.0.0 (flock-based with cleanup)
- ✓ Atomic write operations — v3.0.0 (temp file + mv pattern)
- ✓ Safe checkpoint system with explicit allowlist — v3.0.0 (never captures user data)
- ✓ State Guard with Iron Law enforcement — v3.0.0 (prevents phase advancement loops)
- ✓ FileLock with PID-based stale detection — v3.0.0 (prevents concurrent modification)
- ✓ Audit trail system with event sourcing — v3.0.0 (phase transition history)
- ✓ Signatures.json template — v3.0.0 (5 pattern definitions)
- ✓ SHA-256 hash comparison for sync operations — v3.0.0
- ✓ UpdateTransaction with two-phase commit — v3.0.0 (automatic rollback on failure)
- ✓ Comprehensive test coverage — v3.0.0 (209+ AVA, integration, and E2E tests)
- ✓ Centralized error handling — v3.0.0 (AetherError classes, sysexits.h)
- ✓ Structured logging — v3.0.0 (activity.log integration)
- ✓ Build output timing fixed — v3.0.0 (foreground execution)
- ✓ E2E integration test suite — v3.0.0 (checkpoint → update → build workflow)
- ✓ Init copies system files from hub — v3.0.0 (auto-registers for update --all)
- ✓ commander.js CLI — v3.0.0 (migrated from manual parsing)
- ✓ Colored output — v3.0.0 (picocolors with semantic palette)
- ✓ State loading with locks — v3.0.0 (state-loader.sh)
- ✓ Spawn tree persistence — v3.0.0 (reconstruction from spawn-tree.txt)
- ✓ Context restoration — v3.0.0 (HANDOFF.md lifecycle)

### Active (v3.1 Candidates)

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

**Current State (post-v3.0.0):**
- **Shipped:** v3.0.0 Core Reliability & State Management with 3 phases, 14 plans, 25 requirements
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

**Recent Work (v3.0.0):**
- Fixed critical data loss risk: Safe checkpoint system with explicit allowlist
- Fixed phase advancement loops: Iron Law enforcement requires verification evidence
- Fixed update system reliability: Two-phase commit with automatic rollback
- Fixed build output timing: Foreground execution ensures accurate summaries
- Fixed init gap: Now copies system files and auto-registers repos
- Established comprehensive testing with mocked filesystem (209 tests)

**Known Issues Resolved in v3.0.0:**
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
| Checkpoint allowlist (v3.0.0) | Never risk user data loss | ✓ Good |
| Iron Law enforcement (v3.0.0) | Prevent phase advancement loops | ✓ Good |
| Two-phase commit updates (v3.0.0) | Reliable cross-repo sync | ✓ Good |
| Foreground build execution (v3.0.0) | Accurate output timing | ✓ Good |
| Init auto-registration (v3.0.0) | Seamless update --all | ✓ Good |

## Current Milestone: v3.1 Open Chambers

**Goal:** Implement intelligent model routing for worker castes, establish colony lifecycle management (archive/foundation) with ant-themed terminology, and create an immersive real-time visualization experience

**Target features:**
- **Model Routing System**: Verify and document caste-to-model assignments (glm-5 for prime/architect, kimi-k2.5 for builder, etc.)
- **Model Configuration**: Easy CLI commands to view/set models (`aether models`, `/ant:models`)
- **Colony Lifecycle**: `/ant:archive` (archive + reset), `/ant:foundation` (start fresh) — ant-themed equivalents to CDS milestone commands
- **Milestone Auto-Detection**: Automatically detect colony milestone (First Mound, Open Chambers, Brood Stable, etc.) based on state
- **Immersive Colony Visualization**: Real-time agent activity display with collapsible tree views, tool usage stats, token counts, and progress indicators — like CDS spawning indicators but with ant-themed presentation (ants working, pheromone trails, chamber activity). Semantic color coding (cyan for agents, yellow for timing, green for completion) for visual clarity

---
*Last updated: 2026-02-14 — starting v3.1 Open Chambers*
