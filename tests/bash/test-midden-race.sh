#!/usr/bin/env bash
# Tests for midden PID-scoped temp files (REL-02)
# Verifies that midden-write uses PID-scoped temp files on both locked and lockless paths

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"
MIDDEN_SH="$REPO_ROOT/.aether/utils/midden.sh"

# ============================================================================
# Helper: Create isolated test environment with midden support
# ============================================================================
setup_midden_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data/midden"

    # Copy aether-utils.sh to temp location so it uses temp data dir
    cp "$AETHER_UTILS" "$tmpdir/.aether/aether-utils.sh"
    chmod +x "$tmpdir/.aether/aether-utils.sh"

    # Copy utils directory (needed for acquire_lock, atomic_write, etc.)
    local utils_source="$(dirname "$AETHER_UTILS")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmpdir/.aether/"
    fi

    # Copy exchange directory (needed for XML functions sourced by utils)
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
  "goal": "test midden race",
  "state": "active",
  "current_phase": 1,
  "plan": {"id": "test-plan", "tasks": []},
  "memory": {"instincts": []},
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session",
  "initialized_at": "2026-02-13T16:00:00Z"
}
EOF

    # Initialize empty midden.json
    cat > "$tmpdir/.aether/data/midden/midden.json" << 'EOF'
{"version":"1.0.0","entries":[],"entry_count":0}
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
# Test 1: Locked write path uses PID-scoped temp file
# ============================================================================
test_locked_path_pid_scoped() {
    # Verify that the locked write path in midden.sh uses .tmp.$$ pattern
    if grep -q '\.tmp\.\$\$' "$MIDDEN_SH"; then
        # Found PID-scoped temp file pattern
        local locked_count
        locked_count=$(grep -c '\.tmp\.\$\$' "$MIDDEN_SH")
        if [[ "$locked_count" -ge 2 ]]; then
            return 0
        else
            test_fail "Expected .tmp.\$\$ on both locked and lockless paths" "Found $locked_count occurrences"
            return 1
        fi
    else
        test_fail "Expected .tmp.\$\$ pattern in midden.sh" "Pattern not found"
        return 1
    fi
}

# ============================================================================
# Test 2: Lockless fallback path also uses PID-scoped temp file
# ============================================================================
test_lockless_path_pid_scoped() {
    # The lockless path should have the same .tmp.$$ pattern
    # Count occurrences of the mw_tmp variable assignment with PID scope
    local pid_assigns
    pid_assigns=$(grep -c 'mw_tmp=.*\.tmp\.\$\$' "$MIDDEN_SH" 2>/dev/null || echo "0")
    if [[ "$pid_assigns" -ge 2 ]]; then
        return 0
    else
        test_fail "Expected mw_tmp=*.tmp.\$\$ on both write paths" "Found $pid_assigns assignments"
        return 1
    fi
}

# ============================================================================
# Test 3: Sequential writes both persist
# ============================================================================
test_sequential_writes_persist() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    # Write two entries sequentially
    local result1 result2
    result1=$(run_cmd "$tmpdir" midden-write "security" "First entry" "test")
    result2=$(run_cmd "$tmpdir" midden-write "quality" "Second entry" "test")

    # Verify both entries exist in midden.json
    local entry_count
    entry_count=$(jq '.entries | length' "$tmpdir/.aether/data/midden/midden.json" 2>/dev/null)
    if [[ "$entry_count" != "2" ]]; then
        test_fail "Expected 2 entries after sequential writes" "Got $entry_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify both categories are present
    local has_security has_quality
    has_security=$(jq '[.entries[] | select(.category == "security")] | length' "$tmpdir/.aether/data/midden/midden.json")
    has_quality=$(jq '[.entries[] | select(.category == "quality")] | length' "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$has_security" != "1" || "$has_quality" != "1" ]]; then
        test_fail "Expected 1 security and 1 quality entry" "security=$has_security quality=$has_quality"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: Lockless fallback emits warning
# ============================================================================
test_lockless_warning_emitted() {
    # Verify the lockless fallback path contains the warning message
    if grep -q 'Midden write completed without lock' "$MIDDEN_SH"; then
        return 0
    else
        test_fail "Expected lockless warning message in midden.sh" "Warning not found"
        return 1
    fi
}

# ============================================================================
# Test 5: Retry-once pattern present on both paths
# ============================================================================
test_retry_once_pattern() {
    # Verify the retry-once pattern is present (Silent retry comment)
    local retry_count
    retry_count=$(grep -c 'Silent retry (once)' "$MIDDEN_SH" 2>/dev/null || echo "0")
    if [[ "$retry_count" -ge 2 ]]; then
        return 0
    else
        test_fail "Expected retry-once pattern on both write paths" "Found $retry_count occurrences"
        return 1
    fi
}

# ============================================================================
# Run all tests
# ============================================================================

log_info "Running midden PID-scoped temp file tests"
log_info "Repo root: $REPO_ROOT"

run_test test_locked_path_pid_scoped "Locked write path uses PID-scoped temp file (.tmp.\$\$)"
run_test test_lockless_path_pid_scoped "Lockless fallback path uses PID-scoped temp file"
run_test test_sequential_writes_persist "Sequential writes both persist in midden.json"
run_test test_lockless_warning_emitted "Lockless fallback emits warning to stderr"
run_test test_retry_once_pattern "Retry-once pattern present on both write paths"

test_summary
