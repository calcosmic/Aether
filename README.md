<div align="center">

```
      █████╗ ███████╗████████╗██╗  ██╗███████╗██████╗
     ██╔══██╗██╔════╝╚══██╔══╝██║  ██║██╔════╝██╔══██╗
     ███████║█████╗     ██║   ███████║█████╗  ██████╔╝
     ██╔══██║██╔══╝     ██║   ██╔══██║██╔══╝  ██╔══██╗
     ██║  ██║███████╗   ██║   ██║  ██║███████╗██║  ██║
     ╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
```

**Multi-agent system using ant colony intelligence for Claude Code and OpenCode**

[![npm version](https://img.shields.io/npm/v/aether-colony.svg)](https://www.npmjs.com/package/aether-colony)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**v1.0.0**
</div>

---

## What Is Aether?

Aether brings **ant colony intelligence** to Claude Code and OpenCode. Instead of one agent doing everything sequentially, you get a colony of specialists that self-organize around your goal.

```
👑 Queen (you)
   │
   ▼ pheromone signals
   │
🐜 Workers spawn Workers (max depth 3)
   │
   ├── 🔨🐜 Builders — implement code
   ├── 👁️🐜 Watchers — verify & test
   ├── 🔍🐜 Scouts — research docs
   ├── 🐛🐜 Trackers — investigate bugs
   ├── 🗺️🐜 Colonizers — explore codebases (4 parallel scouts)
   ├── 📋🐜 Route-setters — plan phases
   ├── 🏺🐜 Archaeologists — excavate git history
   ├── 🎲🐜 Chaos Ants — resilience testing
   └── 📚🐜 Keepers — preserve knowledge
```

When a Builder hits something complex, it spawns a Scout to research. When code is written, a Watcher spawns to verify. **The colony adapts to the problem.**

### Key Features

- **9 Active Agent Types** — Real subagents spawned by commands
- **35 Slash Commands** — Lifecycle, research, coordination, and utility
- **Real Agent Spawning** — Run `/ant:build 1` and real builders spawn
- **6-Phase Verification** — Build, types, lint, tests, security, diff
- **Colony Memory** — Learnings and instincts persist across sessions
- **Pheromone Signals** — Focus, Redirect, Feedback to steer the colony
- **Pause/Resume** — Full state serialization for context breaks

---

## Quick Start

### Prerequisites

- [Claude Code](https://claude.ai/code) or [OpenCode](https://opencode.ai)
- Node.js >= 16
- `jq` — `brew install jq` on macOS

### Installation

```bash
# NPX installer (recommended)
npx aether-colony install

# Or npm global install
npm install -g aether-colony
```

### Your First Colony

Open Claude Code in any repo:

```bash
/ant:init "Build a REST API with authentication"
/ant:plan
/ant:build 1
/ant:continue
```

---

## Command Reference

### Core Lifecycle

| Command | Description |
|---------|-------------|
| `/ant:init "goal"` | Initialize colony with mission |
| `/ant:plan` | Generate phased roadmap |
| `/ant:build N` | Execute phase N with worker waves |
| `/ant:continue` | 6-phase verification, advance to next phase |
| `/ant:pause-colony` | Save state for context break |
| `/ant:resume-colony` | Restore from pause |
| `/ant:seal` | Complete and archive colony |

**Core Flow:**
```
/ant:init → /ant:plan → /ant:build 1 → /ant:continue → /ant:build 2 → ... → /ant:seal
```

### Research & Analysis

| Command | Description |
|---------|-------------|
| `/ant:colonize` | 4 parallel scouts analyze your codebase |
| `/ant:archaeology <path>` | Excavate git history for any file |
| `/ant:chaos <target>` | Resilience testing, edge case probing |
| `/ant:swarm ["problem"]` | 4 parallel scouts for stubborn bugs |
| `/ant:dream` | Philosophical codebase wanderer |
| `/ant:organize` | Codebase hygiene report |

### Coordination

| Command | Description |
|---------|-------------|
| `/ant:council` | Clarify intent via multi-choice questions |
| `/ant:focus "area"` | FOCUS signal — guide attention |
| `/ant:redirect "pattern"` | REDIRECT signal — hard constraint |
| `/ant:feedback "msg"` | FEEDBACK signal — teach preferences |

### Visibility

| Command | Description |
|---------|-------------|
| `/ant:status` | Colony overview |
| `/ant:watch` | Real-time swarm display |
| `/ant:history` | Recent activity log |
| `/ant:flags` | List blockers and issues |
| `/ant:memory-details` | Wisdom, pending promotions, failures |
| `/ant:help` | Full command reference |

---

## The Active Castes

These agents are spawned by commands:

| Caste | Role | Spawned By |
|-------|------|------------|
| 👑 **Queen** | Orchestrates, spawns workers | You (the user) |
| 🔨 **Builder** | Writes code, TDD-first | `/ant:build` |
| 👁️ **Watcher** | Tests, validates | `/ant:build` |
| 🔍 **Scout** | Researches, discovers | `/ant:build`, `/ant:oracle`, `/ant:swarm` |
| 🐛 **Tracker** | Investigates bugs | `/ant:swarm` |
| 🗺️ **Surveyor** | Explores codebases | `/ant:colonize` (4 parallel) |
| 📋 **Route-Setter** | Plans phases | `/ant:plan` |
| 🏺 **Archaeologist** | Excavates git history | `/ant:archaeology`, `/ant:build` |
| 🎲 **Chaos** | Resilience testing | `/ant:chaos`, `/ant:build` |
| 📚 **Keeper** | Preserves knowledge | `/ant:continue` |

---

## Spawn Depth

```
👑 Queen (depth 0)
└── 🔨🐜 Builder-1 (depth 1) — can spawn 4 more
    ├── 🔍🐜 Scout-7 (depth 2) — can spawn 2 more
    │   └── 🔍🐜 Scout-12 (depth 3) — no more spawning
    └── 👁️🐜 Watcher-3 (depth 2)
```

---

## 6-Phase Verification

Before any phase advances:

| Gate | Check |
|------|-------|
| Build | Project compiles/bundles |
| Types | Type checker passes |
| Lint | Linter passes |
| Tests | All tests pass |
| Security | No exposed secrets |
| Diff | Review changes |

---

## File Structure

```
<your-repo>/.aether/              # Repo-local runtime
    ├── QUEEN.md                  # Colony wisdom
    ├── workers.md                # Worker specs
    ├── aether-utils.sh           # Utility layer (80+ subcommands)
    │
    ├── docs/                     # Documentation
    ├── utils/                    # Utility scripts
    ├── templates/                # File templates
    │
    ├── data/                     # State (NEVER synced)
    │   ├── COLONY_STATE.json     # Goal, plan, memory
    │   ├── constraints.json      # Focus and redirects
    │   ├── pheromones.json       # Signal tracking
    │   └── midden/               # Failure tracking
    │
    ├── dreams/                   # Session notes
    └── chambers/                 # Archived colonies
```

---

## Pheromone Signals

| Signal | Command | Use When |
|--------|---------|----------|
| FOCUS | `/ant:focus "area"` | "Pay attention here" |
| REDIRECT | `/ant:redirect "avoid"` | "Don't do this" |
| FEEDBACK | `/ant:feedback "note"` | "Adjust based on this" |

---

## Typical Workflow

```
1. /ant:init "Build feature X"     # Set the goal
2. /ant:colonize                    # Analyze codebase (optional)
3. /ant:plan                        # Generate phases
4. /ant:build 1                     # Execute phase 1
5. /ant:continue                    # Verify, advance
6. Repeat until done
7. /ant:seal                        # Complete and archive
```

---

## Safety Features

- **File Locking** — Prevents concurrent modification
- **Atomic Writes** — Temp file + rename pattern
- **State Validation** — Schema validation
- **Session Freshness Detection** — Stale sessions handled

---

## CLI Commands

```bash
aether version              # View version
aether update               # Update system files from hub
aether update --all         # Update all registered repos
aether telemetry            # View usage stats
aether spawn-tree           # Display worker spawn tree
```

---

## License

MIT
