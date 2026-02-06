---
name: ant:status
description: Show colony status at a glance
---

You are the **Queen**. Show colony status.

## Instructions

### Step 1: Read State

Read `.aether/data/COLONY_STATE.json`.

If file missing or `goal: null`:
```
No colony initialized. Run /ant:init first.
```
Stop here.

### Step 2: Compute Summary

From state, extract:

**Phase info:**
- Current phase number: `plan.current_phase`
- Total phases: `plan.total_phases`
- Phase name: `plan.phases[current_phase - 1].name` (if phases exist)

**Task progress:**
- If phases exist, count tasks in current phase from `plan.phases[current_phase - 1].tasks`
- Completed: tasks with `status: "complete"`
- Total: all tasks in current phase

**Signal count:**
- Filter `signals` array: keep only where `expires_at` is "phase_end" OR `expires_at` timestamp > now
- Count remaining as active signals

**Worker count:**
- Count items in `workers` array (or object keys if object)

**Colony state:**
- `state` field (IDLE, EXECUTING, PAUSED)

### Step 3: Display

Output format (~5 lines):

```
Colony: <goal>
Phase <N>/<M>: <phase name>
Tasks: <completed>/<total> complete
Signals: <count> active | Workers: <count> active
State: <state> | Next: /ant:<suggested command>
```

**Suggested command logic:**
- IDLE -> continue
- EXECUTING -> continue
- PAUSED -> resume-colony

**Edge cases:**
- No phases yet: "Phase 0/0: No plan created"
- No tasks in phase: "Tasks: 0/0 complete"
- No signals: "Signals: 0 active"
- No workers: "Workers: 0 active"
