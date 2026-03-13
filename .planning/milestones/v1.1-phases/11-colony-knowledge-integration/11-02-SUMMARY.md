---
phase: 11-colony-knowledge-integration
plan: 02
subsystem: oracle
tags: [oracle, templates, synthesis, wizard, bash]

# Dependency graph
requires:
  - phase: 11-01
    provides: promote_to_colony function and validate-oracle-state template field support
provides:
  - Template-aware build_synthesis_prompt with 5 template case branches
  - Template selection wizard question (Q2) in both oracle commands
  - Template-derived default question pre-population for plan.json
  - Confidence grouping directive in all synthesis output
affects: [11-03-PLAN]

# Tech tracking
tech-stack:
  added: []
  patterns: [case-branch template dispatch in synthesis prompt, template-aware plan.json pre-population]

key-files:
  created: []
  modified:
    - .aether/oracle/oracle.sh
    - .claude/commands/ant/oracle.md
    - .opencode/commands/ant/oracle.md

key-decisions:
  - "Template question inserted as Q2 after Topic, before Depth -- template type informs depth recommendation"
  - "Custom template preserves exact pre-Phase-11 output structure for backward compatibility"
  - "Confidence grouping is a common directive applied to ALL templates including custom"

patterns-established:
  - "Template dispatch via case statement: same pattern as phase directives in build_oracle_prompt"
  - "Template-derived default questions: non-custom templates pre-populate plan.json instead of AI decomposition"

requirements-completed: [COLN-02, OUTP-01, OUTP-03]

# Metrics
duration: 4min
completed: 2026-03-13
---

# Phase 11 Plan 02: Research Strategy Templates Summary

**Template-aware synthesis with 5 research types (tech-eval, architecture-review, bug-investigation, best-practices, custom) driving both question decomposition and output structure**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-13T20:38:35Z
- **Completed:** 2026-03-13T20:42:47Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- build_synthesis_prompt reads template from state.json and emits template-specific output sections via case statement
- Wizard gains template selection as Q2, renumbering existing questions Q3-Q7
- Non-custom templates pre-populate plan.json with predictable default questions
- All templates include confidence grouping directive (high 80%+, medium 50-79%, low <50%)
- OpenCode parity maintained

## Task Commits

Each task was committed atomically:

1. **Task 1: Add template-aware build_synthesis_prompt to oracle.sh** - `8fd7244` (feat)
2. **Task 2: Add template selection question to both wizard commands** - `d635207` (feat)

## Files Created/Modified
- `.aether/oracle/oracle.sh` - Template-aware build_synthesis_prompt with case branches for 5 template types plus confidence grouping
- `.claude/commands/ant/oracle.md` - Template wizard Q2, template in state.json, template-derived plan.json questions, summary display
- `.opencode/commands/ant/oracle.md` - Mirror of all wizard changes for OpenCode parity

## Decisions Made
- Template question placed as Q2 (after Topic, before Depth) because template type informs depth recommendation
- Custom/default template produces byte-identical output to pre-Phase-11 build_synthesis_prompt (minus additive confidence grouping)
- Confidence grouping applied to ALL templates as a common directive after template-specific sections

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Pre-existing lint:sync drift between Claude and OpenCode commands (different emoji formatting, $ARGUMENTS vs $normalized_args pattern) -- expected by-design differences, not introduced by this plan
- Pre-existing test failure in context-continuity.test.js (pheromone compact mode) -- unrelated to template changes

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Template system fully wired: wizard writes template to state.json, build_synthesis_prompt reads it
- Ready for Plan 03 (testing and validation of template-aware synthesis)
- All existing tests pass (1 pre-existing failure unrelated to this work)

## Self-Check: PASSED

All files exist, all commits verified, all content checks pass.

---
*Phase: 11-colony-knowledge-integration*
*Completed: 2026-03-13*
