#!/bin/bash
# Test script for event subscribe operation
# Usage: .aether/utils/test-event-subscribe.sh

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "${SCRIPT_DIR}/event-bus.sh"

echo "=== Event Subscribe Test Suite ==="
echo

# Initialize event bus
echo "1. Initializing event bus..."
initialize_event_bus
echo "Event bus initialized"
echo

# Clean up any existing subscriptions
jq '.subscriptions = [] | .metrics.total_subscriptions = 0 | .metrics.last_updated = null' "$EVENTS_FILE" > /tmp/events_test.json && mv /tmp/events_test.json "$EVENTS_FILE"
echo "Cleaned existing subscriptions"
echo

# Test 1: Basic subscribe
echo "2. Testing basic subscribe..."
sub_id=$(subscribe_to_events "verifier" "watcher" "phase_complete" '{"min_phase": 5}')
if [ -n "$sub_id" ] && [[ "$sub_id" == sub_* ]]; then
    echo "PASS: Created subscription: $sub_id"
else
    echo "FAIL: Failed to create subscription"
    exit 1
fi
echo

# Test 2: Multiple subscriptions
echo "3. Testing multiple subscriptions..."
subscribe_to_events "logger" "architect" "error.*" '{}' > /dev/null
subscribe_to_events "coordinator" "route_setter" "spawn_request" '{"specialist_type": "database"}' > /dev/null
subscribe_to_events "executor" "builder" "task_*" '{}' > /dev/null
sub_count=$(jq -r '.subscriptions | length' "$EVENTS_FILE")
if [ "$sub_count" -eq 4 ]; then
    echo "PASS: Created 3 additional subscriptions (total: $sub_count)"
else
    echo "FAIL: Expected 4 subscriptions, got $sub_count"
    exit 1
fi
echo

# Test 3: Verify subscription structure
echo "4. Verifying subscription structure..."
sub_json=$(jq -r '.subscriptions[-1]' "$EVENTS_FILE")
sub_has_id=$(echo "$sub_json" | jq -r '.id' | grep -c "sub_")
sub_has_pattern=$(echo "$sub_json" | jq -r '.topic_pattern' | grep -c "task_")
sub_has_filter=$(echo "$sub_json" | jq -r '.filter_criteria' | grep -c "{}")
if [ "$sub_has_id" -gt 0 ] && [ "$sub_has_pattern" -gt 0 ]; then
    echo "PASS: Subscription structure valid"
else
    echo "FAIL: Subscription structure invalid"
    exit 1
fi
echo

# Test 4: Verify topic subscriber_count
echo "5. Verifying topic subscriber_count..."
task_subscriber_count=$(jq -r '.topics["task_*"].subscriber_count' "$EVENTS_FILE")
if [ "$task_subscriber_count" -gt 0 ]; then
    echo "PASS: Topic subscriber_count updated (task_*: $task_subscriber_count)"
else
    echo "FAIL: Topic subscriber_count not updated"
    exit 1
fi
echo

# Test 5: Verify metrics
echo "6. Verifying metrics..."
total_subscriptions=$(jq -r '.metrics.total_subscriptions' "$EVENTS_FILE")
if [ "$total_subscriptions" -ge 4 ]; then
    echo "PASS: Metrics updated (total_subscriptions: $total_subscriptions)"
else
    echo "FAIL: Metrics not updated correctly (got $total_subscriptions, expected >= 4)"
    exit 1
fi
echo

# Test 6: Wildcard topic patterns
echo "7. Testing wildcard topic patterns..."
subscribe_to_events "error_collector" "architect" "error.*" '{}' > /dev/null
error_star_exists=$(jq -r '.topics["error.*"]' "$EVENTS_FILE")
if [ "$error_star_exists" != "null" ]; then
    echo "PASS: Wildcard topic pattern supported"
else
    echo "FAIL: Wildcard topic pattern not supported"
    exit 1
fi
echo

# Test 7: Filter criteria
echo "8. Testing filter criteria..."
subscribe_to_events "db_specialist" "builder" "spawn_request" '{"specialist_type": "database"}' > /dev/null
filter_valid=$(jq -r '.subscriptions[-1].filter_criteria.specialist_type' "$EVENTS_FILE")
if [ "$filter_valid" == "database" ]; then
    echo "PASS: Filter criteria stored correctly"
else
    echo "FAIL: Filter criteria not stored (got: $filter_valid)"
    exit 1
fi
echo

# Test 8: List subscriptions
echo "9. Testing list_subscriptions..."
all_subs=$(list_subscriptions | wc -l)
filtered_subs=$(list_subscriptions "verifier" | wc -l)
if [ "$all_subs" -gt 0 ] && [ "$filtered_subs" -ge 1 ]; then
    echo "PASS: All subscriptions: $all_subs, Filtered (verifier): $filtered_subs"
else
    echo "FAIL: list_subscriptions not working correctly"
    exit 1
fi
echo

# Test 9: Unsubscribe
echo "10. Testing unsubscribe..."
initial_count=$(jq -r '.subscriptions | length' "$EVENTS_FILE")
unsubscribe_from_events "$sub_id" > /dev/null
final_count=$(jq -r '.subscriptions | length' "$EVENTS_FILE")
if [ "$final_count" -lt "$initial_count" ]; then
    echo "PASS: Unsubscribe successful (count: $initial_count -> $final_count)"
else
    echo "FAIL: Unsubscribe failed (count: $initial_count -> $final_count)"
    exit 1
fi
echo

# Test 10: Error handling - missing required arguments
echo "11. Testing error handling (missing arguments)..."
if subscribe_to_events "" "watcher" "test" '{}' 2>/dev/null; then
    echo "FAIL: Should have failed with missing subscriber_id"
    exit 1
else
    echo "PASS: Correctly rejected missing subscriber_id"
fi
echo

# Test 11: Default filter parameter
echo "12. Testing default filter parameter..."
test_sub=$(subscribe_to_events "default_test" "watcher" "phase_complete")
if [ -n "$test_sub" ]; then
    filter_check=$(jq -r --arg sub "$test_sub" '.subscriptions[] | select(.id == $sub) | .filter_criteria' "$EVENTS_FILE")
    if [ "$filter_check" == "{}" ]; then
        echo "PASS: Default filter parameter set to {}"
    else
        echo "FAIL: Default filter not set correctly"
        exit 1
    fi
else
    echo "FAIL: Failed to create subscription with default filter"
    exit 1
fi
echo

echo "=== All Subscribe Tests Passed ==="
