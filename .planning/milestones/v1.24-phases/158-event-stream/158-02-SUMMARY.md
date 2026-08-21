---
phase: 158-event-stream
plan: 02
subsystem: event-stream
tags: [go, ndjson, events, runtime, stream]

requires:
  - phase: 158-01
    provides: TypeScript event types, write stub, read stub, and tests
provides:
  - Go event domain types mirroring TS shape
  - Atomic NDJSON writer with mutex-protected persistent stream
  - NDJSON reader with full-file and tail modes
  - Event stream initialization and convenience emitters
  - 11 passing tests covering shape, round-trip, concurrency, tailing, and lifecycle
affects:
  - 158-03 (event integration into existing commands)
  - 159 (end-to-end acceptance)

tech-stack:
  added: []
  patterns:
    - "Go types mirror TS types: same NDJSON shape, different language idioms"
    - "Atomic O_APPEND writes with sync for crash safety"
    - "Polling-based tail with done-channel cancellation"
    - "In-memory stub helpers for testing without file I/O"

key-files:
  created:
    - cmd/event_types.go
    - cmd/event_writer.go
    - cmd/event_reader.go
    - cmd/event_stream.go
    - cmd/events_test.go
  modified: []

key-decisions:
  - "EventPayload is map[string]interface{} on Go side (flexible) vs discriminated union on TS side (typed). Both serialize to the same NDJSON shape."
  - "CreateEventStream persists an open file with per-write sync for crash safety; WriteEvent is one-shot for convenience."
  - "TailEvents uses polling (100ms) rather than fsnotify to avoid dependency and keep stub mode simple."
  - "In-memory buffer helpers (WriteEventToMemory, GetMemoryBuffer) provide a stub mode when file I/O is not desired."

patterns-established:
  - "NDJSON line format: {type, timestamp, payload} — shared contract between Go and TS"
  - "EventStream struct with mutex + closed flag for thread-safe lifecycle"
  - "InitEventStream creates both directory and empty stream file so tailers can start immediately"

requirements-completed:
  - EVENT-03

# Metrics
duration: 8min
completed: 2026-05-24
---

# Phase 158 Plan 02: Go Runtime NDJSON Event Stream Stubs Summary

**Go runtime emits and consumes structured NDJSON events with atomic writes, polling tail, and 11 passing tests — mirroring the TypeScript control plane event shape.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-05-24T14:46:25Z
- **Completed:** 2026-05-24T14:54:45Z
- **Tasks:** 5
- **Files modified:** 5 created

## Accomplishments

- Go event types (`EventType`, `EventLine`, `EventPayload`) with NDJSON marshalling
- Atomic writer (`WriteEvent`, `CreateEventStream`) with mutex-protected persistent stream and disk sync
- NDJSON reader (`ReadEvents`, `TailEvents`) with full-file parse and polling tail mode
- Stream initialization (`InitEventStream`, `GetEventStreamPath`) plus convenience emitters for all event types
- 11 comprehensive tests: shape, write, read, round-trip, format, concurrency, tailing, init, stubs, lifecycle

## Task Commits

Each task was committed atomically:

1. **Task 1: Go Event Types** — `a4d8c9b2` (feat)
2. **Task 2: Go Event Writer** — `25711874` (feat)
3. **Task 3: Go Event Reader** — `56bdfdee` (feat)
4. **Task 4: Event Stream Initialization** — `3a5a7e7a` (feat)
5. **Task 5: Tests** — `18e3163d` (test)

**Plan metadata:** `18e3163d` (test commit includes init fix)

## Files Created/Modified

- `cmd/event_types.go` — Event domain types, constants, constructor, NDJSON helpers
- `cmd/event_writer.go` — Atomic NDJSON writer, persistent EventStream, in-memory stub helpers
- `cmd/event_reader.go` — NDJSON reader, polling tail with done-channel cancellation
- `cmd/event_stream.go` — Stream initialization, directory creation, convenience emitters
- `cmd/events_test.go` — 11 tests covering all read/write paths and edge cases

## Decisions Made

- EventPayload is `map[string]interface{}` on Go side for flexibility, while TS uses a discriminated union. Both produce identical NDJSON lines.
- `CreateEventStream` keeps the file open and syncs on every write for crash safety; `WriteEvent` is one-shot for simple instrumentation.
- `TailEvents` uses 100ms polling instead of fsnotify to avoid a new dependency and keep the stub simple.
- In-memory buffer helpers provide a lightweight stub mode when file I/O is not desired.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] InitEventStream did not create the stream file**
- **Found during:** Task 5 (TestInitEventStreamCreatesDirectory)
- **Issue:** `InitEventStream` created the `.aether/events/` directory but did not touch `current.ndjson`, causing the test to fail when asserting the file exists.
- **Fix:** Added `os.OpenFile` with `O_CREATE` inside `InitEventStream` to ensure the stream file exists before returning.
- **Files modified:** `cmd/event_stream.go`
- **Verification:** `TestInitEventStreamCreatesDirectory` passes
- **Committed in:** `18e3163d` (Task 5 commit)

**2. [Rule 3 - Blocking] Unused import in event_writer.go**
- **Found during:** Task 2 (compilation after writing file)
- **Issue:** `encoding/json` was imported but not used, breaking the build.
- **Fix:** Removed the unused import.
- **Files modified:** `cmd/event_writer.go`
- **Verification:** `go build ./cmd/...` succeeds
- **Committed in:** `25711874` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both fixes necessary for correctness. No scope creep.

## Issues Encountered

- None beyond the two auto-fixed deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Go event stream stubs are complete and tested.
- Ready for Phase 158-03: integrate event emission into existing `cmd/` commands (e.g., `/ant-build`, `/ant-continue`) as proof of concept.
- Ready for Phase 159: end-to-end acceptance verifying Go and TS can read each other's NDJSON output.

---

*Phase: 158-event-stream*
*Completed: 2026-05-24*
