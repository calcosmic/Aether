# Aether Colony Orchestration System

## What This Is

Aether is a multi-agent colony orchestration system for AI-assisted development. It provides 44 slash commands, 24 specialized worker agents, 28 skills (10 colony + 18 domain), a living pheromone signaling system, and intelligent colony initialization with repo scanning and charter management. Distributed as `aether-colony` on npm, it works with Claude Code and OpenCode.

## Core Value

The system must reliably interpret a user request, decompose it into executable work, verify outputs, and ship correct work with minimal user back-and-forth — not just look autonomous, but actually deliver.

## Requirements

### Validated

- Colony lifecycle works (init, plan, build, continue, seal, entomb) — existing
- 24 worker agents defined with caste roles — existing (updated from 22 in v2.4)
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
- ✓ QUEEN.md structured wisdom (4-section template, auto-populated by builds) — v2.2
- ✓ Cross-colony hive brain with domain-scoped retrieval — v2.2
- ✓ Per-caste model routing via slots (opus/sonnet/haiku) — v2.3
- ✓ Oracle and Architect agents with wisdom pipeline wiring — v2.4
- ✓ Fuzzy dedup for instincts + deterministic fallback learning extraction — v2.4
- ✓ Repo scanning module — tech stack, directory, git, survey, complexity in <2s — v2.5
- ✓ Charter management — colony-name + charter-write populating QUEEN.md v2 sections — v2.5
- ✓ Smart init — scan-assemble-approve-create flow with re-init safety — v2.5
- ✓ Intelligence enrichment — prior colony context, pheromone suggestions, governance inference — v2.5

### Active

- Go CLI with Cobra — all 37 commands ported from shell — v5.4
- ✓ Event bus — channels + JSONL replacing file-based pub/sub — v5.4 (Phase 46)
- Trust scoring — native float64 replacing bc subprocess — v5.4
- Memory pipeline — observation → instinct → QUEEN promotion — v5.4
- Graph layer — BFS, cycle detection, relationship tracking — v5.4
- Agent/worker system — goroutine pools replacing shell subprocesses — v5.4
- LLM integration — Anthropic Go SDK for agent Claude calls — v5.4
- XML exchange — native Go replacing xmllint/xmlstarlet — v5.4
- ✓ Storage layer — typed Go structs for all colony data, atomic writes, backup rotation, path resolution — Phase 45
- Full test parity — all existing tests ported to Go — v5.4
- Distribution — Go binary replacing npm package — v5.4

### Out of Scope

- Full rewrite of aether-utils.sh — extract modules, don't rewrite from scratch
- Web/TUI dashboard — CLI tool, ASCII dashboards work in terminal
- Multi-repo colony coordination — future architecture work
- Performance optimization (state caching, lock backoff) — defer unless blocking
- Agent Teams inter-worker communication — subagents can't communicate mid-execution
- LLM-generated prompts — non-deterministic and untestable; bash + jq assembly is deterministic
- Full deep survey on every init — too slow; lightweight scan + suggestion instead

## Current Milestone: v5.4 Shell-to-Go Rewrite

**Goal:** Replace all shell scripts with a native Go binary, eliminating bash/jq/curl dependencies while preserving exact behavioral parity with the existing system.

**Target features:**
- Complete Go implementation of all colony runtime (60+ shell scripts → Go packages)
- Cobra CLI with all 37 commands ported
- Event bus (channels + JSONL), trust scoring, memory pipeline, graph layer
- LLM integration via Anthropic Go SDK
- XML exchange layer (32 functions, native Go replacing xmllint)
- All existing tests ported + new Go tests for parity verification
- Distribution changes from npm package to Go binary

## Context

- Aether is at v5.3.2 on npm, transitioning from shell to Go
- v2.7 shipped: PR workflow, clash detection, pheromone propagation, release hygiene
- v2.6 shipped: Input escaping, cross-colony isolation, depth gating, YAML command generator
- v2.5 shipped: Smart init (repo scanning, charter management, approval flow), 50 new tests
- v2.4 shipped: Oracle + Architect agents, wisdom pipeline wiring, fuzzy dedup
- v2.3 shipped: Per-caste model routing, model-slot CLI, 24 agents configured
- v2.2 shipped: QUEEN.md structured wisdom, cross-colony hive brain, wisdom injection
- v2.1 shipped: Error handling hardened, monolith modularized (10 modules), state API centralized
- Phase 45 complete: typed Go structs for all colony data files (7 types), storage package (atomic writes, backup rotation, path resolution), 74 tests passing
- 60+ shell scripts, ~44 commands, 24 agents, ~150+ subcommands across 10 domain modules
- Oracle research completed (25 iterations, 100% confidence) covering full conversion spec

## Constraints

- **Testing**: All changes must maintain 616+ passing tests; new features need tests
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
| Oracle distribution fix | oracle.sh excluded from npm by .npmignore blanket rule | ✓ Good — moved to .aether/utils/oracle/, position-aware HUB_EXCLUDE |
| Extract modules not rewrite | 76 unused subcommands should be modularized, not deleted | ✓ Good — 9→10 domain modules extracted, 55% reduction |
| Deepen planning quality | Aether plans too quickly vs GSD's per-phase research depth | ✓ Good — Step 3.6 research scout + 16K builder context |
| Focus v2.2 on wisdom systems only | User test showed QUEEN.md and hive are dead features; ceremony/verification issues deferred | ✓ Good — v2.4 picks up where v2.2 left off |
| Per-caste model routing via slots | GLM-5 needs tight constraints for reasoning castes; opus/sonnet slots provide clean routing | ✓ Good — 24 agents configured, caste table static |
| Smart init as update-not-reset | Re-running init should update Queen file, not destroy colony state | ✓ Good — re-init skips template writes, charter-write updates in-place |
| Add governance to QUEEN.md | QUEEN.md should be a full colony charter (intent, vision, governance) not just wisdom | ✓ Good — charter-write populates existing v2 sections, no new headers |
| Intelligent colonize prompting | Users forget to colonize; system should suggest it at appropriate times | ✓ Good — scan detects stale/missing survey, suggests colonize in init prompt |
| Deterministic prompt generation | Colony prompts must be testable and reproducible | ✓ Good — bash + jq assembly, 12 tested pattern checks |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd:transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-01 after Phase 46 (event bus) completion*
