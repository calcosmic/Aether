---
phase: 200-iterative-planning
plan: 36
subsystem: build-lifecycle-tests
tags: [go, tdd, build-start, transactions, retries, ast-ratchet]

requires:
  - phase: 200-35
    provides: Canonical accepted-authority build-start fixture and bounded fixture ratchet
provides:
  - Canonical build-start coverage for external, coherent-retry, plan-only, and active-pause fixtures
  - Removal of every legacy partial attempt, parent-link, and manifest-binding start helper
  - Repository-wide AST ratchet plus all-package compile proof against bypass reintroduction
  - Attempt enumeration that excludes durable start-receipt siblings
affects: [plan-39-public-journey, plan-40-final-gates, build-attempt-journal, coherent-retry, pause-resume]

tech-stack:
  added: []
  patterns: [transaction-only test setup, closed child-retry effects, repository-wide AST retirement ratchet]

key-files:
  created: []
  modified:
    - cmd/build_attempt_external_test.go
    - cmd/coherent_job_retry_test.go
    - cmd/coherent_job_retry_plan_only_test.go
    - cmd/coherent_job_retry_command_test.go
    - cmd/pause_resume_199_test.go
    - cmd/build_attempt.go

key-decisions:
  - "Custom retry fixtures retain their full six-task lane semantics by seeding explicit legacy-unbound plan policy before the canonical transaction, rather than shrinking the scenario to the one-task shared default."
  - "A coherent child retry is derived and committed with explicit parent effects and makeLatest=false; it remains journal-discoverable without impersonating active dispatched work."
  - "The active-pause assertion counts zero new pause receipts after its canonical start, because build start now has its own legitimate durable transaction evidence."
  - "Repository-wide retirement checks inspect executable Go syntax, preserving pure attempt derivation and post-start transitions while rejecting retired calls and direct attempt-plus-pointer writers."

requirements-completed: [PLAN-06]

duration: 21min
completed: 2026-09-09
---

# Phase 200 Plan 36: Legacy Build-Start Helper Retirement Summary

**Every remaining external, retry, plan-only, and pause fixture now starts through the atomic build transaction, and the obsolete multi-write escape hatches are deleted and barred from returning.**

## Performance

- **Duration:** 21 min
- **Started:** 2026-09-09T03:49:02Z
- **Completed:** 2026-09-09T04:09:37Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Migrated the final five fixture families from four parent-start calls, one child-start call, and one hand-built attempt/latest pair to canonical receipt-last build starts.
- Preserved six-task unfinished-only retry selection, parent immutability, explicit parent provenance, plan-only recovery, latest-pointer semantics, and a real live-process pause boundary.
- Added a byte-stable replay case for an already-bound external start: replay returns the same receipt and transaction identity without rewriting the attempt or pointer.
- Removed the legacy attempt, child, parent-link, and manifest-binding writers only after the bounded fixture ratchet proved every executable caller was gone.
- Added a repository-wide AST ratchet that rejects retired declarations/calls and direct attempt-plus-latest write sequences, backed by successful compilation of every Go package.
- Corrected attempt journal enumeration so `.start-receipt.json` siblings are not decoded as fake attempt records.

## Task Commits

Each task used a fail-first test boundary followed by a scoped GREEN commit:

1. **Task 1 RED: Require canonical remaining fixture starts** - `62c9d5ed` (test)
2. **Task 1 GREEN: Migrate external, retry, plan-only, and pause fixtures** - `20a1ff29` (test)
3. **Task 2 RED: Define repository-wide helper retirement and receipt filtering** - `9f78dde5` (test)
4. **Task 2 GREEN: Delete partial writers and filter start receipts** - `2f11e37e` (refactor)

## Files Created/Modified

- `cmd/build_attempt_external_test.go` - Bounded migration ratchet, external replay proof, receipt-enumeration proof, repository-wide AST retirement ratchet, and canonical idempotent-retry fixture.
- `cmd/coherent_job_retry_test.go` - Canonical parent and child starts with explicit parent provenance, parent byte immutability, and makeLatest=false proof.
- `cmd/coherent_job_retry_plan_only_test.go` - Canonical partial parent start preserving the literal plan-only recovery path and latest-parent behavior.
- `cmd/coherent_job_retry_command_test.go` - Canonical six-task parent start preserving unfinished-only redispatch command coverage.
- `cmd/pause_resume_199_test.go` - Active work seeded by a canonical real-process start and moved to dispatching only through the production transition API.
- `cmd/build_attempt.go` - Legacy start/provenance/manifest writers removed; attempt enumeration now excludes durable start receipts.

## Decisions Made

- The remaining custom retry fixtures explicitly use `legacy_unbound` acceptance with evidence not required. This keeps their real six-task graphs while still forcing every build-start effect through `commitBuildStart`.
- Child recovery is committed as the closed coherent-child variant with explicit parent ID/job effects. The canonical rule—not a test override—keeps it out of the latest pointer.
- The pause fixture continues to prove that the refused pause writes zero new lifecycle receipts. Its already-committed build-start receipt is treated as valid pre-existing evidence, not mistaken for a pause side effect.
- The AST ratchet scans non-vendored repository Go syntax. It rejects the exact retired API surface and functions that perform multiple direct writes while resolving the latest-attempt target; pure derivation and transaction target declaration remain legal.

## Verification

- Task 1 literal focused gate - passed.
- Task 2 legacy-retirement plus existing production-caller ratchets - passed.
- Explicit migrated behavior set (external replay, parent/child immutability, unfinished-only command, plan-only recovery, child latest exclusion, idempotent retry, and active pause) - passed.
- Eleven critical migration/retirement cases repeated three times - passed in 197.203s.
- The same eleven cases under `go test -race` - passed in 83.141s.
- `go test ./... -run '^$' -count=1` - every Go package compiled successfully.
- `go vet ./cmd`, implementation-range `git diff --check`, exact six-file ownership, and no-deletion checks - passed.
- Protected `.planning/config.json`, `.gsd/`, and Phase 199 PATTERNS dirt remained unstaged and unchanged by implementation commits.

## TDD Gate Compliance

- Task 1 RED named all five remaining executable legacy calls and the direct pause attempt/latest reconstruction. GREEN removed every violation while retaining functional lane assertions.
- Task 2 RED failed on the six obsolete helper declarations/calls and on a start receipt decoded as a fake attempt. GREEN removed the writers and filtered the receipt suffix; both new tests and the pre-existing Plan 34 caller ratchet then passed.

## Deviations from Plan

None - plan and the execution brief were completed within the exact six-file boundary.

## Issues Encountered

- Custom six-task retry states initially lacked an explicit plan policy, so strict build authority correctly refused them. The fixtures now declare the existing legacy-unbound compatibility policy and evidence-not-required policy before their canonical starts; no production authority rule was weakened.
- The active-pause fixture's original September 4 clock predates the shared accepted candidate. Its canonical start therefore correctly classified that candidate as not yet created. The active-work-only clock moved to September 9, after the candidate, while the original pause/resume chronology tests remain unchanged.
- A canonical build start creates legitimate transaction evidence before pause is attempted. The active-boundary assertion was therefore expressed as zero **new** pause receipts (`after == before`) instead of incorrectly requiring the repository to contain zero receipts of any command.

## Known Stubs

None.

## Threat Flags

None - no network endpoint, authentication path, package, schema, or new filesystem trust boundary was introduced. The change removes write surfaces and strengthens executable-source enforcement.

## User Setup Required

None - no packages, credentials, migrations, or external services were added.

## Next Phase Readiness

- Plan 39 can exercise the public planning/build journey knowing every build-start lane now has one atomic entry point.
- Plan 40 retains the final repository-wide normal/race and owner-visible verification receipt.
- No implementation, focused, repeated, race, compile, vet, diff, ownership, deletion, or protected-state blocker remains.

## Self-Check: PASSED

- All six declared source/test files and this summary exist.
- Commits `62c9d5ed`, `20a1ff29`, `9f78dde5`, and `2f11e37e` exist in RED/GREEN order.
- Focused, repeated, race, compile-only, vet, diff, ownership, deletion, and protected-state checks pass.
- Protected `.planning/config.json`, `.gsd/`, and Phase 199 PATTERNS dirt remains present and unstaged.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
