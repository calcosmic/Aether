#!/usr/bin/env bash
# Session Module Smoke Tests
# Tests session.sh extracted module functions via aether-utils.sh subcommands

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
# Helper: Create isolated test environment with session support
# ============================================================================
setup_session_env() {
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
  "goal": "Test session module",
  "state": "READY",
  "current_phase": 1,
  "milestone": "First Mound",
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

run_session_cmd() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

# ============================================================================
# Test: session.sh module file exists and has valid syntax
# ============================================================================
test_module_exists() {
    local module_path="$PROJECT_ROOT/.aether/utils/session.sh"

    assert_file_exists "$module_path" || return 1
    bash -n "$module_path" 2>/dev/null || return 1
}

# ============================================================================
# Test: session-init creates session.json via dispatcher
# ============================================================================
test_session_init() {
    local tmp_dir
    tmp_dir=$(setup_session_env)

    local result
    result=$(run_session_cmd "$tmp_dir" session-init "test-sid" "Test goal")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Verify session.json was created
    [[ -f "$tmp_dir/.aether/data/session.json" ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify session_id in response
    local sid
    sid=$(echo "$result" | jq -r '.result.session_id')
    [[ "$sid" == "test-sid" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: session-read returns session data after init
# ============================================================================
test_session_read() {
    local tmp_dir
    tmp_dir=$(setup_session_env)

    # Initialize a session first
    run_session_cmd "$tmp_dir" session-init "read-test" "Read test goal" > /dev/null

    local result
    result=$(run_session_cmd "$tmp_dir" session-read)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Verify exists field
    local exists
    exists=$(echo "$result" | jq -r '.result.exists')
    [[ "$exists" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify session data has colony_goal
    local goal
    goal=$(echo "$result" | jq -r '.result.session.colony_goal')
    [[ "$goal" == "Read test goal" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: session-is-stale returns valid JSON with is_stale field
# ============================================================================
test_session_is_stale() {
    local tmp_dir
    tmp_dir=$(setup_session_env)

    # Initialize a session first
    run_session_cmd "$tmp_dir" session-init "stale-test" "Stale test" > /dev/null

    local result
    result=$(run_session_cmd "$tmp_dir" session-is-stale)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Verify is_stale field exists and is boolean
    local is_stale
    is_stale=$(echo "$result" | jq -r '.result.is_stale')
    [[ "$is_stale" == "true" || "$is_stale" == "false" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
echo "=== Session Module Smoke Tests ==="
echo ""

run_test test_module_exists "session.sh exists and passes syntax check"
run_test test_session_init "session-init creates session.json with correct session_id"
run_test test_session_read "session-read returns session data after init"
run_test test_session_is_stale "session-is-stale returns valid JSON with is_stale field"

test_summary
