---
phase: 64-lifecycle-ceremony-discuss-chaos-oracle-patrol
plan: 01
subsystem: cli-commands
tags: [cobra, go, discuss, council, inventory-scan, codebase-analysis]

# Dependency graph
requires: []
provides:
  - "discuss-analyze subcommand for inventory-level codebase scanning"
  - "structured suggested questions in 5 categories complementary to existing discuss"
  - "discuss and council wrappers wired to call discuss-analyze before their existing behavior"
affects: [discuss, council, intent-capture]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "inventory-only scan pattern (no source file reading) for fast codebase analysis"
    - "analyze: source prefix to distinguish from discuss: source prefix"

key-files:
  created:
    - cmd/discuss_analyze.go
    - cmd/discuss_analyze_test.go
  modified:
    - .claude/commands/ant/discuss.md
    - .opencode/commands/ant/discuss.md
    - .claude/commands/ant/council.md
    - .opencode/commands/ant/council.md
    - .aether/commands/discuss.yaml
    - .aether/commands/council.yaml

key-decisions:
  - "Reused projectDetectors and detectGovernance from init-research.go for consistency"
  - "Used analyze: prefix for Source field to distinguish from discuss: prefix"
  - "5 question categories (architecture, dependencies, testing_infrastructure, deployment, performance) complement existing discuss categories without overlap"
  - "Zero new dependencies -- all existing Go stdlib + cobra + storage packages"

patterns-established:
  - "Inventory-only scan pattern: read directory entries, detect markers, walk for counts -- no source file reading"
  - "Source prefix namespacing: analyze: vs discuss: for question provenance"

requirements-completed: [CERE-09]

# Metrics
duration: 6min
completed: 2026-04-27
---

# Phase 64 Plan 01: discuss-analyze Subcommand Summary

**Inventory-level codebase scan subcommand with 5 complementary question categories, wired into discuss and council wrappers via YAML follow_up entries**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-27T18:45:15Z
- **Completed:** 2026-04-27T18:51:47Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- Created `discuss-analyze` cobra subcommand that performs inventory-only codebase scanning (languages, frameworks, governance, architecture patterns) without reading source files
- Generated 5 suggested question categories (architecture, dependencies, testing_infrastructure, deployment, performance) that complement existing discuss categories (surface, integration, scope, verification) without overlap
- Wired discuss-analyze into both Claude and OpenCode wrappers for discuss and council commands
- Added follow_up.analyze and follow_up.analyze_purpose entries to both YAML source files

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED):** `e7b8e5c9` (test) - Add 4 failing tests for discuss-analyze
2. **Task 1 (GREEN):** `3f12b310` (feat) - Implement discuss-analyze subcommand
3. **Task 2:** `56a711bf` (feat) - Wire discuss-analyze into discuss and council wrappers

## Files Created/Modified
- `cmd/discuss_analyze.go` - New cobra subcommand with inventory scan and question generation
- `cmd/discuss_analyze_test.go` - 4 tests: basic scan, distinct categories, empty dir, goal flag
- `.claude/commands/ant/discuss.md` - Added discuss-analyze pre-scan instruction
- `.opencode/commands/ant/discuss.md` - Added discuss-analyze pre-scan instruction
- `.claude/commands/ant/council.md` - Added discuss-analyze context instruction
- `.opencode/commands/ant/council.md` - Added discuss-analyze context instruction
- `.aether/commands/discuss.yaml` - Added follow_up.analyze and analyze_purpose
- `.aether/commands/council.yaml` - Added follow_up section with analyze entries

## Decisions Made
- Reused `projectDetectors` and `detectGovernance` from init-research.go for consistent language/framework detection across commands
- Used `analyze:` source prefix to distinguish analyze-generated questions from `discuss:` questions in provenance tracking
- Kept question generation simple and deterministic -- no LLM calls, just rule-based matching against scan data
- All question options are context-aware (e.g., Docker detected -> container-focused options)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- discuss-analyze subcommand is registered and callable via `aether discuss-analyze --target .`
- Both wrappers reference the new subcommand and will call it before their existing behavior
- CERE-09 requirement satisfied: discuss/council now analyze the codebase before asking comprehensive questions

---
*Phase: 64-lifecycle-ceremony-discuss-chaos-oracle-patrol*
*Completed: 2026-04-27*
