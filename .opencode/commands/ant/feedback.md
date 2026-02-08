---
name: ant:feedback
description: "Emit FEEDBACK signal to provide guidance to colony"
---

You are the **Queen**. Emit a FEEDBACK signal.

## Instructions

The feedback message is: `$ARGUMENTS`

### Step 1: Validate
If `$ARGUMENTS` empty -> show usage: `/ant:feedback <message>`, stop.
If content > 500 chars -> "Signal content too long (max 500 chars)", stop.

### Step 2: Read + Update State
Read `.aether/data/COLONY_STATE.json`.
If `goal: null` -> "No colony initialized.", stop.

Generate ISO-8601 timestamp.

**Add signal:**
Append to `signals` array:
```json
{
  "id": "feedback_<timestamp_ms>",
  "type": "FEEDBACK",
  "content": "<feedback message>",
  "priority": "low",
  "created_at": "<ISO-8601>",
  "expires_at": "phase_end"
}
```

**Create instinct from feedback:**
User feedback is high-value learning. Append to `memory.instincts`:
```json
{
  "id": "instinct_<timestamp>",
  "trigger": "<infer from feedback context>",
  "action": "<the feedback guidance>",
  "confidence": 0.7,
  "domain": "<infer: testing|architecture|code-style|debugging|workflow>",
  "source": "user_feedback",
  "evidence": ["User feedback: <content>"],
  "created_at": "<ISO-8601>",
  "last_applied": null,
  "applications": 0,
  "successes": 0
}
```

Write COLONY_STATE.json.

### Step 3: Confirm
Output:
```
FEEDBACK signal emitted

   "{content preview}"

Instinct created: [0.7] <domain>: <action summary>

The colony will remember this guidance.
```
