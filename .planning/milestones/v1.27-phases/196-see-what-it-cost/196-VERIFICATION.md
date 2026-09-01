---
phase: 196-see-what-it-cost
verified: 2026-08-28T17:10:00Z
reverified: 2026-08-28T18:05:00Z
status: passed
score: 6/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 5/6
  scope: >-
    Narrow re-check of the two gaps only, plus the disposition accuracy correction.
    The five ROADMAP success criteria were NOT re-verified — they were proved by
    mutation in the initial pass and nothing in commits 6d0df805, afc9963b or
    0a238274 touches their proofs (confirmed: the closure test run is green).
  gaps_closed:
    - "196-08 must_have: 'No symbol added by this phase is code with no caller.' — closed by 6d0df805 (deletions) and afc9963b (the rebuilt guard). Verified independently by planting orphans on disk, not by reading the commit."
    - "Knowingly-deferred findings are written down so they are actionable — closed by 0a238274 (D3, D4, D5 added to deferred-items.md)."
  gaps_remaining: []
  regressions: []
  residual_warnings:
    - "The guard's FIELD rule matches selectors by bare name within this phase's own files, so a dead field whose name collides with a live field on another phase struct escapes silently. Proved: adding a never-written, never-read `Notes []string` to spendRow passes the guard, because spendWriteOutcome.Notes is selected in spend_writer.go. Same species as the package-scope ToolCount hole that was closed, narrowed to 9 files. The documented limitation names only the false-POSITIVE direction and omits this false-NEGATIVE one."
    - "cmd/spend_ledger.go:79 names `TestEveryFieldThisPhaseAddedIsUsed`, a test that does not exist anywhere in the tree. The check it refers to is real but lives inside TestNothingThisPhaseAddedIsUncalled."
gaps: []
deferred: []
---

# Phase 196: See What It Cost — Verification Report

**Phase Goal:** Every build and continue ends with one honest, plain-English line stating token usage — never a dollar figure as the headline. The team card also shows which AI model each worker uses and why, so nothing quietly uses an expensive model with no reason on record.

**Verified:** 2026-08-28T17:10:00Z
**Status:** passed (initial pass: gaps_found; both gaps closed and independently re-verified 2026-08-28)
**Re-verification:** Yes — narrow re-check of the two gaps, recorded in "Gap Closure" below
**Method:** goal-backward. Every criterion was proved by running a command, then by
BREAKING the production code and confirming that command fails. Every mutation was
restored byte-identically from a hash-checked copy (never `git stash`, never
checkout). Final `git diff --stat` is empty; `git status --porcelain` shows only the
pre-existing untracked files from other phases.

## Goal Achievement

### Observable Truths — the five ROADMAP Success Criteria

| # | Truth (verbatim from ROADMAP, criterion 1 as corrected by plan 196-08) | Status | Evidence |
|---|---|---|---|
| 1 | Every build and continue ends with one line stating token usage per worker and in total, showing no number at all for a worker whose tool reported none and counting only the measured workers in the total — no dollar amount as the headline, no price table anywhere (`TestBuildEndsWithOneCostLine`). | ✓ VERIFIED | Named test passes; real binary renders the exact D-01 shape; three mutations each fail a named test. See Behavioural Spot-Checks and Mutation Proofs. |
| 2 | On the Claude Code / OpenCode chat path, worker results report their real token usage, so that cost line has real numbers instead of blanks. | ✓ VERIFIED | The transcript join validated against 400 of the owner's real transcripts (32/32 joined); the transcript path is captured by the platform's own hook, proved end to end with the real binary; disabling the join fails four named tests. |
| 3 | Running `aether spend` shows token usage per worker for the current run without changing any files on disk, and none of its numbers are guessed from text length. | ✓ VERIFIED | Real run against a seeded colony: per-worker view rendered, data directory byte-identical before and after. Ratchet scans four packages and self-verifies with a planted violation; a repo-wide search found no length-derived token count outside the scan. |
| 4 | The pre-build team card names each worker's AI model and the reason for that choice; routine roles no longer silently default to `inherit`, and any worker kept on the expensive model has a written reason (`TestRoutineBuilderIsSonnetNeverInherit`, `TestOpusRequiresRecordedReason`). | ✓ VERIFIED | Real card render shows `[sonnet]` / `[opus]` plus the written reason. Two mutations each fail the named tests. All 27 Claude agent files carry a pinned model; none says `inherit`. Caveat WR-05 below. |
| 5 | Three abandoned branches of half-finished cost-tracking work are each reviewed and either merged in (with a written reason) or deleted (with a written reason) — none left dangling. | ✓ VERIFIED | `git branch -a --list "*worktree-agent*"` returns 0. All three tip commits recorded before deletion and still resolvable. `TestBranchDispositionRecordsAllThreeBranches` passes. |

### Observable Truths — plan must_haves not covered by a roadmap criterion

| # | Truth (196-08-PLAN.md) | Status | Evidence |
|---|---|---|---|
| 6 | "No symbol added by this phase is code with no caller." | ✓ VERIFIED (was ✗ FAILED) | Closed by 6d0df805 + afc9963b. Re-verified by planting orphans on disk — including the chain and cycle shapes the old guard could not see. See Gap Closure. |

**Score:** 6/6 truths verified (0 present, behavior-unverified) — 5/6 at initial verification, raised after independent re-check of the closure

### Correction check (criterion 1 / COST-01)

Asked to confirm no criterion still describes the superseded "measured versus
estimated" shape.

```
$ grep -rn "measured versus estimated\|measured vs estimated\|marking which numbers" .planning/ROADMAP.md .planning/REQUIREMENTS.md
.planning/ROADMAP.md:210:   *Corrected 2026-08-28 ... previously said "clearly marking which numbers are measured versus estimated" ...*
.planning/REQUIREMENTS.md:39: ... *Wording corrected 2026-08-28 ... this line previously promised "measured vs estimated labelled" ...*
```

Both hits are inside the correction notes themselves, which say "previously said".
No live criterion or requirement describes the superseded shape. ✓ CORRECT.

### Behavioural Spot-Checks (real binary, real commands)

A binary was built from the working tree (`go build -o <scratch>/aether ./cmd/aether`)
and run against a seeded temporary colony outside the repository.

| # | Behaviour | Command | Result | Status |
|---|---|---|---|---|
| B1 | The closeout renders exactly one cost block in the D-01 shape, on the production `completion_phase` branch, for a packet carrying a real `dispatch_manifest` | `aether ceremony closeout --workflow build --completion-file packet-build.json` | `── What The Helpers Cost ──` / `Cost: 1.0M tokens across 2 workers. The total counts only the 1 whose tools reported a figure.` / `Builder Mason-67  1.0M  measured` / `Scout Roam-90  —  not reported` / `(1 worker's tool did not report usage, so its use is not in the total.)` — and it named **phase 1** while the colony's `current_phase` was **2**, proving the production branch resolves the phase correctly | ✓ PASS |
| B2 | `aether spend` shows per-worker tokens and mutates nothing | `aether spend --phase 1`, with a SHA-256 snapshot of every file under `.aether` taken before and after | `Builder Mason-67  1002550  measured` / `Scout Roam-90  —  not reported` / `Total: 1002550 tokens, counting only the 1 worker whose tools reported a figure.` — and `diff before.txt after.txt` empty. The figure 1002550 is the correct disjoint sum 50 + 1,000,000 + 2,000 + 500, not input+output | ✓ PASS |
| B3 | The team check-in card names each worker's model and the written reason | `aether ceremony team-checkin --workflow build --manifest-file manifest.json` | `🔨🐜 Builder [sonnet] … (the cheaper model)` / `👥🐜 Auditor [opus] … (kept on the more expensive model because it judges whether finished code is genuinely good rather than merely working…)` | ✓ PASS |
| B4 | The platform's own hook records the transcript path (the thing an orchestrating model cannot fabricate) | `echo '{"session_id":…,"transcript_path":…}' \| aether hook-pre-tool-use` | `.aether/data/spend/session.json` written with `platform: claude-code` and the transcript path | ✓ PASS |
| B5 | The three branches are gone and their commits still resolvable | `git branch -a --list "*worktree-agent*"` → 0; `git cat-file -t <each tip>` | 0 branches; all three tips resolve to `commit` | ✓ PASS |

### Mutation Proofs (break the code, confirm the command fails)

Per CLAUDE.md's Definition of Done — a requirement is satisfied only when a command
exists that FAILS when it is unmet. Each mutation was applied to production source,
the test run, then restored from a SHA-256-verified copy.

| # | Criterion | Mutation applied to production code | Result | Verdict |
|---|---|---|---|---|
| M1 | 1 | `spendCostLineFigure` returns `"0"` instead of the dash for an unreported worker | `--- FAIL: TestCostLineShowsNoTotalWhenNothingWasMeasured` — `Mason-67's figure cell = "0", want the dash sentinel` | ✓ LOCKED |
| M2 | 1 | Injected `Estimated bill: $NN.NN` above the total sentence in `renderSpendCostLineFromLedgers` | `--- FAIL: TestBuildEndsWithOneCostLine`, `--- FAIL: TestContinueEndsWithOneCostLine`, `--- FAIL: TestCostLineHasNoMonetaryFigure` | ✓ LOCKED |
| M3 | 1 | Deleted the `completion_phase` branch in `renderCeremonyCloseout` (the branch real packets take), leaving only the `current_phase` fallback | `ok github.com/calcosmic/Aether/cmd` — the entire spend / cost / closeout / e2e test set stayed **green** | ✗ NOT LOCKED (WR-06, see Warnings) |
| M4 | 2 | Disabled the dispatch-description join in `claudeRowWorker` (`description := ""`) | `--- FAIL: TestClaudeBuildAttributesEveryWorkersTokens`, `TestResolverReadsTranscriptOnTheClaudePath`, `TestClaudeTranscriptJoinsOnWhatThePlatformRecords`, `TestTheUnmatchedRecordNoteDoesNotBlameAWorkerThatRan` | ✓ LOCKED |
| M5 | 4 | Set `model: inherit` in `.claude/agents/ant/aether-chronicler.md` | `--- FAIL: TestRoutineBuilderIsSonnetNeverInherit` (twice, by name) and `--- FAIL: TestCasteModelSlotMatchesAgentFrontmatter` | ✓ LOCKED |
| M6 | 4 | Blanked the auditor's recorded reason in `cmd/caste_model_reason.go` | `--- FAIL: TestOpusRequiresRecordedReason` — `auditor runs on the expensive model but records no reason why` | ✓ LOCKED |

### Fixture-Fidelity Audit (the failure shape this phase kept hitting)

Every fault in this phase traced to a fixture built in a shape the runtime cannot
produce, or a test passing through a different route than the one it is named for.
Three specific checks were run.

**1. The Claude transcript join — does the fixture match what the platform really writes?**

Scanned the owner's real transcripts under `~/.claude/projects` (1,799 files; 120
sampled for shapes, 400 for the join):

```
usage-bearing lines by shape:
  ('assistant', 'message.usage')          9773
  ('user',      'toolUseResult.usage')      52
```

Exactly the two shapes the parser handles — no third shape exists, and `queue-operation`
/ `attachment` carry usage zero times, confirming D-04 as corrected rather than the
superseded reading. The join was then measured directly:

```
usage-bearing user lines: 32; joined to a tool_use dispatch: 32; joined-but-no-description: 0
sample: ({'desc': 'Plan Phase 163.1', 'st': 'gsd-planner'}, agentType='gsd-planner', totalTokens=238359)
```

32 of 32 real subagent records join back to their dispatching `tool_use` block by
`tool_use_id`, every one carrying a description. **The fixture's mechanism is real,
not invented.** The only cosmetic divergence is the fixture writing `"name":"Agent"`
where the platform writes the tool name — immaterial, because the parser keys on the
presence of `input.subagent_type`, never on the tool name.

The committed fixture `cmd/testdata/spend/claude/transcript-with-duplicates.jsonl`
keeps its duplicates: 17 usage-bearing assistant lines carrying 9 distinct
`message.id`, six of them repeated (one three times). `TestClaudeTranscriptFixtureContainsRepeatedMessageIDs`
additionally fails if every id repeats — so a reader that "deduplicated" by keeping
the first of each pair could not pass either.

**2. The retry path — does the fixture use names the runtime can actually generate?**

`deterministicAntName` is a pure function of caste and seed, so a re-run of one phase
regenerates the SAME worker name. `TestARetryIsNotCreditedWithTheFirstAttemptsTokens`
now uses one identical description for both attempts (the shape the runtime produces)
and separates them by the run's own time window: the retry reports 40,000, the first
attempt 1,000,000. At the writer layer,
`cmd/spend_writer_test.go:772-774` derives its names from the real generator
(`deterministicAntName("builder", "phase:9:task:0:write the reader")`) rather than
typing literals. **IN-02 is genuinely closed at the load-bearing site.**

**3. Criterion 3's ratchet — does its scan cover the packages where such code could live?**

`lengthScanPackages` scans `cmd`, `pkg/codex`, `pkg/llm`, `pkg/agent` — by package,
not by filename, with one stated exemption (`_test.go`) and no exemption list. Its own
subtests confirm it catches a planted violation in every scanned package. An
independent repo-wide search found no length-derived token count anywhere outside the
scan. Two near-misses were checked and cleared:

- `pkg/terminal/progress.go`'s `EstimatedTotal` is a progress-bar denominator with no
  production caller (`SetEstimatedTotal` is referenced only by its own tests) and
  never reaches a cost figure.
- `pkg/codex.UsageSourceEstimate` still exists as a constant, but **no production code
  produces it** — every reference outside the type is a test. The estimate path is
  dead in production, and `cmd/spend_cost_line_test.go:132` proves such a row would
  still render as "not reported".

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `cmd/spend_ledger.go` | Durable per-worker, per-workflow rows | ✓ VERIFIED | Wired; `ProviderUSD` removed (comment at line 298 records the deletion); `JobName` present; status vocabulary shared with `dispatchStatusIcon` |
| `cmd/spend_writer.go` | Turns finished dispatches into rows | ✓ VERIFIED | Called from `codex_build_finalize.go` and `codex_continue_finalize.go`; proven by real `build-finalize` in the e2e test |
| `cmd/spend_cost_line.go` | The one closeout cost line | ✓ VERIFIED | Sole renderer; called from `ceremony_cmd.go` and `codex_workflow_cmds.go`; finalizers deliberately silent |
| `cmd/spend_cmd.go` | Read-only per-worker view | ✓ VERIFIED | Real run proved byte-identical data directory |
| `cmd/spend_session_capture.go` | Records the platform-delivered transcript path | ✓ VERIFIED | Called from `hook_cmds.go:57`; hook installed in `.claude/settings.json`; proved end to end with the real binary |
| `cmd/wrapper_usage_claude.go` | Deduplicated, bounded, path-contained transcript reader | ✓ VERIFIED | Shape validated against 400 real transcripts |
| `cmd/wrapper_usage_opencode.go` | OpenCode session-store reader | ✓ VERIFIED | Called via the resolver; covered by the anti-orphan guard |
| `cmd/wrapper_usage_resolve.go` | One resolver for both chat platforms | ✓ VERIFIED | Fails safe — leaves a figure out rather than guessing, and says so in plain English |
| `cmd/caste_model_reason.go` | Written reason per expensive role | ✓ VERIFIED | Rendered on the real card |
| `cmd/testdata/spend/claude/transcript-with-duplicates.jsonl` | Real capture that keeps its duplicates | ✓ VERIFIED | 17 usage lines, 9 distinct ids, 6 repeated |
| `196-BRANCH-DISPOSITION.md` | Written reason + tip commit per branch | ✓ VERIFIED | All three tips resolvable; locked by a test |
| `deferred-items.md` | Every knowingly-left item, actionable | ✗ INCOMPLETE | D1 and D2 only; WR-04/05/06 absent |
| `cmd/spend_ledger.go` (roll-up half) | — | ⚠️ ORPHANED | `spendPerWorkerAverageTokens`, `spendRollupByParent`, `spendParentRollup` have no production caller |

### Key Link Verification

| From | To | Via | Status |
|---|---|---|---|
| `cmd/hook_cmds.go` | `cmd/spend_session_capture.go` | `recordSpendSessionFromHook` on every PreToolUse event | ✓ WIRED (proved by running the hook) |
| `cmd/wrapper_usage_resolve.go` | `cmd/wrapper_usage_claude.go` | `parseClaudeTranscriptUsage` on the Claude path | ✓ WIRED |
| `cmd/wrapper_usage_resolve.go` | `cmd/wrapper_usage_opencode.go` | `openCodeSessionUsageForRunOrNone` on the OpenCode path | ✓ WIRED |
| `cmd/codex_build_finalize.go` | `cmd/spend_writer.go` | `writeSpendRowsForRun` before returning | ✓ WIRED |
| `cmd/codex_continue_finalize.go` | `cmd/spend_writer.go` | continue-keyed rows, separate file per workflow | ✓ WIRED |
| `cmd/ceremony_cmd.go` | `cmd/spend_cost_line.go` | `appendSpendCostLine` last on the ending screen | ✓ WIRED (production branch proved by B1) |
| `cmd/ceremony_team_checkin.go` | `cmd/caste_model_reason.go` | `casteModelReason` beside the resolved model | ✓ WIRED (proved by B3) |
| `.claude` + `.opencode` build/continue wrappers | `cmd/spend_cmd.go` | "If the user asks what it cost… run `aether spend`" | ✓ WIRED (4 surfaces; D-06 satisfied) |
| 6 wrapper surfaces | the transcript join | `TestDispatchDescriptionCarriesTheAccountingKey` asserts the dispatch INSTRUCTION LINE, not the file, and includes the flat installed mirrors (`~/.claude/commands/ant-build.md` equivalents) | ✓ WIRED |
| `writeSpendRowsForRun` | `spendRow.ParentName` | nothing | ✗ NOT WIRED — field never assigned in production |

### Data-Flow Trace (Level 4)

| Value on screen | Source | Real data | Status |
|---|---|---|---|
| Per-worker token figure | provider event (direct lane) or session transcript / OpenCode session store (wrapper lane) → `resolveWrapperWorkerUsage` → ledger row → `renderSpendCostLine` | Yes — B1 and B2 rendered the exact recorded 1,002,550 | ✓ FLOWING |
| Run total | `computeSpendTotals().MeasuredTokens`, never re-derived in the renderer | Yes | ✓ FLOWING |
| Unreported worker's cell | `spendRowReportedUsage` → dash sentinel, shared with the detail view so the two can never disagree | Yes (no number invented) | ✓ FLOWING |
| Model + reason on the team card | `resolveCasteModel` → `.claude/agents/ant/*.md` frontmatter, checked in both directions | Yes on Claude; **not true on OpenCode** (WR-05) | ⚠️ STATIC on OpenCode |

### Requirements Coverage

| Requirement | Description | Status | Evidence |
|---|---|---|---|
| COST-01 | One plain-English cost line, no number for an unreported worker, total counts only the measured, dollars never the headline, no price table | ✓ SATISFIED | Criterion 1; M1, M2 |
| COST-02 | Wrapper-path worker results report token usage | ✓ SATISFIED | Criterion 2; M4; real-transcript fidelity audit; B4 |
| COST-03 | `aether spend` per-worker, mutating nothing, no figure from a character budget | ✓ SATISFIED | Criterion 3; B2; ratchet scope audit |
| COST-04 | Team card shows model and reason; no routine role carries `inherit`; every expensive role has a recorded reason | ✓ SATISFIED (Claude) | Criterion 4; B3; M5, M6. Caveat: the claim is not true on OpenCode (WR-05) |
| COST-05 | The three branches reviewed, salvaged or discarded with a written reason, then deleted | ✓ SATISFIED | Criterion 5; B5 |

No orphaned requirements: `grep -E "Phase 196" .planning/REQUIREMENTS.md` maps exactly
COST-01..COST-05, and all five are claimed by plans in this phase.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| — | — | `TBD` / `FIXME` / `XXX` / `TODO` / `HACK` / `PLACEHOLDER` across all eleven files this phase touched | — | None found. Scan clean. |
| `cmd/spend_ledger.go` | 416, 431, 445 | Dead code with a passing test over a shape production never emits | ⚠️ Warning | WR-04 — see Gaps |
| `cmd/ceremony_closeout_spend_test.go` | 69-79 | Fixture packet carries no manifest key, so the tests drive only the fallback branch | ⚠️ Warning | WR-06 — see Warnings |
| `cmd/caste_model_test.go` | 147 | `agentModelLines` reads `.claude/` only | ⚠️ Warning | WR-05 — see Warnings |
| `196-BRANCH-DISPOSITION.md` | "can still be recovered by `git show`… long after the name is gone" | Documentation claim stronger than reality | ℹ️ Info | All three tip commits are **dangling** — reachable from no ref, so `git gc` will eventually prune them. The claim is true today and will silently stop being true. |

## Gap Closure — re-verified 2026-08-28

Both gaps from the initial pass are closed. Every claim below was checked by
running something, not by reading the commits. All plants were applied to
production source on disk and restored byte-identically (SHA-256
`f1f7147d9ab1c4d833c474c001785e38e1abe5f8360d3587acb6f4ee7d6cd4f6` for
`cmd/spend_ledger.go` before and after every plant; `git diff --stat` empty).

**Commits:** `6d0df805` (deletions), `afc9963b` (rebuilt guard), `0a238274`
(deferred entries + disposition correction).

### 1. The dead code is gone, and the guard is honest

`spendPerWorkerAverageTokens`, `spendRollupByParent`, `spendParentRollup`,
`spendRow.ParentName` and `spendRow.ToolCount` are deleted. The only surviving
references anywhere are explanatory comments. `spendRow` now carries a comment
recording that the absence of both fields is deliberate and that neither should
return without a writer in the same change.

The rebuilt guard enumerates nothing: file set derived from disk by name prefix
(`spend_`, `wrapper_usage_`, plus `caste_model_reason.go`), symbols and struct
fields derived from the syntax tree, "called" computed as a reachability
fixpoint over the production package. **I verified that fixpoint claim rather
than accepting it**, by planting on disk:

| Plant | Expected | Guard output | Verdict |
|---|---|---|---|
| `spendVerifierPlantOuterOrphan` — uncalled function | named | named | ✓ |
| `spendVerifierPlantInnerOrphan` — called **only by another orphan** (the exact `spendParentRollup` shape that escaped the old guard) | named | named | ✓ |
| `spendVerifierPlantRecord` — uncalled type | named | named | ✓ |
| `spendVerifierPlantRecord.PlantedFieldUnread` — never read | named | named | ✓ |
| `spendVerifierPlantRecord.PlantedFieldRead` — read by the plant | **not** named | not named | ✓ no false positive |
| `spendVerifierCycleA` / `spendVerifierCycleB` — mutually recursive orphan pair | both named | both named | ✓ a cycle cannot self-certify |
| Re-adding the exact deleted `spendRow.ParentName` and `spendRow.ToolCount` | both named | both named by name | ✓ the specific regression cannot return |

The last row matters most: `cmd/ceremony_emitter.go` assigns a `ToolCount` of
its own, and under the old package-wide name search that unrelated field vouched
for the dead one. It no longer does.

**False positives on live code: none.** The unplanted tree reports zero uncalled
symbols and zero unused fields, and the guard's own planted subtest asserts it
does not name a field that IS read.

**The nine kept names cannot go stale into a false pass.** They are now asserted
as *declared*, not as an inventory, and each failure names what stops working.
A missing name produces a loud failure, never a silent pass. They are backed by
three derivation floors (`t.Fatalf` on fewer than 8 files, 60 symbols, 40
fields); the tree currently derives 9 files, and 110 fields. The two halves
cover each other: renaming a phase file out of the prefix set would keep the file
count at 8 but drop its symbols from the inventory, which the nine-name floor
then catches by name.

**Is the documented limitation stated fairly? Partly — it is incomplete.** The
guard documents that field use is matched by selector within this phase's files
only, and warns this could wrongly report a field read exclusively from outside
the subsystem. That direction is real but harmless: it fails loudly and gets
fixed. **It omits the opposite direction, which fails silently.** Because the
selector match is on the BARE field name across the phase's own files, a dead
field whose name collides with a live field on another phase struct escapes
entirely. Measured:

```
# added to spendRow — never written, never read anywhere:
Notes []string `json:"notes,omitempty"`
$ go test ./cmd/ -run TestNothingThisPhaseAddedIsUncalled
ok      github.com/calcosmic/Aether/cmd 0.862s
```

It passes because `spendWriteOutcome.Notes` is selected in `spend_writer.go`.
This is the same species as the package-scope `ToolCount` hole the rebuild
closed, narrowed from the whole `cmd` package to nine files — a large
improvement, but the residual hole is in the direction that stays quiet, which
is the direction that bit this phase. Recorded as a warning, not a gap: it does
not affect any success criterion, and the symbol half of the guard is airtight.

**No current field is at risk of the documented false positive.** Of the 110
derived fields, 59 are justified by both a composite-literal key and a
phase-local selector, 3 by a literal key alone, and 48 by a phase-local selector
alone. None is justified only by an external selector, so none would be wrongly
reported today.

### 2. The deletions removed no coverage that mattered

Two tests were deleted whole — `TestLedgerRefusesDerivedMetricOverEstimates` and
`TestLedgerParentRollupCountsEachWorkerOnce`. Reading every removed assertion in
`git show 6d0df805 -- cmd/spend_ledger_test.go`, each one names a deleted
function, a deleted field, or the roll-up type. Nothing live lost a proof.

The one assertion with independent value — *"the continue-arriving worker must
be counted, not dropped"* — was preserved, moved onto
`computeSpendTotals().MeasuredRows`. That is a better home than the one it left:
`MeasuredRows` is the figure the owner-facing cost line actually prints
("the total counts only the N whose tools reported a figure"), whereas the
roll-up it used to sit on was never rendered anywhere. Thirteen tests remain in
`cmd/spend_ledger_test.go`, all green, including the persistence, grand-total,
hand-summed-raw-column and no-currency walks.

### 3. D3, D4 and D5 are recorded well enough to act on

All three carry the finding, the exact file and line, why it was not done here,
and the concrete fix — not a note that something was left.

- **D3 (WR-04)** correctly records the substance as *closed* rather than
  deferred, and separates out the one genuinely open question (should a row
  carry which worker spawned it — a feature decision with a display surface).
  Its claim that WR-04's `spendLedger.RunID` complaint is stale checks out:
  `RunID` is written at `spend_writer.go:268` and read at
  `spend_ledger.go:229, 257, 341, 363` to tell a re-finalize from a retry.
- **D4 (WR-05)** states the OpenCode divergence accurately, names both honest
  fixes rather than presuming one, and specifies the test extension that would
  stop the platforms silently disagreeing again.
- **D5 (WR-06) is recorded at the corrected severity.** It says plainly
  *"untested, not broken"*, records that verification ran the production branch
  by hand with the real binary and it correctly rendered phase 1 while
  `current_phase` was 2, and carries the measured fact: **deleting the
  `completion_phase` branch leaves the whole spend / cost / closeout /
  end-to-end set green** (citing mutation M3 above). It also keeps the caveat
  that the fallback IS wrong when reached, with the reasons production does not
  normally reach it, and names the mutation that would prove the fix.

### 4. The disposition record is now true

Every claim in the correction was checked:

```
be1e160b  resolve=commit  refs=0  reflog=0  ancestor-of-HEAD=no
ecd98b5c  resolve=commit  refs=0  reflog=0  ancestor-of-HEAD=no
1cbf3615  resolve=commit  refs=0  reflog=0  ancestor-of-HEAD=no
gc.pruneExpire=(unset -> default 2.weeks.ago)
gc.reflogExpireUnreachable=(unset -> default 30 days)
gc.auto=(unset -> default 6700, auto gc enabled)
git tag -l 'archive/*' -> 0
```

All three tips resolve; none is reachable from any ref; `git reflog --all`
contains zero hits for any of them; the repository runs git's defaults, so
`git gc` — which git triggers on its own during ordinary commands — will prune
them. The document says exactly this, gives the three `git tag` commands that
would make them durable, and records that not creating those tags is a knowing
choice rather than an oversight. The absence of the tags is itself verified
(`git tag -l 'archive/*'` returns none), so the record matches the repository.


## Warnings — the three deferred findings, verified as still live

These were agreed deferred and are NOT counted as criterion failures. The
findings below were absent from `deferred-items.md` at initial verification;
commit `0a238274` recorded them as D3, D4 and D5, and WR-04's substance was then
closed outright by `6d0df805`. The text is kept as the measured record of what
each finding was.

**WR-04 — three ledger features are orphans, one of which cannot work if called.**
Confirmed live. `spendPerWorkerAverageTokens` and `spendRollupByParent` have no
production caller; `spendRow.ParentName` is never assigned, so the roll-up would
return one `(unattributed)` bucket. Its test seeds the field by hand. This is
gap 1 above.

**WR-05 — the card states a model fact that is false on OpenCode.** Confirmed live.
`grep -rln "^model:" .opencode/agents/*.md` returns nothing across all 27 files,
while all 27 `.claude/agents/ant/*.md` carry one. `agentModelLines`
(`cmd/caste_model_test.go:147`) reads `../.claude/agents/ant` only, so nothing detects
the divergence. `.opencode/commands/ant/build.md` renders the same card from the same
runtime renderer. CLAUDE.md names OpenCode a primary platform.

**WR-06 — the closeout cost-line tests never exercise the branch real runs take.**
Confirmed live and measured (mutation M3): deleting the `completion_phase` branch
leaves the entire spend / cost / closeout / e2e test set green.
`costLineCompletionFileForTest` still writes a bare `"phase": 1` with no manifest key.

**Mitigating evidence I gathered myself, which the review could not:** I ran the
production branch by hand with the real binary (spot-check B1) against a packet
carrying a real `dispatch_manifest`, with the colony's `current_phase` deliberately
advanced to 2. It correctly rendered **phase 1**. So the branch is *untested*, not
broken. I also reproduced the fallback's wrong behaviour: a manifest-less packet
after an advance rendered *"No token use was recorded for this phase"* while phase 1's
real rows sat on disk. Production does not normally reach that fallback — `build-finalize`
hard-errors on a packet without `dispatch_manifest` (`cmd/codex_build_finalize.go:379`),
and the continue closeout is only used on the opt-in heavy path, which always carries
`continue_manifest`. The default continue path prints its own ending screen, and that
lane IS driven end to end by `TestNoLaneRendersTwoCostLines`.

## Gaps Summary

**Both gaps are closed, and the phase is verifiably complete.**

The five roadmap success criteria were proved in the initial pass by running real
commands and then breaking the production code to watch named tests fail; nothing
in the closure commits touches those proofs, and the affected test set is green.
They are not re-scored here.

**Gap 1 is closed properly, not papered over.** The five dead symbols and fields
are actually deleted, and the guard that failed to catch them has been rebuilt to
derive its own inventory instead of trusting a hand-written list. I did not take
the reachability claim on report: I planted orphans on disk in six shapes,
including the two the old guard structurally could not see — an orphan referenced
only by another orphan, and a mutually recursive orphan pair — and the guard named
every one while leaving honest code alone. Re-adding the exact deleted fields is
caught by name.

**Gap 2 is closed.** D3, D4 and D5 are recorded with the finding, the file and
line, the reason it was left, and the concrete fix. D5 in particular is recorded
at the corrected severity — untested rather than broken — and carries the measured
fact that deleting the branch leaves the whole spend/cost/closeout set green,
which is the honest statement of what is actually unproved.

**The disposition correction is accurate.** All four of its claims verify, and it
now records a knowing choice ("accept that these three commits will be pruned")
instead of a durability guarantee it could not keep.

**Two things remain, neither a gap, both worth knowing.**

First, the guard's field rule still has a silent hole. It matches selectors by
bare field name within this phase's own files, so a dead field whose name
collides with a live field on another phase struct escapes — measured, with a
never-written `Notes` on `spendRow` passing cleanly. The documented limitation
names only the direction that fails loudly and omits this one. The hole is far
narrower than the package-wide one it replaced, and the symbol half of the guard
is airtight, so this is a warning rather than a gap. But it should be written
into the limitation note as stated, because a limitation that lists only the safe
half of a trade reads as more reassuring than it is.

Second, `cmd/spend_ledger.go:79` names `TestEveryFieldThisPhaseAddedIsUsed` — a
test that does not exist. The check it points at is real and lives inside
`TestNothingThisPhaseAddedIsUncalled`. A comment naming a nonexistent test is a
documentation claim that cannot be run, which is the thing CLAUDE.md's Definition
of Done asks be testable or removed. One-word fix.

---

_Verified: 2026-08-28T17:10:00Z — re-verified after gap closure 2026-08-28T18:05:00Z_
_Verifier: Claude (gsd-verifier)_
