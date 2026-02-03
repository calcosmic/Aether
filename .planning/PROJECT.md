# Aether v3: Claude-Native Queen Ant Colony

## What This Is

Aether is a **unique, standalone multi-agent system** built from first principles on ant colony intelligence. Worker Ants autonomously spawn other Worker Ants without human orchestration. The Queen (user) provides high-level intention via pheromone signals (INIT, FOCUS, REDIRECT, FEEDBACK), and the colony self-organizes to complete tasks through emergent intelligence.

This is a **hybrid prompt+code system** — commands like `/ant:init "Build a REST API"` work directly in Claude Code as skill prompts. Prompts handle reasoning and orchestration; a thin shell utility layer (`aether-utils.sh`) handles deterministic operations (pheromone math, state validation, memory management, error tracking) that LLMs get wrong.

**What makes it unique:**

1. **Autonomous Agent Spawning** — Worker Ants spawn Worker Ants without human orchestration (no other system does this)
2. **Unique Caste Architecture** — Six Worker Ant types with specialist watcher modes, designed from first principles for emergence
3. **Pheromone Communication** — Stigmergic signaling with exponential decay, caste sensitivity profiles, and combination effects
4. **Phased Autonomy** — Structure at boundaries, pure emergence within phases
5. **Colony Memory** — Error tracking, phase learnings, and event awareness that persists across sessions

Unlike AutoGen, LangGraph, CrewAI, or any other framework, Aether requires **zero predefined workflows, agent roles, or orchestration logic**. The colony self-organizes.

## Core Value

**Autonomous Emergence**: Worker Ants detect capability gaps and spawn specialists automatically; pure emergence within structured phases; Queen provides signals not commands.

If this works, everything else follows. If this fails, nothing else matters.

## Requirements

### Validated

*(Shipped in v1 — 2026-02-02)*

- ✓ **Claude-Native Command System** — commands as Claude Code skill prompts — v1
- ✓ **Pheromone Signal System** — INIT, FOCUS, REDIRECT, FEEDBACK with time-based decay (1h, 6h, 24h) — v1
- ✓ **Six Worker Ant Castes** — Colonizer, Route-setter, Builder, Watcher, Scout, Architect — v1
- ✓ **Autonomous Agent Spawning** — Recursive spawning via Task tool with full spec propagation — v1
- ✓ **Phase-Based Execution** — Structure at boundaries, emergence within — v1
- ✓ **Basic State Persistence** — JSON file storage with atomic writes and file locking — v1

*(Shipped in v2 — 2026-02-02)*

- ✓ **Event Polling Integration** — Worker Ants check events at execution boundaries — v2
- ✓ **Visual Process Indicators** — Emoji status, step progress, pheromone bars — v2
- ✓ **E2E Test Guide** — 94 verification checks across 6 workflows — v2

*(Shipped in v3-rebuild — 2026-02-03)*

- ✓ **Claude-Native Command Execution** — Commands use Read/Write tools directly, not bash/jq — v3
- ✓ **Clean State Schema** — Minimal JSON: COLONY_STATE, pheromones, PROJECT_PLAN — v3
- ✓ **Phase Lead Emergence Model** — One ant spawned per phase, self-organizes everything — v3
- ✓ **Recursive Spec Propagation** — Spawned ants get full spec + pheromones at any depth — v3

### Active

*(v4.0 milestone — Hybrid Foundation)*

- [ ] **Utility Layer** — `aether-utils.sh` wrapper script with subcommands for deterministic operations
- [ ] **Pheromone Math Engine** — Decay calculation, signal combination, effective strength computation in shell
- [ ] **State Validator** — Schema validation for all JSON state files, prevents field drift and corruption
- [ ] **Memory Operations** — Token counting, memory compression, eviction logic in shell
- [ ] **Error Tracker** — Pattern counting, category aggregation, deduplication in shell
- [ ] **Audit Fix: file-lock sourcing** — Fix broken dependency chain in atomic-write.sh
- [ ] **Audit Fix: state field consistency** — Resolve 3 "goal" fields and 2 "current_phase" fields to single canonical paths
- [ ] **Audit Fix: race conditions** — Fix temp file creation and concurrent state access
- [ ] **Audit Fix: jq error handling** — Add error handling for jq failures to prevent silent corruption
- [ ] **Audit Fix: state backups** — Add backup rotation before critical state updates
- [ ] **Audit Fix: pheromone schema** — Reconcile old vs new field names between creates and reads
- [ ] **Audit Fix: state integrity** — Add validation on state file load
- [ ] **Command Integration** — Update command prompts to call aether-utils.sh for deterministic operations

### Out of Scope

- **Python CLI/REPL interfaces** — Replaced by Claude-native prompt commands
- **Large bash systems** — v2's 879-line event-bus.sh was too complex; utilities stay thin (<300 lines total)
- **Node.js/Python utility layer** — Shell keeps zero external dependencies
- **Separate /ant:errors command** — Error display integrated into /ant:status
- **Separate /ant:review command** — Review integrated into /ant:continue
- **Separate /ant:memory command** — Memory state shown in /ant:status
- **Separate /ant:adjust command** — Use /ant:focus, /ant:redirect, /ant:feedback directly
- **Separate /ant:recover command** — Recovery integrated into /ant:resume-colony
- **External vector databases** — Using Claude's native semantic understanding
- **Predefined workflows** — Defeats emergence; use phased autonomy instead
- **Code for reasoning/orchestration** — Prompts handle decisions; code handles math

## Context

### Current State (post v3.0 — 2026-02-03)

**What exists (working):**
- 12 commands as Claude Code skill prompts (init, plan, build, status, phase, continue, focus, redirect, feedback, pause-colony, resume-colony, colonize, ant)
- 6 worker ant specs (~200 lines each) with pheromone math, spawning scenarios, event awareness
- 6 state files: COLONY_STATE.json, pheromones.json, PROJECT_PLAN.json, errors.json, memory.json, events.json
- 2 utility scripts: atomic-write.sh, file-lock.sh
- Full visual identity: box-drawing headers, step progress, pheromone decay bars
- 4 specialist watcher modes in watcher-ant.md
- Spawn outcome tracking with Bayesian confidence

**What's broken (from audit):**
- 3 critical issues: file-lock dependency chain, inconsistent state fields, missing utility functions
- 5 high issues: race conditions, jq error handling, missing backups, schema mismatches, no integrity validation
- Pure-prompt approach unreliable for: pheromone decay math, memory token counting, error pattern detection, schema enforcement

**What's needed (v4.0 hybrid):**
- Thin utility layer for deterministic operations
- Audit fixes for production stability
- Command updates to call utilities where appropriate

### Background

Aether is based on **383,000+ words of research** across 25 documents by Ralph (research agent), covering:
- Multi-agent orchestration patterns
- Semantic communication protocols (AINP, SACP)
- Context engines and memory architecture
- Autonomous spawning research (confirmed: no existing system has autonomous spawning)
- Verification and quality systems

## Constraints

- **Hybrid Architecture** — Prompts for reasoning/orchestration, shell scripts for deterministic operations
- **JSON State Persistence** — All state in `.aether/data/*.json` files
- **Task Tool for Spawning** — Autonomous spawning uses Claude's Task tool with full spec injection
- **Standalone Architecture** — Aether is its own system, not dependent on CDS or any framework
- **No External Dependencies** — No vector DBs, no embedding services, no Node.js, no Python
- **Shell Utilities Only** — Utility layer uses bash+jq only, stays thin (<300 lines total)
- **No New Commands** — Functionality enriched in existing 12 commands, not new ones
- **Unique Design** — All architectures uniquely Aether (inspired by research, not copied)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Claude-native vs Python | Commands work directly in Claude, not separate tool | ✓ Good — 12 commands working |
| Read/Write tools vs bash/jq | Claude manipulates JSON directly, more reliable than shell scripts | ✓ Good — v3 rebuild proved this works |
| Phase Lead emergence | One ant spawned per phase, self-organizes | ✓ Good — true emergence achieved |
| Consolidate commands (19 → 12) | Fewer, richer commands over many thin ones | ✓ Good — cleaner UX |
| Specialist modes vs separate specs | Watcher specializations inside watcher-ant.md, not 4 separate files | ✓ Good — v3.0 |
| JSON state for infrastructure | errors.json, memory.json, events.json as state files | ✓ Good — v3.0 |
| Enrich existing commands vs add new | Fold review/errors/memory into status/continue rather than new commands | ✓ Good — v3.0 |
| Hybrid prompt+code | Prompts for reasoning, shell scripts for deterministic math/validation | — Pending |
| Single wrapper script | aether-utils.sh with subcommands vs separate scripts | — Pending |
| Pheromone-based communication | Stigmergic signals enable true emergence | ✓ Good — 4 signal types with decay working |
| Standalone system | Aether is its own framework, zero dependencies | ✓ Confirmed |

## Current Milestone: v4.0 Hybrid Foundation

**Goal:** Add a thin shell utility layer for deterministic operations and fix all audit-identified issues. The system becomes hybrid: prompts reason and decide, code computes and validates. This makes pheromone math, state validation, memory management, and error tracking reliable.

**Target deliverables:**
- `aether-utils.sh` — Single wrapper script with subcommands for all deterministic operations
- Pheromone math: `aether-utils pheromone-decay <strength> <elapsed_seconds>`, `aether-utils pheromone-effective <sensitivity> <strength>`
- State validation: `aether-utils validate-state <file>` — Schema enforcement for all JSON state files
- Memory ops: `aether-utils memory-compress`, `aether-utils memory-token-count` — Deterministic memory management
- Error tracking: `aether-utils error-pattern-check`, `aether-utils error-add` — Pattern detection and deduplication
- All 8 audit issues fixed (3 critical + 5 high priority)
- Command prompts updated to call utilities where deterministic results are needed

**Design principle:** Prompts call utilities via Bash tool. Utilities output JSON that prompts can reason about. The boundary is clear: if it needs judgment, the prompt handles it. If it needs math or validation, the script handles it.

---

*Last updated: 2026-02-03 after v4.0 initialization (hybrid prompt+code approach)*
