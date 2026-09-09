---
phase: 200-iterative-planning
plan: 37
subsystem: planning-authority
tags: [go, tdd, candidate-expiry, injected-clock, atomic-transition, build-authority]

requires:
  - phase: 200-31
    provides: Immutable candidate, acceptance receipt, timeline, and active plan-revision authority
  - phase: 200-32
    provides: Session-atomic plan refresh and candidate cleanup/regeneration
provides:
  - One pure current, expired, stale, or accepted candidate-standing assessment shared across review, acceptance, build, and run authority
  - Exact acceptance-deadline enforcement with an atomic pending-review to expired transition and unchanged active plan
  - Token-free stale and expired review with typed refusal facts and the existing aether plan --refresh recovery
  - Build/run rejection of forged late receipts while preserving timely accepted authority after its former deadline
affects: [planning-review, plan-acceptance, build-authority, run-authority, plan-refresh]

tech-stack:
  added: []
  patterns: [single injected clock sample, pure standing assessment, typed refusal facts, atomic expiry transition]

key-files:
  created:
    - cmd/plan_candidate_expiry_200_test.go
  modified:
    - pkg/colony/planning.go
    - cmd/planning_stage.go
    - cmd/plan_candidate.go
    - cmd/plan_revision.go
    - cmd/plan_authority.go

key-decisions:
  - "ExpiresAt is an acceptance deadline: pending candidates are eligible only while observed time is strictly before it, but a timely accepted candidate remains authoritative afterward."
  - "Review and command acceptance use one package clock seam; the command samples once and passes that exact UTC instant through the repository session."
  - "Expiry uses the existing terminal failed planning stage with the exact candidate_expired reason, preserving the closed stage vocabulary while recording durable expiry."
  - "Planning-stage snapshot base identity stays distinct from immutable candidate-base identity so phase-insert candidates retain their established authority semantics."

patterns-established:
  - "Temporal authority: derive standing from immutable candidate content, current canonical bindings, and one caller-supplied time; never read the clock inside the pure policy."
  - "Safe refusal: return typed facts and a non-nil error together, omit stale tokens/commands, and name both state and active-plan effects."

requirements-completed: [PLAN-04, PLAN-05, PLAN-06]

duration: 34min
completed: 2026-09-09
---

# Phase 200 Plan 37: Candidate Expiry and Stale Recovery Summary

**Plan candidates now have one exact acceptance deadline across review, acceptance, build, and run, with atomic expiry, safe refresh guidance, and permanent authority for receipts accepted in time.**

## Performance

- **Duration:** 34 min
- **Started:** 2026-09-09T02:08:06Z
- **Completed:** 2026-09-09T02:41:42Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- Added a pure standing matrix covering specification, base revision, proposal, timeline, candidate body, stage, clock monotonicity, and exact deadline boundaries.
- Made stale and expired candidates fully reviewable without exposing an acceptance token or command; the only recovery is `aether plan --refresh`.
- Made exact-boundary and later acceptance atomically persist candidate expiry and its terminal stage/index authority while leaving the active plan unchanged.
- Returned stable typed refusal facts together with the command error, including standing, expiry, evidence, state effect, active-plan effect, and recovery.
- Revalidated timely receipt creation at build/run authority, rejecting forged at-deadline receipts without expiring legitimately accepted plans later.

## Task Commits

Each task followed a fail-first TDD boundary and was committed atomically:

1. **Task 1 RED: Define temporal boundaries and every early-stale reason** - `b8264507` (test)
2. **Task 2 GREEN: Enforce one canonical standing in review and acceptance** - `b59ab68a` (feat)
3. **Task 3 GREEN: Validate timely acceptance again at build and run authority** - `1d09365b` (fix)

## Files Created/Modified

- `cmd/plan_candidate_expiry_200_test.go` - Injected-clock boundary, stale-binding, atomic expiry, safe review, replay, and execution-authority proofs.
- `pkg/colony/planning.go` - Requires candidate expiry to be strictly later than creation.
- `cmd/planning_stage.go` - Defines the exact durable `candidate_expired` terminal failure reason.
- `cmd/plan_candidate.go` - Pure candidate standing assessment, safe review projection, one command clock seam, and typed refusal schema.
- `cmd/plan_revision.go` - Session-bound acceptance/replay policy and atomic expiry transition.
- `cmd/plan_authority.go` - Shared standing revalidation for build/run authority and strict late-receipt rejection.

## Decisions Made

- The exact boundary is closed: `now < ExpiresAt` is eligible; `now == ExpiresAt` is expired.
- An accepted receipt is checked before later wall-clock expiry, so valid timely acceptance does not decay.
- The public review path renders safe stale/expired facts, while the existing strict internal loader continues returning integrity errors to callers that depend on that contract.
- The atomic expiry transaction may write its normal transaction proof artifacts, but it cannot mutate or activate the active plan.
- Direct legacy in-process acceptance callers that omit `AcceptedAt` use the same injectable clock seam once; the public command always supplies its single already-sampled value.

## Verification

- Task 1 focused expiry suite - passed (18 tests).
- Task 2 candidate and colony model suite - passed (189 tests across 2 packages).
- Task 3 candidate/build/run authority suite - passed (53 tests).
- Combined candidate and authority suite repeated three times - passed (672 tests across 2 packages).
- Combined candidate and authority suite under `go test -race` - passed (224 tests across 2 packages; no data races).
- `go vet ./cmd ./pkg/colony`, implementation-range `git diff --check`, exact six-file ownership, and deletion checks - passed.
- The known repository-wide `go test ./...` command was not run because this plan calls for bounded candidate/authority verification and the repository documents that command's approximately 11-minute runtime.

## TDD Gate Compliance

- RED commit `b8264507` failed on the deliberately missing standing vocabulary, pure assessment, injected clock seam, refusal schema, and authority enforcement.
- GREEN commit `b59ab68a` made the complete review/acceptance matrix pass, including exact-boundary atomic expiry and typed refusal facts.
- Task 3 retained a failing forged-at-deadline authority proof until `1d09365b` connected build/run validation to the shared standing assessment.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Compatibility Bug] Preserved strict internal candidate review while adding safe public review**
- **Found during:** Task 2 candidate regression suite
- **Issue:** Sending every caller through tolerant stale review changed 110 existing integrity-mutation tests that intentionally require direct review to fail closed.
- **Fix:** Kept `reviewPlanCandidate` strict and routed the public command through `reviewPlanCandidateAt`, which alone produces the required read-only stale/expired presentation.
- **Files modified:** `cmd/plan_candidate.go`
- **Verification:** Task 2 suite passed 189 tests.
- **Committed in:** `b59ab68a`

**2. [Rule 1 - Compatibility Bug] Kept legacy in-process acceptance callers on the injected clock**
- **Found during:** Task 2 implementation
- **Issue:** Existing internal tests and callers omit `AcceptedAt`; removing the fallback outright would break them even though the public command correctly owns the single clock sample.
- **Fix:** Zero-valued internal calls sample the same injectable clock once, while `runPlanCandidateCommand` always supplies its already-sampled time and performs no second read.
- **Files modified:** `cmd/plan_revision.go`, `cmd/plan_candidate.go`
- **Verification:** One-clock-read canary and full Task 2 suite passed.
- **Committed in:** `b59ab68a`

**3. [Rule 1 - Compatibility Bug] Separated phase-insert stage snapshot base from immutable candidate base**
- **Found during:** Task 3 authority regression suite
- **Issue:** Phase-insert stages intentionally bind the full mutable plan-state snapshot, while their candidates bind the immutable active revision. Treating those hashes as one field refused a valid accepted phase insertion.
- **Fix:** Carried the independently verified stage-base expectation into the standing assessment instead of conflating it with candidate-base authority.
- **Files modified:** `cmd/plan_candidate.go`, `cmd/plan_authority.go`
- **Verification:** The formerly failing phase-insert authority regression passed, followed by all 53 Task 3 tests, 672 repeated tests, and 224 race tests.
- **Committed in:** `1d09365b`

---

**Total deviations:** 3 auto-fixed (3 Rule 1 compatibility bugs)
**Impact on plan:** All fixes preserve existing supported authority contracts while enforcing the planned deadline; no command, schema, dependency, or out-of-scope production file was added.

## Issues Encountered

- The first broad Task 3 pass exposed the phase-insert dual-base semantics described above; it was resolved within the plan's declared files.
- No authentication, package, architectural, or external-service blocker occurred.

## Known Stubs

None.

## User Setup Required

None - no packages, credentials, migrations, or external services were added.

## Next Phase Readiness

- Later planning/build work can consume one stable candidate-standing contract rather than re-deriving expiry independently.
- `aether plan --refresh` is the existing deterministic recovery for every stale or expired review.
- No Plan 37 implementation, race, verification, scope, or protected-state blocker remains.

## Self-Check: PASSED

- All six declared source/test files and this summary exist.
- Commits `b8264507`, `b59ab68a`, and `1d09365b` exist in repository history.
- Focused, repeated, race, vet, diff, ownership, deletion, and protected-state checks pass.
- Protected `.planning/config.json`, `.gsd/`, and Phase 199 PATTERNS dirt remains present and unstaged; the two byte-hashed files exactly match their pre-execution hashes.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
