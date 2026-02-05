# 🐜 AETHER v5.0

<div align="center">
  <img src="aether-logo.png" alt="Aether Logo" width="600">
</div>

> **"The whole is greater than the sum of its parts."** — Aristotle

---

## 🐜 What Is Aether?

**Aether is a multi-agent system that applies ant colony intelligence to autonomous agent orchestration, built natively for Claude Code.**

Worker Ants spawn other Worker Ants through bio-inspired pheromone signaling, caste specialization, and Bayesian spawn tracking. The Queen (you) provides intention via pheromone signals. The colony self-organizes.

```
🐜 ┌─────────┐
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
   │   🐜 Colony 🐜   │
   └────────┬─────────┘
            │
            v
   Workers spawn Workers  (max depth 3, max 5 active)
```

When a Worker Ant encounters a capability gap, it spawns a specialist. The colony adapts to the problem.

---

## 📦 Installation

### Via npm (recommended)

```bash
npm install -g aether-colony
```

This installs the `aether` CLI and automatically sets up:
- **Commands** → `~/.claude/commands/ant/` (14 Claude Code skill prompts)
- **Runtime** → `~/.aether/` (worker specs, utility scripts, docs)

### Manual install

```bash
git clone https://github.com/callumcowie/Aether.git
cd Aether
node bin/cli.js install
```

### Verify installation

```bash
aether version          # Shows installed version
ls ~/.claude/commands/ant/  # 14 command files
ls ~/.aether/workers/       # 6 worker specs
```

### Uninstall

```bash
aether uninstall        # Removes global files, preserves learnings
npm uninstall -g aether-colony
```

Per-project `.aether/data/` directories are never touched by uninstall.

---

## 🚀 Quick Start

Open Claude Code in any repo and run:

```bash
/ant:init "Build a REST API with authentication"
/ant:plan
/ant:build 1
```

The colony will self-organize: a Route-setter plans the work, Builders implement it, Watchers validate it, and the Queen orchestrates with live visibility.

---

## 🧬 What Makes It Different

Autonomous agent spawning is not new — systems like AutoGen (ADAS/Meta Agent Search), AutoAgents, and OpenAI's Agents SDK all support dynamic agent creation. What Aether does differently is the **coordination model**:

- 🐜 **Stigmergic communication** — pheromone signals with exponential decay, not direct commands or message passing
- 🎯 **Caste-based sensitivity** — the same signal produces different effective strengths per worker type
- 📊 **Bayesian spawn confidence** — spawn outcomes tracked per caste with alpha/beta updates, so the colony learns which specialists succeed
- 🔄 **Phased autonomy** — structure at boundaries (Queen check-ins), pure emergence within phases
- 🧠 **Auto-learning** — the colony automatically extracts phase learnings and emits feedback pheromones after every build
- ⚡ **Claude Code native** — the entire system is markdown skill prompts + a thin shell utility layer, not a Python/Node framework

This is a novel *implementation approach* to multi-agent coordination, not a novel concept. The ant colony metaphor provides a different set of affordances than traditional orchestration patterns.

---

## 🏗️ How It Works

### 1. 👑 Queen Provides Intention (Not Commands)

```
/ant:init "Build a REST API with authentication"
```

Queen emits **pheromone signals**. Colony self-organizes.

### 2. 🐜 Six Worker Ant Castes

| Caste | Emoji | Role |
|-------|-------|------|
| **Colonizer** | 🔍 | Explores codebase, builds semantic index |
| **Route-setter** | 🗺️ | Plans phases, breaks down tasks |
| **Builder** | 🔨 | Implements code, runs commands |
| **Watcher** | 👁️ | Validates, tests, quality checks |
| **Scout** | 🔎 | Researches, finds information |
| **Architect** | 📐 | Synthesizes knowledge, extracts patterns |

**Each can spawn others** based on local needs.

### 3. 🧪 Pheromone Communication

| Signal | Purpose | Duration | Strength |
|--------|---------|----------|----------|
| 🟢 **INIT** | Set colony goal | Persists | 1.0 |
| 🎯 **FOCUS** | Guide attention | 1 hour | 0.7 |
| 🚫 **REDIRECT** | Warn away from approach | 24 hours | 0.9 |
| 💬 **FEEDBACK** | Teach preferences | 6 hours | 0.5 |

**Signals, not commands.** Pheromones decay exponentially. Each caste has different sensitivity values, so the same signal produces different effective strengths per caste. Ants compute `effective_signal = sensitivity * current_strength` and act based on thresholds.

FEEDBACK and REDIRECT pheromones are also **auto-emitted** at phase boundaries — summarizing what worked/didn't and flagging recurring error patterns. Auto-emitted pheromones are validated by shell utility (minimum 20 chars, non-empty) before being written.

### 4. 🔄 Phased Autonomy

```
Phase Boundary ─────────────────── Phase Boundary
       │                                  │
       ▼                                  ▼
┌──────────────────────────────────────────┐
│  🐜 Emergence Within Phase 🐜           │
│  Workers spawn Workers                   │
│  Colony self-organizes                   │
│  No human intervention                   │
└──────────────────────────────────────────┘
```

**Structure at boundaries, emergence within.**

### 5. 🔧 Hybrid Architecture

Prompts handle reasoning and orchestration. A thin shell utility layer (`aether-utils.sh`, ~370 lines, 18 subcommands) handles deterministic operations that LLMs get wrong: pheromone decay math, state validation, spawn limit enforcement, memory compression, error tracking, activity logging.

### 6. 👁️ Live Visibility

The Queen spawns workers sequentially and displays each worker's activity log output between spawns — you see what each ant did as it completes, not after the entire phase finishes. Workers write structured progress lines to an activity log during execution.

### 7. 🧠 Auto-Learning

After every build, the colony automatically extracts phase learnings from completed work (errors, events, task outcomes) and writes them to colony memory. A FEEDBACK pheromone is auto-emitted summarizing what worked and what failed. No manual `/ant:continue` needed for learning capture.

---

## 🐜 All Commands

| Command | Purpose |
|---------|---------|
| `/ant:init "<goal>"` | 🟢 Set colony intention and initialize |
| `/ant:colonize` | 🔍 Analyze existing codebase |
| `/ant:plan` | 🗺️ Generate project plan (colony self-organizes) |
| `/ant:build <N>` | 🔨 Execute phase N (Queen spawns workers with live visibility) |
| `/ant:focus "<area>"` | 🎯 Guide attention (0.7 strength, 1hr decay) |
| `/ant:redirect "<pat>"` | 🚫 Warn away from pattern (0.9, 24hr decay) |
| `/ant:feedback "<msg>"` | 💬 Adjust behavior (0.5, 6hr decay) |
| `/ant:status` | 📊 Colony status, pheromones, progress |
| `/ant:phase [N\|list]` | 📋 View phase details |
| `/ant:continue` | ▶️ Approve phase, advance to next |
| `/ant:pause-colony` | ⏸️ Save state for session break |
| `/ant:resume-colony` | ▶️ Restore from pause |
| `/ant:organize` | 🧹 Codebase hygiene report |
| `/ant` | ❓ Show help and overview |

---

## 🗂️ File Structure

```
~/.claude/commands/ant/        # Global commands (installed once)
    ├── ant.md                 # Help overview
    ├── init.md                # Initialize colony + create state files
    ├── colonize.md            # Analyze codebase, persist findings
    ├── plan.md                # Generate plan (environment-aware)
    ├── build.md               # Execute phase (Queen-driven, live visibility)
    ├── continue.md            # Advance phase (skip if auto-learned)
    ├── focus.md               # Emit FOCUS signal
    ├── redirect.md            # Emit REDIRECT signal
    ├── feedback.md            # Emit FEEDBACK signal
    ├── status.md              # Colony status dashboard
    ├── phase.md               # Phase details
    ├── pause-colony.md        # Save session state
    ├── resume-colony.md       # Restore session state
    └── organize.md            # Codebase hygiene report

~/.aether/                     # Global runtime (installed once)
├── aether-utils.sh            # ~370-line utility wrapper (18 subcommands)
├── workers/
│   ├── colonizer-ant.md       # 🔍 Codebase exploration spec
│   ├── route-setter-ant.md    # 🗺️ Phase planning spec
│   ├── builder-ant.md         # 🔨 Code implementation spec
│   ├── watcher-ant.md         # 👁️ Validation/testing spec
│   ├── scout-ant.md           # 🔎 Research/information spec
│   └── architect-ant.md       # 📐 Knowledge synthesis spec
├── utils/
│   ├── atomic-write.sh        # Corruption-safe writes
│   └── file-lock.sh           # File locking for concurrent access
├── docs/
│   └── pheromones.md          # Pheromone user guide
├── QUEEN_ANT_ARCHITECTURE.md  # Architecture spec
└── learnings.json             # Cross-project knowledge

.aether/data/                  # Per-project state (created by /ant:init)
├── COLONY_STATE.json          # Colony goal, state, workers, spawn outcomes
├── pheromones.json            # Decaying pheromone signals
├── PROJECT_PLAN.json          # Phase plan with tasks and success criteria
├── errors.json                # Error log + flagged patterns
├── events.json                # Event log (capped at 100)
├── memory.json                # Phase learnings + decisions
└── activity.log               # Live worker progress (per-phase)
```

---

## 🐜 Why Ants?

Ant colonies demonstrate **superlinear intelligence**:

- Single ant: ~250 neurons (can barely navigate)
- Colony of 1M ants: farms, builds, wages war
- **No central brain** — the colony IS the intelligence

Aether translates this to AI agents:
- 👑 Queen = intention (not control)
- 🧪 Pheromones = signals (not commands)
- 🐜 Workers = autonomous specialists (spawn each other)
- 🏛️ Colony = the intelligence (distributed)

---

## 📜 Version History

| Version | Name | Date | Highlights |
|---------|------|------|------------|
| **v5.0** | NPM Distribution | 2026-02-05 | 📦 `npm install -g`, global/local split, CLI |
| **v4.4** | Field-Tested & Polished | 2026-02-05 | 32 field notes addressed, adaptive complexity, spawn tree |
| **v4.3** | Live Visibility & Auto-Learning | 2026-02-04 | 👁️ Activity log, Queen-driven execution, 🧠 auto-learning |
| **v4.2** | Colony Hardening | 2026-02-03 | Per-caste pheromone math, watcher execution, worker progress |
| **v4.1** | Cleanup & Enforcement | 2026-02-03 | Orphan audit, spawn limits, pheromone validation |
| **v4.0** | Hybrid Foundation | 2026-02-03 | `aether-utils.sh`, 18 subcommands, 11 audit fixes |
| **v3.0** | Rebuild | 2026-02-03 | Claude-native rewrite, 19→12 commands, 10→6 workers |
| **v2.0** | Event & Visual | 2026-02-02 | Event polling, visual indicators, E2E testing |
| **v1.0** | MVP | 2026-02-02 | Full colony system, 8 phases, 44 plans |

---

**MIT License**

*🐜 "The whole is greater than the sum of its parts." 🐜* — Aristotle
