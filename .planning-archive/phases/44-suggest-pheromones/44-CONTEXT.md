# Phase 44: Suggest Pheromones - Context

**Gathered:** 2026-02-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Build start triggers code pattern analysis → suggestions displayed with tick-to-approve UI → approved suggestions become pheromones. Reuses existing tick-to-approve pattern. No new UI components.

</domain>

<decisions>
## Implementation Decisions

### Suggestion Timing
- Show suggestions on **every build** (no skipping based on existing pheromones)
- Quick dismiss option to proceed to build without approving
- Exact timing point: Claude's Discretion (choose non-disruptive point)

### Suggestion Types
- All three pheromone types: FOCUS, REDIRECT, FEEDBACK
- Up to 5 suggestions at once
- One-by-one approval (same pattern as learning proposals)
- Dismissed suggestions just disappear (no logging or deferral)

### Analysis Inputs
- Analyze **code patterns** (not git changes or phase context)
- Look for: complexity hotspots, anti-patterns, change frequency — multiple signals
- File scope: Claude's Discretion
- Analysis sophistication: Claude's Discretion (recommend heuristic scoring)

### Frequency Control
- Always show suggestions (every build)
- Cap at 5 suggestions maximum per build
- Avoid duplicate suggestions within same session

### Claude's Discretion
- Exact timing point in build flow (non-disruptive)
- Which files to analyze (colony files, project code, or source only)
- Analysis algorithm sophistication (simple thresholds vs heuristic scoring)
- How to track "already suggested this session" state

</decisions>

<specifics>
## Specific Ideas

- Suggestions should feel helpful, not naggy
- If user dismisses all, build proceeds immediately
- Same one-at-a-time UI as learning proposals (consistent UX)
- Cap of 5 prevents overwhelming the user

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 44-suggest-pheromones*
*Context gathered: 2026-02-22*
