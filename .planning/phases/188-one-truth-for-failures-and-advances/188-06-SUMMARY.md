---
phase: 188-one-truth-for-failures-and-advances
plan: 06
subsystem: infra
tags: [concurrency, atomicity, colony-state, state-mutate, go, ast-ratchet]

# Dependency graph
requires:
  - phase: 188-one-truth-for-failures-and-advances (plans 01-05)
    provides: the midden unification, advancePhase() extraction, state-mutate field guard,
      atomicity ratchet, and retry-exhaustion record this plan's fixes build directly on
provides:
  - finalizeBlockedExternalContinue's write routed through the same
    UpdateJSONAtomically + supersession discipline advancePhase already uses (CR-01)
  - build-finalize's atomic commit (extracted as commitBuildFinalizeState) no longer discards
    its own fresh on-disk read (CR-02)
  - state-mutate's jq-like expression syntax (`.current_phase = N`) now requires the same
    --guard phase-advance:<N> contract the --field syntax already required (CR-03)
  - the atomicity ratchet now structurally detects any UpdateJSONAtomically closure that
    reassigns its fresh-read target wholesale, not just which storage primitive was used (WR-01)
affects: [any future phase touching COLONY_STATE.json writes, state-mutate, or the
  continue/build-finalize commit paths]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Extract-for-testability: when a function has no injectable concurrency hook, extract
      its one atomic-write block into a small, directly-callable function (commitBuildFinalizeState)
      so a concurrent-pause test can call it in isolation -- the same technique 188-05 (D-14)
      established for the autopilot retry-exhaustion record."
    - "Shrink-only, in-file allowlist for a newly-discovered defect class outside the current
      task's own scope (colonyStateDiscardedReadAllowlist) -- same philosophy as the
      JSON-file-backed testdata/colony_state_write_allowlist.json, sized to the one item found."

key-files:
  created: []
  modified:
    - cmd/codex_continue_finalize.go
    - cmd/codex_continue_finalize_test.go
    - cmd/review_depth_test.go
    - cmd/testdata/colony_state_write_allowlist.json
    - cmd/codex_build_finalize.go
    - cmd/build_attempt_external_test.go
    - cmd/state_cmds.go
    - cmd/state_cmds_test.go
    - cmd/colony_state_atomicity_ratchet_test.go

key-decisions:
  - "CR-02's fix required a currency guard SHAPED DIFFERENTLY from the review's own illustrative
    snippet: build-finalize's external/wrapper flow never separately commits a READY->EXECUTING
    checkpoint the way the direct-build path does (confirmed by reading
    runCodexBuildPlanOnlyWithOptions, which returns a manifest without writing COLONY_STATE.json
    at all, and by this repo's own test fixtures seeding State: StateREADY before calling
    build-finalize for the first time), so validateBuildFinalizeStateStillCurrent checks Paused
    + CurrentPhase + phase-not-already-completed, not state.State == EXECUTING."
  - "WR-01's extended ratchet found 3 candidate sites beyond CR-02; 2 were false positives from
    an initial too-shallow detector (target = f(target), a self-transform that still threads the
    fresh read's data through, e.g. normalizeLegacyColonyState) -- fixed by checking whether the
    target identifier appears anywhere in the RHS subtree, not just as a bare top-level identifier."
  - "The 1 genuine 4th finding (cmd/codex_build.go's rollbackCodexBuildFailure) is outside this
    plan's 4 named findings (CR-01/CR-02/CR-03/WR-01) and touches a file never assigned to this
    task -- logged to deferred-items.md and allowlisted (shrink-only), not silently fixed or
    silently ignored."

requirements-completed: []

# Metrics
duration: 47min
completed: 2026-08-20
---

# Phase 188 Plan 06: Code Review Fixes (CR-01, CR-02, CR-03, WR-01) Summary

**Closed three concurrency-safety bugs where a colony pause or other concurrent write could be silently clobbered by continue-finalize, build-finalize, or state-mutate's expression syntax, and taught the atomicity ratchet to structurally detect the exact discard shape that caused two of them.**

## Performance

- **Duration:** 47 min
- **Started:** 2026-08-20T00:59:44+02:00 (worktree base commit)
- **Completed:** 2026-08-20T01:46:34+02:00
- **Tasks:** 4 (CR-01, CR-02, CR-03, WR-01)
- **Files modified:** 9

## Accomplishments

- **CR-01**: `finalizeBlockedExternalContinue` (the blocked-continue-finalize path) no longer
  writes a raw, non-atomic snapshot of state captured before verification/gates/review ran.
  It now uses `store.UpdateJSONAtomically` with the same `validateRuntimeStateStillCurrent`
  supersession check `advancePhase` and `recordBlockedContinueWorkerFlow` already use, and
  returns `continueSupersededResult(...)` (nil error, `superseded: true`) when the runtime state
  has moved on — mirroring exactly how the sibling advance path already handled this.
- **CR-02**: `runCodexBuildFinalize`'s atomic commit (`committedState = updatedState` inside the
  `UpdateJSONAtomically` closure) no longer discards its own fresh on-disk read. The commit was
  extracted into `commitBuildFinalizeState` (a small, directly-testable function, mirroring the
  extract-for-testability technique 188-05 established for the autopilot retry record) and now
  mutates only the freshly-read value's own fields, guarded by a new
  `validateBuildFinalizeStateStillCurrent` check shaped for build-finalize's actual
  READY→BUILT transition (it never separately persists an EXECUTING checkpoint the way the
  direct-build path's own second commit does).
- **CR-03**: `state-mutate`'s free-form jq-like expression syntax (`.current_phase = N`) now
  requires the identical `--guard phase-advance:<N>` contract the `--field current_phase`
  syntax already enforced. The guard check was extracted into `validateCurrentPhaseGuard`,
  shared by both `executeFieldMode` and a new `guardCurrentPhaseSubExpression` check inside
  `executeExpression`'s sub-expression loop — every other field, and every other expression
  shape, remains unguarded exactly as before.
- **WR-01**: The atomicity ratchet (`cmd/colony_state_atomicity_ratchet_test.go`) gained a third
  check, `TestNoUpdateJSONAtomicallyDiscardsFreshRead`, that structurally detects any
  `UpdateJSONAtomically("COLONY_STATE.json", &target, func() error {...})` closure containing a
  bare `target = <expr>` reassignment where `<expr>` never references `target` — the exact shape
  both CR-01 and CR-02 reproduced, which was invisible to the two existing checks (they only
  inspect which storage primitive is called, not what the closure does with the value it reads).

## Task Commits

Each fix was committed atomically:

1. **CR-01: route finalizeBlockedExternalContinue through atomic write** - `9d694ec3` (fix)
2. **CR-02: stop build-finalize's atomic commit discarding its own fresh read** - `30ec0fce` (fix)
3. **CR-03: close state-mutate's expression-syntax current_phase bypass** - `67eacd56` (fix)
4. **WR-01: extend atomicity ratchet to catch fresh-read-discarding closures** - `b318cf1f` (test)

**Plan metadata:** committed together with this SUMMARY (see final commit below)

## Files Created/Modified

- `cmd/codex_continue_finalize.go` - CR-01 fix: `finalizeBlockedExternalContinue`'s atomic write + supersession handling; both call sites updated to surface `runStatus = "superseded"`
- `cmd/codex_continue_finalize_test.go` - CR-01 regression test (`TestFinalizeBlockedExternalContinueDoesNotOverwritePausedState`)
- `cmd/review_depth_test.go` - fixed a pre-existing test (`TestContinueFinalizeResultIncludesReviewDepth`) that called `finalizeBlockedExternalContinue` directly without seeding matching on-disk state, which CR-01's new currency check now requires
- `cmd/testdata/colony_state_write_allowlist.json` - removed the now-stale `finalizeBlockedExternalContinue` entry
- `cmd/codex_build_finalize.go` - CR-02 fix: extracted `commitBuildFinalizeState` + `validateBuildFinalizeStateStillCurrent`; `runCodexBuildFinalize` now calls the extracted function and propagates supersession errors distinctly
- `cmd/build_attempt_external_test.go` - CR-02 regression test (`TestCommitBuildFinalizeStateDoesNotOverwritePausedState`), calling `commitBuildFinalizeState` directly to reach the real vulnerability window past an unrelated earlier digest check
- `cmd/state_cmds.go` - CR-03 fix: `validateCurrentPhaseGuard` extracted and shared; `executeExpression` now takes `cmd *cobra.Command` and runs `guardCurrentPhaseSubExpression` per sub-expression
- `cmd/state_cmds_test.go` - CR-03: `TestStateMutateExpressionNumericStillWorks` updated to assert refusal; added `TestStateMutateExpressionCurrentPhaseSucceedsWithMatchingGuard` and `TestStateMutateExpressionNonCurrentPhaseFieldRemainsUnguarded`
- `cmd/colony_state_atomicity_ratchet_test.go` - WR-01: `colonyStateDiscardedReadSite`, `updateJSONAtomicallyTargetName`, `closureDiscardsFreshRead`, `rhsMentionsIdent`, `TestNoUpdateJSONAtomicallyDiscardsFreshRead`, and the small `colonyStateDiscardedReadAllowlist`

## Decisions Made

See `key-decisions` in the frontmatter above for the two decisions that most affected the shape
of the fixes (CR-02's currency-guard semantics; WR-01's false-positive correction and scope
boundary on the 4th finding).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `TestContinueFinalizeResultIncludesReviewDepth` broke because it never seeded matching on-disk state**
- **Found during:** CR-01 (running the full `Continue`-tagged test suite after the fix)
- **Issue:** This pre-existing test called `finalizeBlockedExternalContinue` directly with a
  `state` value it never wrote to disk. Before CR-01, the function didn't care (it wrote
  whatever `state` it was handed, unconditionally). After CR-01, the function re-reads
  COLONY_STATE.json fresh and refuses if it doesn't match — the test's constructed `state`
  therefore looked "superseded" relative to whatever was actually on disk, and the test's
  assertion (that `review_depth` appears in the blocked-result map) failed because a superseded
  result has a different, smaller shape.
- **Fix:** Added `store.SaveJSON("COLONY_STATE.json", state)` immediately before the call, so
  the fresh read CR-01's fix performs matches what the test constructed.
- **Files modified:** cmd/review_depth_test.go
- **Verification:** `go test ./cmd/ -run TestContinueFinalizeResultIncludesReviewDepth -v` passes
- **Committed in:** 9d694ec3 (CR-01 commit)

**2. [Rule 1 - Bug] WR-01's first detector implementation had 2 false positives**
- **Found during:** WR-01 (first run of the new ratchet check against the live codebase)
- **Issue:** `closureDiscardsFreshRead`'s original check only excluded `target = target` (the
  bare identifier). It incorrectly flagged `cmd/codex_plan.go`'s `state =
  normalizeLegacyColonyState(state)` and `cmd/codex_plan_finalize.go`'s `updatedState =
  normalizeLegacyColonyState(updatedState)` — both a legitimate self-transform idiom (confirmed
  by reading `normalizeLegacyColonyState`, which takes its argument by value and returns a
  transformed copy of the SAME value) that still threads the fresh read's own data through, not
  a wholesale discard.
- **Fix:** Added `rhsMentionsIdent`, which checks whether the target identifier appears anywhere
  in the RHS expression's own subtree (not just as a bare top-level identifier) before flagging.
- **Files modified:** cmd/colony_state_atomicity_ratchet_test.go
- **Verification:** re-ran `TestNoUpdateJSONAtomicallyDiscardsFreshRead`; the two false positives
  disappeared, leaving exactly one genuine finding (`cmd/codex_build.go`'s
  `rollbackCodexBuildFailure`, logged to deferred-items.md, see below)
- **Committed in:** b318cf1f (WR-01 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs found and fixed while verifying the four
assigned findings, neither expanding scope beyond what those four findings required).
**Impact on plan:** Both fixes were necessary for the four assigned findings' own tests to pass
correctly; neither touched unrelated behavior.

## Deferred Discoveries (logged, not fixed — see deferred-items.md)

WR-01's extended ratchet found a 4th, genuine instance of the same discard-fresh-read defect
class, in a file this plan was never assigned to touch: `cmd/codex_build.go`'s
`rollbackCodexBuildFailure` does `current = rollback` inside its `UpdateJSONAtomically` closure,
discarding the fresh read's fields not covered by its own `validateRuntimeStateStillCurrent`
call. Per the executor's scope-boundary rule (only auto-fix issues directly caused by the
current task's changes), this was **not fixed** — `cmd/codex_build.go` is unrelated to
CR-01/CR-02/CR-03. It is logged in `.planning/phases/188-one-truth-for-failures-and-advances/deferred-items.md`
(§3) with a recommended fix shape, and tracked (not hidden) via a small, explicit,
shrink-only `colonyStateDiscardedReadAllowlist` in the ratchet test file itself — a genuinely
NEW discard site anywhere else still fails the test; this one entry stops being tolerated the
moment it is fixed.

A pre-existing, unrelated `gofmt` struct-alignment drift in `cmd/codex_build.go` (confirmed via
`git diff --stat` to have zero changes from this plan's work) is also logged in
deferred-items.md (§4), not fixed.

## Proof (per CLAUDE.md's Definition of Done)

Each of CR-01, CR-02, CR-03 was proven fail-then-pass; WR-01 was proven by reintroducing CR-02's
exact original shape and confirming detection, then restoring byte-for-byte.

**CR-01** — `go test ./cmd/ -run TestFinalizeBlockedExternalContinueDoesNotOverwritePausedState -v`
against the unfixed code:
```
result[superseded] = <nil>, want true (result: map[string]interface {}{...})
expected the competing Paused=true write to survive; got Paused=false
--- FAIL: TestFinalizeBlockedExternalContinueDoesNotOverwritePausedState (0.00s)
```
After the fix: `--- PASS: TestFinalizeBlockedExternalContinueDoesNotOverwritePausedState (0.00s)`

**CR-02** — the atomic-commit block was extracted into `commitBuildFinalizeState` FIRST,
preserving the original bug verbatim, to prove the extraction itself reproduces the defect
before any fix was applied. `go test ./cmd/ -run TestCommitBuildFinalizeStateDoesNotOverwritePausedState -v`
against the extracted-but-still-buggy code:
```
expected superseded build-finalize commit error, got <nil>
--- FAIL: TestCommitBuildFinalizeStateDoesNotOverwritePausedState (0.01s)
```
After the fix: `--- PASS: TestCommitBuildFinalizeStateDoesNotOverwritePausedState (0.01s)`

(Note: an earlier attempt to test this by writing a concurrent pause directly to disk BEFORE
calling `runCodexBuildFinalize` — mirroring the CR-01 technique exactly — was intercepted by a
different, pre-existing, unrelated safety check inside `validateBuildAttemptManifestBinding`
[an `OriginalStateSHA` digest comparison], which refused with `"colony state changed after
build attempt ... was prepared"` before ever reaching CR-02's own vulnerable code. This is why
CR-02's test calls the extracted `commitBuildFinalizeState` directly instead of the full
`runCodexBuildFinalize` — the same reasoning the review itself anticipated: "no injectable
concurrency point.")

**CR-03** — `go test ./cmd/ -run TestStateMutateExpressionNumericStillWorks -v` against the
unfixed code:
```
expected `.current_phase = N` with no --guard to be refused, got: map[ok:true result:map[expr:.current_phase = 2 updated:true]]
--- FAIL: TestStateMutateExpressionNumericStillWorks (0.00s)
```
After the fix: `--- PASS: TestStateMutateExpressionNumericStillWorks (0.00s)`

**WR-01** — the extended ratchet was proven against CR-02's original shape by temporarily
reverting `commitBuildFinalizeState` to `committedState = params.UpdatedState` and re-running
`TestNoUpdateJSONAtomicallyDiscardsFreshRead`:
```
1 NEW store.UpdateJSONAtomically("COLONY_STATE.json", ...) call site(s) discard the primitive's
own fresh on-disk read by reassigning the target wholesale from another value...
  cmd/codex_build_finalize.go:commitBuildFinalizeState (discards fresh read of "committedState")
--- FAIL: TestNoUpdateJSONAtomicallyDiscardsFreshRead (0.11s)
```
The file was then restored via `git checkout -- cmd/codex_build_finalize.go`, confirmed
byte-for-byte identical to the committed fix via both `git diff --exit-code
cmd/codex_build_finalize.go` (exit 0) and a direct `diff` against a pre-reintroduction backup
copy (exit 0), and `TestNoUpdateJSONAtomicallyDiscardsFreshRead` re-confirmed passing.

## Issues Encountered

None beyond the two auto-fixed deviations documented above.

## User Setup Required

None - no external service configuration required.

## Full Verification Run

Before returning, the complete suite was run and is clean:
```
go build ./...        # clean, no output
go vet ./...           # clean, no output
go test ./... -race    # ok, all 18 packages, 0 failures, 0 data races
  github.com/calcosmic/Aether/cmd        408.891s (largest package; includes all new tests)
  ...all other packages ok...
```

## Next Phase Readiness

All four assigned findings (CR-01, CR-02, CR-03, WR-01) from `188-REVIEW.md` are closed, each
with a fail-then-pass regression test committed atomically. One additional, out-of-scope defect
instance of the same class was found, logged, and safely deferred (not silently hidden — the
ratchet still fails if it regresses or if any OTHER new instance appears). No blockers for
whatever consumes this phase's review-fix wave next.

---
*Phase: 188-one-truth-for-failures-and-advances*
*Plan: 06*
*Completed: 2026-08-20*

## Self-Check: PASSED

- FOUND: cmd/codex_continue_finalize.go
- FOUND: cmd/codex_continue_finalize_test.go
- FOUND: cmd/review_depth_test.go
- FOUND: cmd/testdata/colony_state_write_allowlist.json
- FOUND: cmd/codex_build_finalize.go
- FOUND: cmd/build_attempt_external_test.go
- FOUND: cmd/state_cmds.go
- FOUND: cmd/state_cmds_test.go
- FOUND: cmd/colony_state_atomicity_ratchet_test.go
- FOUND: .planning/phases/188-one-truth-for-failures-and-advances/deferred-items.md
- FOUND commit 9d694ec3 (CR-01) in `git log --oneline --all`
- FOUND commit 30ec0fce (CR-02) in `git log --oneline --all`
- FOUND commit 67eacd56 (CR-03) in `git log --oneline --all`
- FOUND commit b318cf1f (WR-01) in `git log --oneline --all`
- `go build ./...`, `go vet ./...`, `go test ./... -race` all clean (0 failures, 0 data races,
  18 packages) — verified immediately before writing this SUMMARY
