---
phase: 50-repair-pipeline
reviewed: 2026-04-25T00:00:00Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - cmd/recover.go
  - cmd/recover_visuals.go
  - cmd/recover_repair.go
  - cmd/recover_test.go
findings:
  critical: 3
  warning: 3
  info: 1
  total: 7
status: issues_found
---

# Phase 50: Code Review Report

**Reviewed:** 2026-04-25T00:00:00Z
**Depth:** standard
**Files Reviewed:** 4
**Status:** issues_found

## Summary

Reviewed the repair pipeline implementation across four files: the command entry point (`recover.go`), visual output rendering (`recover_visuals.go`), the repair orchestrator and seven repair dispatchers (`recover_repair.go`), and the test suite (`recover_test.go`).

The scanner and detection logic is solid with good coverage. However, the repair pipeline has three significant correctness bugs: the rollback mechanism undoes successful repairs when any single repair fails, the dirty worktree repair always rewrites colony state even for git-operation sub-types that do not modify state, and the JSON output rendering uses brittle string manipulation that risks producing invalid JSON.

## Critical Issues

### CR-01: Rollback undoes successful repairs when any single repair fails

**File:** `cmd/recover_repair.go:119-127`
**Issue:** When `result.Failed > 0`, the code rolls back to the pre-repair backup. This restores the entire data directory to its state before any repairs ran. If repair A succeeded (e.g., resetting stale spawns) and repair B failed (e.g., bad manifest recovery), the rollback silently undoes A's fix. The user sees `result.Succeeded > 0` in the output, but their data has been reverted. The `RepairResult` returned to the caller still reports successful repairs that were actually rolled back, making the output misleading.

**Fix:**
The repair orchestrator should either:
1. Not roll back on partial failure -- only roll back if the first repair fails (since subsequent repairs may depend on prior ones being in place), or
2. After rollback, reset `result.Succeeded` to 0 and adjust the records to reflect the rollback state, and inform the user that all repairs were reverted.

```go
// Option 2: After rollback, update result to reflect reality
if result.Failed > 0 {
    if rollbackErr := restoreFromBackup(backupPath, dataDir); rollbackErr != nil {
        fmt.Fprintf(os.Stderr, "  [warn] rollback failed: %v\n", rollbackErr)
    } else {
        // Rollback succeeded -- mark all previously-successful repairs as rolled back
        for i := range result.Repairs {
            if result.Repairs[i].Success {
                result.Repairs[i].Action += " (rolled back)"
            }
        }
        result.Succeeded = 0
    }
}
```

### CR-02: `repairDirtyWorktree` always rewrites colony state for sub-types that do not modify it

**File:** `cmd/recover_repair.go:607-673`
**Issue:** The function unconditionally marshals and writes `COLONY_STATE.json` at lines 661-670 regardless of which sub-type was handled. For the "uncommitted change" (git stash) and "Orphan branch" (git branch -D) sub-types, the function never mutates `state` at all -- it only runs a git command. The state write is a no-op that needlessly risks data loss: if two repairs are running in sequence and the first modifies state, the second's no-op write could clobber the first's changes if it re-read stale state. More critically, the "Orphan branch" sub-type sets `issue.File` to a branch name (not a file path), so the state write is completely irrelevant to the git operation.

**Fix:**
Only write state back when the state was actually modified. Move the state write into the case branches that modify it:

```go
switch {
case strings.Contains(msg, "state-disk mismatch") || strings.Contains(msg, "not in git worktree list"):
    var remaining []colony.WorktreeEntry
    for _, wt := range state.Worktrees {
        if wt.Path != issue.File {
            remaining = append(remaining, wt)
        }
    }
    state.Worktrees = remaining
    record.Action = "remove_orphan_worktree_entry"
    // State was modified -- fall through to write below.

case strings.Contains(msg, "uncommitted change"):
    cmd := exec.Command("git", "-C", issue.File, "stash", "--include-untracked")
    if output, err := cmd.CombinedOutput(); err != nil {
        record.Error = fmt.Sprintf("git stash failed: %v: %s", err, string(output))
        return record
    }
    record.Action = "stash_worktree_changes"
    record.Success = true
    return record // No state modification -- return early.

case strings.Contains(msg, "Orphan branch"):
    branchName := issue.File
    cmd := exec.Command("git", "branch", "-D", branchName)
    if output, err := cmd.CombinedOutput(); err != nil {
        record.Error = fmt.Sprintf("git branch -D failed: %v: %s", err, string(output))
        return record
    }
    record.Action = "delete_orphan_branch"
    record.Success = true
    return record // No state modification -- return early.

default:
    record.Error = "unrecognized dirty_worktree sub-type"
    return record
}
// Only reaches here for state-modifying sub-types.
encoded, err := json.MarshalIndent(state, "", "  ")
// ... write state ...
```

### CR-03: `renderRecoverJSON` uses fragile string slicing to inject repairs into JSON

**File:** `cmd/recover_visuals.go:246-257`
**Issue:** The code manually strips the last 2 characters from the JSON string (`result[:len(result)-2]`) and concatenates the repairs object. This assumes `json.MarshalIndent` always produces exactly `}\n` as the last two characters. If the marshaller's behavior changes or the output has different trailing whitespace (e.g., on different Go versions or platforms), the string slice produces invalid JSON. The error is silently consumed by callers who parse the JSON.

Additionally, the injected `repairsJSON` is indented at root level (0 spaces) but is being inserted inside the top-level object, creating inconsistent indentation.

**Fix:**
Build the complete output struct before marshalling, or use a two-pass marshal approach:

```go
if repairResult != nil {
    output.Repairs = &repairsSummary{
        Attempted: repairResult.Attempted,
        Succeeded: repairResult.Succeeded,
        Failed:    repairResult.Failed,
        Skipped:   repairResult.Skipped,
        Details:   repairResult.Repairs,
    }
}

data, err := json.MarshalIndent(output, "", "  ")
if err != nil {
    return fmt.Sprintf(`{"error": "failed to marshal report: %v"}`, err)
}
return string(data) + "\n"
```

Where `repairsSummary` is a proper struct (or an anonymous struct) that includes the repairs data.

## Warnings

### WR-01: `recoverFixHint` is dead code

**File:** `cmd/recover_visuals.go:131-142`
**Issue:** The function `recoverFixHint` is defined but never called anywhere in the codebase. It appears to have been intended for use in `writeRecoverIssueLine` or `recoverNextStep` but was replaced by inline logic.

**Fix:** Remove the unused function to reduce maintenance burden and avoid confusion.

### WR-02: Rollback result does not reflect rolled-back state to caller

**File:** `cmd/recover_repair.go:119-127`
**Issue:** After a successful rollback, the function returns the original `result` with `Succeeded > 0`, making it appear to the caller that some repairs succeeded. The caller (`runRecover` in `recover.go`) then re-scans and renders the post-repair state, but the `repairResult` it passes to the renderer is misleading. The repair log will show successful repairs that were actually undone.

**Fix:** After a successful rollback, zero out `result.Succeeded` and annotate the repair records to indicate they were rolled back. Alternatively, return a separate error indicating that repairs were attempted and rolled back.

### WR-03: `performRecoverRepairs` error return masks repair outcome

**File:** `cmd/recover_repair.go:25-128`
**Issue:** The function signature `(*RepairResult, error)` suggests that `error` is returned when something goes wrong. However, `error` is only returned when backup fails (line 29). When individual repairs fail, `error` is nil and the failures are captured in `result.Failed`. This is a valid pattern, but the caller in `recover.go:56-68` handles `err != nil` as "repair failed" and renders a partial result. If backup succeeds but all repairs fail, the caller sees `err == nil` and proceeds to render as if everything is fine (with rollback having reverted changes). The control flow is confusing and could lead to maintenance bugs.

**Fix:** Document the contract clearly in the function docstring: "error is only non-nil when backup fails; individual repair failures are reflected in result.Failed."

## Info

### IN-01: Test for rollback (`TestRepairAtomicity_RollbackOnFailure`) is conditionally effective

**File:** `cmd/recover_test.go:1390-1426`
**Issue:** The rollback test at line 1390 depends on `repairBadManifest` failing when it encounters a non-existent or empty manifest file at the path referenced by `issue.File`. The test creates issues with `File: "build/phase-1/manifest.json"` but never creates this file. If `findLastValidJSON` on an empty read returns nil and the function then tries to remove the file (which does not exist), the repair could fail. However, the repair first reads the file with `os.ReadFile`, which would fail since the file does not exist, producing an error record. This makes the test pass, but the test does not verify the rollback itself -- it only checks `if result.Failed > 0` before asserting the state was rolled back. If no repairs fail (edge case), the rollback assertion is silently skipped.

**Fix:** The test should fail if `result.Failed == 0` since the entire test premise is that a failure triggers rollback. Add:
```go
if result.Failed == 0 {
    t.Fatal("expected at least 1 failed repair to trigger rollback")
}
```

---

_Reviewed: 2026-04-25T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
