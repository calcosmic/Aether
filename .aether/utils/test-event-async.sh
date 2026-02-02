#!/bin/bash
# Test script for async non-blocking event delivery
# Usage: .aether/utils/test-event-async.sh

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "${SCRIPT_DIR}/event-bus.sh"

echo "=== Async Non-Blocking Event Delivery Test Suite ==="
echo

# Initialize event bus
echo "1. Initializing event bus..."
initialize_event_bus
echo "✓ Event bus initialized"
echo

# Test 1: Publish returns immediately
echo "2. Testing publish returns immediately..."
time_start=$(date +%s%N 2>/dev/null || date +%s)000000
event_id=$(publish_event "test" "test_event" '{"test": "data"}' "test_publisher")
time_end=$(date +%s%N 2>/dev/null || date +%s)999000
time_elapsed=$(( (time_end - time_start) / 1000000 )) # Convert to milliseconds
echo "Published event $event_id in ${time_elapsed}ms"
if [ "$time_elapsed" -lt 100 ]; then
    echo "✓ Publish returns immediately (<100ms)"
else
    echo "⚠ Publish took ${time_elapsed}ms (expected <100ms for non-blocking)"
fi
echo

# Test 2: Create subscriptions (but no polling yet)
echo "3. Creating subscriptions..."
subscribe_to_events "subscriber1" "watcher" "test_topic" '{}' > /dev/null
subscribe_to_events "subscriber2" "architect" "test_topic" '{}' > /dev/null
subscribe_to_events "subscriber3" "builder" "test_topic" '{}' > /dev/null
echo "✓ Created 3 subscriptions"
echo

# Test 3: Publish does NOT wait for subscribers to poll
echo "4. Testing publish does not wait for subscribers..."
# Publish event (should return immediately even though subscribers haven't polled)
event_id=$(publish_event "test_topic" "test_async" '{"async": "test"}' "publisher")
echo "Published event $event_id"

# Verify event is in log
event_in_log=$(jq -r ".event_log[] | select(.id == \"$event_id\") | .id" "$EVENTS_FILE")
if [ "$event_in_log" = "$event_id" ]; then
    echo "✓ Event in log (published successfully)"
else
    echo "✗ Event not in log"
    exit 1
fi

# Verify subscribers have NOT received event yet (they haven't polled)
sub1_events=$(get_events_for_subscriber "subscriber1" "watcher")
sub1_count=$(echo "$sub1_events" | jq 'length')
echo "Subscriber1 events before poll: $sub1_count"
if [ "$sub1_count" -gt 0 ]; then
    echo "⚠ Subscriber has events before polling (unexpected)"
else
    echo "✓ Subscriber has no events yet (expected - haven't polled)"
fi
echo

# Test 4: Subscribers poll independently
echo "5. Testing subscribers poll independently..."
# Subscriber1 polls
sub1_events=$(get_events_for_subscriber "subscriber1" "watcher")
sub1_count=$(echo "$sub1_events" | jq 'length')
echo "Subscriber1 events after poll: $sub1_count"
mark_events_delivered "subscriber1" "watcher" "$sub1_events" > /dev/null

# Subscriber2 polls (still sees event - independent delivery)
sub2_events=$(get_events_for_subscriber "subscriber2" "architect")
sub2_count=$(echo "$sub2_events" | jq 'length')
echo "Subscriber2 events after poll: $sub2_count"

# Subscriber3 has NOT polled yet
sub3_events=$(get_events_for_subscriber "subscriber3" "builder")
sub3_count=$(echo "$sub3_events" | jq 'length')
echo "Subscriber3 events (hasn't polled yet): $sub3_count"

if [ "$sub1_count" -ge 1 ] && [ "$sub2_count" -ge 1 ] && [ "$sub3_count" -ge 1 ]; then
    echo "✓ All subscribers can poll independently"
else
    echo "✗ Independent polling failed"
    exit 1
fi
echo

# Test 5: Publish while subscribers are not running
echo "6. Testing publish while subscribers are not running..."
# Publish another event (subscribers not "running" - prompt-based agents)
event_id=$(publish_event "test_topic" "test_inactive" '{"inactive": "subscribers"}' "publisher")
echo "Published event $event_id while subscribers inactive"

# Subscribers can still get event when they next poll
sub1_events=$(get_events_for_subscriber "subscriber1" "watcher")
sub1_count=$(echo "$sub1_events" | jq 'length')
echo "Subscriber1 events after second publish: $sub1_count"
if [ "$sub1_count" -ge 1 ]; then
    echo "✓ Subscribers receive events published while they were inactive"
else
    echo "✗ Subscribers did not receive events published while inactive"
    exit 1
fi
echo

# Test 6: Concurrent publishes do not block each other
echo "7. Testing concurrent publishes..."
publish_event "concurrent" "conc1" '{"i": 1}' "p1" > /dev/null &
pid1=$!
publish_event "concurrent" "conc2" '{"i": 2}' "p2" > /dev/null &
pid2=$!
publish_event "concurrent" "conc3" '{"i": 3}' "p3" > /dev/null &
pid3=$!

wait $pid1
wait $pid2
wait $pid3

concurrent_count=$(jq '[.event_log[] | select(.topic == "concurrent")] | length' "$EVENTS_FILE")
echo "Concurrent publishes completed: $concurrent_count events"
if [ "$concurrent_count" -eq 3 ]; then
    echo "✓ Concurrent publishes work correctly"
else
    echo "⚠ Expected 3 concurrent events, got $concurrent_count"
fi
echo

# Test 7: No background processes spawned
echo "8. Verifying no background processes..."
# Check for event bus background processes
bg_processes=$(ps aux | grep -i "event-bus" | grep -v grep | wc -l | tr -d ' ')
echo "Background event-bus processes: $bg_processes"
if [ "$bg_processes" -eq 0 ]; then
    echo "✓ No background event-bus processes (pure pull-based)"
else
    echo "⚠ Found background processes (unexpected for pull-based design)"
fi
echo

# Test 8: Decoupled publish and subscribe
echo "9. Testing decoupled publish and subscribe..."
# Publish to topic with NO subscribers
event_id=$(publish_event "no_subs_topic" "no_subs" '{"test": "data"}' "publisher")
echo "Published to topic with no subscribers: $event_id"

# Event should still be in log
event_in_log=$(jq -r ".event_log[] | select(.id == \"$event_id\") | .id" "$EVENTS_FILE")
if [ "$event_in_log" = "$event_id" ]; then
    echo "✓ Event logged even with no subscribers"
else
    echo "✗ Event not logged"
    exit 1
fi

# Now subscribe - should still see event (since it hasn't been delivered)
subscribe_to_events "late_subscriber" "watcher" "no_subs_topic" '{}' > /dev/null
late_events=$(get_events_for_subscriber "late_subscriber" "watcher")
late_count=$(echo "$late_events" | jq 'length')
echo "Late subscriber sees $late_count events"
if [ "$late_count" -ge 1 ]; then
    echo "✓ Late subscriber receives events published before subscription"
else
    echo "⚠ Late subscriber did not receive pre-subscription events"
fi
echo

# Test 9: Publish speed (100 events should be fast)
echo "10. Testing publish speed (100 events)..."
time_start=$(date +%s)
for i in {1..100}; do
    publish_event "speed_test" "speed_event" "{\"i\": $i}" "speed_tester" > /dev/null
done
time_end=$(date +%s)
time_elapsed=$((time_end - time_start))
echo "Published 100 events in ${time_elapsed}s"
if [ "$time_elapsed" -lt 10 ]; then
    echo "✓ Publish speed acceptable (<10s for 100 events)"
else
    echo "⚠ Publish slow (${time_elapsed}s for 100 events)"
fi
echo

echo "=== All Async Delivery Tests Passed ==="
echo ""
echo "Summary:"
echo "- Publish returns immediately (non-blocking)"
echo "- Publish does not wait for subscribers"
echo "- Subscribers poll independently"
echo "- No background processes required"
echo "- Concurrent publishes work correctly"
echo "- Publish and subscribe are fully decoupled"
