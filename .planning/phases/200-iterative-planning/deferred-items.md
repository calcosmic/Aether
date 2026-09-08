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

## 2026-09-08 — Legacy Plan-only fixtures stop at the approved-SPEC boundary

- **Discovered during:** Phase 200 Plan 17 command-package verification (`go test ./cmd -count=1`).
- **Failing tests:** `TestPlanEmitsLifecycleCeremonyEvents`, `TestPlanFinalizeAddsOrchestratorBoundaryGuidance`, `TestOrchestratorBoundaryQuestionsCreatedForPlanOnlyWorkflows/plan`, `TestDefaultColonyModeDoesNotCreateBoundaryQuestions/plan`, and `TestResolvedBoundaryQuestionFlowsThroughClarifiedIntent`.
- **Observed mismatch:** These older Plan-only fixtures do not create an approved specification. Current planning correctly returns `planning did not start: an approved specification is missing. State is unchanged. Run aether spec` before emitting the legacy event or boundary-question payload; the final test then panics while type-asserting the absent payload.
- **Why deferred:** The failures precede and do not exercise Plan 200-17's impact closure, immutable insert candidate, or Seal authority gate. Updating the Plan/orchestrator compatibility fixtures belongs to the later Phase 200 public-path and end-to-end plans, while weakening the approved-SPEC gate would violate D-10 and D-12.
- **Suggested follow-up:** Update the legacy fixtures to complete the approved specification handoff before invoking Plan, then rerun the five named tests and `go test ./cmd -count=1`.

## 2026-09-08 — Repository-wide planning migration expectations remain red

- **Discovered during:** Phase 200 Plan 18 repository-wide verification (`go test ./... -count=1`).
- **Observed result:** 5,823 tests passed, 27 failed, and 6 were skipped. The failures cluster in legacy Plan finalizer/visual fixtures, lifecycle-card and Next Up ratchets, the completion-packet schema, Phase 199 vocabulary inventory, and orchestrator-boundary fixtures. None exercises or names the shared Plan 18 build/autopilot authority gate; its 23 focused tests and 86 broader build/run/planning tests pass, including the focused race run.
- **Why deferred:** The failing files and contracts are pre-existing or assigned to later Phase 200 public-surface, schema, parity, and end-to-end plans. Changing them from Plan 18 would cross its accepted-plan authority ownership boundary.
- **Suggested follow-up:** Complete the remaining Phase 200 public-path and contract plans, then rerun `go test ./... -count=1`. Retain `go test ./cmd -run 'TestPlanAuthority|TestPlanAcceptanceGate200|TestAutopilotPolicy.*Authority|TestCodexBuild.*Authority' -count=1` as the Plan 18 regression gate.

## 2026-09-08 — Legacy command-suite fixtures still predate iterative planning screens

- **Discovered during:** Phase 200 Plan 19 command-package verification (`go test ./cmd -count=1`).
- **Observed result:** 4,281 tests passed, 30 failed, and 5 were skipped. The failures remain concentrated in pre-Phase-200 Plan/preset/specification fixtures, old Plan visual and golden expectations, lifecycle-card and Next Up ratchets, the completion-packet schema, Phase 199 vocabulary inventory, and orchestrator-boundary fixtures. The next-action ratchet separately reports four existing sites in `cmd/codex_plan.go` and `cmd/spec_cmd.go`; Plan 19 introduced no remaining site.
- **Why deferred:** Those tests and source sites are outside Plan 200-19's four renderer files and are already assigned to later Phase 200 public-surface, schema, parity, and end-to-end plans. The Plan 19 semantic, JSON, authority, width, no-color, Specification-identity, and canonical-routing gates all pass (17 tests total).
- **Suggested follow-up:** Complete Plans 200-20 through 200-25, then rerun `go test ./cmd -count=1` and reconcile the remaining golden/schema/inventory expectations with the approved specification and preset boundaries.

## 2026-09-08 — Remaining cross-plan migration gates are outside init ownership

- **Discovered during:** Phase 200 Plan 24 repository-wide verification (`go test ./...`).
- **Observed result:** 5,822 tests passed, 34 failed, and 6 were skipped. The failures remain in legacy Plan finalizer/visual/orchestrator fixtures, Discuss wrapper/source hygiene, lifecycle startup cards, completion-packet schema, Phase 199 vocabulary inventory, and existing command-advice ratchets. No failure names Plan 24's init runtime, init wrapper, or front-door test files; its exact 24-test verification and 124-surface source check pass.
- **Why deferred:** Resolving these failures requires files owned by other Phase 200 plans or historical Phase 199 metadata. Plan 24 only owns the init-to-discuss handoff and may not rewrite Plan, Discuss, schema, or vocabulary contracts.
- **Suggested follow-up:** Complete the remaining Phase 200 migration plans, then rerun `go test ./...`; retain `go test ./cmd -run 'TestFrontDoorInitWrapperParity|TestInitSuggestedNextMatchesTopProposal|Test.*Init.*Wrapper|TestFrontDoor200' -count=1` plus `go run ./cmd/aether source-check` as the Plan 24 regression gate.

## 2026-09-08 — Remaining command-suite fixtures await the Phase 200 corpus pass

- **Discovered during:** Phase 200 Plan 22 extra command-package verification (`go test ./cmd -count=1`).
- **Observed result:** The sweep initially reported 4,296 passing, 30 failing, and 5 skipped tests. Plan 22 directly fixed the two failures caused by its new contract inventory/structure and the two stale visual-wrapper closeout assertions encountered by the required plan regex. The other reported failures remain in pre-existing Plan finalizer/preset fixtures, Plan visual and golden output, lifecycle startup cards, completion-packet schema, Phase 199 vocabulary inventory, hard-coded next-action ratchets, and orchestrator-boundary fixtures.
- **Why deferred:** Those mechanisms and fixtures are outside Plan 22's public contracts and Codex guidance ownership, and most were already recorded by Plans 14, 17, 18, 19, and 24. Weakening approved-Specification, staged-dispatch, or exact-acceptance behavior to satisfy them would violate Phase 200 authority decisions.
- **Suggested follow-up:** Use Plan 200-23's executable corpus/integration pass to reconcile the remaining fixtures, then rerun `go test ./cmd -count=1`. Plan 22's required 70-test cross-contract regex, 182-test exact Task 1 gate, command-guide/severity checks, and 126-wrapper source parity all pass.
