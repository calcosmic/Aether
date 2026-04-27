---
phase: 60-oracle-loop-fix
plan: 02
subsystem: oracle-loop
tags: [oracle, brief, depth, tdd]
dependency_graph:
  requires: []
  provides: [60-03]
  affects: [60-04]
tech_stack:
  added: []
  patterns: [brief-formulation, depth-configuration, brief-informed-questions]
key_files:
  created: []
  modified:
    - cmd/oracle_loop.go
    - cmd/compatibility_cmds.go
    - cmd/compatibility_cmds_test.go
decisions:
  - "Brief-informed questions use section extraction from markdown-formatted brief text"
  - "Minimum question count padded to 5 when conditional questions are absent"
  - "Existing oracle tests updated to --depth exhaustive for backward compatibility"
  - "Learnings returned in reverse order (most recent first) for brief relevance"
metrics:
  duration_seconds: 918
  completed_at: "2026-04-27T12:02:05Z"
  tasks_completed: 2
  tasks_total: 2
  files_modified: 3
  lines_added: 634
  lines_removed: 14
---

# Phase 60 Plan 02: Oracle Brief and Depth Selection Summary

Research brief formulation and depth selection for the Oracle research loop, enabling context-rich questions and user-controlled research depth.

## Commits

| Hash | Type | Message |
|------|------|---------|
| 583048ca | test | add failing tests for oracle brief formulation and brief-informed questions |
| 599ec1c9 | feat | add research brief formulation and brief-informed question generation |
| 3d577ea6 | test | add failing tests for oracle depth selection |
| 9fd4b8de | feat | add depth selection with CLI flag and state configuration |

## What Changed

### cmd/oracle_loop.go

**New types and configuration:**
- `oracleDepthConfig` struct with MaxIterations, TargetConfidence, Label, Description fields
- `oracleDepthLevels` map with 4 predefined depth levels: quick (2/60), balanced (4/85), deep (6/95), exhaustive (10/99)
- `Depth string` field added to `oracleStateFile` for persistence

**New functions:**
- `resolveOracleDepth(depth)` -- maps string to depth config, defaults to balanced for invalid/empty input
- `formulateOracleBrief(root, topic, detectedType, languages, frameworks)` -- gathers full colony context into a structured research brief with sections for Topic, Project Profile, Colony Goal, Codebase Structure, Active Signals, and Recent Learnings. Writes brief.md to the oracle workspace.
- `buildBriefInformedQuestions(topic, brief, detectedType)` -- generates 5-8 questions that reference actual brief content (file paths, framework names, signal content) instead of generic keyword-matched templates
- `currentOracleRedirectAreas()` -- parallel to `currentOracleFocusAreas()` but filters for REDIRECT signals
- `loadColonyLearnings(root)` -- reads last 5 instincts from COLONY_STATE.json in reverse order
- `loadColonyGoal(root)` -- reads colony goal from COLONY_STATE.json
- `scanCodebaseStructure(root)` -- scans top-level and one-level-deep directories for the brief
- `extractBriefSection(brief, sectionName)` -- extracts content between markdown section headers
- `extractBriefField(brief, fieldName)` -- extracts field values from brief sections

**Modified functions:**
- `startOracleCompatibility(root, topic, depth)` -- now accepts depth parameter, calls `formulateOracleBrief()`, uses `buildBriefInformedQuestions()` instead of `buildOracleQuestions()`, applies depth config to state
- `runOracleCompatibility(root, args, depth)` -- passes depth string through to `startOracleCompatibility()`
- `buildOracleQuestions()` kept as fallback but no longer called

### cmd/compatibility_cmds.go

- Added `--depth` flag to `oracleCmd` with help text listing all four levels
- `oracleCmd.RunE` reads the `--depth` flag and passes it to `runOracleCompatibility()`

### cmd/compatibility_cmds_test.go

**New tests (8 test functions, 15 subtests):**
- `TestFormulateOracleBriefContainsRequiredSections` -- verifies brief contains topic, profile, signals, structure, writes brief.md
- `TestBuildBriefInformedQuestionsReferencesBriefContent` -- verifies 5-8 questions reference actual brief content
- `TestBuildBriefInformedQuestionsWorksWithMinimalBrief` -- verifies minimal brief still produces valid questions
- `TestCurrentOracleRedirectAreasReturnsRedirectSignals` -- verifies REDIRECT signal filtering
- `TestLoadColonyLearningsReturnsRecentInstincts` -- verifies last 5 instincts loaded in reverse order
- `TestLoadColonyLearningsReturnsEmptyWhenMissing` -- graceful handling of missing colony state
- `TestResolveOracleDepth` (8 subtests) -- all depth levels, empty, invalid, case insensitive, whitespace
- `TestOracleDepthFlagSetsMaxIterations` -- integration test verifying --depth deep sets MaxIterations=6

**Updated existing tests:** 4 tests updated to use `--depth exhaustive` to preserve their original iteration budget.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Brief.md file content mismatch with test assertion**
- **Found during:** Task 1 GREEN phase
- **Issue:** Test compared file content (with trailing newline) with returned brief (without trailing newline)
- **Fix:** Updated test to use `strings.TrimSpace()` on file content before comparison
- **Files modified:** cmd/compatibility_cmds_test.go

**2. [Rule 1 - Bug] Existing tests broke due to reduced default iteration cap**
- **Found during:** Task 2 GREEN phase
- **Issue:** Changing default from 8 to 4 iterations caused 3 existing tests to hit `max_iterations_reached` instead of `complete`
- **Fix:** Updated 4 existing tests to explicitly pass `--depth exhaustive` to maintain their original behavior
- **Files modified:** cmd/compatibility_cmds_test.go

**3. [Rule 1 - Bug] runOracleCompatibility signature change broke test callers**
- **Found during:** Task 2 GREEN phase
- **Issue:** Adding `depth` parameter to `runOracleCompatibility()` broke 4 test callers
- **Fix:** Updated all test callers to pass empty string `""` for depth
- **Files modified:** cmd/compatibility_cmds_test.go

## TDD Gate Compliance

- RED commit `583048ca`: test(60-02): add failing tests for oracle brief formulation and brief-informed questions
- GREEN commit `599ec1c9`: feat(60-02): add research brief formulation and brief-informed question generation
- RED commit `3d577ea6`: test(60-02): add failing tests for oracle depth selection
- GREEN commit `9fd4b8de`: feat(60-02): add depth selection with CLI flag and state configuration
- All TDD gates satisfied.

## Known Stubs

None.

## Self-Check: PASSED

All functions exist, all tests pass, all commits verified, no stubs found.
