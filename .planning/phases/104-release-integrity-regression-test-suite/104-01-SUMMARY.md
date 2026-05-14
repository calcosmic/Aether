---
phase: 104-release-integrity-regression-test-suite
plan: 01
subsystem: release-pipeline
tags: [e2e, release, publish, update, install, temp-dir]

# Dependency graph
requires:
  - phase: 100-command-inventory-lifecycle-contracts
    provides: Mock source checkout patterns and Cobra test isolation
provides:
  - End-to-end release pipeline verification test
  - Publish→hub→install→update cycle validation
  - Stale file cleanup verification
affects: [105-remediation]

# Tech tracking
tech-stack:
  added: []
  patterns: [mock-filesystem, internal-function-call, temp-dir-isolation]

key-files:
  created:
    - cmd/release_pipeline_test.go
    - cmd/testdata/release_pipeline_snapshot.json
  modified: []

key-decisions:
  - "Internal functions called directly (not rootCmd.Execute) for speed and mock injection"
  - "All filesystem operations use t.TempDir() — no real ~/.aether/ touched"
  - "OpenCode agent frontmatter requires description (20+ chars), mode, color, tools map"

requirements-completed: [REL-01, REL-02]

# Metrics
duration: 20min
completed: 2026-05-08
---

# Phase 104 Plan 01: Release Pipeline E2E Verification Summary

**3 tests simulating the full publish→hub→install→update cycle using temp directories only.**

## Performance

- **Duration:** ~20 min (agent got stuck in debug loop, manual rescue)
- **Started:** 2026-05-08
- **Completed:** 2026-05-08
- **Tasks:** 1
- **Files created:** 2

## Accomplishments

- `TestReleasePipelineE2E` passes — full cycle: mock source → publish sync → version check → update --force (stale removal) → install to fresh home
- `TestPublishHubSync` passes — verifies version agreement, sync pair counts (7 + 7)
- `TestUpdateStaleFileCleanup` passes — stale files in managed dirs removed by cleanup
- Golden snapshot captures sync pair counts and version

## Task Commits

1. **Task 1: Write release pipeline E2E verification test** - `7c66ee3e` (test)

## Files Created

- `cmd/release_pipeline_test.go` — 3 test functions with mock source checkout helper
- `cmd/testdata/release_pipeline_snapshot.json` — Golden snapshot (7 sync pairs, 7 home pairs, version 1.0.34)

## Issues Encountered

**Agent stuck in debug loop:** The executor agent repeatedly ran `go test ./cmd/ -run TestDebugHubStructure` in a loop. Killed after 15+ iterations. Manual rescue: wrote correct test with proper mock source checkout (companion files in all required dirs), fixed OpenCode agent frontmatter validation (needs description ≥20 chars, mode, color, tools map), and verified all 3 tests pass.

## Self-Check: PASSED

- cmd/release_pipeline_test.go: FOUND
- cmd/testdata/release_pipeline_snapshot.json: FOUND
- Commit 7c66ee3e: FOUND
- All 3 tests pass

---
*Phase: 104-release-integrity-regression-test-suite*
*Completed: 2026-05-08*
