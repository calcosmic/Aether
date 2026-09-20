---
phase: 196-see-what-it-cost
review_iteration: 2
reviewed: 2026-08-28T00:00:00Z
depth: standard
scope: re-check of CR-01, CR-02, CR-03, WR-01, WR-02, WR-03, WR-07 only
fix_range: 5e82678a..HEAD
findings:
  critical: 2
  warning: 2
  info: 3
  total: 7
status: issues_found
---

# Phase 196: Code Review Report — Iteration 2

**Scope:** the seven findings the coordinator says were fixed. WR-04, WR-05, WR-06
and iteration 1's Info items were deliberately deferred and are not re-raised.
Nothing already passed in iteration 1 was re-reviewed.

## Verdict Table

| Finding | Status | One-line evidence |
|---|---|---|
| CR-01 Claude attribution | **CLOSED** | Join independently verified 252/252 on the real corpus; misattribution guard mutation-tested |
| CR-02 retry erases spend | **PARTIALLY CLOSED** | Closed only when the retry's worker names differ; the realistic same-name re-run is refused outright and files nothing |
| CR-03 zero as a measurement | **CLOSED** | Tag set only after a message is read; three named subtests fail on the mutation |
| WR-01 dropped diagnostics | **CLOSED** | Reasons plumbed through; test fails by name on the mutation |
| WR-02 one figure, two workers | **CLOSED (but see NEW-01)** | Both layers hold; the ledger-level layer is what breaks CR-02 |
| WR-03 two totals, one question | **CLOSED** | `tokens.total` no longer decoded at all; test fails by name on the mutation |
| WR-07 session tokens dropped | **CLOSED** | Fields removed, heading rescoped, note on every block; AST guard added |

**New findings:** 2 Critical, 2 Warning, 3 Info.

**Call: NOT SHIPPABLE.** Two of the three original Criticals are now interacting
to produce a new wrong number in the same documented flow CR-02 exists for, and
the Claude reader's missing run window makes a retry report the previous
attempt's figure. Both are in the retry path, both were reproduced by running the
code, and both are marked `measured` on the owner's screen.

---

## 1. Root-cause assessment

### CR-01 — CLOSED at the root

**The 252/252 measurement is real. I reproduced it independently**, with my own
script over `~/.claude/projects`, not the fixer's:

```
files: 1787
subagent_usage_records: 252
with_tool_use_id:       252
joined:                 252
agenttype_eq_subagenttype: 252
has_description:        252
```

(The fixer counted 250 across 1,726 files; my corpus is two days newer and a
superset. Every ratio is 100%.) So: every subagent completion record carries a
`tool_result` naming a `tool_use_id`; every one of those ids joins back to a
`tool_use` block **in the same file**; `agentType` equalled `input.subagent_type`
in every case; and every dispatching block carried a non-empty `description`.
The join the fix relies on is the platform's own and is universal in observed
data.

**The caveat is fairly stated, not a cover.** Every description in my corpus is
GSD-shaped (`'Execute plan 06 of phase 172'`, `'Code review phase 172'`) — there
is genuinely no Aether build in these transcripts, so the *format* claim rests on
`.claude/commands/ant/build.md:344` and `.opencode/commands/ant/build.md:344`
("Use the exact visible description: `{caste emoji} {Caste} {name}: {task}`"),
which sits directly beneath the `subagent_type` instruction the same fix relies
on and which the corpus confirms is honoured for `subagent_type`. That is a fair
reading. It does leave a new, untested dependency — see NEW-04.

**The resolution order cannot credit one worker with another's tokens.** I
mutation-tested it: disabling the description branch in `claudeRowWorker` makes
`TestClaudeTranscriptJoinsOnWhatThePlatformRecords` fail with

> `Roam-90 was credited 777777 tokens from a row the transcript says belongs to Stray-99`

so the description-first / no-fall-through-to-definition ordering is genuinely
load-bearing and genuinely tested. The definition rule refuses a definition more
than one worker ran as, and the direct cases (`row.WorkerName`, `row.AgentID`)
cannot collide with a deterministic worker name.

The failure direction when the description does not name a worker is
"not reported", never "wrong worker" — the safe direction. Confirmed by running
it (NEW-03).

### CR-02 — PARTIALLY CLOSED

The merge-by-run mechanism is correct and mutation-tested: forcing whole
replacement makes `TestRetryDoesNotEraseTheFirstAttemptsSpend` fail by name
("got 1 rows after two attempts of one phase, want 4"). A re-finalize of one
attempt rewrites only its own rows; a second attempt keeps earlier ones.

But it only works when the retry's workers have **different names**, and the
runtime cannot produce different names for a re-run of the same phase. See
NEW-01. The test that proves the fix uses `Anvil-21` for the retry against
`Anvil-20`/`Mason-67` in the first attempt — see NEW-06.

**Degradation path:** `TestEveryProductionRunNamesItsAttempt` is a real AST guard
with an anti-vacuity floor, and it proves every production `spendWriteRequest`
literal *names* `RunID`. It does not prove the value is non-empty.
`spendRunIDFromAttempt` returns `""` for an empty `attemptRel` and
`spendRunIDFromTimestamp` returns `""` for an empty timestamp. On the build lanes
`attemptRel` is guarded non-empty at its source; on `continue-finalize` a plan
record with no `generated_at` would reach the fallback. It is not silent — the
outcome carries the note — so this is Info, not a defect (IN-03).

### CR-03, WR-01, WR-03 — CLOSED

Each fixed at the producer, each mutation-tested, each failing by name:

- CR-03: `readOpenCodeUsage` now returns the zero value on an unreadable
  directory and only tags the source after `read > 0`. Re-tagging on construction
  fails three named subtests including *"Mason-67 is marked as measured with 0
  tokens although nothing about it could be read"*.
- WR-01: reasons returned through `openCodeSessionUsageForRunOrNone` and appended
  to `res.Diagnostics`. Dropping them again fails
  `TestOpenCodeRefusalsReachTheOwner` by name.
- WR-03: `tokens.total` is no longer decoded **at all** — the struct field is
  gone, which is the right shape, because it means a one-line change cannot put
  the second answer back. Re-adding it fails
  `TestOpenCodeReaderPrefersTheDisjointColumns` with the exact harm named.

### WR-02 — CLOSED for its own harm

Two layers: the writer credits one measurement once and notes it, and
`saveSpendLedger` refuses two rows under one name that both carry a figure.
Neither drops a worker — both dispatches keep a row, only one carries the figure
— so the "vanishing worker" direction is not reintroduced. Removing the writer
guard fails `TestOneMeasurementIsNeverCreditedToTwoWorkers`.

The choice is sound in isolation. The ledger-level layer is what collides with
CR-02 (NEW-01).

### WR-07 — CLOSED

`SessionUsage`/`SessionReported` are gone from `wrapperUsageResolution`; the
Claude branch skips the main-session row without collecting it; a new AST guard
(`TestNoResolvedFieldIsPopulatedThenDropped`) fails if any resolution field is
written and never read.

The wording is honest in both directions. The heading is now `What The Helpers
Cost` — it no longer claims the phase's cost while omitting the largest
component — and every rendered block with rows carries *"(The coordinator's own
back-and-forth, the main session itself, is not counted here. These figures are
the helpers it sent.)"*. Removing that line fails `TestCostBlockSaysWhatItCounts`
by name. The em-dash follow-up (7fedc32c) is correct: the em dash is the
"no figure" sentinel and had no business in prose under a column of them.

The detail view's own header (`cmd/spend_cmd.go`, *"Token use for phase N, worker
by worker, as each worker's own tool reported it"*) is already scoped to workers,
so it does not overclaim; it is silent about the session rather than wrong about
it. No stale `What This Phase Has Cost` string remains in any runtime path.

---

## 2. Can the new tests fail?

Seven mutations, each applied to a working copy backed up outside the repo and
restored byte-identically afterwards (verified with `diff`; no `git stash`, no
checkout). Every one produced a **named** failure:

| Mutation | Test that failed | Failed by name? |
|---|---|---|
| Disable the description join | `TestClaudeTranscriptJoinsOnWhatThePlatformRecords`, `TestResolverReadsTranscriptOnTheClaudePath` | yes |
| Replace instead of merge by run | `TestRetryDoesNotEraseTheFirstAttemptsSpend` | yes |
| Tag OpenCode usage on construction | `TestOpenCodeSessionWithNoReadableMessagesIsNotReported` (3 subtests) | yes |
| Drop the OpenCode reasons | `TestOpenCodeRefusalsReachTheOwner` | yes |
| Remove the credited-once guard | `TestOneMeasurementIsNeverCreditedToTwoWorkers` | yes |
| Re-add `tokens.total` | `TestOpenCodeReaderPrefersTheDisjointColumns` | yes |
| Remove the session note | `TestCostBlockSaysWhatItCounts` | yes |

Two tests are weaker than they look — NEW-05 and NEW-06 below. Neither is a
false pass of the fix as a whole (another test covers each mechanism), but both
repeat the fixture-shape mistake this phase exists to correct.

The transcript fixture itself is now genuinely right: `resolverSubagentDispatch`
writes a real two-line dispatch/result pair with `tool_use` → `tool_result` →
`tool_use_id`, `input.subagent_type`, `input.description` and
`toolUseResult.agentType`, matching every field my corpus measurement found.

---

## Critical Issues

### NEW-01: A retry of a phase now files nothing at all — CR-02's fix and WR-02's fix are in direct conflict

**File:** `cmd/spend_ledger.go:194-207` (the refusal), `cmd/spend_writer.go:228-246` (the merge)

**Issue:** CR-02's fix merges an earlier attempt's rows with the current
attempt's. WR-02's fix makes `saveSpendLedger` refuse **any** two rows under one
worker name that both carry a token figure. But `deterministicAntName` is a pure
function of phase and caste — `phase:%d:%s` for specialists,
`phase:%d:continue:%s` for reviewers and the check watcher, and
`coherentJobDispatchName` is a pure function of the phase and the job's task set.
So a re-run of the same phase produces **the same worker names**, and the merge
produces exactly the pair the refusal forbids.

Reproduced. `aether build 9 --force` after a blocked check (the command
`continueNextCommandForBlocked` hands the owner):

```
attempt B: err=save spend ledger: two rows filed under worker Mason-67 both carry
           a token figure — one measurement would be counted twice in this phase's total
           outcome={RowsWritten:0 Reported:2 Notes:[]}

── What The Helpers Cost ──
Cost: 600K tokens across 2 workers. The total counts all 2, because every tool reported a figure.
  🔨🐜 Builder Mason-67  500K  measured
  👁️🐜 Watcher Vigil-12  100K  measured
```

The phase really cost 1,090,000 tokens across four worker runs. The owner is
shown 600K across two, with the sentence *"The total counts all 2, because every
tool reported a figure"* — which is now false, since two more workers ran and
both reported. The second attempt's entire accounting is discarded; only a
`warning:` line on stderr mentions it.

The same happens on the check lane, where it is even easier to reach: a second
`aether continue` on a blocked phase re-dispatches the watcher under
`deterministicAntName("watcher", "phase:N:continue:watcher")` — identical on
every run:

```
second check: err=save spend ledger: two rows filed under worker Vigil-12 both carry a token figure
ledger rows=1 total=120000
```

This is CR-02's original harm — a phase's real cost understated, the retry's
spend invisible — reintroduced through a different door, and it is worse in one
respect: before the fix pair the retry at least replaced the old figure, so the
number was current. Now it is stale *and* incomplete.

**Fix:** The refusal is asking the wrong question. Two measured rows under one
name are only a double-count when they belong to the **same run**. Scope it:

```go
// cmd/spend_ledger.go
measuredByWorker := map[string]bool{}   // key: runID + "\x00" + worker
...
if !rows[i].Usage.Empty() {
    key := strings.TrimSpace(rows[i].RunID) + "\x00" + worker
    if measuredByWorker[key] {
        return fmt.Errorf(
            "save spend ledger: two rows filed under worker %s in the same run both carry a token figure — "+
                "one measurement would be counted twice in this phase's total", worker)
    }
    measuredByWorker[key] = true
}
```

Then rewrite `TestRetryDoesNotEraseTheFirstAttemptsSpend` to use the **same**
worker names on both attempts (see NEW-06), which is what the runtime produces,
and add a case asserting a same-run duplicate is still refused.

Note the display consequence to settle at the same time: the breakdown will then
show two rows named `Mason-67`. They need to be distinguishable — the attempt is
already on the row (`RunID`), so a per-attempt sub-heading, or "(attempt 2)" on
the identity cell, would do it.

---

### NEW-02: The Claude reader has no run window, so a retry is credited with the previous attempt's tokens

**File:** `cmd/wrapper_usage_claude.go:161-236` (whole-file read), `cmd/wrapper_usage_resolve.go:186-193` (first-wins duplicate guard)

**Issue:** `parseClaudeTranscriptUsageWithBounds` reads the whole transcript file.
`wrapperUsageRequest` carries `StartedAt`/`EndedAt` and the OpenCode path uses
them (`openCodeSessionInWindow`); the Claude path ignores them entirely. One chat
session routinely runs a build, a check, and a re-run of the same phase — all
into one transcript file.

Because worker names are deterministic per phase and caste, both attempts'
subagent records carry the **same** name in their descriptions. The resolver's
new duplicate guard then keeps the **first** row in file order — which is the
**oldest** one — and reports the second as unattributed.

Reproduced. Attempt 1's Mason-67 spent 1,000,000; the retry's Mason-67 spent
40,000. Running the retry's finalize:

```
outcome={RowsWritten:1 Reported:1 Notes:[two subagent records in the session transcript
         both resolve to worker Mason-67, so the second was left unattributed
         rather than replacing the first]}
row name=Mason-67 run=attempt-B tokens=1000000   (the RETRY really spent 40000)
```

The retry's row is filed under the retry's own attempt id carrying the **first
attempt's** measurement — a 25x overstatement, marked `measured`, with a
diagnostic that describes the wrong choice as a safety property. Nothing about
this row tells the owner it is stale.

This was unreachable in iteration 1 only because nothing was ever attributed at
all; CR-01's fix made it live and CR-02's fix made a second attempt a normal
recorded event. It is a defect the fixes introduced.

**Fix:** Give the Claude reader the run window the OpenCode reader already has.
The transcript carries a `timestamp` on every line; carry it onto the row and
filter, mirroring `openCodeSessionInWindow`:

```go
// cmd/wrapper_usage_claude.go — capture the completion line's own timestamp
type claudeTranscriptLine struct {
    // ...
    Timestamp string `json:"timestamp"`
}
// carried onto claudeTranscriptUsage.RecordedAt, then in the resolver:
if !timeInWindow(row.RecordedAt, req.StartedAt, req.EndedAt) {
    continue
}
```

Failing that, the duplicate guard must prefer the **last** matching row rather
than the first, and say so — but a window is the real answer, because it also
stops one phase's transcript rows reaching another phase's ledger.

---

## Warnings

### NEW-03: The unattributed diagnostic asserts something the code does not know

**File:** `cmd/wrapper_usage_resolve.go:279-282`

**Issue:** When a dispatch description names none of the run's workers, the note
reads *"the session transcript holds usage for a worker this run did not dispatch
(%q)"*. The code cannot know that. The far likelier cause is that the wrapper
paraphrased the description instead of using the required
`{caste emoji} {Caste} {name}: {task}` form — the description is composed by an
LLM following a markdown instruction, not by the runtime.

Reproduced with a paraphrased description (`"Implement the parser"`) for a worker
the run *did* dispatch:

```
reported=0 notes=[the session transcript holds usage for a worker this run did not
                  dispatch ("Implement the parser"), so it was left unattributed
                  rather than guessed onto one]
```

The owner is told a stranger's spend was seen. The truth is that his own
worker's spend was dropped, and the message points away from the real cause.

**Fix:** Say what actually happened, and name the worker that went unmatched:

```go
return "", fmt.Sprintf(
    "a helper's token record could not be matched to any worker on this run — "+
        "the dispatch it came from was labelled %q, which names none of them, "+
        "so its use was left out rather than guessed onto somebody",
    shortenForDiagnostic(description))
```

---

### NEW-04: All Claude attribution now depends on a wrapper-markdown line that nothing tests

**File:** `.claude/commands/ant/build.md:344`, `.opencode/commands/ant/build.md:344`, `.claude/commands/ant/continue.md:185`

**Issue:** After the fix, whether a worker's tokens are attributed on the primary
platform turns entirely on the description string containing the deterministic
name verbatim. That contract lives in one line of wrapper markdown. Nothing reads
it, and no test asserts it still carries `{name}`.

Claude Code documents the Task tool's `description` as a short label, so the
pressure to shorten that line is real — and shortening it to
`{Caste}: {task}` would silently return the subsystem to reporting
`Cost: not known` on every build, with the full suite green. That is the
Definition of Done's *"a documentation claim about runtime behaviour must be
testable or removed"*, applied to a claim the runtime now depends on.

**Fix:** Assert the contract from Go, in the same shape as the existing
wrapper-parity guards:

```go
func TestDispatchDescriptionCarriesTheAccountingKey(t *testing.T) {
    for _, path := range []string{
        ".claude/commands/ant/build.md",
        ".opencode/commands/ant/build.md",
        ".claude/commands/ant/continue.md",
        ".opencode/commands/ant/continue.md",
    } {
        body := readRepoFile(t, path)
        if !strings.Contains(body, "{caste emoji} {Caste} {name}: {task}") {
            t.Errorf("%s no longer requires the worker's name in the dispatch description — "+
                "that string is what the token ledger joins a Claude Code transcript row to a "+
                "worker on, so dropping it reports every build as costing nothing", path)
        }
    }
}
```

and add a sentence to that line in each wrapper saying the name is what the cost
record joins on, so a future editor sees the consequence.

---

## Info

### IN-01: `TestClaudeBuildAttributesEveryWorkersTokens` passes without the description join

Disabling the description branch leaves this test green — it resolves through the
definition rule, because its two workers have distinct definitions. It is
therefore not the test that proves CR-01; `TestClaudeTranscriptJoinsOnWhatThePlatformRecords`
is. Give it a second builder so the definition rule cannot answer, and it will
test what its name says.

### IN-02: The retry test uses worker names the runtime cannot produce for a retry

`TestRetryDoesNotEraseTheFirstAttemptsSpend` (`cmd/spend_writer_test.go`) files
`Anvil-20`/`Mason-67`/`Vigil-12` then retries with `Anvil-21`. A re-run of one
phase produces the *same* names, because `deterministicAntName` and
`coherentJobDispatchName` are pure functions of the phase and caste. The fixture
is shaped so the collision NEW-01 describes cannot occur — the same
"fixture in a shape production never emits" failure that made CR-01 invisible.
Fixing NEW-01 requires rewriting this test anyway.

### IN-03: The empty-RunID fallback is reachable, but announces itself

`TestEveryProductionRunNamesItsAttempt` proves every production call site *names*
`RunID`, not that the value is non-empty. `spendRunIDFromTimestamp("continue",
plan.GeneratedAt)` yields `""` for a plan record with no `generated_at`, which
falls back to whole-phase replacement. It emits its note rather than passing
silently, and `attemptRel` is guarded non-empty on both build lanes, so this is
acceptable as it stands — worth knowing rather than worth changing.

---

_Reviewed: 2026-08-28_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard, narrow re-check_
