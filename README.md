# 🐜 AETHER v1.0

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

### Prerequisites

- [Claude Code](https://claude.com/claude-code) (Anthropic's CLI for Claude)
- Node.js >= 16
- `jq` (JSON processor) — `brew install jq` on macOS

### Via npm (recommended)

```bash
npm install -g aether-colony
```

This installs the `aether` CLI and automatically sets up:
- **Commands** → `~/.claude/commands/ant/` (16 Claude Code skill prompts)
- **Runtime** → `~/.aether/` (worker specs, utility scripts, docs)

**Existing repos:** If you previously used Aether, delete any local `.claude/commands/ant/` directory in your projects — the global install handles everything now. State auto-upgrades when you run any command.

### From source

```bash
git clone https://github.com/callumcowie/Aether.git
cd Aether
node bin/cli.js install
```

### Verify installation

```bash
aether version              # Shows installed version
ls ~/.claude/commands/ant/   # 16 command files
cat ~/.aether/workers.md     # Worker specs (consolidated)
```

### Update

```bash
# Via npm
npm update -g aether-colony

# From source
cd Aether && git pull && node bin/cli.js install
```

The install command is idempotent — it overwrites existing files safely. Per-project `.aether/data/` state and `~/.aether/learnings.json` (cross-project knowledge) are never touched.

### Uninstall

```bash
aether uninstall             # Removes global files, preserves learnings
npm uninstall -g aether-colony
```

Per-project `.aether/data/` directories are never touched by uninstall. Cross-project learnings (`~/.aether/learnings.json`) are preserved.

---

## 🚀 Quick Start

Open Claude Code in any repo and run:

```bash
/ant:init "Build a REST API with authentication"
/ant:plan
/ant:build 1
```

The colony will self-organize: a Route-setter plans the work, Builders implement it, Watchers validate it, and the Queen orchestrates with live visibility.

### Typical Workflow

```
1. /ant:init "Build a REST API with auth"    # Set colony intention
2. /ant:colonize                              # Analyze existing code (optional)
3. /ant:plan                                  # Colony generates phases
4. /ant:focus "security"                      # Guide attention (optional)
5. /ant:build 1                               # Execute phase 1
6. /ant:continue                              # Review, advance to phase 2
7. /ant:build 2                               # Repeat until done
```

Or use auto-continue to run all phases:

```
/ant:continue --all                           # Runs remaining phases with quality gates
```

Auto-continue halts if a watcher scores a phase below 4/10 or after 2 consecutive failures.

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

**Each can spawn others** based on local needs. Workers signal spawn requests to the Queen, who fulfills them between waves (max depth 2, max 2 sub-spawns per wave).

### 3. 🧪 Pheromone Communication

| Signal | Purpose | Duration | Strength |
|--------|---------|----------|----------|
| 🟢 **INIT** | Set colony goal | Persists | 1.0 |
| 🎯 **FOCUS** | Guide attention | 1 hour | 0.7 |
| 🚫 **REDIRECT** | Warn away from approach | 24 hours | 0.9 |
| 💬 **FEEDBACK** | Teach preferences | 6 hours | 0.5 |

**Signals, not commands.** Pheromones decay exponentially. Each caste has different sensitivity values, so the same signal produces different effective strengths per caste. Ants compute `effective_signal = sensitivity * current_strength` and act based on thresholds (>0.5 PRIORITIZE, 0.3-0.5 NOTE, <0.3 IGNORE).

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

The Queen spawns workers sequentially and displays each worker's activity log output between spawns — you see what each ant did as it completes, not after the entire phase finishes. Workers write structured progress lines to an activity log during execution. Build output is ANSI-colored with caste-specific colors.

### 7. 🧠 Auto-Learning

After every build, the colony automatically extracts phase learnings from completed work (errors, events, task outcomes) and writes them to colony memory. A FEEDBACK pheromone is auto-emitted summarizing what worked and what failed. Learnings stay project-local in `memory.json`; at project completion, you can promote key learnings to the global tier (`~/.aether/learnings.json`) for cross-project knowledge transfer.

### 8. 🔍 Multi-Lens Colonization

When analyzing an existing codebase, `colonize` spawns 3 colonizer ants in parallel — each with a different lens (Structure, Patterns, Stack). The Queen synthesizes their findings, flags disagreements, and sets an adaptive complexity mode (LIGHTWEIGHT/STANDARD/FULL) that scales the colony's overhead to project size.

### 9. 🛡️ Quality Gates

After each wave of workers, the Queen auto-spawns an advisory reviewer (reusing the watcher spec) to assess quality. If a task fails twice, an auto-debugger spawns (reusing the builder spec with PATCH constraints). The watcher uses a calibrated 5-dimension scoring rubric with chain-of-thought reasoning.

---

## 🐜 All Commands

| Command | Purpose |
|---------|---------|
| `/ant:init "<goal>"` | 🟢 Set colony intention and initialize |
| `/ant:colonize` | 🔍 Analyze existing codebase (3 lenses: Structure/Patterns/Stack) |
| `/ant:plan` | 🗺️ Generate project plan (colony self-organizes) |
| `/ant:build <N>` | 🔨 Execute phase N (Queen spawns workers with live visibility) |
| `/ant:continue` | ▶️ Approve phase, extract learnings, advance to next |
| `/ant:continue --all` | ▶️ Auto-run all remaining phases with quality-gated halt |
| `/ant:focus "<area>"` | 🎯 Guide colony attention to specific areas |
| `/ant:redirect "<pat>"` | 🚫 Warn colony away from patterns |
| `/ant:feedback "<msg>"` | 💬 Provide guidance to colony |
| `/ant:status` | 📊 Colony status at a glance |
| `/ant:watch` | 👁️ Live tmux monitoring of colony activity |
| `/ant:phase [N\|list]` | 📋 View phase details |
| `/ant:organize` | 🧹 Codebase hygiene report (stale files, dead code) |
| `/ant:pause-colony` | ⏸️ Save state for session break |
| `/ant:resume-colony` | ▶️ Restore from pause |
| `/ant` | ❓ Show help and overview |

---

## 🗂️ File Structure

```
~/.claude/commands/ant/        # Global commands (installed once, shared across repos)
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

~/.aether/                     # Global runtime (installed once, shared across repos)
├── workers.md                 # Consolidated worker specs (all castes in one file)
├── aether-utils.sh            # Utility layer for logging, validation
├── utils/
│   ├── atomic-write.sh        # Corruption-safe writes
│   └── file-lock.sh           # File locking
├── docs/
│   └── pheromones.md          # Constraint system guide
├── QUEEN_ANT_ARCHITECTURE.md  # Architecture documentation
└── learnings.json             # Cross-project knowledge (50-entry cap)

<your-repo>/.aether/data/      # Per-project state (created by /ant:init)
├── COLONY_STATE.json          # Consolidated state (v3.0) — goal, plan, memory, errors, events
├── constraints.json           # Focus areas and avoid patterns
└── activity.log               # Worker activity log
```

---

## 📈 Current Status

**v1.0** — First Public Release (2026-02-07)

**What's built:**
- 📦 `npm install -g aether-colony` — global install, works in any repo
- 🐜 16 commands as Claude Code skill prompts
- 🐜 True emergence — workers spawn workers directly (no Queen mediation)
- 🔧 Consolidated worker specs in single `workers.md` (91% reduction from v0.x)
- 💾 Single `COLONY_STATE.json` replaces 6 distributed files
- 🚧 Depth-based spawn limits (max depth 3, max 10 workers per phase)
- 🧪 Simple JSON constraints replace complex pheromone decay math
- 👁️ Live worker visibility via tmux (`/ant:watch`)
- 🧠 Cross-project learning extraction
- 🔖 Git checkpoints before phase execution
- 📋 Iterative planning with confidence tracking (Scout + Route-Setter loop)
- 🧹 Codebase hygiene scanning (`/ant:organize`)

**Architecture:**
- Workers use Claude Code's Task tool to spawn sub-workers
- Depth 1 (Prime Worker) → spawns up to 4 specialists
- Depth 2 (Specialists) → spawn only when genuinely surprised (3x complexity)
- Depth 3 (Deep Specialists) → complete work inline, no further spawning

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
| **v1.0** | First Public Release | 2026-02-07 | True emergence (workers spawn workers), simplified state, npm distribution |

<details>
<summary>Development History (pre-release)</summary>

| Version | Name | Date | Highlights |
|---------|------|------|------------|
| v0.5 | NPM Distribution | 2026-02-05 | Global install, CLI, path migration |
| v0.4 | Field-Tested | 2026-02-04 | 32 field notes, spawn tree, auto-continue |
| v0.3 | Rebuild | 2026-02-03 | Claude-native rewrite, utility layer |
| v0.2 | Event & Visual | 2026-02-02 | Event polling, visual indicators |
| v0.1 | MVP | 2026-02-02 | Initial colony system |

</details>

---

**MIT License**

*🐜 "The whole is greater than the sum of its parts." 🐜* — Aristotle
