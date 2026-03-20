# Aether Colony Orchestration System

## What This Is

Aether is a multi-agent colony orchestration system for AI-assisted development. It provides 40 slash commands, 22 specialized worker agents, and a living pheromone signaling system that auto-emits signals during builds, carries context across sessions, and changes worker behavior based on colony intelligence.

## Core Value

The pheromone system is a living system — auto-emitting signals during builds, carrying context across sessions, and actually changing worker behavior — not just a storage format.

## Requirements

### Validated

- Colony lifecycle works (init, plan, build, continue, seal, entomb) — existing
- 22 worker agents defined with caste roles — existing
- State management with file locking and atomic writes — existing
- Pheromone signal storage (FOCUS/REDIRECT/FEEDBACK) — existing
- NPM distribution via `aether-colony` package — existing
- Multi-provider support (Claude Code + OpenCode) — existing
- Midden failure tracking system — existing
- QUEEN.md wisdom promotion pipeline — existing
- ✓ Clean colony state — all test artifacts purged — v1.3
- ✓ Pheromone injection chain — signals flow emit → store → inject → worker — v1.3
- ✓ Worker pheromone protocol — builder/watcher/scout act on signals — v1.3
- ✓ Learning pipeline — observations auto-promote to instincts in worker prompts — v1.3
- ✓ XML exchange — /ant:export-signals, /ant:import-signals, seal auto-export — v1.3
- ✓ Fresh install hardened — lifecycle smoke test + content-aware validate-package.sh — v1.3
- ✓ Documentation accuracy — all docs match verified behavior — v1.3
- ✓ 537+ tests passing (AVA + bash) — v1.3

### Active

(None — next milestone will define new requirements)

### Out of Scope

- Splitting aether-utils.sh into modules — large refactor, separate initiative
- Web/TUI dashboard — CLI tool, ASCII dashboards work in terminal
- Multi-repo colony coordination — future architecture work
- Performance optimization (state caching, lock backoff) — defer unless blocking
- Agent Teams inter-worker communication — subagents can't communicate mid-execution
- Per-worker model routing — Claude Code Task tool doesn't support per-subagent env vars

## Context

- Aether is at v1.3.0, published on npm as `aether-colony`
- v1.3 shipped: 8 phases, 17 plans, 49 commits in a single day
- 40 Claude commands, 40 OpenCode commands, 22 agents, 110 subcommands
- 10,499 lines in aether-utils.sh
- Pheromone system is fully integrated: signals auto-emit, persist across sessions, and influence worker behavior via pheromone_protocol sections
- Learning pipeline works end-to-end: observations promote to instincts that appear in colony-prime prompt_section
- XML exchange wired into commands and seal lifecycle
- Fresh install validated with isolated smoke test

## Constraints

- **Testing**: All changes must maintain 537+ passing tests; new features need tests
- **Compatibility**: Must work with bash 4+, Node 16+, jq 1.6+
- **Distribution**: Changes must pass `bin/validate-package.sh` (content-aware) before publish
- **No breaking changes**: Existing colonies using Aether must not break on update

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Activate XML system (don't archive) | Cross-colony signal transfer has clear value | ✓ Good — /ant:export-signals and /ant:import-signals created |
| Pheromone integration is top priority | User's primary pain point; signals exist but don't influence behavior | ✓ Good — full injection chain + worker protocols |
| Fresh install as "done" test | If someone can install and run a colony without issues, maintenance is complete | ✓ Good — 430-line smoke test validates full lifecycle |
| Clean before integrating | Test data must be purged before pheromone integration can be validated | ✓ Good — Phase 1 first, integration phases after |
| Principle-based agent protocols | Workers are LLMs — they understand intent, don't need 100-line rule sets | ✓ Good — 35 lines per agent, all effective |
| Define "influence" structurally | Signal in prompt + agent has protocol = maximum testable without live LLM | ✓ Good — pragmatic definition, fully tested |

---
*Last updated: 2026-03-20 after v1.3 milestone*
