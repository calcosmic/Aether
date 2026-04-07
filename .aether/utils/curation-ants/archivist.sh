#!/bin/bash
# Curation Archivist — Low-Trust Instinct Archival
# Archives instincts that have decayed below a trust threshold.
#
# Functions:
#   _curation_archivist
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write, COLONY_DATA_DIR,
# DATA_DIR, error constants) is available when sourced.

# ============================================================================
# _curation_archivist
# Archive active instincts with trust_score below threshold.
# Usage: curation-archivist [--threshold <float>] [--dry-run]
#
# Default threshold: 0.25
# Output: json_ok with {archived:N, below_threshold:N, dry_run:bool}
# ============================================================================
_curation_archivist() {
    local ca_threshold="0.25"
    local ca_dry_run="false"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --threshold)
                ca_threshold="${2:-0.25}"
                shift 2
                ;;
            --dry-run)
                ca_dry_run="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done

    local ca_data_dir="${COLONY_DATA_DIR:-${DATA_DIR:-}}"
    if [[ -z "$ca_data_dir" ]]; then
        json_err "$E_VALIDATION_FAILED" "curation-archivist: COLONY_DATA_DIR is not set"
    fi

    local ca_instincts_file="$ca_data_dir/instincts.json"

    # No instincts file — nothing to archive
    if [[ ! -f "$ca_instincts_file" ]]; then
        json_ok "$(jq -nc \
            --argjson archived 0 \
            --argjson below_threshold 0 \
            --argjson dry_run "$ca_dry_run" \
            '{archived:$archived, below_threshold:$below_threshold, dry_run:$dry_run}')"
        return 0
    fi

    if ! jq empty "$ca_instincts_file" 2>/dev/null; then
        json_err "$E_JSON_INVALID" "curation-archivist: instincts.json is not valid JSON"
    fi

    # Count active instincts below threshold
    local ca_below_threshold
    ca_below_threshold=$(jq --argjson thresh "$ca_threshold" \
        '[.instincts[] | select(.archived != true and (.trust_score // .confidence // 0) < $thresh)] | length' \
        "$ca_instincts_file" 2>/dev/null || echo 0)

    local ca_archived=0

    if [[ "$ca_dry_run" == "false" && "$ca_below_threshold" -gt 0 ]]; then
        local ca_ts
        ca_ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

        local ca_updated
        ca_updated=$(jq \
            --argjson thresh "$ca_threshold" \
            --arg ts "$ca_ts" \
            '.instincts |= [.[] | if (.archived != true and (.trust_score // .confidence // 0) < $thresh)
                then . + {archived: true, updated_at: $ts}
                else .
                end]' \
            "$ca_instincts_file" 2>/dev/null)

        if [[ -n "$ca_updated" ]]; then
            atomic_write "$ca_instincts_file" "$ca_updated" 2>/dev/null \
                || json_err "$E_UNKNOWN" "curation-archivist: failed to write instincts.json"
            ca_archived="$ca_below_threshold"
        fi
    elif [[ "$ca_dry_run" == "true" ]]; then
        ca_archived="$ca_below_threshold"
    fi

    json_ok "$(jq -nc \
        --argjson archived "$ca_archived" \
        --argjson below_threshold "$ca_below_threshold" \
        --argjson dry_run "$ca_dry_run" \
        '{archived:$archived, below_threshold:$below_threshold, dry_run:$dry_run}')"
}
