---
phase: 173-delegation-guard
plan: 05
subsystem: infra
tags: [spawn-guard, budget, midden, cobra, cli, go]

# Dependency graph
requires:
  - phase: 173-delegation-guard
    provides: "173-04's widened spawnCanSpawnDecision seam and the spawnTreeBudgetReason contract stub this plan fills in"
provides:
  - "spawnTreeBudgetReason is a real check: a whole-run ceiling of 20 helpers, counting the coordinator's own workers, that denies once the run's consumption reaches the ceiling (D-02/D-03)"
  - "spawnTreeBudgetState/spawnTreeBudgetWarning: reusable budget-state computation and a 75%-threshold plain-English warning line, surfaced unconditionally as budget_consumed/budget_max and conditionally as budget_warning on every spawn-log success payload (D-13)"
  - "A budget-ceiling refusal writes exactly one midden entry naming the delegation budget; a routine depth-cap refusal writes none (D-11)"
  - "The budget check fails closed on any unreadable run state (D-19), except a colony that has never begun a run, which is treated as a legitimately empty run (named exception, T-173-24)"
affects: [173-09, 173-10]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "spawnTreeBudget{Max,Consumed,Remaining,Reason} borrows only the '...Budget' struct-with-Reason naming convention from cmd/queen_spawn_budget.go's queenSpawnBudget, never its type, constants, or values — asserted both by value-space sweep and by source inspection in TestSpawnTreeBudgetAndWaveCapAreSeparateQuantities"
    - "Midden writes for a guard check are scoped to exactly one branch (the ceiling-exhausted branch), never the generic deny path — asserted in both directions (one entry on ceiling refusal, byte-identical file on depth refusal) rather than only the positive half"

key-files:
  created:
    - cmd/spawn_budget_test.go
  modified:
    - cmd/spawn_budget.go
    - cmd/spawn.go

key-decisions:
  - "A colony that has never begun a run is treated as Consumed 0 rather than denied, even though every other spawnTreeBudgetState error path fails closed (D-19). Rationale: a colony with no run history legitimately has no spawns yet, and denying here would lock a fresh colony out of spawning entirely (T-173-24) — this is the one place the budget check does not fail closed, and it is a named, deliberate exception, not an oversight."
  - "Test loop bounds for the two tests that spawn exactly spawnTreeBudgetMax helpers (TestSpawnTreeBudgetRefusesTheTwentyFirstHelper, TestBudgetCeilingWritesMiddenButDepthRefusalDoesNot) use a separate hardcoded literal (spawnTreeBudgetRedProofLiteralMax = 20) instead of referencing spawnTreeBudgetMax directly. Discovered during the D-22 red-proof: referencing the production constant in the loop bound made the test self-referential — widening spawnTreeBudgetMax to 1000 also widened the loop to 1000 iterations and the test still passed, silently failing to prove anything. This is the same numeric-coincidence trap plan 04's summary hit with the depth cap; the fix is the same: use a literal that does not move when the constant under test moves."

patterns-established:
  - "A guard-check's midden write is a named single call site (spawnTreeBudgetCeilingToMidden, called from exactly one branch), not a generic 'log every denial' path — future guard checks (plan 06's ancestor-cycle) should follow the same shape only where their own D-11-equivalent decision calls for it, not by default."

requirements-completed: [SPAWN-03]

# Metrics
duration: ~25min
completed: 2026-08-12
---

# Phase 173 Plan 05: Whole-Run Helper Budget Summary

**Filled `spawnTreeBudgetReason`'s contract stub with a real whole-run ceiling of 20 helpers (counting the coordinator's own workers), wired a 75%-threshold plain-English warning into `spawn-log`'s own output, and split midden logging so only a budget-ceiling refusal is recorded — a routine depth refusal is not.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 3 completed
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments

- `spawnTreeBudgetReason` is now a real check instead of an always-allow stub: a run that spawns 20 helpers across three separate waves of 8, 8, and 4 — never once exceeding the Queen's own per-wave cap of 8 — is still refused at helper 21, because the tree budget counts every helper in the run, not one wave's dispatch list (D-02/D-03/D-04). Proven by `TestSpawnTreeBudgetRefusesTheTwentyFirstHelper` and `TestSpawnTreeBudgetIsNotRestoredBetweenWaves`, the latter also asserting the reported consumed count is 16 (not 8) after two completed waves, so budget consumption is provably not reset by wave completion.
- The tree budget and the Queen's per-wave worker-selection cap (`cmd/queen_spawn_budget.go`) are asserted as genuinely separate quantities two ways: a combinatorial sweep across every flow type, phase mode, phase name, and verification depth `queenSpawnBudgetForPhase` accepts confirms its `MaxWorkers` never equals `spawnTreeBudgetMax` (20), and a source-inspection check confirms `cmd/spawn_budget.go` references none of `cmd/queen_spawn_budget.go`'s declared identifiers — `TestSpawnTreeBudgetAndWaveCapAreSeparateQuantities`.
- `spawn-log`'s success payload now unconditionally reports `budget_consumed`/`budget_max`, and adds a plain-English `budget_warning` line once a run crosses 75% of the ceiling (15 of 20) — proven directly by exercising helpers 1 through 15 in a scratch store and confirming the warning is absent at helper 5 and present at helper 15 (D-13). No dashboard, no polling loop; one line in the output the operator is already reading.
- The whole-run budget check fails closed when its own run-state file is corrupted (`spawn-runs.json` made invalid JSON), denying with a reason naming the budget as unverifiable, with a negative control in the same test proving the same call allows when the state is readable and consumption is low (D-19) — `TestSpawnTreeBudgetFailsClosedWhenTheTreeIsUnreadable`.
- Hitting the budget ceiling writes exactly one midden entry naming the delegation budget; a routine depth-cap refusal (spawning past the two-level cap from 173-04) writes nothing to the midden file at all, asserted both ways in one test so an implementation that logs every refusal cannot pass — `TestBudgetCeilingWritesMiddenButDepthRefusalDoesNot` (D-11).
- Both D-22 red-proofs performed and quoted below: widening `spawnTreeBudgetMax` to 1000 flips three tests to red; making `spawnTreeBudgetReason` allow on error flips the fail-closed test to red. Both restored with a verified zero-diff `git diff cmd/spawn_budget.go`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Count the whole run and refuse past 20** - `0f8bef79` (feat)
2. **Task 2: Surface the warning in the run's own output** - `e6c4d899` (feat)
3. **Task 3: Red-proofs that the two budgets are genuinely different numbers** - `186e94c1` (test)

**Plan metadata:** _pending_ (docs: complete plan — added by the orchestrator after all worktree agents in this wave merge)

## Files Created/Modified

- `cmd/spawn_budget.go` - filled the `CONTRACT-STUB-PLAN-05` stub: `spawnTreeBudgetMax = 20` and `spawnTreeBudgetWarnFraction = 0.75` constants (deliberately not configurable, D-02); `spawnTreeBudget` struct; `spawnTreeBudgetState()` computing consumption against the current run window via `agent.SpawnTree.CurrentRun`/`EntriesForRun`; `spawnTreeBudgetReason()` the real deny-or-allow check; `spawnTreeBudgetCeilingToMidden()` the single midden call site, reachable only from the ceiling-exhausted branch; `spawnTreeBudgetWarning()` the 75%-threshold plain-English line
- `cmd/spawn.go` - `spawnLogCmd` now calls `spawnTreeBudgetState()`/`spawnTreeBudgetWarning()` after a successful `RecordSpawn`, adding `budget_consumed`/`budget_max` unconditionally and `budget_warning` conditionally to the success payload; a budget-state read error is ignored so a reporting failure never turns an already-successful spawn into an error
- `cmd/spawn_budget_test.go` (NEW) - five named tests for SPAWN-03 (see Accomplishments), plus a hardcoded `spawnTreeBudgetRedProofLiteralMax = 20` constant used by the two tests that spawn exactly the ceiling's worth of helpers, kept deliberately separate from `spawnTreeBudgetMax` (see Decisions Made)

## Decisions Made

- **A colony that has never begun a run is Consumed 0, not denied** — the one place `spawnTreeBudgetState` does not fail closed. Every other error path (unresolvable run state, unreadable entries) denies per D-19; a colony with no run history yet has no spawns to count, and denying here would lock a brand-new colony out of spawning at all (T-173-24). Named and justified per the plan's explicit requirement to record this choice.
- **Test loop bounds needed their own literal, separate from the production constant.** During the D-22 red-proof, widening `spawnTreeBudgetMax` from 20 to 1000 did not flip `TestSpawnTreeBudgetRefusesTheTwentyFirstHelper` or `TestBudgetCeilingWritesMiddenButDepthRefusalDoesNot` to red on the first attempt, because both tests' spawn-loop bound was `spawnTreeBudgetMax` itself — the loop widened right alongside the constant under test, so both tests kept passing (with 1000 successful spawns instead of 20) without ever exercising a refusal. Fixed by introducing `spawnTreeBudgetRedProofLiteralMax = 20`, a hardcoded literal the two tests use for their loop bounds instead, with a comment explaining why. Re-ran the red-proof afterward and confirmed all three budget-count-dependent tests correctly went red.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Comment text in cmd/spawn_budget.go inadvertently referenced `queenSpawnBudget`/`MaxWorkers`, violating the plan's own acceptance criterion**
- **Found during:** Task 1 verification (`grep -rn 'queenSpawnBudget\|MaxWorkers' cmd/spawn_budget.go`)
- **Issue:** The doc comments explaining D-04's separation named `queenSpawnBudget` and `MaxWorkers` literally, which the plan's acceptance criterion checks with a plain grep against the whole file — comments included. The comment text itself accidentally violated the identifier-isolation rule it was explaining.
- **Fix:** Reworded both comments to describe `cmd/queen_spawn_budget.go`'s type and field in prose ("the Queen's per-wave, pre-dispatch worker-selection cap defined in cmd/queen_spawn_budget.go (a range of 4 to 8)") without spelling out the exact identifiers.
- **Files modified:** cmd/spawn_budget.go
- **Verification:** `grep -rn 'queenSpawnBudget\|MaxWorkers' cmd/spawn_budget.go` returns nothing; `go build`/`go vet` clean
- **Committed in:** `0f8bef79` (Task 1 commit)

**2. [Rule 1 - Bug] `store.AtomicWrite` validates JSON for `.json` paths, blocking the planned corruption technique for Task 3's fail-closed test**
- **Found during:** Task 3, writing `TestSpawnTreeBudgetFailsClosedWhenTheTreeIsUnreadable`
- **Issue:** The test needed to corrupt `spawn-runs.json` to force `CurrentRun()`/`EntriesForRun()` to return an error. `store.AtomicWrite` (the normal write path) validates JSON content before writing to any `.json` path by design, so writing invalid JSON through it fails with a storage-layer error rather than corrupting the file.
- **Fix:** Wrote the corrupted bytes directly to the resolved on-disk path (`filepath.Join(store.BasePath(), "spawn-runs.json")` via `os.WriteFile`), standing in for the "file was corrupted by something other than this store" case `AtomicWrite`'s own guard cannot prevent.
- **Files modified:** cmd/spawn_budget_test.go
- **Verification:** `TestSpawnTreeBudgetFailsClosedWhenTheTreeIsUnreadable` passes, including its negative control
- **Committed in:** `186e94c1` (Task 3 commit)

**3. [Rule 1 - Bug] Two tests' spawn-loop bounds referenced the production constant under test, making the D-22 red-proof self-referential**
- **Found during:** Task 3's D-22 red-proof exercise (widening `spawnTreeBudgetMax` to 1000)
- **Issue:** `TestSpawnTreeBudgetRefusesTheTwentyFirstHelper` and `TestBudgetCeilingWritesMiddenButDepthRefusalDoesNot` both looped `for i := 1; i <= spawnTreeBudgetMax; i++` to spawn exactly the ceiling's worth of helpers. Widening the constant to 1000 widened this loop bound too, so both tests still passed (spawning 1000 helpers successfully instead of hitting a refusal at 21) — the tests were vacuously true against the very constant they were meant to prove.
- **Fix:** Introduced `spawnTreeBudgetRedProofLiteralMax = 20` as a hardcoded literal, documented as deliberately independent of `spawnTreeBudgetMax`, and switched both tests' loop bounds to it. Re-ran the red-proof: all three budget-count-dependent tests (including the previously-silent two) now correctly go red when the constant widens.
- **Files modified:** cmd/spawn_budget_test.go
- **Verification:** Red-proof output quoted below; both tests go red on widening and green on restore
- **Committed in:** `186e94c1` (Task 3 commit)

---

**Total deviations:** 3 auto-fixed, all Rule 1 (bug fixes to keep the implementation and its own tests honest about what they prove)
**Impact on plan:** All three were caught by the plan's own verification steps (an acceptance-criteria grep, a JSON-validating write path, and the D-22 red-proof itself) before commit. No scope creep — no file outside `files_modified` was touched.

## Issues Encountered

- A background `go test ./...` run during Task 3 verification reported one unrelated failure, `TestCodexReadOnlyProfileSelectsReadOnlySandbox` in `pkg/codex`, with the message "codex login status failed: timed out" — an environment/network dependency on the `codex` CLI being logged in, unrelated to any file this plan touches. Re-ran the full suite afterward and it passed cleanly, confirming this was a transient environment flake, not a regression from this plan's changes. Per the deviation rules' scope boundary, this was not investigated further as it sits outside `cmd/spawn_budget.go`, `cmd/spawn.go`, and `cmd/spawn_budget_test.go`.

## Red-Proof Evidence (D-22)

**Red-proof 1 — widened `spawnTreeBudgetMax` from 20 to 1000:**

```
=== RUN   TestSpawnTreeBudgetRefusesTheTwentyFirstHelper
    spawn_budget_test.go:97: the 21st spawn-log did not exit non-zero: stdout={"ok":true,"result":{"budget_consumed":21,"budget_max":1000,...,"name":"W21","parent":"Queen",...}}
--- FAIL: TestSpawnTreeBudgetRefusesTheTwentyFirstHelper (0.02s)
=== RUN   TestSpawnTreeBudgetIsNotRestoredBetweenWaves
    spawn_budget_test.go:181: the 21st spawn, spread across three waves none of which exceeded 8, did not exit non-zero: stdout={"ok":true,"result":{"budget_consumed":21,"budget_max":1000,...,"name":"D1","parent":"Queen",...}}
--- FAIL: TestSpawnTreeBudgetIsNotRestoredBetweenWaves (0.03s)
=== RUN   TestBudgetCeilingWritesMiddenButDepthRefusalDoesNot
    spawn_budget_test.go:367: budget-ceiling spawn attempt did not exit non-zero: stdout={"ok":true,"result":{"budget_consumed":21,"budget_max":1000,...,"name":"W21","parent":"Queen",...}}
--- FAIL: TestBudgetCeilingWritesMiddenButDepthRefusalDoesNot (0.02s)
```

All three flipped to red as required. Restored `spawnTreeBudgetMax` to 20; `git diff --stat cmd/spawn_budget.go` showed no output (zero residual diff); re-ran and all five tests passed green.

**Red-proof 2 — made `spawnTreeBudgetReason` return empty (allow) on error instead of denying:**

```
=== RUN   TestSpawnTreeBudgetFailsClosedWhenTheTreeIsUnreadable
    spawn_budget_test.go:324: spawn-log with an unreadable run state did not exit non-zero: stdout={"ok":true,"result":{"caste":"builder","claimed_depth":0,"depth":1,...,"name":"W2","parent":"Queen","recorded":true,"task":"t"}}
--- FAIL: TestSpawnTreeBudgetFailsClosedWhenTheTreeIsUnreadable (0.01s)
```

Flipped to red as required. Restored the original error-handling branch (a saved copy of the file, swapped back in); `git diff --stat cmd/spawn_budget.go` showed no output; re-ran and all five tests passed green.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Plan 09** (abandoned/reaped helpers) can now define the authoritative `abandoned` status constant this plan referenced by literal string (`spawnTreeBudgetAbandonedStatus = "abandoned"`, commented as owned by plan 09) — once that status exists, this plan's exclusion of abandoned entries from the budget count (D-18: reaping releases budget) is already wired to honor it.
- **Plan 10**'s marker-survival assertion has one fewer string to check for: `CONTRACT-STUB-PLAN-05` in `cmd/spawn_budget.go` is now gone (confirmed by `grep -c` returning 0). `CONTRACT-STUB-PLAN-06` in `cmd/spawn_ancestor.go` remains plan 06's responsibility, executed in parallel with this plan.
- **Plan 10** should also add `TestSpawnTreeBudgetRefusesTheTwentyFirstHelper`, `TestSpawnTreeBudgetIsNotRestoredBetweenWaves`, `TestSpawnTreeBudgetAndWaveCapAreSeparateQuantities`, `TestSpawnTreeBudgetFailsClosedWhenTheTreeIsUnreadable`, and `TestBudgetCeilingWritesMiddenButDepthRefusalDoesNot` to the CI wiring-gate filter, per this plan's own explicit instruction not to do so itself (the file was not yet tracked in `wiringGateGuardFiles` at plan-05 time).
- No blockers for plan 06 (ancestor-cycle, running in parallel in a separate worktree) or plan 10 (final wiring/marker cleanup).

## Known Stubs

None. `spawnTreeBudgetReason` and its supporting functions are fully implemented, tested, and red-proofed — no placeholder behavior remains in the files this plan touched.

---
*Phase: 173-delegation-guard*
*Completed: 2026-08-12*

## Self-Check: PASSED

- FOUND: cmd/spawn_budget.go
- FOUND: cmd/spawn.go
- FOUND: cmd/spawn_budget_test.go
- FOUND: .planning/phases/173-delegation-guard/173-05-SUMMARY.md
- FOUND commit 0f8bef79 (Task 1)
- FOUND commit e6c4d899 (Task 2)
- FOUND commit 186e94c1 (Task 3)
