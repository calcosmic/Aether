# OPENCODE.md — Aether OpenCode Rules

> **CRITICAL:** This file provides OpenCode-specific guidance for the Aether system. For the complete architecture and update flow, see `../RUNTIME UPDATE ARCHITECTURE.md`.

> **Note:** For Claude Code-specific rules (the other platform), see `../CLAUDE.md`

## How Development Works

```
┌────────────────────────────────────────────────────────────────┐
│  In the Aether repo, .aether/ IS the source of truth.          │
│  Edit system files there and publish directly.                 │
│                                                                │
│  .aether/           → SOURCE OF TRUTH (edit this, published)  │
│  .aether/data/      → LOCAL ONLY (excluded by .npmignore)      │
│  .aether/dreams/    → LOCAL ONLY (excluded by .npmignore)      │
│                                                                │
│  npm install -g . validates .aether/ and pushes to hub.        │
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
npm install -g .   # Validates .aether/, then pushes to hub
```

---

## Critical Architecture

**`.aether/` + `.opencode/` are the source of truth.** `.aether/` is packaged directly into the npm package; private directories are excluded by `.aether/.npmignore`.

```
Aether Repo (this repo)
├── .aether/ (SOURCE OF TRUTH — packaged directly into npm)
│   ├── workers.md, aether-utils.sh, utils/, docs/
│   ├── data/          ← LOCAL ONLY (excluded by .aether/.npmignore)
│   └── dreams/        ← LOCAL ONLY (excluded by .aether/.npmignore)
│
├── .opencode/ ────────────────────────────────────────┤──→ npm package
│   ├── agents/                                        │
│   └── commands/ant/                                  │
│                                                      ▼
│                                                ~/.aether/ (THE HUB)
│                                                ├── system/      ← .aether/
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
| `.aether/` (system files) | Source of truth for workers.md, utils, docs | → `~/.aether/system/` |
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

## Argument Parsing (Fixed 2026-02-15)

**Issue:** OpenCode doesn't pass `$ARGUMENTS` the same way as Claude Code. When users ran `/ant:init Build a REST API`, only "Build" was captured.

**Fix:** All commands now use `normalize-args` helper that checks:
1. `$ARGUMENTS` (Claude Code style)
2. `$@` (OpenCode style)

**Implementation:**
```bash
# At start of each command
Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

# Then use `$normalized_args` instead of `$ARGUMENTS`
```

**For Users:**
If argument parsing issues persist, wrap multi-word arguments in quotes:
```
/ant:init "Build a REST API"   # Always works
/ant:init Build a REST API      # Now works with normalize-args
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
