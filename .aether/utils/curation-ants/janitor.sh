#!/bin/bash
# Curation Janitor — Expired Event/Archive Pruning
# Cleans up expired events and old archived instincts.
#
# Functions:
#   _curation_janitor
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write, COLONY_DATA_DIR,
# DATA_DIR, error constants) is available when sourced.

# ============================================================================
# _curation_janitor
# Remove expired events and prune stale data.
# Usage: curation-janitor [--dry-run] [--max-age-days <N>]
#
# Default max-age-days: 90
# Output: json_ok with {events_removed:N, instincts_pruned:N,
#                       observations_pruned:N, dry_run:bool}
# ============================================================================
_curation_janitor() {
    local cj_dry_run="false"
    local cj_max_age_days=90

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                cj_dry_run="true"
                shift
                ;;
            --max-age-days)
                cj_max_age_days="${2:-90}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    local cj_data_dir="${COLONY_DATA_DIR:-${DATA_DIR:-}}"
    if [[ -z "$cj_data_dir" ]]; then
        json_err "$E_VALIDATION_FAILED" "curation-janitor: COLONY_DATA_DIR is not set"
    fi

    # Step 1: Clean expired events via event-cleanup subcommand
    local cj_events_removed=0
    local cj_cleanup_dry_run_flag=""
    [[ "$cj_dry_run" == "true" ]] && cj_cleanup_dry_run_flag="--dry-run"

    local cj_cleanup_result
    if [[ -n "$cj_cleanup_dry_run_flag" ]]; then
        cj_cleanup_result=$(COLONY_DATA_DIR="$cj_data_dir" DATA_DIR="$cj_data_dir" \
            bash "$0" event-cleanup "$cj_cleanup_dry_run_flag" 2>/dev/null) || true
    else
        cj_cleanup_result=$(COLONY_DATA_DIR="$cj_data_dir" DATA_DIR="$cj_data_dir" \
            bash "$0" event-cleanup 2>/dev/null) || true
    fi
    if echo "$cj_cleanup_result" | jq -e '.ok == true' >/dev/null 2>&1; then
        cj_events_removed=$(echo "$cj_cleanup_result" | jq '.result.removed // 0' 2>/dev/null || echo 0)
    fi

    # Step 2: Prune archived instincts older than max-age-days
    local cj_instincts_pruned=0
    local cj_instincts_file="$cj_data_dir/instincts.json"

    if [[ -f "$cj_instincts_file" ]] && jq empty "$cj_instincts_file" 2>/dev/null; then
        local cj_cutoff
        cj_cutoff=$(date -u -v"-${cj_max_age_days}d" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null \
            || date -u -d "-${cj_max_age_days} days" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null \
            || echo "1970-01-01T00:00:00Z")

        cj_instincts_pruned=$(jq --arg cutoff "$cj_cutoff" \
            '[.instincts[] | select(.archived == true and (.updated_at // .created_at) < $cutoff)] | length' \
            "$cj_instincts_file" 2>/dev/null || echo 0)

        if [[ "$cj_dry_run" == "false" && "$cj_instincts_pruned" -gt 0 ]]; then
            local cj_updated_instincts
            cj_updated_instincts=$(jq --arg cutoff "$cj_cutoff" \
                '.instincts |= [.[] | select(not (.archived == true and (.updated_at // .created_at) < $cutoff))]' \
                "$cj_instincts_file" 2>/dev/null)
            if [[ -n "$cj_updated_instincts" ]]; then
                atomic_write "$cj_instincts_file" "$cj_updated_instincts" 2>/dev/null || true
            fi
        fi
    fi

    # Step 3: Prune stale learning-observations (observation_count=1, older than 90 days)
    local cj_observations_pruned=0
    local cj_obs_file="$cj_data_dir/learning-observations.json"
    local cj_obs_cutoff_days=90

    if [[ -f "$cj_obs_file" ]] && jq empty "$cj_obs_file" 2>/dev/null; then
        local cj_obs_cutoff
        cj_obs_cutoff=$(date -u -v"-${cj_obs_cutoff_days}d" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null \
            || date -u -d "-${cj_obs_cutoff_days} days" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null \
            || echo "1970-01-01T00:00:00Z")

        cj_observations_pruned=$(jq --arg cutoff "$cj_obs_cutoff" \
            '[.observations[] | select((.observation_count // 0) == 1 and (.last_observed // .observed_at // "") < $cutoff)] | length' \
            "$cj_obs_file" 2>/dev/null || echo 0)

        if [[ "$cj_dry_run" == "false" && "$cj_observations_pruned" -gt 0 ]]; then
            local cj_updated_obs
            cj_updated_obs=$(jq --arg cutoff "$cj_obs_cutoff" \
                '.observations |= [.[] | select(not ((.observation_count // 0) == 1 and (.last_observed // .observed_at // "") < $cutoff))]' \
                "$cj_obs_file" 2>/dev/null)
            if [[ -n "$cj_updated_obs" ]]; then
                atomic_write "$cj_obs_file" "$cj_updated_obs" 2>/dev/null || true
            fi
        fi
    fi

    json_ok "$(jq -nc \
        --argjson events_removed "$cj_events_removed" \
        --argjson instincts_pruned "$cj_instincts_pruned" \
        --argjson observations_pruned "$cj_observations_pruned" \
        --argjson dry_run "$cj_dry_run" \
        '{events_removed:$events_removed, instincts_pruned:$instincts_pruned,
          observations_pruned:$observations_pruned, dry_run:$dry_run}')"
}
