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

- Binary install on update — `aether update` downloads binary if missing from PATH
- Version-gated YAML wiring — only swap Go-wired YAML when binary confirmed working

### Validated (v5.5 — partial)

- ✓ goreleaser config — produce platform binaries (darwin/linux/windows, amd64/arm64) — v5.5 (Phase 48)
- ✓ Binary downloader + npm install — users receive Go binary automatically on npm install -g — v5.5 (Phase 49)

### Validated (v5.4)

- ✓ Go CLI with Cobra — 254+ commands ported from shell, all slash commands and playbooks wired — v5.4
- ✓ Event bus — channels + JSONL replacing file-based pub/sub — v5.4
- ✓ Trust scoring — native float64 replacing bc subprocess — v5.4
- ✓ Memory pipeline — observation → instinct → QUEEN promotion — v5.4
- ✓ Graph layer — BFS, cycle detection, relationship tracking — v5.4
- ✓ Agent/worker system — goroutine pools replacing shell subprocesses — v5.4
- ✓ LLM integration — Anthropic Go SDK with streaming and tool-use loop — v5.4
- ✓ XML exchange — native Go replacing xmllint/xmlstarlet — v5.4
- ✓ Storage layer — typed Go structs, atomic writes, backup rotation, path resolution — v5.4
- ✓ Full test parity — 254 commands tested (142 parity + 217 smoke), CI integrated — v5.4
- ✓ Slash command wiring — 87 YAML sources calling Go binary — v5.4
- ✓ Playbook wiring — 11 playbooks (275 Go calls, 0 shell) — v5.4

### Out of Scope

- Full rewrite of aether-utils.sh — extract modules, don't rewrite from scratch
- Web/TUI dashboard — CLI tool, ASCII dashboards work in terminal
- Multi-repo colony coordination — future architecture work
- Performance optimization (state caching, lock backoff) — defer unless blocking
- Agent Teams inter-worker communication — subagents can't communicate mid-execution
- LLM-generated prompts — non-deterministic and untestable; bash + jq assembly is deterministic
- Full deep survey on every init — too slow; lightweight scan + suggestion instead

## Current Milestone: v5.5 Go Binary Release

**Goal:** Ship the Go binary as a downloadable release and wire the update flow to install it, eliminating the shell dependency for end users.

**Target features:**
- goreleaser config producing platform binaries (darwin/linux/windows, amd64/arm64)
- `aether update` auto-installs the binary if missing from PATH
- Version gate preventing YAML wiring until binary is confirmed working

**Previous: v5.4 Shell-to-Go Rewrite** (completed 2026-04-04)

## Context

- v5.4 shipped: Full shell-to-Go conversion — 254+ Go commands, 47 phases, 553 commits over 16 days
- v2.7 shipped: PR workflow, clash detection, pheromone propagation, release hygiene
- v2.6 shipped: Input escaping, cross-colony isolation, depth gating, YAML command generator
- v2.5 shipped: Smart init (repo scanning, charter management, approval flow), 50 new tests
- v2.4 shipped: Oracle + Architect agents, wisdom pipeline wiring, fuzzy dedup
- v2.3 shipped: Per-caste model routing, model-slot CLI, 24 agents configured
- v2.2 shipped: QUEEN.md structured wisdom, cross-colony hive brain, wisdom injection
- v2.1 shipped: Error handling hardened, monolith modularized (10 modules), state API centralized
- Go binary has 254+ commands replacing ~305 shell subcommands
- 47 phase directories completed, 16 with formal VERIFICATION.md

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
*Last updated: 2026-04-04 after v5.5 milestone started*
