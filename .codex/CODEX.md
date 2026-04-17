# CODEX.md -- Aether Codex Developer Guide

> **CRITICAL:** This file provides Codex-specific guidance for the Aether system.
> For the complete architecture and update flow, see `RUNTIME UPDATE ARCHITECTURE.md`.
> For the full project guide, see `CLAUDE.md`.

> **Note:** For Claude Code rules, see `CLAUDE.md`. For OpenCode rules, see `.opencode/OPENCODE.md`.

---

## Introduction

This file is the developer-facing guide for using **Aether** with **OpenAI Codex CLI**.
It explains how the Aether multi-agent colony system works when Codex is the AI platform,
what files live where, and how to develop effectively.

**Relationship to AGENTS.md:** The project-root `AGENTS.md` contains Codex system instructions
that Codex CLI reads automatically. This file (`.codex/CODEX.md`) is a deeper developer
reference for humans and agents working within the Codex platform. Think of AGENTS.md as
"what Codex needs to know" and this file as "what a developer needs to know."

---

## How Development Works

```
+----------------------------------------------------------------+
|  In the Aether repo, .aether/ IS the source of truth.           |
|  Edit system files there and publish directly.                  |
|                                                                 |
|  .aether/           -> SOURCE OF TRUTH (edit this, published)   |
|  .aether/data/      -> LOCAL ONLY (never distributed)            |
|  .aether/dreams/    -> LOCAL ONLY (never distributed)            |
|                                                                 |
|  .codex/agents/     -> SOURCE OF TRUTH (Codex agent TOML defs)  |
|  AGENTS.md          -> SOURCE OF TRUTH (Codex system prompt)    |
|                                                                 |
|  `aether install --package-dir "$PWD"` refreshes hub files      |
|  and Codex assets from this checkout.                           |
+----------------------------------------------------------------+
```

| What you're changing | Where to edit | Why |
|---------------------|---------------|-----|
| Agent definitions | `.codex/agents/*.toml` | Source of truth for Codex agents |
| System prompt | `AGENTS.md` (project root) | Codex reads this automatically |
| Colony rules | `.aether/workers.md` | Source of truth |
| Aether CLI | `cmd/` (Go binary) | Platform-agnostic |
| User docs | `.aether/docs/` | Synced via `aether install` / `aether update` |

**After editing:**
```bash
git add .
git commit -m "your message"
aether install --package-dir "$PWD"
```

---

## Critical Architecture

**`.aether/` + `.codex/` are the source of truth.** Release binaries embed the shipped
companion files, and local development can publish directly from this checkout with
`aether install --package-dir "$PWD"`. Codex agent TOML files live in `.codex/agents/`
and sync to the hub alongside the other platform files.

```
Aether Repo (this repo)
+-- .aether/ (SOURCE OF TRUTH -- embedded in release binaries)
|   +-- workers.md, utils/, docs/
|   +-- data/          <- LOCAL ONLY (excluded)
|   +-- dreams/        <- LOCAL ONLY (excluded)
|
+-- .codex/  ----------------------------------------------+
|   +-- agents/*.toml     Codex agent definitions           |-> embedded install assets
|                                                          |
+-- AGENTS.md              Codex system prompt              |
                                                           v
                                                     ~/.aether/ (THE HUB)
                                                     +-- system/      <- .aether/
                                                     |   +-- codex/   <- .codex/agents/
                                                     |   +-- skills-codex/
                                                     +-- agents/      <- user-level Claude/OpenCode assets
                                                     +-- commands/
                                                     |     +-- claude/
                                                     |     +-- opencode/
                                                     |     (no codex/ -- no slash commands)
                                                           |
  aether update (in ANY repo)  <---------------------------+
  aether lay-eggs (initializes colony)

v
any-repo/.codex/ (WORKING COPY)
+-- agents/          <- from hub system/codex/
+-- skills/aether/   <- from hub system/skills-codex/
+-- data/            <- LOCAL (never touched by updates)
```

---

## Key Directories

| Directory | Purpose | Syncs to Hub |
|-----------|---------|--------------|
| `.codex/agents/` | Codex agent definitions (TOML) | -> `~/.aether/system/codex/` |
| `AGENTS.md` | Codex system instructions | Template-generated at setup |
| `.aether/` (system files) | Source of truth for workers, utils, docs | -> `~/.aether/system/` |
| `.aether/agents-codex/` | Codex packaging mirror | Embedded alongside hub assets |
| `.aether/skills-codex/` | Codex-installed skill mirror | -> `~/.aether/system/skills-codex/` |
| `.aether/data/` | Colony state | **NEVER touched** |
| `.aether/dreams/` | Session notes | **NEVER touched** |

---

## Codex-Specific Conventions

### No Slash Commands

Codex CLI does **not** support slash commands like Claude Code (`/ant:build`) or OpenCode.
Instead, all colony operations use the **`aether` CLI directly** or are described as natural
language prompts in `AGENTS.md`.

```bash
# Instead of /ant:init "Build feature X"
aether init "Build feature X"

# Instead of /ant:build 1
aether build 1

# Instead of /ant:status
aether status

# Instead of /ant:pheromones
aether pheromone-display
```

When the user message is already a literal `aether ...` command, execute that exact CLI
command first. Do not inspect repo files to infer intent, and do not treat command-doc
mirrors as more authoritative than the installed `aether` binary. For lifecycle commands
run through Codex shell execution, prefer `AETHER_OUTPUT_MODE=visual aether ...` unless the
user explicitly wants JSON output.

### Agent Definitions (TOML Format)

Codex agent definitions live in `.codex/agents/*.toml` using the TOML configuration format.
Each file defines one agent with its instructions embedded inline.

**Format:**
```toml
name = "aether-builder"
description = "Use this agent for code implementation..."
nickname_candidates = ["builder", "hammer"]

developer_instructions = '''
You are a **Builder Ant** in the Aether Colony...
[Full agent instructions here]
'''
```

**Key differences from other platforms:**

| Aspect | Claude Code | OpenCode | Codex |
|--------|-------------|----------|-------|
| Format | Markdown (`.md`) | Markdown (`.md`) | TOML (`.toml`) |
| Location | `.claude/agents/ant/` | `.opencode/agents/` | `.codex/agents/` |
| Instructions | File content | File content | `developer_instructions` field |
| Metadata | In file header | In file header | TOML keys (`name`, `description`) |

### Skills

The shared skill sources live in `.aether/skills/` and `.aether/skills-codex/`.
For Codex, installed skills are copied into `.codex/skills/aether/` and matched by
the `aether skill-*` commands against worker role, workspace files, and package manifests.

Skills are not a separate Codex plugin bundle. Codex agent definitions remain in
`.codex/agents/*.toml`, and the Go CLI handles skill indexing, matching, and injection.

Skills are automatically matched and injected into worker prompts during `build`,
`colonize`, and `plan` dispatches.

### Pheromone Signals

Active pheromone signals are automatically injected into worker prompts during
`build`, `colonize`, and `plan` dispatches.

Pheromone signals work identically across all platforms via the `aether` CLI:

```bash
# Set signals
aether pheromone-write --type FOCUS --content "security"
aether pheromone-write --type REDIRECT --content "avoid global state"
aether pheromone-write --type FEEDBACK --content "prefer composition"

# View signals
aether pheromone-display

# Export/import for cross-colony sharing
aether pheromone-export
aether pheromone-import --file signals.xml
```

---

## Workflow for Codex Users

### Colony Setup

```bash
# Initialize colony with a goal
aether init "Build feature X"

# Analyze existing codebase (optional)
aether colonize

# Generate phase plan
aether plan

# Preview or run the autopilot loop
aether run --dry-run
aether run --max-phases 2

# Build a phase (workers execute)
aether build 1

# Verify, learn, advance
aether continue

# Live worker view and research workspace
aether watch
aether oracle "release concern"

# Finish the colony
aether seal
```

### Status and Monitoring

```bash
aether status           # Colony dashboard
aether phase            # Current phase details
aether phase 3          # Specific phase
aether watch            # Live worker activity alias
aether swarm --watch    # Live worker activity
aether flags            # Active flags
aether flag --title "Investigate auth bug"
aether history          # Colony events
aether memory-details   # Drill-down memory view
aether patrol           # System health check
```

### Pheromone Signals

```bash
# Guide colony before builds
aether focus "security"                   # Pay attention here
aether redirect "avoid global state"       # Hard constraint
aether feedback "prefer early returns"     # Gentle adjustment

# View and manage
aether pheromones                          # Full signal table
aether pheromone-display                   # Formatted with strength %
aether export-signals                      # Share across colonies
aether import-signals --file signals.xml   # Import signals
```

### Lifecycle

```bash
aether seal              # Seal colony (Crowned Anthill)
aether update            # Pull latest from hub
aether resume            # Restore saved session context
```

### Advanced

```bash
aether preferences "prefer verbose output"
aether phase-insert --after 1 --name "Fix auth" --description "Stabilize auth flow"
aether run --headless --max-phases 3
aether swarm "build flakes in auth"
aether oracle "auth flake investigation"
aether skill-match --role builder --task "react form validation"
aether skill-inject --role builder --task "react form validation"
aether data-clean        # Clean test artifacts
```

---

## Agent Reference

All 24 agents are defined in `.codex/agents/*.toml`:

| Tier | Agent | TOML File | Role |
|------|-------|-----------|------|
| Core | Queen | `aether-queen.toml` | Orchestrates phases, spawns workers |
| Core | Builder | `aether-builder.toml` | Implements code, TDD-first |
| Core | Watcher | `aether-watcher.toml` | Tests, validates, quality gates |
| Orchestration | Scout | `aether-scout.toml` | Researches, gathers information |
| Orchestration | Route-Setter | `aether-route-setter.toml` | Plans phases, breaks down goals |
| Orchestration | Architect | `aether-architect.toml` | Architecture design, structural planning |
| Surveyor | surveyor-nest | `aether-surveyor-nest.toml` | Maps directory structure |
| Surveyor | surveyor-disciplines | `aether-surveyor-disciplines.toml` | Documents conventions |
| Surveyor | surveyor-pathogens | `aether-surveyor-pathogens.toml` | Identifies tech debt |
| Surveyor | surveyor-provisions | `aether-surveyor-provisions.toml` | Maps dependencies |
| Specialist | Keeper | `aether-keeper.toml` | Preserves knowledge |
| Specialist | Tracker | `aether-tracker.toml` | Investigates bugs |
| Specialist | Probe | `aether-probe.toml` | Coverage analysis |
| Specialist | Weaver | `aether-weaver.toml` | Refactoring specialist |
| Specialist | Auditor | `aether-auditor.toml` | Quality gate |
| Niche | Chaos | `aether-chaos.toml` | Resilience testing |
| Niche | Archaeologist | `aether-archaeologist.toml` | Excavates git history |
| Niche | Gatekeeper | `aether-gatekeeper.toml` | Security gate |
| Niche | Includer | `aether-includer.toml` | Accessibility audits |
| Niche | Measurer | `aether-measurer.toml` | Performance analysis |
| Niche | Sage | `aether-sage.toml` | Wisdom synthesis |
| Niche | Oracle | `aether-oracle.toml` | Deep research, actionable recommendations |
| Niche | Ambassador | `aether-ambassador.toml` | External integrations |
| Niche | Chronicler | `aether-chronicler.toml` | Documentation |

---

## Development Rules

### Editing Files

| What | Where to edit | Notes |
|------|---------------|-------|
| Codex agent defs | `.codex/agents/*.toml` | TOML format; `developer_instructions` field holds content |
| System prompt | `AGENTS.md` | Project root; Codex reads this automatically |
| Colony workers | `.aether/workers.md` | Shared across all platforms |
| Go CLI | `cmd/` | Platform-agnostic; no changes needed for Codex |
| Skills | `.aether/skills/` | Shared across all platforms |
| Templates | `.aether/templates/` | Includes Codex template |

### Protected Paths

Never write to these programmatically:

| Path | Reason |
|------|--------|
| `.aether/data/` | Colony state (COLONY_STATE.json, session files) |
| `.aether/dreams/` | Dream journal entries |
| `.codex/config.toml` | Codex configuration |
| `.env*` | Environment secrets |
| `.github/workflows/` | CI configuration |

### Verification Commands

```bash
# Go tests
go test ./...
go test ./... -race

# Build verification
go build ./cmd/aether

# Lint
go vet ./...

# Binary smoke test
./aether version

# Verify Codex agent files are valid TOML
for f in .codex/agents/*.toml; do
  echo "Checking $f..."
  # Basic syntax check -- TOML is simple enough for grep validation
done

# Verify all 24 agents exist
ls .codex/agents/*.toml | wc -l  # Should be 24
```

### Publishing Changes

```bash
# 1. Edit files in .codex/ or .aether/
# 2. Commit
git add .
git commit -m "update codex agent definitions"

# 3. Refresh the hub from this source checkout
aether install --package-dir "$PWD"

# 4. In other repos, pull updates
aether update
```

---

## Comparison with Claude Code and OpenCode

### Key Differences

| Aspect | Claude Code | OpenCode | Codex |
|--------|-------------|----------|-------|
| System prompt file | `CLAUDE.md` | `.opencode/OPENCODE.md` | `AGENTS.md` |
| Agent format | Markdown `.md` | Markdown `.md` | TOML `.toml` |
| Agent location | `.claude/agents/ant/` | `.opencode/agents/` | `.codex/agents/` |
| Slash commands | Yes (46 commands) | Yes (46 commands) | **No** -- use `aether` CLI |
| Command location | `.claude/commands/ant/` | `.opencode/commands/ant/` | N/A |
| Agent metadata | In markdown header | In markdown header | TOML keys |
| Hub sync path | `~/.aether/agents/` | `~/.aether/agents/` | `~/.aether/system/codex/` + `~/.aether/system/skills-codex/` |
| Worker runtime | `Task` tool | `Task` tool | `codex exec` driven by `aether` |
| Rules file | `.claude/rules/` | Inline | `AGENTS.md` sections |

### What is the same

- **Go binary** (`cmd/aether`) is platform-agnostic -- all three platforms use the same CLI
- **Colony state** (`.aether/data/COLONY_STATE.json`) works identically
- **Pheromone signals** work via `aether pheromone-*` commands on all platforms
- **Skills system** (`.aether/skills/`) is shared and platform-agnostic
- **Hub sync** (`~/.aether/`) distributes to all platforms from the same source
- **Memory/learning pipeline** (wisdom, instincts, hive) is identical
- **TDD discipline** and coding standards apply regardless of platform

### Workflow Translation

| Goal | Claude Code | OpenCode | Codex |
|------|-------------|----------|-------|
| Start colony | `/ant:init "goal"` | `/ant:init "goal"` | `aether init "goal"` |
| Build phase | `/ant:build 1` | `/ant:build 1` | `aether build 1` |
| Check status | `/ant:status` | `/ant:status` | `aether status` |
| Focus signal | `/ant:focus "area"` | `/ant:focus "area"` | `aether focus "area"` |
| Update Aether | `/ant:update` | `/ant:update` | `aether update` |
| Seal colony | `/ant:seal` | `/ant:seal` | `aether seal` |
| Deep research | `/ant:oracle` | `/ant:oracle` | Use the `aether-oracle` agent plus `aether skill-*` and research commands |
| View pheromones | `/ant:pheromones` | `/ant:pheromones` | `aether pheromone-display` |

---

*Updated for Aether v1.0.8 -- 2026-04-17*
