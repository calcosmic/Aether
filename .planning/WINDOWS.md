---
schema_version: 1
open_count: 18
waived_count: 0
fixed_count: 5
total_count: 23
last_updated: 2026-09-13T15:58:28.351Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 193 | unmet-truth | cmd/caste_relevance.go |  | Probe/auditor/gatekeeper still legitimately double-dispatch on build AND continue with no explicit Queen proposal (isAlwaysRequired computes the same required-caste independently on both flows); 193-02 only removed the watcher's implicit build-side dispatch. Flagged in 193-02-PLAN.md frontmatter as unclassified/not-auto-resolved; scoped to Phase 194 (moves the required-caste floor). | fixed |  | 2026-08-22T13:55:31.245Z | 2026-08-23T16:04:14.666Z |
| 2 | 196 | deviation | pkg/codex/worker.go |  | assembledPromptChars deleted outside the plan's files_modified; dead after the fallback removal | open |  | 2026-08-28T10:11:03.801Z |  |
| 3 | 197 | stub | cmd/codex_visuals.go |  | The pause card's Handoff line still reads 'Colony handoff saved for later resumption' - repo-invented word 'colony' with no plain-English gloss. Found by TestNextActionCardSpeaksPlainEnglish; the plain-English check was scoped to the new next-action card because rewording the ten legacy cards is plans 197-04 and 197-06's declared scope. | fixed |  | 2026-08-28T19:48:11.715Z | 2026-08-29T00:00:00.000Z |
| 4 | 197 | deviation | cmd/next_action.go |  | 197-02 edited cmd/next_action.go, which is outside its declared files_modified, to change one word in the failed-phase recommendation ('Running it again' -> 'The next step is to retry it') so the pre-existing TestWorkflowSuggestionsFailedPhase assertion on the word 'retry' kept passing without weakening it. | open |  | 2026-08-28T19:48:15.796Z |  |
| 5 | 198 | deviation | cmd/codex_visuals.go |  | renderPlanVisual's confidence branch only type-asserts to map[string]interface{}; both runCodexPlanWithOptions and runCodexPlanFinalize always store confidence as a codexPlanConfidence struct, so the Confidence line never renders on the direct or chat path. Out of scope for 198-04 (prohibited from editing that file, owned by a same-wave plan); needs a dual-type fix mirroring planning_loop's existing struct/map switch. | fixed |  | 2026-08-29T17:49:53.592Z | 2026-08-29T18:28:52.436Z |
| 6 | 198 | deviation | cmd/codex_visuals.go |  | renderPlanVisual never reads result["research_warning"] (which renderResearchFailedWarning's own doc comment says exists "so the omission is durable and visible" when a phase was planned without its research), result["research_failed_phases"] (the phase IDs that fed that warning), or result["gaps"] (a completed plan's own unresolved gaps). Found and allow-listed, not fixed, in 198-09-PLAN.md Task 1 (cmd/testdata/rendered_field_allowlist.json) because cmd/codex_visuals.go was owned this wave by sibling plan 198-08. | fixed |  | 2026-08-29T19:15:56.000Z | 2026-08-29T20:05:27.000Z |
| 7 | 198 | deviation | cmd/codex_plan_finalize.go |  | runCodexPlanFinalize never calls closeLifecycleRun (unlike continue-finalize/completeSealRuntime), so the plan finalizer's own suggested result["next"] command never folds into the unified next-action envelope renderLifecycleClosing reads back -- the closing card instead independently resolves a next step from live colony state, which usually matches but is not guaranteed to. Found and allow-listed, not fixed, in 198-09-PLAN.md Task 1 (cmd/testdata/rendered_field_allowlist.json, "plan completed" and "plan mid-loop") because cmd/codex_plan_finalize.go was outside that plan's declared files. | fixed |  | 2026-08-29T19:15:56.000Z | 2026-08-29T20:05:27.000Z |
| 8 | 198.2 | deviation | cmd/codex_continue_finalize.go |  | externalContinueReviewReport does not append a 'review wave skipped' narration step when no reviewers were dispatched, unlike the direct lane's runCodexContinueReview -- discovered by 198.2-05's dual-lane outcome-text parity test, out of that plan's scope to fix | open |  | 2026-08-30T14:49:33.437Z |  |
| 9 | 201 | deviation | cmd/golden_workflow_test.go |  | TestGoldenBuildVisualOutput/TestGoldenContinueVisualOutput: stale golden fixtures missing the 'Cost: not known...' line; pre-existing, last touched by Phase 200, discovered during 201-15 full-suite run | open |  | 2026-09-10T19:49:27.735Z |  |
| 10 | 201 | deviation | cmd/phase199_gate_receipt_test.go |  | TestPhase199GateReceipt fails on pre-existing untracked .gsd/ directory present before 201-15 started | open |  | 2026-09-10T19:49:27.844Z |  |
| 11 | 201 | deviation | cmd/queen_judgement_test.go |  | TestQueenChoiceReachesTheDispatchList / TestNoWorkerWithoutStatedReason: a Queen-requested Measurer with a stated reason is dropped before spawn; pre-existing, discovered during 201-15, out of scope (queen_judgement.go not a declared file) | open |  | 2026-09-10T19:49:27.959Z |  |
| 12 | 202 | unrun-verify | cmd/ (~15 tests, see 202-15-SUMMARY.md Issues Encountered) |  | Full unscoped 'go test ./cmd -count=1' could not be confirmed clean: pre-existing failures unrelated to plan 202-15 (TestNextActionNeverHardcoded, TestPhase199GateReceipt, TestQueenChoiceReachesTheDispatchList, TestNoWorkerWithoutStatedReason, TestCurrentVocabulary199, TestHumanFacingOutputGoesThroughWriteVisualOutput, TestGoldenBuildVisualOutput, TestGoldenContinueVisualOutput, TestAuditCatalogGolden, TestBuildStartLegacyHelpersRetired200, TestGoSourceHintsMatchCobraContracts, TestCompletionPacketSchemaMatchesStructs, TestPlanningAdversarial200, TestSkillManifestReadEmpty, TestSkillManifestReadFromHub, TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount, TestFailedCheckSendsExactlyOneBuilderFixAttempt, TestFixAttemptIsCountedSeparately, TestFixAttemptNeverOverwritesTheFirstResult, TestNoSecondAutomaticFixAttempt) plus a documented ~20min machine-specific suite ceiling; every test touching a file this plan changed passes, including under -race. | open |  | 2026-09-11T15:35:47.524Z |  |
| 13 | 202.1 | deviation | cmd/watch_live.go | 425 | Pre-existing (unrelated to this plan) TestHumanFacingOutputGoesThroughWriteVisualOutput failure: runColonyLiveRefreshLoop writes directly to stdout/stderr, bypassing writeVisualOutput | open |  | 2026-09-12T17:42:11.287Z |  |
| 14 | 203 | deviation | pkg/codex/worker.go |  | 203-02 added a one-line exported helper in pkg/codex/worker.go, outside its declared files_modified, so cmd/recruitment_dispatch.go could reuse the existing tested process-group termination logic instead of writing a second copy. Deliberate and documented in 203-02-SUMMARY.md Deviations; the alternative was duplicating kill-the-whole-process-tree logic in new code. | open |  | 2026-09-13T00:07:16.386Z |  |
| 15 | 203 | deviation | cmd/exchange.go |  | 203-05 edited cmd/exchange.go, cmd/exchange_import_sanitize_test.go, cmd/hook_cmds.go and cmd/signal_housekeeping.go, none of which were in its declared files_modified. cmd/exchange.go was required: the real 'aether import pheromones' path never stamped provenance, so the plan's own quarantine must_have would have been decoration without it. The other three carried the strength-floor constant fix. All documented in 203-05-SUMMARY.md Deviations; the plan's files_modified header was incomplete rather than the executor overreaching. | open |  | 2026-09-13T00:07:16.488Z |  |
| 16 | 203 | unrun-verify | cmd/testing_main_test.go |  | A full 'go test ./cmd -count=1' run can silently NOT execute a test while still reporting a lane result. Observed 2026-09-13: TestPlatformParityGolden appeared only in lane parallel-046's own 'missing executed tests' accounting line and never ran, so the wave-2 gate reported no regression while 203-02's new 'aether recruit' command had in fact broken the parity golden. The suite already detects and prints this condition; nothing treats it as a failure, so a reader comparing FAIL lines against a known-red list gets a false all-clear. The parity break itself is fixed (22149a3e); this entry is about the accounting, not that test. | open |  | 2026-09-13T10:00:01.055Z |  |
| 17 | 203 | unrun-verify | .aether/ts-host/test |  | The TypeScript host test suite has 29 failing tests that pre-date Phase 203, in test/lifecycle.test.ts, test/go-bridge.test.ts, test/golden-workflow.test.ts and the classic command parity matrix. Verified 2026-09-13 by running the suite at c9ebe0b1 in a detached worktree and diffing failure names against the post-wave-3 run: 29 before, the same 29 after, zero new. Most assert a pending-planning boundary and now receive 'an approved specification is missing. Run aether spec' instead, so they look like a spec-gate change the TS lane never absorbed. Recorded so a future wave-3-style gate can diff against a known set instead of re-deriving it; NOT investigated or fixed here. | open |  | 2026-09-13T10:57:40.741Z |  |
| 18 | 203 | unmet-truth | cmd/spawn.go |  | SECURITY (fail-open authorization bypass, PRE-EXISTING since phase 173, surfaced 2026-09-13 by a commit security review of cmd/recruitment.go): the delegation depth cap is bypassable by self-assertion. spawnParentIsRoot matches an unauthenticated caller-supplied name against the fixed sentinel list spawnRootParentNames = {Queen, Prime-1, Swarm} (cmd/spawn.go:22) and grants depth 0 with DepthIsAuthoritative=true and no spawn-tree entry required. Both 'aether spawn-can-spawn --name Queen' and the new 'aether recruit --parent Queen' therefore pass the depth check regardless of the caller's real depth, defeating spawnMaxDelegationDepth (the runaway-spawn and cost control). Everything ELSE in that path is correctly fail-closed: depth is read from the recorded spawn tree via latestSpawnEntryByName, never from a flag, and validateRecruitmentIntent (cmd/recruitment_intent.go:153) explicitly refuses a non-sentinel parent whose depth is not authoritative, with the reason 'a parent's depth is never trusted from a self-declared claim'. The sentinel exemption is the single hole. 203-03 mirrored spawn-can-spawn's existing behaviour deliberately and documented it; this is inherited, not introduced. Routed live to plan 203-06, which owns cmd/spawn.go + cmd/recruitment.go + cmd/recruitment_admission.go this wave and whose objective is extending that chokepoint with dimensions it does not yet check. Not fixed here: 203-06 is mid-flight in those exact files and an orchestrator edit would collide. | open |  | 2026-09-13T13:11:36.373Z |  |
| 19 | 203 | unmet-truth | cmd/testdata/orphan_allowlist.json |  | EXPECTED-RED, OWNED BY 203-15: TestNoRegisteredSubcommandIsUnreferenced fails with 'aether recruit is registered but nothing calls it (searched: wrappers, menu specs, hooks, scripts)'. Five plans (203-02/03/04/06/07) built the recruitment command and no wrapper, menu spec, hook or script invokes it yet. This is real and is exactly the orphan failure CLAUDE.md names as the project's signature defect -- it is NOT silenced. Plan 203-15 is the closer: it rewrites .aether/workers.md, which today documents a spawning protocol in convincing detail for a mechanism no caste was ever granted, and replaces it with the real path plus a test. DELIBERATELY NOT ALLOWLISTED: testdata/orphan_allowlist.json is shrink-only against a frozen baseline (TestOrphanAllowlistOnlyShrinks, WIRE-01/D-11), so adding an entry would widen a ratchet to hide a true finding. Leaving it red means the test itself proves 203-15 did its job, and going green is the acceptance signal. If 203-15 lands and this is still red, the phase shipped an orphan. Recorded 2026-09-13 at the wave-4 gate; cmd/testdata/regression_snapshot.json was separately refreshed 390->389 commands, the honest net effect of 203-10 retiring two dead trophallaxis commands. | open |  | 2026-09-13T13:36:11.546Z |  |
| 20 | 203 | deviation | cmd/swarm_scope_199_test.go |  | 203-12 Task 2 deleted SwarmPhase202Limitation, forcing an update to this pre-existing test's stale phase-202 assertions (fixed in the same commit, not deferred) | open |  | 2026-09-13T15:12:22.623Z |  |
| 21 | 203 | deviation | cmd/codex_verify_advance.go |  | Inherited from 203-10, still open after 203-12: recordTrophallaxisDecision's colony.LifecycleDecision is never threaded into runContinueAcceptVerifyAdvance (the single phase-level accept/verify/advance boundary, cmd/codex_verify_advance.go). 203-12 populated the CEC-07 credit join it actually owns (AgencyReceiptEvidence.ChangedDecision/EffectEvidence, cmd/agency_contract.go) from real trophallaxis+credit data, which is a different boundary from runContinueAcceptVerifyAdvance and fully satisfies this plan's own objective text. Whether a trophallaxis decision should ALSO reach the phase-level accept/verify/advance decision remains unresolved and cmd/codex_verify_advance.go is outside 203-12's declared file scope; a follow-up plan (203-14/203-15) should confirm intent. | open |  | 2026-09-13T15:12:32.588Z |  |
| 22 | 202.1 | unmet-truth | cmd/status.go |  | SEVERE / OWNER-REPORTED 2026-09-13: Phase 202.1's Classic-voice guarantee for 'aether status' is proved against a renderer the status command does not call. TestStatusScreenMeetsTheReferenceDensity and TestEveryVoicedScreenMeetsTheReferenceDensity measure renderLifecycleStatus (cmd/lifecycle_status_render.go) fed by classicVoiceStatusFixtureProjection, a hand-built projection. The real 'aether status' RunE calls renderDashboard (cmd/status.go:~60, via outputWorkflow), and renderLifecycleStatus's only non-test caller is cmd/compatibility_cmds.go:473. Measured on the live screen: 22 of 92 content lines symbol-led (~24%) against a reference bar of 41.3% -- the whole top block (Goal, Runtime, Signals, Progress, Focus, Instincts, Flags, Scope, Colony Mode, Depth, Granularity, Parallel) carries no leading symbol, while the memory/signals/worker sections below do. All five voice tests pass. The owner reported not seeing the Classic visuals they commissioned in 202.1; this is why. OPEN QUESTION not yet investigated: the same corpus registers build, continue, plan, discuss, spec, seal and the what-next card -- each needs the same check that its registered render function is the one its command actually calls. A green density test proves nothing about the screen if it measures a different function. | open |  | 2026-09-13T15:56:48.392Z |  |
| 23 | 202.1 | unmet-truth | cmd/planning_visuals.go |  | SCOPE ANSWERED 2026-09-13 (the open question from the status finding): all 21 screens registered in the Classic-voice corpus were traced from their registered render function to its production call sites. EXACTLY TWO are proved against code no command runs. (1) status-full/status-compact measure renderLifecycleStatus, which has ZERO non-test callers; the real 'aether status' renders via renderDashboard (cmd/status.go) -- this is the screen the owner looks at daily and it measures ~24% symbol-led against a 41.3% bar. (2) plan-stop measures renderPlanningStopVisual (cmd/planning_visuals.go:550), which has zero callers of any kind outside tests -- a fully orphaned renderer, so that screen is never drawn by anything. The other NINETEEN are genuinely wired and their density tests measure the function the command actually calls: build (renderBuildVisualWithDispatches, 2 sites), build-partial (renderBuildPartialCreditVisual, 1), continue-final + continue-midphase (renderContinueVisual, 3), seal (renderSealVisual, 1), discuss-questions + discuss-resolved (renderDiscussVisual, 2), spec (renderSpecCommandVisual, 1), what-next card (renderNextActionCardForPlatform, 2), and 8 of 9 planning screens. So 202.1 largely DID deliver; the damage is bounded to the status screen plus one dead renderer. | open |  | 2026-09-13T15:58:28.351Z |  |

````json
[
  {
    "id": 1,
    "kind": "unmet-truth",
    "phase": "193",
    "file": "cmd/caste_relevance.go",
    "line": null,
    "description": "Probe/auditor/gatekeeper still legitimately double-dispatch on build AND continue with no explicit Queen proposal (isAlwaysRequired computes the same required-caste independently on both flows); 193-02 only removed the watcher's implicit build-side dispatch. Flagged in 193-02-PLAN.md frontmatter as unclassified/not-auto-resolved; scoped to Phase 194 (moves the required-caste floor).",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-22T13:55:31.245Z",
    "resolved_at": "2026-08-23T16:04:14.666Z"
  },
  {
    "id": 2,
    "kind": "deviation",
    "phase": "196",
    "file": "pkg/codex/worker.go",
    "line": null,
    "description": "assembledPromptChars deleted outside the plan's files_modified; dead after the fallback removal",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-28T10:11:03.801Z",
    "resolved_at": null
  },
  {
    "id": 3,
    "kind": "stub",
    "phase": "197",
    "file": "cmd/codex_visuals.go",
    "line": null,
    "description": "The pause card's Handoff line still reads 'Colony handoff saved for later resumption' - repo-invented word 'colony' with no plain-English gloss. Found by TestNextActionCardSpeaksPlainEnglish; the plain-English check was scoped to the new next-action card because rewording the ten legacy cards is plans 197-04 and 197-06's declared scope.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-28T19:48:11.715Z",
    "resolved_at": "2026-08-29T00:00:00.000Z"
  },
  {
    "id": 4,
    "kind": "deviation",
    "phase": "197",
    "file": "cmd/next_action.go",
    "line": null,
    "description": "197-02 edited cmd/next_action.go, which is outside its declared files_modified, to change one word in the failed-phase recommendation ('Running it again' -> 'The next step is to retry it') so the pre-existing TestWorkflowSuggestionsFailedPhase assertion on the word 'retry' kept passing without weakening it.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-28T19:48:15.796Z",
    "resolved_at": null
  },
  {
    "id": 5,
    "kind": "deviation",
    "phase": "198",
    "file": "cmd/codex_visuals.go",
    "line": null,
    "description": "renderPlanVisual's confidence branch only type-asserts to map[string]interface{}; both runCodexPlanWithOptions and runCodexPlanFinalize always store confidence as a codexPlanConfidence struct, so the Confidence line never renders on the direct or chat path. Out of scope for 198-04 (prohibited from editing that file, owned by a same-wave plan); needs a dual-type fix mirroring planning_loop's existing struct/map switch.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-29T17:49:53.592Z",
    "resolved_at": "2026-08-29T18:28:52.436Z"
  },
  {
    "id": 6,
    "kind": "deviation",
    "phase": "198",
    "file": "cmd/codex_visuals.go",
    "line": null,
    "description": "renderPlanVisual never reads result[\"research_warning\"] (which renderResearchFailedWarning's own doc comment says exists \"so the omission is durable and visible\" when a phase was planned without its research), result[\"research_failed_phases\"] (the phase IDs that fed that warning), or result[\"gaps\"] (a completed plan's own unresolved gaps). Found and allow-listed, not fixed, in 198-09-PLAN.md Task 1 (cmd/testdata/rendered_field_allowlist.json) because cmd/codex_visuals.go was owned this wave by sibling plan 198-08.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-29T19:15:56.000Z",
    "resolved_at": "2026-08-29T20:05:27.000Z"
  },
  {
    "id": 7,
    "kind": "deviation",
    "phase": "198",
    "file": "cmd/codex_plan_finalize.go",
    "line": null,
    "description": "runCodexPlanFinalize never calls closeLifecycleRun (unlike continue-finalize/completeSealRuntime), so the plan finalizer's own suggested result[\"next\"] command never folds into the unified next-action envelope renderLifecycleClosing reads back -- the closing card instead independently resolves a next step from live colony state, which usually matches but is not guaranteed to. Found and allow-listed, not fixed, in 198-09-PLAN.md Task 1 (cmd/testdata/rendered_field_allowlist.json, \"plan completed\" and \"plan mid-loop\") because cmd/codex_plan_finalize.go was outside that plan's declared files.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-29T19:15:56.000Z",
    "resolved_at": "2026-08-29T20:05:27.000Z"
  },
  {
    "id": 8,
    "kind": "deviation",
    "phase": "198.2",
    "file": "cmd/codex_continue_finalize.go",
    "line": null,
    "description": "externalContinueReviewReport does not append a 'review wave skipped' narration step when no reviewers were dispatched, unlike the direct lane's runCodexContinueReview -- discovered by 198.2-05's dual-lane outcome-text parity test, out of that plan's scope to fix",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-30T14:49:33.437Z",
    "resolved_at": null
  },
  {
    "id": 9,
    "kind": "deviation",
    "phase": "201",
    "file": "cmd/golden_workflow_test.go",
    "line": null,
    "description": "TestGoldenBuildVisualOutput/TestGoldenContinueVisualOutput: stale golden fixtures missing the 'Cost: not known...' line; pre-existing, last touched by Phase 200, discovered during 201-15 full-suite run",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-10T19:49:27.735Z",
    "resolved_at": null
  },
  {
    "id": 10,
    "kind": "deviation",
    "phase": "201",
    "file": "cmd/phase199_gate_receipt_test.go",
    "line": null,
    "description": "TestPhase199GateReceipt fails on pre-existing untracked .gsd/ directory present before 201-15 started",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-10T19:49:27.844Z",
    "resolved_at": null
  },
  {
    "id": 11,
    "kind": "deviation",
    "phase": "201",
    "file": "cmd/queen_judgement_test.go",
    "line": null,
    "description": "TestQueenChoiceReachesTheDispatchList / TestNoWorkerWithoutStatedReason: a Queen-requested Measurer with a stated reason is dropped before spawn; pre-existing, discovered during 201-15, out of scope (queen_judgement.go not a declared file)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-10T19:49:27.959Z",
    "resolved_at": null
  },
  {
    "id": 12,
    "kind": "unrun-verify",
    "phase": "202",
    "file": "cmd/ (~15 tests, see 202-15-SUMMARY.md Issues Encountered)",
    "line": null,
    "description": "Full unscoped 'go test ./cmd -count=1' could not be confirmed clean: pre-existing failures unrelated to plan 202-15 (TestNextActionNeverHardcoded, TestPhase199GateReceipt, TestQueenChoiceReachesTheDispatchList, TestNoWorkerWithoutStatedReason, TestCurrentVocabulary199, TestHumanFacingOutputGoesThroughWriteVisualOutput, TestGoldenBuildVisualOutput, TestGoldenContinueVisualOutput, TestAuditCatalogGolden, TestBuildStartLegacyHelpersRetired200, TestGoSourceHintsMatchCobraContracts, TestCompletionPacketSchemaMatchesStructs, TestPlanningAdversarial200, TestSkillManifestReadEmpty, TestSkillManifestReadFromHub, TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount, TestFailedCheckSendsExactlyOneBuilderFixAttempt, TestFixAttemptIsCountedSeparately, TestFixAttemptNeverOverwritesTheFirstResult, TestNoSecondAutomaticFixAttempt) plus a documented ~20min machine-specific suite ceiling; every test touching a file this plan changed passes, including under -race.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-11T15:35:47.524Z",
    "resolved_at": null
  },
  {
    "id": 13,
    "kind": "deviation",
    "phase": "202.1",
    "file": "cmd/watch_live.go",
    "line": 425,
    "description": "Pre-existing (unrelated to this plan) TestHumanFacingOutputGoesThroughWriteVisualOutput failure: runColonyLiveRefreshLoop writes directly to stdout/stderr, bypassing writeVisualOutput",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-12T17:42:11.287Z",
    "resolved_at": null
  },
  {
    "id": 14,
    "kind": "deviation",
    "phase": "203",
    "file": "pkg/codex/worker.go",
    "line": null,
    "description": "203-02 added a one-line exported helper in pkg/codex/worker.go, outside its declared files_modified, so cmd/recruitment_dispatch.go could reuse the existing tested process-group termination logic instead of writing a second copy. Deliberate and documented in 203-02-SUMMARY.md Deviations; the alternative was duplicating kill-the-whole-process-tree logic in new code.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T00:07:16.386Z",
    "resolved_at": null
  },
  {
    "id": 15,
    "kind": "deviation",
    "phase": "203",
    "file": "cmd/exchange.go",
    "line": null,
    "description": "203-05 edited cmd/exchange.go, cmd/exchange_import_sanitize_test.go, cmd/hook_cmds.go and cmd/signal_housekeeping.go, none of which were in its declared files_modified. cmd/exchange.go was required: the real 'aether import pheromones' path never stamped provenance, so the plan's own quarantine must_have would have been decoration without it. The other three carried the strength-floor constant fix. All documented in 203-05-SUMMARY.md Deviations; the plan's files_modified header was incomplete rather than the executor overreaching.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T00:07:16.488Z",
    "resolved_at": null
  },
  {
    "id": 16,
    "kind": "unrun-verify",
    "phase": "203",
    "file": "cmd/testing_main_test.go",
    "line": null,
    "description": "A full 'go test ./cmd -count=1' run can silently NOT execute a test while still reporting a lane result. Observed 2026-09-13: TestPlatformParityGolden appeared only in lane parallel-046's own 'missing executed tests' accounting line and never ran, so the wave-2 gate reported no regression while 203-02's new 'aether recruit' command had in fact broken the parity golden. The suite already detects and prints this condition; nothing treats it as a failure, so a reader comparing FAIL lines against a known-red list gets a false all-clear. The parity break itself is fixed (22149a3e); this entry is about the accounting, not that test.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T10:00:01.055Z",
    "resolved_at": null
  },
  {
    "id": 17,
    "kind": "unrun-verify",
    "phase": "203",
    "file": ".aether/ts-host/test",
    "line": null,
    "description": "The TypeScript host test suite has 29 failing tests that pre-date Phase 203, in test/lifecycle.test.ts, test/go-bridge.test.ts, test/golden-workflow.test.ts and the classic command parity matrix. Verified 2026-09-13 by running the suite at c9ebe0b1 in a detached worktree and diffing failure names against the post-wave-3 run: 29 before, the same 29 after, zero new. Most assert a pending-planning boundary and now receive 'an approved specification is missing. Run aether spec' instead, so they look like a spec-gate change the TS lane never absorbed. Recorded so a future wave-3-style gate can diff against a known set instead of re-deriving it; NOT investigated or fixed here.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T10:57:40.741Z",
    "resolved_at": null
  },
  {
    "id": 18,
    "kind": "unmet-truth",
    "phase": "203",
    "file": "cmd/spawn.go",
    "line": null,
    "description": "SECURITY (fail-open authorization bypass, PRE-EXISTING since phase 173, surfaced 2026-09-13 by a commit security review of cmd/recruitment.go): the delegation depth cap is bypassable by self-assertion. spawnParentIsRoot matches an unauthenticated caller-supplied name against the fixed sentinel list spawnRootParentNames = {Queen, Prime-1, Swarm} (cmd/spawn.go:22) and grants depth 0 with DepthIsAuthoritative=true and no spawn-tree entry required. Both 'aether spawn-can-spawn --name Queen' and the new 'aether recruit --parent Queen' therefore pass the depth check regardless of the caller's real depth, defeating spawnMaxDelegationDepth (the runaway-spawn and cost control). Everything ELSE in that path is correctly fail-closed: depth is read from the recorded spawn tree via latestSpawnEntryByName, never from a flag, and validateRecruitmentIntent (cmd/recruitment_intent.go:153) explicitly refuses a non-sentinel parent whose depth is not authoritative, with the reason 'a parent's depth is never trusted from a self-declared claim'. The sentinel exemption is the single hole. 203-03 mirrored spawn-can-spawn's existing behaviour deliberately and documented it; this is inherited, not introduced. Routed live to plan 203-06, which owns cmd/spawn.go + cmd/recruitment.go + cmd/recruitment_admission.go this wave and whose objective is extending that chokepoint with dimensions it does not yet check. Not fixed here: 203-06 is mid-flight in those exact files and an orchestrator edit would collide.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T13:11:36.373Z",
    "resolved_at": null
  },
  {
    "id": 19,
    "kind": "unmet-truth",
    "phase": "203",
    "file": "cmd/testdata/orphan_allowlist.json",
    "line": null,
    "description": "EXPECTED-RED, OWNED BY 203-15: TestNoRegisteredSubcommandIsUnreferenced fails with 'aether recruit is registered but nothing calls it (searched: wrappers, menu specs, hooks, scripts)'. Five plans (203-02/03/04/06/07) built the recruitment command and no wrapper, menu spec, hook or script invokes it yet. This is real and is exactly the orphan failure CLAUDE.md names as the project's signature defect -- it is NOT silenced. Plan 203-15 is the closer: it rewrites .aether/workers.md, which today documents a spawning protocol in convincing detail for a mechanism no caste was ever granted, and replaces it with the real path plus a test. DELIBERATELY NOT ALLOWLISTED: testdata/orphan_allowlist.json is shrink-only against a frozen baseline (TestOrphanAllowlistOnlyShrinks, WIRE-01/D-11), so adding an entry would widen a ratchet to hide a true finding. Leaving it red means the test itself proves 203-15 did its job, and going green is the acceptance signal. If 203-15 lands and this is still red, the phase shipped an orphan. Recorded 2026-09-13 at the wave-4 gate; cmd/testdata/regression_snapshot.json was separately refreshed 390->389 commands, the honest net effect of 203-10 retiring two dead trophallaxis commands.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T13:36:11.546Z",
    "resolved_at": null
  },
  {
    "id": 20,
    "kind": "deviation",
    "phase": "203",
    "file": "cmd/swarm_scope_199_test.go",
    "line": null,
    "description": "203-12 Task 2 deleted SwarmPhase202Limitation, forcing an update to this pre-existing test's stale phase-202 assertions (fixed in the same commit, not deferred)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T15:12:22.623Z",
    "resolved_at": null
  },
  {
    "id": 21,
    "kind": "deviation",
    "phase": "203",
    "file": "cmd/codex_verify_advance.go",
    "line": null,
    "description": "Inherited from 203-10, still open after 203-12: recordTrophallaxisDecision's colony.LifecycleDecision is never threaded into runContinueAcceptVerifyAdvance (the single phase-level accept/verify/advance boundary, cmd/codex_verify_advance.go). 203-12 populated the CEC-07 credit join it actually owns (AgencyReceiptEvidence.ChangedDecision/EffectEvidence, cmd/agency_contract.go) from real trophallaxis+credit data, which is a different boundary from runContinueAcceptVerifyAdvance and fully satisfies this plan's own objective text. Whether a trophallaxis decision should ALSO reach the phase-level accept/verify/advance decision remains unresolved and cmd/codex_verify_advance.go is outside 203-12's declared file scope; a follow-up plan (203-14/203-15) should confirm intent.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T15:12:32.588Z",
    "resolved_at": null
  },
  {
    "id": 22,
    "kind": "unmet-truth",
    "phase": "202.1",
    "file": "cmd/status.go",
    "line": null,
    "description": "SEVERE / OWNER-REPORTED 2026-09-13: Phase 202.1's Classic-voice guarantee for 'aether status' is proved against a renderer the status command does not call. TestStatusScreenMeetsTheReferenceDensity and TestEveryVoicedScreenMeetsTheReferenceDensity measure renderLifecycleStatus (cmd/lifecycle_status_render.go) fed by classicVoiceStatusFixtureProjection, a hand-built projection. The real 'aether status' RunE calls renderDashboard (cmd/status.go:~60, via outputWorkflow), and renderLifecycleStatus's only non-test caller is cmd/compatibility_cmds.go:473. Measured on the live screen: 22 of 92 content lines symbol-led (~24%) against a reference bar of 41.3% -- the whole top block (Goal, Runtime, Signals, Progress, Focus, Instincts, Flags, Scope, Colony Mode, Depth, Granularity, Parallel) carries no leading symbol, while the memory/signals/worker sections below do. All five voice tests pass. The owner reported not seeing the Classic visuals they commissioned in 202.1; this is why. OPEN QUESTION not yet investigated: the same corpus registers build, continue, plan, discuss, spec, seal and the what-next card -- each needs the same check that its registered render function is the one its command actually calls. A green density test proves nothing about the screen if it measures a different function.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T15:56:48.392Z",
    "resolved_at": null
  },
  {
    "id": 23,
    "kind": "unmet-truth",
    "phase": "202.1",
    "file": "cmd/planning_visuals.go",
    "line": null,
    "description": "SCOPE ANSWERED 2026-09-13 (the open question from the status finding): all 21 screens registered in the Classic-voice corpus were traced from their registered render function to its production call sites. EXACTLY TWO are proved against code no command runs. (1) status-full/status-compact measure renderLifecycleStatus, which has ZERO non-test callers; the real 'aether status' renders via renderDashboard (cmd/status.go) -- this is the screen the owner looks at daily and it measures ~24% symbol-led against a 41.3% bar. (2) plan-stop measures renderPlanningStopVisual (cmd/planning_visuals.go:550), which has zero callers of any kind outside tests -- a fully orphaned renderer, so that screen is never drawn by anything. The other NINETEEN are genuinely wired and their density tests measure the function the command actually calls: build (renderBuildVisualWithDispatches, 2 sites), build-partial (renderBuildPartialCreditVisual, 1), continue-final + continue-midphase (renderContinueVisual, 3), seal (renderSealVisual, 1), discuss-questions + discuss-resolved (renderDiscussVisual, 2), spec (renderSpecCommandVisual, 1), what-next card (renderNextActionCardForPlatform, 2), and 8 of 9 planning screens. So 202.1 largely DID deliver; the damage is bounded to the status screen plus one dead renderer.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T15:58:28.351Z",
    "resolved_at": null
  }
]
````
| 14 | 202.1 | unmet-truth | pkg/codex/platform_dispatch.go |  | `workerProcessEnv` (pkg/codex/process_tracker.go) had NO caller, so AETHER_WORKER_NAME never reached a spawned worker. `aether hook-stop` therefore could not tell an Aether build worker from a person: it blocked worker Weld-32 mid-build and advised `aether pause`, the worker ran it, and a live CosmicDashboard Autopilot colony was paused mid-phase. Wired the env at the spawn site and exempted Aether-spawned workers from hook-stop. Proven by a REAL spawned subprocess reading back its own environment (TestSpawnedWorkerCarriesItsIdentityInTheEnvironment) rather than by testing the builder in isolation — an isolated builder test passed for the entire time the wiring was missing. | fixed |  | 2026-09-12T21:30:00.000Z | 2026-09-12T21:30:00.000Z |
