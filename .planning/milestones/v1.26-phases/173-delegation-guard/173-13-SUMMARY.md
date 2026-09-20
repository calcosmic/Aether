---
phase: 173-delegation-guard
plan: 13
subsystem: agent
tags: [go, spawn-tree, spawn-budget, fail-closed, gap-closure, documentation]

# Dependency graph
requires:
  - phase: 173-delegation-guard
    provides: "plan 173-11's parse-boundary contract (absent vs. corrupt vs. unreadable) and plan 173-12's budget check that turns those errors into denies -- this plan proves every guard actually uses them, and corrects the five comments that used to say the underlying bug was impossible to trigger"
provides:
  - "three new ledger-corrupt fault-injection subtests (spawn-can-spawn, spawn-log, spawn-can-spawn-swarm) proving each guard denies a present-but-unparseable spawn ledger through its own deny path, not just the one 173-VERIFICATION.md's exploit used"
  - "one new run-state-obstructed subtest on spawn-log, proving the recorder itself refuses when the run record's path is blocked by a directory -- the second, tampering-free route to the same exploit"
  - "five corrected, dated comments (four in cmd/spawn_failclosed_test.go, one in cmd/internal_cmds.go) that no longer claim agent.SpawnTree.Parse() can never return an error"
  - "a rewritten 173-RESIDUE.md that records the exploit, the second route found while planning the fix, which three plans closed both, and four new named residues (ledger forgery, the un-gated direct-recorder call sites, the silent operator view, and the concurrent race) that remain open"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A cross-guard fault table proves breadth (every guard denies on the fault it is actually subject to) separately from depth (one guard's own edge cases) -- new fault axes belong in the table, not as one-off tests, so a future fifth guard is caught by the same coverage check"
    - "A comment asserting 'X can never happen' is itself a claim that needs a runnable proof or a removal date -- this plan corrected five such comments after the code they described changed underneath them"

key-files:
  created: []
  modified:
    - cmd/spawn_failclosed_test.go
    - cmd/internal_cmds.go
    - .planning/phases/173-delegation-guard/173-RESIDUE.md

key-decisions:
  - "The three ledger-corrupt axes inject the identical fault (one line of non-pipe-format text over spawn-tree.txt) rather than three different corruptions, so the table proves the same real-world exploit reaches a deny through three different guards' own code paths, not three unrelated faults"
  - "run-state-obstructed deliberately leaves spawn-tree.txt absent (not corrupted), because its whole point is proving spawn-log refuses when the LEDGER is fine and the RUN RECORD is not -- the opposite combination from the ledger-corrupt axes"
  - "Corrected comments keep the historical claim visible (dated, marked as true-before/false-after) rather than deleting it, matching this repo's own stated preference for narrowing over erasing prior reasoning"

requirements-completed: [SPAWN-03, SPAWN-04]

# Metrics
duration: ~55min
completed: 2026-08-13
---

# Phase 173 Plan 13: Cross-Guard Proof and Residue Correction for the Ledger Fail-Closed Fix Summary

**Added four new fault-injection subtests proving every guard that reads the spawn ledger or the run record now refuses a tampered or blocked one through its own code path, corrected five now-false "this can never happen" comments, and rewrote the phase's own written record to say what was actually found and fixed instead of what was predicted.**

## Performance

- **Duration:** ~55 min
- **Completed:** 2026-08-13T16:53:20Z
- **Tasks:** 2/2
- **Files modified:** 3

## Accomplishments

- `spawn-can-spawn`, `spawn-log`, and `spawn-can-spawn-swarm` each gained a `ledger-corrupt` subtest proving they deny a present-but-unparseable `spawn-tree.txt` through their own deny path -- not merely through the one path 173-VERIFICATION.md's exploit happened to use
- `spawn-log` gained a `run-state-obstructed` subtest, proving the recorder itself (the one guard an LLM cannot route around) refuses when `spawn-runs.json`'s path is blocked, even though `spawn-tree.txt` is perfectly valid -- the second, tampering-free route found while planning plan 173-12's fix
- Five comments across two files that claimed `agent.SpawnTree.Parse()` could never return an error were corrected in place, each dated and naming the plan that made the old claim false, with the history kept visible rather than deleted
- `.planning/phases/173-delegation-guard/173-RESIDUE.md` was rewritten so its own guard/axis table, its criterion-4 verdict, and its list of open limits all describe what verification and this gap closure actually found -- including a limit (ledger forgery) and an unguarded path (direct spawn recorders that bypass the budget check entirely) that had never been written down as their own named residues before this plan
- Two red-proofs were performed by hand: reverting plan 173-11's corrupt-content branch to a nil error makes the `ledger-corrupt` subtests fail by name (both `spawn-can-spawn` and `spawn-can-spawn-swarm`); reverting plan 173-11's `loadRunStateLocked` fix makes `run-state-obstructed` fail by name. Both were restored and confirmed byte-identical to the committed file afterward.
- Full repository test suite (`go test ./... -count=1 -timeout 900s`) passes, all 20 packages

## Task Commits

Each task was committed atomically:

1. **Task 1: Prove the unreadable inputs across every guard, and delete the claim that it was impossible** - `d7d91d5d` (test)
2. **Task 2: Rewrite the residue record to match what is now true** - `49c6f31c` (docs)

**Plan metadata:** (this commit, made by the orchestrator after all worktree agents in the wave complete)

## For the Owner, in Plain English

This phase built several checks that are supposed to stop an AI helper from creating too many other helpers in one run -- a limit of 20. When we actually tested it (in the step called "verification"), we found a way to break that limit: overwriting one internal record file with a single line of garbage text reset the count to zero, and the 21st helper (which should have been refused) was allowed. The last two plans fixed that. While planning the fix, we also found a second, even easier way to get the same broken result -- just deleting a different internal file, with no tampering needed at all -- and that got fixed too.

This plan's job was to prove the fix actually works everywhere it needs to, not just in the one place the original test used, and to fix five comments in the code that used to say "this kind of bug is impossible" -- which was true when they were written, and is no longer true now that the underlying bug is fixed (fixing it necessarily means the code can now detect the very thing those comments said it couldn't).

What this proves: every one of the three checks that reads the helper-count record now correctly refuses when that record is broken or blocked, whichever of the two ways it's broken. What is still not guaranteed, and is now written down plainly rather than left implicit: someone could still hand-write a fake-but-valid-looking record that quietly omits real helpers, and nothing in this system would catch that -- catching it would need a different kind of record-keeping (like a tamper-evident log), which is a bigger change saved for later. There is also a coordinator dispatch path that writes helper records directly without going through the same check -- those helpers are still counted afterward, but not asked permission first. Both of these limits are now named clearly in the project's own notes so nobody mistakenly assumes they were already handled.

## The Five Corrected Statements

All five previously asserted that `agent.SpawnTree.Parse()` (the code that reads the spawn ledger) could never return an error for any byte content -- true before plan 173-11, false after it. Each correction below is dated 2026-08-13 and names plan 173-11 as the change that invalidated the original claim.

### 1. `cmd/spawn_failclosed_test.go` — file-top "Bounded residue" note

**Before:**
> Bounded residue, discovered while writing these tests: agent.SpawnTree's Parse() (pkg/agent/spawn_tree.go) treats a store.ReadFile error on spawn-tree.txt identically to a missing file -- it swallows the error inside parseFile() and always returns a nil error, and malformed pipe-delimited lines are silently skipped rather than surfaced as an error. Parse() itself can therefore never return a non-nil error, and no byte content exists that makes it do so. cmd/internal_cmds.go's Task 1 fix works around this by checking existence and type (os.Stat) ahead of calling Parse(), which is what TestSpawnCanSpawnSwarmFailsClosedOnUnreadableSpawnTree exercises below: it makes the tree path genuinely unreadable by putting a directory there (store.ReadFile / os.ReadFile fail on a directory), which is a real input Parse()'s caller rejects, even though Parse() would not reject it itself.

**After:**
> Bounded residue, discovered while writing these tests, and corrected 2026-08-13 by plan 173-11: at the time these tests were first written, agent.SpawnTree's Parse() (pkg/agent/spawn_tree.go) treated a store.ReadFile error on spawn-tree.txt the same as a missing file -- it swallowed the error inside parseFile() and always returned a nil error, and a malformed pipe-delimited line was silently skipped rather than surfaced as an error. No byte content could make Parse() return an error at that time. cmd/internal_cmds.go's Task 1 fix worked around this by checking existence and type (os.Stat) ahead of calling Parse(), which is what TestSpawnCanSpawnSwarmFailsClosedOnUnreadableSpawnTree exercises below: it makes the tree path unreadable by putting a directory there (store.ReadFile / os.ReadFile fail on a directory), a real input Parse()'s caller rejected even though Parse() would have accepted it.
>
> What changed in plan 173-11: parseFile() now draws a three-way distinction instead of collapsing everything into "empty, no error". An absent file, a zero-byte file, or a whitespace-only file still parse to empty with a nil error -- the one narrow, justified exception (T-173-24), so a fresh colony is never locked out of spawning. Every other outcome -- a read failure that is not "file does not exist" (a directory at the path, a permission denial), or content that does not parse as valid pipe-delimited spawn-tree format -- now returns a non-nil error wrapping agent.ErrSpawnTreeCorrupt. The identical three-way distinction now applies to spawn-runs.json (loadRunStateLocked). The "ledger-corrupt" axes added by plan 173-13 below, on every guard that reads spawn-tree.txt, and the "run-state-obstructed" axis on spawn-log, are the proof that each guard actually turns this new error into a deny.

### 2. `cmd/spawn_failclosed_test.go` — `TestSpawnCanSpawnSwarmFailsClosedOnUnreadableSpawnTree` doc comment

**Before:**
> TestSpawnCanSpawnSwarmFailsClosedOnUnreadableSpawnTree is the T-173-13 assertion: an unreadable spawn-tree.txt must deny rather than be counted as zero live spawns with the full budget free. See the bounded-residue note above the file's top: this test makes the tree unreadable by putting a directory at its path, since agent.SpawnTree.Parse() cannot itself be made to return an error.

**After:**
> TestSpawnCanSpawnSwarmFailsClosedOnUnreadableSpawnTree is the T-173-13 assertion: an unreadable spawn-tree.txt must deny rather than be counted as zero live spawns with the full budget free. This test makes the tree unreadable by putting a directory at its path.
>
> Corrected 2026-08-13 (plan 173-11 gave agent.SpawnTree.Parse() a real error return): before 173-11, the directory route was the ONLY fault this guard could be exercised against, because no byte content written over spawn-tree.txt made Parse() return an error. The directory route and a corrupted-content route are now two distinct, separately exercised faults -- this test proves the directory route (an os.Stat rejection ahead of Parse()); the "ledger-corrupt" axis on this same command in delegationGuardFaultTable below proves the content route (Parse() itself rejecting a present-but-unparseable file).

### 3. `cmd/spawn_failclosed_test.go` — `delegationGuardFaultAxis.NotExercisable` doc comment

**Before:**
> NotExercisable, when non-empty, states why this axis cannot currently be injected (e.g. agent.SpawnTree.Parse() can never itself return an error -- see the bounded-residue note atop this file). The axis is recorded with its reason rather than deleted, so TestDelegationGuardTableCoversEveryGuardCommand can see it was considered rather than silently dropped. Verify must be nil when this is set.

**After:**
> NotExercisable, when non-empty, states why this axis cannot currently be injected -- for example, a fault whose only known trigger requires production code this repository does not have yet. The axis is recorded with its reason rather than deleted, so TestDelegationGuardTableCoversEveryGuardCommand can see it was considered rather than silently dropped. Verify must be nil when this is set.
>
> Corrected 2026-08-13 (plan 173-13): this comment's worked example used to assert a specific claim about agent.SpawnTree.Parse()'s runtime behaviour. Plan 173-11 made that claim false, and an example that asserts current runtime behaviour is exactly how the drift happened -- described further in the file-top note above. The example here is now abstract on purpose.

### 4. `cmd/spawn_failclosed_test.go` — `spawn-tree-unreadable` axis comment

**Before:**
> A directory at spawn-tree.txt's path is a real input this guard's cmd/internal_cmds.go os.Stat check rejects ahead of calling Parse() -- Parse() itself cannot be made to error (see the bounded-residue note atop this file), so this axis is exercisable via the directory route, not via Parse() rejecting content.

**After:**
> A directory at spawn-tree.txt's path is a real input this guard's cmd/internal_cmds.go os.Stat check rejects ahead of calling Parse().
>
> Corrected 2026-08-13 (plan 173-11 gave Parse() a real error return): the directory route and a corrupted-content route are now two distinct faults, both exercised in this table -- this axis proves the directory route; the "ledger-corrupt" axis above proves the content route, which Parse() would have accepted as an empty tree before 173-11.

### 5. `cmd/internal_cmds.go` — comment justifying `spawn-can-spawn-swarm`'s pre-`Parse()` `os.Stat` check

**Before:**
> agent.SpawnTree.Parse() swallows a missing file as "empty" (nil error) exactly like an unreadable one -- it cannot tell the two apart. A fresh colony with no spawn-tree.txt yet must still count zero and be allowed to spawn, so existence and type are checked here, ahead of Parse(), rather than trusting its error return.

**After:**
> Before plan 173-11, agent.SpawnTree.Parse() treated a missing file as "empty" (nil error) the same way it treated an unreadable one -- it had no way to distinguish the two. Corrected 2026-08-13: Parse() now returns a distinct, non-nil error whenever spawn-tree.txt exists but cannot be read or does not parse as valid spawn-tree content; only the file's genuine absence (or zero/whitespace-only content) still parses to empty. This os.Stat check stays anyway -- it is a cheap, explicit check that names its own "tree-unreadable" reason before Parse() ever runs, not a workaround for something Parse() itself now handles correctly. A fresh colony with no spawn-tree.txt yet must still count zero and be allowed to spawn, which the os.IsNotExist branch below still guarantees.

## The Repo-Wide Stale-Claim Grep, Proving All Five Are Gone

```
$ grep -rn --include="*.go" 'never return a non-nil error\|cannot itself be made to return\|can never itself return\|cannot be made to error\|cannot tell the two apart\|swallows a missing file\|skipped entirely by parseSpawnTreeBytes' .
```

Returns nothing (exit code 1 -- no match found anywhere in the repository, including the sixth phrase from plan 11's already-corrected `spawn_ancestor_test.go` fixture comment).

## Verbose Test Output: The Three `ledger-corrupt` Subtests and `run-state-obstructed`

```
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn/run-state-unreadable
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn/ledger-corrupt
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn/requester-not-recorded
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-log/run-state-unreadable
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-log/ledger-corrupt
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-log/run-state-obstructed
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-log/parent-not-recorded
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn-swarm/colony-state-unreadable
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn-swarm/ledger-corrupt
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn-swarm/spawn-tree-unreadable
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/hook-pre-tool-use/requester-identity-unresolvable
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/control/spawn-can-spawn_allows_when_everything_is_valid
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/control/spawn-log_allows_when_everything_is_valid
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/control/spawn-can-spawn-swarm_allows_when_everything_is_valid
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/control/hook-pre-tool-use_allows_when_everything_is_valid
--- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs (0.02s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn/run-state-unreadable (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn/ledger-corrupt (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn/requester-not-recorded (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-log/run-state-unreadable (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-log/ledger-corrupt (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-log/run-state-obstructed (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-log/parent-not-recorded (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn-swarm/colony-state-unreadable (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn-swarm/ledger-corrupt (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn-swarm/spawn-tree-unreadable (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/hook-pre-tool-use/requester-identity-unresolvable (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/control/spawn-can-spawn_allows_when_everything_is_valid (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/control/spawn-log_allows_when_everything_is_valid (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/control/spawn-can-spawn-swarm_allows_when_everything_is_valid (0.00s)
    --- PASS: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/control/hook-pre-tool-use_allows_when_everything_is_valid (0.00s)
=== RUN   TestDelegationGuardTableCoversEveryGuardCommand
--- PASS: TestDelegationGuardTableCoversEveryGuardCommand (0.00s)
PASS
ok  	github.com/calcosmic/Aether/cmd	0.629s
```

## Red-Proof Transcripts

Both performed by mutating `pkg/agent/spawn_tree.go` in place (a copy was saved to a scratch path first), confirming the target subtest fails by name, then restoring from the pre-mutation copy and confirming `git diff --stat pkg/agent/spawn_tree.go` shows zero diff.

### 1. Plan 11's corrupt-content branch reverted to a nil error

**Mutation:** `parseFile`'s corrupt-content branch changed from `return nil, nil, fmt.Errorf("spawn_tree: %s: %w", st.filePath, err)` to `return nil, nil, nil`.

**Result:**
```
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn/ledger-corrupt
    spawn_failclosed_test.go:356: spawn-can-spawn --enforce against a corrupted ledger did not exit non-zero: stdout={"ok":true,"result":{"authoritative":false,"can_spawn":true,"depth":0}}
         stderr=
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn-swarm/ledger-corrupt
    spawn_failclosed_test.go:723: spawn-can-spawn-swarm against a corrupted ledger did not exit non-zero: stdout={"ok":true,"result":{"can_spawn":true,"current_spawns":0,"max_budget":5,"remaining_budget":5}}
         stderr=
--- FAIL: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs (0.01s)
    --- FAIL: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn/ledger-corrupt (0.00s)
    --- FAIL: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-can-spawn-swarm/ledger-corrupt (0.00s)
FAIL
```
Both `ledger-corrupt` subtests that depend on `Parse()`'s error return fail by name (the `spawn-log` `ledger-corrupt` subtest also depends on this and would fail identically, since it reaches the same `spawnTreeBudgetReason` path). Restored; `git diff --stat pkg/agent/spawn_tree.go` showed zero diff, and all subtests passed again.

### 2. Plan 11's `loadRunStateLocked` collapsed condition restored

**Mutation:** replaced the split read/absent/empty branches with the original collapsed condition: `if err != nil || len(strings.TrimSpace(string(data))) == 0 { return spawnRunState{}, nil }`.

**Result:**
```
=== RUN   TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-log/run-state-obstructed
    spawn_failclosed_test.go:569: spawn-log against an obstructed run record did not exit non-zero: stdout={"ok":true,"result":{"budget_consumed":1,"budget_max":20,"caste":"builder","claimed_depth":0,"depth":1,"depth_source":"derived","event_id":"evt_1786639357_2bd5","name":"W1","parent":"Queen","recorded":true,"task":"t"}}
         stderr=
--- FAIL: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs (0.00s)
    --- FAIL: TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs/spawn-log/run-state-obstructed (0.00s)
FAIL
```
This reproduces the exact second-route bypass: a directory obstructing `spawn-runs.json` silently reads as "no runs recorded" and the spawn is allowed and recorded. Restored; `git diff --stat pkg/agent/spawn_tree.go` showed zero diff, and all subtests passed again.

## Files Created/Modified

- `cmd/spawn_failclosed_test.go` -- three `ledger-corrupt` axes added (spawn-can-spawn, spawn-log, spawn-can-spawn-swarm), one `run-state-obstructed` axis added (spawn-log), four comments corrected
- `cmd/internal_cmds.go` -- one comment corrected (no logic change; `git diff` shows only lines beginning with `//` after indentation)
- `.planning/phases/173-delegation-guard/173-RESIDUE.md` -- residue 7's table gained four rows and one corrected row; the criterion-4 closing paragraph replaced with a dated, sequenced record; four new numbered residues (10-13) added

## Decisions Made

- Kept the same fault byte-for-byte (`"garbage not pipe format"`, one line of non-pipe-format text) across all three `ledger-corrupt` axes, so the table proves one real exploit reaches a deny through three different guards' own code, rather than three unrelated fabricated faults
- Left `hook-pre-tool-use`'s axis list untouched, since it reads no file and asserting a file fault against it would be the exact mistake this table's own comments refuse to make
- Corrected comments preserve the historical (now-false) claim, dated, rather than silently deleting it -- matching the "narrower true claim, not erased reasoning" approach this repo's own audits (CLAUDE.md's Definition of Done) require

## Deviations from Plan

None -- plan executed exactly as written. All acceptance-criteria greps, both automated verification commands, both red-proofs, and the full repository test suite all passed as specified.

## Issues Encountered

A `git index.lock` file transiently blocked the first `git add` for Task 1's commit, created by a concurrent shell-status process (gitstatusd) unrelated to this plan. It had already released the lock by the time it was inspected; retrying the `git add` succeeded immediately. No production code or test was affected.

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness

- ROADMAP Phase 173 success criterion 4 ("every guard fails closed") is now proven for both routes 173-VERIFICATION.md and this phase's own planning found to defeat the whole-run spawn budget, with the one remaining bound (ledger forgery, T-173-67) named plainly in `173-RESIDUE.md` residue 10 rather than left implicit.
- Two new residues (11 and 12) name real, un-fixed gaps for a future phase to pick up if prioritized: direct spawn recorders (`recordCodexBuildDispatches` and its five siblings) bypass the whole-run budget check entirely, and the live operator view goes silent rather than erroring on a tampered ledger. Neither was in this gap closure's scope.
- WR-05 (the concurrent check-then-act race) remains open, now named as residue 13 alongside the other open items rather than only in `173-VERIFICATION.md`'s findings.

---
*Phase: 173-delegation-guard*
*Completed: 2026-08-13*

## Self-Check: PASSED

- FOUND: cmd/spawn_failclosed_test.go
- FOUND: cmd/internal_cmds.go
- FOUND: .planning/phases/173-delegation-guard/173-RESIDUE.md
- FOUND commit: d7d91d5d (test(173-13): prove every ledger-reading guard denies a corrupted or obstructed input)
- FOUND commit: 49c6f31c (docs(173-13): rewrite residue record with both closed reset routes)
