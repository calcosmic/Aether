#!/bin/bash
# Aether Spawn Outcome Tracker
# Enhanced with Bayesian Beta distribution confidence scoring
# Phase 8: Replaces simple +0.1/-0.15 arithmetic with α/(α+β) formula
#
# Bayesian inference with Beta(α,β) distribution:
# - Prior: Beta(1,1) represents uniform distribution (no prior knowledge)
# - Success: α_new = α_old + 1 (increment alpha)
# - Failure: β_new = β_old + 1 (increment beta)
# - Confidence (posterior mean): μ = α / (α + β)
#
# Usage:
#   source .aether/utils/spawn-outcome-tracker.sh
#   record_successful_spawn "database_specialist" "database" "spawn_123"
#   record_failed_spawn "database_specialist" "database" "spawn_124" "Connection timeout"
#   confidence=$(get_specialist_confidence "database_specialist" "database")

# Source required utilities
# Find Aether root by walking up from the script location or current directory
if [ -n "${BASH_SOURCE[0]}" ]; then
    SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    AETHER_ROOT="$(cd "$SCRIPT_PATH/../.." && pwd)"
else
    AETHER_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
fi

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

# Source Bayesian confidence library
if [ -f "$AETHER_ROOT/.aether/utils/bayesian-confidence.sh" ]; then
    source "$AETHER_ROOT/.aether/utils/bayesian-confidence.sh"
else
    source ".aether/utils/bayesian-confidence.sh"
fi

# Configuration
COLONY_STATE_FILE="$AETHER_ROOT/.aether/data/COLONY_STATE.json"
LOCK_FILE="$AETHER_ROOT/.aether/locks/spawn_outcome_tracker.lock"

# Constants
DEFAULT_CONFIDENCE=0.5
MAX_CONFIDENCE=1.0
MIN_CONFIDENCE=0.0

# Helper: Get current timestamp in ISO 8601 format
get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# Record a successful spawn and increment confidence (Bayesian alpha increment)
# Arguments: specialist_type, task_type, spawn_id
# Returns: 0 on success, 1 on failure
record_successful_spawn() {
    local specialist_type="$1"
    local task_type="$2"
    local spawn_id="$3"

    # Acquire lock
    if ! acquire_lock "$LOCK_FILE"; then
        echo "Cannot acquire lock to record successful spawn" >&2
        return 1
    fi

    local timestamp=$(get_timestamp)

    # Get current alpha/beta (default to prior 1,1)
    local current=$(jq -r "
        .meta_learning.specialist_confidence.\"$specialist_type\".\"$task_type\"
        // {\"alpha\": 1, \"beta\": 1, \"confidence\": 0.5}
    " "$COLONY_STATE_FILE")

    local alpha=$(echo "$current" | jq -r '.alpha // 1')
    local beta=$(echo "$current" | jq -r '.beta // 1')

    # Update Bayesian parameters: increment alpha for success
    local updated_params=$(update_bayesian_parameters "$alpha" "$beta" "success")
    local new_alpha=$(echo "$updated_params" | cut -d' ' -f1)
    local new_beta=$(echo "$updated_params" | cut -d' ' -f2)

    # Calculate new confidence
    local new_confidence=$(calculate_confidence "$new_alpha" "$new_beta")

    # Calculate derived stats
    local total_spawns=$(echo "$new_alpha + $new_beta - 2" | bc)
    local successful_spawns=$(echo "$new_alpha - 1" | bc)
    local failed_spawns=$(echo "$new_beta - 1" | bc)

    # Update state atomically with full Bayesian object
    local updated_state
    updated_state=$(jq "
        .meta_learning.specialist_confidence.\"$specialist_type\".\"$task_type\" = {
            \"alpha\": $new_alpha,
            \"beta\": $new_beta,
            \"confidence\": $new_confidence,
            \"total_spawns\": $total_spawns,
            \"successful_spawns\": $successful_spawns,
            \"failed_spawns\": $failed_spawns,
            \"last_updated\": \"$timestamp\"
        } |
        .meta_learning.spawn_outcomes += [{
            \"spawn_id\": \"$spawn_id\",
            \"specialist\": \"$specialist_type\",
            \"task_type\": \"$task_type\",
            \"outcome\": \"success\",
            \"timestamp\": \"$timestamp\"
        }] |
        .meta_learning.last_updated = \"$timestamp\"
    " "$COLONY_STATE_FILE")

    if ! atomic_write "$COLONY_STATE_FILE" "$updated_state"; then
        echo "Failed to record successful spawn" >&2
        release_lock
        return 1
    fi

    # Release lock
    release_lock

    return 0
}

# Record a failed spawn and decrement confidence (Bayesian beta increment)
# Arguments: specialist_type, task_type, spawn_id, failure_reason
# Returns: 0 on success, 1 on failure
record_failed_spawn() {
    local specialist_type="$1"
    local task_type="$2"
    local spawn_id="$3"
    local failure_reason="$4"

    # Acquire lock
    if ! acquire_lock "$LOCK_FILE"; then
        echo "Cannot acquire lock to record failed spawn" >&2
        return 1
    fi

    local timestamp=$(get_timestamp)

    # Get current alpha/beta (default to prior 1,1)
    local current=$(jq -r "
        .meta_learning.specialist_confidence.\"$specialist_type\".\"$task_type\"
        // {\"alpha\": 1, \"beta\": 1, \"confidence\": 0.5}
    " "$COLONY_STATE_FILE")

    local alpha=$(echo "$current" | jq -r '.alpha // 1')
    local beta=$(echo "$current" | jq -r '.beta // 1')

    # Update Bayesian parameters: increment beta for failure
    local updated_params=$(update_bayesian_parameters "$alpha" "$beta" "failure")
    local new_alpha=$(echo "$updated_params" | cut -d' ' -f1)
    local new_beta=$(echo "$updated_params" | cut -d' ' -f2)

    # Calculate new confidence
    local new_confidence=$(calculate_confidence "$new_alpha" "$new_beta")

    # Calculate derived stats
    local total_spawns=$(echo "$new_alpha + $new_beta - 2" | bc)
    local successful_spawns=$(echo "$new_alpha - 1" | bc)
    local failed_spawns=$(echo "$new_beta - 1" | bc)

    # Update state atomically with full Bayesian object
    local updated_state
    updated_state=$(jq "
        .meta_learning.specialist_confidence.\"$specialist_type\".\"$task_type\" = {
            \"alpha\": $new_alpha,
            \"beta\": $new_beta,
            \"confidence\": $new_confidence,
            \"total_spawns\": $total_spawns,
            \"successful_spawns\": $successful_spawns,
            \"failed_spawns\": $failed_spawns,
            \"last_updated\": \"$timestamp\"
        } |
        .meta_learning.spawn_outcomes += [{
            \"spawn_id\": \"$spawn_id\",
            \"specialist\": \"$specialist_type\",
            \"task_type\": \"$task_type\",
            \"outcome\": \"failure\",
            \"reason\": \"$failure_reason\",
            \"timestamp\": \"$timestamp\"
        }] |
        .meta_learning.last_updated = \"$timestamp\"
    " "$COLONY_STATE_FILE")

    if ! atomic_write "$COLONY_STATE_FILE" "$updated_state"; then
        echo "Failed to record failed spawn" >&2
        release_lock
        return 1
    fi

    # Release lock
    release_lock

    return 0
}

# Get confidence score for specialist-task pairing (Bayesian)
# Arguments: specialist_type, task_type, [full_object=false]
# Returns: confidence score (0.0 - 1.0) or full Bayesian object if full_object=true
get_specialist_confidence() {
    local specialist_type="$1"
    local task_type="$2"
    local full_object="${3:-false}"

    local result=$(jq -r "
        .meta_learning.specialist_confidence.\"$specialist_type\".\"$task_type\"
        // {\"alpha\": 1, \"beta\": 1, \"confidence\": 0.5}
    " "$COLONY_STATE_FILE")

    if [ "$full_object" = "true" ]; then
        echo "$result"
    else
        echo "$result" | jq -r '.confidence // 0.5'
    fi
}

# Get all spawn outcomes for a specialist
# Arguments: specialist_type
# Returns: JSON array of outcomes
get_specialist_outcomes() {
    local specialist_type="$1"

    jq -r "
        .meta_learning.spawn_outcomes |
        map(select(.specialist == \"$specialist_type\"))
    " "$COLONY_STATE_FILE"
}

# Get overall meta-learning statistics (Bayesian)
# Returns: Formatted statistics with alpha/beta/counts
get_meta_learning_stats() {
    echo "Meta-Learning Statistics (Bayesian Beta Distribution)"
    echo "======================================================"

    jq -r '
        .meta_learning.specialist_confidence |
        to_entries[] |
        .key as $specialist |
        .value |
        to_entries[] |
        "\($specialist) → \(.key): α=\(.value.alpha), β=\(.value.beta), confidence=\(.value.confidence), total=\(.value.total_spawns), successes=\(.value.successful_spawns), failures=\(.value.failed_spawns)"
    ' "$COLONY_STATE_FILE" | while IFS= read -r line; do
        echo "  $line"
    done

    echo ""
    echo "  Confidence formula: α / (α + β)"
    echo "  Prior: α=1, β=1 (confidence=0.5)"
    echo "  Success: increment α, Failure: increment β"
}

# Export functions
export -f record_successful_spawn record_failed_spawn get_specialist_confidence
export -f get_specialist_outcomes get_meta_learning_stats
