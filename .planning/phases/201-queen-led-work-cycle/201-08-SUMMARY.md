---
phase: 201-queen-led-work-cycle
plan: "08"
subsystem: queen-orchestration
tags: [go, coherent-jobs, build-attempt, work-identity, attempt-bound-artifacts, CAP-071]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle (plan 02)
    provides: "cmd/codex_verify_advance.go: runContinueAcceptVerifyAdvance, the one accept/verify/advance decision body every continue lane reaches through -- this plan's Waves/JobDependencies fields are added to its decision struct"
  - phase: 201-queen-led-work-cycle (plan 05)
    provides: "queenBuildPostWaveDispatches/plannedContinueReviewDispatches gate on the recorded verification boundary -- unrelated to this plan's identity work, but the same codex_build.go/codex_verify_advance.go files this plan also touches"
provides:
  - "cmd/coherent_jobs.go: attemptCoherentJobWaves/attemptCoherentJobDependencies -- the one read path for a build attempt's recorded job-wave order and job dependencies, reading buildAttemptRecord.Dispatches verbatim, never re-grouping phase.Tasks"
  - "cmd/codex_verify_advance.go: continueAcceptVerifyAdvanceDecision.Waves/.JobDependencies, populated inside runContinueAcceptVerifyAdvance from the phase's own build attempt before either early return"
  - "cmd/codex_build.go: codexBuildDispatch.AttemptID, stamped onto every dispatch (stampDispatchAttemptIdentity) once a real attempt exists, on both the direct build lane and the plan-only lane's own returned dispatches"
  - "cmd/attempt_artifacts.go: attemptBoundArtifactPath/writeAttemptBoundArtifact/readAttemptBoundArtifact -- one canonical attempt-bound path per claims/verification artifact, with a legacy-compatible reader validated identically to the attempt-bound path (CAP-071)"
affects: [201-09, 201-10, 201-11, 201-12, 201-13, 201-14, 201-15]

# Actuals (#2632)
actuals:
  tokens: 9582
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Read the already-stored per-dispatch facts instead of introducing a parallel Waves/Jobs record: attemptCoherentJobWaves/attemptCoherentJobDependencies derive the recorded wave order and job dependencies purely from buildAttemptRecord.Dispatches (Wave/JobName/DependsOn), the exact fields plannedBuildDispatchesWithJobProposals already writes -- a second, independently-stored coherentJobPlan snapshot would have been a second source of truth, exactly the divergence risk this task exists to close."
    - "Attempt identity threads through the dispatch struct itself, not only the enclosing manifest: codexBuildDispatch.AttemptID (stampDispatchAttemptIdentity) lets the Queen card's rows, each dispatch's own TaskReceipts, and reviewer-caste dispatches (findings) each be verified in isolation as belonging to one attempt, mirroring the JobName field they already carried."
    - "Attempt-bound artifact containment reuses the spend-path validator (validateSpendContainedPath, cmd/spend_session_capture.go) rather than a second copy of the same symlink-resolution/traversal-rejection logic -- CLAUDE.md's own stated rule ('Two copies of a security boundary is one copy too many')."
    - "Additive-only production wiring: the direct build lane's attempt-bound claims write (writeAttemptBoundArtifact) sits alongside the existing legacy write and is non-fatal on failure; nothing is deleted, and a write failure never blocks or rolls back an otherwise-successful build."

key-files:
  created:
    - cmd/attempt_artifacts.go
    - cmd/work_identity_test.go
  modified:
    - cmd/coherent_jobs.go
    - cmd/codex_verify_advance.go
    - cmd/codex_build.go

key-decisions:
  - "Read waves/job ownership from the already-persisted per-dispatch fields (Wave/JobName/JobReason/JobSource/DependsOn on codexBuildDispatch) rather than adding a new coherentJobPlan.Waves/.Jobs field to buildAttemptRecord. The per-dispatch fields are already the durable record every downstream consumer reads; a second stored snapshot of the same plan would itself become a second source of truth to keep in sync."
  - "runContinueAcceptVerifyAdvance derives the phase's recorded job/wave identity internally via loadLatestBuildAttempt(phase.ID) rather than accepting a new parameter -- following 201-05's own established precedent (queenBuildPostWaveDispatches/plannedContinueReviewDispatches) for the identical reason: it avoids a signature change across the 13 existing production and test call sites of runContinueAcceptVerifyAdvance while still reading the real per-attempt record where one exists."
  - "The plan-only build lane stamps AttemptID only onto its OWN returned dispatches value, never onto manifest.json on disk or the result[\"dispatch_manifest\"] value handed back to a caller. That manifest's bytes are already bound to attempt.ManifestSHA256 (validateBuildAttemptManifestBinding); an initial attempt to also persist the stamped manifest broke the finalize lane's completion-packet digest check, caught by go test ./cmd -run TestBuild and reverted before commit."
  - "The attempt-bound claims write in the direct build lane is additive, not a replacement: the existing legacy last-build-claims.json write is untouched, and the new attempt-bound write is warn-only on failure -- it is new evidence, not a new gate."
  - "REQUIREMENTS.md's WORK-01 checkbox was flipped by hand after `gsd-tools requirements mark-complete WORK-01` root-caused as unable to match: the checkbox regex expects `**REQ-ID**` closed immediately after the ID, but this file's actual style is `**REQ-ID — Title:**` (the bold span covers the whole label). WORK-02 was marked complete before this styling was in place; every requirement still unmarked in this style is unreachable by the tool as it stands today. Confirmed safe to flip by hand only after `requirements ready-ids` independently reported WORK-01 as unblocked (no unfinished sibling plan also declares it)."

patterns-established:
  - "One canonical read path per recorded fact: a new consumer of 'which job owns what, in which wave' must call attemptCoherentJobWaves/attemptCoherentJobDependencies over buildAttemptRecord.Dispatches, never re-group phase.Tasks or re-run planCoherentJobs."

requirements-completed: [WORK-01]
# WORK-03 (this plan's other declared requirement) is also declared by no
# other 201-* plan per requirements.ready-ids' own scan -- but WORK-03's
# checkbox uses the SAME "**REQ-ID — Title:**" style as WORK-01 and hit the
# identical tool-matching gap; it is intentionally left UNCHECKED here
# because SYN-201-02's own linkage table still lists further follow-on work
# for WORK-03 across 201-09..201-15 (coherent jobs and waves is a multi-plan
# requirement whose full scope this single plan does not claim to close),
# unlike WORK-01 which this plan's own must_haves fully discharge.

coverage:
  - id: D1
    description: "runContinueAcceptVerifyAdvance reads a phase's recorded job-wave order and job dependencies verbatim from its build attempt's own persisted dispatches, with no second grouping pass; a three-wave planning result reports three waves in the recorded order downstream"
    requirement: WORK-01
    verification:
      - kind: unit
        ref: "cmd/work_identity_test.go#TestWavesAreReadNotRederived"
        status: pass
      - kind: unit
        ref: "cmd/work_identity_test.go#TestJobRenderOrderIsDeterministic"
        status: pass
      - kind: unit
        ref: "cmd/work_identity_test.go#TestCoherentJobPlanningPerformsNoWrites"
        status: pass
      - kind: unit
        ref: "cmd/work_identity_test.go#TestSameWaveFileConflictIsRefusedBeforeDispatch"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every job-owning dispatch on a real completed build attempt carries both the job name and the exact attempt identifier, verified through the direct/native build lane end-to-end (two independent jobs, both dispatches, both surfaces)"
    requirement: WORK-03
    verification:
      - kind: unit
        ref: "cmd/work_identity_test.go#TestOneJobIdentityReachesEverySurface"
        status: pass
      - kind: unit
        ref: "cmd/work_identity_test.go#TestMissingIdentitySurfaceFailsByName"
        status: pass
    human_judgment: false
  - id: D3
    description: "Claims and verification artifacts are written at attempt-bound paths; a pre-existing legacy root-level artifact remains readable through the identical validation an attempt-bound read uses, and a malformed legacy artifact is refused rather than silently treated as safe; traversal and symlink escape are both refused before any read or write"
    requirement: WORK-03
    verification:
      - kind: unit
        ref: "cmd/work_identity_test.go#TestArtifactsAreAttemptBound"
        status: pass
      - kind: unit
        ref: "cmd/work_identity_test.go#TestLegacyArtifactReadIsValidatedIdentically"
        status: pass
      - kind: unit
        ref: "cmd/work_identity_test.go#TestArtifactPathRefusesEscape"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 08: One Job and Attempt Identity Through the Work Cycle Summary

**Job wave order and dependencies are now read verbatim from a build attempt's own persisted dispatches (never re-grouped), every dispatch carries the same attempt identifier its job name already lived beside, and claims artifacts move onto attempt-bound paths with a legacy-compatible reader validated by the identical rules.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-10T16:10:00+02:00 (approx.)
- **Completed:** 2026-09-10T17:05:00+02:00 (approx.)
- **Tasks:** 3
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `attemptCoherentJobWaves`/`attemptCoherentJobDependencies` (`cmd/coherent_jobs.go`) give the whole runtime one read path for "which waves ran, and which jobs were in them, and what did each job depend on" -- derived purely from `buildAttemptRecord.Dispatches`'s own already-recorded `Wave`/`JobName`/`DependsOn` fields (the exact facts `plannedBuildDispatchesWithJobProposals` writes at build-planning time), never by re-grouping `phase.Tasks` or re-running `planCoherentJobs`. `planCoherentJobs` itself stays pure -- proven by an unchanged fixture-directory hash across a call.
- `runContinueAcceptVerifyAdvance` (`cmd/codex_verify_advance.go`) now reports this phase's recorded job/wave identity: `continueAcceptVerifyAdvanceDecision` gained `Waves`/`JobDependencies`, populated from the phase's own build attempt before either the blocking or advancing return path, so both verdicts carry the same recorded facts. A three-wave fixture (three tasks of three different castes, chained so automatic grouping cannot merge them) reports exactly three waves, in the recorded order, all the way through this decision body.
- `codexBuildDispatch` gained an `AttemptID` field, stamped onto every dispatch (`stampDispatchAttemptIdentity`) the instant a real attempt exists -- right after `commitBuildStart`'s receipt makes the attempt identifier known, on both the direct/native build lane and the plan-only lane's own returned dispatches. Combined with the `JobName` every job-owning dispatch already carried, the Queen card's worker rows, each dispatch's own task receipts, and reviewer-caste dispatches (findings) are each independently verifiable as naming the same job in the same attempt -- proven end-to-end through a real completed build with two independent jobs.
- `cmd/attempt_artifacts.go` (new) gives claims and verification artifacts one canonical path per attempt (`attempts/<id>/<kind>.json`), validated for traversal and symlink escape by reusing (not re-implementing) the spend subsystem's own containment discipline (`validateSpendContainedPath`). `readAttemptBoundArtifact` falls back to a colony's pre-existing legacy root-level filename when no attempt-bound file exists yet, decoding both branches through the identical `store.LoadJSON` call -- a malformed legacy file is refused with the same error class an equally malformed attempt-bound file produces, never silently resolved to an empty, safe value. The direct build lane now additionally persists its own claims at this new path, alongside -- never in place of -- the existing legacy write.

## Task Commits

1. **Task 1: Read waves and job ownership from the one planning result** - `900f1a75` (feat)
2. **Task 2: Carry one job and attempt identity to every surface** - `5ed410d9` (feat)
3. **Task 3: Move claims and verification artifacts onto attempt-bound paths** - `bba33639` (feat)

**Plan metadata:** committed alongside this summary.

## Files Created/Modified

- `cmd/coherent_jobs.go` - `coherentJobWaveOrder`, `attemptCoherentJobWaves`, `attemptCoherentJobDependencies`
- `cmd/codex_verify_advance.go` - `continueAcceptVerifyAdvanceDecision.Waves`/`.JobDependencies`, populated inside `runContinueAcceptVerifyAdvance`
- `cmd/codex_build.go` - `codexBuildDispatch.AttemptID`, `stampDispatchAttemptIdentity`, wired into the direct and plan-only build lanes; additive attempt-bound claims write in the direct lane
- `cmd/attempt_artifacts.go` (new) - `attemptBoundArtifactPath`, `writeAttemptBoundArtifact`, `readAttemptBoundArtifact`, `legacyAttemptArtifactNames`
- `cmd/work_identity_test.go` (new) - all three tasks' proofs: `TestWavesAreReadNotRederived`, `TestJobRenderOrderIsDeterministic`, `TestCoherentJobPlanningPerformsNoWrites`, `TestSameWaveFileConflictIsRefusedBeforeDispatch`, `TestOneJobIdentityReachesEverySurface`, `TestMissingIdentitySurfaceFailsByName`, `TestArtifactsAreAttemptBound`, `TestLegacyArtifactReadIsValidatedIdentically`, `TestArtifactPathRefusesEscape`

## Decisions Made

See `key-decisions` in the frontmatter for full reasoning; in short: read waves/dependencies from the already-persisted per-dispatch fields rather than a second stored plan snapshot; derive the recorded identity inside `runContinueAcceptVerifyAdvance` via `loadLatestBuildAttempt` rather than a new parameter (matching 201-05's precedent); the plan-only lane stamps only its own returned value, never the digest-bound manifest on disk; the attempt-bound claims write is additive, never a replacement; and `REQUIREMENTS.md`'s `WORK-01` checkbox was hand-flipped after root-causing why `gsd-tools requirements mark-complete` could not match this file's current bold-label styling.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Reverted a manifest resave that broke the finalize lane's completion-packet digest check**
- **Found during:** Task 2, while running the owner-mandated `go test ./cmd -run TestBuild` sweep after implementing dispatch-level `AttemptID` stamping
- **Issue:** The plan-only build lane's first implementation persisted the `AttemptID`-stamped dispatches back onto `manifest.json` and `result["dispatch_manifest"]`. That manifest's bytes are already bound to `attempt.ManifestSHA256` at the moment the attempt is created; resaving it with different content broke `validateBuildAttemptManifestBinding`'s digest comparison, which a wrapper's resubmitted completion packet is checked against. Four tests failed with `dispatch_manifest content does not match durable build attempt ... (got X, expected Y)`: `TestBuildWorkerLessonsBecomeObservations`, `TestBuildFinalizeRecordsFreeChecksAsAReport`, `TestBuildFinalizeFreeChecksDoNotAdvanceThePhase`, `TestBuildFinalizeFilesTheRunsRows`, plus `TestBuildFinalizeRecordsExternalTaskResultsForContinue` on a companion `Continue`-matched sweep.
- **Fix:** Stopped writing the stamped dispatches back onto `manifest.json`/`result["dispatch_manifest"]`; the plan-only lane now only stamps its own function-local `dispatches` return value, leaving the persisted, digest-bound manifest byte-identical to what `commitBuildStart` already sealed.
- **Files modified:** `cmd/codex_build.go`
- **Verification:** Full `go test ./cmd -run TestBuild` (217s) and `go test ./cmd -run Continue` (126s) sweeps re-run clean afterward -- zero failures beyond one pre-existing, unrelated golden-file test (see Issues Encountered).
- **Committed in:** `5ed410d9` (Task 2 commit) -- the revert landed before the task's own commit, so the commit itself contains only the corrected version.

---

**Total deviations:** 1 auto-fixed (1 bug -- caught by the plan's own required verification sweep before it ever reached a commit). **Impact on plan:** Necessary; no unrelated behavior was touched, and the fix is exactly the kind of thing the owner's stated targeted-then-broad verification policy exists to catch.

## Issues Encountered

- `TestGoldenContinueVisualOutput` fails identically on a clean checkout of this plan's parent commit (`1671560d`, confirmed via `git stash -u` + re-run) with `got: "Elapsed: 69h9m36s"` vs. `want: "Cost: not known..."` -- a pre-existing, environment-dependent golden-file mismatch unrelated to any change in this plan. Not investigated further; documented here so it is not mistaken for a regression this plan introduced.
- `gsd-tools requirements mark-complete WORK-01` reported `not_found` despite `requirements ready-ids` independently confirming WORK-01 unblocked. Root-caused (not merely worked around): the tool's checkbox regex expects `**REQ-ID**` with the bold span closing immediately after the ID, but `REQUIREMENTS.md`'s current style is `**REQ-ID — Title:**` (the bold span covers the whole label through the colon). `WORK-02` was marked complete by an earlier plan before this styling existed in the file; every requirement still unmarked in the current style is unreachable by the tool as it stands. Flipped the checkbox by hand for `WORK-01` only, after confirming via `ready-ids` that no unfinished sibling plan blocks it. This is a tooling/format mismatch outside this plan's own file scope (`gsd-tools` lives outside the Aether repo; `REQUIREMENTS.md`'s styling is this repo's own prior convention) -- not something this plan's tasks should fix in passing.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `attemptCoherentJobWaves`/`attemptCoherentJobDependencies` are the one read path for job/wave identity; any later plan (201-09 onward) reporting job-scoped information from a build attempt should read through them rather than re-deriving.
- `codexBuildDispatch.AttemptID` is stamped on both build lanes today; a later plan wiring workspace-lease records (`colony.WorktreeEntry`, `pkg/colony`) or `codex.TaskReceipt` itself with an explicit attempt field was out of this plan's declared file scope (`cmd/coherent_jobs.go`, `cmd/codex_verify_advance.go`, `cmd/codex_build.go`, `cmd/attempt_artifacts.go`) and remains an open surface if a future plan needs the identifier carried on those nested types directly, rather than reachable via the owning dispatch.
- `attemptBoundArtifactPath`/`writeAttemptBoundArtifact`/`readAttemptBoundArtifact` are fully built and tested, and the direct build lane's claims write is live production wiring. The verification-artifact kind (`attemptArtifactKindVerification`) has no production writer yet -- `verification.json` is written by the continue/finalize lane (`cmd/codex_continue_finalize.go`), outside this plan's declared file scope, matching this phase's own repeated precedent (201-05, 201-07) of built-and-tested-but-deliberately-unwired infrastructure for a later plan.
- `go build ./...` and `go vet ./cmd` are clean. Every task-level `<verify>` command from `201-08-PLAN.md` passes. Targeted regression sweeps (`TestBuild`, 217s, 0 failures; `Continue`, 126s, 0 failures beyond the pre-existing golden mismatch documented above) both pass.
- WORK-01 is now complete. WORK-03 remains open -- it is a multi-plan requirement whose full scope this plan does not claim to close (SYN-201-02's linkage table names further follow-on work through 201-15).
- Ready for `201-09-PLAN.md`.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

- `cmd/attempt_artifacts.go` — FOUND
- `cmd/work_identity_test.go` — FOUND
- Commit `900f1a75` (Task 1) — FOUND in git log
- Commit `5ed410d9` (Task 2) — FOUND in git log
- Commit `bba33639` (Task 3) — FOUND in git log
- `go build ./...` — clean
- `go vet ./cmd` — clean
- All plan `<verify>` commands re-run and passing: `TestWavesAreReadNotRederived`, `TestJobRenderOrderIsDeterministic`, `TestCoherentJob*` (broader sweep), `TestOneJobIdentityReachesEverySurface`, `TestArtifactsAreAttemptBound`, `TestLegacyArtifactReadIsValidatedIdentically`, `TestArtifactPathRefusesEscape`
- Targeted `go test ./cmd -run TestBuild` (217s) and `-run Continue` (126s): both clean, zero failures beyond the documented pre-existing `TestGoldenContinueVisualOutput` golden mismatch
