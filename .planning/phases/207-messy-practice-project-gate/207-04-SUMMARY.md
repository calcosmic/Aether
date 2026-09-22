---
phase: 207-messy-practice-project-gate
plan: 04
subsystem: testing
tags: [claude-code-headless, journey-harness, trial-classification, release-gate, go-test]

# Dependency graph
requires:
  - phase: 207-messy-practice-project-gate
    provides: "207-01 -- cmd/journey.go's report/verdict library and cmd/journey_live_test.go's tracer slice (single-step, real-chat pattern, --resume/transcript-reading mechanics); 207-02 -- the nine built traps cmd/testdata/journey/traps.json declares, journeySeedSupersededPlanCmd's specification-corrected-mid-planning fixture; 207-03 -- statusGuidanceCommandsWithoutMenuWrapper and the committed expected-red register the status step's own genuine result folds into"
provides:
  - "cmd/journey_live_test.go's TestJourney -- a table over all fourteen declared lifecycle steps, chained through one real claude -p session via --resume, each step carrying its own sized caps and its own on-disk fact check derived from the real runtime writers"
  - "an AETHER_JOURNEY_STEP single-step debugging mode (journeyFastForwardToStep) that fast-forwards the earlier lifecycle with direct aether commands rather than replaying the whole chat -- the mechanism Plan 05's revert proof will target one step at a time"
  - "cmd/journey.go's closed trial-outcome vocabulary (passed/flaky-failure/real-failure/incomplete) and journeyGateVerdict extended to seven ordered refusal rules (scope, mode, trial count, per-trial declared==executed, no real failure and not every trial may fail, every expected-red case still-red, no unregistered gap) -- still the one verdict function"
  - "journeyEvaluateExpectedRed -- folds the status step's live statusGuidanceCommandsWithoutMenuWrapper result into the report's expected_red section, distinguishing still-red / now-green / unregistered-gap"
  - "cmd/journey_test.go's journeyPassingReportFixture and five new offline tests proving every one of the seven verdict rules individually, each provably load-bearing"
affects: [207-05-messy-practice-project-gate, 207-06-messy-practice-project-gate]

# Actuals (#2632)
actuals:
  tokens: 19043
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "per-trial declared/executed step counts, not just report-level ones: journeyTrial itself carries StepsDeclared/StepsExecuted/IncompleteAtStep so a truncated trial can be named individually inside a multi-trial report, distinct from the report-level fields Plan 01 already used for a single-trial tracer report"
    - "caps sized from a real measurement, revised in place when wrong: the survey step's first-draft $1.00 budget was measured live against this exact repository's own /ant-colonize wrapper (four real surveyor subagents, $2.09 actual cost) and found genuinely too tight -- 207-RESEARCH.md's own 'instrumented, not fixed' framing, now backed by a real number rather than a guess"
    - "a step-specific exception to a general rule, documented at the point of the exception: the 'start' step is the one step that legitimately needs no Bash tool call (a wrapper correctly declining to touch an already-active colony, discovered live), never a silent global weakening of the Bash-call requirement"

key-files:
  created: []
  modified:
    - cmd/journey.go
    - cmd/journey_live_test.go
    - cmd/journey_test.go

key-decisions:
  - "The 'start' step needs no Bash tool call, uniquely among the fourteen steps: scripts/build-messy-practice-project.sh already runs a real `aether init` while constructing the practice project (so every trap has a .aether/ to seed into), so by the time the chat's own /ant-init runs, a colony is already active. Verified live (2026-09-22): the real wrapper, seeing that in its own injected context, correctly explains why it declines and names the real next step (/ant-plan) without running any Bash command at all -- avoiding a wasted, guaranteed-to-fail invocation. Requiring a Bash call here would fail a step that did exactly the right thing; every other step still requires one, since a fresh lifecycle state can only become true through a real command."
  - "Per-step caps were revised from a real measurement, not tightened or widened blind: the survey step's turns/wall-clock/budget (and, proactively, every other subagent-dispatching step's) were raised with a documented safety margin over the observed $2.09/four-subagent/~250s cost, per 207-RESEARCH.md Assumption A2's own 'instrument and adjust' instruction."
  - "AETHER_JOURNEY_TRIALS (development-only trial-count override, forced to 1 in single-step mode) is how this plan's own money-aware verification ran a genuine whole-chain live pass without paying for three trials -- Task 2's own instruction ('Allow the trial count to be lowered for development, and make that impossible to hide'). journeyGateVerdict refuses any report with fewer than journeyMinimumTrials (3) regardless, so this reduced-trial report can never accidentally satisfy the release gate."
  - "journeyRunOneTrial builds a genuinely fresh practice project per trial (its own t.TempDir(), its own scripts/build-messy-practice-project.sh invocation, its own session id) -- 207-04-PLAN.md Task 2's own instruction that three trials must each start clean, not reuse state."
  - "The single-step debugging mode's fast-forward (journeyFastForwardToStep) is deliberately conservative: only 'survey' is actually invoked directly (via a real `aether colonize` call); every step whose own content is genuinely generative (discuss, specification, plan, build, check) is left un-synthesized, logged as a named, non-fatal note, rather than faked in a shape the real runtime could never produce (CLAUDE.md's Definition of Done). This was verified safe for the one target this plan's own <verify> block actually exercises (pause): pauseSafeBoundary only blocks pause on an in-flight build, which a freshly built project can never be in."

requirements-completed: []

coverage:
  - id: D1
    description: "TestJourney runs all fourteen declared lifecycle steps, in declared order, through one chained real claude -p session (--resume from the first step's captured session id), each step carrying its own transcript, its own sized caps, and its own on-disk fact check -- never the chat's own prose"
    requirement: "UED-08"
    verification:
      - kind: integration
        ref: "AETHER_JOURNEY_STEP=pause go test -tags journey -run TestJourney -count=1 -timeout 1200s ./cmd -v -- real claude -p chat, 43s, asserts <command-name>/ant-pause</command-name>, >=1 Bash tool call, COLONY_STATE.json's pause_handoff and .aether/HANDOFF.md, and journeyGateVerdict correctly refuses the resulting one-step report by name"
        status: pass
      - kind: integration
        ref: "AETHER_JOURNEY_TRIALS=1 go test -tags journey -run TestJourney -count=1 -timeout 5400s ./cmd -v (whole-chain, live, real chat, no AETHER_JOURNEY_STEP) -- the start step passes end to end (12s), the survey step's own claude -p invocation genuinely dispatches four real surveyor subagents and completes (transcript/Bash-call checks pass), and the trial stops honestly at the survey step's own on-disk fact check (see Issues Encountered) -- every later step is recorded not-reached, and journeyGateVerdict correctly refuses the resulting one-trial report by name (trial count)"
        status: pass
    human_judgment: false
  - id: D2
    description: "journeyGateVerdict is extended (never forked) with the trial-count and per-trial rules, a closed trial-outcome vocabulary exists, and the status step's own genuine expected-red result is folded into the report"
    requirement: "UED-08"
    verification:
      - kind: unit
        ref: "cmd/journey_test.go#TestJourneyGateVerdictRefusals (seven named subtests, each individually proven load-bearing by disabling its own check and confirming exactly that subtest -- and only that one -- fails), #TestJourneyGateVerdictAcceptsAGenuineRun, #TestJourneyTrialOutcomeVocabularyIsClosed"
        status: pass
      - kind: unit
        ref: "grep -c 'func journeyGateVerdict' cmd/*.go returns 1"
        status: pass
    human_judgment: false
  - id: D3
    description: "Two journeys running at once use separate practice-project directories and separate report files, and an incomplete trial is never counted as passed"
    verification:
      - kind: unit
        ref: "cmd/journey_test.go#TestJourneyTrialsDoNotShareState (concurrent writes to two report paths, race-detector clean), #TestJourneyIncompleteTrialIsNeverPassed"
        status: pass
    human_judgment: false
  - id: D4
    description: "An AETHER_JOURNEY_STEP single-step debugging mode exists, fast-forwards the earlier lifecycle with direct aether commands (never a replayed chat), and the resulting report is structurally refused by the gate (scope=one-step)"
    requirement: "UED-08"
    verification:
      - kind: integration
        ref: "AETHER_JOURNEY_STEP=pause live run (see D1) -- report carries scope: one-step, journeyGateVerdict refuses it naming the scope"
        status: pass
    human_judgment: false
  - id: D5
    description: "Whether the full fourteen-step chain can complete inside a live claude -p run at all is genuinely proven up to the point this run reached, and honestly not proven beyond it -- a real, load-bearing finding, not a silently dropped step"
    verification: []
    human_judgment: true
    rationale: "This plan's own <assumptions> section explicitly frames the full-chain completion question as genuinely unknown going in ('Whether /ant-build and /ant-continue complete inside a sane wall clock in a practice project this messy is genuinely unknown... record it, do not quietly drop the step'). The live whole-chain run in this plan proved two steps end to end (start, survey's own chat/transcript mechanics) and surfaced one real, unfixed finding at survey's own on-disk fact check (see Issues Encountered) -- not a defect in the harness's own mechanics, but an open question about when Aether's own territory refresh actually fires. A human (or a future plan, 207-05/207-06) judging whether that finding is itself a real Aether defect, a mistaken assumption in this plan's own fact check, or expected behavior deferred to /ant-plan is the honest next step; automation cannot resolve which of those three it is from this run alone."

duration: 38min
completed: 2026-09-22
status: complete
---

# Phase 207 Plan 04: The Whole Fourteen-Step Journey, Chained Through One Chat Summary

**`TestJourney` now drives all fourteen lifecycle steps through one chained, real `claude -p` session via `--resume`; `journeyGateVerdict` gained the trial-count and per-trial rules behind a closed outcome vocabulary; a live run proved the harness's own mechanics end to end on two real steps and surfaced one genuine, unresolved finding about territory-refresh timing that is recorded rather than papered over.**

## Performance

- **Duration:** 38 min
- **Started:** 2026-09-22T13:39Z
- **Completed:** 2026-09-22T14:17Z
- **Tasks:** 3
- **Files modified:** 3 (cmd/journey.go, cmd/journey_live_test.go, cmd/journey_test.go)

## Accomplishments

- `cmd/journey_live_test.go`: `TestJourney` is now a table over all fourteen declared steps, chained through one `claude -p` session (`--resume` from the first step's captured session id), each step carrying its own sized caps (turns/wall-clock/budget), its own transcript-based `<command-name>`/Bash-call assertions, and its own on-disk fact check derived from the real runtime writers (`COLONY_STATE.json`'s `goal`/`specification`/`pause_handoff`/`recovery_provenance`/`seal_outcome` fields, the territory snapshot, the planning stage state, `handoffs/worker-handoffs.json`, `gate-results-*.json`, the chamber directory) -- never the chat's own prose. A failing step marks every later step `not-reached` and stops the trial honestly.
- `journeyFastForwardToStep`: an `AETHER_JOURNEY_STEP` single-step debugging mode that fast-forwards the earlier lifecycle with direct `aether` commands rather than replaying the whole chat, verified live end to end against the `pause` step (real chat, 43s, full transcript/on-disk proof, `journeyGateVerdict` correctly refuses the one-step report by name).
- `cmd/journey.go`: a closed trial-outcome vocabulary (`passed`/`flaky-failure`/`real-failure`/`incomplete`, const-block/names/declared-membership convention), `journeyTrial` extended with per-trial `StepsDeclared`/`StepsExecuted`/`IncompleteAtStep`, `journeyEvaluateExpectedRed` (folds the status step's real `statusGuidanceCommandsWithoutMenuWrapper` result into the report), and `journeyGateVerdict` extended to seven ordered refusal rules -- still the one verdict function (`grep -c 'func journeyGateVerdict' cmd/*.go` = 1).
- `cmd/journey_test.go`: `journeyPassingReportFixture` (shaped exactly the way the live harness actually populates a report) plus five new tests -- `TestJourneyGateVerdictRefusals` (seven named subtests, each individually proven load-bearing by disabling its own check), `TestJourneyGateVerdictAcceptsAGenuineRun`, `TestJourneyTrialOutcomeVocabularyIsClosed`, `TestJourneyTrialsDoNotShareState`, `TestJourneyIncompleteTrialIsNeverPassed`. All offline tests run in ~1s, no chat, no money.
- **Live-measured cost correction:** the survey step's original $1.00 budget cap was genuinely too tight -- `/ant-colonize` dispatches four real surveyor subagents and reached $2.09 actual cost before the cap cut it off. Every subagent-dispatching step's caps were raised with a documented safety margin over that real measurement, never a guess (207-RESEARCH.md Assumption A2).

## Task Commits

1. **Task 1 + Task 2 (combined; see Deviations): All fourteen steps chained + the trial loop and finished verdict** - `8fd1cc22` (feat)
2. **Task 3: Offline tests for every verdict rule** - `30019b1f` (test)
3. **Live-run fix: the start step needs no Bash call when a colony already exists** - `deadd8ac` (fix)
4. **Live-run fix: caps raised from a real measured cost, not a guess** - `4e021346` (fix)

_No separate plan-metadata commit yet -- STATE.md/ROADMAP.md/REQUIREMENTS.md are updated and committed after this file is written, per the executor's atomic close-out order._

## Files Created/Modified

- `cmd/journey.go` - trial-outcome vocabulary, extended `journeyTrial`/`journeyReport` structs, `journeyGateVerdict`'s seven ordered rules, `journeyEvaluateExpectedRed`, `journeyDeriveTrialOutcome`, `journeyReportSummary`
- `cmd/journey_live_test.go` - `TestJourney` (fourteen-step table, trial loop), `journeyRunOneTrial`, `journeyDriveStep`, `journeyAssertStepFact` (per-step on-disk fact checks), `journeyAssertPlanSecondFact`, `journeyRunExpectedRedCheck`, `journeyFastForwardToStep`, per-step caps maps
- `cmd/journey_test.go` - `journeyPassingReportFixture` and the five new offline test functions

## Decisions Made

See `key-decisions` in the frontmatter above -- the "start" step's Bash-call exception (discovered and verified live), caps revised from a real measurement rather than a guess, `AETHER_JOURNEY_TRIALS` as the money-aware verification path, per-trial fresh practice projects, and the deliberately conservative scope of the single-step fast-forward mode.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The "start" step incorrectly required a Bash tool call**
- **Found during:** Live whole-chain run, first attempt
- **Issue:** `scripts/build-messy-practice-project.sh` already runs a real `aether init` while constructing the practice project (so every trap has a `.aether/` to seed into). By the time the chat's own `/ant-init` step ran against this already-initialized project, the real wrapper correctly saw an active colony in its own injected context and explained why it declined -- without running any Bash command at all, avoiding a wasted, guaranteed-to-fail invocation. The harness's blanket "every step needs >=1 Bash call" requirement incorrectly failed this correct behavior.
- **Fix:** Made the "start" step the one documented exception to the Bash-call requirement; every other step is unaffected and still requires one.
- **Files modified:** cmd/journey_live_test.go
- **Verification:** Re-run live; `TestJourney/start` passes end to end.
- **Committed in:** deadd8ac

**2. [Rule 1 - Bug] The survey step's $1.00 budget cap was genuinely too tight**
- **Found during:** Live whole-chain run, first attempt
- **Issue:** `/ant-colonize` dispatches four real surveyor subagents (surveyor-provisions, surveyor-nest, surveyor-disciplines, surveyor-pathogens) and reached $2.09 actual cost before the cap cut it off mid-run (`terminal_reason: budget_exhausted`).
- **Fix:** Raised turns/wall-clock/budget for every subagent-dispatching step (survey, discuss, specification, plan-first, plan-second, build, check) with a documented safety margin over the real measurement; the simple relay steps were left unchanged, already proven adequate live.
- **Files modified:** cmd/journey_live_test.go
- **Verification:** Re-run live; the survey step's own `claude -p` invocation completed (no longer budget-exhausted) -- the trial then stopped at a genuinely different, unrelated point (see Issues Encountered), confirming the cap fix itself worked.
- **Committed in:** 4e021346

---

**Total deviations:** 2 auto-fixed (both Rule 1 bugs, both discovered only by actually running the live chat -- exactly the kind of "architectural dead end costs one commit instead of ten" discovery this phase's own objective names as its purpose).
**Impact on plan:** Both fixes were necessary for the harness to genuinely drive real steps through a real chat rather than failing on its own design assumptions. No scope creep.

## Issues Encountered

**The whole-chain live run did not complete all fourteen steps within this session — a genuine, recorded finding, not a silently dropped step.**

Two whole-chain live runs were made (`AETHER_JOURNEY_TRIALS=1`, no `AETHER_JOURNEY_STEP`, so all fourteen steps in declared order):

- **First run:** stopped at the "start" step's own Bash-call check (Deviation 1) and, after that fix, stopped again at the "survey" step when its own `claude -p` invocation hit `budget_exhausted` (Deviation 2).
- **Second run** (after both fixes): "start" passed cleanly (12s). "survey" genuinely ran to completion this time — the chat dispatched four real surveyor subagents, wrote real survey artifacts (`TRAILS.md`, `BLUEPRINT.md`, etc.), and the transcript/Bash-call assertions both passed — but the trial then stopped at survey's own **on-disk fact check**: the territory snapshot's `source_revision` had not moved to the practice project's current HEAD, despite the out-of-date-code-map trap seeding a snapshot pinned to an old revision with 27+ commits added afterward (which `classifySurveyFreshness` should classify `Stale`).

This is exactly the finding 207-RESEARCH.md's own Q3/Q4 table left open: the territory refresh this trap is meant to force is documented there as triggered by "`/ant-colonize` **or** `/ant-plan`" — this run only reached `/ant-colonize`, and demonstrates the refresh did not happen at that point. Whether `/ant-plan`'s own territory-freshness check would pick it back up was not observed in this run, since the trial correctly stopped there rather than continuing past a failed step. Per CLAUDE.md's explicit instruction for this exact situation ("If a lifecycle step genuinely dead-ends inside the real chat... record it... do not weaken the check, and do not edit Aether's lifecycle code to get past it"), this fact check was left exactly as the plan specifies (`the territory snapshot's recorded source revision has moved`) rather than being loosened, and no change was made to Aether's own colonize/territory-refresh logic (which is also outside this plan's declared files).

**What is proven, and what is not:**
- Proven live: session chaining via `--resume` works across steps; per-step caps sized from a real measurement are adequate for `start` and (once fixed) for `survey`'s own chat mechanics; the transcript-based `<command-name>`/Bash-call assertions correctly pass against real, substantial subagent work; a failing step correctly stops the trial and marks later steps `not-reached`; `journeyGateVerdict` correctly refuses both a one-step report (naming the scope) and a reduced-trial-count report (naming the count).
- Not proven in this run: whether `discuss` through `start-again` (the remaining eleven steps) complete live, and whether the territory-refresh timing question above resolves itself by `/ant-plan` or represents a genuine gap. This is recorded as open work, most directly relevant to Plan 05 (the revert-and-rerun proof, which drives specific steps rather than the whole chain) and Plan 06 (the real three-trial run, which this plan's own `<assumptions>` section always intended to be where the full chain's cost and completion are actually measured).

## User Setup Required

None - no external service configuration required. Requires a locally signed-in `claude` CLI, already present on this machine (`claude --version` -> `2.1.278`).

## Next Phase Readiness

- The chained-session mechanics, per-step caps, trial loop, and extended verdict are proven correct by direct live evidence (two full step-passes, one flawless single-step run) and by 23 passing offline tests covering every rule the gate enforces.
- **Load-bearing for Plan 05:** the `AETHER_JOURNEY_STEP` single-step debugging mode is verified working end to end (against `pause`) and is the intended mechanism for driving one specific step at a time during the revert-and-rerun proof -- Plan 05 should budget time to extend `journeyFastForwardToStep`'s conservative scope if a target step it needs genuinely depends on one of the currently un-synthesized earlier steps (discuss/specification/plan/build/check) having run.
- **Load-bearing for Plan 06:** the survey-step cost measurement ($2.09/four subagents) and the now-corrected caps are the first real data point for sizing the eventual three-trial run's own budget; Plan 06 should expect `discuss` through `check` to be at least as expensive per step, and should budget wall-clock in hours, not minutes, for three full trials.
- **Open question carried forward, not resolved here:** whether the out-of-date-code-map trap's territory refresh fires at `/ant-colonize`, at `/ant-plan`, or requires a flag neither wrapper passes by default. This needs either a source read of the colonize/plan wrapper markdown (outside this plan's files) or a live run that continues past survey to observe `/ant-plan`'s own behavior.
- No blockers to Plan 04's own completion; the open question above is scoped follow-up work, not a defect in this plan's own deliverables.

---
*Phase: 207-messy-practice-project-gate*
*Completed: 2026-09-22*

## Self-Check: PASSED

All 3 modified files verified present with `[ -f ]`; all 4 task commit hashes (`8fd1cc22`, `30019b1f`, `deadd8ac`, `4e021346`) verified present in `git log --oneline --all`.
