#!/usr/bin/env bash
# Tests for state-checkpoint subcommand (REL-04)
# Verifies rolling backup creation and max 3 retention

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment for state-checkpoint
# ============================================================================
setup_checkpoint_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data/backups"
    mkdir -p "$tmpdir/.aether/temp"

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
  "goal": "test state checkpoint",
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

# ============================================================================
# Test 1: state-checkpoint creates a backup file
# ============================================================================
test_checkpoint_creates_backup() {
    local tmpdir
    tmpdir=$(setup_checkpoint_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" state-checkpoint "test-reason") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "Expected exit code 0" "Got exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check ok=true
    local ok_val
    ok_val=$(echo "$result" | jq -r '.ok' 2>/dev/null)
    if [[ "$ok_val" != "true" ]]; then
        test_fail "Expected ok=true" "Got ok=$ok_val from: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check that a backup file exists in backups dir
    local backup_count
    backup_count=$(find "$tmpdir/.aether/data/backups" -name "COLONY_STATE.json.*.backup" -type f 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$backup_count" -lt 1 ]]; then
        test_fail "Expected at least 1 backup file" "Found $backup_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: state-checkpoint retains at most 3 backups
# ============================================================================
test_checkpoint_max_3_retention() {
    local tmpdir
    tmpdir=$(setup_checkpoint_env)

    # Run state-checkpoint 5 times (with slight delay for unique timestamps)
    for i in 1 2 3 4 5; do
        run_cmd "$tmpdir" state-checkpoint "run-$i" >/dev/null
        sleep 1  # Ensure different timestamps
    done

    # Count backup files -- should be at most 3
    local backup_count
    backup_count=$(find "$tmpdir/.aether/data/backups" -name "COLONY_STATE.json.*.backup" -type f 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$backup_count" -gt 3 ]]; then
        test_fail "Expected at most 3 backup files" "Found $backup_count"
        rm -rf "$tmpdir"
        return 1
    fi

    if [[ "$backup_count" -lt 1 ]]; then
        test_fail "Expected at least 1 backup file" "Found $backup_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 3: state-checkpoint refuses corrupt COLONY_STATE.json
# ============================================================================
test_checkpoint_refuses_corrupt() {
    local tmpdir
    tmpdir=$(setup_checkpoint_env)

    # Corrupt COLONY_STATE.json
    echo "NOT VALID JSON {{{" > "$tmpdir/.aether/data/COLONY_STATE.json"

    local result exit_code=0
    result=$(run_cmd "$tmpdir" state-checkpoint "corrupt-test") || exit_code=$?

    # Should exit non-zero
    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "Expected non-zero exit for corrupt state" "Got exit code 0, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should mention corrupt/invalid
    if ! echo "$result" | grep -qi "corrupt\|invalid"; then
        test_fail "Expected corruption error message" "Output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: state-checkpoint registered in help
# ============================================================================
test_checkpoint_in_source() {
    # Verify state-checkpoint case exists in aether-utils.sh
    if grep -q 'state-checkpoint)' "$AETHER_UTILS"; then
        return 0
    else
        test_fail "Expected state-checkpoint case in aether-utils.sh" "Not found"
        return 1
    fi
}

# ============================================================================
# Test 5: state-checkpoint records the correct reason for all 3 checkpoint sites
# ============================================================================
test_checkpoint_reason_recorded() {
    local tmpdir
    tmpdir=$(setup_checkpoint_env)

    local reasons=("pre-build-wave" "pre-continue-advance" "pre-seal")
    local all_pass=true

    for reason in "${reasons[@]}"; do
        local result exit_code=0
        result=$(run_cmd "$tmpdir" state-checkpoint "$reason") || exit_code=$?

        if [[ "$exit_code" -ne 0 ]]; then
            test_fail "Expected exit code 0 for reason '$reason'" "Got exit code $exit_code, output: $result"
            all_pass=false
            continue
        fi

        local recorded_reason
        recorded_reason=$(echo "$result" | jq -r '.result.reason' 2>/dev/null)

        if [[ "$recorded_reason" != "$reason" ]]; then
            test_fail "Expected reason '$reason'" "Got '$recorded_reason' from: $result"
            all_pass=false
        fi
    done

    rm -rf "$tmpdir"

    if [[ "$all_pass" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

# ============================================================================
# Run all tests
# ============================================================================

log_info "Running state-checkpoint subcommand tests"
log_info "Repo root: $REPO_ROOT"

run_test test_checkpoint_creates_backup "state-checkpoint creates backup file"
run_test test_checkpoint_max_3_retention "state-checkpoint retains at most 3 backups"
run_test test_checkpoint_refuses_corrupt "state-checkpoint refuses corrupt COLONY_STATE.json"
run_test test_checkpoint_in_source "state-checkpoint registered in aether-utils.sh"
run_test test_checkpoint_reason_recorded "state-checkpoint records correct reason for all 3 checkpoint sites"

test_summary
