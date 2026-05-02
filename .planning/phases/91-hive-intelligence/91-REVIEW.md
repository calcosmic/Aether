---
phase: 91-hive-intelligence
reviewed: 2026-05-02T12:00:00Z
depth: standard
files_reviewed: 18
files_reviewed_list:
  - cmd/codex_continue_finalize.go
  - cmd/hive_search.go
  - cmd/skill_curator.go
  - cmd/skill_lifecycle.go
  - go.mod
  - go.sum
  - pkg/learn/curator.go
  - pkg/learn/curator_test.go
  - pkg/learn/difficulty.go
  - pkg/learn/difficulty_test.go
  - pkg/learn/skills.go
  - pkg/learn/skills_test.go
  - pkg/learn/sqlite_migrations.go
  - pkg/learn/sqlite_schema.go
  - pkg/learn/sqlite_search.go
  - pkg/learn/sqlite_search_test.go
  - pkg/learn/sqlite_store.go
  - pkg/learn/sqlite_store_test.go
findings:
  critical: 3
  warning: 6
  info: 3
  total: 12
status: issues_found
---

# Phase 91: Code Review Report

**Reviewed:** 2026-05-02T12:00:00Z
**Depth:** standard
**Files Reviewed:** 18
**Status:** issues_found

## Summary

Reviewed 18 files comprising the Phase 91 hive intelligence implementation: SQLite-backed learning store with FTS5 full-text search, skill lifecycle management (create/archive/pin/promote/recover), curator transitions (active/stale/archived), difficulty assessment, and auto-skill creation. Also reviewed the integration point in `codex_continue_finalize.go` where learning capture and auto-skill hooks fire, plus the CLI command files `hive_search.go`, `skill_curator.go`, and `skill_lifecycle.go`.

The overall architecture is structurally sound with good test coverage, proper WAL mode, parameterized queries, idempotent migrations, and FTS5 external content with sync triggers. However, three critical issues were found: (1) the auto-skill creation feature is dead code from the continue-finalize path because `FilesTouched` is always nil, (2) file-move-then-DB-update operations in archive/transition paths create silent data inconsistency on DB failure, and (3) the `defer` inside the `captureLearning` closure defers to the wrong function scope, leaking the SQLite connection.

## Critical Issues

### CR-01: Auto-skill creation is dead code -- FilesTouched always nil in continue-finalize

**File:** `cmd/codex_continue_finalize.go:277`
**Issue:** Every `WorkerResult` is constructed with `FilesTouched: nil` because `codexContinueWorkerFlowStep` has no `FilesModified` field. The `CollectEvidence` function propagates this nil into `Evidence.FilesTouched`, making it empty. Then `IsAutoSkillRejected` (difficulty.go:118) rejects the entry because `len(entry.Evidence.FilesTouched) == 0`.

This means the entire auto-skill creation pipeline (AUTO-01 through AUTO-03) -- difficulty assessment, mode checks, skill creation -- will never produce a skill from the continue-finalize path. The feature is effectively dead code despite being wired up with config loading, mode checks, and non-blocking error handling.

```go
// cmd/codex_continue_finalize.go:272-278
workerResults := make([]learn.WorkerResult, 0, len(workerFlow))
for _, step := range workerFlow {
    workerResults = append(workerResults, learn.WorkerResult{
        Name:         step.Name,
        Caste:        step.Caste,
        Status:       step.Status,
        FilesTouched: nil, // <-- always nil, kills auto-skill creation
    })
}
```

**Fix:** Either populate `FilesTouched` from available data, or remove the `FilesTouched == 0` rejection in `IsAutoSkillRejected` when the entry comes from continue-finalize. The cleanest fix is to add a `FilesModified` field to the worker flow step structs:

```go
// Option A: Add field to codexContinueWorkerFlowStep
type codexContinueWorkerFlowStep struct {
    // ...existing fields...
    FilesModified []string `json:"files_modified,omitempty"`
}

// Then in the evidence collection loop:
workerResults = append(workerResults, learn.WorkerResult{
    Name:         step.Name,
    Caste:        step.Caste,
    Status:       step.Status,
    FilesTouched: step.FilesModified,
})
```

### CR-02: File-move-then-DB-update creates inconsistent state on DB failure

**File:** `pkg/learn/skills.go:231-240` (ArchiveSkill), `pkg/learn/skills.go:200-209` (PatchSkill), `pkg/learn/curator.go:100-119` (transitionStage)
**Issue:** Three functions mutate the filesystem first, then update SQLite. If the file operation succeeds but the DB update fails, the file is in the new location but the DB still references the old path and old stage. The skill becomes unreachable.

- **PatchSkill** (line 200): Rewrites `SKILL.md` content, then updates `patch_count`. If DB fails, file has new content but DB does not reflect the patch.
- **ArchiveSkill** (line 231): Renames the file to archived directory, then updates stage. If DB fails, file lives in `archived/` but DB says `active/`.
- **transitionStage** (line 105): Same pattern. Worse: the `continue` on line 117 silently swallows the DB error, and on the next run the file won't be found at its old path, leaving a permanent desync with no error reported.

With `MaxOpenConns(1)` on SQLite, concurrent access can cause `database is locked` errors triggering this exact scenario.

**Fix:** Update the DB first (atomic within SQLite), then move the file. If the file move fails, roll back the DB:

```go
func (svc *SkillService) ArchiveSkill(name string) error {
    // ...existing validation...
    now := time.Now().UTC().Format(time.RFC3339)
    res, err := svc.db.Exec(`UPDATE skills SET stage = ?, file_path = ?, last_transitioned_at = ? WHERE name = ?`,
        SkillStageArchived, newPath, now, name)
    if err != nil {
        return fmt.Errorf("learn: update skill stage: %w", err)
    }
    n, _ := res.RowsAffected()
    if n == 0 {
        return fmt.Errorf("learn: skill %q not found", name)
    }
    if err := os.MkdirAll(newDir, 0755); err != nil {
        svc.db.Exec(`UPDATE skills SET stage = ?, file_path = ? WHERE name = ?`,
            meta.Stage, meta.FilePath, name)
        return fmt.Errorf("learn: create archive dir: %w", err)
    }
    if err := os.Rename(meta.FilePath, newPath); err != nil {
        svc.db.Exec(`UPDATE skills SET stage = ?, file_path = ? WHERE name = ?`,
            meta.Stage, meta.FilePath, name)
        return fmt.Errorf("learn: move skill to archive: %w", err)
    }
    return nil
}
```

### CR-03: defer inside closure defers to wrong function scope, leaking SQLite connection

**File:** `cmd/codex_continue_finalize.go:335`
**Issue:** `defer sqliteStore.Close()` is inside the `captureLearning` closure, but Go's `defer` is scoped to the enclosing function (`runCodexContinueFinalize`), not the closure. The SQLite connection stays open until `runCodexContinueFinalize` returns -- long after the closure has finished executing. Since the SQLite store uses `MaxOpenConns(1)`, this held connection could block concurrent operations on the same database for the entire remainder of the finalize function.

```go
captureLearning := func() {
    // ...
    sqliteStore, sqliteErr := learn.NewSQLiteColonyStore(...)
    if sqliteErr == nil {
        defer sqliteStore.Close() // <-- defers to runCodexContinueFinalize, not captureLearning
        // ...
        learn.AutoCreateSkillIfDifficult(entry, sqliteStore, ...)
    }
}
captureLearning() // runs immediately, but connection stays open until enclosing func returns
```

**Fix:** Replace `defer` with an explicit close immediately after use:

```go
sqliteStore, sqliteErr := learn.NewSQLiteColonyStore(filepath.Join(store.BasePath(), "colony.db"))
if sqliteErr == nil {
    aetherRoot := storage.ResolveAetherRoot(context.Background())
    mode := learn.LoadAutoSkillMode(store.BasePath())
    if err := learn.AutoCreateSkillIfDifficult(entry, sqliteStore, aetherRoot, mode); err != nil {
        fmt.Fprintf(os.Stderr, "warning: failed to auto-create skill: %v\n", err)
    }
    sqliteStore.Close()
}
```

## Warnings

### WR-01: AssessDifficulty.IsDifficult contradicts DifficultyScoreThreshold

**File:** `pkg/learn/difficulty.go:98`
**Issue:** The `IsDifficult` field uses `score >= DifficultyScoreThreshold || len(reasons) > 0`. The `|| len(reasons) > 0` clause means any task with 3+ workers (Check 3) is always "difficult" even with score 0.1 -- well below the 0.3 threshold. A task with 3 workers that all completed successfully and all gates passed is marked `IsDifficult = true`, inflating auto-skill creation once CR-01 is fixed.

```go
return DifficultyAssessment{
    IsDifficult: score >= DifficultyScoreThreshold || len(reasons) > 0,
    // score=0.1, reasons=["3 workers involved"] => IsDifficult=true despite threshold
}
```

**Fix:** Remove the `|| len(reasons) > 0` clause and rely solely on the score threshold:

```go
return DifficultyAssessment{
    IsDifficult: score >= DifficultyScoreThreshold,
    Reasons:     reasons,
    Score:       score,
}
```

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

### WR-03: Missing rows.Err() checks on all SQL iteration loops

**File:** `pkg/learn/sqlite_store.go:133` (List), `pkg/learn/sqlite_store.go:199` (Compact), `pkg/learn/sqlite_search.go:53` (Search), `pkg/learn/skills.go:330` (ListSkills)
**Issue:** After every `for rows.Next()` loop, the code does not call `rows.Err()` to check for iteration errors. If the loop exits due to a database error (connection dropped, disk I/O error, context cancellation) rather than `rows.Next()` returning false, the error is silently swallowed and a partial result set is returned as if complete.

**Fix:** Add `rows.Err()` check after every `rows.Next()` loop:

```go
for rows.Next() {
    entry, err := scanEntryFromRows(rows)
    if err != nil {
        return nil, fmt.Errorf("learn: scan entry: %w", err)
    }
    result = append(result, *entry)
}
if err := rows.Err(); err != nil {
    return nil, fmt.Errorf("learn: iterate entries: %w", err)
}
```

### WR-04: FTS5 query sanitization does not handle embedded double-quotes in tokens

**File:** `pkg/learn/sqlite_search.go:83`
**Issue:** `sanitizeFTS5Query` strips FTS5 special characters only from the ends of tokens via `strings.Trim`, not from within tokens. A token containing an embedded double-quote (e.g., `foo"bar`) produces a malformed FTS5 query `"foo"bar"` where the embedded quote prematurely closes the quoted token. This can cause a SQLite parse error at runtime.

**Fix:** Remove ALL special characters from within tokens, not just leading/trailing:

```go
token = strings.Map(func(r rune) rune {
    if strings.ContainsRune("*\"(){}:^+", r) {
        return -1 // drop character
    }
    return r
}, token)
```

### WR-05: UnpinSkill exists in skills.go but has no CLI command

**File:** `pkg/learn/skills.go:260-267` and `cmd/skill_lifecycle.go`
**Issue:** `SkillService.UnpinSkill` is implemented but no `skill-unpin` cobra command is registered. Users can pin skills via `aether skill-pin` but have no CLI path to unpin them. The only way to unpin is to directly modify the SQLite database.

**Fix:** Add a `skill-unpin` command in `cmd/skill_lifecycle.go`:

```go
var skillUnpinCmd = &cobra.Command{
    Use:   "skill-unpin [name]",
    Short: "Unpin a skill to re-enable auto-transitions and agent writes",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        name := args[0]
        dbPath := resolveColonyDBPath()
        sqliteStore, err := learn.NewSQLiteColonyStore(dbPath)
        if err != nil {
            outputError(2, fmt.Sprintf("database: %v", err), nil)
            return nil
        }
        defer sqliteStore.Close()
        svc := learn.NewSkillService(sqliteStore.DB(), resolveSkillBaseDir())
        if err := svc.UnpinSkill(name); err != nil {
            outputError(2, fmt.Sprintf("failed to unpin skill: %v", err), nil)
            return nil
        }
        outputOK(map[string]interface{}{"unpinned": true, "name": name})
        return nil
    },
}
```

### WR-06: `skill-recover` CLI command does not validate the name argument

**File:** `cmd/skill_curator.go:57-58`
**Issue:** The `skill-recover` command passes `args[0]` directly to `RecoverSkill`, which constructs a file path via `filepath.Join(skillDirForStage(...), name)`. While `validateSkillName` exists in `skills.go`, it is only called by `CreateSkill`. The recover path does not call it, meaning a user could pass a name containing path traversal characters that `validateSkillName` would reject.

**Fix:** Export `validateSkillName` as `ValidateSkillName` and add validation in the CLI handler:

```go
if err := learn.ValidateSkillName(name); err != nil {
    outputError(2, fmt.Sprintf("invalid skill name: %v", err), nil)
    return nil
}
```

## Info

### IN-01: Duplicate scanEntry/scanSkillMetadata code

**File:** `pkg/learn/sqlite_store.go:270-302` and `pkg/learn/skills.go:399-448`
**Issue:** `scanEntry` and `scanEntryFromRows` are nearly identical (differing only in `*sql.Row` vs `*sql.Rows`). Same duplication exists for `scanSkillMetadata` and `scanSkillMetadataFromRows`. This is a common Go pattern (the stdlib provides no unified scanner interface).

**Fix:** Accept as idiomatic Go, or introduce a small helper interface.

### IN-02: Go 1.26.1 in go.mod may be unreleased

**File:** `go.mod:3`
**Issue:** `go 1.26.1` may not correspond to a released Go version. This could cause build failures for contributors on stable Go releases.

**Fix:** Verify this is intentional. If so, document the required toolchain.

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
