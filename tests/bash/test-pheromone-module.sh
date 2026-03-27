#!/usr/bin/env bash
# Pheromone Module Smoke Tests
# Tests pheromone.sh extracted module functions via aether-utils.sh subcommands

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
# Helper: Create isolated test environment with pheromone support
# ============================================================================
setup_pheromone_env() {
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
  "goal": "Test pheromone module",
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
# Test 1: Module file exists and passes syntax check
# ============================================================================
test_module_exists() {
    test_start "pheromone.sh module exists and has valid syntax"

    local module_file="$PROJECT_ROOT/.aether/utils/pheromone.sh"

    if [[ ! -f "$module_file" ]]; then
        test_fail "Module file exists" "File not found: $module_file"
        return
    fi

    if ! bash -n "$module_file" 2>/dev/null; then
        test_fail "Syntax check passes" "bash -n failed"
        return
    fi

    # Verify key functions are defined
    local func_count
    func_count=$(grep -c '() {' "$module_file" || echo "0")
    if [[ "$func_count" -lt 13 ]]; then
        test_fail "At least 13 functions defined" "Found $func_count"
        return
    fi

    test_pass
}

# ============================================================================
# Test 2: pheromone-count returns valid JSON via dispatcher
# ============================================================================
test_pheromone_count() {
    test_start "pheromone-count returns valid JSON with count fields"

    local tmp_dir
    tmp_dir=$(setup_pheromone_env)

    local result
    result=$(AETHER_ROOT="$tmp_dir" bash "$tmp_dir/.aether/aether-utils.sh" pheromone-count 2>/dev/null) || {
        rm -rf "$tmp_dir"
        test_fail "pheromone-count succeeds" "Command failed"
        return
    }

    if ! echo "$result" | jq empty 2>/dev/null; then
        rm -rf "$tmp_dir"
        test_fail "Valid JSON" "Invalid JSON: $result"
        return
    fi

    # Check for .result with count fields
    local total
    total=$(echo "$result" | jq -r '.result.total // .total // "missing"' 2>/dev/null)
    if [[ "$total" == "missing" ]]; then
        rm -rf "$tmp_dir"
        test_fail "Has total field" "Missing total field in: $result"
        return
    fi

    rm -rf "$tmp_dir"
    test_pass
}

# ============================================================================
# Test 3: pheromone-write creates a signal
# ============================================================================
test_pheromone_write() {
    test_start "pheromone-write creates a signal and returns json_ok"

    local tmp_dir
    tmp_dir=$(setup_pheromone_env)

    local result
    result=$(AETHER_ROOT="$tmp_dir" bash "$tmp_dir/.aether/aether-utils.sh" pheromone-write FOCUS "test signal content" --strength 0.8 --source "test" --reason "smoke test" 2>/dev/null) || {
        rm -rf "$tmp_dir"
        test_fail "pheromone-write succeeds" "Command failed"
        return
    }

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

    # Verify signal was written to pheromones.json
    local sig_count
    sig_count=$(jq '[.signals[] | select(.active == true)] | length' "$tmp_dir/.aether/data/pheromones.json" 2>/dev/null || echo "0")
    if [[ "$sig_count" -lt 1 ]]; then
        rm -rf "$tmp_dir"
        test_fail "Signal written to file" "Active signal count: $sig_count"
        return
    fi

    rm -rf "$tmp_dir"
    test_pass
}

# ============================================================================
# Test 4: pheromone-read returns signals after writing
# ============================================================================
test_pheromone_read() {
    test_start "pheromone-read returns signals after pheromone-write"

    local tmp_dir
    tmp_dir=$(setup_pheromone_env)

    # Write a signal first
    AETHER_ROOT="$tmp_dir" bash "$tmp_dir/.aether/aether-utils.sh" pheromone-write REDIRECT "avoid test pattern" --strength 0.9 --source "test" --reason "smoke test" 2>/dev/null || {
        rm -rf "$tmp_dir"
        test_fail "pheromone-write succeeds" "Write command failed"
        return
    }

    # Read signals
    local result
    result=$(AETHER_ROOT="$tmp_dir" bash "$tmp_dir/.aether/aether-utils.sh" pheromone-read 2>/dev/null) || {
        rm -rf "$tmp_dir"
        test_fail "pheromone-read succeeds" "Read command failed"
        return
    }

    if ! echo "$result" | jq empty 2>/dev/null; then
        rm -rf "$tmp_dir"
        test_fail "Valid JSON" "Invalid JSON: $result"
        return
    fi

    # Check that signals are present in the result
    local sig_count
    sig_count=$(echo "$result" | jq '.result.signals | length' 2>/dev/null || echo "0")
    if [[ "$sig_count" -lt 1 ]]; then
        rm -rf "$tmp_dir"
        test_fail "At least 1 signal returned" "Signal count: $sig_count"
        return
    fi

    rm -rf "$tmp_dir"
    test_pass
}

# ============================================================================
# Test 5: eternal-init creates eternal memory directory
# ============================================================================
test_eternal_init() {
    test_start "eternal-init creates eternal memory directory with mocked HOME"

    local tmp_dir
    tmp_dir=$(setup_pheromone_env)

    # Use a temp HOME to avoid modifying real hub
    local fake_home
    fake_home=$(mktemp -d)

    local result
    result=$(HOME="$fake_home" AETHER_ROOT="$tmp_dir" bash "$tmp_dir/.aether/aether-utils.sh" eternal-init 2>/dev/null) || {
        rm -rf "$tmp_dir" "$fake_home"
        test_fail "eternal-init succeeds" "Command failed"
        return
    }

    if ! echo "$result" | jq empty 2>/dev/null; then
        rm -rf "$tmp_dir" "$fake_home"
        test_fail "Valid JSON" "Invalid JSON: $result"
        return
    fi

    # Check directory was created
    if [[ ! -d "$fake_home/.aether/eternal" ]]; then
        rm -rf "$tmp_dir" "$fake_home"
        test_fail "Eternal directory created" "Directory not found: $fake_home/.aether/eternal"
        return
    fi

    # Check memory.json was created
    if [[ ! -f "$fake_home/.aether/eternal/memory.json" ]]; then
        rm -rf "$tmp_dir" "$fake_home"
        test_fail "memory.json created" "File not found"
        return
    fi

    rm -rf "$tmp_dir" "$fake_home"
    test_pass
}

# ============================================================================
# Run all tests
# ============================================================================
log_info "Running Pheromone Module Smoke Tests"
log_info "============================================"

test_module_exists
test_pheromone_count
test_pheromone_write
test_pheromone_read
test_eternal_init

log_info "============================================"
test_summary
