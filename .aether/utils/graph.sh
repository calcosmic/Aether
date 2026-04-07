# =============================================================================
# DEPRECATED — This script has been superseded by the Go binary (aether CLI).
# All functionality is now available via: aether <subcommand>
# Do NOT modify this file — it is retained for reference only.
# See: cmd/ (Go source) | Run: aether --help
# =============================================================================
#
#!/bin/bash
# Graph traversal layer for instinct relationships — Aether Structural Learning Stack
# Provides: _graph_link, _graph_neighbors, _graph_reach, _graph_cluster
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write,
# COLONY_DATA_DIR, SCRIPT_DIR, error constants) is available.
#
# Graph is stored as JSON at $COLONY_DATA_DIR/instinct-graph.json:
# {
#   "version": "1.0",
#   "edges": [
#     { "source": "id", "target": "id", "relationship": "type",
#       "weight": 0.5, "created_at": "ISO8601" }
#   ]
# }
#
# Relationship types: reinforces, contradicts, extends, supersedes, related

# ============================================================================
# _graph_init_file
# Ensure the graph file exists with an empty structure.
# Internal helper.
# ============================================================================
_graph_init_file() {
    local graph_file="$1"
    if [[ ! -f "$graph_file" ]]; then
        local dir
        dir="$(dirname "$graph_file")"
        mkdir -p "$dir"
        atomic_write "$graph_file" '{"version":"1.0","edges":[]}'
    fi
}

# ============================================================================
# _graph_link
# Create a directed edge between two instinct IDs. If the same
# source+target+relationship already exists, update the weight instead.
#
# Usage: graph-link --source <id> --target <id> --relationship <type>
#                   [--weight <float>]
#
# Relationship types: reinforces, contradicts, extends, supersedes, related
# Default weight: 0.5
#
# Output: {edge_id, source, target, relationship, weight, action}
# ============================================================================
_graph_link() {
    local source_id=""
    local target_id=""
    local relationship=""
    local weight="0.5"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --source)
                source_id="${2:-}"
                shift 2
                ;;
            --target)
                target_id="${2:-}"
                shift 2
                ;;
            --relationship)
                relationship="${2:-}"
                shift 2
                ;;
            --weight)
                weight="${2:-0.5}"
                shift 2
                ;;
            *)
                json_err "$E_VALIDATION_FAILED" "Usage: graph-link --source <id> --target <id> --relationship <type> [--weight <float>]"
                return
                ;;
        esac
    done

    [[ -z "$source_id" || -z "$target_id" || -z "$relationship" ]] && \
        json_err "$E_VALIDATION_FAILED" "Usage: graph-link --source <id> --target <id> --relationship <type> [--weight <float>]"

    # Validate relationship type
    case "$relationship" in
        reinforces|contradicts|extends|supersedes|related) ;;
        *)
            json_err "$E_VALIDATION_FAILED" "Unknown relationship: $relationship. Valid: reinforces, contradicts, extends, supersedes, related"
            return
            ;;
    esac

    # Validate weight is a non-negative number
    if ! [[ "$weight" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
        json_err "$E_VALIDATION_FAILED" "--weight must be a non-negative number, got: $weight"
        return
    fi

    local graph_file="$COLONY_DATA_DIR/instinct-graph.json"
    _graph_init_file "$graph_file"

    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Check if a matching edge (same source+target+relationship) exists
    local existing_edge_id
    existing_edge_id=$(jq -r \
        --arg src "$source_id" \
        --arg tgt "$target_id" \
        --arg rel "$relationship" \
        '.edges[] | select(.source == $src and .target == $tgt and .relationship == $rel) | .edge_id' \
        "$graph_file" | head -1)

    local action
    local updated
    if [[ -n "$existing_edge_id" ]]; then
        # Update existing edge weight
        action="updated"
        updated=$(jq \
            --arg src "$source_id" \
            --arg tgt "$target_id" \
            --arg rel "$relationship" \
            --argjson w "$weight" \
            '.edges = [.edges[] | if (.source == $src and .target == $tgt and .relationship == $rel) then .weight = $w else . end]' \
            "$graph_file")
        local edge_id="$existing_edge_id"
    else
        # Create new edge
        action="created"
        local edge_id
        edge_id="edge_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' \n')"
        updated=$(jq \
            --arg eid "$edge_id" \
            --arg src "$source_id" \
            --arg tgt "$target_id" \
            --arg rel "$relationship" \
            --argjson w "$weight" \
            --arg ts "$ts" \
            '.edges += [{edge_id: $eid, source: $src, target: $tgt, relationship: $rel, weight: $w, created_at: $ts}]' \
            "$graph_file")
    fi

    atomic_write "$graph_file" "$updated"

    # Re-read the final edge_id for the output (handles both create and update paths)
    local final_edge_id
    final_edge_id=$(jq -r \
        --arg src "$source_id" \
        --arg tgt "$target_id" \
        --arg rel "$relationship" \
        '.edges[] | select(.source == $src and .target == $tgt and .relationship == $rel) | .edge_id' \
        "$graph_file" | head -1)

    json_ok "$(jq -n \
        --arg edge_id "$final_edge_id" \
        --arg source "$source_id" \
        --arg target "$target_id" \
        --arg relationship "$relationship" \
        --argjson weight "$weight" \
        --arg action "$action" \
        '{
            edge_id: $edge_id,
            source: $source,
            target: $target,
            relationship: $relationship,
            weight: $weight,
            action: $action
        }')"
}

# ============================================================================
# _graph_neighbors
# Find all nodes connected to a given instinct (1-hop).
#
# Usage: graph-neighbors --id <instinct_id> [--direction out|in|both]
#                        [--relationship <type>]
#
# Default direction: both
#
# Output: {neighbors: [{id, relationship, weight, direction}], count}
# ============================================================================
_graph_neighbors() {
    local instinct_id=""
    local direction="both"
    local filter_rel=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id)
                instinct_id="${2:-}"
                shift 2
                ;;
            --direction)
                direction="${2:-both}"
                shift 2
                ;;
            --relationship)
                filter_rel="${2:-}"
                shift 2
                ;;
            *)
                json_err "$E_VALIDATION_FAILED" "Usage: graph-neighbors --id <instinct_id> [--direction out|in|both] [--relationship <type>]"
                return
                ;;
        esac
    done

    [[ -z "$instinct_id" ]] && \
        json_err "$E_VALIDATION_FAILED" "Usage: graph-neighbors --id <instinct_id> [--direction out|in|both] [--relationship <type>]"

    case "$direction" in
        out|in|both) ;;
        *)
            json_err "$E_VALIDATION_FAILED" "Invalid direction: $direction. Valid: out, in, both"
            return
            ;;
    esac

    local graph_file="$COLONY_DATA_DIR/instinct-graph.json"

    # Return empty if graph file doesn't exist yet
    if [[ ! -f "$graph_file" ]]; then
        json_ok '{"neighbors":[],"count":0}'
        return
    fi

    local neighbors
    neighbors=$(jq -c \
        --arg id "$instinct_id" \
        --arg dir "$direction" \
        --arg rel "$filter_rel" \
        '
        [
            # Outbound edges: id is the source
            if ($dir == "out" or $dir == "both") then
                .edges[]
                | select(.source == $id)
                | select($rel == "" or .relationship == $rel)
                | {id: .target, relationship: .relationship, weight: .weight, direction: "out"}
            else empty end
            ,
            # Inbound edges: id is the target
            if ($dir == "in" or $dir == "both") then
                .edges[]
                | select(.target == $id)
                | select($rel == "" or .relationship == $rel)
                | {id: .source, relationship: .relationship, weight: .weight, direction: "in"}
            else empty end
        ]
        ' \
        "$graph_file")

    local count
    count=$(echo "$neighbors" | jq 'length')

    json_ok "$(jq -n \
        --argjson neighbors "$neighbors" \
        --argjson count "$count" \
        '{neighbors: $neighbors, count: $count}')"
}

# ============================================================================
# _graph_reach
# Find all nodes reachable within N hops using iterative BFS.
# One jq call per hop level to avoid complex recursive expressions.
#
# Usage: graph-reach --id <instinct_id> --hops <N> [--min-weight <float>]
#
# Max hops enforced at 3 to prevent expensive traversals.
# Default min-weight: 0.0
#
# Output: {reachable: [{id, hop, path}], count, hops_searched}
# ============================================================================
_graph_reach() {
    local instinct_id=""
    local hops=""
    local min_weight="0.0"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id)
                instinct_id="${2:-}"
                shift 2
                ;;
            --hops)
                hops="${2:-}"
                shift 2
                ;;
            --min-weight)
                min_weight="${2:-0.0}"
                shift 2
                ;;
            *)
                json_err "$E_VALIDATION_FAILED" "Usage: graph-reach --id <instinct_id> --hops <N> [--min-weight <float>]"
                return
                ;;
        esac
    done

    [[ -z "$instinct_id" || -z "$hops" ]] && \
        json_err "$E_VALIDATION_FAILED" "Usage: graph-reach --id <instinct_id> --hops <N> [--min-weight <float>]"

    # Validate hops is a positive integer
    if ! [[ "$hops" =~ ^[0-9]+$ ]] || [[ "$hops" -lt 1 ]]; then
        json_err "$E_VALIDATION_FAILED" "--hops must be a positive integer, got: $hops"
        return
    fi

    # Clamp hops to max 3
    local MAX_HOPS=3
    local hops_searched="$hops"
    if [[ "$hops" -gt "$MAX_HOPS" ]]; then
        hops_searched="$MAX_HOPS"
    fi

    # Validate min_weight
    if ! [[ "$min_weight" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
        json_err "$E_VALIDATION_FAILED" "--min-weight must be a non-negative number, got: $min_weight"
        return
    fi

    local graph_file="$COLONY_DATA_DIR/instinct-graph.json"

    if [[ ! -f "$graph_file" ]]; then
        json_ok "{\"reachable\":[],\"count\":0,\"hops_searched\":$hops_searched}"
        return
    fi

    # Read all edges once
    local all_edges
    all_edges=$(jq -c '.edges' "$graph_file")

    # BFS: frontier is the set of IDs at the current hop level
    # reachable accumulates {id, hop, path} for all visited nodes
    # visited tracks IDs seen to avoid cycles

    # frontier: JSON array of {id, path} objects
    local frontier
    frontier="[{\"id\":\"$instinct_id\",\"path\":[\"$instinct_id\"]}]"
    local visited
    visited="[\"$instinct_id\"]"
    local reachable="[]"

    local current_hop=0
    while [[ "$current_hop" -lt "$hops_searched" ]]; do
        current_hop=$((current_hop + 1))

        # Expand frontier: find all outbound neighbors not yet visited
        local new_nodes
        new_nodes=$(jq -c \
            --argjson edges "$all_edges" \
            --argjson frontier "$frontier" \
            --argjson visited "$visited" \
            --argjson hop "$current_hop" \
            --argjson mw "$min_weight" \
            '
            [
                $frontier[] as $f |
                $edges[] |
                select(.source == $f.id) |
                select(.weight >= $mw) |
                select(.target as $t | ($visited | index($t)) == null) |
                {id: .target, hop: $hop, path: ($f.path + [.target])}
            ] | unique_by(.id)
            ' \
            <<< "null")

        # If no new nodes found, stop early
        local new_count
        new_count=$(echo "$new_nodes" | jq 'length')
        if [[ "$new_count" -eq 0 ]]; then
            break
        fi

        # Append new nodes to reachable
        reachable=$(jq -c \
            --argjson existing "$reachable" \
            --argjson new "$new_nodes" \
            '$existing + $new' \
            <<< "null")

        # Update visited set
        visited=$(jq -c \
            --argjson visited "$visited" \
            --argjson new "$new_nodes" \
            '$visited + [$new[].id]' \
            <<< "null")

        # Update frontier for next hop
        frontier=$(jq -c \
            --argjson new "$new_nodes" \
            '[.[] | {id: .id, path: .path}]' \
            <<< "$new_nodes")
    done

    local count
    count=$(echo "$reachable" | jq 'length')

    json_ok "$(jq -n \
        --argjson reachable "$reachable" \
        --argjson count "$count" \
        --argjson hops_searched "$hops_searched" \
        '{reachable: $reachable, count: $count, hops_searched: $hops_searched}')"
}

# ============================================================================
# _graph_cluster
# Find clusters of strongly connected instincts.
# A cluster is a group of nodes that share >= min-edges connections
# all with weight >= min-weight.
#
# Usage: graph-cluster [--min-edges <N>] [--min-weight <float>]
#
# Default min-edges: 2
# Default min-weight: 0.3
#
# Output: {clusters: [{nodes, edge_count, avg_weight}], count}
# ============================================================================
_graph_cluster() {
    local min_edges=2
    local min_weight="0.3"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --min-edges)
                min_edges="${2:-2}"
                shift 2
                ;;
            --min-weight)
                min_weight="${2:-0.3}"
                shift 2
                ;;
            *)
                json_err "$E_VALIDATION_FAILED" "Usage: graph-cluster [--min-edges <N>] [--min-weight <float>]"
                return
                ;;
        esac
    done

    # Validate min_edges
    if ! [[ "$min_edges" =~ ^[0-9]+$ ]]; then
        json_err "$E_VALIDATION_FAILED" "--min-edges must be a non-negative integer, got: $min_edges"
        return
    fi

    # Validate min_weight
    if ! [[ "$min_weight" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
        json_err "$E_VALIDATION_FAILED" "--min-weight must be a non-negative number, got: $min_weight"
        return
    fi

    local graph_file="$COLONY_DATA_DIR/instinct-graph.json"

    if [[ ! -f "$graph_file" ]]; then
        json_ok '{"clusters":[],"count":0}'
        return
    fi

    # Strategy:
    # 1. Filter edges by min-weight
    # 2. For each node, count qualifying edges (in + out)
    # 3. Nodes with >= min-edges form candidates
    # 4. Group connected candidate nodes into clusters via union-find in jq

    local clusters
    clusters=$(jq -c \
        --argjson min_edges "$min_edges" \
        --argjson min_weight "$min_weight" \
        '
        # Step 1: filter qualifying edges
        (.edges | map(select(.weight >= $min_weight))) as $qual_edges |

        # Step 2: count edges per node (source + target)
        (
            $qual_edges |
            group_by(.source) |
            map({key: .[0].source, value: length}) |
            from_entries
        ) as $out_counts |
        (
            $qual_edges |
            group_by(.target) |
            map({key: .[0].target, value: length}) |
            from_entries
        ) as $in_counts |

        # Step 3: build total edge counts per node
        (
            [$qual_edges[] | .source, .target] | unique |
            map({
                id: .,
                edge_count: ((($out_counts[.] // 0) + ($in_counts[.] // 0)))
            }) |
            map(select(.edge_count >= $min_edges))
        ) as $candidates |

        # Step 4: group candidates into clusters via connected components
        # Build adjacency: pairs of candidate nodes that share a qualifying edge
        (
            $candidates | map(.id)
        ) as $candidate_ids |

        (
            $qual_edges |
            map(select(
                (.source as $s | ($candidate_ids | index($s)) != null) and
                (.target as $t | ($candidate_ids | index($t)) != null)
            )) |
            map({a: .source, b: .target, w: .weight})
        ) as $adj |

        # Build clusters: use greedy union approach
        # Start each candidate in its own group, merge groups with shared edges
        reduce $adj[] as $edge (
            ($candidate_ids | map({id: ., group: .}));
            . as $groups |
            ($groups | map(select(.id == $edge.a)) | first.group) as $ga |
            ($groups | map(select(.id == $edge.b)) | first.group) as $gb |
            if $ga == $gb then $groups
            else
                # Merge all nodes with group $gb into group $ga
                [ .[] | if .group == $gb then .group = $ga else . end ]
            end
        ) |
        # Group by cluster label
        group_by(.group) |
        map({
            nodes: map(.id),
            edge_count: (
                . as $cluster_nodes |
                ($cluster_nodes | map(.id)) as $node_ids |
                $adj | map(select(
                    (.a as $a | ($node_ids | index($a)) != null) and
                    (.b as $b | ($node_ids | index($b)) != null)
                )) | length
            ),
            avg_weight: (
                . as $cluster_nodes |
                ($cluster_nodes | map(.id)) as $node_ids |
                ($adj | map(select(
                    (.a as $a | ($node_ids | index($a)) != null) and
                    (.b as $b | ($node_ids | index($b)) != null)
                )) | map(.w)) as $weights |
                if ($weights | length) > 0 then
                    ($weights | add) / ($weights | length)
                else 0 end
            )
        }) |
        # Only keep clusters with more than 1 node
        map(select(.nodes | length > 1))
        ' \
        "$graph_file")

    local count
    count=$(echo "$clusters" | jq 'length')

    json_ok "$(jq -n \
        --argjson clusters "$clusters" \
        --argjson count "$count" \
        '{clusters: $clusters, count: $count}')"
}
