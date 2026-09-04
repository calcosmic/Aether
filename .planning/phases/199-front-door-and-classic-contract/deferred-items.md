# Deferred Items

## `TestBranchDispositionRecordsAllThreeBranches` expects a pre-archive path

- **Found during:** Plan 199-01 overall verification (`go test ./...`)
- **Observed:** The suite reported 8,567 passing tests, one failure, and 11 skips because the test expects `.planning/phases/196-see-what-it-cost/196-BRANCH-DISPOSITION.md`.
- **Current artifact:** `.planning/milestones/v1.27-phases/196-see-what-it-cost/196-BRANCH-DISPOSITION.md`
- **Isolation check:** The failing test reproduces on its own and does not exercise the cleanup ownership changes from this plan.
- **Why deferred:** This is pre-existing archive-path maintenance outside Plan 199-01's test-cleanup scope.
- **Follow-up:** Make the branch-disposition test archive-aware, or update its fixture path in a dedicated maintenance change.

## `TestColonyPrimeMdDeletionProducesByteIdenticalOutput` is suite-order sensitive

- **Found during:** Plan 199-03 overall verification (`go test ./...` with the known archived-path test excluded)
- **Observed:** The aggregate normal suite reported 8,636 passing tests, one failure, and 11 skips because colony-prime output differed before versus after deleting a generated `COLONY_PRIME.md`.
- **Isolation check:** The exact failing test passed immediately on its own; Plan 199-03 changes only additive `pkg/colony` lifecycle contracts and optional JSON fields.
- **Race check:** The full race suite passed 8,636 tests across 20 packages when this suite-order-sensitive test and the existing archived-path fixture were excluded.
- **Why deferred:** The failure is an unrelated command-test interaction and is not reproducible in isolation, so changing command behavior here would exceed Plan 199-03's lifecycle-contract scope.
- **Follow-up:** Audit shared process/environment or generated-file state in the command test corpus under a dedicated test-isolation change.

## Legacy Next Up snapshots still encode the pre-199 lifecycle policy

- **Found during:** Plan 199-04 broader command-package verification (`go test ./cmd -count=1`)
- **Observed:** The package run reported 7,088 passing tests, 60 failures, and 10 skips. Most failures assert the policy this plan deliberately supersedes: discuss before plan, build preferred over run, targeted recover/build-force routes instead of resume, mandatory entomb after seal, or golden cards containing those older recommendations.
- **Isolation check:** The Plan 199-04 contract suite passes all 36 focused tests, and the existing read-only-loader and token-accounting guard tests pass. The package still compiles and reaches the later tests; the mismatches are expectation-level changes rather than a failure of the new projection path.
- **Why deferred:** Plans 199-09, 199-10, 199-26, 199-32, and 199-33 own the command-specific renderers, focused closeouts, retired recovery wording, and post-seal status migration. Updating those snapshots here would pre-empt their scoped migrations and hide the deliberate transition.
- **Follow-up:** Update each legacy expectation alongside its owning renderer migration, then run the exact full and race repository gates in Plan 199-29.

## Plan 199-06 full command suite still crosses the staged lifecycle migration

- **Found during:** Plan 199-06 overall command-package verification (`go test ./cmd -count=1`)
- **Observed:** After the plan-owned catalog, reachability, and visual-writer regressions were corrected, the remaining failures were the archived Plan 196 branch-disposition fixture plus legacy Next Up expectations in `TestWrapperPartialFinalizeDoesNotShowTheFinishedBuildScreen`, `TestPrintNextUpUsesTargetedRecoveryCommand`, `TestPrintNextUpReadyUsesCurrentPhaseBuild`, `TestWorkflowSuggestionsFailedPhase`, and `TestSwarmDestroyRunsWorkerWavesAndReturnsStructuredResult`.
- **Isolation check:** Every Plan 199-06 contract and each directly affected audit gate passes in focused runs, including the maintenance landing, typed inspections, live skill inventory/diff, command catalog, reachability, visual-output discipline, and source-surface checks.
- **Why deferred:** These failures are concrete instances of the pre-existing migration items above. Their renderers and recovery wording belong to later Phase 199 plans; changing them in the expert-maintenance plan would cross ownership boundaries.
- **Follow-up:** Resolve them with the owning lifecycle-renderer plans, then rerun the full command and repository suites in Plan 199-29.

## Plan 199-09 supersedes legacy status-dashboard expectations

- **Found during:** Plan 199-09 broader command-package verification (`go test ./cmd`)
- **Observed:** The package run reported 7,157 passing tests, 100 failures, and 10 skips. The status-specific failures still assert the retired dashboard/JSON shape (granularity, version, memory-health, reconciliation, and hand-built Next Up fields), while the catalog/golden failures retain the prior status description. The remaining failures substantially overlap the already-deferred lifecycle-policy and archived-path items above.
- **Isolation check:** The exact Plan 199-09 runtime contract passes all 18 focused cases, including full/compact semantic agreement, all five health states, responsive rendering, absent-evidence wording, and zero-write fixtures. The exact wrapper/source-hygiene commands pass 4 and 7 cases respectively.
- **Why deferred:** This plan deliberately makes the versioned lifecycle projection authoritative and replaces the legacy status result. Reintroducing the retired dashboard solely to satisfy old snapshots would create two competing truth systems; the remaining cross-command migrations and golden refresh belong to later Phase 199 plans.
- **Follow-up:** Migrate or retire the legacy status expectations with their owning Phase 199 renderer/snapshot plans, then run the repository-wide normal and race gates in Plan 199-29.

## Plan 199-07 full command suite crosses the same staged migrations

- **Found during:** Plan 199-07 overall verification (`go test ./cmd -count=1`)
- **Observed:** The exact front-door and wrapper contracts pass, while the aggregate package still reports the already-recorded legacy status/Next Up expectations, the archived Phase 196 branch-disposition fixture, and the stale command-catalog golden from Plans 199-06/09.
- **Isolation check:** All ten `TestFrontDoor*` cases, `TestCommandSourceHygiene`, the source-check regressions, the existing init compatibility suite, and `go test ./pkg/colony` pass in isolation. The catalog mismatch contains only the previously changed `maturity` and `status` metadata; it contains no Plan 199-07 command.
- **Why deferred:** These failures reproduce outside the front-door paths and are already assigned to later renderer/snapshot cleanup and the final Phase 199 verification gate.
- **Follow-up:** Resolve the owning lifecycle migrations, refresh the catalog once their command contracts settle, and rerun the normal/race repository gates in Plan 199-29.

## Plan 199-10 full command suite still includes staged lifecycle migrations

- **Found during:** Plan 199-10 overall command-package verification (`go test ./cmd -count=1`)
- **Observed:** The aggregate package reported 7,201 passing tests, 108 failures, and 10 skips. The Plan 199-10 phase/history/agreement contracts pass all 29 focused cases in both normal and race-enabled runs. The broader failures remain concentrated in the already-recorded legacy status/Next Up expectations, old golden snapshots, missing archived Phase 196 fixture, and command-surface migrations owned by later Phase 199 plans.
- **Isolation check:** Existing `TestHistory*` plus the new history tests pass together (15 cases), the exact phase suite passes 14 cases, and the cross-view agreement suite passes 8 cases. Refreshing the command catalog would also absorb previously deferred `maturity` and `status` metadata, confirming that its mismatch is a shared staged snapshot rather than an isolated Plan 199-10 fix.
- **Why deferred:** Updating the unrelated legacy expectations or shared goldens here would cross the explicit ownership of later renderer, compatibility, and final-verification plans. Plan 199-10 changes no legacy Next Up policy and its focused views already use the authoritative projection.
- **Follow-up:** Complete the remaining Phase 199 command migrations, refresh shared catalogs/goldens once those contracts settle, and rerun the normal and race repository gates in Plan 199-29.

## Plan 199-19 maintenance YAML is not yet represented in the Codex command-guide catalog

- **Found during:** Plan 199-20 broader command-guide compatibility verification (`go test ./cmd -run '^TestCommandGuide' -count=1`).
- **Observed:** `TestCommandGuideCoversAllYamlCommands` reports `maintenance` as missing because `.aether/commands/maintenance.yaml` exists while `commandGuideCatalog` has no corresponding Codex definition.
- **Isolation check:** Plan 199-20's exact init parity, source-hygiene, wrapper-compatibility, and command-guide checks pass all 23 cases, and the live `command-guide init --platform codex` smoke check succeeds.
- **Why deferred:** The missing entry predates and does not exercise Plan 199-20's init-only surfaces. Adding an expert-maintenance Codex orchestration contract here would cross the maintenance surface ownership established by Plan 199-19.
- **Follow-up:** Add or deliberately exempt the maintenance entry with its owning Codex command-guide migration, then rerun the complete command-guide catalog test.

## Plan 199-13 supersedes public suffixed pause/resume compatibility tests

- **Found during:** Plan 199-13 broader command-package verification (`go test ./cmd -count=1`).
- **Observed:** The exact Plan 199-13 transaction, replay, provenance, redirect, wrapper, source-hygiene, and colony-model checks pass. The aggregate package still includes tests and catalog snapshots that require `pause-colony`/`resume-colony` to be public Cobra aliases or wrappers (`TestCanonicalAliasDelegates`, `TestTraceEndToEndResumeGeneratesNewRunID`, lifecycle reachability inventories, alias-repair reporting, and the audit catalog). `TestCLIInterruptedBuildResumesThroughForceRedispatch` also expects the older contradictory `EXECUTING`-with-no-start-time recovery state instead of the new safest runnable `READY` point. Other aggregate failures match the staged status/Next Up and archived-path items already recorded above.
- **Isolation check:** A real subprocess accepts the bounded hidden input redirect before Cobra, while command metadata and generated pause surfaces contain no public alias. The exact crash matrix passes at validation, staging, intent, partial-target, and final-verification faults, with byte-stable replay and zero-write conflict proof.
- **Why deferred:** Restoring Cobra aliases or legacy wrappers would directly violate D-13. Updating command-wide reachability inventories, catalog goldens, and remaining resume wrapper parity crosses the ownership of later Phase 199 compatibility/snapshot plans (including Plan 199-21 and the final Plan 199-29 gate).
- **Follow-up:** Migrate those legacy expectations to canonical `pause`/`resume`, remove the remaining suffixed resume wrapper surface in its owning plan, refresh shared catalogs after command contracts settle, and rerun the full normal/race repository gates in Plan 199-29.
