<div align="center">

# 🐜 Aether Colony

**Multi-agent AI development for Claude Code and OpenCode**

[![npm version](https://img.shields.io/npm/v/aether-colony.svg?style=flat-square)](https://www.npmjs.com/package/aether-colony)
[![npm downloads](https://img.shields.io/npm/dw/aether-colony.svg?style=flat-square)](https://www.npmjs.com/package/aether-colony)
[![License: MIT](https://img.shields.io/github/license/calcosmic/Aether.svg?style=flat-square)](LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/calcosmic/Aether.svg?style=flat-square)](https://github.com/calcosmic/Aether/stargazers)
[![Sponsor](https://img.shields.io/badge/Sponsor-GitHub-%23ea4aaa.svg?style=flat-square&logo=github)](https://github.com/sponsors/calcosmic)

Spawn a colony of 24 AI specialists that self-organize around your goal using pheromone signals.
1,000+ weekly installs.

*Artificial Ecology For Thought and Emergent Reasoning*

*The whole is greater than the sum of its ants.*

```bash
npm install -g aether-colony
```

</div>

<p align="center">
  <img src="./AetherAnts.png" alt="Aether Colony" width="640" />
</p>

```
         👑 Queen (you)
          │
          │  set the goal, steer with pheromone signals
          ▼
    ┌─────────────────────────────────────────┐
    │         Colony self-organizes            │
    │                                         │
    │  🔨 Builders      write code (TDD)      │
    │  👁️ Watchers      verify & test          │
    │  🔍 Scouts        research first         │
    │  🐛 Trackers      investigate bugs       │
    │  🗺️ Colonizers    explore codebases      │
    │  📋 Route-setters plan phases            │
    │  🏺 Archaeologists excavate git history  │
    │  🎲 Chaos Ants    resilience testing     │
    │  📚 Keepers       preserve knowledge     │
    │  🔮 Oracle        deep research          │
    │  ...and 24 specialists total             │
    └─────────────────────────────────────────┘
```

## 🐜 The Problem

AI coding assistants work sequentially — one agent does everything: research, code, test, review. When it hits something complex, it either guesses or asks you. There's no specialization, no parallel work, no memory across sessions.

## 🐜 The Solution

Aether brings **ant colony intelligence** to AI-assisted development. Instead of one AI doing everything, you get a colony of specialists that self-organize around your goal.

Workers spawn workers dynamically (max depth 3, max 10 per phase). When a Builder hits something complex, it spawns a Scout to research. When code is written, a Watcher spawns to verify. The colony adapts to the problem.

You steer the colony with **pheromone signals**, not micromanagement:

```
/ant:focus "security"              # 🎯 "Pay attention here"
/ant:redirect "no jQuery"          # 🚫 "Don't do this" (hard constraint)
/ant:feedback "prefer composition" # 💬 "Adjust based on this"
```

The colony **remembers**. Wisdom, learnings, and instincts persist across sessions. The 🧠 Hive Brain shares knowledge across colonies on your machine.

## 🚀 Quick Start

```bash
# Install globally
npm install -g aether-colony

# In your project repo:
/ant:lay-eggs            # 🥚 Set up Aether (one-time)
/ant:init "Build X"      # 🌱 Start a colony with a goal
/ant:plan                # 📋 Generate phased roadmap
/ant:run                 # 🐜 Autopilot: build, verify, advance all phases
/ant:seal                # 🏺 Done — archive the colony
```

That's it. Five commands from zero to shipped.

## ✨ Key Features

- 🐜 **24 Specialized Agents** — Real subagents spawned via Task tool, from builders to archaeologists
- ⚡ **44 Slash Commands** — Full lifecycle management across Claude Code and OpenCode
- 🎯 **Pheromone System** — Guide the colony with FOCUS, REDIRECT, FEEDBACK signals
- 🧠 **Colony Memory** — Learnings persist across sessions via QUEEN.md wisdom
- 🌐 **Hive Brain** — Cross-colony wisdom sharing with domain-scoped retrieval
- 📚 **Skills System** — 28 skills (10 colony + 18 domain) inject domain knowledge into workers
- 🤖 **Autopilot** (`/ant:run`) — Automated build-verify-advance loop with smart pause conditions
- ✅ **6-Phase Verification** — Build, types, lint, tests, security, diff gates before any phase advances
- 🛡️ **Quality Gates** — Security (Gatekeeper), quality (Auditor), coverage (Probe), performance (Measurer)
- 🔍 **Per-Phase Research** — Scouts investigate domain knowledge before task decomposition
- 🔮 **Oracle Deep Research** — Autonomous research loop for complex investigations
- 💾 **Pause/Resume** — Full state serialization for context breaks

## 📖 Commands

<details>
<summary><strong>🏗️ Core Lifecycle</strong></summary>

| Command | Description |
|---------|-------------|
| `/ant:lay-eggs` | 🥚 Set up Aether in this repo (one-time) |
| `/ant:init "goal"` | 🌱 Initialize colony with mission |
| `/ant:plan` | 📋 Generate phased roadmap with domain research |
| `/ant:build N` | 🔨 Execute phase N with worker waves |
| `/ant:continue` | ➡️ 6-phase verification, advance to next phase |
| `/ant:run` | 🐜 Autopilot — build, verify, advance automatically |
| `/ant:patrol` | 🔍 Pre-seal audit — verify work against plan |
| `/ant:seal` | 🏺 Complete and archive colony |
| `/ant:entomb` | ⚰️ Create chamber from completed colony |
| `/ant:pause-colony` | 💾 Save state for context break |
| `/ant:resume-colony` | 🚦 Restore from pause |

</details>

<details>
<summary><strong>🎯 Pheromone Signals</strong></summary>

| Command | Description |
|---------|-------------|
| `/ant:focus "area"` | 🎯 FOCUS — "Pay attention here" |
| `/ant:redirect "pattern"` | 🚫 REDIRECT — "Don't do this" (hard constraint) |
| `/ant:feedback "note"` | 💬 FEEDBACK — "Adjust based on this" |
| `/ant:pheromones` | 📊 View active signals |
| `/ant:export-signals` | 📤 Export signals to XML |
| `/ant:import-signals` | 📥 Import signals from XML |

</details>

<details>
<summary><strong>🔬 Research & Analysis</strong></summary>

| Command | Description |
|---------|-------------|
| `/ant:colonize` | 📊🗺️ 4 parallel scouts analyze your codebase |
| `/ant:oracle "topic"` | 🔮 Deep research with autonomous loop |
| `/ant:archaeology <path>` | 🏺 Excavate git history for any file |
| `/ant:chaos <target>` | 🎲 Resilience testing, edge case probing |
| `/ant:swarm "problem"` | 🔥 4 parallel scouts for stubborn bugs |
| `/ant:dream` | 💭 Philosophical codebase wanderer |
| `/ant:interpret` | 🔍 Grounds dreams in reality |
| `/ant:organize` | 🧹 Codebase hygiene report |

</details>

<details>
<summary><strong>👁️ Visibility & Status</strong></summary>

| Command | Description |
|---------|-------------|
| `/ant:status` | 📈 Colony overview with memory health |
| `/ant:memory-details` | 🧠 Wisdom, pending promotions, recent failures |
| `/ant:watch` | 👁️ Real-time swarm display |
| `/ant:history` | 📜 Recent activity log |
| `/ant:flags` | 🚩 List blockers and issues |
| `/ant:help` | 🐜 Full command reference |

</details>

<details>
<summary><strong>🔧 Coordination & Maintenance</strong></summary>

| Command | Description |
|---------|-------------|
| `/ant:council` | 📜 Clarify intent via multi-choice questions |
| `/ant:flag "title"` | 🚩 Create project-specific flag |
| `/ant:data-clean` | 🧹 Remove test artifacts from colony data |
| `/ant:preferences` | ⚙️ Add or list user preferences |
| `/ant:skill-create "topic"` | 🐜 Create custom domain skill |
| `/ant:update` | 🔄 Update system files from hub |

</details>

## 🐜 The 24 Agents

| Tier | Agent | Role |
|------|-------|------|
| 👑 **Core** | Builder | 🔨 Writes code, TDD-first |
| 👑 **Core** | Watcher | 👁️ Tests, validates, quality gates |
| 👑 **Core** | Scout | 🔍 Researches, discovers |
| 🏛️ **Orchestration** | Queen | 👑 Orchestrates phases, spawns workers |
| 🏛️ **Orchestration** | Route-Setter | 📋 Plans phases, breaks down goals |
| 🏛️ **Orchestration** | Architect | 🏗️ Architecture design |
| 🗺️ **Surveyor** | surveyor-nest | 📂 Maps directory structure |
| 🗺️ **Surveyor** | surveyor-disciplines | 📏 Documents conventions |
| 🗺️ **Surveyor** | surveyor-pathogens | 🦠 Identifies tech debt |
| 🗺️ **Surveyor** | surveyor-provisions | 📦 Maps dependencies |
| ⚡ **Specialist** | Keeper | 📚 Preserves knowledge |
| ⚡ **Specialist** | Tracker | 🐛 Investigates bugs |
| ⚡ **Specialist** | Probe | 🔬 Coverage analysis |
| ⚡ **Specialist** | Weaver | 🧵 Refactoring specialist |
| ⚡ **Specialist** | Auditor | ✅ Quality gate |
| 🎯 **Niche** | Chaos | 🎲 Resilience testing |
| 🎯 **Niche** | Archaeologist | 🏺 Excavates git history |
| 🎯 **Niche** | Gatekeeper | 🛡️ Security gate |
| 🎯 **Niche** | Includer | ♿ Accessibility audits |
| 🎯 **Niche** | Measurer | ⏱️ Performance analysis |
| 🎯 **Niche** | Sage | 🧙 Wisdom synthesis |
| 🎯 **Niche** | Oracle | 🔮 Deep research |
| 🎯 **Niche** | Ambassador | 🌐 External integrations |
| 🎯 **Niche** | Chronicler | 📝 Documentation |

## 🏗️ Architecture

```
.aether/                      # 🐜 Colony files (repo-local)
├── aether-utils.sh           # ⚡ Dispatcher (~5,200 lines, ~150 subcommands)
├── utils/                    # 🔧 ~29 modular scripts
├── skills/                   # 📚 28 skills (10 colony + 18 domain)
├── commands/                 # 📖 44 YAML command sources
├── exchange/                 # 📤 XML exchange modules
├── docs/                     # 📝 Documentation
├── templates/                # 📋 12 templates
└── data/                     # 💾 Colony state (local only)

~/.aether/                    # 🌐 Hub (cross-colony, user-level)
├── QUEEN.md                  # 👑 Wisdom + preferences
├── hive/wisdom.json          # 🧠 Cross-colony wisdom (200 cap)
└── registry.json             # 📊 All registered colonies
```

## 🔌 Works With

- **[Claude Code](https://docs.anthropic.com/en/docs/claude-code)** — 44 slash commands + 24 agent definitions
- **[OpenCode](https://github.com/opencode-ai/opencode)** — 44 slash commands + agent definitions

## ❤️ Support Aether

If Aether has been useful to you, here's how you can keep the colony alive:

**[Sponsor on GitHub](https://github.com/sponsors/calcosmic)** (preferred)

<details>
<summary>💡 Crypto — no middlemen, no fees</summary>

| Network | Address |
|---------|---------|
| **ETH** (MetaMask) | `0xE7F8C9BE190c207D49DF01b82747cf7B6Bd1c809` |
| **SOL** (Phantom) | `6DVTdoZvvi9siUpgmRJZxk5Kqho8TZiN2ZzyVUVC9gX8` |

</details>

[PayPal](https://www.paypal.com/ncp/payment/RENG7ZMW5F59L) · [Buy Me a Coffee](https://buymeacoffee.com/music5y)

*Your support funds servers, new features, docs, and keeps Aether free & open source (MIT).*

## 📄 License

MIT
