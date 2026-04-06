#!/bin/bash
# Nurse curation ant — trust recalculation for observations and instincts
# Provides: _curation_nurse
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write,
# COLONY_DATA_DIR, SCRIPT_DIR, error constants) is available.
#
# Subcommand: curation-nurse [--dry-run]
# Recalculates trust scores for all observations (that have source_type/evidence_type)
# and applies trust-decay to instincts based on days since created_at.

# ============================================================================
# _curation_nurse
# Recalculate trust scores across observations and instincts.
#
# Usage: curation-nurse [--dry-run]
#
# Output: {observations_updated: N, instincts_updated: N, dry_run: bool}
# ============================================================================
_curation_nurse() {
    local dry_run="false"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run) dry_run="true"; shift ;;
            *) shift ;;
        esac
    done

    local obs_updated=0
    local inst_updated=0
    local obs_file="$COLONY_DATA_DIR/learning-observations.json"
    local inst_file="$COLONY_DATA_DIR/instincts.json"
    local now_epoch
    now_epoch=$(date -u +%s)

    # ── Recalculate observation trust scores ──────────────────────────────────
    if [[ -f "$obs_file" ]]; then
        local obs_count
        obs_count=$(jq '[.observations[] | select(.source_type != null and .evidence_type != null)] | length' "$obs_file" 2>/dev/null || echo 0)

        if [[ "$obs_count" -gt 0 ]]; then
            local updated_obs
            updated_obs=$(jq -c '[.observations[]]' "$obs_file" 2>/dev/null || echo "[]")

            local new_obs_array="[]"
            while IFS= read -r obs_json; do
                local source_type evidence_type last_seen
                source_type=$(echo "$obs_json" | jq -r '.source_type // empty')
                evidence_type=$(echo "$obs_json" | jq -r '.evidence_type // empty')
                last_seen=$(echo "$obs_json" | jq -r '.last_seen // empty')

                if [[ -z "$source_type" || -z "$evidence_type" ]]; then
                    new_obs_array=$(echo "$new_obs_array" | jq --argjson entry "$obs_json" '. += [$entry]')
                    continue
                fi

                local days_since=0
                if [[ -n "$last_seen" ]]; then
                    local last_epoch
                    last_epoch=$(date -u -d "$last_seen" +%s 2>/dev/null || date -u -j -f "%Y-%m-%dT%H:%M:%SZ" "$last_seen" +%s 2>/dev/null || echo "$now_epoch")
                    days_since=$(( (now_epoch - last_epoch) / 86400 ))
                    [[ "$days_since" -lt 0 ]] && days_since=0
                fi

                local trust_result trust_score trust_tier
                trust_result=$(_trust_calculate --source "$source_type" --evidence "$evidence_type" --days-since "$days_since" 2>/dev/null) || true
                if echo "$trust_result" | jq -e '.ok == true' >/dev/null 2>&1; then
                    trust_score=$(echo "$trust_result" | jq -r '.result.score')
                    trust_tier=$(echo "$trust_result" | jq -r '.result.tier')
                    local updated_entry
                    updated_entry=$(echo "$obs_json" | jq \
                        --argjson score "$trust_score" \
                        --arg tier "$trust_tier" \
                        '. + {trust_score: $score, trust_tier: $tier}')
                    new_obs_array=$(echo "$new_obs_array" | jq --argjson entry "$updated_entry" '. += [$entry]')
                    obs_updated=$((obs_updated + 1))
                else
                    new_obs_array=$(echo "$new_obs_array" | jq --argjson entry "$obs_json" '. += [$entry]')
                fi
            done < <(echo "$updated_obs" | jq -c '.[]')

            if [[ "$dry_run" != "true" ]]; then
                local final_obs
                final_obs=$(jq --argjson obs "$new_obs_array" '.observations = $obs' "$obs_file" 2>/dev/null) || true
                [[ -n "$final_obs" ]] && atomic_write "$obs_file" "$final_obs"
            fi
        fi
    fi

    # ── Apply trust-decay to instincts ────────────────────────────────────────
    if [[ -f "$inst_file" ]]; then
        local active_count
        active_count=$(jq '[.instincts[] | select(.archived == false)] | length' "$inst_file" 2>/dev/null || echo 0)

        if [[ "$active_count" -gt 0 ]]; then
            local updated_inst_array="[]"
            local all_inst
            all_inst=$(jq -c '[.instincts[]]' "$inst_file" 2>/dev/null || echo "[]")

            while IFS= read -r inst_json; do
                local archived
                archived=$(echo "$inst_json" | jq -r '.archived // false')

                if [[ "$archived" == "true" ]]; then
                    updated_inst_array=$(echo "$updated_inst_array" | jq --argjson entry "$inst_json" '. += [$entry]')
                    continue
                fi

                local created_at days_inst=0
                created_at=$(echo "$inst_json" | jq -r '.provenance.created_at // empty')
                if [[ -n "$created_at" ]]; then
                    local created_epoch
                    created_epoch=$(date -u -d "$created_at" +%s 2>/dev/null || date -u -j -f "%Y-%m-%dT%H:%M:%SZ" "$created_at" +%s 2>/dev/null || echo "$now_epoch")
                    days_inst=$(( (now_epoch - created_epoch) / 86400 ))
                    [[ "$days_inst" -lt 0 ]] && days_inst=0
                fi

                local current_score
                current_score=$(echo "$inst_json" | jq -r '.trust_score // 0.5')
                local decay_result decayed_score new_tier
                decay_result=$(_trust_decay --score "$current_score" --days "$days_inst" 2>/dev/null) || true

                if echo "$decay_result" | jq -e '.ok == true' >/dev/null 2>&1; then
                    decayed_score=$(echo "$decay_result" | jq -r '.result.decayed')
                    new_tier=$(_trust_score_to_tier "$decayed_score" 2>/dev/null || echo "dormant")
                    local updated_entry
                    updated_entry=$(echo "$inst_json" | jq \
                        --argjson score "$decayed_score" \
                        --arg tier "$new_tier" \
                        '.trust_score = $score | .trust_tier = $tier')
                    updated_inst_array=$(echo "$updated_inst_array" | jq --argjson entry "$updated_entry" '. += [$entry]')
                    inst_updated=$((inst_updated + 1))
                else
                    updated_inst_array=$(echo "$updated_inst_array" | jq --argjson entry "$inst_json" '. += [$entry]')
                fi
            done < <(echo "$all_inst" | jq -c '.[]')

            if [[ "$dry_run" != "true" ]]; then
                local final_inst
                final_inst=$(jq --argjson insts "$updated_inst_array" '.instincts = $insts' "$inst_file" 2>/dev/null) || true
                [[ -n "$final_inst" ]] && atomic_write "$inst_file" "$final_inst"
            fi
        fi
    fi

    json_ok "$(jq -n \
        --argjson obs "$obs_updated" \
        --argjson inst "$inst_updated" \
        --argjson dry "$([ "$dry_run" == "true" ] && echo true || echo false)" \
        '{observations_updated: $obs, instincts_updated: $inst, dry_run: $dry}')"
}
