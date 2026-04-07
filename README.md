<div align="center">

<img src="./AetherBanner.png" alt="Aether" width="100%" />

# Aether

**Artificial Ecology for Thought and Emergent Reasoning**

<br>

[![GitHub release](https://img.shields.io/github/v/release/calcosmic/Aether.svg?style=flat-square)](https://github.com/calcosmic/Aether/releases)
latest)
[![License: MIT](https://img.shields.io/github/license/calcosmic/Aether.svg?style=flat-square)](LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/calcosmic/Aether.svg?style=flat-square)](https://github.com/calcosmic/Aether/stargazers)
[![Sponsor](https://img.shields.io/badge/Sponsor-GitHub-%23ea4aaa.svg?style=flat-square&logo=github)](https://github.com/sponsors/calcosmic?utm_source=github&utm_medium=readme&utm_campaign=aether)

[![Go Report Card](https://goreportcard.com/badge/github.com/calcosmic/Aether?style=flat-square)](https://goreportcard.com/report/github.com/calcosmic/Aether)

[![Go Reference](https://pkg.go.dev/badge/github.com/calcosmic/Aether.svg)](https://pkg.go.dev/github.com/calcosmic/Aether)

<br>

*The whole is greater than the sum of its ants.*

<br>

[![aetherantcolony.com](https://img.shields.io/badge/%F0%9F%90%9C_aetherantcolony.com-7B3FE4?style=for-the-badge&logoColor=white)](https://aetherantcolony.com?utm_source=github&utm_medium=readme&utm_campaign=aether)

</div>

---

---

## Why Aether

Every AI coding tool now has "agents." Most are them are the same repackaged — a loop that plans, executes, and checks. That's not a colony. That's one ant doing laps.

 Aether is different because it'm modeled on how real **ant colonies** work**: no central brain, no single agent trying to be everything. Instead, 24 specialized workers self-organize around your goal.

 A Builder writes code. When it hits something unfamiliar, it doesn't guess — it spawns a Scout to research. When code lands, a Watcher verifies. A Tracker hunts bugs. An Archaeologist excavates git history. They work parallel, in waves, across phases.

 What makes this different: **Pheromone signals, — not prompt engineering** — Guide workers with FOCUS, REDIRECT, and FEEDBACK. The colony adapts without rewriting prompts

 **Memory that compounds** - Learnings from one build become instincts. Instincts promote to QUEEN.md wisdom. High-confidence wisdom flows to the Hive Brain and crosses to other projects. **Skills** - 28 skills inject knowledge into workers

 **Autopilot** - `/ant:run` - Automated build-verify-advance across phases

## Install

**Option 1: Go binary (recommended)**

```bash
# Install the Go binary
go install github.com/calcosmic/Aether@latest

# Set up the hub in your project
aether install   # Populates ~/.aether/ with system files
aether setup    # Sync companion files to local repo

# Start a colony
/ant:lay-eggs            # One-time setup
/ant:init "Build X"      # Start colony with goal
/ant:plan                # Generate phased roadmap
/ant:build 1     # Execute phase with worker waves
/ant:continue          # Verify, learn, advance
/ant:seal                # Done — archive the
```

Five commands from zero to shipped.

## Key Features

| | Feature | Description |
|---|---------|-------------|
| **Agents** | 24 Specialized Workers | Builder, Watcher, Scout, Tracker, Archaeologist, Oracle, and more |
| **Commands** | 45 Slash Commands | Full lifecycle for Claude Code and OpenCode |
| **Signals** | Pheromone System | FOCUS, REDIRECT, FEEDBACK — guide colony attention |
| **Memory** | Colony Wisdom | Learnings and instincts persist via QUEEN.md |
| **Hive Brain** | Cross-colony | Domain-scoped wisdom sharing |
| **Autopilot** | `/ant:run` | Build-verify-advance loop with smart pause |
| **Skills** | 28 Skills | 10 colony + 18 domain knowledge for workers |
| **Research** | Oracle + Scouts | Deep autonomous research before task decomposition |
| **Quality Gates** | 6-phase verification before advancing |
| **Platforms** | Claude Code + OpenCode | Binary + agent support |

## Architecture

```
.aether/                        Colony files (repo-local)
├── commands/                   45 YAML command sources
├── agents-claude/               Claude agent definitions
├── skills/                     28 skills (10 colony + 18 domain)
├── exchange/                   XML exchange modules
├── docs/                       Documentation
├── templates/                  12 templates
└── data/                       Colony state (local only)

~/.aether/                     Hub (cross-colony, user-level)
├── system/                   Companion file source (populated by install)
├── QUEEN.md                 Wisdom + preferences
├── hive/wisdom.json         Cross-colony wisdom (200 cap)
```

**Runtime:** Go 1.22+  
**Distribution:** GoReleaser (Linux, macOS, Windows / amd64 + arm64)

  
**Package:** `aether-colony` on npm (companion files only)

## Works With

- **[Claude Code](https://docs.anthropic.com/en/docs/claude-code?utm_source=github&utm_medium=readme&utm_campaign=aether)** - 45 slash commands + 24 agent definitions
- **[OpenCode](https://github.com/opencode-ai/opencode?utm_source=github&utm_medium=readme&utm_campaign=aether)** - 45 slash commands + agent definitions

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

[PayPal](https://www.paypal.com/ncp/payment/RENG7ZMW5F59L?utm_source=github&utm_medium=readme&utm_campaign=aether) | [Buy Me a Coffee](https://buymeacoffee.com/music5y?utm_source=github&utm_medium=readme&utm_campaign=aether)

## License

MIT
