---
phase: 197-one-answer-to-what-next
plan: 04
subsystem: cli
tags: [go, next-action, closing-card, envelope, lifecycle, platform-translation, golden-transcripts]

requires:
  - phase: 197-one-answer-to-what-next
    provides: "resolveNextAction and the candidate set from 197-01; renderNextActionCard and applyNextActionToResult from 197-02"
provides:
  - "Seven lifecycle closings — starting, discussing, scanning, planning, building, checking and the shared closeout — render renderNextActionCard instead of a hand-written block"
  - "The same seven fold the resolved answer into their result maps, so the screen and the machine-readable envelope cannot name different commands"
  - "nextActionInput.Override — a run's own more-specific command reaches the decision as an input rather than being written over its answer"
  - "lifecycleNextAction / lifecycleNextActionForState / closeLifecycleCommand / closeLifecycleRun / renderLifecycleClosing — the four wiring helpers every later surface reuses"
  - "TestMigratedLifecycleSurfacesAgree — one saved project, seven closings, one command out of all of them"
affects: [197-05, 197-06 remaining closing renderers, 197-07 cross-command envelope assertion]

actuals:
  tokens: 25464
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Override-as-input: a run's more specific command is fed to the resolver and gated like any other, never applied after the answer"
    - "Report above, advice below: what a run did (host dispatch, failed gates, unfinished work, what the last step said) stays above the closing block; only the what-next block is replaced"
    - "One-resolve wiring: closeLifecycleRun runs LAST in a result builder, after everything that can still add a more specific command"
    - "Surface-level agreement invariant: render every migrated closing over ONE saved project and require one command out of all of them"

key-files:
  created:
    - cmd/lifecycle_card_startup_test.go
    - cmd/lifecycle_card_workloop_test.go
    - cmd/lifecycle_card_agreement_test.go
  modified:
    - cmd/codex_visuals.go
    - cmd/ceremony_cmd.go
    - cmd/discuss.go
    - cmd/init_cmd.go
    - cmd/codex_workflow_cmds.go
    - cmd/codex_colonize_finalize.go
    - cmd/codex_plan_finalize.go
    - cmd/codex_build.go
    - cmd/codex_build_finalize.go
    - cmd/codex_continue.go
    - cmd/codex_continue_plan.go
    - cmd/codex_continue_finalize.go
    - cmd/next_action.go
    - cmd/next_action_card.go
    - cmd/testdata/golden_plan.txt
    - cmd/testdata/golden_build.txt
    - cmd/testdata/golden_continue.txt

key-decisions:
  - "A run's own command reaches the card through the resolver as an Override input, gated against the live command tree like any other candidate — so a stale redispatch command falls back rather than being recommended"
  - "Only three situations qualify as a run's own knowledge: the redispatch for unfinished work, the exact command a blocked check named, and an unanswered question. The ordinary 'check it next' is left to the one decision, which is what stops a paused or failed project being told to carry on"
  - "The legacy `next` key was left exactly as it was; the structured keys sit beside it"
  - "Host-dispatch and plan-only instructions moved from the Next Up block into a 'How this run is being driven' section above the card — they report what the run did, they are not advice to the owner"
  - "Saved event lines are now shown message-only; the timestamp and internal event code are dropped, which also makes the recorded transcripts stable"

patterns-established:
  - "Two-run comparison for a whole-command surface: the same prepared project driven once for the screen and once for the envelope, both compared against the resolver's answer"
  - "One-run comparison for a renderer-level variant: the production function produces the result map and the screen together, so the card and the envelope can never be compared against two different situations"

requirements-completed: [NEXT-02]

coverage:
  - id: D1
    description: "Starting a project, talking the goal through, scanning existing code and drawing up a plan all end with the card the one resolver produced, byte for byte, without losing what each run reported about itself."
    requirement: NEXT-02
    verification:
      - kind: integration
        ref: "cmd/lifecycle_card_startup_test.go#TestStartupLifecycleCardsComeFromTheResolver"
        status: pass
    human_judgment: false
  - id: D2
    description: "Those four commands' machine-readable answers name the same command and carry the same alternatives as their cards."
    requirement: NEXT-02
    verification:
      - kind: integration
        ref: "cmd/lifecycle_card_startup_test.go#TestStartupLifecycleEnvelopesMatchTheirCards"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every build closing (full dispatch, plan-only, part-finished, finalize) and every check closing (ordinary advance, plan-only, blocked), plus the shared closeout, end with the same card."
    requirement: NEXT-02
    verification:
      - kind: integration
        ref: "cmd/lifecycle_card_workloop_test.go#TestWorkLoopCardsComeFromTheResolver"
        status: pass
      - kind: integration
        ref: "cmd/lifecycle_card_workloop_test.go#TestWorkLoopEnvelopesMatchTheirCards"
        status: pass
    human_judgment: false
  - id: D4
    description: "A part-finished build's command for the unfinished work reaches the screen, the card's recommendation, the new machine-readable key and the older next key from one decision — the test fails if it reaches only one of them."
    requirement: NEXT-02
    verification:
      - kind: integration
        ref: "cmd/lifecycle_card_workloop_test.go#TestRedispatchOverrideReachesBothTheCardAndTheEnvelope"
        status: pass
    human_judgment: false
  - id: D5
    description: "A blocked check still lists its failed gates and their fix hints above the card; a part-finished build still names which work was done and which was not."
    requirement: NEXT-02
    verification:
      - kind: integration
        ref: "cmd/lifecycle_card_workloop_test.go#TestBlockedCheckStillExplainsItself"
        status: pass
    human_judgment: false
  - id: D6
    description: "Driven over one saved project, all the migrated closings name the same command; a single surface reverted to its old block makes the test fail by name."
    requirement: NEXT-02
    verification:
      - kind: unit
        ref: "cmd/lifecycle_card_agreement_test.go#TestMigratedLifecycleSurfacesAgree"
        status: pass
      - kind: manual_procedural
        ref: "revert demonstration: the scanning closing given back its hand-written block, TestMigratedLifecycleSurfacesAgree FAILED naming it, revert undone"
        status: pass
    human_judgment: false
  - id: D7
    description: "On each recognised platform the screen shows that platform's spelling while the executed value stays in the runtime form (S-01)."
    requirement: NEXT-02
    verification:
      - kind: unit
        ref: "cmd/lifecycle_card_agreement_test.go#TestMigratedLifecycleSurfacesArePlatformCorrect"
        status: pass
    human_judgment: false
  - id: D8
    description: "Every command any of the seven can put in front of the owner resolves against the live command tree (criterion 6, applied to the surface rather than the resolver)."
    verification:
      - kind: unit
        ref: "cmd/lifecycle_card_agreement_test.go#TestEveryCommandTheSevenCanRecommendResolves"
        status: pass
    human_judgment: false
  - id: D9
    description: "No migrated closing tells the owner twice whether it is safe to close the chat, and none still prints the old context sentence."
    verification:
      - kind: integration
        ref: "cmd/lifecycle_card_startup_test.go#TestMigratedStartupClosingsSayItOnce"
        status: pass
      - kind: integration
        ref: "cmd/lifecycle_card_workloop_test.go#TestMigratedWorkLoopClosingsSayItOnce"
        status: pass
    human_judgment: false
  - id: D10
    description: "The closing block of each of the seven uses no word this repository invented without explaining it in the same sentence."
    verification:
      - kind: unit
        ref: "cmd/lifecycle_card_agreement_test.go#TestTheSevenClosingsSpeakPlainEnglish"
        status: pass
    human_judgment: true
    rationale: "The check proves no vocabulary-table word appears unexplained in the closing block, and 197-02's TestPlainEnglishCheckCanFail proves the checker can report a violation. Whether the resulting sentences land for a non-technical reader is a judgement no test asserts."
  - id: D11
    description: "The three recorded transcripts were refreshed and every changed line read; each is a change in advice, and nothing above the closing block moved or vanished."
    verification:
      - kind: manual_procedural
        ref: "diff of cmd/testdata/golden_{plan,build,continue}.txt read line by line and quoted in this summary"
        status: pass
      - kind: integration
        ref: "cmd/golden_workflow_test.go#TestGoldenPlanVisualOutput,TestGoldenBuildVisualOutput,TestGoldenContinueVisualOutput (run -count=2 for stability)"
        status: pass
    human_judgment: false

duration: 118 min
completed: 2026-08-28
status: complete
---

# Phase 197 Plan 04: Seven Lifecycle Closings Render the One Card Summary

**Starting, discussing, scanning, planning, building, checking and the shared closeout now end with the single card the resolver produces, their machine-readable answers carry the identical command and alternatives, and a run's own more specific command reaches the owner through the decision rather than around it.**

## Performance

- **Duration:** 118 min
- **Tasks:** 3
- **Files created:** 3
- **Files modified:** 17 (14 source, 3 recorded transcripts)

## Accomplishments

- **The card is on screen for the first time.** Plans 197-01 and 197-02 built the decision and the card; nothing rendered it. Every one of the seven surfaces now does, and the test asserts the card is present **byte for byte**, not that a block that looks similar exists.
- **Fifteen hand-written closings replaced.** Four for planning (repaired, next-iteration, two manifest-only, ordinary), four for building (full, part-finished, plan-only, finalize), three for checking (ordinary, plan-only, blocked), two for discuss (main and answer-locked-in), one for scanning, one for starting, plus the two closeout renderers.
- **The starting command's fixed trio is gone from the source.** It recited the same three alternatives for every repository on earth. `TestTheStartingCardNoLongerRecitesAFixedTrio` fails if any of the three sentences comes back.
- **`nextActionInput.Override`** — a run's own more specific command is now an INPUT to the decision, gated against the live command tree like any other candidate. A stale redispatch command falls back to the ordinary answer and the substitution is recorded, instead of putting a command the program no longer has in front of the owner.
- **`TestMigratedLifecycleSurfacesAgree`** drives all the migrated closings over one saved project and requires one command out of all of them. It was demonstrated failing (below).
- **Two real defects found and fixed on the way** (both in files this plan does not own — see Deviations): saved event lines were being printed to the owner raw, timestamps and internal codes included; and the ceremony closeout dropped the finished step's own report when its `next` sentence was replaced.

## Task Commits

1. **Task 1 RED: the four startup closings** — `3a46eaa3` (test)
2. **Task 2 RED: build, check and closeout** — `50678f4c` (test)
3. **Tasks 1 + 2 GREEN: the seven closings render the card** — `f0b78843` (feat)
4. **Task 3: the agreement and platform tests** — `e078d20b` (test)
5. **Task 3: the last two variants and the wording check** — `d36bc51c` (test)

The two migrations landed in one `feat` commit because they share `cmd/codex_visuals.go`, `cmd/codex_workflow_cmds.go` and `cmd/next_action.go`; splitting them would have produced two commits neither of which compiled. Both RED runs were recorded and committed **before** any of that implementation existed.

### Recorded RED output

**Task 1 RED** — behavioural, not a build failure. The tests drive the real commands, so they compiled against the unchanged tree and failed on the real absence:

```
--- FAIL: TestStartupLifecycleCardsComeFromTheResolver/starting_a_project
    starting a project does not end with the shared card.
--- FAIL: TestStartupLifecycleCardsComeFromTheResolver/talking_the_goal_through
--- FAIL: TestStartupLifecycleCardsComeFromTheResolver/scanning_the_existing_code
--- FAIL: TestStartupLifecycleCardsComeFromTheResolver/drawing_up_the_plan
--- FAIL: TestStartupLifecycleEnvelopesMatchTheirCards/starting_a_project
    starting a project emits no "next_command" in its machine-readable answer;
    a wrapper cannot read the next step out of it
--- FAIL: TestTheStartingCardNoLongerRecitesAFixedTrio
    the starting command still recites its fixed alternative
    "to lock down key clarifications before planning"
```

**Task 2 RED:**

```
--- FAIL: TestWorkLoopCardsComeFromTheResolver/building_a_phase
--- FAIL: TestWorkLoopCardsComeFromTheResolver/building_a_phase,_plan_only
--- FAIL: TestWorkLoopCardsComeFromTheResolver/checking_the_work
--- FAIL: TestWorkLoopCardsComeFromTheResolver/the_shared_closeout
--- FAIL: TestWorkLoopCardsComeFromTheResolver/building_a_phase_that_only_partly_finished
--- FAIL: TestWorkLoopCardsComeFromTheResolver/a_check_that_is_blocked
    a blocked check folds no resolved answer into its result; the card and the
    envelope cannot come from one decision
--- FAIL: TestRedispatchOverrideReachesBothTheCardAndTheEnvelope
--- FAIL: TestMigratedWorkLoopClosingsSayItOnce/building_a_phase
    building a phase still prints the old context sentence
```

## The revert demonstration

Acceptance criterion: *reverting any single surface to its old block makes the consistency test fail.* The scanning closing (`renderColonizeVisual`) was temporarily given back its hand-written line:

```
--- FAIL: TestMigratedLifecycleSurfacesAgree
    lifecycle_card_agreement_test.go:102: scanning the code tells the owner to
    run "aether plan"; the one answer for this project is "aether build 1" --
    the surfaces have separated
```

Reverted with `git checkout --`; `git status --short` clean afterwards.

## The recorded transcripts, read line by line

All three changed. Every changed line is a change in advice; **no section above the closing block moved or vanished** in any of them. The transcripts were re-run twice without the update flag afterwards to confirm they are stable (an earlier attempt printed a timestamp into the transcript — see Deviations).

### `golden_plan.txt`

```diff
+━━ 📊 W H A T   N E X T ━━
+── Where things stand ──
+Goal: Golden workflow test colony
+Phase 1 of 6: Contract and gap mapping. 0 phase(s) finished so far.
+── What changed ──
+  - The last thing you ran was plan.
+  - Scout summarized surveyed repo context
+  - Activated plan-r1-be3deeac3c65 (initial): Initial plan generated
+  - Generated 6 active phases with 76% confidence; planning loop stopped: pending
 ━━ 🐜 N E X T   U P ━━
-Run `/ant-build 1` to start the next planned phase.
-Alternative: Run `/ant-focus "..."` or `/ant-redirect "..."` if you want to adjust the colony before the first wave.
+Run `/ant-build 1` — Phase 1 (Contract and gap mapping) is ready to start. Helpers will write the code for it and report back before anything is accepted.
+Alternative: Run `/ant-focus "area"` — Point the helpers at one area before the next build starts.
+Alternative: Run `/ant-status` — Look at the dashboard first, without changing anything.
+Alternative: Run `/ant-pheromones` — See the standing instructions you have given the helpers -- what to focus on and what to avoid.
+Everything needed to pick this back up is written down, so it is safe to close this chat.
```

*Why each change is a change in advice:* the same command is recommended (`/ant-build 1`), now with the reason for it; the two alternatives were a hand-typed pair with a placeholder ellipsis and became three real, gated commands each with its own reason; and the standing, the recent history and the walk-away verdict are new information the old block never carried. Everything above `Coordination:` is byte-identical.

### `golden_build.txt`

```diff
 ── Colony Complete ──
-This is the final phase. Seal the colony after continue.
+This is the last phase in the plan. Once its work is checked, the project can be
+signed off as finished.
+━━ 📊 W H A T   N E X T ━━
+── Where things stand ── / ── What changed ── (new)
 ━━ 🐜 N E X T   U P ━━
-Run `/ant-continue` after the work is implemented and independently verified.
-Alternative: Run `/ant-status` if you want to inspect progress before advancing.
-Handoff saved (.aether/HANDOFF.md) — safe to clear your context now. Run `/ant-resume` to restore.
+Run `/ant-continue` — Phase 1 has produced work that has not been checked yet. The next step runs the checks and, if they pass, moves on to the following phase.
+Alternative: Run `/ant-status` — Look at the dashboard first, without changing anything.
+Alternative: Run `/ant-resume-colony` — Reload the fuller picture: the saved notes, the open questions and the task list.
+This is a natural break, and everything is written down -- it is safe to close this chat, and starting a fresh one from here will work better than carrying this one on.
```

*Why each change is a change in advice:* "Seal the colony after continue" used two words this repository invented with no explanation, and is now a plain sentence (S-05 — this is one of the two jargon fixes this plan made in its own seven). The recommendation is the same command with its reason. The walk-away sentence moved from `renderContextClearGuidance` to the card's — the rule is unchanged (the claim still requires the handover note to be seen on disk), the wording is the shared one. The cost block still sits last on the screen.

### `golden_continue.txt`

```diff
+━━ 📊 W H A T   N E X T ━━ + Where things stand + What changed (new)
 ━━ 🐜 N E X T   U P ━━
-Run `/ant-build 2` to start the next phase
-Handoff saved (.aether/HANDOFF.md) — safe to clear your context now. Run `/ant-resume` to restore.
+Run `/ant-build 2` — Phase 2 (Next golden phase) is ready to start. Helpers will write the code for it and report back before anything is accepted.
+Alternative: Run `/ant-focus "area"` / `/ant-status` / `/ant-pheromones` (each with its reason)
+Everything needed to pick this back up is written down, so it is safe to close this chat.
```

*Note:* `golden_continue.txt` was **already stale on this plan's base commit**. Plan 197-02 made `nextUpSuggestionsForState` an adapter over the resolver, which changed this line from "Run `/ant-build 2` to start the next phase" to the resolver's sentence, and left the transcript unrefreshed (its summary records deliberately not touching goldens). The first line of this diff is therefore 197-02's change, not this plan's; the rest is.

## Files Created/Modified

- `cmd/codex_visuals.go` — the four wiring helpers (`lifecycleNextAction`, `lifecycleNextActionForState`, `closeLifecycleCommand`, `closeLifecycleRun`, `renderLifecycleClosing`/`ForState`), the override extractor `lifecycleOverrideFromResult`, `lifecycleCommandInProse`, and every migrated closing in init / colonize / plan / build / continue
- `cmd/ceremony_cmd.go` — the wrapper closeout: the card, the completion report section, and the handoff wording
- `cmd/discuss.go` — both closings; the answer folded into both result maps, with the unblocked run's command fed in as an override
- `cmd/init_cmd.go`, `cmd/codex_workflow_cmds.go`, `cmd/codex_colonize_finalize.go`, `cmd/codex_plan_finalize.go` — the answer folded into each command's result
- `cmd/codex_build.go`, `cmd/codex_build_finalize.go`, `cmd/codex_continue.go`, `cmd/codex_continue_plan.go`, `cmd/codex_continue_finalize.go` — `closeLifecycleRun` at the end of each result builder, after everything that can still add a more specific command
- `cmd/next_action.go` — the `Override` input and its branch; the event-line fix (see Deviations)
- `cmd/next_action_card.go` — `nextActionFromResult`
- `cmd/lifecycle_card_startup_test.go`, `cmd/lifecycle_card_workloop_test.go`, `cmd/lifecycle_card_agreement_test.go` — the three test files

## Decisions Made

**1. Only three situations count as a run's own knowledge.**
`lifecycleOverrideFromResult` returns a command in exactly three cases: a part-finished build's redispatch, the exact command a blocked check named, and an unanswered question blocking the run. The tempting shortcut was to feed each result's existing `next` key in as the override, which would have made the two agree everywhere by construction — and neutered the decision, because the ordinary `next` is the generic "check it next". A paused project closing out of a build would then have been told to carry on regardless, which is precisely the drift 197-02 fixed for the closeout.

**2. The legacy `next` key was left alone.**
Something downstream reads it. For the three override cases it now equals the card's command by construction (asserted). Elsewhere it keeps whatever it said.

**3. Host-dispatch and plan-only instructions became a report, not advice.**
Four closings — colonize's manifest branch, plan's two manifest branches, build's and continue's plan-only screens — had a Next Up block whose content was an instruction to the *platform running the command*, not to the owner. Those moved into a "How this run is being driven" section above the card, rewritten in plain English. The card below then answers the owner's own question. Nothing was lost; the finalizer command each names is still on screen.

**4. A command carrying a fill-in-the-blank is never recommended.**
`lifecycleCommandInProse` refuses a command containing `<` or `>`. A completion packet whose sentence says "rerun `aether build-finalize 1 --completion-file <file>`" gets that sentence shown as a report ("What the last step reported"), and the card recommends something the owner can actually type. This is the same defect 197-02 found in the closeout's `aether build <phase>` placeholder.

**5. The check on wording is scoped to the closing block.**
`TestTheSevenClosingsSpeakPlainEnglish` holds the block from the "What Next" banner down. The banners and section headings above it ("Colony Init", "Colony Complete", the birth ceremony) are the project's deliberate house style, locked by their own tests and by the platform-parity and regression transcripts plan 197-03 owns in this wave.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] The card printed the project's internal record-keeping to the owner**

- **Found during:** Task 2, when the recorded transcripts were first refreshed.
- **Issue:** the card's "What changed" section printed saved event lines verbatim. Events are stored as `timestamp|event_type|source|message`, so the owner was shown `2026-08-28T20:38:55Z|plan_generated|plan|Generated 6 active phases...`. Two faults in one: it is unreadable to a non-technical reader (S-05), and the timestamp is a value that differs on every run, which no recorded transcript can hold — the first refreshed `golden_plan.txt` would have failed on its very next run.
- **Fix:** `nextActionEventSentence` keeps the message and drops the bookkeeping. Applied in the resolver so the machine-readable answer carries the readable form too.
- **Files modified:** `cmd/next_action.go` (not in this plan's declared files).
- **Verification:** the three transcripts refreshed and then re-run twice with `-count=2` without the update flag — stable.
- **Commit:** `f0b78843`

**2. [Rule 2 — Missing critical] The wrapper closeout dropped what the finished step reported**

- **Found during:** Task 2.
- **Issue:** `renderCeremonyCloseout` puts the completion packet's own `completion_next` sentence into `next`, and that sentence was the *only* place the finalizer's own instruction appeared. Replacing the Next Up block with the card would have deleted it — including the failed-finalizer path's "rerun `aether build-finalize 1 --completion-file <file>`", which is the only route out of that failure.
- **Fix:** a "What the last step reported" section above the card, plus feeding a real (placeholder-free) command out of that sentence into the resolver as the override.
- **Files modified:** `cmd/ceremony_cmd.go`, `cmd/codex_visuals.go`.
- **Verification:** `TestCeremonyCloseoutBlockedPathRendersBlockedNotCompletion` and `TestCeremonyCloseoutFailedFinalizerRendersFailureNotCompletion` pass unchanged.
- **Commit:** `f0b78843`

**3. [Rule 3 — Blocking] Four test files outside the declared surface assert the old wording**

- **Found during:** Tasks 1 and 2.
- **Issue:** `cmd/codex_visuals_test.go`, `cmd/continue_wrapper_ceremony_test.go`, `cmd/phase_end_footer_test.go` and `cmd/golden_workflow_test.go` assert the exact sentences this plan is required to replace — "safe to clear your context now.", "This is the final phase. Seal the colony after continue.", "All planned phases are complete. The colony is ready for Crowned Anthill.", "Worker handoffs recorded for phase 1".
- **Fix:** each assertion updated to the card's equivalent sentence. **No check was weakened.** The walk-away contract is asserted exactly as before — the claim requires the handover note to be verifiably on disk, the absent case is an explicit hold — and one assertion was strengthened: `TestContinueWrapperCeremonyContract`'s blocked case previously only checked that the blocked screen carried *no* guidance, and now removes the handover note and requires the blocked screen to hold the owner back and to say so.
- **One fixture was corrected rather than its assertion:** `TestBuildCloseoutHandoffSectionHouseStyle` wrote a `COLONY_STATE.json` with no goal, which the runtime reads as a leftover file rather than a project. The fixture now carries the goal the runtime always writes.
- **Commit:** `f0b78843`

**4. [Rule 3 — Blocking] `cmd/next_action.go` and `cmd/next_action_card.go` are not in this plan's declared files**

- **Issue:** the plan requires the override to reach the card "through the resolver, not around it", which cannot be done without adding the input to the resolver; and a renderer handed only a result map cannot render the answer the command folded into it without a reader for it.
- **Fix:** `nextActionInput.Override` + one branch in `chooseNextAction` (`next_action.go`), and `nextActionFromResult` (`next_action_card.go`). Both are additive; no existing branch was changed.
- **Verification:** `TestEveryDeciderAgreesOnTheNextCommand`, `TestResolveNextActionCoversEveryLifecycleState`, `TestEveryResolverCandidateResolves`, `TestNoCommandIsSpelledInlineAtABranch` and `TestResolverCommandsArePlatformNeutral` all pass unchanged.

**5. [Rule 3 — Blocking] `cmd/codex_build.go`, `cmd/codex_continue.go`, `cmd/codex_continue_plan.go` and `cmd/codex_workflow_cmds.go` are not in the declared list**

- **Issue:** the plan's declared file list names the *finalizers* for build and continue, but the direct lane's result maps are built in `codex_build.go` / `codex_continue.go` / `codex_continue_plan.go`, and the direct-lane commands live in `codex_workflow_cmds.go`. Wiring only the finalizers would have left `aether build` and `aether continue` — the two commands the owner actually runs — with a card and an envelope resolved separately.
- **Fix:** one `closeLifecycleRun` call at the end of each result builder.
- **Files modified:** four files, one line plus a comment each.

### Scope boundaries observed

- `cmd/hook_cmds.go`, `.claude/settings.json` and the three COMMAND transcripts (`command_catalog.json`, `parity_snapshot.json`, `regression_snapshot.json`) — untouched, confirmed by `git diff --name-only` against this plan's base.
- `cmd/status.go`, `cmd/recover_visuals.go`, `cmd/session_flow_cmds.go`, `cmd/update_cmd.go`, the sealing renderers and the pause card — untouched; plan 197-06's.
- `.planning/STATE.md` and `.planning/ROADMAP.md` — untouched.

---

**Total deviations:** 5 (1 bug fixed, 1 missing-critical fixed, 3 blocking file-scope)
**Impact on plan:** no scope creep. Two of the five are genuine defects the migration exposed the moment the card reached a screen; the other three are files the plan's own required behaviour cannot be delivered without.

## Owner-facing jargon found in these seven

Asked explicitly. **Yes — two, both fixed:**

1. The build screen's last-phase line read *"This is the final phase. Seal the colony after continue."* — "seal" and "colony" are both words this repository invented, neither explained. Now: *"This is the last phase in the plan. Once its work is checked, the project can be signed off as finished."*
2. The check's completion line read *"All planned phases are complete. The colony is ready for Crowned Anthill."* — a stage name that means nothing outside this repo. Now: *"Every phase in the plan is finished. The project is ready to be signed off as complete -- the stage this project calls Crowned Anthill."*

Three more were rewritten while migrating rather than found as defects: the wrapper closeout's handoff lines (*"Worker handoffs recorded for phase 1 — the next phase's workers inherit this build's context"* → *"The notes this build's helpers left were saved for phase 1, so the next phase's helpers start from what was already learned"*), and the two host-dispatch instructions, which now say who does what in ordinary words.

**Not fixed, and reported rather than changed:** the banners and ceremony headings above the closing block — "Colony Init", "Colonize", "Colony Complete", "Colony State", "Queen has set the colony's intention", "Colony Status: READY". These are the project's deliberate house style (CLAUDE.md's UX Architecture section), they are locked by their own tests, and rewording them would churn the platform-parity and regression transcripts that plan 197-03 owns in this wave. They are outside this plan's declared surface.

## Issues Encountered

None beyond the deviations above.

## Known Stubs

None.

## Threat Flags

None. This plan adds no network endpoint, no authentication path, no file-access pattern and no schema at a trust boundary. It changes what commands say at the end and adds keys to existing result maps.

## Verification Run

| Command | Result |
|---|---|
| `go test ./cmd -run 'Test(StartupLifecycle\|WorkLoop\|MigratedLifecycleSurfaces\|MigratedStartupClosingsSayItOnce\|MigratedWorkLoopClosingsSayItOnce\|TheStartingCardNoLongerRecitesAFixedTrio\|BlockedCheckStillExplainsItself\|RedispatchOverrideReachesBothTheCardAndTheEnvelope\|EveryCommandTheSevenCanRecommendResolves\|NextUpIsTheOnlyNextUpFunnel\|HumanFacingOutputGoesThroughWriteVisualOutput\|VisualOutputNeverLeaksRawWrapperCommands)' -count=1` | ok (5.8s) |
| `go test ./cmd -run 'Test(CodexBuildFinalize\|CodexContinueFinalize\|CodexPlanFinalize\|CodexColonize\|CeremonyCmd\|Closeout\|Discuss\|BuildWrapperCeremony\|ContinueWrapperCeremony)' -count=1` | ok |
| `go test ./cmd -run 'Test(MigratedLifecycleSurfacesAgree\|MigratedLifecycleSurfacesArePlatformCorrect\|TheSevenClosingsSpeakPlainEnglish\|NextUpTranslatesWrapperCommandsPerPlatform\|RenderNextUpAppliesTranslationAtTheFunnel\|VisualOutputNeverLeaksRawWrapperCommands)' -count=1` | ok |
| `go test ./cmd -run 'TestGolden(Plan\|Build\|Continue)VisualOutput' -count=2` | ok — the refreshed transcripts are stable across repeated runs |
| `go test ./cmd -run 'Test(Continue\|Build\|Plan\|Colonize\|Seal\|Entomb\|Init\|Discuss\|Ceremony\|Closeout\|Visual\|Hint\|Parity\|Catalog\|Workflow\|Host\|Truth\|Envelope\|Compat\|Golden\|Orchestrator\|Boundary\|Status\|Recovery\|Resume\|Pause\|NextAction\|NextUp\|Command)' -count=1` | ok (96s) |
| `go test ./cmd -run 'Test(Session\|Hook\|Run\|Autopilot\|Swarm\|Quick\|Watch\|Spend\|Phase\|Flag\|Signal\|Wrapper\|Skill\|Handoff\|Job\|Worktree\|Verification)' -count=1` | ok (26s) |
| `go build ./...` | clean |
| `go vet ./cmd/` | clean |
| `gofmt -l cmd/ pkg/` | no output |

The full `go test ./cmd` run is the orchestrator's after merge-back, per this executor's instructions. No CLI flag or subcommand was added or removed, so the three COMMAND transcripts needed no refresh — confirmed untouched by `git diff --name-only`.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- **197-06** inherits four reusable wiring helpers and a live example of each shape: a renderer holding only a result map (`renderLifecycleClosing`), one holding the state as well (`renderLifecycleClosingForState`), a command that has already saved its work (`closeLifecycleCommand`), and a result builder with a run-specific command to feed in (`closeLifecycleRun`). Its remaining surfaces — status, recover, the session flows, update, the sealing renderers and the pause card — follow the same four patterns.
- **The pause card's recorded wording defect is still open** and is 197-06's, exactly as 197-02 left it: `.planning/WINDOWS.md` records that the pause card says "Colony handoff saved for later resumption" and carries a `P A U S E   C O L O N Y` banner. Untouched here.
- **197-07** can generalise `TestMigratedLifecycleSurfacesAgree`: adding a surface to `migratedSurfaceRenderings` is the whole cost of bringing it under the invariant.
- **One thing 197-06 should know:** `migratedSurfaceRenderings` deliberately passes bare result maps so every surface falls back to the same resolve. A surface that resolves its answer from something other than the saved project will pass that test and still be wrong; the per-surface tests in the other two files are what catch that.

## Self-Check: PASSED

**Files claimed as created — all present on disk:**

| File | Present |
|---|---|
| `cmd/lifecycle_card_startup_test.go` | yes |
| `cmd/lifecycle_card_workloop_test.go` | yes |
| `cmd/lifecycle_card_agreement_test.go` | yes |
| `.planning/phases/197-one-answer-to-what-next/197-04-SUMMARY.md` | yes |

**Commits claimed — all present in `git log`:** `3a46eaa3`, `50678f4c`, `f0b78843`, `e078d20b`, `d36bc51c`, plus this summary commit.

**Nothing changed outside the declared surface.** `git diff --name-only` against this plan's base commit (`98195e1a`) lists 24 files: the 14 source files and 3 transcripts named above, the 3 new test files, and the 4 existing test files updated for the new wording. `cmd/hook_cmds.go`, the three COMMAND transcripts, `.planning/STATE.md` and `.planning/ROADMAP.md` do not appear in it.

**Temporary demonstration reverted:** the `renderColonizeVisual` revert used for the failure demonstration was undone with `git checkout --`, and `git status --short` was clean afterwards.

---
*Phase: 197-one-answer-to-what-next*
*Completed: 2026-08-28*
