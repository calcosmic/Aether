---
phase: 90-learning-foundation
reviewed: 2026-05-01T12:00:00Z
depth: standard
files_reviewed: 19
files_reviewed_list:
  - cmd/codex_continue_finalize.go
  - cmd/codex_workflow_cmds.go
  - cmd/colony_prime_context.go
  - cmd/graph_consolidation_cmds.go
  - cmd/learn_export.go
  - cmd/learning.go
  - cmd/learning_cmds.go
  - pkg/learn/classify.go
  - pkg/learn/classify_test.go
  - pkg/learn/colony_store.go
  - pkg/learn/colony_store_test.go
  - pkg/learn/evidence.go
  - pkg/learn/export.go
  - pkg/learn/export_test.go
  - pkg/learn/hermes.go
  - pkg/learn/hive_store.go
  - pkg/learn/trigger.go
  - pkg/learn/trigger_test.go
  - pkg/learn/wrappers.go
findings:
  critical: 2
  warning: 6
  info: 4
  total: 12
status: issues_found
---

# Phase 90: Code Review Report

**Reviewed:** 2026-05-01T12:00:00Z
**Depth:** standard
**Files Reviewed:** 19
**Status:** issues_found

## Summary

Reviewed the learning foundation package (`pkg/learn/`) and its CLI integration in `cmd/`. The implementation introduces a repo-isolated learning store, hive-level cross-colony sharing with privacy controls, export/import, classification, and a trust-scoring evidence pipeline. The architecture is sound overall but contains two data integrity bugs (ID collision, silent error discard), a regression risk in the undo-promotions command, interface contract violations, and several code quality issues that should be addressed before this ships.

## Critical Issues

### CR-01: `learning-inject` generates non-unique IDs causing silent data collisions

**File:** `cmd/learning_cmds.go:228`
**Issue:** The `learning-inject` command generates observation IDs using `fmt.Sprintf("obs_%d", time.Now().Unix())`. If two injections happen within the same Unix second, they produce identical `ContentHash` values. The observation is then appended to `learning-observations.json` with no dedup check. This means the second observation silently overwrites the first in any lookup-by-hash path, and the file accumulates duplicate-hash entries. The `ColonyStore` in `pkg/learn/` uses `lrn_YYYYMMDD_seq` IDs with an atomic counter -- the legacy `learning-inject` command should use the same scheme or at minimum a content hash.

**Fix:**
```go
// Replace line 228 with a content-based hash, consistent with ColonyStore:
import "crypto/sha256"

hash := sha256.Sum256([]byte(content))
id := fmt.Sprintf("obs_%x", hash[:16])
```

### CR-02: `learning-undo-promotions` silently discards save errors for reverted observations

**File:** `cmd/learning_cmds.go:423`
**Issue:** After reverting observation `SourceType` values from "promoted" back to "proposed", the save to `learning-observations.json` discards the error with `store.SaveJSON(...)`. If this write fails (disk full, permissions, lock contention), the instincts are already archived (line 396), but the corresponding observations are not reverted. This leaves the system in an inconsistent state: instincts are archived but observations still claim "promoted" status. Re-running the undo would find no matching instincts to revert since they are already archived.

**Fix:**
```go
if reverted > 0 {
    if err := store.SaveJSON("learning-observations.json", obsFile); err != nil {
        outputError(2, fmt.Sprintf("failed to save reverted observations: %v", err), nil)
        return nil
    }
}
```

## Warnings

### WR-01: `ColonyStore.Add` assigns ID to local copy, not the caller's entry

**File:** `pkg/learn/colony_store.go:79-92`
**Issue:** `Add` receives `entry` by value. Line 89 (`entry.ID = assignedID`) modifies only the local copy -- the caller's original entry is unchanged. The `assignedID` is correctly captured inside the closure, but the post-closure assignment on line 89 is dead code. Callers that need the assigned ID (e.g., for immediate reference) cannot get it from the `Entry` they passed in, because `Add` returns only `error`.

**Fix:** Either change the signature to return the assigned ID:
```go
func (c *ColonyStore) Add(entry Entry) (string, error) {
    // ... return assignedID, err
}
```
Or remove the dead assignment on line 89 and document that callers must `List` or `Get` to retrieve the assigned ID.

### WR-02: `HiveStore.List` silently ignores `Phase` and `Classification` filters

**File:** `pkg/learn/hive_store.go:175-196`
**Issue:** `HiveStore` implements the `LearnStore` interface, whose `EntryFilter` includes `Phase` and `Classification` fields. `HiveStore.List` only checks `MinConfidence` and `Limit`, silently ignoring `Phase` and `Classification`. Any caller using the interface with these filters will get incorrect (unfiltered) results. The `ColonyStore.List` correctly handles all filter fields.

**Fix:** Add filter checks for Phase and Classification, or document that HiveStore does not support these filters and return an error if they are set:
```go
if filter.Phase != 0 {
    return nil, fmt.Errorf("learn: HiveStore does not support phase filtering")
}
if filter.Classification != "" {
    return nil, fmt.Errorf("learn: HiveStore does not support classification filtering")
}
```

### WR-03: `IsGeneric` compiles regex on every call

**File:** `pkg/learn/classify.go:52`
**Issue:** `extPattern` is compiled via `regexp.MustCompile` inside the `IsGeneric` function body. While Go caches compiled regexps at the package level (the var is reused), the declaration is inside the function, meaning it recompiles on every call. It should be a package-level `var` like `secretPattern` on line 19.

**Fix:**
```go
// Move to package level, alongside secretPattern:
var extPattern = regexp.MustCompile(`\.\w{1,4}\b`)

func IsGeneric(content string) bool {
    if strings.Contains(content, "/") {
        return false
    }
    if extPattern.MatchString(content) {
        return false
    }
    return true
}
```

### WR-04: `HiveStore` has no file locking -- concurrent access causes data corruption

**File:** `pkg/learn/hive_store.go:52-77`
**Issue:** `HiveStore` reads and writes `~/.aether/hive/wisdom.json` using `os.ReadFile`/`os.WriteFile` with no file locking. If two Aether instances (or two commands within the same instance) access the hive simultaneously (e.g., parallel workers promoting to hive during seal), the classic TOCTOU race applies: both read the same state, both append their entry, and the second write clobbers the first's addition. The `ColonyStore` avoids this by using `store.UpdateFile` which has built-in locking. `HiveStore` bypasses this entirely.

**Fix:** Use file locking (e.g., `storage.Store` methods or a lock file) for hive wisdom reads/writes, consistent with how `ColonyStore` handles concurrent access.

### WR-05: `learning-extract-fallback` has dead code after early-return guard

**File:** `cmd/learning_cmds.go:175-184`
**Issue:** On line 175, `content` is fetched via `mustGetString(cmd, "content")`. If `content == ""`, the function returns nil on line 177. On line 182, the code checks `if extracted == "" && fallback != ""` -- but `extracted` was set to `content` on line 182, which can never be empty at that point (we already returned nil on line 177). The fallback branch is unreachable dead code.

**Fix:** The `content` and `fallback` parameters likely need to come from a different source (e.g., file or stdin), or the empty-content early return should be removed if empty content with a fallback is a valid use case.

### WR-06: `learning-approve-proposals` has `--deferred` and `--verbose` flags that are declared but never read

**File:** `cmd/learning_cmds.go:479-480`
**Issue:** The `--deferred` and `--verbose` flags are registered on `learningApproveProposalsCmd` but never read inside the `RunE` function. Users passing `--deferred` would expect deferred proposals to also be approved, but the flag has no effect. This is a silent contract violation.

**Fix:** Either implement the `--deferred` flag logic (check `obs.SourceType == "deferred"` in addition to "proposed") and remove `--verbose` if not needed, or remove both flag declarations.

## Info

### IN-01: `ColonyStore.Compact` with budget <= 0 removes all entries

**File:** `pkg/learn/colony_store.go:189`
**Issue:** If `Compact(0)` is called, `totalLen+len(e.Content) > 0` is always true for any non-empty entry, so all entries are removed. This is technically correct (zero budget means nothing fits) but could be surprising. Consider adding a guard.

### IN-02: `codex_continue_finalize.go` discards gate result write errors

**File:** `cmd/codex_continue_finalize.go:197`
**Issue:** `gateResultsWrite(gateResultEntries)` error is discarded with `_`. If this write fails, the global gate results history has a gap, but the per-phase write on line 215 may succeed, leading to inconsistent gate history. This is a minor observability gap.

### IN-03: Test helpers duplicate stdlib functions

**File:** `pkg/learn/export_test.go:495-515`
**Issue:** `unmarshalJSON`, `marshalJSONIndent`, and `containsString`/`searchString` are hand-rolled wrappers around `json.Unmarshal`, `json.MarshalIndent`, and `strings.Contains`. These provide no value over direct stdlib calls and add maintenance burden.

### IN-04: `learn.go` and `wrappers.go` are thin delegation wrappers with no tests

**File:** `pkg/learn/learn.go`, `pkg/learn/wrappers.go`
**Issue:** Both files contain only type aliases and thin delegation functions that call `pkg/memory` functions. While this is a valid architectural pattern (decoupling `cmd/` from `pkg/memory`), neither file has any tests. The delegation is trivial enough that bugs are unlikely, but a contract test verifying the wrappers forward correctly would catch regressions if the `pkg/memory` API changes.

---

_Reviewed: 2026-05-01T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
