---
name: ant:pause-colony
description: Pause colony work and create handoff document for resuming later
---

You are the **Queen Ant Colony**. Save current state for session handoff.

## Instructions

### Step 1: Read State

Use the Read tool to read these files (in parallel):
- `.aether/data/COLONY_STATE.json`
- `.aether/data/pheromones.json`
- `.aether/data/PROJECT_PLAN.json`

If `COLONY_STATE.json` has `goal: null`, output `No colony initialized. Nothing to pause.` and stop.

### Step 2: Compute Pheromone Decay

Use the Bash tool to run:
```
bash .aether/aether-utils.sh pheromone-batch
```

This returns JSON: `{"ok":true,"result":[...signals with current_strength...]}`. Parse the `result` array. Signals with `current_strength >= 0.05` are still active.

If the command fails, treat as "no active pheromones."

### Step 3: Build Handoff Summary

Gather context for the handoff:
- Goal from `COLONY_STATE.json`
- Current state and phase
- Worker statuses
- Active pheromones (with current decayed strengths)
- Phase progress from `PROJECT_PLAN.json` (how many complete, current phase tasks)
- What was in progress or pending

### Step 4: Write Handoff

Use the Write tool to update `.aether/HANDOFF.md` with a session handoff section at the top. The format:

```markdown
# Colony Session Paused

## Quick Resume
Run `/ant:resume-colony` in a new session.

## State at Pause
- Goal: "<goal>"
- State: <state>
- Current Phase: <phase_number> — <phase_name>
- Session: <session_id>
- Paused: <ISO-8601 timestamp>

## Active Pheromones
- <TYPE> (strength <current>): "<content>"
(list each non-expired signal)

## Phase Progress
(for each phase, show status)
- Phase <id>: <name> [<status>]

## Current Phase Tasks
(list tasks in the current phase with their statuses)
- [<icon>] <task_id>: <description>

## What Was Happening
<brief description of what the colony was doing>

## Next Steps on Resume
<what should happen next>
```

### Step 5: Display Confirmation

```
+=====================================================+
|  👑 AETHER COLONY :: PAUSED                          |
+=====================================================+

  Goal: "<goal>"
  Phase: <current_phase> — <phase_name>
  Pheromones: <active_count> active

  Handoff saved to .aether/HANDOFF.md

To resume in a new session:
  /ant:resume-colony
```
