#!/bin/bash
# Aether Event Bus Utility
# Implements pub/sub event system for colony-wide coordination
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

# Export functions
export -f initialize_event_bus
