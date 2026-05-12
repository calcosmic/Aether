# Codex Lifecycle Activity Verification

Updated: 2026-05-12

Phase 6 checks that Codex lifecycle commands do not silently slide back into
fake, background-only, or untracked worker activity.

For dummies: when Codex runs the Aether lifecycle, the user should see real
worker panels, Aether should log those workers, and only the runtime finalizer
should write the official colony state.

## Verified Contract

| Command | Required activity evidence | Runtime authority |
|---|---|---|
| `colonize` | `colonize --plan-only`, visible Surveyor workers, `spawn-log`, `spawn-complete`, `ceremony worker-complete` | `colonize-finalize` writes survey state |
| `plan` | `plan --plan-only`, visible Scout and Route-Setter workers, `spawn-log`, `spawn-complete`, `ceremony worker-complete` | `plan-finalize` writes planning state |
| `build` | `build <phase> --plan-only`, visible manifest workers, `spawn-log`, `spawn-complete`, `ceremony worker-complete` | `build-finalize` writes build state and claims |
| `continue` | default `continue --skip-watchers --verification-depth standard`; heavy review uses visible reviewers with `spawn-log`, `spawn-complete`, `ceremony worker-complete` | default runtime continue advances state; heavy review uses `continue-finalize` |

## Regression Tests

The focused phase-6 tests are:

```bash
go test ./cmd -run 'TestCodexLifecycleGuidesRequireVisibleWorkerActivity|TestCodexLifecycleYamlAndGuidesAgreeOnWorkerActivity|TestCodexLifecycleSkillMirrorsWorkerActivityContract'
go test ./cmd -run 'TestContinuePlanOnlyRequiresFinalizerAndDoesNotWriteReview|TestContinuePlanOnlySkipWatchersLightEmitsNoWorkerDispatches'
go test ./cmd -run 'TestValidateBuildProvenance'
go test ./cmd -run 'TestClaimsOrAggregate(RejectsUnsafeClaimPaths|RejectsAbsoluteClaimPaths|RejectsAetherDataClaimPaths|RejectsMissingClaimPaths|RejectsAmbiguousClaimPaths|RejectsSymlinkClaimPaths|NormalizesValidRepoRelativeClaimPaths|WithAntName)$'
```

Those tests compare `cmd/command_guide.go`, `.aether/commands/*.yaml`, and the
shipped Codex lifecycle skill so plan/build/continue guidance cannot drift apart
without failing CI.

Existing command tests also prove runtime recording paths:

- `TestColonizeWritesSurveyArtifactsAndUpdatesState`
- `TestColonizeFinalizeRecordsExternalSurveyors`
- `TestPlanUsesSurveyAndRecordsPlanningDispatches`
- `TestPlanFinalizeRecordsExternalPlanningAndWritesState`
- `TestBuildWritesDispatchArtifactsAndUpdatesState`
- `TestBuildFinalizeRecordsExternalTaskResultsForContinue`
- `TestContinueFinalizeRecordsExternalReviewAndAdvances`
- `TestContinueRecordsWorkerFlowInStateReportAndSpawnSummary`

## Phase 6 Live Loop

The current `aether build 6` run is itself a host-orchestrated lifecycle pass:
the runtime emitted a phase-6 dispatch manifest, the Queen rendered the spawn
plan, and the host is processing the planned workers through the same
`spawn-log` -> visible worker -> `spawn-complete` -> `worker-complete` contract.

The phase-6 manifest contains these workers:

- `Sage-45` for phase research.
- `Gate-48` for security boundary review.
- `Find-42` for root-cause regression context.
- `Hammer-9` for lifecycle worker activity tests.
- `Brick-27` for live-loop comparison evidence.
- `Excavat-48`, `Exam-92`, and `Guard-98` for probe, audit, and verification.

This does not replace separate Claude Code or OpenCode UI smoke testing, but it
does make the Codex contract testable: if command-guide, wrapper YAML, or the
shipped skill stop requiring visible logged workers, tests fail before release.

## Security Boundary Closed

Gatekeeper found one lifecycle honesty blocker during this phase: build worker
claim validation accepted absolute in-repo paths and `.aether/data` paths after
normalization. That blurred the boundary between untrusted worker output and
runtime-owned state.

The fix is in `cmd/codex_build_finalize.go`: build-finalize now rejects absolute
paths and `.aether/data` claim paths before it persists `last-build-claims.json`
or worker handoffs. Build provenance also normalizes completion aliases such as
`code_written` and requires implementation-worker file evidence, so watcher-only
file claims cannot satisfy build provenance. The focused claim and provenance
tests listed above cover those boundaries.

## Continue Contract Drift Closed

Tracker found one remaining stale string: the continue plan-only
`wrapper_contract.source_command` still advertised the old light/skip-watchers
path even when the wrapper contract and command guide used heavy external
review. The source command is now generated from the selected review depth, so
heavy review emits the heavy command and explicit light skip-watcher mode emits
the light skip-watcher command.
