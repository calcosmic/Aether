---
phase: 196-see-what-it-cost
criterion: 5 (three abandoned branches reviewed, each merged or deleted with a written reason)
assessed: 2026-08-27
assessor: adversarial code review (read-only; no branch was merged, checked out, or modified)
method: files extracted with `git show` / `git archive` into an isolated scratch copy of the
  current `oracle-reinstate` tree, then `go vet ./cmd/` and `go test ./cmd/ -timeout 300s`
  (no `-race`, per instruction). Parser assumptions additionally validated against the real
  `~/.claude/projects/*.jsonl` transcripts and the real
  `~/.local/share/opencode/storage/` session store on this machine.
---

# Phase 196 — Salvage Assessment of Three Abandoned Spend Branches

## Verdict table

| Branch | Files | Verdict | One-line reason |
|---|---|---|---|
| `worktree-agent-a47f78913caf6fde0` | `cmd/spend_ledger.go` (266) + test (530) | **MERGE-WITH-FIXES** | Compiles and passes green against today's tree unchanged; it is the durable storage layer criterion 3 needs, and rewriting it would cost more than the two small fixes it needs. |
| `worktree-agent-aa57076cd698da76f` | `cmd/wrapper_usage_opencode.go` (441) + test (230) + 8 fixture files | **MERGE-WITH-FIXES** | Compiles and passes green, and its central assumptions are confirmed correct against the owner's real OpenCode store — but it carries one concurrency bug that will crash the process the moment Phase 196 wires it, and it is untested on the multi-message case that is the only real-world case. |
| `worktree-agent-a59fd3ee68644ea21` | `cmd/wrapper_usage_claude_test.go` (232) + fixture | **DELETE (harvest the fixture and the test file into the plan first)** | It is a test file for an implementation that has never existed in any ref in this repository. Merged as-is it does not fail — it breaks the build of the entire `cmd` package. Its fixture also encodes a shape that no longer matches today's real transcripts and omits the duplication that would produce a 2.1× overcount. |

Net: **two merges, one deletion.** None left dangling.

---

## Branch 1 — `worktree-agent-a47f78913caf6fde0` (the ledger)

### Verdict: MERGE-WITH-FIXES

### What it delivers

Toward **criterion 3** (`aether spend` shows per-worker usage without mutating anything,
and no number is guessed from text length) it delivers the entire data layer and none of
the command:

- `spendRow` / `spendLedger` — per-worker rows carrying `codex.WorkerUsage` verbatim,
  keyed by phase **and** workflow (`spend/phase-<N>-build.json`,
  `spend/phase-<N>-continue.json`), so a continue cannot erase a build's rows. The
  reasoning for the two-file key is written out in full in the file and is correct.
- `saveSpendLedger` / `loadSpendLedger` / `loadSpendLedgersForPhase` — schema-versioned,
  fail-closed reads. A schema mismatch or malformed file is skipped whole, never returned
  half-populated.
- `computeSpendTotals` — keeps measured and estimated subtotals in **separate fields**
  that are never summed into anything presented as a measurement. This is exactly the
  honesty split criterion 1 asks for.
- `spendPerWorkerAverageTokens` — refuses by name to compute a derived average over a row
  set containing estimates unless explicitly told to. That is Definition-of-Done shaped:
  a command that fails when the requirement is unmet.
- `spendRollupByParent` — deterministic per-parent roll-up, unattributed rows still visible.

Toward **criterion 1** it delivers nothing: nothing calls any of it, no build or continue
line is rendered, and no row is ever written.

### Compiles today? Yes, unchanged.

Dropped into a scratch copy of the current tree: `go vet ./cmd/` clean, and all nine tests
pass. It touches none of the shapes Phase 195 changed — it never sees `codexBuildDispatch`,
`CoveredTaskIDs`, `CompletedTaskIDs` or `TaskClaims`. It depends only on `pkg/codex.WorkerUsage`
and `pkg/storage.Store`, both of which are unchanged since 2026-08-14.

```
--- PASS: TestSpendLedgerPersistsAcrossProcesses
--- PASS: TestSpendLedgerContinueDoesNotEraseBuild
--- PASS: TestSpendLedgerSameWorkflowRerunReplaces
--- PASS: TestSpendLedgerLoadEdgeCases
--- PASS: TestLedgerMeasuredEstimatedSubtotalsAreSeparate
--- PASS: TestLedgerGrandTotalEqualsSumOfRows (4 sub-cases)
--- PASS: TestLedgerRefusesDerivedMetricOverEstimates
--- PASS: TestLedgerParentRollupCountsEachWorkerOnce
--- PASS: TestLedgerPhaseTotalAddsBuildAndContinue
```

It also composes with the ledger that already landed: `spendSessionRel = "spend/session.json"`
(`cmd/spend_session_capture.go`, already on `oracle-reinstate`) sits in the same store
subdirectory with no collision.

### Duplicate of anything Phase 195 built? No.

Phase 195's `codex.TaskReceipt` vocabulary is **task**-scoped completion evidence
(`TaskID`, `FilesCreated`, `TestsWritten`). The ledger is **worker**-scoped cost
(`AgentName`, `Caste`, `ParentName`, `Usage`). No overlap, no second evidence path.

### Are the tests real? Mostly yes.

This is not the 186×-undercount pattern CLAUDE.md warns about.
`TestSpendLedgerPersistsAcrossProcesses` seeds Anthropic's documented disjoint-column
example (50 / 100,000 / 2,000 / 500), constructs a **second `storage.NewStore` over the
same directory** to simulate the process boundary honestly rather than re-reading a struct
in RAM, and asserts the hardcoded literal 102,550. `TestLedgerMeasuredEstimatedSubtotalsAreSeparate`
asserts hardcoded 30,000 / 500 / 30,500.

One real weakness (see WARNING W1-1 below): `TestLedgerGrandTotalEqualsSumOfRows` computes
its "independent" expected sum with `row.Usage.BilledTotalTokens()` — the same function
the implementation uses. If `BilledTotalTokens()` were ever wrong, that test would pass
while enshrining the error. That is the literal shape of the historical bug. It is mitigated
because `BilledTotalTokens()` has its own tests in `pkg/codex`, and the aggregation-only
job of this test is legitimately separable — but it should not be the only invariant test.

### Must be dropped or fixed before merge

**FIX 1-1 (BLOCKER before the renderer lands).** `spendTotals` carries `ProviderUSD` and
`ProviderUSDRows` (`cmd/spend_ledger.go:157-158, 196-198`). This is *not* a price table —
D-02 is not violated, the value is relayed from the provider's own `total_cost_usd` and
never computed from a rate — but nothing prevents a later renderer from putting it on the
headline line, which criterion 1 explicitly forbids. Either drop the two fields (nothing
reads them; they can come back when someone actually wants them), or land them together
with a named test — `TestCostLineHasNoDollarFigure` — that reads the rendered build and
continue closeout line and fails if a `$` or a `usd` field reaches it. Prefer dropping.

**FIX 1-2 (WARNING).** Add one invariant test that recomputes the expected total from the
four raw columns *by hand* rather than by calling `BilledTotalTokens()`, so the aggregation
test cannot inherit a future arithmetic bug from the function it is checking.

**FIX 1-3 (WARNING, new since 195).** `spendRow` has no job attribution. Phase 195 made a
single worker able to own a grouped chain of tasks (`JobName` / `JobReason` / `JobSource` /
`CoveredTaskIDs` on `codexBuildDispatch`). Add `JobName` to `spendRow` at merge time — it
is a one-line struct field and an `omitempty` JSON tag, and adding it later means a schema
version bump and a migration.

**FIX 1-4 (WARNING).** `spendRow.Status` is a free-form string while
`codex.TaskReceiptStatusCompleted` already exists as a constant. Reuse the constant or
document that the ledger's status is the dispatch status, not the receipt status — this
repo's documented failure mode is exactly two vocabularies for one idea.

### Revive cost in plain terms

**Very low — under an hour, and most of that is writing the two extra tests.** The code
already compiles and passes as-is. It needs one field removed, one field added, and two
tests. There is no rewrite case here: reproducing this file's two-file keying decision and
its refusal-to-average discipline from scratch would take most of a day and would probably
get the build/continue key wrong the first time, which is the mistake its own comments say
was made and corrected.

### Merge order

**Merge first, before either other branch.** It is the storage layer everything else writes
into, and it is the only one of the three with no dependency on anything not yet built.

---

## Branch 2 — `worktree-agent-aa57076cd698da76f` (OpenCode real usage)

### Verdict: MERGE-WITH-FIXES

### What it delivers

Toward **criterion 2** (the chat path reports real token usage, so the cost line has real
numbers instead of blanks) it delivers the OpenCode half, whole:

- `openCodeSessionUsageForRun` — finds this repository's OpenCode project by matching the
  project record's `worktree` field against the repo root, finds child sessions inside the
  run's time window, matches them to worker names, and reads their token totals from
  OpenCode's own on-disk message records.
- Every figure it produces is **read from disk by the Go runtime**, never relayed by an
  orchestrating model and never derived from text length. That is criterion 3's second half
  satisfied for this platform.
- It is correctly tagged `UsageSourceSessionTranscript` and never `UsageSourceProvider`,
  with a test that fails if that ever changes.
- Ambiguity is refused rather than guessed: two in-window sessions matching one worker name
  resolve to **nothing** plus a plain-English diagnostic, falling through to an honest
  estimate. That is the right call and it is tested.
- Path containment is done properly — `filepath.Rel` with a leading-`..` rejection after
  symlink evaluation, not a lexical prefix match, so `…/storage-evil` cannot pass. Tested.
- Every file read is size-bounded (2 MiB) and every directory scan is count-bounded (2,000).

### Compiles today? Yes, unchanged.

`go vet ./cmd/` clean; all six new tests pass, and they still pass with Branch 1's files
present in the same package (no symbol collisions — `readBoundedFile`, `evalSymlinksOrSelf`,
`timeInWindow` are all new names in `package cmd`).

### Are the fixtures and tests real? Better than expected — I verified them against the machine.

The fixture README honestly labels itself a hand-written approximation, which is normally a
red flag here. I checked the three load-bearing assumptions against the owner's actual
OpenCode store (4,385 real session records, read-only):

| Fixture assumption | Real store says |
|---|---|
| Layout is `project/*.json`, `session/<projectID>/*.json`, `message/<sessionID>/*.json` | **Confirmed** — exact match. |
| Project records carry a `worktree` string | **Confirmed** — keys are `id, sandboxes, time, vcs, worktree`. |
| Child sessions carry `parentID` | **Confirmed** — 3,766 of 4,385 sessions have one. |
| Child session titles carry the deterministic worker name | **Confirmed** — 1,199 real titles match the exact fixture shape, e.g. `🔨 Builder Anvil-20: Extend inject_aws_ui() (@explore subagent)`. |
| `tokens.total` equals `input + output + cache.read + cache.write` | **Confirmed on a real 23-message session**: sum of `total` = 1,496,525, sum of the four columns = 1,496,525, exactly equal. |

That last row is the important one. This is the same arithmetic that produced the 186×
undercount before, and here the parser's summation is right and demonstrably right against
production data. `reasoning` tokens are deliberately excluded with a written reason, and
excluding them is correct — including them would have broken the equality above.

### The two things it gets wrong

**FIX 2-1 — BLOCKER. Unsynchronised global map: this will crash the process.**
`cmd/wrapper_usage_opencode.go:329` declares
`var openCodeWorkerNamePatternCache = map[string]*regexp.Regexp{}` and
`openCodeTitleMatchesWorker` both reads and writes it with no mutex. Go's runtime aborts
the whole process on concurrent map writes — it is not a silent race, it is a hard crash.
Nothing calls this today, so it is invisible; the moment Phase 196 wires usage discovery
per-worker (the obvious wiring, since worker dispatch in this repo is parallel) it becomes
a crash in the cost-reporting path, i.e. a crash at the end of every build. Fix: delete the
cache entirely (`regexp.MustCompile` on a handful of short names per run costs nothing), or
use a `sync.Map`. Deleting is better — the cache is also unbounded.

**FIX 2-2 — BLOCKER for trust. The multi-message case is completely untested.**
`readOpenCodeUsage` *accumulates* `TotalTokens` across every assistant message in a session.
All three fixture sessions contain exactly **one** message. So the test suite proves the
field mapping and proves nothing at all about the accumulation, which is the only case that
ever occurs in reality (the real session I checked had 23 assistant messages). If OpenCode
ever reports a cumulative running total per message instead of a per-message figure, this
silently multiplies the reported cost by roughly the message count. I confirmed against
real data that today it is per-message and summing is correct — but that fact is currently
recorded nowhere the build can check. Fix: add a second message file to `ses_child_a` and
assert the summed result, so the invariant is locked by a test rather than by this document.

**FIX 2-3 — WARNING.** `openCodeSessionUsageForRunOrNone` is documented as "the convenience
entry point plan 174-06 calls". Plan 174-06 does not exist and never did. Both exported
entry points are orphans — code with no caller, which is this repository's named signature
failure. Merging them un-wired recreates that exact condition. **Do not merge this branch
into `main` on its own.** Merge it as part of the Phase 196 plan that wires it, in the same
change, or hold it on the branch until that plan starts.

**FIX 2-4 — WARNING.** `readBoundedFile` stats then reads (a time-of-check/time-of-use gap).
Low severity — the file is the user's own local store — but `os.Open` + `io.LimitReader`
is the same amount of code and has no gap.

**FIX 2-5 — WARNING.** `validateOpenCodeStoragePath` duplicates the containment logic
already in `validateSpendTranscriptPath` (`cmd/spend_session_capture.go`), and the branch
adds an `evalSymlinksOrSelf` helper without refactoring the original to use it. Two copies
of a security boundary is one copy too many; extract the shared helper at merge time.

**Price table:** none. Nothing in this branch mentions dollars, prices, or rates at all.

### Revive cost in plain terms

**Low — roughly half a day**, and that is dominated by wiring it into the build, not by
fixing it. The fixes themselves are: delete a five-line cache, add one fixture file and one
assertion, and extract one shared helper. Rewriting this file from scratch would cost
several days and would have to redo the reverse-engineering of OpenCode's undocumented
storage layout, which the branch has already done correctly — I checked its conclusions
against the real store and they hold.

### Merge order

**Merge second, after Branch 1**, and only inside the plan that also wires it into a real
dispatch path and writes rows into the ledger. Its output type (`openCodeSessionUsage`)
feeds directly into Branch 1's `spendRow` with no adapter.

---

## Branch 3 — `worktree-agent-a59fd3ee68644ea21` (Claude Code real usage — tests only)

### Verdict: DELETE — but harvest the fixture and lift the test file into the Phase 196 plan first

### What it delivers

Nothing executable. It delivers **only** `cmd/wrapper_usage_claude_test.go` and a fixture.
Its `test(174-04)` commit is a red-first TDD step whose green step was never written and
never salvaged.

### Compiles today? No. It breaks the build of the entire `cmd` package.

```
vet: cmd/wrapper_usage_claude_test.go:43:18: undefined: parseClaudeTranscriptUsage
```

It references four symbols that exist nowhere: `parseClaudeTranscriptUsage`,
`parseClaudeTranscriptUsageWithBounds`, `claudeTranscriptUsage`, and
`claudeTranscriptMaxLineBytes`. I searched every ref in the repository —
`git log --all --diff-filter=A -- '*wrapper_usage_claude.go'` returns nothing. The
implementation has never been committed anywhere. This branch cannot be merged in any form
without writing the ~250-line parser first; merging it as-is turns a green repository red.

### Is the fixture real? No, and it is now partly out of date.

Its own README says "hand-written, trimmed, fully redacted approximation". I checked the
two shapes it encodes against 258 real Claude Code transcripts on this machine, and against
the newest one in detail:

- The XML shape — `<usage><subagent_tokens>N</subagent_tokens><tool_uses>N</tool_uses>…</usage>`
  — **is current**: it appears in 40 real transcripts, including today's.
- The colon shape the fixture asserts for its main worker Mason-67 —
  `<usage>subagent_tokens: 110790\ntool_uses: 19…</usage>` — appears in 25 older
  transcripts and **not once in the current session's transcript**. It is a legacy shape.
- The fixture knows about two line types (`user` tool_result, and a `queue-operation`
  task-notification). Real transcripts carry the usage block on **three**: `user`,
  `queue-operation`, **and `attachment`**. The third is unrepresented.

### The defect the fixture actively conceals

This is the most important finding on all three branches. In real Claude Code transcripts,
**the same worker's usage block appears two or three times**, once per line type, for a
single dispatch. On today's transcript:

```
distinct token values:                15
occurrences per value:                [2,2,2,2,2,2,2,2,2,2,2,2,2,2,3]
total if every occurrence is summed:  8,930,280
total if deduplicated by dispatch:    4,237,379
```

A parser written to satisfy this test suite and then run against a real transcript reports
**2.1× the true cost**. The committed fixture contains each usage block exactly once, so
no test in this branch can detect it — the same structural failure as the 186× undercount,
in the same subsystem, one phase later. Any Phase 196 parser must key by `tool_use_id`
(last-wins) and must be locked by a test whose fixture repeats one dispatch's usage across
all three line types.

### What is worth keeping

The test file's *design* is good and should be lifted verbatim into the Phase 196 plan as
the specification for the parser it was written against:

- decoy handling — a `<usage>` block sitting inside a `tool_use` **description** must not
  be matched (structural parsing, not substring search);
- path refusal — a transcript path outside `$HOME/.claude/projects` must error, exercised
  through a `t.TempDir()` `HOME` so the real directory is never read;
- an oversized line must be skipped without abandoning the rest of the file;
- an absent file returns no entries and no error;
- a byte bound truncates without reading the whole file;
- session-transcript rows must never be tagged provider-grade.

Those six requirements are worth more than the code that would satisfy them. Copy them into
the plan, add the tool_use_id deduplication requirement above, add the `attachment` line
type, and drop the stale colon-form assertion to a tolerated-legacy case rather than the
headline one.

### Revive cost in plain terms

**Reviving the branch is strictly more expensive than deleting it and rewriting**, because
there is nothing to revive — the parser does not exist. Writing it is **roughly a day**,
and it is Phase 196's own work regardless of what happens to this branch. Merging the branch
first would buy a broken build in exchange for a test file that can be copied in a second.

### Written reason for deletion

*Deleted because it is a test with no implementation. The code it tests has never existed
in this repository, so merging it would break the build rather than fail a check — and its
fixture no longer matches the shape real transcripts use today, and hides a duplication
that would make the reported cost roughly twice the truth. Its six requirements have been
copied into the Phase 196 plan, which is the part that had value.*

---

## Cross-cutting findings for Phase 196

**F-1 (BLOCKER for planning). A second, cache-blind token vocabulary already exists in the
tree.** `pkg/llm.Usage` (`pkg/llm/client.go:87-90`) has only `InputTokens` and
`OutputTokens` — no cache columns at all — and `pkg/agent/pool.go:183-189` publishes and
reports exactly those two figures. That is the same shape as the historical 186× undercount,
living on a different lane from `codex.WorkerUsage`. Phase 196 must either make
`codex.WorkerUsage` the single authoritative type or explicitly document why two exist;
shipping a cost line while a second divergent accounting path is live is precisely this
repository's documented failure mode.

**F-2. Real usage is only produced on the lane the owner does not use.** Today
`codex.ParseUsage` is called from exactly one place —
`pkg/codex/platform_dispatch.go:236` — which is the CLI-spawned worker path.
Nothing on the Claude Code / OpenCode chat path produces a `WorkerUsage` at all. That is
criterion 2's entire gap, and it is why Branch 2 matters and why Branch 3's missing parser
matters more.

**F-3. Nothing is ever written down.** No caller anywhere constructs a `spendRow`, and no
`aether spend` cobra command exists (`grep spendCmd cmd/` → nothing). Criterion 3 is
currently at zero percent regardless of which branches merge.

**F-4. No price table exists anywhere, on any branch.** D-02 is intact. The only dollar
figures in the whole salvage are `spendTotals.ProviderUSD` (a relay of the provider's own
reported cost, Branch 1) and `WorkerUsage.USDCost` (already on `main`). Neither computes a
price from a rate. Recommend deleting the `ProviderUSD` relay anyway — see FIX 1-1 — since
no criterion asks for it and its existence is a standing temptation.

---

## What Phase 196 still has to build from scratch

Merging both recommended branches leaves criterion 1 and criterion 4 untouched and criterion
3 half-done. The following does not exist anywhere and must be written:

1. **The Claude Code transcript parser** (`cmd/wrapper_usage_claude.go`, ~250 lines).
   Must key by `tool_use_id` and deduplicate across the `user`, `queue-operation` and
   `attachment` line types, or it overcounts by ~2.1×. Branch 3's test file is the
   specification; its fixture must be regenerated from a real transcript with the
   duplication preserved. **~1 day.**

2. **The writer** — the code path that turns a finished dispatch into `spendRow`s and calls
   `saveSpendLedger`, on both the build lane and the continue lane, joining worker name →
   usage → `JobName`. Nothing today writes a single row. **~0.5 day.**

3. **The one honest closeout line** (criterion 1) — the renderer in `cmd/codex_visuals.go`
   that prints per-worker and total tokens with the measured/estimated split visible, plus
   `TestBuildEndsWithOneCostLine` and a companion test that fails if a dollar figure or a
   second cost line ever appears. **~0.5 day.**

4. **`aether spend`** (criterion 3) — the cobra command itself, plus
   `TestSpendDoesNotMutate` asserting that running it changes no file on disk (the
   `--dry-run`-must-not-mutate corollary in CLAUDE.md's Definition of Done, which this repo
   has already violated twice). **~0.5 day.**

5. **Model attribution on the team card** (criterion 4) — entirely untouched by all three
   branches. `codexBuildDispatch.Model` already exists and is already rendered, but nothing
   records *why* a model was chosen, nothing stops the documentation writer, knowledge-keeper
   and accessibility checker from inheriting, and `TestRoutineBuilderIsSonnetNeverInherit`
   and `TestOpusRequiresRecordedReason` do not exist. This is the largest untouched piece
   and shares nothing with the salvaged work. **~1 day.**

6. **Reconciling `pkg/llm.Usage` with `codex.WorkerUsage`** (finding F-1) — decide and
   test which is authoritative before a cost line ships on top of two of them. **~0.5 day.**

Rough total for the phase after the two merges: **four days of new work**, of which the
salvage saves roughly one and a half (the ledger and the OpenCode reader).

---

_Assessment only. No branch was merged, checked out, or modified. Working tree untouched._
