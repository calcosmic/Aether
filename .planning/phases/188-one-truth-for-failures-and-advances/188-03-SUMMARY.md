---
phase: 188-one-truth-for-failures-and-advances
plan: 03
subsystem: cli-safety
tags: [go, cobra, cli-guard, state-mutate, build-finalize, event-log, colony-state]

# Dependency graph
requires: []
provides:
  - "state-mutate --field current_phase refuses to run unless invoked with a phase-advance guard matching the exact value being written"
  - "build-finalize warns loudly (stderr + durable Events entry) every time it accepts a dispatch_manifest with no attempt binding at all, without changing that branch's accept-and-proceed behavior"
affects: [state-mutate, build-finalize, colony-state-guard, attempt-binding]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Mandatory-guard-for-one-field: a CLI flag (--guard) that is optional for every other purpose becomes required and type/target-checked for one specific, destructive field mutation, without touching the flag's other call sites"
    - "Defer durable-event append to the point in a function where the state variable is provably stable, not to the point where the triggering condition was first observed, when an intervening call in the same function can reassign that variable from a fresh disk read"

key-files:
  created: []
  modified:
    - cmd/state_cmds.go
    - cmd/state_cmds_test.go
    - cmd/codex_build_finalize.go
    - cmd/codex_build_finalize_test.go

key-decisions:
  - "current_phase's guard check lives inside executeFieldMode's current_phase case (not the outer RunE), so only this one --field value gains a mandatory-guard requirement; every other --field case is untouched"
  - "The legacy-manifest Events append is written where updatedState first stabilizes (alongside this file's own build_completed event), not immediately after validateBuildAttemptManifestBinding as the plan's literal text suggested -- reconcilePriorCompletedPhaseTasksFromTrustedManifests can reassign `state` wholesale from a fresh disk read a few lines later, which would silently discard an event appended before it runs"
  - "The stderr warning is routed through this file's own stderr package var via visualFprintf (matching collectPendingSuggestions's existing warning in the same file), not raw os.Stderr -- functionally identical at runtime, but testable, and consistent with this file's own established convention"

patterns-established:
  - "A destructive CLI field write can require cmd.Flags().Changed(\"guard\") plus a type+target match against the value being written, reusing an already-passing outer gate check rather than re-running it"

requirements-completed: []

# Metrics
duration: ~35min
completed: 2026-08-19
---

# Phase 188 Plan 03: State-Mutate Guard + Legacy Manifest Warning Summary

**Closed the ungated `state-mutate --field current_phase` bypass with a mandatory, type-and-target-checked guard, and made build-finalize's already-accepted legacy (no-attempt-binding) manifest path announce itself on stderr and in a durable Events entry, on every occurrence.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-08-19
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- `aether state-mutate --field current_phase` now refuses to write unless invoked with `--guard phase-advance:<N>` where `<N>` exactly matches `--value`; every other `--field` case is unaffected.
- `codex_build_finalize.go` now detects `binding.Legacy` and, on every occurrence, prints a plain-language stderr warning immediately and appends a phase-numbered `manifest_legacy_accepted` entry to `state.Events` once the request's state has stabilized -- without changing the branch's accept-and-proceed behavior at all (the ROADMAP's Audit Addendum keeps this branch on purpose).
- Both fixes proven with RED-then-GREEN commits: each new test was run against the original code first and confirmed to fail, then again after the fix and confirmed to pass.

## Task Commits

Each task is TDD (test → feat), each half committed atomically:

1. **Task 1: Require a matching phase-advance guard for state-mutate --field current_phase**
   - `04e29047` (test) - add failing tests for ungated current_phase guard
   - `570ef211` (feat) - require a matching phase-advance guard for current_phase
2. **Task 2: Loud warning when build-finalize accepts a legacy unbound manifest**
   - `d41dad71` (test) - add failing test for silent legacy manifest acceptance
   - `cda3b07a` (feat) - warn loudly when build-finalize accepts a legacy manifest

## Files Created/Modified

- `cmd/state_cmds.go` - `executeFieldMode`'s `current_phase` case now requires `cmd.Flags().Changed("guard")`, parses the guard as `phase-advance:<N>`, and refuses unless `<N>` equals the phase number being written
- `cmd/state_cmds_test.go` - 4 new tests: `TestStateMutateCurrentPhaseRequiresGuard`, `TestStateMutateCurrentPhaseRejectsWrongGuardType`, `TestStateMutateCurrentPhaseRejectsMismatchedGuardTarget`, `TestStateMutateCurrentPhaseSucceedsWithMatchingGuard`, plus a shared `phaseAdvanceReadyState()` fixture helper
- `cmd/codex_build_finalize.go` - `runCodexBuildFinalize` checks `binding.Legacy` right after it is computed (stderr warning) and again once `updatedState` is final (Events append)
- `cmd/codex_build_finalize_test.go` - `TestBuildFinalizeWarnsOnLegacyUnboundManifest` (two subtests: legacy path warns on both channels; ordinary bound path stays silent on both)

## Decisions Made

- **Guard placement inside the `current_phase` case, not the outer `RunE`.** Keeps the new requirement scoped to exactly the one destructive field named by the ROADMAP criterion; every other `--field` case (`goal`, `state`, `milestone`, `colony_depth`, `plan_granularity`, `colony_name`, `parallel_mode`, unknown-field passthrough) is byte-for-byte unaffected, confirmed by running the full pre-existing `TestStateMutate*` suite (21 tests) unmodified.
- **No second gate-check re-run.** The new logic only confirms the `--guard` already supplied (and already verified by the outer `RunE`'s `enforceGuard` when non-empty) is a `phase-advance` guard for the exact phase number being written. It does not call `runGateCheck` a second time -- that would duplicate work `enforceGuard` already did and risk a second, differently-timed gate evaluation.
- **Deferred the Events append for the legacy-manifest warning (see Deviations below) -- a correctness fix, not a style choice.**
- **Reused this file's own `stderr` package var via `visualFprintf`, not raw `os.Stderr` (see Deviations below).**

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug prevention] Deferred the legacy-manifest Events append to avoid a silent-drop edge case**

- **Found during:** Task 2, while tracing every reassignment of the `state` variable between the `validateBuildAttemptManifestBinding` call and the final `updatedState` commit, to place the D-11 Events append correctly.
- **Issue:** The plan's `<action>` text says to append the Events entry "immediately after" the `validateBuildAttemptManifestBinding` call, inside the same `if binding.Legacy` block as the stderr warning. Reading the surrounding code (`cmd/codex_build_finalize.go:438`, `reconcilePriorCompletedPhaseTasksFromTrustedManifests`) shows that a few lines after that point, `state` is reassigned wholesale: `state, _, err = reconcilePriorCompletedPhaseTasksFromTrustedManifests(root, state, phaseNum)`. When `phaseNum > 1` **and** a prior, already-completed phase has an incomplete task needing repair, that function calls `store.UpdateJSONAtomically` with a value `updated` freshly *loaded from disk* -- not derived from the in-memory `state` I would have just appended an event to -- and returns `updated`. In that combination of conditions, an event appended immediately after the binding check would be silently discarded before it ever reaches the committed state, violating D-11's explicit "on every occurrence" requirement (the exact class of gap this repo's Definition of Done exists to catch -- a test could easily pass on the common case while the underlying implementation still had a real gap in a rarer one).
- **Fix:** Split the two channels. The stderr warning stays exactly where the plan says (immediately after the binding check, so it still fires even if something later in the same call fails). The Events append moved to the point later in the same function where `updatedState` is already known to be final and stable -- specifically, right alongside this file's own pre-existing `build_completed` Events append, which is the proven-safe point this exact function already uses for its own request-scoped Events. Both messages repeat the identical plain-language explanation, so the two-channel contract (D-11) is unchanged in substance, only in exactly where within the function the second channel is written.
- **Files modified:** `cmd/codex_build_finalize.go`
- **Verification:** `TestBuildFinalizeWarnsOnLegacyUnboundManifest` asserts the Events entry exists in the returned `updatedState` (the value that gets durably committed); the fixture used (`setupExternalBuildAttemptTest`, phase 1) doesn't itself exercise the reassignment edge case, but the fix is general -- it is correct for phase 1 exactly as it was already correct in the plan's literal placement, and is *additionally* correct for the `phaseNum > 1` + prior-phase-repair case the literal placement would have silently dropped.
- **Committed in:** `cda3b07a` (Task 2 feat commit)

### Implementation-detail note (not a behavior deviation)

**2. Used this file's own `stderr` package var (`visualFprintf(stderr, ...)`) instead of raw `os.Stderr`**

The plan and PATTERNS.md both cite `cmd/init_cmd.go:202`'s `fmt.Fprintf(os.Stderr, ...)` as the precedent to copy, and the acceptance criteria's literal text names `fmt.Fprintf(os.Stderr,` as what should appear. This file (`cmd/codex_build_finalize.go`) already has its own, closer precedent for exactly this situation -- an unconditional, human-facing warning that must also be testable -- at `collectPendingSuggestions`'s existing `visualFprintf(stderr, "warning: suggest-analyze did not run...")`, a few dozen lines below the new code. `stderr` is a package-level `io.Writer` that defaults to `os.Stderr` and is reassignable in tests (already used this way throughout `cmd/*_test.go` via `saveGlobals`/`t.Cleanup`); raw `os.Stderr` is not safely reassignable per-test without redirecting the actual OS file descriptor. Using `visualFprintf(stderr, ...)` produces byte-identical real-world output (still goes to the real stderr stream in production) while making `TestBuildFinalizeWarnsOnLegacyUnboundManifest`'s stderr assertions possible without invasive file-descriptor redirection. Flagging this explicitly since it does not literally match the cited precedent's function name, even though it satisfies the underlying requirement (unconditional, no suppressing conditional, verifiable by reading) and is a closer match to this exact file's own existing pattern.

- **Files modified:** `cmd/codex_build_finalize.go`

---

**Total deviations:** 1 auto-fixed (Rule 1, bug prevention) + 1 implementation-detail note (no behavior change).
**Impact on plan:** Both changes strengthen correctness/testability of exactly what the plan asked for; neither expands scope. The legacy branch itself (D-10, accepted residue) is untouched -- still accepted, still proceeds to `beginBuildAttempt` exactly as before.

## Issues Encountered

None beyond the one deviation documented above.

## Failing-Test Proof (Definition of Done)

Both tests were run against the pre-fix code and confirmed to fail before being run against the fix and confirmed to pass, per this repo's Definition of Done ("a command exists that fails when the requirement is unmet").

**Task 1 -- run against the original, unguarded `executeFieldMode` (`cmd/state_cmds.go` before commit `570ef211`):**

```
=== RUN   TestStateMutateCurrentPhaseRequiresGuard
    state_cmds_test.go:299: expected state-mutate --field current_phase with no --guard to be refused, got: map[ok:true result:map[field:current_phase updated:true value:2]]
--- FAIL: TestStateMutateCurrentPhaseRequiresGuard (0.00s)
=== RUN   TestStateMutateCurrentPhaseRejectsWrongGuardType
    state_cmds_test.go:333: expected current_phase with a task-complete guard to be refused, got: map[ok:true result:map[field:current_phase updated:true value:2]]
--- FAIL: TestStateMutateCurrentPhaseRejectsWrongGuardType (0.00s)
=== RUN   TestStateMutateCurrentPhaseRejectsMismatchedGuardTarget
    state_cmds_test.go:368: expected mismatched guard target (phase-advance:2 for --value 3) to be refused, got: map[ok:true result:map[field:current_phase updated:true value:3]]
--- FAIL: TestStateMutateCurrentPhaseRejectsMismatchedGuardTarget (0.00s)
=== RUN   TestStateMutateCurrentPhaseSucceedsWithMatchingGuard
--- PASS: TestStateMutateCurrentPhaseSucceedsWithMatchingGuard (0.00s)
FAIL
```

(The fourth test passes even pre-fix because it describes existing, unchanged behavior -- a genuine matching guard already worked; it is the negative-space lock, not evidence of the bug.)

**Task 2 -- run against the original `runCodexBuildFinalize` (`cmd/codex_build_finalize.go` before commit `cda3b07a`):**

```
=== RUN   TestBuildFinalizeWarnsOnLegacyUnboundManifest
=== RUN   TestBuildFinalizeWarnsOnLegacyUnboundManifest/a_manifest_with_no_attempt_binding_is_accepted_and_warns_loudly
    codex_build_finalize_test.go:1873: expected an unconditional stderr warning for a legacy unbound manifest, got none
=== RUN   TestBuildFinalizeWarnsOnLegacyUnboundManifest/an_ordinary_bound_manifest_stays_silent_on_both_channels
--- FAIL: TestBuildFinalizeWarnsOnLegacyUnboundManifest (0.20s)
    --- FAIL: TestBuildFinalizeWarnsOnLegacyUnboundManifest/a_manifest_with_no_attempt_binding_is_accepted_and_warns_loudly (0.11s)
    --- PASS: TestBuildFinalizeWarnsOnLegacyUnboundManifest/an_ordinary_bound_manifest_stays_silent_on_both_channels (0.09s)
FAIL
```

(The "stays silent" subtest already passes pre-fix -- there is no warning on either path yet -- which is exactly the no-regression half of the contract the fix must preserve, not evidence of a bug.)

Both were re-run after the corresponding `feat` commit and pass (see Verification below).

## Verification

```
go build ./cmd/                                              -> clean
go vet ./cmd/                                                 -> clean
go test ./cmd/ -run 'TestStateMutate|TestBuildFinalize' -v -count=1  -> PASS (all cases, including all pre-existing ones)
go test ./cmd/... -count=1                                    -> ok  	github.com/calcosmic/Aether/cmd	352.017s
go build ./...                                                -> clean
go vet ./...                                                  -> clean
go test ./... -count=1                                        -> every package ok (cmd, cmd/aether, pkg/agent, pkg/agent/curation,
                                                                   pkg/cache, pkg/codegraph, pkg/codex, pkg/colony, pkg/downloader,
                                                                   pkg/events, pkg/exchange, pkg/graph, pkg/learn, pkg/llm,
                                                                   pkg/memory, pkg/smoke, pkg/storage, pkg/terminal, pkg/trace)
```

Zero regressions across the entire repository, not just the two touched files.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Both ROADMAP criteria this plan owns (3 and 6) are implemented, tested, and proven to fail-before-fix.
- The legacy unbound-manifest branch (D-10, accepted residue) is untouched in behavior -- still accepted, not gated or hardened. If a future phase decides to tighten it further, that is a new decision, not something this plan pre-empted.
- No coordination needed with sibling plans 188-01, 188-02, 188-04, 188-05 -- this plan's files (`cmd/state_cmds.go`, `cmd/state_cmds_test.go`, `cmd/codex_build_finalize.go`, `cmd/codex_build_finalize_test.go`) do not overlap with `cmd/advance_phase.go` (188-02) or any midden/ratchet/autopilot files owned by the other plans in this phase.
- No `advancePhase()` dependency was needed for this plan's criterion 3 work (the state-mutate guard reuses the existing `enforceGuard`/`runGateCheck` machinery, not the new `advancePhase()` function 188-02 is building), so no TODO or stub was left behind.

---
*Phase: 188-one-truth-for-failures-and-advances*
*Completed: 2026-08-19*

## Self-Check: PASSED

- FOUND: cmd/state_cmds.go
- FOUND: cmd/codex_build_finalize.go
- FOUND: .planning/phases/188-one-truth-for-failures-and-advances/188-03-SUMMARY.md
- FOUND commit: 04e29047 (test, Task 1)
- FOUND commit: 570ef211 (feat, Task 1)
- FOUND commit: d41dad71 (test, Task 2)
- FOUND commit: cda3b07a (feat, Task 2)
