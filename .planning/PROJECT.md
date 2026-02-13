# Aether Colony System

## What This Is

A colony-based development framework that orchestrates AI agents using an ant colony metaphor. The system provides slash commands (like `/ant:build`, `/ant:continue`, `/ant:swarm`) that spawn specialized worker agents to execute tasks in parallel. Currently used for developing and maintaining itself.

## Core Value

Autonomous multi-agent orchestration that scales from single-user development to team collaboration, with pheromone-based constraints guiding agent behavior.

## Requirements

### Validated

- ✓ CLI installation and update system — existing (npm published)
- ✓ Core colony commands (init, build, continue, plan, phase) — existing
- ✓ Worker caste system (Builder, Watcher, Scout, Chaos, Oracle) — existing
- ✓ State management (COLONY_STATE.json, flags, constraints) — existing
- ✓ File locking infrastructure — existing (recently fixed)
- ✓ Atomic write operations — existing (recently fixed)
- ✓ Targeted git stashing for checkpoints — existing (recently fixed)

### Active

- [ ] Improve update system reliability
- [ ] Add comprehensive test coverage
- [ ] Enhance error recovery mechanisms
- [ ] Add telemetry/metrics for colony health

### Out of Scope

- Web UI — CLI-first approach
- Cloud deployment — local-first, repo-local state
- OAuth/multi-user auth — single developer focus

## Context

**Technical Environment:**
- Node.js CLI (bin/cli.js)
- Bash utilities (aether-utils.sh with 59 subcommands)
- Markdown-based command definitions
- JSON state files

**Recent Work:**
- Just completed fixing critical bugs: file locking, atomic writes, targeted git stashing
- These fixes resolved race conditions and data loss risks

**Known Issues to Address:**
- Update command not working properly (user reported)
- System getting stuck in loops (user reported)
- Need more robust error handling

## Constraints

- **Tech Stack**: Node.js >= 16, Bash, jq — No external frameworks
- **Distribution**: npm package (aether-colony)
- **Platform**: macOS/Linux, Claude Code and OpenCode support

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Repo-local state | Each project has own .aether/data/ | ✓ Good |
| Markdown commands | Easy to edit, version control | ✓ Good |
| Bash utilities | Works across shells, no Python dep | ✓ Good |
| Colony metaphor | Clear role separation | ✓ Good |

---
*Last updated: 2026-02-13 after bug fixes*
