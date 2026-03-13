---
phase: 09-source-tracking-and-trust-layer
plan: 01
subsystem: oracle
tags: [source-tracking, trust-scoring, citations, jq, plan-json, backward-compatible]

# Dependency graph
requires:
  - phase: 08-orchestrator-upgrade
    provides: convergence metrics, phase transitions, diminishing returns detection
provides:
  - compute_trust_scores function in oracle.sh
  - source tracking prompt requirements in oracle.md
  - backward-compatible plan.json v1.1 schema with sources registry
  - inline citation and Sources section in synthesis pass
  - Source Trust table in research-plan.md
  - updated validate-oracle-state for sources and structured findings
affects: [09-02-tests, 10-steering, 11-colony-integration]

# Tech tracking
tech-stack:
  added: []
  patterns: [structured-findings-with-source-ids, trust-ratio-metric, backward-compatible-schema-evolution]

key-files:
  created: []
  modified:
    - .aether/oracle/oracle.sh
    - .aether/oracle/oracle.md
    - .aether/aether-utils.sh
    - .claude/commands/ant/oracle.md
    - .opencode/commands/ant/oracle.md

key-decisions:
  - "Flag unsourced findings rather than reject them -- trust_summary.no_source makes the gap visible without losing valuable research"
  - "Source tracking is a prompt+schema problem -- AI records sources, oracle.sh counts them structurally"
  - "plan.json v1.1 bump is safe -- no code checks version value, only type"

patterns-established:
  - "Structured findings: key_findings items are objects {text, source_ids, iteration} not strings"
  - "Sources registry: top-level sources object in plan.json keyed by sequential IDs (S1, S2...)"
  - "Trust ratio: multi_source * 100 / total_findings as integer percentage"
  - "Phase directive reminders: each phase gets a one-line source tracking reminder"

requirements-completed: [TRST-01, TRST-02, TRST-03]

# Metrics
duration: 5min
completed: 2026-03-13
---

# Phase 9 Plan 1: Source Tracking and Trust Layer Summary

**compute_trust_scores function, source tracking prompt, inline citations in synthesis, backward-compatible plan.json v1.1 with sources registry**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-13T18:23:03Z
- **Completed:** 2026-03-13T18:28:31Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments
- Added compute_trust_scores function to oracle.sh that counts source_ids per finding and writes trust_summary to plan.json
- Updated oracle.md to require source tracking for every finding with structured objects, source-backed confidence rules, and synthesis citation requirements
- Updated build_synthesis_prompt to require inline [S1] citations and a Sources section in the final report
- Updated generate_research_plan to show Source Trust table when trust data exists
- Added backward-compatible validation for sources registry and structured findings in validate-oracle-state
- Bumped plan.json wizard template to v1.1 with empty sources registry in both Claude and OpenCode commands

## Task Commits

Each task was committed atomically:

1. **Task 1: Add compute_trust_scores to oracle.sh and update build_synthesis_prompt and generate_research_plan** - `846fd57` (feat)
2. **Task 2: Update oracle.md prompt with source tracking requirements and update phase directives** - `6ab0cf2` (feat)
3. **Task 3: Update validate-oracle-state, wizard commands, and version bump** - `87713bd` (feat)

## Files Created/Modified
- `.aether/oracle/oracle.sh` - Added compute_trust_scores function, main loop call, updated build_synthesis_prompt with Sources section and inline citations, updated generate_research_plan with Source Trust table, added source tracking reminders to all 4 phase directives
- `.aether/oracle/oracle.md` - Added Source Tracking (MANDATORY) section, changed key_findings format to structured objects, added source-backed confidence rules, added synthesis citation rule
- `.aether/aether-utils.sh` - Added sources registry validation and structured findings validation to validate-oracle-state plan checks
- `.claude/commands/ant/oracle.md` - Bumped plan.json template to v1.1, added empty sources registry field
- `.opencode/commands/ant/oracle.md` - Mirror of Claude wizard changes for parity

## Decisions Made
- Flag unsourced findings rather than reject them: trust_summary.no_source makes the gap visible without losing valuable research. Stripping findings would be destructive.
- Source tracking is a prompt+schema problem, not a tool problem: The AI already has URL access; the gap was that the prompt did not require capturing them and the schema had no place for them.
- plan.json version bump from 1.0 to 1.1 is safe: validate-oracle-state checks type (string) not value; no code checks for version == "1.0".

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Source tracking infrastructure complete, ready for Phase 9 Plan 2 (trust scoring tests)
- All existing oracle tests pass (20 Ava + 13 bash), confirming backward compatibility
- Both v1.0 (string findings, no sources) and v1.1 (structured findings, sources) formats validated

## Self-Check: PASSED

All 5 modified files verified present. All 3 task commit hashes verified in git log.

---
*Phase: 09-source-tracking-and-trust-layer*
*Completed: 2026-03-13*
