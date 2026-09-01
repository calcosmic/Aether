---
phase: 196-see-what-it-cost
review_iteration: 3
reviewed: 2026-08-28T00:00:00Z
depth: standard
scope: re-check of NEW-01..NEW-04 and the two iteration-2 Info items only
fix_range: 7fedc32c..HEAD
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: clean
---

# Phase 196: Code Review Report — Iteration 3

**Scope:** NEW-01..NEW-04 and the two Info items from iteration 2, plus the two
self-corrections the pass claims. WR-04/05/06 remain deferred and are not
re-raised. Nothing already passed was re-reviewed.

## Verdict Table

| Finding | Status | Evidence |
|---|---|---|
| NEW-01 refusal blocked the retry | **CLOSED** | Refusal scoped by `RunID`; same-run pair still refused, cross-run pair accepted, both checked directly against `saveSpendLedger` |
| NEW-02 retry credited with prior attempt | **CLOSED** | Run window applied; timestamp claim re-measured independently 252/252; retry gets its own 40K, not the prior 1.0M |
| NEW-03 note blamed the wrong cause | **CLOSED** | Message rewritten; mutation restoring the old text fails by name |
| NEW-04 untested wrapper contract | **CLOSED** | All six surfaces carry format + consequence; test asserts the instruction line, not the file |
| IN-01 attribution test passed via the wrong route | **CLOSED (self-catch verified)** | Fails "1 of 3 workers were credited" with the join disabled |
| IN-02 retry test used impossible names | **CLOSED** | Retry now measured with the runtime's own `deterministicAntName` output |
| IN-03 empty-RunID fallback | **CLOSED** | Reachable but announces itself twice; confirmed by running it |

**New findings:** 1 Info. No Critical, no Warning, no regression.

**Call: SHIPPABLE.** The pattern that bit rounds one and two — a fixture in a
shape the runtime cannot produce — is genuinely broken this time, and I verified
both self-catches myself rather than taking them on report.

---

## 1. Root-cause assessment

### NEW-01 — CLOSED

The refusal is now keyed on `RunID + worker` (`cmd/spend_ledger.go:205-227`).
Checked directly against `saveSpendLedger`, bypassing the writer:

- two measured rows, one name, **same** `RunID` → refused: *"two rows filed under
  worker Vigil-12 in the same run both carry a token figure"*. WR-02's protection
  is intact at the ledger layer.
- the same pair across **two** `RunID`s → accepted.

And the writer's own guard still neutralises a same-run duplicate before the
ledger sees it: two dispatches named `Vigil-12` in one run file two rows totalling
1,000, not 2,000, with no error.

**A retry now records the full cost on both lanes**, using the names the runtime
actually generates (`deterministicAntName("builder", "phase:9:builder")` →
`Mason-25`, identical on both attempts — the exact collision that made iteration
2's fixture wrong):

- **Direct lane:** attempts of 600K and 490K file four rows totalling
  **1,090,000** — the hand-summed truth.
- **Platform (Claude) lane:** 1,000,000 then 40,000 → **1,040,000**, each attempt
  carrying its own figure.

**The attempt labelling reads as two runs, not one double bill.** Rendered:

```
── What The Helpers Cost ──
Cost: 1.0M tokens across 4 workers. The total counts all 4, because every tool reported a figure.
  🔨🐜 Builder Mason-25 (attempt 1)  500K  measured
  👁️🐜 Watcher Keen-62 (attempt 1)   100K  measured
  🔨🐜 Builder Mason-25 (attempt 2)  400K  measured
  👁️🐜 Watcher Keen-62 (attempt 2)    90K  measured
```

Without the labels those four lines would read as two names billed twice; with
them the two runs are unmistakable. Labels appear only when a name occurs in more
than one attempt, so an ordinary single-attempt phase is unchanged (confirmed in
the goldens). All four `spendWorkerDescription` call sites — both renderers and
both width measurers — pass the same label, so padding cannot desync. The one
residual imprecision is the summary sentence's worker count (IN-01 below).

### NEW-02 — CLOSED

**The timestamp claim is true; I re-measured it myself** across
`~/.claude/projects` with my own script:

```
records: 252    has_timestamp: 252    rfc3339_parseable: 252    bad: []
```

Every subagent usage record on this machine carries an RFC3339 `timestamp`, and
Go's `time.Parse(time.RFC3339, …)` accepts the real fractional-second form
(`2026-08-11T18:53:59.256Z` → parses) — checked against three timestamps lifted
from the captured fixture.

The window is applied at the resolver (`cmd/wrapper_usage_resolve.go:193-197`)
using the same `timeInWindow` helper the OpenCode reader already used, so the two
platforms now bound records the same way. Walked end to end: one transcript
holding both attempts, attempt 2's finalize credits `Mason-25` with **40,000** —
its own spend — and reports *"1 helper's token record … written outside this
run's own start and finish times"*. Iteration 2's 25x overstatement is gone.

**Undated record:** excluded (`timeInWindow` returns false on the zero time) and
counted in the note. Fails safe — a record that cannot be placed is left out
rather than assumed to belong.

**"No start time → report nothing" is not silent.** Ran it: `StartedAt` zero
produces two notes, the first naming the cause directly — *"this run's start time
was not recorded, so no worker's usage could be matched to it by time"* — from
the writer's pre-existing check, plus the window note. Both reach the owner
through `outcome.Notes`. There is no path where the window silently swallows a
run.

### NEW-03 — CLOSED

The message now says the dispatch label did not match, not that a stranger ran.
Restoring the old wording fails `TestTheUnmatchedRecordNoteDoesNotBlameAWorkerThatRan`
with both of its assertions naming the harm.

### NEW-04 — CLOSED

All six wrapper surfaces carry the format on the instruction line **and** the
consequence sentence — verified by grep, one occurrence each:

```
.claude/commands/ant/build.md      instruction+format=1  note=1
.claude/commands/ant-build.md      instruction+format=1  note=1
.opencode/commands/ant/build.md    instruction+format=1  note=1
.claude/commands/ant/continue.md   instruction+format=1  note=1
.claude/commands/ant-continue.md   instruction+format=1  note=1
.opencode/commands/ant/continue.md instruction+format=1  note=1
```

The six is exhaustive for this contract: the same instruction also appears in
`colonize.md`, `plan.md`, `seal.md` and `swarm.md`, but no ledger is ever written
for those workflows (`spendLedgerWorkflows()` is build and continue only), so
their description format cannot affect attribution. Correctly scoped, not a gap.

The test also closes the loop the string check cannot: it builds a description
from `deterministicAntName`'s real output and puts it through
`workerNameAppearsIn`, the matcher the resolver actually uses.

---

## 2. Do the fixes interact badly?

No. I walked a retry end to end on both lanes with the runtime's own names.

**What the owner sees on a `build --force` after a blocked check:** the second
build runs, its workers carry the same names as the first attempt's, and the
ending screen shows every worker from both attempts, each labelled with its
attempt, under a total that is the arithmetic sum of all of them. Nothing is
refused, nothing is erased, no figure is repeated.

The three mechanisms touch the retry path in sequence and do not fight:

1. **The run window** runs first and is what makes the retry's transcript records
   separable at all — attempt 1's records fall before attempt 2's window and are
   excluded with a note. The windows cannot overlap in practice: `StartedAt` is
   the attempt's own manifest `generated_at`, and a new attempt always begins a
   new manifest after the previous one finished.
2. **The keep-first duplicate rule** then has nothing to do — within one window a
   worker has one record — so the rule that caused iteration 2's misattribution
   is now unreachable in the retry case that triggered it. It survives only as a
   backstop for a genuine within-run duplicate, where keeping the first and
   noting it is the right answer.
3. **The run-scoped ledger refusal** last, which accepts the cross-attempt pair
   the merge produces and still refuses a same-run pair.

The one place the three could have collided — a same-name pair reaching
`saveSpendLedger` from two attempts — is exactly what I tested directly, and it
is accepted.

---

## 3. Can each new test fail?

Six mutations, each on a copy backed up outside the repo and restored
byte-identically afterwards (verified with `diff`; no `git stash`, no checkout).
Every one failed **by name**:

| Mutation | Test that failed | Message |
|---|---|---|
| Unscope the refusal to name-only | `TestRetryDoesNotEraseTheFirstAttemptsSpend`, `TestOneMeasurementIsNeverCreditedToTwoWorkers/the_refusal_is_scoped_to_one_run…` | subtest name states the rule |
| Remove the run window | `TestARetryIsNotCreditedWithTheFirstAttemptsTokens` | — |
| Restore the blaming note | `TestTheUnmatchedRecordNoteDoesNotBlameAWorkerThatRan` | names the wrong-cause harm |
| Shorten build.md's instruction line | `TestDispatchDescriptionCarriesTheAccountingKey` | quotes the shortened line |
| Disable the description join | `TestClaudeBuildAttributesEveryWorkersTokens`, `TestClaudeTranscriptJoinsOnWhatThePlatformRecords` | "1 of 3 workers were credited" |

**Both claimed self-catches verified independently, not taken on report:**

- **The contract test's blind spot was real and is fixed.** I shortened step 4 of
  `.claude/commands/ant/build.md` to `{Caste}: {task}` and confirmed the file
  *still contains* the full format string elsewhere (grep count 1) — so a
  whole-file substring check would indeed have stayed green, exactly as claimed.
  The line-scoped assertion fails and quotes the offending line.
- **The attribution test's route was wrong and is fixed.** With the description
  join disabled it now fails with precisely the claimed message, *"1 of 3 workers
  were credited with a figure"*, and the diagnostics show why: two workers now
  share `aether-builder`, so the definition rule cannot answer for either.

Both self-corrections are genuine.

---

## Info

### IN-01: The retry summary counts worker *runs* as *workers*

**File:** `cmd/spend_cost_line.go:167-196` (`spendCostLineTotalSentence`), called with `len(rows)`

After a retry the sentence reads *"Cost: 1.0M tokens across 4 workers. The total
counts all 4, because every tool reported a figure."* — but two workers ran,
twice each. The token total is exactly right and every row is labelled with its
attempt directly below, so the misreading is recoverable in one glance; it is
also defensible that four figures make up the total. It is still the one place in
the block where the words are looser than the numbers.

**Fix (optional):** count distinct names when any attempt label is in play:

```go
// "Cost: 1.0M tokens across 2 workers over 2 attempts."
```

Two smaller things in the same area, both in text the owner reads:

- The window note disagrees with itself on number at a count of one: *"**1
  helper's token record** … **were** written outside … so **they were** left out"*.
  `spendWorkerRecordWord` was written to read naturally at one and at many; the
  verbs need the same treatment.
- That note carries an em dash, which commit 7fedc32c deliberately removed from
  the cost block's prose because the em dash is the "no figure" sentinel. The
  notes print to stderr rather than under the figure column, so this is probably
  fine — worth a glance rather than a change.

---

_Reviewed: 2026-08-28_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard, narrow re-check_
