# CLAUDE.md — Aether Repo Rules

> **CRITICAL:** See `/Users/callumcowie/repos/Aether/RUNTIME UPDATE ARCHITECTURE.md` for the complete architecture diagram and distribution flow. This document explains how the runtime/ directory, hub, and .aether/ working copy interact — read it before making any changes.

## ⚠️ RULE #1: NEVER EDIT .aether/ SYSTEM FILES

```
┌────────────────────────────────────────────────────────────────┐
│  EDIT runtime/ — NOT .aether/                                  │
│                                                                │
│  runtime/           → SOURCE OF TRUTH (edit this)              │
│  .aether/           → WORKING COPY (gets overwritten)          │
│                                                                │
│  If you edit .aether/, your work WILL BE LOST on next update.  │
└────────────────────────────────────────────────────────────────┘
```

| What you're changing | Where to edit | Why |
|---------------------|---------------|-----|
| workers.md | `runtime/workers.md` | Source of truth |
| aether-utils.sh | `runtime/aether-utils.sh` | Source of truth |
| utils/*.sh | `runtime/utils/` | Source of truth |
| User docs | `runtime/docs/` | Source of truth |
| Slash commands | `.claude/commands/ant/` | Source of truth |
| Visualizations | `.aether/visualizations/` | Exception - distributed directly |
| Your notes | `.aether/docs/` | Never distributed, safe |

**After editing runtime/:**
```bash
git add runtime/
git commit -m "your message"
npm install -g .   # Push to hub
```

---

## Critical Architecture

**runtime/ is the source of truth for npm distribution.** `.aether/` is the working copy in repos.

```
Aether Repo (this repo)
├── runtime/ ──────────────────────────────────────────┐
├── .claude/commands/ant/ ─────────────────────────────┤──→ npm package
├── .opencode/ ────────────────────────────────────────┤
│                                                      ▼
│                                                ~/.aether/ (THE HUB)
│                                                ├── system/      ← runtime/
│                                                ├── commands/    ← slash commands
│                                                └── agents/
│                                                      │
│  aether update (in ANY repo)  ◄──────────────────────┘
│
▼
any-repo/.aether/ (WORKING COPY - gets overwritten)
├── workers.md, aether-utils.sh  ← from hub (system files)
└── data/                        ← LOCAL (never touched by updates)
```

**The destructive loop to avoid:**
1. Edit `.aether/` directly → features work locally
2. Run `npm install` → copies stale `runtime/` to hub
3. Run `aether update` → copies stale hub to `.aether/` → **destroys your work**

**Correct development workflow:**
1. Edit `runtime/` (or `.claude/commands/ant/` for slash commands)
2. Commit changes
3. Run `npm install -g .` to update hub
4. Hub distributes to all repos via `aether update`

---

## Key Directories

| Directory | Purpose | Syncs to Hub |
|-----------|---------|--------------|
| `runtime/` | System files (workers.md, aether-utils.sh, utils/) | → `~/.aether/system/` |
| `.claude/commands/ant/` | Claude Code slash commands | → `~/.claude/commands/ant/` + `~/.aether/commands/claude/` |
| `.opencode/commands/ant/` | OpenCode slash commands (repo-local only) | → `~/.aether/commands/opencode/` |
| `.opencode/agents/` | Agent definitions | → `~/.aether/agents/` |
| `.aether/` | Working copy in THIS repo | Gets overwritten by updates |
| `.aether/data/` | Colony state (COLONY_STATE.json, pheromones.json) | **NEVER touched** |

---

## Pheromone System (User-Colony Communication)

| Signal | Command | Priority | Use For |
|--------|---------|----------|---------|
| FOCUS | `/ant:focus "<area>"` | normal | "Pay attention here" |
| REDIRECT | `/ant:redirect "<avoid>"` | high | "Don't do this" (hard constraint) |
| FEEDBACK | `/ant:feedback "<note>"` | low | "Adjust based on this observation" |

**Before builds:** FOCUS + REDIRECT to steer
**After builds:** FEEDBACK to adjust
**Hard constraints:** REDIRECT (will break)
**Gentle nudges:** FEEDBACK (preferences)

See `.aether/docs/pheromones.md` for full guide.

---

## Caste System

Workers are assigned to castes based on task type:

| Caste | Emoji | Role |
|-------|-------|------|
| builder | 🔨 | Implementation work |
| watcher | 👁️ | Monitoring, observation |
| scout | 🔍 | Research, discovery |
| chaos | 🎲 | Edge case testing |
| oracle | 🔮 | Deep research (RALF loop) |
| architect | 🏗️ | Planning, design |
| prime | 🏛️ | High-level coordination |
| colonizer | 🌱 | New project setup |
| route_setter | 🧭 | Direction setting |
| archaeologist | 📜 | Git history excavation |

See `.aether/docs/biological-reference.md` for full taxonomy.

---

## Milestone Names (Biological Metaphors)

| Milestone | Meaning |
|-----------|---------|
| First Mound | First runnable |
| Open Chambers | Feature work underway |
| Brood Stable | Tests consistently green |
| Ventilated Nest | Perf/latency acceptable |
| Sealed Chambers | Interfaces frozen |
| Crowned Anthill | Release ready |
| New Nest Founded | Next major version |

---

## Verification Commands

```bash
# Verify commands in sync between Claude Code and OpenCode
npm run lint:sync

# Verify model routing configuration
aether verify-models

# Check caste model assignments
aether caste-models list

# Run all tests
npm test
```
