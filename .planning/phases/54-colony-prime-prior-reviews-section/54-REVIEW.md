---
phase: 54-colony-prime-prior-reviews-section
reviewed: 2026-04-26T12:00:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - cmd/colony_prime_prior_reviews_test.go
  - cmd/colony_prime_context.go
  - cmd/context_weighting.go
  - cmd/review_ledger.go
  - pkg/colony/review_ledger.go
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 54: Code Review Report

**Reviewed:** 2026-04-26T12:00:00Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

Reviewed 5 files implementing the prior-reviews section for colony-prime context assembly and the review ledger CLI commands. The implementation is well-structured with good test coverage (12 test cases for prior reviews, 15 for review ledger commands). Tests pass cleanly and `go vet` reports no issues.

Found one data-contract bug where `review-ledger-read` returns a stale summary when entries are filtered, plus three warnings around code duplication, a deprecated API, and a silent cache-write failure. Two informational items cover the duplication pattern and test coverage gap.

## Critical Issues

### CR-01: review-ledger-read returns stale summary when filters are active

**File:** `cmd/review_ledger.go:223-226`
**Issue:** When `review-ledger-read` is called with `--phase` or `--status` filters, the `entries` array is filtered correctly but the response still includes `lf.Summary` -- the summary computed over ALL entries (not the filtered subset). A caller filtering `--status open` would receive a summary claiming there are resolved entries, which contradicts the returned entries list. This is a data-contract mismatch that silently misleads consumers.

**Fix:**
```go
// After filtering, recompute summary for the returned entries
filteredSummary := colony.ComputeSummary(entries)

outputOK(map[string]interface{}{
    "entries": entries,
    "summary": filteredSummary,
})
```

Alternatively, if the full summary is intentionally preserved for context, document it explicitly in the response (e.g., add `"filtered": true` and `"unfiltered_summary"` fields) so consumers know the summary does not match the filtered entries.

## Warnings

### WR-01: Duplicated domain maps between cmd and pkg/colony risk silent divergence

**File:** `cmd/review_ledger.go:15-23`
**Issue:** `validDomains` (`map[string]bool`) and `domainPrefixes` (`map[string]string`) duplicate domain data that already exists as `colony.ValidReviewDomains` in `pkg/colony/review_ledger.go`. If a new domain is added to the package-level map but not to the cmd-level map (or vice versa), the write command would reject valid domains or accept invalid ones. Currently the values are aligned, but there is no synchronization mechanism.

**Fix:** Replace the local maps with references to the package-level data:
```go
// Validation:
if _, ok := colony.ValidReviewDomains[domain]; !ok {
    // ...
}

// Prefix lookup:
prefix := colony.ValidReviewDomains[domain]
```

### WR-02: Deprecated strings.Title usage

**File:** `cmd/colony_prime_context.go:282`
**Issue:** `strings.Title` has been deprecated since Go 1.18 (current project uses Go 1.26). The Go docs recommend `cases.Title` from `golang.org/x/text/cases` for proper Unicode-aware title casing. While `strings.Title` still works, it does not handle Unicode properly and may be removed in future Go versions. For the domain names in this codebase (all ASCII), it functions correctly today.

**Fix:**
```go
import (
    "golang.org/x/text/cases"
    "golang.org/x/text/language"
)

// In buildPriorReviewsSection:
domainLabel := fmt.Sprintf("- %s (%d open)", cases.Title(language.English).String(dd.domain), len(dd.open))
```

Or simpler: since domain names are fixed and known, use a title-case lookup map instead.

### WR-03: Cache write failure is silently ignored

**File:** `cmd/colony_prime_context.go:312`
**Issue:** `_ = s.SaveJSON(cachePath, cache)` discards the error from writing the summary cache. If the write fails (disk full, permissions), the cache will not be updated, and subsequent calls will either use a stale cache or fall through to a full rebuild. The fallback to full rebuild is safe for correctness, but the silent discard means the failure is invisible for debugging.

**Fix:**
```go
if err := s.SaveJSON(cachePath, cache); err != nil {
    // Log the failure for debugging; the section is still returned correctly
    // since cache is an optimization, not a correctness requirement.
}
```
At minimum, log the error to stderr or a debug channel so it is diagnosable.

## Info

### IN-01: agentAllowedDomains map is only defined locally in cmd/review_ledger.go

**File:** `cmd/review_ledger.go:25-33`
**Issue:** The `agentAllowedDomains` map defining which agents can write to which domains is a local variable in the cmd package with no package-level equivalent. If other packages or commands need to validate agent-domain mappings, they would need to duplicate this data. This is not a bug but a maintainability note.

**Fix:** Consider promoting this to `pkg/colony/review_ledger.go` alongside `ValidReviewDomains` if other consumers need it.

### IN-02: No test verifying summary consistency with filtered read results

**File:** `cmd/review_ledger_test.go:313-350`
**Issue:** `TestReviewLedgerRead_FilterByStatus` verifies that the correct number of filtered entries is returned but does not check whether `summary.open` matches the filtered entry count. Adding an assertion would have caught CR-01.

**Fix:**
```go
// In TestReviewLedgerRead_FilterByStatus, after verifying entry count:
summary := result["summary"].(map[string]interface{})
if summary["open"] != float64(2) {
    t.Errorf("summary.open = %v, want 2 (matching filtered entries)", summary["open"])
}
```

---

_Reviewed: 2026-04-26T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
