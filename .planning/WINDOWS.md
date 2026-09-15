---
schema_version: 1
open_count: 32
waived_count: 0
fixed_count: 15
total_count: 47
last_updated: 2026-09-15T01:08:31.907Z
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
| 24 | 203 | deviation | .planning/REQUIREMENTS.md |  | BIO-08 and CEC-07 are ready (every declaring plan has a SUMMARY) but requirements.mark-complete returns not_found and writes nothing -- this repo's checkbox format (- [ ] **REQ-ID -- Title:**) doesn't match the tool's regex (- [ ] **REQ-ID**); re-run mark-complete after reconciling the format | fixed |  | 2026-09-13T17:25:11.044Z | 2026-09-14T00:28:48.863Z |
| 25 | 203 | unmet-truth | cmd/subcommand_reachability_ratchet_test.go |  | SEVERE — THE PHASE'S ACCEPTANCE SIGNAL IS GREEN FOR A FALSE REASON. Plan 203-15 turned TestNoRegisteredSubcommandIsUnreferenced green by adding workerDisciplineCallerFiles (commit 824be303), a FOURTH caller-evidence source admitting .aether/workers.md, so documenting 'aether recruit' there counts as a caller. Its justifying comment states: ".claude/agents/ant/*.md's own 'Read .aether/workers.md for {caste} discipline' line makes a command documented here exactly as genuinely executed as a command a wrapper doc tells the assistant to run." THAT LINE DOES NOT EXIST. Verified 2026-09-13: zero of the 27 files in .claude/agents/ant/ reference workers.md; zero across .claude/agents, .opencode/agents and .codex/agents in any form; and NO runtime code reads workers.md into a prompt or brief (every cmd/ reference is install/platform-sync/source-check distribution or a comment). So workers.md is a document nobody is instructed to read and nothing loads — precisely the 'a doc mention is not an execution' case D-02/D-06 excluded .aether/docs/command-playbooks for. The orphan is REAL and still open: 'aether recruit', built across five plans, has no caller. This is the exact defect the phase existed to eliminate, committed by the plan whose job was to eliminate it, and it is worse than the original red because a future reader sees a passing test. DECISION REQUIRED: either wire it for real (agent definitions, or inject workers.md into the assembled brief) or accept that the mechanism ships unreachable and restore the honest red. Everything else 203-15 did — deleting the false spawning protocol, the latency measurement, locking CLAUDE.md claims to tests — is independent and sound. | open |  | 2026-09-13T18:34:24.429Z |  |
| 26 | 203 | unmet-truth | cmd/subcommand_reachability_ratchet_test.go |  | RESOLVED 2026-09-13, same session as the finding above. The false acceptance signal is corrected and the orphan is genuinely closed. THREE changes, none of them a widened allowlist: (1) renderRecruitmentInvitation (cmd/codex_build.go) writes the recruit instruction into EVERY dispatched worker's composed brief — the text that lands in the worker's own prompt, which is execution, proven by TestEveryDispatchedWorkerIsToldHowToAskForHelp across builder/watcher/scout and by TestTheRecruitInstructionHasOneSource (exactly one emitter, so the lanes cannot drift). (2) The instruction was added to the Claude AND OpenCode agent definitions for aether-builder/watcher/scout, in fenced form so the documented-call extractor sees it; TestClaudeOpenCodeAgentContentParity and TestCrossPlatformAgentParity still pass. (3) workerDisciplineCallerFiles was repointed from .aether/workers.md (which no agent names and no runtime loads) to those three agent definitions, which ARE the worker's prompt, the same class as the wrapper docs callerWrapperCorpora already accepts; the false justifying comment is replaced with the corrected record. PROVEN ABLE TO FAIL: removing the asking_for_help block from the three Claude agent files turns TestNoRegisteredSubcommandIsUnreferenced red naming aether recruit, and restoring it turns it green — the signal now tracks the wiring rather than the checker's generosity. Also fixed .github/workflows/ci.yml line 100, which still named TestSpawnCanSpawnAcceptsDocumentedInvocation after 203-15 renamed it to TestRecruitAcceptsDocumentedInvocation; the executor was sandboxed out of .github/workflows and correctly escalated rather than forcing it. TestWiringGateStepRunsEveryWiringTest now passes. | open |  | 2026-09-13T19:35:51.149Z |  |
| 27 | 203 | deviation | cmd/recruitment_subtree.go |  | Wave 6 (plan 203-14) introduced a user-visible regression its own gate MISSED: renderGovernedSubtreeStatusSection called projectGovernedSubtree, which walks the whole spawn tree across every run, so the new inline family tree presented workers from FINISHED runs on the status screen as if they were live. Caught only by the phase-closing full suite via TestStatusPrefersCurrentRunWorkersOverStaleHistory. That test SILENTLY DID NOT RUN in the wave-5 and wave-6 gates (it appears only in a 'missing executed tests:' accounting line), which is why both gates reported clean. Bisected across four commits to pin wave 6 as the origin: passes at 508e5096 and f348fbdc, fails at dab02fbe. Fixed by scoping the status caller to the current run via SpawnTree.CurrentRun/EntriesForRun rather than changing projectGovernedSubtree, whose whole-history behaviour other callers legitimately want; no current run means no filter, so a colony with history but nothing running still shows what it has. Proven by removing the filter and watching the test name the stale worker. THE STANDING LESSON: this is the seventh silent skip observed on 2026-09-13, and the second time a skipped test was the one that mattered. A suite that can omit a test while reporting a lane result is not a gate; comparing FAIL lines against a known-red list gives a false all-clear for anything that never ran. | open |  | 2026-09-13T19:50:56.397Z |  |
| 28 | 203 | unmet-truth | cmd/recruitment_admission.go |  | CRITICAL, found at the phase-203 code-review gate (203-REVIEW.md CR-01). The host/autopilot recruitment lane skips four of BIO-02's five new admission dimensions. processClaims (.aether/ts-host/src/spawn-orchestrator.ts:108-116) calls 'aether spawn-can-spawn', which runs under origin spawnOriginSpawnCanSpawn; recruitmentAdmissionChecks (cmd/recruitment_admission.go:47-57) maps that origin to an EMPTY check list, so permission, path, cost and duplicate are never evaluated there. Only spawnOriginRecruit (the native 'aether recruit' command and the in-repo lane) gets all eight dimensions. Both spawn-orchestrator.ts's own header comment and 203-CLASSIC-SYNTHESIS.md:206 (acceptance test SYN-203-02) state that a host-lane and a native-lane recruitment against the same ledger state produce the SAME allow/deny answer. They do not. The test whose name most resembles a parity check, TestBothLanesUseOneReasonVocabulary (cmd/recruitment_lane_test.go:298-360), asserts the OPPOSITE -- it fails if that origin's check list is ever non-empty. 203-09-SUMMARY.md:51 records the divergence as deliberate, but it was never reconciled with the written acceptance criterion or the shipped source comment. Failure case: a read-only caste requesting a write workspace is refused with reason 'permission' via 'aether recruit' and admitted via autopilot. Fix is either to share the check set across both origins or to retract the parity sentence from the synthesis doc and the source comment. | open |  | 2026-09-13T22:01:07.596Z |  |
| 29 | 203 | unmet-truth | cmd/pheromone_approval.go |  | CRITICAL, found at the phase-203 code-review gate (203-REVIEW.md CR-03). BIO-08's headline claim -- eight influence actions on an append-only history with a recorded actor -- is false for three of the eight. approvePendingNote, editPendingNote and rejectPendingNote (cmd/pheromone_approval.go) never call appendInfluenceHistory and record no actor at all. The codebase's own comment on PendingSuggestion.Action (pkg/colony/colony.go:347-353) concedes it is 'a single scalar record, not that history itself'. TestEveryDeclaredActionIsReachable only checks that the function and flag exist -- never that performing an action leaves a history entry -- so it is a false certificate for exactly the property the requirement names. BIO-08 is currently marked ready. Fix: route all eight actions through appendInfluenceHistory with the acting identity, and strengthen the test to assert an entry appears. | open |  | 2026-09-13T22:01:07.803Z |  |
| 30 | 203 | unmet-truth | cmd/pheromone_mgmt.go | 417 | CRITICAL, found at the phase-203 code-review gate (203-REVIEW.md CR-04). Owner-only steering-note actions (revoke, appeal, pin, unpin) are authorized purely by a self-reported --actor flag that DEFAULTS to 'owner' (cmd/pheromone_mgmt.go:417) with no caller authentication whatsoever. Any process that can run the aether binary -- including an ordinary dispatched worker -- can revoke a permanent REDIRECT constraint, or arm/disarm outcome-weighted tuning, by simply omitting the flag. Same class as the still-open depth-cap self-assertion hole (entry 18): an authorization decision taken on an unauthenticated caller-supplied string. CLAUDE.md states 'A note the owner pinned in place is never moved by this automatic tuning' -- true of the tuning pass, but the pin itself is settable and clearable by anyone. | open |  | 2026-09-13T22:01:08.001Z |  |
| 31 | 203 | unmet-truth | cmd/codex_build.go | 4733 | CRITICAL and a PARTIAL CORRECTION TO ENTRY 26, found at the phase-203 code-review gate (203-REVIEW.md CR-05). Entry 26 is half right. The acceptance test TestNoRegisteredSubcommandIsUnreferenced genuinely stopped being a false positive -- the allowlist was not widened, and .claude/agents/ant/*.md really are loaded as worker system prompts. But its conclusion that 'the orphan is genuinely closed' is false for real, currently-used dispatch lanes. renderRecruitmentInvitation has exactly ONE production call site (cmd/codex_build.go:4733, in composeBuildManifestBrief), pinned to stay single by TestTheRecruitInstructionHasOneSource. The invitation therefore never reaches: (1) workers dispatched by executeCodexBuildDispatches, the native/direct lane that autopilot uses; (2) Codex-platform workers at all -- 0 of 27 .codex/agents/*.toml files mention recruit; (3) any worker dispatched during 'aether continue' (Watcher, Gatekeeper, Auditor, Probe), whose brief composer never calls it. No test exercises those call chains, so the gap is invisible to the current gate. CLAUDE.md's framing ('A helper working on a piece of the job can now ask the program for backup') therefore overstates the delivered scope. Decision required: wire the remaining lanes, or narrow the shipped claim to the one lane that works. | open |  | 2026-09-13T22:01:39.225Z |  |
| 32 | 203 | unrun-verify | cmd/testing_main_test.go |  | ROOT CAUSE FOUND for the repeated silent-skip problem (entries 12, 16, 27), and it is not flaky sharding -- it is the DEFAULT GO TEST TIMEOUT. Measured 2026-09-13 at the phase-203 closing gate: 'go test ./... -count=1' with no explicit -timeout reported a result after executing 1635 of 5299 tests. Lanes parallel-019 through parallel-048 executed ZERO tests, duration 0s. The cmd package's own full-suite controller (resolveFullSuiteCeilings, cmd/testing_main_test.go:521-531) reads the outer go tool's -test.timeout and deliberately stops ORDERLY just before the tool would SIGQUIT, printing complete per-lane accounting -- so a truncated run looks exactly like a finished one to any reader who greps FAIL lines. The cmd suite needs ~21 minutes on this machine; go test's default per-package timeout is 10 minutes. EVERY unqualified 'go test ./...' gate run in this repo's history has therefore been checking roughly a third of the suite. The FULL-SUITE headline line already reports this truthfully ('FULL-SUITE FAIL discovered=5299 executed=1635') -- nothing was hidden, it was just never read. THE RULE: this repo's suite MUST be run as 'go test ./... -count=1 -timeout 90m', and any gate must assert discovered==executed on the FULL-SUITE headline before trusting a FAIL list. Re-run with -timeout 90m: 5299/5299 executed, 25 distinct failures. | fixed |  | 2026-09-13T22:01:39.405Z | 2026-09-14T00:24:50.915Z |
| 33 | 203 | deviation | cmd/status.go |  | FIXED at the phase-203 closing gate: four failures new since the phase base (1617ef3c), established by running the 25 full-suite failures against a detached worktree at that commit -- 16 of the 25 were already red, 9 were not. (1) TestStatusPausedColonyIgnoresStaleSpawnTreeWorkers: 203-14's family tree was added to the status screen with no liveness gate, so a PAUSED colony showed a previous session's worker as live; commit 33331f9c had scoped it to the current run but by design applies no filter when there is no current run, which is exactly the paused case. Fixed in 91dcb741 by extracting statusLiveSpawnView(state) -- the rule the Active Workers list and the JSON envelope already shared, previously duplicated inline twice -- and gating the family tree on it. (2) TestAetherCorpusCatchesAnUnregisteredFlag: 203-15 replaced workers.md's spawn-can-spawn invocation with recruit, leaving the negative corpus test hunting a call the live corpus no longer holds; it refused to pass vacuously, which is what it was built to do. Repointed at recruit with a --reason-omitting fixture in 5d870684. (3) TestDeliberatelyDroppedDisplayChoicesStayDropped: c7b5d998 (203-14) put the retired phrase 'second terminal window' into a comment that quoted it in order to forbid it; the ratchet is a substring scan and cannot tell. Comment reworded, ratchet untouched, in da98f9a8. (4) Four unrelated failures (TestSkillIndexReadEmpty, TestSkillIsUserCreated, TestSkillIsUserCreatedShipped, TestVisualsDumpExportsCasteIdentityContract) all traced to one pre-existing line: setupSanitizationTest used os.Setenv('AETHER_ROOT', tmpDir), which outlives t.TempDir's cleanup, so every later test in the same lane resolved the repository root to a deleted path. Fixed with t.Setenv in d045056b. All four proven by before/after: red in the full run at 702aee63, green after, with each name confirmed as actually executed rather than skipped. | fixed |  | 2026-09-13T22:01:39.595Z | 2026-09-13T22:01:47.595Z |
| 34 | 203 | unmet-truth | cmd/suggest_approve.go |  | Surfaced while fixing CR-03 (the accept/edit/reject history gap), pre-existing and deliberately left untouched. 'aether suggest-approve --dismiss-all' bulk-dismisses suggested steering notes while bypassing the Action/ActionAt stamp AND the influence history entirely -- so a bulk dismissal leaves no durable record and names no actor, even now that the three single-note actions do. It sits outside the eleven declared influence actions' flag surface, which is why the strengthened TestEveryDeclaredActionAppendsHistoryWithAnActor does not cover it: the table drives pheromoneInfluenceActionNames(), and --dismiss-all is not in that list. Either route it through appendInfluenceHistory like every other decision, or declare it as an action so the table covers it. Until then BIO-08's append-only guarantee holds for every action the table names and not for this one. | open |  | 2026-09-13T22:42:32.987Z |  |
| 35 | 203 | unmet-truth | cmd/codex_continue.go |  | Surfaced while closing CR-05 (wiring the recruit invitation into every dispatch lane). TWO REVIEW CASTES GENUINELY CANNOT USE THE MECHANISM, and one that also cannot is told to anyway. Gatekeeper and Auditor are deliberately granted no Bash tool (codexContinueReviewSpecs' own doc comment, locked by TestReviewSpecsDoNotInstructBashlessCastes after a prior incident where telling a bashless caste to run 'aether review-ledger-write' blocked phase advancement). The CR-05 fix therefore excludes those two from the invitation by name -- correct, and following an existing tested rule rather than inventing scope. But CLAUDE.md's Biological Runtime section still says generically 'A helper working on a piece of the job can now ask the program for backup', which remains very slightly broader than delivered: those two helpers cannot, and never could reach any CLI mechanism. Separately and inconsistently, aether-scout's Claude and OpenCode definitions ALSO grant no Bash (Grep/Glob/WebSearch/WebFetch only), yet the pre-existing TestEveryDispatchedWorkerIsToldHowToAskForHelp requires Scout to receive the invitation exactly like Builder and Watcher, and the CR-05 fix mirrored that accepted content verbatim into .codex/agents/aether-scout.toml. So the same 'no Bash' fact excludes two castes and not a third. Either Scout's tool grant is wrong, or the Gatekeeper/Auditor exclusion is broader than it needs to be -- one rule should decide all three. Not adjudicated during the CR-05 fix because both the Scout grant and its test pre-date this phase. | open |  | 2026-09-13T22:44:36.143Z |  |
| 36 | 203 | unmet-truth | cmd/live_projection.go | 551 | PRE-EXISTING latent bug in the live-view resume path, found at the phase-203 closing gate and NOT introduced by it (git diff 1617ef3c..HEAD on cmd/live_projection.go shows the phase's only change to this file is additive Reason/Refusals fields plus gofmt realignment -- the elapsed/resume logic is untouched). ElapsedSeconds is recomputed from event timestamps ONLY when snapshot.Open is true (cmd/live_projection.go:551). A checkpoint taken mid-episode is therefore stamped with a live elapsed figure; when the run later closes, replayColonyLiveSnapshotResume carries that stale figure forward, while a full replay of the same events ends closed and leaves elapsed at 0. TestLiveProjectionResumesWithoutDoubleCounting asserts the two must be identical and correctly catches the divergence -- but only when the seeded events straddle a one-second boundary, so it is red perhaps one run in several. Observed 2026-09-14 with resumed ElapsedSeconds=1 vs full=0 and every other field byte-identical; passes in isolation. The underlying question is a behaviour decision for the owner, which is why this is recorded rather than patched: a FINISHED run arguably should report its total duration rather than 0, in which case the full-replay path is the one that is wrong, not the resume path. Fixing it by clearing the resumed value would make the test green while making the closed-episode screen less informative. TestSwarmCompatibilityWatchReportsActiveWorkers (active_count = 0, want 1) appeared in the same run, also passes in isolation, and reads the same projection -- likely the same family, not separately diagnosed. | open |  | 2026-09-14T00:24:50.809Z |  |
| 37 | 203 | deviation | .planning/REQUIREMENTS.md |  | FIXED, superseding entry 24. Requirement completion tracking had been silently broken repo-wide: gsd-tools' checkbox pattern is (-\\s*\\[)[ ](\\]\\s*\\*\\*<REQ-ID>\\*\\*) (lib/milestone.cjs:162), which needs the bold to close immediately after the ID, while this file wrote '- [ ] **BIO-02 -- Atomic admission:** ...' with the title inside the bold span. Every requirements mark-complete therefore returned not_found and wrote nothing, and phase.complete reported 'ROADMAP cites REQ-IDs not registered anywhere in REQUIREMENTS.md' for IDs that were plainly present. Reformatted all 65 requirement lines to '- [ ] **REQ-ID** -- Title: ...' -- a pure format change, verified content-identical to the previous file once ** markers are stripped, and no test anywhere pins the old shape (no cmd/*_test.go references REQUIREMENTS.md at all). Phase 203's ten IDs (SYNTH-05, CEC-07, BIO-01..08) are now genuinely ticked by the tool's own write path rather than by hand. The earlier hand-edit of BIO-07 recorded in 203-08-SUMMARY.md can now be done the canonical way. | fixed |  | 2026-09-14T00:28:42.019Z | 2026-09-14T00:28:53.780Z |
| 38 | 204 | unmet-truth | cmd/live_events.go |  | FIXED by 204-15: all nine previously writerless episode-ledger fields (evidence_ids, hard_gate_results, changed_decision_ids, usage, reported_cost_usd, interventions, episode_revision, acceptance_digest, evaluator_digest) now have real production writers on the build lane, the native check lane (runCodexContinue), and the delegate check lane (runCodexContinueFinalize, which previously recorded no episode at all). The episode ledger also joined cmd/memory_schema.go's field-level census as its seventh store, with a writer entry for all 21 of its json fields -- TestEveryMemoryStoreFieldHasALiveWriter now fails by name if a future field-writer entry is deleted, which is the guard that stops this window from reopening. Usage/ReportedCostUSD remain honestly absent on both check lanes (a documented, non-fabricated gap -- see LEARN-02 in REQUIREMENTS.md), which does not reopen this window since the original defect was zero production callers, not incomplete coverage of every field on every lane. | fixed |  | 2026-09-14T12:51:19.761Z | 2026-09-15T00:22:57.000Z |
| 39 | 204 | unmet-truth | cmd/shadow_cmds.go |  | FIXED by 204-12: aether shadow-declare and aether shadow-compare are now registered, reachable commands (cmd/improvement_cmds.go), alongside a new inspection surface, aether improve (default: read-only report; --declare/--compare route to the same mutating forms). The owner-facing exposure the original entry asked for is the /ant-improve wrapper triplet (.aether/commands/improve.yaml, .claude/commands/ant/improve.md, .opencode/commands/ant/improve.md), whose routing text names all three commands, satisfying TestNoRegisteredSubcommandIsUnreferenced without widening cmd/testdata/orphan_allowlist.json. | fixed |  | 2026-09-14T14:21:30.954Z | 2026-09-14T22:48:32.492Z |
| 40 | 204 | unmet-truth | cmd/shadow_cmds.go |  | NARROWED by 204-12, not fixed: shadowEvaluator's run function is now a real per-fixture classifier (shadowClassifyAgainstBank), no longer the always-pass placeholder -- 204-16 re-confirmed TestShadowGraderDistinguishesBeneficialFromHarmful/TestOverfitCandidateIsRefusedAtTheGate/TestEvaluatorDigestIsStableAndChanged all pass. This entry was incorrectly marked fixed by 204-12; reopened by 204-16 per D-11 because the original description's own second sentence is still true today: comparison covers this project's own settings/routing fixtures only and never a code-valued candidate (LEARN-06's own scoping). Remains open until a follow-on plan extends isolation and grading to code-valued candidates. | open | Reopened 2026-09-15 (204-16, D-11): grader is real, but code-valued candidates remain out of scope -- see LEARN-06 in REQUIREMENTS.md for the same caveat. | 2026-09-14T15:40:47.610Z |  |
| 41 | 204 | deviation | cmd/episode_ledger_test.go |  | FIXED by 204-13: runSwarmDestroy (cmd/swarm_cmd.go) and orchestrateRecovery (cmd/recovery_orchestrator.go, only when it owns the episode itself) now both open and close a durable episode via emitColonyLiveEpisodeStarted/Ended, proven by a FAILS-WHEN-UNWIRED mutation on each. TestEveryLifecycleLaneWritesADurableOutcome's t.Skipf branch is now a named t.Fatalf -- all six declared lifecycle lanes pass with zero skip lines. | fixed |  | 2026-09-14T15:40:51.528Z | 2026-09-15T00:22:57.000Z |
| 42 | 204 | unmet-truth | cmd/testdata/fixture-bank/v1/bank.json |  | NARROWED by 204-14, still open: 10 previously-unguarded fixtures gained a real, passing, incident-specific guard test (guarded count 7 -> 17 of 47 total at the time; 22 of 52 after the ledger closures in 204-12 and 204-16 seeded five more fixtures, each guarded on arrival). seedBankUnguardedFloor lowered from 40 to 30 -- the exact real unguarded count, confirmed in this session against cmd/eval_gates.go -- and is now a two-sided ratchet (TestSeedBankUnguardedFloorIsTheRealCount) that fails if the recorded floor is ever raised above the real count, not only if the real count exceeds it. Remains open: 30 of 52 confirmed incidents still carry no guarding test. Close by naming and confirming a real guard test for each remaining fixture's own confirmed incident, shrinking the recorded floor as each is added -- the floor may only shrink, never widen. | open | Narrowed 2026-09-15 (204-14/204-16): floor is 30, not 40 -- see LEARN-05 in REQUIREMENTS.md for the same number. | 2026-09-14T15:40:55.056Z |  |
| 43 | 204 | deviation | cmd/memory_schema.go |  | Owner decision 2026-09-14 (204-03 Task 4, 'mixture' reply accepting the recommendation table verbatim): instinct.related_instincts is retired (memoryStoreFieldRetired, reason: the only reader, pkg/graph, is doubly orphaned and this phase's own ruling (e) forbids citing it as justification for new work) -- no stored record touched, both production writers stopped setting it. midden.acknowledge_reason, learn.parent_id and pheromone.scope stay in memoryStoreFieldExceptions, recorded as knowingly empty (plausibly useful, but nothing in this phase's planned work needs them yet). Recorded here for traceability; closes only if a future phase names a concrete consumer for one of the three empty fields. | open |  | 2026-09-14T15:40:59.645Z |  |
| 44 | 204 | unmet-truth | cmd/learning_validator.go |  | REOPENED by 204-REVIEW.md CR-01 (2026-09-15): 204-16's promoteHelpfulHypotheses (called from runPhaseEndConsolidation alongside the improvement pass) cannot promote anything in production. Root cause is an ID-space mismatch confirmed by grep: recordGuidanceApplicationState's only non-test callers key every GuidanceApplications record by an Instinct's own ID (cmd/instinct_application.go:69,75), never by a learn.Entry's ID -- learn.Entry.ParentID is hypothesis-revision lineage only, and no production writer links a learn.Entry to the Instinct (if any) it came from. Independently, no non-test call site anywhere in cmd/ ever writes the guidanceApplicationStateHelpful/Neutral/Harmful terminal states at all -- the "helpful" signal QUEEN promotion actually reads lives in a different list (recruitmentCreditFile.Entries) that is never bridged into the guidance-application vocabulary. learningEntryHasHelpfulApplication therefore returns false for every real hypothesis, forever. The test suite previously passed only because cmd/learning_validator_test.go hand-typed a learn entry's ID and reused it as the GuidanceApplications guidanceID -- a shape the runtime cannot produce (a false certificate per CLAUDE.md's own Definition of Done). Fixed at the test layer (learning_validator_test.go now proves the honest current limitation via TestHypothesisPromotionNeverCrossesTheIdentifierGap, observed FAILING when the gating logic is mutated to always match) and documented at the production layer (learning_validator.go's package and function doc comments now state the limitation and its two root causes in full). No fabricated identifier bridge was introduced. Closes only when a real production writer either links a learn.Entry to its source Instinct or writes GuidanceApplications under learn.Entry.ID from a real call site, AND a real writer of the Helpful/Neutral/Harmful states exists. | open | ID-space mismatch between learn.Entry and GuidanceApplications, plus no production writer of the Helpful/Neutral/Harmful states at all; no honest identifier bridge exists within reach of the 204-REVIEW.md fix pass (2026-09-15) | 2026-09-14T15:41:03.868Z |  |
| 45 | 204 | unmet-truth | cmd/source_proposal.go |  | FIXED, both halves: (1) 204-13 declared episodeInterventionKind, a closed, source-derived intervention-kind vocabulary with three real production writers, so collectPreventableInterventions now classifies against a curated set instead of treating every free-form string as its own category. (2) 204-16 gave proposeSourceImprovement its first real production caller, triggerRepeatedInterventionProposal (cmd/improvement_pass.go): when the same declared intervention kind recurs on 3+ distinct episodes, it proposes a source change naming the real episodes, on an isolated branch, with sourceProposalReachabilityEntryPoints extended in the same change so TestSourceProposalCannotMergePublishOrDeploy's call-graph walk covers the new entry point too. | fixed |  | 2026-09-14T15:41:07.891Z | 2026-09-15T00:22:57.000Z |
| 46 | 202.1 | unmet-truth | pkg/codex/platform_dispatch.go |  | workerProcessEnv (pkg/codex/process_tracker.go) had NO caller, so AETHER_WORKER_NAME never reached a spawned worker. aether hook-stop therefore could not tell an Aether build worker from a person: it blocked worker Weld-32 mid-build and advised aether pause, the worker ran it, and a live CosmicDashboard Autopilot colony was paused mid-phase. Wired the env at the spawn site and exempted Aether-spawned workers from hook-stop. Proven by a REAL spawned subprocess reading back its own environment (TestSpawnedWorkerCarriesItsIdentityInTheEnvironment) rather than by testing the builder in isolation -- an isolated builder test passed for the entire time the wiring was missing. Migrated 2026-09-14 by plan 204-11 from a stray, out-of-band duplicate row (originally id 14, colliding with the real phase-203 entry 14) that had drifted below the JSON ledger block; original recorded/resolved timestamps were 2026-09-12T21:30:00.000Z. | fixed |  | 2026-09-14T15:42:11.225Z | 2026-09-14T15:42:13.430Z |
| 47 | 204 | unmet-truth | cmd/swarm_cmd.go |  | PRE-EXISTING latent collision in swarm worker naming, found at the Phase 204 gap-closure wave-3 gate (2026-09-15) and NOT introduced by it: no gap-closure plan touches swarm naming. deterministicAntName (cmd/codex_visuals.go) derives a worker name as prefix[hash mod len(prefixes)] plus a number from hash mod 99, seeded by the colony root path, caste and target; buildSwarmManifest (cmd/swarm_cmd.go, the duplicate dispatch name check near line 1256) then refuses the whole manifest when two dispatches land on the same name instead of de-duplicating. With five or six workers drawn from roughly six prefixes times 99 numbers, any two collide about one run in fifty to a hundred, and because the root path is part of the seed, a given colony can be stuck colliding on a given target every time. Observed as TestSwarmFinalizeRecordsExternalTaskResults/timeout failing with duplicate dispatch name Guard-95 once in a full-suite run at 05b0d453; the same test passed in the previous gate at 2eae06e3 and passed three of three re-runs in isolation. Close by making the manifest builder append a disambiguating suffix on collision (or fold the caste index into the seed) with a test that forces two workers onto one name and asserts both are dispatched under distinct names. | open |  | 2026-09-15T01:08:31.907Z |  |

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
  },
  {
    "id": 24,
    "kind": "deviation",
    "phase": "203",
    "file": ".planning/REQUIREMENTS.md",
    "line": null,
    "description": "BIO-08 and CEC-07 are ready (every declaring plan has a SUMMARY) but requirements.mark-complete returns not_found and writes nothing -- this repo's checkbox format (- [ ] **REQ-ID -- Title:**) doesn't match the tool's regex (- [ ] **REQ-ID**); re-run mark-complete after reconciling the format",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-09-13T17:25:11.044Z",
    "resolved_at": "2026-09-14T00:28:48.863Z"
  },
  {
    "id": 25,
    "kind": "unmet-truth",
    "phase": "203",
    "file": "cmd/subcommand_reachability_ratchet_test.go",
    "line": null,
    "description": "SEVERE — THE PHASE'S ACCEPTANCE SIGNAL IS GREEN FOR A FALSE REASON. Plan 203-15 turned TestNoRegisteredSubcommandIsUnreferenced green by adding workerDisciplineCallerFiles (commit 824be303), a FOURTH caller-evidence source admitting .aether/workers.md, so documenting 'aether recruit' there counts as a caller. Its justifying comment states: \".claude/agents/ant/*.md's own 'Read .aether/workers.md for {caste} discipline' line makes a command documented here exactly as genuinely executed as a command a wrapper doc tells the assistant to run.\" THAT LINE DOES NOT EXIST. Verified 2026-09-13: zero of the 27 files in .claude/agents/ant/ reference workers.md; zero across .claude/agents, .opencode/agents and .codex/agents in any form; and NO runtime code reads workers.md into a prompt or brief (every cmd/ reference is install/platform-sync/source-check distribution or a comment). So workers.md is a document nobody is instructed to read and nothing loads — precisely the 'a doc mention is not an execution' case D-02/D-06 excluded .aether/docs/command-playbooks for. The orphan is REAL and still open: 'aether recruit', built across five plans, has no caller. This is the exact defect the phase existed to eliminate, committed by the plan whose job was to eliminate it, and it is worse than the original red because a future reader sees a passing test. DECISION REQUIRED: either wire it for real (agent definitions, or inject workers.md into the assembled brief) or accept that the mechanism ships unreachable and restore the honest red. Everything else 203-15 did — deleting the false spawning protocol, the latency measurement, locking CLAUDE.md claims to tests — is independent and sound.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T18:34:24.429Z",
    "resolved_at": null
  },
  {
    "id": 26,
    "kind": "unmet-truth",
    "phase": "203",
    "file": "cmd/subcommand_reachability_ratchet_test.go",
    "line": null,
    "description": "RESOLVED 2026-09-13, same session as the finding above. The false acceptance signal is corrected and the orphan is genuinely closed. THREE changes, none of them a widened allowlist: (1) renderRecruitmentInvitation (cmd/codex_build.go) writes the recruit instruction into EVERY dispatched worker's composed brief — the text that lands in the worker's own prompt, which is execution, proven by TestEveryDispatchedWorkerIsToldHowToAskForHelp across builder/watcher/scout and by TestTheRecruitInstructionHasOneSource (exactly one emitter, so the lanes cannot drift). (2) The instruction was added to the Claude AND OpenCode agent definitions for aether-builder/watcher/scout, in fenced form so the documented-call extractor sees it; TestClaudeOpenCodeAgentContentParity and TestCrossPlatformAgentParity still pass. (3) workerDisciplineCallerFiles was repointed from .aether/workers.md (which no agent names and no runtime loads) to those three agent definitions, which ARE the worker's prompt, the same class as the wrapper docs callerWrapperCorpora already accepts; the false justifying comment is replaced with the corrected record. PROVEN ABLE TO FAIL: removing the asking_for_help block from the three Claude agent files turns TestNoRegisteredSubcommandIsUnreferenced red naming aether recruit, and restoring it turns it green — the signal now tracks the wiring rather than the checker's generosity. Also fixed .github/workflows/ci.yml line 100, which still named TestSpawnCanSpawnAcceptsDocumentedInvocation after 203-15 renamed it to TestRecruitAcceptsDocumentedInvocation; the executor was sandboxed out of .github/workflows and correctly escalated rather than forcing it. TestWiringGateStepRunsEveryWiringTest now passes.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T19:35:51.149Z",
    "resolved_at": null
  },
  {
    "id": 27,
    "kind": "deviation",
    "phase": "203",
    "file": "cmd/recruitment_subtree.go",
    "line": null,
    "description": "Wave 6 (plan 203-14) introduced a user-visible regression its own gate MISSED: renderGovernedSubtreeStatusSection called projectGovernedSubtree, which walks the whole spawn tree across every run, so the new inline family tree presented workers from FINISHED runs on the status screen as if they were live. Caught only by the phase-closing full suite via TestStatusPrefersCurrentRunWorkersOverStaleHistory. That test SILENTLY DID NOT RUN in the wave-5 and wave-6 gates (it appears only in a 'missing executed tests:' accounting line), which is why both gates reported clean. Bisected across four commits to pin wave 6 as the origin: passes at 508e5096 and f348fbdc, fails at dab02fbe. Fixed by scoping the status caller to the current run via SpawnTree.CurrentRun/EntriesForRun rather than changing projectGovernedSubtree, whose whole-history behaviour other callers legitimately want; no current run means no filter, so a colony with history but nothing running still shows what it has. Proven by removing the filter and watching the test name the stale worker. THE STANDING LESSON: this is the seventh silent skip observed on 2026-09-13, and the second time a skipped test was the one that mattered. A suite that can omit a test while reporting a lane result is not a gate; comparing FAIL lines against a known-red list gives a false all-clear for anything that never ran.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T19:50:56.397Z",
    "resolved_at": null
  },
  {
    "id": 28,
    "kind": "unmet-truth",
    "phase": "203",
    "file": "cmd/recruitment_admission.go",
    "line": null,
    "description": "CRITICAL, found at the phase-203 code-review gate (203-REVIEW.md CR-01). The host/autopilot recruitment lane skips four of BIO-02's five new admission dimensions. processClaims (.aether/ts-host/src/spawn-orchestrator.ts:108-116) calls 'aether spawn-can-spawn', which runs under origin spawnOriginSpawnCanSpawn; recruitmentAdmissionChecks (cmd/recruitment_admission.go:47-57) maps that origin to an EMPTY check list, so permission, path, cost and duplicate are never evaluated there. Only spawnOriginRecruit (the native 'aether recruit' command and the in-repo lane) gets all eight dimensions. Both spawn-orchestrator.ts's own header comment and 203-CLASSIC-SYNTHESIS.md:206 (acceptance test SYN-203-02) state that a host-lane and a native-lane recruitment against the same ledger state produce the SAME allow/deny answer. They do not. The test whose name most resembles a parity check, TestBothLanesUseOneReasonVocabulary (cmd/recruitment_lane_test.go:298-360), asserts the OPPOSITE -- it fails if that origin's check list is ever non-empty. 203-09-SUMMARY.md:51 records the divergence as deliberate, but it was never reconciled with the written acceptance criterion or the shipped source comment. Failure case: a read-only caste requesting a write workspace is refused with reason 'permission' via 'aether recruit' and admitted via autopilot. Fix is either to share the check set across both origins or to retract the parity sentence from the synthesis doc and the source comment.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T22:01:07.596Z",
    "resolved_at": null
  },
  {
    "id": 29,
    "kind": "unmet-truth",
    "phase": "203",
    "file": "cmd/pheromone_approval.go",
    "line": null,
    "description": "CRITICAL, found at the phase-203 code-review gate (203-REVIEW.md CR-03). BIO-08's headline claim -- eight influence actions on an append-only history with a recorded actor -- is false for three of the eight. approvePendingNote, editPendingNote and rejectPendingNote (cmd/pheromone_approval.go) never call appendInfluenceHistory and record no actor at all. The codebase's own comment on PendingSuggestion.Action (pkg/colony/colony.go:347-353) concedes it is 'a single scalar record, not that history itself'. TestEveryDeclaredActionIsReachable only checks that the function and flag exist -- never that performing an action leaves a history entry -- so it is a false certificate for exactly the property the requirement names. BIO-08 is currently marked ready. Fix: route all eight actions through appendInfluenceHistory with the acting identity, and strengthen the test to assert an entry appears.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T22:01:07.803Z",
    "resolved_at": null
  },
  {
    "id": 30,
    "kind": "unmet-truth",
    "phase": "203",
    "file": "cmd/pheromone_mgmt.go",
    "line": 417,
    "description": "CRITICAL, found at the phase-203 code-review gate (203-REVIEW.md CR-04). Owner-only steering-note actions (revoke, appeal, pin, unpin) are authorized purely by a self-reported --actor flag that DEFAULTS to 'owner' (cmd/pheromone_mgmt.go:417) with no caller authentication whatsoever. Any process that can run the aether binary -- including an ordinary dispatched worker -- can revoke a permanent REDIRECT constraint, or arm/disarm outcome-weighted tuning, by simply omitting the flag. Same class as the still-open depth-cap self-assertion hole (entry 18): an authorization decision taken on an unauthenticated caller-supplied string. CLAUDE.md states 'A note the owner pinned in place is never moved by this automatic tuning' -- true of the tuning pass, but the pin itself is settable and clearable by anyone.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T22:01:08.001Z",
    "resolved_at": null
  },
  {
    "id": 31,
    "kind": "unmet-truth",
    "phase": "203",
    "file": "cmd/codex_build.go",
    "line": 4733,
    "description": "CRITICAL and a PARTIAL CORRECTION TO ENTRY 26, found at the phase-203 code-review gate (203-REVIEW.md CR-05). Entry 26 is half right. The acceptance test TestNoRegisteredSubcommandIsUnreferenced genuinely stopped being a false positive -- the allowlist was not widened, and .claude/agents/ant/*.md really are loaded as worker system prompts. But its conclusion that 'the orphan is genuinely closed' is false for real, currently-used dispatch lanes. renderRecruitmentInvitation has exactly ONE production call site (cmd/codex_build.go:4733, in composeBuildManifestBrief), pinned to stay single by TestTheRecruitInstructionHasOneSource. The invitation therefore never reaches: (1) workers dispatched by executeCodexBuildDispatches, the native/direct lane that autopilot uses; (2) Codex-platform workers at all -- 0 of 27 .codex/agents/*.toml files mention recruit; (3) any worker dispatched during 'aether continue' (Watcher, Gatekeeper, Auditor, Probe), whose brief composer never calls it. No test exercises those call chains, so the gap is invisible to the current gate. CLAUDE.md's framing ('A helper working on a piece of the job can now ask the program for backup') therefore overstates the delivered scope. Decision required: wire the remaining lanes, or narrow the shipped claim to the one lane that works.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T22:01:39.225Z",
    "resolved_at": null
  },
  {
    "id": 32,
    "kind": "unrun-verify",
    "phase": "203",
    "file": "cmd/testing_main_test.go",
    "line": null,
    "description": "ROOT CAUSE FOUND for the repeated silent-skip problem (entries 12, 16, 27), and it is not flaky sharding -- it is the DEFAULT GO TEST TIMEOUT. Measured 2026-09-13 at the phase-203 closing gate: 'go test ./... -count=1' with no explicit -timeout reported a result after executing 1635 of 5299 tests. Lanes parallel-019 through parallel-048 executed ZERO tests, duration 0s. The cmd package's own full-suite controller (resolveFullSuiteCeilings, cmd/testing_main_test.go:521-531) reads the outer go tool's -test.timeout and deliberately stops ORDERLY just before the tool would SIGQUIT, printing complete per-lane accounting -- so a truncated run looks exactly like a finished one to any reader who greps FAIL lines. The cmd suite needs ~21 minutes on this machine; go test's default per-package timeout is 10 minutes. EVERY unqualified 'go test ./...' gate run in this repo's history has therefore been checking roughly a third of the suite. The FULL-SUITE headline line already reports this truthfully ('FULL-SUITE FAIL discovered=5299 executed=1635') -- nothing was hidden, it was just never read. THE RULE: this repo's suite MUST be run as 'go test ./... -count=1 -timeout 90m', and any gate must assert discovered==executed on the FULL-SUITE headline before trusting a FAIL list. Re-run with -timeout 90m: 5299/5299 executed, 25 distinct failures.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-09-13T22:01:39.405Z",
    "resolved_at": "2026-09-14T00:24:50.915Z"
  },
  {
    "id": 33,
    "kind": "deviation",
    "phase": "203",
    "file": "cmd/status.go",
    "line": null,
    "description": "FIXED at the phase-203 closing gate: four failures new since the phase base (1617ef3c), established by running the 25 full-suite failures against a detached worktree at that commit -- 16 of the 25 were already red, 9 were not. (1) TestStatusPausedColonyIgnoresStaleSpawnTreeWorkers: 203-14's family tree was added to the status screen with no liveness gate, so a PAUSED colony showed a previous session's worker as live; commit 33331f9c had scoped it to the current run but by design applies no filter when there is no current run, which is exactly the paused case. Fixed in 91dcb741 by extracting statusLiveSpawnView(state) -- the rule the Active Workers list and the JSON envelope already shared, previously duplicated inline twice -- and gating the family tree on it. (2) TestAetherCorpusCatchesAnUnregisteredFlag: 203-15 replaced workers.md's spawn-can-spawn invocation with recruit, leaving the negative corpus test hunting a call the live corpus no longer holds; it refused to pass vacuously, which is what it was built to do. Repointed at recruit with a --reason-omitting fixture in 5d870684. (3) TestDeliberatelyDroppedDisplayChoicesStayDropped: c7b5d998 (203-14) put the retired phrase 'second terminal window' into a comment that quoted it in order to forbid it; the ratchet is a substring scan and cannot tell. Comment reworded, ratchet untouched, in da98f9a8. (4) Four unrelated failures (TestSkillIndexReadEmpty, TestSkillIsUserCreated, TestSkillIsUserCreatedShipped, TestVisualsDumpExportsCasteIdentityContract) all traced to one pre-existing line: setupSanitizationTest used os.Setenv('AETHER_ROOT', tmpDir), which outlives t.TempDir's cleanup, so every later test in the same lane resolved the repository root to a deleted path. Fixed with t.Setenv in d045056b. All four proven by before/after: red in the full run at 702aee63, green after, with each name confirmed as actually executed rather than skipped.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-09-13T22:01:39.595Z",
    "resolved_at": "2026-09-13T22:01:47.595Z"
  },
  {
    "id": 34,
    "kind": "unmet-truth",
    "phase": "203",
    "file": "cmd/suggest_approve.go",
    "line": null,
    "description": "Surfaced while fixing CR-03 (the accept/edit/reject history gap), pre-existing and deliberately left untouched. 'aether suggest-approve --dismiss-all' bulk-dismisses suggested steering notes while bypassing the Action/ActionAt stamp AND the influence history entirely -- so a bulk dismissal leaves no durable record and names no actor, even now that the three single-note actions do. It sits outside the eleven declared influence actions' flag surface, which is why the strengthened TestEveryDeclaredActionAppendsHistoryWithAnActor does not cover it: the table drives pheromoneInfluenceActionNames(), and --dismiss-all is not in that list. Either route it through appendInfluenceHistory like every other decision, or declare it as an action so the table covers it. Until then BIO-08's append-only guarantee holds for every action the table names and not for this one.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T22:42:32.987Z",
    "resolved_at": null
  },
  {
    "id": 35,
    "kind": "unmet-truth",
    "phase": "203",
    "file": "cmd/codex_continue.go",
    "line": null,
    "description": "Surfaced while closing CR-05 (wiring the recruit invitation into every dispatch lane). TWO REVIEW CASTES GENUINELY CANNOT USE THE MECHANISM, and one that also cannot is told to anyway. Gatekeeper and Auditor are deliberately granted no Bash tool (codexContinueReviewSpecs' own doc comment, locked by TestReviewSpecsDoNotInstructBashlessCastes after a prior incident where telling a bashless caste to run 'aether review-ledger-write' blocked phase advancement). The CR-05 fix therefore excludes those two from the invitation by name -- correct, and following an existing tested rule rather than inventing scope. But CLAUDE.md's Biological Runtime section still says generically 'A helper working on a piece of the job can now ask the program for backup', which remains very slightly broader than delivered: those two helpers cannot, and never could reach any CLI mechanism. Separately and inconsistently, aether-scout's Claude and OpenCode definitions ALSO grant no Bash (Grep/Glob/WebSearch/WebFetch only), yet the pre-existing TestEveryDispatchedWorkerIsToldHowToAskForHelp requires Scout to receive the invitation exactly like Builder and Watcher, and the CR-05 fix mirrored that accepted content verbatim into .codex/agents/aether-scout.toml. So the same 'no Bash' fact excludes two castes and not a third. Either Scout's tool grant is wrong, or the Gatekeeper/Auditor exclusion is broader than it needs to be -- one rule should decide all three. Not adjudicated during the CR-05 fix because both the Scout grant and its test pre-date this phase.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-13T22:44:36.143Z",
    "resolved_at": null
  },
  {
    "id": 36,
    "kind": "unmet-truth",
    "phase": "203",
    "file": "cmd/live_projection.go",
    "line": 551,
    "description": "PRE-EXISTING latent bug in the live-view resume path, found at the phase-203 closing gate and NOT introduced by it (git diff 1617ef3c..HEAD on cmd/live_projection.go shows the phase's only change to this file is additive Reason/Refusals fields plus gofmt realignment -- the elapsed/resume logic is untouched). ElapsedSeconds is recomputed from event timestamps ONLY when snapshot.Open is true (cmd/live_projection.go:551). A checkpoint taken mid-episode is therefore stamped with a live elapsed figure; when the run later closes, replayColonyLiveSnapshotResume carries that stale figure forward, while a full replay of the same events ends closed and leaves elapsed at 0. TestLiveProjectionResumesWithoutDoubleCounting asserts the two must be identical and correctly catches the divergence -- but only when the seeded events straddle a one-second boundary, so it is red perhaps one run in several. Observed 2026-09-14 with resumed ElapsedSeconds=1 vs full=0 and every other field byte-identical; passes in isolation. The underlying question is a behaviour decision for the owner, which is why this is recorded rather than patched: a FINISHED run arguably should report its total duration rather than 0, in which case the full-replay path is the one that is wrong, not the resume path. Fixing it by clearing the resumed value would make the test green while making the closed-episode screen less informative. TestSwarmCompatibilityWatchReportsActiveWorkers (active_count = 0, want 1) appeared in the same run, also passes in isolation, and reads the same projection -- likely the same family, not separately diagnosed.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-14T00:24:50.809Z",
    "resolved_at": null
  },
  {
    "id": 37,
    "kind": "deviation",
    "phase": "203",
    "file": ".planning/REQUIREMENTS.md",
    "line": null,
    "description": "FIXED, superseding entry 24. Requirement completion tracking had been silently broken repo-wide: gsd-tools' checkbox pattern is (-\\s*\\[)[ ](\\]\\s*\\*\\*<REQ-ID>\\*\\*) (lib/milestone.cjs:162), which needs the bold to close immediately after the ID, while this file wrote '- [ ] **BIO-02 -- Atomic admission:** ...' with the title inside the bold span. Every requirements mark-complete therefore returned not_found and wrote nothing, and phase.complete reported 'ROADMAP cites REQ-IDs not registered anywhere in REQUIREMENTS.md' for IDs that were plainly present. Reformatted all 65 requirement lines to '- [ ] **REQ-ID** -- Title: ...' -- a pure format change, verified content-identical to the previous file once ** markers are stripped, and no test anywhere pins the old shape (no cmd/*_test.go references REQUIREMENTS.md at all). Phase 203's ten IDs (SYNTH-05, CEC-07, BIO-01..08) are now genuinely ticked by the tool's own write path rather than by hand. The earlier hand-edit of BIO-07 recorded in 203-08-SUMMARY.md can now be done the canonical way.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-09-14T00:28:42.019Z",
    "resolved_at": "2026-09-14T00:28:53.780Z"
  },
  {
    "id": 38,
    "kind": "unmet-truth",
    "phase": "204",
    "file": "cmd/live_events.go",
    "line": null,
    "description": "FIXED by 204-15: all nine previously writerless episode-ledger fields (evidence_ids, hard_gate_results, changed_decision_ids, usage, reported_cost_usd, interventions, episode_revision, acceptance_digest, evaluator_digest) now have real production writers on the build lane, the native check lane (runCodexContinue), and the delegate check lane (runCodexContinueFinalize, which previously recorded no episode at all). The episode ledger also joined cmd/memory_schema.go's field-level census as its seventh store, with a writer entry for all 21 of its json fields -- TestEveryMemoryStoreFieldHasALiveWriter now fails by name if a future field-writer entry is deleted, which is the guard that stops this window from reopening. Usage/ReportedCostUSD remain honestly absent on both check lanes (a documented, non-fabricated gap -- see LEARN-02 in REQUIREMENTS.md), which does not reopen this window since the original defect was zero production callers, not incomplete coverage of every field on every lane.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-09-14T12:51:19.761Z",
    "resolved_at": "2026-09-15T00:22:57.000Z"
  },
  {
    "id": 39,
    "kind": "unmet-truth",
    "phase": "204",
    "file": "cmd/shadow_cmds.go",
    "line": null,
    "description": "FIXED by 204-12: aether shadow-declare and aether shadow-compare are now registered, reachable commands (cmd/improvement_cmds.go), alongside a new inspection surface, aether improve (default: read-only report; --declare/--compare route to the same mutating forms). The owner-facing exposure the original entry asked for is the /ant-improve wrapper triplet (.aether/commands/improve.yaml, .claude/commands/ant/improve.md, .opencode/commands/ant/improve.md), whose routing text names all three commands, satisfying TestNoRegisteredSubcommandIsUnreferenced without widening cmd/testdata/orphan_allowlist.json.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-09-14T14:21:30.954Z",
    "resolved_at": "2026-09-14T22:48:32.492Z"
  },
  {
    "id": 40,
    "kind": "unmet-truth",
    "phase": "204",
    "file": "cmd/shadow_cmds.go",
    "line": null,
    "description": "NARROWED by 204-12, not fixed: shadowEvaluator's run function is now a real per-fixture classifier (shadowClassifyAgainstBank), no longer the always-pass placeholder -- 204-16 re-confirmed TestShadowGraderDistinguishesBeneficialFromHarmful/TestOverfitCandidateIsRefusedAtTheGate/TestEvaluatorDigestIsStableAndChanged all pass. This entry was incorrectly marked fixed by 204-12; reopened by 204-16 per D-11 because the original description's own second sentence is still true today: comparison covers this project's own settings/routing fixtures only and never a code-valued candidate (LEARN-06's own scoping). Remains open until a follow-on plan extends isolation and grading to code-valued candidates.",
    "status": "open",
    "reason": "Reopened 2026-09-15 (204-16, D-11): grader is real, but code-valued candidates remain out of scope -- see LEARN-06 in REQUIREMENTS.md for the same caveat.",
    "recorded_at": "2026-09-14T15:40:47.610Z",
    "resolved_at": null
  },
  {
    "id": 41,
    "kind": "deviation",
    "phase": "204",
    "file": "cmd/episode_ledger_test.go",
    "line": null,
    "description": "FIXED by 204-13: runSwarmDestroy (cmd/swarm_cmd.go) and orchestrateRecovery (cmd/recovery_orchestrator.go, only when it owns the episode itself) now both open and close a durable episode via emitColonyLiveEpisodeStarted/Ended, proven by a FAILS-WHEN-UNWIRED mutation on each. TestEveryLifecycleLaneWritesADurableOutcome's t.Skipf branch is now a named t.Fatalf -- all six declared lifecycle lanes pass with zero skip lines.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-09-14T15:40:51.528Z",
    "resolved_at": "2026-09-15T00:22:57.000Z"
  },
  {
    "id": 42,
    "kind": "unmet-truth",
    "phase": "204",
    "file": "cmd/testdata/fixture-bank/v1/bank.json",
    "line": null,
    "description": "NARROWED by 204-14, still open: 10 previously-unguarded fixtures gained a real, passing, incident-specific guard test (guarded count 7 -> 17 of 47 total at the time; 22 of 52 after the ledger closures in 204-12 and 204-16 seeded five more fixtures, each guarded on arrival). seedBankUnguardedFloor lowered from 40 to 30 -- the exact real unguarded count, confirmed in this session against cmd/eval_gates.go -- and is now a two-sided ratchet (TestSeedBankUnguardedFloorIsTheRealCount) that fails if the recorded floor is ever raised above the real count, not only if the real count exceeds it. Remains open: 30 of 52 confirmed incidents still carry no guarding test. Close by naming and confirming a real guard test for each remaining fixture's own confirmed incident, shrinking the recorded floor as each is added -- the floor may only shrink, never widen.",
    "status": "open",
    "reason": "Narrowed 2026-09-15 (204-14/204-16): floor is 30, not 40 -- see LEARN-05 in REQUIREMENTS.md for the same number.",
    "recorded_at": "2026-09-14T15:40:55.056Z",
    "resolved_at": null
  },
  {
    "id": 43,
    "kind": "deviation",
    "phase": "204",
    "file": "cmd/memory_schema.go",
    "line": null,
    "description": "Owner decision 2026-09-14 (204-03 Task 4, 'mixture' reply accepting the recommendation table verbatim): instinct.related_instincts is retired (memoryStoreFieldRetired, reason: the only reader, pkg/graph, is doubly orphaned and this phase's own ruling (e) forbids citing it as justification for new work) -- no stored record touched, both production writers stopped setting it. midden.acknowledge_reason, learn.parent_id and pheromone.scope stay in memoryStoreFieldExceptions, recorded as knowingly empty (plausibly useful, but nothing in this phase's planned work needs them yet). Recorded here for traceability; closes only if a future phase names a concrete consumer for one of the three empty fields.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-14T15:40:59.645Z",
    "resolved_at": null
  },
  {
    "id": 44,
    "kind": "unmet-truth",
    "phase": "204",
    "file": "cmd/learning_validator.go",
    "line": null,
    "description": "REOPENED by 204-REVIEW.md CR-01 (2026-09-15): 204-16's promoteHelpfulHypotheses (called from runPhaseEndConsolidation alongside the improvement pass) cannot promote anything in production. Root cause is an ID-space mismatch confirmed by grep: recordGuidanceApplicationState's only non-test callers key every GuidanceApplications record by an Instinct's own ID (cmd/instinct_application.go:69,75), never by a learn.Entry's ID -- learn.Entry.ParentID is hypothesis-revision lineage only, and no production writer links a learn.Entry to the Instinct (if any) it came from. Independently, no non-test call site anywhere in cmd/ ever writes the guidanceApplicationStateHelpful/Neutral/Harmful terminal states at all -- the \"helpful\" signal QUEEN promotion actually reads lives in a different list (recruitmentCreditFile.Entries) that is never bridged into the guidance-application vocabulary. learningEntryHasHelpfulApplication therefore returns false for every real hypothesis, forever. The test suite previously passed only because cmd/learning_validator_test.go hand-typed a learn entry's ID and reused it as the GuidanceApplications guidanceID -- a shape the runtime cannot produce (a false certificate per CLAUDE.md's own Definition of Done). Fixed at the test layer (learning_validator_test.go now proves the honest current limitation via TestHypothesisPromotionNeverCrossesTheIdentifierGap, observed FAILING when the gating logic is mutated to always match) and documented at the production layer (learning_validator.go's package and function doc comments now state the limitation and its two root causes in full). No fabricated identifier bridge was introduced. Closes only when a real production writer either links a learn.Entry to its source Instinct or writes GuidanceApplications under learn.Entry.ID from a real call site, AND a real writer of the Helpful/Neutral/Harmful states exists.",
    "status": "open",
    "reason": "ID-space mismatch between learn.Entry and GuidanceApplications, plus no production writer of the Helpful/Neutral/Harmful states at all; no honest identifier bridge exists within reach of the 204-REVIEW.md fix pass (2026-09-15)",
    "recorded_at": "2026-09-14T15:41:03.868Z",
    "resolved_at": null
  },
  {
    "id": 45,
    "kind": "unmet-truth",
    "phase": "204",
    "file": "cmd/source_proposal.go",
    "line": null,
    "description": "FIXED, both halves: (1) 204-13 declared episodeInterventionKind, a closed, source-derived intervention-kind vocabulary with three real production writers, so collectPreventableInterventions now classifies against a curated set instead of treating every free-form string as its own category. (2) 204-16 gave proposeSourceImprovement its first real production caller, triggerRepeatedInterventionProposal (cmd/improvement_pass.go): when the same declared intervention kind recurs on 3+ distinct episodes, it proposes a source change naming the real episodes, on an isolated branch, with sourceProposalReachabilityEntryPoints extended in the same change so TestSourceProposalCannotMergePublishOrDeploy's call-graph walk covers the new entry point too.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-09-14T15:41:07.891Z",
    "resolved_at": "2026-09-15T00:22:57.000Z"
  },
  {
    "id": 46,
    "kind": "unmet-truth",
    "phase": "202.1",
    "file": "pkg/codex/platform_dispatch.go",
    "line": null,
    "description": "workerProcessEnv (pkg/codex/process_tracker.go) had NO caller, so AETHER_WORKER_NAME never reached a spawned worker. aether hook-stop therefore could not tell an Aether build worker from a person: it blocked worker Weld-32 mid-build and advised aether pause, the worker ran it, and a live CosmicDashboard Autopilot colony was paused mid-phase. Wired the env at the spawn site and exempted Aether-spawned workers from hook-stop. Proven by a REAL spawned subprocess reading back its own environment (TestSpawnedWorkerCarriesItsIdentityInTheEnvironment) rather than by testing the builder in isolation -- an isolated builder test passed for the entire time the wiring was missing. Migrated 2026-09-14 by plan 204-11 from a stray, out-of-band duplicate row (originally id 14, colliding with the real phase-203 entry 14) that had drifted below the JSON ledger block; original recorded/resolved timestamps were 2026-09-12T21:30:00.000Z.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-09-14T15:42:11.225Z",
    "resolved_at": "2026-09-14T15:42:13.430Z"
  },
  {
    "id": 47,
    "kind": "unmet-truth",
    "phase": "204",
    "file": "cmd/swarm_cmd.go",
    "line": null,
    "description": "PRE-EXISTING latent collision in swarm worker naming, found at the Phase 204 gap-closure wave-3 gate (2026-09-15) and NOT introduced by it: no gap-closure plan touches swarm naming. deterministicAntName (cmd/codex_visuals.go) derives a worker name as prefix[hash mod len(prefixes)] plus a number from hash mod 99, seeded by the colony root path, caste and target; buildSwarmManifest (cmd/swarm_cmd.go, the duplicate dispatch name check near line 1256) then refuses the whole manifest when two dispatches land on the same name instead of de-duplicating. With five or six workers drawn from roughly six prefixes times 99 numbers, any two collide about one run in fifty to a hundred, and because the root path is part of the seed, a given colony can be stuck colliding on a given target every time. Observed as TestSwarmFinalizeRecordsExternalTaskResults/timeout failing with duplicate dispatch name Guard-95 once in a full-suite run at 05b0d453; the same test passed in the previous gate at 2eae06e3 and passed three of three re-runs in isolation. Close by making the manifest builder append a disambiguating suffix on collision (or fold the caste index into the seed) with a test that forces two workers onto one name and asserts both are dispatched under distinct names.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-15T01:08:31.907Z",
    "resolved_at": null
  }
]
````
