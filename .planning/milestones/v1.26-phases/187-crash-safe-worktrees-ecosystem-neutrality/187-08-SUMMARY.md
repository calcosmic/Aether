---
phase: 187-crash-safe-worktrees-ecosystem-neutrality
plan: 08
subsystem: worktree-safety
tags: [go, git, worktree, crash-safety, verification-remediation, ast-guard]

# Dependency graph
requires:
  - phase: 187-crash-safe-worktrees-ecosystem-neutrality
    provides: worktreeDestructionSafety, preserveWorktreeWork, reportWorktreePreservation, scanUnrecordedWorktrees (plans 01-07)
provides:
  - branchMergeSafety (cmd/worktree_safety.go) — mergedness-only safety check for a bare branch with no worktree directory to inspect
  - repairDirtyWorktree's "Orphan branch" case (cmd/recover_repair.go) gated on branchMergeSafety before `git branch -D` (GAP-4)
  - clearActiveColonyRuntimeFiles (cmd/entomb_cmd.go) scans the worktrees directory via scanUnrecordedWorktrees before wiping (GAP-5)
  - abandon's confirmation preview (cmd/abandon_cmd.go) states plainly when worker workspaces hold unsaved/unmerged work
  - TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate (cmd/worktree_destruction_reachability_test.go) — property guard covering every command actually registered on rootCmd, root set derived from real AddCommand(...) calls, not a hand-maintained list
  - guard-clause fallthrough detection in the AST walker (ifChainAlwaysTerminates/blockAlwaysTerminates) so `if !safety.Safe { ...; return }` followed by the destructive call is recognized as gated
affects: [recover, abandon, entomb, worktree-cleanup, worktree-merge-back, worktree-reap, any future registered command, cmd/worktree_destruction_reachability_test.go]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A safety check written for 'does this worktree directory hold work' cannot answer 'does this bare branch name hold work' — worktreeDestructionSafety's Step 2 short-circuits Safe=true for a nonexistent path, which is correct for a worktree but wrong for a branch that was never checked out into one. branchMergeSafety factors out just the mergedness check so a caller with no directory to inspect still gets real protection."
    - "'Excluded from an automatic-lifecycle reachability guard' and 'verified safe' are different claims. Every one of GAP-1 through GAP-5 lived in a command the narrow guard correctly excluded (human-typed, not automatic) — exclusion was never a safety claim, but for five commands running in production it was read as one until a structural guard covered that space too."
    - "An AST gating detector that only recognizes if/else-if nesting will flag Go's own idiomatic guard-clause shape (`if !cond { ...; return }` followed by unguarded code) as a violation, because the code after the guard clause is never literally nested inside an `if`. The fix is recognizing when an if-chain's every body path terminates (return/continue/break) — control can only reach what follows when the condition was false, which is exactly what 'gated' means."
    - "Deriving a reachability guard's root set from real `.AddCommand(...)` call sites (plus the `[]*cobra.Command{...}` registration-slice idiom this codebase uses in six files) instead of a hand-maintained command list means a command added tomorrow is policed the day it is registered, with no one needing to remember to update a list."

key-files:
  created: []
  modified:
    - cmd/recover_repair.go
    - cmd/worktree_safety.go
    - cmd/entomb_cmd.go
    - cmd/abandon_cmd.go
    - cmd/worktree_destruction_reachability_test.go
    - cmd/worktree_operator_destruction_test.go

key-decisions:
  - "GAP-4: factored the unmerged-commit check out of worktreeDestructionSafety into a new branchMergeSafety(root, branch) rather than writing a second, parallel mergedness check — worktreeDestructionSafety's Step 3.5/4/5 now simply delegate to it, so there is exactly one implementation of 'is this branch merged' in the codebase."
  - "GAP-4: repairDirtyWorktree's orphan-branch case also gained `-C root` on its `git branch -D` call, fixing a second, independent latent bug the pre-fix code had (it ran in the test process's working directory, not the repo, so the delete silently targeted the wrong directory in the fail-then-pass proof) — documented as an incidental find, not a scope expansion, since it's the same three-line call site."
  - "GAP-5: fixed the destructive worktrees-directory wipe exactly once, in the shared clearActiveColonyRuntimeFiles helper both entomb and abandon call, rather than duplicating the scan-then-wipe logic in each caller."
  - "GAP-5: abandon's preview computes worker_workspaces_with_work BEFORE the colony state reset (state.Worktrees is still populated at that point), reusing worktreeDestructionSafety for tracked entries and scanUnrecordedWorktrees for anything on disk but not yet recorded — the same two-source check GAP-3 established for init."
  - "Guard extension: added a SEPARATE sanctionedUngatedDestructionFuncs map for the new all-commands test rather than sharing the narrow guard's `sanctioned` map, so a future change to one guard's exception set cannot silently loosen the other."
  - "Guard extension: the AST walker's guard-clause detection (ifChainAlwaysTerminates) intentionally does NOT attempt general control-flow analysis (no panic detection, no exhaustive-switch reasoning) — it only recognizes the return/continue/break-terminated shape this codebase's own worktree-destruction fixes actually use, which is enough to eliminate every false positive found without inventing analysis nothing in the codebase needs yet."
  - "Guard extension: worktree-reap is NOT given a blanket reachability exemption in the new all-commands test (unlike the narrow guard's operatorOnlyDestructionCommands isolation check) — instead its own destruction site is walked and asserted to be `.Safe`-gated like every other site, which is the actual invariant D-01 requires and a stronger claim than 'unreachable'."

requirements-completed: []

# Metrics
duration: ~140min
completed: 2026-08-19
---

# Phase 187 Plan 08: Close recover/abandon Worktree-Destruction Gaps, Extend the Guard to Every Command Summary

**Closed the two remaining GAP-4/GAP-5 destruction paths (`recover --apply`'s unguarded orphan-branch delete, `entomb`/`abandon`'s shared unguarded worktrees-directory wipe) and extended the AST reachability guard from a seven-command hand-maintained list to every command actually registered on rootCmd, fixing a guard-clause detection blind spot along the way that would have produced five false positives.**

## Performance

- **Duration:** ~140 min
- **Completed:** 2026-08-19
- **Files modified:** 6

## What This Closes

187-VERIFICATION.md (status: gaps_found, 8/11) named two remaining live
destruction paths after plan 07 closed GAP-1/2/3, plus asked for the
property guard to be extended so a fourth review sweep would not be needed:

- **GAP-4** — `aether recover --apply`'s "Orphan branch" repair case ran
  `git branch -D <branchName>` unconditionally. `reportOrphanBranches`
  (cmd/worktree.go) labels a branch "orphan" purely by name pattern and
  absence from state/disk — it never checks whether the branch's commits are
  merged, so the label itself was misleading, and the repair function trusted
  it without an independent check.
- **GAP-5** — `entomb` and `abandon` share `clearActiveColonyRuntimeFiles`,
  which unconditionally `os.RemoveAll`'d `.aether/worktrees`. Both callers
  reset `COLONY_STATE.json`'s `Worktrees` field to nil before calling it, so
  a state-only check would always see "nothing tracked" — the same crash-
  window blind spot GAP-3 closed for `init`, never applied here. `abandon` is
  the higher-risk half: unlike `entomb` (gated behind a sealed colony),
  `abandon --confirm` can run at any point mid-build, and its preview never
  mentioned worktrees at all.
- **Guard extension** — the existing AST reachability guard
  (`TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate`) only
  ever starts its BFS from a hand-maintained list of seven automatic entry
  points. Every one of the five defects found across GAP-1 through GAP-5 lived
  in an operator-TYPED command — a category that guard does not police at
  all, by design. Three successive review sweeps each found more, by hand.

## Accomplishments

**GAP-4 (`cmd/recover_repair.go` + `cmd/worktree_safety.go`):** Factored the
unmerged-commit check out of `worktreeDestructionSafety` into a new
`branchMergeSafety(root, branch)` — the existing function's Step 2 short-
circuits `Safe=true` for a path that doesn't exist on disk, which is correct
for a worktree directory but wrong for a bare branch name that was never
checked out into one at all (exactly the "orphan branch" shape). The
orphan-branch case now computes `branchMergeSafety` before deleting: unsafe
means the branch is preserved and reported via the same
`reportWorktreePreservation` used everywhere else in this phase; safe means
the delete proceeds, now correctly scoped to the actual repo (`-C root`,
fixing a second, independent latent bug the unguarded version also had).

**GAP-5 (`cmd/entomb_cmd.go` + `cmd/abandon_cmd.go`):** `clearActiveColonyRuntimeFiles`
now calls `scanUnrecordedWorktrees` against the worktrees directory before
its `RemoveAll`, exactly like `init`'s GAP-3 fix — this covers both callers
(`entomb`, `abandon`) with one change. `abandon`'s confirmation preview now
computes `worker_workspaces_with_work` before the colony reset (while
`state.Worktrees` is still populated) and states plainly in both the
structured JSON and the rendered plain-language preview when worker
workspaces hold unsaved or unmerged work — a destructive confirmation that
hides what it is about to destroy is not informed consent.

**Guard extension (`cmd/worktree_destruction_reachability_test.go`):** Added
`TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate`, a second property
guard whose BFS root set is derived from every command actually registered
on `rootCmd` — collected by walking every `.AddCommand(...)` call site and
every `[]*cobra.Command{...}` registration-slice literal (the loop-registration
idiom `clash.go`, `worktree.go`, `autopilot.go`, `immune.go`, `queen.go`, and
`swarm.go` all use), resolved back to each command's Cobra literal via a new
`identToCobraNode` map. This is a strict superset of the narrow guard's
seven-command lifecycle list (asserted directly in the test), so a command
registered tomorrow is checked the day it is registered, with no list to
remember to update. Fixed a real detection gap in the AST walker discovered
while extending it: the walker only recognized destructive calls literally
nested inside an `if`/`else-if` chain as "gated" — it did not recognize Go's
own idiomatic guard-clause shape (`if !safety.Safe { ...; return }` followed
by the unguarded destructive call), which is exactly the shape every
production fix in this phase actually uses. Added
`ifChainAlwaysTerminates`/`blockAlwaysTerminates` to recognize when an
if-chain's every body path terminates (return/continue/break), which makes
everything after it in the same statement list implicitly gated on the
condition's negation. This eliminated 5 of 6 initial false positives from the
new guard without touching any production code; the 6th (`worktree-allocate`'s
own same-call rollback of a worktree it just created) is the identical shape
already sanctioned for `allocateBuildWorktree` in the narrow guard, added to
a separate `sanctionedUngatedDestructionFuncs` map with matching
justification.

## Proof: Fail-Then-Pass, Actually Observed

Per this repo's Definition of Done, each fix below was run against the
unfixed code first (`git stash push` on the isolated production files,
re-run the corresponding real-git test, observe the failure, `git stash
pop`), and the failure quoted below was directly observed before the fix was
restored.

**GAP-4 — `TestRecoverApplyPreservesUnmergedOrphanBranch`:**

Pre-fix, the test failed for a slightly different reason than expected —
the unguarded `git branch -D` (with no `-C root`) ran in the wrong
directory and errored "branch not found" rather than silently succeeding,
but the assertion it actually failed on was:

```
worktree_operator_destruction_test.go:631: expected a plain-language report
that the branch was kept, got: [...]
── Repair Log ──
  [FAILED]  (phase-3/orphan-unmerged): git branch -D failed: exit status 1:
  error: branch 'phase-3/orphan-unmerged' not found
```

This confirms two independent pre-fix defects at the same call site: zero
mergedness check (the real GAP-4), and a missing `-C root` that meant the
unguarded delete was targeting the wrong working directory entirely. Both
are fixed by the same three-line change.

**GAP-4 — `TestRecoverApplyDeletesTrulyMergedOrphanBranch` (positive case):**

```
worktree_operator_destruction_test.go:722: expected a truly merged orphan
branch to be deleted by `recover --apply`, but it still exists:
"  phase-4/orphan-merged\n"
```

(Same missing-`-C root` defect — the branch that SHOULD have been deleted
also silently survived, for the wrong reason, pre-fix. Post-fix it is
correctly deleted.)

**GAP-5 — `TestAbandonPreservesUnrecordedWorktreeWithUncommittedWork`:**

```
worktree_operator_destruction_test.go:829: expected the worker workspace's
uncommitted file to survive `abandon --confirm`, but it is gone: stat
.../.aether/worktrees/phase-2-builder-abandoned/mid-build-notes.txt: no such
file or directory
```

**GAP-5 — `TestAbandonPreviewMentionsWorkerWorkspacesWithWork`:**

```
worktree_operator_destruction_test.go:1000: expected worker_workspaces_with_work
>= 1 in the preview, got <nil> (full output: {"ok":true,"result":{...no
worker_workspaces_with_work key at all...}})
worktree_operator_destruction_test.go:1009: expected the rendered preview to
mention worker workspaces holding unsaved/unmerged work, got: [...preview with
no mention of worktrees at all...]
```

Companion positive-path tests confirm the fixes did not turn either command
into a permanent no-op: `TestAbandonStillClearsWorktreesDirectoryWhenTrulyEmpty`
(a worktrees directory holding no real work is still cleared) passed
unchanged both before and after the GAP-5 fix, as expected — it was never
broken by the bug.

**Guard extension — before the gating-detection fix, `go test -run
TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate -v` failed with 6
false-positive violations** (`worktree-cleanup`, `repairDirtyWorktree`,
`worktree-allocate`, `worktree-merge-back` x2, `worktree-reap`) — all six
call sites are correctly gated in production via the guard-clause idiom;
the walker simply didn't recognize that shape yet. After adding
`ifChainAlwaysTerminates`/`blockAlwaysTerminates`, 5 of the 6 resolved with
zero production code changes; the 6th (`worktree-allocate`'s own-call
rollback) was added to `sanctionedUngatedDestructionFuncs` with the same
justification the narrow guard already gives `allocateBuildWorktree`.

**Guard extension — demonstrating it catches what the narrow guard cannot
(the core proof requirement for this plan):** temporarily reintroduced
GAP-4's exact original unguarded shape in `cmd/recover_repair.go` (bare
`git branch -D` with no safety check), then ran both guards:

```
$ go test ./cmd/... -run TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate -v
--- PASS: TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate (0.11s)

$ go test ./cmd/... -run TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate -v
    worktree_destruction_reachability_test.go:1079: found 1 unguarded worktree
    destruction site(s) reachable from a registered command:
    cmd/recover_repair.go:660 — function repairDirtyWorktree is reachable
    from a registered Aether command and calls git branch -D/-d (direct
    exec), but that call is not nested inside a condition that checks a
    worktreeDestructionSafety verdict's .Safe field. [...]
--- FAIL: TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate (0.11s)
```

The narrow guard passes (`recover` is human-typed, correctly outside its
declared scope) while the extended guard correctly fails — proving the
extension adds real coverage the narrow guard structurally cannot provide.
Source was then restored and confirmed byte-for-byte identical to the
pre-injection version via `diff` against a saved copy (not `git diff`
against HEAD, which would also show the legitimate GAP-4 fix itself); both
guards pass again after restoration.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `repairDirtyWorktree`'s orphan-branch `git branch -D` was missing `-C root`**
- **Found during:** Writing the GAP-4 fail-then-pass test
- **Issue:** The pre-fix `exec.Command("git", "branch", "-D", branchName)`
  had no working-directory argument, so it operated on the test process's
  (or, in production, the CLI invocation's) working directory rather than
  the Aether repo root — a second, independent bug beyond GAP-4's missing
  mergedness check, discovered because it made the fail-then-pass test's
  observed pre-fix failure message different from what a "delete succeeds
  but shouldn't" scenario would show.
- **Fix:** Added `-C root` (using the same `resolveAetherRoot()` the
  `Orphan branch` case now also uses for `branchMergeSafety`) to the delete
  call.
- **Files modified:** `cmd/recover_repair.go`
- **Commit:** 9ff05982

### Deferred Issues

None — both gaps and the guard extension were closed within scope, and the
fixes did not surface any out-of-scope defects.

## Verification

- `go build ./...` — clean
- `go vet ./...` — clean
- `gofmt -l` — clean on every file this plan touched (one pre-existing,
  untouched file — `cmd/codex_build.go` — has unrelated formatting drift
  from before this plan; left alone, out of scope)
- `go test ./... -race` — all packages pass, including the full `cmd`
  package (380.1s)
- `TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate` (the
  pre-existing narrow guard) still passes unchanged
- `TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate` (the new,
  extended guard) passes, and was demonstrated to catch a reintroduced
  defect the narrow guard cannot see (see Proof section above)

## Self-Check: PASSED

- `cmd/recover_repair.go` — FOUND (modified)
- `cmd/worktree_safety.go` — FOUND (modified)
- `cmd/entomb_cmd.go` — FOUND (modified)
- `cmd/abandon_cmd.go` — FOUND (modified)
- `cmd/worktree_destruction_reachability_test.go` — FOUND (modified)
- `cmd/worktree_operator_destruction_test.go` — FOUND (modified)
- Commit `9ff05982` — FOUND (`git log --oneline` confirms)
