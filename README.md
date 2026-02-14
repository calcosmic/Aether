```
     _    _____ _____ _   _ _____ ____
    / \  | ____|_   _| | | | ____|  _ \
   / _ \ |  _|   | | | |_| |  _| | |_) |
  / ___ \| |___  | | |  _  | |___|  _ <
 /_/   \_\_____| |_| |_| |_|_____|_| \_\
```

<div align="center">
  <img src="aether-logo.png" alt="Aether Logo" width="500">

  **A multi-agent orchestration system for Claude Code where workers spawn workers.**

  ➡️ Click **Use this template** (top-right) to create your own Aether repo in 30 seconds.

  *Inspired by [glittercowboy's GSD system](https://github.com/glittercowboy/gsd)*

  [![npm version](https://img.shields.io/npm/v/aether-colony.svg)](https://www.npmjs.com/package/aether-colony)
  [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

  **v1.1.0** — Production ready with model routing
</div>

---

> *"The whole is greater than the sum of its parts."* — Aristotle

---

## What Is Aether?

Aether brings **ant colony intelligence** to Claude Code. Instead of one agent doing everything sequentially, you get a colony of specialists that self-organize around your goal.

```
👑 Queen (you)
   │
   ▼ pheromone signals
   │
🐜 Workers spawn Workers (max depth 3)
   │
   ├── 🔨 Builders — implement code
   ├── 👁️ Watchers — verify & test
   ├── 🔍 Scouts — research docs
   ├── 🗺️ Colonizers — explore codebases
   ├── 📋 Route-setters — plan phases
   ├── 🏛️ Architects — extract patterns
   ├── 🏺 Archaeologists — excavate git history
   ├── 🔮 Oracles — deep research
   └── 🎲 Chaos Ants — resilience testing
```

When a Builder hits something complex, it spawns a Scout to research. When code is written, a Watcher spawns to verify. **The colony adapts to the problem.**

### Key Features

- **Model-Aware Routing** — Different castes use different AI models optimized for their tasks
- **33 Slash Commands** — Lifecycle, management, research, and utility commands
- **4 OpenCode Agents** — Specialized agents for different platforms
- **6-Phase Verification** — Build, types, lint, tests, security, diff before advancing
- **Colony Memory** — Learnings and instincts persist across sessions
- **Pause/Resume** — Full state serialization for context breaks

---

## Quick Start

### Prerequisites

- [Claude Code](https://claude.ai/code) (Anthropic's CLI)
- Node.js >= 16
- `jq` — `brew install jq` on macOS

### Installation

```bash
npm install -g aether-colony
```

This installs slash commands so Claude Code can find them:
- 📁 **Claude Code Commands** → `~/.claude/commands/ant/` (33 slash commands)

All runtime state, utilities, and worker specs live **repo-local** in `.aether/` — each project is self-contained.

### Your First Colony

Open Claude Code in any repo:

```bash
/ant:init "Build a REST API with authentication"
/ant:plan
/ant:build 1
/ant:continue
```

That's it. The colony takes over from there.

---

## Commands

Aether has **33 slash commands** organized into categories.

### Core Lifecycle

| Command | Purpose |
|---------|---------|
| `/ant:init "goal"` | Initialize colony with mission |
| `/ant:plan` | Generate phased roadmap (50-iteration research loop) |
| `/ant:build N` | Execute phase N with worker waves |
| `/ant:continue` | 6-phase verification, then advance |
| `/ant:status` | Colony overview |
| `/ant:pause-colony` | Save state for context break |
| `/ant:resume-colony` | Restore from pause |

### Pheromone Signals

| Command | Purpose |
|---------|---------|
| `/ant:focus "area"` | Guide colony attention |
| `/ant:redirect "pattern"` | Warn away from approaches |
| `/ant:feedback "msg"` | Teach preferences |

### Power Commands

| Command | Purpose |
|---------|---------|
| `/ant:swarm "problem"` | Deploy 4 parallel scouts for stubborn bugs |
| `/ant:council` | Clarify intent via multi-choice questions |
| `/ant:oracle` | Deep research with 50+ iterations |

### Research & Analysis

| Command | Purpose |
|---------|---------|
| `/ant:colonize` | Analyze existing codebase |
| `/ant:archaeology <path>` | Excavate git history |
| `/ant:chaos <target>` | Resilience testing |
| `/ant:organize` | Codebase hygiene report |
| `/ant:dream` | Philosophical codebase wanderer |
| `/ant:interpret` | Ground dreams in reality |

### Issue Tracking

| Command | Purpose |
|---------|---------|
| `/ant:flag "issue"` | Create blocker/issue/note |
| `/ant:flags` | List and manage flags |

### Visibility

| Command | Purpose |
|---------|---------|
| `/ant:watch` | Live tmux monitoring |
| `/ant:phase N` | View phase details |
| `/ant:history` | Recent colony activity |
| `/ant:help` | Full command reference |

### System

| Command | Purpose |
|---------|---------|
| `/ant:update` | Sync system files from hub |
| `/ant:migrate-state` | Upgrade old state format |
| `/ant:verify-castes` | Check model routing |
| `/ant:seal` | Complete and archive colony |
| `/ant:entomb` | Create chamber from completed colony |

---

## CLI Commands

The `aether` CLI provides additional utilities:

```bash
# View version and status
aether version

# Manage model routing
aether caste-models list
aether caste-models set builder=kimi-k2.5

# Checkpoints
aether checkpoint create "before refactor"
aether checkpoint list
aether checkpoint restore <id>

# View telemetry
aether telemetry

# Sync state with planning docs
aether sync-state
```

---

## Model Routing

Aether routes different worker castes to optimal AI models via `.aether/model-profiles.yaml`:

| Caste | Model | Best For |
|-------|-------|----------|
| Prime, Archaeologist, Architect | `glm-5` | Long-horizon coordination (200K context) |
| Oracle | `minimax-2.5` | Research, web browsing (76.3% BrowseComp) |
| Builder, Watcher, Route-Setter, Chaos | `kimi-k2.5` | Code generation (76.8% SWE-Bench) |
| Scout, Colonizer | `minimax-2.5` | Parallel exploration, visual coding |

### How It Works

1. **Model Assignment** — Each caste mapped in `model-profiles.yaml`
2. **Environment Setup** — Queen sets `ANTHROPIC_MODEL` before spawning
3. **Proxy Routing** — Requests go through LiteLLM proxy at `localhost:4000`
4. **Fallback** — Unknown castes default to `kimi-k2.5`

### Proxy Configuration

```yaml
# .aether/model-profiles.yaml
proxy:
  endpoint: 'http://localhost:4000'
  auth_token: ${LITELLM_AUTH_TOKEN:-sk-litellm-local}
```

Set `LITELLM_AUTH_TOKEN` environment variable for custom auth.

---

## The Castes

| Caste | Role | Model |
|-------|------|-------|
| 👑 **Queen** | Orchestrates, spawns workers, synthesizes | glm-5 |
| 🔨 **Builder** | Writes code, TDD-first | kimi-k2.5 |
| 👁️ **Watcher** | Tests, validates, quality gates | kimi-k2.5 |
| 🔍 **Scout** | Researches docs, finds answers | minimax-2.5 |
| 🗺️ **Colonizer** | Explores codebases, maps structure | minimax-2.5 |
| 🏛️ **Architect** | Synthesizes patterns, coordinates docs | glm-5 |
| 📋 **Route-Setter** | Plans phases, breaks down goals | kimi-k2.5 |
| 🏺 **Archaeologist** | Excavates git history | glm-5 |
| 🔮 **Oracle** | Deep research, architecture analysis | minimax-2.5 |
| 🎲 **Chaos** | Resilience testing, adversarial probing | kimi-k2.5 |

---

## How It Works

### Spawn Depth

```
👑 Queen (depth 0)
└── 🔨 Builder-1 (depth 1) — can spawn 4 more
    ├── 🔍 Scout-7 (depth 2) — can spawn 2 more
    │   └── 🔍 Scout-12 (depth 3) — no more spawning
    └── 👁️ Watcher-3 (depth 2)
```

- **Depth 1**: Up to 4 spawns
- **Depth 2**: Up to 2 spawns (only if genuinely surprised)
- **Depth 3**: Complete inline, no further spawning
- **Global cap**: 10 workers per phase

### 6-Phase Verification Loop

Before any phase advances, the colony runs:

| Gate | Check |
|------|-------|
| Build | Project compiles/bundles |
| Types | Type checker passes |
| Lint | Linter passes |
| Tests | All tests pass (80%+ coverage target) |
| Security | No exposed secrets or debug artifacts |
| Diff | Review changes, no unintended modifications |

### Colony Memory

The colony learns across sessions:

```
Session 1: /ant:init → build → continue → complete
           └── completion-report.md saved with instincts & learnings

Session 2: /ant:init → reads completion-report.md → seeds memory
           └── Workers receive inherited knowledge in their prompts
```

**Instincts** are trigger→action patterns with confidence scores (0.0–1.0).

**Learnings** start as hypotheses and graduate to "validated" with evidence.

### Milestones

The colony tracks progress through auto-detected milestones:

| Milestone | Trigger |
|-----------|---------|
| First Mound | Colony initialized |
| Open Chambers | 1+ phase completed |
| Brood Stable | 3+ phases completed |
| Ventilated Nest | 5+ phases completed |
| Sealed Chambers | All phases completed |
| Crowned Anthill | Final celebration (explicit) |

Detected automatically via `milestone-detect` utility.

### Colony Lifecycle

```
/ant:init "goal"     → Start colony (First Mound)
       ↓
/ant:plan → /ant:build → /ant:continue (repeat)
       ↓
/ant:seal            → Complete colony, generate report
       ↓
/ant:entomb          → Archive to .aether/chambers/
       ↓
/ant:lay-eggs "new"  → Start fresh colony (preserves instincts)
```

**Lay-Eggs** starts a new colony cycle while inheriting high-confidence instincts from the previous colony — like a queen ant laying eggs to begin anew.

---

## File Structure

```
<your-repo>/.aether/              # Repo-local runtime
    ├── workers.md                # Worker specs and spawn protocol
    ├── aether-utils.sh           # Utility layer (50+ subcommands)
    ├── model-profiles.yaml       # Caste-to-model routing
    ├── verification-loop.md      # 6-phase verification reference
    │
    ├── data/                     # Per-project state
    │   ├── COLONY_STATE.json     # Goal, plan, memory, instincts
    │   ├── flags.json            # Blockers, issues, notes
    │   ├── constraints.json      # Focus areas and redirects
    │   ├── activity.log          # Worker activity stream
    │   ├── spawn-tree.txt        # Spawn hierarchy
    │   ├── telemetry.json        # Model performance data
    │   └── completion-report.md  # End-of-project summary
    │
    ├── dreams/                   # Dream session files
    └── chambers/                 # Entombed (archived) colonies

bin/                              # CLI
├── cli.js                        # Main entry point
└── lib/                          # Library modules
    ├── model-profiles.js         # Model routing logic
    ├── state-sync.js             # State reconciliation
    ├── update-transaction.js     # Atomic updates with rollback
    ├── file-lock.js              # Concurrent access control
    ├── telemetry.js              # Performance tracking
    └── errors.js                 # Error class hierarchy
```

---

## Typical Workflow

```
1. /ant:init "Build feature X"     # Set the goal
2. /ant:colonize                    # Analyze existing code (optional)
3. /ant:plan                        # Colony generates phases
4. /ant:focus "security"            # Guide attention (optional)
5. /ant:build 1                     # Execute phase 1
6. /ant:continue                    # Review, advance
7. /ant:build 2                     # Repeat until done
8. /ant:seal                        # Complete and archive
```

### Between Sessions

```bash
/ant:pause-colony               # Save state + handoff doc
# ... take a break ...
/ant:resume-colony              # Restore and continue
```

### When Stuck

```bash
/ant:dream                      # Let the Dreamer observe
/ant:interpret                  # Ground the dream in evidence
/ant:swarm "the bug description"  # 4 parallel scouts investigate
/ant:oracle "research topic"    # Deep research (50+ iterations)
/ant:archaeology src/module/    # Excavate why code exists
/ant:chaos "auth flow"          # Test resilience
```

---

## OpenCode Agents

Aether includes 4 specialized OpenCode agents:

| Agent | Purpose | Temperature |
|-------|---------|-------------|
| `aether-queen` | Orchestrates phases, spawns workers | 0.3 |
| `aether-builder` | Implements code, TDD-first | 0.2 |
| `aether-scout` | Researches, gathers information | 0.4 |
| `aether-watcher` | Validates, tests, quality gates | 0.1 |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      USER INTERFACE                         │
├──────────────────┬──────────────────┬──────────────────────┤
│  /ant:commands   │   aether CLI     │   OpenCode agents    │
│  (33 commands)   │   (bin/cli.js)   │   (4 agents)         │
└────────┬─────────┴────────┬─────────┴──────────┬───────────┘
         │                  │                    │
         ▼                  ▼                    ▼
┌─────────────────────────────────────────────────────────────┐
│                    UTILITIES LAYER                          │
│  aether-utils.sh (50+ subcommands) + bin/lib/* (10 modules) │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                    STATE LAYER                              │
│         .aether/data/ (COLONY_STATE, flags, constraints)    │
└─────────────────────────────────────────────────────────────┘
```

---

## Safety Features

- **File Locking** — Prevents concurrent modification of state
- **Atomic Writes** — Temp file + rename pattern
- **Update Transactions** — Two-phase commit with rollback
- **Git Checkpoints** — Automatic commits before phases
- **Ant Graveyards** — Failed files marked for future caution

---

## Disciplines

Workers follow strict disciplines:

| Discipline | Rule |
|------------|------|
| **Verification** | No completion claims without fresh evidence |
| **TDD** | No production code without a failing test first |
| **Debugging** | No fixes without root cause investigation (3-fix rule) |
| **Learning** | Pattern detection with validation lifecycle |
| **Coding Standards** | KISS, DRY, YAGNI, readable code |

---

## Installation & Updates

```bash
# Install globally
npm install -g aether-colony

# Verify install
aether version
ls ~/.claude/commands/ant/

# Verify runtime (from inside any repo)
ls .aether/

# Update
npm update -g aether-colony

# Uninstall (preserves project state)
aether uninstall && npm uninstall -g aether-colony
```

---

## Acknowledgments

Massive shoutout to **[glittercowboy](https://github.com/glittercowboy)** and the **[GSD (Get Shit Done) system](https://github.com/glittercowboy/gsd)**. GSD showed what Claude Code could become with the right orchestration. Aether takes that inspiration and adds ant colony dynamics — pheromones, castes, nested spawning, and model-aware routing.

---

## License

MIT — do whatever you want with it.

---

<div align="center">

*🐜 The colony is greater than the sum of its ants. 🐜*

</div>
