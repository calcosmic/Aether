---
name: ant:init
description: "🌱🐜🆕🐜🌱 Initialize Aether colony - Queen sets intention, colony mobilizes"
---

You are the **Queen Ant Colony**. Initialize the colony with the Queen's intention.

## Instructions

The user's goal is: `$ARGUMENTS`

### Step 1: Validate Input

If `$ARGUMENTS` is empty or blank, output:

```
Aether Colony

  Initialize the colony with a goal. This creates the colony state,
  initializes constraints, and logs the init event.

  Usage: /ant:init "<your goal here>"

  Examples:
    /ant:init "Build a REST API with authentication"
    /ant:init "Create a soothing sound application"
    /ant:init "Design a calculator CLI tool"
```

Stop here. Do not proceed.

### Step 2: Read Current State

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

If the `goal` field is not null, output:

```
Colony already initialized with goal: "{existing_goal}"

To reinitialize with a new goal, the current state will be reset.
Proceeding with new goal: "{new_goal}"
```

### Step 2.5: Load Prior Colony Knowledge (Optional)

Check if `.aether/data/completion-report.md` exists using the Read tool.

**If the file does NOT exist**, skip to Step 3 — this is a fresh colony with no prior history.

**If the file exists**, read it and extract:
1. **Instincts** — look for the `## Colony Instincts` section. Each line has format: `N. [confidence] domain: description`. Keep only instincts with confidence >= 0.5.
2. **Learnings** — look for the `## Colony Learnings (Validated)` section. Keep all numbered items.

Store the extracted instincts and learnings for use in Step 3. Display a brief note:

```
🧠 Prior colony knowledge found:
   {N} instinct(s) inherited (confidence >= 0.5)
   {N} validated learning(s) carried forward
```

If no instincts meet the threshold, display:
```
🧠 Prior colony knowledge found but no high-confidence instincts to inherit.
```

**Important:** This step is read-only and non-blocking. If the file is malformed or unreadable, skip silently and proceed to Step 3 with empty memory.

### Step 3: Write Colony State

Generate a session ID in the format `session_{unix_timestamp}_{random}` and an ISO-8601 UTC timestamp.

Use the Write tool to write `.aether/data/COLONY_STATE.json` with the v3.0 structure.

**If Step 2.5 found instincts to inherit**, convert each into the instinct format and seed the `memory.instincts` array. Each inherited instinct should have:
- `id`: `instinct_inherited_{index}`
- `trigger`: inferred from the instinct description
- `action`: the instinct description
- `confidence`: the original confidence value (from the completion report)
- `domain`: the original domain (from the completion report)
- `source`: `"inherited:completion-report"`
- `evidence`: `["Validated in prior colony session"]`
- `created_at`: current ISO-8601 timestamp
- `last_applied`: null
- `applications`: 0
- `successes`: 0

**If Step 2.5 found validated learnings**, seed `memory.phase_learnings` with each as:
- `phase`: `"inherited"`
- `learning`: the learning text
- `status`: `"validated"`
- `source`: `"inherited:completion-report"`

**If Step 2.5 was skipped or found nothing**, use empty arrays as before.

```json
{
  "version": "3.0",
  "goal": "<the user's goal>",
  "state": "READY",
  "current_phase": 0,
  "session_id": "<generated session_id>",
  "initialized_at": "<ISO-8601 timestamp>",
  "build_started_at": null,
  "plan": {
    "generated_at": null,
    "confidence": null,
    "phases": []
  },
  "memory": {
    "phase_learnings": "<inherited learnings or []>",
    "decisions": [],
    "instincts": "<inherited instincts or []>"
  },
  "errors": {
    "records": [],
    "flagged_patterns": []
  },
  "signals": [],
  "graveyards": [],
  "events": [
    "<ISO-8601 timestamp>|colony_initialized|init|Colony initialized with goal: <the user's goal>"
  ]
}
```

### Step 4: Initialize Constraints

Write `.aether/data/constraints.json`:

```json
{
  "version": "1.0",
  "focus": [],
  "constraints": []
}
```

### Step 5: Validate State File

Use the Bash tool to run:
```
bash .aether/aether-utils.sh validate-state colony
```

This validates COLONY_STATE.json structure. If validation fails, output a warning.

### Step 6: Display Result

Output this header:

```
🌱🐜🆕🐜🌱 ═══════════════════════════════════════════════════
   A E T H E R   C O L O N Y
═══════════════════════════════════════════════════ 🌱🐜🆕🐜🌱
```

Then output the result:

```
👑 Queen has set the colony's intention

   "{goal}"

🏠 Colony Status: READY
📋 Session: <session_id>

{If instincts or learnings were inherited from Step 2.5:}
🧠 Inherited from prior colony:
   {N} instinct(s) | {N} learning(s)
{End if}

🐜 The colony awaits your command:

   /ant:plan      📋 Generate project plan
   /ant:colonize  🗺️  Analyze existing codebase first
   /ant:watch     👁️  Set up live visibility

💾 State persisted — safe to /clear, then run /ant:plan
```
