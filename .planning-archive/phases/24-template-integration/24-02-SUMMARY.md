---
phase: 24-template-integration
plan: 02
subsystem: templates
tags: [templates, seal, entomb, ceremony, jq]

requires:
  - phase: 24-template-integration/24-01
    provides: "Templates created: crowned-anthill.template.md, handoff.template.md, colony-state-reset.jq.template"
provides:
  - "seal.md (Claude Code) reads crowned-anthill.template.md instead of inline heredoc"
  - "seal.md (OpenCode) reads handoff.template.md instead of inline HANDOFF heredoc"
  - "entomb.md (both platforms) uses jq -f colony-state-reset.jq.template"
  - "entomb.md (both platforms) reads handoff.template.md instead of inline HANDOFF heredoc"
  - "crowned-anthill.template.md v2.0 with triumphant mood"
  - "handoff.template.md v2.0 with reflective mood"
affects:
  - 24-template-integration/24-03
  - distribution

tech-stack:
  added: []
  patterns:
    - "hub-first template path resolution: check ~/.aether/system/templates/ before .aether/templates/"
    - "Template-not-found error format: 'Template missing: {name}. Run aether update to fix.'"
    - "jq -f template_path for state reset instead of inline filter"

key-files:
  created: []
  modified:
    - ".aether/templates/crowned-anthill.template.md"
    - ".aether/templates/handoff.template.md"
    - ".claude/commands/ant/seal.md"
    - ".opencode/commands/ant/seal.md"
    - ".claude/commands/ant/entomb.md"
    - ".opencode/commands/ant/entomb.md"

key-decisions:
  - "OpenCode entomb now resets memory fields (instincts, phase_learnings, decisions) matching Claude Code — intentional normalization since wisdom is already promoted to QUEEN.md before reset"
  - "crowned-anthill.template.md and handoff.template.md use distinct emotional registers: triumphant (seal) vs reflective (entomb)"
  - "OpenCode seal.md had no CROWNED-ANTHILL.md write step — plan description was inaccurate; only the HANDOFF heredoc existed and was wired to template"

patterns-established:
  - "Ceremony template fill: LLM reads template, fills {{PLACEHOLDER}} values, removes HTML comment header, writes result with Write tool"
  - "Template path resolution always hub-first with local fallback"

requirements-completed: [WIRE-02, WIRE-03, WIRE-04]

duration: 2min
completed: 2026-02-20
---

# Phase 24 Plan 02: Template Integration — Wire Commands Summary

**Ceremony templates wired into seal.md and entomb.md across both platforms: 5 inline structures replaced with template reads, triumphant/reflective voices established**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-20T00:03:00Z
- **Completed:** 2026-02-20T00:05:08Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Refreshed crowned-anthill.template.md to v2.0 with triumphant narrative voice (achievement framing, proud closing)
- Refreshed handoff.template.md to v2.0 with reflective narrative voice (colony rest, quiet dignity)
- Replaced all 5 inline structures (2 HANDOFF heredocs in seal files, 2 HANDOFF heredocs in entomb files, 2 inline jq filters) with template reads
- Normalized OpenCode entomb to reset memory fields matching Claude Code (both now use the same jq template)

## Task Commits

Each task was committed atomically:

1. **Task 1: Refresh ceremony templates and wire seal.md (both platforms)** - `4ba36a9` (feat)
2. **Task 2: Wire entomb.md jq filter and HANDOFF heredoc (both platforms)** - `bbb884b` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `.aether/templates/crowned-anthill.template.md` - v2.0, triumphant mood: "The anthill stands crowned. The work endures."
- `.aether/templates/handoff.template.md` - v2.0, reflective mood: "A colony's rest" opening, dignity in closing
- `.claude/commands/ant/seal.md` - SEAL_EOF heredoc replaced with crowned-anthill.template.md read instructions
- `.opencode/commands/ant/seal.md` - HANDOFF_EOF heredoc at Step 5.5 replaced with handoff.template.md read instructions
- `.claude/commands/ant/entomb.md` - Inline jq filter and HANDOFF_EOF heredoc replaced with template reads
- `.opencode/commands/ant/entomb.md` - Inline jq filter and HANDOFF_EOF heredoc replaced with template reads

## Decisions Made

- OpenCode entomb previously preserved memory fields (instincts, phase_learnings, decisions) during reset. Now uses the same jq template as Claude Code, which resets them. This is intentional — wisdom is already promoted to QUEEN.md before the reset runs, so resetting is correct behavior.
- OpenCode seal.md had no CROWNED-ANTHILL.md write step in its actual implementation (unlike what the plan description implied). Only the HANDOFF heredoc existed and was wired. The must_haves truth "OpenCode seal.md HANDOFF heredoc also wired to template" is satisfied.
- Two distinct emotional registers confirmed: crowned-anthill = triumphant (colony accomplished something real, proud moment), handoff = reflective (a chapter closing, quiet preservation).

## Deviations from Plan

None — plan executed exactly as written. The plan's description that OpenCode seal.md "has a CROWNED-ANTHILL.md heredoc" was inaccurate about the file's actual structure, but the required wiring work (HANDOFF heredoc only) matched the must_haves truths and was completed correctly.

## Issues Encountered

None. The 2 pre-existing test failures in validate-state.test.js were unaffected (documented in STATE.md as pre-existing known debt).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- All 5 inline structures removed from seal.md and entomb.md (both platforms)
- Templates are the single source of truth for ceremony content
- Hub-first path resolution pattern established for all template lookups
- Ready for Phase 24 Plan 03

---
*Phase: 24-template-integration*
*Completed: 2026-02-20*
