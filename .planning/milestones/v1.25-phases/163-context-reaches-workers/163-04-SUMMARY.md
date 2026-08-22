---
phase: 163-context-reaches-workers
plan: 04
subsystem: context-delivery
tags: [go, worker-brief, survey, phase-research, staleness, requirements-doc]

# Dependency graph
requires:
  - phase: 163 (plan 163-01, same wave)
    provides: renderCodexBuildWorkerBrief and its resolveSurveySection / resolvePhaseResearchSection call sites, which this plan wires a staleness notice into and pins with a presence test
provides:
  - surveyStalenessNotice() — commits-since-colonize age line, loud past 25 commits, wired into resolveSurveySection
  - TestBuildWorkerBriefIncludesSurveyAndResearch — the presence test CONTEXT-01/04 never had
  - REQUIREMENTS.md CONTEXT-01/04/08/09 reworded to name outcomes achievable against the current tree
affects: [163-05, 163-06, any future phase reading CONTEXT-01/04/08/09 or resolveSurveySection]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Staleness notices load state internally and return \"\" on any failure — same defensive posture as resolveSurveySection, never able to break a build"
    - "State-derived text reaching a subprocess argument is validated (RFC3339 parse) before the exec.Command call, never interpolated into a shell string"

key-files:
  created:
    - cmd/survey_staleness.go
    - cmd/survey_staleness_test.go
  modified:
    - cmd/helpers.go
    - cmd/codex_build_test.go
    - .planning/REQUIREMENTS.md

key-decisions:
  - "surveyStaleCommitThreshold is a named constant (25) with a comment marking it a judgement call, not a measurement"
  - "No new state field: the existing TerritorySurveyed timestamp is sufficient; no backfill needed for colonies created before this change"
  - "The staleness notice only renders when the Territory Survey section itself renders (survey docs present) — a colonized-but-not-yet-surveyed edge case (state set, no survey/ dir) never occurs in practice as the two are set in the same colonize step"
  - "LOUD-01 ('aether survey-load ... works as the playbooks call it') is intentionally untouched — it concerns the CLI subcommand itself (a Phase 160 concern), not the CONTEXT block's worker-prompt-delivery requirements this plan reworded"

patterns-established:
  - "Any future staleness/freshness notice for state-derived data should follow surveyStalenessNotice's shape: validate before touching a subprocess, return \"\" on any failure, name the manual-refresh command rather than auto-refreshing"

requirements-completed: [CONTEXT-01, CONTEXT-04]

# Metrics
duration: ~35min
completed: 2026-07-29
---

# Phase 163 Plan 04: Survey Staleness Notice + Presence Pin + Requirement Rewording Summary

**A commits-since-colonize staleness notice wired into the worker brief's Territory Survey section, a presence test proving survey and phase research reach the actual rendered prompt (with an observed-and-reverted regression proof), and CONTEXT-01/04/08/09 reworded in REQUIREMENTS.md to describe outcomes achievable against the current tree.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-07-29
- **Tasks:** 3/3
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `surveyStalenessNotice()` (`cmd/survey_staleness.go`) tells a worker how old the territory map is: never-surveyed naming `/ant-colonize`, a quiet age line below 25 commits, and a loud STALE warning at or above the threshold — wired into `resolveSurveySection()` immediately under its heading, so both the brief renderer and the brief inspector get it with no further wiring
- `TestBuildWorkerBriefIncludesSurveyAndResearch` (`cmd/codex_build_test.go`) is the presence test CONTEXT-01 and CONTEXT-04 never had: it builds real survey and phase-research artifacts on disk in a temp colony (not stubbed resolvers) and asserts the rendered brief contains the delivered pointer path, the research content, and the staleness notice — plus the negative case that neither section appears, and no empty heading is emitted, when neither artifact exists
- REQUIREMENTS.md's CONTEXT-01, CONTEXT-04, CONTEXT-08, and CONTEXT-09 now name outcomes instead of the deleted `build-context.md`/`codexBuildPlaybooks()`/`survey-load` mechanisms, each carrying the 2026-07-29 date and the D-07/D-08 decision that authorized the wording change; one new row was added to the "Corrections carried forward" table recording that the CONTEXT block's "four things disconnected" premise was half stale by the time this phase executed

## Task Commits

Each task was committed atomically:

1. **Task 1: Say how old the map is, loudly when it is old** - `c5e48cf6` (feat)
2. **Task 2: Pin survey and research presence with the test they never had** - `792e3a03` (test)
3. **Task 3: Reword CONTEXT-01/04, correct CONTEXT-08, descope CONTEXT-09** - `eb9e63ba` (docs)

**Plan metadata:** SUMMARY.md commit follows this document (see final commit note below — orchestrator-owned STATE.md/ROADMAP.md are excluded per worktree mode).

## Files Created/Modified

- `cmd/survey_staleness.go` - `surveyStalenessNotice()` + `commitsSinceSurvey()`; loads colony state, validates RFC3339 before any git call, runs `git rev-list --count --since=<t> HEAD` via a discrete `exec.CommandContext` argv (never a shell string), returns `""` on any failure
- `cmd/survey_staleness_test.go` - `TestSurveyStaleness` with 5 subtests: never-surveyed naming the fix, commit count present, threshold boundary (quiet below 25, loud at/above), malformed timestamp producing empty notice (proving no git invocation, since every branch past the parse step returns non-empty text), and outside-a-git-repo degrading to quiet wording without a count
- `cmd/helpers.go` - `resolveSurveySection()` now calls `surveyStalenessNotice()` and prepends its output inside the `### Territory Survey` section, above the pointer list
- `cmd/codex_build_test.go` - `TestBuildWorkerBriefIncludesSurveyAndResearch` with 4 subtests, built from real files on disk in a temp colony
- `.planning/REQUIREMENTS.md` - CONTEXT-01/04/08/09 reworded; one new row in "Corrections carried forward"

## Decisions Made

- **No new state field, no backfill.** The plan's interfaces block flagged this explicitly (echoing the milestone's own TYPED-01/03 lesson about requiring a field before backfilling it): `TerritorySurveyed` already exists and is sufficient for every colony, old or new.
- **Staleness notice lives inside `resolveSurveySection`'s own conditional, not as an independent section.** It only appears when the Territory Survey section itself appears (survey docs exist on disk). Since `codex_colonize.go` sets `TerritorySurveyed` and writes the survey docs in the same colonize step, there is no real-world case where state is set but survey docs are absent — the theoretical gap (state set, no docs) is a non-issue in practice, and keeping the notice nested avoids a second wiring point across the brief renderer, the hosted path, and the wrapper path.
- **LOUD-01 left untouched.** It reads `` `aether survey-load "{phase_name}"` works as the playbooks call it `` — a different, still-active Phase 160 requirement about the CLI subcommand itself, not about content reaching worker prompts. Task 3's scope was explicitly CONTEXT-01/04/08/09 plus the Corrections table; touching LOUD-01 would have been out of scope.
- **The CONTEXT block's italic intro paragraph** ("Four things are genuinely disconnected...") was deliberately left in place rather than edited in-line — the "Corrections carried forward" table is this project's established place to record a refuted premise (per its own stated purpose and 10 existing precedent rows), so the correction is recorded there once rather than scattered across every place the stale framing appears.

## Deviations from Plan

None — plan executed exactly as written. Two wording adjustments were needed mid-task to satisfy the plan's own acceptance criteria literally (see below), which is expected refinement within Task 3's scope, not a deviation from it:
- The first draft of the new "Corrections carried forward" row used the literal substring `CONTEXT-01/04/08/09`, which made `grep -c 'CONTEXT-0'` return 19 instead of the required unchanged 18 (one new matching line). Reworded to describe the four lines without repeating the `CONTEXT-0` ID prefix.
- The first draft of CONTEXT-08's corrected budget list quoted the stale figure verbatim as `"playbook injection 7K"` for clarity, which the acceptance criteria explicitly forbids re-introducing anywhere in the file (`grep -n 'playbook injection 7K'` must return nothing). Reworded to `"playbook 7K"` so the correction is legible without resurrecting the exact stale phrase.

## Issues Encountered

- **Pitfall 5 grep, recorded per Task 2's instructions:** `grep -n 'func Test' cmd/codex_build_test.go` before writing the new test showed `TestBuildWorkerBriefOmitsHeartbeat`, `TestBuildWorkerBriefOmitsPlaybooks`, `TestBuildWorkerBriefIsMostlyTask`, and `TestBuildWorkerBriefIncludesCodegraphContext` — no existing assertion on `resolveSurveySection` or `resolvePhaseResearchSection`. The coverage gap was confirmed real before writing new tests.
- **Deliberate-regression proof, recorded per Task 2's instructions:** the `resolveSurveySection` call was temporarily removed from `renderCodexBuildWorkerBrief` in `cmd/codex_build.go`. Observed failure (quoted):
  ```
  --- FAIL: TestBuildWorkerBriefIncludesSurveyAndResearch/survey_pointer_list_reaches_the_brief_with_real_paths
      worker brief missing Territory Survey section: ...
  --- FAIL: TestBuildWorkerBriefIncludesSurveyAndResearch/staleness_notice_from_task_1_reaches_the_brief_alongside_the_survey
      worker brief missing the survey staleness notice: ...
  FAIL	github.com/calcosmic/Aether/cmd	0.666s
  ```
  The change was reverted with `git checkout -- cmd/codex_build.go` immediately after observing the failure; `git diff --stat cmd/codex_build.go` confirmed empty before finishing (plan 01 owns that file this wave, and it remains untouched by this plan's final commits).
- **Flaky unrelated test on the full-suite run:** `go test ./cmd/... -count=1` failed once on `TestCLIContinueEnforcesFreshCriterionEvidence` (an unrelated continue-workflow test, no relation to survey/staleness/build-brief code). It passed in isolation and passed on a full-suite rerun, confirming it is a pre-existing flake unrelated to this plan's changes — out of scope per the scope-boundary rule, not auto-fixed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `resolveSurveySection()` now always tells a worker (and the brief inspector) how old the territory map is, satisfying D-10's loud-past-threshold requirement with no automatic refresh path — `/ant-colonize` remains the only way to refresh
- CONTEXT-01 and CONTEXT-04 are pinned by `TestBuildWorkerBriefIncludesSurveyAndResearch` and will fail loudly if either resolver stops reaching the worker brief
- REQUIREMENTS.md's CONTEXT-08/09 now match the actual budget list and validation strategy in the current tree — no stale playbook-injection figure remains, and the staged-benchmark descoping (D-08) is recorded with its replacement evidence named
- No blockers for the remaining phase 163 plans; `cmd/codex_build.go` (owned by plan 01 this wave) is untouched by this plan's committed changes

---
*Phase: 163-context-reaches-workers*
*Completed: 2026-07-29*

## Self-Check: PASSED

- FOUND: cmd/survey_staleness.go
- FOUND: cmd/survey_staleness_test.go
- FOUND: .planning/phases/163-context-reaches-workers/163-04-SUMMARY.md
- FOUND commit: c5e48cf6 (Task 1)
- FOUND commit: 792e3a03 (Task 2)
- FOUND commit: eb9e63ba (Task 3)
- FOUND commit: d59285bf (SUMMARY)
