# CLAUDE.md — Aether Development Guide

> **Current Version:** v1.0.18
> **Last Updated:** 2026-04-21

---

## Quick Reference

| What | Count/Status |
|------|--------------|
| Version | v1.0.18 |
| Slash commands | 50 (Claude) + 50 (OpenCode); Codex uses native CLI + 25 TOML agents |
| Agent definitions | 25 |
| Skills | 29 (11 colony + 18 domain) |
| Go binary | `aether` CLI (Go binary in cmd/) |
| Tests | 2900+ passing |
| Architecture doc | `RUNTIME UPDATE ARCHITECTURE.md` |

## Platform Policy

- **Primary platforms:** Claude Code and OpenCode. These are the main maintained user surfaces.
- **Secondary platform:** Codex CLI. Codex has best-effort support for the direct `aether` workflow.
- **Expectation:** keep Claude/OpenCode command and agent UX aligned first. Keep Codex safe, usable, and accurate about its native CLI capabilities.

## UX Architecture

Aether uses a hybrid UX model: the Go runtime owns truth, platform wrappers own presentation.

### Ownership Model

| Layer | Owner | What |
|-------|-------|------|
| State mutations | Go runtime (`cmd/`) | Colony state, phase transitions, verification, gating |
| Visual rendering | Go runtime (`cmd/codex_visuals.go`) | Banners, progress bars, caste identity, stage markers |
| Colony framing | Wrapper markdown (`.claude/`, `.opencode/`) | Queen persona, narration, pacing, pre/post-build context |
| Codex UX | Go runtime only | No wrapper markdown — Codex is runtime-native |

### Caste Identity System

Each worker caste has three visual components:
1. **Emoji prefix** — decorative icon (e.g., `🔨` for Builder, `👁️` for Watcher)
2. **ANSI-colored label** — the primary identity, colored by caste (e.g., "Builder" in yellow)
3. **Deterministic name** — hash-based per caste+task (e.g., "Mason-67")

Format: `🔨 Builder Mason-67  Task description`

Color maps in `cmd/codex_visuals.go`: `casteColorMap` (ANSI codes), `casteEmojiMap` (single emoji), `casteLabelMap` (human-readable name). Functions: `casteIdentity()`, `casteLabel()`, `casteEmoji()`.

### Stage Markers

Build and continue output uses `── Stage Name ──` separators between sections (Context, Tasks, Dispatch, Verification, Housekeeping, Next Phase, Colony Complete).

### YAML Source Chain

```
.aether/commands/*.yaml          ← Source definitions (name, runtime command, guardrails)
    ↓ (generation)
.claude/commands/ant/*.md        ← Claude Code wrappers
.opencode/commands/ant/*.md      ← OpenCode wrappers
    ↓ (delegation)
cmd/codex_*.go                   ← Go runtime (authoritative)
```

### Wrapper-Runtime Contract

Full contract documented in `.aether/docs/wrapper-runtime-ux-contract.md`. Key rules:
- Wrappers may add colony framing and narration but must not mutate state
- Wrappers must not duplicate verification or gating logic
- Codex gets UX improvements through the runtime renderer only

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     AETHER REPO (this repo)                      │
│                                                                  │
│   cmd/                 ← Go source code (primary)              │
│   ├── main.go         CLI entry point                          │
│   └── *.go            Command implementations (80+ subcommands)│
│                                                                  │
│   pkg/                 ← Shared Go packages                    │
│   ├── agent/          Agent pool, spawn tree, curation ants    │
│   ├── downloader/     Binary download + extraction             │
│   ├── events/         Event bus with TTL                       │
│   ├── exchange/       XML import/export                        │
│   ├── graph/          Knowledge graph persistence              │
│   ├── memory/         Learning pipeline, instincts, promotion  │
│   └── storage/        JSON store, file locking                 │
│                                                                  │
│   .aether/             ← Companion files (distributed via npm)  │
│   ├── commands/*.yaml   Slash command source definitions        │
│   ├── agents-claude/    Agent definition mirror (Claude, packaging)│
│   ├── agents-codex/     Agent definition mirror (Codex, packaging)│
│   ├── skills/           colony/ (11) + domain/ (18)            │
│   ├── skills-codex/     Codex skill mirror (packaging)          │
│   ├── docs/             Distributed documentation              │
│   └── templates/        Colony state, pheromones, etc.          │
│                                                                  │
│   .aether/data/        ← LOCAL ONLY (gitignored)               │
│   .aether/dreams/      ← LOCAL ONLY (gitignored)               │
│                                                                  │
│   .claude/commands/ant/ ← Slash commands (Claude Code)         │
│   .claude/agents/ant/   ← Agent definitions (Claude Code)      │
│   .opencode/commands/ant/ ← Slash commands (OpenCode)          │
│   .opencode/agents/     ← Agent definitions (OpenCode)         │
│   .codex/agents/        ← Agent definitions (Codex, TOML)      │
│   .codex/CODEX.md       ← Codex commands + rules               │
│                                                                  │
│   ~/.aether/           ← HUB (cross-colony, user-level)        │
│   ├── system/          Companion file source (populated by install)│
│   ├── QUEEN.md         (wisdom + user preferences)              │
│   ├── hive/            (Hive Brain — cross-colony wisdom)       │
│   │   └── wisdom.json  (200-entry cap, LRU eviction)           │
│   └── eternal/         (hive memory — high-value signals)       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

Colony-prime assembles worker context from: QUEEN.md wisdom, eternal memory,
pheromone signals, phase learnings, key decisions, blocker flags, user preferences,
clarified intent, parallel mode, and context capsule — all within a token budget
(see Token Budget below).

**See `RUNTIME UPDATE ARCHITECTURE.md` for complete distribution flow.**

---

## Development Workflow

### Editing System Files

| What you're changing | Where to edit | Why |
|---------------------|---------------|-----|
| workers.md | `.aether/workers.md` | Source of truth |
| Go commands | `cmd/` | Go source code |
| User docs | `.aether/docs/` | Distributed directly |
| Slash commands | `.claude/commands/ant/` | Claude Code commands |
| OpenCode commands | `.opencode/commands/ant/` | OpenCode commands |
| Agent definitions | `.claude/agents/ant/` | Claude Code agents |
| Agent mirror (packaging) | `.aether/agents-claude/` | Must stay in sync with `.claude/agents/ant/` |
| Agent mirror (Codex packaging) | `.aether/agents-codex/` | Must stay in sync with `.codex/agents/` |
| OpenCode agents | `.opencode/agents/` | OpenCode worker definitions |
| Codex agents | `.codex/agents/` | Codex worker definitions (TOML format) |
| Your notes | `.aether/dreams/` | Never distributed |
| Dev docs | `.aether/docs/known-issues.md` | Distributed |

### Autopilot

`/ant:run` — Autopilot that builds, verifies, learns, and advances through phases automatically.

```bash
/ant:run                       # Run all remaining phases
/ant:run --max-phases 2        # Run at most 2 phases then stop
/ant:run --replan-interval 3   # Suggest replan every 3 phases
/ant:run --continue            # Resume after replan pause
/ant:run --dry-run             # Preview the autopilot plan
```

Smart pause conditions: test failures, critical Chaos findings, security gate failures, quality gate failures, runtime verification needed, replan suggestions.

### Intent Capture and Learning

Claude now exposes three pre-build intent commands that route through the shared
Go runtime:

| Command | When to use it | What it does |
|---------|----------------|--------------|
| `/ant:discuss` | After `/ant:init`, before `/ant:plan` | Captures clarifications, stores them as pending decisions, and emits `REDIRECT` pheromones for hard constraints once resolved |
| `/ant:assumptions` | After `/ant:plan`, before `/ant:build` | Surfaces current plan assumptions, writes `assumptions.json`, and emits `FOCUS` / `FEEDBACK` pheromones from the analysis |
| `/ant:profile` | When reviewing or refreshing learned user behavior | Reads or refreshes the behavioral profile and promotes top `[profiled]` directives into `QUEEN.md` |

Typical guided flow:

```bash
/ant:init "Build feature X"
/ant:discuss
/ant:plan
/ant:assumptions
/ant:build 1
/ant:continue
/ant:profile
```

### Publishing Changes

```bash
# 1. Edit files in .aether/ or .claude/commands/ant/
vim .aether/workers.md

# 2. Commit changes
git add .
git commit -m "your message"

# 3. Refresh the hub from this source checkout
aether install --package-dir "$PWD"

# 4. In other repos, pull updates
aether update      # or /ant:update
```

---

## Key Directories

### .aether/ (Source of Truth)

```
.aether/
├── workers.md           # Worker definitions, spawn protocol
├── utils/               # Runtime utilities
│   ├── oracle/oracle.md # Oracle loop instructions (loaded by /ant:oracle)
│   └── queen-to-md.xsl  # XSL transform for queen wisdom export
├── skills/              # colony/ (11) + domain/ (18) skill definitions
├── templates/           # 12 templates (colony-state, pheromones, etc.)
├── docs/                # Distributed documentation
├── exchange/            # XML exchange modules (pheromone-xml, wisdom-xml)
├── agents-claude/       # Claude agent mirror used for packaging
├── agents-codex/        # Codex agent mirror used for packaging
├── skills-codex/        # Codex skill mirror (colony/ + domain/)
├── commands/            # YAML source definitions for slash commands
├── data/                # LOCAL ONLY (never distributed)
│   ├── COLONY_STATE.json  # Colony state with phase tracking + parallel_mode
│   ├── pending-decisions.json
│   ├── assumptions.json
│   ├── behavior-observations.jsonl
│   ├── pheromones.json
│   ├── constraints.json
│   ├── midden/          # Failure tracking
│   └── survey/          # Territory survey results
├── dreams/              # LOCAL ONLY (session notes)
└── oracle/              # LOCAL ONLY (deep research)
```

### .claude/ (Claude Code)

```
.claude/
├── commands/ant/        # 50 slash commands
│   ├── init.md          # Colony initialization
│   ├── discuss.md       # Clarify intent before planning
│   ├── plan.md          # Phase planning
│   ├── assumptions.md   # Surface plan assumptions
│   ├── build.md         # Build orchestrator (loads split playbooks)
│   ├── continue.md      # Continue orchestrator (loads split playbooks)
│   ├── profile.md       # Inspect or refresh the behavioral profile
│   └── ...
├── agents/ant/          # 25 agent definitions
│   ├── aether-builder.md
│   ├── aether-watcher.md
│   ├── aether-scout.md
│   └── ...
└── rules/               # Development rules
    ├── coding-standards.md
    ├── testing.md
    └── ...
```

### .codex/ (Codex CLI)

```
.codex/
├── agents/              # 25 agent definitions (TOML format)
│   ├── aether-builder.toml
│   ├── aether-watcher.toml
│   ├── aether-scout.toml
│   └── ...
└── CODEX.md             # Codex commands + rules
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
- Agent parity model: `.claude/agents/ant/*.md` is canonical, `.aether/agents-claude/*.md` is a byte-identical packaging mirror, `.opencode/agents/*.md` maintains structural parity (same filenames/count), and `.codex/agents/*.toml` is a supported Codex translation plus packaging mirror maintained on a best-effort basis.

---

## Rule Modules

Consolidated guidelines in `.claude/rules/`:
- `@rules/aether-colony.md` — Complete colony system guide (single source)

*(Previous separate rule files have been consolidated into aether-colony.md)*

---

## The 25 Agents

| Tier | Agent | Role |
|------|-------|------|
| Core | Builder | Implements code, TDD-first |
| Core | Watcher | Tests, validates, quality gates |
| Orchestration | Queen | Orchestrates phases, spawns workers |
| Orchestration | Scout | Researches, gathers information |
| Orchestration | Route-Setter | Plans phases, breaks down goals |
| Orchestration | Architect | Architecture design, structural planning |
| Surveyor | surveyor-nest | Maps directory structure |
| Surveyor | surveyor-disciplines | Documents conventions |
| Surveyor | surveyor-pathogens | Identifies tech debt |
| Surveyor | surveyor-provisions | Maps dependencies |
| Specialist | Keeper | Preserves knowledge |
| Specialist | Tracker | Investigates bugs |
| Specialist | Probe | Coverage analysis (NEW) |
| Specialist | Weaver | Refactoring specialist |
| Specialist | Auditor | Quality gate (NEW) |
| Niche | Chaos | Resilience testing |
| Niche | Archaeologist | Excavates git history |
| Niche | Gatekeeper | Security gate (NEW) |
| Niche | Includer | Accessibility audits |
| Niche | Measurer | Performance analysis (NEW) |
| Niche | Sage | Wisdom synthesis |
| Niche | Oracle | Deep research, actionable recommendations |
| Niche | Ambassador | External integrations |
| Niche | Chronicler | Documentation |
| Specialist | Medic | Colony health diagnosis and repair |

---

## Pheromone System

User-colony communication via signals:

| Signal | Command | Priority | Use For |
|--------|---------|----------|---------|
| FOCUS | `/ant:focus "<area>"` | normal | "Pay attention here" |
| REDIRECT | `/ant:redirect "<avoid>"` | high | "Don't do this" (hard constraint) |
| FEEDBACK | `/ant:feedback "<note>"` | low | "Adjust based on this observation" |

**Before builds:** FOCUS + REDIRECT to steer
**After builds:** FEEDBACK to adjust
**Hard constraints:** REDIRECT (will break)
**Gentle nudges:** FEEDBACK (preferences)

**Viewing Signals:**
- `/ant:pheromones` — Full table of all active signals
- `pheromone-display` subcommand — Formatted output with strength % and decay

**Signal Injection:**
- Colony-prime injects active signals into worker prompts via `prompt_section`
- Builder, Watcher, and Scout agents have `pheromone_protocol` sections that instruct them how to act on injected signals
- Signals are grouped by type (FOCUS, REDIRECT, FEEDBACK) in the injected prompt section

**Content Deduplication (v2.0):**
- Each signal gets a SHA-256 `content_hash` on creation
- Writing a duplicate (same type + content hash) reinforces the existing signal instead of creating a new one (strength maxed, `reinforcement_count` incremented)
- `suggest-analyze` deduplicates suggestions against existing active signals and session suggestions

**Prompt Injection Sanitization (v2.0):**
- Pheromone content is sanitized before storage: XML structural tags rejected, angle brackets escaped, shell injection patterns blocked
- Text patterns that attempt LLM instruction override (e.g., "ignore previous instructions") are rejected
- Content capped at 500 characters

**Exchange:**
- `/ant:export-signals` — Export pheromone signals to XML for cross-colony sharing
- `/ant:import-signals` — Import pheromone signals from XML

**Automatic Suggestions:**
- `suggest-analyze` — Analyzes codebase for patterns worth capturing as pheromones
- `suggest-approve` — Tick-to-approve UI for reviewing suggestions
- Runs during build (Step 4.2) to propose contextually relevant signals

**Files:**
- `.aether/data/pheromones.json` — Active signals
- `.aether/data/constraints.json` — Focus areas and constraints (legacy, eventual deprecation)
- `.aether/docs/pheromones.md` — Full guide

---

## Skills System

Skills provide reusable behavior modules and domain knowledge that workers can load
on demand. They come in two categories:

- **Colony skills** (11) — Behavioral patterns that shape how workers operate
  (e.g., TDD discipline, error handling conventions, commit style)
- **Domain skills** (18) — Technical knowledge for specific frameworks, languages,
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

Skill content is injected separately from colony-prime context:

- Own 8K character budget (independent of the colony-prime token budget)
- Injected into builder and watcher prompts
- `skill-inject` assembles matched skills into a prompt section

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

- `/ant:skill-create` — Oracle-powered skill generation from a description
- Manual creation: add a `SKILL.md` file to `~/.aether/skills/domain/`
- Each skill uses frontmatter (name, category, detect patterns, roles)

### Update Safety

- Manifest-based tracking ensures shipped skills update cleanly
- User-created or user-modified skills are never overwritten during `aether update`

---

## Token Budget

Colony-prime assembles worker context within a character budget to avoid prompt bloat:

| Mode | Budget | When |
|------|--------|------|
| Normal | 8,000 chars | Default |
| Compact | 4,000 chars | `--compact` flag or auto-detected |

**Trim order** (first trimmed = lowest retention priority):
1. Rolling summary (trimmed first -- lowest retention priority)
2. Phase learnings
3. Key decisions
4. Hive wisdom
5. Context capsule
6. User preferences
7. QUEEN.md wisdom
8. Pheromone signals (trimmed last -- highest retention priority)
9. Blockers (NEVER trimmed)

Trimmed sections are logged for debugging. See `pheromone-write` subcommand for implementation.

---

## Hive Brain (Cross-Colony Wisdom)

The Hive Brain is the intelligent layer for cross-colony knowledge sharing. It stores
generalized wisdom derived from colony instincts, scoped by domain, and shared across
all colonies on the same machine.

### Storage

```
~/.aether/hive/
└── wisdom.json          # Cross-colony wisdom (200-entry cap, LRU eviction)
```

- Entries are capped at 200; least-recently-used entries are evicted when full
- Hub-level file locking prevents concurrent write corruption

### Subcommands

| Subcommand | Purpose |
|------------|---------|
| `hive-init` | Initialize `~/.aether/hive/` directory and empty `wisdom.json` |
| `hive-store` | Store a wisdom entry with deduplication, merge, and 200-cap enforcement |
| `hive-read` | Read wisdom with domain filtering, confidence threshold, and access tracking |
| `hive-abstract` | Generalize a repo-specific instinct into cross-colony wisdom text |
| `hive-promote` | Orchestrate the abstract + store pipeline (end-to-end promotion) |

### Domain-Scoped Retrieval

Colony-prime retrieves hive wisdom scoped to the current project's domain:

1. Reads domain tags from the colony registry entry for the current repo
2. Calls `hive-read` with those domain tags and a confidence threshold
3. Injects domain-relevant wisdom into worker prompts as `HIVE WISDOM (Cross-Colony Patterns)`
4. **Fallback chain:** hive -> eternal -> empty (graceful degradation if no wisdom exists)

### Seal Promotion Hook

During `/ant:seal` (Step 3.7), high-confidence instincts are promoted to the hive:

1. Extracts instincts with confidence >= 0.8 from `COLONY_STATE.json`
2. Promotes each via `hive-promote` with `--text` and `--source-repo`
3. **NON-BLOCKING** — promotion failures are logged but never stop the seal

### Multi-Repo Confidence Boosting

When the same wisdom is confirmed across multiple repositories, confidence increases:

| Repos Confirming | Confidence |
|-----------------|------------|
| 2 repos | 0.70 |
| 3 repos | 0.85 |
| 4+ repos | 0.95 |

- Confidence is **never downgraded** — uses max of current value and tier value
- Same-repo re-promotion is deduplicated (no duplicate entries)

### Legacy: Eternal Memory

The older eternal memory system (`~/.aether/eternal/`) remains as a fallback:

- `~/.aether/eternal/memory.json` — High-value signals promoted from expired pheromones
- `eternal-store` / `eternal-init` subcommands still functional
- Colony-prime falls back to eternal memory when hive has no matching entries

---

## User Preferences

User preferences are stored in the hub `~/.aether/QUEEN.md` under the `## User Preferences` section.

| Command | Purpose |
|---------|---------|
| `/ant:preferences "text"` | Add a user preference to hub QUEEN.md |
| `/ant:preferences --list` | List all user preferences |

- Preferences capture communication style, expertise level, and decision patterns
- Colony-prime injects user preferences into worker context
- `/ant:profile` promotes learned directives into the same section with a `[profiled]` prefix
- Max 500 characters per preference entry

---

## Registry

Colony registry tracks all repos using Aether (`~/.aether/registry/`):

- **Domain tags** — Categorize colonies by domain (e.g., `["web", "api"]`)
- **Colony goal tracking** — `last_colony_goal` stored per repo entry
- **Active status** — `active_colony` flag per repo
- Legacy entries are auto-normalized with default values on read

---

## Quality Gates

New agents integrated into continue.md:

### Gatekeeper (Security)
- Runs after verification passes
- Scans for exposed secrets, debug artifacts via `check-antipattern` (~6 patterns -- not a full security scanner)
- Creates blockers if security issues found

### Auditor (Quality)
- Runs after Gatekeeper passes
- Analyzes code quality metrics
- Reports quality gate status

### Probe (Coverage)
- Analyzes test coverage gaps
- Reports coverage percentage
- Suggests additional tests

### Measurer (Performance)
- Performance analysis
- Identifies slow operations
- Reports performance metrics

---

## Midden System (Failure Tracking)

The midden tracks failures for colony learning:

- `.aether/data/midden/midden.json` — Failure records
- `midden-write` — Log a failure
- `midden-recent-failures` — Query recent failures
- `midden-review` — Review unacknowledged midden entries grouped by category
- `midden-acknowledge` — Mark midden entries as addressed by id or category

Failures are logged during:
- Build failures (build.md)
- Approach changes (tracked for wisdom)

**Data Maintenance:**
- `/ant:data-clean` — Remove test artifacts from colony data files (pheromones, constraints, midden)

---

## Memory Health System

Colony memory is tracked and displayed:

- `/ant:status` — Shows memory health table
- `/ant:memory-details` — Drill-down view
- `/ant:resume` — Shows memory health section

Metrics tracked:
- Events count
- Learnings count
- Instincts count
- Pheromones count
- Memory age

---

## Changelog System

Automated changelog collection:

- `changelog-append` — Append entry to CHANGELOG.md
- `changelog-collect-plan-data` — Collect plan data for changelog

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
go test ./...

# Verify binary works
aether version

# Run all Go tests
go test ./... -race
```

---

## Session Freshness Detection

All stateful commands use timestamp verification to detect stale sessions:

**Pattern:**
1. Capture `SESSION_START=$(date +%s)` before spawning agents
2. Check file freshness with `session-verify-fresh --command <name>`
3. Auto-clear stale files or prompt user based on command type
4. Verify files are fresh after spawning

**Protected Commands** (never auto-clear):
- `init` — COLONY_STATE.json is precious
- `seal` — Archives are precious
- `entomb` — Chambers are precious

---

## Session Recovery

On the first message of a new conversation, check if `.aether/data/session.json` exists. If it does:

1. Read the file briefly to check for `colony_goal`
2. If a goal exists, display:
   ```
   Previous colony session detected: "{goal}"
   Run /ant:resume to restore context, or continue with a new topic.
   ```
3. Do NOT auto-restore — wait for the user to explicitly run `/ant:resume`

---

## Wisdom Pipeline

The Wisdom Pipeline is the core learning loop of Aether. Colony work produces
observations that flow through the system and become reusable wisdom.

### Pipeline Stages

| Stage | Subcommand | Output |
|-------|-----------|--------|
| 1. Observe | `memory-capture "learning"` | Records observation to learning-observations.json |
| 1a. Trust score | `trust-score-compute` | Assigns weighted trust score to observation (40/35/25, 7 tiers) |
| 1b. Event bus | `event-bus-publish` | Publishes scored event to JSONL event bus with TTL |
| 2. Auto-promote | (internal: `learning-promote-auto`) | Triggers after threshold (2 observations for patterns) |
| 3. Instinct | `instinct-create` | Stores in COLONY_STATE.json with provenance |
| 4. QUEEN.md | `queen-promote` | Writes to QUEEN.md Patterns/Philosophies section |
| 5. Inject | `colony-prime` prompt_section | QUEEN.md wisdom + instincts injected into worker context |
| 6. Hive store | `hive-promote` | Abstracts instinct, stores in hive wisdom.json (confidence >= 0.8) |
| 7. Hive read | `hive-read` | Retrieves cross-colony wisdom scoped by domain |

**See `.aether/docs/structural-learning-stack.md` for the full Structural Learning Stack documentation.**

### Key Thresholds

- **Auto-promotion:** Pattern observations need 2 captures to trigger; confidence starts at 0.75
- **Hive promotion:** Instincts with confidence >= 0.8 are promoted to Hive Brain at `/ant:seal`
- **Cross-colony boost:** Multi-repo confirmation raises confidence (2 repos = 0.70, 4+ = 0.95)

See [Hive Brain](#hive-brain-cross-colony-wisdom) for cross-colony wisdom details.

---

## Structural Learning Stack

The memory consolidation pipeline that upgrades the Wisdom Pipeline with
trust scoring, graph relationships, and automated curation.

**See `.aether/docs/structural-learning-stack.md` for complete documentation.**

Key additions:
- Trust scoring engine (40/35/25 weighted, 60-day half-life, 7 tiers)
- JSONL event bus with pub/sub and TTL cleanup
- Standalone instinct storage with full provenance
- jq-based graph layer for instinct relationships
- 8 curation ants with orchestrated execution
- Lifecycle integration: phase-end at /ant:continue, full at /ant:seal

### Curation Ants

| Ant | Role |
|-----|------|
| `orchestrator` | Coordinates curation pipeline execution |
| `archivist` | Archives and retrieves historical observations |
| `critic` | Evaluates instinct quality and confidence |
| `herald` | Broadcasts high-confidence instincts to hive |
| `janitor` | Cleans stale events and expired TTL entries |
| `librarian` | Indexes and catalogs instinct relationships |
| `nurse` | Heals low-confidence instincts with supporting evidence |
| `scribe` | Records curation decisions and audit trail |
| `sentinel` | Guards against instinct corruption and conflicts |

---

## Parallel Execution Modes

Aether supports two parallel execution strategies, selected at colony init:

| Mode | Value | Description |
|------|-------|-------------|
| In-repo (default) | `"in-repo"` | Workers share the working tree directly |
| Worktree | `"worktree"` | Each worker gets an isolated git worktree, then file and pheromone changes sync back into the main root |

### How It Works

1. **`/ant:init`** prompts for strategy selection (Step 6.5) during colony setup
2. The choice is saved as `parallel_mode` in `COLONY_STATE.json` (omitempty, defaults to `"in-repo"`)
3. **Colony-prime** injects the current parallel mode into worker context
4. **Build-wave playbook** conditionally allocates worktrees when mode is `"worktree"`
5. **Continue-advance playbook** performs mode-aware cleanup/merge checks after build-wave sync-back
6. **Status and resume** commands display the active parallel mode

### CLI Subcommands

| Subcommand | Purpose |
|------------|---------|
| `parallel-mode get` | Read current parallel mode from colony state |
| `parallel-mode set <mode>` | Change parallel mode ("in-repo" or "worktree") |

### Choosing a Mode

- **In-repo** is simpler and works well for most projects. Workers write directly to the repo.
- **Worktree** provides isolation -- each worker gets its own git worktree, and successful file plus pheromone changes are synced back into the main root. Requires a clean working tree and is best for larger tasks with multiple workers touching different files.

---

## The Core Insight

The system's pieces are now **connected**:
- Pheromones update context (colony-prime injects signals into worker prompts)
- Decisions become pheromones (auto-emit during builds)
- Learnings become instincts (observation to promotion pipeline -- see Wisdom Pipeline above)
- Midden affects behavior (threshold auto-REDIRECT)
- Hive Brain crosses colony boundaries (domain-scoped wisdom -> colony-prime)
- Instincts promote to hive at seal (confidence >= 0.8 -> hive-promote)
- Multi-repo confirmation boosts confidence (2 repos = 0.7, 4+ = 0.95)
- User preferences shape worker behavior (QUEEN.md -> colony-prime)
- Autopilot chains build-verify-advance with smart pausing (/ant:run)

**The ongoing challenge is maintenance** -- keeping documentation accurate,
data files clean, and test coverage comprehensive as features evolve.

---

## For OpenCode / Codex

For OpenCode-specific rules and agents, see `.opencode/OPENCODE.md`

For Codex-specific rules and agents, see `.codex/CODEX.md`

---

*Updated for Aether v1.0.17 — 2026-04-21*
