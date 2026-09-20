---
phase: 196-see-what-it-cost
plan: 07
subsystem: infra
tags: [spend-ledger, token-usage, closeout, ceremony, go, tdd]

requires:
  - phase: 196-see-what-it-cost
    provides: "196-01's spendLedger and computeSpendTotals; 196-02's one authoritative token type and the no-length-derivation ratchet; 196-05's writeSpendRowsForRun and resolveWrapperWorkerUsage; 196-06's spendRowReportedUsage, the dash sentinel and the cell-parsed row"
provides:
  - "The check (`continue-finalize`) files its own per-worker rows under the continue key, so a phase's recorded cost is the build's plus the check's rather than half of it"
  - "The writer leaves no file at all when a run filed no rows, because an empty file reads as a run that cost nothing"
  - "renderSpendCostLine — the ONE closeout cost block, in the owner's D-01 shape, shared by every ending screen"
  - "Exactly one cost block per lane, proved by counting occurrences on each lane driven end to end, not by asserting presence"
  - "A no-monetary-amount guard asserted on the rendered output, fed a row that carries the provider's own reported cost"
affects: [196-08]

actuals:
  tokens: 15400
  tasks: 2
  commits: 5

tech-stack:
  added: []
  patterns:
    - "The cost block is the LAST thing on every ending screen, on every lane, so its position is one rule rather than two"
    - "An exactly-one guard counts occurrences and carries an anti-vacuity subtest proving the counter can reach two"
    - "Display width is measured in terminal columns (emoji 2, variation selectors 0), never in code points, so a caste glyph cannot skew a column"

key-files:
  created:
    - cmd/spend_cost_line.go
    - cmd/spend_cost_line_test.go
    - cmd/ceremony_closeout_spend_test.go
  modified:
    - cmd/spend_writer.go
    - cmd/spend_writer_test.go
    - cmd/codex_continue_finalize.go
    - cmd/ceremony_cmd.go
    - cmd/codex_workflow_cmds.go
    - cmd/testdata/golden_build.txt
    - cmd/testdata/golden_continue.txt

key-decisions:
  - "The block is the LAST thing on every ending screen, after the next-step block, on both lanes. The plan allowed either side of the next-step block; picking one side everywhere makes 'the cost line is the last thing you read' a single rule that cannot drift out of agreement between two lanes."
  - "The heading says 'What This Phase Has Cost', not 'this run'. The figures span the building pass and the checking pass, which are recorded separately and neither erases the other, so at the end of a check the block honestly reports the whole phase. A heading claiming 'this run' would be the more precise-sounding lie."
  - "Compact magnitudes are TRUNCATED, never rounded up (1,420,000 -> 1.4M; 9,999 -> 9.9K), so a headline can never claim a run cost more than its rows say it did. The exact recorded figure is one command away in the detail view, which abbreviates nothing."
  - "The finalizers deliberately render no cost block. On the platform-driven lane the finalizer runs first and the ending screen follows it, so a finalizer that also printed the block would end that lane with two. Placing the call at the three terminal surfaces rather than inside the shared renderContinueVisual is what makes exactly-one structural rather than a matter of output mode."
  - "The writer now files nothing when a run produced no rows. An empty ledger on disk is indistinguishable from a run that genuinely cost nothing, and the cost line reads that file — a phase whose check spawned nobody must say 'nothing recorded', never show a zero."
  - "An ESTIMATED row renders the dash exactly like an empty one, matching 196-06's detail view. D-01 as amended forbids rendering an estimate at all, and prompt-character count is the only estimate mechanism in the tree, so showing one — even labelled — would be precisely what success criterion 3 forbids."
  - "The check writes its rows immediately after every reviewer reaches a terminal status, BEFORE any result envelope is assembled, so the record exists whether that check goes on to advance the phase or to block it. A blocked check still spent what it spent."

patterns-established:
  - "Exactly-one guards count, and prove their counter can count: the same test that requires one block also feeds the counter a doubled screen and requires two"
  - "A money guard is a regex run over rendered output, fed an input that actually carries a money value, plus a planted-figure subtest proving the regex recognises one"

requirements-completed: [COST-01, COST-02]

coverage:
  - id: D1
    description: "A finished check files one row per reviewer and watcher it ran, under its own key, and the build's rows for the same phase are untouched"
    requirement: "COST-02"
    verification:
      - kind: integration
        ref: "cmd/spend_writer_test.go#TestContinueFinalizeWritesItsOwnRows"
        status: pass
      - kind: integration
        ref: "cmd/spend_writer_test.go#TestContinueDoesNotEraseBuildRowsEndToEnd"
        status: pass
      - kind: unit
        ref: "cmd/spend_ledger_test.go#TestLedgerPhaseTotalAddsBuildAndContinue"
        status: pass
    human_judgment: false
  - id: D2
    description: "A run that spawned nobody leaves no ledger file, so an empty file can never be read as a run that cost nothing"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "cmd/spend_writer_test.go#TestContinueWithNoWorkersWritesNoFile"
        status: pass
    human_judgment: false
  - id: D3
    description: "Exactly one cost block ends every build and every check, on the platform-driven lane and on the direct in-process lane alike"
    requirement: "COST-01"
    verification:
      - kind: unit
        ref: "cmd/ceremony_closeout_spend_test.go#TestBuildEndsWithOneCostLine"
        status: pass
      - kind: unit
        ref: "cmd/ceremony_closeout_spend_test.go#TestContinueEndsWithOneCostLine"
        status: pass
      - kind: integration
        ref: "cmd/ceremony_closeout_spend_test.go#TestNoLaneRendersTwoCostLines"
        status: pass
      - kind: unit
        ref: "cmd/ceremony_closeout_spend_test.go#TestCloseoutCostLineCountIsAssertedNotAssumed"
        status: pass
    human_judgment: false
  - id: D4
    description: "D-01 as amended: an unreported worker shows no number at all, the total counts only measured workers and says so, and the footnote disappears when everyone was measured"
    requirement: "COST-01"
    verification:
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestUnreportedWorkerShowsNoNumber (asserted on the parsed figure cell)"
        status: pass
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestTotalCountsOnlyMeasuredWorkers"
        status: pass
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestFootnoteAbsentWhenEveryWorkerMeasured"
        status: pass
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestCostLineShowsNoTotalWhenNothingWasMeasured"
        status: pass
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestCostLineWithNoRowsShowsNoNumber"
        status: pass
    human_judgment: false
  - id: D5
    description: "No monetary amount appears anywhere in the rendered output, including for a row carrying the provider's own reported cost"
    requirement: "COST-01"
    verification:
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestCostLineHasNoMonetaryFigure (fed USDCost 12.34; anti-vacuity subtest plants '$12.34')"
        status: pass
      - kind: unit
        ref: "cmd/spend_ledger_test.go#TestSpendLedgerCarriesNoCurrencyField"
        status: pass
    human_judgment: false
  - id: D6
    description: "No figure on the line is derived from a character or prompt length"
    requirement: "COST-01"
    verification:
      - kind: unit
        ref: "cmd/spend_no_length_derivation_test.go#TestNoTokenCountIsDerivedFromLength"
        status: pass
    human_judgment: false
  - id: D7
    description: "The wording reads as plain English to someone who has never opened a file here"
    verification:
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestCostLineShowsPerWorkerAndTotal (refuses 20 repo-invented words in the rendered block)"
        status: pass
    human_judgment: true
    rationale: "A word list can refuse the jargon this repository already knows it invented; whether the sentences actually land for the owner is his call, not a test's. The runnable part — the figures, the sentinel, the total and the absence of jargon — is covered above."

duration: 78 min
completed: 2026-08-28
status: complete
---

# Phase 196 Plan 07: The one cost line Summary

**Every build and every check now ends with a single block naming what the phase has cost, worker by worker, with a dash and the words "not reported" where a tool told us nothing — and the checking pass finally files its own rows, so the total is the whole phase rather than half of it.**

## Performance

- **Duration:** 78 min
- **Started:** 2026-08-28T11:05:00Z
- **Completed:** 2026-08-28T12:23:00Z
- **Tasks:** 2
- **Files modified:** 10 (3 created, 7 modified)

## The line itself

This is the real rendered block, captured from `renderSpendCostLine` over seeded rows, colour stripped:

```
── What This Phase Has Cost ──
Cost: 1.4M tokens across 3 workers. The total counts only the 2 whose tools reported a figure.
  🔨🐜 Builder Mason-67  1.2M  measured
  🔍🐜 Scout Roam-90        —  not reported
  👁️🐜 Watcher Keen-12   220K  measured
(1 worker's tool did not report usage, so its use is not in the total.)
```

When every tool reported, the footnote disappears entirely and the total says so:

```
── What This Phase Has Cost ──
Cost: 1.4M tokens across 2 workers. The total counts all 2, because every tool reported a figure.
  🔨🐜 Builder Mason-67  1.2M  measured
  👁️🐜 Watcher Keen-12   220K  measured
```

When nothing was recorded, one plain sentence and no number at all:

```
── What This Phase Has Cost ──
No token use was recorded for this phase, so there is no figure to show.
```

## Accomplishments

- **The checking pass files its own record.** `continue-finalize` now calls the same writer the build lane has used since 196-05, with the continue workflow word, immediately after every reviewer and watcher reaches a terminal status. Build rows and check rows live in two files keyed separately; an end-to-end test runs a real build write and a real check finalize against one store and requires each file to hold only its own rows.
- **An empty run leaves no file.** The writer used to save a ledger even with nothing in it. An empty ledger on disk is indistinguishable from a run that cost nothing — and the cost line reads that file — so it now returns without saving when there is nothing to file.
- **One renderer, one block per lane.** `cmd/spend_cost_line.go` is the only place the block is built. It is called from three terminal surfaces (the platform-driven ending screen, `aether build <n>`, `aether continue`) and from no finalizer, which is what makes exactly-one structural rather than dependent on output mode. Each lane is driven end to end and its blocks are counted; the counter itself is proved able to reach two.
- **An unreported worker shows no number, asserted on the cell.** The dash test parses the row into cells and requires the figure column to be exactly the dash sentinel, so a zero fails it as surely as a figure does while `Roam-90`'s own digits still pass. An estimated row renders the dash too, because D-01 as amended forbids showing an estimate at all.
- **No money, proved on the output.** The renderer is fed a row carrying the provider's own reported cost (`USDCost: 12.34`) and the guard requires it never to reach the screen — asserted on the rendered text, so a comment can neither satisfy nor break it, with a planted `$12.34` subtest proving the regex recognises one.

## Task Commits

1. **Task 1 (RED): failing tests for the check's own rows** — `b986549d` (test)
2. **Task 1 (GREEN): file the check's own per-worker rows** — `707c4ec8` (feat)
3. **Task 2 (RED): failing tests for the one cost line** — `7d5fa587` (test)
4. **Task 2 (GREEN): the one cost line, on both lanes** — `330c51a7` (feat)
5. **Task 2 (REFACTOR): name the phase in the heading** — `fc0222ad` (refactor)

Real RED output, quoted from the runs:

```
--- FAIL: TestContinueFinalizeWritesItsOwnRows (1.18s)
    spend_writer_test.go:538: read ledger .../spend/phase-1-continue.json: no such file or directory
--- FAIL: TestContinueWithNoWorkersWritesNoFile/the_writer_files_nothing_when_a_run_had_no_workers
    spend_writer_test.go:633: a run with no workers left a file at .../spend/phase-7-continue.json;
        an empty file is indistinguishable from a run that cost nothing
```

```
cmd/ceremony_closeout_spend_test.go:33:42: undefined: spendCostLineHeading
cmd/spend_cost_line_test.go:86:11: undefined: renderSpendCostLine
cmd/spend_cost_line_test.go:341:13: undefined: spendCompactTokenFigure
FAIL	github.com/calcosmic/Aether/cmd [build failed]
```

## Files Created/Modified

- `cmd/spend_cost_line.go` — the one renderer, the compact magnitude, the total sentence, the footnote, the terminal-column width helper and the append entry point
- `cmd/spend_cost_line_test.go` — the block's shape, the dash sentinel on the parsed figure cell, the measured-only total, the vanishing footnote, the money guard, the empty-phase sentence, the magnitude table
- `cmd/ceremony_closeout_spend_test.go` — the exactly-one counts on both workflows and both lanes, plus the counter's own anti-vacuity check
- `cmd/spend_writer.go` — the no-empty-file guard and `spendDispatchesFromContinueFlow`, a conversion rather than a second writer
- `cmd/spend_writer_test.go` — the three continue-lane guards and the real plan-only + finalize driver they share
- `cmd/codex_continue_finalize.go` — the check's own call to the shared writer, non-fatal, before any result envelope is assembled
- `cmd/ceremony_cmd.go` — the block appended to the build and check ending screens only
- `cmd/codex_workflow_cmds.go` — the block appended to `aether build <n>` and to both `aether continue` endings (passing and blocked)
- `cmd/testdata/golden_build.txt`, `cmd/testdata/golden_continue.txt` — refreshed, additively, because the two direct-lane ending screens genuinely changed

## Decisions Made

See `key-decisions` in the frontmatter. The one a later reader is most likely to want the reasoning for:

**The block is last on every screen, and the finalizers render none.** The plan allowed the block on either side of the next-step block. Choosing one side everywhere turns "the cost line is the last thing you read" into a single rule; two lanes with two positions is how the same idea starts being described two ways. And the finalizer staying silent is not an oversight: on the platform-driven lane the finalizer runs first and the ending screen follows it, so a finalizer that also printed the block would end that lane with two — the exact failure `TestNoLaneRendersTwoCostLines` exists to catch.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Column alignment needed terminal-column widths, not code-point counts**

- **Found during:** Task 2 (green step, reading the rendered line by hand)
- **Issue:** The worker column was padded by counting code points. A caste glyph draws two columns and an emoji variation selector draws none, so `👁️🐜 Watcher` and `🔨🐜 Builder` carry different code-point counts for the same width on screen — the figures came out visibly misaligned, on the one surface whose entire purpose is being read at a glance.
- **Fix:** Added `spendDisplayWidth`, which measures terminal columns (emoji 2, variation selectors and the zero-width joiner 0), and padded with it. It measures text for layout only and is never an input to any token count.
- **Files modified:** `cmd/spend_cost_line.go`
- **Verification:** the hand-rendered block aligns; `TestNoTokenCountIsDerivedFromLength` still passes.
- **Committed in:** `330c51a7`

**2. [Rule 2 - Missing Critical] A blocked check must still file its rows**

- **Found during:** Task 1
- **Issue:** The natural place to file the check's rows is where its result envelope is built — but that happens in four different branches, three of which are early returns for a blocked or superseded check. A blocked check still spent everything it spent.
- **Fix:** The write happens once, immediately after every worker reaches a terminal status and before any envelope is assembled, so it covers the advancing path and every blocked path alike.
- **Files modified:** `cmd/codex_continue_finalize.go`
- **Verification:** `TestContinueFinalizeWritesItsOwnRows` and the end-to-end pair pass; `TestFinalizeBlockedExternalContinueDoesNotOverwritePausedState` still passes.
- **Committed in:** `707c4ec8`

---

**Total deviations:** 2 auto-fixed (both missing-critical)
**Impact on plan:** Both were needed for the plan's own promises to hold — an unreadable column defeats the surface, and a blocked run that files nothing understates the phase. No scope creep.

## Issues Encountered

**The direct in-process lane still has nothing to report.** `aether build <n>` and `aether continue` run their workers inside the runtime and file no ledger rows — only `build-finalize` and `continue-finalize` do, and those belong to the platform-driven lane. The block therefore renders honestly on the direct lane as "No token use was recorded for this phase, so there is no figure to show." That is exactly the behaviour the plan specifies for a phase with no recorded rows, and it is why both refreshed golden files show that sentence. Wiring the direct lane's own writer was not in this plan's scope; plan 196-08's end-to-end reachability proof is the right place to decide whether it belongs to this phase at all.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Plan 196-08 can now assert the whole path end to end: a finished run leaves rows, `aether spend` shows them, and the ending screen prints one line over them.
- `renderSpendCostLine` and `spendCostLineHeading` are the symbols 196-08's anti-orphan check should expect to find called; there are three call sites and no fourth is permitted.
- One open question for 196-08, recorded above: the direct in-process lane files no rows, so its cost line always reads "nothing recorded".

## Self-Check: PASSED

Files claimed as created exist on disk:

```
FOUND: cmd/spend_cost_line.go
FOUND: cmd/spend_cost_line_test.go
FOUND: cmd/ceremony_closeout_spend_test.go
```

Commits claimed exist in git:

```
FOUND: b986549d  test(196-07): add failing tests for the continue lane's own ledger rows
FOUND: 707c4ec8  feat(196-07): file the check's own per-worker token rows
FOUND: 7d5fa587  test(196-07): add failing tests for the one closeout cost line
FOUND: 330c51a7  feat(196-07): end every build and check with one honest cost line
FOUND: fc0222ad  refactor(196-07): name the phase in the cost line heading
```

Plan-level verification re-run at close-out:

```
go test ./cmd -run 'Test(CostLine|EndsWithOneCostLine|ContinueFinalizeWrites|Spend|Golden|LifecycleWrapper)' -count=1   ok
go test ./cmd -run 'Test(DocumentedSubcommandsAreSeverityClassified|PlatformParityGolden|RegressionSnapshot|
                         HumanFacingOutputGoesThroughWriteVisualOutput)' -count=1                                       ok
go test ./cmd -run 'Test(NoTokenCountIsDerivedFromLength|SpendLedgerCarriesNoCurrencyField)' -count=1                   ok
go build ./...                                                                                                          ok
go vet ./cmd/ ./pkg/codex/                                                                                              ok
gofmt -l cmd/ pkg/                                                                                                      no output
build closeout and check closeout rendered by hand and read as plain English                                            READS CLEAN
```

Both new guards were additionally proved to fail when unmet, then reverted and the tree re-verified clean:

```
planted "0" in place of the dash sentinel:
  spend_cost_line_test.go:135: Roam-90's figure cell = "0", want exactly the dash sentinel "—"
  spend_cost_line_test.go:135: Guess-11's figure cell = "0", want exactly the dash sentinel "—"

planted a second block on the ending screen:
  ceremony_closeout_spend_test.go:89: the build's ending screen carries 2 cost line block(s), want exactly 1
  ceremony_closeout_spend_test.go:111: the check's ending screen carries 2 cost line block(s), want exactly 1
  TestNoLaneRendersTwoCostLines/the_wrapper_lane... FAIL
```

---
*Phase: 196-see-what-it-cost*
*Completed: 2026-08-28*
