#!/bin/bash
# Test script for event filtering and pull-based delivery
# Usage: .aether/utils/test-event-filtering.sh

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "${SCRIPT_DIR}/event-bus.sh"

echo "=== Event Filtering and Delivery Test Suite ==="
echo

# Backup existing events.json and create fresh test instance
echo "1. Setting up fresh event bus for testing..."
if [ -f "$EVENTS_FILE" ]; then
    cp "$EVENTS_FILE" "${EVENTS_FILE}.backup.$$"
fi

# Create fresh event bus
rm -f "$EVENTS_FILE"
initialize_event_bus > /dev/null
echo "✓ Fresh event bus initialized"
echo

# Test 1: Create subscriptions with different patterns
echo "2. Creating test subscriptions..."
subscribe_to_events "verifier" "watcher" "phase_complete" '{"phase": 8}' > /dev/null
subscribe_to_events "logger" "architect" "error.*" '{}' > /dev/null
subscribe_to_events "task_monitor" "scout" "task_*" '{}' > /dev/null
echo "✓ Created 3 subscriptions"
echo

# Test 2: Publish test events
echo "3. Publishing test events..."
publish_event "phase_complete" "phase_8_complete" '{"phase": 8, "status": "success"}' "queen" > /dev/null
publish_event "error" "error_occurred" '{"error_code": 500, "message": "Database error"}' "worker" "builder" > /dev/null
publish_event "error" "error_critical" '{"error_code": 503, "severity": "critical"}' "worker" "builder" > /dev/null
publish_event "task_started" "task_started" '{"task_id": "123", "name": "build API"}' "executor" "builder" > /dev/null
publish_event "task_completed" "task_completed" '{"task_id": "123", "status": "success"}' "executor" "builder" > /dev/null
echo "✓ Published 5 events"
echo

# Test 3: Filter by topic pattern
echo "4. Testing topic pattern filtering..."
verifier_events=$(get_events_for_subscriber "verifier" "watcher")
verifier_count=$(echo "$verifier_events" | jq 'length')
echo "Verifier (phase_complete): $verifier_count events"
logger_events=$(get_events_for_subscriber "logger" "architect")
logger_count=$(echo "$logger_events" | jq 'length')
echo "Logger (error.*): $logger_count events"
task_monitor_events=$(get_events_for_subscriber "task_monitor" "scout")
task_monitor_count=$(echo "$task_monitor_events" | jq 'length')
echo "Task Monitor (task_*): $task_monitor_count events"
if [ "$verifier_count" -eq 1 ] && [ "$logger_count" -eq 2 ] && [ "$task_monitor_count" -eq 2 ]; then
    echo "✓ Topic pattern filtering works correctly"
else
    echo "✗ Topic pattern filtering failed"
    exit 1
fi
echo

# Test 4: Polling semantics (only new events since last delivery)
echo "5. Testing polling semantics..."
mark_events_delivered "verifier" "watcher" "$verifier_events" > /dev/null
# Publish new event with same phase to match filter criteria
publish_event "phase_complete" "phase_8_another" '{"phase": 8, "task": "verification"}' "queen" > /dev/null
# Get events again - should only see new event
new_verifier_events=$(get_events_for_subscriber "verifier" "watcher")
new_verifier_count=$(echo "$new_verifier_events" | jq 'length')
echo "Verifier events after marking delivered and publishing new: $new_verifier_count"
if [ "$new_verifier_count" -eq 1 ]; then
    echo "✓ Polling semantics work correctly (only new events since last delivery)"
else
    echo "✗ Polling semantics failed"
    exit 1
fi
echo

# Test 5: Filter criteria
echo "6. Testing filter criteria..."
subscribe_to_events "db_specialist" "builder" "spawn_request" '{"specialist_type": "database"}' > /dev/null
publish_event "spawn_request" "spawn_db" '{"specialist_type": "database", "reason": "capability gap"}' "route_setter" > /dev/null
publish_event "spawn_request" "spawn_web" '{"specialist_type": "web", "reason": "capability gap"}' "route_setter" > /dev/null
db_events=$(get_events_for_subscriber "db_specialist" "builder")
db_count=$(echo "$db_events" | jq 'length')
echo "DB specialist (spawn_request with specialist_type=database): $db_count events"
if [ "$db_count" -eq 1 ]; then
    echo "✓ Filter criteria work correctly"
else
    echo "✗ Filter criteria failed"
    exit 1
fi
echo

# Test 6: Empty result (no matching events)
echo "7. Testing empty result (no matching events)..."
subscribe_to_events "non_existent_subscriber" "watcher" "non_existent_topic" '{}' > /dev/null
no_events=$(get_events_for_subscriber "non_existent_subscriber" "watcher")
no_events_count=$(echo "$no_events" | jq 'length')
echo "Non-existent subscriber events: $no_events_count"
if [ "$no_events_count" -eq 0 ]; then
    echo "✓ Returns empty array when no matching events"
else
    echo "✗ Should return empty array"
    exit 1
fi
echo

# Test 7: Delivery tracking updates
echo "8. Testing delivery tracking..."
initial_delivery_count=$(jq -r '.subscriptions[] | select(.subscriber_id == "verifier") | .delivery_count' "$EVENTS_FILE")
mark_events_delivered "verifier" "watcher" "$new_verifier_events" > /dev/null
final_delivery_count=$(jq -r '.subscriptions[] | select(.subscriber_id == "verifier") | .delivery_count' "$EVENTS_FILE")
echo "Verifier delivery_count: $initial_delivery_count → $final_delivery_count"
if [ "$final_delivery_count" -gt "$initial_delivery_count" ]; then
    echo "✓ Delivery tracking updated correctly"
else
    echo "✗ Delivery tracking failed"
    exit 1
fi
echo

# Test 8: Metrics updates
echo "9. Testing metrics updates..."
total_delivered=$(jq -r '.metrics.total_delivered' "$EVENTS_FILE")
backlog_count=$(jq -r '.metrics.backlog_count' "$EVENTS_FILE")
echo "Metrics - total_delivered: $total_delivered, backlog_count: $backlog_count"
if [ "$total_delivered" -gt 0 ]; then
    echo "✓ Metrics updated correctly"
else
    echo "✗ Metrics not updated"
    exit 1
fi
echo

# Test 9: Non-blocking behavior
echo "10. Testing non-blocking behavior (returns immediately with no events)..."
time_start=$(date +%s)
get_events_for_subscriber "non_existent_subscriber" "watcher" > /dev/null
time_end=$(date +%s)
time_elapsed=$((time_end - time_start))
echo "Time elapsed: ${time_elapsed}s"
if [ "$time_elapsed" -lt 2 ]; then
    echo "✓ Non-blocking (returns immediately)"
else
    echo "✗ Blocking (should return immediately)"
    exit 1
fi
echo

echo "=== All Event Filtering Tests Passed ==="
echo

# Restore original events.json if backup exists
if [ -f "${EVENTS_FILE}.backup.$$" ]; then
    mv "${EVENTS_FILE}.backup.$$" "$EVENTS_FILE"
    echo "✓ Restored original events.json"
fi
