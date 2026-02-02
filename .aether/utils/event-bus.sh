#!/bin/bash
# Aether Event Bus Utility
# Implements pub/sub event system for colony-wide coordination
#
# ## Async Non-Blocking Design
#
# The event bus uses pull-based delivery for optimal compatibility with
# prompt-based Worker Ants (which are not persistent processes):
#
# - **Publish**: Worker Ants publish events via publish_event()
#   - Writes event to events.json event_log
#   - Returns immediately with event_id (non-blocking)
#   - Does NOT wait for subscribers
#   - Does NOT call subscriber code
#
# - **Subscribe**: Worker Ants register interest via subscribe_to_events()
#   - Records subscription in events.json
#   - Returns subscription_id
#
# - **Deliver**: Worker Ants poll for events via get_events_for_subscriber()
#   - Returns events matching subscriptions since last poll
#   - Worker Ant processes events when they execute
#   - Marks events as delivered via mark_events_delivered()
#
# This design provides true async semantics:
# - Publishers and subscribers are decoupled
# - No background processes or daemons required
# - Works naturally with prompt-based agents (execute, poll, exit)
# - Concurrent publishes safe (file locking prevents corruption)
#
# Usage:
#   source .aether/utils/event-bus.sh
#   initialize_event_bus
#   publish_event "topic" "type" '{"data": "value"}' "publisher" "caste"
#   subscribe_to_events "subscriber_id" "caste" "topic_pattern" '{}'
#   get_events_for_subscriber "subscriber_id" "caste"

# Event bus storage file
EVENTS_FILE="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")/.aether/data/events.json"

# Source required utilities
source "$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")/.aether/utils/atomic-write.sh"
source "$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")/.aether/utils/file-lock.sh"
source "$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")/.aether/utils/event-metrics.sh"

# Initialize event bus (create events.json if not exists)
# Arguments: none
# Returns: 0 on success, 1 on failure
initialize_event_bus() {
    local events_dir=$(dirname "$EVENTS_FILE")

    # Create directory if not exists
    mkdir -p "$events_dir"

    # Check if events.json already exists
    if [ -f "$EVENTS_FILE" ]; then
        # Validate existing file
        if ! python3 -c "import json; json.load(open('$EVENTS_FILE'))" 2>/dev/null; then
            echo "Error: Invalid JSON in $EVENTS_FILE"
            return 1
        fi
        echo "Event bus already initialized at $EVENTS_FILE"
        return 0
    fi

    # Create initial events.json structure
    local initial_content='{
  "$schema": "Aether Event Bus v1.0",
  "topics": {
    "phase_complete": {
      "description": "Phase execution completed",
      "subscriber_count": 0
    },
    "error": {
      "description": "Error occurred during execution",
      "subscriber_count": 0
    },
    "spawn_request": {
      "description": "Request to spawn specialist Worker Ant",
      "subscriber_count": 0
    },
    "task_started": {
      "description": "Worker Ant started executing a task",
      "subscriber_count": 0
    },
    "task_completed": {
      "description": "Worker Ant completed a task successfully",
      "subscriber_count": 0
    },
    "task_failed": {
      "description": "Worker Ant failed to complete a task",
      "subscriber_count": 0
    }
  },
  "subscriptions": [],
  "event_log": [],
  "metrics": {
    "total_published": 0,
    "total_subscriptions": 0,
    "total_delivered": 0,
    "publish_rate_per_minute": 0.0,
    "average_delivery_latency_ms": 0,
    "backlog_count": 0,
    "last_updated": null
  },
  "config": {
    "max_event_log_size": 1000,
    "max_subscriptions_per_topic": 50,
    "event_retention_hours": 168
  }
}'

    # Write using atomic_write for safety
    if atomic_write "$EVENTS_FILE" "$initial_content"; then
        echo "Event bus initialized at $EVENTS_FILE"
        return 0
    else
        echo "Error: Failed to initialize event bus"
        return 1
    fi
}

# Generate unique event ID
# Arguments: none
# Returns: unique event ID (evt_<timestamp>_<random>)
generate_event_id() {
    local timestamp=$(date +%s)
    local random_string=$(openssl rand -hex 4 2>/dev/null || echo "$(date +%N)%")
    echo "evt_${timestamp}_${random_string}"
}

# Generate correlation ID for event chains
# Arguments: none
# Returns: unique correlation ID
generate_correlation_id() {
    local timestamp=$(date +%s)
    local random_string=$(openssl rand -hex 4 2>/dev/null || echo "$(date +%N)%")
    echo "corr_${timestamp}_${random_string}"
}

# Publish event to event bus
# Arguments: topic, event_type, event_data (JSON string), publisher, publisher_caste (optional)
# Returns: event_id on success, 1 on failure
publish_event() {
    local topic="$1"
    local event_type="$2"
    local event_data="$3"
    local publisher="${4:-unknown}"
    local publisher_caste="${5:-}"

    # Validate arguments
    if [ -z "$topic" ] || [ -z "$event_type" ] || [ -z "$event_data" ]; then
        echo "Error: topic, event_type, and event_data are required" >&2
        return 1
    fi

    # Validate event_data is valid JSON
    if ! echo "$event_data" | python3 -c "import json, sys; json.load(sys.stdin)" 2>/dev/null; then
        echo "Error: event_data must be valid JSON" >&2
        return 1
    fi

    # Check if events.json exists
    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized. Run initialize_event_bus first." >&2
        return 1
    fi

    # Generate event metadata
    local event_id=$(generate_event_id)
    local correlation_id=$(generate_correlation_id)
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire file lock for concurrent access safety
    if ! acquire_lock "$EVENTS_FILE"; then
        echo "Error: Failed to acquire event bus lock" >&2
        return 1
    fi

    # Create temp file for jq update
    local temp_file="/tmp/event_publish.$$.tmp"

    # Build jq command with or without caste
    if [ -n "$publisher_caste" ]; then
        # With caste - use --arg for string value
        jq --arg id "$event_id" \
           --arg topic "$topic" \
           --arg type "$event_type" \
           --argjson data "$event_data" \
           --arg publisher "$publisher" \
           --arg caste "$publisher_caste" \
           --arg timestamp "$timestamp" \
           --arg corr_id "$correlation_id" \
           '
           .event_log += [{
             "id": $id,
             "topic": $topic,
             "type": $type,
             "data": $data,
             "metadata": {
               "publisher": $publisher,
               "publisher_caste": $caste,
               "timestamp": $timestamp,
               "correlation_id": $corr_id
             }
           }] |
           .metrics.total_published += 1 |
           .metrics.backlog_count += 1 |
           .metrics.last_updated = $timestamp |
           (.topics[$topic] //= {"description": "Auto-created topic", "subscriber_count": 0})
           ' "$EVENTS_FILE" > "$temp_file"
    else
        # Without caste - use null
        jq --arg id "$event_id" \
           --arg topic "$topic" \
           --arg type "$event_type" \
           --argjson data "$event_data" \
           --arg publisher "$publisher" \
           --arg timestamp "$timestamp" \
           --arg corr_id "$correlation_id" \
           '
           .event_log += [{
             "id": $id,
             "topic": $topic,
             "type": $type,
             "data": $data,
             "metadata": {
               "publisher": $publisher,
               "publisher_caste": null,
               "timestamp": $timestamp,
               "correlation_id": $corr_id
             }
           }] |
           .metrics.total_published += 1 |
           .metrics.backlog_count += 1 |
           .metrics.last_updated = $timestamp |
           (.topics[$topic] //= {"description": "Auto-created topic", "subscriber_count": 0})
           ' "$EVENTS_FILE" > "$temp_file"
    fi

    if [ $? -ne 0 ]; then
        echo "Error: Failed to update event log" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    # Atomic write
    if ! atomic_write_from_file "$EVENTS_FILE" "$temp_file"; then
        echo "Error: Failed to write event to event bus" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    rm -f "$temp_file"

    # Trim event log if exceeds max size (ring buffer)
    trim_event_log

    # Release lock BEFORE calling update_event_metrics to avoid deadlock
    release_lock

    # Update publish metrics (after releasing lock)
    update_event_metrics "publish" > /dev/null 2>&1

    # Return event ID (non-blocking - write complete, returns immediately)
    echo "$event_id"
    return 0
}

# Trim event log to max_event_log_size (ring buffer)
# Arguments: none
# Returns: 0 on success, 1 on failure
trim_event_log() {
    local max_size=$(jq -r '.config.max_event_log_size' "$EVENTS_FILE" 2>/dev/null || echo "1000")
    local current_size=$(jq -r '.event_log | length' "$EVENTS_FILE" 2>/dev/null || echo "0")

    # Handle null/empty values - ensure numeric comparison
    if [[ ! "$current_size" =~ ^[0-9]+$ ]] || [ -z "$current_size" ]; then
        current_size=0
    fi
    if [[ ! "$max_size" =~ ^[0-9]+$ ]] || [ -z "$max_size" ]; then
        max_size=1000
    fi

    if [ "$current_size" -gt "$max_size" ]; then
        local trim_count=$((current_size - max_size))
        local temp_file="/tmp/event_trim.$$.tmp"

        # Keep most recent events (ring buffer)
        jq --argjson keep "$max_size" \
           '
           if .event_log then
             .event_log = .event_log[-($keep):] |
             .metrics.backlog_count = (.event_log | length)
           else
             .
           end
           ' "$EVENTS_FILE" > "$temp_file"

        if [ $? -eq 0 ]; then
            atomic_write_from_file "$EVENTS_FILE" "$temp_file"
            echo "Trimmed $trim_count old events from event log" >&2
        fi

        rm -f "$temp_file"
    fi

    return 0
}

# Generate unique subscription ID
# Arguments: none
# Returns: unique subscription ID (sub_<timestamp>_<random>)
generate_subscription_id() {
    local timestamp=$(date +%s)
    local random_string=$(openssl rand -hex 4 2>/dev/null || echo "$(date +%N)%")
    echo "sub_${timestamp}_${random_string}"
}

# Subscribe to event topics
# Arguments: subscriber_id, subscriber_caste, topic_pattern, filter_criteria (JSON string, optional)
# Returns: subscription_id on success, 1 on failure
subscribe_to_events() {
    local subscriber_id="$1"
    local subscriber_caste="$2"
    local topic_pattern="$3"
    local filter_criteria="$4"
    if [ -z "$filter_criteria" ]; then
        filter_criteria="{}"
    fi

    # Validate arguments
    if [ -z "$subscriber_id" ] || [ -z "$subscriber_caste" ] || [ -z "$topic_pattern" ]; then
        echo "Error: subscriber_id, subscriber_caste, and topic_pattern are required" >&2
        return 1
    fi

    # Validate filter_criteria is valid JSON (if provided)
    if [ "$filter_criteria" != "{}" ]; then
        if ! echo "$filter_criteria" | python3 -c "import json, sys; json.load(sys.stdin)" 2>/dev/null; then
            echo "Error: filter_criteria must be valid JSON" >&2
            return 1
        fi
    fi

    # Check if events.json exists
    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized. Run initialize_event_bus first." >&2
        return 1
    fi

    # Generate subscription metadata
    local sub_id=$(generate_subscription_id)
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire file lock for concurrent access safety
    if ! acquire_lock "$EVENTS_FILE"; then
        echo "Error: Failed to acquire event bus lock" >&2
        return 1
    fi

    # Create temp file for jq update
    local temp_file="/tmp/event_subscribe.$$.tmp"

    # Add subscription and update metrics via jq
    jq --arg id "$sub_id" \
       --arg subscriber "$subscriber_id" \
       --arg caste "$subscriber_caste" \
       --arg pattern "$topic_pattern" \
       --argjson filter "$filter_criteria" \
       --arg timestamp "$timestamp" \
       '
       .subscriptions += [{
         "id": $id,
         "subscriber_id": $subscriber,
         "subscriber_caste": $caste,
         "topic_pattern": $pattern,
         "filter_criteria": $filter,
         "created_at": $timestamp,
         "last_event_delivered": null,
         "delivery_count": 0
       }] |
       .metrics.total_subscriptions += 1 |
       .metrics.last_updated = $timestamp |
       if .topics[$pattern] then
         .topics[$pattern].subscriber_count += 1
       else
         .topics[$pattern] = {
           "description": "Auto-created topic from subscription",
           "subscriber_count": 1
         }
       end
       ' "$EVENTS_FILE" > "$temp_file"

    if [ $? -ne 0 ]; then
        echo "Error: Failed to add subscription" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    # Atomic write
    if ! atomic_write_from_file "$EVENTS_FILE" "$temp_file"; then
        echo "Error: Failed to write subscription to event bus" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    rm -f "$temp_file"

    # Release lock BEFORE calling update_event_metrics to avoid deadlock
    release_lock

    # Update subscribe metrics (after releasing lock)
    update_event_metrics "subscribe" > /dev/null 2>&1

    # Return subscription ID
    echo "$sub_id"
    return 0
}

# Unsubscribe from event topics
# Arguments: subscription_id
# Returns: 0 on success, 1 on failure
unsubscribe_from_events() {
    local subscription_id="$1"

    if [ -z "$subscription_id" ]; then
        echo "Error: subscription_id is required" >&2
        return 1
    fi

    # Check if events.json exists
    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized" >&2
        return 1
    fi

    # Acquire file lock
    if ! acquire_lock "$EVENTS_FILE"; then
        echo "Error: Failed to acquire event bus lock" >&2
        return 1
    fi

    # Get subscription details before removing (for updating subscriber_count)
    local subscription=$(jq -r --arg id "$subscription_id" '.subscriptions[] | select(.id == $id)' "$EVENTS_FILE")
    local topic_pattern=$(echo "$subscription" | jq -r '.topic_pattern')

    # Create temp file for jq update
    local temp_file="/tmp/event_unsubscribe.$$.tmp"

    # Remove subscription and update metrics
    jq --arg id "$subscription_id" \
       --arg pattern "$topic_pattern" \
       '
       .subscriptions = [.subscriptions[] | select(.id != $id)] |
       if .topics[$pattern] then
         .topics[$pattern].subscriber_count |= (. - 1)
       end |
       .metrics.last_updated = (now | todate)
       ' "$EVENTS_FILE" > "$temp_file"

    if [ $? -ne 0 ]; then
        echo "Error: Failed to remove subscription" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    # Atomic write
    if ! atomic_write_from_file "$EVENTS_FILE" "$temp_file"; then
        echo "Error: Failed to write unsubscribe to event bus" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    rm -f "$temp_file"

    # Release lock
    release_lock

    echo "Unsubscribed: $subscription_id"
    return 0
}

# List all subscriptions (optionally filter by subscriber_id)
# Arguments: [subscriber_id] (optional)
# Returns: JSON array of subscriptions
list_subscriptions() {
    local subscriber_filter="${1:-}"

    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized" >&2
        return 1
    fi

    if [ -n "$subscriber_filter" ]; then
        jq -r --arg subscriber "$subscriber_filter" '.subscriptions[] | select(.subscriber_id == $subscriber)' "$EVENTS_FILE"
    else
        jq -r '.subscriptions[]' "$EVENTS_FILE"
    fi
}

# Get events for a subscriber (pull-based delivery)
# Arguments: subscriber_id, subscriber_caste
# Returns: JSON array of matching events (empty array if none)
get_events_for_subscriber() {
    local subscriber_id="$1"
    local subscriber_caste="${2:-}"

    if [ -z "$subscriber_id" ]; then
        echo "Error: subscriber_id is required" >&2
        return 1
    fi

    # Check if events.json exists
    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized" >&2
        return 1
    fi

    # Acquire file lock for read
    if ! acquire_lock "$EVENTS_FILE"; then
        echo "Error: Failed to acquire event bus lock" >&2
        return 1
    fi

    # Get all subscriptions for this subscriber
    local subscriptions=$(jq -c --arg subscriber "$subscriber_id" \
       '.subscriptions[] | select(.subscriber_id == $subscriber)' \
       "$EVENTS_FILE")

    # If no subscriptions, return empty array
    if [ -z "$subscriptions" ]; then
        release_lock
        echo "[]"
        return 0
    fi

    # Collect matching events from all subscriptions
    local matching_events="[]"
    local subscription_timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    while IFS= read -r sub; do
        # Skip empty lines
        [ -z "$sub" ] && continue

        local topic_pattern=$(echo "$sub" | jq -r '.topic_pattern')
        local filter_criteria=$(echo "$sub" | jq -c '.filter_criteria')
        local last_delivered=$(echo "$sub" | jq -r '.last_event_delivered // "null"')

        # Find events matching topic pattern (with wildcard support)
        # jq test() function performs regex matching
        local events=$(jq -c --arg pattern "$topic_pattern" \
           --argjson filter "$filter_criteria" \
           --arg last "$last_delivered" \
           --arg caste "$subscriber_caste" \
           --arg timestamp "$subscription_timestamp" \
           '
           [.event_log[] | . as $event |
           select(
             # Topic pattern matching (wildcard support via test())
             (.topic | test($pattern)) and
             # Only events since last delivery (or all if never delivered)
             (.metadata.timestamp > $last or $last == "null") and
             # Filter criteria matching (if specified)
             (
               $filter == {} or
               ($filter | to_entries | all(.key as $k | .value as $v | $event.data[$k] == $v))
             )
           )
           ] |
           # Add delivery timestamp for tracking
           map(. + {delivered_at: $timestamp})
           ' "$EVENTS_FILE")

        # Accumulate matching events
        if [ "$events" != "[]" ]; then
            matching_events=$(echo "$matching_events" | jq --argjson new "$events" '. + $new')
        fi
    done <<< "$subscriptions"

    # Release lock
    release_lock

    # Return matching events (empty array if none)
    echo "$matching_events"
    return 0
}

# Mark events as delivered for a subscriber
# Arguments: subscriber_id, subscriber_caste, events_json_array
# Returns: 0 on success, 1 on failure
mark_events_delivered() {
    local subscriber_id="$1"
    local subscriber_caste="${2:-}"
    local events_json="$3"

    if [ -z "$subscriber_id" ]; then
        echo "Error: subscriber_id is required" >&2
        return 1
    fi

    if [ -z "$events_json" ]; then
        echo "Error: events_json is required" >&2
        return 1
    fi

    # Check if events.json exists
    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized" >&2
        return 1
    fi

    # Validate events_json is valid JSON array
    if ! echo "$events_json" | python3 -c "import json, sys; data=json.load(sys.stdin); assert isinstance(data, list)" 2>/dev/null; then
        echo "Error: events_json must be a valid JSON array" >&2
        return 1
    fi

    # If empty array, nothing to mark
    if [ "$events_json" = "[]" ]; then
        return 0
    fi

    # Get the most recent event timestamp
    local latest_timestamp=$(echo "$events_json" | jq -r '[.[].metadata.timestamp] | max' 2>/dev/null)

    if [ -z "$latest_timestamp" ] || [ "$latest_timestamp" = "null" ]; then
        echo "Error: Could not determine latest event timestamp" >&2
        return 1
    fi

    # Acquire file lock
    if ! acquire_lock "$EVENTS_FILE"; then
        echo "Error: Failed to acquire event bus lock" >&2
        return 1
    fi

    local temp_file="/tmp/event_mark_delivered.$$.tmp"

    # Update all subscriptions for this subscriber
    jq --arg subscriber "$subscriber_id" \
       --arg latest "$latest_timestamp" \
       --argjson count "$(echo "$events_json" | jq 'length')" \
       '
       .subscriptions |= map(
         if .subscriber_id == $subscriber then
           .last_event_delivered = $latest |
           .delivery_count += $count
         else
           .
         end
       ) |
       .metrics.total_delivered += $count |
       .metrics.backlog_count -= $count |
       .metrics.last_updated = $latest
       ' "$EVENTS_FILE" > "$temp_file"

    if [ $? -ne 0 ]; then
        echo "Error: Failed to mark events as delivered" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    # Atomic write
    if ! atomic_write_from_file "$EVENTS_FILE" "$temp_file"; then
        echo "Error: Failed to write delivery update to event bus" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    rm -f "$temp_file"

    # Release lock BEFORE calling update_event_metrics to avoid deadlock
    release_lock

    # Update delivery metrics (after releasing lock)
    update_event_metrics "deliver" > /dev/null 2>&1

    return 0
}

# Cleanup old events based on retention time
# Arguments: [retention_hours] (optional, defaults to config.event_retention_hours)
# Returns: 0 on success, 1 on failure
cleanup_old_events() {
    local retention_hours="${1:-}"

    if [ -z "$retention_hours" ]; then
        # Read from config if not specified
        retention_hours=$(jq -r '.config.event_retention_hours' "$EVENTS_FILE")
    fi

    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized" >&2
        return 1
    fi

    # Calculate cutoff timestamp (macOS and Linux compatible)
    local cutoff_timestamp
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        cutoff_timestamp=$(date -v-${retention_hours}H -u +"%Y-%m-%dT%H:%M:%SZ")
    else
        # Linux
        cutoff_timestamp=$(date -d "$retention_hours hours ago" -u +"%Y-%m-%dT%H:%M:%SZ")
    fi

    echo "Cleaning up events older than $retention_hours hours (before $cutoff_timestamp)..."

    # Acquire file lock
    if ! acquire_lock "$EVENTS_FILE"; then
        echo "Error: Failed to acquire event bus lock" >&2
        return 1
    fi

    local temp_file="/tmp/event_cleanup.$$.tmp"

    # Remove events older than cutoff
    jq --arg cutoff "$cutoff_timestamp" \
       '
       .event_log = [.event_log[] | select(.metadata.timestamp > $cutoff)] |
       .metrics.backlog_count = (.event_log | length)
       ' "$EVENTS_FILE" > "$temp_file"

    if [ $? -ne 0 ]; then
        echo "Error: Failed to cleanup old events" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    # Atomic write
    if ! atomic_write_from_file "$EVENTS_FILE" "$temp_file"; then
        echo "Error: Failed to write cleanup to event bus" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    rm -f "$temp_file"

    # Release lock
    release_lock

    echo "Cleanup complete"
    return 0
}

# Get event history with optional filtering
# Arguments: [topic_pattern] [limit] [since_timestamp]
# Returns: JSON array of events
get_event_history() {
    local topic_pattern="${1:-}"
    local limit="${2:-}"
    local since_timestamp="${3:-}"

    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized" >&2
        return 1
    fi

    local jq_filter='.event_log'

    # Apply topic filter if specified
    if [ -n "$topic_pattern" ]; then
        jq_filter="$jq_filter | map(select(.topic | test(\"$topic_pattern\")))"
    fi

    # Apply since filter if specified
    if [ -n "$since_timestamp" ]; then
        jq_filter="$jq_filter | map(select(.metadata.timestamp > \"$since_timestamp\"))"
    fi

    # Apply limit if specified (take most recent N)
    if [ -n "$limit" ]; then
        jq_filter="$jq_filter | .[-($limit):]"
    fi

    # Execute query
    jq "$jq_filter" "$EVENTS_FILE"
}

# Export event log to file
# Arguments: output_file [format] [topic_pattern]
# format: "json" (default) or "text"
# Returns: 0 on success, 1 on failure
export_event_log() {
    local output_file="$1"
    local format="${2:-json}"
    local topic_pattern="${3:-}"

    if [ -z "$output_file" ]; then
        echo "Error: output_file is required" >&2
        return 1
    fi

    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized" >&2
        return 1
    fi

    # Get events (optionally filtered by topic)
    local events
    if [ -n "$topic_pattern" ]; then
        events=$(get_event_history "$topic_pattern")
    else
        events=$(jq -r '.event_log' "$EVENTS_FILE")
    fi

    # Export based on format
    case "$format" in
        json)
            echo "$events" | jq '.' > "$output_file"
            ;;
        text)
            # Human-readable text format
            {
                echo "=== Event Log Export ==="
                echo "Exported at: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
                echo "Total events: $(echo "$events" | jq 'length')"
                echo ""
                echo "$events" | jq -r '.[] |
                    "\(.metadata.timestamp) [\(.topic)] \(.type)\n  Publisher: \(.metadata.publisher)\n  Data: \(.data | tojson)\n"'
            } > "$output_file"
            ;;
        *)
            echo "Error: Invalid format '$format'. Use 'json' or 'text'" >&2
            return 1
            ;;
    esac

    echo "Event log exported to $output_file"
    return 0
}

# Get event log statistics
# Arguments: none
# Returns: JSON with event statistics
get_event_stats() {
    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized" >&2
        return 1
    fi

    jq '
    {
        "total_events": (.event_log | length),
        "topics": (.event_log | group_by(.topic) | map({
            topic: (.[0].topic // "unknown"),
            count: length
        })),
        "types": (.event_log | group_by(.type) | map({
            type: (.[0].type // "unknown"),
            count: length
        })),
        "publishers": (.event_log | group_by(.metadata.publisher) | map({
            publisher: (.[0].metadata.publisher // "unknown"),
            count: length
        })),
        "time_range": {
            "earliest": ([.event_log[].metadata.timestamp] | min),
            "latest": ([.event_log[].metadata.timestamp] | max)
        },
        "metrics": .metrics
    }
    ' "$EVENTS_FILE"
}

# Export all functions
export -f initialize_event_bus generate_event_id generate_correlation_id publish_event trim_event_log generate_subscription_id subscribe_to_events unsubscribe_from_events list_subscriptions get_events_for_subscriber mark_events_delivered cleanup_old_events get_event_history export_event_log get_event_stats
