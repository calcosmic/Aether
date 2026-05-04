---
schema_version: "1.0"
id: protected-local-state-contract
kind: contract
category: contracts
title: Protected Local State Contract
description: "Paths that must never be overwritten during update and rules for atomic state mutations."
output_types: [safety-review, state-review, update-plan, state-review]
agent_roles: [builder, watcher, medic, architect, queen, fixer]
task_types: [state, update, safety, protected, mutation]
task_keywords: [state, protected, overwrite, update, atomic, lock, data, colony, corruption, checkpoint, paths]
workflow_triggers: [build, continue, update]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4200
---

# Protected Local State Contract

This contract defines which paths contain precious local state, what operations
may never overwrite them, and the requirements for atomic state mutations.

## For Beginners

Aether stores colony data locally in the `.aether/` directory. Some of this
data is irreplaceable -- colony state, user dreams, session checkpoints. This
contract ensures that updates, builds, and agents never accidentally destroy
this data. Think of it as a "do not touch" list for the colony's brain.

## Protected Paths

These paths contain local state that is **never distributed** and **never
overwritten** by `aether update`, `aether publish`, or any automated operation.

| Path | Contents | Why Protected |
|------|----------|---------------|
| `.aether/data/` | Colony state, pheromones, midden, handoffs | This is the colony's working memory |
| `.aether/dreams/` | Dream journal entries | User-created session notes |
| `.aether/checkpoints/` | Session checkpoints | Recovery points for paused colonies |
| `.aether/locks/` | File locks | Concurrency control state |

### What Lives in `.aether/data/`

| File | Purpose |
|------|---------|
| `COLONY_STATE.json` | Colony goal, phase, tasks, instincts, parallel mode |
| `pheromones.json` | Active signals (FOCUS, REDIRECT, FEEDBACK) |
| `constraints.json` | Legacy constraints (being deprecated) |
| `pending-decisions.json` | Decisions awaiting resolution |
| `assumptions.json` | Plan assumptions |
| `behavior-observations.jsonl` | Raw behavioral observations |
| `midden/midden.json` | Failure tracking |
| `survey/` | Territory survey results |
| `session.json` | Current session metadata |
| `handoffs/worker-handoffs.json` | Worker relay notes |

## Update Safety Rules

### What `aether update` May Touch

The update command syncs companion files from the hub to the local repo. It may
create or overwrite:

- `.claude/commands/ant/*.md` (slash command wrappers)
- `.opencode/commands/ant/*.md` (OpenCode command wrappers)
- `.codex/agents/*.toml` (Codex agent definitions)
- `.claude/agents/ant/*.md` (Claude agent definitions)
- `.opencode/agents/*.md` (OpenCode agent definitions)
- `.claude/rules/*.md` (development rules)
- `.codex/CODEX.md` (Codex rules)

### What `aether update` Must Never Touch

The update command must never overwrite, delete, or modify:

1. Any file in `.aether/data/`
2. Any file in `.aether/dreams/`
3. Any file in `.aether/checkpoints/`
4. Any file in `.aether/locks/`
5. User-modified files in `~/.aether/skills/domain/` (custom skills)

### Detection of User Modifications

When a shipped skill or reference has been modified by the user, the update
must preserve the user version. The manifest system tracks which files have
been locally modified and skips them during update.

## Atomic State Mutations

All writes to protected state files must be atomic to prevent corruption from
concurrent access or interrupted writes.

### Requirements

1. **Write to temp file first.** Never write directly to the target path.
   Write to a temporary file in the same directory, then rename.

2. **Rename is atomic.** On POSIX systems, `rename()` is atomic. The temp
   file becomes the real file in a single operation.

3. **File locking via `pkg/storage`.** The storage package provides
   file-level locking for concurrent access protection.

4. **No partial reads.** If a read finds a partially-written file (malformed
   JSON), it must fail gracefully with a clear error, not silently corrupt
   further.

### State Mutation Pattern

```
1. Acquire lock on target file
2. Read current state
3. Apply mutation in memory
4. Write to temp file (same directory)
5. Rename temp to target (atomic)
6. Release lock
```

### What Uses This Pattern

- `state-mutate` subcommand (colony state changes)
- `pheromone-write` subcommand (signal creation)
- `midden-write` subcommand (failure logging)
- `hive-store` subcommand (cross-colony wisdom)
- `memory-capture` subcommand (observation recording)

## Protected Commands

Some commands handle precious state and must never auto-clear their artifacts:

| Command | Why Protected |
|---------|---------------|
| `init` | `COLONY_STATE.json` is the colony's foundation |
| `seal` | Archives are permanent records |
| `entomb` | Chambers are permanent archives |

These commands are exempt from session freshness auto-clearing.

## Agent Obligations

**Builders MUST:**
- Use the `state-mutate` runtime command for all colony state changes
- Write to temp files first, then rename
- Acquire locks via `pkg/storage` before writing protected files
- Never directly edit `COLONY_STATE.json` with string replacement

**Watchers MUST:**
- Verify that update operations do not touch protected paths
- Flag any test that writes to `.aether/data/` without cleanup
- Check that builders follow the atomic write pattern

**Medics MUST:**
- Diagnose state corruption by checking for partial writes
- Repair by restoring from checkpoints when available
- Never overwrite user data without explicit confirmation

**All Agents MUST NOT:**
- Delete files in `.aether/data/` unless explicitly cleaning test artifacts
  via `/ant-data-clean`
- Bypass file locking for concurrent writes
- Store secrets or credentials in any `.aether/` path
