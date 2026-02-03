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

  What would you like to build?

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

Use the Write tool to write `.aether/data/COLONY_STATE.json`:

```json
{
  "goal": "<the user's goal>",
  "state": "READY",
  "current_phase": 0,
  "session_id": "<generated session_id>",
  "initialized_at": "<ISO-8601 timestamp>",
  "workers": {
    "colonizer": "idle",
    "route-setter": "idle",
    "builder": "idle",
    "watcher": "idle",
    "scout": "idle",
    "architect": "idle"
  }
}
```

### Step 4: Emit INIT Pheromone

Use the Write tool to write `.aether/data/pheromones.json`:

```json
{
  "signals": [
    {
      "id": "init_<unix_timestamp>",
      "type": "INIT",
      "content": "<the user's goal>",
      "strength": 1.0,
      "half_life_seconds": null,
      "created_at": "<ISO-8601 timestamp>"
    }
  ]
}
```

INIT signals have no half-life — they persist forever.

### Step 5: Display Result

Output this header at the start of your response:

```
+=====================================================+
|  AETHER COLONY :: INIT                               |
+=====================================================+
```

Then show step progress:

```
  ✓ Step 1: Validate Input
  ✓ Step 2: Read Current State
  ✓ Step 3: Write Colony State
  ✓ Step 4: Emit INIT Pheromone
  ✓ Step 5: Display Result
```

Then output a divider and the result:

```
---

Aether Colony — Ready

  Session: <session_id>

  Queen's Intention:
  "<goal>"

  Colony Status: READY
  Workers: colonizer, route-setter, builder, watcher, scout, architect

Next Steps:
  /ant:plan     Generate project plan
  /ant:colonize Analyze existing codebase first (optional)
  /ant:status   View colony status
```
