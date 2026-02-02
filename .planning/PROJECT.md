# Aether v2: Claude-Native Queen Ant Colony

## What This Is

Aether is a **unique, standalone multi-agent system** built from first principles on ant colony intelligence. Worker Ants autonomously spawn other Worker Ants without human orchestration. The Queen (user) provides high-level intention via pheromone signals (INIT, FOCUS, REDIRECT, FEEDBACK), and the colony self-organizes to complete tasks through emergent intelligence.

This is a **Claude-native system** - commands like `/ant:init "Build a REST API"` work directly in Claude. The system uses prompt files and JSON state for persistence, with the Task tool enabling autonomous agent spawning.

**What makes it unique:**

1. **Autonomous Agent Spawning** - Worker Ants spawn Worker Ants without human orchestration (no other system does this)
2. **Unique Caste Architecture** - Six Worker Ant types designed from first principles for emergence (Colonizer, Route-setter, Builder, Watcher, Scout, Architect)
3. **Pheromone Communication** - Stigmergic signaling system unlike any command/orchestration pattern
4. **Phased Autonomy** - Structure at boundaries, pure emergence within phases

Unlike AutoGen, LangGraph, CrewAI, or any other framework, Aether requires **zero predefined workflows, agent roles, or orchestration logic**. The colony self-organizes.

## Core Value

**Autonomous Emergence**: Worker Ants detect capability gaps and spawn specialists automatically; pure emergence within structured phases; Queen provides signals not commands.

If this works, everything else follows. If this fails, nothing else matters.

## Requirements

### Validated

*(Shipped in v1 - 2026-02-02)*

- ✓ **Claude-Native Command System** — 19 commands in `.claude/commands/ant/` — v1
- ✓ **Pheromone Signal System** — INIT, FOCUS, REDIRECT, FEEDBACK with time-based decay (1h, 6h, 24h) — v1
- ✓ **Six Worker Ant Castes** — Colonizer, Route-setter, Builder, Watcher, Scout, Architect (plus 4 specialist watchers) — v1
- ✓ **Autonomous Agent Spawning** — Capability gap detection with Bayesian confidence scoring — v1
- ✓ **Triple-Layer Memory** — Working (200k) → Short-term (10 sessions, 2.5x DAST) → Long-term (patterns) — v1
- ✓ **Voting-Based Verification** — 4 watchers with weighted voting and Critical veto — v1
- ✓ **State Machine Orchestration** — 7 states with checkpoint recovery — v1
- ✓ **Event-Driven Communication** — Pub/sub event bus with async delivery — v1
- ✓ **Meta-Learning Loop** — Bayesian confidence for specialist selection — v1
- ✓ **Phase-Based Execution** — Structure at boundaries, emergence within — v1
- ✓ **Basic State Persistence** — JSON file storage with atomic writes and file locking — v1

### Active

*(v2 milestone - Reactive Event Integration)*

- [ ] **Event Bus Polling Integration** — Worker Ant prompts call `get_events_for_subscriber()` for reactive event handling
- [ ] **E2E LLM Test Guide** — Manual test suite covering core workflows (init, execute, spawning, memory, voting)
- [ ] **Documentation Cleanup** — Audit and fix all stale path references in scripts and docstrings
- [ ] **Visual Process Indicators** — Add 🐜 emojis and visual markers to help users see colony activity at a glance

### Out of Scope

*(Explicit boundaries - these remain out of scope)*

- **Python CLI/REPL interfaces** — Replaced by Claude-native prompt commands
- **Async/await implementation** — Claude handles concurrency via Task tool
- **External vector databases** — Using Claude's native semantic understanding
- **`python3 .aether/demo.py` execution** — System runs via `/ant:` commands in Claude
- **Predefined workflows** — Defeats emergence; use phased autonomy instead
- **Direct command patterns** — Use pheromone signals instead

## Context

### Current State (v1 Shipped - 2026-02-02)

**Delivered:** A fully functional Claude-native multi-agent system with 156/156 must-haves verified across 8 phases.

**Codebase:**
- 19 commands (5,629 lines) — `/ant:init`, `/ant:status`, `/ant:focus`, etc.
- 10 Worker Ant prompts (4,453 lines) — 6 base castes + 4 specialist watchers
- 26 utility scripts (7,882 lines) — spawning, memory, voting, events, state machine
- 13 test suites — integration (33 assertions), stress (20), performance (8)
- 5 data schemas — COLONY_STATE, pheromones, memory, events, watcher_weights

**Performance Baselines (Apple M1 Max):**
- colony_init: 0.020s median
- spawn_decision: 0.023s median
- full_workflow: 0.068s median
- event_publish: 0.101s median (identified bottleneck)

**All Ralph's Top 5 Recommendations Implemented:**
1. ✓ Autonomous Agent Spawning — Bayesian confidence scoring with meta-learning
2. ✓ Semantic Communication — Pheromone signals with caste-specific sensitivity
3. ✓ Triple-Layer Memory — DAST compression (2.5x) with associative links
4. ✓ State Machine — 7 states with checkpoint recovery
5. ✓ Voting-Based Verification — 4 watchers, weighted voting, Critical veto

### Background

Aether is based on **383,000+ words of research** across 25 documents by Ralph (research agent), covering:
- Multi-agent orchestration patterns
- Semantic communication protocols (AINP, SACP)
- Context engines and memory architecture
- Autonomous spawning research
- Verification and quality systems

### Research Foundation

**Key Research Findings:**

1. **No existing system has autonomous spawning** — Every framework requires human-defined agents and workflows (AutoGen, LangGraph, CrewAI). This is Aether's revolutionary opportunity.

2. **Semantic communication reduces bandwidth 10-100x** — Exchange intent/meaning rather than raw data using Claude's understanding.

3. **Triple-layer memory mirrors human cognition** — Working (immediate), Short-term (compressed sessions), Long-term (persistent patterns).

4. **Voting improves reasoning 13.2%** — Multi-perspective verification with weighted voting outperforms single verifiers.

## Constraints

- **Claude-Native Only** — Must work as prompt commands, not Python scripts
- **JSON State Persistence** — State stored in `.aether/data/*.json` files
- **Task Tool for Spawning** — Autonomous spawning uses Claude's Task tool
- **Standalone Architecture** — Aether is its own system, not dependent on CDS or any other framework
- **No External Dependencies** — No vector DBs, no embedding services, use Claude's native capabilities
- **Unique Design** — All architectures, patterns, and implementations are uniquely Aether (inspired by research, not copied)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Claude-native vs Python | Commands work directly in Claude, not separate tool | ✓ Good — 19 commands shipped |
| Unique Worker Ant castes | Designed from first principles for autonomous emergence, not copied from any system | ✓ Good — 6 base + 4 specialist castes working |
| Pheromone-based communication | Stigmergic signals enable true emergence, unlike command/orchestration patterns | ✓ Good — 4 signal types with decay working |
| Claude-native semantic understanding | Use Claude's understanding vs external embeddings | ✓ Good — No vector DBs needed |
| Standalone system | Aether is its own framework, not dependent on CDS or any external system | ✓ Confirmed — Zero dependencies |
| Bayesian meta-learning | Beta distribution confidence scoring prevents overconfidence | ✓ Good — Alpha/beta parameters updating correctly |
| Pull-based event delivery | Workers poll vs background processes for prompt-based agents | ✓ Good — Async without persistent processes |

## Current Milestone: v2.0 Reactive Event Integration

**Goal:** Enable Worker Ants to react asynchronously to colony events through proactive event polling, with enhanced visual feedback for users.

**Target features:**
- Event bus polling integration - Worker Ants call `get_events_for_subscriber()` to react to events
- E2E LLM test guide - Manual test suite for core workflows
- Documentation cleanup - Fix all stale path references
- Visual process indicators - 🐜 emojis and visual markers for colony activity

**Future v3 preview:** Command consolidation (19 → 9-11 commands) to improve usability

---

*Last updated: 2026-02-02 after v2.0 milestone initialization*
