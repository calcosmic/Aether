---
phase: 190-lean-non-duplicated-delivery
plan: 05
subsystem: cli/worker-dispatch
tags: [go, cli, worker-dispatch, pheromones, colony-prime, codex]

# Dependency graph
requires:
  - phase: 190-lean-non-duplicated-delivery
    provides: "190-03's ownership decision (capsule owns pheromones/handoffs for the wrapper flow) and its deferred finding D-190-03-A, which named the native/direct dispatch path's separate pheromone double-channel this plan closes"
provides:
  - "D-190-03-A closed: on every native/direct dispatch path (build, continue review, continue watcher, plan, colonize, seal, swarm) an active pheromone signal reaches the assembled worker prompt exactly once, via the shared capsule (resolveCodexWorkerContext()) alone -- PheromoneSection is no longer independently populated at these 7 call sites"
  - "quick and oracle deliberately left unchanged: their capsules (renderQuickContextCapsule, renderOracleContextCapsule) never render pheromones, so PheromoneSection remains their sole delivery channel"
  - "resolvePheromoneSection()'s doc comment now states the calling contract explicitly: call it only when the caller's capsule does not already embed pheromones"
  - "codexWorkerDispatchesForRecovery (build's retry-instruction builder) fixed for consistency, even though its WorkerDispatch values are not passed through AssemblePrompt today"
  - "TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule + TestEightCommandsDeliverPheromoneExactlyOnce: fail-then-pass proof plus a 9-case breadth test (7 duplicate-fixed call sites + quick + oracle) locking count==1 for every command that assembles a worker prompt"
  - "D-190-05-A: a newly-discovered, separate handoff double-channel on continue's native review/watcher dispatch paths (HandoffSection, not pheromones), logged and deferred, not fixed here"
affects: [any future phase touching codex.WorkerDispatch/WorkerConfig's ContextCapsule/PheromoneSection/HandoffSection fields, or the continue native review/watcher dispatch functions]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-caller audit before a shared-struct fix: for a field populated at N call sites, check what each call site's OTHER field (capsule) actually renders before removing anything -- a caller whose capsule doesn't embed the content would drop to zero, which is worse than the duplicate being fixed"
    - "A spy WorkerInvoker (captures WorkerConfig, delegates to FakeInvoker for the result) proves the REAL wiring end-to-end, not a parallel computation that merely happens to agree with the fix"
    - "Test helpers that mutate global state (store) document their restoration contract in their own doc comment (seedHandoffColonyForBriefTests); every caller must pair the helper with saveGlobals/saveGlobalsCmd -- missing that pairing is invisible until a LATER, unrelated test in the same package trips over the leaked global"

key-files:
  created:
    - cmd/build_pheromone_190_05_test.go
    - .planning/phases/190-lean-non-duplicated-delivery/190-05-SUMMARY.md
  modified:
    - cmd/codex_build.go
    - cmd/codex_build_finalize.go
    - cmd/codex_colonize.go
    - cmd/codex_continue.go
    - cmd/codex_plan.go
    - cmd/seal_final_review.go
    - cmd/swarm_cmd.go
    - .planning/phases/190-lean-non-duplicated-delivery/deferred-items.md

key-decisions:
  - "FIX SHAPE: Option A (fix at the callers), not Option B (dedup in the assembler). The per-caller audit found a CLEAN, static split -- 7 call sites always use resolveCodexWorkerContext() (which unconditionally embeds pheromones) and 2 (quick, oracle) always use their own custom, pheromone-free capsules -- so the decision is determinable per call site at compile time, not a runtime content-sniffing problem. Content-sniffing in the assembler would have papered over that clean split with unnecessary complexity."
  - "SEVEN call sites (across six commands: build native, continue review, continue watcher, plan, colonize, seal, swarm) stop populating codex.WorkerDispatch/WorkerConfig's PheromoneSection field entirely -- the capsule is now their sole steering channel for pheromones, mirroring 190-03's capsule-owns-pheromones decision for the wrapper flow."
  - "TWO call sites (quick, oracle) are UNCHANGED -- their capsules never render pheromones, so PheromoneSection remains the only channel. Confirmed empirically (not assumed): grepped both capsule renderers for 'Pheromone', found none, and the breadth test's fail-then-pass proof for these two cases showed 0 (not 2) when the heading assertion was first written incorrectly, which is the exact zero-trap this plan's own instructions warned against."
  - "codexWorkerDispatchesForRecovery (build retry-instruction builder) also fixed, though its WorkerDispatch values are never passed through AssemblePrompt today (confirmed by tracing its only caller, buildExternalBuildRecoveryInstructions, which only uses them for in-memory same-caste peer lookup) -- fixed to prevent a latent duplication trap for whenever a future change wires this into a live invocation."
  - "D-190-05-A (continue's native review/watcher dispatch paths independently double-deliver HandoffSection via the same capsule-plus-dedicated-field shape) was discovered during the per-caller audit, confirmed real by an empirical probe, and deliberately NOT fixed -- out of this plan's pheromone-scoped objective. Logged with the same rigor as D-190-03-A itself."

requirements-completed: []

# Metrics
duration: ~150min
completed: 2026-08-21
---

# Phase 190 Plan 05: Lean, Non-Duplicated Delivery (native dispatch path) Summary

**Closed D-190-03-A: seven native/direct worker-dispatch call sites (build, continue review, continue watcher, plan, colonize, seal, swarm) stopped independently resolving a second copy of active pheromone signals once the shared colony-prime capsule already carries them -- proven with a spy-invoker fail-then-pass test and a 9-case breadth lock covering every command that assembles a worker prompt, including the two (quick, oracle) that correctly keep the field.**

## Performance

- **Duration:** ~150 min
- **Completed:** 2026-08-21
- **Tasks:** 1 (single cohesive fix, per this gap-closure plan's own framing, matching 190-03's precedent)
- **Files modified:** 9 (7 Go source, 1 new Go test file, 1 planning doc)

## Per-Caller Audit Table

Required by this plan's own blast-radius discipline: for each of the ~8 commands that construct a
`codex.WorkerDispatch`/`codex.WorkerConfig` destined for `AssemblePrompt`/`AssembleHostedPrompt`,
whether its `ContextCapsule` already embeds pheromones, and what this plan's fix does there.

| # | Command | Construction site | `ContextCapsule` source | Embeds `## Pheromone Signals`? | Fix |
|---|---------|-------------------|--------------------------|--------------------------------|-----|
| 1 | build (native) | `executeCodexBuildDispatches`, `cmd/codex_build.go` | `resolveCodexWorkerContext()` | YES, unconditionally when active | `PheromoneSection` removed |
| 2 | continue (review) | `plannedContinueReviewDispatches`, `cmd/codex_continue.go` | `resolveCodexWorkerContext()` | YES | `PheromoneSection` removed |
| 3 | continue (watcher) | `plannedContinueWatcherDispatch`, `cmd/codex_continue.go` | `resolveCodexWorkerContext()` | YES | `PheromoneSection` removed |
| 4 | plan | `dispatchRealPlanningWorkersWithIterationContext`, `cmd/codex_plan.go` | `resolveCodexWorkerContext()` | YES | `PheromoneSection` removed |
| 5 | colonize | `dispatchRealSurveyorsWithTimeout`, `cmd/codex_colonize.go` | `resolveCodexWorkerContext()` | YES | `PheromoneSection` removed |
| 6 | seal | `plannedSealFinalReviewDispatches`, `cmd/seal_final_review.go` | `resolveCodexWorkerContext()` | YES | `PheromoneSection` removed |
| 7 | swarm | `invokeSwarmWorker`, `cmd/swarm_cmd.go` | `resolveCodexWorkerContext()` | YES | `PheromoneSection` removed |
| 8 | quick | `runQuickScout`, `cmd/command_truth.go` | `renderQuickContextCapsule()` (custom, lightweight) | **NO** | Unchanged -- sole channel |
| 9 | oracle | `buildOracleWorkerConfig`, `cmd/oracle_loop.go` | `renderOracleContextCapsule()` (custom, lightweight) | **NO** | Unchanged -- sole channel |

(9 call sites across 8 commands -- continue has two: review and watcher.)

**Also audited and found inert (not part of the 9 above, but same struct):**
- `codexWorkerDispatchesForRecovery` (`cmd/codex_build_finalize.go`, build's retry-instruction
  builder): set both fields, but its `WorkerDispatch` values are traced to a single caller,
  `buildExternalBuildRecoveryInstructions`, which only reads them in-memory for same-caste peer
  lookup -- never passed through `AssemblePrompt`/`AssembleHostedPrompt`. Fixed anyway for
  consistency (see Decisions).
- `buildToWorkerDispatches` (`cmd/codex_dispatch_contract.go`): confirmed to have zero production
  callers (only a test calls it) and never sets `ContextCapsule`/`PheromoneSection` at all --
  out of scope, no fix needed.
- `persistDispatchWorkerHandoff`'s two `codex.WorkerDispatch{}` call sites
  (`cmd/codex_build_finalize.go`, `cmd/codex_continue_finalize.go`): used purely as a data carrier
  for handoff persistence, never set `ContextCapsule`/`PheromoneSection`, never reach
  `AssemblePrompt` -- out of scope.
- `cmd/codex_build_worktree.go`'s two `codex.WorkerConfig{}` construction sites: confirmed to be
  pure field-forwarding from a `codex.WorkerDispatch` parameter that already originates from
  `executeCodexBuildDispatches` (traced via `dispatchCodexBuildWorkersWithReconciliation` /
  `dispatchCodexBuildWorkersInRepo`) -- fixing call site #1 above automatically fixes these too, no
  separate edit needed.
- `codex_continue_plan.go`'s `codexContinuePlanManifest.PheromoneSection` field: a manifest-level,
  resolved-once field for continue's **plan-only wrapper** flow (the CONTINUE analog of build's
  190-03-fixed wrapper manifest), explicitly documented as read once by the wrapper and never
  duplicated per-dispatch (`codexContinueExternalDispatch` has no `PheromoneSection` field at all).
  Structurally unrelated to the native/direct path this plan targets -- out of scope, and not a
  duplicate.

## Accomplishments

- **Root cause confirmed exactly as D-190-03-A described, then generalized.** `executeCodexBuildDispatches`
  (build's native path) computed `capsule := resolveCodexWorkerContext()` and a separate
  `pheromoneSection := resolvePheromoneSection()`, setting both on every `codex.WorkerDispatch`.
  `AssemblePrompt`/`AssembleHostedPrompt` join them as two independent, both-included parts with no
  dedup. Auditing the other ~7 named commands found the IDENTICAL pattern independently reproduced
  at 6 more call sites (continue review, continue watcher, plan, colonize, seal, swarm) -- not just
  build.
- **The "zero" trap was real, not hypothetical.** quick and oracle build their own lightweight,
  purpose-specific capsules (`renderQuickContextCapsule`, `renderOracleContextCapsule`) that never
  render "## Pheromone Signals" at all -- confirmed by grep, then confirmed a second time when the
  breadth test's first draft (assuming every command uses the same heading) failed these two cases
  with "0 headings, want 1" before the assertion was corrected to accept
  `resolvePheromoneSection()`'s own "### Active Pheromone Signals" heading as an equally valid
  single delivery. `PheromoneSection` at these two call sites was correctly left untouched.
- **Fail-then-pass proven with the real wiring, not a parallel computation.** A spy `WorkerInvoker`
  (`pheromoneCaptureInvoker190_05`) captures the actual `codex.WorkerConfig` each function hands to
  the invoker, then assembles the prompt via the same production `codex.AssembleHostedPrompt` every
  hosted platform dispatcher calls. Before the fix: `"assembled prompt has 2 \"Pheromone Signals\"
  headings, want exactly 1"` on all 7 duplicate-fixed cases. After: exactly 1 on all 9 cases
  (including quick/oracle, which were never broken).
- **A real, separate bug in this plan's OWN test code was found and fixed before it shipped
  (Rule 1).** The breadth test's fixture helper (`seedHandoffColonyForBriefTests`, an existing
  190-01 helper this plan reused) sets the package-global `store` and documents that its callers
  must restore it via `saveGlobals`/`saveGlobalsCmd` -- the parent breadth test initially did not,
  leaking a `store` pointer into a deleted temp directory that broke an unrelated, later-running
  test (`TestBuildDispatchStartsHeartbeatMonitor`, which started failing with `"storage: open lock
  file ... no such file or directory"`) whenever the full `cmd` suite ran. Confirmed by bisection
  (full suite: baseline clean, source-fix-alone clean, test-file-added reproducibly broken twice in
  a row) before concluding it was this plan's own test hygiene, not the source fix. Fixed by adding
  `saveGlobalsCmd(t)` to the breadth test; reconfirmed clean on two subsequent full-suite runs.
- **A second, separate duplication mechanism was discovered and deliberately NOT fixed.** Continue's
  native review and watcher dispatch functions independently set `HandoffSection` from
  `renderWorkerHandoffSection("continue", ...)` on top of a capsule that already renders "##
  Previous Worker Handoffs" -- the same shape as D-190-03-A, but for handoffs, and only reproducible
  with a `Workflow: "continue"`-tagged stored handoff (the base test fixture's `Workflow: "build"`
  handoff produces a false negative, which is why this went unnoticed until now). Proven real with a
  throwaway probe (deleted after use), logged as `D-190-05-A` with the same rigor `D-190-03-A` itself
  demonstrated, and left for a future phase since this plan's objective is pheromone-specific.

## Task Commits

This gap-closure plan was executed and committed as one cohesive fix:

1. **Fix: pheromone signals get one home on every native/direct dispatch path** - see commit hash below (fix)

## Files Created/Modified

- `cmd/codex_build.go` - `executeCodexBuildDispatches` no longer sets `PheromoneSection`;
  `resolvePheromoneSection()`'s doc comment states the full calling contract (which callers must NOT
  call it, which two still must)
- `cmd/codex_build_finalize.go` - `codexWorkerDispatchesForRecovery` (build retry-instruction
  builder) fixed for consistency, confirmed inert w.r.t. `AssemblePrompt` today
- `cmd/codex_colonize.go` - `dispatchRealSurveyorsWithTimeout` no longer sets `PheromoneSection`
- `cmd/codex_continue.go` - `plannedContinueReviewDispatches` and `plannedContinueWatcherDispatch`
  no longer set `PheromoneSection`
- `cmd/codex_plan.go` - `dispatchRealPlanningWorkersWithIterationContext` no longer sets
  `PheromoneSection`
- `cmd/seal_final_review.go` - `plannedSealFinalReviewDispatches` no longer sets `PheromoneSection`
- `cmd/swarm_cmd.go` - `invokeSwarmWorker` no longer sets `PheromoneSection`
- `cmd/build_pheromone_190_05_test.go` (new) - `TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule`
  (primary fail-then-pass proof for build's native path, D-190-03-A's exact target) and
  `TestEightCommandsDeliverPheromoneExactlyOnce` (9-case breadth lock, one sub-test per command,
  asserting count==1 for an active signal on every one)
- `.planning/phases/190-lean-non-duplicated-delivery/deferred-items.md` - D-190-03-A marked RESOLVED
  with a pointer to this summary; new `D-190-05-A` entry logs the continue-handoff finding

## Decisions Made

See `key-decisions` in frontmatter. The most consequential: **Option A (fix at the callers) over
Option B (dedup in the assembler)**, because the per-caller audit showed a clean, statically
determinable split rather than a genuinely mixed/irreducible situation --

1. **Seven callers share one capsule renderer, unconditionally.** `resolveCodexWorkerContext()` is
   used identically (same function, same behavior) by build/continue×2/plan/colonize/seal/swarm, and
   it renders `## Pheromone Signals` unconditionally whenever a signal is active
   (`cmd/colony_prime_context.go:571`) -- this is not caller-specific behavior that could change at
   runtime, it is a fixed property of which function each caller calls.
2. **Two callers share a different property, also unconditionally.** quick and oracle each build
   their own custom, purpose-built capsule that never renders pheromones -- also a fixed,
   caller-identity property, not runtime content.
3. **Given both groups are determined by which function is called, not by what the data happens to
   contain at runtime, the decision belongs at the call site, not inside the shared assembler.**
   Content-sniffing in `AssemblePrompt`/`AssembleHostedPrompt` (Option B) would have added runtime
   string-matching brittleness to solve a problem that a one-line removal already solves cleanly at
   each of the 7 call sites -- and would have obscured, rather than documented, which callers still
   need the field.
4. **Mechanism matches 190-03's own precedent.** 190-03 established "the capsule owns steering
   content when a capsule accompanies the dispatch" for the wrapper flow via a boolean flag on a
   shared composer. This plan applies the identical ownership principle to the native/direct path,
   just without needing a flag -- there is no second valid mode for these 7 callers, so removing the
   field outright (rather than gating it) is the simpler, equally-correct mechanism.

**A genuinely separate finding was deliberately NOT folded into the fix.** `D-190-05-A` (continue's
native review/watcher paths double-delivering `HandoffSection`) is structurally identical in shape
to D-190-03-A but is about handoffs, not pheromones, and this plan's objective is pheromone-scoped.
Logged for a future phase with the exact fix and proof pattern to reuse.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug in this plan's own test code] `TestEightCommandsDeliverPheromoneExactlyOnce`
leaked the global `store` variable, breaking an unrelated later test in full-suite runs**
- **Found during:** Final verification (`go test ./cmd/ -count=1`), which failed with
  `TestBuildDispatchStartsHeartbeatMonitor` erroring `"storage: open lock file ... no such file or
  directory"` -- a test this plan never touched, in a file this plan never modified.
- **Issue:** The breadth test's per-command fixture setup reuses `seedHandoffColonyForBriefTests`
  (an existing 190-01 helper), which sets the package-global `store` and documents that callers must
  restore it via `saveGlobals`/`saveGlobalsCmd` -- its own cleanup only deletes the underlying temp
  directory, not the global pointer. The breadth test's parent function never paired the helper with
  that restoration, so after its last sub-test finished, `store` was left pointing at a deleted
  directory for the remainder of the `cmd` package's test run.
- **Fix:** Added `saveGlobalsCmd(t)` to `TestEightCommandsDeliverPheromoneExactlyOnce`, restoring
  `store` once all 9 sub-tests complete.
- **Verification:** Bisected before concluding this was the cause, not the source fix: true baseline
  (no plan changes) ran clean; source fix alone (7 files, no new test file) ran clean; test file
  added reproduced the failure identically on two consecutive full-suite runs; after the
  `saveGlobalsCmd` fix, two more consecutive full-suite runs (`go test ./cmd/ -count=1`) both passed
  clean.
- **Files modified:** `cmd/build_pheromone_190_05_test.go`
- **Committed in:** (same commit as the fix -- caught before this file was ever committed)

**2. [Rule 1 - Consistency, prevents a latent trap] `codexWorkerDispatchesForRecovery` also fixed**
- **Found during:** Per-caller audit, tracing every `codex.WorkerDispatch` construction site.
- **Issue:** Build's retry-instruction builder set both `ContextCapsule` and `PheromoneSection` from
  the same unconditionally-pheromone-embedding capsule, the identical pattern D-190-03-A named --
  but its `WorkerDispatch` values are never passed through `AssemblePrompt` today (traced to a single
  caller, `buildExternalBuildRecoveryInstructions`, which only uses them for in-memory same-caste
  peer lookup).
- **Fix:** Removed the redundant `PheromoneSection` computation and assignment, matching the other
  7 call sites, so a future change wiring this into a live invocation does not silently reintroduce
  the bug this plan just closed elsewhere.
- **Files modified:** `cmd/codex_build_finalize.go`

**3. [Rule 2 discovery, NOT auto-fixed -- logged as deferred] Continue's native review/watcher
dispatch paths independently double-deliver prior-worker handoffs**
- **Found during:** Per-caller audit (checking `HandoffSection` alongside `PheromoneSection` at
  every call site, since D-190-03-A's own text asserted handoffs did NOT duplicate on build's native
  path -- true for build, not verified for continue).
- **Issue:** `plannedContinueReviewDispatches` and `plannedContinueWatcherDispatch` both set
  `HandoffSection: renderWorkerHandoffSection("continue", ...)` on top of a capsule that already
  renders `## Previous Worker Handoffs` -- reproducible only with a `Workflow: "continue"`-tagged
  stored handoff.
- **Why not fixed:** Out of this plan's pheromone-scoped objective ("an active pheromone signal
  reaches each worker TWICE"). Fixing handoffs on a different command is a separate, real change
  this plan's own scope does not name.
- **Verification that this is real, not hypothetical:** A throwaway probe (deleted after use) seeded
  a `Workflow: "continue"` handoff and called `plannedContinueReviewDispatches` directly -- the
  assembled prompt carried `## Previous Worker Handoffs` twice.
- **Logged in:** `.planning/phases/190-lean-non-duplicated-delivery/deferred-items.md` (`D-190-05-A`)

---

**Total deviations:** 3 (1 auto-fixed bug in this plan's own test code, 1 auto-fixed consistency fix
on an inert-today call site, 1 discovered-and-deferred separate architectural finding)
**Impact on plan:** The test-pollution bug was necessary to fix -- without it, this plan's own
final verification would have been reporting a false regression against unrelated code. The
consistency fix was low-risk and closes a latent trap. The deferred finding does not block this
plan's stated success criteria (which are pheromone-specific) but is flagged with the same rigor
D-190-03-A itself demonstrated, since it is a real, already-shipping duplicate on a different, live
path.

## Issues Encountered

**Full-suite verification took materially longer than the fix itself.** `go test ./cmd/ -count=1`
runs in ~330s (the package is one of the largest in the repo). Diagnosing the test-pollution
deviation above required five full-suite runs (baseline, two reproducing the failure, two confirming
the fix) plus several isolated/targeted runs -- the bulk of this plan's wall-clock time. No shortcuts
were taken on this: per this project's own Definition of Done, a test failure discovered during
"final checks" is investigated to a confirmed root cause, not waved off as unrelated without proof.

**A worktree-branch staleness issue at startup, unrelated to this plan's own work.** The assigned
worktree's HEAD was on a stale commit (`6577f51c`, far older than Phase 190) rather than the expected
base (`1430acf4`). The mandatory `<worktree_branch_check>` step's own `git reset --hard` to the
expected base commit resolved this automatically before any plan work began.

## Next Phase Readiness

- `D-190-05-A` (continue's native review/watcher paths' separate handoff double-channel) is ready
  for a future phase to pick up; it names the exact functions, the fixture gotcha (workflow-tagged
  handoff required to reproduce), and the throwaway-probe proof pattern to reuse.
- The phase's own broader goal sentence ("No context section delivered twice") is now true for
  pheromone signals on every command this repo dispatches a Codex/Claude/OpenCode worker from --
  the one gap 190-VERIFICATION.md's own gaps_found score (6/7) named is closed.

---
*Phase: 190-lean-non-duplicated-delivery*
*Completed: 2026-08-21*
