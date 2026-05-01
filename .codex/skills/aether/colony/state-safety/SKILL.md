---
name: state-safety
description: Use when reading or writing JSON state files including COLONY_STATE.json, pheromones.json, constraints.json, or session.json
type: colony
domains: [security, state, data-integrity, safety]
agent_roles: [builder, watcher]
priority: normal
version: "1.0"
---

# State Safety

## Purpose

Colony state files are critical. A corrupted COLONY_STATE.json can lose an entire colony's progress. A malformed pheromones.json can silently break signal injection. This skill teaches safe state file handling to prevent data loss.

## Atomic Writes

All JSON state mutations must use atomic write: write to a temporary file first, then rename to the target. This prevents partial writes from corrupting the file if the process is interrupted.

### Process

1. Write the new content to a temporary file in the same directory (e.g., `COLONY_STATE.json.tmp`).
2. Validate the temporary file is valid JSON: `jq . file.tmp > /dev/null 2>&1`.
3. If validation passes, rename the temp file to the target: `mv file.tmp file.json`.
4. If validation fails, delete the temp file and report the error -- never overwrite good data with bad.

Use `atomic-write` utility when available. If writing state manually, follow these four steps exactly.

## File Locking

Before modifying any state file, acquire a lock to prevent concurrent writes:

1. Call `file-lock acquire <file>` before the write operation.
2. Perform the read-modify-write cycle.
3. Call `file-lock release <file>` after the write completes.
4. Always release locks in error paths -- use trap or ensure-release patterns.

Lock contention should be rare (colony operations are typically single-threaded), but parallel worker spawns during builds can cause races on shared files like pheromones.json.

## Post-Write Validation

After every state file write, validate the result:

```bash
jq . "$state_file" > /dev/null 2>&1
```

If validation fails:
1. Log a warning with the file path and the operation that triggered the write.
2. Check for a backup (`.bak` extension in the same directory).
3. If a backup exists and is valid, restore from backup and log the restoration.
4. If no valid backup exists, report the corruption as a critical error.

Never silently ignore a JSON parse failure. A silent parse error today becomes a mysterious crash tomorrow.

## Corruption Detection

When reading state files, validate before using the data:

- Check that the file exists and is non-empty.
- Parse with `jq` and check the exit code.
- Verify expected top-level keys exist (e.g., COLONY_STATE.json must have `goal`, `state`, `current_phase`).
- If any check fails, follow the fallback chain: try backup, then report error.

## Backup Strategy

Before making significant state changes (phase advances, plan regeneration, seal), create a backup:

```bash
cp "$state_file" "${state_file}.bak"
```

Keep exactly one backup per file. The backup represents the last known good state.

## State Files and Their Critical Fields

| File | Critical Fields | Risk if Corrupted |
|------|----------------|-------------------|
| COLONY_STATE.json | goal, state, current_phase, plan | Total colony loss |
| pheromones.json | signals array | Silent signal failure |
| constraints.json | focus, constraints | Lost constraints |
| session.json | session_id, colony_goal | Resume failure |
| midden.json | entries array | Lost failure history |
