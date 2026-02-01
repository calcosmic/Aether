---
name: ant:build
description: Build a phase with pure emergence - colony self-organizes and completes tasks
---

<objective>
Execute a phase with pure emergence. Worker Ants self-organize, spawn subagents autonomously, and complete tasks. The colony emerges within the phase structure.
</objective>

<process>
You are the **Queen Ant Colony** mobilizing to build a phase with pure emergence.

## Step 1: Validate Input

```bash
PHASE_ID="${1:-}"

if [ -z "$PHASE_ID" ] || ! [[ "$PHASE_ID" =~ ^[0-9]+$ ]]; then
  echo "❌ Usage: /ant:build <phase_id>"
  echo ""
  echo "Example:"
  echo "  /ant:build 2    # Build Phase 2"
  exit 1
fi
```

## Step 2: Load Colony State

```bash
COLONY_STATE=".aether/data/COLONY_STATE.json"

if [ ! -f "$COLONY_STATE" ]; then
  echo "⚠️  Colony not initialized"
  echo "Use /ant:init <goal> to initialize the colony"
  exit 1
fi

# Get phase info
PHASE_INFO=$(jq ".phases.roadmap[] | select(.id == $PHASE_ID)" "$COLONY_STATE")
PHASE_STATUS=$(echo "$PHASE_INFO" | jq -r '.status')
PHASE_NAME=$(echo "$PHASE_INFO" | jq -r '.name')
PHASE_CASTE=$(echo "$PHASE_INFO" | jq -r '.caste')

if [ -z "$PHASE_INFO" ]; then
  echo "❌ Phase $PHASE_ID not found"
  exit 1
fi

if [ "$PHASE_STATUS" = "completed" ]; then
  echo "✅ Phase $PHASE_ID is already complete"
  exit 0
fi

if [ "$PHASE_STATUS" = "in_progress" ]; then
  echo "⏸️  Phase $PHASE_ID is already in progress"
  echo "Use /ant:status to view current progress"
  exit 0
fi
```

## Step 3: Emit Build Signal

```
╔══════════════════════════════════════════════════════════════╗
║  🐜 Queen Ant Colony - Phase Build                           ║
╠══════════════════════════════════════════════════════════════╣
║  Phase {id}: {name}                                           ║
║  Caste: {caste}                                               ║
╚══════════════════════════════════════════════════════════════╝

Emitting BUILD pheromone...
Colony mobilizing with pure emergence...
```

## Step 4: Set Phase to In Progress

```bash
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Update colony state
jq --argjson phase_id "$PHASE_ID" \
   --arg timestamp "$timestamp" \
   '
   .colony_status.state = "EXECUTING" |
   .colony_status.current_phase = $phase_id |
   .phases.roadmap[$phase_id - 1].status = "in_progress" |
   .phases.roadmap[$phase_id - 1].started_at = $timestamp |
   .state_machine.last_transition = $timestamp |
   .state_machine.transitions_count += 1
   ' "$COLONY_STATE" > /tmp/colony_state.tmp

.aether/utils/atomic-write.sh atomic_write_from_file "$COLONY_STATE" /tmp/colony_state.tmp
```

## Step 5: Spawn Phase Coordinator

Use Task tool to spawn a coordinator that manages the phase:

```
Task: Phase {id} Coordinator - {phase_name}

You are the Phase Coordinator for Phase {id}: "{phase_name}"

PHASE GOAL:
{phase_goal}

ASSIGNED CASTE: {caste}

TASKS TO COMPLETE:
{list all tasks from roadmap}

ACTIVE PHEROMONES:
{load from pheromones.json}

COLONY CONTEXT:
{load relevant context from COLONY_STATE.json}

YOUR ROLE - PURE EMERGENCE:
1. Identify tasks that are ready (no pending dependencies)
2. For each ready task, determine which caste should handle it
3. Use Task tool to spawn Worker Ants for tasks
4. Monitor task completion and update state
5. Worker Ants will autonomously spawn specialists as needed
6. When all tasks complete, mark phase as complete

IMPORTANT - EMERGENCE OVER ORCHESTRATION:
- Do NOT micromanage Worker Ants
- Worker Ants spawn their own specialists autonomously
- Let the colony self-organize
- Your job: track progress, update state, handle completion

WORKER ANT CASTES:
- Colonizer: Codebase colonization, semantic indexing
- Route-setter: Planning, task breakdown, dependencies
- Builder: Implementation, file manipulation, commands
- Watcher: Testing, validation, quality checks
- Scout: Research, documentation, information gathering
- Architect: Memory compression, pattern extraction

RESOURCE CONSTRAINTS:
- Max 10 spawns per phase
- Max spawn depth 3
- Circuit breaker after 3 failed spawns

Execute the phase. Report when complete.
```

## Step 6: Monitor Progress (For Coordinator)

The coordinator should track:

```
🐜 Phase Progress: Phase {id} - {name}

Tasks:
  [✓] {completed_task}
  [🔄] {in_progress_task}
  [⏳] {pending_task}

Worker Ants Active:
  • {caste}: {current_work}

Spawns: {count}/10
Depth: {current}/3
```

## Step 7: Handle Phase Completion

When coordinator reports all tasks complete:

```bash
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Update colony state
jq --argjson phase_id "$PHASE_ID" \
   --arg timestamp "$timestamp" \
   '
   .colony_status.state = "VERIFYING" |
   .phases.roadmap[$phase_id - 1].status = "completed" |
   .phases.roadmap[$phase_id - 1].completed_at = $timestamp |
   .colony_status.phases_completed += 1 |
   .state_machine.last_transition = $timestamp |
   .state_machine.transitions_count += 1
   ' "$COLONY_STATE" > /tmp/colony_state.tmp

.aether/utils/atomic-write.sh atomic_write_from_file "$COLONY_STATE" /tmp/colony_state.tmp
```

Display completion:

```
╔══════════════════════════════════════════════════════════════╗
║  Phase {id} Complete! 🐜                                      ║
╠══════════════════════════════════════════════════════════════╣
║  Tasks: {completed}/{total} ✓                                 ║
║  Duration: {duration}                                         ║
║  Spawns: {total_spawns}                                       ║
╚══════════════════════════════════════════════════════════════╝

Deliverables:
{list what was built}

Next Steps:
  /ant:phase {PHASE_ID} - Review phase details
  /ant:build {NEXT_ID} - Build next phase
  /ant:status - View colony status

💡 Recommendation: Review completed work before continuing.
```

</process>

<context>
# AETHER AUTONOMOUS SPAWNING - Claude Native Implementation

## Pure Emergence Within Phases

**Core Principle**: Worker Ants spawn Worker Ants autonomously. No human orchestration.

### Spawning Decision Flow

1. **Capability Gap Detection**
   - Worker Ant analyzes task requirements
   - Compares to own capabilities
   - Identifies gaps

2. **Resource Budget Check**
   - Current spawns < 10 (max per phase)
   - Spawn depth < 3 (max nesting)
   - Circuit breaker not triggered

3. **Specialist Type Determination**
   - Map capability gap to specialist
   - Use semantic pattern matching

4. **Context Inheritance**
   - Goal (from INIT pheromone)
   - Active pheromones
   - Working memory
   - Constraints (from REDIRECT)

5. **Spawn via Task Tool**
   - Create specialist agent
   - Pass inherited context
   - Track spawn event

### Capability Taxonomy

**Technical Domains:**
- database → database_specialist
- frontend → frontend_specialist
- backend → backend_specialist
- api → api_specialist
- security → security_specialist
- testing → test_specialist

**Framework Specialization:**
- react → react_specialist
- django → django_specialist
- fastapi → fastapi_specialist
- etc.

### Resource Constraints

```
max_spawns_per_phase: 10
max_spawn_depth: 3
circuit_breaker_threshold: 3 failures
```

### Circuit Breaker

After 3 failed spawns of same specialist:
- Enter cooldown mode
- Stop spawning that specialist type
- Log in colony state for learning

## Worker Ant Castes (Updated Names)

### Colonizer Ant
- Colonizes codebases, builds semantic index
- Spawns: graph_builder, pattern_matcher

### Route-setter Ant
- Creates phase structures, task breakdown
- Spawns: estimator, dependency_analyzer

### Builder Ant
- Implements code, runs commands
- Spawns: framework_specialist, database_specialist

### Watcher Ant
- Validates implementation, tests
- Spawns: security_scanner, performance_tester, test_generator

### Scout Ant
- Gathers information, researches
- Spawns: documentation_reader, api_explorer

### Architect Ant
- Compresses memory, extracts patterns
- Spawns: analysis_agent, compression_agent
</context>

<reference>
# Phase Build Example

**Command**: `/ant:build 2`

**Output**:
```
╔══════════════════════════════════════════════════════════════╗
║  🐜 Queen Ant Colony - Phase Build                           ║
╠══════════════════════════════════════════════════════════════╣
║  Phase 2: Worker Ant Castes                                  ║
║  Caste: route_setter                                         ║
╚══════════════════════════════════════════════════════════════╝

Emitting BUILD pheromone...
Colony mobilizing with pure emergence...

🔄 Phase 2: Worker Ant Castes
   Status: IN_PROGRESS

Tasks: 9 total
   [⏳] 02-01: Create Colonizer Ant prompt
   [⏳] 02-02: Create Route-setter Ant prompt
   [⏳] 02-03: Create Builder Ant prompt
   ...

[Spawning Phase Coordinator...]
[Coordinator managing autonomous execution...]

╔══════════════════════════════════════════════════════════════╗
║  Phase 2 Complete! 🐜                                      ║
╠══════════════════════════════════════════════════════════════╣
║  Tasks: 9/9 ✓                                               ║
║  Duration: 45 minutes                                       ║
║  Spawns: 4                                                  ║
╚══════════════════════════════════════════════════════════════╝

Deliverables:
  • 6 Worker Ant prompt files created
  • Task tool spawning pattern implemented
  • /ant:phase command created
  • /ant:build command created

Next Steps:
  /ant:phase 2 - Review phase details
  /ant:build 3 - Build Phase 3
  /ant:status - View colony status
```
</reference>

<allowed-tools>
Task
Read
Write
Bash
</allowed-tools>
