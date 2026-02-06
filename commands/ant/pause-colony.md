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

### Step 2: Filter Active Signals

Filter signals from `pheromones.json` using TTL:

```
current_time = current ISO-8601 UTC timestamp

For each signal in signals array:
  if signal.expires_at == "phase_end":
    keep (phase-scoped)
  elif signal.expires_at < current_time:
    skip (expired)
  else:
    keep (active)
```

Count active signals for the display in Step 5.

### Step 3: Build Handoff Summary

Gather context for the handoff:
- Goal from `COLONY_STATE.json`
- Current state and phase
- Worker statuses
- Active pheromones (with current decayed strengths)
- Phase progress from `PROJECT_PLAN.json` (how many complete, current phase tasks)
- What was in progress or pending

### Step 4: Write Handoff and Pause Timestamp

**4a. Record paused_at timestamp:**

Use the Write tool to update `COLONY_STATE.json`:
- Set `paused_at` to the current ISO-8601 UTC timestamp

This tracks when the colony was paused for TTL extension on resume.

**4b. Write HANDOFF.md:**

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

## Active Signals
- {TYPE} [{priority}]: "{content}" ({time_remaining})
(list each non-expired signal with priority and time remaining)

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
