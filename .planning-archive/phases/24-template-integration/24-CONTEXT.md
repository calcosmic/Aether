# Phase 24: Template Integration - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire 5 existing slash commands (init, seal, entomb, build) to read from template files created in Phase 21, instead of having JSON structures and heredoc content hard-coded inline. Requirements: WIRE-01 through WIRE-05.

</domain>

<decisions>
## Implementation Decisions

### Ceremony text
- Refresh ALL template content while wiring — not just swap source, improve the content
- Seal ceremony: warm and narrative, **triumphant** mood — colony has reached its peak, crowned anthill achieved
- Entomb ceremony: warm and narrative, **reflective** mood — colony's story is complete, chapter closing
- Two distinct emotional moments, not the same voice
- Internal templates (colony-state, constraints, worker-result) also refreshed for cohesion

### Inline cleanup
- Old inline JSON/heredocs **completely removed** from command files — single source of truth in templates
- If template file missing: **clear error message** and stop — "Template missing. Run aether update to fix." — don't try to continue
- No fallback to inline, no commented-out backups
- Both Claude Code AND OpenCode commands wired simultaneously — keep in sync, avoid drift

### Claude's Discretion
- Whether to tighten surrounding command code while removing inline content — judge per command, clean what's messy, leave what's fine
- Template loading mechanism (helper function, direct read, etc.)
- Template placeholder filling approach
- Template lookup chain (hub-first was established in Phase 20 for queen-init — extend or adapt as needed)

</decisions>

<specifics>
## Specific Ideas

- Seal = triumphant ("the colony reaches its peak") — like closing out a great season
- Entomb = reflective ("the colony's story is complete") — like the final page of a chapter
- Error messages should guide users to recovery: "Run aether update to fix"

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 24-template-integration*
*Context gathered: 2026-02-20*
