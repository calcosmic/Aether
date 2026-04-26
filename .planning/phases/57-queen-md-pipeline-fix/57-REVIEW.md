---
phase: 57-queen-md-pipeline-fix
reviewed: 2026-04-26T23:30:00Z
depth: standard
files_reviewed: 12
files_reviewed_list:
  - cmd/colony_prime_context.go
  - cmd/colony_prime_queen_test.go
  - cmd/context_queen_test.go
  - cmd/context_weighting.go
  - cmd/context.go
  - cmd/queen_dedup_test.go
  - cmd/queen_global_test.go
  - cmd/queen_hygiene_test.go
  - cmd/queen_seed_test.go
  - cmd/queen.go
  - .claude/commands/ant/seal.md
  - .opencode/commands/ant/seal.md
findings:
  critical: 0
  warning: 3
  info: 3
  total: 6
status: issues_found
---

# Phase 57: Code Review Report

**Reviewed:** 2026-04-26T23:30:00Z
**Depth:** standard
**Files Reviewed:** 12
**Status:** issues_found

## Summary

This phase adds three features to the QUEEN.md pipeline: (1) inclusion of global hub QUEEN.md wisdom in colony-prime output, (2) deduplication of wisdom entries across local and global QUEEN.md writes, and (3) filtering of already-present entries in queen-seed-from-hive. The implementation is well-tested with five new test files and comprehensive coverage of both happy paths and edge cases.

The most significant finding is a dishonest output flag: `queen-promote-instinct` reports `hub_written: true` unconditionally, even when the hub write was skipped (hub store nil) or silently failed. Two additional warnings cover a dedup normalization that is overly aggressive (stripping meaningful parenthetical content from wisdom entries) and a deprecated standard library function.

No security vulnerabilities or data loss risks were identified.

## Warnings

### WR-01: hub_written always reports true regardless of actual write outcome

**File:** `cmd/queen.go:302-306`
**Issue:** The `queen-promote-instinct` command reports `"hub_written": true` in its output unconditionally. The hub write block (lines 285-293) has three failure modes that all result in `hub_written: true` being reported: (a) `hubStore()` returns nil, so the entire block is skipped; (b) `loadQueenText` fails and the block is silently skipped via the `err == nil` check; (c) `writeQueenText` fails and is only logged, not propagated. Any caller relying on `hub_written` to determine whether the hub QUEEN.md was actually updated will get a false positive. The test `TestQueenPromoteInstinctSucceedsWithoutHub` at `cmd/queen_global_test.go:67-101` confirms this -- it runs without a hub directory and the command succeeds, but the output still reports `hub_written: true` despite no hub write occurring. The `result["promoted"]` assertion passes but no assertion checks `hub_written`, so the lie goes undetected.

**Fix:**
```go
// Track actual hub write result
hubWritten := false
if hs := hubStore(); hs != nil {
    if hubText, _, err := loadQueenText(hs); err == nil {
        hubText = appendEntryToQueenSection(hubText, "Wisdom", entry)
        if err := writeQueenText(hs, hubText); err != nil {
            log.Printf("queen-promote-instinct: failed to write hub QUEEN.md: %v", err)
        } else {
            hubWritten = true
        }
    }
}

// ... later in outputOK:
outputOK(map[string]interface{}{
    "promoted":    true,
    "instinct_id": instinctID,
    "hub_written": hubWritten,
})
```

### WR-02: normalizeQueenEntry regex strips all trailing parenthetical content, causing false-positive dedup

**File:** `cmd/queen.go:534`
**Issue:** The regex `\s*\(.*?\)\s*$` strips any trailing parenthetical from a line. This means entries with meaningful parenthetical qualifiers will be incorrectly treated as duplicates. For example, `"Use connection pooling (PostgreSQL)"` and `"Use connection pooling (SQLite)"` would normalize to the same string `"Use connection pooling"` and the second would be silently dropped. Similarly, `"Prefer immutability (functional style)"` would collide with `"Prefer immutability (OOP style)"`. While the date-suffix pattern is the primary target, the regex has no guard against stripping non-date parentheticals. The `?` (non-greedy) quantifier also means only the last parenthetical group is stripped, but the damage is the same for single-group cases.

**Fix:** Either (a) restrict the regex to known date/promotion patterns, e.g. `\s*\((promoted|phase learning|instinct|hive wisdom)[^)]*\)\s*$`, or (b) add a secondary check that compares the raw text when normalization produces a match, to confirm the non-parenthetical prefix is the only difference.

### WR-03: strings.Title is deprecated since Go 1.18

**File:** `cmd/colony_prime_context.go:282`
**Issue:** `strings.Title` has been deprecated since Go 1.18 because it does not handle Unicode properly and does not follow title-casing rules for all languages. The Go documentation recommends using `cases.Title` from `golang.org/x/text/cases` instead. While this is pre-existing code not introduced in this phase, it was not updated alongside the other changes and will produce linter warnings in modern Go toolchains.

**Fix:**
```go
import "golang.org/x/text/cases"
import "golang.org/x/text/language"

// Replace:
strings.Title(dd.domain)
// With:
cases.Title(language.English).String(dd.domain)
```

## Info

### IN-01: queen-seed-from-hive reads wisdom.json directly instead of using hubStore

**File:** `cmd/queen.go:328`
**Issue:** `queenSeedFromHiveCmd` reads `hive/wisdom.json` via `os.ReadFile` and `json.Unmarshal` directly, constructing the path manually with `filepath.Join(hub, "hive", "wisdom.json")`. However, it also calls `hubStore()` on line 321 for the QUEEN.md write. If the `hubStore()` abstraction exists for hub-level file operations (providing locking, atomic writes, etc.), bypassing it for the hive wisdom read creates an inconsistency. The same pattern of direct `os.ReadFile` for hive wisdom exists in `readHiveWisdomEntries` in `context_weighting.go`, suggesting this is a pre-existing pattern rather than a new introduction.

**Fix:** Consider routing hive wisdom reads through the hub store abstraction for consistency, or document that direct reads are intentional for read-only operations.

### IN-02: appendEntriesToQueenSection scans text twice for the same section header

**File:** `cmd/queen.go:569-613`
**Issue:** The function calls `strings.Index(text, sectionHeader)` twice: first at line 570 to extract existing entries for dedup, then again at line 601 to find the insert position. This is redundant since the section header location does not change between the two calls. The double-scan is minor from a performance perspective (QUEEN.md files are small) but adds unnecessary code complexity.

**Fix:** Store the first `idx` result and reuse it, or restructure to perform dedup and insertion in a single pass over the target section.

### IN-03: Duplicate wisdom detection uses different mechanisms in queen-seed-from-hive vs appendEntriesToQueenSection

**File:** `cmd/queen.go:358-361` and `cmd/queen.go:562-598`
**Issue:** `queen-seed-from-hive` applies `isEntryInText` (whole-text scanning) to filter duplicates before calling `appendEntriesToQueenSection`, which itself applies section-scoped dedup via `existingNormalized`. This means entries are deduplicated twice through two different code paths for the same operation. The first check uses `normalizeQueenEntry` against every line in the entire text, while the second only scans within the target section. The dual check is not harmful but is redundant and increases maintenance surface area.

**Fix:** Either remove the pre-filter in `queen-seed-from-hive` and rely solely on `appendEntriesToQueenSection`'s built-in dedup, or remove the dedup from `appendEntriesToQueenSection` and require callers to pre-filter. Pick one canonical location.

---

_Reviewed: 2026-04-26T23:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
