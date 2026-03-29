#!/bin/bash
# Flag utility functions — extracted from aether-utils.sh
# Provides: _flag_add, _flag_check_blockers, _flag_resolve, _flag_acknowledge, _flag_list, _flag_auto_resolve
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, json_warn, atomic_write, acquire_lock,
# release_lock, feature_enabled, LOCK_DIR, DATA_DIR, SCRIPT_DIR, error constants) is available.

_flag_add() {
    # Add a project-specific flag (blocker, issue, or note)
    # Usage: flag-add <type> <title> <description> [source] [phase]
    # Types: blocker (critical, blocks advancement), issue (high, warning), note (low, info)
    type="${1:-issue}"
    title="${2:-}"
    desc="${3:-}"
    source="${4:-manual}"
    phase="${5:-null}"
    [[ -z "$title" ]] && json_err "$E_VALIDATION_FAILED" "Usage: flag-add <type> <title> <description> [source] [phase]"

    mkdir -p "$COLONY_DATA_DIR"
    flags_file="$COLONY_DATA_DIR/flags.json"

    if [[ ! -f "$flags_file" ]]; then
      atomic_write "$flags_file" '{"version":1,"flags":[]}' || json_err "$E_UNKNOWN" "Failed to initialize flags file"
    fi

    id="flag_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
    if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
      json_warn "W_DEGRADED" "File locking disabled - proceeding without lock: $(type _feature_reason &>/dev/null && _feature_reason file_locking || echo 'unknown')"
    else
      acquire_lock "$flags_file" || {
        if type json_err &>/dev/null; then
          json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
        else
          echo '{"ok":false,"error":"Failed to acquire lock on flags.json"}' >&2
          exit 1
        fi
      }
      # Ensure lock is always released on exit (BUG-002 fix)
      trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
    fi

    # Map type to severity
    case "$type" in
      blocker)  severity="critical" ;;
      issue)    severity="high" ;;
      note)     severity="low" ;;
      *)        severity="medium" ;;
    esac

    # Handle phase as number or null
    if [[ "$phase" =~ ^[0-9]+$ ]]; then
      phase_jq="$phase"
    else
      phase_jq="null"
    fi

    updated=$(jq --arg id "$id" --arg type "$type" --arg sev "$severity" \
      --arg title "$title" --arg desc "$desc" --arg source "$source" \
      --argjson phase "$phase_jq" --arg ts "$ts" '
      .flags += [{
        id: $id,
        type: $type,
        severity: $sev,
        title: $title,
        description: $desc,
        source: $source,
        phase: $phase,
        created_at: $ts,
        acknowledged_at: null,
        resolved_at: null,
        resolution: null,
        auto_resolve_on: (if $type == "blocker" and ($source | test("chaos") | not) then "build_pass" else null end)
      }]
    ' "$flags_file") || { json_err "$E_JSON_INVALID" "Failed to add flag"; }

    atomic_write "$flags_file" "$updated"
    trap - EXIT
    release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    json_ok "$(jq -n --arg id "$id" --arg type "$type" --arg severity "$severity" \
      '{id: $id, type: $type, severity: $severity}')"
}

_flag_check_blockers() {
    # Count unresolved blockers for the current phase
    # Usage: flag-check-blockers [phase]
    phase="${1:-}"
    flags_file="$COLONY_DATA_DIR/flags.json"

    if [[ ! -f "$flags_file" ]]; then
      json_ok '{"blockers":0,"issues":0,"notes":0}'
      exit 0
    fi

    if [[ -n "$phase" && "$phase" =~ ^[0-9]+$ ]]; then
      # Filter by phase
      result=$(jq --argjson phase "$phase" '{
        blockers: [.flags[] | select(.type == "blocker" and .resolved_at == null and (.phase == $phase or .phase == null))] | length,
        issues: [.flags[] | select(.type == "issue" and .resolved_at == null and (.phase == $phase or .phase == null))] | length,
        notes: [.flags[] | select(.type == "note" and .resolved_at == null and (.phase == $phase or .phase == null))] | length
      }' "$flags_file")
    else
      # All unresolved
      result=$(jq '{
        blockers: [.flags[] | select(.type == "blocker" and .resolved_at == null)] | length,
        issues: [.flags[] | select(.type == "issue" and .resolved_at == null)] | length,
        notes: [.flags[] | select(.type == "note" and .resolved_at == null)] | length
      }' "$flags_file")
    fi

    json_ok "$result"
}

_flag_resolve() {
    # Resolve a flag with optional resolution message
    # Usage: flag-resolve <flag_id> [resolution_message]
    flag_id="${1:-}"
    resolution="${2:-Resolved}"
    [[ -z "$flag_id" ]] && json_err "$E_VALIDATION_FAILED" "Usage: flag-resolve <flag_id> [resolution_message]"

    flags_file="$COLONY_DATA_DIR/flags.json"
    [[ ! -f "$flags_file" ]] && json_err "$E_FILE_NOT_FOUND" "No flags file found"

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
    if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
      json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
    else
      acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
      trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
    fi

    updated=$(jq --arg id "$flag_id" --arg res "$resolution" --arg ts "$ts" '
      .flags = [.flags[] | if .id == $id then
        .resolved_at = $ts |
        .resolution = $res
      else . end]
    ' "$flags_file") || {
      json_err "$E_JSON_INVALID" "Failed to resolve flag"
    }

    atomic_write "$flags_file" "$updated"
    trap - EXIT
    release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    json_ok "$(jq -n --arg id "$flag_id" '{resolved: $id}')"
}

_flag_acknowledge() {
    # Acknowledge a flag (issue continues but noted)
    # Usage: flag-acknowledge <flag_id>
    flag_id="${1:-}"
    [[ -z "$flag_id" ]] && json_err "$E_VALIDATION_FAILED" "Usage: flag-acknowledge <flag_id>"

    flags_file="$COLONY_DATA_DIR/flags.json"
    [[ ! -f "$flags_file" ]] && json_err "$E_FILE_NOT_FOUND" "No flags file found"

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
    if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
      json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
    else
      acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
      trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
    fi

    updated=$(jq --arg id "$flag_id" --arg ts "$ts" '
      .flags = [.flags[] | if .id == $id then
        .acknowledged_at = $ts
      else . end]
    ' "$flags_file") || {
      json_err "$E_JSON_INVALID" "Failed to acknowledge flag"
    }

    atomic_write "$flags_file" "$updated"
    trap - EXIT
    release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    json_ok "$(jq -n --arg id "$flag_id" '{acknowledged: $id}')"
}

_flag_list() {
    # List flags, optionally filtered
    # Usage: flag-list [--all] [--type blocker|issue|note] [--phase N]
    flags_file="$COLONY_DATA_DIR/flags.json"
    show_all="false"
    filter_type=""
    filter_phase=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --all) show_all="true"; shift ;;
        --type) filter_type="$2"; shift 2 ;;
        --phase) filter_phase="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    if [[ ! -f "$flags_file" ]]; then
      json_ok '{"flags":[],"count":0}'
      exit 0
    fi

    # Build jq filter
    jq_filter='.flags'

    if [[ "$show_all" != "true" ]]; then
      jq_filter+=' | [.[] | select(.resolved_at == null)]'
    fi

    if [[ -n "$filter_type" ]]; then
      jq_filter+=" | [.[] | select(.type == \"$filter_type\")]"
    fi

    if [[ -n "$filter_phase" && "$filter_phase" =~ ^[0-9]+$ ]]; then
      jq_filter+=" | [.[] | select(.phase == $filter_phase or .phase == null)]"
    fi

    result=$(jq "{flags: ($jq_filter), count: ($jq_filter | length)}" "$flags_file")
    json_ok "$result"
}

_flag_auto_resolve() {
    # Auto-resolve flags based on trigger (e.g., build_pass)
    # Usage: flag-auto-resolve <trigger>
    trigger="${1:-build_pass}"
    flags_file="$COLONY_DATA_DIR/flags.json"

    if [[ ! -f "$flags_file" ]]; then json_ok '{"resolved":0}'; exit 0; fi

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
    if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
      json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
    else
      acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
      # Ensure lock is always released on exit (BUG-005/BUG-011 fix)
      trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
    fi

    # Count how many will be resolved
    count=$(jq --arg trigger "$trigger" '
      [.flags[] | select(.auto_resolve_on == $trigger and .resolved_at == null)] | length
    ' "$flags_file") || {
      json_err "$E_JSON_INVALID" "Failed to count flags for auto-resolve"
    }

    # Resolve them
    updated=$(jq --arg trigger "$trigger" --arg ts "$ts" '
      .flags = [.flags[] | if .auto_resolve_on == $trigger and .resolved_at == null then
        .resolved_at = $ts |
        .resolution = "Auto-resolved on " + $trigger
      else . end]
    ' "$flags_file") || {
      json_err "$E_JSON_INVALID" "Failed to auto-resolve flags"
    }

    atomic_write "$flags_file" "$updated"
    trap - EXIT
    release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    json_ok "$(jq -n --argjson count "$count" --arg trigger "$trigger" \
      '{resolved: $count, trigger: $trigger}')"
}
