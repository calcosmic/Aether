# Phase 36: Signal Simplification - Context

**Gathered:** 2026-02-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace pheromone exponential decay with simple TTL-based expiration. Signals use timestamps instead of half-life math, priority replaces sensitivity calculations, and expired signals are filtered on read.

</domain>

<decisions>
## Implementation Decisions

### TTL Duration
- User-specified at emit time with sensible default
- Default TTL: until phase completion (not wall-clock based)
- Pause-aware: TTL timer stops when colony is paused, resumes when active
- Track pause duration in state; extend expires_at on resume

### Priority System
- Priority levels affect both display prominence AND worker behavior
- High priority signals checked first, normal signals secondary (order of attention)
- Default priority and whether to include low priority: Claude's discretion

### Expiration Handling
- Show time remaining in status output (e.g., "FOCUS: API layer (12min left)")
- Log expiration events to colony events when signals expire mid-task
- Cleanup strategy (filter vs remove on read): Claude's discretion

### Signal Format
- Track signal source: "user" or "worker:architect" for debugging
- JSON structure (array vs keyed), unique IDs, history preservation: Claude's discretion

### Claude's Discretion
- Flag design for TTL specification at emit time
- Default priority per signal type
- Whether to include low priority level (two vs three levels)
- Expired signal cleanup strategy
- JSON structure design (array vs keyed by type)
- Whether signals need unique IDs
- Whether to preserve signal history

</decisions>

<specifics>
## Specific Ideas

- Pause-aware TTL addresses the workflow problem: if user pauses and signal expires before task completes, guidance would be lost without pause tracking
- Time remaining display helps user decide whether to extend a signal

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 36-signal-simplification*
*Context gathered: 2026-02-06*
