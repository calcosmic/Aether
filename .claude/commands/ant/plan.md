---
name: ant:plan
description: Show all phases with tasks, milestones, and status
---

<objective>
Display the complete phase plan showing all phases, their tasks, milestones, and current status.
</objective>

<process>
You are the **Queen Ant Colony** displaying the colony's plan.

## Step 1: Check for Initialized Project

Check if `.aether/COLONY_STATE.json` exists. If not:
```
❌ No project initialized. Run /ant:init "<goal>" first.
```

## Step 2: Load Colony State

Read the colony state from `.aether/COLONY_STATE.json`:
```python
import json

with open('.aether/COLONY_STATE.json', 'r') as f:
    state = json.load(f)

phases = state.get('phases', [])
current_phase_id = state.get('current_phase_id')
goal = state.get('goal')
```

## Step 3: Display Phase Plan

Format and display all phases:

```
🐜 QUEEN ANT COLONY - PHASE PLAN

GOAL: {goal}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PHASE {id}: {name} [{status}]
  Description: {description}
  Tasks: {count}
  {List each task with status indicator}

  Milestones:
    • {milestone_1}
    • {milestone_2}
```

Task Status Indicators:
- ⏳ Pending
- 🔄 In Progress
- ✅ Complete
- ❌ Failed

Phase Status:
- PENDING
- PLANNING
- IN_PROGRESS
- AWAITING_REVIEW
- APPROVED
- COMPLETED
- FAILED

## Step 4: Show Current Phase

Highlight the current phase with `[← CURRENT]` marker.

## Step 5: Display Next Steps

```
📋 NEXT STEPS:

  1. /ant:phase {id}           - Review Phase {id} details
  2. /ant:execute {id}         - Start executing Phase {id}
  3. /ant:focus <area>         - Add focus guidance (optional)

💡 RECOMMENDATION: Review current phase with /ant:phase {id} before executing

🔄 CONTEXT: This command is lightweight - safe to continue
```

</process>

<context>
@.aether/phase_engine.py
@.aether/worker_ants.py

Phase Structure:
- Phases created by Planner Ant during /ant:init
- Each phase has tasks, milestones, status
- Status tracks through lifecycle: PENDING → IN_PROGRESS → COMPLETED
- State persisted in .aether/COLONY_STATE.json
</context>

<reference>
# Phase Status Flow

```
PENDING → PLANNING → IN_PROGRESS → AWAITING_REVIEW → APPROVED → COMPLETED
                                      ↓
                                   FAILED
```

## Example Output

```
PHASE 1: Foundation [PENDING]
  Description: Setup project structure, configure development environment
  Tasks: 5
  ⏳ Setup project structure
  ⏳ Configure development environment
  ⏳ Initialize database schema
  ⏳ Setup WebSocket server
  ⏳ Implement basic message routing

  Milestones:
    • WebSocket server running
    • Database connected

PHASE 2: Real-time Communication [PENDING]
  Description: Implement WebSocket connection handling and message queue
  Tasks: 8
  ⏳ Implement WebSocket connection handling
  ⏳ Create message queue system
  ...
```
</reference>

<allowed-tools>
Read
Write
Bash
Glob
Grep
</allowed-tools>
