---
phase: 201-queen-led-work-cycle
plan: "03"
subsystem: queen-orchestration
tags: [go, verification-boundary, queen-judgement, build-attempt, ast-guard]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle (plan 01)
    provides: "SYN-201-03's cited synthesis decision -- a new sibling type (verificationBoundaryDecision{Choice, Reason, Source}) that references, but does not extend, queenCasteJudgement, resolving RESEARCH.md Open Question 1"
provides:
  - "cmd/verification_boundary.go: queenApplyVerificationBoundary, a pure function reconciling the Queen's proposed verification-boundary choice (check_step vs build_end) into a validated verificationBoundaryDecision, defaulting to check_step with no proposal (D-01) and refusing an unreasoned build_end request (D-02)"
  - "buildAttemptRecord.VerificationBoundary (append-safe field) plus attachVerificationBoundary, a narrow setter that binds the decision to the exact attempt and refuses to silently rewrite an already-recorded choice"
  - "verificationBoundaryForAttempt: the single read path every consumer of a recorded decision must use -- it reads, it never re-derives"
  - "An AST guard (TestOneFunctionDerivesTheVerificationBoundary) proving, from the parsed package syntax tree, that only the reconciliation function and the stored-record accessor ever produce a verificationBoundaryDecision"
affects: [201-04, 201-05, 201-06, 201-07, 201-15]

# Actuals (#2632)
actuals:
  tokens: 7392
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Narrow attempt-bound setter (mirrors attachBuildFreeCheckReport/attachCheckFixAttempt precedent): attachVerificationBoundary touches exactly one field via store.UpdateJSONAtomically, is idempotent on a re-attach of the identical decision, and refuses (naming both choices) a re-attach of a different one -- so a recorded boundary can never be silently rewritten mid-flight"
    - "AST producer-set guard (mirrors codex_verify_advance_test.go's continueDecisionOffenders precedent): the guard parses the cmd package, computes which top-level functions return the decision type from the syntax tree, and fails any name outside the two permitted producers -- the offender list is never a hand-maintained enumeration of legitimate callers"

key-files:
  created:
    - cmd/verification_boundary.go
    - cmd/verification_boundary_test.go
  modified:
    - cmd/build_attempt.go

key-decisions:
  - "queenApplyVerificationBoundary accepts phase and state parameters for call-shape parity with queenApplyJudgement and future evidence-bearing rules, but the current reconciliation logic consults neither -- the decision is derived purely from the proposal string and its reason, matching the plan's own must_haves (no depth flag, no state read)."
  - "Normalisation is trim-plus-ASCII-lowercase followed by EXACT string comparison against the two closed-vocabulary constants -- no strings.EqualFold, no separator-substitution fallback (unlike queen_judgement.go's caste-name resolution), and no fuzzy matching. A near-miss like \"build-end\" (hyphen instead of underscore) is refused, not silently resolved, matching the plan's explicit no-folding requirement."
  - "The AST guard's two permitted-producer names (queenApplyVerificationBoundary, verificationBoundaryForAttempt) are hardcoded as the shared INFRASTRUCTURE the guard protects -- mirroring continueDecisionGateEvaluatorNames' own documented precedent in cmd/codex_verify_advance_test.go -- while the actual offender set is always computed from the parsed declarations, never a maintained list of every legitimate caller."

patterns-established:
  - "Verification-boundary decisions are recorded once per attempt and read everywhere else -- no consumer may call queenApplyVerificationBoundary a second time to recompute a choice already on record for that attempt."

requirements-completed: []
# WORK-01 and WORK-04 (this plan's declared requirements) are each also
# declared by sibling 201-* plans (WORK-01: 201-08; WORK-04: 201-02
# [complete], 201-05, 201-15) and therefore stay open per the shared-ID gate
# (#2388) until every declaring plan has a SUMMARY.md -- confirmed via
# `gsd-tools query requirements.ready-ids` (0/2 ready), correct and expected,
# not a gap in this plan's own work.

coverage:
  - id: D1
    description: "A pure reconciliation function (queenApplyVerificationBoundary) turns the Queen's proposed verification-boundary choice into a validated decision: defaults to the check step with no proposal, accepts an explicit check-step or build-end proposal, refuses an unreasoned build-end request or an unrecognised string by name and falls back to the check step, and normalises whitespace/ASCII-case without fuzzy matching"
    requirement: WORK-01
    verification:
      - kind: unit
        ref: "cmd/verification_boundary_test.go#TestVerificationBoundaryDefaultsToTheCheckStep"
        status: pass
      - kind: unit
        ref: "cmd/verification_boundary_test.go#TestVerificationBoundaryRefusesBuildEndWithoutAReason"
        status: pass
      - kind: unit
        ref: "cmd/verification_boundary_test.go#TestVerificationBoundaryNormalisesProposalExactly"
        status: pass
      - kind: unit
        ref: "cmd/verification_boundary_test.go#TestVerificationBoundarySummaryIsStable"
        status: pass
    human_judgment: false
  - id: D2
    description: "The reconciled decision is bound durably to the exact build attempt via a narrow setter that touches only that one field, is idempotent on a repeat of the same decision, refuses to silently rewrite a different one, and decodes cleanly (as 'no decision recorded') on attempt JSON written before this field existed"
    requirement: WORK-04
    verification:
      - kind: unit
        ref: "cmd/verification_boundary_test.go#TestAttachVerificationBoundaryTouchesOneField"
        status: pass
      - kind: unit
        ref: "cmd/verification_boundary_test.go#TestVerificationBoundaryCannotBeSilentlyRewritten"
        status: pass
      - kind: unit
        ref: "cmd/verification_boundary_test.go#TestLegacyAttemptWithoutBoundaryDecodes"
        status: pass
      - kind: unit
        ref: "cmd/verification_boundary_test.go#TestVerificationBoundaryForAttemptReadsTheStoredRecordOnly"
        status: pass
    human_judgment: false
  - id: D3
    description: "An AST guard refuses, by name and file position, any function in the cmd package other than the reconciliation function and the stored-record accessor that independently produces a verificationBoundaryDecision -- the permitted-producer set is computed from parsed declarations, not a maintained list"
    requirement: WORK-04
    verification:
      - kind: unit
        ref: "cmd/verification_boundary_test.go#TestOneFunctionDerivesTheVerificationBoundary"
        status: pass
    human_judgment: false

duration: 50min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 03: Verification-Boundary Decision Summary

**A pure reconciliation function, an attempt-bound durable record, and one read path make "does reviewer judgement land at build-end or at the check step" a single validated fact instead of two independently-derived guesses -- enforced structurally by an AST guard, not discipline.**

## Performance

- **Duration:** 50 min
- **Started:** 2026-09-10T14:05:00+02:00 (approx.)
- **Completed:** 2026-09-10T14:55:00+02:00 (approx.)
- **Tasks:** 3
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments

- Created `cmd/verification_boundary.go` with `queenApplyVerificationBoundary`, the pure reconciliation function that turns a proposed boundary choice into a validated `verificationBoundaryDecision`. Defaults to `check_step` (the safe default, D-01) when no proposal arrives; requires a non-blank reason for `build_end` (D-02), refusing and falling back to `check_step` when one is missing; normalises whitespace/ASCII-case with exact comparison against the two-value closed vocabulary and no fuzzy matching.
- Added `buildAttemptRecord.VerificationBoundary` (an append-safe, `omitempty` pointer field) and `attachVerificationBoundary` in `cmd/build_attempt.go`, mirroring `attachBuildFreeCheckReport`'s and `attachCheckFixAttempt`'s narrow-setter discipline: it touches only this one field, is a no-op success on re-attaching the identical decision, and refuses (naming both the stored and offered choice) a rewrite attempt.
- Added `verificationBoundaryForAttempt` as the single read path every consumer must use -- it reads the stored decision and returns `ok=false` for an attempt with none recorded, including attempts written before this field existed (proven with a real derived-and-saved fixture whose raw JSON contains no `verification_boundary` key).
- Added `TestOneFunctionDerivesTheVerificationBoundary`, an AST guard that parses the `cmd` package with `go/parser`/`go/ast`, computes -- from the actual declarations, not a maintained list -- every top-level function whose return type includes `verificationBoundaryDecision`, and fails by name and file:line position when anything other than the reconciliation function or the stored-record accessor produces one. Proven against both the real package (exactly the two permitted producers) and a synthetic third producer fixture.

## Task Commits

1. **Task 1: Reconcile the Queen's boundary proposal into a validated decision** - `8a6ea3fa` (feat)
2. **Task 2: Bind the decision to the exact attempt and give it one read path** - `c822a667` (feat)
3. **Task 3: Refuse a second derivation of the boundary** - `084a1b7b` (test)

**Plan metadata:** committed alongside this summary.

## Files Created/Modified

- `cmd/verification_boundary.go` - `verificationBoundaryDecision` type, the choice/source constants, `Summary()`, `queenApplyVerificationBoundary` (the reconciliation function), and `verificationBoundaryForAttempt` (the single read path)
- `cmd/verification_boundary_test.go` - Unit coverage of the reconciliation function's own rules (default, refusal, normalisation, summary stability), the attempt-binding narrow setter (field isolation, idempotent re-attach, rewrite refusal, legacy-JSON decoding), and the AST single-producer guard
- `cmd/build_attempt.go` - `buildAttemptRecord.VerificationBoundary` field and `attachVerificationBoundary`, placed beside `attachBuildFreeCheckReport`/`attachCheckFixAttempt`

## Decisions Made

- `queenApplyVerificationBoundary` accepts `phase` and `state` for call-shape parity with `queenApplyJudgement` and to leave room for future evidence-bearing rules, but the current reconciliation logic is derived purely from the proposal string and its reason -- no depth flag, no state read, matching the plan's own must-haves for a pure function.
- Normalisation is trim-plus-ASCII-lowercase followed by exact string comparison, deliberately without `strings.EqualFold` or the separator-substitution fallback `resolveCasteName` uses elsewhere in the package -- a near-miss like `build-end` (hyphen) is refused by name rather than silently resolved, per the plan's explicit "no folding, no prefix matching" requirement.
- The AST guard's two permitted-producer names are hardcoded as the shared infrastructure the guard protects (mirroring `continueDecisionGateEvaluatorNames`' own documented precedent in `cmd/codex_verify_advance_test.go`), while the actual offender set the guard reports is always computed from the parsed declarations found in the package -- never a maintained list of every legitimate caller.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The full, unfiltered `go test ./cmd` package suite (555s, 4,794+ tests) hit the same pre-existing, environment-level resource-exhaustion condition documented in `201-02-SUMMARY.md`'s own Issues Encountered section: `TestFullSuiteRuntime200*`-style parallel "lane" fixtures (a separate test area that spawns many isolated child OS processes) hit `isolated process child hub setup deadline: context deadline exceeded` under the full package's combined parallel load, so dozens of unrelated lanes report "missing executed tests" -- tests that were never scheduled, not tests that failed. This is confirmed pre-existing and unrelated to this plan's changes: (1) zero `--- FAIL` lines appear anywhere in the run's output (`grep -c '^--- FAIL'` = 0); (2) every one of this plan's own new tests (`TestVerificationBoundary*`, `TestAttachVerificationBoundaryTouchesOneField`, `TestVerificationBoundaryCannotBeSilentlyRewritten`, `TestLegacyAttemptWithoutBoundaryDecodes`, `TestOneFunctionDerivesTheVerificationBoundary`) appears in the run's "missing executed tests" lists too (never scheduled in that particular parallel arrangement), while every one of them passed cleanly, repeatedly, in targeted runs: individually (per-task `<verify>` commands, all passing), in a broader `-run 'Boundary'` sweep (79s, also exercises pre-existing `build_attempt_test.go` coverage), in `-run 'BuildAttempt'` (45s), and in `-run 'Continue'` (124s, the three continue lanes from plan 201-02). `go build ./...` and `go vet ./...` are clean. This matches the project's own documented "Suite ceiling measured" finding (cmd suite floors ~20min complete; kernel process-spawn cap under heavy parallel load) -- a known environment limitation, not a defect this plan introduced or could fix.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `verificationBoundaryDecision`, `queenApplyVerificationBoundary`, `attachVerificationBoundary`, and `verificationBoundaryForAttempt` are ready for plan 201-05 ("Eliminate double verification and render the honest unverified build card") to wire into the two currently-independent dispatch sites (`queenBuildPostWaveDispatches` and `queenContinueDispatchesWithJudgement`, `CONCERNS.md`'s "verified twice" defect) -- this plan deliberately built only the recording mechanism, per its own objective ("Recording it once... is what lets plan 201-05 delete one of the two review passes safely"); the wiring itself is out of scope here.
- `go build ./...` and `go vet ./...` are clean. Every task-level `<verify>` command from `201-03-PLAN.md` passes. Targeted regression sweeps (`Boundary`, `BuildAttempt`, `Continue`) all pass; the full unfiltered `go test ./cmd` run hit only the documented pre-existing environment ceiling (see Issues Encountered), not a regression.
- WORK-01 and WORK-04 remain open in `REQUIREMENTS.md` (correctly -- both are shared with plans still in progress: 201-08 for WORK-01, 201-05/201-15 for WORK-04) and will close once every declaring plan has summarized.
- Ready for `201-04-PLAN.md`.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*
