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
|  In the Aether repo, canonical source files are edited once.    |
|  Publish/install writes those files to the global hub.           |
|                                                                 |
|  .aether/           -> SOURCE for workers, skills, docs, utils  |
|  .aether/data/      -> LOCAL ONLY (never distributed)            |
|  .aether/dreams/    -> LOCAL ONLY (never distributed)            |
|                                                                 |
|  .codex/agents/     -> SOURCE OF TRUTH (Codex agent TOML defs)  |
|  AGENTS.md          -> SOURCE OF TRUTH (Codex system prompt)    |
|                                                                 |
|  `aether publish` refreshes hub files and global Codex assets.  |
+----------------------------------------------------------------+
```

| What you're changing | Where to edit | Why |
|---------------------|---------------|-----|
| Agent definitions | `.codex/agents/*.toml` | Source of truth for Codex agents |
| System prompt | `AGENTS.md` (project root) | Codex reads this automatically |
| Colony rules | Aether repo `.aether/` `workers.md` | Source of truth, published to hub |
| Aether CLI | `cmd/` (Go binary) | Platform-agnostic |
| User docs | `.aether/docs/` | Published to the global hub |

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
`aether publish` or `aether install --package-dir "$PWD"`. Codex agent TOML files live in `.codex/agents/`
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
                                                     +-- system/      <- global Aether assets
                                                     |   +-- codex/   <- .codex/agents/
                                                     |   +-- codex-skills/ <- versioned public skills and support
                                                     +-- agents/      <- user-level Claude/OpenCode assets
                                                     +-- commands/
                                                     |     +-- claude/
                                                     |     +-- opencode/
                                                     |     (Codex actions live in system/codex-skills/)
                                                           |
  aether update (in ANY repo)  <---------------------------+
  aether lay-eggs (initializes colony)

v
any-repo/.aether/ (LOCAL ONLY)
+-- data/            <- colony state
+-- dreams/          <- session notes
+-- locks/           <- runtime locks
+-- QUEEN.md         <- repo-specific wisdom
+-- skills/          <- custom repo skills only
```

---

## Key Directories

| Directory | Purpose | Syncs to Hub |
|-----------|---------|--------------|
| `.codex/agents/` | Codex agent definitions (TOML) | -> `~/.aether/system/codex/` |
| `AGENTS.md` | Codex system instructions | Template-generated at setup |
| `.aether/` (system files) | Source of truth for workers, utils, docs | -> `~/.aether/system/` |
| `.aether/data/` | Colony state | **NEVER touched** |
| `.aether/dreams/` | Session notes | **NEVER touched** |

---

## Codex-Specific Conventions

### Public Skills and Direct CLI

Pick an ant skill in Codex to start the matching workflow. The skill reads Aether's
instructions; the `aether` executable still owns state, checks, and authorization.

Aether's Codex actions use skills, with exactly nine public names:

| Skill | Workflow |
|-------|----------|
| `$ant-init` | Start a colony |
| `$ant-discuss` | Clarify intent |
| `$ant-oracle` | Research a concern |
| `$ant-colonize` | Survey existing code |
| `$ant-plan` | Plan phases |
| `$ant-build` | Build a phase |
| `$ant-continue` | Verify and advance |
| `$ant-swarm` | Route work or watch workers |
| `$ant-seal` | Seal and retain work for review |

The remaining 55 action skills and native-worker parity belong to later inserted
phases. These nine names establish entrypoints, not complete lifecycle parity.

The selected installation root is `~/.codex/skills/aether/`, qualified with Codex
CLI 0.154.0. Each public `ant-<command>/SKILL.md` reads private support relative to
its own installed file, not the working directory:

- `support/aether-colony-creation.md`
- `support/aether-colony-research.md`
- `support/aether-colony-build-cycle.md`

These three ordinary files are private support, not public helper skills. Worker
skills still arrive automatically in runtime dispatch briefs. Start a fresh Codex
session after skill files are copied or removed; unchanged updates need no refresh.

When a user types a literal `aether ...` passthrough command such as `status`,
`update`, `focus`, `pheromones`, or `reference-list`, execute that exact command
first. The installed binary and `aether --help` are the runtime source of truth.

For the nine lifecycle actions above, run or inspect
`aether command-guide <command> --platform codex` and follow the matching public
skill and its private support. If the user explicitly says raw, exact,
no-interview, or no-orchestration, execute the requested CLI command directly.
Do not reinterpret a literal passthrough command as a vague workflow request.

For lifecycle shell execution, prefer `AETHER_OUTPUT_MODE=visual aether ...`
unless the user explicitly wants JSON. Preserve exact arguments. Do not preface
literal passthrough execution with repo archaeology or skill narration; the CLI
output is primary, with at most one short sentence of extra explanation.

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

The shared skill sources live in `.aether/skills/`.
Worker skills are published to `~/.aether/system/skills/` and matched
in-process by the Go runtime's skill-matching logic against worker role, workspace files,
and package manifests -- there is no standalone `skill-*` CLI surface (Phase 191 deleted the
8 CLI wrappers as dead code; the matching and injection logic itself is unconditionally
load-bearing and unaffected).

Skills are not a separate Codex plugin bundle. Codex agent definitions remain in
`.codex/agents/*.toml`, and the Go CLI handles skill indexing, matching, and injection.

Skills are automatically matched and injected into worker prompts during `build`,
`colonize`, `plan`, and `continue` dispatches.

The nine public Codex skills and their three private support bodies are
separate from worker skill injection. Their ordinary `support/*.md` files
coordinate interviews, synthesis, dispatch and summaries only as the runtime
guide permits. Keep those sources aligned with `.aether/commands/*.yaml` and
`aether command-guide`. Internal support and worker IDs stay unchanged.

### Pheromone Signals

Active pheromone signals are automatically injected into worker prompts during
`build`, `colonize`, `plan`, and `continue` dispatches.

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

In Codex chat, use `$ant-init "Build feature X"`, `$ant-colonize`, `$ant-plan`,
`$ant-build 1`, `$ant-continue`, and `$ant-seal`. The executable route remains:

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

# Live worker view and autonomous research loop
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
aether memory-details   # What the colony has learned: wisdom, lessons waiting to be promoted, lessons put aside, and recent failures
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
aether data-clean        # Clean test artifacts
```

---

## Agent Reference

All 27 agents are defined in `.codex/agents/*.toml`:

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
| Specialist | Medic | `aether-medic.toml` | Colony health diagnosis and repair |
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
| Colony workers | Aether repo `.aether/` `workers.md` | Shared across all platforms through the hub |
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

# Verify all 27 agents exist
ls .codex/agents/*.toml | wc -l  # Should be 27
```

### Publishing Changes

Authoritative runbook: `.aether/docs/publish-update-runbook.md`

```bash
# 1. Edit files in .codex/ or .aether/
# 2. Commit
git add .
git commit -m "update codex agent definitions"

# 3. Publish to the hub (recommended)
aether publish

# 4. In other repos, pull updates
aether update --force
```

Runtime note:
- `aether publish` is the primary publish command. It builds the binary, syncs companion files to the hub, and verifies version agreement automatically.
- `aether install --package-dir "$PWD"` still works for backward compatibility but does not include version verification.
- `aether publish --channel dev` publishes to the dev channel (`~/.aether-dev/`) for isolated source development.
- `aether integrity` validates the full release pipeline chain (source, binary, hub, companion files, downstream simulation).
- `aether update` automatically runs stale publish detection -- critical stale publishes block the update with a recovery command.
- `aether update` in other repos only syncs from the shared hub. It does not publish local source changes, and without `--force` it can leave stale Aether-managed files behind.
- `aether update --force --download-binary` is the published-release path when you also need the release runtime binary.
- For isolated source-development on this machine, publish the dev channel instead: `aether publish --channel dev --binary-dest "$HOME/.local/bin"`, then use `aether-dev update --force` in target repos. This keeps `~/.aether-dev/` and `aether-dev` separate from the public stable runtime.
- `.aether/version.json` is the source-checkout release version file. `npm/package.json` must match it exactly for published releases.
- If `aether update --force` shows `Commands (claude)` or `Commands (opencode)` as `0 copied, 0 unchanged`, the hub publish is incomplete. Republish from the Aether repo first, then rerun `aether update --force` in the target repo.
- If the change modifies publish/install/update logic, bootstrap once with `go run ./cmd/aether publish --channel stable --binary-dest "$HOME/.local/bin"`.
- Published release flow: bump `.aether/version.json` and `npm/package.json` to the same version, push the commit, then push tag `vX.Y.Z`.

---

## Comparison with Claude Code and OpenCode

### Key Differences

| Aspect | Claude Code | OpenCode | Codex |
|--------|-------------|----------|-------|
| System prompt file | `CLAUDE.md` | `.opencode/OPENCODE.md` | `AGENTS.md` |
| Agent format | Markdown `.md` | Markdown `.md` | TOML `.toml` |
| Agent location | `.claude/agents/ant/` | `.opencode/agents/` | `.codex/agents/` |
| Public actions | 64 slash commands | 64 slash commands | Nine `$ant-*` skills + `aether` CLI |
| Command location | `.claude/commands/ant/` | `.opencode/commands/ant/` | `~/.codex/skills/aether/ant-*/SKILL.md` |
| Agent metadata | In markdown header | In markdown header | TOML keys |
| Hub sync path | global Claude home + hub | global OpenCode home + hub | global Codex home + hub |
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
| Start colony | `/ant-init "goal"` | `/ant-init "goal"` | `$ant-init "goal"` |
| Build phase | `/ant-build 1` | `/ant-build 1` | `$ant-build 1` |
| Check status | `/ant-status` | `/ant-status` | `aether status` |
| Focus signal | `/ant-focus "area"` | `/ant-focus "area"` | `aether focus "area"` |
| Update Aether | `/ant-update` | `/ant-update` | `aether update` |
| Seal colony | `/ant-seal` | `/ant-seal` | `$ant-seal` |
| Deep research | `/ant-oracle` | `/ant-oracle` | `$ant-oracle` |
| View pheromones | `/ant-pheromones` | `/ant-pheromones` | `aether pheromone-display` |

### Next-step hints

The runtime adapts its "Next Up" suggestions to the active platform: Claude
Code and OpenCode see slash wrappers (`/ant-continue`); Codex sees `$ant-continue`
for a supported public action and raw CLI text for other actions. Runtime
command IDs and exact arguments retain their meaning. Detection reads
`AETHER_PLATFORM` first, then the active-platform signals, then the process
tree. If a nested or wrapped invocation is ever misdetected and you see
`/ant-*` suggestions, export the platform explicitly:

```bash
export AETHER_PLATFORM=codex
```

---

*Updated for Aether v1.0.83 — 2026-09-21*
