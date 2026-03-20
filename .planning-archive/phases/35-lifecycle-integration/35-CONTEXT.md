# Phase 35: Lifecycle Integration - Context

**Gathered:** 2026-02-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Add wisdom extraction to lifecycle boundaries (seal.md and entomb.md) so colonies contribute their learnings to QUEEN.md before being completed or archived. Addresses INT-04 (seal.md promotes final colony wisdom) and INT-05 (entomb.md promotes wisdom before archiving).

</domain>

<decisions>
## Implementation Decisions

### Approval Flow
- **Require approval** — same tick-to-approve UI as continue.md
- **Block until approved** — wisdom must be handled before seal/entomb ceremony proceeds
- **Same full UI** — checkboxes, threshold bars, preview/confirm (not abbreviated)
- **Both boundaries require approval** — seal and entomb have identical approval requirements

### What to Extract
- **All pending proposals** from learning-observations.json
- **Proposals only** — no auto-generated colony summary
- **Keep deferred items** — persist for future sessions (don't clear at lifecycle boundary)
- **Show message if empty** — "No wisdom proposals to review" then proceed with ceremony

### Integration Timing
- **Before ceremony** — wisdom extraction first, celebration second
- **AI prompts user within command** — not a separate command the user runs
- **Both continue and lifecycle boundaries show proposals** — continue handles phase-end, seal/entomb do final check
- **All pending with highlighting** — show new vs deferred status visually

### Consistency
- **Same code path** — shared function for both seal.md and entomb.md
- **Reuse existing functions** where possible (learning-display-proposals, learning-approve-proposals)

### Claude's Discretion
- Exact implementation approach (refactor vs new wrapper vs direct reuse)
- Highlighting format for new vs deferred proposals
- Error handling if wisdom extraction fails

</decisions>

<specifics>
## Specific Ideas

- User shouldn't have to run separate commands — AI prompts within the same seal/entomb flow
- The "step" is a checkpoint in the AI's flow, not a user command
- Wisdom is permanent — user should always have final say at lifecycle boundaries

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 35-lifecycle-integration*
*Context gathered: 2026-02-21*
