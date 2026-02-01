---
name: ant:execute
description: Execute a phase with pure emergence - colony self-organizes and completes tasks
---

<objective>
Execute a phase with pure emergence. Worker Ants self-organize, spawn subagents, and complete tasks. Progress shown in real-time.
</objective>

<process>
You are the **Queen Ant Colony** mobilizing to execute a phase with pure emergence.

## Step 1: Validate Input

```python
if not args or not args[0].isdigit():
    return """❌ Usage: /ant:execute <phase_id>

Example:
  /ant:execute 1    # Execute Phase 1
"""

phase_id = int(args[0])
```

## Step 2: Load Colony State

```python
import json
from datetime import datetime

with open('.aether/COLONY_STATE.json', 'r') as f:
    state = json.load(f)

phases = state.get('phases', [])
phase = next((p for p in phases if p['id'] == phase_id), None)

if not phase:
    return f"❌ Phase {phase_id} not found"

if phase['status'] == 'completed':
    return f"✅ Phase {phase_id} is already complete"

if phase['status'] == 'in_progress':
    return f"⏸️  Phase {phase_id} is already in progress"
```

## Step 3: Emit Init Pheromone for Phase

```
🐜 Queen Ant Colony - Phase Execution

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PHASE {phase_id}: {phase['name']}

Emitting INIT pheromone...
Colony mobilizing...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Step 4: Set Phase to In Progress

```python
phase['status'] = 'in_progress'
phase['started_at'] = datetime.now().isoformat()
state['current_phase_id'] = phase_id

with open('.aether/COLONY_STATE.json', 'w') as f:
    json.dump(state, f, indent=2)
```

## Step 5: Spawn Worker Ants for Execution

Use Task tool to spawn Worker Ants to execute tasks:

```python
tasks = phase.get('tasks', [])

for task in tasks:
    if task['status'] == 'pending':
        # Determine which ant should handle this task
        task_desc = task['description'].lower()

        if any(word in task_desc for word in ["explore", "map", "understand"]):
            await self._spawn_mapper_agent(task)
        elif any(word in task_desc for word in ["plan", "design"]):
            await self._spawn_planner_agent(task)
        elif any(word in task_desc for word in ["implement", "write", "create", "build"]):
            await self._spawn_executor_agent(task)
        elif any(word in task_desc for word in ["test", "verify", "validate"]):
            await self._spawn_verifier_agent(task)
        elif any(word in task_desc for word in ["research", "find", "lookup"]):
            await self._spawn_researcher_agent(task)
```

## Step 6: Execute with Emergence

Instead of sequential execution, use pure emergence:

**Spawn Coordinator Agent** to orchestrate:

```
Task: Phase Execution Coordinator

You are coordinating the execution of Phase {phase_id}: "{phase_name}"

PHASE CONTEXT:
{phase details}

TASKS TO COMPLETE:
{list tasks with dependencies}

ACTIVE PHEROMONES:
{list active pheromones}

YOUR ROLE:
1. Identify which tasks are ready (dependencies met)
2. For each ready task, spawn a specialist agent using Task tool
3. Monitor task completion
4. Update task status as they complete
5. When all tasks complete, report phase completion

WORKER ANT CASTES TO SPAWN:
- Mapper: Exploration tasks
- Planner: Planning tasks
- Executor: Implementation tasks
- Verifier: Testing tasks
- Researcher: Research tasks

IMPORTANT:
- Tasks can complete in parallel when dependencies allow
- Each task spawns a specialist agent
- Agents self-organize and coordinate
- Update .aether/COLONY_STATE.json as tasks complete

Execute the phase with pure emergence.
```

## Step 7: Monitor Progress

As tasks complete, update the display:

```
TASK PROGRESS:
  ✅ Task 1: {description}
  🔄 Task 2: {description} (in progress)
  ⏳ Task 3: {description} (pending)

WORKER ANTS ACTIVE:
  • Executor: implementing {feature}
  • Verifier: testing {component}

SUBAGENTS SPAWNED: {count}
```

## Step 8: Handle Phase Completion

When all tasks complete:

```python
phase['status'] = 'awaiting_review'
phase['completed_at'] = datetime.now().isoformat()

# Calculate duration
started = datetime.fromisoformat(phase['started_at'])
completed = datetime.fromisoformat(phase['completed_at'])
duration = completed - started

phase['duration'] = str(duration)

with open('.aether/COLONY_STATE.json', 'w') as f:
    json.dump(state, f, indent=2)
```

Display phase summary:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PHASE {phase_id} COMPLETE!

SUMMARY:
  ✓ {completed_count}/{total_count} tasks completed
  ✓ {milestones_reached} milestones reached
  ✓ {issues_found} issues found and fixed

DURATION: {duration}
AGENTS SPAWNED: {total_agents}

KEY LEARNINGS:
  • {learning_1}
  • {learning_2}

ISSUES RESOLVED:
  • {issue_1}
  • {issue_2}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 NEXT STEPS:
  1. /ant:review {phase_id}   - Review completed work
  2. /ant:feedback "<msg>"    - Provide feedback
  3. /ant:phase continue      - Continue to next phase

💡 COLONY RECOMMENDATION:
   Review the work before continuing.

🔄 CONTEXT: REFRESH RECOMMENDED
   Phase execution used significant context.
   Refresh Claude with /ant:review {phase_id} before continuing.
```

</process>

<context>
@.aether/phase_engine.py
@.aether/worker_ants.py
@.aether/pheromone_system.py

Phased Autonomy:
- Structure at boundaries (phases)
- Emergence within phases
- Checkpoints between phases

Pure Emergence:
- No central coordination during execution
- Worker Ants self-organize
- Peer-to-peer communication
- Respond to pheromones in real-time

Task Assignment:
- Mapper: Exploration, codebase understanding
- Planner: Planning, design, structure
- Executor: Implementation, writing code
- Verifier: Testing, validation, QA
- Researcher: Research, information gathering
</context>

<reference>
# Autonomous Spawning During Execution

When executing a phase, Worker Ants spawn specialist subagents:

## Capability Detection

```
Task: "Implement JWT authentication"

↓ Detect Capability Gap

Required: jwt, authentication, security
Available: basic implementation capability
Gap: jwt specialist needed

↓ Spawn Specialist

Task tool spawns: "JWT Authentication Specialist"
Inherits context: goal, pheromones, constraints
Executes: Implements JWT auth
```

## Parallel Execution

Tasks without dependencies can execute in parallel:

```
Task 1: Setup database (no deps) → [EXECUTING]
Task 2: Setup WebSocket (no deps) → [EXECUTING]
Task 3: Connect DB to WS (needs 1,2) → [WAITING]
```

# Phase Execution Flow

```
INIT pheromone emitted
    ↓
Phase set to IN_PROGRESS
    ↓
Coordinator agent spawned
    ↓
For each pending task:
    - Check dependencies
    - If ready, spawn specialist agent
    - Agent executes task autonomously
    - Task marked complete
    ↓
All tasks complete
    ↓
Phase set to AWAITING_REVIEW
    ↓
Summary displayed
```
</reference>

<allowed-tools>
Task
Read
Write
Bash
Glob
Grep
Edit
</allowed-tools>
