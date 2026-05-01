---
phase: 67-seal-hive-brain-wiring
reviewed: 2026-04-28T01:35:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - .claude/commands/ant/seal.md
  - .opencode/commands/ant/seal.md
  - cmd/codex_workflow_cmds.go
  - cmd/hive.go
  - cmd/seal_ceremony_test.go
findings:
  critical: 1
  warning: 4
  info: 3
  total: 8
status: issues_found
---

# Phase 67: Code Review Report

**Reviewed:** 2026-04-28T01:35:00Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

Reviewed 5 files implementing the seal ceremony hive brain wiring: two command wrapper markdown files (identical content for Claude and OpenCode), the Go seal ceremony logic in `codex_workflow_cmds.go`, the hive storage implementation in `hive.go`, and the corresponding test file. The implementation correctly wires hive promotion into the seal ceremony as non-blocking, and tests cover the key paths including blocker handling, promotion success, and non-blocking failure.

One critical bug found: the hive promotion count is inflated when instincts have high confidence but empty Action text, producing a misleading "Promoted N instinct(s)" message. Several warnings around silent error swallowing in JSON deserialization and write operations. The `sourceRepo` is always passed empty from the seal ceremony, disabling the documented multi-repo confidence boosting feature.

## Critical Issues

### CR-01: Hive promoted count inflated for instincts with empty Action

**File:** `cmd/codex_workflow_cmds.go:299-311`
**Issue:** The local promotion check on line 294 correctly guards with `entry.Confidence >= 0.8 && entry.Action != ""`, but the hive promotion check on line 299 only checks `entry.Confidence >= 0.8`. When `entry.Action` is empty, `promoteToHive("")` returns `nil` immediately (hive.go line 247) without storing anything, but the `else` branch on line 309-310 still executes, incrementing `hivePromotedCount`. This inflates the count shown to the user in "Promoted N instinct(s) to Hive Brain" and in the CROWNED-ANTHILL.md statistics table, reporting more promotions than actually occurred.

**Fix:**
```go
// Line 299: add the same empty Action guard
if entry.Confidence >= 0.8 && entry.Action != "" {
    hiveEligibleCount++
    // Hive Brain promotion (non-blocking per CERE-02)
    domain := entry.Domain
    if domain == "" {
        domain = "general"
    }
    if err := promoteToHive(entry.Action, domain, "", entry.Confidence); err != nil {
        log.Printf("seal: hive-promote failed for %s: %v", entry.ID, err)
        hivePromotionFailures++
    } else {
        hivePromotedCount++
    }
}
```

## Warnings

### WR-01: sourceRepo always empty from seal, disabling multi-repo confidence boosting

**File:** `cmd/codex_workflow_cmds.go:306`
**Issue:** `promoteToHive(entry.Action, domain, "", entry.Confidence)` always passes empty string for `sourceRepo`. According to CLAUDE.md, the multi-repo confidence boosting feature relies on `source_repo` to track which repos confirmed the same wisdom. With this always empty, the feature cannot differentiate same-repo re-promotions from cross-repo confirmations. Wisdom entries will have empty `source_repo` fields, losing traceability.

**Fix:** Determine the current repository name (e.g., from git remote or colony registry) and pass it to `promoteToHive`:
```go
repoName := detectRepoName() // or read from colony registry
if err := promoteToHive(entry.Action, domain, repoName, entry.Confidence); err != nil {
```

### WR-02: Three json.Unmarshal calls silently ignore errors in hive.go

**File:** `cmd/hive.go:95,180,271`
**Issue:** Three locations read and unmarshal the wisdom file but discard the `json.Unmarshal` error:
- Line 95: `json.Unmarshal(raw, &wf)` in `hive-store`
- Line 180: `json.Unmarshal(raw, &wf)` in `hive-read`
- Line 271: `json.Unmarshal(raw, &wf)` in `promoteToHive`

If the wisdom.json file is corrupted or contains malformed JSON, the function proceeds with a zero-initialized `hiveWisdomData` struct and will overwrite the file with an empty entries list, destroying all existing wisdom data.

**Fix:** Check the error return and fail rather than proceeding with empty data:
```go
if raw, err := os.ReadFile(wisdomPath); err == nil {
    if err := json.Unmarshal(raw, &wf); err != nil {
        outputError(2, fmt.Sprintf("corrupted wisdom.json: %v", err), nil)
        return nil
    }
}
```

### WR-03: writeWisdom error silently ignored in hive-read

**File:** `cmd/hive.go:201`
**Issue:** `writeWisdom(wisdomPath, wf)` discards the error return. Every other call site in hive.go properly checks the error from `writeWisdom`. If the write fails (disk full, permissions), the access time updates are lost with no indication to the caller.

**Fix:**
```go
if err := writeWisdom(wisdomPath, wf); err != nil {
    outputError(2, fmt.Sprintf("failed to persist access updates: %v", err), nil)
    return nil
}
```

### WR-04: hive-store computes textHash but deduplication does not use it

**File:** `cmd/hive.go:99-101`
**Issue:** Line 99 computes `textHash := fmt.Sprintf("%x", sha256.Sum256([]byte(text)))` before the dedup loop, but the dedup check on line 101 compares `e.Text == text` (exact string comparison). The hash is only used later for ID generation on line 134. This is wasteful (SHA-256 computation on every call) and the comment "Dedup: check if same text+domain already exists" is misleading because the hash is not part of the dedup logic. If the intent was hash-based dedup for efficiency or to handle very long texts, it should be used in the comparison.

**Fix:** Move the hash computation to after the dedup check, or use it in the dedup comparison:
```go
// Dedup: check if same text+domain already exists
for i, e := range wf.Entries {
    if e.Text == text && e.Domain == domain {
        // Reinforce ...
        return nil
    }
}
// Generate ID hash only after dedup confirms this is a new entry
textHash := fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
```

## Info

### IN-01: Both seal.md command wrappers are byte-identical

**File:** `.claude/commands/ant/seal.md` and `.opencode/commands/ant/seal.md`
**Issue:** The Claude and OpenCode seal command wrappers are identical. This is not necessarily a bug (the architecture docs confirm both platforms delegate to the same Go runtime), but the OpenCode version should be verified to work correctly with the OpenCode agent runtime, particularly the `$ARGUMENTS` interpolation and the post-seal Porter delivery section.

**Fix:** No code change needed; confirm OpenCode compatibility during integration testing.

### IN-02: writeWisdom comment claims "atomically" but uses os.WriteFile

**File:** `cmd/hive.go:370-379`
**Issue:** The function comment says "writes the wisdom file atomically" but `os.WriteFile` is not atomic -- it truncates then writes, leaving a window for partial writes on crash. The project docs mention hub-level file locking, but this function does not acquire any lock. The non-atomic write is acceptable for the current use case but the comment should not claim atomicity.

**Fix:** Update the comment to reflect the actual behavior:
```go
// writeWisdom writes the wisdom file to disk.
func writeWisdom(path string, wf hiveWisdomData) error {
```

### IN-03: TestSealHiveEligibleLog does not verify hive-eligible count in output

**File:** `cmd/seal_ceremony_test.go:392-430`
**Issue:** `TestSealHiveEligibleLog` checks for "Promoted" and "Hive Brain" in output but does not verify the count (2 instincts, so output should say "Promoted 2 instinct(s)"). It also does not verify the hive-eligible count appears anywhere. The test name suggests verifying the eligibility log, but the assertions only check for substring presence.

**Fix:** Add a count-specific assertion:
```go
if !strings.Contains(out, "Promoted 2 instinct(s) to Hive Brain") {
    t.Errorf("expected 'Promoted 2 instinct(s) to Hive Brain', got: %s", out)
}
```

---

_Reviewed: 2026-04-28T01:35:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
