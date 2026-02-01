#!/bin/bash
# Aether Spawn Tracker Utility
# Implements resource budget enforcement and spawn tracking for autonomous spawning
#
# Usage:
#   source .aether/utils/spawn-tracker.sh
#   can_spawn && echo "Can spawn" || echo "Cannot spawn"
#   spawn_id=$(record_spawn "builder" "database_specialist" "Database schema migration")
#   record_outcome "$spawn_id" "success" "Task completed successfully"

# Source required utilities
# Find Aether root by walking up from the script location
SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_PATH/../.." && pwd)"

# Try to source from AETHER_ROOT, fallback to relative paths
if [ -f "$AETHER_ROOT/.aether/utils/atomic-write.sh" ]; then
    source "$AETHER_ROOT/.aether/utils/atomic-write.sh"
else
    source ".aether/utils/atomic-write.sh"
fi

if [ -f "$AETHER_ROOT/.aether/utils/file-lock.sh" ]; then
    source "$AETHER_ROOT/.aether/utils/file-lock.sh"
else
    source ".aether/utils/file-lock.sh"
fi

# Source spawn-outcome-tracker for meta-learning
if [ -f "$AETHER_ROOT/.aether/utils/spawn-outcome-tracker.sh" ]; then
    source "$AETHER_ROOT/.aether/utils/spawn-outcome-tracker.sh"
else
    source ".aether/utils/spawn-outcome-tracker.sh"
fi

# Configuration
COLONY_STATE_FILE="$AETHER_ROOT/.aether/data/COLONY_STATE.json"
LOCK_FILE="$AETHER_ROOT/.aether/locks/spawn_tracker.lock"

# Helper: Get current timestamp in ISO 8601 format
get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# Helper: Get current epoch timestamp
get_epoch() {
    date +%s
}

# Derive task type from task context using keyword matching
# Arguments: task_context
# Returns: task_type string (database, frontend, backend, api, testing, security, etc.)
derive_task_type() {
    local task_context="$1"
    local task_lower=$(echo "$task_context" | tr '[:upper:]' '[:lower:]')

    # Priority order for task type detection
    if [[ "$task_lower" =~ (database|sql|nosql|mongo|postgres|mysql|schema|migration|query) ]]; then
        echo "database"
    elif [[ "$task_lower" =~ (frontend|ui|ux|css|html|component|view|template) ]]; then
        echo "frontend"
    elif [[ "$task_lower" =~ (backend|server|endpoint|route|controller|service) ]]; then
        echo "backend"
    elif [[ "$task_lower" =~ (api|rest|graphql|webhook|integration) ]]; then
        echo "api"
    elif [[ "$task_lower" =~ (test|testing|spec|validation|verify|quality|assert) ]]; then
        echo "testing"
    elif [[ "$task_lower" =~ (security|auth|authentication|authorization|encryption|csrf|xss) ]]; then
        echo "security"
    elif [[ "$task_lower" =~ (performance|optimization|cache|speed|latency|scalability) ]]; then
        echo "performance"
    elif [[ "$task_lower" =~ (devops|deploy|ci/cd|pipeline|docker|kubernetes|infrastructure) ]]; then
        echo "devops"
    elif [[ "$task_lower" =~ (analyze|analysis|research|investigate|explore) ]]; then
        echo "analysis"
    elif [[ "$task_lower" =~ (plan|planning|design|architect|structure) ]]; then
        echo "planning"
    elif [[ "$task_lower" =~ (implement|code|write|create|build|develop) ]]; then
        echo "implementation"
    else
        echo "general"
    fi
}

# Check if spawning is allowed based on resource constraints
# Returns: 0 (can spawn) or 1 (cannot spawn)
can_spawn() {
    # Acquire lock to prevent concurrent spawn decisions
    if ! acquire_lock "$LOCK_FILE"; then
        echo "Cannot acquire spawn decision lock" >&2
        return 1
    fi

    # Read current state
    local max_spawns current_spawns depth circuit_trips cooldown_until
    max_spawns=$(jq -r '.resource_budgets.max_spawns_per_phase' "$COLONY_STATE_FILE")
    current_spawns=$(jq -r '.resource_budgets.current_spawns' "$COLONY_STATE_FILE")
    depth=$(jq -r '.spawn_tracking.depth' "$COLONY_STATE_FILE")
    circuit_trips=$(jq -r '.resource_budgets.circuit_breaker_trips' "$COLONY_STATE_FILE")
    cooldown_until=$(jq -r '.resource_budgets.circuit_breaker_cooldown_until // "null"' "$COLONY_STATE_FILE")

    local reason=""
    local can_spawn_result=0

    # Check spawn budget
    if [ "$current_spawns" -ge "$max_spawns" ]; then
        reason="Spawn budget exceeded ($current_spawns/$max_spawns)"
        can_spawn_result=1
    fi

    # Check spawn depth
    local max_depth
    max_depth=$(jq -r '.resource_budgets.max_spawn_depth' "$COLONY_STATE_FILE")
    if [ "$depth" -ge "$max_depth" ]; then
        reason="Max spawn depth exceeded ($depth/$max_depth)"
        can_spawn_result=1
    fi

    # Check circuit breaker
    if [ "$circuit_trips" -ge 3 ]; then
        reason="Circuit breaker tripped ($circuit_trips/3)"
        can_spawn_result=1
    fi

    # Check cooldown
    if [ "$cooldown_until" != "null" ]; then
        local current_epoch cooldown_epoch
        current_epoch=$(get_epoch)
        cooldown_epoch=$(date -d "$cooldown_until" +%s 2>/dev/null || echo 0)
        if [ "$current_epoch" -lt "$cooldown_epoch" ]; then
            reason="Circuit breaker cooldown active until $cooldown_until"
            can_spawn_result=1
        fi
    fi

    # Release lock
    release_lock

    if [ $can_spawn_result -ne 0 ]; then
        echo "Cannot spawn: $reason" >&2
    fi

    return $can_spawn_result
}

# Record a spawn event
# Arguments: parent_caste, specialist_type, task_context
# Returns: spawn_id
record_spawn() {
    local parent_caste="$1"
    local specialist_type="$2"
    local task_context="$3"

    # Acquire lock
    if ! acquire_lock "$LOCK_FILE"; then
        echo "Cannot acquire lock to record spawn" >&2
        return 1
    fi

    # Generate spawn ID
    local spawn_id="spawn_$(get_epoch)"
    local timestamp=$(get_timestamp)

    # Get current depth
    local current_depth
    current_depth=$(jq -r '.spawn_tracking.depth' "$COLONY_STATE_FILE")
    local new_depth=$((current_depth + 1))

    # Update state atomically
    local updated_state
    updated_state=$(jq "
        .resource_budgets.current_spawns += 1 |
        .spawn_tracking.depth += 1 |
        .spawn_tracking.total_spawns += 1 |
        .spawn_tracking.spawn_history += [{
            \"id\": \"$spawn_id\",
            \"parent\": \"$parent_caste\",
            \"specialist\": \"$specialist_type\",
            \"task\": \"$task_context\",
            \"timestamp\": \"$timestamp\",
            \"depth\": $new_depth,
            \"outcome\": \"pending\"
        }]
    " "$COLONY_STATE_FILE")

    if ! atomic_write "$COLONY_STATE_FILE" "$updated_state"; then
        echo "Failed to record spawn" >&2
        release_lock
        return 1
    fi

    # Release lock
    release_lock

    echo "$spawn_id"
    return 0
}

# Record spawn outcome
# Arguments: spawn_id, outcome (success|failure), notes
record_outcome() {
    local spawn_id="$1"
    local outcome="$2"
    local notes="$3"

    # Acquire lock
    if ! acquire_lock "$LOCK_FILE"; then
        echo "Cannot acquire lock to record outcome" >&2
        return 1
    fi

    local timestamp=$(get_timestamp)

    # Validate outcome
    if [ "$outcome" != "success" ] && [ "$outcome" != "failure" ]; then
        echo "Invalid outcome: $outcome (must be 'success' or 'failure')" >&2
        release_lock
        return 1
    fi

    # Extract spawn details from history
    local spawn_details
    spawn_details=$(jq -r ".spawn_tracking.spawn_history[] | select(.id == \"$spawn_id\")" "$COLONY_STATE_FILE")

    if [ -z "$spawn_details" ]; then
        echo "Spawn ID not found: $spawn_id" >&2
        release_lock
        return 1
    fi

    local specialist_type=$(echo "$spawn_details" | jq -r '.specialist')
    local task_context=$(echo "$spawn_details" | jq -r '.task')

    # Derive task type from context
    local task_type
    task_type=$(derive_task_type "$task_context")

    # Update spawn history entry and performance metrics
    local perf_field="successful_spawns"
    if [ "$outcome" = "failure" ]; then
        perf_field="failed_spawns"
    fi

    local updated_state
    updated_state=$(jq "
        .spawn_tracking.spawn_history |= map(
            if .id == \"$spawn_id\" then
                .outcome = \"$outcome\" |
                .completed_at = \"$timestamp\" |
                .notes = \"$notes\"
            else
                .
            end
        ) |
        .spawn_tracking.depth -= 1 |
        .performance_metrics.$perf_field += 1 |
        .resource_budgets.current_spawns -= 1
    " "$COLONY_STATE_FILE")

    if ! atomic_write "$COLONY_STATE_FILE" "$updated_state"; then
        echo "Failed to record outcome" >&2
        release_lock
        return 1
    fi

    # Release lock before calling outcome tracking (which acquires its own lock)
    release_lock

    # Record outcome for meta-learning confidence tracking
    if [ "$outcome" = "success" ]; then
        record_successful_spawn "$specialist_type" "$task_type" "$spawn_id"
    else
        record_failed_spawn "$specialist_type" "$task_type" "$spawn_id" "$notes"
    fi

    return 0
}

# Get spawn history
# Optional argument: limit (number of recent entries)
get_spawn_history() {
    local limit="${1:-100}"

    jq -r ".spawn_tracking.spawn_history[-${limit}:] | reverse | .[] |
        \"\(.id) | \(.parent) → \(.specialist) | \(.task) | \(.outcome) | \(.timestamp)\"" \
        "$COLONY_STATE_FILE"
}

# Get current spawn statistics
get_spawn_stats() {
    echo "=== Spawn Statistics ==="
    echo "Current Spawns: $(jq -r '.resource_budgets.current_spawns' $COLONY_STATE_FILE)/$(jq -r '.resource_budgets.max_spawns_per_phase' $COLONY_STATE_FILE)"
    echo "Current Depth: $(jq -r '.spawn_tracking.depth' $COLONY_STATE_FILE)/$(jq -r '.resource_budgets.max_spawn_depth' $COLONY_STATE_FILE)"
    echo "Total Spawns: $(jq -r '.spawn_tracking.total_spawns' $COLONY_STATE_FILE)"
    echo "Successful: $(jq -r '.performance_metrics.successful_spawns' $COLONY_STATE_FILE)"
    echo "Failed: $(jq -r '.performance_metrics.failed_spawns' $COLONY_STATE_FILE)"
    echo "Circuit Breaker Trips: $(jq -r '.resource_budgets.circuit_breaker_trips' $COLONY_STATE_FILE)"
}

# Reset spawn counters (for new phase)
reset_spawn_counters() {
    # Acquire lock
    if ! acquire_lock "$LOCK_FILE"; then
        echo "Cannot acquire lock to reset counters" >&2
        return 1
    fi

    local updated_state
    updated_state=$(jq "
        .resource_budgets.current_spawns = 0 |
        .spawn_tracking.depth = 0 |
        .spawn_tracking.total_spawns = 0 |
        .spawn_tracking.spawn_history = [] |
        .spawn_tracking.failed_specialist_types = [] |
        .spawn_tracking.cooldown_specialists = [] |
        .performance_metrics.successful_spawns = 0 |
        .performance_metrics.failed_spawns = 0 |
        .performance_metrics.avg_spawn_duration_seconds = 0
    " "$COLONY_STATE_FILE")

    if ! atomic_write "$COLONY_STATE_FILE" "$updated_state"; then
        echo "Failed to reset spawn counters" >&2
        release_lock
        return 1
    fi

    # Release lock
    release_lock

    echo "Spawn counters reset for new phase"
    return 0
}

# Export functions
export -f can_spawn record_spawn record_outcome get_spawn_history get_spawn_stats reset_spawn_counters derive_task_type get_specialist_confidence
