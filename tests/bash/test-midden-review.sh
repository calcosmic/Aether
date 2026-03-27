#!/usr/bin/env bash
# Tests for midden-review and midden-acknowledge subcommands
# Tasks 2.1 + 2.2 + 2.3

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

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
  "goal": "test midden-review",
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

# Helper: write a midden entry to test env
write_midden_entry() {
    local tmpdir="$1"
    local category="${2:-general}"
    local message="${3:-test failure}"
    local source="${4:-test}"
    run_cmd "$tmpdir" midden-write "$category" "$message" "$source"
}

# ============================================================================
# Test 1: midden-review returns empty when no entries
# ============================================================================
test_review_empty() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-review) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_json_valid "$result"; then
        test_fail "valid JSON" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local count
    count=$(echo "$result" | jq -r '.result.unacknowledged_count')
    if [[ "$count" != "0" ]]; then
        test_fail "unacknowledged_count=0" "count=$count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: midden-review returns unacknowledged entries
# ============================================================================
test_review_returns_entries() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    # Write two entries
    write_midden_entry "$tmpdir" "security" "CVEs found: 3" "gatekeeper" >/dev/null
    write_midden_entry "$tmpdir" "quality" "Code smell detected" "auditor" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-review) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local count
    count=$(echo "$result" | jq -r '.result.unacknowledged_count')
    if [[ "$count" != "2" ]]; then
        test_fail "unacknowledged_count=2" "count=$count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check entries array length
    local entries_len
    entries_len=$(echo "$result" | jq '.result.entries | length')
    if [[ "$entries_len" != "2" ]]; then
        test_fail "entries length=2" "length=$entries_len"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check categories are grouped
    local sec_count
    sec_count=$(echo "$result" | jq -r '.result.categories.security // 0')
    if [[ "$sec_count" != "1" ]]; then
        test_fail "categories.security=1" "security=$sec_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 3: midden-review filters by category
# ============================================================================
test_review_filter_category() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    # Write entries in different categories
    write_midden_entry "$tmpdir" "security" "CVEs found: 3" "gatekeeper" >/dev/null
    write_midden_entry "$tmpdir" "quality" "Code smell" "auditor" >/dev/null
    write_midden_entry "$tmpdir" "security" "Secret exposed" "gatekeeper" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-review --category security) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local count
    count=$(echo "$result" | jq -r '.result.unacknowledged_count')
    if [[ "$count" != "2" ]]; then
        test_fail "unacknowledged_count=2 (security only)" "count=$count"
        rm -rf "$tmpdir"
        return 1
    fi

    # All entries should be security category
    local non_sec
    non_sec=$(echo "$result" | jq '[.result.entries[] | select(.category != "security")] | length')
    if [[ "$non_sec" != "0" ]]; then
        test_fail "all entries are security" "found $non_sec non-security entries"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: midden-acknowledge marks entry by id
# ============================================================================
test_acknowledge_by_id() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    # Write an entry and capture its id
    local write_result
    write_result=$(write_midden_entry "$tmpdir" "security" "CVEs found: 3" "gatekeeper")
    local entry_id
    entry_id=$(echo "$write_result" | jq -r '.result.entry_id')

    if [[ -z "$entry_id" || "$entry_id" == "null" ]]; then
        test_fail "entry should be created" "write failed: $write_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Acknowledge it
    local ack_result exit_code=0
    ack_result=$(run_cmd "$tmpdir" midden-acknowledge --id "$entry_id" --reason "Fixed in v2.1") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $ack_result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$ack_result"; then
        test_fail "ok=true" "ok was not true: $ack_result"
        rm -rf "$tmpdir"
        return 1
    fi

    local ack_count
    ack_count=$(echo "$ack_result" | jq -r '.result.count')
    if [[ "$ack_count" != "1" ]]; then
        test_fail "count=1" "count=$ack_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify the entry in midden.json has acknowledged=true
    local is_ack
    is_ack=$(jq --arg id "$entry_id" '[.entries[] | select(.id == $id)] | first | .acknowledged' "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$is_ack" != "true" ]]; then
        test_fail "entry acknowledged=true" "acknowledged=$is_ack"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify acknowledged_at is set
    local ack_at
    ack_at=$(jq -r --arg id "$entry_id" '[.entries[] | select(.id == $id)] | first | .acknowledged_at' "$tmpdir/.aether/data/midden/midden.json")
    if [[ -z "$ack_at" || "$ack_at" == "null" ]]; then
        test_fail "acknowledged_at is set" "acknowledged_at=$ack_at"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 5: acknowledged entries excluded from midden-review
# ============================================================================
test_acknowledged_excluded() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    # Write two entries
    local write1 write2
    write1=$(write_midden_entry "$tmpdir" "security" "CVEs found: 3" "gatekeeper")
    write2=$(write_midden_entry "$tmpdir" "quality" "Code smell" "auditor")
    local id1
    id1=$(echo "$write1" | jq -r '.result.entry_id')

    # Acknowledge the first entry
    run_cmd "$tmpdir" midden-acknowledge --id "$id1" --reason "Fixed" >/dev/null

    # Review should only show the unacknowledged entry
    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-review) || exit_code=$?

    local count
    count=$(echo "$result" | jq -r '.result.unacknowledged_count')
    if [[ "$count" != "1" ]]; then
        test_fail "unacknowledged_count=1 after acknowledging one" "count=$count"
        rm -rf "$tmpdir"
        return 1
    fi

    # The remaining entry should be the quality one
    local remaining_cat
    remaining_cat=$(echo "$result" | jq -r '.result.entries[0].category')
    if [[ "$remaining_cat" != "quality" ]]; then
        test_fail "remaining entry is quality" "remaining=$remaining_cat"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 6: midden-acknowledge by category marks all matching
# ============================================================================
test_acknowledge_by_category() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    # Write three entries: 2 security, 1 quality
    write_midden_entry "$tmpdir" "security" "CVEs found: 3" "gatekeeper" >/dev/null
    write_midden_entry "$tmpdir" "security" "Secret exposed" "gatekeeper" >/dev/null
    write_midden_entry "$tmpdir" "quality" "Code smell" "auditor" >/dev/null

    # Acknowledge all security entries
    local ack_result exit_code=0
    ack_result=$(run_cmd "$tmpdir" midden-acknowledge --category security --reason "All security issues fixed") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $ack_result"
        rm -rf "$tmpdir"
        return 1
    fi

    local ack_count
    ack_count=$(echo "$ack_result" | jq -r '.result.count')
    if [[ "$ack_count" != "2" ]]; then
        test_fail "count=2 (both security entries)" "count=$ack_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Review should only show the quality entry
    local review_result
    review_result=$(run_cmd "$tmpdir" midden-review)
    local remaining
    remaining=$(echo "$review_result" | jq -r '.result.unacknowledged_count')
    if [[ "$remaining" != "1" ]]; then
        test_fail "1 remaining after category ack" "remaining=$remaining"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

run_test test_review_empty "midden-review: returns empty when no entries"
run_test test_review_returns_entries "midden-review: returns unacknowledged entries"
run_test test_review_filter_category "midden-review: filters by category"
run_test test_acknowledge_by_id "midden-acknowledge: marks entry by id"
run_test test_acknowledged_excluded "midden-review: acknowledged entries excluded"
run_test test_acknowledge_by_category "midden-acknowledge: by category marks all matching"

test_summary
