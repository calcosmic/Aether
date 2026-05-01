---
phase: 75-intelligence-core
reviewed: 2026-04-29T19:30:00Z
depth: standard
files_reviewed: 6
files_reviewed_list:
  - cmd/learning.go
  - cmd/learning_test.go
  - cmd/circuit_breaker.go
  - cmd/circuit_breaker_test.go
  - cmd/codex_build_worktree.go
  - cmd/codex_build.go
findings:
  critical: 2
  warning: 4
  info: 2
  total: 8
status: issues_found
---

# Phase 75: Code Review Report

**Reviewed:** 2026-04-29T19:30:00Z
**Depth:** standard
**Files Reviewed:** 6
**Status:** issues_found

## Summary

Reviewed 6 files: the learning/observation CLI commands and tests, the circuit breaker for build resilience and its tests, the worktree-based parallel build dispatch, and the main codex build orchestration.

The circuit breaker is clean, well-tested, and goroutine-safe. The worktree dispatch has solid path traversal guards and state management. However, there are two critical issues: a nested-loop bug in the build tracer that uses O(n^2) to do an identity lookup (functionally wrong in intent), and a silent error swallow in the auto-promote command. Several warnings address error propagation patterns, redundant map lookups, and potential nil dereference in edge cases.

## Critical Issues

### CR-01: Nested loop in tracer block always finds self-match, masking actual output count

**File:** `cmd/codex_build.go:323-331`
**Issue:** The tracer block iterates over `dispatches` in an outer loop, and for each completed dispatch, performs an inner loop over the same `dispatches` slice to find a match by `d.Name == dispatch.Name`. Since `dispatch` is a range variable from the outer loop, the inner loop will always find `dispatch` itself on the first iteration where names match (which is always, since it is comparing the dispatch to itself). The result is that `filesModified` is always set to `len(dispatch.Outputs)` -- the same value it would have been with a direct assignment.

While the current code produces the correct result by accident (it always finds itself), the intent was clearly to look up a different data structure or pre-computed index. This is not a data correctness bug today, but it is O(n^2) for no reason and signals confused logic. If a future refactor changes the dispatch names or the lookup intent, this will silently break.

```go
for _, dispatch := range dispatches {
    filesModified := 0
    if dispatch.Status == "completed" {
        for _, d := range dispatches {          // O(n) inner loop
            if d.Name == dispatch.Name {         // always matches self
                filesModified = len(d.Outputs)
                break
            }
        }
    }
    _ = tracer.LogArtifact(...)
}
```

**Fix:** Replace the inner loop with a direct field access:

```go
for _, dispatch := range dispatches {
    filesModified := 0
    if dispatch.Status == "completed" {
        filesModified = len(dispatch.Outputs)
    }
    _ = tracer.LogArtifact(*updatedState.RunID, "build.worker", map[string]interface{}{
        "worker":         dispatch.Name,
        "status":         dispatch.Status,
        "files_modified": filesModified,
        "summary":        dispatch.Summary,
    })
}
```

### CR-02: `learningPromoteAutoCmd` silently swallows promotion errors

**File:** `cmd/learning.go:160-168`
**Issue:** In the auto-promote loop, when `promoteService.Promote(ctx, obs, "auto-promote")` returns an error, the error is silently discarded via `continue`. The command reports `promoted: 0` without any indication that promotion was attempted and failed. For a data pipeline that promotes observations to instincts, silently dropping failed promotions means learning is lost with no visibility.

```go
for _, obs := range file.Observations {
    eligible, _ := memory.CheckPromotion(obs)
    if eligible {
        result, err := promoteService.Promote(ctx, obs, "auto-promote")
        if err != nil {
            continue  // error silently discarded
        }
        if result.IsNew {
            promoted++
        }
    }
}
```

**Fix:** Track failed promotions and include them in the output so callers know promotion was attempted but failed:

```go
failed := 0
var failures []string
for _, obs := range file.Observations {
    eligible, _ := memory.CheckPromotion(obs)
    if eligible {
        result, err := promoteService.Promote(ctx, obs, "auto-promote")
        if err != nil {
            failed++
            failures = append(failures, fmt.Sprintf("%s: %v", obs.ContentHash, err))
            continue
        }
        if result.IsNew {
            promoted++
        }
    }
}
outputOK(map[string]interface{}{
    "promoted":       promoted,
    "failed":         failed,
    "total_observed": len(file.Observations),
})
```

## Warnings

### WR-01: `runCodexBuildWithOptions` creates `context.Background()` instead of propagating parent context

**File:** `cmd/codex_build.go:252`
**Issue:** At line 252, `newBuildCeremonyEmitter(context.Background(), root, phase)` creates a ceremony emitter with a detached context that ignores cancellation. The same pattern appears at line 289 where `executeCodexBuildDispatches(context.Background(), ...)` is called with `context.Background()` instead of a cancellable context. If the CLI process receives SIGINT/SIGTERM during a long build, the dispatch goroutines and ceremony emitter will not be cancelled. The `context.Background()` passed at line 289 becomes the `ctx` parameter for `dispatchCodexBuildWorkers`, which checks `ctx.Err()` at line 179/362 for early termination -- but since it is `Background()`, it will never be cancelled.

**Fix:** Create a cancellable context from the command's context or a signal handler:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
// Use ctx instead of context.Background() in ceremony and dispatch calls
ceremony := newBuildCeremonyEmitter(ctx, root, phase)
// ...
dispatches, claims, mode, err := executeCodexBuildDispatches(ctx, root, ...)
```

### WR-02: `dispatchCodexBuildWorkersInRepo` returns `nil` error on `updateCodexBuildDispatchRuntimeStatus` failure

**File:** `cmd/codex_build.go:440-442`
**Issue:** When `updateCodexBuildDispatchRuntimeStatus` fails, the function returns `nil, fmt.Errorf(...)`. This propagates the error up and aborts the entire build. However, this status update is a non-critical bookkeeping operation -- failing it should not abort the build. Compare with the worktree path at `codex_build_worktree.go:329-332` which sets `dr.Status = "failed"` but continues.

In the in-repo path, a transient store error during status update will rollback the entire build via `rollbackCodexBuildFailure`, discarding all worker results.

**Fix:** Make the in-repo path consistent with the worktree path -- log the error but do not abort:

```go
if err := updateCodexBuildDispatchRuntimeStatus(dispatch.WorkerName, dr.Status, buildDispatchResultSummary(dispatch, dr)); err != nil {
    // Log but do not abort -- status update is non-critical bookkeeping
    dr.Error = fmt.Errorf("complete worker %s: %w", dispatch.WorkerName, err)
}
```

### WR-03: Potential nil dereference when `dr.WorkerResult` is nil and status is "completed"

**File:** `cmd/codex_build_worktree.go:268-269`
**Issue:** At line 268-269, the code checks `if dr.Status != "completed" || dr.WorkerResult == nil` to set `finalStatus = colony.WorktreeOrphaned`. However, the code at line 261-262 sets `dr.WorkerResult = &result` when `invokeErr == nil`, and then at line 263 sets `dr.Error = result.Error` when `result.Error != nil`. If the invoker returns `result{Status: "completed", Error: someErr}`, then `dr.Status` becomes `"completed"` and `dr.WorkerResult` is non-nil, so `finalStatus` stays `WorktreeMerged`. But then at line 272, `collectWorktreeTouchedPaths` runs on a potentially invalid result.

More critically, at line 288, the code checks `else if dr.WorkerResult != nil` before accessing `dr.WorkerResult.Summary`. If the `syncWorktreeChangesToRoot` call at line 280 succeeds but `dr.WorkerResult` is nil (which should not happen given the line 269 guard, but could happen if a later code path sets `dr.WorkerResult = nil`), the nil check saves it. This is defensive but the real risk is at line 272 where `dr.WorkerResult` is used without a nil check after the `finalStatus` guard:

```go
if dr.Status != "completed" || dr.WorkerResult == nil {
    finalStatus = colony.WorktreeOrphaned
} else {
    touched, touchErr := collectWorktreeTouchedPaths(session.AbsPath, baseline, result)
    // 'result' is from the outer scope, not dr.WorkerResult -- this is correct but confusing
```

The `result` variable used at line 272 comes from the `invokeCodexWorkerWithRuntimeProgress` call at line 255, which is in the same scope. This is technically correct since `result` is only valid when `invokeErr == nil` (line 257 guard), but the pattern is fragile and easy to break during refactoring.

**Fix:** Use `dr.WorkerResult` consistently instead of the outer `result` variable, and add a nil guard:

```go
if dr.Status != "completed" || dr.WorkerResult == nil {
    finalStatus = colony.WorktreeOrphaned
} else {
    touched, touchErr := collectWorktreeTouchedPaths(session.AbsPath, baseline, *dr.WorkerResult)
```

### WR-04: `cleanupBuildWorktrees` uses `store.SaveJSON` instead of `store.UpdateJSONAtomically`

**File:** `cmd/codex_build_worktree.go:789`
**Issue:** `cleanupBuildWorktrees` loads the colony state, modifies it, and saves it back with `store.SaveJSON`. This is a non-atomic read-modify-write pattern. If another process (e.g., another goroutine in the same build) modifies `COLONY_STATE.json` between the load at line 756 and the save at line 789, those changes will be lost. The same issue exists in `gcOrphanedWorktrees` at line 829. Compare with `runCodexBuildWithOptions` which correctly uses `store.UpdateJSONAtomically` at line 305.

**Fix:** Use `store.UpdateJSONAtomically` for consistency and safety:

```go
var cleanedCount, orphanedCount int
if err := store.UpdateJSONAtomically("COLONY_STATE.json", &state, func() error {
    // ... cleanup logic modifying state.Worktrees
    return nil
}); err != nil {
    return 0, 0, err
}
return cleanedCount, orphanedCount, nil
```

## Info

### IN-01: Custom `max` function shadows Go 1.21+ builtin

**File:** `cmd/codex_build.go:1645-1650`
**Issue:** Go 1.21 introduced a built-in `max` function for ordered types. The project targets Go 1.26 (per `go version`). The custom `max(a, b int) int` at line 1645 shadows the builtin. While this compiles and works correctly, it is unnecessary code that could confuse developers who expect the builtin behavior.

**Fix:** Remove the custom `max` function and use the builtin directly.

### IN-02: `var _ = timeoutCtx` compile-time check is a no-op

**File:** `cmd/learning.go:250`
**Issue:** The line `var _ = timeoutCtx` is intended as a compile-time usage check, but `timeoutCtx` is a function (not an interface). The Go compiler will error on undefined functions regardless. This line does nothing and is dead code.

**Fix:** Remove line 250.

---

_Reviewed: 2026-04-29T19:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
