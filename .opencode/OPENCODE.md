# OPENCODE.md — Aether OpenCode Rules

> **CRITICAL:** This file provides OpenCode-specific guidance for the Aether system. For the complete architecture and update flow, see `../RUNTIME UPDATE ARCHITECTURE.md`.

> **Note:** For Claude Code-specific rules (the other platform), see `../CLAUDE.md`

## How Development Works

```
┌────────────────────────────────────────────────────────────────┐
│  In the Aether repo, .aether/ IS the source of truth.          │
│  Edit system files there and publish directly.                 │
│                                                                │
│  .aether/           → SOURCE OF TRUTH (edit this, published)   │
│  .aether/data/      → LOCAL ONLY (never distributed)           │
│  .aether/dreams/    → LOCAL ONLY (never distributed)           │
│                                                                │
│  aether publish refreshes hub files and rebuilds the binary.  │
│  aether install --package-dir "$PWD" still works but lacks    │
│  automatic version verification.                               │
└────────────────────────────────────────────────────────────────┘
```

| What you're changing | Where to edit | Why |
|---------------------|---------------|-----|
| Agent definitions | `.opencode/agents/` | Source of truth |
| Slash commands | `.opencode/commands/ant/` | Source of truth |
| workers.md | `.aether/workers.md` | Source of truth |
| aether CLI | `cmd/` (Go binary) | Source of truth |

**After editing:**
```bash
git add .
git commit -m "your message"
aether publish
```

> `aether install --package-dir "$PWD"` still works for backward compatibility.

---

## Critical Architecture

**`.aether/` + `.opencode/` are the source of truth.** Source-checkout publishes go through `aether publish` in this repo, which builds the binary, refreshes `~/.aether/system/`, and verifies version agreement. `aether install --package-dir "$PWD"` still works for backward compatibility. Downstream repos then pull those companion files with `aether update` or `/ant-update`. For isolated source-development on the same machine, use the dev channel instead: `aether publish --channel dev --binary-dest "$HOME/.local/bin"`, then use `aether-dev update --force` in target repos. This keeps `~/.aether-dev/` and `aether-dev` separate from the public stable runtime.

```
Aether Repo (this repo)
├── .aether/ (SOURCE OF TRUTH)
│   ├── workers.md, utils/, docs/
│   ├── data/          ← LOCAL ONLY (never distributed)
│   └── dreams/        ← LOCAL ONLY (never distributed)
│
├── .opencode/
│   ├── agents/
│   └── commands/ant/
│
├── aether publish
│
▼
~/.aether/system/
├── .aether/*                    ← system files
├── commands/opencode/           ← OpenCode slash commands
└── agents/                      ← OpenCode agents
   │
   └── aether update or /ant-update
      ▼
      any-repo/
      ├── .aether/
      ├── .opencode/commands/ant/
      └── .opencode/agents/
```

---

## Key Directories

| Directory | Purpose | Syncs to Hub |
|-----------|---------|--------------|
| `.opencode/agents/` | Agent definitions | → `~/.aether/system/agents/` |
| `.opencode/commands/ant/` | OpenCode slash commands | → `~/.aether/system/commands/opencode/` |
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

**Issue:** OpenCode doesn't pass `$ARGUMENTS` the same way as Claude Code. When users ran `/ant-init Build a REST API`, only "Build" was captured.

**Fix:** All commands now use `normalize-args` helper that checks:
1. `$ARGUMENTS` (Claude Code style)
2. `$@` (OpenCode style)

**Implementation:**
```bash
# At start of each command
Run: `normalized_args=$(aether normalize-args "$@")`

# Then use `$normalized_args` instead of `$ARGUMENTS`
```

**For Users:**
If argument parsing issues persist, wrap multi-word arguments in quotes:
```
/ant-init "Build a REST API"   # Always works
/ant-init Build a REST API      # Now works with normalize-args
```

---

## Slash Commands

Slash commands live in `.opencode/commands/ant/`:

| Command | Purpose |
|---------|---------|
| `/ant-build` | Start a build phase |
| `/ant-plan` | Create a phase plan |
| `/ant-watch` | View colony status |
| `/ant-phase` | Phase management |
| `/ant-update` | Update Aether system |

---

## Verification Commands

```bash
# Publish unreleased source-checkout changes from this repo
aether publish

# In any repo, pull the refreshed companion files
/ant-update
aether update --force

# Verify the hub publish actually contains OpenCode surfaces
find ~/.aether/system/commands/opencode -maxdepth 1 -type f | wc -l
find ~/.aether/system/agents -maxdepth 1 -type f | wc -l

# Verify version agreement
aether version --check

# Run a focused release-pipeline integrity check
aether integrity

# If you also need the published release runtime binary
aether update --force --download-binary
```
