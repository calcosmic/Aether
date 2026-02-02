#!/bin/bash
# Test script for event publish operation
# Usage: .aether/utils/test-event-publish.sh

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "${SCRIPT_DIR}/event-bus.sh"

echo "=== Event Publish Test Suite ==="
echo

# Initialize event bus
echo "1. Initializing event bus..."
initialize_event_bus
echo "✓ Event bus initialized"
echo

# Test 1: Basic publish
echo "2. Testing basic publish..."
event_id=$(publish_event "task_started" "task_started" '{"task_id": "test-01", "phase": 9}' "test_publisher")
if [ -n "$event_id" ]; then
    echo "✓ Published event: $event_id"
else
    echo "✗ Failed to publish event"
    exit 1
fi
echo

# Test 2: Multiple events
echo "3. Testing multiple events..."
publish_event "task_completed" "task_completed" '{"task_id": "test-02", "status": "success"}' "executor" "builder" > /dev/null
publish_event "error" "error_occurred" '{"error_code": 404, "message": "Not found"}' "worker" "scout" > /dev/null
publish_event "spawn_request" "spawn_specialist" '{"specialist_type": "database", "reason": "capability gap"}' "route_setter" > /dev/null
event_count=$(jq -r '.event_log | length' "$EVENTS_FILE")
echo "✓ Published 3 additional events (total: $event_count)"
echo

# Test 3: Verify event structure
echo "4. Verifying event structure..."
event_json=$(jq -r '.event_log[-1]' "$EVENTS_FILE")
event_has_id=$(echo "$event_json" | jq -r '.id' | grep -c "evt_")
event_has_topic=$(echo "$event_json" | jq -r '.topic' | grep -c "spawn_request")
event_has_metadata=$(echo "$event_json" | jq -r '.metadata.publisher' | grep -c "route_setter")
if [ "$event_has_id" -gt 0 ] && [ "$event_has_topic" -gt 0 ] && [ "$event_has_metadata" -gt 0 ]; then
    echo "✓ Event structure valid"
else
    echo "✗ Event structure invalid"
    exit 1
fi
echo

# Test 4: Verify metrics
echo "5. Verifying metrics..."
total_published=$(jq -r '.metrics.total_published' "$EVENTS_FILE")
backlog_count=$(jq -r '.metrics.backlog_count' "$EVENTS_FILE")
if [ "$total_published" -ge 4 ] && [ "$backlog_count" -ge 4 ]; then
    echo "✓ Metrics updated (published: $total_published, backlog: $backlog_count)"
else
    echo "✗ Metrics not updated correctly"
    exit 1
fi
echo

# Test 5: Dynamic topic creation
echo "6. Testing dynamic topic creation..."
publish_event "custom_topic" "custom_event" '{"test": "data"}' "test_publisher" > /dev/null
topic_exists=$(jq -r '.topics["custom_topic"]' "$EVENTS_FILE")
if [ "$topic_exists" != "null" ]; then
    echo "✓ Dynamic topic created"
else
    echo "✗ Dynamic topic not created"
    exit 1
fi
echo

# Test 6: Error handling - invalid JSON
echo "7. Testing error handling (invalid JSON)..."
if publish_event "test_topic" "test_type" 'invalid json' "test_publisher" 2>/dev/null; then
    echo "✗ Should have failed with invalid JSON"
    exit 1
else
    echo "✓ Correctly rejected invalid JSON"
fi
echo

# Test 7: Test without caste (optional parameter)
echo "8. Testing publish without caste parameter..."
event_id_no_caste=$(publish_event "no_caste_topic" "no_caste_event" '{"test": "data"}' "test_publisher")
if [ -n "$event_id_no_caste" ]; then
    caste_value=$(jq -r '.event_log[] | select(.id == "'"$event_id_no_caste"'") | .metadata.publisher_caste' "$EVENTS_FILE")
    if [ "$caste_value" = "null" ]; then
        echo "✓ Publish without caste works (publisher_caste is null)"
    else
        echo "✗ publisher_caste should be null when not provided"
        exit 1
    fi
else
    echo "✗ Failed to publish event without caste"
    exit 1
fi
echo

# Test 8: Verify unique IDs
echo "9. Testing unique event IDs..."
publish_event "unique_test" "test1" '{"i": 1}' "test" > /dev/null
publish_event "unique_test" "test2" '{"i": 2}' "test" > /dev/null
publish_event "unique_test" "test3" '{"i": 3}' "test" > /dev/null
id_count=$(jq -r '[.event_log[] | select(.topic == "unique_test") | .id] | unique | length' "$EVENTS_FILE")
if [ "$id_count" -eq 3 ]; then
    echo "✓ All event IDs are unique"
else
    echo "✗ Event IDs are not unique"
    exit 1
fi
echo

# Test 9: Ring buffer trim (optional - would require publishing 1000+ events)
echo "10. Ring buffer trim test (skipped - requires 1000+ events)"
echo "    Run: for i in {1..1001}; do publish_event \"test\" \"test\" '{\"i\":'$i'}' \"test\" > /dev/null; done"
echo

echo "=== All Publish Tests Passed ==="
