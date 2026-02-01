---
name: ant:status
description: Show Queen Ant Colony status - Worker Ants, pheromones, phase progress
---

<objective>
Display comprehensive colony status including Worker Ant activity, active pheromones, phase progress, and colony health.
</objective>

<process>
You are the **Queen Ant Colony** displaying comprehensive status.

## Step 1: Load Colony State

Read from `.aether/COLONY_STATE.json`:
```python
import json
from datetime import datetime

with open('.aether/COLONY_STATE.json', 'r') as f:
    state = json.load(f)
```

## Step 2: Display Status Header

```
🐜 QUEEN ANT COLONY STATUS

{timestamp}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Step 3: Display Goal and Phase

```
GOAL: {goal}

CURRENT PHASE: Phase {id} - {name} [{status}]
  Progress: {completed_tasks}/{total_tasks} tasks ({percentage}%)
  Started: {start_time}
  Active subagents: {count}
```

## Step 4: Display Worker Ant Activity

```
WORKER ANTS:
```

For each of the 6 castes, show:
```python
worker_ants = state.get('worker_ants', {})

for caste_name, ant_state in worker_ants.items():
    status = ant_state.get('status', 'IDLE')
    current_task = ant_state.get('current_task', 'None')
    spawned_count = ant_state.get('spawned_subagents', 0)

    caste_display = caste_name.upper()

    if status == 'ACTIVE':
        print(f"  {caste_display} [ACTIVE]: {current_task}")
        if spawned_count > 0:
            print(f"    → {spawned_count} subagents active")
    else:
        print(f"  {caste_display} [{status}]: {current_task}")
```

## Step 5: Display Active Pheromones

```
ACTIVE PHEROMONES: {count}
```

```python
pheromones = state.get('pheromones', [])

for pheromone in pheromones:
    if pheromone.get('is_active', True):
        signal_type = pheromone['signal_type']
        content = pheromone['content']
        strength = pheromone.get('current_strength', pheromone['strength']) * 100

        print(f"  [{signal_type}] {content} (strength: {strength:.0f}%)")
```

## Step 6: Display Phase Progress

```
PHASE PROGRESS:
  Completed: {completed_count}
  In Progress: {in_progress_count}
  Pending: {pending_count}
  Total: {total_phases}

OVERALL PROGRESS: {overall_percentage}%
```

## Step 7: Display Colony Health

```
COLONY HEALTH:
  • Pheromone signals: {signal_count} active
  • Memory utilization: {memory_usage}%
  • Agent spawn depth: {current_depth}/{max_depth}
  • Circuit breakers: {breaker_status}
```

## Step 8: Display Recent Activity

```
RECENT ACTIVITY:
  • {timestamp} - {activity_1}
  • {timestamp} - {activity_2}
  • {timestamp} - {activity_3}
```

## Step 9: Display Available Actions

Based on current state, show relevant actions:

```
📋 AVAILABLE ACTIONS:

  1. /ant:phase             - View current phase details
  2. /ant:focus <area>      - Add focus pheromone
  3. /ant:status            - Refresh this status
  4. /ant:memory            - View learned patterns
```

If phase is IN_PROGRESS, add:
```
  5. /ant:review {id}       - Review phase progress
```

If phase is AWAITING_REVIEW, add:
```
  5. /ant:review {id}       - Review completed phase
  6. /ant:feedback "<msg>"  - Provide feedback
```

</process>

<context>
@.aether/worker_ants.py
@.aether/pheromone_system.py
@.aether/phase_engine.py

Worker Ant Castes (6):
1. Mapper - Exploration, codebase understanding
2. Planner - Goal decomposition, phase planning
3. Executor - Code implementation, spawning
4. Verifier - Testing, validation, QA
5. Researcher - Information gathering, research
6. Synthesizer - Knowledge synthesis, memory compression

Pheromone Types:
- INIT: Strong attract, triggers planning (100% strength, no decay)
- FOCUS: Medium attract, guides attention (70% strength, 1hr half-life)
- REDIRECT: Strong repel, warns away (70% strength, 24hr half-life)
- FEEDBACK: Variable, adjusts behavior (50% strength, 6hr half-life)
</context>

<reference>
# Worker Ant Status Values

- IDLE: No active task
- ACTIVE: Working on task
- SPAWNING: Creating subagent
- COORDINATING: Communicating with other ants
- BLOCKED: Waiting for dependency

# Example Full Output

```
🐜 QUEEN ANT COLONY STATUS

2025-02-01 15:30:45

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

GOAL: Build a real-time chat application

CURRENT PHASE: Phase 2 - Real-time Communication [IN_PROGRESS]
  Progress: 5/8 tasks (62%)
  Started: 2025-02-01 14:00:00
  Active subagents: 5

WORKER ANTS:
  MAPPER [IDLE]: None
  PLANNER [IDLE]: None
  EXECUTOR [ACTIVE]: Implementing message persistence
    → 3 subagents active
  VERIFIER [ACTIVE]: Testing message delivery
    → 2 subagents active
  RESEARCHER [IDLE]: None
  SYNTHESIZER [IDLE]: None

ACTIVE PHEROMONES: 3
  [INIT] Build chat app (strength: 100%)
  [FOCUS] WebSocket security (strength: 65%)
  [FOCUS] message reliability (strength: 45%)

PHASE PROGRESS:
  Completed: 1
  In Progress: 1
  Pending: 3
  Total: 5

OVERALL PROGRESS: 40%

COLONY HEALTH:
  • Pheromone signals: 3 active
  • Memory utilization: 45%
  • Agent spawn depth: 2/3
  • Circuit breakers: None active

RECENT ACTIVITY:
  • 15:28 - Executor completed message queue implementation
  • 15:25 - Verifier found issue with message ordering
  • 15:20 - Mapper completed WebSocket layer analysis

📋 AVAILABLE ACTIONS:

  1. /ant:phase             - View current phase details
  2. /ant:focus <area>      - Add focus pheromone
  3. /ant:status            - Refresh this status
  4. /ant:memory            - View learned patterns
  5. /ant:review 2          - Review phase progress
```
</reference>

<allowed-tools>
Read
Write
Bash
Glob
Grep
</allowed-tools>
