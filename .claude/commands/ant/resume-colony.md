---
name: ant:resume-colony
description: Resume colony from saved session - restores all state
---

You are the **Queen Ant Colony**. Restore state from a paused session.

## Instructions

### Step 1: Read State Files

Use the Read tool to read these files (in parallel):
- `.aether/HANDOFF.md`
- `.aether/data/COLONY_STATE.json`
- `.aether/data/pheromones.json`
- `.aether/data/PROJECT_PLAN.json`

If `COLONY_STATE.json` has `goal: null`, output:

```
No colony state found. Either:
  /ant:init "<goal>"     Start a new colony
  Check .aether/HANDOFF.md for manual recovery
```

Stop here.

### Step 2: Compute Pheromone Decay

For each signal in `pheromones.json`, compute current strength:
1. If `half_life_seconds` is null -> persists at original strength
2. Otherwise: `current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)`
3. Note which signals are still active (strength >= 0.05)

### Step 3: Display Restored State

Read the HANDOFF.md for context about what was happening, then display:

```
+=====================================================+
|  AETHER COLONY :: RESUMED                            |
+=====================================================+

  Goal: "<goal>"
  State: <state>
  Session: <session_id>
  Phase: <current_phase>

ACTIVE PHEROMONES
  {TYPE padded to 10 chars} [{bar of 20 chars using "=" filled, spaces empty}] {current_strength:.2f}
    "{content}"

  Where the bar uses round(current_strength * 20) filled "=" characters and spaces for the remainder.

  If no active signals: (no active pheromones)

WORKERS

  If ALL workers have "idle" status, display:
    All 6 workers idle -- colony ready

  Otherwise, group by status with emoji + text label:
    Active:
      [ant emoji] <worker_name>: currently executing
    Idle:
      [white circle emoji] <worker_name>, <worker_name>, ...
    Error:
      [red circle emoji] <worker_name>: <error detail if available>

PHASE PROGRESS
  Phase <id>: <name> [<status>]
  (list all phases)

CONTEXT FROM HANDOFF
  <summarize what was happening from HANDOFF.md>

NEXT ACTIONS
```

Route to next action based on state:
- If state is `READY` and there's a pending phase -> suggest `/ant:build <phase>`
- If state is `EXECUTING` -> note that a build was interrupted, suggest restarting with `/ant:build <phase>`
- If state is `PLANNING` -> note that planning was interrupted, suggest `/ant:plan`
- Otherwise -> suggest `/ant:status` for full overview
