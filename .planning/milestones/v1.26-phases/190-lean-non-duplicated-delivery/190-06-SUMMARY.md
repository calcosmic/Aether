---
phase: 190-lean-non-duplicated-delivery
plan: 06
subsystem: cli/worker-dispatch
tags: [go, cli, worker-dispatch, handoffs, colony-prime, codex]

# Dependency graph
requires:
  - phase: 190-lean-non-duplicated-delivery
    provides: "190-05's per-caller audit discipline and its deferred finding D-190-05-A, which named continue's native review/watcher dispatch paths' separate HandoffSection double-channel this plan closes"
provides:
  - "D-190-05-A closed: continue's native review and watcher dispatches no longer carry \"## Previous Worker Handoffs\" twice -- the capsule's build-workflow carryover and the dedicated field's continue-workflow sibling relay each keep exactly one heading"
  - "renderRelatedWorkflowHandoffSection: a new, distinctly-headed (\"## Related Worker Handoffs\") sibling to renderWorkerHandoffSection, for callers whose ContextCapsule already renders the standard heading via a DIFFERENT workflow filter"
  - "Four additional native dispatch paths D-190-05-A never named -- colonize, plan, seal, swarm -- found to share the identical shape (empirically, via throwaway probe) and fixed the same way"
  - "codexWorkerDispatchesForRecovery's HandoffSection (a TRUE duplicate, same \"build\" workflow tag as the capsule) removed outright, mirroring 190-05's PheromoneSection fix at that exact call site"
  - "TestContinueReviewHandoffStaysExactlyOnceViaOwnHeading + TestContinueWatcherHandoffStaysExactlyOnceViaOwnHeading: fail-then-pass proof for both named dispatch functions"
  - "TestNineCommandsDeliverHandoffExactlyOnce: 9-case breadth lock for handoff once-ness, mirroring 190-05's TestEightCommandsDeliverPheromoneExactlyOnce structure exactly"
affects: [any future phase touching codex.WorkerDispatch/WorkerConfig's ContextCapsule/HandoffSection fields, renderWorkerHandoffSection, or the colonize/plan/continue/seal/swarm native dispatch functions]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Distinct heading over deletion when two channels carry MATERIALLY DIFFERENT content: 190-05 deleted a duplicate field because both channels rendered byte-identical text; this plan could not use the same fix because the capsule (hardcoded to \"build\"-workflow handoffs) and the dedicated field (each caller's OWN workflow) filter on different, non-overlapping data. Blind deletion would have dropped real, previously-shipped content to zero. renderRelatedWorkflowHandoffSection keeps both channels alive under distinct headings instead."
    - "A per-caller audit's own instructions can reveal the plan's named target is a proper subset of the real bug: enumerating every HandoffSection construction site (not just the two named in this plan's objective) found the identical shape on four more commands. Verified empirically (throwaway probe, deleted after use) before fixing all of them, rather than leaving a proven-reproducible instance of the exact bug this plan closes for a future phase to (re)discover -- see Decisions Made for the Definition-of-Done reasoning."
    - "Throwaway probe workflow (mirrors 190-05): seed real handoff records with explicit workflow tags via persistDispatchWorkerHandoff, call the real dispatch function, assemble the prompt via the real AssembleHostedPrompt, count headings -- proves the actual wiring, not a parallel computation that merely agrees with the fix."

key-files:
  created:
    - cmd/build_handoff_190_06_test.go
    - .planning/phases/190-lean-non-duplicated-delivery/190-06-SUMMARY.md
  modified:
    - cmd/codex_dispatch_contract.go
    - cmd/codex_continue.go
    - cmd/codex_colonize.go
    - cmd/codex_plan.go
    - cmd/seal_final_review.go
    - cmd/swarm_cmd.go
    - cmd/codex_build_finalize.go
    - .planning/phases/190-lean-non-duplicated-delivery/deferred-items.md

key-decisions:
  - "FIX SHAPE: a new function (renderRelatedWorkflowHandoffSection) with its own heading, not deletion. The capsule's handoff section (cmd/colony_prime_context.go:695) is hardcoded to workflow=\"build\", unconditionally, for every one of its 7 callers (build/continue x2/plan/colonize/seal/swarm) -- it structurally CANNOT render a \"continue\"-, \"colonize\"-, \"plan\"-, \"seal\"-, or \"swarm\"-workflow handoff. Deleting the dedicated field (190-05's exact fix for pheromones, where both channels rendered identical content) would have silently dropped that content to zero -- the exact trap 190-06's own instructions warned against. Both channels now render, each under a heading unique to it."
  - "SCOPE: fixed all 6 real instances found (continue review, continue watcher, colonize, plan, seal, swarm), not just the 2 named in this plan's objective. The per-caller audit this plan's own instructions required (enumerate EVERY HandoffSection construction site) empirically proved (throwaway probe seeding real workflow-tagged handoffs, calling the real dispatch functions, counting real assembled-prompt headings) that colonize, plan, seal, and swarm share the IDENTICAL bug shape D-190-05-A named for continue. This project's own Definition of Done explicitly rejects declaring a 'final gap of the phase' closed while a proven, reproduced instance of the same bug remains undiscovered-but-discoverable (CLAUDE.md's Definition of Done section cites this exact failure pattern -- 18 of 25 milestones framed around restoring something previously marked done). Per this repo's CLAUDE.md, which takes precedence over plan instructions when they conflict, and per the plan's own proof_requirement (\"handoff once-ness is locked across ALL construction sites\", with no scope qualifier), all 6 were fixed with the identical mechanism and locked by the same breadth test."
  - "codexWorkerDispatchesForRecovery (build's retry-instruction builder) is a DIFFERENT case from the 6 above: its HandoffSection used the SAME \"build\" workflow tag as the capsule -- a true, byte-identical duplicate, not a materially-different one. Removed outright (not given a new heading), mirroring 190-05's own PheromoneSection fix at this exact call site. Its WorkerDispatch values are still never passed through AssemblePrompt today (unchanged from 190-05's finding), so this closes a latent trap rather than a live bug."
  - "quick and oracle are UNCHANGED. Their capsules (renderQuickContextCapsule, renderOracleContextCapsule) never render handoffs at all (confirmed by reading both functions in full) -- HandoffSection via the ORIGINAL, unrenamed renderWorkerHandoffSection remains their sole, correct channel, exactly as 190-05 established for their PheromoneSection."
  - "build's native path (executeCodexBuildDispatches) is UNCHANGED and was not touched. HandoffSection is never set on that path (attachBuildDispatchContext, the only setter, is never called there) -- already locked by TestNativeDispatchHandoffStaysExactlyOnceViaCapsule, which passes unmodified."

requirements-completed: []

# Metrics
duration: ~90min
completed: 2026-08-21
---

# Phase 190 Plan 06: Lean, Non-Duplicated Delivery (continue's handoff relay) Summary

**Closed D-190-05-A: continue's native review and watcher dispatches, plus four more native dispatch paths the same audit discovered (colonize, plan, seal, swarm), stopped delivering "## Previous Worker Handoffs" twice -- not by deleting the second channel (which would have silently dropped real sibling-relay content to zero) but by giving it a distinct "## Related Worker Handoffs" heading, proven with a fail-then-pass test per continue function and a 9-case breadth lock mirroring 190-05's pheromone suite.**

## Performance

- **Duration:** ~90 min
- **Completed:** 2026-08-21
- **Tasks:** 1 (single cohesive fix, matching 190-05's and this gap-closure plan's own framing)
- **Files modified:** 9 (7 Go source, 1 new Go test file, 1 planning doc)

## Per-Caller Audit Table

Required by this plan's own `<the_audit_before_the_cut>` discipline: for every `HandoffSection:`
construction site in `cmd/` (grepped, not sampled), whether its `ContextCapsule` already renders
handoff content that would collide, and what this plan's fix does there.

| # | Command | Construction site | `ContextCapsule` source | Capsule's handoff filter | Dedicated field's (pre-fix) filter | Same content? | Fix |
|---|---------|-------------------|--------------------------|---------------------------|--------------------------------------|----------------|-----|
| 1 | build (native) | `executeCodexBuildDispatches`, `cmd/codex_build.go` | `resolveCodexWorkerContext()` | workflow="build" | *(field never set on this path)* | N/A | Unchanged -- already locked (`TestNativeDispatchHandoffStaysExactlyOnceViaCapsule`) |
| 2 | continue (review) | `plannedContinueReviewDispatches`, `cmd/codex_continue.go` | `resolveCodexWorkerContext()` | workflow="build" | workflow="continue" | **NO** -- disjoint workflows | `renderRelatedWorkflowHandoffSection` (new heading) |
| 3 | continue (watcher) | `plannedContinueWatcherDispatch`, `cmd/codex_continue.go` | `resolveCodexWorkerContext()` | workflow="build" | workflow="continue" | **NO** | `renderRelatedWorkflowHandoffSection` (new heading) |
| 4 | plan | `dispatchRealPlanningWorkersWithIterationContext`, `cmd/codex_plan.go` | `resolveCodexWorkerContext()` | workflow="build" | workflow="plan" | **NO** | `renderRelatedWorkflowHandoffSection` (new heading) |
| 5 | colonize | `dispatchRealSurveyorsWithTimeout`, `cmd/codex_colonize.go` | `resolveCodexWorkerContext()` | workflow="build" | workflow="colonize" | **NO** | `renderRelatedWorkflowHandoffSection` (new heading) |
| 6 | seal | `plannedSealFinalReviewDispatches`, `cmd/seal_final_review.go` | `resolveCodexWorkerContext()` | workflow="build" | workflow="seal" | **NO** | `renderRelatedWorkflowHandoffSection` (new heading) |
| 7 | swarm | `invokeSwarmWorker`, `cmd/swarm_cmd.go` | `resolveCodexWorkerContext()` | workflow="build" | workflow="swarm" | **NO** | `renderRelatedWorkflowHandoffSection` (new heading) |
| 8 | quick | `runQuickScout`, `cmd/command_truth.go` | `renderQuickContextCapsule()` (custom) | *(none -- no handoffs rendered)* | workflow="quick" | N/A -- sole channel | Unchanged |
| 9 | oracle | `buildOracleWorkerConfig`, `cmd/oracle_loop.go` | `renderOracleContextCapsule()` (custom) | *(none -- no handoffs rendered)* | workflow="oracle" | N/A -- sole channel | Unchanged |

**Also audited and found a TRUE duplicate (not part of the 9 above -- inert today, fixed for
consistency):**

- `codexWorkerDispatchesForRecovery` (`cmd/codex_build_finalize.go`, build's retry-instruction
  builder): capsule filter workflow="build", dedicated field filter workflow="build" -- **the SAME
  workflow, same content shape as 190-05's pheromone case**, not the "materially different" case the
  6 above are. `HandoffSection` removed outright (not renamed), mirroring 190-05's own
  `PheromoneSection` fix at this exact call site. Its `WorkerDispatch` values are still never passed
  through `AssemblePrompt`/`AssembleHostedPrompt` today (traced the same way 190-05 did, to
  `buildExternalBuildRecoveryInstructions`'s in-memory same-caste peer lookup) -- this closes a latent
  trap, not a live bug.

**Also audited and found not a construction-site risk:**

- `cmd/colony_prime_context.go:695` (inside `buildColonyPrimeOutputOpts`): this IS the capsule's own
  `renderWorkerHandoffSection("build", state.CurrentPhase, "")` call every one of the 7
  capsule-based callers above inherits. It is the shared *source* of the collision, not itself a
  second, independent `HandoffSection:` field -- listed here for completeness, not as a 10th fix
  site.
- `cmd/build_print_brief.go:437` (`expectedBriefSections`'s `Expected` predicate for the print-brief
  absence gate): calls `renderWorkerHandoffSection("build", state.CurrentPhase, "")` to ask "does the
  capsule have handoff content to deliver", exactly mirroring the capsule's own call. This is an
  existence check for the wrapper/plan-only path's absence gate (190-`REVIEW` WR-02), structurally
  unrelated to the native/direct dispatch paths this plan targets -- unchanged, correctly so.

(9 breadth-test cases across 8 commands -- continue has two: review and watcher -- plus 1 true
duplicate fixed outside the breadth test, plus 2 non-risk sites confirmed inert.)

## Accomplishments

- **D-190-05-A confirmed exactly as described, then found to be one instance of a wider pattern.**
  `plannedContinueReviewDispatches` and `plannedContinueWatcherDispatch` both set `ContextCapsule:
  resolveCodexWorkerContext()` (which unconditionally renders `## Previous Worker Handoffs` for
  "build"-workflow records, `cmd/colony_prime_context.go:695`) AND `HandoffSection:
  renderWorkerHandoffSection("continue", ...)` (the SAME heading, for "continue"-workflow records) --
  reproduced empirically with a throwaway probe seeding both a build- and a continue-tagged handoff
  and counting headings in the real assembled prompt: 2, not 1. Auditing the other named
  `HandoffSection` construction sites per this plan's own instructions found the IDENTICAL shape,
  unnamed by D-190-05-A, at colonize, plan, seal, and swarm -- confirmed the same way, not assumed.
- **The fix is NOT 190-05's fix, because the content is not the same.** 190-05 deleted
  `PheromoneSection` at 7 call sites because the capsule already rendered byte-identical pheromone
  text. Here, the capsule's handoff section is hardcoded to workflow="build" (a fixed property of
  `cmd/colony_prime_context.go:695`, unconditional for every one of its 7 callers), while each
  dedicated field filtered its OWN workflow ("continue", "colonize", "plan", "seal", "swarm") --
  disjoint, non-overlapping record sets. `plannedContinueWatcherDispatch`'s own comment ("nothing in
  the design justified the asymmetry; it was omitted") confirms the continue-sibling relay was a
  deliberate, wanted feature, not accidental duplication -- deleting it would have been a silent
  regression, exactly the "zero trap" this plan's own instructions warned against. `renderHandoffSectionNamed`
  (the shared implementation behind both `renderWorkerHandoffSection` and the new
  `renderRelatedWorkflowHandoffSection`) lets each channel keep its content under its own heading
  instead.
- **The "zero" trap was checked, not assumed, on both sides.** For the 6 fixed call sites, the
  breadth test asserts BOTH the capsule's build-carryover sentinel AND the dedicated field's
  own-workflow sentinel appear exactly once each in the assembled prompt -- proving neither channel's
  content silently disappeared while the duplicate heading was being fixed. For quick/oracle
  (confirmed via direct reading of `renderQuickContextCapsule`/`renderOracleContextCapsule`: neither
  references handoffs at all) and for build's native path (already locked, unmodified), nothing was
  touched.
- **Fail-then-pass proven with the real wiring.** `git stash` was used to temporarily revert only
  the 7 source files (keeping the new test file in place) and re-run the exact permanent tests
  against pre-fix code: all 6 affected sub-tests failed with `"assembled prompt has 2 \"## Previous
  Worker Handoffs\" headings, want exactly 1"`, `build_native`/`quick`/`oracle` correctly passed
  (never broken), then the stash was restored and the same tests re-run to confirm all 9 pass. See
  Deviations/Issues below for the exact captured failure text.

## Task Commits

This gap-closure plan was executed and committed as one cohesive fix:

1. **Fix: give continue's handoff relay its own heading, not a deleted field** - `7c88c97f` (fix)

## Files Created/Modified

- `cmd/codex_dispatch_contract.go` - `renderWorkerHandoffSection` now delegates to a new
  `renderHandoffSectionNamed(workflow, phaseID, workerName, sectionName, fallbackHeading)` helper
  (identical filtering/formatting, parameterized heading); its own behavior and heading
  ("## Previous Worker Handoffs") are 100% unchanged for every existing caller. New
  `renderRelatedWorkflowHandoffSection` calls the same helper with a distinct heading
  ("## Related Worker Handoffs")
- `cmd/codex_continue.go` - `plannedContinueReviewDispatches` and `plannedContinueWatcherDispatch`
  now call `renderRelatedWorkflowHandoffSection` instead of `renderWorkerHandoffSection`
- `cmd/codex_colonize.go` - `dispatchRealSurveyorsWithTimeout` now calls
  `renderRelatedWorkflowHandoffSection`
- `cmd/codex_plan.go` - `dispatchRealPlanningWorkersWithIterationContext` now calls
  `renderRelatedWorkflowHandoffSection`
- `cmd/seal_final_review.go` - `plannedSealFinalReviewDispatches` now calls
  `renderRelatedWorkflowHandoffSection`
- `cmd/swarm_cmd.go` - `invokeSwarmWorker` now calls `renderRelatedWorkflowHandoffSection`
- `cmd/codex_build_finalize.go` - `codexWorkerDispatchesForRecovery` no longer sets
  `HandoffSection` at all (true duplicate of the capsule's own "build"-workflow content, removed
  outright)
- `cmd/build_handoff_190_06_test.go` (new) - `TestContinueReviewHandoffStaysExactlyOnceViaOwnHeading`
  and `TestContinueWatcherHandoffStaysExactlyOnceViaOwnHeading` (primary fail-then-pass proof for the
  two named dispatch functions) and `TestNineCommandsDeliverHandoffExactlyOnce` (9-case breadth lock,
  one sub-test per command, asserting the capsule's heading and the related-handoff heading each
  appear exactly once, with both sentinel contents present)
- `.planning/phases/190-lean-non-duplicated-delivery/deferred-items.md` - D-190-05-A marked
  RESOLVED with a pointer to this summary

## Decisions Made

See `key-decisions` in frontmatter. The most consequential: **fixing all 6 real instances found
(continue review, continue watcher, colonize, plan, seal, swarm), not only the 2 named in this
plan's objective.**

This plan's objective named only continue's review and watcher dispatches. Its own
`<the_audit_before_the_cut>` instructions, however, required enumerating EVERY `HandoffSection`
construction site and checking each one's relationship to its capsule -- and its `proof_requirement`
asked for "handoff once-ness ... locked across ALL construction sites the same way signal once-ness
now is" (190-05's breadth lock had zero exceptions). Following that instruction to the letter, before
writing any fix, surfaced that colonize, plan, seal, and swarm share the byte-for-byte identical bug
shape D-190-05-A named for continue: capsule hardcoded to `workflow="build"`
(`cmd/colony_prime_context.go:695`, a fixed property shared by all 7 callers of
`resolveCodexWorkerContext()`), each command's own dedicated field independently filtered to its OWN
workflow tag. This was verified empirically -- not assumed from the pattern match -- with a throwaway
probe (deleted after use) that seeded real handoff records with each command's real workflow tag,
called the real dispatch function, and counted real headings in the real assembled prompt. All four
showed 2 headings before the fix, 1 after.

Leaving those four in a proven-broken state while declaring D-190-05-A ("the last known instance") and
this plan ("the final gap of the phase") closed would have reproduced the exact pattern this
project's own CLAUDE.md Definition of Done exists to prevent: *"18 of 25 milestones have been framed
around restoring, recovering or repairing something previously marked done."* Per my operating
instructions, CLAUDE.md directives take precedence over plan instructions when they conflict; there
was no actual conflict here, since the plan's own proof_requirement already asked for an
all-construction-sites breadth lock -- but had the two readings diverged, CLAUDE.md's would have
governed. All 6 were fixed with the identical `renderRelatedWorkflowHandoffSection` mechanism and are
locked by the same breadth test, so "closed" in this summary means what it says.

`codexWorkerDispatchesForRecovery` was a genuinely different case (a TRUE duplicate, matching 190-05's
pheromone shape exactly, not the "materially different" shape the other 6 share) and was fixed by
removal, not renaming -- mirroring 190-05's own precedent at that same call site, and not treated as
part of the "6 real instances" scope decision above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Formatting] `gofmt` realignment after inserting a doc comment inside a struct-literal
field-alignment block**
- **Found during:** Pre-commit formatting check (`gofmt -l`)
- **Issue:** Inserting a multi-line doc comment between two aligned struct-literal fields in
  `cmd/codex_plan.go` broke `gofmt`'s column-alignment grouping (the comment splits what `gofmt`
  treats as one alignment block into two, each aligned independently)
- **Fix:** Ran `gofmt -w cmd/codex_plan.go`
- **Verification:** `gofmt -l` reports clean; `go build ./...` and the targeted test set both still
  pass after the reformat
- **Files modified:** `cmd/codex_plan.go`
- **Committed in:** `7c88c97f` (part of the fix commit -- caught before this file was ever committed)

**2. [Rule 2 - scope expansion required by this plan's own proof_requirement, documented above under
Decisions Made] Fixed colonize, plan, seal, and swarm's native dispatch paths, and removed
`codexWorkerDispatchesForRecovery`'s true-duplicate `HandoffSection`, beyond the 2 dispatch functions
this plan's objective named**
- **Found during:** The per-caller audit this plan's own instructions required, before any fix was
  written
- **Issue:** Four commands beyond continue shared the identical "capsule hardcoded to `build`-workflow
  handoffs + dedicated field filtered to the command's own workflow" shape D-190-05-A described;
  build's retry-instruction builder had a true (not materially-different) duplicate of the same field
- **Fix:** Applied the identical `renderRelatedWorkflowHandoffSection` mechanism to colonize, plan,
  seal, and swarm; removed `codexWorkerDispatchesForRecovery`'s `HandoffSection` outright
- **Verification:** Throwaway probe (deleted after use) proved each was real before fixing; the
  permanent breadth test (`TestNineCommandsDeliverHandoffExactlyOnce`) now locks all of them, and
  fails (pre-fix, captured via `git stash`) exactly as described above
- **Files modified:** `cmd/codex_colonize.go`, `cmd/codex_plan.go`, `cmd/seal_final_review.go`,
  `cmd/swarm_cmd.go`, `cmd/codex_build_finalize.go`
- **Committed in:** `7c88c97f`

---

**Total deviations:** 2 (1 auto-fixed formatting issue, 1 documented scope decision required by this
plan's own proof_requirement and CLAUDE.md's Definition of Done)
**Impact on plan:** The formatting fix was mechanical and zero-risk. The scope decision expanded the
fix from 2 to 6 dispatch functions plus 1 inert-duplicate removal, but used the SAME mechanism proven
correct for the 2 named functions, is fully covered by the same breadth-lock test, and is the reading
this plan's own proof_requirement asked for ("locked across ALL construction sites"). No architectural
decisions were introduced -- every fix follows the identical "distinct heading for materially different
content, deletion for true duplicates" pattern established for the two named functions.

## Issues Encountered

**Confirming the "materially different content" claim required empirical proof, not just reading.**
Reading `renderWorkerHandoffSection`'s filter logic alone left open whether the capsule's hardcoded
`workflow="build"` filter and each dedicated field's own-workflow filter could ever BOTH have content
simultaneously (if they never could, in practice, the field would functionally never duplicate
anything). A throwaway probe (`cmd/zzz_probe_190_06_test.go`, deleted before the final commit) seeded
both a build-tagged and a command-tagged handoff record and called each real dispatch function
directly, confirming 2 headings for colonize/plan/seal/swarm/continue_review before any fix existed.
This is the same discipline 190-05 used for its own pheromone probe, applied here to settle a factual
question (do these channels' contents overlap in practice) that reading the filter predicates alone
could not answer with confidence.

**Fail-then-pass evidence for the PERMANENT tests (not just the throwaway probe) required a source
revert.** To quote the exact failure text the permanent tests (not the deleted probe) produce against
pre-fix code, the 7 fixed source files were `git stash push --keep-index`'d (the new test file was
left in place, since it has no compile dependency on the new `renderRelatedWorkflowHandoffSection`
function -- it only calls the public dispatch functions and inspects their output), the tests were run
and failed exactly as quoted in the Accomplishments section above, then `git stash pop` restored the
fix and the same tests were re-run to confirm all pass. `go build ./...` was re-verified clean both
before and after the stash pop.

## Next Phase Readiness

- D-190-05-A is closed. Phase 190's own stated goal ("No context section delivered twice") now holds
  for prior-worker handoffs on every native/direct dispatch path this repo spawns a Codex/Claude/
  OpenCode worker from -- the same breadth 190-05 already established for pheromone signals.
  `TestNativeDispatchHandoffStaysExactlyOnceViaCapsule` (build), `TestNineCommandsDeliverHandoffExactlyOnce`
  (continue x2, plan, colonize, seal, swarm, quick, oracle), and `TestEightCommandsDeliverPheromoneExactlyOnce`
  /`TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule` (pheromones, unmodified) together cover
  every construction site this plan's audit found.
- No further deferred items remain open in `deferred-items.md` for the "context section delivered
  twice" defect class as of this plan (D-190-01-A, D-190-03-A, and D-190-05-A are all RESOLVED).
  `D-190-R-A` (cross-domain hive wisdom exclusion, raised by 190-REVIEW) remains open but is a
  different, product-decision-shaped question, not a duplication defect.
- If a future phase adds another command that dispatches a worker via
  `codex.WorkerDispatch`/`codex.WorkerConfig`, the pattern to follow is now explicit in
  `renderWorkerHandoffSection`'s and `renderRelatedWorkflowHandoffSection`'s doc comments: use the
  plain function if the caller's capsule does not already render handoffs (or renders the SAME
  workflow), use the "Related" variant if the caller's capsule renders a DIFFERENT workflow's
  handoffs that must not collide under the same heading.

## Self-Check: PASSED

- FOUND: `cmd/build_handoff_190_06_test.go`
- FOUND: `.planning/phases/190-lean-non-duplicated-delivery/deferred-items.md` (D-190-05-A marked RESOLVED)
- FOUND commit `7c88c97f` (fix: give continue's handoff relay its own heading, not a deleted field)

---
*Phase: 190-lean-non-duplicated-delivery*
*Completed: 2026-08-21*
