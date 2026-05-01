# Phase 82: Loop Detection Telemetry - Context

**Gathered:** 2026-04-30
**Status:** Ready for planning

<domain>
## Phase Boundary

All loop-breaking events from Phases 80 and 81 are logged to the colony event bus with structured telemetry, and `/ant-status` surfaces recent loop-break events in a dedicated section.

This phase covers LOOP-06 from REQUIREMENTS.md.

**What this phase delivers:**
- A single consolidated event topic (`ceremony.loop.break`) for all loop-breaking events
- CeremonyPayload extended with loop_type, detection_signal, and action_taken fields
- Emission calls added at each loop-break point (watcher skip, recovery redirect, circuit breaker, cycle rejection, lifecycle recovery)
- A dedicated "Loop Safety" section in `/ant-status` showing the last 5 loop-break events from the past 7 days

**What this phase does NOT deliver:**
- New loop-breaking logic (Phases 80 and 81 already built the breakers)
- Planning depth system (DEPTH-01, Phase 83)
- Verification depth extension (DEPTH-02, Phase 84)

</domain>

<decisions>
## Implementation Decisions

### Event topic design
- **D-01:** Single consolidated topic `ceremony.loop.break` for all loop-breaking events. The loop_type field in the payload distinguishes between watcher_skip, circuit_break, cycle_detected, recovery_redirect, and lifecycle_recovery.
- **D-02:** Follows the ceremony topic naming convention (`ceremony.{domain}.{event}`).

### Payload structure
- **D-03:** Extend the existing `CeremonyPayload` struct with three new fields: `LoopType string`, `DetectionSignal string`, `ActionTaken string` (all `omitempty`). This reuses the existing 15+ context fields (Phase, Status, Message, Caste, etc.) while adding loop-specific semantics.
- **D-04:** Add the new topic constant `CeremonyTopicLoopBreak = "ceremony.loop.break"` to `pkg/events/ceremony.go` and include it in `CeremonyTopics()`.

### Status display integration
- **D-05:** Add a dedicated "Loop Safety" section in the `/ant-status` dashboard, positioned between the Warnings section and the Progress section.
- **D-06:** When loop-break events exist within the query window, display a summary line ("Loop Safety: N events in last 7 days") followed by a compact table with columns: Time, Type, Signal, Action.
- **D-07:** When no loop-break events exist in the window, omit the section entirely (don't show "No loop events").

### Query scope and retention
- **D-08:** Status queries the last 5 loop-break events from the past 7 days.
- **D-09:** Loop events use the standard event bus TTL (30 days). No custom TTL needed — the 7-day status window is a query filter, not a retention policy.

### Emission points
- **D-10:** Add `emitLoopBreakEvent()` calls at each of the five loop-break points established in Phases 80 and 81:
  1. Watcher auto-skip (cmd/codex_continue.go — watcher failure threshold reached)
  2. Recovery redirect (cmd/codex_continue.go — recovery command differs from last invocation)
  3. Circuit breaker trip (cmd/circuit_breaker.go — worker consecutive failures hit threshold)
  4. Cycle detection rejection (cmd/codex_plan.go — plan rejected due to circular dependencies)
  5. Lifecycle recovery menu (cmd/session_cmds.go — error recovery menu displayed)

### Claude's Discretion
- Exact section ordering within the status dashboard
- Visual formatting of the loop safety table (ANSI colors, spacing)
- Whether to add a `--loop-events` flag to status for extended output
- Internal helper function naming and file organization

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — LOOP-06 (loop detection telemetry)

### Event bus infrastructure
- `pkg/events/ceremony.go` — CeremonyPayload struct, topic constants, CeremonyTopics() function
- `pkg/events/bus.go` — Publish, Query, Replay functions (event bus API)
- `pkg/events/event.go` — Event struct, topic matching, ID generation

### Loop breaker emission points
- `cmd/codex_continue.go` — Watcher auto-skip (LOOP-01), recovery redirect (LOOP-02)
- `cmd/circuit_breaker.go` — Circuit breaker trip (LOOP-03)
- `cmd/codex_plan.go` — Cycle detection rejection (LOOP-04)
- `cmd/session_cmds.go` — Lifecycle command error handling (LOOP-05)

### Status command
- `cmd/status.go` — renderDashboard(), computeWarnings(), section layout

### Ceremony emitter
- `cmd/ceremony_emitter.go` — Existing ceremony event emission functions (pattern to follow)

### Prior phase context
- `.planning/phases/80-build-continue-loop-prevention/80-CONTEXT.md` — LOOP-01/02/03 implementation decisions
- `.planning/phases/81-plan-and-lifecycle-loop-safety/81-CONTEXT.md` — LOOP-04/05 implementation decisions

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `CeremonyPayload` struct — 15+ fields including Phase, PhaseName, Caste, Name, Task, Status, Message. Adding 3 new fields preserves backward compatibility (all omitempty).
- `CeremonyTopicBuildCircuitBreak` — Existing topic for circuit breaker events. The new `CeremonyTopicLoopBreak` coexists with this (circuit breaker already emits ceremony events; loop-break is a cross-cutting safety telemetry layer).
- `bus.Query()` — Already supports topic pattern matching with `since` time filter and limit. Perfect for the 7-day / 5-event query.
- `computeWarnings()` in status.go — Pattern for computing and rendering conditional dashboard sections.

### Established Patterns
- Ceremony events: define topic constant → create payload → call emitter → publish to bus
- Status sections: compute data → render if non-empty → skip if empty
- Emitter functions: `emitBuildCeremonyCircuitBreak()` in `cmd/ceremony_emitter.go` — pattern to follow for `emitLoopBreakEvent()`

### Integration Points
- `pkg/events/ceremony.go` — Add `CeremonyTopicLoopBreak` constant, extend `CeremonyPayload`, update `CeremonyTopics()`
- `cmd/ceremony_emitter.go` — Add `emitLoopBreakEvent()` function
- `cmd/status.go` — Add loop safety section to `renderDashboard()`
- `cmd/codex_continue.go` — Emit at watcher auto-skip and recovery redirect points
- `cmd/circuit_breaker.go` — Emit at circuit breaker trip point
- `cmd/codex_plan.go` — Emit at cycle detection rejection point
- `cmd/session_cmds.go` — Emit at lifecycle error recovery menu point

</code_context>

<specifics>
## Specific Ideas

- The `emitLoopBreakEvent()` function should be a single centralized function that all five emission points call, keeping the pattern consistent
- The loop_type field should use snake_case values matching the requirement IDs: watcher_skip, recovery_redirect, circuit_break, cycle_detected, lifecycle_recovery
- Detection signal should be human-readable: "3 consecutive watcher failures", "recovery command differs from last invocation", "circular dependency detected: Task A -> Task B -> Task A"
- The status section should be compact — one summary line + table, not a verbose log dump

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 82-loop-detection-telemetry*
*Context gathered: 2026-04-30*
