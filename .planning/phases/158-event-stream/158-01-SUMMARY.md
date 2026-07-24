---
phase: 158-event-stream
plan: "01"
subsystem: control-ts
tags: [events, ndjson, typescript, stubs]
dependency_graph:
  requires: []
  provides: [EVENT-01, EVENT-02]
  affects: [control-ts/src/index.ts]
tech_stack:
  added: []
  patterns: [discriminated unions, async generators, in-memory buffers, barrel exports]
key_files:
  created:
    - control-ts/src/events/types.ts
    - control-ts/src/events/writeEvent.ts
    - control-ts/src/events/readEvents.ts
    - control-ts/src/events/index.ts
    - control-ts/tests/events/events.test.ts
  modified:
    - control-ts/src/index.ts
decisions:
  - "Used discriminated union for EventPayload so each event type carries its own shape"
  - "In-memory Map keyed by streamPath simulates file I/O without touching disk"
  - "tailEvents uses polling with configurable interval; real fs.watch can replace later"
  - "Console logging in stub mode gives visibility during development"
metrics:
  duration: "~4 minutes"
  completed_date: "2026-05-24"
---

# Phase 158 Plan 01: NDJSON Event Stream Stubs Summary

One-liner: Type-safe NDJSON event stream stubs with discriminated payload types, in-memory read/write, and full test coverage.

## What Changed

Created the event stream module in `control-ts/src/events/` that defines how the TypeScript control plane emits and consumes structured events. No actual file I/O happens yet — everything is in-memory so tests are fast and isolated.

## Files Created

| File | Purpose |
|------|---------|
| `control-ts/src/events/types.ts` | EventType union, discriminated EventPayload, EventLine interface, createEventLine helper, EventWriter/EventReader types |
| `control-ts/src/events/writeEvent.ts` | writeEvent, createEventStream, getBuffer, clearBuffer |
| `control-ts/src/events/readEvents.ts` | readEvents generator, tailEvents polling generator |
| `control-ts/src/events/index.ts` | Barrel re-export of all public types and functions |
| `control-ts/tests/events/events.test.ts` | 9 tests covering shape, timestamps, write, read, tail, round-trip, NDJSON format |

## Files Modified

| File | Change |
|------|--------|
| `control-ts/src/index.ts` | Added event exports to the public package API |

## Verification

- All 9 tests pass (`npm test -- tests/events/events.test.ts`)
- No TypeScript errors in the new event files

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

- [x] `control-ts/src/events/types.ts` exists
- [x] `control-ts/src/events/writeEvent.ts` exists
- [x] `control-ts/src/events/readEvents.ts` exists
- [x] `control-ts/src/events/index.ts` exists
- [x] `control-ts/tests/events/events.test.ts` exists
- [x] `control-ts/src/index.ts` modified
- [x] Commits verified in git log
