---
phase: 208-never-a-dead-end
plan: 05
subsystem: cli
tags: [go, criterion-evidence, refusal, owner-confirmation, cli]

# Dependency graph
requires:
  - phase: 208-01
    provides: "The typed refusal contract (refuse(), refusalRegistry, renderRefusal) both new refusal rows in this plan register against"
provides:
  - "validatePhaseCriterionEvidenceAgainstDisk: refuses a directory (or a shortcut to one) bound as a criterion artifact, by name, before any worker is dispatched"
  - "criterionBindingIsUnsatisfiable + owner-confirmation routing: an already-bound, in-progress phase stuck on a directory binding advances via the existing owner-confirmation exit instead of failing forever"
  - "aether skip-phase --force now names the owner-answer route first, before confirming the abandon"
affects: [208-06, 208-07, 208-08]

# Actuals (#2632)
actuals:
  tokens: 10321
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "criterionArtifactBindingIsDirectory: one shared lstat-and-mode check (mirrors snapshotBuildArtifact's own discipline) used by both the build-time refusal and the run-time unsatisfiable-binding check, rather than two separate directory tests that could drift apart"
    - "codexBuildClaims.RejectedArtifacts: attachBuildArtifactEvidence now records WHY a claimed path could not become evidence (a folder, not merely 'unclaimed'), computed wholesale from the claimed file lists the same way ArtifactEvidence already is -- never accepted from a worker's own completion packet"
    - "criterionBindingIsUnsatisfiable reuses the existing needs_owner_confirmation state and BlockingIssues-vs-ownerConfirmationIssues split, rather than adding a second owner-routing mechanism"

key-files:
  created:
    - cmd/criterion_binding_dead_end_test.go
  modified:
    - cmd/criterion_evidence.go
    - cmd/codex_build.go
    - cmd/refusal_register.go
    - cmd/criterion_owner_confirmation.go
    - cmd/phase_skip.go

key-decisions:
  - "The plan named cmd/skip_phase_cmd.go; the real file is cmd/phase_skip.go (skip-phase's actual implementation). Edited the real file -- see Deviations."
  - "Registering the two new refusal rows required editing cmd/refusal_register.go, which was not in either task's declared files list. Edited it anyway -- the refusal contract (Phase 208-01) has exactly one checked-in table, and there is no other place a new refusal row can live."
  - "criterion-binding-unsatisfiable's NextCommand is the static, real command 'aether decision-answer' (not the full dynamic --question/--answer/--phase invocation, which only exists per-criterion at evaluation time and cannot live in a static registry row). The exact per-criterion command remains what ownerConfirmationCommand already builds and what skip-phase's new notice, aether continue, and the seal blocker text all point the owner at."
  - "aether plan --refresh (not the plan-text's proposed 'aether plan --revise', which does not exist as a registered command) is the directory-refusal's NextCommand -- confirmed via availableCommand and already used elsewhere in this codebase (next_action.go, plan_candidate.go) as the real command that rewrites a phase's plan."
  - "criterionBindingIsUnsatisfiable stays deliberately narrow: a requirement is unsatisfiable only when EVERY one of its bound artifacts is currently a directory. A mixed requirement (some directory artifacts, some real files) or a requirement naming a merely-missing file is never routed to the owner -- it keeps failing the ordinary way, proven by a negative case in the same test and by a mutation proof."

requirements-completed: [UED-10]

coverage:
  - id: D1
    description: "A phase binding a directory as a criterion artifact is refused by name, with the criterion and path, before any worker is dispatched -- with one command (aether plan --refresh) that gets past it. A symlink to a directory is refused the same way; a not-yet-existing path is left alone."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/criterion_binding_dead_end_test.go#TestDirectoryArtifactIsRefusedByNameAtBuildTime"
        status: pass
      - kind: unit
        ref: "cmd/criterion_binding_dead_end_test.go#TestDirectoryArtifactRefusalNamesTheWayPast"
        status: pass
    human_judgment: false
  - id: D2
    description: "The ordinary, regular-file evidence path (both the new build-time disk check and the existing run-time evaluation) is unaffected by this phase's additions."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/criterion_binding_dead_end_test.go#TestOrdinaryCriterionIsUnaffected"
        status: pass
      - kind: unit
        ref: "cmd/criterion_evidence_test.go#TestEvaluatePhaseCriterionEvidencePassesFreshBoundEvidence"
        status: pass
    human_judgment: false
  - id: D3
    description: "An already-bound, in-progress phase stuck on a directory-bound criterion is routed to the existing owner-confirmation exit (phase advances, aether seal blocks until answered through the real aether decision-answer path) instead of failing forever; a criterion that genuinely failed (a missing file) keeps failing the ordinary way and is never routed to the owner."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/criterion_binding_dead_end_test.go#TestUnsatisfiableCriterionRoutesToTheOwnerAnswer"
        status: pass
    human_judgment: false
  - id: D4
    description: "aether skip-phase --force still works and still records the phase as abandoned, but its screen now names the owner-answer route first, so abandoning a finished phase is no longer the only exit."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/criterion_binding_dead_end_test.go#TestSkipPhaseForceNamesTheTruthfulAlternative"
        status: pass
    human_judgment: false

duration: ~35min
completed: 2026-09-22
status: complete
---

# Phase 208 Plan 05: Never a Dead End -- Directory-Bound Criteria Summary

**A phase binding a whole folder as evidence is now refused by name before any work starts, and a phase already stuck on that mistake gets a real, owner-answered way forward instead of the only exit being "mark this finished phase as abandoned."**

For the owner, in plain English: before this, if a phase's success criterion pointed at a folder (like `server/app`) instead of a specific file, Aether could never check it -- it only ever checks one file at a time -- so that phase could never finish, no matter how genuinely done the work was. Now two things changed. First, Aether catches this mistake immediately, the moment you try to build that phase, and tells you exactly what to fix and the one command to run (`aether plan --refresh`) instead of letting you discover it much later. Second, for a phase that was ALREADY stuck on this before today, the only way out used to be `aether skip-phase --force`, which marks the phase as abandoned even if the work is completely finished -- a lie about your own progress. Now that phase instead asks you, in plain words, to confirm the criterion is actually true (through `aether decision-answer`), and once you do, the phase moves on honestly. `aether skip-phase --force` still exists and still works exactly as before if you genuinely want to abandon a phase, but its screen now tells you about the honest option first.

## Performance

- **Duration:** ~35 min (estimate; exact start time was not captured before the first file read)
- **Completed:** 2026-09-22T17:38:00Z (approx.)
- **Tasks:** 2
- **Files modified:** 6 (1 created, 5 modified)

## Accomplishments
- `cmd/criterion_evidence.go`: `validatePhaseCriterionEvidenceAgainstDisk` walks every bound artifact against the real filesystem and refuses (via the Phase 208-01 typed `refusal` contract) any artifact that already exists as a directory, or a symlink to one -- called from `aether build`'s two direct-dispatch call sites in `cmd/codex_build.go`, the earliest point a phase is used, so the refusal lands before any worker is spawned.
- `cmd/criterion_evidence.go` / `cmd/codex_build.go`: `attachBuildArtifactEvidence` no longer silently drops a claimed path that turned out to be a folder -- it now records the path and the reason on `codexBuildClaims.RejectedArtifacts`, so `evaluatePhaseCriterionEvidence` can say "this path is a folder, and evidence is checked file by file" instead of the misleading "artifact was not claimed by the current build" the field report named.
- `cmd/criterion_evidence.go`: `criterionBindingIsUnsatisfiable` recognises the one case Task 1 now refuses for new phases -- every one of a criterion's bound artifacts is a directory -- and `evaluatePhaseCriterionEvidence` routes exactly that case to the pre-existing `criterionStateNeedsOwnerConfirmation` state (built in Phase 193) instead of blocking forever: the phase still advances, and the criterion's `Summary` names the folder so the screen says why. A criterion that simply failed (a genuinely missing file) is untouched and keeps failing the ordinary way.
- `cmd/refusal_register.go`: two new refusal rows, `criterion-artifact-is-a-directory` (`NextCommand: aether plan --refresh`) and `criterion-binding-unsatisfiable` (`NextCommand: aether decision-answer`), inserted in the registry's required ascending-id order.
- `cmd/phase_skip.go`: `aether skip-phase --force`'s screen now opens with a notice naming the `aether decision-answer` route and stating plainly that skipping still records the phase as abandoned, before the existing force-skip confirmation. What the command actually records (report, state transition, tasks closed) is unchanged.
- `cmd/criterion_owner_confirmation.go`: doc comment on `criterionStateNeedsOwnerConfirmation` updated to name its second origin (an unsatisfiable evidence binding), so the state's meaning stays accurate for future readers.
- `cmd/criterion_binding_dead_end_test.go` (new): five tests plus two mutation proofs (see below) covering both refusal cases end to end, with real directories, real symlinks, and the real `aether decision-answer` recording path.

## Task Commits

1. **Task 1: A directory bound as an artifact is refused by name, early, with the command that gets past it** - `e70620fa` (feat)
2. **Task 2: A criterion no build could ever satisfy goes to the owner's own answer, not to a false record** - `213eddbc` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `cmd/criterion_evidence.go` - `validatePhaseCriterionEvidenceAgainstDisk`, `criterionArtifactBindingIsDirectory`, `criterionBindingIsUnsatisfiable`; `attachBuildArtifactEvidence` rejected-artifact recording; `evaluatePhaseCriterionEvidence` owner-confirmation routing for unsatisfiable bindings
- `cmd/codex_build.go` - `codexBuildClaims.RejectedArtifacts` + `codexRejectedArtifactEvidence`; the two direct-dispatch call sites now call `validatePhaseCriterionEvidenceAgainstDisk(root, phase)`
- `cmd/refusal_register.go` - two new rows: `criterion-artifact-is-a-directory`, `criterion-binding-unsatisfiable`
- `cmd/criterion_owner_confirmation.go` - doc comment on `criterionStateNeedsOwnerConfirmation` names the second origin
- `cmd/phase_skip.go` - `skipPhaseTruthfulAlternativeNotice`, wired into `renderSkipPhaseVisual` before the force-skip confirmation
- `cmd/criterion_binding_dead_end_test.go` - `TestDirectoryArtifactIsRefusedByNameAtBuildTime`, `TestDirectoryArtifactRefusalNamesTheWayPast`, `TestOrdinaryCriterionIsUnaffected`, `TestUnsatisfiableCriterionRoutesToTheOwnerAnswer`, `TestSkipPhaseForceNamesTheTruthfulAlternative`

## Decisions Made
- Used `aether plan --refresh` as the directory-refusal's `NextCommand` rather than the plan text's proposed `aether plan --revise`, which is not a registered command anywhere in this codebase. `aether plan --refresh` is the real, existing command this codebase already uses elsewhere (`next_action.go`, `plan_candidate.go`) to rewrite a phase's plan, and it resolves against the live cobra command tree (proven by `TestDirectoryArtifactRefusalNamesTheWayPast` via `availableCommand`, the same resolver the what-next card uses).
- Kept `criterionBindingIsUnsatisfiable` narrow on purpose: it returns true only when a requirement names at least one artifact AND every one of them is currently a directory. A mixed requirement, or a requirement whose artifact is simply missing (not yet built), is never routed to the owner -- proven by a negative case in `TestUnsatisfiableCriterionRoutesToTheOwnerAnswer` and by a mutation proof that widening the check to "always unsatisfiable" breaks that negative case.
- `criterion-binding-unsatisfiable`'s registered `NextCommand` is the static, generic `aether decision-answer` rather than a full dynamic invocation (the exact `--question`/`--answer`/`--phase` string only exists per-criterion at evaluation time, via `ownerConfirmationCommand`, and a static registry row cannot hold it). The exact command the owner actually needs is what `ownerConfirmationCommand`, the seal blocker text, and skip-phase's new notice all already surface.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The plan named `cmd/skip_phase_cmd.go`, which does not exist**
- **Found during:** Task 2, `read_first`
- **Issue:** Task 2's action and files list both name `cmd/skip_phase_cmd.go`. The actual `skip-phase` command lives in `cmd/phase_skip.go` (`skipPhaseCmd`, `runSkipPhase`, `renderSkipPhaseVisual`); no `skip_phase_cmd.go` file exists anywhere in the tree.
- **Fix:** Edited `cmd/phase_skip.go` instead -- the real, single implementation of `aether skip-phase`.
- **Files modified:** cmd/phase_skip.go
- **Verification:** `TestSkipPhaseForceNamesTheTruthfulAlternative` and the pre-existing `TestSkipPhaseForceAdvancesFromExecuting`/`TestSkipPhaseRequiresForce` (cmd/phase_recovery_test.go) all pass.
- **Committed in:** 213eddbc

**2. [Rule 3 - Blocking] Registering the two new refusal rows required editing `cmd/refusal_register.go`, not in either task's declared files list**
- **Found during:** Task 1 and Task 2, following the plan's explicit "Register the refusal row..." instructions
- **Issue:** Both tasks instruct registering a new row in `refusalRegistry`. That table has exactly one location (`cmd/refusal_register.go`, built in Phase 208-01) -- there is no other place a new refusal row could be added without creating a second, competing table, which Phase 208-01's own structural test (`TestFriendlyErrorsReadTheOneRefusalTable`) forbids.
- **Fix:** Added `criterion-artifact-is-a-directory` (Task 1's commit) and `criterion-binding-unsatisfiable` (Task 2's commit) to `refusalRegistry`, in the table's required ascending-id order.
- **Files modified:** cmd/refusal_register.go
- **Verification:** `TestRefusalRegisterIsSortedAndUnique`, `TestEveryRefusalRowNamesANextCommand`, `TestFriendlyErrorsReadTheOneRefusalTable` all pass.
- **Committed in:** e70620fa (row 1), 213eddbc (row 2)

---

**Total deviations:** 2 auto-fixed (both Rule 3 -- correcting an incorrect file name in the plan, and filling a files-list gap the plan's own instructions required). **Impact on plan:** Neither touched scope or architecture; both were necessary to carry out the plan's own explicit instructions correctly.

## Mutation Proofs

Both required by the plan's acceptance criteria; performed manually during verification (temporary edit -> confirm the named test fails -> revert -> confirm the test passes again). Neither mutation was committed.

**1. Task 1 -- directory check removed:** Temporarily replaced `validatePhaseCriterionEvidenceAgainstDisk`'s refusal loop body with a no-op. Result: `TestDirectoryArtifactIsRefusedByNameAtBuildTime/existing_directory_is_refused` and `.../symlink_to_a_directory_is_refused_the_same_way` both failed with "expected ... to be refused, got nil"; the not-yet-existing-path subtest correctly still passed (it asserts no refusal either way). Reverted; all three subtests pass again.

**2. Task 2 -- unsatisfiable check widened to everything:** Temporarily forced `criterionBindingIsUnsatisfiable` to unconditionally return `("MUTATION...", true)` for every requirement. Result: `TestUnsatisfiableCriterionRoutesToTheOwnerAnswer` failed -- the negative-case criterion (a genuinely missing file) was wrongly routed to `needs_owner_confirmation` instead of failing, and `outstandingOwnerConfirmations` returned both criteria instead of exactly the directory-bound one. Reverted; test passes again. This proves the narrow scope (every artifact must be a directory) is genuinely load-bearing, not decorative.

## Issues Encountered

**Pre-existing, unrelated known-red found during verification:** `TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount` fails on unrelated spawn-budget/caste-selection logic (`worker_count` vs `max_selected_castes`), explicitly listed as pre-existing in `.planning/WINDOWS.md` entry 12 (dated 2026-09-11, well before this plan). Confirmed unrelated: the failure is entirely about caste/worker-count budgeting, nothing this plan's files touch. The sibling test in the same file, `TestCodexBuildPlanOnlySpawnBudgetExplainsPrunedCastes`, passes. Not fixed here (out of scope; already tracked).

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

WINDOWS.md rows 49 and 52 are both closed: a directory bound as criterion evidence is refused by name at build time with a real way past it, and an already-stuck, in-progress phase now has a scoped, owner-answered exit instead of the abandon-only door. The refusal contract from Phase 208-01 now has two more real registered rows and a second proven call site (build-time criterion validation), ready for 208-06/07/08 to build on.

UED-10 remains unchecked in `.planning/REQUIREMENTS.md` for now -- it is a shared requirement ID declared by 208-01, 208-02, 208-05, 208-07, and 208-08, and the shared-ID gate (`requirements.ready-ids`) correctly withholds marking it complete until every plan declaring it has a SUMMARY. 208-02's summary already exists; 208-04, 208-06 (which declares a different ID, UED-11), 208-07, and 208-08 do not yet.

No blockers.

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-22*

## Self-Check: PASSED

- `cmd/criterion_binding_dead_end_test.go` verified present on disk (`[ -f ]`).
- Both task commit hashes (`e70620fa`, `213eddbc`) verified present in `git log --oneline --all`.
- Re-ran acceptance-criteria tests: `TestDirectoryArtifact*`, `TestUnsatisfiableCriterion*`, `TestOrdinaryCriterionIsUnaffected`, `TestSkipPhase*`, `TestOwnerConfirmation*`, `TestCriterionEvidence*`, `TestValidatePhaseCriterionEvidence*`, `TestRefusal*`, `TestEveryRefusalRow*`, `TestFriendlyError*` all pass.
- Re-ran plan-level `<verification>`: `go build ./cmd/aether` and `go vet ./cmd` both clean; the named `go test` verification block passes in full.
- Both mutation proofs re-confirmed (directory check removed -> target test fails; unsatisfiable check widened -> negative case fails), and both reverted before committing.
