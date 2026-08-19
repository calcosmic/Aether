---
phase: 188-one-truth-for-failures-and-advances
plan: 05
subsystem: infra
tags: [go, autopilot, midden, observability, retry]

requires:
  - phase: 188-01
    provides: "cmd/midden_shared.go: loadMiddenFile (canonical read) / appendMiddenEntry (canonical, atomic write) -- the one midden path every reader and this plan's new writer converge on"
provides:
  - "cmd/autopilot_retry_record.go: recordAutopilotRetryExhaustion(s, phaseID, firstErr, secondErr) -- writes one midden entry (category autopilot_retry_exhausted, source \"aether run\") naming the phase and both attempts' error text"
  - "runCompatibilityAutopilot's second build-retry failure branch now calls recordAutopilotRetryExhaustion, best-effort, before pausing -- closing ROADMAP criterion 5's gap (the autopilot's own retry loop previously discarded both failures silently)"
  - "cmd/autopilot_retry_record_test.go: TestRecordAutopilotRetryExhaustion (content proof across 4 subtests, including readability through memory-health's real consumer path) and TestAutopilotRetryExhaustionCallSiteIsWired (structural call-site regression guard, proven to fail when the call is removed)"
affects: [autopilot, memory-health, colony-prime, immune]

tech-stack:
  added: []
  patterns:
    - "Directly-testable record-writer function extracted from an inline give-up branch, mirroring cmd/recovery_orchestrator.go's structured-record shape at a smaller scale, writing through 188-01's shared appendMiddenEntry rather than a bespoke file"

key-files:
  created:
    - cmd/autopilot_retry_record.go
    - cmd/autopilot_retry_record_test.go
  modified:
    - cmd/compatibility_cmds.go

key-decisions:
  - "Put the new function and its tests in dedicated new files (cmd/autopilot_retry_record.go / _test.go) rather than growing the already-929-line cmd/compatibility_cmds.go further -- explicitly permitted by the plan's own <action> discretion"
  - "Added a fourth content-proof subtest beyond the plan's literal Task 2 wording: reading the written record back through loadMemoryHealthSummary (a real criterion-1 consumer), not just loadMiddenFile, to prove readability through an actual consumer path end-to-end, not merely the shared helper the writer and every reader happen to share"

patterns-established:
  - "Best-effort side-record, pause regardless: a record-writer's own error is deliberately discarded at its one call site so a broken failure-log can never block the safety behavior (the pause) it documents"

requirements-completed: []

duration: ~18min
completed: 2026-08-19
---

# Phase 188 Plan 05: Autopilot Retry-Exhaustion Record Summary

**Autopilot's own build-retry loop now writes a midden entry naming the failed phase and both attempts' error text before pausing, via 188-01's shared writer, proven readable through memory-health's real consumer path and proven to break the moment the call site regresses.**

## Performance

- **Duration:** ~18 min (commit timestamps: base `a5606bd8` at 22:04:44Z, GREEN commit at 22:15:15Z, full-suite verification finished 22:21:55Z)
- **Started:** 2026-08-19T22:04:44Z (worktree base)
- **Completed:** 2026-08-19T22:21:55Z
- **Tasks:** 2 (Task 1 was TDD: RED + GREEN; Task 2's own test was authored alongside Task 1's RED commit and verified separately per its own acceptance criteria)
- **Files modified:** 1 modified, 2 created

## Accomplishments

- `recordAutopilotRetryExhaustion` (new `cmd/autopilot_retry_record.go`): a small, directly unit-testable function that builds a plain-language message naming the phase number and both build attempts' error text, then writes it through 188-01's `appendMiddenEntry` -- the same canonical writer every one of criterion 1's five now-unified readers reads from
- Wired into `runCompatibilityAutopilot`'s second build-retry failure branch (`cmd/compatibility_cmds.go`), immediately before the existing pause call, on a strict best-effort basis: the record write's own error is discarded at the call site so it can never block the pause itself (T-188-15)
- Proved the record's *content* is correct with four subtests: distinct-error text, identical-error collapsing (still names both attempts happened), write-failure propagation (not swallowed), and -- the strongest proof -- reading the written record back through `loadMemoryHealthSummary`, a real production consumer from criterion 1's unification, and confirming `RecentFailures` increases and `LastFailure` populates
- Proved the *wiring* is real, not orphaned, with a structural test (`TestAutopilotRetryExhaustionCallSiteIsWired`) that parses `compatibility_cmds.go` with `go/parser`, locates `runCompatibilityAutopilot`'s function body by byte range, and confirms the call falls strictly between the retry's own build attempt and the pause/return that follow it -- then actually removed the call, watched the test fail, and restored it (see the demonstration below)

## Task Commits

Each task was committed atomically:

1. **Task 1a (RED): failing test for `recordAutopilotRetryExhaustion`** -- `f05b96f8` (test)
2. **Task 1b (GREEN): the record-writer, wired into the second-failure branch** -- `2b319abc` (feat)

**Task 2** produced no additional commit: its required test (`TestAutopilotRetryExhaustionCallSiteIsWired`) was authored together with Task 1's Tier 1 test in the same RED commit (`f05b96f8`), since both live in one new test file and Task 1's own TDD cycle needed the whole file present to demonstrate a build-failure RED. Task 2's distinguishing work -- the mandatory fail-then-pass demonstration (temporarily removing the call, confirming the guard fails, restoring it) -- was performed after `2b319abc` and left the working tree byte-identical to that commit (`git diff --stat` empty), so there is nothing further to commit; the observed failure is quoted below.

**Plan metadata:** (this commit, made after this SUMMARY)

## The demonstration the Definition of Done requires

Two separate real fail-then-pass demonstrations were run, matching this plan's dependency note that this record doubles as end-to-end proof that 188-01's midden unification actually closed the loop.

**1. RED, before `recordAutopilotRetryExhaustion` existed** (`go vet ./cmd/`, run before any production code was written):

```
# github.com/calcosmic/Aether/cmd
# [github.com/calcosmic/Aether/cmd]
vet: cmd/autopilot_retry_record_test.go:36:13: undefined: recordAutopilotRetryExhaustion
```

This is the same "RED confirmed as a build failure" idiom 188-01 used for its own TDD task -- a compile failure is a legitimate RED when the behavior under test does not exist yet.

**2. Task 2's required regression demonstration**, after `recordAutopilotRetryExhaustion` existed and was wired in: the call site (`_ = recordAutopilotRetryExhaustion(store, phase.ID, firstErr, err)`) was temporarily commented out of `runCompatibilityAutopilot`, and `TestAutopilotRetryExhaustionCallSiteIsWired` was rerun:

```
=== RUN   TestAutopilotRetryExhaustionCallSiteIsWired
    autopilot_retry_record_test.go:207: recordAutopilotRetryExhaustion( does not appear after the retry's
    runCodexBuildWithOptions( call inside runCompatibilityAutopilot -- the retry-exhaustion record is not
    wired into the second-failure branch
--- FAIL: TestAutopilotRetryExhaustionCallSiteIsWired (0.00s)
FAIL
FAIL	github.com/calcosmic/Aether/cmd	0.746s
FAIL
```

The call was then restored verbatim; `git diff --stat` against the GREEN commit came back empty (byte-identical), and the full pair re-ran clean:

```
--- PASS: TestRecordAutopilotRetryExhaustion (0.01s)
    --- PASS: .../distinct_errors:_message_names_the_phase_and_both_attempts'_text (0.00s)
    --- PASS: .../identical_errors_on_both_attempts:_still_one_entry,_still_says_two_attempts_happened (0.00s)
    --- PASS: .../a_write_failure_is_returned,_not_swallowed (0.00s)
    --- PASS: .../the_record_is_readable_through_memory-health's_real_consumer_path,_closing_188-01's_loop_end-to-end (0.00s)
--- PASS: TestAutopilotRetryExhaustionCallSiteIsWired (0.00s)
PASS
ok  	github.com/calcosmic/Aether/cmd	0.641s
```

Both demonstrations were actually run, not asserted from reading the code -- matching this repo's own `CLAUDE.md` Definition of Done and the explicit instruction that a test asserting a file exists, or a category string is present, does not prove either half of the ROADMAP criterion ("readable" and "names what was tried") on its own.

## Files Created/Modified

- `cmd/autopilot_retry_record.go` -- new: `recordAutopilotRetryExhaustion` and its small `autopilotRetryErrorText` nil-safety helper
- `cmd/autopilot_retry_record_test.go` -- new: `TestRecordAutopilotRetryExhaustion` (4 subtests) and `TestAutopilotRetryExhaustionCallSiteIsWired`
- `cmd/compatibility_cmds.go` -- `runCompatibilityAutopilot`'s second build-retry failure branch now captures `firstErr := err` right after the first attempt's own `if err != nil` check (before the retry overwrites `err`), and calls `recordAutopilotRetryExhaustion(store, phase.ID, firstErr, err)` immediately before the existing `syncRunAutopilotState(state, opts, "paused", "")` / `return nil, err`. Net diff: 6 lines added, 0 removed.

## Decisions Made

- Canonical midden category for this record is `autopilot_retry_exhausted`, source `"aether run"` -- matches D-13 exactly, no deviation
- New function and its tests live in dedicated new files rather than growing `compatibility_cmds.go` (929 lines before this change) -- explicitly one of the two options the plan's own `<action>` text names; documented here per that same text's request ("note in the plan summary which tier was actually achieved" spirit)
- Did not attempt a Tier 3 (genuine forced-double-build-failure, end-to-end `aether run` execution) test. D-14 explicitly names this as lower-value than proving the record-writer's content and call-site wiring directly, and confirms no reliable deterministic double-failure fixture exists in this codebase today; nothing found while implementing this plan changed that. Tiers 1 and 2 (content + structural wiring) were achieved, both with real, actually-run fail-then-pass demonstrations
- Added a fourth Tier 1 subtest not literally spelled out in the plan's Task 2 text: reading the record back through `loadMemoryHealthSummary` (a real production consumer, not just the shared `loadMiddenFile` helper) to make the "readable through the criterion-1 unified readers, closing 188-01's loop end-to-end" requirement concrete and directly verified, per this execution's explicit proof requirement
- The Tier 2 structural guard checks not just "the call appears somewhere before the return" (the plan's stated minimum) but that it appears strictly between the retry's own build call and the pause/return that follow -- a tighter bound that a call placed anywhere earlier in the function could not falsely satisfy

## Deviations from Plan

### Auto-fixed / adjusted from the plan's literal text

**1. [Rule 1-adjacent -- acceptance-criteria wording followed the plan's own alternative, not its literal example] Task 1's acceptance criteria assumed same-file placement**

- **Found during:** Task 1, before implementing
- **Issue:** Task 1's acceptance criteria says grepping `cmd/compatibility_cmds.go` for `recordAutopilotRetryExhaustion(` should return "both the definition and exactly one call site" -- this phrasing assumes the function is defined in that same file, one of the two options the same task's `<action>` text explicitly offers ("or a new small file `cmd/autopilot_retry_record.go`... either location is fine"). Choosing the new-file option (to avoid growing a 929-line file further, per that text's own stated rationale) means grepping `compatibility_cmds.go` alone returns only the one call site, not the definition too.
- **Resolution:** Verified both halves explicitly and separately instead: `grep -n "recordAutopilotRetryExhaustion(" cmd/compatibility_cmds.go` returns exactly one line (the call site, line 471); `grep -rn "func recordAutopilotRetryExhaustion" cmd/*.go` returns exactly one line (the definition, in `cmd/autopilot_retry_record.go`). Same guarantee the acceptance criterion wants (one definition, one call site, no orphaning), verified across the two files the plan's own discretion allows rather than assuming they are the same file.
- **Files affected:** none (verification-only; both grep commands were actually run, see this plan's own execution transcript)
- **Committed in:** `2b319abc` (Task 1 GREEN commit)

---

**Total deviations:** 1 (wording-only, caused by exercising discretion the plan itself explicitly grants; no behavior, scope, or file-set change)
**Impact on plan:** None on scope or correctness. The underlying goal (one function, one call site, both verifiable) is delivered exactly as required; only which literal grep command proves it changed, because the plan itself offered two valid file layouts and this execution picked the one the plan's own rationale favored.

## Issues Encountered

None beyond the deviation above, which was identified and resolved before it could cause a false verification result.

## Next Phase Readiness

- ROADMAP criterion 5 ("an exhausted retry loop leaves a readable record of what was tried") is closed for the autopilot's own build-retry loop; the worker-level recovery orchestrator (a separate system) already satisfied this criterion per D-12 and needed no change
- This plan's own test suite doubles as a second, independent end-to-end confirmation that 188-01's midden unification works: a record written by `recordAutopilotRetryExhaustion` was read back through `loadMiddenFile` directly and through `loadMemoryHealthSummary` (a real consumer), both succeeding
- Full-repo `go build ./...`, `go vet ./...`, and `go test ./... -count=1` all clean -- every package passes: `cmd` (314.086s), `pkg/agent` (8.019s), `pkg/agent/curation` (0.409s), `pkg/cache` (0.703s), `pkg/codegraph` (1.015s), `pkg/codex` (23.719s), `pkg/colony` (2.661s), `pkg/downloader` (3.284s), `pkg/events` (1.500s), `pkg/exchange` (2.402s), `pkg/graph` (3.161s), `pkg/learn` (3.758s), `pkg/llm` (3.700s), `pkg/memory` (3.606s), `pkg/smoke` (2.534s), `pkg/storage` (2.894s), `pkg/terminal` (2.649s), `pkg/trace` (2.520s). Exit code 0.
- `gofmt -l` on all three touched/created files returns nothing (already correctly formatted)
- No blockers for any other plan in this phase. This plan's only dependency (188-01) was already merged into this worktree's base before work started; no other plan in this phase depends on 188-05's output per `188-CONTEXT.md`'s Success Criteria Coverage table

## Known Stubs

None.

## Self-Check

Files:
- FOUND: `cmd/autopilot_retry_record.go`
- FOUND: `cmd/autopilot_retry_record_test.go`
- FOUND: `.planning/phases/188-one-truth-for-failures-and-advances/188-05-SUMMARY.md`

Commits (`git log --oneline --all | grep <hash>`):
- FOUND: `f05b96f8` (test: RED)
- FOUND: `2b319abc` (feat: GREEN)

Repo-wide verification:
- `go build ./...`: clean, zero output
- `go vet ./...`: clean, zero output
- `go test ./... -count=1`: **ok**, every package passes (see full list above). Exit code 0.
- `gofmt -l` on all touched files: clean, zero output

## Self-Check: PASSED

---
*Phase: 188-one-truth-for-failures-and-advances*
*Completed: 2026-08-19*
