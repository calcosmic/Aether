---
phase: 200-iterative-planning
plan: 23
generated_at: 2026-09-08T03:25:11Z
repository_revision: 01b61ab34a3a9a2d7d91a410748594b4eb5bae17
branch: oracle-reinstate
go_version: go1.26.5 darwin/arm64
implementation_gate: blocked_by_inherited_phase_199_inventory
phase_200_owned_gates: pass
owner_product_acceptance: not_claimed
---

# Phase 200 Implementation Gate Receipt

## Outcome

Phase 200's semantic corpus, public-path journey, named migrations, source parity, and every Phase 200-owned failure discovered by the repository sweeps pass. The repository-wide normal and race commands still exit nonzero because one protected Phase 199 documentation/inventory mismatch reports as a failing subtest plus its parent aggregate.

This receipt therefore **does not claim a green implementation gate**. It also does not claim Phase 205 owner product acceptance, publication, deployment, or release completion.

Plain English: the new planning workflow is green; the only remaining red check is an older bookkeeping list that has not been updated for two words already present in the Phase 199 UAT document.

## Tested Tree

- Base revision: `01b61ab34a3a9a2d7d91a410748594b4eb5bae17`
- Branch: `oracle-reinstate`
- Go: `go version go1.26.5 darwin/arm64`
- Final evidence timestamp: `2026-09-08T03:25:11Z`
- Intentional dirty files during final gates: the Task 3 migration fixtures, regenerated completion-packet schema, and current receipt listed in this plan's summary.
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
| `cmd/planning_real_repo_200_test.go` | `a0d19f9edbba7959d4ac24b76866b3b5a13d9f81c77ea9d27ac4fa6a5fc43af4` |
| `.aether/schemas/completion-packet.schema.json` | `99d22bd0062a9bd699490b9d59d34bac812019ff68082247baae2cbfdc1fa476` |
| `cmd/testdata/golden_plan.txt` | `ab6f54ee107f76524536e9f735dc485ae8fef463fa38238603b5808f6d770dff` |

Corpus inventory: 22 mechanisms total, exactly 12 mechanisms `SYN-200-01` through `SYN-200-12`; 102 cases total, exactly 16 Phase 200 cases covering the eight required `V-200-*` groups with one success and one refusal each.

## Final Gate Commands

| Gate | Exact command | Result |
| --- | --- | --- |
| Focused Phase 200 | `go test ./cmd -run 'TestClassicContract.*Phase200\|TestPlanningPublicPaths200\|TestPlanningRealRepo200' -count=1` | PASS — 45 tests, 1 package |
| Complete classic contract | `go test ./cmd -run 'TestClassicContract' -count=1` | PASS — 134 tests, 1 package |
| Named migration selection | `go test ./cmd -run 'Test(FrontDoorInitWrapperParity\|InitSuggestedNextMatchesTopProposal\|FrontDoorHelp\|FrontDoorInit\|PlanManifestCarriesDepthProposal\|PlanEmitsPhaseResearchDispatchesFromDraft\|PhaseResearchDispatchedOncePerPhase\|PlanWrapperCardsParity\|PlanWrapperCeremonyContract\|PlanWrapperStageSkeleton\|LifecycleCommandDocsPreferRuntimeCLI\|PlanningContractDocuments200\|LifecycleFlatMirrorsMatchCanonical\|LifecycleWrappersAvoidRetiredDepthVocabulary\|LifecycleWrappersCarryStructuredBlocks)' -count=1` | PASS — 217 tests, 1 package |
| Source parity | `go run ./cmd/aether source-check` | PASS — 147 surfaces checked: 16 canonical, 5 retired mirrors, 126 generated wrappers; 0 findings; state effect `none` |
| Repository-wide | `go test -timeout 180s ./... -count=1` | FAIL — 3,880 passed, 2 failed, 6 skipped across 20 packages; 19 packages green; one inherited root cause in `cmd` |
| Repository-wide race | `go test -race -timeout 300s ./... -count=1` | FAIL — 4,871 passed, 2 failed, 7 skipped across 20 packages; no race detector diagnostics; same inherited root cause in `cmd` |

`rtk go test` was used as the output-preserving wrapper for noisy test commands; the table spells the underlying Go commands without that presentation wrapper.

The literal commands, with Markdown table escaping removed, were:

```sh
go test ./cmd -run 'TestClassicContract.*Phase200|TestPlanningPublicPaths200|TestPlanningRealRepo200' -count=1
go test ./cmd -run 'TestClassicContract' -count=1
go test ./cmd -run 'Test(FrontDoorInitWrapperParity|InitSuggestedNextMatchesTopProposal|FrontDoorHelp|FrontDoorInit|PlanManifestCarriesDepthProposal|PlanEmitsPhaseResearchDispatchesFromDraft|PhaseResearchDispatchedOncePerPhase|PlanWrapperCardsParity|PlanWrapperCeremonyContract|PlanWrapperStageSkeleton|LifecycleCommandDocsPreferRuntimeCLI|PlanningContractDocuments200|LifecycleFlatMirrorsMatchCanonical|LifecycleWrappersAvoidRetiredDepthVocabulary|LifecycleWrappersCarryStructuredBlocks)' -count=1
go run ./cmd/aether source-check
go test -timeout 180s ./... -count=1
go test -race -timeout 300s ./... -count=1
```

## Remaining Failure

Exact reproducer:

```text
go test ./cmd -run 'TestCurrentVocabulary199' -count=1 -v
```

Observed root cause:

- `TestCurrentVocabulary199/tracked-occurrences-are-exhaustively-classified` reports 193 tracked occurrence keys versus 191 inventory keys.
- `.planning/phases/199-front-door-and-classic-contract/199-UAT.md` contains one `legacy_resume` occurrence classified as zero by the Phase 199 inventory.
- The same Phase 199 UAT file contains one `legacy_pause` occurrence classified as zero by the inventory.
- Go reports the failing subtest and `TestCurrentVocabulary199` parent as two failed test nodes; they are one inventory mismatch.

This mismatch predates Plan 200-23 and is already recorded in `deferred-items.md`. Plan 23 did not edit the Phase 199 UAT document or its inventory because those artifacts are outside this plan's ownership and the orchestrator explicitly prohibited changing the user-owned Phase 199 evidence merely to hide the baseline.

## Bounded Retry Record

Three scoped repair passes were used, after which no further auto-fix was attempted:

1. Migrated obsolete immediate-planning assumptions in finalizer, ceremony, visual, colony-mode, and stuck-plan fixtures; regenerated the completion-packet schema. The focused repair gate passed 10/10, and the next full run reduced the repository result from 12 failed nodes to the two-node Phase 199 aggregate.
2. Replaced the retired immediate-dispatch plan golden with the explicit unbiased preset boundary and made the isolated-child self-test decline unsafe launches near its parent deadline. The dedicated race gate passed 2/2.
3. Migrated startup lifecycle-card fixtures so Discuss owns the draft-Specification boundary and Plan owns the no-write preset boundary. The dedicated race gate passed 16/16.

The final normal and race results above are evidence-only reruns after the third repair. No Phase 200-owned failure recurred.

## Automated Verification Versus Product Acceptance

These checks establish implementation semantics, deterministic artifacts, source parity, and absence of detected Go data races in all executed code. They do not replace owner review of the product experience. Phase 205 remains responsible for owner product acceptance, and separate publish/deploy/release workflows remain required before distribution.
