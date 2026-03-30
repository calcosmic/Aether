#!/usr/bin/env bash
# Flag Module Smoke Tests
# Tests flag.sh extracted module functions via aether-utils.sh subcommands

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
# Helper: Create isolated test environment with flag support
# ============================================================================
setup_flag_env() {
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
  "goal": "Test flag module",
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
  "graveyards": []
}
CSEOF

    echo "$tmp_dir"
}

run_flag_cmd() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

# ============================================================================
# Test: flag.sh module file exists and has valid syntax
# ============================================================================
test_module_exists() {
    local module_path="$PROJECT_ROOT/.aether/utils/flag.sh"

    assert_file_exists "$module_path" || return 1
    bash -n "$module_path" 2>/dev/null || return 1
}

# ============================================================================
# Test: flag-add creates a flag via the dispatcher
# ============================================================================
test_flag_add() {
    local tmp_dir
    tmp_dir=$(setup_flag_env)

    local result
    result=$(run_flag_cmd "$tmp_dir" flag-add blocker "Test blocker" "Testing flag-add")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.type" "blocker" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.severity" "critical" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: flag-list returns flags after adding one
# ============================================================================
test_flag_list() {
    local tmp_dir
    tmp_dir=$(setup_flag_env)

    # Add a flag first
    run_flag_cmd "$tmp_dir" flag-add issue "List test" "Testing flag-list" > /dev/null

    local result
    result=$(run_flag_cmd "$tmp_dir" flag-list)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local count
    count=$(echo "$result" | jq -r '.result.count')
    [[ "$count" -ge 1 ]] || { rm -rf "$tmp_dir"; return 1; }

    assert_json_field_equals "$result" ".result.flags[0].title" "List test" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: flag-check-blockers detects blocker flags
# ============================================================================
test_flag_check_blockers() {
    local tmp_dir
    tmp_dir=$(setup_flag_env)

    # Add a blocker flag
    run_flag_cmd "$tmp_dir" flag-add blocker "Blocking issue" "Testing check-blockers" > /dev/null

    local result
    result=$(run_flag_cmd "$tmp_dir" flag-check-blockers)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local blockers
    blockers=$(echo "$result" | jq -r '.result.blockers')
    [[ "$blockers" -ge 1 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: flag-list --type rejects jq metacharacters (no filter injection)
# ============================================================================
test_flag_list_type_no_injection() {
    local tmp_dir
    tmp_dir=$(setup_flag_env)

    # Add a single blocker flag
    run_flag_cmd "$tmp_dir" flag-add blocker "Safe flag" "Testing injection guard" > /dev/null

    # Inject jq via --type using comma-operator payload.
    # If filter_type is interpolated unsafely into the jq filter string,
    # the payload 'blocker", .version, "' breaks out of the string literal
    # and turns select(.type == "blocker", .version, "") into a comma
    # expression where .version (truthy "1") lets every flag through twice,
    # producing count=2 instead of count=1.
    local result
    result=$(run_flag_cmd "$tmp_dir" flag-list --type 'blocker", .version, "')

    # With the fix, the type string is passed via --arg and no injection
    # occurs: the literal string won't match any .type field, so count must be 0.
    # Before the fix, injection duplicates the flag yielding count=2.
    local count
    count=$(echo "$result" | jq -r '.result.count // 0' 2>/dev/null)
    [[ "$count" -eq 0 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: flag-list --type with valid type works correctly
# ============================================================================
test_flag_list_type_filter() {
    local tmp_dir
    tmp_dir=$(setup_flag_env)

    # Add a blocker and a note
    run_flag_cmd "$tmp_dir" flag-add blocker "Blocker A" "desc" > /dev/null
    run_flag_cmd "$tmp_dir" flag-add note "Note B" "desc" > /dev/null

    # Filter for blocker only
    local result
    result=$(run_flag_cmd "$tmp_dir" flag-list --type blocker)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local count
    count=$(echo "$result" | jq -r '.result.count')
    [[ "$count" -eq 1 ]] || { rm -rf "$tmp_dir"; return 1; }

    assert_json_field_equals "$result" ".result.flags[0].type" "blocker" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
echo "=== Flag Module Smoke Tests ==="
echo ""

run_test test_module_exists "flag.sh exists and passes syntax check"
run_test test_flag_add "flag-add creates flag with correct type and severity"
run_test test_flag_list "flag-list returns added flags"
run_test test_flag_check_blockers "flag-check-blockers detects blocker flags"
run_test test_flag_list_type_no_injection "flag-list --type rejects jq metacharacter injection"
run_test test_flag_list_type_filter "flag-list --type filters correctly by type"

test_summary
