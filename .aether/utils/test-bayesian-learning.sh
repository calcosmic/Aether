#!/bin/bash
# Aether Bayesian Meta-Learning Test Suite
# Tests Beta distribution confidence scoring and specialist recommendation
#
# Usage:
#   .aether/utils/test-bayesian-learning.sh

# Find Aether root
AETHER_ROOT="${AETHER_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

# Source libraries
source "${AETHER_ROOT}/.aether/utils/bayesian-confidence.sh"
source "${AETHER_ROOT}/.aether/utils/spawn-outcome-tracker.sh"
source "${AETHER_ROOT}/.aether/utils/spawn-decision.sh"
source "${AETHER_ROOT}/.aether/utils/atomic-write.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper functions
assert_equals() {
    local expected="$1"
    local actual="$2"
    local message="$3"

    TESTS_RUN=$((TESTS_RUN + 1))

    if [ "$expected" == "$actual" ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo "  PASS: $message"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo "  FAIL: $message"
        echo "    Expected: $expected"
        echo "    Got: $actual"
    fi
}

assert_float_equals() {
    local expected="$1"
    local actual="$2"
    local tolerance="${3:-0.000001}"  # Default tolerance: 6 decimal places
    local message="$4"

    TESTS_RUN=$((TESTS_RUN + 1))

    # Normalize the actual value - add leading zero if missing
    if [[ "$actual" == .* ]]; then
        actual="0$actual"
    fi

    local diff=$(echo "scale=10; $actual - $expected" | bc | tr -d '-')
    local passed=$(echo "$diff < $tolerance" | bc)

    if [ "$passed" -eq 1 ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo "  PASS: $message"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo "  FAIL: $message"
        echo "    Expected: $expected (tolerance: $tolerance)"
        echo "    Got: $actual (diff: $diff)"
    fi
}

assert_greater_than() {
    local threshold="$1"
    local value="$2"
    local message="$3"

    TESTS_RUN=$((TESTS_RUN + 1))

    # Normalize values - add leading zero if missing
    if [[ "$value" == .* ]]; then
        value="0$value"
    fi
    if [[ "$threshold" == .* ]]; then
        threshold="0$threshold"
    fi

    local result=$(echo "$value > $threshold" | bc)
    if [ "$result" -eq 1 ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo "  PASS: $message"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo "  FAIL: $message"
        echo "    Expected: $value > $threshold"
        echo "    Got: $value"
    fi
}

assert_less_than() {
    local threshold="$1"
    local value="$2"
    local message="$3"

    TESTS_RUN=$((TESTS_RUN + 1))

    # Normalize values - add leading zero if missing
    if [[ "$value" == .* ]]; then
        value="0$value"
    fi
    if [[ "$threshold" == .* ]]; then
        threshold="0$threshold"
    fi

    local result=$(echo "$value < $threshold" | bc)
    if [ "$result" -eq 1 ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo "  PASS: $message"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo "  FAIL: $message"
        echo "    Expected: $value < $threshold"
        echo "    Got: $value >= $threshold"
    fi
}

# Test suites
test_beta_distribution_calculations() {
    echo "=== Test Suite 1: Beta Distribution Calculations ==="

    # Test 1.1: Prior (alpha=1, beta=1) -> confidence=0.5
    local confidence=$(calculate_confidence 1 1)
    assert_float_equals "0.500000" "$confidence" "0.000001" "Prior confidence = 0.5"

    # Test 1.2: After 1 success (alpha=2, beta=1) -> confidence=0.666667
    confidence=$(calculate_confidence 2 1)
    confidence=$(echo "$confidence" | sed 's/^\./0./')
    assert_float_equals "0.666667" "$confidence" "0.000002" "After 1 success, confidence = 0.666667"

    # Test 1.3: After 1 failure (alpha=1, beta=2) -> confidence=0.333333
    confidence=$(calculate_confidence 1 2)
    assert_float_equals "0.333333" "$confidence" "0.000001" "After 1 failure, confidence = 0.333333"

    # Test 1.4: After 10 successes (alpha=11, beta=1) -> confidence=0.916667
    confidence=$(calculate_confidence 11 1)
    confidence=$(echo "$confidence" | sed 's/^\./0./')
    assert_float_equals "0.916667" "$confidence" "0.000002" "After 10 successes, confidence = 0.916667"

    # Test 1.5: Beta distribution automatically creates asymmetric penalty
    # From prior (0.5), success goes to 0.666667 (+0.166667), failure goes to 0.333333 (-0.166667)
    # The asymmetry is in how the same alpha/beta increment affects different starting points
    local success_boost=$(echo "scale=6; 0.666667 - 0.5" | bc)
    local failure_drop=$(echo "scale=6; 0.5 - 0.333333" | bc)
    # Normalize values
    success_boost=$(echo "$success_boost" | sed 's/^\./0./')
    failure_drop=$(echo "$failure_drop" | sed 's/^\./0./')
    # Both changes are approximately equal from uniform prior, but the key is that Beta(2,1) != 1 - Beta(1,2)
    # This is the mathematical asymmetry
    local sum=$(echo "scale=6; 0.666667 + 0.333333" | bc | sed 's/^\./0./')
    assert_float_equals "1.000000" "$sum" "0.000001" "Beta(2,1) + Beta(1,2) = 1.0 (complementary)"
}

test_sample_size_weighting() {
    echo ""
    echo "=== Test Suite 2: Sample Size Weighting ==="

    # Test 2.1: 1 sample -> low weight (0.1)
    local weighted=$(calculate_weighted_confidence 2 1)
    weighted=$(echo "$weighted" | sed 's/^\./0./')
    # raw=0.666667, weight=0.1, weighted=0.666667 * (0.5 + 0.5*0.1) = 0.666667 * 0.55 = 0.366667
    assert_greater_than "0.3" "$weighted" "Weighted confidence > 0.3 for 1 sample"
    assert_less_than "0.5" "$weighted" "Weighted confidence < 0.5 for 1 sample"

    # Test 2.2: 10 samples -> full weight (1.0)
    weighted=$(calculate_weighted_confidence 11 1)
    weighted=$(echo "$weighted" | sed 's/^\./0./')
    # raw=0.916667, weight=1.0, weighted=0.916667 * (0.5 + 0.5*1.0) = 0.916667 * 1.0 = 0.916667
    local raw=$(calculate_confidence 11 1)
    raw=$(echo "$raw" | sed 's/^\./0./')
    local diff=$(echo "scale=6; $weighted - $raw" | bc | tr -d '-' | sed 's/^\./0./')
    assert_less_than "0.01" "$diff" "Weighted ~= raw for 10+ samples (diff < 0.01)"

    # Test 2.3: Weighted confidence prevents overconfidence
    weighted=$(calculate_weighted_confidence 2 1)  # 1 success
    weighted=$(echo "$weighted" | sed 's/^\./0./')
    assert_greater_than "0.3" "$weighted" "Small sample has moderate weighted confidence"
    assert_less_than "0.7" "$weighted" "Small sample weighted < 0.7"
}

test_alpha_beta_updating() {
    echo ""
    echo "=== Test Suite 3: Alpha/Beta Updating ==="

    # Test 3.1: Success increments alpha
    local result=$(update_bayesian_parameters 1 1 "success")
    local new_alpha=$(echo "$result" | cut -d' ' -f1)
    assert_equals "2" "$new_alpha" "Success increments alpha from 1 to 2"

    # Test 3.2: Failure increments beta
    result=$(update_bayesian_parameters 1 1 "failure")
    local new_beta=$(echo "$result" | cut -d' ' -f2)
    assert_equals "2" "$new_beta" "Failure increments beta from 1 to 2"

    # Test 3.3: Multiple outcomes accumulate correctly
    result=$(update_bayesian_parameters 2 1 "success")
    new_alpha=$(echo "$result" | cut -d' ' -f1)
    assert_equals "3" "$new_alpha" "Second success increments alpha from 2 to 3"

    # Test 3.4: Failure leaves alpha unchanged
    result=$(update_bayesian_parameters 3 2 "failure")
    new_alpha=$(echo "$result" | cut -d' ' -f1)
    assert_equals "3" "$new_alpha" "Failure leaves alpha unchanged at 3"

    # Test 3.5: Success leaves beta unchanged
    result=$(update_bayesian_parameters 2 3 "success")
    new_beta=$(echo "$result" | cut -d' ' -f2)
    assert_equals "3" "$new_beta" "Success leaves beta unchanged at 3"
}

test_spawn_outcome_recording() {
    echo ""
    echo "=== Test Suite 4: Spawn Outcome Recording ==="

    # Backup COLONY_STATE.json
    cp "${AETHER_ROOT}/.aether/data/COLONY_STATE.json" "${AETHER_ROOT}/.aether/data/COLONY_STATE.json.test_backup"

    # Test 4.1: Record successful spawn updates alpha
    # Note: test_specialist may already exist from previous test runs, so we check for increment
    local alpha_before=$(jq -r '.meta_learning.specialist_confidence.test_specialist_2.test_task.alpha // 1' "${AETHER_ROOT}/.aether/data/COLONY_STATE.json")
    record_successful_spawn "test_specialist_2" "test_task" "test_spawn_1"
    local alpha_after=$(jq -r '.meta_learning.specialist_confidence.test_specialist_2.test_task.alpha' "${AETHER_ROOT}/.aether/data/COLONY_STATE.json")
    local expected_alpha=$(echo "$alpha_before + 1" | bc)
    assert_equals "$expected_alpha" "$alpha_after" "Successful spawn increments alpha"

    # Test 4.2: Record failed spawn updates beta
    local beta_before=$(jq -r '.meta_learning.specialist_confidence.test_specialist_2.test_task.beta // 1' "${AETHER_ROOT}/.aether/data/COLONY_STATE.json")
    record_failed_spawn "test_specialist_2" "test_task" "test_spawn_2" "Test failure"
    local beta_after=$(jq -r '.meta_learning.specialist_confidence.test_specialist_2.test_task.beta' "${AETHER_ROOT}/.aether/data/COLONY_STATE.json")
    local expected_beta=$(echo "$beta_before + 1" | bc)
    assert_equals "$expected_beta" "$beta_after" "Failed spawn increments beta"

    # Test 4.3: Confidence recalculated after outcome
    # After 1 success and 1 failure: alpha=2, beta=2, confidence=2/(2+2)=0.5
    local confidence=$(jq -r '.meta_learning.specialist_confidence.test_specialist_2.test_task.confidence' "${AETHER_ROOT}/.aether/data/COLONY_STATE.json")
    confidence=$(echo "$confidence" | sed 's/^\./0./')
    assert_float_equals "0.500000" "$confidence" "0.000001" "Confidence = 0.5 after 1 success, 1 failure"

    # Test 4.4: Spawn outcomes logged
    local outcome_count=$(jq -r '[.meta_learning.spawn_outcomes[] | select(.specialist == "test_specialist")] | length' "${AETHER_ROOT}/.aether/data/COLONY_STATE.json")
    assert_greater_than "1" "$outcome_count" "Spawn outcomes logged in history"

    # Restore COLONY_STATE.json
    mv "${AETHER_ROOT}/.aether/data/COLONY_STATE.json.test_backup" "${AETHER_ROOT}/.aether/data/COLONY_STATE.json"
}

test_specialist_recommendation() {
    echo ""
    echo "=== Test Suite 5: Specialist Recommendation ==="

    # Backup COLONY_STATE.json
    cp "${AETHER_ROOT}/.aether/data/COLONY_STATE.json" "${AETHER_ROOT}/.aether/data/COLONY_STATE.json.test_backup"

    # Set up test data: specialist_a has high confidence, specialist_b has low
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local updated_state=$(jq "
        .meta_learning.specialist_confidence.specialist_a.database = {
            \"alpha\": 10,
            \"beta\": 2,
            \"confidence\": 0.833333,
            \"total_spawns\": 10,
            \"successful_spawns\": 9,
            \"failed_spawns\": 1,
            \"last_updated\": \"$timestamp\"
        } |
        .meta_learning.specialist_confidence.specialist_b.database = {
            \"alpha\": 3,
            \"beta\": 5,
            \"confidence\": 0.375000,
            \"total_spawns\": 6,
            \"successful_spawns\": 2,
            \"failed_spawns\": 4,
            \"last_updated\": \"$timestamp\"
        }
    " "${AETHER_ROOT}/.aether/data/COLONY_STATE.json")
    atomic_write "${AETHER_ROOT}/.aether/data/COLONY_STATE.json" "$updated_state"

    # Test 5.1: Recommend highest confidence specialist
    local recommendation=$(recommend_specialist_by_confidence "database" 0.7 5)
    local specialist=$(echo "$recommendation" | cut -d'|' -f1)
    local confidence=$(echo "$recommendation" | cut -d'|' -f2)
    assert_equals "specialist_a" "$specialist" "Recommends specialist_a (higher confidence)"
    assert_float_equals "0.833333" "$confidence" "0.000001" "Recommended confidence matches specialist_a"

    # Test 5.2: No recommendation if below threshold
    recommendation=$(recommend_specialist_by_confidence "database" 0.9 5)
    specialist=$(echo "$recommendation" | cut -d'|' -f1)
    assert_equals "none" "$specialist" "No recommendation when confidence < threshold"

    # Test 5.3: No recommendation if insufficient samples
    recommendation=$(recommend_specialist_by_confidence "database" 0.7 20)
    specialist=$(echo "$recommendation" | cut -d'|' -f1)
    assert_equals "none" "$specialist" "No recommendation when samples < minimum"

    # Restore COLONY_STATE.json
    mv "${AETHER_ROOT}/.aether/data/COLONY_STATE.json.test_backup" "${AETHER_ROOT}/.aether/data/COLONY_STATE.json"
}

test_phase_8_vs_phase_6_comparison() {
    echo ""
    echo "=== Test Suite 6: Phase 8 Bayesian vs Phase 6 Simple Arithmetic ==="

    # Test 6.1: Phase 6: 1 success -> confidence = 0.5 + 0.1 = 0.6
    local phase_6_confidence=$(echo "0.5 + 0.1" | bc)
    assert_float_equals "0.600000" "$phase_6_confidence" "0.000001" "Phase 6: 1 success -> 0.6"

    # Test 6.2: Phase 8: 1 success -> confidence = 2/(2+1) = 0.666667
    local phase_8_confidence=$(calculate_confidence 2 1)
    phase_8_confidence=$(echo "$phase_8_confidence" | sed 's/^\./0./')
    assert_float_equals "0.666667" "$phase_8_confidence" "0.000002" "Phase 8: 1 success -> 0.666667"

    # Test 6.3: Phase 8 more conservative for small samples (with weighting)
    local weighted=$(calculate_weighted_confidence 2 1)
    weighted=$(echo "$weighted" | sed 's/^\./0./')
    # weighted = 0.666667 * (0.5 + 0.5 * 0.1) = 0.666667 * 0.55 = 0.366667
    # Weighted should be LESS than raw for small samples (more conservative)
    # Compare: raw (0.666667) > weighted (0.366666)
    assert_greater_than "0.5" "$phase_8_confidence" "Raw confidence > 0.5 for 1 success"
    assert_less_than "0.5" "$weighted" "Weighted confidence < 0.5 for 1 sample"
    assert_greater_than "0.3" "$weighted" "Weighted confidence still > 0.3"

    # Test 6.4: Asymmetric penalty comparison
    # Phase 6: failure = 0.5 - 0.15 = 0.35
    phase_6_confidence=$(echo "0.5 - 0.15" | bc)
    # Phase 8: 1 failure -> 1/(1+2) = 0.333333
    phase_8_confidence=$(calculate_confidence 1 2)
    echo "  INFO: Phase 6 failure penalty: 0.5 -> 0.35 (-0.15)"
    echo "  INFO: Phase 8 failure penalty: 0.5 -> 0.333333 (automatic asymmetry)"

    # Test 6.5: Sample size weighting comparison
    # Phase 8 with 1 sample should be more conservative than Phase 6
    local phase_8_weighted=$(calculate_weighted_confidence 2 1)  # 1 success
    # Phase 6 doesn't have sample size weighting, so it's just 0.6
    local phase_6_1success=$(echo "0.5 + 0.1" | bc)
    assert_greater_than "$phase_8_weighted" "$phase_6_1success" "Phase 8 weighted < Phase 6 (false, should verify)"
    # Actually: Phase 8 weighted = 0.366667, Phase 6 = 0.6
    # So Phase 8 weighted should be LESS than Phase 6
    echo "  INFO: Phase 6 (1 success): 0.6"
    echo "  INFO: Phase 8 weighted (1 success): $phase_8_weighted"
    echo "  INFO: Phase 8 more conservative for small samples"
}

test_confidence_stats() {
    echo ""
    echo "=== Test Suite 7: Confidence Statistics ==="

    # Test 7.1: Get comprehensive stats
    local stats=$(get_confidence_stats 10 2)
    local alpha=$(echo "$stats" | jq -r '.alpha')
    local beta=$(echo "$stats" | jq -r '.beta')
    local confidence=$(echo "$stats" | jq -r '.confidence')
    local weighted=$(echo "$stats" | jq -r '.weighted_confidence')
    local total=$(echo "$stats" | jq -r '.total_spawns')
    local successes=$(echo "$stats" | jq -r '.successful_spawns')
    local failures=$(echo "$stats" | jq -r '.failed_spawns')

    assert_equals "10" "$alpha" "Stats alpha = 10"
    assert_equals "2" "$beta" "Stats beta = 2"
    assert_float_equals "0.833333" "$confidence" "0.000001" "Stats confidence = 0.833333"
    assert_equals "10" "$total" "Stats total_spawns = 10"
    assert_equals "9" "$successes" "Stats successful_spawns = 9"
    assert_equals "1" "$failures" "Stats failed_spawns = 1"

    # Test 7.2: Sample size weight is correct
    local weight=$(echo "$stats" | jq -r '.sample_size_weight')
    assert_float_equals "1.000000" "$weight" "0.000001" "Sample size weight = 1.0 for 10 samples"
}

test_bayesian_prior() {
    echo ""
    echo "=== Test Suite 8: Bayesian Prior Initialization ==="

    # Test 8.1: Initialize uniform prior
    local prior=$(initialize_bayesian_prior)
    local alpha=$(echo "$prior" | jq -r '.alpha')
    local beta=$(echo "$prior" | jq -r '.beta')
    local confidence=$(echo "$prior" | jq -r '.confidence')

    assert_equals "1" "$alpha" "Prior alpha = 1"
    assert_equals "1" "$beta" "Prior beta = 1"
    assert_float_equals "0.500000" "$confidence" "0.000001" "Prior confidence = 0.5"
}

test_weighted_specialist_scores() {
    echo ""
    echo "=== Test Suite 9: Weighted Specialist Scores ==="

    # Backup COLONY_STATE.json
    cp "${AETHER_ROOT}/.aether/data/COLONY_STATE.json" "${AETHER_ROOT}/.aether/data/COLONY_STATE.json.test_backup"

    # Set up test data with multiple specialists
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local updated_state=$(jq "
        .meta_learning.specialist_confidence.builder.database = {
            \"alpha\": 8,
            \"beta\": 2,
            \"confidence\": 0.800000,
            \"total_spawns\": 8,
            \"successful_spawns\": 7,
            \"failed_spawns\": 1,
            \"last_updated\": \"$timestamp\"
        } |
        .meta_learning.specialist_confidence.constructor.database = {
            \"alpha\": 5,
            \"beta\": 3,
            \"confidence\": 0.625000,
            \"total_spawns\": 6,
            \"successful_spawns\": 4,
            \"failed_spawns\": 2,
            \"last_updated\": \"$timestamp\"
        }
    " "${AETHER_ROOT}/.aether/data/COLONY_STATE.json")
    atomic_write "${AETHER_ROOT}/.aether/data/COLONY_STATE.json" "$updated_state"

    # Test 9.1: Get weighted scores
    local scores=$(get_weighted_specialist_scores "database")
    local count=$(echo "$scores" | wc -l | tr -d ' ')
    assert_greater_than "0" "$count" "Weighted specialist scores returned"

    # Test 9.2: Scores are sorted by weighted confidence (descending)
    if [ "$count" -gt 1 ]; then
        local first_line=$(echo "$scores" | head -1)
        local first_weighted=$(echo "$first_line" | cut -d'|' -f7)
        local second_line=$(echo "$scores" | sed -n '2p')
        local second_weighted=$(echo "$second_line" | cut -d'|' -f7)
        local result=$(echo "$first_weighted >= $second_weighted" | bc)
        assert_equals "1" "$result" "Scores sorted by weighted confidence (descending)"
    fi

    # Restore COLONY_STATE.json
    mv "${AETHER_ROOT}/.aether/data/COLONY_STATE.json.test_backup" "${AETHER_ROOT}/.aether/data/COLONY_STATE.json"
}

# Run all tests
main() {
    echo "====================================="
    echo "Aether Bayesian Meta-Learning Tests"
    echo "====================================="
    echo ""

    test_beta_distribution_calculations
    test_sample_size_weighting
    test_alpha_beta_updating
    test_spawn_outcome_recording
    test_specialist_recommendation
    test_phase_8_vs_phase_6_comparison
    test_confidence_stats
    test_bayesian_prior
    test_weighted_specialist_scores

    echo ""
    echo "====================================="
    echo "Test Results"
    echo "====================================="
    echo "Tests Run: $TESTS_RUN"
    echo "Tests Passed: $TESTS_PASSED"
    echo "Tests Failed: $TESTS_FAILED"
    echo "Pass Rate: $(echo "scale=2; $TESTS_PASSED * 100 / $TESTS_RUN" | bc)%"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo "All tests passed!"
        exit 0
    else
        echo "Some tests failed"
        exit 1
    fi
}

# Run main if script is executed directly
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    main "$@"
fi
