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
- `.aether/data/errors.json`
- `.aether/data/memory.json`
- `.aether/data/events.json`

If `COLONY_STATE.json` has `goal: null`, output:

```
Colony not initialized.

  /ant:init "<goal>"  Initialize the colony
```

Stop here.

**Validation:** After reading each state file, verify the content is valid JSON. If any file contains invalid JSON (corrupted data), output an error message:

```
  WARNING: <filename> contains invalid data.
  Recovery: Run /ant:init to reinitialize state files.
```

Continue displaying status for the files that are valid. Skip sections for corrupted files.

### Step 2: Display Status

Show step progress at the start of output:

```
  ✓ Step 1: Read State
  ✓ Step 2: Display Status
```

Then output the status display below.

Output this header (filling in values from `COLONY_STATE.json`):

```
+=====================================================+
|  👑 AETHER COLONY STATUS                             |
|-----------------------------------------------------|
|  Session: <session_id>                               |
|  State:   <state>                                    |
|  Goal:    "<goal>"                                   |
+=====================================================+
```

Then display the following sections, filling in values from the state files.

Between each major section (WORKERS, ACTIVE PHEROMONES, ERRORS, MEMORY, EVENTS, PHASE PROGRESS, NEXT ACTIONS), output a divider:

```
---------------------------------------------------
```

```
WORKERS
```

Display workers grouped by their status from `COLONY_STATE.json`:

If ALL workers have `"idle"` status (the common case), display a compact summary:
```
  All 6 workers idle -- colony ready
```

Otherwise, group by status with caste emoji:

```
  Active:
    🔨🐜 builder: executing phase 3

  Idle:
    🗺️🐜 colonizer  📋🐜 route-setter  👁️🐜 watcher  🔍🐜 scout  🏛️🐜 architect
```

Use the correct caste emoji for each worker:
- colonizer: 🗺️🐜  route-setter: 📋🐜  builder: 🔨🐜
- watcher: 👁️🐜  scout: 🔍🐜  architect: 🏛️🐜

Only show groups that have at least one worker. End with a summary line:
```
  🐜 <N> active | ⚪ <N> idle
```

```
---------------------------------------------------
```

```
🧪 ACTIVE SIGNALS
```

**Filter signals before display:**
```
current_time = current ISO-8601 UTC timestamp

For each signal in pheromones.json signals array:
  if signal.expires_at == "phase_end":
    keep (phase-scoped, always active during phase)
  elif signal.expires_at < current_time:
    skip (expired)
  else:
    keep (active, compute time remaining)
```

**Display format (grouped by priority):**

Show HIGH priority signals first, then NORMAL, then LOW:

```
  {TYPE} [{priority}]: "{content}" ({time_remaining})
```

Where time_remaining is:
- "phase" if expires_at == "phase_end"
- "12m left" / "2h left" if wall-clock expiration (compute from expires_at - current_time)
- For expired signals: skip entirely (already filtered above)

Examples:
```
  REDIRECT [high]: "Don't use JWT for sessions" (phase)
  FOCUS [normal]: "WebSocket security" (45m left)
  FEEDBACK [low]: "Good test coverage" (2h left)
```

If no active signals after filtering:
```
  (no active signals)
```

```
---------------------------------------------------
```

```
💀 ERRORS
```

If `errors.json` was read successfully and has content:

Display flagged patterns first (if any exist in `flagged_patterns` array):
```
  ⚠️ FLAGGED PATTERNS:
    <category>: <count> occurrences — "<description from first error of that category>"
```

Then show recent errors (last 5 from `errors` array, newest first):
```
  Recent:
    🔴 [critical] <category>: <description> (phase <phase>)
    🟠 [high] <category>: <description> (phase <phase>)
    🟡 [medium] <category>: <description> (phase <phase>)
    ⚪ [low] <category>: <description> (phase <phase>)
```

Use the severity emoji that matches: 🔴 critical, 🟠 high, 🟡 medium, ⚪ low.

If `errors` array is empty and `flagged_patterns` is empty:
```
  (no errors recorded)
```

If `errors.json` doesn't exist or couldn't be read, skip this section silently.

```
---------------------------------------------------
```

```
🧠 MEMORY
```

If `memory.json` was read successfully and has content:

Display recent phase learnings (last 3 from `phase_learnings` array, newest first):
```
  Recent Learnings:
    Phase <phase>: <first learning from learnings array>
    Phase <phase>: <first learning from learnings array>
    Phase <phase>: <first learning from learnings array>
```

Display decision count:
```
  Decisions logged: <count of decisions array>
```

If `phase_learnings` array is empty and `decisions` array is empty:
```
  (no memory recorded)
```

If `memory.json` doesn't exist or couldn't be read, skip this section silently.

```
---------------------------------------------------
```

```
📡 EVENTS
```

If `events.json` was read successfully and has content:

Display recent events (last 5 from `events` array, newest first):
```
  Recent:
    [<type>] <content> (<relative time, e.g., "2m ago", "1h ago">)
```

If `events` array is empty:
```
  (no events recorded)
```

If `events.json` doesn't exist or couldn't be read, skip this section silently.

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
