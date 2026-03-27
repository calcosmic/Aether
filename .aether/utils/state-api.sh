#!/bin/bash
# State API facade -- centralized COLONY_STATE.json access
# Provides: _state_read, _state_write, _state_read_field, _state_mutate, _state_migrate
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write, acquire_lock,
# release_lock, LOCK_DIR, DATA_DIR, SCRIPT_DIR, error constants) is available.

_state_read() {
    # Read full COLONY_STATE.json and return via json_ok
    # Usage: state-read
    # No lock needed for reads (jq is atomic on single files)
    # Returns: json_ok with full state, or json_err on missing/invalid file

    sr_state_file="$DATA_DIR/COLONY_STATE.json"

    if [[ ! -f "$sr_state_file" ]]; then
        json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    fi

    sr_content=$(cat "$sr_state_file" 2>/dev/null) || {
        json_err "$E_FILE_NOT_FOUND" "Failed to read COLONY_STATE.json"
    }

    if ! echo "$sr_content" | jq -e . >/dev/null 2>&1; then
        json_err "$E_JSON_INVALID" "COLONY_STATE.json contains invalid JSON"
    fi

    json_ok "$sr_content"
}

_state_read_field() {
    # Read a specific jq field path from COLONY_STATE.json
    # Usage: state-read-field <jq_path>
    # For internal callers: outputs raw value to stdout (no json_ok wrapper)
    # For subcommand entry: case block wraps in json_ok
    # Returns empty string + exit 0 for missing field (callers check emptiness)

    srf_field="${1:-}"

    if [[ -z "$srf_field" ]]; then
        json_err "$E_VALIDATION_FAILED" "state-read-field requires a jq field path argument"
    fi

    srf_state_file="$DATA_DIR/COLONY_STATE.json"

    if [[ ! -f "$srf_state_file" ]]; then
        json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    fi

    # Extract the field value (raw output, no quotes around strings)
    srf_value=$(jq -r "$srf_field // empty" "$srf_state_file" 2>/dev/null) || srf_value=""

    echo "$srf_value"
}

_state_write() {
    # Write COLONY_STATE.json through a locked, validated, atomic path
    # Usage: state-write '<json>'
    #    or: cat state.json | state-write
    # Refactored from inline state-write case block for reuse
    # Validates JSON, acquires lock, creates backup, writes atomically

    sw_content="${1:-}"
    if [[ -z "$sw_content" ]]; then
        sw_content=$(cat)
    fi

    # Validate JSON
    if ! echo "$sw_content" | jq -e . >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
        json_err "$E_JSON_INVALID" "state-write received invalid JSON"
    fi

    sw_state_file="$DATA_DIR/COLONY_STATE.json"

    # Acquire lock (colony-level, not hub-level)
    acquire_lock "$sw_state_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on COLONY_STATE.json"

    # Create backup before writing
    if [[ -f "$sw_state_file" ]]; then
        if ! create_backup "$sw_state_file"; then
            _aether_log_error "Could not create backup of colony state before writing"
        fi
    fi

    # Write atomically; release lock on failure
    atomic_write "$sw_state_file" "$sw_content" || {
        release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
        json_err "$E_UNKNOWN" "Failed to write COLONY_STATE.json"
    }
    release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held

    json_ok '{"written":true}'
}

_state_mutate() {
    # Read-modify-write COLONY_STATE.json with a jq expression
    # Usage: state-mutate '<jq_expression>'
    # Acquires lock, creates backup, applies jq, validates, writes atomically
    # Returns: json_ok with mutated:true, or json_err on failure

    sm_expr="${1:-}"

    if [[ -z "$sm_expr" ]]; then
        json_err "$E_VALIDATION_FAILED" "state-mutate requires a jq expression argument"
    fi

    sm_state_file="$DATA_DIR/COLONY_STATE.json"

    if [[ ! -f "$sm_state_file" ]]; then
        json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    fi

    # Acquire lock for safe read-modify-write
    acquire_lock "$sm_state_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on COLONY_STATE.json"

    # Create backup before mutation
    if type create_backup &>/dev/null; then
        if ! create_backup "$sm_state_file"; then
            _aether_log_error "Could not create backup of colony state before mutation"
        fi
    fi

    # Apply jq expression to current state
    sm_updated=$(jq "$sm_expr" "$sm_state_file" 2>/dev/null) || {
        release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
        json_err "$E_JSON_INVALID" "jq expression failed: $sm_expr"
    }

    # Validate the result is valid JSON
    if [[ -z "$sm_updated" ]] || ! echo "$sm_updated" | jq -e . >/dev/null 2>&1; then
        release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
        json_err "$E_JSON_INVALID" "state-mutate produced invalid JSON"
    fi

    # Write atomically
    atomic_write "$sm_state_file" "$sm_updated" || {
        release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
        json_err "$E_UNKNOWN" "Failed to write mutated COLONY_STATE.json"
    }

    release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held

    json_ok '{"mutated":true}'
}

_state_migrate() {
    # Schema migration helper: auto-upgrades pre-3.0 state files to v3.0
    # Additive only (never removes fields) -- idempotent and safe for concurrent access
    # Moved from validate-state case block for reuse

    sm_state_file="${1:-}"
    [[ -f "$sm_state_file" ]] || return 0

    # First: verify file is parseable JSON at all
    if ! jq -e . "$sm_state_file" >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
        # Corrupt state file -- backup and error
        if type create_backup &>/dev/null; then
            if ! create_backup "$sm_state_file"; then
                _aether_log_error "Could not create backup of corrupted COLONY_STATE.json"
            fi
        fi
        json_err "$E_JSON_INVALID" \
          "COLONY_STATE.json is corrupted (invalid JSON). A backup was saved in .aether/data/backups/. Try: run /ant:init to reset colony state."
    fi

    sm_current_version=$(jq -r '.version // "1.0"' "$sm_state_file" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet

    if [[ "$sm_current_version" != "3.0" ]]; then
        sm_lock_held=false
        # Skip lock acquisition when caller already holds the state lock
        if [[ "${AETHER_STATE_LOCKED:-false}" != "true" ]] && type acquire_lock &>/dev/null; then
            acquire_lock "$sm_state_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on COLONY_STATE.json for migration"
            sm_lock_held=true
        fi

        # Add missing v3.0 fields (additive only -- idempotent and safe for concurrent access)
        sm_updated=$(jq '
            .version = "3.0" |
            if .signals == null then .signals = [] else . end |
            if .graveyards == null then .graveyards = [] else . end |
            if .events == null then .events = [] else . end
        ' "$sm_state_file" 2>/dev/null) || {  # SUPPRESS:OK -- read-default: file may not exist yet
            [[ "$sm_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
            json_err "$E_JSON_INVALID" "Failed to migrate COLONY_STATE.json"
        }

        if [[ -n "$sm_updated" ]]; then
            atomic_write "$sm_state_file" "$sm_updated" || {
                [[ "$sm_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
                json_err "$E_JSON_INVALID" "Failed to write migrated COLONY_STATE.json"
            }
            # Notify user of migration (auto-migrate + notify pattern)
            printf '{"ok":true,"warning":"W_MIGRATED","message":"Migrated colony state from v%s to v3.0"}\n' "$sm_current_version" >&2
        fi

        [[ "$sm_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    fi
}
