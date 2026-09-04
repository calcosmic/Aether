---
phase: 199-front-door-and-classic-contract
plan: "23"
subsystem: lifecycle-wrapper-contract
tags: [seal, force-authority, codex, command-guide, lifecycle]
requires:
  - phase: 199-11
    provides: finalized runtime closure semantics
  - phase: 199-15
    provides: sealed-status and optional-entomb lifecycle projection
provides:
  - owner-only force guidance for managed seal wrappers
  - aligned Codex build-cycle and command-guide lifecycle contract
affects: [Claude wrappers, OpenCode wrappers, Codex lifecycle guidance]
tech-stack:
  added: []
  patterns: [source-backed contract tests, runtime-owned closure guidance]
key-files:
  created: [cmd/seal_wrapper_contract_199_test.go]
  modified: [.aether/commands/seal.yaml, .claude/commands/ant/seal.md, .claude/commands/ant-seal.md, .opencode/commands/ant/seal.md, .aether/skills/colony/aether-colony-build-cycle/SKILL.md, cmd/command_guide.go, cmd/command_guide_test.go]
key-decisions:
  - "Managed wrappers preserve direct owner force input but never construct or offer force authority."
  - "Codex guidance keeps status review primary and entomb explicitly optional after seal."
patterns-established:
  - "Cross-platform wrapper contract tests scan every managed surface for identical lifecycle truth."
requirements-completed: [CEC-08, LIFE-01, LIFE-05]
duration: 6min
completed: 2026-09-04
---

# Phase 199 Plan 23: Seal and Build-Cycle Contract Summary

**Owner-only incomplete closure guidance across managed seal wrappers and the Codex lifecycle guide, with explicit status review before optional entomb.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-09-04T22:42:16Z
- **Completed:** 2026-09-04T22:48:03Z
- **Tasks:** 3/3
- **Files modified:** 8

## Accomplishments

- Added named contract tests for all seal wrapper surfaces and Codex lifecycle guidance.
- Made the canonical seal description and managed wrappers delegate force validation, final review, confirmation, transaction, and rendering to Go.
- Documented automatic territory freshness, equal guided-build/Autopilot choice, bounded repairs/debt, independent safe-path continuation, explicit sealing, status review, and the Phase 200–205 scope fence.

## Task Commits

1. **Task 1: Ratchet seal and build-cycle guide semantics** — `e5383a8d` (test)
2. **Task 2: Synchronize canonical and generated seal wrappers** — `6f14f97c` (feat)
3. **Task 3: Align the existing Codex build-cycle skill and guide** — `cc3eae4e` (docs)

## Files Created/Modified

- `cmd/seal_wrapper_contract_199_test.go` — verifies owner-only force authority and cross-platform seal truth.
- `.aether/commands/seal.yaml` — records exact closure and runtime-ownership contract.
- `.claude/commands/ant/seal.md`, `.claude/commands/ant-seal.md`, `.opencode/commands/ant/seal.md` — synchronized managed seal wrappers.
- `.aether/skills/colony/aether-colony-build-cycle/SKILL.md` — pins intelligent lifecycle boundaries.
- `cmd/command_guide.go`, `cmd/command_guide_test.go` — expose and test matching Codex guidance.

## Decisions Made

- Wrappers may preserve a force flag supplied directly by the owner, but cannot offer, construct, or rerun a force command.
- A forced-incomplete closure is never presented as verified success; status remains the primary post-seal action and entomb is owner-confirmed and optional.

## Verification

- `go test ./cmd -run '^(TestSealWrapperContract199|TestCommandGuideBuildCycle199|TestCommandSourceHygiene|TestForceSealWrapperAsksFirst|TestCommandGuideLifecycle199|TestCommandGuideYamlCodexMetadataMatches|TestLifecycleWrappersCarryCoherentJobContract)$' -count=1` — 21 passed
- `go build ./cmd/aether` — passed
- `go test ./cmd -race -run '^(TestSealWrapperContract199|TestCommandGuideBuildCycle199|TestCommandSourceHygiene)$' -count=1` — 13 passed

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Test package] Corrected the new test package declaration before RED verification.**
- **Found during:** Task 1
- **Fix:** Used package `cmd`, matching the command package under test.

**2. [Rule 1 - Contract ratchet] Replaced the old generic `$ant-` prohibition with a prohibition on creating a native lifecycle surface.**
- **Found during:** Task 3
- **Fix:** Allowed the required Phase 200–205 scope-fence reference while retaining the ban on creating Codex-native lifecycle skills.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Managed seal wrappers and existing Codex guidance now preserve the finalized runtime closure contract. Plan 27 is the next incomplete Phase 199 plan.

## Self-Check: PASSED

- Required source files exist.
- Task commits `e5383a8d`, `6f14f97c`, and `cc3eae4e` exist.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
