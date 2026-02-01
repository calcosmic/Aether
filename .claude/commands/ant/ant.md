---
name: ant
description: Autonomous agent spawning system - Like ant colonies, agents spawn agents without central control
---

<objective>
Execute AETHER system - the FIRST system where AI agents autonomously spawn other agents.

Shows current system status and routes to appropriate subcommands.
</objective>

<reference>
# AETHER Command Reference

**AETHER** (Autonomous Engine for Transformative Human-Enhanced Reasoning) is the first AI system where agents autonomously spawn other agents without human orchestration.

Inspired by ant colonies, AETHER agents:
- Detect capability gaps autonomously
- Spawn specialist agents as needed
- Share memory across the ecosystem
- Learn from every mistake
- Self-organize without central control

**No existing system does this:**
- AutoGen: Humans define all agents
- LangGraph: Predefined workflows only
- CDS: Human-orchestrated specialists
- **AETHER: Agents spawn agents AUTONOMOUSLY** ⭐

## Quick Start

```
/ant                    # Show system overview and status
/ant:build <goal>     # Execute a goal (e.g., "Build a blog with comments")
/ant:status            # Show agent hierarchy and system stats
/ant:memory            # Show memory state (working/short-term/long-term)
/ant:errors            # Show error ledger and flagged issues
```

## Key Innovation

Like ant colonies, AETHER demonstrates **emergent intelligence**:

- **No queen bee** - No central orchestrator
- **Autonomous spawning** - Agents decide when to spawn help
- **Self-organizing** - Teams form without direction
- **Collective intelligence** - System smarter than individuals

## System Components

| Component | Purpose |
|-----------|---------|
| **Agent Spawning** | Agents spawn specialists autonomously when they detect capability gaps |
| **Triple-Layer Memory** | Working (200k), Short-term (10 sessions), Long-term (persistent) |
| **Error Prevention** | Never make the same mistake twice - auto-flags after 3 occurrences |
| **Semantic Communication** | SAP protocol - intent-based messaging (10-100x bandwidth reduction) |
| **Goal Decomposition** | Autonomous planning and task breakdown |

## Usage Examples

**Execute a goal:**

```
/ant:build "Create a REST API with user authentication"
```

AETHER will:
1. Decompose goal into subtasks
2. Spawn specialist agents for each subtask
3. Coordinate execution autonomously
4. Learn from any mistakes

**Check system status:**

```
/ant:status
```

Shows:
- Active agents and hierarchy
- Memory utilization
- Error prevention stats
- Recent goals executed

## Architecture

```
                    ┌─────────────────┐
                    │   AETHER System   │
                    └────────┬────────┘
                             │
              ┌────────────┼────────────┐
              │            │            │
         ┌──▼──┐     ┌───▼──┐     ┌───▼─────┐
         │Goal │     │Agent │     │  Memory │
         │Agent│     │Spawn │     │System  │
         └─────┘     └───────┘     └─────────┘
                           │
                    ┌──────▼───────┐
                    │ Autonomous  │
                    │ Agent Tree  │
                    │ (emerges)   │
                    └─────────────┘
```

## Research Foundation

Built on **100,000+ words** of research across 12 documents:
- Phase 1: Context Engine Foundation ✅
- Phase 2: Multi-Agent Orchestration ✅
- Phase 3: Semantic Codebase Understanding 🐜 (in progress)
- Phases 4-7: Planned (Ralph researching autonomously)

## Getting Started

1. **See system status:**
   ```
   /ant
   ```

2. **Execute your first goal:**
   ```
   /ant:build "Build a TODO app with user auth"
   ```

3. **Watch agents spawn:**
   Real-time hierarchy visualization as agents autonomously create specialists

4. **Check what was learned:**
   ```
   /ant:memory
   ```

## See Also

- `/ant:build` - Execute a goal with autonomous agent spawning
- `/ant:status` - Show detailed system status
- `/ant:memory` - View memory system state
- `/ant:errors` - View error ledger and flagged issues

## Motivation

In ant colonies, no single ant directs the colony. Each ant:
- Acts autonomously based on local cues
- Spawns work when needed (pheromones trigger activity)
- Forms collective intelligence without central control
- Adapts to changing conditions

**AETHER brings this to AI development:**

```
Traditional: Human → Orchestrator → Agents
AETHER:    Goal → Agent → Detects Gap → Spawns → Coordinates → Complete
```

This is **paradigm-shifting** technology. 🐜
</reference>
