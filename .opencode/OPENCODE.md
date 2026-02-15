# OPENCODE.md — Aether OpenCode Rules

> **CRITICAL:** This file provides OpenCode-specific guidance for the Aether system. For the complete architecture and update flow, see `../RUNTIME UPDATE ARCHITECTURE.md`.

> **Note:** For Claude Code-specific rules (the other platform), see `../CLAUDE.md`

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
| Agent definitions | `.opencode/agents/` | Source of truth |
| Slash commands | `.opencode/commands/ant/` | Source of truth |
| workers.md | `.aether/workers.md` | Source of truth |
| aether-utils.sh | `.aether/aether-utils.sh` | Source of truth |

**After editing:**
```bash
git add .
git commit -m "your message"
npm install -g .   # Auto-syncs .aether/ → runtime/, then pushes to hub
```

---

## Critical Architecture

**`.aether/` + `.opencode/` are the source of truth.** `runtime/` is a staging directory auto-populated from `.aether/` on publish.

```
Aether Repo (this repo)
├── .aether/ (SOURCE OF TRUTH for system files)
│   ├── workers.md, aether-utils.sh, utils/, docs/
│   │        │
│   │        │  bin/sync-to-runtime.sh (auto on npm install)
│   │        ▼
├── runtime/ (STAGING — auto-populated)
├── .opencode/ ────────────────────────────────────────┤──→ npm package
│   ├── agents/                                        │
│   └── commands/ant/                                  │
│                                                      ▼
│                                                ~/.aether/ (THE HUB)
│                                                ├── system/      ← runtime/
│                                                ├── commands/    ← slash commands
│                                                └── agents/      ← .opencode/agents/
│                                                      │
│  aether update (in ANY repo)  ◄──────────────────────┘
│  /ant:update (slash command)
│
▼
any-repo/.aether/ (WORKING COPY - gets overwritten)
├── agents/          ← from hub (.opencode/agents/)
├── commands/        ← from hub (.opencode/commands/)
└── data/            ← LOCAL (never touched by updates)
```

---

## Key Directories

| Directory | Purpose | Syncs to Hub |
|-----------|---------|--------------|
| `.opencode/agents/` | Agent definitions | → `~/.aether/agents/` |
| `.opencode/commands/ant/` | OpenCode slash commands | → `~/.aether/commands/opencode/` |
| `.aether/` (system files) | Source of truth for workers.md, utils, docs | → `runtime/` → `~/.aether/system/` |
| `runtime/` | Staging (auto-populated, do not edit) | → `~/.aether/system/` |
| `.aether/data/` | Colony state | **NEVER touched** |

---

## Agent Files

Agent definitions live in `.opencode/agents/`:

```
.opencode/agents/
├── aether-queen.md      # Prime coordinator
├── aether-builder.md    # Implementation
├── aether-watcher.md   # Validation
├── aether-scout.md     # Research
├── aether-ambassador.md # API integration
├── aether-auditor.md   # Code review
├── aether-chronicler.md # Documentation
├── aether-gatekeeper.md # Dependencies
├── aether-guardian.md   # Security
├── aether-includer.md  # Accessibility
├── aether-keeper.md    # Knowledge
├── aether-measurer.md  # Performance
├── aether-probe.md     # Testing
├── aether-sage.md      # Analytics
├── aether-tracker.md   # Debugging
├── aether-weaver.md    # Refactoring
└── workers.md          # Full specifications
```

### Spawning Agents

Use the **Task tool** with `subagent_type`:

```
Use the task tool with:
- subagent_type: "aether-builder"
- prompt: "..."

Results return inline.
```

---

## Slash Commands

Slash commands live in `.opencode/commands/ant/`:

| Command | Purpose |
|---------|---------|
| `/ant:build` | Start a build phase |
| `/ant:plan` | Create a phase plan |
| `/ant:watch` | View colony status |
| `/ant:phase` | Phase management |
| `/ant:update` | Update Aether system |

---

## Verification Commands

```bash
# Update Aether from this repo
npm install -g .

# In any repo, pull latest
/ant:update

# Verify agent files are in place
ls ~/.aether/agents/
```
