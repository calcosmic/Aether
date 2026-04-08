#!/bin/bash
# Curation Scribe — Memory Consolidation Report Generation
# Generates a markdown report of the memory consolidation state.
#
# Functions:
#   _curation_scribe
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, COLONY_DATA_DIR, DATA_DIR,
# error constants) is available when sourced.

# ============================================================================
# _curation_scribe
# Generate a markdown report of memory consolidation state.
# Usage: curation-scribe [--output <path>]
#
# Default output: $COLONY_DATA_DIR/curation-report.md
# Output: json_ok with {report_path:string, sections:N, generated_at:"ISO"}
# ============================================================================
_curation_scribe() {
    local csc_output=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --output)
                csc_output="${2:-}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    local csc_data_dir="${COLONY_DATA_DIR:-${DATA_DIR:-}}"
    if [[ -z "$csc_data_dir" ]]; then
        json_err "$E_VALIDATION_FAILED" "curation-scribe: COLONY_DATA_DIR is not set"
    fi

    [[ -z "$csc_output" ]] && csc_output="$csc_data_dir/curation-report.md"

    local csc_ts
    csc_ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local csc_ts_human
    csc_ts_human=$(date -u +"%Y-%m-%d %H:%M UTC")

    # Step 1: Gather librarian stats (via curation-librarian subcommand if available)
    local csc_total_instincts=0
    local csc_active_instincts=0
    local csc_archived_instincts=0
    local csc_total_observations=0
    local csc_total_events=0

    local csc_lib_result
    if csc_lib_result=$(COLONY_DATA_DIR="$csc_data_dir" DATA_DIR="$csc_data_dir" \
            bash "$0" curation-librarian 2>/dev/null); then
        if echo "$csc_lib_result" | jq -e '.ok == true' >/dev/null 2>&1; then
            csc_total_instincts=$(echo "$csc_lib_result"   | jq '.result.total_instincts   // 0')
            csc_active_instincts=$(echo "$csc_lib_result"  | jq '.result.active_instincts  // 0')
            csc_archived_instincts=$(echo "$csc_lib_result"| jq '.result.archived_instincts // 0')
            csc_total_observations=$(echo "$csc_lib_result"| jq '.result.total_observations // 0')
            csc_total_events=$(echo "$csc_lib_result"      | jq '.result.total_events       // 0')
        fi
    else
        # Gather stats directly if librarian is not yet available
        local csc_instincts_file="$csc_data_dir/instincts.json"
        if [[ -f "$csc_instincts_file" ]] && jq empty "$csc_instincts_file" 2>/dev/null; then
            csc_total_instincts=$(jq '.instincts | length' "$csc_instincts_file" 2>/dev/null || echo 0)
            csc_active_instincts=$(jq '[.instincts[] | select(.archived != true)] | length' "$csc_instincts_file" 2>/dev/null || echo 0)
            csc_archived_instincts=$(jq '[.instincts[] | select(.archived == true)] | length' "$csc_instincts_file" 2>/dev/null || echo 0)
        fi

        local csc_obs_file="$csc_data_dir/learning-observations.json"
        if [[ -f "$csc_obs_file" ]] && jq empty "$csc_obs_file" 2>/dev/null; then
            csc_total_observations=$(jq '.observations | length' "$csc_obs_file" 2>/dev/null || echo 0)
        fi

        local csc_eb_file="$csc_data_dir/event-bus.jsonl"
        if [[ -f "$csc_eb_file" ]]; then
            csc_total_events=$(wc -l < "$csc_eb_file" | tr -d ' ')
        fi
    fi

    # Step 2: Gather top 5 trusted instincts
    local csc_top_instincts=""
    local csc_instincts_file="$csc_data_dir/instincts.json"
    if [[ -f "$csc_instincts_file" ]] && jq empty "$csc_instincts_file" 2>/dev/null; then
        csc_top_instincts=$(jq -r \
            '[.instincts[] | select(.archived != true)] | sort_by(-.trust_score // -.confidence // 0) | .[0:5][] |
             "- **[\(.trust_score // .confidence // 0 | . * 100 | floor)%]** \(.trigger // "unknown trigger")"' \
            "$csc_instincts_file" 2>/dev/null || echo "")
    fi
    [[ -z "$csc_top_instincts" ]] && csc_top_instincts="_No instincts found._"

    # Step 3: Gather recent events (last 10)
    local csc_recent_events=""
    local csc_eb_file="$csc_data_dir/event-bus.jsonl"
    if [[ -f "$csc_eb_file" ]] && [[ -s "$csc_eb_file" ]]; then
        csc_recent_events=$(tail -10 "$csc_eb_file" | \
            jq -r '"- [\(.timestamp // "?")] **\(.topic // "unknown")**: \(.source // "system")"' \
            2>/dev/null || echo "")
    fi
    [[ -z "$csc_recent_events" ]] && csc_recent_events="_No recent events._"

    # Step 4: Generate recommendations
    local csc_recommendations=""
    if [[ "$csc_active_instincts" -gt 40 ]]; then
        csc_recommendations+="- Run \`curation-archivist\` to archive low-trust instincts (capacity at ${csc_active_instincts}/50)."$'\n'
    fi
    if [[ "$csc_total_events" -gt 100 ]]; then
        csc_recommendations+="- Run \`curation-janitor\` to clean up expired events (${csc_total_events} events on bus)."$'\n'
    fi
    if [[ "$csc_total_observations" -gt 200 ]]; then
        csc_recommendations+="- Consider pruning stale learning observations (${csc_total_observations} total)."$'\n'
    fi
    [[ -z "$csc_recommendations" ]] && csc_recommendations="_Memory stores are healthy. No immediate action needed._"

    # Step 5: Write report
    mkdir -p "$(dirname "$csc_output")"

    cat > "$csc_output" <<REPORT
# Memory Consolidation Report

_Generated: ${csc_ts_human}_

---

## Memory Health Summary

| Store | Count |
|-------|-------|
| Total instincts | ${csc_total_instincts} |
| Active instincts | ${csc_active_instincts} |
| Archived instincts | ${csc_archived_instincts} |
| Learning observations | ${csc_total_observations} |
| Events on bus | ${csc_total_events} |

---

## Top Trusted Instincts

${csc_top_instincts}

---

## Recent Events

${csc_recent_events}

---

## Recommendations

${csc_recommendations}
REPORT

    local csc_sections=4

    json_ok "$(jq -nc \
        --arg report_path "$csc_output" \
        --argjson sections "$csc_sections" \
        --arg generated_at "$csc_ts" \
        '{report_path:$report_path, sections:$sections, generated_at:$generated_at}')"
}
