---
phase: 21-template-foundation
plan: 02
subsystem: templates
tags: [markdown, templates, seal, entomb, colony-lifecycle]

requires:
  - phase: 20-distribution-simplification
    provides: "Direct .aether/ packaging pipeline — templates live in .aether/templates/"
provides:
  - "crowned-anthill.template.md — Standalone seal ceremony document template"
  - "handoff.template.md — Standalone entomb handoff document template"
affects: [21-03-template-foundation, seal-command, entomb-command]

tech-stack:
  added: []
  patterns:
    - "{{DOUBLE_BRACE}} placeholder convention for markdown templates"
    - "HTML comment metadata headers for template identification and version"

key-files:
  created:
    - ".aether/templates/crowned-anthill.template.md"
    - ".aether/templates/handoff.template.md"
  modified: []

key-decisions:
  - "Matched source heredoc casing exactly (ENTOMBED title, Entombed in body) rather than normalizing"
  - "Static sections (Chamber Contents, Session Note) preserved verbatim from source — no placeholders needed"

patterns-established:
  - "Markdown template naming: {name}.template.md"
  - "Template metadata: HTML comment block with template name, version, and fill instructions"
  - "Multi-line placeholder convention: {{PHASE_RECAP}} for agent-filled sections"

requirements-completed: [TMPL-03, TMPL-04]

duration: 1min
completed: 2026-02-19
---

# Phase 21 Plan 02: Document Templates Summary

**Crowned-anthill and handoff markdown templates extracted from seal.md/entomb.md heredocs into standalone .aether/templates/ files with {{DOUBLE_BRACE}} placeholders**

## Performance

- **Duration:** 1 min
- **Started:** 2026-02-19T21:05:35Z
- **Completed:** 2026-02-19T21:06:52Z
- **Tasks:** 2
- **Files created:** 2

## Accomplishments
- Created crowned-anthill.template.md with all seal ceremony sections (Colony Stats, Phase Recap, Pheromone Legacy, The Work)
- Created handoff.template.md with all entomb sections (Colony Archived, Chamber Location, Colony Summary, Chamber Contents, Session Note)
- Both templates use consistent {{DOUBLE_BRACE}} placeholder convention
- HTML comment metadata headers added for template identification

## Task Commits

Each task was committed atomically:

1. **Task 1: Create crowned-anthill.template.md** - `0651f80` (feat)
2. **Task 2: Create handoff.template.md** - `4e308d7` (feat)

## Files Created/Modified
- `.aether/templates/crowned-anthill.template.md` - Seal ceremony document template with 7 placeholders (GOAL, SEAL_DATE, VERSION, TOTAL_PHASES, PHASES_COMPLETED, COLONY_AGE_DAYS, PROMOTIONS_MADE, PHASE_RECAP)
- `.aether/templates/handoff.template.md` - Entomb handoff document template with 6 placeholders (CHAMBER_NAME, GOAL, PHASES_COMPLETED, TOTAL_PHASES, MILESTONE, ENTOMB_TIMESTAMP)

## Decisions Made
- Matched source heredoc casing exactly rather than normalizing — preserves fidelity with existing command output
- Static sections (Chamber Contents list, Session Note) preserved verbatim from source with no placeholders needed

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 3 template files now exist in .aether/templates/ (QUEEN.md.template from prior work, crowned-anthill.template.md and handoff.template.md from this plan)
- Ready for Plan 03 to wire templates into commands (seal.md and entomb.md integration)
- Source command files (seal.md, entomb.md) were verified unmodified — safe for Plan 03 to proceed

## Self-Check: PASSED

All files verified present, all commits verified in git log.

---
*Phase: 21-template-foundation*
*Completed: 2026-02-19*
