# Deferred Items

## 2026-09-07 — Phase 199 vocabulary inventory drift

- **Discovered during:** Phase 200 Plan 01 repository-wide verification (`go test ./...`)
- **Failing test:** `TestCurrentVocabulary199/tracked-occurrences-are-exhaustively-classified`
- **Observed mismatch:** `.planning/phases/199-front-door-and-classic-contract/199-UAT.md` contains one `legacy_pause` and one `legacy_resume` occurrence that the Phase 199 vocabulary inventory classifies with count zero.
- **Why deferred:** The failure is confined to Phase 199 documentation/inventory and is unrelated to the Phase 200 `pkg/colony` specification and planning-domain changes. The scoped Phase 200 package tests and vet gate pass.
- **Suggested follow-up:** Reconcile the two UAT occurrences with the exhaustive Phase 199 vocabulary inventory, then rerun `go test ./cmd -run TestCurrentVocabulary199 -count=1`.

## 2026-09-07 — Phase 199 gate receipt timestamp staleness

- **Discovered during:** Phase 200 Plan 02 command-package verification (`go test ./cmd -count=1`)
- **Failing tests:** `TestPhase199GateReceiptSchema` and `TestPhase199GateReceipt`
- **Observed mismatch:** The checked-in Phase 199 receipt now reports stale or invalid execution timestamps for its `go test ./...` gate.
- **Why deferred:** The receipt is Phase 199 historical verification metadata; refreshing or redesigning its time window is unrelated to Phase 200 planning-state validation and migration.
- **Suggested follow-up:** Re-run the Phase 199 gate-receipt workflow with current evidence, then verify `go test ./cmd -run 'TestPhase199GateReceiptSchema|TestPhase199GateReceipt' -count=1`.

## 2026-09-07 — Specification command registration follows projection engine

- **Discovered during:** Phase 200 Plan 08 repository-wide verification (`go test ./... -count=1`)
- **Failing test:** `TestGoSourceHintsMatchCobraContracts`
- **Observed mismatch:** The new canonical `SPEC.md` projection must show the exact draft-approval command, but `aether spec` and its `--approve`, `--revision-id`, `--revision-hash`, and `--approval-token` flags are not registered until the already-planned Phase 200 Plan 10 command-surface work.
- **Why deferred:** Registering Cobra commands changes `cmd/spec_cmd.go` and `cmd/root.go`, which are explicitly owned by Plan 200-10 and outside Plan 200-08's engine/projection file boundary. Hiding or weakening the required next command merely to evade the audit would make the projection dishonest.
- **Suggested follow-up:** Complete Plan 200-10, then rerun `go test ./cmd -run TestGoSourceHintsMatchCobraContracts -count=1` and the repository-wide suite.

## 2026-09-07 — Restored specification journey awaits shared next-action and parity plans

- **Discovered during:** Phase 200 Plan 11 repository-wide verification (`go test ./... -count=1`)
- **Failing tests:** `TestStartupLifecycleCardsComeFromTheResolver`, `TestStartupLifecycleEnvelopesMatchTheirCards`, `TestCLICompiledInstallToSealJourney`, `TestNextActionNeverHardcoded`, `TestAuditCatalogGolden`, `TestCompletionPacketSchemaMatchesStructs`, `TestPlatformParityGolden`, `TestRegressionSnapshot`, `TestSlashCommandGuidancePointsAtRealCommands`, and `TestNoRegisteredSubcommandIsUnreferenced`.
- **Observed mismatch:** Plans 200-10 and 200-11 now expose `aether spec` and make settled Discuss create a draft, while Phase 199's frozen lifecycle resolver, black-box journey, command inventory, wrapper reachability, schema, and parity snapshots still encode the former Discuss-to-Plan route or do not yet know the new public command.
- **Why deferred:** The failing assertions live outside Plan 200-11's Discuss implementation boundary and are explicitly assigned to later Phase 200 work: Plan 200-12 extends shared lifecycle facts/Next Up, Plans 200-20/21/24/25 migrate wrappers and inventories, Plan 200-22 publishes contracts, and Plan 200-23 proves the complete journey. Weakening the new draft-spec boundary or editing those future-owned artifacts early would recreate two competing sources of lifecycle truth.
- **Suggested follow-up:** After Plans 200-12 and 200-20 through 200-25 land, rerun the named ratchets and `go test ./... -count=1`; the Discuss-specific contract is already green under `go test ./cmd -run 'TestDiscuss|TestSpecification.*Draft' -count=1` and the same command with `-race`.

## 2026-09-07 — Staged-plan migration leaves legacy command-package expectations red

- **Discovered during:** Phase 200 Plan 14 command-package verification (`go test ./cmd -count=1`).
- **Failing areas:** Legacy whole-chain Plan iteration tests, Plan visual/golden expectations, colony-mode and boundary-question fixtures, startup lifecycle-card ratchets, the Phase 199 vocabulary inventory, command-hint/next-action audits, and already-recorded schema/audit catalog drift.
- **Observed mismatch:** The repository-wide command suite still expects the pre-Phase-200 whole-chain planning surface in 29 tests. None of the failures names or exercises the new `TestPlanningScoutStage*` or `TestCodexPlanFinalize*Scout` boundary; all 29 focused Plan 14 tests pass normally and under the race detector.
- **Why deferred:** Fixing these assertions would require future-owned lifecycle resolver, visual, wrapper, schema, inventory, and end-to-end files outside Plan 200-14's Scout-finalizer boundary. Those migrations are assigned to later Phase 200 plans, and changing them here would violate the sequential file-ownership contract.
- **Suggested follow-up:** Complete the remaining Phase 200 command/renderer/parity plans, then rerun `go test ./cmd -count=1` and `go test ./... -count=1`. Keep `go test ./cmd -run 'TestPlanningScoutStage|TestCodexPlanFinalize.*Scout' -count=1` as the Plan 14 regression gate.
