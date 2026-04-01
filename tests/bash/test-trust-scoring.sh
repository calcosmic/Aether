#!/usr/bin/env bash
# Trust Scoring Module Tests
# Tests trust-scoring.sh functions via aether-utils.sh subcommands:
#   trust-calculate, trust-decay, trust-tier

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

if [[ ! -f "$AETHER_UTILS" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS"
    exit 1
fi

# ============================================================================
# Helper: isolated env with aether-utils.sh + all utils
# ============================================================================
setup_trust_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"

    cp "$AETHER_UTILS" "$tmpdir/.aether/aether-utils.sh"
    chmod +x "$tmpdir/.aether/aether-utils.sh"

    local utils_source
    utils_source="$(dirname "$AETHER_UTILS")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmpdir/.aether/"
    fi

    local exchange_source
    exchange_source="$(dirname "$AETHER_UTILS")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmpdir/.aether/"
    fi

    echo "$tmpdir"
}

run_cmd() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>/dev/null || true
}

# ============================================================================
# TEST 1: trust-calculate — user_feedback + test_verified + 0 days = high score
# ============================================================================
test_calculate_high_score() {
    local tmpdir
    tmpdir=$(setup_trust_env)

    local result
    result=$(run_cmd "$tmpdir" trust-calculate \
        --source user_feedback --evidence test_verified --days-since 0)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local score
    score=$(echo "$result" | jq -r '.result.score')
    # user_feedback (1.0)*0.4 + test_verified (1.0)*0.35 + 0 days (1.0)*0.25 = 1.0
    [[ $(echo "$score >= 0.95" | bc -l 2>/dev/null || awk "BEGIN{print ($score >= 0.95)}") == "1" ]] || return 1

    local tier
    tier=$(echo "$result" | jq -r '.result.tier')
    [[ "$tier" == "canonical" ]] || return 1
}

# ============================================================================
# TEST 2: trust-calculate — heuristic + anecdotal + 365 days = near floor
# ============================================================================
test_calculate_low_score() {
    local tmpdir
    tmpdir=$(setup_trust_env)

    local result
    result=$(run_cmd "$tmpdir" trust-calculate \
        --source heuristic --evidence anecdotal --days-since 365)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local score
    score=$(echo "$result" | jq -r '.result.score')
    # Should be near the 0.2 floor after heavy decay
    [[ $(echo "$score <= 0.35" | bc -l 2>/dev/null || awk "BEGIN{print ($score <= 0.35)}") == "1" ]] || return 1
}

# ============================================================================
# TEST 3: trust-decay — score 1.0 over 60 days => ~0.5
# ============================================================================
test_decay_half_life() {
    local tmpdir
    tmpdir=$(setup_trust_env)

    local result
    result=$(run_cmd "$tmpdir" trust-decay --score 1.0 --days 60)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local decayed
    decayed=$(echo "$result" | jq -r '.result.decayed')
    # 1.0 * (0.5^1) = 0.5, allow small floating-point tolerance (0.45–0.55)
    [[ $(echo "$decayed >= 0.45" | bc -l 2>/dev/null || awk "BEGIN{print ($decayed >= 0.45)}") == "1" ]] || return 1
    [[ $(echo "$decayed <= 0.55" | bc -l 2>/dev/null || awk "BEGIN{print ($decayed <= 0.55)}") == "1" ]] || return 1
}

# ============================================================================
# TEST 4: trust-decay — floor enforcement (score stays >= 0.2)
# ============================================================================
test_decay_floor() {
    local tmpdir
    tmpdir=$(setup_trust_env)

    # 0.21 over 9999 days would normally decay to essentially 0, but floor is 0.2
    local result
    result=$(run_cmd "$tmpdir" trust-decay --score 0.21 --days 9999)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local decayed
    decayed=$(echo "$result" | jq -r '.result.decayed')
    [[ $(echo "$decayed >= 0.2" | bc -l 2>/dev/null || awk "BEGIN{print ($decayed >= 0.2)}") == "1" ]] || return 1
}

# ============================================================================
# TEST 5: trust-tier — all 7 tiers map correctly
# ============================================================================
test_tier_canonical() {
    local tmpdir
    tmpdir=$(setup_trust_env)
    local result
    result=$(run_cmd "$tmpdir" trust-tier --score 0.95)
    rm -rf "$tmpdir"
    assert_ok_true "$result" || return 1
    [[ $(echo "$result" | jq -r '.result.tier') == "canonical" ]] || return 1
}

test_tier_trusted() {
    local tmpdir
    tmpdir=$(setup_trust_env)
    local result
    result=$(run_cmd "$tmpdir" trust-tier --score 0.85)
    rm -rf "$tmpdir"
    assert_ok_true "$result" || return 1
    [[ $(echo "$result" | jq -r '.result.tier') == "trusted" ]] || return 1
}

test_tier_established() {
    local tmpdir
    tmpdir=$(setup_trust_env)
    local result
    result=$(run_cmd "$tmpdir" trust-tier --score 0.75)
    rm -rf "$tmpdir"
    assert_ok_true "$result" || return 1
    [[ $(echo "$result" | jq -r '.result.tier') == "established" ]] || return 1
}

test_tier_emerging() {
    local tmpdir
    tmpdir=$(setup_trust_env)
    local result
    result=$(run_cmd "$tmpdir" trust-tier --score 0.65)
    rm -rf "$tmpdir"
    assert_ok_true "$result" || return 1
    [[ $(echo "$result" | jq -r '.result.tier') == "emerging" ]] || return 1
}

test_tier_provisional() {
    local tmpdir
    tmpdir=$(setup_trust_env)
    local result
    result=$(run_cmd "$tmpdir" trust-tier --score 0.52)
    rm -rf "$tmpdir"
    assert_ok_true "$result" || return 1
    [[ $(echo "$result" | jq -r '.result.tier') == "provisional" ]] || return 1
}

test_tier_suspect() {
    local tmpdir
    tmpdir=$(setup_trust_env)
    local result
    result=$(run_cmd "$tmpdir" trust-tier --score 0.37)
    rm -rf "$tmpdir"
    assert_ok_true "$result" || return 1
    [[ $(echo "$result" | jq -r '.result.tier') == "suspect" ]] || return 1
}

test_tier_dormant() {
    local tmpdir
    tmpdir=$(setup_trust_env)
    local result
    result=$(run_cmd "$tmpdir" trust-tier --score 0.25)
    rm -rf "$tmpdir"
    assert_ok_true "$result" || return 1
    [[ $(echo "$result" | jq -r '.result.tier') == "dormant" ]] || return 1
}

# ============================================================================
# TEST 6: Error handling — missing required arguments
# ============================================================================
test_calculate_missing_args() {
    local tmpdir
    tmpdir=$(setup_trust_env)

    # Call trust-calculate with no args — should return ok:false
    local result
    result=$(AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" trust-calculate 2>&1 || true)

    rm -rf "$tmpdir"

    # Should either exit non-zero or return ok:false
    assert_ok_false "$result" || [[ "$result" == *"Usage"* ]] || return 1
}

test_decay_missing_args() {
    local tmpdir
    tmpdir=$(setup_trust_env)

    local result
    result=$(AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" trust-decay 2>&1 || true)

    rm -rf "$tmpdir"

    assert_ok_false "$result" || [[ "$result" == *"Usage"* ]] || return 1
}

test_tier_missing_args() {
    local tmpdir
    tmpdir=$(setup_trust_env)

    local result
    result=$(AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" trust-tier 2>&1 || true)

    rm -rf "$tmpdir"

    assert_ok_false "$result" || [[ "$result" == *"Usage"* ]] || return 1
}

# ============================================================================
# TEST: component scores are included in trust-calculate output
# ============================================================================
test_calculate_includes_components() {
    local tmpdir
    tmpdir=$(setup_trust_env)

    local result
    result=$(run_cmd "$tmpdir" trust-calculate \
        --source success_pattern --evidence multi_phase --days-since 30)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1
    assert_json_has_field "$(echo "$result" | jq '.result')" "source_score" || return 1
    assert_json_has_field "$(echo "$result" | jq '.result')" "evidence_score" || return 1
    assert_json_has_field "$(echo "$result" | jq '.result')" "activity_score" || return 1
    assert_json_has_field "$(echo "$result" | jq '.result')" "tier" || return 1
}

# ============================================================================
# TEST: trust-decay output includes required fields
# ============================================================================
test_decay_output_fields() {
    local tmpdir
    tmpdir=$(setup_trust_env)

    local result
    result=$(run_cmd "$tmpdir" trust-decay --score 0.8 --days 30)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1
    local r
    r=$(echo "$result" | jq '.result')
    assert_json_has_field "$r" "original" || return 1
    assert_json_has_field "$r" "decayed" || return 1
    assert_json_has_field "$r" "days" || return 1
    assert_json_has_field "$r" "half_life" || return 1
    [[ $(echo "$r" | jq -r '.half_life') == "60" ]] || return 1
}

# ============================================================================
# Main: run all tests
# ============================================================================

log_info "Running trust-scoring module tests..."
log_info ""

run_test "test_calculate_high_score"         "trust-calculate: user_feedback + test_verified + 0 days => high score"
run_test "test_calculate_low_score"          "trust-calculate: heuristic + anecdotal + 365 days => near floor"
run_test "test_decay_half_life"              "trust-decay: 1.0 over 60 days => ~0.5"
run_test "test_decay_floor"                 "trust-decay: floor enforcement at 0.2"
run_test "test_tier_canonical"              "trust-tier: 0.95 => canonical"
run_test "test_tier_trusted"                "trust-tier: 0.85 => trusted"
run_test "test_tier_established"            "trust-tier: 0.75 => established"
run_test "test_tier_emerging"               "trust-tier: 0.65 => emerging"
run_test "test_tier_provisional"            "trust-tier: 0.52 => provisional"
run_test "test_tier_suspect"                "trust-tier: 0.37 => suspect"
run_test "test_tier_dormant"                "trust-tier: 0.25 => dormant"
run_test "test_calculate_missing_args"      "trust-calculate: missing args => error"
run_test "test_decay_missing_args"          "trust-decay: missing args => error"
run_test "test_tier_missing_args"           "trust-tier: missing args => error"
run_test "test_calculate_includes_components" "trust-calculate: output includes component scores"
run_test "test_decay_output_fields"         "trust-decay: output includes required fields"

test_summary
