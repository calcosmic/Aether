# Phase 34: Core Command Rewrite - Context

**Gathered:** 2026-02-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Rewrite `build.md` (1,119→~300 lines) and `continue.md` (569→~120 lines) with state updates at start-of-next-command. Build writes minimal "EXECUTING" state, continue detects completed work and updates accordingly. State survives context boundaries.

</domain>

<decisions>
## Implementation Decisions

### State boundary handoff
- Build writes minimal state before spawning: `state='EXECUTING'`, `current_phase=N`, timestamp only
- Claude's discretion on detection mechanism (output file marker vs state field)
- Claude's discretion on orphan state handling (reconcile vs require sequence)
- Claude's discretion on completion data (status only vs with summary)

### Simplification approach
- Balance line targets with preserved functionality — targets are goals not mandates
- Keep ANSI color output — part of the colony identity
- Defer pheromone math simplification to Phase 36 (keep existing code for now)
- Defer worker spawning simplification to Phase 35 (keep current inline structure)

### Auto-continue behavior
- Keep Task spawning pattern — isolation is valuable for context management
- Minimal progress output between phases: just "Phase N complete"
- Claude's discretion on whether to keep `--all` mode
- Claude's discretion on halt condition thresholds

### Output formatting
- Keep banners and decorative boxes — visual identity matters
- Keep pheromone bar display (strength visualization)
- Keep full worker spawn output (colored role, task assignment, one-liner per worker)
- Keep detailed error output with context

### Claude's Discretion
- Detection mechanism for build completion
- Orphan state handling approach
- Completion data detail level
- Whether to preserve `--all` mode
- Halt condition thresholds for auto-continue

</decisions>

<specifics>
## Specific Ideas

- State pattern: Build sets EXECUTING, Continue reconciles on next invocation — this is the core architectural change
- Line targets (~300 build, ~120 continue) are aspirational given that we're keeping colors, banners, and full output formatting
- The goal is removing complexity (state sync issues, inline duplication) not visual identity

</specifics>

<deferred>
## Deferred Ideas

- Pheromone exponential decay → simple TTL: Phase 36
- Worker specification simplification: Phase 35
- Sensitivity matrix removal: Phase 36

</deferred>

---

*Phase: 34-core-command-rewrite*
*Context gathered: 2026-02-06*
