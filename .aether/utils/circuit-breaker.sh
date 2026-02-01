#!/bin/bash
# Aether Circuit Breaker Utility
# Implements circuit breaker pattern for failed spawn detection
#
# Usage:
#   source .aether/utils/circuit-breaker.sh
#   check_circuit_breaker "database_specialist"
#   record_spawn_failure "database_specialist" "spawn_123" "Connection timeout"
#   trigger_circuit_breaker_cooldown "database_specialist"
#   reset_circuit_breaker

# Source required utilities
# Find Aether root - prefer current directory if in Aether, otherwise walk up from script
if git rev-parse --show-toplevel >/dev/null 2>&1; then
    # We're in a git repo - use git root as Aether root
    AETHER_ROOT="$(git rev-parse --show-toplevel)"
else
    # Walk up from script location
    SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    AETHER_ROOT="$(cd "$SCRIPT_PATH/../.." && pwd)"
fi

# Try to source atomic-write, handle case where being sourced directly
if [ -f "$AETHER_ROOT/.aether/utils/atomic-write.sh" ]; then
    source "$AETHER_ROOT/.aether/utils/atomic-write.sh"
fi

# Configuration
COLONY_STATE_FILE="$AETHER_ROOT/.aether/data/COLONY_STATE.json"

# Helper: Get current timestamp in ISO 8601 format
get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# Helper: Get current epoch timestamp
get_epoch() {
    date +%s
}

# Check if circuit breaker allows spawning for a specialist type
# Arguments: specialist_type
# Returns: 0 (can spawn) or 1 (circuit breaker active)
check_circuit_breaker() {
    local specialist_type="$1"

    # Read current state
    local trips cooldown_until
    trips=$(jq -r '.resource_budgets.circuit_breaker_trips' "$COLONY_STATE_FILE")
    cooldown_until=$(jq -r '.resource_budgets.circuit_breaker_cooldown_until // "null"' "$COLONY_STATE_FILE")

    # Check if cooldown is active
    if [ "$cooldown_until" != "null" ] && [ -n "$cooldown_until" ]; then
        local current_epoch cooldown_epoch
        current_epoch=$(get_epoch)

        # Parse cooldown timestamp with UTC timezone (Python handles ISO 8601 correctly)
        cooldown_epoch=$(python3 -c "from datetime import datetime, timezone; ts = datetime.strptime('$cooldown_until', '%Y-%m-%dT%H:%M:%SZ').replace(tzinfo=timezone.utc); print(int(ts.timestamp()))" 2>/dev/null || echo 0)

        # Check if cooldown has expired
        if [ "$cooldown_epoch" -gt 0 ] && [ "$current_epoch" -ge "$cooldown_epoch" ]; then
            echo "Circuit breaker cooldown expired, resetting..." >&2
            reset_circuit_breaker
            trips=0
        elif [ "$cooldown_epoch" -gt 0 ]; then
            echo "Circuit breaker active: cooldown until $cooldown_until" >&2
            return 1
        fi
    fi

    # Check trip count
    if [ "$trips" -ge 3 ]; then
        echo "Circuit breaker tripped: $trips/3 failures for $specialist_type" >&2
        return 1
    fi

    return 0
}

# Record a spawn failure and potentially trigger circuit breaker
# Arguments: specialist_type, spawn_id, failure_reason
record_spawn_failure() {
    local specialist_type="$1"
    local spawn_id="$2"
    local failure_reason="$3"

    local timestamp=$(get_timestamp)

    # Count recent failures of this specialist type
    local recent_failures
    recent_failures=$(jq -r "
        .spawn_tracking.spawn_history |
        map(select(.specialist == \"$specialist_type\" and .outcome == \"failure\")) |
        length
    " "$COLONY_STATE_FILE")

    # Increment circuit breaker trips
    local updated_state
    updated_state=$(jq "
        .resource_budgets.circuit_breaker_trips += 1 |
        .spawn_tracking.failed_specialist_types += [\"$specialist_type\"] |
        .spawn_tracking.failed_specialist_types |= unique |
        .spawn_tracking.circuit_breaker_history += [{
            \"specialist\": \"$specialist_type\",
            \"failures\": $((recent_failures + 1)),
            \"reason\": \"$failure_reason\",
            \"timestamp\": \"$timestamp\"
        }]
    " "$COLONY_STATE_FILE")

    if ! atomic_write "$COLONY_STATE_FILE" "$updated_state"; then
        echo "Failed to record spawn failure" >&2
        return 1
    fi

    # Get updated trip count
    local new_trips
    new_trips=$(jq -r '.resource_budgets.circuit_breaker_trips' "$COLONY_STATE_FILE")

    echo "Recorded failure for $specialist_type: $failure_reason" >&2
    echo "Circuit breaker trips: $new_trips/3" >&2

    # Trigger cooldown if 3 failures reached
    if [ "$new_trips" -ge 3 ]; then
        trigger_circuit_breaker_cooldown "$specialist_type"
    fi

    return 0
}

# Trigger circuit breaker cooldown (30 minutes)
# Arguments: specialist_type
trigger_circuit_breaker_cooldown() {
    local specialist_type="$1"

    # Calculate cooldown timestamp (now + 30 minutes)
    local cooldown_timestamp
    local current_epoch=$(get_epoch)
    local cooldown_epoch=$((current_epoch + 1800))  # 30 minutes = 1800 seconds

    # Format as ISO 8601 using Python (cross-platform)
    cooldown_timestamp=$(python3 -c "from datetime import datetime, timezone; print(datetime.fromtimestamp($cooldown_epoch, timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ'))")

    # Update state
    local updated_state
    updated_state=$(jq "
        .resource_budgets.circuit_breaker_cooldown_until = \"$cooldown_timestamp\" |
        .spawn_tracking.cooldown_specialists += [\"$specialist_type\"] |
        .spawn_tracking.cooldown_specialists |= unique
    " "$COLONY_STATE_FILE")

    if ! atomic_write "$COLONY_STATE_FILE" "$updated_state"; then
        echo "Failed to trigger circuit breaker cooldown" >&2
        return 1
    fi

    echo "========================================" >&2
    echo "⚠️  CIRCUIT BREAKER TRIGGERED" >&2
    echo "========================================" >&2
    echo "Specialist: $specialist_type" >&2
    echo "Cooldown until: $cooldown_timestamp" >&2
    echo "Duration: 30 minutes" >&2
    echo "========================================" >&2
    echo "" >&2
    echo "Spawning of $specialist_type is blocked until cooldown expires." >&2
    echo "Use reset_circuit_breaker() to manually reset after resolving the issue." >&2
    echo "" >&2

    return 0
}

# Reset circuit breaker state
# Arguments: none
reset_circuit_breaker() {
    local updated_state
    updated_state=$(jq "
        .resource_budgets.circuit_breaker_trips = 0 |
        .resource_budgets.circuit_breaker_cooldown_until = null |
        .spawn_tracking.failed_specialist_types = [] |
        .spawn_tracking.cooldown_specialists = []
    " "$COLONY_STATE_FILE")

    if ! atomic_write "$COLONY_STATE_FILE" "$updated_state"; then
        echo "Failed to reset circuit breaker" >&2
        return 1
    fi

    echo "✓ Circuit breaker reset" >&2
    echo "  - Trip count: 0" >&2
    echo "  - Cooldown: cleared" >&2
    echo "  - Failed types: cleared" >&2
    echo "" >&2

    return 0
}

# Export functions
export -f check_circuit_breaker record_spawn_failure trigger_circuit_breaker_cooldown reset_circuit_breaker
