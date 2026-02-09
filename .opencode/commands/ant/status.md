---
name: ant:status
description: "📈🐜🏘️🐜📈 Show colony status at a glance"
---

You are the **Queen**. Show colony status.

## Instructions

### Step 1: Read State + Version Check

Read `.aether/data/COLONY_STATE.json`.

If file missing or `goal: null`:
```
No colony initialized. Run /ant:init first.
```
Stop here.

**Auto-upgrade old state:**
If `version` field is missing, "1.0", or "2.0":
1. Preserve: `goal`, `state`, `current_phase`, `plan.phases` (keep phase structure)
2. Write upgraded state:
```json
{
  "version": "3.0",
  "goal": "<preserved>",
  "state": "<preserved or 'READY'>",
  "current_phase": <preserved or 0>,
  "session_id": "migrated_<timestamp>",
  "initialized_at": "<preserved or now>",
  "build_started_at": null,
  "plan": {
    "generated_at": "<preserved or null>",
    "confidence": null,
    "phases": <preserved or []>
  },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": ["<now>|state_upgraded|system|Auto-upgraded from v<old> to v3.0"]
}
```
3. Output: `State auto-upgraded to v3.0`
4. Continue with command.

### Step 2: Compute Summary

From state, extract:

**Phase info:**
- Current phase number: `current_phase`
- Total phases: `plan.phases.length`
- Phase name: `plan.phases[current_phase - 1].name` (if exists)

**Task progress:**
- If phases exist, count tasks in current phase
- Completed: tasks with `status: "completed"`
- Total: all tasks in current phase

**Constraints:**
Read `.aether/data/constraints.json` if exists:
- Focus count: `focus.length`
- Constraints count: `constraints.length`

**Flags:**
Run: `bash ~/.aether/aether-utils.sh flag-check-blockers`
Extract:
- Blockers count (critical, block advancement)
- Issues count (high, warnings)
- Notes count (low, informational)

**Instincts:**
From `memory.instincts`:
- Total count: `instincts.length`
- High confidence (>=0.7): count where confidence >= 0.7
- Top 3: sorted by confidence descending

**Colony state:**
- `state` field (IDLE, READY, EXECUTING, PLANNING)

### Step 3: Display

Output format:

```
       .-.
      (o o)  AETHER COLONY
      | O |  Status Report
       `-`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Goal: <goal (truncated to 60 chars)>

Phase <N>/<M>: <phase name>
   Tasks: <completed>/<total> complete

Focus: <focus_count> areas | Avoid: <constraints_count> patterns
Instincts: <total> learned (<high_confidence> strong)
Flags: <blockers> blockers | <issues> issues | <notes> notes

State: <state>
Next:  /ant:<suggested command>
```

**If instincts exist, also show top 3:**
```
Colony Instincts:
   [0.9] testing: Always run tests before completion
   [0.8] architecture: Use composition over inheritance
   [0.7] debugging: Trace to root cause first
```

**Suggested command logic:**
- IDLE -> init
- READY -> build <next_phase>
- EXECUTING -> continue (wait for build)
- PLANNING -> plan (wait for completion)

**Edge cases:**
- No phases yet: "Phase 0/0: No plan created"
- No tasks in phase: "Tasks: 0/0 complete"
- No constraints file: "Constraints: 0 focus, 0 avoid"
