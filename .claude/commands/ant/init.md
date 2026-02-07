---
name: ant:init
description: Initialize Aether colony - Queen sets intention, colony mobilizes
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

### Step 3: Write Colony State

Generate a session ID in the format `session_{unix_timestamp}_{random}` and an ISO-8601 UTC timestamp.

Use the Write tool to write `.aether/data/COLONY_STATE.json` with the v3.0 structure:

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
    "phase_learnings": [],
    "decisions": []
  },
  "errors": {
    "records": [],
    "flagged_patterns": []
  },
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
bash ~/.aether/aether-utils.sh validate-state colony
```

This validates COLONY_STATE.json structure. If validation fails, output a warning.

### Step 6: Display Result

Output this header:

```
+=====================================================+
|  AETHER COLONY :: INIT                               |
+=====================================================+
```

Then show step progress:

```
  Step 1: Validate Input
  Step 2: Read Current State
  Step 3: Write Colony State
  Step 4: Initialize Constraints
  Step 5: Validate State File
  Step 6: Display Result
```

Then output the result:

```
---

Aether Colony -- Ready

  Session: <session_id>

  Queen's Intention:
  "<goal>"

  Colony Status: READY

Next Steps:
  /ant:plan     Generate project plan (iterative research loop)
  /ant:colonize Analyze existing codebase first (optional)
  /ant:watch    Set up tmux for live visibility
  /ant:status   View colony status
```
