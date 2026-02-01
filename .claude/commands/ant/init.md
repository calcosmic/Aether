---
name: ant:init
description: Initialize Aether colony - Queen sets intention, colony mobilizes with predefined roadmap
---

<objective>
Initialize the Aether Queen Ant Colony by:
1. Setting the Queen's intention via INIT pheromone
2. Initializing colony state with the predefined 10-phase roadmap
3. Creating all state files with proper schemas
4. Setting colony to INIT state, ready for Phase 1

Aether v2 has a fixed 10-phase roadmap for building the colony infrastructure.
</objective>

<process>
You are the **Queen Ant Colony** receiving an intention from the Queen.

## Step 1: Validate Preconditions
Check if colony is already initialized:
```bash
# Check if COLONY_STATE.json exists and has a goal
if [ -f .aether/data/COLONY_STATE.json ]; then
  # Read current goal
  current_goal=$(jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json)
  if [ "$current_goal" != "null" ] && [ -n "$current_goal" ]; then
    echo "⚠️  Colony already initialized with goal: $current_goal"
    echo "Use /ant:status to view current state"
    exit 1
  fi
fi
```

## Step 2: Receive Intention
The user provides a goal. Store it as the colony's intention:
```
🐜 Queen's Intention: "{goal}"
```

## Step 3: Initialize Colony State
Update COLONY_STATE.json with the Queen's intention:
```bash
# Generate session ID
session_id="session_$(date +%s)_$RANDOM"
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Update colony state
jq --arg goal "$1" \
   --arg session "$session_id" \
   --arg timestamp "$timestamp" \
   '
   .colony_metadata.session_id = $session |
   .colony_metadata.created_at = $timestamp |
   .colony_metadata.last_updated = $timestamp |
   .queen_intention.goal = $goal |
   .queen_intention.initialized_at = $timestamp |
   .colony_status.state = "INIT" |
   .colony_status.current_phase = 1 |
   .state_machine.last_transition = $timestamp |
   .state_machine.transitions_count = 1 |
   .state_machine.last_state = "IDLE" |
   .phases.roadmap[0].status = "ready"
   ' .aether/data/COLONY_STATE.json > /tmp/colony_state.tmp

# Atomic write
.aether/utils/atomic-write.sh atomic_write_from_file .aether/data/COLONY_STATE.json /tmp/colony_state.tmp
```

## Step 4: Emit INIT Pheromone
Create the INIT pheromone signal:
```bash
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
pheromone_id="init_$(date +%s)"

jq --arg id "$pheromone_id" \
   --arg timestamp "$timestamp" \
   --arg goal "$1" \
   '
   .active_pheromones += [{
     "id": $id,
     "type": "INIT",
     "strength": 1.0,
     "created_at": $timestamp,
     "decay_rate": null,
     "metadata": {
       "source": "queen",
       "caste": null,
       "context": $goal
     }
   }]
   ' .aether/data/pheromones.json > /tmp/pheromones.tmp

# Atomic write
.aether/utils/atomic-write.sh atomic_write_from_file .aether/data/pheromones.json /tmp/pheromones.tmp
```

## Step 5: Set Worker Ants to Ready State
All Worker Ants should be mobilized (status: ready, not idle):
```bash
jq '
  .castes |= with_entries(
    .value.status = "ready" |
    .value.current_phase = 1
  )
' .aether/data/worker_ants.json > /tmp/worker_ants.tmp

# Atomic write
.aether/utils/atomic-write.sh atomic_write_from_file .aether/data/worker_ants.json /tmp/worker_ants.tmp
```

## Step 6: Initialize Working Memory
Add the intention to working memory:
```bash
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
memory_id="mem_$(date +%s)"

jq --arg id "$memory_id" \
   --arg timestamp "$timestamp" \
   --arg goal "$1" \
   '
   .working_memory.items += [{
     "id": $id,
     "type": "intention",
     "content": $goal,
     "metadata": {
       "timestamp": $timestamp,
       "relevance_score": 1.0,
       "access_count": 1,
       "last_accessed": $timestamp,
       "source": "queen",
       "caste": null
     },
     "associative_links": []
   }]
' .aether/data/memory.json > /tmp/memory.tmp

# Atomic write
.aether/utils/atomic-write.sh atomic_write_from_file .aether/data/memory.json /tmp/memory.tmp
```

## Step 7: Present Results
Show the Queen (user) the colony initialization:

```
╔══════════════════════════════════════════════════════════════╗
║  🐜 Queen Ant Colony Initialized                             ║
╠══════════════════════════════════════════════════════════════╣
║  Session: {session_id}                                       ║
║  Initialized: {timestamp}                                    ║
║                                                               ║
║  Queen's Intention:                                           ║
║  "{goal}"                                                    ║
║                                                               ║
║  Colony Status: INIT                                         ║
║  Current Phase: 1 - Colony Foundation                        ║
║  Roadmap: 10 phases ready                                    ║
║                                                               ║
║  Active Pheromones:                                          ║
║  ✓ INIT (strength 1.0, persists)                             ║
║                                                               ║
║  Worker Ants Mobilized:                                      ║
║  ✓ Colonizer (ready)                                         ║
║  ✓ Route-setter (ready)                                      ║
║  ✓ Builder (ready)                                           ║
║  ✓ Watcher (ready)                                           ║
║  ✓ Scout (ready)                                             ║
║  ✓ Architect (ready)                                         ║
╚══════════════════════════════════════════════════════════════╝

✨ COLONY MOBILIZED

Next Steps:
  /ant:status   - View detailed colony status
  /ant:plan     - Show full 10-phase roadmap
  /ant:phase 1  - Review Phase 1 details
  /ant:focus    - Guide colony attention (optional)
```

</process>

<context>
# AETHER ARCHITECTURE - Queen Ant Colony v2

## Core Philosophy
**Autonomous Emergence**: Worker Ants autonomously spawn other Worker Ants without human orchestration. Queen provides intention via pheromones, colony self-organizes.

## Fixed 10-Phase Roadmap

Aether v2 follows a predefined roadmap to build the colony infrastructure:

1. **Colony Foundation** - JSON state persistence and pheromone signal layer
2. **Worker Ant Castes** - Six Worker Ant prompt behaviors with Task tool spawning
3. **Pheromone Communication** - Stigmergic signals with caste sensitivity
4. **Triple-Layer Memory** - Working → Short-term → Long-term with associative links
5. **Phase Boundaries** - State machine with Queen check-ins and checkpoints
6. **Autonomous Emergence** - Capability gap detection with Worker-spawns-Workers
7. **Colony Verification** - Multi-perspective verification with weighted voting
8. **Colony Learning** - Meta-learning loop with Bayesian confidence scoring
9. **Stigmergic Events** - Event bus for colony-wide pub/sub communication
10. **Colony Maturity** - End-to-end testing and production readiness

## Worker Ant Castes

The six Worker Ant castes designed from first principles for autonomous emergence:

### Colonizer Ant
- **Purpose**: Colonizes codebase, builds semantic index, detects patterns
- **Spawns When**: System init or new codebase encountered
- **Sensitivity**: INIT 1.0, FOCUS 0.8, REDIRECT 0.9, FEEDBACK 0.7

### Route-setter Ant
- **Purpose**: Creates phase structures, task breakdown, dependency analysis
- **Spawns When**: Goal requires decomposition
- **Sensitivity**: INIT 1.0, FOCUS 0.9, REDIRECT 0.8, FEEDBACK 0.8

### Builder Ant
- **Purpose**: Implements code, runs commands, file manipulation
- **Spawns When**: Concrete tasks identified
- **Sensitivity**: INIT 0.9, FOCUS 1.0, REDIRECT 0.7, FEEDBACK 0.9

### Watcher Ant
- **Purpose**: Validates implementation, testing, quality checks
- **Spawns When**: Builder completes work
- **Sensitivity**: INIT 0.8, FOCUS 0.9, REDIRECT 1.0, FEEDBACK 1.0

### Scout Ant
- **Purpose**: Gathers information, searches docs, context retrieval
- **Spawns When**: Unknown domain encountered
- **Sensitivity**: INIT 0.9, FOCUS 0.7, REDIRECT 0.8, FEEDBACK 0.8

### Architect Ant
- **Purpose**: Memory compression, pattern extraction, knowledge synthesis
- **Spawns When**: Memory capacity reached or phase boundary
- **Sensitivity**: INIT 0.8, FOCUS 0.8, REDIRECT 0.9, FEEDBACK 1.0

## Pheromone Signal System

### Signal Types
- **INIT**: Strength 1.0, Persists until phase complete (no decay)
- **FOCUS**: Strength 0.7, 1 hour half-life - guides colony attention
- **REDIRECT**: Strength 0.9, 24 hour half-life - warns away from approaches
- **FEEDBACK**: Strength 0.5, 6 hour half-life - provides guidance

### Effective Strength
```
EffectiveStrength = SignalStrength × CasteSensitivity
```
Different castes respond differently to the same signal based on their sensitivity profile.

## Resource Constraints
- Max spawns per phase: 10
- Max spawn depth: 3 levels (parent → child → grandchild)
- Circuit breaker: 3 failed spawns → cooldown

## State Machine States
- IDLE → INIT → PLANNING → EXECUTING → VERIFYING → COMPLETED/FAILED
</context>

<reference>
# State File Locations

All colony state is persisted in `.aether/data/`:

- **COLONY_STATE.json** - Main colony state, phases, resource budgets
- **pheromones.json** - Active pheromone signals
- **worker_ants.json** - Worker Ant castes and their states
- **memory.json** - Triple-layer memory (working, short-term, long-term)

All file operations use atomic writes via `.aether/utils/atomic-write.sh` to prevent corruption.
</reference>

<allowed-tools>
Write
Bash
Read
</allowed-tools>
