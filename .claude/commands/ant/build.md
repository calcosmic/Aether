---
name: ant:build
description: Build a phase with pure emergence - colony self-organizes and completes tasks
---

You are the **Queen**. Spawn a Prime Worker and let the colony self-organize.

The phase to build is: `$ARGUMENTS`

## Instructions

### Step 1: Validate + Read State

If `$ARGUMENTS` is empty or not a number:

```
Usage: /ant:build <phase_number>

Example:
  /ant:build 1    Build Phase 1
  /ant:build 3    Build Phase 3
```

Stop here.

Read `.aether/data/COLONY_STATE.json`.

Extract:
- `goal`, `state`, `current_phase` from top level
- `plan.phases` for phase data
- `errors.records` for error context
- `memory` for decisions/learnings

**Validate:**
- If `goal: null` -> output `No colony initialized. Run /ant:init first.` and stop.
- If `plan.phases` is empty -> output `No project plan. Run /ant:plan first.` and stop.
- Find the phase matching the requested ID. If not found -> output `Phase {id} not found.` and stop.
- If the phase status is `"completed"` -> output `Phase {id} already completed.` and stop.

### Step 2: Update State

Read then update `.aether/data/COLONY_STATE.json`:
- Set `state` to `"EXECUTING"`
- Set `current_phase` to the phase number
- Set the phase's `status` to `"in_progress"` in `plan.phases[N]`
- Add `build_started_at` field with current ISO-8601 UTC timestamp
- Append to `events`: `"<timestamp>|phase_started|build|Phase <id>: <name> started"`

If `events` exceeds 100 entries, keep only the last 100.

Write COLONY_STATE.json.

### Step 3: Git Checkpoint

Create a git checkpoint for rollback capability.

```bash
git rev-parse --git-dir 2>/dev/null
```

- **If succeeds** (is a git repo): `git add -A && git commit --allow-empty -m "aether-checkpoint: pre-phase-$PHASE_NUMBER"`
  Store the commit hash.
- **If fails** (not a git repo): Set checkpoint hash to `"(not a git repo)"`.

Output header:

```
+=====================================================+
|  AETHER COLONY :: BUILD                              |
+=====================================================+

Phase {id}: {name}
Git Checkpoint: {commit_hash}
```

### Step 4: Load Constraints

Read `.aether/data/constraints.json` if it exists.

Format for display:
```
CONSTRAINTS:
  FOCUS: {focus areas, comma-separated}
  AVOID: {patterns to avoid from constraints}
```

If file doesn't exist or is empty:
```
CONSTRAINTS: (none)
```

### Step 5: Spawn Prime Worker

Log phase start:
```bash
bash ~/.aether/aether-utils.sh activity-log "PHASE_START" "queen" "Phase {id}: {name}"
```

Update watch status:
```
Write .aether/data/watch-status.txt:

AETHER COLONY :: EXECUTING
===========================

State: EXECUTING
Phase: {id}/{total_phases}

Active Workers:
  [Prime Worker] Organizing phase...

Last Activity:
  Prime Worker spawned
```

Spawn **one Prime Worker** via Task tool with `subagent_type="general-purpose"`:

```
You are the Prime Worker for Phase {id} in the Aether Colony.

You are at depth 1. You can spawn up to 4 specialists (depth 2).
Each specialist can spawn up to 2 sub-specialists (depth 3).
Depth 3 workers cannot spawn further.

--- PHASE CONTEXT ---

Goal: "{goal}"

Phase {id}: {phase_name}
{phase_description}

Tasks:
{for each task:}
  - {task_id}: {description}
    Depends on: {depends_on or "none"}
    Status: {status}
{end for}

Success Criteria:
{list success_criteria}

--- CONSTRAINTS ---
{constraints from Step 4, or "(none)"}

--- ERROR CONTEXT ---
{if errors.records has entries for this phase:}
Previous errors in this phase:
{list relevant errors}
{else:}
No previous errors.
{end if}

--- WORKER SPECS ---
Read ~/.aether/workers.md for role definitions and spawn protocol.

--- YOUR MISSION ---

1. Analyze the tasks and decide how to organize the work
2. For simple tasks (< 10 tool calls), do them yourself
3. Spawn specialists for complex/parallel work:
   - 🔨 Builder: code implementation, file manipulation
   - 👁️ Watcher: testing, validation, quality checks
   - 🔍 Scout: research, documentation lookup
   - 🗺️ Colonizer: codebase exploration
4. Synthesize all results
5. Verify success criteria are met
6. Log activity: bash ~/.aether/aether-utils.sh activity-log "ACTION" "caste" "description"

--- OUTPUT FORMAT ---

Return JSON:
{
  "status": "completed" | "failed" | "blocked",
  "summary": "What the phase accomplished",
  "tasks_completed": ["1.1", "1.2"],
  "tasks_failed": [],
  "files_created": ["path1", "path2"],
  "files_modified": ["path3"],
  "spawn_tree": {
    "builder-1": {"task": "...", "status": "completed", "children": {}},
    "watcher-1": {"task": "...", "status": "completed", "children": {}}
  },
  "quality_notes": "Any concerns or recommendations",
  "ui_touched": true | false
}
```

Wait for Prime Worker to complete.

### Step 6: Visual Checkpoint (if UI touched)

Parse Prime Worker result. If `ui_touched` is true:

```
Visual Checkpoint
=================

UI changes detected. Verify appearance before continuing.

Files touched:
{list files from files_created + files_modified that match UI patterns}

Options:
  1. Approve - UI looks correct
  2. Reject - needs changes (describe issues)
  3. Skip - defer visual review
```

Use AskUserQuestion to get approval. Record in events:
- If approved: `"<timestamp>|visual_approved|build|Phase {id} UI approved"`
- If rejected: `"<timestamp>|visual_rejected|build|Phase {id} UI rejected: {reason}"`

### Step 7: Display Results

Display build summary:

```
+=====================================================+
|  BUILD COMPLETE                                      |
+=====================================================+

Phase {id}: {name}
Status: {status}

Git Checkpoint: {commit_hash}

Summary:
{summary from Prime Worker}

Delegation Tree:
  Queen
  └── Prime Worker
{for each spawn in spawn_tree:}
      ├── {emoji} {caste}: {task} [{status}]
{for each child in spawn.children:}
      │   └── {emoji} {caste}: {task} [{status}]
{end for}
{end for}

Task Results:
{for each task in tasks_completed:}
  ✅ {task_id}: completed
{end for}
{for each task in tasks_failed:}
  ❌ {task_id}: failed
{end for}

Files:
  Created: {files_created count}
  Modified: {files_modified count}

Quality Notes:
{quality_notes from Prime Worker, or "None"}

Activity Log: .aether/data/activity.log

Next:
  /ant:continue            Advance to next phase
  /ant:feedback "<note>"   Give feedback first
  /ant:status              View full colony status
```

**IMPORTANT:** Build does NOT update task statuses or advance state. Run `/ant:continue` to:
- Mark tasks as completed
- Extract learnings
- Advance to next phase
