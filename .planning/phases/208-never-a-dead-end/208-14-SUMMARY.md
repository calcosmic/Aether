---
phase: 208-never-a-dead-end
plan: 14
subsystem: colonize-territory-lifecycle
tags: [go, colonize, territory-snapshot, refusal-lifecycle, mutation-testing]

# Dependency graph
requires:
  - phase: 208-never-a-dead-end (plans 01-13)
    provides: the self-recovery mechanism (attemptRefusalSelfRecovery,
      cmd/refusal_self_recovery.go, plan 11), its now-failable guards (plan
      13), and 208-JOURNEY-RUN.md's "Third Real Run" -- the live evidence
      that the survey-and-finalize sequence can complete successfully while
      still leaving the saved map's own recorded revision stale (WINDOWS.md
      row 53's fourth proximate cause)
provides:
  - "bindTransactionalTerritoryPublication (cmd/codex_colonize.go): the one
    shared builder for the saved map's publication binding (transaction id,
    baseline digest, the lane-selecting PublicationMode field, the candidate
    survey directory, and each dispatch's rewritten output paths/brief).
    territoryPlanPreflight (cmd/codex_workflow_cmds.go) and colonize's own
    forced resurvey (runCodexColonizePlanOnly) both call it now instead of
    the plan front door holding the only copy."
  - "TestColonizeRefreshesTheSavedMapsRevision
    (cmd/colonize_snapshot_refresh_test.go): drives a real plan-only
    colonize manifest through a completion packet and the finalize step in
    a real temporary git repository, and asserts the saved map's own record
    file names the repository's current HEAD afterward."
  - "TestSavedMapPublicationHasOneBuilder
    (cmd/colonize_snapshot_refresh_test.go): an AST structural guard that
    fails by name if a second, independent call site anywhere in the module
    sets codexColonizeManifest.PublicationMode outside
    bindTransactionalTerritoryPublication."
  - "A relaxed transactional-finalize tolerance: a forced resurvey with a
    lighter-than-full surveyor team no longer hard-fails; the same
    runtime-synthesis fallback the legacy finalize lane already gave every
    survey now applies on the transactional lane too."
affects: [209, 210]

# Actuals (#2632)
actuals:
  tokens: 6052
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "A publication-binding block (transaction id, baseline digest, a
      lane-selecting field, a candidate directory, rewritten per-dispatch
      output paths/brief) is built in exactly one shared function, called
      from every path that can trigger the same class of refresh, rather
      than being copied per caller -- the drift this repository has
      repeatedly shipped (CLAUDE.md's 'Definition of Done' epigraph list)."
    - "An internal all-or-nothing validation gate is relaxed to match an
      existing, already-safe tolerant fallback elsewhere in the same
      pipeline, with the relaxation observably recorded (logActivity)
      rather than silently accepted -- carrying forward, not undermining,
      the phase's 'warns and carries on' standard."

key-files:
  created:
    - cmd/colonize_snapshot_refresh_test.go
  modified:
    - cmd/codex_colonize.go
    - cmd/codex_colonize_finalize.go
    - cmd/codex_workflow_cmds.go
    - cmd/testdata/refusals/untyped-floor.json

key-decisions:
  - "The cause is the lane-selecting field, confirmed from code and from
    208-JOURNEY-RUN.md's own third-run transcript evidence (no
    territory_snapshot_id in that run's closeout screen, matching a
    finalize that took the non-publishing lane). Fixed by extracting
    bindTransactionalTerritoryPublication and calling it from colonize's
    own forced-resurvey path, not by adding a second writer of the saved
    map or by touching publishTerritorySnapshot itself."
  - "The shared builder is called from runCodexColonizePlanOnly only when
    existingSurvey && opts.ForceResurvey -- a forced resurvey of a project
    that already has a saved map, exactly the situation with a prior
    snapshot that can go stale. A first-ever survey (no existing survey)
    has no prior snapshot to keep in sync and is left on its existing,
    unaffected legacy path; this keeps the fix's blast radius to the
    confirmed cause rather than converting every colonize call into the
    stricter transactional lane and rewriting the many legacy-lane tests
    that assume it (see 'Tolerant-path question' below)."
  - "runTransactionalColonizeFinalize's hard 'accepted N of M required
    worker-authored survey artifacts' error was relaxed to a non-fatal
    logActivity note. See the dedicated section below for the full
    justification, required verbatim by the orchestrator's review request."
  - "cmd/testdata/refusals/untyped-floor.json's recorded count was lowered
    386 -> 385, per TestUntypedRefusalFloorOnlyShrinks's own instruction,
    because the relaxation above removed one untyped fmt.Errorf return from
    the ten declared lifecycle files. See the dedicated section below."

patterns-established:
  - "Reproduce first, in a test that drives the runtime's own real path (a
    built manifest through a completion packet through the finalize
    command), never a direct call to the innermost function under
    suspicion -- confirmed here by two independent verbatim-message
    mutation proofs (see 'Mutation Proof' below)."

requirements-completed: [UED-10]

coverage:
  - id: D1
    description: "The cause of the saved map's stale revision after a successful survey is established from code plus the third walk's own recorded transcript evidence, and named exactly: colonize's own plan-only manifest never carried the PublicationMode/TransactionID/BaselineDigest/CandidateSurveyDir binding territoryPlanPreflight already gave its own manifests, so colonize-finalize took the legacy lane, which writes the seven survey documents but never calls publishTerritorySnapshot -- the sole writer of territory-snapshot.json."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/colonize_snapshot_refresh_test.go#TestColonizeRefreshesTheSavedMapsRevision"
        status: pass
      - kind: other
        ref: "Mutation proof: reverting the fix in a disposable worktree reproduces the exact live symptom (both compared revision values named)."
        status: pass
    human_judgment: false
  - id: D2
    description: "A survey run through the same path the practice project uses (plan-only manifest -> completion packet -> finalize) now leaves the saved map recording the project's current revision, and the chain the rehearsal depends on holds end to end: the program's own freshness reader classifies the result Fresh, and a second survey pass over the same project does not fall into the sibling colonize-finalize-existing-survey-found refusal."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/colonize_snapshot_refresh_test.go#TestColonizeRefreshesTheSavedMapsRevision"
        status: pass
    human_judgment: false
  - id: D3
    description: "Whatever arranges for the saved map to be rewritten is built in exactly one place -- bindTransactionalTerritoryPublication -- and a second, independent call site setting the lane-selecting field is caught by name."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/colonize_snapshot_refresh_test.go#TestSavedMapPublicationHasOneBuilder"
        status: pass
      - kind: other
        ref: "Mutation proof: planting a second PublicationMode-setting function in cmd/codex_workflow_cmds.go inside a disposable worktree makes the guard fail naming that file; removing it makes the guard pass again."
        status: pass
    human_judgment: false
  - id: D4
    description: "The fix does not turn a path that succeeds today into a new dead end: a forced resurvey with a lighter-than-full surveyor team still succeeds (the transactional lane's fallback now matches the legacy lane's), and every existing colonize-finalize/territory-lifecycle test named in the plan's verify command still passes."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "go test ./cmd -run 'TestColonizeFinalizeRecordsExternalSurveyors|TestColonizeFinalizeAllowsFirstTimeWorkerWrittenSurveyArtifacts|TestColonizeFinalizeRejectsStaleManifest|TestColonizeFinalizeRejectsOutOfScopeSurveyClaim|TestColonizeFinalizeRejectsSymlinkSurveyClaim|TestColonizeFinalizeRejectsWorkspaceDriftAfterManifest|TestColonizeFinalizeRecoversAMissingTimestamp|TestColonizeFinalizeRefusalNamesTheWayPast|TestTerritoryLifecycleFreshPassThrough|TestTerritoryLifecycleAutomaticRefresh|TestTerritoryLifecyclePlanRequiresSnapshot' -count=1"
        status: pass
    human_judgment: false
  - id: D5
    description: "The full cmd suite ran to completion with discovered==executed, and every failure is either already on this project's recorded known-red list (WINDOWS.md row 56, or the separately recorded 2026-09-14 baseline) or fixed here -- none is new and outstanding."
    requirement: "UED-10"
    verification:
      - kind: other
        ref: "go test ./cmd -count=1 -timeout 90m, tee'd to /tmp/aether-208-14-fullsuite.log: FULL-SUITE FAIL discovered=6029 executed=6029 lanes=59; 30 distinct failing test names, each appearing twice (a first pass plus the suite's own automatic re-run of failures), all 30 matched verbatim against WINDOWS.md row 56's 14 already-tracked + 15 newly-catalogued names plus the separately recorded TestResolveTestCommand_GoProject baseline -- zero unmatched."
        status: pass
    human_judgment: false
---

# Phase 208 Plan 14: The Saved Map Now Refreshes on Colonize's Own Forced Resurvey Summary

**Extracted `bindTransactionalTerritoryPublication` as the one shared builder for the saved map's publication binding, wired colonize's own forced-resurvey path to call it (closing the gap that only the plan front door ever kept `territory-snapshot.json` in sync), relaxed the transactional finalize lane's all-worker-authored requirement to match the legacy lane's own tolerant fallback, and proved both fixes by mutation in a disposable worktree — the full `cmd` suite then ran clean against the project's own known-red list.**

## Performance

- **Duration:** ~95 min (dominated by one ~31-minute unscoped full-suite background run)
- **Started:** 2026-09-24T07:45:00Z (approx., session start)
- **Completed:** 2026-09-24T08:55:00Z (approx.)
- **Tasks:** 3 planned, all completed
- **Files modified:** 4 (1 created, 3 modified) plus 1 testdata file

## Accomplishments

- **The fourth proximate cause is diagnosed from code and from the third walk's own transcript evidence, not a guess.** `territoryPlanPreflight` (`cmd/codex_workflow_cmds.go`) was the *only* place in the program that set `codexColonizeManifest.PublicationMode = territoryPublicationTransactional` — confirmed with `grep -rn "PublicationMode\s*=" cmd/*.go` returning exactly one non-test hit before this plan. `runCodexColonizePlanOnly` (`cmd/codex_colonize.go`), the function `aether colonize --plan-only` calls (what the rehearsal's survey step reaches through `aether host colonize`), never set it. `runCodexColonizeFinalize`'s branch `if manifest.PublicationMode == territoryPublicationTransactional { return runTransactionalColonizeFinalize(...) }` (`cmd/codex_colonize_finalize.go:365`) is the one line that decides it — a manifest without the field takes the legacy lane, which calls `writeSurveyArtifacts` (writes the seven survey markdown files) but never calls `publishTerritorySnapshot`, the sole writer of `territory-snapshot.json` (its own doc comment: "the sole live-write boundary for a complete territory refresh").
- **The journey record's alternative trace targets were examined and ruled out.** `currentTerritoryRevision`/`canonicalTerritoryRoot` (`cmd/survey_staleness.go`) are called identically by both lanes and by the journey's own check — no divergent root resolution. `classifySurveyFreshness` is read-only and never repairs anything (its own doc comment). `journeySeedStaleSurvey` (`cmd/journey_seed.go`) writes a genuinely valid, self-consistent snapshot pinned to the seeding-time revision — the trap is real, not a fixture defect. The third walk's own closeout screen ("Territory surveyed: 7 documents", no snapshot identifier reported) is consistent with the legacy lane's result map, which never sets `territory_snapshot_id` at all (only the transactional lane's result map does) — this is the same evidence 208-JOURNEY-RUN.md's own "Not proven by this run" section left unresolved, and it now resolves in favor of the lane-selecting-field hypothesis.
- **Plain English:** Aether's own "survey the code" command has two separate paths to do the same job — one used automatically before planning, one used when a person or script runs `colonize` directly. Only the automatic one was ever taught to update Aether's own saved map afterward. Running colonize directly could report "survey complete" while quietly leaving the saved map's own record of *which version of your code it surveyed* untouched — so the next automatic check could still call it out of date immediately afterward, even though the survey had just genuinely finished.
- **Reproduced locally before touching runtime code (Task 1).** `TestColonizeRefreshesTheSavedMapsRevision` builds a real temporary git repository, seeds a valid saved map pinned to that commit, makes three further commits, then drives the exact rehearsal sequence (plan-only manifest → completion packet → `colonize-finalize`) under `AETHER_UNATTENDED=1` (the same condition the third live walk ran under). It failed against the pre-fix runtime with:
  ```
  saved map's recorded revision is bfb8c139c374d6405e54921162c059a77c221707, want current HEAD 377c16b695b63cf2305167a4266547ec0f44613e -- a survey that finished and saved successfully should leave the saved map recording the project's real, current state
  ```
- **Fixed by sharing one builder, not adding a second writer (Task 2).** `bindTransactionalTerritoryPublication` (`cmd/codex_colonize.go`) now holds the transaction-id/baseline-digest/`PublicationMode`/candidate-dir/dispatch-rewrite block that used to live only inside `territoryPlanPreflight`; both callers use it. `runCodexColonizePlanOnly` calls it exactly when `existingSurvey && opts.ForceResurvey` — a forced resurvey of a project that already has a saved map, the situation with a prior snapshot that can go stale. `TestSavedMapPublicationHasOneBuilder` structurally guards that no third call site can set `PublicationMode` directly.
- **The tolerant-path question was checked, not assumed, and answered by relaxing an internal gate rather than by accepting a regression.** See the dedicated section below.

## The tolerant-path question, answered (per the orchestrator's explicit request)

**What changed and why.** `runTransactionalColonizeFinalize`'s original code refused outright — `return nil, fmt.Errorf("transactional territory refresh accepted %d of %d required worker-authored survey artifacts", ...)` — unless *every one* of the seven required survey documents came from a genuine worker claim. The legacy (non-transactional) finalize lane has never required this: `writeSurveyArtifacts` already tolerantly falls back to runtime-synthesized content (rendered from real project facts) for any document no worker claimed, and this fallback is exercised today by `TestColonizeFinalizeAllowsFirstTimeWorkerWrittenSurveyArtifacts` (only 1 of 7 documents worker-authored, the rest synthesized, and finalize still succeeds).

Because this plan's fix now routes colonize's own forced-resurvey path through the *same* transactional lane the plan front door uses, and because CLAUDE.md's own record states "light colonize depth sends two surveyors, not four" (a caste roster genuinely lighter than the full seven-document set), an unmodified strict check would have turned an `aether colonize --force-resurvey` on a light-depth colony — which succeeds today via fallback synthesis — into a hard failure it does not have today. That is precisely the shape of regression this phase exists to remove, and precisely what the plan's own Task 2 action text told me to check before committing to the "share one builder" shape.

**This is not the same thing the plan's prohibition forbids.** The prohibition is about *journey checks, on-disk fact checks, refusal rows, and floors* — the checks that prove the rehearsal actually completed a step, and the registered refusal contract an owner reads on screen. `git diff --stat` confirms none of `cmd/journey*.go`, `scripts/build-messy-practice-project.sh`, or `cmd/refusal_register.go`'s existing rows changed. The check I relaxed is an internal, undocumented, untested (no test anywhere asserted its rejection message before this plan — confirmed by `grep -rn "transactional territory refresh accepted\|preservedWorkerArtifacts" cmd/*_test.go` returning nothing) all-or-nothing gate inside one finalize branch, not a registered refusal an owner ever sees named, and not a journey or on-disk fact check.

**What an owner sees now that they did not before.** Before this plan, running `aether colonize --force-resurvey` on a project surveyed with a light (partial) surveyor team, when that resurvey happened to also need the saved-map-publishing lane, would have failed outright with an internal Go error string — a hard stop with no named way past it, on a survey that would have succeeded seconds earlier under the legacy lane. After this plan, that same resurvey succeeds, exactly as it would have under the legacy lane, and Aether's own activity log now records a plain, honest note when this happens: `"transactional territory refresh published %d of %d required survey documents from worker claims; the rest were synthesized from workspace facts"` (`logActivity`, `cmd/codex_colonize_finalize.go`) — visible to `aether history`/the activity log, not silently swallowed.

**What safety is still enforced on that path after the relaxation.** `validateTerritoryPublicationMarkdown` (called from `publishTerritorySnapshot` for every artifact, unconditionally) still rejects a document whose content, after stripping headings, is empty or one of a fixed set of placeholder strings (`"placeholder"`, `"todo"`, `"tbd"`, `"coming soon"`, `"not available"`, `"synthetic"`) — the exact same content-quality gate that already protects the legacy lane's own fallback-synthesized documents. `validateTransactionalSurveyOutputs` (unchanged) still requires every dispatch's declared output to exist on disk as a real, non-symlink regular file before publication. `validateExternalSurveySpawnEvidence` (unchanged) still requires real, terminal spawn-tree evidence for every dispatch. Only the *count* of worker-versus-synthesized documents stopped being an all-or-nothing gate; the content and provenance gates that actually protect the immutable snapshot from garbage or forged evidence are untouched.

## The untyped-refusal floor, confirmed re-measured not guessed (per the orchestrator's explicit request)

`TestUntypedRefusalFloorOnlyShrinks` (`cmd/refusal_enumerate_test.go`) is a shrink-only ratchet: it calls `enumerateRefusalSites`/`untypedRefusalSites` to *count, from the live source tree, right now*, how many error returns in the ten declared lifecycle files (`cmd/refusal_register.go`'s `refusalDeclaredFiles`, which includes `codex_colonize_finalize.go`) are not yet a typed `refuse(...)` call. It compares that live count against the number checked into `cmd/testdata/refusals/untyped-floor.json` and fails if the recorded floor is *higher* than the live count — i.e., it fails if the floor claims more untyped sites exist than genuinely do, which is exactly what happened here: removing the `fmt.Errorf` return above (an untyped error return, by the enumerator's own definition) dropped the live count from 386 to 385, and the checked-in floor still said 386.

I ran the test before touching the JSON (it failed, naming the real count: *"recorded untyped-floor.json count (386) is inflated above the real untyped count (385) — lower it to 385"*), then edited `untyped-floor.json` to `385` — the exact number the test's own live enumeration reported, not a value I guessed or rounded — and re-ran it to confirm it passes (`go test ./cmd -run TestUntypedRefusalFloorOnlyShrinks -count=1 -v`: PASS). This is the ratchet shrinking (an allowed, in fact required, direction per the ratchet's own doc comment: "This number may only go down... it must never be raised to make a failing check pass"), driven by the enumerator's own live count, not typed by hand.

## Task Commits

1. **Task 1: Establish the actual cause, and reproduce it here with a failing test** - `03b08d06` (test)
2. **Task 2: Fix the cause, without turning a working path into a new dead end** - `5104df68` (fix)
3. **Task 2 (deviation, discovered during Task 3's verification pass): lower the untyped-refusal floor** - `f1ffa454` (fix)

_Task 3 itself changed no files — it ran checks and recorded what they said._

## Files Created/Modified

- `cmd/colonize_snapshot_refresh_test.go` — `TestColonizeRefreshesTheSavedMapsRevision` (the reproduction and end-to-end proof) and `TestSavedMapPublicationHasOneBuilder` (the structural one-builder guard)
- `cmd/codex_colonize.go` — `bindTransactionalTerritoryPublication` (the shared builder), and `runCodexColonizePlanOnly` now calling it when `existingSurvey && opts.ForceResurvey`
- `cmd/codex_colonize_finalize.go` — `runTransactionalColonizeFinalize`'s hard all-worker-authored check relaxed to a non-fatal, logged note
- `cmd/codex_workflow_cmds.go` — `territoryPlanPreflight`'s inline binding block replaced with a call to the shared builder
- `cmd/testdata/refusals/untyped-floor.json` — floor lowered 386 → 385, re-measured from the enumerator's own live count

## Decisions Made

See `key-decisions` in frontmatter, plus the two dedicated sections above (tolerant-path relaxation and the untyped-floor ratchet).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Lowered the untyped-refusal floor after Task 2's own cleanup**
- **Found during:** Task 3 (running the plan's own targeted verify command)
- **Issue:** `TestUntypedRefusalFloorOnlyShrinks` failed: Task 2's relaxation of the transactional finalize's hard error to a `logActivity` call removed one untyped `fmt.Errorf` return from the ten declared lifecycle files, dropping the real count from 386 to 385, but the checked-in floor still said 386.
- **Fix:** Lowered `cmd/testdata/refusals/untyped-floor.json`'s `untyped` field to 385, with an added sentence in its own `reason` field documenting exactly which change caused the drop — the test's own re-measured count, not a guessed value.
- **Files modified:** `cmd/testdata/refusals/untyped-floor.json`
- **Verification:** `go test ./cmd -run TestUntypedRefusalFloorOnlyShrinks -count=1 -v` PASSes; re-ran the full targeted verify command afterward and confirmed only the two already-known-red `TestCodexNativeCancellation*` failures remained.
- **Committed in:** `f1ffa454`

---

**Total deviations:** 1 auto-fixed (1 Rule 1 bug, a ratchet the fix itself caused to shrink).
**Impact on plan:** Necessary consequence of Task 2's own legitimate relaxation; the ratchet moved in its only permitted direction (down), driven by the checker's own live re-measurement. No scope creep — no other file in this ratchet's declared scope was touched.

## Issues Encountered

None beyond the deviation above.

## Full Suite Verification (Task 3)

**Fast checks (all with explicit timeouts, none pushed to background):**

| Command | Result |
|---|---|
| `go build ./...` | Clean |
| `go vet ./cmd ./pkg/...` | Clean |
| `go test ./cmd -run 'Colonize\|Territory\|Refusal\|Refuse\|Unattended\|TestJourneyGateVerdict\|TestEvalGate\|TestLifecycleFlatMirrorsMatchCanonical\|TestPlanAndColonizeWrappersAreByteIdentical' -count=1 -timeout 20m` | 2 failures, both already-known-red (see below); ran twice (once mid-Task-2, once as Task 3's own official run) — 150.986s the second time |
| `go build ./...` again + `go vet` again (post untyped-floor fix) | Clean |
| Compile-only check of the journey-tagged test binary (`go test -tags=journey -run='^$' -c ...`) — no live run started | Clean |

**The two scoped-run failures, triaged by name against `.planning/WINDOWS.md`:**
- `TestCodexNativeCancellationRefusalReplay` and `TestCodexNativeCancellationRefusalEvidence` are both named verbatim in WINDOWS row 56's "15 are newly observed by this run and not yet catalogued anywhere" list (recorded 2026-09-23T09:36:26.742Z, phase 208, plan 208-08's own full-suite run at commit `aaab3dfa`, "Out of scope for 208-08... not bisected against which of 208-01..208-07 introduced them"). Both are in the Codex-native cancellation-replay family, work the owner parked separately (204.2–204.5, 2026-09-20) — consistent with, and independently confirmed against, the register rather than assumed from that context alone.

**Full unscoped suite (`go test ./cmd -count=1 -timeout 90m`), background-started, polled to completion, tee'd to `/tmp/aether-208-14-fullsuite.log`:**

- Headline: `FULL-SUITE FAIL discovered=6029 executed=6029 lanes=59` — complete, not truncated (the awk check from Task 3's own `<verify>` block confirms `discovered==executed`).
- Wall clock: started `2026-09-24T08:10:31Z`, finished `2026-09-24T08:41:41Z` — 31m10s.
- **30 distinct failing test names** (60 raw `--- FAIL` lines — every name appears exactly twice, once in the suite's first pass and once in its own automatic re-run of failures; confirmed by `sort | uniq -c`).
- **Every one of the 30 names matched, verbatim, against `.planning/WINDOWS.md` row 56's own two named lists** (14 "already-tracked" + 15 "newly observed... not yet catalogued") **plus the separately-recorded 2026-09-14 known-red baseline** (`TestResolveTestCommand_GoProject`, cited by row 56's own text): `TestBuildStartLegacyHelpersRetired200`, `TestCheapModelWorkerNeedsNoReasonOnTheCard`, `TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount`, `TestCodexNativeCancellationRefusalEvidence`, `TestCodexNativeCancellationRefusalReplay`, `TestCodexNativeEvidenceReceiptSchemaDispatch`, `TestCodexNativeEvidenceRejects`, `TestCodexNativeFourthReviewLegacyReplayInventory`, `TestCodexNativeGapRecovery`, `TestCodexNativePhaseEvidence`, `TestCodexNativeThirdReviewReplayInventory`, `TestCompletionPacketSchemaMatchesStructs`, `TestContinueCreditsTasksProvenInAnEarlierAttempt`, `TestCurrentVocabulary199`, `TestDefaultOracleInvokerAvoidsOpenCodeInsideOpenCodeAgent`, `TestFailedCheckSendsExactlyOneBuilderFixAttempt`, `TestFixAttemptIsCountedSeparately`, `TestFixAttemptNeverOverwritesTheFirstResult`, `TestGoldenBuildVisualOutput`, `TestGoldenContinueVisualOutput`, `TestGoSourceHintsMatchCobraContracts`, `TestHumanFacingOutputGoesThroughWriteVisualOutput`, `TestNoRegisteredSubcommandIsUnreferenced`, `TestNoSecondAutomaticFixAttempt`, `TestPartialRedispatchRecoveryNeverNamesAlreadyProvenWork`, `TestPhase199GateReceipt`, `TestPlanningAdversarial200`, `TestPlanningPublicPaths200`, `TestResolveTestCommand_GoProject`, `TestSeededBankIsReproducible`.
- **Zero new failures.** Computed by a strict set comparison (`comm -23` between the 30 observed names and the 30 named in the register) — the two sets are identical.

**Protected paths untouched, confirmed by diff:**
```
git diff --stat -- .aether/commands/ .claude/commands/ .opencode/commands/ cmd/journey*.go scripts/build-messy-practice-project.sh cmd/refusal_register.go
```
returns empty, both against the working tree and diffed from the commit immediately before this plan started.

**Link to the next plan's walk:** this plan's three commits (`03b08d06`, `5104df68`, `f1ffa454`) are on branch `oracle-reinstate`, all committed. The next plan's walk will be measured against `f1ffa454ac84e1a0472da83df49288fd08d1d7cd`. `scripts/build-messy-practice-project.sh` installs Aether from this working tree (unchanged by this plan), so these commits are what the next authorised walk will run against. The rehearsal's own journey-tagged test binary compiles cleanly under its build tag; no live walk was started, and none was authorised for this plan (D-05).

**Plain-English readiness statement:** the tree is ready for the one walk the owner authorised — the fix is proven locally, nothing else moved, and every remaining test failure was already known and recorded before this plan started.

## Mutation Proof (per CLAUDE.md's Definition of Done — a test must be able to fail)

All mutations were applied and reverted in disposable git worktrees created under the session scratchpad (named to contain "Aether" per this repo's `TestResolveAetherRoot_GitFallback` constraint), never in this checkout. `git worktree remove --force` ran after every proof; `git status --short` in the main checkout showed only this plan's own files changing throughout (plus the pre-existing, untouched `.aether/CONTEXT.md` edit from another session).

| Mutation | Test | Observed result (verbatim) |
|---|---|---|
| Reverted the fix (Task 2's three runtime source files back to their pre-fix committed state), kept the updated test | `TestColonizeRefreshesTheSavedMapsRevision` | FAILS: `saved map's recorded revision is bcb56668555d28fa62ef666119d3ab086afb986f, want current HEAD 93449b1fc91c6b568088caf70923515b99f4d0ee -- a survey that finished and saved successfully should leave the saved map recording the project's real, current state` |
| Fix applied (restored) | `TestColonizeRefreshesTheSavedMapsRevision` | PASSes |
| Planted a second call site (`secondPublicationModeSetter`) setting `PublicationMode` directly in `cmd/codex_workflow_cmds.go`, outside the shared builder | `TestSavedMapPublicationHasOneBuilder` | FAILS: `only cmd/codex_colonize.go may set PublicationMode -- the saved map's publication binding must be built in exactly one shared place; found a second builder in: [cmd/codex_workflow_cmds.go]` |
| Second call site removed (restored) | `TestSavedMapPublicationHasOneBuilder` | PASSes |

One incidental finding during the second mutation proof: the first attempt at `TestSavedMapPublicationHasOneBuilder` failed *before any mutation was planted*, because the walk reached into a stale, unrelated linked worktree checked out inside this repository (`.claude/worktrees/agent-af06650f4d83f8d05/`, left over from another agent's earlier run, pinned to an older pre-refactor commit). Fixed by adding `"worktrees"` to the walk's existing `SkipDir` list (matching the convention already used by `cmd/build_attempt_external_test.go`) — a one-line addition to the new test file itself, not a runtime file, and not a weakening of what the guard checks.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

The must-have truths this plan's own frontmatter named are met: the cause is established from evidence and named in code, a test reproduces the live symptom locally and turns red the moment the fix is reverted, nothing that succeeds today has been turned into an unnamed stop, and the tree is committed and fully checked. `.planning/WINDOWS.md` row 53 can now be updated by the phase's own closing plan (208-15) to reflect that the underlying cause is fixed and locally proven — this plan does not touch WINDOWS.md itself, leaving that update to the plan that spends the one authorised walk and can report its real, live outcome.

Ready for `208-15` — the one walk D-05 authorises, against commit `f1ffa454ac84e1a0472da83df49288fd08d1d7cd`.

## Self-Check: PASSED

- FOUND: cmd/colonize_snapshot_refresh_test.go
- FOUND: cmd/codex_colonize.go
- FOUND: cmd/codex_colonize_finalize.go
- FOUND: cmd/codex_workflow_cmds.go
- FOUND: cmd/testdata/refusals/untyped-floor.json
- FOUND commit: 03b08d06
- FOUND commit: 5104df68
- FOUND commit: f1ffa454

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-24*
