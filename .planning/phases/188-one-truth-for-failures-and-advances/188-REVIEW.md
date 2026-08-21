---
phase: 188-one-truth-for-failures-and-advances
reviewed: 2026-08-19T22:57:04Z
depth: standard
files_reviewed: 25
files_reviewed_list:
  - cmd/midden_shared.go
  - cmd/midden_shared_test.go
  - cmd/midden_unification_test.go
  - cmd/autopilot.go
  - cmd/context.go
  - cmd/context_test.go
  - cmd/immune.go
  - cmd/medic_scanner.go
  - cmd/medic_scanner_test.go
  - cmd/memory_health.go
  - cmd/advance_phase.go
  - cmd/advance_phase_test.go
  - cmd/codex_continue.go
  - cmd/codex_continue_finalize.go
  - cmd/codex_continue_finalize_test.go
  - cmd/state_cmds.go
  - cmd/state_cmds_test.go
  - cmd/codex_build_finalize.go
  - cmd/codex_build_finalize_test.go
  - cmd/colony_state_atomicity_ratchet_test.go
  - cmd/testdata/colony_state_write_allowlist.json
  - cmd/autopilot_retry_record.go
  - cmd/autopilot_retry_record_test.go
  - cmd/compatibility_cmds.go
  - cmd/run_autopilot_test.go
findings:
  critical: 3
  warning: 1
  info: 2
  total: 6
status: issues_found
---

# Phase 188: Code Review Report

**Reviewed:** 2026-08-19T22:57:04Z
**Depth:** standard (with targeted execution probes per reviewer brief)
**Files Reviewed:** 25
**Status:** issues_found

## Summary

Three of the phase's six claims check out cleanly under direct execution: the midden path
unification (claim 1) is done correctly for all four named consumers, with a second latent bug
(`immune.go` reading a nonexistent `entry["description"]` key) fixed as a side effect; the
legacy-manifest warning (claim 6) fires exactly and only on the legacy path, proven by a passing
positive and negative test; the autopilot retry record (claim 5) genuinely captures both attempts'
error text, not just that retries were exhausted.

The other two claims are each only partially true, and the gap in both cases is the exact failure
mode this phase exists to close: a stale in-memory copy of `COLONY_STATE.json` silently overwriting
a fresher one.

**`advancePhase` itself is correctly built and its zero-exemption reachability ratchet holds** — this
was independently reconfirmed by the orchestrator before this review and is not restated here.
But `advancePhase` is only reachable from the two *successful* commit points of continue and
continue-finalize. Both of those functions have a **sibling failure/blocked-path branch that was
never migrated to the same discipline**, and both branches perform the identical class of write the
`advancePhase` extraction was built to eliminate. One of these — `finalizeBlockedExternalContinue` —
was proven to silently discard a concurrent state change by an executed, throwaway probe (written,
run, and deleted per the review brief; reproduction steps below). The other — `runCodexBuildFinalize`'s
main commit — was confirmed by direct line-for-line comparison against its own correctly-guarded
sibling in the same codebase (`codex_build.go`'s in-process build commit), and by the complete absence
of any concurrent-write test for it, unlike the direct-build path's own such test. Separately, the new
AST atomicity ratchet (claim 4) — while its own advertised zero-exemption property for `advancePhase`
holds — has a structural blind spot that makes it incapable of ever catching either of these: it
never inspects what a `store.UpdateJSONAtomically` closure actually does with the value it freshly
read, only whether the call is `UpdateJSONAtomically` at all.

Claim 3 ("ungated `state-mutate --field current_phase` is now refused") is also only half-delivered:
the refusal lives in one of two equivalent code paths. `state-mutate '.current_phase = N'` (the
free-form jq-like expression syntax, used elsewhere in this system's own command playbooks) performs
the exact same mutation with **zero guard requirement**, proven by an existing, currently-passing
test in the same test file that sits three tests below the new guard tests.

## Critical Issues

### CR-01: continue-finalize's blocked path still clobbers concurrent state — confirmed by execution

**File:** `cmd/codex_continue_finalize.go:1044-1052` (function `finalizeBlockedExternalContinue`, called from `runCodexContinueFinalize` whenever gates or review fail)

**Issue:**

```go
if err := recordExternalContinueWorkerFlow(workerFlow); err != nil {
    return nil, state, err
}
blockedState := state
blockedState.Events = append(trimmedEvents(blockedState.Events), continueWorkerFlowEvents(now, workerFlow)...)
blockedState.Events = append(blockedState.Events, fmt.Sprintf("%s|continue_blocked|continue-finalize|Continue blocked before advancement", now.Format(time.RFC3339)))
if err := store.SaveJSON("COLONY_STATE.json", blockedState); err != nil {
    return nil, state, fmt.Errorf("failed to save colony state: %w", err)
}
```

`state` here is the value `runCodexContinueFinalize` loaded once at the very top of the call
(`validateExternalContinueState` → `loadActiveColonyState()`), before verification, gates, review,
and every other step in between ran. When any of those steps causes a block (the common case this
function exists for), this code does a raw, non-atomic `store.SaveJSON` of that stale snapshot —
never re-reading the current on-disk file, never calling `validateRuntimeStateStillCurrent` or
`validateRuntimeStateMatchesExpected`. Any concurrent write to `COLONY_STATE.json` during that
window (an operator pausing the colony, `aether flag`, another background writer) is silently
discarded and replaced wholesale.

This is the identical bug class 188-02 fixed for the *successful* advance path
(`advanceExternalContinue`, which correctly calls `advancePhase`). The sibling *blocked* path was
never migrated. Compare to the correctly-guarded blocked path on the **default** (non-finalize)
continue flow, `recordBlockedContinueWorkerFlow` (`cmd/codex_continue.go:3671-3691`), which does
exactly this write through `store.UpdateJSONAtomically` with a `validateRuntimeStateStillCurrent`
check inside the closure — proving the correct pattern was known and used elsewhere in this same
phase, just not applied here.

**Confirmed by execution:** a throwaway probe (written to `cmd/zzz_probe_finalize_blocked_test.go`,
run, and deleted — not part of the delivered test suite) seeded a colony state, captured it the way
`runCodexContinueFinalize` does, wrote a concurrent `Paused = true` change directly via the store
(mirroring the existing `TestContinueFinalizeRefusesToAdvanceOnSupersededState` technique), then
called `finalizeBlockedExternalContinue` with the stale, pre-pause state and a forced gate failure.
Result:

```
BUG CONFIRMED: concurrent Paused=true write was clobbered by finalizeBlockedExternalContinue's
stale SaveJSON; after.Paused = false (want true)
```

The phase's own new allowlist (`cmd/testdata/colony_state_write_allowlist.json`) already lists this
exact site — `codex_continue_finalize.go` / `finalizeBlockedExternalContinue` / `SaveJSON` — tagged
`"pre-existing, not migrated by phase 188 (see 188-CONTEXT.md D-07)"`. That blanket "~30 sites, out
of scope" reasoning is appropriate for the other 29 entries (unrelated commands like
`worktree-allocate`, `entomb`, `suggest-approve`), but not for this one: it is the direct sibling,
in the same file, of the exact function this phase's headline fix (`advancePhase`) targets, on the
same request, with the same long-running-verification race window the phase's own fix exists to
close.

**Fix:**
```go
var updated colony.ColonyState
if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
    if err := validateRuntimeStateStillCurrent(updated, phase.ID, state.BuildStartedAt, colony.StateEXECUTING, colony.StateBUILT); err != nil {
        return err
    }
    updated.Events = append(trimmedEvents(updated.Events), continueWorkerFlowEvents(now, workerFlow)...)
    updated.Events = append(updated.Events, fmt.Sprintf("%s|continue_blocked|continue-finalize|Continue blocked before advancement", now.Format(time.RFC3339)))
    return nil
}); err != nil {
    if errors.Is(err, errRuntimeStateSuperseded) {
        return continueSupersededResult(state, phase, err), state, nil // mirror advanceExternalContinue's own supersession handling
    }
    return nil, state, fmt.Errorf("failed to save colony state: %w", err)
}
blockedState := updated
```
Mirror `recordBlockedContinueWorkerFlow`'s exact shape; the caller already knows how to render a
`superseded` result (`advanceExternalContinue` does it a few lines away in the same file).

---

### CR-02: build-finalize's main state commit has the same stale-overwrite shape, invisible to the new ratchet

**File:** `cmd/codex_build_finalize.go:611-619` (function `runCodexBuildFinalize`, the primary commit for every external/wrapper-driven `aether build-finalize` call)

**Issue:**

```go
// Atomically commit the colony state mutation.
var committedState colony.ColonyState
if err := store.UpdateJSONAtomically("COLONY_STATE.json", &committedState, func() error {
    committedState = updatedState
    return nil
}); err != nil {
    finishAttempt(buildAttemptFailed, "failed to commit external built lifecycle state", err)
    return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to save built colony state: %w", err)
}
```

`UpdateJSONAtomically` freshly unmarshals the current on-disk file into `committedState` before this
closure runs — but the closure immediately throws that fresh read away and replaces it wholesale
with `updatedState`, a value built earlier in the function from a `state` loaded once, long before
the checkpoint write, build-attempt transition, claims write, outcome reports, and (in worktree mode)
a full worktree merge. This is mechanically the same shape as the `updated = state` bug 188-02's own
`advance_phase.go` doc comment describes as "structurally impossible to reintroduce" — except it
still exists here, on the build side.

Compare the corresponding **direct-build** commit, `runCodexBuildWithOptions`
(`cmd/codex_build.go:693-707`), which uses the identical `UpdateJSONAtomically` shape but correctly
guards it:
```go
if err := store.UpdateJSONAtomically("COLONY_STATE.json", &committedState, func() error {
    if err := validateRuntimeStateStillCurrent(committedState, phaseNum, &startedAt, colony.StateEXECUTING); err != nil {
        return err
    }
    committedState.State = colony.StateBUILT
    reconcileCompletedBuildTasks(&committedState, phaseNum, dispatches)
    committedState.Events = append(trimmedEvents(committedState.Events), ...)
    return nil
})
```
That direct-build path is exactly what `TestBuildFinalizationDoesNotOverwritePausedState`
(`cmd/codex_build_test.go`) proves resists a concurrent pause. **No equivalent test exists for
`runCodexBuildFinalize`** — `cmd/codex_build_finalize_test.go` (required reading for this review) has
zero references to `Paused` anywhere in the file.

This also demonstrates a real gap in the phase's own AST ratchet (see WR-01): the ratchet only flags
`SaveJSON`/`AtomicWrite` calls, treating any `UpdateJSONAtomically` call as inherently safe. This
call site passes that check cleanly — it is invisible to both new ratchet tests — despite reproducing
the exact clobber shape the ratchet exists to police.

Not independently re-executed end-to-end for this specific function: the vulnerable window opens
and closes inside one synchronous, hook-free call, and `runCodexBuildFinalize` has no injectable
concurrency point the way `runCodexBuildWithOptions`'s own test exploits (a worker-invocation hook).
The mechanism was instead confirmed by executing CR-01's identical pattern in the sibling
continue-finalize function, plus this direct, line-for-line comparison against the correctly-guarded
sibling in the same file family.

**Fix:**
```go
var committedState colony.ColonyState
if err := store.UpdateJSONAtomically("COLONY_STATE.json", &committedState, func() error {
    if err := validateRuntimeStateStillCurrent(committedState, phaseNum, &startedAt, colony.StateEXECUTING); err != nil {
        return err
    }
    committedState.State = colony.StateBUILT
    reconcileCompletedBuildTasks(&committedState, phaseNum, dispatches)
    committedState.Events = append(trimmedEvents(committedState.Events),
        fmt.Sprintf("%s|build_completed|build-finalize|Phase %d external Task workers recorded", completedAt.Format(time.RFC3339), phaseNum),
    )
    if binding.Legacy {
        committedState.Events = append(committedState.Events, ...) // existing legacy-accepted event
    }
    return nil
}); err != nil {
    ...
}
```
i.e. mutate fields on the freshly-read `committedState`, never reassign it wholesale from
`updatedState`, and propagate a supersession error the same way `advancePhase`'s callers do.

---

### CR-03: `state-mutate`'s `current_phase` guard has a complete, currently-open bypass

**File:** `cmd/state_cmds.go` — guard added at `executeFieldMode`'s `case "current_phase":`
(lines 272-303); bypass route is `executeExpression` → `applySubExpression` → `applyFieldSet`
(lines 348-374, 478-497), which performs no field-name-specific validation at all.

**Issue:** `executeFieldMode`'s `current_phase` branch now correctly refuses to move
`state.CurrentPhase` unless `--guard phase-advance:<N>` is supplied and matches — this is real and
well-tested (`TestStateMutateCurrentPhaseRequiresGuard`,
`TestStateMutateCurrentPhaseRejectsWrongGuardType`,
`TestStateMutateCurrentPhaseRejectsMismatchedGuardTarget`,
`TestStateMutateCurrentPhaseSucceedsWithMatchingGuard`, all passing). But `state-mutate` has a
second, completely independent invocation form — a bare jq-like expression argument, e.g.
`state-mutate '.current_phase = 3'` — that reaches `executeExpression`, which has no concept of
"which field is this" at all; it dispatches purely on expression *shape* (`reFieldSet`, etc.) and
writes straight through `sjson`/`AtomicWrite` with zero guard check.

**Confirmed by execution** — this is not a hypothetical, it is the currently-passing behavior of an
existing, unmodified test three tests below the new guard tests in the same file:

```
$ go test ./cmd/ -run TestStateMutateExpressionNumericStillWorks -v
=== RUN   TestStateMutateExpressionNumericStillWorks
--- PASS: TestStateMutateExpressionNumericStillWorks (0.00s)
PASS
```

`cmd/state_cmds_test.go:213-247` (`TestStateMutateExpressionNumericStillWorks`) runs
`rootCmd.SetArgs([]string{"state-mutate", `.current_phase = 3`})` with **no `--guard` flag anywhere**
and asserts `err == nil` and `updated.CurrentPhase == 3` — i.e. it directly proves the refusal this
phase's own criterion 3 claims to have added does not apply to this path. This is exactly the
"new test creates a false sense of completeness while an adjacent, pre-existing test proves the gap
is still open" pattern the review brief asked to watch for — the four new guard tests and this one
old test sit in the same file, and nothing here reconciles them.

This is also a direct answer to "can `advancePhase` be bypassed?": yes — `current_phase` can be
moved (and, via the same mechanism already exercised by `TestStateMutateBracket`, any phase's status
set straight to `"completed"`) via `state-mutate`'s expression syntax without ever calling
`advancePhase`, without its supersession check, without marking tasks complete, and without emitting
the phase-advance events `advancePhase` emits. `state-mutate` with free-form expressions is not a
theoretical/dev-only surface either — it is referenced throughout this repo's own command playbooks
(`.aether/docs/command-playbooks/*.md`).

**Fix:** give `executeExpression` (or `applyFieldSet` specifically) the same field-name check
`executeFieldMode` has — detect when a sub-expression's target path is (or begins with)
`current_phase` and require/re-validate the identical `--guard phase-advance:<N>` contract before
applying it, e.g. by extracting the guard check into a shared helper both `executeFieldMode` and
`executeExpression` call before touching that specific field.

## Warnings

### WR-01: The new atomicity ratchet cannot detect either CR-01 or CR-02 by design

**File:** `cmd/colony_state_atomicity_ratchet_test.go:97-116` (`colonyStateWriteSelectors`)

**Issue:** The ratchet's own zero-exemption reachability property for `advancePhase` is real and
holds (independently reconfirmed before this review). But both new checks (`TestColonyStateWriteAllowlistOnlyShrinks`
and `TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically`) key entirely off which *primitive*
a call site uses — `SaveJSON`/`AtomicWrite` are flagged, `UpdateJSONAtomically` is unconditionally
treated as the "safe primitive this ratchet does not flag" (the file's own words). Neither check
ever inspects what a matched `UpdateJSONAtomically` call's mutate closure actually does with the
freshly-read pointer target. A closure of the shape `target = someOtherStaleVariable` — mechanically
identical, at the call-site level, to the original clobber bug this whole phase exists to fix —
passes both ratchet tests cleanly. CR-02 is a live, currently-shipping instance of exactly this: a
`store.UpdateJSONAtomically("COLONY_STATE.json", ...)` call whose closure discards the fresh read,
and it is invisible to both ratchet tests. The ratchet's own extensive "known evasions" doc comment
(lines 40-74) documents four other evasion classes it deliberately closed off; this one is not among
them, suggesting it was not considered.

**Fix:** Extend `recordColonyStateCallsAndSites` (or a new pass) to additionally flag any
`UpdateJSONAtomically("COLONY_STATE.json", &target, func() error { ... })` whose closure body
contains a bare assignment `target = <expr>` where `<expr>` is not a selector/field access rooted at
`target` itself (i.e. a wholesale reassignment of the pointer target, not a mutation of one of its
fields). This is the actual semantic property "atomic" is standing in for, and it is exactly the
kind of AST-structural check this file already knows how to write.

## Info

### IN-01: Midden path still hardcoded as a literal instead of the new shared constant

**File:** `cmd/medic_scanner.go:511`
**Issue:** `scanDataFiles`'s structured-file list still writes the path as the literal string
`"midden.json"` rather than referencing `middenCanonicalPath` (`cmd/midden_shared.go:17`). It's the
correct value today, so this is not a live bug, but it directly contradicts `midden_shared.go`'s own
doc comment ("every reader in this package must converge on it via `loadMiddenFile` rather than
hardcoding the string again") and means this call site will not follow if the canonical path is ever
revised. The broader codebase has roughly fifteen more such pre-existing, correct-but-unconverged
call sites outside this phase's diff (`cmd/midden_cmds.go`, `cmd/status.go`, `cmd/entomb_cmd.go`,
`cmd/spawn_budget.go`) worth a follow-up cleanup pass, though they are out of this review's file
scope.
**Fix:** `{middenCanonicalPath, "midden.json"}` instead of `{"midden.json", "midden.json"}`.

### IN-02: `appendMiddenEntry`'s ID can collide within the same second

**File:** `cmd/midden_shared.go:38`
**Issue:** `ID: fmt.Sprintf("midden_%d_%d", time.Now().UTC().Unix(), os.Getpid())` — second-
granularity timestamp plus PID. Two entries appended by the same process within the same wall-clock
second (plausible for rapid sequential callers, or two racing goroutines serialized by the same file
lock) get the same ID. This pattern is carried forward unchanged from the two pre-existing writers
`appendMiddenEntry` replaces (`cmd/midden_cmds.go`'s `midden-write`, `cmd/spawn_budget.go`), so it is
not a regression, but it is now baked into the one shared helper this phase's own doc comment invites
future writers to converge on. Any consumer that keys off entry ID (`midden-acknowledge`,
`midden-tag`) could then be unable to distinguish two colliding entries.
**Fix:** Add a monotonic counter, nanosecond-resolution timestamp, or short random suffix to the ID.

---

_Reviewed: 2026-08-19T22:57:04Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
