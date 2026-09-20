---
phase: 195-coherent-jobs
verified: 2026-08-27T12:18:16Z
verified_at_commit: 18492b69
status: passed
score: 4/4 success criteria verified
behavior_unverified: 0
overrides_applied: 0
requirements:
  - id: JOBS-01
    status: satisfied
    evidence: "cmd/coherent_jobs.go validateCoherentJobProposal; TestCoherentJobProposalOrderRefusedByName (mutation-proven); TestBuildJobProposalRoundTrip at CLI level"
  - id: JOBS-02
    status: satisfied
    evidence: "TestMergedDispatchCreditsEveryCoveredTask (mutation-proven, non-vacuous by explicit guard); TestGroupedJobBriefCarriesEveryTaskContract (mutation-proven)"
  - id: JOBS-03
    status: satisfied
    evidence: "TestCalVaultSixBatchesPlanAsOneCoherentJob + TestCalVaultSixBatchesBecomeOneInRepoJob (both fail under mutation A)"
  - id: JOBS-04
    status: satisfied
    evidence: "TestCalVaultSixBatchesBecomeOneWorktreeJob — real runCodexBuild against a real git repo, 1 worktree allocated/merged/cleaned, 6 files reach root, 6 tasks credited"
human_verification:
  - test: "Decide what a 'no change needed' report is allowed to claim (review finding WR-15)."
    expected: "Owner rules on whether a worker may be credited for a task purely because the file that task names already exists in the project, with no edit and no test."
    why_human: "This is a user-facing behaviour rule about when work counts as done, not a wiring defect. The review closed the defect it came from and flagged this as a deliberate design point needing an owner call. cmd/coherent_job_receipts.go:349-399."
    status: resolved
    resolved: 2026-08-27
    resolution: "Owner ruled: the program re-runs the check itself. A completed_no_change receipt is credited only when the runtime re-executes the command the worker named and sees it pass -- reusing Phase 193's allowlisted, argv-only, never-sh -c runner. Implemented in cmd/coherent_job_no_change_recheck.go (commits 4687f514 RED, 60c6ffbc GREEN); ruling recorded in deferred-items.md. Both directions locked: TestNoChangeReceiptCreditedWhenItsNamedCheckPassesOnReRun and TestNoChangeReceiptRefusedWhenItsNamedCheckFailsOnReRun, plus TestNoChangeReceiptCheckIsNeverHandedToAShell (sentinel-file proof no command reached a shell)."
warnings:
  - id: WARN-1
    status: resolved
    resolved: "2026-08-27, commit 803ae358 -- coalesceSequentialDispatches and its only helper dispatchesFormOneJob deleted (92 lines, zero callers); all eight comments citing the coalescer now name planCoherentJobs. No stale reference remains."
    statement: "coalesceSequentialDispatches (cmd/codex_build.go:1281, 62 lines) is orphaned — defined, never called, in production or tests. Six comments across cmd/ still describe it as the live grouping mechanism."
    impact: "No runtime impact; grouping runs through planCoherentJobs. Risk is navigational — a future reader or auditor grepping for the grouping mechanism lands on dead code that says it is the answer."
  - id: WARN-2
    status: resolved
    resolved: "2026-08-27, commit 803ae358 -- WR-15, IN-13, IN-14 and IN-16 recorded in deferred-items.md with enough detail to act on."
    statement: "195-REVIEW.iter3.md's four residual findings (WR-15, IN-13, IN-14, IN-15) were not recorded in deferred-items.md, despite iter3 explicitly instructing that WR-15 'belongs in deferred-items.md alongside IN-04' if not taken now. deferred-items.md records only IN-01..IN-05 from the first review."
    impact: "Bookkeeping. Four knowingly-left items have no entry in the phase's own record of what was knowingly left."
gaps: []
deferred: []
---

# Phase 195: Coherent Jobs — Verification Report

**Phase Goal:** The Queen can bundle several related tasks into one job for one
worker, instead of sending a separate worker per task. Related tasks (same files,
or one depends on another) get grouped by default even with no explicit
instruction — so, for example, six near-identical file-copy steps become one job
instead of six.

**Verified:** 2026-08-27T12:18:16Z at commit `18492b69` (branch `oracle-reinstate`)
**Status:** passed — **4/4 success criteria verified.** The phase goal is achieved.

*Originally filed `human_needed` on 2026-08-27 for one outstanding owner behaviour
decision (WR-15) and two bookkeeping warnings. All three were resolved the same day
— the owner ruled, the ruling was implemented and locked in both directions, the
orphaned coalescer was deleted, and the four residual review items were recorded.
Status raised to `passed` on that basis; see the resolution notes in the frontmatter
above. No success criterion was re-scored: all four were already verified, each by
running its command and by breaking the production code and watching it fail.*
**Method:** goal-backward, adversarial. Every criterion was proven by running its
command AND by breaking the production code and confirming the command fails.

---

## Verdict Table

| # | Success criterion | Verdict | Command that proves it |
|---|---|---|---|
| 1 | Queen can combine tasks into one job with a stated reason; an unsafe dependency order is refused by name | ✓ VERIFIED | `go test ./cmd/ -run 'TestCoherentJobProposalAccepted\|TestCoherentJobProposalOrderRefusedByName\|TestCoherentJobRepairKeepsSafeProposals\|TestCoherentJobCrossCasteNeedsOwnerReason\|TestCoherentJobGraphErrorsHaveNoPlan' -v -timeout 180s` → all PASS; plus `-run 'TestBuildJobProposalRoundTrip\|TestBuildRejectsInvalidJobProposalBeforeAttempt'` → PASS |
| 2 | A combined job's instructions carry every task's requirements, and finishing it credits every covered task (`TestMergedDispatchCreditsEveryCoveredTask`) | ✓ VERIFIED | `go test ./cmd/ -run 'TestMergedDispatchCreditsEveryCoveredTask\|TestGroupedJobBriefCarriesEveryTaskContract\|TestFailedGroupedDispatchCreditsExactlyReceiptedTasks\|TestFailedGroupedDispatchWithoutReceiptsCreditsNone' -v -timeout 300s` → all PASS |
| 3 | The real six-batch CalVault fixture, with no proposal, yields one worker not six | ✓ VERIFIED | `go test ./cmd/ -run 'TestCalVaultSixBatchesPlanAsOneCoherentJob\|TestCalVaultSixBatchesBecomeOneInRepoJob' -v -timeout 300s` → both PASS |
| 4 | The same grouping works in worktree mode — no longer switched off there | ✓ VERIFIED | `go test ./cmd/ -run 'TestCalVaultSixBatchesBecomeOneWorktreeJob\|TestCoherentJobsMatchAcrossParallelModes\|TestGroupedWorktreeOwnsUnionedPaths\|TestDistinctWorktreeJobsStillRejectOverlap' -v -timeout 300s` → all PASS |

**Score: 4/4.**

---

## Definition of Done — discrimination proof

`CLAUDE.md`'s binding rule is that a criterion is satisfied only when a command
exists **that fails when the requirement is unmet**. Passing tests alone do not
meet that bar, so each criterion's named command was run against a deliberately
broken build. All source files were restored from byte-for-byte backups
afterwards; `git status` and `md5` confirm the working tree is unchanged.

| Mutation | What was broken | Result |
|---|---|---|
| **A** — `coherentJobComponentSet.unionIfDependencySafe` returns `false` unconditionally (all automatic grouping off) | criteria 2, 3, 4 | `TestCalVaultSixBatchesPlanAsOneCoherentJob` FAIL · `TestCalVaultSixBatchesBecomeOneInRepoJob` FAIL · `TestCalVaultSixBatchesBecomeOneWorktreeJob` FAIL · `TestCoherentJobsGroupMeaningfulPathsAndDependencies` FAIL (2 subtests) · `TestGroupedJobBriefCarriesEveryTaskContract` FAIL · **`TestMergedDispatchCreditsEveryCoveredTask` FAIL** |
| **B** — `validateCoherentJobProposal`'s dependency-order loop iterates nothing (never refuses an unsafe order) | criterion 1 | `TestCoherentJobProposalOrderRefusedByName` FAIL — `unsafe proposal status = "accepted", want refused` · `TestCoherentJobRepairKeepsSafeProposals` FAIL |
| **C** — `dispatchCoveredTaskIDs` returns only `CoveredTaskIDs[:1]` (the exact downstream field-report bug: "did all six, recorded only job #1") | criterion 2 | **`TestMergedDispatchCreditsEveryCoveredTask` FAIL** · `TestGroupedJobCreditsEveryCoveredTaskDuringContinue` FAIL |
| **D** — `renderGroupedDispatchTaskContracts` drops all but the first covered task | criterion 2 (requirements carried) | `TestGroupedJobBriefCarriesEveryTaskContract` FAIL — `grouped brief missing "beta-evidence"`, `missing "go test ./cmd -run Beta"` |
| **E** — `task_receipts` renamed throughout the checked-in completion schema | JOBS-02 wire contract | `TestCompletionPacketSchemaMatchesStructs` FAIL |

### `TestMergedDispatchCreditsEveryCoveredTask` — the criterion-2 named test

The brief asked specifically whether this test still exists, still runs, and still
discriminates. All three: `cmd/merged_dispatch_task_credit_test.go:40`, runs in
0.03s, and **fails under two independent mutations (A and C)** — one that removes
the grouping it depends on, one that removes the crediting it asserts.

It cannot pass vacuously. It builds its own anti-vacuity guard:

```go
if chain.Name == "" {
    t.Fatalf("fixture produced no merged dispatch, so it cannot exercise covered-task crediting at all; dispatches: %+v", manifest.Dispatches)
}
```

That guard is what fires under mutation A. And it is not a unit stub: it runs
`runCodexBuildPlanOnly` against a real colony state, pulls the real
`dispatch_manifest`, feeds real completion packets through
`mergeExternalBuildResults`, then `reconcileCompletedBuildTasks`, and asserts
every task in reloaded colony state reached `completed`.

---

## Required Artifacts

| Artifact | Lines | Exists | Substantive | Wired | Status |
|---|---|---|---|---|---|
| `cmd/coherent_jobs.go` | 945 | ✓ | ✓ | ✓ `codex_build.go:1447` | ✓ VERIFIED |
| `cmd/coherent_jobs_test.go` | 621 | ✓ | ✓ 10 tests | ✓ | ✓ VERIFIED |
| `pkg/codex/task_receipt.go` | 56 | ✓ | ✓ | ✓ `pkg/codex/worker.go` | ✓ VERIFIED |
| `.aether/schemas/completion-packet.schema.json` | 1199 | ✓ | ✓ | ✓ generated-from-Go, mutation E | ✓ VERIFIED |
| `cmd/dispatch_coalesce_test.go` | 414 | ✓ | ✓ 12 tests | ✓ | ✓ VERIFIED |
| `cmd/coherent_job_receipts.go` | 686 | ✓ | ✓ | ✓ native + external + worktree lanes | ✓ VERIFIED |
| `cmd/merged_dispatch_task_credit_test.go` | 392 | ✓ | ✓ 3 tests | ✓ | ✓ VERIFIED |
| `cmd/ceremony_team_checkin.go` | 587 | ✓ | ✓ | ✓ `codex_workflow_cmds.go` | ✓ VERIFIED |
| `cmd/coherent_job_retry.go` | 347 | ✓ | ✓ | ✓ calls `planCoherentJobs` | ✓ VERIFIED |
| `cmd/codex_build_worktree.go` | 1559 | ✓ | ✓ | ✓ | ✓ VERIFIED |
| `.claude/commands/ant/build.md` + 2 mirrors | — | ✓ | ✓ | ✓ byte-identical, md5 `b96aea89…` ×3 | ✓ VERIFIED |
| `.aether/commands/build.yaml` | — | ✓ | ✓ `task_receipts` ×2, `job_proposal` ×1 | ✓ | ✓ VERIFIED |
| `CLAUDE.md` §"Coherent Jobs and Completion Evidence (2026-08-27)" | — | ✓ | ✓ | ✓ cites live tests | ✓ VERIFIED |
| `.planning/todos/completed/2026-08-23-one-worker-…md` | — | ✓ | ✓ names `TestOneWorkerBuildSkipsCheckin` | ✓ moved out of `pending/` | ✓ VERIFIED |

---

## Key Link Verification

| From | To | Via | Status |
|---|---|---|---|
| `cmd/codex_build.go:1447` | `cmd/coherent_jobs.go` | `planCoherentJobs`, called inside `plannedBuildDispatchesWithJobProposals` — the **single** production bridge, reached by all three build entry points (`:385`, `:669`, `:1383`) | ✓ WIRED |
| `cmd/codex_workflow_cmds.go:1474` | `cmd/codex_build.go` | `--job-proposal` StringArray → `parseCoherentJobProposals` → `JobProposals` in `codexBuildOptions`. Confirmed live on the real CLI: `aether build --help` lists the flag | ✓ WIRED |
| grouping | worktree ownership | `planCoherentJobs` runs at `codex_build.go:1447`; `effectiveParallelMode` is first read at `:452`. Grouping is **upstream of** the mode branch — there is no code path where worktree mode can switch it off | ✓ WIRED (criterion 4's mechanism) |
| `cmd/codex_build_finalize.go` | `cmd/coherent_job_receipts.go` | `admitCoherentJobTaskReceipts` → `finalizeCoherentJobTaskReceiptEvidence` (two-stage, root-backed) | ✓ WIRED |
| `cmd/coherent_job_retry.go:104` | `cmd/coherent_jobs.go` | recovery re-plans through the same `planCoherentJobs` | ✓ WIRED |
| manifest | owner | `JobDecisions` written on all four manifest lanes (`:462`, `:812`, `:948`, `:2675`); wrapper §"Coherent Jobs" step 3 instructs relaying `offending_task_id` / `dependency_id` in plain English | ✓ WIRED |

---

## Behavioural Runs (real output)

```
$ go build ./...            → exit 0
$ go vet ./cmd/...          → exit 0
$ gofmt -l cmd/ pkg/        → (no output — clean)

$ go test ./cmd/ -run 'Coherent|CalVault|Grouped|CoveredTask|TaskReceipt|Checkin|JobProposal|MergedDispatch|Worktree' -v -timeout 900s
ok  github.com/calcosmic/Aether/cmd  24.984s
183 PASS, 0 FAIL, 1 SKIP

$ go test ./pkg/codex/ -run 'TaskReceipt|Receipt' -v -timeout 300s
--- PASS: TestTaskReceiptContractRoundTripPreservesTaskEvidence
--- PASS: TestWorkerClaimsTaskReceiptsPreserveTaskSpecificEvidence
--- PASS: TestWorkerClaimsWithoutTaskReceiptsIsBackwardCompatible
--- PASS: TestTaskReceiptContractWorkerSchemaIsStrictAndAdditive
ok  github.com/calcosmic/Aether/pkg/codex  0.323s

$ go test ./cmd/ -run 'TestContractSchema|TestParity|TestLifecycleWrapper|TestCLIFlagAudit|TestCommandGuide|TestClaudemdCoherentJobs|TestCLAUDEMD' -v -timeout 500s
ok  github.com/calcosmic/Aether/cmd  → 26 PASS, 0 FAIL

$ aether build --help | grep job-proposal
      --job-proposal stringArray   Queen coherent-job proposal as one JSON object
                                   (repeatable; fields: name, task_ids, owner_caste,
                                   relationship, benefit, owner_reason)
```

The one SKIP is `TestWorktreeAllocateNoStore` — a pre-existing defensive nil-guard
test that skips because `PersistentPreRunE` always initialises the store. Unrelated
to Phase 195 and unchanged by it.

Per the brief's constraint, the full suite and `-race` were not re-run; they were
run green during the phase and re-running costs ~10 minutes with no new evidence.

---

## Criterion detail

### Criterion 1 — grouping with a reason, unsafe order refused by name

`validateCoherentJobProposal` (`cmd/coherent_jobs.go:245`) refuses a proposal
missing `relationship` or `benefit` before anything else, so a job cannot exist
without a stated reason (D-04). Two distinct unsafe-order refusals are
implemented, both naming the offender:

- a member appearing before an in-job dependency → *"task X appears before its
  unmet dependency Y"*, with `OffendingTaskID` and `DependencyID` set
- a member depending on an external task that itself depends back into the
  proposal → named the same way

D-06 is real, not theoretical: `TestCoherentJobRepairKeepsSafeProposals` proves
one refusal leaves unrelated safe jobs intact and repairs only the bad group
(`ReplacementJobNames` populated). D-07 is real: `TestCoherentJobGraphErrorsHaveNoPlan`
proves a cycle or missing dependency returns an error with **no plan at all**, and
`TestBuildRejectsInvalidJobProposalBeforeAttempt` proves at the CLI level that no
`manifest.json` and no `latest-attempt.json` are written before the rejection.

### Criterion 2 — instructions carry every task, completion credits every task

Two halves, each independently mutation-proven. The brief half:
`renderGroupedDispatchTaskContracts` emits a `## Covered Task Contracts` section
carrying each covered task's goal, constraints, hints, criteria and evidence
requirements — and the test additionally asserts each relevant path appears
**exactly once**, so the fix cannot be "paste everything twice". The credit half:
`dispatchCoveredTaskIDs` expands the covered chain, and `TestMergedDispatchCreditsEveryCoveredTask`
runs the whole real path from plan-only manifest to reconciled colony state.

Partial credit (D-08/D-09) is genuinely discriminating in both directions:
`TestFailedGroupedDispatchCreditsExactlyReceiptedTasks` (four of six credited) and
`TestFailedGroupedDispatchWithoutReceiptsCreditsNone` (no receipts, none credited)
both pass, and receipts are root-backed — the claimed file must actually exist in
the checkout for `finalizeCoherentJobTaskReceiptEvidence` to credit it.

### Criterion 3 — CalVault six batches → one worker, no proposal

Verified at two levels. Planner level (`TestCalVaultSixBatchesPlanAsOneCoherentJob`):
six chained batch tasks sharing `docs/recovery-matrix.md`, no proposal passed
(`nil`), one job out, `Source == automatic`, six task IDs in plan order. Dispatch
level (`TestCalVaultSixBatchesBecomeOneInRepoJob`): the same six through the real
`plannedBuildDispatchesForSelection` produce exactly one wave dispatch,
caste `builder`, `CoveredTaskIDs == [t1..t6]`. Its failure message is written for
a human: *"six dependent copy steps became N workers, want 1; each extra worker
pays full startup cost and re-reads the same list."*

Over-grouping is guarded the other way by `TestCoherentJobsIgnoreIncidentalPaths`
(a shared README/changelog/dependency-manifest never joins unrelated work) and
`TestCoherentJobsRespectCasteBoundaries`.

### Criterion 4 — same behaviour in worktree mode

The strongest evidence in the phase. `TestCalVaultSixBatchesBecomeOneWorktreeJob`
runs the full `runCodexBuild` with `ParallelMode: worktree` against a real git
repo and asserts, in order: exactly **1** builder checkout used, exactly **1** wave
dispatch in the persisted manifest covering all six IDs with a non-empty
`JobReason`, exactly **1** worktree in colony state with status `merged`, that
worktree's directory **removed** after merge, all six batch files present in the
**root** checkout, and all six tasks `completed`. It fails under mutation A.

Structurally, grouping cannot be switched off in worktree mode because it happens
before the mode is ever read — see the key-link table. `TestCoherentJobsMatchAcrossParallelModes`
asserts the two modes produce the identical canonical plan, and
`TestDistinctWorktreeJobsStillRejectOverlap` proves the change did not weaken the
overlap guard: two genuinely unrelated tasks sharing one incidental file are still
refused before any worker runs.

---

## Requirements Coverage

All four IDs declared in PLAN frontmatter are traced to REQUIREMENTS.md and
implemented. No orphaned requirement: REQUIREMENTS.md maps exactly JOBS-01..04 to
Phase 195 (lines 105–108), and all four appear across the ten plans' `requirements`
fields. No plan claims an ID absent from REQUIREMENTS.md.

---

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `cmd/codex_build.go` | 1281 | `coalesceSequentialDispatches` — orphan (defined, zero callers in production *and* tests) | ⚠️ WARNING | Dead predecessor of the grouping mechanism, 62 lines. Six comments in `cmd/` still cite it as the live path (`codex_build.go:52`, `codex_build_finalize.go:81` and `:1606`, plus 3 test comments). No runtime effect. |

Debt-marker scan (`TODO`/`FIXME`/`XXX`/`TBD`/`HACK`/`PLACEHOLDER`/"not yet
implemented") across all 13 production files this phase touched: **zero hits.**
`gofmt -l cmd/ pkg/`: clean.

---

## Human Verification Required

### 1. What may a "no change needed" report claim? (review finding WR-15)

**Test:** Decide the rule. Today, if a worker reports "this task needed no change"
and the task named a file, the worker is credited for that task as long as the
file simply exists in the project — and for most tasks that edit existing code,
that file already existed before the build started.
**Expected:** An owner ruling on whether that counts as done, or whether proof of
work should be required.
**Why human:** This is a rule about when work counts as finished — user-facing
behaviour, the owner's call under CLAUDE.md's decision boundaries, not a wiring
defect. The review closed the defect this came from and judged the residual
shippable; it explicitly asked for an owner decision. Three real bounds limit the
blast radius, all traced by the reviewer and re-read here: a build made only of
no-change credit still cannot finalise (`receiptEvidenceCount == 0` → SAFE-01
rejects it); no-change credit carries no artifacts so it cannot satisfy a phase
criterion that demands artifacts; and the receipt must still clear scope, status,
summary and handoff validation. Source: `cmd/coherent_job_receipts.go:349-399`.

---

## Warnings (no owner action required)

**WARN-1 — orphaned `coalesceSequentialDispatches`.** The old grouping function
survives with no caller anywhere, while six comments across `cmd/` still describe
it as the mechanism. This repo's own history is the reason to say so: superseded
machinery that is never deleted is how a later reader concludes the wrong function
is live. Deleting it and repointing the six comments at `planCoherentJobs` is a
few minutes of work and changes no behaviour.

**WARN-2 — iter3's residuals were never recorded as deferrals.** `195-REVIEW.iter3.md`
ends with four open items (WR-15, IN-13, IN-14, IN-15) and states outright that
WR-15 *"belongs in `deferred-items.md` alongside IN-04"* if not fixed. It was not
fixed and it is not there — `deferred-items.md` records only IN-01..IN-05 from the
first review. Four knowingly-left items have no entry in the phase's own record of
what was knowingly left. That is precisely the drift `.planning/v1.14-MILESTONE-AUDIT.md`
documents, in miniature.

---

## Gaps Summary

**None.** All four success criteria are met, each proven by a command that was
demonstrated to fail when the behaviour is removed. Every must-have artifact
across the ten plans exists, is substantive, and is wired into a live call path —
including the two that a stub could most easily have faked: the `--job-proposal`
flag (confirmed on the real CLI, not just in source) and the worktree lane
(confirmed by a real end-to-end build against a real git repo, not a unit stub).

The phase goal is achieved. Proceed. The one human item is an owner policy
decision that was already surfaced by the phase's own review, and the two warnings
are tidy-up, not defects.

---

_Verified: 2026-08-27T12:18:16Z_
_Verifier: Claude (gsd-verifier), goal-backward with mutation-based discrimination_
_All source mutations reverted; working tree confirmed byte-identical to `18492b69`_
