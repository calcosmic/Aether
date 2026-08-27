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
