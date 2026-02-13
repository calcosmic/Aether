```
     _    _____ _____ _   _ _____ ____
    / \  | ____|_   _| | | | ____|  _ \
   / _ \ |  _|   | | | |_| |  _| | |_) |
  / ___ \| |___  | | |  _  | |___|  _ <
 /_/   \_\_____| |_| |_| |_|_____|_| \_\
```

<div align="center">
  <img src="aether-logo.png" alt="Aether Logo" width="500">

  **A multi-agent system for Claude Code and OpenCode where workers spawn workers.**

  ➡️ Click **Use this template** (top-right) to create your own Aether repo in 30 seconds.

  *Inspired by [glittercowboy's GSD system](https://github.com/glittercowboy/gsd)*

  [![npm version](https://img.shields.io/npm/v/aether-colony.svg)](https://www.npmjs.com/package/aether-colony)
  [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

  **v1.0.0** — First stable production release
</div>

---

> *"The whole is greater than the sum of its parts."* — Aristotle

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
   ├── 🔨 Builders — implement code
   ├── 👁️ Watchers — verify & test
   ├── 🔍 Scouts — research docs
   ├── 🗺️ Colonizers — explore codebases
   ├── 📋 Route-setters — plan phases
   ├── 🏛️ Architects — extract patterns
   ├── 🏺 Archaeologists — excavate git history
   └── 🎲 Chaos Ants — resilience testing
```

When a Builder hits something complex, it spawns a Scout to research. When code is written, a Watcher spawns to verify. **The colony adapts to the problem.**

---

## Quick Start

### Prerequisites

- [Claude Code](https://claude.ai/code) (Anthropic's CLI) and/or [OpenCode](https://github.com/sst/opencode)
- Node.js >= 16
- `jq` — `brew install jq` on macOS

### Installation

```bash
npm install -g aether-colony
```

This installs slash commands so your editor can find them:
- 📁 **Claude Code Commands** → `~/.claude/commands/ant/` (25 slash commands)
- 📁 **OpenCode Commands** → `~/.config/opencode/commands/ant/` (25 slash commands)
- 📁 **OpenCode Agents** → `~/.config/opencode/agents/` (4 specialized agents)

All runtime state, utilities, and worker specs live **repo-local** in `.aether/` — each project is self-contained.

### Your First Colony

Open Claude Code or OpenCode in any repo:

```bash
/ant:init "Build a REST API with authentication"
/ant:plan
/ant:build 1
/ant:continue
```

That's it. The colony takes over from there.

---

## Commands

Aether has **25 slash commands** organized into 8 categories.

### Core Workflow

| Command | Purpose |
|---------|---------|
| `/ant:init "goal"` | Set colony mission |
| `/ant:plan` | Generate phased roadmap |
| `/ant:build N` | Execute phase N |
| `/ant:continue` | Review & advance to next phase |

**`/ant:init "goal"`** — Initialize the colony with your project goal. Creates colony state, sets up constraints, and seeds memory from previous sessions. If a `completion-report.md` exists from a prior colony, instincts and learnings are automatically inherited.
```bash
/ant:init "Build a REST API with JWT authentication"
```

**`/ant:plan`** — Generate a phased project plan. The colony iterates (up to 50 times) with Scout research and Route-Setter planning until confidence reaches 95%. Includes anti-stuck detection.

**`/ant:build N`** — Execute phase N of your plan. The Queen spawns Builders and Watchers in parallel waves. Workers can spawn sub-workers up to depth 3.
```bash
/ant:build 1              # Build phase 1 (compact output)
/ant:build 3 --verbose    # Build phase 3 (full details)
```

**`/ant:continue`** — Reviews work through 6 verification gates (build, types, lint, tests, security, diff), checks success criteria with evidence, then advances to the next phase.

---

### Pheromone Signals

| Command | Purpose |
|---------|---------|
| `/ant:focus "area"` | Guide colony attention |
| `/ant:redirect "pattern"` | Warn away from approaches |
| `/ant:feedback "msg"` | Teach preferences |

**`/ant:focus "area"`** — Tell the colony to pay special attention to something. Max 5 focus areas.
```bash
/ant:focus "error handling in the auth module"
```

**`/ant:redirect "pattern"`** — Warn the colony away from specific approaches. These act as hard constraints.
```bash
/ant:redirect "don't use jsonwebtoken, use jose instead"
```

**`/ant:feedback "msg"`** — Teach the colony your preferences. Creates an instinct that persists across phases.
```bash
/ant:feedback "prefer composition over inheritance"
```

---

### Power Commands

| Command | Purpose |
|---------|---------|
| `/ant:council` | Clarify intent via multi-choice |
| `/ant:swarm "problem"` | Stubborn bug destroyer |

**`/ant:council`** — Convene the council when you need to clarify your intent through guided multi-choice questions. Auto-injects appropriate signals based on your answers.
```bash
/ant:council    # Opens interactive clarification session
```

**`/ant:swarm "problem"`** — The nuclear option for stubborn bugs. Deploys 4 parallel scouts to investigate from different angles, then applies the best fix automatically.
```bash
/ant:swarm "Tests keep failing in auth module with undefined error"
```

The 4 scouts:
- 🏛️ **Git Archaeologist** — Traces git history to find when it broke
- 🔍 **Pattern Hunter** — Finds similar working code in the codebase
- 💥 **Error Analyst** — Parses error chains to identify root cause
- 🌐 **Web Researcher** — Searches docs/issues for known solutions

---

### Dreaming & Reflection

| Command | Purpose |
|---------|---------|
| `/ant:dream` | Philosophical codebase wanderer |
| `/ant:interpret` | Ground dreams in reality |

**`/ant:dream`** — The Dreamer is the colony's philosopher. It wanders the codebase reading code, git history, and TO-DOs. It observes patterns others miss because they're too busy building. Sessions saved to `.aether/dreams/`.
```bash
/ant:dream    # The Dreamer explores and writes
```

**`/ant:interpret`** — The Interpreter loads dream sessions, investigates observations against the actual codebase with evidence, and delivers verdicts (confirmed / partially confirmed / unconfirmed / refuted).
```bash
/ant:interpret    # Review and ground the latest dream
```

---

### Analysis & Investigation

| Command | Purpose |
|---------|---------|
| `/ant:colonize` | Analyze existing codebase |
| `/ant:archaeology <path>` | Excavate git history |
| `/ant:chaos <target>` | Resilience testing |
| `/ant:organize` | Codebase hygiene report |

**`/ant:colonize`** — Analyze an existing codebase before planning. Scans key files and creates a codebase map including detected build/test/lint commands. Run this before `/ant:plan` when working with existing projects.

**`/ant:archaeology <path>`** — The Archaeologist excavates git history to explain WHY code exists. Surfaces tribal knowledge buried in commits, identifies workarounds, tech debt, and dead code candidates.
```bash
/ant:archaeology src/auth/
/ant:archaeology lib/legacy/cache.ts
```

**`/ant:chaos <target>`** — The Chaos Ant is a resilience tester that probes edge cases, boundary conditions, and unexpected inputs. Investigates 5 scenarios across 5 categories to strengthen code.
```bash
/ant:chaos src/auth/login.ts
/ant:chaos "user signup flow"
```

**`/ant:organize`** — Run a hygiene report on the codebase. Scans for stale files, dead code, orphaned configs. Report-only — doesn't modify files.

---

### Issue Tracking

| Command | Purpose |
|---------|---------|
| `/ant:flag "issue"` | Create a flag |
| `/ant:flags` | List all flags |

**`/ant:flag "issue"`** — Create a flag to track blockers, issues, or notes. Flags persist across context resets.
```bash
/ant:flag "Database migration needs manual review" --type blocker
/ant:flag "Consider adding rate limiting" --type issue
/ant:flag "Good pattern for error handling" --type note
```

**`/ant:flags`** — List all flags, resolve them, or acknowledge issues.
```bash
/ant:flags                    # List all active flags
/ant:flags --resolve 3 "Fixed in commit abc123"
```

---

### Visibility & Status

| Command | Purpose |
|---------|---------|
| `/ant:status` | Colony overview |
| `/ant:phase N` | View phase details |
| `/ant:watch` | Live tmux monitoring |
| `/ant:help` | Full command reference |

**`/ant:status`** — Quick overview of colony state: current phase, confidence, active constraints, recent activity, and flags. Use `--verbose` for extended details.

**`/ant:phase N`** — View details of a specific phase including tasks, status, and assigned castes.

**`/ant:watch`** — Set up a tmux session with 4 panes showing real-time colony activity: status, progress bar, spawn tree, and activity log. Requires tmux.

**`/ant:help`** — Full command reference covering all commands, session resume workflow, and state file inventory.

---

### Session Management

| Command | Purpose |
|---------|---------|
| `/ant:pause-colony` | Save state for break |
| `/ant:resume-colony` | Restore from pause |
| `/ant:migrate-state` | Upgrade old state format |
| `/ant:update` | Update system files from hub |

**`/ant:pause-colony`** — Save full colony state to a handoff document when you need to take a break. Creates `.aether/HANDOFF.md`.

**`/ant:resume-colony`** — Restore colony state from a previous pause.

**`/ant:migrate-state`** — One-time migration for colonies created with older state formats.

**`/ant:update`** — Update this repo's Aether system files from the global distribution hub (`~/.aether/`). Syncs commands, agents, and runtime files.

> **Tip:** All commands show auto-recovery headers after `/clear`. You never lose context — the colony always knows where it was.

---

## Features

### Core
- 🐜 **Nested Spawning** — Workers spawn sub-workers (depth 1→2→3 chains)
- 🎨 **Colorized Output** — Each caste has its own terminal color
- 👁️ **Runtime Verification** — Watchers actually execute code, not just read it
- 🚩 **Flagging System** — Issues persist across context resets
- 🔨 **Named Ants** — Hammer-42, Vigil-17, Quest-33... they feel real
- 📊 **Spawn Tree Visualization** — See the colony hierarchy in real-time

### Memory & Learning
- 🧠 **Colony Memory Inheritance** — New colonies inherit instincts and learnings from previous sessions via `completion-report.md`
- 📚 **Knowledge in Prompts** — Spawned workers receive top instincts, recent learnings, and flagged error patterns
- 💭 **Dream/Interpret Cycle** — A philosophical Dreamer observes; an Interpreter grounds dreams in evidence

### Safety & Git
- 💾 **Git Checkpoints** — Automatic `aether-checkpoint:` commits before each phase
- 🔄 **Gate-Based Commits** — Colony suggests commits at verified boundaries (you stay in control)
- ⚰️ **Ant Graveyards** — When a builder fails on a file, a marker records what went wrong. Future builders increase caution.

### Verification
- 🔒 **6-Phase Verification Loop** — Build, types, lint, tests, security scan, diff review before phase advances
- 🏗️ **CLAUDE.md-Aware Commands** — Workers resolve build/test/lint commands via priority chain: CLAUDE.md → CODEBASE.md → fallback
- 📋 **Automatic Changelog** — `/ant:continue` appends a changelog entry for each completed phase

### Developer Experience
- 📏 **Compact-by-Default** — Status (~8 lines) and build (~12 lines) summaries. Use `--verbose` for full details.
- 🔄 **Auto-Recovery Headers** — Every command shows context after `/clear`
- ✅ **Lint Suite** — `npm run lint` validates shell scripts, JSON config, and mirror sync

---

## How It Works

### The Castes

| Caste | Role |
|-------|------|
| 👑 **Queen** | Orchestrates, spawns workers, synthesizes results |
| 🔨 **Builder** | Writes code, runs commands |
| 👁️ **Watcher** | Tests, validates, quality gates |
| 🔍 **Scout** | Researches docs, finds answers |
| 🗺️ **Colonizer** | Explores codebases, maps structure |
| 🏛️ **Architect** | Synthesizes patterns, extracts learnings |
| 📋 **Route-Setter** | Plans phases, breaks down goals |
| 🏺 **Archaeologist** | Excavates git history, surfaces tribal knowledge |
| 🎲 **Chaos** | Resilience testing, adversarial probing |

### Pheromone Signals

Instead of direct commands, you emit signals that the colony interprets:

| Signal | Purpose | Priority | Default Expiration |
|--------|---------|----------|--------------------|
| 🎯 `FOCUS` | "Pay attention to this" | normal | phase end |
| 🚫 `REDIRECT` | "Avoid this approach" | high | phase end |
| 💬 `FEEDBACK` | "Here's what I like/dislike" | low | phase end |

Workers read all active signals and adjust behavior. Use `--ttl` flag for wall-clock expiration (e.g., `--ttl 2h`).

### Iterative Planning (95% Confidence)

When you run `/ant:plan`, the colony iterates:

1. **Scout** researches the codebase, identifies knowledge gaps
2. **Route-Setter** drafts/refines the plan based on findings
3. **Loop** continues until confidence reaches 95% (max 50 iterations)

Confidence is measured across 5 dimensions: codebase knowledge, requirement clarity, risk identification, dependencies, and effort estimation.

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

### Colony Memory

The colony learns across sessions:

```
Session 1: /ant:init → build → continue → complete
           └── completion-report.md saved with instincts & learnings

Session 2: /ant:init → reads completion-report.md → seeds memory
           └── Workers receive inherited knowledge in their prompts
```

**Instincts** are trigger→action patterns with confidence scores (0.0–1.0). They strengthen with successful application and weaken on failure.

**Learnings** start as hypotheses and graduate to "validated" only when tested with evidence.

### Disciplines

Workers follow strict disciplines:

| Discipline | Purpose |
|------------|---------|
| **Verification** | No completion claims without fresh evidence |
| **TDD** | No production code without a failing test first |
| **Debugging** | No fixes without root cause investigation |
| **Learning** | Pattern detection with validation lifecycle |
| **Coding Standards** | KISS, DRY, YAGNI, readable code |

---

## File Structure

```
<your-repo>/.aether/              # Repo-local runtime
    ├── workers.md                # Worker specs with spawn protocol
    ├── aether-utils.sh           # Utility layer (59 subcommands)
    ├── verification-loop.md      # 6-phase verification reference
    ├── DISCIPLINES.md            # All worker disciplines
    ├── utils/                    # Colorization, spawn tree, file locking
    │
    ├── data/                     # Per-project state
    │   ├── COLONY_STATE.json     # Goal, plan, memory, instincts
    │   ├── flags.json            # Blockers, issues, notes
    │   ├── constraints.json      # Focus areas and avoidance patterns
    │   ├── activity.log          # Worker activity stream
    │   ├── spawn-tree.txt        # Spawn hierarchy
    │   ├── completion-report.md  # End-of-project summary
    │   ├── graveyard.json        # Failed file markers
    │   └── pathogens.json        # Error pattern signatures
    │
    └── dreams/                   # Dream session files

~/.claude/commands/ant/           # Claude Code slash commands
~/.config/opencode/               # OpenCode config
    ├── commands/ant/             # OpenCode slash commands
    └── agents/                   # Specialized agents (queen, builder, scout, watcher)
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
```

### Between Sessions

```bash
/ant:pause-colony               # Save state + handoff doc
/ant:resume-colony              # or just /ant:status — auto-recovery kicks in
```

### When Stuck

```bash
/ant:dream                      # Let the Dreamer observe
/ant:interpret                  # Ground the dream in evidence
/ant:swarm "the bug description"  # Nuclear option for stubborn issues
/ant:archaeology src/module/    # Excavate why code exists
/ant:chaos "auth flow"          # Test resilience
```

---

## Cross-Tool Compatibility

Both Claude Code and OpenCode share the same state files in `.aether/data/`. This means you can:

- Start a project in Claude Code, continue in OpenCode
- Switch tools when hitting rate limits
- Use Claude for orchestration, other models for bulk coding
- Mix and match based on task requirements

### OpenCode Agents

Aether includes 4 specialized OpenCode agents:

| Agent | Purpose | Temperature |
|-------|---------|-------------|
| `aether-queen` | Orchestrates phases, spawns workers | 0.3 |
| `aether-builder` | Implements code, TDD-first | 0.2 |
| `aether-scout` | Researches, gathers information | 0.4 |
| `aether-watcher` | Validates, tests, quality gates | 0.1 |

**Optional: Model-per-agent configuration**

```json
{
  "agents": {
    "aether-queen": { "model": "anthropic/claude-sonnet" },
    "aether-builder": { "model": "your-preferred/coding-model" },
    "aether-scout": { "model": "your-preferred/research-model" },
    "aether-watcher": { "model": "anthropic/claude-sonnet" }
  }
}
```

---

## Installation & Updates

```bash
# Install globally
npm install -g aether-colony

# Verify install
aether version
ls ~/.claude/commands/ant/
ls ~/.config/opencode/commands/ant/

# Verify runtime (from inside any repo)
ls .aether/

# Lint (after cloning)
npm run lint

# Update
npm update -g aether-colony

# Uninstall (preserves project state)
aether uninstall && npm uninstall -g aether-colony
```

---

## Acknowledgments

Massive shoutout to **[glittercowboy](https://github.com/glittercowboy)** and the **[GSD (Get Shit Done) system](https://github.com/glittercowboy/gsd)**. GSD showed what Claude Code could become with the right orchestration. Aether takes that inspiration and adds ant colony dynamics — pheromones, castes, and nested spawning.

---

## License

MIT — do whatever you want with it.

---

<div align="center">

*🐜 The colony is greater than the sum of its ants. 🐜*

</div>
