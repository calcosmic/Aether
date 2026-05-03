---
phase: 94-recovery-data-model
reviewed: 2026-05-03T12:00:00Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - cmd/recovery_classify.go
  - cmd/recovery_classify_test.go
findings:
  critical: 0
  warning: 3
  info: 3
  total: 6
status: issues_found
---

# Phase 94: Code Review Report

**Reviewed:** 2026-05-03T12:00:00Z
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found

## Summary

Reviewed `cmd/recovery_classify.go` and `cmd/recovery_classify_test.go` at standard depth. The implementation follows the established patterns from `cmd/gate.go` closely -- same type structure for classification entries, same table rendering pattern, same CLI flag registration in `init()`. All 29 tests pass.

The code is generally well-structured. The classification logic is clean and deterministic as required by the design decisions. Three warnings were found: a duplicate error-message output pattern in the write command, an inconsistency in how `shouldSkipGate` type signatures are used across the codebase (pre-existing, but this file adds related code that should be aware of it), and a missing test for the `recovery-log-write` CLI command. Three info items cover an empty test body, the `unparseable_output` classification being both Systemic and RequiresAttempt (which may be a design tension), and a minor timestamp derivation concern.

## Warnings

### WR-01: Duplicate error output on required flags in recoveryLogWriteCmd

**File:** `cmd/recovery_classify.go:230-250`
**Issue:** The `recoveryLogWriteCmd` calls `mustGetString()` which already outputs an error and returns `""` when the flag is missing or empty (see `helpers.go:66-77`). The command then checks if the result is `""` and calls `outputErrorMessage()` again. This means a missing `--worker` flag produces two error messages: `flag --worker is required` from `mustGetString` and `--worker is required` from the explicit check. The `gateResultsWriteCmd` in `gate.go:751-753` avoids this by using `mustGetString` alone and returning nil on empty (no second check). The explicit `outputErrorMessage` calls also use a different format (no `flag --` prefix), creating inconsistent error output across commands.
**Fix:** Remove the redundant `outputErrorMessage` + `return nil` blocks, matching the pattern in `gate.go`. Use `mustGetString` alone and return nil when it returns empty:
```go
worker := mustGetString(cmd, "worker")
if worker == "" {
    return nil
}
status := mustGetString(cmd, "status")
if status == "" {
    return nil
}
```

### WR-02: Missing test for recovery-log-write CLI command

**File:** `cmd/recovery_classify_test.go`
**Issue:** There are tests for `failure-classify` CLI (both JSON and table output) and `recovery-log-read` CLI, but no test for the `recovery-log-write` CLI command. This is the most complex command in the file -- it reads existing entries, appends a new one with classification logic, and writes back. This write path is untested at the CLI integration level. A bug in the read-then-append-write pipeline (e.g., data loss on concurrent writes, malformed JSON from partial writes) would not be caught by tests.
**Fix:** Add a test `TestRecoveryLogWriteCmd` that exercises the full CLI path: set up a test store, call `recovery-log-write` with required flags, verify the entry is persisted and readable via `recovery-log-read`. Also test missing required flags to verify the validation path.

### WR-03: unparseable_output classified as both Systemic and RequiresAttempt -- design tension

**File:** `cmd/recovery_classify.go:54`
**Issue:** `unparseable_output` is classified as `Systemic` (fundamental problem) but `RequiresAttempt` (try once). The comment on line 54 says "may indicate deeper issue" which aligns with Systemic, but a truly Systemic issue is unlikely to resolve on a single retry. By contrast, `partial_completion` is `Transient` + `RequiresAttempt`, which makes logical sense (the transient condition may resolve). If the output is garbled because of a systemic parser issue, retrying will produce the same garbled output. If it is garbled because of a transient API glitch, it should be `Transient`. This classification may lead to wasted retry attempts on systemic issues in Phase 96 when the recovery dispatcher consumes these records.
**Fix:** Consider reclassifying to either `Transient` + `RequiresAttempt` (if the garbling is assumed to be environmental) or `Systemic` + `Blocking` (if the garbling indicates a fundamental problem). The current split classification sends a contradictory signal to the recovery dispatcher.

## Info

### IN-01: TestFailureClassifications_NoCrossDomainImports is an empty test

**File:** `cmd/recovery_classify_test.go:185-189`
**Issue:** This test has an empty body with a comment saying "This test exists as documentation" and enforcement is via CI grep. While the intent is reasonable, an empty test always passes and provides no runtime protection. If the CI grep is removed or skipped, the domain boundary has no enforcement at all.
**Fix:** Consider adding a minimal runtime check, e.g., reading the file content and asserting it does not contain `GateClassificationTier`. Even a simple `go:embed` or build-tag approach would provide better protection than an empty test.

### IN-02: RecoveryLogEntry ID uses UnixNano without collision guard

**File:** `cmd/recovery_classify.go:256`
**Issue:** `fmt.Sprintf("rl_%d", time.Now().UnixNano())` generates IDs based on nanosecond timestamp. In practice this is unlikely to collide, but if two writes happen in the same nanosecond (e.g., batch processing or test scenarios), duplicate IDs would occur. The gate.go file does not have this pattern for IDs, so there is no established precedent either way.
**Fix:** This is low risk for production use. If ID uniqueness becomes important in Phase 96, consider adding a counter or random suffix.

### IN-03: Two separate time.Now() calls in recoveryLogWriteCmd create timestamp skew

**File:** `cmd/recovery_classify.go:256,264,269`
**Issue:** Three separate `time.Now().UTC()` calls are made during entry construction: one for the ID (line 256), one for `Failure.Timestamp` (line 264), and one for `Timestamp` (line 269). These could differ by microseconds. While not a correctness bug, it means the ID timestamp and the recorded timestamp may not match exactly.
**Fix:** Capture `now := time.Now().UTC()` once at the top of the entry construction block and reuse it for all three timestamp fields.

---

_Reviewed: 2026-05-03T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
