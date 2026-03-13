# Phase 6: State Architecture Foundation - Context

**Gathered:** 2026-03-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace the oracle's flat markdown append with structured, machine-readable state files that bridge context between stateless iterations. Oracle creates and maintains JSON state files and human-readable markdown summaries so each iteration builds on the last.

</domain>

<decisions>
## Implementation Decisions

### Research plan display
- Executive summary style — big picture only (topic, overall status, key findings, what's next)
- Not a detailed dashboard — a few lines the user can scan quickly
- research-plan.md is the human-readable entry point

### Topic breakdown
- Initial decomposition is mostly fixed — oracle sets up the plan at the start and sticks to it
- Oracle should NOT keep adding new sub-questions mid-research; it works through the original plan
- If a sub-question turns out to be irrelevant, remove it from the plan entirely (don't leave it marked as skipped)
- Questions that don't produce useful results should be cleaned out, not accumulated

### File organization
- State files stay in `.aether/oracle/` — consistent with existing oracle location
- Previous research sessions are archived (e.g., `oracle/archive/`) so past research is recoverable
- New sessions overwrite active state files, but old sessions are preserved in archive

### Claude's Discretion
- Update frequency for research-plan.md (every iteration vs key moments)
- What to emphasize in progress view (findings vs gaps vs both)
- Whether to show the oracle's planned next move
- Confidence representation (labels vs percentages vs hybrid)
- Whether to include an overall progress indicator
- Sub-question granularity (3-4 broad vs 6-8 detailed) — adapt to topic complexity
- Flat vs hierarchical sub-question structure
- Sub-question status levels (binary vs three-state)
- Whether to flag contradictory source information visibly
- File split between the 4 state files and research-plan.md
- Whether research-plan.md serves as a single overview or information stays distributed

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. User wants a clean, scannable summary (not a detailed research dashboard) and a stable research plan that doesn't keep expanding.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 06-state-architecture-foundation*
*Context gathered: 2026-03-13*
