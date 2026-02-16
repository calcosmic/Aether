# Phase 10: Entombment & Egg Laying - Context

**Gathered:** 2026-02-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Colony lifecycle management — users can archive completed colonies (entomb), start fresh colonies (lay eggs), browse history (explore tunnels), and see automatic milestone detection.

</domain>

<decisions>
## Implementation Decisions

### Archive organization
- Archive structure: Claude's discretion (flat vs nested)
- Naming convention: `{goal}-{timestamp}` format (e.g., add-user-auth-2026-02-14T153022Z)
- Metadata preserved (manifest.json): Date, goal, phases completed, decisions, learnings, final state
- Default browsing view: Tree view — hierarchical by milestone

### Entombment behavior
- Transfer method: Copy files, verify manifest, then clean up original (copy-then-verify for safety)
- What gets archived: COLONY_STATE.json + manifest.json only (minimal)
- Confirmation: Always require explicit confirmation
- Failed colonies: Prevent entombment — only completed/collected colonies can be archived

### Fresh colony behavior
- Preservation: Clear progress but preserve pheromones — keep accumulated learnings/decisions
- Spawning from archive: No — entombed colonies are read-only archives only
- Naming: Prompt for new goal when laying eggs (fresh intention)
- Multi-colony: Single active colony only — laying eggs implies destructive transition (archive current first)

### Milestone detection
- Detection: Automatic only — system determines milestone based on state
- Triggers: Claude's discretion (phase completion vs requirements count)
- Names: Hybrid — both themed names and versions
- Display format: "Open Chambers (v3.1.0)" — themed name + version in parentheses

### Claude's Discretion
- Archive folder structure (flat vs nested by milestone)
- Milestone transition logic and triggers
- Exact tree view formatting for tunnel browsing

</decisions>

<specifics>
## Specific Ideas

- "I think multi-colony switching would be amazing" — noted for future phase
- Human-readable archive names with goal included (not just timestamps)
- Want pheromone preservation (learnings carry forward between colonies)

</specifics>

<deferred>
## Deferred Ideas

- Multi-colony switching — support multiple concurrent colonies with switching capability (separate phase, requires careful consideration of implementation)

</deferred>

---

*Phase: 10-entombment-egg-laying*
*Context gathered: 2026-02-14*
