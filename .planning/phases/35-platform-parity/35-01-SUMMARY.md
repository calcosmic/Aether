---
phase: 35-platform-parity
plan: 01
subsystem: testing
tags: [go, testing, agent-parity, codex, opencode, claude]

requires:
  - phase: 34-cleanup
    provides: Clean working tree with no stale worktrees or branches

provides:
  - Drift detection test for Claude/OpenCode agent content parity
  - Codex agent completeness checklist test
  - Per-agent line-count diff reporting for all 25 agents

affects:
  - 35-platform-parity plan 02 (fix drift)
  - 35-platform-parity plan 03 (Codex updates)

tech-stack:
  added: []
  patterns:
    - "Advisory tests: log warnings instead of hard-failing for platform-specific adaptations"
    - "Byte-for-byte parity tests with per-agent diff reporting"

key-files:
  created: []
  modified:
    - cmd/codex_e2e_test.go - Added TestClaudeOpenCodeAgentContentParity and TestCodexAgentCompleteness

key-decisions:
  - "Both tests committed in a single commit because they were added in the same edit block"
  - "Parity test is hard-failing to force explicit decisions about intentional drift"
  - "Codex completeness test is advisory (logs warnings) because platform-specific adaptations are legitimate"

patterns-established:
  - "Content parity tests compare byte-for-byte and report line-count diffs per agent"
  - "Completeness checklists validate conceptual coverage (TDD, boundaries, escalation) rather than exact content"

requirements-completed: []
---

# Phase 35 Plan 01: Drift Detection Infrastructure Summary

**Agent content parity tests that detect all 25 OpenCode agents drifting from Claude masters (48-316 lines each) and flag 53 Codex completeness warnings across deprecated patterns and missing Phase 31-33 concepts.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-23T03:34:00Z
- **Completed:** 2026-04-23T03:42:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- Added `TestClaudeOpenCodeAgentContentParity` — byte-for-byte comparison of all 25 Claude vs OpenCode agents with per-agent line-count diff reporting.
- Added `TestCodexAgentCompleteness` — advisory checklist verifying `developer_instructions`, TDD references, protected/boundary rules, escalation handling, and deprecated pattern flags (`flag-add`, `activity-log`).
- Parity test correctly fails, proving drift detection works: all 25 agents are out of sync (diffs range from 16 to 316 lines).
- Codex test passes while logging 53 warnings: 21 deprecated pattern hits and 32 missing content hits.

## Task Commits

Both tasks were committed together in a single commit because they were added in the same edit block:

1. **Task 1: Add OpenCode agent content parity test** — `f4ec5711` (test)
2. **Task 2: Add Codex agent completeness checklist test** — `f4ec5711` (test)

**Plan metadata:** `f4ec5711` (test: add agent parity and completeness tests)

## Files Created/Modified

- `cmd/codex_e2e_test.go` — Added two new test functions (127 lines inserted):
  - `TestClaudeOpenCodeAgentContentParity` — hard-failing drift detector
  - `TestCodexAgentCompleteness` — advisory completeness checklist

## Decisions Made

- Parity test is hard-failing (`t.Errorf`) to ensure CI catches drift immediately and forces an explicit decision about whether divergence is intentional.
- Codex test is advisory (`t.Logf`) because Codex agents are platform-specific by design (TOML format, `shell`/`apply_patch` tools, JSON output) and legitimate adaptations should not break the build.
- Both tests reuse existing helpers (`findRepoRoot`, `listAgentBaseNames`, `listShippedAetherCodexAgentBaseNames`) to avoid duplication.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Both tests were added in a single edit session and committed together. The commit message was amended to reflect both tests after the initial commit message only described Task 1.

## Known Stubs

None.

## Threat Flags

None.

## Next Phase Readiness

- Drift detection is now in place. Plan 02 can proceed to fix the actual OpenCode agent drift (copy Claude content to OpenCode).
- Codex completeness warnings are documented and ready for Plan 03 to address.

---
*Phase: 35-platform-parity*
*Completed: 2026-04-23*
