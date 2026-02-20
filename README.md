```
     _    _____ _____ _   _ _____ ____
    / \  | ____|_   _| | | | ____|  _ \
   / _ \ |  _|   | | | |_| |  _| | |_) |
  / ___ \| |___  | | |  _  | |___|  _ <
 /_/   \_\_____| |_| |_| |_|_____|_| \_\
```

<div align="center">

  **22 specialized agents that spawn, coordinate, and self-organize.**

  ➡️ Click **Use this template** (top-right) to create your own Aether repo in 30 seconds.

  *Inspired by [glittercowboy's GSD system](https://github.com/glittercowboy/gsd)*

  [![npm version](https://img.shields.io/npm/v/aether-colony.svg)](https://www.npmjs.com/package/aether-colony)
  [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

  **v2.0.0** — Worker Emergence (22 real Claude Code agents)
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
   ├── 🏗️ Architects — extract patterns
   ├── 🏺 Archaeologists — excavate git history
   ├── 🔮 Oracles — deep research (RALF pattern)
   └── 🎲 Chaos Ants — resilience testing
```

When a Builder hits something complex, it spawns a Scout to research. When code is written, a Watcher spawns to verify. **The colony adapts to the problem.**

### Key Features

- **22 Claude Code Agents** — Real subagents, not definitions — `/ant:build` spawns a genuine `aether-builder`
- **34 Slash Commands** — Lifecycle, research, coordination, and utility
- **Real Agent Spawning** — Run `/ant:build 1` and a real builder spawns to write your code
- **6-Phase Verification** — Build, types, lint, tests, security, diff
- **Colony Memory** — Learnings and instincts persist across sessions
- **Pheromone Signals** — Focus, Redirect, Feedback to steer the colony
- **Pause/Resume** — Full state serialization for context breaks
- **Oracle Deep Research** — 50+ iteration autonomous research loop

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

This installs 22 Claude Code agents to `~/.claude/agents/ant/` plus 34 slash commands.

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
| `/ant:entomb` | Create chamber from completed colony |

**Core Flow:**
```
/ant:init → /ant:plan → /ant:build 1 → /ant:continue → /ant:build 2 → ... → /ant:seal
```

### Research & Analysis

| Command | Description |
|---------|-------------|
| `/ant:colonize` | 4 parallel scouts analyze your codebase |
| `/ant:archaeology <path>` | Excavate git history for any file |
| `/ant:oracle ["topic"]` | Deep research (50+ iteration loop) |
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
| `/ant:help` | Full command reference |

---

## CLI Commands

```bash
# View version and status
aether version

# Update all registered repos
aether update --all

# Checkpoints (safe snapshots)
aether checkpoint create "before refactor"
aether checkpoint list
aether checkpoint restore <id>

# View telemetry
aether telemetry

# Context
aether context        # Show context including nestmates
aether nestmates      # List sibling colonies
aether spawn-tree     # Display worker spawn tree
```

---

## The Castes

Workers have distinct personalities and roles, organized by tier:

### Core Workers

| Caste | Role | Personality |
|-------|------|-------------|
| 👑 **Queen** | Orchestrates, spawns workers | Coordinating |
| 🔨 **Builder** | Writes code, TDD-first | Pragmatic, direct |
| 👁️ **Watcher** | Tests, validates | Vigilant, careful |
| 🔍 **Scout** | Researches, discovers | Curious |

### Orchestration

| Caste | Role | Personality |
|-------|------|-------------|
| 📋 **Route-Setter** | Plans phases | Structured |
| 🗺️ **Colonizer** | Explores codebases, maps structure | Exploratory |
| 📊 **Surveyor** | Measures codebase metrics | Systematic |

### Specialists

| Caste | Role | Personality |
|-------|------|-------------|
| 📚 **Keeper** | Curates knowledge, patterns | Preserving |
| 🐛 **Tracker** | Investigates bugs, root cause | Methodical |
| 🧪 **Probe** | Generates tests | Thorough |
| 🔄 **Weaver** | Refactors code | Transformative |
| 👥 **Auditor** | Reviews code quality | Critical |

### Niche

| Caste | Role | Personality |
|-------|------|-------------|
| 📦 **Gatekeeper** | Dependency audits | Protective |
| ♿ **Includer** | Accessibility audits | Inclusive |
| ⚡ **Measurer** | Performance profiling | Precise |
| 🎲 **Chaos** | Resilience testing | Adversarial |
| 🏺 **Archaeologist** | Excavates git history | Investigative |
| 🔌 **Ambassador** | Third-party APIs | Diplomatic |
| 📝 **Chronicler** | Documentation | Recording |
| 🔮 **Sage** | Deep research (RALF loop) | Analytical |

Run `/ant:colonize` and 4 surveyor agents map your entire codebase. Run `/ant:build 1` and a real builder spawns.

---

## Spawn Depth

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

---

## 6-Phase Verification

Before any phase advances:

| Gate | Check |
|------|-------|
| Build | Project compiles/bundles |
| Types | Type checker passes |
| Lint | Linter passes |
| Tests | All tests pass (80%+ coverage target) |
| Security | No exposed secrets or debug artifacts |
| Diff | Review changes, no unintended modifications |

---

## File Structure

See [repo-structure.md](repo-structure.md) for a quick overview of all directories.

```
<your-repo>/.aether/              # Repo-local runtime
    ├── workers.md                # Worker specs and spawn protocol
    ├── aether-utils.sh           # Utility layer (80+ subcommands)
    ├── model-profiles.yaml       # Model routing config
    │
    ├── docs/                     # Documentation
    ├── utils/                    # Utility scripts
    ├── oracle/                   # Oracle research
    │
    ├── data/                     # State (NEVER synced)
    │   ├── COLONY_STATE.json     # Goal, plan, memory
    │   ├── constraints.json      # Focus and redirects
    │   └── spawn-tree.txt        # Worker hierarchy
    │
    ├── dreams/                   # Session notes (NEVER synced)
    └── chambers/                 # Archived colonies
```

---

## Pheromone Signals

| Signal | Command | Use When |
|--------|---------|----------|
| FOCUS | `/ant:focus "area"` | "Pay attention here" |
| REDIRECT | `/ant:redirect "avoid"` | "Don't do this" (hard constraint) |
| FEEDBACK | `/ant:feedback "note"` | "Adjust based on this" |

- **Before builds**: FOCUS + REDIRECT to steer
- **After builds**: FEEDBACK to adjust
- **Hard constraints**: REDIRECT (will break)
- **Gentle nudges**: FEEDBACK (preferences)

---

## Typical Workflows

### Starting a New Project

```
1. /ant:init "Build feature X"     # Set the goal
2. /ant:colonize                    # Analyze existing code (optional)
3. /ant:plan                        # Generate phases
4. /ant:focus "security"            # Guide attention (optional)
5. /ant:build 1                     # Execute phase 1
6. /ant:continue                    # Verify, advance
7. Repeat until done
8. /ant:seal                        # Complete and archive
```

### Deep Research

```
/ant:oracle "research topic"    # Launch Oracle
/ant:oracle status              # Check progress
/ant:oracle stop                # Stop if needed
# Read findings in .aether/oracle/discoveries/
```

### When Stuck

```
/ant:dream                      # Let the Dreamer observe
/ant:swarm "bug description"    # 4 parallel scouts investigate
/ant:archaeology src/module/    # Excavate why code exists
/ant:chaos "auth flow"          # Test resilience
```

---

## Safety Features

- **File Locking** — Prevents concurrent modification
- **Atomic Writes** — Temp file + rename pattern
- **Update Transactions** — Two-phase commit with rollback
- **State Validation** — Schema validation before modifications
- **Git Checkpoints** — Automatic commits before phases
- **Checkpoint System** — Safe snapshots with `aether checkpoint`

---

## Known Issues

1. **Init may fail if `.aether/data/` doesn't exist** — Run `mkdir -p .aether/data` before `/ant:init` if you see "Error writing file".

2. **Long commands can cause Claude to lose context** — Commands like `build.md` (1,000+ lines) may cause Claude to skip steps. Use `/ant:status` and `/ant:continue` to recover.

---

## Installation & Updates

```bash
# Install globally
npm install -g aether-colony

# Verify install
aether version
ls ~/.claude/commands/ant/

# Update all registered repos
aether update --all

# Update npm package
npm update -g aether-colony
```

---

## Acknowledgments

Inspired by **[glittercowboy](https://github.com/glittercowboy)** and the **[GSD system](https://github.com/glittercowboy/gsd)**. GSD showed what Claude Code could become with the right orchestration. Aether adds ant colony dynamics — pheromones, castes, nested spawning, and self-organizing workers.

---

## License

MIT — do whatever you want with it.

---

<div align="center">

*🐜 The colony is greater than the sum of its ants. 🐜*

</div>
