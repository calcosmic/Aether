#!/bin/bash
# Aether Colony State Loader
# Loads and validates COLONY_STATE.json with file lock protection
#
# Usage:
#   source .aether/utils/state-loader.sh
#   load_colony_state
#   # ... use state ...
#   unload_colony_state
#
# Provides: load_colony_state, unload_colony_state, get_handoff_summary, display_resumption_context

# Aether root detection - use git root if available, otherwise use current directory
if git rev-parse --show-toplevel >/dev/null 2>&1; then
    AETHER_ROOT="$(git rev-parse --show-toplevel)"
else
    AETHER_ROOT="$(pwd)"
fi

SCRIPT_DIR="${SCRIPT_DIR:-$AETHER_ROOT/.aether}"
DATA_DIR="$AETHER_ROOT/.aether/data"

# Initialize lock state before sourcing (file-lock.sh trap needs these)
LOCK_ACQUIRED=${LOCK_ACQUIRED:-false}
CURRENT_LOCK=${CURRENT_LOCK:-""}

# Source shared infrastructure if available
[[ -f "$SCRIPT_DIR/utils/file-lock.sh" ]] && source "$SCRIPT_DIR/utils/file-lock.sh"
[[ -f "$SCRIPT_DIR/utils/error-handler.sh" ]] && source "$SCRIPT_DIR/utils/error-handler.sh"

# State loading globals
LOADED_STATE=""
STATE_LOCK_ACQUIRED=false
HANDOFF_DETECTED=false
HANDOFF_CONTENT=""

# --- load_colony_state ---
# Main loading function that acquires lock, validates, and loads state
# Returns: 0 on success, 1 on failure
# Exports: LOADED_STATE, STATE_LOCK_ACQUIRED, HANDOFF_DETECTED, HANDOFF_CONTENT
load_colony_state() {
    local state_file="$DATA_DIR/COLONY_STATE.json"

    # Check file exists
    if [[ ! -f "$state_file" ]]; then
        if type json_err &>/dev/null; then
            json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}' "Run /ant:init to initialize colony"
        else
            echo '{"ok":false,"error":{"code":"E_FILE_NOT_FOUND","message":"COLONY_STATE.json not found"}}' >&2
        fi
        return 1
    fi

    # Acquire lock
    if type acquire_lock &>/dev/null; then
        if ! acquire_lock "$state_file"; then
            if type json_err &>/dev/null; then
                json_err "$E_LOCK_FAILED" "Failed to acquire state lock" '{"file":"COLONY_STATE.json"}' "Wait for other operations to complete"
            else
                echo '{"ok":false,"error":{"code":"E_LOCK_FAILED","message":"Failed to acquire state lock"}}' >&2
            fi
            return 1
        fi
    else
        # Locking unavailable - proceed with warning
        if type json_warn &>/dev/null; then
            json_warn "W_DEGRADED" "File locking unavailable - proceeding without lock"
        fi
    fi

    # Validate state before loading
    local validation
    validation=$(bash "$SCRIPT_DIR/aether-utils.sh" validate-state colony 2>/dev/null)
    if ! echo "$validation" | jq -e '.result.pass' >/dev/null 2>&1; then
        # Validation failed - release lock and report error
        if type release_lock &>/dev/null; then
            release_lock
        fi

        local validation_error
        validation_error=$(echo "$validation" | jq -r '.result.checks[] | select(. != "pass") | .' 2>/dev/null | head -1 || echo "unknown validation error")

        if type json_err &>/dev/null; then
            json_err "$E_VALIDATION_FAILED" "State validation failed" "{\"details\":\"$validation_error\"}" "Check COLONY_STATE.json for errors or run /ant:init to reset"
        else
            echo "{\"ok\":false,\"error\":{\"code\":\"E_VALIDATION_FAILED\",\"message\":\"State validation failed: $validation_error\"}}" >&2
        fi
        return 1
    fi

    # Read state into variable
    local state
    state=$(cat "$state_file")
    if [[ -z "$state" ]]; then
        if type release_lock &>/dev/null; then
            release_lock
        fi
        if type json_err &>/dev/null; then
            json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json is empty" '{"file":"COLONY_STATE.json"}' "Run /ant:init to initialize colony"
        else
            echo '{"ok":false,"error":{"code":"E_FILE_NOT_FOUND","message":"COLONY_STATE.json is empty"}}' >&2
        fi
        return 1
    fi

    # Check for handoff document
    local handoff_file="$AETHER_ROOT/.aether/HANDOFF.md"
    if [[ -f "$handoff_file" ]]; then
        HANDOFF_DETECTED=true
        HANDOFF_CONTENT=$(cat "$handoff_file")
    else
        HANDOFF_DETECTED=false
        HANDOFF_CONTENT=""
    fi

    # Export loaded state
    LOADED_STATE="$state"
    STATE_LOCK_ACQUIRED=true

    return 0
}

# --- unload_colony_state ---
# Cleanup function that releases lock and unsets state
# Should be called when done with state
unload_colony_state() {
    if [[ "$STATE_LOCK_ACQUIRED" == "true" ]]; then
        if type release_lock &>/dev/null; then
            release_lock
        fi
        STATE_LOCK_ACQUIRED=false
    fi
    LOADED_STATE=""
    HANDOFF_DETECTED=false
    HANDOFF_CONTENT=""
}

# --- get_handoff_summary ---
# Extract brief summary from handoff content
# Returns: "Phase X - Name" format or empty string
get_handoff_summary() {
    if [[ "$HANDOFF_DETECTED" != "true" || -z "$HANDOFF_CONTENT" ]]; then
        echo ""
        return 1
    fi

    # Parse HANDOFF.md for Phase line
    # Format: "Phase: X - Name" or "## Phase X: Name"
    local phase_line
    phase_line=$(echo "$HANDOFF_CONTENT" | grep -E "^(Phase:|## Phase)" | head -1)

    if [[ -n "$phase_line" ]]; then
        # Extract phase number and name
        local phase_num
        local phase_name
        phase_num=$(echo "$phase_line" | grep -oE "[0-9]+" | head -1)
        phase_name=$(echo "$phase_line" | sed -E 's/.*Phase:?[[:space:]]*[0-9]+[[:space:]-]+//' | sed 's/^[[:space:]]*//')

        if [[ -n "$phase_num" && -n "$phase_name" ]]; then
            echo "Phase $phase_num - $phase_name"
        elif [[ -n "$phase_num" ]]; then
            echo "Phase $phase_num"
        else
            echo "Resuming colony session"
        fi
    else
        echo "Resuming colony session"
    fi
}

# --- display_resumption_context ---
# Show brief resume message and clean up handoff
# Returns: 0 on success
display_resumption_context() {
    if [[ "$HANDOFF_DETECTED" != "true" ]]; then
        return 0
    fi

    local summary
    summary=$(get_handoff_summary)

    if [[ -n "$summary" ]]; then
        echo "Resuming: $summary"
    fi

    # Remove handoff file after successful display
    local handoff_file="$AETHER_ROOT/.aether/HANDOFF.md"
    if [[ -f "$handoff_file" ]]; then
        rm -f "$handoff_file"
    fi

    HANDOFF_DETECTED=false
    HANDOFF_CONTENT=""

    return 0
}

# --- ensure_cleanup ---
# Internal function to ensure lock is released on script exit
_ensure_cleanup() {
    if [[ "$STATE_LOCK_ACQUIRED" == "true" ]]; then
        unload_colony_state
    fi
}

# Register cleanup on exit
# Note: This works alongside file-lock.sh's trap
_original_trap_exit() {
    _ensure_cleanup
}

# Export functions for use in other scripts
export -f load_colony_state unload_colony_state get_handoff_summary display_resumption_context
export -f _ensure_cleanup _original_trap_exit
export LOADED_STATE STATE_LOCK_ACQUIRED HANDOFF_DETECTED HANDOFF_CONTENT
