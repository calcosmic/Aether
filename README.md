```
     _    _____ _____ _   _ _____ ____
    / \  | ____|_   _| | | | ____|  _ \
   / _ \ |  _|   | | | |_| |  _| | |_) |
  / ___ \| |___  | | |  _  | |___|  _ <
 /_/   \_\_____| |_| |_| |_|_____|_| \_\
```

<div align="center">
  <img src="aether-logo.png" alt="Aether Logo" width="500">

  **A multi-agent system for Claude Code and OpenCode where workers spawn other workers.**

  ➡️ Click **Use this template** (top-right) to create your own Aether repo in 30 seconds.

  *Inspired by [glittercowboy's GSD system](https://github.com/glittercowboy/gsd)*

  [![npm version](https://img.shields.io/npm/v/aether-colony.svg)](https://www.npmjs.com/package/aether-colony)
  [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

  **v1.0.0** — First stable production release 🎉
</div>

---

> *"The whole is greater than the sum of its parts."* — Aristotle

---

---

## 🐜 What Is Aether?

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
   └── 🏛️ Architects — extract patterns
```

When a Builder hits something complex, it spawns a Scout to research. When code is written, a Watcher spawns to verify. **The colony adapts to the problem.**

---

## 🚀 Quick Start

### Prerequisites

- [Claude Code](https://claude.ai/code) (Anthropic's CLI) and/or [OpenCode](https://github.com/sst/opencode)
- Node.js >= 16
- `jq` — `brew install jq` on macOS

### Installation

```bash
npm install -g aether-colony
```

This installs:
- 📁 **Claude Code Commands** → `~/.claude/commands/ant/` (20 slash commands)
- 📁 **OpenCode Commands** → `~/.config/opencode/commands/ant/` (20 slash commands)
- 📁 **OpenCode Agents** → `~/.config/opencode/agents/` (4 specialized agents)
- 📁 **Runtime** → `~/.aether/` (worker specs, utilities)

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

## 🎯 Commands

### Core Workflow

| Command | Purpose |
|---------|---------|
| `/ant:init "goal"` | 🟢 Set colony mission |
| `/ant:plan` | 🗺️ Generate phased roadmap |
| `/ant:build N` | 🔨 Execute phase N |
| `/ant:continue` | ▶️ Review & advance to next phase |

**`/ant:init "goal"`** — Initialize the colony with your project goal. This creates the colony state, sets up constraints, and prepares the system. Run this first in any new project.
```bash
/ant:init "Build a REST API with JWT authentication"
```

**`/ant:plan`** — Generate a phased project plan. The colony iterates (up to 50 times) with Scout research and Route-Setter planning until confidence reaches 95%. Includes anti-stuck detection that pauses for user input if progress stalls.

**`/ant:build N`** — Execute phase N of your plan. Spawns a Prime Worker who coordinates Builders, Watchers, and Scouts. Workers can spawn sub-workers up to depth 3. Each phase ends with Watcher verification.
```bash
/ant:build 1    # Build phase 1
/ant:build 3    # Build phase 3
```

**`/ant:continue`** — After a build completes, this reviews the work through 6 verification gates (build, types, lint, tests, security, diff), then advances to the next phase. Use `--all` to auto-continue through all phases with quality gates.

---

### Pheromone Signals

| Command | Purpose |
|---------|---------|
| `/ant:focus "area"` | 🎯 Guide colony attention |
| `/ant:redirect "pattern"` | 🚫 Warn away from approaches |
| `/ant:feedback "msg"` | 💬 Teach preferences |

**`/ant:focus "area"`** — Tell the colony to pay special attention to something. Max 5 focus areas at a time. Workers prioritize these during their work.
```bash
/ant:focus "error handling in the auth module"
/ant:focus "performance optimization"
```

**`/ant:redirect "pattern"`** — Warn the colony away from specific approaches or patterns. Max 10 redirects. These act as hard constraints workers must respect.
```bash
/ant:redirect "don't use jsonwebtoken, use jose instead"
/ant:redirect "avoid synchronous file operations"
```

**`/ant:feedback "msg"`** — Teach the colony your preferences. Creates an instinct (learned behavior) that persists across phases. Good for style preferences or project-specific patterns.
```bash
/ant:feedback "prefer composition over inheritance"
/ant:feedback "always add JSDoc comments to public functions"
```

---

### Power Commands

| Command | Purpose |
|---------|---------|
| `/ant:council` | 📜🐜🏛️🐜📜 Clarify intent via multi-choice |
| `/ant:swarm "problem"` | 🔥🐜🗡️🐜🔥 Stubborn bug destroyer |

**`/ant:council`** 📜🐜🏛️🐜📜 — Convene the council when you need to clarify your intent through guided multi-choice questions. Works anytime, even mid-build. The council asks about project direction, quality priorities, or constraints, then auto-injects the appropriate FOCUS/REDIRECT/FEEDBACK signals based on your answers.
```bash
/ant:council    # Opens interactive clarification session
```
Use this when:
- Starting a complex project and want to set clear direction
- The colony seems confused about your preferences
- You want to inject multiple constraints at once

**`/ant:swarm "problem"`** 🔥🐜🗡️🐜🔥 — The nuclear option for stubborn bugs. Deploys 4 parallel scouts to investigate from different angles, then applies the best fix automatically. Use when you've tried fixing something multiple times and it keeps failing.
```bash
/ant:swarm "Tests keep failing in auth module with undefined error"
/ant:swarm "API returns 500 but logs show no errors"
```
The 4 scouts:
- 🏛️ **Git Archaeologist** — Traces git history to find when it broke
- 🔍 **Pattern Hunter** — Finds similar working code in the codebase
- 💥 **Error Analyst** — Parses error chains to identify root cause
- 🌐 **Web Researcher** — Searches docs/issues for known solutions

After investigation, swarm cross-compares findings, ranks solutions by confidence, creates a git checkpoint, applies the fix, and verifies it works. If it fails, it rolls back automatically.

---

### Issue Tracking

| Command | Purpose |
|---------|---------|
| `/ant:flag "issue"` | 🚩 Create a flag |
| `/ant:flags` | 📋 List all flags |

**`/ant:flag "issue"`** — Create a flag to track blockers, issues, or notes. Flags persist across context resets and can block phase advancement.
```bash
/ant:flag "Database migration needs manual review" --type blocker
/ant:flag "Consider adding rate limiting" --type issue
/ant:flag "Good pattern for error handling" --type note
```
Types:
- `blocker` — Blocks phase advancement until resolved
- `issue` — Warning that should be addressed
- `note` — Informational, no action required

**`/ant:flags`** — List all flags, resolve them, or acknowledge issues.
```bash
/ant:flags                    # List all active flags
/ant:flags --all              # Include resolved flags
/ant:flags --resolve 3 "Fixed in commit abc123"
```

---

### Visibility & Status

| Command | Purpose |
|---------|---------|
| `/ant:status` | 📊 Colony overview |
| `/ant:phase N` | 📋 View phase details |
| `/ant:watch` | 👁️ Live tmux monitoring |

**`/ant:status`** — Quick overview of colony state: current phase, confidence, active constraints, recent activity, and any flags.

**`/ant:phase N`** — View details of a specific phase including tasks, status, and which worker castes are assigned.
```bash
/ant:phase 2    # View phase 2 details
```

**`/ant:watch`** — Set up a tmux session with 4 panes showing real-time colony activity: status, progress bar, spawn tree visualization, and activity log stream. Requires tmux.

---

### Codebase Analysis

| Command | Purpose |
|---------|---------|
| `/ant:colonize` | 🔍 Analyze existing codebase |
| `/ant:organize` | 🧹 Codebase hygiene report |

**`/ant:colonize`** — Analyze an existing codebase before planning. Scans key files (package.json, README, entry points, configs) and creates a codebase map. Run this before `/ant:plan` when working with existing projects.

**`/ant:organize`** — Run a hygiene report on the codebase. An Architect ant scans for stale files, dead code, orphaned configs, and other cleanup opportunities. Report-only — doesn't modify files.

---

### Session Management

| Command | Purpose |
|---------|---------|
| `/ant:pause-colony` | ⏸️ Save state for break |
| `/ant:resume-colony` | ▶️ Restore from pause |

**`/ant:pause-colony`** — Save full colony state to a handoff document when you need to take a break or switch contexts. Creates `.aether/HANDOFF.md` with everything needed to resume.

**`/ant:resume-colony`** — Restore colony state from a previous pause. Reads the handoff document and gets you back to where you left off.

---

## ✨ Features

- 🐜 **Nested Spawning** — Workers spawn sub-workers (depth 1→2→3 chains)
- 🎨 **Colorized Output** — Each caste has its own terminal color
- 👁️ **Runtime Verification** — Watchers actually execute code, not just read it
- 🚩 **Flagging System** — Issues persist across context resets
- 🔨 **Named Ants** — Hammer-42, Vigil-17, Quest-33... they feel real
- 📊 **Spawn Tree Visualization** — See the colony hierarchy in real-time

---

## 🏗️ How It Works

### The Castes

| Caste | Emoji | Role |
|-------|-------|------|
| **Builder** | 🔨 | Writes code, runs commands |
| **Watcher** | 👁️ | Tests, validates, quality gates |
| **Scout** | 🔍 | Researches docs, finds answers |
| **Colonizer** | 🗺️ | Explores codebases, maps structure |
| **Route-setter** | 📋 | Plans phases, breaks down goals |
| **Architect** | 🏛️ | Synthesizes patterns, extracts learnings |

### Pheromone Signals

Instead of direct commands, you emit signals that the colony interprets:

| Signal | Purpose | Decay |
|--------|---------|-------|
| 🎯 `FOCUS` | "Pay attention to this" | 1 hour |
| 🚫 `REDIRECT` | "Avoid this approach" | 24 hours |
| 💬 `FEEDBACK` | "Here's what I like/dislike" | 6 hours |

Each caste has different sensitivity to signals. Builders prioritize FOCUS, Watchers prioritize REDIRECT warnings.

### Iterative Planning (95% Confidence)

When you run `/ant:plan`, the colony doesn't just generate a plan once. It iterates:

1. **Scout** researches the codebase, identifies knowledge gaps
2. **Route-Setter** drafts/refines the plan based on findings
3. **Loop** continues until confidence reaches 95% (max 50 iterations)

```
Iteration 12/50 | Confidence: 78%
├── Researching: API authentication patterns
└── Gaps remaining: 2 (rate limiting, error handling)
```

Confidence is measured across 5 dimensions: codebase knowledge, requirement clarity, risk identification, dependencies, and effort estimation. The loop includes anti-stuck checks — if progress stalls, it pauses for user input rather than spinning.

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

---

## 📁 File Structure

```
~/.claude/commands/ant/           # Claude Code global slash commands
    ├── init.md, plan.md, build.md, continue.md...
    └── (20 command files)

~/.config/opencode/               # OpenCode global config
    ├── commands/ant/             # OpenCode slash commands (20 files)
    └── agents/                   # Specialized agents (queen, builder, scout, watcher)

~/.aether/                        # Global runtime (shared)
    ├── workers.md                # Worker specs with spawn protocol
    ├── aether-utils.sh           # Utility layer (34 subcommands)
    └── utils/                    # Colorization, spawn tree viz

<your-repo>/.aether/data/         # Per-project state (SHARED between tools)
    ├── COLONY_STATE.json         # Goal, plan, memory, errors
    ├── flags.json                # Blockers, issues, notes
    ├── activity.log              # Worker activity stream
    └── spawn-tree.txt            # Spawn hierarchy
```

### Cross-Tool Compatibility

Both Claude Code and OpenCode share the same state files in `.aether/data/`. This means you can:

- Start a project in Claude Code, continue in OpenCode
- Switch tools when hitting rate limits
- Use Claude for orchestration, GLM-4.7 for bulk coding
- Mix and match based on task requirements

---

## 🔄 Typical Workflow

```
1. /ant:init "Build feature X"     # Set the goal
2. /ant:colonize                    # Analyze existing code (optional)
3. /ant:plan                        # Colony generates phases
4. /ant:focus "security"            # Guide attention (optional)
5. /ant:build 1                     # Execute phase 1
6. /ant:continue                    # Review, advance
7. /ant:build 2                     # Repeat until done
```

Or use auto-continue:

```bash
/ant:continue --all    # Runs all phases with quality gates
```

Auto-continue halts if a Watcher scores below 4/10 or after 2 consecutive failures.

---

## 🔧 Installation & Updates

```bash
# Install globally
npm install -g aether-colony

# Verify
aether version
ls ~/.claude/commands/ant/           # Claude Code commands
ls ~/.config/opencode/commands/ant/  # OpenCode commands

# Update
npm update -g aether-colony

# Uninstall (preserves project state)
aether uninstall && npm uninstall -g aether-colony
```

### OpenCode Setup

OpenCode uses whatever model you have configured as your default. The included agents work with any provider.

**Optional: Model-per-agent configuration**

For advanced users who want different models for different castes, add to your `opencode.json`:

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

This is entirely optional - by default, all agents use your configured default model.

---

## 🙏 Acknowledgments

Massive shoutout to **[glittercowboy](https://github.com/glittercowboy)** and the **[GSD (Get Shit Done) system](https://github.com/glittercowboy/gsd)**. GSD showed what Claude Code could become with the right orchestration. Aether takes that inspiration and adds ant colony dynamics — pheromones, castes, and nested spawning.

---

## 📄 License

MIT — do whatever you want with it.

---

<div align="center">

*🐜 The colony is greater than the sum of its ants. 🐜*

</div>
