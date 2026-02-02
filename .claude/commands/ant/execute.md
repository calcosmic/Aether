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

Initialize step tracking (using bash for progress display):
```bash
# Step tracking for progress display
declare -a STEPS=("Validate Input" "Load Colony State" "Emit Init Pheromone for Phase" "Set Phase to In Progress" "Spawn Worker Ants for Execution" "Execute with Emergence")
declare -a STEP_STATUS=("in_progress" "pending" "pending" "pending" "pending" "pending")

show_step_progress() {
  echo ""
  echo "📊 Execution Progress:"
  for i in "${!STEPS[@]}"; do
    local step_num=$((i + 1))
    local step="${STEPS[$i]}"
    local status="${STEP_STATUS[$i]}"

    case $status in
      completed) echo "  [✓] Step $step_num/6: $step" ;;
      in_progress) echo "  [→] Step $step_num/6: $step..." ;;
      failed) echo "  [🔴] Step $step_num/6: $step — failed" ;;
      *) echo "  [ ] Step $step_num/6: $step" ;;
    esac
  done
  echo ""
}

# Mark current step as in progress
update_step_status() {
  local step_num=$1
  local status=$2
  STEP_STATUS[$((step_num - 1))]=$status
  show_step_progress
}

# Show initial progress
show_step_progress
```


```python
if not args or not args[0].isdigit():
    return """❌ Usage: /ant:execute <phase_id>

Example:
  /ant:execute 1    # Execute Phase 1
"""

phase_id = int(args[0])
```

Mark step 1 complete:
```bash
update_step_status 1 "completed"
```

## Step 2: Load Colony State

Mark step 2 in progress:
```bash
update_step_status 2 "in_progress"
```

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

Mark step 2 complete:
```bash
update_step_status 2 "completed"
```

## Step 3: Emit Init Pheromone for Phase

Mark step 3 in progress:
```bash
update_step_status 3 "in_progress"
```

```
🐜 Queen Ant Colony - Phase Execution

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PHASE {phase_id}: {phase['name']}

Emitting INIT pheromone...
Colony mobilizing...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Mark step 3 complete:
```bash
update_step_status 3 "completed"
```

## Step 4: Set Phase to In Progress

Mark step 4 in progress:
```bash
update_step_status 4 "in_progress"
```

```python
phase['status'] = 'in_progress'
phase['started_at'] = datetime.now().isoformat()
state['current_phase_id'] = phase_id

with open('.aether/COLONY_STATE.json', 'w') as f:
    json.dump(state, f, indent=2)
```

Mark step 4 complete:
```bash
update_step_status 4 "completed"
```

## Step 5: Spawn Worker Ants for Execution

Mark step 5 in progress:
```bash
update_step_status 5 "in_progress"
```

Use Task tool to spawn Worker Ants to execute tasks:

```python
tasks = phase.get('tasks', [])

for task in tasks:
    if task['status'] == 'pending':
        # Determine which ant should handle this task
        task_desc = task['description'].lower()

        if any(word in task_desc for word in ["explore", "colonize", "understand"]):
            await self._spawn_colonizer_agent(task)
        elif any(word in task_desc for word in ["plan", "design"]):
            await self._spawn_planner_agent(task)
        elif any(word in task_desc for word in ["implement", "write", "create", "build"]):
            await self._spawn_executor_agent(task)
        elif any(word in task_desc for word in ["test", "verify", "validate"]):
            await self._spawn_verifier_agent(task)
        elif any(word in task_desc for word in ["research", "find", "lookup"]):
            await self._spawn_researcher_agent(task)
```

Mark step 5 complete:
```bash
update_step_status 5 "completed"
```

## Step 6: Execute with Emergence

Mark step 6 in progress:
```bash
update_step_status 6 "in_progress"
```

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
- Colonizer: Colonization tasks
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

Mark step 6 complete:
```bash
update_step_status 6 "completed"
```
</process>

<context>
# AETHER AUTONOMOUS SPAWNING SYSTEM - Claude Native Implementation

## Autonomous Spawning Decision Flow

When a Worker Ant receives a task, it follows this decision process:

### Step 1: Capability Gap Detection
Analyze task requirements vs own capabilities:
```
required_capabilities = analyze_task_requirements(task)
my_capabilities = get_my_capabilities()
gaps = required_capabilities - my_capabilities
```

Task analysis uses semantic pattern matching:
- Database patterns: database, sql, query, orm, migration, postgres, mysql, mongodb
- API patterns: api, endpoint, route, controller, rest, graphql, websocket
- Auth patterns: auth, login, jwt, session, oauth, password
- Testing patterns: test, spec, mock, unit, integration, e2e
- DevOps patterns: deploy, docker, k8s, ci, cd, infrastructure
- Security patterns: security, encrypt, vulnerability, owasp

### Step 2: Resource Budget Check
Before spawning, verify resource limits:
- current_subagents < max_subagents (10 per phase)
- spawn_depth < max_depth (3 levels max)
- spawning_disabled == false (circuit breaker check)

### Step 3: Specialist Type Determination
Map capability gaps to specialist types:
```
if "database" in gaps: return "database_specialist"
if "auth" or "security" in gaps: return "security_specialist"
if "testing" in gaps: return "test_specialist"
if "frontend" or "ui" in gaps: return "frontend_specialist"
if "backend" or "api" in gaps: return "api_specialist"
if "devops" or "deployment" in gaps: return "devops_specialist"
return "general_specialist"
```

### Step 4: Inherited Context Creation
Create context package for spawned specialist:
```python
inherited = {
    "parent_agent_id": parent_id,
    "parent_task": current_task,
    "goal": get_goal_from_init_pheromone(),
    "pheromone_signals": get_active_pheromones(),
    "working_memory": get_relevant_working_memory(),
    "relevant_code": await semantic_search_for_code(task),
    "constraints": get_constraints_from_redirect_pheromones()
}
```

### Step 5: Spawn via Task Tool
```
Task: {specialist_type}

You are a {specialist_type} spawned by {parent_caste}.

TASK: {task_description}

INHERITED CONTEXT:
- Goal: {goal}
- Active Pheromones: {pheromone_signals}
- Parent's Context: {working_memory}
- Constraints: {constraints}
- Relevant Code: {relevant_code}

CAPABILITY GAPS DETECTED: {gaps}
REASON: {spawning_reason}

Execute autonomously. Report results when complete.
```

## Worker Ant Caste Assignments

When tasks are ready, assign to appropriate caste:

### Task Type → Caste Mapping
- **Exploration/Colonization** → Colonizer Ant
- **Planning/Design** → Route-setter Ant
- **Implementation** → Builder Ant
- **Testing/Validation** → Watcher Ant
- **Research** → Scout Ant
- **Memory/Synthesis** → Architect Ant

## Pheromone-Based Behavior Modification

Active pheromones modify how each caste behaves:

### FOCUS Pheromone Effects
- Colonizer: Colonizes focused area first
- Planner: Plans focused tasks earlier
- Executor: Prioritizes focused work (sensitivity 0.9)
- Verifier: Intensifies testing in focused area
- Researcher: Researches focused topic first

### REDIRECT Pheromone Effects
- Executor: Strongly avoids pattern (sensitivity 0.9)
- Planner: Avoids in future plans (sensitivity 0.8)
- Verifier: Validates against constraint

### FEEDBACK Pheromone Effects
- Positive: Pattern reinforced, stored for reuse
- Quality: Verifier intensifies, Executor reviews code
- Speed: Executor parallelizes, Planner simplifies
- Direction: Planner pivots, Executor adjusts

## Resource Budget Tracking

Track spawning to prevent infinite loops:
```python
resource_budget = {
    "max_subagents": 10,
    "max_depth": 3,
    "current_subagents": 0,
    "spawning_disabled": false
}
```

Circuit breaker triggers:
- 3 failed spawns on same specialist type → cooldown
- Max subagents reached → disable spawning
- Max depth exceeded → must handle task personally

## Meta-Learning Integration (Future)

Track spawn outcomes for learning:
```python
spawn_event = {
    "parent_agent": caste,
    "task_description": task,
    "task_category": categorize(task),
    "specialist_type": specialist,
    "capability_gap": gaps,
    "timestamp": now()
}

outcome = {
    "success": bool,
    "quality_score": 0.0-1.0,
    "innovation_score": 0.0-1.0,
    "duration": seconds,
    "user_feedback": str
}
```

Bayesian confidence scoring:
```
confidence = (successes + 1) / (successes + failures + 2)
```

## Pure Emergence Within Phases

During phase execution:
- Worker Ants self-organize (no central coordination)
- Spawn specialists based on local needs
- Coordinate peer-to-peer
- Respond to pheromones in real-time
- No Queen intervention until phase boundary

## Task Assignment Example

```
Task: "Implement JWT authentication"

Analysis:
- Contains "jwt", "authentication" → security domain
- Requires implementation → Executor task
- Capability gap: "auth", "jwt" → security_specialist needed

Flow:
1. Executor detects capability gap
2. Checks resource budget (0/10 used, depth 0)
3. Determines specialist: security_specialist
4. Creates inherited context with goal, pheromones, constraints
5. Spawns via Task tool
6. Tracks spawn event for meta-learning
```

Aether Phased Autonomy:
- Structure at boundaries (phases)
- Emergence within phases
- Checkpoints between phases
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
