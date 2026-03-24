#!/usr/bin/env bash
# Queen Module Smoke Tests
# Tests queen.sh extracted module functions via aether-utils.sh subcommands

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
# Helper: Create isolated test environment with queen support
# ============================================================================
setup_queen_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils" "$tmp_dir/.aether/templates"

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

    # Copy templates if available (needed for queen-init)
    local templates_source="$(dirname "$AETHER_UTILS_SOURCE")/templates"
    if [[ -d "$templates_source" ]]; then
        cp -r "$templates_source" "$tmp_dir/.aether/"
    fi

    # Write a minimal COLONY_STATE.json
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'CSEOF'
{
  "version": "3.0",
  "goal": "Test queen module",
  "state": "READY",
  "current_phase": 1,
  "milestone": "First Mound",
  "session_id": "test-queen",
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

run_queen_cmd() {
    local tmp_dir="$1"
    shift
    HOME="$tmp_dir" AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

# ============================================================================
# Test: queen.sh module file exists and has valid syntax
# ============================================================================
test_module_exists() {
    local module_path="$PROJECT_ROOT/.aether/utils/queen.sh"

    assert_file_exists "$module_path" || return 1
    bash -n "$module_path" 2>/dev/null || return 1
}

# ============================================================================
# Test: queen-init creates QUEEN.md from template
# ============================================================================
test_queen_init() {
    local tmp_dir
    tmp_dir=$(setup_queen_env)

    # Ensure no existing QUEEN.md
    rm -f "$tmp_dir/.aether/QUEEN.md"

    local result
    result=$(run_queen_cmd "$tmp_dir" queen-init)

    # Check if we got a valid JSON response
    if echo "$result" | jq -e '.ok' >/dev/null 2>&1; then
        local ok_val
        ok_val=$(echo "$result" | jq -r '.ok')
        if [[ "$ok_val" == "true" ]]; then
            # Verify QUEEN.md was created
            if [[ -f "$tmp_dir/.aether/QUEEN.md" ]]; then
                rm -rf "$tmp_dir"
                return 0
            fi
        fi
    fi

    rm -rf "$tmp_dir"
    return 1
}

# ============================================================================
# Test: queen-thresholds returns JSON with threshold values
# ============================================================================
test_queen_thresholds() {
    local tmp_dir
    tmp_dir=$(setup_queen_env)

    local result
    result=$(run_queen_cmd "$tmp_dir" queen-thresholds)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Verify result contains threshold data (JSON structure with .result)
    local has_data
    has_data=$(echo "$result" | jq -r '.result | keys | length')
    [[ "$has_data" -gt 0 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
echo "=== Queen Module Smoke Tests ==="
echo ""

run_test test_module_exists "queen.sh exists and passes syntax check"
run_test test_queen_init "queen-init creates QUEEN.md via dispatcher"
run_test test_queen_thresholds "queen-thresholds returns JSON with threshold values"

test_summary
