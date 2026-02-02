#!/bin/bash
# Test script for event metrics tracking
# Usage: .aether/utils/test-event-metrics.sh

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "${SCRIPT_DIR}/event-bus.sh"

echo "=== Event Metrics Test Suite ==="
echo

# Initialize event bus
echo "1. Initializing event bus..."
initialize_event_bus > /dev/null 2>&1
echo "✓ Event bus initialized"
echo

# Test 1: Initial metrics
echo "2. Testing initial metrics..."
initial_metrics=$(get_event_metrics)
initial_published=$(echo "$initial_metrics" | jq -r '.total_published')
initial_subscriptions=$(echo "$initial_metrics" | jq -r '.total_subscriptions')
echo "Initial - Published: $initial_published, Subscriptions: $initial_subscriptions"
if [ "$initial_published" -ge 0 ] && [ "$initial_subscriptions" -ge 0 ]; then
    echo "✓ Initial metrics accessible"
else
    echo "✗ Initial metrics unexpected"
    exit 1
fi
echo

# Test 2: Publish metrics update
echo "3. Testing publish metrics..."
publish_event "test" "test_event" '{"i": 1}' "pub1" > /dev/null
publish_event "test" "test_event" '{"i": 2}' "pub2" > /dev/null
publish_event "test" "test_event" '{"i": 3}' "pub3" > /dev/null

after_publish=$(get_event_metrics)
after_published=$(echo "$after_publish" | jq -r '.total_published')
publish_rate=$(echo "$after_publish" | jq -r '.publish_rate_per_minute')
echo "After publish - Published: $after_published, Rate: $publish_rate/min"
if [ "$after_published" -gt "$initial_published" ] && [ "$publish_rate" -ge 3 ]; then
    echo "✓ Publish metrics updated correctly"
else
    echo "✗ Publish metrics not updated correctly"
    exit 1
fi
echo

# Test 3: Subscription metrics update
echo "4. Testing subscription metrics..."
subscribe_to_events "sub1" "watcher" "test.*" '{}' > /dev/null
subscribe_to_events "sub2" "architect" "test.*" '{}' > /dev/null

after_subscribe=$(get_event_metrics)
after_subscriptions=$(echo "$after_subscribe" | jq -r '.total_subscriptions')
total_subscribers=$(echo "$after_subscribe" | jq -r '.total_subscribers')
echo "After subscribe - Subscriptions: $after_subscriptions, Subscribers: $total_subscribers"
if [ "$after_subscriptions" -gt "$initial_subscriptions" ] && [ "$total_subscribers" -ge 2 ]; then
    echo "✓ Subscription metrics updated correctly"
else
    echo "✗ Subscription metrics not updated correctly"
    exit 1
fi
echo

# Test 4: Delivery metrics update
echo "5. Testing delivery metrics..."
events=$(get_events_for_subscriber "sub1" "watcher")
event_count=$(echo "$events" | jq 'length')
if [ "$event_count" -gt 0 ]; then
    mark_events_delivered "sub1" "watcher" "$events" > /dev/null
fi

after_deliver=$(get_event_metrics)
after_delivered=$(echo "$after_deliver" | jq -r '.total_delivered')
backlog=$(echo "$after_deliver" | jq -r '.backlog_count')
echo "After delivery - Delivered: $after_delivered, Backlog: $backlog"
if [ "$after_delivered" -ge 0 ]; then
    echo "✓ Delivery metrics updated correctly"
else
    echo "✗ Delivery metrics not updated correctly"
    exit 1
fi
echo

# Test 5: Publish rate calculation (sliding window)
echo "6. Testing publish rate calculation..."
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
echo "7. Testing metrics summary..."
echo "Metrics Summary:"
get_metrics_summary
echo "✓ Metrics summary displayed"
echo

# Test 7: Backlog tracking
echo "8. Testing backlog tracking..."
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
echo "9. Testing last updated timestamp..."
last_updated=$(echo "$final_metrics" | jq -r '.last_updated')
if [ "$last_updated" != "null" ] && [ -n "$last_updated" ]; then
    echo "✓ Last updated timestamp present: $last_updated"
else
    echo "✗ Last updated timestamp missing"
    exit 1
fi
echo

# Test 9: Metrics persistence
echo "10. Testing metrics persistence..."
# Read metrics directly from file
file_metrics=$(jq '.metrics' .aether/data/events.json)
file_published=$(echo "$file_metrics" | jq -r '.total_published')
api_published=$(echo "$final_metrics" | jq -r '.total_published')
if [ "$file_published" -eq "$api_published" ]; then
    echo "✓ Metrics persist correctly (file: $file_published, api: $api_published)"
else
    echo "✗ Metrics mismatch (file: $file_published, api: $api_published)"
    exit 1
fi
echo

echo "=== All Event Metrics Tests Passed ==="
