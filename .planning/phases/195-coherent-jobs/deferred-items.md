# Deferred Items

## Full-suite black-box timeout

- **Found during:** Plan 195-01 overall verification
- **Command:** `go test ./...`
- **Observed:** 1,520 tests passed and 1 skipped before `cmd` reached Go's 10-minute package timeout. The last started `cmd` test in the JSON log was `TestCLIErrorEnvelopeExitsNonZero` in `cmd/blackbox_harness_test.go`.
- **Why deferred:** The black-box harness and CLI error-envelope path were not changed by Plan 195-01. All 47 coherent-job tests pass normally and under `-race`; `go build ./cmd/aether` and `go vet ./...` also pass.
- **Follow-up:** Investigate why the black-box harness child process does not exit in the full-suite environment before relying on `go test ./...` as a bounded gate.

## `cmd` visual-output tests assume a Codex platform environment

- **Found during:** Plan 195-03 Task 2 verification
- **Command:** `go test ./cmd -count=1`
- **Observed:** 11 visual-output tests fail under a Claude Code session
  (`TestPlanVisualOutput`, `TestBuildVisualOutputShowsSpawnPlan`,
  `TestCeremonyCloseoutBlockedPathRendersBlockedNotCompletion`,
  `TestColonizeVisualOutputShowsDispatchPreview`,
  `TestContinueBlockedVisualOutputShowsWorkerFlow`,
  `TestContinueVisualOutputShowsColonyCompleteStageMarker`,
  `TestPrintNextUpVisualOutput`,
  `TestRenderBinaryActionVisualPublishGuidanceSeparatesRepoSetupFromUpdate`,
  `TestRenderUpdateVisualShowsRemovedAssets`, `TestSetupVisualOutput`,
  `TestPauseResumePatrolPhaseAndHistoryVisualOutput`). Each asserts the raw
  `aether <verb>` form, but `translateHintCommandsForPlatform` rewrites those
  verbs to `/ant-<verb>` on every non-Codex platform. All 11 pass with
  `AETHER_PLATFORM=codex`.
- **Why deferred:** Pre-existing and unrelated to Plan 195-03 — the failures
  reproduce on the plan's own base commit and none of the listed tests touch
  coherent jobs. Confirmed by re-running the same eleven under
  `AETHER_PLATFORM=codex`, where they pass.
- **RESOLVED 2026-08-27** before Phase 195 wave 3: a `pinRawCommandNames(t)`
  helper in `cmd/codex_visuals_test.go` sets `AETHER_PLATFORM=codex` in each of
  the eleven, so they assert the raw form deterministically on any runtime. Done
  because every post-merge and regression gate for the rest of Phase 195 would
  otherwise report eleven phantom failures. See the command-naming chokepoint:
  `/ant-*` translation lives only in the visual writer.

## `cmd` package exceeds a 25-minute `go test` budget

- **Found during:** Plan 195-03 Task 2 verification
- **Command:** `go test ./cmd -count=1 -timeout 25m`
- **Observed:** `panic: test timed out after 25m0s` with `TestSurveyStaleness`
  the test in flight. `TestSurveyStaleness` passes in 1.9s in isolation, so
  this is total package runtime, not a hang.
- **Why deferred:** Extends the Plan 195-01 finding above from Go's 10-minute
  default to an explicit 25-minute budget; the cause is cumulative `cmd`
  runtime, not any single test.
- **Follow-up:** The `cmd` package needs either `t.Parallel()` coverage, a
  split, or a documented long-run gate. Until then every `cmd` gate must pass
  an explicit `-timeout` well above 25m. `workflow.test_gate_timeout` was raised
  from 1500s to 3600s on 2026-08-27 so Phase 195's post-merge and regression
  gates do not abort on a healthy suite.

## Load-sensitive test timing in `cmd` and `pkg/codex`

- **Found during:** Phase 195 wave 3 and wave 4 post-merge gates
- **Observed:** two tests fail only when the full package runs under load and
  pass on isolated reruns —
  `pkg/codex TestAvailabilityProbeRetriesOnlyTimeouts/a_transient_stall_is_retried_and_succeeds`
  (passed 3/3 alone), and
  `cmd TestCLIContinueEnforcesFreshCriterionEvidence` (passed 2/2 alone).
- **Partially resolved 2026-08-27:** the `cmd` case was traced to a hardcoded
  `--worker-timeout 1s` in `blackbox_harness_test.go` on a subtest whose build
  is expected to SUCCEED — the cap was only a backstop there, so it was widened
  to 30s. The deliberate timeout-rejection case (`"rejects timeout"` adapter
  mode in `TestCLIBuildWorkerOutcomes`) still uses 1s and is untouched.
- **RESOLVED 2026-08-27:** the `pkg/codex` probe retry test failed a third time on
  the phase's final full gate. Its budget is a floor on how long the SECOND probe
  attempt may take (shell startup plus one echo), and 200ms then 2s were both too
  tight on a loaded machine. Raised to 8s with the slow branch sleeping 120s, and
  verified with `-count=3` while eight busy loops saturated the CPU. Costs ~6s of
  runtime, because the first attempt must burn the full budget to time out.
- **Why deferred:** neither is caused by Phase 195; both are pre-existing timing
  assumptions that only surface when the machine is busy.
- **Follow-up:** audit remaining wall-clock literals in tests. A test that fails
  because the machine was busy is indistinguishable from a real regression at
  the moment it fails, which is exactly the signal these gates exist to give.

## `cmd/criterion_owner_confirmation.go` is not gofmt-clean

- **Found during:** Plan 195-08 Task 2 verification
- **Command:** `gofmt -l cmd/ pkg/`
- **Observed:** `cmd/criterion_owner_confirmation.go` is reported unformatted. `git diff HEAD -- cmd/criterion_owner_confirmation.go` is empty, so the drift predates this plan and is not caused by it.
- **Why deferred:** Out of scope boundary — the file is unrelated to coherent jobs or worktree receipts, and reformatting it here would put an unexplained hunk into a receipt-boundary commit.
- **Follow-up:** Run `gofmt -w cmd/criterion_owner_confirmation.go` in a dedicated hygiene commit.

## External completion packets cannot carry worktree-only file claims

- **Found during:** Plan 195-08 Task 2 (external-lane worktree parity)
- **Command:** `go test ./cmd -run TestGroupedWorktreePartialReceiptsExternalLaneMatchesNative`
- **Observed:** `validateCompletionPacketSemantics` -> `validateAndNormalizeClaimPathToRoot` refuses any `files_modified` claim that does not resolve inside the repository root (`claim_path.escapes_root`). A wrapper-submitted completion whose proof still lives only inside a worktree is therefore rejected before the receipt boundary is ever reached.
- **Why deferred:** That validator is a path-laundering guard. Widening it to admit paths that are not in the repository is a trust-boundary change, not a wiring change, and this plan explicitly declined to weaken a security check to make a test pass. In practice the external/wrapper lane never allocates worktrees today (only the native dispatch path does), so no shipped flow reaches this refusal.
- **Follow-up:** If a wrapper lane ever gains worktree allocation, decide deliberately whether `validateAndNormalizeClaimPathToRoot` should accept a path that resolves inside a colony-tracked worktree under `.aether/worktrees/` (still inside root, still not arbitrary) — a scoped widening, with its own adversarial tests.

## `TestAvailabilityProbeRetriesOnlyTimeouts` fails when the whole suite runs at once

- **Found during:** Plan 195-10 Task 1, `go test ./... -count=1` (the phase-wide gate)
- **Command:** `go list ./... | grep -v '/cmd$' | xargs go test -count=1`
- **Observed:** `pkg/codex` failed on the `a transient stall is retried and succeeds` subtest with `a probe that stalled once and then answered was reported as a failure: timed out`. Re-running the single test (`ok 4.542s`) and the whole package alone (`ok 22.658s`) both pass. The failure only appears when every package is compiled and run concurrently.
- **Why deferred:** Out of scope boundary — this plan modified `CLAUDE.md`, one todo file, and added `cmd/claudemd_coherent_jobs_test.go`. It touches nothing in `pkg/codex`. The fixture sets `AETHER_PROBE_TIMEOUT=2s` and its own comment concedes the wall-clock fragility: "the budget has to clear /bin/sh startup on a machine running the whole suite". Under full-suite load 2s is not enough for `/bin/sh` to reach `touch`, so the second attempt times out too.
- **Follow-up:** Same root cause as the wall-clock literals already logged above from plan 195-08. Either raise this fixture's budget, or replace the sleep-based stall with a deterministic signal (a fifo or a marker the harness controls) so the test cannot depend on machine load.

## The full and race gates exhausted the machine's disk

- **Found during:** Plan 195-10 Task 1, `go test ./... -race -count=1`
- **Command:** `go list ./... | grep -v '/cmd$' | xargs go test -race -count=1 -p 2`
- **Observed:** Six `pkg/agent/curation` tests and four packages failed with `TempDir: mkdir ...: no space left on device` / `[build failed]`. `df -h /` reported 571Mi available on a 1.8Ti volume (100% used). No test asserted anything false; the race-instrumented build had nowhere to write.
- **Why deferred:** Environmental, not a code defect. `go clean -cache` (7.2G of regenerable build cache) restored 7.8Gi free and the identical command then passed every package, as did `go test ./cmd -race -count=1` (`ok 826.019s`).
- **Follow-up:** None for this repo. Noted so that a future `no space left on device` in a gate log is recognised as a disk condition rather than investigated as a regression.

## Code review Info findings IN-01..IN-05 (195-REVIEW.md)

Recorded here because the iteration-2 re-review found no deferral record for
any of them anywhere (IN-11, 195-REVIEW.iter2.md), and this repo's own history
says an undocumented deferral becomes invisible debt.

### IN-01 — the job planner mutated its caller's task list — RESOLVED 2026-08-27

- **File:** `cmd/coherent_jobs.go`, `normalizeCoherentJobProposal`
- **Was:** the proposal arrived by value, but a slice header copy still shares
  its backing array, so trimming task IDs in place rewrote the caller's own
  data — contradicting `planCoherentJobs`' documented purity.
- **Fixed in the second fix round:** the task list is copied before trimming.
  Locked by `TestJobPlannerDoesNotMutateItsCallersProposal`.

### IN-02 — an owner-facing sentence was capitalized by byte, not character — RESOLVED 2026-08-27

- **File:** `cmd/ceremony_team_checkin.go`, the team check-in summary
- **Was:** `strings.ToUpper(s[:1]) + s[1:]` splits any character that takes more
  than one byte to store, printing rubbish. Every current sentence is plain
  ASCII, so nothing was visibly broken yet.
- **Fixed in the second fix round:** extracted as `sentenceCase`, which decodes
  a whole character. Locked by `TestSentenceCaseKeepsWholeCharacters`.

### IN-03 — two workers with no task id compared equal — RESOLVED 2026-08-27

- **File:** `cmd/codex_build_worktree.go`, `worktreeOwnershipIdentity`
- **Was:** ownership was keyed on the task id alone, so two different workers
  that both arrived without one were treated as the same owner and a genuine
  collision between them over one file was not refused. No dispatch shape in
  the runtime produces that pair today.
- **Fixed in the second fix round:** the key falls back to the worker's name.
  Locked by `TestTwoUnnamedWorkersStillConflictOverOneFile`.

### IN-04 — recovery-job reuse is approximate — STILL OPEN

- **File:** `cmd/coherent_job_retry.go`, `findExistingBuildAttemptRetry` and
  `commitPartialBuildRetryPlan`
- **Issue:** two separate approximations.
  1. `findExistingBuildAttemptRetry` returns *any* recovery record linked to
     the parent attempt, without comparing its `SelectedTasks` against the
     unfinished set computed right now. If a second partial finalize for the
     same parent produces a *different* unfinished set (tasks credited in
     between, or a re-run that proved more), the stale record is reused and its
     `SelectedTasks` no longer describe the work that remains. Only
     `RedispatchCommand` — recomputed each time from the live plan — is
     currently correct in that situation, which is why nothing user-facing has
     broken yet.
  2. `beginChildBuildAttempt(..., plan.Jobs[0].Name, ...)` records only the
     FIRST of N recovery jobs as `ParentJobName`. When several dispatches
     failed part-way, the other jobs' names are lost from the journal.
- **Why deferred:** neither is reachable by a shipped flow today (nothing reads
  `ParentJobName`; the only consumer of the recovery record is the id/path pair
  reported back to the owner), and fixing (1) properly means deciding what
  should happen to a superseded recovery record — reuse it, supersede it, or
  refuse — which is a behaviour decision, not a wiring fix.
- **Follow-up:** compare the existing child's `SelectedTasks` with the freshly
  computed unfinished set before reusing it, and decide the supersede rule
  deliberately; join all recovery job names for `ParentJobName`. Lock with a
  test that finalizes two different partials against one parent.

### IN-05 — three finalize helpers each recompute the same two sets — STILL OPEN

- **File:** `cmd/codex_build_finalize.go`, `allSelectedBuildTasksCredited`,
  `unfinishedBuildTaskIDs`, `creditedBuildTaskIDs`
- **Issue:** all three call `buildFullBuildTaskIDSet` + `completedBuildTaskIDs`
  independently on the same inputs, on every finalize. Correct, but the
  partition is derived three times and could drift if one of the three is ever
  changed alone.
- **Why deferred:** pure tidiness with no behavioural symptom; the change
  touches the commit path of every build finalize, which is not a place to make
  a cosmetic edit inside a fix round aimed at trust-boundary defects.
- **Follow-up:** compute the credited/unfinished partition once in
  `runCodexBuildFinalize` and pass it down to all three.

## Open items carried out of the third review (195-REVIEW.iter3.md)

The closure review ended `status: clean` / shippable with four items it did not
fix. Recorded here because "knowingly left" only counts when it is written in the
record of what was knowingly left — the third review said outright that WR-15
belonged in this file, and it was not added until the phase verifier caught the
omission.

### WR-15 — what a "nothing needed changing" report may claim (OWNER DECISION)

- **Severity:** WARNING — a residual design point, not a defect in the fix.
- **Where:** `cmd/coherent_job_receipts.go` (the no-change branch of the two-stage
  receipt boundary).
- **What changed and why it matters:** before the third fix pass, a
  `completed_no_change` receipt claiming no path was refused outright whenever its
  task declared any path, because stage 1's declared-file binding found no match.
  It is now admitted, and credited when every file the task declares exists in the
  checkout. Declared paths come from the task's evidence artifacts and file-shaped
  hints — which, for a task that modifies existing code, already exist before the
  build starts. So the credited surface moved: a hole reachable only by path-less
  tasks was closed, and a narrower but more common one opened — "the file it names
  is there, so I say the work was already done".
- **Why it was judged shippable anyway,** three bounds all re-traced by the phase
  verifier: a build made only of no-change credit still fails the phantom-build
  guard; no-change credit carries no artifacts, so it cannot satisfy a criterion
  that demands one; and the receipt must still pass the scope, status, summary and
  handoff checks.
- **The actual question for the owner:** what must a worker show before "I checked
  and nothing needed changing" is accepted? Options are (a) leave as is, (b) require
  the named command to be re-run by the program rather than reported, which this
  repo already does for builder-reported evidence elsewhere (Phase 193, D-04), or
  (c) refuse no-change credit from a FAILED worker entirely and let the retry
  credit it through the ordinary whole-success path. This is a rule about
  user-facing behaviour, so it is the owner's call, not a reviewer's.

### IN-13 — the replay's existing-record branch has no assertion

- Cosmetic. `idempotentExternalPartialFinalizeResult`'s branch for an
  already-present recovery record is unasserted, so a future change could alter it
  without any test noticing.

### IN-14 — the ownership-key fallback collapses on two fully-anonymous dispatches

- The worktree ownership key falls back through task ID then worker name; two
  dispatches with BOTH empty would still collide. No runtime shape produces that
  today, which is why it is Info rather than a defect.

### IN-16 — worktree refusal messages went to stderr, not the owner-facing stream

- Carried from the second review as IN-12 and still unaddressed. Cosmetic, still
  visible to the owner, and no finding asked for the move in the first place.

### Also closed while recording these

`coalesceSequentialDispatches` and its only helper `dispatchesFormOneJob` were
deleted (92 lines, zero callers anywhere — production or test), and the eight
comments across `cmd/` that still cited the coalescer as the live grouping
mechanism now name `planCoherentJobs`. The phase verifier flagged this: superseded
machinery left in place with live-sounding comments is exactly how a later reader
concludes the wrong function is authoritative, which is this repo's documented
signature failure.
