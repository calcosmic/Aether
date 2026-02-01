# Aether v2: Claude-Native Queen Ant Colony

## What This Is

Aether is a revolutionary multi-agent system where Worker Ants autonomously spawn other Worker Ants without human orchestration. The user (Queen) provides high-level intention via pheromone signals (INIT, FOCUS, REDIRECT, FEEDBACK), and the colony self-organizes to complete tasks through emergent intelligence.

This is a **Claude-native system** - commands like `/ant:init "Build a REST API"` work directly in Claude (like CDS), not as Python scripts. The system uses prompt files and JSON state for persistence, with the Task tool enabling autonomous agent spawning.

**What makes it unique:** No existing system (AutoGen, LangGraph, CrewAI) supports true autonomous agent spawning - they all require humans to define agent roles, workflows, and orchestration logic. Aether will be the first.

## Core Value

**Autonomous Emergence**: Worker Ants detect capability gaps and spawn specialists automatically; pure emergence within structured phases; Queen provides signals not commands.

If this works, everything else follows. If this fails, nothing else matters.

## Requirements

### Validated

*(Already working in existing Python system)*

- ✓ **Pheromone Signal System** — INIT, FOCUS, REDIRECT, FEEDBACK signals with decay
- ✓ **Six Worker Ant Castes** — Mapper, Planner, Executor, Verifier, Researcher, Synthesizer
- ✓ **Phase-Based Execution** — Structure at boundaries, emergence within
- ✓ **Basic State Persistence** — JSON file storage (`.aether/COLONY_STATE.json`)

### Active

*(Current scope - building toward these)*

- [ ] **Claude-Native Command System** — Commands are prompt files in `.claude/commands/ant/`, execute directly in Claude
- [ ] **Autonomous Agent Spawning** — Workers detect capability gaps and spawn specialists via Task tool (Ralph #1)
- [ ] **Semantic Communication Layer** — Intent-based communication using Claude's native understanding (Ralph #2)
- [ ] **Triple-Layer Memory** — Working (200k) → Short-term (DAST 2.5x) → Long-term with associative links (Ralph #3)
- [ ] **Voting-Based Verification** — Multiple verifiers with weighted voting and belief calibration (Ralph #5)
- [ ] **State Machine Orchestration** — Explicit states, transitions, checkpointing for reliability
- [ ] **Event-Driven Communication** — Pub/sub backbone for scalable asynchronous coordination
- [ ] **Meta-Learning Loop** — Bayesian confidence scoring for specialist selection improvement

### Out of Scope

*(Explicit boundaries - these are the Python system, not Claude-native)*

- **Python CLI/REPL interfaces** — Replaced by Claude-native prompt commands
- **Async/await implementation** — Claude handles concurrency via Task tool
- **External vector databases** — Using Claude's native semantic understanding
- **`python3 .aether/demo.py` execution** — System runs via `/ant:` commands in Claude

## Context

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

### Existing Codebase

The Python system (`.aether/*.py`) demonstrates:
- Pheromone system with signal decay
- Six Worker Ant implementations
- Phase engine with state machine
- Triple-layer memory system
- Meta-learning for specialist selection

**Our job:** Extract the concepts, discard the implementation, rebuild as Claude-native prompts.

### Ralph's Top 5 Recommendations (All in v1)

1. **Autonomous Agent Spawning** (HIGH) — Agents detect capability gaps and spawn specialists
2. **Semantic Communication Layer** (HIGH) — Intent-based communication, 10-100x bandwidth reduction
3. **Triple-Layer Memory** (HIGH) — Working → Short-term (DAST) → Long-term with associative links
4. **State Machine Orchestration** (MEDIUM) — Explicit states, checkpointing, reliability
5. **Voting-Based Verification** (MEDIUM) — Multi-perspective with weighted voting, 13.2% improvement

## Constraints

- **Claude-Native Only** — Must work as prompt commands, not Python scripts
- **JSON State Persistence** — State stored in `.aether/data/*.json` files
- **Task Tool for Spawning** — Autonomous spawning uses Claude's Task tool
- **CDS for Development** — Using CDS framework to manage this project (temporary, will remove)
- **No External Dependencies** — No vector DBs, no embedding services, use Claude's native capabilities

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Claude-native vs Python | Commands work directly in Claude like CDS, not separate tool | ✓ Good |
| Hybrid spawning architecture | Task tool for execution + prompts for specialist creation | — Pending |
| Claude-native semantic understanding | Use Claude's understanding vs external embeddings | — Pending |
| All 4 HIGH PRIORITY in v1 | Ralph's research shows these are the revolutionary features | — Pending |
| CDS for development | Use CDS to build Aether, then remove both CDS and Ralph | — Pending |

---
*Last updated: 2025-02-01 after project initialization*
