#!/bin/bash
# Critic curation ant — contradiction detection between instincts
# Provides: _curation_critic
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write,
# COLONY_DATA_DIR, error constants) is available.
#
# Subcommand: curation-critic [--auto-resolve]
# Finds contradicting instincts using text heuristics and graph edges.

# ============================================================================
# _curation_critic_opposing_words
# Internal helper: check if two action strings oppose each other.
# Detects "always"/"never" and "do"/"don't" opposites in the same domain.
# Outputs "true" if opposing, "false" otherwise.
# ============================================================================
_curation_critic_opposing_words() {
    local action_a="$1"
    local action_b="$2"

    local low_a low_b
    low_a=$(echo "$action_a" | tr '[:upper:]' '[:lower:]')
    low_b=$(echo "$action_b" | tr '[:upper:]' '[:lower:]')

    # "always" in one and "never" in the other
    if [[ "$low_a" == *"always"* && "$low_b" == *"never"* ]]; then
        echo "true"; return
    fi
    if [[ "$low_a" == *"never"* && "$low_b" == *"always"* ]]; then
        echo "true"; return
    fi

    # "don't" or "do not" in one and plain action (no negation) in the other
    if [[ "$low_a" == *"don't"* || "$low_a" == *"do not"* ]]; then
        # Strip negation from a: "don't add X" -> "add X" — check overlap with b
        local stripped_a
        stripped_a=$(echo "$low_a" | sed "s/don't //g; s/do not //g")
        if [[ "$low_b" == *"$stripped_a"* || "$stripped_a" == *"$low_b"* ]]; then
            # Only flag if they share substantial content (>= 4 chars of overlap)
            local overlap_len=${#stripped_a}
            [[ "$overlap_len" -ge 4 ]] && echo "true" && return
        fi
    fi
    if [[ "$low_b" == *"don't"* || "$low_b" == *"do not"* ]]; then
        local stripped_b
        stripped_b=$(echo "$low_b" | sed "s/don't //g; s/do not //g")
        if [[ "$low_a" == *"$stripped_b"* || "$stripped_b" == *"$low_a"* ]]; then
            local overlap_len=${#stripped_b}
            [[ "$overlap_len" -ge 4 ]] && echo "true" && return
        fi
    fi

    echo "false"
}

# ============================================================================
# _curation_critic
# Find contradicting instincts and optionally auto-resolve them.
#
# Usage: curation-critic [--auto-resolve]
#
# Contradiction criteria:
#   - Same domain
#   - Opposing triggers (same trigger text) with opposing action keywords
#   - OR existing "contradicts" edge in the graph
#
# Auto-resolve: archive the lower-trust instinct, create "contradicts" graph edge.
#
# Output: {contradictions: [{instinct_a, instinct_b, reason, resolved}], count: N}
# ============================================================================
_curation_critic() {
    local auto_resolve="false"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --auto-resolve) auto_resolve="true"; shift ;;
            *) shift ;;
        esac
    done

    local inst_file="$COLONY_DATA_DIR/instincts.json"

    if [[ ! -f "$inst_file" ]]; then
        json_ok '{"contradictions":[],"count":0}'
        return
    fi

    # Load all active instincts into a bash-friendly form
    local active_instincts
    active_instincts=$(jq -c '[.instincts[] | select(.archived == false)]' "$inst_file" 2>/dev/null || echo "[]")

    local total_active
    total_active=$(echo "$active_instincts" | jq 'length')

    local contradictions_json="[]"
    local resolved_ids=()

    if [[ "$total_active" -ge 2 ]]; then
        # Compare all pairs using indexed access
        local i=0
        while [[ $i -lt $((total_active - 1)) ]]; do
            local inst_a
            inst_a=$(echo "$active_instincts" | jq -c ".[$i]")
            local id_a domain_a trigger_a action_a score_a
            id_a=$(echo "$inst_a" | jq -r '.id')
            domain_a=$(echo "$inst_a" | jq -r '.domain // ""')
            trigger_a=$(echo "$inst_a" | jq -r '.trigger // ""')
            action_a=$(echo "$inst_a" | jq -r '.action // ""')
            score_a=$(echo "$inst_a" | jq -r '.trust_score // 0')

            local j=$((i + 1))
            while [[ $j -lt $total_active ]]; do
                local inst_b
                inst_b=$(echo "$active_instincts" | jq -c ".[$j]")
                local id_b domain_b trigger_b action_b score_b
                id_b=$(echo "$inst_b" | jq -r '.id')
                domain_b=$(echo "$inst_b" | jq -r '.domain // ""')
                trigger_b=$(echo "$inst_b" | jq -r '.trigger // ""')
                action_b=$(echo "$inst_b" | jq -r '.action // ""')
                score_b=$(echo "$inst_b" | jq -r '.trust_score // 0')

                local contradiction_reason=""

                # Heuristic 1: Same domain + similar trigger + opposing actions
                if [[ -n "$domain_a" && "$domain_a" == "$domain_b" ]]; then
                    # Check if triggers are similar (first 40 chars match or substantial overlap)
                    local trig_prefix_a trig_prefix_b
                    trig_prefix_a=$(echo "$trigger_a" | cut -c1-40 | tr '[:upper:]' '[:lower:]')
                    trig_prefix_b=$(echo "$trigger_b" | cut -c1-40 | tr '[:upper:]' '[:lower:]')

                    if [[ "$trig_prefix_a" == "$trig_prefix_b" && -n "$trig_prefix_a" ]]; then
                        local opposing
                        opposing=$(_curation_critic_opposing_words "$action_a" "$action_b")
                        if [[ "$opposing" == "true" ]]; then
                            contradiction_reason="same domain and trigger with opposing actions"
                        fi
                    fi
                fi

                # Heuristic 2: Check existing "contradicts" graph edge
                if [[ -z "$contradiction_reason" ]]; then
                    local graph_file="$COLONY_DATA_DIR/instinct-graph.json"
                    if [[ -f "$graph_file" ]]; then
                        local has_edge
                        has_edge=$(jq -r \
                            --arg a "$id_a" --arg b "$id_b" \
                            '[.edges[] | select(
                                .relationship == "contradicts" and (
                                    (.source == $a and .target == $b) or
                                    (.source == $b and .target == $a)
                                )
                            )] | length' "$graph_file" 2>/dev/null || echo 0)
                        if [[ "$has_edge" -gt 0 ]]; then
                            contradiction_reason="existing contradicts graph edge"
                        fi
                    fi
                fi

                if [[ -n "$contradiction_reason" ]]; then
                    local resolved="false"

                    if [[ "$auto_resolve" == "true" ]]; then
                        # Archive the lower-trust instinct
                        local lower_id higher_id
                        local cmp
                        cmp=$(awk "BEGIN{print ($score_a >= $score_b)}" 2>/dev/null || echo "1")
                        if [[ "$cmp" == "1" ]]; then
                            lower_id="$id_b"
                            higher_id="$id_a"
                        else
                            lower_id="$id_a"
                            higher_id="$id_b"
                        fi

                        # Only archive if not already scheduled (avoid double-archive)
                        local already_resolving="false"
                        for rid in "${resolved_ids[@]:-}"; do
                            [[ "$rid" == "$lower_id" ]] && already_resolving="true" && break
                        done

                        if [[ "$already_resolving" != "true" ]]; then
                            _instinct_archive --id "$lower_id" >/dev/null 2>&1 || true
                            resolved_ids+=("$lower_id")
                            resolved="true"

                            # Create "contradicts" graph edge (best effort)
                            _graph_link --source "$higher_id" --target "$lower_id" \
                                --relationship contradicts >/dev/null 2>&1 || true
                        fi
                    fi

                    contradictions_json=$(echo "$contradictions_json" | jq \
                        --arg a "$id_a" \
                        --arg b "$id_b" \
                        --arg reason "$contradiction_reason" \
                        --argjson resolved "$([ "$resolved" == "true" ] && echo true || echo false)" \
                        '. += [{instinct_a: $a, instinct_b: $b, reason: $reason, resolved: $resolved}]')
                fi

                j=$((j + 1))
            done
            i=$((i + 1))
        done
    fi

    local count
    count=$(echo "$contradictions_json" | jq 'length')

    json_ok "$(jq -n \
        --argjson contradictions "$contradictions_json" \
        --argjson count "$count" \
        '{contradictions: $contradictions, count: $count}')"
}
