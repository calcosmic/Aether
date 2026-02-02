# AETHER v2

<div align="center">
  <img src="aether-logo.png" alt="Aether Logo" width="600">
</div>

> **"The whole is greater than the sum of its parts."** — Aristotle 🐜

---

## What Is Aether?

**Aether is a multi-agent system where Worker Ants autonomously spawn other Worker Ants.**

No human orchestration. No predefined workflows. Pure emergence.

```
Traditional Systems:        Aether:
┌─────────┐               ┌─────────┐
│ Human   │               │  Queen  │
└────┬────┘               └────┬────┘
     │                         │
     v                         v
┌─────────┐               ┌─────────┐
│Orchestr.│  (NOT Aether) │ Signals │
└────┬────┘               └────┬────┘
     │                         │
     v                         v
┌─────────────────┐       ┌─────────────────┐
│ Predefined Agent│       │Self-Organizing │
│   Workers       │       │    Colony       │
└─────────────────┘       └────────┬────────┘
                                    │
                                    v
                          Workers spawn Workers
```

**Why This Matters:**

Every AI system requires humans to anticipate every capability before execution begins. Aether doesn't.

When a Worker Ant encounters a capability gap, it spawns a specialist. The colony adapts to the problem.

---

## The Core Innovation

### Autonomous Agent Spawning

```
Colonizer Ant  →  "Need security analysis"  →  spawns Security Scout
Route-setter   →  "Need database schema"    →  spawns Database Architect
Builder        →  "Need API tests"          →  spawns Test Generator
```

**No existing system does this.**

- **AutoGen**: Humans define all agents
- **LangGraph**: Predefined DAGs
- **CrewAI**: Human-designed teams
- **Aether**: Colony spawns itself

---

## How It Works

### 1. Queen Provides Intention (Not Commands)

```
/ant:init "Build a REST API with authentication"
```

Queen emits **pheromone signals**. Colony self-organizes.

### 2. Six Worker Ant Castes

| Caste | Role |
|-------|------|
| **Colonizer** | Explores codebase, builds semantic index |
| **Route-setter** | Plans phases, breaks down tasks |
| **Builder** | Implements code, runs commands |
| **Watcher** | Validates, tests, quality checks |
| **Scout** | Researches, finds information |
| **Architect** | Compresses memory, extracts patterns |

**Each can spawn others** based on local needs.

### 3. Pheromone Communication

| Signal | Purpose | Duration |
|--------|---------|----------|
| **INIT** | Set colony goal | Persists |
| **FOCUS** | Guide attention | 1 hour |
| **REDIRECT** | Warn away from approach | 24 hours |
| **FEEDBACK** | Teach preferences | 6 hours |

**Signals, not commands.** Colony responds to combination.

### 4. Triple-Layer Memory

```
┌──────────────────────────────────────────────────┐
│  WORKING MEMORY   │  SHORT-TERM    │  LONG-TERM  │
│  (200k tokens)    │  (10 sessions) │  (patterns) │
│  Immediate        │  Compressed    │  Persistent │
└──────────────────────────────────────────────────┘
         │                   │                  │
         └──── 2.5x DAST compression ──────────────┘
```

Human cognition mirrored:
- **Working**: Current task context
- **Short-term**: Recent sessions (2.5x compressed)
- **Long-term**: Persistent patterns, learned expertise

**Cross-layer search** returns ranked results from all layers.

### 5. Phased Autonomy

```
Structure ──────────────┐  Phase Boundary  ────────────┐
at boundaries            │                      │
                         ▼                      ▼
    ┌─────────────────────────────────────────────┐
    │  Emergence Within Phase                    │
    │  Workers spawn Workers                      │
    │  Colony self-organizes                      │
    │  No human intervention                      │
    └─────────────────────────────────────────────┘
```

**Structure at boundaries, emergence within.**

---

## Why It's Revolutionary

### Problem: Unforeseen Requirements

Traditional systems fail when:
- "We need security audit" (but no security agent defined)
- "Database requires migration" (but no migration specialist)
- "API needs rate limiting" (but no infrastructure expert)

**Aether's Solution**: Workers spawn Workers.

### Problem: Context Rot

LLMs forget everything between sessions.

**Aether's Solution**: Triple-layer memory with automatic compression. Patterns persist across sessions.

### Problem: Orchestrator Bottleneck

Central orchestrator becomes bottleneck and single point of failure.

**Aether's Solution**: Stigmergic communication. Pheromones = environment = distributed coordination.

---

## Current Progress

**v1 Milestone**: ✅ SHIPPED (2026-02-02)

All 52 requirements satisfied. 8 phases (3-10), 156/156 must-haves verified.

| Phase | Status |
|-------|--------|
| 1. Colony Foundation | ✅ Complete |
| 2. Worker Ant Castes | ✅ Complete |
| 3. Pheromone Communication | ✅ Complete |
| 4. Triple-Layer Memory | ✅ Complete |
| 5. Phase Boundaries | ✅ Complete |
| 6. Autonomous Emergence | ✅ Complete |
| 7. Colony Verification | ✅ Complete |
| 8. Colony Learning | ✅ Complete |
| 9. Stigmergic Events | ✅ Complete |
| 10. Colony Maturity | ✅ Complete |

**What's Working:**
- ✅ Autonomous spawning with Bayesian meta-learning
- ✅ Pheromone signals (INIT, FOCUS, REDIRECT, FEEDBACK) with time-based decay
- ✅ Triple-layer memory (Working → Short-term DAST → Long-term patterns)
- ✅ Multi-perspective verification (4 watchers, weighted voting, Critical veto)
- ✅ Event-driven coordination (pub/sub event bus, async delivery)
- ✅ State machine (7 states, checkpoints, recovery)
- ✅ 19 commands, 10 Worker Ants, 26 utility scripts
- ✅ Comprehensive testing (41+ assertions, stress tests, performance baselines)

---

## Usage

### Initialize Colony

```bash
/ant:init "Build a REST API with PostgreSQL"
```

### Colony Commands

```
/ant:status          # Show colony state
/ant:phase 1         # Show phase details
/ant:focus "auth"    # Guide attention to area
/ant:memory          # Search triple-layer memory
```

### Memory System

```bash
/ant:memory search "database"      # Search all layers
/ant:memory status                 # Show memory statistics
/ant:memory verify                 # Check 200k token limit
```

---

## Architecture Sketch

```
                    ┌──────────────────┐
                    │   QUEEN SIGNAL   │
                    │  (Intention)     │
                    └────────┬─────────┘
                             │
                    ┌────────▼─────────┐
                    │  PHEROMONE LAYER │
                    │  Init•Focus•Red  │
                    └────────┬─────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
┌───────▼──────┐    ┌───────▼──────┐    ┌───────▼──────┐
│  WORKING     │    │  SHORT-TERM  │    │  LONG-TERM   │
│  200k tokens │    │  10 sessions │    │  Patterns    │
└──────────────┘    └──────────────┘    └──────────────┘
        │                    │                    │
        └────────────────────┼────────────────────┘
                             │
                    ┌────────▼─────────┐
                    │   WORKER ANTS    │
                    │  (6 Castes)      │
                    │  Spawn each other│
                    └──────────────────┘
```

---

## The Aether Difference

| Aspect | Traditional | Aether |
|--------|------------|---------|
| **Control** | Human orchestrator | Queen signals, colony self-organizes |
| **Communication** | Direct commands | Pheromone signals (stigmergy) |
| **Planning** | Human-defined workflows | Queen sets intention, colony creates structure |
| **Execution** | Sequential task lists | Emergent execution within phases |
| **Intelligence** | Individual agent smarts | Colony intelligence (distributed) |

---

## Why Ants?

Ant colonies demonstrate **superlinear intelligence**:

- Single ant: ~250 neurons (can barely navigate)
- Colony of 1M ants: farms, builds, wages war
- **No central brain** — the colony IS the intelligence

**Key insight**: Intelligence scales with autonomous agent creation, not smarter individuals.

Aether translates this to AI:
- Queen = intention (not control)
- Pheromones = signals (not commands)
- Workers = autonomous specialists (spawn each other)
- Colony = the intelligence (distributed)

---

## File Structure

```
.aether/
├── data/
│   ├── COLONY_STATE.json    # Colony state
│   ├── pheromones.json      # Signal layer
│   └── memory.json          # Triple-layer memory
├── utils/
│   ├── memory-ops.sh        # Working Memory operations
│   ├── memory-compress.sh   # DAST compression
│   ├── memory-search.sh     # Cross-layer search
│   ├── atomic-write.sh      # Corruption-safe writes
│   └── file-lock.sh         # Concurrent access prevention
├── workers/
│   ├── colonizer-ant.md     # Codebase exploration
│   ├── route-setter-ant.md  # Phase planning
│   ├── builder-ant.md       # Code implementation
│   ├── watcher-ant.md       # Validation/testing
│   ├── scout-ant.md         # Research/information
│   └── architect-ant.md     # Memory compression
└── .claude/commands/ant/
    ├── init.md              # Initialize colony
    ├── focus.md             # Emit FOCUS signal
    ├── redirect.md          # Emit REDIRECT signal
    ├── feedback.md          # Emit FEEDBACK signal
    └── memory.md            # Memory operations
```

---

**MIT License**

*"The whole is greater than the sum of its parts."* — Aristotle 🐜
