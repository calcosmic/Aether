#!/bin/bash
# Aether Spawn Outcome Tracker
# Implements confidence scoring for meta-learning with asymmetric penalty
#
# Confidence scoring rules:
# - Start at 0.5 (neutral)
# - Success: +0.1 (up to max 1.0)
# - Failure: -0.15 (down to min 0.0, asymmetric penalty)
#
# Asymmetric penalty makes failures more impactful, which feeds into
# Phase 8 Bayesian confidence updating.
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

# Configuration
COLONY_STATE_FILE="$AETHER_ROOT/.aether/data/COLONY_STATE.json"
LOCK_FILE="$AETHER_ROOT/.aether/locks/spawn_outcome_tracker.lock"

# Constants
DEFAULT_CONFIDENCE=0.5
SUCCESS_INCREMENT=0.1
FAILURE_DECREMENT=0.15
MAX_CONFIDENCE=1.0
MIN_CONFIDENCE=0.0

# Helper: Get current timestamp in ISO 8601 format
get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# Record a successful spawn and increment confidence
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

    # Get current confidence (default to DEFAULT_CONFIDENCE if not exists)
    local current_confidence
    current_confidence=$(jq -r "
        .meta_learning.specialist_confidence.\"$specialist_type\".\"$task_type\" // $DEFAULT_CONFIDENCE
    " "$COLONY_STATE_FILE")

    # Calculate new confidence (capped at MAX_CONFIDENCE)
    local new_confidence=$(echo "$current_confidence + $SUCCESS_INCREMENT" | bc)
    if (( $(echo "$new_confidence > $MAX_CONFIDENCE" | bc -l) )); then
        new_confidence=$MAX_CONFIDENCE
    fi

    # Update state atomically
    local updated_state
    updated_state=$(jq "
        .meta_learning.specialist_confidence.\"$specialist_type\".\"$task_type\" = $new_confidence |
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

# Record a failed spawn and decrement confidence (asymmetric penalty)
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

    # Get current confidence (default to DEFAULT_CONFIDENCE if not exists)
    local current_confidence
    current_confidence=$(jq -r "
        .meta_learning.specialist_confidence.\"$specialist_type\".\"$task_type\" // $DEFAULT_CONFIDENCE
    " "$COLONY_STATE_FILE")

    # Calculate new confidence with asymmetric penalty (capped at MIN_CONFIDENCE)
    local new_confidence=$(echo "$current_confidence - $FAILURE_DECREMENT" | bc)
    if (( $(echo "$new_confidence < $MIN_CONFIDENCE" | bc -l) )); then
        new_confidence=$MIN_CONFIDENCE
    fi

    # Update state atomically
    local updated_state
    updated_state=$(jq "
        .meta_learning.specialist_confidence.\"$specialist_type\".\"$task_type\" = $new_confidence |
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

# Get confidence score for specialist-task pairing
# Arguments: specialist_type, task_type
# Returns: confidence score (0.0 - 1.0)
get_specialist_confidence() {
    local specialist_type="$1"
    local task_type="$2"

    # Read confidence from state (default to DEFAULT_CONFIDENCE)
    local confidence
    confidence=$(jq -r "
        .meta_learning.specialist_confidence.\"$specialist_type\".\"$task_type\" // $DEFAULT_CONFIDENCE
    " "$COLONY_STATE_FILE")

    echo "$confidence"
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

# Get overall meta-learning statistics
# Returns: Formatted statistics
get_meta_learning_stats() {
    echo "=== Meta-Learning Statistics ==="
    echo "Total Outcomes Recorded: $(jq -r '.meta_learning.spawn_outcomes | length' "$COLONY_STATE_FILE")"
    echo "Last Updated: $(jq -r '.meta_learning.last_updated // "Never"' "$COLONY_STATE_FILE")"
    echo ""
    echo "Specialist Confidence Scores:"
    jq -r '
        .meta_learning.specialist_confidence |
        to_entries[] |
        "  \(.key):" +
        (
            .value |
            to_entries[] |
            "    \(.key): \(.value)"
        )
    ' "$COLONY_STATE_FILE" 2>/dev/null || echo "  No confidence data yet"
}

# Export functions
export -f record_successful_spawn record_failed_spawn get_specialist_confidence
export -f get_specialist_outcomes get_meta_learning_stats
