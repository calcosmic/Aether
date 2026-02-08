```
     _    _____ _____ _   _ _____ ____
    / \  | ____|_   _| | | | ____|  _ \
   / _ \ |  _|   | | | |_| |  _| | |_) |
  / ___ \| |___  | | |  _  | |___|  _ <
 /_/   \_\_____| |_| |_| |_|_____|_| \_\
```

# 🐜 AETHER v2.0

A multi-agent system for Claude Code where **workers spawn other workers**.

Inspired by [Tache's GSD system](https://github.com/tache-ai/gsd) — Aether takes that foundation and adds ant colony dynamics: pheromone signals, caste specialization, and nested spawning.

---

## Quick Start

```bash
npm install -g aether-colony
```

In Claude Code:

```bash
/ant:init "Build a REST API with authentication"
/ant:plan
/ant:build 1
```

The colony self-organizes from there.

---

## How It Works

```
👑 Queen (you)
   │
   ▼
🐜 Workers spawn Workers (max depth 3)
   │
   ├── 🔨 Builders — write code
   ├── 👁️ Watchers — verify & test
   ├── 🔍 Scouts — research
   └── 📋 Route-setters — plan phases
```

You provide intention via pheromone signals (`FOCUS`, `REDIRECT`, `FEEDBACK`). The colony interprets them and adapts.

---

## Commands

| Command | Purpose |
|---------|---------|
| `/ant:init "goal"` | Set colony mission |
| `/ant:plan` | Generate phases |
| `/ant:build N` | Execute phase N |
| `/ant:continue` | Advance to next phase |
| `/ant:focus "area"` | Guide attention |
| `/ant:status` | Colony overview |
| `/ant:watch` | Live tmux monitoring |
| `/ant:flag "issue"` | Track blockers |

---

## v2.0 Features

- 🐜 **Nested spawning** — Workers spawn sub-workers (depth 1→2→3)
- 🎨 **Colorized output** — Caste-specific colors
- 👁️ **Runtime verification** — Watchers execute code, not just read it
- 🚩 **Flagging system** — Issues persist across context resets
- 🔨 **Named ants** — Hammer-42, Vigil-17, Quest-33

---

## Installation

**Prerequisites:** Node.js >= 16, `jq` (`brew install jq`)

```bash
# Install
npm install -g aether-colony

# Update
npm update -g aether-colony

# Uninstall
aether uninstall && npm uninstall -g aether-colony
```

---

## File Structure

```
~/.claude/commands/ant/     # Slash commands
~/.aether/                  # Worker specs, utilities
<repo>/.aether/data/        # Per-project state
```

---

## Acknowledgments

Shoutout to **[Tache](https://github.com/tache-ai)** and the **[GSD system](https://github.com/tache-ai/gsd)** for the inspiration.

---

## License

MIT

---

*🐜 "The whole is greater than the sum of its parts." 🐜*
