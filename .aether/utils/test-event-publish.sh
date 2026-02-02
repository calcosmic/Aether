#!/bin/bash
# Test script for event publish operation
# Usage: .aether/utils/test-event-publish.sh

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "${SCRIPT_DIR}/event-bus.sh"
source "${SCRIPT_DIR}/test-helpers.sh"

echo "=== Event Publish Test Suite ==="
echo

# Setup test environment
test_setup "event_publish"

# Test 1: Basic publish
test_section "Test: Basic publish"
event_id=$(publish_event "task_started" "task_started" '{"task_id": "test-01", "phase": 9}' "test_publisher")
if [ -n "$event_id" ]; then
    echo "✓ Published event: $event_id"
else
    echo "✗ Failed to publish event"
    test_teardown "event_publish" 1
    exit 1
fi
echo

# Test 2: Multiple events
test_section "Test: Multiple events"
publish_event "task_completed" "task_completed" '{"task_id": "test-02", "status": "success"}' "executor" "builder" > /dev/null
publish_event "error" "error_occurred" '{"error_code": 404, "message": "Not found"}' "worker" "scout" > /dev/null
publish_event "spawn_request" "spawn_specialist" '{"specialist_type": "database", "reason": "capability gap"}' "route_setter" > /dev/null
event_count=$(jq -r '.event_log | length' "$EVENTS_FILE")
if [ "$event_count" -eq 4 ]; then
    echo "✓ Published 3 additional events (total: $event_count)"
else
    echo "✗ Event count unexpected (expected 4, got $event_count)"
    test_teardown "event_publish" 1
    exit 1
fi
echo

# Test 3: Verify event structure
test_section "Test: Event structure"
event_json=$(jq -r '.event_log[-1]' "$EVENTS_FILE")
event_has_id=$(echo "$event_json" | jq -r '.id' | grep -c "evt_")
event_has_topic=$(echo "$event_json" | jq -r '.topic' | grep -c "spawn_request")
event_has_metadata=$(echo "$event_json" | jq -r '.metadata.publisher' | grep -c "route_setter")
if [ "$event_has_id" -gt 0 ] && [ "$event_has_topic" -gt 0 ] && [ "$event_has_metadata" -gt 0 ]; then
    echo "✓ Event structure valid"
else
    echo "✗ Event structure invalid"
    test_teardown "event_publish" 1
    exit 1
fi
echo

# Test 4: Verify metrics
test_section "Test: Metrics update"
total_published=$(jq -r '.metrics.total_published' "$EVENTS_FILE")
backlog=$(jq -r '.metrics.backlog_count' "$EVENTS_FILE")
if [ "$total_published" -eq 4 ] && [ "$backlog" -eq 4 ]; then
    echo "✓ Metrics updated (published: $total_published, backlog: $backlog)"
else
    echo "✗ Metrics not updated correctly"
    test_teardown "event_publish" 1
    exit 1
fi
echo

# Test 5: Dynamic topic creation
test_section "Test: Dynamic topic creation"
publish_event "custom_topic" "custom_event" '{"test": "data"}' "test_publisher" > /dev/null
topic_exists=$(jq -r '.topics["custom_topic"]' "$EVENTS_FILE")
if [ "$topic_exists" != "null" ]; then
    echo "✓ Dynamic topic created"
else
    echo "✗ Dynamic topic not created"
    test_teardown "event_publish" 1
    exit 1
fi
echo

# Test 6: Error handling - invalid JSON
test_section "Test: Invalid JSON rejection"
if publish_event "test_topic" "test_type" 'invalid json' "test_publisher" 2>/dev/null; then
    echo "✗ Should have failed with invalid JSON"
    test_teardown "event_publish" 1
    exit 1
else
    echo "✓ Correctly rejected invalid JSON"
fi
echo

# Test 7: Test without caste (optional parameter)
test_section "Test: Publish without caste"
event_id_no_caste=$(publish_event "no_caste_topic" "no_caste_event" '{"test": "data"}' "test_publisher")
if [ -n "$event_id_no_caste" ]; then
    caste_value=$(jq -r '.event_log[] | select(.id == "'"$event_id_no_caste"'") | .metadata.publisher_caste' "$EVENTS_FILE")
    if [ "$caste_value" = "null" ]; then
        echo "✓ Publish without caste works (publisher_caste is null)"
    else
        echo "✗ publisher_caste should be null when not provided"
        test_teardown "event_publish" 1
        exit 1
    fi
else
    echo "✗ Failed to publish event without caste"
    test_teardown "event_publish" 1
    exit 1
fi
echo

# Test 8: Verify unique IDs
test_section "Test: Unique event IDs"
publish_event "unique_test" "test1" '{"i": 1}' "test" > /dev/null
publish_event "unique_test" "test2" '{"i": 2}' "test" > /dev/null
publish_event "unique_test" "test3" '{"i": 3}' "test" > /dev/null
id_count=$(jq -r '[.event_log[] | select(.topic == "unique_test") | .id] | unique | length' "$EVENTS_FILE")
if [ "$id_count" -eq 3 ]; then
    echo "✓ All event IDs are unique"
else
    echo "✗ Event IDs are not unique (expected 3 unique IDs, got $id_count)"
    test_teardown "event_publish" 1
    exit 1
fi
echo

# Test 9: Ring buffer trim (optional - would require publishing 1000+ events)
echo "10. Ring buffer trim test (skipped - requires 1000+ events)"
echo "    Run: for i in {1..1001}; do publish_event \"test\" \"test\" '{\"i\":'$i'}' \"test\" > /dev/null; done"
echo

echo "=== All Publish Tests Passed ==="
echo

# Teardown and cleanup
cleanup_test_files
test_teardown "event_publish" 0
