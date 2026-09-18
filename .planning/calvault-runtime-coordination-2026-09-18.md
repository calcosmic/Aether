# CalVault runtime incident coordination

Date: 2026-09-18
Source: [user-requested field handoff](../.handoff-calvault-runtime-20260918.md)
Status: source fixes implemented following the owner's instruction to proceed; focused integration and race checks passed. Broader suite timed out and reported failures; baseline attribution recorded below. No runtime publication or CalVault retry performed.

In plain English: repair Aether's reporting and prerequisite checks without treating unfinished work as approved migration work. The handoff is coordination, not approval of any pending permission prompt.

## Preserved work and authority

- Collector design is committed at `3c399305`; Plan 24 acquisition remains blocked and ANT-04–07 pending. These runtime incidents do not unlock its proof gates.
- Preserve the original handoff, all prior acquisition evidence, Phase 205 evidence, installed 1.0.81, unrelated `.gsd/` and concurrent source changes.
- No CalVault migration approval, manifest promotion, private-content edit, draft deletion, automatic retry, publication or owner-acceptance action is authorized by this coordination.
- Handoff anchor: `run-f3f98c73aeb67b5c319f454db3376cef`; attempt `attempt-20260918T104109.911596000Z-29906.json`; reported SHA-256 `8861df859cb006e8cfc5fbb19de55c24585979e21261fd72081dc4d6eecae6ff`. These are reported provenance, not a newly reproduced run.

## Initial ownership routing (historical)

Only the root and the existing Codex executor `/root/execute_204_2_24` are visible in this thread. The executor has received findings 1/3 for read-only triage. Other Codex/Claude sessions must acknowledge the proposed lanes below before source edits overlap. Lane labels are routing assignments, not claims that invisible agents have received a message or begun implementation.

| Finding | Owning lane / scope | Required coordination and regression |
|---|---|---|
| 1. P1 Auditor schema contradiction | Codex producer: `pkg/codex/worker.go`, related schema/prompt/parser tests; assigned existing Codex executor for triage | Auditor consumer lane owns `cmd/codex_continue_finalize.go` and review normalization tests. Agree one typed review shape; valid actual producer output must pass strict schema AND `normalizeContinueReviewEvidence`. Do not weaken review validation. |
| 2. P1 dispatch after prerequisite timeout | Queen/build lifecycle lane: `cmd/queen_wave_lifecycle.go`, dispatch portions of `cmd/codex_build.go`, lifecycle tests; awaiting external owner acknowledgement | Gate on accepted predecessor task receipts, including grouped jobs. Fake invoker must never receive dependent task 2.4 after missing 2.2/2.3 credit; independent safe work remains eligible. |
| 3. P2 false timeout activity | Codex producer lane owns timeout parsing/result representation alongside finding 1; assigned same executor for triage | Agree optional/unknown call-count semantics with runtime rendering owner before edits. Preserve incremental events and recoverable raw output; heartbeat means process liveness only. Test nonzero observed activity and genuinely unknown count without completion credit. |
| 4. P2 misleading rollback | Same Queen/build lifecycle owner as finding 2 for `cmd/codex_build.go`; coordinate visual renderer changes with finding 3 | State rollback must distinguish surviving uncredited drafts. Test worker writes then times out; report retained paths and reconciliation action, never erase files to match wording. |
| 5. P2 stale previous-phase closeout | Context/bookkeeping lane: authoritative closure selection and next-phase brief assembly; awaiting external owner acknowledgement | Follow actual context assembly to its source before claiming file ownership. Regression: blocked review → accepted out-of-band closure → pause/resume → next-phase brief uses accepted closure; historical reports remain retained and labelled. |

`pkg/codex/worker.go` has one writer for findings 1/3. `cmd/codex_build.go` has one writer for findings 2/4 and any call-count rendering there. Use narrow function ownership only after both writers agree; otherwise serialize. The context lane coordinates if its fix also touches build assembly. Producer/consumer contract integration must land together or be tested together before release.

## Initial delivery order and evidence (historical)

1. Confirm ownership and current tree before edits; preserve concurrent work.
2. Fix and test the strict Auditor producer/consumer contract and dependency admission first.
3. Fix timeout evidence, honest activity reporting and retained-draft messaging; increasing a timeout alone is insufficient.
4. Fix authoritative previous-phase context selection without deleting historical closeouts.
5. Run focused regressions and appropriate broader checks on the integrated candidate. Record source identities and results; no test run has occurred under this coordination yet.
6. Report the candidate and remaining limitations. Any CalVault retry must be separately scoped and respect its existing approvals and drafts. Runtime fixes do not make the 1,444 unapproved rows executable.

The original handoff retains exact project evidence paths and reproduction details. Future owners should append acknowledgement, concrete files and fix/test commits here rather than overwrite the incident history.

## Initial Codex producer acknowledgement (historical)

`/root/execute_204_2_24` completed read-only triage on 2026-09-18; no edits, tests, CalVault access or permission actions. It accepts the producer lane pending cross-session edit ownership confirmation.

- Confirmed strict artifacts/review contradiction. Do not universally add nullable `review`: non-reviewer `review:null` can trigger the consumer's explicit-artifact rejection. Agree caste-specific output or exact null handling with the Auditor owner. Root reserves the cross-contract regression coordination until a single implementation owner acknowledges it.
- Confirmed timeout returns before claims parsing and defaults the integer tool count to zero. The parent ticker also calls `Pulse`, conflating heartbeat and observed output. Include `cmd/codex_continue.go:1819` reported-count handling in downstream coordination.
- Producer test candidates: `pkg/codex/worker_test.go`, narrowly scoped new schema/timeout tests, existing `worker_debug_artifact_test.go` and `task_receipt_contract_test.go`. Hosted-provider diagnostic persistence in `platform_dispatch.go` is a reuse candidate, not additional edit ownership.
- Silent timeout must remain unknown; explicit measured zero stays zero. Parent heartbeat alone cannot establish provider activity. Draft-producing timeout retains diagnostics/drafts without completion credit.

## Implementation after owner instruction to proceed

The following assignments supersede the read-only routing above. Existing work and evidence were preserved. All changes are confined to Aether source and tests.

| Owner | Delivered change |
|---|---|
| `/root/execute_204_2_24` — Codex producer | Caste-specific strict review artifacts for Auditor and Gatekeeper, with the actual invocation schema exported for contract testing. Failed/blocked reviewers can report unavailable review honestly; completed Auditor validation remains mandatory. Explicit tool-count presence, separately labelled observed calls, retained timeout diagnostics, and distinct provider activity versus process heartbeat. Files: `pkg/codex/worker.go`, `platform_dispatch.go`, `dispatch.go`, `worker_field_failure_test.go`. |
| `/root/calvault_lifecycle` — Queen/build lifecycle | Before each wave, require accepted prerequisite task credit, including individual receipts from grouped jobs; leave independent work eligible. Blocked dependents are never invoked and are recorded as blocked in the spawn tree. State rollback reports surviving tracked/untracked drafts without deleting them or assigning them completion credit. Files: `cmd/codex_build.go`, `queen_wave_lifecycle.go`, `calvault_build_safety.go`, `calvault_build_safety_test.go`. |
| `/root/calvault_context` — Context/bookkeeping | Bind accepted out-of-band closure to its phase, latest attempt and hashes of superseded reports. Newer reports or attempts invalidate the context substitution. Support existing closures conservatively using attempt provenance and file/report timestamps. Historical reports remain intact. Files: `cmd/phase_carry_forward.go`, `verify_out_of_band.go`, `calvault_carry_forward_test.go`. |
| `/root` — Integration/rendering | Carry measured-versus-unknown counts and diagnostic references through adapters and live summaries. Test actual producer schema against unchanged consumer validation for Auditor, Gatekeeper and non-reviewers. Files: `cmd/codex_build_progress.go`, `codex_continue.go`, `internal_worker_adapter.go`, `continue_worker_measurements_test.go`, `calvault_review_contract_test.go`. |

Cross-review corrected whitespace normalization of dependency aliases, accidental credit from prior unnamed tasks, skipped spawn-tree nodes, closure snapshot timing, corrupt attempt pointers, and legacy timestamp precision. The full producer/consumer review contract includes Gatekeeper because its existing brief also requests review findings.

### Validation record

- Initial focused command regressions passed normally (17.828s) and with the Go race detector (36.055s).
- Producer regressions passed normally and with the race detector, including an actual fake-CLI timeout that writes a retained draft and reports partial observed activity without completion.
- Final integrated command/producer regressions passed: `go test ./cmd ./pkg/codex -run 'TestCalVault|TestQueenWaveLifecycle|TestReviewerArtifact|TestAuditorScore|TestWorkerFlowStep|TestUnmeasuredWorker|TestLiveAndSummaryWorker|TestAuditorProducerSchema|TestWorkerToolCountPresence|TestWorkerHeartbeatDoesNotInvent|TestCodexTimeoutPreserves' -count=1 -timeout 5m` (cmd 131.504s, pkg/codex 2.622s). The same selection with `-race -timeout 8m` passed (cmd 129.692s, pkg/codex 6.547s).
- `go test ./... -count=1 -timeout 15m` did **not** pass. The command suite controller reached its deadline after executing 2,257 of 5,775 tests. All reported non-command packages passed. The broad run compiled before the final cross-review corrections; final focused checks above include those corrections.
- The broad run identified one new test-fixture violation: direct attempt-plus-latest writes in `calvault_carry_forward_test.go`. Replaced those with the canonical plan-only build-start transaction. Both carry-forward tests then passed; the static detector now lists only its two pre-existing fixture violations.
- Compared failures against an isolated, indexed `git archive` of base commit `3c399305f878db8dc3abbe475965b767af26195e`, with base Git objects available for evidence checks. Confirmed existing failures: current vocabulary inventory; build/continue visual golden snapshots; stale Phase 199 gate receipt; incomplete native qualification; Go test-command extraction; two legacy build-start fixture violations; earlier-attempt task credit; redispatch of already-proven task; missing automatic fix-attempt journal. These are not waived or repaired by this incident change. Native proof requirements and their blocked status remain unchanged.
- Logs: `/tmp/aether-calvault-final-focused.log`, `/tmp/aether-calvault-final-race.log`, `/tmp/aether-calvault-all.log`, `/tmp/aether-calvault-baseline-other.log`, `/tmp/aether-calvault-baseline-receipt.log`, `/tmp/aether-calvault-baseline-native.log`, `/tmp/aether-calvault-baseline-credit.log`.
- All 57 protected planning/evidence hashes matched the pre-change inventory. Installed runtime SHA-256 remains `b37f56f6f5c9800483df8e6ea702b5002c9bee7eaff0300bc02830b8b96479f3`.
- Final corrected carry-forward fixtures passed race detection (4.212s); log `/tmp/aether-calvault-closure-final-race.log`. The two remaining automatic-fix failures (missing separate journal and repeated automatic fix) also reproduced unchanged on the base commit (15.891s); log `/tmp/aether-calvault-baseline-fixremaining.log`. Every substantive failure observed in the broad run is now either corrected in our new fixture or reproduced on the unchanged base. The unexecuted tests remain unverified; no whole-suite pass is claimed.

### Delivery boundary

These fixes make prerequisite admission and reporting honest; they do not approve migration work. Installed Aether 1.0.81 has not been replaced. No CalVault retry, migration, manifest promotion, draft deletion, pending-prompt approval, Phase 205 acceptance or evidence qualification was performed. Plan 24's scoped collector remains a design, and acquisition/proof gates remain blocked.

## Subsequent local installation — 2026-09-18

The earlier delivery boundary above is historical. The five fixes were applied
to the previously installed stable baseline, focused normal/race checks passed,
and the repaired executable was atomically installed. CalVault resolves the new
SHA-256 `45663b7cc4ca018b5024ad78675f4255f894143b820484a8c2846c8270a7f816`.
The version label and hub remain 1.0.81. See
[local installation receipt](calvault-local-runtime-install-2026-09-18.md).
Full-suite qualification remains unpassed; no CalVault retry or migration approval.
