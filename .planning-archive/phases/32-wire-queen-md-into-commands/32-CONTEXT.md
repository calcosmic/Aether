# Phase 32: Wire QUEEN.md into Commands - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire existing queen-* commands into slash commands (init.md, build.md) so workers receive wisdom automatically. This creates the unified colony-prime() function and handles two-level QUEEN.md architecture (global + local).

 Promotion proposals, observation tracking, and user approval UX are in Phases 33-34.

</domain>

<decisions>
## Implementation Decisions

### Worker priming content
- **All three sources:** Workers receive wisdom (QUEEN.md) + pheromones (FOCUS/REDIRECT/FEEDBACK) + instincts
- **Format:** Mixed format — QUEEN.md stays markdown, pheromones stay XML, instincts in markdown
- **QUEEN.md content:** Categories only (Philosophies, Patterns, Redirects, Stack Wisdom, Decrees) — metadata and evolution log excluded from worker context
- **Two-level architecture:** Global ~/.aether/QUEEN.md loads first, then local .aether/QUEEN.md (like CLAUDE.md pattern)
- **Instincts:** Dynamic per colony, stored as a section within QUEEN.md

### Integration pattern
- **colony-prime() function:** New unified function that internally calls queen-read + pheromone-prime
- **build.md integration:** Call colony-prime() once for unified worker context (not multiple separate calls)
- **Fail gracefully:** If sub-functions fail, log warnings but don't crash the build

### Error handling
- **QUEEN.md missing:** Fail hard — stop build with clear error message requiring user to run /ant:init
- **pheromones.json missing:** Silently continue with warning — don't block the build, workers just won't receive pheromone signals
- **Template creation:** init.md creates default QUEEN.md from template (no fallback to running without it)

### Metadata format
- **Format:** JSON inside HTML comment (`<!-- ... -->`)
- **Fields:** version, thresholds (philosophy:5, pattern:3, redirect:2, stack:1, decree:0), stats (counts per category)
- **Storage:** Inline in QUEEN.md HTML comment block (not separate file, not in colony state)

### Claude's Discretion
- Exact placement of colony-prime() call in build.md
- Error message wording for QUEEN.md missing
- How to handle partial template corruption
- Default instincts content for new QUEEN.md files

</decisions>

<specifics>
## Specific Ideas

- "QUEEN.md should feel like CLAUDE.md for — rules and patterns that guide the colony,- Two-level pattern exactly mirrors CLAUDE.md: global first, then project-specific
- Error messages should be actionable, not just "Error: QUEEN.md missing"

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 32-wire-queen-md-into-commands*
*Context gathered: 2026-02-20*
