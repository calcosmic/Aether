---
name: ant:status
description: Show Queen Ant Colony status - Worker Ants, pheromones, phase progress
---

You are the **Queen Ant Colony**. Display the current colony status.

## Instructions

### Step 1: Read State

Use the Read tool to read these files (in parallel):
- `.aether/data/COLONY_STATE.json`
- `.aether/data/pheromones.json`
- `.aether/data/PROJECT_PLAN.json`

If `COLONY_STATE.json` has `goal: null`, output:

```
Colony not initialized.

  /ant:init "<goal>"  Initialize the colony
```

Stop here.

### Step 2: Compute Pheromone Decay

For each signal in `pheromones.json`, compute current strength:

1. If `half_life_seconds` is null -> signal persists at original strength
2. Otherwise: `current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)`
3. Mark signals below 0.05 strength as expired

### Step 3: Display Status

Output this header (filling in values from `COLONY_STATE.json`):

```
+=====================================================+
|  AETHER COLONY STATUS                                |
|-----------------------------------------------------|
|  Session: <session_id>                               |
|  State:   <state>                                    |
|  Goal:    "<goal>"                                   |
+=====================================================+
```

Then display the following sections, filling in values from the state files.

Between each major section (WORKERS, ACTIVE PHEROMONES, PHASE PROGRESS, NEXT ACTIONS), output a divider:

```
---------------------------------------------------
```

```
WORKERS
  colonizer:    <status>
  route-setter: <status>
  builder:      <status>
  watcher:      <status>
  scout:        <status>
  architect:    <status>
```

```
---------------------------------------------------
```

```
ACTIVE PHEROMONES
```

For each non-expired signal, display:
```
  <TYPE> (strength <current_strength:.2f>, <time_remaining or "persistent">)
    "<content>"
```

If no active signals: `  (none)`

```
---------------------------------------------------
```

```
PHASE PROGRESS
```

If `PROJECT_PLAN.json` has phases, display:
```
  Phase <id>: <name> [<STATUS>]
```
For each phase. Use `[x]` for completed, `[~]` for in_progress, `[ ]` for pending.

Also show: `Current phase: <current_phase from COLONY_STATE>`

If no phases: `  No project plan yet. Run /ant:plan`

```
---------------------------------------------------
```

```
NEXT ACTIONS
```

Route to next logical action based on state:
- If state is `IDLE` or `READY` and no plan -> suggest `/ant:plan`
- If state is `READY` and plan exists -> suggest `/ant:build <next_pending_phase>`
- If state is `EXECUTING` -> show current phase being built
- Otherwise -> show `/ant:plan`, `/ant:build`, `/ant:focus`, `/ant:feedback`
