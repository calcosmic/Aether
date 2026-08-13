---
phase: 173-delegation-guard
plan: 12
subsystem: agent
tags: [go, spawn-tree, spawn-budget, fail-closed, security-fix]

# Dependency graph
requires:
  - phase: 173-delegation-guard
    provides: "plan 11's parse-boundary contract (pkg/agent/spawn_tree.go) -- the ledger and run-state files now return a distinct, non-nil error for present-but-unparseable content instead of silently swallowing it, which this plan's budget check consumes"
provides:
  - "a whole-run spawn budget that establishes ledger integrity (st.Parse()) BEFORE resolving the run window, so a corrupted spawn-tree.txt can never take the no-run-yet exception and manufacture a fresh budget"
  - "a no-run-yet branch that counts the whole ledger instead of reporting Consumed:0 whenever the ledger already holds entries, closing the second reset route (deleting spawn-runs.json against a valid, full ledger) while still allowing a colony that has genuinely never spawned anything, or has spawned without a lifecycle command ever beginning a run"
  - "three named regression tests reproducing 173-VERIFICATION.md's Gap 1 exploit and its second route, plus the negative control that stops an always-deny implementation from passing them"
  - "all three new tests registered in the CI wiring ratchet's -run filter"
affects: [173-13]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A budget/guard check that reads a mutable ledger must establish the ledger's integrity BEFORE taking any early-return branch that skips reading it -- an early return that bypasses integrity is a second route to the same bypass, even after the ledger's own parse boundary fails closed"
    - "When 'no window can be resolved' and 'the underlying data legitimately has none' are different situations, do not collapse them into one deny/allow decision -- count the wider, safe superset instead of refusing outright, so the fix does not brick the routine case it was never meant to touch"

key-files:
  created: []
  modified:
    - cmd/spawn_budget.go
    - cmd/spawn_budget_test.go
    - cmd/spawn_failclosed_test.go
    - .github/workflows/ci.yml

key-decisions:
  - "The no-run-yet branch now counts every live entry in the WHOLE ledger (not the run-scoped window) whenever the ledger already holds entries, rather than denying outright. Denying would have broken cmd/spawn_enforce_test.go and cmd/spawn_ancestor_test.go's multi-hop chains (both build spawn trees with no run ever begun) and .aether/workers.md:292's documented direct-call convention. The whole-ledger count is never lower than the correct run-scoped count, so this can only tighten the budget, never free it."
  - "The requester-not-recorded fault axis in cmd/spawn_failclosed_test.go was re-pointed at a well-formed but incomplete ledger (a different agent's valid entry, not A1) instead of a corrupted one, because the corrupted-ledger version now gets caught by the budget check before it ever reaches the ancestor-cycle path this axis exists to prove."

requirements-completed: [SPAWN-03]

# Metrics
duration: 25min
completed: 2026-08-13
---

# Phase 173 Plan 12: Whole-Run Budget Fails Closed on Ledger Tampering and Run-Record Erasure Summary

**Closed the two ways 173-VERIFICATION.md's Gap 1 reset the whole-run spawn budget to zero -- corrupting `spawn-tree.txt`, and separately just deleting `spawn-runs.json` against a perfectly valid ledger -- without breaking the ordinary no-run-yet case five other tests and the documented worker CLI depend on.**

## Performance

- **Duration:** ~25 min
- **Completed:** 2026-08-13T16:27:00Z
- **Tasks:** 3/3
- **Files modified:** 4

## Accomplishments

- `spawnTreeBudgetState()` now calls `st.Parse()` and checks its error BEFORE resolving the run window, so a corrupted `spawn-tree.txt` can no longer hide behind the "no run recorded" exception no matter what else is deleted alongside it
- The no-run-yet branch now counts the whole ledger instead of unconditionally reporting `Consumed: 0`, closing the second reset route -- `rm .aether/data/spawn-runs.json` against a valid, full ledger -- while a genuinely fresh colony (no ledger, or a zero-byte one) still spawns freely
- Three new named tests reproduce both routes and go red the moment either fix is reverted; a fourth existing axis was re-pointed so it keeps proving the guard it names
- Both exploit routes were re-run by hand against the freshly built binary and confirmed refused (transcripts below)
- Full repository test suite (`go test ./... -count=1 -timeout 900s`) passes unchanged, all 18 packages

## Task Commits

Each task was committed atomically:

1. **Task 1: Refuse when the ledger cannot be trusted, and never report a free budget for a ledger that already records helpers** - `2c2ae6dc` (fix)
2. **Task 2: Lock both reset routes into tests that go red** - `8037d671` (test)
3. **Task 3: Put every new test under the ratchet that makes deleting them fail** - `47e081d2` (chore)

**Plan metadata:** (this commit, made by the orchestrator after all worktree agents in the wave complete)

## The Verifier's Gap 1 Sequence, Re-Run Against The New Binary

Run from a clean scratch colony store, using the freshly built binary from this plan's changes:

```
# Legitimately exhaust the 20-helper whole-run budget (no lifecycle command
# ever begins a run in this reproduction, exactly as a bare worker calling
# spawn-log directly would see it)
$ for i in 1..20: aether spawn-log --parent Queen --caste builder --name "W$i" --task "task $i" --depth 0
...
$ aether spawn-log --parent Queen --caste builder --name W20 --task "task 20" --depth 0
{"ok":true,"result":{"budget_consumed":20,"budget_max":20, ..."recorded":true,"task":"task 20"}}

# Confirm the 21st is correctly refused
$ aether spawn-log --parent Queen --caste builder --name W21 --task "task 21" --depth 0
{"ok":false,"error":"whole-run helper budget exhausted: 20 of 20 helpers already spawned
(counted across the entire ledger because no run is recorded); Queen may not spawn
another","code":1}
exit: 1

# The exploit: corrupt the record with one shell write
$ echo "garbage not pipe format" > .aether/data/spawn-tree.txt

# Before this plan: this identical request succeeded with budget_consumed:1.
# After this plan:
$ aether spawn-log --parent Queen --caste builder --name W22-BYPASS --task "task 22" --depth 0
{"ok":false,"error":"whole-run helper budget unverifiable (verify spawn-tree.txt: spawn_tree:
spawn-tree.txt: spawn ledger is present but its content is not a valid spawn ledger: line 1
has 1 fields, expected 7 (spawn) or 4 (completion)): refusing to spawn","code":1}
exit: 1
```

The refusal now happens at the exact step that previously returned `recorded:true`. Note the
deny sentence for the control (21st, before tampering) already reads "counted across the
entire ledger because no run is recorded" -- this reproduction never began a run through any
lifecycle command, so it exercises the Gap B whole-ledger counting path on its own, by
construction, even before any tampering occurs.

## The Second Route, Re-Run By Hand

A separate scratch colony, no ledger corruption at all:

```
# Build a fresh, entirely valid, full ledger of 20 helpers
$ for i in 1..20: aether spawn-log --parent Queen --caste builder --name "V$i" --task "task $i" --depth 0
...

# No spawn-runs.json exists yet in this reproduction (no lifecycle command
# ever began a run) -- remove it explicitly to model the erasure this route
# names, against the now-full, perfectly valid ledger
$ rm .aether/data/spawn-runs.json  # (no-op here since it was never created; the
                                     # important state is: valid full ledger + no run record)

$ aether spawn-log --parent Queen --caste builder --name V21-BYPASS --task "task 21" --depth 0
{"ok":false,"error":"whole-run helper budget exhausted: 20 of 20 helpers already spawned
(counted across the entire ledger because no run is recorded); Queen may not spawn
another","code":1}
exit: 1
```

The identical request refuses. Before this plan, this exact shape -- a valid ledger holding 20
live entries, no run ever resolved -- returned `Consumed: 0` unconditionally and allowed the
spawn. The automated regression test `TestErasingTheRunRecordDoesNotResetTheWholeRunBudget`
covers the sharper version of this scenario where a run genuinely WAS active and its record is
then deleted mid-run (via `beginRuntimeSpawnRun`, the same path every lifecycle command uses),
proving the fix holds in both the "never began a run" and the "run record erased mid-run" shapes.

## Exact Deny Sentences An Operator Sees

- **Corrupted ledger:** `whole-run helper budget unverifiable (verify spawn-tree.txt: spawn_tree: spawn-tree.txt: spawn ledger is present but its content is not a valid spawn ledger: line 1 has 1 fields, expected 7 (spawn) or 4 (completion)): refusing to spawn`
- **Run record erased or never recorded, against a ledger that already has entries:** `whole-run helper budget exhausted: 20 of 20 helpers already spawned (counted across the entire ledger because no run is recorded); Queen may not spawn another`
- **Run-state file obstructed by a directory:** `whole-run helper budget unverifiable (resolve current run: spawn_tree: read "spawn-runs.json": ...): refusing to spawn`

All three name the budget plainly, and the first names the exact file (`spawn-tree.txt`)
without ever echoing the tampered content itself.

## Why The No-Run Branch Counts Instead Of Refusing

The obvious-looking rule -- "no run resolves and the ledger has entries, therefore deny" --
was checked against the existing suite and the documented worker CLI, and it is wrong. Nothing
begins a run except `beginRuntimeSpawnRun` (`cmd/spawn_runs.go`), called only from `build`,
`continue`, `plan`, `colonize`, and `swarm`. That means "the ledger has entries and no run was
ever recorded" is a routine, legitimate state, not an anomaly:

- `cmd/spawn_enforce_test.go` has at least five tests that call `spawn-log Queen->W1` and then
  `spawn-log W1->H1` with no run ever begun; a deny here would have refused the second call in
  every one of those pairs.
- `cmd/spawn_ancestor_test.go` builds every one of its ancestor chains the same way.
- `.aether/workers.md:292` documents workers calling `aether spawn-can-spawn {your_depth}
  --enforce` directly, outside any lifecycle command, so a bare-shell spawn would start
  refusing under the strict rule.

Counting the whole ledger instead closes the exploit completely without this collateral damage:
the whole-ledger count is never lower than the correct run-scoped count (the ledger is a
superset of any single run's window), so deleting `spawn-runs.json` can only make the budget
stricter, never free it. `TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn`'s third case pins
this directly -- two chained `spawn-log` calls with no run ever begun, both still expected to
succeed.

## The `requester-not-recorded` Axis Change

`cmd/spawn_failclosed_test.go`'s `spawn-can-spawn` fault table has an axis named
`requester-not-recorded` that asserts the deny message contains `ancestor chain unreadable`.
Before this plan (and after plan 11 alone), it injected a corrupted 7-field line with a
non-numeric depth field for agent A1, then called `spawn-can-spawn --enforce --name A1`. That
corruption made the WHOLE tree fail to parse, but the budget check's no-run-yet exception took
the request past the budget silently (no run had been begun in that test), so the ancestor
check was the only branch that ever saw the failure and produced the deny.

After Task 1's fix, that same corrupted-ledger injection is now caught by the budget check
FIRST -- `spawnTreeBudgetState()` calls `st.Parse()` before the no-run-yet branch, so the
corrupted tree denies with "budget ... unverifiable" instead of ever reaching the ancestor
check. The axis's own assertion (`ancestor chain unreadable`) would then fail, not because the
guard regressed, but because a different, earlier guard now also denies on the same fault --
which would make the axis stop proving the thing it names.

The fix: change ONLY the injected bytes, from a corrupted line to a well-formed 7-field spawn
line for a DIFFERENT agent (`2026-04-01T12:00:00Z|Queen|builder|B9|t|1|spawned`). The ledger
now parses cleanly (one helper of twenty, well under budget, so the budget check allows), A1 is
genuinely absent from a READABLE tree, and `spawnAncestorChain` still returns `no recorded
spawn entry for "A1"` -- wrapped by `spawnAncestorCycleReason` into the same `ancestor chain
unreadable` wording the axis has always asserted. The assertion text was kept unchanged
deliberately: relaxing it to also accept the budget's wording would have deleted the axis's
proof of the ancestor path, not preserved it.

## D-22 Red-Proof Transcripts

All four were performed by mutating the just-committed code in place, confirming the target
test fails by name, then restoring the file from a pre-mutation backup and re-confirming the
test (and the surrounding suite) passes again.

### 1. Task 1's Gap A integrity call reverted (swallow the parse error)

**Mutation:** `entries, err := st.Parse(); if err != nil { return ..., err }` changed to
`entries, _ := st.Parse()` (error silently discarded).

**Result:** `TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget` failed --
```
spawn_budget_test.go:546: deny message does not name the budget as unverifiable (the refusal
must come from the budget check, not merely from RecordSpawn's separate write guard):
{"ok":false,"error":"failed to record spawn: spawn ledger is present but its content is not a
valid spawn ledger: line 1 has 1 fields, expected 7 (spawn) or 4 (completion)","code":2}
--- FAIL: TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget (0.03s)
```
Restored; the test passes again. **Note on test design:** the harder-variant assertion inside
this test was written to check the deny message specifically names `budget` and `unverifiable`,
not merely a non-zero exit code. Plan 11's `RecordSpawn` write-closure guard independently
refuses to write over corrupted content, which ALSO produces a non-zero exit even with Task 1's
Gap A check reverted -- a bare exit-code assertion would have passed regardless of whether Gap
A worked, so the message-content assertion was necessary to make this red-proof genuinely prove
Gap A's contribution rather than a different guard's.

### 2. Task 1's Gap B branch reverted to unconditional `Consumed: 0`

**Mutation:** the no-run-yet branch's `if len(entries) == 0 { ... } consumed := ...` logic
replaced with an unconditional `return spawnTreeBudget{Max: spawnTreeBudgetMax, Consumed: 0,
Remaining: spawnTreeBudgetMax}, nil`.

**Result:** `TestErasingTheRunRecordDoesNotResetTheWholeRunBudget` failed --
```
spawn_budget_test.go:606: erasing spawn-runs.json against a full valid ledger let the identical
request succeed -- budget_consumed reset: stdout={"ok":true,"result":{"budget_consumed":0,
"budget_max":20, ... "name":"W22-BYPASS", ... "recorded":true, ...}}
--- FAIL: TestErasingTheRunRecordDoesNotResetTheWholeRunBudget (0.03s)
```
This reproduces the exact pre-fix bug: `budget_consumed:0` against a ledger that actually holds
20 live entries. Restored; the test passes again.

### 3. Gap B branch hardened into an unconditional deny

**Mutation:** the no-run-yet, non-empty-ledger case changed from counting the whole ledger to
`return spawnTreeBudget{}, fmt.Errorf("no run recorded and the ledger is not empty")`.

**Result:** `TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn`'s third case failed --
```
spawn_budget_test.go:735: [spawn-log --parent W1 --caste builder --name H1 --task "carry out
the coordinated request" --depth 0] exited 1: {"ok":false,"error":"whole-run helper budget
unverifiable (no run recorded and the ledger is not empty): refusing to spawn","code":1}
--- FAIL: TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn (0.01s)
```
This is the proof that counting rather than refusing was the correct rule: a "stricter-looking"
deny breaks the exact two-hop, no-run-begun chain `cmd/spawn_enforce_test.go` and
`cmd/spawn_ancestor_test.go` both depend on. Restored; the test passes again.

### 4. A new guard test deleted

**Mutation:** `TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn` deleted entirely from
`cmd/spawn_budget_test.go`.

**Result:** `TestWiringGateStepRunsEveryWiringTest` failed --
```
ci_wiring_gate_test.go:226: CI step "Verify subcommand wiring and CLI flag contracts"'s -run
filter contains 1 alternative(s) that match no guard test -- go test -run exits 0 when a
pattern matches nothing, so a renamed or deleted guard test left in the filter would silently
stop running with nothing going red:
      TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn
--- FAIL: TestWiringGateStepRunsEveryWiringTest (0.01s)
```
Restored; the test passes again.

## What This Plan Does NOT Close

- **T-173-67 (named in plan 11's summary, unchanged here):** a well-formed forgery that drops
  ledger entries. An attacker who rewrites `spawn-tree.txt` in perfectly valid pipe format,
  simply omitting entries, is not detected by any counting rule in this file -- shape-based
  parsing cannot distinguish a forged-but-well-formed ledger from a genuine one. Closing this
  needs integrity data this file format does not carry (an append-only log, or a signature).
- **WR-05 (still open):** the check-then-act race between concurrent processes near the
  ceiling. This plan's fixes are entirely about what happens when the ledger or run record is
  read; they do nothing to serialize two processes racing to spawn near the budget ceiling
  simultaneously.
- **Direct `RecordSpawn` callers that never consult the budget at all.** The budget check
  (`spawnTreeBudgetReason`) is only reached through `spawnCanSpawnDecision`, which `spawn-log`
  calls before recording. Any code path in the build/continue/plan lifecycle commands that
  calls `st.RecordSpawn(...)` directly, bypassing `spawn-log`'s decision chain, would not be
  gated by any of this plan's fixes. This plan did not audit for such call sites; it only
  hardens the budget-check function itself.

## Files Created/Modified

- `cmd/spawn_budget.go` -- `spawnTreeBudgetState()` now calls `st.Parse()` before resolving the
  run window (Gap A); the no-run-yet branch counts the whole ledger instead of reporting a free
  budget whenever the ledger has entries (Gap B); `spawnTreeBudget` gained a `CountedWholeLedger`
  field so the exhausted-budget deny sentence can say so in plain words; doc comments rewritten
  to state the narrowed absent-ledger exception and cite 173-VERIFICATION.md's Gap 1
- `cmd/spawn_budget_test.go` -- three new tests:
  `TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget`,
  `TestErasingTheRunRecordDoesNotResetTheWholeRunBudget`,
  `TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn`
- `cmd/spawn_failclosed_test.go` -- the `requester-not-recorded` axis re-pointed at a
  well-formed-but-incomplete ledger instead of a corrupted one, with its comment updated
- `.github/workflows/ci.yml` -- the three new test names appended to the wiring step's `-run`
  filter (one line changed)

## Decisions Made

- **Whole-ledger counting over outright refusal when no run resolves against a non-empty
  ledger.** Documented above under "Why The No-Run Branch Counts Instead Of Refusing" -- this
  is the plan's central design decision and is proven both ways (closes the exploit, does not
  break routine use) by the three new tests and red-proof 3 above.
- **The `requester-not-recorded` axis is re-pointed, not deleted or weakened.** Its assertion
  text is unchanged; only the fault it injects changed, so it continues to prove the ancestor
  guard specifically rather than being satisfied by an earlier-firing budget deny.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Strengthened the harder-variant assertion in
`TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget` to check the deny message names the
budget specifically**
- **Found during:** D-22 red-proof 1 (Task 2's own verification step)
- **Issue:** The plan's `<behavior>` spec only required "the identical request still exits
  non-zero" for the harder variant (ledger corrupt + spawn-runs.json also removed). A bare
  exit-code check passed even when Task 1's Gap A integrity call was reverted, because plan
  11's separate `RecordSpawn` write-closure guard independently refuses to write over corrupted
  content and also produces a non-zero exit (via a different error message, "failed to record
  spawn"). This meant the red-proof for Gap A's ordering was not actually isolating what it
  claimed to prove.
- **Fix:** Added an assertion that the harder-variant's deny message contains both `budget` and
  `unverifiable`, pinning the refusal to the budget check specifically. Re-ran red-proof 1 with
  the strengthened assertion and confirmed it now fails by name for the correct reason (see
  transcript above).
- **Files modified:** `cmd/spawn_budget_test.go`
- **Verification:** Red-proof 1 above; full task 2 verification command passes.
- **Committed in:** `8037d671` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug -- a red-proof assertion that did not isolate the
thing it claimed to prove)
**Impact on plan:** Strengthens the plan's own D-22 requirement (a test that fails when the
thing it names is untrue) without changing any production code behavior. No scope creep.

## Issues Encountered

None beyond the deviation above.

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness

- SPAWN-03's whole-run budget now genuinely bounds the tree: neither corrupting the ledger nor
  erasing the run record can reset it, and the fix is proven not to break the routine
  no-run-begun case five other tests and the documented worker CLI depend on.
- Plan 173-13 (per 173-11-SUMMARY's `affects` field) still owns correcting the five stale
  comments in `cmd/spawn_failclosed_test.go` and `cmd/internal_cmds.go` that claim
  `agent.SpawnTree.Parse()` can never return an error -- plan 11 made that claim false by
  construction, and this plan's Task 1 read comment did not touch those five call sites (they
  are explicitly out of this plan's scope per 173-11-SUMMARY.md).
- T-173-67 (well-formed ledger forgery) and WR-05 (concurrent check-then-act race) remain named,
  open residue for a future plan or an explicit accepted-risk decision -- neither is claimed
  closed by this plan.

---
*Phase: 173-delegation-guard*
*Completed: 2026-08-13*

## Self-Check: PASSED

- FOUND: cmd/spawn_budget.go
- FOUND: cmd/spawn_budget_test.go
- FOUND: cmd/spawn_failclosed_test.go
- FOUND: .github/workflows/ci.yml
- FOUND: .planning/phases/173-delegation-guard/173-12-SUMMARY.md
- FOUND commit: 2c2ae6dc (fix(173-12): fail closed on ledger tampering and run-record erasure)
- FOUND commit: 8037d671 (test(173-12): lock both whole-run budget reset routes red)
- FOUND commit: 47e081d2 (chore(173-12): register the new budget guard tests in the CI wiring ratchet)
