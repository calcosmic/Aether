# Architecture

**Analysis Date:** 2026-02-17

## Pattern Overview

**Overall:** Multi-agent orchestration system using ant colony metaphor with centralized state management and distributed worker execution.

**Key Characteristics:**
- Queen-Worker hierarchy: Central coordinator (CLI) spawns specialized worker agents
- Pheromone-based communication: Workers leave signals (FOCUS, REDIRECT, FEEDBACK) that influence colony behavior
- State persistence: Colony state survives across sessions via JSON files in `.aether/data/`
- Model routing: Workers assigned to "castes" with different AI model affinities (architectural aspiration, currently limited by platform)
- Session freshness detection: Timestamp-based verification prevents stale session files from breaking workflows

## Layers

**CLI Layer:**
- Purpose: Entry point for all user commands
- Location: `/Users/callumcowie/repos/Aether/bin/cli.js`
- Contains: Command routing, error handling, hub sync
- Depends on: Node.js, commander, js-yaml, picocolors
- Used by: End users invoking `aether <command>`

**Utility Layer:**
- Purpose: Core shell functions for colony operations
- Location: `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`
- Contains: ~3,700 lines of bash functions for state management, spawn tracking, verification
- Depends on: bash, jq (optional), git
- Used by: CLI commands, slash commands, spawned workers

**Command Layer:**
- Purpose: User-facing commands (slash commands)
- Location: `/Users/callumcowie/repos/Aether/.claude/commands/ant/` and `/Users/callumcowie/repos/Aether/.opencode/commands/ant/`
- Contains: Markdown command definitions invoked by Claude Code/OpenCode
- Depends on: aether-utils.sh, CLI
- Used by: Claude Code and OpenCode when user invokes `/ant:<command>`

**State Layer:**
- Purpose: Persistent colony data
- Location: `/Users/callumcowie/repos/Aether/.aether/data/`
- Contains: COLONY_STATE.json, pheromones.json, checkpoints/, locks/
- Depends on: aether-utils.sh for reads/writes
- Used by: All layers for persistence

**Exchange Layer (XML):**
- Purpose: Structured data exchange between colony and external systems
- Location: `/Users/callumcowie/repos/Aether/.aether/exchange/`
- Contains: pheromone-xml.sh, wisdom-xml.sh, registry-xml.sh
- Depends on: XML utilities in utils/
- Used by: Integration points with external systems

## Data Flow

**Command Execution Flow:**

1. User invokes `/ant:<command>` in Claude Code/OpenCode
2. Slash command definition (`.claude/commands/ant/<command>.md`) loads
3. Command invokes `bash .aether/aether-utils.sh <subcommand>` or `aether <command>`
4. Utility layer performs operation, updates state files
5. Response returned to user via Claude Code

**Worker Spawn Flow:**

1. Prime caste worker coordinates task
2. Spawns child worker via Claude Code Task tool
3. Child worker inherits parent session model
4. Worker logs activity to spawn-tree
5. On completion, worker reports results to parent

**State Persistence Flow:**

1. Command calls `read_colony_state` function in aether-utils.sh
2. Function reads `.aether/data/COLONY_STATE.json`
3. Command modifies state in memory
4. Command calls state update function
5. Atomic write to temp file, then rename (prevents corruption)
6. State persisted across sessions

## Key Abstractions

**Colony State:**
- Purpose: Represents current colony status
- Examples: `/Users/callumcowie/repos/Aether/.aether/data/COLONY_STATE.json`
- Pattern: JSON file with phases, goals, milestones, caste assignments

**Pheromones:**
- Purpose: Signals between user and colony
- Examples: `/Users/callumcowie/repos/Aether/.aether/data/pheromones.json`
- Pattern: Priority-based signal queue (FOCUS=normal, REDIRECT=high, FEEDBACK=low)

**Workers (Caste System):**
- Purpose: Role definitions for different task types
- Examples: `/Users/callumcowie/repos/Aether/.aether/workers.md`
- Pattern: 21 castes (builder, watcher, scout, chaos, oracle, architect, prime, colonizer, route_setter, archaeologist, ambassador, auditor, chronicler, guardian, includer, keeper, measurer, probe, sage, tracker, weaver)

**Checkpoints:**
- Purpose: Session recovery points
- Location: `/Users/callumcowie/repos/Aether/.aether/checkpoints/`
- Pattern: git stash with allowlist of files to preserve

**Chambers:**
- Purpose: Archived completed colonies
- Location: `/Users/callumcowie/repos/Aether/.aether/chambers/`
- Pattern: Named directories with frozen COLONY_STATE.json

## Entry Points

**CLI Entry:**
- Location: `/Users/callumcowie/repos/Aether/bin/cli.js`
- Triggers: `aether <command>` from terminal
- Responsibilities: Command parsing, hub sync, state management, error handling

**Slash Commands (Claude Code):**
- Location: `/Users/callumcowie/repos/Aether/.claude/commands/ant/*.md`
- Triggers: `/ant:<command>` in Claude Code
- Responsibilities: User-facing operations (colonize, build, test, etc.)

**Slash Commands (OpenCode):**
- Location: `/Users/callumcowie/repos/Aether/.opencode/commands/ant/*.md`
- Triggers: `/ant:<command>` in OpenCode
- Responsibilities: Same as Claude Code commands, OpenCode-specific syntax

**Utility Entry:**
- Location: `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`
- Triggers: `bash .aether/aether-utils.sh <command> [args...]`
- Responsibilities: Core operations (state management, spawn tracking, verification)

## Error Handling

**Strategy:** Structured error classes with JSON output and recovery hints.

**Patterns:**
- Error codes: `E_FILE_NOT_FOUND`, `E_VALIDATION_FAILED`, `E_GIT_ERROR`, etc. (defined in `/Users/callumcowie/repos/Aether/bin/lib/errors.js`)
- JSON errors: `json_err` function returns `{"ok":false,"error":{...}}`
- Recovery hints: Each error includes `"recovery"` field with suggested action
- File locks: `file-lock.sh` prevents concurrent state modifications

## Cross-Cutting Concerns

**Logging:** Activity log via `activity-log` function in aether-utils.sh, writes to `.aether/data/activity.log`

**Validation:** State validation via `validate-state` command, checks COLONY_STATE.json schema

**Authentication:** Not applicable - uses AI provider API keys via environment (ANTHROPIC_API_KEY)

**Session Freshness:** Timestamp-based verification in `session-verify-fresh` function, prevents stale session files

---

*Architecture analysis: 2026-02-17*
