---
phase: 89-gate-self-healing-smart-planning
reviewed: 2026-05-02T19:00:00Z
depth: standard
files_reviewed: 6
files_reviewed_list:
  - cmd/oracle_loop.go
  - cmd/oracle_loop_test.go
  - cmd/init_ceremony.go
  - cmd/init_ceremony_test.go
  - cmd/status.go
  - cmd/status_test.go
findings:
  critical: 1
  warning: 4
  info: 3
  total: 8
status: issues_found
---

# Phase 89: Code Review Report

**Reviewed:** 2026-05-02T19:00:00Z
**Depth:** standard
**Files Reviewed:** 6
**Status:** issues_found

## Summary

Reviewed six files: the oracle loop engine and its tests, the init ceremony command and its tests, and the status dashboard with its tests. One critical bug was found in the init ceremony reject-restart flow: when a user rejects a launch brief, the ceremony loops with an empty goal instead of prompting for a new one. Several warnings cover a nil pointer risk in status rendering, redundant JSON unmarshaling, fragile iteration counter handling, and an edge case in smart question selection. Test files are generally well-structured with good coverage of the core flows but do not test the reject-restart cycle.

## Critical Issues

### CR-01: Init ceremony reject-restart passes empty goal to research

**File:** `cmd/init_ceremony.go:296-310`
**Issue:** When the user rejects the launch brief (choice 3), the code sets `goal = ""` and continues the outer ceremony loop. The message printed to stderr says "Returning to goal prompt" but no actual prompt is displayed to capture a new goal. The outer loop immediately calls `runCeremonyResearch("", target)` with an empty goal string. This propagates down to `initResearchCmd` with `--goal ""`, causing the research phase to run with a blank goal. The result is a meaningless charter and brief presented to the user. The cycle repeats forever if the user keeps rejecting, or produces garbage research output if they approve. The message promises a "goal prompt" that never appears.
**Fix:**
```go
case 3: // Reject
    fmt.Fprintln(os.Stderr, "  Brief rejected. Returning to goal prompt.")
    newGoal := promptString("Enter a new goal")
    if strings.TrimSpace(newGoal) == "" {
        fmt.Fprintln(os.Stderr, "  No goal provided. Exiting ceremony.")
        return nil
    }
    goal = newGoal
    break
```

## Warnings

### WR-01: renderDashboard dereferences Goal pointer without nil check

**File:** `cmd/status.go:484`
**Issue:** `renderDashboard` dereferences `state.Goal` at `goal := *state.Goal` without checking for nil. The caller `statusCmd.RunE` calls `loadActiveColonyState` which validates that `state.Goal` is non-nil before returning, so in the current normal flow this is safe. However, `renderDashboard` is an exported helper that could be called from other contexts where the Goal pointer is nil. Any future caller passing a state with a nil Goal will cause a panic.
**Fix:**
```go
goal := ""
if state.Goal != nil {
    goal = *state.Goal
}
if len(goal) > 60 {
    goal = goal[:57] + "..."
}
```

### WR-02: renderGateStatusSection unmarshals the same JSON twice

**File:** `cmd/status.go:213-231`
**Issue:** When the gate results JSON starts with `{` (wrapper format), the code unmarshals it once at lines 214-218 to extract `Results`, then unmarshals the same `raw` bytes again at lines 229-231 to extract `Attempts`. This is redundant work. If the `gateResultsFile` struct changes, both unmarshal sites must be kept in sync.
**Fix:** Unmarshal once and use both fields from the same variable:
```go
var results []GateCheckResult
unblockAttempts := 0
if raw[0] == '{' {
    var wrapped gateResultsFile
    if err := json.Unmarshal(raw, &wrapped); err != nil {
        return ""
    }
    results = wrapped.Results
    unblockAttempts = wrapped.Attempts
} else if err := json.Unmarshal(raw, &results); err != nil {
    return ""
}
```

### WR-03: Oracle loop iteration counter overwritten with iterationsRun

**File:** `cmd/oracle_loop.go:570`
**Issue:** After a successful iteration, `state.Iteration` is overwritten with `iterationsRun` at line 570. While these two counters currently stay in sync (both increment once per iteration), the overwrite is fragile. `state.Iteration` is incremented at line 432, while `iterationsRun` is incremented at line 461. They represent different concerns: persisted state versus local loop tracking. The blanket overwrite at line 570 means any future code path that increments one but not the other would silently mask the discrepancy rather than surfacing it.
**Fix:** Remove the overwrite since `state.Iteration` was already correctly incremented at line 432:
```go
// Remove line 570: state.Iteration = iterationsRun
state.OverallConfidence = oracleOverallConfidence(plan)
```

### WR-04: selectOracleQuestionSmart returns synthetic question when all answered but confidence is below target

**File:** `cmd/oracle_loop.go:2408-2411`
**Issue:** When all questions have "answered" status, `selectOracleQuestionSmart` returns a synthetic `oracleQuestion` with `ID: "iteration-N"`. This question does not exist in the plan. If the loop reaches another iteration (because `oracleReadyForCompletion` at line 600 returns false -- possible when questions are "answered" at low confidence values), the worker response for this synthetic question will fail in `applyOracleWorkerResponse` with "references unknown question." This causes the oracle loop to block with a worker_error rather than continuing to deepen its research.
**Fix:** Add a guard in the oracle loop after question selection:
```go
target := selectOracleQuestionSmart(plan, state)
if strings.HasPrefix(target.ID, "iteration-") && strings.Contains(target.Text, "All oracle questions") {
    return finalizeOracleLoop(paths, state, plan, detectedType, languages, frameworks,
        iterationsRun, "max_iterations_reached", "all_answered_below_target", "aether oracle status")
}
```

## Info

### IN-01: Mixed indentation in buildBriefInformedQuestions

**File:** `cmd/oracle_loop.go:1532-1539`
**Issue:** Lines 1532-1539 use leading spaces for indentation while the rest of the function uses tabs. This creates inconsistent whitespace in the source file.
**Fix:** Convert the space-indented lines to tab indentation consistent with the rest of the file.

### IN-02: countConstraints always returns zero

**File:** `cmd/status.go:998-1002`
**Issue:** `countConstraints` loads `constraints.json` but always returns `(0, 0)` with a comment noting the file is currently an empty object. The dashboard always displays "Focus: 0 areas | Avoid: 0 patterns" even when FOCUS/REDIRECT signals exist in pheromones. The actual counts are available from the pheromone system but are not wired through this function.
**Fix:** Either wire this to the pheromone signal counts (which are already displayed separately in the "Active Pheromones" section) or remove the misleading zero-count line from the dashboard.

### IN-03: init_ceremony.go uses goto for control flow

**File:** `cmd/init_ceremony.go:464-477`
**Issue:** `createCeremonyColony` uses `goto createFreshColony` to jump past the existing-colony check block. While functional, Go idiom generally prefers structured control flow (if/else, early returns) over goto. The goto is used to skip over error-return paths for active colonies.
**Fix:** Consider refactoring to use an `else` block or extract a helper function to avoid goto, improving readability.

---

_Reviewed: 2026-05-02T19:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
