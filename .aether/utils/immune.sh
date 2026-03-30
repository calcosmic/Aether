#!/bin/bash
# Immune response system — trophallaxis repair and scarification
# Provides: _trophallaxis_diagnose, _trophallaxis_retry, _scar_add, _scar_list,
#           _scar_check, _immune_auto_scar
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write, acquire_lock,
# release_lock, LOCK_DIR, COLONY_DATA_DIR, error constants) is available.

# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------

_immune_data_dir() {
    echo "$COLONY_DATA_DIR/immune"
}

_immune_scars_file() {
    echo "$COLONY_DATA_DIR/immune/scars.json"
}

_immune_retry_log_file() {
    echo "$COLONY_DATA_DIR/immune/retry-log.json"
}

_immune_ensure_dir() {
    mkdir -p "$(_immune_data_dir)"
}

_immune_ensure_scars_file() {
    local sf
    sf="$(_immune_scars_file)"
    _immune_ensure_dir
    if [[ ! -f "$sf" ]]; then
        printf '%s\n' '{"version":"1.0","scars":[]}' > "$sf"
    fi
}

_immune_ensure_retry_log() {
    local rf
    rf="$(_immune_retry_log_file)"
    _immune_ensure_dir
    if [[ ! -f "$rf" ]]; then
        printf '%s\n' '{"version":"1.0","tasks":{}}' > "$rf"
    fi
}

# ---------------------------------------------------------------------------
# _trophallaxis_diagnose
# Usage: trophallaxis-diagnose --task-id <id> --failure <desc> [--phase N]
# ---------------------------------------------------------------------------
_trophallaxis_diagnose() {
    local td_task_id=""
    local td_failure=""
    local td_phase=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --task-id) td_task_id="${2:-}"; shift 2 ;;
            --failure)  td_failure="${2:-}";  shift 2 ;;
            --phase)    td_phase="${2:-}";    shift 2 ;;
            *) shift ;;
        esac
    done

    if [[ -z "$td_task_id" ]]; then
        json_err "$E_VALIDATION_FAILED" "trophallaxis-diagnose requires --task-id"
        return
    fi

    if [[ -z "$td_failure" ]]; then
        json_err "$E_VALIDATION_FAILED" "trophallaxis-diagnose requires --failure"
        return
    fi

    local td_midden_file="$COLONY_DATA_DIR/midden/midden.json"
    local td_related=0
    local td_related_entries="[]"
    local td_approach=""

    # Search midden for related entries using keywords from failure description
    if [[ -f "$td_midden_file" ]]; then
        # Extract first keyword (longest word >= 4 chars) for search
        local td_keyword
        td_keyword=$(echo "$td_failure" | tr '[:upper:]' '[:lower:]' | \
            grep -oE '[a-z]{4,}' | sort -rn -k1,1 | head -1 || echo "")

        if [[ -n "$td_keyword" ]]; then
            td_related=$(jq \
                --arg q "$td_keyword" \
                '[.entries // [] | .[] |
                  select(.acknowledged != true) |
                  select(.message | ascii_downcase | contains($q))
                ] | length' "$td_midden_file" 2>/dev/null || echo "0")

            td_related_entries=$(jq \
                --arg q "$td_keyword" \
                '[.entries // [] | .[] |
                  select(.acknowledged != true) |
                  select(.message | ascii_downcase | contains($q)) |
                  {id, timestamp, category, source, message}
                ] | .[:5]' "$td_midden_file" 2>/dev/null || echo "[]")
        else
            # No usable keyword — scan all recent entries
            td_related=$(jq '[.entries // [] | .[] | select(.acknowledged != true)] | length' \
                "$td_midden_file" 2>/dev/null || echo "0")
        fi
    fi

    # Build diagnosis text
    local td_diagnosis
    if [[ "$td_related" -gt 0 ]]; then
        td_diagnosis="Found $td_related related failure(s) in midden matching: $td_failure"
        td_approach="Review related midden entries for patterns. Address root cause before retrying."
    else
        td_diagnosis="No related failures found in midden for: $td_failure"
        td_approach="Investigate failure from first principles. Check logs and dependencies."
    fi

    # Confidence: higher when related failures exist
    local td_confidence
    if [[ "$td_related" -ge 3 ]]; then
        td_confidence="0.9"
    elif [[ "$td_related" -ge 1 ]]; then
        td_confidence="0.7"
    else
        td_confidence="0.4"
    fi

    json_ok "$(jq -n \
        --arg task_id "$td_task_id" \
        --arg failure "$td_failure" \
        --arg diagnosis "$td_diagnosis" \
        --argjson related_failures "$td_related" \
        --arg suggested_approach "$td_approach" \
        --argjson confidence "$td_confidence" \
        --argjson related_entries "$td_related_entries" \
        '{
            task_id: $task_id,
            failure: $failure,
            diagnosis: $diagnosis,
            related_failures: $related_failures,
            suggested_approach: $suggested_approach,
            confidence: $confidence,
            related_entries: $related_entries
        }')"
}

# ---------------------------------------------------------------------------
# _trophallaxis_retry
# Usage: trophallaxis-retry --task-id <id> --diagnosis <json>
# ---------------------------------------------------------------------------
_trophallaxis_retry() {
    local tr_task_id=""
    local tr_diagnosis=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --task-id)  tr_task_id="${2:-}";   shift 2 ;;
            --diagnosis) tr_diagnosis="${2:-}"; shift 2 ;;
            *) shift ;;
        esac
    done

    if [[ -z "$tr_task_id" ]]; then
        json_err "$E_VALIDATION_FAILED" "trophallaxis-retry requires --task-id"
        return
    fi

    _immune_ensure_retry_log

    local tr_log_file
    tr_log_file="$(_immune_retry_log_file)"
    local tr_timestamp
    tr_timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Read current retry count for this task
    local tr_current_count
    tr_current_count=$(jq -r --arg tid "$tr_task_id" \
        '.tasks[$tid].retry_count // 0' "$tr_log_file" 2>/dev/null || echo "0")
    local tr_new_count=$(( tr_current_count + 1 ))

    # Validate diagnosis JSON (graceful: use empty object if invalid)
    local tr_diag_json
    if echo "$tr_diagnosis" | jq empty 2>/dev/null; then
        tr_diag_json="$tr_diagnosis"
    else
        tr_diag_json="{}"
    fi

    # Build the retry entry
    local tr_entry
    tr_entry=$(jq -n \
        --arg ts "$tr_timestamp" \
        --argjson retry_count "$tr_new_count" \
        --argjson diagnosis "$tr_diag_json" \
        '{timestamp: $ts, retry_count: $retry_count, diagnosis: $diagnosis}')

    # Update log with locking
    local tr_updated
    if acquire_lock "$tr_log_file" 2>/dev/null; then
        tr_updated=$(jq \
            --arg tid "$tr_task_id" \
            --argjson entry "$tr_entry" \
            --argjson new_count "$tr_new_count" \
            '.tasks[$tid] = {
                retry_count: $new_count,
                last_attempt: $entry.timestamp,
                last_diagnosis: $entry.diagnosis,
                history: ((.tasks[$tid].history // []) + [$entry])
            }' "$tr_log_file" 2>/dev/null)

        if [[ -n "$tr_updated" ]]; then
            atomic_write "$tr_log_file" "$tr_updated"
        fi
        release_lock 2>/dev/null || true
    else
        # Lockless fallback
        tr_updated=$(jq \
            --arg tid "$tr_task_id" \
            --argjson entry "$tr_entry" \
            --argjson new_count "$tr_new_count" \
            '.tasks[$tid] = {
                retry_count: $new_count,
                last_attempt: $entry.timestamp,
                last_diagnosis: $entry.diagnosis,
                history: ((.tasks[$tid].history // []) + [$entry])
            }' "$tr_log_file" 2>/dev/null)

        if [[ -n "$tr_updated" ]]; then
            atomic_write "$tr_log_file" "$tr_updated"
        fi
    fi

    json_ok "$(jq -n \
        --arg task_id "$tr_task_id" \
        --argjson retry_count "$tr_new_count" \
        '{task_id: $task_id, retry_count: $retry_count, diagnosis_injected: true}')"
}

# ---------------------------------------------------------------------------
# _scar_add
# Usage: scar-add --pattern <desc> --severity <low|medium|high> [--phase N] [--source <src>]
# ---------------------------------------------------------------------------
_scar_add() {
    local sa_pattern=""
    local sa_severity=""
    local sa_phase=""
    local sa_source="unknown"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --pattern)  sa_pattern="${2:-}";  shift 2 ;;
            --severity) sa_severity="${2:-}"; shift 2 ;;
            --phase)    sa_phase="${2:-}";    shift 2 ;;
            --source)   sa_source="${2:-}";   shift 2 ;;
            *) shift ;;
        esac
    done

    if [[ -z "$sa_pattern" ]]; then
        json_err "$E_VALIDATION_FAILED" "scar-add requires --pattern"
        return
    fi

    # Default severity if not provided
    if [[ -z "$sa_severity" ]]; then
        sa_severity="medium"
    fi

    # Validate severity
    case "$sa_severity" in
        low|medium|high) ;;
        *)
            json_err "$E_VALIDATION_FAILED" "scar-add --severity must be low, medium, or high"
            return
            ;;
    esac

    _immune_ensure_scars_file

    local sa_scars_file
    sa_scars_file="$(_immune_scars_file)"
    local sa_timestamp
    sa_timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local sa_id
    sa_id="scar_$(date +%s)_$$"

    # Build phase value for JSON
    local sa_phase_val
    if [[ -n "$sa_phase" ]]; then
        sa_phase_val="$sa_phase"
    else
        sa_phase_val="null"
    fi

    local sa_new_scar
    sa_new_scar=$(jq -n \
        --arg id "$sa_id" \
        --arg pattern "$sa_pattern" \
        --arg severity "$sa_severity" \
        --arg phase "$sa_phase_val" \
        --arg source "$sa_source" \
        --arg created_at "$sa_timestamp" \
        '{
            id: $id,
            pattern: $pattern,
            severity: $severity,
            phase: (if $phase == "null" then null else ($phase | tonumber? // $phase) end),
            source: $source,
            created_at: $created_at,
            retry_count: 0,
            active: true
        }')

    local sa_updated
    if acquire_lock "$sa_scars_file" 2>/dev/null; then
        sa_updated=$(jq \
            --argjson scar "$sa_new_scar" \
            '.scars += [$scar]' "$sa_scars_file" 2>/dev/null)
        if [[ -n "$sa_updated" ]]; then
            atomic_write "$sa_scars_file" "$sa_updated"
        fi
        release_lock 2>/dev/null || true
    else
        sa_updated=$(jq \
            --argjson scar "$sa_new_scar" \
            '.scars += [$scar]' "$sa_scars_file" 2>/dev/null)
        if [[ -n "$sa_updated" ]]; then
            atomic_write "$sa_scars_file" "$sa_updated"
        fi
    fi

    local sa_count
    sa_count=$(jq '[.scars[]] | length' "$sa_scars_file" 2>/dev/null || echo "1")

    json_ok "$(jq -n \
        --arg id "$sa_id" \
        --argjson scar_count "$sa_count" \
        '{id: $id, scar_count: $scar_count}')"
}

# ---------------------------------------------------------------------------
# _scar_list
# Usage: scar-list [--active] [--severity <level>]
# ---------------------------------------------------------------------------
_scar_list() {
    local sl_active_only=false
    local sl_severity=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --active)   sl_active_only=true; shift ;;
            --severity) sl_severity="${2:-}"; shift 2 ;;
            *) shift ;;
        esac
    done

    local sl_scars_file
    sl_scars_file="$(_immune_scars_file)"

    if [[ ! -f "$sl_scars_file" ]]; then
        json_ok '{"total":0,"active":0,"scars":[]}'
        return 0
    fi

    local sl_result
    sl_result=$(jq \
        --argjson active_only "$sl_active_only" \
        --arg severity "$sl_severity" \
        '
        [.scars // [] | .[] |
            if $active_only then select(.active == true) else . end |
            if ($severity | length) > 0 then select(.severity == $severity) else . end
        ] |
        . as $filtered |
        {
            total: ($filtered | length),
            active: ($filtered | map(select(.active == true)) | length),
            scars: $filtered
        }
        ' "$sl_scars_file" 2>/dev/null)

    if [[ -z "$sl_result" ]]; then
        json_ok '{"total":0,"active":0,"scars":[]}'
        return 0
    fi

    json_ok "$sl_result"
}

# ---------------------------------------------------------------------------
# _scar_check
# Usage: scar-check --task <desc>
# ---------------------------------------------------------------------------
_scar_check() {
    local sc_task=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --task) sc_task="${2:-}"; shift 2 ;;
            *) shift ;;
        esac
    done

    if [[ -z "$sc_task" ]]; then
        json_err "$E_VALIDATION_FAILED" "scar-check requires --task"
        return
    fi

    local sc_scars_file
    sc_scars_file="$(_immune_scars_file)"

    if [[ ! -f "$sc_scars_file" ]]; then
        json_ok '{"matches":0,"scars":[]}'
        return 0
    fi

    local sc_result
    sc_result=$(jq \
        --arg task "$sc_task" \
        '
        [.scars // [] | .[] |
            select(.active == true) |
            # Split pattern into words, check if any word from pattern appears in task
            select(
                (.pattern | ascii_downcase) as $pat |
                ($pat | split(" ") | .[] | select(length >= 3)) as $word |
                ($task | ascii_downcase) | contains($word)
            )
        ] |
        . as $matches |
        {
            matches: ($matches | length),
            scars: $matches
        }
        ' "$sc_scars_file" 2>/dev/null)

    if [[ -z "$sc_result" ]]; then
        json_ok '{"matches":0,"scars":[]}'
        return 0
    fi

    json_ok "$sc_result"
}

# ---------------------------------------------------------------------------
# _immune_auto_scar
# Usage: immune-auto-scar --task-id <id>
# ---------------------------------------------------------------------------
_immune_auto_scar() {
    local ias_task_id=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --task-id) ias_task_id="${2:-}"; shift 2 ;;
            *) shift ;;
        esac
    done

    if [[ -z "$ias_task_id" ]]; then
        json_err "$E_VALIDATION_FAILED" "immune-auto-scar requires --task-id"
        return
    fi

    local ias_log_file
    ias_log_file="$(_immune_retry_log_file)"

    if [[ ! -f "$ias_log_file" ]]; then
        json_ok '{"auto_scarred":false,"retry_count":0}'
        return 0
    fi

    local ias_retry_count
    ias_retry_count=$(jq -r --arg tid "$ias_task_id" \
        '.tasks[$tid].retry_count // 0' "$ias_log_file" 2>/dev/null || echo "0")

    if [[ "$ias_retry_count" -lt 3 ]]; then
        json_ok "$(jq -n \
            --argjson retry_count "$ias_retry_count" \
            '{auto_scarred: false, retry_count: $retry_count}')"
        return 0
    fi

    # Auto-create a scar for this task
    local ias_last_diagnosis
    ias_last_diagnosis=$(jq -r --arg tid "$ias_task_id" \
        '.tasks[$tid].last_diagnosis.failure // ""' "$ias_log_file" 2>/dev/null || echo "")

    local ias_pattern
    if [[ -n "$ias_last_diagnosis" ]]; then
        ias_pattern="$ias_last_diagnosis"
    else
        ias_pattern="task $ias_task_id failed persistently (auto-scarred after $ias_retry_count retries)"
    fi

    # Determine severity based on retry count
    local ias_severity="medium"
    if [[ "$ias_retry_count" -ge 5 ]]; then
        ias_severity="high"
    fi

    _scar_add --pattern "$ias_pattern" --severity "$ias_severity" --source "immune-auto-scar" >/dev/null

    json_ok "$(jq -n \
        --argjson retry_count "$ias_retry_count" \
        '{auto_scarred: true, retry_count: $retry_count}')"
}
