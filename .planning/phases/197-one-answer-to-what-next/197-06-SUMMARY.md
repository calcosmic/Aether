---
phase: 197-one-answer-to-what-next
plan: 06
subsystem: cli
tags: [go, next-action, closing-card, envelope, lifecycle, cobra-alias, jargon]

requires:
  - phase: 197-one-answer-to-what-next
    provides: "resolveNextAction and the availability gate (197-01); renderNextActionCard and applyNextActionToResult (197-02); the four wiring helpers and the seven-surface consistency test (197-04); pause as the canonical Cobra command name (197-05)"
provides:
  - "Six remaining lifecycle surfaces -- pausing, resuming (both forms), sealing (both forms), updating (both outcomes), recovering and status -- render renderNextActionCard instead of a hand-written block"
  - "Each folds the same resolved answer into its result map, so the screen and the machine-readable envelope cannot name different commands"
  - "candidateResumeDashboard (\"aether resume-dashboard\") -- the genuinely distinct read-only quick view the pause card's duplicate alternative always meant"
  - "resume_override_command/_why -- the resume dashboard's two special facts (a durable worker result waiting to finalize; a build process verified still running) reach the resolver as inputs"
  - "recoverOverrideFromIssues / recoverNextAction -- the sixth and last hand-rolled next-step decider in the runtime, retired"
  - "statusActiveWorkers / statusOverrideFacts -- status's two overrides (workers in flight, a guided action) computed once and threaded through both the JSON envelope and the screen"
  - "migratedSurfaceRenderings extended to thirteen renderings across all eleven NEXT-02 commands"
affects: [197-07 cross-command envelope assertion, NEXT-02 requirement traceability]

actuals:
  tokens: 17800
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Cobra-alias-aware duplicate detection: a card's alternatives are deduped by resolving each spelling against the live command tree (rootCmd.Find), not by comparing the literal strings -- two different spellings of a declared Cobra alias pair are the same command"
    - "Non-jargon LastCommand: a command's own name (\"seal\", \"resume-colony\") is never fed to the resolver as the what-changed fact when that name itself is a word this repo invented; a plain phrase (\"signing the project off as finished\", \"picking the project back up\") is passed instead so the fact never needs the word explained a second time"
    - "Report above, decide below (continued from 197-04): status's in-flight-workers watch-progress tips and seal's host-dispatch/finalizer instruction stay a report above the card; only the recommendation itself comes from the resolver"
    - "Same fact, two folds: buildResumeDashboardResult resolves and folds an answer under its own name for the compact dashboard, and resumeColonyCmd re-folds under its own name afterward, carrying the same override forward rather than dropping it on an unrelated second resolve"

key-files:
  created:
    - cmd/lifecycle_card_session_test.go
    - cmd/lifecycle_card_endgame_test.go
  modified:
    - cmd/codex_visuals.go
    - cmd/session_flow_cmds.go
    - cmd/update_cmd.go
    - cmd/status.go
    - cmd/recover_visuals.go
    - cmd/seal_final_review.go
    - cmd/codex_workflow_cmds.go
    - cmd/context.go
    - cmd/next_action.go
    - cmd/next_action_card_test.go
    - cmd/codex_visuals_test.go
    - cmd/ceremony_restoration_test.go
    - cmd/recover_test.go
    - cmd/lifecycle_card_agreement_test.go
    - .planning/WINDOWS.md

key-decisions:
  - "The pause card's duplicate alternative was a resolver-level bug, not only a rendering one: resume-colony's own Cobra command declares resume as its Aliases entry, so 'aether resume' and 'aether resume-colony' were always the identical *cobra.Command -- the resolver's own state.Paused branch offered candidateResume as primary and candidateResumeColony as an alternative, which is the same command twice however it is spelled. Fixed at the source (chooseNextAction's Paused branch), not papered over in the card."
  - "candidateResumeDashboard, not a rewritten candidateResume: rather than repoint the existing 'aether resume' candidate at the read-only dashboard (which would change what the PRIMARY recommendation does when paused), a new candidate was added for the genuinely distinct quick view -- aether resume-dashboard, the command the pause card's own wording ('the quick version -- without the detail') already meant and the resume.md wrapper already documents as the correct read-only alternative."
  - "resume-dashboard is deliberately absent from migratedSurfaceRenderings (Task 3's consistency map): its real content depends on session.json, learning-observations.json and several other files the shared fixture does not seed. It has full, dedicated screen-and-envelope coverage in lifecycle_card_session_test.go over its own fixture instead, which is more honest than a bare-map rendering that would prove less than it looks like it proves."
  - "Status's JSON and visual paths are unified on ONE buildStatusResult call: previously they were two independent code paths (`outputOK(buildStatusResult(...))` vs `renderDashboard(state, store)` built its own advice from scratch). statusCmd now calls buildStatusResult once and passes the same result map to both outputOK and renderDashboard, so the override facts (in-flight workers, guided actions) cannot reach one and not the other."
  - "recoverOverrideFromIssues returns no command for the two 'review the ... above' default cases (an unrecognized critical/warning category): the findings are already listed above the card, and there is nothing more specific to run than the ordinary answer the resolver already gives for this project."
  - "A command's own name is never fed to the resolver as LastCommand when that name is itself a word this repo invented (S-05): both 'seal' and 'resume-colony' tripped the plain-English checker inside the 'what changed' sentence, discovered by the plain-English test added in this plan and fixed by passing a self-explanatory phrase instead of the literal command name."

patterns-established:
  - "Deduping a card's alternatives against the LIVE Cobra tree, not against literal strings, is the only correct check for a duplicate command once aliases exist -- codified as closingCommandCobraTarget in next_action_card_test.go, reusable by any future card test."

requirements-completed: []

# NEXT-02 is declared by four plans in this phase (197-02, 197-04, 197-06, 197-07).
# Per the shared-ID gate (#2388), it stays Pending in REQUIREMENTS.md until 197-07
# -- the last plan declaring it -- also finishes. This plan completes the final six
# of the eleven commands NEXT-02 names (pause, resume, seal, update, recover,
# status); the other five (init, discuss, plan, build, continue) were completed by
# 197-04. All eleven now render from the resolver; the requirement itself is left
# for 197-07 to mark, per the gate.

coverage:
  - id: D1
    description: "Pausing ends with the card, and its two alternatives are genuinely distinct commands rather than the same command described twice."
    requirement: NEXT-02
    verification:
      - kind: integration
        ref: "cmd/lifecycle_card_session_test.go#TestSessionLifecycleCardsComeFromTheResolver/pausing"
        status: pass
      - kind: unit
        ref: "cmd/next_action_card_test.go#TestNoCardOffersTheSameCommandTwice/the_pause_card"
        status: pass
    human_judgment: false
  - id: D2
    description: "Resuming ends with the card in both its compact (resume-dashboard) and full (resume-colony) form, and the resume dashboard's two special facts -- a durable worker result waiting to finalize, or a build genuinely still running -- survive the migration intact."
    requirement: NEXT-02
    verification:
      - kind: integration
        ref: "cmd/lifecycle_card_session_test.go#TestSessionLifecycleCardsComeFromTheResolver/resuming,_full_form"
        status: pass
      - kind: integration
        ref: "cmd/lifecycle_card_session_test.go#TestSessionLifecycleCardsComeFromTheResolver/resuming,_compact_form"
        status: pass
    human_judgment: true
    rationale: "The two override branches (finalize-pending, build-still-running) are exercised structurally through resume_override_command/_why reaching lifecycleOverrideFromResult, but no fixture in this plan drives an actual live build process or a durable pending completion packet through the full resume command end to end -- that requires infrastructure (a real process, a real completion file) beyond this plan's scope."
  - id: D3
    description: "Updating ends with the card, including when it repaired a missing command copy and when it found nothing to do -- the run's own outcome is a fact fed to the resolver as \"what changed\", not appended after the card as a second, separate line."
    requirement: NEXT-02
    verification:
      - kind: integration
        ref: "cmd/lifecycle_card_session_test.go#TestUpdateEndsWithTheCard"
        status: pass
      - kind: integration
        ref: "cmd/lifecycle_card_session_test.go#TestUpdateEnvelopeCarriesTheCardsFields"
        status: pass
    human_judgment: false
  - id: D4
    description: "Sealing ends with the card, and the card explains in ordinary words what finishing a project means and what archiving it would do (S-05)."
    requirement: NEXT-02
    verification:
      - kind: integration
        ref: "cmd/lifecycle_card_endgame_test.go#TestSealEndsWithTheCard"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_card_endgame_test.go#TestSealCardExplainsFinishingAndArchiving"
        status: pass
      - kind: integration
        ref: "cmd/lifecycle_card_endgame_test.go#TestSealEnvelopeMatchesCard"
        status: pass
    human_judgment: false
  - id: D5
    description: "Recovering ends with the card, its findings still appear above it, and the recovery view's own next-step function -- the sixth and last hand-rolled decider -- returns the resolver's answer, fed this scan's own top finding as an override."
    requirement: NEXT-02
    verification:
      - kind: integration
        ref: "cmd/lifecycle_card_endgame_test.go#TestRecoverEndsWithTheCard"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_card_endgame_test.go#TestRecoveryNextStepComesFromTheResolver"
        status: pass
    human_judgment: false
  - id: D6
    description: "Status ends with the card. When workers are still running, it still says to wait and still names a way to watch progress. When a guided action exists, it leads the advice. Both overrides reach the JSON envelope AND the screen from one decision -- an override applied to only one of them is the failure this test catches."
    requirement: NEXT-02
    verification:
      - kind: integration
        ref: "cmd/lifecycle_card_endgame_test.go#TestStatusEndsWithTheCard"
        status: pass
      - kind: integration
        ref: "cmd/lifecycle_card_endgame_test.go#TestStatusOverridesReachBothTheCardAndTheEnvelope/workers_in_flight"
        status: pass
      - kind: integration
        ref: "cmd/lifecycle_card_endgame_test.go#TestStatusOverridesReachBothTheCardAndTheEnvelope/guided_action"
        status: pass
    human_judgment: false
  - id: D7
    description: "Driven over one saved project, all thirteen migrated renderings (the eight from wave 3 plus pausing/updating/sealing/recovering/status added here) name the same command; a single surface reverted to its old block makes the test fail by name. Demonstrated on the pause card and reverted."
    verification:
      - kind: unit
        ref: "cmd/lifecycle_card_agreement_test.go#TestMigratedLifecycleSurfacesAgree"
        status: pass
      - kind: manual_procedural
        ref: "revert demonstration: renderPauseVisual given back a hand-written block, TestMigratedLifecycleSurfacesAgree FAILED naming \"pausing the project\", revert undone"
        status: pass
    human_judgment: false
  - id: D8
    description: "On each recognised platform the newly migrated surfaces show that platform's spelling while the executed value stays in the runtime form (S-01), and every command any of the thirteen surfaces can put in front of the owner resolves against the live command tree."
    verification:
      - kind: unit
        ref: "cmd/lifecycle_card_agreement_test.go#TestMigratedLifecycleSurfacesArePlatformCorrect"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_card_agreement_test.go#TestEveryCommandTheMigratedSurfacesRenderResolves"
        status: pass
    human_judgment: false

duration: 172 min
completed: 2026-08-29
status: complete
---

# Phase 197 Plan 06: Six Remaining Lifecycle Surfaces Render the One Card Summary

**Pausing, resuming (both forms), sealing (both forms), updating (both outcomes), recovering and status now end with the single card the resolver produces; the pause card's real duplicate-command defect (two spellings of the same aliased Cobra command) is fixed at the resolver, not papered over; and the last hand-rolled next-step decider in the runtime is retired.**

## Performance

- **Duration:** 172 min
- **Tasks:** 3
- **Files created:** 2
- **Files modified:** 15 (14 source/test, 1 planning doc)

## Accomplishments

- **All eleven NEXT-02 commands now render from the one resolver.** Wave 3 (197-04) migrated init, discuss, plan, build and continue; this plan finishes pause, resume (compact and full), seal (ordinary and plan-only), update (no-op and repaired), recover and status.
- **The pause card's duplicate-command defect was a resolver bug, not a rendering one, and is fixed at the source.** `resumeColonyCmd`'s Cobra command declares `Aliases: []string{"resume"}` — "aether resume" and "aether resume-colony" have always been the exact same `*cobra.Command`. The resolver's own `state.Paused` branch offered `candidateResume` as primary and `candidateResumeColony` as an alternative: the same command, twice, regardless of the card rendering it. Fixed by adding `candidateResumeDashboard` (`aether resume-dashboard`, the genuinely distinct read-only quick view already documented in the `resume.md` wrapper and already implied by the old card's own "without the detail" wording) and removing the alias-duplicate from the Paused branch's alternatives.
- **`TestNoCardOffersTheSameCommandTwice` (197-02's own test) was strengthened, not duplicated.** It compared literal strings, which cannot see that two different spellings resolve to one Cobra command — the pause card's real defect shipped past it once wording drifted away from a literal repeated string. `closingCommandCobraTarget` now resolves each command through `rootCmd.Find` and dedupes on the `*cobra.Command` it names. Demonstrated failing against the pre-change tree, naming the pause card exactly (see Task Commits below).
- **The sixth and last hand-rolled next-step decider is gone.** `recoverNextStep` (recover.go's own private decision, unconnected to `resolveNextAction`) is replaced by `recoverOverrideFromIssues` + `recoverNextAction`, used by both the text report and the JSON envelope, including the previously-blank healthy-colony case.
- **Status's JSON and visual output paths were two independent code paths before this plan** — `outputOK(buildStatusResult(...))` for JSON, `renderDashboard(state, store)` computing its own advice from scratch for the screen. They now share one `buildStatusResult` call whose two overrides (in-flight workers, a guided action) are computed once and folded into the result both paths read.
- **A real plain-English gap, caught by the tests this plan wrote and fixed at the source:** feeding a command's own name to the resolver as the "what changed" fact puts a repo-invented word in a sentence with nothing else there to explain it. `seal` and `resume-colony` both tripped `untranslatedRepoWords`; both now pass a plain phrase ("signing the project off as finished", "picking the project back up") instead of the literal command name.
- **`TestMigratedLifecycleSurfacesAgree` extended from eight renderings to thirteen** (pausing, updating, sealing, recovering, status added; resuming deliberately left to its own dedicated fixture — see Decisions). Demonstrated failing on a reverted pause card and reverted.

## Task Commits

1. **Task 1 RED: pause/resume/update tests** — `5b80623a` (test)
2. **Task 1 GREEN: pause/resume/update render the card** — `a92f3c4e` (feat)
3. **Task 2 RED: seal/recover/status tests** — `39c5717b` (test)
4. **Task 2 GREEN: seal/recover/status render the card** — `52ed0d1e` (feat)
5. **Task 3: extend the consistency test to all thirteen surfaces** — `7e7a516f` (test)

### Recorded RED output (Task 1's required demonstration)

`TestNoCardOffersTheSameCommandTwice/the_pause_card`, run against the pre-change tree (the `next_action.go` Paused-branch fix stashed, everything else in place):

```
--- FAIL: TestNoCardOffersTheSameCommandTwice/the_pause_card
    the pause card offers "aether resume-colony" and "aether resume" -- both
    resolve to the exact same command (resume-colony) on the live command tree,
    described as if they were different choices:
    ...
    Run `aether resume-colony` when you want to pick this project back up. It
    reloads the full picture: the saved notes, the open questions and the task
    list.
    Alternative: Run `aether resume` for the quick version -- where things
    stand, without the detail.
```

The rest of Task 1's suite (resuming, updating) also failed at this point — `resuming, full form` and `resuming, compact form` printed the old hand-written `computeNextAction` block instead of the card; `update` printed no `What Next` card at all. Recorded via `go test ./cmd -run 'TestSessionLifecycleCardsComeFromTheResolver|...' -v` before any implementation existed.

### Task 3's revert demonstration

Acceptance criterion: *reverting any single surface to its old block makes the consistency test fail by name.* `renderPauseVisual`'s closing was temporarily reverted to a hand-written `renderNextUp` call:

```
--- FAIL: TestMigratedLifecycleSurfacesAgree
    lifecycle_card_agreement_test.go:125: pausing the project tells the owner
    to run "aether resume"; the one answer for this project is "aether
    build 1" -- the surfaces have separated
```

Reverted by hand (the edit was a single-line change, verified byte-identical to the committed version via `git status --short cmd/codex_visuals.go` showing clean before re-running the full suite).

## Files Created/Modified

- `cmd/codex_visuals.go` — `renderPauseVisual`, `renderResumeVisual`, `renderUpdateVisual`, `renderSealVisual` all end with `renderLifecycleClosing`/render the card; `lifecycleOverrideFromResult` gained a `resume_override_command`/`_why` branch
- `cmd/session_flow_cmds.go` — `pauseColonyCmd` and `resumeColonyCmd` fold the answer via `closeLifecycleCommand` before rendering
- `cmd/update_cmd.go` — `runUpdate`'s non-dry-run path folds the answer with `updateLastCommandFact`, replacing the old post-hoc string append after the closing block
- `cmd/status.go` — `buildStatusResult` computes `statusActiveWorkers`/`statusOverrideFacts` and folds the answer once; `statusCmd` and `renderDashboard` both read the same result map
- `cmd/recover_visuals.go` — `recoverNextStep` replaced by `recoverOverrideFromIssues`/`recoverNextAction`, used by `renderRecoverDiagnosis` (including the healthy-colony branch, previously blank) and `renderRecoverJSON`
- `cmd/seal_final_review.go` — `runSealPlanOnly` folds the answer via `closeLifecycleRun`; `renderSealPlanOnlyVisual`'s dispatch instruction moved to a "How this run is being driven" report above the card
- `cmd/codex_workflow_cmds.go` — the real seal command folds and renders the card
- `cmd/context.go` — `buildResumeDashboardResult` computes and folds the two resume-specific overrides
- `cmd/next_action.go` — `candidateResumeDashboard` added; the Paused branch's alternatives fixed
- `cmd/next_action_card_test.go`, `cmd/codex_visuals_test.go`, `cmd/ceremony_restoration_test.go`, `cmd/recover_test.go` — updated for the strengthened duplicate check and the new function/renderer signatures
- `cmd/lifecycle_card_session_test.go`, `cmd/lifecycle_card_endgame_test.go` — the two new test files
- `cmd/lifecycle_card_agreement_test.go` — extended to thirteen renderings, plus `TestEveryCommandTheMigratedSurfacesRenderResolves`
- `.planning/WINDOWS.md` — item 3 (the pause card's "Colony handoff saved" jargon) marked fixed; this plan's own line fix resolved it

## Decisions Made

See `key-decisions` in the frontmatter for full rationale. In short: the pause duplicate is fixed at the resolver (a new distinct candidate, not a rewritten one); resume-dashboard is excluded from the shared consistency map in favour of its own dedicated, realistic fixture; status's JSON and visual paths are unified on one `buildStatusResult` call; recover's "review the ... above" defaults deliberately carry no override command; and a command's own name is never fed to the resolver as the "what changed" fact when that name is itself repo jargon.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] The resolver's own Paused branch recommended the same command twice**

- **Found during:** Task 1, writing the RED test for the pause card.
- **Issue:** `chooseNextAction`'s `state.Paused` branch offered `candidateResume` ("aether resume") as primary and `candidateResumeColony` ("aether resume-colony") as an alternative. `resumeColonyCmd` declares `Aliases: []string{"resume"}`, so both strings resolve to the identical `*cobra.Command` — a resolver-level instance of the exact defect the plan asked the card to fix, not only a rendering-level one.
- **Fix:** Added `candidateResumeDashboard` (`aether resume-dashboard`) and swapped it in for `candidateResumeColony` in the Paused branch's alternatives.
- **Files modified:** `cmd/next_action.go` (declared).
- **Verification:** `TestNoCardOffersTheSameCommandTwice`, `TestSessionLifecycleCardsComeFromTheResolver`.
- **Commit:** `a92f3c4e`

**2. [Rule 1 — Bug] Feeding a command's own name as LastCommand could put unexplained jargon in "what changed"**

- **Found during:** Task 2 (seal), then confirmed for resume-colony while preparing Task 3's extended surface list.
- **Issue:** `changedFromInput` renders `"The last thing you ran was " + LastCommand + "."` verbatim. `seal` and `colony` (inside "resume-colony") are both words CLAUDE.md's vocabulary table requires explaining in the same sentence; a bare command name has no explanation attached.
- **Fix:** Pass a plain phrase instead of the literal command name for these two: `"signing the project off as finished"` (seal, and its plan-only variant `"getting ready to sign the project off as finished"`), `"picking the project back up"` (resume-colony).
- **Files modified:** `cmd/codex_workflow_cmds.go`, `cmd/seal_final_review.go`, `cmd/session_flow_cmds.go` (all declared or already-touched files).
- **Verification:** `TestSealCardExplainsFinishingAndArchiving`, `TestTheSevenClosingsSpeakPlainEnglish` (extended).
- **Commit:** `52ed0d1e`, `7e7a516f`

**3. [Rule 3 — Blocking] Signature changes to renderSealVisual, renderUpdateVisual and recoverOverrideFromIssues required updating their existing call sites and tests**

- **Issue:** `renderSealVisual` and `renderUpdateVisual` needed a `result` parameter to fold the resolved answer through; `recoverNextStep(issues) string` needed to become `recoverOverrideFromIssues(issues, state) (string, string)` to feed the resolver's state-dependent phase number into `aether build %d`. Existing callers (`cmd/codex_workflow_cmds.go`) and existing tests (`cmd/codex_visuals_test.go`, `cmd/ceremony_restoration_test.go`, `cmd/recover_test.go`) are outside this plan's declared file list.
- **Fix:** Updated all call sites; updated the five existing tests that asserted the old hand-typed wording or old signatures to assert the new behaviour (the shared card's presence, or the tuple return) without weakening any check.
- **Files modified:** `cmd/codex_workflow_cmds.go`, `cmd/codex_visuals_test.go`, `cmd/ceremony_restoration_test.go`, `cmd/recover_test.go`.
- **Verification:** `go test ./cmd -run 'Test(SealFinalReview|Recover|Status|StatusUX|ProofCmd|RenderUpdateVisual|SealRendersCrownedAnthill)'`.
- **Commits:** `a92f3c4e`, `52ed0d1e`

**4. [Rule 3 — Blocking] Strengthening `TestNoCardOffersTheSameCommandTwice` (197-02's file) rather than writing a duplicate**

- **Issue:** A test of exactly this name already existed in `cmd/next_action_card_test.go` (197-02's declared file), asserting duplicate detection by literal string comparison — which cannot see the alias-pair defect this plan needed to catch (see Deviation 1).
- **Fix:** Extended the existing test with `closingCommandCobraTarget`, resolving each command against the live Cobra tree and deduping on the `*cobra.Command`; gave the pause-card fixture a real, saved, paused project (via a new `pauseCardFixture` helper) instead of a bare result map, since the defect only reproduces when the resolver actually reaches the Paused branch.
- **Files modified:** `cmd/next_action_card_test.go`.
- **Verification:** Recorded RED above; `TestNoCardOffersTheSameCommandTwice` passes post-fix.
- **Commit:** `5b80623a`, `a92f3c4e`

### Scope boundaries observed

- `cmd/recover.go` — untouched; only `cmd/recover_visuals.go` needed the decider retirement.
- `renderUpdateVisual`'s dry-run branch — left hand-written. It answers "how might you re-run update" (safe vs force, which flags), not "what should the project do next" — a different question the general resolver's candidate set cannot express, and neither the plan's behaviour section nor either new test file names it.
- `.planning/STATE.md` and `.planning/ROADMAP.md` — untouched, per worktree-mode convention (orchestrator's job).

---

**Total deviations:** 4 auto-fixed (2 bugs, 2 blocking file-scope)
**Impact on plan:** No scope creep. Both bugs are genuine defects the migration surfaced the moment a real test drove the affected surface; both file-scope deviations are direct, minimal, mechanically-required corollaries of the plan's own required behaviour (a state-dependent override, a stronger duplicate check).

## Issues Encountered

**`TestBuildDispatchStartsHeartbeatMonitor` fails only under the full `go test ./cmd` suite (~230–250s run), but passes reliably in isolation (`-count=3`, all green).** This is the exact same pre-existing flake 197-05's summary already documented and deferred — confirmed still present, still unrelated to this plan's files (none of which touch worker-dispatch heartbeat monitoring). Not fixed, per scope boundary; not re-logged to `deferred-items.md` since it is already recorded there.

One transient false failure during full-suite verification is worth recording for anyone re-running this: `TestCLIProviderBackedPlanRevisionJourney` failed once with "black-box command changed the source checkout / after: M .planning/WINDOWS.md" — caused by this executor editing `.planning/WINDOWS.md` by hand *while* a background full-suite run was in flight, which the test's before/after git-diff snapshot correctly caught as a checkout mutation. Not a defect in the test or in this plan's code; re-ran clean once the edit and the test run stopped overlapping, and the full suite was re-run once more start-to-finish with no concurrent edits to confirm.

## Known Stubs

None. Every card renders through the same `renderNextActionCard`/`renderLifecycleClosing` path already proven in wave 3, and every new candidate (`candidateResumeDashboard`) resolves against the live command tree like every other.

## Threat Flags

None. This plan changes what six commands say at the end and fixes a resolver-level advice bug; it adds no network endpoint, no auth path, no file-access pattern and no schema at a trust boundary.

## Verification Run

| Command | Result |
|---|---|
| `go test ./cmd -run 'Test(SessionLifecycleCardsComeFromTheResolver\|NoCardOffersTheSameCommandTwice\|SessionLifecycleEnvelopesMatchTheirCards\|SessionFlow\|UpdateCmd\|ResumeColony\|PauseColony\|CanonicalAliasDelegates)' -count=1` | ok |
| `go test ./cmd -run 'Test(EndgameLifecycleCardsComeFromTheResolver\|StatusOverridesReachBothTheCardAndTheEnvelope\|RecoveryNextStepComesFromTheResolver\|EndgameLifecycleEnvelopesMatchTheirCards\|SealFinalReview\|Recover\|Status\|StatusUX\|ProofCmd)' -count=1` | ok |
| `go test ./cmd -run 'Test(MigratedLifecycleSurfacesAgree\|MigratedLifecycleSurfacesArePlatformCorrect\|VisualOutputNeverLeaksRawWrapperCommands\|NextUpIsTheOnlyNextUpFunnel\|HumanFacingOutputGoesThroughWriteVisualOutput)' -count=1` | ok |
| `go test ./cmd -run 'Test(SessionLifecycle\|EndgameLifecycle\|MigratedLifecycleSurfaces\|NoCardOffersTheSameCommandTwice\|StatusOverridesReachBothTheCardAndTheEnvelope\|RecoveryNextStepComesFromTheResolver)' -count=1` | ok |
| `go test ./cmd -run 'Test(SessionFlow\|UpdateCmd\|SealFinalReview\|Recover\|Status\|StatusUX\|ProofCmd\|CanonicalAliasDelegates\|EveryDeciderAgreesOnTheNextCommand)' -count=1` | ok |
| `go build ./cmd/aether && go vet ./cmd` | clean |
| `gofmt -l cmd/` | no output |
| `go test ./cmd -count=1` (full suite, run twice) | 1 pre-existing, unrelated, documented flake both times (`TestBuildDispatchStartsHeartbeatMonitor`); everything else passes |
| `go build ./...` | clean |

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- **197-07** inherits all thirteen migrated renderings and can generalise `TestMigratedLifecycleSurfacesAgree` further if it needs to (the map already carries the pattern for adding a fourteenth). It can mark NEXT-02 complete once its own work lands — this plan and 197-04 together satisfy all eleven of NEXT-02's named commands, but the shared-ID gate correctly holds the requirement Pending until 197-07 (the last plan declaring it) also finishes.
- **No private next-step decider survives anywhere in the runtime.** `workflowSuggestionsForState`, `nextCommandFromState`, `nextCommandForHookState`, `nextUpSuggestionsForState`, `closeoutNextCommand` (197-02/197-04) and `recoverNextStep` (this plan) are all adapters over `resolveNextAction` now.
- **`computeNextAction` (context.go) remains** — it is a separate, out-of-scope feature (the context-capsule injected into worker prompts, not a closing card) and was never one of the eleven NEXT-02 commands; left untouched per the plan's own "do not change what any command DOES" boundary.

## Self-Check: PASSED

**Files claimed as created — all present on disk:**

| File | Present |
|---|---|
| `cmd/lifecycle_card_session_test.go` | yes |
| `cmd/lifecycle_card_endgame_test.go` | yes |
| `.planning/phases/197-one-answer-to-what-next/197-06-SUMMARY.md` | yes |

**Commits claimed — all present in `git log`:** `5b80623a`, `a92f3c4e`, `39c5717b`, `52ed0d1e`, `7e7a516f`, plus this summary commit.

**Nothing changed outside the declared surface plus documented deviations.** `git diff --stat ec4de395..HEAD -- cmd/` lists exactly the 16 source/test files named above (14 modified, 2 created); `.planning/STATE.md` and `.planning/ROADMAP.md` do not appear in it.

**Temporary demonstrations reverted:** the RED test run against `next_action.go`'s pre-fix state used `git stash`, popped immediately after capturing the failure (`git stash pop`, confirmed clean). The Task 3 revert demonstration on `renderPauseVisual` was undone by hand and confirmed via `git status --short cmd/codex_visuals.go` showing no diff before the file's edits were re-verified against the committed version.

---
*Phase: 197-one-answer-to-what-next*
*Completed: 2026-08-29*
