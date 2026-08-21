---
phase: 173-delegation-guard
verified: 2026-08-13T17:08:32Z
status: human_needed
score: 6/6 ROADMAP success criteria verified (up from 5/6); the one demonstrated exploit and a second exploit found during gap-closure planning are both independently reproduced-and-refused in this re-verification
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: "5/6 ROADMAP success criteria fully verified; 1 falsified with a demonstrated exploit"
  gaps_closed:
    - "Every delegation guard fails closed when its inputs are unreadable (ROADMAP Phase 173 success criterion 4, and the phase's own goal statement: 'every guard fails closed') — the corrupted-ledger route this verifier demonstrated is now refused, byte-identically re-reproduced in this session"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "Run a real /ant-build (or /ant-continue) that hits either the depth cap or the whole-run budget ceiling, and read the output as the owner would — not the raw JSON, the narration shown in the terminal."
    expected: "The refusal names the helper, its would-be parent, and the reason, in plain English, inside the ceremony narration."
    why_human: "173-RESIDUE.md residue 8 states this explicitly as unproven: no plan in this phase, including the three gap-closure plans, touches build.md or continue.md. The Go-level Detail/error string is proven correct and present in the CLI's own JSON/error output (confirmed again in this re-verification's manual reproductions), but whether it reaches the wrapper-level narration a non-technical operator actually reads has never been observed. Carried forward unchanged from the initial verification — no plan since has addressed it."
---

# Phase 173: Delegation Guard Verification Report

**Phase Goal:** Recursive delegation is bounded by the runtime at the one chokepoint an LLM
cannot route around — the spawn-recording call. Depth is derived from the parent's recorded
entry rather than asserted by the caller, a whole-tree budget bounds what depth alone cannot,
every guard fails closed, and the operator can watch the tree while it grows. Nothing gains the
ability to delegate in this phase; enforcement lands before capability because parent/depth
linkage is recorded at spawn time and cannot be retrofitted to past runs.

**Verified:** 2026-08-13T17:08:32Z
**Status:** human_needed
**Re-verification:** Yes — after gap closure (plans 173-11, 173-12, 173-13)

## Summary For The Owner (plain English)

This morning I found a real hole in this phase's work: an AI helper could reset the "you've used
all 20 helpers you're allowed this run" limit back to zero just by overwriting one internal record
file with a line of garbage text — no special permissions needed, just an ordinary command. I
proved it by doing it myself.

The team fixed it, and while planning the fix they found a second, even easier way to cause the
same problem — deleting a *different* internal file (the one that says which batch of work is
currently running), with no tampering of the first file at all. They fixed that too.

I did not take their word for it. I rebuilt the program from the fixed code and, in a disposable
test folder well away from this repository, personally repeated my exact original attack — filled
the 20-helper limit, watched the 21st helper get correctly refused, then overwrote the record file
with garbage, and tried the same request again. This time it was refused too, and it named the
exact reason ("the spawn ledger is present but its content is not a valid spawn ledger"). The
record file was left untouched — the broken evidence wasn't quietly erased and replaced with a
clean one, which is what happened before. I then built a fresh test colony, gave it a valid,
completely full 20-helper record, and tried the second attack — deleting the "which batch is
running" file, and separately replacing it with a folder instead of a file. Both were refused too.
I also confirmed the fix didn't overcorrect: a brand-new colony with no record files at all can
still spawn its first helper normally, so nobody is locked out by mistake.

Every one of the program's own automated checks for this fix passes, and so does the complete test
suite for the whole codebase (over 5,000 tests, all packages, nothing broken by this fix). The
team also went back and fixed five comments in the code that used to say "this kind of corruption
can never happen" — those comments were true when written and are now false, so leaving them would
have misled the next person to read the code.

One thing from my first check is still true and still needs a human to look at it, because nothing
about it changed in this round of fixes: when a helper actually gets refused during a real project
build, does the plain-English explanation of why show up in the normal on-screen narration you'd
read, or only in a technical error message you'd have to go looking for? Nobody has watched that
happen yet. That's the only reason this report isn't an unqualified "done."

## Goal Achievement

### Observable Truths (ROADMAP Phase 173 Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `spawn-can-spawn --depth 99` denies; a spawn one level past the cap exits non-zero and writes no spawn-tree entry | VERIFIED (regression check) | Unaffected by the gap-closure plans (no file touched by plans 11-13 sits on this code path). `go test ./... -count=1 -timeout 900s` passes all 20 packages, including `TestSpawnCanSpawnDeniesPastDepthCap`/`TestSpawnLogRefusesPastCapAndWritesNoEntry`. |
| 2 | `spawn-tree-depth` reports depth 2 for a three-level tree built entirely with `--depth 0` | VERIFIED (regression check) | Unaffected by the gap-closure plans. Full suite passes, including `TestSpawnTreeDepthReportsTwoForAThreeLevelTree`/`TestSpawnLogDerivesDepthFromRecordedParent`/`TestWorkersMdStatesOneDepthConvention`. |
| 3 | With the wave cap at 8 and the tree budget at 20, a run never exceeding 8 per wave is refused at helper 21; wave-1 consumption is not restored in wave 2 — **and now also: neither corrupting the ledger nor deleting the run record can reset the count** | VERIFIED (no longer asterisked) | `go test ./cmd -run 'TestSpawnTreeBudgetRefusesTheTwentyFirstHelper|TestSpawnTreeBudgetIsNotRestoredBetweenWaves|TestSpawnTreeBudgetAndWaveCapAreSeparateQuantities|TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget|TestErasingTheRunRecordDoesNotResetTheWholeRunBudget|TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn'` — all pass. The undercutting caveat from the initial verification report no longer applies: the whole-run budget genuinely bounds the tree now, not just under non-adversarial wave shapes. |
| 4 | With `COLONY_STATE.json` and the spawn tree made unreadable, every delegation guard denies and exits non-zero, and the hook denies an unresolvable requester | **VERIFIED — the falsified criterion, now closed by building** | See "The Decisive Re-Reproduction" below. Both the originally-demonstrated exploit (ledger corruption) and the second exploit found during gap-closure planning (run-record erasure/obstruction) are independently reproduced by me in this session and confirmed refused. `go test ./cmd -run 'TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs|TestDelegationGuardTableCoversEveryGuardCommand|TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget|TestErasingTheRunRecordDoesNotResetTheWholeRunBudget|TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn'` all pass; all 16 subtests of the cross-guard table pass, including the 3 new `ledger-corrupt` and 1 new `run-state-obstructed` subtests. |
| 5 | A spawn whose (caste, normalised task) already appears in its own ancestor chain is refused, naming the ancestor | VERIFIED (regression check) | Unaffected by the gap-closure plans' production logic (only a fixture comment in `cmd/spawn_ancestor_test.go` changed, no assertion). Full suite passes, including `TestSpawnCanSpawnDeniesAncestorCycle`/`TestSpawnAncestorCheckFailsClosedOnUnreadableTree`. |
| 6 | `spawn-tree-active` renders the tree indented by depth with parent attribution while a run is in progress, and mutates nothing | VERIFIED (regression check) | Unaffected by the gap-closure plans. Full suite passes, including `TestSpawnTreeActiveRendersIndentedByDepthMidRun`/`TestSpawnTreeActiveMutatesNothing`. |

**Score:** 6/6 criteria verified. Criterion 4, the one falsified in the initial verification, is now
closed — not by narrowing its wording, but by building a fix and independently re-attacking it.

### The Decisive Re-Reproduction

Performed from a disposable scratch colony store (`COLONY_DATA_DIR` pointed at a temp directory
well outside this repository), against the actual binary rebuilt from the current, committed source
tree (`go build ./cmd/aether`).

**Step 1 — the fresh-colony invariant still holds (the fix did not overcorrect):**

```
$ COLONY_DATA_DIR=<scratch>/data aether spawn-log --parent Queen --caste builder --name W1 --task "task 1" --depth 0
{"ok":true,"result":{"budget_consumed":1,"budget_max":20,...,"recorded":true,...}}
exit: 0
```
A colony with no ledger file and no run-state file at all can still spawn its first helper.

**Step 2 — my original exploit, re-run against the fixed binary:**

```
# Legitimately exhaust the 20-helper whole-run budget
$ for i in 1..20: aether spawn-log --parent Queen --caste builder --name "W$i" --task "task $i" --depth 0

# Confirm the 21st is correctly refused (control)
$ aether spawn-log --parent Queen --caste builder --name W21 --task "task 21" --depth 0
{"ok":false,"error":"whole-run helper budget exhausted: 20 of 20 helpers already spawned
(counted across the entire ledger because no run is recorded); Queen may not spawn another","code":1}
exit: 1

# The exact attack that worked this morning
$ echo "garbage not pipe format" > <scratch>/data/spawn-tree.txt

# Retry the identical, already-refused request
$ aether spawn-log --parent Queen --caste builder --name W22-BYPASS --task "task 22" --depth 0
{"ok":false,"error":"whole-run helper budget unverifiable (verify spawn-tree.txt: spawn_tree:
spawn-tree.txt: spawn ledger is present but its content is not a valid spawn ledger: line 1 has
1 fields, expected 7 (spawn) or 4 (completion)): refusing to spawn","code":1}
exit: 1

# The tampering evidence was NOT erased -- the file is still exactly the garbage I wrote
$ cat <scratch>/data/spawn-tree.txt
garbage not pipe format
```

This morning's identical sequence returned `{"ok":true,"result":{"budget_consumed":1,...}}` and
`exit: 0` at this exact step. It now refuses, names the corruption specifically, and leaves the
tampered file untouched as evidence.

**Step 3 — the second exploit (found during gap-closure planning, not part of my original
reproduction), re-run against the fixed binary, both variants:**

```
# A valid, genuinely active run, fully exhausted at 20 helpers (started_at set in the past so
# the entries fall inside the run's own window, confirmed by the control refusal below)
$ for i in 1..20: aether spawn-log --parent Queen --caste builder --name "V$i" --task "task $i" --depth 0
$ aether spawn-log --parent Queen --caste builder --name V21 --task "task 21" --depth 0
{"ok":false,"error":"whole-run helper budget exhausted: 20 of 20 helpers already spawned in
this run; Queen may not spawn another","code":1}
exit: 1   # control: budget genuinely full, counted via the run window this time

# Variant A: delete the run-record file entirely, leaving the valid, full ledger untouched
$ rm <scratch>/data/spawn-runs.json
$ aether spawn-log --parent Queen --caste builder --name V22-BYPASS --task "task 22" --depth 0
{"ok":false,"error":"whole-run helper budget exhausted: 20 of 20 helpers already spawned
(counted across the entire ledger because no run is recorded); Queen may not spawn another","code":1}
exit: 1

# Variant B (fresh scratch colony, same setup): replace the run-record file with a directory
$ rm <scratch2>/data/spawn-runs.json && mkdir <scratch2>/data/spawn-runs.json
$ aether spawn-log --parent Queen --caste builder --name D21-BYPASS --task "task 21" --depth 0
{"ok":false,"error":"whole-run helper budget unverifiable (resolve current run: spawn_tree: read
\"spawn-runs.json\": ... is a directory): refusing to spawn","code":1}
exit: 1

# The harder combined variant: corrupt the ledger too, in the same already-run-record-deleted colony
$ echo "more garbage" > <scratch>/data/spawn-tree.txt
$ aether spawn-log --parent Queen --caste builder --name V23-BYPASS --task "task 23" --depth 0
{"ok":false,"error":"whole-run helper budget unverifiable (verify spawn-tree.txt: ... line 1 has
1 fields, expected 7 (spawn) or 4 (completion)): refusing to spawn","code":1}
exit: 1
```

Both routes — the one I demonstrated this morning, and the second one the team found while planning
the fix — are refused, independently, by me, against the actual built binary, in a clean environment
with no pre-existing state. All four command outputs above were captured directly in this
verification session, not copied from a plan summary.

**Automated coverage matches the manual reproduction exactly:**
`go test ./pkg/agent -run 'TestSpawnTreeParseTreatsAnAbsentLedgerAsEmptyButACorruptOneAsAnError|TestSpawnTreeParseAcceptsEveryShapeTheWriterProduces|TestSpawnTreeRefusesToRewriteACorruptLedger|TestSpawnRunStateTellsAnAbsentRunFileApartFromAnUnreadableOne'`
and
`go test ./cmd -run 'TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget|TestErasingTheRunRecordDoesNotResetTheWholeRunBudget|TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn|TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs|TestDelegationGuardTableCoversEveryGuardCommand'`
all pass (transcripts confirmed in this session, not merely re-read from a SUMMARY.md).

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `pkg/agent/spawn_tree.go` | Three-way parse contract (absent/valid/corrupt), write-closure abort on corruption | VERIFIED | `ErrSpawnTreeCorrupt` sentinel present and wrapped with `%w`; `parseFile`, `loadRunStateLocked`, `RecordSpawn`, `updateStatus` all confirmed edited and tested; four new package tests pass |
| `cmd/spawn_budget.go` | Ledger-integrity check ahead of the no-run-yet shortcut; whole-ledger counting when no run resolves | VERIFIED | `st.Parse()` called before `CurrentRun()`; non-empty ledger with no run counts the whole ledger instead of reporting `Consumed: 0`; three new tests pass and were independently manually reproduced above |
| `cmd/spawn_failclosed_test.go` | Cross-guard proof extended with `ledger-corrupt` (3 guards) and `run-state-obstructed` (1 guard) axes; five stale "impossible" comments corrected | VERIFIED | All 16 subtests of `TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs` pass, including the 4 new ones by name; repo-wide grep for the six stale phrasings returns nothing |
| `cmd/internal_cmds.go` | Comment corrected to stop claiming `Parse()` cannot distinguish absent from unreadable | VERIFIED | Confirmed comment-only change; the `os.Stat` guard logic itself is unchanged (still a valid, independent defense-in-depth check) |
| `.github/workflows/ci.yml` | All three new plan-12 budget-guard tests registered in the wiring ratchet's `-run` filter | VERIFIED | `grep` confirms all three test names present; `TestWiringGateStepRunsEveryWiringTest` passes, proving the filter matches every guard test with no stale alternatives |
| `.planning/phases/173-delegation-guard/173-RESIDUE.md` | Rewritten to record both exploit routes, their closure, and newly-found bounds (10-13) | VERIFIED | Read in full; residue 7's table gained the 4 new (guard, axis) rows; the "to be judged against evidence at verification time" hedge is gone, replaced with a dated, sequenced account of what verification found, what planning additionally found, and what closed both; four new numbered residues (10-13) are present, each with a bounded claim, a wrong inference, and a "what would close it" line, matching what actually exists in the code (`recordCodexBuildDispatches` confirmed by direct code read to call `RecordSpawn` without consulting `spawnCanSpawnDecision`) |
| `.aether/workers.md` | States one depth convention | PARTIAL — unchanged, carried forward | The "SPAWN CAPABILITY" child-prompt template block (line 366-369) still tells a depth-2 helper it MAY spawn and cites a "Depth 3" that the recorded convention says cannot exist. Not touched by any gap-closure plan (out of scope); runtime still denies the attempt regardless (fail-closed holds), so this is a documentation warning, not a guard failure |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `pkg/agent/spawn_tree.go Parse()` | `cmd/spawn_budget.go spawnTreeBudgetState` | integrity check taken before the no-run-yet early return | WIRED | Confirmed by code read (`st.Parse()` precedes `CurrentRun()`), by the passing test suite, and by my own manual reproduction above — corrupting the ledger now denies before the budget check's shortcut can be taken |
| `pkg/agent/spawn_tree.go CurrentRun`/`loadRunStateLocked` | `cmd/spawn_budget.go` no-run-yet branch | whole-ledger counting when the run window cannot be resolved | WIRED | Confirmed: deleting or obstructing `spawn-runs.json` against a full ledger now counts the whole ledger rather than reporting zero, in both the automated test and my manual reproduction |
| `cmd/spawn_budget_test.go` (3 new tests) | `.github/workflows/ci.yml -run` filter | `TestWiringGateStepRunsEveryWiringTest` | WIRED | All three new test names present in the filter; the ratchet test passes |
| `cmd/spawn_failclosed_test.go delegationGuardFaultTable` (4 new axes) | `TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs` | table-driven subtests | WIRED | Verbose test output shows all 4 new subtests (`ledger-corrupt` ×3, `run-state-obstructed` ×1) running and passing |

### Data-Flow Trace (Level 4)

Same conventional-sense caveat as the initial verification (this is guard logic, not a UI rendering
pipeline) — the trace here follows the corrupted byte sequence from disk through the fixed chain:
`Parse()` (now returns a non-nil error wrapping `ErrSpawnTreeCorrupt`) → `spawnTreeBudgetState()`
(now calls `Parse()` before `CurrentRun()`'s shortcut and propagates the error) →
`spawnTreeBudgetReason()` (already turned any non-nil state error into a deny sentence) →
`spawnCanSpawnDecision` → `spawnLogCmd.RunE` → **denied before `RecordSpawn` is ever reached**. This
is the corrected version of the exact chain the initial verification traced as silently absorbing
the corruption at the first step; it now denies at the first step instead, and my manual
reproduction confirms the deny sentence and the untouched file bytes at the end of that chain.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Fresh colony (no ledger, no run file) can still spawn | `spawn-log` with empty scratch store | `recorded:true`, exit 0 | PASS |
| Legitimate 21st helper refused before any tampering (control) | `spawn-log` after 20 successful spawns | exit 1, names the budget | PASS |
| **Ledger corruption no longer resets the budget** | corrupt `spawn-tree.txt` with one garbage line, retry the refused spawn | **exit 1, names the corruption, file bytes unchanged** | **PASS — this morning's exploit is closed** |
| **Run-record deletion no longer resets the budget** | valid full ledger, `rm spawn-runs.json`, retry | **exit 1, whole ledger counted instead of a free budget** | **PASS — the second exploit is closed** |
| **Run-record directory obstruction denies outright** | valid full ledger, directory at `spawn-runs.json`'s path, retry | **exit 1, names the budget as unverifiable** | **PASS** |
| **Combined harder variant (corrupt ledger + deleted run record)** | both faults present simultaneously, retry | **exit 1, corruption named, bytes unchanged** | **PASS** |
| Full repository test suite | `go test ./... -count=1 -timeout 900s` | all 20 packages pass, exit 0 | PASS |
| CI wiring ratchet includes the 3 new budget tests | `go test ./cmd -run TestWiringGateStepRunsEveryWiringTest` | passes | PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes are declared by this phase's plans or exist under that
convention for this phase's subject area (unchanged from the initial verification). Skipped —
verification method is Go's test runner plus manual CLI reproduction, both exercised above.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| SPAWN-01 | 173-04 | Refuse at a configured depth | SATISFIED | Unaffected by gap closure; regression-verified |
| SPAWN-02 | 173-02 | A spawned child records its true depth | SATISFIED | Unaffected by gap closure; regression-verified |
| SPAWN-03 | 173-05, 173-11, 173-12 | Whole-tree budget bounds what depth alone cannot | **SATISFIED (upgraded from BLOCKED)** | Both reset routes are independently reproduced-and-refused in this session; the budget now genuinely bounds the tree against tampering, not just under normal wave shapes |
| SPAWN-04 | 173-01, 173-03, 173-07, 173-10, 173-13 | PreToolUse hook denies before platform acts; fails closed on unresolved requester; every guard proven to fail closed on its own unreadable inputs | SATISFIED | The hook itself was already fully verified and untouched by this gap closure; the cross-guard table now also proves the recorder, the advisory checker, and the swarm checker each independently deny a corrupted or obstructed ledger/run-record, closing the exact defense-in-depth gap the initial verification flagged |
| SPAWN-05 | 173-06 | Ancestor-chain repetition detected and refused | SATISFIED | Unaffected by gap closure (only a fixture comment changed); regression-verified |
| SPAWN-06 | 173-02, 173-10 | Decision: what depth 0 means | SATISFIED (decision recorded); PARTIAL on its documentation consequence, unchanged | `.aether/workers.md`'s child-prompt template block (WR-01) still contradicts the recorded convention; not addressed by any gap-closure plan; fail-closed enforcement is unaffected |
| SPAWN-07 | 173-08 | Operator watches the tree grow | SATISFIED | Unaffected by gap closure; regression-verified |
| SPAWN-08 | 173-09 | Abandoned child reaped, budget released, operator command | SATISFIED | Unaffected by gap closure; regression-verified |

No orphaned requirements: REQUIREMENTS.md lists SPAWN-01 through SPAWN-08 for Phase 173, and every
one is claimed by at least one plan's frontmatter `requirements:` field across all 13 plans
(cross-referenced above; plans 11-13 additionally claim SPAWN-03 and SPAWN-04 for the gap-closure
work).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `pkg/agent/spawn_tree.go` | 328-339 (pre-fix) | Silent error-swallowing (`parseFile` always returned nil error) | **RESOLVED** | This was the root cause of the whole-run budget bypass; now returns a distinguishable error. No longer a blocker. |
| `.aether/workers.md` | 364-369 | Stale/contradictory prose (WR-01, still open) | ⚠️ Warning, carried forward unchanged | Child-prompt template still tells a depth-2 helper it MAY spawn and cites a "Depth 3" the recorded convention says cannot exist. Not in scope for any of the three gap-closure plans. Runtime still denies the attempt (fail-closed holds). |
| `.aether/workers.md` | 304 | Stale documented return shape (WR-02, still open) | ⚠️ Warning, carried forward unchanged | Not addressed by gap closure; out of scope. |
| `cmd/spawn_budget.go` | ~99-117 | Inspection command mutates state (WR-03, still open) | ⚠️ Warning, carried forward unchanged | `spawn-can-spawn` (no `--enforce`) still writes to `midden.json` at the budget ceiling on an advisory call. Not touched by any gap-closure plan; the plan explicitly stated it would not call `spawnTreeBudgetCeilingToMidden` from the new error branch, but did not revisit this pre-existing issue. |
| `cmd/hook_cmds.go` | 177-201 | Two-sided heuristic weakness (WR-04, still open) | ⚠️ Warning, carried forward unchanged | Named, bounded, disclosed by the phase's own team; not in scope for gap closure. |
| `cmd/spawn.go` | 94-121 | Check-then-act race across processes (WR-05, still open) | ⚠️ Warning, carried forward unchanged | Explicitly named as still-open in 173-RESIDUE.md's new residue 13. Not attempted by any of the three gap-closure plans. |
| `cmd/spawn_ancestor.go` | 69-77 | Silent truncation on unresolvable mid-chain ancestor (WR-07, still open) | ⚠️ Warning, carried forward unchanged | Not in scope for gap closure. |
| `.aether/commands/patrol.yaml` vs 3 wrapper files | — | Spec/wrapper drift (WR-06, still open) | ⚠️ Warning, carried forward unchanged | Not in scope for gap closure; confirmed still present by a fresh grep in this session. |
| `cmd/codex_build.go` | 2700-2708 | `recordCodexBuildDispatches` calls `RecordSpawn` directly, bypassing `spawnCanSpawnDecision`/the budget check entirely | ℹ️ Info — newly named as residue 11, accurately disclosed | Confirmed by direct code read in this session: the function calls `spawnTree.RecordSpawn` with no call to the decision chokepoint. This is pre-existing behavior (not introduced or worsened by the gap closure) and is now honestly named in 173-RESIDUE.md rather than left undocumented. Spawns via this path are still counted by anything reading the ledger afterward, but are never asked permission first. |

No unreferenced `TBD`/`FIXME`/`XXX` markers found in any file touched by the gap-closure plans
(`pkg/agent/spawn_tree.go`, `pkg/agent/spawn_tree_test.go`, `cmd/spawn_budget.go`,
`cmd/spawn_budget_test.go`, `cmd/spawn_failclosed_test.go`, `cmd/internal_cmds.go`,
`cmd/spawn_ancestor_test.go`) — confirmed by a direct grep in this session.

### Human Verification Required

**1. Refusal reason visible to the non-technical operator inside the real ceremony (carried forward, unchanged)**

**Test:** Run a real `/ant-build` (or `/ant-continue`) that hits either the depth cap or the
whole-run budget ceiling, and read the output as the owner would — not the raw JSON, the
narration shown in the terminal.
**Expected:** The refusal names the helper, its would-be parent, and the reason, in plain
English, inside the ceremony narration.
**Why human:** None of the three gap-closure plans touch `build.md` or `continue.md` — confirmed
by diffing the gap-closure commit range against those files in this session (no changes). The
Go-level `Detail`/error string is proven correct and present in the CLI's own JSON/error output
(re-confirmed by my own manual reproductions above), but whether it reaches the wrapper-level
narration a non-technical operator actually reads has never been observed, in either verification
pass. Per CLAUDE.md, the owner of this repo is non-technical, so a refusal they cannot see is a
refusal that does not help them.

### Gaps Summary

The one gap from the initial verification — the whole-run spawn budget being resettable to zero by
tampering with `spawn-tree.txt` — is closed. I did not accept the gap-closure plans' summaries as
proof of this; I rebuilt the binary from the current source tree and personally re-ran my exact
original attack in a disposable scratch environment, and it now fails where it previously
succeeded, with the tampered file preserved as evidence rather than silently replaced. I also
personally attempted the second exploit route the team found while planning the fix (deleting, and
separately obstructing with a directory, the file that records which run is current) against a
valid, full ledger, and both attempts were refused. I confirmed the fix did not overcorrect by
checking that a genuinely fresh colony — no ledger file at all — can still spawn its first helper.

Every automated test named in the gap-closure plans and summaries was independently re-run in this
session and passed, including the full repository test suite (20 packages, no regressions). The
five comments that used to claim this kind of corruption "could never happen" are corrected and
dated, and a repo-wide grep confirms none of the six stale phrasings remain anywhere in the tree.

Five previously-flagged warnings (`.aether/workers.md`'s two documentation drifts, the advisory
`spawn-can-spawn` command's midden write, the hook's heuristic weakness, the check-then-act race,
the ancestor-chain silent-truncation edge case, and the `patrol.yaml`/wrapper drift) remain open —
none were in scope for any of the three gap-closure plans, and none of them are the criterion-4
guard-failure gap this re-verification was specifically checking. They are carried forward as
informational warnings, not blockers, exactly as in the initial verification.

The one remaining human-verification item is unchanged from the initial verification: whether a
refusal's plain-English reason actually surfaces in the `/ant-build`/`/ant-continue` narration a
non-technical operator reads, as opposed to only the raw CLI JSON error. No plan in this phase, in
either wave, has touched the wrapper files that would answer this. This is why the overall status
is `human_needed` rather than `passed`, even though all six ROADMAP success criteria and all eight
requirements are now independently verified: a human still needs to watch one real build hit a
guard and confirm the reason is visible where the owner would actually look.

---

_Verified: 2026-08-13T17:08:32Z_
_Verifier: Claude (gsd-verifier)_
