# =============================================================================
# DEPRECATED — This script has been superseded by the Go binary (aether CLI).
# All functionality is now available via: aether <subcommand>
# Do NOT modify this file — it is retained for reference only.
# See: cmd/ (Go source) | Run: aether --help
# =============================================================================
#
#!/bin/bash
# State API facade -- centralized COLONY_STATE.json access
# Provides: _state_read, _state_write, _state_read_field, _state_mutate, _state_migrate,
#           _colony_vital_signs
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
    # Usage: state-mutate [--arg NAME VALUE] [--argjson NAME VALUE] '<jq_expression>'
    # Supports jq --arg, --argjson, --slurpfile, --rawfile flags (forwarded to jq)
    # Acquires lock, creates backup, applies jq, validates, writes atomically
    # Returns: json_ok with mutated:true, or json_err on failure

    # Parse jq flags (--arg, --argjson, --slurpfile, --rawfile) from arguments
    # The jq expression is always the last argument (after all flags)
    local sm_jq_flags=()
    local sm_expr=""
    local i=0
    local args=("$@")

    while [[ $i -lt ${#args[@]} ]]; do
        case "${args[$i]}" in
            --arg|--argjson|--slurpfile|--rawfile)
                # Flag requires a name and value — consume next two args
                sm_jq_flags+=("${args[$i]}" "${args[$((i+1))]}" "${args[$((i+2))]}")
                i=$((i + 3))
                ;;
            *)
                # Last argument is the jq expression
                sm_expr="${args[$i]}"
                i=$((i + 1))
                ;;
        esac
    done

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

    # Apply jq expression to current state (with forwarded flags)
    sm_updated=$(jq ${sm_jq_flags[@]+"${sm_jq_flags[@]}"} "$sm_expr" "$sm_state_file" 2>/dev/null) || {
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

# ============================================================================
# _colony_vital_signs
# Compute colony health metrics from existing data files
# Usage: colony-vital-signs
# Returns: JSON with build_velocity, error_rate, signal_health, memory_pressure,
#          colony_age_hours, and overall_health (0-100)
# Gracefully degrades: missing files produce zero/default values
# ============================================================================
_colony_vital_signs() {
    local cvs_state_file="$DATA_DIR/COLONY_STATE.json"
    local cvs_midden_file="$COLONY_DATA_DIR/midden/midden.json"
    local cvs_phero_file="$COLONY_DATA_DIR/pheromones.json"
    local cvs_session_file="$COLONY_DATA_DIR/session.json"

    # --- Compute 24h window boundary ---
    local cvs_now
    cvs_now=$(date -u +%s 2>/dev/null || echo "0")
    local cvs_window_start=$(( cvs_now - 86400 ))

    # ---- build_velocity: count phase_completed events in last 24h ----
    local cvs_phases_per_day=0
    if [[ -f "$cvs_state_file" ]]; then
        cvs_phases_per_day=$(jq --argjson win "$cvs_window_start" '
            [.events[]? |
             select(. != null) |
             select(test("\\|phase_completed\\|")) |
             capture("^(?P<ts>[^|]+)\\|") |
             .ts |
             gsub("[TZ:-]"; " ") |
             split(" ") |
             if length >= 6 then
               (.[0:6] | join(" ")) |
               # convert to comparable string for ordering -- full ISO compare
               . as $s | $s
             else . end
            ] | length
        ' "$cvs_state_file" 2>/dev/null || echo "0")

        # Simpler approach: use string comparison on ISO timestamps
        # Compute the 24h-ago timestamp as ISO string
        local cvs_cutoff_iso
        cvs_cutoff_iso=$(date -u -r "$cvs_window_start" '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null \
            || date -u -d "@$cvs_window_start" '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null \
            || echo "")

        if [[ -n "$cvs_cutoff_iso" ]]; then
            cvs_phases_per_day=$(jq --arg cutoff "$cvs_cutoff_iso" '
                [.events[]? |
                 select(. != null and (type == "string")) |
                 select(test("\\|phase_completed\\|")) |
                 split("|") | .[0] |
                 select(. >= $cutoff)
                ] | length
            ' "$cvs_state_file" 2>/dev/null || echo "0")
        fi
    fi
    # Normalize: ensure integer
    cvs_phases_per_day=$(( cvs_phases_per_day + 0 )) 2>/dev/null || cvs_phases_per_day=0

    # Determine trend (simple heuristic: any builds = steady, 0 = idle)
    local cvs_bv_trend="idle"
    [[ "$cvs_phases_per_day" -ge 1 ]] && cvs_bv_trend="steady"
    [[ "$cvs_phases_per_day" -ge 3 ]] && cvs_bv_trend="accelerating"

    # ---- error_rate: unreviewed midden entries in last 24h ----
    local cvs_errors_per_day=0
    local cvs_err_status="clean"
    if [[ -f "$cvs_midden_file" ]]; then
        local cvs_cutoff_iso_err
        cvs_cutoff_iso_err=$(date -u -r "$cvs_window_start" '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null \
            || date -u -d "@$cvs_window_start" '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null \
            || echo "")

        if [[ -n "$cvs_cutoff_iso_err" ]]; then
            cvs_errors_per_day=$(jq --arg cutoff "$cvs_cutoff_iso_err" '
                [(.entries // [])[]? |
                 select(.reviewed == false or .reviewed == null) |
                 select((.timestamp // "") >= $cutoff)
                ] | length
            ' "$cvs_midden_file" 2>/dev/null || echo "0")
        else
            # Fallback: count all unreviewed
            cvs_errors_per_day=$(jq '
                [(.entries // [])[]? | select(.reviewed == false or .reviewed == null)] | length
            ' "$cvs_midden_file" 2>/dev/null || echo "0")
        fi
    fi
    cvs_errors_per_day=$(( cvs_errors_per_day + 0 )) 2>/dev/null || cvs_errors_per_day=0

    if [[ "$cvs_errors_per_day" -eq 0 ]]; then
        cvs_err_status="clean"
    elif [[ "$cvs_errors_per_day" -le 2 ]]; then
        cvs_err_status="nominal"
    elif [[ "$cvs_errors_per_day" -le 5 ]]; then
        cvs_err_status="elevated"
    else
        cvs_err_status="critical"
    fi

    # ---- signal_health: count active pheromones ----
    local cvs_active_count=0
    local cvs_sig_status="dormant"
    if [[ -f "$cvs_phero_file" ]]; then
        cvs_active_count=$(jq '
            [.signals[]? | select(.active == true)] | length
        ' "$cvs_phero_file" 2>/dev/null || echo "0")
    fi
    cvs_active_count=$(( cvs_active_count + 0 )) 2>/dev/null || cvs_active_count=0

    if [[ "$cvs_active_count" -eq 0 ]]; then
        cvs_sig_status="dormant"
    elif [[ "$cvs_active_count" -le 3 ]]; then
        cvs_sig_status="guided"
    else
        cvs_sig_status="active"
    fi

    # ---- memory_pressure: count instincts ----
    local cvs_instinct_count=0
    local cvs_mem_status="empty"
    if [[ -f "$cvs_state_file" ]]; then
        # instincts may be a JSON string (serialized array) or a real array
        local cvs_raw_instincts
        cvs_raw_instincts=$(jq -r '.memory.instincts // "[]"' "$cvs_state_file" 2>/dev/null || echo "[]")
        # Handle both string-encoded and native array
        cvs_instinct_count=$(echo "$cvs_raw_instincts" | jq -r 'if type == "string" then (. | fromjson | length) elif type == "array" then length else 0 end' 2>/dev/null || echo "0")
    fi
    cvs_instinct_count=$(( cvs_instinct_count + 0 )) 2>/dev/null || cvs_instinct_count=0

    if [[ "$cvs_instinct_count" -eq 0 ]]; then
        cvs_mem_status="empty"
    elif [[ "$cvs_instinct_count" -le 5 ]]; then
        cvs_mem_status="growing"
    elif [[ "$cvs_instinct_count" -le 15 ]]; then
        cvs_mem_status="healthy"
    else
        cvs_mem_status="rich"
    fi

    # ---- colony_age_hours: hours since initialized_at ----
    local cvs_age_hours=0
    if [[ -f "$cvs_state_file" ]]; then
        local cvs_init_at
        cvs_init_at=$(jq -r '.initialized_at // empty' "$cvs_state_file" 2>/dev/null || echo "")
        if [[ -n "$cvs_init_at" ]]; then
            local cvs_init_ts
            cvs_init_ts=$(date -u -j -f '%Y-%m-%dT%H:%M:%SZ' "$cvs_init_at" '+%s' 2>/dev/null \
                || date -u -d "$cvs_init_at" '+%s' 2>/dev/null \
                || echo "0")
            if [[ "$cvs_init_ts" -gt 0 && "$cvs_now" -gt "$cvs_init_ts" ]]; then
                cvs_age_hours=$(( (cvs_now - cvs_init_ts) / 3600 ))
            fi
        fi
    fi

    # ---- overall_health: weighted 0-100 score ----
    # Components (max points each):
    #   recent builds (+30): has at least one phase_completed in 24h
    #   low errors (+30):    zero unreviewed errors in 24h
    #   signals exist (+20): at least one active pheromone
    #   instincts growing (+20): at least one instinct
    local cvs_score=0
    [[ "$cvs_phases_per_day" -ge 1 ]] && cvs_score=$(( cvs_score + 30 ))
    [[ "$cvs_errors_per_day" -eq 0 ]] && cvs_score=$(( cvs_score + 30 ))
    [[ "$cvs_active_count" -ge 1 ]] && cvs_score=$(( cvs_score + 20 ))
    [[ "$cvs_instinct_count" -ge 1 ]] && cvs_score=$(( cvs_score + 20 ))
    [[ "$cvs_score" -gt 100 ]] && cvs_score=100

    json_ok "$(jq -n \
        --argjson phases_per_day "$cvs_phases_per_day" \
        --arg bv_trend "$cvs_bv_trend" \
        --argjson errors_per_day "$cvs_errors_per_day" \
        --arg err_status "$cvs_err_status" \
        --argjson active_count "$cvs_active_count" \
        --arg sig_status "$cvs_sig_status" \
        --argjson instinct_count "$cvs_instinct_count" \
        --arg mem_status "$cvs_mem_status" \
        --argjson age_hours "$cvs_age_hours" \
        --argjson overall_health "$cvs_score" \
        '{
            build_velocity:   {phases_per_day: $phases_per_day, trend: $bv_trend},
            error_rate:       {errors_per_day: $errors_per_day, status: $err_status},
            signal_health:    {active_count: $active_count, status: $sig_status},
            memory_pressure:  {instinct_count: $instinct_count, status: $mem_status},
            colony_age_hours: $age_hours,
            overall_health:   $overall_health
        }')"
}
