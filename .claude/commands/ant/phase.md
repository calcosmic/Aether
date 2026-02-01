---
name: ant:phase
description: Show phase details - Queen reviews phase status, tasks, and caste assignment
---

<objective>
Display detailed phase information including tasks, caste assignments, status, and requirements mapping. Enables Queen to review phase details before execution.
</objective>

<process>
You are the **Queen Ant Colony** displaying phase details.

## Step 1: Parse Arguments

User can call:
- `/ant:phase` - Show current phase
- `/ant:phase N` - Show specific phase
- `/ant:phase list` - List all phases

```bash
PHASE_NUM="${1:-current}"
```

## Step 2: Load Colony State

```bash
COLONY_STATE=".aether/data/COLONY_STATE.json"
ROADMAP=".planning/ROADMAP.md"

if [ ! -f "$COLONY_STATE" ]; then
  echo "⚠️  Colony not initialized"
  echo "Use /ant:init <goal> to initialize the colony"
  exit 1
fi

# Get current phase
CURRENT_PHASE=$(jq -r '.colony_status.current_phase' "$COLONY_STATE")
```

## Step 3: Display Phase Information

For single phase view:

```
╔══════════════════════════════════════════════════════════════╗
║  Phase {id}: {name}                                          ║
╠══════════════════════════════════════════════════════════════╣
║  Status: {status}                                            ║
║  Caste: {assigned_caste}                                     ║
║  Progress: {completed}/{total} tasks ({percentage}%)        ║
╚══════════════════════════════════════════════════════════════╝

🎯 Goal:
{phase_goal}

📋 Tasks ({task_count} total):
```

Display each task with status:
```
  [{status}] {task_id}: {description}
      Caste: {assigned_caste}
      Requirements: {req_list}
```

## Step 4: Show Requirements Mapping

```
📐 Requirements Covered:
{requirements_list}

📦 Dependencies:
{dependency_list}
```

## Step 5: Show Success Criteria

```
✅ Success Criteria ({count}):
{success_criteria_list}
```

## Step 6: Show Available Actions

Based on phase status:

```
📋 Available Actions:
```

If phase is "ready" or "pending":
```
  /ant:execute {phase_id} - Execute this phase
```

If phase is "in_progress":
```
  /ant:status - View detailed status
  /ant:focus <area> - Guide colony attention
```

If phase is "completed":
```
  /ant:review {phase_id} - Review completed phase
```

## Step 7: For Phase List View

```
╔══════════════════════════════════════════════════════════════╗
║  Aether v2 - 10 Phase Roadmap                               ║
╠══════════════════════════════════════════════════════════════╣
║  Progress: Phase {current}/10 - {overall_percentage}%      ║
╚══════════════════════════════════════════════════════════════╝

{phase_list_with_status}
```

</process>

<context>
# AETHER PHASE STRUCTURE

## All 10 Phases

1. **Colony Foundation** (caste: colonizer)
   - JSON state persistence and pheromone signal layer

2. **Worker Ant Castes** (caste: route_setter)
   - Six Worker Ant prompt behaviors with Task tool spawning

3. **Pheromone Communication** (caste: builder)
   - Stigmergic signals with caste sensitivity

4. **Triple-Layer Memory** (caste: architect)
   - Working → Short-term → Long-term with associative links

5. **Phase Boundaries** (caste: route_setter)
   - State machine with Queen check-ins and checkpoints

6. **Autonomous Emergence** (caste: builder)
   - Capability gap detection with Worker-spawns-Workers

7. **Colony Verification** (caste: watcher)
   - Multi-perspective verification with weighted voting

8. **Colony Learning** (caste: architect)
   - Meta-learning loop with Bayesian confidence scoring

9. **Stigmergic Events** (caste: scout)
   - Event bus for colony-wide pub/sub communication

10. **Colony Maturity** (caste: watcher)
    - End-to-end testing and production readiness

## Phase Status Values

- **pending**: Not started, waiting for previous phases
- **ready**: Ready to begin execution
- **in_progress**: Currently executing
- **completed**: Successfully completed
- **failed**: Failed, needs recovery

## Caste Assignments

Each phase has a primary caste that leads the work:
- **colonizer**: Exploration and mapping
- **route_setter**: Planning and structure
- **builder**: Implementation and execution
- **watcher**: Validation and quality
- **scout**: Research and information
- **architect**: Memory and knowledge

Other castes may be spawned as specialists during phase execution.
</context>

<reference>
# Example: Single Phase View

```
╔══════════════════════════════════════════════════════════════╗
║  Phase 1: Colony Foundation                                  ║
╠══════════════════════════════════════════════════════════════╣
║  Status: completed                                           ║
║  Caste: colonizer                                            ║
║  Progress: 8/8 tasks (100%)                                 ║
╚══════════════════════════════════════════════════════════════╝

🎯 Goal:
Colony state persists safely across context refreshes with corruption-proof JSON storage and pheromone signal system

📋 Tasks (8 total):
  [✓] 01-01: Create colony state schema (COLONY_STATE.json)
      Caste: builder
      Requirements: STATE-01

  [✓] 01-02: Create pheromone signal schema (pheromones.json)
      Caste: builder
      Requirements: STATE-02

  [✓] 01-03: Create Worker Ant state schema (worker_ants.json)
      Caste: builder
      Requirements: STATE-03

  [✓] 01-04: Create memory schema (memory.json)
      Caste: builder
      Requirements: STATE-04

  [✓] 01-05: Implement file locking mechanism
      Caste: builder
      Requirements: STATE-06

  [✓] 01-06: Implement atomic write pattern
      Caste: builder
      Requirements: STATE-07

  [✓] 01-07: Create /ant:init command prompt
      Caste: builder
      Requirements: CMD-01

  [✓] 01-08: Create /ant:status command prompt
      Caste: builder
      Requirements: CMD-02

📐 Requirements Covered:
- CMD-01: User can initialize project with /ant:init
- CMD-02: User can view colony status with /ant:status
- STATE-01 through STATE-07: State persistence requirements

✅ Success Criteria (5):
1. Queen can initialize colony and see COLONY_STATE.json created
2. Queen can run /ant:status and see colony state
3. Colony state persists across context refreshes
4. Multiple Worker Ants can read/write without corruption
5. Atomic writes prevent partial state corruption

📋 Available Actions:
  /ant:phase 2 - View next phase
  /ant:execute 2 - Execute Phase 2
  /ant:status - View colony status
```

# Example: Phase List View

```
╔══════════════════════════════════════════════════════════════╗
║  Aether v2 - 10 Phase Roadmap                               ║
╠══════════════════════════════════════════════════════════════╣
║  Progress: Phase 1/10 - 10%                                ║
╚══════════════════════════════════════════════════════════════╝

  [✓] Phase 1: Colony Foundation (colonizer)
      8/8 tasks • 5 requirements • 100%

  [→] Phase 2: Worker Ant Castes (route_setter)
      0/9 tasks • 7 requirements • 0%

  [ ] Phase 3: Pheromone Communication (builder)
      0/8 tasks • 8 requirements • 0%

  [ ] Phase 4: Triple-Layer Memory (architect)
      0/10 tasks • 11 requirements • 0%

  [ ] Phase 5: Phase Boundaries (route_setter)
      0/9 tasks • 13 requirements • 0%

  [ ] Phase 6: Autonomous Emergence (builder)
      0/8 tasks • 8 requirements • 0%

  [ ] Phase 7: Colony Verification (watcher)
      0/10 tasks • 10 requirements • 0%

  [ ] Phase 8: Colony Learning (architect)
      0/6 tasks • 6 requirements • 0%

  [ ] Phase 9: Stigmergic Events (scout)
      0/7 tasks • 7 requirements • 0%

  [ ] Phase 10: Colony Maturity (watcher)
      0/10 tasks • all requirements • 0%

Legend: [✓] completed [→] in progress [ ] pending
```
</reference>

<allowed-tools>
Read
Bash
Grep
</allowed-tools>
