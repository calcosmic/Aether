---
phase: 23-agent-resilience
plan: 01
subsystem: agents
tags: [opencode, agents, resilience, failure-handling, error-recovery]

# Dependency graph
requires:
  - phase: 22-agent-boilerplate-cleanup
    provides: Clean agent files with consistent description format ready for content additions
provides:
  - Tiered failure handling (minor/major) with 2-attempt retry limit across all 7 HIGH-risk agents
  - Self-check success verification steps for all 7 agents
  - Explicit read-only boundary declarations for all 7 agents
  - Peer review triggers for queen and builder (Watcher reviews both)
  - Escalation format (what failed, 2-3 options, recommendation) standardized across all 7 agents
affects: [23-agent-resilience, future agent invocations colony-wide]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Tiered severity pattern: minor failures retry silently (max 2 attempts), major failures STOP immediately"
    - "Escalation format: what failed + 2-3 options with trade-offs + recommendation"
    - "Peer review pattern: HIGH-stakes agents (queen, builder) reviewed by Watcher before output is final"
    - "Self-verify pattern: agents verify their own work before reporting complete"

key-files:
  created: []
  modified:
    - .opencode/agents/aether-queen.md
    - .opencode/agents/aether-builder.md
    - .opencode/agents/aether-watcher.md
    - .opencode/agents/aether-weaver.md
    - .opencode/agents/aether-route-setter.md
    - .opencode/agents/aether-ambassador.md
    - .opencode/agents/aether-tracker.md

key-decisions:
  - "Existing rules (3-Fix Rule, Iron Laws, Verification Discipline) are referenced by new sections, not redefined — additive pattern"
  - "Builder and Queen include Watcher peer review triggers; Watcher self-verifies as it IS the verifier"
  - "2-attempt retry limit applies to individual operations; 3-Fix Rule applies to debugging cycles (distinct scopes)"
  - "Weaver and Tracker failure_modes explicitly distinguish these two limits to prevent confusion"
  - "No peer review triggers for Weaver, Route-Setter, Ambassador, Tracker per research classification (self-verify only)"

patterns-established:
  - "Resilience section pattern: <failure_modes> + <success_criteria> + <read_only> appended after Output Format block"
  - "Tiered severity: minor=retry silently max 2 attempts, major=STOP immediately, 2-retries-exhausted=promote to major"
  - "Escalation format: (1) what failed with exact text, (2) 2-3 options with trade-offs, (3) recommendation"
  - "Never fail silently: every failure mode has an explicit response"

requirements-completed: [RESIL-01, RESIL-02, RESIL-03]

# Metrics
duration: 4min
completed: 2026-02-19
---

# Phase 23 Plan 01: Agent Resilience (HIGH-Risk Agents) Summary

**21 XML resilience sections added to 7 HIGH-risk agents: tiered failure handling, self-check verification, and boundary declarations — never fail silently.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-19T23:09:30Z
- **Completed:** 2026-02-19T23:13:37Z
- **Tasks:** 2 of 2
- **Files modified:** 7

## Accomplishments

- Added `<failure_modes>`, `<success_criteria>`, and `<read_only>` sections to all 7 HIGH-risk agents (queen, builder, watcher, weaver, route-setter, ambassador, tracker)
- Standardized escalation format across all 7 agents: what failed, 2-3 options with trade-offs, recommendation
- Established peer review chain: Queen and Builder are reviewed by Watcher; Watcher self-verifies as the colony's verifier
- All existing agent rules (3-Fix Rule, Iron Laws, Verification Discipline) referenced by new sections without contradiction

## Task Commits

Each task was committed atomically:

1. **Task 1: Add resilience sections to queen, builder, watcher** - `7defdf8` (feat)
2. **Task 2: Add resilience sections to weaver, route-setter, ambassador, tracker** - `93270f9` (feat)

## Files Created/Modified

- `.opencode/agents/aether-queen.md` - Added failure_modes (COLONY_STATE corruption, orphaned spawn, destructive git), success_criteria (Watcher peer review trigger, state validation), read_only (source file boundary)
- `.opencode/agents/aether-builder.md` - Added failure_modes (3-Fix Rule integration, protected path guard), success_criteria (Watcher peer review trigger, file existence check), read_only (colony state boundary)
- `.opencode/agents/aether-watcher.md` - Added failure_modes (false negative risk, Iron Law reference), success_criteria (fresh re-run requirement, score justification), read_only (source file read-only posture)
- `.opencode/agents/aether-weaver.md` - Added failure_modes (behavior change = STOP immediately), success_criteria (before/after test suite comparison), read_only (no new features, no test expectation changes)
- `.opencode/agents/aether-route-setter.md` - Added failure_modes (corrupted state guard, phase overwrite guard), success_criteria (plan structure validation, file path existence check), read_only (planning-only boundary)
- `.opencode/agents/aether-ambassador.md` - Added failure_modes (secret write = STOP, auth failure escalation), success_criteria (real test call, secret check), read_only (env var documentation only)
- `.opencode/agents/aether-tracker.md` - Added failure_modes (3-Fix Rule integration, fix-introduces-new-failure = revert), success_criteria (reproduction check, regression check), read_only (same as Builder)

## Decisions Made

- Existing rules (3-Fix Rule, Iron Laws, Verification Discipline) are referenced by new sections — additive pattern, no redefinition
- Builder and Queen include Watcher peer review triggers; Watcher self-verifies (it IS the verifier)
- 2-attempt retry limit and 3-Fix Rule are explicitly distinguished in Builder and Tracker — they operate at different scopes
- No peer review triggers added to Weaver, Route-Setter, Ambassador, Tracker — per research classification these are self-verify agents
- Ambassador gets the most security-specific failure mode: API key/secret write = immediate STOP (no retry)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All 7 HIGH-risk agents now have consistent resilience sections
- Pattern is established for Phase 23 Plan 02 (MEDIUM-risk agents) if it follows the same structure
- Peer review chain is defined: queen/builder surface to Watcher, Watcher self-verifies

---
*Phase: 23-agent-resilience*
*Completed: 2026-02-19*
