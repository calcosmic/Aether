# Phase 7: Iteration Prompt Engineering - Context

**Gathered:** 2026-03-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Each oracle iteration reads structured state files (from Phase 6), targets the highest-priority knowledge gap, and writes valid state updates. Research deepens across iterations rather than appending or restating. Phase-aware prompts change behavior based on research lifecycle stage (survey / investigate / synthesize / verify).

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion

All implementation decisions for this phase are delegated to Claude's judgment. The following areas should be resolved during research and planning based on codebase patterns, the Phase 6 state architecture, and best practices:

- **Research lifecycle phases** — How survey / investigate / synthesize / verify phases work, what triggers transitions between them, how prompt behavior changes at each stage
- **Gap targeting strategy** — How the prompt selects what to research next, confidence scoring approach (0-100%), prioritization logic for lowest-confidence open questions
- **State update format** — What each iteration writes back to gaps.md, plan.json, and synthesis.md, how the prompt instructs Claude to produce valid structured updates
- **Depth vs repetition prevention** — How prompts ensure iterations go deeper rather than restating, what constitutes "measurably deeper findings," how the prompt prevents research loops

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. Decisions should be guided by:
- The state file structure established in Phase 6 (state.json, plan.json, gaps.md, synthesis.md)
- The success criteria in ROADMAP.md (gap-targeted research, phase-aware prompts, shrinking gaps, confidence-driven prioritization, measurable depth)
- The `--json-schema` CLI flag availability (with fallback to prompt-based JSON enforcement per STATE.md blocker)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 07-iteration-prompt-engineering*
*Context gathered: 2026-03-13*
