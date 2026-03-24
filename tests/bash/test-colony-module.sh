#!/usr/bin/env bash
# Colony Module Smoke Tests
# Tests colony-archive-xml extracted into chamber-utils.sh via aether-utils.sh dispatcher

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
# Helper: Create isolated test environment with colony support
# ============================================================================
setup_colony_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils" "$tmp_dir/.aether/exchange"

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
  "goal": "Test colony module",
  "state": "READY",
  "current_phase": 1,
  "plan": {"id": "test-plan", "tasks": []},
  "memory": {
    "events": [],
    "instincts": [],
    "phase_learnings": []
  },
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z"
}
CSEOF

    # Write empty pheromones.json
    cat > "$tmp_dir/.aether/data/pheromones.json" << 'PHEOF'
{
  "version": "1.0.0",
  "colony_id": "test-colony",
  "generated_at": "2026-01-01T00:00:00Z",
  "signals": []
}
PHEOF

    echo "$tmp_dir"
}

# ============================================================================
# Test 1: chamber-utils.sh module exists and passes syntax check
# ============================================================================
test_module_exists() {
    test_start "chamber-utils.sh module exists and has valid syntax"

    local module_file="$PROJECT_ROOT/.aether/utils/chamber-utils.sh"

    if [[ ! -f "$module_file" ]]; then
        test_fail "Module file exists" "File not found: $module_file"
        return
    fi

    if ! bash -n "$module_file" 2>/dev/null; then
        test_fail "Syntax check passes" "bash -n failed"
        return
    fi

    # Verify _colony_archive_xml function is defined
    if ! grep -q '_colony_archive_xml()' "$module_file"; then
        test_fail "_colony_archive_xml function defined" "Function not found"
        return
    fi

    test_pass
}

# ============================================================================
# Test 2: colony-archive-xml dispatch works via aether-utils.sh
# ============================================================================
test_colony_archive_xml() {
    test_start "colony-archive-xml dispatches through aether-utils.sh"

    local tmp_dir
    tmp_dir=$(setup_colony_env)

    local output_path="$tmp_dir/test-archive.xml"

    # colony-archive-xml requires xmllint -- check availability
    if ! command -v xmllint >/dev/null 2>&1; then
        # Without xmllint, colony-archive-xml should return an error about xmllint
        local result
        result=$(AETHER_ROOT="$tmp_dir" bash "$tmp_dir/.aether/aether-utils.sh" colony-archive-xml "$output_path" 2>&1) || true

        if echo "$result" | grep -q "xmllint"; then
            rm -rf "$tmp_dir"
            test_pass
            return
        fi

        rm -rf "$tmp_dir"
        test_fail "Error mentions xmllint" "Unexpected output: $result"
        return
    fi

    # xmllint is available -- run full command
    # colony-archive-xml sources exchange scripts that may produce their own stdout;
    # the final JSON line from colony-archive-xml is the one we want
    local raw_result
    raw_result=$(AETHER_ROOT="$tmp_dir" bash "$tmp_dir/.aether/aether-utils.sh" colony-archive-xml "$output_path" 2>/dev/null) || {
        rm -rf "$tmp_dir"
        test_fail "colony-archive-xml succeeds" "Command failed"
        return
    }

    # Take only the last line (the actual colony-archive-xml JSON response)
    local result
    result=$(echo "$raw_result" | tail -1)

    if ! echo "$result" | jq empty 2>/dev/null; then
        rm -rf "$tmp_dir"
        test_fail "Valid JSON" "Invalid JSON: $result"
        return
    fi

    local ok_val
    ok_val=$(echo "$result" | jq -r '.ok' 2>/dev/null)
    if [[ "$ok_val" != "true" ]]; then
        rm -rf "$tmp_dir"
        test_fail "ok is true" "ok is $ok_val in: $result"
        return
    fi

    # Verify output file was created
    if [[ ! -f "$output_path" ]]; then
        rm -rf "$tmp_dir"
        test_fail "Archive XML file created" "File not found: $output_path"
        return
    fi

    rm -rf "$tmp_dir"
    test_pass
}

# ============================================================================
# Run all tests
# ============================================================================
log_info "Running Colony Module Smoke Tests"
log_info "============================================"

test_module_exists
test_colony_archive_xml

log_info "============================================"
test_summary
