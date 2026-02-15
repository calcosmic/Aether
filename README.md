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

  **v3.1.13** — Production ready with model routing
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

- **Model-Aware Routing** — Different castes use different AI models optimized for their tasks
- **33 Slash Commands** — Lifecycle, management, research, and utility commands
- **4 OpenCode Agents** — Specialized agents for different platforms
- **6-Phase Verification** — Build, types, lint, tests, security, diff before advancing
- **Colony Memory** — Learnings and instincts persist across sessions
- **Pause/Resume** — Full state serialization for context breaks
- **Oracle Deep Research** — 50+ iteration autonomous research loop (RALF pattern)
- **Multi-Agent Surveys** — 4 parallel scouts for codebase analysis

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

## Complete Command Reference (33 Commands)

### 🌱 Core Lifecycle Commands

| Command | Emoji | Description |
|---------|-------|-------------|
| `/ant:init "goal"` | 🌱 | Initialize colony with mission |
| `/ant:plan` | 📝 | Generate phased roadmap (50-iteration research loop) |
| `/ant:build N` | 🔨 | Execute phase N with worker waves |
| `/ant:continue` | ➡️ | 6-phase verification, then advance to next phase |
| `/ant:pause-colony` | 💾 | Save state for context break |
| `/ant:resume-colony` | ▶️ | Restore from pause |
| `/ant:lay-eggs "new goal"` | 🥚 | Start fresh colony (preserves instincts) |
| `/ant:seal` | 🏺 | Complete and archive colony |
| `/ant:entomb` | ⚰️ | Create chamber from completed colony |

**Core Lifecycle Flow:**
```
/ant:init → /ant:plan → /ant:build 1 → /ant:continue → /ant:build 2 → ... → /ant:seal → /ant:entomb
```

### 📊 Research & Analysis Commands

| Command | Emoji | Description |
|---------|-------|-------------|
| `/ant:colonize` | 🗺️ | **Multi-agent territory survey** — 4 parallel scouts analyze your codebase and produce: `STRUCTURE.md`, `INTEGRATIONS.md`, `CONVENTIONS.md`, `ARCHITECTURE.md`, `CONCERNS.md` |
| `/ant:archaeology <path>` | 🏺 | Excavate git history for any file/directory — traces why code exists, surfaces tribal knowledge, identifies "don't touch" areas |
| `/ant:oracle ["topic"]` | 🔮 | **Deep research with RALF pattern** — 50+ iteration autonomous research loop. Use `stop` or `status` as arguments |
| `/ant:chaos <target>` | 🎲 | Resilience testing — probes edge cases, boundary conditions, finds cracks before they break |
| `/ant:swarm ["problem"]` | 🔥 | Deploy 4 parallel scouts for stubborn bugs OR view real-time swarm display |
| `/ant:dream` | 💭 | The Dreamer — philosophical codebase wanderer that observes and imagines |
| `/ant:interpret` | 🔍 | Ground dreams in reality — validates observations against actual code |
| `/ant:organize` | 🧹 | Codebase hygiene report — scans for stale files, dead code, orphaned configs |

**Research Command Details:**

#### `/ant:colonize` — Territory Survey
Dispatches 4 parallel Scout agents to analyze your codebase:
- **Scout 1**: Maps directory structure, identifies entry points, dependencies
- **Scout 2**: Maps integrations (databases, APIs, third-party services)
- **Scout 3**: Documents conventions (naming, patterns, architecture decisions)
- **Scout 4**: Identifies concerns (tech debt, risks, areas needing attention)

Produces 5 documentation files in `.aether/docs/`.

#### `/ant:oracle` — Deep Research (RALF Pattern)
The Oracle runs autonomously in a separate process using the Recursive Agent Loop Framework:
1. Configure research topic via interactive wizard
2. Oracle iterates 50+ times, accumulating knowledge
3. Each iteration reads previous progress, researches gaps
4. Produces comprehensive findings in `.aether/oracle/discoveries/`

Non-invasive: Never touches colony state, only writes to `.aether/oracle/`.

### 🧭 Planning & Coordination Commands

| Command | Emoji | Description |
|---------|-------|-------------|
| `/ant:council` | 🏛️ | Clarify intent via multi-choice questions |
| `/ant:focus "area"` | 🔦 | Emit FOCUS signal — guide colony attention |
| `/ant:redirect "pattern"` | ⚠️ | Emit REDIRECT signal — warn away from approaches |
| `/ant:feedback "msg"` | 💬 | Emit FEEDBACK signal — teach preferences |

**Pheromone Signals:**
- **FOCUS** (normal priority): "Pay attention here"
- **REDIRECT** (high priority): "Don't do this" (hard constraint)
- **FEEDBACK** (low priority): "Adjust based on this"

### 📋 Visibility & Status Commands

| Command | Emoji | Description |
|---------|-------|-------------|
| `/ant:status` | 📈 | Colony overview — current phase, progress, active workers |
| `/ant:phase N` | 📝 | View phase details — tasks, status, assignments |
| `/ant:history` | 📜 | Recent colony activity log |
| `/ant:maturity` | 👑 | View colony maturity journey with ASCII art anthill |
| `/ant:watch` | 👁️ | Set up tmux session to watch ants working in real-time |
| `/ant:tunnels [ch1] [ch2]` | 🕳️ | Explore tunnels — browse archived colonies, compare chambers |
| `/ant:flags` | 🚩 | List and manage flags (blockers, issues, notes) |
| `/ant:help` | 📖 | Full command reference |

### 🚩 Issue Tracking Commands

| Command | Emoji | Description |
|---------|-------|-------------|
| `/ant:flag "issue"` | 🚩 | Create blocker/issue/note |
| `/ant:flags` | 📋 | List and manage flags |

### ⚙️ System Commands

| Command | Emoji | Description |
|---------|-------|-------------|
| `/ant:update` | 🔄 | Sync system files from global hub |
| `/ant:verify-castes` | ✓ | Check caste model assignments and system status |
| `/ant:migrate-state` | 🚚 | One-time state migration from v1 to v2.0 format |

---

## CLI Commands

The `aether` CLI provides additional utilities:

```bash
# View version and status
aether version

# Update all registered repos
aether update --all
aether update --all --force  # Force even with dirty repos

# Manage model routing
aether caste-models list
aether caste-models set builder=kimi-k2.5
aether caste-models reset builder

# Checkpoints (safe snapshots)
aether checkpoint create "before refactor"
aether checkpoint list
aether checkpoint restore <id>
aether checkpoint verify <id>

# View telemetry
aether telemetry
aether telemetry model kimi-k2.5
aether telemetry performance

# Sync state with planning docs
aether sync-state

# Context
aether context        # Show auto-loaded context including nestmates
aether nestmates      # List sibling colonies
aether spawn-tree     # Display worker spawn tree

# Initialize in current repo
aether init --goal "My project"
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

| Caste | Emoji | Role | Model |
|-------|-------|------|-------|
| 👑 **Queen** | — | Orchestrates, spawns workers, synthesizes | glm-5 |
| 🔨 **Builder** | 🛠️ | Writes code, TDD-first | kimi-k2.5 |
| 👁️ **Watcher** | 👀 | Tests, validates, quality gates | kimi-k2.5 |
| 🔍 **Scout** | 🗺️ | Researches docs, finds answers | minimax-2.5 |
| 🗺️ **Colonizer** | 📊 | Explores codebases, maps structure | minimax-2.5 |
| 🏗️ **Architect** | 🏛️ | Synthesizes patterns, coordinates docs | glm-5 |
| 📋 **Route-Setter** | 🧭 | Plans phases, breaks down goals | kimi-k2.5 |
| 🏺 **Archaeologist** | 📜 | Excavates git history | glm-5 |
| 🔮 **Oracle** | 🔮 | Deep research, architecture analysis | minimax-2.5 |
| 🎲 **Chaos** | 🎲 | Resilience testing, adversarial probing | kimi-k2.5 |

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
    ├── QUEEN_ANT_ARCHITECTURE.md # Complete system architecture
    ├── coding-standards.md       # Coding standards reference
    ├── debugging.md              # Debugging discipline
    ├── tdd.md                    # TDD discipline
    │
    ├── docs/                     # Documentation
    │   ├── known-issues.md       # Known bugs and workarounds
    │   ├── implementation-learnings.md  # Workflow patterns
    │   ├── codebase-review.md    # Command inventory
    │   ├── planning-discipline.md # Planning guidelines
    │   └── ...
    │
    ├── utils/                    # Utility scripts
    │   ├── atomic-write.sh
    │   ├── colorize-log.sh
    │   ├── file-lock.sh
    │   ├── spawn-tree.sh
    │   └── ...
    │
    ├── oracle/                   # Oracle research infrastructure
    │   ├── oracle.sh             # RALF loop script
    │   ├── oracle.md             # Oracle agent prompt
    │   ├── research.json         # Active research config
    │   ├── progress.md           # Research progress
    │   └── discoveries/          # Research findings
    │
    ├── data/                     # Per-project state (NEVER synced)
    │   ├── COLONY_STATE.json     # Goal, plan, memory, instincts
    │   ├── flags.json            # Blockers, issues, notes
    │   ├── constraints.json      # Focus areas and redirects
    │   ├── activity.log          # Worker activity stream
    │   ├── spawn-tree.txt        # Spawn hierarchy
    │   ├── telemetry.json        # Model performance data
    │   └── completion-report.md  # End-of-project summary
    │
    ├── dreams/                   # Dream session files (NEVER synced)
    ├── checkpoints/              # Update rollback data (NEVER synced)
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

## Typical Workflows

### Starting a New Project

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

### Deep Research Workflow

```
/ant:oracle "research topic"    # Configure and launch Oracle
# Oracle runs autonomously for 50+ iterations
/ant:oracle status              # Check progress
/ant:oracle stop                # Stop if needed
# Read findings in .aether/oracle/discoveries/
```

### Codebase Analysis Workflow

```
/ant:colonize                   # 4 scouts survey territory
# Read generated docs in .aether/docs/
/ant:archaeology src/legacy/    # Excavate git history
/ant:organize                   # Hygiene report
/ant:chaos "auth module"        # Resilience test
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

Aether includes specialized OpenCode agents:

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

### Three-Tier Distribution

```
Aether Repo (.aether/)  →  Hub (~/.aether/)  →  Target Repos (.aether/)
         │                        │                        │
         │   npm install -g .     │    aether update       │
         └───────────────────────→┴───────────────────────→┘
                              (excluding user data)
```

**User data directories are NEVER synced:** `data/`, `dreams/`, `checkpoints/`, `locks/`, `temp/`

---

## Safety Features

- **File Locking** — Prevents concurrent modification of state
- **Atomic Writes** — Temp file + rename pattern
- **Update Transactions** — Two-phase commit with rollback
- **Git Checkpoints** — Automatic commits before phases
- **Ant Graveyards** — Failed files marked for future caution
- **Checkpoint System** — Safe snapshots before updates with `aether checkpoint`

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
ls ~/.aether/  # Check hub structure

# Verify runtime (from inside any repo)
ls .aether/

# Update system files in all registered repos
aether update --all
aether update --all --force  # Force even with dirty repos

# Update npm package
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
