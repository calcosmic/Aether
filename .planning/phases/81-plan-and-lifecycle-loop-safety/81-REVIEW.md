---
phase: 81-plan-and-lifecycle-loop-safety
reviewed: 2026-04-30T00:00:00Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - cmd/codex_plan.go
  - cmd/codex_workflow_cmds.go
  - cmd/entomb_cmd.go
  - cmd/recovery_engine.go
  - cmd/recovery_engine_test.go
  - cmd/session_flow_cmds.go
  - cmd/status.go
  - cmd/status_test.go
  - pkg/colony/cycle.go
  - pkg/colony/cycle_test.go
findings:
  critical: 1
  warning: 4
  info: 2
  total: 7
status: issues_found
---

# Phase 81: Code Review Report

**Reviewed:** 2026-04-30
**Depth:** standard
**Files Reviewed:** 10
**Status:** issues_found

## Summary

This phase introduces two safety mechanisms: (1) a task dependency cycle detector (LOOP-04) in `pkg/colony/cycle.go` wired into `aether plan`, and (2) a recovery engine (LOOP-05) in `cmd/recovery_engine.go` that replaces bare `outputError` calls with structured recovery menus on `seal`, `entomb`, `resume`, and `status`. The implementation is well-structured with good test coverage. However, there is a JSON injection vulnerability in the recovery menu's JSON output path, a missed `MissingDepError` case in the plan command's cycle check, and several moderate quality issues.

## Critical Issues

### CR-01: JSON injection in recovery menu error output

**File:** `cmd/recovery_engine.go:224-226`
**Issue:** The `renderRecoveryMenu` function writes the error message into a JSON envelope using manual string formatting. While `jsonEscape` is applied to `errMsg`, the `detailBytes` (from `json.Marshal`) are interpolated directly into the format string via `%s`. This is safe for the current code since `detailBytes` comes from `json.Marshal` and is already valid JSON. However, the overall approach of manually constructing JSON via `fmt.Fprintf` is fragile -- any future caller passing untrusted content into the `details` parameter could produce malformed JSON or inject unexpected keys.

The more immediate problem is that `jsonEscape` strips surrounding quotes from `json.Marshal` output but does not handle all edge cases. If `errMsg` contains characters like `\n`, `\t`, or `\uXXXX` sequences that `json.Marshal` double-escapes, the result could be incorrect. More critically, `jsonEscape` discards the error return from `json.Marshal` (line 253: `b, _ := json.Marshal(s)`), which means a nil input would produce `"null"` and the function would return the literal string `null` instead of an empty string.

**Fix:**
```go
// Replace the manual JSON construction with proper encoding:
func renderRecoveryMenu(failedCmd string, errMsg string, details interface{}) string {
    options := recoveryOptionsForCommand(failedCmd, errMsg)

    if shouldRenderVisualOutput(stderr) {
        return buildVisualRecoveryMenu(failedCmd, errMsg, options)
    }

    // JSON mode: output error envelope with recovery_options
    recoveryDetails := map[string]interface{}{
        "recovery_options": options,
    }
    if details != nil {
        recoveryDetails["original_details"] = details
    }
    envelope := map[string]interface{}{
        "ok":     false,
        "error":  errMsg,
        "code":   1,
        "details": recoveryDetails,
    }
    detailBytes, err := json.Marshal(envelope)
    if err != nil {
        detailBytes = []byte(fmt.Sprintf(`{"ok":false,"error":"recovery render failed","code":1,"details":{}}`))
    }
    fmt.Fprintf(stderr, "%s\n", string(detailBytes))
    return ""
}
```

This eliminates the manual `jsonEscape` function entirely and uses `json.Marshal` for the entire envelope, which is both safer and simpler.

## Warnings

### WR-01: MissingDepError not handled in plan cycle validation

**File:** `cmd/codex_plan.go:353-360`
**Issue:** When `colony.DetectCycles` returns a `MissingDepError` (e.g., a task depends on a nonexistent task ID like "9.9"), the plan command wraps it with `"plan dependency validation failed"` rather than providing a specific, actionable message. A `MissingDepError` is not a cycle -- it is a dangling reference. The user gets a generic error instead of being told which task has a broken dependency and which dependency is missing.

**Fix:**
```go
if err := colony.DetectCycles(phases); err != nil {
    var cycleErr *colony.CycleError
    if errors.As(err, &cycleErr) {
        return nil, fmt.Errorf("plan contains circular dependency: %s. Remove the cycle and regenerate the plan", cycleErr)
    }
    var missingErr *colony.MissingDepError
    if errors.As(err, &missingErr) {
        return nil, fmt.Errorf("plan contains invalid dependency: %s. Fix or remove the dependency and regenerate the plan", missingErr)
    }
    return nil, fmt.Errorf("plan dependency validation failed: %w", err)
}
```

### WR-02: Test helper `errorAs` reimplements `errors.As` without pointer unwrapping

**File:** `pkg/colony/cycle_test.go:170-188`
**Issue:** The `errorAs` helper reimplements `errors.As` using direct type assertions. This works for the current cases where errors are concrete `*CycleError` or `*MissingDepError` values, but it will silently fail if the error is wrapped (e.g., `fmt.Errorf("...: %w", &CycleError{...})`). The standard library's `errors.As` handles wrapped errors correctly via interface unwrapping. Since the production code in `codex_plan.go` already imports and uses `errors.As`, the test code should too for consistency and correctness.

**Fix:**
```go
// In cycle_test.go, replace the custom errorAs with:
import "errors"

func errorAs(err error, target interface{}) bool {
    return errors.As(err, target)
}
```

### WR-03: `countTotalPlans` is a misleading name that aliases `completedPhaseCount`

**File:** `cmd/entomb_cmd.go:657-659`
**Issue:** The function `countTotalPlans` is documented as "count total plans across all phases" but simply delegates to `completedPhaseCount`. The comment even acknowledges this mismatch: "Since Phase struct doesn't have PlanCount, we count completed phases as proxy." This is used in `updateRegistryFinalStats` as the `PlanCount` field, which means the registry records the number of completed phases, not the number of plans. If any downstream consumer interprets `PlanCount` as the actual number of plan generations (which could differ from completed phases if the plan was refreshed), this is a data integrity issue.

**Fix:** Rename the function and the registry field to match what is actually being counted:
```go
// In entomb_cmd.go, update the registry call:
updateRegistryFinalStats(repoPath, registryFinalStats{
    PhaseCount:    len(state.Plan.Phases),
    PlanCount:     completedPhaseCount(state), // already correct semantically
    // ...
})
```
And either remove the `countTotalPlans` wrapper entirely (call `completedPhaseCount` directly) or rename it to something that does not imply "total plans generated."

### WR-04: Recovery engine only covers four lifecycle commands -- others still use bare `outputError`

**File:** `cmd/recovery_engine.go:68-113` (candidates map)
**Issue:** The recovery candidates map only covers `seal`, `entomb`, `status`, and `resume`. Other lifecycle commands that could fail with user-actionable errors (e.g., `build`, `continue`, `plan`) still use `outputError` and do not get recovery menus. The `genericFallback` function provides a safety net, but the `plan` command in particular can fail with "no colony initialized" or "state corruption" errors that would benefit from the same recovery UX.

**Fix:** Add recovery candidates for `plan` and `build` at minimum:
```go
"plan": {
    "no_colony": {
        {Label: "Initialize a colony", Command: "aether init \"goal\"", Rationale: "Plan requires an active colony"},
    },
    "state_corruption": {
        {Label: "Run diagnostics", Command: "aether patrol", Rationale: "Identify the state issue"},
    },
},
```

## Info

### IN-01: `classifyError` uses substring matching which may produce false positives

**File:** `cmd/recovery_engine.go:46-64`
**Issue:** The `classifyError` function matches `"json:"` as a substring to detect state corruption. Any error message that happens to contain the word "json:" (even in a non-corruption context) would be misclassified. This is a minor concern since the error messages are internally generated, but worth noting for maintenance.

**Fix:** No immediate action needed. Consider prefix matching or structured error types in a future iteration.

### IN-02: `extractCycle` fallback path is unreachable in correct usage

**File:** `pkg/colony/cycle.go:108-119`
**Issue:** The fallback at line 118 (`return append(path, target)`) is documented as "should not happen if called correctly" but has no test coverage for the fallback path. If a bug in the DFS logic caused the target to not appear in the path, this would silently return an incorrect cycle representation.

**Fix:** Consider returning an error from `extractCycle` in the fallback case, or add a comment/assertion that makes the invariant explicit:
```go
// extractCycle extracts the cycle from the DFS path when a back-edge to
// target is found. It returns the cycle path including target at both ends.
func extractCycle(path []string, target string) []string {
    for i, node := range path {
        if node == target {
            cycle := make([]string, len(path)-i+1)
            copy(cycle, path[i:])
            cycle[len(cycle)-1] = target
            return cycle
        }
    }
    // This should never happen: target must be in the DFS path when
    // a gray-to-gray edge is detected. Return a degraded result.
    return append([]string{target}, path...)
}
```

---

_Reviewed: 2026-04-30_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
