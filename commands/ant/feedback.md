---
name: ant:feedback
description: Emit FEEDBACK pheromone - provide guidance to colony based on observations
---

You are the **Queen Ant Colony**. Emit a FEEDBACK pheromone to adjust colony behavior.

## Instructions

The feedback message is: `$ARGUMENTS`

### Step 1: Validate Input

If `$ARGUMENTS` is empty or blank, output:

```
Usage: /ant:feedback "<message>" [--ttl <duration>]

Options:
  --ttl <duration>  Set expiration time (e.g., 30m, 2h, 1d). Default: phase_end

Examples:
  /ant:feedback "Great progress on the API layer"
  /ant:feedback "Need more test coverage" --ttl 2h
  /ant:feedback "Too slow, simplify the approach"
  /ant:feedback "Wrong direction, reconsider architecture"
```

Stop here.

### Step 2: Read State

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

If `goal` is null, output `No colony initialized. Run /ant:init first.` and stop.

### Step 3: Parse TTL Flag and Append FEEDBACK Signal

**Parse TTL:**
- If `$ARGUMENTS` contains `--ttl` followed by a duration:
  - Extract the duration value (e.g., "30m", "2h", "1d")
  - Parse duration: "m" = minutes, "h" = hours, "d" = days
  - Calculate `expires_at` = current timestamp + duration
  - Remove `--ttl <duration>` from the feedback content
- Otherwise: set `expires_at` = "phase_end" (default)

**Write Signal:**

Use the Read tool to read `.aether/data/pheromones.json`.

Add a new signal to the `signals` array and use the Write tool to write the updated file:

```json
{
  "id": "feedback_<unix_timestamp>",
  "type": "FEEDBACK",
  "content": "<the feedback message>",
  "priority": "low",
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
  "type": "feedback",
  "content": "<the feedback message>",
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
  "source": "feedback",
  "content": "FEEDBACK: <content> (priority low, expires <time or 'phase end'>)",
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
FEEDBACK pheromone emitted

  Message: "<feedback>"
  Priority: low
  Expires: <time remaining or "end of phase">

  FEEDBACK can be emitted at any time, even during /ant:build.
  It provides gentle guidance without breaking emergence.

Next Steps:
  /ant:status          View colony status and all signals
  /ant:focus "<area>"  Strengthen attention on specific area
```
