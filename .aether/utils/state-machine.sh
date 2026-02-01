#!/bin/bash
# Aether State Machine Utility
# Implements state machine logic for colony orchestration
#
# Usage:
#   source .aether/utils/state-machine.sh
#   is_valid_transition "IDLE" "INIT" && echo "Valid"

# Source required utilities
_AETHER_UTILS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$_AETHER_UTILS_DIR/atomic-write.sh"
source "$_AETHER_UTILS_DIR/file-lock.sh"

# Colony state file path
COLONY_STATE="${COLONY_STATE:-.aether/data/COLONY_STATE.json}"

# Get current colony state from COLONY_STATE.json
# Returns: State string or "IDLE" if not set
get_current_state() {
    jq -r '.colony_status.state // "IDLE"' "$COLONY_STATE" 2>/dev/null || echo "IDLE"
}

# Get all valid states from COLONY_STATE.json
# Returns: Newline-separated list of valid states
get_valid_states() {
    jq -r '.state_machine.valid_states[]' "$COLONY_STATE" 2>/dev/null || echo "IDLE"
}

# Check if a state is valid
# Args: state_name
# Returns: 0 if valid, 1 if not
is_valid_state() {
    local state_name="$1"
    local valid_states=$(get_valid_states)
    echo "$valid_states" | grep -qx "$state_name"
}

# Check if a state transition is valid
# Args: from_state, to_state
# Returns: 0 if valid, 1 if not
# Note: Uses case statement for bash 3.x compatibility (no associative arrays)
is_valid_transition() {
    local from_state="$1"
    local to_state="$2"
    local key="${from_state}_${to_state}"

    # Use case statement for bash 3.x compatibility (macOS default)
    case "$key" in
        IDLE_INIT| \
        INIT_PLANNING| \
        PLANNING_EXECUTING| \
        EXECUTING_VERIFYING| \
        VERIFYING_COMPLETED| \
        VERIFYING_EXECUTING| \
        EXECUTING_FAILED| \
        FAILED_PLANNING| \
        COMPLETED_IDLE)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Validate state transition with error messages
# Args: from_state, to_state
# Returns: 0 if valid, 1 if not (prints error message)
validate_transition() {
    local from_state="$1"
    local to_state="$2"

    if is_valid_transition "$from_state" "$to_state"; then
        return 0
    else
        echo "Invalid transition: $from_state -> $to_state" >&2
        return 1
    fi
}

# Export functions for use in other scripts
export -f get_current_state get_valid_states is_valid_state
export -f is_valid_transition validate_transition
