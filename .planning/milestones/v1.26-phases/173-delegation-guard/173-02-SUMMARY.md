---
phase: 173-delegation-guard
plan: 02
subsystem: infra
tags: [spawn-tree, cobra, cli, ci-wiring, go]

# Dependency graph
requires:
  - phase: 172-wiring-proof
    provides: the named CI wiring-gate step and its -run filter, plus the guard-file AST enumeration this plan's tests plug into
provides:
  - spawn-log derives the recorded depth from the parent's own recorded entry (coordinator sentinels Queen/Prime-1/Swarm resolve to depth 1); caller-supplied --depth is retained only as claimed_depth for reporting
  - an unresolvable --parent is refused before any write (fail-closed, D-19); nothing is recorded and spawn-tree.txt is untouched
  - .aether/workers.md corrected to a single depth convention (Coordinator 0 / Worker 1 / Helper 2) agreeing with D-01/D-05, replacing the wrong "Global Cap: 10" line with the real wave-cap-4-to-8 / tree-budget-20 pair
  - six named, execution-based red-proofs registered in the CI wiring-gate -run filter
affects: [173-04, 173-05, 173-08, 173-10]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Recorder-derives-truth: the spawn-recording call (spawn-log), not the caller, is the depth authority — a worker can lie about its depth but cannot avoid being recorded correctly"
    - "Fail-closed parent resolution: an unresolvable --parent denies and writes nothing, rather than defaulting to depth 0"

key-files:
  created: []
  modified:
    - cmd/spawn.go
    - cmd/write_cmds_test.go
    - cmd/spawn_enforce_test.go
    - .aether/workers.md
    - .github/workflows/ci.yml

key-decisions:
  - "TestSpawnLogLegacyFlagAliases (pre-existing) asserted depth 0 for a spawn naming Queen as parent — that encoded the pre-D-05 behavior this plan intentionally overrides. Updated in place to expect depth 1, with a comment pointing at D-05 and the new dedicated proof test."

patterns-established:
  - "deriveSpawnDepth(st, parent) is the single depth-derivation seam: sentinel match -> 1, recorded parent -> parent.Depth+1, otherwise deny. Later plans (173-04/05/08) should extend this seam rather than adding a second one."

requirements-completed: [SPAWN-02, SPAWN-06]

# Metrics
duration: 30min
completed: 2026-08-12
---

# Phase 173 Plan 02: Depth Derivation and Convention Correction Summary

**spawn-log now derives recorded depth from the parent's own recorded entry (or a coordinator sentinel) instead of trusting the caller's --depth, refuses unresolvable parents fail-closed with nothing written, and .aether/workers.md states the same depth convention the recorder enforces.**

## Performance

- **Duration:** ~30 min
- **Tasks:** 3 completed
- **Files modified:** 5 (cmd/spawn.go, cmd/write_cmds_test.go, cmd/spawn_enforce_test.go, .aether/workers.md, .github/workflows/ci.yml)

## Accomplishments

- `deriveSpawnDepth` is the new depth authority: a coordinator sentinel (`Queen`, `Prime-1`, `Swarm`, case-insensitive) resolves to depth 1; any other parent must already have a recorded spawn-tree entry, and the recorded depth is that entry's depth + 1; anything else is refused before `RecordSpawn` is ever called
- `spawn-log`'s JSON payload now reports `claimed_depth` (what the caller sent) alongside `depth` (what was actually recorded) and `depth_source: "derived"`, so a discrepancy is visible rather than silently overridden (T-173-10)
- `.aether/workers.md` corrected in three places per RESEARCH.md Pitfall 1's warning that fixing only the `--depth 0` examples leaves the real contradiction (the behavior table, and the wrong Global Cap number) intact
- Six new named tests, each proving one claim by execution and registered in the CI wiring-gate `-run` filter so a regression reads as a wiring problem, not one anonymous failure among hundreds

## Task Commits

Each task was committed atomically:

1. **Task 1: Derive the recorded depth from the parent's own entry** - `7c91e1ce` (feat)
2. **Task 2: Make .aether/workers.md state one depth convention** - `ba631d1c` (docs)
3. **Task 3: Red-proofs for derived depth and the corrected convention, registered in CI** - `7b21dc2b` (test)

**Plan metadata:** _pending_ (docs: complete plan — added by the orchestrator after all worktree agents in this wave merge)

## Files Created/Modified

- `cmd/spawn.go` - added `spawnRootParentNames`, `spawnParentIsRoot`, `deriveSpawnDepth`; `spawnLogCmd.RunE` now calls `deriveSpawnDepth` instead of trusting `cmd.Flags().GetInt("depth")`; extended the success payload with `claimed_depth`/`depth_source`; updated the `--depth` flag's help text to state it is advisory
- `cmd/write_cmds_test.go` - updated `TestSpawnLogLegacyFlagAliases` to expect depth 1 (not 0) for a spawn naming `Queen` as parent, matching D-05
- `cmd/spawn_enforce_test.go` - added `spawnLogArgs`/`runSpawnLogExpectingSuccess` helpers and six new tests (see Deviations/Task 3 below)
- `.aether/workers.md` - both `spawn-log` examples now send `--depth 1` with an advisory comment; the Depth-Based Behavior table rewritten to three rows (Coordinator/Worker/Helper) with the "Yes (if surprised)" depth-2 delegation permission removed; the wrong "Global Cap: 10" line replaced with the real wave-cap (4-8) and tree-budget (20) pair plus the 8-wide x 2-deep = 73 arithmetic
- `.github/workflows/ci.yml` - appended the six new test function names to the named "Verify subcommand wiring and CLI flag contracts" step's `-run` filter

## Decisions Made

- **`TestSpawnLogLegacyFlagAliases` update is in-scope, not a deviation requiring a Rule 4 pause.** The test asserted the exact pre-D-05 behavior (`--parent "Queen"` implies depth 0) that Task 1's acceptance criteria explicitly requires to change (`aether spawn-log --parent Queen ... records depth 1`). Fixing a test that encodes the old, wrong contract is Rule 1 (bug fix), and this plan's own Task 1 acceptance criteria names the new expected value directly.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated a pre-existing test asserting the pre-D-05 depth contract**
- **Found during:** Task 1 (verification run of `go test ./cmd -run 'TestSpawn'`)
- **Issue:** `TestSpawnLogLegacyFlagAliases` asserted `depth == 0` for a spawn with `--name "Queen"` as the parent alias source — the exact behavior D-05 and this plan's Task 1 require to change (Queen is now a coordinator sentinel, deriving to depth 1)
- **Fix:** Updated the assertion to `depth == 1` with a comment citing D-05 and pointing at `TestSpawnLogIgnoresCallerSuppliedDepth` for the direct proof
- **Files modified:** cmd/write_cmds_test.go
- **Verification:** `go test ./cmd -run 'TestSpawn' -count=1` passes; full `go test ./... -count=1 -timeout 900s` passes
- **Committed in:** `7c91e1ce` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix, Rule 1)
**Impact on plan:** Necessary consequence of the plan's own required behavior change. No scope creep.

## Issues Encountered

- **Test byte-length mismatch on parsing error output.** The first draft of `TestSpawnLogRefusesAnUnknownParentAndWritesNothing` parsed `buf` (stdout) for the error envelope, but `outputError` writes its JSON envelope to `stderr`, not `stdout` — confirmed by tracing `outputError` in `cmd/helpers.go` and cross-checking `TestSpawnCanSpawnEnforceDeniesWithNonZeroExit`'s existing pattern. Fixed by parsing `errBuf` instead, before the first commit (no separate deviation entry — caught during initial test-writing, not after a task was already marked done).

## Red-Proof Evidence (D-22)

Performed before Task 3's commit, per the plan's explicit instruction, then fully restored (confirmed via `git diff` showing no residual changes):

**Derivation bypass** (`cmd/spawn.go` temporarily made to record `claimedDepth` directly instead of calling `deriveSpawnDepth`):

```
=== RUN   TestSpawnLogDerivesDepthFromRecordedParent
    spawn_enforce_test.go:284: W1 recorded depth = 0, want 1
--- FAIL: TestSpawnLogDerivesDepthFromRecordedParent (0.02s)
=== RUN   TestSpawnTreeDepthReportsTwoForAThreeLevelTree
    spawn_enforce_test.go:325: max_depth = 0, want 2 (D-07: coordinator 0, worker 1, helper 2)
--- FAIL: TestSpawnTreeDepthReportsTwoForAThreeLevelTree (0.00s)
=== RUN   TestSpawnLogRefusesAnUnknownParentAndWritesNothing
    spawn_enforce_test.go:388: spawn-log with unknown parent did not exit non-zero: stdout={"ok":true,"result":{...,"parent":"Ghost-99","recorded":true,...}}
         stderr=
--- FAIL: TestSpawnLogRefusesAnUnknownParentAndWritesNothing (0.00s)
```

**Table restoration** (`.aether/workers.md`'s depth-2 row temporarily restored to `| 2 | Specialist | Yes (if surprised) | 2 | ... |`), proving `TestWorkersMdStatesOneDepthConvention` fails on the table itself, not only the `--depth 0` absence check:

```
=== RUN   TestWorkersMdStatesOneDepthConvention
    spawn_enforce_test.go:540: table row for depth 2 has a Can Spawn cell containing " Yes (if surprised) " — D-01 forbids delegation at depth 2 or deeper
--- FAIL: TestWorkersMdStatesOneDepthConvention (0.00s)
```

Both reverts were undone before committing; `git diff` against the committed state showed zero residual diff in `cmd/spawn.go` and `.aether/workers.md` after restoration, and all six tests plus the full suite (`go test ./... -count=1 -timeout 900s`) passed green afterward.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The chokepoint SPAWN-06 depends on (honest recorded depth) is now real: `spawn-tree-depth`, `spawn-tree-active`, and any future consumer of `SpawnEntry.Depth` reads truth, not a caller-asserted number.
- **Plan 04's whole-tree budget (D-02/D-03) can now build on `deriveSpawnDepth`'s parent-lookup mechanics** — the same `latestSpawnEntryByName` walk this plan uses is the natural place to also sum ancestor-chain or whole-tree counts.
- **T-173-07 (a caller naming a real but shallower ancestor to gain depth) remains open, named residue** per this plan's own `must_haves`: `deriveSpawnDepth` requires a *real* recorded parent, but cannot verify the caller is honestly identifying itself as that parent's child. Tracked for `173-RESIDUE.md` (plan 10).
- `spawnCanSpawnDecision` (the separate `spawn-can-spawn` allow/deny stub) is untouched by this plan, as scoped — plans 04/05/08 own it.

---
*Phase: 173-delegation-guard*
*Completed: 2026-08-12*

## Self-Check: PASSED

- FOUND: cmd/spawn.go
- FOUND: cmd/write_cmds_test.go (modified, verified via commit diff)
- FOUND: cmd/spawn_enforce_test.go
- FOUND: .aether/workers.md
- FOUND: .github/workflows/ci.yml
- FOUND: .planning/phases/173-delegation-guard/173-02-SUMMARY.md
- FOUND commit 7c91e1ce (Task 1)
- FOUND commit ba631d1c (Task 2)
- FOUND commit 7b21dc2b (Task 3)
