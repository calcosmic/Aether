---
phase: 205-owner-acceptance-and-restoration-seal
plan: 05
subsystem: verification
tags: [criterion-evidence, deterministic-floor, continue, regression-fixture]

# Dependency graph
requires: []
provides:
  - "criterionNamedCommands / criterionNamedCommandsWereRun in cmd/criterion_evidence.go: a criterion whose own wording names a command absent from the commands this phase actually ran is refused, naming both the criterion and the missing command"
  - "Permanent regression fixture (TestCriterionNamingAnUnrunCommandIsRefused) reproducing field report 2026-09-14-cosmic-seal-entomb-lifecycle.md finding 7"
affects: [criterion evidence evaluation, aether continue (both lanes), any future phase whose success criteria name a specific command]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 4560
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Command-coverage recognition reuses existing recognisers (commandSafeToReRun for argv-shaped runner invocations, detectVerificationCommandKind for the four generic build/type/lint/test prefixes) instead of inventing a third command-shape parser"

key-files:
  created:
    - cmd/criterion_command_coverage_test.go
  modified:
    - cmd/criterion_evidence.go

key-decisions:
  - "Reconstructed the field report's criterion wording rather than quoting it verbatim: the report only ever prints the truncated tail (\"...and the operator broker suite all pass\"), never the full original sentence or where in it the command was named. The fixture criterion embeds the report's own diagnosed gap (a backtick-quoted `npm run test:operator`) alongside the report's exact quoted tail phrase, so the historical failure mode reproduces precisely: build/types/lint/tests all resolve and pass, but the specific suite the criterion names never ran."
  - "Command recognition uses two existing recognisers rather than a new one: commandSafeToReRun (already in this file, argv-shaped + no shell metacharacters) for backtick-quoted commands, and detectVerificationCommandKind (cmd/codex_continue.go, CLAUDE.md-parsing) for a fixed, bounded set of bare-prose command shapes. Neither risks matching ordinary prose, which the blast-radius run confirmed against this repository's own test fixtures (zero new flags)."
  - "Comparison is substring-based, not exact-match, on both sides normalised (lower-cased, whitespace-collapsed): a criterion naming a shorter form of the actual configured command (\"npm test\" against a configured \"npm test -- --ci\") is still credited."

requirements-completed: [PROOF-05]

coverage:
  - id: D1
    description: "A criterion whose own wording names a command that never ran is refused, naming the criterion and the missing command, instead of being credited by unrelated build/types/lint/tests checks"
    requirement: "PROOF-05"
    verification:
      - kind: unit
        ref: "cmd/criterion_command_coverage_test.go#TestCriterionNamingAnUnrunCommandIsRefused"
        status: pass
      - kind: unit
        ref: "cmd/criterion_command_coverage_test.go#TestCriterionNamingARunCommandIsCredited"
        status: pass
      - kind: unit
        ref: "cmd/criterion_command_coverage_test.go#TestCriterionNamingNoCommandKeepsExistingBehaviour"
        status: pass
      - kind: unit
        ref: "cmd/criterion_command_coverage_test.go#TestCriterionCommandExtractionIgnoresProse"
        status: pass
      - kind: integration
        ref: "cmd/criterion_command_coverage_test.go#TestCriterionCommandCoverageIsReachedFromBothCheckLanes"
        status: pass
    human_judgment: false

# Metrics
duration: ~35min
completed: 2026-09-15
status: complete
---

# Phase 205 Plan 05: Criterion Command-Coverage Refusal Summary

**A success criterion naming an unconfigured command (e.g. `npm run test:operator`) in its own wording is now refused instead of silently credited by unrelated build/type/lint/test checks — closing the exact gap a real downstream run hit.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-09-15 (session start, prior to first commit)
- **Completed:** 2026-09-15T09:00Z
- **Tasks:** 2 completed
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- Added `criterionNamedCommands` and `criterionNamedCommandsWereRun` to `cmd/criterion_evidence.go`, extending the existing per-criterion evaluation loop in `evaluatePhaseCriterionEvidence` — not a second crediting path.
- Reproduced the field report's real failure (finding 7, `.planning/field-reports/2026-09-14-cosmic-seal-entomb-lifecycle.md`) as a permanent regression fixture: `TestCriterionNamingAnUnrunCommandIsRefused` failed before the fix (evaluation wrongly credited the criterion, `Passed: true`) and passes after.
- Proved the refusal reaches both criterion-crediting lanes (`TestCriterionCommandCoverageIsReachedFromBothCheckLanes`: in-process `runCodexContinueVerification` and wrapper/external `runCodexContinueVerificationSnapshot`, both via the shared `runDeterministicFloor` body).
- Ran the scoped blast-radius suite (`go test ./cmd/ -run 'Criterion|Continue|Verification'`, 221s, no new failures beyond one pre-existing known-red baseline test) — the new refusal newly flags zero criteria anywhere else in this repository's own test fixtures.

## Task Commits

TDD RED/GREEN cycle, both tasks proven in the same commit pair since Task 2's test (`TestCriterionCommandCoverageIsReachedFromBothCheckLanes`) is one cohesive feature slice with Task 1's:

1. **Task 1: A criterion naming a command that never ran is refused by name** - `e54be24b` (test, RED) / `47b3513b` (feat, GREEN)
2. **Task 2: Prove the new refusal cannot be bypassed, and check the blast radius** - `47b3513b` (the same GREEN commit; its test file already included `TestCriterionCommandCoverageIsReachedFromBothCheckLanes` from the RED commit onward) + this SUMMARY documents the blast-radius run's results (no additional production changes were needed — the extraction rule produced zero false positives against this repository's own fixtures, so nothing required tightening)

**Plan metadata:** (this SUMMARY's commit, made by the orchestrator after merge)

_Note: this is a `tdd="true"` tracer task; RED/GREEN commits substitute for the standard one-task-one-commit pattern._

## Files Created/Modified

- `cmd/criterion_command_coverage_test.go` - Six new test functions (incl. two with subtests) proving the refusal, its non-interference with unaffected criteria, and lane parity
- `cmd/criterion_evidence.go` - `criterionNamedCommands`, `normalizeCriterionCommandText`, `executedCriterionCommands`, `criterionNamedCommandsWereRun`, and the new command-coverage step wired into `evaluatePhaseCriterionEvidence`'s existing per-requirement loop

## Decisions Made

- **Reconstructed the field-report criterion wording.** The field report (`.planning/field-reports/2026-09-14-cosmic-seal-entomb-lifecycle.md`, finding 7) only ever quotes the truncated tail — `"...and the operator broker suite all pass"` — never the full original sentence, and RESEARCH.md itself flags this as unconfirmed against the target project's actual criterion text. The fixture reconstructs a complete sentence carrying that exact quoted tail plus the backtick-quoted command RESEARCH.md identifies as the gap (`` `npm run test:operator` ``), so the regression fixture exercises the precise failure shape (all four generic checks pass; the specific named suite never ran) without inventing a scenario the report never described.
- **Reused two existing recognisers instead of building a third.** `commandSafeToReRun` (already in `cmd/criterion_evidence.go`, used by the builder-evidence re-run path) recognises argv-shaped runner invocations inside backticks; `detectVerificationCommandKind` (`cmd/codex_continue.go`, the CLAUDE.md verification-command parser) recognises a fixed, bounded set of bare-prose command shapes. Both were chosen specifically because they are already proven not to over-match ordinary English prose — confirmed by the blast-radius run finding zero new false positives in this repository's own fixtures.
- **Substring comparison, not exact match.** A criterion naming a shorter form of the actual configured command (e.g. `npm test` against a configured `npm test -- --ci`) is still credited; only a command genuinely absent from every executed step's text is flagged.

## Deviations from Plan

None — plan executed as written, including the TDD RED/GREEN structure Task 1 required. Task 2 required no additional production changes: the blast-radius check it specified found zero newly-flagged criteria in this repository, so there was nothing to tighten.

## Issues Encountered

- The scoped blast-radius run (`go test ./cmd/ -run 'Criterion|Continue|Verification' -count=1 -timeout 90m`) exits 1, not 0 as the plan's acceptance criterion literally states, because of exactly one pre-existing failure: `TestGoldenContinueVisualOutput` (golden fixture mismatch on an unrelated emoji-formatting line, `"📊 Review depth: ..."` vs `"Review depth: ..."`). This test is on this repository's documented known-red baseline (verified 2026-09-14 at commit `3802d180`, listed in the orchestrator-supplied repo notes) and is unrelated to any file this plan touched. Not fixed here, per the repo notes' explicit instruction to record baseline hits rather than fix them. Every test whose name matches `Criterion` passed, and the two new lane-parity/refusal tests both exit 0 individually.
- No `.aether/data/COLONY_STATE.json` exists in this worktree (gitignored, not populated), so there was no local runtime colony state with real phase criteria to check directly against the new refusal. The check was still performed at the level the plan actually asks for — this repository's own Go test fixtures (`go test ./cmd/ -run 'Criterion|Continue|Verification'`) — and found zero newly-flagged criteria.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The command-coverage refusal is live in `evaluatePhaseCriterionEvidence` and reachable from both continue lanes; no further wiring needed.
- No criteria in this repository's own suite were newly flagged, so no follow-up cleanup is required before other Phase 205 plans (which build on the same continue/verification machinery in later waves) proceed.

---
*Phase: 205-owner-acceptance-and-restoration-seal*
*Completed: 2026-09-15*

## Self-Check: PASSED

- FOUND: cmd/criterion_evidence.go
- FOUND: cmd/criterion_command_coverage_test.go
- FOUND: e54be24b (test commit, in git log)
- FOUND: 47b3513b (feat commit, in git log)
