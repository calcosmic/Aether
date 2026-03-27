# Phase 43: Make Learning Flow - Context

**Gathered:** 2026-02-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire the learning pipeline so observations automatically flow to QUEEN.md promotions. The components exist (learning-observe, learning-check-promotion, queen-promote, tick-to-approve UI) — this phase connects them. No new components.

</domain>

<decisions>
## Implementation Decisions

### Proposal Display Mode
- Present proposals one at a time (not batch)
- Minimal format: observation text + approve button
- Three actions available: Approve / Reject / Skip
- After user acts, auto-show the next pending proposal (no manual continue)

### Threshold Flexibility
- Check thresholds at **end of build** (not after each observation)
- Observations accumulate cumulatively across sessions (forever)
- Threshold values: Claude's Discretion (recommend starting with 3)
- Post-promotion observation handling: Claude's Discretion (recommend archive + reset)

### Failure Behavior
- If QUEEN.md write fails: prompt user to retry
- If user declines retry: skip to next proposal, keep failed one pending
- Failed promotions stay pending for next /ant:continue
- If observation recording fails: continue build gracefully, log error

### Claude's Discretion
- Specific threshold values (category-based or uniform)
- What happens to observations after promotion (archive, clear, or mark)
- Exact retry prompt wording
- Error message format

</decisions>

<specifics>
## Specific Ideas

- Flow should feel automatic — user just approves what bubbles up
- Don't interrupt builds with proposal reviews — wait until end
- Failed promotions shouldn't block the colony, just retry later

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 43-make-learning-flow*
*Context gathered: 2026-02-22*
