# =============================================================================
# DEPRECATED — This script has been superseded by the Go binary (aether CLI).
# All functionality is now available via: aether <subcommand>
# Do NOT modify this file — it is retained for reference only.
# See: cmd/ (Go source) | Run: aether --help
# =============================================================================
#
#!/bin/bash
# Council deliberation module — Advocate/Challenger/Sage model with spawn budget guards
# Provides: _council_deliberate, _council_advocate, _council_challenger, _council_sage,
#           _council_history, _council_budget_check
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write, acquire_lock,
# release_lock, LOCK_DIR, COLONY_DATA_DIR, error constants) is available.
# _spawn_can_spawn is available from spawn.sh (sourced before this module).

# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------

_council_data_dir() {
    echo "$COLONY_DATA_DIR/council"
}

_council_deliberations_file() {
    echo "$COLONY_DATA_DIR/council/deliberations.json"
}

_council_ensure_file() {
    local cf
    cf="$(_council_deliberations_file)"
    mkdir -p "$(_council_data_dir)"
    if [[ ! -f "$cf" ]]; then
        printf '%s\n' '{"version":"1.0","deliberations":[]}' > "$cf"
    fi
}

# ---------------------------------------------------------------------------
# _council_deliberate
# Usage: council-deliberate --proposal <text> [--budget N] [--depth light|standard|deep]
# ---------------------------------------------------------------------------
_council_deliberate() {
    local cd_proposal=""
    local cd_budget="3"
    local cd_depth="standard"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --proposal) cd_proposal="${2:-}"; shift 2 ;;
            --budget)   cd_budget="${2:-3}";  shift 2 ;;
            --depth)    cd_depth="${2:-standard}"; shift 2 ;;
            *) shift ;;
        esac
    done

    if [[ -z "$cd_proposal" ]]; then
        json_err "$E_VALIDATION_FAILED" "council-deliberate requires --proposal"
        return
    fi

    if ! [[ "$cd_budget" =~ ^[0-9]+$ ]]; then
        json_err "$E_VALIDATION_FAILED" "council-deliberate --budget must be a positive integer"
        return
    fi

    local cd_ts
    cd_ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local cd_unix
    cd_unix=$(date -u +%s)
    local cd_id="delib_${cd_unix}"

    _council_ensure_file

    local cd_file
    cd_file="$(_council_deliberations_file)"

    acquire_lock "$cd_file" 2>/dev/null || true
    # shellcheck disable=SC2064
    trap "release_lock '$cd_file' 2>/dev/null || true" EXIT

    local cd_updated
    cd_updated=$(jq \
        --arg id "$cd_id" \
        --arg proposal "$cd_proposal" \
        --arg ts "$cd_ts" \
        --argjson budget "$cd_budget" \
        --arg depth "$cd_depth" \
        '.deliberations += [{
            "id": $id,
            "proposal": $proposal,
            "advocate": null,
            "challenger": null,
            "sage": null,
            "budget": $budget,
            "depth": $depth,
            "created_at": $ts,
            "status": "pending"
        }]' "$cd_file") || {
        release_lock "$cd_file" 2>/dev/null || true
        trap - EXIT
        json_err "$E_UNKNOWN" "Failed to write deliberation"
        return
    }

    atomic_write "$cd_file" "$cd_updated" || {
        release_lock "$cd_file" 2>/dev/null || true
        trap - EXIT
        json_err "$E_UNKNOWN" "Failed to persist deliberation"
        return
    }

    release_lock "$cd_file" 2>/dev/null || true
    trap - EXIT

    json_ok "$(jq -n \
        --arg id "$cd_id" \
        --arg proposal "$cd_proposal" \
        --argjson budget "$cd_budget" \
        '{"id":$id,"proposal":$proposal,"status":"pending","budget":$budget}')"
}

# ---------------------------------------------------------------------------
# _council_advocate
# Usage: council-advocate --deliberation-id <id> --argument <text>
# ---------------------------------------------------------------------------
_council_advocate() {
    local ca_id=""
    local ca_argument=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --deliberation-id) ca_id="${2:-}"; shift 2 ;;
            --argument)        ca_argument="${2:-}"; shift 2 ;;
            *) shift ;;
        esac
    done

    if [[ -z "$ca_id" ]]; then
        json_err "$E_VALIDATION_FAILED" "council-advocate requires --deliberation-id"
        return
    fi

    if [[ -z "$ca_argument" ]]; then
        json_err "$E_VALIDATION_FAILED" "council-advocate requires --argument"
        return
    fi

    local ca_file
    ca_file="$(_council_deliberations_file)"

    if [[ ! -f "$ca_file" ]]; then
        json_err "$E_VALIDATION_FAILED" "No deliberations found; run council-deliberate first"
        return
    fi

    local ca_exists
    ca_exists=$(jq -r --arg id "$ca_id" '.deliberations[] | select(.id == $id) | .id' "$ca_file" 2>/dev/null || echo "")
    if [[ -z "$ca_exists" ]]; then
        json_err "$E_VALIDATION_FAILED" "Deliberation not found: $ca_id"
        return
    fi

    acquire_lock "$ca_file" 2>/dev/null || true
    # shellcheck disable=SC2064
    trap "release_lock '$ca_file' 2>/dev/null || true" EXIT

    local ca_updated
    ca_updated=$(jq \
        --arg id "$ca_id" \
        --arg arg "$ca_argument" \
        '(.deliberations[] | select(.id == $id)).advocate = $arg
         | (.deliberations[] | select(.id == $id)).status = "in_progress"' \
        "$ca_file") || {
        release_lock "$ca_file" 2>/dev/null || true
        trap - EXIT
        json_err "$E_UNKNOWN" "Failed to record advocate argument"
        return
    }

    atomic_write "$ca_file" "$ca_updated" || {
        release_lock "$ca_file" 2>/dev/null || true
        trap - EXIT
        json_err "$E_UNKNOWN" "Failed to persist advocate argument"
        return
    }

    release_lock "$ca_file" 2>/dev/null || true
    trap - EXIT

    json_ok '{"role":"advocate","recorded":true}'
}

# ---------------------------------------------------------------------------
# _council_challenger
# Usage: council-challenger --deliberation-id <id> --argument <text>
# ---------------------------------------------------------------------------
_council_challenger() {
    local cc_id=""
    local cc_argument=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --deliberation-id) cc_id="${2:-}"; shift 2 ;;
            --argument)        cc_argument="${2:-}"; shift 2 ;;
            *) shift ;;
        esac
    done

    if [[ -z "$cc_id" ]]; then
        json_err "$E_VALIDATION_FAILED" "council-challenger requires --deliberation-id"
        return
    fi

    if [[ -z "$cc_argument" ]]; then
        json_err "$E_VALIDATION_FAILED" "council-challenger requires --argument"
        return
    fi

    local cc_file
    cc_file="$(_council_deliberations_file)"

    if [[ ! -f "$cc_file" ]]; then
        json_err "$E_VALIDATION_FAILED" "No deliberations found; run council-deliberate first"
        return
    fi

    local cc_exists
    cc_exists=$(jq -r --arg id "$cc_id" '.deliberations[] | select(.id == $id) | .id' "$cc_file" 2>/dev/null || echo "")
    if [[ -z "$cc_exists" ]]; then
        json_err "$E_VALIDATION_FAILED" "Deliberation not found: $cc_id"
        return
    fi

    acquire_lock "$cc_file" 2>/dev/null || true
    # shellcheck disable=SC2064
    trap "release_lock '$cc_file' 2>/dev/null || true" EXIT

    local cc_updated
    cc_updated=$(jq \
        --arg id "$cc_id" \
        --arg arg "$cc_argument" \
        '(.deliberations[] | select(.id == $id)).challenger = $arg
         | (.deliberations[] | select(.id == $id)).status = "in_progress"' \
        "$cc_file") || {
        release_lock "$cc_file" 2>/dev/null || true
        trap - EXIT
        json_err "$E_UNKNOWN" "Failed to record challenger argument"
        return
    }

    atomic_write "$cc_file" "$cc_updated" || {
        release_lock "$cc_file" 2>/dev/null || true
        trap - EXIT
        json_err "$E_UNKNOWN" "Failed to persist challenger argument"
        return
    }

    release_lock "$cc_file" 2>/dev/null || true
    trap - EXIT

    json_ok '{"role":"challenger","recorded":true}'
}

# ---------------------------------------------------------------------------
# _council_sage
# Usage: council-sage --deliberation-id <id> --synthesis <text> --recommendation <text>
# ---------------------------------------------------------------------------
_council_sage() {
    local cs_id=""
    local cs_synthesis=""
    local cs_recommendation=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --deliberation-id) cs_id="${2:-}";             shift 2 ;;
            --synthesis)       cs_synthesis="${2:-}";       shift 2 ;;
            --recommendation)  cs_recommendation="${2:-}";  shift 2 ;;
            *) shift ;;
        esac
    done

    if [[ -z "$cs_id" ]]; then
        json_err "$E_VALIDATION_FAILED" "council-sage requires --deliberation-id"
        return
    fi

    if [[ -z "$cs_synthesis" ]]; then
        json_err "$E_VALIDATION_FAILED" "council-sage requires --synthesis"
        return
    fi

    if [[ -z "$cs_recommendation" ]]; then
        json_err "$E_VALIDATION_FAILED" "council-sage requires --recommendation"
        return
    fi

    local cs_file
    cs_file="$(_council_deliberations_file)"

    if [[ ! -f "$cs_file" ]]; then
        json_err "$E_VALIDATION_FAILED" "No deliberations found; run council-deliberate first"
        return
    fi

    local cs_exists
    cs_exists=$(jq -r --arg id "$cs_id" '.deliberations[] | select(.id == $id) | .id' "$cs_file" 2>/dev/null || echo "")
    if [[ -z "$cs_exists" ]]; then
        json_err "$E_VALIDATION_FAILED" "Deliberation not found: $cs_id"
        return
    fi

    acquire_lock "$cs_file" 2>/dev/null || true
    # shellcheck disable=SC2064
    trap "release_lock '$cs_file' 2>/dev/null || true" EXIT

    local cs_updated
    cs_updated=$(jq \
        --arg id "$cs_id" \
        --arg synthesis "$cs_synthesis" \
        --arg rec "$cs_recommendation" \
        '(.deliberations[] | select(.id == $id)).sage = {"synthesis": $synthesis, "recommendation": $rec}
         | (.deliberations[] | select(.id == $id)).status = "complete"' \
        "$cs_file") || {
        release_lock "$cs_file" 2>/dev/null || true
        trap - EXIT
        json_err "$E_UNKNOWN" "Failed to record sage synthesis"
        return
    }

    atomic_write "$cs_file" "$cs_updated" || {
        release_lock "$cs_file" 2>/dev/null || true
        trap - EXIT
        json_err "$E_UNKNOWN" "Failed to persist sage synthesis"
        return
    }

    release_lock "$cs_file" 2>/dev/null || true
    trap - EXIT

    json_ok "$(jq -n \
        --arg rec "$cs_recommendation" \
        '{"role":"sage","recommendation":$rec,"deliberation_complete":true}')"
}

# ---------------------------------------------------------------------------
# _council_history
# Usage: council-history [--limit N]
# ---------------------------------------------------------------------------
_council_history() {
    local ch_limit=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --limit) ch_limit="${2:-}"; shift 2 ;;
            *) shift ;;
        esac
    done

    local ch_file
    ch_file="$(_council_deliberations_file)"

    if [[ ! -f "$ch_file" ]]; then
        json_ok '{"total":0,"deliberations":[]}'
        return
    fi

    local ch_total
    ch_total=$(jq '.deliberations | length' "$ch_file" 2>/dev/null || echo 0)

    if [[ -n "$ch_limit" ]] && [[ "$ch_limit" =~ ^[0-9]+$ ]]; then
        local ch_result
        ch_result=$(jq \
            --argjson limit "$ch_limit" \
            --argjson total "$ch_total" \
            '{"total":$total,"deliberations":(.deliberations | .[-($limit):])}' \
            "$ch_file" 2>/dev/null) || ch_result='{"total":0,"deliberations":[]}'
        json_ok "$ch_result"
    else
        local ch_result
        ch_result=$(jq \
            --argjson total "$ch_total" \
            '{"total":$total,"deliberations":.deliberations}' \
            "$ch_file" 2>/dev/null) || ch_result='{"total":0,"deliberations":[]}'
        json_ok "$ch_result"
    fi
}

# ---------------------------------------------------------------------------
# _council_budget_check
# Usage: council-budget-check [--budget N]
# ---------------------------------------------------------------------------
_council_budget_check() {
    local cb_budget="3"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --budget) cb_budget="${2:-3}"; shift 2 ;;
            *) shift ;;
        esac
    done

    if ! [[ "$cb_budget" =~ ^[0-9]+$ ]]; then
        json_err "$E_VALIDATION_FAILED" "council-budget-check --budget must be a positive integer"
        return
    fi

    # Delegate to spawn-can-spawn at depth 1
    local cb_spawn_result
    cb_spawn_result=$(_spawn_can_spawn 1 2>/dev/null || echo '{"can_spawn":false,"current_total":0,"global_cap":10}')

    local cb_can
    cb_can=$(echo "$cb_spawn_result" | jq -r '.result.can_spawn // .can_spawn // false' 2>/dev/null || echo "false")
    local cb_current
    cb_current=$(echo "$cb_spawn_result" | jq -r '.result.current_total // .current_total // 0' 2>/dev/null || echo 0)
    local cb_cap
    cb_cap=$(echo "$cb_spawn_result" | jq -r '.result.global_cap // .global_cap // 10' 2>/dev/null || echo 10)
    local cb_remaining=$(( cb_cap - cb_current ))
    [[ $cb_remaining -lt 0 ]] && cb_remaining=0

    # allowed is true only if spawn is allowed AND remaining >= requested budget
    local cb_allowed="false"
    if [[ "$cb_can" == "true" ]] && [[ $cb_remaining -ge $cb_budget ]]; then
        cb_allowed="true"
    fi

    json_ok "$(jq -n \
        --argjson allowed "$cb_allowed" \
        --argjson remaining "$cb_remaining" \
        --argjson budget "$cb_budget" \
        '{"allowed":$allowed,"remaining":$remaining,"budget":$budget}')"
}
