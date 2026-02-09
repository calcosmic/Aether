---
name: ant:migrate-state
description: "🚚🐜📦🐜🚚 One-time state migration utility"
---

# /ant:migrate-state - One-Time State Migration

Migrate colony state from v1 (6-file) format to v2.0 (consolidated single-file) format.

**Usage:** Run once to migrate existing state. Safe to run multiple times - skips if already migrated.

---

## Step 1: Check Migration Status

Read `.aether/data/COLONY_STATE.json` to check if already migrated.

**If file contains `"version": "2.0"` or `"version": "3.0"`:**
- Output: "State already migrated. No action needed."
- Stop execution.

**If no version field or version < 2.0:**
- Continue to Step 2.

---

## Step 2: Read All State Files

Use the read tool to read all 6 state files from `.aether/data/`:

1. `COLONY_STATE.json` - Colony goal, state machine, workers, spawn outcomes
2. `PROJECT_PLAN.json` - Phases, tasks, success criteria
3. `pheromones.json` - Active signals
4. `memory.json` - Phase learnings, decisions, patterns
5. `errors.json` - Error records, flagged patterns
6. `events.json` - Event log

Handle missing files gracefully (use empty defaults).

---

## Step 3: Construct Consolidated State

Build the v3.0 consolidated structure:

```json
{
  "version": "3.0",
  "goal": "<from COLONY_STATE.goal or null>",
  "state": "<from COLONY_STATE.state or 'IDLE'>",
  "current_phase": "<from COLONY_STATE.current_phase or 0>",
  "session_id": "<from COLONY_STATE.session_id or null>",
  "initialized_at": "<from COLONY_STATE.initialized_at or null>",
  "build_started_at": null,
  "plan": {
    "generated_at": "<from PROJECT_PLAN.generated_at or null>",
    "confidence": null,
    "phases": "<from PROJECT_PLAN.phases or []>"
  },
  "memory": {
    "phase_learnings": "<from memory.phase_learnings or []>",
    "decisions": "<from memory.decisions or []>",
    "instincts": []
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

Write the new consolidated COLONY_STATE.json with the v3.0 structure.

---

## Step 6: Display Summary

Output a migration summary:

```
State Migration Complete (v1 -> v3.0)
======================================

Migrated data:
- Goal: <goal or "(not set)">
- State: <state>
- Current phase: <phase>
- Plan phases: <count>
- Phase learnings: <count>
- Error records: <count>
- Events: <count>

Files backed up to: .aether/data/backup-v1/
New state file: .aether/data/COLONY_STATE.json (v3.0)

All commands now use consolidated state format.
```
