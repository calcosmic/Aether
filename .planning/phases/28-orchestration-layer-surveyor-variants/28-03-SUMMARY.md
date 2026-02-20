---
phase: 28-orchestration-layer-surveyor-variants
plan: "03"
subsystem: agents
tags: [claude-code, subagents, surveyor, colony, aether]

# Dependency graph
requires:
  - phase: 27-distribution-infrastructure-first-core-agents
    provides: agent format (YAML frontmatter + 8-section XML body), distribution pipeline, PWR standards

provides:
  - aether-surveyor-nest: architecture and directory survey agent writing BLUEPRINT.md and CHAMBERS.md
  - aether-surveyor-disciplines: conventions and testing survey agent writing DISCIPLINES.md and SENTINEL-PROTOCOLS.md
  - aether-surveyor-pathogens: technical debt survey agent writing PATHOGENS.md
  - aether-surveyor-provisions: dependencies and integrations survey agent writing PROVISIONS.md and TRAILS.md

affects:
  - /ant:colonize command (spawns all 4 surveyors)
  - aether-queen (orchestrates surveyors during colonize flow)
  - 28-04 (remaining agent conversions that follow the same 8-section template)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Surveyor agents use Write tool with strict .aether/data/survey/ boundary — locked decision override from roadmap's read-only criteria"
    - "8-section XML template (role, execution_flow, critical_rules, return_format, success_criteria, failure_modes, escalation, boundaries) applied uniformly"
    - "descriptions name output documents and spawn command as routing triggers"

key-files:
  created:
    - .claude/agents/ant/aether-surveyor-nest.md
    - .claude/agents/ant/aether-surveyor-disciplines.md
    - .claude/agents/ant/aether-surveyor-pathogens.md
    - .claude/agents/ant/aether-surveyor-provisions.md
  modified: []

key-decisions:
  - "Surveyors get Write tool (not read-only) — locked decision from 28-CONTEXT.md overriding roadmap's original read-only criteria; write scope restricted to .aether/data/survey/ only"
  - "No Edit tool on any surveyor — they create new survey documents, never edit existing source files"
  - "No Task tool on any surveyor — Claude Code subagents cannot spawn other subagents"
  - "consumption tables ported from OpenCode source into execution_flow to preserve context on how survey docs are used by builders"

patterns-established:
  - "Surveyor pattern: Read/Grep/Glob/Bash/Write tools with Write restricted to .aether/data/survey/"
  - "Surveyor descriptions name the exact documents written and the /ant:colonize spawn source"
  - "OpenCode read_only section becomes boundaries section in Claude Code format"
  - "OpenCode consumption section embedded in execution_flow (not a top-level section)"

requirements-completed:
  - CORE-06
  - CORE-07
  - CORE-08
  - CORE-09

# Metrics
duration: 5min
completed: 2026-02-20
---

# Phase 28 Plan 03: Surveyor Variants Summary

**4 PWR-compliant surveyor subagents ported from OpenCode with Write tool restricted to `.aether/data/survey/` — covering architecture, conventions, tech debt, and dependencies**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-20T08:28:19Z
- **Completed:** 2026-02-20T08:33:11Z
- **Tasks:** 2
- **Files created:** 4

## Accomplishments

- Created all 4 surveyor agents as Claude Code subagents with the standard 8-section XML template
- Each surveyor has Write in tools (locked decision override) with boundaries restricting writes to `.aether/data/survey/` only
- No surveyor has Edit or Task tools — they create survey documents, not edit source
- Survey methodology faithfully ported from OpenCode XML bodies, zero OpenCode bash patterns remaining
- All descriptions name the output documents and `/ant:colonize` as spawn source for correct routing

## Task Commits

Each task was committed atomically:

1. **Task 1: Create Surveyor-Nest and Surveyor-Disciplines agents** - `1a858ab` (feat)
2. **Task 2: Create Surveyor-Pathogens and Surveyor-Provisions agents** - `ab54033` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `.claude/agents/ant/aether-surveyor-nest.md` — Architecture and directory survey; writes BLUEPRINT.md and CHAMBERS.md
- `.claude/agents/ant/aether-surveyor-disciplines.md` — Conventions and testing survey; writes DISCIPLINES.md and SENTINEL-PROTOCOLS.md
- `.claude/agents/ant/aether-surveyor-pathogens.md` — Technical debt survey; writes PATHOGENS.md
- `.claude/agents/ant/aether-surveyor-provisions.md` — Dependencies and integrations survey; writes PROVISIONS.md and TRAILS.md

## Decisions Made

- Surveyors get Write tool (not read-only) per locked decision from 28-CONTEXT.md, overriding the roadmap's original read-only criteria. Write scope is restricted to `.aether/data/survey/` only via the boundaries section.
- No Edit tool on any surveyor — surveyors create new documents, they do not edit existing source files.
- No Task tool on any surveyor — Claude Code subagents cannot spawn other subagents (escalation section replaces spawning behavior).
- OpenCode `<read_only>` section maps to `<boundaries>` section in the Claude Code 8-section template.
- OpenCode `<consumption>` section embedded at the end of `<execution_flow>` to preserve context on how survey docs are used downstream.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All 4 surveyor agents are ready for distribution via `npm install -g .`
- Surveyors are the final set of agents needed by `/ant:colonize` — the colonize command can now spawn nest, disciplines, pathogens, and provisions surveyors as Claude Code subagents
- Phase 28 can continue with remaining agent conversions following the same 8-section pattern

---
*Phase: 28-orchestration-layer-surveyor-variants*
*Completed: 2026-02-20*
