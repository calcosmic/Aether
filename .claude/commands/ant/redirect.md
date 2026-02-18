---
name: ant:redirect
description: "Emit REDIRECT signal to warn colony away from patterns"
---

You are the **Queen**. Emit a REDIRECT pheromone signal.

## Instructions

The pattern to avoid is: `$ARGUMENTS`

### Step 1: Validate

If `$ARGUMENTS` empty -> show usage: `/ant:redirect <pattern to avoid>`, stop.
If content > 500 chars -> "Redirect content too long (max 500 chars)", stop.

Parse optional flags from `$ARGUMENTS`:
- `--ttl <value>`: signal lifetime (e.g., `2h`, `1d`, `7d`). Default: `phase_end`.
- Strip flags from content before using it as the pattern.

### Step 2: Write Signal

Read `.aether/data/COLONY_STATE.json`.
If `goal: null` -> "No colony initialized.", stop.

Run using the Bash tool with description "Setting colony redirect...":
```bash
bash .aether/aether-utils.sh pheromone-write REDIRECT "<content>" --strength 0.9 --reason "User warned colony away from pattern" --ttl <ttl>
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
REDIRECT signal emitted
  Avoid: "<content truncated to 60 chars>"
  Strength: 0.9 | Expires: <phase end or ttl value>
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
