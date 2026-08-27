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
