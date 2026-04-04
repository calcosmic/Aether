<!-- Generated from .aether/commands/redirect.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:redirect
description: "⚠️🐜🚧🐜⚠️ Emit REDIRECT signal to warn colony away from patterns"
---

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.



You are the **Queen**. Add an AVOID constraint.


## Instructions

The pattern to avoid is: `$normalized_args`

### Step 1: Validate

If `$normalized_args` empty -> show usage: `/ant:redirect <pattern to avoid>`, stop.
If content > 500 chars -> "Redirect content too long (max 500 chars)", stop.



### Step 2: Write Signal

Read `.aether/data/COLONY_STATE.json`.
If `goal: null` -> "No colony initialized.", stop.



Read `.aether/data/constraints.json`. If file doesn't exist, create it with:
```json
{"version": "1.0", "focus": [], "constraints": []}
```

Generate constraint ID: `c_<unix_timestamp_ms>`

Append to `constraints` array:
```json
{
  "id": "<generated_id>",
  "type": "AVOID",
  "content": "<pattern to avoid>",
  "source": "user:redirect",
  "created_at": "<ISO-8601 timestamp>"
}
```

If `constraints` array exceeds 10 entries, remove the oldest entries to keep only 10.

Write constraints.json.

**Write pheromone signal and update context:**
```bash
aether pheromone-write --type REDIRECT --content "$normalized_args" --strength 0.9 --reason "User warned colony away from pattern" 2>/dev/null || true
aether context-update --section constraint --key redirect --content "$normalized_args" "user" 2>/dev/null || true
```

### Step 3: Confirm

Output header:

```
⚠️🐜🚧🐜⚠️ ═══════════════════════════════════════════════════
   R E D I R E C T   S I G N A L
═══════════════════════════════════════════════════ ⚠️🐜🚧🐜⚠️
```

Then output:
```
🚫 REDIRECT signal emitted

   Avoid: "{content preview}"

🐜 Colony warned away from this pattern.
```



