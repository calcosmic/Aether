---
phase: 204.2-codex-native-worker-lifecycle
verified: 2026-09-17
status: gaps_found
score: 0/5 fully qualified final-candidate success criteria
requirements_passed: []
requirements_pending: [ANT-04, ANT-05, ANT-06, ANT-07]
tested_regression_source: f9e32ca000c88b9b723393e4dbf2f2b5801593a8
post_full_test_fix_source: cf51e9ac86d11139d806591245d8afa5cbdd593b
tested_live_source: 165db28c767864a619a6d883c7a2d0db9593741a
---

# Phase 204.2: Codex Native Worker Lifecycle Verification

The implementation has executable boundary controls, but the retained final165 real-host matrix is incomplete and predates the subsequent production repairs. **Status: gaps_found; ANT-04 through ANT-07 remain pending.** Ten required scenarios are incomplete and one scenario demonstrates a missing-skill refusal. The required actual-evidence gate fails. Passing deterministic tests or successful replay of an incomplete capture cannot turn that result into native workflow qualification.

In plain English: Aether can check and save the right worker records in the controlled tests. The final real sessions still do not prove every promised worker, answer-delivery and recovery behavior. This report records the gap rather than declaring the phase finished.

## Source and evidence identity

- Live production and reviewed validator: `165db28c767864a619a6d883c7a2d0db9593741a`.
- Production digest: `sha256:f86229cf5dcdd2176c321c080a5e923b99c0e6fcc23cfd905385a5c634011210`.
- Candidate binary: `sha256:5e643b424b7fbcabfda1cb7d653316ad8d372b3e3d0d005e36dc6be7e8357288`; version 1.0.79, explicitly built with `-buildvcs=false`.
- Corrected source compilation: [root's disposable build receipt](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/corrected-build/receipt.json), exit 0 at `200b14c4`; binary `sha256:73a8143154acb14adaa385ea32dd36e8ff45f10c7f545b840c62b336427021cf`, also built with `-buildvcs=false`. This verifies compilation only; it was neither installed nor exercised as a new live candidate.
- Separately pinned live harness: `sha256:bafaed8881fa4853d20e030348bc26a56f6f65762ca92b7ace03d96572ca367c`.
- Superseded regression candidates: `5b6b4ab2266c1efbca2e531bbc6c35f173800433` used test corpus `sha256:8e37daa0383bf737a2662354b68656e3c0cdc36adb80e7874a87f151e14685c6`; `200b14c49ef539936d5e009ad5de1ef9eba9dc89` used production `sha256:c548b071d4e83c1483189f21e03728a96cc63934cdbf33349e224de2762211f2` and test corpus `sha256:fbf5ae798c526ed3d91ef22a97aa4a62af7375901ff8c3d6754c58279bf022db`. Each retained full normal preceded a necessary correction; neither was followed by full race.
- Final CI-only correction and regression source: `f9e32ca000c88b9b723393e4dbf2f2b5801593a8`; production `sha256:641942bc26bc7cc9022078c8247f914521e3c6d37bbe06bc76c395bf309b0a8f`; test corpus and live harness unchanged from200b. Both standalone clones independently matched all source/discovery bindings, listing21 packages /6,797 top-level cases /372 focused cases.
- Post-full test-only fixture fix: `cf51e9ac86d11139d806591245d8afa5cbdd593b`; test corpus `sha256:d2bcfc93a43d0cd9f6879d81e7773d3a8290f78e5a90b682e8ef82fc790e5c98`. Only `cmd/release_candidate_blackbox_test.go` changed in the corpus. All 1,709 production inventory entries and the live harness still match f9. The full pair predates this repair; no full execution on the new corpus is claimed. [Separate source check](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/post-full-test-fixture-source.json).
- [Final actual qualification](evidence/native-qualification.json) SHA-256 `41c1ea7b1b4f008ef3c8e024a293be8f142a2ae6ea5ba70777366610ea009be4`; [regression accounting](evidence/native-regression.json).
- Codex CLI 0.154.0, parent model gpt-6-astra/xhigh; Claude Code 2.1.272 comparison stopped at an authentication boundary.

Production inventory uses repository-relative paths and excludes `_test.go` and `.planning/`. The first full normal exposed a real provider/native recovery classification defect and missing wrapper guidance. Their authorized repairs change the final production digest. The final165 captures and aggregate remain unchanged historical evidence; they do not qualify the repaired production. No live binary is relabeled with the later Git HEAD. All regression assertions have a separate corpus pin. The actual gate checks current production bytes, the separately pinned harness, original/derived artifact hashes, exact source bindings and candidate provenance before calling the existing reviewed raw-receipt validator; a production mismatch is an additional required failure.

## Success criteria

| Criterion | Deterministic production-path evidence | Retained final165 real evidence and current disposition |
|---|---|---|
| SC1 — Installed skill, real accepted Builder, edits and credit once | Registered reserve/bind/record/stage/finalize; immutable journal and duplicate/stale refusals; `TestCodexNativeWorkerTracer`, admission, terminal, finalizer and auxiliary Watcher controls | **Gap.** Final ordinary child was actually spawned and waited, but its host task path and diagnostic UUID could not be bound consistently. It produced no edit/check/credit proof. Some later scenarios preserve child edits and runtime credit; they do not qualify the ordinary scenario. |
| SC2 — Explicit independent Watcher; ordinary proportionate team | `TestCodexNativeAuxiliaryWatcherLifecycle`, guide platform isolation and accepted manifest controls | **Gap with partial observations.** A distinct Keen-6 Watcher child ran an uncached TestClamp check and wrote testing/quality findings. The final reviewed scenario timed out at 900 seconds and lacks complete attribution/check/finalization qualification. No unconditional Watcher floor was added. |
| SC3 — Complete context and exact scoped answers | Returned launch bytes, brief/source digests, bounded packing, registered question → bound answer → returned delivery → exact ACK, stale interleavings, protected-route and flag-write controls | **Gap.** The final question capture has a saved bound envelope, completed-send observation and later changed behavior. Inherited history already contained the fixture answer; therefore the behavior is not exclusively attributable to the scoped message. Encrypted host exports establish call/child linkage, not independently decrypted payload equality. |
| SC4 — Finished helper survives fresh-session interruption | Public pause/resume transactions, preserved per-child terminal authority, no-launch/unresolved/cancelled projections, idempotent staging/finalization | **Gap with partial observations.** Partial-resume retained finished worker bytes, used public resume in a fresh parent and launched one new helper for unfinished work; finalizer replay was stable. Parent command attribution and corrected initial prompt prevent full qualification. Early-resume did not reach the complete fresh-session boundary; spawn-gap remains incomplete. |
| SC5 — Actual controls, visibility and honest usage | Native capability admission, actual observation projection, no PID death inference, cancellation request versus acknowledgement, unreported usage preserved | **Gap.** Workspace-write shows an inside write and nested response, but lacks the required outside-write attempt; read-only lacks the required write attempt. Cancellation records a request, not confirmed cancellation. Missing usage remains unreported. |

## Requirement dispositions and call paths

| Requirement | Production boundary | Evidence-backed disposition |
|---|---|---|
| ANT-04 — Governed native dispatch | Installed ant-build → Codex-only command guide → Go plan-only manifest → non-launching native bridge → exact child record → shared stage/finalizer | **Pending.** Deterministic identity and causal negative controls pass; the final ordinary real-host assignment/edit/check/credit chain is incomplete. |
| ANT-05 — Durable native recovery | Existing per-child worker-run journal → shared aggregate staging → public pause/resume and saved attempt currency → existing finalizer | **Pending.** Controlled crash/currency/replay boundaries pass, and partial-resume has substantive raw evidence; complete current-source real qualification is still missing. |
| ANT-06 — Complete helper context | Canonical context resolver and exact launch prompt → protected native decision scope → immutable context envelope → completed-send observation only | **Pending.** No generic answer leakage, stale delivery or read-only acknowledgement is accepted in controls. Exact host plaintext and exclusive causal answer influence are not fully proved by the final capture. |
| ANT-07 — Honest capabilities and visibility | Typed native observations and current host contract → inline activity and truthful recovery; explicit unsupported-control refusal | **Pending.** Required permission/cancellation/workspace probes are incomplete. The missing-skill refusal is observed, not a successful workflow. |

Relevant implementation is in `cmd/codex_native_worker.go`, `codex_native_context.go`, `codex_native_decisions.go`, `codex_native_finalize.go`, `codex_native_recovery.go`, `build_worker_run.go`, `build_attempt.go` and the existing finalizer/session routes. The saved native terminal is the per-child authority; only the shared finalizer can grant build credit. Stage and context reads do not independently grant credit. A replay does not launch or acknowledge again.

**Owner check/credit policy:** failed deterministic checks block VERIFIED phase advancement while retaining valid receipt-backed BUILD task credit. Plan05's actual exit-7 control proves that distinction. This report does not treat BUILD credit as verified completion or claim failed checks erase attributable completed work.

## Actual capture review

The [Plan09 summary](204.2-09-SUMMARY.md) and aggregate preserve all eleven final captures unchanged. Plan08's [bounded read-only capture review](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/live-boundary-review.json) links each case to its original/derived source-pinned receipt. Outcomes are ordinary, early-resume, review, partial-resume, question, cancellation, spawn-gap, controls, controls-read-only and claude **incomplete**; missing-skill **observed refusal**.

Watcher evidence is attributable to child `01a0afcf-69c0-7880-bfdc-ea78cb5e918b`, distinct from Builder `01a0afc8-ab78-71b3-b853-e39ba798133d`. Its raw child log records the uncached test at lines 59/62 and testing/quality ledger writes at 73/80, with readback at 104. Findings identify the existing five table cases, inclusive boundary behavior and untested equal/inverted/extreme cases. This is useful independent checking, not a demonstrated newly discovered code defect. Saved completion prose is not used to replace missing raw qualification evidence.

The question capture binds Weld-70 / child `01a0afe0-84d7-7280-886a-e118cdb0f8b3` to delivery `814d0f8fcafa124b87084731e9144f22a99e15403d9aa6c1b2b333483d705d21`, payload SHA-256 `66285ea75fed1e27570db242d886af00c83878a1c1d973c366baea09b9c646c0`, and completed host send `call_B6ZlpIwl8qOzgDfLHlqALgyJ`. The answer is explicitly a predeclared fixture response, never owner testimony. The saved chain and later behavior remain partial evidence because of inherited-answer contamination and encrypted plaintext limits.

Partial-resume uses fresh parent `01a0afd9-c617-7be1-bb11-2f97ffb8367d`, preserves the first worker and finalizer replay, and starts one permitted unfinished assignment. It does not claim zero total new helpers. Cancellation keeps `cancel_requested` nonterminal when acknowledgement is absent. The ordinary capture's missing diagnostic child ID does not mean no real waiting child existed.

Original Plan01 Builder/early-resume evidence remains meaningful history at its own production digest. Original Plan07 c370 captures remain immutable earlier-candidate evidence. Neither qualifies the final f9/current candidate. Replay exit zero proves stable derivation/integrity of the retained outcome, not successful execution of an incomplete scenario. `parent_substitution=true` includes unclassified parent operations and is not by itself proof that the parent implemented the child's code.

## Required checks and full accounting

| Run at f9e32ca0 | Top-level cases | Total cases | Pass / fail / skip | Exit / duration |
|---|---:|---:|---:|---|
| Focused normal | 372/372 | 1,071 | 1,069 / 1 / 1 | 1 / 6m30s |
| Focused race | 372/372 | 1,071 | 1,069 / 1 / 1 | 1 / 8m30s |
| Full normal | 6,797/6,797 | 12,676 | 12,632 / 27 / 17 | 1 / 50m00s |
| Full race | 6,797/6,797 | 12,676 | 12,632 / 27 / 17 | 1 / 52m44s |

Both full runs account for all 21 packages and every discovered top-level Test/Example. Eighteen other tested packages passed; root and cmd/aether are no-test package skips. There are no missing, duplicate or out-of-order authoritative cases. The normal/race parser excludes exactly 12,097/12,098 explicitly marked diagnostic-echo events without assigning execution credit. Stderr is empty, and no race-detector report or outer 90-minute timeout occurred. One individual black-box test times out as detailed below.

The [normal raw log](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/final-ci-normal.jsonl) hashes to `c31106fe5d54a5d099c3333e4e2a126c653f765379d923189d4f8d6e863455c9`; [race raw log](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/final-ci-race.jsonl) hashes to `cf7c18e08576020c635a85209afc3ef446fdc28fa83cf5b30602425653735692`. Full normal ended 18:56:29 UTC and race 18:59:12 UTC. The independently pinned accountant outputs, discovery, original invocations and per-case reviews are linked from the regression receipt; complete accounting is not a passing regression verdict.

The literal strict Task1 command ran after accounting, exited 1 in 14.01 seconds, and refused both incomplete/stale actual native evidence and the required focused evidence failure. Its [invocation and exact receipt pin](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/final-ci-strict-task1-invocation.json) preserve the limits of that result. The literal [strict Task2 check](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/final-ci-strict-task2-invocation.json) at cf51 exited 1 in 3.02 seconds: actual evidence remains incomplete/stale, and the explicit corpus check correctly refuses to treat the f9 full receipt as a run of the repaired test code.

### Exact residuals

Both full runs contain the same 19 failing top-level tests, totaling 27 failed cases. Fifteen top-level tests and their three failed subcases match the retained substantive historical diagnostics. [Normal reviews](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/final-ci-normal-reviewed-diagnostics.json) and [race reviews](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/final-ci-race-reviewed-diagnostics.json) pin every current diagnostic, baseline and attributable comparison.

| Historical top-level test | Retained substantive failure |
|---|---|
| TestBuildStartLegacyHelpersRetired200 | Exactly two old fixture writers: classic_voice_event_test.go:77 and failure_evidence_test.go:29 |
| TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount | Worker count 8 equals maximum selected castes 8 |
| TestCurrentVocabulary199 | 197 keys versus 193; same four missing path/family/count classifications; includes failed tracked-occurrences subcase |
| TestFailedCheckSendsExactlyOneBuilderFixAttempt | Zero fix records instead of one |
| TestFixAttemptIsCountedSeparately | Missing separate fix journal |
| TestFixAttemptNeverOverwritesTheFirstResult | One original journal instead of two; only added normalized dump field is empty BriefSHA256 |
| TestGoSourceHintsMatchCobraContracts | Unknown aether interpret hint in partial_work_label.go |
| TestGoldenBuildVisualOutput | Queen parenthetical differs from older visual expectation |
| TestGoldenContinueVisualOutput | Chart icon differs from older visual expectation |
| TestHumanFacingOutputGoesThroughWriteVisualOutput | Same three watch_live.go direct-output sites |
| TestNoSecondAutomaticFixAttempt | Unexpected second automatic check-fix state |
| TestPhase199GateReceipt | Stale protected fingerprint |
| TestPlanningAdversarial200 | Nested public-path/vocabulary failure with the same four classification keys; includes failed integrated-public-reads subcase |
| TestPlanningPublicPaths200 | Same nested vocabulary failure; includes failed Phase199-focused-gates subcase |
| TestResolveTestCommand_GoProject | Same historical CLAUDE prose parsed into the Go test command |

`TestAuditCatalogGolden`, `TestCompletionPacketSchemaMatchesStructs` and the repaired `TestWiringGateStepRunsEveryWiringTest` passed both full runs. They are not inherited exceptions.

`TestCodexNativePhaseEvidence` is the required new evidence failure, kept outside the historical list. Three additional top-level failures account for eight failed cases in each full run:

| New failing group | Original observed failure and disposition |
|---|---|
| TestCLIInterruptedBuildResumesThroughForceRedispatch | Ten-second sentinel wait expires in both runs; race additionally reports nonempty fixture locks during TempDir cleanup. The interrupted/resume behavior is not proved by those failed invocations. |
| TestDisposedBranchesAreGoneFromTheRepository | Three historical disposed commit objects are absent from the independent clones. Their recorded hashes remain unchanged; the full audit failed. |
| TestPackedNPMReleaseCandidateContract and five subcases | Missing consumer node_modules/aether-colony/bin/aether.js. Retained npm debug logs directly show ancestor home-project discovery and home node_modules writes because the empty consumer lived under the home tree. This is an execution-isolation defect, not an approved historical failure. |

Separate bounded followups preserve these original failures:

- [Archived-object diagnosis](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/new-failure-diagnosis/branches/receipt.json), SHA-256 `a7f376c644c5c946166803cf5c1931fbae0b09fa558593c833240557510b1a38`: source already contained all three objects. Importing exactly those objects into the disposable normal clone restored its targeted normal and race test at unchanged f9 source. No production file or source-repository ref changed; neither original full failure is rewritten as pass.
- [NPM fixture correction and causal controls](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/new-failure-diagnosis/npm-fixture/receipt.json), SHA-256 `f1ebf4fa8bc4f606ce4baaf49655c42ef813b8878455dc5ce73d358c562bfe41`: root's test-only cf51 fix passes explicit `npm --prefix consumer`. The original command reproduces ancestor manifest/lock mutation and sentinel deletion in a disposable project; the corrected command installs into the consumer and preserves all ancestor sentinels. The existing package contract passes 6/6 normal and 6/6 race, with no skips, under that disposable ancestor. This is bounded repair evidence on the changed corpus, not replacement full-run credit.
- [Interrupted-build diagnosis](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/new-failure-diagnosis/interrupted/REVIEW.md), SHA-256 `c773a4f8948ee7e1d8c1179a219d89105b8b2d1d8b4fadfae6a63c329efecb60`: one f9 targeted normal passes with a longer isolated TMPDIR. Simple path length is not supported as the cause. The existing harness calls Fatal at the ten-second readiness timeout before Kill/Wait and hides child output; race remnants show subsequent writes against the deleted fixture. Extra full-suite load is plausible but unproven. The original readiness cause and cleanup/diagnostic defect remain unresolved; no timeout change, source correction or broad rerun was made.

The accidental home-NPM mutation received a separately authorized [partial recovery](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/npm-isolation-incident/recovery-receipt.json). Only attributable Aether entries were removed from three home JSON files, preserving their modes; the Aether package directory and bin symlink were moved into quarantine. All 12,618 unrelated hash checks passed, and recovery did not run npm. The incident inventory still contains 31 additions, 64 changes and 17 removals whose prior state is not established; those were not blindly reversed. This is partial containment, not a claim that the home project has been fully restored.

The first full normal at `5b6b4ab2` ran for 26m06s and exited 1. Its controller reported 5,732/5,732 command cases, while strict JSON accounting found two confirmation prompts had swallowed terminal protocol lines. The first raw log retains 12,640 runs and 12,638 terminal actions: 12,581 pass, 40 fail and 17 skips. Thirty top-level failures include fifteen historical names, fourteen newly exposed regressions and the required actual-evidence failure. Both the repaired schema and catalog checks passed. Fourteen old failures had the same substantive diagnostics; the fifteenth source scanner had three additional native corruption-fixture findings, subsequently removed by a bounded pointer-only helper. The first run was not followed by race. Corrections and source changes justify the new reviewed normal/race pair, while all original evidence remains retained.

The original shared-checkout full runs were serialized because TestMain's cleanup has process-local authority only. The200b full normal was complete but caught one new CI selector omission: its gate did not include the three newly added native reachability tests. The one-line YAML repair f9e32ca0 passed22/22 targeted normal and22/22 race checks without changing Go/test bytes. Root then authorized one final full normal and one final full race concurrently in two independent standalone clones, each with separate Git directories/refs/indexes, temporary roots and npm caches. The [isolation review](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/standalone-concurrency-read-review.json) and [manifest](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/plan08/final-ci-isolation-manifest.json) record the removed mutation boundary and remaining CPU/I/O timing limits; no new failure is waived as contention. Both use the literal `go test ./... [-race] -count=1 -timeout 90m -json` contract without selectors or skips. Independent discovery includes Example functions and packages with no tests. The explicitly delimited `full-suite controller failed:` diagnostic echo is retained in raw logs but supplies no duplicate execution credit; duplicates before that boundary and missing cases still fail validation.

The final evidence gate runs by default with the checked-in actual aggregate, or an explicit `AETHER_CODEX_NATIVE_RECEIPT_PATH`. Strict `AETHER_CODEX_NATIVE_REQUIRE_REGRESSION=1` additionally requires completed source/corpus-bound regression accounting. Neither mode waives the current native evidence gap. The legacy raw receipt test explicitly dispatches aggregate, versioned raw and historical unversioned raw schemas; unknown nonempty schemas fail.

Historical classification records provenance, not a passing behavior claim. In particular, retained worker-count, automatic fix-attempt and lifecycle source/receipt failures remain limitations of the affected behavior; none can discharge an ANT requirement or success criterion. The fixed schema and catalog cases must pass on the corrected candidate and are never inherited exceptions.

At f9 the strict regression validator stops at the first failing required focused run; after the cf51 test-only fix it stops earlier at the changed corpus. Neither failed invocation is represented as successful validation of every downstream full-run receipt. Full normal/race execution and historical failure comparison are independently established by the separately pinned complete raw logs, discovery, accountant outputs and attributable diagnostic reviews. Current production remains unqualified for actual-host behavior; all final165 observations above retain their historical candidate identity.

Deterministic controls first establish a passing fixture before removing child identity, changing artifact digests/source, substituting parent-only edits or supplying exit zero without native events. Parser controls cover Examples, marked diagnostic echo, real duplicates, orphan/out-of-order terminals, missing cases, package failure and race/fatal output. Discovery controls reject narrowed full arguments, invented denominators and empty focused selections; changed assertion bytes invalidate the regression corpus even with unchanged test names.

The earlier complete normal explicitly skipped three actual-host opt-ins, archived data-flow/worker-economy audits, opt-in fixture/golden regeneration and missing local release/colony/Queen fixtures. The legacy empty-capsule and unwritable-store cases also skipped because their preconditions could not be produced in that environment; those skipped cases supply no proof of those failure paths. Final run skip identities/reasons are retained with their own raw accounting.

The parameterized legacy receipt tests return without claiming actual replay when their receipt variables are absent. FreshHost's explicit opt-in skip gives no live proof. Normal/race suites do not set the aggregate receipt-path environment variable; the default phase gate still executes.

## Probe and prohibition dispositions

| Planning item | Disposition and limits |
|---|---|
| ANT-04 adjacency / empty / ordering | Deterministic duplicate/conflicting/empty/stale admission and equal-wave order controls pass; final live workflow qualification remains missing. |
| ANT-06 boundary / precision | Exact returned bytes, Unicode, packing, request-size and successful-send-only ACK controls pass; encrypted host plaintext remains separately unverified. |
| ANT-05 unclassified assumption | **Flagged, unresolved.** Partial-resume and spawn-gap exercise concrete cases, but incomplete final evidence does not classify every interruption/reconnect edge. |
| ANT-07 unclassified assumption | **Flagged, unresolved.** Findings are specific to measured Codex version/configuration; missing sentinel attempts and cancellation acknowledgement remain missing. |
| No parent/subprocess substitution presented as native success | **Flagged-unverified for the global claim.** The bounded engineering reviews preserve incomplete outcomes and lane distinctions, but unclassified parent operations prevent a complete no-substitution judgment. |
| Engineering proof is not owner acceptance | **Bounded engineering judgment recorded and finally rechecked.** [Codex/root's final preservation review](/Users/callumcowie/.aether-backups/phase204-2-execution-20260917/owner-boundary-final-review.json), recorded at 19:10:25 UTC on 2026-09-17, confirms the original inventory hash is unchanged and no tracked Phase205 path changed since execution began. Its authority is engineering execution review, expressly not owner acceptance; no personal verdict is inferred. |

At final source f9 (unchanged Go bytes from200b), `pkg/codex/platform_contract.go:57–61,76–85` explicitly marks per-child permission selection, separate native worktree allocation and Aether-governed nesting unavailable. Admission refuses those requested controls and accepts only the supported inherited workspace-write profile in the accepted shared workspace. A real nested host response does not establish governed Aether recruitment. These known unsupported controls are distinct from incomplete workspace/sentinel probes. Raw child-attributed `token_usage_record` events do exist, but Aether does not collect them: saved usage and cost remain unreported, never zero.

## Owner boundary and remaining scope

The original Phase205 before-inventory remains SHA-256 `3b2b678284a26332addeadd2bd700bd4f51c4e2b0e6907a987116c85e5c2a8dd`. Its checkpoint and receipts were read-only; no owner session or walkthrough was performed. The required ten personal journeys and owner verdict remain owner actions.

No shared Aether runtime installation, authentication change, publish, release, CalVault action, Phase205 execution or milestone seal was performed by Plan08. The NPM fixture nevertheless caused the unintended home-project package mutation described above; that must not be hidden by the runtime-install statement. Phase204.3 owns full workflow/command coverage; 204.4 owns shared recruitment/native worktree/Swarm recovery; 204.5 owns final candidate qualification and the new 205 handoff. Those later scopes do not waive the outstanding 204.2 proof.

Close the current mandatory actual-host gaps on an explicitly qualified candidate, preserve the original captures, and rerun affected evidence if production/harness changes. The root orchestrator owns final GSD code review and shared tracker disposition. **A completed execution/accounting plan is not a passed phase verdict.**
