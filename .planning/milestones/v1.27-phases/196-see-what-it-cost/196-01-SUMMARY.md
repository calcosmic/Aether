---
phase: 196-see-what-it-cost
plan: 01
subsystem: infra
tags: [spend-ledger, token-usage, storage, go, tdd]

requires:
  - phase: 174-spend
    provides: codex.WorkerUsage, BilledTotalTokens, the spend/ store subdirectory
  - phase: 195-coherent-jobs
    provides: codexBuildDispatch.JobName / CoveredTaskIDs — one worker owning several tasks
provides:
  - Durable per-worker, per-workflow spend rows under spend/phase-<N>-build.json and spend/phase-<N>-continue.json
  - computeSpendTotals with measured and estimated subtotals kept in separate fields and no currency amount anywhere
  - spendRow.JobName, so a Phase 195 grouped job is attributable in the ledger
  - A closed dispatch status vocabulary on ledger rows, refused by name on save
affects: [196-02, 196-03, 196-04, 196-05, 196-06, 196-07, 196-08]

actuals:
  tokens: 14657
  tasks: 2
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Aggregation invariants proved against hand-computed literals, with an AST guard that fails if the test ever calls the helper it checks"
    - "Structural currency-field ban asserted by reflection over the Go types and over the marshalled JSON keys, never by searching source text"
    - "Fail-closed save: every row validated before anything is written, so a refused ledger leaves no file"

key-files:
  created:
    - cmd/spend_ledger.go
    - cmd/spend_ledger_test.go
  modified: []

key-decisions:
  - "spendTotals.ProviderUSD and ProviderUSDRows deleted rather than kept behind a renderer test — nothing read them, and an unread money field on the totals struct is a standing invitation to put one on the headline (D-01, FIX 1-1)"
  - "The currency ban is scoped to the ledger's own types and skips codex.WorkerUsage by name; the shared provider-usage type keeps its USDCost for callers outside this phase, and this plan is forbidden from changing it"
  - "An empty row status is refused rather than folded to failed — normalizeRuntimeDispatchStatus reads an unstated status as failed, which is right for a live summary but would put a wrong fact in a durable ledger"
  - "Status synonyms fold onto one canonical word on a copy of the rows, so validation never rewrites the caller's slice underneath it"
  - "The salvaged source-tag vocabulary was left untouched, as the plan directs — its nine tests pass as written"

patterns-established:
  - "AST-based test guards: TestRawColumnInvariantDoesNotCallTheBilledTotalHelper walks the syntax tree, so a comment naming the helper can neither satisfy nor break it"
  - "Currency-name detection by word-splitting an identifier rather than substring matching, so RecordedAt and WorkerCount cannot trip it"

requirements-completed: [COST-03, COST-05]

coverage:
  - id: D1
    description: "The salvaged spend ledger is in the tree, compiling, with its nine original tests still green (D-03)"
    requirement: "COST-03"
    verification:
      - kind: unit
        ref: "cmd/spend_ledger_test.go#TestSpendLedgerPersistsAcrossProcesses,TestSpendLedgerContinueDoesNotEraseBuild,TestSpendLedgerSameWorkflowRerunReplaces,TestSpendLedgerLoadEdgeCases,TestLedgerMeasuredEstimatedSubtotalsAreSeparate,TestLedgerGrandTotalEqualsSumOfRows,TestLedgerRefusesDerivedMetricOverEstimates,TestLedgerParentRollupCountsEachWorkerOnce,TestLedgerPhaseTotalAddsBuildAndContinue"
        status: pass
    human_judgment: false
  - id: D2
    description: "No field on the ledger or its totals relays a currency amount, so no later renderer can put one on the headline (D-01, FIX 1-1)"
    requirement: "COST-05"
    verification:
      - kind: unit
        ref: "cmd/spend_ledger_test.go#TestSpendLedgerCarriesNoCurrencyField"
        status: pass
    human_judgment: false
  - id: D3
    description: "The aggregation invariant is proven against hand-computed literals, not against the helper the implementation calls (FIX 1-2)"
    requirement: "COST-03"
    verification:
      - kind: unit
        ref: "cmd/spend_ledger_test.go#TestLedgerGrandTotalFromRawColumnsWithoutHelper"
        status: pass
      - kind: unit
        ref: "cmd/spend_ledger_test.go#TestRawColumnInvariantDoesNotCallTheBilledTotalHelper"
        status: pass
    human_judgment: false
  - id: D4
    description: "A row records the job name a Phase 195 grouped worker owned; an ungrouped worker records none (FIX 1-3)"
    requirement: "COST-03"
    verification:
      - kind: unit
        ref: "cmd/spend_ledger_test.go#TestSpendRowCarriesJobName"
        status: pass
      - kind: unit
        ref: "cmd/spend_ledger_test.go#TestSpendRowWithoutJobNameOmitsIt"
        status: pass
    human_judgment: false
  - id: D5
    description: "A row's status comes from the dispatch status vocabulary already in the tree, not a second free-form one (FIX 1-4)"
    requirement: "COST-03"
    verification:
      - kind: unit
        ref: "cmd/spend_ledger_test.go#TestSpendLedgerRefusesUnknownRowStatus"
        status: pass
    human_judgment: false

duration: 22min
completed: 2026-08-28
status: complete
---

# Phase 196 Plan 01: Salvaged Spend Ledger Summary

**Durable per-worker, per-workflow token ledger landed from the abandoned branch with the provider currency relay deleted, Phase 195 grouped-job attribution added, an aggregation invariant proved against hand-summed literals under an AST guard, and one dispatch status vocabulary enforced fail-closed on save.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-08-28T09:30:00Z
- **Completed:** 2026-08-28T09:52:00Z
- **Tasks:** 2
- **Files modified:** 2 (both created)

## Accomplishments

- The salvaged ledger (`worktree-agent-a47f78913caf6fde0`) is in the tree at file level and its nine original tests pass unchanged, confirming the salvage assessment's compile-and-green claim held against today's tree.
- `spendTotals.ProviderUSD` and `ProviderUSDRows` are gone, along with their accumulation. `codex.WorkerUsage` is untouched and keeps its own `USDCost`.
- `TestSpendLedgerCarriesNoCurrencyField` walks the ledger types by reflection **and** their marshalled JSON keys, failing by name if a money field is ever reintroduced.
- `TestLedgerGrandTotalFromRawColumnsWithoutHelper` asserts 102,550 and 108,050 as literals summed by hand from the four raw disjoint columns, and `TestRawColumnInvariantDoesNotCallTheBilledTotalHelper` parses the test file's syntax tree so it can never quietly start calling `BilledTotalTokens()` again.
- `spendRow.JobName` (omit-when-empty) records which grouped Phase 195 job a worker's tokens belong to; the schema version constant is unchanged, so no migration is owed.
- Row status is validated on save against the dispatch vocabulary `dispatchStatusIcon` already renders. An unknown or empty word is refused by name, naming the worker, and no file is written.

## Task Commits

1. **Task 1 (salvage landing)** — `cd80dd73` (feat) — salvaged files in unchanged, nine tests green recorded as evidence
2. **Task 1 RED** — `4a5db992` (test) — currency lock failing, aggregation invariant + AST guard added
3. **Task 1 GREEN** — `83ee6f0a` (feat) — currency relay removed from `spendTotals`
4. **Task 2 RED** — `5185d076` (test) — job-attribution and status tests, package does not compile
5. **Task 2 GREEN** — `5c2e2fae` (feat) — `JobName` field and status vocabulary validation

## RED evidence (real output)

**Task 1 RED** — `go test ./cmd -run 'TestLedgerGrandTotalFromRawColumnsWithoutHelper|TestRawColumnInvariantDoesNotCallTheBilledTotalHelper|TestSpendLedgerCarriesNoCurrencyField' -count=1 -v`:

```
--- PASS: TestLedgerGrandTotalFromRawColumnsWithoutHelper (0.00s)
--- PASS: TestRawColumnInvariantDoesNotCallTheBilledTotalHelper (0.00s)
=== RUN   TestSpendLedgerCarriesNoCurrencyField
    spend_ledger_test.go:783: field spendTotals.ProviderUSD names a money amount; the ledger and its totals must carry none (D-01: no currency figure may reach the cost line)
    spend_ledger_test.go:787: field spendTotals.ProviderUSD serializes as "provider_usd", which names a money amount
    spend_ledger_test.go:783: field spendTotals.ProviderUSDRows names a money amount; the ledger and its totals must carry none (D-01: no currency figure may reach the cost line)
    spend_ledger_test.go:787: field spendTotals.ProviderUSDRows serializes as "provider_usd_rows", which names a money amount
    spend_ledger_test.go:818: serialized key spendTotals.provider_usd_rows names a money amount
    spend_ledger_test.go:818: serialized key spendTotals.provider_usd names a money amount
--- FAIL: TestSpendLedgerCarriesNoCurrencyField (0.00s)
FAIL	github.com/calcosmic/Aether/cmd	0.849s
```

The aggregation invariant passed on arrival, and that is the honest result: FIX 1-2 asks for a lock on arithmetic that is already correct, not for a new behaviour. To prove that lock is not decorative, its AST guard was deliberately violated (a `BilledTotalTokens()` call inserted into the guarded function) and re-run:

```
--- FAIL: TestRawColumnInvariantDoesNotCallTheBilledTotalHelper (0.00s)
    spend_ledger_test.go:687: TestLedgerGrandTotalFromRawColumnsWithoutHelper calls BilledTotalTokens() -- its expected totals must be hand-computed literals, never produced by the arithmetic it is checking (the 186x-undercount shape)
FAIL	github.com/calcosmic/Aether/cmd	0.703s
```

The violation was then reverted and the guard returned to `ok`.

**Task 2 RED** — `go test ./cmd -run 'Test(SpendRowCarriesJobName|SpendRowWithoutJobName|SpendLedgerRefusesUnknownRowStatus)' -count=1`:

```
# github.com/calcosmic/Aether/cmd [github.com/calcosmic/Aether/cmd.test]
cmd/spend_ledger_test.go:907:5: unknown field JobName in struct literal of type spendRow
cmd/spend_ledger_test.go:924:20: loaded.Rows[0].JobName undefined (type spendRow has no field or method JobName)
cmd/spend_ledger_test.go:926:19: loaded.Rows[0].JobName undefined (type spendRow has no field or method JobName)
FAIL	github.com/calcosmic/Aether/cmd [build failed]
```

## Acceptance criteria — commands run and results

**Task 1**

| Criterion | Command | Result |
|---|---|---|
| Nine salvaged tests pass | `go test ./cmd -run 'TestSpendLedgerPersistsAcrossProcesses\|TestSpendLedgerContinueDoesNotEraseBuild\|TestSpendLedgerSameWorkflowRerunReplaces\|TestSpendLedgerLoadEdgeCases\|TestLedgerMeasuredEstimatedSubtotalsAreSeparate\|TestLedgerGrandTotalEqualsSumOfRows\|TestLedgerRefusesDerivedMetricOverEstimates\|TestLedgerParentRollupCountsEachWorkerOnce\|TestLedgerPhaseTotalAddsBuildAndContinue' -count=1` | PASS (`ok ... 0.713s`) |
| Raw-column invariant exists, asserts 102,550 as a literal, no helper call in its source | `awk '/^func TestLedgerGrandTotalFromRawColumnsWithoutHelper/,/^}$/' cmd/spend_ledger_test.go \| grep -c 102550` → `5`; same range grepped for `BilledTotalTokens\|TotalInputTokens\|billedTotal` → `0` | PASS |
| Currency lock fails if a money field returns | `go test ./cmd -run TestSpendLedgerCarriesNoCurrencyField -count=1` | PASS (and demonstrated failing at RED above) |
| `go vet ./cmd` clean | `go vet ./cmd` | PASS (exit 0, no output) |

**Task 2**

| Criterion | Command | Result |
|---|---|---|
| Job-name and status tests pass | `go test ./cmd -run 'TestSpendRowCarriesJobName\|TestSpendRowWithoutJobNameOmitsIt\|TestSpendLedgerRefusesUnknownRowStatus' -count=1` | PASS (`ok ... 0.629s`) |
| Grouped fixture round-trips job name; ungrouped serializes no key | asserted on the on-disk file bytes inside those two tests | PASS |
| Refusal names the status and the worker, writes no file | asserted in `TestSpendLedgerRefusesUnknownRowStatus` (`os.Stat` → `IsNotExist`) | PASS |
| Schema version constant unchanged | `grep spendLedgerSchemaVersion cmd/spend_ledger.go` → `= 1`; `git show cd80dd73:cmd/spend_ledger.go` → `= 1` | PASS |

**Plan-level verification**

| Command | Result |
|---|---|
| `go test ./cmd -run 'Test(SpendLedger\|SpendRow\|Ledger)' -count=1` | `ok  github.com/calcosmic/Aether/cmd  0.601s` |
| `go build ./cmd/aether` | exit 0 |
| `go build ./...` | exit 0 |
| `go vet ./cmd/ ./pkg/codex/` | exit 0 |
| `gofmt -l cmd/ pkg/` | no output |
| Diff read for surviving currency fields | Only prose in comments explaining the removal; no field, tag or JSON key |

## Files Created/Modified

- `cmd/spend_ledger.go` — per-worker, per-workflow durable spend rows; measured/estimated subtotals kept apart; no currency field; `JobName` attribution; fail-closed status validation on save
- `cmd/spend_ledger_test.go` — the nine salvaged tests plus five new ones covering the four assessment fixes

## Decisions Made

- **Delete the currency relay rather than gate it.** FIX 1-1 offered either dropping `ProviderUSD`/`ProviderUSDRows` or keeping them behind a renderer test. Dropped, as the assessment preferred: nothing read them, and D-01 forbids a money figure on the cost line.
- **Scope the currency ban to the ledger's own types.** `codex.WorkerUsage` is skipped by name in both the reflection walk and the JSON walk, documented in the test. It is the shared provider-usage type, it keeps `USDCost` for callers outside this phase, and the plan forbids touching it. The boundary being defended is `spendTotals`, which is what a headline renderer reads.
- **Refuse an empty status.** `normalizeRuntimeDispatchStatus` reads an unstated status as `failed`. That is the right default for a live summary but would write a wrong fact into a durable ledger, so the writer is made to state an outcome.
- **Make the FIX 1-2 independence executable.** CLAUDE.md's Definition of Done requires a command that fails when the requirement is unmet, so the "must not call the helper" rule is enforced by an AST walk rather than left as a comment. The walk uses the syntax tree, not text search, in line with the repo's "structure, not substrings" rule.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added an AST guard for the aggregation invariant's independence**

- **Found during:** Task 1
- **Issue:** The plan's acceptance criterion "its source contains no call to the billed-total helper" had no executable enforcement. A later edit could reintroduce the exact 186x-undercount shape and nothing would fail — which is precisely the condition CLAUDE.md's Definition of Done exists to forbid.
- **Fix:** Added `TestRawColumnInvariantDoesNotCallTheBilledTotalHelper`, which parses `spend_ledger_test.go` with `go/parser` and fails if any call inside the guarded function resolves to `BilledTotalTokens`, `TotalInputTokens` or `billedTotal`.
- **Files modified:** `cmd/spend_ledger_test.go`
- **Verification:** Guard deliberately violated and observed failing, then reverted and observed passing (output quoted above).
- **Committed in:** `4a5db992`

**2. [Rule 2 - Missing Critical] Refuse an empty row status, not only an unknown one**

- **Found during:** Task 2
- **Issue:** The plan's behaviour line refuses a status "outside the dispatch status vocabulary". The empty string is outside it, but the package's existing normalizer silently maps empty to `failed` — so a writer that forgot to set a status would durably record a healthy worker as failed.
- **Fix:** `normalizeSpendRowStatus` returns not-ok for the empty string before any folding, and `saveSpendLedger` emits a distinct refusal naming the worker.
- **Files modified:** `cmd/spend_ledger.go`, `cmd/spend_ledger_test.go`
- **Verification:** `TestSpendLedgerRefusesUnknownRowStatus/an_empty_status_is_refused_rather_than_silently_read_as_failed` PASS, with an `os.Stat` check that no file was written.
- **Committed in:** `5185d076` (test), `5c2e2fae` (implementation)

**3. [Rule 2 - Missing Critical] Validate on a copy so a save never mutates the caller's rows**

- **Found during:** Task 2
- **Issue:** Folding status synonyms in place would mutate the caller's slice through the shared backing array — a silent side effect on a value the caller still holds.
- **Fix:** `saveSpendLedger` copies `entry.Rows` before normalizing, and a subtest asserts the caller's slice is unchanged after a successful save.
- **Files modified:** `cmd/spend_ledger.go`, `cmd/spend_ledger_test.go`
- **Verification:** `TestSpendLedgerRefusesUnknownRowStatus/validating_a_save_does_not_mutate_the_caller's_rows` PASS.
- **Committed in:** `5c2e2fae`

---

**Total deviations:** 3 auto-fixed (3 missing-critical). **Impact on plan:** All three harden requirements the plan already stated; none expands scope. No architectural change, no new dependency, no file outside the plan's `files_modified`.

## Issues Encountered

None. The salvaged branch compiled and passed green against today's tree exactly as the assessment recorded, so no rescue work was needed.

## Known Stubs

None. Every symbol added in this plan is exercised by a test in the same commit. The ledger as a whole still has no production writer — that is plan 196-05's stated work, not a stub introduced here, and it is why this plan carries no wiring test.

## Threat Flags

None. This plan adds no network endpoint, no auth path, and no new file-access pattern; it writes only through the existing `pkg/storage.Store` under the `spend/` subdirectory that `cmd/spend_session_capture.go` already uses.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The names plan 196-02 onward depend on are all present and stable: `spendRow` (with `JobName`), `spendLedger`, `saveSpendLedger`, `loadSpendLedger`, `loadSpendLedgersForPhase`, `computeSpendTotals`, `spendPerWorkerAverageTokens`, `spendRollupByParent`.
- The store layout is unchanged: `spend/phase-<N>-build.json` and `spend/phase-<N>-continue.json`.
- The salvaged source-tag vocabulary is unchanged, as the plan directs. After 196-02 removes the character-derived producer, the tag reading `estimate` will mean "not measured" and `EstimatedTokens` will be empty by construction.
- Any writer added later must set a row status from `spendRowStatusVocabulary()`; a save with an unstated or unknown status now fails loudly rather than storing it.

## Self-Check: PASSED

- `cmd/spend_ledger.go` — FOUND on disk
- `cmd/spend_ledger_test.go` — FOUND on disk
- `cd80dd73`, `4a5db992`, `83ee6f0a`, `5185d076`, `5c2e2fae` — all FOUND in `git log`
- All Task 1 and Task 2 acceptance criteria re-run after the final commit; all pass (tables above)
- Plan-level verification re-run after the final commit: tests ok, `go build ./...` exit 0, `go vet ./cmd/ ./pkg/codex/` exit 0, `gofmt -l cmd/ pkg/` empty

---
*Phase: 196-see-what-it-cost*
*Completed: 2026-08-28*
