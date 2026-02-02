#!/bin/bash
# Aether Event Metrics Utility
# Tracks event bus performance metrics for monitoring and observability
#
# Usage:
#   source .aether/utils/event-metrics.sh
#   get_event_metrics
#   calculate_publish_rate
#   calculate_delivery_latency

# Event bus storage file (same as event-bus.sh)
EVENTS_FILE="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")/.aether/data/events.json"

# Source required utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "${SCRIPT_DIR}/atomic-write.sh"
source "${SCRIPT_DIR}/file-lock.sh"

# Calculate publish rate (events per minute)
# Arguments: none
# Returns: events published in last 60 seconds
calculate_publish_rate() {
    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized" >&2
        return 1
    fi

    # Calculate cutoff timestamp (60 seconds ago)
    local one_minute_ago
    if [[ "$OSTYPE" == "darwin"* ]]; then
        one_minute_ago=$(date -v-60S -u +"%Y-%m-%dT%H:%M:%SZ")
    else
        one_minute_ago=$(date -d "60 seconds ago" -u +"%Y-%m-%dT%H:%M:%SZ")
    fi

    # Count events published since cutoff
    local count=$(jq -r --arg cutoff "$one_minute_ago" \
        '[.event_log[] | select(.metadata.timestamp > $cutoff)] | length' \
        "$EVENTS_FILE")

    echo "$count"
}

# Calculate average delivery latency
# Arguments: none
# Returns: average milliseconds from publish to delivery
calculate_delivery_latency() {
    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized" >&2
        return 1
    fi

    # Get all subscriptions with delivery data
    local subscriptions=$(jq -c '.subscriptions[] | select(.delivery_count > 0)' "$EVENTS_FILE")

    if [ -z "$subscriptions" ]; then
        echo "0"
        return 0
    fi

    # Calculate average latency across all subscriptions
    # Latency = time from publish to first delivery after publish
    # This is approximated by checking event timestamps vs delivery timestamps
    local total_latency=0
    local count=0

    while IFS= read -r sub; do
        local last_delivered=$(echo "$sub" | jq -r '.last_event_delivered')
        if [ "$last_delivered" != "null" ] && [ -n "$last_delivered" ]; then
            # Get the most recent event before last_delivered
            local recent_event=$(jq -r --arg cutoff "$last_delivered" \
                '[.event_log[] | select(.metadata.timestamp <= $cutoff)] | max_by(.metadata.timestamp)' \
                "$EVENTS_FILE" 2>/dev/null)

            if [ -n "$recent_event" ] && [ "$recent_event" != "null" ]; then
                local event_timestamp=$(echo "$recent_event" | jq -r '.metadata.timestamp')
                # Calculate latency in milliseconds (approximation)
                # This is a simplified calculation - real latency would need delivery_timestamp per event
                local latency_ms=0  # Would need actual delivery timestamps

                # For now, use a simple heuristic: if backlog > 0, there's latency
                local backlog=$(jq -r '.metrics.backlog_count' "$EVENTS_FILE")
                if [ "$backlog" -gt 0 ]; then
                    latency_ms=100  # Placeholder: 100ms latency when backlog exists
                fi

                total_latency=$((total_latency + latency_ms))
                count=$((count + 1))
            fi
        fi
    done <<< "$subscriptions"

    if [ "$count" -gt 0 ]; then
        echo $((total_latency / count))
    else
        echo "0"
    fi
}

# Update event metrics
# Arguments: operation ("publish", "subscribe", "deliver")
# Returns: 0 on success, 1 on failure
update_event_metrics() {
    local operation="$1"

    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized" >&2
        return 1
    fi

    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local publish_rate=$(calculate_publish_rate)

    # Acquire lock
    if ! acquire_lock "$EVENTS_FILE"; then
        echo "Error: Failed to acquire event bus lock" >&2
        return 1
    fi

    local temp_file="/tmp/event_metrics.$$.tmp"

    case "$operation" in
        publish)
            jq --arg now "$timestamp" \
               --argjson rate "$publish_rate" \
               '.metrics.publish_rate_per_minute = $rate |
                .metrics.last_updated = $now' \
               "$EVENTS_FILE" > "$temp_file"
            ;;
        subscribe)
            # publish_rate updated on publish, subscribe just updates timestamp
            jq --arg now "$timestamp" \
               '.metrics.last_updated = $now' \
               "$EVENTS_FILE" > "$temp_file"
            ;;
        deliver)
            local delivery_latency=$(calculate_delivery_latency)
            jq --arg now "$timestamp" \
               --argjson latency "$delivery_latency" \
               '.metrics.average_delivery_latency_ms = $latency |
                .metrics.last_updated = $now' \
               "$EVENTS_FILE" > "$temp_file"
            ;;
        *)
            echo "Error: Unknown operation '$operation'" >&2
            rm -f "$temp_file"
            release_lock
            return 1
            ;;
    esac

    if [ $? -ne 0 ]; then
        echo "Error: Failed to update metrics" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    # Atomic write
    if ! atomic_write_from_file "$EVENTS_FILE" "$temp_file"; then
        echo "Error: Failed to write metrics update" >&2
        rm -f "$temp_file"
        release_lock
        return 1
    fi

    rm -f "$temp_file"

    # Release lock
    release_lock

    return 0
}

# Get event metrics
# Arguments: none
# Returns: JSON with current metrics
get_event_metrics() {
    if [ ! -f "$EVENTS_FILE" ]; then
        echo "Error: Event bus not initialized" >&2
        return 1
    fi

    # Get base metrics
    local base_metrics=$(jq '.metrics' "$EVENTS_FILE")

    # Calculate real-time metrics
    local publish_rate=$(calculate_publish_rate)
    local delivery_latency=$(calculate_delivery_latency)
    local backlog_count=$(jq -r '.metrics.backlog_count' "$EVENTS_FILE")
    local total_subscribers=$(jq -r '.subscriptions | length' "$EVENTS_FILE")

    # Combine base and real-time metrics
    jq -n \
        --argjson base "$base_metrics" \
        --argjson rate "$publish_rate" \
        --argjson latency "$delivery_latency" \
        --argjson backlog "$backlog_count" \
        --argjson subscribers "$total_subscribers" \
        '{
            total_published: $base.total_published,
            total_subscriptions: $base.total_subscriptions,
            total_delivered: $base.total_delivered,
            total_subscribers: $subscribers,
            publish_rate_per_minute: $rate,
            average_delivery_latency_ms: $latency,
            backlog_count: $backlog,
            last_updated: $base.last_updated
        }'
}

# Get metrics summary (human-readable)
# Arguments: none
# Returns: Formatted metrics summary
get_metrics_summary() {
    local metrics=$(get_event_metrics)

    echo "=== Event Bus Metrics ==="
    echo "Total Published: $(echo "$metrics" | jq -r '.total_published')"
    echo "Total Subscriptions: $(echo "$metrics" | jq -r '.total_subscriptions')"
    echo "Total Subscribers: $(echo "$metrics" | jq -r '.total_subscribers')"
    echo "Total Delivered: $(echo "$metrics" | jq -r '.total_delivered')"
    echo "Publish Rate: $(echo "$metrics" | jq -r '.publish_rate_per_minute') events/min"
    echo "Avg Delivery Latency: $(echo "$metrics" | jq -r '.average_delivery_latency_ms') ms"
    echo "Backlog: $(echo "$metrics" | jq -r '.backlog_count') events"
    echo "Last Updated: $(echo "$metrics" | jq -r '.last_updated')"
    echo "========================="
}

# Export functions
export -f calculate_publish_rate calculate_delivery_latency update_event_metrics get_event_metrics get_metrics_summary
