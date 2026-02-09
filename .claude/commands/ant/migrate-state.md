# 🚚🐜📦🐜🚚 /ant:migrate-state - One-Time State Migration

Migrate colony state from v1 (6-file) format to v2.0 (consolidated single-file) format.

**Usage:** Run once to migrate existing state. Safe to run multiple times - skips if already migrated.

---

## Step 1: Check Migration Status

Read `.aether/data/COLONY_STATE.json` to check if already migrated.

**If file contains `"version": "2.0"`:**
- Output: "State already migrated to v2.0. No action needed."
- Stop execution.

**If no version field or version < 2.0:**
- Continue to Step 2.

---

## Step 2: Read All State Files

Use the Read tool to read all 6 state files from `.aether/data/`:

1. `COLONY_STATE.json` - Colony goal, state machine, workers, spawn outcomes
2. `PROJECT_PLAN.json` - Phases, tasks, success criteria
3. `pheromones.json` - Active signals
4. `memory.json` - Phase learnings, decisions, patterns
5. `errors.json` - Error records, flagged patterns
6. `events.json` - Event log

Handle missing files gracefully (use empty defaults).

---

## Step 3: Construct Consolidated State

Build the v2.0 consolidated structure:

```json
{
  "version": "2.0",
  "goal": "<from COLONY_STATE.goal or null>",
  "state": "<from COLONY_STATE.state or 'IDLE'>",
  "current_phase": "<from COLONY_STATE.current_phase or 0>",
  "session_id": "<from COLONY_STATE.session_id or null>",
  "initialized_at": "<from COLONY_STATE.initialized_at or null>",
  "mode": "<from COLONY_STATE.mode or null>",
  "mode_set_at": "<from COLONY_STATE.mode_set_at or null>",
  "mode_indicators": "<from COLONY_STATE.mode_indicators or null>",
  "workers": "<from COLONY_STATE.workers or default idle workers>",
  "spawn_outcomes": "<from COLONY_STATE.spawn_outcomes or default outcomes>",
  "plan": {
    "generated_at": "<from PROJECT_PLAN.generated_at or null>",
    "phases": "<from PROJECT_PLAN.phases or []>"
  },
  "signals": "<from pheromones.signals or []>",
  "memory": {
    "phase_learnings": "<from memory.phase_learnings or []>",
    "decisions": "<from memory.decisions or []>",
    "patterns": "<from memory.patterns or []>"
  },
  "errors": {
    "records": "<from errors.errors or []>",
    "flagged_patterns": "<from errors.flagged_patterns or []>"
  },
  "events": "<converted event strings or []>"
}
```

**Event Conversion:**
Convert each event object to a pipe-delimited string:
- Old format: `{"id":"evt_123","type":"colony_initialized","source":"init","content":"msg","timestamp":"2026-..."}`
- New format: `"2026-... | colony_initialized | init | msg"`

If events array is already strings (or empty), keep as-is.

---

## Step 4: Create Backup

Create backup directory and move old files:

```bash
mkdir -p .aether/data/backup-v1
```

Move these files to backup (if they exist):
- `.aether/data/PROJECT_PLAN.json` -> `.aether/data/backup-v1/PROJECT_PLAN.json`
- `.aether/data/pheromones.json` -> `.aether/data/backup-v1/pheromones.json`
- `.aether/data/memory.json` -> `.aether/data/backup-v1/memory.json`
- `.aether/data/errors.json` -> `.aether/data/backup-v1/errors.json`
- `.aether/data/events.json` -> `.aether/data/backup-v1/events.json`
- `.aether/data/COLONY_STATE.json` -> `.aether/data/backup-v1/COLONY_STATE.json`

---

## Step 5: Write Consolidated State

Write the new consolidated COLONY_STATE.json with the v2.0 structure from Step 3.

Format the JSON with 2-space indentation for readability.

---

## Step 6: Display Summary

Output a migration summary:

```
State Migration Complete (v1 -> v2.0)
======================================

Migrated data:
- Goal: <goal or "(not set)">
- State: <state>
- Current phase: <phase>
- Workers: <count of non-idle workers>
- Plan phases: <count>
- Active signals: <count>
- Phase learnings: <count>
- Decisions: <count>
- Error records: <count>
- Events: <count>

Files backed up to: .aether/data/backup-v1/
New state file: .aether/data/COLONY_STATE.json (v2.0)

All commands now use consolidated state format.
```

---

## Notes

- This is a one-time migration command
- After v5.1 ships, this command can be removed
- All 12+ ant commands will be updated to use the new single-file format
- The backup directory preserves original files for rollback if needed
