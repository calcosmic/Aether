#!/usr/bin/env bash
# Spawn Module Smoke Tests
# Tests spawn.sh extracted module functions via aether-utils.sh subcommands

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"

# Source test helpers
source "$SCRIPT_DIR/test-helpers.sh"

# Verify jq is available
require_jq

# Verify aether-utils.sh exists
if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated test environment with spawn support
# ============================================================================
setup_spawn_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils"

    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    # Write a minimal COLONY_STATE.json with spawn_tree and workers
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'CSEOF'
{
  "version": "3.0",
  "goal": "Test spawn module",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "plan": { "phases": [{ "id": 1, "name": "Test Phase", "status": "pending" }] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": [],
  "workers": [],
  "spawn_tree": []
}
CSEOF

    echo "$tmp_dir"
}

run_spawn_cmd() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

# ============================================================================
# Test: spawn.sh module file exists and has valid syntax
# ============================================================================
test_module_exists() {
    local module_path="$PROJECT_ROOT/.aether/utils/spawn.sh"

    assert_file_exists "$module_path" || return 1
    bash -n "$module_path" 2>/dev/null || return 1
}

# ============================================================================
# Test: spawn-can-spawn returns JSON with can_spawn field
# ============================================================================
test_spawn_can_spawn() {
    local tmp_dir
    tmp_dir=$(setup_spawn_env)

    local result
    result=$(run_spawn_cmd "$tmp_dir" spawn-can-spawn 1)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local can_spawn
    can_spawn=$(echo "$result" | jq -r '.result.can_spawn')
    [[ "$can_spawn" == "true" || "$can_spawn" == "false" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: spawn-get-depth returns a depth value
# ============================================================================
test_spawn_get_depth() {
    local tmp_dir
    tmp_dir=$(setup_spawn_env)

    local result
    result=$(run_spawn_cmd "$tmp_dir" spawn-get-depth Queen)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.depth" "0" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: spawn-efficiency returns valid JSON with efficiency metrics
# ============================================================================
test_spawn_efficiency() {
    local tmp_dir
    tmp_dir=$(setup_spawn_env)

    local result
    result=$(run_spawn_cmd "$tmp_dir" spawn-efficiency)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local total
    total=$(echo "$result" | jq -r '.result.total')
    [[ "$total" =~ ^[0-9]+$ ]] || { rm -rf "$tmp_dir"; return 1; }

    local efficiency
    efficiency=$(echo "$result" | jq -r '.result.efficiency_pct')
    [[ "$efficiency" =~ ^[0-9]+$ ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
echo "=== Spawn Module Smoke Tests ==="
echo ""

run_test test_module_exists "spawn.sh exists and passes syntax check"
run_test test_spawn_can_spawn "spawn-can-spawn returns JSON with can_spawn field"
run_test test_spawn_get_depth "spawn-get-depth returns correct depth for Queen"
run_test test_spawn_efficiency "spawn-efficiency returns valid efficiency metrics"

test_summary
