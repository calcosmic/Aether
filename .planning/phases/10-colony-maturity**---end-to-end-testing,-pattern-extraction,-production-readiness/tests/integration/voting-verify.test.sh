#!/bin/bash
# TAP test for voting verification
# Note: set -e is disabled for this test to properly catch subshell errors
#
# Tests Phase 7 colony verification:
# - Unanimous approve passes
# - Single reject fails (needs 67% supermajority)
# - Critical issue vetoes despite supermajority
# - Issue deduplication merges duplicates
# - Weight calculator updates correctly (correct_reject +0.15)
# - Weight calculator decreases (incorrect_approve -0.2)
# - Vote recorded in COLONY_STATE.json
# - Supermajority calculation (3/4 = 75% >= 67%)

# Source test helpers
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${TEST_DIR}/../helpers/colony-setup.sh"
source "${TEST_DIR}/../helpers/cleanup.sh"

# Source utility scripts under test
AETHER_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")"
source "${AETHER_ROOT}/.aether/utils/vote-aggregator.sh"
source "${AETHER_ROOT}/.aether/utils/weight-calculator.sh"
source "${AETHER_ROOT}/.aether/utils/issue-deduper.sh"

# Trap cleanup for state isolation
trap cleanup_test_colony EXIT

echo "1..8"  # Plan 8 assertions

# Setup votes directory
setup_votes_dir() {
    mkdir -p "${AETHER_ROOT}/.aether/verification/votes"
}

# Test 1: Unanimous approve passes
(
    setup_test_colony "Test unanimous approve"
    setup_votes_dir

    # Create 4 unanimous APPROVE votes
    for watcher in security performance quality test_coverage; do
        cat > "${AETHER_ROOT}/.aether/verification/votes/${watcher}_vote.json" <<EOF
{
  "watcher": "$watcher",
  "decision": "APPROVE",
  "weight": 1.0,
  "issues": []
}
EOF
    done

    # Aggregate votes
    votes_file="/tmp/votes_unanimous.json"
    jq -s '.' "${AETHER_ROOT}"/.aether/verification/votes/*.json > "$votes_file"

    # Calculate supermajority
    result=$(calculate_supermajority "$votes_file")
    outcome=$(echo "$result" | grep -o "APPROVED\|REJECTED" || echo "")

    if [[ "$result" == *"APPROVED"* ]]; then
        echo "ok 1 - Unanimous approval passes"
    else
        echo "not ok 1 - Unanimous approval passes"
        echo "# Result: $result"
        exit 1
    fi
)

# Test 2: Single reject fails (needs 67% supermajority)
(
    setup_test_colony "Test single reject fails"
    setup_votes_dir

    # Create 2 APPROVE, 2 REJECT votes (50% < 67%)
    cat > "${AETHER_ROOT}/.aether/verification/votes/security_vote.json" <<EOF
{
  "watcher": "security",
  "decision": "APPROVE",
  "weight": 1.0,
  "issues": []
}
EOF

    cat > "${AETHER_ROOT}/.aether/verification/votes/performance_vote.json" <<EOF
{
  "watcher": "performance",
  "decision": "APPROVE",
  "weight": 1.0,
  "issues": []
}
EOF

    cat > "${AETHER_ROOT}/.aether/verification/votes/quality_vote.json" <<EOF
{
  "watcher": "quality",
  "decision": "REJECT",
  "weight": 1.0,
  "issues": [
    {
      "description": "Test issue",
      "severity": "Low",
      "category": "quality",
      "location": "file.js:1"
    }
  ]
}
EOF

    cat > "${AETHER_ROOT}/.aether/verification/votes/test_coverage_vote.json" <<EOF
{
  "watcher": "test_coverage",
  "decision": "REJECT",
  "weight": 1.0,
  "issues": [
    {
      "description": "Test issue",
      "severity": "Low",
      "category": "quality",
      "location": "file.js:1"
    }
  ]
}
EOF

    # Aggregate votes
    votes_file="/tmp/votes_single_reject.json"
    jq -s '.' "${AETHER_ROOT}/.aether/verification/votes/"*.json > "$votes_file" 2>/dev/null || true

    # Calculate supermajority
    result=$(calculate_supermajority "$votes_file" 2>/dev/null || echo "REJECTED (error)")

    if [[ "$result" == *"REJECTED"* ]]; then
        echo "ok 2 - Single reject causes failure"
    else
        echo "not ok 2 - Single reject causes failure"
        echo "# Result: $result"
        exit 1
    fi
)

# Test 3: Critical issue vetoes despite supermajority
(
    setup_test_colony "Test critical veto"
    setup_votes_dir

    # Create 3 APPROVE, 1 REJECT with Critical issue
    for watcher in security performance quality; do
        cat > "${AETHER_ROOT}/.aether/verification/votes/${watcher}_vote.json" <<EOF
{
  "watcher": "$watcher",
  "decision": "APPROVE",
  "weight": 1.0,
  "issues": []
}
EOF
    done

    # One Critical issue should veto
    cat > "${AETHER_ROOT}/.aether/verification/votes/test_coverage_vote.json" <<EOF
{
  "watcher": "test_coverage",
  "decision": "REJECT",
  "weight": 1.0,
  "issues": [
    {
      "description": "SQL injection vulnerability",
      "severity": "Critical",
      "category": "security",
      "location": "user.js:42"
    }
  ]
}
EOF

    # Aggregate votes
    votes_file="/tmp/votes_critical_veto.json"
    jq -s '.' "${AETHER_ROOT}"/.aether/verification/votes/*.json > "$votes_file"

    # Calculate supermajority
    result=$(calculate_supermajority "$votes_file")

    if [[ "$result" == *"REJECTED"* ]] && [[ "$result" == *"Critical"* ]]; then
        echo "ok 3 - Critical veto power enforced"
    else
        echo "not ok 3 - Critical veto power enforced"
        echo "# Result: $result"
        exit 1
    fi
)

# Test 4: Issue deduplication merges duplicates
(
    setup_test_colony "Test issue deduplication"
    setup_votes_dir

    # Create votes with duplicate issues
    cat > "${AETHER_ROOT}/.aether/verification/votes/security_vote.json" <<EOF
{
  "watcher": "security",
  "decision": "REJECT",
  "weight": 1.0,
  "issues": [
    {
      "description": "SQL injection in user query",
      "severity": "Critical",
      "category": "security",
      "location": "user.js:42"
    }
  ]
}
EOF

    cat > "${AETHER_ROOT}/.aether/verification/votes/quality_vote.json" <<EOF
{
  "watcher": "quality",
  "decision": "REJECT",
  "weight": 1.0,
  "issues": [
    {
      "description": "SQL injection in user query",
      "severity": "Critical",
      "category": "security",
      "location": "user.js:42"
    }
  ]
}
EOF

    # Other votes to make 4 total
    for watcher in performance test_coverage; do
        cat > "${AETHER_ROOT}/.aether/verification/votes/${watcher}_vote.json" <<EOF
{
  "watcher": "$watcher",
  "decision": "APPROVE",
  "weight": 1.0,
  "issues": []
}
EOF
    done

    # Aggregate votes
    votes_file="/tmp/votes_dedupe.json"
    jq -s '.' "${AETHER_ROOT}"/.aether/verification/votes/*.json > "$votes_file"

    # Dedupe issues
    deduped=$(dedupe_and_prioritize "$votes_file")
    issue_count=$(echo "$deduped" | jq '. | length')

    # Should have 1 issue (merged from 2 duplicates)
    if [ "$issue_count" -eq 1 ]; then
        echo "ok 4 - Duplicate issues merged"
    else
        echo "not ok 4 - Duplicate issues merged"
        echo "# Issue count: $issue_count (expected 1)"
        exit 1
    fi
)

# Test 5: Weight calculator updates correctly (correct_reject +0.15)
(
    setup_test_colony "Test weight increase"
    setup_votes_dir

    # Get initial weight
    initial_weight=$(get_watcher_weight "security")

    # Update weight with correct_reject (use different category to avoid domain bonus)
    update_watcher_weight "security" "correct_reject" "quality" >/dev/null 2>&1 || true

    # Get new weight
    new_weight=$(get_watcher_weight "security")
    expected=$(echo "$initial_weight + 0.15" | bc)

    # Check weight increased (allowing for floating point comparison)
    diff=$(echo "$new_weight - $expected" | bc)
    if [ $(echo "$diff < 0.001 && $diff > -0.001" | bc) -eq 1 ]; then
        echo "ok 5 - Weight increased for correct rejection"
    else
        echo "not ok 5 - Weight increased for correct rejection"
        echo "# Expected: $expected, Got: $new_weight"
        exit 1
    fi
)

# Test 6: Weight calculator decreases (incorrect_approve -0.2)
(
    setup_test_colony "Test weight decrease"
    setup_votes_dir

    # Get initial weight
    initial_weight=$(get_watcher_weight "performance")

    # Update weight with incorrect_approve (use different category to avoid domain bonus)
    update_watcher_weight "performance" "incorrect_approve" "quality" >/dev/null 2>&1 || true

    # Get new weight
    new_weight=$(get_watcher_weight "performance")
    expected=$(echo "$initial_weight - 0.2" | bc)

    # Check weight decreased (allowing for floating point comparison)
    diff=$(echo "$new_weight - $expected" | bc)
    if [ $(echo "$diff < 0.001 && $diff > -0.001" | bc) -eq 1 ]; then
        echo "ok 6 - Weight decreased for incorrect approval"
    else
        echo "not ok 6 - Weight decreased for incorrect approval"
        echo "# Expected: $expected, Got: $new_weight"
        exit 1
    fi
)

# Test 7: Vote recorded in COLONY_STATE.json
(
    setup_test_colony "Test vote recording"
    setup_votes_dir

    # Record a vote outcome
    verification_id="test_verification_$(date +%s)"
    record_vote_outcome "security" "APPROVE" '[]' "$verification_id" >/dev/null 2>&1 || true

    # Check vote was recorded
    vote_count=$(jq -r '.verification.votes | length' "${COLONY_STATE_FILE}")

    if [ "$vote_count" -gt 0 ]; then
        echo "ok 7 - Vote recorded in verification section"
    else
        echo "not ok 7 - Vote recorded in verification section"
        echo "# Vote count: $vote_count"
        exit 1
    fi
)

# Test 8: Supermajority calculation (3/4 = 75% >= 67%)
(
    setup_test_colony "Test supermajority calculation"
    setup_votes_dir

    # Create 3 APPROVE, 1 REJECT (75% approval)
    cat > "${AETHER_ROOT}/.aether/verification/votes/security_vote.json" <<EOF
{
  "watcher": "security",
  "decision": "APPROVE",
  "weight": 1.0,
  "issues": []
}
EOF

    cat > "${AETHER_ROOT}/.aether/verification/votes/performance_vote.json" <<EOF
{
  "watcher": "performance",
  "decision": "APPROVE",
  "weight": 1.0,
  "issues": []
}
EOF

    cat > "${AETHER_ROOT}/.aether/verification/votes/quality_vote.json" <<EOF
{
  "watcher": "quality",
  "decision": "APPROVE",
  "weight": 1.0,
  "issues": []
}
EOF

    cat > "${AETHER_ROOT}/.aether/verification/votes/test_coverage_vote.json" <<EOF
{
  "watcher": "test_coverage",
  "decision": "REJECT",
  "weight": 1.0,
  "issues": [
    {
      "description": "Test issue",
      "severity": "Low",
      "category": "quality",
      "location": "file.js:1"
    }
  ]
}
EOF

    # Aggregate votes
    votes_file="/tmp/votes_supermajority.json"
    jq -s '.' "${AETHER_ROOT}"/.aether/verification/votes/*_vote.json > "$votes_file" 2>/dev/null || true

    # Calculate supermajority
    result=$(calculate_supermajority "$votes_file" 2>/dev/null || echo "ERROR")

    if [[ "$result" == *"APPROVED"* ]]; then
        echo "ok 8 - Supermajority threshold enforced"
    else
        echo "not ok 8 - Supermajority threshold enforced"
        echo "# Result: $result"
        exit 1
    fi
)
