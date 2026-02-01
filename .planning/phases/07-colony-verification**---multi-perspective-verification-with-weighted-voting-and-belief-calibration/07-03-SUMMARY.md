---
phase: 07-colony-verification
plan: 03
subsystem: verification
tags: [watchers, specialized-prompts, performance, quality, test-coverage, multi-perspective, json-votes]

# Dependency graph
requires:
  - phase: 07-colony-verification
    plan: 01
    provides: vote-aggregator.sh, watcher_weights.json, structured vote format
provides:
  - Performance Watcher prompt with algorithmic complexity and I/O bottleneck detection
  - Quality Watcher prompt with maintainability and code convention validation
  - Test-Coverage Watcher prompt with test completeness and edge case verification
  - Complete set of 4 specialized Watchers ready for parallel spawning
affects: [07-04-parallel-spawning-integration, 07-05-verification-workflow]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Specialized Watcher prompts with domain-specific issue detection
    - Consistent JSON vote format across all Watchers
    - Weight-based voting with domain expertise matching
    - Severity-based issue categorization (Critical, High, Medium, Low)

key-files:
  created:
    - .aether/workers/performance-watcher.md
    - .aether/workers/quality-watcher.md
    - .aether/workers/test-coverage-watcher.md
  modified: []

key-decisions:
  - "Performance Watcher specializes in complexity, I/O, memory, blocking operations"
  - "Quality Watcher specializes in maintainability, readability, conventions, duplication"
  - "Test-Coverage Watcher specializes in completeness, coverage, assertions, edge_cases"
  - "All Watchers follow same JSON vote format as Security Watcher for aggregation compatibility"
  - "Severity thresholds: Critical > High > Medium > Low for consistent prioritization"
  - "Test coverage threshold: 70% branch coverage as minimum standard"

patterns-established:
  - "Pattern: Specialized Watcher prompts - domain-specific analysis, consistent output format"
  - "Pattern: Severity-based categorization - Critical/High/Medium/Low with clear thresholds"
  - "Pattern: JSON vote structure - watcher, decision, weight, issues array, timestamp"
  - "Pattern: Domain expertise matching - issue categories map to watcher specializations"

# Metrics
duration: 2min
completed: 2026-02-01
---

# Phase 7: Colony Verification - Specialized Watcher Prompts Summary

**Three specialized Watcher prompts (Performance, Quality, Test-Coverage) with domain-specific issue detection, consistent JSON vote format, and severity-based categorization for multi-perspective verification**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-01T19:57:15Z
- **Completed:** 2026-02-01T20:00:05Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments

- Created Performance Watcher prompt specializing in algorithmic complexity analysis, I/O bottlenecks, memory leaks, and blocking operations
- Created Quality Watcher prompt specializing in maintainability (complexity, length), readability (naming, magic numbers), conventions, and code duplication
- Created Test-Coverage Watcher prompt specializing in test completeness, coverage metrics, assertion quality, and edge case detection
- All three Watchers follow the same JSON vote format as Security Watcher for seamless integration with vote-aggregator.sh
- Each Watcher reads current weight from watcher_weights.json and outputs structured votes to .aether/verification/votes/

## Task Commits

Each task was committed atomically:

1. **Task 1: Create Performance Watcher prompt** - `82d0a9e` (feat)
2. **Task 2: Create Quality Watcher prompt** - `c551e98` (feat)
3. **Task 3: Create Test-Coverage Watcher prompt** - `c8cfc4c` (feat)

**Plan metadata:** (to be committed after SUMMARY.md and STATE.md)

## Files Created/Modified

- `.aether/workers/performance-watcher.md` - Performance-focused verification with complexity, I/O, memory, blocking categories
- `.aether/workers/quality-watcher.md` - Quality-focused verification with maintainability, readability, conventions, duplication categories
- `.aether/workers/test-coverage-watcher.md` - Test coverage verification with completeness, coverage, assertions, edge_cases categories

## Decisions Made

None - followed CONTEXT.md and PLAN.md specifications exactly.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all tasks completed without issues.

## Verification Results

All verification checks passed:

1. **File existence:**
   - performance-watcher.md created ✓
   - quality-watcher.md created ✓
   - test-coverage-watcher.md created ✓

2. **Required sections:**
   - All three have Purpose and Specialization sections ✓
   - All three have Current Weight reading from watcher_weights.json ✓
   - All three have Severity categories (Critical/High/Medium/Low) ✓
   - All three have domain-specific issue categories ✓
   - All three have JSON vote output format ✓
   - All three have Example Output sections ✓

3. **JSON format consistency:**
   - watcher field: "performance", "quality", "test_coverage" ✓
   - decision field: "APPROVE" or "REJECT" ✓
   - weight field: numeric value ✓
   - issues array with severity, category, description, location, recommendation ✓
   - timestamp field: ISO_8601 format ✓

4. **Domain specialization:**
   - Performance: complexity, io, memory, blocking categories ✓
   - Quality: maintainability, readability, conventions, duplication categories ✓
   - Test-Coverage: completeness, coverage, assertions, edge_cases categories ✓

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

**Ready for Wave 3 (07-04: Parallel Spawning Integration):**
- All 4 specialized Watcher prompts created (Security, Performance, Quality, Test-Coverage)
- All Watchers follow consistent JSON vote format matching vote-aggregator.sh expectations
- All Watchers read current weight from watcher_weights.json
- All Watchers output votes to .aether/verification/votes/ directory

**No blockers or concerns.**

---
*Phase: 07-colony-verification*
*Completed: 2026-02-01*
