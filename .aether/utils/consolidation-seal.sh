#!/bin/bash
# Consolidation Seal — Full seal-ceremony consolidation
# Provides: _consolidation_seal
#
# Runs once during /ant:seal. Orchestrates curation-run, instinct-decay-all,
# curation-archivist, event publish, and curation-scribe in sequence.
# Each step is non-blocking: failures are logged but do not stop the seal.
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, COLONY_DATA_DIR, DATA_DIR,
# error constants) is available when sourced.

# ============================================================================
# _consolidation_seal
# Full consolidation for the seal ceremony.
#
# Usage: consolidation-seal [--dry-run]
#
# Steps:
#   1. curation-run          — full 8-ant orchestration
#   2. instinct-decay-all    — final trust decay pass
#   3. curation-archivist    — archive borderline instincts (threshold 0.3)
#   4. event-publish         — publish consolidation.seal event
#   5. curation-scribe       — generate final report
#
# Output: json_ok with {type, steps, event_published, dry_run}
# ============================================================================
_consolidation_seal() {
    local dry_run="false"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run) dry_run="true"; shift ;;
            *)         shift ;;
        esac
    done

    local steps_json="[]"
    local event_published="false"

    # _cs_step <name> <subcommand> [extra_args...]
    # Runs the subcommand non-blocking (failures logged, not fatal).
    # Appends a step entry to steps_json.
    _cs_step() {
        local step_name="$1"
        shift
        local subcmd="$1"
        shift
        # Remaining positional args are passed to the subcommand

        local step_result step_status step_summary step_report_path="null"

        # Build argument array respecting dry_run
        local -a args=("$subcmd")
        if [[ "$dry_run" == "true" ]]; then
            args+=("--dry-run")
        fi
        # Append any extra args
        while [[ $# -gt 0 ]]; do
            args+=("$1")
            shift
        done

        step_result=$(bash "$0" "${args[@]}" 2>/dev/null) || true

        if echo "$step_result" | jq -e '.ok == true' >/dev/null 2>&1; then
            step_status="ok"
            step_summary=$(echo "$step_result" | jq -r '
                .result |
                if type == "object" then
                    [ to_entries[] | "\(.key) \(.value)" ] | join(", ")
                else
                    tostring
                end
            ' 2>/dev/null | head -c 120 || echo "ok")
            [[ -z "$step_summary" ]] && step_summary="ok"

            # Extract report_path if present (scribe step)
            step_report_path=$(echo "$step_result" | jq -r '.result.report_path // "null"' 2>/dev/null || echo "null")
        else
            step_status="failed"
            local err_msg
            err_msg=$(echo "$step_result" | jq -r '.error // "unknown error"' 2>/dev/null || echo "unknown error")
            step_summary="$err_msg"
        fi

        steps_json=$(echo "$steps_json" | jq \
            --arg name "$step_name" \
            --arg status "$step_status" \
            --arg summary "$step_summary" \
            --arg rp "$step_report_path" \
            '. += [{
                "name":        $name,
                "status":      $status,
                "summary":     $summary,
                "report_path": (if $rp == "null" then null else $rp end)
            }]')
    }

    # ── Step 1: Full 8-ant curation run ────────────────────────────────────────
    _cs_step "curation-run" "curation-run"

    # ── Step 2: Final trust decay pass ─────────────────────────────────────────
    _cs_step "instinct-decay-all" "instinct-decay-all"

    # ── Step 3: Archive borderline instincts (threshold 0.3) ───────────────────
    # Note: --dry-run is injected by _cs_step; but archivist also takes --threshold
    # We cannot let _cs_step inject --dry-run before --threshold so we handle manually
    local arch_result arch_status arch_summary
    local -a arch_args=("curation-archivist" "--threshold" "0.3")
    if [[ "$dry_run" == "true" ]]; then
        arch_args+=("--dry-run")
    fi
    arch_result=$(bash "$0" "${arch_args[@]}" 2>/dev/null) || true
    if echo "$arch_result" | jq -e '.ok == true' >/dev/null 2>&1; then
        arch_status="ok"
        arch_summary=$(echo "$arch_result" | jq -r '
            .result |
            if type == "object" then
                [ to_entries[] | "\(.key) \(.value)" ] | join(", ")
            else tostring end
        ' 2>/dev/null | head -c 120 || echo "ok")
        [[ -z "$arch_summary" ]] && arch_summary="ok"
    else
        arch_status="failed"
        arch_summary=$(echo "$arch_result" | jq -r '.error // "unknown error"' 2>/dev/null || echo "unknown error")
    fi
    steps_json=$(echo "$steps_json" | jq \
        --arg status "$arch_status" \
        --arg summary "$arch_summary" \
        '. += [{"name":"archivist","status":$status,"summary":$summary,"report_path":null}]')

    # ── Step 4: Publish consolidation.seal event ───────────────────────────────
    if [[ "$dry_run" != "true" ]]; then
        local payload
        payload=$(jq -nc \
            --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
            --argjson steps "$steps_json" \
            '{"timestamp":$ts,"steps":$steps}')
        local ep_result
        ep_result=$(bash "$0" event-publish \
            --topic "consolidation.seal" \
            --payload "$payload" \
            --source "consolidation-seal" 2>/dev/null) || true
        if echo "$ep_result" | jq -e '.ok == true' >/dev/null 2>&1; then
            event_published="true"
        fi
    else
        # In dry-run mode, report event as published (no side effects)
        event_published="true"
    fi

    # ── Step 5: Generate final scribe report ───────────────────────────────────
    local scribe_result scribe_status scribe_summary scribe_report_path="null"
    local -a scribe_args=("curation-scribe")
    if [[ "$dry_run" == "true" ]]; then
        scribe_args+=("--dry-run")
    fi
    scribe_result=$(bash "$0" "${scribe_args[@]}" 2>/dev/null) || true
    if echo "$scribe_result" | jq -e '.ok == true' >/dev/null 2>&1; then
        scribe_status="ok"
        scribe_summary=$(echo "$scribe_result" | jq -r '
            .result |
            if type == "object" then
                [ to_entries[] | "\(.key) \(.value)" ] | join(", ")
            else tostring end
        ' 2>/dev/null | head -c 120 || echo "ok")
        [[ -z "$scribe_summary" ]] && scribe_summary="ok"
        scribe_report_path=$(echo "$scribe_result" | jq -r '.result.report_path // "null"' 2>/dev/null || echo "null")
    else
        scribe_status="failed"
        scribe_summary=$(echo "$scribe_result" | jq -r '.error // "unknown error"' 2>/dev/null || echo "unknown error")
    fi
    steps_json=$(echo "$steps_json" | jq \
        --arg status "$scribe_status" \
        --arg summary "$scribe_summary" \
        --arg rp "$scribe_report_path" \
        '. += [{
            "name":        "scribe",
            "status":      $status,
            "summary":     $summary,
            "report_path": (if $rp == "null" then null else $rp end)
        }]')

    # ── Output ─────────────────────────────────────────────────────────────────
    json_ok "$(jq -n \
        --argjson steps "$steps_json" \
        --argjson event_published "$([ "$event_published" == "true" ] && echo true || echo false)" \
        --argjson dry "$([ "$dry_run" == "true" ] && echo true || echo false)" \
        '{
            "type":            "seal",
            "steps":           $steps,
            "event_published": $event_published,
            "dry_run":         $dry
        }')"
}
