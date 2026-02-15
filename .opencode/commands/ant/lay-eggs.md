---
name: ant:lay-eggs
description: "🥚🐜🥚 Lay first eggs of new colony (First Eggs milestone)"
---

You are the **Queen**. Begin a new colony, preserving pheromones.

## Instructions

Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:
```bash
# Generate session ID
layeggs_id="layeggs-$(date +%s)"

# Initialize swarm display
bash .aether/aether-utils.sh swarm-display-init "$layeggs_id"
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Laying first eggs" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "nursery" 0
```

### Step 1: Validate Input

- If `$ARGUMENTS` is empty:
  ```
  Usage: /ant:lay-eggs "<new colony goal>"

  Start a fresh colony, preserving pheromones from prior colonies.
  Requires current colony to be entombed or reset.

  Example:
    /ant:lay-eggs "Build a REST API with authentication"
  ```
  Stop here.

### Step 2: Check Current Colony

- Read `.aether/data/COLONY_STATE.json`
- If goal is not null AND phases exist with status != "completed":
  ```
  Active colony exists: {goal}

  To start a new colony, you must first:
  1. Complete all phases, then /ant:entomb to archive
  2. Or manually reset by deleting .aether/data/COLONY_STATE.json

  Current: Phase {current_phase}, {phases_count} phases in plan
  ```
  Stop here.

### Step 3: Extract Preserved Knowledge

- Read current state to extract preserved fields:
  - `memory.phase_learnings` (all items)
  - `memory.decisions` (all items)
  - `memory.instincts` (all items with confidence >= 0.5)
- Store for use in Step 4

### Step 4: Create New Colony State

Generate new state following RESEARCH.md Pattern 2 (State Reset with Pheromone Preservation):

**Fields to preserve from old state:**
- memory.phase_learnings
- memory.decisions
- memory.instincts (high confidence only)

**Fields to reset:**
- goal: new goal from $ARGUMENTS
- state: "READY"
- current_phase: 0
- session_id: new session_{unix_timestamp}_{random}
- initialized_at: current ISO-8601 timestamp
- build_started_at: null
- plan: { generated_at: null, confidence: null, phases: [] }
- errors: { records: [], flagged_patterns: [] }
- signals: []
- graveyards: []
- events: [colony_initialized event with new goal]

**New milestone fields:**
- milestone: "First Mound"
- milestone_updated_at: current timestamp
- milestone_version: "v0.1.0"

Write to `.aether/data/COLONY_STATE.json`

### Step 5: Reset Constraints

Write `.aether/data/constraints.json`:
```json
{
  "version": "1.0",
  "focus": [],
  "constraints": []
}
```

### Step 6: Display Result

**If visual_mode is true, render final swarm display:**
```bash
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "First eggs laid" "Colony" '{"read":3,"grep":0,"edit":2,"bash":1}' 100 "nursery" 100
bash .aether/aether-utils.sh swarm-display-render "$layeggs_id"
```

```
🥚 ═══════════════════════════════════════════════════
   F I R S T   E G G S   L A I D
══════════════════════════════════════════════════ 🥚

👑 New colony goal:
   "{goal}"

📋 Session: {session_id}
🏆 Milestone: First Mound (v0.1.0)

{If inherited knowledge:}
🧠 Inherited from prior colonies:
   {N} instinct(s) | {N} decision(s) | {N} learning(s)
{End if}

🐜 The colony begins anew.

   /ant:plan      📋 Chart the course
   /ant:colonize  🗺️  Analyze existing code
```

Include edge case handling:
- If no prior knowledge: omit the inheritance section
- If prior colony had no phases: allow laying eggs without entombment
