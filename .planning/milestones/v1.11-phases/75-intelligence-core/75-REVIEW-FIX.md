---
phase: 75-intelligence-core
fixed_at: 2026-04-29T19:45:00Z
review_path: .planning/phases/75-intelligence-core/75-REVIEW.md
iteration: 1
findings_in_scope: 6
fixed: 6
skipped: 0
status: all_fixed
---

# Phase 75: Code Review Fix Report

**Fixed at:** 2026-04-29T19:45:00Z
**Source review:** .planning/phases/75-intelligence-core/75-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 6 (2 critical, 4 warning)
- Fixed: 6
- Skipped: 0

## Fixed Issues

### CR-01: Nested loop in tracer block always finds self-match, masking actual output count

**Files modified:** `cmd/codex_build.go`
**Commit:** `09574916`
**Applied fix:** Replaced the O(n^2) inner loop that always matched `dispatch` to itself with a direct `len(dispatch.Outputs)` field access. The nested loop was functionally correct by accident but signaled confused intent and wasted cycles.

### CR-02: `learningPromoteAutoCmd` silently swallows promotion errors

**Files modified:** `cmd/learning.go`
**Commit:** `78c6c6e5`
**Applied fix:** Added `failed` counter and `failures` slice to track promotion errors. The output now includes a `failed` count alongside `promoted` and `total_observed`, so callers know when promotion was attempted but failed.

### WR-01: `runCodexBuildWithOptions` creates `context.Background()` instead of propagating parent context

**Files modified:** `cmd/codex_build.go`
**Commit:** `dfc45065`
**Applied fix:** Added `os`, `os/signal`, `syscall` imports and created a `signal.NotifyContext` at the start of the build ceremony section. Both `newBuildCeremonyEmitter` and `executeCodexBuildDispatches` now receive the signal-aware context instead of `context.Background()`, allowing SIGINT/SIGTERM to properly cancel dispatch goroutines.

### WR-02: `dispatchCodexBuildWorkersInRepo` returns error on non-critical status update failure

**Files modified:** `cmd/codex_build_worktree.go`
**Commit:** `3dc584d9`
**Applied fix:** Made the in-repo path consistent with the worktree path. Instead of returning an error (which aborts the entire build), the status update failure now sets `dr.Status = "failed"` and `dr.Error` on the dispatch result, allowing the build to continue with other workers.

### WR-03: Potential nil dereference when using outer `result` variable instead of `dr.WorkerResult`

**Files modified:** `cmd/codex_build_worktree.go`
**Commit:** `77726c84`
**Applied fix:** Changed `collectWorktreeTouchedPaths(session.AbsPath, baseline, result)` to use `*dr.WorkerResult` instead of the outer `result` variable. This makes the code consistently reference the stored result and eliminates a fragile coupling to the outer scope.

### WR-04: `cleanupBuildWorktrees` and `gcOrphanedWorktrees` use non-atomic read-modify-write

**Files modified:** `cmd/codex_build_worktree.go`
**Commit:** `9b33cc94`
**Applied fix:** Converted both `cleanupBuildWorktrees` and `gcOrphanedWorktrees` from the `LoadJSON` + modify + `SaveJSON` pattern to `store.UpdateJSONAtomically`, which uses file locking and atomic writes. Added nil guard for `store`. This prevents lost writes if another goroutine modifies `COLONY_STATE.json` between load and save.

---

_Fixed: 2026-04-29T19:45:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
