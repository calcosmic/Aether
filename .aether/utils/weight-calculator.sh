#!/bin/bash
# Aether Weight Calculator Utility
# Implements belief calibration for Watcher weights based on vote outcomes
#
# Usage:
#   source .aether/utils/weight-calculator.sh
#   weight=$(get_watcher_weight "security")
#   update_watcher_weight "security" "correct_reject" "security"

# Source required utilities
SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_PATH/../.." && pwd)"

if [ -f "$AETHER_ROOT/.aether/utils/atomic-write.sh" ]; then
    source "$AETHER_ROOT/.aether/utils/atomic-write.sh"
else
    source ".aether/utils/atomic-write.sh"
fi

# Configuration
WATCHER_WEIGHTS_FILE="$AETHER_ROOT/.aether/data/watcher_weights.json"
MIN_WEIGHT=0.1
MAX_WEIGHT=3.0

# Helper: Get current timestamp in ISO 8601 format
get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# Get current weight for a watcher
# Arguments: watcher_type
# Returns: numeric weight value
get_watcher_weight() {
    local watcher_type="$1"

    local weight
    weight=$(jq -r ".watcher_weights.$watcher_type" "$WATCHER_WEIGHTS_FILE")

    if [ -z "$weight" ] || [ "$weight" == "null" ]; then
        echo "Error: Unknown watcher type: $watcher_type" >&2
        echo "1.0"
        return 1
    fi

    echo "$weight"
    return 0
}

# Clamp weight to [min, max] bounds
# Arguments: weight
# Returns: clamped weight value
clamp_weight() {
    local weight="$1"

    # Use bc for floating-point comparison
    local clamped
    clamped=$(echo "scale=1; $weight < $MIN_WEIGHT ? $MIN_WEIGHT : ($weight > $MAX_WEIGHT ? $MAX_WEIGHT : $weight)" | bc)

    echo "$clamped"
}

# Update watcher weight based on vote outcome
# Arguments: watcher_type, vote_outcome, issue_category
# Returns: 0 on success, 1 on failure
#
# vote_outcome values:
#   - correct_approve: Watcher approved and phase succeeded (weight +0.1)
#   - correct_reject: Watcher rejected and issues were found (weight +0.15)
#   - incorrect_approve: Watcher approved but issues found later (weight -0.2)
#   - incorrect_reject: Watcher rejected but was wrong (weight -0.1)
update_watcher_weight() {
    local watcher_type="$1"
    local vote_outcome="$2"
    local issue_category="$3"

    # Validate watcher type
    local valid_watchers=("security" "performance" "quality" "test_coverage")
    local is_valid=false
    for watcher in "${valid_watchers[@]}"; do
        if [ "$watcher_type" == "$watcher" ]; then
            is_valid=true
            break
        fi
    done

    if [ "$is_valid" == "false" ]; then
        echo "Error: Invalid watcher type: $watcher_type (must be one of: ${valid_watchers[*]})" >&2
        return 1
    fi

    # Get current weight
    local current_weight
    current_weight=$(get_watcher_weight "$watcher_type")

    if [ $? -ne 0 ]; then
        return 1
    fi

    # Determine weight adjustment based on outcome
    local adjustment
    case "$vote_outcome" in
        correct_approve)
            adjustment=0.1
            ;;
        correct_reject)
            adjustment=0.15
            ;;
        incorrect_approve)
            adjustment=-0.2
            ;;
        incorrect_reject)
            adjustment=-0.1
            ;;
        *)
            echo "Error: Invalid vote_outcome: $vote_outcome (must be: correct_approve, correct_reject, incorrect_approve, incorrect_reject)" >&2
            return 1
            ;;
    esac

    # Calculate new weight
    local new_weight
    new_weight=$(echo "scale=1; $current_weight + $adjustment" | bc)

    # Apply domain expertise bonus if issue_category matches watcher_type
    # This doubles the weight adjustment for domain-matching issues
    if [ -n "$issue_category" ] && [ "$issue_category" == "$watcher_type" ]; then
        # For domain expertise, we boost the weight more significantly
        # If adjustment is positive, multiply by 2. If negative, apply less penalty
        if [ "$(echo "$adjustment > 0" | bc)" == "1" ]; then
            new_weight=$(echo "scale=1; $new_weight * 2" | bc)
        else
            # For incorrect votes in domain, reduce the penalty (only apply half)
            new_weight=$(echo "scale=1; $current_weight + ($adjustment / 2)" | bc)
        fi
    fi

    # Clamp to bounds
    new_weight=$(clamp_weight "$new_weight")

    # Update watcher_weights.json atomically
    local updated_weights
    updated_weights=$(jq "
        .watcher_weights.$watcher_type = $new_weight |
        .last_updated = \"$(get_timestamp)\"
    " "$WATCHER_WEIGHTS_FILE")

    if ! atomic_write "$WATCHER_WEIGHTS_FILE" "$updated_weights"; then
        echo "Failed to update watcher weight" >&2
        return 1
    fi

    echo "Updated $watcher_type weight: $current_weight → $new_weight (outcome: $vote_outcome)"
    return 0
}

# Reset all watcher weights to default (1.0)
# Returns: 0 on success, 1 on failure
reset_watcher_weights() {
    local updated_weights
    updated_weights=$(jq "
        .watcher_weights = {
            \"security\": 1.0,
            \"performance\": 1.0,
            \"quality\": 1.0,
            \"test_coverage\": 1.0
        } |
        .last_updated = \"$(get_timestamp)\"
    " "$WATCHER_WEIGHTS_FILE")

    if ! atomic_write "$WATCHER_WEIGHTS_FILE" "$updated_weights"; then
        echo "Failed to reset watcher weights" >&2
        return 1
    fi

    echo "All watcher weights reset to 1.0"
    return 0
}

# Get all watcher weights as formatted output
# Returns: Formatted table of watcher weights
get_all_weights() {
    jq -r '
        "Watcher Weights:",
        "  security: \(.watcher_weights.security)",
        "  performance: \(.watcher_weights.performance)",
        "  quality: \(.watcher_weights.quality)",
        "  test_coverage: \(.watcher_weights.test_coverage)",
        "",
        "Bounds: [\(.weight_bounds.min), \(.weight_bounds.max)]",
        "Last updated: \(.last_updated)"
    ' "$WATCHER_WEIGHTS_FILE"
}

# Validate weight bounds
# Arguments: weight
# Returns: 0 if within bounds, 1 if out of bounds
validate_weight_bounds() {
    local weight="$1"
    local min_check max_check

    min_check=$(echo "$weight >= $MIN_WEIGHT" | bc)
    max_check=$(echo "$weight <= $MAX_WEIGHT" | bc)

    if [ "$min_check" == "1" ] && [ "$max_check" == "1" ]; then
        return 0
    else
        return 1
    fi
}

# Export functions
export -f get_watcher_weight clamp_weight update_watcher_weight reset_watcher_weights get_all_weights validate_weight_bounds
