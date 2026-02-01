# Architecture Patterns

**Domain:** Claude-native multi-agent systems
**Researched:** 2026-02-01
**Overall confidence:** HIGH

## Executive Summary

Claude-native multi-agent systems represent a paradigm shift from traditional Python-based frameworks. Instead of code-based orchestration, these systems use **prompts as code** and **JSON as state**. The architecture is fundamentally text-based: command files (`.md`) define agent behaviors through prompts, while state files (`.json`) persist system state between sessions.

**Key insight:** Unlike Python systems (AutoGen, LangGraph, CrewAI) that require runtime execution, Claude-native systems are declarative. The "code" is prompt files that Claude interprets directly. State is persisted as JSON, enabling checkpoint/resume workflows.

The research reveals five critical architectural layers:
1. **Command Layer** - Prompt files define agent behaviors
2. **State Layer** - JSON files persist system state
3. **Memory Layer** - Working → Short-term → Long-term hierarchy
4. **Communication Layer** - Pheromone signals, event bus
5. **Orchestration Layer** - State machine, phase execution
6. **Spawning Layer** - Task tool for autonomous agent creation

## Recommended Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    CLAUDE-NATIVE ARCHITECTURE                    │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  COMMAND LAYER (.claude/commands/ant/*.md)              │  │
│  │  Prompt files = Agent behaviors                          │  │
│  │  - init.md, plan.md, execute.md, etc.                    │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          ↓                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  STATE LAYER (.aether/data/*.json)                      │  │
│  │  JSON files = System persistence                         │  │
│  │  - COLONY_STATE.json, pheromones.json, memory.json       │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          ↓                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  MEMORY LAYER (Triple-layer hierarchy)                  │  │
│  │  Working (200k) → Short-term → Long-term                 │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          ↓                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  COMMUNICATION LAYER                                     │  │
│  │  Pheromone signals (INIT, FOCUS, REDIRECT, FEEDBACK)     │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          ↓                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  ORCHESTRATION LAYER                                     │  │
│  │  State machine: IDLE → INIT → PLANNING → EXECUTING       │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          ↓                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  SPAWNING LAYER                                          │  │
│  │  Task tool spawns autonomous subagents                   │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Component Boundaries

| Component | Responsibility | Communicates With | Data Format |
|-----------|---------------|-------------------|-------------|
| **Command Layer** | Defines agent behaviors via prompts | State Layer (reads/writes JSON) | Markdown prompts |
| **State Layer** | Persists system state | All layers (read/write access) | JSON |
| **Memory Layer** | Manages context compression/retrieval | Orchestration, Spawning | JSON with metadata |
| **Communication Layer** | Coordinates agents via signals | All layers (broadcasts signals) | JSON signal objects |
| **Orchestration Layer** | Manages phase state machine | State, Memory, Spawning | JSON state transitions |
| **Spawning Layer** | Creates specialist agents autonomously | Task tool (Claude native) | Prompt inheritance |

### Data Flow

**1. Initialization Flow**
```
User invokes /ant:init "<goal>"
    ↓
Command Layer: init.md prompt executed
    ↓
Orchestration Layer: Sets state to INIT
    ↓
Communication Layer: Emits INIT pheromone
    ↓
Spawning Layer: Spawns Planner agent via Task tool
    ↓
State Layer: Saves phase plan to COLONY_STATE.json
```

**2. Execution Flow**
```
User invokes /ant:execute <phase_id>
    ↓
Command Layer: execute.md prompt executed
    ↓
State Layer: Loads phase from COLONY_STATE.json
    ↓
Orchestration Layer: Sets phase to IN_PROGRESS
    ↓
Communication Layer: Emits INIT pheromone for phase
    ↓
Spawning Layer: Coordinator agent spawned
    ↓
Coordinator spawns specialist agents via Task tool
    ↓
Agents work autonomously, update state JSON files
    ↓
Phase completion triggers state transition
```

**3. Memory Flow**
```
Agent generates content
    ↓
Stored in Working Memory (in-context)
    ↓
Phase boundary triggers compression
    ↓
DAST compression → Short-term Memory (2.5x reduction)
    ↓
Pattern extraction → Long-term Memory (persistent)
    ↓
Associative links created between layers
```

## Patterns to Follow

### Pattern 1: Prompt-as-Code Architecture

**What:** Agent behaviors are defined as Markdown prompt files, not Python code.

**When:** Building Claude-native systems where commands execute directly in Claude.

**Example:**
```markdown
---
name: ant:init
description: Initialize new project
---

<objective>
Initialize project by setting intention and creating phase structure.
</objective>

<process>
You are the Queen Ant Colony receiving intention...

[Process steps in prose, not code]
</process>

<context>
@.aether/worker_ants.py
@.aether/phase_engine.py
</context>

<allowed-tools>
Task
Write
Bash
Read
</allowed-tools>
```

**Why:** Enables version-controlled prompt engineering, declarative agent definitions, and Claude-native execution.

### Pattern 2: JSON State Persistence

**What:** All system state persisted as JSON files, not databases or Python objects.

**When:** Systems requiring checkpoint/resume, cross-session continuity.

**Example:**
```json
{
  "goal": "Build REST API",
  "status": "executing",
  "current_phase_id": 2,
  "phases": [...],
  "worker_ants": {...},
  "pheromones": [...],
  "meta_learning": {...}
}
```

**Why:** Enables git-tracked state, diff-able changes, easy inspection/debugging, cross-session continuity.

### Pattern 3: Pheromone Signal Communication

**What:** Agents communicate through environment signals, not direct messages.

**When:** Coordinating multiple agents without central orchestrator.

**Example:**
```json
{
  "signal_type": "FOCUS",
  "content": "WebSocket security",
  "strength": 0.7,
  "half_life_hours": 1.0,
  "created_at": "2026-02-01T14:31:09"
}
```

**Why:** Enables stigmergic coordination, adaptive behavior, emergence without orchestration.

### Pattern 4: Triple-Layer Memory

**What:** Hierarchical memory with automatic compression and associative linking.

**When:** Long-running projects requiring context continuity.

**Example:**
```
Working Memory (200k tokens, in-context)
    ↓ [Phase boundary]
Short-term Memory (10 sessions, 2.5x compressed)
    ↓ [Pattern extraction]
Long-term Memory (unlimited, persistent patterns)
```

**Why:** Mirrors human cognition, enables cross-session learning, prevents context bloat.

### Pattern 5: State Machine Orchestration

**What:** Explicit states with transitions and checkpoints.

**When:** Reliable multi-phase workflows.

**Example:**
```
IDLE → INIT → PLANNING → IN_PROGRESS → AWAITING_REVIEW → COMPLETED
                                      ↓
                                   FAILED
```

**Why:** Predictable behavior, error recovery, checkpoint/resume capability.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Python-Based Orchestration

**What:** Using Python scripts to orchestrate agents instead of prompt commands.

**Why bad:** Defeats Claude-native architecture, requires runtime environment, harder to version control prompts.

**Instead:** Use prompt files in `.claude/commands/` with Claude's Task tool for spawning.

### Anti-Pattern 2: Monolithic State Files

**What:** Single large JSON file storing all state.

**Why bad:** Diff merge conflicts, slow to read/write, hard to inspect.

**Instead:** Separate state by concern:
- `COLONY_STATE.json` (phase orchestration)
- `pheromones.json` (communication signals)
- `memory.json` (memory layer)
- `worker_ants.json` (agent status)

### Anti-Pattern 3: Direct Command Messaging

**What:** Agents sending direct commands to other agents.

**Why bad:** Creates tight coupling, prevents emergence, requires orchestrator.

**Instead:** Use pheromone signals that agents respond to autonomously.

### Anti-Pattern 4: Context Window Bloat

**What:** Accumulating conversation history without compression.

**Why bad:** Exceeds context limits, degrades performance, loses important information in noise.

**Instead:** Implement phase-boundary compression with DAST (Discriminative Abstractive Summarization Technique).

### Anti-Pattern 5: Hardcoded Agent Roles

**What:** Defining all agent types and roles in advance.

**Why bad:** Can't handle unforeseen requirements, defeats autonomous spawning.

**Instead:** Enable capability gap detection and autonomous specialist spawning via Task tool.

## Scalability Considerations

| Concern | At 100 agents | At 10K agents | At 1M agents |
|---------|---------------|---------------|--------------|
| **State file size** | JSON files < 1MB | JSON files ~10-50MB | Need sharding/partitioning |
| **Pheromone evaluation** | Linear scan is fine | Need indexing | Need spatial hashing |
| **Memory compression** | Manual triggers | Scheduled compression | Continuous background |
| **Agent spawning** | Task tool unlimited | Need spawning budgets | Need hierarchical spawning |
| **Signal propagation** | Broadcast all | Interest-based pub/sub | Geographic partitioning |

## Prompt Organization Patterns

### Directory Structure

```
.claude/
└── commands/
    └── ant/                    # Namespace for Aether commands
        ├── init.md             # Project initialization
        ├── plan.md             # Display phase plan
        ├── phase.md            # Phase details
        ├── execute.md          # Phase execution
        ├── review.md           # Phase review
        ├── focus.md            # Emit focus pheromone
        ├── redirect.md         # Emit redirect pheromone
        ├── feedback.md         # Emit feedback pheromone
        ├── status.md           # Colony status
        ├── memory.md           # Memory status
        ├── colonize.md         # Codebase analysis
        ├── pause-colony.md     # Session pause
        └── resume-colony.md    # Session resume
```

### Prompt Template Structure

Every command prompt should follow this structure:

```markdown
---
name: ant:command-name
description: One-line description
---

<objective>
What this command accomplishes
</objective>

<process>
Step-by-step process in prose:
1. Step one
2. Step two
...
</process>

<context>
@related-files
References to related components
</context>

<reference>
# Detailed Reference

Additional context, examples, patterns
</reference>

<allowed-tools>
Tool1
Tool2
Tool3
</allowed-tools>
```

**Why this structure:**
- **Frontmatter** enables command discovery and naming
- **Objective** clarifies intent for Claude
- **Process** provides step-by-step guidance
- **Context** links related files for reference
- **Reference** provides detailed documentation
- **Allowed-tools** defines permissions

## Build Order Implications

Based on component dependencies, recommended build order:

### Phase 1: Foundation (Core Infrastructure)
**Components:** State Layer, Command Layer structure
**Why:** Everything depends on JSON state persistence and prompt command structure.

**Deliverables:**
- `.aether/data/COLONY_STATE.json` schema
- `.claude/commands/ant/` directory structure
- Basic state read/write patterns

### Phase 2: Memory System
**Components:** Memory Layer (Working → Short-term → Long-term)
**Why:** Orchestration needs memory to persist phase context.

**Deliverables:**
- `.aether/data/memory.json` with triple-layer structure
- DAST compression algorithm (in prompt logic)
- Associative linking between layers

### Phase 3: Communication System
**Components:** Communication Layer (pheromone signals)
**Why:** Coordination requires signal system before multi-agent execution.

**Deliverables:**
- `.aether/data/pheromones.json`
- Signal emission/decay logic
- Signal evaluation patterns

### Phase 4: Orchestration Engine
**Components:** Orchestration Layer (state machine)
**Why:** Manages phase transitions and agent lifecycle.

**Deliverables:**
- Phase state machine (in prompt logic)
- Checkpoint/resume capability
- Phase transition guards

### Phase 5: Autonomous Spawning
**Components:** Spawning Layer (Task tool integration)
**Why:** Highest value feature, requires all previous layers.

**Deliverables:**
- Capability gap detection (in prompt logic)
- Autonomous specialist spawning via Task tool
- Spawning budgets and governance

### Phase 6: Advanced Features
**Components:** Meta-learning, verification, optimization
**Why:** Enhancements after core system works.

**Deliverables:**
- Bayesian confidence scoring
- Multi-verifier voting
- Performance optimization

## Architectural Tradeoffs

| Decision | Option A (Recommended) | Option B | Why A |
|----------|------------------------|----------|-------|
| **State storage** | JSON files | Database | JSON is git-tracked, diff-able, human-readable |
| **Agent spawning** | Task tool | Python subprocess | Task tool is Claude-native, no runtime dependency |
| **Memory** | Triple-layer hierarchy | Single-layer cache | Mirrors human cognition, proven compression |
| **Communication** | Pheromone signals | Direct messaging | Enables emergence, prevents bottlenecks |
| **Orchestration** | State machine | Event loop | Predictable, debuggable, checkpointable |

## Comparison with Traditional Multi-Agent Systems

| Aspect | Claude-Native (Aether) | Traditional (AutoGen, LangGraph) |
|--------|------------------------|----------------------------------|
| **Definition** | Prompt files | Python code |
| **Execution** | Claude interprets prompts | Python runtime |
| **State** | JSON files | Python objects/DB |
| **Spawning** | Task tool (Claude native) | Framework APIs |
| **Memory** | Prompt-based compression | Context window only |
| **Communication** | Pheromone signals | Message passing |
| **Orchestration** | State machine in prompts | DAG/workflow engines |
| **Version control** | Git-tracked prompts | Code-based |
| **Debugging** | Read JSON state | Debuggers/logging |
| **Deployment** | Copy `.claude/` directory | Install packages |

## Sources

### HIGH Confidence (Official Documentation)

- [Anthropic: How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system) (June 2025) - Official multi-agent architecture patterns
- [Anthropic: Claude Code Best Practices](https://www.anthropic.com/engineering/claude-code-best-practices) (April 2025) - Slash command architecture, prompt organization
- [Anthropic: When to use multi-agent systems](https://claude.com/blog/building-multi-agent-systems-when-and-how-to-use-them) (January 2026) - Multi-agent definition and implementation patterns

### MEDIUM Confidence (Verified with Official Sources)

- [When to use multi-agent systems](https://www.anthropic.com/engineering/multi-agent-research-system) - Architecture overview with state machine patterns
- [Design Patterns for Agentic AI](https://appstekcorp.com/blog/design-patterns-for-agentic-ai-and-multi-agent-systems/) - State machine orchestration patterns

### LOW Confidence (Community Sources - Flagged for Validation)

- [国外大神逆向了Claude Code](https://zhuanlan.zhihu.com/p/1943399204027373513) (August 2025) - Reverse engineering analysis (needs verification)
- [Claude Code's entire system prompt leaked](https://medium.com/coding-nexus/claude-codes-entire-system-prompt-just-leaked-10d16bb30b87) - Unofficial leak (unverified)

### Current Codebase Analysis

- `/Users/callumcowie/repos/Aether/.claude/commands/ant/*.md` - Existing prompt commands (15 files analyzed)
- `/Users/callumcowie/repos/Aether/.aether/data/*.json` - Existing state files (6 files analyzed)
- `/Users/callumcowie/repos/Aether/.aether/memory/meta_learning_demo.json` - Meta-learning implementation example
- `/Users/callumcowie/repos/Aether/README.md` - System architecture documentation
- `/Users/callumcowie/repos/Aether/.planning/PROJECT.md` - Project context and constraints

### Gap Analysis

**Missing:** Specific documentation on "CDS" (Cosmic Dev System) architecture. Based on context references, CDS appears to be a prompt-based development framework similar to what Aether is building. Unable to locate official CDS documentation - may be internal system or unreleased project.

**Workaround:** Analyzed existing Aether codebase which implements Claude-native patterns, providing concrete examples of the architecture.
