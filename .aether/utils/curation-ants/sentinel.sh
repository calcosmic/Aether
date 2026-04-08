#!/bin/bash
# Curation Sentinel — Memory Health Monitoring
# Checks health of all memory stores and reports issues.
#
# Functions:
#   _curation_sentinel
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, COLONY_DATA_DIR, DATA_DIR,
# error constants) is available when sourced.

# ============================================================================
# _curation_sentinel
# Check health of all memory stores.
# Usage: curation-sentinel
#
# Output: json_ok with {checks:[{store,status,details}], healthy:N, issues:N}
# ============================================================================
_curation_sentinel() {
    local cs_data_dir="${COLONY_DATA_DIR:-${DATA_DIR:-}}"
    if [[ -z "$cs_data_dir" ]]; then
        json_err "$E_VALIDATION_FAILED" "curation-sentinel: COLONY_DATA_DIR is not set"
    fi

    local cs_checks_json="[]"
    local cs_healthy=0
    local cs_issues=0

    # Helper: append a check entry
    # status "healthy" increments healthy; "optional_missing" is neutral;
    # all other statuses (missing, corrupt, empty) increment issues.
    _cs_add_check() {
        local store="$1"
        local status="$2"
        local details="$3"

        cs_checks_json=$(echo "$cs_checks_json" | jq \
            --arg store "$store" \
            --arg status "$status" \
            --arg details "$details" \
            '. += [{store:$store, status:$status, details:$details}]')

        if [[ "$status" == "healthy" ]]; then
            cs_healthy=$(( cs_healthy + 1 ))
        elif [[ "$status" != "optional_missing" ]]; then
            cs_issues=$(( cs_issues + 1 ))
        fi
    }

    # Helper: check a JSON file
    _cs_check_json_file() {
        local store="$1"
        local filepath="$2"
        local required="${3:-false}"

        if [[ ! -f "$filepath" ]]; then
            if [[ "$required" == "true" ]]; then
                _cs_add_check "$store" "missing" "Required file not found: $filepath"
            else
                _cs_add_check "$store" "optional_missing" "Optional file not found: $filepath"
            fi
            return
        fi

        if [[ ! -s "$filepath" ]]; then
            _cs_add_check "$store" "empty" "File exists but is empty: $filepath"
            return
        fi

        if ! jq empty "$filepath" 2>/dev/null; then
            _cs_add_check "$store" "corrupt" "File contains invalid JSON: $filepath"
            return
        fi

        _cs_add_check "$store" "healthy" "OK"
    }

    # 1. learning-observations.json
    _cs_check_json_file "learning-observations" \
        "$cs_data_dir/learning-observations.json" "false"

    # 2. instincts.json (optional)
    _cs_check_json_file "instincts" \
        "$cs_data_dir/instincts.json" "false"

    # 3. instinct-graph.json (optional)
    _cs_check_json_file "instinct-graph" \
        "$cs_data_dir/instinct-graph.json" "false"

    # 4. event-bus.jsonl (optional — check last line if exists)
    local eb_file="$cs_data_dir/event-bus.jsonl"
    if [[ ! -f "$eb_file" ]]; then
        _cs_add_check "event-bus" "optional_missing" "Optional file not found: $eb_file"
    elif [[ ! -s "$eb_file" ]]; then
        _cs_add_check "event-bus" "healthy" "File is empty (no events)"
    else
        local last_line
        last_line=$(tail -1 "$eb_file")
        if echo "$last_line" | jq empty 2>/dev/null; then
            _cs_add_check "event-bus" "healthy" "OK"
        else
            _cs_add_check "event-bus" "corrupt" "Last line is not valid JSON"
        fi
    fi

    # 5. pheromones.json (required)
    _cs_check_json_file "pheromones" \
        "$cs_data_dir/pheromones.json" "true"

    # 6. COLONY_STATE.json (required)
    _cs_check_json_file "COLONY_STATE" \
        "$cs_data_dir/COLONY_STATE.json" "true"

    json_ok "$(jq -nc \
        --argjson checks "$cs_checks_json" \
        --argjson healthy "$cs_healthy" \
        --argjson issues "$cs_issues" \
        '{checks:$checks, healthy:$healthy, issues:$issues}')"
}
