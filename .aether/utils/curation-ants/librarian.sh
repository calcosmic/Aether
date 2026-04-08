#!/bin/bash
# Librarian curation ant — inventory statistics across all memory stores
# Provides: _curation_librarian
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, COLONY_DATA_DIR, error constants)
# is available.
#
# Subcommand: curation-librarian
# Generates inventory statistics for all colony memory stores.

# ============================================================================
# _curation_librarian
# Generate inventory statistics across all memory stores.
#
# Usage: curation-librarian
#
# Output:
#   {
#     observations: N,
#     instincts: {total: N, active: N, archived: N},
#     graph_edges: N,
#     events: N,
#     signals: {active: N, total: N},
#     midden: N,
#     generated_at: "ISO8601"
#   }
# ============================================================================
_curation_librarian() {
    local obs_file="$COLONY_DATA_DIR/learning-observations.json"
    local inst_file="$COLONY_DATA_DIR/instincts.json"
    local graph_file="$COLONY_DATA_DIR/instinct-graph.json"
    local event_file="$COLONY_DATA_DIR/event-bus.jsonl"
    local pheromone_file="$COLONY_DATA_DIR/pheromones.json"
    local midden_file="$COLONY_DATA_DIR/midden/midden.json"

    # Observations count
    local obs_count=0
    if [[ -f "$obs_file" ]]; then
        obs_count=$(jq '[.observations[]] | length' "$obs_file" 2>/dev/null || echo 0)
    fi

    # Instinct counts: total, active, archived
    local inst_total=0 inst_active=0 inst_archived=0
    if [[ -f "$inst_file" ]]; then
        inst_total=$(jq '[.instincts[]] | length' "$inst_file" 2>/dev/null || echo 0)
        inst_active=$(jq '[.instincts[] | select(.archived == false)] | length' "$inst_file" 2>/dev/null || echo 0)
        inst_archived=$(jq '[.instincts[] | select(.archived == true)] | length' "$inst_file" 2>/dev/null || echo 0)
    fi

    # Graph edge count
    local edge_count=0
    if [[ -f "$graph_file" ]]; then
        edge_count=$(jq '[.edges[]] | length' "$graph_file" 2>/dev/null || echo 0)
    fi

    # Event count (JSONL — count non-empty lines)
    local event_count=0
    if [[ -f "$event_file" ]]; then
        event_count=$(grep -c '.' "$event_file" 2>/dev/null || echo 0)
    fi

    # Signal counts: active, total
    local sig_active=0 sig_total=0
    if [[ -f "$pheromone_file" ]]; then
        sig_total=$(jq '[.signals[]] | length' "$pheromone_file" 2>/dev/null || echo 0)
        sig_active=$(jq '[.signals[] | select(.active == true)] | length' "$pheromone_file" 2>/dev/null || echo 0)
    fi

    # Midden entry count
    local midden_count=0
    if [[ -f "$midden_file" ]]; then
        midden_count=$(jq '[.entries[]] | length' "$midden_file" 2>/dev/null || echo 0)
    fi

    local generated_at
    generated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    json_ok "$(jq -n \
        --argjson obs "$obs_count" \
        --argjson inst_total "$inst_total" \
        --argjson inst_active "$inst_active" \
        --argjson inst_archived "$inst_archived" \
        --argjson edges "$edge_count" \
        --argjson events "$event_count" \
        --argjson sig_active "$sig_active" \
        --argjson sig_total "$sig_total" \
        --argjson midden "$midden_count" \
        --arg generated_at "$generated_at" \
        '{
            observations: $obs,
            instincts: {total: $inst_total, active: $inst_active, archived: $inst_archived},
            graph_edges: $edges,
            events: $events,
            signals: {active: $sig_active, total: $sig_total},
            midden: $midden,
            generated_at: $generated_at
        }')"
}
