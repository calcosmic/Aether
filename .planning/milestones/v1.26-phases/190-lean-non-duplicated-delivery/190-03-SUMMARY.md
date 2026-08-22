---
phase: 190-lean-non-duplicated-delivery
plan: 03
subsystem: cli/build-orchestration
tags: [go, cli, worker-dispatch, wrapper-contract, cobra, colony-prime]

# Dependency graph
requires:
  - phase: 190-lean-non-duplicated-delivery
    provides: "190-01's duplicatedBriefSections detector (cmd/build_print_brief.go) and its deferred finding D-190-01-A, which named the exact composing functions this plan fixes"
provides:
  - "composeBuildManifestBrief(root, phase, dispatch, startedAt, includeSteeringSections bool) -- a flag distinguishing the wrapper plan-only flow (capsule-accompanied, steering sections omitted from the brief) from the direct-path manifest artifact (no capsule, steering sections stay self-contained)"
  - "aether build <phase> --print-brief passes cleanly on a real colony with an active pheromone signal and stored worker handoffs -- the manifest-level context_capsule is now the sole channel for both sections in the wrapper flow"
  - "checklistRowForEither -- the --print-brief checklist's Pheromone Signals / Previous Worker Handoffs rows now check both brief and capsule, so they report present accurately regardless of which side currently owns the content"
  - "Three wrapper doc mirrors (build.md canonical + flat + OpenCode) no longer claim the brief itself carries pheromone signals and prior handoffs"
  - "D-190-03-A: a newly-discovered, separate pheromone double-channel on the native/direct dispatch path (executeCodexBuildDispatches' ContextCapsule + standalone PheromoneSection fields), logged and deferred, not fixed here"
affects: [any future phase touching cmd/codex_build.go's dispatch/brief composition, cmd/build_print_brief.go's inspection path, or codex.WorkerDispatch's ContextCapsule/PheromoneSection/HandoffSection fields]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A boolean flag distinguishes two still-valid caller contracts on one shared composer (composeBuildManifestBrief's includeSteeringSections), reusing the exact clearInlineBrief idiom 190-01 established for writeBuildWorkerBriefFiles -- neither caller's existing tests were allowed to regress"
    - "A checklist/inspector row that only checks one of two now-legitimate homes for a section must check both, or it misreports ABSENT for content that is genuinely delivered elsewhere (checklistRowForEither)"
    - "Prove a duplication fix with a positive count == 1 assertion scoped to a single delivery unit (one worker's own assembled context via --worker), not an aggregate multi-worker printout where the count is trivially N for N workers"

key-files:
  created:
    - .planning/phases/190-lean-non-duplicated-delivery/190-03-SUMMARY.md
  modified:
    - cmd/codex_build.go
    - cmd/codex_build_test.go
    - cmd/build_print_brief.go
    - cmd/build_print_brief_test.go
    - cmd/build_manifest_brief_composition_test.go
    - cmd/phase_baseline_test.go
    - .claude/commands/ant/build.md
    - .claude/commands/ant-build.md
    - .opencode/commands/ant/build.md
    - .planning/phases/190-lean-non-duplicated-delivery/deferred-items.md

key-decisions:
  - "OWNERSHIP: the manifest-level context capsule (resolveCodexWorkerContext()) is the sole, canonical channel for both pheromone signals and prior-worker handoffs in the wrapper plan-only flow. composeBuildManifestBrief stops embedding either when a capsule will accompany it."
  - "MECHANISM: includeSteeringSections is a boolean flag on composeBuildManifestBrief, not a behavior change to the function itself -- attachBuildDispatchContext (capsule-accompanied callers) passes false; writeBuildWorkerBriefFiles's own fallback composition (the capsule-less direct-path artifact) passes true, unchanged from before."
  - "A checklist regression this uncovered (Pheromone Signals / Previous Worker Handoffs rows checking brief only) was fixed as part of this task (Rule 1 -- a bug this change directly caused), not deferred."
  - "Wrapper doc prose (3 byte-identical build.md mirrors) updated to stop claiming the brief itself carries pheromone signals and prior handoffs, matching the new contract."
  - "A newly-discovered, SEPARATE pheromone double-channel on the native/direct dispatch path (executeCodexBuildDispatches passes both ContextCapsule and a standalone PheromoneSection into AssemblePrompt/AssembleHostedPrompt, which join them as two independent parts) was proven by a throwaway probe, logged as D-190-03-A, and NOT fixed -- composeBuildManifestBrief's output is never read by that path at all, and fixing it means an architectural decision about a struct shared by 8 call sites (build/continue/plan/colonize/quick/oracle/seal/swarm), outside this plan's files_modified."

requirements-completed: []

# Metrics
duration: ~110min
completed: 2026-08-20
---

# Phase 190 Plan 03: Lean, Non-Duplicated Delivery (one home each) Summary

**Pheromone signals and prior-worker handoffs now reach a wrapper-spawned build worker exactly once each, via the manifest-level context capsule alone -- composeBuildManifestBrief's per-dispatch brief no longer renders its own competing copy when a capsule will also be prepended.**

## Performance

- **Duration:** ~110 min
- **Completed:** 2026-08-20
- **Tasks:** 1 (single cohesive architectural fix, per this gap-closure plan's own framing)
- **Files modified:** 10 (2 Go source, 4 Go test, 3 wrapper markdown, 1 planning doc)

## Accomplishments

- **Root cause confirmed and fixed.** `resolveCodexWorkerContext()` (the manifest-level capsule,
  `cmd/colony_prime_context.go:571,695`) and `composeBuildManifestBrief` (the per-dispatch brief,
  `cmd/codex_build.go`) independently rendered `## Pheromone Signals` and `## Previous Worker Handoffs`
  from the same underlying data. Since the wrapper contract prepends the capsule once ahead of every
  dispatch's brief, a wrapper-spawned worker's assembled context carried each section twice whenever
  either was active. `composeBuildManifestBrief` now takes `includeSteeringSections bool`: its one
  caller that also carries a capsule (`attachBuildDispatchContext`, used by the plan-only wrapper flow
  and by `--print-brief`'s simulation of it) passes `false`; the one caller with no accompanying capsule
  (`writeBuildWorkerBriefFiles`'s fallback composition for the direct/native path's manifest.json
  artifact) keeps passing `true`, unchanged.
- **Fail-then-pass proven, not asserted.** Before this fix, a fixture colony with one active pheromone
  signal and one stored worker handoff made `aether build <phase> --print-brief` fail with:
  `"dispatch Dash-21 delivers duplicated context: Pheromone Signals, Previous Worker Handoffs (each
  owned section and the handoff schema must appear exactly once in the assembled worker context)"`.
  The identical fixture now passes cleanly on both the checklist and `--full` code paths, with
  once-ness asserted positively (`strings.Count(...) == 1`, scoped to one worker via `--worker`, not an
  absence-only check that would also pass on zero).
- **A real, additional bug was found and fixed while making this change (Rule 1).**
  `renderBriefChecklist`'s Pheromone Signals / Previous Worker Handoffs rows only ever searched the
  brief text for their heading. Once that content moved to the capsule exclusively, those rows would
  have silently misreported `ABSENT` for content the worker genuinely still receives -- the exact
  inverse of the misreport the checklist's own doc comment already warns against. `checklistRowForEither`
  now checks both brief and capsule.
- **Both delivery paths traced before removal, per this plan's own "not zero" constraint.** The
  direct/native `aether build <phase>` path's manifest.json artifact carries no `context_capsule` at
  all (confirmed: `manifest.ContextCapsule == ""` for that path, proven by a new test), so its brief
  file must and does keep the full, self-contained composition -- proven with a positive count == 1
  assertion, not just "still non-empty."
- **A second, separate duplication mechanism was discovered and deliberately NOT fixed.** The native
  dispatch path (`executeCodexBuildDispatches`) independently passes both `ContextCapsule` (which
  already contains `## Pheromone Signals`) and a standalone `PheromoneSection` field into
  `AssemblePrompt`/`AssembleHostedPrompt`, which join them as two separate, both-included parts. This
  is structurally unrelated to `composeBuildManifestBrief` (that composer's output is never read on
  this path at all) and touches a struct shared by 8 call sites across build/continue/plan/colonize/
  quick/oracle/seal/swarm -- a genuine architectural question outside this plan's scope. Proven real by
  a throwaway probe (deleted after use), logged as `D-190-03-A` in `deferred-items.md` with the same
  rigor as `D-190-01-A`.

## Task Commits

This gap-closure plan was executed and committed as one cohesive fix (the ownership decision, the
checklist fix it surfaced, and the wrapper-prose sync are inseparable parts of the same change):

1. **Fix: pheromone signals and prior handoffs get one home in build briefs** - `2698bad7` (fix)

## Files Created/Modified

- `cmd/codex_build.go` - `composeBuildManifestBrief` gains `includeSteeringSections bool`;
  `attachBuildDispatchContext` passes `false` (capsule-accompanied), `writeBuildWorkerBriefFiles`'s
  fallback passes `true` (capsule-less artifact); doc comments explain the ownership decision inline
- `cmd/build_print_brief.go` - `checklistRowForEither` fixes the checklist misreport this change
  uncovers for the Pheromone Signals / Previous Worker Handoffs rows
- `cmd/codex_build_test.go` - Updates `composeBuildManifestBrief` call sites for the new parameter;
  adds `TestDirectBuildManifestHasNoCapsuleSoBriefStaysTheSoleSteeringChannel` (PATH B proof: no
  capsule, brief stays sole channel, exactly once)
- `cmd/build_print_brief_test.go` - Replaces `TestPrintBriefCommandFailsWhenPrintWorkerBriefsFindsDuplication`
  (which proved the bug) with `TestPrintBriefStaysCleanWithActiveSignalAndStoredHandoffs` (PATH A proof:
  clean pass, checklist reports present, `--full --worker` scoped output has exactly one of each heading)
- `cmd/build_manifest_brief_composition_test.go` - `TestPlanOnlyManifestBriefCarriesPheromones` now
  asserts the signal reaches `manifest.ContextCapsule` and explicitly asserts the brief_path file does
  NOT also carry it (proving the duplication is gone, not just that one copy survived)
- `cmd/phase_baseline_test.go` - Updates two unrelated `composeBuildManifestBrief` call sites for the
  new parameter (phase-baseline content is unaffected by the flag either way)
- `.claude/commands/ant/build.md`, `.claude/commands/ant-build.md`, `.opencode/commands/ant/build.md`
  - Identical rewrite: the brief's own parenthetical content list no longer claims pheromone signals or
    prior handoffs; `context_capsule`'s description now states it is their sole source. Verified
    byte-identical after editing.
- `.planning/phases/190-lean-non-duplicated-delivery/deferred-items.md` - D-190-01-A marked RESOLVED
  with a pointer to this summary; new `D-190-03-A` entry logs the native-path pheromone finding

## Decisions Made

See `key-decisions` in frontmatter. The most consequential: **the capsule owns pheromones and handoffs
for the build wrapper flow**, not the per-dispatch brief. Reasoning, verified against code (not just
inferred from CLAUDE.md's prose, per this plan's own instruction to check both readings):

1. **Pheromones -- CLAUDE.md is a direct tiebreaker.** Colony-prime (the capsule) is documented as the
   injector, and pheromone signals are named as the *highest-retention-priority* section in colony-prime's
   own token-budget trim order ("Pheromone signals (trimmed last -- highest retention priority)").
   Moving them out of the capsule would contradict that documented design; keeping the capsule as sole
   owner does not.
2. **Handoffs -- verified by reading, not assumed.** `buildColonyPrimeOutput` (the capsule) already,
   independently, unconditionally renders `## Previous Worker Handoffs` via the exact same
   `renderWorkerHandoffSection` function the per-dispatch brief also calls, whenever handoff records
   exist for the SAME plan-only flow this plan targets. The capsule was never a "zero" alternative for
   handoffs -- it was already delivering them, redundantly, alongside the brief's own copy. This mirrors
   the pheromone case exactly, so the same ownership answer applies.
3. **Mechanism reuses, rather than reinvents, an established idiom.** `includeSteeringSections` is the
   same "flag on a shared function distinguishes two still-valid callers" pattern 190-01's
   `clearInlineBrief` already established for `writeBuildWorkerBriefFiles` -- named explicitly as
   follow-up option (a) in D-190-01-A's own "Recommended follow-up" section.

**A genuinely separate finding was deliberately NOT folded into the fix.** The native dispatch path's
own pheromone double-channel (`D-190-03-A`) is structurally independent of everything this plan changed
-- `composeBuildManifestBrief`'s output is never consumed there. Fixing it would mean deciding whether
`codex.WorkerDispatch`'s `PheromoneSection`/`HandoffSection` fields should exist at all once
`ContextCapsule` already carries the same content, across 8 unrelated call sites this plan's
`files_modified` does not cover -- a new architectural question for a future phase, not a mechanical
extension of this one.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug caused by this change] Fixed the --print-brief checklist's Pheromone Signals /
Previous Worker Handoffs rows misreporting ABSENT**
- **Found during:** Implementation, first test run after the composeBuildManifestBrief change
- **Issue:** `renderBriefChecklist`'s two rows for these sections used `checklistRowFor(brief, ...)`,
  which only ever searched the per-dispatch brief text. Once the brief stopped embedding these sections
  (this plan's own fix), the rows would report `ABSENT` even though the worker genuinely still receives
  the content via the capsule -- the checklist's own doc comment states it must report what the worker
  *actually receives*, so this would have been a fresh accuracy bug introduced by the fix itself.
- **Fix:** Added `checklistRowForEither(brief, capsule, label, heading)`, which checks both sources and
  reports present if either carries the heading. Wired in for the two affected rows only; the other
  checklist rows (Territory Survey, Phase Research, etc.) are unaffected since nothing moved their
  content.
- **Files modified:** `cmd/build_print_brief.go`
- **Committed in:** `2698bad7`

**2. [Rule 2 discovery, NOT auto-fixed -- logged as deferred] The native dispatch path independently
double-delivers pheromone signals through a separate, unrelated channel**
- **Found during:** Tracing "both delivery paths" per this plan's own architectural-decision constraint
  2 (the requirement to confirm neither path drops to zero before removing anything)
- **Issue:** `executeCodexBuildDispatches` passes both a freshly-resolved `ContextCapsule` (which already
  contains `## Pheromone Signals`) and a separately-resolved, standalone `PheromoneSection` field into
  `AssemblePrompt`/`AssembleHostedPrompt`, which concatenate them as two independent, both-included
  prompt parts.
- **Why not fixed:** Structurally unrelated to this plan's actual change -- `composeBuildManifestBrief`'s
  output is never read by this path at all (it builds `TaskBrief` from the raw, un-composed
  `renderCodexBuildWorkerBrief`). Fixing it means an architectural decision about whether
  `codex.WorkerDispatch`'s dedicated `PheromoneSection`/`HandoffSection` fields should exist at all once
  `ContextCapsule` already carries the same content, across 8 call sites (build, continue, plan,
  colonize, quick, oracle, seal, swarm) this plan's `files_modified` does not cover.
- **Verification that this is real, not hypothetical:** A throwaway probe (deleted after use) seeded one
  active FOCUS signal and called the exact two functions `executeCodexBuildDispatches` calls
  (`resolveCodexWorkerContext()`, `resolvePheromoneSection()`) directly -- both returned non-empty, both
  contained the identical signal text, under two different headings (`## Pheromone Signals` vs.
  `### Active Pheromone Signals`).
- **Logged in:** `.planning/phases/190-lean-non-duplicated-delivery/deferred-items.md` (`D-190-03-A`)

---

**Total deviations:** 2 (1 auto-fixed checklist bug this change directly caused, 1 discovered-and-deferred
separate architectural finding)
**Impact on plan:** The auto-fixed checklist bug was necessary -- without it, this plan's own fix would
leave the inspector tool lying about delivery. The deferred finding does not block this plan's stated
success criteria (which are scoped to the plan-only wrapper flow `--print-brief` inspects) but is
flagged prominently, matching the rigor `D-190-01-A` itself demonstrated, since it is a real,
already-shipping duplicate on a different, live path.

## Issues Encountered

**A worktree-branch staleness issue at startup, unrelated to this plan's own work.** The assigned
worktree's HEAD was on a stale commit (`6577f51c`, far older than Phase 190) rather than the expected
base (`3e26c279`). The mandatory `<worktree_branch_check>` step's own `git reset --hard` to the expected
base commit resolved this automatically before any plan work began; flagged here only because it means
this plan's very first actions included a corrective reset, not because it affected the outcome.

**An early test-design bug in this plan's own proof, caught and fixed before committing.** My first
attempt at the `--full` mode count-== 1 assertion counted headings across the ENTIRE multi-dispatch
`--print-brief --full` output (5 dispatches in the fixture), which trivially found 5 occurrences, not 1
-- `--full` composes one section per dispatch by design. Fixed by scoping the count to a single
dispatch via `--worker`, which is what "exactly once per delivery path" actually means. Caught by
running the test before committing, not shipped.

## Next Phase Readiness

- `D-190-03-A` (the native dispatch path's separate pheromone double-channel) is ready for a future
  phase to pick up; it names the exact fields, call sites, and the throwaway-probe proof pattern to
  reuse.
- ROADMAP Phase 190's success criterion 2 ("`--print-brief` asserts zero duplicated sections --
  pheromones and handoffs get one home each") is now fully satisfied: the detector (190-01) both exists
  and finds nothing on a real colony with active steering data (190-03).

---
*Phase: 190-lean-non-duplicated-delivery*
*Completed: 2026-08-20*
