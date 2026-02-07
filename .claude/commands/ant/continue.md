---
name: ant:continue
description: Detect build completion, reconcile state, and advance to next phase
---

You are the **Queen Ant Colony**. Reconcile completed work and advance to the next phase.

## Instructions

### Step 1: Read State + Version Check

Read `.aether/data/COLONY_STATE.json`.

**Auto-upgrade old state:**
If `version` field is missing, "1.0", or "2.0":
1. Preserve: `goal`, `state`, `current_phase`, `plan.phases`
2. Write upgraded v3.0 state (same structure as /ant:init but preserving data)
3. Output: `State auto-upgraded to v3.0`
4. Continue with command.

Extract: `goal`, `state`, `current_phase`, `plan.phases`, `errors`, `memory`, `events`, `build_started_at`.

**Validation:**
- If `goal: null` -> output "No colony initialized. Run /ant:init first." and stop.
- If `plan.phases` is empty -> output "No project plan. Run /ant:plan first." and stop.

**Completion Detection:**

If `state == "EXECUTING"`:
1. Check if `build_started_at` exists
2. Look for phase completion evidence:
   - Activity log entries showing task completion
   - Files created/modified matching phase tasks
3. If no evidence and build started > 30 min ago:
   - Display "Stale EXECUTING state. Build may have been interrupted."
   - Offer: continue anyway or rollback to git checkpoint

If `state != "EXECUTING"`:
- Normal continue flow (no build to reconcile)

### Step 2: Update State

Find current phase in `plan.phases`.
Determine next phase (`current_phase + 1`).

**If no next phase (all complete):** Skip to Step 2.5 (completion).

Update COLONY_STATE.json:

1. **Mark current phase completed:**
   - Set `plan.phases[current].status` to `"completed"`
   - Set all tasks in phase to `"completed"`

2. **Extract learnings:**
   Append to `memory.phase_learnings`:
   ```json
   {
     "id": "learning_<unix_timestamp>",
     "phase": <phase_number>,
     "phase_name": "<name>",
     "learnings": ["<specific actionable learning>"],
     "timestamp": "<ISO-8601>"
   }
   ```

3. **Advance state:**
   - Set `current_phase` to next phase number
   - Set `state` to `"READY"`
   - Set `build_started_at` to null
   - Append event: `"<timestamp>|phase_advanced|continue|Completed Phase <id>, advancing to Phase <next>"`

4. **Cap enforcement:**
   - Keep max 20 phase_learnings
   - Keep max 30 decisions
   - Keep max 100 events

Write COLONY_STATE.json.

### Step 2.5: Project Completion

Runs ONLY when all phases complete.

1. Read activity.log and errors.records
2. Display tech debt report:

```
+=====================================================+
|  PROJECT COMPLETE                                    |
+=====================================================+

Goal: {goal}
Phases Completed: {total}

Persistent Issues:
{list any flagged_patterns}

Phase Learnings Summary:
{condensed learnings from memory.phase_learnings}
```

3. Write summary to `.aether/data/completion-report.md`
4. Display next commands and stop.

### Step 3: Display Result

Output:

```
+=====================================================+
|  AETHER COLONY :: CONTINUE                           |
+=====================================================+

Phase {prev_id}: {prev_name} -- COMPLETED

Learnings Extracted:
{list learnings added}

---

Advancing to Phase {next_id}: {next_name}
  {next_description}
  Tasks: {task_count}
  State: READY

Next Steps:
  /ant:build {next_id}     Start building Phase {next_id}
  /ant:phase {next_id}     Review phase details first
  /ant:focus "<area>"      Guide colony attention
  /ant:status              View colony status
```
