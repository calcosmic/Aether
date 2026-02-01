# Phase 6: Autonomous Emergence - Research

**Researched:** 2026-02-01
**Domain:** Autonomous agent spawning, capability gap detection, resource governance
**Confidence:** HIGH

## Summary

Phase 6 implements Aether's core innovation: **Worker Ants spawn Worker Ants autonomously**. This is genuinely revolutionary—no existing system has fully autonomous agent spawning where agents decide when they need help, what specialists to create, and handle full lifecycle without human intervention. Research from Ralph (AUTONOMOUS_AGENT_SPAWNING_RESEARCH.md) confirms this is first-of-its-kind territory.

The implementation follows Aether's unique prompt-based architecture (bash/jq + Claude Code Task tool), not Python. All state operations use jq with atomic writes from Phase 1. All spawning decisions are made by Worker Ants via prompt-based logic, not by orchestration code. The colony self-organizes within phase boundaries while Queen provides intention at boundaries.

**Primary recommendation:** Implement capability gap detection as prompt-based decision logic in Worker Ant commands, resource budget tracking via jq in COLONY_STATE.json, Task tool spawning with context inheritance, and circuit breakers using spawn history metadata. This creates emergence from simple local rules.

## Standard Stack

The established libraries/tools for autonomous spawning in Aether:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| **Claude Code Task tool** | built-in | Spawn subagents with inherited context | Only mechanism for agent spawning in Aether, handles context passing |
| **jq** | 1.6+ | JSON state manipulation (spawn counters, history) | Standard JSON processor for bash, already used throughout Aether |
| **bash** | 4.0+ | Spawn decision logic, resource checks | Aether's native scripting language, matches pheromone system |
| **atomic-write.sh** | existing | Spawn event tracking, state updates | Already implemented in Phase 1, proven pattern |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **file-lock.sh** | existing | Concurrent spawn prevention | Already implemented in Phase 1, prevents race conditions |
| **memory-ops.sh** | existing | Spawn outcome storage | Phase 4 memory system for meta-learning foundation |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Prompt-based decision logic | Python decision tree | Python breaks Aether's bash-native pattern, prompt-based is more flexible |
| Task tool spawning | Custom spawn function | Task tool is only supported mechanism, custom functions don't inherit context |
| JSON spawn tracking | SQLite spawn log | Database overkill for prototype, JSON sufficient and simpler |

**Installation:**
```bash
# All tools already available in standard environment
# Task tool: Built into Claude Code
# jq: brew install jq (macOS) or apt install jq (Linux)
# No additional installation needed for Phase 6
```

## Architecture Patterns

### Recommended Project Structure
```
.aether/
├── utils/
│   ├── spawn-decision.sh          # NEW: Capability gap detection
│   ├── spawn-tracker.sh           # NEW: Resource budget enforcement
│   ├── circuit-breaker.sh         # NEW: Failed spawn detection
│   ├── atomic-write.sh            # EXISTING: Use for spawn tracking
│   └── file-lock.sh               # EXISTING: Use for concurrent spawns
├── data/
│   ├── COLONY_STATE.json          # UPDATE: Add spawn_tracking, circuit_breaker
│   └── worker_ants.json           # EXISTING: Caste capabilities for gap detection
└── commands/
    └── ant/
        └── workers/
            ├── colonizer.md       # UPDATE: Add spawning decision section
            ├── route_setter.md    # UPDATE: Add spawning decision section
            ├── builder.md         # UPDATE: Add spawning decision section
            ├── watcher.md         # UPDATE: Add spawning decision section
            ├── scout.md           # UPDATE: Add spawning decision section
            └── architect.md       # UPDATE: Add spawning decision section
```

### Pattern 1: Capability Gap Detection (Prompt-Based)

**What:** Worker Ant analyzes task requirements vs own capabilities and decides whether to spawn specialist.

**When to use:** Every Worker Ant task execution. Before attempting task, check if spawn needed.

**Example (in Builder Ant prompt):**
```markdown
## Spawning Decision Logic

You are the Builder Ant. Before attempting any task, assess your capabilities:

### Step 1: Analyze Task Requirements

Given task: "{task_description}"

Extract required capabilities:
- Technical domains: [database, frontend, backend, api, security, testing, etc.]
- Frameworks: [react, django, fastapi, etc.]
- Skills: [analysis, planning, implementation, etc.]

### Step 2: Compare to Own Capabilities

Your capabilities:
- code_implementation
- command_execution
- file_operations
- testing_setup
- build_automation

### Step 3: Detect Capability Gaps

Explicit domain mismatch:
- Does task require skills outside your caste?
- Examples: "database migration" (database skill), "React component" (frontend framework)

Failure after attempts:
- If you attempted task and failed or got stuck
- Unclear path forward after reasonable effort

Pattern recognition:
- Have you seen similar tasks before that required specialists?
- Check spawn history in meta_learning

### Step 4: Calculate Spawn Decision Score

Multi-factor scoring:
```
spawn_score = (
    gap_score * 0.40 +      # Capability gap size (0-1)
    priority * 0.20 +        # Task importance (0-1)
    load * 0.15 +            # Current colony load (0-1, inverted)
    budget_remaining * 0.15 + # Spawns available (0-1)
    resources * 0.10         # System resources (0-1)
)

# Decision threshold
if spawn_score >= 0.6:
    spawn_specialist()
else:
    attempt_task()
```

### Step 5: Determine Specialist Type

Use capability mapping:
```
Required capability → Specialist caste
database → database_specialist ( Scout with database focus )
frontend → frontend_specialist ( Builder with frontend expertise )
api → api_specialist ( Route-setter with API design )
testing → test_specialist ( Watcher with testing focus )
security → security_specialist ( Watcher with security focus )
```

If no direct mapping, use semantic analysis of task description.

### Step 6: Check Resource Constraints

Before spawning, verify:
```bash
# Max 10 spawns per phase
current_spawns=$(jq -r '.resource_budgets.current_spawns' "$COLONY_STATE")
max_spawns=$(jq -r '.resource_budgets.max_spawns_per_phase' "$COLONY_STATE")
if [ "$current_spawns" -ge "$max_spawns" ]; then
    echo "Spawn budget exhausted"
    return 1
fi

# Max spawn depth 3
spawn_depth=$(jq -r '.spawn_tracking.depth' "$COLONY_STATE")
max_depth=$(jq -r '.resource_budgets.max_spawn_depth' "$COLONY_STATE")
if [ "$spawn_depth" -ge "$max_depth" ]; then
    echo "Max spawn depth reached"
    return 1
fi

# Circuit breaker not triggered
circuit_breaker_trips=$(jq -r '.resource_budgets.circuit_breaker_trips' "$COLONY_STATE")
if [ "$circuit_breaker_trips" -ge 3 ]; then
    echo "Circuit breaker triggered"
    return 1
fi
```

### Step 7: Spawn Specialist via Task Tool

```
Task: {specialist_type} Specialist

You are a {specialist_type} specialist spawned by {parent_caste} Ant.

PARENT CONTEXT:
- Goal: {goal from INIT pheromone}
- Current task: {task_description}
- Active pheromones: {from pheromones.json}
- Working memory: {relevant items from memory.json}
- Spawn depth: {current_depth + 1}

YOUR SPECIALIZATION:
{specialization_details}

Execute the specialized task. Report outcome to parent.
```
```

**Key insights from research:**
- Prompt-based decision logic is more flexible than hard-coded rules
- Multi-factor scoring prevents unnecessary spawns
- Resource constraints checked BEFORE spawning (critical safeguard)
- Semantic analysis fallback handles novel capability gaps

### Pattern 2: Resource Budget Tracking (jq + atomic writes)

**What:** Track spawn count, depth, and circuit breaker state in COLONY_STATE.json.

**When to use:** Before every spawn decision (check limits), after every spawn (increment counters).

**Example:**
```bash
#!/bin/bash
# spawn-tracker.sh - Track spawn resources

source .aether/utils/atomic-write.sh

COLONY_STATE=".aether/data/COLONY_STATE.json"

# Check if spawn is allowed
can_spawn() {
    local current_spawns=$(jq -r '.resource_budgets.current_spawns' "$COLONY_STATE")
    local max_spawns=$(jq -r '.resource_budgets.max_spawns_per_phase' "$COLONY_STATE")
    local spawn_depth=$(jq -r '.spawn_tracking.depth // 0' "$COLONY_STATE")
    local max_depth=$(jq -r '.resource_budgets.max_spawn_depth' "$COLONY_STATE")
    local circuit_trips=$(jq -r '.resource_budgets.circuit_breaker_trips' "$COLONY_STATE")

    # Check spawn budget
    if [ "$current_spawns" -ge "$max_spawns" ]; then
        echo "Spawn budget exhausted: $current_spawns/$max_spawns"
        return 1
    fi

    # Check spawn depth
    if [ "$spawn_depth" -ge "$max_depth" ]; then
        echo "Max spawn depth reached: $spawn_depth/$max_depth"
        return 1
    fi

    # Check circuit breaker
    if [ "$circuit_trips" -ge 3 ]; then
        echo "Circuit breaker triggered: $circuit_trips trips"
        return 1
    fi

    return 0
}

# Record spawn event
record_spawn() {
    local parent_caste="$1"
    local specialist_type="$2"
    local task_context="$3"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local spawn_id="spawn_$(date +%s)"

    # Increment spawn counter
    jq '
        .resource_budgets.current_spawns += 1 |
        .spawn_tracking.depth += 1 |
        .spawn_tracking.total_spawns += 1 |
        .spawn_tracking.spawn_history += [{
            "id": "'$spawn_id'",
            "parent": "'$parent_caste'",
            "specialist": "'$specialist_type'",
            "task": "'$task_context'",
            "timestamp": "'$timestamp'",
            "depth": (.spawn_tracking.depth | tonumber),
            "outcome": "pending"
        }]
    ' "$COLONY_STATE" > /tmp/spawn_record.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/spawn_record.tmp
    rm -f /tmp/spawn_record.tmp

    echo "$spawn_id"
}

# Record spawn outcome
record_outcome() {
    local spawn_id="$1"
    local outcome="$2"  # success | failure
    local notes="${3:-}"

    # Update spawn history
    jq --arg id "$spawn_id" \
       --arg outcome "$outcome" \
       --arg notes "$notes" \
       --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
       '
       .spawn_tracking.spawn_history |= map(
           if .id == $id then
               .outcome = $outcome |
               .completed_at = $timestamp |
               .notes = $notes
           else
               .
           end
       )
       ' "$COLONY_STATE" > /tmp/outcome.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/outcome.tmp
    rm -f /tmp/outcome.tmp

    # Update performance metrics
    if [ "$outcome" = "success" ]; then
        jq '.performance_metrics.successful_spawns += 1' "$COLONY_STATE" > /tmp/metrics.tmp
        atomic_write_from_file "$COLONY_STATE" /tmp/metrics.tmp
        rm -f /tmp/metrics.tmp
    else
        jq '.performance_metrics.failed_spawns += 1' "$COLONY_STATE" > /tmp/metrics.tmp
        atomic_write_from_file "$COLONY_STATE" /tmp/metrics.tmp
        rm -f /tmp/metrics.tmp
    fi

    # Decrement spawn depth on completion
    jq '.spawn_tracking.depth |= max(0; . - 1)' "$COLONY_STATE" > /tmp/depth.tmp
    atomic_write_from_file "$COLONY_STATE" /tmp/depth.tmp
    rm -f /tmp/depth.tmp
}

export -f can_spawn record_spawn record_outcome
```

**Key insights:**
- All spawn checks MUST happen before Task tool call
- Atomic writes prevent race conditions in spawn counter
- Depth tracking prevents infinite spawn chains
- Spawn history provides debugging capability and meta-learning data

### Pattern 3: Circuit Breaker Implementation

**What:** Detect consecutive failed spawns of same specialist type and trigger cooldown.

**When to use:** After every spawn failure. Check if same specialist failed 3 times recently.

**Example:**
```bash
#!/bin/bash
# circuit-breaker.sh - Prevent repeated failed spawns

source .aether/utils/atomic-write.sh

COLONY_STATE=".aether/data/COLONY_STATE.json"

# Check circuit breaker status
check_circuit_breaker() {
    local specialist_type="$1"

    # Get circuit breaker state
    local trips=$(jq -r '.resource_budgets.circuit_breaker_trips' "$COLONY_STATE")
    local cooldown_until=$(jq -r '.resource_budgets.circuit_breaker_cooldown_until // ""' "$COLONY_STATE")

    # Check if in cooldown
    if [ -n "$cooldown_until" ]; then
        local now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        if [ "$cooldown_until" > "$now" ]; then
            echo "Circuit breaker active until $cooldown_until"
            return 1
        fi
    fi

    # Check trip count
    if [ "$trips" -ge 3 ]; then
        echo "Circuit breaker triggered: $trips failures"
        return 1
    fi

    return 0
}

# Record spawn failure (trips circuit breaker)
record_spawn_failure() {
    local specialist_type="$1"
    local spawn_id="$2"
    local failure_reason="$3"

    # Count recent failures of this specialist
    local recent_failures=$(jq -r "
        .spawn_tracking.spawn_history |
        map(select(.specialist == \"$specialist_type\" and .outcome == \"failure\")) |
        length
    " "$COLONY_STATE")

    # Increment trip count
    jq --arg specialist "$specialist_type" \
       --argjson failures "$recent_failures" \
       --arg reason "$failure_reason" \
       '
       .resource_budgets.circuit_breaker_trips += 1 |
       .spawn_tracking.failed_specialist_types += [$specialist] |
       .spawn_tracking.circuit_breaker_history += [{
           "specialist": $specialist,
           "failures": ($failures + 1),
           "reason": $reason,
           "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
       }]
       ' "$COLONY_STATE" > /tmp/circuit_trip.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/circuit_trip.tmp
    rm -f /tmp/circuit_trip.tmp

    # If 3 failures, trigger cooldown
    if [ "$((recent_failures + 1))" -ge 3 ]; then
        trigger_circuit_breaker_cooldown "$specialist_type"
    fi
}

# Trigger cooldown period
trigger_circuit_breaker_cooldown() {
    local specialist_type="$1"

    # 30-minute cooldown
    local cooldown_until=$(date -u -d "+30 minutes" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -v+30M +"%Y-%m-%dT%H:%M:%SZ")

    jq --arg specialist "$specialist_type" \
       --arg until "$cooldown_until" \
       '
       .resource_budgets.circuit_breaker_cooldown_until = $until |
       .spawn_tracking.cooldown_specialists += [$specialist]
       ' "$COLONY_STATE" > /tmp/cooldown.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/cooldown.tmp
    rm -f /tmp/cooldown.tmp

    echo "⚠️  Circuit breaker triggered: $specialist_type cooldown until $cooldown_until"
}

# Reset circuit breaker (after success or manual intervention)
reset_circuit_breaker() {
    jq '
        .resource_budgets.circuit_breaker_trips = 0 |
        .resource_budgets.circuit_breaker_cooldown_until = null |
        .spawn_tracking.failed_specialist_types = []
    ' "$COLONY_STATE" > /tmp/reset.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/reset.tmp
    rm -f /tmp/reset.tmp

    echo "✅ Circuit breaker reset"
}

export -f check_circuit_breaker record_spawn_failure trigger_circuit_breaker_cooldown reset_circuit_breaker
```

**Key insights from research:**
- Circuit breaker prevents infinite retry loops on impossible tasks
- 3-failure threshold balances persistence vs waste
- 30-minute cooldown allows colony to try different approaches
- Same-specialist cache prevents spawning identical specialists

### Pattern 4: Context Inheritance for Spawned Specialists

**What:** Spawned specialist inherits parent's goal, pheromones, working memory, and constraints.

**When to use:** Every Task tool spawn call. Pass context via prompt template.

**Example (Task tool call in Builder Ant):**
```markdown
Task: {specialist_type} Specialist

## Inherited Context

### Queen's Goal
```
{from COLONY_STATE.json: queen_intention.goal}
```

### Active Pheromone Signals
```
{from pheromones.json: active_pheromones}
- FOCUS: {context} (strength: {strength})
- REDIRECT: {context} (strength: {strength})
```

### Working Memory (Recent Context)
```
{from memory.json: working_memory items, relevance-sorted}
- {item.content} (relevance: {item.relevance})
```

### Constraints (from REDIRECT pheromones)
```
{from memory.json: short-term patterns, type=constraint}
- {pattern.content}
```

### Parent Context
```
Parent caste: {current_caste}
Parent task: {task_description}
Spawn depth: {current_depth + 1}/3
```

## Your Specialization

You are a {specialist_type} specialist with expertise in:
- {capability_1}
- {capability_2}
- {capability_3}

Your parent ({parent_caste} Ant) detected a capability gap and spawned you.

## Your Task

{task_description}

## Execution Instructions

1. Use your specialized expertise to complete the task
2. Respect inherited constraints (REDIRECT pheromones)
3. Follow active focus areas (FOCUS pheromones)
4. Add findings to working memory
5. Report outcome to parent

## Success Criteria

{success_criteria}

Complete the task. Report outcome as:
- ✓ SUCCESS: {what was accomplished}
- ✗ FAILURE: {what went wrong, what was tried}
```

**Key insights:**
- Context inheritance makes specialists immediately effective
- Pheromone signals maintain colony coordination
- Working memory provides recent context
- Constraints prevent harmful actions
- Spawn depth prevents runaway chains

### Pattern 5: Spawn Outcome Tracking for Meta-Learning

**What:** Record spawn success/failure for Phase 8 (Bayesian confidence scoring).

**When to use:** After every spawn completion. Store in meta_learning section of COLONY_STATE.json.

**Example:**
```bash
#!/bin/bash
# spawn-outcome-tracker.sh - Record outcomes for meta-learning

source .aether/utils/atomic-write.sh

COLONY_STATE=".aether/data/COLONY_STATE.json"

# Record successful spawn
record_successful_spawn() {
    local specialist_type="$1"
    local task_type="$2"
    local spawn_id="$3"

    # Update specialist confidence
    jq --arg specialist "$specialist_type" \
       --arg task "$task_type" \
       '
       .meta_learning.specialist_confidence[$specialist][$task] |= (
           . // 0.5 |
           . + 0.1 |
           min(1.0)
       ) |
       .meta_learning.spawn_outcomes += [{
           "spawn_id": $spawn_id,
       "specialist": $specialist,
       "task_type": $task,
       "outcome": "success",
       "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
   }] |
   .meta_learning.last_updated = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
   ' "$COLONY_STATE" > /tmp/success.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/success.tmp
    rm -f /tmp/success.tmp
}

# Record failed spawn
record_failed_spawn() {
    local specialist_type="$1"
    local task_type="$2"
    local spawn_id="$3"
    local failure_reason="$4"

    # Decrease specialist confidence
    jq --arg specialist "$specialist_type" \
       --arg task "$task_type" \
       --arg reason "$failure_reason" \
       '
       .meta_learning.specialist_confidence[$specialist][$task] |= (
           . // 0.5 |
           . - 0.15 |
           max(0.0)
       ) |
       .meta_learning.spawn_outcomes += [{
           "spawn_id": $spawn_id,
           "specialist": $specialist,
           "task_type": $task,
           "outcome": "failure",
           "reason": $reason,
           "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
   }] |
   .meta_learning.last_updated = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
   ' "$COLONY_STATE" > /tmp/failure.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/failure.tmp
    rm -f /tmp/failure.tmp
}

# Get specialist confidence for task type
get_specialist_confidence() {
    local specialist_type="$1"
    local task_type="$2"

    local confidence=$(jq -r "
        .meta_learning.specialist_confidence[\"$specialist_type\"][\"$task_type\"] // 0.5
    " "$COLONY_STATE")

    echo "$confidence"
}

export -f record_successful_spawn record_failed_spawn get_specialist_confidence
```

**Key insights:**
- Confidence scores start at 0.5 (neutral)
- Success: +0.1 confidence (max 1.0)
- Failure: -0.15 confidence (min 0.0, asymmetric penalty)
- This data feeds Phase 8 Bayesian updating
- Specialist selection uses confidence scores in spawn decisions

### Anti-Patterns to Avoid

- **Hard-coded spawn rules**: Never hard-code "if task contains X, spawn Y". Use prompt-based semantic analysis.
- **Skipping resource checks**: Never spawn without verifying budget, depth, circuit breaker. Leads to infinite loops.
- **Direct state mutation**: Never modify COLONY_STATE.json directly. Always use atomic-write.sh.
- **Orphaned spawns**: Never spawn without recording spawn_id. Lost spawns can't be tracked.
- **Context-less spawns**: Never spawn specialist without passing inherited context. Inefficient specialists.
- **Ignoring circuit breaker**: Never bypass circuit breaker even if spawn budget available. Repeated failures waste resources.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Spawn counter management | Manual integer variables | **jq + atomic-write.sh** | Race conditions during concurrent spawns, counter corruption |
| JSON state updates | Custom bash string manipulation | **jq** | Edge cases in JSON (nested objects, unicode, escaping) break string parsing |
| Concurrent spawn prevention | Custom lock files | **file-lock.sh** | Stale locks, PID tracking, cleanup on exit are non-trivial |
| Spawn history tracking | Simple log file | **COLONY_STATE.json array** | History needed for debugging, circuit breaker, meta-learning |
| Capability gap detection | If/else rules | **Semantic analysis in prompt** | Novel gaps don't match hard-coded rules, prompt is more flexible |

**Key insight:** The existing Aether patterns (atomic-write.sh, file-lock.sh, jq) already solve the hard problems. Reuse them for spawn tracking. Don't reinvent concurrent state management.

## Common Pitfalls

### Pitfall 1: Infinite Spawn Loops

**What goes wrong:** Worker spawns specialist, specialist spawns another, chain never ends. Colony spawns hundreds of agents, system crashes.

**Why it happens:** No spawn depth limit. No circuit breaker. Specialists keep spawning when stuck.

**How to avoid:**
1. **Max spawn depth 3**: Track depth in COLONY_STATE.json, increment on spawn, decrement on completion.
2. **Circuit breaker**: 3 failed spawns → cooldown.
3. **Same-specialist cache**: Don't spawn same specialist type twice for same task.

**Prevention strategy:**
```bash
# Before every spawn
check_spawn_depth() {
    local current_depth=$(jq -r '.spawn_tracking.depth // 0' "$COLONY_STATE")
    local max_depth=$(jq -r '.resource_budgets.max_spawn_depth' "$COLONY_STATE")

    if [ "$current_depth" -ge "$max_depth" ]; then
        echo "❌ Max spawn depth reached: $current_depth/$max_depth"
        echo "   Task must be handled at current level or reported to parent"
        return 1
    fi

    return 0
}
```

**Warning signs:** Spawn depth > 5. Same specialist spawned repeatedly. Colony state grows rapidly.

### Pitfall 2: Spawn Budget Exhaustion Early in Phase

**What goes wrong:** First few tasks spawn all 10 specialists. Later tasks have no budget. Phase stalls.

**Why it happens:** Spawn threshold too low (spawns too eagerly). No prioritization of spawns.

**How to avoid:**
1. **Spawn decision threshold 0.6**: Only spawn when multi-factor score >= 0.6.
2. **Hybrid timing**: Known gaps spawn immediately, ambiguous cases attempt first.
3. **Budget awareness**: Check remaining budget in spawn decision scoring.

**Prevention strategy:**
```markdown
### Spawn Decision with Budget Awareness

```bash
# Calculate spawn budget factor
current_spawns=$(jq -r '.resource_budgets.current_spawns' "$COLONY_STATE")
max_spawns=$(jq -r '.resource_budgets.max_spawns_per_phase' "$COLONY_STATE")
budget_remaining=$((max_spawns - current_spawns))
budget_factor=$(echo "scale=2; $budget_remaining / $max_spawns" | bc)

# Include in spawn score
spawn_score = (
    gap_score * 0.40 +
    priority * 0.20 +
    load * 0.15 +
    budget_factor * 0.15 +  # Uses remaining budget
    resources * 0.10
)
```

If budget < 3 spawns, increase threshold to 0.8 (only spawn critical gaps).
```

**Warning signs:** Spawn budget exhausted before 50% of tasks complete.

### Pitfall 3: Context Loss Between Parent and Child

**What goes wrong:** Spawned specialist doesn't know goal, pheromones, or constraints. Works on wrong thing, violates constraints.

**Why it happens:** Task tool call doesn't pass inherited context. Specialist prompt missing context section.

**How to avoid:**
1. **Context inheritance template**: All Task tool calls include inherited context section.
2. **Pheromone signals**: Pass active FOCUS, REDIRECT signals to specialist.
3. **Working memory**: Include relevant items (sorted by relevance).
4. **Constraints**: Explicitly pass REDIRECT-derived constraints.

**Prevention strategy:**
```markdown
## Template for All Task Tool Spawns

```
Task: {specialist_type} Specialist

## Inherited Context
[ALL sections populated from colony state]

### Queen's Goal
{goal from COLONY_STATE.json}

### Active Pheromones
{active_pheromones from pheromones.json}

### Constraints
{constraints from REDIRECT pheromones}

### Working Memory
{relevant items from memory.json}
```
```

**Warning signs:** Specialist asks "what's the goal?" or violates known constraints.

### Pitfall 4: Circuit Breaker Never Resets

**What goes wrong:** Circuit breaker trips after 3 failures, never resets. Colony can't spawn specialists even after cooldown.

**Why it happens:** No reset mechanism. Or cooldown never expires (timestamp bug).

**How to avoid:**
1. **Auto-reset after cooldown**: When cooldown expires, reset trip count to 0.
2. **Manual reset command**: /ant:reset-circuit-breaker for Queen intervention.
3. **Success resets circuit breaker**: Successful spawn resets trip count.

**Prevention strategy:**
```bash
# On colony initialization, check cooldown
check_circuit_breaker_cooldown() {
    local cooldown_until=$(jq -r '.resource_budgets.circuit_breaker_cooldown_until // ""' "$COLONY_STATE")

    if [ -n "$cooldown_until" ]; then
        local now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

        # Cooldown expired, reset circuit breaker
        if [ "$cooldown_until" \< "$now" ]; then
            reset_circuit_breaker
            echo "✅ Circuit breaker cooldown expired, reset"
        fi
    fi
}
```

**Warning signs:** Circuit breaker trips persist for hours. No spawns despite budget available.

### Pitfall 5: Spawn Outcome Not Recorded

**What goes wrong:** Spawn succeeds/fails but outcome never recorded. Meta-learning has no data. Confidence scores never update.

**Why it happens:** Specialist doesn't report outcome. Parent doesn't record to state. Outcome tracking forgotten.

**How to avoid:**
1. **Mandatory outcome reporting**: All Task tool spawns require outcome report.
2. **Parent records outcome**: Parent Ant calls record_outcome() after child completes.
3. **Automated tracking**: record_spawn() sets outcome to "pending", record_outcome() updates to "success"/"failure".

**Prevention strategy:**
```markdown
## Specialist Outcome Reporting Template

All spawned specialists MUST report outcome:

```
## Outcome Report

Spawn ID: {spawn_id}
Specialist: {specialist_type}
Task: {task_description}

Result: [✓ SUCCESS | ✗ FAILURE]

What was accomplished:
{for success: what was done}

What went wrong:
{for failure: error, what was tried}

Recommendations:
{for parent: what to do next}
```

Parent Ant: Use this report to call record_outcome().
```

**Warning signs:** meta_learning.spawn_outcomes array empty. specialist_confidence never updates.

## Code Examples

Verified patterns from official sources:

### Capability Gap Detection (Builder Ant Prompt Section)

```markdown
## Capability Gap Detection

Before attempting any task, assess whether you need specialist support.

### Step 1: Extract Task Requirements

Given: "{task_description}"

Required capabilities:
- Technical: [database, frontend, backend, api, security, testing, performance, devops]
- Frameworks: [react, vue, django, fastapi, etc.]
- Skills: [analysis, planning, implementation, validation]

### Step 2: Compare to Your Capabilities

Your capabilities (Builder Ant):
- code_implementation
- command_execution
- file_operations
- testing_setup
- build_automation

### Step 3: Identify Gaps

Explicit mismatch examples:
- "database schema migration" → Requires database expertise (you lack)
- "React component library" → Requires frontend specialization (you lack)
- "API rate limiting" → Requires API design expertise (you lack)

### Step 4: Calculate Spawn Score

```bash
gap_score=0.8        # Large capability gap
priority=0.9         # High priority task
load=0.3             # Colony lightly loaded
budget_remaining=0.7 # 7/10 spawns available
resources=0.8        # System resources available

spawn_score = (
    0.8 * 0.40 +     # gap_score
    0.9 * 0.20 +     # priority
    0.3 * 0.15 +     # load (inverted)
    0.7 * 0.15 +     # budget_remaining
    0.8 * 0.10       # resources
) = 0.68

# 0.68 >= 0.6 threshold → SPAWN
```

### Step 5: Map Gap to Specialist

Capability gap → Specialist:
- database → database_specialist (Scout with database expertise)
- react → frontend_specialist (Builder with React specialization)
- api → api_specialist (Route-setter with API design focus)
- testing → test_specialist (Watcher with testing specialization)

### Step 6: Verify Resource Constraints

```bash
# Max 10 spawns per phase
current_spawns=3, max_spawns=10 ✓

# Max spawn depth 3
current_depth=1, max_depth=3 ✓

# Circuit breaker
trips=0, threshold=3 ✓

# All constraints satisfied → PROCEED
```

### Step 7: Spawn Specialist

Use Task tool with inherited context (see Context Inheritance pattern).
```

### Resource Budget Enforcement

```bash
#!/bin/bash
# spawn-tracker.sh

source .aether/utils/atomic-write.sh
source .aether/utils/file-lock.sh

COLONY_STATE=".aether/data/COLONY_STATE.json"

# Check if spawn allowed
can_spawn() {
    # Acquire lock to prevent concurrent spawn decisions
    if ! acquire_lock "$COLONY_STATE"; then
        echo "Failed to acquire lock for spawn check"
        return 1
    fi

    local current_spawns=$(jq -r '.resource_budgets.current_spawns' "$COLONY_STATE")
    local max_spawns=$(jq -r '.resource_budgets.max_spawns_per_phase' "$COLONY_STATE")
    local spawn_depth=$(jq -r '.spawn_tracking.depth // 0' "$COLONY_STATE")
    local max_depth=$(jq -r '.resource_budgets.max_spawn_depth' "$COLONY_STATE")
    local circuit_trips=$(jq -r '.resource_budgets.circuit_breaker_trips' "$COLONY_STATE")

    # Check spawn budget
    if [ "$current_spawns" -ge "$max_spawns" ]; then
        echo "❌ Spawn budget exhausted: $current_spawns/$max_spawns"
        release_lock
        return 1
    fi

    # Check spawn depth
    if [ "$spawn_depth" -ge "$max_depth" ]; then
        echo "❌ Max spawn depth reached: $spawn_depth/$max_depth"
        release_lock
        return 1
    fi

    # Check circuit breaker
    if [ "$circuit_trips" -ge 3 ]; then
        echo "❌ Circuit breaker triggered: $circuit_trips trips"
        release_lock
        return 1
    fi

    release_lock
    return 0
}

# Record spawn event
record_spawn() {
    local parent_caste="$1"
    local specialist_type="$2"
    local task_context="$3"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local spawn_id="spawn_$(date +%s)"

    # Increment spawn counter and depth
    jq --arg id "$spawn_id" \
       --arg parent "$parent_caste" \
       --arg specialist "$specialist_type" \
       --arg task "$task_context" \
       --arg timestamp "$timestamp" \
       '
        .resource_budgets.current_spawns += 1 |
        .spawn_tracking.depth += 1 |
        .spawn_tracking.total_spawns += 1 |
        .spawn_tracking.spawn_history += [{
            "id": $id,
            "parent": $parent,
            "specialist": $specialist,
            "task": $task,
            "timestamp": $timestamp,
            "depth": (.spawn_tracking.depth | tonumber),
            "outcome": "pending"
        }]
       ' "$COLONY_STATE" > /tmp/spawn_record.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/spawn_record.tmp
    rm -f /tmp/spawn_record.tmp

    echo "$spawn_id"
}

# Record spawn outcome
record_outcome() {
    local spawn_id="$1"
    local outcome="$2"  # success | failure
    local notes="${3:-}"

    # Update spawn history entry
    jq --arg id "$spawn_id" \
       --arg outcome "$outcome" \
       --arg notes "$notes" \
       --arg completed_at "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
       '
       .spawn_tracking.spawn_history |= map(
           if .id == $id then
               .outcome = $outcome |
               .completed_at = $completed_at |
               .notes = $notes
           else
               .
           end
       ) |
       if $outcome == "success" then
           .performance_metrics.successful_spawns += 1
       else
           .performance_metrics.failed_spawns += 1
       end |
       .spawn_tracking.depth |= max(0; . - 1)
       ' "$COLONY_STATE" > /tmp/outcome.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/outcome.tmp
    rm -f /tmp/outcome.tmp
}

export -f can_spawn record_spawn record_outcome
```

### Circuit Breaker Implementation

```bash
#!/bin/bash
# circuit-breaker.sh

source .aether/utils/atomic-write.sh

COLONY_STATE=".aether/data/COLONY_STATE.json"

# Check if circuit breaker allows spawn
check_circuit_breaker() {
    local specialist_type="$1"

    local trips=$(jq -r '.resource_budgets.circuit_breaker_trips' "$COLONY_STATE")
    local cooldown_until=$(jq -r '.resource_budgets.circuit_breaker_cooldown_until // ""' "$COLONY_STATE")

    # Check cooldown
    if [ -n "$cooldown_until" ]; then
        local now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        if [ "$cooldown_until" \> "$now" ]; then
            echo "❌ Circuit breaker cooldown active until $cooldown_until"
            return 1
        fi

        # Cooldown expired, reset
        reset_circuit_breaker
    fi

    # Check trip count
    if [ "$trips" -ge 3 ]; then
        echo "❌ Circuit breaker triggered: $trips failures"
        return 1
    fi

    return 0
}

# Record spawn failure (trips circuit breaker)
record_spawn_failure() {
    local specialist_type="$1"
    local spawn_id="$2"
    local failure_reason="$3"

    # Count recent failures of this specialist
    local recent_failures=$(jq -r "
        .spawn_tracking.spawn_history |
        map(select(.specialist == \"$specialist_type\" and .outcome == \"failure\")) |
        length
    " "$COLONY_STATE")

    # Increment trip count
    jq --arg specialist "$specialist_type" \
       --arg reason "$failure_reason" \
       '
       .resource_budgets.circuit_breaker_trips += 1 |
       .spawn_tracking.failed_specialist_types += [$specialist] |
       .spawn_tracking.circuit_breaker_history += [{
           "specialist": $specialist,
           "failures": ($recent_failures + 1),
           "reason": $reason,
           "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
       }]
       ' "$COLONY_STATE" > /tmp/circuit_trip.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/circuit_trip.tmp
    rm -f /tmp/circuit_trip.tmp

    # Trigger cooldown if 3 failures
    if [ "$((recent_failures + 1))" -ge 3 ]; then
        trigger_circuit_breaker_cooldown "$specialist_type"
    fi
}

# Trigger 30-minute cooldown
trigger_circuit_breaker_cooldown() {
    local specialist_type="$1"

    # Calculate cooldown timestamp
    if date -v+30M >/dev/null 2>&1; then
        # macOS
        local cooldown_until=$(date -u -v+30M +"%Y-%m-%dT%H:%M:%SZ")
    else
        # Linux
        local cooldown_until=$(date -u -d "+30 minutes" +"%Y-%m-%dT%H:%M:%SZ")
    fi

    jq --arg specialist "$specialist_type" \
       --arg until "$cooldown_until" \
       '
       .resource_budgets.circuit_breaker_cooldown_until = $until |
       .spawn_tracking.cooldown_specialists += [$specialist]
       ' "$COLONY_STATE" > /tmp/cooldown.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/cooldown.tmp
    rm -f /tmp/cooldown.tmp

    echo "⚠️  Circuit breaker: $specialist_type cooldown until $cooldown_until"
}

# Reset circuit breaker
reset_circuit_breaker() {
    jq '
        .resource_budgets.circuit_breaker_trips = 0 |
        .resource_budgets.circuit_breaker_cooldown_until = null |
        .spawn_tracking.failed_specialist_types = []
    ' "$COLONY_STATE" > /tmp/reset.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/reset.tmp
    rm -f /tmp/reset.tmp

    echo "✅ Circuit breaker reset"
}

export -f check_circuit_breaker record_spawn_failure trigger_circuit_breaker_cooldown reset_circuit_breaker
```

### Spawn Outcome Tracking (Meta-Learning Foundation)

```bash
#!/bin/bash
# spawn-outcome-tracker.sh

source .aether/utils/atomic-write.sh

COLONY_STATE=".aether/data/COLONY_STATE.json"

# Record successful spawn
record_successful_spawn() {
    local specialist_type="$1"
    local task_type="$2"
    local spawn_id="$3"

    # Increment confidence score
    jq --arg specialist "$specialist_type" \
       --arg task "$task_type" \
       '
       .meta_learning.specialist_confidence[$specialist][$task] |= (
           . // 0.5 |
           . + 0.1 |
           min(1.0)
       ) |
       .meta_learning.spawn_outcomes += [{
           "spawn_id": $spawn_id,
           "specialist": $specialist,
           "task_type": $task,
           "outcome": "success",
           "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
       }] |
       .meta_learning.last_updated = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
       ' "$COLONY_STATE" > /tmp/success.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/success.tmp
    rm -f /tmp/success.tmp
}

# Record failed spawn
record_failed_spawn() {
    local specialist_type="$1"
    local task_type="$2"
    local spawn_id="$3"
    local failure_reason="$4"

    # Decrement confidence score (asymmetric penalty)
    jq --arg specialist "$specialist_type" \
       --arg task "$task_type" \
       --arg reason "$failure_reason" \
       '
       .meta_learning.specialist_confidence[$specialist][$task] |= (
           . // 0.5 |
           . - 0.15 |
           max(0.0)
       ) |
       .meta_learning.spawn_outcomes += [{
           "spawn_id": $spawn_id,
           "specialist": $specialist,
           "task_type": $task,
           "outcome": "failure",
           "reason": $reason,
           "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
       }] |
       .meta_learning.last_updated = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
       ' "$COLONY_STATE" > /tmp/failure.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/failure.tmp
    rm -f /tmp/failure.tmp
}

# Get confidence score for specialist-task pairing
get_specialist_confidence() {
    local specialist_type="$1"
    local task_type="$2"

    local confidence=$(jq -r "
        .meta_learning.specialist_confidence[\"$specialist_type\"][\"$task_type\"] // 0.5
    " "$COLONY_STATE")

    echo "$confidence"
}

export -f record_successful_spawn record_failed_spawn get_specialist_confidence
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Human-defined agents | Autonomous spawning decisions | 2026 (TheAgentics, Aether) | Agents self-organize, no human orchestration needed |
| Hard-coded spawn rules | Prompt-based semantic analysis | 2026 (LLM prompt engineering) | Flexible gap detection, handles novel scenarios |
| No resource governance | Spawning budgets + circuit breakers | 2025 (self-replicating AI research) | Prevents infinite loops, production-safe |
| No learning from spawns | Meta-learning with confidence scores | 2026 (Bayesian updating research) | Colony improves specialist selection over time |

**Deprecated/outdated:**
- **Human orchestration**: Replaced by autonomous spawning. Workers spawn Workers without Queen approval.
- **Hard-coded specialist mappings**: Replaced by semantic analysis. Keyword table exists but prompt-based fallback handles novel gaps.
- **Unbounded spawning**: Replaced by resource budgets (max 10 per phase, depth 3). Prevents resource exhaustion.
- **No spawn outcome tracking**: Replaced by meta-learning foundation. Phase 8 will use this for Bayesian updating.

## Open Questions

Things that couldn't be fully resolved:

1. **Spawn decision threshold calibration**
   - What we know: Multi-factor scoring with 0.6 threshold is recommended. Research suggests adaptive thresholds.
   - What's unclear: Should threshold adjust based on remaining budget? Should different castes have different thresholds?
   - Recommendation: Start with 0.6 threshold for all castes. Monitor spawn patterns in early phases. Adjust if over-spawning or under-spawning. Research adaptive threshold patterns from [2026 Agentic Coding Trends Report](https://resources.anthropic.com/hubfs/2026%2520Agentic%2520Coding%2520Trends%2520Report.pdf?hsLang=en).

2. **Specialist type granularity**
   - What we know: 6 base castes exist. Capability mapping suggests specialist subtypes (database_specialist, frontend_specialist, etc.).
   - What's unclear: Should we create dedicated specialist prompt files? Or use caste prompts with specialization parameters?
   - Recommendation: Start with caste-based specialization (e.g., "Scout Ant with database expertise"). If patterns emerge, create dedicated specialist prompts in Phase 8 after meta-learning reveals high-value specialist types.

3. **Spawn timeout handling**
   - What we know: Specialists inherit context and execute tasks. Task tool manages execution.
   - What's unclear: What happens if specialist never reports outcome? How long to wait before marking as failed?
   - Recommendation: Task tool has built-in timeout. If spawn exceeds timeout, record as failure and trigger circuit breaker. Monitor timeout patterns—may indicate capability gaps are too broad.

4. **Cross-phase spawn tracking**
   - What we know: Max 10 spawns per phase enforced in resource_budgets.current_spawns.
   - What's unclear: Should spawns reset at phase boundary? Or accumulate across phases?
   - Recommendation: Reset spawn counter at phase boundary (start of each phase). Each phase is independent execution context. Track total_spawns lifetime in performance_metrics for analytics.

5. **Semantic analysis implementation**
   - What we know: Prompt-based semantic analysis handles novel capability gaps.
   - What's unclear: How accurate is Claude at detecting gaps? False positive rate? False negative rate?
   - Recommendation: Trust Claude's semantic analysis for Phase 6. Monitor spawn outcomes in meta_learning. If high false positive rate (unnecessary spawns), increase threshold to 0.7. If high false negative rate (missed gaps), decrease to 0.5.

## Sources

### Primary (HIGH confidence)
- **Ralph Research (AUTONOMOUS_AGENT_SPAWNING_RESEARCH.md)** - Comprehensive research on autonomous spawning, capability gap detection, circuit breakers, resource governance
- **Existing Aether Architecture** - COLONY_STATE.json schema, worker_ants.json caste definitions, atomic-write.sh (Phase 1), pheromone system (Phase 3), memory system (Phase 4), state machine (Phase 5)
- **Claude Code Task tool** - Official mechanism for spawning subagents with context inheritance

### Secondary (MEDIUM confidence)
- [2026 Agentic Coding Trends Report](https://resources.anthropic.com/hubfs/2026%2520Agentic%2520Coding%2520Trends%2520Report.pdf?hsLang=en) - Official Anthropic report on multi-agent trends, deployment gap (66% test, 11% produce)
- [TheAgentics: Self-Replicating AI Agents](https://theagentics.co/insights/self-replicating-ai-agents-the-rise-of-ai-that-builds-ai) - Safeguards and kill switches for autonomous spawning
- [Claude Code Sub-Agent Coordination](https://www.linkedin.com/pulse/claude-code-sub-agent-coordination-rich-miller-9jiuc) - Task tool patterns and agent orchestration
- [Multi-Agent Models of Organizational Intelligence](https://arxiv.org/pdf/2601.14351) - Specialized agent teams (planners, executors, critics, experts)
- [2026 Technology Trend: Multi-Agent AI Systems](https://www.iankhan.com/2026-technology-trend-2-multi-agent-ai-systems-become-the-default-for-complex-work/) - Multi-agent systems becoming operational default

### Tertiary (LOW confidence)
- [Feature Request: Custom Subagent Support](https://github.com/anthropics/claude-code/issues/19276) - Task tool extensibility discussion (verified with official docs)
- [Spring AI Agentic Patterns: Subagent Orchestration](https://spring.io/blog/2026/01/27/spring-ai-agentic-patterns-4-task-subagents) - Tool inheritance patterns (verified with practice)
- [7 Enterprise AI Agent Trends for 2026](https://beam.ai/ar/agentic-insights/enterprise-ai-agent-trends-2026) - Autonomous operation requirements (verified with deployment research)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Based on existing Aether patterns (atomic-write.sh, jq, bash) and Claude Code Task tool
- Architecture: HIGH - Prompt-based decision logic matches Aether philosophy, resource governance patterns from Ralph research
- Pitfalls: HIGH - Based on distributed systems research (infinite loops, resource exhaustion, circuit breakers)
- Code examples: HIGH - Verified against existing Aether patterns and bash/jq best practices

**Research date:** 2026-02-01
**Valid until:** 2026-03-01 (30 days - stable domain, autonomous spawning is well-researched by Ralph)

**Key recommendations for planner:**
1. Implement capability gap detection as prompt-based decision logic in all 6 Worker Ant commands
2. Implement spawn-tracker.sh with can_spawn(), record_spawn(), record_outcome() functions
3. Implement circuit-breaker.sh with check_circuit_breaker(), record_spawn_failure(), trigger_circuit_breaker_cooldown(), reset_circuit_breaker() functions
4. Implement spawn-outcome-tracker.sh with record_successful_spawn(), record_failed_spawn(), get_specialist_confidence() functions
5. Update COLONY_STATE.json schema with spawn_tracking section (depth, total_spawns, spawn_history, failed_specialist_types, cooldown_specialists, circuit_breaker_history)
6. Add context inheritance template to all Task tool spawn calls in Worker Ant prompts
7. Verify resource constraints (budget, depth, circuit breaker) BEFORE every Task tool spawn
8. Test spawning safeguards with attempt to trigger infinite loops, verify circuit breaker engages
