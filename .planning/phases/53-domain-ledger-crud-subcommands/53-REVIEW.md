---
phase: 53-domain-ledger-crud-subcommands
reviewed: 2026-04-26T12:00:00Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - cmd/review_ledger.go
  - cmd/review_ledger_test.go
  - pkg/colony/review_ledger.go
  - pkg/colony/review_ledger_test.go
findings:
  critical: 0
  warning: 4
  info: 3
  total: 7
status: issues_found
---

# Phase 53: Code Review Report

**Reviewed:** 2026-04-26T12:00:00Z
**Depth:** standard
**Files Reviewed:** 4
**Status:** issues_found

## Summary

Reviewed the review ledger CRUD subcommands (`cmd/review_ledger.go`, `pkg/colony/review_ledger.go`) and their tests. The implementation is structurally sound -- it follows existing codebase patterns for cobra commands, envelope output, and store-based persistence. However, there are several issues ranging from a misleading summary returned during filtered reads, missing input validation for severity and status values, a dead exported variable, and insufficient test coverage for edge cases.

No critical security issues were found: the `--domain` flag is validated against a whitelist map before use in path construction, so path traversal is not possible. The storage layer provides file locking for concurrent access.

## Warnings

### WR-01: `review-ledger-read` returns unfiltered summary alongside filtered entries

**File:** `cmd/review_ledger.go:226-229`
**Issue:** When the `--phase` or `--status` filter is active, the command returns the full ledger summary (which covers all entries) alongside the filtered subset of entries. A caller filtering for `--status open --phase 2` receives a summary whose `total`/`open`/`resolved` counts reflect the entire ledger, not just the matching entries. This is a data consistency bug -- consumers will see `total: 50` with only 3 entries in the response.

**Fix:** Recompute the summary from the filtered entries before returning:
```go
outputOK(map[string]interface{}{
    "entries": entries,
    "summary": colony.ComputeSummary(entries),
})
```

### WR-02: No validation of `--severity` values in `review-ledger-write`

**File:** `cmd/review_ledger.go:139-140`
**Issue:** The severity value from `--findings` JSON is accepted without validation. Any string is cast to `ReviewSeverity` and stored. This means entries like `{"severity": "OMEGA", ...}` are silently persisted. The `ComputeSummary` function's switch statement will then ignore these entries in the severity breakdown, causing the `by_severity` counts to not sum to `total`. Downstream consumers (markdown rendering, dashboard) may break on unexpected severity values.

**Fix:** Add severity validation after parsing findings:
```go
validSeverities := map[string]bool{
    "HIGH": true, "MEDIUM": true, "LOW": true, "INFO": true,
}
for _, f := range findings {
    if !validSeverities[strings.ToUpper(f.Severity)] {
        outputError(1, fmt.Sprintf("invalid severity %q: must be HIGH, MEDIUM, LOW, or INFO", f.Severity), nil)
        return nil
    }
}
```

### WR-03: No validation of `--status` filter value in `review-ledger-read`

**File:** `cmd/review_ledger.go:215-223`
**Issue:** The `--status` flag accepts any arbitrary string and uses it for exact matching. If a caller passes `--status "pending"` (not a valid status), the filter silently returns zero entries with no error or warning. This is confusing UX -- the command succeeds with `ok: true` and an empty entries array, making it indistinguishable from a domain that genuinely has no entries.

**Fix:** Validate the status value when provided:
```go
status := mustGetStringCompatOptional(cmd, "status")
if status != "" {
    switch status {
    case "open", "resolved":
        // valid
    default:
        outputError(1, fmt.Sprintf("invalid --status %q: must be open or resolved", status), nil)
        return nil
    }
    // ... filter logic
}
```

### WR-04: `review-ledger-write` accepts empty findings array without error

**File:** `cmd/review_ledger.go:109-112`
**Issue:** There is a maximum check (`len(findings) > maxFindingsPerWrite`) but no minimum check. Passing `--findings "[]"` is valid JSON, passes all validation, and results in a successful write that loads the existing ledger, appends zero entries, recomputes the (unchanged) summary, and saves it back. This is a wasted I/O operation and creates a misleading `"written": true` response when nothing was actually written.

**Fix:** Add a minimum check:
```go
if len(findings) == 0 {
    outputError(1, "--findings must contain at least one finding", nil)
    return nil
}
```

## Info

### IN-01: Dead exported variable `ValidReviewDomains` in `pkg/colony/review_ledger.go`

**File:** `pkg/colony/review_ledger.go:20`
**Issue:** `ValidReviewDomains` is exported but never referenced outside the file. The `cmd/` layer maintains its own separate `validDomains` map (line 15) and `domainPrefixes` map (line 20). This creates a duplication risk -- if a new domain is added to one map but not the other, behavior becomes inconsistent. Either the `cmd/` layer should use `ValidReviewDomains` from `pkg/colony`, or the `pkg/colony` version should be unexported.

**Fix:** Either remove `ValidReviewDomains` from `pkg/colony/review_ledger.go` (since the cmd layer owns domain validation) or refactor the cmd layer to use it. Given the cmd layer needs the separate `agentAllowedDomains` mapping anyway, removing the dead export is the simpler path.

### IN-02: ID format breaks at 1000+ entries per domain-phase combination

**File:** `pkg/colony/review_ledger.go:98-99`
**Issue:** `FormatEntryID` uses `%03d` for zero-padding, producing IDs like `sec-2-001` through `sec-2-999`. At index 1000, it produces `sec-2-1000` (4 digits), which breaks the visual format but does not cause a functional bug since `NextEntryIndex` uses `strconv.Atoi` on the suffix, not fixed-width parsing. The IDs remain unique and deterministic. This is cosmetic only given the 50-per-write limit and typical phase sizes, but worth noting for documentation.

**Fix:** No code change required. Consider documenting the 999-per-phase ceiling as a known limit, or widening to `%04d` if there is any realistic scenario of exceeding 999 entries per domain-phase pair.

### IN-03: Missing test coverage for several edge cases

**File:** `cmd/review_ledger_test.go`
**Issue:** The tests cover the happy paths and basic error cases but miss several important scenarios:

- **Severity validation**: No test passes an invalid severity like `"CRITICAL"` or `"OMEGA"` to verify behavior
- **Empty findings array**: No test for `--findings "[]"`
- **Non-array findings JSON**: No test for `--findings "{}"` (valid JSON but not an array -- accepted by `json.Unmarshal` into a slice as empty)
- **ID overflow at 1000**: No test verifying ID format beyond 999 entries
- **Resolve on nonexistent ledger**: No test for `review-ledger-resolve` against a domain with no ledger file (the "ledger not found" error path at line 303-305)
- **Read with invalid status**: No test for `--status "invalid"` on read

**Fix:** Add test cases for each of these scenarios to prevent regressions and document expected behavior.

---

_Reviewed: 2026-04-26T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
