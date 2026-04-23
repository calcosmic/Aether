---
phase: 45-e2e-regression-coverage
reviewed: 2026-04-24T12:00:00Z
depth: standard
files_reviewed: 1
files_reviewed_list:
  - cmd/e2e_regression_test.go
findings:
  critical: 1
  warning: 2
  info: 2
  total: 5
status: issues_found
---

# Phase 45: Code Review Report

**Reviewed:** 2026-04-24T12:00:00Z
**Depth:** standard
**Files Reviewed:** 1
**Status:** issues_found

## Summary

Reviewed `cmd/e2e_regression_test.go` -- a new E2E regression test file covering four scenarios: stable publish-update, dev publish-update, stale publish detection, and channel isolation. The tests are structurally sound and follow existing project patterns. However, one critical nil-panic risk exists in the JSON assertion logic, and a couple of quality issues were identified.

## Critical Issues

### CR-01: Nil map index panic on missing JSON keys in stale publish assertion

**File:** `cmd/e2e_regression_test.go:192-195`
**Issue:** Lines 192-193 perform type assertions with comma-ok but discard the `ok` boolean:

```go
inner, _ := result["result"].(map[string]interface{})
stale, _ := inner["stale_publish"].(map[string]interface{})
```

If the JSON output is valid but does not contain a `"result"` key (e.g., the command output structure changes), `inner` is `nil`. The subsequent `inner["stale_publish"]` panics with "index into nil map". Similarly, if `"result"` exists but lacks `"stale_publish"`, then `stale` is `nil` and `stale["classification"]` on line 195 panics.

While this is test code and a panic would fail the test, the error message would be an unhelpful nil-dereference stack trace rather than a clear assertion message indicating which key was missing. More importantly, if the update command changes its output format in a way that removes the `result` wrapper but still returns an error, the test panics instead of reporting the structural mismatch -- masking the actual regression.

This same pattern exists in `e2e_stale_publish_test.go` and `update_cmd_test.go`, but this review is scoped to the new file.

**Fix:**
```go
inner, ok := result["result"].(map[string]interface{})
if !ok {
    t.Fatalf("expected result to be a map, got: %v", result["result"])
}
stale, ok := inner["stale_publish"].(map[string]interface{})
if !ok {
    t.Fatalf("expected stale_publish to be a map, got: %v", inner["stale_publish"])
}
```

## Warnings

### WR-01: Version global not saved/restored by saveGlobals, managed manually

**File:** `cmd/e2e_regression_test.go:157-159`
**Issue:** `TestE2ERegressionStalePublishDetection` mutates the package-level `Version` variable directly and restores it with a manual `defer`. The `saveGlobals` helper does not cover `Version`. While the cleanup ordering is correct (defer runs before t.Cleanup), this creates a fragile pattern: if a future developer adds another test to this file that also mutates `Version` or moves the defer, the global could leak between tests. Every other global mutation in the codebase goes through `saveGlobals`.

**Fix:** Add `Version` to the `saveGlobals` function in `testing_main_test.go` so all mutable globals are managed uniformly. Alternatively, add a comment in this test explaining why `Version` is handled separately.

### WR-02: Missing t.Setenv("HOME", ...) in TestE2ERegressionChannelIsolation

**File:** `cmd/e2e_regression_test.go:213-218`
**Issue:** `TestE2ERegressionChannelIsolation` creates a `homeDir` via `t.TempDir()` but never sets `HOME` to it, unlike the other three tests in this file (lines 19, 81, 145 all call `t.Setenv("HOME", homeDir)`). The test passes `--home-dir` explicitly to both publish calls, so the missing env var does not cause a bug today. However, if someone adds an `update` call or any other subcommand to this test without realizing HOME is unset, it will write to the real user's home directory -- a side-effect risk in test code.

**Fix:**
```go
func TestE2ERegressionChannelIsolation(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)

    homeDir := t.TempDir()
    t.Setenv("HOME", homeDir) // add this line
    // ...
}
```

## Info

### IN-01: Redundant defer rootCmd.SetArgs after resetRootCmd

**File:** `cmd/e2e_regression_test.go:29`
**Issue:** Line 29 has `defer rootCmd.SetArgs([]string{})` but `resetRootCmd(t)` on line 16 already registers a `t.Cleanup` that calls `rootCmd.SetArgs([]string{})`. The explicit defer is redundant. The same redundancy exists on lines 93 and 227. Line 176 is the only instance that is not redundant (because `resetRootCmd` runs `SetArgs` in cleanup, not defer, and the ordering is fine either way).

**Fix:** Remove the redundant `defer rootCmd.SetArgs([]string{})` lines at lines 29, 93, and 227, since `resetRootCmd` handles this.

### IN-02: os.IsNotExist check misses other error types in file existence assertions

**File:** `cmd/e2e_regression_test.go:64`
**Issue:** The pattern `if _, err := os.Stat(path); os.IsNotExist(err) { t.Fatal(...) }` only fails the test if the file does not exist. If `os.Stat` returns a different error (e.g., permission denied on the temp dir), the test silently passes despite the file being inaccessible. This is a common Go pattern and unlikely to cause issues with `t.TempDir()`, but `if err != nil { t.Fatal(...) }` is more robust. The same pattern appears at lines 64 and 127.

**Fix:** Use `if err != nil { t.Fatalf("downstream workers.md not accessible: %v", err) }` instead of checking only `os.IsNotExist`.

---

_Reviewed: 2026-04-24T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
