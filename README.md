# AETHER v4.1

<div align="center">
  <img src="aether-logo.png" alt="Aether Logo" width="600">
</div>

> **"The whole is greater than the sum of its parts."** — Aristotle

---

## What Is Aether?

**Aether is a multi-agent system that applies ant colony intelligence to autonomous agent orchestration, built natively for Claude Code.**

Worker Ants spawn other Worker Ants through bio-inspired pheromone signaling, caste specialization, and Bayesian spawn tracking. The Queen (you) provides intention via pheromone signals. The colony self-organizes.

```
┌─────────┐
│  Queen   │  (you — provides intention, not commands)
└────┬────┘
     │
     v
┌─────────┐
│ Signals │  (pheromones: INIT, FOCUS, REDIRECT, FEEDBACK)
└────┬────┘
     │
     v
┌──────────────────┐
│  Self-Organizing  │
│     Colony        │
└────────┬─────────┘
         │
         v
Workers spawn Workers  (max depth 3, max 5 active)
```

When a Worker Ant encounters a capability gap, it spawns a specialist. The colony adapts to the problem.

---

## What Makes It Different

Autonomous agent spawning is not new — systems like AutoGen (ADAS/Meta Agent Search), AutoAgents, and OpenAI's Agents SDK all support dynamic agent creation. What Aether does differently is the **coordination model**:

- **Stigmergic communication** — pheromone signals with exponential decay, not direct commands or message passing
- **Caste-based sensitivity** — the same signal produces different effective strengths per worker type
- **Bayesian spawn confidence** — spawn outcomes tracked per caste with alpha/beta updates, so the colony learns which specialists succeed
- **Phased autonomy** — structure at boundaries (Queen check-ins), pure emergence within phases
- **Claude Code native** — the entire system is markdown skill prompts + a thin shell utility layer, not a Python/Node framework

This is a novel *implementation approach* to multi-agent coordination, not a novel concept. The ant colony metaphor provides a different set of affordances than traditional orchestration patterns.

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

**Signals, not commands.** Pheromones decay exponentially. Each caste has different sensitivity values, so the same signal produces different effective strengths per caste. Ants compute `effective_signal = sensitivity * current_strength` and act based on thresholds.

FEEDBACK and REDIRECT pheromones are also **auto-emitted** at phase boundaries — summarizing what worked/didn't and flagging recurring error patterns. Auto-emitted pheromones are validated by shell utility (minimum 20 chars, non-empty) before being written.

### 4. Phased Autonomy

```
Phase Boundary ─────────────────── Phase Boundary
       │                                  │
       ▼                                  ▼
┌──────────────────────────────────────────┐
│  Emergence Within Phase                  │
│  Workers spawn Workers                   │
│  Colony self-organizes                   │
│  No human intervention                   │
└──────────────────────────────────────────┘
```

**Structure at boundaries, emergence within.**

### 5. Hybrid Architecture

Prompts handle reasoning and orchestration. A thin shell utility layer (`aether-utils.sh`, 229 lines, 13 subcommands) handles deterministic operations that LLMs get wrong: pheromone decay math, state validation, spawn limit enforcement, memory compression, error tracking.

---

## Current Status

**v4.1** — Cleanup & Enforcement (2026-02-03)

**What's built:**
- 12 commands as Claude Code skill prompts
- 6 worker ant specs with pheromone math, spawning scenarios, enforcement gates
- `aether-utils.sh` — 229-line utility wrapper with 13 subcommands
- 6 JSON state files with atomic writes and file locking
- Spawn limit enforcement (max 5 workers, max depth 3) via shell validation gates
- Pheromone quality enforcement via shell validation before writes
- Post-action validation checklists in all worker specs
- Bayesian spawn confidence tracking per caste
- Auto-emitted pheromones at phase boundaries
- Git checkpoints before phase execution
- Event logging, error tracking with pattern flagging

**What's not proven:**
- The system has not been run end-to-end on a real project. Individual components (utility subcommands, state management, command structure) are tested and working. But no colony has actually self-organized — no `/ant:init` with a real goal, no `/ant:build` spawning live workers, no pheromone-guided emergence observed in practice.
- LLM compliance with enforcement gates (spawn-check, pheromone-validate, post-action validation) is specified in prompt text but depends on whether Claude actually follows those instructions at runtime.

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

## Why Ants?

Ant colonies demonstrate **superlinear intelligence**:

- Single ant: ~250 neurons (can barely navigate)
- Colony of 1M ants: farms, builds, wages war
- **No central brain** — the colony IS the intelligence

Aether translates this to AI agents:
- Queen = intention (not control)
- Pheromones = signals (not commands)
- Workers = autonomous specialists (spawn each other)
- Colony = the intelligence (distributed)

---

## File Structure

```
.aether/
├── aether-utils.sh            # 229-line utility wrapper (13 subcommands)
├── data/
│   ├── COLONY_STATE.json      # Colony state, workers, spawn outcomes
│   ├── pheromones.json        # Decaying pheromone signals
│   ├── PROJECT_PLAN.json      # Phase plan with tasks and success criteria
│   ├── errors.json            # Error log + flagged patterns
│   ├── events.json            # Event log (capped at 100)
│   └── memory.json            # Phase learnings + decisions
├── utils/
│   ├── atomic-write.sh        # Corruption-safe writes
│   └── file-lock.sh           # File locking for concurrent access
├── workers/
│   ├── colonizer-ant.md       # Codebase exploration spec
│   ├── route-setter-ant.md    # Phase planning spec
│   ├── builder-ant.md         # Code implementation spec
│   ├── watcher-ant.md         # Validation/testing spec (4 specialist modes)
│   ├── scout-ant.md           # Research/information spec
│   └── architect-ant.md       # Knowledge synthesis spec
└── HANDOFF.md                 # Session handoff (for pause/resume)
.claude/commands/ant/
    ├── ant.md                 # Help overview
    ├── init.md                # Initialize colony + create state files
    ├── colonize.md            # Analyze codebase, persist findings
    ├── plan.md                # Generate plan (environment-aware)
    ├── build.md               # Execute phase (git checkpoint, watcher verification)
    ├── continue.md            # Advance phase (auto-emit pheromones)
    ├── focus.md               # Emit FOCUS signal
    ├── redirect.md            # Emit REDIRECT signal
    ├── feedback.md            # Emit FEEDBACK signal
    ├── status.md              # Colony status dashboard
    ├── phase.md               # Phase details
    ├── pause-colony.md        # Save session state
    └── resume-colony.md       # Restore session state
```

---

**MIT License**

*"The whole is greater than the sum of its parts."* — Aristotle
