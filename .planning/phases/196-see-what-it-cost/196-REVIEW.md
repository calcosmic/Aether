---
phase: 196-see-what-it-cost
reviewed: 2026-08-28T00:00:00Z
depth: standard
diff_base: d7efbc068dcb12f1d912855aa05db430bb921ea6
files_reviewed: 24
files_reviewed_list:
  - cmd/spend_ledger.go
  - cmd/spend_writer.go
  - cmd/spend_cmd.go
  - cmd/spend_cost_line.go
  - cmd/spend_session_capture.go
  - cmd/wrapper_usage_claude.go
  - cmd/wrapper_usage_opencode.go
  - cmd/wrapper_usage_resolve.go
  - cmd/caste_model_reason.go
  - cmd/codex_build.go
  - cmd/codex_build_finalize.go
  - cmd/codex_continue.go
  - cmd/codex_continue_finalize.go
  - cmd/codex_workflow_cmds.go
  - cmd/ceremony_cmd.go
  - cmd/ceremony_team_checkin.go
  - cmd/codex_visuals.go
  - pkg/llm/client.go
  - pkg/llm/streaming.go
  - pkg/agent/pool.go
  - pkg/codex/usage.go
  - pkg/codex/platform_dispatch.go
  - pkg/codex/worker.go
  - .github/workflows/ci.yml
findings:
  critical: 3
  warning: 7
  info: 5
  total: 15
status: issues_found
---

# Phase 196: Code Review Report

**Reviewed:** 2026-08-28
**Depth:** standard
**Files Reviewed:** 24 production files plus the new tests and fixtures
**Status:** issues_found

## Summary

The arithmetic core of this phase is sound. `billedTotal` sums all four disjoint
columns, `BilledTotalTokens` is the single entry point, the character-derived
estimator is genuinely deleted at source, `TestNoTokenCountIsDerivedFromLength`
is a real ratchet with an anti-vacuity floor and a planted-violation subtest, and
`TestBothAccountingPathsAgreeOnTheTotal` compares two independently produced
totals against a hand-written literal. No price table, no token-to-dollar
arithmetic, no currency on any headline. The path-containment helper is a single
implementation with symlink evaluation on both sides and `filepath.Rel`
containment; the unsynchronised regex cache that would have crashed the process
is gone.

The defects are not in the arithmetic. They are in **attribution** and in
**lifetime**, and three of them put a wrong number in front of the owner:

1. On Claude Code — the repo's primary platform — the ledger can never attribute
   a single token, because the identity the transcript records and the identity
   the ledger asks for are two different fields. The test that "proves" this path
   works fabricates a transcript shape Claude Code does not write.
2. Any second build of a phase — including Phase 195's own documented
   `recovery_command` partial retry, and the `build --force` path this repo's
   own continue emits after a block — erases the previous attempt's rows. I
   measured a phase that really cost 1,150,000 tokens reporting 250K.
3. An OpenCode session whose message directory is missing or empty is filed as a
   **measurement of zero**, counted in the "measured workers" tally, and pulled
   into the total.

Each finding below was reproduced by running the code, not inferred from
reading it. Scratch test files used to reproduce them have been deleted; the
tree builds clean.

## Critical Issues

### CR-01: On Claude Code the ledger can never attribute any worker's tokens

**File:** `cmd/wrapper_usage_claude.go:280-286`, `cmd/wrapper_usage_resolve.go:163-171`, `cmd/spend_writer.go:54-59`

**Issue:** The Claude reader names a subagent row by `toolUseResult.agentType`
(`wrapper_usage_claude.go:280`), which is the **agent definition** the wrapper
passed as `subagent_type` — `.claude/commands/ant/build.md:343` says exactly
that: *"Spawn the matching platform agent using `agent_name` as the subagent
type."* The phase's own real captured fixture confirms the value:
`cmd/testdata/spend/claude/transcript-with-duplicates.jsonl` line 6 carries
`"agentType":"gsd-executor"`.

The resolver then matches that name against the run's requested worker names
(`wrapper_usage_resolve.go:164`), and those names come from
`spendWorkerNameForDispatch`, which deliberately prefers `dispatch.Name` — the
deterministic per-worker name `Mason-67` — over `dispatch.AgentName`
(`spend_writer.go:54-59`). `Mason-67` and `aether-builder` never match, and the
`AgentID` fallback at `wrapper_usage_resolve.go:166` compares against a UUID.

Reproduced: feeding `parseClaudeTranscriptUsage` a line in the real shape
(`agentType: "aether-builder"`, `agentId` a uuid, 231,802 billed tokens) returns
`WorkerName="aether-builder"`. Running `writeSpendRowsForRun` with
`Platform: "claude"` and a dispatch `{Name: "Mason-67", AgentName:
"aether-builder"}` produces:

```
── What This Phase Has Cost ──
Cost: not known. The one worker that ran had no figure reported by its tool, so there is no total to show.
  🔨🐜 Builder Mason-67  —  not reported
```

Every real subagent row is instead routed to the `default` branch at
`wrapper_usage_resolve.go:168-171`, emitting *"the session transcript holds usage
for "aether-builder", which is not one of this run's workers"* — one note per
worker, on every build.

**Why no test caught it:** `cmd/wrapper_usage_resolve_test.go:54-62` builds its
transcript with `subagentLine("Mason-67", "agent-a", …)` — it writes the
deterministic worker name into the `agentType` field. That is a shape Claude
Code does not produce, and the phase's own real fixture (`agentType:
"gsd-executor"`) contradicts it. `TestBuildFinalizeFilesTheRunsRows` and
`TestContinueDoesNotEraseBuildRowsEndToEnd` use `Platform: "claude"` but assert
only that rows exist with the right name, caste and status — never that a figure
was attributed. This is the "test that cannot fail" class named in CLAUDE.md's
Definition of Done.

**Fix:** Match on both identities, definition name included, and keep the
deterministic name as the row key:

```go
// cmd/wrapper_usage_resolve.go — carry both identities into the request.
type wrapperUsageRequest struct {
    // ...
    // WorkerNames is the accounting key; AgentNameByWorker maps that key to
    // the agent DEFINITION the platform records as agentType.
    AgentNameByWorker map[string]string
}

// and in the claude branch:
byAgentName := map[string]string{} // agentType -> worker name
for worker, agentName := range req.AgentNameByWorker {
    if agentName != "" {
        byAgentName[agentName] = worker
    }
}
for _, row := range rows {
    // ... main-session branch unchanged ...
    switch {
    case seen[row.WorkerName]:
        measured[row.WorkerName] = row.Usage
    case byAgentName[row.WorkerName] != "":
        measured[byAgentName[row.WorkerName]] = row.Usage
    // ...
    }
}
```

Note the collision this exposes: two workers sharing one agent definition
(`aether-builder` twice in one build) both map to the same `agentType`, so
`byAgentName` must resolve them by dispatch order or `agentId`, not by
last-write-wins — the current `measured[row.WorkerName] = row.Usage` silently
drops all but the last. Then replace the resolver test's fabricated `agentType`
with the real one, and add an assertion to
`TestBuildFinalizeFilesTheRunsRows` that at least one row on the `claude`
platform carries `spendRowReportedUsage(row) == true`.

---

### CR-02: A re-run or partial retry of a phase erases the previous attempt's spend

**File:** `cmd/spend_writer.go:178-186`, `cmd/spend_ledger.go:172-207`

**Issue:** `saveSpendLedger` writes the workflow's ledger whole
(`store.SaveJSON(rel, entry)`, `spend_ledger.go:206`). The doc comment defends
this as the no-double-count invariant: *"Rerunning the same run's finalize
REPLACES this workflow's rows rather than appending to them."* That reasoning is
correct for re-finalizing **the same run**. It is wrong for a **second run of the
same phase**, which is a documented, routine flow here:

- Phase 195's `recovery_command` redispatches only the unfinished tasks
  (`.claude/commands/ant/build.md`), then calls `build-finalize` on the same
  phase.
- `continueNextCommandForBlocked` emits `build --force` / `force-redispatch`
  after a blocked check (`cmd/codex_continue.go:849, 929`).

Nothing distinguishes the two cases. `spendLedger.RunID` exists at
`spend_ledger.go:154` precisely to make that distinction possible and is **never
written and never read** anywhere in the tree.

Reproduced end to end. Attempt 1 files three workers totalling 900,000 tokens
(a 500K builder, a 300K failed builder, a 100K watcher). The recovery redispatch
of the one unfinished task then files one worker at 250,000, and the phase's
cost line becomes:

```
── What This Phase Has Cost ──
Cost: 250K tokens for the one worker that ran, whose tool reported a figure.
  🔨🐜 Builder Anvil-20  250K  measured
```

1,150,000 tokens were really spent on that phase; the owner is shown 250K, with
no note that anything is missing, under a heading that says *"What This Phase Has
Cost"*. The failed first attempt — exactly the spend the owner most wants to see
— is the part that disappears. This directly contradicts the writer's own stated
rule at `spend_writer.go:80-82`: *"A worker the platform reported nothing for
still gets a row … so it cannot vanish and make the run look cheaper than it
was."*

`TestBuildFinalizeRerunReplacesItsOwnRowsOnly` (`spend_writer_test.go:215`) locks
the replace in, but only for a re-finalize of an identical dispatch set — it
cannot see the attempt case.

**Fix:** Key replacement by run identity, not by phase+workflow alone. Populate
`RunID` from the attempt id the build already owns and merge by run:

```go
// cmd/spend_writer.go
existing, _ := loadSpendLedger(req.Phase, req.Workflow)
merged := make([]spendRow, 0, len(existing.Rows)+len(rows))
for _, row := range existing.Rows {
    if row.RunID != req.RunID { // rows from earlier attempts survive
        merged = append(merged, row)
    }
}
merged = append(merged, rows...)
```

with `RunID` added to `spendRow` and set from `manifest.AttemptID` (platform
lane) / the direct lane's attempt record. Re-finalizing the same run still
replaces exactly its own rows; a second attempt adds to the phase instead of
erasing it. Add `TestRetryDoesNotEraseTheFirstAttemptsSpend` asserting the phase
total after a partial-retry finalize is the sum of both attempts.

---

### CR-03: An OpenCode session with no readable messages is filed as a measured zero

**File:** `cmd/wrapper_usage_opencode.go:384-391`, `cmd/spend_cmd.go:74-76`

**Issue:** `readOpenCodeUsage` constructs its return value with the source tag
already set (`usage := codex.WorkerUsage{Source: codex.UsageSourceSessionTranscript}`,
line 385) and returns it unchanged when `os.ReadDir` on
`<root>/message/<sessionID>/` fails (lines 388-391), and equally when the
directory exists but holds no `assistant`-role records.

Every downstream "was this reported?" decision keys on the source tag, by design
(`wrapperUsageWasReported`, `wrapper_usage_resolve.go:106-108`;
`spendRowReportedUsage`, `spend_cmd.go:74-76`). So a session that was matched by
title but whose messages could not be read becomes a row that says the worker
cost **zero tokens, measured**.

Reproduced: `readOpenCodeUsage(tmpdir, "ses_nonexistent")` returns
`{Source:"session-transcript"}` with every column zero; `Empty()` is `false`,
`spendRowReportedUsage` is `true`, the rendered figure is `"0"`, and
`computeSpendTotals` reports `MeasuredRows: 1, MeasuredTokens: 0`.

The consequences the owner sees: the worker's real spend is silently excluded
from the total; the "N workers whose tools reported a figure" count is inflated;
and the footnote that would have said something was missing is suppressed,
because `unreported` is zero. This is the precise outcome D-01 as amended exists
to prevent — *"a worker whose tool reported nothing shows NO number. Not a zero,
which reads as 'this worker was free'"*.

**Fix:** Only tag the usage once something was actually read:

```go
func readOpenCodeUsage(root, sessionID string) codex.WorkerUsage {
    var usage codex.WorkerUsage
    dir := filepath.Join(root, "message", sessionID)
    entries, err := os.ReadDir(dir)
    if err != nil {
        return codex.WorkerUsage{} // nothing read: not reported, never a zero
    }
    read := 0
    for _, entry := range entries {
        // ... existing filtering ...
        read++
        usage.InputTokens += jsonNumberOrZero(msg.Tokens.Input)
        // ...
    }
    if read == 0 {
        return codex.WorkerUsage{}
    }
    usage.Source = codex.UsageSourceSessionTranscript
    return usage
}
```

Lock it with a test that matches a session by title, deletes its message
directory, and asserts the worker renders with the dash rather than `0`.

## Warnings

### WR-01: Every OpenCode diagnostic the reader constructs is thrown away

**File:** `cmd/wrapper_usage_opencode.go:234`, `cmd/wrapper_usage_resolve.go:174-183`

**Issue:** `openCodeSessionUsageForRun` returns `(entries, reasons)` and
documents the second value as *"a slice of plain-English diagnostics … because a
partial, honestly-reported result is the correct outcome here"*. Its only caller
discards it: `entries, _ := openCodeSessionUsageForRun(...)` at line 234. The
resolver's OpenCode branch appends nothing to `res.Diagnostics` except its own
"no store found" note.

So all three carefully written notes are unreachable in production: *"worker %q
matched %d in-window opencode sessions; refusing to guess"* (line 213), *"no
opencode project matches worktree %q"* (line 175), and *"opencode storage root
rejected"* (line 167). Concretely: a build where two sessions share a worker name
resolves that worker to no figure, and the owner is told nothing about why — the
row simply reads "not reported" with no distinction from a tool that genuinely
said nothing. `TestOpenCodeSessionUsageRefusesAmbiguousWorkerMatch` calls the
inner function directly, so it never sees the drop.

**Fix:** Return the reasons through `openCodeSessionUsageForRunOrNone` and append
them to `res.Diagnostics` in the resolver's OpenCode branch, exactly as the
Claude branch does with its unattributed note.

---

### WR-02: Two dispatches sharing a name are each credited the full measurement

**File:** `cmd/spend_writer.go:129-167`

**Issue:** The resolver deduplicates requested names
(`wrapper_usage_resolve.go:128-136`), so `usageByWorker` holds one entry per
distinct name. The row loop then iterates over **every dispatch**
(`spend_writer.go:135`) and looks up `usageByWorker[name]` for each, so N
dispatches sharing one name each receive the full measurement.

Reproduced: two dispatches named `Vigil-12`, one attached measurement of 1,000
tokens, produces a ledger of two measured rows and a cost line reading
`Cost: 2.0K tokens across 2 workers` for a run that spent 1,000.

Names are `deterministicAntName(caste, "phase:N:<caste>")` and are unique across
the current dispatch tables (the `watcher` caste is explicitly excluded from
continue review specs at `codex_continue.go:1438`, and the pre-wave and post-wave
tables are disjoint), so I could not find a live path today. Nothing enforces
that invariant: `saveSpendLedger` validates row *statuses* by name
(`spend_ledger.go:183-202`) but never checks name uniqueness, and one new caste
added to two tables reintroduces the doubling silently.

**Fix:** Refuse or disambiguate duplicates in `saveSpendLedger`, in the same
shape as the status refusal:

```go
seen := map[string]bool{}
for i := range rows {
    if seen[rows[i].AgentName] {
        return fmt.Errorf("save spend ledger: two rows are filed under worker %s — "+
            "one measurement would be credited to both", rows[i].AgentName)
    }
    seen[rows[i].AgentName] = true
}
```

---

### WR-03: The OpenCode reader sums two representations of the same quantity and the consumer prefers the fragile one

**File:** `cmd/wrapper_usage_opencode.go:413-417`, `pkg/codex/usage.go:294-299`

**Issue:** `readOpenCodeUsage` accumulates the four disjoint columns **and**
`tokens.total` into `usage.TotalTokens` (line 417). `BilledTotalTokens` then
returns `TotalTokens` whenever it is positive, ignoring the columns entirely
(`usage.go:295-298`).

The fixture README records that `tokens.total == input + output + cache.read +
cache.write` held on one 23-message real session, so the two agree today. They
stop agreeing the moment a single message record omits `total` or reports it as
zero while its columns are populated — an aborted or errored assistant turn is
the obvious candidate. A session of four messages where one lacks `total` reports
the sum of the other three as the worker's whole spend, labelled `measured`, with
nothing anywhere signalling the shortfall. The fixture has no such record, so no
test can catch it.

**Fix:** Do not sum `total` at all — let the authoritative helper derive it from
the columns, which is what the rest of the phase mandates:

```go
// Deliberately not summed: the authoritative total is BilledTotalTokens()
// over the four disjoint columns. Reading the store's own `total` alongside
// them means two answers to one question, and the consumer silently prefers
// the one a single malformed record can shrink.
usage.InputTokens += jsonNumberOrZero(msg.Tokens.Input)
usage.OutputTokens += jsonNumberOrZero(msg.Tokens.Output)
usage.CachedInputTokens += jsonNumberOrZero(msg.Tokens.Cache.Read)
usage.CacheCreationTokens += jsonNumberOrZero(msg.Tokens.Cache.Write)
```

Add a fixture message with the columns present and `total` absent, and assert
the worker's billed total still equals the hand-summed columns.

---

### WR-04: Three new ledger features are orphans, and one of them cannot work even if called

**File:** `cmd/spend_ledger.go:325-338`, `cmd/spend_ledger.go:342-377`, `cmd/spend_ledger.go:63-95`

**Issue:** `spendPerWorkerAverageTokens` and `spendRollupByParent` (with its
`spendParentRollup` type) have no production caller anywhere in `cmd/`. Their
only callers are their own tests, `TestLedgerRefusesDerivedMetricOverEstimates`
and `TestLedgerParentRollupCountsEachWorkerOnce`. The phase's own anti-orphan
check, `TestNothingThisPhaseAddedIsUncalled`
(`spend_pipeline_e2e_test.go:460`), lists nine symbols and omits exactly these —
so the guard written to catch this failure mode was scoped around it.

Worse, `spendRollupByParent` groups on `spendRow.ParentName`, and **no production
code ever sets `ParentName`**: `writeSpendRowsForRun` constructs each row with
`AgentName`, `Caste`, `Task`, `JobName`, `Status`, `Usage` and nothing else
(`spend_writer.go:155-164`). Called against a real ledger it would return a
single `(unattributed)` bucket. Its test seeds `ParentName` by hand, so it passes
over a field production never fills. `spendRow.ToolCount` (line 93) and
`spendLedger.RunID` (line 154) are likewise declared, never written, never read.

**Fix:** Either wire the roll-up into `aether spend` and populate `ParentName`
from `agent.SpawnEntry` at write time, or delete
`spendPerWorkerAverageTokens`, `spendRollupByParent`, `spendParentRollup`,
`spendRow.ParentName` and `spendRow.ToolCount` and their tests. If the roll-up
stays, add `ParentName` to the anti-orphan check's list and assert in
`TestBuildFinalizeFilesTheRunsRows` that filed rows carry a non-empty parent.
(`RunID` is better kept and populated — see CR-02.)

---

### WR-05: The team card tells OpenCode users which model each worker runs on, and OpenCode does not route models that way

**File:** `cmd/ceremony_team_checkin.go:89, 136-150`, `cmd/ceremony_team_checkin.go:584-590`

**Issue:** Every line of the check-in card now ends with either *"(kept on the
more expensive model because it …)"* or *"(the cheaper model)"*, derived from
`resolveCasteModel` → `casteModelSlot`, which
`cmd/codex_visuals.go:4540-4548` documents as mirroring
`.claude/agents/ant/aether-<role>.md` frontmatter.

`.opencode/commands/ant/build.md:297` renders the same card from the same runtime
renderer, and no file under `.opencode/agents/` carries a `model:` line at all —
I checked all 27. On OpenCode the model is whatever the user has configured, so
the card states a routing fact that is not true on that platform, in a surface
whose entire purpose is telling a non-technical owner what he is about to pay
for. `TestCasteModelSlotMatchesAgentFrontmatter` only reads `.claude/`, so
nothing detects the divergence.

CLAUDE.md's Platform Policy names OpenCode a primary platform, and its
Definition of Done requires that *"a documentation claim about runtime behaviour
must be testable or removed"* — this is the same rule applied to a runtime-
rendered claim.

**Fix:** Gate the model clause on the platform actually routing by role:

```go
if !platformRoutesModelsByRole(buildHostPlatform()) {
    // no model clause: the platform does not route per role, so naming one
    // would assert something this card cannot know.
} else if modelReason := modelReasons[caste]; modelReason != "" {
    // ...
}
```

and extend `TestCasteModelSlotMatchesAgentFrontmatter` to fail if an
`.opencode/agents/*.md` gains a `model:` line that disagrees with the table.

---

### WR-06: The closeout cost-line tests never exercise the branch real runs take

**File:** `cmd/ceremony_cmd.go:610-613, 638-642`, `cmd/ceremony_closeout_spend_test.go:67-77`

**Issue:** The closeout resolves the phase as `completion_phase`, falling back to
`current_phase` (lines 610-612 and 638-640). `completion_phase` is set only when
the completion packet carries a manifest key that `closeoutManifest`
(`cmd/closeout_cmd.go:213-231`) recognises.

`costLineCompletionFileForTest` writes a packet with a bare top-level `"phase":
1` and no manifest key at all, so `closeoutManifest` finds nothing. I confirmed
by instrumenting `renderCeremonyCloseout("build", …)`: `completion_phase=<nil>,
current_phase=1`. Every one of `TestBuildEndsWithOneCostLine`,
`TestContinueEndsWithOneCostLine`, `TestCloseoutCostLineCountIsAssertedNotAssumed`
and `TestNoLaneRendersTwoCostLines` therefore exercises only the fallback. The
`completion_phase` line — the one real build and continue packets take, since
build packets are required to carry `dispatch_manifest`
(`cmd/build_attempt.go:636`) and continue packets carry `continue_manifest`
(`.claude/commands/ant/continue.md:191`) — could be deleted without failing a
test.

The fallback is also wrong when reached: `closeout_cmd.go:46` reads
`state.CurrentPhase` **after** the check has advanced the colony, so a packet
without a manifest would render the *next* phase's ledger — an empty block
("No token use was recorded for this phase") at the end of a check that just
spent real tokens, or, if the next phase already has a build recorded, another
phase's numbers under this one's heading.

**Fix:** Give the fixture packet a real `dispatch_manifest` / `continue_manifest`
so the tests drive the production branch, and add a case asserting the cost block
names the phase that was just run when `current_phase` has already advanced past
it.

---

### WR-07: The orchestrating session's tokens are read, kept, then dropped — under a heading that claims the phase's cost

**File:** `cmd/wrapper_usage_resolve.go:64-73, 154-161`, `cmd/spend_writer.go:119-132`, `cmd/spend_cost_line.go:52`

**Issue:** The Claude reader separates the session's own turns into
`SessionUsage`/`SessionReported`, documented as *"kept apart rather than folded
into any worker's row or silently discarded"*. `writeSpendRowsForRun` — the only
production consumer of `wrapperUsageResolution` — reads `resolution.Workers` and
`resolution.Diagnostics` and never touches `SessionUsage`. It is discarded.

That matters because of the corpus figures this phase measured itself
(`cmd/testdata/spend/claude/README.md`): 142,581 usage-bearing `assistant` lines
against 252 subagent completion records. The orchestrating session is where the
overwhelming majority of a run's tokens live. The block is headed *"What This
Phase Has Cost"* (`spend_cost_line.go:52`), with a comment arguing that a
narrower heading *"would be the more precise-sounding lie"* — but the total under
it excludes the largest component of the phase's real spend, and the footnote
only ever mentions unreported **workers**.

**Fix:** Either file the session as its own row (caste `session`, name "the main
session") so it appears in the breakdown and the total, or change the heading and
total sentence to say what they measure — "What the workers cost" — and add a
line stating that the main session's own use is not counted. Whichever is chosen,
delete `SessionUsage`/`SessionReported` if nothing consumes them.

## Info

### IN-01: `capitalizeFirstRune` slices bytes, not runes

**File:** `cmd/spend_cmd.go:216-221`

**Issue:** `strings.ToUpper(text[:1]) + text[1:]` indexes bytes despite the name.
Safe only because every current caller passes a string beginning with an ASCII
digit (`spendWorkerWord`). A future caller passing a multi-byte first rune
produces mojibake.

**Fix:** Use `utf8.DecodeRuneInString`, or rename the function to
`capitalizeFirstByte` so the constraint is visible at the call site.

---

### IN-02: The detail view uses the rune-count width the cost line's own comment says is wrong

**File:** `cmd/spend_cmd.go:116-128, 164`, `cmd/spend_cost_line.go:196-216`

**Issue:** `spendDisplayWidth` was written because *"a caste glyph occupies two
columns while an emoji variation selector occupies none"*, and the cost line uses
it. `renderSpendText`/`spendColumnWidths` still pad with `len([]rune(plain))`, so
`aether spend` misaligns its figure column for exactly the mixed
`👁️🐜 Watcher` / `🔨🐜 Builder` case the helper exists to fix — two width
implementations for one job, in one phase.

**Fix:** Call `spendDisplayWidth` from `spendColumnWidths` and `renderSpendText`.

---

### IN-03: A negative token count is silently shown as positive

**File:** `cmd/spend_cost_line.go:62-64`

**Issue:** `if count < 0 { count = -count }` renders a hand-edited or corrupt
ledger row of `-500000` as `500K` with no sign and no note. The ledger is a JSON
file on disk with no numeric range validation in `saveSpendLedger`.

**Fix:** Reject a negative token count in `saveSpendLedger` by name, in the same
shape as the status refusal, rather than absolutising it at the renderer.

---

### IN-04: A live provider SDK stream is tagged `session-transcript`

**File:** `pkg/agent/pool.go` (`WorkerUsageFromStreamUsage`), `pkg/codex/usage.go:49`

**Issue:** The agent pool's usage comes from the Anthropic SDK's own streamed
`Usage` — a provider-grade measurement by any reading — but is tagged
`UsageSourceSessionTranscript`, on the stated rule that only `ParseUsage` may set
the provider tag. The consequence is that `Measured()` returns false for it, so
`computeSpendTotals.ProviderRows` (`spend_ledger.go:309-311`) undercounts
provider-grade rows. Nothing renders `ProviderRows` today, so nothing visible is
wrong — but the field now means something different from its name.

**Fix:** Either widen the provider tag's rule to "read by the Go runtime from a
provider API response, wherever that happens", or drop `ProviderRows` from
`spendTotals` until something reads it.

---

### IN-05: The anti-orphan check matches substrings, not calls

**File:** `cmd/spend_pipeline_e2e_test.go:521-542`

**Issue:** `productionCallersOf` scans for any non-comment line containing the
symbol, skipping only the declaring line. A mention inside a string literal, a
longer identifier that contains the name, or a struct-field reference all count
as a "caller". It is a useful floor and it caught the real orphan it was written
for, but it cannot distinguish a call from a mention — and it silently permits
the orphans in WR-04 by simply not listing them.

**Fix:** Parse with `go/ast` (the same package's other guards already do) and
count `*ast.CallExpr` nodes, then add every symbol this phase introduced rather
than a hand-picked nine.

---

_Reviewed: 2026-08-28_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
