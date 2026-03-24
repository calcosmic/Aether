#!/usr/bin/env bash
# Tests for state-write subcommand (REL-05)
# Verifies locked, validated, atomic writes to COLONY_STATE.json

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment for state-write
# ============================================================================
setup_state_write_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data/backups"
    mkdir -p "$tmpdir/.aether/temp"
    mkdir -p "$tmpdir/.aether/locks"

    # Copy aether-utils.sh to temp location
    cp "$AETHER_UTILS" "$tmpdir/.aether/aether-utils.sh"
    chmod +x "$tmpdir/.aether/aether-utils.sh"

    # Copy utils directory
    local utils_source="$(dirname "$AETHER_UTILS")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmpdir/.aether/"
    fi

    # Copy exchange directory
    local exchange_source="$(dirname "$AETHER_UTILS")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmpdir/.aether/"
    fi

    # Copy schemas directory if it exists
    local schemas_source="$(dirname "$AETHER_UTILS")/schemas"
    if [[ -d "$schemas_source" ]]; then
        cp -r "$schemas_source" "$tmpdir/.aether/"
    fi

    # Create valid COLONY_STATE.json
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "version": "3.0",
  "goal": "test state write",
  "state": "active",
  "current_phase": 1,
  "phase": {"number": 1, "name": "test"},
  "plan": {"id": "test-plan", "tasks": []},
  "memory": {"instincts": []},
  "errors": {"records": []},
  "events": [],
  "session_id": "colony_testcolony_abc",
  "initialized_at": "2026-02-13T16:00:00Z"
}
EOF

    echo "$tmpdir"
}

# Helper: run aether-utils against a test env
run_cmd() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>&1
}

# Helper: run aether-utils, capturing stderr separately
run_cmd_split() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@"
}

# ============================================================================
# Test 1: Pipe valid JSON to state-write -- verify written and json_ok
# ============================================================================
test_state_write_valid_json() {
    local tmpdir
    tmpdir=$(setup_state_write_env)

    local new_state='{"version":"3.0","goal":"updated","state":"READY","current_phase":2}'

    local result
    result=$(echo "$new_state" | run_cmd "$tmpdir" state-write)

    # Should return json_ok with written:true
    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true in output" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_contains "$result" '"written":true'; then
        test_fail "Expected written:true in result" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify the file was actually updated
    local written_goal
    written_goal=$(jq -r '.goal' "$tmpdir/.aether/data/COLONY_STATE.json" 2>/dev/null)
    if [[ "$written_goal" != "updated" ]]; then
        test_fail "Expected goal to be 'updated'" "$written_goal"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: Pipe invalid JSON to state-write -- verify error (non-zero exit)
# ============================================================================
test_state_write_invalid_json() {
    local tmpdir
    tmpdir=$(setup_state_write_env)

    local result
    local exit_code=0
    result=$(echo "not valid json {{{" | run_cmd "$tmpdir" state-write) || exit_code=$?

    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "Expected non-zero exit code for invalid JSON" "exit $exit_code"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should contain error indication
    if ! assert_contains "$result" "invalid"; then
        # Also accept "E_JSON_INVALID" in output
        if ! assert_contains "$result" "JSON"; then
            test_fail "Expected error message about invalid JSON" "$result"
            rm -rf "$tmpdir"
            return 1
        fi
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 3: Write valid state -- verify a backup was created
# ============================================================================
test_state_write_creates_backup() {
    local tmpdir
    tmpdir=$(setup_state_write_env)

    # Count backups before
    local before_count
    before_count=$(find "$tmpdir/.aether/data/backups" -name "COLONY_STATE.json.*.backup" -type f 2>/dev/null | wc -l | tr -d ' ')

    local new_state='{"version":"3.0","goal":"backup-test","state":"READY","current_phase":3}'
    echo "$new_state" | run_cmd "$tmpdir" state-write >/dev/null 2>&1

    # Count backups after
    local after_count
    after_count=$(find "$tmpdir/.aether/data/backups" -name "COLONY_STATE.json.*.backup" -type f 2>/dev/null | wc -l | tr -d ' ')

    if [[ "$after_count" -le "$before_count" ]]; then
        test_fail "Expected backup to be created" "before=$before_count, after=$after_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: Verify state-write subcommand exists in aether-utils.sh case statement
# ============================================================================
test_state_write_exists_in_case() {
    if ! grep -q 'state-write)' "$AETHER_UTILS"; then
        test_fail "state-write case not found in aether-utils.sh" ""
        return 1
    fi
    return 0
}

# ============================================================================
# Test 5: Verify continue-advance.md references state-write
# ============================================================================
test_continue_advance_uses_state_write() {
    local playbook="$REPO_ROOT/.aether/docs/command-playbooks/continue-advance.md"
    if ! grep -q 'state-write' "$playbook"; then
        test_fail "state-write not referenced in continue-advance.md" ""
        return 1
    fi

    # Also verify no remaining direct write instructions
    local direct_writes
    direct_writes=$(grep -c 'Write COLONY_STATE' "$playbook" 2>/dev/null | tr -d '[:space:]' || echo "0")
    if [[ "$direct_writes" -gt 0 ]]; then
        test_fail "Found direct 'Write COLONY_STATE' instructions in continue-advance.md" "count=$direct_writes"
        return 1
    fi

    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running state-write tests"
log_info "Repo root: $REPO_ROOT"

run_test test_state_write_valid_json "Pipe valid JSON to state-write succeeds"
run_test test_state_write_invalid_json "Pipe invalid JSON to state-write fails"
run_test test_state_write_creates_backup "state-write creates backup before writing"
run_test test_state_write_exists_in_case "state-write subcommand exists in case statement"
run_test test_continue_advance_uses_state_write "continue-advance.md references state-write"

test_summary
