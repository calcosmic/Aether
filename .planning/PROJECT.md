# Aether v3: Claude-Native Queen Ant Colony

## What This Is

Aether is a **unique, standalone multi-agent system** built from first principles on ant colony intelligence. Worker Ants autonomously spawn other Worker Ants without human orchestration. The Queen (user) provides high-level intention via pheromone signals (INIT, FOCUS, REDIRECT, FEEDBACK), and the colony self-organizes to complete tasks through emergent intelligence.

This is a **Claude-native system** — commands like `/ant:init "Build a REST API"` work directly in Claude Code. Commands are skill prompts that use Read/Write tools for state manipulation and the Task tool for autonomous agent spawning.

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

*(v3.0 milestone — Restore the Soul)*

- [ ] **Rich Visual Identity** — Box-drawing headers, step progress tracking, emoji animations, pheromone strength bars in all commands
- [ ] **Specialist Watcher Modes** — Security, performance, quality, test-coverage specializations within watcher-ant.md
- [ ] **Deep Worker Specs** — Pheromone calculation examples, combination effects, feedback interpretation, event awareness (~200 lines per spec)
- [ ] **Error Tracking System** — errors.json ledger with root cause analysis, pattern flagging, prevention tracking
- [ ] **Colony Memory** — memory.json for phase learnings, decision history, pattern recognition across sessions
- [ ] **Event Awareness** — events.json log that workers check at startup for colony context
- [ ] **Enhanced Status Dashboard** — /ant:status shows full colony health: workers, pheromones with decay bars, errors, memory, phase progress
- [ ] **Phase Review in Continue** — /ant:continue shows what was built before advancing
- [ ] **Spawn Outcome Tracking** — Track which specialist spawns succeed/fail to improve future spawning decisions

### Out of Scope

- **Python CLI/REPL interfaces** — Replaced by Claude-native prompt commands
- **Bash-based event bus** — v2's 879-line event-bus.sh replaced by simple JSON event log
- **Bash-based memory scripts** — v2's memory-search.sh/memory-compress.sh replaced by JSON state
- **Separate /ant:errors command** — Error display integrated into /ant:status
- **Separate /ant:review command** — Review integrated into /ant:continue
- **Separate /ant:memory command** — Memory state shown in /ant:status
- **Separate /ant:adjust command** — Use /ant:focus, /ant:redirect, /ant:feedback directly
- **Separate /ant:recover command** — Recovery integrated into /ant:resume-colony
- **External vector databases** — Using Claude's native semantic understanding
- **Predefined workflows** — Defeats emergence; use phased autonomy instead

## Context

### Current State (v3-rebuild — 2026-02-03)

**What exists (working):**
- 12 commands as Claude Code skill prompts (init, plan, build, status, phase, continue, focus, redirect, feedback, pause-colony, resume-colony, colonize, ant)
- 6 worker ant specs (~90 lines each) with spawning guides
- 3 state files: COLONY_STATE.json, pheromones.json, PROJECT_PLAN.json
- 2 utility scripts: atomic-write.sh, file-lock.sh
- Clean Read/Write tool flow — no bash/jq in commands

**What was lost in rebuild (to restore in v3.0):**
- Visual identity — box headers, step progress, emoji animations, pheromone bars
- 4 specialist watcher types — security, performance, quality, test-coverage
- Worker spec depth — pheromone math examples, combination effects, event awareness
- Error tracking system — error ledger with root cause analysis
- Memory system — phase learnings, decision history, patterns
- Event awareness — workers knowing what happened since last execution
- Spawn outcome tracking — learning from specialist success/failure

### Background

Aether is based on **383,000+ words of research** across 25 documents by Ralph (research agent), covering:
- Multi-agent orchestration patterns
- Semantic communication protocols (AINP, SACP)
- Context engines and memory architecture
- Autonomous spawning research (confirmed: no existing system has autonomous spawning)
- Verification and quality systems

## Constraints

- **Claude-Native Only** — Commands are skill prompts using Read/Write/Task tools
- **JSON State Persistence** — All state in `.aether/data/*.json` files manipulated via Read/Write tools
- **Task Tool for Spawning** — Autonomous spawning uses Claude's Task tool with full spec injection
- **Standalone Architecture** — Aether is its own system, not dependent on CDS or any framework
- **No External Dependencies** — No vector DBs, no embedding services, no bash in commands
- **No New Commands** — Restore functionality by enriching existing 12 commands, not adding new ones
- **Unique Design** — All architectures uniquely Aether (inspired by research, not copied)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Claude-native vs Python | Commands work directly in Claude, not separate tool | ✓ Good — 12 commands working |
| Read/Write tools vs bash/jq | Claude manipulates JSON directly, more reliable than shell scripts | ✓ Good — v3 rebuild proved this works |
| Phase Lead emergence | One ant spawned per phase, self-organizes | ✓ Good — true emergence achieved |
| Consolidate commands (19 → 12) | Fewer, richer commands over many thin ones | ✓ Good — cleaner UX |
| Specialist modes vs separate specs | Watcher specializations inside watcher-ant.md, not 4 separate files | — Pending |
| JSON state for infrastructure | errors.json, memory.json, events.json vs bash utility scripts | — Pending |
| Enrich existing commands vs add new | Fold review/errors/memory into status/continue rather than new commands | — Pending |
| Pheromone-based communication | Stigmergic signals enable true emergence | ✓ Good — 4 signal types with decay working |
| Standalone system | Aether is its own framework, zero dependencies | ✓ Confirmed |

## Current Milestone: v3.0 Restore the Soul

**Goal:** Bring back the sophistication, visual identity, and depth that made Aether the most advanced self-spawning agent system — rebuilt natively for the Claude Code skill prompt architecture.

**Target features:**
- Rich visual output in every command (box headers, step progress, emoji status, pheromone bars)
- Specialist watcher modes (security, performance, quality, test-coverage) in watcher-ant.md
- Deep worker specs with pheromone math, combination effects, event awareness (~200 lines each)
- Error tracking system (errors.json with root cause analysis and pattern flagging)
- Colony memory (memory.json for phase learnings and decision history)
- Event awareness (events.json log workers check at startup)
- Enhanced /ant:status dashboard with full colony health
- Phase review integrated into /ant:continue
- Spawn outcome tracking for meta-learning

---

*Last updated: 2026-02-03 after v3.0 milestone initialization*
