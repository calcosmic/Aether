# Phase 35: Worker Simplification - Context

**Gathered:** 2026-02-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Collapse 6 worker specs (architect, builder, colonizer, route-setter, scout, watcher) into single workers.md. Remove sensitivity matrices, simplify spawning protocols. Target: ~200 lines from current 1,866.

</domain>

<decisions>
## Implementation Decisions

### Signal Handling
- Per-role keywords instead of sensitivity matrices
- Each role lists which signal types it responds to (e.g., "Builder: FOCUS, REDIRECT")
- No pheromone math, no threshold calculations, no effective signal strength
- Queen reads signals before spawning; workers just know "if you see X, do Y"

### Shared Sections
- Keep minimal "All Workers" section (~30 lines) at top with:
  - Activity log command pattern (5 lines)
  - Spawn request format and available castes (10 lines)
  - Visual identity pattern (5 lines)
- Remove from all workers:
  - Post-action validation checklists
  - Pheromone math instructions
  - Combination effects tables
  - Event/Memory reading ceremony (workers read what they need, no instructions)

### Output Format
- Minimal standard template for all workers:
  - Task, Status, Summary
  - Files (only if worker touched files)
  - Next Steps / Recommendations (required)
- Remove per-role elaborate templates (Scout's 8-section report, etc.)
- ~5-10 lines per report, not 20+

### Selection Criteria
- Queen uses Claude's judgment to select worker type (no keyword rules)
- Each role includes one-liner "when to use" hint to guide Queen
- Soft role boundaries: workers adapt to what they find
- Builder can do light research; Scout can read code
- Roles are guidance not constraints

</decisions>

<specifics>
## Specific Ideas

- Structure: "All Workers" shared section at top, then per-role sections
- Per-role section should include: purpose (2-3 sentences), when to use (1 line), signals it responds to (list), any role-specific workflow hints
- Total target is ~200 lines from 1,866 — aggressive simplification

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 35-worker-simplification*
*Context gathered: 2026-02-06*
