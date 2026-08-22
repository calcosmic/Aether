---
phase: 188-one-truth-for-failures-and-advances
plan: 02
subsystem: runtime-state
tags: [go, continue, continue-finalize, atomic-write, concurrency, state-machine]

# Dependency graph
requires:
  - phase: 187-crash-safe-worktrees-and-ecosystem-neutrality
    provides: fail-closed-on-uncertainty precedent (worktreeDestructionSafety) this plan's supersession check follows
provides:
  - "advancePhase() -- the one shared, atomic phase-advance core, exported from cmd/advance_phase.go for both continue paths to call"
  - "The default aether continue path (runCodexContinue) wired to advancePhase(), zero behavioral change"
  - "The aether continue-finalize path (advanceExternalContinue) wired to advancePhase(), closing a live state-clobber bug and adding the supersession check it never had"
affects: [188-04-the-ratchet, continue, continue-finalize, state-mutate]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared atomic-commit core with scalar-only params (no full struct to clobber with) as a structural guard against reintroducing a stale-overwrite bug"
    - "Supersession refusal (errRuntimeStateSuperseded) surfaced as a nil-error, blocked/superseded map result at the outer function boundary, not as a Go error -- callers check result[\"superseded\"] before running post-advance side effects"

key-files:
  created:
    - cmd/advance_phase.go
    - cmd/advance_phase_test.go
  modified:
    - cmd/codex_continue.go
    - cmd/codex_continue_finalize.go
    - cmd/codex_continue_finalize_test.go

key-decisions:
  - "advancePhaseParams deliberately carries no colony.ColonyState or colony.Phase field -- only scalars (PhaseID, ExpectedBuildStartedAt, AllowedStates, Source, Now) -- so the clobber bug being fixed (assigning a stale full-state value over what UpdateJSONAtomically just loaded) is structurally impossible to reintroduce, not just avoided by convention"
  - "advanceExternalContinue's supersession branch returns a nil Go error (not errRuntimeStateSuperseded) so its caller, runCodexContinueFinalize, treats it as a completed-but-blocked result -- this required adding a result[\"superseded\"] check at the call site (not explicitly spelled out in the plan's action text) so the caller returns immediately instead of falling through to phase-end consolidation and the phase-commit git commit on a phase that never actually advanced"
  - "Used the colony's Paused flag (not BuildStartedAt) as the supersession lever in the new regression test, since continue-finalize's own earlier validateExternalContinueState check already rejects a stale BuildStartedAt/CurrentPhase/State mismatch before advanceExternalContinue is ever reached -- Paused is checked nowhere in that earlier validation, so it is the one field that reaches advancePhase's check untouched, exactly reproducing 'a concurrent process changed something the caller's original read never accounted for'"

patterns-established:
  - "Pattern: extract-not-invent for atomic state cores -- advancePhase is a direct, faithful lift of runCodexContinue's already-correct block, not a rewrite from a spec"

requirements-completed: []

# Metrics
duration: ~60min (includes three full `cmd` package test runs plus one full-repo test run for verification)
completed: 2026-08-19
---

# Phase 188 Plan 02: One Shared Atomic Phase-Advance Core Summary

**Extracted `advancePhase()` from `aether continue`'s already-correct atomic-commit block and rewired `aether continue-finalize` onto it, closing a live bug where a concurrent state change (e.g., an operator pausing the colony mid-flight) was silently discarded and the phase force-advanced anyway.**

## Performance

- **Duration:** ~60 min
- **Tasks:** 3/3 completed
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `cmd/advance_phase.go` now holds the one atomic core (`advancePhase`, `advancePhaseParams`, `advancePhaseResult`) both `aether continue` and `aether continue-finalize` call to commit a phase advancement -- previously each had its own ~60-line copy.
- `aether continue`'s path (`runCodexContinue`) was rewired with zero behavioral change (it was the correct copy `advancePhase` was extracted from).
- `aether continue-finalize`'s path (`advanceExternalContinue`) had two real bugs closed: it clobbered the freshly-loaded state with a stale captured variable (`updated = state`), and it had no supersession check at all (`validateRuntimeStateStillCurrent` was never called on this path before). Both are now impossible via the shared core.
- The clobber bug was reproduced by a test **before** it was fixed, watched fail, then watched pass after the fix -- see "Definition of Done evidence" below.
- The supersession check's necessity was independently proven by temporarily disabling it and re-observing the new test fail, per the plan's own acceptance criteria.

## Task Commits

Each task was committed atomically:

1. **Task 1: Define advancePhase() -- the one atomic phase-advance core** - `8ac4d793` (feat)
2. **Task 2: Wire the default continue path to advancePhase()** - `5d0be8a4` (refactor)
3. **Task 3: Wire the finalize continue path to advancePhase(), removing the clobber bug and the trailing non-atomic write** - `e361f951` (fix)

**Plan metadata:** committed separately after this SUMMARY (see final commit).

## Files Created/Modified

- `cmd/advance_phase.go` - New. `advancePhase()`, `advancePhaseParams`, `advancePhaseResult` -- the shared atomic core.
- `cmd/advance_phase_test.go` - New. Happy-path (non-final + final), two supersession-refuses-and-writes-nothing cases, and a Source-literal-in-events test.
- `cmd/codex_continue.go` - `runCodexContinue`'s inline atomic-commit block replaced with a call to `advancePhase`.
- `cmd/codex_continue_finalize.go` - `advanceExternalContinue`'s buggy inline block replaced with a call to `advancePhase`; trailing non-atomic `SaveJSON` replaced with `appendRuntimeStateEventsIfCurrent`; `runCodexContinueFinalize`'s call site now checks `result["superseded"]` before running phase-end consolidation/commit.
- `cmd/codex_continue_finalize_test.go` - New test `TestContinueFinalizeRefusesToAdvanceOnSupersededState`.

## Decisions Made

- See `key-decisions` in frontmatter above (advancePhaseParams scalar-only design, the call-site `superseded` check, and the `Paused`-flag test technique).

## Definition of Done evidence (state-corruption bug reproduced before the fix)

Per this repo's Definition of Done and the orchestrator's explicit instruction, the bug was reproduced by a test and watched fail **before** any fix was applied.

**Test:** `TestContinueFinalizeRefusesToAdvanceOnSupersededState` (`cmd/codex_continue_finalize_test.go`), written and run against the still-buggy `advanceExternalContinue` (before Task 3's rewrite). It seeds a normal in-progress phase, runs `aether continue --plan-only` to get a manifest, then writes a competing state change directly via the store (`Paused: true`, simulating a concurrent operator pause) between the plan-only read and the finalize commit, then calls `runCodexContinueFinalize`.

**Observed failure against the pre-fix code:**
```
codex_continue_finalize_test.go:190: result[blocked] = <nil>, want true (result: map[string]interface {}{"advanced":true, ... "current_phase":2, ... "next":"aether build 2", ... "state":"READY", ...})
codex_continue_finalize_test.go:193: result[superseded] = <nil>, want true (...)
codex_continue_finalize_test.go:196: result[advanced] = true, want false
codex_continue_finalize_test.go:207: phase 1 was advanced to completed despite the superseded (paused) state
--- FAIL: TestContinueFinalizeRefusesToAdvanceOnSupersededState (1.46s)
```
The phase was silently advanced (`current_phase` moved from 1 to 2, `next` became `"aether build 2"`) even though the colony had been concurrently paused -- the pause write was discarded exactly as 188-CONTEXT.md's D-05 describes.

**After the Task 3 fix** (delegating to `advancePhase`, adding the call-site `superseded` check): the same test passes -- `result[blocked]=true`, `result[superseded]=true`, `result[advanced]=false`, phase 1 still `in_progress` on disk, `Paused=true` preserved.

**Supersession-check regression proof** (per the plan's own Task 3 acceptance criteria): `validateRuntimeStateStillCurrent`'s call inside `cmd/advance_phase.go` was temporarily commented out, the test re-run, and it failed again with the identical clobber symptoms (`result[advanced]=true`, phase silently completed). The check was then restored and both `TestAdvancePhase*` and `TestContinueFinalizeRefusesToAdvanceOnSupersededState` re-confirmed passing.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Call site did not short-circuit on a superseded/blocked result from `advanceExternalContinue`**
- **Found during:** Task 3, while wiring `advanceExternalContinue` to `advancePhase`.
- **Issue:** The plan's action text specifies that `advanceExternalContinue` returns a nil Go error with a blocked/superseded result map on supersession (mirroring `continueSupersededResult`'s shape), "so the caller (`runCodexContinueFinalize`) treats this as a completed-but-blocked result rather than a hard failure." However, the existing call site in `runCodexContinueFinalize` only branched on `err != nil` and otherwise fell straight through to `runPhaseEndConsolidation`, `attachPhaseCommitResult` (a real git commit), and a ceremony emission -- all of which assume the phase genuinely advanced. Implemented literally, a nil-error supersession result would have incorrectly triggered phase-end consolidation and a git commit for a phase that never advanced.
- **Fix:** Added a `result["superseded"]` check immediately after the `err != nil` branch that returns immediately (mirroring the existing gate-blocked and review-blocked early-return branches earlier in the same function), before the consolidation/commit logic runs.
- **Files modified:** `cmd/codex_continue_finalize.go`
- **Verification:** `TestContinueFinalizeRefusesToAdvanceOnSupersededState` asserts `err == nil`, `result["blocked"]==true`, `result["superseded"]==true`, and that the on-disk phase status is unchanged -- this assertion would fail without the fix (consolidation/commit would still attempt to run against a phase that was never marked completed).
- **Committed in:** `e361f951` (Task 3 commit)

**2. [Rule 1 - Bug] New test file's alphabetical position exposed a pre-existing global-state leak in an unrelated test**
- **Found during:** Task 1, running the full `cmd` package test suite (`go test ./cmd/... -count=1`) as required by `<parallel_execution_note>`.
- **Issue:** `TestBuildDispatchStartsHeartbeatMonitor` (`cmd/codex_build_test.go`, out of this plan's `files_modified`) never initializes its own package-level `store` -- it implicitly relies on whatever `store` a preceding test in the same binary left behind. `setupBuildFlowTest` (the shared test helper my new tests use) sets `store` to a `*storage.Store` rooted at its own `t.TempDir()`, but does not restore the previous value on cleanup (only `stdout`/`stderr` are restored). Because `cmd/advance_phase_test.go` sorts alphabetically before `cmd/codex_build_test.go`, my last test became the one immediately preceding the heartbeat test in the compiled binary's execution order, so the heartbeat test inherited a `store` pointer into a directory `t.TempDir()` had already deleted, and failed with a lock-file-open error.
- **Fix:** Added `saveGlobals(t)` (the existing, repo-established helper for snapshotting and restoring package globals including `store`) as the first line of `advancePhaseFixture` in my own new test file -- the same pairing convention already used by several existing tests (e.g. `TestContinueFinalizeAddsOrchestratorBoundaryGuidance`, `TestContinueStaleStateDoesNotOverwritePausedState`). This restores `store` to its pre-test value once each of my tests ends, instead of leaving it dangling. Did not touch `cmd/codex_build_test.go` (out of scope; the root fragility there -- not establishing its own store -- remains, but is now not triggered by this plan's tests).
- **Files modified:** `cmd/advance_phase_test.go`
- **Verification:** Reproduced the failure in isolation (`go test ./cmd/ -run 'TestAdvancePhase|TestBuildDispatchStartsHeartbeatMonitor'`), confirmed the fix resolves it in the same isolated run, then re-ran the full `cmd` package suite twice (once still failing, once clean) and the full-repo suite once clean.
- **Committed in:** `8ac4d793` (Task 1 commit; the fix was applied before this file was ever committed, so it does not appear as a separate follow-up commit)

---

**Total deviations:** 2 auto-fixed (2 Rule 1 bug fixes)
**Impact on plan:** Both fixes were necessary for correctness -- the first prevents a supersession refusal from still triggering downstream side effects meant only for genuine advances; the second prevents this plan's own new test file from destabilizing an unrelated, already-fragile test via file-ordering. No scope creep: neither touched `cmd/codex_build_test.go` or any file outside this plan's `files_modified` list.

## Issues Encountered

None beyond the two auto-fixed items above.

## Verification Record

```
go build ./cmd/           -> exit 0
go vet ./cmd/             -> exit 0
go build ./...            -> exit 0
go vet ./...               -> exit 0
go test ./cmd/ -run TestAdvancePhase -v -count=1                                    -> PASS (4 test functions, 6 subtests)
go test ./cmd/ -run TestContinueFinalizeRefusesToAdvanceOnSupersededState -v -count=1 -> PASS
go test ./cmd/ -run 'TestContinueStaleStateDoesNotOverwritePausedState|TestContinueMissingPacketHonorsStoredDepth' -v -count=1 -> PASS (direct runCodexContinue callers)
go test ./cmd/ -run '<123-test broad finalize/parity/consolidation sweep>' -v -count=1 -> 123 PASS, 0 FAIL
go test ./cmd/... -count=1  -> ok (313-371s across two runs; first run caught the Deviation #2 leak, second run clean)
go test ./...  -count=1     -> ok, all 18 tested packages pass (cmd 313.6s, plus 17 pkg/* packages)
```

Note on the plan's literal `<verification>` block: `-run 'TestAdvancePhase|TestCodexContinue|TestCodexContinueFinalize|TestContinueFinalizeRefusesToAdvanceOnSupersededState'` matches 0 tests for the `TestCodexContinue` and `TestCodexContinueFinalize` fragments, because no test functions in this codebase are named with those exact prefixes (confirmed via `grep -c "^func TestCodexContinue" cmd/codex_continue_test.go` = 0). Per the plan's own Task 2/3 acceptance-criteria fallback ("broaden the -run pattern if this does not match the actual existing test names"), coverage was instead confirmed via `grep -l "runCodexContinue\b"` / `grep -l "advanceExternalContinue\|runCodexContinueFinalize"` to discover the real test names, run directly (see above), and via the full package and full-repo suites.

## Next Phase Readiness

- `advancePhase` exists under exactly that name, as required by 188-04 (the ratchet plan in the next wave), which roots its zero-tolerance reachability check at this literal function name.
- Both continue paths now share one atomic, supersession-checked commit core; the D-06 non-atomic trailing write is gone from `advanceExternalContinue`.
- No blockers for 188-04. `cmd/codex_continue.go` and `cmd/codex_continue_finalize.go` are otherwise unchanged in shape/return signatures, so the ratchet's call-graph BFS should see one clean root with two callers.

---
*Phase: 188-one-truth-for-failures-and-advances*
*Completed: 2026-08-19*

## Self-Check: PASSED

- FOUND: cmd/advance_phase.go
- FOUND: cmd/advance_phase_test.go
- FOUND: cmd/codex_continue.go
- FOUND: cmd/codex_continue_finalize.go
- FOUND: cmd/codex_continue_finalize_test.go
- FOUND: commit 8ac4d793 (Task 1)
- FOUND: commit 5d0be8a4 (Task 2)
- FOUND: commit e361f951 (Task 3)
