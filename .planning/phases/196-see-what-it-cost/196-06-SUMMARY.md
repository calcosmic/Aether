---
phase: 196-see-what-it-cost
plan: 06
subsystem: infra
tags: [spend-ledger, token-usage, cobra, read-only, go, tdd]

requires:
  - phase: 196-see-what-it-cost
    provides: "196-01's spendLedger, loadSpendLedgersForPhase, computeSpendTotals, spendRow.JobName"
  - phase: 174-spend
    provides: "codex.WorkerUsage, BilledTotalTokens, the measured/estimated Source split"
provides:
  - "`aether spend` — the read-only per-worker token view, keyed by phase, spanning both the build and the continue ledger"
  - "TestSpendDoesNotMutate — a whole-store content-hash proof that the inspection path writes nothing, with a planted-write subtest so it cannot pass vacuously"
  - "The dash sentinel rendered for any worker whose tool reported nothing, asserted on the PARSED FIGURE CELL rather than the whole row"
  - "Reachability for the command through a plain-English sentence in all six build/continue wrapper copies, with no allowlist entry and no command definition file"
affects: [196-07, 196-08]

actuals:
  tokens: 6783
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "A rendered owner-facing row is parsed into cells by a two-or-more-space separator, so a guard can assert one column without matching the whole line"
    - "An inspection command's no-mutation property is proved by hashing the whole store across two runs, plus a subtest that plants a real write to prove the snapshot is not blind"
    - "Figures are printed exactly as recorded — no separators, no 1.2M shortening — because abbreviation is arithmetic the view is forbidden to do"

key-files:
  created:
    - cmd/spend_cmd.go
    - cmd/spend_cmd_test.go
  modified:
    - cmd/testdata/command_catalog.json
    - .claude/commands/ant/build.md
    - .claude/commands/ant-build.md
    - .opencode/commands/ant/build.md
    - .claude/commands/ant/continue.md
    - .claude/commands/ant-continue.md
    - .opencode/commands/ant/continue.md

key-decisions:
  - "Figures print exactly as recorded — 1200000, not 1.2M and not 1,200,000 — because shortening or grouping is arithmetic this view is forbidden to perform, and the whole promise of the detail view is that every number in it can be checked against its recorded row unchanged. D-01's compact shape belongs to the closeout headline in 196-07, not here."
  - "The command reads the current phase through loadColonyState, deliberately NOT loadActiveColonyState: the latter can persist a legacy-shape repair to COLONY_STATE.json, which would make an inspection command a writer on some stores and not others — the exact silent-mutation shape TestSpendDoesNotMutate exists to forbid."
  - "An ESTIMATED row is rendered with the dash sentinel exactly like an empty one. D-01 as amended forbids rendering an estimate at all, and prompt-character-count is the only estimate mechanism in the tree, so showing one — even labelled — would be precisely what success criterion 3 forbids."
  - "The rendered row is parsed by a two-or-more-space cell separator and the job name is whitespace-collapsed before rendering, so no cell can ever contain a run of two spaces. This is what lets the dash-sentinel guard assert the figure column alone; a whole-row 'contains no digit' check would fail on correct output, because deterministic worker names carry digits (Scout Roam-90)."
  - "The test harness seeds a COLONY_STATE.json for every case, because the root command's own first-run welcome banner writes a .welcomed marker when no colony exists. That write belongs to cmd/ux_firstrun.go, not to this command, and letting it fire would have made the no-mutation proof assert something it does not own."

patterns-established:
  - "Cell-parsed row assertions: a guard on an owner-facing table asserts the specific column it cares about, never the whole line, so correct output containing incidental digits cannot force the guard to be weakened later"
  - "Anti-vacuity subtest on a snapshot guard: the same test that proves nothing changed also plants a real change and requires the snapshot to notice it"

requirements-completed: [COST-03]

coverage:
  - id: D1
    description: "Running `aether spend` shows per-worker token usage for a phase, spanning both the build-keyed and continue-keyed ledgers, with neither erasing the other"
    requirement: "COST-03"
    verification:
      - kind: unit
        ref: "cmd/spend_cmd_test.go#TestSpendShowsEveryWorkerRow"
        status: pass
      - kind: unit
        ref: "cmd/spend_cmd_test.go#TestSpendShowsBuildAndContinueRows"
        status: pass
    human_judgment: false
  - id: D2
    description: "Running the command changes no file on disk — the inspection path does not mutate state"
    requirement: "COST-03"
    verification:
      - kind: unit
        ref: "cmd/spend_cmd_test.go#TestSpendDoesNotMutate"
        status: pass
      - kind: manual_procedural
        ref: "find $STORE -type f -exec shasum {} \\; compared before and after a hand run against a seeded store"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every figure shown comes from a recorded ledger row; the command computes no figure of its own"
    requirement: "COST-03"
    verification:
      - kind: unit
        ref: "cmd/spend_cmd_test.go#TestSpendShowsEveryWorkerRow (seeded literals asserted; no unseeded figure permitted in any figure cell)"
        status: pass
      - kind: unit
        ref: "cmd/spend_no_length_derivation_test.go#TestNoTokenCountIsDerivedFromLength"
        status: pass
    human_judgment: false
  - id: D4
    description: "D-01 as amended: a worker whose tool reported nothing shows no number at all, and the total counts only measured workers"
    requirement: "COST-03"
    verification:
      - kind: unit
        ref: "cmd/spend_cmd_test.go#TestSpendShowsNoNumberForUnreportedRows"
        status: pass
      - kind: unit
        ref: "cmd/spend_cmd_test.go#TestSpendEmptyLedgerShowsNoFigure"
        status: pass
    human_judgment: false
  - id: D5
    description: "D-06: the command is reachable through a plain sentence in the build and continue wrapper output, with no command definition file and no allowlist entry"
    requirement: "COST-03"
    verification:
      - kind: unit
        ref: "cmd/subcommand_reachability_ratchet_test.go#TestNoRegisteredSubcommandIsUnreferenced"
        status: pass
      - kind: unit
        ref: "cmd/subcommand_reachability_ratchet_test.go#TestOrphanAllowlistOnlyShrinks"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_wrapper_contract_test.go#TestLifecycleFlatMirrorsMatchCanonical"
        status: pass
    human_judgment: false
  - id: D6
    description: "The rendered view reads as plain English to a non-technical owner, with repo vocabulary translated inline"
    verification: []
    human_judgment: true
    rationale: "Whether the wording lands for the owner is his call, not a test's. The runnable part — that the figures, the sentinel and the total are correct — is covered by D1–D4; the readability of the sentences around them is judgment."

duration: 41 min
completed: 2026-08-28
status: complete
---

# Phase 196 Plan 06: The read-only spend view Summary

**`aether spend` lists every worker's recorded token figure for a phase across both the build and continue ledgers, proves by whole-store content hash that it writes nothing, and shows a dash instead of a number for any worker whose tool reported none.**

## Performance

- **Duration:** 41 min
- **Started:** 2026-08-28T09:52:00Z
- **Completed:** 2026-08-28T10:33:00Z
- **Tasks:** 2
- **Files modified:** 9 (2 created, 7 modified)

## Accomplishments

- **The per-worker view exists and reads only.** `aether spend [--phase N]` loads a phase's rows through `loadSpendLedgersForPhase`, takes the total straight off `computeSpendTotals`, and takes each row's figure straight off `BilledTotalTokens()`. It constructs no row, recomputes no total, and calls no save anywhere.
- **The no-mutation property is proved, not promised.** `TestSpendDoesNotMutate` hashes every file under the store, runs the command twice, and requires both the hashes and stdout to be unchanged. It carries an anti-vacuity subtest that plants a real ledger write and fails if the snapshot does not notice it.
- **An unreported worker shows no number.** Both an empty-usage row and an *estimated* row render the dash sentinel and the words "not reported", and neither enters the total. The assertion is on the parsed figure cell, so it survives worker names that carry digits of their own.
- **The command cannot become another orphan.** The reachability ratchet named `aether spend` as having no caller before the wrapper edit and passes after it — with no allowlist entry, and per D-06 no command definition file.

## Task Commits

1. **Task 1 (RED): failing tests for the read-only spend view** — `e5b15839` (test)
2. **Task 1 (GREEN): the read-only per-worker spend view** — `f4ba6197` (feat)
3. **Task 2: name the spend view in the build and continue wrappers** — `dc08ccda` (docs)

No refactor commit was needed — the implementation landed at its final shape apart from the column-alignment work, which was part of the same green step.

## Files Created/Modified

- `cmd/spend_cmd.go` — the command, its owner-facing renderer, its JSON surface, and the phase resolution that uses a pure read
- `cmd/spend_cmd_test.go` — the five named guards plus their two anti-vacuity subtests and the shared row-cell parser
- `cmd/testdata/command_catalog.json` — refreshed golden, recording `spend` and its `--phase` flag
- `.claude/commands/ant/build.md`, `.claude/commands/ant-build.md`, `.opencode/commands/ant/build.md` — the two-line mention in **After the Build**, byte-identical across all three
- `.claude/commands/ant/continue.md`, `.claude/commands/ant-continue.md`, `.opencode/commands/ant/continue.md` — the same two lines in **After Continue**, byte-identical across all three

## Decisions Made

See `key-decisions` in the frontmatter. The two that a later reader is most likely to want the reasoning for:

**Exact figures, not compact ones.** D-01's worked example shows `1.2M` / `220K`. That shape belongs to the one-line closeout in plan 196-07, which is a headline. This is the *detail* view, and its entire value is that a reader can hold a number in it against the recorded row and see them match. Abbreviating is arithmetic; this command is forbidden from doing arithmetic of its own, so it prints `1200000`.

**An estimate is treated as "not reported".** `computeSpendTotals` keeps measured and estimated in separate fields, so the ledger can still carry an estimate. The renderer refuses to show one. D-01's amendment is explicit that no estimated figure is ever rendered, and since prompt-character-count is the only estimate mechanism in the tree, showing one — even beside an honest label — would be exactly what success criterion 3 forbids, wearing a label.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The root command's first-run banner polluted the empty-ledger test**

- **Found during:** Task 1 (green step)
- **Issue:** `TestSpendEmptyLedgerShowsNoFigure` ran against a completely empty store, so `checkAndEmitFirstRun` (`cmd/ux_firstrun.go`) fired its welcome banner into the captured stdout — three lines that parse as worker rows — and wrote a `.welcomed` marker file. The banner is the root command's behaviour, not this command's.
- **Fix:** The shared test setup now seeds a minimal `COLONY_STATE.json`, which is also the only state in which anyone would ever run this command. This suppresses the banner and, importantly, keeps `TestSpendDoesNotMutate` honest — the test now asserts only over writes this command could be responsible for, rather than accidentally asserting over an unrelated first-run marker.
- **Files modified:** `cmd/spend_cmd_test.go`
- **Verification:** all five named tests pass; the planted-write subtest still fails the snapshot when a write is introduced.
- **Committed in:** `f4ba6197`

**2. [Rule 2 - Missing Critical] Row cells needed a guaranteed separator to make the sentinel assertion sound**

- **Found during:** Task 1 (green step)
- **Issue:** The plan requires the dash assertion to be made on the parsed figure cell. That is only sound if a cell can never itself contain the separator. `spendRow.JobName` is free text and could carry a run of two spaces, which would have split one cell into two and silently shifted the figure column.
- **Fix:** The job name is whitespace-collapsed with `strings.Fields` before rendering, and the invariant is documented on `spendCellSeparator`.
- **Files modified:** `cmd/spend_cmd.go`
- **Verification:** `TestSpendShowsEveryWorkerRow` fails the parse (`want at least 3 cells`) if a row ever renders fewer than three cells.
- **Committed in:** `f4ba6197`

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical)
**Impact on plan:** Both were necessary for the plan's own guards to mean what they say. No scope creep — nothing outside `cmd/spend_cmd*.go` changed as a result.

## Issues Encountered

**The orphan ratchet's failing test is not the one the plan names.** The plan's Task 1 acceptance criteria cite `TestOrphanAllowlistOnlyShrinks`, which passed both before and after the wrapper edit — it only guards the allowlist's direction. The test that actually detects an uncalled command is `TestNoRegisteredSubcommandIsUnreferenced`, and it reported `aether spend is registered but nothing calls it` before the wrapper edit and passed after it. Both were run; the second is the one that carries the evidence, and it is the one recorded in the coverage block.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Plan 196-07 (the one closeout line) can now assume a detail view exists and is named by both wrappers, so its own line does not have to carry per-worker breakdown.
- The `spendRowReportedUsage` predicate and the dash sentinel are the shape 196-07's closeout renderer should reuse, so the headline and the detail view cannot disagree about which workers counted.
- No blockers.

## Self-Check: PASSED

Files claimed as created exist on disk:

```
FOUND: cmd/spend_cmd.go
FOUND: cmd/spend_cmd_test.go
```

Commits claimed exist in git:

```
FOUND: e5b15839  test(196-06): add failing tests for the read-only spend view
FOUND: f4ba6197  feat(196-06): add the read-only per-worker spend view
FOUND: dc08ccda  docs(196-06): name the spend view in the build and continue wrappers
```

Plan-level verification re-run at close-out:

```
go test ./cmd -run 'Test(Spend|AuditCatalogGolden|LifecycleWrapper|Orphan|NoRegisteredSubcommandIsUnreferenced)' -count=1   ok
go build ./...                                                                                                             ok
go vet ./cmd/ ./pkg/codex/                                                                                                 ok
gofmt -l cmd/ pkg/                                                                                                         no output
hand run against a seeded store, store hashed before and after                                                             STORE UNCHANGED
```

Both guards were additionally proved to fail when unmet: planting a `store.SaveJSON` inside the command failed `TestSpendDoesNotMutate` (`file count changed: before=3 after=4`), and rendering `"0"` in place of the sentinel failed `TestSpendShowsNoNumberForUnreportedRows` for both unreported workers. Both plants were reverted and the tree re-verified clean.

---
*Phase: 196-see-what-it-cost*
*Completed: 2026-08-28*
