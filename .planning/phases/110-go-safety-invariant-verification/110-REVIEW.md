---
phase: 110-go-safety-invariant-verification
reviewed: 2026-05-12T12:00:00Z
depth: standard
files_reviewed: 1
files_reviewed_list:
  - cmd/safety_invariant_test.go
findings:
  critical: 0
  warning: 3
  info: 1
  total: 4
status: issues_found
---

# Phase 110: Code Review Report

**Reviewed:** 2026-05-12T12:00:00Z
**Depth:** standard
**Files Reviewed:** 1
**Status:** issues_found

## Summary

Reviewed `cmd/safety_invariant_test.go`, which implements six safety invariant tests (SAFE-01 through SAFE-06) to verify that Go remains the sole authority for state mutation, finalizers reject corrupted manifests, atomic writes work correctly, install/update/publish have no TS host involvement, verification contracts pass, and plan-only modes produce unchanged JSON.

The test file is well-structured and covers important safety boundaries. Issues found center on test isolation gaps (missing `saveGlobals` in two tests), a weak snapshot comparison that could miss same-size content mutations, and a missing positive-path test case in the finalizer provenance checks.

## Warnings

### WR-01: Missing `saveGlobals` in TestStateMutationSoleAuthority leaks global `store`

**File:** `cmd/safety_invariant_test.go:29`
**Issue:** `TestStateMutationSoleAuthority` calls `setupBuildFlowTest(t)` which overwrites the global `store` variable with a temp-directory-backed store, but it does NOT call `saveGlobals(t)` first. `setupBuildFlowTest` only registers cleanup for `stdout`/`stderr` -- it does NOT restore the original `store`. After this test completes, the global `store` points to a cleaned-up temp directory, which could cause panics or data corruption in any subsequent test that uses `store` without re-initializing it.
**Fix:**
```go
func TestStateMutationSoleAuthority(t *testing.T) {
	saveGlobals(t)        // <-- add this line
	dataDir := setupBuildFlowTest(t)
	// ... rest of test
```

### WR-02: Missing `saveGlobals` in TestLockingUnchanged leaks global `store`

**File:** `cmd/safety_invariant_test.go:397`
**Issue:** Same as WR-01. `TestLockingUnchanged` calls `setupBuildFlowTest(t)` without first calling `saveGlobals(t)`. The global `store` is overwritten without saving the original value, so after this test finishes, `store` points to a cleaned-up temp directory.
**Fix:**
```go
func TestLockingUnchanged(t *testing.T) {
	saveGlobals(t)        // <-- add this line
	dataDir := setupBuildFlowTest(t)
	// ... rest of test
```

### WR-03: `assertDataDirUnchanged` does not compare modification times -- same-size content mutations pass silently

**File:** `cmd/safety_invariant_test.go:61` (calls `assertDataDirUnchanged` defined in `cmd/boundary_contract_test.go:40-57`)
**Issue:** The `fileSnapshot` struct captures both `size` and `modTime`, but `assertDataDirUnchanged` only compares `size`. A file rewritten with different content of identical length (or content that happens to produce the same file size) would pass the assertion undetected. This weakens the SAFE-01, SAFE-03, and SAFE-06 tests that rely on this function to prove no state mutation occurred. The `modTime` field is captured but never used, which is misleading.
**Fix:** In `boundary_contract_test.go`, update `assertDataDirUnchanged` to also compare `modTime`, or better yet, hash file contents:
```go
if beforeSnap.size != afterSnap.size || beforeSnap.modTime != afterSnap.modTime {
    t.Errorf("file changed during orchestration: %s", name)
}
```
Or use SHA-256 content hashing in `fileSnapshot` for stronger guarantees.

## Info

### IN-01: TestFinalizerProvenance has no positive-path test cases

**File:** `cmd/safety_invariant_test.go:91-388`
**Issue:** All test cases in `TestFinalizerProvenance` have `wantReject: true`. There are no valid manifests that should be accepted. Without a positive-path case (a manifest that passes all validation), the test only proves rejections work. If a future refactor changes the finalizer to reject ALL manifests (including valid ones), this test would still pass because it only checks rejection scenarios.
**Fix:** Add at least one `wantReject: false` case per finalizer type with a fully valid manifest to verify the acceptance path still works.

---

_Reviewed: 2026-05-12T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
