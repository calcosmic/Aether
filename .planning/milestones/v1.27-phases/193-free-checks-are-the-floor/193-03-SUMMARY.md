---
phase: 193-free-checks-are-the-floor
plan: 03
subsystem: verification
tags: [go, continue-pipeline, deterministic-floor, cli-flags, testing]

# Dependency graph
requires:
  - phase: 193-01
    provides: "runDeterministicFloor(ctx, root, phase, manifest, watcher, timeout) -- the shared deterministic-floor body both continue lanes call; continueWatcherDecision -- the single dispatch decision point"
provides:
  - "TestDeterministicChecksCannotBeSkipped, TestExecutedCheckSetIsInvariantAcrossDepthAndProposal, TestFailingCheckStillBlocksOnEverySkipPath (cmd/floor_unskippable_test.go) -- FLOOR-01 as a runnable command over ten named skip-path rows on both continue lanes"
  - "TestNoContinueFlagClaimsToSkipAChecked -- a generic guard against any future continue flag whose help text claims a check can be turned off"
  - "Corrected --skip-watchers help text: states plainly that only AI reviewer workers are skipped and the program's own checks always run"
affects: [193-04, 193-05]

# Actuals (#2632)
actuals:
  tokens: 5500
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Skip-path row table with a configure/verifyConditionIsReal pair per row: configure drives the real production condition (invoker override, manifest shape, state counter), verifyConditionIsReal independently proves the row's premise through the real production function rather than stubbing continueWatcherDecision's return value."
    - "Flag-text guard by pairing, not by name: TestNoContinueFlagClaimsToSkipAChecked scans usage strings for a skip/disable word paired with a check word, with a small named-and-reasoned allowance list, so a future flag inherits the guard automatically."

key-files:
  created:
    - cmd/floor_unskippable_test.go
  modified:
    - cmd/codex_workflow_cmds.go

key-decisions:
  - "No implementation change was needed in cmd/codex_continue.go for Task 1. runDeterministicFloor (193-01) already runs unconditionally at the top of both runCodexContinueVerification and runCodexContinueVerificationSnapshot, before continueWatcherDecision (or any depth/proposal value, which those functions do not even accept as parameters) is consulted -- so all ten skip-path rows already passed against the pre-change code. The plan explicitly allows this outcome (\"If a skip path turns out to already satisfy the test, say so explicitly in the SUMMARY rather than removing the row\"); the tests are kept as regression guards, not because they caught a bug today."
  - "Light depth (not standard) is used to drive the \"caste proposal naming no reviewer\" row's verifyConditionIsReal. At standard depth, cmd/caste_relevance.go's isAlwaysRequired forces \"probe\" back into the continue review specs whenever queenPhaseProducesTestableCode(phase) is true -- which it is here, because the check scans this TEST BINARY's own workspace (the real Aether repo, full of *.go files) for program code, independent of the fixture phase's own doc-only wording. That is the exact probe/auditor/gatekeeper double-dispatch-with-no-proposal gap 193-02 already flagged as out of scope and recorded in .planning/WINDOWS.md (kind: unmet-truth, scoped to Phase 194). Using light depth sidesteps it correctly (isAlwaysRequired only restores \"watcher\" at light depth, which review-spec building always excludes anyway) rather than papering over it inside this plan's declared scope."
  - "Six of the ten skip-path rows (skip-watchers, light, heavy, and the three explicit --verification-depth values, plus the caste-proposal row) are called with skipWatchers=true. Depth and caste-proposal values are never passed to runCodexContinueVerification/runCodexContinueVerificationSnapshot at all -- their structural absence from those function signatures IS the proof those parameters cannot reach the floor. Each such row's verifyConditionIsReal instead drives the real, independent production function (resolveVerificationDepth, queenContinueReviewSpecsWithJudgement) that the row's condition actually governs, so the row is not vacuous. The remaining three rows (unavailable provider, host-boundary, consecutive-failure) are called with skipWatchers=false specifically so continueWatcherDecision's own real branch for each condition fires, rather than being short-circuited by skipWatchers itself -- matching the read_first guidance not to stub the decision function."

requirements-completed: []  # FLOOR-01 is also declared by 193-05; requirements.ready-ids confirms it is not yet ready to mark complete from this plan alone.

coverage:
  - id: D1
    description: "Every reviewer-skip path that exists today -- the skip-watchers flag, light depth, heavy depth, an explicit verification depth of light/standard/heavy, a caste proposal naming no reviewer, an unavailable worker provider, the host-boundary skip, and the consecutive-failure auto-skip -- leaves build, types, lint and tests all executed (not skipped, carrying a command, producing an outcome), claimed-files-exist checked, and criterion evidence evaluated, on both continue lanes (FLOOR-01)."
    requirement: FLOOR-01
    verification:
      - kind: unit
        ref: "cmd/floor_unskippable_test.go#TestDeterministicChecksCannotBeSkipped (10 named subtests)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The multiset of executed check names is byte-identical across every depth flag, review policy and caste proposal combination in the table, on both lanes independently -- the proportion assertion that a future flag cannot silently reduce what is checked."
    requirement: FLOOR-01
    verification:
      - kind: unit
        ref: "cmd/floor_unskippable_test.go#TestExecutedCheckSetIsInvariantAcrossDepthAndProposal"
        status: pass
    human_judgment: false
  - id: D3
    description: "A floor that runs but is then ignored is caught: every skip-path row still reports ChecksPassed=false and non-empty blocking issues, on both lanes, when the tests command genuinely fails."
    requirement: FLOOR-01
    verification:
      - kind: unit
        ref: "cmd/floor_unskippable_test.go#TestFailingCheckStillBlocksOnEverySkipPath (10 named subtests)"
        status: pass
    human_judgment: false
  - id: D4
    description: "No documented continue flag claims a check can be turned off; the corrected --skip-watchers help text states plainly that only AI reviewer workers are skipped and the program's own checks always run, in plain English with no repo vocabulary beyond the flag's own name."
    requirement: FLOOR-01
    verification:
      - kind: unit
        ref: "cmd/floor_unskippable_test.go#TestNoContinueFlagClaimsToSkipAChecked"
        status: pass
      - kind: other
        ref: "/tmp/aether-193 continue --help | grep -c 'rely on verification commands only' -> 0; --skip-watchers line present exactly once"
        status: pass
    human_judgment: false
  - id: D5
    description: "No regression introduced across the whole cmd package by this plan's test file or the flag-text change."
    verification:
      - kind: unit
        ref: "go test ./cmd -count=1 (full package, 0 failures, 412.1s)"
        status: pass
      - kind: unit
        ref: "go test ./cmd -race -count=1 -run <the four tests this plan added> (0 failures, 6.4s)"
        status: pass
      - kind: other
        ref: "go build ./... && go build -o /tmp/aether-193 ./cmd/aether && go vet ./..."
        status: pass
    human_judgment: false

# Metrics
duration: 23min
completed: 2026-08-22
status: complete
---

# Phase 193 Plan 03: Unskippable Floor and Corrected Flag Text Summary

**FLOOR-01 is now a runnable command: ten named subtests walk every reviewer-skip path that exists today across both continue lanes and fail by name if any one of them leaves build, types, lint, tests, claimed-files-exist, or criterion evidence unrun -- all ten already pass against the pre-existing code from 193-01, so this plan shipped as pure regression-guard tests plus one corrected flag description, with no production logic change needed.**

## Performance

- **Duration:** ~23 min
- **Started:** 2026-08-22T14:13:52Z (immediately following 193-02's completion)
- **Completed:** 2026-08-22T14:37:17Z
- **Tasks:** 2 completed
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- `TestDeterministicChecksCannotBeSkipped` walks ten named skip-path rows (skip-watchers, light depth, heavy depth, explicit verification depth of light/standard/heavy, a caste proposal naming no reviewer, an unavailable worker provider, the host-boundary skip, the consecutive-failure auto-skip) over a fixture whose four verification commands all resolve, and asserts on BOTH continue lanes (in-process `runCodexContinueVerification` and wrapper `runCodexContinueVerificationSnapshot`) that build/types/lint/tests all ran, claims were checked, and criterion evidence was evaluated. All ten already pass — no implementation change was needed.
- `TestExecutedCheckSetIsInvariantAcrossDepthAndProposal` proves the executed-check multiset is byte-identical across every row, on each lane independently — the proportion assertion that survives even if a future flag is named something else entirely.
- `TestFailingCheckStillBlocksOnEverySkipPath` re-runs the same table with the tests command genuinely failing and confirms every row still reports `ChecksPassed=false` with a non-empty blocking-issue list, on both lanes — a floor that runs but is silently ignored is the same defect as one that never ran.
- The `--skip-watchers` help text no longer says the run "relies on verification commands only" (implying the program's checks are an optional fallback); it now says the flag skips AI reviewer workers only and the program's own checks (build, types, lint, tests) always run either way — checked with `go build -o /tmp/aether-193 ./cmd/aether && /tmp/aether-193 continue --help`.
- `TestNoContinueFlagClaimsToSkipAChecked` walks every registered `continueCmd` flag and fails if a future flag's help text pairs a skip/disable word with a check word, unless the flag is named in a short, one-line-reasoned allowance list — confirmed RED against the pre-fix text (temporarily emptying the allowance list reproduced the failure), then GREEN once the corrected `--skip-watchers` text was allow-listed with the reason that its own wording deliberately says checks are NOT skipped.
- While designing the "caste proposal naming no reviewer" row, discovered the same probe/auditor/gatekeeper double-dispatch gap 193-02 had already flagged and recorded in `.planning/WINDOWS.md` (kind: `unmet-truth`, scoped to Phase 194) — confirmed it live via `queenContinueReviewSpecsWithJudgement` at standard depth, and worked around it correctly (light depth) inside this plan's own declared scope rather than touching the required-caste floor.

## Task Commits

Each task was committed atomically:

1. **Task 1: TestDeterministicChecksCannotBeSkipped walks every skip path** — `15b9f073` (test)
2. **Task 2: Correct the owner-facing text, and guard it** — `20e372ed` (fix)

**Plan metadata:** (this commit, following)

_Note: Task 1 is a single `test(...)` commit with no following `feat(...)` commit — see "TDD Gate Compliance" below for why that is the correct outcome here, not a missed gate._

## Files Created/Modified

- `cmd/floor_unskippable_test.go` (new) — `floorUnskippableFixture`, the `floorSkipPathRow` table (`floorSkipPathRows`, `floorSkipPathState`), `assertFloorFullyExecuted`, `executedCheckNames`, `TestDeterministicChecksCannotBeSkipped`, `TestExecutedCheckSetIsInvariantAcrossDepthAndProposal`, `TestFailingCheckStillBlocksOnEverySkipPath`, `continueFlagCheckSkipPairingAllowance`, `TestNoContinueFlagClaimsToSkipAChecked`.
- `cmd/codex_workflow_cmds.go` — one line: the `--skip-watchers` flag's registered help string.

## Decisions Made

See `key-decisions` in the frontmatter for the three load-bearing ones (no implementation change needed and why, the light-depth workaround for the pre-existing probe gap, and the skipWatchers value chosen per row).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug in my own first draft] "Caste proposal naming no reviewer" row initially used standard depth, which the pre-existing (out-of-scope) probe/auditor/gatekeeper required-caste gap makes fail for an unrelated reason**
- **Found during:** Task 1, first RED/pre-change run of the new tests (`go test ./cmd -run 'TestDeterministicChecksCannotBeSkipped|...' -v`).
- **Issue:** `queenContinueReviewSpecsWithJudgement(phase, colony.VerificationDepthStandard, []string{"builder"}, "team asked for a builder only")` still returned a `probe` spec, because `cmd/caste_relevance.go`'s `isAlwaysRequired` forces `probe` back in at standard depth whenever `queenPhaseProducesTestableCode(phase)` is true — and that check scans the *test binary's own workspace* (this repo, full of Go source) for program code, independent of the fixture phase's doc-only wording. This is exactly the gap 193-02 already discovered and recorded in `.planning/WINDOWS.md` (kind: `unmet-truth`, scoped to Phase 194) — not a new defect, and out of this plan's declared scope to fix.
- **Fix:** Changed the row's `verifyConditionIsReal` to call `queenContinueReviewSpecsWithJudgement` at light depth instead of standard. At light depth `isAlwaysRequired` only ever restores `watcher`, which review-spec building always filters out regardless — so the row's premise ("a proposal naming no reviewer produces no review specs") holds for the real reason the row is testing (the proposal), not by accident of workspace state.
- **Files modified:** `cmd/floor_unskippable_test.go` (test-only, before any implementation code was touched).
- **Verification:** `go test ./cmd -run 'TestDeterministicChecksCannotBeSkipped$' -count=1 -v` — all ten subtests PASS.
- **Committed in:** `15b9f073` (the corrected version is what was committed; the standard-depth draft was never committed).

---

**Total deviations:** 1 (a self-caught issue in my own first draft, fixed before the RED/GREEN commit — not a production-code deviation).
**Impact on plan:** None on scope or files_modified. No production code outside the plan's declared files was touched; the fix stayed entirely inside the new test file's own row logic.

## TDD Gate Compliance

Task 1 was marked `tdd="true"`. Per the plan's own explicit text — *"If a skip path turns out to already satisfy the test, say so explicitly in the SUMMARY rather than removing the row: the row is the guard against a future regression, and its value does not depend on it having caught something today"* — the RED confirmation (all three tests run against the pre-change code, recorded in "Issues Encountered" below) came back green for all ten rows, and the expected shape of a follow-up fix ("runDeterministicFloor invoked unconditionally... before continueWatcherDecision is consulted") was already true from 193-01. No `feat(193-03)` GREEN commit follows the `test(193-03)` RED commit for Task 1 because there was nothing to make green — this is the plan's explicitly sanctioned outcome, not a missed gate.

Task 2 was not marked `tdd="true"` in the plan, but was implemented test-first anyway for the same honesty discipline: `TestNoContinueFlagClaimsToSkipAChecked` was confirmed to fail against the original `--skip-watchers` text (temporarily emptying the allowance list reproduced: `--skip-watchers help text pairs a skip/disable word with the check word "verification" ...`), then the flag text was corrected and the allowance restored, and the test passes. Both changes landed in the single `fix(193-03)` commit since they are one indivisible change (the guard would otherwise immediately flag the very text it was written to allow).

## Issues Encountered

None blocking. For the record — the plan's fail-then-pass discipline for Task 1's three tests:

**Pre-change (before any code was touched, testing against the code as left by 193-02):**
```
go test ./cmd -run 'TestDeterministicChecksCannotBeSkipped|TestExecutedCheckSetIsInvariantAcrossDepthAndProposal|TestFailingCheckStillBlocksOnEverySkipPath' -count=1 -v
```
First draft (standard-depth caste-proposal row): 1 subtest failure — `a_caste_proposal_naming_no_reviewer` under `TestDeterministicChecksCannotBeSkipped`, for the pre-existing, out-of-scope reason above. All other 9 rows across all three tests: PASS.
After switching that row to light depth (still no implementation change, test-file-only fix): all three tests, all ten rows each: PASS — `ok github.com/calcosmic/Aether/cmd 2.121s`.

**Post-change (after Task 2's flag-text fix, full targeted run):**
```
go build ./cmd/aether && go vet ./cmd && go test ./cmd -run 'TestDeterministicChecksCannotBeSkipped|TestExecutedCheckSetIsInvariantAcrossDepthAndProposal|TestFailingCheckStillBlocksOnEverySkipPath|TestNoContinueFlagClaimsToSkipAChecked' -count=1 -v
```
All 31 subtests (10+10+10+1) PASS.

**Task 2's own RED/GREEN (not required by the plan, done for discipline):**
Temporarily emptied `continueFlagCheckSkipPairingAllowance` and ran `TestNoContinueFlagClaimsToSkipAChecked` against the unfixed `--skip-watchers` text: FAIL — `--skip-watchers help text pairs a skip/disable word with the check word "verification", implying a check can be turned off: "Skip watcher agent spawn; rely on verification commands only"`. Restored the allowance entry, applied the real fix to `cmd/codex_workflow_cmds.go`, reran: PASS.

`go test ./cmd -count=1` (the whole ~5,900-test package, not just this plan's `-run` filters): 0 failures, `ok github.com/calcosmic/Aether/cmd 412.098s`. The run's wall-clock time was dominated by `TestPackedNPMReleaseCandidateContract`'s real `npm install` of a packed tarball (a known slow, pre-existing test — see the deferred item in STATE.md's Deferred Items table, `191.1-01`), confirmed by process inspection during the run; not caused by this plan's changes.

`go test ./cmd -race -count=1` scoped to the four tests this plan added (`TestDeterministicChecksCannotBeSkipped`, `TestExecutedCheckSetIsInvariantAcrossDepthAndProposal`, `TestFailingCheckStillBlocksOnEverySkipPath`, `TestNoContinueFlagClaimsToSkipAChecked`): 0 failures, `ok github.com/calcosmic/Aether/cmd 6.417s` — following 193-01/193-02's own established precedent of scoping race runs rather than racing the whole ~5,900-test package.

`go build ./...` (whole repo, not just `cmd/aether`) and `go vet ./...` (whole repo): both exit 0.

## User Setup Required

None — no external service configuration required. This plan is entirely local Go code and tests.

## Next Phase Readiness

Ready for 193-04 (wave 3) and 193-05. `cmd/codex_continue.go` was in this plan's declared `files_modified` but ended up untouched — 193-01 already did the work this plan's test walks. `.planning/WINDOWS.md`'s existing `unmet-truth` entry (probe/auditor/gatekeeper double-dispatch with no explicit Queen proposal) was independently re-confirmed while designing this plan's caste-proposal row; no new entry was needed, and Phase 194 remains the right owner for closing it.

---
*Phase: 193-free-checks-are-the-floor*
*Completed: 2026-08-22*

## Self-Check: PASSED

All created/modified files (`cmd/floor_unskippable_test.go`, `cmd/codex_workflow_cmds.go`, this SUMMARY) and both task commits (`15b9f073`, `20e372ed`) verified present.
