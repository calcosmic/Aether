---
phase: 196-see-what-it-cost
plan: 03
subsystem: infra
tags: [token-usage, spend, claude-code, transcript, deduplication, fixture, path-containment, go, tdd]

requires:
  - phase: 174-spend
    provides: codex.WorkerUsage, UsageSourceSessionTranscript, BilledTotalTokens, validateSpendTranscriptPath
  - phase: 196-02
    provides: one authoritative token type, the no-length-derivation ratchet, an unreported worker carrying no figure
provides:
  - parseClaudeTranscriptUsage / parseClaudeTranscriptUsageWithBounds — the Claude Code session-transcript reader
  - Deduplication by message.id (last occurrence wins), corroborated by requestId, across both usage-bearing line types
  - A real redacted capture that KEEPS its duplicate usage blocks, plus the integrity test that fails if the repetition is ever tidied away
  - A corrected symlink resolution in validateSpendTranscriptPath for a path that does not exist yet
affects: [196-04, 196-05, 196-06, 196-07, 196-08]

actuals:
  tokens: 20250
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Structural JSON decoding into declared positions, never a substring scan — a usage object quoted in free text is text, not accounting"
    - "The line type is the TOP-LEVEL type field; a nested type on a content block is never consulted"
    - "A regression fixture is a REAL capture that retains the defect it guards against; a fixture-integrity test protects the other tests from a future tidy-up"
    - "Adversarial inputs (decoys, oversized lines, legacy shapes) are constructed in-test so the captured fixture stays pristine"
    - "bufio.Reader with a per-line cap instead of bufio.Scanner, so one oversized line is skipped rather than abandoning the file"

key-files:
  created:
    - cmd/wrapper_usage_claude.go
    - cmd/wrapper_usage_claude_test.go
    - cmd/testdata/spend/claude/transcript-with-duplicates.jsonl
    - cmd/testdata/spend/claude/README.md
  modified:
    - cmd/spend_session_capture.go

key-decisions:
  - "The two usage-bearing line types were re-measured on this machine's whole corpus (1,726 transcripts, 1.0 GB): assistant 142,581 with usage at .message.usage, and user 252 with usage at .toolUseResult.usage. Nothing else carries usage anywhere. This confirms D-04 as corrected and supersedes both the original mechanism and its figures."
  - "The user-type usage-bearing line is a dispatched subagent's completion record. It has NO message.id at all, so it is the identifier-less case: counted once on its own under its own line identity, never dropped and never folded into the session's row."
  - "The deduplication key is message.id, falling back to requestId, then the line's uuid, then its position. requestId is a genuine corroborator: with message.id disabled the fixture still deduplicates correctly through it."
  - "TotalTokens is deliberately left unset on every returned row so consumers reach the authoritative sum through codex.WorkerUsage.BilledTotalTokens(). A second total-computing implementation is Pitfall 1."
  - "The legacy colon-shaped <usage>subagent_tokens: N</usage> text block is tolerated as in 'does not break the read' and is NOT counted. Scraping a number out of a text blob is the substring matching this plan exists to forbid."
  - "Decoys are constructed in-test rather than added to the fixture, so the committed fixture remains an unedited real capture."

patterns-established:
  - "Non-vacuity is demonstrated by breaking the implementation on purpose and recording the real failure output, not asserted in prose"
  - "Test literals are hand-derived with independent tooling (jq/awk) and checked by addition; the test's own walk of the fixture uses different machinery (untyped maps) from the reader's structs"

requirements-completed: [COST-02]

coverage:
  - id: D1
    description: "A reply republished across several transcript lines is billed once, not two or three times"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "cmd/wrapper_usage_claude_test.go#TestClaudeTranscriptDeduplicatesUsageByMessageID"
        status: pass
    human_judgment: false
  - id: D2
    description: "Both usage-bearing line types are read, including the identifier-less subagent completion record"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "cmd/wrapper_usage_claude_test.go#TestClaudeTranscriptReadsBothUsageBearingLineTypes"
        status: pass
    human_judgment: false
  - id: D3
    description: "The regression fixture is a real capture that keeps its duplicates, and cannot quietly become a fixture that cannot fail"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "cmd/wrapper_usage_claude_test.go#TestClaudeTranscriptFixtureContainsRepeatedMessageIDs"
        status: pass
    human_judgment: false
  - id: D4
    description: "Matching is structural: a usage block quoted in a tool description, or on a line whose top-level type is outside the accepted set, is never counted"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "cmd/wrapper_usage_claude_test.go#TestClaudeTranscriptIgnoresUsageInsideToolDescription"
        status: pass
      - kind: unit
        ref: "cmd/wrapper_usage_claude_test.go#TestClaudeTranscriptReadsTopLevelLineTypeNotNestedType"
        status: pass
      - kind: unit
        ref: "cmd/wrapper_usage_claude_test.go#TestClaudeTranscriptLegacyColonFormIsTolerated"
        status: pass
    human_judgment: false
  - id: D5
    description: "The read is contained to the platform's projects directory and bounded in both file size and line size"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "cmd/wrapper_usage_claude_test.go#TestClaudeTranscriptRefusesPathOutsideProjectsRoot"
        status: pass
      - kind: unit
        ref: "cmd/wrapper_usage_claude_test.go#TestClaudeTranscriptSkipsOversizedLineAndContinues"
        status: pass
      - kind: unit
        ref: "cmd/wrapper_usage_claude_test.go#TestClaudeTranscriptByteBoundStopsEarly"
        status: pass
      - kind: unit
        ref: "cmd/wrapper_usage_claude_test.go#TestClaudeTranscriptAbsentFileIsEmptyAndNoError"
        status: pass
    human_judgment: false
  - id: D6
    description: "Every transcript row is tagged session-transcript and never provider-grade"
    requirement: "COST-02"
    verification:
      - kind: unit
        ref: "cmd/wrapper_usage_claude_test.go#TestClaudeTranscriptRowsAreNeverProviderGrade"
        status: pass
    human_judgment: false
  - id: D7
    description: "The reader is reached by a real dispatch path"
    verification: []
    human_judgment: true
    rationale: "Out of scope here by design — the plan states nothing calls this reader yet. Plan 196-05 wires it into the finalize path and plan 196-08 fails if it is still unreached. Until then this is an unwired parser by explicit instruction, and a human should confirm 196-05 actually closed it."

duration: 41min
completed: 2026-08-28
status: complete
---

# Phase 196 Plan 03: The Claude Transcript Reader Summary

**A Claude Code transcript reader that bills a reply once instead of once per line — deduplicated by `message.id` across both usage-bearing line types, locked by a real 28-line capture whose duplicate usage blocks are preserved on purpose, where a reader that stops deduplicating reports 2,764,260 tokens against a true 1,572,834.**

## Performance

- **Duration:** 41 min
- **Started:** 2026-08-28T11:07:00Z
- **Completed:** 2026-08-28T11:48:00Z
- **Tasks:** 2 (Task 2 as RED then GREEN)
- **Files modified:** 5 (4 created, 1 modified)

## The re-measurement (this plan's load-bearing evidence)

The plan required the capture to be **re-measured on this machine** rather than
trusting any figure recorded before, because the first version of decision D-04
was wrong about the mechanism and its two totals came from that wrong reading.

### Corpus-wide, by top-level line type

Measured across **the whole of `~/.claude/projects` — 1,726 transcripts, 1.0 GB**,
counting every line carrying a usage object, grouped by the line's **top-level**
`type` field:

```
assistant   142581    usage at .message.usage
user           252    usage at .toolUseResult.usage
```

**No other top-level type carries usage anywhere in the corpus.**
`queue-operation` and `attachment` carry it zero times. This confirms D-04 as
corrected and, in doing so, confirms why the original reading was wrong: it had
matched a *nested* `type` on a content block and mistaken it for the line type.

Two facts the correction did not record, both discovered here and both
load-bearing for the implementation:

1. **The two types carry usage at different positions.** `assistant` at
   `.message.usage`, `user` at `.toolUseResult.usage`. A single path would have
   found one of them.
2. **The `user` line is a dispatched subagent's completion record and has no
   `message.id` at all.** It carries `agentType` and `agentId` instead. It is
   therefore the plan's identifier-less case, not an incidental one.

### The mechanism, seen directly

Claude Code writes **one line per content block** of a reply — the thinking, the
text, each tool call — and every one of those lines repeats the same cumulative
usage object. Three consecutive fixture lines share `message.id`
`msg_011CdwU6bPPCyfEmc7tF5PXA` and `requestId` `req_011CdwU6aJe7MJmQWvwKGrrj`,
differ only in `uuid` and in which content block they carry, and each restates
`output_tokens: 971`.

### The fixture's two totals — the literals every test uses

| Total | Value | How measured |
|---|---|---|
| Naive — every usage-bearing occurrence added up | **2,764,260** | `jq` emitting one row per usage-bearing line, summed with `awk`, checked by hand addition |
| Deduplicated — the truth | **1,572,834** | Same rows; last occurrence wins per `message.id`; the identifier-less `user` row counted once |

Naive is **1.757x** the truth. Per-row: the session's own turns total
**1,341,032** (17 + 13,886 + 1,321,174 + 5,955) and the one subagent totals
**231,802** (2 + 330 + 230,803 + 667); 1,341,032 + 231,802 = 1,572,834.

**Neither superseded literal appears anywhere** in the reader, the tests, the
fixture, its README or this summary — the two figures are named only in
`196-CONTEXT.md`, where the correction records them as wrong. Verified by grep
across the repository.

## Accomplishments

- **`cmd/testdata/spend/claude/transcript-with-duplicates.jsonl`** — 28
  contiguous lines of one real session, redacted through a `jq` whitelist, with
  18 usage-bearing occurrences over 9 distinct `message.id` plus one
  identifier-less subagent row. Two ids occur three times, two occur twice,
  three occur exactly once, and non-usage line types (`last-prompt`, `mode`,
  `attachment`, ordinary tool results) are kept so the fixture also shows those
  being correctly skipped.
- **`TestClaudeTranscriptFixtureContainsRepeatedMessageIDs`** — the test that
  protects the other tests. It fails if the repetition is tidied away, if either
  line type disappears, if the identifier-less row goes, or if either measured
  total drifts.
- **`cmd/wrapper_usage_claude.go`** — the reader. Deduplicates by `message.id`
  (last wins), corroborated by `requestId`; reads the line type from the
  top-level field only; takes usage from its declared position; enforces the
  same projects-directory containment rule as the session record before opening
  anything; bounded by file size and by line size; tags every row
  session-transcript and never provider-grade.
- **Ten named tests** covering all six requirements harvested from the deleted
  branch plus the deduplication requirement and the top-level-line-type
  requirement.

## Task Commits

1. **Task 1: capture the fixture and its integrity test** — `2b3e3f50` (test)
2. **Task 2 RED: skeleton reader + nine failing tests** — `2e909dd8` (test)
3. **Task 2 GREEN: the reader** — `49e0739b` (feat)
4. **Follow-up: stop reproducing the superseded figures** — `95513b94` (docs)

## RED evidence (real output)

The skeleton reader was committed first with real signatures and empty bodies, so
the package **still compiled** and the tests failed for the right reason. The
deleted branch's mistake was the opposite: it committed tests against four
symbols that had never existed in any ref, turning a green repository red rather
than failing a check.

`go test ./cmd -run 'TestClaudeTranscript' -count=1`:

```
--- FAIL: TestClaudeTranscriptDeduplicatesUsageByMessageID (0.00s)
    wrapper_usage_claude_test.go:432: total = 0, want 1572834 (naive occurrence sum would be 2764260)
--- FAIL: TestClaudeTranscriptReadsBothUsageBearingLineTypes (0.00s)
    wrapper_usage_claude_test.go:483: got 0 worker row(s), want 2 (the session's own turns and one subagent): []
--- FAIL: TestClaudeTranscriptReadsTopLevelLineTypeNotNestedType (0.00s)
    wrapper_usage_claude_test.go:550: total = 0, want 110 — the queue-operation line's nested "assistant" type was treated as the line type ...
--- FAIL: TestClaudeTranscriptIgnoresUsageInsideToolDescription (0.00s)
    wrapper_usage_claude_test.go:574: total = 0, want 110 — a usage block quoted inside a tool description was counted
--- FAIL: TestClaudeTranscriptRefusesPathOutsideProjectsRoot (0.00s)
    --- FAIL: .../a_path_in_a_wholly_unrelated_directory
    --- FAIL: .../a_crafted_sibling_of_the_projects_root
    --- FAIL: .../a_traversal_out_of_the_projects_root
    --- FAIL: .../a_relative_path
    --- FAIL: .../an_empty_path
--- FAIL: TestClaudeTranscriptSkipsOversizedLineAndContinues (0.01s)
    wrapper_usage_claude_test.go:632: total = 0, want 330 — the oversized line must be skipped and BOTH surrounding lines still read
--- FAIL: TestClaudeTranscriptByteBoundStopsEarly (0.00s)
    wrapper_usage_claude_test.go:678: bounded total = 0, want 143153 ...
--- FAIL: TestClaudeTranscriptRowsAreNeverProviderGrade (0.00s)
    wrapper_usage_claude_test.go:709: no rows returned; this test would pass vacuously
--- FAIL: TestClaudeTranscriptLegacyColonFormIsTolerated (0.00s)
    wrapper_usage_claude_test.go:748: total = 0, want 110 — the legacy text usage block was counted; it is text, not accounting
FAIL
```

## Non-vacuity, demonstrated by breaking the code on purpose

Prose claiming a test "would catch" something is what this repository's audits
describe its own failures as. Each of these was produced by editing the shipped
reader, running the suite, recording the output, and restoring the file.

**1. Deduplication removed entirely** (every block keyed by line position):

```
--- FAIL: TestClaudeTranscriptDeduplicatesUsageByMessageID
    the reader billed 2764260, which is the sum of EVERY usage-bearing occurrence
    in the fixture. It is counting the same reply two and three times. The truth is 1572834.
--- FAIL: TestClaudeTranscriptReadsBothUsageBearingLineTypes
    session billed total = 2532458, want 1341032 — the subagent's row may have been folded into it
--- FAIL: TestClaudeTranscriptByteBoundStopsEarly
    bounded total = 429459, want 143153
```

**2. The line type read from the nested content-block field** (the superseded
reading, reconstructed):

```
--- FAIL: TestClaudeTranscriptReadsTopLevelLineTypeNotNestedType
    total = 3600110, want 110 — the queue-operation line's nested "assistant" type was
    treated as the line type, which is exactly the mistake that produced the superseded
    reading of D-04
```

3,600,110 is 110 plus the decoy's four 900,000 columns. The decoy is built the
way the plan demands: **usage-bearing, top-level type `queue-operation` (outside
the accepted set), nested `message.type` `assistant` (inside it)**. Built the
other way round a naive substring reader would skip the line for its own reasons
and the assertion would prove nothing.

**3. The fixture tidied so each `message.id` appears once** (18 lines kept, ids
made unique):

```
--- FAIL: TestClaudeTranscriptFixtureContainsRepeatedMessageIDs/at_least_one_message_id_repeats...
    fixture carries 17 distinct message id(s), want 9
--- FAIL: .../the_naive_and_deduplicated_totals_still_measure_what_the_README_says
    the fixture's naive sum (2764260) is not strictly larger than its deduplicated total
    (2764260) — there is no duplication left to catch
```

**4. One incidental finding worth recording.** Disabling *only* the `message.id`
key left the fixture still correctly deduplicated — because `requestId` is
identical across the duplicate lines and takes over. That is evidence the
corroborating identifier is real rather than decorative, and it is why proof (1)
above had to disable the whole key chain to produce the naive figure.

## Acceptance criteria — commands run and results

**Task 1**

| Criterion | Command / evidence | Result |
|---|---|---|
| Fixture-integrity test passes | `go test ./cmd -run 'TestClaudeTranscriptFixtureContainsRepeatedMessageIDs' -count=1` | PASS (4 subtests) |
| Fails if every id appears once | fixture tidied on disk, test re-run, fixture restored | FAIL as required (output above) |
| Both surviving types present | subtest `both surviving usage-bearing line types are present` | PASS |
| README states both totals as literals and names the redaction rules | `cmd/testdata/spend/claude/README.md` | PASS |
| Neither literal is a superseded figure | grep for both superseded figures across `cmd/` and this summary — no match | PASS |

**Task 2**

| Criterion | Command | Result |
|---|---|---|
| All nine named tests pass | `go test ./cmd -run 'TestClaudeTranscriptDeduplicatesUsageByMessageID\|...\|TestClaudeTranscriptLegacyColonFormIsTolerated' -count=1` | PASS |
| Dedup test asserts the literal AND that the naive sum is strictly larger | `TestClaudeTranscriptDeduplicatesUsageByMessageID` | PASS; fails with `2764260` when deduplication is removed |
| Both literals match the README and neither is superseded | constants `claudeFixtureNaiveOccurrenceSum` / `claudeFixtureDeduplicatedTotal` | PASS |
| Line-type assertion reads the top-level field; nested decoy changes nothing | `TestClaudeTranscriptReadsTopLevelLineTypeNotNestedType` | PASS; fails with `3600110` when the nested field is consulted |
| Path-refusal test uses a temporary home | `t.Setenv("HOME", t.TempDir())` in every case; the real projects directory is never opened by any test in the file | PASS |
| No expected value produced by calling the reader or the billed-total helper | every expectation is a `const` literal; the fixture walk uses untyped maps, not the reader's structs | PASS (by inspection) |
| `go vet ./cmd` clean | `go vet ./cmd` | PASS (exit 0) |

**Plan-level verification**

| Command | Result |
|---|---|
| `go test ./cmd -run 'TestClaudeTranscript' -count=1` | `ok github.com/calcosmic/Aether/cmd 0.625s` |
| `go build ./cmd/aether` | exit 0 |
| `go build ./...` | exit 0 |
| `go vet ./cmd ./pkg/codex` | exit 0 |
| `gofmt -l cmd/ pkg/` | no output |
| `go test ./cmd -run 'TestSpend' -count=1` | `ok` — 196-01's ledger and the session-capture tests still green |
| `go test ./cmd -run 'TestNoTokenCountIsDerivedFromLength\|TestBothAccountingPathsAgreeOnTheTotal\|TestWiringGuardsHaveNoRuntimeEscapeHatch\|TestWiringGateStepRunsEveryWiringTest' -count=1` | `ok` — 196-02's ratchets cover the new file and pass |
| `go test ./cmd -run 'Orphan\|Reachability\|DeadCode\|Unused' -count=1` | `ok` |
| Read the fixture by eye | Confirmed: `msg_011CdwU6bPPCyfEmc7tF5PXA` and `msg_011CdwW3P6c2VmnuTWw88aJc` each appear three times among usage-bearing lines; both `assistant` and `user` usage-bearing lines are present |

## Files Created/Modified

- `cmd/wrapper_usage_claude.go` — the reader: `parseClaudeTranscriptUsage`,
  `parseClaudeTranscriptUsageWithBounds`, the deduplication key, the per-worker
  accumulator and the bounded line reader
- `cmd/wrapper_usage_claude_test.go` — ten tests plus the fixture walk
- `cmd/testdata/spend/claude/transcript-with-duplicates.jsonl` — the real
  redacted capture
- `cmd/testdata/spend/claude/README.md` — provenance, redaction rules, corpus
  measurement and both totals
- `cmd/spend_session_capture.go` — `evalSpendPathSymlinks` replaces the
  containment helper's symlink fallback (see deviation 1)

## Decisions Made

- **Worker keying.** `assistant` lines are the orchestrating session's own turns
  and aggregate into one row named `main-session`. Each `user` /
  `toolUseResult` line is a dispatched subagent, keyed by `agentId` so two
  dispatches of the same type stay two workers, named by `agentType`
  (`gsd-executor` in the fixture). A completion record with neither is kept as
  its own row rather than merged, because a worker whose identity was not
  recorded is still a worker.
- **The deduplication key chain.** `message.id`, then `requestId`, then the
  line's `uuid`, then its position. The last two exist so an identifier-less
  usage-bearing line is counted once on its own — never dropped for lack of an
  identifier, and never folded into somebody else's spend.
- **`TotalTokens` left unset on every row.** Consumers reach the total through
  `codex.WorkerUsage.BilledTotalTokens()`. A second total-computing
  implementation in a second place is Pitfall 1.
- **"Tolerated" for the legacy colon form means accepted without error and not
  counted.** Reading a number out of a text blob is precisely the substring
  matching this plan forbids, so the legacy shape contributes nothing.
- **Decoys are constructed in-test, not added to the fixture.** The committed
  fixture stays an unedited real capture; the adversarial shapes it cannot
  supply are written to a temporary home directory by the test that needs them.
- **`bufio.Reader`, not `bufio.Scanner`.** Scanner abandons the whole file on a
  line longer than its buffer; the requirement is the opposite.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The containment helper refused a transcript path that did not exist yet**

- **Found during:** Task 2 (GREEN), surfaced by `TestClaudeTranscriptAbsentFileIsEmptyAndNoError`
- **Issue:** `validateSpendTranscriptPath` evaluated symlinks on both the root
  and the candidate, falling back to the *un-evaluated* path when
  `filepath.EvalSymlinks` failed. It fails for a path that does not exist. On
  macOS `/var` is a symlink to `/private/var`, so containment then compared an
  evaluated root against an unevaluated candidate and rejected a perfectly
  legitimate path with `escapes`. The containment answer depended on whether the
  file happened to have been written, which is not what containment means. The
  bug was latent for the existing caller (the hook always passes a live
  transcript) and immediate for this reader, which must answer "no rows, no
  error" for a session that never wrote one.
- **Fix:** `evalSpendPathSymlinks` resolves the deepest **existing** ancestor and
  rejoins the remainder. It cannot loosen containment: every segment that exists
  is still resolved, `filepath.Clean` has already collapsed any `..`, and the
  caller still decides containment with `filepath.Rel`.
- **Files modified:** `cmd/spend_session_capture.go`
- **Verification:** `go test ./cmd -run 'TestSpendSession' -count=1` — `ok`,
  including the crafted-sibling (`projects-evil`), relative-path, null-byte and
  outside-home refusals; and the reader's own five refusal subtests still fail
  closed.
- **Committed in:** `49e0739b`

**2. [Rule 2 - Missing Critical] A named test for the top-level-versus-nested line type**

- **Found during:** Task 2
- **Issue:** The plan's acceptance criterion "a line-type assertion reads the
  top-level field, and a fixture line whose nested content block carries a
  differently-named type does not change the result" had no named test in the
  criterion's own `-run` filter. Under CLAUDE.md's Definition of Done an
  unenforced criterion is unsatisfied, and this is the single criterion the
  whole plan exists to protect.
- **Fix:** Added `TestClaudeTranscriptReadsTopLevelLineTypeNotNestedType`,
  covered by the plan's `TestClaudeTranscript` verification filter, with the
  decoy built in the required orientation.
- **Verification:** Passes; fails with `total = 3600110` when the reader is made
  to consult the nested field.
- **Committed in:** `2e909dd8` (RED) / `49e0739b` (GREEN)

**3. [Rule 2 - Missing Critical] The fixture-integrity test also asserts both totals**

- **Found during:** Task 1
- **Issue:** As specified, the integrity test would have asserted only that a
  repetition exists. A future edit could keep one duplicated id while changing
  the numbers, leaving the reader's literals silently wrong.
- **Fix:** The integrity test recomputes both totals from the raw fixture with
  its own arithmetic and asserts the README's figures, plus that the naive sum is
  strictly larger than the deduplicated total.
- **Verification:** Both assertions fire on a tidied fixture (output above).
- **Committed in:** `2b3e3f50`

---

**Total deviations:** 3 auto-fixed (1 bug, 2 missing-critical).
**Impact on plan:** All three tighten requirements the plan already stated. One
file outside `files_modified` was touched (`cmd/spend_session_capture.go`), to
fix a containment bug the plan's own "absent file returns no rows and no error"
behaviour exposed. No architectural change, no new dependency, no scope creep.

## Issues Encountered

- **The corpus has grown since D-04 was written**, from 746 transcripts to 1,726.
  The re-measurement was therefore not a confirmation of the same numbers but a
  fresh one: 142,581 assistant and 252 user usage-bearing lines, against D-04's
  48,063 and 250. The *shape* of the finding — two usage-bearing types, nothing
  else — held exactly.
- **The `user` usage-bearing lines were nearly missed.** A first pass looking for
  `.message.usage` found only `assistant` lines and appeared to contradict D-04's
  `user 252`. A wider scan for a `usage` key at any path found them at
  `.toolUseResult.usage` — a different position, not a different absence. Had the
  reader been written from the first pass it would have silently missed every
  subagent's spend, which is exactly the number this phase exists to show.

## Known Stubs

None. Every symbol added is exercised by a test in the same commit.

The reader is **unwired by explicit instruction** — the plan states "Nothing
calls this reader yet. Plan 196-05 wires it into the finalize path in this same
phase and plan 196-08 fails if it is still unreached." This is a stated interface
constraint of the plan, not an omission, and it is recorded as coverage item D7
with `human_judgment: true` so it cannot pass unnoticed.

## Threat Flags

None. No new network endpoint, auth path or schema change at a trust boundary.
The one security-relevant change is a **tightening in effect and a correction in
kind**: `evalSpendPathSymlinks` makes containment resolve symlinks on the
existing part of a path that does not exist yet, where the previous fallback
compared a resolved root against an unresolved candidate. Every existing refusal
case — crafted sibling directory, relative path, null byte, outside the home tree
— still fails closed, re-run and passing.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Names plan 196-05 depends on:** `parseClaudeTranscriptUsage(path)`,
  `parseClaudeTranscriptUsageWithBounds(path, maxBytes)`, the
  `claudeTranscriptUsage` row (`WorkerName`, `AgentID`, `Usage`), and the
  constants `claudeTranscriptMainSessionWorker`,
  `claudeTranscriptMaxLineBytes`, `claudeTranscriptDefaultMaxBytes`.
- **What 196-05 must decide.** The reader returns one row for the orchestrating
  session's own turns and one row per dispatched subagent. The ledger writer has
  to choose whether the session's own row appears as a worker on the closeout
  line (D-01) or is presented separately; the reader deliberately does not make
  that decision for it.
- **`validateSpendTranscriptPath` is now safe to call on a path that has not been
  written yet.** 196-05 extracts the shared helper; `evalSpendPathSymlinks`
  should travel with it.
- **A row from this reader is never `Measured()`.** Any renderer that keys "is
  this a real number?" on `Measured()` would show every transcript-read worker as
  unmeasured. The ledger's measured/estimated split correctly uses `Estimated()`.

## Self-Check: PASSED

- `cmd/wrapper_usage_claude.go` — FOUND on disk
- `cmd/wrapper_usage_claude_test.go` — FOUND on disk
- `cmd/testdata/spend/claude/transcript-with-duplicates.jsonl` — FOUND on disk
- `cmd/testdata/spend/claude/README.md` — FOUND on disk
- `2b3e3f50`, `2e909dd8`, `49e0739b`, `95513b94` — all FOUND in `git log`
- All task acceptance criteria re-run after the final commit; all pass (tables above)
- Plan-level verification re-run after the final commit: `TestClaudeTranscript` ok,
  `go build ./...` exit 0, `go vet ./cmd ./pkg/codex` exit 0, `gofmt -l cmd/ pkg/` empty
- The fixture on disk is byte-identical to the redacted capture produced in Task 1,
  confirmed after every deliberate-breakage experiment

---
*Phase: 196-see-what-it-cost*
*Completed: 2026-08-28*
