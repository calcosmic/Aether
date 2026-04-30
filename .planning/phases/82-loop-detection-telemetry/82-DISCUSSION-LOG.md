# Phase 82: Loop Detection Telemetry - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-30
**Phase:** 82-loop-detection-telemetry
**Areas discussed:** Event topic design, Payload structure, Status display integration, Query scope and retention

---

## Event topic design

| Option | Description | Selected |
|--------|-------------|----------|
| Single consolidated topic | One topic (ceremony.loop.break). Simpler to query/subscribe. Loop_type in payload distinguishes events. | ✓ |
| Per-type topics | Separate topics per loop type (ceremony.loop.watcher_skip, etc.). Matches existing ceremony convention. | |
| Per-command group topics | Group by source command (ceremony.continue.loop_break, etc.). | |

**User's choice:** Single consolidated topic
**Notes:** Follows ceremony naming convention (ceremony.{domain}.{event}). Loop_type field in payload handles differentiation.

---

## Payload structure

| Option | Description | Selected |
|--------|-------------|----------|
| Extend CeremonyPayload | Add loop_type, detection_signal, action_taken fields to existing struct. Reuses 15+ existing fields. | ✓ |
| New dedicated payload | New LoopBreakPayload struct with loop-specific fields plus common context. | |

**User's choice:** Extend CeremonyPayload
**Notes:** Three new omitempty fields. Backward compatible. One payload type to maintain.

---

## Status display integration

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated section | New "Loop Safety" section in dashboard. Always visible when events exist. | ✓ |
| Integrated into Warnings | Loop events appear alongside stale-state and failed-phase warnings. | |
| On-demand only (flag) | Only visible with `aether status --loop-events`. | |

**User's choice:** Dedicated section
**Notes:** Positioned between Warnings and Progress. Summary line + compact table. Omitted entirely when no events.

---

## Query scope and retention

| Option | Description | Selected |
|--------|-------------|----------|
| Last 5 events, 7-day window | Simple, doesn't overwhelm dashboard. Standard 30-day TTL. | ✓ |
| Last 10 events, full TTL window | More visibility, more dashboard space. | |
| Last 5 events, extended 90-day TTL | Longer retention for safety-relevant events. | |

**User's choice:** Last 5 events, 7-day window
**Notes:** 7-day window is a query filter, not a retention policy. Standard TTL keeps things simple.

---

## Claude's Discretion

- Exact section ordering within the status dashboard
- Visual formatting of the loop safety table
- Whether to add a --loop-events flag for extended output
- Internal helper function naming and file organization

## Deferred Ideas

None.
