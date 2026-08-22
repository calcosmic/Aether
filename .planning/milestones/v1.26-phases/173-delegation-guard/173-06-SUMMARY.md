---
phase: 173-delegation-guard
plan: 06
subsystem: infra
tags: [spawn-guard, ancestor-cycle, cobra, cli, go]

# Dependency graph
requires:
  - phase: 173-delegation-guard
    provides: "173-04's widened spawnCanSpawnDecision seam and the spawnAncestorCycleReason(in spawnDecisionInput) string contract stub this plan fills, plus the spawnDecisionInput/spawnDecisionResult vocabulary and spawnParentIsRoot/latestSpawnEntryByName primitives it is built on"
provides:
  - "spawnAncestorCycleReason is a real ancestor-chain cycle check instead of an always-allow stub: it refuses a spawn whose caste and normalised task already appear in its own ancestor chain, naming the matched ancestor"
  - "spawnAncestorChain(st, startName): parses the shared agent.SpawnTree once and walks it in memory, bounded by a visited-name set, returning the requester's own entry followed by its ancestors up to the coordinator root"
  - "normalizeSpawnTask(task): a small, exhaustively-testable text normaliser (lowercase, whitespace-collapse, trim, one trailing full stop) that is the bounded, named residue -- text matching, not semantic matching"
affects: [173-10]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Ancestor-chain walk built entirely on the shared agent.SpawnTree.Parse() output, parsed once per check and never re-parsed per hop -- the fourth divergent hand-rolled pipe parser this repo almost grew, avoided"
    - "Fail-closed on unreadable ancestry: an unresolvable requester name (not the coordinator sentinel) is a deny, never an allow, matching the depth check's D-19 posture from plan 04"

key-files:
  created: []
  modified:
    - cmd/spawn_ancestor.go
    - cmd/spawn_ancestor_test.go
    - cmd/spawn_enforce_test.go

key-decisions:
  - "Termination of the ancestor walk is bounded by a visited-name set, not a hop-count limit. The plan offered either as acceptable and named the visited-set as 'preferable if simpler' -- it is simpler (no dependency on spawnMaxDelegationDepth's arithmetic staying in sync) and is correct even if the cap constant ever changes."
  - "spawn-can-spawn registers no --caste/--task flags (only spawn-log's authoritative call populates spawnDecisionInput.Caste/Task), so TestSpawnCanSpawnDeniesAncestorCycle cannot get a JSON 'reason' field from the CLI for this specific check. It proves the CLI-level deny (exit code, message names the ancestor, spawn-tree.txt byte-identical) end to end via spawn-log, then separately calls the real, un-substituted spawnCanSpawnDecision var directly with the exact input spawn-log's RunE built, to confirm Reason == 'ancestor-cycle' and not 'depth'. This does not touch cmd/spawn.go (out of scope for this plan, and shared with the parallel 173-05 worktree) -- it calls the existing production seam."
  - "Four pre-existing tests in cmd/spawn_enforce_test.go (TestSpawnLogDerivesDepthFromRecordedParent, TestSpawnTreeDepthReportsTwoForAThreeLevelTree, TestSpawnLogRefusesPastCapAndWritesNoEntry, TestSpawnCanSpawnNameOverridesClaimedDepth) built two-level chains using the shared spawnLogArgs helper's fixed caste 'builder' and fixed task 't' at both levels. The new ancestor-cycle check correctly refuses that as a repeat -- these tests' actual subject is depth derivation, the depth cap, and depth override, not ancestor-cycle behaviour, so they were repaired with distinct task text per level (via the new spawnLogArgsWithCasteTask helper) rather than left broken. Same pattern as 173-04 SUMMARY's Deviation 1."

patterns-established:
  - "spawnLogArgsWithCasteTask(parent, name, caste, task) in cmd/spawn_ancestor_test.go: a spawn-log argument builder that, unlike spawn_enforce_test.go's fixed-caste/fixed-task spawnLogArgs, lets a test control caste and task independently -- needed by any future test that builds a multi-level chain and must avoid accidentally tripping the ancestor-cycle check."

requirements-completed: [SPAWN-05]

# Metrics
duration: 15min
completed: 2026-08-12
---

# Phase 173 Plan 06: Ancestor-Chain Cycle Detection Summary

**Filled `spawnAncestorCycleReason`'s contract stub with a real ancestor-chain walk (built on the shared `agent.SpawnTree.Parse()`, not a new hand-rolled parser) that catches a caste repeating the exact task it was itself spawned to do, one level down -- the loop a depth cap alone cannot see because each hop stays within the cap and the chain never terminates on its own.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-08-12T21:05:48Z
- **Completed:** 2026-08-12T21:20:10Z
- **Tasks:** 2 completed
- **Files modified:** 3 (2 test files, 1 source file)

## Accomplishments

- `spawnAncestorCycleReason` is now a real check instead of the always-allow `CONTRACT-STUB-PLAN-06` stub plan 04 wired in: it builds the requester's own ancestor chain, compares each ancestor's caste (case-insensitively) and normalised task against the prospective child, and denies on the first match -- naming the ancestor and its depth in the message, per D-10.
- `spawnAncestorChain` parses the spawn tree exactly once per check via the shared `agent.SpawnTree.Parse()` and walks the result in memory. Termination is bounded by a visited-name set (chosen over a hop-count cap, per the plan's own preference when simpler) -- a self-referential or mutually-referential parent link in a corrupted tree cannot loop the walk forever.
- `normalizeSpawnTask` applies exactly four collapsing rules and no more -- lowercase, whitespace-run collapse, leading/trailing trim, and a single trailing full stop -- proven by a table test asserting both colliding pairs and three pairs that must NOT collide (differing by a middle word, a number, and an interior punctuation mark). This is T-173-31's named, bounded residue: the guard matches text, not meaning.
- An A-to-B-to-A cycle that never exceeds the depth cap (each hop is one level down) is caught end to end through `spawn-log`, proven by `TestSpawnCanSpawnDeniesAncestorCycle`: exit code non-zero, the matched ancestor named in the message, `spawn-tree.txt` byte-identical to before the refused attempt, and the underlying `spawnDecisionResult.Reason` confirmed as `"ancestor-cycle"` (not `"depth"`) by calling the real production decision function directly with the same input `spawn-log` built.
- Two false-positive controls prove the guard does not over-refuse: same caste with a genuinely different task is allowed (`TestSpawnAncestorCheckAllowsDifferentTaskSameCaste`), and the same task text handled by a different caste is allowed (`TestSpawnAncestorCheckAllowsSameTaskDifferentCaste`).
- The chain-unreadable path fails closed (D-19): `TestSpawnAncestorCheckFailsClosedOnUnreadableTree` corrupts `spawn-tree.txt` directly on disk (a non-numeric depth field, which the shared parser skips entirely, dropping the requester's own entry) and confirms the check denies with a reason naming the chain as unreadable -- carrying its own negative control in the same function showing a readable tree with no matching ancestor allows.

## Task Commits

Each task was committed atomically:

1. **Task 1: Walk the ancestor chain and refuse a repeated caste-and-task pair** - `53c2ed87` (feat)
2. **Task 2: Red-proofs for the cycle, the false-positive control, and the normalisation boundary** - `96ae16a6` (test)

**Plan metadata:** _pending_ (docs: complete plan — added by the orchestrator after all worktree agents in this wave merge)

## Files Created/Modified

- `cmd/spawn_ancestor.go` - filled `spawnAncestorCycleReason`'s body; added `spawnAncestorChain` (single-parse, visited-set-bounded ancestor walk) and `normalizeSpawnTask` (four-rule text normaliser); removed the `CONTRACT-STUB-PLAN-06` marker.
- `cmd/spawn_ancestor_test.go` (NEW) - five tests: the A-to-B-to-A cycle proof driven through `spawn-log`, two false-positive controls, a `normalizeSpawnTask` table test with non-colliding pairs, and the fail-closed-on-unreadable-chain proof with its own negative control; plus the `spawnLogArgsWithCasteTask` helper these tests (and repaired pre-existing tests) share.
- `cmd/spawn_enforce_test.go` - repaired four pre-existing tests whose two-level fixtures reused the same fixed caste/task at both levels, which the new ancestor-cycle check now correctly refuses (see Deviations below).

## Decisions Made

- **Visited-name set over a hop-count cap** for bounding the ancestor walk -- simpler, and correct independent of `spawnMaxDelegationDepth`'s value.
- **Reason confirmation via direct function call, not CLI JSON** -- `spawn-can-spawn` has no `--caste`/`--task` flags, so it structurally cannot exercise or report on the ancestor-cycle check. `TestSpawnCanSpawnDeniesAncestorCycle` proves the CLI-level refusal (exit code, message, no write) through `spawn-log`, and separately confirms the specific `Reason` value by calling the real `spawnCanSpawnDecision` var directly -- the same production seam, not a mock, and no change to `cmd/spawn.go`.
- **Repaired, not left broken, four pre-existing tests** whose fixtures collided with the new check -- see Deviations below.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Four pre-existing tests asserted stub-era (no-ancestor-check) behaviour that the new real check correctly breaks**

- **Found during:** Task 2 verification (`go test ./cmd -run 'TestSpawn' -count=1 -v`)
- **Issue:** `TestSpawnLogDerivesDepthFromRecordedParent`, `TestSpawnTreeDepthReportsTwoForAThreeLevelTree`, `TestSpawnLogRefusesPastCapAndWritesNoEntry`, and `TestSpawnCanSpawnNameOverridesClaimedDepth` all built a two-level `Queen -> W1 -> H1` chain using the shared `spawnLogArgs` helper, which hardcodes caste `"builder"` and task `"t"` for every call. With the ancestor-cycle check now real, `W1`'s attempt to spawn `H1` with the identical caste and task `W1` was itself given is a genuine repeat and is correctly refused -- but none of these four tests are about ancestor-cycle behaviour; their subjects are depth derivation, the fixed max-depth number, the depth cap's refusal, and depth-claim override, respectively.
- **Fix:** Added `spawnLogArgsWithCasteTask(parent, name, caste, task)` in the new test file (an argument builder that, unlike `spawnLogArgs`, lets a test set caste and task independently) and used it to give `W1` and `H1` distinct task text ("coordinate the initial request" / "carry out the coordinated request") in each of the four fixtures. `TestSpawnLogRefusesPastCapAndWritesNoEntry`'s third attempt (`H1` spawning `X1`, which must still be refused by the depth cap specifically) keeps its original task `"t"`, distinct from both W1's and H1's new text, so that refusal remains a depth-cap denial and not an ancestor-cycle denial.
- **Files modified:** cmd/spawn_enforce_test.go
- **Verification:** `go test ./cmd -run 'TestSpawn' -count=1 -v` -- all 30 spawn-related tests green; full `go test ./... -count=1 -timeout 900s` passes (18 packages, 352s for `cmd`)
- **Committed in:** `96ae16a6` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug-fix class covering four collateral test repairs, Rule 1)
**Impact on plan:** Necessary, in-scope consequence of Task 1's intended behaviour change (a real ancestor-cycle check where an always-allow stub previously sat). No file outside `cmd/spawn_enforce_test.go` (a test file, not shared with the parallel 173-05 worktree's `cmd/spawn.go`/`cmd/spawn_budget.go`/`cmd/spawn_budget_test.go`) needed touching.

## Issues Encountered

None beyond the deviation above. The D-22 red-proof exercise (below) surfaced one design nuance worth recording: a naive "always-deny" substitution placed at the very top of `spawnAncestorCycleReason` also breaks the coordinator-root short-circuit, denying even the legitimate `Queen -> A1` setup spawns every fixture depends on. The substitution was placed *after* the root-sentinel early return instead, so setup spawns keep succeeding and only the actual match/no-match logic is replaced.

## Red-Proof Evidence (D-22)

**Red-proof 1 -- `spawnAncestorCycleReason` forced to return `""` unconditionally** (always-allow):

```
=== RUN   TestSpawnCanSpawnDeniesAncestorCycle
    spawn_ancestor_test.go:66: spawn-log repeating A1's own caste and task did not exit non-zero: stdout={"ok":true,"result":{"caste":"builder","claimed_depth":0,"depth":2,"depth_source":"derived","event_id":"evt_1786569370_4be6","name":"C1","parent":"A1","recorded":true,"task":"fix the login form"}}
         stderr=
--- FAIL: TestSpawnCanSpawnDeniesAncestorCycle (0.00s)
=== RUN   TestSpawnAncestorCheckAllowsDifferentTaskSameCaste
--- PASS: TestSpawnAncestorCheckAllowsDifferentTaskSameCaste (0.00s)
=== RUN   TestSpawnAncestorCheckAllowsSameTaskDifferentCaste
--- PASS: TestSpawnAncestorCheckAllowsSameTaskDifferentCaste (0.00s)
=== RUN   TestNormalizeSpawnTaskMatchesOnlyWhitespaceCaseAndTrailingStop
--- PASS: TestNormalizeSpawnTaskMatchesOnlyWhitespaceCaseAndTrailingStop (0.00s)
=== RUN   TestSpawnAncestorCheckFailsClosedOnUnreadableTree
    spawn_ancestor_test.go:247: expected an unreadable ancestor chain to deny (D-19 fail-closed), got allow
--- FAIL: TestSpawnAncestorCheckFailsClosedOnUnreadableTree (0.00s)
```

Tests 1 and 5 (the two tests that expect a deny) went red exactly as required; tests 2, 3 and 4 (which do not depend on the check ever denying) stayed green throughout -- proving the deny tests cannot be satisfied by an always-allow guard.

**Red-proof 2 -- `spawnAncestorCycleReason` forced to return a constant `"always deny"` after the coordinator-root short-circuit** (always-deny for any non-root requester):

```
=== RUN   TestSpawnCanSpawnDeniesAncestorCycle
    spawn_ancestor_test.go:71: deny message does not name the matched ancestor "A1": {"ok":false,"error":"always deny","code":1}
--- FAIL: TestSpawnCanSpawnDeniesAncestorCycle (0.00s)
=== RUN   TestSpawnAncestorCheckAllowsDifferentTaskSameCaste
    spawn_ancestor_test.go:129: [spawn-log --parent A1 --caste builder --name C1 --task add pagination to the results list --depth 0] exited 1: {"ok":false,"error":"always deny","code":1}
--- FAIL: TestSpawnAncestorCheckAllowsDifferentTaskSameCaste (0.00s)
=== RUN   TestSpawnAncestorCheckAllowsSameTaskDifferentCaste
    spawn_ancestor_test.go:152: [spawn-log --parent A1 --caste watcher --name C1 --task fix the login form --depth 0] exited 1: {"ok":false,"error":"always deny","code":1}
--- FAIL: TestSpawnAncestorCheckAllowsSameTaskDifferentCaste (0.00s)
=== RUN   TestNormalizeSpawnTaskMatchesOnlyWhitespaceCaseAndTrailingStop
--- PASS: TestNormalizeSpawnTaskMatchesOnlyWhitespaceCaseAndTrailingStop (0.00s)
=== RUN   TestSpawnAncestorCheckFailsClosedOnUnreadableTree
    spawn_ancestor_test.go:226: expected a readable tree with no matching ancestor to allow, got deny reason: "always deny"
--- FAIL: TestSpawnAncestorCheckFailsClosedOnUnreadableTree (0.00s)
```

Tests 2 and 3 (the named false-positive controls) went red as D-22 requires. Test 1 also went red here, but for a stricter reason than a plain allow/deny flip: it still correctly denied, but its message-content assertion (the deny reason must name the specific matched ancestor, `"A1"`) failed against the generic constant string -- proving the test checks *which* ancestor matched, not merely that something was denied. Test 5 also went red on its own embedded negative control, which is itself an allow-when-no-match assertion of the same shape as tests 2/3. Test 4 (`normalizeSpawnTask`, independent of this function) stayed green throughout, as expected.

The function was then restored to its Task 1 form; `git diff cmd/spawn_ancestor.go` showed zero residual changes, and all five new tests plus the four repaired pre-existing tests passed green afterward, alongside the full `go test ./... -count=1 -timeout 900s` run (18 packages, all green).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 10's marker-survival assertion has one fewer stub to check for: `CONTRACT-STUB-PLAN-06` no longer exists anywhere in `cmd/spawn_ancestor.go`.
- Both of plan 04's contract stubs are now real: plan 05 fills `cmd/spawn_budget.go`'s `spawnTreeBudgetReason` in a separate parallel worktree; this plan fills `cmd/spawn_ancestor.go`'s `spawnAncestorCycleReason`. Neither touched `cmd/spawn.go`, so both should merge cleanly.
- `spawnLogArgsWithCasteTask` in `cmd/spawn_ancestor_test.go` is available to any future test in the `cmd` package that needs a multi-level spawn-tree fixture with independently controlled caste/task text, avoiding the same false-positive collision this plan repaired in `cmd/spawn_enforce_test.go`.
- No blockers for plan 10.

## Known Stubs

None. `spawnAncestorCycleReason` is now the fully real check this plan's own objective describes; no placeholder or deferred logic remains in `cmd/spawn_ancestor.go`.

---
*Phase: 173-delegation-guard*
*Completed: 2026-08-12*

## Self-Check: PASSED

- FOUND: cmd/spawn_ancestor.go
- FOUND: cmd/spawn_ancestor_test.go
- FOUND: cmd/spawn_enforce_test.go
- FOUND: .planning/phases/173-delegation-guard/173-06-SUMMARY.md
- FOUND commit 53c2ed87 (Task 1)
- FOUND commit 96ae16a6 (Task 2)
