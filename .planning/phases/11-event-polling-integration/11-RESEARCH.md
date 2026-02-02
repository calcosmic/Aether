# Phase 11: Event Polling Integration - Research

**Researched:** 2026-02-02
**Domain:** Event-driven coordination with pull-based polling
**Confidence:** HIGH

## Summary

Phase 11 integrates event polling into Worker Ant execution flow, enabling asynchronous coordination without persistent processes. The event bus infrastructure (`.aether/utils/event-bus.sh`) is complete with pub/sub, topic filtering, and delivery tracking. The gap is that Worker Ant prompts don't yet call `get_events_for_subscriber()` at execution boundaries.

**Primary recommendation:** Add "Check Events" section to all Worker Ant prompts with caste-specific sensitivity profiles, integrating polling at execution start, after file writes, and after command completion.

## Standard Stack

### Core
| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| event-bus.sh | v1.0 | Pull-based pub/sub event system | Already implemented, tested, verified in Phase 9 |
| get_events_for_subscriber() | v1.0 | Poll for events matching subscriptions | Returns events since last poll, prevents reprocessing |
| mark_events_delivered() | v1.0 | Mark processed events to prevent reprocessing | Updates last_event_delivered timestamp |
| subscribe_to_events() | v1.0 | Register interest in event topics | Records subscription with topic pattern and filter criteria |

### Worker Ant Castes (10 total)
| Caste | Type | Event Sensitivity |
|-------|-------|-------------------|
| colonizer | Base | phase_complete, spawn_request |
| route-setter | Base | phase_complete, task_started |
| builder | Base | task_started, task_completed, error |
| watcher | Base | task_failed, error, task_completed |
| scout | Base | spawn_request, error |
| architect | Base | phase_complete, task_completed |
| security-watcher | Specialist | error, task_failed (security issues) |
| performance-watcher | Specialist | task_completed, error (performance issues) |
| quality-watcher | Specialist | task_completed, error (quality issues) |
| test-coverage-watcher | Specialist | task_completed, error (coverage gaps) |

### Event Topics (6 defined)
| Topic | Description | Subscribers |
|-------|-------------|-------------|
| phase_complete | Phase execution completed | architect, route-setter, colonizer |
| error | Error occurred during execution | all castes (high priority) |
| spawn_request | Request to spawn specialist Worker Ant | colonizer, scout |
| task_started | Worker Ant started executing a task | route-setter, watcher |
| task_completed | Worker Ant completed a task successfully | watcher, architect |
| task_failed | Worker Ant failed to complete a task | watcher, architect |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Pull-based polling | Push-based (WebSocket, long-poll) | Push requires persistent processes; pull works with prompt-based agents |
| Bash event bus | Python Redis/RabbitMQ | Bash implementation is Claude-native, no external dependencies |

**Installation:**
No installation required - event bus already exists at `.aether/utils/event-bus.sh`

## Architecture Patterns

### Recommended Worker Ant Prompt Structure

Each Worker Ant prompt (`.aether/workers/*.md`) should have this execution flow:

```markdown
## Your Workflow

### 0. Check Events (NEW)
Before starting work, poll for relevant events:

```bash
# Source event bus
source .aether/utils/event-bus.sh

# Get events for this Worker Ant
events=$(get_events_for_subscriber "$(basename "$0" .md)" "<caste>")

# Process events if present
if [ "$events" != "[]" ]; then
  echo "⚠️ Received events:"
  echo "$events" | jq -r '.[] | "\(.metadata.timestamp) [\(.topic)] \(.type)"'

  # Take action based on events
  # (caste-specific response logic)
fi

# Mark events as delivered
mark_events_delivered "$(basename "$0" .md)" "<caste>" "$events"
```

### 1. Receive Task
[existing content]

### 2. Understand Current State
[existing content]
```

### Pattern 1: Event Polling at Execution Boundaries

**What:** Poll events at three key points in Worker Ant execution
**When to use:** All Worker Ant castes at all execution boundaries

**Boundaries:**
1. **Execution start:** Before starting any task
2. **After file writes:** After writing/modifyng files
3. **After command completion:** After running bash commands

**Example:**
```bash
# After writing code
echo "✅ Code written: $file"

# Check for new events
events=$(get_events_for_subscriber "builder" "builder")
if [ "$events" != "[]" ]; then
  # Check if task was cancelled or redirected
  cancelled=$(echo "$events" | jq -r '[.[] | select(.type == "cancelled")] | length')
  if [ "$cancelled" -gt 0 ]; then
    echo "⚠️ Task cancelled, stopping work"
    return 0
  fi
fi
mark_events_delivered "builder" "builder" "$events"
```

### Pattern 2: Caste-Specific Event Sensitivity

**What:** Different castes prioritize different events based on their role
**When to use:** All castes, but with different event subscriptions

**Implementation:**
- Subscribe to relevant topics during colony initialization
- Filter events by caste-specific criteria
- Prioritize event response based on caste sensitivity profile

**Example subscriptions:**
```bash
# Builder Ant subscribes to task events
subscribe_to_events "builder" "builder" "task_started" '{"phase": "current"}'
subscribe_to_events "builder" "builder" "error" '{}'

# Watcher Ant subscribes to completion and failure events
subscribe_to_events "watcher" "watcher" "task_completed" '{}'
subscribe_to_events "watcher" "watcher" "task_failed" '{}'
subscribe_to_events "watcher" "watcher" "error" '{}'
```

### Pattern 3: Event-Driven Caste Coordination

**What:** Castes coordinate via events without direct messaging
**When to use:** When multiple castes need to react to the same event

**Example flow:**
1. Builder Ant completes task → publishes `task_completed` event
2. Watcher Ant receives event → starts verification
3. Architect Ant receives event → updates memory compression
4. Route-setter Ant receives event → updates phase progress

### Anti-Patterns to Avoid

- **Polling in tight loops:** Only poll at execution boundaries, not in loops
- **Ignoring events:** Never skip event polling - always check and mark delivered
- **Event processing without marking:** Always call `mark_events_delivered()` after processing
- **Blocking on events:** Events are async hints, not synchronous commands
- **Duplicate subscriptions:** Check existing subscriptions before subscribing again

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Event pub/sub | Custom notification system | event-bus.sh | Already has atomic writes, file locking, topic filtering |
| Event filtering | Custom filter logic | subscribe_to_events() with filter_criteria | Built-in JSON filtering, tested |
| Delivery tracking | Custom delivery state | mark_events_delivered() | Prevents reprocessing with timestamp tracking |
| Event history | Custom event log | get_event_history() | Built-in history with filtering |
| Event metrics | Custom metrics tracking | event-metrics.sh | Publish rate, delivery latency, backlog tracking |

**Key insight:** The event bus is production-ready with 47/47 event truths verified in Phase 9. Don't reimplement pub/sub, just integrate polling into Worker Ant prompts.

## Common Pitfalls

### Pitfall 1: Not Sourcing event-bus.sh

**What goes wrong:** Functions not found, bash errors
**Why it happens:** Worker Ant prompts are markdown, not bash scripts
**How to avoid:** Always source event-bus.sh before calling event functions
**Warning signs:** `command not found: get_events_for_subscriber`

### Pitfall 2: Incorrect Subscriber ID

**What goes wrong:** Events not delivered to correct Worker Ant
**Why it happens:** Subscriber ID must match the subscription
**How to avoid:** Use consistent naming: `$(basename "$0" .md)` or caste name
**Warning signs:** Always getting empty event arrays

### Pitfall 3: Forgetting to Mark Events Delivered

**What goes wrong:** Same events delivered repeatedly
**Why it happens:** `mark_events_delivered()` updates `last_event_delivered` timestamp
**How to avoid:** Always call after processing, even if events array is empty
**Warning signs:** Seeing same events in every poll

### Pitfall 4: Processing Events Synchronously

**What goes wrong:** Worker Ant blocks waiting for events
**Why it happens:** Treating events as commands instead of hints
**How to avoid:** Poll, process if present, continue with task (don't wait)
**Warning signs:** Worker Ant appears to hang

### Pitfall 5: Not Handling Empty Event Arrays

**What goes wrong:** jq errors on empty input
**Why it happens:** `get_events_for_subscriber()` returns `[]` when no events
**How to avoid:** Always check if events != "[]" before processing
**Warning signs:** `Cannot iterate over null (null)` errors

## Code Examples

### Check Events at Execution Start

```bash
# Source event bus
source .aether/utils/event-bus.sh

# Get events for this Worker Ant
my_caste="builder"
my_id="$(basename "$0" .md)"
events=$(get_events_for_subscriber "$my_id" "$my_caste")

# Process events if present
if [ "$events" != "[]" ]; then
  echo "📨 Received $(echo "$events" | jq 'length') events"

  # Check for errors (high priority for all castes)
  error_count=$(echo "$events" | jq -r '[.[] | select(.topic == "error")] | length')
  if [ "$error_count" -gt 0 ]; then
    echo "⚠️ Errors detected - check with Queen before proceeding"
  fi

  # Check for phase completion (architect, route-setter, colonizer)
  if [[ "$my_caste" =~ (architect|route-setter|colonizer) ]]; then
    phase_events=$(echo "$events" | jq -r '[.[] | select(.topic == "phase_complete")]')
    if [ "$phase_events" != "[]" ]; then
      echo "✅ Phase complete - prepare for memory compression"
    fi
  fi
fi

# Always mark events as delivered
mark_events_delivered "$my_id" "$my_caste" "$events"
```

### Subscribe to Event Topics

```bash
# Source event bus
source .aether/utils/event-bus.sh

# Initialize subscriptions for this caste
my_caste="watcher"
my_id="$(basename "$0" .md)"

# Subscribe to relevant topics
subscribe_to_events "$my_id" "$my_caste" "task_completed" '{}'
subscribe_to_events "$my_id" "$my_caste" "task_failed" '{}'
subscribe_to_events "$my_id" "$my_caste" "error" '{}'
subscribe_to_events "$my_id" "$my_caste" "phase_complete" '{}'

echo "✅ Subscribed to event topics for $my_caste"
```

### Publish Event After Task Completion

```bash
# Source event bus
source .aether/utils/event-bus.sh

# After completing a task
publish_event "task_completed" "implementation" \
  '{"task": "Implement auth module", "worker": "builder", "files": ["auth.py"]}' \
  "builder" "builder"

echo "✅ Published task_completed event"
```

### Filter Events by Caste-Specific Criteria

```bash
# Source event bus
source .aether/utils/event-bus.sh

# Security-watcher only cares about security-related errors
subscribe_to_events "security-watcher" "security-watcher" "error" \
  '{"category": "security"}'

# Performance-watcher only cares about performance issues
subscribe_to_events "performance-watcher" "performance-watcher" "error" \
  '{"category": "performance"}'
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Push-based events (WebSocket, long-poll) | Pull-based polling | Phase 9 (v1.0) | Optimal for prompt-based agents that execute and exit |
| Synchronous event delivery | Async event hints | Phase 9 (v1.0) | No blocking, workers can proceed without waiting |
| Central event dispatcher | Pub/sub with topic filtering | Phase 9 (v1.0) | Decoupled communication, no central bottleneck |

**Deprecated/outdated:**
- Persistent daemon processes: Not needed with pull-based polling
- Push notifications: Incompatible with prompt-based agent execution model
- Event-driven orchestration: Replaced by pheromone-based guidance

## Open Questions

1. **Event subscription lifecycle management**
   - What we know: Subscriptions are recorded in events.json
   - What's unclear: When to unsubscribe (phase end? colony shutdown?)
   - Recommendation: Subscribe during colony init, unsubscribe at phase end

2. **Event priority handling**
   - What we know: All events in array are returned together
   - What's unclear: How to prioritize when multiple events received
   - Recommendation: Process error events first, then by caste sensitivity

3. **Event filtering in prompts**
   - What we know: Filter criteria supported in subscribe_to_events()
   - What's unclear: How to express caste-specific filtering in markdown prompts
   - Recommendation: Add caste-specific subscription examples to each prompt

## Sources

### Primary (HIGH confidence)
- .aether/utils/event-bus.sh - Complete event bus implementation (879 lines)
- .aether/pheromone_system.py - Sensitivity profiles for all castes
- .aether/data/worker_ants.json - Worker Ant caste definitions
- .planning/milestones/v1-MILESTONE-AUDIT.md - Audit findings on event integration gap

### Secondary (MEDIUM confidence)
- .aether/workers/*.md - 10 Worker Ant prompt files (existing structure)
- .aether/utils/test-event-*.sh - Event bus test suites (filtering, async, logging)

### Tertiary (LOW confidence)
- None - all findings verified from source code

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - event-bus.sh verified, 47/47 truths passed in Phase 9
- Architecture: HIGH - pull-based polling pattern confirmed optimal for prompt-based agents
- Pitfalls: HIGH - based on audit findings and test suite observations

**Research date:** 2026-02-02
**Valid until:** 30 days (stable infrastructure, no expected changes to event bus)
