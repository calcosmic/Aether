---
name: ant:status
description: Show Queen Ant Colony status - Worker Ants, pheromones, phase progress
---

<objective>
Display comprehensive colony status including Worker Ant activity, active pheromones, phase progress, and colony health for the Aether v2 system.
</objective>

<process>
You are the **Queen Ant Colony** displaying comprehensive status.

## Step 1: Load Colony State

Read all state files from `.aether/data/`:
```bash
# Load colony state
COLONY_STATE=".aether/data/COLONY_STATE.json"
WORKER_ANTS=".aether/data/worker_ants.json"
PHEROMONES=".aether/data/pheromones.json"
MEMORY=".aether/data/memory.json"

# Check if colony is initialized
if [ ! -f "$COLONY_STATE" ]; then
  echo "⚠️  Colony not initialized"
  echo "Use /ant:init <goal> to initialize the colony"
  exit 1
fi

# Source checkpoint utility for crash detection
source .aether/utils/checkpoint.sh

# Auto-detect and recover from crashes
detect_crash_and_recover

# Extract key values
GOAL=$(jq -r '.queen_intention.goal' "$COLONY_STATE")
STATE=$(jq -r '.colony_status.state' "$COLONY_STATE")
CURRENT_PHASE=$(jq -r '.colony_status.current_phase' "$COLONY_STATE")
SESSION_ID=$(jq -r '.colony_metadata.session_id' "$COLONY_STATE")
CREATED_AT=$(jq -r '.colony_metadata.created_at' "$COLONY_STATE")
```

## Step 2: Define Helper Functions

```bash
# Map status to emoji with text label for accessibility
get_status_emoji() {
  local status=$1
  case $status in
    ACTIVE|active)   echo "🟢 ACTIVE" ;;
    IDLE|idle)     echo "⚪ IDLE" ;;
    PENDING|pending|ready|READY)  echo "⏳ PENDING" ;;
    ERROR|error)    echo "🔴 ERROR" ;;
    *)        echo "❓ UNKNOWN" ;;
  esac
}

# Display pheromone strength as visual progress bar
show_progress_bar() {
  local value=$1  # 0.0 to 1.0
  local width=${2:-20}

  # Clamp value to 0.0-1.0 range
  value=$(awk -v v="$value" 'BEGIN {print (v < 0 ? 0 : (v > 1 ? 1 : v))}')

  local filled=$(awk -v v="$value" -v w="$width" 'BEGIN {printf "%.0f", v * w}')
  local empty=$((width - filled))

  # Build bar with box-drawing character
  printf "["
  printf "━%.0s" $(seq 1 $filled)
  printf " %.0s" $(seq 1 $empty)
  printf "] %.2f\n" "$value"
}
```

## Step 3: Display Status Header

```
╔══════════════════════════════════════════════════════════════╗
║  🐜 Queen Ant Colony Status                                 ║
╠══════════════════════════════════════════════════════════════╣
║  Session: {session_id}                                       ║
║  State: {state}                                              ║
║  Initialized: {created_at}                                   ║
╚══════════════════════════════════════════════════════════════╝
```

## Step 4: Display Goal and Current Phase

```
🎯 Queen's Intention:
"{goal}"

📍 Current Phase: Phase {id} - {name}
   Status: {status}
   Caste: {assigned_caste}
   Goal: {phase_goal}
```

## Step 5: Display Worker Ant Status

```
🐜 Worker Ant Colony:
```

Group workers by activity state with emoji indicators:
```bash
# Parse worker_ants.json and group by status
declare -a active_workers=()
declare -a idle_workers=()
declare -a error_workers=()
declare -a pending_workers=()

# Extract worker registry data
jq -r '.worker_registry | to_entries[] | "\(.key)|\(.value.status)|\(.value.caste // "N/A")"' "$WORKER_ANTS" | while IFS='|' read -r worker_id status caste; do
  case $status in
    active|ACTIVE)
      echo "  🟢 ACTIVE: $worker_id ($caste)"
      ;;
    idle|IDLE)
      echo "  ⚪ IDLE: $worker_id ($caste)"
      ;;
    error|ERROR)
      echo "  🔴 ERROR: $worker_id ($caste)"
      ;;
    pending|PENDING|ready|READY)
      echo "  ⏳ PENDING: $worker_id ($caste)"
      ;;
    *)
      echo "  ❓ UNKNOWN: $worker_id ($caste) - $status"
      ;;
  esac
done

# Count workers by status
active_count=$(jq '[.worker_registry[] | select(.status == "ACTIVE" or .status == "active")] | length' "$WORKER_ANTS")
idle_count=$(jq '[.worker_registry[] | select(.status == "IDLE" or .status == "idle")] | length' "$WORKER_ANTS")
error_count=$(jq '[.worker_registry[] | select(.status == "ERROR" or .status == "error")] | length' "$WORKER_ANTS")
pending_count=$(jq '[.worker_registry[] | select(.status == "PENDING" or .status == "pending" or .status == "ready" or .status == "READY")] | length' "$WORKER_ANTS")

echo ""
echo "Summary: 🟢 $active_count ACTIVE | ⚪ $idle_count IDLE | 🔴 $error_count ERROR | ⏳ $pending_count PENDING"
```

## Step 6: Display Active Pheromones

```
🌿 Active Pheromones:
```

```bash
# Count and display active pheromones with progress bars
pheromone_count=$(jq '.active_pheromones | length' "$PHEROMONES")

if [ "$pheromone_count" -eq 0 ]; then
  echo "  No active pheromones"
else
  jq -r '.active_pheromones[] | "\(.type)|\(.signal)|\(.strength)|\(.timestamp)"' "$PHEROMONES" | while IFS='|' read -r type signal strength timestamp; do
    echo "  [$type] $signal"
    echo "    Strength: $(show_progress_bar "$strength")"
    echo "    Updated: $timestamp"
    echo ""
  done
fi
```

## Step 7: Display Phase Progress

```
📊 Phase Progress:
```

```bash
# Calculate phase progress
completed=$(jq '[.phases.roadmap[] | select(.status == "completed")] | length' "$COLONY_STATE")
in_progress=$(jq '[.phases.roadmap[] | select(.status == "in_progress")] | length' "$COLONY_STATE")
pending=$(jq '[.phases.roadmap[] | select(.status == "pending" or .status == "ready")] | length' "$COLONY_STATE")
total=10

percentage=$((completed * 100 / total))

echo "  Completed: $completed/$total"
echo "  In Progress: $in_progress"
echo "  Pending: $pending"
echo "  Overall: $percentage%"
```

Display progress bar:
```
  [$([######..........] for 60%, etc.)
```

## Step 8: Display Resource Budgets

```
⚡ Resource Budgets:
```

```bash
# Show resource usage
current_spawns=$(jq '.resource_budgets.current_spawns' "$COLONY_STATE")
max_spawns=$(jq '.resource_budgets.max_spawns_per_phase' "$COLONY_STATE")
breaker_trips=$(jq '.resource_budgets.circuit_breaker_trips' "$COLONY_STATE")

echo "  Spawns: $current_spawns/$max_spawns this phase"
echo "  Circuit Breaker Trips: $breaker_trips"
```

## Step 9: Display Memory Status

```
🧠 Memory Status:
```

```bash
# Show memory layer status
working_items=$(jq '.working_memory.items | length' "$MEMORY")
working_tokens=$(jq '.working_memory.current_tokens' "$MEMORY")
working_max=$(jq '.working_memory.max_capacity_tokens' "$MEMORY")
working_pct=$((working_tokens * 100 / working_max))

short_term_sessions=$(jq '.short_term_memory.sessions | length' "$MEMORY")
short_term_max=$(jq '.short_term_memory.max_sessions' "$MEMORY")

long_term_patterns=$(jq '.long_term_memory.patterns | length' "$MEMORY")

echo "  Working Memory: $working_tokens/$working_max tokens ($working_pct%)"
echo "    - Items: $working_items"
echo "  Short-term Memory: $short_term_sessions/$short_term_max sessions"
echo "  Long-term Memory: $long_term_patterns patterns"
```

## Step 10: Display Performance Metrics

```
📈 Performance:
```

```bash
# Show metrics
total_time=$(jq '.performance_metrics.total_execution_time_seconds' "$COLONY_STATE")
phases_complete=$(jq '.performance_metrics.phases_completed' "$COLONY_STATE")
total_spawns=$(jq '.performance_metrics.total_spawns' "$COLONY_STATE")
success_rate=$(jq '.performance_metrics.successful_spawns / .total_spawns * 100' "$COLONY_STATE")

echo "  Total Execution: ${total_time}s"
echo "  Phases Completed: $phases_complete"
echo "  Total Spawns: $total_spawns"
echo "  Success Rate: ${success_rate}%"
```

## Step 11: Display Available Actions

Based on current state, show relevant actions:

```
📋 Available Actions:

  /ant:plan     - Show full 10-phase roadmap
  /ant:phase N  - View phase details
  /ant:execute N - Execute a phase
  /ant:focus    - Emit focus pheromone
  /ant:redirect - Emit redirect pheromone
  /ant:feedback - Emit feedback pheromone
```

If colony is not initialized:
```
  /ant:init <goal> - Initialize the colony
```

## Step 12: Display State History (Optional)

If state has history:
```
📜 Recent State Transitions:
```

```bash
jq -r '.colony_status.state_history[-5:] | .[] | "  \(.timestamp) - \(.from_state) → \(.to_state)"' "$COLONY_STATE"
```

</process>

<context>
# AETHER COLONY STATUS - Complete Caste Information

## Worker Ant Castes (6 unique castes with detailed profiles)

### 1. Colonizer Ant
- **Purpose**: Colonizes codebase, builds semantic index, detects patterns
- **Sensitivity**: INIT=1.0, FOCUS=0.8, REDIRECT=0.9, FEEDBACK=0.7
- **Status Values**: idle, ready, active, blocked, spawning
- **Capabilities**: codebase_analysis, semantic_indexing, pattern_detection, dependency_mapping

### 2. Route-setter Ant
- **Purpose**: Creates phase structures, task breakdown, dependency analysis
- **Sensitivity**: INIT=1.0, FOCUS=0.9, REDIRECT=0.8, FEEDBACK=0.8
- **Status Values**: idle, ready, active, blocked, spawning
- **Capabilities**: phase_planning, task_breakdown, dependency_analysis, resource_allocation

### 3. Builder Ant
- **Purpose**: Implements code, runs commands, file manipulation
- **Sensitivity**: INIT=0.9, FOCUS=1.0, REDIRECT=0.7, FEEDBACK=0.9
- **Status Values**: idle, ready, active, blocked, spawning
- **Capabilities**: code_implementation, command_execution, file_operations, testing_setup

### 4. Watcher Ant
- **Purpose**: Validates implementation, testing, quality checks
- **Sensitivity**: INIT=0.8, FOCUS=0.9, REDIRECT=1.0, FEEDBACK=1.0
- **Status Values**: idle, ready, active, blocked, spawning
- **Capabilities**: validation, testing, quality_checks, security_review

### 5. Scout Ant
- **Purpose**: Gathers information, searches docs, context retrieval
- **Sensitivity**: INIT=0.9, FOCUS=0.7, REDIRECT=0.8, FEEDBACK=0.8
- **Status Values**: idle, ready, active, blocked, spawning
- **Capabilities**: information_gathering, documentation_search, context_retrieval

### 6. Architect Ant
- **Purpose**: Memory compression, pattern extraction, knowledge synthesis
- **Sensitivity**: INIT=0.8, FOCUS=0.8, REDIRECT=0.9, FEEDBACK=1.0
- **Status Values**: idle, ready, active, blocked, spawning
- **Capabilities**: memory_compression, pattern_extraction, knowledge_synthesis

## Pheromone Signal Types

### INIT
- **Purpose**: Set colony intention, mobilize colony
- **Strength**: 1.0 (maximum)
- **Duration**: Persists until phase complete
- **Effect**: Strong attract, all castes respond

### FOCUS
- **Purpose**: Guide colony attention to specific area
- **Strength**: 0.7 (default)
- **Duration**: 1 hour half-life
- **Effect**: Medium attract, guides prioritization

### REDIRECT
- **Purpose**: Warn colony away from approach/pattern
- **Strength**: 0.9
- **Duration**: 24 hour half-life
- **Effect**: Strong repel, prevents bad patterns

### FEEDBACK
- **Purpose**: Adjust colony behavior based on Queen's feedback
- **Strength**: 0.5-0.7 (variable)
- **Duration**: 6 hour half-life
- **Effect**: Variable, adjusts behavior

## Colony State Machine

### States
- **IDLE**: No active phase
- **INIT**: Colony initializing
- **PLANNING**: Phase planning in progress
- **EXECUTING**: Phase execution in progress
- **VERIFYING**: Phase verification in progress
- **COMPLETED**: Phase complete, awaiting review
- **FAILED**: Phase failed

## Aether v2 10-Phase Roadmap

1. Colony Foundation - JSON state persistence and pheromone signal layer
2. Worker Ant Castes - Six Worker Ant prompt behaviors with Task tool spawning
3. Pheromone Communication - Stigmergic signals with caste sensitivity
4. Triple-Layer Memory - Working → Short-term → Long-term with associative links
5. Phase Boundaries - State machine with Queen check-ins and checkpoints
6. Autonomous Emergence - Capability gap detection with Worker-spawns-Workers
7. Colony Verification - Multi-perspective verification with weighted voting
8. Colony Learning - Meta-learning loop with Bayesian confidence scoring
9. Stigmergic Events - Event bus for colony-wide pub/sub communication
10. Colony Maturity - End-to-end testing and production readiness
</context>

<reference>
# Example Full Output (Colony Initialized)

```
╔══════════════════════════════════════════════════════════════╗
║  🐜 Queen Ant Colony Status                                 ║
╠══════════════════════════════════════════════════════════════╣
║  Session: session_1738392000_12345                          ║
║  State: INIT                                                ║
║  Initialized: 2025-02-01T15:00:00Z                          ║
╚══════════════════════════════════════════════════════════════╝

🎯 Queen's Intention:
"Build the Aether colony infrastructure"

📍 Current Phase: Phase 1 - Colony Foundation
   Status: ready
   Caste: colonizer
   Goal: Colony state persists safely across context refreshes

🐜 Worker Ant Colony:
  ✋ Colonizer Ant - READY
  ✋ Route-setter Ant - READY
  ✋ Builder Ant - READY
  ✋ Watcher Ant - READY
  ✋ Scout Ant - READY
  ✋ Architect Ant - READY

🌿 Active Pheromones:
  [INIT] Build the Aether colony infrastructure (strength: 100%)

📊 Phase Progress:
  Completed: 0/10
  In Progress: 0
  Pending: 10
  Overall: 0%
  [░░░░░░░░░░░]

⚡ Resource Budgets:
  Spawns: 0/10 this phase
  Circuit Breaker Trips: 0

🧠 Memory Status:
  Working Memory: 0/200000 tokens (0%)
    - Items: 1
  Short-term Memory: 0/10 sessions
  Long-term Memory: 0 patterns

📈 Performance:
  Total Execution: 0s
  Phases Completed: 0
  Total Spawns: 0
  Success Rate: 0%

📋 Available Actions:

  /ant:plan     - Show full 10-phase roadmap
  /ant:phase 1  - View phase details
  /ant:execute 1 - Execute Phase 1
  /ant:focus    - Emit focus pheromone
```

# Example Output (Colony Not Initialized)

```
⚠️  Colony not initialized

Use /ant:init <goal> to initialize the colony

Example: /ant:init "Build a REST API"
```
</reference>

<allowed-tools>
Read
Bash
Glob
</allowed-tools>
