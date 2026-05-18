---
phase: 141-survey-noise-filter-source-anchors
verified: 2026-05-18T20:15:00Z
status: passed
score: 9/9 must-haves verified
overrides_applied: 0
---

# Phase 141: Survey Noise Filter + Source Anchors Verification Report

**Phase Goal:** Unify 5 divergent skip lists into a single canonical ScanFilter, extend it to cover Python/Node/Rust/Go ecosystem noise directories, and extract source anchors from cleaned survey output so plans can reference concrete files.
**Verified:** 2026-05-18T20:15:00Z
**Status:** passed
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | All 5 call sites (codegraph Scan, surveyWorkspace, init_research, skills, discuss_analyze) use ShouldSkipDir from scan_filter.go | VERIFIED | `codegraph.go:508` uses `ShouldSkipDir` (same package). `codex_colonize.go:454,523,914` uses `codegraph.ShouldSkipDir`. `init_research.go:1853` uses `codegraph.ShouldSkipDir`. `skills.go:1268,1314` uses `codegraph.ShouldSkipDir`. `discuss_analyze.go:132` uses `codegraph.ShouldSkipDir`. Grep for `dirsToSkip|shouldSkipSurveyDir|extendedSkipDirs|skillScanSkipDirs` in cmd/ returns zero matches. |
| 2 | ShouldSkipDir returns true for .venv, __pycache__, site-packages, .pytest_cache, .tox, .mypy_cache, node_modules, .gradle, .cache, target, .cargo | VERIFIED | `scan_filter.go` noiseDirs map contains all listed entries (30 total). `TestShouldSkipDir_ExtendedCoverage` explicitly asserts each. `TestShouldSkipDir` table-driven test covers all 30 entries. |
| 3 | No local skip-list maps remain in the codebase | VERIFIED | Grep for `dirsToSkip|shouldSkipSurveyDir|extendedSkipDirs|skillScanSkipDirs` across `cmd/` and `pkg/codegraph/` returns zero matches (excluding comments). `TestSkipListDivergence` regression test enforces this at test time. |
| 4 | A fixture with .venv/site-packages noise produces zero .venv references in survey output | VERIFIED | `TestVenvNoiseExclusion` creates `.venv/lib/site-packages/requests/api.py` and asserts zero `.venv` or `site-packages` paths. Test passes. `TestVenvNoiseExclusion_SourceFilesPreserved` verifies `src/app.py` is still present. |
| 5 | Source anchors are extracted during surveyWorkspace using the shared ScanFilter | VERIFIED | `extractSourceAnchors` function at `codex_colonize.go:902` calls `codegraph.ShouldSkipDir(d.Name())` at line 914 and `codegraph.ShouldSkipFile(base)` at line 921. Called from `surveyWorkspace` at line 572. |
| 6 | Anchors are capped at 50, sorted by path depth (shallowest first), then alphabetically | VERIFIED | Sort at lines 938-945 uses `strings.Count(candidates[i], "/")` for depth, then alphabetical tiebreak. Cap at line 947: `if len(candidates) > maxAnchors { candidates = candidates[:maxAnchors] }`. `TestExtractSourceAnchors_CapAt50` and `TestExtractSourceAnchors_SortsByDepthThenAlpha` pass. |
| 7 | Anchors exclude test files, config files, and noise directories | VERIFIED | `extractSourceAnchors` calls `isTestFile(base)` at line 923, `isConfigFile(base)` at line 923, and `codegraph.ShouldSkipDir`/`ShouldSkipFile` at lines 914/921. `TestExtractSourceAnchors_ExcludesNoise`, `TestExtractSourceAnchors_ExcludesTests`, `TestExtractSourceAnchors_ExcludesMinified` all pass. |
| 8 | anchors.json is written to .aether/data/survey/ during survey output | VERIFIED | `writeSurveyCompatibilityJSON` at `codex_colonize.go:977-981` adds `"anchors.json"` entry with `source_anchors`, `anchor_count`, and `summary` fields. `TestAnchorsWrittenToSurvey` passes. |
| 9 | loadCodexSurveyContext reads anchors.json and populates SourceAnchors in codexSurveyContext | VERIFIED | `codex_plan.go:56` has `SourceAnchors []string` field. Line 1243-1244: `readSummary("anchors.json")` reads payload and populates via `jsonStringSlice`. Line 1270: `ctx.SourceAnchors = uniqueSortedStrings(ctx.SourceAnchors)`. `TestLoadSurveyContext_IncludesAnchors` and `TestLoadSurveyContext_AnchorsEmptyWhenNoFile` pass. |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `pkg/codegraph/scan_filter.go` | Canonical ScanFilter with ShouldSkipDir, ShouldSkipFile, NoiseDirCount | VERIFIED | 97 lines, exports 3 functions, 30-entry noiseDirs map, 7-entry noiseFileExts map, 4 compound suffixes |
| `pkg/codegraph/scan_filter_test.go` | Filter unit tests, extended coverage, divergence detection | VERIFIED | 163 lines, 7 test functions covering all noise dirs, negative cases, compound extensions, bare env exclusion, count threshold |
| `pkg/codegraph/codegraph.go` | Scan function using shared filter | VERIFIED | `dirsToSkip` removed; line 508 calls `ShouldSkipDir(d.Name())` |
| `cmd/codex_colonize.go` | surveyWorkspace using shared filter, extractSourceAnchors, anchors.json output | VERIFIED | `shouldSkipSurveyDir` deleted; 3 call sites use `codegraph.ShouldSkipDir`; `SourceAnchors` field on `codexWorkspaceFacts`; `extractSourceAnchors` function; `anchors.json` in `writeSurveyCompatibilityJSON` |
| `cmd/codex_colonize_test.go` | Anchor extraction tests, venv noise exclusion, divergence test | VERIFIED | `TestExtractSourceAnchors_*` (7 tests), `TestVenvNoiseExclusion*` (2 tests), `TestSkipListDivergence`, `TestAnchorsWrittenToSurvey`, `TestLoadSurveyContext_*` (2 tests) |
| `cmd/codex_plan.go` | SourceAnchors on codexSurveyContext, anchors.json reading | VERIFIED | `SourceAnchors []string` field at line 56; initialized as `[]string{}` at line 1199; `readSummary("anchors.json")` at line 1243; dedup at line 1270 |
| `cmd/init_research.go` | Uses codegraph.ShouldSkipDir | VERIFIED | `extendedSkipDirs` deleted; line 1853 calls `codegraph.ShouldSkipDir(d.Name())` |
| `cmd/skills.go` | Uses codegraph.ShouldSkipDir | VERIFIED | `skillScanSkipDirs` deleted; lines 1268 and 1314 call `codegraph.ShouldSkipDir` |
| `cmd/discuss_analyze.go` | Uses codegraph.ShouldSkipDir | VERIFIED | Line 132 calls `codegraph.ShouldSkipDir(d.Name())`; compiles cleanly |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `pkg/codegraph/codegraph.go` | `pkg/codegraph/scan_filter.go` | `ShouldSkipDir` (same package) | WIRED | Line 508 calls `ShouldSkipDir(d.Name())` |
| `cmd/codex_colonize.go` | `pkg/codegraph/scan_filter.go` | `codegraph.ShouldSkipDir` | WIRED | Lines 454, 523, 914 |
| `cmd/codex_colonize.go` extractSourceAnchors | `pkg/codegraph/scan_filter.go` | `codegraph.ShouldSkipDir` and `codegraph.ShouldSkipFile` | WIRED | Lines 914, 921 |
| `cmd/init_research.go` | `pkg/codegraph/scan_filter.go` | `codegraph.ShouldSkipDir` | WIRED | Line 1853 |
| `cmd/skills.go` | `pkg/codegraph/scan_filter.go` | `codegraph.ShouldSkipDir` | WIRED | Lines 1268, 1314 |
| `cmd/discuss_analyze.go` | `pkg/codegraph/scan_filter.go` | `codegraph.ShouldSkipDir` | WIRED | Line 132 |
| `cmd/codex_colonize.go` writeSurveyCompatibilityJSON | `.aether/data/survey/anchors.json` | JSON file write with `source_anchors` field | WIRED | Lines 977-981, `os.WriteFile` at line 989 |
| `cmd/codex_plan.go` loadCodexSurveyContext | `.aether/data/survey/anchors.json` | `readSummary("anchors.json")` | WIRED | Lines 1243-1244 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| `extractSourceAnchors` | `candidates []string` | `filepath.WalkDir` on `root` | FLOWING | Walks actual filesystem, filters via ScanFilter, returns real relative paths from repo |
| `writeSurveyCompatibilityJSON` | `facts.SourceAnchors` | `extractSourceAnchors` return value | FLOWING | Written to `anchors.json` as `source_anchors` array |
| `loadCodexSurveyContext` | `ctx.SourceAnchors` | `readSummary("anchors.json")` | FLOWING | Reads `source_anchors` from anchors.json, deduplicates, stores in context |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Scan filter unit tests pass | `go test ./pkg/codegraph/ -run "TestShouldSkip\|TestNoiseDirCount\|TestScanSkips" -v -count=1` | All 7 test functions PASS | PASS |
| Venv noise exclusion and divergence tests pass | `go test ./cmd/ -run "TestVenvNoise\|TestSkipListDivergence" -v -count=1` | All 3 tests PASS | PASS |
| Source anchor extraction tests pass | `go test ./cmd/ -run "TestExtractSourceAnchors\|TestAnchorsWrittenToSurvey\|TestLoadSurveyContext" -v -count=1` | All 9 tests PASS | PASS |
| Full test suite on modified packages | `go test ./pkg/codegraph/ ./cmd/ -count=1` | `ok` both packages | PASS |
| Go vet on modified packages | `go vet ./pkg/codegraph/ ./cmd/` | No output (clean) | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| GROUND-01 | 141-01 | Unify 5 divergent skip lists into one canonical ScanFilter | SATISFIED | `pkg/codegraph/scan_filter.go` created with `ShouldSkipDir`, `ShouldSkipFile`, `NoiseDirCount`. All 5 call sites migrated. Zero local skip-list variables remain. |
| GROUND-02 | 141-01 | Extend filter to cover Python/Node/Rust/Go noise directories | SATISFIED | noiseDirs map has 30 entries: Python (`.venv`, `venv`, `__pycache__`, `.mypy_cache`, `.pytest_cache`, `.tox`, `.ruff_cache`, `.pytype`, `site-packages`), Node.js (`node_modules`, `.next`, `.nuxt`, `.svelte-kit`), Go (`vendor`), Rust (`target`, `.cargo`), plus caches (`.cache`, `.terraform`, `.gradle`). `node_modules/.cache` is implicitly covered because `node_modules` is skipped at the directory level. `.cargo/registry` is implicitly covered because `.cargo` is skipped. |
| GROUND-03 | 141-02 | Extract source anchors from cleaned survey output (top 50, depth-first) | SATISFIED | `extractSourceAnchors` at `codex_colonize.go:902` produces up to 50 paths sorted by depth then alphabetically. Uses shared ScanFilter for noise exclusion. |
| GROUND-04 | 141-02 | Write source anchors to survey output so planners can use them | SATISFIED | `anchors.json` written with `source_anchors`, `anchor_count`, `summary` fields. `codexSurveyContext.SourceAnchors` populated by `loadCodexSurveyContext`. |
| GROUND-05 | 141-01 | Regression test: `.venv` fixture produces clean survey | SATISFIED | `TestVenvNoiseExclusion` creates `.venv/lib/site-packages/requests/api.py` fixture and asserts zero `.venv` or `site-packages` paths in survey output. `TestSkipListDivergence` prevents re-introduction. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None | - | - | - | No debt markers, empty returns, stubs, or hardcoded empty data in phase-modified files |

### Human Verification Required

None. All truths are verifiable programmatically through tests, grep, and code inspection.

### Gaps Summary

No gaps found. All 9 observable truths verified against actual codebase evidence. All 5 requirements (GROUND-01 through GROUND-05) satisfied. All 16 key links wired. All 9 artifacts substantive and wired. Data flows confirmed for all dynamic artifacts. Full test suite passes on modified packages. Go vet clean.

---

_Verified: 2026-05-18T20:15:00Z_
_Verifier: Claude (gsd-verifier)_
