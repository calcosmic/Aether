# CLAUDE.md — Aether Repo Rules

> **CRITICAL:** See `RUNTIME UPDATE ARCHITECTURE.md` for the complete architecture diagram and distribution flow.

## How Development Works

```
┌────────────────────────────────────────────────────────────────┐
│  In the Aether repo, .aether/ IS the source of truth.          │
│  Edit system files there naturally.                            │
│                                                                │
│  .aether/           → SOURCE OF TRUTH (edit this)              │
│  runtime/           → STAGING (auto-populated on publish)      │
│                                                                │
│  A sync script copies .aether/ → runtime/ before packaging.   │
└────────────────────────────────────────────────────────────────┘
```

| What you're changing | Where to edit | Why |
|---------------------|---------------|-----|
| workers.md | `.aether/workers.md` | Source of truth |
| aether-utils.sh | `.aether/aether-utils.sh` | Source of truth |
| utils/*.sh | `.aether/utils/` | Source of truth |
| User docs | `.aether/docs/` | Source of truth (allowlisted docs get distributed) |
| Slash commands | `.claude/commands/ant/` | Claude Code commands |
| OpenCode commands | `.opencode/commands/ant/` | OpenCode commands |
| Agent definitions | `.opencode/agents/` | Agent definitions |
| Visualizations | `.aether/visualizations/` | Distributed directly |
| Your notes | `.aether/dreams/` | Never distributed, safe |
| Dev docs | `.aether/docs/known-issues.md`, `implementation-learnings.md` | Distributed — extracted findings |
| Aether TODOs | `TO-DOS.md` (root) | Source of truth for Aether development |

> **Note:** For OpenCode-specific rules, see `.opencode/OPENCODE.md`

**After editing system files:**
```bash
git add .
git commit -m "your message"
npm install -g .   # Auto-syncs .aether/ → runtime/, then pushes to hub
```

---

## Critical Architecture

**In the Aether repo, `.aether/` system files are the source of truth.** A sync script (`bin/sync-to-runtime.sh`) copies them to `runtime/` automatically when you run `npm install -g .`. The `runtime/` directory is a staging area for the npm package.

```
Aether Repo (this repo)
├── .aether/ (SOURCE OF TRUTH for system files)
│   ├── workers.md, aether-utils.sh, utils/, docs/
│   └── data/                        ← LOCAL (never touched)
│         │
│         │  bin/sync-to-runtime.sh (auto on npm install)
│         ▼
├── runtime/ (STAGING — auto-populated from .aether/)
├── .claude/commands/ant/ ─────────────────────────────┐
├── .opencode/ ────────────────────────────────────────┤──→ npm package
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

**Development workflow:**
1. Edit `.aether/` system files (or `.claude/commands/ant/` for slash commands) naturally
2. Commit changes
3. Run `npm install -g .` — auto-syncs `.aether/` → `runtime/`, then pushes to hub
4. Hub distributes to all repos via `aether update`

**In other repos:** `.aether/` is a working copy that gets overwritten by `aether update`. Don't edit system files there — they come from the hub.

---

## Key Directories

| Directory | Purpose | Syncs to Hub |
|-----------|---------|--------------|
| `.aether/` (system files) | Source of truth for workers.md, aether-utils.sh, utils/, docs/ | → `runtime/` → `~/.aether/system/` |
| `.claude/commands/ant/` | Claude Code slash commands | → `~/.aether/commands/claude/` |
| `.opencode/commands/ant/` | OpenCode slash commands | → `~/.aether/commands/opencode/` |
| `.opencode/agents/` | Agent definitions | → `~/.aether/agents/` |
| `runtime/` | Staging directory (auto-populated, do not edit directly) | → `~/.aether/system/` |
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
| ambassador | 🔌 | Third-party API integration |
| auditor | 👥 | Code review, quality audits |
| chronicler | 📝 | Documentation generation |
| gatekeeper | 📦 | Dependency management |
| guardian | 🛡️ | Security audits |
| includer | ♿ | Accessibility audits |
| keeper | 📚 | Knowledge curation |
| measurer | ⚡ | Performance profiling |
| probe | 🧪 | Test generation |
| sage | 📜 | Analytics & insights |
| tracker | 🐛 | Bug investigation |
| weaver | 🔄 | Code refactoring |

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

# Communication style

  - Explain things twice: once technically, once in plain English ("for
  dummies").
  - Keep technical details accurate; keep plain-English explanations simple.
  - Example format: "I'm using X because Y. In other words: ..."

---

## Active Development

### Session Freshness Detection System (In Progress)

All stateful commands now use timestamp verification to detect stale sessions. This prevents old session files from silently breaking workflows.

**Pattern:**
1. Capture `SESSION_START=$(date +%s)` before spawning agents
2. Check file freshness with `session-verify-fresh --command <name>`
3. Auto-clear stale files or prompt user based on command type
4. Verify files are fresh after spawning

**Current Phase:** See `docs/aether_dev_handoff.md` for implementation status

**Full Plan:** `docs/session-freshness-implementation-plan.md`

**Protected Commands** (never auto-clear):
- `init` - COLONY_STATE.json is precious
- `seal` - Archives are precious
- `entomb` - Chambers are precious