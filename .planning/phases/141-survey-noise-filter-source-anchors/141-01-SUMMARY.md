---
phase: 141-survey-noise-filter-source-anchors
plan: 01
subsystem: codebase-scanning
tags: [scan-filter, noise-exclusion, skip-list, codegraph, go]

# Dependency graph
requires: []
provides:
  - pkg/codegraph/scan_filter.go with ShouldSkipDir, ShouldSkipFile, NoiseDirCount
  - Unified 36-entry noiseDirs map covering Python, Node.js, Go, Rust, build artifacts, caches, Aether-managed, IDE/editor, temp
  - All five call sites migrated to shared filter
  - Divergence regression test preventing re-introduction of local skip lists
affects: [141-02-source-anchors, codex_plan, survey]

# Tech tracking
tech-stack:
  added: []
  patterns: [shared-filter-module, canonical-noise-directory-map]

key-files:
  created:
    - pkg/codegraph/scan_filter.go
    - pkg/codegraph/scan_filter_test.go
  modified:
    - pkg/codegraph/codegraph.go
    - pkg/codegraph/codegraph_test.go
    - cmd/codex_colonize.go
    - cmd/codex_colonize_test.go
    - cmd/init_research.go
    - cmd/skills.go
    - cmd/discuss_analyze.go

key-decisions:
  - "Bare 'env' excluded from noiseDirs (Pitfall 8) -- repos commonly have env/ source directories"
  - "site-packages included as top-level entry despite usually being inside .venv -- safety net for rare standalone cases"
  - "Used runtime.Caller in TestSkipListDivergence for reliable path resolution in test environments"

patterns-established:
  - "Canonical noise map: all directory exclusion goes through pkg/codegraph/scan_filter.go ShouldSkipDir()"
  - "Divergence test: TestSkipListDivergence prevents re-introduction of local skip-list maps"

requirements-completed: [GROUND-01, GROUND-02, GROUND-05]

# Metrics
duration: 5min
completed: 2026-05-18
---

# Phase 141 Plan 1: Survey Noise Filter Summary

**Canonical ScanFilter unifying 5 divergent skip lists into single ShouldSkipDir with 36-entry noise map and divergence regression test**

## Performance

- **Duration:** 5 min
- **Started:** 2026-05-18T19:17:45Z
- **Completed:** 2026-05-18T19:22:00Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Single canonical ScanFilter in pkg/codegraph/scan_filter.go replaces 5 divergent skip lists
- 36-entry noiseDirs map covers version control, Python (11 entries incl .venv, __pycache__, site-packages, .pytest_cache, .tox, .mypy_cache), Node.js (4), Go (vendor), Rust (target, .cargo), build artifacts (5), caches (3), Aether-managed (4), IDE (2), temp (2)
- ShouldSkipFile with compound extension checking (.min.js before .js)
- Divergence regression test prevents re-introduction of local skip lists in cmd/

## Task Commits

Each task was committed atomically:

1. **Task 1: Create canonical ScanFilter and tests, migrate codegraph.go** - `75717a8c` (feat)
2. **Task 2: Migrate four remaining call sites and add regression tests** - `bc1e0114` (fix)

## Files Created/Modified
- `pkg/codegraph/scan_filter.go` - Canonical noiseDirs map, ShouldSkipDir, ShouldSkipFile, NoiseDirCount
- `pkg/codegraph/scan_filter_test.go` - 7 test functions: TestShouldSkipDir, TestShouldSkipDir_Negative, TestShouldSkipDir_ExtendedCoverage, TestShouldSkipFile, TestShouldSkipFile_CompoundExtensions, TestShouldSkipDir_NoBareEnv, TestNoiseDirCount
- `pkg/codegraph/codegraph.go` - Removed dirsToSkip variable, Scan() now calls ShouldSkipDir()
- `pkg/codegraph/codegraph_test.go` - Renamed TestDirsToSkip to TestScanSkipsNoiseDirs
- `cmd/codex_colonize.go` - Deleted shouldSkipSurveyDir, uses codegraph.ShouldSkipDir at 2 call sites
- `cmd/codex_colonize_test.go` - Added TestVenvNoiseExclusion, TestVenvNoiseExclusion_SourceFilesPreserved, TestSkipListDivergence
- `cmd/init_research.go` - Deleted extendedSkipDirs variable, uses codegraph.ShouldSkipDir
- `cmd/skills.go` - Deleted skillScanSkipDirs variable, uses codegraph.ShouldSkipDir at 2 call sites
- `cmd/discuss_analyze.go` - Replaced extendedSkipDirs reference with codegraph.ShouldSkipDir

## Decisions Made
- Bare "env" excluded from noiseDirs per Pitfall 8 -- many repos have env/ source directories
- site-packages included as top-level entry for safety (rare standalone cases)
- TestVenvNoiseExclusion uses existing struct fields (TopLevelDirs, TestFiles) rather than SourceFiles (which is plan 141-02 scope)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed TestVenvNoiseExclusion referencing non-existent SourceFiles field**
- **Found during:** Task 2 verification
- **Issue:** Test referenced facts.SourceFiles which does not exist on codexWorkspaceFacts struct (SourceFiles is plan 141-02 scope)
- **Fix:** Replaced SourceFiles reference with TopLevelDirs checking; rewrote TestVenvNoiseExclusion_SourceFilesPreserved to verify src/ and cmd/ directories appear in TopLevelDirs
- **Files modified:** cmd/codex_colonize_test.go
- **Verification:** go test ./cmd/ -run TestVenvNoise -v passes
- **Committed in:** bc1e0114 (Task 2 commit)

**2. [Rule 1 - Bug] Fixed TestSkipListDivergence relative path failure in test environments**
- **Found during:** Task 2 verification
- **Issue:** Test used os.ReadDir("cmd") with relative path, failing when test working directory differs from repo root
- **Fix:** Used runtime.Caller(0) to resolve the cmd/ directory path relative to the test file's location
- **Files modified:** cmd/codex_colonize_test.go
- **Verification:** go test ./cmd/ -run TestSkipListDivergence -v passes
- **Committed in:** bc1e0114 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes necessary for test correctness. No scope creep.

## Issues Encountered
None -- all planned work completed with two test fixes for pre-existing compilation errors.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- ScanFilter is ready for plan 141-02 (source anchors) to import and use
- All five call sites verified using shared filter with zero local skip-list variables remaining
- go vet and full test suite pass on modified packages

## Self-Check: PASSED

All 10 files verified present. All 3 commits verified in git log. Zero uncommitted changes.

---
*Phase: 141-survey-noise-filter-source-anchors*
*Completed: 2026-05-18*
