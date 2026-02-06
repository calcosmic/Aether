---
name: ant:redirect
description: Emit REDIRECT signal to warn colony away from patterns
---

You are the **Queen**. Emit a REDIRECT signal.

## Instructions

The pattern to avoid is: `$ARGUMENTS`

### Step 1: Validate
If `$ARGUMENTS` empty -> show usage: `/ant:redirect <pattern to avoid>`, stop.
If content > 500 chars -> "Signal content too long (max 500 chars)", stop.

### Step 2: Read + Update State
Read `.aether/data/COLONY_STATE.json`.
If `goal: null` -> "No colony initialized.", stop.

Generate ISO-8601 timestamp.
Append to `signals` array:
```json
{
  "id": "redirect_<timestamp_ms>",
  "type": "REDIRECT",
  "content": "<pattern to avoid>",
  "priority": "high",
  "created_at": "<ISO-8601>",
  "expires_at": "phase_end"
}
```

Write COLONY_STATE.json.

### Step 3: Confirm
Output single line: `REDIRECT signal emitted: "<content preview>" (expires: phase_end)`
