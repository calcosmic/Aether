#!/usr/bin/env bash
# Tests for learning-observations circuit breaker recovery (REL-03)
# Verifies corruption recovery, backup restoration, and retry logic

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment for learning-observe
# ============================================================================
setup_learning_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"

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

    # Create minimal COLONY_STATE.json
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "version": "3.0",
  "goal": "test learning recovery",
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
        bash "$tmpdir/.aether/aether-utils.sh" "$@"
}

# Helper: run aether-utils capturing stderr
run_cmd_with_stderr() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>&1
}

# ============================================================================
# Test 1: Corrupt main file, valid .bak.1 -- recovers from backup with warning
# ============================================================================
test_recovery_from_backup() {
    local tmpdir
    tmpdir=$(setup_learning_env)
    local obs_file="$tmpdir/.aether/data/learning-observations.json"

    # Create a valid backup
    cat > "$obs_file" << 'EOF'
{"observations":[{"content_hash":"sha256:test","content":"test pattern","wisdom_type":"pattern","observation_count":2,"first_seen":"2026-03-01T00:00:00Z","last_seen":"2026-03-02T00:00:00Z","colonies":["testcolony"]}]}
EOF
    cp "$obs_file" "${obs_file}.bak.1"

    # Corrupt the main file
    echo "NOT VALID JSON {{{{" > "$obs_file"

    # Run learning-observe -- should recover from backup
    local output exit_code=0
    output=$(run_cmd_with_stderr "$tmpdir" learning-observe "New test pattern" "pattern" 2>&1) || exit_code=$?

    # Should succeed (exit 0)
    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "Expected exit code 0 after recovery" "Got exit code $exit_code, output: $output"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should contain warning about recovery
    if ! echo "$output" | grep -q "corrupted -- restored from backup"; then
        test_fail "Expected recovery warning in output" "Output: $output"
        rm -rf "$tmpdir"
        return 1
    fi

    # The observations file should now be valid JSON
    if ! jq -e . "$obs_file" >/dev/null 2>&1; then
        test_fail "Expected observations file to be valid JSON after recovery" "File is still corrupt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: Corrupt main file, no backups -- resets to empty with warning
# ============================================================================
test_first_time_recovery() {
    local tmpdir
    tmpdir=$(setup_learning_env)
    local obs_file="$tmpdir/.aether/data/learning-observations.json"

    # Create observations file and corrupt it (no backups)
    echo "NOT VALID JSON {{{{" > "$obs_file"

    # Run learning-observe -- should reset to empty
    local output exit_code=0
    output=$(run_cmd_with_stderr "$tmpdir" learning-observe "Test pattern" "pattern" 2>&1) || exit_code=$?

    # Should succeed
    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "Expected exit code 0 for first-time recovery" "Got exit code $exit_code, output: $output"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should contain first-time recovery warning
    if ! echo "$output" | grep -q "first-time recovery"; then
        test_fail "Expected first-time recovery warning" "Output: $output"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 3: Corrupt main file AND all 3 backups -- hard stop
# ============================================================================
test_all_backups_corrupted() {
    local tmpdir
    tmpdir=$(setup_learning_env)
    local obs_file="$tmpdir/.aether/data/learning-observations.json"

    # Corrupt the main file
    echo "NOT VALID JSON" > "$obs_file"

    # Corrupt all backups
    echo "ALSO BAD JSON 1" > "${obs_file}.bak.1"
    echo "ALSO BAD JSON 2" > "${obs_file}.bak.2"
    echo "ALSO BAD JSON 3" > "${obs_file}.bak.3"

    # Run learning-observe -- should fail with error
    local output exit_code=0
    output=$(run_cmd_with_stderr "$tmpdir" learning-observe "Test pattern" "pattern" 2>&1) || exit_code=$?

    # Should exit non-zero
    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "Expected non-zero exit when all backups corrupt" "Got exit code 0, output: $output"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should mention all backups corrupted
    if ! echo "$output" | grep -q "all 3 backups are corrupted"; then
        test_fail "Expected 'all 3 backups corrupted' message" "Output: $output"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: Valid file -- normal operation (no warnings)
# ============================================================================
test_normal_operation() {
    local tmpdir
    tmpdir=$(setup_learning_env)
    local obs_file="$tmpdir/.aether/data/learning-observations.json"

    # Create valid observations file
    echo '{"observations":[]}' > "$obs_file"

    # Run learning-observe -- should work normally
    local output exit_code=0
    output=$(run_cmd_with_stderr "$tmpdir" learning-observe "Normal test pattern" "pattern" 2>&1) || exit_code=$?

    # Should succeed
    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "Expected exit code 0 for normal operation" "Got exit code $exit_code, output: $output"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should NOT contain any recovery warnings
    if echo "$output" | grep -q "corrupted"; then
        test_fail "Expected no corruption warnings for valid file" "Output contains 'corrupted': $output"
        rm -rf "$tmpdir"
        return 1
    fi

    # Result should be valid JSON with ok=true
    local ok_val
    ok_val=$(echo "$output" | jq -r '.ok' 2>/dev/null)
    if [[ "$ok_val" != "true" ]]; then
        test_fail "Expected ok=true" "Got ok=$ok_val from: $output"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 5: Verify backup rotation happens before writes
# ============================================================================
test_backup_rotation_source() {
    # Verify that the source code contains .bak.N rotation patterns before writes
    local bak_rotation_count
    bak_rotation_count=$(grep -c 'cp -f.*\.bak\.' "$AETHER_UTILS" 2>/dev/null || echo "0")

    # Should have at least 6 cp -f .bak lines (3 per write path x 2 paths)
    if [[ "$bak_rotation_count" -ge 6 ]]; then
        return 0
    else
        test_fail "Expected at least 6 .bak rotation lines in source" "Found $bak_rotation_count"
        return 1
    fi
}

# ============================================================================
# Test 6: Verify retry-once loop is present in recovery code
# ============================================================================
test_retry_loop_present() {
    # Verify the lo_attempt retry loop exists in the source
    if grep -q 'lo_attempt' "$AETHER_UTILS"; then
        return 0
    else
        test_fail "Expected lo_attempt retry loop in aether-utils.sh" "Pattern not found"
        return 1
    fi
}

# ============================================================================
# Run all tests
# ============================================================================

log_info "Running learning-observations circuit breaker tests"
log_info "Repo root: $REPO_ROOT"

run_test test_recovery_from_backup "Corrupt file + valid .bak.1: recovers with warning"
run_test test_first_time_recovery "Corrupt file + no backups: resets to empty"
run_test test_all_backups_corrupted "Corrupt file + corrupt backups: hard stop"
run_test test_normal_operation "Valid file: normal operation without warnings"
run_test test_backup_rotation_source "Backup rotation code present before writes"
run_test test_retry_loop_present "Retry-once loop (lo_attempt) present in recovery"

test_summary
