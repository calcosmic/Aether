---
phase: 208-never-a-dead-end
plan: 15
subsystem: colonize-territory-lifecycle
tags: [journey-rehearsal, live-walk, colonize, refusal-lifecycle, self-recovery]

# Dependency graph
requires:
  - phase: 208-never-a-dead-end (plan 14)
    provides: the saved-map revision fix (bindTransactionalTerritoryPublication),
      proved locally and committed at f1ffa454, which this plan's one
      authorised walk was spent to measure live
provides:
  - "208-JOURNEY-RUN.md's fourth dated section: live confirmation that the
    survey step's own on-disk fact check now passes for the first time ever
    in a real walk, plus three further genuinely new steps reached (discuss,
    specification, plan-first) before the walk stopped on an ordinary owner
    decision (choosing a planning preset) the harness's one-shot-prompt
    design cannot answer -- a new stopping point, honestly recorded, not a
    refusal and not fixed-and-retried per D-05."
affects: [209, 210]

# Actuals (#2632)
actuals:
  tokens: 4845
  tasks: 2
  commits: 1

tech-stack:
  added: []
  patterns:
    - "A harness step's own on-disk fact check only prints its two compared
      values on the failing branch -- a passing run genuinely cannot quote
      them after the fact once the temporary project is cleaned up. Recorded
      honestly rather than reconstructed with a plausible-looking guess
      (CLAUDE.md's Definition of Done: a false certificate is worse than a
      missing figure)."

key-files:
  modified:
    - .planning/phases/208-never-a-dead-end/208-JOURNEY-RUN.md

key-decisions:
  - "Spent the one walk D-05 authorised, in the order it required: preconditions
    checked and recorded first (claude CLI signed in, both 208-14 fix commits
    ancestors of HEAD, working tree clean except the pre-existing unrelated
    .aether/CONTEXT.md edit), then AETHER_JOURNEY_TRIALS=1 make eval-gate-journey
    run once, in the background, polled to completion."
  - "The walk cleared the survey step for the first time ever in a live run,
    then reached discuss and specification (also both new ground), then
    stopped at plan-first on a genuine owner decision (choosing a planning
    preset) -- not a refusal. Per D-05, this is reported honestly as a new
    stopping point rather than fixed-and-retried without asking the owner
    again."
  - "The survey step's own two compared revision values cannot be quoted in
    this record: the harness's own code only prints them on the failing
    branch, no git command anywhere in the retained transcript surfaced
    either value, and the temporary practice-project directory was already
    removed by Go's t.TempDir() cleanup before this record could be written.
    What is certain from the code path itself (a passing subtest requires
    equality) is that the two values matched; the literal strings are a
    genuine, honestly-reported gap in this run's own evidence, not a
    guessed or carried-over number."
  - "No second walk, no three-trial release-gate run, and no fix-and-retry
    of the new stopping point were attempted -- exactly one measurement,
    exactly as D-02/D-04/D-05 authorise."

requirements-completed: [UED-10]

coverage:
  - id: D1
    description: "Exactly one real walk was driven, after 208-14's fix was committed and verified as an ancestor of the run's HEAD, measured from the run's own log, transcript, and process timestamps -- nothing projected or carried over from the three earlier runs."
    requirement: "UED-10"
    verification:
      - kind: other
        ref: "AETHER_JOURNEY_TRIALS=1 make eval-gate-journey, started 2026-09-24T08:53:08Z, polled to completion at 2026-09-24T08:57:38Z, against commit 52dd3a636cdd66efe287af85ca6b2f4e73c28241 (208-14's three fix commits confirmed ancestors via git merge-base --is-ancestor)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The survey step's outcome is recorded as pass or fail; this run passed for the first time ever in a live walk, though the harness's own code does not print the two compared literal values on the passing branch and this run honestly records that gap rather than inventing a value."
    requirement: "UED-10"
    verification:
      - kind: other
        ref: "Real transcript b31c24c9-580f-4e9a-8493-a206dfe75071.jsonl: aether host colonize returned existing_survey:true,force_resurvey:true with no refusal text; four surveyor subagents completed; colonize-finalize returned ok:true; go test -v showed '--- PASS: TestJourney/survey (136.84s)'"
        status: pass
    human_judgment: true
    rationale: "The literal two compared revision strings are not recoverable from this run's own surviving evidence (harness prints them only on failure; the temp practice project was already cleaned up); a human should read the honest limitation recorded in 208-JOURNEY-RUN.md rather than have it silently auto-passed."
  - id: D3
    description: "How many refusals printed and how many next commands ran is recorded as a number read from the harness's own report, including which of the two possible reasons applies for a zero."
    requirement: "UED-10"
    verification:
      - kind: other
        ref: "go test -v output: 'trial 0: 0 printed refusal(s) found, 0 next command(s) run.' Cross-checked against the real transcript: no refusal text appears anywhere; the search ran for survey/discuss/specification (their fact checks passed) and found nothing to run, while plan-first's own fact check failed before its search was reached."
        status: pass
    human_judgment: false
  - id: D4
    description: "Every further step the walk reached is recorded with its own outcome; the walk stopped at a new step (plan-first) and this is reported in plain words with nothing further spent."
    requirement: "UED-10"
    verification:
      - kind: other
        ref: "go test -v: '--- PASS: TestJourney/start (11.16s)', '--- PASS: TestJourney/survey (136.84s)', discuss (23.30s) and specification (34.17s) both PASS, '--- FAIL: TestJourney/plan-first (24.82s)' with detail 'no plan artifact at .../phase-plan.json'"
        status: pass
    human_judgment: false
  - id: D5
    description: "The single-trial run is reported as a measurement, never a passed gate, quoting the gate's own refusal verbatim; no second walk or three-trial release-gate run was started."
    requirement: "UED-10"
    verification:
      - kind: other
        ref: "go test -v: 'journeyGateVerdict: refused -- journey report carries 1 trial(s), want at least 3 -- a reduced trial count can never satisfy the release gate.' git status --short and git diff --stat against cmd/journey*.go, scripts/build-messy-practice-project.sh, cmd/refusal_register.go confirmed empty during and after the run."
        status: pass
    human_judgment: false
  - id: D6
    description: "208-JOURNEY-RUN.md carries a fourth dated section for this run, with all three existing sections left untouched, word for word."
    requirement: "UED-10"
    verification:
      - kind: other
        ref: "git diff --stat .planning/phases/208-never-a-dead-end/208-JOURNEY-RUN.md: 248 insertions(+), 0 deletions -- pure append confirmed by grep for removed lines returning nothing"
        status: pass
    human_judgment: false

duration: 65min
completed: 2026-09-24
status: complete
---

# Phase 208 Plan 15: The Fourth Walk Clears Survey, Stops Three Steps Later on an Owner Decision Summary

**The one authorised walk cleared the out-of-date-map dead end for the first time ever (survey passed live), reached discuss and specification cleanly, then stopped at plan-first on an ordinary "choose a planning preset" question the automated harness has no way to answer — a genuinely new, non-refusal stopping point, recorded honestly per D-05 with nothing fixed-and-retried.**

## Performance

- **Duration:** 65 min
- **Started:** 2026-09-24T08:47:00Z (approx., session start)
- **Completed:** 2026-09-24T09:52:00Z (approx.)
- **Tasks:** 2 planned, both completed
- **Files modified:** 1 (`.planning/phases/208-never-a-dead-end/208-JOURNEY-RUN.md`, append-only)

## Accomplishments

- **Preconditions verified before spending anything.** `claude --version` returned `2.1.281 (Claude Code)` (a signed-in CLI). `git status --short` showed only the pre-existing, unrelated `.aether/CONTEXT.md` edit. All three of 208-14's fix commits (`03b08d06`, `5104df68`, `f1ffa454`) were confirmed ancestors of HEAD (`52dd3a636cdd66efe287af85ca6b2f4e73c28241`) via `git merge-base --is-ancestor`. The journey-tagged test binary was confirmed to compile cleanly under its build tag.
- **The one authorised walk was driven once, in the background, and polled to completion.** `AETHER_JOURNEY_TRIALS=1 make eval-gate-journey`, started `2026-09-24T08:53:08Z`, finished `2026-09-24T08:57:38Z` (270s / 4m30s total). The gate's own coverage check confirmed a complete, non-truncated run: `eval-gate-journey: discovered=17 executed=17`.
- **The survey step passed its own on-disk fact check for the first time ever in a live walk of this rehearsal.** The real transcript (session `b31c24c9-580f-4e9a-8493-a206dfe75071`) shows `aether host colonize` returning `existing_survey:true, force_resurvey:true` directly, with no refusal text printed at all — 208-11's self-recovery working exactly as designed, same as the third run. Four real surveyor subagents (`Plot-62`, `Scope-52`, `Survey-10`, `Survey-25`) dispatched and completed, and `aether colonize-finalize` returned `ok:true` with an honest closeout ("Territory surveyed: 7 documents"). `go test -v` recorded `--- PASS: TestJourney/survey (136.84s)` — 208-14's fix, proved locally on 2026-09-24, is now proved live.
- **Discuss and specification also passed — new ground never reached by any earlier walk.** The assistant correctly read the already-settled specification, correctly reported no material questions outstanding at discuss, and correctly reported the specification as already approved and matching disk at the specification step, without prompting or changing anything.
- **The walk stopped at plan-first (step 5 of 14) — three steps further than every earlier run, and for a genuinely different kind of reason.** The assistant reached the "choose a planning preset" boundary, rendered all four options with a stated pick ("Fast"), and asked the owner to reply with one word. The journey harness sends exactly one prompt per step with no follow-up turn, so nobody was there to answer, and the session ended there. The step's own on-disk fact check then failed: no `phase-plan.json` artifact existed, because planning never started. This is not a refusal (no `Disposition: stop`, no `NextCommand`, not in `cmd/refusal_register.go`) — it is an ordinary, by-design owner decision that `.aether/commands/plan.yaml` itself states must never be silently chosen. Per D-05, this new stopping point is reported honestly; nothing was fixed or retried.
- **Refusals printed and next-commands run: 0 and 0, for a genuinely mixed reason this time.** For survey, discuss, and specification (all three passed their own fact checks), `journeyRunPrintedNextCommands`'s search code ran for real and found nothing to run, because nothing was ever printed. For plan-first, the on-disk fact check failed before that step's own search was reached. Either way, no refusal was ever printed anywhere in this transcript, so the phase's still-open clause — proving the rehearsal runs a printed refusal's own next command for real, against a genuine printed refusal — remains unproven by this run.
- **The survey step's own two compared revision values could not be quoted in this record, and that gap is itself recorded honestly rather than papered over.** `journeyAssertStepFact`'s own code only prints the saved map's recorded revision and the practice project's real HEAD on the *failing* branch; a passing subtest logs neither. No `git rev-parse HEAD` or `git log` command appears anywhere in the retained transcript, and the temporary practice-project directory — the one place both values still existed — was already removed by Go's own `t.TempDir()` cleanup before this record could be written. What is certain from the code path itself (`if snap.SourceRevision != head { fail(...) }`) is that the two values were equal at the moment the check ran; the literal strings are not recoverable from any surviving evidence of this run.
- **The below-minimum trial count was correctly refused, exactly as required.** `journeyGateVerdict` refused the report verbatim: *"journey report carries 1 trial(s), want at least 3 -- a reduced trial count can never satisfy the release gate."* This is the expected, correct outcome of an intentionally single-trial measurement, not a failure of this plan.
- **The rehearsal record now carries a fourth dated section, and the three earlier sections are untouched.** `git diff --stat` on `208-JOURNEY-RUN.md` shows 248 insertions, 0 deletions — a pure append, confirmed by checking that no existing line was removed or altered.

## Task Commits

1. **Task 1: Drive the one authorised walk and measure what it actually did** — no commit (this task only observed and measured; no files were changed).
2. **Task 2: Write this run into the record, honestly, whatever it showed** — `1eba6c78` (docs)

## Files Created/Modified

- `.planning/phases/208-never-a-dead-end/208-JOURNEY-RUN.md` — appended a fourth dated section recording this run's full evidence (run identity, command/timing, caps, the one trial, why it stopped, the four settled questions, what remains unproven, combined totals); the three earlier sections are byte-unchanged.

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

None — plan executed exactly as written. The walk's real outcome (clearing survey, reaching three new steps, then stopping on a genuine owner decision rather than a refusal) is exactly the kind of honestly-recorded result the plan's own action text anticipated ("If the walk stops anywhere ... stop there. Capture the stopping evidence verbatim, report it to the owner in plain words, and spend nothing further").

## Issues Encountered

None beyond the observability gap in the survey step's own passing-branch logging, documented above and in `208-JOURNEY-RUN.md`'s fourth section — a genuine limitation of the harness's own code, not an issue this plan's own scope covers fixing (Task 2 explicitly defers any code or defect-register change to the next plan).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

The one walk D-05 authorised has been spent, measured, and honestly recorded. The saved-map fix (208-14) is now confirmed working live, not just locally: the survey step passed its own on-disk fact check for the first time ever in a real walk. The rehearsal has never completed end to end, and now stops at a new point — plan-first, an ordinary owner decision, not a refusal — three steps further than ever before. The phase's still-open clause (the rehearsal running a printed refusal's own next command for real, against a genuine printed refusal, outside a synthetic fixture) remains unproven, because no refusal printed anywhere in this run.

Per this plan's own instruction, the defect register (`.planning/WINDOWS.md`) and `.planning/ROADMAP.md` are deliberately untouched here — `git status --short` confirms only `208-JOURNEY-RUN.md` was modified by this plan (plus the pre-existing, unrelated `.aether/CONTEXT.md` edit from another session). That closing decision — whether to spend a further walk, and how to record the new stopping point on the defect register per D-06 — belongs to the plan that closes this phase.

## Self-Check: PASSED

- FOUND: .planning/phases/208-never-a-dead-end/208-JOURNEY-RUN.md (fourth section present, 248 lines added)
- FOUND commit: 1eba6c78
- CONFIRMED: `go test ./cmd -run 'TestJourneyGateVerdictRefusals|TestJourneyIncompleteTrialIsNeverPassed|TestJourneyGateRefusesAPartialRun|TestEvalGateVocabularyMatchesTheManifest' -count=1 -timeout 8m` → PASS
- CONFIRMED: `git status --short` shows only the pre-existing `.aether/CONTEXT.md` edit; no `cmd/` or `scripts/` file modified by this plan
- CONFIRMED: `git diff --stat .planning/phases/208-never-a-dead-end/208-JOURNEY-RUN.md` shows 248 insertions(+), 0 deletions(-) since before this plan started

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-24*
