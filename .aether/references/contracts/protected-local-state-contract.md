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
| `.aether/data/` (except sanctioned scratch subpaths below) | Colony state, pheromones, midden, handoffs | This is the colony's working memory |
| `.aether/dreams/` | Dream journal entries | User-created session notes |
| `.aether/checkpoints/` | Session checkpoints | Recovery points for paused colonies |
| `.aether/locks/` | File locks | Concurrency control state |

**Sanctioned scratch subpaths:** `.aether/data/planning/`, `.aether/data/phase-research/`,
`.aether/data/survey/`, and `.aether/data/worker-debug/` are exempt from the
blanket protection above — a worker following its own task brief is expected
to write here (e.g. a scout persisting phase research, a surveyor persisting
territory survey artifacts). `protectedHookWriteReason`
(`cmd/hook_cmds.go`) enforces this as an exact-subpath allowlist on Claude
Code via the `PreToolUse` hook; every other path under `.aether/data/` stays
blocked. Everything else in this contract's "never overwritten" guarantee is
unaffected — `aether update`/`aether publish` still never touch any file
under `.aether/data/`, sanctioned or not.

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
| `planning/` | Planning artifacts a worker persists during the plan workflow (sanctioned scratch subpath, worker-writable) |
| `phase-research/` | Phase domain research written by a scout (sanctioned scratch subpath, worker-writable) |
| `survey/` | Territory survey results (sanctioned scratch subpath, worker-writable) |
| `worker-debug/` | Worker debug artifacts for diagnostics (sanctioned scratch subpath, worker-writable) |
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

1. Any file in `.aether/data/` — including the sanctioned scratch subpaths
   (`planning/`, `phase-research/`, `survey/`, `worker-debug/`). Those
   subpaths are worker-writable via the `PreToolUse` hook allowlist, but that
   is a separate protection domain from this update-safety guarantee: `aether
   update`/`aether publish` never touch any file under `.aether/data/`,
   sanctioned or not.
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

## Residual Risks

These are known, accepted trade-offs in the current enforcement model, not
defects — documented here so the allowlist description above does not
overstate what is actually mechanically guaranteed.

### Enforcement is asymmetric across platforms

`protectedHookWriteReason` (`cmd/hook_cmds.go`) is the only *mechanical*
enforcement of this contract, and it only runs on Claude Code via the
`PreToolUse` hook. On OpenCode and Codex there is no equivalent hook:

- `.opencode/OPENCODE.md` describes its restrictions as "conduct, not a
  sandbox" — a worker on OpenCode is only asked, in prose, to respect
  protected paths and the sanctioned scratch subpaths.
- Codex native has no hook mechanism at all; the same restrictions exist
  only as behavioral text in the worker's task brief.

Concretely: removing `scout` from `repositoryReadOnlyCastes`
(`pkg/codex/permission_profile.go`) grants `workspace_write` plus a prose
restriction ("write phase research artifacts under
`.aether/data/phase-research` only"). A scout on Claude Code that tries to
write outside that scope is mechanically blocked; the identical scout on
OpenCode or Codex is not — nothing but the model's compliance with its own
instructions stands in the way. This is a deliberate scope trade for this
phase (D-05), not an oversight, but it means "protected" in this document
means "protected on Claude Code" until an equivalent hook exists for the
other two platforms.

### Symlink resolution narrows, but does not eliminate, the lexical gap

`normalizeHookPath` resolves symlinks on the deepest existing ancestor
(`filepath.EvalSymlinks`) before allowlist/blocklist matching, so a symlink
planted inside a sanctioned scratch subdir (e.g.
`.aether/data/planning/link -> ../COLONY_STATE.json`) can no longer redirect
a write onto protected state through that specific vector. This only
hardens the Claude Code hook path described above — it does nothing for
OpenCode or Codex, which have no matching-and-resolution step to harden in
the first place.

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
