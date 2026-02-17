---
name: ant:feedback
description: "Emit FEEDBACK signal to provide guidance to colony"
---

You are the **Queen**. Emit a FEEDBACK pheromone signal.

## Instructions

The feedback message is: `$ARGUMENTS`

### Step 1: Validate

If `$ARGUMENTS` empty -> show usage: `/ant:feedback <message>`, stop.
If content > 500 chars -> "Signal content too long (max 500 chars)", stop.

Parse optional flags from `$ARGUMENTS`:
- `--ttl <value>`: signal lifetime (e.g., `2h`, `1d`, `7d`). Default: `phase_end`.
- Strip flags from content before using it as the feedback message.

### Step 2: Write Signal and Create Instinct

Read `.aether/data/COLONY_STATE.json`.
If `goal: null` -> "No colony initialized.", stop.

Run:
```bash
bash .aether/aether-utils.sh pheromone-write FEEDBACK "<content>" --strength 0.7 --reason "User feedback guidance" --ttl <ttl>
```

**Create instinct from feedback:**
User feedback is high-value learning. Generate ISO-8601 timestamp and append to `memory.instincts` in COLONY_STATE.json:
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

### Step 3: Get Active Counts

Run:
```bash
bash .aether/aether-utils.sh pheromone-count
```

### Step 4: Confirm

Output (4 lines, no banners):
```
FEEDBACK signal emitted
  Note: "<content truncated to 60 chars>"
  Strength: 0.7 | Expires: <phase end or ttl value>
  Active signals: <focus_count> FOCUS, <redirect_count> REDIRECT, <feedback_count> FEEDBACK

Instinct created: [0.7] <domain>: <action summary>
```
