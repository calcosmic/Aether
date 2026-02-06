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

### Step 2: Extend Signal TTLs by Pause Duration

Check if `paused_at` exists in `COLONY_STATE.json`:

```
If paused_at is set:
  current_time = current ISO-8601 UTC timestamp
  pause_duration = current_time - paused_at (in seconds/minutes)

  For each signal in pheromones.json signals array:
    if signal.expires_at == "phase_end":
      keep as-is (phase-scoped, no TTL to extend)
    elif signal.expires_at < current_time:
      skip (already expired before pause)
    else:
      expires_at = expires_at + pause_duration (extend TTL)

  Clear paused_at from COLONY_STATE.json
  Write updated COLONY_STATE.json
  Write updated pheromones.json with extended TTLs
```

This ensures signals don't expire during legitimate pauses.

### Step 2.5: Filter Active Signals

After TTL extension, filter signals for display:

```
current_time = current ISO-8601 UTC timestamp

For each signal in signals array:
  if signal.expires_at == "phase_end":
    keep (phase-scoped)
  elif signal.expires_at < current_time:
    skip (expired)
  else:
    keep (active, compute time remaining)
```

### Step 3: Display Restored State

Read the HANDOFF.md for context about what was happening, then display:

```
+=====================================================+
|  👑 AETHER COLONY :: RESUMED                         |
+=====================================================+

  Goal: "<goal>"
  State: <state>
  Session: <session_id>
  Phase: <current_phase>

ACTIVE SIGNALS
  {TYPE} [{priority}]: "{content}" ({time_remaining})

  Where time_remaining is:
  - "phase" if expires_at == "phase_end"
  - "45m left" / "2h left" for wall-clock expiration

  If no active signals: (no active signals)

WORKERS

  If ALL workers have "idle" status, display:
    All 6 workers idle -- colony ready

  Otherwise, group by status with caste emoji:
    Active:
      🔨🐜 builder: currently executing
    Idle:
      🗺️🐜 colonizer  📋🐜 route-setter  👁️🐜 watcher  🔍🐜 scout  🏛️🐜 architect

  Use the correct caste emoji for each worker:
    colonizer: 🗺️🐜  route-setter: 📋🐜  builder: 🔨🐜
    watcher: 👁️🐜  scout: 🔍🐜  architect: 🏛️🐜

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
