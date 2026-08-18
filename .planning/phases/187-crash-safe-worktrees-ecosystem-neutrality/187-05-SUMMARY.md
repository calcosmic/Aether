---
phase: 187-crash-safe-worktrees-ecosystem-neutrality
plan: 05
subsystem: infra
tags: [git, worktree, crash-safety, data-loss, go]

# Dependency graph
requires:
  - phase: 187-crash-safe-worktrees-ecosystem-neutrality
    provides: worktreeDestructionSafety, preserveWorktreeWork, gcOrphanedWorktrees, worktree-reap (plans 01-04)
provides:
  - Fail-closed empty/malicious branch handling in worktreeDestructionSafety (CR-01)
  - Top-level worktree verification before trusting git status (CR-02)
  - Fail-fast removeGitWorktree that never deletes a branch after worktree removal fails (CR-03)
  - Honest preserved=false from preserveWorktreeWork when state is undeterminable, enforced at the worktree-reap consumer (CR-04)
  - cleanupBuildWorktrees routed through the same safety guard as gcOrphanedWorktrees, closing the unguarded build-path destruction (CR-05)
  - Five new real-git fail-then-pass tests plus two corrected wrong-reason tests plus two ratchets on the sanctioned-caller set
affects: [187-06, any future worktree lifecycle work, recover, worktree-reap]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "worktreeDestructionSafety must fail closed before any git query that could vacuously succeed on empty input"
    - "removeGitWorktree fails fast: never run branch -D after worktree remove has already failed"
    - "preserveWorktreeWork's boolean return must be honored, not discarded with `_,`, at every destructive call site"
    - "Every lifecycle path that can reach removeGitWorktree must route through worktreeDestructionSafety + preserveWorktreeWork first"

key-files:
  created: []
  modified:
    - cmd/worktree_safety.go
    - cmd/worktree_safety_test.go
    - cmd/worktree_reap.go
    - cmd/codex_build_worktree.go
    - cmd/codex_build_worktree_test.go
    - cmd/codex_build.go
    - cmd/worktree_crash_safety_test.go

key-decisions:
  - "CR-02's fixture for TestWorktreeSafetyRefusesWhenGitCannotAnswer needed a .gitignore covering the whole /.aether/ directory, not just the specific subdirectory under test, because newWorktreeSafetyFixture already leaves .aether/worktrees/... untracked before the test body runs"
  - "git worktree lock does NOT reproduce CR-03's defect (git protects a locked/registered worktree's branch from -D); an unwritable worktree directory (chmod 0500) does, because git unregisters the worktree before failing to delete its remaining contents -- verified manually against real git 2.52 before writing the Go test"
  - "CR-04's end-to-end test needed a genuine git-worktree-add'd directory with an empty Branch field, not a plain stale directory, because a plain non-worktree directory makes removeGitWorktree fail for an unrelated reason (git worktree remove refuses a non-worktree path) which masks the CR-04 defect entirely"
  - "codex_build.go's swallowed cleanupBuildWorktrees error is now surfaced via emitVisualProgress rather than treated as fatal, since cleanup itself is non-destructive (CR-05) but a silent failure would still hide unfinalized worktrees from the build operator"

requirements-completed: []

# Metrics
duration: ~90min
completed: 2026-08-18
---

# Phase 187 Plan 05: Close the Five Confirmed Data-Loss Defects Summary

**Closed CR-01 through CR-05 from the 187-REVIEW.md code review — five confirmed, execution-verified data-loss defects in the crash-safe worktree machinery, each now covered by a real-git test that fails without the fix and passes with it.**

## Performance

- **Duration:** ~90 min
- **Completed:** 2026-08-18
- **Tasks:** 5 (one per defect)
- **Files modified:** 7

## Accomplishments

- Closed CR-01: an entry with an empty branch name no longer reads as "safe to destroy" — `worktreeDestructionSafety` now fails closed before running any git query that could vacuously succeed on empty input, and the `rev-list` call now separates the revision range from flags with `--` so a branch name beginning with `-` cannot be misread as an option.
- Closed CR-02: the dirty check now confirms the entry's path is genuinely the top level of its own git worktree (via `rev-parse --show-toplevel`) before trusting any status answer, so a stale/deregistered directory can no longer silently inherit the enclosing repo's "clean" status.
- Closed CR-03: `removeGitWorktree` now fails fast — if `git worktree remove` fails, the function returns immediately and never reaches `git branch -D`, so a caller reading a non-nil error as "nothing was destroyed" is now actually correct.
- Closed CR-04: `preserveWorktreeWork`'s undeterminable-state branch now reports `preserved=false` with an honest detail string instead of `preserved=true` having stashed nothing; the `worktree-reap --force --include-unmerged` consumer now refuses on `!preserved` as well as on a non-nil error, closing the path where it printed "its changes were saved first" and then destroyed an unexamined worktree.
- Closed CR-05: `cleanupBuildWorktrees`, which runs unconditionally on every build and selects exactly the entries a same-phase crash leaves behind, now routes through the same `worktreeDestructionSafety` + `preserveWorktreeWork` guard as `gcOrphanedWorktrees` instead of calling `removeGitWorktree` directly. The stale "only remaining caller" doc comment was corrected, the swallowed build-path error is now surfaced, and two ratchets were added: a source-level ratchet on `cleanupBuildWorktrees`'s body (mirroring the existing GC ratchet) and a whole-codebase scan asserting `removeGitWorktree`'s only callers are the sanctioned set (`removeGitWorktree` itself, `allocateBuildWorktree`'s rollback, `finalizeBuildWorktree`'s post-sync cleanup, and `runWorktreeReap`).

## Task Commits

Each defect was committed atomically:

1. **CR-01: fail closed on empty worktree branch name** - `e9c1e19c` (fix)
2. **CR-02: guard dirty check against wrong-repo answers** - `4cf37805` (fix)
3. **CR-03: make removeGitWorktree fail-fast** - `dedebd24` (fix)
4. **CR-04: report preserved=false when worktree state is undeterminable** - `f25d1d4d` (fix)
5. **CR-05: guard cleanupBuildWorktrees on the build path** - `a8077c4c` (fix)

## Files Created/Modified

- `cmd/worktree_safety.go` - CR-01 empty-branch guard + `--` flag separation; CR-02 top-level worktree check; CR-04 honest `preserved=false` on the undeterminable branch
- `cmd/worktree_safety_test.go` - New tests `TestWorktreeSafetyRefusesEmptyBranch`, `TestWorktreeSafetyRefusesStaleDirectoryWithUntrackedWork`, `TestPreserveWorktreeWorkReportsFalseWhenStateUndeterminable`; corrected `TestWorktreeSafetyRefusesWhenGitCannotAnswer` to use a clean root
- `cmd/worktree_reap.go` - CR-04 consumer fix (`!preservedOK` refusal); corrected doc comment on `worktreeReapCmd` naming the actual sanctioned caller set
- `cmd/codex_build_worktree.go` - CR-03 fail-fast `removeGitWorktree`; CR-05 `cleanupBuildWorktrees` routed through the safety guard
- `cmd/codex_build_worktree_test.go` - Updated `TestRemoveGitWorktreeErrorPropagation`'s message assertion for the new fail-fast error format
- `cmd/codex_build.go` - Surfaced the previously swallowed `cleanupBuildWorktrees` error via `emitVisualProgress`
- `cmd/worktree_crash_safety_test.go` - New tests `TestRemoveGitWorktreeDoesNotDeleteBranchWhenRemovalFails`, `TestWorktreeReapIncludeUnmergedRefusesUndeterminableState`, `TestCleanupBuildWorktreesSurvivesCrashOnBuildPath`, `TestCleanupBuildWorktreesNeverCallsRemoveGitWorktree`, `TestRemoveGitWorktreeHasOnlySanctionedCallers`

## Decisions Made

- Used `chmod 0500` on the worktree directory (not `git worktree lock`) to reproduce CR-03's data-loss scenario, after manually confirming against real git 2.52 that a locked worktree already protects its branch from `-D` — only an unregistered-but-partially-undeleted worktree actually loses the branch.
- Broadened the `.gitignore` pattern in the corrected `TestWorktreeSafetyRefusesWhenGitCannotAnswer` to cover the whole `.aether/` directory rather than just the specific test subdirectory, since the shared fixture helper already leaves an untracked worktree directory in place before the test body runs.
- Used a genuine `git worktree add`-created directory with an empty `Branch` field (not a plain non-worktree stale directory) for CR-04's end-to-end reap test, after discovering that a plain stale directory makes `removeGitWorktree` fail for an unrelated, masking reason (`git worktree remove` refuses non-worktree paths outright).
- Left `finalizeBuildWorktree` untouched — it runs only after `reconcileWorktreeWave` has already synced a successful worker's output into the root checkout, which is a different, non-crash-recovery lifecycle stage than the five defects in scope, and the review did not flag it as a CR.

## Deviations from Plan

None — all five fixes match the review's suggested approach. The only additions beyond the review's literal suggested code were the two extra ratchet tests for CR-05 (a source-level ratchet on `cleanupBuildWorktrees` plus the whole-codebase sanctioned-caller scan), which the review explicitly asked for ("widen the ratchet so it asserts the property rather than one function name").

## Issues Encountered

- **CR-02 test setup:** the first draft of the corrected `TestWorktreeSafetyRefusesWhenGitCannotAnswer` still failed its own "clean root" sanity check, because the shared `newWorktreeSafetyFixture` helper creates an untracked worktree directory before the test body runs, and a narrow `.gitignore` pattern didn't cover it. Fixed by gitignoring the whole `.aether/` directory (matching the real, documented "local-only" configuration) and confirming git re-evaluates ignore status on every `status` call even for pre-existing untracked paths.
- **CR-03 test technique:** the first reproduction attempt (`git worktree lock`) passed even without the fix, because a genuinely locked/registered worktree makes real git refuse `branch -D` on its own ("used by worktree at ..."), independent of the code under test. Diagnosed by manually running the three-command sequence against the real git binary outside Go, found that an unwritable worktree directory (`chmod 0500`) reproduces the actual defect: git unregisters the worktree before failing to delete its remaining contents, so `branch -D` genuinely succeeds afterward if the code doesn't fail fast.
- **CR-04 test technique:** the first end-to-end reproduction (a plain stale, non-worktree directory) also passed without the fix, because `git worktree remove` itself refuses a path that was never a real worktree, masking the CR-04 defect behind an unrelated failure. Fixed by using a genuine `git worktree add`-created worktree with `entry.Branch = ""`, where `git worktree remove` succeeds in deleting the directory and only the trailing `branch -D ""` step fails — exactly the shape where the buggy `preserved=true` mattered.

## Pre-Fix Failure Messages (fail-then-pass proof)

Each test below was run against the unfixed code (via `git stash` on the relevant source file) and observed to fail with the message shown, then confirmed to pass after restoring the fix.

**CR-01 — `TestWorktreeSafetyRefusesEmptyBranch`:**
```
expected Safe=false for an entry with an empty branch name, got Safe=true (reason: "clean and fully merged") — an empty branch must never resolve to 'safe to destroy'
```

**CR-02 — `TestWorktreeSafetyRefusesStaleDirectoryWithUntrackedWork`:**
```
expected Safe=false for a stale directory holding untracked work that git resolves to the enclosing (clean) repo, got Safe=true (reason: "clean and fully merged", dirty=0)
```
**CR-02 — corrected `TestWorktreeSafetyRefusesWhenGitCannotAnswer`** (previously passed for the wrong reason; now fails without the fix):
```
expected Safe=false when git cannot determine worktree state — uncertainty must not read as permission
```

**CR-03 — `TestRemoveGitWorktreeDoesNotDeleteBranchWhenRemovalFails`:**
```
expected branch "phase-1/builder-unwritable" to still exist after a failed removal, but it is gone — removeGitWorktree deleted the branch even though it reported an error
```

**CR-04 — `TestPreserveWorktreeWorkReportsFalseWhenStateUndeterminable`:**
```
expected preserved=false when the worktree's state could not be determined — nothing was actually saved, so reporting preserved=true is a lie a destructive caller could act on. detail="could not check worktree state, so branch phase-1/builder-undeterminable was left alone"
```
**CR-04 — `TestWorktreeReapIncludeUnmergedRefusesUndeterminableState`** (end-to-end):
```
expected the undeterminable-state worktree to survive --force --include-unmerged, but it is gone: stat .../.aether/worktrees/phase-1-builder-undeterminable: no such file or directory
```

**CR-05 — `TestCleanupBuildWorktreesSurvivesCrashOnBuildPath`:**
```
expected orphaned >= 1 (the dirty worktree must be preserved, not destroyed), got orphaned=0 cleaned=1
expected worktree directory to still exist after cleanupBuildWorktrees, stat error: stat .../.aether/worktrees/phase-4-builder-crashed-midbuild: no such file or directory
```
**CR-05 ratchets** (both fail on the pre-fix `codex_build_worktree.go`):
```
TestCleanupBuildWorktreesNeverCallsRemoveGitWorktree:
  cleanupBuildWorktrees's body still contains a call to removeGitWorktree outside of comments — this reintroduces the exact unguarded build-path destruction CR-05 removed

TestRemoveGitWorktreeHasOnlySanctionedCallers:
  found call(s) to removeGitWorktree outside the sanctioned caller set [...]: [codex_build_worktree.go: if removeErr := removeGitWorktree(root, absPath, entry.Branch); removeErr != nil {]
```

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- All five confirmed data-loss defects from `187-REVIEW.md` are closed and covered by real-git, fail-then-pass tests.
- The full `go test ./...` suite passes (all packages, no failures) and `go build ./...` / `go vet ./...` are both clean.
- The review's six Warnings (WR-01..WR-06) and three Info items (IN-01..IN-03) were explicitly out of scope for this plan (only CR-01..CR-05 were assigned) and remain open for a future gap-closure pass if the owner wants them addressed.
- `finalizeBuildWorktree`'s and `allocateBuildWorktree`'s own `removeGitWorktree` calls remain unguarded by design (they are not crash-recovery paths — see Decisions Made) and are now explicitly documented and ratchet-enforced as the sanctioned exception set, rather than silently coexisting with an inaccurate comment.

---
*Phase: 187-crash-safe-worktrees-ecosystem-neutrality*
*Completed: 2026-08-18*
