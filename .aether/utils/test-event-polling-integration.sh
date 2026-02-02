#!/bin/bash
# Event Polling Integration Test Suite
# Tests event polling behavior for all Worker Ant castes
#
# Tests:
# - Worker Ants can poll for events using get_events_for_subscriber()
# - Worker Ants receive only events matching their subscription criteria
# - Worker Ants mark events as delivered to prevent reprocessing
# - Different castes receive different events based on caste-specific subscriptions
# - Event polling works at execution boundaries (start, after writes, after commands)

# Change to repo root to ensure consistent paths
cd "$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")"

# Source event bus
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "${SCRIPT_DIR}/event-bus.sh"

# Disable trap to avoid interference with test execution
trap - EXIT TERM INT

# Get absolute lock dir path
LOCK_DIR_ABS="$(pwd)/.aether/locks"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper functions
test_assert() {
    local test_name="$1"
    local condition="$2"
    TESTS_RUN=$((TESTS_RUN + 1))

    if eval "$condition"; then
        echo "  PASS: $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo "  FAIL: $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 0  # Return 0 to continue testing even on failure
    fi
}

# Setup: Initialize event bus
setup() {
    echo "=== Setup: Initializing event bus ==="
    # Clean up any stale locks first
    if [ -d "$LOCK_DIR_ABS" ]; then
        find "$LOCK_DIR_ABS" -name "*.lock" -type f -delete 2>/dev/null || true
        find "$LOCK_DIR_ABS" -name "*.pid" -type f -delete 2>/dev/null || true
    fi
    # Backup existing events.json if present
    if [ -f "$EVENTS_FILE" ]; then
        cp "$EVENTS_FILE" "${EVENTS_FILE}.backup.$$"
    fi
    initialize_event_bus > /dev/null 2>&1
    echo "Event bus initialized"
    echo ""
}

# Teardown: Restore backup
teardown() {
    echo "=== Teardown: Restoring event bus state ==="
    # Clean up any stale locks
    if [ -d "$LOCK_DIR_ABS" ]; then
        find "$LOCK_DIR_ABS" -name "*.lock" -type f -delete 2>/dev/null || true
        find "$LOCK_DIR_ABS" -name "*.pid" -type f -delete 2>/dev/null || true
    fi
    if [ -f "${EVENTS_FILE}.backup.$$" ]; then
        mv "${EVENTS_FILE}.backup.$$" "$EVENTS_FILE"
        echo "Restored original events.json"
    fi
    echo ""
}

# Test 1: Colonizer Ant can subscribe and poll for events
test_colonizer_event_polling() {
    echo "=== Test 1: Colonizer Ant Event Polling ==="

    local caste="colonizer"
    local subscriber_id="test_colonizer_1"

    # Subscribe to colonizer-specific topics
    subscribe_to_events "$subscriber_id" "$caste" "phase_complete" '{}' > /dev/null 2>&1
    subscribe_to_events "$subscriber_id" "$caste" "spawn_request" '{}' > /dev/null 2>&1
    subscribe_to_events "$subscriber_id" "$caste" "error" '{}' > /dev/null 2>&1

    # Publish test events
    publish_event "phase_complete" "test_phase" '{"phase": "11"}' "test_publisher" "colonizer" > /dev/null 2>&1
    publish_event "spawn_request" "test_spawn" '{"specialist": "scout"}' "test_publisher" "colonizer" > /dev/null 2>&1
    publish_event "error" "test_error" '{"message": "test error"}' "test_publisher" "builder" > /dev/null 2>&1

    # Poll for events
    events=$(get_events_for_subscriber "$subscriber_id" "$caste" 2>/dev/null)
    event_count=$(echo "$events" | jq 'length' 2>/dev/null || echo "0")

    test_assert "Colonizer receives events (count >= 2)" '[ "$event_count" -ge 2 ]'
    test_assert "Colonizer receives spawn_request events" '[ "$(echo "$events" | jq -r "[.[] | select(.topic == \"spawn_request\")] | length" 2>/dev/null)" -ge 1 ]'

    # Mark events as delivered
    mark_events_delivered "$subscriber_id" "$caste" "$events" > /dev/null 2>&1

    # Poll again - should receive empty array (events already delivered)
    events_after=$(get_events_for_subscriber "$subscriber_id" "$caste" 2>/dev/null)
    test_assert "Colonizer does not reprocess delivered events" '[ "$events_after" == "[]" ]'

    echo ""
}

# Test 2: Builder Ant event filtering
test_builder_event_filtering() {
    echo "=== Test 2: Builder Ant Event Filtering ==="

    local caste="builder"
    local subscriber_id="test_builder_1"

    # Subscribe to builder-specific topics
    subscribe_to_events "$subscriber_id" "$caste" "task_started" '{}' > /dev/null 2>&1
    subscribe_to_events "$subscriber_id" "$caste" "task_completed" '{}' > /dev/null 2>&1
    subscribe_to_events "$subscriber_id" "$caste" "error" '{}' > /dev/null 2>&1

    # Publish test events
    publish_event "task_started" "test_task" '{"task": "Build auth module"}' "test_publisher" "queen" > /dev/null 2>&1
    publish_event "task_completed" "test_task" '{"task": "Build auth module"}' "test_publisher" "builder" > /dev/null 2>&1
    publish_event "phase_complete" "test_phase" '{"phase": "11"}' "test_publisher" "colonizer" > /dev/null 2>&1

    # Poll for events
    events=$(get_events_for_subscriber "$subscriber_id" "$caste" 2>/dev/null)
    task_started_count=$(echo "$events" | jq -r '[.[] | select(.topic == "task_started")] | length' 2>/dev/null || echo "0")
    phase_complete_count=$(echo "$events" | jq -r '[.[] | select(.topic == "phase_complete")] | length' 2>/dev/null || echo "0")

    test_assert "Builder receives task_started events" '[ "$task_started_count" -ge 1 ]'
    test_assert "Builder does not receive phase_complete events (not subscribed)" '[ "$phase_complete_count" -eq 0 ]'

    echo ""
}

# Test 3: Watcher Ant task event monitoring
test_watcher_task_monitoring() {
    echo "=== Test 3: Watcher Ant Task Monitoring ==="

    local caste="watcher"
    local subscriber_id="test_watcher_1"

    # Subscribe to watcher-specific topics
    subscribe_to_events "$subscriber_id" "$caste" "task_completed" '{}' > /dev/null 2>&1
    subscribe_to_events "$subscriber_id" "$caste" "task_failed" '{}' > /dev/null 2>&1
    subscribe_to_events "$subscriber_id" "$caste" "error" '{}' > /dev/null 2>&1

    # Publish test events
    publish_event "task_completed" "test_task" '{"task": "Build auth module"}' "test_publisher" "builder" > /dev/null 2>&1
    publish_event "task_failed" "test_task" '{"task": "Build payment module", "error": "timeout"}' "test_publisher" "builder" > /dev/null 2>&1
    publish_event "error" "test_error" '{"message": "test error"}' "test_publisher" "builder" > /dev/null 2>&1

    # Poll for events
    events=$(get_events_for_subscriber "$subscriber_id" "$caste" 2>/dev/null)
    event_count=$(echo "$events" | jq 'length' 2>/dev/null || echo "0")
    failed_count=$(echo "$events" | jq -r '[.[] | select(.topic == "task_failed")] | length' 2>/dev/null || echo "0")

    test_assert "Watcher receives task events (count >= 2)" '[ "$event_count" -ge 2 ]'
    test_assert "Watcher receives task_failed events" '[ "$failed_count" -eq 1 ]'

    echo ""
}

# Test 4: Security Watcher specialist filtering
test_security_watcher_filtering() {
    echo "=== Test 4: Security Watcher Specialist Filtering ==="

    local caste="security-watcher"
    local subscriber_id="test_security_watcher_1"

    # Subscribe with filter criteria
    subscribe_to_events "$subscriber_id" "$caste" "error" '{"category": "security"}' > /dev/null 2>&1
    subscribe_to_events "$subscriber_id" "$caste" "task_completed" '{}' > /dev/null 2>&1
    subscribe_to_events "$subscriber_id" "$caste" "task_failed" '{}' > /dev/null 2>&1

    # Publish test events
    publish_event "error" "test_error" '{"category": "security", "message": "SQL injection"}' "test_publisher" "builder" > /dev/null 2>&1
    publish_event "error" "test_error" '{"category": "performance", "message": "Slow query"}' "test_publisher" "builder" > /dev/null 2>&1
    publish_event "task_completed" "test_task" '{"task": "Fix auth bug"}' "test_publisher" "builder" > /dev/null 2>&1

    # Poll for events
    events=$(get_events_for_subscriber "$subscriber_id" "$caste" 2>/dev/null)
    security_error_count=$(echo "$events" | jq -r '[.[] | select(.topic == "error" and .data.category == "security")] | length' 2>/dev/null || echo "0")
    performance_error_count=$(echo "$events" | jq -r '[.[] | select(.topic == "error" and .data.category == "performance")] | length' 2>/dev/null || echo "0")

    test_assert "Security Watcher receives security-related errors" '[ "$security_error_count" -eq 1 ]'
    test_assert "Security Watcher does not receive performance errors (filtered)" '[ "$performance_error_count" -eq 0 ]'

    echo ""
}

# Test 5: Event delivery tracking prevents reprocessing
test_delivery_tracking() {
    echo "=== Test 5: Event Delivery Tracking ==="

    local caste="architect"
    local subscriber_id="test_architect_1"

    # Subscribe to topics
    subscribe_to_events "$subscriber_id" "$caste" "phase_complete" '{}' > /dev/null 2>&1
    subscribe_to_events "$subscriber_id" "$caste" "task_completed" '{}' > /dev/null 2>&1

    # Publish event
    publish_event "phase_complete" "test_phase" '{"phase": "11"}' "test_publisher" "queen" > /dev/null 2>&1

    # First poll
    events_first=$(get_events_for_subscriber "$subscriber_id" "$caste" 2>/dev/null)
    first_count=$(echo "$events_first" | jq 'length' 2>/dev/null || echo "0")

    # Mark as delivered
    mark_events_delivered "$subscriber_id" "$caste" "$events_first" > /dev/null 2>&1

    # Second poll
    events_second=$(get_events_for_subscriber "$subscriber_id" "$caste" 2>/dev/null)
    second_count=$(echo "$events_second" | jq 'length' 2>/dev/null || echo "0")

    test_assert "First poll returns events" '[ "$first_count" -gt 0 ]'
    test_assert "Second poll returns empty array (already delivered)" '[ "$second_count" -eq 0 ]'

    echo ""
}

# Test 6: Multiple castes receive different events
test_caste_specific_subscriptions() {
    echo "=== Test 6: Caste-Specific Subscriptions ==="

    # Publish diverse events
    publish_event "phase_complete" "test_phase" '{"phase": "11"}' "test_publisher" "queen" > /dev/null 2>&1
    publish_event "spawn_request" "test_spawn" '{"specialist": "scout"}' "test_publisher" "colonizer" > /dev/null 2>&1
    publish_event "task_started" "test_task" '{"task": "Build auth"}' "test_publisher" "queen" > /dev/null 2>&1
    publish_event "error" "test_error" '{"message": "test error"}' "test_publisher" "builder" > /dev/null 2>&1

    # Colonizer subscribes to phase_complete and spawn_request
    subscribe_to_events "test_colonizer_2" "colonizer" "phase_complete" '{}' > /dev/null 2>&1
    subscribe_to_events "test_colonizer_2" "colonizer" "spawn_request" '{}' > /dev/null 2>&1

    # Builder subscribes to task_started and error
    subscribe_to_events "test_builder_2" "builder" "task_started" '{}' > /dev/null 2>&1
    subscribe_to_events "test_builder_2" "builder" "error" '{}' > /dev/null 2>&1

    # Poll for events
    colonizer_events=$(get_events_for_subscriber "test_colonizer_2" "colonizer" 2>/dev/null)
    builder_events=$(get_events_for_subscriber "test_builder_2" "builder" 2>/dev/null)

    colonizer_count=$(echo "$colonizer_events" | jq 'length' 2>/dev/null || echo "0")
    builder_count=$(echo "$builder_events" | jq 'length' 2>/dev/null || echo "0")

    test_assert "Colonizer receives phase_complete and spawn_request" '[ "$colonizer_count" -ge 2 ]'
    test_assert "Builder receives task_started and error" '[ "$builder_count" -ge 2 ]'

    echo ""
}

# Run all tests
main() {
    echo "========================================================"
    echo "   Event Polling Integration Test Suite"
    echo "   Testing event polling for all Worker Ant castes"
    echo "========================================================"
    echo ""

    setup

    test_colonizer_event_polling
    test_builder_event_filtering
    test_watcher_task_monitoring
    test_security_watcher_filtering
    test_delivery_tracking
    test_caste_specific_subscriptions

    teardown

    echo "========================================================"
    echo "   Test Results"
    echo "========================================================"
    echo "Tests run:    $TESTS_RUN"
    echo "Tests passed: $TESTS_PASSED"
    echo "Tests failed: $TESTS_FAILED"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo "All tests passed!"
        return 0
    else
        echo "Some tests failed"
        return 1
    fi
}

# Run tests if executed directly
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    main "$@"
fi
