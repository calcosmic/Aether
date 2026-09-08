---
phase: 200-iterative-planning
plan: 23
generated_at: 2026-09-08T05:26:48Z
repository_revision: c431f0d3dd2c8daed55fac3e94e082c8d4937b4c
branch: oracle-reinstate
go_version: go1.26.5 darwin/arm64
implementation_gate: blocked_by_inherited_phase_199_inventory
phase_200_owned_gates: pass
owner_product_acceptance: not_claimed
---

# Phase 200 Implementation Gate Receipt

## Outcome

Phase 200's semantic corpus, public-path journey, staged candidate acceptance, compiled lifecycle, named migrations, source parity, and every Phase 200-owned failure discovered by the original and two independent repository sweeps now pass. The repository-wide normal and race commands still exit nonzero on exactly four protected Phase 199 test nodes: the vocabulary subtest and parent plus the Phase 199 receipt schema and receipt validators.

This receipt therefore **does not claim a green implementation gate**. It also does not claim Phase 205 owner product acceptance, publication, deployment, or release completion.

Plain English: two independent reruns found planning regressions that earlier, truncated runs missed. Those regressions are fixed. The only remaining red checks are an older Phase 199 bookkeeping list that omits two words already in its UAT document and the Phase 199 receipt checks invalidated by that stale evidence.

## Tested Tree

- Base revision: `c431f0d3dd2c8daed55fac3e94e082c8d4937b4c`
- Branch: `oracle-reinstate`
- Go: `go version go1.26.5 darwin/arm64`
- Final evidence timestamp: `2026-09-08T05:26:48Z`
- Intentional Plan 23 dirty files during final gates: none; all follow-up repair commits through `c431f0d3` were complete before the final full and race commands started.
- Protected pre-existing items left untouched and unstaged: modified `.planning/config.json`; untracked `.gsd/`; untracked `.planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md`.
- Execution isolation: shared checkout, explicitly forced to none by the orchestrator.

## Deterministic Artifact Digests

All values are SHA-256 digests of the final tested bytes.

| Artifact | SHA-256 |
| --- | --- |
| `cmd/testdata/classic-contract/v1/schema.json` | `d6ff25e119c2f8e1c91233acb00924de5fdd5560268ad362d43e40e8b7cf950e` |
| `cmd/testdata/classic-contract/v1/mechanisms.json` | `32d1f6573c9b905756bead362d04047fe0bfdc1647aa45ce82a29f665902efbf` |
| `cmd/testdata/classic-contract/v1/cases.json` | `73f86a3607871e9f613ea576a44ec79870b7ca3b8d5c7f81907964ce491d87dc` |
| `cmd/classic_contract_test.go` | `59790e6dc71cbd2ffcabd471a0ea5ae1b9361f903b1c8adaf8e9e17dbcc8740b` |
| `cmd/planning_public_paths_200_test.go` | `91077b1ddb5f459a508a109c3db9954ed3fe09866fd684e8e1e0570ee7b819f7` |
| `cmd/planning_real_repo_200_test.go` | `25eac28f914eee121a23a2b6af604e8038e55e767877bb28bfd194cc8eacc2f9` |
| `.aether/schemas/completion-packet.schema.json` | `99d22bd0062a9bd699490b9d59d34bac812019ff68082247baae2cbfdc1fa476` |
| `cmd/testdata/golden_plan.txt` | `ffe07e301748b3e4e5c7274ee0c27449313e71d43cc6454a5060a55a385c1765` |
| `cmd/next_action.go` | `ce3bac5a713326f534308096462a90f2c23eef2644eac382bd969aced86a67ae` |
| `cmd/codex_plan.go` | `14737314f9483e3905354e1fff0468cbafafa010979662adb304233a87850c12` |
| `cmd/spec_cmd.go` | `e09a0ea60c59405e96f8f31778b38cc4d59ae70a7844e975d104d16378315af5` |
| `cmd/orchestrator_boundary_questions_test.go` | `8f1eff9dc21823b67033fc2a7d8a8c786feb7212163b201a57a9c1befafb6bba` |
| `cmd/orchestrator_boundary_guidance_test.go` | `bf2ec92014a7938579739d4d7b862249eb088fb8e3312900a8d4e8950e48945c` |
| `cmd/testdata/command_catalog.json` | `52b53d57d8ed12066845bd04c8a4c271fbc7919fe5fb72cf8d6987a8733b5325` |
| `cmd/testdata/parity_snapshot.json` | `4b20a3d7e5713f46a30dc17117b863671db6da7653e45d4e0c6e9faca03273d5` |
| `cmd/testdata/regression_snapshot.json` | `e81330f2c1dc55a25abaf4d4fd126b771bc249cac46edad1e8ed23ebd6350b13` |
| `cmd/planning_visuals.go` | `bf416ca6829bb1f40dac4e78cc787911ac0b133d71a650cf787d6ccbb5359b71` |
| `cmd/planning_state.go` | `9fc71203af6589d7a3115d38012cf245df873280cdf5e2f096d081dc5c757a49` |
| `cmd/codex_plan_finalize.go` | `b15e40a9056b91966d8add223335cc8c6e108430deede021a56cf3b6bf1c2c51` |
| `cmd/blackbox_harness_test.go` | `24770a321b448ba320baeb2400a79f983d5416b163c52d62e4dae5546112b9e3` |
| `cmd/e2e_lifecycle_test.go` | `fd29ae330107dbc36f375b4c0f38099c897cac6f1717080314c69fead10814ec` |
| `cmd/testing_main_test.go` | `9c4eae5b160a4e35db997698802f929279ab53cfd58dde8d4f0ac3d9c6b845d2` |

Corpus inventory: 22 mechanisms total, exactly 12 mechanisms `SYN-200-01` through `SYN-200-12`; 102 cases total, exactly 16 Phase 200 cases covering the eight required `V-200-*` groups with one success and one refusal each.

## Final Gate Commands

### Independent corrections and superseded results

The dispatch baseline was **5,840 passed, 28 failed, and 6 skipped**. That established the starting migration surface before Plan 23's final reconciliation.

The first independent Wave 13 run reported **5,889 passed, 9 failed, and 6 skipped**. Seven Phase 200 nodes exposed command-advice registry bypasses and missing fresh staged-plan Orchestrator boundaries. The exact post-fix set passed 13/13.

The second independent run reported **8,950 passed, 26 failed, and 8 skipped**. Beyond the four protected Phase 199 nodes, it exposed stale command/parity/regression goldens, the orphan `plan-research-approve` command, legacy plan-only/delegate expectations, and a nil staged-manifest panic. The exact post-fix set passed 36/36.

Two uncapped follow-ups then exposed deeper layers hidden by the earlier panics and RTK's 1,048,576-byte retained-log cap:

- **9,523 passed, 15 failed, 11 skipped:** eight isolated cases passed focused; three deterministic lifecycle cases exposed direct-acceptance fixtures and accepted-revision status drift.
- **9,455 passed, 15 failed, 11 skipped:** the remaining deterministic node was a Plan closing golden; ten otherwise-green isolated tests reached the stock parent-package deadline before their deferred launches.

Those results supersede the earlier receipt's truncated 3,881/2/6 normal and 4,875/2/7 race claims. The final evidence below comes from uninterrupted, uncapped commands after every repair commit.

| Gate | Exact command | Result |
| --- | --- | --- |
| Independent failure reproducer | `go test ./cmd -run 'Test(NextActionNeverHardcoded\|PlanFinalizeAddsOrchestratorBoundaryGuidance\|OrchestratorBoundaryQuestionsCreatedForPlanOnlyWorkflows\|DefaultColonyModeDoesNotCreateBoundaryQuestions\|ResolvedBoundaryQuestionFlowsThroughClarifiedIntent)$' -count=1` | PASS after repair — 13 tests, 1 package |
| Second independent reproducer | `go test ./cmd -run 'Test(PlatformParityGolden\|RegressionSnapshot\|PlanOnlyUnchanged\|NoRegisteredSubcommandIsUnreferenced\|TerritoryWrapperAuthority199\|WrapperOrchestratedCommandsPreserveLiveWorkerCeremony\|PlanDelegateManifestCarriesOneSteeringNote\|PlanAndColonizeDelegateLanesCarryEveryMemorySource\|PlanAndColonizeCapsulesMatchTheInProcessLane\|DelegateCapsuleRendersSteeringAndRelayExactlyOnce\|DelegateCapsuleIsStableAcrossRuns)$' -count=1` | PASS after repair — 36 tests, 1 package |
| Latent lifecycle selection | `go test ./cmd -run 'Test(CLIVersionedPlanRevisionSurvivesRestartAndBindsNextBuild\|LegacyReviewerSeverityDirectNormalizationFailsClosed\|LegacyReviewerSeverityExternalFinalizeFailsClosed\|TerritoryLifecycleTransactionalPublish\|PauseResume199SafeBoundary\|EveryLifecycleCommandEndsWithNextAction\|FullLifecycleInDownstreamRepo\|SuggestionOnlyCriticalDirectFlowFailsBeforeAdvancement\|SuggestionOnlyCriticalExternalFinalizeFailsBeforeAdvancement\|SealTransaction199HivePolicy\|ReviewerArtifactDirectFlowFailsClosed\|ReviewerArtifactExternalFinalizeFailsClosed)$' -count=1` | PASS — 16 tests, 1 package |
| Golden and isolation helpers | `go test ./cmd -run 'Test(GoldenPlanVisualOutput\|IsolatedProcessHelper.*)$' -count=1` | PASS — 15 tests, 1 package |
| Focused Phase 200 | `go test ./cmd -run 'TestClassicContract.*Phase200\|TestPlanningPublicPaths200\|TestPlanningRealRepo200' -count=1` | PASS — 45 tests, 1 package |
| Complete classic contract | `go test ./cmd -run 'TestClassicContract' -count=1` | PASS — 134 tests, 1 package |
| Named migration selection | `go test ./cmd -run 'Test(FrontDoorInitWrapperParity\|InitSuggestedNextMatchesTopProposal\|FrontDoorHelp\|FrontDoorInit\|PlanManifestCarriesDepthProposal\|PlanEmitsPhaseResearchDispatchesFromDraft\|PhaseResearchDispatchedOncePerPhase\|PlanWrapperCardsParity\|PlanWrapperCeremonyContract\|PlanWrapperStageSkeleton\|LifecycleCommandDocsPreferRuntimeCLI\|PlanningContractDocuments200\|LifecycleFlatMirrorsMatchCanonical\|LifecycleWrappersAvoidRetiredDepthVocabulary\|LifecycleWrappersCarryStructuredBlocks)' -count=1` | PASS — 217 tests, 1 package |
| Source parity | `go run ./cmd/aether source-check` | PASS — 147 surfaces checked: 16 canonical, 5 retired mirrors, 126 generated wrappers; 0 findings; state effect `none` |
| Repository-wide | `go test ./... -count=1` | EXPECTED BASELINE FAIL — 9,550 passed, 4 failed, 11 skipped across 20 packages; 19 packages green; only protected Phase 199 nodes in `cmd` |
| Repository-wide race | `go test ./... -race -count=1` | EXPECTED BASELINE FAIL — 8,526 passed, 4 failed, 8 skipped across 20 packages; no race detector diagnostics; only protected Phase 199 nodes in `cmd` |

`rtk go test` was used as the output-preserving wrapper for noisy test commands; the table spells the underlying Go commands without that presentation wrapper.

The literal commands, with Markdown table escaping removed, were:

```sh
go test ./cmd -run 'Test(NextActionNeverHardcoded|PlanFinalizeAddsOrchestratorBoundaryGuidance|OrchestratorBoundaryQuestionsCreatedForPlanOnlyWorkflows|DefaultColonyModeDoesNotCreateBoundaryQuestions|ResolvedBoundaryQuestionFlowsThroughClarifiedIntent)$' -count=1
go test ./cmd -run 'Test(PlatformParityGolden|RegressionSnapshot|PlanOnlyUnchanged|NoRegisteredSubcommandIsUnreferenced|TerritoryWrapperAuthority199|WrapperOrchestratedCommandsPreserveLiveWorkerCeremony|PlanDelegateManifestCarriesOneSteeringNote|PlanAndColonizeDelegateLanesCarryEveryMemorySource|PlanAndColonizeCapsulesMatchTheInProcessLane|DelegateCapsuleRendersSteeringAndRelayExactlyOnce|DelegateCapsuleIsStableAcrossRuns)$' -count=1
go test ./cmd -run 'Test(CLIVersionedPlanRevisionSurvivesRestartAndBindsNextBuild|LegacyReviewerSeverityDirectNormalizationFailsClosed|LegacyReviewerSeverityExternalFinalizeFailsClosed|TerritoryLifecycleTransactionalPublish|PauseResume199SafeBoundary|EveryLifecycleCommandEndsWithNextAction|FullLifecycleInDownstreamRepo|SuggestionOnlyCriticalDirectFlowFailsBeforeAdvancement|SuggestionOnlyCriticalExternalFinalizeFailsBeforeAdvancement|SealTransaction199HivePolicy|ReviewerArtifactDirectFlowFailsClosed|ReviewerArtifactExternalFinalizeFailsClosed)$' -count=1
go test ./cmd -run 'Test(GoldenPlanVisualOutput|IsolatedProcessHelper.*)$' -count=1
go test ./cmd -run 'TestClassicContract.*Phase200|TestPlanningPublicPaths200|TestPlanningRealRepo200' -count=1
go test ./cmd -run 'TestClassicContract' -count=1
go test ./cmd -run 'Test(FrontDoorInitWrapperParity|InitSuggestedNextMatchesTopProposal|FrontDoorHelp|FrontDoorInit|PlanManifestCarriesDepthProposal|PlanEmitsPhaseResearchDispatchesFromDraft|PhaseResearchDispatchedOncePerPhase|PlanWrapperCardsParity|PlanWrapperCeremonyContract|PlanWrapperStageSkeleton|LifecycleCommandDocsPreferRuntimeCLI|PlanningContractDocuments200|LifecycleFlatMirrorsMatchCanonical|LifecycleWrappersAvoidRetiredDepthVocabulary|LifecycleWrappersCarryStructuredBlocks)' -count=1
go run ./cmd/aether source-check
go test ./... -count=1
go test ./... -race -count=1
```

## Remaining Failure

Exact reproducer:

```text
go test ./cmd -run 'Test(CurrentVocabulary199|Phase199GateReceiptSchema|Phase199GateReceipt)$' -count=1 -v
```

Observed root cause:

- `TestCurrentVocabulary199/tracked-occurrences-are-exhaustively-classified` reports 193 tracked occurrence keys versus 191 inventory keys.
- `.planning/phases/199-front-door-and-classic-contract/199-UAT.md` contains one `legacy_resume` occurrence classified as zero by the Phase 199 inventory.
- The same Phase 199 UAT file contains one `legacy_pause` occurrence classified as zero by the inventory.
- Go reports the failing subtest and `TestCurrentVocabulary199` parent as two failed test nodes.
- `TestPhase199GateReceiptSchema` and `TestPhase199GateReceipt` are the other two protected nodes; the stale vocabulary evidence/totals invalidate that checked-in Phase 199 receipt.

This protected baseline predates Plan 200-23 and is already recorded in `deferred-items.md`. Plan 23 did not edit the Phase 199 UAT document, inventory, or receipt because those artifacts are outside this plan's ownership and the orchestrator explicitly prohibited changing user-owned Phase 199 evidence merely to hide the baseline.

## Bounded Retry Record

Nine scoped repair passes were used across the initial and two independent gates:

1. Migrated obsolete immediate-planning assumptions in finalizer, ceremony, visual, colony-mode, and stuck-plan fixtures; regenerated the completion-packet schema. The focused repair gate passed 10/10, and the next full run reduced the repository result from 12 failed nodes to the two-node Phase 199 aggregate.
2. Replaced the retired immediate-dispatch plan golden with the explicit unbiased preset boundary and made the isolated-child self-test decline unsafe launches near its parent deadline. The dedicated race gate passed 2/2.
3. Migrated startup lifecycle-card fixtures so Discuss owns the draft-Specification boundary and Plan owns the no-write preset boundary. The dedicated race gate passed 16/16.
4. Moved exact Specification approval/repair and Plan preset commands into the shared next-action candidate registry, gated them against the live Cobra tree, and folded the preset-required result through the shared resolver. The ratchet plus command/spec/golden selection passed 17/17. Commit: `0a671f86`.
5. Restored Orchestrator boundary materialization, manifest fields, and exact preset re-entry on the fresh staged-plan path; migrated the old whole-chain boundary fixtures through approved Specification and explicit-preset authority. The exact independent failure selection passed 13/13 and the broader boundary/planning selection passed 66/66. Commit: `1d981d8b`.
6. Removed the orphan `plan-research-approve` command through Cobra's source of truth, refreshed command/parity/regression goldens, and moved plan-only/delegate assertions to staged one-Scout authority. The second independent selection passed 36/36. Commits: `db590537`, `6952546c`.
7. Migrated real revision, compiled-install, provider-backed, and downstream lifecycle fixtures to staged candidate review plus exact acceptance. Direct legacy activation is no longer used as current authority. Commit: `00377491`.
8. Fixed candidate readiness after a completed prefix and synchronized mutable phase/task/watcher facts into the active revision across build and continue process boundaries. The latent lifecycle set passed 16/16. Commit: `00377491`.
9. Routed preset-required Plan output through the shared lifecycle closing renderer, recorded that card in the golden, and extended only Go's stock parent-package deadline so deferred isolated tests still launch with their own bounded child budgets. The helper/golden set passed 15/15. Commits: `5629597a`, `99164f77`, `c431f0d3`.

The final normal and race results above are evidence-only reruns after the ninth repair. No Phase 200-owned failure recurred.

## Automated Verification Versus Product Acceptance

These checks establish implementation semantics, deterministic artifacts, source parity, and absence of detected Go data races in all executed code. They do not replace owner review of the product experience. Phase 205 remains responsible for owner product acceptance, and separate publish/deploy/release workflows remain required before distribution.
