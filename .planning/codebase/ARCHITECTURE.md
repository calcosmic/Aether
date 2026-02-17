# Architecture

**Analysis Date:** 2026-02-17

## Pattern Overview

**Overall:** Multi-agent colony system with distributed worker spawning and pheromone-based coordination

**Key Characteristics:**
- Self-organizing emergence through worker spawning workers (no central orchestration)
- Depth-based behavior control (max depth 3, with spawn limits at each level)
- Dual-layer system: JavaScript CLI layer + Bash utility layer
- Hub-based distribution model (npm package + hub directory)
- Pheromone-based constraint system (FOCUS, REDIRECT, FEEDBACK signals)

## Layers

**CLI Layer (JavaScript):**
- Location: `bin/cli.js`
- Purpose: Command dispatch, package installation, hub synchronization, state management
- Contains: Commander-based CLI, error handling, update transactions, model profiles
- Depends on: Node.js standard library, commander, js-yaml, picocolors
- Used by: Users invoking `aether` commands directly

**Utility Layer (Bash):**
- Location: `.aether/aether-utils.sh` (source), `runtime/aether-utils.sh` (distributed)
- Purpose: Colony operations, context management, activity logging, spawn management
- Contains: 50+ subcommands for colony lifecycle, JSON output helpers, lock management
- Depends on: bash, jq (optional), git (optional)
- Used by: Slash commands, worker scripts, colony operations

**Command Layer (Markdown prompts):**
- Location: `.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`
- Purpose: User-facing slash commands for colony interaction
- Contains: 31 slash commands (init, build, plan, continue, watch, etc.)
- Used by: Claude Code and OpenCode users directly

**Agent Layer:**
- Location: `.opencode/agents/*.md`
- Purpose: OpenCode-specific agent definitions
- Used by: OpenCode platform

## Data Flow

**Initialization Flow:**
1. User runs `/ant:init <goal>` or `aether init <goal>`
2. CLI creates `COLONY_STATE.json` in `.aether/data/`
3. Context.md updated with initial state
4. Colony enters Phase 1 (initialization)

**Build Flow:**
1. User runs `/ant:build`
2. CLI spawns Prime Worker via Task tool
3. Prime Worker spawns specialists (Builder, Watcher, Scout)
4. Workers log activity to `activity.log`
5. Spawn tree tracked in `spawn-tree.txt`

**Update Distribution Flow:**
1. Developer edits `.aether/` files (source of truth)
2. Runs `npm install -g .` (preinstall: sync-to-runtime.sh)
3. Files copied to `runtime/` directory
4. CLI pushes to hub (`~/.aether/`)
5. Users in other repos run `aether update` to receive

**State Persistence:**
- Colony state: `.aether/data/COLONY_STATE.json`
- Pheromones: `.aether/data/pheromones.json`
- Constraints: `.aether/data/constraints.json`
- Flags: `.aether/data/flags.json`
- Activity: `.aether/data/activity.log`
- Spawn tree: `.aether/data/spawn-tree.txt`

## Key Abstractions

**Colony State:**
- Purpose: Represents current colony status, goal, phase, milestone
- Examples: `.aether/data/COLONY_STATE.json`
- Pattern: JSON file with current_phase, goal, milestone, created_at fields

**Pheromones:**
- Purpose: Constraint signals that guide worker behavior
- Examples: FOCUS (normal priority), REDIRECT (high priority), FEEDBACK (low priority)
- Pattern: JSON file with signal type, content, priority, created_at

**Workers:**
- Purpose: Autonomous agents that perform tasks
- Examples: Builder, Watcher, Scout, Prime, Architect, Oracle
- Pattern: Defined in `workers.md`, spawned via Task tool with depth tracking

**Spawn Tree:**
- Purpose: Visual representation of worker hierarchy
- Examples: `.aether/data/spawn-tree.txt`
- Pattern: Indented tree showing parent-child relationships

**Chambers:**
- Purpose: Archived colony states for later review
- Location: `.aether/chambers/`
- Pattern: Timestamped directories with COLONY_STATE snapshot

## Entry Points

**CLI Entry:**
- Location: `bin/cli.js`
- Triggers: `aether <command>` or npm bin wrapper
- Responsibilities: Command parsing, error handling, hub sync, update transactions

**Slash Commands:**
- Location: `.claude/commands/ant/*.md`
- Triggers: `/ant:<command>` in Claude Code
- Responsibilities: User intent interpretation, worker spawning, context updates

**Utility Entry:**
- Location: `.aether/aether-utils.sh`
- Triggers: `bash .aether/aether-utils.sh <subcommand> [args]`
- Responsibilities: JSON I/O operations, lock management, file operations

**Install Hook:**
- Location: `bin/sync-to-runtime.sh`
- Triggers: `npm install -g .`
- Responsibilities: Copy allowlisted files from `.aether/` to `runtime/`

## Error Handling

**Strategy:** Structured error classes with recovery suggestions

**Patterns:**
- JavaScript: Custom error classes in `bin/lib/errors.js` (AetherError, HubError, RepoError, etc.)
- Bash: JSON error output via `json_err()` function, error constants (E_*)
- Global handlers: Process-level uncaughtException and unhandledRejection handlers

## Cross-Cutting Concerns

**Logging:** Activity log in `.aether/data/activity.log` with timestamps and emoji indicators

**Validation:** COLONY_STATE.json validated before operations, constraint allowlists for checkpoints

**Authentication:** N/A (not an authentication system)

**Lock Management:** File-based locking in `.aether/locks/` with timeout support

---

*Architecture analysis: 2026-02-17*
