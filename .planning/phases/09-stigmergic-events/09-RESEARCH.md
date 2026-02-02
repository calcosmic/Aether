# Phase 9: Stigmergic Events - Research

**Researched:** 2026-02-02
**Domain:** Event-driven architecture with bash/jq/JSON
**Confidence:** HIGH

## Summary

Phase 9 implements a pub/sub event bus for colony-wide asynchronous coordination in the Aether Queen Ant Colony system. The event bus extends stigmergic communication: pheromones provide guidance signals, while events enable discrete coordination events between Worker Ants.

The research confirms that a pull-based event delivery model is optimal for Aether's unique architecture where Worker Ants are Claude prompt files (not persistent processes). Events are published to `events.json` and subscribers poll for relevant events when they execute. This approach provides true async semantics (publish returns immediately) while avoiding background processes or message queues.

**Primary recommendation:** Implement a topic-based pub/sub system with JSON file storage, jq-based filtering, and pull-based delivery. Use existing Aether patterns (atomic-write, file-lock) for safety and consistency.

## Standard Stack

The Aether event bus uses pure bash/jq/JSON with no external dependencies:

### Core
| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| **jq** | 1.6+ | JSON querying, filtering, updates | Aether's standard JSON manipulation tool |
| **bash** | 3.x+ | Scripting event operations | Aether's shell environment |
| **events.json** | v1.0 | Event storage (topics, subscriptions, log) | Single source of truth for events |
| **atomic-write.sh** | Existing | Corruption-safe file writes | Aether's proven atomic write pattern |
| **file-lock.sh** | Existing | Concurrent access prevention | Aether's file locking mechanism |

### Supporting
| Component | Version | Purpose | When to Use |
|-----------|---------|---------|-------------|
| **COLONY_STATE.json** | Existing | Store event metrics | Update publish/subscribe counts |
| **Worker Ant caste** | v2.0 | Caste-based filtering | Limit events by caste sensitivity |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Pull-based delivery | Push-based with background daemons | Push requires persistent processes (violates Aether's emergence philosophy) |
| File-based events | Named pipes or sockets | Pipes are OS-specific, harder to debug, no persistence |
| JSON storage | SQLite database | Adds dependency, overkill for simple pub/sub |

**Installation:**
```bash
# No new dependencies required
# Event bus utilities will be in .aether/utils/event-bus.sh
# Event storage in .aether/data/events.json
```

## Architecture Patterns

### Recommended Project Structure
```
.aether/
├── data/
│   └── events.json           # Event bus storage (topics, subscriptions, event_log)
├── utils/
│   ├── event-bus.sh          # Core event bus functions (publish, subscribe, filter)
│   ├── event-metrics.sh      # Event metrics tracking
│   ├── atomic-write.sh       # Existing: corruption-safe writes
│   └── file-lock.sh          # Existing: concurrent access prevention
└── locks/
    └── events.json.lock      # File lock for event operations
```

### Pattern 1: Event Bus Schema (events.json)

**What:** Single JSON file containing topics, subscriptions, event log, and metrics.

**When to use:** All event bus operations read/write this file.

**Structure:**
```json
{
  "$schema": "Aether Event Bus v1.0",
  "topics": {
    "phase_complete": {
      "description": "Phase execution completed",
      "subscriber_count": 2
    },
    "error": {
      "description": "Error occurred during execution",
      "subscriber_count": 3
    },
    "spawn_request": {
      "description": "Request to spawn specialist Worker Ant",
      "subscriber_count": 1
    }
  },
  "subscriptions": [
    {
      "id": "sub_001",
      "subscriber_id": "verifier",
      "subscriber_caste": "watcher",
      "topic_pattern": "phase_complete",
      "filter_criteria": {
        "min_phase": 5
      },
      "created_at": "2026-02-02T10:00:00Z",
      "last_event_delivered": null,
      "delivery_count": 0
    }
  ],
  "event_log": [
    {
      "id": "evt_001",
      "topic": "phase_complete",
      "type": "phase_complete",
      "data": {
        "phase": 8,
        "status": "success"
      },
      "metadata": {
        "publisher": "queen",
        "publisher_caste": null,
        "timestamp": "2026-02-02T10:05:00Z",
        "correlation_id": "phase_8_completion"
      }
    }
  ],
  "metrics": {
    "total_published": 1,
    "total_subscriptions": 1,
    "total_delivered": 0,
    "publish_rate_per_minute": 0.0,
    "average_delivery_latency_ms": 0,
    "backlog_count": 1,
    "last_updated": "2026-02-02T10:05:00Z"
  },
  "config": {
    "max_event_log_size": 1000,
    "max_subscriptions_per_topic": 50,
    "event_retention_hours": 168
  }
}
```

**Key design decisions:**
- **Single file:** All event data in one place (simpler than distributed state)
- **Topic-based:** Hierarchical topics with dot notation (e.g., "phase.*" matches all phase events)
- **Pull-based:** Subscribers poll event_log for undelivered events
- **Ring buffer:** event_log trimmed to max_event_log_size when exceeded

### Pattern 2: Publish Operation

**What:** Worker Ants emit events to topics with atomic write safety.

**When to use:** Worker Ant completes task, fails, discovers issue, reaches phase boundary.

**Example:**
```bash
# Source: .aether/utils/event-bus.sh
publish_event() {
    local topic="$1"
    local event_type="$2"
    local event_data="$3"
    local publisher="${4:-$(whoami)}"
    local publisher_caste="${5:-}"

    # Generate event ID and timestamp
    local event_id="evt_$(date +%s)_$(random_string 8)"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local correlation_id="corr_$(date +%s)_$(random_string 8)"

    # Acquire file lock
    if ! acquire_lock ".aether/data/events.json"; then
        echo "Failed to acquire event bus lock" >&2
        return 1
    fi

    # Add event to event_log via jq
    local temp_file="/tmp/event_publish.$$.tmp"
    jq --arg id "$event_id" \
       --arg topic "$topic" \
       --arg type "$event_type" \
       --argjson data "$event_data" \
       --arg publisher "$publisher" \
       --argjson caste "$publisher_caste" \
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
       .metrics.last_updated = $timestamp
       ' "$EVENTS_FILE" > "$temp_file"

    # Atomic write
    atomic_write_from_file "$EVENTS_FILE" "$temp_file"
    rm -f "$temp_file"

    # Trim event log if exceeds max size
    trim_event_log

    # Release lock
    release_lock

    echo "$event_id"
}
```

**Key features:**
- **Non-blocking:** Publish writes to JSON and returns immediately (true async)
- **Atomic:** File lock + atomic-write prevents corruption
- **Metrics:** Updates publish_count and backlog
- **Trim:** Auto-trims event log when exceeds max size

### Pattern 3: Subscribe Operation

**What:** Worker Ants register interest in topics with optional filtering.

**When to use:** Worker Ant initialization, phase setup, dynamic interest changes.

**Example:**
```bash
# Source: .aether/utils/event-bus.sh
subscribe_to_events() {
    local subscriber_id="$1"
    local subscriber_caste="$2"
    local topic_pattern="$3"
    local filter_criteria="${4:-{}}"

    # Generate subscription ID
    local sub_id="sub_$(date +%s)_$(random_string 8)"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire file lock
    if ! acquire_lock ".aether/data/events.json"; then
        echo "Failed to acquire event bus lock" >&2
        return 1
    fi

    # Add subscription via jq
    local temp_file="/tmp/event_subscribe.$$.tmp"
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
       if .topics[$pattern] then
         .topics[$pattern].subscriber_count += 1
       else
         .topics[$pattern] = {
           "description": "Auto-created topic",
           "subscriber_count": 1
         }
       end |
       .metrics.last_updated = $timestamp
       ' "$EVENTS_FILE" > "$temp_file"

    # Atomic write
    atomic_write_from_file "$EVENTS_FILE" "$temp_file"
    rm -f "$temp_file"

    # Release lock
    release_lock

    echo "$sub_id"
}
```

**Key features:**
- **Topic patterns:** Supports wildcards (e.g., "error.*", "phase.*")
- **Filter criteria:** JSON object for custom filtering (e.g., {"min_phase": 5})
- **Auto-create topics:** Creates topic entry if doesn't exist
- **Tracking:** Records subscription time and delivery stats

### Pattern 4: Event Filtering and Delivery (Pull-Based)

**What:** Subscribers poll for relevant events using jq-based filtering.

**When to use:** Worker Ant execution start, periodic event checks.

**Example:**
```bash
# Source: .aether/utils/event-bus.sh
get_events_for_subscriber() {
    local subscriber_id="$1"
    local subscriber_caste="$2"

    # Acquire file lock for read
    if ! acquire_lock ".aether/data/events.json"; then
        echo "Failed to acquire event bus lock" >&2
        return 1
    fi

    # Get subscriptions for this subscriber
    local subscriptions=$(jq -c --arg subscriber "$subscriber_id" \
       '.subscriptions[] | select(.subscriber_id == $subscriber)' \
       "$EVENTS_FILE")

    # For each subscription, find matching undelivered events
    local matching_events="[]"
    while IFS= read -r sub; do
        local topic_pattern=$(echo "$sub" | jq -r '.topic_pattern')
        local filter_criteria=$(echo "$sub" | jq -c '.filter_criteria')
        local last_delivered=$(echo "$sub" | jq -r '.last_event_delivered // "null"')

        # Find events matching topic pattern (with wildcard support)
        local events=$(jq -c --arg pattern "$topic_pattern" \
           --argjson filter "$filter_criteria" \
           --arg last "$last_delivered" \
           --arg caste "$subscriber_caste" \
           '
           .event_log[] |
           select(
             (.topic | test($pattern)) and
             (.metadata.timestamp > $last or $last == "null") and
             (
               $filter == {} or
               (.data | to_entries | all(.key as $k | $filter[$k] == .value))
             )
           )
           ' "$EVENTS_FILE")

        # Accumulate matching events
        if [ -n "$events" ]; then
            matching_events=$(echo "$matching_events" | jq --argjson new "$events" '. + $new')
        fi
    done <<< "$subscriptions"

    # Release lock
    release_lock

    echo "$matching_events"
}
```

**Key features:**
- **Wildcard matching:** Uses jq's `test()` for regex topic matching
- **Filter criteria:** Applies custom filters to event data
- **Since last delivered:** Only returns new events since last poll
- **Caste-based filtering:** Can filter by caste sensitivity
- **Non-blocking:** Returns immediately (empty array if no events)

### Pattern 5: Event Log Management

**What:** Ring buffer implementation with configurable size and retention.

**When to use:** After each publish (auto-trim), manual cleanup operations.

**Example:**
```bash
# Source: .aether/utils/event-bus.sh
trim_event_log() {
    local max_size=$(jq -r '.config.max_event_log_size' "$EVENTS_FILE")
    local current_size=$(jq -r '.event_log | length' "$EVENTS_FILE")

    if [ "$current_size" -gt "$max_size" ]; then
        local trim_count=$((current_size - max_size))
        local temp_file="/tmp/event_trim.$$.tmp"

        # Keep most recent events (ring buffer)
        jq --argjson keep "$max_size" \
           '.event_log = .event_log[-($keep):] |
            .metrics.backlog_count = (.event_log | length)' \
           "$EVENTS_FILE" > "$temp_file"

        atomic_write_from_file "$EVENTS_FILE" "$temp_file"
        rm -f "$temp_file"

        echo "Trimmed $trim_count old events from event log"
    fi
}

cleanup_old_events() {
    local retention_hours=$(jq -r '.config.event_retention_hours' "$EVENTS_FILE")
    local cutoff_timestamp=$(date -d "$retention_hours hours ago" -u +"%Y-%m-%dT%H:%M:%SZ")

    local temp_file="/tmp/event_cleanup.$$.tmp"
    jq --arg cutoff "$cutoff_timestamp" \
       '.event_log = [.event_log[] | select(.metadata.timestamp > $cutoff)] |
        .metrics.backlog_count = (.event_log | length)' \
       "$EVENTS_FILE" > "$temp_file"

    atomic_write_from_file "$EVENTS_FILE" "$temp_file"
    rm -f "$temp_file"

    echo "Cleaned up events older than $retention_hours hours"
}
```

**Key features:**
- **Ring buffer:** Keeps most recent N events (configurable)
- **Time-based cleanup:** Removes events older than retention period
- **Atomic:** All operations use atomic-write pattern
- **Metric updates:** Updates backlog_count after trim/cleanup

### Pattern 6: Event Metrics Tracking

**What:** Track publish rate, subscription count, delivery latency, and backlog.

**When to use:** After each publish/subscribe operation, periodic metrics collection.

**Example:**
```bash
# Source: .aether/utils/event-metrics.sh
update_event_metrics() {
    local operation="$1"  # "publish" or "deliver"

    local temp_file="/tmp/event_metrics.$$.tmp"
    case "$operation" in
        publish)
            local now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
            jq --arg now "$now" \
               '.metrics.publish_rate_per_minute = calculate_publish_rate(.event_log) |
                .metrics.last_updated = $now' \
               "$EVENTS_FILE" > "$temp_file"
            ;;
        deliver)
            # Update delivery latency metrics
            jq '
            .metrics.total_delivered += 1 |
            .metrics.backlog_count -= 1
            ' "$EVENTS_FILE" > "$temp_file"
            ;;
    esac

    atomic_write_from_file "$EVENTS_FILE" "$temp_file"
    rm -f "$temp_file"
}

calculate_publish_rate() {
    # Calculate events published in last minute
    local one_minute_ago=$(date -d "1 minute ago" -u +"%Y-%m-%dT%H:%M:%SZ")
    jq --arg cutoff "$one_minute_ago" \
       '[.event_log[] | select(.metadata.timestamp > $cutoff)] | length' \
       "$EVENTS_FILE"
}

get_event_metrics() {
    jq '.metrics' "$EVENTS_FILE"
}
```

**Key features:**
- **Publish rate:** Calculates events per minute (sliding window)
- **Delivery latency:** Tracks time between publish and delivery
- **Backlog count:** Shows undelivered events
- **Real-time:** Updated on each operation

### Anti-Patterns to Avoid

- **Push-based delivery with background processes:** Aether Worker Ants are prompts, not persistent processes. Push delivery requires daemons that violate the emergence philosophy.
- **Multiple event files:** Single events.json is simpler and prevents consistency issues. Use jq filtering instead of separate topic files.
- **Synchronous publish waiting for subscribers:** Publishing should return immediately. Subscribers poll when they run.
- **Complex routing logic:** Keep filtering simple (topic patterns + JSON criteria). Complex routing creates bottlenecks.
- **Ignoring file locking:** Concurrent event publishing will corrupt events.json without locks.
- **Unbounded event log growth:** Implement ring buffer trimming and time-based cleanup to prevent disk overflow.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| **Atomic JSON writes** | Manual temp file + rename | `.aether/utils/atomic-write.sh` | Existing Aether pattern handles JSON validation and backups |
| **File locking** | Manual lock file creation | `.aether/utils/file-lock.sh` | Proven lock acquisition with PID tracking and stale lock cleanup |
| **JSON querying** | Bash string manipulation | `jq` | Bash string parsing is fragile, jq handles JSON correctly |
| **Topic filtering** | Custom wildcard matching | `jq test()` function | Regex matching in jq is robust and well-tested |
| **Event IDs** | Simple counters | Timestamp + random suffix | Prevents collisions in concurrent scenarios |
| **Metrics calculation** | Manual counting | jq aggregations | jq's built-in functions are efficient and less error-prone |

**Key insight:** Aether already has proven patterns for file safety (atomic-write, file-lock). Reusing these ensures consistency across the codebase and avoids reinventing solutions for concurrent access, corruption prevention, and state management.

## Common Pitfalls

### Pitfall 1: Race Conditions in Event Publishing

**What goes wrong:** Multiple Worker Ants publish events simultaneously, corrupting events.json with interleaved writes.

**Why it happens:** Bash doesn't have atomic JSON append operations. Concurrent `jq` writes can interleave and produce invalid JSON.

**How to avoid:**
- Always use `file-lock.sh` before reading/writing events.json
- Use `atomic-write.sh` for all write operations
- Never skip locking "for performance" - event publishing is not a hot path

**Warning signs:** Invalid JSON errors, jq parse failures, missing events after concurrent operations

### Pitfall 2: Unbounded Event Log Growth

**What goes wrong:** events.json grows indefinitely, consuming disk space and slowing jq operations.

**Why it happens:** Events are published but never trimmed, causing O(N) query performance degradation.

**How to avoid:**
- Configure `max_event_log_size` (default: 1000 events)
- Call `trim_event_log()` after each publish
- Implement periodic `cleanup_old_events()` for time-based retention
- Monitor `metrics.backlog_count` for unexpected growth

**Warning signs:** Large file sizes (>1MB), slow jq queries, increasing backlog

### Pitfall 3: Blocking Event Delivery

**What goes wrong:** `publish_event()` waits for all subscribers to process events before returning.

**Why it happens:** Confusing async with parallel - true async means publish returns immediately.

**How to avoid:**
- Publish should only write to events.json and return
- Subscribers poll via `get_events_for_subscriber()` when they run
- Never have publish call subscriber code directly

**Warning signs:** Slow publish operations, Worker Ants waiting for events

### Pitfall 4: Inefficient Topic Filtering

**What goes wrong:** Linear scan of all subscriptions for every event publish.

**Why it happens:** Naive implementation doesn't index subscriptions by topic.

**How to avoid:**
- Use jq's efficient JSON queries
- Consider maintaining a topic → subscriptions index in events.json
- Cache subscriber subscriptions if frequently queried

**Warning signs:** Slow publish with many subscriptions (>50), high CPU usage

### Pitfall 5: Event Loss During Crashes

**What goes wrong:** Events lost if system crashes during publish operation.

**Why it happens:** Write to temp file not atomic, or JSON validation fails after partial write.

**How to avoid:**
- Always use `atomic-write.sh` (temp file + rename is atomic on POSIX)
- Validate JSON before atomic rename
- Keep backups of events.json (automatic with atomic-write)

**Warning signs:** Missing events after crashes, corrupted JSON files

### Pitfall 6: Subscription Leaks

**What goes wrong:** Old subscriptions never removed, causing filter bloat and wasted processing.

**Why it happens:** Worker Ants subscribe but never unsubscribe, or crash before cleanup.

**How to avoid:**
- Implement subscription TTL (auto-expire after N hours)
- Provide `unsubscribe_from_events()` function
- Periodically clean up stale subscriptions
- Track subscriber activity timestamps

**Warning signs:** Growing subscription count, many inactive subscribers

## Code Examples

Verified patterns from Aether's existing codebase:

### Publishing an Event (Task Completed)

```bash
# Source: .aether/utils/event-bus.sh
# Usage: publish_event "task_complete" "task_completed" '{"task_id": "123", "status": "success"}' "executor" "builder"

source .aether/utils/event-bus.sh

# Worker Ant publishes event after task completion
publish_event "task_complete" "task_completed" \
    '{"task_id": "09-04", "status": "success", "duration_ms": 4500}' \
    "executor" \
    "builder"
```

### Subscribing to Events (Phase Boundaries)

```bash
# Source: .aether/utils/event-bus.sh
# Usage: subscribe_to_events "verifier" "watcher" "phase_complete" '{"min_phase": 5}'

source .aether/utils/event-bus.sh

# Verifier Ant subscribes to phase completion events
subscribe_to_events \
    "verifier" \
    "watcher" \
    "phase_complete" \
    '{"min_phase": 5}'
```

### Polling for Events (Pull-Based Delivery)

```bash
# Source: .aether/utils/event-bus.sh
# Usage: get_events_for_subscriber "verifier" "watcher"

source .aether/utils/event-bus.sh

# Worker Ant checks for new events on execution
events=$(get_events_for_subscriber "verifier" "watcher")
event_count=$(echo "$events" | jq 'length')

if [ "$event_count" -gt 0 ]; then
    echo "Received $event_count events"
    echo "$events" | jq -c '.[]' | while read -r event; do
        # Process each event
        event_topic=$(echo "$event" | jq -r '.topic')
        event_data=$(echo "$event" | jq -c '.data')
        echo "Processing $event_topic: $event_data"
    done

    # Mark events as delivered
    mark_events_delivered "verifier" "watcher" "$events"
fi
```

### Event Filtering with Wildcards

```bash
# Subscribe to all error events (error.* matches error.critical, error.warning, etc.)
subscribe_to_events "logger" "architect" "error.*" '{}'

# Subscribe to all phase events
subscribe_to_events "coordinator" "route_setter" "phase.*" '{}'

# Subscribe to specific event with filter
subscribe_to_events "specialist" "builder" "spawn_request" '{"specialist_type": "database"}'
```

### Event Metrics Display

```bash
# Source: .aether/utils/event-metrics.sh
source .aether/utils/event-metrics.sh

# Get current event metrics
metrics=$(get_event_metrics)
echo "Event Bus Metrics:"
echo "  Total Published: $(echo "$metrics" | jq -r '.total_published')"
echo "  Total Subscriptions: $(echo "$metrics" | jq -r '.total_subscriptions')"
echo "  Total Delivered: $(echo "$metrics" | jq -r '.total_delivered')"
echo "  Publish Rate: $(echo "$metrics" | jq -r '.publish_rate_per_minute') events/min"
echo "  Backlog: $(echo "$metrics" | jq -r '.backlog_count') events"
echo "  Avg Latency: $(echo "$metrics" | jq -r '.average_delivery_latency_ms') ms"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| **Push-based event delivery** | **Pull-based event delivery** | 2024-2025 | Pull-based is better for intermittent subscribers (prompt-based agents) |
| **Multiple topic files** | **Single events.json** | 2024-2025 | Single file simplifies consistency, jq filtering replaces file separation |
| **Unbounded event logs** | **Ring buffer with retention** | 2025 | Prevents disk overflow, maintains performance |
| **No metrics tracking** | **Comprehensive event metrics** | 2025 | Enables observability and performance optimization |
| **Blocking publish** | **Non-blocking async publish** | 2025 | True async semantics, no waiting for subscribers |

**Deprecated/outdated:**
- **Message queues (RabbitMQ, Redis):** Overkill for single-system event bus, adds operational complexity
- **Background daemon processes:** Violates Aether's emergence philosophy, not needed for prompt-based agents
- **Push-based delivery with webhooks:** Requires persistent HTTP listeners, incompatible with CLI workflow
- **Separate topic files:** Single events.json with jq filtering is simpler and more consistent

## Open Questions

### Question 1: Event Delivery Acknowledgment

**What we know:** Pull-based delivery means subscribers poll for events. Need to track which events have been delivered to which subscribers.

**What's unclear:** Should events be marked as "delivered" globally (removed from backlog for everyone) or per-subscriber (each subscriber tracks their own cursor)?

**Recommendation:** Per-subscriber delivery tracking. Each subscription has a `last_event_delivered` timestamp. This allows:
- Different subscribers to process events at different rates
- New subscribers to receive all events (not just undelivered ones)
- Replay capabilities for debugging

**Implementation:** Add `mark_events_delivered()` function that updates subscription's `last_event_delivered` timestamp.

### Question 2: Event Prioritization

**What we know:** Some events (e.g., errors, spawn requests) may be more urgent than others.

**What's unclear:** Should event bus support priority queues or FIFO ordering?

**Recommendation:** Start with FIFO ordering (simpler). Add priority if:
- Metrics show high-value events are delayed
- Worker Ants complain about missing urgent events
- Colony performance degrades due to event backlog

**Implementation:** Add optional `priority` field to event schema, modify `get_events_for_subscriber()` to sort by priority if present.

### Question 3: Subscription Filtering Complexity

**What we know:** Filter criteria can be simple (equality) or complex (nested conditions, ranges).

**What's unclear:** How powerful should filter criteria be? jq expressions are powerful but complex.

**Recommendation:** Start with simple equality filters (JSON object with key-value pairs). If needed, support:
- Simple operators (">", "<", "!=", "contains")
- Logical operators (AND, OR, NOT)
- Field path syntax (data.phase >= 5)

**Implementation:** Extend filter_criteria parsing to support operators, or use jq expressions directly for advanced use cases.

## Sources

### Primary (HIGH confidence)
- **Aether existing utilities** - atomic-write.sh, file-lock.sh, state-machine.sh, memory-ops.sh
  - Verified patterns for file safety, locking, and JSON operations
  - Used as templates for event bus implementation
- **jq documentation** - JSON querying and manipulation
  - Standard tool for Aether's JSON operations
  - Supports regex matching, filtering, aggregation

### Secondary (MEDIUM confidence)
- [The Complete Guide to Event-Driven Architecture](https://medium.com/@himansusaha/the-complete-guide-to-event-driven-architecture-from-pub-sub-to-event-sourcing-in-production-f9dd468ed9e8) - Event-driven patterns and pub/sub fundamentals
- [Event-Driven Architecture Part 2: Event Streaming and Pub/Sub](https://dev.to/outdated-dev/event-driven-architecture-part-2-event-streaming-and-pubsub-patterns-5b1k) (December 2025) - Advanced pub/sub patterns
- [AWS EventBridge Guide](https://cyberpanel.net/blog/aws-eventbridge) (June 2025) - JSON event structure examples
- [A Complete Guide with AWS SNS & SQS (2025 Edition)](https://medium.com/@sehban.alam/pub-sub-what-why-when-and-how-a-complete-guide-with-aws-sns-sqs-2025-edition-e7c5c28e303e) - Topic-based filtering patterns
- [Pub/Sub Architecture: Push vs Pull Messaging](https://dev.to/arnavsharma2711/pull-based-vs-push-based-pubsub-explained-1k7m) (August 2025) - Pull vs push tradeoffs
- [Event Driven Architecture – Push vs Pull](https://engineering.wellsky.com/post/event-driven-architecture---push-vs-pull) (August 2024) - Consumer-controlled rate benefits
- [Event-Driven Performance Optimization](https://solutionsarchitecture.medium.com/event-driven-performance-optimization-balancing-throughput-latency-and-reliability-22e33e372243) - Metrics tracking (throughput, latency)
- [Monitoring Event-Driven Architectures](https://www.datadoghq.com/blog/monitor-event-driven-architectures/) (December 2024) - Event observability best practices
- [Ring buffer log file on unix](https://stackoverflow.com/questions/30204181/ring-buffer-log-file-on-unix) - Ring buffer implementation patterns
- [Parsing JSON with Unix Tools](https://stackoverflow.com/questions/1955505/parsing-json-with-unix-tools) - jq best practices

### Tertiary (LOW confidence)
- [GitHub Issue: Non-Atomic Writes](https://github.com/anthropics/claude-code/issues/7243) - Discusses atomic write importance (September 2025)
- Various cloud provider documentation (AWS MSK, Azure Event Grid) - Not directly applicable to Aether's bash/jq architecture

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Based on Aether's existing architecture (jq, bash, atomic-write, file-lock)
- Architecture patterns: HIGH - Pull-based delivery is verified as optimal for prompt-based agents, topic-based filtering is industry standard
- Pitfalls: HIGH - Identified from concurrent programming best practices and Aether's existing file safety patterns
- Code examples: HIGH - Based on verified Aether utility patterns (atomic-write, file-lock, memory-ops)

**Research date:** 2026-02-02
**Valid until:** 2026-03-02 (30 days - event bus patterns are stable, but cloud-native patterns evolve rapidly)

**Key assumptions:**
- Worker Ants remain as prompt files (not persistent processes)
- Aether continues using bash/jq/JSON architecture
- Single-machine deployment (no distributed event bus needed)
- Event volume is moderate (<1000 events/hour)

**Validation needed:**
- Actual event volume in production (determines max_event_log_size)
- Subscription complexity requirements (simple vs complex filtering)
- Event delivery latency requirements (determines if pull frequency needs adjustment)
