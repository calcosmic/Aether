# Phase 10: Steering Integration - Context

**Gathered:** 2026-03-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Mid-session research control via pheromone signals and configurable strategy. Users can emit FOCUS/REDIRECT/FEEDBACK signals that the oracle reads between iterations and acts on. Users can configure search strategy (breadth-first, depth-first, adaptive) in the wizard before research begins, and set focus areas to prioritize certain aspects.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion

All implementation areas are delegated to Claude's judgment. Researcher and planner have full flexibility on:

- **Signal timing and delivery** — When the oracle checks for signals between iterations, how quickly they take effect, and how in-progress work is handled when a signal arrives
- **Strategy selection UX** — How the user picks breadth-first / depth-first / adaptive in the wizard, whether strategy can change mid-session, and what each strategy feels like in practice
- **Focus area behavior** — How specific focus areas can be, how visibly the oracle shifts priorities, and how conflicts between focus signals and current progress are resolved
- **Signal feedback to user** — How the user knows their signal was received and acted on, what confirmation or status changes they see between iterations

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 10-steering-integration*
*Context gathered: 2026-03-13*
