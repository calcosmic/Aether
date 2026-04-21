# AGENTS.md -- Aether Development Guide (Codex CLI)

> **Current Version:** v1.0.17
> **Last Updated:** 2026-04-21
> **Platform:** Codex CLI (OpenAI)

This file provides project-level instructions for Codex CLI, equivalent to
`CLAUDE.md` for Claude Code. Aether supports three platforms: Claude Code,
OpenCode, and Codex CLI.

## Platform Policy

- **Primary platforms:** Claude Code and OpenCode. These are the main maintained user surfaces.
- **Secondary platform:** Codex CLI. Codex has best-effort support for the direct `aether` workflow.
- **What that means for Codex:** keep the native CLI lifecycle, install/update flow, state integrity, and worker dispatch safe and usable. Codex UX polish and wrapper semantics may lag behind Claude/OpenCode.

---

## Quick Reference

| What | Count/Status |
|------|--------------|
| Version | v1.0.17 |
| Agent definitions | 24 (TOML in `.codex/agents/`) |
| Skills | 28 (10 colony + 18 domain) |
| Go binary | `aether` CLI (Go binary in cmd/) |
| Verification | `go test ./...` and `go test ./... -race` clean |
| Architecture doc | `RUNTIME UPDATE ARCHITECTURE.md` |

---

## How Aether Works in Codex CLI

Codex CLI does not support slash commands. All colony operations use the `aether`
CLI binary directly or natural language prompts. There is no equivalent of
typing `/ant:build 1` -- instead you run:

```bash
aether build 1
aether plan
aether init "Build feature X"
aether continue
aether status
```

When the user types a literal `aether ...` command in Codex, execute that exact
CLI command first. Do not reinterpret it as a fuzzy workflow prompt, and use
`aether --help` as the runtime source of truth if markdown docs disagree. For
lifecycle commands run through Codex shell execution, prefer
`AETHER_OUTPUT_MODE=visual aether ...` unless the user explicitly wants JSON.
Do not preface literal commands with repo archaeology, skill narration, or
"I'm checking..." commentary. The CLI output is primary; your own wrapper
should be zero or one short sentence.

Agent definitions live in `.codex/agents/*.toml` (TOML format) and Codex reads
them as part of its agent discovery system.

## UX Architecture

Codex gets its colony identity entirely from the Go runtime visual renderer
(`cmd/codex_visuals.go`). Unlike Claude/OpenCode, Codex does NOT use wrapper
markdown — the CLI output IS the presentation.

### Caste Identity System

Each worker caste has:
- **Emoji prefix** — decorative icon (e.g., `🔨` for Builder, `👁️` for Watcher)
- **ANSI-colored label** — the primary identity, colored by caste (e.g., "Builder" in yellow, "Watcher" in cyan)
- **Deterministic name** — hash-based per caste+task (e.g., "Mason-67", "Keen-6")

Visual format: `🔨 Builder Mason-67  Task description`

Color detection uses `NO_COLOR`, `AETHER_FORCE_COLOR`, `CLICOLOR_FORCE`, and TTY
checks. In non-TTY or JSON mode, colors are disabled and only plain text appears.

### Stage Markers

Build and continue output uses stage transition markers:
```
── Context ──
── Tasks ──
── Dispatch ──
── Verification ──
── Housekeeping ──
── Next Phase ──
── Colony Complete ──
```

### Output Modes

| Mode | When | Content |
|------|------|---------|
| Visual | `AETHER_OUTPUT_MODE=visual` or TTY | ANSI banners, emojis, colors, stage markers, context-clear guidance |
| JSON | `AETHER_OUTPUT_MODE=json` or piped | Structured data envelopes for programmatic use |

### What Codex Does NOT Get

Claude/OpenCode wrappers add colony framing (Queen persona, pre-build context,
post-build narration) on top of the runtime output. Codex sees only the raw
runtime output. To improve Codex UX, change `cmd/codex_visuals.go` — not the
wrapper markdown.

---

## Architecture Overview

```
+------------------------------------------------------------------+
|                     AETHER REPO (this repo)                       |
|                                                                   |
|   cmd/                 <- Go source code (primary)               |
|   +-- main.go         CLI entry point                            |
|   +-- *.go            Command implementations (80+ subcommands)  |
|                                                                   |
|   pkg/                 <- Shared Go packages                     |
|   +-- agent/          Agent pool, spawn tree, curation ants      |
|   +-- downloader/     Binary download + extraction               |
|   +-- events/         Event bus with TTL                         |
|   +-- exchange/       XML import/export                          |
|   +-- graph/          Knowledge graph persistence                |
|   +-- memory/         Learning pipeline, instincts, promotion    |
|   +-- storage/        JSON store, file locking                   |
|                                                                   |
|   .aether/             <- Companion files (embedded in release binaries) |
|   +-- agents-claude/   Claude packaging mirror                  |
|   +-- agents-codex/    Codex packaging mirror                   |
|   +-- skills/          Shared skill source                       |
|   +-- skills-codex/    Codex-installed skill mirror              |
|   +-- docs/            Distributed documentation                 |
|   +-- templates/       Colony state, pheromones, etc.           |
|                                                                   |
|   .aether/data/        <- LOCAL ONLY (gitignored)                |
|   .aether/dreams/      <- LOCAL ONLY (gitignored)                |
|                                                                   |
|   .claude/commands/ant/ <- Slash commands (Claude Code)          |
|   .claude/agents/ant/   <- Agent definitions (Claude Code)       |
|   .opencode/commands/ant/ <- Slash commands (OpenCode)           |
|   .opencode/agents/     <- Agent definitions (OpenCode)          |
|   .codex/agents/        <- Agent definitions (Codex CLI)         |
|                                                                   |
|   ~/.aether/           <- HUB (cross-colony, user-level)         |
|   +-- system/          Companion file source (populated by install)|
|   +-- QUEEN.md         (wisdom + user preferences)               |
|   +-- hive/            (Hive Brain -- cross-colony wisdom)       |
|   |   +-- wisdom.json  (200-entry cap, LRU eviction)            |
|   +-- eternal/         (hive memory -- high-value signals)       |
|                                                                   |
+------------------------------------------------------------------+
```

Colony-prime assembles worker context from: QUEEN.md wisdom, eternal memory,
pheromone signals, phase learnings, key decisions, blocker flags, user preferences,
parallel mode, and context capsule -- all within a token budget (see Token Budget).

**See `RUNTIME UPDATE ARCHITECTURE.md` for complete distribution flow.**

---

## Development Workflow

### Editing System Files

| What you're changing | Where to edit | Why |
|---------------------|---------------|-----|
| workers.md | `.aether/workers.md` | Source of truth |
| Go commands | `cmd/` | Go source code |
| User docs | `.aether/docs/` | Distributed directly |
| Codex agent definitions | `.codex/agents/*.toml` | Codex CLI agent format |
| Codex packaging mirror | `.aether/agents-codex/*.toml` | Must stay in sync with `.codex/agents/*.toml` |
| Codex skill mirror | `.aether/skills-codex/` | Codex-installed skill bundle |
| Claude Code commands | `.claude/commands/ant/` | Claude Code commands |
| OpenCode commands | `.opencode/commands/ant/` | OpenCode commands |
| Claude Code agents | `.claude/agents/ant/` | Claude Code agents |
| Agent mirror (packaging) | `.aether/agents-claude/` | Must stay in sync with `.claude/agents/ant/` |
| OpenCode agents | `.opencode/agents/` | OpenCode worker definitions |
| Your notes | `.aether/dreams/` | Never distributed |
| Dev docs | `.aether/docs/known-issues.md` | Distributed |

### Core Workflow

Codex currently exposes the core colony lifecycle directly:

```bash
aether lay-eggs
aether init "Build feature X"
aether discuss
aether colonize
aether plan
aether run --dry-run
aether swarm --watch
aether build 1
aether continue
aether seal
aether oracle "release concern"
```

Codex also exposes compatibility entrypoints for the flows users reach for most:
`aether run` for autopilot-style build/continue looping, `aether watch` for live
worker visibility, and `aether oracle` for the autonomous Oracle RALF research loop.
Treat these as supported best-effort compatibility entrypoints within Codex's
native CLI workflow, rather than strict mirrors of Claude/OpenCode slash-command UX.

### Publishing Changes

```bash
# 1. Edit files in .aether/ or .codex/agents/
vim .aether/workers.md

# 2. Commit changes
git add .
git commit -m "your message"

# 3. Refresh the installed hub files from this source checkout
aether install --package-dir "$PWD"

# 4. In other repos, pull companion-file updates
aether update
```

Runtime note:
- `aether install --package-dir "$PWD"` is the step that publishes unreleased Go runtime fixes on this machine because it refreshes the shared hub and, from a source checkout, rebuilds the shared `aether` binary.
- `aether update` in other repos only syncs companion files by default. It does not publish an unreleased local runtime change.
- `aether update --download-binary` can fetch a published release binary, but it cannot pull an unreleased local source change.

---

## Aether CLI Commands

Since Codex CLI has no slash commands, all colony operations use the `aether` CLI.

### Setup and Getting Started

| Command | Purpose |
|---------|---------|
| `aether lay-eggs` | Set up Aether in this repo (one-time, creates .aether/) |
| `aether init "Build feature X"` | Start a colony with a goal |
| `aether discuss` | Surface unresolved intent questions before planning and store clarifications |
| `aether colonize` | Analyze existing codebase |
| `aether plan` | Generate project phases |
| `aether build <phase>` | Execute a phase with Codex worker dispatch |
| `aether continue` | Verify work, extract learnings, advance |
| `aether run` | Autopilot the remaining build/continue loop |
| `aether swarm [problem]` | Route to the right explicit workflow step or watch active workers |
| `aether oracle [topic]` | Run or inspect the autonomous Oracle RALF research loop |

### Pheromone Signals

| Command | Priority | Purpose |
|---------|----------|---------|
| `aether focus "<area>"` | normal | Guide colony attention |
| `aether redirect "<pattern>"` | high | Hard constraint -- avoid this |
| `aether feedback "<note>"` | low | Gentle adjustment |
| `aether pheromones` | -- | View all active signals |
| `aether export-signals` | -- | Export signals to XML |
| `aether import-signals --file <path>` | -- | Import signals from XML |

### Status and Monitoring

| Command | Purpose |
|---------|---------|
| `aether status` | Colony dashboard |
| `aether watch` | Live worker activity compatibility view |
| `aether phase [N]` | View phase details |
| `aether flags` | List active flags |
| `aether flag "<title>"` | Create a flag |
| `aether history` | Browse colony events |
| `aether memory-details` | Drill-down memory view |
| `aether patrol` | System health check |

### Session Management

| Command | Purpose |
|---------|---------|
| `aether resume` | Restore colony context from handoff (`resume-colony` alias) |

### Lifecycle

| Command | Purpose |
|---------|---------|
| `aether seal` | Seal colony (Crowned Anthill) |
| `aether update` | Update system files from hub |

### Advanced

| Command | Purpose |
|---------|---------|
| `aether preferences "text"` | Set user preference |
| `aether insert-phase` | Insert phase into plan |
| `aether data-clean` | Clean test artifacts from data files |
| `aether parallel-mode get` | Show the active parallel execution mode |
| `aether parallel-mode set <mode>` | Change between `in-repo` and `worktree` execution |
| `aether skill-list` | List installed skills |
| `aether skill-match --role <role> --task "<task>"` | Match skills for a worker |
| `aether skill-inject --role <role> --task "<task>"` | Render injected skill context |

### Typical Workflow

```bash
# First time in a repo
aether lay-eggs

# Starting a colony
aether init "Build feature X"
aether discuss                           # optional but recommended before planning
aether colonize                          # if existing codebase
aether plan
aether watch                             # optional live worker view
aether focus "security"                  # optional guidance
aether build 1
aether continue
aether build 2                           # repeat until complete

# Or let Codex run the loop
aether run --max-phases 2

# After a session break
aether resume
aether status

# Research a release concern
aether oracle "release parity"

# After completing a colony
aether seal
aether init "next project goal"
```

---

## Key Directories

### .codex/ (Codex CLI Agent Definitions)

```
.codex/
+-- agents/                # 24 agent definitions (TOML format)
|   +-- aether-builder.toml
|   +-- aether-watcher.toml
|   +-- aether-scout.toml
|   +-- ...
```

Each TOML file defines an agent with: name, description, nickname_candidates,
and developer_instructions. Codex reads these for agent discovery.

### .aether/ (Source of Truth)

```
.aether/
+-- workers.md           # Worker definitions, spawn protocol
+-- utils/               # Runtime utilities
|   +-- oracle/oracle.md # Oracle loop instructions
|   +-- queen-to-md.xsl  # XSL transform for queen wisdom export
+-- skills/              # colony/ (10) + domain/ (18) skill definitions
+-- templates/           # 12 templates (colony-state, pheromones, etc.)
+-- docs/                # Distributed documentation
+-- exchange/            # XML exchange modules (pheromone-xml, wisdom-xml)
+-- agents-claude/       # Claude agent mirror used for packaging
+-- agents-codex/        # Codex agent mirror used for packaging
+-- skills-codex/        # Codex-installed skill mirror
+-- data/                # LOCAL ONLY (never distributed)
|   +-- COLONY_STATE.json  # Colony state with phase tracking + parallel_mode
|   +-- pheromones.json
|   +-- constraints.json
|   +-- midden/          # Failure tracking
|   +-- survey/          # Territory survey results
+-- dreams/              # LOCAL ONLY (session notes)
+-- oracle/              # LOCAL ONLY (deep research)
```

### Split Playbooks (Reliability)

High-length commands are split into smaller execution playbooks:

- `.aether/docs/command-playbooks/build-prep.md`
- `.aether/docs/command-playbooks/build-context.md`
- `.aether/docs/command-playbooks/build-wave.md`
- `.aether/docs/command-playbooks/build-verify.md`
- `.aether/docs/command-playbooks/build-complete.md`
- `.aether/docs/command-playbooks/continue-verify.md`
- `.aether/docs/command-playbooks/continue-gates.md`
- `.aether/docs/command-playbooks/continue-advance.md`
- `.aether/docs/command-playbooks/continue-finalize.md`

Authority note:
- In Claude Code, `.claude/commands/ant/build.md` and `.claude/commands/ant/continue.md` are orchestrators only.
- Build/continue execution behavior is defined in `.aether/docs/command-playbooks/*.md`.
- OpenCode maintains separate command specs in `.opencode/commands/ant/*.md`.
- Codex CLI uses the `aether` CLI binary directly (no slash command mechanism).
- Agent parity model: `.claude/agents/ant/*.md` is canonical, `.aether/agents-claude/*.md` is a byte-identical packaging mirror, `.opencode/agents/*.md` maintains structural parity, and `.codex/agents/*.toml` is a supported Codex translation aligned on core role intent and maintained on a best-effort basis.

---

## The 24 Agents

| Tier | Agent | TOML File | Role |
|------|-------|-----------|------|
| Core | Builder | `aether-builder.toml` | Implements code, TDD-first |
| Core | Watcher | `aether-watcher.toml` | Tests, validates, quality gates |
| Orchestration | Queen | `aether-queen.toml` | Orchestrates phases, spawns workers |
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

## Pheromone System

User-colony communication via signals:

| Signal | CLI Command | Priority | Use For |
|--------|-------------|----------|---------|
| FOCUS | `aether focus "<area>"` | normal | "Pay attention here" |
| REDIRECT | `aether redirect "<avoid>"` | high | "Don't do this" (hard constraint) |
| FEEDBACK | `aether feedback "<note>"` | low | "Adjust based on this observation" |

**Before builds:** FOCUS + REDIRECT to steer
**After builds:** FEEDBACK to adjust
**Hard constraints:** REDIRECT (will break)
**Gentle nudges:** FEEDBACK (preferences)

**Viewing Signals:**
- `aether pheromones` -- Full table of all active signals
- `pheromone-display` subcommand -- Formatted output with strength % and decay

**Signal Injection:**
- Colony-prime injects active signals into worker prompts via `prompt_section`
- Builder, Watcher, and Scout agents have `pheromone_protocol` sections that instruct them how to act on injected signals
- Signals are grouped by type (FOCUS, REDIRECT, FEEDBACK) in the injected prompt section
- In Codex CLI, active pheromone signals are automatically injected into worker prompts during `build`, `colonize`, and `plan` dispatches

**Content Deduplication (v2.0):**
- Each signal gets a SHA-256 `content_hash` on creation
- Writing a duplicate (same type + content hash) reinforces the existing signal instead of creating a new one
- `suggest-analyze` deduplicates suggestions against existing active signals and session suggestions

**Prompt Injection Sanitization (v2.0):**
- Pheromone content is sanitized before storage: XML structural tags rejected, angle brackets escaped, shell injection patterns blocked
- Content capped at 500 characters

**Exchange:**
- `aether export-signals` -- Export pheromone signals to XML for cross-colony sharing
- `aether import-signals --file <path>` -- Import pheromone signals from XML

**Files:**
- `.aether/data/pheromones.json` -- Active signals
- `.aether/data/constraints.json` -- Focus areas and constraints (legacy, eventual deprecation)
- `.aether/docs/pheromones.md` -- Full guide

---

## Skills System

Skills provide reusable behavior modules and domain knowledge that workers can load
on demand. Two categories:

- **Colony skills** (10) -- Behavioral patterns that shape how workers operate
  (e.g., TDD discipline, error handling conventions, commit style)
- **Domain skills** (18) -- Technical knowledge for specific frameworks, languages,
  or tools (e.g., React patterns, Go idioms, database optimization)

### Where Skills Live

| Location | Purpose |
|----------|---------|
| `.aether/skills/` | Source of truth (packaged with Aether) |
| `~/.aether/skills/` | Installed skills (hub-level, shared across colonies) |
| `~/.aether/skills/domain/` | Custom user-created domain skills |

### How Matching Works

1. Colony-prime builds a skills index via `skill-index` (cached for performance)
2. `skill-match` scores each skill against the current worker using:
   - Worker role (builder, watcher, etc.)
   - Active pheromone signals (FOCUS/REDIRECT)
   - `skill-detect` patterns matched against the codebase
3. Top 3 colony skills + top 3 domain skills are selected per worker

### Skill Injection

- Own 8K character budget (independent of the colony-prime token budget)
- Injected into builder and watcher prompts
- `skill-inject` assembles matched skills into a prompt section
- In Codex CLI, skills are automatically matched and injected into worker prompts during `build`, `colonize`, and `plan` dispatches

### Subcommands

| Subcommand | Purpose |
|------------|---------|
| `skill-index` | Build/read cached skills index |
| `skill-detect` | Detect domain skills matching codebase |
| `skill-match` | Match skills to worker by role + task + pheromones |
| `skill-inject` | Load matched skills into prompt section |
| `skill-list` | List all installed skills |
| `skill-parse-frontmatter` | Parse SKILL.md frontmatter to JSON |
| `skill-diff` | Compare user skill with shipped version |
| `skill-cache-rebuild` | Force rebuild of index cache |

### Custom Skills

- Manual creation: add a `SKILL.md` file to `~/.aether/skills/domain/`
- Use `aether skill-list`, `aether skill-match`, and `aether skill-diff` to validate custom skill behavior

### Update Safety

- Repo-local Codex skill copies are preserved during normal `aether update`
- `aether update --force` refreshes tracked files from the hub and may discard local edits
- User skills stored outside the tracked Aether paths remain untouched

---

## Token Budget

Colony-prime assembles worker context within a character budget to avoid prompt bloat:

| Mode | Budget | When |
|------|--------|------|
| Normal | 8,000 chars | Default |
| Compact | 4,000 chars | `--compact` flag or auto-detected |

**Retention priority** (highest packed first):
1. Blockers (never trimmed)
2. Pheromone signals
3. User preferences
4. Instincts
5. Colony state
6. Hive wisdom
7. Key decisions
8. Phase learnings

---

## Hive Brain (Cross-Colony Wisdom)

The Hive Brain stores generalized wisdom derived from colony instincts, scoped by
domain, and shared across all colonies on the same machine.

### Storage

```
~/.aether/hive/
+-- wisdom.json          # Cross-colony wisdom (200-entry cap, LRU eviction)
```

### Subcommands

| Subcommand | Purpose |
|------------|---------|
| `hive-init` | Initialize hive directory and empty wisdom.json |
| `hive-store` | Store wisdom entry with dedup and 200-cap enforcement |
| `hive-read` | Read wisdom with domain filtering and confidence threshold |
| `hive-abstract` | Generalize repo-specific instinct into cross-colony wisdom |
| `hive-promote` | Orchestrate abstract + store pipeline |

### Multi-Repo Confidence Boosting

| Repos Confirming | Confidence |
|-----------------|------------|
| 2 repos | 0.70 |
| 3 repos | 0.85 |
| 4+ repos | 0.95 |

Confidence is never downgraded. During `aether seal`, instincts with confidence
>= 0.8 are promoted to the Hive Brain (non-blocking).

---

## User Preferences

Stored in the hub `~/.aether/QUEEN.md` under the `## User Preferences` section:

- `aether preferences "text"` -- Add a user preference
- `aether preferences --list` -- List all user preferences

Colony-prime injects user preferences into worker context. Max 500 characters
per entry.

---

## Quality Gates

| Gate | Agent | Runs When | Purpose |
|------|-------|-----------|---------|
| Security | Gatekeeper | After verification | Scans for exposed secrets, debug artifacts |
| Quality | Auditor | After Gatekeeper | Analyzes code quality metrics |
| Coverage | Probe | After Auditor | Analyzes test coverage gaps |
| Performance | Measurer | After Probe | Performance analysis and metrics |

---

## Midden System (Failure Tracking)

- `.aether/data/midden/midden.json` -- Failure records
- `midden-write` -- Log a failure
- `midden-recent-failures` -- Query recent failures
- `midden-review` -- Review unacknowledged entries by category
- `midden-acknowledge` -- Mark entries as addressed
- `aether data-clean` -- Remove test artifacts from data files

---

## Milestone Names

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
# Run Go tests
go test ./...

# Run Go tests with race detection
go test ./... -race

# Verify Go binary builds
go build ./cmd/aether

# Run Go vet
go vet ./...

# Verify goreleaser config
goreleaser check

# Build snapshot (no tag required)
goreleaser build --snapshot --clean

# Verify binary works
aether version
```

---

## Session Freshness Detection

All stateful commands use timestamp verification to detect stale sessions:

1. Capture `SESSION_START=$(date +%s)` before spawning agents
2. Check file freshness with `session-verify-fresh --command <name>`
3. Auto-clear stale files or prompt user based on command type
4. Verify files are fresh after spawning

**Protected Commands** (never auto-clear): `init`, `seal`

---

## Session Recovery

On the first message of a new conversation, check if `.aether/data/session.json`
exists. If it does, read briefly for `colony_goal` and display:

```
Previous colony session detected: "{goal}"
Run `aether resume` to restore context, or continue with a new topic.
```

Do NOT auto-restore -- wait for the user to explicitly request it.

---

## Wisdom Pipeline

The core learning loop: observations flow through the system and become reusable
wisdom.

| Stage | Subcommand | Output |
|-------|-----------|--------|
| 1. Observe | `memory-capture --type "learning" --content "..."` | Records observation |
| 1a. Trust score | `trust-score-compute` | Weighted trust score (40/35/25, 7 tiers) |
| 1b. Event bus | `event-bus-publish` | JSONL event bus with TTL |
| 2. Auto-promote | (internal) | Triggers after 2 observations |
| 3. Instinct | `instinct-create` | Stores in `instincts.json` |
| 4. QUEEN.md | `queen-promote` | Writes to QUEEN.md |
| 5. Inject | colony-prime | Injected into worker context |
| 6. Hive store | `hive-promote` | Abstracts to hive (confidence >= 0.8) |
| 7. Hive read | `hive-read` | Cross-colony retrieval by domain |

**See `.aether/docs/structural-learning-stack.md` for full documentation.**

---

## Structural Learning Stack

Memory consolidation pipeline with trust scoring, graph relationships, and
automated curation.

Key additions:
- Trust scoring engine (40/35/25 weighted, 60-day half-life, 7 tiers)
- JSONL event bus with pub/sub and TTL cleanup
- Standalone instinct storage with full provenance
- 8 curation ants with orchestrated execution
- Lifecycle integration: phase-end at `aether continue`, full at `aether seal`

### Curation Ants

| Ant | Role |
|-----|------|
| orchestrator | Coordinates curation pipeline execution |
| archivist | Archives and retrieves historical observations |
| critic | Evaluates instinct quality and confidence |
| herald | Broadcasts high-confidence instincts to hive |
| janitor | Cleans stale events and expired TTL entries |
| librarian | Indexes and catalogs instinct relationships |
| nurse | Heals low-confidence instincts with supporting evidence |
| scribe | Records curation decisions and audit trail |
| sentinel | Guards against instinct corruption and conflicts |

---

## Parallel Execution Modes

Aether supports two parallel execution strategies:

| Mode | Value | Description |
|------|-------|-------------|
| In-repo (default) | `"in-repo"` | Workers share the working tree directly |
| Worktree | `"worktree"` | Each worker gets an isolated git worktree, then file and pheromone changes sync back into the main root |

- `aether parallel-mode get` -- Read current mode
- `aether parallel-mode set <mode>` -- Change mode

**In-repo** is simpler for most projects. **Worktree** provides isolation for
larger tasks with multiple workers touching different files, while keeping the
main root as the final source of truth after sync-back.

---

## The Core Insight

The system's pieces are now **connected**:
- Pheromones update context (colony-prime injects signals into worker prompts)
- Decisions become pheromones (auto-emit during builds)
- Learnings become instincts (observation to promotion pipeline)
- Midden affects behavior (threshold auto-REDIRECT)
- Hive Brain crosses colony boundaries (domain-scoped wisdom -> colony-prime)
- Instincts promote to hive at seal (confidence >= 0.8 -> hive-promote)
- Multi-repo confirmation boosts confidence (2 repos = 0.7, 4+ = 0.95)
- User preferences shape worker behavior (QUEEN.md -> colony-prime)
- Codex uses the direct `build` -> `continue` -> `seal` lifecycle

**The ongoing challenge is maintenance** -- keeping documentation accurate,
data files clean, and test coverage comprehensive as features evolve.

---

## Platform Cross-Reference

| Concept | Claude Code | OpenCode | Codex CLI |
|---------|-------------|----------|-----------|
| Project instructions | `CLAUDE.md` | `.opencode/OPENCODE.md` | `AGENTS.md` |
| Agent definitions | `.claude/agents/ant/*.md` | `.opencode/agents/*.md` | `.codex/agents/*.toml` |
| Slash commands | `.claude/commands/ant/*.md` | `.opencode/commands/ant/*.md` | N/A (use `aether` CLI) |
| Rules | `.claude/rules/*.md` | `.opencode/rules/*.md` | `AGENTS.md` sections |

---

*Updated for Aether v1.0.17 -- 2026-04-21*
