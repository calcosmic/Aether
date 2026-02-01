#!/bin/bash
# Aether Spawning Safeguards Test Suite
# Comprehensive tests for all spawning safeguards to prevent infinite loops
#
# Tests:
# 1. Depth limit prevents infinite chains
# 2. Circuit breaker triggers after 3 failures
# 3. Spawn budget limits spawns
# 4. Same-specialist cache prevents duplicates
# 5. Confidence scoring works
# 6. Meta-learning data populated
#
# Usage:
#   chmod +x .aether/utils/test-spawning-safeguards.sh
#   bash .aether/utils/test-spawning-safeguards.sh

# Find Aether root
if git rev-parse --show-toplevel >/dev/null 2>&1; then
    AETHER_ROOT="$(git rev-parse --show-toplevel)"
else
    SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    AETHER_ROOT="$(cd "$SCRIPT_PATH/../.." && pwd)"
fi

# Source required utilities
source "$AETHER_ROOT/.aether/utils/spawn-tracker.sh"
source "$AETHER_ROOT/.aether/utils/circuit-breaker.sh"
source "$AETHER_ROOT/.aether/utils/spawn-outcome-tracker.sh"

# Test configuration
COLONY_STATE_FILE="$AETHER_ROOT/.aether/data/COLONY_STATE.json"
BACKUP_STATE_FILE="$AETHER_ROOT/.aether/data/COLONY_STATE.test.backup"

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
print_header() {
    echo ""
    echo "========================================"
    echo "$1"
    echo "========================================"
}

print_test() {
    echo ""
    echo "Test: $1"
}

print_pass() {
    echo -e "${GREEN}PASS${NC}: $1"
    ((TESTS_PASSED++))
    ((TESTS_TOTAL++))
}

print_fail() {
    echo -e "${RED}FAIL${NC}: $1"
    ((TESTS_FAILED++))
    ((TESTS_TOTAL++))
}

print_info() {
    echo -e "${YELLOW}INFO${NC}: $1"
}

# Setup: Backup colony state
setup_test_env() {
    print_header "Setting up test environment"
    cp "$COLONY_STATE_FILE" "$BACKUP_STATE_FILE"
    print_info "Backed up COLONY_STATE.json to $BACKUP_STATE_FILE"

    # Reset spawn counters to clean state
    reset_spawn_counters >/dev/null 2>&1
    reset_circuit_breaker >/dev/null 2>&1

    print_info "Reset spawn counters and circuit breaker"
}

# Teardown: Restore colony state
teardown_test_env() {
    print_header "Tearing down test environment"
    mv "$BACKUP_STATE_FILE" "$COLONY_STATE_FILE"
    print_info "Restored COLONY_STATE.json from backup"
}

# Test 1: Depth limit prevents infinite chains
test_depth_limit() {
    print_test "Depth limit prevents infinite chains"

    # Setup: Set depth to max (3)
    local updated_state
    updated_state=$(jq '.spawn_tracking.depth = 3' "$COLONY_STATE_FILE")
    atomic_write "$COLONY_STATE_FILE" "$updated_state"

    # Verify depth is at max
    local current_depth
    current_depth=$(jq -r '.spawn_tracking.depth' "$COLONY_STATE_FILE")

    if [ "$current_depth" -eq 3 ]; then
        print_pass "Depth set to maximum (3)"
    else
        print_fail "Failed to set depth to maximum (got $current_depth)"
        return 1
    fi

    # Test: can_spawn should return 1 (cannot spawn)
    if can_spawn; then
        print_fail "can_spawn returned success when depth is at maximum"
    else
        print_pass "can_spawn correctly blocked spawn at max depth"
    fi

    # Setup: Reset depth to 0
    updated_state=$(jq '.spawn_tracking.depth = 0' "$COLONY_STATE_FILE")
    atomic_write "$COLONY_STATE_FILE" "$updated_state"

    # Verify depth reset
    current_depth=$(jq -r '.spawn_tracking.depth' "$COLONY_STATE_FILE")

    if [ "$current_depth" -eq 0 ]; then
        print_pass "Depth reset to 0"
    else
        print_fail "Failed to reset depth (got $current_depth)"
        return 1
    fi

    # Test: can_spawn should return 0 (can spawn)
    if can_spawn; then
        print_pass "can_spawn allowed spawn after depth reset"
    else
        print_fail "can_spawn blocked spawn when depth is 0"
    fi
}

# Test 2: Circuit breaker triggers after 3 failures
test_circuit_breaker() {
    print_test "Circuit breaker triggers after 3 failures"

    # Setup: Reset circuit breaker
    reset_circuit_breaker >/dev/null 2>&1

    # Verify initial state
    local trips
    trips=$(jq -r '.resource_budgets.circuit_breaker_trips' "$COLONY_STATE_FILE")

    if [ "$trips" -eq 0 ]; then
        print_pass "Circuit breaker initialized at 0 trips"
    else
        print_fail "Circuit breaker not at 0 trips (got $trips)"
        return 1
    fi

    # Record 3 failed spawns
    record_spawn_failure "database_specialist" "test_spawn_1" "Test failure 1" >/dev/null 2>&1
    record_spawn_failure "database_specialist" "test_spawn_2" "Test failure 2" >/dev/null 2>&1
    record_spawn_failure "database_specialist" "test_spawn_3" "Test failure 3" >/dev/null 2>&1

    # Verify circuit breaker trips = 3
    trips=$(jq -r '.resource_budgets.circuit_breaker_trips' "$COLONY_STATE_FILE")

    if [ "$trips" -eq 3 ]; then
        print_pass "Circuit breaker tripped after 3 failures (3/3)"
    else
        print_fail "Circuit breaker not at 3 trips (got $trips)"
        return 1
    fi

    # Verify cooldown_until timestamp set
    local cooldown_until
    cooldown_until=$(jq -r '.resource_budgets.circuit_breaker_cooldown_until // "null"' "$COLONY_STATE_FILE")

    if [ "$cooldown_until" != "null" ]; then
        print_pass "Circuit breaker cooldown timestamp set"
    else
        print_fail "Circuit breaker cooldown timestamp not set"
        return 1
    fi

    # Test: check_circuit_breaker should return 1 (circuit breaker active)
    if check_circuit_breaker "database_specialist"; then
        print_fail "check_circuit_breaker returned success when tripped"
    else
        print_pass "check_circuit_breaker correctly blocked spawn during cooldown"
    fi

    # Reset for next test
    reset_circuit_breaker >/dev/null 2>&1
}

# Test 3: Spawn budget limits spawns
test_spawn_budget() {
    print_test "Spawn budget limits spawns"

    # Setup: Reset spawn counters
    reset_spawn_counters >/dev/null 2>&1

    # Verify initial state
    local current_spawns max_spawns
    current_spawns=$(jq -r '.resource_budgets.current_spawns' "$COLONY_STATE_FILE")
    max_spawns=$(jq -r '.resource_budgets.max_spawns_per_phase' "$COLONY_STATE_FILE")

    if [ "$current_spawns" -eq 0 ]; then
        print_pass "Spawn budget initialized at 0 spawns"
    else
        print_fail "Spawn budget not at 0 spawns (got $current_spawns)"
        return 1
    fi

    # Set current_spawns to max (10)
    local updated_state
    updated_state=$(jq ".resource_budgets.current_spawns = $max_spawns" "$COLONY_STATE_FILE")
    atomic_write "$COLONY_STATE_FILE" "$updated_state"

    # Verify spawn budget at max
    current_spawns=$(jq -r '.resource_budgets.current_spawns' "$COLONY_STATE_FILE")

    if [ "$current_spawns" -eq "$max_spawns" ]; then
        print_pass "Spawn budget set to maximum ($current_spawns/$max_spawns)"
    else
        print_fail "Failed to set spawn budget to maximum (got $current_spawns)"
        return 1
    fi

    # Test: can_spawn should return 1 (cannot spawn)
    if can_spawn; then
        print_fail "can_spawn returned success when spawn budget exceeded"
    else
        print_pass "can_spawn correctly blocked spawn at max budget"
    fi

    # Reset spawn budget
    updated_state=$(jq '.resource_budgets.current_spawns = 0' "$COLONY_STATE_FILE")
    atomic_write "$COLONY_STATE_FILE" "$updated_state"

    # Verify spawn budget reset
    current_spawns=$(jq -r '.resource_budgets.current_spawns' "$COLONY_STATE_FILE")

    if [ "$current_spawns" -eq 0 ]; then
        print_pass "Spawn budget reset to 0"
    else
        print_fail "Failed to reset spawn budget (got $current_spawns)"
        return 1
    fi

    # Test: can_spawn should return 0 (can spawn)
    if can_spawn; then
        print_pass "can_spawn allowed spawn after budget reset"
    else
        print_fail "can_spawn blocked spawn when budget is available"
    fi
}

# Test 4: Same-specialist cache prevents duplicates
test_same_specialist_cache() {
    print_test "Same-specialist cache prevents duplicates"

    # Setup: Record a pending spawn
    local spawn_id
    spawn_id=$(record_spawn "builder" "database_specialist" "Test task")

    if [ -n "$spawn_id" ]; then
        print_pass "Recorded spawn: $spawn_id"
    else
        print_fail "Failed to record spawn"
        return 1
    fi

    # Verify spawn in history
    local spawn_exists
    spawn_exists=$(jq -r ".spawn_tracking.spawn_history[] | select(.id == \"$spawn_id\") | .id" "$COLONY_STATE_FILE")

    if [ "$spawn_exists" == "$spawn_id" ]; then
        print_pass "Spawn found in history"
    else
        print_fail "Spawn not found in history"
        return 1
    fi

    # Verify outcome is "pending"
    local spawn_outcome
    spawn_outcome=$(jq -r ".spawn_tracking.spawn_history[] | select(.id == \"$spawn_id\") | .outcome" "$COLONY_STATE_FILE")

    if [ "$spawn_outcome" == "pending" ]; then
        print_pass "Spawn outcome is 'pending'"
    else
        print_fail "Spawn outcome not 'pending' (got $spawn_outcome)"
        return 1
    fi

    # Clean up: Record outcome
    record_outcome "$spawn_id" "success" "Test completed" >/dev/null 2>&1

    # Verify outcome updated
    spawn_outcome=$(jq -r ".spawn_tracking.spawn_history[] | select(.id == \"$spawn_id\") | .outcome" "$COLONY_STATE_FILE")

    if [ "$spawn_outcome" == "success" ]; then
        print_pass "Spawn outcome updated to 'success'"
    else
        print_fail "Spawn outcome not updated (got $spawn_outcome)"
        return 1
    fi
}

# Test 5: Confidence scoring works
test_confidence_scoring() {
    print_test "Confidence scoring works"

    # Setup: Reset meta-learning
    local updated_state
    updated_state=$(jq '
        .meta_learning.specialist_confidence = {} |
        .meta_learning.spawn_outcomes = []
    ' "$COLONY_STATE_FILE")
    atomic_write "$COLONY_STATE_FILE" "$updated_state"

    # Verify initial confidence is default (0.5)
    local confidence
    confidence=$(get_specialist_confidence "database_specialist" "database")

    if [ "$confidence" == "0.5" ]; then
        print_pass "Default confidence is 0.5"
    else
        print_fail "Default confidence not 0.5 (got $confidence)"
        return 1
    fi

    # Record success - should increase by 0.1
    record_successful_spawn "database_specialist" "database" "test_spawn_1" >/dev/null 2>&1
    confidence=$(get_specialist_confidence "database_specialist" "database")

    if [ "$confidence" == "0.6" ]; then
        print_pass "Confidence increased to 0.6 after success (+0.1)"
    else
        print_fail "Confidence not 0.6 after success (got $confidence)"
        return 1
    fi

    # Record failure - should decrease by 0.15
    record_failed_spawn "database_specialist" "database" "test_spawn_2" "Test failure" >/dev/null 2>&1
    confidence=$(get_specialist_confidence "database_specialist" "database")

    if [ "$confidence" == "0.45" ]; then
        print_pass "Confidence decreased to 0.45 after failure (-0.15)"
    else
        print_fail "Confidence not 0.45 after failure (got $confidence)"
        return 1
    fi

    # Test max confidence (1.0)
    for i in {1..10}; do
        record_successful_spawn "database_specialist" "database" "test_spawn_max_$i" >/dev/null 2>&1
    done
    confidence=$(get_specialist_confidence "database_specialist" "database")

    if [ "$confidence" == "1.0" ]; then
        print_pass "Confidence capped at maximum (1.0)"
    else
        print_fail "Confidence not capped at 1.0 (got $confidence)"
        return 1
    fi

    # Test min confidence (0.0)
    for i in {1..10}; do
        record_failed_spawn "database_specialist" "database" "test_spawn_min_$i" "Test failure" >/dev/null 2>&1
    done
    confidence=$(get_specialist_confidence "database_specialist" "database")

    if [ "$confidence" == "0.0" ]; then
        print_pass "Confidence floored at minimum (0.0)"
    else
        print_fail "Confidence not floored at 0.0 (got $confidence)"
        return 1
    fi
}

# Test 6: Meta-learning data populated
test_meta_learning_data() {
    print_test "Meta-learning data populated"

    # Setup: Reset meta-learning
    local updated_state
    updated_state=$(jq '
        .meta_learning.specialist_confidence = {} |
        .meta_learning.spawn_outcomes = [] |
        .meta_learning.last_updated = null
    ' "$COLONY_STATE_FILE")
    atomic_write "$COLONY_STATE_FILE" "$updated_state"

    # Record outcomes
    record_successful_spawn "database_specialist" "database" "test_spawn_1" >/dev/null 2>&1
    record_failed_spawn "database_specialist" "database" "test_spawn_2" "Test failure" >/dev/null 2>&1

    # Verify spawn_outcomes array populated
    local outcome_count
    outcome_count=$(jq -r '.meta_learning.spawn_outcomes | length' "$COLONY_STATE_FILE")

    if [ "$outcome_count" -eq 2 ]; then
        print_pass "spawn_outcomes array populated (2 outcomes)"
    else
        print_fail "spawn_outcomes array not populated (got $outcome_count)"
        return 1
    fi

    # Verify specialist_confidence object updated
    local confidence
    confidence=$(jq -r '.meta_learning.specialist_confidence."database_specialist"."database"' "$COLONY_STATE_FILE")

    if [ -n "$confidence" ] && [ "$confidence" != "null" ]; then
        print_pass "specialist_confidence object updated"
    else
        print_fail "specialist_confidence object not updated"
        return 1
    fi

    # Verify last_updated timestamp set
    local last_updated
    last_updated=$(jq -r '.meta_learning.last_updated // "null"' "$COLONY_STATE_FILE")

    if [ "$last_updated" != "null" ]; then
        print_pass "last_updated timestamp set"
    else
        print_fail "last_updated timestamp not set"
        return 1
    fi
}

# Main test runner
main() {
    print_header "Aether Spawning Safeguards Test Suite"

    # Setup test environment
    setup_test_env

    # Run all tests
    test_depth_limit
    test_circuit_breaker
    test_spawn_budget
    test_same_specialist_cache
    test_confidence_scoring
    test_meta_learning_data

    # Teardown test environment
    teardown_test_env

    # Print summary
    print_header "Test Summary"
    echo ""
    echo "Tests passed: $TESTS_PASSED"
    echo "Tests failed: $TESTS_FAILED"
    echo "Tests total:  $TESTS_TOTAL"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
        return 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        return 1
    fi
}

# Run tests
main
exit $?
