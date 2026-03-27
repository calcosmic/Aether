#!/usr/bin/env bash
# Swarm Module Smoke Tests
# Tests swarm.sh extracted module functions via aether-utils.sh subcommands

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
# Helper: Create isolated test environment with swarm support
# ============================================================================
setup_swarm_env() {
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

    # Write a minimal COLONY_STATE.json
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'CSEOF'
{
  "version": "3.0",
  "goal": "Test swarm module",
  "state": "READY",
  "current_phase": 1,
  "milestone": "First Mound",
  "session_id": "test-swarm",
  "initialized_at": "2026-01-01T00:00:00Z",
  "tasks": [],
  "events": [],
  "instincts": [],
  "errors": {"count": 0, "last_error": null},
  "memory": {"observations": [], "key_decisions": [], "learnings": []}
}
CSEOF

    echo "$tmp_dir"
}

run_swarm_cmd() {
    local tmp_dir="$1"
    shift
    bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

# ============================================================================
# Test 1: Module exists and passes syntax check
# ============================================================================
test_module_exists() {
    local module_path="$PROJECT_ROOT/.aether/utils/swarm.sh"
    assert_file_exists "$module_path" || return 1
    bash -n "$module_path" 2>/dev/null || return 1
}

# ============================================================================
# Test 2: swarm-findings-init creates findings state
# ============================================================================
test_swarm_findings_init() {
    local tmp_dir
    tmp_dir=$(setup_swarm_env)

    local result
    result=$(run_swarm_cmd "$tmp_dir" swarm-findings-init "test-swarm-123")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local swarm_id
    swarm_id=$(echo "$result" | jq -r '.result.swarm_id')
    [[ "$swarm_id" == "test-swarm-123" ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify findings file was created
    [[ -f "$tmp_dir/.aether/data/swarm-findings-test-swarm-123.json" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 3: swarm-timing-start records timing data
# ============================================================================
test_swarm_timing_start() {
    local tmp_dir
    tmp_dir=$(setup_swarm_env)

    local result
    result=$(run_swarm_cmd "$tmp_dir" swarm-timing-start "test-builder-1")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local ant
    ant=$(echo "$result" | jq -r '.result.ant')
    [[ "$ant" == "test-builder-1" ]] || { rm -rf "$tmp_dir"; return 1; }

    local started_at
    started_at=$(echo "$result" | jq -r '.result.started_at')
    [[ -n "$started_at" && "$started_at" != "null" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 4: swarm-activity-log records activity
# ============================================================================
test_swarm_activity_log() {
    local tmp_dir
    tmp_dir=$(setup_swarm_env)

    local result
    result=$(run_swarm_cmd "$tmp_dir" swarm-activity-log "test-scout" "started" "exploring codebase")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local logged
    logged=$(echo "$result" | jq -r '.result')
    [[ "$logged" == "logged" ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify log file exists
    [[ -f "$tmp_dir/.aether/data/swarm-activity.log" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
echo "=== Swarm Module Smoke Tests ==="
echo ""

run_test test_module_exists "swarm.sh exists and passes syntax check"
run_test test_swarm_findings_init "swarm-findings-init creates findings state"
run_test test_swarm_timing_start "swarm-timing-start records timing data"
run_test test_swarm_activity_log "swarm-activity-log records activity"

test_summary
