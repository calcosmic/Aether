---
phase: 46-event-bus
plan: 01
subsystem: infra
tags: [go, channels, pubsub, jsonl, ttl, event-bus, crash-recovery]

# Dependency graph
requires:
  - phase: 45-core-storage
    provides: storage.Store with AppendJSONL, ReadJSONL, AtomicWrite operations
provides:
  - Typed event bus with in-memory pub/sub via Go channels
  - JSONL persistence for crash recovery
  - TTL-based event expiration and pruning
  - Topic-based subscription with wildcard pattern matching
  - Query and Replay for historical event retrieval
affects: [47-memory-pipeline, 49-agent-system, 50-cli-commands]

# Tech tracking
tech-stack:
  added: [go-channels, crypto/rand, encoding/json.RawMessage]
  patterns: [pub-sub-via-channels, buffered-subscriber-256, non-blocking-publish, jsonl-persistence, ttl-pruning, atomic-rewrite-cleanup]

key-files:
  created:
    - pkg/events/event.go
    - pkg/events/bus.go
    - pkg/events/bus_test.go
  modified: []

key-decisions:
  - "Timestamp stored as string matching shell format (2006-01-02T15:04:05Z) rather than time.Time for JSON parity"
  - "Subscriber channels buffered at 256 capacity with non-blocking send (drop on full) matching shell slow-consumer behavior"
  - "Subscribers stored as flat slice (not map) since subscriber count is typically small"
  - "Replay uses bubble sort for timestamp ordering since event batches are small"

patterns-established:
  - "Event struct uses string timestamps for JSON serialization parity with shell"
  - "Bus methods handle missing JSONL file gracefully (return empty/zero, no error)"
  - "Cleanup uses atomic rewrite via storage.Store.AtomicWrite"
  - "LoadAndReplay provides crash recovery by broadcasting persisted events to subscribers"

requirements-completed: [EVT-01, EVT-02, EVT-03]

# Metrics
duration: 15min
completed: 2026-04-01
---

# Phase 46 Plan 01: Event Bus Core Summary

**Go channel-based pub/sub with JSONL persistence, wildcard topic matching, TTL pruning, and crash recovery -- 32 tests race-clean**

## Performance

- **Duration:** 15 min
- **Started:** 2026-04-01T21:40:00Z
- **Completed:** 2026-04-01T21:55:01Z
- **Tasks:** 4
- **Files modified:** 3

## Accomplishments
- Event struct with JSON serialization matching shell event-bus.sh output format exactly
- Bus with Publish/Subscribe/Unsubscribe/Close lifecycle using buffered Go channels
- Query, Replay, Cleanup (with dry-run), and LoadAndReplay crash recovery
- 32 comprehensive tests including concurrent publish, race detector clean

## Task Commits

Each task was committed atomically:

1. **Task 1: Event types and Bus struct** - `a488761` (feat)
2. **Tasks 2-4: Bus core, persistence, query/replay/cleanup, test suite** - `91b30d6` (feat, squashed)

**Plan metadata:** pending (docs: complete plan)

## Files Created/Modified
- `pkg/events/event.go` - Event type, Config, TopicMatch, GenerateEventID, FormatTimestamp, ComputeExpiry
- `pkg/events/bus.go` - Bus with Publish, Subscribe, Unsubscribe, Close, Query, Replay, Cleanup, LoadAndReplay
- `pkg/events/bus_test.go` - 32 tests covering all success criteria

## Decisions Made
- Timestamp stored as formatted string (`"2026-04-01T12:00:00Z"`) rather than `time.Time` to match shell JSON output exactly and simplify string-based comparison for expiry checks
- Subscriber channels use capacity 256 buffer with non-blocking send (`select/default`), matching shell behavior where slow consumers never block publishers
- Subscribers stored as flat `[]subscriber` slice rather than `map[string][]chan Event` since typical subscriber counts are small and iteration is fast
- Replay uses bubble sort for timestamp ordering -- adequate for small event batches that fit in a single JSONL file
- Event ID format matches shell exactly: `evt_{unix_timestamp}{4hex_chars}` with crypto/rand providing the random bytes

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Test file writing had encoding issues when using shell heredocs containing Go raw string literals (backtick conflicts). Resolved by using the Write tool directly instead of bash heredocs.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Event bus complete and ready for Phase 47 (Memory Pipeline) which will publish observations as events
- All EVT requirements (EVT-01, EVT-02, EVT-03) satisfied
- Bus API is stable: Publish, Subscribe, Query, Replay, Cleanup, LoadAndReplay

## Self-Check: PASSED

- All 3 source files exist: pkg/events/event.go, pkg/events/bus.go, pkg/events/bus_test.go
- Both commits found: a488761 (Task 1), 91b30d6 (Tasks 2-4 squashed)
- SUMMARY.md exists at expected path

---
*Phase: 46-event-bus*
*Completed: 2026-04-01*
