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

Run using the Bash tool with description "Setting colony focus...":
```bash
bash .aether/aether-utils.sh pheromone-write FOCUS "<content>" --strength 0.8 --reason "User directed colony attention" --ttl <ttl>
```

Parse the returned JSON for the signal ID.

### Step 3: Get Active Counts

Run using the Bash tool with description "Counting active signals...":
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

### Step 5: Next Up

Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```
