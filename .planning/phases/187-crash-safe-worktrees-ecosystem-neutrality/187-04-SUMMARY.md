---
phase: 187-crash-safe-worktrees-ecosystem-neutrality
plan: 04
subsystem: infra
tags: [git, worktree, testing, ecosystem-neutrality, build-finalize]

# Dependency graph
requires:
  - phase: 187-02
    provides: "resolveTestCommand() wiring pattern for the worktree-merge-back CLI command, mirrored here"
  - phase: 187-03
    provides: "gcOrphanedWorktrees rewrite in the same file (disjoint line range, applied cleanly)"
provides:
  - "mergePhaseWorktrees (the live build-path merge gate called from codex_build_finalize.go) drives its test gate from resolveTestCommand() instead of a hard-coded go test ./... literal"
  - "D-03 refusal path: an undeterminable test command blocks the merge before any git command runs, and returns through the existing err value"
  - "Four real-git fixture tests: Node-repo success, unknown-command refusal, refusal preservation, Go-repo regression"
affects: [190-lean-non-duplicated-delivery]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Test command resolved once before the worktree loop (not per-entry), matching cmd/worktree.go's plan-02 approach for the same resolver"
    - "Refusal returns before the loop, so no exec.CommandContext(..., \"git\", ...) call is reachable from the empty-command branch"

key-files:
  created:
    - cmd/merge_phase_worktrees_test.go
  modified:
    - cmd/codex_build_worktree.go

key-decisions:
  - "Did not consolidate mergePhaseWorktrees with worktreeMergeBackCmd (cmd/worktree.go) -- CONTEXT.md's <deferred> section assigns that to Phase 190; this plan only stops both sites from hard-coding the Go command."
  - "mergePhaseWorktrees does not itself write state.Worktrees[i].Status = WorktreeMerged after a successful merge -- this is pre-existing behavior unrelated to the ecosystem-neutrality fix (worktreeMergeBackCmd in worktree.go does update status; mergePhaseWorktrees never has). The Go-repo regression test asserts the git-level merge (branch's commit reachable from main) rather than a state field this function has never written, to avoid asserting a claim the function doesn't make."
  - "Refusal message and format follow cmd/worktree.go's plan-02 wording almost verbatim, substituting the phase number for the branch name, since mergePhaseWorktrees operates on a whole phase's worktrees rather than one branch."

requirements-completed: []

# Metrics
duration: 45min
completed: 2026-08-18
---

# Phase 187 Plan 04: Lean Build-Path Merge Gate Summary

**`mergePhaseWorktrees` -- the merge gate that actually runs during a real worktree-mode build -- now resolves the project's own test command instead of hard-coding `go test ./...`, refuses to merge and preserves the work when no command can be determined, and is covered for the first time by tests that build a real git worktree and reach the test gate, clash gate, and merge.**

## Performance

- **Duration:** 45 min
- **Started:** 2026-08-18T20:05:00Z (approx, worktree branch base corrected before Task 1)
- **Completed:** 2026-08-18T20:51:38Z
- **Tasks:** 2
- **Files modified:** 2 (1 modified, 1 created)

## Accomplishments

- `mergePhaseWorktrees` in `cmd/codex_build_worktree.go` now resolves `testCommand := strings.TrimSpace(resolveTestCommand())` once before the worktree loop and executes it via `strings.Fields` + `exec.CommandContext`, replacing the hard-coded `"go", "test", "./..."` argv literal that ran unconditionally on every worktree-mode build finalization (called live from `cmd/codex_build_finalize.go:569`).
- Added the D-03 refusal branch: an empty resolver result returns an error containing the exact substring `cannot determine how to test this project` through the function's existing `err` return value, before the loop runs and before any git command executes -- no test, clash check, checkout, or merge is reachable from that branch.
- Added `cmd/merge_phase_worktrees_test.go` with four new tests exercising `mergePhaseWorktrees` directly against real git fixtures (real `git init`, `git worktree add`, real commits): a Node-repo merge success (CLAUDE.md-resolved `npm test`, no `go.mod`), an unresolvable-project refusal (asserting the error message, empty `merged`, and empty `failed`), a real-git preservation proof (worktree directory, `git worktree list`, `git branch --list`, and `git log` on the branch all still present and correct after refusal), and a Go-repo regression guard that drives the test gate, clash gate, checkout fallback, and merge end to end for the first time.
- Manually demonstrated fail-then-pass for both changes: with the empty-command refusal branch removed, `TestMergePhaseWorktreesRefusesWhenTestCommandUnknown` panics on an empty `testFields` slice (`slice bounds out of range [1:0]`), proving the guard is load-bearing; with the gate reverted to the hard-coded `go test ./...` literal, `TestMergePhaseWorktreesUsesProjectTestCommandInNodeRepo` fails because the Node fixture has no `go.mod` and `go test` errors. Both mutations were reverted and the file confirmed byte-identical to the Task 1 commit (`git status --short` and `git diff --stat` both empty) before Task 2 was committed.
- Both merge-gate implementations in the codebase (`cmd/worktree.go` from plan 02, `cmd/codex_build_worktree.go` from this plan) are now confirmed ecosystem-neutral: `grep -c '"go", "test", "\./\.\.\."' cmd/codex_build_worktree.go cmd/worktree.go` reports `0` for both files.

## Task Commits

Each task was committed atomically:

1. **Task 1: Drive the mergePhaseWorktrees test gate from resolveTestCommand** - `ed415daa` (feat)
2. **Task 2: Real-git fixture tests for mergePhaseWorktrees** - `a7644928` (test)

**Plan metadata:** (this commit, made by the orchestrator after wave merge)

## Files Created/Modified

- `cmd/codex_build_worktree.go` - `mergePhaseWorktrees` now resolves the test command once before the loop, refuses-and-preserves on an empty result (returning before any git command runs), and executes the resolved command via `strings.Fields` instead of the hard-coded Go literal for the non-empty case. Confined to lines 1149-1246 (plan 03's earlier edits to `gcOrphanedWorktrees`, `removeGitWorktree`, and `reconcileWorktreeWave` in the same file are untouched).
- `cmd/merge_phase_worktrees_test.go` - Four new tests plus shared fixture helpers (`mergeWorktreeFixture`, `finishMergeWorktreeFixture`) that build a real git repository, a real `git worktree add`, and real commits, then call `mergePhaseWorktrees` directly.

## Decisions Made

- Followed `cmd/worktree.go`'s plan-02 approach for consistency: resolve once, `strings.TrimSpace` + `strings.Fields`, refuse before any git command, do not copy `checkTestsPass`'s pass-by-default-on-empty shape.
- Left the two duplicate merge-gate implementations un-consolidated, per CONTEXT.md's `<deferred>` section assigning that to Phase 190.
- Adjusted the Go-repo regression test's final assertion after discovering `mergePhaseWorktrees` never writes `state.Worktrees[i].Status = WorktreeMerged` on success (unlike `worktreeMergeBackCmd`, which does) -- this is pre-existing, out-of-scope behavior. The test instead asserts the git-level outcome (`merged` contains the branch, the worktree's commit is reachable from `main`) rather than a state field this function has never set.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug in test, caught before commit] Go-repo regression test asserted a state field `mergePhaseWorktrees` never writes**

- **Found during:** Task 2, first run of `TestMergePhaseWorktreesStillMergesGoRepo`
- **Issue:** The test's first draft asserted `state.Worktrees[i].Status == colony.WorktreeMerged` after a successful merge, following the shape of the plan-02 CLI test (`worktreeMergeBackCmd`, which does update status). `mergePhaseWorktrees` itself never mutates `state.Worktrees` at all -- confirmed by reading the full function body -- so the assertion failed even though the merge itself succeeded correctly (`merged` contained the branch, and the commit was genuinely present on `main`).
- **Fix:** Reworded the assertion to check the git-level outcome directly (`git log --oneline main` contains the worktree's commit message) instead of a state field the function under test does not set. This is a test-only correction; no production code changed as a result.
- **Files modified:** `cmd/merge_phase_worktrees_test.go`
- **Verification:** `TestMergePhaseWorktreesStillMergesGoRepo` passes; the full suite of four new tests plus the pre-existing `TestMergePhaseWorktreesEmpty` all pass.
- **Committed in:** `a7644928` (Task 2 commit; caught and fixed before commit, so no separate fix commit exists)

**Total deviations:** 1 (test-only correction, no production code impact). No architectural changes, no scope creep.

## Fail-Then-Pass Demonstration

Performed as required by Task 2's acceptance criteria:

1. **Refusal branch removed:** Temporarily deleted the `if len(testFields) == 0 { ... return ... }` block from `mergePhaseWorktrees`, leaving `testFields` resolved but unchecked. Ran `TestMergePhaseWorktreesRefusesWhenTestCommandUnknown`: it panicked with `runtime error: slice bounds out of range [1:0]` inside `exec.CommandContext(testCtx, testFields[0], testFields[1:]...)`, proving the guard prevents an unreachable-otherwise crash on an undeterminable project and that the test genuinely exercises the removed branch.
2. **Restored** the file from a pre-mutation backup; confirmed `git status --short cmd/codex_build_worktree.go` and `git diff --stat cmd/codex_build_worktree.go` were both empty (byte-identical to the Task 1 commit) before proceeding.
3. **Hard-coded literal reinstated:** Temporarily replaced `exec.CommandContext(testCtx, testFields[0], testFields[1:]...)` with the original `exec.CommandContext(testCtx, "go", "test", "./...")`. Ran `TestMergePhaseWorktreesUsesProjectTestCommandInNodeRepo`: it failed (`expected 0 failed, got [phase-1/builder-a (tests failed)]`; `expected phase-1/builder-a in merged, got []`) because the Node fixture has no `go.mod` and `go test` errors out, confirming the test genuinely proves the resolver is wired in rather than passing vacuously.
4. **Restored again**, confirmed byte-identical to the Task 1 commit via the same two checks, then ran the full four-test suite plus `TestMergePhaseWorktreesEmpty`: all pass.
5. Ran the full `go test ./cmd/ -count=1` suite: **passed** (299.9s, exit code 0) -- no regressions from either the fix or the new test file.

## Issues Encountered

**Worktree branch base was stale at agent start**, same class of issue as plans 02 and 03: `HEAD` was on an old commit (`6577f51c`, a CI-fix chain unrelated to Phase 187) instead of the expected wave-3 tracking commit (`63e924bc...`). Confirmed HEAD was on the correct `worktree-agent-*` namespace (not a protected ref), the working tree was clean, and `63e924bc` was not an ancestor of the stale HEAD (diverged lineage, not a fast-forward situation). Per the documented recovery protocol, the branch was reset to the expected base before any task work began.

**Threat model note (T-187-13):** as recorded in the plan's threat register, a slow non-Go test suite exceeding the 120s `BuildTimeout` is now reachable on non-Go projects and is explicitly deferred (flagged in `cmd/timeouts.go:11`, out of scope for this phase). No code change was made for this; it is restated here per the register's own instruction.

## User Setup Required

None -- no external service configuration required. This plan changes internal Go function behavior and adds test coverage; nothing requires end-user configuration.

## Next Phase Readiness

Both hard-coded `go test ./...` merge-gate implementations in the codebase are now ecosystem-neutral (`cmd/worktree.go` from plan 02, `cmd/codex_build_worktree.go` from this plan), closing ROADMAP criterion 2 in full. `mergePhaseWorktrees` now has real-path test coverage proving the test gate, clash gate, checkout fallback, and merge all function correctly, closing ROADMAP criterion 3.

The two duplicate merge-gate implementations remain intentionally un-consolidated, per CONTEXT.md's `<deferred>` section -- this is an explicit candidate for Phase 190 (Lean, Non-Duplicated Delivery). No blockers identified for downstream plans.

---
*Phase: 187-crash-safe-worktrees-ecosystem-neutrality*
*Completed: 2026-08-18*
