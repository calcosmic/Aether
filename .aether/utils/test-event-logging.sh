#!/bin/bash
# Test script for event logging and cleanup
# Usage: .aether/utils/test-event-logging.sh

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "${SCRIPT_DIR}/event-bus.sh"

echo "=== Event Logging and Cleanup Test Suite ==="
echo

# Initialize event bus
echo "1. Initializing event bus..."
initialize_event_bus
echo "✓ Event bus initialized"
echo

# Test 1: Publish test events
echo "2. Publishing test events..."
for i in {1..20}; do
    publish_event "test_topic" "test_event" "{\"iteration\": $i}" "test_publisher" > /dev/null
done
publish_event "error" "error_1" '{"code": 404}' "worker" > /dev/null
publish_event "error" "error_2" '{"code": 500}' "worker" > /dev/null
publish_event "phase_complete" "phase_9" '{"phase": 9}' "queen" > /dev/null
total_events=$(jq -r '.event_log | length' "$EVENTS_FILE")
echo "✓ Published 23 events (total: $total_events)"
echo

# Test 2: Get event history
echo "3. Testing get_event_history..."
all_events=$(get_event_history | jq 'length')
echo "All events: $all_events"
error_events=$(get_event_history "error.*" | jq 'length')
echo "Error events: $error_events"
last_5_events=$(get_event_history "" "5" | jq 'length')
echo "Last 5 events: $last_5_events"
if [ "$all_events" -ge 23 ] && [ "$error_events" -eq 2 ] && [ "$last_5_events" -eq 5 ]; then
    echo "✓ Event history queries work correctly"
else
    echo "✗ Event history queries failed"
    exit 1
fi
echo

# Test 3: Get event statistics
echo "4. Testing get_event_stats..."
stats=$(get_event_stats)
total_in_stats=$(echo "$stats" | jq -r '.total_events')
echo "Total events in stats: $total_in_stats"
topics_count=$(echo "$stats" | jq -r '.topics | length')
echo "Unique topics: $topics_count"
if [ "$total_in_stats" -ge 23 ] && [ "$topics_count" -ge 3 ]; then
    echo "✓ Event statistics computed correctly"
else
    echo "✗ Event statistics failed"
    exit 1
fi
echo

# Test 4: Export event log (JSON format)
echo "5. Testing export_event_log (JSON format)..."
export_event_log "/tmp/test_events.json" "json"
if [ -f "/tmp/test_events.json" ]; then
    exported_count=$(jq 'length' /tmp/test_events.json)
    echo "✓ Exported $exported_count events to /tmp/test_events.json"
else
    echo "✗ Export failed"
    exit 1
fi
echo

# Test 5: Export event log (text format)
echo "6. Testing export_event_log (text format)..."
export_event_log "/tmp/test_events.txt" "text"
if [ -f "/tmp/test_events.txt" ]; then
    line_count=$(wc -l < /tmp/test_events.txt)
    echo "✓ Exported text log with $line_count lines to /tmp/test_events.txt"
    echo "First 10 lines:"
    head -10 /tmp/test_events.txt
else
    echo "✗ Export failed"
    exit 1
fi
echo

# Test 6: Export with topic filter
echo "7. Testing export with topic filter..."
export_event_log "/tmp/error_events.txt" "text" "error.*"
error_export_count=$(grep -c "\[error\]" /tmp/error_events.txt || echo 0)
echo "✓ Exported $error_export_count error events to /tmp/error_events.txt"
echo

# Test 7: Ring buffer trim (simulate by reducing max_event_log_size)
echo "8. Testing ring buffer trim..."
# Temporarily reduce max size to force trim
jq '.config.max_event_log_size = 15' "$EVENTS_FILE" > /tmp/events_trim_test.tmp
atomic_write_from_file "$EVENTS_FILE" /tmp/events_trim_test.tmp > /dev/null
trim_event_log
events_after_trim=$(jq -r '.event_log | length' "$EVENTS_FILE")
echo "Events after trim (max=15): $events_after_trim"
# Restore original max size
jq '.config.max_event_log_size = 1000' "$EVENTS_FILE" > /tmp/events_restore.tmp
atomic_write_from_file "$EVENTS_FILE" /tmp/events_restore.tmp > /dev/null
if [ "$events_after_trim" -le 15 ]; then
    echo "✓ Ring buffer trim works correctly"
else
    echo "✗ Ring buffer trim failed"
    exit 1
fi
echo

# Test 8: Time-based cleanup (simulate by setting retention to 0 hours)
echo "9. Testing time-based cleanup..."
# First, get count before cleanup
before_cleanup=$(jq -r '.event_log | length' "$EVENTS_FILE")
# Run cleanup with 0 hours (should remove all events)
cleanup_old_events 0 2>&1 | grep "Cleaning up"
after_cleanup=$(jq -r '.event_log | length' "$EVENTS_FILE")
echo "Events before cleanup: $before_cleanup, after: $after_cleanup"
if [ "$after_cleanup" -eq 0 ]; then
    echo "✓ Time-based cleanup works correctly"
else
    echo "✗ Time-based cleanup failed"
    exit 1
fi
echo

# Test 9: Query with since_timestamp
echo "10. Testing query with since_timestamp..."
publish_event "test" "new_event" '{"timestamp": "now"}' "test" > /dev/null
now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
one_hour_ago=$(date -v-1H -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -d "1 hour ago" -u +"%Y-%m-%dT%H:%M:%SZ")
recent_events=$(get_event_history "" "" "$one_hour_ago" | jq 'length')
echo "Events since 1 hour ago: $recent_events"
if [ "$recent_events" -ge 1 ]; then
    echo "✓ Since timestamp filter works correctly"
else
    echo "✗ Since timestamp filter failed"
    exit 1
fi
echo

echo "=== All Event Logging Tests Passed ==="
