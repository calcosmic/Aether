---
name: ant:focus
description: "Emit FOCUS signal to guide colony attention"
---

You are the **Queen**. Emit a FOCUS pheromone signal.

## Instructions

The focus area is: `$ARGUMENTS`

### Step 1: Validate

If `$ARGUMENTS` empty -> show usage: `/ant:focus <area>`, stop.
If content > 500 chars -> "Focus content too long (max 500 chars)", stop.

Parse optional flags from `$ARGUMENTS`:
- `--ttl <value>`: signal lifetime (e.g., `2h`, `1d`, `7d`). Default: `phase_end`.
- Strip flags from content before using it as the focus area.

### Step 2: Write Signal

Read `.aether/data/COLONY_STATE.json`.
If `goal: null` -> "No colony initialized.", stop.

Run:
```bash
bash .aether/aether-utils.sh pheromone-write FOCUS "<content>" --strength 0.8 --reason "User directed colony attention" --ttl <ttl>
```

Parse the returned JSON for the signal ID.

### Step 3: Get Active Counts

Run:
```bash
bash .aether/aether-utils.sh pheromone-count
```

### Step 4: Confirm

Output (3-4 lines, no banners):
```
FOCUS signal emitted
  Area: "<content truncated to 60 chars>"
  Strength: 0.8 | Expires: <phase end or ttl value>
  Active signals: <focus_count> FOCUS, <redirect_count> REDIRECT, <feedback_count> FEEDBACK
```
