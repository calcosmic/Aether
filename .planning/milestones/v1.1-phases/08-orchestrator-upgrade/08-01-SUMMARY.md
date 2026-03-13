---
phase: 08-orchestrator-upgrade
plan: 01
subsystem: oracle
tags: [bash, jq, convergence, signal-handling, synthesis, json-recovery]

# Dependency graph
requires:
  - phase: 07-iteration-prompt-engineering
    provides: "Phase-aware prompt (build_oracle_prompt), iteration counter, phase transitions (determine_phase)"
provides:
  - "compute_convergence function computing structural metrics from plan.json"
  - "update_convergence_metrics function maintaining convergence history in state.json"
  - "check_convergence function for composite score threshold checking"
  - "detect_diminishing_returns function with rolling window and phase-adjusted thresholds"
  - "validate_and_recover function with pre-iteration and atomic-write backup fallback"
  - "build_synthesis_prompt and run_synthesis_pass for structured final reports"
  - "cleanup_and_synthesize trap handler for SIGINT/SIGTERM with re-entrancy protection"
  - "Synthesis pass on every exit path (stop, max-iter, convergence, interrupt, corruption)"
affects: [09-trust-calibration, 10-steering-intelligence, 11-colony-integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Composite convergence score from 3 structural metrics (gap resolution 40%, coverage 30%, novelty 30%)"
    - "Rolling window diminishing returns detection with phase-adjusted novelty thresholds"
    - "Pre-iteration backup pattern for JSON recovery"
    - "Trap handler with INTERRUPTED flag for re-entrancy protection"
    - "Synthesis-on-every-exit ensuring useful output from any oracle run"

key-files:
  created: []
  modified:
    - ".aether/oracle/oracle.sh"
    - ".aether/oracle/oracle.md"

key-decisions:
  - "Convergence composite score: gap_resolution*40% + coverage*30% + (low_novelty?100:0)*30% using integer arithmetic"
  - "Convergence requires composite >= 85 AND 2 consecutive low-novelty iterations"
  - "Diminishing returns: 3-iteration window with phase-adjusted thresholds (investigate: 0, others: 1)"
  - "Synthesis pass timeout: 180 seconds via timeout command with graceful fallback if unavailable"
  - "Max iterations exit changed from exit 1 to synthesis pass + exit 0"
  - "ORACLE_CONVERGENCE_THRESHOLD and ORACLE_DR_WINDOW env vars for empirical tuning"

patterns-established:
  - "Pre-iteration backup: cp state/plan files before AI invocation, validate after"
  - "Validate-and-recover: check JSON validity, try pre-iteration backup, fall back to atomic-write restore"
  - "Synthesis-on-exit: every exit path calls run_synthesis_pass with a reason string"
  - "INTERRUPTED flag pattern: trap handler checks flag to prevent re-entrant synthesis"

requirements-completed: [LOOP-04, INTL-05, OUTP-02]

# Metrics
duration: 3min
completed: 2026-03-13
---

# Phase 8 Plan 1: Orchestrator Upgrade Summary

**Multi-signal convergence detection, diminishing returns with strategy change, synthesis-on-every-exit, SIGINT trap handler, and JSON recovery with pre-iteration backups added to oracle.sh**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-13T17:11:30Z
- **Completed:** 2026-03-13T17:14:43Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- oracle.sh upgraded from 319 to 560+ lines with 8 new functions and restructured main loop
- Convergence detection uses structural plan.json metrics (gap resolution, coverage, novelty) instead of AI self-assessment
- Every exit path (stop, max-iter, convergence, SIGINT/SIGTERM, corruption) triggers a synthesis pass producing a structured report
- Malformed JSON triggers recovery from pre-iteration backup or atomic-write backup system
- All 25 existing tests (14 ava + 11 bash) pass with zero regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Add convergence, diminishing returns, synthesis, signal handling, and recovery functions** - `c682a56` (feat)
2. **Task 2: Add synthesis pass awareness to oracle.md** - `da7c307` (feat)

## Files Created/Modified
- `.aether/oracle/oracle.sh` - Added 8 new functions (compute_convergence, update_convergence_metrics, check_convergence, detect_diminishing_returns, validate_and_recover, build_synthesis_prompt, run_synthesis_pass, cleanup_and_synthesize); restructured main loop with trap, pre-iteration backups, convergence checks, diminishing returns handling
- `.aether/oracle/oracle.md` - Added synthesis pass awareness rule to Important Rules section

## Decisions Made
- Composite convergence score uses integer arithmetic scaled by 100 (multiply by 40/30/30 and divide by 100) for Bash 3.2 compatibility
- Novelty component in composite score rewards LOW novelty (research has stopped finding new things at high coverage), scoring 100 when novelty_delta <= 1
- Phase-adjusted novelty thresholds: investigate phase uses 0 (any new finding counts as progress), all other phases use 1
- Diminishing returns in survey/investigate forces phase to synthesize; in synthesize/verify triggers immediate synthesis pass
- Synthesis pass uses timeout 180 with fallback to no-timeout if the timeout command is unavailable
- validate_and_recover logs all recovery actions to stderr for debugging prompt issues
- Max iterations changed from exit 1 to synthesis pass + exit 0 (useful output, not an error)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- oracle.sh now has complete convergence detection and synthesis infrastructure
- Convergence thresholds (85 composite, 3-iteration window) are configurable via env vars for empirical tuning in Phase 9
- Phase 8 Plan 2 (convergence tests) can now test the new functions
- state.json schema extended with optional convergence object (backward compatible)

## Self-Check: PASSED

All files exist, all commits verified.

---
*Phase: 08-orchestrator-upgrade*
*Completed: 2026-03-13*
