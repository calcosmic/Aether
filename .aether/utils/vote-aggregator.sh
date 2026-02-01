#!/bin/bash
# Aether Vote Aggregator Utility
# Implements vote aggregation, supermajority calculation, and Critical veto power
#
# Usage:
#   source .aether/utils/vote-aggregator.sh
#   result=$(aggregate_votes "/path/to/votes/dir")
#   decision=$(calculate_supermajority "$votes_file")

# Source required utilities
SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_PATH/../.." && pwd)"

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
WATCHER_WEIGHTS_FILE="$AETHER_ROOT/.aether/data/watcher_weights.json"
SUPERMAJORITY_THRESHOLD=67

# Helper: Get current timestamp in ISO 8601 format
get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# Aggregate votes from all vote JSON files in a directory
# Arguments: votes_dir
# Returns: Combined JSON array of votes with weights
aggregate_votes() {
    local votes_dir="$1"

    if [ ! -d "$votes_dir" ]; then
        echo "Error: Votes directory does not exist: $votes_dir" >&2
        return 1
    fi

    # Count vote files
    local vote_count=$(find "$votes_dir" -name "*.json" -type f | wc -l | tr -d ' ')

    if [ "$vote_count" -ne 4 ]; then
        echo "Error: Expected 4 vote files, found $vote_count" >&2
        return 1
    fi

    # Combine all vote files and add weights
    local combined
    combined=$(jq -s '
        map(. + {
            weight: (.watcher as $w | $WATCHER_WEIGHTS.watcher_weights[$w] // 1.0)
        })
    ' --argjson WATCHER_WEIGHTS "$(cat "$WATCHER_WEIGHTS_FILE")" "$votes_dir"/*.json)

    echo "$combined"
    return 0
}

# Calculate supermajority with Critical veto check
# Arguments: votes_file (JSON array of votes)
# Returns: 0 (APPROVED) or 1 (REJECTED), prints decision to stdout
calculate_supermajority() {
    local votes_file="$1"

    if [ ! -f "$votes_file" ]; then
        echo "Error: Votes file does not exist: $votes_file" >&2
        return 1
    fi

    # Check for Critical veto FIRST (any Critical severity REJECT blocks approval)
    local has_critical
    has_critical=$(jq '
        [.[] | select(.decision == "REJECT")] |
        any(.issues[]?; .severity == "Critical")
    ' "$votes_file")

    if [ "$has_critical" == "true" ]; then
        echo "REJECTED (Critical veto)"
        return 1
    fi

    # Calculate weighted approval percentage
    local approve_weight total_weight percentage
    approve_weight=$(jq '[.[] | select(.decision == "APPROVE")] | map(.weight) | add // 0' "$votes_file")
    total_weight=$(jq '[.[] | .weight] | add' "$votes_file")

    # Handle edge case of zero total weight
    if [ -z "$total_weight" ] || [ "$total_weight" == "null" ] || [ "$total_weight" == "0" ]; then
        echo "REJECTED (No valid votes)"
        return 1
    fi

    percentage=$(echo "scale=2; $approve_weight / $total_weight * 100" | bc)

    # Check if supermajority threshold met
    local threshold_check=$(echo "$percentage >= $SUPERMAJORITY_THRESHOLD" | bc)

    if [ "$threshold_check" == "1" ]; then
        echo "APPROVED (${percentage}% >= ${SUPERMAJORITY_THRESHOLD}%)"
        return 0
    else
        echo "REJECTED (${percentage}% < ${SUPERMAJORITY_THRESHOLD}%)"
        return 1
    fi
}

# Record vote outcome in COLONY_STATE.json verification.votes array
# Arguments: watcher, decision, issues_json, verification_id
# Returns: 0 on success, 1 on failure
record_vote_outcome() {
    local watcher="$1"
    local decision="$2"
    local issues_json="$3"
    local verification_id="$4"

    if [ -z "$verification_id" ]; then
        echo "Error: verification_id is required" >&2
        return 1
    fi

    # Validate decision
    if [ "$decision" != "APPROVE" ] && [ "$decision" != "REJECT" ]; then
        echo "Error: Invalid decision: $decision (must be 'APPROVE' or 'REJECT')" >&2
        return 1
    fi

    local timestamp=$(get_timestamp)

    # Get watcher weight
    local watcher_weight
    watcher_weight=$(jq -r ".watcher_weights.$watcher" "$WATCHER_WEIGHTS_FILE")

    if [ -z "$watcher_weight" ] || [ "$watcher_weight" == "null" ]; then
        watcher_weight=1.0
    fi

    # Update COLONY_STATE.json atomically
    local updated_state
    updated_state=$(jq "
        .verification.votes += [{
            \"id\": \"$verification_id\",
            \"watcher\": \"$watcher\",
            \"decision\": \"$decision\",
            \"weight\": $watcher_weight,
            \"issues\": $issues_json,
            \"timestamp\": \"$timestamp\",
            \"outcome\": \"pending\"
        }] |
        .verification.last_updated = \"$timestamp\"
    " "$COLONY_STATE_FILE")

    if ! atomic_write "$COLONY_STATE_FILE" "$updated_state"; then
        echo "Failed to record vote outcome" >&2
        return 1
    fi

    return 0
}

# Get vote history for a verification event
# Arguments: verification_id
get_vote_history() {
    local verification_id="$1"

    jq -r ".verification.votes[] | select(.id == \"$verification_id\")" "$COLONY_STATE_FILE"
}

# Export functions
export -f aggregate_votes calculate_supermajority record_vote_outcome get_vote_history
