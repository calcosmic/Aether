---
name: ant:resume-colony
description: 🚦➡️🐜💨💨 Resume colony from saved session - restores all state
---

You are the **Queen Ant Colony**. Restore state from a paused session.

## Instructions

### Step 1: Read State Files

Use the Read tool to read these files (in parallel):
- `.aether/HANDOFF.md`
- `.aether/data/COLONY_STATE.json`

If `COLONY_STATE.json` has `goal: null`, output:

```
No colony state found. Either:
  /ant:init "<goal>"     Start a new colony
  Check .aether/HANDOFF.md for manual recovery
```

Stop here.

### Step 2: Compute Active Signals

Read active signals from COLONY_STATE.json `signals` array (already loaded in Step 1).

Filter signals where:
- `expires_at` is null (permanent signals like INIT), OR
- `expires_at` > current timestamp (not expired)

If `signals` array is empty or all expired, treat as "no active pheromones."

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
  {TYPE padded to 10 chars} [{bar of 20 chars using filled/empty}] {current_strength:.2f}
    "{content}"

  Where the bar uses round(current_strength * 20) filled characters and spaces for the remainder.

  If no active signals: (no active pheromones)

WORKERS

  If ALL workers have "idle" status, display:
    All 6 workers idle -- colony ready

  Otherwise, group by status with caste name:
    Active:
      builder: currently executing
    Idle:
      colonizer  route-setter  watcher  scout  architect

PHASE PROGRESS
  Phase <id>: <name> [<status>]
  (list all phases from plan.phases)

CONTEXT FROM HANDOFF
  <summarize what was happening from HANDOFF.md>

NEXT ACTIONS
```

Route to next action based on state:
- If state is `READY` and there's a pending phase -> suggest `/ant:build <phase>`
- If state is `EXECUTING` -> note that a build was interrupted, suggest restarting with `/ant:build <phase>`
- If state is `PLANNING` -> note that planning was interrupted, suggest `/ant:plan`
- Otherwise -> suggest `/ant:status` for full overview
