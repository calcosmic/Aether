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
Usage: /ant:feedback "<message>"

Examples:
  /ant:feedback "Great progress on the API layer"
  /ant:feedback "Need more test coverage"
  /ant:feedback "Too slow, simplify the approach"
  /ant:feedback "Wrong direction, reconsider architecture"
```

Stop here.

### Step 2: Read State

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

If `goal` is null, output `No colony initialized. Run /ant:init first.` and stop.

Extract:
- `goal` from top level
- `current_phase` from top level

### Step 3: Update State (Single Read-Modify-Write)

Read `.aether/data/COLONY_STATE.json` (if not already in memory from Step 2).

Generate a Unix timestamp and 4 random hex characters for IDs.

Modify the state:

**1. Append to `signals` array:**
```json
{
  "id": "feedback_<unix_timestamp>",
  "type": "FEEDBACK",
  "content": "<the feedback message>",
  "strength": 0.5,
  "half_life_seconds": 21600,
  "created_at": "<ISO-8601 UTC timestamp>"
}
```

**2. Append to `memory.decisions` array:**
```json
{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "feedback",
  "content": "<the feedback message>",
  "context": "Phase <current_phase> -- <colony state>",
  "phase": <current_phase>,
  "timestamp": "<ISO-8601 UTC>"
}
```
If `memory.decisions` exceeds 30 entries, remove the oldest to keep only 30.

**3. Append to `events` array as pipe-delimited string:**
```
"<ISO-8601 UTC> | pheromone_emitted | feedback | FEEDBACK: <content> (strength 0.5, half-life 6hr)"
```
If `events` exceeds 100 entries, remove the oldest to keep only 100.

Use the Write tool to write the FULL updated state back to `.aether/data/COLONY_STATE.json`.

### Step 4: Display Result

```
FEEDBACK pheromone emitted

  Message: "<feedback>"
  Strength: 0.5
  Half-life: 6 hours

  Colony response by sensitivity:
    watcher (0.9)      -- strong: will intensify verification
    builder (0.7)      -- moderate: will adjust implementation
    route-setter (0.7) -- moderate: will adjust planning
    architect (0.6)    -- moderate: will record for learning
    colonizer (0.5)    -- moderate: will adjust exploration
    scout (0.5)        -- moderate: will adjust research focus

  FEEDBACK can be emitted at any time, even during /ant:build.
  It provides gentle guidance without breaking emergence.

Next Steps:
  /ant:status          View colony status and all pheromones
  /ant:focus "<area>"  Strengthen attention on specific area
```
