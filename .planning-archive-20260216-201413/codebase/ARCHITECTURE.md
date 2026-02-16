# Architecture

**Analysis Date:** 2026-02-13

## Pattern Overview

**Overall:** Event-driven multi-agent orchestration system with ant colony metaphor

**Key Characteristics:**
- **Queen-Worker Hierarchy**: Queen orchestrates, workers execute autonomously
- **Nested Spawning**: Workers spawn sub-workers up to depth 3
- **Pheromone-based Constraints**: Declarative focus/avoid patterns guide worker behavior
- **State Machine**: Colony transitions through IDLE -> READY -> PLANNING -> EXECUTING states
- **Tool-agnostic**: Supports both Claude Code and OpenCode via parallel command sets

## Layers

**CLI Layer:**
- Purpose: User interface and command routing
- Location: `bin/cli.js`
- Contains: npm install/update/version commands, hub setup
- Depends on: Node.js fs/path/crypto modules
- Used by: npm postinstall hook, `aether` CLI command

**Command Layer:**
- Purpose: Slash command definitions and worker prompts
- Location: `.claude/commands/ant/`, `.opencode/commands/ant/`
- Contains: 28 markdown command files (init, build, plan, continue, etc.)
- Depends on: Runtime layer (aether-utils.sh)
- Used by: Claude Code and OpenCode when slash commands invoked

**Runtime Layer:**
- Purpose: Core utility functions for colony operations
- Location: `runtime/`, `.aether/` (repo-local copies)
- Contains: `aether-utils.sh` (59 subcommands), utility scripts, worker specs
- Depends on: jq (JSON processor), bash
- Used by: Commands invoke utilities via `bash .aether/aether-utils.sh <subcommand>`

**State Layer:**
- Purpose: Persistent colony state storage
- Location: `.aether/data/`
- Contains: COLONY_STATE.json, constraints.json, flags.json, activity.log, spawn-tree.txt
- Depends on: File system
- Used by: All commands read/write state

## Data Flow

**Colony Initialization:**

1. User runs `/ant:init "goal"`
2. Command reads prior completion-report.md (if exists) for inherited knowledge
3. Writes COLONY_STATE.json with v3.0 structure
4. Initializes constraints.json
5. Registers repo in ~/.aether/registry.json

**Phase Execution:**

1. User runs `/ant:build N`
2. Command validates state, creates git checkpoint
3. Loads constraints from constraints.json
4. Optionally spawns Archaeologist for pre-build scan
5. Spawns Wave 1 Builders in parallel using Task tool
6. Collects results, spawns Watcher for verification
7. Spawns Chaos Ant for resilience testing
8. Synthesizes results, updates state, displays summary

**State Management:**
- All state is JSON-based, stored in `.aether/data/`
- Atomic writes via `utils/atomic-write.sh`
- File locking via `utils/file-lock.sh`
- State validated via `validate-state` subcommand

## Key Abstractions

**Worker Castes:**
- Purpose: Specialized agent roles for different task types
- Examples: Builder (code), Watcher (verification), Scout (research), Colonizer (exploration)
- Pattern: Each caste has defined discipline, spawn rules, output format
- Location: `.aether/workers.md` (source of truth), `runtime/workers.md` (staging for npm)

**Pheromone Signals (Constraints):**
- Purpose: Guide worker behavior without direct commands
- Examples: FOCUS areas, REDIRECT patterns to avoid
- Pattern: Declarative constraints in constraints.json, read by workers at spawn
- Location: `.aether/data/constraints.json`

**Spawn Tree:**
- Purpose: Track worker hierarchy and depth limits
- Examples: Queen -> Builder-1 (depth 1) -> Scout-7 (depth 2)
- Pattern: Logged to spawn-tree.txt, visualized in /ant:watch
- Location: `.aether/data/spawn-tree.txt`

**Flags:**
- Purpose: Persist blockers, issues, and notes across context resets
- Examples: Blockers (critical), Issues (high), Notes (low)
- Pattern: Auto-resolve on build_pass trigger, resolved via /ant:flags
- Location: `.aether/data/flags.json`

## Entry Points

**CLI Entry:**
- Location: `bin/cli.js`
- Triggers: `aether install|update|version|uninstall|help`
- Responsibilities: Hub setup, command sync, version management

**Command Entry:**
- Location: `.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`
- Triggers: `/ant:init`, `/ant:plan`, `/ant:build`, `/ant:continue`, etc.
- Responsibilities: Parse user intent, orchestrate workers, manage state

**Agent Entry (OpenCode):**
- Location: `.opencode/agents/*.md`
- Triggers: OpenCode agent selection
- Responsibilities: Pre-configured agent personas for queen, builder, scout, watcher

## Error Handling

**Strategy:** Layered error handling with graceful degradation

**Patterns:**
- Shell utilities output JSON with `ok: true/false` and exit codes
- Commands check exit codes and parse JSON errors
- Non-blocking operations (version check, registry) fail silently
- Git operations use stash checkpoints for rollback capability
- Graveyard markers record failed file attempts for future workers

**Error Recovery:**
- `/ant:swarm` deploys parallel scouts to investigate stubborn bugs
- `autofix-checkpoint` and `autofix-rollback` provide git-based undo
- Flags persist blockers across context resets

## Cross-Cutting Concerns

**Logging:** Activity log via `activity-log` subcommand, stored in `.aether/data/activity.log`

**Validation:** State validation via `validate-state` subcommand with JSON schema checks

**Authentication:** Not applicable (local tool)

**Synchronization:** Hash-based idempotent sync between repo and ~/.aether/ hub

---

*Architecture analysis: 2026-02-13*
