---
phase: 73-rich-init-research
plan: 02
subsystem: cli
tags: [go, directory-classification, governance-parsing, init-research, yaml, gjson]

# Dependency graph
requires:
  - phase: 73-01
    provides: "depEntry, techStackDetail types, 9 dependency parsers, hasFile helper"
provides:
  - dirClassification type and classifyDirectory function with 5 classification types
  - governanceDetail type with tool, file, category, rules, extends, config fields
  - 14 deep governance parsers across 5 categories (linter, formatter, test, ci, build)
  - deepParseGovernance orchestrator function
  - dir_classification and governance_details output fields in init-research
  - 14 new tests covering classification and governance deep parsing
affects: [73-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Multiline regex for line-by-line parsing in Go (Makefile, justfile)"
    - "GitHub Actions workflow glob capped at 20 files (DoS mitigation)"
    - "Tolerant parsing: all governance parsers return nil on any error"

key-files:
  created: []
  modified:
    - cmd/init_research.go
    - cmd/init_research_test.go

key-decisions:
  - "Used gjson for JSON config parsing (ESLint, Prettier, Biome) and yaml.v3 for YAML configs"
  - "Jest/Vitest use regex extraction since config files are JS/TS (not parseable as JSON/YAML)"
  - "GitHub Actions returns one governanceDetail per workflow file (not merged)"
  - "GitLab CI excludes structural keys (stages, variables, default, include) from job count"

patterns-established:
  - "Classification function returns type + signals explaining WHY the classification was chosen"
  - "All governance parsers use tolerant error handling (nil on error, cap at 1MB)"
  - "deepParseGovernance orchestrator calls all parsers and aggregates results"
  - "New output fields are additive -- existing fields remain unchanged"

requirements-completed: [INIT-04, INIT-05]

# Metrics
duration: 9min
completed: 2026-04-29
---

# Phase 73 Plan 02: Directory Classification and Governance Deep Parsing Summary

**Directory structure classification (5 types with detection signals) and deep governance config parsing (14 parsers across 5 categories) wired into init-research output**

## Performance

- **Duration:** 9 min
- **Started:** 2026-04-28T22:59:27Z
- **Completed:** 2026-04-28T23:07:56Z
- **Tasks:** 3
- **Files modified:** 2

## Accomplishments
- Implemented directory classification with 5 types (monorepo, microservices, standard_app, library, unknown) and detection signals explaining why
- Implemented 14 deep governance parsers across 5 categories (linter, formatter, test, ci, build)
- Both new output fields (`dir_classification`, `governance_details`) are additive -- existing output unchanged
- All 14 new tests pass alongside all 23 pre-existing tests (zero regressions)

## Task Commits

Each task was committed atomically:

1. **Task 1: Directory classification with detection signals** - `f13e46aa` (feat)
2. **Task 2a: Deep governance parsing for linters and formatters** - `df84164e` (feat)
3. **Task 2b: Deep governance parsing for test/CI/build + orchestration** - `ab1eb4bc` (feat)

## Files Created/Modified
- `cmd/init_research.go` - Added dirClassification type, classifyDirectory function, governanceDetail type, 14 deep governance parsers, deepParseGovernance orchestrator, hasDir helper, wired dir_classification and governance_details into outputOK
- `cmd/init_research_test.go` - Added 14 tests: 5 classification tests, 4 linter/formatter tests, 4 test/CI/build tests, 1 backward compatibility test

## Decisions Made
- Used gjson for JSON config parsing (ESLint rules/extends, Prettier options, Biome sections)
- Used yaml.v3 for YAML config parsing (golangci-lint, pytest.ini, GitLab CI, Taskfile)
- Jest/Vitest use regex extraction since config files are JS/TS (not parseable as structured data)
- GitHub Actions returns one governanceDetail per workflow file (more useful than merged)
- GitLab CI excludes structural keys from job count to avoid counting non-job entries

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed Go raw string containing backtick character**
- **Found during:** Task 2b (implementing parseJestDeep)
- **Issue:** Go raw string literal (backtick-delimited) cannot contain backtick characters, but the regex for matching JS template literals needed a backtick
- **Fix:** Removed backtick from the regex character class, matching only single and double quotes
- **Files modified:** cmd/init_research.go
- **Verification:** `go build ./cmd/...` passes, all tests pass
- **Committed in:** ab1eb4bc (Task 2b commit)

**2. [Rule 1 - Bug] Fixed multiline regex for Makefile/justfile target matching**
- **Found during:** Task 2b (running TestDeepParseMakefile)
- **Issue:** Go's regexp `^` anchor matches start of text by default, not start of line. Makefile target regex `^([a-zA-Z0-9]...):` only matched the first target
- **Fix:** Added `(?m)` multiline flag to both Makefile and justfile regex patterns
- **Files modified:** cmd/init_research.go
- **Verification:** TestDeepParseMakefile now passes with all 3 targets found
- **Committed in:** ab1eb4bc (Task 2b commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both auto-fixes essential for correctness. No scope creep.

## Issues Encountered
- The Edit tool had trouble matching tab-indented Go code strings with special characters (backticks, regex). Used Python string replacement as a workaround for reliable editing.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Directory classification is production-ready with tests for all 5 types
- Deep governance parsing covers all 5 categories at equal depth (14 parsers)
- Both new output fields are additive -- wrappers and ceremony don't need changes
- No blockers for Phase 73 Plan 03 (pheromone expansion) or Phase 74 (suggest-analyze)

---
*Phase: 73-rich-init-research*
*Completed: 2026-04-29*
