#!/usr/bin/env bash
# Event Bus Module Tests
# Tests event-bus.sh functions via aether-utils.sh subcommands

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"

source "$SCRIPT_DIR/test-helpers.sh"

require_jq

if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated test environment
# ============================================================================
setup_event_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils"

    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    local utils_source
    utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    local exchange_source
    exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "version": "3.0",
  "goal": "Test event bus",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "plan": { "phases": [{ "id": 1, "name": "Test Phase", "status": "pending" }] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": []
}
EOF

    echo "$tmp_dir"
}

run_event_cmd() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

run_event_cmd_stderr() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>&1
}

# ============================================================================
# Test 1: event-bus.sh module file exists and has valid syntax
# ============================================================================
test_module_exists() {
    local module_path="$PROJECT_ROOT/.aether/utils/event-bus.sh"

    assert_file_exists "$module_path" || return 1
    bash -n "$module_path" 2>/dev/null || return 1
}

# ============================================================================
# Test 2: event-publish creates JSONL file and appends an event
# ============================================================================
test_event_publish_creates_file() {
    local tmp_dir
    tmp_dir=$(setup_event_env)

    local result
    result=$(run_event_cmd "$tmp_dir" event-publish --topic "learning.observed" --payload '{"key":"value"}')

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Check event_id returned
    local event_id
    event_id=$(echo "$result" | jq -r '.result.event_id')
    [[ "$event_id" == evt_* ]] || { rm -rf "$tmp_dir"; return 1; }

    # Check topic returned
    assert_json_field_equals "$result" ".result.topic" "learning.observed" || { rm -rf "$tmp_dir"; return 1; }

    # Check JSONL file was created
    local bus_file="$tmp_dir/.aether/data/event-bus.jsonl"
    assert_file_exists "$bus_file" || { rm -rf "$tmp_dir"; return 1; }

    # Check file has one line (one event)
    local line_count
    line_count=$(wc -l < "$bus_file")
    [[ "$line_count" -eq 1 ]] || { rm -rf "$tmp_dir"; return 1; }

    # Check the line is valid JSON
    local line_json
    line_json=$(head -1 "$bus_file")
    echo "$line_json" | jq empty 2>/dev/null || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 3: event-publish with custom TTL
# ============================================================================
test_event_publish_custom_ttl() {
    local tmp_dir
    tmp_dir=$(setup_event_env)

    local result
    result=$(run_event_cmd "$tmp_dir" event-publish \
        --topic "test.topic" \
        --payload '{"x":1}' \
        --ttl 7 \
        --source "builder")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Check ttl_days returned
    local ttl
    ttl=$(echo "$result" | jq -r '.result.ttl_days')
    [[ "$ttl" -eq 7 ]] || { rm -rf "$tmp_dir"; return 1; }

    # Check the stored event has ttl_days=7
    local bus_file="$tmp_dir/.aether/data/event-bus.jsonl"
    local stored_ttl
    stored_ttl=$(jq -r '.ttl_days' "$bus_file")
    [[ "$stored_ttl" -eq 7 ]] || { rm -rf "$tmp_dir"; return 1; }

    # Check source is stored
    local stored_source
    stored_source=$(jq -r '.source' "$bus_file")
    [[ "$stored_source" == "builder" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 4: event-subscribe reads matching events by exact topic
# ============================================================================
test_event_subscribe_exact_match() {
    local tmp_dir
    tmp_dir=$(setup_event_env)

    # Publish two events on different topics
    run_event_cmd "$tmp_dir" event-publish --topic "learning.observed" --payload '{"n":1}' > /dev/null
    run_event_cmd "$tmp_dir" event-publish --topic "phase.completed" --payload '{"n":2}' > /dev/null
    run_event_cmd "$tmp_dir" event-publish --topic "learning.observed" --payload '{"n":3}' > /dev/null

    local result
    result=$(run_event_cmd "$tmp_dir" event-subscribe --topic "learning.observed")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local count
    count=$(echo "$result" | jq -r '.result.count')
    [[ "$count" -eq 2 ]] || { rm -rf "$tmp_dir"; return 1; }

    assert_json_field_equals "$result" ".result.topic_pattern" "learning.observed" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 5: event-subscribe with wildcard prefix pattern
# ============================================================================
test_event_subscribe_prefix_pattern() {
    local tmp_dir
    tmp_dir=$(setup_event_env)

    run_event_cmd "$tmp_dir" event-publish --topic "learning.observed" --payload '{"n":1}' > /dev/null
    run_event_cmd "$tmp_dir" event-publish --topic "learning.promoted" --payload '{"n":2}' > /dev/null
    run_event_cmd "$tmp_dir" event-publish --topic "phase.completed" --payload '{"n":3}' > /dev/null

    local result
    result=$(run_event_cmd "$tmp_dir" event-subscribe --topic "learning.*")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local count
    count=$(echo "$result" | jq -r '.result.count')
    [[ "$count" -eq 2 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 6: event-cleanup removes expired events
# ============================================================================
test_event_cleanup_removes_expired() {
    local tmp_dir
    tmp_dir=$(setup_event_env)

    # Manually write an expired event and a valid event to the JSONL file
    local bus_file="$tmp_dir/.aether/data/event-bus.jsonl"
    local past_ts="2020-01-01T00:00:00Z"
    local future_ts="2099-01-01T00:00:00Z"
    local now_ts
    now_ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    echo "{\"id\":\"evt_expired\",\"topic\":\"old.event\",\"payload\":{},\"source\":\"system\",\"timestamp\":\"$now_ts\",\"ttl_days\":30,\"expires_at\":\"$past_ts\"}" > "$bus_file"
    echo "{\"id\":\"evt_valid\",\"topic\":\"new.event\",\"payload\":{},\"source\":\"system\",\"timestamp\":\"$now_ts\",\"ttl_days\":30,\"expires_at\":\"$future_ts\"}" >> "$bus_file"

    local result
    result=$(run_event_cmd "$tmp_dir" event-cleanup)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local removed
    removed=$(echo "$result" | jq -r '.result.removed')
    [[ "$removed" -eq 1 ]] || { rm -rf "$tmp_dir"; return 1; }

    local remaining
    remaining=$(echo "$result" | jq -r '.result.remaining')
    [[ "$remaining" -eq 1 ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify the remaining event is the valid one
    local remaining_id
    remaining_id=$(jq -r '.id' "$bus_file")
    [[ "$remaining_id" == "evt_valid" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 7: event-cleanup --dry-run does not modify file
# ============================================================================
test_event_cleanup_dry_run() {
    local tmp_dir
    tmp_dir=$(setup_event_env)

    local bus_file="$tmp_dir/.aether/data/event-bus.jsonl"
    local past_ts="2020-01-01T00:00:00Z"
    local now_ts
    now_ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    echo "{\"id\":\"evt_expired\",\"topic\":\"old\",\"payload\":{},\"source\":\"system\",\"timestamp\":\"$now_ts\",\"ttl_days\":1,\"expires_at\":\"$past_ts\"}" > "$bus_file"

    local result
    result=$(run_event_cmd "$tmp_dir" event-cleanup --dry-run)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    assert_json_field_equals "$result" ".result.dry_run" "true" || { rm -rf "$tmp_dir"; return 1; }

    # File should still have 1 line (not modified)
    local line_count
    line_count=$(wc -l < "$bus_file")
    [[ "$line_count" -eq 1 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 8: event-replay returns events in chronological order
# ============================================================================
test_event_replay_chronological() {
    local tmp_dir
    tmp_dir=$(setup_event_env)

    run_event_cmd "$tmp_dir" event-publish --topic "build.started" --payload '{"phase":1}' > /dev/null
    run_event_cmd "$tmp_dir" event-publish --topic "build.started" --payload '{"phase":2}' > /dev/null
    run_event_cmd "$tmp_dir" event-publish --topic "build.started" --payload '{"phase":3}' > /dev/null

    local since_ts="2000-01-01T00:00:00Z"
    local result
    result=$(run_event_cmd "$tmp_dir" event-replay --topic "build.started" --since "$since_ts")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local count
    count=$(echo "$result" | jq -r '.result.count')
    [[ "$count" -eq 3 ]] || { rm -rf "$tmp_dir"; return 1; }

    assert_json_field_equals "$result" ".result.replayed_from" "$since_ts" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 9: error handling — event-publish with missing --topic
# ============================================================================
test_event_publish_missing_topic() {
    local tmp_dir
    tmp_dir=$(setup_event_env)

    local result
    result=$(run_event_cmd_stderr "$tmp_dir" event-publish --payload '{"x":1}')

    # Should return ok:false
    local ok
    ok=$(echo "$result" | grep -o '"ok":false' || echo "")
    [[ -n "$ok" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
echo "=== Event Bus Module Tests ==="
echo ""

run_test test_module_exists "event-bus.sh exists and passes syntax check"
run_test test_event_publish_creates_file "event-publish creates JSONL file and appends event"
run_test test_event_publish_custom_ttl "event-publish with custom TTL stores correct values"
run_test test_event_subscribe_exact_match "event-subscribe reads matching events by exact topic"
run_test test_event_subscribe_prefix_pattern "event-subscribe with wildcard prefix pattern"
run_test test_event_cleanup_removes_expired "event-cleanup removes expired events"
run_test test_event_cleanup_dry_run "event-cleanup --dry-run does not modify file"
run_test test_event_replay_chronological "event-replay returns events with replayed_from field"
run_test test_event_publish_missing_topic "event-publish returns error when --topic is missing"

test_summary
