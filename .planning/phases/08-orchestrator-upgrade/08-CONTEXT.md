# Phase 8: Orchestrator Upgrade - Context

**Gathered:** 2026-03-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Upgrade oracle.sh with multi-signal convergence detection, intelligent loop control, and graceful interruption handling. The orchestrator decides when research is complete using structural metrics, produces useful partial results on interruption, and recovers from malformed state.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion

All implementation decisions for this phase are delegated to Claude. The following areas should be guided by the success criteria in ROADMAP.md:

**Convergence signals**
- Which combination of gap resolution rate, novelty rate, and coverage completeness to use
- Threshold values for declaring convergence (start with research recommendations, iterate)
- How to weight multiple signals against each other
- Whether convergence requires all signals or a weighted composite

**Diminishing returns**
- How many low-change iterations trigger strategy change vs synthesis
- What "strategy change" means in practice (e.g., switch phase, broaden/narrow scope)
- How aggressive detection should be (err toward doing more research vs stopping early)

**Interruption handling**
- What the synthesis pass produces on stop signal or max-iterations
- Format and depth of the partial report
- Whether synthesis runs automatically or needs a flag

**Error recovery**
- How to detect and recover from malformed JSON in state files
- Whether recovery is silent-fix or warn-and-continue
- What validation runs after each iteration and what triggers recovery

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. Implementation should follow the success criteria defined in ROADMAP.md Phase 8.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 08-orchestrator-upgrade*
*Context gathered: 2026-03-13*
