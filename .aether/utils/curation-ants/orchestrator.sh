#!/bin/bash
# Curation Orchestrator — runs all 8 curation ants in sequence
# Provides: _curation_run
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, COLONY_DATA_DIR, DATA_DIR,
# error constants) is available when sourced.

# ============================================================================
# _curation_run
# Run all 8 curation ants in the correct order.
#
# Usage: curation-run [--dry-run] [--verbose]
#
# Execution order:
#   1. sentinel   — health check (abort if corrupt)
#   2. nurse      — recalculate trust scores
#   3. critic     — detect contradictions
#   4. herald     — promote high-trust to QUEEN.md
#   5. janitor    — clean expired events/archives
#   6. archivist  — archive low-trust instincts
#   7. librarian  — inventory stats
#   8. scribe     — generate report
#
# Output: json_ok with {steps, total_steps, succeeded, failed, dry_run,
#                        report_path, duration_ms}
# ============================================================================
_curation_run() {
    local dry_run="false"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)  dry_run="true";  shift ;;
            --verbose)  shift ;;
            *)          shift ;;
        esac
    done

    # Portable millisecond timer: try python3, fall back to seconds*1000
    local start_ms
    start_ms=$(python3 -c "import time; print(int(time.time()*1000))" 2>/dev/null \
               || echo $(( $(date +%s) * 1000 )))

    local steps_json="[]"
    local succeeded=0
    local failed=0
    local report_path="null"
    # Shared variable: holds the raw JSON result of the last step executed
    local _CR_LAST_RESULT=""

    # _cr_step <name> [extra_args...]
    # Runs curation-<name> (with --dry-run if set), updates steps_json,
    # succeeded/failed counters, and sets _CR_LAST_RESULT.
    # Must be called WITHOUT command substitution so variable mutations persist.
    _cr_step() {
        local step_name="$1"
        shift

        local cmd="curation-${step_name}"

        # Build the args list (avoid empty-array nounset issues)
        local step_result step_status step_summary
        if [[ "$dry_run" == "true" && $# -gt 0 ]]; then
            step_result=$(bash "$0" "$cmd" "--dry-run" "$@" 2>/dev/null) || true
        elif [[ "$dry_run" == "true" ]]; then
            step_result=$(bash "$0" "$cmd" "--dry-run" 2>/dev/null) || true
        elif [[ $# -gt 0 ]]; then
            step_result=$(bash "$0" "$cmd" "$@" 2>/dev/null) || true
        else
            step_result=$(bash "$0" "$cmd" 2>/dev/null) || true
        fi

        _CR_LAST_RESULT="$step_result"

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
            succeeded=$(( succeeded + 1 ))
        else
            step_status="failed"
            local err_msg
            err_msg=$(echo "$step_result" | jq -r '.error // "unknown error"' 2>/dev/null || echo "unknown error")
            step_summary="$err_msg"
            failed=$(( failed + 1 ))
        fi

        steps_json=$(echo "$steps_json" | jq \
            --arg name "$step_name" \
            --arg status "$step_status" \
            --arg summary "$step_summary" \
            '. += [{"name": $name, "status": $status, "summary": $summary}]')
    }

    # ── Step 1: Sentinel — health check ─────────────────────────────────────
    _cr_step "sentinel"

    # Abort remaining steps if sentinel found critical corruption
    local sentinel_status
    sentinel_status=$(echo "$steps_json" | jq -r '.[-1].status')
    if [[ "$sentinel_status" == "ok" ]]; then
        local corrupt_count
        corrupt_count=$(echo "$_CR_LAST_RESULT" | jq '[.result.checks[]? | select(.status == "corrupt")] | length' 2>/dev/null || echo 0)
        if [[ "$corrupt_count" -gt 0 ]]; then
            for skipped_step in nurse critic herald janitor archivist librarian scribe; do
                steps_json=$(echo "$steps_json" | jq \
                    --arg name "$skipped_step" \
                    '. += [{"name": $name, "status": "skipped", "summary": "skipped: sentinel detected corrupt stores"}]')
            done

            local end_ms
            end_ms=$(python3 -c "import time; print(int(time.time()*1000))" 2>/dev/null \
                     || echo $(( $(date +%s) * 1000 )))
            local duration_ms=$(( end_ms - start_ms ))

            json_ok "$(jq -n \
                --argjson steps "$steps_json" \
                --argjson total 8 \
                --argjson succeeded "$succeeded" \
                --argjson failed "$failed" \
                --argjson dry "$([ "$dry_run" == "true" ] && echo true || echo false)" \
                --argjson dur "$duration_ms" \
                '{steps:$steps, total_steps:$total, succeeded:$succeeded, failed:$failed,
                  dry_run:$dry, report_path:null, duration_ms:$dur}')"
            return
        fi
    fi

    # ── Step 2: Nurse — recalculate trust scores ─────────────────────────────
    _cr_step "nurse"

    # ── Step 3: Critic — detect contradictions ───────────────────────────────
    _cr_step "critic"

    # ── Step 4: Herald — promote high-trust to QUEEN.md ─────────────────────
    _cr_step "herald"

    # ── Step 5: Janitor — clean expired events and archives ──────────────────
    _cr_step "janitor"

    # ── Step 6: Archivist — archive low-trust instincts ─────────────────────
    _cr_step "archivist"

    # ── Step 7: Librarian — inventory stats ──────────────────────────────────
    _cr_step "librarian"

    # ── Step 8: Scribe — generate report ─────────────────────────────────────
    _cr_step "scribe"

    # Extract report_path from scribe result
    if echo "$_CR_LAST_RESULT" | jq -e '.ok == true' >/dev/null 2>&1; then
        local raw_path
        raw_path=$(echo "$_CR_LAST_RESULT" | jq -r '.result.report_path // empty' 2>/dev/null || echo "")
        if [[ -n "$raw_path" ]]; then
            report_path=$(printf '%s' "$raw_path" | jq -Rs '.')
        fi
    fi

    local end_ms
    end_ms=$(python3 -c "import time; print(int(time.time()*1000))" 2>/dev/null \
             || echo $(( $(date +%s) * 1000 )))
    local duration_ms=$(( end_ms - start_ms ))

    json_ok "$(jq -n \
        --argjson steps "$steps_json" \
        --argjson total 8 \
        --argjson succeeded "$succeeded" \
        --argjson failed "$failed" \
        --argjson dry "$([ "$dry_run" == "true" ] && echo true || echo false)" \
        --argjson report_path "$report_path" \
        --argjson dur "$duration_ms" \
        '{steps:$steps, total_steps:$total, succeeded:$succeeded, failed:$failed,
          dry_run:$dry, report_path:$report_path, duration_ms:$dur}')"
}
