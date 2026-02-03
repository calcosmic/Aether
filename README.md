# AETHER v3

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
Builder Ant  →  "Need auth library docs"   →  reads scout spec  →  spawns Scout
Colonizer    →  "Complex business logic"   →  reads architect spec → spawns Architect
Scout        →  "Need codebase structure"  →  reads colonizer spec → spawns Colonizer
```

Each caste spec includes pheromone sensitivity tables, spawning instructions, and worked examples. Spawned ants inherit the full spec chain and can spawn further ants recursively (max depth 3).

**Spawning intelligence is guided by:**
- **Pheromone sensitivity** — different castes respond differently to the same signal
- **Bayesian confidence** — spawn outcomes tracked per caste (`alpha/beta`), low-confidence castes trigger alternative consideration
- **Capability gap detection** — ants identify what they can't do and pick the right specialist

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
| **Architect** | Synthesizes knowledge, extracts patterns |

**Each can spawn others** based on local needs.

### 3. Pheromone Communication

| Signal | Purpose | Duration | Strength |
|--------|---------|----------|----------|
| **INIT** | Set colony goal | Persists | 1.0 |
| **FOCUS** | Guide attention | 1 hour | 0.7 |
| **REDIRECT** | Warn away from approach | 24 hours | 0.9 |
| **FEEDBACK** | Teach preferences | 6 hours | 0.5 |

**Signals, not commands.** Pheromones decay exponentially (`strength * e^(-0.693 * elapsed / half_life)`). Each caste has different sensitivity values, so the same signal produces different effective strengths per caste. Ants compute `effective_signal = sensitivity * current_strength` and act based on thresholds.

FEEDBACK and REDIRECT pheromones are also **auto-emitted** at phase boundaries — summarizing what worked/didn't and flagging recurring error patterns.

### 4. Phased Autonomy

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

### Problem: Orchestrator Bottleneck

Central orchestrator becomes bottleneck and single point of failure.

**Aether's Solution**: Stigmergic communication. Pheromones = environment = distributed coordination.

---

## Current Status

**v3**: Rebuilt from first principles. Stripped ~1.3MB of dead code from v2, rewrote all commands as clean Claude Code slash-command prompts. No Python, no bash scripts — the entire system is markdown prompts and JSON state.

**What's Working:**
- 4 pheromone types (INIT, FOCUS, REDIRECT, FEEDBACK) with exponential decay math
- 6 worker castes with per-caste sensitivity tables, combination effects, and feedback interpretation
- Pure emergence: `/ant:build` spawns one ant that self-organizes the entire phase
- Recursive spawning with full spec chain propagation (depth 3, max 5 sub-ants)
- Bayesian spawn confidence tracking — alpha/beta updated per caste on phase outcomes
- Mandatory watcher verification after every build (quality score, recommendation, issue severity)
- Auto-emitted pheromones at phase boundaries (FEEDBACK always, REDIRECT on flagged patterns)
- Git checkpoints before phase execution for rollback capability
- Worker state tracking (active/idle) across all commands
- Environment-aware planning (detects project type, injects tool constraints)
- Event logging, error tracking with pattern flagging (3+ occurrences)
- Colony memory (phase learnings, decisions) persisted across sessions
- Colonization findings persisted to memory for use by planner and builders
- 13 commands, 6 worker specs, pure JSON state management

---

## Usage

### Quick Start

```bash
/ant:init "Build a REST API with PostgreSQL"
/ant:plan
/ant:build 1
```

### All Commands

| Command | Purpose |
|---------|---------|
| `/ant:init "<goal>"` | Set colony intention and initialize |
| `/ant:colonize` | Analyze existing codebase |
| `/ant:plan` | Generate project plan (colony self-organizes) |
| `/ant:build <N>` | Execute phase N (one ant spawned, self-organizes) |
| `/ant:focus "<area>"` | Guide attention (0.7 strength, 1hr decay) |
| `/ant:redirect "<pat>"` | Warn away from pattern (0.9, 24hr decay) |
| `/ant:feedback "<msg>"` | Adjust behavior (0.5, 6hr decay) |
| `/ant:status` | Colony status, pheromones, progress |
| `/ant:phase [N\|list]` | View phase details |
| `/ant:continue` | Approve phase, advance to next |
| `/ant:pause-colony` | Save state for session break |
| `/ant:resume-colony` | Restore from pause |

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
│   ├── COLONY_STATE.json    # Colony state, workers, spawn outcomes
│   ├── pheromones.json      # Decaying pheromone signals
│   ├── PROJECT_PLAN.json    # Phase plan with tasks and success criteria
│   ├── errors.json          # Error log + flagged patterns
│   ├── events.json          # Event log (capped at 100)
│   └── memory.json          # Phase learnings + decisions
├── utils/
│   └── atomic-write.sh      # Corruption-safe writes
├── workers/
│   ├── colonizer-ant.md     # Codebase exploration spec
│   ├── route-setter-ant.md  # Phase planning spec
│   ├── builder-ant.md       # Code implementation spec
│   ├── watcher-ant.md       # Validation/testing spec (4 specialist modes)
│   ├── scout-ant.md         # Research/information spec
│   └── architect-ant.md     # Knowledge synthesis spec
└── HANDOFF.md               # Session handoff (for pause/resume)
.claude/commands/ant/
    ├── ant.md               # Help overview
    ├── init.md              # Initialize colony + create state files
    ├── colonize.md          # Analyze codebase, persist findings
    ├── plan.md              # Generate plan (environment-aware)
    ├── build.md             # Execute phase (git checkpoint, watcher verification)
    ├── continue.md          # Advance phase (auto-emit pheromones)
    ├── focus.md             # Emit FOCUS signal
    ├── redirect.md          # Emit REDIRECT signal
    ├── feedback.md          # Emit FEEDBACK signal
    ├── status.md            # Colony status dashboard
    ├── phase.md             # Phase details
    ├── pause-colony.md      # Save session state
    └── resume-colony.md     # Restore session state
```

---

**MIT License**

*"The whole is greater than the sum of its parts."* — Aristotle 🐜
