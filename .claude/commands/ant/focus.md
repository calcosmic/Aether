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
Usage: /ant:focus "<area>"

Examples:
  /ant:focus "WebSocket security"
  /ant:focus "database optimization"
  /ant:focus "user authentication flow"
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
  "id": "focus_<unix_timestamp>",
  "type": "FOCUS",
  "content": "<the focus area>",
  "strength": 0.7,
  "half_life_seconds": 3600,
  "created_at": "<ISO-8601 UTC timestamp>"
}
```

**2. Append to `memory.decisions` array:**
```json
{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "focus",
  "content": "<the focus area>",
  "context": "Phase <current_phase> -- <colony state>",
  "phase": <current_phase>,
  "timestamp": "<ISO-8601 UTC>"
}
```
If `memory.decisions` exceeds 30 entries, remove the oldest to keep only 30.

**3. Append to `events` array as pipe-delimited string:**
```
"<ISO-8601 UTC> | pheromone_emitted | focus | FOCUS: <content> (strength 0.7, half-life 1hr)"
```
If `events` exceeds 100 entries, remove the oldest to keep only 100.

Use the Write tool to write the FULL updated state back to `.aether/data/COLONY_STATE.json`.

### Step 4: Display Result

```
FOCUS pheromone emitted

  Area: "<focus area>"
  Strength: 0.7
  Half-life: 1 hour

  Colony response by sensitivity:
    builder (0.9)     -- strong: will prioritize
    scout (0.9)       -- strong: will research first
    watcher (0.8)     -- strong: increased scrutiny
    colonizer (0.7)   -- moderate: will explore
    route-setter (0.5) -- moderate: factor into plan
    architect (0.4)   -- weak: noted for patterns

Next Steps:
  /ant:redirect "<pattern>"  Warn colony away from something
  /ant:status                View all active pheromones
  /ant:build <phase>         Start building (focus will influence workers)
```
