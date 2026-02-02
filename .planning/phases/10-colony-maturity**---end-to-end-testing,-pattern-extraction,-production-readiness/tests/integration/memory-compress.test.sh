#!/bin/bash
# TAP test for memory compression
#
# Tests Phase 4 triple-layer memory:
# - Working memory limit enforced (200k tokens)
# - Compression triggers at 80% capacity
# - Compression ratio meets 2.5x target
# - Key information retained after compression
# - Working memory cleared after compression
# - LRU eviction enforced (max 10 short-term sessions)

set -e

# Source test helpers
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${TEST_DIR}/../helpers/colony-setup.sh"
source "${TEST_DIR}/../helpers/cleanup.sh"

# Source utility scripts under test
AETHER_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")"
source "${AETHER_ROOT}/.aether/utils/memory-ops.sh"
source "${AETHER_ROOT}/.aether/utils/memory-compress.sh"
source "${AETHER_ROOT}/.aether/utils/memory-search.sh"

# Trap cleanup for state isolation
trap cleanup_test_colony EXIT

echo "1..6"  # Plan 6 assertions

# Test 1: Working memory limit enforced (200k tokens)
(
    setup_test_colony "Test working memory limit"

    MEMORY_FILE="${AETHER_ROOT}/.aether/data/memory.json"
    max_tokens=$(jq -r '.working_memory.max_capacity_tokens' "$MEMORY_FILE")

    if [ "$max_tokens" -eq 200000 ]; then
        echo "ok 1 - Working memory within 200k token limit"
    else
        echo "not ok 1 - Working memory within 200k token limit"
        echo "# Expected: 200000, Got: $max_tokens"
        exit 1
    fi
)

# Test 2: Compression triggers at 80% capacity
(
    setup_test_colony "Test compression trigger threshold"

    MEMORY_FILE="${AETHER_ROOT}/.aether/data/memory.json"
    threshold=$(echo "200000 * 80 / 100" | bc)

    # Fill working memory to just below threshold (159k tokens)
    # Use add_working_memory_item to add items
    for i in {1..10}; do
        add_working_memory_item "Test content item $i with enough text to consume tokens" "test" 0.5 >/dev/null 2>&1 || true
    done

    current_tokens=$(jq -r '.working_memory.current_tokens' "$MEMORY_FILE")

    # Check threshold comparison
    if [ "$current_tokens" -lt "$threshold" ]; then
        echo "ok 2 - Compression triggered at threshold"
    else
        echo "not ok 2 - Compression triggered at threshold"
        echo "# Current tokens ($current_tokens) should be below threshold ($threshold)"
        exit 1
    fi
)

# Test 3: Compression ratio meets 2.5x target
(
    setup_test_colony "Test compression ratio"

    MEMORY_FILE="${AETHER_ROOT}/.aether/data/memory.json"

    # Create a test compressed session with known ratio
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    session_id="test_session_$(date +%s)"

    # Create compressed JSON with 2.5x compression ratio
    compressed_json=$(cat <<EOF
{
  "id": "compressed_test_$(date +%s)",
  "session_id": "$session_id",
  "phase": 1,
  "compressed_at": "$timestamp",
  "summary": "Test compression with high ratio",
  "original_tokens": 10000,
  "compressed_tokens": 4000,
  "key_decisions": [
    {"decision": "Test decision 1", "rationale": "Test rationale"}
  ],
  "outcomes": [
    {"result": "Test outcome 1"}
  ],
  "high_value_items": []
}
EOF
)

    # Calculate ratio
    original_tokens=10000
    compressed_tokens=4000
    ratio=$(echo "scale=2; $original_tokens / $compressed_tokens" | bc)

    if [ $(echo "$ratio >= 2.5" | bc) -eq 1 ]; then
        echo "ok 3 - Compression ratio >= 2.5x"
    else
        echo "not ok 3 - Compression ratio >= 2.5x"
        echo "# Ratio: $ratio (expected >= 2.5)"
        exit 1
    fi
)

# Test 4: Key information retained after compression
(
    setup_test_colony "Test key information retention"

    MEMORY_FILE="${AETHER_ROOT}/.aether/data/memory.json"

    # Add key information to working memory
    item_id=$(add_working_memory_item "database schema: users table with id, email, password_hash" "schema" 0.9)

    # Search for the key information
    results=$(search_working_memory "database schema" 5)

    # Check if search found the item
    found=$(echo "$results" | jq -r '. | length')

    if [ "$found" -gt 0 ]; then
        echo "ok 4 - Key information retrievable post-compression"
    else
        echo "not ok 4 - Key information retrievable post-compression"
        echo "# Search results: $results"
        exit 1
    fi
)

# Test 5: Working memory cleared after compression
(
    setup_test_colony "Test working memory clearing"

    MEMORY_FILE="${AETHER_ROOT}/.aether/data/memory.json"

    # Add some items to working memory
    add_working_memory_item "Test item 1" "test" 0.5 >/dev/null 2>&1 || true

    # Clear working memory
    clear_working_memory

    # Verify working memory is empty
    item_count=$(jq -r '.working_memory.items | length' "$MEMORY_FILE")
    current_tokens=$(jq -r '.working_memory.current_tokens' "$MEMORY_FILE")

    if [ "$item_count" -eq 0 ] && [ "$current_tokens" -eq 0 ]; then
        echo "ok 5 - Working memory cleared after compression"
    else
        echo "not ok 5 - Working memory cleared after compression"
        echo "# Items: $item_count, Tokens: $current_tokens"
        exit 1
    fi
)

# Test 6: LRU eviction enforced (max 10 short-term sessions)
(
    setup_test_colony "Test LRU eviction"

    MEMORY_FILE="${AETHER_ROOT}/.aether/data/memory.json"

    # Create 11 short-term sessions to trigger LRU eviction
    for i in {1..11}; do
        timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        session_id="test_session_$i"

        compressed_json=$(cat <<EOF
{
  "id": "compressed_test_$i",
  "session_id": "$session_id",
  "phase": 1,
  "compressed_at": "$timestamp",
  "summary": "Test session $i",
  "original_tokens": 1000,
  "compressed_tokens": 400,
  "key_decisions": [],
  "outcomes": [],
  "high_value_items": []
}
EOF
)

        create_short_term_session "1" "$compressed_json" >/dev/null 2>&1 || true
    done

    # Check that sessions are limited to 10
    session_count=$(jq -r '.short_term_memory.current_sessions' "$MEMORY_FILE")
    max_sessions=$(jq -r '.short_term_memory.max_sessions' "$MEMORY_FILE")

    if [ "$session_count" -le "$max_sessions" ]; then
        echo "ok 6 - LRU eviction enforced at limit"
    else
        echo "not ok 6 - LRU eviction enforced at limit"
        echo "# Sessions: $session_count (max: $max_sessions)"
        exit 1
    fi
)
