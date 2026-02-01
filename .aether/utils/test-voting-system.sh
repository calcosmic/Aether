#!/bin/bash
# Aether Voting System Test Suite
# Tests supermajority calculation, Critical veto, issue deduping, weight calculation, vote recording
#
# Usage:
#   bash .aether/utils/test-voting-system.sh

# Find Aether root
if git rev-parse --show-toplevel >/dev/null 2>&1; then
    AETHER_ROOT="$(git rev-parse --show-toplevel)"
else
    SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    AETHER_ROOT="$(cd "$SCRIPT_PATH/../.." && pwd)"
fi

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test helper functions
test_start() {
    local test_name="$1"
    echo -n "Testing: $test_name ... "
    ((TESTS_RUN++))
}

test_pass() {
    echo -e "${GREEN}PASS${NC}"
    ((TESTS_PASSED++))
}

test_fail() {
    local reason="$1"
    echo -e "${RED}FAIL${NC} ($reason)"
    ((TESTS_FAILED++))
}

test_skip() {
    local reason="$1"
    echo -e "${YELLOW}SKIP${NC} ($reason)"
}

# Source utilities to test
source "$AETHER_ROOT/.aether/utils/vote-aggregator.sh"
source "$AETHER_ROOT/.aether/utils/issue-deduper.sh"
source "$AETHER_ROOT/.aether/utils/weight-calculator.sh"

# Create temporary test directory
TEST_DIR="$AETHER_ROOT/.aether/temp/voting_tests"
mkdir -p "$TEST_DIR/votes"

# ============================================================================
# TEST 1: Supermajority Calculation - Edge Cases
# ============================================================================
test_supermajority_edge_cases() {
    echo ""
    echo "=== Test Category: Supermajority Edge Cases ==="

    # Test 1.1: 4/4 APPROVE (100%) - should APPROVE
    test_start "4/4 APPROVE (100% >= 67%)"
    cat > "$TEST_DIR/votes/test_1_1.json" <<'EOF'
[
  {"watcher": "security", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "performance", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "quality", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "test_coverage", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"}
]
EOF
    result=$(calculate_supermajority "$TEST_DIR/votes/test_1_1.json" 2>&1)
    if echo "$result" | grep -q "APPROVED"; then
        test_pass
    else
        test_fail "Expected APPROVED, got: $result"
    fi

    # Test 1.2: 3/4 APPROVE (75%) - should APPROVE
    test_start "3/4 APPROVE (75% >= 67%)"
    cat > "$TEST_DIR/votes/test_1_2.json" <<'EOF'
[
  {"watcher": "security", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "performance", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "quality", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "test_coverage", "decision": "REJECT", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"}
]
EOF
    result=$(calculate_supermajority "$TEST_DIR/votes/test_1_2.json" 2>&1)
    if echo "$result" | grep -q "APPROVED"; then
        test_pass
    else
        test_fail "Expected APPROVED, got: $result"
    fi

    # Test 1.3: 2/4 APPROVE (50%) - should REJECT
    test_start "2/4 APPROVE (50% < 67%)"
    cat > "$TEST_DIR/votes/test_1_3.json" <<'EOF'
[
  {"watcher": "security", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "performance", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "quality", "decision": "REJECT", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "test_coverage", "decision": "REJECT", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"}
]
EOF
    result=$(calculate_supermajority "$TEST_DIR/votes/test_1_3.json" 2>&1)
    if echo "$result" | grep -q "REJECTED"; then
        test_pass
    else
        test_fail "Expected REJECTED, got: $result"
    fi

    # Test 1.4: 1/4 APPROVE (25%) - should REJECT
    test_start "1/4 APPROVE (25% < 67%)"
    cat > "$TEST_DIR/votes/test_1_4.json" <<'EOF'
[
  {"watcher": "security", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "performance", "decision": "REJECT", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "quality", "decision": "REJECT", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "test_coverage", "decision": "REJECT", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"}
]
EOF
    result=$(calculate_supermajority "$TEST_DIR/votes/test_1_4.json" 2>&1)
    if echo "$result" | grep -q "REJECTED"; then
        test_pass
    else
        test_fail "Expected REJECTED, got: $result"
    fi

    # Test 1.5: 0/4 APPROVE (0%) - should REJECT
    test_start "0/4 APPROVE (0% < 67%)"
    cat > "$TEST_DIR/votes/test_1_5.json" <<'EOF'
[
  {"watcher": "security", "decision": "REJECT", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "performance", "decision": "REJECT", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "quality", "decision": "REJECT", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "test_coverage", "decision": "REJECT", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"}
]
EOF
    result=$(calculate_supermajority "$TEST_DIR/votes/test_1_5.json" 2>&1)
    if echo "$result" | grep -q "REJECTED"; then
        test_pass
    else
        test_fail "Expected REJECTED, got: $result"
    fi
}

# ============================================================================
# TEST 2: Critical Veto Power
# ============================================================================
test_critical_veto() {
    echo ""
    echo "=== Test Category: Critical Veto Power ==="

    # Test 2.1: Critical issue blocks approval despite 3/4 APPROVE
    test_start "Critical issue blocks 3/4 APPROVE (veto)"
    cat > "$TEST_DIR/votes/test_2_1.json" <<'EOF'
[
  {"watcher": "security", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "performance", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "quality", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "test_coverage", "decision": "REJECT", "weight": 1.0, "issues": [
    {"severity": "Critical", "category": "test", "description": "No tests", "location": "test.py"}
  ], "timestamp": "2026-02-01T20:00:00Z"}
]
EOF
    result=$(calculate_supermajority "$TEST_DIR/votes/test_2_1.json" 2>&1)
    if echo "$result" | grep -q "REJECTED.*Critical veto"; then
        test_pass
    else
        test_fail "Expected Critical veto REJECT, got: $result"
    fi

    # Test 2.2: No Critical issue allows 3/4 APPROVE
    test_start "No Critical issue allows 3/4 APPROVE"
    cat > "$TEST_DIR/votes/test_2_2.json" <<'EOF'
[
  {"watcher": "security", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "performance", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "quality", "decision": "APPROVE", "weight": 1.0, "issues": [], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "test_coverage", "decision": "REJECT", "weight": 1.0, "issues": [
    {"severity": "High", "category": "test", "description": "Low coverage", "location": "test.py"}
  ], "timestamp": "2026-02-01T20:00:00Z"}
]
EOF
    result=$(calculate_supermajority "$TEST_DIR/votes/test_2_2.json" 2>&1)
    if echo "$result" | grep -q "APPROVED"; then
        test_pass
    else
        test_fail "Expected APPROVED, got: $result"
    fi
}

# ============================================================================
# TEST 3: Issue Deduplication
# ============================================================================
test_issue_deduping() {
    echo ""
    echo "=== Test Category: Issue Deduplication ==="

    # Test 3.1: Duplicate issues merged and tagged
    test_start "Duplicate issues merged with 'Multiple Watchers' tag"
    cat > "$TEST_DIR/votes/test_3_1.json" <<'EOF'
[
  {"watcher": "security", "decision": "REJECT", "weight": 1.0, "issues": [
    {"severity": "Critical", "category": "auth", "description": "Missing auth", "location": "app.py:10"}
  ], "timestamp": "2026-02-01T20:00:00Z"},
  {"watcher": "quality", "decision": "REJECT", "weight": 1.0, "issues": [
    {"severity": "Critical", "category": "auth", "description": "Missing auth", "location": "app.py:10"}
  ], "timestamp": "2026-02-01T20:00:00Z"}
]
EOF
    result=$(dedupe_and_prioritize "$TEST_DIR/votes/test_3_1.json")
    tag=$(echo "$result" | jq -r '.[0].tag')
    if [ "$tag" == "Multiple Watchers" ]; then
        test_pass
    else
        test_fail "Expected 'Multiple Watchers' tag, got: $tag"
    fi

    # Test 3.2: Issues sorted by severity (Critical first)
    test_start "Issues sorted by severity (Critical > High > Medium)"
    # (Implementation depends on issue-deduper.sh max_by logic)
    result=$(dedupe_and_prioritize "$TEST_DIR/votes/test_3_1.json")
    first_severity=$(echo "$result" | jq -r '.[0].severity')
    if [ "$first_severity" == "Critical" ]; then
        test_pass
    else
        test_fail "Expected Critical first, got: $first_severity"
    fi
}

# ============================================================================
# TEST 4: Weight Calculator
# ============================================================================
test_weight_calculator() {
    echo ""
    echo "=== Test Category: Weight Calculator ==="

    # Save original weights
    ORIGINAL_SECURITY=$(jq -r '.watcher_weights.security' "$AETHER_ROOT/.aether/data/watcher_weights.json")

    # Test 4.1: Correct approve increases weight
    test_start "Correct APPROVE increases weight (+0.1)"
    update_watcher_weight "security" "correct_approve" "" >/dev/null 2>&1
    new_weight=$(jq -r '.watcher_weights.security' "$AETHER_ROOT/.aether/data/watcher_weights.json")
    expected=$(awk "BEGIN {print $ORIGINAL_SECURITY + 0.1}")
    if [ "$(awk "BEGIN {print ($new_weight == $expected) ? 1 : 0}")" == "1" ]; then
        test_pass
    else
        test_fail "Expected $expected, got $new_weight"
    fi

    # Test 4.2: Correct reject increases weight more
    test_start "Correct REJECT increases weight (+0.15)"
    current=$(jq -r '.watcher_weights.security' "$AETHER_ROOT/.aether/data/watcher_weights.json")
    update_watcher_weight "security" "correct_reject" "" >/dev/null 2>&1
    new_weight=$(jq -r '.watcher_weights.security' "$AETHER_ROOT/.aether/data/watcher_weights.json")
    expected=$(awk "BEGIN {print $current + 0.15}")
    if [ "$(awk "BEGIN {print ($new_weight == $expected) ? 1 : 0}")" == "1" ]; then
        test_pass
    else
        test_fail "Expected $expected, got $new_weight"
    fi

    # Test 4.3: Incorrect approve decreases weight
    test_start "Incorrect APPROVE decreases weight (-0.2)"
    current=$(jq -r '.watcher_weights.security' "$AETHER_ROOT/.aether/data/watcher_weights.json")
    update_watcher_weight "security" "incorrect_approve" "" >/dev/null 2>&1
    new_weight=$(jq -r '.watcher_weights.security' "$AETHER_ROOT/.aether/data/watcher_weights.json")
    expected=$(awk "BEGIN {print $current - 0.2}")
    if [ "$(awk "BEGIN {print ($new_weight == $expected) ? 1 : 0}")" == "1" ]; then
        test_pass
    else
        test_fail "Expected $expected, got $new_weight"
    fi

    # Test 4.4: Weight clamped at minimum (0.1)
    test_start "Weight clamped at minimum (0.1)"
    # Set weight to 0.05, then apply -0.2 decrement
    jq '.watcher_weights.security = 0.05' "$AETHER_ROOT/.aether/data/watcher_weights.json" > "$TEST_DIR/temp_weights.json" && mv "$TEST_DIR/temp_weights.json" "$AETHER_ROOT/.aether/data/watcher_weights.json"
    update_watcher_weight "security" "incorrect_approve" "" >/dev/null 2>&1
    new_weight=$(jq -r '.watcher_weights.security' "$AETHER_ROOT/.aether/data/watcher_weights.json")
    if [ "$(awk "BEGIN {print ($new_weight == 0.1) ? 1 : 0}")" == "1" ]; then
        test_pass
    else
        test_fail "Expected 0.1 (clamped), got $new_weight"
    fi

    # Test 4.5: Weight clamped at maximum (3.0)
    test_start "Weight clamped at maximum (3.0)"
    # Set weight to 2.95, then apply +0.15 increment
    jq '.watcher_weights.security = 2.95' "$AETHER_ROOT/.aether/data/watcher_weights.json" > "$TEST_DIR/temp_weights.json" && mv "$TEST_DIR/temp_weights.json" "$AETHER_ROOT/.aether/data/watcher_weights.json"
    update_watcher_weight "security" "correct_reject" "" >/dev/null 2>&1
    new_weight=$(jq -r '.watcher_weights.security' "$AETHER_ROOT/.aether/data/watcher_weights.json")
    if [ "$(awk "BEGIN {print ($new_weight == 3.0) ? 1 : 0}")" == "1" ]; then
        test_pass
    else
        test_fail "Expected 3.0 (clamped), got $new_weight"
    fi

    # Test 4.6: Domain expertise bonus (×2)
    test_start "Domain expertise bonus applied (×2)"
    jq '.watcher_weights.security = 1.0' "$AETHER_ROOT/.aether/data/watcher_weights.json" > "$TEST_DIR/temp_weights.json" && mv "$TEST_DIR/temp_weights.json" "$AETHER_ROOT/.aether/data/watcher_weights.json"
    update_watcher_weight "security" "correct_approve" "security" >/dev/null 2>&1
    new_weight=$(jq -r '.watcher_weights.security' "$AETHER_ROOT/.aether/data/watcher_weights.json")
    # (1.0 + 0.1) * 2 = 2.2
    expected="2.2"
    if [ "$(awk "BEGIN {print ($new_weight == $expected) ? 1 : 0}")" == "1" ]; then
        test_pass
    else
        test_fail "Expected $expected (with domain bonus), got $new_weight"
    fi

    # Restore original weight
    jq ".watcher_weights.security = $ORIGINAL_SECURITY" "$AETHER_ROOT/.aether/data/watcher_weights.json" > "$TEST_DIR/temp_weights.json" && mv "$TEST_DIR/temp_weights.json" "$AETHER_ROOT/.aether/data/watcher_weights.json"
}

# ============================================================================
# TEST 5: Vote Recording
# ============================================================================
test_vote_recording() {
    echo ""
    echo "=== Test Category: Vote Recording ==="

    # Test 5.1: Vote recorded in COLONY_STATE.json
    test_start "Vote recorded in COLONY_STATE.json verification.votes"
    verification_id="test_verification_$(date +%s)"
    record_vote_outcome "security" "APPROVE" "[]" "$verification_id" >/dev/null 2>&1
    vote_count=$(jq -r ".verification.votes | map(select(.id == \"$verification_id\")) | length" "$AETHER_ROOT/.aether/data/COLONY_STATE.json")
    if [ "$vote_count" -eq 1 ]; then
        test_pass
    else
        test_fail "Expected 1 vote recorded, got $vote_count"
    fi

    # Test 5.2: Vote outcome set to "pending"
    test_start "Vote outcome set to 'pending'"
    outcome=$(jq -r ".verification.votes[] | select(.id == \"$verification_id\") | .outcome" "$AETHER_ROOT/.aether/data/COLONY_STATE.json")
    if [ "$outcome" == "pending" ]; then
        test_pass
    else
        test_fail "Expected 'pending', got $outcome"
    fi

    # Cleanup test vote
    jq "(.verification.votes |= map(select(.id != \"$verification_id\")))" "$AETHER_ROOT/.aether/data/COLONY_STATE.json" > "$TEST_DIR/temp_colony_state.json" && mv "$TEST_DIR/temp_colony_state.json" "$AETHER_ROOT/.aether/data/COLONY_STATE.json"
}

# ============================================================================
# MAIN TEST RUNNER
# ============================================================================
main() {
    echo "=========================================="
    echo "Aether Voting System Test Suite"
    echo "=========================================="
    echo ""

    # Run all test categories
    test_supermajority_edge_cases
    test_critical_veto
    test_issue_deduping
    test_weight_calculator
    test_vote_recording

    # Print summary
    echo ""
    echo "=========================================="
    echo "Test Summary"
    echo "=========================================="
    echo "Tests Run:    $TESTS_RUN"
    echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed.${NC}"
        exit 1
    fi
}

# Run tests
main
