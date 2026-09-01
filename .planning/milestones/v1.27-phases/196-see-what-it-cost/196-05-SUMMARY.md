---
phase: 196-see-what-it-cost
plan: 05
subsystem: infra
tags: [spend-ledger, token-usage, opencode, claude-code, salvage, concurrency, go, tdd]

requires:
  - phase: 196-01
    provides: the spend ledger, spendRow.JobName, the closed dispatch status vocabulary
  - phase: 196-02
    provides: one authoritative token type, the no-length-derivation ratchet, an unreported worker carrying no figure
  - phase: 196-03
    provides: parseClaudeTranscriptUsage, evalSpendPathSymlinks, the deduplicated transcript reader
provides:
  - The OpenCode session-store reader, salvaged with its process-aborting cache removed and its accumulation path proven on a two-message fixture
  - validateSpendContainedPath — one containment boundary for every path the spend subsystem opens, used by both platform validators
  - resolveWrapperWorkerUsage — one per-run usage resolver serving the Claude, OpenCode and directly-spawned paths
  - writeSpendRowsForRun — the writer that turns finished dispatches into ledger rows
  - A build finalize path that leaves per-worker rows on disk, making both platform readers reachable from a real dispatch
affects: [196-06, 196-07, 196-08]

actuals:
  tokens: 25080
  tasks: 3
  commits: 7

tech-stack:
  added: []
  patterns:
    - "A concurrency rule stated twice: once as a race-detector proof and once as a structural scan, so it also fails under a plain `go test`"
    - "A security boundary is extracted to one helper and the singleness itself is asserted by an AST count, not by a comment"
    - "Reported-versus-absent is carried by an explicit flag and by the source tag, never inferred from whether a number is zero"
    - "Ledger assertions read the saved file's own bytes back off disk rather than inspecting the writer's return value"

key-files:
  created:
    - cmd/wrapper_usage_opencode.go
    - cmd/wrapper_usage_opencode_test.go
    - cmd/wrapper_usage_resolve.go
    - cmd/wrapper_usage_resolve_test.go
    - cmd/spend_writer.go
    - cmd/spend_writer_test.go
    - cmd/testdata/spend/opencode/README.md
    - cmd/testdata/spend/opencode/project/prj_fixture.json
    - cmd/testdata/spend/opencode/session/prj_fixture/ses_child_a.json
    - cmd/testdata/spend/opencode/session/prj_fixture/ses_child_b.json
    - cmd/testdata/spend/opencode/session/prj_fixture/ses_stale.json
    - cmd/testdata/spend/opencode/message/ses_child_a/msg_1.json
    - cmd/testdata/spend/opencode/message/ses_child_a/msg_2.json
    - cmd/testdata/spend/opencode/message/ses_child_b/msg_1.json
    - cmd/testdata/spend/opencode/message/ses_stale/msg_1.json
  modified:
    - cmd/spend_session_capture.go
    - cmd/codex_build_finalize.go

key-decisions:
  - "The worker-name pattern cache was deleted outright rather than wrapped in a mutex, as the salvage assessment preferred: it was also unbounded, and compiling a handful of short quoted names per run costs nothing"
  - "A worker is accounted under codexBuildDispatch.Name (the deterministic 'Mason-67'), not AgentName ('aether-builder'), because several workers in one build share an agent definition and the platform's own session titles carry the deterministic name"
  - "The orchestrating session's own turns are returned separately from the worker list rather than as a worker row or discarded — 196-03 left this choice to this plan"
  - "'Reported' keys on the usage source tag, never on a number: a worker that genuinely billed zero and a worker whose tool said nothing both present as zero, and D-01 renders those two differently"
  - "A dispatch whose status is outside the ledger vocabulary is dropped with a named note rather than refusing the whole ledger, because saveSpendLedger is fail-closed for the entire file"
  - "The provider's own attached measurement outranks a session-artifact read for the same worker on every path"

patterns-established:
  - "Structural concurrency guard: a package-level mutable var in a cost-path file is refused by name, so the bug class cannot return even when tests run without -race"
  - "Containment singleness asserted by counting the functions that call filepath.Rel and mention '..' across the spend files, then naming the one that is allowed to"

requirements-completed: [COST-02, COST-03, COST-05]

coverage:
  - id: D1
    description: "The OpenCode reader is in the tree without the crash it carried: nothing in the cost path can abort the process on a concurrent map write"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "cmd/wrapper_usage_opencode_test.go#TestOpenCodeWorkerNameMatchingIsConcurrencySafe"
        status: pass
    human_judgment: false
  - id: D2
    description: "The accumulation across messages is proven on a session with more than one message, the only case that occurs in reality"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "cmd/wrapper_usage_opencode_test.go#TestOpenCodeUsageSumsEveryAssistantMessage"
        status: pass
      - kind: unit
        ref: "cmd/wrapper_usage_opencode_test.go#TestOpenCodeSessionUsageReadsDisjointTokenColumns"
        status: pass
    human_judgment: false
  - id: D3
    description: "Exactly one containment helper and one symlink resolver exist, and both platform validators use them"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "cmd/wrapper_usage_opencode_test.go#TestSpendPathContainmentHasOneImplementation"
        status: pass
      - kind: unit
        ref: "cmd/wrapper_usage_opencode_test.go#TestBoundedReadOpensThenLimits"
        status: pass
    human_judgment: false
  - id: D4
    description: "One resolver returns per-worker usage for a run on the Claude, OpenCode and directly-spawned paths, and never reads a figure from a completion packet"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "cmd/wrapper_usage_resolve_test.go#TestResolverReadsTranscriptOnTheClaudePath,TestResolverReadsSessionStoreOnTheOpenCodePath,TestResolverUsesAttachedUsageOnTheDirectPath,TestResolverNeverReadsUsageFromACompletionPacket,TestResolverIsIdempotent"
        status: pass
      - kind: unit
        ref: "cmd/spend_packet_guard_test.go#TestCompletionPacketRefusesAssertedWorkerUsage"
        status: pass
    human_judgment: false
  - id: D5
    description: "A worker whose tool reported nothing carries no token figure at all, in the resolver and on disk, and is never dropped (D-01 as amended)"
    requirement: "COST-05"
    verification:
      - kind: unit
        ref: "cmd/wrapper_usage_resolve_test.go#TestResolverMarksUnreportedWorkersWithNoFigure"
        status: pass
      - kind: unit
        ref: "cmd/spend_writer_test.go#TestUnreportedWorkerRowHoldsNoFigure"
        status: pass
      - kind: unit
        ref: "cmd/spend_no_length_derivation_test.go#TestNoTokenCountIsDerivedFromLength"
        status: pass
    human_judgment: false
  - id: D6
    description: "A finished build writes one ledger row per worker, carrying the job name a grouped worker owned, replacing its own rows on a rerun and never touching the continue-keyed file"
    requirement: "COST-03"
    verification:
      - kind: unit
        ref: "cmd/spend_writer_test.go#TestBuildFinalizeWritesOneRowPerWorker,TestGroupedWorkerRowCarriesItsJobName,TestBuildFinalizeRerunReplacesItsOwnRowsOnly"
        status: pass
    human_judgment: false
  - id: D7
    description: "Both platform readers are reachable from a real dispatch path: a real build-finalize run leaves rows on disk"
    requirement: "COST-03"
    verification:
      - kind: integration
        ref: "cmd/spend_writer_test.go#TestBuildFinalizeFilesTheRunsRows"
        status: pass
    human_judgment: false
  - id: D8
    description: "A ledger write that fails is reported and does not fail the build"
    requirement: "COST-03"
    verification:
      - kind: integration
        ref: "cmd/spend_writer_test.go#TestLedgerWriteFailureDoesNotFailTheBuild"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-08-28
status: complete
---

# Phase 196 Plan 05: Salvage, Resolve, and Write Summary

**The OpenCode session-store reader is in the tree with the map that would have aborted the process at the end of every build deleted, its accumulation locked by a two-message fixture, and its duplicated path boundary collapsed into one — joined to the Claude transcript reader behind a single per-run resolver, and wired into the build finalize path so a finished build now leaves one honest ledger row per worker on disk.**

## Performance

- **Duration:** 20 min
- **Started:** 2026-08-28T10:55:28Z
- **Completed:** 2026-08-28T11:15:48Z
- **Tasks:** 3 (each RED then GREEN, plus a salvage-landing commit)
- **Files modified:** 17 (15 created, 2 modified)

## Accomplishments

- **The salvaged reader landed unchanged first**, and its six original tests passed against today's tree with `go vet` clean — the evidence the salvage assessment's compile-and-green claim held (commit `30db0453`).
- **FIX 2-1, the crash, is gone.** `openCodeWorkerNamePatternCache` is deleted and the pattern is compiled per call. The rule is stated twice: `TestOpenCodeWorkerNameMatchingIsConcurrencySafe` proves it under `-race` with 32 goroutines against a hand-written match table, and a structural subtest refuses **any** package-level `var` in the file, so the bug class cannot return even on a run without `-race`.
- **FIX 2-2, the untested path, is covered.** `message/ses_child_a/` now holds two assistant messages, and `TestOpenCodeUsageSumsEveryAssistantMessage` asserts the summed columns as hand-written literals — explicitly failing with a named message if the result equals the first message alone or the last message alone.
- **FIX 2-5, the duplicated boundary, is one copy.** `validateSpendContainedPath` is now the single containment rule for every path the spend subsystem opens; `validateSpendTranscriptPath` and `validateOpenCodeStoragePath` both call it, and `evalSymlinksOrSelf` is gone in favour of 196-03's `evalSpendPathSymlinks`. `TestSpendPathContainmentHasOneImplementation` counts the implementations by walking the syntax tree and fails if a second appears.
- **FIX 2-4, the read gap, is closed.** `readBoundedFile` opens the file, checks regularity on the already-open descriptor, and bounds the read with `io.LimitReader` instead of stat-then-read.
- **One resolver, three paths.** `resolveWrapperWorkerUsage` reads the recorded Claude transcript, the OpenCode session store, or the provider's own attached measurement, and returns exactly one entry per requested worker. A worker the platform reported nothing for comes back `Reported: false` with the zero-value usage — no source tag, no columns, no total.
- **The rows exist.** `writeSpendRowsForRun` is called from `runCodexBuildFinalize` after dispatches reach their terminal status. A real `build-finalize` run now leaves `spend/phase-<N>-build.json` on disk, and `TestBuildFinalizeFilesTheRunsRows` fails if it does not — which is what makes both readers reachable rather than merged-and-orphaned.

## Task Commits

1. **Salvage landing** — `30db0453` (feat) — the branch's files in at file level, six tests green unchanged
2. **Task 1 RED** — `62246f56` (test) — the four fixes' tests, failing
3. **Task 1 GREEN** — `422c13bb` (feat) — cache deleted, boundary shared, read bounded at the open handle
4. **Task 2 RED** — `cade4108` (test) — resolver skeleton with real signatures plus six failing tests
5. **Task 2 GREEN** — `1cf3b5fe` (feat) — the resolver
6. **Task 3 RED** — `e65aa807` (test) — writer skeleton plus six failing disk-reading tests
7. **Task 3 GREEN** — `fc757a1c` (feat) — the writer and the finalize wiring

## RED evidence (real output)

**The salvage landing, before any fix** — `go test ./cmd -run 'TestOpenCodeSessionUsage' -count=1 -v`:

```
--- PASS: TestOpenCodeSessionUsageReadsDisjointTokenColumns (0.01s)
--- PASS: TestOpenCodeSessionUsageIgnoresSessionsOutsideTheRunWindow (0.01s)
--- PASS: TestOpenCodeSessionUsageRefusesAmbiguousWorkerMatch (0.01s)
--- PASS: TestOpenCodeSessionUsageIgnoresNonMatchingWorktree (0.01s)
--- PASS: TestOpenCodeSessionUsageRefusesStorageRootOutsideHome (0.00s)
--- PASS: TestOpenCodeSessionUsageToleratesMalformedRecords (0.01s)
ok  	github.com/calcosmic/Aether/cmd	0.750s
```

**Task 1 RED, plain `go test`:**

```
--- FAIL: TestOpenCodeWorkerNameMatchingIsConcurrencySafe/the_opencode_reader_shares_no_package-level_mutable_state
    package-level var openCodeWorkerNamePatternCache at wrapper_usage_opencode.go:370:5 is shared mutable
    state in the cost path; the unsynchronised worker-name pattern cache lived exactly here and Go aborts
    the process on a concurrent map write (FIX 2-1)
--- FAIL: TestSpendPathContainmentHasOneImplementation
    the spend path containment rule is implemented in 2 place(s) [validateOpenCodeStoragePath
    validateSpendTranscriptPath], want exactly one named validateSpendContainedPath; two copies of a
    security boundary is one copy too many (FIX 2-5)
    symlink evaluation is implemented in 2 place(s) [evalSpendPathSymlinks evalSymlinksOrSelf] ...
--- FAIL: TestBoundedReadOpensThenLimits/the_bound_is_applied_by_limiting_the_read,_not_by_a_prior_stat_of_the_path
    readBoundedFile calls os.Stat on the path before reading it -- that is the check-then-read gap FIX 2-4 removes
    readBoundedFile calls os.ReadFile, which reads the whole file regardless of the bound
```

**Task 1 RED under the race detector** — the crash the assessment predicted, seen directly:

```
WARNING: DATA RACE
Write at 0x00c00012f110 by goroutine 35:
  runtime.mapaccess2_faststr()
  github.com/calcosmic/Aether/cmd.openCodeTitleMatchesWorker()
      /Users/callumcowie/repos/Aether/cmd/wrapper_usage_opencode.go:383
...
--- FAIL: TestOpenCodeWorkerNameMatchingIsConcurrencySafe/matching_from_many_goroutines_agrees_with_the_single-threaded_answer
    testing.go:1712: race detected during execution of test
```

**Task 2 RED** — `go test ./cmd -run 'Test(Resolver|...)' -count=1`:

```
--- FAIL: TestResolverReadsTranscriptOnTheClaudePath
    worker "Mason-67" is missing from the resolution entirely; a vanished worker makes a run look cheaper than it was: []
--- FAIL: TestResolverMarksUnreportedWorkersWithNoFigure/the_claude_path_with_no_session_recorded_at_all
    got 0 worker rows, want 2 -- every expected worker must come back, reported or not: []
```

**Task 3 RED** (re-recorded after the harness path correction described under deviations, with the writer body stubbed):

```
--- FAIL: TestBuildFinalizeWritesOneRowPerWorker
    read ledger .../.aether/data/spend/phase-7-build.json: no such file or directory
--- FAIL: TestGroupedWorkerRowCarriesItsJobName            (same, no file)
--- FAIL: TestUnreportedWorkerRowHoldsNoFigure             (same, no file)
--- FAIL: TestBuildFinalizeRerunReplacesItsOwnRowsOnly     (same, no file)
--- FAIL: TestLedgerWriteFailureDoesNotFailTheBuild/the_writer_surfaces_a_save_failure_as_an_error
--- FAIL: TestLedgerWriteFailureDoesNotFailTheBuild/a_build_whose_accounting_cannot_be_filed_still_finishes
--- FAIL: TestBuildFinalizeFilesTheRunsRows
    read ledger .../.aether/data/spend/phase-1-build.json: no such file or directory
```

## Non-vacuity, demonstrated by breaking the code on purpose

Two of this plan's rules lock arithmetic that was already correct, so they passed on arrival. Prose claiming a test "would catch" something is what this repository's audits describe its own failures as, so each was produced by editing the shipped code, running the suite, recording the output, and restoring the file.

**1. `readOpenCodeUsage` made last-message-wins instead of summing:**

```
--- FAIL: TestOpenCodeSessionUsageReadsDisjointTokenColumns
    mason InputTokens = 1201, want 1744
    mason OutputTokens = 806, want 929
    mason CachedInputTokens = 30500, want 50270
    mason TotalTokens = 36603, want 57039
--- FAIL: TestOpenCodeUsageSumsEveryAssistantMessage
    TotalTokens = 36603, which is the LAST message alone -- later messages are overwriting earlier ones
    instead of adding to them
```

**2. A completion-packet symbol planted in the resolver:**

```
--- FAIL: TestResolverNeverReadsUsageFromACompletionPacket/the_resolver_names_no_completion-packet_symbol_at_all
    plantedCompletionPacketUsage at wrapper_usage_resolve.go:211:6 names the completion packet; a token
    figure relayed by the orchestrating model is an assertion, not a measurement
```

Both files were restored and re-run green afterwards.

## The ledger file, read by hand

The plan's fourth verification step asks for a build finalize against a temporary store with the resulting ledger read by hand. Run through the real `build-finalize` command, the file on disk is:

```json
{
  "schema_version": 1,
  "phase": 1,
  "phase_name": "Spend rows on disk",
  "workflow": "build",
  "recorded_at": "2026-08-28T11:15:33Z",
  "rows": [
    {
      "name": "Chip-87",
      "caste": "builder",
      "parent": "",
      "task": "Create evidence",
      "job_name": "single-1.1",
      "status": "completed",
      "usage": {}
    }
  ]
}
```

`"usage": {}` is the point, not a gap: that test environment has no platform session store, so nothing measured this worker — and D-01 as amended gives it no number at all rather than a zero or a guess. The row still exists, carrying the worker's name, its grouped job and its outcome, so the worker cannot vanish and make the run look cheaper than it was.

## Acceptance criteria — commands run and results

**Task 1**

| Criterion | Command | Result |
|---|---|---|
| Salvaged tests pass, including the six | `go test ./cmd -run 'TestOpenCode' -count=1` | PASS |
| Concurrency test passes and fails if the cache returns | `go test ./cmd -run 'TestOpenCodeWorkerNameMatchingIsConcurrencySafe' -race -count=1` | PASS; failed with a real DATA RACE and a named structural failure before the fix |
| Multi-message sum asserted as a hand-written literal | `TestOpenCodeUsageSumsEveryAssistantMessage` | PASS; every expected figure is a `const` with its addition written out |
| Exactly one containment helper, both callers use it | `go test ./cmd -run 'TestSpendPathContainmentHasOneImplementation' -count=1` | PASS; named both duplicates before the fix |
| No expected value produced by calling the reader or the billed-total helper | inspection of the test source; all expectations are `const` literals | PASS |
| Plan verify | `go test ./cmd -run 'Test(OpenCode\|SpendSessionRecord)' -race -count=1` | `ok ... 2.102s` |

**Task 2**

| Criterion | Command | Result |
|---|---|---|
| Six resolver tests pass | `go test ./cmd -run 'TestResolverReadsTranscriptOnTheClaudePath\|TestResolverReadsSessionStoreOnTheOpenCodePath\|TestResolverUsesAttachedUsageOnTheDirectPath\|TestResolverMarksUnreportedWorkersWithNoFigure\|TestResolverNeverReadsUsageFromACompletionPacket\|TestResolverIsIdempotent' -count=1` | PASS |
| Missing/unreadable source yields not-reported, no figure, no build-stopping error | `TestResolverMarksUnreportedWorkersWithNoFigure` — four cases, none returns an error | PASS |
| The two existing guards still pass | `go test ./cmd -run 'Test(CompletionPacketRefusesAssertedWorkerUsage\|NoTokenCountIsDerivedFromLength)' -count=1` | PASS |

**Task 3**

| Criterion | Command | Result |
|---|---|---|
| Five named writer tests pass | `go test ./cmd -run 'TestBuildFinalizeWritesOneRowPerWorker\|TestGroupedWorkerRowCarriesItsJobName\|TestUnreportedWorkerRowHoldsNoFigure\|TestBuildFinalizeRerunReplacesItsOwnRowsOnly\|TestLedgerWriteFailureDoesNotFailTheBuild' -count=1` | PASS |
| The row assertions read the saved file back from disk | `loadSpendLedgerFromDisk` decodes the file's own bytes; the job-name test additionally decodes raw row maps to prove the absent key | PASS |
| A build finalize leaves the continue-keyed file untouched | `TestBuildFinalizeFilesTheRunsRows` (`os.Stat` → `IsNotExist`) and `TestBuildFinalizeRerunReplacesItsOwnRowsOnly` (byte-compare of the seeded file) | PASS |
| Existing finalize behaviour unchanged | `go test ./cmd -run 'Test(BuildFinalize\|MergedDispatchCreditsEveryCoveredTask)' -count=1` | PASS |

**Plan-level verification**

| Command | Result |
|---|---|
| `go test ./cmd -run 'Test(OpenCode\|Resolver\|Spend\|BuildFinalize)' -count=1` | `ok  github.com/calcosmic/Aether/cmd  3.240s` |
| `go test ./cmd -run 'TestOpenCodeWorkerNameMatchingIsConcurrencySafe' -race -count=1` | `ok ... 2.025s` |
| `go build ./cmd/aether` | exit 0 |
| `go build ./...` | exit 0 |
| `go vet ./cmd ./pkg/codex` | exit 0 |
| `gofmt -l cmd/ pkg/` | no output |
| Build finalize against a temporary store, ledger read by hand | done — file quoted above |
| Repo guards re-run: `Test(NoTokenCountIsDerivedFromLength\|BothAccountingPathsAgreeOnTheTotal\|WiringGuardsHaveNoRuntimeEscapeHatch\|WiringGateStepRunsEveryWiringTest\|CompletionPacket\|ClaudeTranscript)` | `ok ... 1.173s` |
| `go test ./cmd -run 'Orphan\|Reachability\|DeadCode\|Unused' -count=1` | `ok ... 2.399s` |

## Files Created/Modified

- `cmd/wrapper_usage_opencode.go` — the salvaged reader with the pattern cache deleted, containment delegated, and bounded reads opening before limiting
- `cmd/wrapper_usage_opencode_test.go` — the six salvaged tests plus four new ones covering the four fixes
- `cmd/testdata/spend/opencode/` — the salvaged fixture plus `message/ses_child_a/msg_2.json`, with a README rewritten to record what was checked against the real store
- `cmd/wrapper_usage_resolve.go` — `resolveWrapperWorkerUsage`, the platform normalizer, and the one reported-versus-absent rule
- `cmd/wrapper_usage_resolve_test.go` — six named tests plus the completion-packet AST guard
- `cmd/spend_writer.go` — `writeSpendRowsForRun` and the run-window helper
- `cmd/spend_writer_test.go` — six tests, all reading the saved ledger back off disk, including two that drive the real `build-finalize` command
- `cmd/spend_session_capture.go` — `validateSpendContainedPath` extracted as the one containment boundary; `validateSpendTranscriptPath` reduced to naming its root
- `cmd/codex_build_finalize.go` — the finalize path files the run's rows before returning, reports a failure without failing the build, and surfaces `spend_rows_written` / `spend_rows_measured` / `spend_ledger_note`

## Decisions Made

- **Delete the cache, do not lock it.** The salvage assessment offered a `sync.Map` or deletion and preferred deletion; the cache was also unbounded, and compiling a few short quoted names per run costs nothing measurable.
- **A worker is accounted under `dispatch.Name`, not `dispatch.AgentName`.** Observed from a real finalize run: `AgentName` is the agent *definition* (`aether-builder`), shared by several workers in one build, while `Name` is the deterministic per-worker name (`Chip-87`) that OpenCode's own session titles carry and that the resolver matches on. Accounting under `AgentName` would have merged distinct workers into one row.
- **The orchestrating session's own turns are returned separately.** 196-03 explicitly left this to this plan. The session is not a dispatched worker, so presenting it as one would attribute the owner's own conversation to a build worker; discarding it would lose real spend. It is returned in its own field, and a test fails if it ever appears in the worker list or is folded into a worker's figure.
- **"Reported" keys on the source tag, never on a number.** A worker that genuinely billed zero and a worker whose tool reported nothing both present as zero tokens, and D-01 as amended renders those two differently. Only the tag can tell them apart, so the flag and the on-disk signal are both the tag.
- **The provider's own attached measurement outranks a session-artifact read** for the same worker on every path: the artifact is the platform's record of a measurement, the attachment is the measurement.
- **A dispatch with an unusable status is dropped with a named note, not allowed to sink the ledger.** `saveSpendLedger` is fail-closed for the whole file (196-01's design, correctly), so one bad row would otherwise lose the entire run's accounting. Nothing invents a status the dispatch never stated.
- **The two stale doc comments naming "plan 174-06" were corrected**, not left. CLAUDE.md makes a documentation claim about runtime behaviour that cannot be true a named defect, and one of them described falling through to an estimate that D-01 as amended abolished.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] The concurrency rule is also stated structurally, so it fails without `-race`**

- **Found during:** Task 1
- **Issue:** The plan's criterion is a race-detector test. CI and most local runs do not pass `-race` (this plan's own test discipline forbids it on the broad sweep), so a restored cache would have been caught only on the one command that remembers to ask for it.
- **Fix:** Added the subtest `the opencode reader shares no package-level mutable state`, an AST scan that refuses any package-level `var` in `cmd/wrapper_usage_opencode.go` and names it.
- **Verification:** It named `openCodeWorkerNamePatternCache` at `wrapper_usage_opencode.go:370:5` before the fix, under a plain `go test`.
- **Committed in:** `62246f56` (test) / `422c13bb` (fix)

**2. [Rule 2 - Missing Critical] "Exactly one containment helper exists" made executable**

- **Found during:** Task 1
- **Issue:** The plan states this as an acceptance criterion with no enforcing command. Under CLAUDE.md's Definition of Done an unenforced criterion is unsatisfied, and a second copy of a security boundary reappearing is precisely the failure FIX 2-5 exists to prevent.
- **Fix:** `TestSpendPathContainmentHasOneImplementation` parses the three spend files, counts every function that calls `filepath.Rel` and mentions `".."`, requires exactly one and names it, does the same for `filepath.EvalSymlinks`, and asserts both platform validators call the shared helper.
- **Verification:** Named both duplicates and both symlink helpers before the fix (output above).
- **Committed in:** `62246f56` (test) / `422c13bb` (fix)

**3. [Rule 2 - Missing Critical] The bounded-read shape is asserted, not just its behaviour**

- **Found during:** Task 1
- **Issue:** Refusing an oversized file passes identically whether the bound is applied by a prior `os.Stat` or by limiting the read, so a behavioural test alone cannot hold FIX 2-4 in place.
- **Fix:** `TestBoundedReadOpensThenLimits` adds a subtest asserting `readBoundedFile` calls `os.Open` and `io.LimitReader` and calls neither `os.Stat` nor `os.ReadFile`, alongside the behavioural cases.
- **Verification:** Named all four conditions before the fix.
- **Committed in:** `62246f56` (test) / `422c13bb` (fix)

**4. [Rule 2 - Missing Critical] A named wiring test, `TestBuildFinalizeFilesTheRunsRows`**

- **Found during:** Task 3
- **Issue:** The plan's five named writer tests all exercise the writer directly. None of them would fail if the call were removed from `runCodexBuildFinalize` — which is the exact condition the salvage assessment refuses and this plan exists to avoid.
- **Fix:** Added a test that drives the real `build-finalize` command end to end and then reads `spend/phase-1-build.json` off disk, and additionally asserts the continue-keyed file was not created.
- **Verification:** Failed with "no such file or directory" while the writer body was stubbed; passes with the wiring in place.
- **Committed in:** `e65aa807` (test) / `fc757a1c` (wiring)

**5. [Rule 1 - Bug] Test-harness path correction, and the RED re-recorded because of it**

- **Found during:** Task 3
- **Issue:** The writer fixture read the ledger from `newTestStore`'s first return value, which is the repo-shaped temp root, while the store itself lives at `<root>/.aether/data`. Four of the six writer tests were therefore red for a path reason rather than for the absence of the writer.
- **Fix:** The fixture now computes the store directory explicitly. Because the first RED run's evidence was partly wrong, the RED was **re-recorded** against the corrected harness with the writer body stubbed, and that corrected output is what this summary quotes.
- **Verification:** All six fail with the stub and pass with the implementation, on the corrected paths.
- **Committed in:** `e65aa807` (test, with the correction landing in `fc757a1c`)

**6. [Rule 2 - Missing Critical] Two documentation claims corrected**

- **Found during:** Task 1
- **Issue:** `openCodeSessionUsageForRunOrNone`'s doc comment said it is "the convenience entry point plan 174-06 calls" — a plan that never existed — and the ambiguity branch said an unresolved worker "falls through to plan 174-06's estimate", which D-01 as amended abolished. CLAUDE.md makes an untestable claim about runtime behaviour a named defect.
- **Fix:** Both rewritten to describe what actually happens: the resolver calls it, and an ambiguous worker is rendered with no figure at all.
- **Verification:** `go build ./...` and the full OpenCode suite pass; the claims now match the code this plan wired.
- **Committed in:** `422c13bb`

---

**Total deviations:** 6 auto-fixed (1 bug, 5 missing-critical). **Impact on plan:** Five tighten requirements the plan already stated; one corrects a test-harness path and re-records the evidence honestly. One file outside `files_modified` was touched — none: `cmd/spend_session_capture.go` and `cmd/codex_build_finalize.go` are both in the plan's list. No architectural change, no new dependency, no scope creep.

## Issues Encountered

- **The salvaged branch's worker-name assumption did not survive contact with a real dispatch.** The first end-to-end finalize run showed `agent_name: aether-builder` and `name: Chip-87` on the same dispatch. Accounting under `AgentName` — the reading the ledger's own field name invites — would have collapsed every builder in a build into one row. Corrected before the writer's GREEN commit, with the reasoning written into the code.
- **Two of the four Task 1 fixes lock arithmetic that was already right.** That is the honest result and it is what the salvage assessment predicted: the reader's summation was verified correct against the owner's real store, and FIX 2-2's job is to make that fact something the build can check rather than something a document asserts. Their non-vacuity is demonstrated by deliberate breakage above rather than claimed.

## Known Stubs

None. Every symbol added in this plan is exercised by a test in the same commit, and the two entry points the salvage assessment flagged as orphans are now reached from `runCodexBuildFinalize`.

Two things remain deliberately unbuilt here and belong to later plans in this phase: nothing yet **renders** the cost line from these rows (196-06/07), and the continue lane does not yet write its own ledger (this plan's stated scope is the build lane). Neither is a stub — the ledger is written and readable today, and `aether spend` is 196-07's work.

## Broken-windows ledger

No entry appended: this plan left no stub, no skipped test and no unrun `<verify>`.

## Threat Flags

None, and one boundary is **tightened rather than widened**: the spend subsystem's path containment rule now has exactly one implementation instead of two, and the OpenCode validator inherits 196-03's corrected symlink resolution rather than its own weaker fallback. Every existing refusal case — storage root outside the home tree, crafted sibling directory, relative path, null byte, traversal — was re-run and still fails closed. The bounded read closes a check-then-read gap. No new network endpoint, no auth path, no schema change.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Names 196-06/07 depend on:** `resolveWrapperWorkerUsage(wrapperUsageRequest) wrapperUsageResolution`, the `wrapperWorkerUsage` row (`WorkerName`, `Usage`, `Reported`), `wrapperUsageWasReported`, `writeSpendRowsForRun(spendWriteRequest) (spendWriteOutcome, error)`, and `spendWorkerNameForDispatch`.
- **The renderer must key "is this a real number?" on `Reported` or on the source tag, never on whether the total is zero.** A genuine zero and an absent measurement are different facts and D-01 as amended renders them differently.
- **The orchestrating session's spend is available but is not a worker.** `wrapperUsageResolution.SessionUsage` / `SessionReported` carry it; 196-06/07 decides whether the cost line shows it, and a test already fails if it is ever folded into a worker's row.
- **The finalize result now carries `spend_rows_written`, `spend_rows_measured` and (when there is something to say) `spend_ledger_note`**, so a wrapper can mention the detail view per D-06 without recomputing anything.
- **The continue lane still writes nothing.** `writeSpendRowsForRun` takes the workflow word and `spendWorkflowContinue` is already valid, so wiring it is a call site, not new code.

## Self-Check: PASSED

- `cmd/wrapper_usage_opencode.go` — FOUND on disk
- `cmd/wrapper_usage_opencode_test.go` — FOUND on disk
- `cmd/wrapper_usage_resolve.go` — FOUND on disk
- `cmd/wrapper_usage_resolve_test.go` — FOUND on disk
- `cmd/spend_writer.go` — FOUND on disk
- `cmd/spend_writer_test.go` — FOUND on disk
- `cmd/testdata/spend/opencode/README.md` — FOUND on disk
- `cmd/testdata/spend/opencode/message/ses_child_a/msg_2.json` — FOUND on disk
- `30db0453`, `62246f56`, `422c13bb`, `cade4108`, `1cf3b5fe`, `e65aa807`, `fc757a1c` — all FOUND in `git log`
- All task acceptance criteria re-run after the final commit; all pass (tables above)
- Plan-level verification re-run after the final commit: tests ok, `go build ./...` exit 0, `go vet ./cmd ./pkg/codex` exit 0, `gofmt -l cmd/ pkg/` empty
- Working tree clean of unstaged changes to any file this plan touched

---
*Phase: 196-see-what-it-cost*
*Completed: 2026-08-28*
