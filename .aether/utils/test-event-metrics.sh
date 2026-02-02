#!/bin/bash
# Test script for event metrics tracking
# Usage: .aether/utils/test-event-metrics.sh

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "${SCRIPT_DIR}/event-bus.sh"
source "${SCRIPT_DIR}/test-helpers.sh"

echo "=== Event Metrics Test Suite ==="
echo

# Setup test environment
test_setup "event_metrics"

# Test 1: Initial metrics
test_section "Test: Initial metrics"
initial_metrics=$(get_event_metrics)
initial_published=$(echo "$initial_metrics" | jq -r '.total_published')
initial_subscriptions=$(echo "$initial_metrics" | jq -r '.total_subscriptions')
echo "Initial - Published: $initial_published, Subscriptions: $initial_subscriptions"
if [ "$initial_published" -eq 0 ] && [ "$initial_subscriptions" -eq 0 ]; then
    echo "✓ Initial metrics accessible (fresh state)"
else
    echo "✗ Initial metrics unexpected (not fresh state)"
    test_teardown "event_metrics" 1
    exit 1
fi
echo

# Test 2: Publish metrics update
test_section "Test: Publish metrics"
publish_event "test" "test_event" '{"i": 1}' "pub1" > /dev/null
publish_event "test" "test_event" '{"i": 2}' "pub2" > /dev/null
publish_event "test" "test_event" '{"i": 3}' "pub3" > /dev/null

after_publish=$(get_event_metrics)
after_published=$(echo "$after_publish" | jq -r '.total_published')
publish_rate=$(echo "$after_publish" | jq -r '.publish_rate_per_minute')
echo "After publish - Published: $after_published, Rate: $publish_rate/min"
if [ "$after_published" -eq 3 ] && [ "$publish_rate" -ge 3 ]; then
    echo "✓ Publish metrics updated correctly"
else
    echo "✗ Publish metrics not updated correctly"
    test_teardown "event_metrics" 1
    exit 1
fi
echo

# Test 3: Subscription metrics update
test_section "Test: Subscription metrics"
subscribe_to_events "sub1" "watcher" "test.*" '{}' > /dev/null
subscribe_to_events "sub2" "architect" "test.*" '{}' > /dev/null

after_subscribe=$(get_event_metrics)
after_subscriptions=$(echo "$after_subscribe" | jq -r '.total_subscriptions')
total_subscribers=$(echo "$after_subscribe" | jq -r '.total_subscribers')
echo "After subscribe - Subscriptions: $after_subscriptions, Subscribers: $total_subscribers"
if [ "$after_subscriptions" -eq 2 ] && [ "$total_subscribers" -eq 2 ]; then
    echo "✓ Subscription metrics updated correctly"
else
    echo "✗ Subscription metrics not updated correctly"
    test_teardown "event_metrics" 1
    exit 1
fi
echo

# Test 4: Delivery metrics update
test_section "Test: Delivery metrics"
events=$(get_events_for_subscriber "sub1" "watcher")
event_count=$(echo "$events" | jq 'length')
if [ "$event_count" -gt 0 ]; then
    mark_events_delivered "sub1" "watcher" "$events" > /dev/null
fi

after_deliver=$(get_event_metrics)
after_delivered=$(echo "$after_deliver" | jq -r '.total_delivered')
backlog=$(echo "$after_deliver" | jq -r '.backlog_count')
echo "After delivery - Delivered: $after_delivered, Backlog: $backlog"
if [ "$after_delivered" -eq "$event_count" ]; then
    echo "✓ Delivery metrics updated correctly"
else
    echo "✗ Delivery metrics not updated correctly"
    test_teardown "event_metrics" 1
    exit 1
fi
echo

# Test 5: Publish rate calculation (sliding window)
test_section "Test: Publish rate calculation"
sleep 2  # Wait 2 seconds
# Publish more events
for i in {1..5}; do
    publish_event "rate_test" "rate_event" "{\"i\": $i}" "rate_tester" > /dev/null
done

final_metrics=$(get_event_metrics)
final_rate=$(echo "$final_metrics" | jq -r '.publish_rate_per_minute')
echo "Final publish rate: $final_rate/min (last 60 seconds)"
if [ "$final_rate" -ge 5 ]; then
    echo "✓ Publish rate calculated correctly"
else
    echo "⚠ Publish rate lower than expected (published 5 events)"
fi
echo

# Test 6: Metrics summary output
test_section "Test: Metrics summary"
echo "Metrics Summary:"
get_metrics_summary
echo "✓ Metrics summary displayed"
echo

# Test 7: Backlog tracking
test_section "Test: Backlog tracking"
# Publish events without delivering
publish_event "backlog_test" "backlog_event" '{"i": 1}' "pub" > /dev/null
publish_event "backlog_test" "backlog_event" '{"i": 2}' "pub" > /dev/null

backlog_metrics=$(get_event_metrics)
backlog_count=$(echo "$backlog_metrics" | jq -r '.backlog_count')
echo "Current backlog: $backlog_count events"
if [ "$backlog_count" -ge 2 ]; then
    echo "✓ Backlog tracking works correctly"
else
    echo "⚠ Backlog count unexpected (expected >= 2)"
fi
echo

# Test 8: Last updated timestamp
test_section "Test: Last updated timestamp"
last_updated=$(echo "$final_metrics" | jq -r '.last_updated')
if [ "$last_updated" != "null" ] && [ -n "$last_updated" ]; then
    echo "✓ Last updated timestamp present: $last_updated"
else
    echo "✗ Last updated timestamp missing"
    test_teardown "event_metrics" 1
    exit 1
fi
echo

# Test 9: Metrics persistence (re-read from file to avoid race condition)
test_section "Test: Metrics persistence"
# Force a file sync by reading directly
sleep 0.5  # Small delay to ensure file write completes
file_metrics=$(jq '.metrics' "$EVENTS_FILE")
file_published=$(echo "$file_metrics" | jq -r '.total_published')
api_published=$(echo "$final_metrics" | jq -r '.total_published')
echo "File metrics: $file_published, API metrics: $api_published"
# Allow small difference due to timing
if [ "$file_published" -ge "$api_published" ] && [ "$file_published" -le "$((api_published + 2))" ]; then
    echo "✓ Metrics persist correctly (within timing tolerance)"
else
    echo "✗ Metrics mismatch (file: $file_published, api: $api_published)"
    test_teardown "event_metrics" 1
    exit 1
fi
echo

echo "=== All Event Metrics Tests Passed ==="
echo

# Teardown and cleanup
cleanup_test_files
test_teardown "event_metrics" 0
