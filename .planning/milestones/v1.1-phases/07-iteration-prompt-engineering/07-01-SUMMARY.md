---
phase: 07-iteration-prompt-engineering
plan: 01
subsystem: oracle
tags: [bash, jq, prompt-engineering, phase-transitions, confidence-rubric, research-lifecycle]

# Dependency graph
requires:
  - phase: 06-state-architecture-foundation
    provides: "oracle.sh orchestrator with state.json/plan.json, generate_research_plan, jq validation"
provides:
  - determine_phase function with structural transition thresholds (survey/investigate/synthesize/verify)
  - build_oracle_prompt function prepending phase-specific directives to oracle.md
  - Iteration counter increment and phase transition check after each AI call
  - Phase-aware oracle.md with confidence rubric, depth enforcement, and targeting rules
affects: [08-convergence-orchestrator, 09-trust-calibration, 11-colony-integration]

# Tech tracking
tech-stack:
  added: []
  patterns: [phase-transition-thresholds, phase-directive-heredoc, confidence-rubric-anchoring, depth-enforcement-read-before-write]

key-files:
  created: []
  modified:
    - .aether/oracle/oracle.sh
    - .aether/oracle/oracle.md

key-decisions:
  - "Phase transitions use structural jq metrics: 25% avg or all-touched for survey->investigate, 60% avg or <2 below 50% for investigate->synthesize, 80% avg for synthesize->verify"
  - "build_oracle_prompt uses heredoc directives (10-15 lines each) prepended to oracle.md content"
  - "Confidence rubric anchored to evidence quality with explicit anti-inflation rule: one blog post = 30% not 70%"
  - "Iteration and phase managed exclusively by oracle.sh, not the AI prompt -- separation of control"

patterns-established:
  - "Phase-directive prepend: build_oracle_prompt emits phase heredoc then cats oracle.md, piped to AI CLI"
  - "Structural transition: determine_phase reads plan.json metrics via jq to decide phase progression"
  - "Depth enforcement: prompt requires reading existing findings before writing, rejects restatements"

requirements-completed: [LOOP-02, LOOP-03, INTL-02, INTL-03]

# Metrics
duration: 3min
completed: 2026-03-13
---

# Phase 07 Plan 01: Iteration Prompt Engineering Summary

**Phase-aware oracle with determine_phase structural transitions (25%/60%/80%), build_oracle_prompt directive prepend, iteration lifecycle management, and confidence rubric with depth enforcement**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-13T16:21:51Z
- **Completed:** 2026-03-13T16:25:07Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- determine_phase function evaluates plan.json metrics via jq to drive survey->investigate->synthesize->verify progression
- build_oracle_prompt function prepends phase-specific heredoc directives (survey breadth, investigate depth, synthesize connections, verify confirmation) before oracle.md content
- Iteration counter increment and phase transition check after each AI call in the main loop
- Rewritten oracle.md with phase-aware targeting, depth enforcement (read before write), 6-tier confidence rubric, and anti-inflation guidance

## Task Commits

Each task was committed atomically:

1. **Task 1: Add phase transition and iteration management to oracle.sh** - `1bfc02e` (feat)
2. **Task 2: Rewrite oracle.md with phase-aware instructions and confidence rubric** - `ca287e6` (feat)

## Files Created/Modified
- `.aether/oracle/oracle.sh` - Added determine_phase (structural thresholds), build_oracle_prompt (phase directives), iteration increment, phase transition check; changed AI invocation to use build_oracle_prompt pipe
- `.aether/oracle/oracle.md` - Rewritten with phase directive acknowledgment, phase-specific targeting (survey prefers untouched), depth enforcement (must read before writing), 6-tier confidence rubric, anti-inflation/deflation rules

## Decisions Made
- Phase transition thresholds: 25% avg confidence or all-questions-touched triggers survey->investigate; 60% avg or fewer than 2 questions below 50% triggers investigate->synthesize; 80% avg triggers synthesize->verify
- build_oracle_prompt uses bash heredoc strings for each phase directive, keeping directives in oracle.sh rather than separate files (simpler, all phase logic in one place)
- Confidence rubric explicitly anchors scores to evidence quality: single source caps at 50%, one blog post = 30% not 70%
- oracle.sh exclusively manages iteration counter and phase field -- AI prompt explicitly told not to modify these

## Deviations from Plan

None - plan executed exactly as written.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 08 (convergence orchestrator) can build on iteration management to add recovery logic and convergence detection
- Phase 09 (trust calibration) can leverage the confidence rubric and phase transition data
- The determine_phase thresholds (25%/60%/80%) are initial values -- Phase 08 may need to tune these based on empirical observation

## Self-Check: PASSED

All files verified present, all commits verified in git log.

---
*Phase: 07-iteration-prompt-engineering*
*Completed: 2026-03-13*
