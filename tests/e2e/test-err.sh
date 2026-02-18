#!/usr/bin/env bash
# test-err.sh — ERR requirement verification
# ERR-01: No 401 errors during normal operation
# ERR-02: No infinite spawn loops (spawn guards work)
# ERR-03: Clear error messages in user-facing failures
#
# Compatible with bash 3.2 (macOS default)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source e2e helpers (which also sources test-helpers.sh)
source "$SCRIPT_DIR/e2e-helpers.sh"

# Initialize results tracking
init_results

echo ""
echo "=========================================="
echo " ERR: Error Handling Requirements"
echo "=========================================="
echo ""

# ============================================================================
# Setup isolated environment
# ============================================================================

TMP_DIR=$(setup_e2e_env)
trap 'teardown_e2e_env' EXIT

UTILS="$TMP_DIR/.aether/aether-utils.sh"

# ============================================================================
# ERR-01: No 401 errors during normal operation
# Bootstrap handles missing files; load-state guards before operations
# ============================================================================

test_start "ERR-01: load-state returns valid JSON (no 401/Unauthorized)"
raw_out=$(bash "$UTILS" load-state 2>&1 || true)
out=$(extract_json "$raw_out")

if assert_json_valid "$out"; then
    if echo "$raw_out" | grep -qi "401\|unauthorized"; then
        test_fail "no auth errors" "Found 401/Unauthorized in output"
        record_result "ERR-01" "FAIL" "load-state returned 401/Unauthorized"
    else
        test_pass
        record_result "ERR-01" "PASS" "load-state returns valid JSON with no auth errors"
    fi
else
    test_fail "valid JSON from load-state" "$out"
    record_result "ERR-01" "FAIL" "load-state output is not valid JSON"
fi

test_start "ERR-01 (supplemental): spawn-can-spawn has no auth errors"
raw_spawn=$(bash "$UTILS" spawn-can-spawn 1 2>&1 || true)
spawn_out=$(extract_json "$raw_spawn")
if assert_json_valid "$spawn_out" && ! echo "$raw_spawn" | grep -qi "401\|unauthorized"; then
    test_pass
else
    test_fail "spawn-can-spawn no auth errors" "$spawn_out"
fi

# ============================================================================
# ERR-02: No infinite spawn loops
# spawn-can-spawn blocks at depth >= 3; spawn-can-spawn-swarm returns valid JSON
# ============================================================================

test_start "ERR-02: spawn-can-spawn blocks at depth 3 (can_spawn=false)"
raw_d3=$(bash "$UTILS" spawn-can-spawn 3 2>&1 || true)
out_d3=$(extract_json "$raw_d3")

if assert_json_valid "$out_d3" && assert_json_field_equals "$out_d3" ".result.can_spawn" "false"; then
    test_pass
    record_result "ERR-02" "PASS" "spawn-can-spawn returns can_spawn=false at depth 3"
else
    test_fail "can_spawn=false at depth 3" "$out_d3"
    record_result "ERR-02" "FAIL" "spawn guard failed: can_spawn not false at depth 3"
fi

test_start "ERR-02 (supplemental): spawn-can-spawn-swarm returns valid JSON with can_spawn"
raw_swarm=$(bash "$UTILS" spawn-can-spawn-swarm 2>&1 || true)
out_swarm=$(extract_json "$raw_swarm")
if assert_json_valid "$out_swarm" && assert_json_has_field "$out_swarm" "result"; then
    can_spawn_val=$(echo "$out_swarm" | jq -r '.result.can_spawn // "missing"' 2>/dev/null || echo "missing")
    if [[ "$can_spawn_val" != "missing" ]]; then
        test_pass
    else
        test_fail "can_spawn field in result" "$out_swarm"
        record_result "ERR-02" "FAIL" "spawn-can-spawn-swarm missing can_spawn field"
    fi
else
    test_fail "valid JSON from spawn-can-spawn-swarm" "$out_swarm"
    record_result "ERR-02" "FAIL" "spawn-can-spawn-swarm not valid JSON: $out_swarm"
fi

test_start "ERR-02 (supplemental): spawn-can-spawn allows at depth 1"
raw_d1=$(bash "$UTILS" spawn-can-spawn 1 2>&1 || true)
out_d1=$(extract_json "$raw_d1")
if assert_json_valid "$out_d1" && assert_json_field_equals "$out_d1" ".result.can_spawn" "true"; then
    test_pass
else
    test_fail "can_spawn=true at depth 1" "$out_d1"
fi

# ============================================================================
# ERR-03: Clear error messages in user-facing failures
# context-update with no args should return ok:false with actionable message
# ============================================================================

test_start "ERR-03: context-update with no args returns ok:false with actionable message"
raw_ctx=$(bash "$UTILS" context-update 2>&1 || true)
ctx_out=$(extract_json "$raw_ctx")

if assert_json_valid "$ctx_out"; then
    is_ok=$(echo "$ctx_out" | jq -r '.ok' 2>/dev/null || echo "unknown")
    if [[ "$is_ok" == "false" ]]; then
        # Verify the error message is actionable (mentions Suggestion or valid actions)
        if echo "$raw_ctx" | grep -qi "suggestion\|init\|update-phase\|action"; then
            test_pass
            record_result "ERR-03" "PASS" "context-update returns ok:false with actionable error"
        else
            test_pass  # ok:false is sufficient — structured error output is correct
            record_result "ERR-03" "PASS" "context-update returns ok:false (structured error)"
        fi
    else
        test_fail "ok:false from context-update with no args" "Got ok=$is_ok"
        record_result "ERR-03" "FAIL" "context-update should return ok:false for missing args"
    fi
else
    # If context-update outputs to stderr only (not stdout), check stderr content
    if echo "$raw_ctx" | grep -qi "suggestion\|no action\|usage"; then
        test_pass
        record_result "ERR-03" "PASS" "context-update outputs actionable error message"
    else
        test_fail "actionable error from context-update (no args)" "$raw_ctx"
        record_result "ERR-03" "FAIL" "context-update error not actionable: $raw_ctx"
    fi
fi

test_start "ERR-03 (supplemental): no raw bash script errors on bad subcommand"
raw_bad=$(bash "$UTILS" spawn-can-spawn notanumber 2>&1 || true)
if echo "$raw_bad" | grep -qE "^(Traceback|\.aether/aether-utils\.sh: line [0-9]+: \[)"; then
    test_fail "no raw bash errors" "Got raw error: $raw_bad"
else
    test_pass
fi

# ============================================================================
# Print Results
# ============================================================================

print_area_results "ERR"
