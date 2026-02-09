---
name: ant:pause-colony
description: "💾🐜⏸️🐜💾 Pause colony work and create handoff document for resuming later"
---

You are the **Queen Ant Colony**. Save current state for session handoff.

## Instructions

### Step 1: Read State

Use the read tool to read `.aether/data/COLONY_STATE.json`.

If `goal` is null, output `No colony initialized. Nothing to pause.` and stop.

### Step 2: Compute Active Signals

Read active signals from COLONY_STATE.json `signals` array.

Filter signals where:
- `expires_at` is null (permanent signals), OR
- `expires_at` > current timestamp (not expired)

If `signals` array is empty or all expired, treat as "no active pheromones."

### Step 3: Build Handoff Summary

Gather context for the handoff from `COLONY_STATE.json`:
- `goal` from top level
- `state` and `current_phase` from top level
- `workers` object
- Active signals from `signals` array
- Phase progress from `plan.phases`
- What was in progress or pending

### Step 4: Write Handoff

Use the write tool to update `.aether/HANDOFF.md`:

```markdown
# Colony Session Paused

## Quick Resume
Run `/ant:resume-colony` in a new session.

## State at Pause
- Goal: "<goal>"
- State: <state>
- Current Phase: <phase_number> - <phase_name>
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
|  AETHER COLONY :: PAUSED                            |
+=====================================================+

  Goal: "<goal>"
  Phase: <current_phase> - <phase_name>
  Pheromones: <active_count> active

  Handoff saved to .aether/HANDOFF.md

To resume in a new session:
  /ant:resume-colony
```
