# CLAUDE.md — Aether Development Guide

> **Current Version:** v2.0.0
> **Architecture:** v4.0 (runtime/ eliminated, direct packaging)
> **Last Updated:** 2026-03-20 (v2.0 — autopilot, hive intelligence, user preferences)

---

## Quick Reference

| What | Count/Status |
|------|--------------|
| Version | v2.0.0 |
| Slash commands | 43 (Claude) + 43 (OpenCode) |
| Agent definitions | 22 |
| aether-utils.sh | 10,000+ lines, 110 subcommands |
| Tests | 530+ passing |
| Architecture doc | `RUNTIME UPDATE ARCHITECTURE.md` |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     AETHER REPO (this repo)                      │
│                                                                  │
│   .aether/             ← SOURCE OF TRUTH (packaged directly)    │
│   ├── workers.md       (edit here)                              │
│   ├── aether-utils.sh  (10,000+ lines, 110 subcommands)          │
│   ├── utils/           (18 utility scripts)                     │
│   ├── docs/            (distributed documentation)              │
│   └── templates/       (12 templates)                           │
│                                                                  │
│   .aether/data/        ← LOCAL ONLY (excluded by .npmignore)    │
│   .aether/dreams/      ← LOCAL ONLY (excluded by .npmignore)    │
│                                                                  │
│   .claude/commands/ant/ ← 42 slash commands (Claude Code)       │
│   .claude/agents/ant/   ← 22 agent definitions                  │
│   .opencode/commands/ant/ ← 40 slash commands (OpenCode)        │
│   .opencode/agents/     ← Agent definitions (OpenCode)          │
│                                                                  │
│   ~/.aether/           ← HUB (cross-colony, user-level)         │
│   ├── QUEEN.md         (wisdom + user preferences)               │
│   └── eternal/         (hive memory — high-value signals)        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

Colony-prime assembles worker context from: QUEEN.md wisdom, eternal memory,
pheromone signals, phase learnings, key decisions, blocker flags, user preferences,
and context capsule — all within a token budget (see Token Budget below).

**See `RUNTIME UPDATE ARCHITECTURE.md` for complete distribution flow.**

---

## Development Workflow

### Editing System Files

| What you're changing | Where to edit | Why |
|---------------------|---------------|-----|
| workers.md | `.aether/workers.md` | Source of truth |
| aether-utils.sh | `.aether/aether-utils.sh` | Source of truth |
| utils/*.sh | `.aether/utils/` | Source of truth |
| User docs | `.aether/docs/` | Distributed directly |
| Slash commands | `.claude/commands/ant/` | Claude Code commands |
| OpenCode commands | `.opencode/commands/ant/` | OpenCode commands |
| Agent definitions | `.claude/agents/ant/` | Claude Code agents |
| Agent mirror (packaging) | `.aether/agents-claude/` | Must stay in sync with `.claude/agents/ant/` |
| OpenCode agents | `.opencode/agents/` | OpenCode worker definitions |
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

### Publishing Changes

```bash
# 1. Edit files in .aether/ or .claude/commands/ant/
vim .aether/workers.md

# 2. Commit changes
git add .
git commit -m "your message"

# 3. Validate and push to hub
npm install -g .   # Runs validate-package.sh, then setupHub()

# 4. In other repos, pull updates
aether update      # or /ant:update
```

---

## Key Directories

### .aether/ (Source of Truth)

```
.aether/
├── workers.md           # Worker definitions, spawn protocol
├── aether-utils.sh      # 110 subcommands for state management
├── utils/               # 18 utility scripts
│   ├── file-lock.sh     # Locking primitives
│   ├── atomic-write.sh  # Safe file writes
│   ├── swarm-display.sh # Visualization
│   └── xml-*.sh         # XML processing
├── templates/           # 12 templates (colony-state, pheromones, etc.)
├── docs/                # Distributed documentation
├── exchange/            # XML exchange modules (pheromone-xml, wisdom-xml)
├── agents-claude/       # Claude agent mirror used for packaging
├── data/                # LOCAL ONLY (never distributed)
│   ├── COLONY_STATE.json
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
├── commands/ant/        # 42 slash commands
│   ├── init.md          # Colony initialization
│   ├── plan.md          # Phase planning
│   ├── build.md         # Build orchestrator (loads split playbooks)
│   ├── continue.md      # Continue orchestrator (loads split playbooks)
│   └── ...
├── agents/ant/          # 22 agent definitions
│   ├── aether-builder.md
│   ├── aether-watcher.md
│   ├── aether-scout.md
│   └── ...
└── rules/               # Development rules
    ├── coding-standards.md
    ├── testing.md
    └── ...
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
- Agent parity model: `.claude/agents/ant/*.md` is canonical, `.aether/agents-claude/*.md` is a byte-identical packaging mirror, `.opencode/agents/*.md` maintains structural parity (same filenames/count).

---

## Rule Modules

Consolidated guidelines in `.claude/rules/`:
- `@rules/aether-colony.md` — Complete colony system guide (single source)

*(Previous separate rule files have been consolidated into aether-colony.md)*

---

## The 22 Agents

| Tier | Agent | Role |
|------|-------|------|
| Core | Builder | Implements code, TDD-first |
| Core | Watcher | Tests, validates, quality gates |
| Orchestration | Queen | Orchestrates phases, spawns workers |
| Orchestration | Scout | Researches, gathers information |
| Orchestration | Route-Setter | Plans phases, breaks down goals |
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
| Niche | Ambassador | External integrations |
| Niche | Chronicler | Documentation |

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

## Token Budget

Colony-prime assembles worker context within a character budget to avoid prompt bloat:

| Mode | Budget | When |
|------|--------|------|
| Normal | 8,000 chars | Default |
| Compact | 4,000 chars | `--compact` flag or auto-detected |

**Priority order** (trimmed from bottom when over budget):
1. Rolling summary (highest priority — never trimmed first)
2. Phase learnings
3. Key decisions
4. Hive wisdom
5. Context capsule
6. User preferences
7. QUEEN.md wisdom
8. Pheromone signals (lowest priority)

Trimmed sections are logged for debugging.

---

## Hive Intelligence

Cross-colony knowledge stored in the hub (`~/.aether/eternal/`):

- `~/.aether/eternal/memory.json` — High-value signals promoted from expired pheromones
- Colony-prime reads eternal memory and injects it as `HIVE WISDOM (Cross-Colony Patterns)` into worker prompts
- `eternal-store` subcommand — Store a high-value signal in eternal memory
- `eternal-init` subcommand — Initialize the eternal memory structure

Hive wisdom is shared across all colonies on the same machine, providing cross-project learning.

---

## User Preferences

User preferences are stored in the hub `~/.aether/QUEEN.md` under the `## User Preferences` section.

| Command | Purpose |
|---------|---------|
| `/ant:preferences "text"` | Add a user preference to hub QUEEN.md |
| `/ant:preferences --list` | List all user preferences |

- Preferences capture communication style, expertise level, and decision patterns
- Colony-prime injects user preferences into worker context
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
- Scans for exposed secrets, debug artifacts
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
# Verify command and agent sync policy
npm run lint:sync

# Run all linters
npm run lint

# Run all tests
npm test

# Verify package before publishing
bash bin/validate-package.sh

# See what npm would package
npm pack --dry-run
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

## The Core Insight

The system's pieces are now **connected**:
- Pheromones update context (colony-prime injects signals into worker prompts)
- Decisions become pheromones (auto-emit during builds)
- Learnings become instincts (observation to promotion pipeline)
- Midden affects behavior (threshold auto-REDIRECT)
- Hive wisdom crosses colony boundaries (eternal memory -> colony-prime)
- User preferences shape worker behavior (QUEEN.md -> colony-prime)
- Autopilot chains build-verify-advance with smart pausing (/ant:run)

**The ongoing challenge is maintenance** -- keeping documentation accurate,
data files clean, and test coverage comprehensive as features evolve.

---

## For OpenCode

For OpenCode-specific rules and agents, see `.opencode/OPENCODE.md`

---

*Updated for Aether v2.0.0 — 2026-03-20 (autopilot, hive intelligence, user preferences, pheromone hardening)*
