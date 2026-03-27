#!/usr/bin/env bash
# Suggest Module Smoke Tests
# Tests suggest.sh extracted module functions via aether-utils.sh subcommands

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
# Helper: Create isolated test environment with suggest support
# ============================================================================
setup_suggest_env() {
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
  "goal": "Test suggest module",
  "state": "READY",
  "current_phase": 1,
  "milestone": "First Mound",
  "session_id": "test-suggest",
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

run_suggest_cmd() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

# ============================================================================
# Test: suggest.sh module file exists and has valid syntax
# ============================================================================
test_module_exists() {
    local module_path="$PROJECT_ROOT/.aether/utils/suggest.sh"

    assert_file_exists "$module_path" || return 1
    bash -n "$module_path" 2>/dev/null || return 1
}

# ============================================================================
# Test: suggest-check returns JSON response via dispatcher
# ============================================================================
test_suggest_check() {
    local tmp_dir
    tmp_dir=$(setup_suggest_env)

    local result
    result=$(run_suggest_cmd "$tmp_dir" suggest-check "test-hash-abc123")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Verify already_suggested field exists and is false (nothing recorded yet)
    local already
    already=$(echo "$result" | jq -r '.result.already_suggested')
    [[ "$already" == "false" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: suggest-record stores suggestion data and suggest-check finds it
# ============================================================================
test_suggest_record() {
    local tmp_dir
    tmp_dir=$(setup_suggest_env)

    # Create a session.json first so suggest-record can append to it
    cat > "$tmp_dir/.aether/data/session.json" << 'SJEOF'
{
  "colony_goal": "Test suggest module",
  "suggested_pheromones": []
}
SJEOF

    # Record a suggestion
    local result
    result=$(run_suggest_cmd "$tmp_dir" suggest-record "test-hash-xyz789" "FOCUS")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Verify recorded field
    local recorded
    recorded=$(echo "$result" | jq -r '.result.recorded')
    [[ "$recorded" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }

    # Now verify suggest-check finds it
    local check_result
    check_result=$(run_suggest_cmd "$tmp_dir" suggest-check "test-hash-xyz789")
    local already
    already=$(echo "$check_result" | jq -r '.result.already_suggested')
    [[ "$already" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
echo "=== Suggest Module Smoke Tests ==="
echo ""

run_test test_module_exists "suggest.sh exists and passes syntax check"
run_test test_suggest_check "suggest-check returns JSON with already_suggested field"
run_test test_suggest_record "suggest-record stores data and suggest-check finds it"

test_summary
