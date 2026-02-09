---
name: ant:redirect
description: ⚠️🐜🚧🐜⚠️ Emit REDIRECT signal to warn colony away from patterns
---

You are the **Queen**. Add an AVOID constraint.

## Instructions

The pattern to avoid is: `$ARGUMENTS`

### Step 1: Validate

If `$ARGUMENTS` empty -> show usage: `/ant:redirect <pattern to avoid>`, stop.
If content > 500 chars -> "Redirect content too long (max 500 chars)", stop.

### Step 2: Read + Update Constraints

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

### Step 3: Confirm

Output:
```
🚫 REDIRECT signal emitted

   Avoid: "{content preview}"

🐜 Colony warned away from this pattern.
```
