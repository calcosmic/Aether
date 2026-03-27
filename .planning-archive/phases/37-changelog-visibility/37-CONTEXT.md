# Phase 37: Changelog + Visibility - Context

**Gathered:** 2026-02-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Continuous changelog updates and visible memory health. Workers update CHANGELOG.md during work. `/ant:resume` shows recent learnings, failed approaches, and accumulated wisdom. `/ant:status` shows memory health metrics. Human can see colony memory at a glance.

**Requirements:** LOG-01, VIS-01, VIS-02

</domain>

<decisions>
## Implementation Decisions

### CHANGELOG.md Format
- Organized by date, then phase: `## 2026-02-21` with `### Phase 36` subsections under each date
- Not flat log, not purely hierarchical by phase

### Changelog Update Timing
- After each plan completes (not after each task, not after each phase)
- One entry per plan, aggregating all decisions/files from that plan

### Changelog Content (each entry includes)
- Files changed (modified, added, deleted)
- Decisions made during the plan (why this approach)
- What worked/didn't (from Phase 36 midden/failures system)
- Requirements addressed by this plan

### Resume Dashboard Layout
- PRIMARY: "Where am I now" — current phase progress, last task, what's next
- SECONDARY: Learnings, failures, wisdom shown as counts only (not full items)
- Drill-down available if user wants details
- Recency from research: last 5 decisions, last 10 events

### Status Memory Health Display
- **Metrics to show (all four):**
  - Wisdom count (total in QUEEN.md)
  - Pending promotions (observations pending, deferred proposals)
  - Recent failures (from midden/)
  - Last activity (when QUEEN.md updated, last learning captured)
- **Format:** Table format (rows and columns, easy to scan in terminal)

### Claude's Discretion
- Exact table column widths and spacing
- Threshold for "recent" failures (last 5? last 24 hours?)
- Drill-down command for learnings details
- Changelog file location (root or .aether/)

</decisions>

<specifics>
## Specific Ideas

From v4.0 research:
- "Recency-weighted pruning (last 5 decisions, last 10 events)"
- "NEST.md generation: on-demand (in resume.md) not continuous"
- "Phase summaries: 2-3 bullet points per phase"

User preference: Counts with drill-down over full item display — keeps resume focused on "where am I now" without overwhelming.

</specifics>

<deferred>
## Deferred Ideas

From original v4.0 research, deferred to future phases:
- TRAILS/decision history — full structured decision archive
- BROOD/phase-index.json — quick-reference phase completion status
- Auto-NEST-update on significant events
- Cross-colony search in ROYAL-CHAMBER/

</deferred>

---

*Phase: 37-changelog-visibility*
*Context gathered: 2026-02-21*
