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

  Initialize the colony with a goal. This creates the consolidated colony state,
  resets all workers to idle, emits an INIT pheromone, and logs the init event.

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

Use the Write tool to write `.aether/data/COLONY_STATE.json` with the complete v2.0 structure:

```json
{
  "version": "2.0",
  "goal": "<the user's goal>",
  "state": "READY",
  "current_phase": 0,
  "session_id": "<generated session_id>",
  "initialized_at": "<ISO-8601 timestamp>",
  "mode": null,
  "mode_set_at": null,
  "mode_indicators": null,
  "workers": {
    "colonizer": "idle",
    "route-setter": "idle",
    "builder": "idle",
    "watcher": "idle",
    "scout": "idle",
    "architect": "idle"
  },
  "spawn_outcomes": {
    "colonizer":    {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "route-setter": {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "builder":      {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "watcher":      {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "scout":        {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "architect":    {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0}
  },
  "plan": {
    "generated_at": null,
    "phases": []
  },
  "signals": [
    {
      "id": "init_<unix_timestamp>",
      "type": "INIT",
      "content": "<the user's goal>",
      "priority": "high",
      "expires_at": null,
      "created_at": "<ISO-8601 timestamp>"
    }
  ],
  "memory": {
    "phase_learnings": [],
    "decisions": [],
    "patterns": []
  },
  "errors": {
    "records": [],
    "flagged_patterns": []
  },
  "events": [
    "<ISO-8601 timestamp> | colony_initialized | init | Colony initialized with goal: <the user's goal>"
  ]
}
```

INIT signals have no expiration (expires_at: null) and persist forever.

### Step 4: Validate State File

Use the Bash tool to run:
```
bash ~/.aether/aether-utils.sh validate-state colony
```

This validates COLONY_STATE.json structure and returns `{"ok":true,"result":{"pass":true|false}}`.

If `pass` is false, output a warning identifying which validation(s) failed. This catches initialization bugs immediately.

### Step 5: Display Result

Output this header at the start of your response:

```
+=====================================================+
|  👑 AETHER COLONY :: INIT                            |
+=====================================================+
```

Then show step progress:

```
  ✓ Step 1: Validate Input
  ✓ Step 2: Read Current State
  ✓ Step 3: Write Colony State
  ✓ Step 4: Validate State File
  ✓ Step 5: Display Result
```

Then output a divider and the result:

```
---

👑 Aether Colony — Ready

  Session: <session_id>

  Queen's Intention:
  "<goal>"

  Colony Status: READY
  Workers:
    🗺️ colonizer  📋 route-setter  🔨 builder
    👁️ watcher    🔍 scout         🏛️ architect

Next Steps:
  /ant:plan     Generate project plan
  /ant:colonize Analyze existing codebase first (optional)
  /ant:status   View colony status
```
