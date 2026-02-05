# Phase 32: Polish & Safety Rails - Context

**Gathered:** 2026-02-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Colony maintains codebase hygiene through a report-only organizer/archivist ant and users understand pheromone signals through practical documentation. The archivist reports stale files, dead code, and orphaned configs but never deletes or modifies anything. Pheromone docs explain when and why to use FOCUS, REDIRECT, and FEEDBACK with real scenarios.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion

User delegated all implementation decisions. Claude has full flexibility on:

**Archivist report design:**
- Report structure, sections, severity levels
- Output format (terminal, markdown file, or both)
- How to distinguish actionable findings vs informational observations
- Whether to use existing watcher-ant.md caste or create archivist-specific prompting

**Archivist scan scope:**
- What qualifies as stale, dead, or orphaned
- Which directories and file patterns to check
- Aggressiveness of detection (conservative vs thorough)
- Whether to leverage existing colony data (errors.json, activity log) for signals

**Archivist invocation:**
- Command integration (standalone command vs flag on existing command)
- Whether it auto-runs at project completion or is manual-only
- How it relates to the tech debt report from Phase 30

**Pheromone documentation:**
- Document location and format
- Depth of coverage per pheromone type
- Whether to use real examples from colony test sessions or constructed scenarios
- Structure (reference card vs tutorial vs both)

</decisions>

<specifics>
## Specific Ideas

No specific requirements -- open to standard approaches. The roadmap specifies:
- Archivist is **report-only** with no deletions or modifications
- Pheromone docs should use **practical scenarios drawn from real colony usage**

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 32-polish-safety-rails*
*Context gathered: 2026-02-05*
