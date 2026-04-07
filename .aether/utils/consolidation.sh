# =============================================================================
# DEPRECATED — This script has been superseded by the Go binary (aether CLI).
# All functionality is now available via: aether <subcommand>
# Do NOT modify this file — it is retained for reference only.
# See: cmd/ (Go source) | Run: aether --help
# =============================================================================
#
#!/bin/bash
# Consolidation utility — lightweight end-of-phase consolidation
# Provides: _consolidation_phase_end
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write,
# COLONY_DATA_DIR, SCRIPT_DIR, error constants) is available.
#
# Subcommand: consolidation-phase-end [--dry-run]
# Runs nurse + herald + janitor curation ants in sequence,
# then publishes a consolidation.phase_end event to the event bus.
# Each step is non-blocking: failures are logged and execution continues.

# ============================================================================
# _consolidation_phase_end
# Run lightweight end-of-phase consolidation.
#
# Usage: consolidation-phase-end [--dry-run]
#
# Output: json_ok with:
#   {type, steps:[{name, status, summary}], event_published, dry_run}
# ============================================================================
_consolidation_phase_end() {
    local dry_run="false"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run) dry_run="true"; shift ;;
            *) shift ;;
        esac
    done

    local dry_run_flag=""
    [[ "$dry_run" == "true" ]] && dry_run_flag="--dry-run"

    local steps_json="[]"

    # ── Step 1: curation-nurse (trust recalculation) ──────────────────────────
    local nurse_status="ok"
    local nurse_summary=""
    local nurse_result
    # shellcheck disable=SC2086
    nurse_result=$(bash "$0" curation-nurse $dry_run_flag 2>/dev/null) || nurse_status="failed"

    if [[ "$nurse_status" == "ok" ]]; then
        nurse_summary=$(echo "$nurse_result" | jq -r \
            '"observations_updated=\(.result.observations_updated // 0) instincts_updated=\(.result.instincts_updated // 0)"' \
            2>/dev/null || echo "completed")
    else
        nurse_summary="nurse step failed; continuing"
    fi

    steps_json=$(echo "$steps_json" | jq -c \
        --arg name "nurse" \
        --arg status "$nurse_status" \
        --arg summary "$nurse_summary" \
        '. + [{name:$name,status:$status,summary:$summary}]')

    # ── Step 2: curation-herald (promote high-trust to QUEEN.md) ─────────────
    local herald_status="ok"
    local herald_summary=""
    local herald_result
    # shellcheck disable=SC2086
    herald_result=$(bash "$0" curation-herald $dry_run_flag 2>/dev/null) || herald_status="failed"

    if [[ "$herald_status" == "ok" ]]; then
        herald_summary=$(echo "$herald_result" | jq -r \
            '"eligible=\(.result.eligible // 0) promoted=\(.result.promoted // 0)"' \
            2>/dev/null || echo "completed")
    else
        herald_summary="herald step failed; continuing"
    fi

    steps_json=$(echo "$steps_json" | jq -c \
        --arg name "herald" \
        --arg status "$herald_status" \
        --arg summary "$herald_summary" \
        '. + [{name:$name,status:$status,summary:$summary}]')

    # ── Step 3: curation-janitor (clean expired) ─────────────────────────────
    local janitor_status="ok"
    local janitor_summary=""
    local janitor_result
    # shellcheck disable=SC2086
    janitor_result=$(bash "$0" curation-janitor $dry_run_flag 2>/dev/null) || janitor_status="failed"

    if [[ "$janitor_status" == "ok" ]]; then
        janitor_summary=$(echo "$janitor_result" | jq -r \
            '"events_removed=\(.result.events_removed // 0) instincts_pruned=\(.result.instincts_pruned // 0)"' \
            2>/dev/null || echo "completed")
    else
        janitor_summary="janitor step failed; continuing"
    fi

    steps_json=$(echo "$steps_json" | jq -c \
        --arg name "janitor" \
        --arg status "$janitor_status" \
        --arg summary "$janitor_summary" \
        '. + [{name:$name,status:$status,summary:$summary}]')

    # ── Step 4: publish event to event bus ───────────────────────────────────
    local event_published="false"

    local current_phase
    current_phase=$(jq -r '.current_phase // 0' "$COLONY_DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "0")

    local payload
    payload=$(jq -nc \
        --argjson phase "$current_phase" \
        --argjson dry_run "$dry_run" \
        --argjson steps "$steps_json" \
        '{phase:$phase,dry_run:$dry_run,steps:$steps}')

    if bash "$0" event-publish \
            --topic "consolidation.phase_end" \
            --payload "$payload" \
            --source "consolidation" \
            > /dev/null 2>&1; then
        event_published="true"
    fi

    json_ok "$(jq -nc \
        --argjson steps "$steps_json" \
        --argjson event_published "$event_published" \
        --argjson dry_run "$dry_run" \
        '{type:"phase_end",steps:$steps,event_published:$event_published,dry_run:$dry_run}')"
}
