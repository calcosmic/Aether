---
phase: 203-biological-runtime
part: c
reviewed: 2026-09-13T21:29:35Z
depth: standard
files_reviewed: 27
files_reviewed_list:
  - cmd/live_events.go
  - cmd/live_projection.go
  - cmd/watch_live.go
  - pkg/events/colony_live.go
  - cmd/status.go
  - cmd/lifecycle_status_render.go
  - cmd/codex_visuals.go
  - cmd/next_action.go
  - cmd/next_action_card.go
  - cmd/spend_cost_line.go
  - cmd/hook_cmds.go
  - cmd/subcommand_reachability_ratchet_test.go
  - cmd/workers_doc_test.go
  - cmd/command_call_audit_test.go
  - cmd/display_house_style_test.go
  - cmd/status_test.go
  - cmd/hook_cmds_test.go
  - cmd/lifecycle_status_199_test.go
  - cmd/lifecycle_card_endgame_test.go
  - cmd/codex_visuals_test.go
  - cmd/claudemd_biological_runtime_test.go
  - cmd/claudemd_classic_voice_test.go
  - .claude/agents/ant/aether-builder.md
  - .opencode/agents/aether-builder.md
  - .codex/agents/aether-route-setter.toml
  - .aether/workers.md
  - .github/workflows/ci.yml
status: issues
critical: 1
warning: 3
info: 2
---

## Verdict on the orphan question (read this first)

**The acceptance signal (`TestNoRegisteredSubcommandIsUnreferenced`) is now genuinely tied to real evidence and can genuinely fail — I independently reproduced this.** I ran the test suite, confirmed `workerDisciplineCallerFiles` now points at `.claude/agents/ant/{aether-builder,aether-watcher,aether-scout}.md` (real files that Claude Code's `--agent` flag loads as a subagent's system prompt — traced through `pkg/codex/platform_dispatch.go:899` `claudeBaseWorkerArgs(...) --agent aether-builder`), confirmed `cmd/testdata/orphan_allowlist.json` carries no `recruit` entry (so it isn't passing via a widened allowlist), and confirmed `renderRecruitmentInvitation` (`cmd/codex_build.go:4741`) has exactly one production caller, `composeBuildManifestBrief` (`cmd/codex_build.go:4667`), which is itself reached from `attachBuildDispatchContext` at two real production call sites (`cmd/codex_build.go:764`, inside `runCodexBuildWithOptions`'s plan-only branch, and `cmd/build_print_brief.go:108`). WINDOWS.md entry 26's narrow claim — that the *test itself* stopped being a false positive — holds up.

**But the underlying wiring WINDOWS.md entry 26 calls "genuinely closed" is not closed for every dispatch lane, and the acceptance test does not (and structurally cannot, given `TestTheRecruitInstructionHasOneSource` pins `renderRecruitmentInvitation` to exactly one caller) prove otherwise.** See CR-01 below: on the native/direct build dispatch lane (`executeCodexBuildDispatches` → `buildCodexWorkerDispatches` → `renderCodexBuildWorkerBrief`, used by autopilot/`/ant-run` and by any non-wrapper `aether build <phase>` invocation per the code's own comments), and specifically on the Codex platform, a dispatched worker's assembled prompt contains **zero** mention of `aether recruit`, in the TOML agent instructions, in the task brief, or anywhere else. Zero of the 27 `.codex/agents/*.toml` files mention "recruit" in any form, and `renderCodexBuildWorkerBrief` (the sole `TaskBrief` source on that lane) never calls `renderRecruitmentInvitation`. So `aether recruit` remains a real, functioning, registered command that a Codex-platform worker — and any worker dispatched via the native/direct lane without going through the Claude/OpenCode host's own `--agent` subagent loading — has literally no way to discover from its own context. The phase's own reachability ratchet cannot see this because its evidence sources are file-presence checks (agent `.md`/`.toml` corpora, wrapper docs) and a single unit test that calls `composeBuildManifestBrief` directly rather than exercising the real dispatch call chain — precisely CLAUDE.md's failure mode #4 ("a green test that measures a function the real command never calls"), just one hop narrower than the original orphan.

Net: the phase correctly closed the orphan for the **interactive Claude Code / OpenCode wrapper plan-only lane** (redundantly, via both the composed brief and the agent `.md` file). It did **not** close it for the **native/direct dispatch lane**, and it definitely did not close it for the **Codex platform**, which CLAUDE.md documents as a supported (if secondary) platform for "the direct `aether build` workflow" — exactly the lane this gap lives in.

## Findings

### CR-01 (Critical): `aether recruit` is still unreachable from a Codex-platform worker's prompt, and from any worker dispatched via the native/direct build lane

**File:** `cmd/codex_build.go:3860` (`renderCodexBuildWorkerBrief`, the sole `TaskBrief` source for the native lane) and `.codex/agents/aether-builder.toml` / `aether-watcher.toml` / `aether-scout.toml` (0 of 27 `.codex/agents/*.toml` files mention "recruit")

**What is wrong:** `renderRecruitmentInvitation()` is only ever written into a worker's prompt through `composeBuildManifestBrief` (`cmd/codex_build.go:4733`), which has exactly two production callers, both serving the **wrapper plan-only** flow (`attachBuildDispatchContext`, `cmd/codex_build.go:764` and `cmd/build_print_brief.go:108`). The **native/direct** dispatch lane — `runCodexBuildWithOptions`'s non-plan-only branch calling `executeCodexBuildDispatches` (`cmd/codex_build.go:1236`) → `buildCodexWorkerDispatches` (`cmd/codex_build.go:2542`) — sets `TaskBrief: renderCodexBuildWorkerBrief(...)` (`cmd/codex_build.go:2566`) directly, never through `composeBuildManifestBrief`, so the invitation text never appears there. For the Codex platform specifically, `pkg/codex/worker.go:385` assembles the final prompt via `AssemblePrompt(config.AgentTOMLPath, ...)`, whose only "instructions" source is `LoadAgentInstructions` (`pkg/codex/prompt.go:30`) reading the `developer_instructions` field out of the `.codex/agents/*.toml` file — none of which were updated (verified: `grep -il "recruit" .codex/agents/*.toml` returns nothing across all 27 files). `.codex/CODEX.md` also does not mention it. So a Codex-dispatched worker's fully assembled prompt (TOML instructions + capsule + handoff + skill + pheromone + task brief) contains no occurrence of "recruit" anywhere.

For Claude/OpenCode, the native lane happens to still work by accident of a *different* mechanism: `ClaudeDispatcher.InvokeWithProgress` passes `--agent <agentName>` to the `claude` CLI (`pkg/codex/platform_dispatch.go:899-905`), and the CLI itself loads `.claude/agents/ant/aether-builder.md` (which does carry the invitation) as the subagent's own system prompt, independent of anything Aether's Go code assembles. `OpenCodeDispatcher` similarly routes through a Task-tool-style subagent dispatch naming `config.AgentName` (`renderOpenCodeSubagentDispatchPrompt`, `pkg/codex/platform_dispatch.go:988`). Codex has no equivalent external loading step — `LoadAgentInstructions` IS the only place Codex-specific "agent definition" content enters the prompt, and Aether's own code populates it, incompletely.

**Concrete failure scenario:** A user on the documented, supported Codex CLI platform runs `aether build 5` directly (the "native/direct" invocation CLAUDE.md itself documents as a real, supported lane, distinct from the interactive wrapper). A Builder gets stuck mid-task and needs a Watcher's help. It has no way to know `aether recruit` exists — nothing in its prompt ever mentioned it. It either stalls, guesses at a nonexistent mechanism, or gives up on a task it could have finished with help, silently defeating BIO-01/BIO-02's stated goal of "a helper working on a piece of the job can now ask the program for backup." The exact same failure applies to any autopilot (`/ant-run`) build on any platform, since `executeCodexBuildDispatches` is the lane the code's own comment (`cmd/codex_build.go:2614`) names as serving "every autopilot /ant-run build" — Claude/OpenCode autopilot builds are saved only by the incidental `--agent` flag behavior above, which nothing in this codebase asserts or tests.

**Why the acceptance test cannot catch this:** `TestEveryDispatchedWorkerIsToldHowToAskForHelp` (`cmd/recruitment_wiring_test.go:26`) calls `composeBuildManifestBrief` directly as a unit — it never calls `executeCodexBuildDispatches`, `buildCodexWorkerDispatches`, `renderCodexBuildWorkerBrief`, or `AssemblePrompt`/`LoadAgentInstructions`. `TestTheRecruitInstructionHasOneSource` (`cmd/recruitment_wiring_test.go:54`) actively *enforces* that `renderRecruitmentInvitation` has exactly one caller, which architecturally guarantees the native lane's own brief composer can never independently carry the text without that test itself needing to change. I grepped every test file that exercises the native lane (`renderCodexBuildWorkerBrief`, `buildCodexWorkerDispatches`, `executeCodexBuildDispatches`) for any mention of "recruit" and found none.

**Fix:** Either (a) call `renderRecruitmentInvitation()` from `renderCodexBuildWorkerBrief` (or append it in `buildCodexWorkerDispatches`'s `TaskBrief` construction) so the native lane's own brief carries it — the same `TestTheRecruitInstructionHasOneSource` invariant can be relaxed to "one canonical constant, potentially multiple emission sites" rather than "exactly one caller" — and (b) add the same `## Asking For Help` block (or an equivalent `developer_instructions` addendum) to the 27 `.codex/agents/*.toml` files, mirroring what was done for the Claude/OpenCode `.md` agents. Add a test that asserts the string appears in the *actual* assembled Codex prompt (via `AssemblePrompt`/`LoadAgentInstructions` against the real `.toml` files) and in the actual native-lane `TaskBrief`, not merely in `composeBuildManifestBrief`'s output.

---

### WR-01 (Warning): Phase 202.1's Classic-voice guarantee for `aether status` still measures a renderer the real command never calls — confirmed still true

**File:** `cmd/status.go:60` (real `aether status` calls `renderDashboard`, `cmd/status.go:903`) vs. `cmd/lifecycle_status_render.go:32` (`renderLifecycleStatus`, what `TestStatusScreenMeetsTheReferenceDensity`/`TestEveryVoicedScreenMeetsTheReferenceDensity` actually measure)

**What is wrong:** I independently re-verified WINDOWS.md entries 22/23, which are open (not resolved). `grep -rln "renderLifecycleStatus(" cmd/*.go` returns only `cmd/lifecycle_status_render.go` itself (the definition), `cmd/classic_voice_status_test.go`, `cmd/lifecycle_status_199_test.go`, and `cmd/orientation_agreement_199_test.go` — every non-test reference is gone. The only near-production caller is `cmd/compatibility_cmds.go:473`, and that calls a *different* function, `renderLifecycleStatusCompact`, not `renderLifecycleStatus`. The real `aether status` RunE handler at `cmd/status.go:60` renders via `renderDashboard`, which is a structurally different function with its own top block (Goal, Runtime, Signals, Progress, Focus, Instincts, Flags, Scope, Colony Mode, Depth, Granularity, Parallel) that per WINDOWS.md's own measurement carries no leading voice symbol at all. This is not something Phase 203 introduced, but Phase 203 shipped new `aether status` content (`renderGovernedSubtreeStatusSection`, confirmed wired into `renderDashboard` at `cmd/status.go:1144`) into the same unvoiced screen without correcting the underlying test-target mismatch, so the phase inherited and extended a screen whose own "Classic voice" guarantee is provably not enforced by any passing test.

**Concrete failure scenario:** A future refactor strips every leading symbol from `renderDashboard`'s top block (the block the owner actually reads) and the entire green test suite, including all five voice tests, continues to pass — because none of them touch the function the owner's terminal actually renders.

**Fix:** Point the density test at `renderDashboard`'s actual rendered output (or add a new test that does), and either delete `renderLifecycleStatus` or make it a real second consumer of the same content so the two can't structurally diverge again.

### WR-02 (Warning): `renderPlanningStopVisual` remains a fully orphaned renderer — confirmed still true

**File:** `cmd/planning_visuals.go:550`

**What is wrong:** `grep -rln "renderPlanningStopVisual("` returns only `cmd/planning_visuals.go` (definition) and `cmd/classic_voice_plan_test.go` (test). No non-test caller exists anywhere in `cmd/`. The `plan-stop` screen registered in the Classic-voice corpus is therefore never drawn by any real invocation of `aether plan`; its density test proves a property of dead code.

**Fix:** Either wire `renderPlanningStopVisual` into the actual plan-stop code path, or remove it and its corpus registration/density test so the corpus doesn't carry a screen nothing draws.

### WR-03 (Warning): The reachability ratchet and wiring gate can silently not execute in CI without the run being marked red — a live, load-bearing risk for exactly the tests this review depends on

**File:** `cmd/testing_main_test.go` (per WINDOWS.md entries 16 and 27, not something I re-derived independently from first principles, but I did confirm via `.github/workflows/ci.yml:100` that the wiring-gate step is a single `go test -run '<huge alternation>'` invocation, the same shape the ledger describes as having silently dropped `TestPlatformParityGolden` and `TestStatusPrefersCurrentRunWorkersOverStaleHistory` in two separate incidents on 2026-09-13)

**What is wrong:** The CI step that is supposed to be the load-bearing proof that `aether recruit` is wired (`TestNoRegisteredSubcommandIsUnreferenced`, `TestRecruitAcceptsDocumentedInvocation`, and dozens of others) is one long `-run` regex alternation. WINDOWS.md documents two confirmed incidents this same session where a named test in exactly this kind of invocation silently failed to execute at all while the lane still reported a clean result. Nothing distinguishes "this test ran and passed" from "this test was silently dropped and the suite reported success anyway" in the CI output as currently structured. Since CR-01 above shows the *actual* wiring has a real gap the current tests don't cover, and since this section's own tests are demonstrated (by the project's own ledger, same day) to be capable of silently not running, the confidence anyone should place in "the gate is green" is lower than the gate's own report claims.

**Fix:** This is already tracked (WINDOWS.md #16/#27) as a standing, unresolved defect outside this phase's declared scope to fix; noting it here because it directly undermines confidence in the specific wiring claims this review was asked to verify. A minimal fix: after the `-run` alternation completes, assert the count of `--- PASS`/`--- FAIL` lines in the test's own JSON/verbose output equals the number of test names in the `-run` string, failing loudly on any mismatch.

---

### IN-01 (Info): `workerDisciplineCallerFiles`'s comment and the D-08 family tree/decision-changed lines are sound and correctly wired — no action needed, noted for completeness

I verified `renderGovernedSubtreeStatusSection` (`cmd/recruitment_subtree.go:325`) is called from `renderDashboard` (`cmd/status.go:1144`, inside the function `aether status`'s RunE actually calls), that it correctly scopes to `SpawnTree.CurrentRun()`/`EntriesForRun()` with a "no current run means no filter" fallback (the fix for WINDOWS.md entry 27's stale-worker regression), and that `TestStatusPrefersCurrentRunWorkersOverStaleHistory` (`cmd/status_test.go:724`) uses a realistic fixture (real `spawn-tree.txt` pipe-delimited lines and a real `spawn-runs.json` shape) rather than a fixture in a shape the runtime cannot produce. `appendSpendCostLine` (`cmd/spend_cost_line.go:367`) is confirmed as the single funnel for the end-of-run family tree (`renderRecruitmentFamilyTree`) and the decision-changed notes, called from all four documented closing screens (`cmd/ceremony_cmd.go:334,752`, `cmd/codex_workflow_cmds.go:490`, `cmd/lifecycle_closeout.go:830`, `cmd/work_closeout.go:221`). `emitInlineRecruitLine`/`emitInlineRefusalLine` (`cmd/codex_visuals.go:568,590`) both route through `emitVisualLine`, which correctly gates on `shouldRenderVisualOutput`/`streamingAllowedForCurrentCommand` and writes via `writeVisualOutput` (the house-style-compliant funnel), and both `emitColonyLiveRecruitAdmitted`/`emitColonyLiveRecruitRefused` have real, non-test call sites in both `cmd/recruitment.go` (198, 268) and `cmd/recruitment_lane.go` (190, 221) — i.e. both the native `aether recruit` CLI lane and the in-repo build-lane recruitment path genuinely emit the inline lines the owner is promised. No defect found in this subsystem.

### IN-02 (Info): `recruitmentRecoveryNextAction`'s placeholder argument counts are correctly matched per class

I checked `recruitmentRecoveryAdviceTemplate` (`cmd/next_action.go:927`) against its caller `recruitmentRecoveryNextAction` (`cmd/recruitment_recovery.go:75`): the `missing`/`timed-out` classes correctly receive 3 args (`child, recruitmentID, recruitmentID`) matching their `%[1]s`/`%[2]s`/`%[3]s` templates, and the remaining three classes correctly receive 2 args. No `fmt.Sprintf` argument-order bug found here, despite this being exactly the kind of code (positional-index format strings, five branches, easy to get one wrong) I'd expect to find one in.

---

_Reviewed: 2026-09-13T21:29:35Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
