---
phase: 198-put-the-thrown-away-data-back-on-screen
plan: 02
subsystem: cli-display
tags: [go, cli, continue, verification, streaming, testing]

# Dependency graph
requires: []
provides:
  - "codexVerificationStep.Duration — measured wall-clock time for build/types/lint/tests, never estimated"
  - "emitVerificationStepStart/emitVerificationStepFinish — live start/finish lines for each verification check"
  - "verificationStepDisplayName — plain-English labels (Build/Types/Lint/Tests) for the owner"
affects: [198-01, 198-03, 198-04, 198-05, 198-06, 198-07, 198-08, 198-09]

# Actuals (#2632)
actuals:
  tokens: 5722
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Live progress hooked inside the single shared verification-step body (runVerificationStep), not in either continue lane's caller, so both lanes stream identically by construction"
    - "Named return + deferred finish-emit so a start/finish pair can never be dropped by a future early return"
    - "AST-based structural test (go/ast) instead of a text grep, so a comment cannot satisfy or break the guard"

key-files:
  created:
    - cmd/continue_live_progress_test.go
  modified:
    - cmd/codex_continue.go
    - cmd/golden_workflow_test.go
    - cmd/testdata/golden_continue.txt

key-decisions:
  - "Used emitVisualProgress (not emitVisualLine) for both start and finish lines, per PATTERNS.md's explicit direction, even though it inserts a blank line after each -- SEE-03 only requires append-only, not compact."
  - "Duration is measured with time.Since around the shell-out only inside runVerificationStep and left at zero on every return path that never ran a command, so a skipped check's live line shows no elapsed time at all rather than a fabricated 0s."
  - "Fixed a pre-existing test-isolation gap: currentStreamingCommand is a package-level var saveGlobals does not cover, so a prior test that ran a real cobra command could leave it pointing at a quiet-classified name and silently swallow emitVisualProgress in any later test that never touches rootCmd. setVisualOutputMode now saves/resets/restores it, mirroring TestCeremonyLevelGatesStreaming's existing guard."

requirements-completed: [SHOW-03]

coverage:
  - id: D1
    description: "Each verification check (build, types, lint, tests) prints a start line when it begins and a finish line when it ends, live, on both the in-process and wrapper/external continue lanes"
    requirement: "SHOW-03"
    verification:
      - kind: unit
        ref: "cmd/continue_live_progress_test.go#TestBothContinueLanesEmitLiveCheckLines"
        status: pass
    human_judgment: false
  - id: D2
    description: "A failed check's finish line carries the step's own one-line plain-English reason inline, on the same line as the fail mark"
    requirement: "SHOW-03"
    verification:
      - kind: unit
        ref: "cmd/continue_live_progress_test.go#TestFailedCheckLineCarriesItsReasonInline"
        status: pass
    human_judgment: false
  - id: D3
    description: "The elapsed time shown on a check's live finish line is measured (time.Since around the shell-out), never estimated, and a skipped check claims no duration at all"
    verification:
      - kind: unit
        ref: "cmd/continue_live_progress_test.go#TestVerificationStepDurationIsMeasured"
        status: pass
    human_judgment: false
  - id: D4
    description: "Live check lines are append-only scrollback in the one terminal -- no in-place repaint, no cursor-control escape sequences (SEE-03)"
    verification:
      - kind: unit
        ref: "cmd/continue_live_progress_test.go#TestLiveCheckLinesAreAppendOnly"
        status: pass
    human_judgment: false
  - id: D5
    description: "The two new emitters never write directly to stdout/stderr and always route through emitVisualProgress (the one streaming channel), and JSON output mode emits no progress lines at all"
    verification:
      - kind: unit
        ref: "cmd/continue_live_progress_test.go#TestLiveCheckLinesUseTheVisualWriter"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-08-29
status: complete
---

# Phase 198 Plan 02: Live Verification Progress Summary

**Build/types/lint/tests now each print a "Running X…" line when they start and a measured pass/fail/skip line when they finish, streamed identically on both continue lanes through one hook point.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-08-29T16:24:00Z
- **Completed:** 2026-08-29T16:49:22Z
- **Tasks:** 2 completed
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments
- Added a measured `Duration` field to `codexVerificationStep`, populated with `time.Since` around the shell-out only, left at zero on every early-return path that never actually ran a command.
- Added `emitVerificationStepStart`/`emitVerificationStepFinish`, hooked directly inside `runVerificationStep` (the single function both `runCodexContinueVerification` and `runCodexContinueVerificationSnapshot` call through `runDeterministicFloor`), so the ordinary fast path and the deeper review path stream the same two lines per check by construction, not by discipline.
- A failing check's finish line carries its own `Summary` text inline on the same line as the fail mark; a skipped check's finish line names why and claims no elapsed time.
- Locked the shape with two structural guards: an append-only/no-repaint test and an AST-based test proving the emitters never bypass the visual writer, plus a JSON-mode check that machine output stays untouched.
- Refreshed `testdata/golden_continue.txt` for the new lines and taught the golden normalizer to treat their measured duration the same way existing step/ceremony durations are already normalized for stability.

## Task Commits

Each task was committed atomically:

1. **Task 1: Measure each check and stream its start and finish line** — `5ac770fa` (feat)
2. **Task 2: Lock the streaming shape — append-only, one terminal, one writer** — `d63585a3` (test)
3. **Deviation fix: golden output + test-isolation leak** — `aa49aa51` (fix)

## Files Created/Modified
- `cmd/codex_continue.go` — `codexVerificationStep.Duration`, `emitVerificationStepStart`/`emitVerificationStepFinish`, `verificationStepDisplayName`; `runVerificationStep` rewritten with a named return + deferred finish emit so the start/finish pairing can't be lost by a future early return.
- `cmd/continue_live_progress_test.go` (new) — the five tests named in the plan, plus `setVisualOutputMode`/`assertLiveCheckLinesForAllChecks` helpers.
- `cmd/golden_workflow_test.go` — new `liveCheckLineDurationRe` normalizer for the live check lines' measured duration.
- `cmd/testdata/golden_continue.txt` — regenerated to include the new live lines.

## Decisions Made
- Chose `emitVisualProgress` over `emitVisualLine` for both emitters, per PATTERNS.md's explicit "Streaming exit point" guidance, even though it inserts a blank line after each call — SEE-03's requirement is append-only scrollback, not visual density, and Task 2's own AST test names `emitVisualProgress` as the required reach target.
- Used a named return (`step codexVerificationStep`) with a single `defer emitVerificationStepFinish(step)` in `runVerificationStep`, per the task's own instruction that the pairing "cannot be lost by a future early return" — every one of the five existing return paths (empty command required/optional, unresolvable command required/optional, and the main success/failure path) now assigns to the named return and falls through to one deferred emit, rather than five separate emit call sites.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `cmd/testdata/golden_continue.txt` no longer matched real `continue` output**
- **Found during:** post-Task-2 full-suite verification (`go test ./cmd/...`)
- **Issue:** `TestGoldenContinueVisualOutput` runs a real `continue` invocation and diffs it byte-for-byte against a checked-in golden file. The new live check lines are genuine, intended output (Task 1's whole point), so the golden file was now stale by design, not by accident.
- **Fix:** Regenerated the golden fixture with `-update-golden`, inspected the diff line-by-line to confirm it added *only* the four expected start/finish pairs in the expected place (immediately after the existing "Running deterministic verification…" line, before `── Dispatch ──`), and added `liveCheckLineDurationRe` to `normalizeForGolden` so the measured `(N.Ns)` duration on each finish line normalizes the same way `stepElapsedRe`/`ceremonyElapsedRe` already normalize other durations — otherwise the golden would flake on host timing variance.
- **Files modified:** `cmd/golden_workflow_test.go`, `cmd/testdata/golden_continue.txt`
- **Verification:** `go test ./cmd -run TestGoldenContinueVisualOutput -count=1` passes without `-update-golden`.
- **Committed in:** `aa49aa51`

**2. [Rule 3 - Blocking] New tests flaked only when run as part of the full `cmd` suite**
- **Found during:** running `go test ./cmd/...` after Task 2 (per this plan's project notes)
- **Issue:** `TestBothContinueLanesEmitLiveCheckLines`, `TestFailedCheckLineCarriesItsReasonInline`, `TestVerificationStepDurationIsMeasured`, and `TestLiveCheckLinesAreAppendOnly` all passed in isolation but failed in the full suite. Root cause: `currentStreamingCommand` (`cmd/codex_visuals.go`) is a package-level var that `saveGlobals` does not save/restore. A prior test elsewhere in the binary that executes a real cobra command through `rootCmd.Execute()` (`root.go` sets this var on every invocation) can leave it pointing at a quiet-classified command name (e.g. anything ending `-finalize`), which makes `streamingAllowedForCurrentCommand()` return false and silently no-ops every `emitVisualProgress` call in a later test that never touches `rootCmd` at all.
- **Fix:** `setVisualOutputMode` (the shared test helper all five new tests call) now also saves, resets to `""`, and restores `currentStreamingCommand`, mirroring the existing guard already present in `TestCeremonyLevelGatesStreaming` (`display_house_style_test.go`).
- **Files modified:** `cmd/continue_live_progress_test.go`
- **Verification:** `go test ./cmd/...` (full suite, ~254s) passes cleanly.
- **Committed in:** `aa49aa51`

---

**Total deviations:** 2 auto-fixed (1 bug in a stale test fixture, 1 pre-existing test-isolation gap that only affected the new tests). **Impact:** Both fixes were necessary to keep the full test suite green after landing genuinely new, intended behavior; no scope creep — the isolation fix touches only the new test file's own helper, not the leaking test elsewhere.

## Issues Encountered
None beyond the two deviations documented above.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- `codexVerificationStep.Duration` is now available for any later plan in this phase that needs the same measured figure (e.g. a closing verification summary that must read from the same source as these live lines, per D-03/Phase 196 D-01).
- `verificationStepDisplayName` is the canonical plain-English label mapping for build/types/lint/tests; reuse it rather than re-deriving check labels elsewhere in this phase.
- No blockers for subsequent plans in phase 198.

---
*Phase: 198-put-the-thrown-away-data-back-on-screen*
*Completed: 2026-08-29*

## Self-Check: PASSED

- All created/modified files confirmed present on disk: `cmd/codex_continue.go`, `cmd/continue_live_progress_test.go`, `cmd/golden_workflow_test.go`, `cmd/testdata/golden_continue.txt`.
- All three commit hashes confirmed in `git log`: `5ac770fa`, `d63585a3`, `aa49aa51`.
- Plan-level `<verification>` block re-run clean: `go build ./...`, `go vet ./cmd`, and the full named test set (`TestBothContinueLanesEmitLiveCheckLines`, `TestFailedCheckLineCarriesItsReasonInline`, `TestVerificationStepDurationIsMeasured`, `TestLiveCheckLinesAreAppendOnly`, `TestLiveCheckLinesUseTheVisualWriter`, `TestBothContinueLanesApplyTheSameFloor`, `TestDeterministicFloorIsTheOnlySourceOfAPass`) all pass.
- Full `go test ./cmd/...` suite (~254s) passes cleanly with no other regressions.
