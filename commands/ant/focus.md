---
name: ant:focus
description: Emit focus pheromone - guide colony attention to specific area
---

You are the **Queen Ant Colony**. Emit a FOCUS pheromone to guide colony attention.

## Instructions

The focus area is: `$ARGUMENTS`

### Step 1: Validate Input

If `$ARGUMENTS` is empty or blank, output:

```
Usage: /ant:focus "<area>" [--ttl <duration>]

Options:
  --ttl <duration>  Set expiration time (e.g., 30m, 2h, 1d). Default: phase_end

Examples:
  /ant:focus "WebSocket security"
  /ant:focus "database optimization" --ttl 2h
  /ant:focus "user authentication flow" --ttl 30m
```

Stop here.

### Step 2: Read State

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

If `goal` is null, output `No colony initialized. Run /ant:init first.` and stop.

### Step 3: Parse TTL Flag and Append FOCUS Signal

**Parse TTL:**
- If `$ARGUMENTS` contains `--ttl` followed by a duration:
  - Extract the duration value (e.g., "30m", "2h", "1d")
  - Parse duration: "m" = minutes, "h" = hours, "d" = days
  - Calculate `expires_at` = current timestamp + duration
  - Remove `--ttl <duration>` from the focus area content
- Otherwise: set `expires_at` = "phase_end" (default)

**Write Signal:**

Use the Read tool to read `.aether/data/pheromones.json`.

Add a new signal to the `signals` array and use the Write tool to write the updated file:

```json
{
  "id": "focus_<unix_timestamp>",
  "type": "FOCUS",
  "content": "<the focus area>",
  "priority": "normal",
  "created_at": "<ISO-8601 UTC timestamp>",
  "expires_at": "<ISO-8601 UTC timestamp or 'phase_end'>",
  "source": "user"
}
```

Preserve all existing signals in the array.

### Step 4: Log Decision

Read `.aether/data/memory.json`. Append a decision record to the `decisions` array:

```json
{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "focus",
  "content": "<the focus area>",
  "context": "Phase <current_phase> -- <colony state>",
  "phase": <current_phase from COLONY_STATE.json>,
  "timestamp": "<ISO-8601 UTC>"
}
```

If the `decisions` array exceeds 30 entries, remove the oldest entries to keep only 30.

Use the Write tool to write the updated memory.json.

### Step 5: Write Event

Read `.aether/data/events.json`. Append to the `events` array:

```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "pheromone_emitted",
  "source": "focus",
  "content": "FOCUS: <content> (priority normal, expires <time or 'phase end'>)",
  "timestamp": "<ISO-8601 UTC>"
}
```

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100.

Use the Write tool to write the updated events.json.

### Step 6: Display Result

Calculate time remaining:
- If `expires_at` is "phase_end": display "end of phase"
- Otherwise: calculate difference between `expires_at` and current time, format as "Xh Ym" or "Xm"

```
FOCUS pheromone emitted

  Area: "<focus area>"
  Priority: normal
  Expires: <time remaining or "end of phase">

  Workers will prioritize this area during the current phase.
  FOCUS signals guide attention without constraining approaches.

Next Steps:
  /ant:redirect "<pattern>"  Warn colony away from something
  /ant:status                View all active signals
  /ant:build <phase>         Start building (focus will influence workers)
```
