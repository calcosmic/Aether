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

# Check if phase boundary conditions are met
# Returns: 0 if at boundary, 1 if not
# Note: For Phase 5, this is infrastructure-only. Actual detection will be in Phase 6+
check_phase_boundary() {
    local current_state=$(get_current_state)
    local current_phase=$(jq -r '.colony_status.current_phase' "$COLONY_STATE")

    # Phase boundary = EXECUTING → VERIFYING transition
    # Trigger: All phase tasks complete
    if [ "$current_state" = "EXECUTING" ]; then
        # Placeholder infrastructure for phase boundary detection
        # Actual detection will be implemented in Phase 6+ when Worker Ants execute phases
        # For now, provide structure for Queen check-ins
        return 1  # Not at boundary (infrastructure only)
    fi

    # Infrastructure placeholder for phase_tasks_complete check
    # This will be implemented when Worker Ants execute phases
    return 1
}

# Set queen_checkin status and pause colony for Queen review
# Args: phase_number
# Returns: 0 on success, 1 on failure
await_queen_decision() {
    local phase="$1"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Set queen_checkin status
    jq --arg phase "$phase" \
       --arg timestamp "$timestamp" \
       '.colony_status.queen_checkin = {
         "phase": $phase,
         "status": "awaiting_review",
         "timestamp": $timestamp,
         "queen_decision": null
       }' "$COLONY_STATE" > /tmp/state.tmp

    atomic_write_from_file "$COLONY_STATE" /tmp/state.tmp
    rm -f /tmp/state.tmp

    # Display check-in message
    echo ""
    echo "COLONY CHECK-IN: Phase $phase complete"
    echo "   Review with: /ant:phase $phase"
    echo "   Options: /ant:continue, /ant:adjust, /ant:execute {phase}"
    echo ""

    # Adapt next phase from memory
    echo "Adapting next phase from memory..."
    adapt_next_phase_from_memory "$phase"
    echo ""

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

# Helper function to generate random strings for pheromone IDs
# Args: length (default 6)
# Returns: Random alphanumeric string
random_string() {
    local length=${1:-6}
    LC_ALL=C tr -dc 'a-zA-Z0-9' < /dev/urandom | fold -w "$length" | head -n 1
}

# Adapt next phase from previous phase memory using high-confidence patterns
# Args: current_phase (optional, defaults to colony current phase)
# Returns: 0 on success, 1 on failure
adapt_next_phase_from_memory() {
    local current_phase="${1:-$(jq -r '.colony_status.current_phase // 1' "$COLONY_STATE")}"
    local next_phase=$((current_phase + 1))
    local memory_file=".aether/data/memory.json"
    local pheromones_file=".aether/data/pheromones.json"

    echo "Adapting next phase $next_phase from previous phase $current_phase..."

    # Check if memory file exists
    if [ ! -f "$memory_file" ]; then
        echo "Memory file not found. Skipping adaptation."
        return 0
    fi

    # Read previous phase compressed memory from short-term
    local phase_memory=$(jq -r "
        .short_term_memory.sessions[] |
        select(.phase == $current_phase) |
        .compressed_content
    " "$memory_file" 2>/dev/null)

    if [ -n "$phase_memory" ]; then
        echo "  Found compressed memory from phase $current_phase"
    fi

    # Read high-confidence patterns from long-term (confidence > 0.7)
    local patterns_json=$(jq -c "
        .long_term_memory.patterns[] |
        select(.confidence > 0.7) |
        select(.metadata.related_phases[]? == $current_phase)
    " "$memory_file" 2>/dev/null)

    if [ -z "$patterns_json" ]; then
        echo "  No high-confidence patterns found for phase $current_phase"
        return 0
    fi

    echo "  Found high-confidence patterns from previous phase"

    # Extract pattern types
    local focus_areas=$(echo "$patterns_json" | jq -r 'select(.type == "focus_preference") | .pattern')
    local constraints=$(echo "$patterns_json" | jq -r 'select(.type == "constraint") | .pattern')
    local success_patterns=$(echo "$patterns_json" | jq -r 'select(.type == "success_pattern") | .pattern')
    local failure_patterns=$(echo "$patterns_json" | jq -r 'select(.type == "failure_pattern") | .pattern')

    # Count patterns
    local focus_count=$(echo "$focus_areas" | grep -c . 2>/dev/null || echo 0)
    local constraint_count=$(echo "$constraints" | grep -c . 2>/dev/null || echo 0)
    local success_count=$(echo "$success_patterns" | grep -c . 2>/dev/null || echo 0)
    local failure_count=$(echo "$failure_patterns" | grep -c . 2>/dev/null || echo 0)

    echo "  Patterns extracted: $focus_count focus, $constraint_count constraints, $success_count successes, $failure_count failures"

    # Emit FOCUS pheromones for high-value areas
    if [ -n "$focus_areas" ]; then
        echo "$focus_areas" | while IFS= read -r area; do
            if [ -n "$area" ]; then
                local pheromone_id="focus_$(date +%s)_$(random_string 6)"
                local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
                local strength=0.8

                # Direct jq update to pheromones.json
                jq --arg id "$pheromone_id" \
                   --arg area "$area" \
                   --arg strength "$strength" \
                   --arg timestamp "$timestamp" \
                   --arg phase "$next_phase" \
                   '.active_pheromones += [{
                     "id": $id,
                     "type": "FOCUS",
                     "strength": ($strength | tonumber),
                     "created_at": $timestamp,
                     "decay_rate": 3600,
                     "metadata": {
                       "source": "memory_adaptation",
                       "phase": $phase,
                       "context": $area
                     }
                   }]' "$pheromones_file" > /tmp/pheromones.tmp

                atomic_write_from_file "$pheromones_file" /tmp/pheromones.tmp
                rm -f /tmp/pheromones.tmp

                echo "  → FOCUS: $area (strength: $strength)"
            fi
        done
    fi

    # Emit REDIRECT pheromones for constraints
    if [ -n "$constraints" ]; then
        echo "$constraints" | while IFS= read -r pattern; do
            if [ -n "$pattern" ]; then
                local pheromone_id="redirect_$(date +%s)_$(random_string 6)"
                local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
                local strength=0.9

                # Direct jq update to pheromones.json
                jq --arg id "$pheromone_id" \
                   --arg pattern "$pattern" \
                   --arg strength "$strength" \
                   --arg timestamp "$timestamp" \
                   --arg phase "$next_phase" \
                   '.active_pheromones += [{
                     "id": $id,
                     "type": "REDIRECT",
                     "strength": ($strength | tonumber),
                     "created_at": $timestamp,
                     "decay_rate": 86400,
                     "metadata": {
                       "source": "memory_adaptation",
                       "phase": $phase,
                       "context": $pattern
                     }
                   }]' "$pheromones_file" > /tmp/pheromones.tmp

                atomic_write_from_file "$pheromones_file" /tmp/pheromones.tmp
                rm -f /tmp/pheromones.tmp

                echo "  → REDIRECT: $pattern (strength: $strength)"
            fi
        done
    fi

    # Store adaptation in colony state
    local adapted_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Convert pattern lists to JSON arrays
    local focus_json=$(echo "$focus_areas" | jq -R -s -c 'split("\n") | map(select(length > 0))')
    local constraints_json=$(echo "$constraints" | jq -R -s -c 'split("\n") | map(select(length > 0))')
    local successes_json=$(echo "$success_patterns" | jq -R -s -c 'split("\n") | map(select(length > 0))')
    local failures_json=$(echo "$failure_patterns" | jq -R -s -c 'split("\n") | map(select(length > 0))')

    # Update colony state with adaptation
    local temp_state="/tmp/state_adapt.$$.tmp"
    if ! jq --arg next "$next_phase" \
       --argjson focus "$focus_json" \
       --argjson constraints "$constraints_json" \
       --argjson successes "$successes_json" \
       --argjson failures "$failures_json" \
       --arg adapted_at "$adapted_at" \
       --arg adapted_from "$current_phase" \
       '
       if .phases.roadmap[$next | tonumber - 1] then
         .phases.roadmap[$next | tonumber - 1].adaptation = {
           "inherited_focus": $focus,
           "inherited_constraints": $constraints,
           "success_patterns": $successes,
           "failure_patterns": $failures,
           "adapted_from": $adapted_from,
           "adapted_at": $adapted_at
         }
       else
         .
       end
       ' "$COLONY_STATE" > "$temp_state"; then
        echo "Failed to update adaptation in colony state" >&2
        rm -f "$temp_state"
        return 1
    fi

    if ! atomic_write_from_file "$COLONY_STATE" "$temp_state"; then
        echo "Failed to write adaptation to colony state" >&2
        rm -f "$temp_state"
        return 1
    fi

    rm -f "$temp_state"

    echo "Adaptation stored for phase $next_phase"
    echo "  Focus areas: $focus_count"
    echo "  Constraints: $constraint_count"
    echo "  Success patterns: $success_count"
    echo "  Failure patterns: $failure_count"

    return 0
}

# Export functions for use in other scripts
export -f get_current_state get_valid_states is_valid_state
export -f is_valid_transition validate_transition
export -f get_next_checkpoint_number transition_state archive_state_history
export -f emit_checkin_pheromone check_phase_boundary await_queen_decision
export -f adapt_next_phase_from_memory random_string
