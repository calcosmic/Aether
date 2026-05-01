---
phase: 66-continue-flow-fix
reviewed: 2026-04-28T12:00:00Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - cmd/codex_continue.go
  - cmd/codex_continue_test.go
  - cmd/gate_test.go
  - cmd/session_flow_cmds_test.go
findings:
  critical: 1
  warning: 4
  info: 3
  total: 8
status: issues_found
---

# Phase 66: Code Review Report

**Reviewed:** 2026-04-28T12:00:00Z
**Depth:** standard
**Files Reviewed:** 4
**Status:** issues_found

## Summary

Reviewed the continue flow fix implementation across four source files: `codex_continue.go` (2650 lines of core continue logic), `codex_continue_test.go` (extensive integration tests), `gate_test.go` (gate check unit and CLI tests), and `session_flow_cmds_test.go` (session lifecycle and pheromone detection tests).

The codebase is generally well-tested with strong integration coverage. However, one logic bug in the watcher dispatch filtering will cause it to match incorrect dispatches, and there are several quality issues including an unused function parameter, a potential index panic, and test hygiene concerns.

## Critical Issues

### CR-01: `evaluateContinueWatcherVerification` uses AND instead of OR for watcher dispatch matching

**File:** `cmd/codex_continue.go:1272`
**Issue:** The condition `stage != "verification" && caste != "watcher"` uses `&&` (logical AND) instead of `||` (logical OR). This means the `continue` statement only fires when BOTH conditions are true -- i.e., when the dispatch is neither a verification stage NOR a watcher caste. Consequently, a dispatch that is a watcher but NOT in the verification stage (or vice versa) will be incorrectly matched and returned as the build watcher verification.

For example, a dispatch with `stage="wave"` and `caste="watcher"` should match (it is a watcher), but the condition evaluates to `("wave" != "verification") && ("watcher" != "watcher")` = `true && false` = `false`, so the dispatch IS NOT skipped and IS returned as the build watcher -- which is correct in this specific case. However, a dispatch with `stage="verification"` and `caste="builder"` should NOT match (it is a builder in the verification stage, not a watcher). The condition evaluates to `("verification" != "verification") && ("builder" != "watcher")` = `false && true` = `false`, so it IS NOT skipped and IS returned as the build watcher -- which is incorrect. The intent was to skip dispatches that are NEITHER verification stage NOR watcher caste, but the AND logic causes it to also return verification-stage non-watchers.

**Fix:**
```go
// Line 1272: change && to ||
if stage != "verification" && caste != "watcher" {
    continue
}
// Should be:
if stage != "verification" || caste != "watcher" {
    continue
}
// Or more explicitly, match only dispatches that are BOTH verification stage AND watcher:
if stage == "verification" && caste == "watcher" {
    // ... process this dispatch
}
```

## Warnings

### WR-01: Unused `claimsSatisfied` parameter in `continueTasksSupportAdvancement`

**File:** `cmd/codex_continue.go:1432`
**Issue:** The function `continueTasksSupportAdvancement(tasks []codexContinueTaskAssessment, claimsSatisfied bool)` accepts a `claimsSatisfied` parameter that is never referenced in the function body. The caller passes `claimsSatisfied` at line 1362, suggesting this was intended to gate advancement on claim verification but the logic was either removed or never implemented. This dead parameter makes the code misleading -- callers may assume claims are being checked when they are not.
**Fix:** Either implement claims-aware logic in the function body, or remove the parameter and update all callers.

### WR-02: Potential index-out-of-range panic in `continueReviewFlowSummary`

**File:** `cmd/codex_continue.go:2490`
**Issue:** The expression `role[:1]` will panic if `role` is a single non-empty character that becomes empty after `strings.TrimSpace`. While `role` is checked to be non-empty before the slice, the code path at line 2489 checks `role != ""` and then accesses `role[:1]`. However, the risk is minimal because `strings.TrimSpace` already stripped the string. A more defensive approach would use a safer accessor.
**Fix:**
```go
if role != "" {
    // Use rune-aware title case to avoid index panic with multi-byte characters
    role = strings.ToUpper(string([]rune(role)[0])) + role[1:]
}
```

### WR-03: `continueTasksSupportAdvancement` blocks advancement on `manually_reconciled` tasks even when all tasks are reconciled

**File:** `cmd/codex_continue.go:1438`
**Issue:** The outcome `"manually_reconciled"` is in the blocklist for `continueTasksSupportAdvancement`, meaning a phase where ALL tasks are manually reconciled will always return `false` for `positiveEvidence`, preventing advancement. However, the `assessCodexContinue` function at line 1403 computes `passed = verification.ChecksPassed && positiveEvidence`, so even when all tasks are reconciled and verification passes, the assessment will fail. This makes the `--reconcile-task` feature unable to fully advance a phase when all tasks need reconciliation. The test `TestContinueBlocksWhenReconciledTaskLacksClaimEvidence` confirms this behavior but only tests the partial case -- when ALL tasks are reconciled, the user gets stuck with no recovery path beyond re-dispatching.
**Fix:** When all tasks in the phase are `"manually_reconciled"` and verification passes, allow advancement by adding a check for the "all reconciled" state.

### WR-04: `runShellCommand` passes user-controlled verification commands through shell execution without sanitization

**File:** `cmd/codex_continue.go:2526-2538`
**Issue:** The `runShellCommand` function executes commands via `sh -c <command>` (or `cmd /C` on Windows), where the command string is derived from markdown documentation files like `CLAUDE.md`, `CODEX.md`, etc. These commands are parsed from arbitrary markdown content using regex-like extraction. A malicious or malformed markdown file could inject shell commands. While these files are typically under version control, the attack surface exists when processing untrusted project documentation.
**Fix:** Consider adding a command allowlist or sanitization step for verification commands, or at minimum validate that extracted commands match known-safe patterns (which `looksLikeVerificationCommand` partially does but allows `sh`, `bash`, and `make` which are arbitrary execution vectors).

## Info

### IN-01: Test helper `seedContinueBuildPacket` silently normalizes "spawned" to "completed"

**File:** `cmd/codex_continue_test.go:2490-2493`
**Issue:** The test helper silently changes dispatch statuses from `"spawned"` to `"completed"`, which can mask test bugs. Tests that intend to verify "spawned" dispatch behavior must manually work around this by writing manifests directly (as seen in `TestContinueDetectsAbandonedBuild`). This is a test maintainability concern rather than a production bug.
**Fix:** Remove the silent normalization or make it opt-in via a parameter.

### IN-02: `gate_test.go` uses raw `map[string]interface{}` instead of typed `colony.ColonyState`

**File:** `cmd/gate_test.go:26-36`
**Issue:** Several tests in `gate_test.go` (e.g., `TestGateCheck_TaskComplete_AllPass`, `TestGateCheck_PhaseAdvance_PendingTasks`) construct colony state using raw maps and `json.Marshal`/`os.WriteFile` instead of using typed structs like `colony.ColonyState`. This is fragile -- if struct field names change, these tests will silently test the wrong state shape.
**Fix:** Use the typed `colony.ColonyState` struct and `store.SaveJSON` consistently, as other tests do (e.g., `TestPreBuildGates`).

### IN-03: `gate_test.go` does not restore original `store` global on cleanup

**File:** `cmd/gate_test.go:23-24` and similar patterns
**Issue:** Tests like `TestGateCheck_TaskComplete_AllPass` set the global `store` variable but do not register a `t.Cleanup` to restore the original value. If these tests run in parallel (even accidentally), they can corrupt each other's state. Other test files in this codebase correctly use `saveGlobals(t)` / `resetRootCmd(t)` patterns.
**Fix:** Use `saveGlobals(t)` and `resetRootCmd(t)` in all gate tests, or at minimum add `t.Cleanup(func() { store = origStore })`.

---

_Reviewed: 2026-04-28T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
