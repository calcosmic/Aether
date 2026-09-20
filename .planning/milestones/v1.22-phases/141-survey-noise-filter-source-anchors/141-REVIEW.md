---
phase: 141-survey-noise-filter-source-anchors
reviewed: 2026-05-18T22:00:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - pkg/codegraph/scan_filter.go
  - pkg/codegraph/scan_filter_test.go
  - pkg/codegraph/codegraph.go
  - cmd/codex_colonize.go
  - cmd/codex_colonize_test.go
  - cmd/codex_plan.go
  - cmd/init_research.go
  - cmd/skills.go
  - cmd/discuss_analyze.go
findings:
  critical: 0
  warning: 3
  info: 2
  total: 5
status: issues_found
---

# Phase 141: Code Review Report

**Reviewed:** 2026-05-18T22:00:00Z
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Phase 141 introduces a canonical `ScanFilter` in `pkg/codegraph/scan_filter.go` consolidating five previously-divergent directory skip lists into a single `ShouldSkipDir()` function. It also adds `extractSourceAnchors` for collecting source file paths during survey, and wires `SourceAnchors` through the survey JSON -> planner context pipeline. The core filter logic is clean and well-tested. Three issues were found: a missing root guard in `skills.go`, a dropped data path in `loadCodexSurveyContext`, and an unused `SourceAnchors` field at the consumption site.

## Warnings

### WR-01: Missing root-path guard on ShouldSkipDir in skills.go workspace snapshot walk

**File:** `cmd/skills.go:1268`
**Issue:** The `getWorkspaceSnapshot` function calls `codegraph.ShouldSkipDir(d.Name())` without guarding `path != cacheKey`. All other call sites in the codebase (`codex_colonize.go:514`, `codex_colonize.go:914`, `init_research.go:1853`, `discuss_analyze.go:132`) correctly use `path != root` (or equivalent) to prevent skipping the root directory itself. The `skills.go` call site omits this guard.

If `cacheKey` happens to be a directory whose basename matches a noise entry (e.g., a directory literally named `vendor`, `build`, `bin`, `tmp`, `dist`, or `coverage`), the entire walk is immediately skipped, producing an empty snapshot. While unlikely in practice, this is an inconsistency that could cause silent data loss in edge cases (e.g., CI runners that checkout into oddly-named directories).

**Fix:**
```go
// Line 1268 of skills.go -- add path != cacheKey guard
if path != cacheKey && codegraph.ShouldSkipDir(d.Name()) {
    return filepath.SkipDir
}
```

### WR-02: Live SourceAnchors from surveyWorkspace silently discarded in loadCodexSurveyContext

**File:** `cmd/codex_plan.go:1247-1259`
**Issue:** `loadCodexSurveyContext` calls `surveyWorkspace(root)` which internally runs `extractSourceAnchors(root, 50)` -- a full directory walk. However, `facts.SourceAnchors` is never merged into `ctx.SourceAnchors`. Only the JSON file path is used (line 1243-1244 reading `anchors.json`). This means:

1. On fresh colonies with no prior survey (no `anchors.json`), `SourceAnchors` is always empty even though the full filesystem walk already computed them.
2. `extractSourceAnchors` runs as part of `surveyWorkspace` but its result is thrown away -- wasted computation.

**Fix:**
```go
// After line 1258, before the uniqueSortedStrings block, add:
if len(ctx.SourceAnchors) == 0 {
    ctx.SourceAnchors = facts.SourceAnchors
}
```

### WR-03: SourceAnchors populated in context but never rendered into planning worker brief

**File:** `cmd/codex_plan.go:56` and `cmd/codex_plan.go:1533`
**Issue:** `codexSurveyContext.SourceAnchors` is populated in `loadCodexSurveyContext` (line 1199, 1244, 1270) but never consumed. The `renderPlanningWorkerBrief` function does not reference `survey.SourceAnchors` or `survey.SourceAnchors` anywhere in its output. This means the anchors are collected, stored, and loaded but never reach the planning workers that could use them for grounded plan generation -- defeating the stated purpose of source anchors ("source files for plan grounding").

This is likely a wiring gap where the field was added to the struct and loading path but the rendering side was not completed.

**Fix:** Add a source anchors section to `renderPlanningWorkerBrief`, for example after the graph context section:
```go
if len(survey.SourceAnchors) > 0 {
    b.WriteString("\nSource anchor files (read these first for grounded planning):\n")
    for _, anchor := range survey.SourceAnchors {
        b.WriteString("- ")
        b.WriteString(anchor)
        b.WriteString("\n")
    }
}
```

## Info

### IN-01: .d.ts files excluded by compound noise suffixes

**File:** `pkg/codegraph/scan_filter.go:72`
**Issue:** `.d.ts` (TypeScript declaration files) are listed in `compoundNoiseSuffixes`. While auto-generated `.d.ts` files are noise, hand-written `.d.ts` files exist in some projects (especially library packages with explicit type declarations). Consider whether this exclusion should only apply to auto-generated `.d.ts` files in `node_modules/` (which are already skipped by the directory filter).

### IN-02: extractSourceAnchors hardcoded cap of 50

**File:** `cmd/codex_colonize.go:572` and `cmd/codex_colonize.go:902`
**Issue:** The max anchors cap is hardcoded to 50 at the call site (`extractSourceAnchors(root, 50)`). This magic number appears in two places and should ideally be a named constant or configurable parameter to make the intent clear and allow adjustment without searching the codebase.

---

_Reviewed: 2026-05-18T22:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
