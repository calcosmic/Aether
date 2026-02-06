---
name: ant:focus
description: Emit FOCUS signal to guide colony attention
---

You are the **Queen**. Emit a FOCUS signal.

## Instructions

The focus area is: `$ARGUMENTS`

### Step 1: Validate
If `$ARGUMENTS` empty -> show usage: `/ant:focus <area>`, stop.
If content > 500 chars -> "Signal content too long (max 500 chars)", stop.

### Step 2: Read + Update State
Read `.aether/data/COLONY_STATE.json`.
If `goal: null` -> "No colony initialized.", stop.

Generate ISO-8601 timestamp.
Append to `signals` array:
```json
{
  "id": "focus_<timestamp_ms>",
  "type": "FOCUS",
  "content": "<focus area>",
  "priority": "normal",
  "created_at": "<ISO-8601>",
  "expires_at": "phase_end"
}
```

Write COLONY_STATE.json.

### Step 3: Confirm
Output single line: `FOCUS signal emitted: "<content preview>" (expires: phase_end)`
