#!/bin/bash
# Herald curation ant — QUEEN.md promotion of high-trust instincts
# Provides: _curation_herald
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write,
# COLONY_DATA_DIR, AETHER_ROOT, SCRIPT_DIR, error constants) is available.
#
# Subcommand: curation-herald [--min-trust <float>] [--dry-run]
# Promotes instincts with trust_score >= min-trust to QUEEN.md wisdom.

# ============================================================================
# _curation_herald
# Promote high-trust instincts to QUEEN.md.
#
# Usage: curation-herald [--min-trust <float>] [--dry-run]
# Default min-trust: 0.75
#
# Output: {eligible: N, promoted: N, already_in_queen: N, dry_run: bool}
# ============================================================================
_curation_herald() {
    local min_trust="0.75"
    local dry_run="false"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --min-trust) min_trust="${2:-0.75}"; shift 2 ;;
            --dry-run)   dry_run="true";         shift   ;;
            *)           shift ;;
        esac
    done

    local inst_file="$COLONY_DATA_DIR/instincts.json"

    if [[ ! -f "$inst_file" ]]; then
        json_ok "$(jq -n --argjson dry "$([ "$dry_run" == "true" ] && echo true || echo false)" \
            '{eligible: 0, promoted: 0, already_in_queen: 0, dry_run: $dry}')"
        return
    fi

    # Collect eligible instincts: active, trust_score >= min_trust
    local eligible_instincts
    eligible_instincts=$(jq -c \
        --argjson min "$min_trust" \
        '[.instincts[] | select(.archived == false and .trust_score >= $min)]' \
        "$inst_file" 2>/dev/null || echo "[]")

    local eligible_count
    eligible_count=$(echo "$eligible_instincts" | jq 'length')

    local promoted=0
    local already_in_queen=0

    if [[ "$eligible_count" -gt 0 && "$dry_run" != "true" ]]; then
        while IFS= read -r inst_json; do
            local trigger action confidence domain
            trigger=$(echo "$inst_json" | jq -r '.trigger // ""')
            action=$(echo "$inst_json" | jq -r '.action // ""')
            confidence=$(echo "$inst_json" | jq -r '.confidence // 0.75')
            domain=$(echo "$inst_json" | jq -r '.domain // "workflow"')

            [[ -z "$trigger" || -z "$action" ]] && continue

            local promote_result
            promote_result=$(_queen_promote_instinct "$trigger" "$action" "$confidence" "$domain" 2>/dev/null) || true

            if echo "$promote_result" | jq -e '.ok == true' >/dev/null 2>&1; then
                local was_promoted
                was_promoted=$(echo "$promote_result" | jq -r '.result.promoted // false')
                local reason
                reason=$(echo "$promote_result" | jq -r '.result.reason // ""')

                if [[ "$was_promoted" == "true" ]]; then
                    promoted=$((promoted + 1))
                elif [[ "$reason" == "duplicate" ]]; then
                    already_in_queen=$((already_in_queen + 1))
                fi
            fi
        done < <(echo "$eligible_instincts" | jq -c '.[]')
    elif [[ "$eligible_count" -gt 0 && "$dry_run" == "true" ]]; then
        # Dry run: count what would be promoted vs already present
        local queen_file="${AETHER_ROOT:-}/.aether/QUEEN.md"
        while IFS= read -r inst_json; do
            local action
            action=$(echo "$inst_json" | jq -r '.action // ""')
            [[ -z "$action" ]] && continue

            if [[ -f "$queen_file" ]] && grep -Fq -- "$action" "$queen_file" 2>/dev/null; then
                already_in_queen=$((already_in_queen + 1))
            else
                promoted=$((promoted + 1))
            fi
        done < <(echo "$eligible_instincts" | jq -c '.[]')
    fi

    json_ok "$(jq -n \
        --argjson eligible "$eligible_count" \
        --argjson promoted "$promoted" \
        --argjson already "$already_in_queen" \
        --argjson dry "$([ "$dry_run" == "true" ] && echo true || echo false)" \
        '{eligible: $eligible, promoted: $promoted, already_in_queen: $already, dry_run: $dry}')"
}
