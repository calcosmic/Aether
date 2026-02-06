---
name: ant:redirect
description: Emit REDIRECT pheromone - Queen warns colony away from specific approaches
---

You are the **Queen Ant Colony**. Emit a REDIRECT pheromone to warn the colony away from a pattern.

## Instructions

The pattern to avoid is: `$ARGUMENTS`

### Step 1: Validate Input

If `$ARGUMENTS` is empty or blank, output:

```
Usage: /ant:redirect "<pattern to avoid>"

Examples:
  /ant:redirect "Don't use JWT for sessions"
  /ant:redirect "Avoid synchronous I/O"
  /ant:redirect "No global mutable state"
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
  "id": "redirect_<unix_timestamp>",
  "type": "REDIRECT",
  "content": "<the pattern to avoid>",
  "strength": 0.9,
  "half_life_seconds": 86400,
  "created_at": "<ISO-8601 UTC timestamp>"
}
```

**2. Append to `memory.decisions` array:**
```json
{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "redirect",
  "content": "<the pattern to avoid>",
  "context": "Phase <current_phase> -- <colony state>",
  "phase": <current_phase>,
  "timestamp": "<ISO-8601 UTC>"
}
```
If `memory.decisions` exceeds 30 entries, remove the oldest to keep only 30.

**3. Append to `events` array as pipe-delimited string:**
```
"<ISO-8601 UTC> | pheromone_emitted | redirect | REDIRECT: <content> (strength 0.9, half-life 24hr)"
```
If `events` exceeds 100 entries, remove the oldest to keep only 100.

Use the Write tool to write the FULL updated state back to `.aether/data/COLONY_STATE.json`.

### Step 4: Display Result

```
REDIRECT pheromone emitted

  Avoid: "<pattern>"
  Strength: 0.9
  Half-life: 24 hours

  Colony response by sensitivity:
    builder (0.9)      -- strong: will avoid this pattern in code
    route-setter (0.8) -- strong: will exclude from planning
    watcher (0.5)      -- moderate: will validate against constraint
    scout (0.4)        -- weak: will note when researching
    colonizer (0.3)    -- weak: will note in codebase analysis
    architect (0.3)    -- weak: will note for patterns

  REDIRECT signals act as hard constraints during /ant:build.
  Workers with high sensitivity will refuse approaches matching this pattern.

Next Steps:
  /ant:focus "<area>"   Guide attention toward something
  /ant:status           View all active pheromones
```
