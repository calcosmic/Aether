# Phase 29: Colony Intelligence & Quality Signals - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Colony produces calibrated quality assessments, adapts its overhead to project size, and leverages multiple perspectives during colonization. Covers: multi-ant colonization (INT-01), aggressive wave parallelism (INT-03), Phase Lead auto-approval (INT-04), watcher scoring rubric (INT-05), colony overhead adaptation (INT-07), and adaptive complexity mode (ARCH-03).

</domain>

<decisions>
## Implementation Decisions

### Multi-colonizer synthesis
- 3 colonizer ants review the codebase independently during `/ant:colonize`
- When colonizers disagree on a finding, flag the disagreement explicitly in the synthesis report — present both views and let the user decide
- The synthesized report replaces the current single-colonizer output format — user sees one unified synthesis report, individual colonizer reports stored internally but not displayed
- Whether colonizers have distinct specialization lenses or review from the same angle is Claude's discretion

### Wave parallelism strategy
- Phase Lead shows the wave structure to the user before executing — user sees which tasks are in which wave and the parallelism strategy
- If a parallel wave produces a conflict (two workers modify the same file unexpectedly), halt the build and report the conflict — let the user decide how to proceed
- How aggressively to parallelize (file-based grouping vs default-parallel) is Claude's discretion
- How to detect conceptual dependencies beyond file overlap is Claude's discretion

### Claude's Discretion
- Colonizer specialization strategy (distinct lenses vs same scope)
- Parallelism aggression level and independence detection heuristics
- Watcher scoring rubric dimensions and weights (not discussed — Claude designs based on research)
- Complexity thresholds for LIGHTWEIGHT/STANDARD/FULL modes (not discussed — Claude designs based on research)
- Auto-approval threshold for Phase Lead (not discussed — Claude designs based on research)

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. User trusts Claude's judgment on scoring rubric design, complexity thresholds, and mode boundaries.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 29-colony-intelligence*
*Context gathered: 2026-02-04*
