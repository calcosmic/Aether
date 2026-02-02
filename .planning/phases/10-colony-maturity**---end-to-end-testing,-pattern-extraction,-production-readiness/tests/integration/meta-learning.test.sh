#!/bin/bash
# TAP test for Bayesian meta-learning
#
# Tests Phase 8 colony learning:
# - Initial confidence is 0.5 (uniform prior Beta(1,1))
# - Success increases confidence (alpha increment)
# - Failure decreases confidence (beta increment, asymmetric)
# - Confidence bounded [0.0, 1.0]
# - Sample size weighting prevents overconfidence
# - Specialist recommendation uses confidence threshold
# - Meta-learning data persisted to COLONY_STATE.json

set -e

# Source test helpers
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${TEST_DIR}/../helpers/colony-setup.sh"
source "${TEST_DIR}/../helpers/cleanup.sh"

# Source utility scripts under test
AETHER_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")"
source "${AETHER_ROOT}/.aether/utils/bayesian-confidence.sh"
source "${AETHER_ROOT}/.aether/utils/spawn-outcome-tracker.sh"
source "${AETHER_ROOT}/.aether/utils/spawn-decision.sh"

# Trap cleanup for state isolation
trap cleanup_test_colony EXIT

echo "1..7"  # Plan 7 assertions

# Test 1: Initial confidence is 0.5 (uniform prior Beta(1,1))
(
    setup_test_colony "Test initial confidence"

    # Calculate confidence from Beta(1,1) prior
    confidence=$(calculate_confidence 1 1)

    # Check confidence is 0.5 (allowing for floating point comparison)
    diff=$(echo "$confidence - 0.5" | bc)
    if [ $(echo "$diff < 0.000002 && $diff > -0.000002" | bc) -eq 1 ]; then
        echo "ok 1 - Initial confidence is neutral (0.5)"
    else
        echo "not ok 1 - Initial confidence is neutral (0.5)"
        echo "# Expected: 0.5, Got: $confidence"
        exit 1
    fi
)

# Test 2: Success increases confidence (alpha increment)
(
    setup_test_colony "Test confidence increase"

    # Start with Beta(1,1) = 0.5
    initial_confidence=$(calculate_confidence 1 1)

    # Update with success (alpha++)
    updated_params=$(update_bayesian_parameters 1 1 "success")
    new_alpha=$(echo "$updated_params" | cut -d' ' -f1)
    new_beta=$(echo "$updated_params" | cut -d' ' -f2)

    new_confidence=$(calculate_confidence "$new_alpha" "$new_beta")

    # New confidence should be 2/3 = 0.666...
    expected=$(echo "2 / 3" | bc -l)
    diff=$(echo "$new_confidence - $expected" | bc)

    if [ $(echo "$diff < 0.000002 && $diff > -0.000002" | bc) -eq 1 ]; then
        echo "ok 2 - Confidence increased after success"
    else
        echo "not ok 2 - Confidence increased after success"
        echo "# Expected: $expected, Got: $new_confidence"
        exit 1
    fi
)

# Test 3: Failure decreases confidence (beta increment, asymmetric)
(
    setup_test_colony "Test confidence decrease"

    # Start with Beta(1,1) = 0.5
    # Update with failure (beta++)
    updated_params=$(update_bayesian_parameters 1 1 "failure")
    new_alpha=$(echo "$updated_params" | cut -d' ' -f1)
    new_beta=$(echo "$updated_params" | cut -d' ' -f2)

    new_confidence=$(calculate_confidence "$new_alpha" "$new_beta")

    # New confidence should be 1/3 = 0.333...
    expected=$(echo "1 / 3" | bc -l)
    diff=$(echo "$new_confidence - $expected" | bc)

    if [ $(echo "$diff < 0.000002 && $diff > -0.000002" | bc) -eq 1 ]; then
        echo "ok 3 - Confidence decreased after failure"
    else
        echo "not ok 3 - Confidence decreased after failure"
        echo "# Expected: $expected, Got: $new_confidence"
        exit 1
    fi
)

# Test 4: Confidence bounded [0.0, 1.0]
(
    setup_test_colony "Test confidence bounds"

    # Test upper bound: try alpha=100, beta=1
    high_confidence=$(calculate_confidence 100 1)

    # Should be close to 1.0 (100/101 ≈ 0.9901)
    # The formula alpha/(alpha+beta) naturally bounds to [0,1]
    if [ $(echo "$high_confidence > 0.99" | bc) -eq 1 ] && [ $(echo "$high_confidence <= 1.0" | bc) -eq 1 ]; then
        echo "ok 4 - Confidence clamped to valid range"
    else
        echo "not ok 4 - Confidence clamped to valid range"
        echo "# High confidence: $high_confidence (expected 0.99 < x <= 1.0)"
        exit 1
    fi
)

# Test 5: Sample size weighting prevents overconfidence
(
    setup_test_colony "Test sample size weighting"

    # Both have same raw confidence (2/3 = 0.667)
    # Small sample: alpha=2, beta=1 (total = 2+1-2 = 1 sample)
    small_weighted=$(calculate_weighted_confidence 2 1)

    # Large sample: alpha=20, beta=10 (total = 20+10-2 = 28 samples)
    large_weighted=$(calculate_weighted_confidence 20 10)

    # Large sample should have higher weighted confidence (more trustworthy)
    if [ $(echo "$large_weighted > $small_weighted" | bc) -eq 1 ]; then
        echo "ok 5 - Small samples have lower weighted confidence"
    else
        echo "not ok 5 - Small samples have lower weighted confidence"
        echo "# Small weighted: $small_weighted, Large weighted: $large_weighted"
        exit 1
    fi
)

# Test 6: Specialist recommendation uses confidence threshold
(
    setup_test_colony "Test specialist recommendation"

    # Set up meta-learning with confidence below threshold
    # Alpha=2, Beta=3 => confidence = 2/5 = 0.4 (< 0.7 threshold)
    specialist="test_specialist"
    task_type="testing"

    # Update COLONY_STATE with low confidence specialist
    jq "
        .meta_learning.specialist_confidence.\"$specialist\".\"$task_type\" = {
            \"alpha\": 2,
            \"beta\": 3,
            \"confidence\": 0.4,
            \"total_spawns\": 3,
            \"successful_spawns\": 1,
            \"failed_spawns\": 2,
            \"last_updated\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
        }
    " "${COLONY_STATE_FILE}" > /tmp/colony_state.tmp
    mv /tmp/colony_state.tmp "${COLONY_STATE_FILE}"

    # Get recommendation (should return "none" due to low confidence)
    recommendation=$(recommend_specialist_by_confidence "$task_type" 0.7 5)
    rec_caste=$(echo "$recommendation" | cut -d'|' -f1)

    if [ "$rec_caste" = "none" ]; then
        echo "ok 6 - Recommendation requires confidence >= 0.7"
    else
        echo "not ok 6 - Recommendation requires confidence >= 0.7"
        echo "# Expected: none, Got: $rec_caste"
        exit 1
    fi
)

# Test 7: Meta-learning data persisted to COLONY_STATE.json
(
    setup_test_colony "Test meta-learning persistence"

    # Record a successful spawn outcome
    record_successful_spawn "test_specialist" "database" "spawn_test_001" 2>/dev/null || true

    # Check meta_learning section has specialist_confidence
    confidence=$(jq -r '.meta_learning.specialist_confidence.test_specialist.database.confidence // "null"' "${COLONY_STATE_FILE}")

    if [ "$confidence" != "null" ]; then
        # Check it's a valid number
        is_number=$(echo "$confidence > 0 && $confidence <= 1" | bc 2>/dev/null || echo "0")
        if [ "$is_number" -eq 1 ]; then
            echo "ok 7 - Confidence scores persisted"
        else
            echo "not ok 7 - Confidence scores persisted"
            echo "# Invalid confidence value: $confidence"
            exit 1
        fi
    else
        echo "not ok 7 - Confidence scores persisted"
        echo "# Confidence not found in COLONY_STATE.json"
        exit 1
    fi
)
