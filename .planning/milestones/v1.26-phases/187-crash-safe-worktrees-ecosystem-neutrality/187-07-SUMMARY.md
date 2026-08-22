---
phase: 187-crash-safe-worktrees-ecosystem-neutrality
plan: 07
subsystem: worktree-safety
tags: [go, git, worktree, crash-safety, verification-remediation]

# Dependency graph
requires:
  - phase: 187-crash-safe-worktrees-ecosystem-neutrality
    provides: worktreeDestructionSafety, preserveWorktreeWork, reportWorktreePreservation, removeGitWorktree (plans 01-06)
provides:
  - worktreeMergeBackCmd's Step 5 (cmd/worktree.go) gated on worktreeDestructionSafety before any destructive git command
  - worktreeCleanupCmd (cmd/clash.go) rewritten to resolve the real worktree path, gate on worktreeDestructionSafety, and preserve-then-destroy on --force (matching worktree-reap)
  - scanUnrecordedWorktrees (cmd/worktree_safety.go) — finds git worktrees on disk that are not yet tracked in COLONY_STATE.json
  - init's worktrees-directory wipe (cmd/init_cmd.go) now consults scanUnrecordedWorktrees before trusting wtPreserved==0
  - Real-git fail-then-pass tests for all three gaps in cmd/worktree_operator_destruction_test.go
affects: [worktree-merge-back, worktree-cleanup, init, any future worktree destruction call site, cmd/worktree_destruction_reachability_test.go's documentation of what it does and does not cover]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Being operator-invoked (excluded from the AST reachability guard's lifecycleEntryCommands) is not itself a safety property — worktree-merge-back and worktree-cleanup were both operator-typed AND unguarded until this plan; the guard's exclusion list only says 'a human must type this', not 'this is safe'"
    - "A directory listing (os.ReadDir) independent of state is the only way to see a worktree created but not yet recorded — any preservation count derived solely from state.Worktrees has a crash window it cannot see by construction"
    - "git -C <path> rev-parse walks UP to an enclosing repository rather than failing for a non-worktree directory; any code that treats 'git can answer a question about this path' as 'this path is itself a worktree' needs the same --show-toplevel identity check worktreeDestructionSafety's own Step 2.5 already used"

key-files:
  created:
    - cmd/worktree_operator_destruction_test.go
  modified:
    - cmd/worktree.go
    - cmd/clash.go
    - cmd/init_cmd.go
    - cmd/worktree_safety.go
    - cmd/worktree_destruction_reachability_test.go
    - cmd/testdata/command_catalog.json

key-decisions:
  - "worktree-merge-back: kept the merge itself unconditional (it only touches the already-safe committed HEAD) and gated only Step 5's destructive cleanup — on unsafe, the entry is marked WorktreeOrphaned (not WorktreeMerged) so `aether recover` / `worktree-reap` can still find it, and the command reports cleaned_up:false rather than silently succeeding as if nothing were left behind"
  - "worktree-cleanup: added a --force flag with the same preserve-then-destroy contract as worktree-reap's --force --include-unmerged path, rather than making the command permanently refuse dirty worktrees with no destructive escape hatch at all"
  - "worktree-cleanup: fixed the latent path bug (branch name passed directly as the worktree PATH argument) by resolving the real path via `git worktree list --porcelain`, matching on `branch refs/heads/<name>`, instead of assuming the two strings are interchangeable"
  - "init: did not add worktree-merge-back or worktree-cleanup to lifecycleEntryCommands in the AST reachability guard — they remain human-invoked, not automatic, so belong outside that list by the guard's own stated definition; instead updated the list's doc comment to state plainly that exclusion from the guard is not evidence of safety, and that the real proof for these two commands lives in the new real-git tests, not in reachability analysis"
  - "scanUnrecordedWorktrees reuses worktreeDestructionSafety rather than re-implementing a dirty/unmerged check, and applies the identical --show-toplevel identity guard worktreeDestructionSafety's Step 2.5 already uses, after a first version of this fix mis-flagged a stray non-worktree directory as unsafe because git resolved the check upward into the enclosing Aether repo"

requirements-completed: []

# Metrics
duration: ~110min
completed: 2026-08-19
---

# Phase 187 Plan 07: Close the Final Two Unguarded Worktree-Destruction Commands Summary

**Routed worktree-merge-back's post-merge cleanup, worktree-cleanup, and init's worktrees-directory wipe through the existing worktreeDestructionSafety guard — the three remaining places a real, shipped Aether command could still silently discard uncommitted or unmerged worker output — each proven by a real-git test that failed against the unfixed code and passes against the fix.**

## Performance

- **Duration:** ~110 min
- **Completed:** 2026-08-19
- **Tasks:** 3 gaps (GAP-1, GAP-2, GAP-3), closed together as one commit
- **Files modified:** 6 (1 created, 5 modified)

## What This Closes

187-VERIFICATION.md (status: gaps_found) named three places where "no Aether
command ever deletes unmerged or dirty work" did not hold:

- **GAP-1** — `worktree-merge-back`'s Step 5 auto-cleanup ran `git worktree
  remove --force` unconditionally after a successful merge, with no check for
  uncommitted content left in the worktree alongside the merged commit.
- **GAP-2** — `worktree-cleanup` had no safety check of any kind, and passed
  the branch name directly as the worktree's PATH argument to `git worktree
  remove` (a second, independent bug).
- **GAP-3** — `init`'s worktrees-directory wipe (`os.RemoveAll`) was gated
  only on a count of entries `gcOrphanedWorktrees` found in
  `COLONY_STATE.json`; a worktree created by `git worktree add` but killed
  before its state entry was appended (the exact crash window this phase
  exists to protect) was invisible to that count and got wiped.

## Accomplishments

**GAP-1 (`cmd/worktree.go`):** Step 5 now calls `worktreeDestructionSafety`
before running any destructive git command. If unsafe, `preserveWorktreeWork`
stashes the dirty content (or, for unmerged commits, simply declines to
delete the branch), `reportWorktreePreservation` writes the plain-language
D-02 report, the worktree entry is marked `WorktreeOrphaned` instead of
`WorktreeMerged`, and the command returns `cleaned_up:false` with the
preservation detail. The merge itself (already durable on `main`) is
unaffected either way.

**GAP-2 (`cmd/clash.go`):** Added `resolveWorktreePathForBranch`, which asks
git directly (`git worktree list --porcelain`, matched against `branch
refs/heads/<name>`) instead of assuming the branch name is the worktree's
directory path. `worktreeCleanupCmd` now loads the tracked
`colony.WorktreeEntry` when one exists (falling back to a minimal
path+branch entry when it doesn't), calls `worktreeDestructionSafety`, and:
refuses and reports when unsafe; with the new `--force` flag, preserves
(stashes) first and only then destroys, exactly matching `worktree-reap`'s
`--force --include-unmerged` contract; destroys directly when safe.

**GAP-3 (`cmd/init_cmd.go` + `cmd/worktree_safety.go`):** Added
`scanUnrecordedWorktrees(root, worktreesDir, knownPaths)`, which lists
`worktreesDir` on disk, confirms each unknown subdirectory is genuinely the
top level of its own git worktree (the same `--show-toplevel` identity check
`worktreeDestructionSafety`'s own Step 2.5 uses, added after an early version
of this scan mis-flagged a stray non-worktree directory by letting git
resolve the check upward into the enclosing repo), recovers its branch name
from git itself, and runs `worktreeDestructionSafety` on it. `init` now adds
any unsafe unrecorded worktrees to `wtPreserved` and reports each one before
deciding whether to wipe the directory.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `scanUnrecordedWorktrees` initially mis-flagged a stray non-worktree directory as unsafe**
- **Found during:** Writing `TestInitStillWipesWorktreesDirectoryWhenTrulyEmpty`
- **Issue:** The first version of `scanUnrecordedWorktrees` confirmed a
  candidate directory was a git worktree by running `git -C <path> rev-parse
  --abbrev-ref HEAD` and checking for a non-error exit. Git does not fail for
  a plain subdirectory that is not itself a worktree — it walks UP the
  directory tree until it finds an enclosing `.git`. For a stray, empty,
  non-worktree directory under `.aether/worktrees/`, this resolved to the
  Aether root's own repository and returned `main`, which `worktreeDestructionSafety`
  then correctly judged unsafe ("this folder is no longer a separate worker
  workspace") — but for the wrong reason: the directory was never a worktree
  to begin with, so `init` refused to wipe an empty directory that held
  nothing.
- **Fix:** Added the identical `--show-toplevel` identity check
  `worktreeDestructionSafety`'s own Step 2.5 already uses: resolve
  `git -C <path> rev-parse --show-toplevel`, `EvalSymlinks` both sides, and
  only treat the candidate as a worktree if it resolves to itself.
- **Files modified:** `cmd/worktree_safety.go`
- **Commit:** 9fbc3a29

### Deferred Issues

None — all three gaps were closed within scope, and the fixes did not
surface any out-of-scope defects.

## Proof: Fail-Then-Pass, Actually Observed

Per this repo's Definition of Done, each test below was run against the
unfixed code first (by `git stash push` on the three production files while
keeping the tests and the shared `worktreeDestructionSafety`/`preserveWorktreeWork`
helpers in place), and the failure was directly observed before the fix was
restored via `git stash pop`. All three genuinely failed pre-fix and pass
post-fix — no test passed before its corresponding fix was applied.

**GAP-1 — `TestWorktreeMergeBackPreservesUncommittedWorkAfterMerge`:**
```
worktree_operator_destruction_test.go:153: expected the uncommitted file
to survive merge-back (directly or via stash), but it is gone: stat
/var/folders/.../.aether/worktrees/phase-1-builder-preserve/leftover-notes.txt:
no such file or directory (stash list also empty)
```

**GAP-2 — `TestWorktreeCleanupRefusesToDestroyDirtyWorktree`:**
```
worktree_operator_destruction_test.go:250: expected a plain-language report
that the work was kept, got: {"ok":false,"error":"failed to remove worktree:
exit status 128: fatal: 'phase-1/builder-dirty' is not a working tree\n","code":2}
```
(The pre-fix command passed the branch name directly as the worktree's path
argument. For this fixture's directory naming convention that string is not
a valid path, so git refuses outright rather than silently deleting anything
— but the underlying defect is identical: zero safety check runs, and for
any branch name that DOES equal its own worktree directory name, the same
call would have deleted real, uncommitted content unconditionally, exit 0,
no warning.)

**GAP-3 — `TestInitPreservesUnrecordedWorktreeWithUncommittedWork`:**
```
worktree_operator_destruction_test.go:420: expected the unrecorded
worktree's uncommitted file to survive init, but it is gone: stat
/var/folders/.../.aether/worktrees/phase-9-builder-unrecorded/crash-notes.txt:
no such file or directory (stash list also empty, directory itself gone)
```

Two companion positive-path tests confirm the fixes did not turn these
commands into permanent no-ops: `TestWorktreeCleanupRemovesCleanMergedWorktree`
(a genuinely clean, merged worktree is still removed) and
`TestInitStillWipesWorktreesDirectoryWhenTrulyEmpty` (a worktrees directory
holding no real work is still wiped).

## Verification

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./... -race` — all packages pass, including the full `cmd` package
  (392.9s). One golden-file test (`TestAuditCatalogGolden`) needed
  regeneration after `worktree-cleanup` gained a `--force` flag — this is the
  catalog test doing its job (a real, intentional CLI surface change), not a
  defect; the diff is exactly the one new flag.
- `TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate`
  (the AST reachability guard from plan 06) still passes unchanged — its
  `lifecycleEntryCommands` list correctly does not include `worktree-merge-back`
  or `worktree-cleanup`, since both remain human-invoked, not automatic. Its
  doc comment was updated to state explicitly that this exclusion is not a
  safety claim about those two commands, and to point at this plan's real-git
  tests as the actual proof.

## Self-Check: PASSED

- `cmd/worktree.go` — FOUND (modified)
- `cmd/clash.go` — FOUND (modified)
- `cmd/init_cmd.go` — FOUND (modified)
- `cmd/worktree_safety.go` — FOUND (modified)
- `cmd/worktree_destruction_reachability_test.go` — FOUND (modified)
- `cmd/worktree_operator_destruction_test.go` — FOUND (created)
- `cmd/testdata/command_catalog.json` — FOUND (modified)
- Commit `9fbc3a29` — FOUND (`git log --oneline` confirms)
