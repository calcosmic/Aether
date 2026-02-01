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
source "$_AETHER_UTILS_DIR/checkpoint.sh"

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

# Get next checkpoint number
# Returns: Next checkpoint number from COLONY_STATE.json
get_next_checkpoint_number() {
    jq -r '.checkpoints.checkpoint_count // 0' "$COLONY_STATE"
}

# Transition colony state with pheromone trigger, file locking, and atomic writes
# Args: new_state, [trigger_pheromone]
# Returns: 0 on success, 1 on failure
transition_state() {
    local new_state="$1"
    local trigger_pheromone="${2:-manual}"

    # Acquire file lock to prevent concurrent transitions
    if ! acquire_lock "$COLONY_STATE"; then
        echo "Failed to acquire lock for state transition" >&2
        return 1
    fi

    # Ensure lock is released on exit
    trap release_lock EXIT TERM INT

    # Read current state
    local current_state=$(get_current_state)

    # Validate transition
    if ! is_valid_transition "$current_state" "$new_state"; then
        echo "Invalid transition: $current_state -> $new_state" >&2
        release_lock
        return 1
    fi

    # Pre-transition checkpoint
    echo "Saving pre-transition checkpoint..."
    if ! save_checkpoint "pre_${current_state}_to_${new_state}"; then
        echo "Pre-transition checkpoint failed" >&2
        release_lock
        return 1
    fi

    # Generate transition metadata
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local checkpoint="checkpoint_$(get_next_checkpoint_number).json"

    # Update COLONY_STATE.json via jq with atomic write
    local temp_file="/tmp/state_transition.$$.tmp"
    if ! jq --arg current "$current_state" \
       --arg new "$new_state" \
       --arg trigger "$trigger_pheromone" \
       --arg timestamp "$timestamp" \
       --arg checkpoint "$checkpoint" \
       '
       .colony_status.state = $new |
       .state_machine.last_transition = $timestamp |
       .state_machine.transitions_count += 1 |
       .state_machine.state_history += [{
         "from": $current,
         "to": $new,
         "trigger": $trigger,
         "timestamp": $timestamp,
         "checkpoint": $checkpoint
       }]
       ' "$COLONY_STATE" > "$temp_file"; then
        echo "Failed to update state with jq" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    # Atomic write to COLONY_STATE.json
    if ! atomic_write_from_file "$COLONY_STATE" "$temp_file"; then
        echo "Failed to write state transition atomically" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    # Cleanup temp file
    rm -f "$temp_file"

    # Archive old history if exceeds 100 entries
    archive_state_history

    # Post-transition checkpoint
    echo "Saving post-transition checkpoint..."
    if ! save_checkpoint "post_${current_state}_to_${new_state}"; then
        echo "Post-transition checkpoint failed" >&2
        release_lock
        return 1
    fi

    # Release lock
    release_lock

    # Reset trap since we've cleaned up
    trap - EXIT TERM INT

    # Echo confirmation message
    echo "State transition: $current_state -> $new_state"
    echo "Trigger: $trigger_pheromone"
    echo "Timestamp: $timestamp"

    return 0
}

# Emit CHECKIN pheromone for phase boundary Queen notification
# Args: phase_number
# Returns: 0 on success, 1 on failure
emit_checkin_pheromone() {
    local phase="$1"
    local pheromone_id="checkin_$(date +%s)"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local pheromones_file=".aether/data/pheromones.json"

    # Add CHECKIN pheromone to pheromones.json
    jq --arg id "$pheromone_id" \
       --arg phase "$phase" \
       --arg timestamp "$timestamp" \
       --arg context "Phase boundary - awaiting Queen review" \
       '.active_pheromones += [{
         "id": $id,
         "type": "CHECKIN",
         "strength": 1.0,
         "created_at": $timestamp,
         "decay_rate": null,
         "metadata": {
           "source": "colony",
           "phase": $phase,
           "context": $context
         }
       }]' "$pheromones_file" > /tmp/pheromones.tmp

    atomic_write_from_file "$pheromones_file" /tmp/pheromones.tmp
    rm -f /tmp/pheromones.tmp

    echo "CHECKIN pheromone emitted for phase $phase"

    return 0
}

# Archive state history to memory system if exceeds MAX_HISTORY entries
# Returns: 0 on success (no-op if history <= MAX_HISTORY)
archive_state_history() {
    local MAX_HISTORY=100
    local history_length=$(jq -r '.state_machine.state_history | length' "$COLONY_STATE")

    # Only archive if history exceeds MAX_HISTORY
    if [ "$history_length" -gt "$MAX_HISTORY" ]; then
        echo "State history exceeds $MAX_HISTORY entries ($history_length), archiving..."

        # Extract full history
        local full_history=$(jq -r '.state_machine.state_history' "$COLONY_STATE")

        # Create archive entry
        local archive_entry=$(echo "$full_history" | jq -c --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" '{
          type: "state_history_archive",
          content: .,
          timestamp: $timestamp,
          metadata: {
            source: "state_machine",
            entries: length
          }
        }')

        # Add to Working Memory (if memory-ops.sh exists)
        if [ -f ".aether/utils/memory-ops.sh" ]; then
            source .aether/utils/memory-ops.sh
            echo "$archive_entry" | jq -c '.' | \
                add_working_memory_item "state_history_archive" 0.3
            echo "State history archived to Working Memory"
        else
            echo "Warning: memory-ops.sh not found, trimming history without archive"
        fi

        # Keep only last 100 entries in state
        local temp_file="/tmp/state_archive.$$.tmp"
        if ! jq '.state_machine.state_history = .state_machine.state_history[-100:]' \
           "$COLONY_STATE" > "$temp_file"; then
            echo "Failed to trim state history" >&2
            rm -f "$temp_file"
            return 1
        fi

        if ! atomic_write_from_file "$COLONY_STATE" "$temp_file"; then
            echo "Failed to write trimmed state history" >&2
            rm -f "$temp_file"
            return 1
        fi

        rm -f "$temp_file"
        echo "State history trimmed: $history_length entries -> 100 entries"
    fi

    return 0
}

# Export functions for use in other scripts
export -f get_current_state get_valid_states is_valid_state
export -f is_valid_transition validate_transition
export -f get_next_checkpoint_number transition_state archive_state_history
export -f emit_checkin_pheromone
