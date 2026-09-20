---
phase: 200-iterative-planning
plan: 33
subsystem: build-lifecycle
tags: [go, tdd, transactions, idempotency, authority-binding, fault-injection]

requires:
  - phase: 200-28
    provides: Repository-scoped planning mutation sessions and session-aware lifecycle transactions
  - phase: 200-31
    provides: Canonical accepted plan/specification/candidate/timeline authority
  - phase: 200-32
    provides: Session-first planning writers and exact authority-artifact baselines
provides:
  - One canonical request and closed target matrix for every named build-start variant
  - Pure deterministic attempt journal and latest-pointer derivation from explicit inputs
  - Session-held all-target commit, rollback, durable receipt, and exact read-only replay
affects: [plan-34-build-caller-migration, build, continue, retry, reviewer-window]

tech-stack:
  added: []
  patterns: [closed effect matrix, derive-then-commit, receipt-last publication, post-commit dispatch]

key-files:
  created:
    - cmd/build_start_transaction.go
    - cmd/build_start_transaction_200_test.go
  modified:
    - cmd/build_attempt.go

key-decisions:
  - "Every named build-start variant maps to one finite required/allowed effect rule; unsupported effect combinations fail before mutation."
  - "Build start holds the Plan 28 repository session across Plan 31 authority reload, deterministic target derivation, exact-baseline validation, commit, and rollback."
  - "The content-addressed build-start receipt is the final declared target; exact replay validates its request, authority, identity, target set, actions, and digests without dispatching again."
  - "Attempt derivation accepts clock, process, run, platform, workspace, task, and provenance inputs explicitly while legacy writers remain thin compatibility adapters."

patterns-established:
  - "Closed build-start matrix: variant rules reject both missing required effects and undeclared optional effects before target preparation."
  - "Durable dispatch boundary: worker callbacks can run only after the receipt-last lifecycle transaction commits successfully."

requirements-completed: [PLAN-06]

duration: 30min
completed: 2026-09-09
---

# Phase 200 Plan 33: Atomic Build-Start Transaction Summary

**Canonical authority-bound build starts now derive every attempt and conditional effect up front, commit them all-or-none under one repository session, and expose dispatch only after a durable receipt.**

## Performance

- **Duration:** 30 min
- **Started:** 2026-09-09T01:02:23Z
- **Completed:** 2026-09-09T01:32:16Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments

- Defined a closed request/effect/receipt contract for direct, plan-only, Queen-led, externally finalized unbound, automatic check-fix, and coherent child-retry starts.
- Extracted deterministic attempt ID, journal, latest pointer, execution binding, parent retry, and check-fix derivation while preserving the legacy writer as a compatibility adapter.
- Reloaded canonical accepted authority and captured every state, authority, artifact, cleanup, reviewer, and receipt baseline within one repository session.
- Made the build-start receipt the last transaction target, rejected conflicting or semantically forged replay, and prevented every faulted or stale start from reaching the dispatch callback.

## Task Commits

Each TDD gate and implementation task was committed atomically:

1. **Task 1 RED: Define the complete build-start target and fault matrix** - `d230a13e` (test)
2. **Task 2 GREEN: Build canonical requests and pure attempt records** - `5461987a` (feat)
3. **Task 3 GREEN: Commit all start effects under one repository session** - `03af1d17` (feat)

## Files Created/Modified

- `cmd/build_start_transaction.go` - Closed variants, canonical request/effect validation, authority reload, target preparation, lifecycle commit, rollback, receipt, and replay validation.
- `cmd/build_attempt.go` - Pure attempt identity, journal, and latest-pointer derivation shared by canonical and legacy start paths.
- `cmd/build_start_transaction_200_test.go` - Six-variant target matrix plus fault, rename, stale-authority, accepted-authority, replay, reviewer, and no-global-store proofs.

## Decisions Made

- A variant owns an exact effect shape. Conditional stale paths and reviewer files remain explicit targets, while attempt and receipt targets are mandatory for every variant.
- Reviewer-window and pending-decision bytes are derived from session reads, so their conditional presence is still baseline-protected and part of the same commit.
- Replay trusts neither mutable attempt/state projections nor a hash alone: it re-derives the request identity and current authority, then validates receipt semantics and required target actions.
- Production callers remain intentionally untouched until Plan 34; `commitBuildStart` currently has no production call site.

## Verification

- Task 2 focused transaction/build-attempt suite - 56 passed.
- Final focused transaction/build-attempt suite - 62 passed.
- Fault and exact-replay tests repeated three times - 111 passed.
- Full focused transaction/build-attempt suite under `go test -race` - passed in 38.164s.
- Linked Plan 28 session, Plan 31 acceptance, and lifecycle transaction regressions - 77 passed.
- `go vet ./cmd`, `git diff --check f3dad75f..HEAD`, no-production-caller scan, and exact three-file ownership check - passed.
- Protected `.planning/config.json`, `.gsd` content excluding its orchestrator sentinel, Phase 199 evidence, and `200-VERIFICATION.md` retained their starting SHA-256 hashes.

## TDD Gate Compliance

- RED commit `d230a13e` failed because the canonical build-start API did not yet exist.
- GREEN commits `5461987a` and `03af1d17` implemented pure derivation and session-held commit after the fail-first contract.
- A Task 3 forged-receipt test also failed before semantic replay validation was tightened, then passed with the implementation.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. The two known categories of older hand-built planning fixtures were outside this scoped verification and were neither changed nor used to weaken canonical validation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 34 can migrate production build-start callers onto `commitBuildStart` without redefining authority, attempt, effect, or replay semantics.
- No Plan 33 correctness or scope blocker remains; raw repository-wide tests remain intentionally outside this plan because their known legacy fixture failures are unrelated.

## Self-Check: PASSED

- Summary and all three declared source/test files exist.
- Task commits `d230a13e`, `5461987a`, and `03af1d17` exist in repository history.
- The implementation range contains exactly the three plan-owned paths, and `commitBuildStart` has no production caller before Plan 34.
- All protected paths retain their recorded starting hashes.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
