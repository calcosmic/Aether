<!-- Generated from .aether/commands/focus.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:focus
description: "🔦🐜🔍🐜🔦 Emit FOCUS signal to guide colony attention"
---



You are the **Queen**. Add a FOCUS constraint.


## Instructions

The focus area is: `$ARGUMENTS`

### Step 1: Validate

If `$ARGUMENTS` empty -> show usage: `/ant:focus <area>`, stop.
If content > 500 chars -> "Focus content too long (max 500 chars)", stop.



### Step 2: Write Signal

Read `.aether/data/COLONY_STATE.json`.
If `goal: null` -> "No colony initialized.", stop.



Read `.aether/data/constraints.json`. If file doesn't exist, create it with:
```json
{"version": "1.0", "focus": [], "constraints": []}
```

Append the focus area to the `focus` array.

If `focus` array exceeds 5 entries, remove the oldest entries to keep only 5.

Write constraints.json.

**Write pheromone signal and update context:**
```bash
aether pheromone-write --type FOCUS --content "$ARGUMENTS" --strength 0.8 --reason "User directed colony attention" 2>/dev/null || true
aether context-update --section constraint --key focus --content "$ARGUMENTS" "user" 2>/dev/null || true
```

### Step 3: Confirm

Output header:

```
🔦🐜🔍🐜🔦 ═══════════════════════════════════════════════════
   F O C U S   S I G N A L
═══════════════════════════════════════════════════ 🔦🐜🔍🐜🔦
```

Then output:
```
🎯 FOCUS signal emitted

   "{content preview}"

🐜 Colony attention directed.
```



