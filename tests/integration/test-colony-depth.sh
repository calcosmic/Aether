#!/bin/bash
# test-colony-depth.sh — Integration tests for colony depth selector
# Tests the colony-depth get/set subcommand lifecycle

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
UTILS="$PROJECT_ROOT/.aether/aether-utils.sh"
pass=0; fail=0; total=0

# --- Setup ---
TMPDIR_BASE=$(mktemp -d)
trap 'rm -rf "$TMPDIR_BASE"' EXIT

setup_colony() {
    local test_dir="$TMPDIR_BASE/test_$$_$RANDOM"
    mkdir -p "$test_dir/.aether/data"
    cat > "$test_dir/.aether/data/COLONY_STATE.json" << 'JSON'
{"version":"3.0","goal":"test","state":"BUILDING","current_phase":1,"session_id":"test-session","plan":{"phases":[]},"memory":{},"errors":{"records":[]},"events":[]}
JSON
    echo "$test_dir"
}

run_depth() {
    local dir="$1"; shift
    AETHER_ROOT="$dir" bash "$UTILS" colony-depth "$@" 2>/dev/null
}

assert_eq() {
    total=$((total + 1))
    if [[ "$1" == "$2" ]]; then
        echo "  PASS: $3"
        pass=$((pass + 1))
    else
        echo "  FAIL: $3 (expected '$2', got '$1')"
        fail=$((fail + 1))
    fi
}

# --- Test 1: Default depth is "standard" ---
echo "Test 1: Default depth on missing field"
dir=$(setup_colony)
result=$(run_depth "$dir" get)
depth=$(echo "$result" | jq -r '.result.depth')
source=$(echo "$result" | jq -r '.result.source')
assert_eq "$depth" "standard" "default depth is standard"
assert_eq "$source" "default" "source is default when field missing"

# --- Test 2: Set and get depth ---
echo "Test 2: Set and get depth"
dir=$(setup_colony)
run_depth "$dir" set deep > /dev/null
result=$(run_depth "$dir" get)
depth=$(echo "$result" | jq -r '.result.depth')
source=$(echo "$result" | jq -r '.result.source')
assert_eq "$depth" "deep" "depth is deep after set"
assert_eq "$source" "colony_state" "source is colony_state after set"

# --- Test 3: All valid values ---
echo "Test 3: All valid depth values"
dir=$(setup_colony)
for val in light standard deep full; do
    run_depth "$dir" set "$val" > /dev/null
    depth=$(run_depth "$dir" get | jq -r '.result.depth')
    assert_eq "$depth" "$val" "depth set to $val"
done

# --- Test 4: Invalid value rejected ---
echo "Test 4: Invalid value rejected"
dir=$(setup_colony)
# json_err outputs to stderr; capture both streams, use .ok|tostring to handle boolean false
result=$(AETHER_ROOT="$dir" bash "$UTILS" colony-depth set invalid 2>&1 || true)
ok=$(echo "$result" | jq -r '.ok | tostring' 2>/dev/null || echo "parse_error")
assert_eq "$ok" "false" "invalid value returns ok:false"

# --- Test 5: Set returns updated:true ---
echo "Test 5: Set response format"
dir=$(setup_colony)
result=$(run_depth "$dir" set deep)
updated=$(echo "$result" | jq -r '.result.updated')
assert_eq "$updated" "true" "set returns updated:true"

# --- Test 6: Backward compatibility (existing colony without depth) ---
echo "Test 6: Backward compatibility"
dir=$(setup_colony)
# Verify existing COLONY_STATE.json fields are preserved after set
run_depth "$dir" set full > /dev/null
goal=$(jq -r '.goal' "$dir/.aether/data/COLONY_STATE.json")
version=$(jq -r '.version' "$dir/.aether/data/COLONY_STATE.json")
assert_eq "$goal" "test" "goal preserved after depth set"
assert_eq "$version" "3.0" "version preserved after depth set"

# --- Summary ---
echo ""
echo "Colony Depth Tests: $pass/$total passed, $fail failed"
[[ $fail -eq 0 ]] && exit 0 || exit 1
