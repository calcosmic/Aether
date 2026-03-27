#!/usr/bin/env bash
# Tests for hive-store confidence boosting on cross-repo merge
# Task 5.1: Confidence tiers based on source_repos count
#   2 repos -> 0.7, 3 repos -> 0.85, 4+ repos -> 0.95
# Confidence should never downgrade (max of current vs tier)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment with its own HOME
# ============================================================================
setup_hive_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"
    echo "$tmpdir"
}

# Helper: run any hive subcommand with isolated HOME
run_hive_cmd() {
    local tmpdir="$1"
    shift
    HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" "$@" 2>/dev/null
}

# ============================================================================
# Test 1: Cross-repo merge with 2 repos sets confidence to 0.7
# ============================================================================

test_confidence_boost_2_repos() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Store from repo-a with low initial confidence
    run_hive_cmd "$tmpdir" hive-store \
        --text "Always handle errors explicitly" \
        --source-repo "/tmp/repo-a" \
        --confidence 0.5 \
        --category "patterns" >/dev/null

    # Same text from repo-b -> should merge and boost confidence to 0.7
    local merge_result
    merge_result=$(run_hive_cmd "$tmpdir" hive-store \
        --text "Always handle errors explicitly" \
        --source-repo "/tmp/repo-b" \
        --confidence 0.5 \
        --category "patterns")

    if ! assert_json_field_equals "$merge_result" ".result.action" "merged"; then
        test_fail "Expected action=merged" "$merge_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check confidence in return JSON
    local result_confidence
    result_confidence=$(echo "$merge_result" | jq -r '.result.confidence')
    if [[ "$result_confidence" != "0.7" ]] && [[ "$result_confidence" != "0.70" ]]; then
        test_fail "Expected confidence=0.7 in merge result for 2 repos" "Got $result_confidence"
        rm -rf "$tmpdir"
        return 1
    fi

    # Also verify in wisdom.json file
    local wisdom_file="$tmpdir/.aether/hive/wisdom.json"
    local file_confidence
    file_confidence=$(jq '.entries[0].confidence' "$wisdom_file")
    # Compare with awk to handle floating point
    local match
    match=$(awk -v a="$file_confidence" -v b="0.7" 'BEGIN { print (a >= b - 0.001 && a <= b + 0.001) ? "yes" : "no" }')
    if [[ "$match" != "yes" ]]; then
        test_fail "Expected confidence=0.7 in wisdom.json for 2 repos" "Got $file_confidence"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: Cross-repo merge with 3 repos sets confidence to 0.85
# ============================================================================

test_confidence_boost_3_repos() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Store from 3 different repos
    run_hive_cmd "$tmpdir" hive-store \
        --text "Use dependency injection for testability" \
        --source-repo "/tmp/repo-a" \
        --confidence 0.5 \
        --category "architecture" >/dev/null

    run_hive_cmd "$tmpdir" hive-store \
        --text "Use dependency injection for testability" \
        --source-repo "/tmp/repo-b" \
        --confidence 0.5 \
        --category "architecture" >/dev/null

    local merge_result
    merge_result=$(run_hive_cmd "$tmpdir" hive-store \
        --text "Use dependency injection for testability" \
        --source-repo "/tmp/repo-c" \
        --confidence 0.5 \
        --category "architecture")

    if ! assert_json_field_equals "$merge_result" ".result.action" "merged"; then
        test_fail "Expected action=merged" "$merge_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check confidence is 0.85 for 3 repos
    local result_confidence
    result_confidence=$(echo "$merge_result" | jq -r '.result.confidence')
    local match
    match=$(awk -v a="$result_confidence" -v b="0.85" 'BEGIN { print (a >= b - 0.001 && a <= b + 0.001) ? "yes" : "no" }')
    if [[ "$match" != "yes" ]]; then
        test_fail "Expected confidence=0.85 in merge result for 3 repos" "Got $result_confidence"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify in file
    local wisdom_file="$tmpdir/.aether/hive/wisdom.json"
    local file_confidence
    file_confidence=$(jq '.entries[0].confidence' "$wisdom_file")
    match=$(awk -v a="$file_confidence" -v b="0.85" 'BEGIN { print (a >= b - 0.001 && a <= b + 0.001) ? "yes" : "no" }')
    if [[ "$match" != "yes" ]]; then
        test_fail "Expected confidence=0.85 in wisdom.json for 3 repos" "Got $file_confidence"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 3: Cross-repo merge with 4+ repos sets confidence to 0.95
# ============================================================================

test_confidence_boost_4plus_repos() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Store from 4 different repos
    run_hive_cmd "$tmpdir" hive-store \
        --text "Log all security events" \
        --source-repo "/tmp/repo-a" \
        --confidence 0.5 \
        --category "security" >/dev/null

    run_hive_cmd "$tmpdir" hive-store \
        --text "Log all security events" \
        --source-repo "/tmp/repo-b" \
        --confidence 0.5 \
        --category "security" >/dev/null

    run_hive_cmd "$tmpdir" hive-store \
        --text "Log all security events" \
        --source-repo "/tmp/repo-c" \
        --confidence 0.5 \
        --category "security" >/dev/null

    local merge_result
    merge_result=$(run_hive_cmd "$tmpdir" hive-store \
        --text "Log all security events" \
        --source-repo "/tmp/repo-d" \
        --confidence 0.5 \
        --category "security")

    if ! assert_json_field_equals "$merge_result" ".result.action" "merged"; then
        test_fail "Expected action=merged" "$merge_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check confidence is 0.95 for 4+ repos
    local result_confidence
    result_confidence=$(echo "$merge_result" | jq -r '.result.confidence')
    local match
    match=$(awk -v a="$result_confidence" -v b="0.95" 'BEGIN { print (a >= b - 0.001 && a <= b + 0.001) ? "yes" : "no" }')
    if [[ "$match" != "yes" ]]; then
        test_fail "Expected confidence=0.95 in merge result for 4 repos" "Got $result_confidence"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: 5+ repos still caps at 0.95
# ============================================================================

test_confidence_boost_5_repos_still_095() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    for repo in a b c d e; do
        run_hive_cmd "$tmpdir" hive-store \
            --text "Keep configuration in environment variables" \
            --source-repo "/tmp/repo-$repo" \
            --confidence 0.5 \
            --category "ops" >/dev/null
    done

    # Verify confidence is still 0.95 (not higher)
    local wisdom_file="$tmpdir/.aether/hive/wisdom.json"
    local file_confidence
    file_confidence=$(jq '.entries[0].confidence' "$wisdom_file")
    local match
    match=$(awk -v a="$file_confidence" -v b="0.95" 'BEGIN { print (a >= b - 0.001 && a <= b + 0.001) ? "yes" : "no" }')
    if [[ "$match" != "yes" ]]; then
        test_fail "Expected confidence=0.95 capped for 5 repos" "Got $file_confidence"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 5: Confidence never downgrades (high initial stays high)
# ============================================================================

test_confidence_never_downgrades() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Store with high initial confidence (0.9)
    run_hive_cmd "$tmpdir" hive-store \
        --text "Never store plaintext passwords" \
        --source-repo "/tmp/repo-a" \
        --confidence 0.9 \
        --category "security" >/dev/null

    # Merge from 2nd repo -> tier says 0.7, but current is 0.9 -> should stay 0.9
    local merge_result
    merge_result=$(run_hive_cmd "$tmpdir" hive-store \
        --text "Never store plaintext passwords" \
        --source-repo "/tmp/repo-b" \
        --confidence 0.5 \
        --category "security")

    if ! assert_json_field_equals "$merge_result" ".result.action" "merged"; then
        test_fail "Expected action=merged" "$merge_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Confidence should remain 0.9 (not downgraded to 0.7)
    local result_confidence
    result_confidence=$(echo "$merge_result" | jq -r '.result.confidence')
    local match
    match=$(awk -v a="$result_confidence" -v b="0.9" 'BEGIN { print (a >= b - 0.001 && a <= b + 0.001) ? "yes" : "no" }')
    if [[ "$match" != "yes" ]]; then
        test_fail "Expected confidence=0.9 (never downgrade from 0.9 to 0.7)" "Got $result_confidence"
        rm -rf "$tmpdir"
        return 1
    fi

    # Also verify in wisdom.json
    local wisdom_file="$tmpdir/.aether/hive/wisdom.json"
    local file_confidence
    file_confidence=$(jq '.entries[0].confidence' "$wisdom_file")
    match=$(awk -v a="$file_confidence" -v b="0.9" 'BEGIN { print (a >= b - 0.001 && a <= b + 0.001) ? "yes" : "no" }')
    if [[ "$match" != "yes" ]]; then
        test_fail "Expected confidence=0.9 in wisdom.json (no downgrade)" "Got $file_confidence"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 6: Return JSON includes confidence field
# ============================================================================

test_merge_return_includes_confidence() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    run_hive_cmd "$tmpdir" hive-store \
        --text "Validate all external inputs" \
        --source-repo "/tmp/repo-a" \
        --confidence 0.5 \
        --category "security" >/dev/null

    local merge_result
    merge_result=$(run_hive_cmd "$tmpdir" hive-store \
        --text "Validate all external inputs" \
        --source-repo "/tmp/repo-b" \
        --confidence 0.5 \
        --category "security")

    # Verify confidence field exists in return JSON
    local has_confidence
    has_confidence=$(echo "$merge_result" | jq 'has("result") and (.result | has("confidence"))')
    if [[ "$has_confidence" != "true" ]]; then
        test_fail "Expected .result.confidence field in merge return JSON" "$merge_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify it's a number
    local conf_type
    conf_type=$(echo "$merge_result" | jq -r '.result.confidence | type')
    if [[ "$conf_type" != "number" ]]; then
        test_fail "Expected confidence to be a number" "Got type=$conf_type"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running hive-store confidence boosting tests"
log_info "Repo root: $REPO_ROOT"

run_test test_confidence_boost_2_repos "2 repos -> confidence boosted to 0.7"
run_test test_confidence_boost_3_repos "3 repos -> confidence boosted to 0.85"
run_test test_confidence_boost_4plus_repos "4+ repos -> confidence boosted to 0.95"
run_test test_confidence_boost_5_repos_still_095 "5 repos -> confidence capped at 0.95"
run_test test_confidence_never_downgrades "Confidence never downgrades (max of current vs tier)"
run_test test_merge_return_includes_confidence "Merge return JSON includes confidence field"

test_summary
