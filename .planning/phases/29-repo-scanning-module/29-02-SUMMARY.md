---
phase: 29-repo-scanning-module
plan: 02
subsystem: infra
tags: [bash, jq, scan, repo-introspection, smart-init, git]

# Dependency graph
requires:
  - phase: 29-01
    provides: "scan.sh module skeleton with stub functions and dispatch wiring"
provides:
  - "6 fully functional scan functions producing real repo introspection data"
  - "Tech stack detection: languages (7), frameworks (8), package managers (8)"
  - "Directory structure measurement: file count, max depth, top-level dirs"
  - "Git history summary: repo check, commit count, recent 10 commits"
  - "Survey status: completeness check, staleness detection (7-day window), suggestions"
  - "Prior colony detection: active colony state, archived chambers"
  - "Complexity classification: small/medium/large with threshold matrix"
affects: [29-03-PLAN, 30-charter-functions, 31-init-rewrite, 32-intelligence]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "File-presence detection pattern for language/framework/package-manager identification"
    - "Platform-aware stat command (macOS stat -f %m vs Linux stat -c %Y)"
    - "Threshold matrix classification (file count + depth + deps -> size)"
    - "Sub-scan functions return raw JSON, entry point wraps in json_ok"

key-files:
  modified:
    - .aether/utils/scan.sh

key-decisions:
  - "Sub-scan functions return raw JSON via stdout (not json_ok); entry point _scan_init_research wraps final result in json_ok"
  - "Entry point passes target_dir argument to all sub-scan functions (was not in original skeleton)"
  - "Directory listing filters to directories only (not files) and excludes excluded dirs"
  - "Survey staleness uses 7-day window with platform-aware stat fallback"
  - "Complexity thresholds: large (500+ files OR 8+ depth OR 50+ deps), medium (100+ OR 5+ OR 15+), small otherwise"

patterns-established:
  - "jq incremental array building pattern: echo '$arr' | jq '. + [\"item\"]' for building arrays in bash"
  - "Graceful degradation: every scan function handles missing files/dirs/non-git repos without crashing"

requirements-completed: [SCAN-02, SCAN-03]

# Metrics
duration: 4min
completed: 2026-03-27
---

# Phase 29 Plan 2: Scan Implementations Summary

**Six repo introspection scan functions with real detection logic: tech stack (7 languages, 8 frameworks, 8 package managers), directory structure measurement, git history summary, survey status with staleness checks, prior colony detection, and complexity classification -- all completing in under 1.1 seconds**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-27T15:47:22Z
- **Completed:** 2026-03-27T15:51:17Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Implemented all 6 scan functions replacing stubs from Plan 29-01 with real repo introspection logic
- Tech stack detection covers 7 languages (TS, JS, Python, Go, Rust, Ruby, Java), 8 frameworks (Next.js, Angular, Vue, React, Express, Fastify, Svelte, NestJS), and 8 package managers (npm, yarn, pnpm, go-modules, cargo, bundler, pip, poetry)
- Git history handles non-git repos gracefully (returns is_git_repo:false instead of crashing)
- Survey status detects missing/incomplete/stale surveys with actionable suggestions
- Complexity classification uses threshold matrix: large (500+ files OR 8+ depth OR 50+ deps), medium (100+ OR 5+ OR 15+), small otherwise
- Performance: 1.037 seconds on the Aether repo (5188+ files), well under the 2-second target
- All 616 existing tests continue to pass

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement tech stack, directory structure, and git history scan functions** - `74a3d7e` (feat)
2. **Task 2: Implement survey status, prior colonies, and complexity scan functions** - `2c7fb01` (feat)

## Files Created/Modified
- `.aether/utils/scan.sh` - All 6 stub functions replaced with real implementations

## Decisions Made
- Sub-scan functions return raw JSON (not json_ok), entry point wraps final assembly -- cleaner separation of concerns
- Entry point updated to pass target_dir to all sub-scan functions (original skeleton called them without arguments)
- Directory listing changed from `ls -1` (includes files) to `ls -1d */` (directories only) with explicit exclusion filtering
- Platform-aware stat: uses `stat -f %m` on macOS, `stat -c %Y` on Linux for survey staleness
- Survey staleness window set to 7 days with COLONY_STATE.json territory_surveyed as primary source, file mtime as fallback

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed top_level_dirs including files instead of only directories**
- **Found during:** Task 1 (Implement directory structure scan)
- **Issue:** `ls -1 "$root" | grep -v '^\.'` listed both files and directories, and the grep -v for excluded dirs used malformed IFS-based pattern that didn't work
- **Fix:** Changed to `ls -1d "$root"/*/` to list only directories, replaced broken grep with explicit loop checking against _SCAN_EXCLUDE_DIRS array
- **Files modified:** .aether/utils/scan.sh
- **Verification:** `init-research | jq '.result.directory_structure.top_level_dirs'` now returns only directory names without files or excluded dirs
- **Committed in:** `74a3d7e` (Task 1 commit)

**2. [Rule 3 - Blocking] Updated entry point to pass target_dir and capture raw JSON from sub-scans**
- **Found during:** Task 1 (Implement tech stack scan)
- **Issue:** Plan specified sub-scans return raw JSON (not json_ok), but the entry point from Plan 29-01 called `jq -r '.result'` on each sub-scan output (expecting json_ok format) and did not pass target_dir argument
- **Fix:** Changed entry point to pass "$target_dir" to each sub-scan call and capture raw JSON output directly (no `.result` extraction)
- **Files modified:** .aether/utils/scan.sh
- **Verification:** All sub-scan functions produce valid JSON that assembles correctly in the final output
- **Committed in:** `74a3d7e` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both auto-fixes necessary for correctness. The calling convention change (raw JSON) was explicitly specified in the plan but not implemented in the skeleton.

## Issues Encountered
None beyond the auto-fixed deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 6 scan functions produce real, validated data -- Phase 31 (init.md rewrite) can consume the structured JSON output
- Performance target met (1.037s on large repo) -- no optimization needed
- Schema contract stable with schema_version:1 field for future evolution
- Plan 29-03 (if applicable) can build on this foundation

## Self-Check: PASSED

- FOUND: .aether/utils/scan.sh
- FOUND: .planning/phases/29-repo-scanning-module/29-02-SUMMARY.md
- FOUND: 74a3d7e (Task 1 commit)
- FOUND: 2c7fb01 (Task 2 commit)

---
*Phase: 29-repo-scanning-module*
*Completed: 2026-03-27*
