#!/usr/bin/env bash
# Graph Module Tests
# Tests graph.sh functions via aether-utils.sh subcommands:
#   graph-link, graph-neighbors, graph-reach, graph-cluster

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

if [[ ! -f "$AETHER_UTILS" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS"
    exit 1
fi

# ============================================================================
# Helper: isolated env with aether-utils.sh + all utils
# ============================================================================
setup_graph_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"

    cp "$AETHER_UTILS" "$tmpdir/.aether/aether-utils.sh"
    chmod +x "$tmpdir/.aether/aether-utils.sh"

    local utils_source
    utils_source="$(dirname "$AETHER_UTILS")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmpdir/.aether/"
    fi

    local exchange_source
    exchange_source="$(dirname "$AETHER_UTILS")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmpdir/.aether/"
    fi

    echo "$tmpdir"
}

run_cmd() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>/dev/null || true
}

run_cmd_with_stderr() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>&1 || true
}

# ============================================================================
# TEST 1: graph-link creates edge and graph file
# ============================================================================
test_graph_link_creates_edge() {
    local tmpdir
    tmpdir=$(setup_graph_env)

    local result
    result=$(run_cmd "$tmpdir" graph-link \
        --source instinct_abc --target instinct_xyz --relationship reinforces)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local action
    action=$(echo "$result" | jq -r '.result.action')
    [[ "$action" == "created" ]] || return 1

    local source
    source=$(echo "$result" | jq -r '.result.source')
    [[ "$source" == "instinct_abc" ]] || return 1

    local target
    target=$(echo "$result" | jq -r '.result.target')
    [[ "$target" == "instinct_xyz" ]] || return 1

    local relationship
    relationship=$(echo "$result" | jq -r '.result.relationship')
    [[ "$relationship" == "reinforces" ]] || return 1
}

# ============================================================================
# TEST 2: graph-link deduplication updates weight instead of duplicating
# ============================================================================
test_graph_link_deduplication() {
    local tmpdir
    tmpdir=$(setup_graph_env)

    # Create initial edge
    run_cmd "$tmpdir" graph-link \
        --source instinct_a --target instinct_b --relationship extends --weight 0.4 >/dev/null

    # Link same pair again with a different weight
    local result
    result=$(run_cmd "$tmpdir" graph-link \
        --source instinct_a --target instinct_b --relationship extends --weight 0.9)

    # Check that it was an update, not a create
    local action
    action=$(echo "$result" | jq -r '.result.action')

    # Verify graph file has only one edge for this pair
    local graph_file="$tmpdir/.aether/data/instinct-graph.json"
    local edge_count
    edge_count=$(jq '[.edges[] | select(.source == "instinct_a" and .target == "instinct_b" and .relationship == "extends")] | length' "$graph_file")

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1
    [[ "$action" == "updated" ]] || return 1
    [[ "$edge_count" -eq 1 ]] || return 1

    local new_weight
    new_weight=$(echo "$result" | jq -r '.result.weight')
    # Accept either string or number comparison
    [[ "$new_weight" == "0.9" ]] || [[ $(awk "BEGIN{print ($new_weight == 0.9)}") == "1" ]] || return 1
}

# ============================================================================
# TEST 3: graph-neighbors finds connected nodes (both directions)
# ============================================================================
test_graph_neighbors_both_directions() {
    local tmpdir
    tmpdir=$(setup_graph_env)

    # instinct_center -> instinct_out1 (outbound)
    run_cmd "$tmpdir" graph-link \
        --source instinct_center --target instinct_out1 --relationship reinforces >/dev/null

    # instinct_in1 -> instinct_center (inbound)
    run_cmd "$tmpdir" graph-link \
        --source instinct_in1 --target instinct_center --relationship extends >/dev/null

    local result
    result=$(run_cmd "$tmpdir" graph-neighbors --id instinct_center)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local count
    count=$(echo "$result" | jq -r '.result.count')
    [[ "$count" -eq 2 ]] || return 1

    # Verify both neighbors are present
    local neighbor_ids
    neighbor_ids=$(echo "$result" | jq -r '.result.neighbors[].id')
    echo "$neighbor_ids" | grep -q "instinct_out1" || return 1
    echo "$neighbor_ids" | grep -q "instinct_in1" || return 1
}

# ============================================================================
# TEST 4: graph-neighbors with direction filter (out only)
# ============================================================================
test_graph_neighbors_direction_filter() {
    local tmpdir
    tmpdir=$(setup_graph_env)

    # instinct_hub -> instinct_child1 (outbound)
    run_cmd "$tmpdir" graph-link \
        --source instinct_hub --target instinct_child1 --relationship reinforces >/dev/null

    # instinct_parent1 -> instinct_hub (inbound)
    run_cmd "$tmpdir" graph-link \
        --source instinct_parent1 --target instinct_hub --relationship supersedes >/dev/null

    # Request only outbound neighbors
    local result
    result=$(run_cmd "$tmpdir" graph-neighbors --id instinct_hub --direction out)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local count
    count=$(echo "$result" | jq -r '.result.count')
    [[ "$count" -eq 1 ]] || return 1

    local neighbor_id
    neighbor_id=$(echo "$result" | jq -r '.result.neighbors[0].id')
    [[ "$neighbor_id" == "instinct_child1" ]] || return 1

    local direction
    direction=$(echo "$result" | jq -r '.result.neighbors[0].direction')
    [[ "$direction" == "out" ]] || return 1
}

# ============================================================================
# TEST 5: graph-reach finds 2-hop reachable nodes
# ============================================================================
test_graph_reach_two_hops() {
    local tmpdir
    tmpdir=$(setup_graph_env)

    # Chain: instinct_root -> instinct_mid -> instinct_leaf
    run_cmd "$tmpdir" graph-link \
        --source instinct_root --target instinct_mid --relationship reinforces >/dev/null
    run_cmd "$tmpdir" graph-link \
        --source instinct_mid --target instinct_leaf --relationship extends >/dev/null

    local result
    result=$(run_cmd "$tmpdir" graph-reach --id instinct_root --hops 2)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local count
    count=$(echo "$result" | jq -r '.result.count')
    [[ "$count" -ge 2 ]] || return 1

    # instinct_mid should be at hop 1
    local mid_hop
    mid_hop=$(echo "$result" | jq -r '.result.reachable[] | select(.id == "instinct_mid") | .hop')
    [[ "$mid_hop" == "1" ]] || return 1

    # instinct_leaf should be at hop 2
    local leaf_hop
    leaf_hop=$(echo "$result" | jq -r '.result.reachable[] | select(.id == "instinct_leaf") | .hop')
    [[ "$leaf_hop" == "2" ]] || return 1
}

# ============================================================================
# TEST 6: graph-cluster identifies groups of connected instincts
# ============================================================================
test_graph_cluster_identifies_groups() {
    local tmpdir
    tmpdir=$(setup_graph_env)

    # Create a dense cluster: a <-> b <-> c (mutual references)
    run_cmd "$tmpdir" graph-link \
        --source instinct_g1 --target instinct_g2 --relationship reinforces --weight 0.8 >/dev/null
    run_cmd "$tmpdir" graph-link \
        --source instinct_g2 --target instinct_g3 --relationship reinforces --weight 0.8 >/dev/null
    run_cmd "$tmpdir" graph-link \
        --source instinct_g3 --target instinct_g1 --relationship reinforces --weight 0.8 >/dev/null

    local result
    result=$(run_cmd "$tmpdir" graph-cluster)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local count
    count=$(echo "$result" | jq -r '.result.count')
    [[ "$count" -ge 1 ]] || return 1

    # Verify clusters have required fields
    local has_nodes
    has_nodes=$(echo "$result" | jq 'if .result.clusters | length > 0 then .result.clusters[0] | has("nodes") else true end')
    [[ "$has_nodes" == "true" ]] || return 1
}

# ============================================================================
# TEST 7: Error handling — missing required arguments
# ============================================================================
test_graph_link_missing_args() {
    local tmpdir
    tmpdir=$(setup_graph_env)

    local result
    result=$(run_cmd_with_stderr "$tmpdir" graph-link 2>&1 || true)

    rm -rf "$tmpdir"

    assert_ok_false "$result" || [[ "$result" == *"Usage"* ]] || return 1
}

test_graph_neighbors_missing_args() {
    local tmpdir
    tmpdir=$(setup_graph_env)

    local result
    result=$(run_cmd_with_stderr "$tmpdir" graph-neighbors 2>&1 || true)

    rm -rf "$tmpdir"

    assert_ok_false "$result" || [[ "$result" == *"Usage"* ]] || return 1
}

test_graph_reach_missing_args() {
    local tmpdir
    tmpdir=$(setup_graph_env)

    local result
    result=$(run_cmd_with_stderr "$tmpdir" graph-reach 2>&1 || true)

    rm -rf "$tmpdir"

    assert_ok_false "$result" || [[ "$result" == *"Usage"* ]] || return 1
}

# ============================================================================
# Main: run all tests
# ============================================================================

log_info "Running graph module tests..."
log_info ""

run_test "test_graph_link_creates_edge"         "graph-link: creates edge and graph file"
run_test "test_graph_link_deduplication"        "graph-link: deduplication updates weight instead of creating duplicate"
run_test "test_graph_neighbors_both_directions" "graph-neighbors: finds nodes in both directions"
run_test "test_graph_neighbors_direction_filter" "graph-neighbors: direction filter returns only outbound"
run_test "test_graph_reach_two_hops"            "graph-reach: finds 2-hop reachable nodes with correct hop levels"
run_test "test_graph_cluster_identifies_groups" "graph-cluster: identifies clusters with required fields"
run_test "test_graph_link_missing_args"         "graph-link: missing args => error"
run_test "test_graph_neighbors_missing_args"    "graph-neighbors: missing args => error"
run_test "test_graph_reach_missing_args"        "graph-reach: missing args => error"

test_summary
