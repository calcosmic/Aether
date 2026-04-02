<div align="center">

<img src="./AetherBanner.png" alt="Aether" width="100%" />

# Aether

**Artificial Ecology for Thought and Emergent Reasoning**

<br>

[![npm version](https://img.shields.io/npm/v/aether-colony.svg?style=flat-square)](https://www.npmjs.com/package/aether-colony?utm_source=github&utm_medium=readme&utm_campaign=aether)
[![npm downloads](https://img.shields.io/npm/dw/aether-colony.svg?style=flat-square)](https://www.npmjs.com/package/aether-colony?utm_source=github&utm_medium=readme&utm_campaign=aether)
[![License: MIT](https://img.shields.io/github/license/calcosmic/Aether.svg?style=flat-square)](LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/calcosmic/Aether.svg?style=flat-square)](https://github.com/calcosmic/Aether/stargazers)
[![Sponsor](https://img.shields.io/badge/Sponsor-GitHub-%23ea4aaa.svg?style=flat-square&logo=github)](https://github.com/sponsors/calcosmic?utm_source=github&utm_medium=readme&utm_campaign=aether)

[![weekly downloads](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/calcosmic/Aether/main/docs/badges/weekly-downloads.json&style=flat-square)](https://www.npmjs.com/package/aether-colony?utm_source=github&utm_medium=readme&utm_campaign=aether)
[![monthly downloads](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/calcosmic/Aether/main/docs/badges/monthly-downloads.json&style=flat-square)](https://www.npmjs.com/package/aether-colony?utm_source=github&utm_medium=readme&utm_campaign=aether)
[![total downloads](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/calcosmic/Aether/main/docs/badges/total-downloads.json&style=flat-square)](https://www.npmjs.com/package/aether-colony?utm_source=github&utm_medium=readme&utm_campaign=aether)
[![bundle size](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/calcosmic/Aether/main/docs/badges/bundle-size.json&style=flat-square)](https://www.npmjs.com/package/aether-colony?utm_source=github&utm_medium=readme&utm_campaign=aether)
![test count](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/calcosmic/Aether/main/docs/badges/test-count.json&style=flat-square)
![commands](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/calcosmic/Aether/main/docs/badges/commands.json&style=flat-square)
![agents](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/calcosmic/Aether/main/docs/badges/agents.json&style=flat-square)
[![colony version](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/calcosmic/Aether/main/docs/badges/colony-version.json&style=flat-square)](https://github.com/calcosmic/Aether/releases)

<br>

```
$ npx aether-colony
```

<br>

*The whole is greater than the sum of its ants.* 🐜

<br>

[![🐜 aetherantcolony.com](https://img.shields.io/badge/%F0%9F%90%9C_aetherantcolony.com-7B3FE4?style=for-the-badge&logoColor=white)](https://aetherantcolony.com?utm_source=github&utm_medium=readme&utm_campaign=aether)

</div>

---

## Why Aether

Every AI coding tool now has "agents." Most of them are the same thing repackaged — a loop that plans, executes, and checks. That's not a colony. That's one ant doing laps.

Aether is different because it's modeled on how **real ant colonies work**: no central brain, no single agent trying to be everything. Instead, 24 specialized workers self-organize around your goal. A Builder writes code. When it hits something unfamiliar, it doesn't guess — it spawns a Scout to research. When code lands, a Watcher spawns to verify. A Tracker hunts bugs. An Archaeologist digs through git history. They work in parallel, in waves, across phases.

**What actually makes this different:**

🐜 **Pheromone signals, not prompt engineering.** You steer the colony the way a queen steers ants — with signals, not instructions. Drop a FOCUS signal on "security" and every worker weighs it. Drop a REDIRECT on "no raw SQL" and it becomes a hard constraint. The colony adapts without you rewriting prompts.

🐜 **Memory that compounds.** Learnings from one build become instincts for the next. Instincts promote to QUEEN.md wisdom. High-confidence wisdom flows to the Hive Brain and crosses into other projects on your machine. Your colonies get smarter over time.

🐜 **Built to be fun.** You set a goal, the colony builds it. You steer with signals, not micromanagement. You watch workers spawn workers. You seal the colony when it's done and start a new one. It feels like tending a garden, not writing tickets.

```
/ant:focus "security"              # 🐜 "Pay attention here"
/ant:redirect "no jQuery"          # 🐜 "Don't do this"
/ant:feedback "prefer composition" # 🐜 "Adjust based on this"
```

## Quick Start

```bash
# Interactive setup (recommended)
npx aether-colony

# Or install globally
npm install -g aether-colony
```

Then in your project:

```
/ant:lay-eggs            # Set up Aether (one-time)
/ant:init "Build X"      # Start a colony with a goal
/ant:plan                # Generate phased roadmap
/ant:run                 # Autopilot: build, verify, advance all phases
/ant:seal                # Done — archive the colony
```

Five commands from zero to shipped.

## Key Features

| | Feature | Description |
|---|---------|-------------|
| **Agents** | 24 Specialized Workers | Builders, Watchers, Scouts, Trackers, Archaeologists, Oracle, and more |
| **Commands** | 45 Slash Commands | Full lifecycle management across Claude Code and OpenCode |
| **Signals** | Pheromone System | Guide the colony with FOCUS, REDIRECT, and FEEDBACK signals |
| **Memory** | Colony Wisdom | Learnings and instincts persist across sessions via QUEEN.md |
| **Sharing** | Hive Brain | Cross-colony wisdom sharing with domain-scoped retrieval |
| **Skills** | 28 Skills | 10 colony + 18 domain skills inject knowledge into workers |
| **Autopilot** | `/ant:run` | Automated build-verify-advance loop with smart pause conditions |
| **Verification** | 6-Phase Gates | Build, types, lint, tests, security, diff gates before advancing |
| **Quality** | 4 Gate Agents | Security (Gatekeeper), quality (Auditor), coverage (Probe), performance (Measurer) |
| **Research** | Oracle + Scouts | Deep autonomous research before task decomposition |
| **State** | Pause/Resume | Full state serialization for context breaks |

## Commands

<details>
<summary><strong>Core Lifecycle</strong></summary>

| Command | Description |
|---------|-------------|
| `/ant:lay-eggs` | Set up Aether in this repo (one-time) |
| `/ant:init "goal"` | Initialize colony with mission |
| `/ant:plan` | Generate phased roadmap with domain research |
| `/ant:build N` | Execute phase N with worker waves |
| `/ant:continue` | 6-phase verification, advance to next phase |
| `/ant:run` | Autopilot — build, verify, advance automatically |
| `/ant:patrol` | Pre-seal audit — verify work against plan |
| `/ant:seal` | Complete and archive colony |
| `/ant:entomb` | Create chamber from completed colony |
| `/ant:pause-colony` | Save state for context break |
| `/ant:resume-colony` | Restore from pause |

</details>

<details>
<summary><strong>Pheromone Signals</strong></summary>

| Command | Description |
|---------|-------------|
| `/ant:focus "area"` | FOCUS — "Pay attention here" |
| `/ant:redirect "pattern"` | REDIRECT — "Don't do this" (hard constraint) |
| `/ant:feedback "note"` | FEEDBACK — "Adjust based on this" |
| `/ant:pheromones` | View active signals |
| `/ant:export-signals` | Export signals to XML |
| `/ant:import-signals` | Import signals from XML |

</details>

<details>
<summary><strong>Research & Analysis</strong></summary>

| Command | Description |
|---------|-------------|
| `/ant:colonize` | 4 parallel scouts analyze your codebase |
| `/ant:oracle "topic"` | Deep research with autonomous loop |
| `/ant:archaeology <path>` | Excavate git history for any file |
| `/ant:chaos <target>` | Resilience testing, edge case probing |
| `/ant:swarm "problem"` | 4 parallel scouts for stubborn bugs |
| `/ant:dream` | Philosophical codebase wanderer |
| `/ant:interpret` | Grounds dreams in reality |
| `/ant:organize` | Codebase hygiene report |

</details>

<details>
<summary><strong>Visibility & Status</strong></summary>

| Command | Description |
|---------|-------------|
| `/ant:status` | Colony overview with memory health |
| `/ant:memory-details` | Wisdom, pending promotions, recent failures |
| `/ant:watch` | Real-time swarm display |
| `/ant:history` | Recent activity log |
| `/ant:flags` | List blockers and issues |
| `/ant:help` | Full command reference |

</details>

<details>
<summary><strong>Coordination & Maintenance</strong></summary>

| Command | Description |
|---------|-------------|
| `/ant:council` | Clarify intent via multi-choice questions |
| `/ant:flag "title"` | Create project-specific flag |
| `/ant:data-clean` | Remove test artifacts from colony data |
| `/ant:preferences` | Add or list user preferences |
| `/ant:skill-create "topic"` | Create custom domain skill |
| `/ant:update` | Update system files from hub |

</details>

## The 24 Agents

| Tier | Agent | Role |
|------|-------|------|
| **Core** | Builder | Writes code, TDD-first |
| **Core** | Watcher | Tests, validates, quality gates |
| **Core** | Scout | Researches, discovers |
| **Orchestration** | Queen | Orchestrates phases, spawns workers |
| **Orchestration** | Route-Setter | Plans phases, breaks down goals |
| **Orchestration** | Architect | Architecture design |
| **Surveyor** | surveyor-nest | Maps directory structure |
| **Surveyor** | surveyor-disciplines | Documents conventions |
| **Surveyor** | surveyor-pathogens | Identifies tech debt |
| **Surveyor** | surveyor-provisions | Maps dependencies |
| **Specialist** | Keeper | Preserves knowledge |
| **Specialist** | Tracker | Investigates bugs |
| **Specialist** | Probe | Coverage analysis |
| **Specialist** | Weaver | Refactoring specialist |
| **Specialist** | Auditor | Quality gate |
| **Niche** | Chaos | Resilience testing |
| **Niche** | Archaeologist | Excavates git history |
| **Niche** | Gatekeeper | Security gate |
| **Niche** | Includer | Accessibility audits |
| **Niche** | Measurer | Performance analysis |
| **Niche** | Sage | Wisdom synthesis |
| **Niche** | Oracle | Deep research |
| **Niche** | Ambassador | External integrations |
| **Niche** | Chronicler | Documentation |

## Architecture

```
.aether/                        Colony files (repo-local)
├── aether-utils.sh             Dispatcher (~5,500 lines, ~130+ subcommands)
├── utils/                      35 modular scripts
├── skills/                     28 skills (10 colony + 18 domain)
├── commands/                   45 YAML command sources
├── exchange/                   XML exchange modules
├── docs/                       Documentation
├── templates/                  12 templates
└── data/                       Colony state (local only)

~/.aether/                      Hub (cross-colony, user-level)
├── QUEEN.md                    Wisdom + preferences
├── hive/wisdom.json            Cross-colony wisdom (200 cap)
└── registry.json               All registered colonies
```

## Works With

- **[Claude Code](https://docs.anthropic.com/en/docs/claude-code?utm_source=github&utm_medium=readme&utm_campaign=aether)** — 45 slash commands + 24 agent definitions
- **[OpenCode](https://github.com/opencode-ai/opencode?utm_source=github&utm_medium=readme&utm_campaign=aether)** — 45 slash commands + agent definitions

## Support

If Aether has been useful to you:

**[Sponsor on GitHub](https://github.com/sponsors/calcosmic?utm_source=github&utm_medium=readme&utm_campaign=aether)**

<details>
<summary>Crypto</summary>

| Network | Address |
|---------|---------|
| **ETH** | `0xE7F8C9BE190c207D49DF01b82747cf7B6Bd1c809` |
| **SOL** | `6DVTdoZvvi9siUpgmRJZxk5Kqho8TZiN2ZzyVUVC9gX8` |

</details>

[PayPal](https://www.paypal.com/ncp/payment/RENG7ZMW5F59L?utm_source=github&utm_medium=readme&utm_campaign=aether) · [Buy Me a Coffee](https://buymeacoffee.com/music5y?utm_source=github&utm_medium=readme&utm_campaign=aether)

## License

MIT
