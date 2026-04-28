---
phase: 73-rich-init-research
plan: 01
subsystem: cli
tags: [go, dependency-parsing, init-research, gjson, toml, xml, regex]

# Dependency graph
requires: []
provides:
  - depEntry and techStackDetail struct types for dependency data
  - 9 dependency file parsers covering package.json, go.mod, Cargo.toml, pyproject.toml, composer.json, requirements.txt, Gemfile, pom.xml, mix.exs
  - tech_stack_detail output field in init-research command
  - 11 new tests covering all parser formats, integration, and backward compatibility
affects: [73-02, 73-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Tolerant parsing: return nil/empty on any error, never panic"
    - "1MB file size cap for all dependency file reads (DoS mitigation)"

key-files:
  created: []
  modified:
    - cmd/init_research.go
    - cmd/init_research_test.go

key-decisions:
  - "Used gjson IsObject() not IsMap() (correct gjson v1.18 API)"
  - "Kept encoding/json out of imports (gjson handles all JSON parsing)"
  - "Added encoding/xml and regexp imports only in Task 2 when those parsers were implemented"

patterns-established:
  - "Tolerant parsing pattern: every parser returns nil/empty on any error"
  - "1MB max file read cap for untrusted dependency files"
  - "gjson ForEach for JSON map iteration (package.json, composer.json)"
  - "toml.Decode into map[string]interface{} for TOML parsing (Cargo.toml, pyproject.toml)"
  - "Minimal XML structs for pom.xml (mavenProject, mavenDeps, mavenDep)"

requirements-completed: [INIT-03]

# Metrics
duration: 5min
completed: 2026-04-28
---

# Phase 73 Plan 01: Dependency Parsing Summary

**9 dependency file parsers with tolerant error handling, wired into init-research output as tech_stack_detail**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-28T22:49:46Z
- **Completed:** 2026-04-28T22:54:49Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Implemented 9 dependency file parsers covering all 8 supported formats (package.json, go.mod, Cargo.toml, pyproject.toml, composer.json, requirements.txt, Gemfile, pom.xml, mix.exs)
- Added `tech_stack_detail` output field to init-research with language, source_file, dependencies, dev_dependencies, and indirect arrays
- All 11 new tests pass alongside all 12 pre-existing tests (zero regressions)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add dependency types and JSON/Go/TOML-based parsers** - `a3fbfd73` (feat)
2. **Task 2: Add regexp/XML/line-based parsers and finalize orchestration** - `2df491db` (feat)

## Files Created/Modified
- `cmd/init_research.go` - Added depEntry, techStackDetail types; 9 parser functions; parseDependencyFiles orchestrator; wired into outputOK
- `cmd/init_research_test.go` - Added 11 tests: 5 format-specific (Task 1), 4 format-specific + integration + backward compat (Task 2)

## Decisions Made
- Used gjson `IsObject()` not `IsMap()` (correct gjson v1.18 API -- IsMap does not exist)
- Did not add `encoding/json` to imports since gjson handles all JSON parsing
- Added `encoding/xml` and `regexp` imports only when their parsers were implemented in Task 2
- Pom.xml dependency names include groupId prefix (e.g., "org.springframework:spring-core") for uniqueness

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed gjson API usage**
- **Found during:** Task 1 (implementing parsePackageJsonDeps)
- **Issue:** Plan specified `IsMap()` which does not exist in gjson v1.18.0
- **Fix:** Changed to `IsObject()` which is the correct gjson API for checking if a result is a JSON object
- **Files modified:** cmd/init_research.go
- **Verification:** `go vet ./cmd/...` passes, all tests pass
- **Committed in:** a3fbfd73 (Task 1 commit)

**2. [Rule 2 - Missing Critical] Added 1MB file size cap**
- **Found during:** Task 1 (implementing parsers)
- **Issue:** Threat model T-73-01 requires capping file reads to prevent DoS on giant files
- **Fix:** Added `maxDepFileSize` constant (1MB) and size checks in all 9 parsers
- **Files modified:** cmd/init_research.go
- **Verification:** All parsers check `len(data) > maxDepFileSize` before parsing
- **Committed in:** a3fbfd73 (Task 1 commit)

**3. [Rule 3 - Blocking] Fixed embedded_assets build failure**
- **Found during:** Task 1 (running tests)
- **Issue:** Worktree missing `.aether/rules/` directory caused `go test` to fail on embedded_assets.go pattern
- **Fix:** Created `.aether/rules/` directory in worktree
- **Files modified:** .aether/rules/.gitkeep (not committed -- worktree-local fix)
- **Verification:** `go test ./cmd/...` passes
- **Committed in:** N/A (worktree-local, not tracked)

---

**Total deviations:** 3 auto-fixed (1 bug, 1 missing critical, 1 blocking)
**Impact on plan:** All auto-fixes essential for correctness and security. No scope creep.

## Issues Encountered
- The Edit tool had trouble matching tab-indented Go code strings. Used Python script and heredoc approaches as workarounds for reliable string replacement.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 9 dependency parsers are production-ready with tests
- The `tech_stack_detail` output field is available for downstream consumers (Phase 74 suggest-analyze)
- No blockers for Phase 73 Plan 02 (directory classification) or Plan 03 (governance deep parsing)

---
*Phase: 73-rich-init-research*
*Completed: 2026-04-28*
