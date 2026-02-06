---
name: ant:status
description: Show Queen Ant Colony status - Worker Ants, pheromones, phase progress
---

You are the **Queen Ant Colony**. Display the current colony status.

## Instructions

### Step 1: Read State

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

This consolidated file contains all colony state:
- Top level: `goal`, `state`, `session_id`, `current_phase`, `version`
- `workers`: worker status map
- `spawn_outcomes`: per-caste spawn statistics
- `plan.phases`: project phases
- `signals`: pheromone signals
- `memory`: phase learnings, decisions, patterns
- `errors`: records, flagged_patterns
- `events`: event log as pipe-delimited strings

If `goal` is null, output:

```
Colony not initialized.

  /ant:init "<goal>"  Initialize the colony
```

Stop here.

**Validation:** Verify the content is valid JSON. If the file contains invalid JSON (corrupted data), output an error message:

```
  WARNING: COLONY_STATE.json contains invalid data.
  Recovery: Run /ant:init to reinitialize state.
```

### Step 2: Compute Pheromone Decay

Use the Bash tool to run:
```
bash ~/.aether/aether-utils.sh pheromone-batch
```

This returns JSON: `{"ok":true,"result":[...signals with current_strength...]}`. Parse the `result` array. Each signal object includes a `current_strength` field with the decayed value. Signals with `current_strength < 0.05` are effectively expired.

If the command fails (file not found, invalid JSON), treat as "no active pheromones."

### Step 2.5: Clean Expired Pheromones

If any signals from Step 2 had `current_strength < 0.05`, use the Bash tool to run:
```
bash ~/.aether/aether-utils.sh pheromone-cleanup
```

This removes expired signals from the `signals` array in `COLONY_STATE.json` and returns `{"ok":true,"result":{"removed":N,"remaining":N}}`.

If no signals are expired, skip this step.

### Step 3: Display Status

Show step progress at the start of output:

```
  ✓ Step 1: Read State
  ✓ Step 2: Compute Pheromone Decay
  ✓ Step 2.5: Clean Expired Pheromones
  ✓ Step 3: Display Status
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

Then display the following sections, filling in values from the state file.

Between each major section (WORKERS, ACTIVE PHEROMONES, ERRORS, MEMORY, EVENTS, PHASE PROGRESS, NEXT ACTIONS), output a divider:

```
---------------------------------------------------
```

```
WORKERS
```

Display workers grouped by their status from `workers` object:

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
🧪 ACTIVE PHEROMONES
```

For each non-expired signal in `signals` array, display with a visual strength bar:

```
  {TYPE padded to 10 chars} [{bar}] {current_strength:.2f}
    "{content}"
```

Where the bar has 20 characters total:
- Filled portion: repeat `█` (full block) for `round(current_strength * 20)` times
- Empty portion: fill remaining with spaces
- Wrap in square brackets

Examples:
```
  INIT       [████████████████████] 1.00  (persistent)
  FOCUS      [███████████████     ] 0.75
    "WebSocket security"
  REDIRECT   [██████              ] 0.30
    "Don't use JWT for sessions"
```

If no active signals after filtering:
```
  (no active pheromones)
```

If there ARE active signals, also display per-caste effective signals:

```
  Per-Caste Sensitivity:
```

For each active pheromone signal, compute `effective = sensitivity × current_strength` for each caste using:

```
                INIT  FOCUS  REDIRECT  FEEDBACK
  colonizer     1.0   0.7    0.3       0.5
  route-setter  1.0   0.5    0.8       0.7
  builder       0.5   0.9    0.9       0.7
  watcher       0.3   0.8    0.5       0.9
  scout         0.7   0.9    0.4       0.5
  architect     0.2   0.4    0.3       0.6
```

Display as a compact table showing only castes that would PRIORITIZE (effective > 0.5):

```
  Per-Caste Sensitivity:
    {SIGNAL_TYPE}: 🔨🐜 builder {effective:.2f}  🔍🐜 scout {effective:.2f}  👁️🐜 watcher {effective:.2f}
```

If no caste would PRIORITIZE a signal, show `(below action threshold for all castes)`.

```
---------------------------------------------------
```

```
💀 ERRORS
```

Read from `errors` object in COLONY_STATE.json:

Display flagged patterns first (if any exist in `flagged_patterns` array):
```
  ⚠️ FLAGGED PATTERNS:
    <category>: <count> occurrences — "<description from first error of that category>"
```

Then show recent errors (last 5 from `records` array, newest first):
```
  Recent:
    🔴 [critical] <category>: <description> (phase <phase>)
    🟠 [high] <category>: <description> (phase <phase>)
    🟡 [medium] <category>: <description> (phase <phase>)
    ⚪ [low] <category>: <description> (phase <phase>)
```

Use the severity emoji that matches: 🔴 critical, 🟠 high, 🟡 medium, ⚪ low.

If `records` array is empty and `flagged_patterns` is empty:
```
  (no errors recorded)
```

```
---------------------------------------------------
```

```
🧠 MEMORY
```

Read from `memory` object in COLONY_STATE.json:

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

```
---------------------------------------------------
```

```
📡 EVENTS
```

Read from `events` array in COLONY_STATE.json. Events are stored as pipe-delimited strings:

Format: `"<timestamp> | <type> | <source> | <content>"`

Parse each event by splitting on ` | ` (space-pipe-space):
- Index 0: timestamp (ISO-8601)
- Index 1: type (e.g., "colony_initialized", "phase_started")
- Index 2: source (e.g., "init", "builder")
- Index 3: content (human-readable message)

Display recent events (last 5 from `events` array, newest first):
```
  Recent:
    [<type>] <content> (<relative time, e.g., "2m ago", "1h ago">)
```

If `events` array is empty:
```
  (no events recorded)
```

```
---------------------------------------------------
```

```
PHASE PROGRESS
```

Read from `plan.phases` array in COLONY_STATE.json:

If phases exist, display:
```
  Phase <id>: <name> [<STATUS>]
```
For each phase. Use `[x]` for completed, `[~]` for in_progress, `[ ]` for pending.

Also show: `Current phase: <current_phase from top level>`

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
