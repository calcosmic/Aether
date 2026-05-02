---
phase: 91-hive-intelligence
reviewed: 2026-05-02T12:00:00Z
depth: standard
files_reviewed: 15
files_reviewed_list:
  - cmd/codex_continue_finalize.go
  - cmd/skill_curator.go
  - go.mod
  - go.sum
  - pkg/learn/curator_test.go
  - pkg/learn/curator.go
  - pkg/learn/difficulty_test.go
  - pkg/learn/difficulty.go
  - pkg/learn/skills.go
  - pkg/learn/sqlite_migrations.go
  - pkg/learn/sqlite_schema.go
  - pkg/learn/sqlite_search_test.go
  - pkg/learn/sqlite_search.go
  - pkg/learn/sqlite_store_test.go
  - pkg/learn/sqlite_store.go
findings:
  critical: 2
  warning: 6
  info: 3
  total: 11
status: issues_found
---

# Phase 91: Code Review Report

**Reviewed:** 2026-05-02T12:00:00Z
**Depth:** standard
**Files Reviewed:** 15
**Status:** issues_found

## Summary

Reviewed the Phase 91 implementation: a SQLite-backed learning store with FTS5 full-text search, skill lifecycle management (curator with active/stale/archived stages), difficulty assessment, and auto-skill creation hooks integrated into the continue-finalize command. The overall architecture is sound -- proper use of WAL mode, parameterized queries, migration system with idempotency, and external content FTS5 with sync triggers. However, there are two correctness bugs that can cause silent data inconsistency, a logic error in difficulty scoring that produces false positives, and a query sanitization gap that allows malformed FTS5 queries.

## Critical Issues

### CR-01: File-then-DB update pattern causes silent data desync on failure

**File:** `pkg/learn/skills.go:190-199` (PatchSkill), `pkg/learn/skills.go:215-231` (ArchiveSkill), `pkg/learn/curator.go:95-121` (transitionStage)

**Issue:** Three functions mutate the filesystem first, then update SQLite. If the DB update fails, the file system and database are permanently out of sync. `CreateSkill` handles this correctly (lines 162-163 clean up the file on DB failure), but `PatchSkill`, `ArchiveSkill`, and `transitionStage` do not.

- **PatchSkill** (line 191): Writes new content to `SKILL.md`, then updates `patch_count`. If the DB `Exec` fails, the file has been rewritten but the DB doesn't reflect the patch.
- **ArchiveSkill** (line 221): Renames the file to the archived directory, then updates the DB stage. If the DB update fails, the file lives in `archived/` but the DB says `active/`.
- **transitionStage** (line 105): Same pattern. Worse: the `continue` on line 117 silently swallows the DB error, and on the next run the file won't be found at its old `file_path` (line 101's `os.Stat` fails), leaving a permanent desync with no error reported.

**Fix:** Either reverse the order (update DB first, then move files), or add rollback logic for the file operation when the DB update fails. At minimum, log the error instead of silently continuing:

```go
// In transitionStage -- at minimum, log the desync
_, err := c.db.Exec(`UPDATE skills SET stage = ?, file_path = ?, last_transitioned_at = ? WHERE id = ?`,
    targetStage, newPath, now.Format(time.RFC3339), s.id)
if err != nil {
    // Attempt to roll back the file move
    _ = os.Rename(newPath, s.filePath)
    return 0, fmt.Errorf("learn: update skill %q stage: %w", s.name, err)
}
```

For `ArchiveSkill` and `PatchSkill`, reverse the operation order:

```go
// ArchiveSkill: update DB first, then move file
_, err = svc.db.Exec(`UPDATE skills SET stage = ?, file_path = ?, last_transitioned_at = ? WHERE name = ?`,
    SkillStageArchived, newPath, time.Now().UTC().Format(time.RFC3339), name)
if err != nil {
    os.Remove(newPath) // clean up the empty dir
    return fmt.Errorf("learn: update skill stage: %w", err)
}
if err := os.Rename(meta.FilePath, newPath); err != nil {
    // Roll back DB
    _, _ = svc.db.Exec(`UPDATE skills SET stage = ?, file_path = ? WHERE name = ?`,
        meta.Stage, meta.FilePath, name)
    return fmt.Errorf("learn: move skill to archive: %w", err)
}
```

### CR-02: FTS5 query sanitization does not handle embedded double-quotes in tokens

**File:** `pkg/learn/sqlite_search.go:69-90`

**Issue:** `sanitizeFTS5Query` strips FTS5 special characters only from the ends of tokens via `strings.Trim`, not from within tokens. A token containing an embedded double-quote (e.g., `foo"bar`) produces a malformed FTS5 query `"foo"bar"` where the embedded quote prematurely closes the quoted token. This can cause a SQLite parse error at runtime, crashing the search operation.

The same applies to other special characters embedded within tokens -- e.g., `foo(bar)baz` passes through as `"foo(bar)baz"` which happens to be safe because parens inside quotes are literal in FTS5, but embedded quotes are not safe.

**Fix:** Replace all embedded double-quotes within tokens before wrapping:

```go
func sanitizeFTS5Query(query string) string {
    query = strings.TrimSpace(query)
    if query == "" {
        return ""
    }
    tokens := strings.Fields(query)
    var sanitized []string
    for _, token := range tokens {
        upper := strings.ToUpper(token)
        if upper == "AND" || upper == "OR" || upper == "NOT" {
            continue
        }
        // Remove ALL special characters, not just leading/trailing
        token = strings.Map(func(r rune) rune {
            if strings.ContainsRune("*\"(){}:^+", r) {
                return -1 // drop character
            }
            return r
        }, token)
        if token == "" {
            continue
        }
        sanitized = append(sanitized, `"`+token+`"`)
    }
    return strings.Join(sanitized, " AND ")
}
```

## Warnings

### WR-01: AssessDifficulty marks tasks as difficult when they are not

**File:** `pkg/learn/difficulty.go:87-101`

**Issue:** Check 3 (line 88) adds a reason and score contribution for having 3+ workers, regardless of task outcome. The final `IsDifficult` on line 98 uses `score >= DifficultyScoreThreshold || len(reasons) > 0`. Since having 3+ workers always adds a reason, any task with 3+ workers that all completed successfully with all gates passed is still marked `IsDifficult = true` (because `len(reasons) > 0`). This produces false-positive difficulty assessments, potentially triggering unnecessary auto-skill creation.

**Fix:** Only add the multi-worker reason if the task actually had some difficulty signal:

```go
// Check 3: Multiple workers indicates complex task (only if no other difficulty signals)
if len(evidence.Workers) >= 3 && len(reasons) == 0 {
    reasons = append(reasons, fmt.Sprintf("%d workers involved", len(evidence.Workers)))
    score += 0.1
}
```

Or remove the `|| len(reasons) > 0` fallback and rely solely on the score threshold.

### WR-02: AssessDifficulty ignores worker failures when gates did not pass

**File:** `pkg/learn/difficulty.go:69`

**Issue:** The condition `failures > 0 && evidence.GatesPassed > 0` requires gates to have passed before counting worker failures. If workers failed AND gates also failed, the worker failures are completely ignored. This scenario (workers failing AND gates failing) should be the strongest difficulty signal, not a no-op.

**Fix:** Change the condition to check for worker failures independently:

```go
if failures > 0 {
    reasons = append(reasons, fmt.Sprintf("%d worker(s) failed before success", failures))
    weight := float64(failures) / float64(len(evidence.Workers))
    if weight > 1.0 {
        weight = 1.0
    }
    score += 0.3 * weight
}
```

### WR-03: Missing `rows.Err()` check after every `for rows.Next()` loop

**File:** `pkg/learn/sqlite_store.go:133-139` (List), `pkg/learn/sqlite_store.go:199-209` (Compact), `pkg/learn/sqlite_search.go:53-59` (Search), `pkg/learn/skills.go:278-284` (ListSkills)

**Issue:** After every `for rows.Next()` loop, the code does not call `rows.Err()` to check for iteration errors. If the loop exits due to a database error (network failure, disk I/O, context cancellation) rather than `rows.Next()` returning false, the error is silently swallowed and a partial/incomplete result set is returned as if it were complete.

**Fix:** Add `rows.Err()` check after each loop:

```go
for rows.Next() {
    // ... scan ...
}
if err := rows.Err(); err != nil {
    return nil, fmt.Errorf("learn: iterate entries: %w", err)
}
```

### WR-04: `defer` inside closure defers to wrong function scope

**File:** `cmd/codex_continue_finalize.go:335`

**Issue:** `defer sqliteStore.Close()` is inside the `captureLearning` closure, but Go's `defer` is scoped to the enclosing function (`runCodexContinueFinalize`), not the closure. The SQLite connection stays open until `runCodexContinueFinalize` returns -- long after the closure has finished. Since the SQLite store uses `MaxOpenConns(1)`, this held connection could block concurrent operations on the same database.

**Fix:** Replace `defer` with an explicit close:

```go
sqliteStore, sqliteErr := learn.NewSQLiteColonyStore(filepath.Join(store.BasePath(), "colony.db"))
if sqliteErr == nil {
    aetherRoot := storage.ResolveAetherRoot(context.Background())
    mode := learn.LoadAutoSkillMode(store.BasePath())
    if err := learn.AutoCreateSkillIfDifficult(entry, sqliteStore, aetherRoot, mode); err != nil {
        fmt.Fprintf(os.Stderr, "warning: failed to auto-create skill: %v\n", err)
    }
    sqliteStore.Close() // explicit close, not deferred
}
```

### WR-05: `buildSkillContent` uses `filepath.Base` on free-text content

**File:** `pkg/learn/difficulty.go:237`

**Issue:** `filepath.Base(entry.Content)` is used to generate the skill's heading. `filepath.Base` extracts the last path component -- for content like "fixed bug in pkg/auth/middleware.go", it returns "middleware.go" instead of the full content. For content without slashes, it happens to return the full string, but the function's semantic intent (generating a heading from free text) does not match `filepath.Base`'s purpose (extracting a filename from a path).

**Fix:** Use a proper title generation approach:

```go
// Use first line or first N characters as the heading
heading := entry.Content
if idx := strings.Index(heading, "\n"); idx >= 0 {
    heading = heading[:idx]
}
if len(heading) > 100 {
    heading = heading[:100] + "..."
}
b.WriteString("# ")
b.WriteString(heading)
```

### WR-06: `skill-recover` CLI command does not validate the name argument

**File:** `cmd/skill_curator.go:57-58`

**Issue:** The `skill-recover` command passes `args[0]` directly to `RecoverSkill`, which constructs a file path via `filepath.Join(skillDirForStage(...), name)`. While `validateSkillName` exists in `skills.go`, it is only called by `CreateSkill`. The recover path does not call it, meaning a user could potentially pass a name containing characters that `validateSkillName` would reject, leading to unexpected directory creation.

**Fix:** Add name validation in the CLI handler:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    name := args[0]
    if err := learn.ValidateSkillName(name); err != nil {
        outputError(2, fmt.Sprintf("invalid skill name: %v", err), nil)
        return nil
    }
    // ... rest of handler
```

Note: `validateSkillName` is currently unexported; it should be exported as `ValidateSkillName` for use by the CLI layer.

## Info

### IN-01: Duplicate `scanSkillMetadata` and `scanSkillMetadataFromRows` functions

**File:** `pkg/learn/skills.go:348-396`

**Issue:** `scanSkillMetadata` (line 348) and `scanSkillMetadataFromRows` (line 375) have identical scanning logic differing only in whether they accept `*sql.Row` or `*sql.Rows`. The same duplication exists for `scanEntry` and `scanEntryFromRows` in `sqlite_store.go`. Go's `*sql.Row` is a thin wrapper, but the idiomatic way to deduplicate is to have the `*sql.Row` variant call `QueryRow` and reuse the rows scanner.

**Fix:** Consider using a helper that accepts `scanner` interface, or accept the duplication as a minor style choice.

### IN-02: `ExtractKeywords` is unexported but has thorough test coverage

**File:** `pkg/learn/difficulty.go:208-230`

**Issue:** `extractKeywords` is unexported, which is correct since it's an internal helper for `deriveSkillName`. However, the stop words list (line 211-215) is hardcoded. If the project grows, consider making this configurable or extracting it to a constant.

### IN-03: `TestAssessDifficulty_EasyTask` does not test the 3+ worker edge case

**File:** `pkg/learn/difficulty_test.go:63-79`

**Issue:** The "easy task" test uses only 1 worker, so it does not catch the false-positive described in WR-01. Adding a test with 3+ workers, all completed, all gates passed, would immediately expose the logic bug.

**Fix:** Add a test case:

```go
func TestAssessDifficulty_EasyTaskWithMultipleWorkers(t *testing.T) {
    evidence := Evidence{
        Workers: []WorkerEvidence{
            {Name: "B1", Caste: "builder", Status: "completed"},
            {Name: "B2", Caste: "builder", Status: "completed"},
            {Name: "B3", Caste: "builder", Status: "completed"},
        },
        GatesPassed: 3,
        GatesTotal:  3,
    }
    assessment := AssessDifficulty(evidence)
    if assessment.IsDifficult {
        t.Errorf("3 workers all completed should NOT be difficult, got IsDifficult=true (score=%.2f, reasons=%v)",
            assessment.Score, assessment.Reasons)
    }
}
```

---

_Reviewed: 2026-05-02T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
