---
phase: 75-intelligence-core
reviewed: 2026-04-29T00:00:00Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - .aether/docs/command-playbooks/continue-advance.md
  - .aether/docs/command-playbooks/continue-full.md
  - cmd/ceremony_emitter.go
  - cmd/circuit_breaker_test.go
  - cmd/circuit_breaker.go
  - cmd/codex_build_worktree.go
  - cmd/codex_build.go
  - cmd/codex_workflow_cmds.go
  - cmd/learning.go
  - pkg/events/ceremony.go
findings:
  critical: 1
  warning: 5
  info: 3
  total: 9
status: issues_found
---

# Phase 75: Code Review Report

**Reviewed:** 2026-04-29T00:00:00Z
**Depth:** standard
**Files Reviewed:** 10
**Status:** issues_found

## Summary

Reviewed 10 files spanning the intelligence core domain: ceremony event emission, circuit breaker for build resilience, worktree-based parallel dispatch, build orchestration, workflow commands (seal, plan, build, continue, signals), learning/observation pipeline, and ceremony topic constants.

The circuit breaker implementation is well-structured with good concurrency safety and comprehensive tests. The ceremony emitter provides a clean abstraction over event publishing. The worktree parallel dispatch has solid path traversal guards and state management. However, there are several logic inconsistencies, a goroutine safety issue in the ceremony emitter, a semantic contradiction in the circuit breaker's status reporting, and an indentation artifact in the ceremony topics.

## Critical Issues

### CR-01: Race condition on global `activeBuildCeremony` during concurrent build wave dispatch

**File:** `cmd/ceremony_emitter.go:36-72`
**Issue:** The `activeBuildCeremony` global variable uses a `sync.RWMutex` for access, but the `setActiveBuildCeremony` function restores the previous value in a deferred closure. When the worktree-based dispatch (`dispatchCodexBuildWorkers` in `codex_build_worktree.go`) runs multiple waves concurrently via goroutines, each wave's results call `emitBuildCeremonyWorkerFinished` which reads `currentBuildCeremony()`. If a second build starts before the first completes, the restore closure from `setActiveBuildCeremony` will overwrite the new build's ceremony, causing events from the new build to be lost or published to the wrong narrator.

More critically, in the worktree dispatch path, the `emitBuildCeremonyWaveEnd` call at line 340 happens after `wg.Wait()`, but `emitBuildCeremonyWorkerFinished` at line 334 happens inside the goroutine -- between worker dispatches across waves, the ceremony is stable, but if two separate `runCodexBuildWithOptions` calls overlap (e.g., user triggers two builds), the global swap/restore pattern breaks.

**Fix:** The simplest fix is to pass the ceremony emitter as a parameter through the dispatch chain instead of relying on a global. Alternatively, scope the ceremony emitter per-run-handle so concurrent builds cannot clobber each other:

```go
// Instead of global activeBuildCeremony, pass via context or closure
func dispatchCodexBuildWorkers(ctx context.Context, root string, phase colony.Phase, dispatches []codex.WorkerDispatch, invoker codex.WorkerInvoker, startedAt time.Time, parallelMode colony.ParallelMode, cb *CircuitBreaker, ceremony *buildCeremonyEmitter) ([]codex.DispatchResult, error) {
    // use ceremony directly instead of currentBuildCeremony()
}
```

## Warnings

### WR-01: Circuit breaker reports "completed" wave status even when all workers failed

**File:** `cmd/ceremony_emitter.go:458-482`
**Issue:** The `emitBuildCeremonyWaveEnd` function unconditionally sets `waveStatus` to `"completed"` at line 472, then only overrides it to `"blocked"` if `completed < len(waveResults) || len(blockers) > 0`. This means if all workers in a wave failed (completed == 0) but there are zero blockers (e.g., failures without error messages), the wave is reported as `"completed"`.

Looking at the dispatch path in `codex_build_worktree.go:317`, when `dr.Status` is empty it is set to `"failed"`, but the error is not always populated into `blockers` -- specifically in the worktree path when `cb.RecordFailure` fires at line 325, there is no error, so no blocker is appended. Similarly, in the `codex_build.go:438` in-repo path, `cb.RecordFailure` happens without adding a blocker. The wave status would then be "completed" despite all workers having failed.

**Fix:** Check for the failed status explicitly in addition to blockers:

```go
failedCount := 0
for _, result := range results {
    if result.Status == "completed" {
        completed++
    } else if result.Status == "failed" {
        failedCount++
    }
    // ... collect blockers as before
}
waveStatus := "completed"
if completed < len(waveResults) || failedCount > 0 || len(blockers) > 0 {
    waveStatus = "blocked"
}
```

### WR-02: `ceremonyStepCompleted` contradicts `ceremonyStepStatus` for "spawned" and "planned"

**File:** `cmd/ceremony_emitter.go:227-242`
**Issue:** `ceremonyStepStatus` maps empty string, `"spawned"`, and `"planned"` to `"completed"` (line 229-230). However, `ceremonyStepCompleted` does NOT treat `"spawned"` or `"planned"` as completed -- it only treats `""`, `"completed"`, `"manually-reconciled"`, and `"skipped"` as completed (line 237-238).

This means in `emitLifecycleCeremonySequence` at line 188, a step with status `"spawned"` will be counted as `completed` (via `ceremonyStepCompleted` returning false because "spawned" is not in the completed set), but the ceremony event will report status `"completed"` (via `ceremonyStepStatus` returning "completed"). The `completed` counter at line 189 increments only when `ceremonyStepCompleted` returns true, so `"spawned"` steps are NOT counted as completed for the wave-end summary, but their individual spawn events say `"completed"`. This is a semantic inconsistency.

**Fix:** Make both functions agree. Either `"spawned"` should be counted as completed in both, or neither:

```go
func ceremonyStepCompleted(status string) bool {
    switch strings.TrimSpace(status) {
    case "", "completed", "manually-reconciled", "skipped", "spawned", "planned":
        return true
    default:
        return false
    }
}
```

Or, if "spawned" should not be "completed", fix `ceremonyStepStatus` to not map it that way.

### WR-03: `emitLifecycleCeremonySequence` mutates its input slice

**File:** `cmd/ceremony_emitter.go:161-166`
**Issue:** The function modifies `step.Wave` when `wave <= 0` (setting it to 1). Since `steps` is a slice passed by value but the elements are pointers to structs (actually they are value-typed `lifecycleCeremonyStep` structs, but slices share backing arrays), the mutation at line 166 modifies the caller's data. The callers in `emitPlanCeremonyDispatchSequence`, `emitColonizeCeremonyDispatchSequence`, and `emitContinueCeremonyFlowSequence` all pass locally-constructed slices, so in practice the mutation is harmless -- but it is a footgun if any caller reuses steps or expects them to be immutable.

**Fix:** Either document the mutation clearly, or copy the step before modifying:

```go
stepCopy := step
if stepCopy.Wave <= 0 {
    stepCopy.Wave = 1
}
waves[stepCopy.Wave] = append(waves[stepCopy.Wave], stepCopy)
```

### WR-04: Duplicate hive promotion loop in `codex_workflow_cmds.go` seal command

**File:** `cmd/codex_workflow_cmds.go:330-352`
**Issue:** The seal command iterates over `entries` twice with the same condition (`entry.Confidence >= 0.8 && entry.Action != ""`): once for local QUEEN.md promotion (lines 332-335) and once for hive promotion (lines 337-351). The `hiveEligibleCount` is incremented inside the second loop, but `promotedInstinctNames` is only appended in the first loop. If `promoteInstinctLocal` fails for an entry but `promoteToHive` succeeds, the instinct is promoted to the hive but not listed in `promotedInstinctNames`. This creates an inconsistency between what is reported as "promoted" and what actually reached the hive.

This is not a data loss bug (hive promotion succeeds independently), but the `Promoted Instincts` list in CROWNED-ANTHILL.md will be incomplete.

**Fix:** Track hive promotion results independently, or move `hiveEligibleCount` into the first loop and iterate only once:

```go
for _, entry := range entries {
    if entry.Confidence >= 0.8 && entry.Action != "" {
        hiveEligibleCount++
        if err := promoteInstinctLocal(store, entry.ID, entry.Action); err == nil {
            promotedInstinctNames = append(promotedInstinctNames, entry.ID)
        }
        // hive promotion (non-blocking)
        domain := entry.Domain
        if domain == "" {
            domain = "general"
        }
        if err := promoteToHive(entry.Action, domain, repoName, entry.Confidence); err != nil {
            log.Printf("seal: hive-promote failed for %s: %v", entry.ID, err)
            hivePromotionFailures++
        } else {
            hivePromotedCount++
        }
    }
}
```

### WR-05: `learning.go` error handling returns nil instead of propagating errors

**File:** `cmd/learning.go:17-20, 50-53, 73-76, 143-145, 187-190, 213-216`
**Issue:** Multiple commands return `nil` error after calling `outputErrorMessage` or `outputError`. For example, at line 19-20: when `store == nil`, the function returns `nil` error. Similarly, when `CaptureWithTrust` fails at line 51, the function returns `nil` error. This means the cobra command framework considers the command successful even when it failed. For CLI tools that may be called from scripts, this means exit code 0 on failure, which is incorrect.

This pattern is consistent across all four commands in the file. It appears to be a project convention (the error is communicated via `outputError`'s JSON format), but it means any caller using `RunE` error propagation or shell `$?` will see success when the operation actually failed.

**Fix:** If the project convention is intentional, this is not a bug per se. However, for commands that may be called from shell scripts, consider returning an actual error for hard failures:

```go
if store == nil {
    outputErrorMessage("no store initialized")
    return fmt.Errorf("no store initialized")
}
```

## Info

### IN-01: Inconsistent indentation in `CeremonyTopics()` function

**File:** `pkg/events/ceremony.go:73`
**Issue:** Line 73 has an extra tab indent compared to surrounding lines:
```go
			CeremonyTopicBuildCircuitBreak,  // extra indent
```
This is a cosmetic issue but suggests a merge or copy-paste artifact.

**Fix:** Remove the extra tab to match surrounding lines:
```go
		CeremonyTopicBuildCircuitBreak,
```

### IN-02: Compile-time check `var _ = timeoutCtx` is unnecessary

**File:** `cmd/learning.go:250`
**Issue:** The line `var _ = timeoutCtx` is a compile-time interface satisfaction check, but `timeoutCtx` is a function, not an interface. This pattern is typically used for interfaces (e.g., `var _ io.Reader = ...`). For a function, the compiler will error if the function is undefined regardless. This line is dead code.

**Fix:** Remove the line.

### IN-03: `continue-advance.md` and `continue-full.md` contain significant content duplication

**File:** `.aether/docs/command-playbooks/continue-advance.md` and `.aether/docs/command-playbooks/continue-full.md`
**Issue:** The `continue-full.md` file contains the complete continue flow (1737 lines) while `continue-advance.md` contains Steps 2.0.4 through 2.1.5 (704 lines). These two files share substantial duplicated content (Steps 2.1 through 2.1.5 appear in both with identical or near-identical text). If one is updated without the other, they will diverge silently.

**Fix:** Extract the shared sections into a separate playbook (e.g., `continue-pheromones.md`) and have both files reference it, or document clearly which file is authoritative and mark the other as a derived/split file.

---

_Reviewed: 2026-04-29T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
