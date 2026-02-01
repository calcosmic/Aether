---
name: ant:phase
description: Show current phase or specific phase details with state-aware prompts
---

<objective>
Display phase status with different output based on phase state (pending/in-progress/complete).
Shows tasks, Worker Ant activity, active pheromones, and next steps.
</objective>

<process>
You are the **Queen Ant Colony** displaying phase status.

## Step 1: Parse Arguments

If no phase_id provided, show current phase. Otherwise show specific phase.

## Step 2: Load Colony State

Read from `.aether/COLONY_STATE.json`:
```python
import json

with open('.aether/COLONY_STATE.json', 'r') as f:
    state = json.load(f)

phases = state.get('phases', [])
current_phase_id = state.get('current_phase_id')
```

## Step 3: Determine Which Phase to Display

```python
if phase_id is None:
    phase_id = current_phase_id

phase = next((p for p in phases if p['id'] == phase_id), None)

if not phase:
    return f"❌ Phase {phase_id} not found"
```

## Step 4: State-Aware Display

Format output based on phase status:

### PENDING Phase
```
PHASE {id}: {name} [PENDING]

Description: {description}

TASKS ({total}):
{List all tasks with ⏳ indicator}

MILESTONES:
  • {milestone_1}
  • {milestone_2}

📋 NEXT STEPS:
  1. /ant:execute {id}         - Start executing this phase
  2. /ant:focus <area>         - Guide colony attention

💡 COLONY RECOMMENDATION:
   Consider focusing on: "{suggested_focus_area}"

🔄 CONTEXT: This command is lightweight - safe to continue
```

### IN_PROGRESS Phase
```
PHASE {id}: {name} [IN_PROGRESS] {progress}% complete

Description: {description}

TASKS ({completed}/{total}):
{List tasks with appropriate indicators}

WORKER ANTS ACTIVE:
  • Mapper: {status} - {current_task}
  • Planner: {status} - {current_task}
  • Executor: {status} - {current_task}
  • Verifier: {status} - {current_task}
  • Researcher: {status} - {current_task}
  • Synthesizer: {status} - {current_task}

SUBAGENTS SPAWNED: {count}
  {List active subagents}

ACTIVE PHEROMONES: {count}
  [{type}] {message} (strength: {strength}%)

MILESTONES:
  ✅ {completed_milestone}
  ⏳ {pending_milestone}

📋 PROGRESS UPDATE:
   {estimated_tasks_remaining} tasks remaining
   Estimated time: {time_estimate}

🔄 CONTEXT: Phase execution in progress - use /ant:status for real-time updates
```

### COMPLETE Phase
```
PHASE {id}: {name} [✅ COMPLETE]

Description: {description}

COMPLETED: {completion_date}
DURATION: {duration}

TASKS: {total}/{total} completed
  {List all completed tasks with ✅}

MILESTONES REACHED:
  ✅ {milestone_1}
  ✅ {milestone_2}

KEY LEARNINGS:
  • {learning_1}
  • {learning_2}

ISSUES RESOLVED:
  • {issue_1} - {fix}
  • {issue_2} - {fix}

FEATURES DELIVERED:
  ✓ {feature_1}
  ✓ {feature_2}

📋 NEXT STEPS:
  1. /ant:review {id}          - Review completed work
  2. /ant:phase {next_id}      - Continue to next phase
  3. /ant:focus <area>         - Set focus for next phase

💡 COLONY RECOMMENDATION:
   {recommendation_for_next_phase}

🔄 CONTEXT: REFRESH RECOMMENDED
   Phase complete - safe to refresh Claude before continuing
```

### AWAITING_REVIEW Phase
```
PHASE {id}: {name} [⏸️ AWAITING REVIEW]

Phase execution complete. Queen review requested.

COMPLETION SUMMARY:
  Tasks: {completed}/{total}
  Milestones: {milestones_reached}/{total_milestones}
  Duration: {duration}
  Agents spawned: {count}

PENDING REVIEW:
  □ Review work completed
  □ Provide feedback via /ant:feedback
  □ Approve to continue or request changes

📋 NEXT STEPS:
  1. /ant:review {id}          - Review completed work
  2. /ant:feedback "<msg>"     - Provide feedback
  3. /ant:phase {id} approve   - Approve and continue

💡 RECOMMENDATION:
   Review the work before approving next phase.

🔄 CONTEXT: Review checkpoint - safe to refresh
```

</process>

<context>
@.aether/phase_engine.py
@.aether/worker_ants.py
@.aether/pheromone_system.py

Phase States:
- PENDING: Created but not started
- IN_PROGRESS: Currently executing
- AWAITING_REVIEW: Complete, waiting for Queen approval
- COMPLETED: Approved and finalized
- FAILED: Execution failed

Worker Ant Castes:
- Mapper: Exploration and codebase understanding
- Planner: Goal decomposition and phase planning
- Executor: Code implementation
- Verifier: Testing and validation
- Researcher: Information gathering
- Synthesizer: Knowledge synthesis and memory compression
</context>

<reference>
# Task Status Indicators

- ⏳ Pending
- 🔄 In Progress
- ✅ Complete
- ❌ Failed
- ⏸️ Blocked

# Pheromone Display Format

```
[INIT] Goal message (strength: 100%)
[FOCUS] Area to focus on (strength: 70%)
[REDIRECT] Pattern to avoid (strength: 80%)
[FEEDBACK] Feedback message (strength: 60%)
```
</reference>

<allowed-tools>
Read
Write
Bash
Glob
Grep
</allowed-tools>
