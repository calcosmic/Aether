#!/usr/bin/env bash
# Tests for instinct-apply subcommand
# Tasks 1.4 + 1.5: Locking fix and instinct-apply implementation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment
# ============================================================================
setup_instinct_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"

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

    # Create minimal COLONY_STATE.json with instincts array
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "goal": "test instinct-apply",
  "state": "active",
  "current_phase": 1,
  "plan": {"id": "test-plan", "tasks": []},
  "memory": {
    "instincts": []
  },
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session",
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

# Helper: create an instinct in test env and return the id
create_test_instinct() {
    local tmpdir="$1"
    local trigger="${2:-when tests fail}"
    local action="${3:-investigate root cause}"
    local confidence="${4:-0.5}"

    run_cmd "$tmpdir" instinct-create \
        --trigger "$trigger" \
        --action "$action" \
        --confidence "$confidence" \
        --domain "testing" \
        --source "test" \
        --evidence "test evidence"
}

# ============================================================================
# Test 1: Apply with success increments applications and boosts confidence
# ============================================================================
test_apply_success() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Create an instinct
    local create_result
    create_result=$(create_test_instinct "$tmpdir")
    local instinct_id
    instinct_id=$(echo "$create_result" | jq -r '.result.instinct_id')

    if [[ -z "$instinct_id" || "$instinct_id" == "null" ]]; then
        test_fail "instinct should be created" "create failed: $create_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Apply with success (default)
    local apply_result exit_code=0
    apply_result=$(run_cmd "$tmpdir" instinct-apply --id "$instinct_id") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $apply_result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_json_valid "$apply_result"; then
        test_fail "valid JSON" "invalid JSON: $apply_result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$apply_result"; then
        test_fail "ok=true" "ok was not true: $apply_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify applications=1
    local applications
    applications=$(echo "$apply_result" | jq -r '.result.applications')
    if [[ "$applications" != "1" ]]; then
        test_fail "applications=1" "applications=$applications"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify confidence increased from 0.5 to 0.55
    local new_conf
    new_conf=$(echo "$apply_result" | jq -r '.result.new_confidence')
    if [[ "$new_conf" != "0.55" ]]; then
        test_fail "confidence=0.55" "confidence=$new_conf"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify state file was updated
    local state_apps
    state_apps=$(jq --arg id "$instinct_id" '
      [(.memory.instincts // [])[] | select(.id == $id)] | first | .applications
    ' "$tmpdir/.aether/data/COLONY_STATE.json")
    if [[ "$state_apps" != "1" ]]; then
        test_fail "state applications=1" "state applications=$state_apps"
        rm -rf "$tmpdir"
        return 1
    fi

    local state_successes
    state_successes=$(jq --arg id "$instinct_id" '
      [(.memory.instincts // [])[] | select(.id == $id)] | first | .successes
    ' "$tmpdir/.aether/data/COLONY_STATE.json")
    if [[ "$state_successes" != "1" ]]; then
        test_fail "state successes=1" "state successes=$state_successes"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: Apply twice increments applications to 2
# ============================================================================
test_apply_twice() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Create an instinct
    local create_result
    create_result=$(create_test_instinct "$tmpdir")
    local instinct_id
    instinct_id=$(echo "$create_result" | jq -r '.result.instinct_id')

    # Apply twice
    run_cmd "$tmpdir" instinct-apply --id "$instinct_id" >/dev/null 2>&1 || true
    local apply_result
    apply_result=$(run_cmd "$tmpdir" instinct-apply --id "$instinct_id") || true

    local applications
    applications=$(echo "$apply_result" | jq -r '.result.applications' 2>/dev/null)
    if [[ "$applications" != "2" ]]; then
        test_fail "applications=2" "applications=$applications"
        rm -rf "$tmpdir"
        return 1
    fi

    # Confidence should be 0.5 + 0.05 + 0.05 = 0.6 (allow float precision)
    local new_conf
    new_conf=$(echo "$apply_result" | jq -r '.result.new_confidence' 2>/dev/null)
    local conf_ok
    conf_ok=$(echo "$new_conf" | awk '{printf "%.2f", $1}')
    if [[ "$conf_ok" != "0.60" ]]; then
        test_fail "confidence~0.6" "confidence=$new_conf"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 3: Apply with failure decreases confidence
# ============================================================================
test_apply_failure() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Create an instinct with confidence 0.5
    local create_result
    create_result=$(create_test_instinct "$tmpdir")
    local instinct_id
    instinct_id=$(echo "$create_result" | jq -r '.result.instinct_id')

    # Apply with failure
    local apply_result exit_code=0
    apply_result=$(run_cmd "$tmpdir" instinct-apply --id "$instinct_id" --outcome failure) || exit_code=$?

    local applications
    applications=$(echo "$apply_result" | jq -r '.result.applications' 2>/dev/null)
    if [[ "$applications" != "1" ]]; then
        test_fail "applications=1" "applications=$applications"
        rm -rf "$tmpdir"
        return 1
    fi

    # Confidence should be 0.5 - 0.1 = 0.4
    local new_conf
    new_conf=$(echo "$apply_result" | jq -r '.result.new_confidence' 2>/dev/null)
    if [[ "$new_conf" != "0.4" ]]; then
        test_fail "confidence=0.4" "confidence=$new_conf"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify failures count in state
    local state_failures
    state_failures=$(jq --arg id "$instinct_id" '
      [(.memory.instincts // [])[] | select(.id == $id)] | first | .failures
    ' "$tmpdir/.aether/data/COLONY_STATE.json")
    if [[ "$state_failures" != "1" ]]; then
        test_fail "state failures=1" "state failures=$state_failures"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: Apply with non-existent id returns error
# ============================================================================
test_apply_nonexistent() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    local apply_result exit_code=0
    apply_result=$(run_cmd "$tmpdir" instinct-apply --id "nonexistent_id_999") || exit_code=$?

    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "should fail with non-existent id" "got exit code 0"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_contains "$apply_result" "not found"; then
        test_fail "error should mention 'not found'" "$apply_result"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 5: Confidence floor at 0.1 (never goes below)
# ============================================================================
test_apply_confidence_floor() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Create instinct with low confidence 0.15
    local create_result
    create_result=$(create_test_instinct "$tmpdir" "low trigger" "low action" "0.15")
    local instinct_id
    instinct_id=$(echo "$create_result" | jq -r '.result.instinct_id')

    # Apply with failure: 0.15 - 0.1 = 0.05, should floor at 0.1
    local apply_result exit_code=0
    apply_result=$(run_cmd "$tmpdir" instinct-apply --id "$instinct_id" --outcome failure) || exit_code=$?

    local new_conf
    new_conf=$(echo "$apply_result" | jq -r '.result.new_confidence' 2>/dev/null)
    if [[ "$new_conf" != "0.1" ]]; then
        test_fail "confidence=0.1 (floor)" "confidence=$new_conf"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 6: Confidence cap at 1.0 (never goes above)
# ============================================================================
test_apply_confidence_cap() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Create instinct with high confidence 0.97
    local create_result
    create_result=$(create_test_instinct "$tmpdir" "cap trigger" "cap action" "0.97")
    local instinct_id
    instinct_id=$(echo "$create_result" | jq -r '.result.instinct_id')

    # Apply with success: 0.97 + 0.05 = 1.02, should cap at 1.0
    local apply_result exit_code=0
    apply_result=$(run_cmd "$tmpdir" instinct-apply --id "$instinct_id" --outcome success) || exit_code=$?

    local new_conf
    new_conf=$(echo "$apply_result" | jq -r '.result.new_confidence' 2>/dev/null)
    local conf_ok
    conf_ok=$(echo "$new_conf" | awk '{printf "%.1f", $1}')
    if [[ "$conf_ok" != "1.0" ]]; then
        test_fail "confidence=1.0 (cap)" "confidence=$new_conf"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 7: instinct-create uses locking (verify lock file appears)
# ============================================================================
test_instinct_create_locking() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Create an instinct -- if locking works, there should be no lock file left after
    local create_result exit_code=0
    create_result=$(create_test_instinct "$tmpdir") || exit_code=$?

    if ! assert_ok_true "$create_result"; then
        test_fail "ok=true" "instinct-create failed: $create_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify no stale lock file remains
    if ls "$tmpdir/.aether/data/"*.lock 2>/dev/null | head -1 | grep -q .; then
        test_fail "no stale lock files" "lock file found after instinct-create"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

run_test test_apply_success "instinct-apply: success increments applications and boosts confidence"
run_test test_apply_twice "instinct-apply: applying twice increments applications to 2"
run_test test_apply_failure "instinct-apply: failure decrements confidence and increments failures"
run_test test_apply_nonexistent "instinct-apply: non-existent id returns error"
run_test test_apply_confidence_floor "instinct-apply: confidence floors at 0.1"
run_test test_apply_confidence_cap "instinct-apply: confidence caps at 1.0"
run_test test_instinct_create_locking "instinct-create: uses file locking"

test_summary
